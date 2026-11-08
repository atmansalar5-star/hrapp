// پیاده‌سازی ماژول «حقوق و دستمزد» — تطبیق حقوق و «تفاوت تطبیق».
//
// چرا این فایل: هنگام اجرای طرح تازهٔ حقوق و مزایا (مثلاً طرح طبقه‌بندی مشاغل) ممکن است
// حقوق کارمند کمتر شود؛ مابه‌التفاوت به‌عنوان «تفاوت تطبیق» ثبت می‌شود و با افزایش‌های
// بعدی جذب می‌گردد. قاعده در `rules.ComputePayGap` و `rules.AbsorbPayGap` است و این فایل
// فقط پارامترهای شرکت را می‌خواند، سند کارگزینی کارمند را از پورت ماژول احکام می‌گیرد و
// فرم محاسباتی را برمی‌گرداند؛ پس فرم وب، حکم و فیش یک عدد می‌بینند.
package payroll

import (
	"context"
	"encoding/json"
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

// registerPayGapEndpoints مسیرهای تطبیق حقوق و تفاوت تطبیق را ثبت می‌کند.
func registerPayGapEndpoints(mux *http.ServeMux, deps modulekit.Deps, _ *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }

	// پارامترها، دستورالعمل، کارکنان و سند کارگزینی جاری برای پیش‌پرکردن.
	mux.HandleFunc("GET /api/v1/payroll/paygap-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, payGapDefaults(r.Context(), deps))
	}))

	// سند کارگزینی یک کارمند (اجزای مزد، مزد مبنا و تفاوت تطبیق جاری) برای پیش‌پرکردن فرم.
	mux.HandleFunc("GET /api/v1/payroll/paygap-employee", read(func(w http.ResponseWriter, r *http.Request) {
		out, ok := payGapEmployee(r.Context(), deps, r.URL.Query().Get("id"))
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound,
				"سند کارگزینی ابلاغ‌شده‌ای برای این کارمند یافت نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))

	// فرم محاسباتی تطبیق: جمع پیشین در برابر جمع طرح تازه → تفاوت تطبیق و سطرهای محاسبه.
	mux.HandleFunc("POST /api/v1/payroll/paygap-plan", read(func(w http.ResponseWriter, r *http.Request) {
		var in payGapRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی فرم تطبیق خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, payGapPlan(deps, in))
	}))

	// جذب: اثر زنجیرهٔ افزایش‌ها روی تفاوت تطبیق (نمودار جذب و باقی‌مانده).
	mux.HandleFunc("POST /api/v1/payroll/paygap-absorb", read(func(w http.ResponseWriter, r *http.Request) {
		var in payGapAbsorbRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی محاسبهٔ جذب خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, payGapAbsorb(deps, in))
	}))
}

// payGapRequest ورودی فرم تطبیق یک کارمند.
type payGapRequest struct {
	EmployeeID    string `json:"employeeId"`
	EmployeeName  string `json:"employeeName"`
	PreviousTotal int64  `json:"previousTotal"`
	NewTotal      int64  `json:"newTotal"`
}

// payGapAbsorbRequest ورودی محاسبهٔ جذب.
type payGapAbsorbRequest struct {
	PayGap    int64   `json:"payGap"`
	Increases []int64 `json:"increases"`
}

// payGapDefaults پارامترها، دستورالعمل و کارکنان فرم تطبیق.
func payGapDefaults(ctx context.Context, deps modulekit.Deps) map[string]any {
	params := payGapParams(deps)
	return map[string]any{
		"params":      params,
		"paramViews":  payGapParamViews(deps),
		"instruction": rules.PayGapInstructionFa(),
		"employees":   payGapEmployees(ctx, deps),
		"today":       payGapToday(deps),
		"schemeFa":    "تطبیق حقوق و مزایا و تفاوت تطبیق",
		"noteFa": "تفاوت تطبیق مبلغی ثابت و جذب‌شدنی است: جزو مزد مبنا نیست، پس مبنای " +
			"اضافه‌کاری را بالا نمی‌برد، ولی بخشی از حقوق و مزایای مشمول بیمه و مالیات است. " +
			"مبلغ محاسبه‌شده را در حکم کارگزینی ثبت کنید تا در فیش و فرمت حقوق بیاید.",
	}
}

// payGapPlan فرم محاسباتی تطبیق یک کارمند.
func payGapPlan(deps modulekit.Deps, in payGapRequest) map[string]any {
	params := payGapParams(deps)
	result := rules.ComputePayGap(rules.PayGapInput{
		EmployeeID:    strings.TrimSpace(in.EmployeeID),
		EmployeeName:  strings.TrimSpace(in.EmployeeName),
		PreviousTotal: in.PreviousTotal,
		NewTotal:      in.NewTotal,
	}, params)
	out := map[string]any{
		"result":      result,
		"params":      params,
		"today":       payGapToday(deps),
		"schemeFa":    "تطبیق حقوق و مزایا",
		"payGapBasis": rules.IntMoneyFa(params.RoundTo),
		"noteFa": "این مبلغ مبنای ثبت در حکم کارگزینی است؛ پس از ثبت، در فیش حقوقی و " +
			"«فرمت حقوق» به‌صورت سطر جداگانه می‌آید و با افزایش‌های بعدی جذب می‌شود.",
	}
	// مستندسازی: اگر کارمندی انتخاب شده باشد، سند کارگزینی جاری او هم در پاسخ می‌آید تا
	// کاربر ببیند تفاوت تطبیق کنونی چه قدر است (برای پیش‌گیری از دوباره‌شماری).
	if strings.TrimSpace(in.EmployeeID) != "" {
		if doc, ok := payGapEmployee(context.Background(), deps, in.EmployeeID); ok {
			out["document"] = doc
		}
	}
	return out
}

// payGapAbsorb اثر افزایش‌های پیشنهادی روی تفاوت تطبیق.
func payGapAbsorb(deps modulekit.Deps, in payGapAbsorbRequest) map[string]any {
	params := payGapParams(deps)
	remaining, steps := rules.AbsorbPayGapSeries(in.PayGap, in.Increases, params)
	summary := "با افزایش‌های ثبت‌شده، تفاوت تطبیق به‌طور کامل جذب می‌شود."
	if remaining > 0 {
		summary = "پس از افزایش‌های ثبت‌شده، " + rules.IntMoneyFa(remaining) + " ریال تفاوت تطبیق باقی می‌ماند."
	}
	rows := make([]map[string]string, 0, len(steps))
	for i, step := range steps {
		rows = append(rows, map[string]string{
			"order":       rules.PersianDigitsInt(i + 1),
			"increaseFa":  rules.IntMoneyFa(step.Increase),
			"reductionFa": rules.IntMoneyFa(step.Reduction),
			"remainingFa": rules.IntMoneyFa(step.Remaining),
			"detailFa":    rules.PersianDigits(step.DetailFa),
		})
	}
	return map[string]any{
		"payGap":      in.PayGap,
		"payGapFa":    rules.IntMoneyFa(in.PayGap),
		"remaining":   remaining,
		"remainingFa": rules.IntMoneyFa(remaining),
		"steps":       steps,
		"rows":        rows,
		"params":      params,
		"summaryFa":   rules.PersianDigits(summary),
		"noteFa": "هر افزایش، به نسبت پارامتر شرکت از تفاوت تطبیق می‌کاهد و هرگز بیش از " +
			"باقی‌ماندهٔ آن جذب نمی‌شود؛ پس تفاوت تطبیق منفی نمی‌شود.",
	}
}

// payGapEmployee سند کارگزینی جاری کارمند: پایه، اجزای مزد، مزد مبنا و تفاوت تطبیق.
func payGapEmployee(ctx context.Context, deps modulekit.Deps, employeeID string) (map[string]any, bool) {
	id := strings.TrimSpace(employeeID)
	if id == "" {
		return nil, false
	}
	doc, found := currentWageDoc(ctx, deps, id)
	if !found {
		return nil, false
	}
	base := doc.BaseSalary
	if base <= 0 {
		base = doc.BaseSalary
	}
	components := doc.WageStructure.Components(base, payGapCompanyPercent(deps))
	return map[string]any{
		"employeeId":    id,
		"sourceFa":      doc.SourceFa,
		"baseSalary":    base,
		"mazdBase":      rules.RoundHalfUp(components.Total()),
		"payGap":        doc.PayGap,
		"payGapFa":      rules.IntMoneyFa(doc.PayGap),
		"payGapSince":   doc.PayGapSince,
		"hasPayGap":     doc.HasPayGap(),
		"wageParts":     components.Breakdown(),
		"breakdownFa":   "مزد مبنا از اجزای ثابت سند ساخته می‌شود؛ تفاوت تطبیق جزو آن نیست.",
		"climateRegion": doc.ClimateRegion,
	}, true
}

// currentWage سند کارگزینی جاری کارمند: پایهٔ حقوق + اجزای مزد + منبع سند.
type currentWage struct {
	BaseSalary int64
	contracts.WageStructure
	SourceFa string
}

// currentWageDoc سند کارگزینی کارمند: اولویت با حکم ابلاغ‌شده و در نبود آن قرارداد جاری.
func currentWageDoc(ctx context.Context, deps modulekit.Deps, employeeID string) (currentWage, bool) {
	if reader, ok := deps.Ports.SalaryDecrees(); ok {
		if d, found, err := reader.NotifiedDecreeOf(ctx, employeeID); err == nil && found {
			return currentWage{
				BaseSalary:    d.BaseSalary,
				WageStructure: d.WageStructure,
				SourceFa:      "حکم کارگزینی " + d.ID,
			}, true
		}
	}
	if reader, ok := deps.Ports.ContractDocuments(); ok {
		if c, found, err := reader.ActiveContractOf(ctx, employeeID); err == nil && found {
			return currentWage{
				BaseSalary:    c.MonthlySalary,
				WageStructure: c.WageStructure,
				SourceFa:      "قرارداد کار " + c.ID,
			}, true
		}
	}
	return currentWage{}, false
}

// payGapEmployees کارکنان با پایهٔ حقوق پرونده (برای انتخاب در فرم تطبیق).
func payGapEmployees(ctx context.Context, deps modulekit.Deps) []map[string]any {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return []map[string]any{}
	}
	list, err := reader.List(ctx, contracts.EmployeeFilter{Size: 500})
	if err != nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(list.Items))
	for _, e := range list.Items {
		out = append(out, map[string]any{
			"id": e.ID, "titleFa": e.FullName + " — " + e.ID,
			"position": e.Position, "unit": e.DepartmentName,
			"baseSalary": e.BaseSalary, "jobGradeFa": e.JobGradeFa,
		})
	}
	return out
}

// payGapParams آستانه‌های تفاوت تطبیق را از پارامترهای شرکت فعال می‌خواند.
func payGapParams(deps modulekit.Deps) rules.PayGapParams {
	values := map[string]float64{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "paygap.") {
			continue
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64); err == nil {
			values[p.Key] = v
		}
	}
	return rules.PayGapParamsFromValues(values)
}

// payGapParamViews پارامترهای تفاوت تطبیق را برای نمایش در فرم برمی‌گرداند.
func payGapParamViews(deps modulekit.Deps) []map[string]string {
	out := []map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "paygap.") {
			continue
		}
		out = append(out, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa,
		})
	}
	return out
}

// payGapToday تاریخ امروز شمسی.
func payGapToday(deps modulekit.Deps) string {
	if now, err := calendar.FromGregorian(deps.Clock()); err == nil {
		return now.String()
	}
	return ""
}

// payGapCompanyPercent ضرایب و نرخ‌های محاسبهٔ مزد را از پروفایل شرکت فعال می‌خواند.
//
// چرا همان الگوی اسناد چاپی: مزد مبنایی که این فرم نشان می‌دهد باید همان عددی باشد که حکم
// و فیش حقوق حساب می‌کنند؛ پس ضریب‌ها از یک جا (پارامترهای شرکت) خوانده می‌شوند.
func payGapCompanyPercent(deps modulekit.Deps) map[string]float64 {
	out := map[string]float64{}
	for _, p := range deps.Org.Params {
		if v, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64); err == nil {
			out[p.Key] = v
		}
	}
	return out
}

// payGapPeriodFa برچسب دورهٔ جاری برای سرصفحهٔ فرم.
func payGapPeriodFa(deps modulekit.Deps) string {
	if now, err := calendar.FromGregorian(deps.Clock()); err == nil {
		return fmt.Sprintf("%04d/%02d", now.Year, now.Month)
	}
	return ""
}
