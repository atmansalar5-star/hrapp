// لایهٔ پایدارسازی PostgreSQL ماژول «staff_services» — کوئری‌های پارامتری‌شده.

package infra

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/staff_services/domain"
	"hrms/internal/platform/calendar"
)

// PostgresStore انبار ماژول روی PostgreSQL — همان قرارداد MemoryStore.
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

const dependentColumns = `id, employee_id, employee_name, department_name, full_name, relation,
	national_id, birth_date, coverage_start, insurance, services, documents_fa, status, note_fa, decided_by, version`

const flightColumns = `id, employee_id, employee_name, department_name, origin_fa, destination_fa,
	depart_date, return_date, purpose, passengers, companions_fa, estimated_cost, status, note_fa,
	decided_by, ticket_no, version`

// GetDependent خواندن یک تکفل.
func (s *PostgresStore) GetDependent(ctx context.Context, id string) (domain.Dependent, bool, error) {
	row := s.db.QueryRowContext(ctx, `select `+dependentColumns+` from hr_personnel.dependent where id = $1`, id)
	d, err := scanDependent(row)
	if err == sql.ErrNoRows {
		return domain.Dependent{}, false, nil
	}
	return d, err == nil, err
}

// ListDependents فهرست تکفل‌شدگان با فیلتر و محدودهٔ دسترسی نقش.
func (s *PostgresStore) ListDependents(ctx context.Context, f contracts.DependentFilter) (contracts.DependentPage, error) {
	today, err := calendar.Parse(f.Today)
	if err != nil {
		today, _ = calendar.FromGregorian(s.clock())
	}

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
	if f.Relation != "" {
		conds = append(conds, "relation = "+ph(f.Relation))
	}
	if f.Status != "" {
		conds = append(conds, "status = "+ph(f.Status))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + q + "%"
		conds = append(conds, "(full_name ilike "+ph(like)+" or employee_name ilike "+ph(like)+
			" or national_id ilike "+ph(like)+")")
	}
	cond := "true"
	if len(conds) > 0 {
		cond = strings.Join(conds, " and ")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `select count(*) from hr_personnel.dependent where `+cond, args...).Scan(&total); err != nil {
		return contracts.DependentPage{}, err
	}
	page, size := normalizePaging(f.Page, f.Size)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`select %s from hr_personnel.dependent where %s order by id limit %d offset %d`,
		dependentColumns, cond, size, (page-1)*size), args...)
	if err != nil {
		return contracts.DependentPage{}, err
	}
	defer rows.Close()

	out := []contracts.Dependent{}
	for rows.Next() {
		d, err := scanDependent(rows)
		if err != nil {
			return contracts.DependentPage{}, err
		}
		out = append(out, d.ToContract(today))
	}
	return contracts.DependentPage{Items: out, Total: total, Page: page, Size: size}, rows.Err()
}

// SaveDependent درج/به‌روزرسانی تکفل با نسخه‌بندی خوش‌بینانه.
func (s *PostgresStore) SaveDependent(ctx context.Context, d domain.Dependent) (domain.Dependent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Dependent{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var current int
	err = tx.QueryRowContext(ctx, `select version from hr_personnel.dependent where id = $1 for update`, d.ID).Scan(&current)
	switch {
	case err == sql.ErrNoRows:
		d.Version = 1
	case err != nil:
		return domain.Dependent{}, err
	default:
		d.Version = current + 1
	}

	if _, err := tx.ExecContext(ctx, `
		insert into hr_personnel.dependent
			(id, employee_id, employee_name, department_name, full_name, relation, national_id,
			 birth_date, coverage_start, insurance, services, documents_fa, status, note_fa, decided_by, version, updated_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16, now())
		on conflict (id) do update set
			employee_id = excluded.employee_id, employee_name = excluded.employee_name,
			department_name = excluded.department_name, full_name = excluded.full_name,
			relation = excluded.relation, national_id = excluded.national_id,
			birth_date = excluded.birth_date, coverage_start = excluded.coverage_start,
			insurance = excluded.insurance, services = excluded.services, documents_fa = excluded.documents_fa,
			status = excluded.status, note_fa = excluded.note_fa,
			decided_by = excluded.decided_by, version = excluded.version, updated_at = now()`,
		d.ID, d.EmployeeID, d.EmployeeName, d.DepartmentName, d.FullName, d.Relation, d.NationalID,
		d.BirthDate, d.CoverageStart, d.Insurance, joinServices(d.Services), d.DocumentsFa, d.Status, d.NoteFa, d.DecidedBy, d.Version,
	); err != nil {
		return domain.Dependent{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Dependent{}, err
	}
	return d, nil
}

// DeleteDependent حذف تکفل.
func (s *PostgresStore) DeleteDependent(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `delete from hr_personnel.dependent where id = $1`, id)
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

// NextDependentID شناسهٔ تازه از sequence.
func (s *PostgresStore) NextDependentID() (string, error) {
	var seq int64
	if err := s.db.QueryRow(`select nextval('hr_personnel.dependent_seq')`).Scan(&seq); err != nil {
		return "", err
	}
	return fmt.Sprintf("DEP-%04d", seq), nil
}

// GetFlight خواندن یک درخواست پرواز.
func (s *PostgresStore) GetFlight(ctx context.Context, id string) (domain.FlightRequest, bool, error) {
	row := s.db.QueryRowContext(ctx, `select `+flightColumns+` from hr_personnel.flight_request where id = $1`, id)
	f, err := scanFlight(row)
	if err == sql.ErrNoRows {
		return domain.FlightRequest{}, false, nil
	}
	return f, err == nil, err
}

// ListFlights فهرست درخواست‌های پرواز.
func (s *PostgresStore) ListFlights(ctx context.Context, f contracts.FlightRequestFilter) (contracts.FlightRequestPage, error) {
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
	if f.Purpose != "" {
		conds = append(conds, "purpose = "+ph(f.Purpose))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		like := "%" + q + "%"
		conds = append(conds, "(employee_name ilike "+ph(like)+" or origin_fa ilike "+ph(like)+
			" or destination_fa ilike "+ph(like)+" or id ilike "+ph(like)+")")
	}
	cond := "true"
	if len(conds) > 0 {
		cond = strings.Join(conds, " and ")
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `select count(*) from hr_personnel.flight_request where `+cond, args...).Scan(&total); err != nil {
		return contracts.FlightRequestPage{}, err
	}
	page, size := normalizePaging(f.Page, f.Size)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`select %s from hr_personnel.flight_request where %s order by depart_date desc, id desc limit %d offset %d`,
		flightColumns, cond, size, (page-1)*size), args...)
	if err != nil {
		return contracts.FlightRequestPage{}, err
	}
	defer rows.Close()

	out := []contracts.FlightRequest{}
	for rows.Next() {
		f, err := scanFlight(rows)
		if err != nil {
			return contracts.FlightRequestPage{}, err
		}
		out = append(out, f.ToContract())
	}
	return contracts.FlightRequestPage{Items: out, Total: total, Page: page, Size: size}, rows.Err()
}

// SaveFlight درج/به‌روزرسانی درخواست پرواز.
func (s *PostgresStore) SaveFlight(ctx context.Context, f domain.FlightRequest) (domain.FlightRequest, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.FlightRequest{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var current int
	err = tx.QueryRowContext(ctx, `select version from hr_personnel.flight_request where id = $1 for update`, f.ID).Scan(&current)
	switch {
	case err == sql.ErrNoRows:
		f.Version = 1
	case err != nil:
		return domain.FlightRequest{}, err
	default:
		f.Version = current + 1
	}

	if _, err := tx.ExecContext(ctx, `
		insert into hr_personnel.flight_request
			(id, employee_id, employee_name, department_name, origin_fa, destination_fa, depart_date,
			 return_date, purpose, passengers, companions_fa, estimated_cost, status, note_fa,
			 decided_by, ticket_no, version, updated_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17, now())
		on conflict (id) do update set
			employee_id = excluded.employee_id, employee_name = excluded.employee_name,
			department_name = excluded.department_name, origin_fa = excluded.origin_fa,
			destination_fa = excluded.destination_fa, depart_date = excluded.depart_date,
			return_date = excluded.return_date, purpose = excluded.purpose,
			passengers = excluded.passengers, companions_fa = excluded.companions_fa,
			estimated_cost = excluded.estimated_cost, status = excluded.status,
			note_fa = excluded.note_fa, decided_by = excluded.decided_by,
			ticket_no = excluded.ticket_no, version = excluded.version, updated_at = now()`,
		f.ID, f.EmployeeID, f.EmployeeName, f.DepartmentName, f.OriginFa, f.DestinationFa, f.DepartDate,
		f.ReturnDate, f.Purpose, f.Passengers, f.CompanionsFa, f.EstimatedCost, f.Status, f.NoteFa,
		f.DecidedBy, f.TicketNo, f.Version,
	); err != nil {
		return domain.FlightRequest{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.FlightRequest{}, err
	}
	return f, nil
}

// NextFlightID شناسهٔ درخواست پرواز تازه.
func (s *PostgresStore) NextFlightID() (string, error) {
	var seq int64
	if err := s.db.QueryRow(`select nextval('hr_personnel.flight_seq')`).Scan(&seq); err != nil {
		return "", err
	}
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("FL-%d-%04d", d.Year, seq), nil
}

func scanDependent(row scanner) (domain.Dependent, error) {
	var servicesCSV string
	var d domain.Dependent
	err := row.Scan(&d.ID, &d.EmployeeID, &d.EmployeeName, &d.DepartmentName, &d.FullName, &d.Relation,
		&d.NationalID, &d.BirthDate, &d.CoverageStart, &d.Insurance, &servicesCSV, &d.DocumentsFa, &d.Status,
		&d.NoteFa, &d.DecidedBy, &d.Version)
	if err != nil {
		return d, err
	}
	d.Services = splitServices(servicesCSV)
	return d, nil
}

func scanFlight(row scanner) (domain.FlightRequest, error) {
	var f domain.FlightRequest
	err := row.Scan(&f.ID, &f.EmployeeID, &f.EmployeeName, &f.DepartmentName, &f.OriginFa, &f.DestinationFa,
		&f.DepartDate, &f.ReturnDate, &f.Purpose, &f.Passengers, &f.CompanionsFa, &f.EstimatedCost,
		&f.Status, &f.NoteFa, &f.DecidedBy, &f.TicketNo, &f.Version)
	return f, err
}

type scanner interface{ Scan(dest ...any) error }

// joinServices فهرست کلیدهای خدمت را به متن جدا‌شده با کاما تبدیل می‌کند (شکل ذخیره).
func joinServices(services []string) string {
	return strings.Join(contracts.NormalizeDependentServices(services), ",")
}

// splitServices متن ذخیره‌شده را به کلیدهای معتبر برمی‌گرداند.
func splitServices(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	return contracts.NormalizeDependentServices(strings.Split(value, ","))
}
