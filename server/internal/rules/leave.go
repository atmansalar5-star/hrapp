// قاعدهٔ محاسباتی موتور قواعد — «leave».

package rules

import (
	"hrms/internal/platform/calendar"
)

// LeaveParams پارامترهای مرخصی — از ماژول تنظیمات می‌آید، نه از کد.
type LeaveParams struct {
	AccrualPerMonth float64           `json:"accrualPerMonth"`
	AnnualCap       float64           `json:"annualCap"`
	CarryCap        float64           `json:"carryCap"`
	Weekend         []int             `json:"weekend"`
	Holidays        map[string]string `json:"holidays"`
}

// DefaultLeaveParams مقادیر پیش‌فرض قانون کار: ۲٫۵ روز در ماه، سقف ۳۰ روز و انتقال حداقل ۹ روز.
func DefaultLeaveParams() LeaveParams {
	return LeaveParams{
		AccrualPerMonth: 2.5,
		AnnualCap:       30,
		CarryCap:        9,
		Weekend:         []int{6},
		Holidays:        map[string]string{},
	}
}

// Calendar تقویم کاری متناظر با این پارامترها.
func (p LeaveParams) Calendar() calendar.WorkCalendar {
	holidays := p.Holidays
	if holidays == nil {
		holidays = map[string]string{}
	}
	weekend := p.Weekend
	if weekend == nil {
		weekend = []int{6}
	}
	return calendar.WorkCalendar{Weekend: weekend, Holidays: holidays}
}

// CountLeaveDays شمار روزهای مرخصی یک بازه.
// `sandwich = false` تنها روزهای کاری را می‌شمارد و `true` قاعدهٔ ساندویچ را اعمال می‌کند.
func CountLeaveDays(from, to calendar.Date, p LeaveParams, sandwich bool) (float64, error) {
	c := p.Calendar()
	if sandwich {
		n, err := c.SandwichDays(from, to)
		if err != nil {
			return 0, err
		}
		return float64(n), nil
	}
	n, err := c.WorkingDays(from, to)
	if err != nil {
		return 0, err
	}
	return float64(n), nil
}

// RawAccrual تعهدی خام پیش از اعمال سقف سالانه.
func RawAccrual(monthsWorked int, p LeaveParams) float64 {
	return round2(float64(monthsWorked) * p.AccrualPerMonth)
}

// CappedAccrual تعهدی پس از اعمال سقف سالانه.
func CappedAccrual(monthsWorked int, p LeaveParams) float64 {
	raw := RawAccrual(monthsWorked, p)
	if raw > p.AnnualCap {
		return p.AnnualCap
	}
	return raw
}

// CarryForward انتقال مانده به سال بعد با اعمال سقف انتقال.
func CarryForward(balance, carryCap float64) float64 {
	if balance > carryCap {
		balance = carryCap
	}
	if balance < 0 {
		balance = 0
	}
	return round2(balance)
}

// IsEligible آیا مانده برای درخواست کافی است.
func IsEligible(balance, requested float64) bool { return balance >= requested }

// BalanceAfter مانده پس از تأیید؛ در صورت ناکافی بودن، مقدار و false.
func BalanceAfter(balance, requested float64) (float64, bool) {
	if !IsEligible(balance, requested) {
		return balance, false
	}
	return round2(balance - requested), true
}

// AccrualPlan برنامهٔ تعهدی دوازده ماه یک سال کارکرد (برای صفحهٔ موجودی مرخصی).
func AccrualPlan(p LeaveParams) []float64 {
	out := make([]float64, 0, 12)
	for m := 1; m <= 12; m++ {
		out = append(out, CappedAccrual(m, p))
	}
	return out
}
