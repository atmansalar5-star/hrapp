// انبار Postgres صندوق خروجی.
//
// جدولش در مهاجرت `003_event_outbox.sql` ساخته می‌شود (schema `hr_platform`). عمداً در
// یک schema مستقل از ماژول‌ها است: رخداد، دارایی پلتفرم است و هیچ ماژولی نباید مالک
// آن باشد؛ همان‌گونه که لاگ حسابرسی هم مال هیچ ماژولی نیست.

package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// PostgresStore انبار پایدار صندوق خروجی.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore انبار پایدار می‌سازد؛ db نباید nil باشد.
func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

const insertSQL = `
insert into hr_platform.event_outbox
  (event_id, name, occurred_at, actor, role, module, method, path, status, subject, payload, next_attempt_at)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$3)
on conflict (event_id) do nothing`

// Append رخداد را ثبت می‌کند؛ تکرار شناسه بی‌اثر است.
func (s *PostgresStore) Append(ctx context.Context, e Event) error {
	payload := e.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("سریال‌سازی دادهٔ رخداد ناموفق: %w", err)
	}
	at := e.OccurredAt
	if at.IsZero() {
		at = time.Now()
	}
	_, err = s.db.ExecContext(ctx, insertSQL,
		e.ID, e.Name, at, e.Actor, e.Role, e.Module, e.Method, e.Path, e.Status, e.Subject, raw)
	if err != nil {
		return fmt.Errorf("ثبت رخداد ناموفق: %w", err)
	}
	return nil
}

const dueSQL = `
select id, event_id, name, occurred_at, actor, role, module, method, path, status,
       subject, payload, attempts, next_attempt_at, delivered_at, last_error, status_delivery
  from hr_platform.event_outbox
 where status_delivery = 'pending' and next_attempt_at <= $1
 order by next_attempt_at
 limit $2`

// Due ردیف‌های سررسیدشدهٔ در انتظار را برمی‌گرداند.
func (s *PostgresStore) Due(ctx context.Context, now time.Time, limit int) ([]Delivery, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, dueSQL, now, limit)
	if err != nil {
		return nil, fmt.Errorf("خواندن صف رخداد ناموفق: %w", err)
	}
	defer rows.Close()
	return scanDeliveries(rows)
}

// MarkDelivered ردیف را تحویل‌شده ثبت می‌کند.
func (s *PostgresStore) MarkDelivered(ctx context.Context, id int64, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`update hr_platform.event_outbox
		    set delivered_at = $2, status_delivery = 'delivered', last_error = ''
		  where id = $1`, id, at)
	if err != nil {
		return fmt.Errorf("ثبت تحویل رخداد ناموفق: %w", err)
	}
	return nil
}

// MarkAttempt تلاش ناموفق را ثبت می‌کند.
func (s *PostgresStore) MarkAttempt(ctx context.Context, id int64, attempts int, next time.Time, message, status string) error {
	if status == "" {
		status = StatusPending
	}
	_, err := s.db.ExecContext(ctx,
		`update hr_platform.event_outbox
		    set attempts = $2, next_attempt_at = $3, last_error = $4, status_delivery = $5
		  where id = $1`, id, attempts, next, message, status)
	if err != nil {
		return fmt.Errorf("ثبت تلاش ناموفق رخداد ناموفق: %w", err)
	}
	return nil
}

const recentSQL = `
select id, event_id, name, occurred_at, actor, role, module, method, path, status,
       subject, payload, attempts, next_attempt_at, delivered_at, last_error, status_delivery
  from hr_platform.event_outbox
 where ($1 = '' or status_delivery = $1)
 order by id desc
 limit $2`

// Recent آخرین ردیف‌ها را برمی‌گرداند.
func (s *PostgresStore) Recent(ctx context.Context, status string, limit int) ([]Delivery, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, recentSQL, status, limit)
	if err != nil {
		return nil, fmt.Errorf("خواندن فهرست رخدادها ناموفق: %w", err)
	}
	defer rows.Close()
	return scanDeliveries(rows)
}

// Requeue ردیف را برای تلاش دوباره به صف برمی‌گرداند.
func (s *PostgresStore) Requeue(ctx context.Context, id int64, at time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`update hr_platform.event_outbox
		    set attempts = 0, next_attempt_at = $2, delivered_at = null,
		        last_error = '', status_delivery = 'pending'
		  where id = $1`, id, at)
	if err != nil {
		return fmt.Errorf("بازگرداندن رخداد به صف ناموفق: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Counts شمار ردیف‌ها به تفکیک وضعیت.
func (s *PostgresStore) Counts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`select status_delivery, count(*) from hr_platform.event_outbox group by status_delivery`)
	if err != nil {
		return nil, fmt.Errorf("شمارش رخدادها ناموفق: %w", err)
	}
	defer rows.Close()
	out := map[string]int{StatusPending: 0, StatusDelivered: 0, StatusFailed: 0}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		out[status] = n
	}
	return out, rows.Err()
}

// scanDeliveries ردیف‌های کوئری را به Delivery تبدیل می‌کند.
func scanDeliveries(rows *sql.Rows) ([]Delivery, error) {
	out := []Delivery{}
	for rows.Next() {
		var (
			d         Delivery
			eventID   string
			name      string
			occurred  time.Time
			actor     string
			role      string
			module    string
			method    string
			path      string
			httpCode  int
			subject   string
			rawPayload []byte
			attempts  int
			next      time.Time
			delivered sql.NullTime
			lastError string
			status    string
		)
		if err := rows.Scan(&d.ID, &eventID, &name, &occurred, &actor, &role, &module, &method,
			&path, &httpCode, &subject, &rawPayload, &attempts, &next, &delivered, &lastError, &status); err != nil {
			return nil, err
		}
		d.Event = Event{
			ID: eventID, Name: name, OccurredAt: occurred, Actor: actor, Role: role,
			Module: module, Method: method, Path: path, Status: httpCode, Subject: subject,
		}
		if len(rawPayload) > 0 {
			var payload map[string]any
			if err := json.Unmarshal(rawPayload, &payload); err == nil {
				d.Event.Payload = payload
			}
		}
		d.Attempts = attempts
		d.NextAttemptAt = next
		d.LastError = lastError
		d.lastStatus = status
		if delivered.Valid {
			at := delivered.Time
			d.DeliveredAt = &at
		}
		out = append(out, d)
	}
	return out, rows.Err()
}