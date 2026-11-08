// Package rules — قاعدهٔ «تفاوت تطبیق حقوق و مزایا».
//
// چرا اینجا: وقتی طرح حقوق و مزایا (مثلاً طرح طبقه‌بندی مشاغل) اجرا می‌شود، ممکن است حقوق
// و مزایای محاسبه‌شدهٔ کارمند بر پایهٔ طرح تازه از جمع پیشین او کمتر شود. در این حالت
// مابه‌التفاوت پیشین به‌عنوان «تفاوت تطبیق» پرداخت می‌شود و با افزایش‌های بعدی جذب می‌شود.
// تصمیم «آیا مشمول تفاوت تطبیق است، چقدر، و با هر افزایش چقدر کم می‌شود» یک قاعدهٔ پرداخت
// است، نه یک محاسبهٔ نمایشی؛ اگر در فرم نوشته شود، فرم وب و فیش چاپی و اپلیکیشن سه عدد
// می‌سازند.
//
// همهٔ آستانه‌ها پارامتریک‌اند (پارامتر شرکت):
//
//	paygap.roundTo          گرد کردن مبلغ تفاوت تطبیق (ریال)
//	paygap.absorptionRatio  نسبت جذب با هر افزایش (۱ = جذب کامل)
//	paygap.inWageBase       آیا تفاوت تطبیق جزو «مزد مبنا» است
//	paygap.insurable        آیا مشمول بیمه است
//	paygap.taxable          آیا مشمول مالیات است
package rules

import (
	"strconv"
	"strings"
)

// PayGapParams آستانه‌های تفاوت تطبیق از پارامترهای شرکت.
type PayGapParams struct {
	// RoundTo گِرد کردن مبلغ تفاوت تطبیق به این مضرب (ریال)؛ صفر یعنی بدون گرد کردن.
	RoundTo int64 `json:"roundTo"`
	// AbsorptionRatio نسبت جذب با هر افزایش؛ ۱ یعنی تمام افزایش، تفاوت تطبیق را کم می‌کند.
	AbsorptionRatio float64 `json:"absorptionRatio"`
	// InWageBase آیا تفاوت تطبیق جزو مزد مبنا شمرده می‌شود (پیش‌فرض: نه).
	InWageBase bool `json:"inWageBase"`
	// Insurable و Taxable مبنای کسور؛ پیش‌فرض: تفاوت تطبیق بخشی از حقوق و مزایای مشمول است.
	Insurable bool `json:"insurable"`
	Taxable   bool `json:"taxable"`
}

// DefaultPayGapParams پیش‌فرض‌های اعلام‌شدهٔ استقرار.
func DefaultPayGapParams() PayGapParams {
	return PayGapParams{
		RoundTo:         10_000,
		AbsorptionRatio: 1,
		InWageBase:      false,
		Insurable:       true,
		Taxable:         true,
	}
}

// PayGapParamsFromValues پارامترهای شرکت را روی پیش‌فرض می‌نشاند؛ کلید غایب پیش‌فرض می‌ماند.
func PayGapParamsFromValues(values map[string]float64) PayGapParams {
	p := DefaultPayGapParams()
	if v, ok := values["paygap.roundTo"]; ok && v > 0 {
		p.RoundTo = int64(v)
	}
	if v, ok := values["paygap.absorptionRatio"]; ok && v > 0 {
		p.AbsorptionRatio = v
		if p.AbsorptionRatio > 1 {
			p.AbsorptionRatio = 1
		}
	}
	if v, ok := values["paygap.inWageBase"]; ok {
		p.InWageBase = v != 0
	}
	if v, ok := values["paygap.insurable"]; ok {
		p.Insurable = v != 0
	}
	if v, ok := values["paygap.taxable"]; ok {
		p.Taxable = v != 0
	}
	return p
}

// PayGapLine یک سطر از فرم محاسباتی تفاوت تطبیق.
type PayGapLine struct {
	Order    int    `json:"order"`
	Key      string `json:"key"`
	TitleFa  string `json:"titleFa"`
	AmountFa string `json:"amountFa"`
	DetailFa string `json:"detailFa"`
	// Amount مبلغ ریالی سطر (برای جمع‌زدن و نمایش عددی).
	Amount int64 `json:"amount"`
}

// PayGapInput ورودی تطبیق یک کارمند: جمع حقوق و مزایای پیشین و جمع بر پایهٔ طرح تازه.
//
// هر دو جمع باید از یک جا بیایند: «مزد مبنا + پرداخت‌های ثابت دوره»؛ اگر یکی از دو سمت
// با مبنای دیگری حساب شود، نتیجهٔ تطبیق بی‌اعتبار است.
type PayGapInput struct {
	EmployeeID    string `json:"employeeId,omitempty"`
	EmployeeName  string `json:"employeeName,omitempty"`
	PreviousTotal int64  `json:"previousTotal"`
	NewTotal      int64  `json:"newTotal"`
}

// PayGapResult نتیجهٔ تطبیق یک کارمند.
type PayGapResult struct {
	PreviousTotal int64 `json:"previousTotal"`
	NewTotal      int64 `json:"newTotal"`
	// Difference اختلاف دو جمع (پیشین − تازه)؛ منفی یعنی طرح تازه بیشتر است.
	Difference int64 `json:"difference"`
	// PayGap مبلغ تفاوت تطبیق ماهانه (صفر یعنی مشمول تفاوت تطبیق نیست).
	PayGap int64 `json:"payGap"`
	// Covered آیا کارمند مشمول تفاوت تطبیق می‌شود.
	Covered   bool         `json:"covered"`
	Lines     []PayGapLine `json:"lines"`
	SummaryFa string       `json:"summaryFa"`
	NoteFa    string       `json:"noteFa,omitempty"`
}

// ComputePayGap تطبیق حقوق یک کارمند را می‌سنجد و فرم محاسباتی را می‌سازد.
//
// قاعده‌ها (همه در همین تابع اعلام شده‌اند):
//
//	۱. اگر جمع طرح تازه کمتر از جمع پیشین باشد، مابه‌التفاوت «تفاوت تطبیق» است.
//	۲. مبلغ تفاوت تطبیق به مضرب «paygap.roundTo» گرد می‌شود (نیم‌به‌بالا)؛ گِرد کردن
//	   به بالاتر از مابه‌التفاوت مجاز است، چون جذب‌شدنی است و کارمند را متضرر نمی‌کند.
//	۳. اگر جمع تازه برابر یا بیشتر باشد، تفاوت تطبیقی ساخته نمی‌شود (کارمند مشمول نیست).
//	۴. تفاوت تطبیق جزو مزد مبنا نیست (مگر پارامتر شرکت بگوید)، ولی بخشی از حقوق و مزایای
//	   مشمول بیمه و مالیات است مگر پارامتر شرکت خلاف آن را بگوید.
func ComputePayGap(in PayGapInput, p PayGapParams) PayGapResult {
	res := PayGapResult{
		PreviousTotal: in.PreviousTotal,
		NewTotal:      in.NewTotal,
		Difference:    in.PreviousTotal - in.NewTotal,
		Lines:         []PayGapLine{},
	}
	res.Lines = append(res.Lines, PayGapLine{
		Order: 1, Key: "previous", TitleFa: "جمع حقوق و مزایای پیشین",
		Amount: in.PreviousTotal, AmountFa: IntMoneyFa(in.PreviousTotal),
		DetailFa: "جمع مزد مبنا و پرداخت‌های ثابت دورهٔ پیش از اجرای طرح تازه.",
	})
	res.Lines = append(res.Lines, PayGapLine{
		Order: 2, Key: "current", TitleFa: "جمع حقوق و مزایا بر پایهٔ طرح تازه",
		Amount: in.NewTotal, AmountFa: IntMoneyFa(in.NewTotal),
		DetailFa: "جمع همان اجزا با نرخ‌ها و جدول‌های تازه.",
	})

	gap := int64(0)
	covered := res.Difference > 0
	if covered {
		gap = res.Difference
		if p.RoundTo > 0 {
			gap = roundUpTo(gap, p.RoundTo)
		}
	}
	res.PayGap = gap
	res.Covered = covered

	switch {
	case !covered && res.Difference == 0:
		res.Lines = append(res.Lines, PayGapLine{
			Order: 3, Key: "difference", TitleFa: "اختلاف دو جمع",
			AmountFa: "بدون اختلاف", DetailFa: "دو جمع برابرند؛ تفاوت تطبیقی ساخته نمی‌شود.",
		})
	case !covered:
		res.Lines = append(res.Lines, PayGapLine{
			Order: 3, Key: "difference", TitleFa: "اختلاف دو جمع",
			Amount: res.Difference, AmountFa: IntMoneyFa(-res.Difference),
			DetailFa: "طرح تازه بیشتر از جمع پیشین است؛ کارمند مشمول تفاوت تطبیق نمی‌شود.",
		})
	default:
		res.Lines = append(res.Lines, PayGapLine{
			Order: 3, Key: "difference", TitleFa: "مابه‌التفاوت پیشین",
			Amount: res.Difference, AmountFa: IntMoneyFa(res.Difference),
			DetailFa: "جمع پیشین از طرح تازه بیشتر است؛ این اختلاف مبنای تفاوت تطبیق است.",
		})
		res.Lines = append(res.Lines, PayGapLine{
			Order: 4, Key: "round", TitleFa: "گرد کردن مبلغ",
			Amount: gap, AmountFa: IntMoneyFa(gap),
			DetailFa: "مبلغ تفاوت تطبیق به مضرب " + IntMoneyFa(p.RoundTo) +
				" ریال گرد شد (جذب‌شدنی است و به نفع کارمند گرد می‌شود).",
		})
		res.Lines = append(res.Lines, PayGapLine{
			Order: 5, Key: "basis", TitleFa: "جایگاه در شاکلهٔ مزد",
			AmountFa: payGapBasisFa(p),
			DetailFa: "تفاوت تطبیق مبلغی ثابت و جذب‌شدنی است: با هر افزایش حقوق، به نسبت " +
				percentageFa(p.AbsorptionRatio) + " کاهش می‌یابد تا به صفر برسد.",
		})
	}

	res.SummaryFa = payGapSummaryFa(res)
	res.NoteFa = "تفاوت تطبیق تا زمانی پرداخت می‌شود که جذب نشده باشد؛ در هر افزایش حقوق، " +
		"مبلغ افزایش به نسبت تعیین‌شده از تفاوت تطبیق کم می‌کند و باقی‌ماندهٔ آن در فیش و حکم " +
		"ماه‌های بعد می‌ماند. جذب کامل، پایان تفاوت تطبیق است."
	res.SummaryFa = persianDigits(res.SummaryFa)
	res.NoteFa = persianDigits(res.NoteFa)
	for i := range res.Lines {
		res.Lines[i].AmountFa = persianDigits(res.Lines[i].AmountFa)
		res.Lines[i].DetailFa = persianDigits(res.Lines[i].DetailFa)
	}
	return res
}

// payGapBasisFa متن جایگاه تفاوت تطبیق در شاکلهٔ مزد.
func payGapBasisFa(p PayGapParams) string {
	parts := []string{}
	if p.InWageBase {
		parts = append(parts, "جزو مزد مبنا")
	} else {
		parts = append(parts, "خارج از مزد مبنا")
	}
	if p.Insurable {
		parts = append(parts, "مشمول بیمه")
	} else {
		parts = append(parts, "بدون کسور بیمه")
	}
	if p.Taxable {
		parts = append(parts, "مشمول مالیات")
	} else {
		parts = append(parts, "معاف از مالیات")
	}
	return strings.Join(parts, "، ")
}

// payGapSummaryFa جملهٔ جمع‌بندی تطبیق.
func payGapSummaryFa(res PayGapResult) string {
	if !res.Covered {
		return "این کارمند مشمول تفاوت تطبیق نمی‌شود؛ حقوق و مزایای طرح تازه از جمع پیشین کمتر نیست."
	}
	return "کارمند مشمول تفاوت تطبیق است؛ ماهانه " + IntMoneyFa(res.PayGap) +
		" ریال پرداخت می‌شود و با افزایش‌های بعدی جذب می‌گردد."
}

// PayGapAbsorption نتیجهٔ جذب تفاوت تطبیق با یک افزایش.
type PayGapAbsorption struct {
	// Gap تفاوت تطبیق پیش از افزایش.
	Gap int64 `json:"gap"`
	// Increase مبلغ افزایش حقوق در این مرحله.
	Increase int64 `json:"increase"`
	// Reduction مبلغی که از تفاوت تطبیق کم شد.
	Reduction int64 `json:"reduction"`
	// Remaining تفاوت تطبیق پس از جذب.
	Remaining int64 `json:"remaining"`
	// FullyAbsorbed آیا تفاوت تطبیق به صفر رسید.
	FullyAbsorbed bool   `json:"fullyAbsorbed"`
	DetailFa      string `json:"detailFa"`
}

// AbsorbPayGap اثر یک افزایش را روی تفاوت تطبیق می‌سنجد.
//
// قاعده: کاهش = افزایش × نسبت جذب؛ هرگز بیش از مبلغ تفاوت تطبیق کاهش نمی‌یابد، پس عدد
// منفی ساخته نمی‌شود. اگر افزایش کامل تفاوت تطبیق را بپوشاند، «جذب کامل» رخ می‌دهد.
func AbsorbPayGap(gap, increase int64, p PayGapParams) PayGapAbsorption {
	out := PayGapAbsorption{Gap: gap, Increase: increase}
	if gap <= 0 || increase <= 0 {
		out.Remaining = nonNegative64(gap)
		out.DetailFa = "افزایشی برای جذب ثبت نشده است؛ تفاوت تطبیق بدون تغییر می‌ماند."
		return out
	}
	reduction := RoundHalfUp(float64(increase) * p.AbsorptionRatio)
	if reduction > gap {
		reduction = gap
	}
	if reduction < 0 {
		reduction = 0
	}
	out.Reduction = reduction
	out.Remaining = gap - reduction
	out.FullyAbsorbed = out.Remaining == 0
	if out.FullyAbsorbed {
		out.DetailFa = "افزایش این مرحله تفاوت تطبیق را به‌طور کامل جذب کرد."
	} else {
		out.DetailFa = "باقی‌ماندهٔ تفاوت تطبیق " + IntMoneyFa(out.Remaining) +
			" ریال است و در پرداخت‌های بعدی ادامه می‌یابد."
	}
	return out
}

// AbsorbPayGapSeries اثر زنجیره‌ای افزایش‌ها را می‌سنجد (نمودار جذب در فرم و سند).
func AbsorbPayGapSeries(gap int64, increases []int64, p PayGapParams) (int64, []PayGapAbsorption) {
	remaining := nonNegative64(gap)
	steps := make([]PayGapAbsorption, 0, len(increases))
	for _, inc := range increases {
		step := AbsorbPayGap(remaining, inc, p)
		steps = append(steps, step)
		remaining = step.Remaining
	}
	return remaining, steps
}

// roundUpTo مبلغ را به مضرب گِردشده به بالا می‌برد (نیم‌به‌بالا پس از تقسیم).
func roundUpTo(amount, roundTo int64) int64 {
	if roundTo <= 0 || amount <= 0 {
		return amount
	}
	steps := float64(amount) / float64(roundTo)
	return int64(RoundHalfUp(steps)) * roundTo
}

// nonNegative64 مقدار منفی را صفر می‌کند.
func nonNegative64(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

// IntMoneyFa یک مبلغ ریالی را با جداکنندهٔ هزارگان و ارقام فارسی برمی‌گرداند.
//
// چرا مبلغ با جداکننده: تفاوت تطبیق در فرم و سند چاپی خوانده می‌شود و «۱۲٬۵۰۰٬۰۰۰» از
// «12500000» خواناتر است. داده و API همان عدد لاتین را نگه می‌دارند.
func IntMoneyFa(amount int64) string {
	digits := strconv.FormatInt(amount, 10)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign = "−"
		digits = strings.TrimPrefix(digits, "-")
	}
	out := ""
	count := 0
	for i := len(digits) - 1; i >= 0; i-- {
		out = string(digits[i]) + out
		count++
		if count%3 == 0 && i > 0 {
			out = "٬" + out
		}
	}
	return persianDigits(sign + out)
}

// percentageFa نسبت را به درصد فارسی تبدیل می‌کند (۰٫۵ → ۵۰٪).
func percentageFa(ratio float64) string {
	percent := ratio * 100
	text := strconv.FormatFloat(percent, 'f', -1, 64)
	return persianDigits(text) + "٪"
}

// PayGapInstructionFa متن دستورالعمل «تفاوت تطبیق حقوق و مزایا».
//
// چرا در دامنه: دستورالعمل بخشی از قاعده است؛ فرم تطبیق، فیش حقوق و سند حکم باید یک
// فرآیند را نشان دهند تا کسی گام تازه‌ای از خودش نسازد.
func PayGapInstructionFa() []string {
	return []string{
		"۱. پیش از اجرای طرح تازه (طبقه‌بندی مشاغل یا بازنگری حقوق)، جمع «مزد مبنا + پرداخت‌های ثابت دوره» هر کارمند بر پایهٔ طرح پیشین ثبت می‌شود.",
		"۲. همان جمع با نرخ‌ها و جدول‌های طرح تازه برای همان کارمند محاسبه می‌شود؛ دو سمت مقایسه باید یک مبنای محاسبه داشته باشند.",
		"۳. اگر جمع طرح تازه کمتر از جمع پیشین شود، مابه‌التفاوت «تفاوت تطبیق» است و کارمند مشمول آن می‌شود.",
		"۴. مبلغ تفاوت تطبیق به مضرب پارامتر شرکت گرد می‌شود؛ گرد کردن به بالا، چون مبلغ جذب‌شدنی است و حقوق کارمند را کاهش نمی‌دهد.",
		"۵. تفاوت تطبیق جزو «مزد مبنا» نیست، پس مبنای اضافه‌کاری و فوقالعاده‌های درصدی را بالا نمی‌برد؛ ولی بخشی از حقوق و مزایای مشمول بیمه و مالیات است، مگر پارامتر شرکت خلاف آن را تعیین کند.",
		"۶. مبلغ تفاوت تطبیق در حکم حقوق (فرمت حقوق) ثبت می‌شود و در فیش حقوقی به‌صورت سطر جداگانه در تفکیک اقلام می‌آید.",
		"۷. با هر افزایش حقوق (حکم تازه، ارتقای گروه یا افزایش گروهی)، به نسبت پارامتر «نسبت جذب»، از تفاوت تطبیق کاسته می‌شود.",
		"۸. هرگز مبلغ افزایش بیش از تفاوت تطبیق باقی‌مانده جذب نمی‌شود؛ تفاوت تطبیق هرگز منفی یا بدهی نمی‌شود.",
		"۹. وقتی تفاوت تطبیق به صفر رسید، پرداخت آن پایان می‌یابد و در حکم بعدی صفر ثبت می‌شود؛ سابقهٔ آن در گردش کار حکم می‌ماند.",
		"۱۰. تغییر در پارامترهای شرکت (نسبت جذب، گرد کردن، مبانی کسور) از تاریخ اجرای خود اثر می‌گذارد و محاسبهٔ پیشین را باطل نمی‌کند.",
	}
}
