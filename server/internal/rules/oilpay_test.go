package rules

import "testing"

// oilInput ورودی پایهٔ آزمون: پایهٔ حقوق ۲۰٬۰۰۰٬۰۰۰ ریال، بی‌نوبت‌کاری.
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

// TestOilAllowanceCatalog: درصدهای جدول مصوب روی مبنای فوق‌العاده‌ها.
func TestOilAllowanceCatalog(t *testing.T) {
	slip := ComputeOilPayslip(oilInput(), DefaultOilPayParams())
	want := map[string]int64{
		"deprivation":         6_875_000, // ۲۷٫۵٪ از ۲۵٬۰۰۰٬۰۰۰
		"special":             750_000,   // ۳٪
		"timeConsumed":        4_125_000, // ۱۶٫۵٪
		"workshop":            2_500_000, // ۱۰٪
		"attractionSpecialty": 6_750_000, // ۲۷٪
		"production":          7_500_000, // ۳۰٪
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
	// بدون نوبت‌کاری، قلم نوبت‌کار پرداخت نمی‌شود.
	if _, ok := lineOf(slip, OilAllowanceShift); ok {
		t.Fatal("قلم نوبت‌کار نباید برای کارمند غیر نوبت‌کار بیاید")
	}
	// جمع شش قلم = ۲۸٬۵۰۰٬۰۰۰
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
	// کسور صندوق ۷٪ = ۳٬۷۴۵٬۰۰۰
	if slip.FundEmployee != 3_745_000 {
		t.Fatalf("کسور صندوق نادرست است: %d", slip.FundEmployee)
	}
	// سهم کارفرما ۲۰٪ = ۱۰٬۷۰۰٬۰۰۰
	if slip.FundEmployer != 10_700_000 {
		t.Fatalf("سهم کارفرما نادرست است: %d", slip.FundEmployer)
	}
	if slip.Net != slip.Gross-slip.Deductions || slip.Net <= 0 {
		t.Fatalf("خالص باید ناخالص منهای کسور باشد: %d", slip.Net)
	}
}

// TestOilOvertimeUsesAllowanceBase: اضافه‌کاری از نرخ ساعتی مبنا با ضریب پارامتریک.
func TestOilOvertimeUsesAllowanceBase(t *testing.T) {
	in := oilInput()
	in.OvertimeHours = 8
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	if slip.OvertimeLine == nil {
		t.Fatal("سطر اضافه‌کاری ساخته نشد")
	}
	// ۲۵٬۰۰۰٬۰۰۰ ÷ ۱۷۶ × ۸ × ۱٫۴ = ۱٬۵۹۰٬۹۰۹
	if slip.OvertimeLine.Amount != 1_590_909 {
		t.Fatalf("اضافه‌کاری نادرست است: %d", slip.OvertimeLine.Amount)
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

// TestOilCustomClimatePercent: درصد بدی آب و هوای دستی.
func TestOilCustomClimatePercent(t *testing.T) {
	in := oilInput()
	in.ClimatePercent = 30
	slip := ComputeOilPayslip(in, DefaultOilPayParams())
	if slip.Climate != 6_000_000 || slip.AllowanceBase != 26_000_000 {
		t.Fatalf("درصد دستی بدی آب و هوا نادرست اعمال شد: climate=%d base=%d", slip.Climate, slip.AllowanceBase)
	}
}

// TestOilZeroBaseIsSafe: پایهٔ صفر نباید خطا بدهد.
func TestOilZeroBaseIsSafe(t *testing.T) {
	slip := ComputeOilPayslip(OilPayInput{}, DefaultOilPayParams())
	if slip.Gross != 0 || slip.Net != 0 || slip.Allowances != 0 {
		t.Fatalf("پایهٔ صفر باید فیش صفر بدهد: %+v", slip)
	}
}

// TestOilCatalogMatchesApprovedTable: فهرست اقلام همان جدول مصوب سازمان است.
func TestOilCatalogMatchesApprovedTable(t *testing.T) {
	params := DefaultOilPayParams()
	want := map[string]float64{
		"deprivation": 27.5, "special": 3, "timeConsumed": 16.5, "workshop": 10,
		"attractionSpecialty": 27, "production": 30, "shiftWorker": 35,
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
