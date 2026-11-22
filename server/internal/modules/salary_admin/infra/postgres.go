// لایهٔ پایدارسازی PostgreSQL ماژول «salary_admin» — کوئری‌های پارامتری‌شده.

package infra

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/salary_admin/domain"
	"hrms/internal/platform/calendar"
)

// PostgresStore انبار احکام و گواهی‌ها روی PostgreSQL — همان قرارداد MemoryStore.
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

// decreeColumns ستون‌های حکم. ستون‌های wage_* اجزای ثابت مزد و ستون‌های climate_* ضرایب
// فوق‌العاده‌های اقلیمی همان حکم‌اند (مهاجرت ۰۰۰۳)؛ فیش حقوق مزد مبنا را از همین‌ها می‌سازد.
const decreeColumns = `id, employee_id, employee_name, department_name, position_title, job_group,
	base_salary, effective_date, reason_fa, status, supersedes_id, note_fa, created_by, decided_by, employment_type, version,
	climate_region, rank_wage, seniority_wage, post_allowance, sacrifice_allowance, climate_allowance,
	bad_climate_percent, deprivation_percent, region_hardship_percent, extra_time_percent, salary_repair,
	pay_gap, pay_gap_since, coalesce(pay_group, 0), coalesce(pay_model, '')`

const certificateColumns = `id, serial, employee_id, employee_name, department_name, position_title, kind,
	purpose_fa, requested_at, issued_at, delivered_at, status, note_fa, decided_by, version`

/* ------------------------------- حکم حقوق ------------------------------- */

// GetDecree خواندن یک حکم.
func (s *PostgresStore) GetDecree(ctx context.Context, id string) (domain.SalaryDecree, bool, error) {
	row := s.db.QueryRowContext(ctx, `select `+decreeColumns+` from hr_salary_admin.decree where id = $1`, id)
	d, err := scanDecree(row)
	if err == sql.ErrNoRows {
		return domain.SalaryDecree{}, false, nil
	}
	return d, err == nil, err
}

// NotifiedForEmployee حکم‌های ابلاغ‌شدهٔ کارمند.
func (s *PostgresStore) NotifiedForEmployee(ctx context.Context, employeeID string) ([]domain.SalaryDecree, error) {
	rows, err := s.db.QueryContext(ctx,
		`select `+decreeColumns+` from hr_salary_admin.decree where employee_id = $1 and status = $2`,
		employeeID, domain.DecreeNotified)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SalaryDecree{}
	for rows.Next() {
		d, err := scanDecree(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListDecrees فهرست احکام با فیلتر و محدودهٔ دسترسی نقش (اعمال محدوده در SQL).
func (s *PostgresStore) ListDecrees(ctx context.Context, f contracts.SalaryDecreeFilter) (contracts.SalaryDecreePage, error) {
	conds := []string{}
	args := []any{}
	ph := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	switch f.Scope {
	case "own":
		conds = append(conds, "employee_id = "+ph(f.ActorEmployeeID))
	case "own_unit":
		if f.ActorDepartment != "" {
			conds = append(conds, "department_name = "+ph(f.ActorDepartment))
		}
	}
	if f.EmployeeID != "" {
		conds = append(conds, "employee_id = "+ph(f.EmployeeID))
	}
	if f.Status != "" {
		conds = append(conds, "status = "+ph(f.Status))
	}
	if f.JobGroup != "" {
		conds = append(conds, "job_group = "+ph(f.JobGroup))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + q + "%"
		conds = append(conds, "(employee_name ilike "+ph(like)+" or id ilike "+ph(like)+" or job_group ilike "+ph(like)+")")
	}
	cond := "true"
	if len(conds) > 0 {
		cond = strings.Join(conds, " and ")
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `select count(*) from hr_salary_admin.decree where `+cond, args...).Scan(&total); err != nil {
		return contracts.SalaryDecreePage{}, err
	}
	page, size := normalizePaging(f.Page, f.Size)
	offset := (page - 1) * size
	query := `select ` + decreeColumns + ` from hr_salary_admin.decree where ` + cond +
		fmt.Sprintf(" order by id desc limit %d offset %d", size, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return contracts.SalaryDecreePage{}, err
	}
	defer rows.Close()
	items := []contracts.SalaryDecree{}
	for rows.Next() {
		d, err := scanDecree(rows)
		if err != nil {
			return contracts.SalaryDecreePage{}, err
		}
		items = append(items, d.ToSalaryDecree())
	}
	return contracts.SalaryDecreePage{Items: items, Total: total, Page: page, Size: size}, rows.Err()
}

// SaveDecree درج یا به‌روزرسانی حکم (upsert).
func (s *PostgresStore) SaveDecree(ctx context.Context, d domain.SalaryDecree) (domain.SalaryDecree, error) {
	var version int
	err := s.db.QueryRowContext(ctx, `
		insert into hr_salary_admin.decree (`+decreeColumns+`)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,1,
			$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30)
		on conflict (id) do update set
			employee_id = excluded.employee_id, employee_name = excluded.employee_name,
			department_name = excluded.department_name, position_title = excluded.position_title,
			job_group = excluded.job_group, base_salary = excluded.base_salary,
			effective_date = excluded.effective_date, reason_fa = excluded.reason_fa,
			status = excluded.status, supersedes_id = excluded.supersedes_id, note_fa = excluded.note_fa,
			created_by = excluded.created_by, decided_by = excluded.decided_by,
			employment_type = excluded.employment_type,
			climate_region = excluded.climate_region, rank_wage = excluded.rank_wage,
			seniority_wage = excluded.seniority_wage, post_allowance = excluded.post_allowance,
			sacrifice_allowance = excluded.sacrifice_allowance, climate_allowance = excluded.climate_allowance,
			bad_climate_percent = excluded.bad_climate_percent, deprivation_percent = excluded.deprivation_percent,
			region_hardship_percent = excluded.region_hardship_percent, extra_time_percent = excluded.extra_time_percent,
			salary_repair = excluded.salary_repair,
			pay_gap = excluded.pay_gap, pay_gap_since = excluded.pay_gap_since,
			pay_group = excluded.pay_group, pay_model = excluded.pay_model,
			version = hr_salary_admin.decree.version + 1, updated_at = now()
		returning version`,
		d.ID, d.EmployeeID, d.EmployeeName, d.DepartmentName, d.Position, d.JobGroup,
		d.BaseSalary, d.EffectiveDate, d.ReasonFa, d.Status, d.SupersedesID, d.NoteFa,
		d.CreatedBy, d.DecidedBy, d.EmploymentType,
		d.ClimateRegion, d.RankWage, d.SeniorityWage, d.PostAllowance, d.SacrificeAllowance,
		d.ClimateAllowance, d.BadClimatePercent, d.DeprivationPercent, d.RegionHardshipPercent,
		d.ExtraTimePercent, d.SalaryRepair, d.PayGap, d.PayGapSince, d.PayGroup, d.PayModel).Scan(&version)
	if err != nil {
		return domain.SalaryDecree{}, err
	}
	d.Version = version
	return d, nil
}

// DeleteDecree حذف حکم پیش‌نویس.
func (s *PostgresStore) DeleteDecree(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `delete from hr_salary_admin.decree where id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrDecreeNotFound
	}
	return nil
}

// NextDecreeID شناسهٔ حکم تازه از تسلسل پایگاه‌داده.
func (s *PostgresStore) NextDecreeID() (string, error) {
	var seq int
	if err := s.db.QueryRowContext(context.Background(),
		`select nextval('hr_salary_admin.decree_seq')`).Scan(&seq); err != nil {
		return "", err
	}
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("DEC-%d-%04d", d.Year, seq), nil
}

/* ------------------------------ گواهی اشتغال ------------------------------ */

// GetCertificate خواندن یک گواهی.
func (s *PostgresStore) GetCertificate(ctx context.Context, id string) (domain.EmploymentCertificate, bool, error) {
	row := s.db.QueryRowContext(ctx, `select `+certificateColumns+` from hr_salary_admin.certificate where id = $1`, id)
	c, err := scanCertificate(row)
	if err == sql.ErrNoRows {
		return domain.EmploymentCertificate{}, false, nil
	}
	return c, err == nil, err
}

// ListCertificates فهرست گواهی‌ها.
func (s *PostgresStore) ListCertificates(ctx context.Context, f contracts.CertificateFilter) (contracts.CertificatePage, error) {
	conds := []string{}
	args := []any{}
	ph := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	switch f.Scope {
	case "own":
		conds = append(conds, "employee_id = "+ph(f.ActorEmployeeID))
	case "own_unit":
		if f.ActorDepartment != "" {
			conds = append(conds, "department_name = "+ph(f.ActorDepartment))
		}
	}
	if f.EmployeeID != "" {
		conds = append(conds, "employee_id = "+ph(f.EmployeeID))
	}
	if f.Status != "" {
		conds = append(conds, "status = "+ph(f.Status))
	}
	if f.Kind != "" {
		conds = append(conds, "kind = "+ph(f.Kind))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + q + "%"
		conds = append(conds, "(employee_name ilike "+ph(like)+" or serial ilike "+ph(like)+" or id ilike "+ph(like)+")")
	}
	cond := "true"
	if len(conds) > 0 {
		cond = strings.Join(conds, " and ")
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `select count(*) from hr_salary_admin.certificate where `+cond, args...).Scan(&total); err != nil {
		return contracts.CertificatePage{}, err
	}
	page, size := normalizePaging(f.Page, f.Size)
	offset := (page - 1) * size
	query := `select ` + certificateColumns + ` from hr_salary_admin.certificate where ` + cond +
		fmt.Sprintf(" order by id desc limit %d offset %d", size, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return contracts.CertificatePage{}, err
	}
	defer rows.Close()
	items := []contracts.EmploymentCertificate{}
	for rows.Next() {
		c, err := scanCertificate(rows)
		if err != nil {
			return contracts.CertificatePage{}, err
		}
		items = append(items, c.ToCertificate())
	}
	return contracts.CertificatePage{Items: items, Total: total, Page: page, Size: size}, rows.Err()
}

// SaveCertificate درج یا به‌روزرسانی گواهی.
func (s *PostgresStore) SaveCertificate(ctx context.Context, c domain.EmploymentCertificate) (domain.EmploymentCertificate, error) {
	var version int
	err := s.db.QueryRowContext(ctx, `
		insert into hr_salary_admin.certificate (`+certificateColumns+`)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,1)
		on conflict (id) do update set
			serial = excluded.serial, employee_id = excluded.employee_id,
			employee_name = excluded.employee_name, department_name = excluded.department_name,
			position_title = excluded.position_title, kind = excluded.kind, purpose_fa = excluded.purpose_fa,
			requested_at = excluded.requested_at, issued_at = excluded.issued_at,
			delivered_at = excluded.delivered_at, status = excluded.status, note_fa = excluded.note_fa,
			decided_by = excluded.decided_by, version = hr_salary_admin.certificate.version + 1,
			updated_at = now()
		returning version`,
		c.ID, c.Serial, c.EmployeeID, c.EmployeeName, c.DepartmentName, c.Position, c.Kind,
		c.PurposeFa, c.RequestedAt, c.IssuedAt, c.DeliveredAt, c.Status, c.NoteFa, c.DecidedBy).Scan(&version)
	if err != nil {
		return domain.EmploymentCertificate{}, err
	}
	c.Version = version
	return c, nil
}

// DeleteCertificate حذف گواهی در وضعیت درخواست.
func (s *PostgresStore) DeleteCertificate(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `delete from hr_salary_admin.certificate where id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCertificateNotFound
	}
	return nil
}

// NextCertificateID شناسه و سریال گواهی تازه.
func (s *PostgresStore) NextCertificateID() (string, string, error) {
	var seq int
	if err := s.db.QueryRowContext(context.Background(),
		`select nextval('hr_salary_admin.certificate_seq')`).Scan(&seq); err != nil {
		return "", "", err
	}
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "", "", err
	}
	suffix := fmt.Sprintf("%04d", seq)
	return "CRT-" + suffix, fmt.Sprintf("CRT-%d-%s", d.Year, suffix), nil
}

/* -------------------------------- اسکنرها -------------------------------- */

type rowScanner interface{ Scan(dest ...any) error }

func scanDecree(row rowScanner) (domain.SalaryDecree, error) {
	var d domain.SalaryDecree
	err := row.Scan(&d.ID, &d.EmployeeID, &d.EmployeeName, &d.DepartmentName, &d.Position, &d.JobGroup,
		&d.BaseSalary, &d.EffectiveDate, &d.ReasonFa, &d.Status, &d.SupersedesID, &d.NoteFa,
		&d.CreatedBy, &d.DecidedBy, &d.EmploymentType, &d.Version,
		&d.ClimateRegion, &d.RankWage, &d.SeniorityWage, &d.PostAllowance, &d.SacrificeAllowance,
		&d.ClimateAllowance, &d.BadClimatePercent, &d.DeprivationPercent, &d.RegionHardshipPercent,
		&d.ExtraTimePercent, &d.SalaryRepair, &d.PayGap, &d.PayGapSince, &d.PayGroup, &d.PayModel)
	return d, err
}

func scanCertificate(row rowScanner) (domain.EmploymentCertificate, error) {
	var c domain.EmploymentCertificate
	err := row.Scan(&c.ID, &c.Serial, &c.EmployeeID, &c.EmployeeName, &c.DepartmentName, &c.Position,
		&c.Kind, &c.PurposeFa, &c.RequestedAt, &c.IssuedAt, &c.DeliveredAt, &c.Status, &c.NoteFa,
		&c.DecidedBy, &c.Version)
	return c, err
}
