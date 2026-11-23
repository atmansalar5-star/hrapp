// مسیرهای «تخصیص فوق العاده‌های حقوق» — کد رسمی به کارمند یا گروهی از کارکنان.
//
// سه کار انجام می‌شود:
//   - پیش‌فرض‌ها: جدول رسمی کدها، روش‌های محاسبه، مبناها، دامنه‌ها و کارکنان با ویژگی‌های
//     لازم برای تطبیق دامنه (واحد، نوع استخدام، گروه شغلی، گروه نظام پرداخت).
//   - پیش‌نمایش تخصیص: برای یک کارمند و یک ماه، سطرهای تخصیص‌یافته با کد، مبلغ و مشمولیت
//     مالیات/بیمه — همان چیزی که فرم پیش از ثبت نشان می‌دهد.
//   - اتصال به فیش: پاسخ مسیر یکپارچهٔ فیش، بلوک `assignments` را هم می‌آورد تا جمع
//     تخصیص‌ها (و مشمولیتشان) کنار فیش دیده شود.
//
// قاعده‌ها همه در بستهٔ `rules` است؛ اینجا فقط خواندن رکوردها، ساختن تصویر کارمند و
// برگرداندن پاسخ است.
package payroll

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/payroll/domain"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// allowancePreviewRequest ورودی پیش‌نمایش تخصیص.
type allowancePreviewRequest struct {
	EmployeeID string             `json:"employeeId"`
	Period     string             `json:"period,omitempty"`
	Bases      map[string]float64 `json:"bases,omitempty"`
}

// registerAllowanceEndpoints مسیرهای تخصیص فوق العاده را ثبت می‌کند.
func registerAllowanceEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }

	// جدول رسمی کدها و فهرست‌های مرجع فرم تخصیص.
	mux.HandleFunc("GET /api/v1/payroll/allowance-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		params := oilPayParams(deps, nil)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"codes":       rules.PayslipCodes(),
			"calcKinds":   rules.AllowanceCalcKinds(),
			"bases":       rules.AllowanceBasisKeys(),
			"targetKinds": rules.AllowanceTargetKinds(),
			"employees":   allowanceEmployees(r.Context(), deps, params),
			"period":      oilPeriod(deps),
			"today":       payTodayFa(deps),
			"gaps":        rules.PayslipCodeGaps(),
			"noteFa": "کد رسمی را انتخاب کنید، روش محاسبه و مشمولیت کسور را تعیین کنید و " +
				"دامنه را روی یک کارمند یا گروهی از کارکنان بگذارید. قاعده‌های محاسبه و " +
				"تطبیق دامنه در سرور اجرا می‌شود.",
		})
	}))

	// پیش‌نمایش: چه اقلامی به این کارمند می‌خورد و چقدر می‌شود.
	mux.HandleFunc("POST /api/v1/payroll/allowance-preview", read(func(w http.ResponseWriter, r *http.Request) {
		var in allowancePreviewRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ورودی پیش‌نمایش خوانده نشد.", nil)
			return
		}
		view, ok := allowanceEmployeeView(r.Context(), deps, oilPayParams(deps, nil), strings.TrimSpace(in.EmployeeID))
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "کارمند انتخابی یافت نشد.", nil)
			return
		}
		period := strings.TrimSpace(in.Period)
		if period == "" {
			period = oilPeriod(deps)
		}
		assignments := activeAllowanceAssignments(r.Context(), svc)
		bases := allowanceBases(view, deps, in.Bases)
		lines := rules.ResolveAllowanceAssignments(assignments, view, rules.AllowanceEnv{Bases: bases, Period: period})
		totals := rules.SumAllowanceLines(lines)

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"employee":    view,
			"period":      period,
			"lines":       lines,
			"totals":      totals,
			"bases":       bases,
			"assignments": len(assignments),
			"noteFa": "سطرها از تخصیص‌های فعال همین کارمند ساخته شده‌اند؛ «مشمول مالیات» و " +
				"«مشمول بیمه» روی هر سطر نشان داده می‌شود.",
		})
	}))
}

// allowanceEmployees فهرست کارکنان با ویژگی‌های لازم تطبیق دامنه.
func allowanceEmployees(ctx context.Context, deps modulekit.Deps, params rules.OilPayParams) []map[string]string {
	out := []map[string]string{}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return out
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Size: 500})
	if err != nil {
		return out
	}
	for _, e := range page.Items {
		group := rules.OilPayGroupForJobGradeNumber(params, e.EmploymentType, e.JobGrade)
		out = append(out, map[string]string{
			"id":               e.ID,
			"titleFa":          e.FullName + " — " + e.ID,
			"unit":             e.DepartmentName,
			"employmentType":   e.EmploymentType,
			"employmentTypeFa": e.EmploymentTypeFa,
			"jobGrade":         e.JobGrade,
			"jobGradeFa":       e.JobGradeFa,
			"payGroup":         intPad(int64(group)),
			"payGroupFa":       rules.OilPayGroupTitleFa(group),
			"baseSalary":       intPad(e.BaseSalary),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["id"] < out[j]["id"] })
	return out
}

// allowanceEmployeeView تصویر کارمند برای تطبیق دامنه.
func allowanceEmployeeView(ctx context.Context, deps modulekit.Deps, params rules.OilPayParams, employeeID string) (rules.AllowanceEmployee, bool) {
	reader, ok := deps.Ports.Employees()
	if !ok || employeeID == "" {
		return rules.AllowanceEmployee{}, false
	}
	employee, found, err := reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
	if err != nil || !found {
		return rules.AllowanceEmployee{}, false
	}
	return rules.AllowanceEmployee{
		ID:             employee.ID,
		DepartmentName: employee.DepartmentName,
		EmploymentType: employee.EmploymentType,
		JobGrade:       employee.JobGrade,
		PayGroup:       rules.OilPayGroupForJobGradeNumber(params, employee.EmploymentType, employee.JobGrade),
	}, true
}

// allowanceBases مبناهای محاسبهٔ اقلام درصدی/ضریبی برای همین کارمند.
//
// مبناها با پارامترهای نظام صندوق نفت ساخته می‌شوند (پایه + بدی آب و هوا = مبنای
// فوق‌العاده‌ها و …) تا نتیجه با فیش همان نظام یکسان باشد؛ ورودی درخواست می‌تواند هر مبنا
// را برای پیش‌نمایش بازنویسی کند.
func allowanceBases(view rules.AllowanceEmployee, deps modulekit.Deps, override map[string]float64) map[string]float64 {
	params := oilPayParams(deps, nil)
	base := allowanceEmployeeBaseSalary(deps, view.ID)
	climate := base * params.ClimatePercent / 100
	allowanceBase := base + climate
	specialPercent := 0.0
	for _, a := range rules.OilAllowanceCatalog(params.Allowances) {
		if a.Key == rules.OilAllowanceSpecial {
			specialPercent = a.Percent
		}
	}
	attractionBase := allowanceBase + allowanceBase*specialPercent/100

	bases := map[string]float64{
		"base":           base,
		"allowanceBase":  allowanceBase,
		"attractionBase": attractionBase,
		"gross":          attractionBase,
		"insuranceBase":  attractionBase,
		"constant":       0,
	}
	for key, value := range override {
		bases[key] = value
	}
	return bases
}

// allowanceEmployeeBaseSalary پایهٔ حقوق کارمند از پرونده.
func allowanceEmployeeBaseSalary(deps modulekit.Deps, employeeID string) float64 {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return 0
	}
	employee, found, err := reader.Get(context.Background(), contracts.EmployeeFilter{Scope: "all"}, employeeID)
	if err != nil || !found {
		return 0
	}
	return float64(employee.BaseSalary)
}

// activeAllowanceAssignments تخصیص‌های فعال را از انبار رکوردها می‌خواند و به ساختار قواعد
// نگاشت می‌کند. تخصیص پایان‌یافته یا پیش‌نویس در محاسبه نمی‌آید.
func activeAllowanceAssignments(ctx context.Context, svc *recordkit.Service) []rules.AllowanceAssignment {
	if svc == nil {
		return nil
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "payroll", Kind: domain.KindAllowanceAssignment, Scope: "all", Size: 500,
	})
	if err != nil {
		return nil
	}
	out := make([]rules.AllowanceAssignment, 0, len(page.Items))
	for _, rec := range page.Items {
		if rec.Status != "active" {
			continue
		}
		f := rec.Fields
		out = append(out, rules.AllowanceAssignment{
			ID:            rec.ID,
			Code:          strings.TrimSpace(f["code"]),
			TitleFa:       strings.TrimSpace(f["titleFa"]),
			Category:      strings.TrimSpace(f["category"]),
			Calc:          strings.TrimSpace(f["calc"]),
			Rate:          strings.TrimSpace(f["rate"]),
			Basis:         strings.TrimSpace(f["basis"]),
			Formula:       strings.TrimSpace(f["formula"]),
			Taxable:       boolField(f["taxable"]),
			Insured:       boolField(f["insured"]),
			ShowOnPayslip: boolFieldOrEmpty(f["showOnPayslip"]),
			ShowInReports: boolFieldOrEmpty(f["showInReports"]),
			ReportCode:    strings.TrimSpace(f["reportCode"]),
			Target:        rules.AllowanceTarget{Kind: strings.TrimSpace(f["targetKind"]), Value: strings.TrimSpace(f["targetValue"])},
			EffectiveFrom: strings.TrimSpace(f["effectiveFrom"]),
			EffectiveTo:   strings.TrimSpace(f["effectiveTo"]),
			NoteFa:        strings.TrimSpace(f["noteFa"]),
			Status:        rec.Status,
		})
	}
	return out
}

// allowanceAssignmentBlock بلوک تخصیص‌ها برای پاسخ مسیر یکپارچهٔ فیش.
//
// `lines` همهٔ سطرهای تخصیص‌یافته است و `payslipLines` فقط آنها که در فرم «نمایش در فیش
// حقوقی» دارند؛ تصمیم نمایش در خود سند تخصیص ثبت شده و اینجا فقط اجرا می‌شود.
func allowanceAssignmentBlock(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, employeeID, period string) map[string]any {
	view, ok := allowanceEmployeeView(ctx, deps, oilPayParams(deps, nil), employeeID)
	if !ok {
		return map[string]any{"lines": []rules.AllowanceLine{}, "payslipLines": []rules.AllowanceLine{},
			"reportLines": []rules.AllowanceLine{}, "totals": rules.AllowanceTotals{}}
	}
	bases := allowanceBases(view, deps, nil)
	lines := rules.ResolveAllowanceAssignments(activeAllowanceAssignments(ctx, svc), view,
		rules.AllowanceEnv{Bases: bases, Period: period})
	return map[string]any{
		"lines":        lines,
		"payslipLines": rules.AllowancePayslipLines(lines),
		"reportLines":  rules.AllowanceReportLines(lines),
		"totals":       rules.SumAllowanceLines(lines),
		"bases":        bases,
		"noteFa": "هر قلم تخصیص می‌گوید در فیش حقوقی دیده شود یا نه و در گزارش‌های چاپی " +
			"درج شود یا نه؛ «کد گزارش» همان کد رسمی گزارش (کدال) است.",
	}
}

// boolFieldOrEmpty خواندن پرچم بله/خیر با پیش‌فرض «بله» برای مقدار خالی.
func boolFieldOrEmpty(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	return boolField(raw)
}

// boolField فیلد بله/خیر رکورد را می‌خواند (بله/yes/true/1).
func boolField(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "بله", "yes", "true", "1", "y":
		return true
	default:
		return false
	}
}
