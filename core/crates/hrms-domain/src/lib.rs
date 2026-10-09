//! انواع دامنهٔ منابع انسانی و اعتبارسنجی.
//!
//! این crate «زبان مشترک» سامانه است: هر چیزی که هم کلاینت و هم سرور باید دربارهٔ آن
//! توافق داشته باشند اینجا تعریف می‌شود. قواعد محاسباتی در `hrms-rules` و سیاست
//! دسترسی در `hrms-policy` جای دارد.
//!
//! قاعدهٔ مهم: هیچ قاعدهٔ کسب‌وکاری در UI نوشته نمی‌شود؛ اعتبارسنجی همین‌جا انجام و
//! در سرور هم تکرار می‌شود تا کلاینت هرگز نتواند داده نامعتبر بفرستد.

use hrms_calendar::JalaliDate;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

/// نوع قرارداد استخدامی.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum ContractType {
    FullTime,
    Contractor,
    PartTime,
}

impl ContractType {
    /// برچسب فارسی مطابق سند طراحی.
    pub fn fa(&self) -> &'static str {
        match self {
            ContractType::FullTime => "تمام‌وقت",
            ContractType::Contractor => "پیمانکاری",
            ContractType::PartTime => "پاره‌وقت",
        }
    }

    /// تجزیه از رشتهٔ فارسی ذخیره‌شده در دیتابیس.
    pub fn parse(s: &str) -> Option<Self> {
        match s.trim() {
            "تمام‌وقت" | "تمام وقت" => Some(ContractType::FullTime),
            "پیمانکاری" => Some(ContractType::Contractor),
            "پاره‌وقت" | "پاره وقت" => Some(ContractType::PartTime),
            _ => None,
        }
    }
}

/// وضعیت کارمند در چرخهٔ عمر سازمانی.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum EmployeeStatus {
    Active,
    OnLeave,
    Probation,
    Terminated,
}

impl EmployeeStatus {
    /// برچسب فارسی.
    pub fn fa(&self) -> &'static str {
        match self {
            EmployeeStatus::Active => "شاغل",
            EmployeeStatus::OnLeave => "مرخصی",
            EmployeeStatus::Probation => "آزمایشی",
            EmployeeStatus::Terminated => "ترک خدمت",
        }
    }

    /// کلید ذخیره‌سازی و انتقال (هم‌خوان با enum قرارداد OpenAPI).
    pub fn key(&self) -> &'static str {
        match self {
            EmployeeStatus::Active => "active",
            EmployeeStatus::OnLeave => "leave",
            EmployeeStatus::Probation => "probation",
            EmployeeStatus::Terminated => "terminated",
        }
    }

    /// تجزیه از کلید.
    pub fn parse(s: &str) -> Option<Self> {
        match s {
            "active" => Some(EmployeeStatus::Active),
            "leave" => Some(EmployeeStatus::OnLeave),
            "probation" => Some(EmployeeStatus::Probation),
            "terminated" => Some(EmployeeStatus::Terminated),
            _ => None,
        }
    }
}

/// کارمند — شکل انتقال داده مطابق قرارداد OpenAPI.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Employee {
    pub id: String,
    pub full_name: String,
    pub national_id: String,
    pub department_id: String,
    #[serde(default)]
    pub department_name: String,
    pub position: String,
    pub contract_type: ContractType,
    pub status: EmployeeStatus,
    /// تاریخ استخدام به قالب شمسی YYYY/MM/DD.
    pub hire_date: String,
    /// حقوق پایه به ریال.
    pub base_salary: i64,
    #[serde(default)]
    pub leave_balance: f64,
    #[serde(default)]
    pub phone: String,
    #[serde(default)]
    pub email: String,
    #[serde(default)]
    pub manager_name: String,
    #[serde(default)]
    pub version: u32,
}

/// ورودی ایجاد/ویرایش کارمند.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct EmployeeInput {
    pub full_name: String,
    pub national_id: String,
    pub department_id: String,
    pub position: String,
    pub contract_type: ContractType,
    pub hire_date: String,
    #[serde(default)]
    pub status: Option<EmployeeStatus>,
    #[serde(default)]
    pub base_salary: i64,
    #[serde(default)]
    pub phone: String,
    #[serde(default)]
    pub email: String,
}

/// خطای اعتبارسنجی فیلد‌به‌فیلد با پیام فارسی (در UI کنار همان فیلد نمایش داده می‌شود).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ValidationErrors(pub BTreeMap<String, String>);

impl ValidationErrors {
    /// آیا خطایی ثبت شده است.
    pub fn is_empty(&self) -> bool {
        self.0.is_empty()
    }

    /// افزودن خطای یک فیلد.
    pub fn add(&mut self, field: &str, message: &str) {
        self.0.insert(field.to_string(), message.to_string());
    }
}

/// اعتبارسنجی کد ملی ایران (۱۰ رقم با رقم کنترل).
pub fn is_valid_national_id(input: &str) -> bool {
    let digits: Vec<u32> = input.chars().filter_map(|c| c.to_digit(10)).collect();
    if digits.len() != 10 || input.chars().any(|c| !c.is_ascii_digit()) {
        return false;
    }
    if digits.iter().all(|&d| d == digits[0]) {
        return false; // کدهای تکراری مانند ۱۱۱۱۱۱۱۱۱۱ معتبر نیستند
    }
    let sum: u32 = digits
        .iter()
        .take(9)
        .enumerate()
        .map(|(i, &d)| d * (10 - i as u32))
        .sum();
    let r = sum % 11;
    let check = digits[9];
    if r < 2 {
        check == r
    } else {
        check == 11 - r
    }
}

/// اعتبارسنجی ورودی کارمند؛ همهٔ پیام‌ها فارسی و آمادهٔ نمایش در فرم هستند.
pub fn validate_employee_input(input: &EmployeeInput) -> ValidationErrors {
    let mut errs = ValidationErrors::default();

    if input.full_name.trim().chars().count() < 3 {
        errs.add("fullName", "نام و نام خانوادگی باید حداقل ۳ نویسه باشد.");
    }
    if !is_valid_national_id(&input.national_id) {
        errs.add("nationalId", "کد ملی باید ۱۰ رقم و دارای رقم کنترل معتبر باشد.");
    }
    if input.department_id.trim().is_empty() {
        errs.add("departmentId", "انتخاب واحد سازمانی الزامی است.");
    }
    if input.position.trim().is_empty() {
        errs.add("position", "عنوان پست الزامی است.");
    }
    match JalaliDate::parse(&input.hire_date) {
        Ok(_) => {}
        Err(_) => {
            errs.add("hireDate", "تاریخ استخدام باید به قالب شمسی YYYY/MM/DD باشد.");
        }
    }
    if input.base_salary < 0 {
        errs.add("baseSalary", "حقوق پایه نمی‌تواند منفی باشد.");
    }
    if !input.email.is_empty() && !looks_like_email(&input.email) {
        errs.add("email", "قالب رایانامه معتبر نیست.");
    }
    if !input.phone.is_empty() && input.phone.chars().filter(|c| c.is_ascii_digit()).count() < 8 {
        errs.add("phone", "شمارهٔ تماس باید حداقل ۸ رقم داشته باشد.");
    }
    errs
}

fn looks_like_email(s: &str) -> bool {
    let mut parts = s.split('@');
    match (parts.next(), parts.next(), parts.next()) {
        (Some(local), Some(domain), None) => !local.is_empty() && domain.contains('.') && !domain.starts_with('.'),
        _ => false,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn valid_input() -> EmployeeInput {
        EmployeeInput {
            full_name: "نگار حسینی".into(),
            national_id: "0034567895".into(),
            department_id: "D4".into(),
            position: "کارشناس فروش".into(),
            contract_type: ContractType::FullTime,
            hire_date: "1401/11/20".into(),
            status: None,
            base_salary: 34_000_000,
            phone: "0912-0000003".into(),
            email: "negar.h@demo.ir".into(),
        }
    }

    #[test]
    fn accepts_valid_input() {
        assert!(validate_employee_input(&valid_input()).is_empty());
    }

    #[test]
    fn national_id_check_digit() {
        assert!(is_valid_national_id("0034567895"));
        assert!(!is_valid_national_id("0034567890"), "رقم کنترل نادرست باید رد شود");
        assert!(!is_valid_national_id("1111111111"), "کدهای تکراری معتبر نیستند");
        assert!(!is_valid_national_id("003456789"), "کد کوتاه معتبر نیست");
        assert!(!is_valid_national_id("۰۰۳۴۵۶۷۸۹۵"), "ارقام فارسی در کد ملی پذیرفته نمی‌شود");
    }

    #[test]
    fn rejects_bad_national_id_and_date() {
        let mut input = valid_input();
        input.national_id = "1111111111".into();
        input.hire_date = "2024-03-20".into();
        let errs = validate_employee_input(&input);
        assert!(errs.0.contains_key("nationalId"));
        assert!(!errs.0.contains_key("hireDate"), "قالب میلادی هم با جداکنندهٔ - پذیرفته می‌شود");

        input.hire_date = "1404/13/01".into();
        let errs = validate_employee_input(&input);
        assert!(errs.0.contains_key("hireDate"));
    }

    #[test]
    fn contract_type_round_trip() {
        for t in [ContractType::FullTime, ContractType::Contractor, ContractType::PartTime] {
            assert_eq!(ContractType::parse(t.fa()), Some(t));
        }
        assert_eq!(EmployeeStatus::parse("probation"), Some(EmployeeStatus::Probation));
        assert_eq!(EmployeeStatus::OnLeave.key(), "leave");
    }
}
