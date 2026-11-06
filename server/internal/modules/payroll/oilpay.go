// پیاده‌سازی ماژول «حقوق و دستمزد» — نظام پرداخت صندوق بازنشستگی نفت.
//
// چرا این فایل جدا از `handlers.go`: سازمان دو نظام پرداخت دارد و این یکی ساختار خودش را
// دارد (پایهٔ حقوق، فوق‌العادهٔ مناطق گرمسیری ۲۵٪، و بقیهٔ فوق‌العاده‌ها درصدی از «مبنای
// فوق‌العاده‌ها»). قاعده در `rules.ComputeOilPayslip` است و این فایل فقط پارامترهای شرکت
// را می‌خواند، کارمند را از پورت ماژول کارکنان می‌گیرد و فیش را برمی‌گرداند؛ پس فرم وب،
// رکورد فیش و هر گزارش دیگر یک عدد می‌دهند.
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

// registerOilPayEndpoints مسیرهای نظام پرداخت صندوق نفت را ثبت می‌کند.
func registerOilPayEndpoints(mux *http.ServeMux, deps modulekit.Deps, _ *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }

	// پارامترها، فهرست فوق‌العاده‌ها و نوبت‌کاری‌ها، کارکنان و دورهٔ جاری.
	mux.HandleFunc("GET /api/v1/payroll/oil-payslip-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, oilPayDefaults(r.Context(), deps))
	}))

	// محاسبهٔ فیش: ورودی همان فیلدهای فرم، خروجی سطرهای فیش و جمع‌ها.
	mux.HandleFunc("POST /api/v1/payroll/oil-payslip", read(func(w http.ResponseWriter, r *http.Request) {
		var in rules.OilPayInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی محاسبهٔ فیش خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, oilPayslip(deps, in))
	}))
}

// oilPayDefaults پارامترها و فهرست‌های مرجع نظام صندوق نفت.
func oilPayDefaults(ctx context.Context, deps modulekit.Deps) map[string]any {
	params := oilPayParams(deps)
	return map[string]any{
		"params":         params,
		"allowances":     rules.OilAllowances(params),
		"shiftOptions":   []map[string]string{{"key": "no", "titleFa": "غیر نوبت‌کار"}, {"key": "yes", "titleFa": "نوبت‌کار"}},
		"employees":      oilEmployeeOptions(ctx, deps),
		"paramViews":     oilParamViews(deps),
		"period":         oilPeriod(deps),
		"today":          oilToday(deps),
		"daysInMonth":    oilDaysInMonth(deps),
		"schemeFa":       "نظام پرداخت صندوق بازنشستگی نفت",
		"allowanceBases": []map[string]string{{"key": "allowanceBase", "titleFa": "مبنای فوق‌العاده‌ها (پایه + گرمسیری)"}, {"key": "baseSalary", "titleFa": "پایهٔ حقوق"}},
		"noteFa":         oilPayNoteFa(),
	}
}

// oilPayslip فیش را با پارامترهای شرکت می‌سازد و دورهٔ جاری را هم برمی‌گرداند.
func oilPayslip(deps modulekit.Deps, in rules.OilPayInput) map[string]any {
	slip := rules.ComputeOilPayslip(in, oilPayParams(deps))
	return map[string]any{
		"scheme":      rules.SchemeOil,
		"schemeFa":    "نظام پرداخت صندوق بازنشستگی نفت",
		"slip":        slip,
		"input":       in,
		"params":      oilPayParams(deps),
		"noteFa":      oilPayNoteFa(),
		"period":      oilPeriod(deps),
		"daysInMonth": oilDaysInMonth(deps),
	}
}

// oilPayParams آستانه‌ها و درصدهای نظام صندوق را از پروفایل شرکت فعال می‌خواند.
//
// هر کلید `oilpay.*` که در پارامترهای شرکت باشد همین‌جا خوانده می‌شود؛ پس تغییر مصوبه
// نیاز به انتشار نسخهٔ تازه ندارد.
func oilPayParams(deps modulekit.Deps) rules.OilPayParams {
	params := rules.DefaultOilPayParams()
	if v, ok := oilParam(deps, "oilpay.climatePercent"); ok && v > 0 {
		params.ClimatePercent = v
	}
	// فوق‌العاده‌های درصدی: درصد هر قلم از پارامتر همان قلم می‌آید
	// (مثلاً oilpay.allowance.deprivation = 27.5).
	for i := range params.Allowances {
		if v, ok := oilParam(deps, "oilpay.allowance."+params.Allowances[i].Key); ok && v >= 0 {
			params.Allowances[i].Percent = v
		}
	}
	if v, ok := oilParam(deps, "oilpay.fund.employeePercent"); ok && v >= 0 {
		params.FundEmployeePercent = v
	}
	if v, ok := oilParam(deps, "oilpay.fund.employerPercent"); ok && v >= 0 {
		params.FundEmployerPercent = v
	}
	if v, ok := oilParam(deps, "oilpay.insurance.employeePercent"); ok && v >= 0 {
		params.InsuranceEmployeePercent = v
	}
	if v, ok := oilParam(deps, "oilpay.tax.exemption"); ok && v > 0 {
		params.TaxExemption = int64(v)
	}
	if v, ok := oilParam(deps, "oilpay.overtimeFactor"); ok && v > 0 {
		params.OvertimeFactor = v
	}
	if v, ok := oilParam(deps, "oilpay.monthlyHours"); ok && v > 0 {
		params.MonthlyHours = v
	}
	return params
}

// oilParam مقدار عددی یک پارامتر استقرار.
func oilParam(deps modulekit.Deps, key string) (float64, bool) {
	for _, p := range deps.Org.Params {
		if p.Key != key {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64)
		if err != nil {
			return 0, false
		}
		return value, true
	}
	return 0, false
}

// oilParamViews پارامترهای نظام صندوق را برای نمایش در فرم برمی‌گرداند.
func oilParamViews(deps modulekit.Deps) []map[string]string {
	out := []map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "oilpay.") {
			continue
		}
		out = append(out, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa,
		})
	}
	return out
}

// oilEmployeeOptions کارکنان برای انتخاب در فرم (از پورت ماژول کارکنان).
func oilEmployeeOptions(ctx context.Context, deps modulekit.Deps) []map[string]string {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return []map[string]string{}
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Size: 500})
	if err != nil {
		return []map[string]string{}
	}
	out := make([]map[string]string, 0, len(page.Items))
	for _, e := range page.Items {
		out = append(out, map[string]string{
			"id": e.ID, "titleFa": e.FullName + " — " + e.ID,
			"unit": e.DepartmentName, "position": e.Position,
			"baseSalary":     strconv.FormatInt(e.BaseSalary, 10),
			"employmentType": e.EmploymentType,
		})
	}
	return out
}

// oilPeriod دورهٔ جاری به شکل YYYY/MM (ماه حقوق).
func oilPeriod(deps modulekit.Deps) string {
	now, err := calendar.FromGregorian(deps.Clock())
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d/%02d", now.Year, now.Month)
}

// oilToday تاریخ امروز شمسی.
func oilToday(deps modulekit.Deps) string {
	now, err := calendar.FromGregorian(deps.Clock())
	if err != nil {
		return ""
	}
	return now.String()
}

// oilDaysInMonth شمار روزهای ماه جاری (برای نسبت نوبت‌کاری).
func oilDaysInMonth(deps modulekit.Deps) float64 {
	now, err := calendar.FromGregorian(deps.Clock())
	if err != nil {
		return 30
	}
	length, err := calendar.MonthLength(now.Year, now.Month)
	if err != nil {
		return 30
	}
	return float64(length)
}

// oilPayNoteFa توضیح قاعده‌ها برای فرم؛ متن از سرور می‌آید تا در رابط تکرار نشود.
func oilPayNoteFa() string {
	return "این نظام پرداخت برای کارکنان تابع صندوق بازنشستگی نفت است: کارمند یک پایهٔ حقوق " +
		"دارد، فوق‌العادهٔ بدی آب و هوا (مناطق گرمسیری) درصدی از همان پایه است " +
		"(پارامتر oilpay.climatePercent) و جمع این دو «مبنای فوق‌العاده‌ها» می‌سازد. " +
		"همهٔ فوق‌العاده‌های دیگر — محرومیت، ویژه، صرف وقت، کارگاهی، جذب و تخصصی شغل، " +
		"فرآورشی و نوبت‌کار — درصدی از همین مبنا محاسبه می‌شوند؛ پس تغییر پایه یا درصد بدی " +
		"آب و هوا، همهٔ اقلام را با هم به‌روز می‌کند.\n" +
		"قلم نوبت‌کار تنها برای کارمند نوبت‌کار پرداخت می‌شود و به نسبت روزهای نوبت‌کاری به " +
		"روزهای ماه محاسبه می‌گردد. اضافه‌کاری از نرخ ساعتی مبنا با ضریب پارامتریک " +
		"(oilpay.overtimeFactor) می‌آید. کسور شامل سهم کارمند صندوق " +
		"(oilpay.fund.employeePercent)، بیمهٔ سهم کارمند (اگر پارامتر داشته باشد) و مالیات پله‌ای است.\n" +
		"درصد هر قلم مصوب در پارامترهای شرکت با کلید oilpay.allowance.<قلم> قابل تغییر است؛ " +
		"اقلام تازه در فهرست موتور قواعد اعلام می‌شوند و فیش و فرم بدون تغییر کد آن‌ها را نشان می‌دهند."
}
