// پایانه‌های HTTP ماژول «payroll».

package payroll

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/payroll/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registerFinanceEndpoints مسیرهای پارامتریک حقوق را ثبت می‌کند: خواندن پارامترها و
// ضرائب، گزینه‌های محاسبه، و شبیه‌سازی فیش بر پایهٔ همان پارامترها.
//
// قاعدهٔ معماری: این مسیرها هیچ عددی از خودشان نمی‌سازند؛ پارامترها را از استقرار شرکت
// و رکوردهای «پارامتر حقوق» می‌خوانند و محاسبه را به موتور قواعد (`rules`) می‌سپارند.
func registerFinanceEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "write", fn) }

	mux.HandleFunc("GET /api/v1/payroll/parameters", read(func(w http.ResponseWriter, r *http.Request) {
		params, values, items := collectParams(r.Context(), deps, svc)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"values": values,
			"items":  items,
			"typed":  params,
		})
	}))

	mux.HandleFunc("GET /api/v1/payroll/options", read(func(w http.ResponseWriter, r *http.Request) {
		type opt struct {
			Value   string `json:"value"`
			LabelFa string `json:"labelFa"`
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"categories": []opt{{"earning", "مزایا"}, {"deduction", "کسور"}},
			"calcs":      []opt{{domain.CalcFixed, "مبلغ ثابت"}, {domain.CalcPercent, "درصدی از مبنا"}, {domain.CalcFormula, "فرمول محاسبه"}},
			"bases": []opt{
				{"base", "پایهٔ حقوق"}, {"mazd", "مزد مبنا"}, {"gross", "جمع ناخالص"},
				{"housing", "حق مسکن"}, {"karana", "کارانه"}, {"constant", "مقدار ثابت"},
			},
		})
	}))

	mux.HandleFunc("POST /api/v1/payroll/simulate", write(func(w http.ResponseWriter, r *http.Request) {
		var body simulateInput
		if !httpx.DecodeJSON(w, r, &body) {
			return
		}
		params, values, _ := collectParams(r.Context(), deps, svc)
		for k, v := range body.Overrides {
			values[k] = v
		}
		params = applyOverrides(params, values)

		input := body.input()
		result := rules.ComputePayroll(input, params)
		env, _ := simulateEnv(body, values, params, input, result)

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"result":    result,
			"params":    params,
			"elements":  elementBreakdown(activeElements(r.Context(), svc), env),
			"baseGiven": body.BaseSalary,
		})
	}))

	// فیش حقوقی پارامتریک یک کارمند: همان اقلام و همان موتور شبیه‌ساز. صفحهٔ «فیش
	// حقوقی» از همین می‌خواند تا فوق‌العاده‌ها و اجزای مزد در فیش دیده شوند.
	mux.HandleFunc("GET /api/v1/payroll/payslip", read(func(w http.ResponseWriter, r *http.Request) {
		reader, ok := deps.Ports.Employees()
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "دادهٔ کارکنان در این استقرار در دسترس نیست", nil)
			return
		}
		empID := strings.TrimSpace(r.URL.Query().Get("employee"))
		if empID == "" {
			if u, ok := auth.UserFrom(r.Context()); ok {
				empID = u.EmployeeID
			}
		}
		if empID == "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "کد کارمند مشخص نیست", nil)
			return
		}
		emp, found, err := reader.Get(r.Context(), contracts.EmployeeFilter{Scope: "all"}, empID)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		if !found {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "کارمند یافت نشد", nil)
			return
		}

		params, values, _ := collectParams(r.Context(), deps, svc)
		body := simulateInput{BaseSalary: emp.BaseSalary}
		input := body.input()
		result := rules.ComputePayroll(input, params)
		env, wages := simulateEnv(body, values, params, input, result)

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"employee": map[string]any{
				"id": emp.ID, "fullName": emp.FullName, "department": emp.DepartmentName,
				"position": emp.Position, "baseSalary": emp.BaseSalary,
			},
			"period":   periodFa(deps),
			"result":   result,
			"wages":    wages,
			"elements": elementBreakdown(activeElements(r.Context(), svc), env),
		})
	}))

	// پورت اقلام حقوق: فیش چاپی و اسناد کارگزینی همان تفکیک اقلام شبیه‌ساز را نشان می‌دهند.
	deps.Ports.SetPayElements(payElementProvider{svc: svc, deps: deps})
}

// simulateInput ورودی شبیه‌سازی فیش. افزون بر مزد شغل و اضافه‌کاری، اجزای مزد
// (رتبه/سنوات/پست/ایثارگری)، اقلام متغیر (کارانه، مأموریت) و ساعت‌های جمعه/تعطیل‌کاری
// هم گرفته می‌شود تا همهٔ اقلام پارامتریک قابل محاسبه باشند.
type simulateInput struct {
	BaseSalary         int64             `json:"baseSalary"`
	Children           int               `json:"children"`
	OvertimeHours      int64             `json:"overtimeHours"`
	OvertimeHourlyRate int64             `json:"overtimeHourlyRate"`
	OtherDeductions    int64             `json:"otherDeductions"`
	RankWage           int64             `json:"rankWage"`
	SeniorityWage      int64             `json:"seniorityWage"`
	PostAllowance      int64             `json:"postAllowance"`
	Sacrifice          int64             `json:"sacrificeAllowance"`
	KaranaAmount       int64             `json:"karanaAmount"`
	MissionDays        int64             `json:"missionDays"`
	MissionPerDiem     int64             `json:"missionPerDiem"`
	FridayHours        int64             `json:"fridayHours"`
	HolidayHours       int64             `json:"holidayHours"`
	Overrides          map[string]string `json:"overrides"`
}

// input ورودی محاسبهٔ موتور قواعد را می‌سازد.
func (b simulateInput) input() rules.PayrollInput {
	return rules.PayrollInput{
		BaseSalary:         b.BaseSalary,
		Children:           b.Children,
		OvertimeHours:      b.OvertimeHours,
		OvertimeHourlyRate: b.OvertimeHourlyRate,
		OtherDeductions:    b.OtherDeductions,
	}
}

// simulateEnv محیط محاسبهٔ اقلام را می‌سازد: متغیرهای پایه از ورودی و نتیجهٔ قواعد،
// پارامترهای نقطه‌دار فعال، اجزای مزد، و مقادیر متغیر کارانه/مأموریت/جمعه‌کاری.
// اجزای مزد را هم برمی‌گرداند تا فیش بتواند مزد مبنا را نشان دهد.
func simulateEnv(body simulateInput, values map[string]string, params rules.PayrollParams, input rules.PayrollInput, result rules.PayrollResult) (rules.ElementEnv, rules.WageComponents) {
	vars := rules.BaseElementVars(input, result)
	for k, v := range rules.ParamsVars(params) {
		vars[k] = v
	}
	for k, v := range values {
		if f, ok := floatOf(v); ok {
			vars[k] = f
		}
	}
	vars["karanaAmount"] = float64(body.KaranaAmount)
	vars["approvedKaranaAmount"] = float64(body.KaranaAmount)
	vars["missionDays"] = float64(body.MissionDays)
	vars["mission.perDiem"] = float64(body.MissionPerDiem)
	vars["fridayHours"] = float64(body.FridayHours)
	vars["holidayHours"] = float64(body.HolidayHours)

	// مزد مبنا از جمع اجزای مزد ساخته می‌شود؛ مزد شغل پیش‌فرض همان مزد ورودی است.
	wages := rules.WageComponents{
		Job:           pickWage(body.BaseSalary, vars["wage.job"]),
		Rank:          pickWage(body.RankWage, vars["wage.rank"]),
		Seniority:     pickWage(body.SeniorityWage, vars["wage.seniority"]),
		PostAllowance: pickWage(body.PostAllowance, vars["allowance.post"]),
		Sacrifice:     pickWage(body.Sacrifice, vars["allowance.sacrifice"]),
	}
	for k, v := range wages.Vars() {
		vars[k] = v
	}
	return rules.ElementEnv{
		Bases:     rules.BaseElementBases(input, params, result, float64(body.KaranaAmount), wages.Total()),
		Vars:      vars,
		Canonical: rules.CanonicalElementAmounts(result),
	}, wages
}

// periodFa دورهٔ جاری را به شکل «۱۴۰۵/۰۷» برمی‌گرداند.
func periodFa(deps modulekit.Deps) string {
	today, err := calendar.FromGregorian(deps.Now())
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%04d/%02d", today.Year, today.Month)
}

// pickWage مقدار ورودی کاربر را اگر داده شده باشد برمی‌گرداند، وگرنه مقدار پارامتر.
func pickWage(given int64, fallback float64) float64 {
	if given != 0 {
		return float64(given)
	}
	return fallback
}

// paramItem یک پارامتر حقوق در پاسخ API.
type paramItem struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	Value   string `json:"value"`
	UnitFa  string `json:"unitFa"`
	GroupFa string `json:"groupFa"`
	Status  string `json:"status,omitempty"`
	Record  string `json:"recordId,omitempty"`
}

// collectParams پارامترها را از استقرار شرکت فعال و رکوردهای «پارامتر حقوق» یک‌کاسه
// می‌کند. رکوردهای فعال بر پارامتر پایهٔ شرکت اولویت دارند.
func collectParams(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) (rules.PayrollParams, map[string]string, []paramItem) {
	values := map[string]string{}
	items := []paramItem{}
	seen := map[string]int{}

	// لایهٔ ۱: پارامترهای گروه «حقوق و دستمزد» از استقرار شرکت فعال.
	for _, p := range deps.Org.Params {
		if p.GroupFa != "حقوق و دستمزد" {
			continue
		}
		values[p.Key] = p.Value
		items = append(items, paramItem{Key: p.Key, TitleFa: p.TitleFa, Value: p.Value, UnitFa: p.UnitFa, GroupFa: p.GroupFa})
		seen[p.Key] = len(items) - 1
	}

	// لایهٔ ۲: رکوردهای «پارامتر حقوق» — فقط پارامتر تأییدشده (active) اثر عملیاتی
	// دارد؛ پیش‌نویس و غیرفعال صرفاً دادهٔ نمایشی‌اند و در محاسبهٔ حقوق به کار نمی‌روند.
	// تاریخ اجرا (effectiveFrom) رعایت می‌شود: پارامتری که تاریخش هنوز نرسیده اثر
	// ندارد، و اگر چند نسخهٔ جاری از یک کلید باشد، تازه‌ترین تاریخ اجرا برنده است؛ نسخهٔ
	// قدیمی حذف نمی‌شود بلکه کنار می‌رود (تاریخ‌گذاری مؤثر بدون بازنویسی سابقه — همان
	// الگوی سالانهٔ نرخ بیمه و معافیت مالیات در سامانه‌های پیشرو).
	if svc != nil {
		todayInt := 0
		if today, err := calendar.FromGregorian(deps.Now()); err == nil {
			todayInt = today.Year*10000 + today.Month*100 + today.Day
		}
		page, err := svc.List(ctx, contracts.RecordFilter{Module: "payroll", Kind: "payParams", Size: 500})
		if err == nil {
			type version struct {
				item paramItem
				from int // تاریخ اجرا به قالب YYYYMMDD؛ صفر یعنی بلافاصله/بدون تاریخ
				line int // شمارهٔ ردیف برای شکستن تساوی (ردیف تازه‌تر برنده)
			}
			versions := map[string]version{}
			for i, rec := range page.Items {
				if rec.Status != "active" {
					continue
				}
				key := strings.TrimSpace(rec.Fields["key"])
				if key == "" {
					continue
				}
				from := 0
				if raw := strings.TrimSpace(rec.Fields["effectiveFrom"]); raw != "" {
					day, err := calendar.Parse(raw)
					if err != nil {
						continue // تاریخ اجرای نامعتبر اثر ندارد
					}
					from = day.Year*10000 + day.Month*100 + day.Day
					if from > todayInt {
						continue // هنوز به تاریخ اجرا نرسیده‌ایم
					}
				}
				v := version{
					item: paramItem{
						Key: key, TitleFa: rec.Title, Value: rec.Fields["value"],
						UnitFa: rec.Fields["unitFa"], GroupFa: rec.Fields["groupFa"],
						Status: rec.Status, Record: rec.ID,
					},
					from: from, line: i,
				}
				prev, ok := versions[key]
				better := !ok
				if ok {
					// تقدم نسخه: هر نسخهٔ تاریخ‌دارِ جاری بر هر نسخهٔ بی‌تاریخ مقدم است
					// (بی‌تاریخ یعنی «قدیمی/نامعلوم»)؛ وگرنه تاریخ بزرگ‌تر برنده است و در
					// تساوی، ردیف تازه‌تر. بدون این تقدم، نتیجه به ترتیب ردیف‌های انبار
					// وابسته می‌شد و نسخهٔ جایگزینِ امروزی گاه نادیده گرفته می‌شد.
					switch {
					case prev.from == 0 && v.from != 0:
						better = true
					case prev.from != 0 && v.from == 0:
						better = false
					case prev.from != 0 && v.from != 0:
						better = v.from > prev.from || (v.from == prev.from && v.line > prev.line)
					default:
						better = v.line > prev.line // هر دو بی‌تاریخ؛ ردیف تازه‌تر برنده
					}
				}
				if better {
					versions[key] = v
				}
			}
			for _, v := range versions {
				values[v.item.Key] = v.item.Value
				if idx, ok := seen[v.item.Key]; ok {
					items[idx].Value = v.item.Value
					items[idx].Status = v.item.Status
					items[idx].Record = v.item.Record
				} else {
					items = append(items, v.item)
					seen[v.item.Key] = len(items) - 1
				}
			}
		}
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].GroupFa < items[j].GroupFa })
	return paramsFromValues(values), values, items
}

// paramsFromValues مقادیر رشته‌ای را به پارامترهای محاسبهٔ حقوق تبدیل می‌کند.
// پیش‌فرض‌ها از موتور قواعد می‌آید و هر مقدار موجود، جایگزین می‌شود.
func paramsFromValues(values map[string]string) rules.PayrollParams {
	p := rules.DefaultPayrollParams()
	if v, ok := floatOf(values["insurance.employeeRate"]); ok {
		p.InsuranceEmployeeRate = v
	}
	if v, ok := floatOf(values["insurance.employerRate"]); ok {
		p.InsuranceEmployerRate = v
	}
	if v, ok := intOf(values["insurance.ceiling"]); ok {
		p.InsuranceCeiling = v
	}
	if v, ok := intOf(values["allowance.housing"]); ok {
		p.HousingAllowance = v
	}
	if v, ok := intOf(values["allowance.child"]); ok {
		p.ChildAllowancePerChild = v
	}
	if v, ok := intOf(values["tax.exemption"]); ok {
		p.TaxExemption = v
	}
	if raw, ok := values["tax.brackets"]; ok && strings.TrimSpace(raw) != "" {
		if brackets, err := parseBrackets(raw); err == nil {
			p.TaxBrackets = brackets
		}
	}
	return p
}

// applyOverrides همان محاسبه را با مقادیر به‌روزشدهٔ ورودی کاربر تکرار می‌کند.
func applyOverrides(_ rules.PayrollParams, values map[string]string) rules.PayrollParams {
	return paramsFromValues(values)
}

// parseBrackets رشتهٔ «سقف:نرخ,سقف:نرخ» را به پله‌های مالیاتی تبدیل می‌کند؛ سقف صفر
// یعنی بی‌سقف (آخرین پله).
func parseBrackets(raw string) ([]rules.TaxBracket, error) {
	out := []rules.TaxBracket{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pair := strings.SplitN(part, ":", 2)
		if len(pair) != 2 {
			return nil, fmt.Errorf("پلهٔ مالیاتی نامعتبر: %s", part)
		}
		upto, okUpto := intOf(pair[0])
		rate, okRate := floatOf(pair[1])
		if !okUpto || !okRate {
			return nil, fmt.Errorf("پلهٔ مالیاتی نامعتبر: %s", part)
		}
		out = append(out, rules.TaxBracket{Upto: upto, Rate: rate})
	}
	return out, nil
}

// activeElements اقلام حقوق غیرغیرفعال را از انبار می‌خواند.
func activeElements(ctx context.Context, svc *recordkit.Service) []contracts.Record {
	if svc == nil {
		return nil
	}
	page, err := svc.List(ctx, contracts.RecordFilter{Module: "payroll", Kind: "payItems", Size: 500})
	if err != nil {
		return nil
	}
	out := make([]contracts.Record, 0, len(page.Items))
	for _, rec := range page.Items {
		if rec.Status != "inactive" {
			out = append(out, rec)
		}
	}
	return out
}

// elementBreakdown اقلام فعال حقوق را با محیط داده‌شده محاسبه می‌کند تا کاربر اثر هر
// قلم را روی فیش ببیند. محاسبه در موتور قواعد (`rules.ComputeElements`) انجام می‌گیرد؛
// قلم فرمولی هم مثل ثابت و درصدی عدد می‌دهد و فقط فرمولی که متغیرش موجود نباشد
// «حل‌نشده» می‌ماند.
func elementBreakdown(records []contracts.Record, env rules.ElementEnv) []rules.PayElementLine {
	return rules.ComputeElements(toPayElements(records), env)
}

// toPayElements رکوردهای «قلم حقوق» را به تعریف پارامتریک قابل محاسبه تبدیل می‌کند.
func toPayElements(records []contracts.Record) []rules.PayElement {
	out := make([]rules.PayElement, 0, len(records))
	for _, rec := range records {
		out = append(out, rules.PayElement{
			Code:     rec.Fields["code"],
			TitleFa:  rec.Title,
			Category: rec.Fields["category"],
			Calc:     rec.Fields["calc"],
			Rate:     rec.Fields["rate"],
			Basis:    rec.Fields["basis"],
			Formula:  rec.Fields["formula"],
		})
	}
	return out
}

// payElementProvider پورت اقلام حقوق برای موتور اسناد؛ فیش چاپی و احکام کارگزینی از
// همین اقلام و همین موتور محاسبه استفاده می‌کنند تا با شبیه‌ساز یکسان بمانند.
type payElementProvider struct {
	svc  *recordkit.Service
	deps modulekit.Deps
}

// PayElements اقلام فعال حقوق را به ترتیب نمایش برمی‌گرداند.
func (p payElementProvider) PayElements(ctx context.Context) []rules.PayElement {
	return toPayElements(activeElements(ctx, p.svc))
}

// PayParamValues مقادیر عددی پارامترهای فعال حقوق را می‌دهد تا اسناد چاپی (حکم
// کارگزینی و قرارداد کار) بدون دسترسی مستقیم به انبار ماژول حقوق بتوانند اقلام را حساب کنند.
func (p payElementProvider) PayParamValues(ctx context.Context) map[string]string {
	_, values, _ := collectParams(ctx, p.deps, p.svc)
	return values
}

// basisFa برچسب فارسی مبنای محاسبه؛ از موتور قواعد خوانده می‌شود تا یک منبع حقیقت باشد.
func basisFa(basis string) string { return rules.BasisFa(basis) }

// floatOf تبدیل یک رشته (عدد یا کلید پارامتر) به عدد اعشاری.
func floatOf(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if v, err := strconv.ParseFloat(raw, 64); err == nil {
		return v, true
	}
	return 0, false
}

// intOf تبدیل یک رشته به عدد صحیح (با پذیرش ارقام فارسی).
func intOf(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return v, true
	}
	if f, ok := floatOf(raw); ok {
		return int64(f), true
	}
	return 0, false
}
