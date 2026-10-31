// Package api لایهٔ HTTP ماژول احکام حقوق و گواهی اشتغال — فقط ترجمهٔ درخواست/پاسخ.
// هیچ قاعدهٔ کسب‌وکاری اینجا نیست: دامنه اعتبارسنجی می‌کند، app گردش‌کار را می‌برد و
// این لایه صرفاً فیلتر نقش، کد وضعیت و شکل پاسخ را می‌سازد.
package api

import (
	"errors"
	"net/http"
	"strconv"

	"hrms/internal/contracts"
	"hrms/internal/modules/salary_admin/app"
	"hrms/internal/modules/salary_admin/domain"
	"hrms/internal/modules/salary_admin/infra"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/rules"
)

// Handler هندلرهای ماژول.
type Handler struct {
	svc  *app.Service
	deps modulekit.Deps
}

// New هندلر می‌سازد.
func New(svc *app.Service, deps modulekit.Deps) *Handler {
	return &Handler{svc: svc, deps: deps}
}

func (h *Handler) scopeInfo(r *http.Request, moduleID string) (scope, actorID, actorDepartment string) {
	scope = "all"
	if user, ok := auth.UserFrom(r.Context()); ok {
		scope = h.deps.Policy.Scope(user.Role, moduleID)
		actorID = user.EmployeeID
		actorDepartment = user.DepartmentID
	}
	return
}

func (h *Handler) decreeFilter(r *http.Request) contracts.SalaryDecreeFilter {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))
	scope, actorID, actorDept := h.scopeInfo(r, "salary_admin")
	return contracts.SalaryDecreeFilter{
		Query:           q.Get("query"),
		EmployeeID:      q.Get("employeeId"),
		Status:          q.Get("status"),
		JobGroup:        q.Get("jobGroup"),
		EmploymentType:  q.Get("employmentType"),
		Sort:            q.Get("sort"),
		Page:            page,
		Size:            size,
		Scope:           scope,
		ActorEmployeeID: actorID,
		ActorDepartment: actorDept,
	}
}

func (h *Handler) certificateFilter(r *http.Request) contracts.CertificateFilter {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))
	scope, actorID, actorDept := h.scopeInfo(r, "salary_admin")
	return contracts.CertificateFilter{
		Query:           q.Get("query"),
		EmployeeID:      q.Get("employeeId"),
		Kind:            q.Get("kind"),
		Status:          q.Get("status"),
		Page:            page,
		Size:            size,
		Scope:           scope,
		ActorEmployeeID: actorID,
		ActorDepartment: actorDept,
	}
}

/* ------------------------------- حکم حقوق ------------------------------- */

// ListDecrees فهرست احکام حقوق.
func (h *Handler) ListDecrees(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.ListDecrees(r.Context(), h.decreeFilter(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// GetDecree یک حکم.
func (h *Handler) GetDecree(w http.ResponseWriter, r *http.Request) {
	d, ok, err := h.svc.GetDecree(r.Context(), h.decreeFilter(r), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

// CreateDecree صدور پیش‌نویس حکم.
func (h *Handler) CreateDecree(w http.ResponseWriter, r *http.Request) {
	var in contracts.SalaryDecreeInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.CreateDecree(r.Context(), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// UpdateDecree ویرایش پیش‌نویس حکم.
func (h *Handler) UpdateDecree(w http.ResponseWriter, r *http.Request) {
	var in contracts.SalaryDecreeInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.UpdateDecree(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// IssueDecree صدور حکم.
func (h *Handler) IssueDecree(w http.ResponseWriter, r *http.Request) {
	updated, err := h.svc.IssueDecree(r.Context(), r.PathValue("id"), h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// NotifyDecree ابلاغ حکم.
func (h *Handler) NotifyDecree(w http.ResponseWriter, r *http.Request) {
	updated, err := h.svc.NotifyDecree(r.Context(), r.PathValue("id"), h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// ReviseDecree اصلاح حکم (نسخهٔ تازه).
func (h *Handler) ReviseDecree(w http.ResponseWriter, r *http.Request) {
	var in contracts.SalaryDecreeDecision
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.ReviseDecree(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// IssueGroupDecree صدور گروهی حکم کارگزینی بر پایهٔ نوع استخدام.
func (h *Handler) IssueGroupDecree(w http.ResponseWriter, r *http.Request) {
	var in contracts.SalaryDecreeGroupInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	result, err := h.svc.IssueGroupDecree(r.Context(), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, result)
}

// DeleteDecree حذف پیش‌نویس حکم.
func (h *Handler) DeleteDecree(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteDecree(r.Context(), r.PathValue("id"), h.actor(r)); err != nil {
		h.fail(w, err)
		return
	}
	httpx.NoContent(w)
}

/* ------------------------------ گواهی اشتغال ------------------------------ */

// ListCertificates فهرست گواهی‌ها.
func (h *Handler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.ListCertificates(r.Context(), h.certificateFilter(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// GetCertificate یک گواهی.
func (h *Handler) GetCertificate(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.svc.GetCertificate(r.Context(), h.certificateFilter(r), r.PathValue("id"))
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

// CreateCertificate ثبت درخواست گواهی.
func (h *Handler) CreateCertificate(w http.ResponseWriter, r *http.Request) {
	var in contracts.CertificateInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.CreateCertificate(r.Context(), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// UpdateCertificate ویرایش درخواست گواهی.
func (h *Handler) UpdateCertificate(w http.ResponseWriter, r *http.Request) {
	var in contracts.CertificateInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.UpdateCertificate(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// DecideCertificate تأیید یا رد درخواست گواهی.
func (h *Handler) DecideCertificate(w http.ResponseWriter, r *http.Request) {
	var in contracts.CertificateDecision
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.DecideCertificate(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// IssueCertificate صدور گواهی.
func (h *Handler) IssueCertificate(w http.ResponseWriter, r *http.Request) {
	updated, err := h.svc.IssueCertificate(r.Context(), r.PathValue("id"), h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// DeliverCertificate تحویل گواهی.
func (h *Handler) DeliverCertificate(w http.ResponseWriter, r *http.Request) {
	updated, err := h.svc.DeliverCertificate(r.Context(), r.PathValue("id"), h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// DeleteCertificate حذف درخواست گواهی.
func (h *Handler) DeleteCertificate(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteCertificate(r.Context(), r.PathValue("id"), h.actor(r)); err != nil {
		h.fail(w, err)
		return
	}
	httpx.NoContent(w)
}

// Options فهرست‌های کمکی فرم‌ها (انواع گواهی، وضعیت‌ها).
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	kinds := []map[string]string{}
	for _, k := range domain.CertKinds {
		kinds = append(kinds, map[string]string{"value": k, "label": domain.CertKindFa[k]})
	}
	decreeStatuses := []map[string]string{}
	for _, s := range domain.DecreeStatuses {
		decreeStatuses = append(decreeStatuses, map[string]string{"value": s, "label": domain.DecreeStatusFa[s]})
	}
	certStatuses := []map[string]string{}
	for _, s := range domain.CertStatuses {
		certStatuses = append(certStatuses, map[string]string{"value": s, "label": domain.CertStatusFa[s]})
	}
	// مناطق اقلیمی از دامنهٔ قواعد می‌آید تا فرم حکم، فهرست ثابت خودش را نسازد (هیچ قاعدهٔ
	// کسب‌وکاری در رابط نیست) و ضرایب پیشنهادی هر منطقه هم همان‌جا دیده شود.
	regions := []map[string]any{}
	for _, region := range rules.ClimateRegionsTable {
		regions = append(regions, map[string]any{
			"value": region.Key, "label": region.TitleFa,
			"badClimate": region.BadClimate, "deprivation": region.Deprivation,
			"regionHardship": region.RegionHardship, "extraTime": region.ExtraTime,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"certificateKinds": kinds,
		"decreeStatuses":   decreeStatuses,
		"certStatuses":     certStatuses,
		"employmentTypes":  contracts.EmploymentTypes(),
		"employmentGroups": contracts.EmploymentGroups(),
		"climateRegions":   regions,
	})
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
	case errors.Is(err, infra.ErrDecreeNotFound), errors.Is(err, infra.ErrCertificateNotFound):
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
	default:
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
	}
}
