// پیاده‌سازی ماژول «payroll» — «reconcile»: تطبیق کارکرد دوره با حقوق (زیرکار ۱.۳
// فاز ۱ نقشهٔ راه).
//
// این گزارش همان میزی است که برای «تطبیق رقم‌به‌رقم یک دورهٔ واقعی با محاسبهٔ دستی»
// لازم است: برای هر کارمند، کارکرد ثبت‌شده (روز موظفی، تردد، غیبت، تأخیر/تعجیل،
// شب‌کاری) کنار مبلغ محاسبه‌شدهٔ موتور قواعد و مبلغ نوشته‌شده روی دورهٔ پردازش
// می‌آید و مغایرت‌ها فهرست می‌شود.
//
// قاعدهٔ معماری: هیچ مبلغی اینجا ساخته نمی‌شود؛ مبلغ از `rules.ComputePayroll` و
// کارکرد از پورت قراردادی ماژول حضور می‌آید. مبالغ «مشورتی» مزایای نوبت‌کاری/شب‌کاری/
// جمعه‌کاری هم فقط برای نشان‌دادن اختلاف موتور فعلی گزارش می‌شوند و در محاسبه نوشته
// نمی‌شوند (موتور قواعد با هستهٔ Rust هم‌ارزی‌سنجی می‌شود و بی‌fixture تغییر نمی‌کند).

package payroll

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registerReconcileEndpoints مسیر گزارش تطبیق کارکرد با حقوق را ثبت می‌کند.
func registerReconcileEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }

	mux.HandleFunc("GET /api/v1/payroll/worktime-reconciliation", read(func(w http.ResponseWriter, r *http.Request) {
		period := strings.TrimSpace(calendar.NormalizeDigits(r.URL.Query().Get("period")))
		if period == "" {
			period = periodFa(deps)
		}
		rows, noteFa, err := workTimeRows(r.Context(), deps, svc, period)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		wt := workTimeParams(deps, r.Context(), svc)
		factors := shiftFactors(r.Context(), deps, svc)

		out := make([]reconcileRow, 0, len(rows))
		premiumTotal := int64(0)
		mismatches := 0
		for _, row := range rows {
			hourly := rules.HourlyWage(row.BaseSalary, wt)
			item := reconcileRow{
				workTimeRow:   row,
				nightFactor:   factors["nightShift.factor"].Value,
				fridayFactor:  factors["friday.factor"].Value,
				NightPremium:  rules.RoundHalfUp(hourly * row.NightHours * (factors["nightShift.factor"].Value - 1)),
				FridayPremium: rules.RoundHalfUp(hourly * row.FridayHours * (factors["friday.factor"].Value - 1)),
			}
			item.PremiumTotal = item.NightPremium + item.FridayPremium
			item.WithPremiums = row.Net + item.PremiumTotal
			item.IssuesFa = issuesOf(row)
			item.Mismatch = len(item.IssuesFa) > 0
			if item.Mismatch {
				mismatches++
			}
			premiumTotal += item.PremiumTotal
			out = append(out, item)
		}

		summary := summarize(rows)
		comparison := compareWithRun(r.Context(), svc, period, summary)

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"period":               period,
			"rows":                 out,
			"total":                len(out),
			"sum":                  summary,
			"run":                  comparison,
			"factors":              factorList(factors),
			"mismatchCount":        mismatches,
			"advisoryPremiumTotal": premiumTotal,
			"noteFa": noteFa + " این گزارش کارکرد و مبلغ را کنار هم می‌گذارد تا تطبیق " +
				"رقم‌به‌رقم با محاسبهٔ دستی ممکن شود؛ مغایرت‌ها فقط گزارش می‌شوند و خودکار اصلاح نمی‌شوند.",
		})
	}))
}

// reconcileRow یک ردیف تطبیق: کارکرد کارمند + مبلغ محاسبه‌شده + مزایای مشورتی نوبت‌کاری.
type reconcileRow struct {
	workTimeRow
	nightFactor   float64
	fridayFactor  float64
	NightPremium  int64 `json:"nightPremiumAdvisory"`
	FridayPremium int64 `json:"fridayPremiumAdvisory"`
	// PremiumTotal و WithPremiums جمع مزایای محاسبه‌نشدهٔ موتور فعلی و خالص با آن‌ها.
	PremiumTotal int64    `json:"premiumAdvisoryTotal"`
	WithPremiums int64    `json:"netWithAdvisoryPremiums"`
	IssuesFa     []string `json:"issuesFa,omitempty"`
	Mismatch     bool     `json:"mismatch"`
}

// runComparison سنجش جمع محاسبه‌شدهٔ گزارش با آنچه روی رکورد دورهٔ پردازش نوشته شده.
type runComparison struct {
	RunID       string `json:"runId,omitempty"`
	Period      string `json:"period"`
	Status      string `json:"status,omitempty"`
	Found       bool   `json:"found"`
	StoredGross int64  `json:"storedGross"`
	StoredTax   int64  `json:"storedTax"`
	StoredIns   int64  `json:"storedInsurance"`
	StoredNet   int64  `json:"storedNet"`
	DiffGross   int64  `json:"diffGross"`
	DiffTax     int64  `json:"diffTax"`
	DiffIns     int64  `json:"diffInsurance"`
	DiffNet     int64  `json:"diffNet"`
	Matched     bool   `json:"matched"`
	NoteFa      string `json:"noteFa"`
}

// compareWithRun جمع محاسبه‌شده را با رکورد دورهٔ پردازش همان دوره می‌سنجد.
func compareWithRun(ctx context.Context, svc *recordkit.Service, period string, summary workTimeTotals) runComparison {
	out := runComparison{Period: period}
	run, ok := payrollRunFor(ctx, svc, period)
	if !ok {
		out.NoteFa = "برای این دوره رکورد پردازش (payrollRuns) پیدا نشد؛ فقط جمع محاسبه‌شده گزارش می‌شود."
		return out
	}
	out.RunID = run.ID
	out.Status = run.Status
	out.Found = true
	out.StoredGross = intField(run.Fields["gross"])
	out.StoredTax = intField(run.Fields["tax"])
	out.StoredIns = intField(run.Fields["insurance"])
	out.StoredNet = intField(run.Fields["net"])
	out.DiffGross = summary.Gross - out.StoredGross
	out.DiffTax = summary.Tax - out.StoredTax
	out.DiffIns = summary.InsuranceEmployee - out.StoredIns
	out.DiffNet = summary.Net - out.StoredNet
	out.Matched = out.DiffGross == 0 && out.DiffTax == 0 && out.DiffIns == 0 && out.DiffNet == 0
	if out.Matched {
		out.NoteFa = "جمع محاسبه‌شده با مبلغ نوشته‌شده روی دورهٔ پردازش یکی است (تطبیق کامل)."
		return out
	}
	out.NoteFa = "جمع محاسبه‌شده با مبلغ نوشته‌شده روی دورهٔ پردازش یکی نیست؛ تفاوت در " +
		"فیلدهای diff آمده است (دورهٔ پردازش شاید پیش از تغییر کارکرد محاسبه شده است)."
	return out
}

// payrollRunFor رکورد دورهٔ پردازش یک دورهٔ شمسی را پیدا می‌کند.
func payrollRunFor(ctx context.Context, svc *recordkit.Service, period string) (contracts.Record, bool) {
	if svc == nil {
		return contracts.Record{}, false
	}
	page, err := svc.List(ctx, contracts.RecordFilter{Module: "payroll", Kind: "payrollRuns", Size: 200, Scope: "all"})
	if err != nil {
		return contracts.Record{}, false
	}
	for _, rec := range page.Items {
		if strings.TrimSpace(rec.Fields["period"]) == period {
			return rec, true
		}
	}
	return contracts.Record{}, false
}

// issuesOf مغایرت‌های یک ردیف کارکرد را به فارسی می‌نویسد.
func issuesOf(row workTimeRow) []string {
	out := []string{}
	if row.RecordsCount == 0 {
		out = append(out, "در این دوره هیچ کارکرد مؤثری برای این کارمند ثبت نشده است.")
	}
	if row.PendingRecords > 0 {
		out = append(out, fmt.Sprintf("%d رکورد در گردش است و در این محاسبه اثر ندارد.", row.PendingRecords))
	}
	if row.MissingDays > 0 {
		out = append(out, fmt.Sprintf("%s روز موظفی نوبت مصوب، بی‌تردد و بی‌مرخصی است (غیبت).",
			trimFloatFa(row.MissingDays)))
	}
	if row.ExpectedDays > 0 && row.WorkedDays < row.ExpectedDays {
		out = append(out, fmt.Sprintf("روزهای کارکرد (%s) از روزهای موظفی (%s) کمتر است؛ مبنای کسر صریح نشده است.",
			trimFloatFa(row.WorkedDays), trimFloatFa(row.ExpectedDays)))
	}
	if row.LateMinutes > 0 || row.EarlyLeaveMinutes > 0 {
		out = append(out, fmt.Sprintf("تأخیر %d دقیقه و تعجیل %d دقیقه ثبت شده است؛ کسر آن هنوز در موتور حقوق نیست.",
			row.LateMinutes, row.EarlyLeaveMinutes))
	}
	if row.NightHours > 0 || row.FridayHours > 0 {
		out = append(out, fmt.Sprintf("شب‌کاری %s ساعت و کار تعطیل/پایان هفته %s ساعت ثبت شده است که موتور جاری مزایای آن را محاسبه نمی‌کند (مبلغ مشورتی در همین ردیف آمده است).",
			trimFloatFa(row.NightHours), trimFloatFa(row.FridayHours)))
	}
	if row.PendingOvertimeHours > 0 {
		out = append(out, fmt.Sprintf("%s ساعت اضافه‌کاری در انتظار تأیید است.", trimFloatFa(row.PendingOvertimeHours)))
	}
	if row.PaidLeaveDays > 0 || row.UnpaidLeaveDays > 0 || row.SickLeaveDays > 0 || row.MissionDays > 0 {
		out = append(out, fmt.Sprintf("مرخصی استحقاقی %s، بدون حقوق %s، استعلاجی %s و مأموریت %s روز در کارکرد لحاظ شده است.",
			trimFloatFa(row.PaidLeaveDays), trimFloatFa(row.UnpaidLeaveDays),
			trimFloatFa(row.SickLeaveDays), trimFloatFa(row.MissionDays)))
	}
	return out
}

// workTimeParams پارامترهای کارکرد (ساعت ماهانه از پارامتر شرکت فعال).
func workTimeParams(deps modulekit.Deps, ctx context.Context, svc *recordkit.Service) rules.WorkTimeParams {
	out := rules.DefaultWorkTimeParams()
	_, values, _ := collectParams(ctx, deps, svc)
	if hours, ok := intOf(values["payroll.monthlyHours"]); ok && hours > 0 {
		out.MonthHours = float64(hours)
	}
	for _, p := range deps.Org.Params {
		if p.Key != "payroll.monthlyHours" {
			continue
		}
		if hours, ok := intOf(p.Value); ok && hours > 0 {
			out.MonthHours = float64(hours)
		}
	}
	return out
}

// shiftFactor ضریب پارامتریک یک مزیت با منبعش (برای شفافیت گزارش).
type shiftFactor struct {
	Key      string  `json:"key"`
	TitleFa  string  `json:"titleFa"`
	Value    float64 `json:"value"`
	UnitFa   string  `json:"unitFa,omitempty"`
	SourceFa string  `json:"sourceFa"`
}

// shiftFactors ضریب نوبت‌کاری و جمعه‌کاری را از پارامتر شرکت فعال و رکوردهای پارامتر
// فعال می‌خواند. رکورد پارامتر (نسخهٔ تأییدشده) بر پارامتر پایهٔ شرکت اولویت دارد.
func shiftFactors(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) map[string]shiftFactor {
	out := map[string]shiftFactor{
		"nightShift.factor": {Key: "nightShift.factor", TitleFa: "ضریب نوبت‌کاری (شب‌کاری)", Value: 1.35, UnitFa: "برابر", SourceFa: "پیش‌فرض سامانه"},
		"friday.factor":     {Key: "friday.factor", TitleFa: "ضریب جمعه‌کاری", Value: 1.4, UnitFa: "برابر", SourceFa: "پیش‌فرض سامانه"},
	}
	for _, p := range deps.Org.Params {
		if item, ok := out[p.Key]; ok {
			if v, err := strconv.ParseFloat(strings.TrimSpace(calendar.NormalizeDigits(p.Value)), 64); err == nil && v > 0 {
				item.Value = v
				item.TitleFa = firstNonEmptyFa(p.TitleFa, item.TitleFa)
				item.UnitFa = firstNonEmptyFa(p.UnitFa, item.UnitFa)
				item.SourceFa = "پارامتر شرکت فعال"
				out[p.Key] = item
			}
		}
	}
	if svc != nil {
		if page, err := svc.List(ctx, contracts.RecordFilter{Module: "payroll", Kind: "payParams", Size: 500, Scope: "all"}); err == nil {
			for _, rec := range page.Items {
				if rec.Status != "active" {
					continue
				}
				key := strings.TrimSpace(rec.Fields["key"])
				item, ok := out[key]
				if !ok {
					continue
				}
				if v, err := strconv.ParseFloat(strings.TrimSpace(calendar.NormalizeDigits(rec.Fields["value"])), 64); err == nil && v > 0 {
					item.Value = v
					item.SourceFa = "رکورد پارامتر فعال"
					out[key] = item
				}
			}
		}
	}
	return out
}

// factorList فهرست مرتب ضرایب برای پاسخ API.
func factorList(factors map[string]shiftFactor) []shiftFactor {
	out := make([]shiftFactor, 0, len(factors))
	for _, key := range []string{"nightShift.factor", "friday.factor"} {
		if item, ok := factors[key]; ok {
			out = append(out, item)
		}
	}
	return out
}

// intField مقدار عددی یک فیلد رکورد (صفر اگر خالی یا نامعتبر باشد).
func intField(raw string) int64 {
	raw = strings.TrimSpace(calendar.NormalizeDigits(raw))
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return int64(v)
}

// trimFloatFa عدد اعشاری را بدون صفرهای اضافی می‌نویسد.
func trimFloatFa(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }

// firstNonEmptyFa نخستین رشتهٔ غیرخالی.
func firstNonEmptyFa(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
