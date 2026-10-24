// آزمون دامنهٔ ماژول «cards» (charge_test).

package domain

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// recordInput ورودی رکورد ساختگی برای سنجش قواعد دامنه.
func recordInput(fields map[string]string) contracts.RecordInput {
	return contracts.RecordInput{Kind: "cardRates", Fields: fields}
}

func allAllocated() map[string]bool {
	return map[string]bool{
		"petrokartCultural": true, "petrokartRestaurant": true,
		"educationBonus": true, "naftmart": true,
	}
}

func person(id, name, relation string, excellent bool) Beneficiary {
	return Beneficiary{
		EmployeeID: id, EmployeeName: "کارمند " + id, Name: name,
		Relation: relation, Excellent: excellent, Allocated: allAllocated(),
	}
}

// TestCatalogCoversBothCards: پتروکارت سه بخش (فرهنگی، رستوران، پاداش تحصیلی) و
// نفت‌کارت چهار قلم رفاهی دارد.
func TestCatalogCoversBothCards(t *testing.T) {
	petro := SectionsOf(CardPetrokart)
	if len(petro) != 3 {
		t.Fatalf("پتروکارت باید ۳ بخش داشته باشد، %d داشت", len(petro))
	}
	naft := SectionsOf(CardNaftmart)
	if len(naft) != 4 {
		t.Fatalf("نفت‌کارت باید چهار قلم رفاهی داشته باشد، %d داشت", len(naft))
	}
	// هر بخش باید نرخ پیش‌فرض مثبت و کلید تخصیص داشته باشد.
	for _, s := range Sections() {
		if s.DefaultRate <= 0 {
			t.Fatalf("نرخ پیش‌فرض «%s» باید مثبت باشد", s.Key)
		}
		if AllocationField(s.Key) == "" {
			t.Fatalf("بخش «%s» کلید تخصیص ندارد", s.Key)
		}
	}
}

// TestBuildChargesPerDependent: مبلغ ماهیانه به ازای هر فرد مشمول ساخته می‌شود؛
// دو نفر مشمول یعنی دو برابر.
func TestBuildChargesPerDependent(t *testing.T) {
	in := ChargeInput{
		Card:          CardPetrokart,
		Sections:      []string{"petrokart_cultural"},
		RateBySection: map[string]int64{"petrokart_cultural": 5_000_000},
		Beneficiaries: []Beneficiary{
			person("EMP-001", "خود کارمند", "self", false),
			person("EMP-001", "فرزند اول", "فرزند", false),
		},
		Period: "1404/07",
	}
	lines, warnings := Build(in)
	if len(lines) != 2 {
		t.Fatalf("باید ۲ ردیف ساخته شود، %d ساخته شد (%v)", len(lines), warnings)
	}
	for _, l := range lines {
		if l.Amount != 5_000_000 {
			t.Fatalf("مبلغ هر ردیف باید ۵٬۰۰۰٬۰۰۰ باشد، %d بود", l.Amount)
		}
	}
	if total, _, _ := SumLines(lines); total != 10_000_000 {
		t.Fatalf("جمع باید ۱۰٬۰۰۰٬۰۰۰ باشد، %d بود", total)
	}
}

// TestBuildRespectsAllocation: بخشی که به فرد تخصیص داده نشده شارژ نمی‌شود.
func TestBuildRespectsAllocation(t *testing.T) {
	b := person("EMP-002", "فرزند", "فرزند", false)
	b.Allocated["petrokartCultural"] = false // عدم تخصیص
	lines, _ := Build(ChargeInput{
		Card:          CardPetrokart,
		Sections:      []string{"petrokart_cultural", "petrokart_restaurant"},
		RateBySection: map[string]int64{"petrokart_cultural": 1, "petrokart_restaurant": 1},
		Beneficiaries: []Beneficiary{b},
	})
	if len(lines) != 1 || lines[0].Section != "petrokart_restaurant" {
		t.Fatalf("بخش فرهنگی تخصیص‌نداده نباید شارژ شود: %+v", lines)
	}
}

// TestEducationBonusRequiresExcellent: پاداش تحصیلی فقط برای فرزند ممتاز.
func TestEducationBonusRequiresExcellent(t *testing.T) {
	in := ChargeInput{
		Card:          CardPetrokart,
		Sections:      []string{EducationSectionKey},
		RateBySection: map[string]int64{EducationSectionKey: 12_000_000},
		Beneficiaries: []Beneficiary{person("EMP-003", "فرزند عادی", "فرزند", false)},
	}
	lines, warnings := Build(in)
	if len(lines) != 0 {
		t.Fatalf("برای فرزند غیرممتاز نباید پاداش تحصیلی ساخته شود: %+v", lines)
	}
	if len(warnings) == 0 || !strings.Contains(strings.Join(warnings, " "), "ممتاز") {
		t.Fatalf("باید هشدار ممتاز نبودن داده شود: %v", warnings)
	}

	in.Beneficiaries = []Beneficiary{person("EMP-003", "فرزند ممتاز", "فرزند", true)}
	lines, _ = Build(in)
	if len(lines) != 1 || lines[0].Section != EducationSectionKey {
		t.Fatalf("برای فرزند ممتاز باید پاداش تحصیلی ساخته شود: %+v", lines)
	}
}

// TestOccasionAddsRowPerPerson: مناسبت تقویمی به‌ازای هر فرد مشمول یک ردیف می‌سازد.
func TestOccasionAddsRowPerPerson(t *testing.T) {
	lines, _ := Build(ChargeInput{
		Card:          CardNaftmart,
		Sections:      []string{},
		RateBySection: map[string]int64{},
		Beneficiaries: []Beneficiary{
			person("EMP-004", "کارمند", "self", false),
			person("EMP-004", "همسر", "همسر", false),
		},
		Occasion: &OccasionCharge{ID: "OCC-0001", TitleFa: "نوروز", DateFa: "1405/01/01", Amount: 4_000_000, Card: CardNaftmart},
	})
	occasionRows := 0
	for _, l := range lines {
		if l.Section == "occasion" {
			occasionRows++
			if l.OccasionFa != "نوروز" {
				t.Fatalf("نام مناسبت ردیف درست نیست: %q", l.OccasionFa)
			}
		}
	}
	if occasionRows != 2 {
		t.Fatalf("مناسبت باید برای ۲ نفر ۲ ردیف بسازد، %d ساخت", occasionRows)
	}
}

// TestSectionOfAnotherCardIsRejected: بخش کارت دیگر شارژ نمی‌شود و هشدار می‌دهد.
func TestSectionOfAnotherCardIsRejected(t *testing.T) {
	lines, warnings := Build(ChargeInput{
		Card:          CardNaftmart,
		Sections:      []string{"petrokart_cultural", "naftmart_dairy"},
		RateBySection: DefaultRates(),
		Beneficiaries: []Beneficiary{person("EMP-005", "کارمند", "self", false)},
	})
	if len(lines) != 1 || lines[0].Section != "naftmart_dairy" {
		t.Fatalf("فقط بخش همان کارت باید ساخته شود: %+v", lines)
	}
	if !strings.Contains(strings.Join(warnings, " "), "تعلق ندارد") {
		t.Fatalf("انتظار هشدار ناسازگاری کارت بود: %v", warnings)
	}
}

// TestMissingRateFallsBackToDefault: نرخ تعریف‌نشده به نرخ پیش‌فرض دامنه برمی‌گردد.
func TestMissingRateFallsBackToDefault(t *testing.T) {
	lines, warnings := Build(ChargeInput{
		Card:          CardPetrokart,
		Sections:      []string{"petrokart_restaurant"},
		RateBySection: map[string]int64{},
		Beneficiaries: []Beneficiary{person("EMP-006", "کارمند", "self", false)},
	})
	if len(lines) != 1 {
		t.Fatalf("ردیف ساخته نشد: %v", warnings)
	}
	sec, _ := SectionByKey("petrokart_restaurant")
	if lines[0].Amount != sec.DefaultRate {
		t.Fatalf("باید نرخ پیش‌فرض %d به‌کار می‌رفت، %d شد", sec.DefaultRate, lines[0].Amount)
	}
}

// TestBeneficiaryRuleBlocksBonusWithoutExcellence: قاعدهٔ دامنه در لایهٔ داده هم اجرا می‌شود.
func TestBeneficiaryRuleBlocksBonusWithoutExcellence(t *testing.T) {
	err := validateBeneficiary(recordInput(map[string]string{"educationBonus": "1", "excellent": "0"}))
	if err == nil {
		t.Fatal("تخصیص پاداش تحصیلی بدون ممتاز بودن باید رد شود")
	}
	if err := validateBeneficiary(recordInput(map[string]string{"educationBonus": "1", "excellent": "1"})); err != nil {
		t.Fatalf("فرزند ممتاز باید مجاز باشد: %v", err)
	}
}

// TestRateRuleRejectsForeignSection: نرخ فقط برای بخش‌های همان کارت.
func TestRateRuleRejectsForeignSection(t *testing.T) {
	if err := validateRate(recordInput(map[string]string{"card": CardNaftmart, "section": "petrokart_cultural"})); err == nil {
		t.Fatal("بخش پتروکارت روی نفت‌کارت باید رد شود")
	}
	if err := validateRate(recordInput(map[string]string{"card": CardNaftmart, "section": "naftmart_food"})); err != nil {
		t.Fatalf("بخش درست باید پذیرفته شود: %v", err)
	}
}
