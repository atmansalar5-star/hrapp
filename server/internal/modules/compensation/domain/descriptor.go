// Package domain دامنهٔ ماژول جبران خدمات ۴پی.
//
// شش نوع رکورد، هر یک یک «پ» از نظام جبران خدمات:
//
//	compensationModel    معماری و پارامترهای مصوب بستهٔ پرداخت            (چهارچوب ۴پی)
//	jobPricing           قیمت‌گذاری شغل: امتیاز ارزشیابی → پله و باند      (P1 — شغل)
//	competencyLevel      سطح شایستگی و صلاحیت فرد                       (P2 — شخص)
//	performanceFactor    ضریب عملکرد دورهٔ ارزیابی                      (P3 — عملکرد)
//	payBand              باند و پله‌های پرداخت هر گروه شغلی              (P4 — پرداخت)
//	compensationDecision تصمیم چهارعاملی برای یک کارمند (افزایش/تثبیت)   (جمع‌بندی ۴پی)
//
// قاعدهٔ محاسبه در موتور قواعد (`internal/rules/compensation.go`) است و این بسته فقط
// «شکل داده، گردشکار و اعتبارسنجی» را اعلام می‌کند؛ پس هیچ عددی در رابط کاربری نیست.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// کلیدهای نوع رکورد این ماژول.
const (
	// KindCompensationModel مدل و پارامترهای مصوب بستهٔ جبران خدمات.
	KindCompensationModel = "compensationModel"
	// KindJobPricing قیمت‌گذاری شغل (P1): تبدیل امتیاز ارزشیابی شغل به پلهٔ گروه شغلی و
	// باند پرداخت. سنجش عامل‌های شغل در ماژول «طبقه‌بندی مشاغل» انجام می‌شود و اینجا فقط
	// امتیاز مصوب به پله و باند نگاشت می‌شود تا یک شغل دو ارزشیابی موازی نداشته باشد.
	KindJobPricing = "jobPricing"
	// KindCompetencyLevel سطح شایستگی فرد (P2).
	KindCompetencyLevel = "competencyLevel"
	// KindPerformanceFactor ضریب عملکرد دوره (P3).
	KindPerformanceFactor = "performanceFactor"
	// KindPayBand باند و پله‌های پرداخت (P4).
	KindPayBand = "payBand"
	// KindCompensationDecision تصمیم چهارعاملی برای یک کارمند.
	KindCompensationDecision = "compensationDecision"
)

// Descriptor شناسهٔ ماژول جبران خدمات ۴پی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "compensation",
		TitleFa: "جبران خدمات ۴پی",
		Kinds: []recordkit.Kind{
			compensationModelKind(),
			jobPricingKind(),
			competencyLevelKind(),
			performanceFactorKind(),
			payBandKind(),
			compensationDecisionKind(),
		},
		Summaries: []string{
			"نظام جبران خدمات چهارعاملی: شغل (P1)، شخص (P2)، عملکرد (P3) و پرداخت (P4).",
			"باند و پلهٔ پرداخت هر گروه شغلی از پارامترهای مصوب شرکت ساخته می‌شود.",
			"تصمیم چهارعاملی، پرداخت هدف و افزایش پیشنهادی هر کارمند را با شرح خط‌به‌خط ثبت می‌کند.",
		},
	}
}

// compensationModelKind مدل مصوب بستهٔ جبران خدمات: چهارچوبی که همهٔ محاسبه‌ها از آن
// پارامتر می‌گیرند. چرا رکورد و نه فقط پارامتر: پارامترها «مقدار»اند و این مدل «سند
// مصوب» سازمان است — با تاریخ اجرا، دامنهٔ اجرا و گردشکار تصویب.
func compensationModelKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindCompensationModel, TitleFa: "مدل جبران خدمات ۴پی", IDPrefix: "CMOD",
		Fields: recordkit.Fields(
			"modelName|نام مدل|text|1",
			"policyScope|دامنهٔ اجرا|select|1",
			"effectiveDate|تاریخ اجرا|date|1",
			"bandBaseMidpoint|میانهٔ باند پلهٔ ۱ (ریال)|money|1",
			"bandSlopePercent|شیب میانهٔ باند بین پله‌ها (٪)|number|1",
			"bandSpreadPercent|گسترهٔ باند: کف تا سقف (٪)|number|1",
			"bandStepPercent|گام پله درون باند (٪)|number|1",
			"jobPointsPerGrade|امتیاز ارزشیابی شغل برای هر پله|number|1",
			"personMinFactor|کف ضریب شایستگی فرد|number|1",
			"personMaxFactor|سقف ضریب شایستگی فرد|number|1",
			"perfFactorExcellent|ضریب عملکرد «عالی»|number|1",
			"perfFactorGood|ضریب عملکرد «خوب»|number|1",
			"perfFactorMeets|ضریب عملکرد «قابل قبول»|number|1",
			"perfFactorBelow|ضریب عملکرد «نیازمند بهبود»|number|1",
			"marketAdjustPercent|تعدیل سالانهٔ بازار (٪)|number|0",
			"maxAnnualIncreasePercent|سقف افزایش سالانهٔ هر کارمند (٪)|number|1",
			"payrollApply|اعمال در حکم و پرداخت|select|1",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "approved|مصوب", "active|جاری", "archived|بایگانی‌شده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|approved", "approved|active,draft", "active|archived",
		),
		Actions: recordkit.Actions(
			"approve|تصویب مدل|primary|approved|0",
			"activate|جاری‌سازی مدل|primary|active|0",
			"archive|بایگانی مدل|ghost|archived|1",
		),
		Validate:         validateCompensationModel,
		EditableStatuses: []string{"draft", "approved", "active"},
	}
}

// jobPricingKind قیمت‌گذاری شغل (P1): امتیاز ارزشیابی شغل به پلهٔ گروه شغلی و باند پرداخت
// نگاشت می‌شود. امتیاز ارزشیابی در ماژول «طبقه‌بندی مشاغل» سنجیده می‌شود و اینجا فقط
// «قیمت» شغل تعیین می‌شود؛ پس یک شغل دو ارزشیابی موازی ندارد.
//
// چرا امتیاز و پلهٔ حاصل هر دو ذخیره می‌شوند: تصمیم‌های گذشته باید با همان عدد زمان
// تصویب بازخوانی شوند؛ اگر پارامتر «امتیاز هر پله» بعداً عوض شود، قیمت‌گذاری‌های پیشین
// تغییر نمی‌کنند.
func jobPricingKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindJobPricing, TitleFa: "قیمت‌گذاری شغل (P1)", IDPrefix: "CJPR",
		Fields: recordkit.Fields(
			"positionCode|کد پست|text|1",
			"positionTitle|عنوان پست|text|1",
			"jobFamily|خانوادهٔ شغلی|select|1",
			"evaluationPoints|امتیاز ارزشیابی مصوب شغل|number|1",
			"factorKnowledge|عامل دانش و مهارت (امتیاز)|number|0",
			"factorProblemSolving|عامل حل مسئله و پیچیدگی (امتیاز)|number|0",
			"factorResponsibility|عامل مسئولیت و اثر تصمیم (امتیاز)|number|0",
			"factorWorkingConditions|عامل شرایط کار (امتیاز)|number|0",
			"gradeRank|پلهٔ گروه شغلی|number|1",
			"bandMidpoint|میانهٔ باند پله (ریال)|money|0",
			"pricingDate|تاریخ قیمت‌گذاری|date|1",
			"approvalNo|شمارهٔ مصوبهٔ قیمت‌گذاری|text|0",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "reviewed|بررسی کمیته", "approved|مصوب", "archived|بایگانی‌شده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|reviewed", "reviewed|approved,draft", "approved|archived",
		),
		Actions: recordkit.Actions(
			"review|ارسال به کمیتهٔ ارزشیابی|primary|reviewed|0",
			"approve|تصویب قیمت‌گذاری|primary|approved|0",
			"archive|بایگانی|ghost|archived|1",
		),
		Validate:         validateJobPricing,
		EditableStatuses: []string{"draft", "reviewed"},
		HistoryDate:      "pricingDate",
	}
}

// competencyLevelKind سطح شایستگی فرد (P2). کارمندمحور است: هر کارمند سطح جاری خودش را
// دارد و سابقهٔ سطح‌ها با تاریخ تعیین سطح مرتب می‌شود.
func competencyLevelKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindCompetencyLevel, TitleFa: "سطح شایستگی فرد (P2)", IDPrefix: "CCMP",
		EmployeeScoped: true,
		Fields: recordkit.Fields(
			"competencyModel|مدل/خانوادهٔ شغلی شایستگی|select|1",
			"level|سطح شایستگی|select|1",
			"levelDate|تاریخ تعیین سطح|date|1",
			"levelScore|امتیاز سنجش (۰ تا ۱۰۰)|number|0",
			"assessorName|ارزیاب|text|0",
			"evidence|مستندات سنجش|text|0",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "confirmed|تأییدشده", "expired|منقضی",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|confirmed", "confirmed|expired",
		),
		Actions: recordkit.Actions(
			"confirm|تأیید سطح شایستگی|primary|confirmed|0",
			"expire|پایان اعتبار سطح|ghost|expired|1",
		),
		Validate:         validateCompetencyLevel,
		EditableStatuses: []string{"draft", "confirmed"},
		HistoryDate:      "levelDate",
	}
}

// performanceFactorKind ضریب عملکرد دوره (P3): امتیاز نهایی ارزیابی و ضریب پرداخت متغیر.
func performanceFactorKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindPerformanceFactor, TitleFa: "ضریب عملکرد دوره (P3)", IDPrefix: "CPRF",
		EmployeeScoped: true,
		Fields: recordkit.Fields(
			"cycleTitle|عنوان دورهٔ ارزیابی|text|1",
			"cycleYear|سال دوره|number|1",
			"score|امتیاز نهایی عملکرد (۰ تا ۱۰۰)|number|1",
			"level|سطح عملکرد|select|1",
			"factor|ضریب عملکرد دوره|number|1",
			"managerName|ارزیاب/مدیر واحد|text|0",
			"approvedDate|تاریخ تأیید نتیجه|date|0",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "approved|تأییدشده", "closed|بسته‌شده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|approved", "approved|closed",
		),
		Actions: recordkit.Actions(
			"approve|تأیید ضریب عملکرد|primary|approved|0",
			"close|بستن دوره|ghost|closed|0",
		),
		Validate:         validatePerformanceFactor,
		EditableStatuses: []string{"draft", "approved"},
		HistoryDate:      "approvedDate",
	}
}

// payBandKind باند و پله‌های پرداخت (P4): کف، میانه، سقف و گام پلهٔ هر گروه شغلی.
func payBandKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindPayBand, TitleFa: "باند و پله‌های پرداخت (P4)", IDPrefix: "CBND",
		Fields: recordkit.Fields(
			"modelName|مدل جبران خدمات|text|0",
			"gradeRank|پلهٔ گروه شغلی|number|1",
			"gradeTitle|عنوان گروه شغلی|text|1",
			"minPay|کف باند (ریال)|money|1",
			"midPay|میانهٔ باند (ریال)|money|1",
			"maxPay|سقف باند (ریال)|money|1",
			"stepAmount|مبلغ گام پله (ریال)|money|1",
			"effectiveDate|تاریخ اجرا|date|1",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "approved|مصوب", "active|جاری", "archived|بایگانی‌شده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|approved", "approved|active,draft", "active|archived",
		),
		Actions: recordkit.Actions(
			"approve|تصویب باند|primary|approved|0",
			"activate|جاری‌سازی باند|primary|active|0",
			"archive|بایگانی باند|ghost|archived|1",
		),
		Validate:         validatePayBand,
		EditableStatuses: []string{"draft", "approved", "active"},
		HistoryDate:      "effectiveDate",
	}
}

// compensationDecisionKind تصمیم چهارعاملی یک کارمند: خروجی رسمی نظام ۴پی که مبنای صدور
// حکم حقوقی است. اعداد محاسبه‌شده در خود رکورد می‌مانند تا سابقهٔ تصمیم‌ها پایدار بماند.
func compensationDecisionKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindCompensationDecision, TitleFa: "تصمیم چهارعاملی جبران خدمات", IDPrefix: "CDEC",
		EmployeeScoped: true,
		Fields: recordkit.Fields(
			"decisionNo|شمارهٔ تصمیم|text|1",
			"decisionDate|تاریخ تصمیم|date|1",
			"effectiveDate|تاریخ اجرا|date|1",
			"jobScore|امتیاز ارزشیابی شغل (P1)|number|1",
			"gradeRank|پلهٔ گروه شغلی|number|1",
			"competencyLevel|سطح شایستگی (P2)|select|1",
			"performanceScore|امتیاز عملکرد (P3)|number|1",
			"currentPay|پرداخت فعلی ماهانه (ریال)|money|1",
			"targetPay|پرداخت هدف ماهانه (ریال)|money|1",
			"increaseAmount|افزایش مصوب ماهانه (ریال)|money|1",
			"compaRatioPercent|نسبت پرداخت به میانهٔ باند (٪)|number|0",
			"marketAdjust|تعدیل بازار|select|0",
			"decisionFa|خلاصهٔ تصمیم و مبنا|text|1",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "approved|مصوب", "applied|اجراشده", "cancelled|لغوشده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|approved,cancelled", "approved|applied,cancelled", "applied|",
		),
		Actions: recordkit.Actions(
			"approve|تصویب تصمیم|primary|approved|0",
			"apply|اعلام برای صدور حکم|primary|applied|0",
			"cancel|لغو تصمیم|danger|cancelled|1",
		),
		Validate:         validateCompensationDecision,
		EditableStatuses: []string{"draft", "approved"},
		HistoryDate:      "effectiveDate",
	}
}

/* ------------------------------ فهرست‌های مرجع ------------------------------ */

// CompensationPolicyScope دامنهٔ اجرای مدل جبران خدمات.
type CompensationPolicyScope struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// PolicyScopes دامنه‌های اجرای مدل (کلیدهای لاتین، عنوان فارسی).
func PolicyScopes() []CompensationPolicyScope {
	return []CompensationPolicyScope{
		{Key: "organization", TitleFa: "کل سازمان"},
		{Key: "unit", TitleFa: "یک واحد سازمانی"},
		{Key: "job_family", TitleFa: "یک خانوادهٔ شغلی"},
		{Key: "management", TitleFa: "گروه‌های مدیریتی"},
	}
}

// PolicyScopeTitleFa عنوان فارسی دامنهٔ اجرا؛ دامنهٔ ناشناخته خودِ کلید را برمی‌گرداند.
func PolicyScopeTitleFa(key string) string {
	for _, item := range PolicyScopes() {
		if item.Key == key {
			return item.TitleFa
		}
	}
	return key
}

// JobFamily خانوادهٔ شغلی برای طبقه‌بندی ارزشیابی شغل.
type JobFamily struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// JobFamilies خانواده‌های شغلی این استقرار (پایهٔ گروه‌بندی ارزشیابی شغل).
func JobFamilies() []JobFamily {
	return []JobFamily{
		{Key: "operations", TitleFa: "بهره‌برداری و تولید"},
		{Key: "maintenance", TitleFa: "تعمیرات و نگهداری"},
		{Key: "engineering", TitleFa: "مهندسی و فنی"},
		{Key: "hse", TitleFa: "بهداشت، ایمنی و محیط زیست"},
		{Key: "support", TitleFa: "پشتیبانی و خدمات"},
		{Key: "finance", TitleFa: "مالی و بازرگانی"},
		{Key: "hr", TitleFa: "منابع انسانی"},
		{Key: "it", TitleFa: "فناوری اطلاعات"},
	}
}

// ReferenceOption یک گزینهٔ فهرست مرجع (کلید و عنوان فارسی).
type ReferenceOption struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// CompetencyLevels سطوح مدل شایستگی (P2) — عنوان‌ها از موتور قواعد می‌آید تا محاسبه و
// نمایش یک منبع داشته باشند.
func CompetencyLevels() []ReferenceOption {
	params := rules.DefaultCompensationParams()
	out := make([]ReferenceOption, 0, params.PersonLevels)
	for level := 1; level <= params.PersonLevels; level++ {
		out = append(out, ReferenceOption{Key: strconv.Itoa(level), TitleFa: rules.CompensationLevelTitleFa(level)})
	}
	return out
}

// PerformanceLevels سطوح عملکرد (P3) با ضریب هر سطح.
func PerformanceLevels() []ReferenceOption {
	params := rules.DefaultCompensationParams()
	return []ReferenceOption{
		{Key: "excellent", TitleFa: "عالی (امتیاز " + rules.IntFa(int(params.PerfExcellentScore)) + " و بالاتر)"},
		{Key: "good", TitleFa: "خوب (امتیاز " + rules.IntFa(int(params.PerfGoodScore)) + " و بالاتر)"},
		{Key: "meets", TitleFa: "قابل قبول (امتیاز " + rules.IntFa(int(params.PerfMeetsScore)) + " و بالاتر)"},
		{Key: "below", TitleFa: "نیازمند بهبود"},
	}
}

// YesNo فهرست بله/خیر برای فیلدهای انتخابی.
func YesNo() []ReferenceOption {
	return []ReferenceOption{{Key: "yes", TitleFa: "بله"}, {Key: "no", TitleFa: "خیر"}}
}

// CompetencyModels مدل‌های شایستگی (خانواده‌های شغلی) برای فیلد انتخابی P2.
func CompetencyModels() []ReferenceOption {
	families := JobFamilies()
	out := make([]ReferenceOption, 0, len(families))
	for _, f := range families {
		out = append(out, ReferenceOption{Key: f.Key, TitleFa: f.TitleFa})
	}
	return out
}

// PerformanceLevelOf کلید سطح عملکرد از امتیاز؛ متن فارسی سطح از موتور قواعد می‌آید.
func PerformanceLevelOf(score float64) string {
	return rules.CompensationPerformanceLevelOf(score, rules.DefaultCompensationParams()).Key
}

/* ------------------------------ اعتبارسنجی‌ها ------------------------------ */

// numberValue مقدار عددی یک فیلد را برمی‌گرداند؛ مقدار نامعتبر صفر می‌شود.
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

// intField مقدار عددی صحیح را برای ذخیره در فیلد رشته‌ای برمی‌گرداند.
func intField(v int) string { return strconv.Itoa(v) }

// validateCompensationModel قاعده‌های مدل: پارامترها باید در بازهٔ معقول و سازگار باشند؛
// مدلی که باند بی‌معنا می‌سازد نباید تصویب شود.
func validateCompensationModel(in contracts.RecordInput) error {
	f := in.Fields
	if strings.TrimSpace(f["modelName"]) == "" {
		return &recordkit.RuleError{Message: "نام مدل جبران خدمات را بنویسید."}
	}
	known := false
	for _, scope := range PolicyScopes() {
		if scope.Key == strings.TrimSpace(f["policyScope"]) {
			known = true
			break
		}
	}
	if !known {
		return &recordkit.RuleError{Message: "دامنهٔ اجرای مدل را از فهرست انتخاب کنید."}
	}
	if raw := strings.TrimSpace(f["effectiveDate"]); raw != "" {
		if _, err := calendar.Parse(calendar.NormalizeDigits(raw)); err != nil {
			return &recordkit.RuleError{Message: "تاریخ اجرای مدل معتبر نیست (قالب YYYY/MM/DD)."}
		}
	} else {
		return &recordkit.RuleError{Message: "تاریخ اجرای مدل را وارد کنید."}
	}
	if numberValue(f["bandBaseMidpoint"]) <= 0 {
		return &recordkit.RuleError{Message: "میانهٔ باند پلهٔ ۱ باید بزرگ‌تر از صفر باشد."}
	}
	if v := numberValue(f["bandSlopePercent"]); v < 0 || v > 100 {
		return &recordkit.RuleError{Message: "شیب میانهٔ باند باید بین ۰ و ۱۰۰ درصد باشد."}
	}
	if v := numberValue(f["bandSpreadPercent"]); v < 0 || v > 200 {
		return &recordkit.RuleError{Message: "گسترهٔ باند باید بین ۰ و ۲۰۰ درصد باشد."}
	}
	if v := numberValue(f["bandStepPercent"]); v < 0 || v > 50 {
		return &recordkit.RuleError{Message: "گام پله باید بین ۰ و ۵۰ درصد میانهٔ باند باشد."}
	}
	if numberValue(f["jobPointsPerGrade"]) <= 0 {
		return &recordkit.RuleError{Message: "امتیاز ارزشیابی شغل برای هر پله باید بزرگ‌تر از صفر باشد."}
	}
	minFactor := numberValue(f["personMinFactor"])
	maxFactor := numberValue(f["personMaxFactor"])
	if minFactor <= 0 || maxFactor <= 0 {
		return &recordkit.RuleError{Message: "ضرایب شایستگی باید بزرگ‌تر از صفر باشند."}
	}
	if minFactor > maxFactor {
		return &recordkit.RuleError{Message: "کف ضریب شایستگی نباید از سقف آن بزرگ‌تر باشد."}
	}
	if maxFactor > 2 {
		return &recordkit.RuleError{Message: "سقف ضریب شایستگی بیش از ۲ برابر مجاز نیست."}
	}
	for _, key := range []string{"perfFactorExcellent", "perfFactorGood", "perfFactorMeets", "perfFactorBelow"} {
		v := numberValue(f[key])
		if v <= 0 || v > 2 {
			return &recordkit.RuleError{Message: "ضرایب عملکرد باید بین ۰ و ۲ باشند."}
		}
	}
	if v := numberValue(f["marketAdjustPercent"]); v < 0 || v > 100 {
		return &recordkit.RuleError{Message: "تعدیل بازار باید بین ۰ و ۱۰۰ درصد باشد."}
	}
	if v := numberValue(f["maxAnnualIncreasePercent"]); v < 0 || v > 100 {
		return &recordkit.RuleError{Message: "سقف افزایش سالانه باید بین ۰ و ۱۰۰ درصد باشد."}
	}
	if v := strings.TrimSpace(f["payrollApply"]); v != "" && v != "yes" && v != "no" {
		return &recordkit.RuleError{Message: "انتخاب «اعمال در پرداخت» باید بله یا خیر باشد."}
	}
	return nil
}

// validateJobPricing قاعده‌های قیمت‌گذاری شغل: امتیاز مصوب مثبت، عوامل شغل اختیاری ولی
// سازگار با امتیاز، و پلهٔ گروه شغلی در نردبان سازمان باشد.
func validateJobPricing(in contracts.RecordInput) error {
	f := in.Fields
	if strings.TrimSpace(f["positionTitle"]) == "" {
		return &recordkit.RuleError{Message: "عنوان پست را بنویسید."}
	}
	known := false
	for _, family := range JobFamilies() {
		if family.Key == strings.TrimSpace(f["jobFamily"]) {
			known = true
			break
		}
	}
	if !known {
		return &recordkit.RuleError{Message: "خانوادهٔ شغلی را از فهرست انتخاب کنید."}
	}
	points := numberValue(f["evaluationPoints"])
	if points <= 0 {
		return &recordkit.RuleError{Message: "امتیاز ارزشیابی مصوب شغل را وارد کنید."}
	}
	// عامل‌های شغل اختیاری‌اند؛ اگر وارد شده باشند، جمعشان باید همان امتیاز مصوب باشد تا
	// قیمت‌گذاری با ارزشیابی ماژول طبقه‌بندی مشاغل ناهمخوان نشود.
	total := 0.0
	hasFactors := false
	for _, key := range []string{"factorKnowledge", "factorProblemSolving", "factorResponsibility", "factorWorkingConditions"} {
		v := numberValue(f[key])
		if v < 0 || v > 100 {
			return &recordkit.RuleError{Message: "امتیاز هر عامل شغل باید بین ۰ و ۱۰۰ باشد."}
		}
		if v > 0 {
			hasFactors = true
		}
		total += v
	}
	if hasFactors {
		if diff := points - total; diff > 0.01 || diff < -0.01 {
			return &recordkit.RuleError{Message: "امتیاز مصوب با جمع عامل‌های شغل همخوان نیست؛ از دکمهٔ محاسبه استفاده کنید."}
		}
	}
	rank := int(numberValue(f["gradeRank"]))
	if rank < 1 || rank > len(rules.Grades()) {
		return &recordkit.RuleError{Message: "پلهٔ گروه شغلی باید بین ۱ و " + rules.IntFa(len(rules.Grades())) + " باشد."}
	}
	if raw := strings.TrimSpace(f["pricingDate"]); raw != "" {
		if _, err := calendar.Parse(calendar.NormalizeDigits(raw)); err != nil {
			return &recordkit.RuleError{Message: "تاریخ قیمت‌گذاری معتبر نیست (قالب YYYY/MM/DD)."}
		}
	} else {
		return &recordkit.RuleError{Message: "تاریخ قیمت‌گذاری شغل را وارد کنید."}
	}
	return nil
}

// validateCompetencyLevel قاعده‌های سطح شایستگی (P2).
func validateCompetencyLevel(in contracts.RecordInput) error {
	f := in.Fields
	if strings.TrimSpace(f["competencyModel"]) == "" {
		return &recordkit.RuleError{Message: "مدل یا خانوادهٔ شغلی شایستگی را انتخاب کنید."}
	}
	level := int(numberValue(f["level"]))
	if level < 1 || level > rules.DefaultCompensationParams().PersonLevels {
		return &recordkit.RuleError{Message: "سطح شایستگی باید بین ۱ و " +
			rules.IntFa(rules.DefaultCompensationParams().PersonLevels) + " باشد."}
	}
	if raw := strings.TrimSpace(f["levelDate"]); raw != "" {
		if _, err := calendar.Parse(calendar.NormalizeDigits(raw)); err != nil {
			return &recordkit.RuleError{Message: "تاریخ تعیین سطح معتبر نیست (قالب YYYY/MM/DD)."}
		}
	} else {
		return &recordkit.RuleError{Message: "تاریخ تعیین سطح شایستگی را وارد کنید."}
	}
	if v := numberValue(f["levelScore"]); v < 0 || v > 100 {
		return &recordkit.RuleError{Message: "امتیاز سنجش شایستگی باید بین ۰ و ۱۰۰ باشد."}
	}
	return nil
}

// validatePerformanceFactor قاعده‌های ضریب عملکرد (P3): امتیاز در بازه، سطح با امتیاز
// همخوان و ضریب همان سطح.
func validatePerformanceFactor(in contracts.RecordInput) error {
	f := in.Fields
	if strings.TrimSpace(f["cycleTitle"]) == "" {
		return &recordkit.RuleError{Message: "عنوان دورهٔ ارزیابی را بنویسید."}
	}
	if year := numberValue(f["cycleYear"]); year < 1300 || year > 1500 {
		return &recordkit.RuleError{Message: "سال دورهٔ ارزیابی معتبر نیست."}
	}
	score := numberValue(f["score"])
	if score < 0 || score > 100 {
		return &recordkit.RuleError{Message: "امتیاز عملکرد باید بین ۰ و ۱۰۰ باشد."}
	}
	expected := PerformanceLevelOf(score)
	if value := strings.TrimSpace(f["level"]); value != "" && value != expected {
		return &recordkit.RuleError{Message: "سطح عملکرد با امتیاز واردشده همخوان نیست؛ سطح درست: " + expected + "."}
	}
	factor := numberValue(f["factor"])
	if factor <= 0 || factor > 2 {
		return &recordkit.RuleError{Message: "ضریب عملکرد باید بین ۰ و ۲ باشد."}
	}
	return nil
}

// validatePayBand قاعده‌های باند پرداخت (P4): کف < میانه < سقف و پلهٔ معتبر.
func validatePayBand(in contracts.RecordInput) error {
	f := in.Fields
	rank := int(numberValue(f["gradeRank"]))
	if rank < 1 || rank > len(rules.Grades()) {
		return &recordkit.RuleError{Message: "پلهٔ گروه شغلی باید بین ۱ و " + rules.IntFa(len(rules.Grades())) + " باشد."}
	}
	if strings.TrimSpace(f["gradeTitle"]) == "" {
		return &recordkit.RuleError{Message: "عنوان گروه شغلی را بنویسید."}
	}
	minPay := numberValue(f["minPay"])
	midPay := numberValue(f["midPay"])
	maxPay := numberValue(f["maxPay"])
	if minPay <= 0 || midPay <= 0 || maxPay <= 0 {
		return &recordkit.RuleError{Message: "کف، میانه و سقف باند باید بزرگ‌تر از صفر باشند."}
	}
	if !(minPay < midPay && midPay < maxPay) {
		return &recordkit.RuleError{Message: "ترتیب باند باید کف < میانه < سقف باشد."}
	}
	if step := numberValue(f["stepAmount"]); step <= 0 {
		return &recordkit.RuleError{Message: "مبلغ گام پله باید بزرگ‌تر از صفر باشد."}
	}
	if raw := strings.TrimSpace(f["effectiveDate"]); raw != "" {
		if _, err := calendar.Parse(calendar.NormalizeDigits(raw)); err != nil {
			return &recordkit.RuleError{Message: "تاریخ اجرای باند معتبر نیست (قالب YYYY/MM/DD)."}
		}
	} else {
		return &recordkit.RuleError{Message: "تاریخ اجرای باند را وارد کنید."}
	}
	return nil
}

// validateCompensationDecision قاعده‌های تصمیم چهارعاملی: شماره و مبنا نوشته شده، پله و
// سطح معتبر، پرداخت هدف داخل باند همان پله و افزایش از تفاوت هدف و پرداخت فعلی بیشتر نباشد.
func validateCompensationDecision(in contracts.RecordInput) error {
	f := in.Fields
	if strings.TrimSpace(f["decisionNo"]) == "" {
		return &recordkit.RuleError{Message: "شمارهٔ تصمیم را بنویسید."}
	}
	if strings.TrimSpace(f["decisionFa"]) == "" {
		return &recordkit.RuleError{Message: "خلاصهٔ تصمیم و مبنای آن را بنویسید."}
	}
	for _, key := range []string{"decisionDate", "effectiveDate"} {
		raw := strings.TrimSpace(f[key])
		if raw == "" {
			return &recordkit.RuleError{Message: "تاریخ تصمیم و تاریخ اجرا الزامی است."}
		}
		if _, err := calendar.Parse(calendar.NormalizeDigits(raw)); err != nil {
			return &recordkit.RuleError{Message: "تاریخ «" + key + "» معتبر نیست (قالب YYYY/MM/DD)."}
		}
	}
	rank := int(numberValue(f["gradeRank"]))
	if rank < 1 || rank > len(rules.Grades()) {
		return &recordkit.RuleError{Message: "پلهٔ گروه شغلی باید بین ۱ و " + rules.IntFa(len(rules.Grades())) + " باشد."}
	}
	level := int(numberValue(f["competencyLevel"]))
	if level < 1 || level > rules.DefaultCompensationParams().PersonLevels {
		return &recordkit.RuleError{Message: "سطح شایستگی باید بین ۱ و " +
			rules.IntFa(rules.DefaultCompensationParams().PersonLevels) + " باشد."}
	}
	if score := numberValue(f["performanceScore"]); score < 0 || score > 100 {
		return &recordkit.RuleError{Message: "امتیاز عملکرد باید بین ۰ و ۱۰۰ باشد."}
	}
	current := numberValue(f["currentPay"])
	target := numberValue(f["targetPay"])
	increase := numberValue(f["increaseAmount"])
	if current <= 0 || target <= 0 {
		return &recordkit.RuleError{Message: "پرداخت فعلی و پرداخت هدف باید بزرگ‌تر از صفر باشند."}
	}
	if target < current && increase > 0 {
		return &recordkit.RuleError{Message: "پرداخت هدف از پرداخت فعلی کمتر است؛ افزایش را صفر بگذارید."}
	}
	if increase > target-current+0.01 {
		return &recordkit.RuleError{Message: "افزایش مصوب نباید از تفاوت پرداخت هدف و پرداخت فعلی بیشتر باشد."}
	}
	if value := strings.TrimSpace(f["marketAdjust"]); value != "" && value != "yes" && value != "no" {
		return &recordkit.RuleError{Message: "انتخاب «تعدیل بازار» باید بله یا خیر باشد."}
	}
	return nil
}

/* --------------------------------- بذر نمونه --------------------------------- */

// Seed دادهٔ نمونهٔ ماژول: یک مدل جاری، شش باند پرداخت، سه ارزشیابی شغل، سه سطح
// شایستگی، سه ضریب عملکرد و یک تصمیم چهارعاملی نمونه.
//
// چرا نمونه لازم است: کسی که تازه سامانه را بالا می‌آورد باید ببیند باندها چطور از
// پارامترها ساخته می‌شوند و تصمیم چهارعاملی چه شکلی دارد؛ این داده فقط در نصب خالی
// ساخته می‌شود و دادهٔ واقعی سازمان را بازنویسی نمی‌کند.
func Seed() []contracts.RecordInput {
	model := contracts.RecordInput{
		Kind: KindCompensationModel, Title: "مدل جبران خدمات ۴پی — استقرار ۱۴۰۵",
		NoteFa: "مدل پایهٔ استقرار: باندهای پرداخت از همین پارامترها ساخته می‌شود",
		Fields: map[string]string{
			"modelName":                "مدل چهارعاملی ۱۴۰۵",
			"policyScope":              "organization",
			"effectiveDate":            "1405/01/01",
			"bandBaseMidpoint":         "200000000",
			"bandSlopePercent":         "18",
			"bandSpreadPercent":        "40",
			"bandStepPercent":          "3",
			"jobPointsPerGrade":        "60",
			"personMinFactor":          "0.9",
			"personMaxFactor":          "1.1",
			"perfFactorExcellent":      "1.15",
			"perfFactorGood":           "1.08",
			"perfFactorMeets":          "1",
			"perfFactorBelow":          "0.92",
			"marketAdjustPercent":      "10",
			"maxAnnualIncreasePercent": "25",
			"payrollApply":             "yes",
		},
		SeedStatus: "active",
	}

	// باندهای پرداخت همان مدل: از پارامترهای پیش‌فرض موتور ساخته می‌شوند تا نمونه با
	// پارامترها یک‌سان باشد و دستی نوشته نشود.
	params := rules.DefaultCompensationParams()
	bands := make([]contracts.RecordInput, 0, params.JobMaxGrade)
	for _, band := range rules.CompensationBands(params) {
		bands = append(bands, contracts.RecordInput{
			Kind: KindPayBand, Title: "باند پرداخت پلهٔ " + rules.IntFa(band.Rank) + " — " + band.GradeFa,
			NoteFa: "ساخته‌شده از مدل جاری و پارامترهای شرکت",
			Fields: map[string]string{
				"modelName":     "مدل چهارعاملی ۱۴۰۵",
				"gradeRank":     intField(band.Rank),
				"gradeTitle":    band.GradeFa,
				"minPay":        strconv.FormatInt(band.Min, 10),
				"midPay":        strconv.FormatInt(band.Midpoint, 10),
				"maxPay":        strconv.FormatInt(band.Max, 10),
				"stepAmount":    strconv.FormatInt(band.Step, 10),
				"effectiveDate": "1405/01/01",
			},
			SeedStatus: "active",
		})
	}

	jobPricings := []contracts.RecordInput{
		{Kind: KindJobPricing, Title: "قیمت‌گذاری شغل — سرپرست بهره‌برداری",
			NoteFa: "امتیاز ارزشیابی ۲۴۰ → پلهٔ ۵ و باند مدیریت",
			Fields: map[string]string{
				"positionCode": "POS-0142", "positionTitle": "سرپرست بهره‌برداری", "jobFamily": "operations",
				"evaluationPoints": "240", "factorKnowledge": "70", "factorProblemSolving": "65",
				"factorResponsibility": "65", "factorWorkingConditions": "40", "gradeRank": "5",
				"bandMidpoint": strconv.FormatInt(rules.CompensationBandFor(5, params).Midpoint, 10),
				"pricingDate":  "1405/01/12", "approvalNo": "۱۴۰۵/ط‌م/۰۱۸",
			},
			SeedStatus: "approved"},
		{Kind: KindJobPricing, Title: "قیمت‌گذاری شغل — کارشناس تعمیرات",
			NoteFa: "امتیاز ارزشیابی ۱۸۰ → پلهٔ ۴ و باند سرپرستی",
			Fields: map[string]string{
				"positionCode": "POS-0208", "positionTitle": "کارشناس تعمیرات", "jobFamily": "maintenance",
				"evaluationPoints": "180", "factorKnowledge": "60", "factorProblemSolving": "50",
				"factorResponsibility": "45", "factorWorkingConditions": "25", "gradeRank": "4",
				"bandMidpoint": strconv.FormatInt(rules.CompensationBandFor(4, params).Midpoint, 10),
				"pricingDate":  "1405/01/12", "approvalNo": "۱۴۰۵/ط‌م/۰۱۹",
			},
			SeedStatus: "approved"},
		{Kind: KindJobPricing, Title: "قیمت‌گذاری شغل — کارشناس منابع انسانی",
			NoteFa: "امتیاز ارزشیابی ۱۸۰ → پلهٔ ۴ و باند سرپرستی",
			Fields: map[string]string{
				"positionCode": "POS-0311", "positionTitle": "کارشناس منابع انسانی", "jobFamily": "hr",
				"evaluationPoints": "180", "factorKnowledge": "65", "factorProblemSolving": "45",
				"factorResponsibility": "45", "factorWorkingConditions": "25", "gradeRank": "4",
				"bandMidpoint": strconv.FormatInt(rules.CompensationBandFor(4, params).Midpoint, 10),
				"pricingDate":  "1405/01/12", "approvalNo": "۱۴۰۵/ط‌م/۰۲۰",
			},
			SeedStatus: "approved"},
		// دو پست از پست‌های واقعی همین استقرار تا نمونهٔ سرتاسری ساخته شود:
		// «کارشناس شبکه» (EMP-011) و «کارشناس ارشد مالی» (EMP-004).
		{Kind: KindJobPricing, Title: "قیمت‌گذاری شغل — کارشناس شبکه",
			NoteFa: "امتیاز ارزشیابی ۱۸۰ → پلهٔ ۴؛ نمونهٔ سرتاسری کارمند دارای حکم ابلاغ‌شده",
			Fields: map[string]string{
				"positionCode": "POS-0455", "positionTitle": "کارشناس شبکه", "jobFamily": "it",
				"evaluationPoints": "180", "factorKnowledge": "65", "factorProblemSolving": "45",
				"factorResponsibility": "45", "factorWorkingConditions": "25", "gradeRank": "4",
				"bandMidpoint": strconv.FormatInt(rules.CompensationBandFor(4, params).Midpoint, 10),
				"pricingDate":  "1405/01/12", "approvalNo": "۱۴۰۵/ط‌م/۰۲۱",
			},
			SeedStatus: "approved"},
		{Kind: KindJobPricing, Title: "قیمت‌گذاری شغل — کارشناس ارشد مالی",
			NoteFa: "امتیاز ارزشیابی ۲۴۰ → پلهٔ ۵؛ نمونهٔ کارمندی که پرداخت فعلی‌اش از سقف باند بالاتر است",
			Fields: map[string]string{
				"positionCode": "POS-0502", "positionTitle": "کارشناس ارشد مالی", "jobFamily": "finance",
				"evaluationPoints": "240", "factorKnowledge": "70", "factorProblemSolving": "65",
				"factorResponsibility": "65", "factorWorkingConditions": "40", "gradeRank": "5",
				"bandMidpoint": strconv.FormatInt(rules.CompensationBandFor(5, params).Midpoint, 10),
				"pricingDate":  "1405/01/12", "approvalNo": "۱۴۰۵/ط‌م/۰۲۲",
			},
			SeedStatus: "approved"},
	}

	competencies := []contracts.RecordInput{
		{Kind: KindCompetencyLevel, Title: "سطح شایستگی — سرپرست بهره‌برداری", EmployeeID: "EMP-001",
			NoteFa: "سطح ۴ (پیشرفته) بر پایهٔ سنجش مرکز ارزیابی",
			Fields: map[string]string{
				"competencyModel": "operations", "level": "4", "levelDate": "1405/02/15",
				"levelScore": "84", "assessorName": "کمیتهٔ شایستگی", "evidence": "گزارش مرکز ارزیابی و سابقهٔ ۱۲ سال",
			},
			SeedStatus: "confirmed"},
		{Kind: KindCompetencyLevel, Title: "سطح شایستگی — کارشناس تعمیرات", EmployeeID: "EMP-004",
			NoteFa: "سطح ۳ (مستقل) بر پایهٔ سنجش سرپرست",
			Fields: map[string]string{
				"competencyModel": "maintenance", "level": "3", "levelDate": "1405/02/15",
				"levelScore": "72", "assessorName": "مدیر تعمیرات", "evidence": "ارزیابی مهارت فنی و آزمون عملی",
			},
			SeedStatus: "confirmed"},
		{Kind: KindCompetencyLevel, Title: "سطح شایستگی — کارشناس منابع انسانی", EmployeeID: "EMP-008",
			NoteFa: "سطح ۵ (مرجع) بر پایهٔ سابقه و تألیف دستورالعمل",
			Fields: map[string]string{
				"competencyModel": "hr", "level": "5", "levelDate": "1405/02/15",
				"levelScore": "93", "assessorName": "مدیر منابع انسانی", "evidence": "دستورالعمل‌های تدوین‌شده و بازخورد مدیران",
			},
			SeedStatus: "confirmed"},
	}

	performances := []contracts.RecordInput{
		{Kind: KindPerformanceFactor, Title: "ضریب عملکرد ۱۴۰۴ — سرپرست بهره‌برداری", EmployeeID: "EMP-001",
			NoteFa: "امتیاز ۸۶ و ضریب «خوب»",
			Fields: map[string]string{
				"cycleTitle": "ارزیابی عملکرد سالانهٔ ۱۴۰۴", "cycleYear": "1404", "score": "86",
				"level": "good", "factor": "1.08", "managerName": "مدیر بهره‌برداری", "approvedDate": "1405/01/20",
			},
			SeedStatus: "approved"},
		{Kind: KindPerformanceFactor, Title: "ضریب عملکرد ۱۴۰۴ — کارشناس تعمیرات", EmployeeID: "EMP-004",
			NoteFa: "امتیاز ۷۸ و ضریب «قابل قبول»",
			Fields: map[string]string{
				"cycleTitle": "ارزیابی عملکرد سالانهٔ ۱۴۰۴", "cycleYear": "1404", "score": "78",
				"level": "meets", "factor": "1", "managerName": "مدیر تعمیرات", "approvedDate": "1405/01/20",
			},
			SeedStatus: "approved"},
		{Kind: KindPerformanceFactor, Title: "ضریب عملکرد ۱۴۰۴ — کارشناس منابع انسانی", EmployeeID: "EMP-008",
			NoteFa: "امتیاز ۹۲ و ضریب «عالی»",
			Fields: map[string]string{
				"cycleTitle": "ارزیابی عملکرد سالانهٔ ۱۴۰۴", "cycleYear": "1404", "score": "92",
				"level": "excellent", "factor": "1.15", "managerName": "مدیر منابع انسانی", "approvedDate": "1405/01/20",
			},
			SeedStatus: "approved"},
		{Kind: KindCompetencyLevel, Title: "سطح شایستگی — کارشناس شبکه", EmployeeID: "EMP-011",
			NoteFa: "سطح ۳ (مستقل) بر پایهٔ ارزیابی مهارت فنی",
			Fields: map[string]string{
				"competencyModel": "it", "level": "3", "levelDate": "1405/02/15",
				"levelScore": "74", "assessorName": "مدیر فناوری اطلاعات", "evidence": "ارزیابی مهارت شبکه و گواهینامه‌های تخصصی",
			},
			SeedStatus: "confirmed"},
		{Kind: KindPerformanceFactor, Title: "ضریب عملکرد ۱۴۰۴ — کارشناس شبکه", EmployeeID: "EMP-011",
			NoteFa: "امتیاز ۸۲ و ضریب «خوب»",
			Fields: map[string]string{
				"cycleTitle": "ارزیابی عملکرد سالانهٔ ۱۴۰۴", "cycleYear": "1404", "score": "82",
				"level": "good", "factor": "1.08", "managerName": "مدیر فناوری اطلاعات", "approvedDate": "1405/01/20",
			},
			SeedStatus: "approved"},
	}

	decisions := []contracts.RecordInput{
		{Kind: KindCompensationDecision, Title: "تصمیم چهارعاملی — سرپرست بهره‌برداری (۱۴۰۵)", EmployeeID: "EMP-001",
			NoteFa: "خروجی موتور ۴پی: شغل پلهٔ ۵، شخص سطح ۴، عملکرد «خوب»",
			Fields: map[string]string{
				"decisionNo": "۱۴۰۵/ج‌خ/۰۲۷", "decisionDate": "1405/02/25", "effectiveDate": "1405/03/01",
				"jobScore": "240", "gradeRank": "5", "competencyLevel": "4", "performanceScore": "86",
				"currentPay": "268000000", "targetPay": "285000000", "increaseAmount": "17000000",
				"compaRatioPercent": "94.1", "marketAdjust": "no",
				"decisionFa": "با ارزشیابی شغل پلهٔ ۵، سطح شایستگی ۴ و عملکرد «خوب»: پرداخت هدف ۲۸۵٬۰۰۰٬۰۰۰ ریال و افزایش ماهانه ۱۷٬۰۰۰٬۰۰۰ ریال مصوب شد.",
			},
			SeedStatus: "approved"},
	}

	out := make([]contracts.RecordInput, 0, 1+len(bands)+len(jobPricings)+len(competencies)+len(performances)+len(decisions))
	out = append(out, model)
	out = append(out, bands...)
	out = append(out, jobPricings...)
	out = append(out, competencies...)
	out = append(out, performances...)
	out = append(out, decisions...)
	return out
}
