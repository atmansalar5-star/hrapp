// پیاده‌سازی ماژول «statistics» — «analytics».

package statistics

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/rules"
)

// registerAnalyticsEndpoints مسیرهای تجمیعی آمار را ثبت می‌کند: اقلام حقوق (اطلاعات پایه
// حقوق و مزایا و کارانه)، حضور/مرخصی/مأموریت/اضافه‌کاری و شاخص‌های منابع انسانی.
//
// قاعدهٔ معماری: این مسیرها هیچ قاعدهٔ تازه‌ای نمی‌سازند؛ اقلام حقوق را از موتور مشترک
// (`rules.ComputeElements`) می‌گیرند و رکوردهای حضور را از پورت قراردادی همان ماژول
// می‌خوانند. هیچ import مستقیمی بین ماژول‌ها وجود ندارد.
func registerAnalyticsEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("statistics", "read", fn) }

	mux.HandleFunc("GET /api/v1/statistics/pay-elements", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, payElementStats(r.Context(), deps))
	}))
	mux.HandleFunc("GET /api/v1/statistics/attendance", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, attendanceStats(r.Context(), deps))
	}))
	mux.HandleFunc("GET /api/v1/statistics/headcount", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, headcountStats(r.Context(), deps))
	}))
}

// elementStat یک ردیف آمار اقلام حقوق.
type elementStat struct {
	Code     string  `json:"code"`
	TitleFa  string  `json:"titleFa"`
	Category string  `json:"category"`
	Calc     string  `json:"calc"`
	Basis    string  `json:"basis"`
	BasisFa  string  `json:"basisFa"`
	Formula  string  `json:"formula"`
	Rate     string  `json:"rate"`
	Amount   float64 `json:"amount"`
	Resolved bool    `json:"resolved"`
	DetailFa string  `json:"detailFa"`
}

// payElementStats آمار «اطلاعات پایه حقوق و مزایا» و «کارانه» را می‌سازد.
func payElementStats(ctx context.Context, deps modulekit.Deps) map[string]any {
	provider, ok := deps.Ports.PayElements()
	if !ok {
		return map[string]any{"elements": []elementStat{}, "noteFa": "پورت اقلام حقوق در این استقرار در دسترس نیست"}
	}
	elements := provider.PayElements(ctx)
	params := rules.DefaultPayrollParams()
	input := rules.PayrollInput{BaseSalary: baseSalaryOf(deps)}
	result := rules.ComputePayroll(input, params)

	vars := rules.BaseElementVars(input, result)
	for k, v := range rules.ParamsVars(params) {
		vars[k] = v
	}
	for k, v := range provider.PayParamValues(ctx) {
		if f, ok := parseNum(v); ok {
			vars[k] = f
		}
	}
	// اجزای مزد از همان سازندهٔ مشترک می‌آید: مزد شغل همان پایهٔ سازمان و بقیهٔ اجزا
	// (رتبه، سنوات، پست، ایثارگری، ثابت‌های شغلی و اقلیمی) از پارامترهای فعال.
	wageVars := map[string]float64{"wage.job": float64(input.BaseSalary)}
	for k, v := range vars {
		if _, ok := wageVars[k]; !ok {
			wageVars[k] = v
		}
	}
	wages := rules.WageComponentsFromVars(wageVars)
	for k, v := range wages.Vars() {
		vars[k] = v
	}
	env := rules.ElementEnv{
		Bases:     rules.BaseElementBases(input, params, result, 0, wages),
		Vars:      vars,
		Canonical: rules.CanonicalElementAmounts(result),
	}

	defs := map[string]rules.PayElement{}
	for _, el := range elements {
		defs[el.Code] = el
	}

	rows := []elementStat{}
	var earn, ded float64
	for _, line := range rules.ComputeElements(elements, env) {
		row := elementStat{
			Code: line.Code, TitleFa: line.TitleFa, Category: line.Category, Calc: line.Calc,
			Amount: line.Amount, Resolved: line.Resolved, DetailFa: line.DetailFa,
		}
		if el, ok := defs[line.Code]; ok {
			row.Basis = el.Basis
			row.BasisFa = rules.BasisFa(el.Basis)
			row.Formula = el.Formula
			row.Rate = el.Rate
		}
		if line.Category == "earning" {
			earn += line.Amount
		} else {
			ded += line.Amount
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Amount > rows[j].Amount })

	// تفکیک گروهی: درصدی / ثابت / فرمولی — برای نمودار و برش آماری.
	byCalc := map[string]int{}
	byBasis := map[string]float64{}
	for _, row := range rows {
		byCalc[row.Calc]++
		byBasis[row.Basis] += row.Amount
	}

	karana := []elementStat{}
	for _, e := range rows {
		if strings.Contains(e.Code, "KARANA") || strings.Contains(e.TitleFa, "کارانه") || e.Basis == "karana" {
			karana = append(karana, e)
		}
	}

	return map[string]any{
		"elements": rows,
		"karana":   karana,
		"byCalc":   byCalc,
		"byBasis":  byBasis,
		"summary": map[string]any{
			"count":          len(rows),
			"earningTotal":   earn,
			"deductionTotal": ded,
			"mazdBase":       wages.Total(),
			"wageJob":        wages.Job,
			"wageRank":       wages.Rank,
			"wageSeniority":  wages.Seniority,
			"postAllowance":  wages.PostAllowance,
			"sacrifice":      wages.Sacrifice,
			// اجزای ثابت شغلی و مبالغ مصوبی که مزد مبنا را کامل می‌کنند؛ بدون اینها جمع
			// سطرهای کارت با «مزد مبنا» نمی‌خواند.
			"jobAttachments":   wages.JobAndAttachments(),
			"special":          wages.Special,
			"workshop":         wages.Workshop,
			"attractionPublic": wages.AttractionPublic,
			"personalBase":     wages.PersonalBase,
			"jobSpecialty":     wages.JobSpecialty,
			"salaryRepair":     wages.SalaryRepair,
			"climate":          wages.Climate,
			"breakdown":        wages.Breakdown(),
			"gross":            result.Gross,
			"net":              result.Net,
		},
		"noteFa": "مبالغ با موتور مشترک قواعد و برای کارمند نمونه محاسبه شده است؛ مبلغ قطعی هر کارمند در فیش او مرجع است.",
	}
}

// attendanceSummary جمع تردد/مرخصی/مأموریت/اضافه‌کاری.
type attendanceSummary struct {
	Records       int              `json:"records"`
	ByStatus      map[string]int   `json:"byStatus"`
	LeaveDays     float64          `json:"leaveDays"`
	LeaveTypes    map[string]int   `json:"leaveTypes"`
	MissionDays   float64          `json:"missionDays"`
	OvertimeHours float64          `json:"overtimeHours"`
	WorkedHours   float64          `json:"workedHours"`
	Pending       int              `json:"pending"`
	Approved      int              `json:"approved"`
	Rejected      int              `json:"rejected"`
	Rows          []map[string]any `json:"rows"`
	// DeptRows برش کارکرد به تفکیک واحد سازمانی (از پورت کارکنان)؛ برای تب «حضور و
	// غیاب و تردد» که پیش از این فقط جمع کل را نشان می‌داد.
	DeptRows []statRow `json:"deptRows"`
}

// attendanceStats رکوردهای حضور را از پورت قراردادی همان ماژول جمع می‌زند.
func attendanceStats(ctx context.Context, deps modulekit.Deps) attendanceSummary {
	out := attendanceSummary{ByStatus: map[string]int{}, LeaveTypes: map[string]int{}, Rows: []map[string]any{}, DeptRows: []statRow{}}
	index := employeeIndex(ctx, deps)
	byDept := map[string]*statRow{}
	for _, spec := range []struct {
		kind  string
		label string
	}{
		{"attendanceRecords", "تردد روزانه"},
		{"leaveRequests", "درخواست مرخصی"},
		{"missions", "مأموریت"},
		{"overtime", "اضافه‌کاری"},
	} {
		records := fetchRecords(ctx, deps, spec.kind)
		row := map[string]any{"kind": spec.kind, "titleFa": spec.label, "count": len(records)}
		for _, rec := range records {
			out.Records++
			out.ByStatus[rec.Status]++
			// دستهٔ تأیید با همان نگاشت مشترک تب‌های آمار شمرده می‌شود تا عددهای
			// «در انتظار/تأییدشده/ردشده» در همهٔ صفحه‌ها یک زبان داشته باشند.
			switch statusBucket(rec.Status) {
			case "approved":
				out.Approved++
			case "rejected":
				out.Rejected++
			default:
				out.Pending++
			}
			switch spec.kind {
			case "attendanceRecords":
				worked := num(rec.Fields["workedHours"])
				out.WorkedHours += worked
				addDeptWork(byDept, index, rec.EmployeeID, worked)
			case "leaveRequests":
				out.LeaveDays += num(rec.Fields["days"])
				if t := strings.TrimSpace(rec.Fields["leaveType"]); t != "" {
					out.LeaveTypes[t]++
				}
			case "missions":
				out.MissionDays += num(rec.Fields["days"])
			case "overtime":
				out.OvertimeHours += num(rec.Fields["hours"])
			}
		}
		out.Rows = append(out.Rows, row)
	}
	out.DeptRows = sortedRows(byDept, func(r statRow) float64 { return r.Hours })
	var deptHours float64
	for _, r := range out.DeptRows {
		deptHours += r.Hours
	}
	for i := range out.DeptRows {
		out.DeptRows[i].SharePct = sharePct(out.DeptRows[i].Hours, deptHours)
	}
	return out
}

// addDeptWork یک رکورد تردد را به ردیف واحد سازمانی کارمندش می‌افزاید تا برش «کارکرد به
// تفکیک واحد» ساخته شود.
func addDeptWork(byDept map[string]*statRow, index map[string]contracts.Employee, employeeID string, worked float64) {
	dept := "نامشخص"
	if e, ok := index[employeeID]; ok && strings.TrimSpace(e.DepartmentName) != "" {
		dept = e.DepartmentName
	}
	row, ok := byDept[dept]
	if !ok {
		row = &statRow{Key: dept, TitleFa: dept}
		byDept[dept] = row
	}
	row.Count++
	row.Hours += worked
}

// fetchRecords رکوردهای یک نوع را از پورت فقط‌خواندنی ماژول مالک می‌خواند. مرز ماژول
// حفظ می‌شود: مالک داده و دامنهٔ دسترسی همان‌جا می‌ماند و اینجا فقط جمع زده می‌شود.
func fetchRecords(ctx context.Context, deps modulekit.Deps, kind string) []contracts.Record {
	lister, ok := deps.Ports.RecordList("attendance")
	if !ok {
		return nil
	}
	page, err := lister.List(ctx, contracts.RecordFilter{Kind: kind, Size: 500})
	if err != nil {
		return nil
	}
	return page.Items
}

// headcountStats آمار جمعیت کارکنان به تفکیک اداره، وضعیت و نوع قرارداد.
func headcountStats(ctx context.Context, deps modulekit.Deps) map[string]any {
	out := map[string]any{"total": 0, "byDepartment": map[string]int{}, "byStatus": map[string]int{}, "byContract": map[string]int{}}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return out
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 500})
	if err != nil {
		return out
	}
	byDept := map[string]int{}
	byStatus := map[string]int{}
	byContract := map[string]int{}
	var payroll int64
	for _, e := range page.Items {
		byDept[e.DepartmentName]++
		byStatus[e.Status]++
		byContract[e.ContractType]++
		payroll += e.BaseSalary
	}
	out["total"] = page.Total
	out["byDepartment"] = byDept
	out["byStatus"] = byStatus
	out["byContract"] = byContract
	out["payrollTotal"] = payroll
	if len(page.Items) > 0 {
		out["avgSalary"] = payroll / int64(len(page.Items))
	}
	return out
}

// baseSalaryOf مزد شغل پایهٔ آماری: بزرگ‌ترین مزد پایهٔ کارکنان (نمایندهٔ سازمان).
func baseSalaryOf(deps modulekit.Deps) int64 {
	if p, ok := deps.Ports.Employees(); ok {
		if page, err := p.List(context.Background(), contracts.EmployeeFilter{Scope: "all", Size: 500}); err == nil {
			var max int64
			for _, e := range page.Items {
				if e.BaseSalary > max {
					max = e.BaseSalary
				}
			}
			if max > 0 {
				return max
			}
		}
	}
	return 0
}

func num(raw string) float64 {
	if f, ok := parseNum(raw); ok {
		return f
	}
	return 0
}

func parseNum(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}
