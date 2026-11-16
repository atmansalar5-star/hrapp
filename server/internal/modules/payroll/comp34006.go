// پیاده‌سازی ماژول «حقوق و دستمزد» — نظام جبران خدمات دستورالعمل ۳۴۰۰۶.
//
// چرا این فایل کنار `oilpay.go` است و نه داخل آن: سازمان سه نظام پرداخت مستقل دارد و هر
// کدام ساختار خودش را دارد؛ مدل ۳۴۰۰۶ مبنایش «امتیاز × ضریب ریالی» است و جدول‌های ۱ تا ۵
// خودش را دارد. قاعده در `rules.Comp34006Calc` است و این فایل فقط پارامترهای شرکت را
// می‌خواند، کارمند را از پورت ماژول کارکنان می‌گیرد و بستهٔ پرداخت را برمی‌گرداند؛ پس فرم
// وب، روش پرداخت کارمند و هر گزارش دیگر یک عدد می‌بینند.
package payroll

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// Comp34006SchemeKey کلید رسمی این نظام پرداخت (همان کلید «روش پرداخت کارمند»).
const Comp34006SchemeKey = "comp34006"

// registerComp34006Endpoints مسیرهای نظام جبران خدمات دستورالعمل ۳۴۰۰۶ را ثبت می‌کند.
func registerComp34006Endpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }

	// پارامترها، جدول‌های سند، فهرست‌های مرجع و کارکنان.
	mux.HandleFunc("GET /api/v1/payroll/comp34006-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, comp34006Defaults(r.Context(), deps))
	}))

	// محاسبهٔ بستهٔ پرداخت: ورودی همان فیلدهای فرم، خروجی سطرهای کد ۳۴۶۰۱ تا ۳۴۶۱۸.
	mux.HandleFunc("POST /api/v1/payroll/comp34006-calc", read(func(w http.ResponseWriter, r *http.Request) {
		var in rules.Comp34006Input
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی محاسبهٔ دستورالعمل ۳۴۰۰۶ خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, comp34006Calc(deps, in))
	}))

	// فیش ماهانه: همان محاسبه، به‌همراه کسور قانونی (بیمه و مالیات) و خالص پرداختی.
	mux.HandleFunc("POST /api/v1/payroll/comp34006-payslip", read(func(w http.ResponseWriter, r *http.Request) {
		var in rules.Comp34006PayslipInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی فیش ماهانهٔ دستورالعمل ۳۴۰۰۶ خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, comp34006Payslip(r.Context(), deps, svc, in))
	}))
}

// comp34006Payslip فیش ماهانهٔ یک کارمند را می‌سازد.
//
// کسور قانونی از `collectParams` (همان پارامترهای نظام جاری) خوانده می‌شود تا نرخ بیمه،
// سقف بیمه، معافیت و پله‌های مالیات در هر سه نظام پرداخت یک عدد بدهند. امتیاز ایثارگری
// کارمند هم مثل فیش جاری از پروندهٔ تأییدشده خوانده می‌شود، نه از بدنهٔ درخواست.
func comp34006Payslip(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in rules.Comp34006PayslipInput) map[string]any {
	cp := comp34006Params(deps, nil)
	pp, _, _ := collectParams(ctx, deps, svc)
	if strings.TrimSpace(in.PeriodFa) == "" {
		// دورهٔ خالی یعنی «ماه جاری»؛ از deps.Now() می‌خوانیم که با Clock نبوده هم کار می‌کند.
		if today, err := calendar.FromGregorian(deps.Now()); err == nil {
			in.PeriodFa = fmt.Sprintf("%d/%02d", today.Year, today.Month)
		}
	}
	in.Veteran = veteranEntitlement(ctx, deps, in.EmployeeID)
	slip := rules.Comp34006PayslipFor(in, cp, pp)
	return map[string]any{
		"scheme":         Comp34006SchemeKey,
		"schemeFa":       Comp34006SchemeFa,
		"slip":           slip,
		"params":         cp,
		"statutory":      pp,
		"period":         in.PeriodFa,
		"today":          payTodayFa(deps),
		"independenceFa": rules.Comp34006IndependenceFa(cp.PayrollApply),
		"instructionFa":  rules.Comp34006PayslipInstructionFa(),
		"noteFa":         comp34006NoteFa(),
	}
}

// comp34006Defaults پارامترها، جدول‌های سند و فهرست‌های مرجع این نظام.
func comp34006Defaults(ctx context.Context, deps modulekit.Deps) map[string]any {
	params := comp34006Params(deps, nil)
	return map[string]any{
		"scheme":               Comp34006SchemeKey,
		"schemeFa":             Comp34006SchemeFa,
		"params":               params,
		"ladders":              rules.Comp34006Ladders(params),
		"factorTable":          rules.Comp34006FactorTable(params),
		"postClasses":          rules.Comp34006PostClasses(params),
		"roleTypes":            rules.Comp34006RoleTypes(params),
		"regions":              rules.Comp34006Regions(params),
		"shiftPatterns":        rules.Comp34006ShiftPatterns(params),
		"grades":               comp34006GradeOptions(params),
		"employees":            comp34006EmployeeOptions(ctx, deps),
		"apply":                params.PayrollApply,
		"today":                payTodayFa(deps),
		"independenceFa":       rules.Comp34006IndependenceFa(params.PayrollApply),
		"instructionFa":        rules.Comp34006InstructionFa(),
		"payslipInstructionFa": rules.Comp34006PayslipInstructionFa(),
		"noteFa":               comp34006NoteFa(),
	}
}

// comp34006Calc بستهٔ پرداخت یک کارمند را با پارامترهای شرکت می‌سازد.
func comp34006Calc(deps modulekit.Deps, in rules.Comp34006Input) map[string]any {
	params := comp34006Params(deps, nil)
	result := rules.Comp34006Calc(in, params)
	return map[string]any{
		"scheme":         Comp34006SchemeKey,
		"schemeFa":       Comp34006SchemeFa,
		"result":         result,
		"params":         params,
		"independenceFa": rules.Comp34006IndependenceFa(params.PayrollApply),
		"noteFa":         comp34006NoteFa(),
	}
}

// comp34006GradeOptions گریدهای شغلی با امتیاز هر جدول، برای فهرست انتخابی فرم.
//
// چرا در سرور ساخته می‌شود: امتیاز هر گرید از پارامترهای شرکت می‌آید؛ اگر در رابط ساخته
// شود، با تغییر پارامترها ناهمخوان می‌شود.
func comp34006GradeOptions(params rules.Comp34006Params) []map[string]any {
	out := make([]map[string]any, 0, params.MaxJobGrade)
	for grade := 1; grade <= params.MaxJobGrade; grade++ {
		out = append(out, map[string]any{
			"grade":      grade,
			"titleFa":    rules.Comp34006GradeTitleFa(grade),
			"jobPoints":  rules.Comp34006Points("job", grade, params),
			"postPoints": rules.Comp34006Points("strategic", grade, params),
			"rolePoints": rules.Comp34006Points("role", grade, params),
		})
	}
	return out
}

// comp34006EmployeeOptions کارکنان برای فهرست انتخابی فرم.
func comp34006EmployeeOptions(ctx context.Context, deps modulekit.Deps) []map[string]string {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return nil
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 300})
	if err != nil {
		return nil
	}
	out := make([]map[string]string, 0, len(page.Items))
	for _, emp := range page.Items {
		out = append(out, map[string]string{
			"id": emp.ID, "titleFa": emp.FullName, "departmentFa": emp.DepartmentName,
		})
	}
	return out
}

// comp34006Params پارامترهای این مدل را از پارامترهای شرکت فعال می‌خواند.
//
// هر کلید `comp34006.*` که در پروفایل شرکت باشد همین‌جا خوانده می‌شود؛ پس تغییر مصوبهٔ
// سالانه (ضریب ریالی، سقف امتیاز جدول‌ها، درصدها و مبالغ مصوب) نیاز به انتشار نسخهٔ تازه
// ندارد و پارامترهای نبوده، همان مقادیر خود سند می‌مانند.
func comp34006Params(deps modulekit.Deps, overrides map[string]string) rules.Comp34006Params {
	values := map[string]string{}
	for _, p := range deps.Org.Params {
		if strings.HasPrefix(p.Key, "comp34006.") {
			values[p.Key] = p.Value
		}
	}
	for key, value := range overrides {
		if strings.HasPrefix(key, "comp34006.") {
			values[key] = value
		}
	}
	return rules.Comp34006ParamsFromText(values)
}

// Comp34006SchemeFa عنوان رسمی این نظام پرداخت.
const Comp34006SchemeFa = "نظام جبران خدمات دستورالعمل ۳۴۰۰۶ (IMS-HR-34006)"

// comp34006NoteFa یادداشت رسمی مرز این مدل با دو مدل دیگر پرداخت.
func comp34006NoteFa() string {
	return "این مدل، سومین نظام پرداخت مستقل سازمان است (کنار مدل حقوق و دستمزد جاری و مدل " +
		"صندوق نفت). انتخاب آن برای هر کارمند در «روش پرداخت کارکنان» ثبت می‌شود و تا وقتی " +
		"پارامتر «اعمال در پرداخت» روشن نشود، محاسبهٔ آن فقط پیشنهادی است و فیش ماهانه را " +
		"عوض نمی‌کند."
}
