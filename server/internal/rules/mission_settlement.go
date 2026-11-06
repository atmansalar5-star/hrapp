// Package rules — «تسویهٔ مأموریت»: قبض‌ها و فاکتورهای تأییدشده، اعزام پزشکی و مأموریت خارج از کشور.
//
// چرا جدا از `ComputeMissionAllowance`: آن تابع فقط «فوقالعادهٔ روزانهٔ ریالی» را می‌سازد؛ این
// فایل لایهٔ تسویه روی آن است — اسناد پشتیبان (قبض تاکسی و فاکتورهای مورد تأیید)، اعزام
// پزشکی (هزینه بر پایهٔ بیمار یا همراه بیمار و امکانات اسکان) و مأموریت خارج از کشور
// (پرداخت دوگانهٔ ریالی و ارزی با نرخ تسعیر).
//
// الگو: ساختار دستورالعمل مأموریت شرکت ملی نفت ایران مبنای این چیدمان است (فوقالعادهٔ روزانه،
// اسکان، ایاب‌وذهاب و اسناد پشتیبان). نرخ‌ها و درصدها همه پارامتریک‌اند تا با جزئیات دقیق
// دستورالعمل جایگزین شوند و هیچ عددی در کد ثابت نماند.

package rules

import (
	"math"
	"strings"
)

// MissionDocument سند پشتیبان مأموریت: قبض تاکسی، فاکتور هتل، بلیت و مانند آن.
//
// فقط سند «تأییدشده» در تسویه می‌آید؛ سند در گردش یا ردشده در فهرست می‌ماند ولی پرداخت
// نمی‌شود، چون مبنای مالی سازمان تأیید کارشناس است.
type MissionDocument struct {
	Key      string  `json:"key"` // شناسهٔ سند (برای پیگیری در رکورد ضمائم)
	Kind     string  `json:"kind"`
	TitleFa  string  `json:"titleFa,omitempty"`
	DocNo    string  `json:"docNo"`
	DocDate  string  `json:"docDate"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	// Approved تأیید کارشناس/مدیر مالی؛ ApprovedBy نام تأییدکننده.
	Approved   bool   `json:"approved"`
	ApprovedBy string `json:"approvedBy,omitempty"`
	// Payer پرداخت‌کنندهٔ هزینه: company (سازمان) | employee (کارمند) | insurer (بیمه).
	Payer  string `json:"payer,omitempty"`
	NoteFa string `json:"noteFa,omitempty"`
}

// MissionSettlementInput ورودی تسویهٔ مأموریت: فوقالعادهٔ روزانه + اسناد + بخش‌های ویژه.
type MissionSettlementInput struct {
	// فیلدهای محاسبهٔ روزانهٔ ریالی.
	MissionAllowanceInput
	// MissionType نوع مأموریت: administrative | medical | foreign.
	MissionType string `json:"missionType"`

	// Documents قبض‌ها و فاکتورهای پشتیبان (قبض تاکسی، فاکتور هتل، بلیت و...).
	Documents []MissionDocument `json:"documents"`

	// --- اعزام پزشکی ---
	// Beneficiary مشمول هزینه: patient (بیمار) یا companion (همراه بیمار).
	Beneficiary string `json:"beneficiary"`
	// AccommodationType نوع اسکان و AccommodationCost هزینهٔ واقعی آن.
	AccommodationType string  `json:"accommodationType"`
	AccommodationCost float64 `json:"accommodationCost"`
	// CompanionCount شمار همراهان و CompanionSharePercent سهم وعده‌های همراه از پارامتر شرکت.
	CompanionCount        int     `json:"companionCount"`
	CompanionSharePercent float64 `json:"companionSharePercent"`

	// --- مأموریت خارج از کشور ---
	// Currency ارز پرداخت (EUR/USD/AED/GBP)، FxRate نرخ تسعیر هر واحد ارز به ریال.
	Currency string  `json:"currency"`
	FxRate   float64 `json:"fxRate"`
	// FxPerDiem فوقالعادهٔ روزانهٔ ارزی و FxAmount سایر مبالغ ارزی (بلیت/اقامت خارجی).
	FxPerDiem float64 `json:"fxPerDiem"`
	FxAmount  float64 `json:"fxAmount"`
	// RialAmount مبالغ ریالی همان مأموریت (مسیر داخلی، اسکان ریالی).
	RialAmount float64 `json:"rialAmount"`
}

// MissionSettlement خروجی تسویه: تفکیک سطرها + جمع ریالی + جمع ارزی و معادل ریالی آن.
type MissionSettlement struct {
	MissionType   string           `json:"missionType"`
	MissionTypeFa string           `json:"missionTypeFa"`
	Allowance     MissionAllowance `json:"allowance"`
	// DocumentsTotal جمع اسناد تأییدشدهٔ ریالی؛ DocumentsFxTotal جمع اسناد ارزی همان ارز.
	DocumentsTotal   float64 `json:"documentsTotal"`
	DocumentsFxTotal float64 `json:"documentsFxTotal"`
	Accommodation    float64 `json:"accommodation"`
	// Medical و Foreign فقط برای نوع همان مأموریت پر می‌شوند.
	Medical *MissionMedicalInfo `json:"medical,omitempty"`
	Foreign *MissionForeignInfo `json:"foreign,omitempty"`
	// TotalRial مبلغ قابل پرداخت ریالی و TotalFx ارزی؛ TotalRialEquivalent معادل ریالی ارز.
	TotalRial           float64                `json:"totalRial"`
	TotalFx             float64                `json:"totalFx"`
	TotalRialEquivalent float64                `json:"totalRialEquivalent"`
	Lines               []MissionAllowanceLine `json:"lines"`
	// PendingDocuments شمار اسنادی که هنوز تأیید نشده‌اند (در جمع نمی‌آیند).
	PendingDocuments int    `json:"pendingDocuments"`
	NoteFa           string `json:"noteFa,omitempty"`
}

// MissionMedicalInfo اطلاعات محاسبهٔ اعزام پزشکی.
type MissionMedicalInfo struct {
	Beneficiary     string `json:"beneficiary"`
	BeneficiaryFa   string `json:"beneficiaryFa"`
	Accommodation   string `json:"accommodationType"`
	AccommodationFa string `json:"accommodationFa"`
	// MealsPayable آیا سهم وعده‌ها برای این مشمول پرداخت می‌شود.
	MealsPayable bool    `json:"mealsPayable"`
	MealsAmount  float64 `json:"mealsAmount"`
	NoteFa       string  `json:"noteFa,omitempty"`
}

// MissionForeignInfo اطلاعات محاسبهٔ مأموریت خارج از کشور.
type MissionForeignInfo struct {
	Currency       string  `json:"currency"`
	CurrencyFa     string  `json:"currencyFa"`
	FxRate         float64 `json:"fxRate"`
	FxPerDiem      float64 `json:"fxPerDiem"`
	FxDailyTotal   float64 `json:"fxDailyTotal"`
	FxOther        float64 `json:"fxOther"`
	RialAmount     float64 `json:"rialAmount"`
	RialEquivalent float64 `json:"rialEquivalent"`
	NoteFa         string  `json:"noteFa,omitempty"`
}

// missionTypeFa عنوان فارسی انواع مأموریت (هم‌خوان با کاتالوگ دامنه).
func missionTypeFa(key string) string {
	switch strings.TrimSpace(key) {
	case "administrative":
		return "مأموریت اداری"
	case "medical":
		return "اعزام پزشکی"
	case "foreign":
		return "مأموریت خارج از کشور"
	}
	return strings.TrimSpace(key)
}

// beneficiaryFa عنوان فارسی مشمول هزینهٔ اعزام پزشکی.
func beneficiaryFa(key string) string {
	switch strings.TrimSpace(key) {
	case "patient":
		return "بیمار"
	case "companion":
		return "همراه بیمار"
	}
	return strings.TrimSpace(key)
}

// ComputeMissionSettlement تسویهٔ کامل مأموریت را می‌سازد.
//
// قاعده‌ها (به ترتیب):
//
//	۱. فوقالعادهٔ روزانهٔ ریالی با همان تابع `ComputeMissionAllowance` ساخته می‌شود.
//	۲. اسناد پشتیبان: فقط اسناد «تأییدشده» جمع می‌شوند؛ قبض تاکسی و کرایهٔ کیلومتری در
//	   ایاب‌وذهاب، فاکتور هتل/مهمانسرا در اقامت و بقیه در «سایر فاکتورهای تأییدشده».
//	   اسناد ارزی از جمع ریالی جدا می‌شوند.
//	۳. اعزام پزشکی: هزینه بر پایهٔ مشمول است — بیمار: همهٔ اجزا؛ همراه بیمار: اسکان و
//	   ایاب‌وذهاب (سهم وعده‌ها فقط اگر پارامتر شرکت درصدی داده باشد). اسکان واقعی
//	   (هتل/بیمارستان/…) جای سهم خواب می‌نشیند.
//	۴. مأموریت خارج از کشور: فوقالعادهٔ ارزی (روزانهٔ ارزی × مدت) + مبالغ ارزی، در کنار
//	   مبلغ ریالی؛ معادل ریالی با نرخ تسعیر حساب می‌شود تا ثبت مالی یک عدد داشته باشد.
//	۵. جمع ریالی = فوقالعادهٔ روزانه + اقامت + اسناد ریالی تأییدشده + مبلغ ریالی مأموریت.
func ComputeMissionSettlement(in MissionSettlementInput) MissionSettlement {
	kind := strings.TrimSpace(in.MissionType)
	if kind == "" {
		kind = "administrative"
	}
	// مأموریت خارج از کشور با ارز کار می‌کند؛ سهم وعده‌ها روی نرخ ارزی اعمال نمی‌شود مگر
	// پارامتر ارزی جدا داده شود، پس محاسبهٔ ریالی برای مسیر داخلی همان مأموریت است.
	allowance := ComputeMissionAllowance(in.MissionAllowanceInput)

	out := MissionSettlement{
		MissionType:   kind,
		MissionTypeFa: missionTypeFa(kind),
		Allowance:     allowance,
		Lines:         append([]MissionAllowanceLine{}, allowance.Lines...),
	}
	// اسناد پشتیبان (قبض تاکسی و فاکتورهای تأییدشده).
	for i := range in.Documents {
		doc := in.Documents[i]
		amount := math.Round(doc.Amount)
		title := doc.TitleFa
		if strings.TrimSpace(title) == "" {
			title = documentKindFa(doc.Kind)
		}
		if !doc.Approved {
			out.PendingDocuments++
			out.Lines = append(out.Lines, MissionAllowanceLine{
				Key: "doc:" + doc.Key, TitleFa: title + " (تأییدنشده)", Amount: 0,
				DetailFa: "در انتظار تأیید — در جمع نمی‌آید" + documentDetail(doc),
			})
			continue
		}
		if isFxCurrency(doc.Currency) {
			out.DocumentsFxTotal += amount
			out.Lines = append(out.Lines, MissionAllowanceLine{
				Key: "doc:" + doc.Key, TitleFa: title + " (ارزی)", Amount: amount,
				DetailFa: documentDetail(doc) + " — " + strings.ToUpper(doc.Currency),
			})
			continue
		}
		out.DocumentsTotal += amount
		out.Lines = append(out.Lines, MissionAllowanceLine{
			Key: "doc:" + doc.Key, TitleFa: title, Amount: amount,
			DetailFa: documentDetail(doc),
		})
	}

	// اعزام پزشکی.
	if kind == "medical" {
		info := MissionMedicalInfo{
			Beneficiary:     in.Beneficiary,
			BeneficiaryFa:   beneficiaryFa(in.Beneficiary),
			Accommodation:   in.AccommodationType,
			AccommodationFa: accommodationFa(in.AccommodationType),
		}
		if info.BeneficiaryFa == "" {
			info.BeneficiaryFa = "بیمار"
		}
		// اسکان واقعی: جای سهم خواب می‌نشیند (پرداخت دوگانه نمی‌شود).
		if in.AccommodationCost > 0 {
			out.Accommodation = math.Round(in.AccommodationCost)
			// سهم خواب از محاسبهٔ روزانه حذف می‌شود.
			out.Allowance = dropLodgingShare(out.Allowance)
			out.Lines = append(out.Lines, MissionAllowanceLine{
				Key: "medicalAccommodation", TitleFa: "هزینهٔ اسکان " + info.AccommodationFa,
				Amount: out.Accommodation, DetailFa: "جای سهم خواب پرداخت می‌شود",
			})
		}
		// همراه بیمار: سهم وعده‌ها فقط با درصد مصوب شرکت.
		if in.Beneficiary == "companion" {
			share := in.CompanionSharePercent
			if share <= 0 {
				info.MealsPayable = false
				info.MealsAmount = 0
				out.Allowance = dropMealsShare(out.Allowance)
				out.Lines = append(out.Lines, MissionAllowanceLine{
					Key: "companionMeals", TitleFa: "سهم وعده‌های همراه بیمار", Amount: 0,
					DetailFa: "پرداخت نمی‌شود — هزینه بر پایهٔ همراه بیمار فقط اسکان و ایاب‌وذهاب است",
				})
			} else {
				meals := math.Round(out.Allowance.MealsTotal * share / 100)
				info.MealsPayable = true
				info.MealsAmount = meals
				out.Allowance.MealsTotal = meals
				out.Lines = append(out.Lines, MissionAllowanceLine{
					Key: "companionMeals", TitleFa: "سهم وعده‌های همراه بیمار", Amount: meals,
					DetailFa: percentDetail(share/100, out.Allowance.DailyRate, out.Allowance.Days) + " از جمع وعده‌ها",
				})
			}
			info.NoteFa = "همراه بیمار: اسکان و ایاب‌وذهاب پرداخت می‌شود؛ سهم وعده‌ها تابع پارامتر شرکت است."
		} else {
			info.MealsPayable = true
			info.MealsAmount = out.Allowance.MealsTotal
			info.NoteFa = "بیمار: همهٔ اجزای فوقالعادهٔ روزانه، اسکان و ایاب‌وذهاب پرداخت می‌شود."
		}
		out.Medical = &info
	}

	// مأموریت خارج از کشور: فوقالعادهٔ روزانهٔ ارزی + مبالغ ارزی، در کنار مبلغ ریالی.
	if kind == "foreign" {
		currency := strings.ToUpper(strings.TrimSpace(in.Currency))
		if currency == "" {
			currency = "EUR"
		}
		fxPerDiem := math.Round(in.FxPerDiem)
		days := out.Allowance.Days
		fxDaily := math.Round(fxPerDiem * days)
		info := MissionForeignInfo{
			Currency:     currency,
			CurrencyFa:   currencyFa(currency),
			FxRate:       in.FxRate,
			FxPerDiem:    fxPerDiem,
			FxDailyTotal: fxDaily,
			FxOther:      math.Round(in.FxAmount),
			RialAmount:   math.Round(in.RialAmount),
		}
		info.RialEquivalent = math.Round((fxDaily + info.FxOther + out.DocumentsFxTotal) * in.FxRate)
		if fxDaily > 0 {
			out.Lines = append(out.Lines, MissionAllowanceLine{
				Key: "fxPerDiem", TitleFa: "فوقالعادهٔ روزانهٔ ارزی", Amount: fxDaily,
				DetailFa: numberFa(fxPerDiem) + " " + currency + " × " + numberFa(days) + " روز",
			})
		}
		if info.FxOther > 0 {
			out.Lines = append(out.Lines, MissionAllowanceLine{
				Key: "fxOther", TitleFa: "سایر هزینه‌های ارزی", Amount: info.FxOther,
				DetailFa: "بلیت و اقامت خارج از کشور",
			})
		}
		if info.RialAmount > 0 {
			out.Lines = append(out.Lines, MissionAllowanceLine{
				Key: "foreignRial", TitleFa: "هزینه‌های ریالی مأموریت", Amount: info.RialAmount,
				DetailFa: "مسیر داخلی و اسکان ریالی",
			})
		}
		if in.FxRate > 0 {
			info.NoteFa = "معادل ریالی ارز با نرخ تسعیر " + moneyFa(in.FxRate) + " ریال برای هر " + currency + " محاسبه شد."
		} else {
			info.NoteFa = "نرخ تسعیر ثبت نشده است؛ مبلغ ارزی جداگانه پرداخت می‌شود."
		}
		out.Foreign = &info
		out.TotalFx = fxDaily + info.FxOther + out.DocumentsFxTotal
	}

	// جمع نهایی.
	out.TotalRial = math.Round(out.Allowance.MealsTotal + out.Allowance.LodgingCost +
		out.Allowance.Facilities + out.Allowance.Transport + out.Accommodation +
		out.DocumentsTotal + rialAmountOf(in, kind))
	out.TotalRialEquivalent = math.Round(out.TotalRial + out.TotalFx*in.FxRate)
	if len(out.Lines) == 0 {
		out.NoteFa = allowance.NoteFa
	}
	return out
}

// rialAmountOf مبلغ ریالی مستقیم ورودی (فقط برای مأموریت خارج از کشور جدا شمرده می‌شود).
func rialAmountOf(in MissionSettlementInput, kind string) float64 {
	if kind != "foreign" {
		return 0
	}
	return math.Round(in.RialAmount)
}

// dropLodgingShare سهم خواب را از محاسبهٔ روزانه حذف می‌کند (اسکان واقعی جایش می‌نشیند).
func dropLodgingShare(a MissionAllowance) MissionAllowance {
	a.LodgingCost = 0
	lines := make([]MissionAllowanceLine, 0, len(a.Lines))
	for _, line := range a.Lines {
		if line.Key == "lodging" {
			line.Amount = 0
			line.DetailFa = "پرداخت نشد — هزینهٔ اسکان جای آن آمده است"
		}
		lines = append(lines, line)
	}
	a.Lines = lines
	return a
}

// dropMealsShare سهم وعده‌ها را صفر می‌کند (همراه بیمار بدون مصوبهٔ درصدی).
func dropMealsShare(a MissionAllowance) MissionAllowance {
	a.MealsTotal = 0
	lines := make([]MissionAllowanceLine, 0, len(a.Lines))
	for _, line := range a.Lines {
		switch line.Key {
		case "breakfast", "lunch", "dinner":
			line.Amount = 0
			line.DetailFa = "پرداخت نشد — مشمول همراه بیمار"
		}
		lines = append(lines, line)
	}
	a.Lines = lines
	return a
}

// isFxCurrency آیا ارز سند غیرریالی است.
func isFxCurrency(code string) bool {
	value := strings.ToUpper(strings.TrimSpace(code))
	return value != "" && value != "IRR"
}

// currencyFa عنوان فارسی ارز.
func currencyFa(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "IRR":
		return "ریال ایران"
	case "EUR":
		return "یورو"
	case "USD":
		return "دلار آمریکا"
	case "AED":
		return "درهم امارات"
	case "GBP":
		return "پوند انگلیس"
	}
	return strings.ToUpper(strings.TrimSpace(code))
}

// accommodationFa عنوان فارسی نوع اسکان.
func accommodationFa(key string) string {
	switch strings.TrimSpace(key) {
	case "hotel":
		return "هتل"
	case "guesthouse":
		return "مهمانسرا"
	case "hospital":
		return "بیمارستان / محل درمان"
	case "relative":
		return "منزل اقوام"
	case "dormitory":
		return "خوابگاه سازمانی"
	}
	return "بدون اسکان سازمانی"
}

// documentKindFa عنوان فارسی نوع سند پشتیبان.
func documentKindFa(kind string) string {
	switch strings.TrimSpace(kind) {
	case "taxi":
		return "قبض تاکسی"
	case "hotel":
		return "فاکتور هتل"
	case "guesthouse":
		return "فاکتور مهمانسرا"
	case "air_ticket":
		return "بلیت هواپیما"
	case "bus_ticket":
		return "بلیت اتوبوس"
	case "mileage":
		return "قبض کرایهٔ کیلومتری"
	case "medical":
		return "فاکتور درمانی"
	}
	return "سند پشتیبان"
}

// documentDetail شرح یک سند پشتیبان: شماره، تاریخ و تأییدکننده.
func documentDetail(doc MissionDocument) string {
	parts := []string{}
	if no := strings.TrimSpace(doc.DocNo); no != "" {
		parts = append(parts, "شمارهٔ "+no)
	}
	if date := strings.TrimSpace(doc.DocDate); date != "" {
		parts = append(parts, "تاریخ "+date)
	}
	if doc.Approved {
		by := strings.TrimSpace(doc.ApprovedBy)
		if by == "" {
			by = "کارشناس"
		}
		parts = append(parts, "تأیید "+by)
	}
	if note := strings.TrimSpace(doc.NoteFa); note != "" {
		parts = append(parts, note)
	}
	return strings.Join(parts, " — ")
}
