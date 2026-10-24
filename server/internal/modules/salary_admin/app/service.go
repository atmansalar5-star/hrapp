// Package app لایهٔ کاربرد ماژول احکام حقوق و گواهی اشتغال.
//
// گردش‌کار اینجاست: حکم از پیش‌نویس → صدور → ابلاغ می‌رود و اصلاح آن یک حکم تازه
// می‌سازد که پیشین را منسوخ می‌کند؛ گواهی از درخواست → تأیید → صدور → تحویل می‌گذرد.
// قواعد خالص در domain است و این لایه فقط آن‌ها را به هم می‌بندد.
package app

import (
	"context"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/salary_admin/domain"
	"hrms/internal/modules/salary_admin/infra"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/synclog"
)

// Repository قرارداد ذخیره‌سازی؛ هر دو انبار (حافظه و Postgres) آن را برمی‌آورند.
type Repository interface {
	GetDecree(ctx context.Context, id string) (domain.SalaryDecree, bool, error)
	NotifiedForEmployee(ctx context.Context, employeeID string) ([]domain.SalaryDecree, error)
	ListDecrees(ctx context.Context, f contracts.SalaryDecreeFilter) (contracts.SalaryDecreePage, error)
	SaveDecree(ctx context.Context, d domain.SalaryDecree) (domain.SalaryDecree, error)
	DeleteDecree(ctx context.Context, id string) error
	NextDecreeID() (string, error)

	GetCertificate(ctx context.Context, id string) (domain.EmploymentCertificate, bool, error)
	ListCertificates(ctx context.Context, f contracts.CertificateFilter) (contracts.CertificatePage, error)
	SaveCertificate(ctx context.Context, c domain.EmploymentCertificate) (domain.EmploymentCertificate, error)
	DeleteCertificate(ctx context.Context, id string) error
	NextCertificateID() (string, string, error)
}

// EmployeeLookup پورت خواندن کارمند — از راه contracts.EmployeeReader می‌آید.
type EmployeeLookup interface {
	ByID(ctx context.Context, employeeID string) (contracts.Employee, bool, error)
	// ListByEmploymentType کارکنان فعال یک نوع استخدام (و اختیاراً یک اداره) را
	// برمی‌گرداند؛ پایهٔ صدور گروهی حکم.
	ListByEmploymentType(ctx context.Context, employmentType, departmentID string) ([]contracts.Employee, error)
}

// LookupFromReader پوشش روی پورت کارکنان.
func LookupFromReader(reader contracts.EmployeeReader) EmployeeLookup {
	return employeeReaderAdapter{reader: reader}
}

type employeeReaderAdapter struct{ reader contracts.EmployeeReader }

func (a employeeReaderAdapter) ByID(ctx context.Context, employeeID string) (contracts.Employee, bool, error) {
	if a.reader == nil {
		return contracts.Employee{}, false, nil
	}
	return a.reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
}

// ListByEmploymentType کارکنان یک نوع استخدام را از پورت کارکنان می‌خواند. کارکنان
// خاتمه‌یافته حذف می‌شوند تا برای نیروی خارج‌شده حکم صادر نشود.
func (a employeeReaderAdapter) ListByEmploymentType(ctx context.Context, employmentType, departmentID string) ([]contracts.Employee, error) {
	if a.reader == nil {
		return nil, nil
	}
	page, err := a.reader.List(ctx, contracts.EmployeeFilter{
		Scope:          "all",
		Size:           1000,
		EmploymentType: employmentType,
		DepartmentID:   departmentID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.Employee, 0, len(page.Items))
	for _, e := range page.Items {
		if e.Status == "terminated" {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// ValidationError خطای اعتبارسنجی فیلد‌به‌فیلد.
type ValidationError struct{ Details map[string]string }

func (e *ValidationError) Error() string { return "اطلاعات ارسالی معتبر نیست." }

// RuleError نقض یک قاعدهٔ کسب‌وکار.
type RuleError struct{ Message string }

func (e *RuleError) Error() string { return e.Message }

// Service خدمات احکام حقوق و گواهی اشتغال.
type Service struct {
	repo      Repository
	employees EmployeeLookup
	audit     *audit.Log
	sync      *synclog.Log
	clock     func() time.Time
}

// New سرویس می‌سازد.
func New(repo Repository, employees EmployeeLookup, auditLog *audit.Log, syncLog *synclog.Log, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{repo: repo, employees: employees, audit: auditLog, sync: syncLog, clock: clock}
}

func (s *Service) today() string {
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "1400/01/01"
	}
	return d.String()
}

/* ------------------------------- حکم حقوق ------------------------------- */

// ListDecrees فهرست احکام با محدودهٔ دسترسی نقش.
func (s *Service) ListDecrees(ctx context.Context, f contracts.SalaryDecreeFilter) (contracts.SalaryDecreePage, error) {
	return s.repo.ListDecrees(ctx, f)
}

// NotifiedDecrees حکم‌های ابلاغ‌شدهٔ یک کارمند را برمی‌گرداند.
//
// مصرف آن پورت بین‌ماژولی است (محاسبهٔ پاداش پایان خدمت): آن مصرف‌کننده به حکم کارمند
// در محدودهٔ نقش خودش نیاز دارد، نه به فهرست سازمانی؛ پس فیلتر محدوده اینجا اعمال
// نمی‌شود و بررسی نقش در همان ماژول مصرف‌کننده انجام می‌شود.
func (s *Service) NotifiedDecrees(ctx context.Context, employeeID string) ([]contracts.SalaryDecree, error) {
	items, err := s.repo.NotifiedForEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	out := make([]contracts.SalaryDecree, 0, len(items))
	for _, d := range items {
		out = append(out, d.ToSalaryDecree())
	}
	return out, nil
}

// GetDecree خواندن یک حکم با بررسی محدودهٔ دسترسی.
func (s *Service) GetDecree(ctx context.Context, f contracts.SalaryDecreeFilter, id string) (contracts.SalaryDecree, bool, error) {
	d, ok, err := s.repo.GetDecree(ctx, id)
	if err != nil || !ok {
		return contracts.SalaryDecree{}, false, err
	}
	if !withinScope(d.EmployeeID, d.DepartmentName, f.Scope, f.ActorEmployeeID, f.ActorDepartment) {
		return contracts.SalaryDecree{}, false, nil
	}
	return d.ToSalaryDecree(), true, nil
}

// CreateDecree صدور پیش‌نویس حکم حقوق.
func (s *Service) CreateDecree(ctx context.Context, in contracts.SalaryDecreeInput, actor string) (contracts.SalaryDecree, error) {
	if errs := domain.ValidateSalaryDecree(in); len(errs) > 0 {
		return contracts.SalaryDecree{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	if !ok {
		return contracts.SalaryDecree{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}
	id, err := s.repo.NextDecreeID()
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	draft := domain.SalaryDecree{
		ID:             id,
		EmployeeID:     employee.ID,
		EmployeeName:   employee.FullName,
		DepartmentName: employee.DepartmentName,
		Position:       employee.Position,
		EmploymentType: employmentTypeOf(in.EmploymentType, employee.EmploymentType),
		JobGroup:       in.JobGroup,
		BaseSalary:     in.BaseSalary,
		EffectiveDate:  in.EffectiveDate,
		ReasonFa:       in.ReasonFa,
		Status:         domain.DecreeDraft,
		NoteFa:         in.NoteFa,
		CreatedBy:      actor,
	}
	saved, err := s.repo.SaveDecree(ctx, draft)
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	s.record(actor, domain.ActionCreate, saved, "صدور پیش‌نویس حکم حقوق")
	return saved.ToSalaryDecree(), nil
}

// UpdateDecree ویرایش حکم پیش‌نویس.
func (s *Service) UpdateDecree(ctx context.Context, id string, in contracts.SalaryDecreeInput, actor string) (contracts.SalaryDecree, error) {
	current, ok, err := s.repo.GetDecree(ctx, id)
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	if !ok {
		return contracts.SalaryDecree{}, infra.ErrDecreeNotFound
	}
	if current.Status != domain.DecreeDraft {
		return contracts.SalaryDecree{}, &RuleError{Message: "تنها حکم پیش‌نویس قابل ویرایش است؛ برای حکم ابلاغ‌شده از اصلاح استفاده کنید."}
	}
	if errs := domain.ValidateSalaryDecree(in); len(errs) > 0 {
		return contracts.SalaryDecree{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	if !ok {
		return contracts.SalaryDecree{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}
	current.EmployeeID = employee.ID
	current.EmployeeName = employee.FullName
	current.DepartmentName = employee.DepartmentName
	current.Position = employee.Position
	current.JobGroup = in.JobGroup
	current.BaseSalary = in.BaseSalary
	current.EffectiveDate = in.EffectiveDate
	current.ReasonFa = in.ReasonFa
	current.NoteFa = in.NoteFa
	saved, err := s.repo.SaveDecree(ctx, current)
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	s.record(actor, domain.ActionUpdate, saved, "ویرایش پیش‌نویس حکم حقوق")
	return saved.ToSalaryDecree(), nil
}

// IssueDecree صدور حکم از پیش‌نویس.
func (s *Service) IssueDecree(ctx context.Context, id, actor string) (contracts.SalaryDecree, error) {
	d, err := s.decreeTransition(ctx, id, actor, domain.DecreeIssued, domain.ActionIssue, "صدور حکم حقوق")
	return d.ToSalaryDecree(), err
}

// NotifyDecree ابلاغ حکم صادرشده؛ در صورت جانشینی، حکم پیشین منسوخ می‌شود
// (منسوخ‌سازی داخل decreeTransition انجام می‌گیرد تا یک‌جا و اتمی بماند).
func (s *Service) NotifyDecree(ctx context.Context, id, actor string) (contracts.SalaryDecree, error) {
	d, err := s.decreeTransition(ctx, id, actor, domain.DecreeNotified, domain.ActionNotify, "ابلاغ حکم حقوق")
	return d.ToSalaryDecree(), err
}

// ReviseDecree اصلاح حکم ابلاغ‌شده: حکم تازهٔ پیش‌نویس که جانشین حکم پیشین می‌شود.
func (s *Service) ReviseDecree(ctx context.Context, id string, in contracts.SalaryDecreeDecision, actor string) (contracts.SalaryDecree, error) {
	current, ok, err := s.repo.GetDecree(ctx, id)
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	if !ok {
		return contracts.SalaryDecree{}, infra.ErrDecreeNotFound
	}
	if errs := domain.ValidateRevise(current, in); len(errs) > 0 {
		return contracts.SalaryDecree{}, &ValidationError{Details: errs}
	}
	salary := current.BaseSalary
	if in.BaseSalary > 0 {
		salary = in.BaseSalary
	}
	newID, err := s.repo.NextDecreeID()
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	revised := current
	revised.ID = newID
	revised.BaseSalary = salary
	revised.Status = domain.DecreeDraft
	revised.SupersedesID = current.ID
	revised.Version = 0
	revised.ReasonFa = in.ReasonFa
	revised.NoteFa = in.NoteFa
	revised.CreatedBy = actor
	revised.DecidedBy = ""
	revised.EffectiveDate = s.today()
	saved, err := s.repo.SaveDecree(ctx, revised)
	if err != nil {
		return contracts.SalaryDecree{}, err
	}
	s.record(actor, domain.ActionRevise, saved, "اصلاح حکم حقوق "+current.ID)
	return saved.ToSalaryDecree(), nil
}

// DeleteDecree حذف حکم پیش‌نویس.
// IssueGroupDecree صدور گروهی حکم کارگزینی بر پایهٔ نوع استخدام.
//
// قاعدهٔ کسب‌وکار اینجا نیست: مبلغ تازه با `domain.GroupNewSalary` حساب می‌شود و
// انتخاب هدف‌ها از پورت کارکنان می‌آید. اگر DryRun باشد هیچ حکمی ذخیره نمی‌شود و فقط
// پیش‌نمایش برگردانده می‌شود. کارکنان خارج از محدودهٔ نقش در نتیجه «skipped» می‌شوند.
func (s *Service) IssueGroupDecree(ctx context.Context, in contracts.SalaryDecreeGroupInput, actor string) (contracts.SalaryDecreeGroupResult, error) {
	if errs := domain.ValidateGroupDecree(in); len(errs) > 0 {
		return contracts.SalaryDecreeGroupResult{}, &ValidationError{Details: errs}
	}

	result := contracts.SalaryDecreeGroupResult{DryRun: in.DryRun, EmploymentType: in.EmploymentType, Rows: []contracts.SalaryDecreeGroupRow{}}

	targets := []contracts.Employee{}
	if len(in.EmployeeIDs) > 0 {
		for _, id := range in.EmployeeIDs {
			emp, ok, err := s.employees.ByID(ctx, id)
			if err != nil {
				return contracts.SalaryDecreeGroupResult{}, err
			}
			if ok && emp.Status != "terminated" {
				targets = append(targets, emp)
			}
		}
	} else {
		list, err := s.employees.ListByEmploymentType(ctx, in.EmploymentType, in.DepartmentID)
		if err != nil {
			return contracts.SalaryDecreeGroupResult{}, err
		}
		targets = list
	}

	for _, emp := range targets {
		row := contracts.SalaryDecreeGroupRow{
			EmployeeID:       emp.ID,
			EmployeeName:     emp.FullName,
			DepartmentName:   emp.DepartmentName,
			EmploymentType:   emp.EmploymentType,
			EmploymentTypeFa: contracts.EmploymentTypeTitleFa(emp.EmploymentType),
			OldSalary:        emp.BaseSalary,
			NewSalary:        domain.GroupNewSalary(emp.BaseSalary, in.IncreasePercent, in.IncreaseAmount, in.RoundTo),
		}
		row.Delta = row.NewSalary - row.OldSalary

		if in.DryRun {
			row.Status = "preview"
			result.Rows = append(result.Rows, row)
			result.TotalDelta += row.Delta
			continue
		}

		id, err := s.repo.NextDecreeID()
		if err != nil {
			row.Status, row.ErrorFa = "failed", err.Error()
			result.Failed++
			result.Rows = append(result.Rows, row)
			continue
		}
		decree := domain.SalaryDecree{
			ID:             id,
			EmployeeID:     emp.ID,
			EmployeeName:   emp.FullName,
			DepartmentName: emp.DepartmentName,
			Position:       emp.Position,
			EmploymentType: employmentTypeOf("", emp.EmploymentType),
			JobGroup:       jobGroupOf(emp),
			BaseSalary:     row.NewSalary,
			EffectiveDate:  in.EffectiveDate,
			ReasonFa:       in.ReasonFa,
			Status:         domain.DecreeDraft,
			NoteFa:         in.NoteFa,
			CreatedBy:      actor,
		}
		saved, err := s.repo.SaveDecree(ctx, decree)
		if err != nil {
			row.Status, row.ErrorFa = "failed", err.Error()
			result.Failed++
			result.Rows = append(result.Rows, row)
			continue
		}
		s.record(actor, "create", saved, "صدور گروهی حکم — "+contracts.EmploymentTypeTitleFa(emp.EmploymentType))
		row.DecreeID = saved.ID
		row.Status = "draft"
		result.Created++
		result.TotalDelta += row.Delta
		result.Rows = append(result.Rows, row)
	}

	result.Skipped = len(in.EmployeeIDs) - len(targets)
	if result.Skipped < 0 {
		result.Skipped = 0
	}
	return result, nil
}

// employmentTypeOf نوع استخدام را برمی‌گرداند؛ خالی → استنتاج از رکورد کارمند.
func employmentTypeOf(input, employeeType string) string {
	if t, ok := contracts.EmploymentTypeByKey(input); ok {
		return t.Key
	}
	if t, ok := contracts.EmploymentTypeByKey(employeeType); ok {
		return t.Key
	}
	return "official_oil"
}

// jobGroupOf گروه شغلی پیش‌فرض حکم گروهی را از پست کارمند می‌سازد (قابل ویرایش بعدی).
func jobGroupOf(emp contracts.Employee) string {
	if t, ok := contracts.EmploymentTypeByKey(emp.EmploymentType); ok {
		return t.GroupFa
	}
	return "گروه عمومی"
}

func (s *Service) DeleteDecree(ctx context.Context, id, actor string) error {
	current, ok, err := s.repo.GetDecree(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return infra.ErrDecreeNotFound
	}
	if current.Status != domain.DecreeDraft {
		return &RuleError{Message: "تنها پیش‌نویس حکم قابل حذف است."}
	}
	if err := s.repo.DeleteDecree(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Append(actor, "salary_admin", domain.ActionDelete, id, "حذف پیش‌نویس حکم حقوق "+current.EmployeeName)
	}
	return nil
}

func (s *Service) decreeTransition(ctx context.Context, id, actor, target, action, detail string) (domain.SalaryDecree, error) {
	current, ok, err := s.repo.GetDecree(ctx, id)
	if err != nil {
		return domain.SalaryDecree{}, err
	}
	if !ok {
		return domain.SalaryDecree{}, infra.ErrDecreeNotFound
	}
	if !domain.CanDecreeTransition(current.Status, target) {
		return domain.SalaryDecree{}, &RuleError{Message: "این تغییر وضعیت حکم از وضعیت فعلی مجاز نیست."}
	}
	current.Status = target
	if target == domain.DecreeNotified {
		current.DecidedBy = actor
	}
	saved, err := s.repo.SaveDecree(ctx, current)
	if err != nil {
		return domain.SalaryDecree{}, err
	}
	s.record(actor, action, saved, detail)
	if target == domain.DecreeNotified && saved.SupersedesID != "" {
		superseded := saved.SupersedesID
		previous, ok, err := s.repo.GetDecree(ctx, superseded)
		if err == nil && ok && previous.Status != domain.DecreeSuperseded {
			previous.Status = domain.DecreeSuperseded
			previous.DecidedBy = actor
			previous.NoteFa = "با صدور و ابلاغ حکم " + saved.ID + " منسوخ شد"
			if _, err := s.repo.SaveDecree(ctx, previous); err != nil {
				return domain.SalaryDecree{}, err
			}
			if s.audit != nil {
				s.audit.Append(actor, "salary_admin", domain.ActionSupersede, previous.ID, "منسوخ‌سازی با حکم "+saved.ID)
			}
		}
	}
	return saved, nil
}

/* ------------------------------ گواهی اشتغال ------------------------------ */

// ListCertificates فهرست گواهی‌ها.
func (s *Service) ListCertificates(ctx context.Context, f contracts.CertificateFilter) (contracts.CertificatePage, error) {
	return s.repo.ListCertificates(ctx, f)
}

// GetCertificate خواندن یک گواهی با بررسی محدودهٔ دسترسی.
func (s *Service) GetCertificate(ctx context.Context, f contracts.CertificateFilter, id string) (contracts.EmploymentCertificate, bool, error) {
	c, ok, err := s.repo.GetCertificate(ctx, id)
	if err != nil || !ok {
		return contracts.EmploymentCertificate{}, false, err
	}
	if !withinScope(c.EmployeeID, c.DepartmentName, f.Scope, f.ActorEmployeeID, f.ActorDepartment) {
		return contracts.EmploymentCertificate{}, false, nil
	}
	return c.ToCertificate(), true, nil
}

// CreateCertificate ثبت درخواست گواهی اشتغال.
func (s *Service) CreateCertificate(ctx context.Context, in contracts.CertificateInput, actor string) (contracts.EmploymentCertificate, error) {
	if errs := domain.ValidateCertificate(in); len(errs) > 0 {
		return contracts.EmploymentCertificate{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	if !ok {
		return contracts.EmploymentCertificate{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}
	id, serial, err := s.repo.NextCertificateID()
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	cert := domain.EmploymentCertificate{
		ID:             id,
		Serial:         serial,
		EmployeeID:     employee.ID,
		EmployeeName:   employee.FullName,
		DepartmentName: employee.DepartmentName,
		Position:       employee.Position,
		Kind:           in.Kind,
		PurposeFa:      in.PurposeFa,
		RequestedAt:    s.today(),
		Status:         domain.CertRequested,
		NoteFa:         in.NoteFa,
	}
	saved, err := s.repo.SaveCertificate(ctx, cert)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	s.recordCert(actor, domain.ActionRequest, saved, "ثبت درخواست گواهی "+domain.CertKindFa[saved.Kind])
	return saved.ToCertificate(), nil
}

// DecideCertificate تأیید یا رد درخواست گواهی.
func (s *Service) DecideCertificate(ctx context.Context, id string, in contracts.CertificateDecision, actor string) (contracts.EmploymentCertificate, error) {
	current, ok, err := s.repo.GetCertificate(ctx, id)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	if !ok {
		return contracts.EmploymentCertificate{}, infra.ErrCertificateNotFound
	}
	if current.Status != domain.CertRequested {
		return contracts.EmploymentCertificate{}, &RuleError{Message: "تنها گواهی در انتظار تأیید قابل تصمیم است."}
	}
	if !in.Approve && in.NoteFa == "" {
		return contracts.EmploymentCertificate{}, &ValidationError{Details: map[string]string{"noteFa": "برای رد درخواست، ثبت دلیل الزامی است."}}
	}
	target := domain.CertApproved
	action := domain.ActionApprove
	detail := "تأیید درخواست گواهی"
	if !in.Approve {
		target = domain.CertRejected
		action = domain.ActionReject
		detail = "رد درخواست گواهی"
	}
	current.Status = target
	current.DecidedBy = actor
	if in.NoteFa != "" {
		current.NoteFa = in.NoteFa
	}
	saved, err := s.repo.SaveCertificate(ctx, current)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	s.recordCert(actor, action, saved, detail)
	return saved.ToCertificate(), nil
}

// IssueCertificate صدور گواهی تأییدشده.
func (s *Service) IssueCertificate(ctx context.Context, id, actor string) (contracts.EmploymentCertificate, error) {
	current, ok, err := s.repo.GetCertificate(ctx, id)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	if !ok {
		return contracts.EmploymentCertificate{}, infra.ErrCertificateNotFound
	}
	if !domain.CanCertTransition(current.Status, domain.CertIssued) {
		return contracts.EmploymentCertificate{}, &RuleError{Message: "تنها گواهی تأییدشده قابل صدور است."}
	}
	current.Status = domain.CertIssued
	current.IssuedAt = s.today()
	current.DecidedBy = actor
	saved, err := s.repo.SaveCertificate(ctx, current)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	s.recordCert(actor, domain.ActionIssue, saved, "صدور گواهی اشتغال")
	return saved.ToCertificate(), nil
}

// DeliverCertificate ثبت تحویل گواهی صادرشده.
func (s *Service) DeliverCertificate(ctx context.Context, id, actor string) (contracts.EmploymentCertificate, error) {
	current, ok, err := s.repo.GetCertificate(ctx, id)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	if !ok {
		return contracts.EmploymentCertificate{}, infra.ErrCertificateNotFound
	}
	if !domain.CanCertTransition(current.Status, domain.CertDelivered) {
		return contracts.EmploymentCertificate{}, &RuleError{Message: "تنها گواهی صادرشده قابل تحویل است."}
	}
	current.Status = domain.CertDelivered
	current.DeliveredAt = s.today()
	saved, err := s.repo.SaveCertificate(ctx, current)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	s.recordCert(actor, domain.ActionDeliver, saved, "تحویل گواهی اشتغال")
	return saved.ToCertificate(), nil
}

// UpdateCertificate ویرایش درخواست گواهی (فقط در وضعیت درخواست).
func (s *Service) UpdateCertificate(ctx context.Context, id string, in contracts.CertificateInput, actor string) (contracts.EmploymentCertificate, error) {
	current, ok, err := s.repo.GetCertificate(ctx, id)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	if !ok {
		return contracts.EmploymentCertificate{}, infra.ErrCertificateNotFound
	}
	if current.Status != domain.CertRequested {
		return contracts.EmploymentCertificate{}, &RuleError{Message: "تنها گواهی در وضعیت درخواست قابل ویرایش است."}
	}
	if errs := domain.ValidateCertificate(in); len(errs) > 0 {
		return contracts.EmploymentCertificate{}, &ValidationError{Details: errs}
	}
	if in.Kind != "" {
		current.Kind = in.Kind
	}
	current.PurposeFa = in.PurposeFa
	current.NoteFa = in.NoteFa
	saved, err := s.repo.SaveCertificate(ctx, current)
	if err != nil {
		return contracts.EmploymentCertificate{}, err
	}
	return saved.ToCertificate(), nil
}

// DeleteCertificate حذف درخواست گواهی (فقط در وضعیت درخواست).
func (s *Service) DeleteCertificate(ctx context.Context, id, actor string) error {
	current, ok, err := s.repo.GetCertificate(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return infra.ErrCertificateNotFound
	}
	if current.Status != domain.CertRequested {
		return &RuleError{Message: "تنها درخواست گواهی قابل حذف است."}
	}
	return s.repo.DeleteCertificate(ctx, id)
}

/* -------------------------------- کمکی‌ها -------------------------------- */

func (s *Service) record(actor, action string, d domain.SalaryDecree, detail string) {
	if s.audit != nil {
		s.audit.Append(actor, "salary_admin", action, d.ID, detail+" — "+d.EmployeeName)
	}
	if s.sync != nil {
		s.sync.Record("SalaryDecree", synclog.OpUpsert, d.ID, d.Version, map[string]any{
			"id":            d.ID,
			"employeeId":    d.EmployeeID,
			"employeeName":  d.EmployeeName,
			"jobGroup":      d.JobGroup,
			"baseSalary":    d.BaseSalary,
			"effectiveDate": d.EffectiveDate,
			"status":        d.Status,
			"version":       d.Version,
		})
	}
}

func (s *Service) recordCert(actor, action string, c domain.EmploymentCertificate, detail string) {
	if s.audit != nil {
		s.audit.Append(actor, "salary_admin", action, c.ID, detail+" — "+c.EmployeeName)
	}
	if s.sync != nil {
		s.sync.Record("EmploymentCertificate", synclog.OpUpsert, c.ID, c.Version, map[string]any{
			"id":           c.ID,
			"serial":       c.Serial,
			"employeeId":   c.EmployeeID,
			"employeeName": c.EmployeeName,
			"kind":         c.Kind,
			"status":       c.Status,
			"version":      c.Version,
		})
	}
}

func withinScope(employeeID, department, scope, actorID, actorDepartment string) bool {
	switch scope {
	case "own":
		return actorID != "" && employeeID == actorID
	case "own_unit":
		return actorDepartment == "" || department == actorDepartment
	default:
		return true
	}
}
