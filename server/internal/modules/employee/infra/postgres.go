// انبار PostgreSQL ماژول کارکنان.
//
// چرا این فایل اضافه شد: تا پیش از این، کارکنان فقط در انبار درون‌حافظه‌ای نگه داشته
// می‌شدند؛ یعنی ویرایش اطلاعات پایه با راه‌اندازی دوبارهٔ سرور از دست می‌رفت و در
// بک‌آپ پایگاه‌داده (`hrdump`) هم نمی‌آمد. این انبار همان قرارداد `app.Repository` را
// روی جدول `hr_employee.employee` پیاده می‌کند، پس هیچ لایهٔ بالاتر تغییر نمی‌کند:
// همان فیلترها، همان محدودهٔ دسترسی نقش، همان صفحه‌بندی و مرتب‌سازی، و همان شناسه‌های
// `EMP-###`.
//
// قاعده‌های نگه‌داری داده:
//   - حذف *نرم* است (`deleted = true`) تا tombstone برای سینک کلاینت‌های بی‌شبکه بماند؛
//     فهرست‌ها همیشه `deleted = false` را می‌خوانند.
//   - `version` در هر نوشتن یک پله بالا می‌رود و `server_seq` نشانگر ترتیب سرور است
//     (مبنای دلتای سینک؛ جدول `hr_employee.change_log` برای همین در مهاجرت ۰۰۰۱ هست و
//     نوشتنش کار لایهٔ سینک است تا نام کنشگر واقعی در آن ثبت شود).
//   - نام واحد سازمانی از جدول `hr_employee.department` خوانده می‌شود (نه کپی‌شده در
//     ردیف کارمند) تا تغییر نام واحد در همهٔ پرونده‌ها یک‌بار اثر کند.
package infra

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/employee/domain"
	"hrms/internal/platform/calendar"
)

// PgStore انبار کارکنان روی PostgreSQL.
type PgStore struct {
	db    *sql.DB
	clock func() time.Time

	// mu و seq شمارندهٔ شناسهٔ کارمند (`EMP-###`) را نگه می‌دارند. مقدار آغازین از
	// بیشترین شناسهٔ موجود در جدول خوانده می‌شود تا با دادهٔ بازگردانده‌شده تصادم نکند.
	mu     sync.Mutex
	seq    int
	seqSet bool
}

// NewPgStore انبار PostgreSQL می‌سازد. اگر clock nil باشد، زمان سیستم به کار می‌رود.
func NewPgStore(db *sql.DB, clock func() time.Time) *PgStore {
	if clock == nil {
		clock = time.Now
	}
	return &PgStore{db: db, clock: clock}
}

// employeeColumns ستون‌های انتخابی هر کوئری؛ نام واحد با join می‌آید.
const employeeColumns = `
	e.id, e.full_name, e.national_id, e.department_id, coalesce(d.name, '') as department_name,
	e.position, e.contract_type, e.employment_type, e.status, e.hire_date,
	e.base_salary, e.leave_balance, coalesce(e.phone, ''), coalesce(e.email, ''),
	coalesce(e.manager_name, ''), coalesce(e.mobile, ''), coalesce(e.home_phone, ''),
	coalesce(e.postal_code, ''), coalesce(e.address, ''), coalesce(e.gender, ''),
	coalesce(e.residence_site, ''), coalesce(e.job_grade, ''), e.birth_date, e.version`

const employeeFrom = `from hr_employee.employee e
	left join hr_employee.department d on d.id = e.department_id`

// scanner هر چیزی که Scan دارد (ردیف `*sql.Row` یا `*sql.Rows`).
type scanner interface{ Scan(dest ...any) error }

// scanEmployee یک ردیف کارمند را به شکل دامنه برمی‌گرداند.
func scanEmployee(row scanner) (domain.Employee, error) {
	var e domain.Employee
	var hireDate time.Time
	// تاریخ تولد اختیاری است؛ رکوردهای قدیمی مقدار NULL دارند.
	var birthDate sql.NullTime
	if err := row.Scan(&e.ID, &e.FullName, &e.NationalID, &e.DepartmentID, &e.DepartmentName,
		&e.Position, &e.ContractType, &e.EmploymentType, &e.Status, &hireDate,
		&e.BaseSalary, &e.LeaveBalance, &e.Phone, &e.Email, &e.ManagerName,
		&e.Mobile, &e.HomePhone, &e.PostalCode, &e.Address, &e.Gender,
		&e.ResidenceSite, &e.JobGrade, &birthDate, &e.Version); err != nil {
		return domain.Employee{}, err
	}
	// تاریخ استخدام در داده و API به قالب شمسی نگه داشته می‌شود و در جدول از نوع date
	// است؛ تبدیل در همین لبه انجام می‌شود (قاعدهٔ پروژه: تبدیل تقویمی در لایهٔ کد).
	e.HireDate = jalaliString(hireDate)
	if birthDate.Valid {
		e.BirthDate = jalaliString(birthDate.Time)
	}
	return e, nil
}

// jalaliString تاریخ میلادی جدول را به رشتهٔ شمسی `YYYY/MM/DD` تبدیل می‌کند.
func jalaliString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	date, err := calendar.FromGregorian(value)
	if err != nil {
		return value.Format("2006/01/02")
	}
	return fmt.Sprintf("%04d/%02d/%02d", date.Year, date.Month, date.Day)
}

// jalaliDate رشتهٔ شمسی را به تاریخ میلادی برای نوشتن در جدول تبدیل می‌کند.
func jalaliDate(value string) (time.Time, error) {
	date, err := calendar.Parse(strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return calendar.ToGregorian(date.Year, date.Month, date.Day)
}

// birthDateValue تاریخ تولد شمسی را برای نوشتن در ستون date آماده می‌کند؛
// رشتهٔ خالی یعنی «ثبت نشده» و NULL در جدول می‌نشیند.
func birthDateValue(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	date, err := jalaliDate(value)
	if err != nil {
		return nil
	}
	return date
}

// Get خواندن یک کارمند حذف‌نشده با شناسه.
func (s *PgStore) Get(ctx context.Context, id string) (domain.Employee, bool, error) {
	row := s.db.QueryRowContext(ctx, `select `+employeeColumns+` `+employeeFrom+
		` where e.id = $1 and not e.deleted`, strings.TrimSpace(id))
	e, err := scanEmployee(row)
	if err == sql.ErrNoRows {
		return domain.Employee{}, false, nil
	}
	if err != nil {
		return domain.Employee{}, false, err
	}
	return e, true, nil
}

// whereClause شرط‌های فهرست را از فیلتر می‌سازد و آرگومان‌هایشان را برمی‌گرداند.
//
// هیچ مقدار کاربری داخل متن SQL نوشته نمی‌شود؛ همه پارامتر عددی (`$n`) می‌شوند.
func (s *PgStore) whereClause(f contracts.EmployeeFilter) (string, []any) {
	where := []string{"not e.deleted"}
	args := []any{}
	addArg := func(value any) string {
		args = append(args, value)
		return "$" + strconv.Itoa(len(args))
	}
	add := func(column string, value string) {
		where = append(where, column+" = "+addArg(value))
	}
	addList := func(single string, list []string, column string) {
		values := trimmedValues(list)
		if len(values) == 0 {
			if value := strings.TrimSpace(single); value != "" {
				add(column, value)
			}
			return
		}
		placeholders := make([]string, 0, len(values))
		for _, v := range values {
			placeholders = append(placeholders, addArg(v))
		}
		where = append(where, column+" in ("+strings.Join(placeholders, ", ")+")")
	}

	// محدودهٔ دسترسی نقش: کارمند فقط خودش، مدیر واحد فقط واحد خودش.
	switch f.Scope {
	case "own":
		if f.ActorID != "" {
			add("e.id", f.ActorID)
		}
	case "own_unit":
		if f.ActorDepartment != "" {
			add("e.department_id", f.ActorDepartment)
		}
	}
	if f.DepartmentID != "" {
		add("e.department_id", f.DepartmentID)
	}
	if f.ContractType != "" {
		add("e.contract_type", f.ContractType)
	}
	// وضعیت و نوع استخدام: فهرست چندانتخابی بر مقدار تک‌مقداری مقدم است (همان قاعدهٔ
	// contracts.MatchValue در انبار حافظه).
	addList(f.Status, f.Statuses, "e.status")
	addList(f.EmploymentType, f.EmploymentTypes, "e.employment_type")

	if query := strings.TrimSpace(f.Query); query != "" {
		placeholder := addArg("%" + strings.ToLower(query) + "%")
		where = append(where, `(lower(e.full_name) like `+placeholder+
			` or lower(e.id) like `+placeholder+
			` or e.national_id like `+placeholder+
			` or lower(e.position) like `+placeholder+
			` or lower(coalesce(d.name, '')) like `+placeholder+`)`)
	}
	return " where " + strings.Join(where, " and "), args
}

// List فهرست کارکنان با همان فیلترها، محدودهٔ دسترسی، مرتب‌سازی و صفحه‌بندی انبار حافظه.
func (s *PgStore) List(ctx context.Context, f contracts.EmployeeFilter) (contracts.EmployeePage, error) {
	cond, args := s.whereClause(f)

	var total int
	if err := s.db.QueryRowContext(ctx, `select count(*) `+employeeFrom+cond, args...).Scan(&total); err != nil {
		return contracts.EmployeePage{}, err
	}

	page, size := normalizePaging(f.Page, f.Size)
	offset := (page - 1) * size
	args = append(args, size, offset)
	query := `select ` + employeeColumns + ` ` + employeeFrom + cond +
		` order by ` + orderBy(f.Sort) +
		` limit $` + strconv.Itoa(len(args)-1) + ` offset $` + strconv.Itoa(len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return contracts.EmployeePage{}, err
	}
	defer rows.Close()

	items := make([]contracts.Employee, 0, size)
	for rows.Next() {
		e, err := scanEmployee(rows)
		if err != nil {
			return contracts.EmployeePage{}, err
		}
		items = append(items, e.ToContract())
	}
	if err := rows.Err(); err != nil {
		return contracts.EmployeePage{}, err
	}
	return contracts.EmployeePage{Items: items, Total: total, Page: page, Size: size}, nil
}

// CountByStatus شمارش کارکنان به تفکیک وضعیت (کارت‌های آماری داشبورد).
func (s *PgStore) CountByStatus(ctx context.Context, f contracts.EmployeeFilter) (map[string]int, error) {
	// محدودهٔ دسترسی نقش عیناً اعمال می‌شود تا شمارش با فهرستی که کاربر می‌بیند یکی بماند.
	cond, args := s.whereClause(f)
	rows, err := s.db.QueryContext(ctx,
		`select e.status, count(*) `+employeeFrom+cond+` group by e.status`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, rows.Err()
}

// Save درج یا به‌روزرسانی کارمند؛ نسخه یک پله بالا می‌رود و نشانگر سرور تازه می‌شود.
//
// تراز مرخصی مثل انبار حافظه تنها در *درج* نوشته می‌شود و در به‌روزرسانی دست‌نخورده
// می‌ماند تا ویرایش مشخصات، ماندهٔ مرخصی محاسبه‌شده را پاک نکند.
func (s *PgStore) Save(ctx context.Context, e domain.Employee) (domain.Employee, error) {
	hireDate, err := jalaliDate(e.HireDate)
	if err != nil {
		return domain.Employee{}, fmt.Errorf("تاریخ استخدام نامعتبر است: %s", e.HireDate)
	}
	id := strings.TrimSpace(e.ID)
	if _, err := strconv.Atoi(strings.TrimPrefix(id, "EMP-")); err != nil {
		return domain.Employee{}, fmt.Errorf("شناسهٔ کارمند نامعتبر است: %s", id)
	}
	if err := s.ensureDepartment(ctx, e.DepartmentID); err != nil {
		return domain.Employee{}, err
	}

	_, err = s.db.ExecContext(ctx, `
		insert into hr_employee.employee (
			id, full_name, national_id, department_id, position, contract_type, employment_type,
			status, hire_date, base_salary, leave_balance, phone, email, manager_name,
			mobile, home_phone, postal_code, address, gender, birth_date,
			residence_site, job_grade, version, server_seq, deleted, created_at, updated_at
		) values (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,
			1, coalesce((select max(server_seq) from hr_employee.employee), 0) + 1, false, now(), now()
		)
		on conflict (id) do update set
			full_name = excluded.full_name,
			national_id = excluded.national_id,
			department_id = excluded.department_id,
			position = excluded.position,
			contract_type = excluded.contract_type,
			employment_type = excluded.employment_type,
			status = excluded.status,
			hire_date = excluded.hire_date,
			base_salary = excluded.base_salary,
			phone = excluded.phone,
			email = excluded.email,
			manager_name = excluded.manager_name,
			mobile = excluded.mobile,
			home_phone = excluded.home_phone,
			postal_code = excluded.postal_code,
			address = excluded.address,
			gender = excluded.gender,
			birth_date = excluded.birth_date,
			residence_site = excluded.residence_site,
			job_grade = excluded.job_grade,
			version = hr_employee.employee.version + 1,
			server_seq = coalesce((select max(server_seq) from hr_employee.employee), 0) + 1,
			deleted = false,
			updated_at = now()`,
		id, strings.TrimSpace(e.FullName), strings.TrimSpace(e.NationalID), e.DepartmentID,
		strings.TrimSpace(e.Position), e.ContractType, e.EmploymentType, e.Status,
		hireDate, e.BaseSalary, e.LeaveBalance, e.Phone, e.Email, e.ManagerName,
		e.Mobile, e.HomePhone, e.PostalCode, e.Address, e.Gender,
		birthDateValue(e.BirthDate), e.ResidenceSite, strings.TrimSpace(e.JobGrade))
	if err != nil {
		return domain.Employee{}, err
	}

	// پاسخ از خود جدول خوانده می‌شود تا نسخه، نام واحد و ماندهٔ مرخصی با دادهٔ
	// ذخیره‌شده یکی باشد (نه با ورودی سرویس).
	saved, ok, err := s.Get(ctx, id)
	if err != nil {
		return domain.Employee{}, err
	}
	if !ok {
		return domain.Employee{}, ErrNotFound
	}
	s.trackID(id)
	return saved, nil
}

// Delete حذف نرم کارمند (tombstone برای سینک)؛ اگر پیدا نشود خطای «یافت نشد».
func (s *PgStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `
		update hr_employee.employee
		set deleted = true,
		    version = version + 1,
		    server_seq = coalesce((select max(server_seq) from hr_employee.employee), 0) + 1,
		    updated_at = now()
		where id = $1 and not deleted`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// NextID شناسهٔ کارمند تازه (`EMP-###`).
//
// چرا شمارندهٔ درون‌حافظه‌ای: انبار حافظه هم همین کار را می‌کند و شناسه‌ها خوانا
// می‌مانند. مقدار آغازین یک‌بار از بیشترین شناسهٔ جدول خوانده می‌شود تا با دادهٔ
// بازگردانده‌شده یا واردشدهٔ دستی تصادم نکند.
func (s *PgStore) NextID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.seqSet {
		s.seq = s.maxSeqFromDB()
		s.seqSet = true
	}
	s.seq++
	return fmt.Sprintf("EMP-%03d", s.seq)
}

// ensureDepartment واحد سازمانی کارمند را در صورت نبود می‌سازد تا کلید خارجی نشکند.
//
// چرا: کاربر ممکن است واحد تازه‌ای وارد کند (مثلاً در داده‌ای که خودش اضافه می‌کند)؛
// پیام خطای «کلید خارجی» گنگ است، پس واحد ناشناخته از فهرست پایه ساخته می‌شود.
func (s *PgStore) ensureDepartment(ctx context.Context, departmentID string) error {
	id := strings.TrimSpace(departmentID)
	if id == "" {
		return fmt.Errorf("واحد سازمانی کارمند الزامی است")
	}
	_, err := s.db.ExecContext(ctx, `
		insert into hr_employee.department (id, name, head_name, parent_id, headcount)
		values ($1, $2, $3, null, 0)
		on conflict (id) do update set name = excluded.name`,
		id, DepartmentName(id), "")
	return err
}

// maxSeqFromDB بیشترین شمارهٔ شناسهٔ موجود را می‌خواند (صفر اگر جدول خالی باشد).
func (s *PgStore) maxSeqFromDB() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var maxID sql.NullString
	if err := s.db.QueryRowContext(ctx, `select max(id) from hr_employee.employee`).Scan(&maxID); err != nil {
		return 0
	}
	if !maxID.Valid {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(maxID.String), "EMP-"))
	if err != nil {
		return 0
	}
	return n
}

// trackID شمارنده را از شناسهٔ نوشته‌شده جلو می‌برد (درج با شناسهٔ صریح، مثل دادهٔ نمونه).
func (s *PgStore) trackID(id string) {
	n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(id), "EMP-"))
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if n > s.seq {
		s.seq = n
	}
	s.seqSet = true
}

// orderBy ستون مرتب‌سازی امن (فهرست سفید) برای پارامتر sort.
//
// چرا فهرست سفید: مقدار sort از کوئری رشته می‌آید؛ نگاشت مستقیم آن به SQL درِ تزریق
// می‌گشاید. هر مقدار ناشناخته به ترتیب پیش‌فرض (شناسه) برمی‌گردد.
func orderBy(spec string) string {
	field, desc := "id", false
	if spec != "" {
		parts := strings.SplitN(spec, ":", 2)
		field = parts[0]
		if len(parts) == 2 && strings.EqualFold(parts[1], "desc") {
			desc = true
		}
	}
	columns := map[string]string{
		"id":             "e.id",
		"fullName":       "e.full_name",
		"hireDate":       "e.hire_date",
		"baseSalary":     "e.base_salary",
		"departmentName": "coalesce(d.name, '')",
		"status":         "e.status",
	}
	column, ok := columns[field]
	if !ok {
		column = "e.id"
	}
	if desc {
		return column + " desc, e.id desc"
	}
	return column + ", e.id"
}

// trimmedValues فهرست را از مقدار خالی پاک می‌کند.
func trimmedValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if value := strings.TrimSpace(v); value != "" {
			out = append(out, value)
		}
	}
	return out
}
