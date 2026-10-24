// Package api لایهٔ HTTP ماژول تکفل و درخواست پرواز — فقط ترجمهٔ درخواست/پاسخ.
package api

import (
	"errors"
	"net/http"
	"strconv"

	"hrms/internal/contracts"
	"hrms/internal/modules/staff_services/app"
	"hrms/internal/modules/staff_services/domain"
	"hrms/internal/modules/staff_services/infra"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
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

func (h *Handler) scope(r *http.Request) (scope, actorID, actorDepartment string) {
	scope = "all"
	if user, ok := auth.UserFrom(r.Context()); ok {
		scope = h.deps.Policy.Scope(user.Role, "staff_services")
		actorID = user.EmployeeID
		actorDepartment = user.DepartmentID
	}
	return scope, actorID, actorDepartment
}

func (h *Handler) dependentFilter(r *http.Request) contracts.DependentFilter {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))
	scope, actorID, actorDepartment := h.scope(r)
	return contracts.DependentFilter{
		Query:           q.Get("query"),
		EmployeeID:      q.Get("employeeId"),
		Relation:        q.Get("relation"),
		Status:          q.Get("status"),
		Page:            page,
		Size:            size,
		Scope:           scope,
		ActorEmployeeID: actorID,
		ActorDepartment: actorDepartment,
	}
}

func (h *Handler) flightFilter(r *http.Request) contracts.FlightRequestFilter {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))
	scope, actorID, actorDepartment := h.scope(r)
	return contracts.FlightRequestFilter{
		Query:           q.Get("query"),
		EmployeeID:      q.Get("employeeId"),
		Status:          q.Get("status"),
		Purpose:         q.Get("purpose"),
		Page:            page,
		Size:            size,
		Scope:           scope,
		ActorEmployeeID: actorID,
		ActorDepartment: actorDepartment,
	}
}

// ---- افراد تحت تکفل ----

// ListDependents فهرست تکفل‌شدگان.
func (h *Handler) ListDependents(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.ListDependents(r.Context(), h.dependentFilter(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// GetDependent یک تکفل.
func (h *Handler) GetDependent(w http.ResponseWriter, r *http.Request) {
	d, ok, err := h.svc.GetDependent(r.Context(), h.dependentFilter(r), r.PathValue("id"))
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

// CreateDependent ثبت تکفل تازه.
func (h *Handler) CreateDependent(w http.ResponseWriter, r *http.Request) {
	var in contracts.DependentInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.CreateDependent(r.Context(), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// UpdateDependent ویرایش تکفل.
func (h *Handler) UpdateDependent(w http.ResponseWriter, r *http.Request) {
	var in contracts.DependentInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.UpdateDependent(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// DecideDependent تأیید یا رد تکفل.
func (h *Handler) DecideDependent(w http.ResponseWriter, r *http.Request) {
	var in contracts.ContractDecision
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.DecideDependent(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// RemoveDependent خروج از تکفل.
func (h *Handler) RemoveDependent(w http.ResponseWriter, r *http.Request) {
	var in contracts.ContractDecision
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.RemoveDependent(r.Context(), r.PathValue("id"), in.NoteFa, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// DeleteDependent حذف ثبت ردشده.
func (h *Handler) DeleteDependent(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteDependent(r.Context(), r.PathValue("id"), h.actor(r)); err != nil {
		h.fail(w, err)
		return
	}
	httpx.NoContent(w)
}

// ---- درخواست پرواز ----

// ListFlights فهرست درخواست‌های پرواز.
func (h *Handler) ListFlights(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.ListFlights(r.Context(), h.flightFilter(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

// GetFlight یک درخواست پرواز.
func (h *Handler) GetFlight(w http.ResponseWriter, r *http.Request) {
	f, ok, err := h.svc.GetFlight(r.Context(), h.flightFilter(r), r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	if !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, f)
}

// CreateFlight ثبت فرم پرواز.
func (h *Handler) CreateFlight(w http.ResponseWriter, r *http.Request) {
	var in contracts.FlightRequestInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	created, err := h.svc.CreateFlight(r.Context(), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

// UpdateFlight ویرایش فرم پرواز.
func (h *Handler) UpdateFlight(w http.ResponseWriter, r *http.Request) {
	var in contracts.FlightRequestInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.UpdateFlight(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// DecideFlight اقدام گردش‌کار روی فرم پرواز.
func (h *Handler) DecideFlight(w http.ResponseWriter, r *http.Request) {
	var in contracts.FlightDecision
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	updated, err := h.svc.DecideFlight(r.Context(), r.PathValue("id"), in, h.actor(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// Options فهرست‌های کمکی فرم (نسبت، هدف سفر، وضعیت‌ها).
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	relations := []map[string]string{}
	for _, rel := range domain.Relations {
		relations = append(relations, map[string]string{"value": rel, "label": domain.RelationFa[rel]})
	}
	purposes := []map[string]string{}
	for _, p := range domain.Purposes {
		purposes = append(purposes, map[string]string{"value": p, "label": domain.PurposeFa[p]})
	}
	statuses := []map[string]string{}
	for _, s := range []string{domain.DependentPending, domain.DependentApproved, domain.DependentRejected, domain.DependentRemoved} {
		statuses = append(statuses, map[string]string{"value": s, "label": domain.DependentStatusFa[s]})
	}
	flightStatuses := []map[string]string{}
	for _, s := range []string{domain.FlightPending, domain.FlightApproved, domain.FlightIssued, domain.FlightSettled, domain.FlightRejected, domain.FlightCancelled} {
		flightStatuses = append(flightStatuses, map[string]string{"value": s, "label": domain.FlightStatusFa[s]})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"relations":      relations,
		"purposes":       purposes,
		"statuses":       statuses,
		"flightStatuses": flightStatuses,
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
	case errors.Is(err, infra.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
	default:
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
	}
}
