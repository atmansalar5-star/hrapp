// نظام جبران خدمات دستورالعمل IMS-HR-34006 — موتور محاسبهٔ سمت سرور.
//
// چرا این فایل جدا از `compensation.go` و `oilpay.go` نوشته می‌شود: این‌ها سه «مدل پرداخت»
// جدا و مستقل‌اند، نه سه صفحه از یک مدل:
//
//	payroll    مدل حقوق و دستمزد جاری سازمان (پیش‌فرض)
//	oilpay     نظام پرداخت صندوق بازنشستگی نفت
//	comp34006  دستورالعمل جبران خدمات شرکت (سند IMS-HR-34006) — همین فایل
//
// مدل ۴پی (`compensation.go`) با این سه یکی نیست: آن یک مدل «طراحی بستهٔ پرداخت» است و
// باند و میانه می‌سازد؛ این مدل، مدل *محاسبهٔ مبلغ* است و مبنایش «امتیاز × ضریب ریالی»
// است. قاعدهٔ استقلال در همه یکسان است: هیچ مدلی محاسبهٔ مدل دیگر را عوض نمی‌کند.
//
// مبانی سند که در همین فایل به قاعده تبدیل شده‌اند (M = ماده، ب = بند):
//
//	M6-1  حق گرید شغلی        = امتیاز گرید شغلی × ضریب ریالی
//	M6-1  تبصره ۲            کاهش ۵٪ حق گرید شغلی بابت هر گرید شخصی پایین‌تر از صلاحیت
//	M6-2  پایهٔ سنواتی        مبلغ مصوب سالانه (پارامتر)
//	M6-3  پست استراتژیک       = امتیاز پست × ضریب پست × ضریب ریالی
//	M6-4  گرید شخصی          = امتیاز گرید شخصی × ضریب ریالی
//	M6-5  فوق‌العادهٔ نقش      = امتیاز نقش × ضریب نوع نقش × ضریب ریالی
//	M6-6  جذب بازار کار      حداکثر معادل فوق‌العادهٔ گرید شخصی، با تصویب کمیته
//	M6-7  فوق‌العادهٔ منطقه    = امتیاز منطقه × ضریب منطقه × ضریب ریالی (+ درصد حق گرید شغلی)
//	M6-8  پاداش عملکرد       بر پایهٔ دستورالعمل مدیریت عملکرد
//	M6-9  ثابت ماهانه        مبلغ مصوب وزارت کار
//	M6-10 عائله‌مندی و اولاد  مبلغ مصوب × تعداد فرزند
//	M6-11 حق مسکن            مبلغ مصوب
//	M6-12 خواربار و بن       مبلغ مصوب
//	M6-13 تفاوت تطبیق        در طرح G2025 پرداخت نمی‌شود (پیش‌فرض صفر)
//	M6-14 اضافه‌کاری          (مزد روزانه ÷ ۷٫۳۳) × ۱٫۴ برای هر ساعت
//	M6-15 نوبت‌کاری           (حقوق پایه ÷ ۳۰) × درصد نوبت‌کاری × روزها
//	M6-16 مأموریت            معادل یک روز حقوق پایه برای هر روز مأموریت
//	M6-17 اقماری             فوق‌العادهٔ منطقه ÷ ۱۴ برای هر روز
//	M6-18 تعطیل‌کاری         ۴۰٪ روزهای هفته و ۸۰٪ جمعه، علاوه بر حقوق آن روز
//
// قاعدهٔ معماری: هر عددی که «سیاست شرکت» است پارامتر است (`comp34006.*`) و پنج جدول سند
// به‌صورت «نردبان» نگه داشته می‌شوند. چرا نردبان و نه ۲۵ عدد ثابت در کد: سند خودش برای هر
// جدول ستونِ «میزان افزایش امتیاز هر گرید» دارد (۱٫۰۵ در جدول‌های ۱ و ۵، ۱٫۱۵ در
// جدول‌های ۲ تا ۴). جدول ۱ با گرید ۱ بسته می‌شود (۳٬۱۰۰) و جدول‌های ۲ تا ۵ با عدد گِرد
// گرید آخر (۱۰٬۰۰۰ / ۴٬۰۰۰ / ۲٬۵۰۰ / ۳٬۰۰۰)؛ پس هر نردبان با همان لنگری بسته می‌شود که
// خود سند از آن ساخته شده و خانه‌هایش سلول‌به‌سلول با سند می‌خواند. تغییر مصوبهٔ سالانه هم
// فقط تغییر پارامتر است، نه انتشار نسخهٔ تازه.
package rules

import (
	"math"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// پارامترها
// ---------------------------------------------------------------------------

// Comp34006Params پارامترهای مدل دستورالعمل ۳۴۰۰۶ (کلیدها با پیشوند `comp34006.`).
type Comp34006Params struct {
	// RialFactor ضریب ریالی شرکت (ریال)؛ سند برای ۱۳۹۹ مقدار ۷٬۵۰۰ ریال را ثبت کرده و
	// هر سال با تصویب هیئت مدیره تغییر می‌کند.
	RialFactor int64 `json:"rialFactor"`
	// MaxJobGrade شمار گریدهای شغلی (سند: ۲۵ گرید؛ ۲۱ تا ۲۵ معادل A تا E).
	MaxJobGrade int `json:"maxJobGrade"`
	// JobBasePoints امتیاز حق گرید شغلی در گرید ۱ (جدول ۱: ۳٬۱۰۰) و JobGrowth نرخ رشد آن.
	// سند جدول ۱ را از گرید ۱ می‌بندد، چون گرید ۱ باید حداقل حقوق وزارت کار باشد.
	JobBasePoints float64 `json:"jobBasePoints"`
	JobGrowth     float64 `json:"jobGrowth"`
	// JobFloorAmount کف ماهانهٔ حق گرید شغلی؛ سند: گرید ۱ باید حداقل برابر حداقل حقوق
	// اعلامی وزارت کار باشد. صفر یعنی کف اعمال نشود.
	JobFloorAmount int64 `json:"jobFloorAmount"`
	// امتیاز گرید آخر در جدول ۲ (پست استراتژیک)، جدول ۳ (گرید شخصی) و جدول ۴ (نقش).
	StrategicTopPoints float64 `json:"strategicTopPoints"`
	StrategicGrowth    float64 `json:"strategicGrowth"`
	PersonalTopPoints  float64 `json:"personalTopPoints"`
	PersonalGrowth     float64 `json:"personalGrowth"`
	RoleTopPoints      float64 `json:"roleTopPoints"`
	RoleGrowth         float64 `json:"roleGrowth"`
	// امتیاز گرید آخر در سه ستون جدول ۵ (منطقه) و نرخ رشد مشترک آن‌ها.
	RegionClimateTopPoints     float64 `json:"regionClimateTopPoints"`
	RegionDeprivationTopPoints float64 `json:"regionDeprivationTopPoints"`
	RegionHardshipTopPoints    float64 `json:"regionHardshipTopPoints"`
	RegionGrowth               float64 `json:"regionGrowth"`
	// ضرایب چهار دستهٔ پست استراتژیک (سند: حیاتی ۱، کلیدی ۰٫۵، محوری ۰٫۲۵، ضروری ۰).
	PostCriticalFactor  float64 `json:"postCriticalFactor"`
	PostKeyFactor       float64 `json:"postKeyFactor"`
	PostPivotFactor     float64 `json:"postPivotFactor"`
	PostEssentialFactor float64 `json:"postEssentialFactor"`
	// ضرایب سه نوع نقش (سند: ۵۰٪، ۳۰٪، ۲۰٪) و حداکثر شمار نقش‌های همزمان.
	RoleType1Percent   float64 `json:"roleType1Percent"`
	RoleType2Percent   float64 `json:"roleType2Percent"`
	RoleType3Percent   float64 `json:"roleType3Percent"`
	MaxConcurrentRoles int     `json:"maxConcurrentRoles"`
	// فوق‌العادهٔ منطقهٔ تهران و ماهشهر (سند: تهران ۵۵٪ حق گرید شغلی و ۶۸٫۷۵٪ جدول؛
	// ماهشهر ۸۰٪ و ۱۰۰٪).
	RegionTehranJobGradePercent   float64 `json:"regionTehranJobGradePercent"`
	RegionTehranTablePercent      float64 `json:"regionTehranTablePercent"`
	RegionMahshahrJobGradePercent float64 `json:"regionMahshahrJobGradePercent"`
	RegionMahshahrTablePercent    float64 `json:"regionMahshahrTablePercent"`
	// RegionMode چگونگی ترکیب دو بخش فوق‌العادهٔ منطقه: `both` (جمع)، `table` (فقط سهم
	// جدول) یا `jobGrade` (فقط درصد حق گرید شغلی). سند این ترکیب را صریح نگفته است، پس
	// پارامتری شد تا شرکت خودش تصمیم بگیرد.
	RegionMode string `json:"regionMode"`
	// QualificationDeductPercentPerGrade کاهش حق گرید شغلی بابت هر گرید شخصی پایین‌تر
	// از صلاحیت شغل (سند: ۵٪).
	QualificationDeductPercentPerGrade float64 `json:"qualificationDeductPercentPerGrade"`
	// درصد نوبت‌کاری هر ترکیب شیفت و درصد توافقی شرکت (سند: ۱۰٪، ۱۵٪، ۲۲٫۵٪ و تبصرهٔ ۳۵٪).
	ShiftMorningEveningPercent float64 `json:"shiftMorningEveningPercent"`
	ShiftThreeShiftPercent     float64 `json:"shiftThreeShiftPercent"`
	ShiftMorningNightPercent   float64 `json:"shiftMorningNightPercent"`
	ShiftEveningNightPercent   float64 `json:"shiftEveningNightPercent"`
	ShiftCompanyPercent        float64 `json:"shiftCompanyPercent"`
	// OvertimeDivisor تقسیم‌کنندهٔ مزد روزانه به ساعتی (سند: ۷٫۳۳) و OvertimeFactor
	// ضریب هر ساعت اضافه‌کاری (سند: ۱٫۴).
	OvertimeDivisor float64 `json:"overtimeDivisor"`
	OvertimeFactor  float64 `json:"overtimeFactor"`
	// HolidayWeekdayPercent و HolidayFridayPercent درصد تعطیل‌کاری روزهای هفته و جمعه
	// (سند: ۴۰٪ و ۸۰٪) که «علاوه بر حقوق آن روز» پرداخت می‌شود.
	HolidayWeekdayPercent float64 `json:"holidayWeekdayPercent"`
	HolidayFridayPercent  float64 `json:"holidayFridayPercent"`
	// OffshoreDays مبنای روزهای اقماری (سند: ۱۴ روز).
	OffshoreDays int `json:"offshoreDays"`
	// MonthlyDays مبنای روزهای ماه در فرمول‌های سند (سند: ۳۰ روز) — مبنای تقسیم حقوق پایه
	// برای مزد روزانه، نوبت‌کاری و مأموریت.
	MonthlyDays int `json:"monthlyDays"`
	// MaxPersonalGrade بالاترین گرید شخصی (سند: ۱ تا ۲۵ در پنج مسیر حرفه‌ای).
	MaxPersonalGrade int `json:"maxPersonalGrade"`
	// MarketDefaultPercent درصد پیش‌فرض فوق‌العادهٔ جذب بازار کار وقتی مصوبه درصدی تعیین
	// نکرده باشد (سند: حداکثر معادل فوق‌العادهٔ گرید شخصی، یعنی ۱۰۰٪).
	MarketDefaultPercent float64 `json:"marketDefaultPercent"`
	// مبالغ مصوب سالانه که سند به «اعلامی هر سال» واگذار کرده است.
	SeniorityAmount int64 `json:"seniorityAmount"`
	FixedMonthly    int64 `json:"fixedMonthly"`
	ChildAllowance  int64 `json:"childAllowance"`
	HousingAmount   int64 `json:"housingAmount"`
	GroceryAmount   int64 `json:"groceryAmount"`
	// MarketMaxFactor سقف فوق‌العادهٔ جذب بازار کار به‌عنوان ضریبی از فوق‌العادهٔ گرید
	// شخصی (سند: حداکثر معادل آن).
	MarketMaxFactor float64 `json:"marketMaxFactor"`
	// PerformancePercent درصد پاداش عملکرد از حقوق پایه؛ صفر یعنی «از ماژول عملکرد»
	// خوانده شود و این موتور مبلغی پیشنهاد نکند (سند به دستورالعمل مدیریت عملکرد ارجاع می‌دهد).
	PerformancePercent float64 `json:"performancePercent"`
	// BaseIncludeFixed آیا «عدد ثابت ماهیانه» در حقوق پایه (مبنا) جمع شود؟ سند در مادهٔ ۵
	// حقوق مبنا را «حق گرید شغلی + عدد ثابت ماهیانه» و در بند ۶-۱ «حق گرید شغلی + پایهٔ
	// سنواتی» تعریف می‌کند؛ این پارامتر همان ابهام را حل می‌کند.
	BaseIncludeFixed bool `json:"baseIncludeFixed"`
	// PayrollApply آیا خروجی این مدل در صدور حکم و محاسبهٔ فیش مصرف شود.
	//
	// پیش‌فرض «خاموش» و عمدی است: این مدل مثل مدل صندوق نفت یک مدل *مستقل* است و تا
	// شرکت آن را برای خودش روشن نکند هیچ عددی از آن به حکم یا فیش راه نمی‌یابد؛ و حتی
	// با روشن‌شدنش، احکام و پرداخت‌های پیشین بازنویسی نمی‌شوند.
	PayrollApply bool `json:"payrollApply"`
}

// DefaultComp34006Params پیش‌فرض‌های آغازین از خود سند (ویرایش ۰۱، تصویب ۱۴۰۰/۰۱/۰۱).
func DefaultComp34006Params() Comp34006Params {
	return Comp34006Params{
		RialFactor:  7_500,
		MaxJobGrade: 25,

		JobBasePoints:  3_100,
		JobGrowth:      1.05,
		JobFloorAmount: 0,

		StrategicTopPoints: 10_000,
		StrategicGrowth:    1.15,
		PersonalTopPoints:  10_000,
		PersonalGrowth:     1.15,
		RoleTopPoints:      4_000,
		RoleGrowth:         1.15,

		RegionClimateTopPoints:     2_500,
		RegionDeprivationTopPoints: 2_500,
		RegionHardshipTopPoints:    3_000,
		RegionGrowth:               1.05,

		PostCriticalFactor:  1,
		PostKeyFactor:       0.5,
		PostPivotFactor:     0.25,
		PostEssentialFactor: 0,

		RoleType1Percent:   50,
		RoleType2Percent:   30,
		RoleType3Percent:   20,
		MaxConcurrentRoles: 3,

		RegionTehranJobGradePercent:   55,
		RegionTehranTablePercent:      68.75,
		RegionMahshahrJobGradePercent: 80,
		RegionMahshahrTablePercent:    100,
		RegionMode:                    "both",

		QualificationDeductPercentPerGrade: 5,

		ShiftMorningEveningPercent: 10,
		ShiftThreeShiftPercent:     15,
		ShiftMorningNightPercent:   22.5,
		ShiftEveningNightPercent:   22.5,
		ShiftCompanyPercent:        35,

		OvertimeDivisor: 7.33,
		OvertimeFactor:  1.4,

		HolidayWeekdayPercent: 40,
		HolidayFridayPercent:  80,

		OffshoreDays:         14,
		MonthlyDays:          30,
		MaxPersonalGrade:     25,
		MarketDefaultPercent: 100,

		SeniorityAmount:    0,
		FixedMonthly:       0,
		ChildAllowance:     0,
		HousingAmount:      0,
		GroceryAmount:      0,
		MarketMaxFactor:    1,
		PerformancePercent: 0,
		BaseIncludeFixed:   false,

		// «خاموش» یعنی انتخاب این مدل هیچ اثری در پرداخت جاری ندارد.
		PayrollApply: false,
	}
}

// Comp34006ParamsFromValues پارامترهای شرکت (عدد) را روی پیش‌فرض می‌نشاند؛ کلید نبوده یا
// خارج از بازهٔ معقول، پیش‌فرض را دست‌نخورده می‌گذارد.
func Comp34006ParamsFromValues(values map[string]float64) Comp34006Params {
	p := DefaultComp34006Params()
	set := func(key string, apply func(float64)) {
		if v, found := values[key]; found {
			apply(v)
		}
	}
	set("comp34006.rialFactor", func(v float64) {
		if v > 0 {
			p.RialFactor = int64(v)
		}
	})
	set("comp34006.maxJobGrade", func(v float64) {
		if v >= 1 && v <= 60 {
			p.MaxJobGrade = int(v)
		}
	})
	set("comp34006.job.basePoints", func(v float64) {
		if v > 0 {
			p.JobBasePoints = v
		}
	})
	set("comp34006.job.growth", func(v float64) {
		if v >= 1 && v <= 2 {
			p.JobGrowth = v
		}
	})
	set("comp34006.job.floorAmount", func(v float64) {
		if v >= 0 {
			p.JobFloorAmount = int64(v)
		}
	})
	set("comp34006.strategic.topPoints", func(v float64) {
		if v > 0 {
			p.StrategicTopPoints = v
		}
	})
	set("comp34006.strategic.growth", func(v float64) {
		if v >= 1 && v <= 2 {
			p.StrategicGrowth = v
		}
	})
	set("comp34006.personal.topPoints", func(v float64) {
		if v > 0 {
			p.PersonalTopPoints = v
		}
	})
	set("comp34006.personal.growth", func(v float64) {
		if v >= 1 && v <= 2 {
			p.PersonalGrowth = v
		}
	})
	set("comp34006.role.topPoints", func(v float64) {
		if v > 0 {
			p.RoleTopPoints = v
		}
	})
	set("comp34006.role.growth", func(v float64) {
		if v >= 1 && v <= 2 {
			p.RoleGrowth = v
		}
	})
	set("comp34006.region.climate.topPoints", func(v float64) {
		if v > 0 {
			p.RegionClimateTopPoints = v
		}
	})
	set("comp34006.region.deprivation.topPoints", func(v float64) {
		if v > 0 {
			p.RegionDeprivationTopPoints = v
		}
	})
	set("comp34006.region.hardship.topPoints", func(v float64) {
		if v > 0 {
			p.RegionHardshipTopPoints = v
		}
	})
	set("comp34006.region.growth", func(v float64) {
		if v >= 1 && v <= 2 {
			p.RegionGrowth = v
		}
	})
	set("comp34006.post.criticalFactor", func(v float64) {
		if v >= 0 && v <= 3 {
			p.PostCriticalFactor = v
		}
	})
	set("comp34006.post.keyFactor", func(v float64) {
		if v >= 0 && v <= 3 {
			p.PostKeyFactor = v
		}
	})
	set("comp34006.post.pivotFactor", func(v float64) {
		if v >= 0 && v <= 3 {
			p.PostPivotFactor = v
		}
	})
	set("comp34006.post.essentialFactor", func(v float64) {
		if v >= 0 && v <= 3 {
			p.PostEssentialFactor = v
		}
	})
	set("comp34006.role.type1Percent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.RoleType1Percent = v
		}
	})
	set("comp34006.role.type2Percent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.RoleType2Percent = v
		}
	})
	set("comp34006.role.type3Percent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.RoleType3Percent = v
		}
	})
	set("comp34006.role.maxConcurrent", func(v float64) {
		if v >= 0 && v <= 10 {
			p.MaxConcurrentRoles = int(v)
		}
	})
	set("comp34006.region.tehran.jobGradePercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.RegionTehranJobGradePercent = v
		}
	})
	set("comp34006.region.tehran.tablePercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.RegionTehranTablePercent = v
		}
	})
	set("comp34006.region.mahshahr.jobGradePercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.RegionMahshahrJobGradePercent = v
		}
	})
	set("comp34006.region.mahshahr.tablePercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.RegionMahshahrTablePercent = v
		}
	})
	set("comp34006.qualification.deductPercentPerGrade", func(v float64) {
		if v >= 0 && v <= 100 {
			p.QualificationDeductPercentPerGrade = v
		}
	})
	set("comp34006.shift.morningEveningPercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.ShiftMorningEveningPercent = v
		}
	})
	set("comp34006.shift.threeShiftPercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.ShiftThreeShiftPercent = v
		}
	})
	set("comp34006.shift.morningNightPercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.ShiftMorningNightPercent = v
		}
	})
	set("comp34006.shift.eveningNightPercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.ShiftEveningNightPercent = v
		}
	})
	set("comp34006.shift.companyPercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.ShiftCompanyPercent = v
		}
	})
	set("comp34006.overtime.divisor", func(v float64) {
		if v > 0 && v <= 31 {
			p.OvertimeDivisor = v
		}
	})
	set("comp34006.overtime.factor", func(v float64) {
		if v > 0 && v <= 5 {
			p.OvertimeFactor = v
		}
	})
	set("comp34006.holiday.weekdayPercent", func(v float64) {
		if v >= 0 && v <= 300 {
			p.HolidayWeekdayPercent = v
		}
	})
	set("comp34006.holiday.fridayPercent", func(v float64) {
		if v >= 0 && v <= 300 {
			p.HolidayFridayPercent = v
		}
	})
	set("comp34006.offshore.days", func(v float64) {
		if v >= 1 && v <= 30 {
			p.OffshoreDays = int(v)
		}
	})
	set("comp34006.monthlyDays", func(v float64) {
		if v >= 1 && v <= 31 {
			p.MonthlyDays = int(v)
		}
	})
	set("comp34006.maxPersonalGrade", func(v float64) {
		if v >= 1 && v <= 60 {
			p.MaxPersonalGrade = int(v)
		}
	})
	set("comp34006.market.defaultPercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.MarketDefaultPercent = v
		}
	})
	set("comp34006.seniority.amount", func(v float64) {
		if v >= 0 {
			p.SeniorityAmount = int64(v)
		}
	})
	set("comp34006.fixedMonthly.amount", func(v float64) {
		if v >= 0 {
			p.FixedMonthly = int64(v)
		}
	})
	set("comp34006.child.amount", func(v float64) {
		if v >= 0 {
			p.ChildAllowance = int64(v)
		}
	})
	set("comp34006.housing.amount", func(v float64) {
		if v >= 0 {
			p.HousingAmount = int64(v)
		}
	})
	set("comp34006.grocery.amount", func(v float64) {
		if v >= 0 {
			p.GroceryAmount = int64(v)
		}
	})
	set("comp34006.market.maxFactor", func(v float64) {
		if v >= 0 && v <= 3 {
			p.MarketMaxFactor = v
		}
	})
	set("comp34006.performance.percent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.PerformancePercent = v
		}
	})
	if v, ok := values["comp34006.region.mode"]; ok {
		switch int(v) {
		case 1:
			p.RegionMode = "table"
		case 2:
			p.RegionMode = "jobGrade"
		default:
			p.RegionMode = "both"
		}
	}
	if v, ok := values["comp34006.base.includeFixed"]; ok {
		p.BaseIncludeFixed = v != 0
	}
	if v, ok := values["comp34006.payroll.apply"]; ok {
		p.PayrollApply = v != 0
	}
	if p.RegionMode != "table" && p.RegionMode != "jobGrade" && p.RegionMode != "both" {
		p.RegionMode = "both"
	}
	return p
}

// Comp34006ParamsFromText همان نرمال‌سازی متن پارامترهای شرکت (ارقام فارسی، درصد،
// جداکنندهٔ هزارگان) پیش از تبدیل به عدد — مثل بقیهٔ پارامترهای این بسته.
func Comp34006ParamsFromText(values map[string]string) Comp34006Params {
	out := map[string]float64{}
	for key, raw := range values {
		if !strings.HasPrefix(key, "comp34006.") {
			continue
		}
		text := strings.TrimSpace(strings.NewReplacer("٪", "", "%", "", ",", "", "،", "", "٬", "", "'", "", " ", "").Replace(raw))
		if text == "" {
			continue
		}
		text = faDigitsToLatin(text)
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			continue
		}
		out[key] = value
	}
	return Comp34006ParamsFromValues(out)
}

// ---------------------------------------------------------------------------
// جدول‌های سند به‌صورت نردبان
// ---------------------------------------------------------------------------

// Comp34006Ladder یک جدول امتیاز سند: لنگر جدول (گرید ۱ یا گرید آخر)، نرخ رشد هر گرید.
//
// چرا دو نوع لنگر: سند در جدول ۱ از گرید ۱ شروع می‌کند (گرید ۱ باید حداقل حقوق وزارت کار
// باشد) و در جدول‌های ۲ تا ۵ عدد گِرد را در گرید آخر دارد؛ پس هر جدول با همان لنگری بسته
// می‌شود که خودِ سند از آن ساخته شده است تا خانه‌ها سلول‌به‌سلول بخوانند.
type Comp34006Ladder struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	// BasePoints امتیاز گرید ۱؛ اگر بزرگ‌تر از صفر باشد، لنگر همین است.
	BasePoints float64 `json:"basePoints,omitempty"`
	// TopPoints امتیاز گرید آخر؛ وقتی BasePoints صفر باشد، لنگر همین است.
	TopPoints float64 `json:"topPoints,omitempty"`
	Growth    float64 `json:"growth"`
}

// Comp34006Ladders هفت نردبان امتیاز (پنج جدول سند، جدول ۵ با سه ستون).
func Comp34006Ladders(p Comp34006Params) []Comp34006Ladder {
	return []Comp34006Ladder{
		{Key: "job", TitleFa: "امتیاز گرید شغلی (جدول ۱)", BasePoints: p.JobBasePoints, Growth: p.JobGrowth},
		{Key: "strategic", TitleFa: "حداکثر امتیاز پست استراتژیک (جدول ۲)", TopPoints: p.StrategicTopPoints, Growth: p.StrategicGrowth},
		{Key: "personal", TitleFa: "امتیاز گرید شخصی (جدول ۳)", TopPoints: p.PersonalTopPoints, Growth: p.PersonalGrowth},
		{Key: "role", TitleFa: "امتیاز فوق‌العادهٔ نقش (جدول ۴)", TopPoints: p.RoleTopPoints, Growth: p.RoleGrowth},
		{Key: "regionClimate", TitleFa: "امتیاز بدی آب و هوا (جدول ۵)", TopPoints: p.RegionClimateTopPoints, Growth: p.RegionGrowth},
		{Key: "regionDeprivation", TitleFa: "امتیاز محرومیت از تسهیلات زندگی (جدول ۵)", TopPoints: p.RegionDeprivationTopPoints, Growth: p.RegionGrowth},
		{Key: "regionHardship", TitleFa: "امتیاز صعوبت شرایط محل زندگی (جدول ۵)", TopPoints: p.RegionHardshipTopPoints, Growth: p.RegionGrowth},
	}
}

// Comp34006LadderOf یک نردبان از کلید آن؛ کلید ناشناخته یعنی نردبان گرید شغلی.
func Comp34006LadderOf(key string, p Comp34006Params) Comp34006Ladder {
	for _, l := range Comp34006Ladders(p) {
		if l.Key == key {
			return l
		}
	}
	ladders := Comp34006Ladders(p)
	return ladders[0]
}

// Comp34006Points امتیاز یک گرید در نردبان داده‌شده.
//
// با لنگر گرید ۱: امتیاز = پایه × رشد^(گرید − ۱)؛ با لنگر گرید آخر: امتیاز =
// سقف × رشد^(گرید − گرید آخر). خروجی به عدد صحیح گرد می‌شود، مثل خود جدول‌های سند.
func Comp34006Points(key string, grade int, p Comp34006Params) float64 {
	l := Comp34006LadderOf(key, p)
	growth := l.Growth
	if growth <= 0 {
		growth = 1
	}
	maxGrade := p.MaxJobGrade
	if maxGrade < 1 {
		maxGrade = 1
	}
	if grade < 1 {
		grade = 1
	}
	if grade > maxGrade {
		grade = maxGrade
	}
	if l.BasePoints > 0 {
		return math.Round(l.BasePoints * math.Pow(growth, float64(grade-1)))
	}
	top := l.TopPoints
	if top <= 0 {
		top = 1
	}
	return math.Round(top * math.Pow(growth, float64(grade-maxGrade)))
}

// Comp34006GradeTitleFa عنوان فارسی گرید شغلی؛ در گریدهای ۲۱ تا ۲۵ معادل لاتین هم
// می‌آید، همان‌طور که در جدول‌های سند نوشته شده است.
func Comp34006GradeTitleFa(grade int) string {
	switch grade {
	case 21:
		return "۲۱ (A)"
	case 22:
		return "۲۲ (B)"
	case 23:
		return "۲۳ (C)"
	case 24:
		return "۲۴ (D)"
	case 25:
		return "۲۵ (E)"
	default:
		if grade <= 0 {
			return "نامشخص"
		}
		return IntFa(grade)
	}
}

// ---------------------------------------------------------------------------
// دستهٔ پست استراتژیک، نوع نقش، منطقه و شیفت
// ---------------------------------------------------------------------------

// Comp34006PostClass یکی از چهار دستهٔ پست استراتژیک سند (بند ۶-۳).
type Comp34006PostClass struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	NoteFa  string  `json:"noteFa,omitempty"`
	Factor  float64 `json:"factor"`
}

// Comp34006PostClasses چهار دستهٔ پست با ضریب هر دسته.
func Comp34006PostClasses(p Comp34006Params) []Comp34006PostClass {
	return []Comp34006PostClass{
		{Key: "critical", TitleFa: "پست حیاتی (A)", Factor: p.PostCriticalFactor,
			NoteFa: "همهٔ پست‌های سازمانی این شغل در چارت تفصیلی حیاتی محسوب می‌شوند."},
		{Key: "key", TitleFa: "پست کلیدی (B)", Factor: p.PostKeyFactor},
		{Key: "pivot", TitleFa: "پست محوری (C)", Factor: p.PostPivotFactor},
		{Key: "essential", TitleFa: "پست ضروری (D)", Factor: p.PostEssentialFactor,
			NoteFa: "ضریب سند برای پست ضروری صفر است."},
	}
}

// Comp34006PostClassOf کلید معتبر دستهٔ پست؛ مقدار ناشناخته یعنی «ضروری».
func Comp34006PostClassOf(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "critical", "key", "pivot", "essential":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "essential"
	}
}

// Comp34006PostClassTitleFa عنوان فارسی دستهٔ پست.
func Comp34006PostClassTitleFa(key string, p Comp34006Params) string {
	normalized := Comp34006PostClassOf(key)
	for _, c := range Comp34006PostClasses(p) {
		if c.Key == normalized {
			return c.TitleFa
		}
	}
	return normalized
}

// Comp34006PostFactor ضریب دستهٔ پست.
func Comp34006PostFactor(key string, p Comp34006Params) float64 {
	normalized := Comp34006PostClassOf(key)
	for _, c := range Comp34006PostClasses(p) {
		if c.Key == normalized {
			return c.Factor
		}
	}
	return 0
}

// Comp34006RoleType یک نوع نقش (یا یک ترکیب شیفت) با درصد آن — شکل مشترک برای فهرست فرم.
type Comp34006RoleType struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	NoteFa  string  `json:"noteFa,omitempty"`
	Percent float64 `json:"percent"`
}

// Comp34006RoleTypes سه نوع نقش با درصد سند.
func Comp34006RoleTypes(p Comp34006Params) []Comp34006RoleType {
	return []Comp34006RoleType{
		{Key: "type1", TitleFa: "نقش نوع اول (پذیرش شغل دیگر با حفظ سمت)", Percent: p.RoleType1Percent},
		{Key: "type2", TitleFa: "نقش نوع دوم (عضویت در کمیسیون‌های رسمی مصوب)", Percent: p.RoleType2Percent},
		{Key: "type3", TitleFa: "نقش نوع سوم (مربی‌گری، منتورینگ و کمیته‌های رسمی)", Percent: p.RoleType3Percent,
			NoteFa: "سند این نقش را ویژهٔ گریدهای شخصی ۲۱ و بالاتر می‌داند."},
	}
}

// Comp34006RolePercent درصد نوع نقش؛ نوع ناشناخته یعنی بدون نقش (صفر).
func Comp34006RolePercent(key string, p Comp34006Params) float64 {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, rt := range Comp34006RoleTypes(p) {
		if rt.Key == normalized {
			return rt.Percent
		}
	}
	return 0
}

// Comp34006RoleTypeTitleFa عنوان فارسی نوع نقش.
func Comp34006RoleTypeTitleFa(key string, p Comp34006Params) string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, rt := range Comp34006RoleTypes(p) {
		if rt.Key == normalized {
			return rt.TitleFa
		}
	}
	return "بدون نقش"
}

// Comp34006ShiftPercent درصد نوبت‌کاری هر ترکیب شیفت.
//
// چرا نوع شیفت ورودی است: سند برای هر ترکیب شیفت درصد جدا داده و در تبصره، درصد توافقی
// شرکت (۳۵٪) را هم مجاز کرده است؛ پس موتور باید هر دو را حساب کند.
func Comp34006ShiftPercent(pattern string, p Comp34006Params) float64 {
	switch strings.ToLower(strings.TrimSpace(pattern)) {
	case "morning_evening":
		return p.ShiftMorningEveningPercent
	case "three_shift":
		return p.ShiftThreeShiftPercent
	case "morning_night":
		return p.ShiftMorningNightPercent
	case "evening_night":
		return p.ShiftEveningNightPercent
	case "company":
		return p.ShiftCompanyPercent
	default:
		return 0
	}
}

// Comp34006ShiftPatterns فهرست ترکیب‌های شیفت برای فرم.
func Comp34006ShiftPatterns(p Comp34006Params) []Comp34006RoleType {
	return []Comp34006RoleType{
		{Key: "morning_evening", TitleFa: "صبح و عصر", Percent: p.ShiftMorningEveningPercent},
		{Key: "three_shift", TitleFa: "صبح و عصر و شب", Percent: p.ShiftThreeShiftPercent},
		{Key: "morning_night", TitleFa: "صبح و شب", Percent: p.ShiftMorningNightPercent},
		{Key: "evening_night", TitleFa: "عصر و شب", Percent: p.ShiftEveningNightPercent},
		{Key: "company", TitleFa: "درصد توافقی شرکت (تبصرهٔ ۶-۱۵)", Percent: p.ShiftCompanyPercent},
	}
}

// Comp34006Region نوع منطقه (سند: تهران نوع ۲، ماهشهر نوع ۱).
type Comp34006Region struct {
	Key             string  `json:"key"`
	TitleFa         string  `json:"titleFa"`
	JobGradePercent float64 `json:"jobGradePercent"`
	TablePercent    float64 `json:"tablePercent"`
}

// Comp34006Regions دو منطقهٔ نام‌برده در سند.
func Comp34006Regions(p Comp34006Params) []Comp34006Region {
	return []Comp34006Region{
		{Key: "tehran", TitleFa: "تهران (نوع ۲)", JobGradePercent: p.RegionTehranJobGradePercent, TablePercent: p.RegionTehranTablePercent},
		{Key: "mahshahr", TitleFa: "ماهشهر (نوع ۱)", JobGradePercent: p.RegionMahshahrJobGradePercent, TablePercent: p.RegionMahshahrTablePercent},
	}
}

// Comp34006RegionOf منطقهٔ معتبر؛ مقدار ناشناخته یعنی «بدون فوق‌العادهٔ منطقه».
func Comp34006RegionOf(raw string, p Comp34006Params) (Comp34006Region, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	for _, r := range Comp34006Regions(p) {
		if r.Key == value {
			return r, true
		}
	}
	return Comp34006Region{}, false
}

// ---------------------------------------------------------------------------
// ورودی و خروجی محاسبه
// ---------------------------------------------------------------------------

// Comp34006Input ورودی محاسبهٔ بستهٔ پرداخت یک کارمند بر پایهٔ دستورالعمل ۳۴۰۰۶.
type Comp34006Input struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	// JobGrade گرید شغلی کارمند (۱ تا ۲۵).
	JobGrade int `json:"jobGrade"`
	// PersonalGrade گرید شخصی کارمند (۱ تا ۲۵) که پایهٔ فوق‌العادهٔ گرید شخصی است.
	PersonalGrade int `json:"personalGrade"`
	// QualificationGapGrades شمار گریدهای شخصی پایین‌تر از صلاحیت شغل؛ به هر گرید ۵٪ از
	// حق گرید شغلی کم می‌شود (تبصرهٔ ۲ بند ۶-۱).
	QualificationGapGrades int `json:"qualificationGapGrades"`
	// PostClass دستهٔ پست استراتژیک: critical | key | pivot | essential.
	PostClass string `json:"postClass"`
	// RoleType نوع نقش (حداکثر سه نقش در سند) و RoleCount شمار نقش‌های همزمان.
	RoleType  string `json:"roleType"`
	RoleCount int    `json:"roleCount"`
	// Region منطقه: tehran | mahshahr | خالی.
	Region string `json:"region"`
	// MarketAttractionApproved آیا فوق‌العادهٔ جذب بازار کار با تصویب کمیته پرداخت شود و
	// MarketAttractionPercent درصدی از سقف فوق‌العادهٔ گرید شخصی که مصوب شده است.
	MarketAttractionApproved bool    `json:"marketAttractionApproved"`
	MarketAttractionPercent  float64 `json:"marketAttractionPercent"`
	// PerformanceScore امتیاز ارزیابی عملکرد دوره؛ فقط وقتی پارامتر درصد پاداش عملکرد
	// روشن باشد مصرف می‌شود.
	PerformanceScore float64 `json:"performanceScore"`
	// ChildrenCount شمار فرزندان مشمول حق اولاد.
	ChildrenCount int `json:"childrenCount"`
	// ShiftPattern ترکیب شیفت و ShiftDays شمار روزهای نوبت‌کاری در ماه.
	ShiftPattern string `json:"shiftPattern"`
	ShiftDays    int    `json:"shiftDays"`
	// OvertimeHours ساعت اضافه‌کاری و HolidayHours/FridayHours ساعت تعطیل‌کاری.
	OvertimeHours float64 `json:"overtimeHours"`
	HolidayHours  float64 `json:"holidayHours"`
	FridayHours   float64 `json:"fridayHours"`
	// MissionDays روزهای مأموریت و OffshoreDays روزهای اقماری.
	MissionDays  int `json:"missionDays"`
	OffshoreDays int `json:"offshoreDays"`
	// MonthlyWorkDays مبنای روزهای ماه؛ صفر یعنی از پارامتر شرکت
	// (`comp34006.monthlyDays`، پیش‌فرض ۳۰ مثل فرمول‌های سند) خوانده شود.
	MonthlyWorkDays int `json:"monthlyWorkDays"`
}

// Comp34006Line یک سطر پرداخت با کد رسمی سند (۳۴۶۰۱ تا ۳۴۶۱۸).
type Comp34006Line struct {
	Code    string `json:"code"`
	TitleFa string `json:"titleFa"`
	Amount  int64  `json:"amount"`
	// FormulaFa صورت محاسبه، برای نمایش در فرم و سند تصمیم.
	FormulaFa string `json:"formulaFa,omitempty"`
	RefFa     string `json:"refFa,omitempty"`
}

// Comp34006Result خروجی محاسبهٔ بستهٔ دستورالعمل ۳۴۰۰۶.
type Comp34006Result struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName,omitempty"`

	JobGrade       int     `json:"jobGrade"`
	JobGradeFa     string  `json:"jobGradeFa"`
	PersonalGrade  int     `json:"personalGrade"`
	JobGradePoints float64 `json:"jobGradePoints"`

	JobGradeAllow  int64 `json:"jobGradeAllowance"`
	Seniority      int64 `json:"seniority"`
	BaseSalary     int64 `json:"baseSalary"`
	PersonalAllow  int64 `json:"personalAllowance"`
	StrategicAllow int64 `json:"strategicAllowance"`
	RoleAllow      int64 `json:"roleAllowance"`
	MarketAllow    int64 `json:"marketAllowance"`
	RegionAllow    int64 `json:"regionAllowance"`
	Performance    int64 `json:"performanceAllowance"`
	FixedMonthly   int64 `json:"fixedMonthly"`
	ChildAllow     int64 `json:"childAllowance"`
	HousingAllow   int64 `json:"housingAllowance"`
	GroceryAllow   int64 `json:"groceryAllowance"`
	AdjustmentDiff int64 `json:"adjustmentDifference"`
	Overtime       int64 `json:"overtime"`
	ShiftAllow     int64 `json:"shiftAllowance"`
	MissionAllow   int64 `json:"missionAllowance"`
	OffshoreAllow  int64 `json:"offshoreAllowance"`
	HolidayAllow   int64 `json:"holidayAllowance"`

	Gross int64           `json:"gross"`
	Lines []Comp34006Line `json:"lines"`

	PayrollApply bool     `json:"payrollApply"`
	SummaryFa    string   `json:"summaryFa"`
	WarningsFa   []string `json:"warningsFa,omitempty"`
	Valid        bool     `json:"valid"`
	MessageFa    string   `json:"messageFa,omitempty"`
}

// Comp34006Calc محاسبهٔ خط‌به‌خط بستهٔ پرداخت بر پایهٔ دستورالعمل IMS-HR-34006.
//
// ترتیب محاسبه همان ترتیب مادهٔ ۶ سند است: نخست حق گرید شغلی و پایهٔ سنواتی که «حقوق
// پایه (مبنا)» را می‌سازند، سپس فوق‌العاده‌های مبتنی بر امتیاز، و در پایان اقلام مبتنی بر
// حقوق پایه (اضافه‌کاری، نوبت‌کاری، مأموریت، اقماری، تعطیل‌کاری).
func Comp34006Calc(in Comp34006Input, p Comp34006Params) Comp34006Result {
	out := Comp34006Result{
		EmployeeID:    strings.TrimSpace(in.EmployeeID),
		EmployeeName:  strings.TrimSpace(in.EmployeeName),
		JobGrade:      in.JobGrade,
		PersonalGrade: in.PersonalGrade,
		PayrollApply:  p.PayrollApply,
	}
	if p.MaxJobGrade < 1 {
		out.MessageFa = "شمار گریدهای شغلی در پارامترهای شرکت معتبر نیست."
		return out
	}
	if in.JobGrade < 1 || in.JobGrade > p.MaxJobGrade {
		out.MessageFa = "گرید شغلی باید بین ۱ و " + IntFa(p.MaxJobGrade) + " باشد."
		return out
	}
	personal := in.PersonalGrade
	if personal < 1 {
		personal = 1
	}
	// سقف گرید شخصی جدول ۳ است؛ اگر شرکت آن را تعیین نکرده باشد، همان سقف گرید شغلی می‌شود.
	maxPersonal := p.MaxPersonalGrade
	if maxPersonal < 1 {
		maxPersonal = p.MaxJobGrade
	}
	if personal > maxPersonal {
		personal = maxPersonal
	}
	out.PersonalGrade = personal
	out.JobGradeFa = Comp34006GradeTitleFa(in.JobGrade)

	// ۳۴۶۰۱ — حق گرید شغلی: امتیاز گرید × ضریب ریالی، منهای ضریب کاهندهٔ صلاحیت.
	out.JobGradePoints = Comp34006Points("job", in.JobGrade, p)
	jobGradeAllow := float64(p.RialFactor) * out.JobGradePoints
	if gap := in.QualificationGapGrades; gap > 0 && p.QualificationDeductPercentPerGrade > 0 {
		if factor := 1 - p.QualificationDeductPercentPerGrade/100*float64(gap); factor > 0 {
			jobGradeAllow *= factor
		} else {
			jobGradeAllow = 0
		}
	}
	out.JobGradeAllow = RoundHalfUp(jobGradeAllow)
	if p.JobFloorAmount > 0 && out.JobGradeAllow < p.JobFloorAmount {
		out.JobGradeAllow = p.JobFloorAmount
	}

	// ۳۴۶۰۲ — پایهٔ سنواتی (مبلغ مصوب سالانه).
	out.Seniority = p.SeniorityAmount

	// حقوق پایه (مبنا): مادهٔ ۵ و بند ۶-۱ سند دو تعریف دارند؛ پارامتر آن را حل می‌کند.
	out.BaseSalary = out.JobGradeAllow + out.Seniority
	if p.BaseIncludeFixed {
		out.BaseSalary += p.FixedMonthly
	}

	// ۳۴۶۰۴ — فوق‌العادهٔ گرید شخصی: امتیاز گرید شخصی × ضریب ریالی.
	personalPoints := Comp34006Points("personal", personal, p)
	out.PersonalAllow = RoundHalfUp(float64(p.RialFactor) * personalPoints)

	// ۳۴۶۰۳ — فوق‌العادهٔ پست استراتژیک: امتیاز پست × ضریب پست × ضریب ریالی.
	postKey := Comp34006PostClassOf(in.PostClass)
	postPoints := Comp34006Points("strategic", in.JobGrade, p)
	out.StrategicAllow = RoundHalfUp(float64(p.RialFactor) * Comp34006PostFactor(postKey, p) * postPoints)

	// ۳۴۶۰۵ — فوق‌العادهٔ نقش: امتیاز نقش × ضریب نوع نقش × ضریب ریالی.
	roleCount := in.RoleCount
	if roleCount < 0 {
		roleCount = 0
	}
	if p.MaxConcurrentRoles > 0 && roleCount > p.MaxConcurrentRoles {
		roleCount = p.MaxConcurrentRoles
	}
	rolePercent := Comp34006RolePercent(in.RoleType, p)
	rolePoints := Comp34006Points("role", in.JobGrade, p)
	if rolePercent > 0 && roleCount > 0 {
		out.RoleAllow = RoundHalfUp(float64(p.RialFactor) * (rolePercent / 100) * rolePoints * float64(roleCount))
	}

	// ۳۴۶۰۶ — فوق‌العادهٔ جذب بازار کار: حداکثر معادل فوق‌العادهٔ گرید شخصی، با تصویب.
	if in.MarketAttractionApproved {
		percent := in.MarketAttractionPercent
		if percent <= 0 {
			percent = p.MarketDefaultPercent
		}
		if capPercent := p.MarketMaxFactor * 100; percent > capPercent {
			percent = capPercent
		}
		if percent > 0 {
			out.MarketAllow = RoundHalfUp(float64(out.PersonalAllow) * percent / 100)
		}
	}

	// ۳۴۶۰۷ — فوق‌العادهٔ منطقه: سهم جدول ۵ و/یا درصد حق گرید شغلی (پارامتر RegionMode).
	if region, ok := Comp34006RegionOf(in.Region, p); ok {
		tablePoints := Comp34006Points("regionClimate", in.JobGrade, p) +
			Comp34006Points("regionDeprivation", in.JobGrade, p) +
			Comp34006Points("regionHardship", in.JobGrade, p)
		tablePart := float64(p.RialFactor) * (region.TablePercent / 100) * tablePoints
		jobPart := float64(out.JobGradeAllow) * region.JobGradePercent / 100
		switch p.RegionMode {
		case "table":
			out.RegionAllow = RoundHalfUp(tablePart)
		case "jobGrade":
			out.RegionAllow = RoundHalfUp(jobPart)
		default:
			out.RegionAllow = RoundHalfUp(tablePart + jobPart)
		}
	}

	// ۳۴۶۰۸ — پاداش عملکرد (درصد پارامتری از حقوق پایه؛ صفر یعنی از ماژول عملکرد).
	if p.PerformancePercent > 0 && in.PerformanceScore > 0 {
		out.Performance = RoundHalfUp(float64(out.BaseSalary) * p.PerformancePercent / 100 * in.PerformanceScore / 100)
	}

	// ۳۴۶۰۹ — ثابت ماهانه، ۳۴۶۱۰ — حق عائله‌مندی و اولاد، ۳۴۶۱۱ — حق مسکن،
	// ۳۴۶۱۲ — حق خواربار و بن.
	out.FixedMonthly = p.FixedMonthly
	children := in.ChildrenCount
	if children < 0 {
		children = 0
	}
	out.ChildAllow = p.ChildAllowance * int64(children)
	out.HousingAllow = p.HousingAmount
	out.GroceryAllow = p.GroceryAmount

	// ۳۴۶۱۳ — تفاوت تطبیق: سند می‌گوید در طرح G2025 پرداخت نمی‌شود، پس صفر می‌ماند.
	out.AdjustmentDiff = 0

	// ۳۴۶۱۴ — اضافه‌کاری: (حقوق پایه ÷ روزهای ماه ÷ تقسیم‌کننده) × ضریب، برای هر ساعت.
	days := in.MonthlyWorkDays
	if days <= 0 {
		days = p.MonthlyDays
	}
	if days <= 0 {
		days = 30
	}
	divisor := p.OvertimeDivisor
	if divisor <= 0 {
		divisor = DefaultComp34006Params().OvertimeDivisor
	}
	dailyBase := float64(out.BaseSalary) / float64(days)
	hourlyBase := dailyBase / divisor
	if in.OvertimeHours > 0 {
		out.Overtime = RoundHalfUp(hourlyBase * p.OvertimeFactor * in.OvertimeHours)
	}

	// ۳۴۶۱۵ — فوق‌العادهٔ نوبت‌کاری: (حقوق پایه ÷ روزهای ماه) × درصد نوبت‌کاری × روزها.
	// مبنای روزهای ماه پارامتری است (`comp34006.monthlyDays`، سند: ۳۰).
	shiftPercent := Comp34006ShiftPercent(in.ShiftPattern, p)
	if shiftPercent > 0 && in.ShiftDays > 0 {
		out.ShiftAllow = RoundHalfUp(float64(out.BaseSalary) / float64(days) * (shiftPercent / 100) * float64(in.ShiftDays))
	}

	// ۳۴۶۱۶ — فوق‌العادهٔ مأموریت: معادل یک روز حقوق پایه برای هر روز مأموریت.
	if in.MissionDays > 0 {
		out.MissionAllow = RoundHalfUp(dailyBase * float64(in.MissionDays))
	}

	// ۳۴۶۱۷ — فوق‌العادهٔ اقماری: فوق‌العادهٔ منطقه ÷ ۱۴ برای هر روز.
	if in.OffshoreDays > 0 && out.RegionAllow > 0 {
		offshoreDays := p.OffshoreDays
		if offshoreDays <= 0 {
			offshoreDays = 14
		}
		out.OffshoreAllow = RoundHalfUp(float64(out.RegionAllow) / float64(offshoreDays) * float64(in.OffshoreDays))
	}

	// ۳۴۶۱۸ — فوق‌العادهٔ تعطیل‌کاری: ۴۰٪ روزهای هفته و ۸۰٪ جمعه، علاوه بر حقوق آن روز.
	if in.HolidayHours > 0 {
		out.HolidayAllow += RoundHalfUp(hourlyBase * in.HolidayHours * (1 + p.HolidayWeekdayPercent/100))
	}
	if in.FridayHours > 0 {
		out.HolidayAllow += RoundHalfUp(hourlyBase * in.FridayHours * (1 + p.HolidayFridayPercent/100))
	}

	out.Lines = comp34006Lines(out, in, p, postKey, personalPoints, rolePercent, shiftPercent)
	out.Gross = 0
	for _, line := range out.Lines {
		out.Gross += line.Amount
	}

	if in.OffshoreDays > 0 && out.RegionAllow == 0 {
		out.WarningsFa = append(out.WarningsFa,
			"فوق‌العادهٔ اقماری به فوق‌العادهٔ منطقه گره خورده است؛ بدون منطقه، اقماری محاسبه نمی‌شود.")
	}
	if in.QualificationGapGrades > 0 {
		out.WarningsFa = append(out.WarningsFa,
			"ضریب کاهندهٔ صلاحیت اعمال شد: "+IntFa(in.QualificationGapGrades)+" گرید × "+
				TrimFloatFa(p.QualificationDeductPercentPerGrade)+"٪.")
	}
	if !p.PayrollApply {
		out.WarningsFa = append(out.WarningsFa,
			"پارامتر «اعمال در پرداخت» این مدل خاموش است؛ این محاسبه پیشنهادی است و به حکم و فیش راه نمی‌یابد.")
	}

	out.Valid = true
	out.SummaryFa = comp34006SummaryFa(out)
	return out
}

// comp34006Lines سطرهای پرداخت با کد رسمی سند؛ اقلام صفر در فهرست نمی‌آیند تا سند تصمیم
// فقط اقلام واقعی این کارمند را نشان دهد.
func comp34006Lines(
	out Comp34006Result, in Comp34006Input, p Comp34006Params,
	postKey string, personalPoints, rolePercent, shiftPercent float64,
) []Comp34006Line {
	lines := []Comp34006Line{{
		Code: "34601", TitleFa: "حق گرید شغلی", Amount: out.JobGradeAllow,
		FormulaFa: IntFa(int64(out.JobGradePoints)) + " امتیاز × ضریب ریالی " + IntMoneyFa(p.RialFactor),
		RefFa:     "بند ۶-۱ دستورالعمل (امتیاز گرید × ضریب ریالی)",
	}}
	if out.Seniority > 0 {
		lines = append(lines, Comp34006Line{Code: "34602", TitleFa: "پایهٔ سنواتی", Amount: out.Seniority,
			RefFa: "بند ۶-۲ دستورالعمل"})
	}
	if out.StrategicAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34603", TitleFa: "فوق‌العادهٔ پست استراتژیک (حق مقام)", Amount: out.StrategicAllow,
			FormulaFa: Comp34006PostClassTitleFa(postKey, p) + " × ضریب " + TrimFloatFa(Comp34006PostFactor(postKey, p)),
			RefFa:     "بند ۶-۳ دستورالعمل"})
	}
	if out.PersonalAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34604", TitleFa: "فوق‌العادهٔ گرید شخصی", Amount: out.PersonalAllow,
			FormulaFa: IntFa(int64(personalPoints)) + " امتیاز × ضریب ریالی " + IntMoneyFa(p.RialFactor),
			RefFa:     "بند ۶-۴ دستورالعمل"})
	}
	if out.RoleAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34605", TitleFa: "فوق‌العادهٔ نقش", Amount: out.RoleAllow,
			FormulaFa: Comp34006RoleTypeTitleFa(in.RoleType, p) + " × " + TrimFloatFa(rolePercent) + "٪",
			RefFa:     "بند ۶-۵ دستورالعمل"})
	}
	if out.MarketAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34606", TitleFa: "فوق‌العادهٔ جذب بازار کار", Amount: out.MarketAllow,
			RefFa: "بند ۶-۶ دستورالعمل (حداکثر معادل فوق‌العادهٔ گرید شخصی)"})
	}
	if out.RegionAllow > 0 {
		region, _ := Comp34006RegionOf(in.Region, p)
		lines = append(lines, Comp34006Line{Code: "34607", TitleFa: "فوق‌العادهٔ منطقه", Amount: out.RegionAllow,
			FormulaFa: region.TitleFa + " — جدول ۵ با " + TrimFloatFa(region.TablePercent) + "٪ و " +
				TrimFloatFa(region.JobGradePercent) + "٪ حق گرید شغلی («" + p.RegionMode + "»)",
			RefFa: "بند ۶-۷ دستورالعمل"})
	}
	if out.Performance > 0 {
		lines = append(lines, Comp34006Line{Code: "34608", TitleFa: "پاداش عملکرد", Amount: out.Performance,
			RefFa: "بند ۶-۸ دستورالعمل"})
	}
	if out.FixedMonthly > 0 {
		lines = append(lines, Comp34006Line{Code: "34609", TitleFa: "ثابت ماهانه", Amount: out.FixedMonthly,
			RefFa: "بند ۶-۹ دستورالعمل"})
	}
	if out.ChildAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34610", TitleFa: "حق عائله‌مندی و اولاد", Amount: out.ChildAllow,
			RefFa: "بند ۶-۱۰ دستورالعمل"})
	}
	if out.HousingAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34611", TitleFa: "حق مسکن", Amount: out.HousingAllow,
			RefFa: "بند ۶-۱۱ دستورالعمل"})
	}
	if out.GroceryAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34612", TitleFa: "حق خواربار و بن", Amount: out.GroceryAllow,
			RefFa: "بند ۶-۱۲ دستورالعمل"})
	}
	// ۳۴۶۱۳ تفاوت تطبیق در سند صفر است؛ اگر روزی پرداخت شود همین سطر ظاهر می‌شود.
	if out.AdjustmentDiff > 0 {
		lines = append(lines, Comp34006Line{Code: "34613", TitleFa: "تفاوت تطبیق", Amount: out.AdjustmentDiff,
			RefFa: "بند ۶-۱۳ دستورالعمل (در طرح G2025 پرداخت نمی‌شود)"})
	}
	if out.Overtime > 0 {
		lines = append(lines, Comp34006Line{Code: "34614", TitleFa: "اضافه‌کاری", Amount: out.Overtime,
			FormulaFa: TrimFloatFa(in.OvertimeHours) + " ساعت، هر ساعت (حقوق پایه ÷ " +
				TrimFloatFa(p.OvertimeDivisor) + ") با ضریب " + TrimFloatFa(p.OvertimeFactor),
			RefFa: "بند ۶-۱۴ دستورالعمل"})
	}
	if out.ShiftAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34615", TitleFa: "فوق‌العادهٔ نوبت‌کاری", Amount: out.ShiftAllow,
			FormulaFa: IntFa(in.ShiftDays) + " روز با " + TrimFloatFa(shiftPercent) + "٪ حقوق پایه",
			RefFa:     "بند ۶-۱۵ دستورالعمل"})
	}
	if out.MissionAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34616", TitleFa: "فوق‌العادهٔ مأموریت", Amount: out.MissionAllow,
			FormulaFa: IntFa(in.MissionDays) + " روز، هر روز معادل حقوق پایهٔ روزانه",
			RefFa:     "بند ۶-۱۶ دستورالعمل"})
	}
	if out.OffshoreAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34617", TitleFa: "فوق‌العادهٔ اقماری", Amount: out.OffshoreAllow,
			FormulaFa: IntFa(in.OffshoreDays) + " روز، هر روز یک‌چهاردهم فوق‌العادهٔ منطقه (" +
				IntFa(p.OffshoreDays) + " روز مبنا)",
			RefFa: "بند ۶-۱۷ دستورالعمل"})
	}
	if out.HolidayAllow > 0 {
		lines = append(lines, Comp34006Line{Code: "34618", TitleFa: "فوق‌العادهٔ تعطیل‌کاری", Amount: out.HolidayAllow,
			FormulaFa: TrimFloatFa(p.HolidayWeekdayPercent) + "٪ روزهای هفته و " +
				TrimFloatFa(p.HolidayFridayPercent) + "٪ جمعه، علاوه بر حقوق آن روز",
			RefFa: "بند ۶-۱۸ دستورالعمل"})
	}
	return lines
}

// comp34006SummaryFa جمع‌بندی یک‌خطی فارسی برای فرم و سند.
func comp34006SummaryFa(r Comp34006Result) string {
	parts := []string{
		"گرید شغلی " + r.JobGradeFa,
		"حق گرید شغلی: " + IntMoneyFa(r.JobGradeAllow),
		"حقوق پایه: " + IntMoneyFa(r.BaseSalary),
	}
	if r.RegionAllow > 0 {
		parts = append(parts, "منطقه: "+IntMoneyFa(r.RegionAllow))
	}
	parts = append(parts, "جمع اقلام: "+IntMoneyFa(r.Gross))
	return strings.Join(parts, "؛ ") + "."
}

// Comp34006IndependenceFa متن رسمی «استقلال مدل ۳۴۰۰۶» — همان چیزی که در فرم، سند و
// مستندات تکرار می‌شود تا مرز مدل‌های پرداخت سازمان روشن بماند.
func Comp34006IndependenceFa(apply bool) string {
	common := "نظام جبران خدمات دستورالعمل ۳۴۰۰۶ (IMS-HR-34006) یک مدل پرداخت مستقل است؛ " +
		"در کنار مدل حقوق و دستمزد جاری و مدل صندوق نفت زندگی می‌کند و اقلام و جدول‌های آن " +
		"جداگانه محاسبه و نگهداری می‌شوند."
	if apply {
		return common + " در این استقرار پارامتر «اعمال در پرداخت» روشن است، پس از این پس " +
			"محاسبهٔ این مدل می‌تواند مبنای صدور حکم تازه شود؛ احکام و پرداخت‌های پیشین بر پایهٔ " +
			"مدل پیشین خودشان معتبر می‌مانند و بازنویسی نمی‌شوند."
	}
	return common + " اکنون «اعمال در پرداخت» خاموش است: انتخاب این مدل برای یک کارمند فقط " +
		"روش محاسبه را ثبت می‌کند و هیچ عددی از آن به حکم یا فیش راه نمی‌یابد تا شرکت خودش " +
		"آن را روشن کند."
}

// Comp34006InstructionFa راهنمای فارسی شروع کار با این مدل (ترتیب ثابت‌کردن داده‌ها).
func Comp34006InstructionFa() []string {
	return []string{
		"۱. ضریب ریالی سال را در پارامترهای شرکت ثبت کنید؛ همهٔ اقلام امتیازی از همین ضریب ساخته می‌شوند.",
		"۲. جدول‌های ۱ تا ۵ سند را بازبینی کنید: سقف امتیاز گرید آخر و نرخ رشد هر جدول را با مصوبهٔ شرکت تطبیق دهید.",
		"۳. گرید شغلی و گرید شخصی هر کارمند را ثبت کنید؛ این دو، مبنای حق گرید شغلی و فوق‌العادهٔ گرید شخصی‌اند.",
		"۴. دستهٔ پست استراتژیک هر پست (حیاتی، کلیدی، محوری، ضروری) را در شناسنامهٔ پست تعیین کنید.",
		"۵. نوع نقش‌های مستمر (نوع اول، دوم و سوم) را برای افراد دارای نقش ثبت کنید.",
		"۶. منطقهٔ محل خدمت (تهران، ماهشهر) را مشخص کنید تا فوق‌العادهٔ منطقه و اقماری محاسبه شود.",
		"۷. مبالغ مصوب سالانه (پایهٔ سنواتی، ثابت ماهانه، حق مسکن، خواربار و بن، حق اولاد) را وارد کنید.",
		"۸. برای هر کارمند مشمول، در «روش پرداخت کارکنان» مدل «دستورالعمل ۳۴۰۰۶» را انتخاب کنید.",
		"۹. تا وقتی پارامتر «اعمال در پرداخت» روشن نشده است، محاسبهٔ این مدل فقط پیشنهادی است و فیش تغییر نمی‌کند.",
	}
}

// Comp34006FactorTable جدول خلاصهٔ پارامترها و جدول‌ها برای نمایش در فرم.
func Comp34006FactorTable(p Comp34006Params) []CompensationLine {
	out := []CompensationLine{
		{Key: "rialfactor", TitleFa: "ضریب ریالی سال", ValueFa: IntMoneyFa(p.RialFactor),
			RefFa: "پارامتر شرکت comp34006.rialFactor"},
	}
	for _, l := range Comp34006Ladders(p) {
		out = append(out, CompensationLine{
			Key:     "ladder." + l.Key,
			TitleFa: l.TitleFa,
			ValueFa: "گرید ۱: " + TrimFloatFa(Comp34006Points(l.Key, 1, p)) + " امتیاز · رشد " +
				TrimFloatFa(l.Growth) + " · گرید " + IntFa(p.MaxJobGrade) + ": " +
				TrimFloatFa(Comp34006Points(l.Key, p.MaxJobGrade, p)) + " امتیاز",
			RefFa: "جدول‌های سند IMS-HR-34006",
		})
	}
	for _, c := range Comp34006PostClasses(p) {
		out = append(out, CompensationLine{
			Key: "post." + c.Key, TitleFa: "ضریب " + c.TitleFa,
			ValueFa: TrimFloatFa(c.Factor), RefFa: "بند ۶-۳ دستورالعمل",
		})
	}
	for _, rt := range Comp34006RoleTypes(p) {
		out = append(out, CompensationLine{
			Key: "role." + rt.Key, TitleFa: rt.TitleFa,
			ValueFa: TrimFloatFa(rt.Percent) + "٪", RefFa: "بند ۶-۵ دستورالعمل",
		})
	}
	for _, r := range Comp34006Regions(p) {
		out = append(out, CompensationLine{
			Key: "region." + r.Key, TitleFa: "منطقهٔ " + r.TitleFa,
			ValueFa: TrimFloatFa(r.TablePercent) + "٪ جدول ۵ + " + TrimFloatFa(r.JobGradePercent) + "٪ حق گرید شغلی",
			RefFa:   "بند ۶-۷ دستورالعمل",
		})
	}
	for _, s := range Comp34006ShiftPatterns(p) {
		out = append(out, CompensationLine{
			Key: "shift." + s.Key, TitleFa: "نوبت‌کاری " + s.TitleFa,
			ValueFa: TrimFloatFa(s.Percent) + "٪", RefFa: "بند ۶-۱۵ دستورالعمل",
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// فیش ماهانه
// ---------------------------------------------------------------------------

// Comp34006PayslipInput ورودی فیش ماهانهٔ نظام ۳۴۰۰۶.
type Comp34006PayslipInput struct {
	Comp34006Input
	// PeriodFa دورهٔ فیش به شمسی (YYYY/MM)؛ همان چیزی که روی فیش چاپ می‌شود.
	PeriodFa string `json:"periodFa"`
	// OtherDeductions سایر کسور ماه (مساعده، اقساط وام و مانند آن).
	OtherDeductions int64 `json:"otherDeductions"`
	// Veteran امتیاز ایثارگری تأییدشدهٔ همین کارمند.
	//
	// چرا از JSON خوانده نمی‌شود: مثل فیش نظام جاری، از پروندهٔ تأییدشدهٔ ماژول امور
	// ایثارگران پر می‌شود و نباید از بدنهٔ درخواست قابل جعل باشد.
	Veteran *VeteranEntitlement `json:"-"`
}

// Comp34006Payslip فیش ماهانهٔ یک کارمند در نظام دستورالعمل ۳۴۰۰۶.
//
// ساختار فیش همان ساختار سند است: نخست اقلام بستانکار (کدهای ۳۴۶۰۱ تا ۳۴۶۱۸) که جمعشان
// ناخالص را می‌سازد، سپس کسور قانونی (بیمهٔ سهم کارمند و مالیات) و سایر کسور، و در پایان
// خالص پرداختی. قاعدهٔ بیمه و مالیات از همان تابع مشترک سه نظام پرداخت می‌آید
// (`StatutoryDeductions`) تا با فیش نظام جاری و صندوق نفت یک‌جور بماند.
type Comp34006Payslip struct {
	Scheme   string `json:"scheme"`
	SchemeFa string `json:"schemeFa"`
	PeriodFa string `json:"periodFa"`

	EmployeeID    string `json:"employeeId"`
	EmployeeName  string `json:"employeeName,omitempty"`
	JobGrade      int    `json:"jobGrade"`
	JobGradeFa    string `json:"jobGradeFa"`
	PersonalGrade int    `json:"personalGrade"`

	// Earnings اقلام بستانکار با کد رسمی دستورالعمل.
	Earnings   []Comp34006Line `json:"earnings"`
	Gross      int64           `json:"gross"`
	BaseSalary int64           `json:"baseSalary"`

	// Deductions کسور: بیمهٔ سهم کارمند، مالیات و سایر کسور.
	Deductions        []Comp34006Line `json:"deductions"`
	InsuranceBase     int64           `json:"insuranceBase"`
	InsuranceEmployee int64           `json:"insuranceEmployee"`
	InsuranceEmployer int64           `json:"insuranceEmployer"`
	TaxableIncome     int64           `json:"taxableIncome"`
	Tax               int64           `json:"tax"`
	OtherDeductions   int64           `json:"otherDeductions"`
	TotalDeductions   int64           `json:"totalDeductions"`
	Net               int64           `json:"net"`
	// EmployerCost هزینهٔ تمام‌شدهٔ کارفرما: ناخالص + سهم کارفرمای بیمه.
	EmployerCost int64 `json:"employerCost"`

	VeteranApplied       bool   `json:"veteranApplied"`
	VeteranTaxExempt     int64  `json:"veteranTaxExempt"`
	VeteranPensionExempt int64  `json:"veteranPensionExempt"`
	VeteranNoteFa        string `json:"veteranNoteFa,omitempty"`

	PayrollApply bool     `json:"payrollApply"`
	SummaryFa    string   `json:"summaryFa"`
	WarningsFa   []string `json:"warningsFa,omitempty"`
	Valid        bool     `json:"valid"`
	MessageFa    string   `json:"messageFa,omitempty"`
}

// کلیدهای سطر کسور در فیش (کد قانونی ندارند، پس کلید خوانا می‌گیرند).
const (
	Comp34006DeductionInsurance = "INS"
	Comp34006DeductionTax       = "TAX"
	Comp34006DeductionOther     = "OTHER"
)

// Comp34006PayslipFor فیش ماهانهٔ یک کارمند را از بستهٔ پرداخت این مدل می‌سازد.
func Comp34006PayslipFor(in Comp34006PayslipInput, cp Comp34006Params, pp PayrollParams) Comp34006Payslip {
	calc := Comp34006Calc(in.Comp34006Input, cp)
	out := Comp34006Payslip{
		Scheme:        "comp34006",
		SchemeFa:      "نظام جبران خدمات دستورالعمل ۳۴۰۰۶ (IMS-HR-34006)",
		PeriodFa:      strings.TrimSpace(in.PeriodFa),
		EmployeeID:    calc.EmployeeID,
		EmployeeName:  calc.EmployeeName,
		JobGrade:      calc.JobGrade,
		JobGradeFa:    calc.JobGradeFa,
		PersonalGrade: calc.PersonalGrade,
		PayrollApply:  cp.PayrollApply,
	}
	if !calc.Valid {
		out.MessageFa = calc.MessageFa
		return out
	}
	out.Earnings = calc.Lines
	out.Gross = calc.Gross
	out.BaseSalary = calc.BaseSalary

	// کسور قانونی از همان قاعدهٔ مشترک سه نظام پرداخت.
	statutory := StatutoryDeductions(out.Gross, pp, in.Veteran)
	out.InsuranceBase = statutory.InsuranceBase
	out.InsuranceEmployee = statutory.InsuranceEmployee
	out.InsuranceEmployer = statutory.InsuranceEmployer
	out.TaxableIncome = statutory.TaxableIncome
	out.Tax = statutory.Tax
	out.VeteranApplied = statutory.VeteranApplied
	out.VeteranTaxExempt = statutory.VeteranTaxExempt
	out.VeteranPensionExempt = statutory.VeteranPensionExempt
	out.VeteranNoteFa = statutory.VeteranNoteFa

	other := in.OtherDeductions
	if other < 0 {
		other = 0
	}
	out.OtherDeductions = other
	out.TotalDeductions = out.InsuranceEmployee + out.Tax + other
	out.Net = out.Gross - out.TotalDeductions
	out.EmployerCost = out.Gross + out.InsuranceEmployer

	out.Deductions = []Comp34006Line{
		{Code: Comp34006DeductionInsurance, TitleFa: "بیمهٔ سهم کارمند",
			Amount: out.InsuranceEmployee,
			FormulaFa: "مبنای بیمه " + IntMoneyFa(out.InsuranceBase) + " با نرخ " +
				TrimFloatFa(math.Round(pp.InsuranceEmployeeRate*10000)/100) + "٪",
			RefFa: "پارامترهای سازمانی payroll.insurance.*"},
		{Code: Comp34006DeductionTax, TitleFa: "مالیات بر درآمد",
			Amount:    out.Tax,
			FormulaFa: "درآمد مشمول " + IntMoneyFa(out.TaxableIncome) + " با پله‌های پارامتری",
			RefFa:     "پارامترهای سازمانی payroll.tax.*"},
	}
	if other > 0 {
		out.Deductions = append(out.Deductions, Comp34006Line{Code: Comp34006DeductionOther,
			TitleFa: "سایر کسور (مساعده، اقساط و …)", Amount: other,
			RefFa: "ثبت دستی در همین فیش"})
	}

	out.WarningsFa = append(out.WarningsFa, calc.WarningsFa...)
	if out.VeteranApplied {
		out.WarningsFa = append(out.WarningsFa, statutory.VeteranNoteFa)
	}
	if out.PeriodFa == "" {
		out.WarningsFa = append(out.WarningsFa, "دورهٔ فیش ثبت نشده است؛ پیش از صدور، دوره را مشخص کنید.")
	}
	if out.Net < 0 {
		out.WarningsFa = append(out.WarningsFa,
			"خالص پرداختی منفی است؛ اقلام بستانکار و کسور را بازبینی کنید.")
	}

	out.Valid = true
	out.SummaryFa = comp34006PayslipSummaryFa(out)
	return out
}

// comp34006PayslipSummaryFa جمع‌بندی یک‌خطی فیش برای فرم و سند.
func comp34006PayslipSummaryFa(s Comp34006Payslip) string {
	parts := []string{}
	if s.PeriodFa != "" {
		parts = append(parts, "دورهٔ "+s.PeriodFa)
	}
	parts = append(parts,
		"گرید شغلی "+s.JobGradeFa,
		"ناخالص: "+IntMoneyFa(s.Gross),
		"کسور: "+IntMoneyFa(s.TotalDeductions),
		"خالص پرداختی: "+IntMoneyFa(s.Net),
	)
	return strings.Join(parts, "؛ ") + "."
}

// Comp34006PayslipInstructionFa متن فارسی روش صدور فیش ماهانه در این نظام.
func Comp34006PayslipInstructionFa() []string {
	return []string{
		"۱. دادهٔ ماه را کامل کنید: روزهای نوبت‌کاری، ساعت اضافه‌کاری و تعطیل‌کاری، روزهای مأموریت و اقماری از ماژول حضور و از مجوزهای همان ماه.",
		"۲. گرید شغلی و گرید شخصی، دستهٔ پست، نوع نقش و منطقهٔ هر کارمند را بازبینی کنید؛ این‌ها پایهٔ اقلام امتیازی‌اند.",
		"۳. فیش را محاسبه کنید؛ هر سطر با کد رسمی دستورالعمل و بند مرجعش نمایش داده می‌شود.",
		"۴. کسور قانونی (بیمهٔ سهم کارمند و مالیات) از قاعدهٔ مشترک سه نظام پرداخت محاسبه می‌شود؛ سایر کسور (مساعده و اقساط) را در همان فیش وارد کنید.",
		"۵. فیش را ثبت و سپس تأیید، صدور و پرداخت کنید تا سابقهٔ ماه در پروندهٔ کارمند بماند.",
		"۶. تا وقتی پارامتر «اعمال در پرداخت» روشن نشده، فیش این نظام پیشنهادی است و جای فیش جاری را نمی‌گیرد.",
	}
}
