//! پیاده‌سازی SQLite انبار محلی (پشت پرچم ویژگی `sqlite`).
//!
//! برای اندروید و iOS این ماژول با SQLCipher ساخته می‌شود تا فایل انبار رمزنگاری‌شده
//! باشد؛ کلید از Keystore/Keychain خوانده می‌شود و هرگز در حافظهٔ جاوااسکریپت نمی‌آید.
//!
//! استعمال: cargo test -p hrms-store --features sqlite

use crate::outbox::{Op, OutboxEntry, OutboxState};
use crate::schema::{queries, DDL};
use crate::{Record, Store, StoreError};
use rusqlite::{params, Connection, OptionalExtension};

/// انبار محلی روی SQLite.
pub struct SqliteStore {
    conn: Connection,
}

fn backend(e: rusqlite::Error) -> StoreError {
    StoreError::Backend(e.to_string())
}

impl SqliteStore {
    /// باز کردن (یا ساخت) انبار در مسیر داده‌شده و اعمال شِما.
    pub fn open(path: &str) -> Result<Self, StoreError> {
        let conn = Connection::open(path).map_err(backend)?;
        conn.execute_batch(DDL).map_err(backend)?;
        Ok(Self { conn })
    }

    /// انبار موقت در حافظه (برای آزمون).
    pub fn open_in_memory() -> Result<Self, StoreError> {
        let conn = Connection::open_in_memory().map_err(backend)?;
        conn.execute_batch(DDL).map_err(backend)?;
        Ok(Self { conn })
    }

    /// ثبت عملیات آفلاین؛ اگر `op_id` تکراری باشد نادیده گرفته می‌شود.
    pub fn enqueue(&self, entry: &OutboxEntry) -> Result<bool, StoreError> {
        let payload = serde_json::to_string(&entry.payload).map_err(|e| StoreError::Backend(e.to_string()))?;
        let n = self
            .conn
            .execute(
                queries::ENQUEUE,
                params![
                    entry.op_id,
                    entry.entity,
                    entry.op.as_str(),
                    entry.entity_id,
                    payload,
                    entry.base_version,
                    entry.client_ts
                ],
            )
            .map_err(backend)?;
        Ok(n > 0)
    }

    /// عملیات‌های در انتظار ارسال.
    pub fn pending(&self) -> Result<Vec<OutboxEntry>, StoreError> {
        let mut stmt = self.conn.prepare(queries::PENDING).map_err(backend)?;
        let rows = stmt
            .query_map([], |row| {
                let op: String = row.get(2)?;
                let payload: String = row.get(4)?;
                let state: String = row.get(7)?;
                Ok(OutboxEntry {
                    op_id: row.get(0)?,
                    entity: row.get(1)?,
                    op: match op.as_str() {
                        "create" => Op::Create,
                        "delete" => Op::Delete,
                        _ => Op::Update,
                    },
                    entity_id: row.get(3)?,
                    payload: serde_json::from_str(&payload).unwrap_or(serde_json::Value::Null),
                    client_ts: row.get(6)?,
                    base_version: row.get(5)?,
                    state: match state.as_str() {
                        "sent" => OutboxState::Sent,
                        "rejected" => OutboxState::Rejected,
                        _ => OutboxState::Pending,
                    },
                    reason: row.get(8)?,
                })
            })
            .map_err(backend)?;
        rows.collect::<Result<Vec<_>, _>>().map_err(backend)
    }

    /// علامت‌گذاری عملیات‌های پذیرفته‌شده توسط سرور.
    pub fn mark_sent(&self, op_ids: &[String]) -> Result<usize, StoreError> {
        let mut n = 0;
        for id in op_ids {
            n += self
                .conn
                .execute("update sync_outbox set state = 'sent' where op_id = ?1", params![id])
                .map_err(backend)?;
        }
        Ok(n)
    }

    /// ثبت دلیل رد عملیات توسط سرور.
    pub fn mark_rejected(&self, op_id: &str, reason: &str) -> Result<(), StoreError> {
        self.conn
            .execute("update sync_outbox set state = 'rejected', reason = ?2 where op_id = ?1", params![op_id, reason])
            .map_err(backend)?;
        Ok(())
    }

    /// عملیات‌های پایان‌یافته را پاک می‌کند.
    pub fn prune_finished(&self) -> Result<usize, StoreError> {
        self.conn
            .execute("delete from sync_outbox where state <> 'pending'", [])
            .map_err(backend)
    }
}

impl Store for SqliteStore {
    fn get(&self, entity: &str, id: &str) -> Result<Option<Record>, StoreError> {
        let mut stmt = self.conn.prepare(queries::GET).map_err(backend)?;
        stmt.query_row(params![entity, id], |row| {
            let payload: String = row.get(3)?;
            Ok(Record {
                entity: row.get(0)?,
                id: row.get(1)?,
                version: row.get::<_, i64>(2)? as u32,
                payload: serde_json::from_str(&payload).unwrap_or(serde_json::Value::Null),
                server_seq: row.get(4)?,
                deleted: row.get::<_, i64>(5)? != 0,
            })
        })
        .optional()
        .map_err(backend)
    }

    fn put(&mut self, record: Record) -> Result<Record, StoreError> {
        let key = (record.entity.clone(), record.id.clone());
        let existing = self.get(&key.0, &key.1)?;
        let mut out = record.clone();
        match existing {
            Some(prev) => {
                if out.version != 0 && out.version != prev.version {
                    return Err(StoreError::VersionConflict { expected: out.version, actual: prev.version });
                }
                out.version = prev.version + 1;
                out.server_seq = prev.server_seq;
            }
            None => {
                out.version = 1;
                out.server_seq = 0;
            }
        }
        out.deleted = false;
        let payload = serde_json::to_string(&out.payload).map_err(|e| StoreError::Backend(e.to_string()))?;
        self.conn
            .execute(
                queries::UPSERT,
                params![out.entity, out.id, out.version as i64, payload, out.server_seq, 0i64],
            )
            .map_err(backend)?;
        Ok(out)
    }

    fn delete(&mut self, entity: &str, id: &str) -> Result<Record, StoreError> {
        let mut rec = self.get(entity, id)?.ok_or(StoreError::NotFound)?;
        rec.deleted = true;
        rec.version += 1;
        rec.server_seq = 0;
        let payload = serde_json::to_string(&rec.payload).map_err(|e| StoreError::Backend(e.to_string()))?;
        self.conn
            .execute(queries::UPSERT, params![rec.entity, rec.id, rec.version as i64, payload, 0i64, 1i64])
            .map_err(backend)?;
        Ok(rec)
    }

    fn dirty(&self) -> Vec<Record> {
        let mut out = Vec::new();
        let Ok(mut stmt) = self.conn.prepare(queries::DIRTY) else {
            return out;
        };
        let Ok(rows) = stmt.query_map([], |row| {
            let payload: String = row.get(3)?;
            Ok(Record {
                entity: row.get(0)?,
                id: row.get(1)?,
                version: row.get::<_, i64>(2)? as u32,
                payload: serde_json::from_str(&payload).unwrap_or(serde_json::Value::Null),
                server_seq: row.get(4)?,
                deleted: row.get::<_, i64>(5)? != 0,
            })
        }) else {
            return out;
        };
        for r in rows.flatten() {
            out.push(r);
        }
        out
    }

    fn apply_server_change(&mut self, record: Record) -> Result<(), StoreError> {
        if let Some(local) = self.get(&record.entity, &record.id)? {
            if local.server_seq == 0 && !local.deleted {
                return Ok(()); // تغییر محلی ثبت‌نشده حرمت دارد
            }
        }
        let payload = serde_json::to_string(&record.payload).map_err(|e| StoreError::Backend(e.to_string()))?;
        self.conn
            .execute(
                queries::UPSERT,
                params![
                    record.entity,
                    record.id,
                    record.version as i64,
                    payload,
                    record.server_seq,
                    if record.deleted { 1i64 } else { 0i64 }
                ],
            )
            .map_err(backend)?;
        Ok(())
    }

    fn cursor(&self) -> i64 {
        self.conn
            .query_row(queries::CURSOR, [], |row| row.get::<_, String>(0))
            .optional()
            .ok()
            .flatten()
            .and_then(|v| v.parse().ok())
            .unwrap_or(0)
    }

    fn set_cursor(&mut self, seq: i64) {
        let _ = self.conn.execute(queries::SET_CURSOR, params![seq.to_string()]);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    fn entry(op_id: &str) -> OutboxEntry {
        OutboxEntry {
            op_id: op_id.into(),
            entity: "Employee".into(),
            op: Op::Update,
            entity_id: "EMP-001".into(),
            payload: json!({ "position": "کارشناس ارشد" }),
            client_ts: "1404/07/05 09:30".into(),
            base_version: 3,
            state: OutboxState::Pending,
            reason: None,
        }
    }

    #[test]
    fn schema_applies_and_round_trips_a_record() {
        let mut store = SqliteStore::open_in_memory().unwrap();
        let saved = store
            .put(Record {
                entity: "Employee".into(),
                id: "EMP-001".into(),
                version: 0,
                payload: json!({ "fullName": "سارا محمدی" }),
                server_seq: 0,
                deleted: false,
            })
            .unwrap();
        assert_eq!(saved.version, 1);
        let back = store.get("Employee", "EMP-001").unwrap().unwrap();
        assert_eq!(back.payload["fullName"], "سارا محمدی");
        assert_eq!(store.dirty().len(), 1);
    }

    #[test]
    fn outbox_is_idempotent_on_sqlite_too() {
        let store = SqliteStore::open_in_memory().unwrap();
        assert!(store.enqueue(&entry("op-1")).unwrap());
        assert!(!store.enqueue(&entry("op-1")).unwrap(), "op_id تکراری ثبت نمی‌شود");
        assert_eq!(store.pending().unwrap().len(), 1);
        store.mark_rejected("op-1", "مانده کافی نیست").unwrap();
        assert_eq!(store.pending().unwrap().len(), 0);
        assert_eq!(store.prune_finished().unwrap(), 1);
    }

    #[test]
    fn cursor_is_persisted() {
        let mut store = SqliteStore::open_in_memory().unwrap();
        assert_eq!(store.cursor(), 0);
        store.set_cursor(128);
        assert_eq!(store.cursor(), 128);
    }
}
