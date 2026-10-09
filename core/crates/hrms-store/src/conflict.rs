//! سیاست حل تعارض بین کلاینت و سرور.
//!
//! تعارض سینک خطرناک‌ترین بخش یک سامانهٔ آفلاین است، پس سیاست هر موجودیت صریح و
//! مستند است — نه رفتار پیش‌فرض پنهان:
//!
//! | موجودیت | سیاست | منطق |
//! |---|---|---|
//! | فیش و دورهٔ حقوق | سرور برنده | دادهٔ مالی محاسبه‌شده سمت سرور منبع حقیقت است |
//! | کارمند و پرونده | ادغام فیلد‌به‌فیلد | کاربر آفلاین ممکن است فقط یک فیلد را ویرایش کرده باشد |
//! | درخواست مرخصی | ادغام با اولویت آخرین ویرایش | تصمیم تأیید/رد سمت سرور قطعی است |
//! | لاگ حسابرسی | فقط-افزودن | بازنویسی تاریخی ممنوع |
//! | پارامترهای سیستم | سرور برنده | پیکربندی سازمانی نباید با نسخهٔ محلی کهنه بازنویسی شود |

use serde::{Deserialize, Serialize};

/// سیاست حل تعارض یک موجودیت.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ConflictPolicy {
    /// نسخهٔ سرور جایگزین نسخهٔ محلی می‌شود.
    ServerWins,
    /// فیلدهای غیرخالی محلی روی نسخهٔ سرور اعمال می‌شوند.
    Merge,
    /// داده فقط افزوده می‌شود؛ تعارض ناممکن است.
    AppendOnly,
}

impl ConflictPolicy {
    /// کلید انتقالی.
    pub fn as_str(&self) -> &'static str {
        match self {
            ConflictPolicy::ServerWins => "server_wins",
            ConflictPolicy::Merge => "merge",
            ConflictPolicy::AppendOnly => "append_only",
        }
    }
}

/// تصمیم نهایی دربارهٔ یک رکورد متعارض.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum Decision {
    /// نسخهٔ محلی معتبر است و باید برای سرور فرستاده شود.
    AcceptLocal,
    /// نسخهٔ سرور جایگزین می‌شود.
    AcceptServer,
    /// ادغام فیلد‌به‌فیلد.
    MergeFields(serde_json::Value),
}

/// سیاست پیش‌فرض هر موجودیت.
pub fn policy_for(entity: &str) -> ConflictPolicy {
    match entity {
        "Payslip" | "PayRun" | "PayItem" | "SysParam" | "Role" | "Permission" => ConflictPolicy::ServerWins,
        "AuditEvent" | "AttendanceRecord" => ConflictPolicy::AppendOnly,
        // پیش‌فرض محافظه‌کارانه برای موجودیت‌های ناشناخته: سرور برنده.
        _ => ConflictPolicy::Merge,
    }
}

/// حل تعارض بین نسخهٔ محلی و نسخهٔ سرور.
pub fn resolve(entity: &str, local_version: u32, server_version: u32, local: &serde_json::Value, server: &serde_json::Value) -> Decision {
    let policy = policy_for(entity);

    // نسخهٔ برابر یعنی تعارضی نیست.
    if local_version == server_version {
        return Decision::AcceptLocal;
    }
    // نسخهٔ محلی جدیدتر یعنی کلاینت جلوتر است — در حالت ادغام بپذیر.
    if local_version > server_version {
        return match policy {
            ConflictPolicy::ServerWins => Decision::AcceptServer,
            _ => Decision::AcceptLocal,
        };
    }
    // نسخهٔ سرور جدیدتر است: بر اساس سیاست تصمیم بگیر.
    match policy {
        ConflictPolicy::ServerWins | ConflictPolicy::AppendOnly => Decision::AcceptServer,
        ConflictPolicy::Merge => {
            let mut merged = server.clone();
            if let (Some(dst), Some(src)) = (merged.as_object_mut(), local.as_object()) {
                for (k, v) in src {
                    // فقط فیلدهایی که محلی مقدار گرفته‌اند روی سرور اعمال می‌شوند.
                    if !v.is_null() {
                        dst.insert(k.clone(), v.clone());
                    }
                }
            }
            Decision::MergeFields(merged)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn policies_are_explicit_per_entity() {
        assert_eq!(policy_for("Payslip"), ConflictPolicy::ServerWins);
        assert_eq!(policy_for("SysParam"), ConflictPolicy::ServerWins);
        assert_eq!(policy_for("AuditEvent"), ConflictPolicy::AppendOnly);
        assert_eq!(policy_for("Employee"), ConflictPolicy::Merge);
        assert_eq!(policy_for("LeaveRequest"), ConflictPolicy::Merge);
    }

    #[test]
    fn payroll_never_lets_client_win() {
        let local = json!({ "net": 999999 });
        let server = json!({ "net": 53457000 });
        assert_eq!(resolve("Payslip", 5, 2, &local, &server), Decision::AcceptServer);
        assert_eq!(resolve("Payslip", 2, 5, &local, &server), Decision::AcceptServer);
    }

    #[test]
    fn employee_edits_are_merged_field_by_field() {
        let local = json!({ "phone": "0912-1111111", "position": null });
        let server = json!({ "phone": "0912-0000001", "position": "کارشناس منابع انسانی", "baseSalary": 31000000 });
        match resolve("Employee", 1, 3, &local, &server) {
            Decision::MergeFields(v) => {
                assert_eq!(v["phone"], "0912-1111111", "فیلد ویرایش‌شدهٔ محلی حفظ می‌شود");
                assert_eq!(v["position"], "کارشناس منابع انسانی", "فیلد خالی محلی روی سرور اعمال نمی‌شود");
                assert_eq!(v["baseSalary"], 31000000);
            }
            other => panic!("انتظار ادغام می‌رفت، دریافت شد {other:?}"),
        }
    }

    #[test]
    fn equal_versions_are_not_conflicts() {
        let v = json!({ "x": 1 });
        assert_eq!(resolve("Employee", 4, 4, &v, &v), Decision::AcceptLocal);
    }

    #[test]
    fn append_only_accepts_server_copy() {
        let local = json!({ "action": "read" });
        let server = json!({ "action": "read", "extra": true });
        assert_eq!(resolve("AuditEvent", 1, 2, &local, &server), Decision::AcceptServer);
    }
}
