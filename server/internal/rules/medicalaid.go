// Package rules — قاعدهٔ «کمک هزینه‌های درمانی» بر پایهٔ تعرفه‌های مصوب.
//
// چرا اینجا: تصمیم «این فاکتور دارو/اروتز/عینک چه اندازه پرداخت می‌شود» یک قاعدهٔ پرداخت
// است، نه یک محاسبهٔ نمایشی. دستورالعمل سازمان بهداشت و درمان صنعت نفت سه بخش دارد
// (کمک هزینهٔ درمانی، اروتز و پروتز، دندانپزشکی) و هر بخش تعرفهٔ خودش را دارد؛ اگر این
// محاسبه در فرم نوشته شود، فرم وب، اپلیکیشن و سند چاپی سه عدد می‌سازند.
//
// **تعرفه‌ها دادهٔ پارامتریک‌اند، نه عدد ثابت کد:** هر ردیف تعرفه (شرح قلم، واحد، مبلغ،
// سقف، سهم سازمان) یک رکورد است و از ماژول درمان خوانده می‌شود؛ این بسته فقط قاعدهٔ
// محاسبه را می‌داند. پس تغییر تعرفهٔ سالانه نیازی به انتشار نسخهٔ تازه ندارد.
//
// آستانه‌ها هم پارامتر شرکت‌اند:
//
//	medical.aid.orgSharePercent        سهم پیش‌فرض سازمان (درصد) وقتی قلم تعرفه سهم ندارد
//	medical.aid.annualCeiling          سقف پرداخت کمک هزینهٔ درمانی در یک سال (ریال)
//	medical.aid.perItemCeiling         سقف پیش‌فرض هر قلم (ریال)
//	medical.aid.requirePrescription    آیا اقلام نیازمند نسخه بدون نسخه پرداخت نمی‌شوند
package rules

import (
	"strconv"
	"strings"
)

// کلیدهای دستهٔ کمک هزینه.
const (
	// MedicalAidOrtesisProsthesis اروتز و پروتز (اعضای مصنوعی، کفش طبی، کمربند، جوارب،
	// عصا و واکر، اروتزهای سرپایی و پروتزهای بیرونی).
	MedicalAidOrtesisProsthesis = "orthesis_prosthesis"
	// MedicalAidEquipment تجهیزات و لوازم پزشکی (سمعک، حلزون شنوایی، گلوکومتر، پمپ
	// انسولین، سنسور قند خون، لنز داخل چشمی و لوازم مصرفی بیماران).
	MedicalAidEquipment = "equipment"
	// MedicalAidDrug دارو.
	MedicalAidDrug = "drug"
	// MedicalAidGlasses عینک (فریم، عدسی، عینک طبی).
	MedicalAidGlasses = "glasses"
	// MedicalAidDental دندانپزشکی.
	MedicalAidDental = "dental"
	// MedicalAidParaclinic پاراکلینیک: آزمایش، تصویربرداری، فیزیوتراپی و مانند آن.
	MedicalAidParaclinic = "paraclinic"
	// MedicalAidVisit ویزیت و طبابت.
	MedicalAidVisit = "visit"
	// MedicalAidOther سایر هزینه‌های درمانی.
	MedicalAidOther = "other"
)

// MedicalAidCategory یک دستهٔ کمک هزینهٔ درمانی.
type MedicalAidCategory struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// MedicalAidCategories فهرست دسته‌های کمک هزینه (خروجی فرم و برچسب سند).
func MedicalAidCategories() []MedicalAidCategory {
	return []MedicalAidCategory{
		{Key: MedicalAidOrtesisProsthesis, TitleFa: "اروتز و پروتز", NoteFa: "اعضای مصنوعی، کفش طبی، کمربند، کالای طبی و مانند آن."},
		{Key: MedicalAidEquipment, TitleFa: "تجهیزات و لوازم پزشکی", NoteFa: "سمعک، حلزون شنوایی، پمپ انسولین، سنسور قند و لوازم مصرفی بیماران."},
		{Key: MedicalAidDrug, TitleFa: "دارو", NoteFa: "داروهای نسخه‌ای بر پایهٔ نسخهٔ پزشک."},
		{Key: MedicalAidGlasses, TitleFa: "عینک", NoteFa: "عینک طبی: فریم و عدسی."},
		{Key: MedicalAidDental, TitleFa: "دندانپزشکی", NoteFa: "خدمات دندانپزشکی بر پایهٔ تعرفهٔ مصوب."},
		{Key: MedicalAidParaclinic, TitleFa: "پاراکلینیک", NoteFa: "آزمایش، تصویربرداری، فیزیوتراپی و مانند آن."},
		{Key: MedicalAidVisit, TitleFa: "ویزیت و طبابت", NoteFa: "ویزیت پزشک و مشاورهٔ درمانی."},
		{Key: MedicalAidOther, TitleFa: "سایر هزینه‌های درمانی", NoteFa: "هر هزینهٔ درمانی دیگری که در تعرفهٔ مصوب قلم دارد."},
	}
}

// MedicalAidCategoryTitleFa عنوان فارسی یک دسته؛ کلید ناشناخته خودش برمی‌گردد.
func MedicalAidCategoryTitleFa(key string) string {
	value := strings.TrimSpace(key)
	for _, c := range MedicalAidCategories() {
		if c.Key == value {
			return c.TitleFa
		}
	}
	return value
}

// MedicalAidParams آستانه‌های کمک هزینهٔ درمانی از پارامترهای شرکت.
type MedicalAidParams struct {
	// OrgSharePercent سهم پیش‌فرض سازمان (درصد از مبلغ تعرفه) وقتی قلم تعرفه سهم خودش را ندارد.
	OrgSharePercent float64 `json:"orgSharePercent"`
	// AnnualCeiling سقف پرداخت کمک هزینه در یک سال برای هر کارمند (ریال)؛ صفر یعنی بی‌سقف.
	AnnualCeiling int64 `json:"annualCeiling"`
	// PerItemCeiling سقف پیش‌فرض هر قلم (ریال)؛ صفر یعنی بی‌سقف.
	PerItemCeiling int64 `json:"perItemCeiling"`
	// RequirePrescription آیا اقلام نیازمند نسخه بدون نسخهٔ تأییدشده پرداخت نمی‌شوند.
	RequirePrescription bool `json:"requirePrescription"`
	// VerifierNameFa نام مرجع تصویب برای متن سند (نمایشی).
	VerifierNameFa string `json:"verifierNameFa,omitempty"`
}

// DefaultMedicalAidParams پیش‌فرض‌های اعلام‌شده؛ در استقرار واقعی از پارامتر شرکت می‌آید.
//
// سهم پیش‌فرض سازمان **۱۰۰٪** است، چون دستورالعمل برای هر قلم «مبلغ کمک هزینه» تعیین می‌کند و
// پرداخت تا همان سقف کامل است؛ درصد کمتر تنها در ردیف‌هایی است که خودِ دستورالعمل سهم کمتری
// گفته (مثل سمعک گوش دوم) و همان در ردیف تعرفه ثبت می‌شود.
func DefaultMedicalAidParams() MedicalAidParams {
	return MedicalAidParams{
		OrgSharePercent:     1,
		AnnualCeiling:       0,
		PerItemCeiling:      0,
		RequirePrescription: true,
		VerifierNameFa:      "سازمان بهداشت و درمان صنعت نفت",
	}
}

// MedicalAidParamsFromValues پارامترهای شرکت را روی پیش‌فرض می‌نشاند. کلید با پیشوند
// `medical.aid.` خوانده می‌شود؛ کلید غایب پیش‌فرض می‌ماند و پارامتر ناقص محاسبه را
// متوقف نمی‌کند.
func MedicalAidParamsFromValues(values map[string]float64) MedicalAidParams {
	p := DefaultMedicalAidParams()
	if v, ok := values["medical.aid.orgSharePercent"]; ok {
		// درصد می‌تواند نسبتی (۰٫۷) یا درصدی (۷۰) نوشته شود؛ هر دو پذیرفته است.
		if v > 1 {
			v = v / 100
		}
		if v >= 0 && v <= 1 {
			p.OrgSharePercent = v
		}
	}
	if v, ok := values["medical.aid.annualCeiling"]; ok && v > 0 {
		p.AnnualCeiling = int64(v)
	}
	if v, ok := values["medical.aid.perItemCeiling"]; ok && v > 0 {
		p.PerItemCeiling = int64(v)
	}
	if v, ok := values["medical.aid.requirePrescription"]; ok {
		p.RequirePrescription = v != 0
	}
	return p
}

// MedicalTariffItem یک ردیف از تعرفهٔ مصوب کمک هزینهٔ درمانی.
//
// چرا مبلغ تعرفه در دیتا و نه در کد: تعرفه هر سال مصوب می‌شود و سالانه عوض می‌شود؛ نگه‌داشتنش
// در رکورد، امکان بازنگری پارامتریک و سابقهٔ نسخه‌ها را می‌دهد.
type MedicalTariffItem struct {
	// Code شناسهٔ قلم تعرفه (کلید ماشین‌خوان، ماندگار بین سال‌ها).
	Code string `json:"code"`
	// Category دستهٔ کمک هزینه (کلیدهای MedicalAid*).
	Category string `json:"category"`
	// TitleFa شرح قلم تعرفه، عیناً بر پایهٔ دستورالعمل.
	TitleFa string `json:"titleFa"`
	// UnitFa واحد تعرفه (هر عدد، هر جلسه، هر ماه، هر بسته).
	UnitFa string `json:"unitFa,omitempty"`
	// Amount تعرفهٔ مصوب هر واحد (ریال).
	Amount int64 `json:"amount"`
	// CeilingAmount سقف پرداخت همان قلم (ریال)؛ صفر یعنی از سقف پیش‌فرض پارامتر استفاده شود.
	CeilingAmount int64 `json:"ceilingAmount,omitempty"`
	// OrgSharePercent سهم سازمان از تعرفه (`0.7` = ۷۰٪)؛ صفر یعنی سهم پیش‌فرض پارامتر.
	OrgSharePercent float64 `json:"orgSharePercent,omitempty"`
	// InsuranceSharePercent سهمی از تعرفه که سازمان از بیمه‌گر مطالبه می‌کند (نمایشی).
	InsuranceSharePercent float64 `json:"insuranceSharePercent,omitempty"`
	// RequirePrescription آیا این قلم بدون نسخهٔ تأییدشده پرداخت نمی‌شود.
	RequirePrescription bool `json:"requirePrescription,omitempty"`
	// RefFa ارجاع به بند/جدول دستورالعمل (برای دفاع در برابر پرسش مراجع نظارتی).
	RefFa string `json:"refFa,omitempty"`
}

// MedicalAidClaimItem یک قلم درخواست کمک هزینه با تعرفهٔ متناظر.
type MedicalAidClaimItem struct {
	Code     string `json:"code"`
	TitleFa  string `json:"titleFa,omitempty"`
	Quantity int    `json:"quantity"`
	// InvoiceAmount مبلغ فاکتور برای این قلم (ریال) — آنچه کارمند پرداخت کرده است.
	InvoiceAmount int64 `json:"invoiceAmount"`
	// Tariff تعرفهٔ مصوب همان قلم؛ اگر نیامده باشد، فقط مبلغ فاکتور مبنای محاسبه می‌شود.
	Tariff MedicalTariffItem `json:"tariff"`
	// PrescriptionNo شمارهٔ نسخه/معرفی‌نامه و PrescriptionApproved تأیید آن.
	PrescriptionNo       string `json:"prescriptionNo,omitempty"`
	PrescriptionApproved bool   `json:"prescriptionApproved"`
}

// MedicalAidInput ورودی محاسبهٔ یک درخواست کمک هزینهٔ درمانی.
type MedicalAidInput struct {
	EmployeeID   string `json:"employeeId,omitempty"`
	EmployeeName string `json:"employeeName,omitempty"`
	// Category دستهٔ درخواست (کلیدهای MedicalAid*).
	Category string `json:"category"`
	// PeriodYearYear سال شمسی که سقف سالانه به آن مربوط است (نمایشی/کنترل سقف).
	PeriodYear int `json:"periodYear,omitempty"`
	// Items اقلام درخواست.
	Items []MedicalAidClaimItem `json:"items"`
	// AnnualPaid جمع پرداخت‌های پیشین همین سال برای همین کارمند (ریال).
	AnnualPaid int64 `json:"annualPaid"`
}

// MedicalAidLine یک سطر از فرم محاسباتی کمک هزینه.
type MedicalAidLine struct {
	Order    int    `json:"order"`
	Code     string `json:"code"`
	TitleFa  string `json:"titleFa"`
	UnitFa   string `json:"unitFa,omitempty"`
	Quantity int    `json:"quantity"`
	// InvoiceAmount مبلغ فاکتور قلم و TariffAmount تعرفهٔ مصوب همان تعداد.
	InvoiceAmount int64 `json:"invoiceAmount"`
	TariffAmount  int64 `json:"tariffAmount"`
	// BaseAmount مبنای محاسبه (کمینهٔ فاکتور و تعرفه) و CeilingApplied سقفی که کسر شد.
	BaseAmount     int64   `json:"baseAmount"`
	CeilingApplied int64   `json:"ceilingApplied"`
	OrgShare       float64 `json:"orgSharePercent"`
	// PayableAmount سهم سازمان از این قلم و EmployeeAmount آنچه بر عهدهٔ کارمند می‌ماند.
	PayableAmount  int64  `json:"payableAmount"`
	EmployeeAmount int64  `json:"employeeAmount"`
	InsuranceShare int64  `json:"insuranceShare,omitempty"`
	Payable        bool   `json:"payable"`
	DetailFa       string `json:"detailFa"`
	RefFa          string `json:"refFa,omitempty"`
}

// MedicalAidResult خروجی محاسبهٔ درخواست کمک هزینهٔ درمانی.
type MedicalAidResult struct {
	Category   string `json:"category"`
	CategoryFa string `json:"categoryFa"`

	ItemsCount int `json:"itemsCount"`
	// TotalInvoice جمع فاکتورها، TotalBase جمع مبنای محاسبه و TotalPayable سهم سازمان.
	TotalInvoice int64 `json:"totalInvoice"`
	TotalBase    int64 `json:"totalBase"`
	TotalPayable int64 `json:"totalPayable"`
	// TotalEmployee آنچه بر عهدهٔ کارمند می‌ماند (مازاد تعرفه، سقف قلم و کسری سهم سازمان).
	TotalEmployee int64 `json:"totalEmployee"`
	// TotalInsurance سهمی از پرداخت که سازمان از بیمه‌گر مطالبه می‌کند.
	TotalInsurance int64 `json:"totalInsurance"`

	// CeilingApplied مبلغی که به سقف قلم یا سقف سالانه خورده و پرداخت نشد.
	CeilingApplied int64 `json:"ceilingApplied"`
	// AnnualPaidBefore/AnnualPaidAfter جمع پرداخت سال پیش و پس از این درخواست.
	AnnualPaidBefore int64 `json:"annualPaidBefore"`
	AnnualPaidAfter  int64 `json:"annualPaidAfter"`
	AnnualCeiling    int64 `json:"annualCeiling"`
	// AnnualRemaining باقی‌ماندهٔ سقف سالانه پس از این درخواست (صفر یعنی مصرف کامل).
	AnnualRemaining int64 `json:"annualRemaining"`
	// AnnualExceeded یعنی درخواست از سقف سالانه گذشته و بخشی پرداخت نمی‌شود.
	AnnualExceeded bool `json:"annualExceeded"`

	Lines      []MedicalAidLine `json:"lines"`
	BlockingFa []string         `json:"blockingFa,omitempty"`
	SummaryFa  string           `json:"summaryFa"`
	NoteFa     string           `json:"noteFa,omitempty"`
}

// ComputeMedicalAid سهم سازمان، بیمه و کارمند را برای یک درخواست کمک هزینه محاسبه می‌کند.
//
// قاعده‌ها (همه در همین تابع اعلام شده‌اند):
//
//	۱. مبلغ پایهٔ هر قلم = کمینهٔ «مبلغ فاکتور» و «تعرفهٔ مصوب × تعداد». سازمان بیش از
//	   تعرفه نمی‌پردازد و بیش از فاکتور هم نمی‌پردازد.
//	۲. سقف قلم: اگر پایه از سقف آن قلم (یا سقف پیش‌فرض پارامتر) بیشتر باشد، مازاد کسر
//	   می‌شود و به حساب کارمند می‌رود.
//	۳. سهم سازمان از مبلغ در سقف‌آمده است: `OrgSharePercent` قلم و در نبود آن سهم پیش‌فرض
//	   پارامتر. باقی‌مانده بر عهدهٔ کارمند است.
//	۴. سهم بیمه: بخشی از سهم سازمان است که سازمان از بیمه‌گر مطالبه می‌کند؛ در جمع پرداخت
//	   دوباره شمرده نمی‌شود (فقط برای گزارش و مطالبه）。
//	۵. نسخه: اگر پارامتر `requirePrescription` روشن و قلم نیازمند نسخه باشد و نسخه تأیید
//	   نشده باشد، آن قلم پرداخت نمی‌شود و دلیلش در سطر می‌آید.
//	۶. سقف سالانه: اگر جمع پرداخت‌های سال + این درخواست از سقف پارامتر بگذرد، مازاد کسر
//	   و در سطرها و جمع‌بندی گزارش می‌شود (صفر بودن پارامتر یعنی بی‌سقف).
//
// خروجی، خط‌به‌خط است تا فرم وب و سند چاپی همان چیزی را نشان دهند که سرور حساب کرده است.
func ComputeMedicalAid(in MedicalAidInput, params MedicalAidParams) MedicalAidResult {
	res := MedicalAidResult{
		Category:         strings.TrimSpace(in.Category),
		CategoryFa:       MedicalAidCategoryTitleFa(in.Category),
		ItemsCount:       len(in.Items),
		AnnualPaidBefore: nonNegative64(in.AnnualPaid),
		AnnualCeiling:    params.AnnualCeiling,
		Lines:            make([]MedicalAidLine, 0, len(in.Items)),
	}

	for i, item := range in.Items {
		line := computeMedicalAidLine(i+1, item, params)
		res.Lines = append(res.Lines, line)
		res.TotalInvoice += line.InvoiceAmount
		res.TotalBase += line.BaseAmount
		res.TotalPayable += line.PayableAmount
		res.TotalEmployee += line.EmployeeAmount
		res.TotalInsurance += line.InsuranceShare
		res.CeilingApplied += line.CeilingApplied
		if !line.Payable {
			res.BlockingFa = append(res.BlockingFa, line.TitleFa+" — "+line.DetailFa)
		}
	}

	// سقف سالانه: پس از جمع اقلام سنجیده می‌شود تا کسر از بزرگ‌ترین سطرها آغاز شود.
	if res.AnnualCeiling > 0 {
		allowed := res.AnnualCeiling - res.AnnualPaidBefore
		if allowed < 0 {
			allowed = 0
		}
		if res.TotalPayable > allowed {
			over := res.TotalPayable - allowed
			res.AnnualExceeded = true
			applyAnnualCeiling(res.Lines, over, &res)
		}
		res.AnnualPaidAfter = res.AnnualPaidBefore + res.TotalPayable
		res.AnnualRemaining = res.AnnualCeiling - res.AnnualPaidAfter
		if res.AnnualRemaining < 0 {
			res.AnnualRemaining = 0
		}
	}

	res.SummaryFa = medicalAidSummaryFa(res)
	res.NoteFa = "محاسبه بر پایهٔ تعرفهٔ مصوب سازمان بهداشت و درمان صنعت نفت و پارامترهای شرکت " +
		"انجام شده است؛ سازمان بیش از تعرفهٔ مصوب نمی‌پردازد و مازاد فاکتور بر عهدهٔ کارمند است. " +
		"ردیف‌های تعرفه و سقف‌های هر قلم در فهرست تعرفه‌های درمان نگه داشته می‌شوند و با بازنگری سالانه به‌روز می‌شوند."
	return res
}

// computeMedicalAidLine محاسبهٔ یک قلم درخواست.
func computeMedicalAidLine(order int, item MedicalAidClaimItem, params MedicalAidParams) MedicalAidLine {
	qty := item.Quantity
	if qty <= 0 {
		qty = 1
	}
	tariff := item.Tariff
	line := MedicalAidLine{
		Order: order, Code: tariff.Code, TitleFa: firstNonEmpty(item.TitleFa, tariff.TitleFa, tariff.Code),
		UnitFa: tariff.UnitFa, Quantity: qty, InvoiceAmount: nonNegative64(item.InvoiceAmount),
		RefFa: tariff.RefFa,
	}
	line.TariffAmount = nonNegative64(tariff.Amount) * int64(qty)

	// ۱) مبنای محاسبه: کمینهٔ فاکتور و تعرفه (تعرفه ناشناخته = فقط فاکتور؟ نه: بدون تعرفه
	// سازمان تعهدی نمی‌پذیرد، پس پایه صفر می‌شود و علتش در شرح می‌آید).
	switch {
	case tariff.Amount <= 0:
		line.DetailFa = "تعرفهٔ مصوب این قلم ثبت نشده است؛ پرداخت بدون تعرفه مجاز نیست."
		line.EmployeeAmount = line.InvoiceAmount
		return line
	case line.InvoiceAmount <= 0:
		line.DetailFa = "مبلغ فاکتور ثبت نشده است."
		return line
	case line.InvoiceAmount < line.TariffAmount:
		line.BaseAmount = line.InvoiceAmount
	default:
		line.BaseAmount = line.TariffAmount
	}

	// ۲) سقف قلم.
	ceiling := tariff.CeilingAmount
	if ceiling <= 0 {
		ceiling = params.PerItemCeiling
	}
	if ceiling > 0 && line.BaseAmount > ceiling {
		line.CeilingApplied = line.BaseAmount - ceiling
		line.BaseAmount = ceiling
	}

	// ۳) نسخهٔ لازم.
	if params.RequirePrescription && tariff.RequirePrescription && !item.PrescriptionApproved {
		line.DetailFa = "این قلم نیازمند نسخهٔ تأییدشده است و نسخه‌اش تأیید نشده؛ پرداخت نمی‌شود."
		line.EmployeeAmount = line.InvoiceAmount
		line.CeilingApplied = 0
		line.BaseAmount = 0
		return line
	}

	// ۴) سهم سازمان و کارمند.
	share := tariff.OrgSharePercent
	if share <= 0 {
		share = params.OrgSharePercent
	}
	line.OrgShare = share
	line.PayableAmount = RoundHalfUp(float64(line.BaseAmount) * share)
	line.InsuranceShare = RoundHalfUp(float64(line.PayableAmount) * tariff.InsuranceSharePercent)
	line.EmployeeAmount = line.InvoiceAmount - line.PayableAmount
	line.Payable = line.PayableAmount > 0

	// ۵) شرح محاسبه برای فرم و سند.
	parts := []string{strconv.Itoa(qty) + " × " + IntMoneyFa(tariff.Amount) + " ریال تعرفه"}
	if item.InvoiceAmount < line.TariffAmount {
		parts = append(parts, "فاکتور کمتر از تعرفه ("+IntMoneyFa(item.InvoiceAmount)+")")
	}
	if line.CeilingApplied > 0 {
		parts = append(parts, "کسر سقف قلم "+IntMoneyFa(line.CeilingApplied))
	}
	parts = append(parts, "سهم سازمان "+percentageFa(share)+" = "+IntMoneyFa(line.PayableAmount))
	if line.EmployeeAmount > 0 {
		parts = append(parts, "سهم کارمند "+IntMoneyFa(line.EmployeeAmount))
	}
	line.DetailFa = strings.Join(parts, " · ")
	return line
}

// applyAnnualCeiling کسر سقف سالانه را از سطرها آغاز می‌کند و جمع‌های نتیجه را به‌روز می‌کند.
//
// قاعدهٔ ترتیب: از آخرین سطر به عقب (اقلام کم‌اهمیت‌تر) کسر می‌شود تا اقلام درمانی مهم در
// اول سند تا حد ممکن پرداخت بمانند؛ مبلغ کسرشده از سهم سازمان کم و به سهم کارمند اضافه می‌شود.
func applyAnnualCeiling(lines []MedicalAidLine, over int64, res *MedicalAidResult) {
	remaining := over
	for i := len(lines) - 1; i >= 0 && remaining > 0; i-- {
		if lines[i].PayableAmount <= 0 {
			continue
		}
		cut := lines[i].PayableAmount
		if cut > remaining {
			cut = remaining
		}
		lines[i].PayableAmount -= cut
		lines[i].EmployeeAmount += cut
		lines[i].CeilingApplied += cut
		lines[i].Payable = lines[i].PayableAmount > 0
		lines[i].DetailFa += " · کسر سقف سالانه " + IntMoneyFa(cut)
		remaining -= cut
	}
	res.CeilingApplied += over - remaining
	res.TotalEmployee += over - remaining
	res.TotalPayable -= over - remaining
}

// medicalAidSummaryFa جملهٔ جمع‌بندی فرم محاسباتی.
func medicalAidSummaryFa(res MedicalAidResult) string {
	if res.ItemsCount == 0 {
		return "قلمی برای محاسبه ثبت نشده است."
	}
	summary := "سهم سازمان " + IntMoneyFa(res.TotalPayable) + " ریال از " + strconv.Itoa(res.ItemsCount) + " قلم"
	if res.CeilingApplied > 0 {
		summary += " — مبلغ کسرشده به دلیل سقف: " + IntMoneyFa(res.CeilingApplied) + " ریال"
	}
	if res.TotalEmployee > 0 {
		summary += " — بر عهدهٔ کارمند: " + IntMoneyFa(res.TotalEmployee) + " ریال"
	}
	if res.AnnualCeiling > 0 {
		summary += " — باقی‌ماندهٔ سقف سالانه: " + IntMoneyFa(res.AnnualRemaining) + " ریال"
	}
	if res.AnnualExceeded {
		summary += " (سقف سالانه پر شده است)"
	}
	return summary
}

// MedicalAidInstructionFa متن دستورالعمل کمک هزینه‌های درمانی برای فرم و چاپ.
//
// چرا در دامنه: دستورالعمل بخشی از قاعده است؛ فرم درخواست، سند تصویب و کارتابل باید یک
// فرآیند را نشان دهند تا کسی گام تازه‌ای از خودش نسازد.
func MedicalAidInstructionFa() []string {
	return []string{
		"۱. تعرفه‌های مصوب هر سال در فهرست تعرفه‌های درمان ثبت می‌شود (اروتز و پروتز، دارو، عینک، دندانپزشکی، پاراکلینیک، ویزیت و سایر) با واحد، مبلغ، سقف و سهم سازمان.",
		"۲. هر درخواست به یک دسته تعلق دارد و اقلام آن از فهرست تعرفه انتخاب می‌شوند؛ شرح قلم عیناً همان عنوان دستورالعمل است تا سابقهٔ پرداخت قابل دفاع بماند.",
		"۳. مبلغ پایهٔ هر قلم کمینهٔ «مبلغ فاکتور» و «تعرفهٔ مصوب × تعداد» است؛ سازمان بیش از تعرفه نمی‌پردازد.",
		"۴. اگر فاکتور کمتر از تعرفه باشد، همان مبلغ فاکتور مبنا می‌شود؛ مازاد فاکتور بر تعرفه هرگز پرداخت نمی‌شود.",
		"۵. سقف هر قلم از خود تعرفه یا از پارامتر شرکت خوانده می‌شود؛ مازاد بر سقف بر عهدهٔ کارمند است.",
		"۶. سهم سازمان درصدی از مبلغ در سقف‌آمده است؛ باقی‌مانده سهم کارمند است. سهم بیمه فقط بخشی از سهم سازمان است که از بیمه‌گر مطالبه می‌شود.",
		"۷. اقلام نیازمند نسخه (دارو، اروتز و پروتز و مانند آن) بدون نسخهٔ تأییدشده پرداخت نمی‌شوند.",
		"۸. سقف سالانهٔ کمک هزینهٔ درمانی بر پایهٔ پرداخت‌های همان سال کنترل می‌شود؛ مازاد کسر و در سند ذکر می‌شود.",
		"۹. تصویب درخواست با کارشناس بهداشت و درمان و تأیید مدیر منابع انسانی است و پرداخت پس از تصویب ثبت می‌شود.",
		"۱۰. تغییر تعرفه‌ها نیازی به تغییر نرم‌افزار ندارد: ردیف تعرفه ویرایش یا ردیف تازه ثبت می‌شود و از تاریخ اجرای خودش مبنای محاسبه است.",
	}
}

// firstNonEmpty نخستین مقدار غیرخالی.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
