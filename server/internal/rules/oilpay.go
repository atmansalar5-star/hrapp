// Package rules — قاعدهٔ «نظام پرداخت صندوق بازنشستگی نفت».
//
// چرا اینجا و نه در رابط: سازمان دو نظام پرداخت دارد. نظام نخست (کارکنان مشمول تأمین
// اجتماعی) مزد مبنا و اقلام گوناگون دارد و در `ComputePayroll` و موتور اقلام است. نظام
// دوم — کارکنان تابع صندوق بازنشستگی نفت — ساختار ساده و پیوسته‌ای دارد.
//
// سند مرجع این نظام «آیین‌نامهٔ فوق‌العاده‌ها»ی شرکت است (۲۱ ردیف و سه جدول پیوست). زنجیرهٔ
// محاسبه بر پایهٔ همان سند:
//
//	(A) پایهٔ حقوق
//	(B) فوق‌العادهٔ بدی آب و هوا (مناطق گرمسیری جنوب) = ۲۵٪ × A
//	    «مبنای فوق‌العاده‌ها» = A + B
//	(C) فوق‌العادهٔ ویژه = ۳۰٪ × (A + B)
//	    «مبنای جذب» = A + B + C
//
//	فوق‌العاده‌های ردیف ۵، ۸، ۱۳ و ۱۷ آیین‌نامه درصدی از (A+B) هستند؛
//	ردیف ۷ (جذب عمومی و تخصصی شغل) درصدی از (A+B+C) است و درصدش از دو جدول می‌آید:
//	«جدول ۳» پایهٔ سمت (حق جذب عمومی + حق مقام = Y) و «درصد مدرک تحصیلی» (Z).
//	ردیف ۴ (محرومیت از تسهیلات زندگی) و ردیف ۶ (جبران صرف وقت اضافی) جدول ۱ و ۲ هستند و
//	درصدشان به «محل سکونت» کارمند بستگی دارد (روزکار/نوبت‌کار هم تفکیک شده است).
//	ردیف ۱۰: اضافه‌کار هر ساعت = ۰٫۰۱۴ × A برای روزکار و ۰٫۰۱۶ × A برای نوبت‌کار.
//
// همهٔ درصدها پارامتریک‌اند (`oilpay.*`) و هر خانه از جدول‌ها هم با کلید پارامتری خودش
// (`oilpay.site.<محل>.deprivation` و `oilpay.grade.<پایه>.rank`) قابل بازنویسی است؛ پس
// تغییر مصوبه نیازی به انتشار نسخهٔ تازه ندارد.
package rules

import (
	"math"
	"strconv"
	"strings"
)

// OilAllowance یک فوق‌العادهٔ نظام پرداخت صندوق نفت.
//
// Base می‌گوید درصد روی چه مبنایی بسته می‌شود:
//
//	allowanceBase  یعنی «پایهٔ حقوق + فوق‌العادهٔ بدی آب و هوا» (A+B)،
//	attractionBase یعنی «A+B+فوق‌العادهٔ ویژه» (مبنای ردیف ۷ آیین‌نامه)،
//	baseSalary     یعنی فقط پایهٔ حقوق (برای قلم‌هایی که مصوبهٔ جدا دارند).
//
// RequiresShift یعنی این قلم تنها برای کارمند نوبت‌کار پرداخت می‌شود (نوبت‌کار).
type OilAllowance struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	GroupFa string  `json:"groupFa"`
	Percent float64 `json:"percent"`
	Base    string  `json:"base"`
	// DrivenBy میگوید درصد این قلم از کدام جدول آیین‌نامه می‌آید (خالی = درصد ثابت مصوب):
	//
	//	residence یعنی جدول ۱ و ۲ بر پایهٔ محل سکونت (محرومیت و صرف وقت)،
	//	grade     یعنی جدول ۳ بر پایهٔ پایهٔ سمت (جذب عمومی و تخصصی شغل).
	//
	// فرم وب از همین پرچم می‌فهمد کدام درصد را نباید به‌صورت دستی بفرستد؛ اگر فرم همهٔ
	// درصدها را می‌فرستاد، جدول‌های آیین‌نامه هرگز اعمال نمی‌شدند.
	DrivenBy      string `json:"drivenBy,omitempty"`
	RequiresShift bool   `json:"requiresShift,omitempty"`
	NoteFa        string `json:"noteFa,omitempty"`
}

// گروه‌های اقلام فیش.
const (
	OilGroupBase    = "پایه و مبنای پرداخت"
	OilGroupClimate = "فوق‌العاده‌های شغل و اقلیم"
	OilGroupShift   = "نوبت‌کاری و کار اضافه"
	OilGroupDeduct  = "کسور"
)

// مبنای محاسبهٔ هر قلم.
const (
	OilBaseAllowance  = "allowanceBase"
	OilBaseSalary     = "baseSalary"
	OilBaseAttraction = "attractionBase"
)

// کلیدهای اقلامی که درصدشان از جدول‌های آیین‌نامه یا از وضعیت کارمند می‌آید.
const (
	// OilAllowanceShift قلم نوبت‌کار (تنها قلمی که به نوبت‌کاری کارمند گره خورده است).
	OilAllowanceShift = "shiftWorker"
	// OilAllowanceClimate قلم «بدی آب و هوا» که مبنای فوق‌العاده‌ها را می‌سازد.
	OilAllowanceClimate = "badClimate"
	// OilAllowanceSpecial قلم «فوق‌العادهٔ ویژه» (ردیف ۳ آیین‌نامه) که مبنای جذب را می‌سازد.
	OilAllowanceSpecial = "special"
	// OilAllowanceDeprivation قلم «محرومیت از تسهیلات زندگی» (ردیف ۴، جدول ۱).
	OilAllowanceDeprivation = "deprivation"
	// OilAllowanceTimeConsumed قلم «جبران صرف وقت اضافی» (ردیف ۶، جدول ۲).
	OilAllowanceTimeConsumed = "timeConsumed"
	// OilAllowanceAttraction قلم «جذب عمومی و تخصصی شغل» (ردیف ۷، جدول ۳).
	OilAllowanceAttraction = "attractionSpecialty"
)

// OilJobGrade یک ردیف از «جدول ۳» آیین‌نامه: درصد جذب و تخصصی شغل هر پایهٔ سمت.
//
// PublicAttract «حق جذب عمومی» و PositionRank «حق مقام» است؛ Total همان ستون «جمع (Y)»
// جدول است. پایهٔ سمت دادهٔ ساختار سازمانی است، پس فهرست در دامنه اعلام می‌شود تا فرم و
// فیش و گزارش‌ها همه از یک منبع بخوانند (همان الگوی `ClimateRegionsTable`).
type OilJobGrade struct {
	Key           string  `json:"key"`
	TitleFa       string  `json:"titleFa"`
	PublicAttract float64 `json:"publicAttract"`
	PositionRank  float64 `json:"positionRank"`
}

// Total درصد جمع ستون «جمع (Y)» جدول ۳.
func (g OilJobGrade) Total() float64 { return g.PublicAttract + g.PositionRank }

// oilJobGradeCatalog جدول ۳ آیین‌نامه — درصد جذب و تخصصی شغل بر پایهٔ پایهٔ سمت.
var oilJobGradeCatalog = []OilJobGrade{
	{Key: "6-10", TitleFa: "پایهٔ سمت ۶ تا ۱۰", PublicAttract: 4, PositionRank: 8},
	{Key: "11-15", TitleFa: "پایهٔ سمت ۱۱ تا ۱۵", PublicAttract: 8, PositionRank: 8},
	{Key: "16", TitleFa: "پایهٔ سمت ۱۶", PublicAttract: 8, PositionRank: 15},
	{Key: "17", TitleFa: "پایهٔ سمت ۱۷", PublicAttract: 8, PositionRank: 20},
	{Key: "A", TitleFa: "پایهٔ سمت A", PublicAttract: 8, PositionRank: 25},
	{Key: "B", TitleFa: "پایهٔ سمت B", PublicAttract: 8, PositionRank: 35},
	{Key: "C", TitleFa: "پایهٔ سمت C", PublicAttract: 8, PositionRank: 60},
	{Key: "D", TitleFa: "پایهٔ سمت D", PublicAttract: 8, PositionRank: 80},
	{Key: "OD", TitleFa: "پایهٔ سمت OD", PublicAttract: 8, PositionRank: 100},
}

// OilJobGrades جدول ۳ (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func OilJobGrades() []OilJobGrade {
	out := make([]OilJobGrade, len(oilJobGradeCatalog))
	copy(out, oilJobGradeCatalog)
	return out
}

// OilJobGradeByKey یافتن یک پایهٔ سمت؛ حروف لاتین بی‌توجه به بزرگی/کوچکی پذیرفته می‌شوند.
func OilJobGradeByKey(key string) (OilJobGrade, bool) {
	value := strings.TrimSpace(key)
	for _, grade := range oilJobGradeCatalog {
		if strings.EqualFold(grade.Key, value) {
			return grade, true
		}
	}
	return OilJobGrade{}, false
}

// OilResidence یک ردیف از جدول ۱ و ۲ آیین‌نامه: درصد فوق‌العاده‌ها بر پایهٔ محل سکونت.
//
// Deprivation از «جدول ۱ — فوق‌العادهٔ محرومیت از تسهیلات زندگی» و TimeConsumedDay /
// TimeConsumedShift از «جدول ۲ — فوق‌العادهٔ جبران صرف وقت اضافی» است (روزکار و نوبت‌کار
// دو ستون جدا دارند).
type OilResidence struct {
	Key               string  `json:"key"`
	TitleFa           string  `json:"titleFa"`
	Deprivation       float64 `json:"deprivation"`
	TimeConsumedDay   float64 `json:"timeConsumedDay"`
	TimeConsumedShift float64 `json:"timeConsumedShift"`
}

// oilResidenceCatalog جدول‌های ۱ و ۲ آیین‌نامه (محل سکونت → درصدها).
var oilResidenceCatalog = []OilResidence{
	{Key: "mahshahr", TitleFa: "ماهشهر", Deprivation: 25, TimeConsumedDay: 9.5, TimeConsumedShift: 10.5},
	{Key: "bandar_imam", TitleFa: "بندر امام خمینی", Deprivation: 25, TimeConsumedDay: 9.5, TimeConsumedShift: 10.5},
	{Key: "jarrahi", TitleFa: "جراحی و شهرک بعثت", Deprivation: 27.5, TimeConsumedDay: 16.5, TimeConsumedShift: 18},
	{Key: "ramshir", TitleFa: "رامشیر", Deprivation: 30, TimeConsumedDay: 25, TimeConsumedShift: 27},
	{Key: "ramhormoz", TitleFa: "رامهرمز", Deprivation: 37.5, TimeConsumedDay: 25, TimeConsumedShift: 27},
	{Key: "shadegan", TitleFa: "شادگان", Deprivation: 30, TimeConsumedDay: 27.5, TimeConsumedShift: 29},
	{Key: "omidieh", TitleFa: "امیدیه", Deprivation: 30, TimeConsumedDay: 31, TimeConsumedShift: 33},
	{Key: "abadan", TitleFa: "آبادان", Deprivation: 27.5, TimeConsumedDay: 37.5, TimeConsumedShift: 40},
}

// OilResidences جدول ۱ و ۲ (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func OilResidences() []OilResidence {
	out := make([]OilResidence, len(oilResidenceCatalog))
	copy(out, oilResidenceCatalog)
	return out
}

// OilResidenceByKey یافتن محل سکونت با کلید.
func OilResidenceByKey(key string) (OilResidence, bool) {
	value := strings.TrimSpace(key)
	for _, site := range oilResidenceCatalog {
		if site.Key == value {
			return site, true
		}
	}
	return OilResidence{}, false
}

// OilResidenceTitleFa عنوان فارسی محل سکونت؛ کلید ناشناخته خودش برمی‌گردد.
func OilResidenceTitleFa(key string) string {
	if site, ok := OilResidenceByKey(key); ok {
		return site.TitleFa
	}
	return strings.TrimSpace(key)
}

// OilJobGradeTitleFa عنوان فارسی پایهٔ سمت؛ کلید ناشناخته خودش برمی‌گردد.
func OilJobGradeTitleFa(key string) string {
	if grade, ok := OilJobGradeByKey(key); ok {
		return grade.TitleFa
	}
	return strings.TrimSpace(key)
}

// OilPayParams پارامترهای نظام پرداخت صندوق بازنشستگی نفت (از پروفایل شرکت).
type OilPayParams struct {
	// ClimatePercent درصد فوق‌العادهٔ بدی آب و هوا از پایهٔ حقوق
	// (پارامتر oilpay.climatePercent).
	ClimatePercent float64 `json:"climatePercent"`
	// Allowances فهرست فوق‌العاده‌های درصدی این نظام.
	Allowances []OilAllowance `json:"allowances"`
	// FundEmployeePercent و FundEmployerPercent کسور و سهم کارفرمای صندوق.
	FundEmployeePercent float64 `json:"fundEmployeePercent"`
	FundEmployerPercent float64 `json:"fundEmployerPercent"`
	// InsuranceEmployeePercent بیمهٔ درمان/تأمین اجتماعی سهم کارمند (اگر جدا پرداخت شود).
	InsuranceEmployeePercent float64 `json:"insuranceEmployeePercent"`
	// InsuranceCeiling سقف مبنای کسور (صفر = بی‌سقف).
	InsuranceCeiling int64 `json:"insuranceCeiling"`
	// TaxExemption معافیت مالیاتی ماهانه و TaxBrackets پله‌های مالیات.
	TaxExemption int64        `json:"taxExemption"`
	TaxBrackets  []TaxBracket `json:"taxBrackets"`
	// OvertimeRateDay و OvertimeRateShift ضریب اضافه‌کار هر ساعت به نسبت پایهٔ حقوق؛
	// آیین‌نامه برای روزکار ۰٫۰۱۴ و برای نوبت‌کار ۰٫۰۱۶ مقرر کرده است
	// (پارامترهای oilpay.overtimeRate.day و oilpay.overtimeRate.shift).
	OvertimeRateDay   float64 `json:"overtimeRateDay"`
	OvertimeRateShift float64 `json:"overtimeRateShift"`
	// DegreeAllowancePercents درصد فوق‌العادهٔ تخصصی بر اساس مدرک تحصیلی
	// (کلید مقطع → درصد). مقطع بدون درصد، مشمول این فوق‌العاده نیست.
	DegreeAllowancePercents map[string]float64 `json:"degreeAllowancePercents"`
	// JobGrades جدول ۳ (پایهٔ سمت → درصد جذب عمومی و حق مقام).
	JobGrades []OilJobGrade `json:"jobGrades"`
	// Residences جدول ۱ و ۲ (محل سکونت → درصد محرومیت و صرف وقت).
	Residences []OilResidence `json:"residences"`
	// GradeSiteMap نگاشت «گروه شغلی کارمند» (نردبان `DefaultGradeLadder`) به «پایهٔ سمت»
	// جدول ۳. تنها کاربردش پیشنهاد خودکار پایهٔ سمت در فرم فیش است؛ درصد هیچ قلمی از آن
	// محاسبه نمی‌شود. پیش‌فرض این نگاشت پیشنهاد سازمان است و با پارامتر شرکت
	// (oilpay.gradeMap.<گروه>) بازنویسی می‌شود؛ نبودِ نگاشت یعنی انتخاب پایهٔ سمت به
	// کاربر واگذار می‌شود.
	GradeSiteMap map[string]string `json:"gradeSiteMap,omitempty"`
}

// DefaultOilPayParams مقادیر پیش‌فرض (آیین‌نامهٔ فوق‌العاده‌ها) تا وقتی پارامتر شرکت
// تنظیم نشده است.
func DefaultOilPayParams() OilPayParams {
	return OilPayParams{
		ClimatePercent: 25,
		Allowances: []OilAllowance{
			{Key: OilAllowanceDeprivation, TitleFa: "فوق‌العادهٔ محرومیت از تسهیلات زندگی",
				GroupFa: OilGroupClimate, Percent: 27.5, Base: OilBaseAllowance, DrivenBy: "residence",
				NoteFa: "ردیف ۴ آیین‌نامه — درصد از جدول ۱ بر پایهٔ محل سکونت، روی مبنای فوق‌العاده‌ها."},
			{Key: OilAllowanceSpecial, TitleFa: "فوق‌العادهٔ ویژه", GroupFa: OilGroupClimate,
				Percent: 30, Base: OilBaseAllowance,
				NoteFa: "ردیف ۳ آیین‌نامه — ۳۰٪ از پایهٔ حقوق و بدی آب و هوا؛ مبنای جذب را می‌سازد."},
			{Key: OilAllowanceTimeConsumed, TitleFa: "فوق‌العادهٔ جبران صرف وقت اضافی",
				GroupFa: OilGroupClimate, Percent: 16.5, Base: OilBaseAllowance, DrivenBy: "residence",
				NoteFa: "ردیف ۶ آیین‌نامه — درصد از جدول ۲ بر پایهٔ محل سکونت (روزکار/نوبت‌کار)."},
			{Key: "workshop", TitleFa: "فوق‌العادهٔ کارگاهی", GroupFa: OilGroupClimate,
				Percent: 10, Base: OilBaseAllowance,
				NoteFa: "ردیف ۵ آیین‌نامه — شرایط محیط کار، ۱۰٪ از مبنای فوق‌العاده‌ها."},
			{Key: OilAllowanceAttraction, TitleFa: "فوق‌العادهٔ جذب عمومی و تخصصی شغل",
				GroupFa: OilGroupClimate, Percent: 0, Base: OilBaseAttraction, DrivenBy: "grade",
				NoteFa: "ردیف ۷ آیین‌نامه — درصد از جدول ۳ بر پایهٔ پایهٔ سمت، روی مبنای جذب."},
			{Key: "production", TitleFa: "فوق‌العادهٔ کار مراکز تولیدی فرآورشی",
				GroupFa: OilGroupClimate, Percent: 30, Base: OilBaseAllowance,
				NoteFa: "ردیف ۸ آیین‌نامه — ۳۰٪ از مبنای فوق‌العاده‌ها."},
			{Key: OilAllowanceShift, TitleFa: "فوق‌العادهٔ نوبت‌کاری", GroupFa: OilGroupShift,
				Percent: 35, Base: OilBaseAllowance, RequiresShift: true,
				NoteFa: "ردیف ۱۷ آیین‌نامه — تنها برای نوبت‌کار؛ به نسبت روزهای نوبت‌کاری در ماه."},
		},
		// کسور صندوق بازنشستگی نفت: سهم کارمند ۶٪ و سهم کارفرما ۱۹٪ از جمع ناخالص.
		FundEmployeePercent:      6,
		FundEmployerPercent:      19,
		InsuranceEmployeePercent: 0,
		InsuranceCeiling:         0,
		TaxExemption:             24_000_000,
		TaxBrackets: []TaxBracket{
			{Upto: 50_000_000, Rate: 0.10},
			{Upto: 100_000_000, Rate: 0.15},
			{Upto: 200_000_000, Rate: 0.20},
			{Upto: 0, Rate: 0.30},
		},
		OvertimeRateDay:   0.014,
		OvertimeRateShift: 0.016,
		// درصد مدرک تحصیلی = ستون «درصد (Z)» جدول ۳ آیین‌نامه:
		// تا دیپلم ۱۷٫۵٪ · فوق دیپلم ۲۷٪ · لیسانس ۳۸٫۵٪ · فوق لیسانس ۴۶٫۲٪ · دکترا ۵۴٪.
		// مقطع «فوق دکتری» تا اعلام مصوبهٔ جدا، درصدی ندارد.
		DegreeAllowancePercents: map[string]float64{
			"diploma":   17.5,
			"associate": 27,
			"bachelor":  38.5,
			"master":    46.2,
			"phd":       54,
		},
		JobGrades:  OilJobGrades(),
		Residences: OilResidences(),
		// نگاشت پیشنهادی گروه شغلی → پایهٔ سمت جدول ۳. این یک «پیشنهاد» است، نه قاعده:
		// فرم فیش آن را پیش‌فرض می‌گذارد و کاربر میتواند پایهٔ سمت دیگری برگزیند.
		GradeSiteMap: map[string]string{
			"employee":       "6-10",
			"expert":         "11-15",
			"senior_expert":  "16",
			"supervisor":     "17",
			"manager":        "A",
			"senior_manager": "B",
		},
	}
}

// OilPayInput ورودی محاسبهٔ فیش نظام صندوق نفت.
type OilPayInput struct {
	// BaseSalary پایهٔ حقوق ماهانه (ریال).
	BaseSalary int64 `json:"baseSalary"`
	// ClimatePercent درصد بدی آب و هوا این کارمند؛ صفر یعنی درصد پارامتر شرکت.
	ClimatePercent float64 `json:"climatePercent"`
	// AllowancePercents درصد دستی هر قلم (کلید قلم → درصد)؛ نبودِ کلید یعنی درصد مصوب.
	AllowancePercents map[string]float64 `json:"allowancePercents,omitempty"`
	// Degree کلید مدرک تحصیلی کارمند (diploma | associate | bachelor | master | phd | postdoc)؛
	// مبنای فوق‌العادهٔ تخصصی است.
	Degree string `json:"degree,omitempty"`
	// JobGrade کلید پایهٔ سمت (جدول ۳ آیین‌نامه)؛ درصد «حق جذب عمومی و حق مقام» را می‌آورد.
	JobGrade string `json:"jobGrade,omitempty"`
	// Residence کلید محل سکونت (جدول ۱ و ۲ آیین‌نامه)؛ درصد محرومیت و صرف وقت را می‌آورد.
	Residence string `json:"residence,omitempty"`
	// Shift یعنی کارمند در این ماه نوبت‌کار است (قلم نوبت‌کار و ستون نوبت‌کار جدول ۲).
	Shift bool `json:"shift"`
	// ShiftDays شمار روزهای نوبت‌کاری در ماه (صفر = تمام ماه) و DaysInMonth روزهای ماه.
	ShiftDays   float64 `json:"shiftDays"`
	DaysInMonth float64 `json:"daysInMonth"`
	// OvertimeHours ساعت اضافه‌کاری ماه.
	OvertimeHours float64 `json:"overtimeHours"`
	// OtherDeductions کسور دیگر (وام، مساعده، جریمه).
	OtherDeductions int64 `json:"otherDeductions"`
	// EmployeeID کارمند فیش؛ فقط برای این است که لایهٔ ماژول بتواند پروندهٔ ایثارگری همان
	// کارمند را بخواند و امتیازش را در Veteran بگذارد. خودِ محاسبه به آن کاری ندارد.
	EmployeeID string `json:"employeeId,omitempty"`
	// Veteran امتیاز ایثارگری تأییدشدهٔ همین کارمند (از JSON خوانده نمی‌شود تا از درخواست
	// HTTP قابل جعل نباشد).
	Veteran *VeteranEntitlement `json:"-"`
}

// OilPayLine یک سطر فیش (فوق‌العاده یا کسر).
type OilPayLine struct {
	Key      string  `json:"key"`
	TitleFa  string  `json:"titleFa"`
	GroupFa  string  `json:"groupFa"`
	Percent  float64 `json:"percent,omitempty"`
	Base     int64   `json:"base,omitempty"`
	Amount   int64   `json:"amount"`
	DetailFa string  `json:"detailFa,omitempty"`
}

// OilPayslip خروجی کامل فیش نظام پرداخت صندوق نفت.
type OilPayslip struct {
	Scheme   string `json:"scheme"`
	SchemeFa string `json:"schemeFa"`
	// پایه، بدی آب و هوا (گرمسیری)، مبنای فوق‌العاده‌ها، فوق‌العادهٔ ویژه و مبنای جذب.
	BaseSalary     int64   `json:"baseSalary"`
	ClimatePercent float64 `json:"climatePercent"`
	Climate        int64   `json:"climate"`
	AllowanceBase  int64   `json:"allowanceBase"`
	Special        int64   `json:"special"`
	AttractionBase int64   `json:"attractionBase"`
	// اقلام فوق‌العاده، اضافه‌کاری و جمع‌ها.
	AllowanceLines []OilPayLine `json:"allowanceLines"`
	OvertimeLine   *OilPayLine  `json:"overtimeLine,omitempty"`
	Allowances     int64        `json:"allowances"`
	Gross          int64        `json:"gross"`
	// کسور.
	FundEmployee      int64 `json:"fundEmployee"`
	FundEmployer      int64 `json:"fundEmployer"`
	InsuranceEmployee int64 `json:"insuranceEmployee"`
	TaxableIncome     int64 `json:"taxableIncome"`
	Tax               int64 `json:"tax"`
	OtherDeductions   int64 `json:"otherDeductions"`
	Deductions        int64 `json:"deductions"`
	Net               int64 `json:"net"`
	// Lines همهٔ سطرهای فیش به ترتیب نمایش (پایه، فوق‌العاده‌ها و کسور).
	Lines []OilPayLine `json:"lines"`
	// Degree و DegreePercent مدرک مبنای فوق‌العادهٔ تخصصی و درصد آن (خالی = مشمول نبود).
	Degree        string  `json:"degree,omitempty"`
	DegreePercent float64 `json:"degreePercent,omitempty"`
	// JobGrade و JobGradeFa پایهٔ سمت مبنا (جدول ۳) و عنوان فارسی آن.
	JobGrade   string `json:"jobGrade,omitempty"`
	JobGradeFa string `json:"jobGradeFa,omitempty"`
	// Residence و ResidenceFa محل سکونت مبنا (جدول ۱ و ۲) و عنوان فارسی آن.
	Residence   string `json:"residence,omitempty"`
	ResidenceFa string `json:"residenceFa,omitempty"`
	// امتیاز ایثارگری اعمال‌شده در همین فیش (خالی = بدون پروندهٔ تأییدشده).
	VeteranApplied       bool   `json:"veteranApplied"`
	VeteranTaxExempt     int64  `json:"veteranTaxExempt"`
	VeteranPensionExempt int64  `json:"veteranPensionExempt"`
	VeteranGroupFa       string `json:"veteranGroupFa,omitempty"`
	VeteranNoteFa        string `json:"veteranNoteFa,omitempty"`
	NoteFa               string `json:"noteFa,omitempty"`
}

// SchemeOil شناسهٔ نظام پرداخت صندوق بازنشستگی نفت.
const SchemeOil = "oil_pension_fund"

// ComputeOilPayslip فیش ماهانهٔ نظام پرداخت صندوق بازنشستگی نفت را می‌سازد.
//
// قاعده‌ها به ترتیب (شماره‌ها همان ردیف‌های آیین‌نامهٔ فوق‌العاده‌ها هستند):
//
//	۱. فوق‌العادهٔ بدی آب و هوا = درصد همان فوق‌العاده × پایهٔ حقوق (A×۲۵٪).
//	۲. مبنای فوق‌العاده‌ها = A + B.
//	۳. فوق‌العادهٔ ویژه (C) = ۳۰٪ × (A+B) و مبنای جذب = A+B+C.
//	۴. هر قلم فهرست = درصد همان قلم × مبنای همان قلم.
//	   · محرومیت و صرف وقت: درصد از جدول ۱ و ۲ بر پایهٔ محل سکونت (اگر انتخاب شده باشد).
//	   · جذب عمومی و تخصصی شغل: «جمع (Y)» جدول ۳ بر پایهٔ پایهٔ سمت.
//	۵. فوق‌العادهٔ تخصصی مدرک = ضریب Z × مبنای جذب؛ با ردیف ۷ جمعِ (A+B+C)×(Y+Z) می‌دهد.
//	۶. نوبت‌کار تنها وقتی پرداخت می‌شود که کارمند نوبت‌کار باشد و به نسبت روزهای
//	   نوبت‌کاری به روزهای ماه محاسبه می‌شود (خالی = تمام ماه).
//	۷. اضافه‌کار = پایهٔ حقوق × نرخ روزکار/نوبت‌کار (۰٫۰۱۴ / ۰٫۰۱۶) × ساعت اضافه‌کاری.
//	۸. جمع ناخالص = پایه + بدی آب و هوا + همهٔ فوق‌العاده‌ها + اضافه‌کاری.
//	۹. کسور = سهم کارمند صندوق (+ بیمه) با سقف، مالیات پله‌ای و کسور دیگر.
//	۱۰. خالص = ناخالص − کسور.
func ComputeOilPayslip(in OilPayInput, params OilPayParams) OilPayslip {
	base := in.BaseSalary
	if base < 0 {
		base = 0
	}
	climatePercent := in.ClimatePercent
	if climatePercent <= 0 {
		climatePercent = params.ClimatePercent
	}
	climate := RoundHalfUp(float64(base) * climatePercent / 100)
	allowanceBase := base + climate

	// فوق‌العادهٔ ویژه پیش از بقیه حساب می‌شود، چون «مبنای جذب» به آن وابسته است.
	specialPercent := oilAllowancePercent(in, params, OilAllowanceSpecial)
	special := RoundHalfUp(float64(allowanceBase) * specialPercent / 100)
	attractionBase := allowanceBase + special

	grade, gradeOK := OilJobGradeByKey(in.JobGrade)
	residence, residenceOK := OilResidenceByKey(in.Residence)

	out := OilPayslip{
		Scheme:          SchemeOil,
		SchemeFa:        "نظام پرداخت صندوق بازنشستگی نفت",
		BaseSalary:      base,
		ClimatePercent:  climatePercent,
		Climate:         climate,
		AllowanceBase:   allowanceBase,
		Special:         special,
		AttractionBase:  attractionBase,
		AllowanceLines:  []OilPayLine{},
		OtherDeductions: in.OtherDeductions,
	}
	if gradeOK {
		out.JobGrade, out.JobGradeFa = grade.Key, grade.TitleFa
	}
	if residenceOK {
		out.Residence, out.ResidenceFa = residence.Key, residence.TitleFa
	}

	// سطر پایه و بدی آب و هوا، تا فیش از بالا خوانا باشد.
	out.Lines = append(out.Lines,
		OilPayLine{Key: "baseSalary", TitleFa: "پایهٔ حقوق", GroupFa: OilGroupBase,
			Amount: base, DetailFa: "پایهٔ حقوق ماهانهٔ کارمند (A)"},
		OilPayLine{Key: OilAllowanceClimate, TitleFa: "فوق‌العادهٔ بدی آب و هوا (مناطق گرمسیری)",
			GroupFa: OilGroupBase, Percent: climatePercent, Base: base, Amount: climate,
			DetailFa: numberFa(climatePercent) + "٪ × پایهٔ حقوق (" + moneyFa(float64(base)) + " ریال) — ردیف ۲ آیین‌نامه"},
		OilPayLine{Key: "allowanceBase", TitleFa: "مبنای فوق‌العاده‌ها", GroupFa: OilGroupBase,
			Amount: allowanceBase, DetailFa: "پایهٔ حقوق + فوق‌العادهٔ بدی آب و هوا (A+B)"},
	)

	// فوق‌العاده‌های درصدی مصوب.
	monthDays := in.DaysInMonth
	if monthDays <= 0 {
		monthDays = 30
	}
	for _, allowance := range params.Allowances {
		percent, hasManual := oilAllowancePercentManual(in, allowance.Key)
		if !hasManual {
			percent = allowance.Percent
			// درصدهای جدولی: جدول ۱/۲ بر پایهٔ محل سکونت، جدول ۳ بر پایهٔ پایهٔ سمت.
			switch allowance.Key {
			case OilAllowanceAttraction:
				if gradeOK {
					percent = grade.Total()
				}
			case OilAllowanceDeprivation:
				if residenceOK {
					percent = residence.Deprivation
				}
			case OilAllowanceTimeConsumed:
				if residenceOK {
					percent = residence.TimeConsumedDay
					if in.Shift {
						percent = residence.TimeConsumedShift
					}
				}
			}
		}
		if percent == 0 {
			continue
		}
		if allowance.RequiresShift && !in.Shift {
			continue
		}
		lineBase := allowanceBase
		baseFa := "مبنای فوق‌العاده‌ها"
		if allowance.Base == OilBaseSalary {
			lineBase = base
			baseFa = "پایهٔ حقوق"
		} else if allowance.Base == OilBaseAttraction {
			lineBase = attractionBase
			baseFa = "مبنای جذب (پایه + گرمسیری + ویژه)"
		}
		amount := float64(lineBase) * percent / 100
		detail := numberFa(percent) + "٪ × " + baseFa + " (" + moneyFa(float64(lineBase)) + " ریال)"
		if allowance.Key == OilAllowanceAttraction && !hasManual && gradeOK {
			detail = "جدول ۳ — حق جذب عمومی " + numberFa(grade.PublicAttract) + "٪ + حق مقام " +
				numberFa(grade.PositionRank) + "٪ (" + grade.TitleFa + ") = " +
				numberFa(percent) + "٪ × " + baseFa + " (" + moneyFa(float64(lineBase)) + " ریال)"
		}
		if allowance.Key == OilAllowanceDeprivation && !hasManual && residenceOK {
			detail = "جدول ۱ — محل سکونت " + residence.TitleFa + ": " + numberFa(percent) +
				"٪ × " + baseFa + " (" + moneyFa(float64(lineBase)) + " ریال)"
		}
		if allowance.Key == OilAllowanceTimeConsumed && !hasManual && residenceOK {
			detail = "جدول ۲ — محل سکونت " + residence.TitleFa + " (" + oilWorkerKindFa(in.Shift) +
				"): " + numberFa(percent) + "٪ × " + baseFa + " (" + moneyFa(float64(lineBase)) + " ریال)"
		}
		if allowance.RequiresShift {
			days := in.ShiftDays
			if days <= 0 || days > monthDays {
				days = monthDays
			}
			amount = amount * days / monthDays
			if days < monthDays {
				detail += " × " + numberFa(days) + " از " + numberFa(monthDays) + " روز"
			}
		}
		line := OilPayLine{
			Key: allowance.Key, TitleFa: allowance.TitleFa, GroupFa: allowance.GroupFa,
			Percent: percent, Base: lineBase, Amount: RoundHalfUp(amount), DetailFa: detail,
		}
		out.AllowanceLines = append(out.AllowanceLines, line)
		out.Lines = append(out.Lines, line)
		out.Allowances += line.Amount
	}

	// فوق‌العادهٔ تخصصی بر اساس مدرک تحصیلی: ضریب Z × مبنای جذب.
	//
	// چرا روی «مبنای جذب» و نه مبنای فوق‌العاده‌ها: ردیف ۷ آیین‌نامه درصد جذب و تخصصی شغل را
	// (A+B+C)×(Y+Z) می‌داند؛ Y از پایهٔ سمت و Z از مدرک تحصیلی. برای خوانایی فیش، این دو
	// جزء در دو سطر جدا نمایش داده می‌شوند و جمعشان همان مبلغ ردیف ۷ است.
	if degree := strings.TrimSpace(in.Degree); degree != "" {
		if percent, ok := params.DegreeAllowancePercents[degree]; ok && percent > 0 {
			amount := RoundHalfUp(float64(attractionBase) * percent / 100)
			line := OilPayLine{
				Key: OilDegreeAllowance, TitleFa: OilDegreeAllowanceTitleFa(),
				GroupFa: OilGroupClimate, Percent: percent, Base: attractionBase, Amount: amount,
				DetailFa: "ضریب مدرک (Z) " + numberFa(percent) + "٪ × مبنای جذب (" +
					moneyFa(float64(attractionBase)) + " ریال)",
			}
			out.AllowanceLines = append(out.AllowanceLines, line)
			out.Lines = append(out.Lines, line)
			out.Allowances += amount
			out.Degree = degree
			out.DegreePercent = percent
		}
	}

	// اضافه‌کار: پایهٔ حقوق × نرخ مصوب هر ساعت (روزکار ۰٫۰۱۴ / نوبت‌کار ۰٫۰۱۶) × ساعت‌ها.
	if in.OvertimeHours > 0 {
		rate := params.OvertimeRateDay
		kindFa := "روزکار"
		if in.Shift {
			rate = params.OvertimeRateShift
			kindFa = "نوبت‌کار"
		}
		if rate > 0 {
			amount := RoundHalfUp(float64(base) * rate * in.OvertimeHours)
			line := OilPayLine{
				Key: "overtime", TitleFa: "اضافه‌کاری", GroupFa: OilGroupShift, Amount: amount,
				DetailFa: numberFa(in.OvertimeHours) + " ساعت × " + oilNumberFa(rate) +
					" × پایهٔ حقوق (" + kindFa + ") = " + moneyFa(float64(base)) + " ریال",
			}
			out.OvertimeLine = &line
			out.Lines = append(out.Lines, line)
			out.Allowances += amount
		}
	}

	out.Gross = base + climate + out.Allowances

	// کسور: سهم کارمند صندوق (و بیمه، اگر پارامتر داشته باشد) با سقف مبنای کسور.
	deductionBase := out.Gross
	if params.InsuranceCeiling > 0 && deductionBase > params.InsuranceCeiling {
		deductionBase = params.InsuranceCeiling
	}
	out.FundEmployee = RoundHalfUp(float64(deductionBase) * params.FundEmployeePercent / 100)
	out.FundEmployer = RoundHalfUp(float64(deductionBase) * params.FundEmployerPercent / 100)
	out.InsuranceEmployee = RoundHalfUp(float64(deductionBase) * params.InsuranceEmployeePercent / 100)
	if out.FundEmployee > 0 {
		out.Lines = append(out.Lines, OilPayLine{Key: "fundEmployee", TitleFa: "کسور صندوق بازنشستگی نفت",
			GroupFa: OilGroupDeduct, Percent: params.FundEmployeePercent, Base: deductionBase, Amount: out.FundEmployee,
			DetailFa: numberFa(params.FundEmployeePercent) + "٪ × جمع ناخالص"})
	}
	if out.InsuranceEmployee > 0 {
		out.Lines = append(out.Lines, OilPayLine{Key: "insuranceEmployee", TitleFa: "بیمهٔ سهم کارمند",
			GroupFa: OilGroupDeduct, Percent: params.InsuranceEmployeePercent, Base: deductionBase, Amount: out.InsuranceEmployee,
			DetailFa: numberFa(params.InsuranceEmployeePercent) + "٪ × جمع ناخالص"})
	}

	taxableBefore := out.Gross - out.FundEmployee - out.InsuranceEmployee - params.TaxExemption
	if taxableBefore < 0 {
		taxableBefore = 0
	}

	// امتیاز ایثارگری: کاهش کسور سهم کارمند (صندوق/بیمه) و معافیت مالیاتی. مبنای هر دو
	// تعدیل، عدد پیش از اعمال امتیاز است تا درصد معافیت روی همان مبنای قانونی بنشیند.
	relief := VeteranReliefResult{}
	if in.Veteran != nil {
		relief = ApplyVeteranRelief(VeteranReliefInput{
			TaxableIncome:     taxableBefore,
			EmployeeDeduction: out.FundEmployee + out.InsuranceEmployee,
		}, *in.Veteran)
	}
	if relief.PensionExemptAmount > 0 {
		out.FundEmployee -= relief.PensionExemptAmount
		if out.FundEmployee < 0 {
			out.FundEmployee = 0
		}
		out.Lines = append(out.Lines, OilPayLine{Key: "veteranPensionExempt",
			TitleFa: "معافیت کسور بازنشستگی (ایثارگری)", GroupFa: OilGroupDeduct,
			Amount:   -relief.PensionExemptAmount,
			DetailFa: "بر پایهٔ پروندهٔ تأییدشدهٔ امور ایثارگران"})
	}

	taxable := taxableBefore
	if relief.TaxExemptAmount > 0 {
		taxable -= relief.TaxExemptAmount
		if taxable < 0 {
			taxable = 0
		}
		out.Lines = append(out.Lines, OilPayLine{Key: "veteranTaxExempt",
			TitleFa: "معافیت مالیاتی ایثارگری", GroupFa: OilGroupDeduct,
			Amount:   -relief.TaxExemptAmount,
			DetailFa: "ماده ۵۶ قانون جامع خدمات‌رسانی به ایثارگران"})
	}
	out.VeteranApplied = relief.Applied
	out.VeteranTaxExempt = relief.TaxExemptAmount
	out.VeteranPensionExempt = relief.PensionExemptAmount
	out.VeteranNoteFa = relief.NoteFa
	if in.Veteran != nil && in.Veteran.Approved {
		out.VeteranGroupFa = in.Veteran.GroupFa
	}
	out.TaxableIncome = taxable
	out.Tax = TaxFor(taxable, params.TaxBrackets)
	if out.Tax > 0 {
		out.Lines = append(out.Lines, OilPayLine{Key: "tax", TitleFa: "مالیات",
			GroupFa: OilGroupDeduct, Amount: out.Tax,
			DetailFa: "پله‌ای بر درآمد مشمول " + moneyFa(float64(taxable)) + " ریال"})
	}
	if out.OtherDeductions > 0 {
		out.Lines = append(out.Lines, OilPayLine{Key: "otherDeductions", TitleFa: "سایر کسور",
			GroupFa: OilGroupDeduct, Amount: out.OtherDeductions, DetailFa: "وام، مساعده یا جریمه"})
	}

	out.Deductions = out.FundEmployee + out.InsuranceEmployee + out.Tax + out.OtherDeductions
	out.Net = out.Gross - out.Deductions

	note := "نظام پرداخت صندوق بازنشستگی نفت بر پایهٔ آیین‌نامهٔ فوق‌العاده‌ها: فوق‌العادهٔ بدی " +
		"آب و هوا " + numberFa(climatePercent) + "٪ از پایهٔ حقوق است (A+B = " +
		moneyFa(float64(allowanceBase)) + " ریال)، فوق‌العادهٔ ویژه " + numberFa(specialPercent) +
		"٪ از همان مبنا (مبنای جذب = " + moneyFa(float64(attractionBase)) + " ریال) و بقیهٔ " +
		"فوق‌العاده‌ها درصدی از مبنای خودشان هستند."
	if residenceOK {
		note += " درصد محرومیت و صرف وقت از جدول ۱ و ۲ برای محل سکونت " + residence.TitleFa + " آمده است."
	}
	if gradeOK {
		note += " درصد جذب عمومی و تخصصی شغل از جدول ۳ برای " + grade.TitleFa + " (" +
			numberFa(grade.Total()) + "٪) گرفته شده است."
	}
	if relief.Applied {
		note += " امتیاز ایثارگری (" + out.VeteranGroupFa + ") اعمال شد: معافیت مالیاتی " +
			moneyFa(float64(relief.TaxExemptAmount)) + " ریال و معافیت کسور " +
			moneyFa(float64(relief.PensionExemptAmount)) + " ریال."
	}
	out.NoteFa = note
	return out
}

// OilDegreeAllowance عنوان قلم فوق‌العادهٔ تخصصی بر اساس مدرک تحصیلی.
const OilDegreeAllowance = "degreeSpecialty"

// OilDegreeAllowanceTitleFa عنوان فارسی قلم فوق‌العادهٔ تخصصی.
func OilDegreeAllowanceTitleFa() string {
	return "فوق‌العادهٔ تخصصی بر اساس مدرک تحصیلی"
}

// OilSiteForGrade پایهٔ سمت پیشنهادی برای یک گروه شغلی (خالی = نگاشتی نیست).
func OilSiteForGrade(params OilPayParams, grade string) string {
	if params.GradeSiteMap == nil {
		return ""
	}
	return strings.TrimSpace(params.GradeSiteMap[strings.TrimSpace(grade)])
}

// OilDegreePercent درصد فوق‌العادهٔ تخصصی یک مقطع از پارامترها (صفر = مشمول نیست).
func OilDegreePercent(params OilPayParams, degree string) float64 {
	if params.DegreeAllowancePercents == nil {
		return 0
	}
	return params.DegreeAllowancePercents[strings.TrimSpace(degree)]
}

// OilAllowances فهرست فوق‌العاده‌های نظام صندوق برای فرم و پنل تنظیمات.
func OilAllowances(params OilPayParams) []OilAllowance {
	out := make([]OilAllowance, len(params.Allowances))
	copy(out, params.Allowances)
	return out
}

// oilAllowancePercentManual درصد دستی یک قلم از ورودی (اگر کاربر در فرم وارد کرده باشد).
func oilAllowancePercentManual(in OilPayInput, key string) (float64, bool) {
	if in.AllowancePercents == nil {
		return 0, false
	}
	value, ok := in.AllowancePercents[key]
	return value, ok
}

// oilAllowancePercent درصد نهایی یک قلم: ورودی دستی بر درصد مصوب اولویت دارد.
func oilAllowancePercent(in OilPayInput, params OilPayParams, key string) float64 {
	if value, ok := oilAllowancePercentManual(in, key); ok {
		return value
	}
	for _, allowance := range params.Allowances {
		if allowance.Key == key {
			return allowance.Percent
		}
	}
	return 0
}

// oilWorkerKindFa برچسب فارسی نوع کار برای شرح جدول ۲.
func oilWorkerKindFa(shift bool) string {
	if shift {
		return "نوبت‌کار"
	}
	return "روزکار"
}

// OilPercentFa درصد را با یک رقم اعشار در صورت لزوم برمی‌گرداند.
func OilPercentFa(v float64) string {
	return oilNumberFa(math.Round(v*10) / 10)
}

// oilNumberFa عدد را برای متن فارسی بدون صفر انتهایی برمی‌گرداند.
func oilNumberFa(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
