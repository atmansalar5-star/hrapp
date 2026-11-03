// دادهٔ اولیهٔ (seed) ماژول کارکنان روی PostgreSQL.
//
// چرا جدا از انبار: روی استقرار واقعی، جدول کارکنان در اولین اجرا خالی است و بدون این
// بذرکاری، سامانه با فهرست خالی بالا می‌آید و آزمون‌های دودی وب به داده نیاز دارند.
// قاعده همان قاعدهٔ بقیهٔ ماژول‌هاست: فقط جدول *خالی* پر می‌شود؛ اگر کاربر دادهٔ خودش
// را وارد کرده باشد، هیچ‌چیز بازنویسی نمی‌شود.

package infra

import (
	"context"
)

// SeedIfEmpty واحدهای سازمانی و کارکنان نمونه را می‌ریزد و شمار ردیف‌های نوشته‌شده را
// برمی‌گرداند — ولی تنها وقتی جدول کارکنان خالی باشد.
func (s *PgStore) SeedIfEmpty(ctx context.Context) (departments, employees int, err error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `select count(*) from hr_employee.employee`).Scan(&count); err != nil {
		return 0, 0, err
	}
	if count > 0 {
		return 0, 0, nil
	}

	// نخست واحدها: کلید خارجی `employee.department_id` به وجود واحد نیاز دارد.
	for _, d := range Departments() {
		_, err := s.db.ExecContext(ctx, `
			insert into hr_employee.department (id, name, head_name, parent_id, headcount)
			values ($1, $2, $3, null, $4)
			on conflict (id) do update set
				name = excluded.name, head_name = excluded.head_name, headcount = excluded.headcount`,
			d.ID, d.Name, d.Head, d.Headcount)
		if err != nil {
			return departments, employees, err
		}
		departments++
	}

	for _, e := range SeedEmployees() {
		if _, err := s.Save(ctx, e); err != nil {
			return departments, employees, err
		}
		employees++
	}
	return departments, employees, nil
}
