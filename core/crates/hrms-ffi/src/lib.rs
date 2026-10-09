//! تنها سطح صادرشدهٔ هسته به بیرون.
//!
//! قاعدهٔ معماری: هیچ crate دیگری از هسته بیرون نمی‌رود. کلاینت (React Native) فقط از
//! طریق یک نقطهٔ ورود با هسته حرف می‌زند:
//!
//! ```text
//! hrms_call("payroll.compute", r#"{"input":{...},"params":{...}}"#) -> JSON
//! ```
//!
//! ورودی و خروجی همیشه JSON است تا مرز پل باریک و نسخه‌پذیر بماند و لازم نباشد برای هر
//! تغییر دامنه، لایهٔ بومی موبایل بازتولید شود. روش‌ها در [`METHODS`] فهرست شده‌اند.
//!
//! برای اندروید/iOS همین نقطهٔ ورود با `uniffi::export` پوشانده می‌شود و
//! `NativeHrmsCore` در React Native روی آن می‌نشیند؛ در ویندوز همین کتابخانه به‌صورت
//! DLL بار می‌شود تا همان هسته، کلاینت دسکتاپ را هم پشتیبانی کند.

use serde_json::{json, Value};
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::sync::{Mutex, OnceLock};

/// فهرست روش‌های پشتیبانی‌شدهٔ پل.
pub const METHODS: [&str; 14] = [
    "core.version",
    "core.bootstrap",
    "policy.menu",
    "policy.authorize",
    "employee.validate",
    "payroll.compute",
    "leave.compute",
    "calendar.info",
    "store.enqueue",
    "store.pending",
    "store.markSent",
    "store.markRejected",
    "store.prune",
    "store.cursor",
];

/// نسخهٔ هسته — برای تشخیص ناهم‌خوانی هسته و کلاینت در زمان اجرا.
pub const CORE_VERSION: &str = env!("CARGO_PKG_VERSION");

/// خلاصهٔ راه‌اندازی: نسخه، ماژول‌ها و کلیدهای روش‌ها.
/// کلاینت در اولین اجرا این را می‌گیرد و در صورت ناهم‌خوانی نسخه هشدار می‌دهد.
pub fn bootstrap() -> Value {
    let modules: Vec<Value> = hrms_policy::modules()
        .iter()
        .map(|m| {
            json!({
                "moduleId": m.module_id,
                "order": m.order,
                "titleFa": m.title_fa,
                "apiPrefix": m.api_prefix,
                "featureFlag": m.feature_flag,
                "screens": m.screens.iter().map(|s| s.key).collect::<Vec<_>>(),
            })
        })
        .collect();

    json!({
        "coreVersion": CORE_VERSION,
        "schemaVersion": hrms_store::schema::SCHEMA_VERSION,
        "modules": modules,
        "methods": METHODS,
        "roles": hrms_policy::ROLE_KEYS.iter().map(|r| r.as_str()).collect::<Vec<_>>(),
    })
}

/// منوی نقش‌محور برای ساخت ناوبری کلاینت.
pub fn menu(role: &str) -> Result<Value, String> {
    let role = hrms_policy::RoleKey::parse(role).ok_or_else(|| format!("نقش ناشناخته: {role}"))?;
    serde_json::to_value(hrms_policy::menu_for(role)).map_err(|e| e.to_string())
}

/// بررسی دسترسی نقش به یک اقدام در یک ماژول.
pub fn authorize(role: &str, module_id: &str, action: &str) -> Result<bool, String> {
    let role = hrms_policy::RoleKey::parse(role).ok_or_else(|| format!("نقش ناشناخته: {role}"))?;
    let action = hrms_policy::Action::parse(action).ok_or_else(|| format!("اقدام ناشناخته: {action}"))?;
    Ok(hrms_policy::allows(role, module_id, action))
}

/// اعتبارسنجی ورودی کارمند؛ خروجی خطاهای فیلد‌به‌فیلد فارسی است.
pub fn validate_employee(payload: &Value) -> Result<Value, String> {
    let input: hrms_domain::EmployeeInput = serde_json::from_value(payload.clone()).map_err(|e| format!("ورودی کارمند نامعتبر: {e}"))?;
    let errors = hrms_domain::validate_employee_input(&input);
    Ok(json!({ "valid": errors.is_empty(), "errors": errors.0 }))
}

/// محاسبهٔ فیش حقوقی.
pub fn compute_payroll(input: &Value, params: &Value) -> Result<Value, String> {
    let input: hrms_rules::payroll::PayrollInput = serde_json::from_value(input.clone()).map_err(|e| format!("ورودی حقوق نامعتبر: {e}"))?;
    let params: hrms_rules::payroll::PayrollParams = serde_json::from_value(params.clone()).map_err(|e| format!("پارامترهای حقوق نامعتبر: {e}"))?;
    let result = hrms_rules::payroll::compute(&input, &params);
    serde_json::to_value(result).map_err(|e| e.to_string())
}

/// محاسبهٔ روز و ماندهٔ مرخصی.
pub fn compute_leave(payload: &Value) -> Result<Value, String> {
    let from = payload.get("from").and_then(Value::as_str).ok_or("فیلد from لازم است")?;
    let to = payload.get("to").and_then(Value::as_str).ok_or("فیلد to لازم است")?;
    let balance = payload.get("balance").and_then(Value::as_f64).unwrap_or(0.0);
    let sandwich = payload.get("sandwich").and_then(Value::as_bool).unwrap_or(false);

    let from = hrms_calendar::JalaliDate::parse(from).map_err(|e| e.to_string())?;
    let to = hrms_calendar::JalaliDate::parse(to).map_err(|e| e.to_string())?;

    let mut calendar = hrms_calendar::WorkCalendar::default_iran();
    if let Some(weekend) = payload.get("weekend").and_then(Value::as_array) {
        calendar.weekend = weekend.iter().filter_map(|v| v.as_u64().map(|n| n as u32)).collect();
    }
    if let Some(holidays) = payload.get("holidays").and_then(Value::as_object) {
        for (k, v) in holidays {
            calendar.holidays.insert(k.clone(), v.as_str().unwrap_or("").to_string());
        }
    }

    let params = hrms_rules::leave::LeaveParams::default();
    let days = hrms_rules::leave::count_leave_days(&from, &to, &calendar, sandwich).map_err(|e| e.to_string())?;
    let eligible = hrms_rules::leave::is_eligible(balance, days);

    Ok(json!({
        "days": days,
        "workingDays": calendar.working_days(&from, &to).map_err(|e| e.to_string())?,
        "sandwichDays": calendar.sandwich_days(&from, &to).map_err(|e| e.to_string())?,
        "eligible": eligible,
        "balanceAfter": hrms_rules::leave::balance_after(balance, days),
        "params": params,
    }))
}

/// اطلاعات تقویمی یک تاریخ: میلادی، روز هفته، برچسب ماه و طول ماه.
pub fn calendar_info(payload: &Value) -> Result<Value, String> {
    let date = payload.get("date").and_then(Value::as_str).ok_or("فیلد date لازم است")?;
    let d = hrms_calendar::JalaliDate::parse(date).map_err(|e| e.to_string())?;
    let (gy, gm, gd) = d.to_gregorian().map_err(|e| e.to_string())?;
    Ok(json!({
        "jalali": d.to_string(),
        "jalaliPersian": hrms_calendar::to_persian_digits(&d.to_string()),
        "gregorian": format!("{gy:04}-{gm:02}-{gd:02}"),
        "weekdayIndex": d.weekday_index().map_err(|e| e.to_string())?,
        "weekdayFa": d.weekday_fa().map_err(|e| e.to_string())?,
        "monthLabelFa": d.month_label_fa().map_err(|e| e.to_string())?,
        "monthLength": hrms_calendar::JalaliDate::month_length(d.year, d.month).map_err(|e| e.to_string())?,
        "leap": hrms_calendar::JalaliDate::is_leap(d.year).map_err(|e| e.to_string())?,
    }))
}

/// صف آفلاین مشترک این فرآیند (در استقرار واقعی همین صف روی SQLite نوشته می‌شود).
fn outbox() -> &'static Mutex<hrms_store::outbox::Outbox> {
    static OUTBOX: OnceLock<Mutex<hrms_store::outbox::Outbox>> = OnceLock::new();
    OUTBOX.get_or_init(|| Mutex::new(hrms_store::outbox::Outbox::new()))
}

/// نشانگر سینک ذخیره‌شدهٔ کلاینت.
fn cursor() -> &'static Mutex<i64> {
    static CURSOR: OnceLock<Mutex<i64>> = OnceLock::new();
    CURSOR.get_or_init(|| Mutex::new(0))
}

// ------------------------------------------------------------------ صف آفلاین

fn store_enqueue(payload: &Value) -> Result<Value, String> {
    use hrms_store::outbox::{Op, OutboxEntry, OutboxState};

    let op = match payload.get("op").and_then(Value::as_str).unwrap_or("update") {
        "create" => Op::Create,
        "delete" => Op::Delete,
        _ => Op::Update,
    };
    let entry = OutboxEntry {
        op_id: payload.get("opId").and_then(Value::as_str).ok_or("فیلد opId لازم است")?.to_string(),
        entity: payload.get("entity").and_then(Value::as_str).ok_or("فیلد entity لازم است")?.to_string(),
        op,
        entity_id: payload.get("entityId").and_then(Value::as_str).unwrap_or_default().to_string(),
        payload: payload.get("payload").cloned().unwrap_or(Value::Null),
        client_ts: payload.get("clientTs").and_then(Value::as_str).unwrap_or_default().to_string(),
        base_version: payload.get("baseVersion").and_then(Value::as_u64).unwrap_or(0) as u32,
        state: OutboxState::Pending,
        reason: None,
    };

    let mut queue = outbox().lock().map_err(|_| "قفل صف آفلاین در دسترس نیست")?;
    let enqueued = queue.enqueue(entry);
    Ok(json!({ "enqueued": enqueued, "pending": queue.pending_count() }))
}

fn store_pending() -> Result<Value, String> {
    let queue = outbox().lock().map_err(|_| "قفل صف آفلاین در دسترس نیست")?;
    let entries: Vec<Value> = queue
        .pending()
        .iter()
        .map(|e| {
            json!({
                "opId": e.op_id,
                "entity": e.entity,
                "op": e.op.as_str(),
                "entityId": e.entity_id,
                "payload": e.payload,
                "clientTs": e.client_ts,
                "baseVersion": e.base_version,
            })
        })
        .collect();
    Ok(json!({ "entries": entries, "pending": entries.len() }))
}

fn store_mark_sent(payload: &Value) -> Result<Value, String> {
    let ids: Vec<String> = payload
        .get("opIds")
        .and_then(Value::as_array)
        .map(|a| a.iter().filter_map(|v| v.as_str().map(str::to_string)).collect())
        .unwrap_or_default();
    let mut queue = outbox().lock().map_err(|_| "قفل صف آفلاین در دسترس نیست")?;
    queue.mark_sent(&ids);
    Ok(json!({ "pending": queue.pending_count() }))
}

fn store_mark_rejected(payload: &Value) -> Result<Value, String> {
    let op_id = payload.get("opId").and_then(Value::as_str).ok_or("فیلد opId لازم است")?;
    let reason = payload.get("reason").and_then(Value::as_str).unwrap_or("رد شده توسط سرور");
    let mut queue = outbox().lock().map_err(|_| "قفل صف آفلاین در دسترس نیست")?;
    queue.mark_rejected(op_id, reason);
    Ok(json!({ "pending": queue.pending_count() }))
}

fn store_prune() -> Result<Value, String> {
    let mut queue = outbox().lock().map_err(|_| "قفل صف آفلاین در دسترس نیست")?;
    Ok(json!({ "removed": queue.prune_finished(), "pending": queue.pending_count() }))
}

fn get_cursor() -> Result<Value, String> {
    let seq = *cursor().lock().map_err(|_| "قفل نشانگر سینک در دسترس نیست")?;
    Ok(json!({ "cursor": seq }))
}

fn set_cursor(payload: &Value) -> Result<Value, String> {
    let seq = payload.get("seq").and_then(Value::as_i64).unwrap_or(0);
    let mut slot = cursor().lock().map_err(|_| "قفل نشانگر سینک در دسترس نیست")?;
    if seq > *slot {
        *slot = seq;
    }
    Ok(json!({ "cursor": *slot }))
}

/// فراخوانی عمومی پل: نام روش + بار JSON → نتیجهٔ JSON.
///
/// اگر روش ناشناخته باشد، خطای JSON برگردانده می‌شود (نه panic) تا کلاینت بتواند
/// پیام فارسی را نمایش دهد.
pub fn call(method: &str, payload: Option<&Value>) -> Value {
    let empty = Value::Null;
    let payload = payload.unwrap_or(&empty);
    let result = match method {
        "core.version" => Ok(json!({ "coreVersion": CORE_VERSION })),
        "core.bootstrap" => Ok(bootstrap()),
        "policy.menu" => {
            let role = payload.get("role").and_then(Value::as_str).unwrap_or_default();
            menu(role)
        }
        "policy.authorize" => {
            let role = payload.get("role").and_then(Value::as_str).unwrap_or_default();
            let module = payload.get("module").and_then(Value::as_str).unwrap_or_default();
            let action = payload.get("action").and_then(Value::as_str).unwrap_or_default();
            authorize(role, module, action).map(|ok| json!({ "allowed": ok }))
        }
        "employee.validate" => validate_employee(payload),
        "payroll.compute" => {
            let input = payload.get("input").cloned().unwrap_or(Value::Null);
            let params = payload.get("params").cloned().unwrap_or(Value::Null);
            compute_payroll(&input, &params)
        }
        "leave.compute" => compute_leave(payload),
        "calendar.info" => calendar_info(payload),
        "store.enqueue" => store_enqueue(payload),
        "store.pending" => store_pending(),
        "store.markSent" => store_mark_sent(payload),
        "store.markRejected" => store_mark_rejected(payload),
        "store.prune" => store_prune(),
        "store.cursor" => get_cursor(),
        "store.setCursor" => set_cursor(payload),
        other => Err(format!("روش ناشناخته: {other}")),
    };

    match result {
        Ok(value) => json!({ "ok": true, "result": value }),
        Err(message) => json!({ "ok": false, "error": message }),
    }
}

/// نقطهٔ ورود C ABI برای لایهٔ بومی موبایل و دسکتاپ.
///
/// # Safety
/// `method` و `payload` باید اشاره‌گر به رشته‌های UTF-8 پایدار و پایان‌یافته با NUL
/// باشند. رشتهٔ بازگشتی باید با [`hrms_string_free`] آزاد شود.
#[no_mangle]
pub unsafe extern "C" fn hrms_call(method: *const c_char, payload: *const c_char) -> *mut c_char {
    let method = if method.is_null() {
        String::new()
    } else {
        CStr::from_ptr(method).to_string_lossy().into_owned()
    };
    let payload_value: Value = if payload.is_null() {
        Value::Null
    } else {
        let raw = CStr::from_ptr(payload).to_string_lossy().into_owned();
        serde_json::from_str(&raw).unwrap_or(Value::Null)
    };

    let response = call(&method, Some(&payload_value));
    let text = serde_json::to_string(&response).unwrap_or_else(|_| r#"{"ok":false,"error":"خطای سریال‌سازی"}"#.to_string());
    CString::new(text).unwrap_or_default().into_raw()
}

/// آزادسازی رشتهٔ بازگشتی از [`hrms_call`].
///
/// # Safety
/// اشاره‌گر باید دقیقاً همان چیزی باشد که `hrms_call` برگردانده و پیش‌تر آزاد نشده باشد.
#[no_mangle]
pub unsafe extern "C" fn hrms_string_free(ptr: *mut c_char) {
    if !ptr.is_null() {
        drop(CString::from_raw(ptr));
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bootstrap_lists_all_modules_and_methods() {
        let v = bootstrap();
        assert_eq!(v["modules"].as_array().unwrap().len(), 10);
        assert_eq!(v["methods"].as_array().unwrap().len(), METHODS.len());
        assert_eq!(v["schemaVersion"], 1);
    }

    #[test]
    fn unknown_method_returns_persian_error_not_panic() {
        let r = call("nope.nope", None);
        assert_eq!(r["ok"], false);
        assert!(r["error"].as_str().unwrap().contains("روش ناشناخته"));
    }

    #[test]
    fn authorize_matches_server_policy() {
        assert_eq!(call("policy.authorize", Some(&json!({ "role": "hr_spec", "module": "employee", "action": "delete" })))["result"]["allowed"], false);
        assert_eq!(call("policy.authorize", Some(&json!({ "role": "hr_manager", "module": "employee", "action": "delete" })))["result"]["allowed"], true);
    }

    #[test]
    fn menu_hides_forbidden_modules_for_employee() {
        let r = call("policy.menu", Some(&json!({ "role": "employee" })));
        let ids: Vec<String> = r["result"]
            .as_array()
            .unwrap()
            .iter()
            .map(|m| m["moduleId"].as_str().unwrap().to_string())
            .collect();
        assert!(!ids.contains(&"settings".to_string()));
        assert!(ids.contains(&"payroll".to_string()));
    }

    #[test]
    fn payroll_and_leave_compute_through_the_bridge() {
        let payroll = call(
            "payroll.compute",
            Some(&json!({
                "input": { "baseSalary": 52000000, "children": 0, "overtimeHours": 0, "overtimeHourlyRate": 0, "otherDeductions": 0 },
                "params": {
                    "insuranceEmployeeRate": 0.07, "insuranceEmployerRate": 0.23, "insuranceCeiling": 200000000,
                    "housingAllowance": 9000000, "childAllowancePerChild": 7000000, "taxExemption": 24000000,
                    "taxBrackets": [
                        { "upto": 50000000, "rate": 0.10 }, { "upto": 100000000, "rate": 0.15 },
                        { "upto": 200000000, "rate": 0.20 }, { "upto": 0, "rate": 0.30 }
                    ]
                }
            })),
        );
        assert_eq!(payroll["result"]["net"], 53_457_000);

        let leave = call("leave.compute", Some(&json!({ "from": "1404/07/14", "to": "1404/07/16", "balance": 12.0 })));
        assert_eq!(leave["result"]["days"], 3.0);
        assert_eq!(leave["result"]["balanceAfter"], 9.0);
    }

    #[test]
    fn calendar_info_reports_persian_labels() {
        let r = call("calendar.info", Some(&json!({ "date": "1404/07/01" })));
        assert_eq!(r["result"]["gregorian"], "2025-09-23");
        assert_eq!(r["result"]["weekdayFa"], "سه‌شنبه");
        assert_eq!(r["result"]["monthLabelFa"], "مهر ۱۴۰۴");
    }

    #[test]
    fn outbox_is_idempotent_and_prunes_finished_entries() {
        let entry = json!({
            "opId": "op-test-1",
            "entity": "Employee",
            "op": "update",
            "entityId": "EMP-001",
            "payload": { "position": "کارشناس ارشد" },
            "clientTs": "1404/07/05 09:30",
            "baseVersion": 3
        });

        let first = call("store.enqueue", Some(&entry));
        assert_eq!(first["result"]["enqueued"], true);
        let second = call("store.enqueue", Some(&entry));
        assert_eq!(second["result"]["enqueued"], false, "همان opId دوباره ثبت نمی‌شود");

        let pending = call("store.pending", None);
        assert_eq!(pending["result"]["pending"], 1);

        call("store.markSent", Some(&json!({ "opIds": ["op-test-1"] })));
        let after = call("store.pending", None);
        assert_eq!(after["result"]["pending"], 0);
        assert_eq!(call("store.prune", None)["result"]["removed"], 1);
    }

    #[test]
    fn sync_cursor_never_moves_backwards() {
        assert_eq!(call("store.setCursor", Some(&json!({ "seq": 40 })))["result"]["cursor"], 40);
        assert_eq!(call("store.setCursor", Some(&json!({ "seq": 12 })))["result"]["cursor"], 40);
        assert_eq!(call("store.cursor", None)["result"]["cursor"], 40);
    }

    #[test]
    fn validation_errors_are_persian_and_field_scoped() {
        let r = call(
            "employee.validate",
            Some(&json!({
                "fullName": "ا", "nationalId": "123", "departmentId": "", "position": "",
                "contractType": "fullTime", "hireDate": "1404/13/40"
            })),
        );
        assert_eq!(r["result"]["valid"], false);
        let errors = r["result"]["errors"].as_object().unwrap();
        assert!(errors.contains_key("nationalId"));
        assert!(errors.contains_key("hireDate"));
        assert!(errors["fullName"].as_str().unwrap().contains("۳ نویسه"));
    }
}
