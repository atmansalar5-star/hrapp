// Package synclog دفتر تغییرات سرور برای هم‌گام‌سازی کلاینت آفلاین.
//
// سرور تنها صادرکنندهٔ شمارهٔ ترتیب (`seq`) است؛ کلاینت با نشانگر خود
// (`GET /api/v1/sync/changes?since=N`) دلتای تغییرات را می‌گیرد و عملیات ثبت‌شده در
// صف آفلاین را با `POST /api/v1/sync/mutations` می‌فرستد.
//
// تضمین‌های این پکیج:
//   - اعمال عملیات **ایدمپوتنت** است: تکرار یک `opId` رخداد تازه نمی‌سازد.
//   - عملیات کهنه (`baseVersion` قدیمی‌تر از نسخهٔ سرور) با `version_conflict` رد
//     می‌شود تا ویرایش محلی، تغییر سرور را بی‌صدا پاک نکند.
//   - حذف به‌صورت tombstone منتشر می‌شود تا کلاینت‌های دیگر هم مطلع شوند.
package synclog

import (
	"fmt"
	"sort"
	"sync"

	"hrms/internal/contracts"
)

// عملیات‌های مجاز.
const (
	OpUpsert = "upsert"
	OpDelete = "delete"
)

// Change یک تغییر در دفتر.
type Change struct {
	Seq     int            `json:"seq"`
	Entity  string         `json:"entity"`
	Op      string         `json:"op"`
	ID      string         `json:"id"`
	Version int            `json:"version"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Log دفتر تغییرات در حافظه با نسخه‌بندی هر رکورد.
type Log struct {
	mu       sync.Mutex
	seq      int
	changes  []Change
	versions map[string]int
	applied  map[string]bool
}

// New دفتر خالی می‌سازد.
func New() *Log {
	return &Log{
		versions: make(map[string]int),
		applied:  make(map[string]bool),
	}
}

func key(entity, id string) string { return entity + "/" + id }

// Seq آخرین شمارهٔ ترتیب صادرشده.
func (l *Log) Seq() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.seq
}

// Since دلتای تغییرات پس از نشانگر کلاینت.
func (l *Log) Since(since int) contracts.ChangeSet {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]contracts.ChangeItem, 0, len(l.changes))
	for _, c := range l.changes {
		if c.Seq <= since {
			continue
		}
		out = append(out, contracts.ChangeItem{
			Entity:  c.Entity,
			Op:      c.Op,
			ID:      c.ID,
			Version: c.Version,
			Payload: c.Payload,
		})
	}
	return contracts.ChangeSet{Since: since, NextSeq: l.seq, Changes: out}
}

// Record ثبت یک تغییر سمت سرور (پس از اعمال در ماژول مربوطه).
func (l *Log) Record(entity, op, id string, version int, payload map[string]any) Change {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.seq++
	c := Change{Seq: l.seq, Entity: entity, Op: op, ID: id, Version: version, Payload: payload}
	l.changes = append(l.changes, c)
	l.versions[key(entity, id)] = version
	return c
}

// Version نسخهٔ فعلی یک رکورد.
func (l *Log) Version(entity, id string) (int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.versions[key(entity, id)]
	return v, ok
}

// Count تعداد تغییرات ثبت‌شده.
func (l *Log) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.changes)
}

// Apply دسته‌ای عملیات آفلاین را اعمال می‌کند و نتیجهٔ هر عملیات را برمی‌گرداند.
func (l *Log) Apply(mutations []contracts.Mutation) contracts.MutationResult {
	result := contracts.MutationResult{Accepted: []string{}, Rejected: []contracts.RejectedMutation{}}

	for _, m := range mutations {
		// اعتبارسنجی ساختاری
		if m.OpID == "" || m.ID == "" || m.Entity == "" {
			result.Rejected = append(result.Rejected, contracts.RejectedMutation{
				OpID:    m.OpID,
				Reason:  contracts.CodeValidationFailed,
				Message: "شناسهٔ عملیات، موجودیت و شناسهٔ رکورد الزامی است.",
			})
			continue
		}

		l.mu.Lock()
		// ایدمپوتنسی: همان opId دوباره اعمال نمی‌شود.
		if l.applied[m.OpID] {
			l.mu.Unlock()
			result.Accepted = append(result.Accepted, m.OpID)
			continue
		}
		current, exists := l.versions[key(m.Entity, m.ID)]
		l.mu.Unlock()

		switch m.Op {
		case "create":
			if exists {
				result.Rejected = append(result.Rejected, contracts.RejectedMutation{
					OpID:          m.OpID,
					Reason:        contracts.CodeConflict,
					ServerVersion: current,
					Message:       "این رکورد پیش‌تر در سرور ساخته شده است.",
				})
				continue
			}
			l.markApplied(m.OpID)
			l.Record(m.Entity, OpUpsert, m.ID, 1, m.Payload)

		case "update":
			if !exists {
				result.Rejected = append(result.Rejected, contracts.RejectedMutation{
					OpID:    m.OpID,
					Reason:  contracts.CodeNotFound,
					Message: "رکورد موردنظر در سرور یافت نشد.",
				})
				continue
			}
			if m.BaseVersion > 0 && m.BaseVersion != current {
				result.Rejected = append(result.Rejected, contracts.RejectedMutation{
					OpID:          m.OpID,
					Reason:        contracts.CodeConflict,
					ServerVersion: current,
					Message:       fmt.Sprintf("رکورد در سرور تغییر کرده است (نسخهٔ سرور %d).", current),
				})
				continue
			}
			l.markApplied(m.OpID)
			l.Record(m.Entity, OpUpsert, m.ID, current+1, m.Payload)

		case "delete":
			if !exists {
				result.Rejected = append(result.Rejected, contracts.RejectedMutation{
					OpID:    m.OpID,
					Reason:  contracts.CodeNotFound,
					Message: "رکورد موردنظر برای حذف یافت نشد.",
				})
				continue
			}
			l.markApplied(m.OpID)
			l.Record(m.Entity, OpDelete, m.ID, current+1, nil)

		default:
			result.Rejected = append(result.Rejected, contracts.RejectedMutation{
				OpID:    m.OpID,
				Reason:  contracts.CodeValidationFailed,
				Message: "نوع عملیات باید create، update یا delete باشد.",
			})
			continue
		}

		result.Accepted = append(result.Accepted, m.OpID)
	}

	result.NextSeq = l.Seq()
	return result
}

func (l *Log) markApplied(opID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.applied[opID] = true
}

// Replay ترتیب تغییرات را برای بازرسی برمی‌گرداند.
func (l *Log) Replay() []Change {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Change, len(l.changes))
	copy(out, l.changes)
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}
