// انبار حافظه‌ای صندوق خروجی: برای اجرای بدون پایگاه‌داده و برای آزمون‌ها.
//
// رفتارش همان قرارداد Store است؛ فقط داده با خاموشی سرور از بین می‌رود — که در توسعه
// و آزمون پذیرفتنی است.

package events

import (
	"context"
	"database/sql"
	"sort"
	"sync"
	"time"
)

// MemoryStore انبار درون‌حافظه‌ای با سقف نگهداری برای جلوگیری از رشد بی‌پایان.
type MemoryStore struct {
	mu     sync.Mutex
	rows   []Delivery
	nextID int64
	// limit بیشینهٔ ردیف‌های نگهداری‌شده؛ قدیمی‌ترها دور ریخته می‌شوند.
	limit int
}

// NewMemoryStore انبار حافظه‌ای می‌سازد؛ limit صفر یعنی ۵۰۰ ردیف.
func NewMemoryStore(limit int) *MemoryStore {
	if limit <= 0 {
		limit = 500
	}
	return &MemoryStore{limit: limit}
}

// Append رخداد را ثبت می‌کند؛ شناسهٔ تکراری نادیده گرفته می‌شود.
func (s *MemoryStore) Append(_ context.Context, e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range s.rows {
		if row.Event.ID == e.ID {
			return nil
		}
	}
	s.nextID++
	s.rows = append(s.rows, Delivery{
		ID:            s.nextID,
		Event:         e,
		NextAttemptAt: e.OccurredAt,
	})
	if len(s.rows) > s.limit {
		s.rows = s.rows[len(s.rows)-s.limit:]
	}
	return nil
}

// Due ردیف‌های در انتظار و سررسیدشده را برمی‌گرداند.
func (s *MemoryStore) Due(_ context.Context, now time.Time, limit int) ([]Delivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Delivery, 0, limit)
	for _, row := range s.rows {
		if row.DeliveredAt != nil || row.status() != StatusPending {
			continue
		}
		if row.NextAttemptAt.After(now) {
			continue
		}
		out = append(out, row)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NextAttemptAt.Before(out[j].NextAttemptAt) })
	return out, nil
}

// MarkDelivered ردیف را تحویل‌شده ثبت می‌کند.
func (s *MemoryStore) MarkDelivered(_ context.Context, id int64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.rows {
		if s.rows[i].ID == id {
			delivered := at
			s.rows[i].DeliveredAt = &delivered
			s.rows[i].LastError = ""
			s.rows[i].lastStatus = StatusDelivered
			return nil
		}
	}
	return nil
}

// MarkAttempt تلاش ناموفق را ثبت می‌کند.
func (s *MemoryStore) MarkAttempt(_ context.Context, id int64, attempts int, next time.Time, message, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.rows {
		if s.rows[i].ID == id {
			s.rows[i].Attempts = attempts
			s.rows[i].NextAttemptAt = next
			s.rows[i].LastError = message
			s.rows[i].lastStatus = status
			return nil
		}
	}
	return nil
}

// Recent آخرین ردیف‌ها را برمی‌گرداند (تازه‌ترین اول).
func (s *MemoryStore) Recent(_ context.Context, status string, limit int) ([]Delivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Delivery, 0, len(s.rows))
	for i := len(s.rows) - 1; i >= 0; i-- {
		row := s.rows[i]
		if status != "" && row.status() != status {
			continue
		}
		out = append(out, row)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// Requeue ردیف را برای تلاش دوباره به صف برمی‌گرداند؛ ردیف ناشناخته sql.ErrNoRows
// می‌دهد تا رفتارش با انبار Postgres یکی باشد.
func (s *MemoryStore) Requeue(_ context.Context, id int64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.rows {
		if s.rows[i].ID == id {
			s.rows[i].Attempts = 0
			s.rows[i].NextAttemptAt = at
			s.rows[i].DeliveredAt = nil
			s.rows[i].LastError = ""
			s.rows[i].lastStatus = StatusPending
			return nil
		}
	}
	return sql.ErrNoRows
}

// Counts شمار ردیف‌ها به تفکیک وضعیت.
func (s *MemoryStore) Counts(_ context.Context) (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]int{StatusPending: 0, StatusDelivered: 0, StatusFailed: 0}
	for _, row := range s.rows {
		out[row.status()]++
	}
	return out, nil
}

// status وضعیت ردیف را از فیلدهایش می‌سازد.
func (d *Delivery) status() string {
	switch {
	case d.DeliveredAt != nil:
		return StatusDelivered
	case d.lastStatus != "":
		return d.lastStatus
	default:
		return StatusPending
	}
}