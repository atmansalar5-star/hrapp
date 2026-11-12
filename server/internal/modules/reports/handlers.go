// Ù¾Ø§ÛŒØ§Ù†Ù‡â€ŒÙ‡Ø§ÛŒ HTTP Ù…Ø§Ú˜ÙˆÙ„ Â«reportsÂ».

package reports

import (
	"context"
	"net/http"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/datasource"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/repfilter"
)

// registerBuilderEndpoints Ù…Ø³ÛŒØ±Ù‡Ø§ÛŒ Ú¯Ø²Ø§Ø±Ø´â€ŒØ³Ø§Ø² Ø±Ø§ Ø«Ø¨Øª Ù…ÛŒâ€ŒÚ©Ù†Ø¯: ÙÙ‡Ø±Ø³Øª Ù…Ù†Ø§Ø¨Ø¹ Ø¯Ø§Ø¯Ù‡ Ùˆ Ø§Ø¬Ø±Ø§ÛŒ
// Ú¯Ø²Ø§Ø±Ø´ Ø¨Ø± Ù¾Ø§ÛŒÙ‡Ù” Ø§Ù‚Ù„Ø§Ù… Ø§Ù†ØªØ®Ø§Ø¨â€ŒØ´Ø¯Ù‡. Ù…Ù†Ø§Ø¨Ø¹ Ø¯Ø§Ø¯Ù‡ Ø§Ø² Ù¾ÙˆØ±Øª Ù…Ø´ØªØ±Ú© Ù…ÛŒâ€ŒØ¢ÛŒØ¯Ø› Ù‡Ø± Ù…Ø§Ú˜ÙˆÙ„ Ù…Ù†Ø¨Ø¹ Ø®ÙˆØ¯Ø´
// Ø±Ø§ Ø«Ø¨Øª Ù…ÛŒâ€ŒÚ©Ù†Ø¯ Ùˆ Ø§ÛŒÙ† Ù…Ø§Ú˜ÙˆÙ„ Ù‡ÛŒÚ† Ø¯Ø§Ø¯Ù‡â€ŒØ§ÛŒ Ø±Ø§ Ù…Ø³ØªÙ‚ÛŒÙ… Ø§Ø² Ù…Ø§Ú˜ÙˆÙ„ Ø¯ÛŒÚ¯Ø± Ù†Ù…ÛŒâ€ŒØ®ÙˆØ§Ù†Ø¯.
func registerBuilderEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	report := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("reports", "report", fn) }

	// Ù‚Ø§Ù„Ø¨ ÙˆØ±ÙˆØ¯ Ø§Ø·Ù„Ø§Ø¹Ø§Øª Ø¯Ø³ØªÛŒ Ø³ØªÙˆÙ†â€ŒÙ‡Ø§ÛŒ ÛŒÚ© Ù…Ù†Ø¨Ø¹ Ú¯Ø²Ø§Ø±Ø´: Ù†ÙˆØ¹ØŒ Ø§Ù„Ú¯ÙˆØŒ Ù…Ù‚Ø§Ø¯ÛŒØ± Ù…Ø¬Ø§Ø² Ùˆ Ù†Ù…ÙˆÙ†Ù‡.
	// Ù…Ù†Ø¨Ø¹ Ù‡Ù…Ø§Ù† Ø³Ø§Ø®ØªØ§Ø± Ø¯Ø§Ø¯Ù‡ Ø§Ø³Øª Ú©Ù‡ Ù‚Ø§Ù„Ø¨ Ø§Ú©Ø³Ù„ ÙØ±Ù…â€ŒÙ‡Ø§ Ø§Ø² Ø¢Ù† Ø³Ø§Ø®ØªÙ‡ Ù…ÛŒâ€ŒØ´ÙˆØ¯.
	mux.HandleFunc("GET /api/v1/reports/entry-format", report(func(w http.ResponseWriter, r *http.Request) {
		source := strings.TrimSpace(r.URL.Query().Get("source"))
		if source == "" {
			list := make([]map[string]any, 0)
			for _, s := range deps.Ports.ReportSources() {
				meta := s.Metadata()
				list = append(list, map[string]any{
					"key": meta.Key, "titleFa": meta.TitleFa, "fields": len(meta.Fields),
				})
			}
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"sources": list, "total": len(list)})
			return
		}
		s, ok := deps.Ports.ReportSource(source)
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound,
				"Ù…Ù†Ø¨Ø¹ Ø¯Ø§Ø¯Ù‡Ù” Ú¯Ø²Ø§Ø±Ø´ Ø¨Ø§ Ø§ÛŒÙ† Ø´Ù†Ø§Ø³Ù‡ Ù¾ÛŒØ¯Ø§ Ù†Ø´Ø¯Ø› ÙÙ‡Ø±Ø³Øª Ø±Ø§ Ø§Ø² Ù‡Ù…ÛŒÙ† Ù…Ø³ÛŒØ± Ø¨Ø¯ÙˆÙ† Ù¾Ø§Ø±Ø§Ù…ØªØ± Ø¨Ú¯ÛŒØ±ÛŒØ¯.", nil)
			return
		}
		meta := s.Metadata()
		format := datasource.EntryFormatFromReportFields(meta.Fields)
		filters := make([]map[string]string, 0, len(meta.Filters))
		for _, f := range meta.Filters {
			filters = append(filters, map[string]string{"key": f.Key, "titleFa": f.LabelFa, "type": f.Type})
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"source": meta.Key, "titleFa": meta.TitleFa,
			"format": format, "sampleRow": datasource.SampleRow(format),
			"filters": filters,
			"noteFa": "Ø§ÛŒÙ† Ø¨Ø±Ú¯Ù‡ Ù…ÛŒâ€ŒÚ¯ÙˆÛŒØ¯ Ø³ØªÙˆÙ†â€ŒÙ‡Ø§ÛŒ Ø§ÛŒÙ† Ú¯Ø²Ø§Ø±Ø´ Ú†Ù‡ Ù†ÙˆØ¹ÛŒ Ø¯Ø§Ø±Ù†Ø¯ Ùˆ Ù…Ù‚Ø¯Ø§Ø± Ø±Ø§ Ú†Ù‡ Ø´Ú©Ù„ÛŒ ÙˆØ§Ø±Ø¯ Ú©Ù†ÛŒØ¯Ø› " +
				"Ù‡Ù…Ø§Ù† Ø§Ù„Ú¯ÙˆÛŒ Ù‚Ø§Ù„Ø¨ Ø¯Ø³ØªÛŒ ÙØ±Ù…â€ŒÙ‡Ø§ Ø§Ø³Øª.",
		})
	}))

	mux.HandleFunc("GET /api/v1/reports/sources", report(func(w http.ResponseWriter, r *http.Request) {
		sources := deps.Ports.ReportSources()
		metas := make([]contracts.ReportSourceMeta, 0, len(sources))
		for _, s := range sources {
			metas = append(metas, s.Metadata())
		}
		// Ø¹Ù…Ù„Ú¯Ø±Ù‡Ø§ÛŒ Ø´Ø±Ø· ÛŒÚ©â€ŒØ¨Ø§Ø± Ùˆ Ù…Ø´ØªØ±Ú© Ø§Ø¹Ù„Ø§Ù… Ù…ÛŒâ€ŒØ´ÙˆÙ†Ø¯Ø› Ø±Ø§Ø¨Ø· Ø´Ø±Ø·â€ŒØ³Ø§Ø² Ø±Ø§ Ø§Ø² Ù‡Ù…ÛŒÙ† Ù…ÛŒâ€ŒØ³Ø§Ø²Ø¯
		// ØªØ§ Ø¨Ø±Ú†Ø³Ø¨ Ø¹Ù…Ù„Ú¯Ø±Ù‡Ø§ Ø¯Ø± UI Ù‡Ø§Ø±Ø¯Ú©Ø¯ Ù†Ø´ÙˆØ¯.
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"sources": metas, "operators": repfilter.Operators(), "total": len(metas),
		})
	}))

	mux.HandleFunc("POST /api/v1/reports/run", report(func(w http.ResponseWriter, r *http.Request) {
		var req contracts.ReportRequest
		if !httpx.DecodeJSON(w, r, &req) {
			return
		}
		if err := repfilter.Validate(req.Conditions); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		result, err := runReport(r.Context(), deps, req)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		deps.RecordAudit(r, "reports", "report-run", req.Source, "Ø§Ø¬Ø±Ø§ÛŒ Ú¯Ø²Ø§Ø±Ø´ Ø¨Ø± Ù¾Ø§ÛŒÙ‡Ù” Ø§Ù‚Ù„Ø§Ù… Ø§Ù†ØªØ®Ø§Ø¨ÛŒ")
		httpx.WriteJSON(w, http.StatusOK, result)
	}))
}

// runReport Ù…Ù†Ø¨Ø¹ Ø¯Ø§Ø¯Ù‡Ù” Ø®ÙˆØ§Ø³ØªÙ‡â€ŒØ´Ø¯Ù‡ Ø±Ø§ Ù¾ÛŒØ¯Ø§ Ùˆ Ú¯Ø²Ø§Ø±Ø´ Ø±Ø§ Ø§Ø¬Ø±Ø§ Ù…ÛŒâ€ŒÚ©Ù†Ø¯ Ùˆ Ø³Ø±ØµÙØ­Ù‡Ù” Ø¨Ø±Ù†Ø¯ Ø´Ø±Ú©Øª
// ÙØ¹Ø§Ù„ Ø±Ø§ Ø±ÙˆÛŒ Ù†ØªÛŒØ¬Ù‡ Ù…ÛŒâ€ŒÙ†Ø´Ø§Ù†Ø¯.
func runReport(ctx context.Context, deps modulekit.Deps, req contracts.ReportRequest) (contracts.ReportResult, error) {
	source, ok := deps.Ports.ReportSource(req.Source)
	if !ok {
		// Ù†Ø¨ÙˆØ¯ Ù…Ù†Ø¨Ø¹: Ø¨Ù‡â€ŒØ¬Ø§ÛŒ Ø®Ø·Ø§ÛŒ Ù…Ø¨Ù‡Ù…ØŒ ÙÙ‡Ø±Ø³Øª Ù…Ù†Ø§Ø¨Ø¹ Ù…ÙˆØ¬ÙˆØ¯ Ø±Ø§ Ø¨Ø±Ù…ÛŒâ€ŒÚ¯Ø±Ø¯Ø§Ù†ÛŒÙ….
		return contracts.ReportResult{}, &unknownSourceError{source: req.Source, available: availableSources(deps)}
	}
	result, err := source.Rows(ctx, req)
	if err != nil {
		return contracts.ReportResult{}, err
	}
	// Ø§Ù‚Ù„Ø§Ù… Ø§Ù†ØªØ®Ø§Ø¨ÛŒ Ø¨Ù‡ Ù‡Ù…Ø§Ù† ØªØ±ØªÛŒØ¨ Ø³ØªÙˆÙ†â€ŒÙ‡Ø§Ø› ØªØ§ Ø®Ø±ÙˆØ¬ÛŒ Ú†Ø§Ù¾ÛŒ/Ø°Ø®ÛŒØ±Ù‡â€ŒØ´Ø¯Ù‡ Ø¨ØªÙˆØ§Ù†Ø¯ Ù‡Ù…ÛŒÙ† Ú¯Ø²Ø§Ø±Ø´ Ø±Ø§
	// Ø¨Ø§Ø²Ø³Ø§Ø²ÛŒ Ú©Ù†Ø¯ØŒ Ø¨Ø¯ÙˆÙ† Ø¢Ù†Ú©Ù‡ Ú©Ù„Ø§ÛŒÙ†Øª Ù…Ø¬Ø¨ÙˆØ± Ø¨Ø§Ø´Ø¯ Ø§Ù†ØªØ®Ø§Ø¨â€ŒÙ‡Ø§ Ø±Ø§ Ø¬Ø¯Ø§ Ù†Ú¯Ù‡ Ø¯Ø§Ø±Ø¯.
	items := make([]string, 0, len(result.Columns))
	for _, c := range result.Columns {
		items = append(items, c.Key)
	}
	result.Items = items
	result.Brand = brandOf(ctx, deps)
	result.Print = reportPrintChoice(deps, req.Print)
	return result, nil
}

// reportPrintChoice ØªØµÙ…ÛŒÙ… Â«Ù†Ù…Ø§ÛŒØ´ ØªÙˆØ¶ÛŒØ­Ø§Øª Ø¯Ø± Ø¨Ø±Ú¯Ù‡Ù” Ú†Ø§Ù¾ÛŒÂ» Ø±Ø§ Ù…ÛŒâ€ŒØ³Ø§Ø²Ø¯.
//
// Ø§ÙˆÙ„ÙˆÛŒØª: Ø§Ù†ØªØ®Ø§Ø¨ ØµØ±ÛŒØ­ Ù‡Ù…ÛŒÙ† Ø§Ø¬Ø±Ø§ØŒ Ø³Ù¾Ø³ Ù¾Ø§Ø±Ø§Ù…ØªØ± Ø´Ø±Ú©Øª `reports.print.showDescription`
// (Ù¾ÛŒØ´â€ŒÙØ±Ø¶: Ø±ÙˆØ´Ù†). Ù‚Ø§Ø¹Ø¯Ù‡ Ø¯Ø± Ø³Ø±ÙˆØ± Ø§Ø³Øª ØªØ§ Ø±Ø§Ø¨Ø· Ú©Ø§Ø±Ø¨Ø±ÛŒ ÙÙ‚Ø· Ù†Ù…Ø§ÛŒØ´ Ø¯Ù‡Ø¯ Ùˆ Ø±ÙØªØ§Ø± Ø¯Ø± Ù‡Ù…Ù‡Ù”
// Ø®Ø±ÙˆØ¬ÛŒâ€ŒÙ‡Ø§ (Ú†Ø§Ù¾ØŒ PDFØŒ Ø®Ø±ÙˆØ¬ÛŒ Ø³Ø±ÙˆØ±ÛŒ) ÛŒÚ©ÛŒ Ø¨Ù…Ø§Ù†Ø¯.
func reportPrintChoice(deps modulekit.Deps, choice *contracts.ReportPrintChoice) *contracts.ReportPrint {
	show := true
	source := "Ù¾Ø§Ø±Ø§Ù…ØªØ± Ø´Ø±Ú©Øª reports.print.showDescription"
	for _, p := range deps.Org.Params {
		if p.Key != "reports.print.showDescription" {
			continue
		}
		v := strings.TrimSpace(p.Value)
		show = !(v == "0" || strings.EqualFold(v, "no") || strings.EqualFold(v, "false"))
	}
	if choice != nil && choice.ShowDescription != nil {
		show = *choice.ShowDescription
		source = "Ø§Ù†ØªØ®Ø§Ø¨ Ù‡Ù…ÛŒÙ† Ø§Ø¬Ø±Ø§"
	}
	note := "Ø¨Ø®Ø´ ØªÙˆØ¶ÛŒØ­Ø§Øª Ø¯Ø± Ø¨Ø±Ú¯Ù‡Ù” Ú†Ø§Ù¾ÛŒ Ù†Ù…Ø§ÛŒØ´ Ø¯Ø§Ø¯Ù‡ Ù…ÛŒâ€ŒØ´ÙˆØ¯."
	if !show {
		note = "Ø¨Ø®Ø´ ØªÙˆØ¶ÛŒØ­Ø§Øª Ø¯Ø± Ø¨Ø±Ú¯Ù‡Ù” Ú†Ø§Ù¾ÛŒ Ù†Ù…Ø§ÛŒØ´ Ø¯Ø§Ø¯Ù‡ Ù†Ù…ÛŒâ€ŒØ´ÙˆØ¯."
	}
	return &contracts.ReportPrint{ShowDescription: show, SourceFa: source, NoteFa: note}
}

// brandOf Ø³Ø±ØµÙØ­Ù‡Ù” Ø¨Ø±Ù†Ø¯ Ú¯Ø²Ø§Ø±Ø´ Ø±Ø§ Ø§Ø² Ø§Ø³ØªÙ‚Ø±Ø§Ø± Ù¾Ø§Ø±Ø§Ù…ØªØ±ÛŒÚ© Ø´Ø±Ú©Øª ÙØ¹Ø§Ù„ Ù…ÛŒâ€ŒØ³Ø§Ø²Ø¯.
// Ù‡ÛŒÚ† Ù†Ø§Ù… ÛŒØ§ Ù†Ø´Ø§Ù†ÛŒ Ø§ÛŒÙ†Ø¬Ø§ Ø³Ø§Ø®ØªÙ‡ Ù†Ù…ÛŒâ€ŒØ´ÙˆØ¯Ø› Ù‡Ù…Ù‡ Ø§Ø² Ù¾Ø±ÙˆÙØ§ÛŒÙ„ Ø´Ø±Ú©Øª Ù…ÛŒâ€ŒØ¢ÛŒØ¯ØŒ Ù¾Ø³ Ú¯Ø²Ø§Ø±Ø´ Ø´Ø±Ú©Øªâ€ŒÙ‡Ø§ÛŒ
// Ù…Ø®ØªÙ„Ù Ø¨Ø§ Ù‡ÙˆÛŒØª Ø®ÙˆØ¯Ø´Ø§Ù† Ú†Ø§Ù¾ Ù…ÛŒâ€ŒØ´ÙˆØ¯.
func brandOf(ctx context.Context, deps modulekit.Deps) *contracts.ReportBrand {
	brand := &contracts.ReportBrand{
		TitleFa:    deps.Org.TitleFa,
		HoldingFa:  deps.Org.HoldingFa,
		ShortCode:  deps.Org.ShortCode,
		LogoURL:    logoURL(deps.Org.LogoFile),
		BrandColor: deps.Org.BrandColor,
	}
	now := time.Now()
	if deps.Clock != nil {
		now = deps.Clock()
	}
	if d, err := calendar.FromGregorian(now); err == nil {
		brand.GeneratedAt = d.String()
	}
	if user, ok := auth.UserFrom(ctx); ok {
		brand.GeneratedBy = user.Username
	}
	return brand
}

// logoURL Ù…Ø³ÛŒØ± Ù†Ø´Ø§Ù† Ø´Ø±Ú©Øª Ø±Ø§ Ø¨Ù‡ Ø¢Ø¯Ø±Ø³ Ù‚Ø§Ø¨Ù„ Ù…ØµØ±Ù Ø¯Ø± Ø®Ø±ÙˆØ¬ÛŒ Ú¯Ø²Ø§Ø±Ø´ ØªØ¨Ø¯ÛŒÙ„ Ù…ÛŒâ€ŒÚ©Ù†Ø¯. Ú¯Ø²Ø§Ø±Ø´ Ø±ÙˆÛŒ
// Ù‡Ù…Ø§Ù† Ù…Ø¨Ø¯Ø£ Ø±Ø§Ø¨Ø· ÙˆØ¨ Ø³Ø±Ùˆ Ù…ÛŒâ€ŒØ´ÙˆØ¯ØŒ Ù¾Ø³ Ù…Ø³ÛŒØ± Ù†Ø³Ø¨ÛŒ Ù¾Ø±ÙˆÙØ§ÛŒÙ„ Ø´Ø±Ú©Øª Ú©Ø§ÙÛŒ Ø§Ø³Øª.
func logoURL(file string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	return "/" + strings.TrimPrefix(file, "/")
}

func availableSources(deps modulekit.Deps) []string {
	out := []string{}
	for _, s := range deps.Ports.ReportSources() {
		out = append(out, s.Metadata().Key)
	}
	return out
}

type unknownSourceError struct {
	source    string
	available []string
}

func (e *unknownSourceError) Error() string {
	msg := "Ù…Ù†Ø¨Ø¹ Ø¯Ø§Ø¯Ù‡Ù” Ú¯Ø²Ø§Ø±Ø´ Â«" + e.source + "Â» Ø´Ù†Ø§Ø®ØªÙ‡ Ù†Ø´Ø¯."
	if len(e.available) > 0 {
		msg += " Ù…Ù†Ø§Ø¨Ø¹ Ù…ÙˆØ¬ÙˆØ¯: "
		for i, k := range e.available {
			if i > 0 {
				msg += "ØŒ "
			}
			msg += k
		}
	}
	return msg
}
