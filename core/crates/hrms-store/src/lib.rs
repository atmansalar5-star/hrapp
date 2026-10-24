//! انبار محلی کلاینت.
//!
//! کلاینت باید بتواند بدون شبکه کار کند: خواندن از انبار محلی، ثبت تغییر در Outbox و
//! سپس هم‌گام‌سازی با سرور. سه بخش این crate:
//!
//! * [`outbox`] — صف عملیات ثبت‌شده در حالت آفلاین با تضمین ایدمپوتنت بودن (`op_id`).
//! * [`conflict`] — سیاست حل تعارض به‌ازای هر موجودیت (حقوق: سرور برنده، مرخصی: ادغام،
//!   آدیت: فقط-افزودن).
//! * [`Store`] — قرارداد انبار؛ پیاده‌سازی مرجع در حافظه و پیاده‌سازی SQLite پشت
//!   پرچم ویژگی `sqlite` (برای اندروید با SQLCipher فعال می‌شود).
//!
//! واگذاریِ شمارهٔ ترتیب (`seq`) فقط در اختیار سرور است؛ کلاینت هرگز seq نمی‌سازد.

pub mod conflict;
pub mod outbox;
pub mod schema;

#[cfg(feature = "sqlite")]
pub mod sqlite;

use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

/// یک رکورد عمومی در انبار محلی.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Record {
    pub entity: String,
    pub id: String,
    pub version: u32,
    pub payload: serde_json::Value,
    /// شمارهٔ ترتیب سرور؛ صفر یعنی رکورد هنوز روی سرور ثبت نشده است.
    pub server_seq: i64,
    /// حذف‌شده (tombstone) — برای انتشار حذف در سینک لازم است.
    pub deleted: bool,
}

/// خطاهای انبار.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum StoreError {
    NotFound,
    VersionConflict { expected: u32, actual: u32 },
    Backend(String),
}

impl std::fmt::Display for StoreError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            StoreError::NotFound => write!(f, "رکورد در انبار محلی یافت نشد."),
            StoreError::VersionConflict { expected, actual } => {
                write!(f, "نسخهٔ رکورد تغییر کرده است (انتظار {expected}، نسخهٔ فعلی {actual}).")
            }
            StoreError::Backend(m) => write!(f, "خطای انبار محلی: {m}"),
        }
    }
}

impl std::error::Error for StoreError {}

/// قرارداد انبار محلی. هر پیاده‌سازی (حافظه/SQLite) باید همین رفتار را رعایت کند.
pub trait Store {
    /// خواندن یک رکورد.
    fn get(&self, entity: &str, id: &str) -> Result<Option<Record>, StoreError>;
    /// نوشتن رکورد با بررسی نسخه (خوش‌بینانه).
    fn put(&mut self, record: Record) -> Result<Record, StoreError>;
    /// حذف نرم (tombstone).
    fn delete(&mut self, entity: &str, id: &str) -> Result<Record, StoreError>;
    /// تغییرات محلی که هنوز به سرور نرسیده‌اند.
    fn dirty(&self) -> Vec<Record>;
    /// اعمال تغییر آمده از سرور (بدون افزایش نسخهٔ محلیِ کثیف).
    fn apply_server_change(&mut self, record: Record) -> Result<(), StoreError>;
    /// نشانگر سینک ذخیره‌شده.
    fn cursor(&self) -> i64;
    /// ثبت نشانگر سینک پس از هم‌گام‌سازی موفق.
    fn set_cursor(&mut self, seq: i64);
}

/// پیاده‌سازی مرجع در حافظه — برای آزمون و اجرای دسکتاپ.
#[derive(Debug, Default)]
pub struct MemoryStore {
    records: BTreeMap<(String, String), Record>,
    cursor: i64,
}

impl MemoryStore {
    /// انبار خالی.
    pub fn new() -> Self {
        Self::default()
    }

    /// تعداد رکوردهای زنده (بدون tombstone).
    pub fn len(&self) -> usize {
        self.records.values().filter(|r| !r.deleted).count()
    }

    /// آیا انبار خالی است.
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }
}

impl Store for MemoryStore {
    fn get(&self, entity: &str, id: &str) -> Result<Option<Record>, StoreError> {
        Ok(self.records.get(&(entity.to_string(), id.to_string())).cloned())
    }

    fn put(&mut self, mut record: Record) -> Result<Record, StoreError> {
        let key = (record.entity.clone(), record.id.clone());
        match self.records.get(&key) {
            Some(existing) => {
                // نسخهٔ ارسالی کلاینت باید با نسخهٔ محلی بخواند؛ وگرنه تعارض داریم.
                if record.version != 0 && record.version != existing.version {
                    return Err(StoreError::VersionConflict { expected: record.version, actual: existing.version });
                }
                record.version = existing.version + 1;
                record.server_seq = existing.server_seq;
            }
            None => {
                record.version = 1;
                record.server_seq = 0;
            }
        }
        record.deleted = false;
        self.records.insert(key, record.clone());
        Ok(record)
    }

    fn delete(&mut self, entity: &str, id: &str) -> Result<Record, StoreError> {
        let key = (entity.to_string(), id.to_string());
        let existing = self.records.get_mut(&key).ok_or(StoreError::NotFound)?;
        existing.deleted = true;
        existing.version += 1;
        Ok(existing.clone())
    }

    fn dirty(&self) -> Vec<Record> {
        let mut out: Vec<Record> = self.records.values().filter(|r| r.server_seq == 0).cloned().collect();
        out.sort_by(|a, b| (a.entity.clone(), a.id.clone()).cmp(&(b.entity.clone(), b.id.clone())));
        out
    }

    fn apply_server_change(&mut self, record: Record) -> Result<(), StoreError> {
        let key = (record.entity.clone(), record.id.clone());
        // نسخهٔ سرور همیشه برندهٔ نسخهٔ محلی است، ولی رکورد کثیف محلی دست‌نخورده می‌ماند.
        if let Some(local) = self.records.get(&key) {
            if local.server_seq == 0 && !local.deleted {
                return Ok(());
            }
        }
        self.records.insert(key, record);
        Ok(())
    }

    fn cursor(&self) -> i64 {
        self.cursor
    }

    fn set_cursor(&mut self, seq: i64) {
        if seq > self.cursor {
            self.cursor = seq;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    fn record(id: &str, name: &str) -> Record {
        Record {
            entity: "Employee".into(),
            id: id.into(),
            version: 0,
            payload: json!({ "fullName": name }),
            server_seq: 0,
            deleted: false,
        }
    }

    #[test]
    fn put_increments_version_and_marks_dirty() {
        let mut store = MemoryStore::new();
        let saved = store.put(record("EMP-001", "سارا محمدی")).unwrap();
        assert_eq!(saved.version, 1);
        assert_eq!(store.dirty().len(), 1);

        let saved2 = store.put(Record { version: 1, ..record("EMP-001", "سارا محمدی") }).unwrap();
        assert_eq!(saved2.version, 2);
    }

    #[test]
    fn stale_version_is_rejected() {
        let mut store = MemoryStore::new();
        store.put(record("EMP-001", "سارا محمدی")).unwrap();
        let err = store.put(Record { version: 7, ..record("EMP-001", "سارا محمدی") }).unwrap_err();
        assert_eq!(err, StoreError::VersionConflict { expected: 7, actual: 1 });
    }

    #[test]
    fn server_changes_do_not_overwrite_dirty_local_records() {
        let mut store = MemoryStore::new();
        store.put(record("EMP-001", "ویرایش محلی")).unwrap();
        let server = Record { version: 9, server_seq: 42, ..record("EMP-001", "نسخهٔ سرور") };
        store.apply_server_change(server).unwrap();
        let local = store.get("Employee", "EMP-001").unwrap().unwrap();
        assert_eq!(local.payload["fullName"], "ویرایش محلی", "کار محلی نباید بی‌اعلام پاک شود");
        assert_eq!(store.dirty().len(), 1);
    }

    #[test]
    fn cursor_only_moves_forward() {
        let mut store = MemoryStore::new();
        store.set_cursor(10);
        store.set_cursor(4);
        assert_eq!(store.cursor(), 10);
    }

    #[test]
    fn delete_creates_tombstone() {
        let mut store = MemoryStore::new();
        store.put(record("EMP-002", "آرش رضایی")).unwrap();
        let deleted = store.delete("Employee", "EMP-002").unwrap();
        assert!(deleted.deleted);
        assert_eq!(store.len(), 0);
    }
}
