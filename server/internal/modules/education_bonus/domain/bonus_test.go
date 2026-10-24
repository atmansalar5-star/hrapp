// آزمون دامنهٔ ماژول «education_bonus» (bonus_test).

package domain

import (
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// TestLevelsCatalog بررسی فهرست اعلامی مقاطع و مبالغ پیش‌فرض.
func TestLevelsCatalog(t *testing.T) {
	levels := Levels()
	if len(levels) != 6 {
		t.Fatalf("تعداد مقاطع باید ۶ باشد، %d بود", len(levels))
	}
	for _, l := range levels {
		if l.DefaultRate <= 0 {
			t.Fatalf("مبلغ پیش‌فرض مقطع «%s» باید مثبت باشد", l.Key)
		}
	}
	if _, ok := LevelByKey("high2"); !ok {
		t.Fatal("مقطع متوسطه دوم باید در فهرست باشد")
	}
	if LevelTitleFa("elementary") != "ابتدایی" {
		t.Fatalf("عنوان مقطع ابتدایی درست نیست: %q", LevelTitleFa("elementary"))
	}
}

// TestBuildPaysOnlyExcellent دانش‌آموز غیرممتاز نباید ردیف بگیرد و باید هشدار بدهد.
func TestBuildPaysOnlyExcellent(t *testing.T) {
	students := []Student{
		{EmployeeID: "EMP-005", EmployeeName: "محمد احمدی", Name: "علی احمدی", Relation: "فرزند", Level: "high2", Excellent: true, Eligible: true},
		{EmployeeID: "EMP-005", EmployeeName: "محمد احمدی", Name: "زهرا احمدی", Relation: "فرزند", Level: "elementary", Excellent: false, Eligible: true},
	}
	lines, warnings := Build(BonusInput{Rates: DefaultRates(), Students: students, Period: "1404/1405"})
	if len(lines) != 1 {
		t.Fatalf("فقط دانش‌آموز ممتاز باید ردیف بگیرد؛ %d ردیف ساخته شد", len(lines))
	}
	if lines[0].Student != "علی احمدی" {
		t.Fatalf("ردیف ساخته‌شده برای دانش‌آموز درست نیست: %q", lines[0].Student)
	}
	if len(warnings) == 0 {
		t.Fatal("برای دانش‌آموز غیرممتاز باید هشدار داده شود")
	}
}

// TestBuildSkipsIneligible دانش‌آموز غیرمشمول بی‌صدا نادیده گرفته می‌شود.
func TestBuildSkipsIneligible(t *testing.T) {
	students := []Student{
		{EmployeeID: "EMP-005", EmployeeName: "م", Name: "زهرا", Level: "elementary", Excellent: true, Eligible: false},
	}
	lines, _ := Build(BonusInput{Rates: DefaultRates(), Students: students})
	if len(lines) != 0 {
		t.Fatalf("دانش‌آموز غیرمشمول نباید ردیف بگیرد؛ %d ردیف ساخته شد", len(lines))
	}
}

// TestBuildRateFallback نبود مبلغ مصوب مقطع به مبلغ پیش‌فرض برمی‌گردد و هشدار می‌دهد.
func TestBuildRateFallback(t *testing.T) {
	students := []Student{
		{EmployeeID: "EMP-010", EmployeeName: "ف", Name: "مهدی", Level: "elementary", Excellent: true, Eligible: true},
	}
	lines, warnings := Build(BonusInput{Rates: map[string]int64{}, Students: students})
	if len(lines) != 1 {
		t.Fatalf("با نبود مبلغ مصوب باید مبلغ پیش‌فرض به‌کار رود؛ %d ردیف", len(lines))
	}
	want, _ := LevelByKey("elementary")
	if lines[0].Amount != want.DefaultRate {
		t.Fatalf("مبلغ ردیف باید %d باشد، %d بود", want.DefaultRate, lines[0].Amount)
	}
	if len(warnings) == 0 {
		t.Fatal("برای مبلغ پیش‌فرض باید هشدار داده شود")
	}
}

// TestBuildLevelFilter فقط مقطع‌های خواسته‌شده ساخته می‌شوند.
func TestBuildLevelFilter(t *testing.T) {
	students := []Student{
		{EmployeeID: "A", EmployeeName: "الف", Name: "یک", Level: "elementary", Excellent: true, Eligible: true},
		{EmployeeID: "B", EmployeeName: "ب", Name: "دو", Level: "high2", Excellent: true, Eligible: true},
	}
	lines, _ := Build(BonusInput{Levels: []string{"high2"}, Rates: DefaultRates(), Students: students})
	if len(lines) != 1 || lines[0].Level != "high2" {
		t.Fatalf("با فیلتر مقطع باید فقط یک ردیف متوسطه دوم باشد؛ %d ردیف", len(lines))
	}
}

// TestSumLines جمع کل و جمع هر مقطع.
func TestSumLines(t *testing.T) {
	lines := []BonusLine{
		{Level: "elementary", Amount: 8_000_000},
		{Level: "elementary", Amount: 8_000_000},
		{Level: "high2", Amount: 15_000_000},
	}
	total, byLevel, count := SumLines(lines)
	if total != 31_000_000 {
		t.Fatalf("جمع کل درست نیست: %d", total)
	}
	if byLevel["elementary"] != 16_000_000 || count["elementary"] != 2 {
		t.Fatalf("جمع مقطع ابتدایی درست نیست: %d (%d ردیف)", byLevel["elementary"], count["elementary"])
	}
}

// TestValidateStudentRule قاعدهٔ دامنه: مشمول بودن بدون نشانهٔ ممتاز مجاز نیست.
func TestValidateStudentRule(t *testing.T) {
	err := validateStudent(contracts.RecordInput{Fields: map[string]string{"eligible": "1", "excellent": "0"}})
	if err == nil {
		t.Fatal("مشمول بدون ممتاز باید خطای قاعده بدهد")
	}
	if _, ok := err.(*recordkit.RuleError); !ok {
		t.Fatalf("خطا باید از نوع RuleError باشد، %T بود", err)
	}
	if err := validateStudent(contracts.RecordInput{Fields: map[string]string{"eligible": "1", "excellent": "1"}}); err != nil {
		t.Fatalf("مشمول ممتاز نباید خطا بدهد: %v", err)
	}
}

// TestValidateRateRule مقطع ناشناخته باید خطا بدهد.
func TestValidateRateRule(t *testing.T) {
	if err := validateRate(contracts.RecordInput{Fields: map[string]string{"level": "nope"}}); err == nil {
		t.Fatal("مقطع ناشناخته باید خطا بدهد")
	}
	if err := validateRate(contracts.RecordInput{Fields: map[string]string{"level": "high2"}}); err != nil {
		t.Fatalf("مقطع معتبر نباید خطا بدهد: %v", err)
	}
}

// TestDescriptorAndSeed بررسی تعداد نوع‌های رکورد و ناتهی بودن بذرکاری.
func TestDescriptorAndSeed(t *testing.T) {
	d := Descriptor()
	if d.Module != "education_bonus" {
		t.Fatalf("شناسهٔ ماژول درست نیست: %q", d.Module)
	}
	if len(d.Kinds) != 3 {
		t.Fatalf("تعداد نوع‌های رکورد باید ۳ باشد، %d بود", len(d.Kinds))
	}
	if len(Seed()) == 0 {
		t.Fatal("بذرکاری نباید خالی باشد")
	}
}
