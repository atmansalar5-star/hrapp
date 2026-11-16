// آزمون قاعدهٔ «oilpay_veteran» در موتور قواعد. آزمون خودکار؛ هر تغییر رفتار پیاده‌سازی باید آزمون
// همین پرونده را به‌روز کند.

package rules

import (
	"strings"
	"testing"
)

// oilVeteranInput ورودی فیش صندوق نفت برای کارمند ایثارگر.
func oilVeteranInput(ent *VeteranEntitlement) OilPayInput {
	return OilPayInput{
		EmployeeID: "EMP-009", BaseSalary: 120_000_000, JobGrade: "A", Veteran: ent,
	}
}

// entitlementFor امتیاز تأییدشدهٔ یک گروه را با پارامترهای دلخواه می‌سازد.
func entitlementFor(group string, percent float64, captivity int, params VeteranParams) VeteranEntitlement {
	return ComputeVeteranEntitlements(VeteranProfile{
		Group: group, DisabilityPercent: percent, CaptivityMonths: captivity, ApprovedFile: true,
	}, params)
}

// TestOilPayslipBandsStayConsistentWithVeteranRelief بندهای فیش رسمی باید با جمع‌های فیش
// هم‌خوان بمانند: امتیاز ایثارگری نباید سطر منفی بسازد یا خالص را از جمع ناخالص بیشتر کند.
func TestOilPayslipBandsStayConsistentWithVeteranRelief(t *testing.T) {
	params := DefaultOilPayParams()
	ent := entitlementFor(VeteranGroupDisabled, 25, 0, DefaultVeteranParams())
	slip := ComputeOilPayslip(oilVeteranInput(&ent), params)

	if !slip.VeteranApplied || slip.VeteranGroupFa != "جانباز" {
		t.Fatalf("امتیاز ایثارگری باید در فیش اعمال شود: %+v", slip)
	}
	if slip.Tax != 0 || slip.VeteranTaxExempt <= 0 {
		t.Fatalf("معافیت مالیاتی مادهٔ ۵۶ باید مالیات را صفر کند: مالیات=%d معافیت=%d",
			slip.Tax, slip.VeteranTaxExempt)
	}
	if slip.Net > slip.Gross {
		t.Fatalf("خالص پرداختی نباید از جمع ناخالص بیشتر شود: خالص=%d ناخالص=%d", slip.Net, slip.Gross)
	}
	if slip.Deductions != slip.FundEmployee+slip.InsuranceEmployee+slip.Tax+slip.OtherDeductions {
		t.Fatalf("جمع کسور با اجزایش نمی‌خواند: %+v", slip)
	}
	if want := slip.Gross - slip.Deductions; slip.Net != want {
		t.Fatalf("خالص پرداختی باید ناخالص منهای کسور باشد: %d در برابر %d", slip.Net, want)
	}
	var bandNet, credit, debit int64
	for _, band := range slip.Bands {
		bandNet += band.Net
		credit += band.Credit
		debit += band.Debit
	}
	if bandNet != slip.Net {
		t.Fatalf("جمع خالص بندها با خالص فیش نمی‌خواند: %d در برابر %d", bandNet, slip.Net)
	}
	if credit-debit != slip.Net {
		t.Fatalf("جمع بستانکار و بدهکار بندها با خالص فیش نمی‌خواند: %d در برابر %d", credit-debit, slip.Net)
	}
	for _, line := range slip.Lines {
		if line.Amount < 0 {
			t.Fatalf("سطرهای فیش نباید مبلغ منفی داشته باشند (خطای بندها): %+v", line)
		}
	}
	// شرح سطر مالیات باید بگوید چرا مالیات صفر است؛ سطر اعتباری جدا ساخته نمی‌شود.
	found := false
	for _, line := range slip.Lines {
		if line.Key == "tax" {
			found = strings.Contains(line.DetailFa, "ایثارگری")
		}
		if line.Key == "veteranTaxExempt" || line.Key == "veteranPensionExempt" {
			t.Fatalf("سطر اعتباری امتیاز نباید در فیش ساخته شود: %+v", line)
		}
	}
	if slip.Tax == 0 && !found && slip.TaxableIncome == 0 {
		t.Log("مالیات صفر است و سطر مالیات ساخته نشده؛ شرح معافیت در یادداشت فیش آمده است")
	}
	if !strings.Contains(slip.NoteFa, "امتیاز ایثارگری") {
		t.Fatalf("یادداشت فیش باید اعمال امتیاز را بگوید: %q", slip.NoteFa)
	}
}

// TestOilPayslipWithoutVeteranStaysUnchanged فیش کارمند بدون پروندهٔ تأییدشده نباید هیچ
// سطر یا جمع تازه‌ای بگیرد.
func TestOilPayslipWithoutVeteranStaysUnchanged(t *testing.T) {
	params := DefaultOilPayParams()
	plain := ComputeOilPayslip(oilVeteranInput(nil), params)
	if plain.VeteranApplied || plain.VeteranTaxExempt != 0 || plain.VeteranPensionExempt != 0 {
		t.Fatalf("بدون پرونده نباید امتیازی اعمال شود: %+v", plain)
	}
	if plain.VeteranGroupFa != "" {
		t.Fatalf("بدون پرونده نباید گروه ایثارگری در فیش بیاید: %q", plain.VeteranGroupFa)
	}
	if strings.Contains(plain.NoteFa, "امتیاز ایثارگری") {
		t.Fatalf("یادداشت فیش بدون پرونده نباید امتیاز را بگوید: %q", plain.NoteFa)
	}
	// پروندهٔ تأییدنشده هم هیچ اثری ندارد.
	draft := entitlementFor(VeteranGroupDisabled, 25, 0, DefaultVeteranParams())
	draft.Approved = false
	unapproved := ComputeOilPayslip(oilVeteranInput(&draft), params)
	if unapproved.Tax != plain.Tax || unapproved.Net != plain.Net {
		t.Fatalf("پروندهٔ تأییدنشده نباید فیش را تغییر دهد: %+v", unapproved)
	}
}

// TestOilPayslipPensionExemptFollowsCompanyParam کاهش کسور سهم کارمند با مصوبهٔ شرکت، خودِ
// سطر صندوق را کم می‌کند (نه سطر اعتباری جدا) و بدهکار بند هم با همان عدد می‌خواند.
func TestOilPayslipPensionExemptFollowsCompanyParam(t *testing.T) {
	params := DefaultOilPayParams()
	veteranParams := DefaultVeteranParams()
	veteranParams.PensionExemptPercent = 50
	ent := entitlementFor(VeteranGroupFreed, 0, 24, veteranParams)

	plain := ComputeOilPayslip(oilVeteranInput(nil), params)
	slip := ComputeOilPayslip(oilVeteranInput(&ent), params)

	if slip.VeteranPensionExempt <= 0 {
		t.Fatalf("معافیت کسور باید اعمال شود: %+v", slip)
	}
	wantFund := RoundHalfUp(float64(slip.FundEmployee+slip.VeteranPensionExempt)) -
		RoundHalfUp(float64(slip.FundEmployee+slip.VeteranPensionExempt)*50/100)
	if slip.FundEmployee > wantFund {
		t.Fatalf("کسور صندوق پس از معافیت باید کمتر شود: %d (انتظار ≈%d)", slip.FundEmployee, wantFund)
	}
	if slip.FundEmployee >= plain.FundEmployee {
		t.Fatalf("کسور صندوق با معافیت باید کمتر از حالت بدون امتیاز باشد: %d در برابر %d",
			slip.FundEmployee, plain.FundEmployee)
	}
	for _, line := range slip.Lines {
		if line.Key == "fundEmployee" {
			if line.Amount != slip.FundEmployee {
				t.Fatalf("سطر کسور صندوق باید مبلغ پس از معافیت را نشان دهد: %d در برابر %d",
					line.Amount, slip.FundEmployee)
			}
			if !strings.Contains(line.DetailFa, "معافیت کسور ایثارگری") {
				t.Fatalf("شرح سطر کسور صندوق باید معافیت ایثارگری را بگوید: %q", line.DetailFa)
			}
		}
	}
	if slip.Net <= plain.Net {
		t.Fatalf("خالص پرداختی ایثارگر باید بیشتر از حالت بدون امتیاز باشد: %d در برابر %d",
			slip.Net, plain.Net)
	}
}
