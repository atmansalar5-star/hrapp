package rules

import "testing"

// oilInput ورودی پایهٔ آزمون: پایهٔ حقوق ۲۰٬۰۰۰٬۰۰۰ ریال، بی‌نوبت‌کاری.
//
// با درصدهای مصوب آیین‌نامه: A=۲۰٬۰۰۰٬۰۰۰ · B=۵٬۰۰۰٬۰۰۰ (۲۵٪) ·
// مبنای فوق‌العاده‌ها=۲۵٬۰۰۰٬۰۰۰ · C=۷٬۵۰۰٬۰۰۰ (۳۰٪) · مبنای جذب=۳۲٬۵۰۰٬۰۰۰.
func oilInput() OilPayInput {
	return OilPayInput{BaseSalary: 20_000_000, DaysInMonth: 30}
}

// lineOf یک سطر فیش را از خروجی پیدا می‌کند.
func lineOf(slip OilPayslip, key string) (OilPayLine, bool) {
	for _, l := range slip.Lines {
		if l.Key == key {
			return l, true
		}
	}
	return OilPayLine{}, false
}

// TestOilClimateBuildsAllowanceBase: بدی آب و هوا ۲۵٪ پایه، و مبنا = پایه + بدی آب و هوا.
func TestOilClimateBuildsAllowanceBase(t *testing.T) {
	slip := ComputeOilPayslip(oilInput(), DefaultOilPayParams())
	if slip.ClimatePercent != 25 {
		t.Fatalf("درصد بدی آب و هوا ۲۵ است: %v", slip.ClimatePercent)
	}
	if slip.Climate != 5_000_000 {
		t.Fatalf("بدی آب و هوا ۲۵٪ از ۲۰٬۰۰۰٬۰۰۰ باید ۵٬۰۰۰٬۰۰۰ باشد: %d", slip.Climate)
	}
	if slip.AllowanceBase != 25_000_000 {
		t.Fatalf("مبنای فوق‌العاده‌ها باید جمع پایه و بدی آب و هوا باشد: %d", slip.AllowanceBase)
	}
}

// TestOilSpecialBuildsAttractionBase: ردیف ۳ آیین‌نامه — فوق‌العادهٔ ویژه ۳۰٪ (A+B) است و
// «مبنای جذب» را می‌سازد.
func TestOilSpecialBuildsAttractionBase(t *testing.T) {
	slip := ComputeOilPayslip(oilInput(), DefaultOilPayParams())
	// ۳۰٪ × ۲۵٬۰۰۰٬۰۰۰ = ۷٬۵۰۰٬۰۰۰
	if slip.Special != 7_500_000 {
		t.Fatalf("فوق‌العادهٔ ویژه باید ۷٬۵۰۰٬۰۰۰ باشد: %d", slip.Special)
	}
	if slip.AttractionBase != 32_500_000 {
		t.Fatalf("مبنای جذب باید A+B+C باشد: %d", slip.AttractionBase)
	}
	line, ok := lineOf(slip, OilAllowanceSpecial)
	if !ok || line.Percent != 30 || line.Base != 25_000_000 || line.Amount != 7_500_000 {
		t.Fatalf("سطر فوق‌العادهٔ ویژه نادرست است: %+v", line)
	}
}

// TestOilAllowanceCatalog: درصدهای مصوب روی مبنای فوق‌العاده‌ها.
func TestOilAllowanceCatalog(t *testing.T) {
	slip := ComputeOilPayslip(oilInput(), DefaultOilPayParams())
	want := map[string]int64{
		"deprivation":  6_875_000, // ۲۷٫۵٪ از ۲۵٬۰۰۰٬۰۰۰
		"special":      7_500_000, // ۳۰٪
		"timeConsumed": 4_125_000, // ۱۶٫۵٪
		"workshop":     2_500_000, // ۱۰٪
		"production":   7_500_000, // ۳۰٪
	}
	for key, amount := range want {
		line, ok := lineOf(slip, key)
		if !ok {
			t.Fatalf("قلم %s در فیش نیست", key)
		}
		if line.Base != 25_000_000 {
			t.Fatalf("مبنای قلم %s باید مبنای فوق‌العاده‌ها باشد: %d", key, line.Base)
		}
		if line.Amount != amount {
			t.Fatalf("مبلغ قلم %s نادرست است: %d (انتظار %d)", key, line.Amount, amount)
		}
	}
	// بدون پایهٔ سمت، قلم جذب و تخصصی شغل (ردیف ۷) ساخته نمی‌شود.
	if _, ok := lineOf(slip, OilAllowanceAttraction); ok {
		t.Fatal("بدون پایهٔ سمت نباید قلم جذب عمومی و تخصصی شغل ساخته شود")
	}
	// بدون نوبت‌کاری، قلم نوبت‌کار پرداخت نمی‌شود.
	if _, ok := lineOf(slip, OilAllowanceShift); ok {
		t.Fatal("قلم نوبت‌کار نباید برای کارمند غیر نوبت‌کار بیاید")
	}
	// جمع پنج قلم = ۲۸٬۵۰۰٬۰۰۰
	if slip.Allowances != 28_500_000 {
		t.Fatalf("جمع فوق‌العاده‌ها نادرست است: %d", slip.Allowances)
	}
}

// TestOilShiftWorker35: قلم نوبت‌کار ۳۵٪ مبنای فوق‌العاده‌ها، تمام ماه.
func TestOilShiftWorker35(t *testing.T) {
	in := oilInput()
	in.Shift = true
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	line, ok := lineOf(slip, OilAllowanceShift)
	if !ok {
		t.Fatal("قلم نوبت‌کار ساخته نشد")
	}
	// ۳۵٪ × ۲۵٬۰۰۰٬۰۰۰ = ۸٬۷۵۰٬۰۰۰
	if line.Amount != 8_750_000 {
		t.Fatalf("نوبت‌کار ماه کامل نادرست است: %d", line.Amount)
	}
	if slip.Allowances != 28_500_000+8_750_000 {
		t.Fatalf("جمع فوق‌العاده‌ها با نوبت‌کار نادرست است: %d", slip.Allowances)
	}
}

// TestOilShiftProrated: نیم‌ماه نوبت‌کاری = نصف قلم.
func TestOilShiftProrated(t *testing.T) {
	in := oilInput()
	in.Shift = true
	in.ShiftDays = 15
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	line, _ := lineOf(slip, OilAllowanceShift)
	if line.Amount != 4_375_000 {
		t.Fatalf("نوبت‌کار نیم‌ماه باید ۴٬۳۷۵٬۰۰۰ باشد: %d", line.Amount)
	}
}

// TestOilGrossAndDeductions: ناخالص = پایه + بدی آب و هوا + فوق‌العاده‌ها.
func TestOilGrossAndDeductions(t *testing.T) {
	slip := ComputeOilPayslip(oilInput(), DefaultOilPayParams())
	// ۲۰٬۰۰۰٬۰۰۰ + ۵٬۰۰۰٬۰۰۰ + ۲۸٬۵۰۰٬۰۰۰ = ۵۳٬۵۰۰٬۰۰۰
	if slip.Gross != 53_500_000 {
		t.Fatalf("جمع ناخالص نادرست است: %d", slip.Gross)
	}
	// کسور صندوق ۶٪ = ۳٬۲۱۰٬۰۰۰
	if slip.FundEmployee != 3_210_000 {
		t.Fatalf("کسور صندوق نادرست است: %d", slip.FundEmployee)
	}
	// سهم کارفرما ۱۹٪ = ۱۰٬۱۶۵٬۰۰۰
	if slip.FundEmployer != 10_165_000 {
		t.Fatalf("سهم کارفرما نادرست است: %d", slip.FundEmployer)
	}
	if slip.Net != slip.Gross-slip.Deductions || slip.Net <= 0 {
		t.Fatalf("خالص باید ناخالص منهای کسور باشد: %d", slip.Net)
	}
}

// TestOilOvertimePerHourRate: ردیف ۱۰ آیین‌نامه — اضافه‌کار هر ساعت ۰٫۰۱۴ پایهٔ حقوق برای
// روزکار و ۰٫۰۱۶ برای نوبت‌کار (نه بر مبنای فوق‌العاده‌ها).
func TestOilOvertimePerHourRate(t *testing.T) {
	in := oilInput()
	in.OvertimeHours = 8
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	if slip.OvertimeLine == nil {
		t.Fatal("سطر اضافه‌کار ساخته نشد")
	}
	// ۲۰٬۰۰۰٬۰۰۰ × ۰٫۰۱۴ × ۸ = ۲٬۲۴۰٬۰۰۰
	if slip.OvertimeLine.Amount != 2_240_000 {
		t.Fatalf("اضافه‌کار روزکار نادرست است: %d", slip.OvertimeLine.Amount)
	}

	shift := oilInput()
	shift.Shift = true
	shift.OvertimeHours = 8
	slip = ComputeOilPayslip(shift, DefaultOilPayParams())
	// ۲۰٬۰۰۰٬۰۰۰ × ۰٫۰۱۶ × ۸ = ۲٬۵۶۰٬۰۰۰
	if slip.OvertimeLine == nil || slip.OvertimeLine.Amount != 2_560_000 {
		t.Fatalf("اضافه‌کار نوبت‌کار نادرست است: %+v", slip.OvertimeLine)
	}
}

// TestOilOvertimeRatesParametric: نرخ‌های اضافه‌کار از پارامترها می‌آیند.
func TestOilOvertimeRatesParametric(t *testing.T) {
	params := DefaultOilPayParams()
	if params.OvertimeRateDay != 0.014 || params.OvertimeRateShift != 0.016 {
		t.Fatalf("نرخ مصوب اضافه‌کار ۰٫۰۱۴/۰٫۰۱۶ است: %v / %v", params.OvertimeRateDay, params.OvertimeRateShift)
	}
	params.OvertimeRateDay = 0.02
	in := oilInput()
	in.OvertimeHours = 10
	slip := ComputeOilPayslip(in, params)
	// ۲۰٬۰۰۰٬۰۰۰ × ۰٫۰۲ × ۱۰ = ۴٬۰۰۰٬۰۰۰
	if slip.OvertimeLine == nil || slip.OvertimeLine.Amount != 4_000_000 {
		t.Fatalf("نرخ پارامتریک اضافه‌کار اعمال نشد: %+v", slip.OvertimeLine)
	}
}

// TestOilManualPercentOverridesParam: درصد دستی بر مصوب مقدم است.
func TestOilManualPercentOverridesParam(t *testing.T) {
	in := oilInput()
	in.AllowancePercents = map[string]float64{"deprivation": 30}
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	line, ok := lineOf(slip, "deprivation")
	if !ok || line.Percent != 30 || line.Amount != 7_500_000 {
		t.Fatalf("درصد دستی محرومیت نادرست اعمال شد: %+v", line)
	}
}

// TestOilCustomClimatePercent: درصد بدی آب و هوای دستی، مبنای جذب را هم به‌روز می‌کند.
func TestOilCustomClimatePercent(t *testing.T) {
	in := oilInput()
	in.ClimatePercent = 30
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	if slip.Climate != 6_000_000 || slip.AllowanceBase != 26_000_000 {
		t.Fatalf("درصد دستی بدی آب و هوا نادرست اعمال شد: climate=%d base=%d", slip.Climate, slip.AllowanceBase)
	}
	// C = ۳۰٪ × ۲۶٬۰۰۰٬۰۰۰ = ۷٬۸۰۰٬۰۰۰ · مبنای جذب = ۳۳٬۸۰۰٬۰۰۰
	if slip.Special != 7_800_000 || slip.AttractionBase != 33_800_000 {
		t.Fatalf("مبنای جذب با درصد دستی نادرست است: special=%d attraction=%d", slip.Special, slip.AttractionBase)
	}
}

// TestOilZeroBaseIsSafe: پایهٔ صفر نباید خطا بدهد.
func TestOilZeroBaseIsSafe(t *testing.T) {
	slip := ComputeOilPayslip(OilPayInput{}, DefaultOilPayParams())
	if slip.Gross != 0 || slip.Net != 0 || slip.Allowances != 0 {
		t.Fatalf("پایهٔ صفر باید فیش صفر بدهد: %+v", slip)
	}
}

// TestOilCatalogMatchesApprovedTable: فهرست اقلام همان جدول مصوب آیین‌نامه است.
func TestOilCatalogMatchesApprovedTable(t *testing.T) {
	params := DefaultOilPayParams()
	want := map[string]float64{
		"deprivation": 27.5, "special": 30, "timeConsumed": 16.5, "workshop": 10,
		"attractionSpecialty": 0, "production": 30, "shiftWorker": 35,
	}
	if len(params.Allowances) != len(want) {
		t.Fatalf("شمار اقلام مصوب %d است: %d", len(want), len(params.Allowances))
	}
	for _, a := range params.Allowances {
		expected, ok := want[a.Key]
		if !ok || a.Percent != expected {
			t.Fatalf("درصد قلم %s نادرست است: %v", a.Key, a.Percent)
		}
	}
	if params.ClimatePercent != 25 {
		t.Fatalf("درصد بدی آب و هوای مصوب ۲۵ است: %v", params.ClimatePercent)
	}
}

// TestOilPercentFa قالب درصد برای نمایش.
func TestOilPercentFa(t *testing.T) {
	if OilPercentFa(27.5) != "27.5" || OilPercentFa(10) != "10" {
		t.Fatalf("قالب درصد نادرست است: %s / %s", OilPercentFa(27.5), OilPercentFa(10))
	}
}

// TestOilJobGradeTable: جدول ۳ آیین‌نامه — جمع (Y) هر پایهٔ سمت.
func TestOilJobGradeTable(t *testing.T) {
	want := map[string]float64{
		"6-10": 12, "11-15": 16, "16": 23, "17": 28,
		"A": 33, "B": 43, "C": 68, "D": 88, "OD": 108,
	}
	if len(OilJobGrades()) != len(want) {
		t.Fatalf("شمار ردیف‌های جدول ۳ نادرست است: %d", len(OilJobGrades()))
	}
	for key, total := range want {
		grade, ok := OilJobGradeByKey(key)
		if !ok {
			t.Fatalf("پایهٔ سمت %s در جدول ۳ نیست", key)
		}
		if grade.Total() != total {
			t.Fatalf("جمع (Y) پایهٔ %s نادرست است: %v (انتظار %v)", key, grade.Total(), total)
		}
	}
	// حروف کوچک هم پذیرفته می‌شوند و کلید ناشناخته مبنای پرداخت نمی‌شود.
	if grade, ok := OilJobGradeByKey("od"); !ok || grade.Key != "OD" {
		t.Fatalf("کلید OD با حروف کوچک شناسایی نشد: %+v", grade)
	}
	if _, ok := OilJobGradeByKey("99"); ok {
		t.Fatal("پایهٔ سمت ناشناخته نباید شناسایی شود")
	}
}

// TestOilAttractionOnAttractionBase: ردیف ۷ — درصد پایهٔ سمت روی مبنای (A+B+C).
func TestOilAttractionOnAttractionBase(t *testing.T) {
	in := oilInput()
	in.JobGrade = "A"
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	line, ok := lineOf(slip, OilAllowanceAttraction)
	if !ok {
		t.Fatal("قلم جذب عمومی و تخصصی شغل ساخته نشد")
	}
	// ۳۳٪ × ۳۲٬۵۰۰٬۰۰۰ = ۱۰٬۷۲۵٬۰۰۰
	if line.Percent != 33 || line.Base != 32_500_000 || line.Amount != 10_725_000 {
		t.Fatalf("قلم جذب پایهٔ سمت A نادرست است: %+v", line)
	}
	if slip.JobGrade != "A" || slip.JobGradeFa == "" {
		t.Fatalf("پایهٔ سمت در فیش ثبت نشده است: %q", slip.JobGrade)
	}
	// ناخالص = ۵۳٬۵۰۰٬۰۰۰ + ۱۰٬۷۲۵٬۰۰۰
	if slip.Gross != 64_225_000 {
		t.Fatalf("ناخالص با قلم جذب نادرست است: %d", slip.Gross)
	}
}

// TestOilDegreeSpecialty: فوق‌العادهٔ تخصصی مدرک (ضریب Z) روی مبنای جذب.
func TestOilDegreeSpecialty(t *testing.T) {
	// مبنای جذب در این ورودی ۳۲٬۵۰۰٬۰۰۰ ریال است.
	cases := map[string]int64{
		"diploma":   5_687_500,  // ۱۷٫۵٪
		"associate": 8_775_000,  // ۲۷٪
		"bachelor":  12_512_500, // ۳۸٫۵٪
		"master":    15_015_000, // ۴۶٫۲٪
		"phd":       17_550_000, // ۵۴٪
	}
	for degree, want := range cases {
		in := oilInput()
		in.Degree = degree
		slip := ComputeOilPayslip(in, DefaultOilPayParams())
		line, ok := lineOf(slip, OilDegreeAllowance)
		if !ok {
			t.Fatalf("قلم تخصصی برای مدرک %s ساخته نشد", degree)
		}
		if line.Amount != want {
			t.Fatalf("مبلغ تخصصی %s نادرست است: %d (انتظار %d)", degree, line.Amount, want)
		}
		if line.Base != 32_500_000 {
			t.Fatalf("مبنای قلم تخصصی باید مبنای جذب باشد: %d", line.Base)
		}
		if slip.Degree != degree || slip.DegreePercent != line.Percent {
			t.Fatalf("مدرک و درصد در فیش ثبت نشده است: %s / %v", slip.Degree, slip.DegreePercent)
		}
		// جمع ناخالص باید شامل همین مبلغ باشد.
		if slip.Gross != 53_500_000+want {
			t.Fatalf("جمع ناخالص با تخصصی %s نادرست است: %d", degree, slip.Gross)
		}
	}
}

// TestOilRow7MatchesBylaw: جمع دو سطر «جذب عمومی و تخصصی شغل» و «تخصصی مدرک» باید همان
// فرمول ردیف ۷ آیین‌نامه باشد: (A+B+C) × (Y+Z).
func TestOilRow7MatchesBylaw(t *testing.T) {
	in := oilInput()
	in.JobGrade = "A"      // Y = ۳۳
	in.Degree = "bachelor" // Z = ۳۸٫۵
	slip := ComputeOilPayslip(in, DefaultOilPayParams())

	attraction, okAttraction := lineOf(slip, OilAllowanceAttraction)
	degree, okDegree := lineOf(slip, OilDegreeAllowance)
	if !okAttraction || !okDegree {
		t.Fatal("ردیف ۷ باید دو سطر جذب و تخصصی مدرک داشته باشد")
	}
	// (A+B+C) × (Y+Z) = ۳۲٬۵۰۰٬۰۰۰ × ۷۱٫۵٪ = ۲۳٬۲۳۷٬۵۰۰
	const wantRow7 = 23_237_500
	if attraction.Amount+degree.Amount != wantRow7 {
		t.Fatalf("جمع ردیف ۷ نادرست است: %d (انتظار %d)",
			attraction.Amount+degree.Amount, wantRow7)
	}
}

// TestOilDegreeNotSet: بی‌مدرک یا مقطع بدون مصوبه، قلم تخصصی ندارد.
func TestOilDegreeNotSet(t *testing.T) {
	slip := ComputeOilPayslip(oilInput(), DefaultOilPayParams())
	if _, ok := lineOf(slip, OilDegreeAllowance); ok {
		t.Fatal("بدون مدرک نباید قلم تخصصی ساخته شود")
	}
	in := oilInput()
	in.Degree = "postdoc" // مقطعی که درصد مصوب ندارد
	slip = ComputeOilPayslip(in, DefaultOilPayParams())
	if _, ok := lineOf(slip, OilDegreeAllowance); ok {
		t.Fatal("مقطع بدون درصد مصوب نباید قلم تخصصی بسازد")
	}
	if slip.Gross != 53_500_000 {
		t.Fatalf("ناخالص بدون تخصصی باید ۵۳٬۵۰۰٬۰۰۰ باشد: %d", slip.Gross)
	}
}

// TestOilDegreePercentsParametric: درصدهای تخصصی همان جدول مصوب‌اند و از پارامتر می‌آیند.
func TestOilDegreePercentsParametric(t *testing.T) {
	params := DefaultOilPayParams()
	want := map[string]float64{"diploma": 17.5, "associate": 27, "bachelor": 38.5, "master": 46.2, "phd": 54}
	for degree, percent := range want {
		if got := OilDegreePercent(params, degree); got != percent {
			t.Fatalf("درصد مدرک %s نادرست است: %v", degree, got)
		}
	}
	if OilDegreePercent(params, "postdoc") != 0 || OilDegreePercent(params, "") != 0 {
		t.Fatal("مقطع بدون مصوبه باید صفر باشد")
	}
	if OilDegreeAllowanceTitleFa() != "فوق‌العادهٔ تخصصی بر اساس مدرک تحصیلی" {
		t.Fatalf("عنوان قلم تخصصی نادرست است: %s", OilDegreeAllowanceTitleFa())
	}
}

// TestOilResidenceTables: جدول ۱ و ۲ آیین‌نامه (محل سکونت → درصدها).
func TestOilResidenceTables(t *testing.T) {
	want := map[string][3]float64{
		"mahshahr":    {27.5, 16.5, 18},
		"bandar_imam": {25, 9.5, 10.5},
		"jarrahi":     {27.5, 16.5, 18},
		"ramshir":     {30, 25, 27},
		"ramhormoz":   {37.5, 25, 27},
		"shadegan":    {30, 27.5, 29},
		"omidieh":     {30, 31, 33},
		"abadan":      {27.5, 37.5, 40},
	}
	sites := OilResidences()
	if len(sites) != len(want) {
		t.Fatalf("شمار ردیف‌های جدول ۱ و ۲ نادرست است: %d", len(sites))
	}
	for key, values := range want {
		site, ok := OilResidenceByKey(key)
		if !ok {
			t.Fatalf("محل سکونت %s در جدول نیست", key)
		}
		if site.Deprivation != values[0] || site.TimeConsumedDay != values[1] || site.TimeConsumedShift != values[2] {
			t.Fatalf("درصدهای %s نادرست است: %+v", key, site)
		}
	}
}

// TestOilResidenceDrivesPercentages: با انتخاب محل سکونت، درصد محرومیت و صرف وقت از جدول
// می‌آید و بر درصد پارامتر شرکت اولویت دارد.
func TestOilResidenceDrivesPercentages(t *testing.T) {
	in := oilInput()
	in.Residence = "mahshahr"
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	deprivation, _ := lineOf(slip, OilAllowanceDeprivation)
	if deprivation.Percent != 27.5 || deprivation.Amount != 6_875_000 {
		t.Fatalf("محرومیت ماهشهر باید ۲۷٫۵٪ و ۶٬۸۷۵٬۰۰۰ باشد: %+v", deprivation)
	}
	timeConsumed, _ := lineOf(slip, OilAllowanceTimeConsumed)
	if timeConsumed.Percent != 16.5 || timeConsumed.Amount != 4_125_000 {
		t.Fatalf("صرف وقت روزکار ماهشهر باید ۱۶٫۵٪ و ۴٬۱۲۵٬۰۰۰ باشد: %+v", timeConsumed)
	}
	if slip.Residence != "mahshahr" || slip.ResidenceFa == "" {
		t.Fatalf("محل سکونت در فیش ثبت نشده است: %q", slip.Residence)
	}
}

// TestOilResidenceShiftColumn: ستون نوبت‌کار جدول ۲ با وضعیت نوبت‌کاری انتخاب می‌شود.
func TestOilResidenceShiftColumn(t *testing.T) {
	in := oilInput()
	in.Residence = "abadan"
	in.Shift = true
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	timeConsumed, _ := lineOf(slip, OilAllowanceTimeConsumed)
	// نوبت‌کار آبادان ۴۰٪ × ۲۵٬۰۰۰٬۰۰۰ = ۱۰٬۰۰۰٬۰۰۰
	if timeConsumed.Percent != 40 || timeConsumed.Amount != 10_000_000 {
		t.Fatalf("صرف وقت نوبت‌کار آبادان نادرست است: %+v", timeConsumed)
	}
	deprivation, _ := lineOf(slip, OilAllowanceDeprivation)
	// جدول ۱ برای آبادان ۲۷٫۵٪ است (مستقل از نوبت‌کاری).
	if deprivation.Percent != 27.5 || deprivation.Amount != 6_875_000 {
		t.Fatalf("محرومیت آبادان نادرست است: %+v", deprivation)
	}
}

// TestOilUnknownResidenceFallsBackToParams: محل سکونت ناشناخته درصد پارامتر شرکت را
// دست‌نخورده می‌گذارد.
func TestOilUnknownResidenceFallsBackToParams(t *testing.T) {
	in := oilInput()
	in.Residence = "tehran"
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	deprivation, _ := lineOf(slip, OilAllowanceDeprivation)
	if deprivation.Percent != 27.5 {
		t.Fatalf("محل سکونت ناشناخته نباید درصد را عوض کند: %v", deprivation.Percent)
	}
	if slip.Residence != "" {
		t.Fatalf("محل سکونت ناشناخته نباید در فیش ثبت شود: %q", slip.Residence)
	}
}

// TestOilResidenceOverriddenByManualPercent: درصد دستی فرم بر جدول محل سکونت مقدم است.
func TestOilResidenceOverriddenByManualPercent(t *testing.T) {
	in := oilInput()
	in.Residence = "mahshahr"
	in.AllowancePercents = map[string]float64{"deprivation": 12}
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	deprivation, _ := lineOf(slip, OilAllowanceDeprivation)
	if deprivation.Percent != 12 || deprivation.Amount != 3_000_000 {
		t.Fatalf("درصد دستی باید بر جدول مقدم باشد: %+v", deprivation)
	}
}
