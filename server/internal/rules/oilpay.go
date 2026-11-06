// Package rules — قاعدهٔ «نظام پرداخت صندوق بازنشستگی نفت».
//
// چرا اینجا و نه در رابط: سازمان دو نظام پرداخت دارد. نظام نخست (کارکنان مشمول تأمین
// اجتماعی) مزد مبنا و اقلام گوناگون دارد و در `ComputePayroll` و موتور اقلام است. نظام
// دوم — کارکنان تابع صندوق بازنشستگی نفت — ساختار ساده و پیوسته‌ای دارد:
//
//	۱. کارمند یک «پایهٔ حقوق» دارد.
//	۲. «فوق‌العادهٔ بدی آب و هوا» (همان فوق‌العادهٔ مناطق گرمسیری) درصدی از پایهٔ حقوق است.
//	۳. جمع پایهٔ حقوق و بدی آب و هوا «مبنای فوق‌العاده‌ها» است.
//	۴. بقیهٔ فوق‌العاده‌ها درصدی از همین مبنا هستند؛ پس هر تغییری در پایه یا درصد بدی
//	   آب و هوا، همهٔ اقلام را با هم به‌روز می‌کند و هیچ عددی دو جا حساب نمی‌شود.
//	۵. کسور (صندوق/بیمه و مالیات) از جمع ناخالص و با پارامترهای شرکت کسر می‌شود.
//
// فهرست اقلام و درصدهایشان از جدول مصوب سازمان آمده است (فایل «فوق‌العاده»):
//
//	بدی آب و هوا ۲۵٪ · محرومیت ۲۷٫۵٪ · ویژه ۳٪ · صرف وقت ۱۶٫۵٪ · کارگاهی ۱۰٪ ·
//	جذب و تخصصی شغل ۲۷٪ · فرآورشی ۳۰٪ · نوبت‌کار ۳۵٪
//
// همهٔ درصدها پارامتریک‌اند (`oilpay.*`)؛ قلم تازه فقط در فهرست همین فایل (یا پارامتر
// شرکت) اضافه می‌شود و فرم و فیش بدون تغییر کد آن را نشان می‌دهند.
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
//	allowanceBase یعنی «پایهٔ حقوق + فوق‌العادهٔ بدی آب و هوا» (قاعدهٔ سازمان)،
//	baseSalary    یعنی فقط پایهٔ حقوق (برای قلم‌هایی که مصوبهٔ جدا دارند).
//
// RequiresShift یعنی این قلم تنها برای کارمند نوبت‌کار پرداخت می‌شود (نوبت‌کار).
type OilAllowance struct {
	Key           string  `json:"key"`
	TitleFa       string  `json:"titleFa"`
	GroupFa       string  `json:"groupFa"`
	Percent       float64 `json:"percent"`
	Base          string  `json:"base"`
	RequiresShift bool    `json:"requiresShift,omitempty"`
	NoteFa        string  `json:"noteFa,omitempty"`
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
	OilBaseAllowance = "allowanceBase"
	OilBaseSalary    = "baseSalary"
)

// کلید قلم نوبت‌کار (تنها قلمی که به نوبت‌کاری کارمند گره خورده است).
const OilAllowanceShift = "shiftWorker"

// کلید قلم «بدی آب و هوا» که مبنای فوق‌العاده‌ها را می‌سازد.
const OilAllowanceClimate = "badClimate"

// OilPayParams پارامترهای نظام پرداخت صندوق بازنشستگی نفت (از پروفایل شرکت).
type OilPayParams struct {
	// ClimatePercent درصد فوق‌العادهٔ بدی آب و هوا از پایهٔ حقوق
	// (پارامتر oilpay.climatePercent).
	ClimatePercent float64 `json:"climatePercent"`
	// Allowances فهرست فوق‌العادههای درصدی این نظام.
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
	// OvertimeFactor ضریب اضافه‌کاری (۱٫۴ = ۴۰٪ بالاتر) و MonthlyHours ساعت موظفی ماه.
	OvertimeFactor float64 `json:"overtimeFactor"`
	MonthlyHours   float64 `json:"monthlyHours"`
}

// DefaultOilPayParams مقادیر پیش‌فرض (جدول مصوب فوق‌العاده‌ها) تا وقتی پارامتر شرکت
// تنظیم نشده است.
func DefaultOilPayParams() OilPayParams {
	return OilPayParams{
		ClimatePercent: 25,
		Allowances: []OilAllowance{
			{Key: "deprivation", TitleFa: "فوق‌العادهٔ محرومیت", GroupFa: OilGroupClimate,
				Percent: 27.5, Base: OilBaseAllowance, NoteFa: "درصدی از پایهٔ حقوق و فوق‌العادهٔ بدی آب و هوا."},
			{Key: "special", TitleFa: "فوق‌العادهٔ ویژه", GroupFa: OilGroupClimate,
				Percent: 3, Base: OilBaseAllowance, NoteFa: "درصدی از مبنای فوق‌العاده‌ها."},
			{Key: "timeConsumed", TitleFa: "فوق‌العادهٔ صرف وقت", GroupFa: OilGroupClimate,
				Percent: 16.5, Base: OilBaseAllowance, NoteFa: "درصدی از مبنای فوق‌العاده‌ها."},
			{Key: "workshop", TitleFa: "فوق‌العادهٔ کارگاهی", GroupFa: OilGroupClimate,
				Percent: 10, Base: OilBaseAllowance, NoteFa: "برای کار در کارگاه و محیط عملیاتی."},
			{Key: "attractionSpecialty", TitleFa: "فوق‌العادهٔ جذب و تخصصی شغل", GroupFa: OilGroupClimate,
				Percent: 27, Base: OilBaseAllowance, NoteFa: "برای جذب و نگهداشت مشاغل تخصصی."},
			{Key: "production", TitleFa: "فوق‌العادهٔ فرآورشی", GroupFa: OilGroupClimate,
				Percent: 30, Base: OilBaseAllowance, NoteFa: "برای مشاغل واحدهای فرآورشی."},
			{Key: OilAllowanceShift, TitleFa: "فوق‌العادهٔ نوبت‌کار", GroupFa: OilGroupShift,
				Percent: 35, Base: OilBaseAllowance, RequiresShift: true,
				NoteFa: "تنها برای کارمند نوبت‌کار؛ به نسبت روزهای نوبت‌کاری در ماه."},
		},
		FundEmployeePercent:      7,
		FundEmployerPercent:      20,
		InsuranceEmployeePercent: 0,
		InsuranceCeiling:         0,
		TaxExemption:             24_000_000,
		TaxBrackets: []TaxBracket{
			{Upto: 50_000_000, Rate: 0.10},
			{Upto: 100_000_000, Rate: 0.15},
			{Upto: 200_000_000, Rate: 0.20},
			{Upto: 0, Rate: 0.30},
		},
		OvertimeFactor: 1.4,
		MonthlyHours:   176,
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
	// Shift یعنی کارمند در این ماه نوبت‌کار است (قلم نوبت‌کار پرداخت می‌شود).
	Shift bool `json:"shift"`
	// ShiftDays شمار روزهای نوبت‌کاری در ماه (صفر = تمام ماه) و DaysInMonth روزهای ماه.
	ShiftDays   float64 `json:"shiftDays"`
	DaysInMonth float64 `json:"daysInMonth"`
	// OvertimeHours ساعت اضافه‌کاری ماه.
	OvertimeHours float64 `json:"overtimeHours"`
	// OtherDeductions کسور دیگر (وام، مساعده، جریمه).
	OtherDeductions int64 `json:"otherDeductions"`
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
	// پایه، بدی آب و هوا (گرمسیری) و مبنای فوق‌العاده‌ها.
	BaseSalary     int64   `json:"baseSalary"`
	ClimatePercent float64 `json:"climatePercent"`
	Climate        int64   `json:"climate"`
	AllowanceBase  int64   `json:"allowanceBase"`
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
	Lines  []OilPayLine `json:"lines"`
	NoteFa string       `json:"noteFa,omitempty"`
}

// SchemeOil شناسهٔ نظام پرداخت صندوق بازنشستگی نفت.
const SchemeOil = "oil_pension_fund"

// ComputeOilPayslip فیش ماهانهٔ نظام پرداخت صندوق بازنشستگی نفت را می‌سازد.
//
// قاعده‌ها به ترتیب:
//
//	۱. فوق‌العادهٔ بدی آب و هوا = درصد همان فوق‌العاده × پایهٔ حقوق.
//	۲. مبنای فوق‌العاده‌ها = پایهٔ حقوق + فوق‌العادهٔ بدی آب و هوا.
//	۳. هر قلم فهرست = درصد همان قلم × مبنا (یا پایهٔ حقوق اگر قلم چنین گفته).
//	۴. «نوبت‌کار» تنها وقتی پرداخت می‌شود که کارمند نوبت‌کار باشد و به نسبت روزهای
//	   نوبت‌کاری به روزهای ماه محاسبه می‌شود (خالی = تمام ماه).
//	۵. اضافه‌کاری = (مبنا ÷ ساعت موظفی) × ساعت اضافه‌کاری × ضریب اضافه‌کاری.
//	۶. جمع ناخالص = پایه + بدی آب و هوا + همهٔ فوق‌العاده‌ها + اضافه‌کاری.
//	۷. کسور = سهم کارمند صندوق (+ بیمه) با سقف، مالیات پله‌ای و کسور دیگر.
//	۸. خالص = ناخالص − کسور.
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

	out := OilPayslip{
		Scheme:          SchemeOil,
		SchemeFa:        "نظام پرداخت صندوق بازنشستگی نفت",
		BaseSalary:      base,
		ClimatePercent:  climatePercent,
		Climate:         climate,
		AllowanceBase:   allowanceBase,
		AllowanceLines:  []OilPayLine{},
		OtherDeductions: in.OtherDeductions,
	}

	// سطر پایه و بدی آب و هوا، تا فیش از بالا خوانا باشد.
	out.Lines = append(out.Lines,
		OilPayLine{Key: "baseSalary", TitleFa: "پایهٔ حقوق", GroupFa: OilGroupBase,
			Amount: base, DetailFa: "پایهٔ حقوق ماهانهٔ کارمند"},
		OilPayLine{Key: OilAllowanceClimate, TitleFa: "فوق‌العادهٔ بدی آب و هوا (مناطق گرمسیری)",
			GroupFa: OilGroupBase, Percent: climatePercent, Base: base, Amount: climate,
			DetailFa: numberFa(climatePercent) + "٪ × پایهٔ حقوق (" + moneyFa(float64(base)) + " ریال)"},
		OilPayLine{Key: "allowanceBase", TitleFa: "مبنای فوق‌العاده‌ها", GroupFa: OilGroupBase,
			Amount: allowanceBase, DetailFa: "پایهٔ حقوق + فوق‌العادهٔ بدی آب و هوا"},
	)

	// فوق‌العاده‌های درصدی مصوب.
	monthDays := in.DaysInMonth
	if monthDays <= 0 {
		monthDays = 30
	}
	for _, allowance := range params.Allowances {
		percent := allowance.Percent
		if manual, ok := in.AllowancePercents[allowance.Key]; ok {
			percent = manual
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
		}
		amount := float64(lineBase) * percent / 100
		detail := numberFa(percent) + "٪ × " + baseFa + " (" + moneyFa(float64(lineBase)) + " ریال)"
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

	// اضافه‌کاری بر پایهٔ مبنای فوق‌العاده‌ها و ساعت موظفی پارامتریک.
	if in.OvertimeHours > 0 && params.MonthlyHours > 0 {
		hourly := float64(allowanceBase) / params.MonthlyHours
		amount := RoundHalfUp(hourly * in.OvertimeHours * params.OvertimeFactor)
		line := OilPayLine{
			Key: "overtime", TitleFa: "اضافه‌کاری", GroupFa: OilGroupShift, Amount: amount,
			DetailFa: numberFa(in.OvertimeHours) + " ساعت × نرخ ساعتی مبنا × ضریب " + numberFa(params.OvertimeFactor),
		}
		out.OvertimeLine = &line
		out.Lines = append(out.Lines, line)
		out.Allowances += amount
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

	taxable := out.Gross - out.FundEmployee - out.InsuranceEmployee - params.TaxExemption
	if taxable < 0 {
		taxable = 0
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
	out.NoteFa = "نظام پرداخت صندوق بازنشستگی نفت: پایهٔ حقوق، فوق‌العادهٔ بدی آب و هوا (مناطق " +
		"گرمسیری) " + numberFa(climatePercent) + "٪ از پایه، و همهٔ فوق‌العاده‌های دیگر " +
		"(محرومیت، ویژه، صرف وقت، کارگاهی، جذب و تخصصی شغل، فرآورشی و نوبت‌کار) درصدی از " +
		"«مبنای فوق‌العاده‌ها» (" + moneyFa(float64(allowanceBase)) + " ریال) محاسبه می‌شوند."
	return out
}

// OilAllowances فهرست فوق‌العاده‌های نظام صندوق برای فرم و پنل تنظیمات.
func OilAllowances(params OilPayParams) []OilAllowance {
	out := make([]OilAllowance, len(params.Allowances))
	copy(out, params.Allowances)
	return out
}

// OilPercentFa درصد را با یک رقم اعشار در صورت لزوم برمی‌گرداند.
func OilPercentFa(v float64) string {
	return oilNumberFa(math.Round(v*10) / 10)
}

// oilNumberFa عدد را برای متن فارسی بدون صفر انتهایی برمی‌گرداند.
func oilNumberFa(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
