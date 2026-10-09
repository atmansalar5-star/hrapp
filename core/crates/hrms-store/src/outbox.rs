//! صف عملیات آفلاین (Outbox).
//!
//! هر تغییر در حالت آفلاین به‌صورت یک «عملیات» با شناسهٔ یکتا (`op_id`) ثبت می‌شود.
//! سرور همین شناسه را برای ایدمپوتنت بودن استفاده می‌کند؛ اگر پاسخ شبکه گم شود و
//! کلاینت دوباره بفرستد، سرور عملیات را دوباره اعمال نمی‌کند.

use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

/// نوع عملیات.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum Op {
    Create,
    Update,
    Delete,
}

impl Op {
    /// کلید انتقالی روی سیم.
    pub fn as_str(&self) -> &'static str {
        match self {
            Op::Create => "create",
            Op::Update => "update",
            Op::Delete => "delete",
        }
    }
}

/// وضعیت عملیات در صف.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub enum OutboxState {
    Pending,
    Sent,
    Rejected,
}

impl OutboxState {
    /// کلید ذخیره‌سازی.
    pub fn as_str(&self) -> &'static str {
        match self {
            OutboxState::Pending => "pending",
            OutboxState::Sent => "sent",
            OutboxState::Rejected => "rejected",
        }
    }
}

/// یک عملیات در صف.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct OutboxEntry {
    pub op_id: String,
    pub entity: String,
    pub op: Op,
    pub entity_id: String,
    pub payload: serde_json::Value,
    /// زمان ثبت به وقت محلی (فقط برای نمایش؛ ترتیب نهایی را سرور تعیین می‌کند).
    pub client_ts: String,
    /// نسخهٔ رکورد در زمان ثبت، برای تشخیص تعارض.
    pub base_version: u32,
    pub state: OutboxState,
    /// دلیل رد شدن (پیام فارسی سرور) در صورت وجود.
    pub reason: Option<String>,
}

/// صف عملیات.
#[derive(Debug, Default)]
pub struct Outbox {
    entries: BTreeMap<String, OutboxEntry>,
    order: Vec<String>,
}

impl Outbox {
    /// صف خالی.
    pub fn new() -> Self {
        Self::default()
    }

    /// ثبت عملیات تازه. اگر `op_id` تکراری باشد، دوباره ثبت نمی‌شود (ایدمپوتنت).
    pub fn enqueue(&mut self, entry: OutboxEntry) -> bool {
        if self.entries.contains_key(&entry.op_id) {
            return false;
        }
        self.order.push(entry.op_id.clone());
        self.entries.insert(entry.op_id.clone(), entry);
        true
    }

    /// عملیات‌های در انتظار ارسال، به ترتیب ثبت.
    pub fn pending(&self) -> Vec<OutboxEntry> {
        self.order
            .iter()
            .filter_map(|id| self.entries.get(id))
            .filter(|e| e.state == OutboxState::Pending)
            .cloned()
            .collect()
    }

    /// علامت‌گذاری عملیات‌های ارسال‌شده به‌عنوان پذیرفته‌شده.
    pub fn mark_sent(&mut self, op_ids: &[String]) {
        for id in op_ids {
            if let Some(e) = self.entries.get_mut(id) {
                e.state = OutboxState::Sent;
            }
        }
    }

    /// علامت‌گذاری عملیات ردشده همراه با پیام دلیل.
    pub fn mark_rejected(&mut self, op_id: &str, reason: &str) {
        if let Some(e) = self.entries.get_mut(op_id) {
            e.state = OutboxState::Rejected;
            e.reason = Some(reason.to_string());
        }
    }

    /// پاک‌سازی عملیات‌های پایان‌یافته (پس از تأیید سرور و به‌روزرسانی رمزنگاری‌شده).
    pub fn prune_finished(&mut self) -> usize {
        let before = self.entries.len();
        self.order.retain(|id| match self.entries.get(id) {
            Some(e) => e.state == OutboxState::Pending,
            None => false,
        });
        self.entries.retain(|_, e| e.state == OutboxState::Pending);
        before - self.entries.len()
    }

    /// تعداد عملیات‌های در انتظار (برای نمایش نشان «در انتظار هم‌گام‌سازی» در UI).
    pub fn pending_count(&self) -> usize {
        self.entries.values().filter(|e| e.state == OutboxState::Pending).count()
    }

    /// عملیات‌های ردشده برای نمایش به کاربر.
    pub fn rejected(&self) -> Vec<OutboxEntry> {
        self.entries.values().filter(|e| e.state == OutboxState::Rejected).cloned().collect()
    }
}

/// ساخت شناسهٔ یکتا برای عملیات کلاینت.
pub fn new_op_id(entity: &str, entity_id: &str, client_ts: &str) -> String {
    format!("{entity}:{entity_id}:{client_ts}")
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
    fn enqueue_is_idempotent() {
        let mut q = Outbox::new();
        assert!(q.enqueue(entry("EMP-001:1")));
        assert!(!q.enqueue(entry("EMP-001:1")), "همان op_id دوباره ثبت نمی‌شود");
        assert_eq!(q.pending_count(), 1);
    }

    #[test]
    fn pending_preserves_order() {
        let mut q = Outbox::new();
        q.enqueue(entry("a"));
        q.enqueue(entry("b"));
        q.enqueue(entry("c"));
        let ids: Vec<String> = q.pending().iter().map(|e| e.op_id.clone()).collect();
        assert_eq!(ids, vec!["a", "b", "c"]);
    }

    #[test]
    fn mark_sent_and_rejected_then_prune() {
        let mut q = Outbox::new();
        q.enqueue(entry("a"));
        q.enqueue(entry("b"));
        q.mark_sent(&["a".to_string()]);
        q.mark_rejected("b", "مانده مرخصی کافی نیست.");
        assert_eq!(q.pending_count(), 0);
        assert_eq!(q.rejected().len(), 1);
        assert_eq!(q.rejected()[0].reason.as_deref(), Some("مانده مرخصی کافی نیست."));
        assert_eq!(q.prune_finished(), 2);
    }

    #[test]
    fn op_keys_match_wire_format() {
        assert_eq!(Op::Delete.as_str(), "delete");
        assert_eq!(OutboxState::Sent.as_str(), "sent");
        assert_eq!(new_op_id("Employee", "EMP-001", "1404/07/05"), "Employee:EMP-001:1404/07/05");
    }
}
