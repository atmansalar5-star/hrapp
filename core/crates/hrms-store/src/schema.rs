//! شِمای انبار محلی کلاینت (SQLite).
//!
//! یک انبار واحد با کلید مرکب (entity, id) تا همهٔ ماژول‌ها بتوانند روی یک فایل کار
//! کنند، به‌همراه جدول‌های اختصاصی سینک و حسابرسی محلی. رمزنگاری در حالت واقعی با
//! SQLCipher انجام می‌شود؛ کلید از Keystore سیستم‌عامل می‌آید و هرگز در کد نمی‌نشیند.

/// نسخهٔ شِمای انبار محلی؛ با هر تغییر ساختار باید افزایش یابد.
pub const SCHEMA_VERSION: i32 = 1;

/// دستورهای ساخت شِما (اجرای مکرر بی‌خطر است).
pub const DDL: &str = r#"
pragma journal_mode = WAL;
pragma foreign_keys = ON;

create table if not exists schema_meta (
  key   text primary key,
  value text not null
);

create table if not exists record (
  entity      text    not null,
  id          text    not null,
  version     integer not null default 1,
  payload     text    not null,
  server_seq  integer not null default 0,
  deleted     integer not null default 0,
  updated_at  text    not null default (datetime('now')),
  primary key (entity, id)
);

-- رکوردهایی که هنوز روی سرور ثبت نشده‌اند، به ترتیب تغییر خوانده می‌شوند.
create index if not exists record_dirty_idx on record (server_seq, entity, id);

create table if not exists sync_outbox (
  op_id        text primary key,
  entity       text    not null,
  op           text    not null check (op in ('create','update','delete')),
  entity_id    text    not null,
  payload      text    not null,
  base_version integer not null default 0,
  client_ts    text    not null,
  state        text    not null default 'pending' check (state in ('pending','sent','rejected')),
  reason       text,
  created_at   text    not null default (datetime('now'))
);

create index if not exists sync_outbox_pending_idx on sync_outbox (state, created_at);

-- لاگ حسابرسی محلی: فقط-افزودن، با زنجیرهٔ هش برای اثبات دست‌نخورده بودن.
create table if not exists audit_local (
  seq       integer primary key autoincrement,
  at        text not null,
  actor     text not null,
  module    text not null,
  action    text not null,
  target    text,
  detail    text,
  prev_hash text not null,
  hash      text not null
);

insert or ignore into schema_meta (key, value) values ('schema_version', '1');
insert or ignore into schema_meta (key, value) values ('sync_cursor', '0');
"#;

/// کوئری‌های پرکاربرد انبار (برای جلوگیری از پخش‌شدن SQL در کد).
pub mod queries {
    /// خواندن یک رکورد.
    pub const GET: &str = "select entity, id, version, payload, server_seq, deleted from record where entity = ?1 and id = ?2";
    /// خواندن رکوردهای کثیف (ثبت‌نشده روی سرور).
    pub const DIRTY: &str = "select entity, id, version, payload, server_seq, deleted from record where server_seq = 0 order by entity, id";
    /// درج یا به‌روزرسانی رکورد.
    pub const UPSERT: &str = "insert into record (entity, id, version, payload, server_seq, deleted) values (?1, ?2, ?3, ?4, ?5, ?6)
        on conflict(entity, id) do update set version = excluded.version, payload = excluded.payload, server_seq = excluded.server_seq, deleted = excluded.deleted, updated_at = datetime('now')";
    /// ثبت عملیات در Outbox (تکراری‌ها نادیده گرفته می‌شوند).
    pub const ENQUEUE: &str = "insert or ignore into sync_outbox (op_id, entity, op, entity_id, payload, base_version, client_ts, state) values (?1, ?2, ?3, ?4, ?5, ?6, ?7, 'pending')";
    /// عملیات‌های در انتظار، به ترتیب ثبت.
    pub const PENDING: &str = "select op_id, entity, op, entity_id, payload, base_version, client_ts, state, reason from sync_outbox where state = 'pending' order by created_at, op_id";
    /// نشانگر سینک ذخیره‌شده.
    pub const CURSOR: &str = "select value from schema_meta where key = 'sync_cursor'";
    /// ثبت نشانگر سینک.
    pub const SET_CURSOR: &str = "insert into schema_meta (key, value) values ('sync_cursor', ?1) on conflict(key) do update set value = excluded.value";
}
