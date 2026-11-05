// Package rules — قاعدهٔ «محاسبهٔ فوقالعادهٔ مأموریت».
//
// چرا اینجا و نه در رابط: فوقالعادهٔ مأموریت یک قاعدهٔ پرداخت است، نه یک محاسبهٔ نمایشی.
// ضریب وعدهها، جایگزینی سهم خواب با هزینهٔ هتل/مهمانسرا، جبران استفاده از امکانات شخصی و
// هزینههای ایابوذهاب همه در دامنه اعلام میشوند تا فرم، سند چاپی و حقوق از یک منبع
// بخوانند و عدد دو جا نداشته باشد.
package rules

import (
	"math"
	"strconv"
	"strings"
)

// MissionFactors ضریب سهم هر وعده و خواب از نرخ روزانهٔ مأموریت (۰ تا ۱).
//
// جمع این چهار ضریب معمولاً ۱ است؛ یعنی نرخ روزانهٔ مأموریت بین صبحانه، نهار، شام و خواب
// تقسیم میشود. اگر پذیرایی یکی از وعدهها با میزبان باشد، همان ضریب را صفر میگذارند و
// آن وعده پرداخت نمیشود.
type MissionFactors struct {
	Breakfast float64 `json:"breakfast"`
	Lunch     float64 `json:"lunch"`
	Dinner    float64 `json:"dinner"`
	Lodging   float64 `json:"lodging"`
}

// DefaultMissionFactors ضریبهای پیش‌فرض: ۱۰٪ صبحانه، ۳۵٪ نهار، ۳۵٪ شام، ۲۰٪ خواب.
func DefaultMissionFactors() MissionFactors {
	return MissionFactors{Breakfast: 0.10, Lunch: 0.35, Dinner: 0.35, Lodging: 0.20}
}

// Total جمع چهار ضریب («جمع عوامل» در فرم).
func (f MissionFactors) Total() float64 { return f.Breakfast + f.Lunch + f.Dinner + f.Lodging }

// MissionParams پارامترهای استقرار برای محاسبهٔ مأموریت.
type MissionParams struct {
	// PerDiem فوقالعادهٔ روزانهٔ مأموریت (پارامتر شرکت: mission.perDiem).
	PerDiem float64 `json:"perDiem"`
	// MissionFactor ضریب عوامل مأموریت (نوع سفر: درون‌شهری/برون‌شهری/خارجی).
	MissionFactor float64 `json:"missionFactor"`
	// PersonalFacilitiesPercent درصد جبران استفاده از امکانات شخصی از نرخ روزانه
	// (پارامتر شرکت: mission.personalFacilitiesPercent؛ پیش‌فرض ۱۰٪).
	PersonalFacilitiesPercent float64        `json:"personalFacilitiesPercent"`
	Factors                   MissionFactors `json:"factors"`
}

// DefaultMissionParams مقادیر پیش‌فرض وقتی پارامتر شرکت تنظیم نشده است.
func DefaultMissionParams() MissionParams {
	return MissionParams{
		PerDiem:                   4_500_000,
		MissionFactor:             1,
		PersonalFacilitiesPercent: 10,
		Factors:                   DefaultMissionFactors(),
	}
}

// MissionAllowanceInput ورودی فرم محاسبهٔ مأموریت: همان فیلدهایی که کاربر پر می‌کند.
type MissionAllowanceInput struct {
	StartDate string `json:"startDate"` // تاریخ شروع (شمسی YYYY/MM/DD)
	StartTime string `json:"startTime"` // ساعت شروع (HH:MM)
	// DurationDays مدت مأموریت به روز؛ عدد اعشاری مجاز است (نیم‌روز = ۰٫۵).
	DurationDays float64 `json:"durationDays"`
	// PerDiem نرخ روزانه؛ اگر صفر باشد از پارامتر شرکت می‌آید.
	PerDiem float64 `json:"perDiem"`
	// MissionFactor ضریب عوامل مأموریت؛ صفر یعنی ۱.
	MissionFactor float64 `json:"missionFactor"`
	// Factors ضریب وعده‌ها و خواب؛ اگر همه صفر باشند، پیش‌فرض به‌کار می‌رود.
	Factors MissionFactors `json:"factors"`
	// UsePersonalFacilities استفاده از امکانات شخصی (خودرو/منزل خود) با جبران درصدی.
	UsePersonalFacilities bool `json:"usePersonalFacilities"`
	// PersonalFacilitiesAmount مبلغ جبران؛ صفر یعنی درصد پیش‌فرض × نرخ روزانه × مدت.
	PersonalFacilitiesAmount float64 `json:"personalFacilitiesAmount"`
	// UseHotel و HotelCost: اگر هتل استفاده شود، سهم خواب پرداخت نمی‌شود و به‌جایش
	// هزینهٔ واقعی هتل اضافه می‌شود.
	UseHotel  bool    `json:"useHotel"`
	HotelCost float64 `json:"hotelCost"`
	// UseGuesthouse و GuesthouseCost همان قاعدهٔ هتل برای مهمانسرا.
	UseGuesthouse  bool    `json:"useGuesthouse"`
	GuesthouseCost float64 `json:"guesthouseCost"`
	// TaxiCost هزینهٔ تاکسی از محل اقامت تا فرودگاه.
	TaxiCost float64 `json:"taxiCost"`
	// BusCost کرایهٔ اتوبوس و MileageKm/MileageRate کرایهٔ کیلومتری.
	BusCost     float64 `json:"busCost"`
	MileageKm   float64 `json:"mileageKm"`
	MileageRate float64 `json:"mileageRate"`
	// PersonalFacilitiesPercent درصد جبران امکانات شخصی از پارامتر شرکت.
	PersonalFacilitiesPercent float64 `json:"personalFacilitiesPercent"`
}

// MissionAllowanceLine یک سطر تفکیک محاسبه (برای جدول و سند چاپی).
type MissionAllowanceLine struct {
	Key      string `json:"key"`
	TitleFa  string `json:"titleFa"`
	DetailFa string `json:"detailFa,omitempty"`
	// Factor ضریب همان سطر (برای وعده‌ها و خواب)؛ مبلغ‌های ثابت صفر می‌مانند.
	Factor float64 `json:"factor,omitempty"`
	Amount float64 `json:"amount"`
}

// MissionAllowance خروجی محاسبهٔ مأموریت.
type MissionAllowance struct {
	Days        float64 `json:"days"`
	DailyRate   float64 `json:"dailyRate"`
	MealsTotal  float64 `json:"mealsTotal"`
	LodgingCost float64 `json:"lodgingCost"`
	Facilities  float64 `json:"facilities"`
	Transport   float64 `json:"transport"`
	// Total فوقالعادهٔ مأموریت قابل پرداخت.
	Total float64 `json:"total"`
	// FactorsTotal جمع ضریب‌های به‌کاررفته (برای نمایش «جمع عوامل»).
	FactorsTotal float64                `json:"factorsTotal"`
	Lines        []MissionAllowanceLine `json:"lines"`
	NoteFa       string                 `json:"noteFa,omitempty"`
}

// ComputeMissionAllowance فوقالعادهٔ مأموریت قابل پرداخت را با تفکیک سطرها می‌سازد.
//
// قاعده‌ها (همه در همین تابع اعلام شده‌اند):
//
//	۱. نرخ روزانه = نرخ پایه × ضریب عوامل مأموریت.
//	۲. سهم هر وعده و خواب = نرخ روزانه × ضریب همان جزء × مدت (روز).
//	۳. با استفاده از هتل یا مهمانسرا، سهم خواب پرداخت نمی‌شود و هزینهٔ واقعی اقامت به‌جای
//	   آن می‌آید (جلوگیری از پرداخت دوگانه).
//	۴. استفاده از امکانات شخصی، جبران درصدی از نرخ روزانه برای هر روز دارد؛ اگر مبلغ دستی
//	   وارد شود همان مبلغ پرداخت می‌شود.
//	۵. تاکسی، اتوبوس و کرایهٔ کیلومتری (کیلومتر × نرخ) مستقیم جمع می‌شوند.
//	۶. جمع قابل پرداخت = وعده‌ها + خواب/اقامت + جبران امکانات + ایابوذهاب.
func ComputeMissionAllowance(in MissionAllowanceInput) MissionAllowance {
	params := DefaultMissionParams()
	if in.PerDiem > 0 {
		params.PerDiem = in.PerDiem
	}
	if in.MissionFactor > 0 {
		params.MissionFactor = in.MissionFactor
	}
	if in.PersonalFacilitiesPercent > 0 {
		params.PersonalFacilitiesPercent = in.PersonalFacilitiesPercent
	}
	factors := in.Factors
	if factors.Total() <= 0 {
		factors = DefaultMissionFactors()
	}

	days := math.Round(in.DurationDays*100) / 100
	out := MissionAllowance{Days: days, FactorsTotal: round3(factors.Total())}
	if days <= 0 {
		out.NoteFa = "مدت مأموریت صفر است؛ برای محاسبه، تاریخ شروع و مدت مأموریت را وارد کنید."
		out.Lines = []MissionAllowanceLine{}
		return out
	}

	dailyRate := math.Round(params.PerDiem * params.MissionFactor)
	out.DailyRate = dailyRate
	lines := []MissionAllowanceLine{}

	meal := func(key, title string, factor float64) float64 {
		return math.Round(dailyRate * factor * days)
	}
	breakfast := meal("breakfast", "عامل صبحانه", factors.Breakfast)
	lunch := meal("lunch", "عامل نهار", factors.Lunch)
	dinner := meal("dinner", "عامل شام", factors.Dinner)
	lines = append(lines,
		MissionAllowanceLine{Key: "breakfast", TitleFa: "صبحانه", Factor: factors.Breakfast, Amount: breakfast,
			DetailFa: percentDetail(factors.Breakfast, dailyRate, days)},
		MissionAllowanceLine{Key: "lunch", TitleFa: "نهار", Factor: factors.Lunch, Amount: lunch,
			DetailFa: percentDetail(factors.Lunch, dailyRate, days)},
		MissionAllowanceLine{Key: "dinner", TitleFa: "شام", Factor: factors.Dinner, Amount: dinner,
			DetailFa: percentDetail(factors.Dinner, dailyRate, days)},
	)
	out.MealsTotal = breakfast + lunch + dinner

	// خواب: اگر اقامت در هتل/مهمانسرا باشد، سهم خواب پرداخت نمی‌شود.
	lodging := meal("lodging", "عامل خواب", factors.Lodging)
	if in.UseHotel || in.UseGuesthouse {
		lines = append(lines, MissionAllowanceLine{Key: "lodging", TitleFa: "سهم خواب", Factor: factors.Lodging,
			Amount: 0, DetailFa: "پرداخت نشد — اقامت در " + lodgingPlaceFa(in)})
		lodging = 0
	} else {
		lines = append(lines, MissionAllowanceLine{Key: "lodging", TitleFa: "خواب", Factor: factors.Lodging,
			Amount: lodging, DetailFa: percentDetail(factors.Lodging, dailyRate, days)})
	}
	out.LodgingCost = lodging

	if in.UseHotel && in.HotelCost > 0 {
		out.LodgingCost += math.Round(in.HotelCost)
		lines = append(lines, MissionAllowanceLine{Key: "hotel", TitleFa: "هزینهٔ هتل",
			Amount: math.Round(in.HotelCost), DetailFa: "استفاده از هتل"})
	}
	if in.UseGuesthouse && in.GuesthouseCost > 0 {
		out.LodgingCost += math.Round(in.GuesthouseCost)
		lines = append(lines, MissionAllowanceLine{Key: "guesthouse", TitleFa: "هزینهٔ مهمانسرا",
			Amount: math.Round(in.GuesthouseCost), DetailFa: "استفاده از مهمانسرا"})
	}

	// جبران استفاده از امکانات شخصی.
	if in.UsePersonalFacilities {
		amount := math.Round(in.PersonalFacilitiesAmount)
		detail := "مبلغ ثبت‌شده"
		if amount <= 0 {
			amount = math.Round(dailyRate * params.PersonalFacilitiesPercent / 100 * days)
			detail = percentDetail(params.PersonalFacilitiesPercent/100, dailyRate, days)
		}
		out.Facilities = amount
		lines = append(lines, MissionAllowanceLine{Key: "personalFacilities", TitleFa: "امکانات شخصی",
			Amount: amount, DetailFa: detail})
	}

	// ایابوذهاب: تاکسی، اتوبوس و کرایهٔ کیلومتری.
	out.Transport = 0
	if in.TaxiCost > 0 {
		out.Transport += math.Round(in.TaxiCost)
		lines = append(lines, MissionAllowanceLine{Key: "taxi", TitleFa: "تاکسی فرودگاه",
			Amount: math.Round(in.TaxiCost), DetailFa: "هزینهٔ تاکسی از محل اقامت تا فرودگاه"})
	}
	if in.BusCost > 0 {
		out.Transport += math.Round(in.BusCost)
		lines = append(lines, MissionAllowanceLine{Key: "bus", TitleFa: "کرایهٔ اتوبوس",
			Amount: math.Round(in.BusCost), DetailFa: "هزینهٔ اتوبوس"})
	}
	if in.MileageKm > 0 && in.MileageRate > 0 {
		mileage := math.Round(in.MileageKm * in.MileageRate)
		out.Transport += mileage
		lines = append(lines, MissionAllowanceLine{Key: "mileage", TitleFa: "کرایهٔ کیلومتری",
			Amount: mileage, DetailFa: numberFa(in.MileageKm) + " کیلومتر × " + moneyFa(in.MileageRate) + " ریال"})
	}

	out.Total = out.MealsTotal + out.LodgingCost + out.Facilities + out.Transport
	out.Lines = lines
	if in.StartDate != "" || in.StartTime != "" {
		out.NoteFa = "شروع: " + strings.TrimSpace(in.StartDate+" "+in.StartTime) + " — مدت: " + numberFa(days) + " روز"
	}
	return out
}

// lodgingPlaceFa محل اقامت را برای شرح سطر خواب می‌سازد.
func lodgingPlaceFa(in MissionAllowanceInput) string {
	switch {
	case in.UseHotel && in.UseGuesthouse:
		return "هتل و مهمانسرا"
	case in.UseHotel:
		return "هتل"
	case in.UseGuesthouse:
		return "مهمانسرا"
	}
	return "اقامت سازمانی"
}

// percentDetail شرح «درصد × نرخ روزانه × روز» برای سطرهای ضریب‌دار.
func percentDetail(factor, dailyRate, days float64) string {
	return numberFa(round3(factor)*100) + "٪ × " + moneyFa(dailyRate) + " ریال × " + numberFa(days) + " روز"
}

// round2 گرد کردن به دو رقم اعشار (روز).
// round3 گرد کردن به سه رقم اعشار (ضریب).
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// numberFa عدد اعشاری را بدون صفرهای انتهایی به رشته تبدیل می‌کند.
func numberFa(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// moneyFa مبلغ را با جداکنندهٔ هزارگان برمی‌گرداند (ارقام لاتین؛ فارسی‌سازی در رابط).
func moneyFa(v float64) string {
	raw := strconv.FormatInt(int64(math.Round(v)), 10)
	neg := strings.HasPrefix(raw, "-")
	raw = strings.TrimPrefix(raw, "-")
	var b strings.Builder
	for i, c := range raw {
		if i > 0 && (len(raw)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
