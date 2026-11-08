// Package rules میزبان دوم قواعد کسب‌وکار (سرور).
//
// قواعد مرجع در `core/crates/hrms-rules` نوشته می‌شوند و خروجی آن‌ها در
// `core/testdata/conformance/*.json` ثابت می‌شود. این پکیج همان قواعد را برای اجرای
// سمت سرور پیاده می‌کند و آزمون‌های `parity_test.go` آن را با fixtureهای طلایی
// می‌سنجند. اگر دو میزبان از هم جدا بیفتند، CI شکست می‌خورد.
//
// ماژول‌هایی که به قاعده نیاز دارند (حقوق، مرخصی، عملکرد) از همین پکیج استفاده
// می‌کنند و برای محاسبه، سرور همیشه مرجع نهایی است.
package rules

import (
	"math"
)

// TaxBracket یک پلهٔ مالیاتی؛ `Upto = 0` یعنی بی‌سقف.
type TaxBracket struct {
	Upto int64   `json:"upto"`
	Rate float64 `json:"rate"`
}

// PayrollParams پارامترهای محاسبهٔ حقوق (از ماژول تنظیمات، نسخه‌دار).
type PayrollParams struct {
	InsuranceEmployeeRate  float64      `json:"insuranceEmployeeRate"`
	InsuranceEmployerRate  float64      `json:"insuranceEmployerRate"`
	InsuranceCeiling       int64        `json:"insuranceCeiling"`
	HousingAllowance       int64        `json:"housingAllowance"`
	ChildAllowancePerChild int64        `json:"childAllowancePerChild"`
	TaxExemption           int64        `json:"taxExemption"`
	TaxBrackets            []TaxBracket `json:"taxBrackets"`
}

// PayrollInput ورودی محاسبهٔ یک کارمند در یک دوره.
type PayrollInput struct {
	BaseSalary         int64 `json:"baseSalary"`
	Children           int   `json:"children"`
	OvertimeHours      int64 `json:"overtimeHours"`
	OvertimeHourlyRate int64 `json:"overtimeHourlyRate"`
	OtherDeductions    int64 `json:"otherDeductions"`
	// PayGap تفاوت تطبیق ماهانه (ریال): مابه‌التفاوت حقوق پیشین با طرح تازه که جذب‌شدنی
	// است. صفر یعنی کارمند مشمول تفاوت تطبیق نیست.
	//
	// چرا در جمع ناخالص: تفاوت تطبیق بخشی از حقوق و مزایای ماه است، پس هم بیمه و هم
	// مالیات به آن تعلق می‌گیرد (مگر پارامتر شرکت خلافش را بگوید). ولی جزو «مزد مبنا»
	// نیست و مبنای اضافه‌کاری و فوقالعاده‌های درصدی را بالا نمی‌برد.
	PayGap int64 `json:"payGap"`
}

// PayrollResult خروجی محاسبهٔ فیش.
type PayrollResult struct {
	OvertimePay       int64 `json:"overtimePay"`
	Gross             int64 `json:"gross"`
	InsuranceBase     int64 `json:"insuranceBase"`
	InsuranceEmployee int64 `json:"insuranceEmployee"`
	InsuranceEmployer int64 `json:"insuranceEmployer"`
	TaxableIncome     int64 `json:"taxableIncome"`
	Tax               int64 `json:"tax"`
	Net               int64 `json:"net"`
}

// DefaultPayrollParams پارامترهای پیش‌فرض سامانه.
// این‌ها فقط مقادیر آغازین ماژول تنظیمات هستند و در استقرار واقعی از جدول
// پارامترها (نسخه‌دار) خوانده می‌شوند تا تغییر قانون نیازی به انتشار نسخهٔ تازه نداشته باشد.
func DefaultPayrollParams() PayrollParams {
	return PayrollParams{
		InsuranceEmployeeRate:  0.07,
		InsuranceEmployerRate:  0.23,
		InsuranceCeiling:       200_000_000,
		HousingAllowance:       9_000_000,
		ChildAllowancePerChild: 7_000_000,
		TaxExemption:           24_000_000,
		TaxBrackets: []TaxBracket{
			{Upto: 50_000_000, Rate: 0.10},
			{Upto: 100_000_000, Rate: 0.15},
			{Upto: 200_000_000, Rate: 0.20},
			{Upto: 0, Rate: 0.30},
		},
	}
}

// RoundHalfUp گرد کردن نیم‌به‌بالا (برای مبالغ ریالی، هم‌رفتار با هستهٔ Rust).
func RoundHalfUp(x float64) int64 {
	if x >= 0 {
		return int64(math.Floor(x + 0.5))
	}
	return -int64(math.Floor(-x + 0.5))
}

// round2 گرد کردن به دو رقم اعشار با رفتار نیم‌به‌بالا.
func round2(x float64) float64 {
	if x < 0 {
		return math.Trunc(x*100-0.5) / 100
	}
	return math.Trunc(x*100+0.5) / 100
}

// TaxFor مالیات پله‌ای بر درآمد مشمول.
func TaxFor(income int64, brackets []TaxBracket) int64 {
	if income <= 0 {
		return 0
	}
	var tax float64
	var prev int64
	for _, b := range brackets {
		if income <= prev {
			break
		}
		upper := b.Upto
		if upper <= 0 {
			upper = income
		}
		slice := min64(income, upper) - prev
		if slice > 0 {
			tax += float64(slice) * b.Rate
		}
		prev = upper
	}
	return RoundHalfUp(tax)
}

// ComputePayroll محاسبهٔ کامل یک فیش.
//
// تفاوت تطبیق (اگر ثبت شده باشد) به جمع ناخالص اضافه می‌شود و در نتیجه مبنای بیمه و
// مالیات هم آن را می‌بیند؛ ولی در PayrollResult جداگانه نمی‌آید چون بخشی از همان جمع است.
func ComputePayroll(input PayrollInput, params PayrollParams) PayrollResult {
	overtimePay := input.OvertimeHours * input.OvertimeHourlyRate
	childrenPay := int64(input.Children) * params.ChildAllowancePerChild
	gross := input.BaseSalary + params.HousingAllowance + childrenPay + overtimePay + nonNegative64(input.PayGap)

	insuranceBase := gross
	if params.InsuranceCeiling > 0 && gross > params.InsuranceCeiling {
		insuranceBase = params.InsuranceCeiling
	}
	insuranceEmployee := RoundHalfUp(float64(insuranceBase) * params.InsuranceEmployeeRate)
	insuranceEmployer := RoundHalfUp(float64(insuranceBase) * params.InsuranceEmployerRate)

	taxable := gross - insuranceEmployee - params.TaxExemption
	if taxable < 0 {
		taxable = 0
	}
	tax := TaxFor(taxable, params.TaxBrackets)

	return PayrollResult{
		OvertimePay:       overtimePay,
		Gross:             gross,
		InsuranceBase:     insuranceBase,
		InsuranceEmployee: insuranceEmployee,
		InsuranceEmployer: insuranceEmployer,
		TaxableIncome:     taxable,
		Tax:               tax,
		Net:               gross - insuranceEmployee - tax - input.OtherDeductions,
	}
}

// PayrollTotals جمع یک دورهٔ حقوق برای صفحهٔ «پردازش حقوق».
type PayrollTotals struct {
	Count             int
	Gross             int64
	Tax               int64
	InsuranceEmployee int64
	InsuranceEmployer int64
	Net               int64
}

// Totals جمع نتایج یک دوره.
func Totals(results []PayrollResult) PayrollTotals {
	t := PayrollTotals{Count: len(results)}
	for _, r := range results {
		t.Gross += r.Gross
		t.Tax += r.Tax
		t.InsuranceEmployee += r.InsuranceEmployee
		t.InsuranceEmployer += r.InsuranceEmployer
		t.Net += r.Net
	}
	return t
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
