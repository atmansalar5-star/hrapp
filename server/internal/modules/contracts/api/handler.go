// Package api لایهٔ HTTP ماژول قراردادها — فقط ترجمهٔ درخواست/پاسخ.
//
// هیچ قاعدهٔ کسب‌وکاری اینجا نیست: دامنه اعتبارسنجی می‌کند، app گردش‌کار را می‌برد و
// این لایه صرفاً فیلتر نقش، کد وضعیت و شکل پاسخ را می‌سازد.
package api

import (
	"errors"
	"net/http"
	"strconv"

	"hrms/internal/contracts"
	"hrms/internal/modules/contracts/app"
	"hrms/internal/modules/contracts/domain"
	"hrms/internal/modules/contracts/infra"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// Handler هندلرهای ماژول قراردادها.
type Handler struct {
	svc  *app.Service
	deps modulekit.Deps
}

// New هندلر می‌سازد.
func New(svc *app.Service, deps modulekit.Deps) *Handler {
	return &Handler{svc: svc, deps: deps}
}

// filterFrom درخواست را به فیلتر تبدیل و محدودهٔ دسترسی نقش را اعمال می‌کند.
func (h *Handler) filterFrom(r *http.Request) contracts.ContractFilter {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))
	expiring, _ := strconv.Atoi(q.Get("expiringDays"))

	f := contracts.ContractFilter{
		Query:          q.Get("query"),
		EmployeeID:     q.Get("employeeId"),
		Kind:           q.Get("kind"),
		Status:         q.Get("status"),
		EmploymentType: q.Get("employmentType"),
		Sort:           q.Get("sort"),
		Page:           page,
		Size:           size,
		ExpiringDays:   expiring,
		Today:          q.Get("today"),
		Scope:          "all",
	}
	if user, ok := auth.UserFrom(r.Context()); ok {
		f.Scope = h.deps.Policy.Scope(user.Role, "contracts")
		f.ActorEmployeeID = user.EmployeeID
		f.ActorDepartment = user.DepartmentID
	}
	return f
}

// List فهرست قراردادها.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.List(r.Context(), h.filterFrom(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// Renewals قراردادهای نزدیک به سرآمد (صفحهٔ تمدید و پایان قرارداد).
func (h *Handler) Renewals(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	items, err := h.svc.Expiring(r.Context(), days)
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

// Get یک قرارداد.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.svc.Get(r.Context(), h.filterFrom(r), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

// Create انعقاد قرارداد تازه.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in contracts.ContractInput
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

// Update ویرایش پیش‌نویس.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var in contracts.ContractInput
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

// Submit ارسال برای تأیید.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	updated, err := h.svc.Submit(r.Context(), r.PathValue("id"), h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// Decision تأیید یا رد قرارداد.
func (h *Handler) Decision(w http.ResponseWriter, r *http.Request) {
	var in contracts.ContractDecision
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.Decide(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// Terminate خاتمهٔ قرارداد جاری.
func (h *Handler) Terminate(w http.ResponseWriter, r *http.Request) {
	var in contracts.ContractDecision
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.Terminate(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// Renew تمدید قرارداد مدت موقت.
func (h *Handler) Renew(w http.ResponseWriter, r *http.Request) {
	var in contracts.ContractRenewInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.Renew(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// Convert تبدیل مدت موقت به مدت دائم.
func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	var in contracts.ContractRenewInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.ConvertToPermanent(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// Delete حذف پیش‌نویس.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id"), h.actor(r)); err != nil {
		h.fail(w, err)
		return
	}
	httpx.NoContent(w)
}

// Options فهرست‌های کمکی فرم قرارداد، تا کلاینت این فهرست‌ها را دوباره نسازد.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	kinds := []map[string]string{}
	for _, k := range domain.Kinds {
		kinds = append(kinds, map[string]string{"value": k, "label": domain.KindFa[k]})
	}
	statuses := []map[string]string{}
	for _, s := range domain.Statuses {
		statuses = append(statuses, map[string]string{"value": s, "label": domain.StatusFa[s]})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"kinds":            kinds,
		"statuses":         statuses,
		"employmentTypes":  contracts.EmploymentTypes(),
		"employmentGroups": contracts.EmploymentGroups(),
	})
}

// Templates قالب‌های متن قرارداد را از پورت قالب‌های اسناد برمی‌گرداند.
//
// متن قرارداد قابل طراحی است: قالب فعال از ماژول تنظیمات می‌آید و این ماژول تنها
// قالب‌های نوع «قرارداد» را فیلتر می‌کند. نبود قالب یعنی متن پیش‌فرض سامانه.
func (h *Handler) Templates(w http.ResponseWriter, r *http.Request) {
	templates := []contracts.DocumentTemplate{}
	if provider, ok := h.deps.Ports.DocumentTemplates(); ok {
		if list, err := provider.Templates(r.Context()); err != nil {
			h.fail(w, err)
			return
		} else {
			for _, t := range list {
				if t.DocType == "contract" {
					templates = append(templates, t)
				}
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"templates": templates, "total": len(templates)})
}

func (h *Handler) actor(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok {
		return user.Username
	}
	return "ناشناس"
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	var validation *app.ValidationError
	var rule *app.RuleError
	switch {
	case errors.As(err, &validation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, validation.Details)
	case errors.As(err, &rule):
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, rule.Message, nil)
	case errors.Is(err, infra.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
	default:
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
	}
}
