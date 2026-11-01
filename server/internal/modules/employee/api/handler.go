// Package api لایهٔ HTTP ماژول کارکنان.
//
// این لایه فقط ترجمهٔ درخواست/پاسخ است: هیچ قاعدهٔ کسب‌وکاری اینجا نوشته نمی‌شود.
// محدودهٔ دسترسی (سازمانی/واحد/خود) از سیاست نقش گرفته و به فیلتر پورت داده می‌شود؛
// یعنی محدودیت در سطح کوئری اعمال می‌شود، نه با پنهان‌کردن سطر در UI.
package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/employee/app"
	"hrms/internal/modules/employee/infra"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// Handler هندلرهای ماژول کارکنان.
type Handler struct {
	svc  *app.Service
	deps modulekit.Deps
}

// New هندلر می‌سازد.
func New(svc *app.Service, deps modulekit.Deps) *Handler {
	return &Handler{svc: svc, deps: deps}
}

// filterFrom درخواست را به فیلتر پورت تبدیل می‌کند و محدودهٔ نقش را اعمال می‌کند.
func (h *Handler) filterFrom(r *http.Request) contracts.EmployeeFilter {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))

	// فیلترهای چندانتخابی هم پذیرفته می‌شوند: رابط می‌تواند چند مقدار را با کاما
	// بفرستد (مثلاً employmentType=official_worker,fixed_1662) و سرور آن را
	// «هر یک از این‌ها» می‌فهمد. یک مقدار تنها همان رفتار پیشین را دارد.
	statusValues := contracts.SplitValues(q.Get("status"))
	employmentValues := contracts.SplitValues(q.Get("employmentType"))

	f := contracts.EmployeeFilter{
		Query:           q.Get("query"),
		DepartmentID:    q.Get("departmentId"),
		Status:          singleOf(statusValues),
		Statuses:        statusValues,
		ContractType:    q.Get("contractType"),
		EmploymentType:  singleOf(employmentValues),
		EmploymentTypes: employmentValues,
		Sort:            q.Get("sort"),
		Page:            page,
		Size:            size,
		Scope:           "all",
	}

	if user, ok := auth.UserFrom(r.Context()); ok {
		f.Scope = h.deps.Policy.Scope(user.Role, "employee")
		f.ActorID = user.EmployeeID
		f.ActorDepartment = user.DepartmentID
	}
	return f
}

// List فهرست کارکنان با جستجو، فیلتر، مرتب‌سازی و صفحه‌بندی.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.List(r.Context(), h.filterFrom(r))
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// Get پروندهٔ کارمند.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	employee, ok, err := h.svc.Get(r.Context(), h.filterFrom(r), id)
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, employee)
}

// Create ایجاد کارمند.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in contracts.EmployeeInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.Create(r.Context(), in, h.actor(r))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// Update ویرایش کارمند.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var in contracts.EmployeeInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.Update(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// Delete حذف کارمند.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id"), h.actor(r)); err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// Departments واحدهای سازمانی برای فیلتر و نمودار درختی.
func (h *Handler) Departments(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Departments(r.Context())
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// departmentList واحدهای سازمانی برای فرم‌ها؛ در خطا فهرست خالی برمی‌گردد تا فرم از کار نیفتد.
func (h *Handler) departmentList(r *http.Request) []contracts.Department {
	list, err := h.svc.Departments(r.Context())
	if err != nil {
		return []contracts.Department{}
	}
	return list
}

// Options فهرست‌های کمکی برای فرم (وضعیت‌ها و انواع قرارداد) تا کلاینت این فهرست‌ها
// را از سرور بگیرد و دو نسخه نشود.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"statuses":         contracts.EmployeeStatusKeys(),
		"statusOptions":    contracts.EmployeeStatuses(),
		"contractTypes":    []string{"تمام‌وقت", "پیمانکاری", "پاره‌وقت"},
		"employmentTypes":  contracts.EmploymentTypes(),
		"employmentGroups": contracts.EmploymentGroups(),
		"departments":      h.departmentList(r),
	})
}

// singleOf مقدار فیلتر تک‌مقداری را از فهرست تیک‌خورده می‌سازد: تنها وقتی که کاربر
// دقیقاً یک گزینه را انتخاب کرده باشد؛ در غیر این صورت خالی می‌ماند و "فهرست
// چندانتخابی" جایش را می‌گیرد (تا فیلتر تکراری روی یک مقدار اعمال نشود).
func singleOf(values []string) string {
	if len(values) == 1 {
		return values[0]
	}
	return ""
}

func (h *Handler) actor(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok {
		return user.Username
	}
	return "ناشناس"
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var validation *app.ValidationError
	switch {
	case errors.As(err, &validation):
		httpx.WriteError(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, validation.Details)
	case errors.Is(err, infra.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
	default:
		msg := err.Error()
		if strings.Contains(msg, "نامعتبر") {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, msg, nil)
			return
		}
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, msg, nil)
	}
}
