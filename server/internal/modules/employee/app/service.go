// Package app لایهٔ کاربرد ماژول کارکنان: گردش کار، اعتبارسنجی، آدیت و سینک.
package app

import (
	"context"
	"fmt"

	"hrms/internal/contracts"
	"hrms/internal/modules/employee/domain"
	"hrms/internal/modules/employee/infra"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/synclog"
)

// Repository قرارداد ذخیره‌سازی که ماژول به آن وابسته است.
// پیاده‌سازی فعلی درون‌حافظه‌ای است و پیاده‌سازی Postgres بعداً همین قرارداد را برمی‌آورد.
type Repository interface {
	Get(ctx context.Context, id string) (domain.Employee, bool, error)
	List(ctx context.Context, f contracts.EmployeeFilter) (contracts.EmployeePage, error)
	// CountByStatus برای کارت‌های آماری داشبورد لازم است؛ همان چیزی که پورت
	// contracts.EmployeeReader وعده می‌دهد و ماژول داشبورد مصرف می‌کند.
	CountByStatus(ctx context.Context, f contracts.EmployeeFilter) (map[string]int, error)
	Save(ctx context.Context, e domain.Employee) (domain.Employee, error)
	Delete(ctx context.Context, id string) error
}

// ValidationError خطای اعتبارسنجی با جزئیات فیلد‌به‌فیلد.
type ValidationError struct {
	Details map[string]string
}

func (e *ValidationError) Error() string {
	return "اطلاعات ارسالی معتبر نیست."
}

// Service خدمات ماژول کارکنان.
type Service struct {
	repo  Repository
	audit *audit.Log
	sync  *synclog.Log
}

// New سرویس می‌سازد؛ آدیت و سینک اختیاری‌اند تا آزمون‌ها ساده بمانند.
func New(repo Repository, auditLog *audit.Log, syncLog *synclog.Log) *Service {
	return &Service{repo: repo, audit: auditLog, sync: syncLog}
}

// List فهرست کارکنان با اعمال محدودهٔ دسترسی.
func (s *Service) List(ctx context.Context, f contracts.EmployeeFilter) (contracts.EmployeePage, error) {
	return s.repo.List(ctx, f)
}

// Get پروندهٔ کارمند با اعمال محدودهٔ دسترسی؛ در صورت خارج بودن از محدوده، یافت‌نشده برمی‌گردد.
func (s *Service) Get(ctx context.Context, f contracts.EmployeeFilter, id string) (contracts.Employee, bool, error) {
	e, ok, err := s.repo.Get(ctx, id)
	if err != nil || !ok {
		return contracts.Employee{}, false, err
	}
	if !withinScope(f, e) {
		// عمداً «یافت نشد» برمی‌گردانیم تا وجود کارمند در واحدهای دیگر لو نرود.
		return contracts.Employee{}, false, nil
	}
	return e.ToContract(), true, nil
}

// CountByStatus شمارش بر پایهٔ وضعیت (مصرف‌کنندهٔ پورت: داشبورد).
func (s *Service) CountByStatus(ctx context.Context, f contracts.EmployeeFilter) (map[string]int, error) {
	return s.repo.CountByStatus(ctx, f)
}

// Create ایجاد کارمند تازه.
func (s *Service) Create(ctx context.Context, in contracts.EmployeeInput, actor string) (contracts.Employee, error) {
	if errs := domain.Validate(in); len(errs) > 0 {
		return contracts.Employee{}, &ValidationError{Details: errs}
	}
	repo, ok := s.repo.(interface{ NextID() string })
	if !ok {
		return contracts.Employee{}, fmt.Errorf("انبار از ساخت شناسهٔ تازه پشتیبانی نمی‌کند")
	}
	e := domain.FromInput(repo.NextID(), in, infra.DepartmentName(in.DepartmentID))

	saved, err := s.repo.Save(ctx, e)
	if err != nil {
		return contracts.Employee{}, err
	}
	s.record(actor, "create", saved)
	return saved.ToContract(), nil
}

// Update ویرایش کارمند.
func (s *Service) Update(ctx context.Context, id string, in contracts.EmployeeInput, actor string) (contracts.Employee, error) {
	if errs := domain.Validate(in); len(errs) > 0 {
		return contracts.Employee{}, &ValidationError{Details: errs}
	}
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Employee{}, err
	}
	if !ok {
		return contracts.Employee{}, infra.ErrNotFound
	}
	updated := current.Apply(in)
	if in.DepartmentID != "" {
		updated.DepartmentName = infra.DepartmentName(in.DepartmentID)
	}

	saved, err := s.repo.Save(ctx, updated)
	if err != nil {
		return contracts.Employee{}, err
	}
	s.record(actor, "update", saved)
	return saved.ToContract(), nil
}

// Delete حذف کارمند.
func (s *Service) Delete(ctx context.Context, id string, actor string) error {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return infra.ErrNotFound
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Append(actor, "employee", "delete", id, "حذف پروندهٔ "+current.FullName)
	}
	if s.sync != nil {
		s.sync.Record("Employee", synclog.OpDelete, id, current.Version+1, nil)
	}
	return nil
}

// Departments واحدهای سازمانی برای فیلتر و نمودار درختی.
func (s *Service) Departments() []contracts.Department { return infra.Departments() }

func (s *Service) record(actor, action string, e domain.Employee) {
	if s.audit != nil {
		s.audit.Append(actor, "employee", action, e.ID, "پروندهٔ "+e.FullName)
	}
	if s.sync != nil {
		payload := map[string]any{
			"id":             e.ID,
			"fullName":       e.FullName,
			"nationalId":     e.NationalID,
			"departmentId":   e.DepartmentID,
			"departmentName": e.DepartmentName,
			"position":       e.Position,
			"contractType":   e.ContractType,
			"status":         e.Status,
			"hireDate":       e.HireDate,
			"baseSalary":     e.BaseSalary,
			"version":        e.Version,
		}
		s.sync.Record("Employee", synclog.OpUpsert, e.ID, e.Version, payload)
	}
}

func withinScope(f contracts.EmployeeFilter, e domain.Employee) bool {
	switch f.Scope {
	case "own":
		return f.ActorID == "" || e.ID == f.ActorID
	case "own_unit":
		return f.ActorDepartment == "" || e.DepartmentID == f.ActorDepartment
	default:
		return true
	}
}
