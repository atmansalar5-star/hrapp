package rules

// این فایل قاعدهٔ «کارکرد دوره» را می‌سازد: پل میان رکوردهای حضور (تردد، مرخصی،
// مأموریت، اضافه‌کاری) و موتور حقوق.
//
// قاعده اینجا می‌نشیند، نه در رابط کاربری: تبدیل ساعت به روز، تفکیک روز پرداختی از
// روز کسری، مزد روزانه و نرخ ساعتی اضافه‌کاری. ماژول حضور فقط جمع خام را می‌دهد و
// ماژول حقوق همین توابع را صدا می‌زند؛ پس یک منبع حقیقت وجود دارد.

// WorkTimeParams پارامترهای تبدیل کارکرد به مبالغ حقوق.
type WorkTimeParams struct {
	// MonthHours ساعت کار موظفی ماهانه (پارامتر شرکت: payroll.monthlyHours).
	MonthHours float64 `json:"monthHours"`
	// DayHours ساعت کار یک روز کاری.
	DayHours float64 `json:"dayHours"`
	// MonthDays روزهای مبنای مزد روزانه (رویهٔ حقوق و دستمزد: ۳۰ روز).
	MonthDays float64 `json:"monthDays"`
}

// DefaultWorkTimeParams مقادیر پیش‌فرض: ۱۷۶ ساعت در ماه، ۸ ساعت در روز، مبنای ۳۰ روز.
func DefaultWorkTimeParams() WorkTimeParams {
	return WorkTimeParams{MonthHours: 176, DayHours: 8, MonthDays: 30}
}

// WorkTimeInput جمع خام کارکرد یک کارمند در یک دوره (خروجی ماژول حضور).
type WorkTimeInput struct {
	Period          string  `json:"period"`
	WorkedHours     float64 `json:"workedHours"`
	OvertimeHours   float64 `json:"overtimeHours"`
	PaidLeaveDays   float64 `json:"paidLeaveDays"`
	UnpaidLeaveDays float64 `json:"unpaidLeaveDays"`
	SickLeaveDays   float64 `json:"sickLeaveDays"`
	MissionDays     float64 `json:"missionDays"`
	AbsenceDays     float64 `json:"absenceDays"`
}

// WorkTimeSummary کارکرد آمادهٔ استفاده در حقوق.
type WorkTimeSummary struct {
	WorkTimeInput
	// WorkedDays ساعت کارکرد تبدیل‌شده به روز.
	WorkedDays float64 `json:"workedDays"`
	// PayableDays روزهای قابل پرداخت: کارکرد + مرخصی استحقاقی/استعلاجی + مأموریت.
	PayableDays float64 `json:"payableDays"`
	// DeductibleDays روزهای کسری: مرخصی بی‌حقوق + غیبت.
	DeductibleDays float64 `json:"deductibleDays"`
	// TotalDays روزهای دوره (پرداختی + کسری).
	TotalDays float64 `json:"totalDays"`
	// PresenceRatio نسبت حضور (روز کارکرد به کل روزهای دوره) بین ۰ و ۱.
	PresenceRatio float64 `json:"presenceRatio"`
	// TotalPayableHours ساعت‌های قابل پرداخت: کارکرد + روزهای پرداختی × ساعت روز.
	TotalPayableHours float64 `json:"totalPayableHours"`
}

// ComputeWorkTime جمع خام کارکرد را به کارکرد قابل استفاده در حقوق تبدیل می‌کند.
func ComputeWorkTime(in WorkTimeInput, p WorkTimeParams) WorkTimeSummary {
	dayHours := p.DayHours
	if dayHours <= 0 {
		dayHours = 8
	}
	workedDays := round2(in.WorkedHours / dayHours)
	payable := round2(workedDays + in.PaidLeaveDays + in.SickLeaveDays + in.MissionDays)
	deductible := round2(in.UnpaidLeaveDays + in.AbsenceDays)
	total := round2(payable + deductible)
	ratio := 0.0
	if total > 0 {
		ratio = round2(workedDays / total)
	}
	return WorkTimeSummary{
		WorkTimeInput:     in,
		WorkedDays:        workedDays,
		PayableDays:       payable,
		DeductibleDays:    deductible,
		TotalDays:         total,
		PresenceRatio:     ratio,
		TotalPayableHours: round2(payable * dayHours),
	}
}

// DailyWage مزد یک روز کار بر پایهٔ مبنای روزهای ماه.
func DailyWage(baseSalary int64, p WorkTimeParams) float64 {
	days := p.MonthDays
	if days <= 0 {
		days = 30
	}
	return float64(baseSalary) / days
}

// HourlyWage مزد یک ساعت کار: مزد ماهانه تقسیم بر (روزهای ماه × ساعت روز).
func HourlyWage(baseSalary int64, p WorkTimeParams) float64 {
	dayHours := p.DayHours
	if dayHours <= 0 {
		dayHours = 8
	}
	return DailyWage(baseSalary, p) / dayHours
}

// UnpaidDeduction کسر مرخصی بی‌حقوق و غیبت بر پایهٔ مزد روزانه.
func UnpaidDeduction(deductibleDays float64, baseSalary int64, p WorkTimeParams) int64 {
	if deductibleDays <= 0 || baseSalary <= 0 {
		return 0
	}
	return RoundHalfUp(deductibleDays * DailyWage(baseSalary, p))
}

// OvertimeHourlyRate نرخ ساعتی اضافه‌کاری: مزد ساعتی × ضریب مصوب.
func OvertimeHourlyRate(baseSalary int64, factor float64, p WorkTimeParams) int64 {
	if baseSalary <= 0 || factor <= 0 {
		return 0
	}
	return RoundHalfUp(HourlyWage(baseSalary, p) * factor)
}

// DefaultOvertimeFactor ضریب پیش‌فرض اضافه‌کاری (۱٫۴) وقتی رکورد ضریب مشخصی نداشته باشد.
const DefaultOvertimeFactor = 1.4
