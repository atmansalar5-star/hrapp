//! زنجیرهٔ محاسبهٔ حقوق: ناخالص ← بیمهٔ کارمند/کارفرما ← مالیات پله‌ای ← خالص.
//!
//! همهٔ پارامترها (نرخ‌ها، سقف بیمه، معافیت و پله‌های مالیات) از ماژول تنظیمات و
//! نسخه‌دار می‌آیند؛ این crate هیچ عدد قانونی را در خود ثابت نمی‌کند. به همین دلیل
//! تغییر قانون مالیات فقط یک رکورد پارامتر است، نه انتشار نسخهٔ جدید برنامه.
//!
//! محاسبات مالی با عدد صحیح ریال انجام می‌شود و گرد کردن نیم‌به‌بالا است تا خروجی
//! کلاینت و سرور بیت‌به‌بیت یکسان بماند.

use crate::round_half_up;
use serde::{Deserialize, Serialize};

/// یک پلهٔ مالیاتی: درآمد تا سقف `upto` با نرخ `rate`. سقف صفر یعنی بی‌نهایت.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaxBracket {
    pub upto: i64,
    pub rate: f64,
}

/// پارامترهای محاسبهٔ حقوق (از ماژول تنظیمات، نسخه‌دار و قابل بازبینی).
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PayrollParams {
    pub insurance_employee_rate: f64,
    pub insurance_employer_rate: f64,
    /// سقف مبلغ مشمول بیمه؛ صفر یعنی بی‌سقف.
    pub insurance_ceiling: i64,
    pub housing_allowance: i64,
    pub child_allowance_per_child: i64,
    /// سقف معافیت ماهانهٔ مالیات.
    pub tax_exemption: i64,
    pub tax_brackets: Vec<TaxBracket>,
}

/// ورودی محاسبهٔ یک کارمند در یک دوره.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PayrollInput {
    pub base_salary: i64,
    #[serde(default)]
    pub children: u32,
    #[serde(default)]
    pub overtime_hours: i64,
    #[serde(default)]
    pub overtime_hourly_rate: i64,
    #[serde(default)]
    pub other_deductions: i64,
}

/// خروجی محاسبهٔ فیش حقوقی.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PayrollResult {
    pub overtime_pay: i64,
    pub gross: i64,
    pub insurance_base: i64,
    pub insurance_employee: i64,
    pub insurance_employer: i64,
    pub taxable_income: i64,
    pub tax: i64,
    pub net: i64,
}

/// مالیات پله‌ای بر درآمد مشمول.
pub fn tax_for(income: i64, brackets: &[TaxBracket]) -> i64 {
    if income <= 0 {
        return 0;
    }
    let mut tax = 0.0;
    let mut prev = 0i64;
    for b in brackets {
        if income <= prev {
            break;
        }
        let upper = if b.upto > 0 { b.upto } else { income };
        let slice = income.min(upper) - prev;
        if slice > 0 {
            tax += slice as f64 * b.rate;
        }
        prev = upper;
    }
    round_half_up(tax)
}

/// محاسبهٔ کامل یک فیش.
pub fn compute(input: &PayrollInput, params: &PayrollParams) -> PayrollResult {
    let overtime_pay = input.overtime_hours * input.overtime_hourly_rate;
    let children_pay = input.children as i64 * params.child_allowance_per_child;
    let gross = input.base_salary + params.housing_allowance + children_pay + overtime_pay;

    let insurance_base = if params.insurance_ceiling > 0 && gross > params.insurance_ceiling {
        params.insurance_ceiling
    } else {
        gross
    };
    let insurance_employee = round_half_up(insurance_base as f64 * params.insurance_employee_rate);
    let insurance_employer = round_half_up(insurance_base as f64 * params.insurance_employer_rate);

    let taxable_income = (gross - insurance_employee - params.tax_exemption).max(0);
    let tax = tax_for(taxable_income, &params.tax_brackets);
    let net = gross - insurance_employee - tax - input.other_deductions;

    PayrollResult {
        overtime_pay,
        gross,
        insurance_base,
        insurance_employee,
        insurance_employer,
        taxable_income,
        tax,
        net,
    }
}

/// محاسبهٔ جمع یک دورهٔ حقوق برای همهٔ کارکنان (خروجی صفحهٔ «پردازش حقوق»).
#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PayrollTotals {
    pub count: u32,
    pub gross: i64,
    pub tax: i64,
    pub insurance_employee: i64,
    pub insurance_employer: i64,
    pub net: i64,
}

/// جمع یک دورهٔ حقوق.
pub fn totals(results: &[PayrollResult]) -> PayrollTotals {
    let mut t = PayrollTotals { count: results.len() as u32, ..Default::default() };
    for r in results {
        t.gross += r.gross;
        t.tax += r.tax;
        t.insurance_employee += r.insurance_employee;
        t.insurance_employer += r.insurance_employer;
        t.net += r.net;
    }
    t
}

#[cfg(test)]
mod tests {
    use super::*;

    fn params() -> PayrollParams {
        PayrollParams {
            insurance_employee_rate: 0.07,
            insurance_employer_rate: 0.23,
            insurance_ceiling: 200_000_000,
            housing_allowance: 9_000_000,
            child_allowance_per_child: 7_000_000,
            tax_exemption: 24_000_000,
            tax_brackets: vec![
                TaxBracket { upto: 50_000_000, rate: 0.10 },
                TaxBracket { upto: 100_000_000, rate: 0.15 },
                TaxBracket { upto: 200_000_000, rate: 0.20 },
                TaxBracket { upto: 0, rate: 0.30 },
            ],
        }
    }

    #[test]
    fn matches_golden_fixture_cases() {
        // مقادیر مورد انتظار همان core/testdata/conformance/payroll.json است.
        let cases: [(PayrollInput, PayrollResult); 4] = [
            (
                PayrollInput { base_salary: 52_000_000, children: 0, overtime_hours: 0, overtime_hourly_rate: 0, other_deductions: 0 },
                PayrollResult { overtime_pay: 0, gross: 61_000_000, insurance_base: 61_000_000, insurance_employee: 4_270_000, insurance_employer: 14_030_000, taxable_income: 32_730_000, tax: 3_273_000, net: 53_457_000 },
            ),
            (
                PayrollInput { base_salary: 34_000_000, children: 2, overtime_hours: 12, overtime_hourly_rate: 250_000, other_deductions: 1_500_000 },
                PayrollResult { overtime_pay: 3_000_000, gross: 60_000_000, insurance_base: 60_000_000, insurance_employee: 4_200_000, insurance_employer: 13_800_000, taxable_income: 31_800_000, tax: 3_180_000, net: 51_120_000 },
            ),
            (
                PayrollInput { base_salary: 250_000_000, children: 1, overtime_hours: 0, overtime_hourly_rate: 0, other_deductions: 0 },
                PayrollResult { overtime_pay: 0, gross: 266_000_000, insurance_base: 200_000_000, insurance_employee: 14_000_000, insurance_employer: 46_000_000, taxable_income: 228_000_000, tax: 40_900_000, net: 211_100_000 },
            ),
            (
                PayrollInput { base_salary: 15_000_000, children: 0, overtime_hours: 0, overtime_hourly_rate: 0, other_deductions: 0 },
                PayrollResult { overtime_pay: 0, gross: 24_000_000, insurance_base: 24_000_000, insurance_employee: 1_680_000, insurance_employer: 5_520_000, taxable_income: 0, tax: 0, net: 22_320_000 },
            ),
        ];
        for (input, expected) in cases {
            assert_eq!(compute(&input, &params()), expected, "حقوق پایه {}", input.base_salary);
        }
    }

    #[test]
    fn tax_brackets_are_progressive() {
        let b = params().tax_brackets;
        assert_eq!(tax_for(0, &b), 0);
        assert_eq!(tax_for(40_000_000, &b), 4_000_000);
        assert_eq!(tax_for(50_000_000, &b), 5_000_000);
        assert_eq!(tax_for(100_000_000, &b), 12_500_000);
        assert_eq!(tax_for(200_000_000, &b), 32_500_000);
        assert_eq!(tax_for(300_000_000, &b), 62_500_000);
    }

    #[test]
    fn period_totals_aggregate() {
        let inputs = vec![
            compute(&PayrollInput { base_salary: 52_000_000, children: 0, overtime_hours: 0, overtime_hourly_rate: 0, other_deductions: 0 }, &params()),
            compute(&PayrollInput { base_salary: 34_000_000, children: 2, overtime_hours: 12, overtime_hourly_rate: 250_000, other_deductions: 1_500_000 }, &params()),
        ];
        let t = totals(&inputs);
        assert_eq!(t.count, 2);
        assert_eq!(t.gross, 121_000_000);
        assert_eq!(t.tax, 6_453_000);
        assert_eq!(t.net, 104_577_000);
    }
}
