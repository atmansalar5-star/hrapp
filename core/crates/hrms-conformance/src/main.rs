//! ابزار تطبیق قواعد («یک قاعده، دو میزبان»).
//!
//! این CLI تنها مرجع تولید `core/testdata/conformance/*.json` است. سرور Go همان
//! فایل‌ها را می‌خواند و پیاده‌سازی خودش را با آن‌ها می‌سنجد، پس اگر قاعده‌ای در Rust
//! و Go از هم جدا بیفتد، CI شکست می‌خورد.
//!
//! استعمال:
//! * `hrms-conformance`          → بازبینی: محاسبهٔ دوباره و مقایسه با فایل‌های کامیت‌شده
//! * `hrms-conformance --write`  → بازتولید فایل‌ها پس از تغییر آگاهانهٔ قواعد
//!
//! مقایسه معنایی (عددی با رواداری ۱e-۶) است تا قالب‌بندی JSON مرجع مقایسه نباشد.

use hrms_calendar::{JalaliDate, WorkCalendar};
use hrms_rules::kpi::KpiItem;
use hrms_rules::leave::LeaveParams;
use hrms_rules::payroll::{compute, PayrollInput, PayrollParams, TaxBracket};
use serde_json::{json, Value};
use std::path::{Path, PathBuf};
use std::process::ExitCode;

/// مسیر پوشهٔ fixtureها نسبت به این crate.
fn fixtures_dir() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/conformance")
}

/// تقویم کاری fixture: جمعه تعطیل + یک تعطیل رسمی نمونه.
fn fixture_calendar() -> WorkCalendar {
    let mut cal = WorkCalendar::default_iran();
    cal.holidays.insert("1404/07/02".to_string(), "تعطیل رسمی نمونه".to_string());
    cal
}

/// پارامترهای حقوق fixture (همان مقادیر و ترتیبی که در فایل مرجع است).
fn fixture_payroll_params() -> PayrollParams {
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

fn gregorian_string(d: &JalaliDate) -> String {
    let (gy, gm, gd) = d.to_gregorian().expect("تاریخ معتبر");
    format!("{gy:04}-{gm:02}-{gd:02}")
}

/// fixture تقویم شمسی و تقویم کاری.
fn build_calendar() -> Value {
    let years: Vec<Value> = [1398, 1399, 1400, 1401, 1402, 1403, 1404, 1405, 1408]
        .iter()
        .map(|&jy| {
            let first = JalaliDate::new(jy, 1, 1).expect("سال معتبر");
            json!({
                "year": jy,
                "leap": JalaliDate::is_leap(jy).expect("سال معتبر"),
                "esfandLength": JalaliDate::month_length(jy, 12).expect("ماه معتبر"),
                "farvardin1Gregorian": gregorian_string(&first),
            })
        })
        .collect();

    let conversions: Vec<Value> = [(1404, 1, 1), (1404, 6, 31), (1404, 7, 1), (1404, 7, 22), (1404, 12, 29), (1403, 12, 30), (1400, 12, 29), (1399, 12, 30), (1398, 7, 14)]
        .iter()
        .map(|&(jy, jm, jd)| {
            let d = JalaliDate::new(jy, jm, jd).expect("تاریخ معتبر");
            json!({
                "jalali": d.to_string(),
                "gregorian": gregorian_string(&d),
                "jdn": d.day_number().expect("شمارهٔ روز"),
                "weekdayIndex": d.weekday_index().expect("روز هفته"),
                "weekdayFa": d.weekday_fa().expect("روز هفته"),
                "monthLabelFa": d.month_label_fa().expect("برچسب ماه"),
            })
        })
        .collect();

    let month_lengths: Vec<Value> = [(1404, 1), (1404, 6), (1404, 7), (1404, 12), (1403, 12), (1400, 12)]
        .iter()
        .map(|&(jy, jm)| {
            json!({
                "year": jy,
                "month": jm,
                "days": JalaliDate::month_length(jy, jm).expect("ماه معتبر"),
            })
        })
        .collect();

    let windows: [(&str, &str, &str, Vec<u32>, &str); 4] = [
        ("بازهٔ یک هفته‌ای با یک جمعه", "1404/07/01", "1404/07/05", vec![6], "none"),
        ("بازه‌ای که با تعطیل هفتگی آغاز می‌شود", "1404/07/03", "1404/07/06", vec![6], "none"),
        ("بازه با یک تعطیل رسمی در میانه", "1404/07/01", "1404/07/05", vec![6], "holiday"),
        ("کل ماه مهر با پنجشنبه و جمعه تعطیل", "1404/07/01", "1404/07/30", vec![5, 6], "none"),
    ];

    let work_windows: Vec<Value> = windows
        .iter()
        .map(|(name, from, to, weekend, holiday_mode)| {
            let f = JalaliDate::parse(from).expect("تاریخ معتبر");
            let t = JalaliDate::parse(to).expect("تاریخ معتبر");
            let mut cal = WorkCalendar { weekend: weekend.clone(), holidays: Default::default() };
            let holidays = if *holiday_mode == "holiday" {
                cal.holidays.insert("1404/07/02".to_string(), "تعطیل رسمی نمونه".to_string());
                json!({ "1404/07/02": "تعطیل رسمی نمونه" })
            } else {
                json!({})
            };
            json!({
                "name": name,
                "from": from,
                "to": to,
                "weekend": weekend,
                "holidays": holidays,
                "workingDays": cal.working_days(&f, &t).expect("بازهٔ معتبر"),
                "sandwichDays": cal.sandwich_days(&f, &t).expect("بازهٔ معتبر"),
            })
        })
        .collect();

    json!({
        "id": "calendar",
        "description": "صحت تبدیل تقویم شمسی/میلادی، کبیسه، طول ماه و شمارش روز کاری",
        "years": years,
        "conversions": conversions,
        "monthLengths": month_lengths,
        "workWindows": work_windows,
    })
}

/// fixture قواعد مرخصی.
fn build_leave() -> Value {
    let cal = fixture_calendar();
    let params = LeaveParams::default();

    let cases: [(&str, &str, &str, bool, f64); 5] = [
        ("مرخصی سه‌روزه با یک جمعه", "1404/07/14", "1404/07/16", false, 12.0),
        ("مرخصی هفت‌روزه شامل جمعه", "1404/07/05", "1404/07/11", false, 12.0),
        ("همان بازه با قاعدهٔ ساندویچ", "1404/07/05", "1404/07/11", true, 12.0),
        ("مرخصی یک‌روزه روی تعطیل رسمی", "1404/07/02", "1404/07/02", false, 5.0),
        ("ماندهٔ ناکافی", "1404/07/01", "1404/07/10", false, 2.0),
    ];

    let cases: Vec<Value> = cases
        .iter()
        .map(|(name, from, to, sandwich, balance)| {
            let f = JalaliDate::parse(from).expect("تاریخ معتبر");
            let t = JalaliDate::parse(to).expect("تاریخ معتبر");
            let days = hrms_rules::leave::count_leave_days(&f, &t, &cal, *sandwich).expect("بازهٔ معتبر");
            let eligible = hrms_rules::leave::is_eligible(*balance, days);
            let after = hrms_rules::leave::balance_after(*balance, days).unwrap_or(*balance);
            json!({
                "name": name,
                "from": from,
                "to": to,
                "sandwich": sandwich,
                "balance": balance,
                "days": days,
                "eligible": eligible,
                "balanceAfter": after,
            })
        })
        .collect();

    let accrual_cases: Vec<Value> = [0u32, 1, 6, 12, 18, 30]
        .iter()
        .map(|&months| {
            json!({
                "monthsWorked": months,
                "rawAccrual": hrms_rules::leave::raw_accrual(months, &params),
                "cappedAccrual": hrms_rules::leave::capped_accrual(months, &params),
            })
        })
        .collect();

    let carry_cases: Vec<Value> = [14.5, 6.0, 9.0, 0.0]
        .iter()
        .map(|&balance| {
            json!({
                "balance": balance,
                "carryCap": params.carry_cap,
                "carried": hrms_rules::leave::carry_forward(balance, params.carry_cap),
            })
        })
        .collect();

    let holidays: Value = json!({ "1404/07/02": "تعطیل رسمی نمونه" });

    json!({
        "id": "leave",
        "description": "قواعد شمارش روز مرخصی، ساندویچ، مانده و ذخیره‌سازی سالانه",
        "params": {
            "accrualPerMonth": params.accrual_per_month,
            "annualCap": params.annual_cap,
            "carryCap": params.carry_cap,
            "weekend": cal.weekend,
            "holidays": holidays,
        },
        "cases": cases,
        "accrualCases": accrual_cases,
        "carryCases": carry_cases,
    })
}

/// fixture محاسبهٔ حقوق.
fn build_payroll() -> Value {
    let params = fixture_payroll_params();
    let inputs: [(&str, i64, u32, i64, i64, i64); 4] = [
        ("کارمند بدون اضافه‌کاری", 52_000_000, 0, 0, 0, 0),
        ("کارمند با دو فرزند و اضافه‌کاری", 34_000_000, 2, 12, 250_000, 1_500_000),
        ("حقوق بالای سقف بیمه", 250_000_000, 1, 0, 0, 0),
        ("درآمد زیر سقف معافیت", 15_000_000, 0, 0, 0, 0),
    ];

    let cases: Vec<Value> = inputs
        .iter()
        .map(|(name, base_salary, children, overtime_hours, overtime_hourly_rate, other_deductions)| {
            let input = PayrollInput {
                base_salary: *base_salary,
                children: *children,
                overtime_hours: *overtime_hours,
                overtime_hourly_rate: *overtime_hourly_rate,
                other_deductions: *other_deductions,
            };
            let result = compute(&input, &params);
            json!({
                "name": name,
                "baseSalary": base_salary,
                "children": children,
                "overtimeHours": overtime_hours,
                "overtimeHourlyRate": overtime_hourly_rate,
                "otherDeductions": other_deductions,
                "expected": serde_json::to_value(&result).expect("خروجی قابل سریال‌سازی"),
            })
        })
        .collect();

    json!({
        "id": "payroll",
        "description": "زنجیرهٔ محاسبهٔ حقوق: ناخالص ← بیمه ← مالیات پله‌ای ← خالص",
        "params": serde_json::to_value(&params).expect("پارامترها قابل سریال‌سازی"),
        "cases": cases,
    })
}

/// fixture ارزیابی عملکرد.
fn build_kpi() -> Value {
    let case_defs: [(&str, Vec<(&str, f64, f64)>); 4] = [
        ("ارزیابی متوازن", vec![("KPI-1", 30.0, 4.5), ("KPI-2", 30.0, 3.0), ("KPI-3", 40.0, 4.0)]),
        ("وزن صفر", vec![("KPI-1", 0.0, 5.0), ("KPI-2", 50.0, 3.5)]),
        ("همه صفر", vec![("KPI-1", 0.0, 5.0)]),
        ("نمرهٔ کامل", vec![("KPI-1", 100.0, 5.0)]),
    ];

    let cases: Vec<Value> = case_defs
        .iter()
        .map(|(name, items)| {
            let items: Vec<KpiItem> = items
                .iter()
                .map(|(id, weight, score)| KpiItem { kpi_id: (*id).to_string(), weight: *weight, score: *score })
                .collect();
            let outcome = hrms_rules::kpi::evaluate(&items);
            json!({
                "name": name,
                "items": items.iter().map(|i| json!({
                    "kpiId": i.kpi_id,
                    "weight": i.weight,
                    "score": i.score,
                })).collect::<Vec<_>>(),
                "totalWeight": outcome.total_weight,
                "weightedScore": outcome.weighted_score,
                "gradeFa": outcome.grade_fa,
            })
        })
        .collect();

    json!({
        "id": "kpi",
        "description": "میانگین موزون شاخص‌های عملکرد و رتبه‌بندی فارسی",
        "cases": cases,
    })
}

/// مقایسهٔ معنایی دو سند JSON با مسیر خطا.
fn compare(path: &str, computed: &Value, on_disk: &Value, diffs: &mut Vec<String>) {
    match (computed, on_disk) {
        (Value::Number(a), Value::Number(b)) => {
            let (x, y) = (a.as_f64().unwrap_or(f64::NAN), b.as_f64().unwrap_or(f64::NAN));
            if !(x - y).abs() < 1e-6 {
                diffs.push(format!("{path}: محاسبه={x} ولی fixture={y}"));
            }
        }
        (Value::Object(a), Value::Object(b)) => {
            for (k, v) in a {
                match b.get(k) {
                    Some(other) => compare(&format!("{path}.{k}"), v, other, diffs),
                    None => diffs.push(format!("{path}.{k}: در fixture وجود ندارد")),
                }
            }
            for k in b.keys() {
                if !a.contains_key(k) {
                    diffs.push(format!("{path}.{k}: محاسبه تولید نمی‌کند"));
                }
            }
        }
        (Value::Array(a), Value::Array(b)) => {
            if a.len() != b.len() {
                diffs.push(format!("{path}: تعداد عناصر محاسبه={} ولی fixture={}", a.len(), b.len()));
                return;
            }
            for (i, (x, y)) in a.iter().zip(b.iter()).enumerate() {
                compare(&format!("{path}[{i}]"), x, y, diffs);
            }
        }
        (x, y) if x != y => diffs.push(format!("{path}: محاسبه={x} ولی fixture={y}")),
        _ => {}
    }
}

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let write = args.iter().any(|a| a == "--write");
    let dir = fixtures_dir();

    let fixtures = [
        ("calendar", build_calendar()),
        ("leave", build_leave()),
        ("payroll", build_payroll()),
        ("kpi", build_kpi()),
    ];

    let mut failures: Vec<String> = Vec::new();

    for (name, computed) in fixtures {
        let path = dir.join(format!("{name}.json"));
        if write {
            let text = serde_json::to_string_pretty(&computed).expect("سریال‌سازی") + "\n";
            match std::fs::write(&path, text) {
                Ok(()) => println!("→ بازتولید شد: {}", path.display()),
                Err(e) => {
                    eprintln!("✖ نوشتن {} ناموفق: {e}", path.display());
                    return ExitCode::FAILURE;
                }
            }
            continue;
        }

        let on_disk: Value = match std::fs::read_to_string(&path) {
            Ok(text) => serde_json::from_str(&text).unwrap_or(Value::Null),
            Err(e) => {
                failures.push(format!("{}: خواندن ناموفق ({e})", path.display()));
                continue;
            }
        };

        let mut diffs = Vec::new();
        compare(name, &computed, &on_disk, &mut diffs);
        if diffs.is_empty() {
            println!("✓ {name}: با fixture طلایی هم‌خوان است");
        } else {
            failures.push(format!("{name}: {} اختلاف", diffs.len()));
            for d in diffs.iter().take(10) {
                failures.push(format!("    {d}"));
            }
        }
    }

    if write {
        println!("\n✓ fixtureها بازتولید شدند. تغییرات را بازبینی کنید و کامیت کنید.");
        return ExitCode::SUCCESS;
    }

    if failures.is_empty() {
        println!("\n✓ همهٔ قواعد با fixtureهای طلایی هم‌خوان است (پایهٔ تطبیق Rust و Go).");
        ExitCode::SUCCESS
    } else {
        eprintln!("\n✖ تطبیق قواعد شکست خورد — یا قاعده تغییر کرده (با --write بازتولید کنید) یا پیاده‌سازی از fixture جدا افتاده:");
        for f in failures {
            eprintln!("  {f}");
        }
        ExitCode::FAILURE
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn comparison_detects_numeric_drift() {
        let mut diffs = Vec::new();
        compare("root", &json!({ "a": 1.0, "b": [1, 2] }), &json!({ "a": 1.0000001, "b": [1, 2] }), &mut diffs);
        assert!(diffs.is_empty(), "رواداری عددی باید اختلاف ناچیز را بپذیرد");

        let mut diffs = Vec::new();
        compare("root", &json!({ "a": 1.0 }), &json!({ "a": 2.0 }), &mut diffs);
        assert_eq!(diffs.len(), 1);

        let mut diffs = Vec::new();
        compare("root", &json!({ "a": 1 }), &json!({ "b": 1 }), &mut diffs);
        assert_eq!(diffs.len(), 2, "هم کلید اضافه و هم کلید غایب باید گرفته شود");
    }

    #[test]
    fn built_fixtures_match_committed_numbers() {
        // نقطهٔ کلیدی: قواعد هسته باید همان اعدادی را بدهند که در فایل‌های کامیت‌شده است.
        let payroll = build_payroll();
        assert_eq!(payroll["cases"][0]["expected"]["net"], 53_457_000);
        assert_eq!(payroll["cases"][2]["expected"]["insuranceBase"], 200_000_000);
        assert_eq!(payroll["cases"][3]["expected"]["tax"], 0);

        let calendar = build_calendar();
        assert_eq!(calendar["conversions"][2]["gregorian"], "2025-09-23");
        assert_eq!(calendar["conversions"][2]["weekdayFa"], "سه‌شنبه");
        assert_eq!(calendar["workWindows"][2]["workingDays"], 3);
        assert_eq!(calendar["workWindows"][2]["sandwichDays"], 5);

        let leave = build_leave();
        assert_eq!(leave["cases"][4]["eligible"], false);
        assert_eq!(leave["cases"][4]["days"], 8.0);

        let kpi = build_kpi();
        assert_eq!(kpi["cases"][0]["weightedScore"], 3.85);
        assert_eq!(kpi["cases"][0]["gradeFa"], "خوب");
    }
}
