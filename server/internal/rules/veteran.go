// Package rules — قاعدهٔ امور ایثارگران و امتیازات قانونی آنان.
//
// چرا این فایل جدا از حقوق: امتیازهای ایثارگری هم در پرونده (وقتی کارشناس پرونده را
// بررسی میکند) و هم در حقوق (وقتی فیش محاسبه میشود) و هم در ساختار سازمانی و پروندهٔ
// تحصیلی (معادلسازی «یک مقطع بالاتر») به کار میآید. اگر قاعده در ماژول حقوق میماند،
// ماژولهای دیگر ناچار بودند همان منطق را تکرار کنند؛ پس قاعده یکجا در دامنه است و همهٔ
// مصرفکنندگان همین توابع را صدا میزنند.
//
// مبنای قانونی پیشفرضها (قابل ویرایش در پارامترهای شرکت):
//
//	۱. بند «... حقوق و فوقالعادهٔ شغل و سایر فوقالعادهها اعم از مستمر و غیرمستمر شاهد،
//	   جانبازان و آزادگان و رزمندگان با حداقل دوازده ماه حضور در جبهه از پرداخت مالیات
//	   معاف میباشند» — ماده ۵۶ قانون جامع خدماترسانی به ایثارگران (مصوب ۱۳۸۶/۰۴/۰۳).
//	۲. «مدت اسارت ... به ازاء هر یک سال اسارت دو سال به عنوان سابقهٔ خدمت رسمی و مرتبط
//	   تلقی میگردد ... و دستگاه ذینفع موظف است کسورات بازنشستگی را ... بپردازد» — ماده ۱۳
//	   قانون حمایت از آزادگان (اسرای آزاد شده) مصوب ۱۳۶۸؛ یعنی بار کسور سنوات ارفاقی بر
//	   عهدهٔ دستگاه است و از سهم کارمند کسر نمیشود.
//	۳. «مدرک تحصیلی آزادگان از نظر استخدامی معادل یک مقطع تحصیلی بالاتر تلقی میشود» —
//	   تبصرهٔ مادهٔ ۲۱ قانون حمایت از آزادگان؛ برای جانبازان نیز «دستورالعمل نحوه احتساب
//	   یک مقطع تحصیلی بالاتر جانبازان» همین امتیاز را مقرر کرده است.
//	۴. سنوات ارفاقی جانبازان بر پایهٔ درصد جانبازی — صدر مادهٔ واحدهٔ «قانون نحوه
//	   بازنشستگی جانبازان انقلاب اسلامی و جنگ تحمیلی و معلولین عادی و شاغلین مشاغل سخت و
//	   زیانآور» مصوب ۱۳۶۷ و آییننامهٔ آن (۲۰٪ تا ۴۰٪: حداقل ۳ و حداکثر ۶ سال، و مجموع
//	   سابقه بههمراه ارفاقی حداکثر ۱۰ سال).
//	۵. استراحت و ذخیرهٔ مرخصی آزادگان به نسبت مدت اسارت (یکدهم مدت اسارت، حداقل دو ماه و
//	   حداکثر شش ماه) — مواد ۸ و ۹ قانون حمایت از آزادگان.
//
// هر عددی که «سیاست شرکت» است (درصد معافیت کسور، تعداد پایهٔ تشویقی، درصد سهمیهٔ استخدامی)
// در پارامترها آمده و در استقرار از تنظیمات شرکت خوانده میشود؛ پس تغییر سیاست نیازی به
// تغییر کد ندارد. هیچ قاعدهای در رابط کاربری نیست.
package rules

import (
	"strconv"
	"strings"
)

// گروههای ایثارگری؛ کلید ماشینخوان، ماندگار و مستقل از عنوان نمایشی.
const (
	// VeteranGroupNone بدون سابقهٔ ایثارگری.
	VeteranGroupNone = "none"
	// VeteranGroupCombatant رزمنده (حضور در جبهه).
	VeteranGroupCombatant = "combatant"
	// VeteranGroupDisabled جانباز (با درصد جانبازی).
	VeteranGroupDisabled = "disabled"
	// VeteranGroupFreed آزاده (اسیر آزادشده، با مدت اسارت).
	VeteranGroupFreed = "freed"
	// VeteranGroupMartyrFamily خانوادهٔ شهید: پدر، مادر و همسر شهید.
	VeteranGroupMartyrFamily = "martyr_family"
	// VeteranGroupMartyrChild فرزند شهید.
	VeteranGroupMartyrChild = "martyr_child"
	// VeteranGroupVeteranChild فرزند جانباز یا آزاده (مشمول سهمیهٔ استخدامی).
	VeteranGroupVeteranChild = "veteran_child"
	// VeteranGroupOther سایر ایثارگران و مشمولان قانون.
	VeteranGroupOther = "other"
)

// VeteranGroup یک گروه ایثارگری برای فرم پرونده و برچسب سند.
type VeteranGroup struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// VeteranGroups فهرست گروههای ایثارگری (خروجی فرم؛ فهرست در UI ساخته نمیشود).
func VeteranGroups() []VeteranGroup {
	return []VeteranGroup{
		{Key: VeteranGroupCombatant, TitleFa: "رزمنده", NoteFa: "حضور در جبهه؛ معافیت مالیاتی ماده ۵۶ مشروط به حداقل دوازده ماه حضور است."},
		{Key: VeteranGroupDisabled, TitleFa: "جانباز", NoteFa: "با درصد جانبازی؛ مبنای سنوات ارفاقی و کسور بر عهدهٔ دستگاه."},
		{Key: VeteranGroupFreed, TitleFa: "آزاده", NoteFa: "با مدت اسارت؛ هر سال اسارت دو سال سابقهٔ خدمت رسمی محسوب میشود."},
		{Key: VeteranGroupMartyrFamily, TitleFa: "خانوادهٔ شهید (پدر، مادر، همسر)", NoteFa: "مشمول معافیت مالیاتی ماده ۵۶ و امتیازهای سهمیه."},
		{Key: VeteranGroupMartyrChild, TitleFa: "فرزند شهید", NoteFa: "مشمول سهمیهٔ استخدامی و امتیازهای استخدامی ایثارگران."},
		{Key: VeteranGroupVeteranChild, TitleFa: "فرزند جانباز یا آزاده", NoteFa: "مشمول سهمیهٔ استخدامی و امتیازهای استخدامی ایثارگران."},
		{Key: VeteranGroupOther, TitleFa: "سایر ایثارگران", NoteFa: "هر مورد دیگر که با مستندات مرجع حمایتی تأیید شود."},
	}
}

// VeteranGroupOf گروه را از کلید ماشینخوان یا عنوان فارسی پیدا میکند؛ ناشناخته ⇒ گروه «سایر».
//
// چرا هر دو شکل: پروندههای قدیمی این ماژول نوع ایثارگری را متنی («جانباز») ذخیره کردهاند و
// باید بدون مهاجرت داده خوانده شوند.
func VeteranGroupOf(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return VeteranGroupNone
	}
	for _, g := range VeteranGroups() {
		if value == g.Key || value == g.TitleFa {
			return g.Key
		}
	}
	// ترتیب مهم است: «فرزند شهید» و «فرزند جانباز» پیش از «شهید» و «جانباز» سنجیده می‌شوند،
	// وگرنه فرزندِ ایثارگر همان گروه خودِ ایثارگر شمرده می‌شد.
	switch {
	case strings.Contains(value, "فرزند") && strings.Contains(value, "شهید"):
		return VeteranGroupMartyrChild
	case strings.Contains(value, "فرزند"):
		return VeteranGroupVeteranChild
	case strings.Contains(value, "جانباز"):
		return VeteranGroupDisabled
	case strings.Contains(value, "آزاده"):
		return VeteranGroupFreed
	case strings.Contains(value, "رزمند"):
		return VeteranGroupCombatant
	case strings.Contains(value, "همسر") || strings.Contains(value, "مادر") || strings.Contains(value, "پدر"):
		return VeteranGroupMartyrFamily
	case strings.Contains(value, "شهید") || strings.Contains(value, "شاهد"):
		return VeteranGroupMartyrFamily
	}
	return VeteranGroupOther
}

// VeteranGroupTitleFa عنوان فارسی گروه؛ کلید ناشناخته خودش برمیگردد (برای متن قدیمی پرونده).
func VeteranGroupTitleFa(key string) string {
	value := strings.TrimSpace(key)
	for _, g := range VeteranGroups() {
		if value == g.Key {
			return g.TitleFa
		}
	}
	return value
}

// VeteranProfile پروندهٔ ایثارگری یک کارمند به شکلی که موتور میفهمد.
type VeteranProfile struct {
	// Group کلید گروه ایثارگری (VeteranGroup*).
	Group string `json:"group"`
	// DisabilityPercent درصد جانبازی (۰ تا ۱۰۰).
	DisabilityPercent float64 `json:"disabilityPercent"`
	// CaptivityMonths مدت اسارت به ماه.
	CaptivityMonths int `json:"captivityMonths"`
	// FrontMonths حضور در جبهه به ماه.
	FrontMonths int `json:"frontMonths"`
	// ApprovedFile یعنی پرونده «تأیید نهایی» شده است؛ تا آن زمان هیچ امتیازی داده نمیشود.
	ApprovedFile bool `json:"approvedFile"`
}

// VeteranParams پارامترهای سیاستی امتیازهای ایثارگری (از پارامترهای شرکت).
type VeteranParams struct {
	// TaxExemptPercent درصد معافیت مالیاتی حقوق (پیشفرض ۱۰۰ بر پایهٔ ماده ۵۶).
	TaxExemptPercent float64 `json:"taxExemptPercent"`
	// TaxMinFrontMonths حداقل حضور در جبهه برای معافیت مالیاتی رزمندگان (ماده ۵۶: ۱۲ ماه).
	TaxMinFrontMonths int `json:"taxMinFrontMonths"`
	// PensionExemptPercent درصد معافیتی که از کسور سهم کارمند کسر میشود. قانون، کسور
	// سنوات ارفاقی را بر عهدهٔ دستگاه گذاشته است؛ مقدار این کاهش سیاست شرکت است.
	PensionExemptPercent float64 `json:"pensionExemptPercent"`
	// CaptivityServiceFactor ضریب تبدیل هر سال اسارت به سابقهٔ خدمت (قانون: ۲).
	CaptivityServiceFactor float64 `json:"captivityServiceFactor"`
	// ErafaqiMonthsOverride اگر بزرگتر از صفر باشد، جدول قانونی سنوات ارفاقی نادیده گرفته
	// و همین عدد ماه برای جانبازان استفاده میشود (سیاست شرکت).
	ErafaqiMonthsOverride int `json:"erafaqiMonthsOverride"`
	// ErafaqiMaxMonths سقف مجموع سنوات ارفاقی به ماه (آییننامه: ۱۰ سال = ۱۲۰ ماه).
	ErafaqiMaxMonths int `json:"erafaqiMaxMonths"`
	// EducationUpgrade امتیاز «یک مقطع تحصیلی بالاتر» برای جانباز و آزاده.
	EducationUpgrade bool `json:"educationUpgrade"`
	// IncentiveSteps تعداد پایهٔ تشویقی به تفکیک گروه (سیاست شرکت؛ پیشفرض: ۲ برای خانوادهٔ
	// شهید و ۱ برای جانباز، آزاده و رزمنده).
	IncentiveSteps map[string]int `json:"incentiveSteps"`
	// IncentiveStepsMax سقف پایهٔ تشویقی در یک سال.
	IncentiveStepsMax int `json:"incentiveStepsMax"`
	// CaptivityLeaveRatio نسبت مرخصی تشویقی به مدت اسارت (قانون: یکدهم).
	CaptivityLeaveRatio float64 `json:"captivityLeaveRatio"`
	// CaptivityLeaveMinMonths و CaptivityLeaveMaxMonths کف و سقف مرخصی تشویقی آزادگان.
	CaptivityLeaveMinMonths int `json:"captivityLeaveMinMonths"`
	CaptivityLeaveMaxMonths int `json:"captivityLeaveMaxMonths"`
	// EmploymentQuotaPercent درصد سهمیهٔ استخدامی ایثارگران (نمایشی و گزارشی).
	EmploymentQuotaPercent float64 `json:"employmentQuotaPercent"`
	// PayrollApply آیا امتیازها در محاسبهٔ فیش حقوق اعمال شوند.
	PayrollApply bool `json:"payrollApply"`
}

// DefaultVeteranParams پیشفرضهای مستند به قانون (هر عدد در پانویس همین فایل ارجاع دارد).
func DefaultVeteranParams() VeteranParams {
	return VeteranParams{
		TaxExemptPercent:  100,
		TaxMinFrontMonths: 12,
		// پیشفرض صفر: قانون کسور «سنوات ارفاقی» را بر عهدهٔ دستگاه گذاشته است، ولی کاهش
		// کسور سهم کارمند سیاست شرکت است و بدون تصویب آن اعمال نمیشود.
		PensionExemptPercent:   0,
		CaptivityServiceFactor: 2,
		ErafaqiMaxMonths:       120,
		EducationUpgrade:       true,
		IncentiveSteps: map[string]int{
			VeteranGroupCombatant:    1,
			VeteranGroupDisabled:     1,
			VeteranGroupFreed:        1,
			VeteranGroupMartyrFamily: 2,
			VeteranGroupMartyrChild:  1,
			VeteranGroupVeteranChild: 0,
			VeteranGroupOther:        0,
		},
		IncentiveStepsMax:       2,
		CaptivityLeaveRatio:     0.1,
		CaptivityLeaveMinMonths: 2,
		CaptivityLeaveMaxMonths: 6,
		EmploymentQuotaPercent:  25,
		PayrollApply:            true,
	}
}

// VeteranParamsFromValues پارامترهای شرکت را روی پیشفرض مینشاند؛ کلید غایب پیشفرض میماند.
func VeteranParamsFromValues(values map[string]float64) VeteranParams {
	p := DefaultVeteranParams()
	if v, ok := values["veteran.tax.exemptPercent"]; ok && v >= 0 && v <= 100 {
		p.TaxExemptPercent = v
	}
	if v, ok := values["veteran.tax.minFrontMonths"]; ok && v >= 0 {
		p.TaxMinFrontMonths = int(v)
	}
	if v, ok := values["veteran.pension.exemptPercent"]; ok && v >= 0 && v <= 100 {
		p.PensionExemptPercent = v
	}
	if v, ok := values["veteran.pension.captivityServiceFactor"]; ok && v > 0 {
		p.CaptivityServiceFactor = v
	}
	if v, ok := values["veteran.pension.erafaqiMonthsOverride"]; ok && v >= 0 {
		p.ErafaqiMonthsOverride = int(v)
	}
	if v, ok := values["veteran.pension.erafaqiMaxMonths"]; ok && v > 0 {
		p.ErafaqiMaxMonths = int(v)
	}
	if v, ok := values["veteran.education.upgrade"]; ok {
		p.EducationUpgrade = v != 0
	}
	for _, key := range []string{
		VeteranGroupCombatant, VeteranGroupDisabled, VeteranGroupFreed,
		VeteranGroupMartyrFamily, VeteranGroupMartyrChild, VeteranGroupVeteranChild,
	} {
		if v, ok := values["veteran.incentive.steps."+key]; ok && v >= 0 {
			p.IncentiveSteps[key] = int(v)
		}
	}
	if v, ok := values["veteran.incentive.stepsMax"]; ok && v > 0 {
		p.IncentiveStepsMax = int(v)
	}
	if v, ok := values["veteran.leave.captivityRatio"]; ok && v > 0 {
		p.CaptivityLeaveRatio = v
	}
	if v, ok := values["veteran.leave.minMonths"]; ok && v >= 0 {
		p.CaptivityLeaveMinMonths = int(v)
	}
	if v, ok := values["veteran.leave.maxMonths"]; ok && v > 0 {
		p.CaptivityLeaveMaxMonths = int(v)
	}
	if v, ok := values["veteran.quota.employmentPercent"]; ok && v >= 0 && v <= 100 {
		p.EmploymentQuotaPercent = v
	}
	if v, ok := values["veteran.payroll.apply"]; ok {
		p.PayrollApply = v != 0
	}
	return p
}

// VeteranLine یک سطر امتیاز با شرح فارسی (برای نمایش و سند).
type VeteranLine struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	ValueFa string `json:"valueFa"`
	RefFa   string `json:"refFa,omitempty"`
}

// VeteranEntitlement امتیازهای محاسبهشدهٔ یک پروندهٔ ایثارگری.
type VeteranEntitlement struct {
	Group         string  `json:"group"`
	GroupFa       string  `json:"groupFa"`
	Approved      bool    `json:"approved"`
	IsVeteran     bool    `json:"isVeteran"`
	TaxExempt     float64 `json:"taxExemptPercent"`
	PensionExempt float64 `json:"pensionExemptPercent"`
	// CaptivityServiceMonths افزایش سنواتی ناشی از مدت اسارت (ماده ۱۳).
	CaptivityServiceMonths int `json:"captivityServiceMonths"`
	// ErafaqiMonths سنوات ارفاقی جانبازان بر پایهٔ درصد جانبازی.
	ErafaqiMonths int `json:"erafaqiMonths"`
	// TotalExtraServiceMonths جمع افزایش سنواتی قابل ثبت در پرونده.
	TotalExtraServiceMonths int           `json:"totalExtraServiceMonths"`
	EducationUpgrade        bool          `json:"educationUpgrade"`
	IncentiveSteps          int           `json:"incentiveSteps"`
	LeaveDays               int           `json:"leaveDays"`
	EmploymentQuotaPercent  float64       `json:"employmentQuotaPercent"`
	Lines                   []VeteranLine `json:"lines"`
	SummaryFa               string        `json:"summaryFa"`
	NoteFa                  string        `json:"noteFa,omitempty"`
}

// ErafaqiMonthsForPercent جدول قانونی سنوات ارفاقی جانبازان به ماه.
//
// مأخذ: صدر مادهٔ واحدهٔ قانون نحوه بازنشستگی جانبازان ... مصوب ۱۳۶۷ و آییننامهٔ آن؛
// ۲۰٪ تا ۴۰٪ حداقل ۳ و حداکثر ۶ سال، و برای درصدهای بالاتر بیشتر (تا سقف مقرر در
// پارامتر `veteran.pension.erafaqiMaxMonths`). جدول در کد است چون «متن قانون» است، نه نرخ
// شرکت؛ ولی با پارامتر می توان آن را برای شرکت بازنویسی کرد.
func ErafaqiMonthsForPercent(percent float64) int {
	switch {
	case percent >= 40:
		return 72
	case percent >= 30:
		return 60
	case percent >= 25:
		return 48
	case percent >= 20:
		return 36
	}
	// برای درصدهای بالاتر از ۴۰٪ و مشاغل سخت و زیانآور، سقف مجموعِ سابقه و ارفاقی تا ده
	// سال است و میزان دقیق با آییننامهٔ همان گروه تعیین میشود؛ مقدار شرکت را میتوان با
	// پارامتر `veteran.pension.erafaqiMonthsOverride` ثبت کرد.
	return 0
}

// groupEligibleForTax مشمولان معافیت مالیاتی ماده ۵۶.
func groupEligibleForTax(group string, profile VeteranProfile, params VeteranParams) bool {
	switch group {
	case VeteranGroupDisabled, VeteranGroupFreed, VeteranGroupMartyrFamily:
		return true
	case VeteranGroupCombatant:
		// ماده ۵۶: رزمندهٔ مشمول معافیت، کسی است که حداقل دوازده ماه حضور در جبهه داشته باشد.
		return profile.FrontMonths >= params.TaxMinFrontMonths
	}
	return false
}

// groupEligibleForEducation ارتقای مدرک (یک مقطع بالاتر): جانباز و آزاده.
func groupEligibleForEducation(group string) bool {
	return group == VeteranGroupDisabled || group == VeteranGroupFreed
}

// ComputeVeteranEntitlements امتیازهای یک پروندهٔ ایثارگری را از پارامترهای شرکت میسازد.
//
// قاعدهٔ محافظهکارانه: تا وقتی پرونده «تأیید نهایی» نشده باشد، هیچ امتیاز مالی و سنواتی
// اعمال نمیشود؛ فقط متن راهنما برمیگردد (پروندهٔ در حال بررسی نباید حقوق را تغییر دهد).
func ComputeVeteranEntitlements(profile VeteranProfile, params VeteranParams) VeteranEntitlement {
	group := profile.Group
	if group == "" {
		group = VeteranGroupNone
	}
	out := VeteranEntitlement{
		Group:    group,
		GroupFa:  VeteranGroupTitleFa(group),
		Approved: profile.ApprovedFile,
		Lines:    []VeteranLine{},
	}
	if group == VeteranGroupNone {
		out.NoteFa = "این کارمند پروندهٔ ایثارگری ندارد."
		out.SummaryFa = "بدون امتیاز ایثارگری."
		return out
	}
	out.IsVeteran = true
	if !profile.ApprovedFile {
		out.NoteFa = "پروندهٔ ایثارگری ثبت شده ولی تأیید نهایی نشده است؛ امتیازها پس از تأیید اعمال میشود."
		out.SummaryFa = "پروندهٔ ایثارگری " + out.GroupFa + " — در انتظار تأیید نهایی."
		return out
	}

	// ۱) معافیت مالیاتی (ماده ۵۶ قانون جامع خدماترسانی به ایثارگران).
	if groupEligibleForTax(group, profile, params) {
		out.TaxExempt = params.TaxExemptPercent
		out.Lines = append(out.Lines, VeteranLine{
			Key: "taxExempt", TitleFa: "معافیت مالیاتی حقوق و فوقالعادهها",
			ValueFa: IntFa(params.TaxExemptPercent) + "٪",
			RefFa:   "ماده ۵۶ قانون جامع خدماترسانی به ایثارگران",
		})
	} else {
		out.Lines = append(out.Lines, VeteranLine{
			Key: "taxExempt", TitleFa: "معافیت مالیاتی حقوق و فوقالعادهها", ValueFa: "مشمول نیست",
			RefFa: "ماده ۵۶ قانون جامع خدماترسانی به ایثارگران (رزمندهٔ مشمول: حداقل ۱۲ ماه حضور در جبهه)",
		})
	}

	// ۲) سیاست شرکت در کاهش کسور سهم کارمند (معافیت کسور بازنشستگی).
	out.PensionExempt = params.PensionExemptPercent

	// ۳) افزایش سنواتی آزادگان (ماده ۱۳ قانون حمایت از آزادگان: هر سال اسارت دو سال سابقه).
	if group == VeteranGroupFreed && profile.CaptivityMonths > 0 {
		years := float64(profile.CaptivityMonths) / 12
		out.CaptivityServiceMonths = RoundHalfUpToInt(years * params.CaptivityServiceFactor * 12)
		out.Lines = append(out.Lines, VeteranLine{
			Key: "captivityService", TitleFa: "سابقهٔ خدمت بابت مدت اسارت",
			ValueFa: IntFa(out.CaptivityServiceMonths) + " ماه (" + IntFa(profile.CaptivityMonths) + " ماه اسارت × " +
				TrimFloatFa(params.CaptivityServiceFactor) + ")",
			RefFa: "ماده ۱۳ قانون حمایت از آزادگان مصوب ۱۳۶۸",
		})
	}

	// ۳) سنوات ارفاقی جانبازان بر پایهٔ درصد جانبازی.
	if group == VeteranGroupDisabled && profile.DisabilityPercent > 0 {
		months := params.ErafaqiMonthsOverride
		if months <= 0 {
			months = ErafaqiMonthsForPercent(profile.DisabilityPercent)
		}
		if months > params.ErafaqiMaxMonths {
			months = params.ErafaqiMaxMonths
		}
		out.ErafaqiMonths = months
		if months > 0 {
			out.Lines = append(out.Lines, VeteranLine{
				Key: "erafaqi", TitleFa: "سنوات ارفاقی جانبازی",
				ValueFa: IntFa(months) + " ماه برای " + IntFa(profile.DisabilityPercent) + "٪ جانبازی",
				RefFa:   "قانون نحوه بازنشستگی جانبازان مصوب ۱۳۶۷ و آییننامهٔ آن",
			})
			out.Lines = append(out.Lines, VeteranLine{
				Key: "erafaqiPension", TitleFa: "کسور سنوات ارفاقی",
				ValueFa: "بر عهدهٔ دستگاه (از سهم کارمند کسر نمیشود)",
				RefFa:   "ماده ۱۳ قانون حمایت از آزادگان و رویهٔ صندوقهای بازنشستگی",
			})
		}
	}

	out.TotalExtraServiceMonths = out.CaptivityServiceMonths + out.ErafaqiMonths
	if out.PensionExempt > 0 {
		out.Lines = append(out.Lines, VeteranLine{
			Key: "pensionExempt", TitleFa: "معافیت کسور بازنشستگی (سیاست شرکت)",
			ValueFa: IntFa(params.PensionExemptPercent) + "٪ از کسور سهم کارمند",
			RefFa:   "پارامتر شرکت veteran.pension.exemptPercent",
		})
	}

	// ۴) معادلسازی یک مقطع تحصیلی بالاتر.
	if params.EducationUpgrade && groupEligibleForEducation(group) {
		out.EducationUpgrade = true
		out.Lines = append(out.Lines, VeteranLine{
			Key: "educationUpgrade", TitleFa: "معادلسازی مدرک، یک مقطع تحصیلی بالاتر",
			ValueFa: "مجاز", RefFa: "تبصرهٔ مادهٔ ۲۱ قانون حمایت از آزادگان و دستورالعمل جانبازان",
		})
	}

	// ۵) پایهٔ تشویقی (سیاست شرکت؛ سقف پارامتری).
	steps := params.IncentiveSteps[group]
	if steps > params.IncentiveStepsMax {
		steps = params.IncentiveStepsMax
	}
	if steps < 0 {
		steps = 0
	}
	out.IncentiveSteps = steps
	if steps > 0 {
		out.Lines = append(out.Lines, VeteranLine{
			Key: "incentiveSteps", TitleFa: "پایهٔ تشویقی",
			ValueFa: IntFa(steps) + " پایه", RefFa: "سیاست شرکت (پارامتر veteran.incentive.steps.*)",
		})
	}

	// ۶) مرخصی تشویقی آزادگان به نسبت مدت اسارت (مواد ۸ و ۹ قانون حمایت از آزادگان).
	if group == VeteranGroupFreed && profile.CaptivityMonths > 0 {
		months := float64(profile.CaptivityMonths) * params.CaptivityLeaveRatio
		if months < float64(params.CaptivityLeaveMinMonths) {
			months = float64(params.CaptivityLeaveMinMonths)
		}
		if months > float64(params.CaptivityLeaveMaxMonths) {
			months = float64(params.CaptivityLeaveMaxMonths)
		}
		out.LeaveDays = RoundHalfUpToInt(months * 30)
		out.Lines = append(out.Lines, VeteranLine{
			Key: "leave", TitleFa: "مرخصی تشویقی آزادگان",
			ValueFa: IntFa(out.LeaveDays) + " روز (" + IntFa(RoundHalfUpToInt(months)) + " ماه)",
			RefFa:   "مواد ۸ و ۹ قانون حمایت از آزادگان",
		})
	}

	// ۷) سهمیهٔ استخدامی (گزارشی؛ مبنای جذب و گزارش جامعهٔ ایثارگران).
	out.EmploymentQuotaPercent = params.EmploymentQuotaPercent
	out.Lines = append(out.Lines, VeteranLine{
		Key: "quota", TitleFa: "سهمیهٔ استخدامی ایثارگران",
		ValueFa: IntFa(params.EmploymentQuotaPercent) + "٪", RefFa: "ماده ۲۱ قانون جامع خدماترسانی به ایثارگران",
	})

	out.SummaryFa = veteranSummaryFa(out)
	return out
}

// veteranSummaryFa جمعبندی فارسی امتیازها.
func veteranSummaryFa(ent VeteranEntitlement) string {
	parts := []string{"گروه: " + ent.GroupFa}
	if ent.TaxExempt > 0 {
		parts = append(parts, "معافیت مالیاتی "+IntFa(ent.TaxExempt)+"٪")
	}
	if ent.TotalExtraServiceMonths > 0 {
		parts = append(parts, "افزایش سنواتی "+IntFa(ent.TotalExtraServiceMonths)+" ماه")
	}
	if ent.EducationUpgrade {
		parts = append(parts, "یک مقطع تحصیلی بالاتر")
	}
	if ent.IncentiveSteps > 0 {
		parts = append(parts, "پایهٔ تشویقی "+IntFa(ent.IncentiveSteps))
	}
	if ent.LeaveDays > 0 {
		parts = append(parts, "مرخصی تشویقی "+IntFa(ent.LeaveDays)+" روز")
	}
	return strings.Join(parts, "؛ ") + "."
}

// VeteranReliefInput ورودی اعمال امتیاز ایثارگری بر یک فیش حقوق.
//
// چرا ورودی کوچک است: موتور حقوق فقط همین دو مبنا را لازم دارد؛ بقیهٔ امتیازها (سنوات،
// پایهٔ تشویقی، مدرک) روی «حکم» اثر میگذارند، نه روی محاسبهٔ ماهانه.
type VeteranReliefInput struct {
	// TaxableIncome درآمد مشمول مالیات پیش از اعمال امتیاز (ریال).
	TaxableIncome int64 `json:"taxableIncome"`
	// EmployeeDeduction کسور سهم کارمند (بیمه/بازنشستگی) پیش از اعمال امتیاز (ریال).
	EmployeeDeduction int64 `json:"employeeDeduction"`
}

// VeteranReliefResult تعدیل محاسبهٔ حقوق بر پایهٔ امتیاز ایثارگری.
type VeteranReliefResult struct {
	// TaxExemptAmount مبلغی که از درآمد مشمول کسر میشود (ریال).
	TaxExemptAmount int64 `json:"taxExemptAmount"`
	// PensionExemptAmount مبلغی که از کسور سهم کارمند کسر میشود (ریال).
	PensionExemptAmount int64 `json:"pensionExemptAmount"`
	// Applied آیا امتیازی اعمال شد (برای سطر فیش).
	Applied bool   `json:"applied"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// ApplyVeteranRelief امتیاز ایثارگری را روی مبناهای حقوق مینشاند.
//
// قاعدهٔ ایمنی: تعدیل هرگز از خود مبنا بیشتر نمیشود (مالیات منفی یا کسور منفی ممکن نیست).
func ApplyVeteranRelief(in VeteranReliefInput, ent VeteranEntitlement) VeteranReliefResult {
	out := VeteranReliefResult{}
	if !ent.Approved || !ent.IsVeteran {
		return out
	}
	if ent.TaxExempt > 0 && in.TaxableIncome > 0 {
		amount := RoundHalfUp(float64(in.TaxableIncome) * ent.TaxExempt / 100)
		if amount > in.TaxableIncome {
			amount = in.TaxableIncome
		}
		if amount < 0 {
			amount = 0
		}
		out.TaxExemptAmount = amount
	}
	if ent.PensionExempt > 0 && in.EmployeeDeduction > 0 {
		amount := RoundHalfUp(float64(in.EmployeeDeduction) * ent.PensionExempt / 100)
		if amount > in.EmployeeDeduction {
			amount = in.EmployeeDeduction
		}
		if amount < 0 {
			amount = 0
		}
		out.PensionExemptAmount = amount
	}
	out.Applied = out.TaxExemptAmount > 0 || out.PensionExemptAmount > 0
	if out.Applied {
		out.NoteFa = "امتیاز ایثارگری (" + ent.GroupFa + ") اعمال شد."
	}
	return out
}

// VeteranInstructionFa گامهای اجرایی امور ایثارگران برای نمایش در صفحه.
func VeteranInstructionFa() []string {
	return []string{
		"۱. پروندهٔ ایثارگری کارمند با مستندات مرجع حمایتی (کارت ایثارگری، حکم جانبازی، سند اسارت) ثبت میشود.",
		"۲. نوع ایثارگری از گروههای مصوب انتخاب میشود: رزمنده، جانباز، آزاده، خانوادهٔ شهید، فرزند شهید و فرزند جانباز.",
		"۳. دامنهٔ امتیازها پیش از تأیید نهایی محاسبه و نمایش داده میشود، ولی تا «تأیید پرونده» هیچ امتیازی اثر مالی ندارد.",
		"۴. معافیت مالیاتی حقوق بر پایهٔ ماده ۵۶ قانون جامع خدماترسانی به ایثارگران اعمال میشود.",
		"۵. مدت اسارت آزادگان به ازای هر سال، دو سال سابقهٔ خدمت رسمی است و کسور آن بر عهدهٔ دستگاه است.",
		"۶. سنوات ارفاقی جانبازان بر پایهٔ درصد جانبازی از جدول قانونی خوانده و در پرونده ثبت میشود.",
		"۷. «یک مقطع تحصیلی بالاتر» در پروندهٔ تحصیلی و مدرک ملاک حقوق اعمال میشود.",
		"۸. پایهٔ تشویقی از سیاست شرکت میآید و سقف آن پارامتری است؛ هیچ عددی در رابط نوشته نمیشود.",
		"۹. کاهش کسور سهم کارمند (معافیت کسور بازنشستگی) تنها با مقدار مصوب شرکت اعمال میشود.",
		"۱۰. تغییر پارامترها از تاریخ اجرای خود اثر میکند و محاسبهٔ پیشین را باطل نمیکند.",
	}
}

/* ------------------------------ کمکهای کوچک ------------------------------ */

/* HELPERS */

// IntFa عدد را با ارقام فارسی و بدون اعشار اضافه برمی‌گرداند (int و float هر دو پذیرفته است).
func IntFa(v any) string {
	switch n := v.(type) {
	case int:
		return persianDigits(strconv.Itoa(n))
	case int64:
		return persianDigits(strconv.FormatInt(n, 10))
	case float64:
		return persianDigits(strconv.FormatFloat(n, 'f', -1, 64))
	}
	return persianDigits(strconv.Itoa(0))
}

// TrimFloatFa عدد اعشاری را کوتاه و با ارقام فارسی می‌نویسد (مثلاً «۲۲٫۵» ⇒ «22.5» فارسی‌رقم).
func TrimFloatFa(v float64) string {
	return persianDigits(strconv.FormatFloat(v, 'f', -1, 64))
}

// RoundHalfUpToInt گرد کردن نیم‌به‌بالا به عدد صحیح.
func RoundHalfUpToInt(x float64) int {
	return int(RoundHalfUp(x))
}

/* ------------------- خواندن پرونده و پارامترها از دادهٔ ذخیره‌شده ------------------- */

// VeteranProfileFromFields پروفایل ایثارگری را از فیلدهای ذخیره‌شدهٔ پرونده می‌سازد.
//
// چرا در موتور قواعد و نه در ماژول: ماژول امور ایثارگران (برای نمایش و ثبت امتیاز) و موتور
// حقوق (برای اعمال امتیاز در فیش) باید یک عدد از یک پرونده بخوانند؛ اگر هر کدام خودش
// فیلدها را تفسیر کند، دو محاسبهٔ ناهمخوان پیدا می‌شود.
func VeteranProfileFromFields(fields map[string]string, approved bool) VeteranProfile {
	return VeteranProfile{
		Group:             VeteranGroupOf(fields["veteranType"]),
		DisabilityPercent: ParseFaNumber(fields["percent"]),
		CaptivityMonths:   int(ParseFaNumber(fields["captivityMonths"])),
		FrontMonths:       int(ParseFaNumber(fields["frontMonths"])),
		ApprovedFile:      approved,
	}
}

// VeteranParamsFromText پارامترهای شرکت را از جفت‌های متنی کلید/مقدار می‌خواند.
//
// ارقام فارسی، جداکنندهٔ هزارگان و علامت درصد پذیرفته می‌شود و کلید بی‌مقدار یا ناخوانا
// نادیده گرفته می‌شود تا پیش‌فرض قانونی بماند. تنها کلیدهای `veteran.*` اثر دارند.
func VeteranParamsFromText(values map[string]string) VeteranParams {
	numeric := make(map[string]float64, len(values))
	for key, raw := range values {
		if v, ok := parseFaNumberOK(raw); ok {
			numeric[key] = v
		}
	}
	return VeteranParamsFromValues(numeric)
}

// ParseFaNumber عدد را از متن فارسی می‌خواند (بی‌اعتبار ⇒ صفر).
func ParseFaNumber(raw string) float64 {
	v, _ := parseFaNumberOK(raw)
	return v
}

// parseFaNumberOK عدد را از متن فارسی می‌خواند و می‌گوید مقدار خوانا بود یا نه.
func parseFaNumberOK(raw string) (float64, bool) {
	value := strings.TrimSpace(strings.Map(latinDigit, raw))
	value = faNumberReplacer.Replace(value)
	if value == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// faNumberReplacer جداکننده‌ها و علامت‌های درصد/اعشار فارسی را به شکل خوانا برمی‌گرداند.
var faNumberReplacer = strings.NewReplacer(",", "", "٬", "", "%", "", "٪", "", "٫", ".")

// latinDigit ارقام فارسی و عربی را به لاتین تبدیل می‌کند (بقیهٔ نویسه‌ها دست‌نخورده).
func latinDigit(r rune) rune {
	switch {
	case r >= '۰' && r <= '۹':
		return '0' + (r - '۰')
	case r >= '٠' && r <= '٩':
		return '0' + (r - '٠')
	}
	return r
}
