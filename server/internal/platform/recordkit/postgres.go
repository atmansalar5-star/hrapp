// کوئری‌های PostgreSQL رکوردکیت — همه باید پارامتری‌شده باشند (آزمون SQLi).

package recordkit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hrms/internal/contracts"
)

// PostgresStore انبار رکوردهای گردش‌کاری روی PostgreSQL — همان قرارداد MemoryStore.
//
// جدول مشترک `hr_records.record` است: چون همهٔ این ماژول‌ها یک الگوی یکسان دارند، یک
// جدول با ستون `module` و `kind` از تکرار schema جلوگیری می‌کند. جداسازی داده بین
// شرکت‌های مستقر در نسخهٔ چندشرکتی با همین ستون `module`/پارامتر استقرار انجام می‌شود.
type PostgresStore struct {
	db    *sql.DB
	clock func() time.Time
}

// NewPostgresStore انبار Postgres می‌سازد.
func NewPostgresStore(db *sql.DB, clock func() time.Time) *PostgresStore {
	if clock == nil {
		clock = time.Now
	}
	return &PostgresStore{db: db, clock: clock}
}

const recordColumns = `id, module, kind, title, employee_id, employee_name, department_name,
	status, fields, note_fa, created_by, decided_by, version`

// Get خواندن یک رکورد همراه با سابقهٔ رخدادها.
func (s *PostgresStore) Get(ctx context.Context, id string) (Record, bool, error) {
	row := s.db.QueryRowContext(ctx, `select `+recordColumns+` from hr_records.record where id = $1`, id)
	r, err := scanRecord(row)
	if err == sql.ErrNoRows {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	events, err := s.events(ctx, id)
	if err != nil {
		return Record{}, false, err
	}
	r.Events = events
	return r, true, nil
}

// List فهرست رکوردها با فیلتر، جستجو و محدودهٔ دسترسی نقش (اعمال محدوده در SQL).
func (s *PostgresStore) List(ctx context.Context, f contracts.RecordFilter) (Page, error) {
	conds := []string{}
	args := []any{}
	ph := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.Module != "" {
		conds = append(conds, "module = "+ph(f.Module))
	}
	if f.Kind != "" {
		conds = append(conds, "kind = "+ph(f.Kind))
	}
	if f.Status != "" {
		conds = append(conds, "status = "+ph(f.Status))
	}
	if f.EmployeeID != "" {
		conds = append(conds, "employee_id = "+ph(f.EmployeeID))
	}
	switch f.Scope {
	case "own":
		conds = append(conds, "employee_id = "+ph(f.ActorEmployeeID))
	case "own_unit":
		if f.ActorDepartment != "" {
			conds = append(conds, "department_name = "+ph(f.ActorDepartment))
		}
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + q + "%"
		conds = append(conds, "(title ilike "+ph(like)+" or id ilike "+ph(like)+
			" or employee_name ilike "+ph(like)+" or fields::text ilike "+ph(like)+")")
	}
	cond := "true"
	if len(conds) > 0 {
		cond = strings.Join(conds, " and ")
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `select count(*) from hr_records.record where `+cond, args...).Scan(&total); err != nil {
		return Page{}, err
	}
	page, size := normalizePaging(f.Page, f.Size)
	offset := (page - 1) * size
	query := `select ` + recordColumns + ` from hr_records.record where ` + cond +
		fmt.Sprintf(" order by id desc limit %d offset %d", size, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	items := []Record{}
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return Page{}, err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	// سابقهٔ رخداد برای هر رکورد جدا خوانده می‌شود؛ فهرست‌ها کوتاه‌اند و این ساده‌تر از
	// ساخت یک کوئری تجمعی پیچیده است.
	for i := range items {
		events, err := s.events(ctx, items[i].ID)
		if err != nil {
			return Page{}, err
		}
		items[i].Events = events
	}
	return Page{Items: items, Total: total, Page: page, Size: size}, nil
}

// Save درج یا به‌روزرسانی رکورد و بازنویسی سابقهٔ رخدادها.
func (s *PostgresStore) Save(ctx context.Context, r Record) (Record, error) {
	raw, err := json.Marshal(r.Fields)
	if err != nil {
		return Record{}, err
	}
	var version int
	err = s.db.QueryRowContext(ctx, `
		insert into hr_records.record (`+recordColumns+`)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,1)
		on conflict (id) do update set
			module = excluded.module,
			kind = excluded.kind, title = excluded.title, employee_id = excluded.employee_id,
			employee_name = excluded.employee_name, department_name = excluded.department_name,
			status = excluded.status, fields = excluded.fields, note_fa = excluded.note_fa,
			decided_by = excluded.decided_by,
			version = hr_records.record.version + 1, updated_at = now()
		returning version`,
		r.ID, r.Module, r.Kind, r.Title, r.EmployeeID, r.EmployeeName, r.DepartmentName,
		r.Status, string(raw), r.NoteFa, r.CreatedBy, r.DecidedBy).Scan(&version)
	if err != nil {
		return Record{}, err
	}
	r.Version = version
	if err := s.replaceEvents(ctx, r.ID, r.Events); err != nil {
		return Record{}, err
	}
	return r, nil
}

// Delete حذف رکورد و رخدادهایش (با cascade).
func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `delete from hr_records.record where id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// NextID شناسهٔ تازه از شمارندهٔ مستقل هر نوع رکورد.
func (s *PostgresStore) NextID(kind, prefix string) (string, error) {
	var n int64
	err := s.db.QueryRowContext(context.Background(), `
		insert into hr_records.seq (key, n) values ($1, 1)
		on conflict (key) do update set n = hr_records.seq.n + 1
		returning n`, kind).Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%04d", prefix, n), nil
}

func (s *PostgresStore) events(ctx context.Context, id string) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`select at, actor, action, note_fa from hr_records.record_event where record_id = $1 order by at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.At, &e.Actor, &e.Action, &e.NoteFa); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *PostgresStore) replaceEvents(ctx context.Context, id string, events []Event) error {
	if _, err := s.db.ExecContext(ctx, `delete from hr_records.record_event where record_id = $1`, id); err != nil {
		return err
	}
	for _, e := range events {
		if _, err := s.db.ExecContext(ctx,
			`insert into hr_records.record_event (record_id, at, actor, action, note_fa) values ($1,$2,$3,$4,$5)`,
			id, e.At, e.Actor, e.Action, e.NoteFa); err != nil {
			return err
		}
	}
	return nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanRecord(row rowScanner) (Record, error) {
	var r Record
	var raw string
	err := row.Scan(&r.ID, &r.Module, &r.Kind, &r.Title, &r.EmployeeID, &r.EmployeeName, &r.DepartmentName,
		&r.Status, &raw, &r.NoteFa, &r.CreatedBy, &r.DecidedBy, &r.Version)
	if err != nil {
		return Record{}, err
	}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &r.Fields); err != nil {
			r.Fields = map[string]string{}
		}
	}
	return r, nil
}
