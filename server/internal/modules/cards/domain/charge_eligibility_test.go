// آزمون قواعد تعلق خدمات و سقف سرها در موتور شارژ کارت‌های رفاهی.
//
// این دو قاعده در دامنه اعلام شده‌اند و اینجا سنجیده می‌شوند:
//   - افراد تحت تکفل تنها برای خدماتی شارژ می‌شوند که «تعلق» آن‌ها در فرم تکفل ثبت شده باشد؛
//   - سقف سرها (خود کارمند + افراد تحت تکفل) رعایت می‌شود و خود کارمند همیشه شارژ می‌شود.
package domain

import (
	"strings"
	"testing"
)

// petrokartInput ورودی نمونهٔ شارژ پتروکارت با افراد مشمول داده‌شده.
func petrokartInput(maxHeads int, beneficiaries ...Beneficiary) ChargeInput {
	return ChargeInput{
		Card:          CardPetrokart,
		RateBySection: map[string]int64{"petrokart_cultural": 5_000_000, "petrokart_restaurant": 8_000_000},
		Beneficiaries: beneficiaries,
		MaxHeads:      maxHeads,
		ServiceKey:    "petrokart",
		ServiceKeyFa:  "پتروکارت",
	}
}

func selfBeneficiary(employeeID, name string) Beneficiary {
	return Beneficiary{
		EmployeeID: employeeID, EmployeeName: name, Name: name, Relation: "خود کارمند", Self: true,
		Allocated: map[string]bool{"petrokartCultural": true, "petrokartRestaurant": true},
	}
}

func dependentBeneficiary(employeeID, name string, allocated bool, eligible map[string]bool, known bool) Beneficiary {
	return Beneficiary{
		EmployeeID: employeeID, EmployeeName: name, Name: name, Relation: "فرزند",
		Allocated:     map[string]bool{"petrokartCultural": allocated, "petrokartRestaurant": allocated},
		Eligible:      eligible,
		EligibleKnown: known,
	}
}

// TestChargeSkipsDependentWithoutEntitlement فرد تحت تکفل بدون تعلق پتروکارت شارژ نمی‌شود.
func TestChargeSkipsDependentWithoutEntitlement(t *testing.T) {
	in := petrokartInput(0,
		selfBeneficiary("EMP-001", "کارمند نمونه"),
		dependentBeneficiary("EMP-001", "فرزند با تعلق", true, map[string]bool{"petrokart": true}, true),
		dependentBeneficiary("EMP-001", "فرزند بدون تعلق", true, map[string]bool{"medical": true, "supplementary_insurance": true}, true),
	)
	lines, warnings := Build(in)

	charged := map[string]bool{}
	for _, line := range lines {
		charged[line.Beneficiary] = true
	}
	if !charged["کارمند نمونه"] {
		t.Error("خود کارمند باید شارژ شود")
	}
	if !charged["فرزند با تعلق"] {
		t.Error("فرد تحت تکفل با تعلق پتروکارت باید شارژ شود")
	}
	if charged["فرزند بدون تعلق"] {
		t.Error("فرد تحت تکفل بدون تعلق پتروکارت نباید شارژ شود")
	}
	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "فرزند بدون تعلق") && strings.Contains(warning, "تعلق") {
			found = true
		}
	}
	if !found {
		t.Errorf("هشدار «تعلق ثبت نشده» باید برای فرد بدون تعلق بیاید: %v", warnings)
	}
}

// TestChargeSkipsDependentWithoutEligibilityForm فردی که فرم تکفلش پیدا نشده شارژ نمی‌شود.
func TestChargeSkipsDependentWithoutEligibilityForm(t *testing.T) {
	in := petrokartInput(0,
		selfBeneficiary("EMP-001", "کارمند نمونه"),
		dependentBeneficiary("EMP-001", "فرد بدون فرم", true, nil, false),
	)
	lines, warnings := Build(in)
	for _, line := range lines {
		if line.Beneficiary == "فرد بدون فرم" {
			t.Fatal("فردی که وضعیت تعلقش ثبت نشده نباید شارژ شود")
		}
	}
	if len(lines) == 0 {
		t.Fatal("خود کارمند باید شارژ شود")
	}
	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "فرم افراد تحت تکفل") {
			found = true
		}
	}
	if !found {
		t.Errorf("هشدار نبود فرم تکفل باید بیاید: %v", warnings)
	}
}

// TestChargeHeadCap سقف سرها: خود کارمند همیشه شارژ می‌شود و افراد بیش از سقف نه.
func TestChargeHeadCap(t *testing.T) {
	in := petrokartInput(3,
		selfBeneficiary("EMP-001", "کارمند نمونه"),
		dependentBeneficiary("EMP-001", "فرزند یک", true, map[string]bool{"petrokart": true}, true),
		dependentBeneficiary("EMP-001", "فرزند دو", true, map[string]bool{"petrokart": true}, true),
		dependentBeneficiary("EMP-001", "فرزند سه", true, map[string]bool{"petrokart": true}, true),
	)
	lines, warnings := Build(in)

	charged := map[string]bool{}
	for _, line := range lines {
		charged[line.Beneficiary] = true
	}
	if !charged["کارمند نمونه"] {
		t.Error("خود کارمند با هر سقفی باید شارژ شود")
	}
	chargedDependents := 0
	for _, name := range []string{"فرزند یک", "فرزند دو", "فرزند سه"} {
		if charged[name] {
			chargedDependents++
		}
	}
	if chargedDependents != 2 {
		t.Errorf("با سقف ۳ سر، دو فرد تحت تکفل باید شارژ شوند، %d شد", chargedDependents)
	}
	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, "سقف سرهای شارژ") && strings.Contains(warning, "۳") {
			found = true
		}
	}
	if !found {
		t.Errorf("هشدار سقف باید با عدد فارسی بیاید: %v", warnings)
	}
}

// TestChargeWithoutCap بی‌سقفی (صفر) همهٔ مشمولان با تعلق را شارژ می‌کند.
func TestChargeWithoutCap(t *testing.T) {
	in := petrokartInput(0,
		selfBeneficiary("EMP-001", "کارمند نمونه"),
		dependentBeneficiary("EMP-001", "فرزند یک", true, map[string]bool{"petrokart": true}, true),
		dependentBeneficiary("EMP-001", "فرزند دو", true, map[string]bool{"petrokart": true}, true),
		dependentBeneficiary("EMP-001", "فرزند سه", true, map[string]bool{"petrokart": true}, true),
	)
	lines, _ := Build(in)
	names := map[string]bool{}
	for _, line := range lines {
		names[line.Beneficiary] = true
	}
	if len(names) != 4 {
		t.Errorf("بی‌سقفی باید همهٔ چهار نفر را شارژ کند: %v", names)
	}
}
