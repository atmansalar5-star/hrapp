// آزمون قاعدهٔ «formula» در موتور قواعد.

package rules

import (
	"math"
	"testing"
)

func TestEvalFormula(t *testing.T) {
	vars := map[string]float64{
		"overtimeHours":   10,
		"hourlyRate":      700_000,
		"overtime.factor": 1.4,
		"base":            120_000_000,
	}
	cases := []struct {
		name string
		expr string
		want float64
	}{
		{"اعداد و تقدم عملگر", "2 + 3 * 4", 14},
		{"پرانتز", "(2 + 3) * 4", 20},
		{"منفی یکانی", "-5 + 10", 5},
		{"توان راست‌به‌چپ", "2 ^ 3 ^ 2", 512},
		{"متغیرهای نقطه‌دار", "overtimeHours * hourlyRate * overtime.factor", 9_800_000},
		{"تقسیم", "base / 4", 30_000_000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EvalFormula(tc.expr, vars)
			if err != nil {
				t.Fatalf("EvalFormula(%q) خطا داد: %v", tc.expr, err)
			}
			if math.Abs(got-tc.want) > 1e-6 {
				t.Fatalf("EvalFormula(%q) = %v، انتظار %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestEvalFormulaErrors(t *testing.T) {
	for _, expr := range []string{
		"taxableIncome * taxRate (پله‌ای)", // متغیر ناشناخته و متن توصیفی
		"unknownVar + 1",
		"10 / 0",
		"2 +",
		"(1 + 2",
	} {
		if _, err := EvalFormula(expr, map[string]float64{"taxableIncome": 1}); err == nil {
			t.Fatalf("EvalFormula(%q) باید خطا بدهد", expr)
		}
	}
}

func TestFormulaVariables(t *testing.T) {
	got := FormulaVariables("a + b.a * 2 - a")
	want := []string{"a", "b.a"}
	if len(got) != len(want) {
		t.Fatalf("متغیرها: %v، انتظار %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("متغیرها: %v، انتظار %v", got, want)
		}
	}
}

func TestComputeElementsResolvesFormula(t *testing.T) {
	params := DefaultPayrollParams()
	input := PayrollInput{BaseSalary: 120_000_000, OvertimeHours: 10, OvertimeHourlyRate: 700_000}
	result := ComputePayroll(input, params)

	vars := BaseElementVars(input, result)
	for k, v := range ParamsVars(params) {
		vars[k] = v
	}
	vars["overtime.factor"] = 1.4
	vars["friday.factor"] = 1.4
	vars["allowance.badClimate"] = 0.25

	// مزد مبنا = مزد شغل + مزد رتبه؛ فوق‌العادهٔ درصدی روی همین مبنا حساب می‌شود.
	wages := WageComponents{Job: 120_000_000, Rank: 10_000_000}
	for k, v := range wages.Vars() {
		vars[k] = v
	}
	vars["fridayHours"] = 4

	env := ElementEnv{
		Bases:     BaseElementBases(input, params, result, 0, wages),
		Vars:      vars,
		Canonical: CanonicalElementAmounts(result),
	}

	lines := ComputeElements([]PayElement{
		{Code: "OVERTIME", TitleFa: "اضافه‌کاری", Category: "earning", Calc: ElementCalcFormula, Formula: "overtimeHours * hourlyRate * overtime.factor"},
		{Code: "TAX", TitleFa: "مالیات", Category: "deduction", Calc: ElementCalcFormula, Formula: "taxableIncome * taxRate (پله‌ای)"},
		{Code: "HOUSING", TitleFa: "حق مسکن", Category: "earning", Calc: ElementCalcFixed, Rate: "allowance.housing"},
		{Code: "BAD_CLIMATE", TitleFa: "بدی آب و هوا", Category: "earning", Calc: ElementCalcPercent, Rate: "allowance.badClimate", Basis: "mazd"},
		{Code: "FRIDAY", TitleFa: "جمعه‌کاری", Category: "earning", Calc: ElementCalcFormula, Formula: "fridayHours * hourlyRate * friday.factor"},
	}, env)

	// مزد مبنا = ۱۳۰٬۰۰۰٬۰۰۰؛ پس بدی آب و هوا = ۲۵٪ = ۳۲٬۵۰۰٬۰۰۰.
	want := map[string]float64{"BAD_CLIMATE": 32_500_000, "FRIDAY": 3_920_000}
	for _, line := range lines {
		if !line.Resolved {
			t.Fatalf("قلم %s حل نشد: %s", line.Code, line.DetailFa)
		}
		if expect, ok := want[line.Code]; ok && math.Abs(line.Amount-expect) > 1e-6 {
			t.Fatalf("قلم %s = %v، انتظار %v", line.Code, line.Amount, expect)
		}
	}
}
