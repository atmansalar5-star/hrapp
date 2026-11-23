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

// TestASG10ReportingFlags می‌سنجد جای نمایش هر قلم تخصیص (فیش حقوقی، گزارش‌های چاپی و کد
// گزارش رسمی) در محاسبه و جمع‌ها رعایت می‌شود و کد گزارش نامعتبر رد می‌شود.
func TestASG10ReportingFlags(t *testing.T) {
	employee := AllowanceEmployee{ID: "EMP-001"}
	assignments := []AllowanceAssignment{
		{ID: "A1", Code: "151", TitleFa: "مسکن", Category: AllowanceCategoryEarning,
			Calc: AllowanceCalcFixed, Rate: "9000000", ShowOnPayslip: true, ShowInReports: false,
			ReportCode: "151-R", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01"},
		{ID: "A2", Code: "5400", TitleFa: "کسر اقساط وام اضطراری", Category: AllowanceCategoryDeduction,
			Calc: AllowanceCalcFixed, Rate: "2000000", ShowOnPayslip: false, ShowInReports: true,
			ReportCode: "5400-R", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01"},
	}
	lines := ResolveAllowanceAssignments(assignments, employee, AllowanceEnv{Period: "1405/02"})
	if len(lines) != 2 {
		t.Fatalf("دو تخصیص باید دو سطر بدهد: %+v", lines)
	}
	if lines[0].ReportCode != "151-R" || !lines[0].ShowOnPayslip || lines[0].ShowInReports {
		t.Fatalf("پرچم‌های نمایش سطر اول درست منتقل نشده: %+v", lines[0])
	}
	totals := SumAllowanceLines(lines)
	if totals.PayslipEarnings != 9_000_000 || totals.ReportEarnings != 0 {
		t.Fatalf("جمع مزایای فیش/گزارش درست نیست: %+v", totals)
	}
	if totals.PayslipDeductions != 0 || totals.ReportDeductions != 2_000_000 {
		t.Fatalf("جمع کسور فیش/گزارش درست نیست: %+v", totals)
	}
	if payslip := AllowancePayslipLines(lines); len(payslip) != 1 || payslip[0].Code != "151" {
		t.Fatalf("سطرهای فیش باید فقط اقلام «نمایش در فیش» باشند: %+v", payslip)
	}
	if reports := AllowanceReportLines(lines); len(reports) != 1 || reports[0].Code != "5400" {
		t.Fatalf("سطرهای گزارش باید فقط اقلام «درج در گزارش» باشند: %+v", reports)
	}

	base := AllowanceAssignment{Code: "151", Calc: AllowanceCalcFixed, Rate: "1",
		Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01"}
	base.ReportCode = "151-R"
	if errs := ValidateAllowanceAssignment(base); len(errs) != 0 {
		t.Fatalf("کد گزارش معتبر نباید خطا بدهد: %+v", errs)
	}
	base.ReportCode = "کد نادرست!"
	if errs := ValidateAllowanceAssignment(base); errs["reportCode"] == "" {
		t.Fatalf("کد گزارش نامعتبر باید رد شود: %+v", errs)
	}
}

// TestASG12AllowanceDefinition می‌سنجد تعریف فوق العاده (گام یکم) درست سنجیده و کامل
// می‌شود: کد رسمی یا کد داخلی، روش محاسبه، متغیرها و کد گزارش.
func TestASG12AllowanceDefinition(t *testing.T) {
	def := NormalizeAllowanceDefinition(AllowanceDefinition{
		Code: "151", Calc: AllowanceCalcFixed, Rate: "9000000", VarsText: "SITE=1",
		ShowOnPayslip: true, ShowInReports: true, ReportCode: "151-R",
	})
	if !def.CodeOfficial || def.CodeTitleFa != "مسکن" || def.TitleFa != "مسکن" {
		t.Fatalf("کد رسمی باید عنوان و نوع قلم را بدهد: %+v", def)
	}
	if def.Category != AllowanceCategoryEarning {
		t.Fatalf("نوع قلم از کد رسمی استنتاج نشد: %+v", def)
	}
	if errs := ValidateAllowanceDefinition(def); len(errs) != 0 {
		t.Fatalf("تعریف معتبر نباید خطا بدهد: %+v", errs)
	}

	// کد داخلی (قلم بی‌کد رسمی) هم پذیرفته می‌شود ولی «رسمی» نشان نمی‌خورد.
	internal := NormalizeAllowanceDefinition(AllowanceDefinition{
		Code: "X-100", TitleFa: "فوق العادهٔ داخلی", Category: AllowanceCategoryEarning,
		Calc: AllowanceCalcFixed, Rate: "1000000",
	})
	if internal.CodeOfficial {
		t.Fatal("کد داخلی نباید رسمی شمرده شود")
	}
	if errs := ValidateAllowanceDefinition(internal); len(errs) != 0 {
		t.Fatalf("کد داخلی معتبر نباید خطا بدهد: %+v", errs)
	}

	// متغیرها: نام=مقدار، جدا‌شده با کاما/نقطه‌ویرگول/خط تازه و با ارقام فارسی.
	vars, problems := ParseAllowanceVars("children=2, RATE=0.15؛ SITE=۱\nEXTRA=3")
	if len(problems) != 0 {
		t.Fatalf("متغیرهای معتبر نباید ایراد بدهند: %v", problems)
	}
	if vars["CHILDREN"] != 2 || vars["RATE"] != 0.15 || vars["SITE"] != 1 || vars["EXTRA"] != 3 {
		t.Fatalf("متغیرها درست خوانده نشدند: %+v", vars)
	}
	if text := AllowanceVarsText(vars); text != "CHILDREN=2, EXTRA=3, RATE=0.15, SITE=1" {
		t.Fatalf("متن متغیرها باید مرتب و خوانا باشد: %q", text)
	}

	badVars := []string{"CHILDREN", "2X=3", "A=x", "RATE="}
	for _, raw := range badVars {
		if _, problems := ParseAllowanceVars(raw); len(problems) == 0 {
			t.Fatalf("متغیر نامعتبر «%s» باید ایراد بدهد", raw)
		}
	}

	// ورودی نامعتبر تعریف: کد بی‌شکل، روش محاسبه نبود، مقدار صفر، کد گزارش بدشکل.
	badDefs := []AllowanceDefinition{
		{Code: "!!", Calc: AllowanceCalcFixed, Rate: "1"},
		{Code: "151", Rate: "1"},
		{Code: "151", Calc: AllowanceCalcFixed, Rate: "0"},
		{Code: "151", Calc: AllowanceCalcPercent, Rate: "10"},
		{Code: "151", Calc: AllowanceCalcFormula},
		{Code: "151", Calc: AllowanceCalcFixed, Rate: "1", VarsText: "CHILDREN=2x"},
		{Code: "151", Calc: AllowanceCalcFixed, Rate: "1", ReportCode: "کد گزارش!"},
	}
	for i, in := range badDefs {
		if errs := ValidateAllowanceDefinition(in); len(errs) == 0 {
			t.Fatalf("تعریف نادرست %d باید خطا بدهد: %+v", i, in)
		}
	}
}

// TestASG13DefinitionMerge می‌سنجد تخصیص (گام دوم) با تعریفش یکی می‌شود، بازنویسی مقدار و
// مبنا اثر می‌گذارد، متغیرهای تعریف به محاسبه می‌رسند و تخصیص قدیمی دست‌نخورده می‌ماند.
func TestASG13DefinitionMerge(t *testing.T) {
	def := NormalizeAllowanceDefinition(AllowanceDefinition{
		ID: "ALD-0001", Code: "122", Calc: AllowanceCalcPercent, Rate: "10", Basis: "allowanceBase",
		Taxable: true, Insured: true, ShowOnPayslip: true, ShowInReports: true, ReportCode: "122-R",
		VarsText: "SITE=1",
	})
	employee := AllowanceEmployee{ID: "EMP-001", EmploymentType: "official_oil"}
	env := AllowanceEnv{Bases: map[string]float64{"allowanceBase": 100_000_000, "base": 50_000_000}, Period: "1405/02"}

	// الف) بدون بازنویسی: ۱۰٪ مبنای فوق العاده = ۱۰ میلیون.
	plain := AllowanceAssignment{
		ID: "ALW-1", Definition: "ALD-0001", Status: "active",
		Target: AllowanceTarget{Kind: AllowanceTargetEmployment, Value: "official_oil"}, EffectiveFrom: "1405/01",
	}
	lines := ResolveAllowanceLines(map[string]AllowanceDefinition{"ALD-0001": def}, []AllowanceAssignment{plain}, employee, env)
	if len(lines) != 1 {
		t.Fatalf("تخصیص تعریف‌محور باید یک سطر بدهد: %+v", lines)
	}
	line := lines[0]
	if line.Code != "122" || line.Amount != 10_000_000 || line.DefinitionID != "ALD-0001" {
		t.Fatalf("سطر تعریف‌محور درست نیست: %+v", line)
	}
	if !line.Taxable || !line.Insured || line.ReportCode != "122-R" || !line.ShowOnPayslip {
		t.Fatalf("پرچم‌ها باید از تعریف بیایند: %+v", line)
	}

	// ب) بازنویسی مقدار روی همان تعریف: ۱۵٪ مالک ۱۰٪ است.
	override := plain
	override.RateOverride = "15"
	lines = ResolveAllowanceLines(map[string]AllowanceDefinition{"ALD-0001": def}, []AllowanceAssignment{override}, employee, env)
	if lines[0].Amount != 15_000_000 {
		t.Fatalf("بازنویسی مقدار باید اثر کند: %+v", lines[0])
	}

	// ج) بازنویسی مبنا.
	overrideBasis := override
	overrideBasis.BasisOverride = "base"
	lines = ResolveAllowanceLines(map[string]AllowanceDefinition{"ALD-0001": def}, []AllowanceAssignment{overrideBasis}, employee, env)
	if lines[0].Amount != 7_500_000 || lines[0].Basis != "base" {
		t.Fatalf("بازنویسی مبنا باید اثر کند: %+v", lines[0])
	}

	// د) متغیرهای تعریف در فرمول به کار می‌روند و متغیر درخواست مقدم است.
	formulaDef := NormalizeAllowanceDefinition(AllowanceDefinition{
		ID: "ALD-0002", Code: "X-100", TitleFa: "فرمولی", Category: AllowanceCategoryEarning,
		Calc: AllowanceCalcFormula, Formula: "BASE*CHILDREN", VarsText: "CHILDREN=2",
	})
	formulaAssignment := AllowanceAssignment{
		ID: "ALW-2", Definition: "ALD-0002", Status: "active",
		Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01",
	}
	lines = ResolveAllowanceLines(map[string]AllowanceDefinition{"ALD-0002": formulaDef}, []AllowanceAssignment{formulaAssignment}, employee, env)
	if lines[0].Amount != 100_000_000 {
		t.Fatalf("متغیر تعریف در فرمول به کار نرفت: %+v", lines[0])
	}
	envWithVar := env
	envWithVar.Vars = map[string]float64{"CHILDREN": 3}
	lines = ResolveAllowanceLines(map[string]AllowanceDefinition{"ALD-0002": formulaDef}, []AllowanceAssignment{formulaAssignment}, employee, envWithVar)
	if lines[0].Amount != 150_000_000 {
		t.Fatalf("متغیر درخواست باید مقدم بر متغیر تعریف باشد: %+v", lines[0])
	}

	// ه) تخصیص قدیمی (بدون تعریف) دست‌نخورده حساب می‌شود.
	legacy := AllowanceAssignment{
		ID: "ALW-3", Code: "151", Category: AllowanceCategoryEarning, Calc: AllowanceCalcFixed, Rate: "9000000",
		Status: "active", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01",
	}
	lines = ResolveAllowanceLines(map[string]AllowanceDefinition{"ALD-0001": def}, []AllowanceAssignment{legacy}, employee, env)
	if len(lines) != 1 || lines[0].Amount != 9_000_000 || lines[0].DefinitionID != "" {
		t.Fatalf("تخصیص درون‌خطی قدیمی باید بی‌تغییر بماند: %+v", lines)
	}

	// و) تعریف پیدا نشود: سطر با توضیح صریح می‌آید و مبلغ نمی‌سازد.
	lines = ResolveAllowanceLines(nil, []AllowanceAssignment{plain}, employee, env)
	if len(lines) != 1 || lines[0].Amount != 0 || lines[0].DetailFa == "" {
		t.Fatalf("نبود تعریف باید صریح گزارش شود: %+v", lines)
	}
}

// TestASG17OverlapWarning می‌سنجد هم‌پوشانی تخصیص‌ها (یک قلم از دو تخصیص فعال) هشدار
// می‌دهد تا مبلغ بی‌صدا دوبار حساب نشود.
func TestASG17OverlapWarning(t *testing.T) {
	employee := AllowanceEmployee{ID: "EMP-001", EmploymentType: "official_oil"}
	def := NormalizeAllowanceDefinition(AllowanceDefinition{
		ID: "ALD-0001", Code: "122", Calc: AllowanceCalcFixed, Rate: "1000000",
		ShowOnPayslip: true, ShowInReports: true,
	})
	defs := map[string]AllowanceDefinition{"ALD-0001": def}

	// الف) یک تخصیص گروهی + یک تخصیص فردی روی همان تعریف ⇒ هشدار.
	group := AllowanceAssignment{ID: "ALW-1", Definition: "ALD-0001", Status: "active",
		Target: AllowanceTarget{Kind: AllowanceTargetEmployment, Value: "official_oil"}, EffectiveFrom: "1405/01"}
	person := AllowanceAssignment{ID: "ALW-2", Definition: "ALD-0001", Status: "active",
		Target: AllowanceTarget{Kind: AllowanceTargetEmployee, Value: "EMP-001"}, EffectiveFrom: "1405/01"}
	lines := ResolveAllowanceLines(defs, []AllowanceAssignment{group, person}, employee, AllowanceEnv{Period: "1405/02"})
	warnings := AllowanceOverlapWarnings(lines)
	if len(lines) != 2 || len(warnings) != 1 {
		t.Fatalf("دو تخصیص روی یک تعریف باید یک هشدار بدهد: lines=%d warnings=%v", len(lines), warnings)
	}
	if !strings.Contains(warnings[0], "122") || !strings.Contains(warnings[0], "2") {
		t.Fatalf("متن هشدار باید کد و تعداد را بگوید: %q", warnings[0])
	}

	// ب) اقلام متفاوت ⇒ بی‌هشدار.
	other := NormalizeAllowanceDefinition(AllowanceDefinition{
		ID: "ALD-0002", Code: "151", Calc: AllowanceCalcFixed, Rate: "9000000"})
	otherAssign := AllowanceAssignment{ID: "ALW-3", Definition: "ALD-0002", Status: "active",
		Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01"}
	defs["ALD-0002"] = other
	lines = ResolveAllowanceLines(defs, []AllowanceAssignment{group, otherAssign}, employee, AllowanceEnv{Period: "1405/02"})
	if warnings := AllowanceOverlapWarnings(lines); len(warnings) != 0 {
		t.Fatalf("اقلام متفاوت نباید هشدار بدهند: %v", warnings)
	}

	// ج) تخصیص قدیمی و تعریف‌محور با همان کد ⇒ هشدار (هم‌پوشانی کد).
	legacy := AllowanceAssignment{ID: "ALW-4", Code: "122", Calc: AllowanceCalcFixed, Rate: "1000000",
		Status: "active", Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01"}
	lines = ResolveAllowanceLines(defs, []AllowanceAssignment{group, legacy}, employee, AllowanceEnv{Period: "1405/02"})
	if warnings := AllowanceOverlapWarnings(lines); len(warnings) != 1 {
		t.Fatalf("هم‌پوشانی کد یکسان باید هشدار بدهد (حتی اگر یکی تعریف‌محور باشد): %v", warnings)
	}

	// د) سطر بی‌تخصیص (تعریف گمشده) در هشدار شمرده نمی‌شود.
	missing := AllowanceAssignment{ID: "ALW-5", Definition: "ALD-9999", Status: "active",
		Target: AllowanceTarget{Kind: AllowanceTargetAll}, EffectiveFrom: "1405/01"}
	lines = ResolveAllowanceLines(defs, []AllowanceAssignment{group, missing}, employee, AllowanceEnv{Period: "1405/02"})
	if warnings := AllowanceOverlapWarnings(lines); len(warnings) != 0 {
		t.Fatalf("سطر پیام‌دار نباید هشدار هم‌پوشانی بسازد: %v", warnings)
	}
}
