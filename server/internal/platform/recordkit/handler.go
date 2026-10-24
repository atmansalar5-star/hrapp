// پایانه‌های رکوردکیت (schema/list/get/create/update/delete/upload) — پرکاربردترین سطح حملهٔ داده؛ بررسی IDOR.

package recordkit

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// RepositoryFor انتخاب انبار بر پایهٔ اتصال پایگاه‌داده: Postgres در استقرار واقعی و
// حافظه در اجرای توسعه و آزمون. هیچ ماژولی خودش این انتخاب را تکرار نمی‌کند.
//
// ترتیب اولویت: انبار تزریق‌شده (اجرای مستقل روی یک رایانه، مثل انبار فایل JSON) مقدم
// است؛ سپس Postgres؛ و در نهایت حافظه.
func RepositoryFor(deps modulekit.Deps, clock func() time.Time) Repository {
	if custom, ok := deps.Records.(Repository); ok && custom != nil {
		return custom
	}
	if deps.DB != nil {
		return NewPostgresStore(deps.DB, clock)
	}
	return NewMemoryStore()
}

// Handler هندلرهای مشترک رکوردهای گردش‌کاری.
type Handler struct {
	svc      *Service
	deps     modulekit.Deps
	moduleID string
}

// NewHandler هندلر می‌سازد.
func NewHandler(svc *Service, deps modulekit.Deps) *Handler {
	return &Handler{svc: svc, deps: deps, moduleID: svc.Module()}
}

// Register همهٔ مسیرهای استاندارد رکوردها را روی mux سوار می‌کند.
//
// یک الگوی یکسان برای همهٔ ماژول‌های اداری: شناسهٔ ماژول (schema)، فهرست، ایجاد،
// خواندن، ویرایش، حذف و اقدام گردش‌کار. هیچ ماژولی مسیر دست‌ساز جداگانه ندارد.
func Register(mux *http.ServeMux, deps modulekit.Deps, svc *Service, prefix string) {
	h := NewHandler(svc, deps)
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc {
		return deps.Require(svc.Module(), action, fn)
	}
	mux.HandleFunc("GET "+prefix+"/schema", r("read", h.Schema))
	mux.HandleFunc("GET "+prefix+"/records", r("read", h.List))
	mux.HandleFunc("POST "+prefix+"/records", r("write", h.Create))
	mux.HandleFunc("GET "+prefix+"/records/{id}", r("read", h.Get))
	mux.HandleFunc("PATCH "+prefix+"/records/{id}", r("write", h.Update))
	mux.HandleFunc("DELETE "+prefix+"/records/{id}", r("delete", h.Delete))
	mux.HandleFunc("POST "+prefix+"/records/{id}/action", r("approve", h.Act))
	deps.Logger(svc.Module()).Info("ماژول رکوردهای گردش‌کاری ثبت شد", "prefix", prefix, "routes", 7)
}

func (h *Handler) filter(r *http.Request) contracts.RecordFilter {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))
	f := contracts.RecordFilter{
		Module:     h.moduleID,
		Kind:       q.Get("kind"),
		Status:     q.Get("status"),
		Query:      q.Get("query"),
		EmployeeID: q.Get("employeeId"),
		Page:       page,
		Size:       size,
		Scope:      "all",
	}
	if user, ok := auth.UserFrom(r.Context()); ok {
		f.Scope = h.deps.Policy.Scope(user.Role, h.moduleID)
		f.ActorEmployeeID = user.EmployeeID
		f.ActorDepartment = user.DepartmentID
	}
	return f
}

// Schema شناسهٔ ماژول (انواع رکورد، فیلدها، وضعیت‌ها، اقدام‌ها) برای ساخت رابط.
func (h *Handler) Schema(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.svc.Schema())
}

// List فهرست رکوردها.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.List(r.Context(), h.filter(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// Get خواندن یک رکورد.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	rec, ok, err := h.svc.Get(r.Context(), h.filter(r), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rec)
}

// Create ایجاد رکورد تازه.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in contracts.RecordInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.Create(r.Context(), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// Update ویرایش رکورد.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var in contracts.RecordInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.Update(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// Act انجام اقدام گردش‌کار.
func (h *Handler) Act(w http.ResponseWriter, r *http.Request) {
	var in contracts.RecordAction
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.Act(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// Delete حذف رکورد.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id"), h.actor(r)); err != nil {
		h.fail(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) actor(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok {
		return user.Username
	}
	return "ناشناس"
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	var validation *ValidationError
	var rule *RuleError
	switch {
	case errors.As(err, &validation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, validation.Details)
	case errors.As(err, &rule):
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, rule.Message, nil)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
	default:
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
	}
}
