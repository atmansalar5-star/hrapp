//! قواعد مرخصی: شمارش روز، ساندویچ، مانده، ذخیره‌سازی و استحقاق.
//!
//! مبنا: روز کاری در تقویم سازمان (تعطیل هفتگی و تعطیلات رسمی)، به‌همراه قاعدهٔ
//! «ساندویچ» که تعطیل واقع‌شده در میانهٔ مرخصی را هم جزء مرخصی می‌شمارد.

use crate::round2;
use hrms_calendar::{CalendarError, JalaliDate, WorkCalendar};
use serde::{Deserialize, Serialize};

/// پارامترهای مرخصی — از ماژول تنظیمات می‌آید و آزمایشی (hard-coded) نیست.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LeaveParams {
    /// روزهای مرخصی تعهدی در هر ماه کارکرد.
    pub accrual_per_month: f64,
    /// سقف سالانهٔ تعهدی.
    pub annual_cap: f64,
    /// سقف انتقال مانده به سال بعد.
    pub carry_cap: f64,
}

impl Default for LeaveParams {
    fn default() -> Self {
        // مقادیر پیش‌فرض قانون کار: ۲۶ روز سالانه (۲‌٫۱۷ ماهانه) با انتقال حداکثر ۹ روز.
        LeaveParams { accrual_per_month: 2.5, annual_cap: 30.0, carry_cap: 9.0 }
    }
}

/// شمار روزهای مرخصی یک بازه.
///
/// * `sandwich = false` → تنها روزهای کاری شمرده می‌شوند.
/// * `sandwich = true` → از نخستین تا واپسین روز کاری؛ تعطیل‌های میانی هم جزء مرخصی‌اند.
pub fn count_leave_days(
    from: &JalaliDate,
    to: &JalaliDate,
    calendar: &WorkCalendar,
    sandwich: bool,
) -> Result<f64, CalendarError> {
    let days = if sandwich {
        calendar.sandwich_days(from, to)?
    } else {
        calendar.working_days(from, to)?
    };
    Ok(days as f64)
}

/// تعهدی خام پیش از اعمال سقف سالانه.
pub fn raw_accrual(months_worked: u32, params: &LeaveParams) -> f64 {
    round2(months_worked as f64 * params.accrual_per_month)
}

/// تعهدی پس از اعمال سقف سالانه.
pub fn capped_accrual(months_worked: u32, params: &LeaveParams) -> f64 {
    let raw = raw_accrual(months_worked, params);
    if raw > params.annual_cap {
        params.annual_cap
    } else {
        raw
    }
}

/// انتقال مانده به سال بعد با اعمال سقف انتقال.
pub fn carry_forward(balance: f64, carry_cap: f64) -> f64 {
    round2(balance.min(carry_cap).max(0.0))
}

/// آیا مانده برای درخواست کافی است.
pub fn is_eligible(balance: f64, requested: f64) -> bool {
    balance >= requested
}

/// مانده پس از تأیید درخواست؛ در صورت ناکافی بودن `None`.
pub fn balance_after(balance: f64, requested: f64) -> Option<f64> {
    if is_eligible(balance, requested) {
        Some(round2(balance - requested))
    } else {
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn cal() -> WorkCalendar {
        let mut c = WorkCalendar::default_iran();
        c.holidays.insert("1404/07/02".into(), "تعطیل رسمی نمونه".into());
        c
    }

    #[test]
    fn counts_match_golden_fixture() {
        // همان مقادیری که در core/testdata/conformance/leave.json ثابت شده است.
        let cases: [(&str, &str, bool, f64); 5] = [
            ("1404/07/14", "1404/07/16", false, 3.0),
            ("1404/07/05", "1404/07/11", false, 6.0),
            ("1404/07/05", "1404/07/11", true, 6.0),
            ("1404/07/02", "1404/07/02", false, 0.0),
            ("1404/07/01", "1404/07/10", false, 8.0),
        ];
        for (from, to, sandwich, expected) in cases {
            let f = JalaliDate::parse(from).unwrap();
            let t = JalaliDate::parse(to).unwrap();
            assert_eq!(count_leave_days(&f, &t, &cal(), sandwich).unwrap(), expected, "{from}..{to}");
        }
    }

    #[test]
    fn accrual_and_carry() {
        let p = LeaveParams::default();
        assert_eq!(raw_accrual(0, &p), 0.0);
        assert_eq!(raw_accrual(8, &p), 20.0);
        assert_eq!(capped_accrual(18, &p), 30.0, "سقف سالانه اعمال می‌شود");
        assert_eq!(capped_accrual(30, &p), 30.0);
        assert_eq!(carry_forward(14.5, p.carry_cap), 9.0);
        assert_eq!(carry_forward(6.0, p.carry_cap), 6.0);
        assert_eq!(carry_forward(-3.0, p.carry_cap), 0.0);
    }

    #[test]
    fn eligibility_blocks_insufficient_balance() {
        assert!(is_eligible(12.0, 8.0));
        assert!(!is_eligible(2.0, 8.0));
        assert_eq!(balance_after(12.0, 3.0), Some(9.0));
        assert_eq!(balance_after(2.0, 8.0), None);
    }
}
