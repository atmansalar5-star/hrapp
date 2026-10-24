package rules

import "testing"

// کارکرد: تبدیل ساعت به روز و تفکیک روز پرداختی از روز کسری.
func TestComputeWorkTimeSplitsPayableAndDeductibleDays(t *testing.T) {
	in := WorkTimeInput{
		Period: "1404/07", WorkedHours: 160, OvertimeHours: 12,
		PaidLeaveDays: 2, UnpaidLeaveDays: 3, SickLeaveDays: 1, MissionDays: 3, AbsenceDays: 1,
	}
	got := ComputeWorkTime(in, DefaultWorkTimeParams())

	if got.WorkedDays != 20 {
		t.Fatalf("روز کارکرد باید ۲۰ باشد، شد %v", got.WorkedDays)
	}
	// پرداختی = ۲۰ (کارکرد) + ۲ (استحقاقی) + ۱ (استعلاجی) + ۳ (مأموریت)
	if got.PayableDays != 26 {
		t.Fatalf("روز پرداختی باید ۲۶ باشد، شد %v", got.PayableDays)
	}
	// کسری = ۳ (بی‌حقوق) + ۱ (غیبت)
	if got.DeductibleDays != 4 {
		t.Fatalf("روز کسری باید ۴ باشد، شد %v", got.DeductibleDays)
	}
	if got.TotalDays != 30 {
		t.Fatalf("کل روزها باید ۳۰ باشد، شد %v", got.TotalDays)
	}
	// نسبت حضور = ۲۰ از ۳۰
	if got.PresenceRatio != 0.67 {
		t.Fatalf("نسبت حضور باید ۰٫۶۷ باشد، شد %v", got.PresenceRatio)
	}
	if got.TotalPayableHours != 208 {
		t.Fatalf("ساعت پرداختی باید ۲۰۸ باشد، شد %v", got.TotalPayableHours)
	}
}

// کارکرد خالی: هیچ روزی شمرده نمی‌شود و تقسیم بر صفر رخ نمی‌دهد.
func TestComputeWorkTimeWithEmptyInput(t *testing.T) {
	got := ComputeWorkTime(WorkTimeInput{}, DefaultWorkTimeParams())
	if got.PayableDays != 0 || got.DeductibleDays != 0 || got.TotalDays != 0 || got.PresenceRatio != 0 {
		t.Fatalf("کارکرد خالی باید همه‌جا صفر بماند: %+v", got)
	}
}

// مزد روزانه و کسر مرخصی بی‌حقوق (مبنای ۳۰ روز).
func TestUnpaidDeductionUsesMonthlyDayBase(t *testing.T) {
	params := DefaultWorkTimeParams()
	if DailyWage(60_000_000, params) != 2_000_000 {
		t.Fatalf("مزد روزانه باید ۲٬۰۰۰٬۰۰۰ باشد")
	}
	if got := UnpaidDeduction(3, 60_000_000, params); got != 6_000_000 {
		t.Fatalf("کسر ۳ روز بی‌حقوق باید ۶٬۰۰۰٬۰۰۰ باشد، شد %d", got)
	}
	if got := UnpaidDeduction(0, 60_000_000, params); got != 0 {
		t.Fatalf("بدون روز کسری نباید کسری ثبت شود، شد %d", got)
	}
}

// نرخ ساعتی اضافه‌کاری با ضریب مصوب.
func TestOvertimeHourlyRateAppliesFactor(t *testing.T) {
	params := DefaultWorkTimeParams()
	// ۶۰٬۰۰۰٬۰۰۰ / ۳۰ / ۸ = ۲۵۰٬۰۰۰ و × ۱٫۴ = ۳۵۰٬۰۰۰
	if got := OvertimeHourlyRate(60_000_000, DefaultOvertimeFactor, params); got != 350_000 {
		t.Fatalf("نرخ ساعتی اضافه‌کاری باید ۳۵۰٬۰۰۰ باشد، شد %d", got)
	}
	// همان نرخ در ورودی موتور حقوق، همان مبلغ را می‌دهد.
	rate := OvertimeHourlyRate(60_000_000, DefaultOvertimeFactor, params)
	result := ComputePayroll(PayrollInput{
		BaseSalary: 60_000_000, OvertimeHours: 12, OvertimeHourlyRate: rate,
	}, DefaultPayrollParams())
	if result.OvertimePay != 4_200_000 {
		t.Fatalf("اضافه‌کاری ۱۲ ساعت باید ۴٬۲۰۰٬۰۰۰ باشد، شد %d", result.OvertimePay)
	}
}
