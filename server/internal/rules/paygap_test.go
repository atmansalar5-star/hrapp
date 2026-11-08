// آزمون قاعدهٔ «تفاوت تطبیق» — تطبیق حقوق و جذب آن.

package rules

import "testing"

// gapParams پارامترهای نمونهٔ آزمون.
func gapParams() PayGapParams {
	p := DefaultPayGapParams()
	p.RoundTo = 10_000
	p.AbsorptionRatio = 1
	return p
}

// TestPayGapWhenNewPayIsLower مابه‌التفاوت کمتر بودن طرح تازه، تفاوت تطبیق می‌سازد.
func TestPayGapWhenNewPayIsLower(t *testing.T) {
	res := ComputePayGap(PayGapInput{PreviousTotal: 123_456_789, NewTotal: 100_000_000}, gapParams())
	if !res.Covered {
		t.Fatal("کارمند باید مشمول تفاوت تطبیق شود")
	}
	// ۲۳٬۴۵۶٬۷۸۹ به مضرب ۱۰٬۰۰۰ به بالا: ۲۳٬۴۶۰٬۰۰۰
	if res.PayGap != 23_460_000 {
		t.Fatalf("مبلغ تفاوت تطبیق نادرست است: %d", res.PayGap)
	}
	if res.Difference != 23_456_789 {
		t.Fatalf("مابه‌التفاوت نادرست است: %d", res.Difference)
	}
	if len(res.Lines) < 5 {
		t.Fatalf("فرم محاسباتی باید دست‌کم ۵ سطر داشته باشد: %d", len(res.Lines))
	}
}

// TestPayGapNotCoveredWhenNewPayIsHigher جمع بالاتر طرح تازه، تفاوت تطبیق نمی‌سازد.
func TestPayGapNotCoveredWhenNewPayIsHigher(t *testing.T) {
	res := ComputePayGap(PayGapInput{PreviousTotal: 80_000_000, NewTotal: 95_000_000}, gapParams())
	if res.Covered || res.PayGap != 0 {
		t.Fatalf("بدون کاهش حقوق نباید تفاوت تطبیق باشد: %+v", res)
	}
	if res.Difference != -15_000_000 {
		t.Fatalf("اختلاف باید منفی باشد: %d", res.Difference)
	}
}

// TestPayGapEqualTotals جمع برابر، تفاوت تطبیق نمی‌سازد.
func TestPayGapEqualTotals(t *testing.T) {
	res := ComputePayGap(PayGapInput{PreviousTotal: 50_000_000, NewTotal: 50_000_000}, gapParams())
	if res.Covered || res.PayGap != 0 {
		t.Fatalf("جمع برابر نباید تفاوت تطبیق بسازد: %+v", res)
	}
}

// TestPayGapRoundToZero بدون گرد کردن، مبلغ عیناً مابه‌التفاوت است.
func TestPayGapRoundToZero(t *testing.T) {
	p := gapParams()
	p.RoundTo = 0
	res := ComputePayGap(PayGapInput{PreviousTotal: 12_345_678, NewTotal: 10_000_000}, p)
	if res.PayGap != 2_345_678 {
		t.Fatalf("بدون گرد کردن باید مابه‌التفاوت عیناً بیاید: %d", res.PayGap)
	}
}

// TestPayGapAbsorptionFullIncrease جذب کامل با افزایش بزرگ‌تر از باقی‌مانده.
func TestPayGapAbsorptionFullIncrease(t *testing.T) {
	step := AbsorbPayGap(5_000_000, 9_000_000, gapParams())
	if !step.FullyAbsorbed || step.Remaining != 0 {
		t.Fatalf("افزایش بزرگ‌تر باید تفاوت تطبیق را کامل جذب کند: %+v", step)
	}
	if step.Reduction != 5_000_000 {
		t.Fatalf("جذب نباید از خود تفاوت تطبیق بیشتر شود: %d", step.Reduction)
	}
}

// TestPayGapAbsorptionPartialIncrease جذب جزئی با افزایش کوچک‌تر.
func TestPayGapAbsorptionPartialIncrease(t *testing.T) {
	step := AbsorbPayGap(5_000_000, 2_000_000, gapParams())
	if step.FullyAbsorbed || step.Remaining != 3_000_000 || step.Reduction != 2_000_000 {
		t.Fatalf("جذب جزئی نادرست است: %+v", step)
	}
}

// TestPayGapAbsorptionRatio نسبت جذب کمتر از یک، بخشی از افزایش را جذب می‌کند.
func TestPayGapAbsorptionRatio(t *testing.T) {
	p := gapParams()
	p.AbsorptionRatio = 0.5
	step := AbsorbPayGap(10_000_000, 4_000_000, p)
	if step.Reduction != 2_000_000 || step.Remaining != 8_000_000 {
		t.Fatalf("با نسبت ۵۰٪ باید ۲ میلیون جذب شود: %+v", step)
	}
}

// TestPayGapAbsorptionSeries زنجیرهٔ افزایش‌ها باقی‌مانده را به صفر می‌رساند.
func TestPayGapAbsorptionSeries(t *testing.T) {
	remaining, steps := AbsorbPayGapSeries(6_000_000, []int64{2_000_000, 2_000_000, 2_000_000}, gapParams())
	if remaining != 0 {
		t.Fatalf("پس از سه افزایش دو میلیونی باید صفر شود: %d", remaining)
	}
	if len(steps) != 3 || !steps[2].FullyAbsorbed {
		t.Fatalf("مرحلهٔ پایانی باید جذب کامل باشد: %+v", steps)
	}
}

// TestPayGapParamsFromValues پارامتر شرکت روی پیش‌فرض می‌نشیند.
func TestPayGapParamsFromValues(t *testing.T) {
	p := PayGapParamsFromValues(map[string]float64{
		"paygap.roundTo":          50_000,
		"paygap.absorptionRatio":  0.75,
		"paygap.inWageBase":       1,
		"paygap.insurable":        0,
		"paygap.absorptionRatio2": 9,
	})
	if p.RoundTo != 50_000 || p.AbsorptionRatio != 0.75 || !p.InWageBase || p.Insurable {
		t.Fatalf("پارامترها نادرست نشستند: %+v", p)
	}
	if p.Taxable != DefaultPayGapParams().Taxable {
		t.Fatal("کلید غایب باید پیش‌فرض بماند")
	}
	// نسبت جذب بیش از یک به یک محدود می‌شود (بیش از ۱۰۰٪ جذب معنا ندارد).
	p = PayGapParamsFromValues(map[string]float64{"paygap.absorptionRatio": 3})
	if p.AbsorptionRatio != 1 {
		t.Fatalf("نسبت جذب باید به ۱ محدود شود: %v", p.AbsorptionRatio)
	}
}

// TestPayGapInstructionCoversSteps دستورالعمل باید گام‌های اصلی را داشته باشد.
func TestPayGapInstructionCoversSteps(t *testing.T) {
	steps := PayGapInstructionFa()
	if len(steps) < 8 {
		t.Fatalf("دستورالعمل تفاوت تطبیق باید دست‌کم ۸ گام داشته باشد: %d", len(steps))
	}
}

// TestIntMoneyFaThousands مبلغ با جداکنندهٔ هزارگان و ارقام فارسی.
func TestIntMoneyFaThousands(t *testing.T) {
	if got := IntMoneyFa(12_500_000); got != "۱۲٬۵۰۰٬۰۰۰" {
		t.Fatalf("قالب مبلغ نادرست است: %s", got)
	}
	if got := IntMoneyFa(-1_500); got != "−۱٬۵۰۰" {
		t.Fatalf("قالب مبلغ منفی نادرست است: %s", got)
	}
	if got := IntMoneyFa(0); got != "۰" {
		t.Fatalf("صفر باید «۰» باشد: %s", got)
	}
}
