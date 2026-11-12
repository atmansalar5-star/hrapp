// Ù…Ø³ÛŒØ±Ù‡Ø§ÛŒ Â«Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ú©Ø§Ø±Ú©Ù†Ø§Ù†Â» â€” Ú©Ù†ØªØ±Ù„â€ŒÙ¾Ù†Ù„ÛŒ Ú©Ù‡ Ø¨Ø±Ø§ÛŒ Ù‡Ø± Ú©Ø§Ø±Ù…Ù†Ø¯ ØªØ¹ÛŒÛŒÙ† Ù…ÛŒâ€ŒÚ©Ù†Ø¯ Ù¾Ø±Ø¯Ø§Ø®ØªØ´ Ø¨Ø§
// Ú©Ø¯Ø§Ù… Ù…Ø¯Ù„ Ù…Ø­Ø§Ø³Ø¨Ù‡ Ø´ÙˆØ¯.
//
// Ù…Ø±Ø² Ù…Ø¯Ù„â€ŒÙ‡Ø§: Ø§ÛŒÙ† Ù…Ø³ÛŒØ± ÙÙ‚Ø· Ø§Ù†ØªØ®Ø§Ø¨ Ø±Ø§ Ø«Ø¨Øª Ùˆ Ù†Ù…Ø§ÛŒØ´ Ù…ÛŒâ€ŒØ¯Ù‡Ø¯Ø› Ù‡ÛŒÚ† Ù…Ø­Ø§Ø³Ø¨Ù‡â€ŒØ§ÛŒ Ø±Ø§ Ø¹ÙˆØ¶ Ù†Ù…ÛŒâ€ŒÚ©Ù†Ø¯.
// Ù…Ø¯Ù„ Ø­Ù‚ÙˆÙ‚ Ùˆ Ø¯Ø³ØªÙ…Ø²Ø¯ Ø¬Ø§Ø±ÛŒ Ùˆ Ù…Ø¯Ù„ ØµÙ†Ø¯ÙˆÙ‚ Ù†ÙØª Ù…Ø³ØªÙ‚Ù„ Ùˆ Ø¬Ø§Ø±ÛŒ Ù…ÛŒâ€ŒÙ…Ø§Ù†Ù†Ø¯ Ùˆ Ù…Ø¯Ù„ Û´Ù¾ÛŒØŒ Ù…Ø¯Ù„ ØªÙˆØ³Ø¹Ù‡â€ŒØ§ÛŒ
// Ùˆ Ø§Ø®ØªÛŒØ§Ø±ÛŒ Ø§Ø³Øª (Ø¨Ù‡ Ù‡Ù…ÛŒÙ† Ø¯Ù„ÛŒÙ„ Ø§Ù†ØªØ®Ø§Ø¨ Ø¢Ù† Ù¾ÛŒØ´ Ø§Ø² ÙØ¹Ø§Ù„â€ŒØ¨ÙˆØ¯Ù† Ù…Ø§Ú˜ÙˆÙ„ Ùˆ Ù¾Ø§Ø±Ø§Ù…ØªØ± Â«Ø§Ø¹Ù…Ø§Ù„ Ø¯Ø±
// Ù¾Ø±Ø¯Ø§Ø®ØªÂ» Ø±Ø¯ Ù…ÛŒâ€ŒØ´ÙˆØ¯ Ùˆ Ù¾Ø³ Ø§Ø² Ø¢Ù† Ù‡Ù… ÙÙ‚Ø· ÛŒØ§Ø¯Ø¢ÙˆØ±ÛŒ Ù…ÛŒâ€ŒÚ¯ÛŒØ±Ø¯).
package payroll

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/payroll/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// compensationModuleID Ø´Ù†Ø§Ø³Ù‡Ù” Ù…Ø§Ú˜ÙˆÙ„ Ù…Ø¯Ù„ ØªÙˆØ³Ø¹Ù‡â€ŒØ§ÛŒ Û´Ù¾ÛŒ (ÙÙ‚Ø· Ø¨Ø±Ø§ÛŒ Ø¨Ø±Ø±Ø³ÛŒ ÙØ¹Ø§Ù„â€ŒØ¨ÙˆØ¯Ù†).
const compensationModuleID = "compensation"

// payMethodRequest ÙˆØ±ÙˆØ¯ÛŒ Ø«Ø¨Øª Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª ÛŒÚ© Ú©Ø§Ø±Ù…Ù†Ø¯.
type payMethodRequest struct {
	EmployeeID    string `json:"employeeId"`
	PayModel      string `json:"payModel"`
	EffectiveDate string `json:"effectiveDate"`
	ReasonFa      string `json:"reasonFa"`
}

// payMethodRow ÛŒÚ© Ø³Ø·Ø± Ú©Ù†ØªØ±Ù„â€ŒÙ¾Ù†Ù„: Ú©Ø§Ø±Ù…Ù†Ø¯ Ùˆ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø¬Ø§Ø±ÛŒ Ø§Ùˆ.
type payMethodRow struct {
	EmployeeID     string `json:"employeeId"`
	FullName       string `json:"fullName"`
	DepartmentName string `json:"departmentName"`
	PayModel       string `json:"payModel"`
	PayModelFa     string `json:"payModelFa"`
	RecordID       string `json:"recordId,omitempty"`
	EffectiveDate  string `json:"effectiveDate,omitempty"`
	Status         string `json:"status,omitempty"`
	StatusFa       string `json:"statusFa,omitempty"`
	// SourceFa Ù…Ù†Ø¨Ø¹ Ø§Ù†ØªØ®Ø§Ø¨: Â«Ø«Ø¨Øªâ€ŒØ´Ø¯Ù‡Â» ÛŒØ§ Â«Ù¾ÛŒØ´â€ŒÙØ±Ø¶ Ø³Ø§Ø²Ù…Ø§Ù†Â».
	SourceFa string `json:"sourceFa"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// registerPayMethodEndpoints Ù…Ø³ÛŒØ±Ù‡Ø§ÛŒ Ú©Ù†ØªØ±Ù„â€ŒÙ¾Ù†Ù„ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø±Ø§ Ø«Ø¨Øª Ù…ÛŒâ€ŒÚ©Ù†Ø¯.
func registerPayMethodEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "write", fn) }

	// ÙÙ‡Ø±Ø³Øª Ø±ÙˆØ´â€ŒÙ‡Ø§ÛŒ Ù…ØµÙˆØ¨ + ÙˆØ¶Ø¹ÛŒØª Ù…Ø¯Ù„ Û´Ù¾ÛŒ Ø¯Ø± Ø§ÛŒÙ† Ø§Ø³ØªÙ‚Ø±Ø§Ø±.
	mux.HandleFunc("GET /api/v1/payroll/pay-method-options", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, payMethodOptions(deps))
	}))

	// ÙÙ‡Ø±Ø³Øª Ú©Ø§Ø±Ú©Ù†Ø§Ù† Ø¨Ø§ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø¬Ø§Ø±ÛŒâ€ŒØ´Ø§Ù†.
	mux.HandleFunc("GET /api/v1/payroll/pay-methods", read(func(w http.ResponseWriter, r *http.Request) {
		rows, err := payMethodRows(r.Context(), deps, svc, strings.TrimSpace(r.URL.Query().Get("query")))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		out := payMethodOptions(deps)
		out["items"] = rows
		out["count"] = len(rows)
		httpx.WriteJSON(w, http.StatusOK, out)
	}))

	// Ø«Ø¨Øª ÛŒØ§ ØªØºÛŒÛŒØ± Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª ÛŒÚ© Ú©Ø§Ø±Ù…Ù†Ø¯Ø› Ø±Ú©ÙˆØ±Ø¯ Ø¬Ø§Ø±ÛŒ Ù¾ÛŒØ´ÛŒÙ† Ù…Ù†Ø³ÙˆØ® Ù…ÛŒâ€ŒØ´ÙˆØ¯.
	mux.HandleFunc("POST /api/v1/payroll/pay-methods", write(func(w http.ResponseWriter, r *http.Request) {
		var in payMethodRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ÙˆØ±ÙˆØ¯ÛŒ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø®ÙˆØ§Ù†Ø¯Ù‡ Ù†Ø´Ø¯.", nil)
			return
		}
		employeeID := strings.TrimSpace(in.EmployeeID)
		if employeeID == "" {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, "Ú©Ø§Ø±Ù…Ù†Ø¯ Ø±Ø§ Ø§Ù†ØªØ®Ø§Ø¨ Ú©Ù†ÛŒØ¯.", nil)
			return
		}
		payModel := strings.TrimSpace(in.PayModel)
		known := false
		for _, choice := range domain.PayModelChoices() {
			if choice.Key == payModel {
				known = true
				break
			}
		}
		if !known {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø±Ø§ Ø§Ø² ÙÙ‡Ø±Ø³Øª Ù…ØµÙˆØ¨ Ø§Ù†ØªØ®Ø§Ø¨ Ú©Ù†ÛŒØ¯.", nil)
			return
		}
		if payModel == domain.PayModelFourP && !fourPEnabled(deps) {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"Ù…Ø¯Ù„ Û´Ù¾ÛŒ Ø¯Ø± Ø§ÛŒÙ† Ø§Ø³ØªÙ‚Ø±Ø§Ø± ÙØ¹Ø§Ù„ Ù†ÛŒØ³ØªØ› Ù†Ø®Ø³Øª Ù…Ø§Ú˜ÙˆÙ„ Â«Ù†Ø¸Ø§Ù… Ø¬Ø¨Ø±Ø§Ù† Ø®Ø¯Ù…Ø§Øª Û´Ù¾ÛŒÂ» Ø±Ø§ ÙØ¹Ø§Ù„ Ú©Ù†ÛŒØ¯.", nil)
			return
		}
		effective := strings.TrimSpace(in.EffectiveDate)
		if effective == "" {
			effective = payTodayFa(deps)
		}
		if _, err := calendar.Parse(calendar.NormalizeDigits(effective)); err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"ØªØ§Ø±ÛŒØ® Ø§Ø¬Ø±Ø§ÛŒ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ù…Ø¹ØªØ¨Ø± Ù†ÛŒØ³Øª (Ù‚Ø§Ù„Ø¨ YYYY/MM/DD).", nil)
			return
		}
		noteFa := ""
		current, found := activePayModelRecord(r.Context(), svc, employeeID)
		if found && current.Fields["payModel"] == payModel {
			rows, err := payMethodRows(r.Context(), deps, svc, "")
			if err != nil {
				httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
				return
			}
			out := payMethodOptions(deps)
			out["items"] = rows
			out["noteFa"] = "Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø§ÛŒÙ† Ú©Ø§Ø±Ù…Ù†Ø¯ Ø§Ø² Ù¾ÛŒØ´ Ù‡Ù…ÛŒÙ† Ø¨ÙˆØ¯Ø› ØªØºÛŒÛŒØ±ÛŒ Ø«Ø¨Øª Ù†Ø´Ø¯."
			httpx.WriteJSON(w, http.StatusOK, out)
			return
		}
		actor := actorOf(r)
		if found {
			if _, err := svc.Act(r.Context(), current.ID, contracts.RecordAction{
				Action: "supersede",
				NoteFa: "ØªØºÛŒÛŒØ± Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø§Ø² Â«" + domain.PayModelTitleFa(current.Fields["payModel"]) + "Â» Ø¨Ù‡ Â«" +
					domain.PayModelTitleFa(payModel) + "Â»",
			}, actor); err != nil {
				httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
				return
			}
		}
		created, err := svc.Create(r.Context(), contracts.RecordInput{
			Kind:       domain.KindEmployeePayModel,
			Title:      "Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª â€” " + employeeLabel(r.Context(), deps, employeeID),
			EmployeeID: employeeID,
			Fields: map[string]string{
				"payModel":      payModel,
				"effectiveDate": calendar.NormalizeDigits(effective),
				"reasonFa":      strings.TrimSpace(in.ReasonFa),
			},
			NoteFa: "Ø«Ø¨Øª Ø§Ø² Ú©Ù†ØªØ±Ù„â€ŒÙ¾Ù†Ù„ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ú©Ø§Ø±Ú©Ù†Ø§Ù†",
		}, actor)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		// Ø±ÙˆØ´ ØªØ§Ø²Ù‡ Ø¨Ø§ÛŒØ¯ Â«Ø¬Ø§Ø±ÛŒÂ» Ø´ÙˆØ¯ ØªØ§ Ø¯Ø± Ù…Ø­Ø§Ø³Ø¨Ù‡ Ùˆ Ú©Ù†ØªØ±Ù„â€ŒÙ¾Ù†Ù„ Ø¯ÛŒØ¯Ù‡ Ø´ÙˆØ¯.
		if _, err := svc.Act(r.Context(), created.ID, contracts.RecordAction{
			Action: "activate", NoteFa: "Ø¬Ø§Ø±ÛŒâ€ŒØ³Ø§Ø²ÛŒ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø§Ø² Ú©Ù†ØªØ±Ù„â€ŒÙ¾Ù†Ù„",
		}, actor); err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		if payModel == domain.PayModelFourP && !fourPApplied(deps) {
			noteFa = "Ø±ÙˆØ´ Û´Ù¾ÛŒ Ø§Ù†ØªØ®Ø§Ø¨ Ø´Ø¯ØŒ ÙˆÙ„ÛŒ Ù¾Ø§Ø±Ø§Ù…ØªØ± Â«Ø§Ø¹Ù…Ø§Ù„ Ø¯Ø± Ù¾Ø±Ø¯Ø§Ø®ØªÂ» Ù…Ø¯Ù„ Û´Ù¾ÛŒ Ø¯Ø± ØªÙ†Ø¸ÛŒÙ…Ø§Øª Ø´Ø±Ú©Øª Ø®Ø§Ù…ÙˆØ´ Ø§Ø³ØªØ› " +
				"ØªØ§ Ø±ÙˆØ´Ù†â€ŒØ´Ø¯Ù† Ø¢Ù†ØŒ Ù…Ø­Ø§Ø³Ø¨Ù‡Ù” Ø­Ù‚ÙˆÙ‚ Ø¨Ø§ Ù…Ø¯Ù„ Ù¾ÛŒØ´ÛŒÙ† Ø§Ù†Ø¬Ø§Ù… Ù…ÛŒâ€ŒØ´ÙˆØ¯."
		}
		rows, err := payMethodRows(r.Context(), deps, svc, "")
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		out := payMethodOptions(deps)
		out["items"] = rows
		out["recordId"] = created.ID
		if noteFa != "" {
			out["noteFa"] = noteFa
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))
}

// payMethodOptions ÙÙ‡Ø±Ø³Øª Ø±ÙˆØ´â€ŒÙ‡Ø§ÛŒ Ù…ØµÙˆØ¨ Ùˆ ÙˆØ¶Ø¹ÛŒØª Ù…Ø¯Ù„ Û´Ù¾ÛŒ.
func payMethodOptions(deps modulekit.Deps) map[string]any {
	return map[string]any{
		"methods":      domain.PayModelChoices(),
		"grades":       nil,
		"fourPEnabled": fourPEnabled(deps),
		"fourPApplied": fourPApplied(deps),
		"today":        payTodayFa(deps),
		"independenceNoteFa": "Ø§Ù†ØªØ®Ø§Ø¨ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®ØªØŒ Ù…Ø­Ø§Ø³Ø¨Ù‡Ù” Ù…ÙˆØ¬ÙˆØ¯ Ø±Ø§ Ø¹ÙˆØ¶ Ù†Ù…ÛŒâ€ŒÚ©Ù†Ø¯: Ù…Ø¯Ù„ Ø­Ù‚ÙˆÙ‚ Ùˆ Ø¯Ø³ØªÙ…Ø²Ø¯ Ø¬Ø§Ø±ÛŒ Ùˆ Ù…Ø¯Ù„ " +
			"ØµÙ†Ø¯ÙˆÙ‚ Ù†ÙØª Ù…Ø³ØªÙ‚Ù„ Ùˆ Ø¬Ø§Ø±ÛŒâ€ŒØ§Ù†Ø¯ Ùˆ Ù…Ø¯Ù„ Û´Ù¾ÛŒ ØªÙˆØ³Ø¹Ù‡â€ŒØ§ÛŒ Ùˆ Ø§Ø®ØªÛŒØ§Ø±ÛŒ Ø§Ø³Øª. Ø§ÛŒÙ† Ø§Ù†ØªØ®Ø§Ø¨ ØªØ¹ÛŒÛŒÙ† Ù…ÛŒâ€ŒÚ©Ù†Ø¯ " +
			"Ú©Ø§Ø±Ø´Ù†Ø§Ø³ Ú©Ø¯Ø§Ù… ÙØ±Ù… Ùˆ Ú©Ø¯Ø§Ù… Ù…ÙˆØªÙˆØ± Ù…Ø­Ø§Ø³Ø¨Ù‡ Ø±Ø§ Ø¨Ø±Ø§ÛŒ Ø§ÛŒÙ† Ú©Ø§Ø±Ù…Ù†Ø¯ Ø¨Ù‡â€ŒÚ©Ø§Ø± Ø¨Ø¨Ø±Ø¯.",
	}
}

// payTodayFa ØªØ§Ø±ÛŒØ® Ø§Ù…Ø±ÙˆØ² Ø¨Ù‡ Ø´Ù…Ø³ÛŒ (YYYY/MM/DD) Ø¨Ø± Ù¾Ø§ÛŒÙ‡Ù” Ø³Ø§Ø¹Øª Ø³Ø§Ù…Ø§Ù†Ù‡.
func payTodayFa(deps modulekit.Deps) string {
	if deps.Clock == nil {
		return calendar.Format(time.Now())
	}
	return calendar.Format(deps.Clock())
}

// fourPEnabled Ø¢ÛŒØ§ Ù…Ø§Ú˜ÙˆÙ„ Ù…Ø¯Ù„ Û´Ù¾ÛŒ Ø¯Ø± Ø§ÛŒÙ† Ø§Ø³ØªÙ‚Ø±Ø§Ø± ÙØ¹Ø§Ù„ Ø§Ø³Øª.
func fourPEnabled(deps modulekit.Deps) bool {
	return deps.Registry != nil && deps.Registry.IsEnabled(compensationModuleID)
}

// fourPApplied Ø¢ÛŒØ§ Ù¾Ø§Ø±Ø§Ù…ØªØ± Ø´Ø±Ú©ØªØŒ Ø§Ø¹Ù…Ø§Ù„ Ù…Ø¯Ù„ Û´Ù¾ÛŒ Ø¯Ø± Ù¾Ø±Ø¯Ø§Ø®Øª Ø±Ø§ Ø±ÙˆØ´Ù† Ú©Ø±Ø¯Ù‡ Ø§Ø³Øª.
func fourPApplied(deps modulekit.Deps) bool {
	for _, p := range deps.Org.Params {
		if p.Key != "compensation.payroll.apply" {
			continue
		}
		v := strings.TrimSpace(p.Value)
		return v == "1" || strings.EqualFold(v, "yes") || strings.EqualFold(v, "true")
	}
	return false
}

// activePayModelRecord Ø±Ú©ÙˆØ±Ø¯ Ø¬Ø§Ø±ÛŒ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª ÛŒÚ© Ú©Ø§Ø±Ù…Ù†Ø¯.
func activePayModelRecord(ctx context.Context, svc *recordkit.Service, employeeID string) (contracts.Record, bool) {
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "payroll", Kind: domain.KindEmployeePayModel, EmployeeID: employeeID, Scope: "all", Size: 50,
	})
	if err != nil || len(page.Items) == 0 {
		return contracts.Record{}, false
	}
	var best contracts.Record
	found := false
	for _, rec := range page.Items {
		if rec.Status != "active" {
			continue
		}
		if !found || rec.Fields["effectiveDate"] > best.Fields["effectiveDate"] {
			best, found = rec, true
		}
	}
	return best, found
}

// payMethodRows Ú©Ø§Ø±Ú©Ù†Ø§Ù† Ø±Ø§ Ø¨Ø§ Ø±ÙˆØ´ Ù¾Ø±Ø¯Ø§Ø®Øª Ø¬Ø§Ø±ÛŒâ€ŒØ´Ø§Ù† ØªØ±Ú©ÛŒØ¨ Ù…ÛŒâ€ŒÚ©Ù†Ø¯Ø› Ú©Ø§Ø±Ù…Ù†Ø¯ Ø¨Ø¯ÙˆÙ† Ø±Ú©ÙˆØ±Ø¯ØŒ
// Â«Ù¾ÛŒØ´â€ŒÙØ±Ø¶ Ø³Ø§Ø²Ù…Ø§Ù†Â» (Ù…Ø¯Ù„ Ø­Ù‚ÙˆÙ‚ Ùˆ Ø¯Ø³ØªÙ…Ø²Ø¯ Ø¬Ø§Ø±ÛŒ) Ù†Ø´Ø§Ù† Ù…ÛŒâ€ŒØ¯Ù‡Ø¯.
func payMethodRows(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, query string) ([]payMethodRow, error) {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return nil, nil
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 300, Query: query})
	if err != nil {
		return nil, err
	}
	records, _ := svc.List(ctx, contracts.RecordFilter{
		Module: "payroll", Kind: domain.KindEmployeePayModel, Scope: "all", Size: 500,
	})
	current := map[string]contracts.Record{}
	for _, rec := range records.Items {
		if rec.Status != "active" || rec.EmployeeID == "" {
			continue
		}
		if previous, ok := current[rec.EmployeeID]; !ok || rec.Fields["effectiveDate"] > previous.Fields["effectiveDate"] {
			current[rec.EmployeeID] = rec
		}
	}
	rows := make([]payMethodRow, 0, len(page.Items))
	for _, emp := range page.Items {
		row := payMethodRow{
			EmployeeID: emp.ID, FullName: emp.FullName, DepartmentName: emp.DepartmentName,
			PayModel: domain.PayModelPayroll, PayModelFa: domain.PayModelTitleFa(domain.PayModelPayroll),
			SourceFa: "Ù¾ÛŒØ´â€ŒÙØ±Ø¶ Ø³Ø§Ø²Ù…Ø§Ù†",
		}
		if rec, ok := current[emp.ID]; ok {
			row.RecordID = rec.ID
			row.PayModel = domain.PayModelOf(rec.Fields["payModel"])
			row.PayModelFa = domain.PayModelTitleFa(row.PayModel)
			row.EffectiveDate = rec.Fields["effectiveDate"]
			row.Status = rec.Status
			row.StatusFa = rec.StatusFa
			row.SourceFa = "Ø«Ø¨Øªâ€ŒØ´Ø¯Ù‡"
			row.NoteFa = rec.Fields["reasonFa"]
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].EmployeeID < rows[j].EmployeeID })
	return rows, nil
}

// employeeLabel Ù†Ø§Ù… Ù†Ù…Ø§ÛŒØ´ÛŒ Ú©Ø§Ø±Ù…Ù†Ø¯ Ø¨Ø±Ø§ÛŒ Ø¹Ù†ÙˆØ§Ù† Ø±Ú©ÙˆØ±Ø¯.
func employeeLabel(ctx context.Context, deps modulekit.Deps, employeeID string) string {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return employeeID
	}
	emp, found, err := reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
	if err != nil || !found {
		return employeeID
	}
	return emp.FullName
}
