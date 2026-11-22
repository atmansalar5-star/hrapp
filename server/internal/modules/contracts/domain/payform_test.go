// آزمون زیرفرم‌های «حقوق و مزایا» بر پایهٔ نوع استخدام.
// ارجاع: docs/contract-pay-form-spec-fa.md (بندهای CPF-01 … CPF-04).

package domain

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// TestCPF01PayFamilyMapping می‌سنجد خانوادهٔ پرداخت از نوع استخدام درست استخراج شود
// (کلید لاتین و عنوان فارسی) و نوع ناشناخته محافظه‌کارانه «رسمی» شود.
func TestCPF01PayFamilyMapping(t *testing.T) {
	cases := map[string]string{
		"official_oil":       PayFamilyOfficial,
		"کارمند رسمی نفت":    PayFamilyOfficial,
		"fixed_1662":         PayFamilyOfficial,
		"contractor_staff":   PayFamilyContracting,
		"contractor_project": PayFamilyContracting,
		"پیمانکار ستادی":     PayFamilyContracting,
		"seconded_to":        PayFamilySecondment,
		"seconded_from":      PayFamilySecondment,
		"":                   PayFamilyOfficial,
		"چیز-ناشناخته":       PayFamilyOfficial,
	}
	for input, want := range cases {
		if got := PayFamilyOf(input); got != want {
			t.Fatalf("PayFamilyOf(%q) = %q، انتظار %q", input, got, want)
		}
	}
}

// TestCPF02PayFormPerFamily می‌سنجد هر خانواده زیرفرم مخصوص خودش را دارد: رسمی با جدول
// مزد، پیمانکاری با مشخصات پیمان و مبنای پرداخت، مأموریت با ترتیب پرداخت.
func TestCPF02PayFormPerFamily(t *testing.T) {
	official := PayFormOf("official_oil", KindPermanent)
	if !official.WageTable {
		t.Fatal("زیرفرم رسمی باید جدول مزد رسمی را داشته باشد")
	}
	if !formHasField(official, "rankWage") || !formHasField(official, "climateRegion") {
		t.Fatal("زیرفرم رسمی باید اجزای مزد و منطقهٔ اقلیمی را داشته باشد")
	}
	if formHasField(official, "payBasis") {
		t.Fatal("زیرفرم رسمی نباید فیلد مبنای پیمان داشته باشد")
	}

	contracting := PayFormOf("contractor_staff", KindFixedTerm)
	if contracting.WageTable {
		t.Fatal("زیرفرم پیمانکاری نباید جدول مزد رسمی را نشان دهد")
	}
	for _, key := range []string{"contractorName", "contractNo", "payBasis", "payAmount", "insurancePayer"} {
		field, ok := findField(contracting, key)
		if !ok {
			t.Fatalf("زیرفرم پیمانکاری باید فیلد «%s» را داشته باشد", key)
		}
		if !field.Required {
			t.Fatalf("فیلد «%s» در پیمانکاری الزامی است", key)
		}
	}

	secondment := PayFormOf("seconded_to", KindFixedTerm)
	if secondment.WageTable {
		t.Fatal("سند مأموریت نباید جدول مزد رسمی را نشان دهد")
	}
	for _, key := range []string{"hostCompany", "missionStart", "salaryPayer"} {
		if _, ok := findField(secondment, key); !ok {
			t.Fatalf("زیرفرم مأموریت باید فیلد «%s» را داشته باشد", key)
		}
	}
	share, ok := findField(secondment, "originSharePercent")
	if !ok || share.RequiredWhen == nil {
		t.Fatal("سهم شرکت مبدأ باید شرطی (وابسته به ترتیب پرداخت) باشد")
	}

	// همهٔ زیرفرم‌ها یک‌بار از API هم قابل دریافت‌اند و هر سه خانواده را دارند.
	families := map[string]bool{}
	for _, form := range PayFormsOf() {
		families[form.Family] = true
		if strings.TrimSpace(form.TitleFa) == "" || len(form.Groups) == 0 {
			t.Fatalf("زیرفرم خانوادهٔ %s ناقص است", form.Family)
		}
	}
	for _, family := range []string{PayFamilyOfficial, PayFamilyContracting, PayFamilySecondment} {
		if !families[family] {
			t.Fatalf("خانوادهٔ %s در فهرست زیرفرم‌ها نیست", family)
		}
	}
}

// TestCPF03PayFormValidation می‌سنجد الزامی‌ها، قیدهای شرطی و ممنوعیت‌ها اعمال شوند.
func TestCPF03PayFormValidation(t *testing.T) {
	official := PayInput{
		Kind: KindPermanent, EmploymentType: "official_oil", MonthlySalary: 40_000_000,
		Wage: contracts.WageStructure{ClimateRegion: "south_coast", RankWage: 9_000_000},
	}
	if errs := ValidatePayForm(official); len(errs) != 0 {
		t.Fatalf("قرارداد رسمی معتبر باید بی‌خطا باشد: %+v", errs)
	}

	// پیمانکاری بدون مشخصات پیمان: خطا روی همان فیلدها.
	contracting := PayInput{Kind: KindFixedTerm, EmploymentType: "contractor_staff", MonthlySalary: 25_000_000}
	errs := ValidatePayForm(contracting)
	for _, key := range []string{"contractorName", "contractNo", "payBasis", "payAmount", "insurancePayer"} {
		if !hasError(errs, key) {
			t.Fatalf("فیلد «%s» باید خطا بدهد: %+v", key, errs)
		}
	}

	// پیمانکاری با اجزای مزد رسمی: رد می‌شود (فراخوان مستقیم API هم نباید دور بزند).
	contracting.Wage = contracts.WageStructure{RankWage: 5_000_000}
	contracting.Meta = map[string]any{
		"contractorName": "شرکت پیمانکار نمونه", "contractNo": "P-1405-01",
		"payBasis": "monthly", "payAmount": int64(25_000_000), "insurancePayer": "contractor",
	}
	if errs := ValidatePayForm(contracting); !hasError(errs, "wageStructure") {
		t.Fatalf("اجزای مزد رسمی در پیمانکاری باید رد شود: %+v", errs)
	}
	contracting.Wage = contracts.WageStructure{}
	if errs := ValidatePayForm(contracting); len(errs) != 0 {
		t.Fatalf("پیمانکاری کامل باید بی‌خطا باشد: %+v", errs)
	}

	// مأموریت: ترتیب پرداخت الزامی، و سهم مبدأ با پرداخت تقسیمی اجباری.
	secondment := PayInput{
		Kind: KindFixedTerm, EmploymentType: "seconded_to", MonthlySalary: 30_000_000,
		Meta: map[string]any{"hostCompany": "شرکت مقصد", "missionStart": "1405/01/15", "salaryPayer": "shared"},
	}
	if errs := ValidatePayForm(secondment); !hasError(errs, "originSharePercent") {
		t.Fatalf("با پرداخت تقسیمی، سهم شرکت مبدأ الزامی است: %+v", errs)
	}
	secondment.Meta["originSharePercent"] = 0.4
	if errs := ValidatePayForm(secondment); len(errs) != 0 {
		t.Fatalf("مأموریت کامل باید بی‌خطا باشد: %+v", errs)
	}

	// بدون نوع استخدام (استنتاج از نوع قرارداد) ⇒ رسمی؛ مبلغ صفر خطا می‌دهد.
	empty := PayInput{Kind: KindPermanent}
	if errs := ValidatePayForm(empty); !hasError(errs, "monthlySalary") {
		t.Fatalf("مبلغ پایهٔ صفر باید خطا بدهد: %+v", errs)
	}
}

// TestCPF04PayFormBounds می‌سنجد مقدارهای خارج از دامنه (درصد بیش از ۱۰۰، گزینهٔ
// ناشناخته در فهرست انتخابی) رد شوند.
func TestCPF04PayFormBounds(t *testing.T) {
	in := PayInput{
		Kind: KindFixedTerm, EmploymentType: "seconded_to", MonthlySalary: 30_000_000,
		Meta: map[string]any{
			"hostCompany": "شرکت مقصد", "missionStart": "1405/01/15", "salaryPayer": "شرکت-ناشناخته",
			"originSharePercent": 1.5,
		},
	}
	errs := ValidatePayForm(in)
	if !hasError(errs, "salaryPayer") {
		t.Fatalf("گزینهٔ ناشناختهٔ ترتیب پرداخت باید رد شود: %+v", errs)
	}
	if !hasError(errs, "originSharePercent") {
		t.Fatalf("درصد بیش از ۱۰۰ باید رد شود: %+v", errs)
	}
}

func formHasField(form PayForm, key string) bool {
	_, ok := findField(form, key)
	return ok
}

func findField(form PayForm, key string) (PayField, bool) {
	for _, group := range form.Groups {
		for _, field := range group.Fields {
			if field.Key == key {
				return field, true
			}
		}
	}
	return PayField{}, false
}

func hasError(errs []PayFieldError, key string) bool {
	for _, e := range errs {
		if e.Field == key {
			return true
		}
	}
	return false
}
