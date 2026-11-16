// آزمون قاعدهٔ «parity» در موتور قواعد.

package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"hrms/internal/platform/calendar"
)

// fixturesDir مسیر fixtureهای طلایی که هستهٔ Rust تولید می‌کند.
func fixturesDir() string {
	return filepath.Join("..", "..", "..", "core", "testdata", "conformance")
}

func readFixture(t *testing.T, name string, dst any) {
	t.Helper()
	path := filepath.Join(fixturesDir(), name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("خواندن fixture %s ناموفق: %v\nراهنما: هستهٔ Rust منبع این فایل‌ها است (hrms-conformance).", path, err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		t.Fatalf("fixture %s قابل تجزیه نیست: %v", path, err)
	}
}

// TestPayrollParityWithGoldenFixtures همان دروازهٔ «یک قاعده، دو میزبان» است:
// اعداد سرور Go باید دقیقاً با خروجی هستهٔ Rust یکی باشد.
func TestPayrollParityWithGoldenFixtures(t *testing.T) {
	var fixture struct {
		Params PayrollParams `json:"params"`
		Cases  []struct {
			Name               string        `json:"name"`
			BaseSalary         int64         `json:"baseSalary"`
			Children           int           `json:"children"`
			OvertimeHours      int64         `json:"overtimeHours"`
			OvertimeHourlyRate int64         `json:"overtimeHourlyRate"`
			OtherDeductions    int64         `json:"otherDeductions"`
			Expected           PayrollResult `json:"expected"`
		} `json:"cases"`
	}
	readFixture(t, "payroll.json", &fixture)

	if len(fixture.Cases) == 0 {
		t.Fatal("fixture حقوق هیچ موردی ندارد")
	}
	for _, c := range fixture.Cases {
		got := ComputePayroll(PayrollInput{
			BaseSalary:         c.BaseSalary,
			Children:           c.Children,
			OvertimeHours:      c.OvertimeHours,
			OvertimeHourlyRate: c.OvertimeHourlyRate,
			OtherDeductions:    c.OtherDeductions,
		}, fixture.Params)

		if got != c.Expected {
			t.Errorf("مورد «%s»:\n  سرور  = %+v\n  هستهٔ Rust = %+v", c.Name, got, c.Expected)
		}
	}
}

func TestLeaveParityWithGoldenFixtures(t *testing.T) {
	var fixture struct {
		Params LeaveParams `json:"params"`
		Cases  []struct {
			Name         string  `json:"name"`
			From         string  `json:"from"`
			To           string  `json:"to"`
			Sandwich     bool    `json:"sandwich"`
			Balance      float64 `json:"balance"`
			Days         float64 `json:"days"`
			Eligible     bool    `json:"eligible"`
			BalanceAfter float64 `json:"balanceAfter"`
		} `json:"cases"`
		AccrualCases []struct {
			MonthsWorked  int     `json:"monthsWorked"`
			RawAccrual    float64 `json:"rawAccrual"`
			CappedAccrual float64 `json:"cappedAccrual"`
		} `json:"accrualCases"`
		CarryCases []struct {
			Balance  float64 `json:"balance"`
			CarryCap float64 `json:"carryCap"`
			Carried  float64 `json:"carried"`
		} `json:"carryCases"`
	}
	readFixture(t, "leave.json", &fixture)

	for _, c := range fixture.Cases {
		from, err := calendar.Parse(c.From)
		if err != nil {
			t.Fatalf("تاریخ %s نامعتبر: %v", c.From, err)
		}
		to, err := calendar.Parse(c.To)
		if err != nil {
			t.Fatalf("تاریخ %s نامعتبر: %v", c.To, err)
		}

		days, err := CountLeaveDays(from, to, fixture.Params, c.Sandwich)
		if err != nil {
			t.Fatalf("شمارش روز مرخصی %s ناموفق: %v", c.Name, err)
		}
		if days != c.Days {
			t.Errorf("مورد «%s»: روزهای سرور=%v ولی هستهٔ Rust=%v", c.Name, days, c.Days)
		}
		if got := IsEligible(c.Balance, days); got != c.Eligible {
			t.Errorf("مورد «%s»: استحقاق سرور=%v ولی هستهٔ Rust=%v", c.Name, got, c.Eligible)
		}
		after, ok := BalanceAfter(c.Balance, days)
		if !ok {
			after = c.Balance
		}
		if after != c.BalanceAfter {
			t.Errorf("مورد «%s»: مانده سرور=%v ولی هستهٔ Rust=%v", c.Name, after, c.BalanceAfter)
		}
	}

	for _, c := range fixture.AccrualCases {
		if got := RawAccrual(c.MonthsWorked, fixture.Params); got != c.RawAccrual {
			t.Errorf("تعهدی خام %d ماه: سرور=%v هسته=%v", c.MonthsWorked, got, c.RawAccrual)
		}
		if got := CappedAccrual(c.MonthsWorked, fixture.Params); got != c.CappedAccrual {
			t.Errorf("تعهدی سقف‌دار %d ماه: سرور=%v هسته=%v", c.MonthsWorked, got, c.CappedAccrual)
		}
	}

	for _, c := range fixture.CarryCases {
		if got := CarryForward(c.Balance, c.CarryCap); got != c.Carried {
			t.Errorf("انتقال مانده %v: سرور=%v هسته=%v", c.Balance, got, c.Carried)
		}
	}
}

func TestKpiParityWithGoldenFixtures(t *testing.T) {
	var fixture struct {
		Cases []struct {
			Name          string    `json:"name"`
			Items         []KpiItem `json:"items"`
			TotalWeight   float64   `json:"totalWeight"`
			WeightedScore float64   `json:"weightedScore"`
			GradeFa       string    `json:"gradeFa"`
		} `json:"cases"`
	}
	readFixture(t, "kpi.json", &fixture)

	for _, c := range fixture.Cases {
		outcome := Evaluate(c.Items)
		if outcome.TotalWeight != c.TotalWeight {
			t.Errorf("مورد «%s»: جمع وزن سرور=%v هسته=%v", c.Name, outcome.TotalWeight, c.TotalWeight)
		}
		if outcome.WeightedScore != c.WeightedScore {
			t.Errorf("مورد «%s»: نمرهٔ موزون سرور=%v هسته=%v", c.Name, outcome.WeightedScore, c.WeightedScore)
		}
		if outcome.GradeFa != c.GradeFa {
			t.Errorf("مورد «%s»: رتبهٔ سرور=%q هسته=%q", c.Name, outcome.GradeFa, c.GradeFa)
		}
	}
}

func TestPayrollTotalsAggregateAcrossPeriod(t *testing.T) {
	params := PayrollParams{
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
	results := []PayrollResult{
		ComputePayroll(PayrollInput{BaseSalary: 52_000_000}, params),
		ComputePayroll(PayrollInput{BaseSalary: 34_000_000, Children: 2, OvertimeHours: 12, OvertimeHourlyRate: 250_000, OtherDeductions: 1_500_000}, params),
	}
	got := Totals(results)
	if got.Count != 2 || got.Gross != 121_000_000 || got.Tax != 6_453_000 || got.Net != 104_577_000 {
		t.Errorf("جمع دوره نادرست است: %+v", got)
	}
}

func TestRoundingMatchesRustBehaviour(t *testing.T) {
	if RoundHalfUp(4270000.5) != 4270001 {
		t.Errorf("گرد کردن نیم‌به‌بالا نادرست است: %d", RoundHalfUp(4270000.5))
	}
	if RoundHalfUp(-1.5) != -2 {
		t.Errorf("گرد کردن منفی نادرست است: %d", RoundHalfUp(-1.5))
	}
	if TaxFor(100_000_000, []TaxBracket{{Upto: 50_000_000, Rate: 0.10}, {Upto: 100_000_000, Rate: 0.15}}) != 12_500_000 {
		t.Errorf("مالیات پله‌ای نادرست است: %d", TaxFor(100_000_000, []TaxBracket{{Upto: 50_000_000, Rate: 0.10}, {Upto: 100_000_000, Rate: 0.15}}))
	}
}

// TestTaxForClosedLastBracketWithoutOpenEnded پلهٔ آخر «بسته» را می‌سنجد: درآمد بالای سقف
// آخرین پله نباید بی‌مالیات بماند (این تابع میان هر سه نظام پرداخت مشترک است و همان رفتار
// در core/crates/hrms-rules/src/payroll.rs هم پیاده شده است).
func TestTaxForClosedLastBracketWithoutOpenEnded(t *testing.T) {
	brackets := []TaxBracket{{Upto: 50_000_000, Rate: 0.10}, {Upto: 200_000_000, Rate: 0.15}}
	// ۵۰م × ۱۰٪ = ۵م و ۱۵۰م × ۱۵٪ = ۲۲٫۵م ⇒ ۲۷٫۵م برای درآمد ۲۰۰م
	if got := TaxFor(200_000_000, brackets); got != 27_500_000 {
		t.Fatalf("مالیات درآمد ۲۰۰م باید ۲۷٬۵۰۰٬۰۰۰ باشد، شد %d", got)
	}
	// درآمد بالای سقف آخر: ۱۰۰م باقی‌مانده هم با نرخ پلهٔ آخر مشمول می‌شود ⇒ ۴۲٫۵م
	if got := TaxFor(300_000_000, brackets); got != 42_500_000 {
		t.Fatalf("درآمد بالای سقف آخر باید مشمول بماند (۴۲٬۵۰۰٬۰۰۰)، شد %d", got)
	}
	// فهرست خالی و درآمد نامثبت
	if got := TaxFor(300_000_000, nil); got != 0 {
		t.Fatalf("بدون پله، مالیات باید صفر باشد: %d", got)
	}
	if got := TaxFor(0, brackets); got != 0 {
		t.Fatalf("درآمد صفر مالیات ندارد: %d", got)
	}
	// پلهٔ بی‌سقف (`Upto=0`) همان رفتار پیشین را نگه می‌دارد
	open := []TaxBracket{{Upto: 50_000_000, Rate: 0.10}, {Upto: 0, Rate: 0.15}}
	if got := TaxFor(300_000_000, open); got != 42_500_000 {
		t.Fatalf("با پلهٔ بی‌سقف باید ۴۲٬۵۰۰٬۰۰۰ باشد، شد %d", got)
	}
}
