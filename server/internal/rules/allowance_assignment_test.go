// آزمون موتور تخصیص فوق العاده‌های حقوق (کد رسمی، روش محاسبه، دامنه و مشمولیت کسور).
// ارجاع: docs/allowance-assignment-spec-fa.md (بندهای ASG-01 … ASG-04).

package rules

import (
	"strings"
	"testing"
)

// TestASG01PayslipCodeCatalog می‌سنجد جدول رسمی کدهای فوق العاده کامل، یکتا و درست نگاشت
// شده باشد و شکاف‌های پیاده‌سازی صریح گزارش شوند.
func TestASG01PayslipCodeCatalog(t *testing.T) {
	codes := PayslipCodes()
	if len(codes) < 28 {
		t.Fatalf("جدول رسمی باید دست‌کم ۲۸ ردیف داشته باشد، %d آمد", len(codes))
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if strings.TrimSpace(c.Code) == "" || strings.TrimSpace(c.TitleFa) == "" {
			t.Fatalf("هر ردیف باید کد و عنوان داشته باشد: %+v", c)
		}
		if seen[c.Code] {
			t.Fatalf("کد تکراری در جدول: %s", c.Code)
		}
		seen[c.Code] = true
		switch c.Kind {
		case PayslipKindEarning, PayslipKindDeduction, PayslipKindLoan:
		default:
			t.Fatalf("نوع قلم نامعتبر برای کد %s: %s", c.Code, c.Kind)
		}
	}

	// نمونه‌های شاخص جدول تصویر.
	wants := map[string]string{
		"101": "حقوق پایه", "102": "بدی آب و هوا", "104": "محرومیت از تسهیلات زندگی",
		"127": "جبران صرف وقت اضافی", "144": "درصد فوق العاده شغل", "151": "مسکن",
		"181": "حق اولاد", "159": "عائله مندی", "557": "مالیات بر درآمد",
		"551": "بازنشستگی سهم کارمند", "5400": "کسر اقساط وام اضطراری",
	}
	for code, title := range wants {
		got := PayslipCodeTitleFa(code)
		if got != title {
			t.Fatalf("عنوان کد %s باید «%s» باشد، «%s» آمد", code, title, got)
		}
	}

	// کد با ارقام فارسی و صفر ابتدایی هم پیدا می‌شود (ورودی فرم).
	if got := PayslipCodeTitleFa("۰۵۴۰۰"); got != "کسر اقساط وام اضطراری" {
		t.Fatalf("کد فارسی/با صفر ابتدایی باید پیدا شود: %s", got)
	}

	// نگاشت کلید قلم سامانه به کد رسمی.
	if code := PayslipCodeNumberOf("deprivation"); code != "104" {
		t.Fatalf("محرومیت باید کد ۱۰۴ بگیرد، %s آمد", code)
	}
	if code := PayslipCodeNumberOf("tax"); code != "557" {
		t.Fatalf("مالیات باید کد ۵۵۷ بگیرد، %s آمد", code)
	}
	if code := PayslipCodeNumberOf("shiftWorker"); code != "" {
		t.Fatalf("نوبتکاری در جدول رسمی کد ندارد، %s آمد", code)
	}
	if gaps := PayslipCodeGaps(); len(gaps) == 0 {
		t.Fatal("شکاف‌های کد باید صریح گزارش شوند")
	}
}

// TestASG02TargetMatching می‌سنجد دامنهٔ تخصیص (کارمند، واحد، نوع استخدام، گروه شغلی، گروه
// نظام پرداخت، همه) درست تطبیق داده می‌شود.
func TestASG02TargetMatching(t *testing.T) {
	employee := AllowanceEmployee{
		ID: "EMP-004", DepartmentName: "منابع انسانی", EmploymentType: "official_oil",
		JobGrade: "senior_expert", PayGroup: 1,
	}
	cases := []struct {
		target AllowanceTarget
		want   bool
	}{
		{AllowanceTarget{Kind: AllowanceTargetEmployee, Value: "EMP-004"}, true},
		{AllowanceTarget{Kind: AllowanceTargetEmployee, Value: "EMP-005"}, false},
		{AllowanceTarget{Kind: AllowanceTargetDepartment, Value: "منابع انسانی"}, true},
		{AllowanceTarget{Kind: AllowanceTargetDepartment, Value: "مالی"}, false},
		{AllowanceTarget{Kind: AllowanceTargetEmployment, Value: "official_oil"}, true},
		{AllowanceTarget{Kind: AllowanceTargetEmployment, Value: "contractor_staff"}, false},
		{AllowanceTarget{Kind: AllowanceTargetJobGrade, Value: "senior_expert"}, true},
		{AllowanceTarget{Kind: AllowanceTargetJobGrade, Value: "manager"}, false},
		{AllowanceTarget{Kind: AllowanceTargetPayGroup, Value: "1"}, true},
		{AllowanceTarget{Kind: AllowanceTargetPayGroup, Value: "2"}, false},
		{AllowanceTarget{Kind: AllowanceTargetAll}, true},
		{AllowanceTarget{Kind: "نامشخص", Value: "x"}, false},
	}
	for _, c := range cases {
		if got := c.target.Matches(employee); got != c.want {
			t.Fatalf("دامنهٔ %+v باید %v باشد، %v شد", c.target, c.want, got)
		}
	}
}

// TestASG03ResolveCalculation می‌سنجد پنج روش محاسبه (ماهانه ثابت، یک‌بار مرتبه، درصدی،
// ضریبی و فرمولی) درست حساب می‌شوند و پرچم مشمولیت کسور روی سطر می‌نشیند.
func TestASG03ResolveCalculation(t *testing.T) {
	employee := AllowanceEmployee{ID: "EMP-001", DepartmentName: "فناوری اطلاعات", EmploymentType: "official_oil", PayGroup: 1}
	env := AllowanceEnv{
		Bases:  map[string]float64{"base": 100_000_000, "allowanceBase": 125_000_000, "gross": 160_000_000},
		Vars:   map[string]float64{"CHILDREN": 2},
		Period: "1405/07",
	}
	assignments := []AllowanceAssignment{
		{ID: "A1", Code: "151", Calc: AllowanceCalcFixed, Rate: "9000000", Category: AllowanceCategoryEarning,
			Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01/01", Status: "active"},
		{ID: "A2", Code: "364", Calc: AllowanceCalcOnce, Rate: "50000000", Category: AllowanceCategoryEarning,
			Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/07/01", Status: "active"},
		{ID: "A3", Code: "144", Calc: AllowanceCalcPercent, Rate: "15", Basis: "allowanceBase",
			Category: AllowanceCategoryEarning, Taxable: true, Insured: true,
			Target:        AllowanceTarget{Kind: AllowanceTargetEmployee, Value: "EMP-001"},
			EffectiveFrom: "1405/01/01", Status: "active"},
		{ID: "A4", Code: "103", Calc: AllowanceCalcCoefficient, Rate: "0.5", Basis: "base",
			Category: AllowanceCategoryEarning, Target: AllowanceTarget{Kind: AllowanceTargetAll},
			EffectiveFrom: "1405/01/01", Status: "active"},
		{ID: "A5", Code: "181", Calc: AllowanceCalcFormula, Formula: "CHILDREN*1000000+BASE*0.01",
			Category: AllowanceCategoryEarning, Target: AllowanceTarget{Kind: AllowanceTargetAll},
			EffectiveFrom: "1405/01/01", Status: "active"},
		{ID: "A6", Code: "5400", Calc: AllowanceCalcFixed, Rate: "2500000", Category: AllowanceCategoryDeduction,
			Target:        AllowanceTarget{Kind: AllowanceTargetEmployee, Value: "EMP-009"},
			EffectiveFrom: "1405/01/01", Status: "active"},
		{ID: "A7", Code: "557", Calc: AllowanceCalcFixed, Rate: "1000000", Category: AllowanceCategoryDeduction,
			Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01/01",
			EffectiveTo: "1405/06/29", Status: "active"},
		{ID: "A8", Code: "151", Calc: AllowanceCalcFixed, Rate: "9000000", Category: AllowanceCategoryEarning,
			Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01/01", Status: "draft"},
	}

	lines := ResolveAllowanceAssignments(assignments, employee, env)
	if len(lines) != 5 {
		t.Fatalf("پنج سطر فعال انتظار می‌رفت (تخصیص پیش‌نویس، منقضی و کارمند دیگر کنار می‌روند)، %d آمد: %+v", len(lines), lines)
	}
	byCode := map[string]AllowanceLine{}
	for _, l := range lines {
		byCode[l.Code] = l
	}
	if got := byCode["151"].Amount; got != 9_000_000 {
		t.Fatalf("مبلغ ثابت مسکن باید ۹٬۰۰۰٬۰۰۰ باشد: %d", got)
	}
	if got := byCode["364"].Amount; got != 50_000_000 {
		t.Fatalf("یک‌بار مرتبه عیدی در ماه شروع باید ۵۰٬۰۰۰٬۰۰۰ باشد: %d", got)
	}
	if got := byCode["144"].Amount; got != 18_750_000 {
		t.Fatalf("۱۵٪ از مبنای فوق‌العاده‌ها (۱۲۵ م) باید ۱۸٬۷۵۰٬۰۰۰ باشد: %d", got)
	}
	if !byCode["144"].Taxable || !byCode["144"].Insured {
		t.Fatal("پرچم مشمولیت مالیات/بیمه باید روی سطر بیاید")
	}
	if got := byCode["103"].Amount; got != 50_000_000 {
		t.Fatalf("ضریب ۰٫۵ × پایهٔ ۱۰۰ م باید ۵۰٬۰۰۰٬۰۰۰ باشد: %d", got)
	}
	if got := byCode["181"].Amount; got != 3_000_000 {
		t.Fatalf("فرمول ۲×۱ م + ۱٪×۱۰۰ م باید ۳٬۰۰۰٬۰۰۰ باشد: %d", got)
	}

	totals := SumAllowanceLines(lines)
	if totals.Earnings != 9_000_000+50_000_000+18_750_000+50_000_000+3_000_000 {
		t.Fatalf("جمع مزایا نادرست است: %+v", totals)
	}
	if totals.TaxableEarnings != 18_750_000 || totals.InsuredEarnings != 18_750_000 {
		t.Fatalf("جمع مشمول مالیات/بیمه نادرست است: %+v", totals)
	}

	// ماه دیگری از پنجره: عیدی «یک‌بار مرتبه» و تخصیص پایان‌یافته نمی‌آیند.
	later := ResolveAllowanceAssignments(assignments, employee, AllowanceEnv{Bases: env.Bases, Vars: env.Vars, Period: "1405/08"})
	for _, l := range later {
		if l.Code == "364" || l.Code == "557" {
			t.Fatalf("سطر %s نباید در ماه ۱۴۰۵/۰۸ بیاید", l.Code)
		}
	}
}

// TestASG04Validation می‌سنجد ورودی ناقص/ناسازگار تخصیص رد می‌شود.
func TestASG04Validation(t *testing.T) {
	base := AllowanceAssignment{
		Code: "151", Calc: AllowanceCalcFixed, Rate: "9000000",
		Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01/01",
	}
	if errs := ValidateAllowanceAssignment(base); len(errs) != 0 {
		t.Fatalf("تخصیص معتبر نباید خطا بدهد: %+v", errs)
	}

	bad := []AllowanceAssignment{
		{Code: "999", Calc: AllowanceCalcFixed, Rate: "1", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01/01"},
		{Code: "151", Calc: AllowanceCalcFixed, Rate: "0", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01/01"},
		{Code: "151", Calc: AllowanceCalcPercent, Rate: "10", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01/01"},
		{Code: "151", Calc: AllowanceCalcFormula, Formula: "BASE ** 2", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01/01"},
		{Code: "151", Calc: AllowanceCalcFixed, Rate: "1", Target: AllowanceTarget{Kind: AllowanceTargetEmployee}, EffectiveFrom: "1405/01/01"},
		{Code: "151", Calc: AllowanceCalcFixed, Rate: "1", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "۱۴۰۵/۱/۱"},
	}
	for i, in := range bad {
		if errs := ValidateAllowanceAssignment(in); len(errs) == 0 {
			t.Fatalf("ورودی نادرست %d باید خطا بدهد: %+v", i, in)
		}
	}
}
