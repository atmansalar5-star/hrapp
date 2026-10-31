// لایهٔ پایدارسازی PostgreSQL ماژول «contracts» — کوئری‌های پارامتری‌شده.

package infra

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/contracts/domain"
	"hrms/internal/platform/calendar"
)

// PostgresStore انبار قراردادها روی PostgreSQL.
//
// همان قرارداد Repository را برمی‌آورد که MemoryStore برمی‌آورد، پس لایه‌های بالاتر
// (app و api) از تفاوت انبار بی‌خبرند. جدول‌ها در migrations همین ماژول ساخته می‌شوند.
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

// contractColumns ستون‌های قرارداد. ستون‌های wage_* / climate_* اجزای ثابت مزد و ضرایب
// اقلیمی همان قراردادند (مهاجرت ۰۰۰۲) و employment_type نوع استخدام است؛ هر سه در
// مهاجرت نخست جا افتاده بودند و قرارداد روی Postgres آن‌ها را از دست می‌داد.
const contractColumns = `id, employee_id, employee_name, department_name, position_title, kind,
	start_date, end_date, monthly_salary, status, supersedes_id, renew_count, note_fa,
	created_by, decided_by, version, employment_type,
	climate_region, rank_wage, seniority_wage, post_allowance, sacrifice_allowance, climate_allowance,
	bad_climate_percent, deprivation_percent, region_hardship_percent, extra_time_percent, salary_repair`

// Get خواندن قرارداد همراه با سابقهٔ رخدادها.
func (s *PostgresStore) Get(ctx context.Context, id string) (domain.Contract, bool, error) {
	row := s.db.QueryRowContext(ctx, `select `+contractColumns+` from hr_contracts.contract where id = $1`, id)
	c, err := scanContract(row)
	if err == sql.ErrNoRows {
		return domain.Contract{}, false, nil
	}
	if err != nil {
		return domain.Contract{}, false, err
	}
	events, err := s.events(ctx, id)
	if err != nil {
		return domain.Contract{}, false, err
	}
	c.Events = events
	return c, true, nil
}

// ActiveForEmployee قراردادهای غیرخاتمه‌یافتهٔ کارمند.
func (s *PostgresStore) ActiveForEmployee(ctx context.Context, employeeID string) ([]domain.Contract, error) {
	rows, err := s.db.QueryContext(ctx,
		`select `+contractColumns+` from hr_contracts.contract
		 where employee_id = $1 and status not in ($2, $3)
		 order by start_date`, employeeID, domain.StatusTerminated, domain.StatusRejected)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectContracts(rows)
}

// List فهرست قراردادها با فیلتر، جستجو، سرآمد و صفحه‌بندی.
//
// فیلتر محدودهٔ نقش (own/own_unit) در SQL اعمال می‌شود، نه با پنهان‌کردن سطر در UI:
// یک محدودیت امنیتی نباید به رفتار کلاینت وابسته باشد.
func (s *PostgresStore) List(ctx context.Context, f contracts.ContractFilter) (contracts.ContractPage, error) {
	today, err := calendar.Parse(f.Today)
	if err != nil {
		today, _ = calendar.FromGregorian(s.clock())
	}
	todayStr := today.String()

	conds := []string{}
	args := []any{}
	// ph یک مقدار را به فهرست پارامترها می‌افزاید و جایگاهش ($n) را برمی‌گرداند.
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
	if f.Kind != "" {
		conds = append(conds, "kind = "+ph(f.Kind))
	}
	if f.Status != "" {
		conds = append(conds, "status = "+ph(f.Status))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + q + "%"
		conds = append(conds, "(employee_name ilike "+ph(like)+" or id ilike "+ph(like)+
			" or department_name ilike "+ph(like)+" or position_title ilike "+ph(like)+")")
	}
	if f.ExpiringDays > 0 {
		upper, err := today.AddDays(f.ExpiringDays)
		if err != nil {
			return contracts.ContractPage{}, err
		}
		conds = append(conds,
			"kind = "+ph(domain.KindFixedTerm),
			"status = "+ph(domain.StatusActive),
			"end_date >= "+ph(todayStr),
			"end_date <= "+ph(upper.String()),
		)
	}

	cond := "true"
	if len(conds) > 0 {
		cond = strings.Join(conds, " and ")
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `select count(*) from hr_contracts.contract where `+cond, args...).Scan(&total); err != nil {
		return contracts.ContractPage{}, err
	}

	page, size := normalizePaging(f.Page, f.Size)
	order := "start_date desc, id desc"
	switch strings.SplitN(f.Sort, ":", 2)[0] {
	case "employeeName":
		order = "employee_name asc, start_date desc"
	case "endDate":
		order = "end_date desc, start_date desc"
	case "monthlySalary":
		order = "monthly_salary desc"
	case "status":
		order = "status asc, start_date desc"
	}

	query := fmt.Sprintf(`select %s from hr_contracts.contract where %s order by %s limit %d offset %d`,
		contractColumns, cond, order, size, (page-1)*size)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return contracts.ContractPage{}, err
	}
	defer rows.Close()
	items, err := collectContracts(rows)
	if err != nil {
		return contracts.ContractPage{}, err
	}

	out := make([]contracts.Contract, 0, len(items))
	for _, c := range items {
		out = append(out, c.ToContract(today))
	}
	return contracts.ContractPage{Items: out, Total: total, Page: page, Size: size}, nil
}

// Save درج/به‌روزرسانی قرارداد و بازنویسی سابقهٔ رخدادهایش در یک تراکنش.
func (s *PostgresStore) Save(ctx context.Context, c domain.Contract) (domain.Contract, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Contract{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var currentVersion int
	err = tx.QueryRowContext(ctx, `select version from hr_contracts.contract where id = $1 for update`, c.ID).Scan(&currentVersion)
	switch {
	case err == sql.ErrNoRows:
		c.Version = 1
	case err != nil:
		return domain.Contract{}, err
	default:
		c.Version = currentVersion + 1
	}

	if _, err := tx.ExecContext(ctx, `
		insert into hr_contracts.contract
			(id, employee_id, employee_name, department_name, position_title, kind, start_date, end_date,
			 monthly_salary, status, supersedes_id, renew_count, note_fa, created_by, decided_by, version,
			 employment_type, climate_region, rank_wage, seniority_wage, post_allowance, sacrifice_allowance,
			 climate_allowance, bad_climate_percent, deprivation_percent, region_hardship_percent,
			 extra_time_percent, salary_repair, updated_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28, now())
		on conflict (id) do update set
			employee_id = excluded.employee_id, employee_name = excluded.employee_name,
			department_name = excluded.department_name, position_title = excluded.position_title,
			kind = excluded.kind, start_date = excluded.start_date, end_date = excluded.end_date,
			monthly_salary = excluded.monthly_salary, status = excluded.status,
			supersedes_id = excluded.supersedes_id, renew_count = excluded.renew_count,
			note_fa = excluded.note_fa, created_by = excluded.created_by,
			decided_by = excluded.decided_by, version = excluded.version,
			employment_type = excluded.employment_type, climate_region = excluded.climate_region,
			rank_wage = excluded.rank_wage, seniority_wage = excluded.seniority_wage,
			post_allowance = excluded.post_allowance, sacrifice_allowance = excluded.sacrifice_allowance,
			climate_allowance = excluded.climate_allowance, bad_climate_percent = excluded.bad_climate_percent,
			deprivation_percent = excluded.deprivation_percent,
			region_hardship_percent = excluded.region_hardship_percent,
			extra_time_percent = excluded.extra_time_percent, salary_repair = excluded.salary_repair,
			updated_at = now()`,
		c.ID, c.EmployeeID, c.EmployeeName, c.DepartmentName, c.Position, c.Kind, c.StartDate, c.EndDate,
		c.MonthlySalary, c.Status, c.SupersedesID, c.RenewCount, c.NoteFa, c.CreatedBy, c.DecidedBy, c.Version,
		c.EmploymentType, c.ClimateRegion, c.RankWage, c.SeniorityWage, c.PostAllowance, c.SacrificeAllowance,
		c.ClimateAllowance, c.BadClimatePercent, c.DeprivationPercent, c.RegionHardshipPercent,
		c.ExtraTimePercent, c.SalaryRepair,
	); err != nil {
		return domain.Contract{}, err
	}

	if _, err := tx.ExecContext(ctx, `delete from hr_contracts.contract_event where contract_id = $1`, c.ID); err != nil {
		return domain.Contract{}, err
	}
	for _, e := range c.Events {
		at := e.At
		if at.IsZero() {
			at = s.clock()
		}
		if _, err := tx.ExecContext(ctx,
			`insert into hr_contracts.contract_event (contract_id, at, actor, action, note_fa) values ($1,$2,$3,$4,$5)`,
			c.ID, at, e.Actor, e.Action, e.NoteFa); err != nil {
			return domain.Contract{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Contract{}, err
	}
	return c, nil
}

// Delete حذف قرارداد پیش‌نویس.
func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `delete from hr_contracts.contract where id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// NextID شناسهٔ تازه از sequence پایگاه‌داده؛ در استقرار چندنمونه‌ای هم یکتا می‌ماند.
func (s *PostgresStore) NextID() (string, error) {
	var seq int64
	if err := s.db.QueryRow(`select nextval('hr_contracts.contract_seq')`).Scan(&seq); err != nil {
		return "", err
	}
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("CT-%d-%04d", d.Year, seq), nil
}

func (s *PostgresStore) events(ctx context.Context, contractID string) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`select at, actor, action, note_fa from hr_contracts.contract_event where contract_id = $1 order by at, id`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Event{}
	for rows.Next() {
		var e domain.Event
		if err := rows.Scan(&e.At, &e.Actor, &e.Action, &e.NoteFa); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanContract(row scanner) (domain.Contract, error) {
	var c domain.Contract
	err := row.Scan(&c.ID, &c.EmployeeID, &c.EmployeeName, &c.DepartmentName, &c.Position, &c.Kind,
		&c.StartDate, &c.EndDate, &c.MonthlySalary, &c.Status, &c.SupersedesID, &c.RenewCount,
		&c.NoteFa, &c.CreatedBy, &c.DecidedBy, &c.Version, &c.EmploymentType,
		&c.ClimateRegion, &c.RankWage, &c.SeniorityWage, &c.PostAllowance, &c.SacrificeAllowance,
		&c.ClimateAllowance, &c.BadClimatePercent, &c.DeprivationPercent, &c.RegionHardshipPercent,
		&c.ExtraTimePercent, &c.SalaryRepair)
	return c, err
}

func collectContracts(rows *sql.Rows) ([]domain.Contract, error) {
	out := []domain.Contract{}
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
