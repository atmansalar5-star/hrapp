// مسیرهای اختصاصی نظام جبران خدمات ۴پی.
//
// چرا جدا از رکوردهای عمومی: سه کار اینجا انجام می‌شود که در موتور رکورد جا ندارد —
// ۱) خواندن پارامترهای مصوب شرکت و ساختن باندها و جدول ضریب‌ها، ۲) جمع‌آوری دادهٔ
// پیش‌فرض چهار عامل از پروندهٔ کارمند و حکم حقوق او، ۳) دادن ورودی به موتور قواعد و
// برگرداندن محاسبهٔ خط‌به‌خط. هیچ قاعده‌ای اینجا نوشته نمی‌شود؛ موتور `internal/rules`
// مرجع است و این فایل فقط داده را می‌برد و نتیجه را برمی‌گرداند.
//
// مرز دو نظام: نظام پرداخت جیب‌یست (G20) نظامی جدا و مستقل است؛ مسیرها، پارامترها و خروجی
// خودش را در همین بسته و در فایل g20.go دارد. این فایل هیچ تابع، پارامتر یا دادهٔ G20 را
// نمیخواند و پاسخهای ۴پی هم هیچ کلید G20 برنمیگردانند.
package compensation

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/compensation/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// ماژول‌هایی که دادهٔ پایهٔ چهار عامل از آن‌ها خوانده می‌شود (از راه پورت قرارداد).
const (
	performanceModuleID = "performance"
	perfResultKind      = "perfResults"
	orgStructureModule  = "orgstructure"
	positionKind        = "positions"
	jobClassModule      = "jobclass"
	jobDescriptionKind  = "jobDescriptions"
	jobCategoryKind     = "jobCategories"
)

// registerFourPEndpoints مسیرهای فهرست‌های مرجع، پیش‌فرض‌های چهارعاملی و محاسبه را ثبت می‌کند.
//
// همهٔ این مسیرها «خواندنی»اند: هیچ‌کدام داده ذخیره نمی‌کنند. ثبت تصمیم از همان مسیر
// استاندارد رکوردها (`POST /api/v1/compensation/records`) انجام می‌شود تا گردشکار،
// اعتبارسنجی و سابقهٔ رکورد یک‌جا و در دامنه اعمال شود.
func registerFourPEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("compensation", "read", fn) }

	// فهرست‌های مرجع ماژول: هر چه فرم و جدول لازم دارد، از سرور می‌آید.
	mux.HandleFunc("GET /api/v1/compensation/options", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, optionsPayload(deps))
	}))

	// پیش‌فرض‌های چهارعاملی یک کارمند: پارامترها، باندها، جدول ضریب‌ها و دادهٔ خود کارمند.
	mux.HandleFunc("GET /api/v1/compensation/4p-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		employeeID := strings.TrimSpace(r.URL.Query().Get("employeeId"))
		out := optionsPayload(deps)
		view, err := employeeDefaults(r.Context(), deps, svc, employeeID)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		for key, value := range view {
			out[key] = value
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))

	// محاسبهٔ بستهٔ جبران خدمات: ورودی ناقص با دادهٔ پیش‌فرض کارمند پر می‌شود تا فرم
	// بتواند فقط 「گزینه‌های سمت کاربر」 را بفرستد و عدد تکراری نسازد.
	mux.HandleFunc("POST /api/v1/compensation/4p-calc", read(func(w http.ResponseWriter, r *http.Request) {
		var in compensationCalcRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ورودی محاسبهٔ جبران خدمات خوانده نشد.", nil)
			return
		}
		params := compensationParams(deps)
		calc := in.toInput()
		// پیش‌فرض‌های کارمند: هر ورودی که صفر مانده باشد از پرونده و حکم حقوق پر می‌شود.
		if strings.TrimSpace(calc.EmployeeID) != "" {
			view, err := employeeDefaults(r.Context(), deps, svc, calc.EmployeeID)
			if err != nil {
				httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
				return
			}
			if defaults, ok := view["employee"].(map[string]any); ok {
				if calc.EmployeeName == "" {
					calc.EmployeeName, _ = defaults["fullName"].(string)
				}
				if calc.JobScore <= 0 {
					calc.JobScore = numberFromAny(defaults["jobScore"])
				}
				if calc.GradeOverride <= 0 {
					calc.GradeOverride = int(numberFromAny(defaults["gradeRank"]))
				}
				if calc.CompetencyLevel <= 0 {
					calc.CompetencyLevel = int(numberFromAny(defaults["competencyLevel"]))
				}
				if calc.PerformanceScore <= 0 {
					calc.PerformanceScore = numberFromAny(defaults["performanceScore"])
				}
				if calc.CurrentPay <= 0 {
					calc.CurrentPay = int64(numberFromAny(defaults["currentPay"]))
				}
			}
		}
		if calc.JobScore <= 0 && calc.GradeOverride <= 0 {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"امتیاز ارزشیابی شغل (P1) یا پلهٔ گروه شغلی را وارد کنید.", nil)
			return
		}
		if calc.CompetencyLevel <= 0 {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"سطح شایستگی فرد (P2) را انتخاب کنید.", nil)
			return
		}
		if calc.PerformanceScore <= 0 {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"امتیاز عملکرد دوره (P3) را وارد کنید.", nil)
			return
		}
		if calc.CurrentPay <= 0 {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"پرداخت فعلی کارمند (پایهٔ حقوق آخرین حکم) را وارد کنید.", nil)
			return
		}
		result := rules.CompensationCalc(calc, params)
		out := optionsPayload(deps)
		out["result"] = result
		out["employeeId"] = calc.EmployeeID
		out["inputs"] = calc
		// نتیجهٔ محاسبه، خودش هیچ اثری در پرداخت ندارد: ثبت تصمیم و سپس (در صورت روشن
		// بودن پارامتر شرکت) صدور حکم، دو گام بعدی و اختیاری است.
		out["applyNoteFa"] = "این محاسبه فقط پیشنهاد است: تا تصمیم ثبت و تصویب نشود و پارامتر " +
			"«اعمال در پرداخت» شرکت روشن نباشد، هیچ تغییری در حکم یا فیش رخ نمی‌دهد."
		httpx.WriteJSON(w, http.StatusOK, out)
	}))

	// محاسبهٔ قیمت‌گذاری شغل (P1): جمع عامل‌های شغل → امتیاز، پلهٔ گروه شغلی و باند.
	mux.HandleFunc("POST /api/v1/compensation/job-pricing-calc", read(func(w http.ResponseWriter, r *http.Request) {
		var in jobPricingCalcRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ورودی محاسبهٔ قیمت‌گذاری شغل خوانده نشد.", nil)
			return
		}
		params := compensationParams(deps)
		points := in.Factors.Total()
		if points <= 0 {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"دست‌کم یک عامل شغل باید امتیاز داشته باشد.", nil)
			return
		}
		rank := rules.CompensationGradeFromScore(points, params)
		band := rules.CompensationBandFor(rank, params)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"evaluationPoints": points,
			"gradeRank":        rank,
			"gradeFa":          band.GradeFa,
			"band":             band,
			"lines": []rules.CompensationLine{
				{Key: "factors", TitleFa: "جمع عامل‌های شغل", ValueFa: rules.TrimFloatFa(points) + " امتیاز",
					RefFa: "ارزشیابی شغل — ماژول طبقه‌بندی مشاغل"},
				{Key: "grade", TitleFa: "پلهٔ گروه شغلی", ValueFa: "پلهٔ " + rules.IntFa(rank) + " («" + band.GradeFa + "»)",
					RefFa: "پارامتر شرکت compensation.job.pointsPerGrade"},
				{Key: "band", TitleFa: "باند پرداخت پله", ValueFa: "میانه " + rules.IntMoneyFa(band.Midpoint),
					RefFa: "پارامترهای compensation.band.*"},
			},
			"noteFa": "امتیاز شغل از جمع عامل‌های ارزشیابی می‌آید و پله و باند از پارامترهای مصوب شرکت ساخته می‌شود.",
		})
	}))
}

/* ------------------------------ بارهای پاسخ ------------------------------ */

// optionsPayload فهرست‌های مرجع، پارامترها، باندها و جدول ضریب‌های نظام ۴پی.
func optionsPayload(deps modulekit.Deps) map[string]any {
	params := compensationParams(deps)
	return map[string]any{
		"params":            params,
		"bands":             rules.CompensationBands(params),
		"factorTable":       rules.CompensationFactorTable(params),
		"grades":            rules.Grades(),
		"instruction":       rules.CompensationInstructionFa(),
		"jobFamilies":       domain.JobFamilies(),
		"competencyModels":  domain.CompetencyModels(),
		"competencyLevels":  domain.CompetencyLevels(),
		"performanceLevels": domain.PerformanceLevels(),
		"policyScopes":      domain.PolicyScopes(),
		"yesNo":             domain.YesNo(),
		// فهرستهای مرجع نظام جیب‌یست (خانوادهٔ امتیاز و منبع دادهٔ شاخص) در مسیر خودِ آن
		// نظام سرو می‌شود: /api/v1/compensation/g20-options؛ پاسخ ۴پی هیچ دادهٔ G20 ندارد.
		"today": calendar.Format(nowOf(deps)),
		// مرز مدل‌ها: مدل ۴پی مستقل و اختیاری است؛ این کلیدها همان چیزی است که رابط و سند
		// نشان می‌دهند تا کسی گمان نکند پرداخت جاری یا مدل صندوق نفت عوض شده است.
		"modelOptional":      !params.PayrollApply,
		"modelStatusFa":      modelStatusFa(params.PayrollApply),
		"independenceNoteFa": rules.CompensationIndependenceFa(params.PayrollApply),
		"noteFa":             "باندها و ضریب‌ها از پارامترهای مصوب شرکت ساخته می‌شوند؛ تغییر پارامتر، همهٔ محاسبه‌های بعدی را عوض می‌کند و تصمیم‌های ثبت‌شده دست‌نخورده می‌مانند.",
	}
}

// modelStatusFa وضعیت یک‌خطی مدل برای نشان بالای فرم‌ها و برگهٔ تصمیم.
func modelStatusFa(apply bool) string {
	if apply {
		return "مدل توسعه‌ای ۴پی — در این استقرار «اعمال در پرداخت» روشن است؛ فقط تصمیم‌های تازه می‌توانند مبنای حکم شوند"
	}
	return "مدل توسعه‌ای و اختیاری ۴پی — «اعمال در پرداخت» خاموش است؛ هیچ عددی از این مدل به حکم و فیش راه نمی‌یابد و مدل‌های پیشین دست‌نخورده می‌مانند"
}

// nowOf زمان جاری سامانه با ساعت قابل تزریق (برای آزمون‌های قطعی).
func nowOf(deps modulekit.Deps) time.Time {
	if deps.Clock != nil {
		return deps.Clock()
	}
	return time.Now()
}

// compensationParams پارامترهای مصوب شرکت برای نظام ۴پی.
//
// کلیدهای نظام جیب‌یست (`compensation.g20.*`) عمداً خوانده نمی‌شوند: دو نظام هیچ پارامتر
// مشترکی ندارند و خوانندهٔ پارامترهای ۴پی نباید حتی به کلیدهای نظام دیگر نگاه کند.
func compensationParams(deps modulekit.Deps) rules.CompensationParams {
	values := map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "compensation.") || strings.HasPrefix(p.Key, "compensation.g20.") {
			continue
		}
		values[p.Key] = p.Value
	}
	return rules.CompensationParamsFromText(values)
}

// compensationCalcRequest ورودی محاسبهٔ چهارعاملی از فرم.
type compensationCalcRequest struct {
	EmployeeID       string  `json:"employeeId"`
	EmployeeName     string  `json:"employeeName"`
	JobScore         float64 `json:"jobScore"`
	GradeOverride    int     `json:"gradeOverride"`
	CompetencyLevel  int     `json:"competencyLevel"`
	PerformanceScore float64 `json:"performanceScore"`
	CurrentPay       float64 `json:"currentPay"`
	MarketAdjust     bool    `json:"marketAdjust"`
}

// toInput ورودی فرم را به ورودی موتور قواعد تبدیل می‌کند.
func (in compensationCalcRequest) toInput() rules.CompensationInput {
	return rules.CompensationInput{
		EmployeeID:       strings.TrimSpace(in.EmployeeID),
		EmployeeName:     strings.TrimSpace(in.EmployeeName),
		JobScore:         in.JobScore,
		GradeOverride:    in.GradeOverride,
		CompetencyLevel:  in.CompetencyLevel,
		PerformanceScore: in.PerformanceScore,
		CurrentPay:       int64(in.CurrentPay),
		MarketAdjust:     in.MarketAdjust,
	}
}

// jobPricingCalcRequest ورودی قیمت‌گذاری شغل (امتیاز چهار عامل).
type jobPricingCalcRequest struct {
	Factors jobFactors `json:"factors"`
}

// jobFactors امتیاز چهار عامل ارزشیابی شغل.
type jobFactors struct {
	Knowledge         float64 `json:"knowledge"`
	ProblemSolving    float64 `json:"problemSolving"`
	Responsibility    float64 `json:"responsibility"`
	WorkingConditions float64 `json:"workingConditions"`
}

// Total جمع امتیاز عامل‌های شغل.
func (f jobFactors) Total() float64 {
	return f.Knowledge + f.ProblemSolving + f.Responsibility + f.WorkingConditions
}

/* --------------------------- پیش‌فرض‌های کارمند --------------------------- */

// employeeDefaults دادهٔ پیش‌فرض چهار عامل یک کارمند را از پرونده‌ها و حکم حقوق جمع می‌کند.
//
// قاعدهٔ اولویت‌ها اینجا نوشته نمی‌شود: نمرهٔ عملکرد از ماژول مدیریت عملکرد (نتیجهٔ
// نهایی) خوانده می‌شود و اگر نبود از رکورد ضریب عملکرد همین ماژول؛ سطح شایستگی و
// قیمت‌گذاری شغل از رکوردهای تأییدشدهٔ همین ماژول؛ و پرداخت فعلی از آخرین حکم
// ابلاغ‌شدهٔ کارمند (پورت قرارداد ماژول احکام).
func employeeDefaults(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, employeeID string) (map[string]any, error) {
	out := map[string]any{"employeeId": employeeID}
	if employeeID == "" {
		return out, nil
	}
	employee := map[string]any{
		"fullName": "", "position": "", "departmentName": "",
		"jobScore": 0.0, "gradeRank": 0, "competencyLevel": 0, "performanceScore": 0.0,
		"currentPay": 0, "currentPaySourceFa": "", "competencyLevelFa": "", "performanceLevelFa": "",
	}
	if reader, ok := deps.Ports.Employees(); ok {
		if emp, found, err := reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID); err == nil && found {
			employee["fullName"] = emp.FullName
			employee["position"] = emp.Position
			employee["departmentName"] = emp.DepartmentName
		}
	}

	// سطح شایستگی تأییدشدهٔ همین کارمند (تازه‌ترین).
	if rec, ok := latestEmployeeRecord(ctx, svc, domain.KindCompetencyLevel, employeeID); ok {
		level := int(numberValue(rec.Fields["level"]))
		employee["competencyLevel"] = level
		employee["competencyLevelFa"] = rules.CompensationLevelTitleFa(level)
	}

	// ضریب عملکرد: اول نتیجهٔ نهایی ماژول مدیریت عملکرد، بعد رکورد همین ماژول.
	if score, source, ok := performanceScoreOf(ctx, deps, svc, employeeID); ok {
		employee["performanceScore"] = score
		employee["performanceScoreSourceFa"] = source
		level := rules.CompensationPerformanceLevelOf(score, compensationParams(deps))
		employee["performanceLevelFa"] = level.TitleFa
	}

	// قیمت‌گذاری شغل: تطبیق عنوان پست کارمند با قیمت‌گذاری‌های مصوب.
	position, _ := employee["position"].(string)
	if rec, ok := jobPricingFor(ctx, svc, position); ok {
		employee["jobScore"] = numberValue(rec.Fields["evaluationPoints"])
		employee["gradeRank"] = int(numberValue(rec.Fields["gradeRank"]))
		employee["jobPricingId"] = rec.ID
	}

	// پرداخت فعلی از آخرین حکم ابلاغ‌شده.
	if reader, ok := deps.Ports.SalaryDecrees(); ok {
		if decree, found, err := reader.NotifiedDecreeOf(ctx, employeeID); err == nil && found {
			employee["currentPay"] = decree.BaseSalary
			employee["currentPaySourceFa"] = "پایهٔ حقوق حکم " + decree.ID + " («" + decree.StatusFa + "»)"
			employee["jobGroup"] = decree.JobGroup
		}
	}
	out["employee"] = employee

	// تصمیم‌های ثبت‌شدهٔ همین کارمند (برای نمایش سابقه در فرم).
	decisions, err := svc.List(ctx, contracts.RecordFilter{
		Module: "compensation", Kind: domain.KindCompensationDecision, EmployeeID: employeeID, Scope: "all", Size: 20,
	})
	if err == nil {
		out["decisions"] = decisions.Items
	}
	return out, nil
}

// latestEmployeeRecord تازه‌ترین رکورد یک نوع کارمندمحور را بر پایهٔ تاریخ اعتبار برمی‌گرداند.
func latestEmployeeRecord(ctx context.Context, svc *recordkit.Service, kind, employeeID string) (contracts.Record, bool) {
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "compensation", Kind: kind, EmployeeID: employeeID, Scope: "all", Size: 50,
	})
	if err != nil || len(page.Items) == 0 {
		return contracts.Record{}, false
	}
	records := page.Items
	sort.SliceStable(records, func(i, j int) bool {
		return recordDateValue(records[i]) < recordDateValue(records[j])
	})
	return records[len(records)-1], true
}

// recordDateValue تاریخ مؤثر رکورد برای مرتب‌سازی؛ اگر هیچ تاریخ فیلمی نبود، زمان آخرین
// رخداد گردشکار مبنای چیدن زمانی می‌شود.
func recordDateValue(rec contracts.Record) string {
	for _, key := range []string{"levelDate", "approvedDate", "effectiveDate", "pricingDate", "decisionDate"} {
		if value := strings.TrimSpace(rec.Fields[key]); value != "" {
			return calendar.NormalizeDigits(value)
		}
	}
	if n := len(rec.Events); n > 0 {
		return rec.Events[n-1].At
	}
	return ""
}

// performanceScoreOf امتیاز عملکرد کارمند: نتیجهٔ نهایی ماژول مدیریت عملکرد، وگرنه رکورد
// ضریب عملکرد همین ماژول.
func performanceScoreOf(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, employeeID string) (float64, string, bool) {
	if port, ok := deps.Ports.RecordList(performanceModuleID); ok {
		page, err := port.List(ctx, contracts.RecordFilter{
			Module: performanceModuleID, Kind: perfResultKind, EmployeeID: employeeID, Scope: "all", Size: 50,
		})
		if err == nil {
			best := 0.0
			for _, rec := range page.Items {
				if rec.Status != "final" {
					continue
				}
				if score := numberValue(rec.Fields["score"]); score > best {
					best = score
				}
			}
			if best > 0 {
				return best, "نتیجهٔ نهایی ارزیابی عملکرد (ماژول مدیریت عملکرد)", true
			}
		}
	}
	if rec, ok := latestEmployeeRecord(ctx, svc, domain.KindPerformanceFactor, employeeID); ok {
		if score := numberValue(rec.Fields["score"]); score > 0 {
			return score, "ضریب عملکرد ثبت‌شده در همین ماژول (" + rec.ID + ")", true
		}
	}
	return 0, "", false
}

// jobPricingFor قیمت‌گذاری مصوب یک پست را با عنوان پست پیدا می‌کند.
func jobPricingFor(ctx context.Context, svc *recordkit.Service, position string) (contracts.Record, bool) {
	title := strings.TrimSpace(position)
	if title == "" {
		return contracts.Record{}, false
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "compensation", Kind: domain.KindJobPricing, Scope: "all", Size: 200,
	})
	if err != nil {
		return contracts.Record{}, false
	}
	for _, rec := range page.Items {
		if rec.Status == "archived" {
			continue
		}
		if strings.TrimSpace(rec.Fields["positionTitle"]) == title {
			return rec, true
		}
	}
	return contracts.Record{}, false
}

// numberValue مقدار عددی یک فیلد رشته‌ای (با ارقام فارسی و جداکنندهٔ هزارگان).
func numberValue(raw string) float64 {
	text := strings.TrimSpace(calendar.NormalizeDigits(raw))
	if text == "" {
		return 0
	}
	text = strings.NewReplacer(",", "", "٬", "", "٪", "", "%", "").Replace(text)
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0
	}
	return value
}

// numberFromAny عدد را از هر نوع واسط (JSON) بیرون می‌کشد.
func numberFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		return numberValue(v)
	default:
		return 0
	}
}
