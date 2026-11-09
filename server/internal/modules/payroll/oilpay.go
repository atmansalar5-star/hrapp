// پیاده‌سازی ماژول «حقوق و دستمزد» — نظام پرداخت صندوق بازنشستگی نفت.
//
// چرا این فایل جدا از `handlers.go`: سازمان دو نظام پرداخت دارد و این یکی ساختار خودش را
// دارد (پایهٔ حقوق، فوق‌العادهٔ مناطق گرمسیری ۲۵٪، فوق‌العادهٔ ویژه ۳۰٪ و جدول‌های ۱ تا ۳
// آیین‌نامه). قاعده در `rules.ComputeOilPayslip` است و این فایل فقط پارامترهای شرکت را
// می‌خواند، کارمند را از پورت ماژول کارکنان می‌گیرد و فیش را برمی‌گرداند؛ پس فرم وب،
// رکورد فیش و هر گزارش دیگر یک عدد می‌دهند.
package payroll

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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

	// پارامترها، فهرست فوق‌العاده‌ها، جدول‌های آیین‌نامه، کارکنان و دورهٔ جاری.
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
		httpx.WriteJSON(w, http.StatusOK, oilPayslip(r.Context(), deps, in))
	}))
}

// oilPayDefaults پارامترها و فهرست‌های مرجع نظام صندوق نفت.
func oilPayDefaults(ctx context.Context, deps modulekit.Deps) map[string]any {
	params := oilPayParams(deps, nil)
	return map[string]any{
		"params":       params,
		"allowances":   rules.OilAllowances(params),
		"shiftOptions": []map[string]string{{"key": "no", "titleFa": "غیر نوبت‌کار"}, {"key": "yes", "titleFa": "نوبت‌کار"}},
		"degrees":      oilDegreeOptions(params),
		// جدول ۳ (پایهٔ سمت → حق جذب عمومی و حق مقام) و جدول‌های ۱ و ۲ (محل سکونت).
		"jobGrades":   oilJobGradeOptions(params),
		"residences":  oilResidenceOptions(params),
		"employees":   oilEmployeeOptions(ctx, deps, params),
		"paramViews":  oilParamViews(deps, nil),
		"period":      oilPeriod(deps),
		"today":       oilToday(deps),
		"daysInMonth": oilDaysInMonth(deps),
		"schemeFa":    "نظام پرداخت صندوق بازنشستگی نفت",
		"allowanceBases": []map[string]string{
			{"key": rules.OilBaseAllowance, "titleFa": "مبنای فوق‌العاده‌ها (پایه + گرمسیری)"},
			{"key": rules.OilBaseAttraction, "titleFa": "مبنای جذب (پایه + گرمسیری + ویژه)"},
			{"key": rules.OilBaseSalary, "titleFa": "پایهٔ حقوق"},
		},
		"noteFa": oilPayNoteFa(),
	}
}

// oilPayslip فیش را با پارامترهای شرکت می‌سازد و دورهٔ جاری را هم برمی‌گرداند.
//
// امتیاز ایثارگری از پروندهٔ تأییدشدهٔ همان کارمند خوانده می‌شود؛ خودِ محاسبه در
// `rules.ComputeOilPayslip` است و این تابع فقط ورودی موتور را کامل می‌کند.
func oilPayslip(ctx context.Context, deps modulekit.Deps, in rules.OilPayInput) map[string]any {
	in.Veteran = veteranEntitlement(ctx, deps, in.EmployeeID)
	params := oilPayParams(deps, in.ParamOverrides)
	slip := rules.ComputeOilPayslip(in, params)
	return map[string]any{
		"scheme":      rules.SchemeOil,
		"schemeFa":    "نظام پرداخت صندوق بازنشستگی نفت",
		"slip":        slip,
		"input":       in,
		"params":      params,
		"noteFa":      oilPayNoteFa(),
		"period":      oilPeriod(deps),
		"daysInMonth": oilDaysInMonth(deps),
	}
}

// oilPayParams آستانه‌ها و درصدهای نظام صندوق را از پروفایل شرکت فعال می‌خواند و روی آن،
// بازنویسی‌های همین محاسبه را می‌نشاند.
//
// هر کلید `oilpay.*` که در پارامترهای شرکت باشد همین‌جا خوانده می‌شود؛ پس تغییر مصوبه
// نیاز به انتشار نسخهٔ تازه ندارد. خانه‌های جدول‌های آیین‌نامه هم کلید پارامتری خودشان را
// دارند (`oilpay.grade.<پایه>.public|rank` و `oilpay.site.<محل>.deprivation|timeConsumedDay|
// timeConsumedShift`) تا مصوبهٔ تازه بدون تغییر کد اعمال شود.
//
// پارامتر overrides همان محاسبه را با اعداد دیگری اجرا می‌کند و پروفایل شرکت را تغییر
// نمی‌دهد؛ فرم فیش با همین راه همهٔ ضرائب، نرخ‌ها و خانه‌های جدول‌ها را پیش از اجرا قابل
// تغییر می‌کند.
func oilPayParams(deps modulekit.Deps, overrides map[string]string) rules.OilPayParams {
	values := oilParamValues(deps, overrides)
	params := rules.DefaultOilPayParams()
	// اقلام «مبلغ ثابت» فیش رسمی (عایله‌مندی، مسکن، حق اولاد، فوق‌العادهٔ خاص) هم به فهرست
	// می‌آیند تا مبلغ مصوبشان از پارامتر خوانده و در فیش نشان داده شود.
	params.Allowances = rules.OilAllowanceCatalog(params.Allowances)
	if v, ok := oilParamOf(values, "oilpay.climatePercent"); ok && v > 0 {
		params.ClimatePercent = v
	}
	// فوق‌العاده‌های درصدی: درصد هر قلم از پارامتر همان قلم می‌آید
	// (مثلاً oilpay.allowance.special = 30).
	for i := range params.Allowances {
		if v, ok := oilParamOf(values, "oilpay.allowance."+params.Allowances[i].Key); ok && v >= 0 {
			if params.Allowances[i].Fixed {
				params.Allowances[i].Amount = int64(v)
			} else {
				params.Allowances[i].Percent = v
			}
		}
	}
	if v, ok := oilParamOf(values, "oilpay.fund.employeePercent"); ok && v >= 0 {
		params.FundEmployeePercent = v
	}
	if v, ok := oilParamOf(values, "oilpay.fund.employerPercent"); ok && v >= 0 {
		params.FundEmployerPercent = v
	}
	if v, ok := oilParamOf(values, "oilpay.insurance.employeePercent"); ok && v >= 0 {
		params.InsuranceEmployeePercent = v
	}
	if v, ok := oilParamOf(values, "oilpay.tax.exemption"); ok && v >= 0 {
		params.TaxExemption = int64(v)
	}
	// نرخ اضافه‌کار هر ساعت به نسبت پایهٔ حقوق (آیین‌نامه: روزکار ۰٫۰۱۴، نوبت‌کار ۰٫۰۱۶).
	if v, ok := oilParamOf(values, "oilpay.overtimeRate.day"); ok && v >= 0 {
		params.OvertimeRateDay = v
	}
	if v, ok := oilParamOf(values, "oilpay.overtimeRate.shift"); ok && v >= 0 {
		params.OvertimeRateShift = v
	}
	// مبنای دستی کسور و مالیات (صفر = مبنای خودکار) و مبلغ هر پایهٔ تشویقی ایثارگری.
	if v, ok := oilParamOf(values, "oilpay.fund.base"); ok && v >= 0 {
		params.FundBase = int64(v)
	}
	if v, ok := oilParamOf(values, "oilpay.tax.base"); ok && v >= 0 {
		params.TaxBase = int64(v)
	}
	if v, ok := oilParamOf(values, "oilpay.veteran.incentiveStep"); ok && v >= 0 {
		params.VeteranIncentiveStep = int64(v)
	}
	// فوق‌العادهٔ تخصصی بر اساس مدرک: درصد هر مقطع از پارامتر همان مقطع
	// (مثلاً oilpay.degree.bachelor = 38.5). صفر یعنی مقطع مشمول نیست.
	if params.DegreeAllowancePercents == nil {
		params.DegreeAllowancePercents = map[string]float64{}
	}
	for _, level := range contracts.DegreeLevels() {
		if v, ok := oilParamOf(values, "oilpay.degree."+level.Key); ok && v >= 0 {
			params.DegreeAllowancePercents[level.Key] = v
		}
	}
	oilTaxBrackets(values, &params)
	// حالت دستی: وقتی مالیات از سطرهای مصوبه می‌آید، محاسبهٔ پله‌ای خاموش می‌شود تا
	// مالیات دو بار کسر نشود (oilpay.tax.mode = manual).
	if strings.EqualFold(strings.TrimSpace(values["oilpay.tax.mode"]), "manual") {
		params.TaxBrackets = nil
	}
	oilTableParams(values, &params)
	oilGradeMapParams(values, &params)
	return params
}

// oilGradeMapParams نگاشت «گروه شغلی → پایهٔ سمت» را از پارامترهای شرکت بازنویسی می‌کند.
//
// کلید پارامتر: oilpay.gradeMap.<کلید گروه شغلی> و مقدارش کلید پایهٔ سمت جدول ۳
// (مثلاً oilpay.gradeMap.manager = A). فقط پیشنهاد فرم است؛ درصد هیچ قلمی از آن نمی‌آید.
func oilGradeMapParams(values map[string]string, params *rules.OilPayParams) {
	if params.GradeSiteMap == nil {
		params.GradeSiteMap = map[string]string{}
	}
	for key, raw := range values {
		parts := strings.Split(key, ".")
		if len(parts) != 3 || parts[0] != "oilpay" || parts[1] != "gradeMap" {
			continue
		}
		grade := strings.TrimSpace(parts[2])
		site := strings.TrimSpace(raw)
		if grade == "" {
			continue
		}
		if site == "" || site == "-" {
			delete(params.GradeSiteMap, grade)
			continue
		}
		params.GradeSiteMap[grade] = site
	}
}

// oilTableParams خانه‌های جدول ۱، ۲ و ۳ آیین‌نامه را از پارامترهای مؤثر بازنویسی می‌کند.
//
// چرا با پیشوند و نه با فهرست ثابت کلیدها: خانه‌های جدول‌ها ۳۵ عددند و افزودن همه به
// پروفایل شرکت، فهرست پارامترها را ناخوانا می‌کرد. هر کلیدی که با این الگو بیاید اعمال
// می‌شود و کلید ناشناخته بی‌اثر می‌ماند. چون ورودی این تابع «نقشهٔ مؤثر» است، بازنویسی فرم
// فیش روی همین خانه‌ها هم اثر می‌گذارد.
func oilTableParams(values map[string]string, params *rules.OilPayParams) {
	for key, raw := range values {
		parts := strings.Split(key, ".")
		if len(parts) != 4 || parts[0] != "oilpay" {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil || value < 0 {
			continue
		}
		switch parts[1] {
		case "grade":
			for i := range params.JobGrades {
				if !strings.EqualFold(params.JobGrades[i].Key, parts[2]) {
					continue
				}
				switch parts[3] {
				case "public":
					params.JobGrades[i].PublicAttract = value
				case "rank":
					params.JobGrades[i].PositionRank = value
				}
			}
		case "site":
			for i := range params.Residences {
				if params.Residences[i].Key != parts[2] {
					continue
				}
				switch parts[3] {
				case "deprivation":
					params.Residences[i].Deprivation = value
				case "timeConsumedDay":
					params.Residences[i].TimeConsumedDay = value
				case "timeConsumedShift":
					params.Residences[i].TimeConsumedShift = value
				}
			}
		}
	}
}

// oilTaxBrackets پله‌های مالیات را از قالب متنی «سقف:نرخ,سقف:نرخ» بازنویسی می‌کند.
//
// چرا متنی: پله‌های مالیات فهرست‌اند و در پروفایل شرکت یک کلید عددی ندارند؛ این قالب کوتاه
// اجازه می‌دهد کارشناس پیش از اجرا پله‌ها را عوض کند. نرخ بزرگ‌تر از ۱ درصد شمرده می‌شود
// (۱۰ یعنی ۱۰٪)، سقف صفر یعنی «آخرین پله» و قالب نامعتبر نادیده گرفته می‌شود تا پیش‌فرض
// قانونی بماند.
func oilTaxBrackets(values map[string]string, params *rules.OilPayParams) {
	raw, ok := values["oilpay.tax.brackets"]
	if !ok || strings.TrimSpace(raw) == "" {
		return
	}
	out := make([]rules.TaxBracket, 0, 4)
	for _, part := range strings.Split(raw, ",") {
		pair := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(pair) != 2 {
			return
		}
		upto, err := strconv.ParseInt(strings.TrimSpace(pair[0]), 10, 64)
		if err != nil || upto < 0 {
			return
		}
		rate, err := strconv.ParseFloat(strings.TrimSpace(pair[1]), 64)
		if err != nil || rate < 0 {
			return
		}
		if rate > 1 {
			rate /= 100
		}
		out = append(out, rules.TaxBracket{Upto: upto, Rate: rate})
	}
	if len(out) > 0 {
		params.TaxBrackets = out
	}
}

// oilParamValues نقشهٔ مؤثر پارامترهای نظام صندوق: پروفایل شرکت + بازنویسی همین محاسبه.
//
// چرا فقط `oilpay.*` پذیرفته می‌شود: این نقشه از بدنهٔ درخواست HTTP پر می‌شود؛ با محدود
// کردن پیشوند، فرم فیش نمی‌تواند پارامترهای نظام دیگر (مالیات عمومی، حضور و غیاب،
// ایثارگری و ...) را دست بزند.
func oilParamValues(deps modulekit.Deps, overrides map[string]string) map[string]string {
	out := make(map[string]string, len(deps.Org.Params)+len(overrides))
	for _, p := range deps.Org.Params {
		if strings.HasPrefix(p.Key, "oilpay.") {
			out[p.Key] = p.Value
		}
	}
	for key, value := range overrides {
		k := strings.TrimSpace(key)
		if !strings.HasPrefix(k, "oilpay.") {
			continue
		}
		out[k] = strings.TrimSpace(value)
	}
	return out
}

// oilParamOf مقدار عددی یک پارامتر از نقشهٔ مؤثر.
func oilParamOf(values map[string]string, key string) (float64, bool) {
	raw, ok := values[key]
	if !ok {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// oilParamViews پارامترهای نظام صندوق را برای نمایش و ویرایش در فرم برمی‌گرداند.
//
// کلیدهای بازنویسی‌شده هم اگر در پروفایل شرکت نباشند در فهرست می‌آیند تا فرم همیشه همهٔ
// ضرائب مؤثر همین محاسبه را نشان دهد.
func oilParamViews(deps modulekit.Deps, overrides map[string]string) []map[string]string {
	titles := map[string]map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "oilpay.") {
			continue
		}
		titles[p.Key] = map[string]string{"titleFa": p.TitleFa, "unitFa": p.UnitFa}
	}
	values := oilParamValues(deps, overrides)
	out := []map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "oilpay.") {
			continue
		}
		out = append(out, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": values[p.Key], "unitFa": p.UnitFa,
		})
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		if _, ok := titles[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		title := map[string]string{}
		if p, ok := titles[key]; ok {
			title = p
		}
		out = append(out, map[string]string{
			"key": key, "titleFa": title["titleFa"], "value": values[key], "unitFa": title["unitFa"],
		})
	}
	return out
}

// oilDegreeOptions مقاطع تحصیلی و ضریب فوق‌العادهٔ تخصصی هر مقطع.
//
// فهرست مقاطع از کاتالوگ مشترک `contracts.DegreeLevels` می‌آید (همان کلیدهایی که
// سابقهٔ تحصیلی کارمند استفاده می‌کند) و درصدها از پارامترهای شرکت.
func oilDegreeOptions(params rules.OilPayParams) []map[string]any {
	out := make([]map[string]any, 0, 6)
	for _, level := range contracts.DegreeLevels() {
		percent := rules.OilDegreePercent(params, level.Key)
		out = append(out, map[string]any{
			"key": level.Key, "titleFa": level.TitleFa, "order": level.Order, "percent": percent,
		})
	}
	return out
}

// oilJobGradeOptions ردیف‌های جدول ۳ آیین‌نامه (پایهٔ سمت) با درصد جمع.
func oilJobGradeOptions(params rules.OilPayParams) []map[string]any {
	out := make([]map[string]any, 0, len(params.JobGrades))
	for _, grade := range params.JobGrades {
		out = append(out, map[string]any{
			"key": grade.Key, "titleFa": grade.TitleFa,
			"publicAttract": grade.PublicAttract, "positionRank": grade.PositionRank,
			"total": grade.Total(),
		})
	}
	return out
}

// oilResidenceOptions ردیف‌های جدول ۱ و ۲ آیین‌نامه (محل سکونت).
func oilResidenceOptions(params rules.OilPayParams) []map[string]any {
	out := make([]map[string]any, 0, len(params.Residences))
	for _, site := range params.Residences {
		out = append(out, map[string]any{
			"key": site.Key, "titleFa": site.TitleFa,
			"deprivation":       site.Deprivation,
			"timeConsumedDay":   site.TimeConsumedDay,
			"timeConsumedShift": site.TimeConsumedShift,
		})
	}
	return out
}

// oilEmployeeOptions کارکنان برای انتخاب در فرم (از پورت ماژول کارکنان).
func oilEmployeeOptions(ctx context.Context, deps modulekit.Deps, params rules.OilPayParams) []map[string]string {
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
			// محل سکونت پرونده: درصد جدول ۱ و ۲ را خودکار در فرم می‌گذارد.
			"residenceSite":   e.ResidenceSite,
			"residenceSiteFa": e.ResidenceSiteFa,
			// گروه شغلی پرونده (اجرای ابلاغ آن را می‌نویسد) و پایهٔ سمت پیشنهادی جدول ۳
			// برای همان گروه؛ فرم فیش با انتخاب کارمند، پایهٔ سمت را خودکار می‌گذارد.
			"jobGrade":       e.JobGrade,
			"jobGradeFa":     e.JobGradeFa,
			"jobGradeSite":   rules.OilSiteForGrade(params, e.JobGrade),
			"jobGradeSiteFa": rules.OilJobGradeTitleFa(rules.OilSiteForGrade(params, e.JobGrade)),
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
	return "این نظام پرداخت برای کارکنان تابع صندوق بازنشستگی نفت است و بر پایهٔ " +
		"آیین‌نامهٔ فوق‌العاده‌ها کار می‌کند: پایهٔ حقوق (A) + فوق‌العادهٔ بدی آب و هوا " +
		"(۲۵٪ پایه، مناطق گرمسیری) «مبنای فوق‌العاده‌ها» (A+B) را می‌سازد؛ فوق‌العادهٔ ویژه " +
		"۳۰٪ این مبنا است و جمع آن دو «مبنای جذب» (A+B+C) را می‌دهد.\n" +
		"فوق‌العادهٔ محرومیت از تسهیلات زندگی (جدول ۱) و جبران صرف وقت اضافی (جدول ۲) درصدشان " +
		"از محل سکونت کارمند می‌آید و ستون روزکار/نوبت‌کار جدول ۲ با وضعیت نوبت‌کاری انتخاب " +
		"می‌شود. فوق‌العادهٔ جذب عمومی و تخصصی شغل (جدول ۳) از پایهٔ سمت (حق جذب عمومی + حق " +
		"مقام) و فوق‌العادهٔ تخصصی مدرک از ضریب مدرک تحصیلی می‌آید؛ هر دو روی مبنای جذب " +
		"بسته می‌شوند.\n" +
		"فوق‌العادهٔ نوبت‌کاری تنها برای نوبت‌کار پرداخت می‌شود و به نسبت روزهای نوبت‌کاری به " +
		"روزهای ماه محاسبه می‌گردد. اضافه‌کار هر ساعت ۰٫۰۱۴ پایهٔ حقوق برای روزکار و ۰٫۰۱۶ برای " +
		"نوبت‌کار است. کسور شامل سهم کارمند صندوق (۶٪)، بیمهٔ سهم کارمند (اگر پارامتر داشته " +
		"باشد) و مالیات پله‌ای است.\n" +
		"درصد هر قلم و هر خانهٔ جدول‌ها پارامتریک است (oilpay.allowance.<قلم>، " +
		"oilpay.degree.<مقطع>، oilpay.site.<محل>.* و oilpay.grade.<پایه>.*)؛ پس تغییر مصوبه " +
		"بدون انتشار نسخهٔ تازه اعمال می‌شود."
}
