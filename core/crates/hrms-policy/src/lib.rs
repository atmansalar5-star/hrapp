//! موتور سیاست دسترسی نقش‌محور.
//!
//! جدول سیاست در `policy_gen.rs` از همان مانیفست ماژول‌های سرور تولید می‌شود
//! (`server/internal/modules/*/module.json`). یعنی RBAC در کلاینت و سرور یک منبع
//! واحد دارد و نمی‌تواند از هم جدا بیفتد؛ کلاینت فقط می‌تواند چیزی را پنهان کند که
//! سرور هم رد می‌کند، و منوی غیرمجاز هرگز رندر نمی‌شود.

use serde::{Deserialize, Serialize};

/// نقش‌های سامانه.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum RoleKey {
    Admin,
    HrManager,
    HrSpec,
    LineManager,
    Employee,
}

impl RoleKey {
    /// کلید انتقالی روی سیم (هم‌خوان با JWT و قرارداد OpenAPI).
    pub fn as_str(&self) -> &'static str {
        match self {
            RoleKey::Admin => "admin",
            RoleKey::HrManager => "hr_manager",
            RoleKey::HrSpec => "hr_spec",
            RoleKey::LineManager => "line_manager",
            RoleKey::Employee => "employee",
        }
    }

    /// نام فارسی برای نمایش.
    pub fn fa(&self) -> &'static str {
        match self {
            RoleKey::Admin => "مدیر سیستم",
            RoleKey::HrManager => "مدیر منابع انسانی",
            RoleKey::HrSpec => "کارشناس منابع انسانی",
            RoleKey::LineManager => "مدیر واحد",
            RoleKey::Employee => "کارمند",
        }
    }

    /// تجزیه از کلید.
    pub fn parse(s: &str) -> Option<Self> {
        match s.trim() {
            "admin" => Some(RoleKey::Admin),
            "hr_manager" => Some(RoleKey::HrManager),
            "hr_spec" => Some(RoleKey::HrSpec),
            "line_manager" => Some(RoleKey::LineManager),
            "employee" => Some(RoleKey::Employee),
            _ => None,
        }
    }
}

/// اقدام‌های قابل انجام روی یک ماژول.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum Action {
    Read,
    Write,
    Delete,
    Approve,
    Report,
    Admin,
}

impl Action {
    /// کلید انتقالی.
    pub fn as_str(&self) -> &'static str {
        match self {
            Action::Read => "read",
            Action::Write => "write",
            Action::Delete => "delete",
            Action::Approve => "approve",
            Action::Report => "report",
            Action::Admin => "admin",
        }
    }

    /// تجزیه از کلید.
    pub fn parse(s: &str) -> Option<Self> {
        match s.trim() {
            "read" => Some(Action::Read),
            "write" => Some(Action::Write),
            "delete" => Some(Action::Delete),
            "approve" => Some(Action::Approve),
            "report" => Some(Action::Report),
            "admin" => Some(Action::Admin),
            _ => None,
        }
    }
}

/// دامنهٔ دادهٔ نقش در یک ماژول.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum Scope {
    All,
    OwnUnit,
    Own,
}

impl Scope {
    /// کلید انتقالی.
    pub fn as_str(&self) -> &'static str {
        match self {
            Scope::All => "all",
            Scope::OwnUnit => "own_unit",
            Scope::Own => "own",
        }
    }
}

/// سیاست نمایش یک صفحه.
#[derive(Debug, Clone)]
pub struct ScreenPolicy {
    pub key: &'static str,
    pub title_fa: &'static str,
    pub component: &'static str,
    pub icon: &'static str,
    pub roles: &'static [RoleKey],
}

/// سیاست یک ماژول.
#[derive(Debug, Clone)]
pub struct ModulePolicy {
    pub module_id: &'static str,
    pub order: u32,
    pub title_fa: &'static str,
    pub api_prefix: &'static str,
    pub feature_flag: &'static str,
    pub allowed: &'static [(RoleKey, Action)],
    pub scopes: &'static [(RoleKey, Scope)],
    pub screens: &'static [ScreenPolicy],
}

mod policy_gen;
pub use policy_gen::{POLICIES, ROLE_KEYS};

/// همهٔ ماژول‌ها به ترتیب منو.
pub fn modules() -> &'static [ModulePolicy] {
    POLICIES
}

/// یافتن ماژول با شناسه.
pub fn module(module_id: &str) -> Option<&'static ModulePolicy> {
    POLICIES.iter().find(|m| m.module_id == module_id)
}

/// آیا نقش اجازهٔ اقدام را در ماژول دارد.
pub fn allows(role: RoleKey, module_id: &str, action: Action) -> bool {
    match module(module_id) {
        Some(m) => m.allowed.iter().any(|&(r, a)| r == role && a == action),
        None => false,
    }
}

/// دامنهٔ دادهٔ نقش در ماژول (پیش‌فرض: سازمانی).
pub fn scope_of(role: RoleKey, module_id: &str) -> Scope {
    match module(module_id) {
        Some(m) => m
            .scopes
            .iter()
            .find(|&&(r, _)| r == role)
            .map(|&(_, s)| s)
            .unwrap_or(Scope::All),
        None => Scope::Own,
    }
}

/// صفحه‌های قابل نمایش یک نقش در یک ماژول.
pub fn screens_for(role: RoleKey, module_id: &str) -> Vec<&'static ScreenPolicy> {
    module(module_id)
        .map(|m| m.screens.iter().filter(|s| s.roles.contains(&role)).collect())
        .unwrap_or_default()
}

/// آیا نقش اجازهٔ باز کردن صفحهٔ مشخص را دارد.
pub fn can_open_screen(role: RoleKey, module_id: &str, screen_key: &str) -> bool {
    screens_for(role, module_id).iter().any(|s| s.key == screen_key)
}

/// ماژول‌هایی که نقش دست‌کم به یک صفحهٔ آن‌ها دسترسی دارد.
pub fn accessible_modules(role: RoleKey) -> Vec<&'static ModulePolicy> {
    POLICIES.iter().filter(|m| m.screens.iter().any(|s| s.roles.contains(&role))).collect()
}

/// یک آیتم منو برای کلاینت.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MenuEntry {
    pub module_id: String,
    pub title_fa: String,
    pub order: u32,
    pub screens: Vec<MenuScreen>,
}

/// یک زیرآیتم منو.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MenuScreen {
    pub key: String,
    pub title_fa: String,
    pub component: String,
    pub icon: String,
}

/// درخت منوی نقشمحور؛ همین خروجی هم در کلاینت و هم در پاسخ `/api/v1/auth/me` استفاده می‌شود.
pub fn menu_for(role: RoleKey) -> Vec<MenuEntry> {
    let mut out = Vec::new();
    for m in POLICIES {
        let screens: Vec<MenuScreen> = m
            .screens
            .iter()
            .filter(|s| s.roles.contains(&role))
            .map(|s| MenuScreen {
                key: s.key.to_string(),
                title_fa: s.title_fa.to_string(),
                component: s.component.to_string(),
                icon: s.icon.to_string(),
            })
            .collect();
        if screens.is_empty() {
            continue;
        }
        out.push(MenuEntry {
            module_id: m.module_id.to_string(),
            title_fa: m.title_fa.to_string(),
            order: m.order,
            screens,
        });
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn registry_has_all_ten_modules_in_order() {
        assert_eq!(modules().len(), 10, "سند طراحی ۱۰ منوی اصلی می‌خواهد");
        let orders: Vec<u32> = modules().iter().map(|m| m.order).collect();
        assert_eq!(orders, (1..=10).collect::<Vec<u32>>());
    }

    #[test]
    fn settings_is_admin_only_for_user_management() {
        assert!(can_open_screen(RoleKey::Admin, "settings", "users"));
        assert!(!can_open_screen(RoleKey::HrManager, "settings", "users"));
        assert!(can_open_screen(RoleKey::HrManager, "settings", "orgSettings"));
        assert!(!allows(RoleKey::HrSpec, "settings", Action::Read));
    }

    #[test]
    fn hr_specialist_cannot_delete_employees() {
        assert!(allows(RoleKey::HrSpec, "employee", Action::Write));
        assert!(!allows(RoleKey::HrSpec, "employee", Action::Delete));
        assert!(allows(RoleKey::HrManager, "employee", Action::Delete));
    }

    #[test]
    fn scopes_limit_employee_and_line_manager() {
        assert_eq!(scope_of(RoleKey::LineManager, "employee"), Scope::OwnUnit);
        assert_eq!(scope_of(RoleKey::Employee, "employee"), Scope::Own);
        assert_eq!(scope_of(RoleKey::Admin, "employee"), Scope::All);
        assert_eq!(scope_of(RoleKey::Employee, "payroll"), Scope::Own);
    }

    #[test]
    fn employee_menu_hides_forbidden_modules() {
        let menu = menu_for(RoleKey::Employee);
        let ids: Vec<&str> = menu.iter().map(|m| m.module_id.as_str()).collect();
        assert!(!ids.contains(&"recruitment"), "کارمند نباید استخدام را ببیند");
        assert!(!ids.contains(&"settings"), "کارمند نباید تنظیمات را ببیند");
        assert!(ids.contains(&"dashboard"));
        assert!(ids.contains(&"payroll"), "فیش حقوقی برای کارمند قابل مشاهده است");
        // منوی کارمند باید فقط صفحه‌های مجاز را داشته باشد
        let payroll = menu.iter().find(|m| m.module_id == "payroll").unwrap();
        assert_eq!(payroll.screens.len(), 1);
        assert_eq!(payroll.screens[0].key, "payslip");
    }

    #[test]
    fn line_manager_has_leave_approval_but_not_payroll_run() {
        assert!(allows(RoleKey::LineManager, "attendance", Action::Approve));
        assert!(can_open_screen(RoleKey::LineManager, "attendance", "leaveApprove"));
        assert!(!can_open_screen(RoleKey::LineManager, "payroll", "payrollRun"));
        assert!(can_open_screen(RoleKey::LineManager, "payroll", "payslip"));
    }

    #[test]
    fn role_keys_round_trip_and_cover_manifest() {
        for r in ROLE_KEYS {
            assert_eq!(RoleKey::parse(r.as_str()), Some(r));
            assert!(!r.fa().is_empty());
        }
        for m in modules() {
            let roles: Vec<RoleKey> = m.allowed.iter().map(|&(r, _)| r).collect();
            for r in ROLE_KEYS {
                if roles.contains(&r) {
                    assert!(!screens_for(r, m.module_id).is_empty(), "{} برای {}", m.module_id, r.as_str());
                }
            }
        }
    }
}
