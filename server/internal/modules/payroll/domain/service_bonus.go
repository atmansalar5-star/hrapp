// Package domain — قواعد محاسبهٔ پاداش پایان خدمت (پاداش سنوات) کارکنان رسمی.
//
// مرجع: فایل «محاسبه پاداش پایان خدمت.xlsx» (پاداش سنوات خدمت کارکنان رسمی صنعت نفت).
// این بسته دقیقاً همان زنجیرهٔ محاسبهٔ صورت‌حساب را بازتولید می‌کند تا عدد سرور و عدد
// برگهٔ سازمان یکی باشد؛ آزمون `service_bonus_test.go` همین را با اعداد واقعی همان
// پرونده می‌سنجد.
//
// ساختار صورت‌حساب که عیناً پیاده شده:
//
//	سابقه: سال + ماه/۱۲ + روز/۳۶۵  → عدد اعشاری سابقه
//	شمال (مبنا: پایه حقوق)
//	  ویژه = پایه × ٪ویژه
//	  کارگاهی = پایه × ٪کارگاهی
//	  جذب عمومی = (پایه + ویژه) × ٪جذب عمومی
//	  جذب پایه شخصی = (پایه + ویژه) × ٪پایه شخصی
//	  تخصصی شغل = (پایه + ویژه) × ٪تخصصی
//	  (بدی آب و هوا در شمال صفر است — همان‌گونه که در صورت‌حساب مرجع)
//	جنوب (بدون بدی آب و هوا در مبنا)
//	  بدی آب و هوا = پایه × ٪بدی آب و هوا
//	  ویژه = (پایه + بدی آب و هوا) × ٪ویژه
//	  کارگاهی = (پایه + بدی آب و هوا) × ٪کارگاهی
//	  جذب عمومی = (پایه + بدی + ویژه) × ٪جذب عمومی
//	  جذب پایه شخصی = (پایه + بدی + ویژه) × ٪پایه شخصی
//	  تخصصی شغل = (پایه + بدی + ویژه) × ٪تخصصی
//	جمع هر ستون = پایه + اقلام + ترمیم حقوق
//
//	پاداش حق جذب ۸٪ = (پایه+ویژه+کارگاهی+جذب عمومی+ترمیم)شمال × سابقهٔ شمال
//	                + (پایه+بدی+ویژه+کارگاهی+جذب عمومی+ترمیم)جنوب × سابقهٔ جنوب (دو رقم اعشار)
//	پاداش پایه شخصی = (جذب پایه شخصی + تخصصی شغل)شمال × سابقهٔ شمال
//	                + همان در جنوب × سابقهٔ جنوب (کامل، بی گرد کردن)
//	مبلغ پاداش سنوات = پاداش ۸٪ + پاداش شمال + پاداش جنوب
//
// قرارداد ضرایب: هر ضریب به‌صورت «سهم» داده می‌شود (۰٫۲۵ یعنی ۲۵٪) — همان قراردادی که
// پارامترهای حقوق این ماژول از قبل دارند؛ در نتیجه هیچ‌جا تبدیل درصد/سهم لازم نیست.
//
// نکتهٔ عمدی: در صورت‌حساب مرجع، سابقهٔ جنوب برای سطر ۸٪ تا دو رقم اعشار گرد می‌شود
// (`ROUNDDOWN(A8,2)`) ولی برای سطر پایه شخصی با سابقهٔ کامل حساب می‌شود. این تفاوت
// «اشتباه» نیست — عین رفتار برگهٔ سازمان است و برای هم‌خوانی عددها حفظ شده است.
package domain

import (
	"math"
	"strconv"
	"strings"

	"hrms/internal/contracts"
)

// کلیدهای پارامتر محاسبهٔ پاداش در دفتر پارامترهای حقوق. مقدارها از همان‌جا خوانده
// می‌شوند تا مدیر بتواند بی‌نیاز از انتشار نسخهٔ تازه، درصدها را ویرایش کند.
const (
	ParamBadClimate       = "allowance.badClimate"
	ParamSpecial          = "allowance.special"
	ParamWorkshop         = "allowance.workshop"
	ParamAttractionPublic = "allowance.attractionPublic"
	ParamPersonalBase     = "allowance.personalBase"
	ParamJobSpecialty     = "allowance.jobSpecialty"
	ParamSalaryRepair     = "allowance.salaryRepair"
)

// DegreeParameterKeys جدول مرجع ضریب مدرک تحصیلی (همان جدول صورت‌حساب مرجع).
//
// این ضرایب در محاسبه فقط به‌عنوان «مرجع» نمایش داده می‌شوند؛ درصد به‌کاررفته در
// محاسبه از پارامتر تخصصی شغل می‌آید و کاربر می‌تواند آن را در فرم بازنویسی کند.
var DegreeParameterKeys = []string{
	"serviceBonus.degree.belowDiploma",
	"serviceBonus.degree.diploma",
	"serviceBonus.degree.associate",
	"serviceBonus.degree.bachelor",
	"serviceBonus.degree.master",
	"serviceBonus.degree.doctorate",
}

// DegreeTitlesFa عنوان فارسی ردیف‌های جدول مدرک تحصیلی.
var DegreeTitlesFa = map[string]string{
	"serviceBonus.degree.belowDiploma": "زیردیپلم",
	"serviceBonus.degree.diploma":      "دیپلم",
	"serviceBonus.degree.associate":    "فوق دیپلم",
	"serviceBonus.degree.bachelor":     "لیسانس",
	"serviceBonus.degree.master":       "فوق لیسانس",
	"serviceBonus.degree.doctorate":    "دکترا",
}

// RateParameterTitles عنوان فارسی پارامترهای درصدی محاسبه.
var RateParameterTitles = map[string]string{
	ParamBadClimate:       "بدی آب و هوا",
	ParamSpecial:          "ویژه",
	ParamWorkshop:         "کارگاهی",
	ParamAttractionPublic: "جذب عمومی",
	ParamPersonalBase:     "جذب (پایه شخصی)",
	ParamJobSpecialty:     "تخصصی شغل (مدرک)",
	ParamSalaryRepair:     "ترمیم حقوق",
}

// TenureYears سابقه را به عدد اعشاری تبدیل می‌کند: سال + ماه/۱۲ + روز/۳۶۵.
func TenureYears(t contracts.ServiceBonusTenure) float64 {
	return float64(t.Years) + float64(t.Months)/12 + float64(t.Days)/365
}

// AddTenures جمع دو سابقه را به تفکیک سال/ماه/روز می‌سازد (همان جمع ستون‌های مرجع).
func AddTenures(a, b contracts.ServiceBonusTenure) contracts.ServiceBonusTenure {
	years, months, days := a.Years+b.Years, a.Months+b.Months, a.Days+b.Days
	months += days / 365
	days = days % 365
	years += months / 12
	months = months % 12
	return contracts.ServiceBonusTenure{Years: years, Months: months, Days: days}
}

// TenureFromDays سابقه را از تعداد روز می‌سازد (مبنای «سابقه از پروندهٔ پرسنلی»).
func TenureFromDays(total int) contracts.ServiceBonusTenure {
	if total <= 0 {
		return contracts.ServiceBonusTenure{}
	}
	return contracts.ServiceBonusTenure{Years: total / 365, Months: (total % 365) / 30, Days: (total % 365) % 30}
}

// Truncate2 گرد کردن رو به پایین تا دو رقم اعشار — معادل ROUNDDOWN در صورت‌حساب.
func Truncate2(v float64) float64 { return math.Floor(v*100) / 100 }

// ComputeServiceBonus صورت‌حساب پاداش پایان خدمت را می‌سازد.
//
// هیچ ورودی‌ای در این تابع اختیاری نیست: درصدها، پایه، ترمیم حقوق و سابقهٔ شمال/جنوب
// همه صریح داده می‌شوند تا نتیجه بازتولیدپذیر بماند و آزمون‌پذیر باشد.
func ComputeServiceBonus(in contracts.ServiceBonusInput) contracts.ServiceBonusResult {
	r := in.Rates
	base := float64(in.BaseSalary)
	repair := float64(in.SalaryRepair)

	// ستون شمال (در صورت‌حساب مرجع: ستون D).
	nSpecial := base * r.Special
	nWorkshop := base * r.Workshop
	nAttractionPublic := (base + nSpecial) * r.AttractionPublic
	nPersonalBase := (base + nSpecial) * r.PersonalBase
	nJobSpecialty := (base + nSpecial) * r.JobSpecialty
	northLines := []contracts.ServiceBonusLine{
		{TitleFa: "پایه", Percent: 100, BaseFa: "پایه حقوق", AmountRial: base},
		{TitleFa: "بدی آب و هوا", Percent: 0, BaseFa: "پایه حقوق", AmountRial: 0},
		{TitleFa: "ویژه", Percent: r.Special * 100, BaseFa: "پایه حقوق", AmountRial: nSpecial},
		{TitleFa: "کارگاهی", Percent: r.Workshop * 100, BaseFa: "پایه حقوق", AmountRial: nWorkshop},
		{TitleFa: "جذب عمومی", Percent: r.AttractionPublic * 100, BaseFa: "پایه + ویژه", AmountRial: nAttractionPublic},
		{TitleFa: "جذب (پایه شخصی)", Percent: r.PersonalBase * 100, BaseFa: "پایه + ویژه", AmountRial: nPersonalBase},
		{TitleFa: "تخصصی شغل (مدرک)", Percent: r.JobSpecialty * 100, BaseFa: "پایه + ویژه", AmountRial: nJobSpecialty},
	}
	northTotal := base + nSpecial + nWorkshop + nAttractionPublic + nPersonalBase + nJobSpecialty + repair

	// ستون جنوب (در صورت‌حساب مرجع: ستون A).
	sBadClimate := base * r.BadClimate
	sSpecial := (base + sBadClimate) * r.Special
	sWorkshop := (base + sBadClimate) * r.Workshop
	sAttractionPublic := (base + sBadClimate + sSpecial) * r.AttractionPublic
	sPersonalBase := (base + sBadClimate + sSpecial) * r.PersonalBase
	sJobSpecialty := (base + sBadClimate + sSpecial) * r.JobSpecialty
	southLines := []contracts.ServiceBonusLine{
		{TitleFa: "پایه", Percent: 100, BaseFa: "پایه حقوق", AmountRial: base},
		{TitleFa: "بدی آب و هوا", Percent: r.BadClimate * 100, BaseFa: "پایه حقوق", AmountRial: sBadClimate},
		{TitleFa: "ویژه", Percent: r.Special * 100, BaseFa: "پایه + بدی آب و هوا", AmountRial: sSpecial},
		{TitleFa: "کارگاهی", Percent: r.Workshop * 100, BaseFa: "پایه + بدی آب و هوا", AmountRial: sWorkshop},
		{TitleFa: "جذب عمومی", Percent: r.AttractionPublic * 100, BaseFa: "پایه + بدی + ویژه", AmountRial: sAttractionPublic},
		{TitleFa: "جذب (پایه شخصی)", Percent: r.PersonalBase * 100, BaseFa: "پایه + بدی + ویژه", AmountRial: sPersonalBase},
		{TitleFa: "تخصصی شغل (مدرک)", Percent: r.JobSpecialty * 100, BaseFa: "پایه + بدی + ویژه", AmountRial: sJobSpecialty},
	}
	southTotal := base + sBadClimate + sSpecial + sWorkshop + sAttractionPublic + sPersonalBase + sJobSpecialty + repair

	northYears := TenureYears(in.North)
	southYears := TenureYears(in.South)
	southYearsUsed := Truncate2(southYears)

	// سطر ۸ درصد: پایه + ویژه + کارگاهی + جذب عمومی + ترمیم حقوق.
	northEight := base + nSpecial + nWorkshop + nAttractionPublic + repair
	southEight := base + sBadClimate + sSpecial + sWorkshop + sAttractionPublic + repair
	attractionNorth := northEight * northYears
	attractionSouth := southEight * southYearsUsed

	// سطر پایه شخصی: جذب پایه شخصی + تخصصی شغل.
	personalNorth := (nPersonalBase + nJobSpecialty) * northYears
	personalSouth := (sPersonalBase + sJobSpecialty) * southYears

	total := attractionNorth + attractionSouth + personalNorth + personalSouth
	return contracts.ServiceBonusResult{
		EmployeeID:           in.EmployeeID,
		BaseDate:             in.BaseDate,
		BaseSalary:           in.BaseSalary,
		SalaryRepair:         in.SalaryRepair,
		North:                in.North,
		South:                in.South,
		Total:                AddTenures(in.North, in.South),
		NorthYears:           northYears,
		SouthYears:           southYears,
		SouthYearsUsed:       southYearsUsed,
		NorthLines:           northLines,
		SouthLines:           southLines,
		NorthTotal:           northTotal,
		SouthTotal:           southTotal,
		AttractionBonusNorth: attractionNorth,
		AttractionBonusSouth: attractionSouth,
		AttractionBonus:      attractionNorth + attractionSouth,
		PersonalBonusNorth:   personalNorth,
		PersonalBonusSouth:   personalSouth,
		TotalRial:            total,
		HalfTotalRial:        total / 2,
		NoteFa:               in.NoteFa,
	}
}

// RatesFromParams ضرایب محاسبه را از دفتر پارامترهای حقوق می‌سازد. پارامتری که در دفتر
// نباشد از پیش‌فرض صورت‌حساب مرجع (صنعت نفت) گرفته می‌شود تا محاسبه هرگز با صفر انجام
// نشود؛ همان پیش‌فرض‌ها در بذر پارامترهای ماژول حقوق هم نوشته شده‌اند.
func RatesFromParams(params map[string]string) contracts.ServiceBonusRates {
	num := func(key string, fallback float64) float64 {
		raw := strings.TrimSpace(params[key])
		if raw == "" {
			return fallback
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fallback
		}
		return v
	}
	return contracts.ServiceBonusRates{
		BadClimate:       num(ParamBadClimate, 0.25),
		Special:          num(ParamSpecial, 0.30),
		Workshop:         num(ParamWorkshop, 0.10),
		AttractionPublic: num(ParamAttractionPublic, 0.08),
		PersonalBase:     num(ParamPersonalBase, 0.38),
		JobSpecialty:     num(ParamJobSpecialty, 0.27),
	}
}
