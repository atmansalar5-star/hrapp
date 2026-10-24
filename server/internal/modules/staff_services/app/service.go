// Package app لایهٔ کاربرد ماژول تکفل و درخواست پرواز.
//
// دو گردش‌کار اینجا اداره می‌شوند: تکفل (ثبت → بررسی/تأیید → خروج) و فرم پرواز
// (ثبت → تأیید مدیر واحد → صدور بلیت → تسویه/لغو). قواعد خالص در domain است؛ این لایه
// فقط ترتیب، دسترسی به کارمند، و ثبت حسابرسی/هم‌گام‌سازی را می‌داند.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/staff_services/domain"
	"hrms/internal/modules/staff_services/infra"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/synclog"
)

// Repository قرارداد ذخیره‌سازی هر دو موجودیت ماژول.
type Repository interface {
	GetDependent(ctx context.Context, id string) (domain.Dependent, bool, error)
	ListDependents(ctx context.Context, f contracts.DependentFilter) (contracts.DependentPage, error)
	SaveDependent(ctx context.Context, d domain.Dependent) (domain.Dependent, error)
	DeleteDependent(ctx context.Context, id string) error
	NextDependentID() (string, error)

	GetFlight(ctx context.Context, id string) (domain.FlightRequest, bool, error)
	ListFlights(ctx context.Context, f contracts.FlightRequestFilter) (contracts.FlightRequestPage, error)
	SaveFlight(ctx context.Context, f domain.FlightRequest) (domain.FlightRequest, error)
	NextFlightID() (string, error)
}

// EmployeeLookup پورت خواندن کارمند (از راه contracts.EmployeeReader).
type EmployeeLookup interface {
	ByID(ctx context.Context, employeeID string) (contracts.Employee, bool, error)
}

// LookupFromReader پوشش کوچک روی پورت کارکنان.
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

// ValidationError خطای اعتبارسنجی فیلد‌به‌فیلد.
type ValidationError struct{ Details map[string]string }

func (e *ValidationError) Error() string { return "اطلاعات ارسالی معتبر نیست." }

// RuleError نقض قاعدهٔ کسب‌وکار.
type RuleError struct{ Message string }

func (e *RuleError) Error() string { return e.Message }

// Service خدمات تکفل و درخواست پرواز.
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

func (s *Service) today() calendar.Date {
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return calendar.Date{Year: 1400, Month: 1, Day: 1}
	}
	return d
}

// ---------- افراد تحت تکفل ----------

// ListDependents فهرست تکفل‌شدگان با محدودهٔ دسترسی نقش.
func (s *Service) ListDependents(ctx context.Context, f contracts.DependentFilter) (contracts.DependentPage, error) {
	if f.Today == "" {
		f.Today = s.today().String()
	}
	return s.repo.ListDependents(ctx, f)
}

// GetDependent یک تکفل با بررسی محدودهٔ دسترسی.
func (s *Service) GetDependent(ctx context.Context, f contracts.DependentFilter, id string) (contracts.Dependent, bool, error) {
	d, ok, err := s.repo.GetDependent(ctx, id)
	if err != nil || !ok {
		return contracts.Dependent{}, false, err
	}
	if !withinScope(d.EmployeeID, d.DepartmentName, f.Scope, f.ActorEmployeeID, f.ActorDepartment) {
		return contracts.Dependent{}, false, nil
	}
	return d.ToContract(s.today()), true, nil
}

// CreateDependent ثبت فرد تحت تکفل (وضعیت: در انتظار بررسی).
func (s *Service) CreateDependent(ctx context.Context, in contracts.DependentInput, actor string) (contracts.Dependent, error) {
	if errs := domain.ValidateDependent(in); len(errs) > 0 {
		return contracts.Dependent{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.Dependent{}, err
	}
	if !ok {
		return contracts.Dependent{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}
	if err := s.ensureNoDuplicateDependent(ctx, in.EmployeeID, in.NationalID, ""); err != nil {
		return contracts.Dependent{}, err
	}

	id, err := s.repo.NextDependentID()
	if err != nil {
		return contracts.Dependent{}, err
	}
	d := domain.FromDependentInput(id, in, employee.FullName, employee.DepartmentName)
	saved, err := s.repo.SaveDependent(ctx, d)
	if err != nil {
		return contracts.Dependent{}, err
	}
	s.recordDependent(actor, domain.DependentActionCreate, saved, "ثبت فرد تحت تکفل")
	return saved.ToContract(s.today()), nil
}

// UpdateDependent ویرایش تکفل در انتظار بررسی یا ردشده.
func (s *Service) UpdateDependent(ctx context.Context, id string, in contracts.DependentInput, actor string) (contracts.Dependent, error) {
	current, ok, err := s.repo.GetDependent(ctx, id)
	if err != nil {
		return contracts.Dependent{}, err
	}
	if !ok {
		return contracts.Dependent{}, infra.ErrNotFound
	}
	if current.Status == domain.DependentRemoved {
		return contracts.Dependent{}, &RuleError{Message: "این تکفل خارج شده است؛ برای بازگشت، ثبت تازه انجام دهید."}
	}
	if errs := domain.ValidateDependent(in); len(errs) > 0 {
		return contracts.Dependent{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.Dependent{}, err
	}
	if !ok {
		return contracts.Dependent{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}
	if err := s.ensureNoDuplicateDependent(ctx, in.EmployeeID, in.NationalID, id); err != nil {
		return contracts.Dependent{}, err
	}

	updated := domain.ApplyDependentInput(current, in, employee.FullName, employee.DepartmentName)
	if updated.Status == domain.DependentRejected {
		updated.Status = domain.DependentPending
	}
	saved, err := s.repo.SaveDependent(ctx, updated)
	if err != nil {
		return contracts.Dependent{}, err
	}
	s.recordDependent(actor, domain.DependentActionUpdate, saved, "ویرایش اطلاعات تکفل")
	return saved.ToContract(s.today()), nil
}

// DecideDependent تأیید یا رد تکفل.
func (s *Service) DecideDependent(ctx context.Context, id string, in contracts.ContractDecision, actor string) (contracts.Dependent, error) {
	current, ok, err := s.repo.GetDependent(ctx, id)
	if err != nil {
		return contracts.Dependent{}, err
	}
	if !ok {
		return contracts.Dependent{}, infra.ErrNotFound
	}
	if current.Status != domain.DependentPending {
		return contracts.Dependent{}, &RuleError{Message: "تنها تکفل در انتظار بررسی قابل تصمیم است."}
	}
	if in.Approve {
		// تأیید تکفل بدون مدرک هویتی ثبت نمی‌شود؛ پوشش بیمه بر پایهٔ همین مدارک است.
		if strings.TrimSpace(current.DocumentsFa) == "" {
			return contracts.Dependent{}, &ValidationError{Details: map[string]string{"documentsFa": "برای تأیید تکفل، ثبت مدارک هویتی الزامی است."}}
		}
		current.Status = domain.DependentApproved
	} else {
		if strings.TrimSpace(in.NoteFa) == "" {
			return contracts.Dependent{}, &ValidationError{Details: map[string]string{"noteFa": "برای رد تکفل ثبت دلیل الزامی است."}}
		}
		current.Status = domain.DependentRejected
		current.NoteFa = in.NoteFa
	}
	current.DecidedBy = actor
	saved, err := s.repo.SaveDependent(ctx, current)
	if err != nil {
		return contracts.Dependent{}, err
	}
	s.recordDependent(actor, domain.DependentActionDecide, saved, domain.DependentStatusFa[saved.Status])
	return saved.ToContract(s.today()), nil
}

// RemoveDependent خروج فرد از تکفل (با ثبت دلیل).
func (s *Service) RemoveDependent(ctx context.Context, id, note, actor string) (contracts.Dependent, error) {
	current, ok, err := s.repo.GetDependent(ctx, id)
	if err != nil {
		return contracts.Dependent{}, err
	}
	if !ok {
		return contracts.Dependent{}, infra.ErrNotFound
	}
	if current.Status != domain.DependentApproved {
		return contracts.Dependent{}, &RuleError{Message: "تنها تکفل تأییدشده قابل خروج است."}
	}
	if strings.TrimSpace(note) == "" {
		return contracts.Dependent{}, &ValidationError{Details: map[string]string{"noteFa": "برای خروج از تکفل ثبت دلیل الزامی است."}}
	}
	current.Status = domain.DependentRemoved
	current.Insurance = false
	current.NoteFa = note
	current.DecidedBy = actor
	saved, err := s.repo.SaveDependent(ctx, current)
	if err != nil {
		return contracts.Dependent{}, err
	}
	s.recordDependent(actor, domain.DependentActionRemove, saved, "خروج از تکفل")
	return saved.ToContract(s.today()), nil
}

// DeleteDependent حذف ثبت ردشده (پاک‌سازی ورودی نادرست).
func (s *Service) DeleteDependent(ctx context.Context, id, actor string) error {
	current, ok, err := s.repo.GetDependent(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return infra.ErrNotFound
	}
	if current.Status != domain.DependentRejected {
		return &RuleError{Message: "تنها تکفل ردشده قابل حذف است."}
	}
	if err := s.repo.DeleteDependent(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Append(actor, "staff_services", "delete", id, "حذف ثبت ردشدهٔ "+current.FullName)
	}
	return nil
}

// ensureNoDuplicateDependent جلوی ثبت دوبارهٔ یک کد ملی برای همان کارمند را می‌گیرد.
func (s *Service) ensureNoDuplicateDependent(ctx context.Context, employeeID, nationalID, ignoreID string) error {
	nationalID = calendar.NormalizeDigits(strings.TrimSpace(nationalID))
	if nationalID == "" {
		return nil
	}
	page, err := s.repo.ListDependents(ctx, contracts.DependentFilter{EmployeeID: employeeID, Page: 1, Size: 200})
	if err != nil {
		return err
	}
	for _, d := range page.Items {
		if d.ID == ignoreID || d.NationalID != nationalID {
			continue
		}
		if d.Status == domain.DependentRemoved || d.Status == domain.DependentRejected {
			continue
		}
		return &ValidationError{Details: map[string]string{
			"nationalId": "این فرد با همین کد ملی برای کارمند ثبت شده است (شمارهٔ " + d.ID + ").",
		}}
	}
	return nil
}

// ---------- فرم درخواست پرواز ----------

// ListFlights فهرست درخواست‌های پرواز.
func (s *Service) ListFlights(ctx context.Context, f contracts.FlightRequestFilter) (contracts.FlightRequestPage, error) {
	return s.repo.ListFlights(ctx, f)
}

// GetFlight یک درخواست پرواز با بررسی محدودهٔ دسترسی.
func (s *Service) GetFlight(ctx context.Context, f contracts.FlightRequestFilter, id string) (contracts.FlightRequest, bool, error) {
	req, ok, err := s.repo.GetFlight(ctx, id)
	if err != nil || !ok {
		return contracts.FlightRequest{}, false, err
	}
	if !withinScope(req.EmployeeID, req.DepartmentName, f.Scope, f.ActorEmployeeID, f.ActorDepartment) {
		return contracts.FlightRequest{}, false, nil
	}
	return req.ToContract(), true, nil
}

// CreateFlight ثبت فرم درخواست پرواز (وضعیت: در انتظار تأیید مدیر واحد).
func (s *Service) CreateFlight(ctx context.Context, in contracts.FlightRequestInput, actor string) (contracts.FlightRequest, error) {
	if errs := domain.ValidateFlight(in, s.today()); len(errs) > 0 {
		return contracts.FlightRequest{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.FlightRequest{}, err
	}
	if !ok {
		return contracts.FlightRequest{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}
	id, err := s.repo.NextFlightID()
	if err != nil {
		return contracts.FlightRequest{}, err
	}
	req := domain.FromFlightInput(id, in, employee.FullName, employee.DepartmentName)
	saved, err := s.repo.SaveFlight(ctx, req)
	if err != nil {
		return contracts.FlightRequest{}, err
	}
	s.recordFlight(actor, "create", saved, "ثبت درخواست پرواز")
	return saved.ToContract(), nil
}

// UpdateFlight ویرایش فرم پرواز تا پیش از تصمیم.
func (s *Service) UpdateFlight(ctx context.Context, id string, in contracts.FlightRequestInput, actor string) (contracts.FlightRequest, error) {
	current, ok, err := s.repo.GetFlight(ctx, id)
	if err != nil {
		return contracts.FlightRequest{}, err
	}
	if !ok {
		return contracts.FlightRequest{}, infra.ErrNotFound
	}
	if current.Status != domain.FlightPending {
		return contracts.FlightRequest{}, &RuleError{Message: "پس از تصمیم مدیر، ویرایش فرم مجاز نیست؛ درخواست را لغو و فرم تازه ثبت کنید."}
	}
	if errs := domain.ValidateFlight(in, s.today()); len(errs) > 0 {
		return contracts.FlightRequest{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.FlightRequest{}, err
	}
	if !ok {
		return contracts.FlightRequest{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}
	updated := domain.ApplyFlightInput(current, in, employee.FullName, employee.DepartmentName)
	saved, err := s.repo.SaveFlight(ctx, updated)
	if err != nil {
		return contracts.FlightRequest{}, err
	}
	s.recordFlight(actor, "update", saved, "ویرایش فرم درخواست پرواز")
	return saved.ToContract(), nil
}

// DecideFlight اعمال اقدام گردش‌کار روی فرم پرواز (تأیید، رد، صدور، تسویه، لغو).
func (s *Service) DecideFlight(ctx context.Context, id string, in contracts.FlightDecision, actor string) (contracts.FlightRequest, error) {
	current, ok, err := s.repo.GetFlight(ctx, id)
	if err != nil {
		return contracts.FlightRequest{}, err
	}
	if !ok {
		return contracts.FlightRequest{}, infra.ErrNotFound
	}
	target, known := domain.TargetOfFlightAction(in.Action)
	if !known {
		return contracts.FlightRequest{}, &ValidationError{Details: map[string]string{"action": "اقدام درخواستی شناخته‌شده نیست."}}
	}
	if !domain.CanTransitionFlight(current.Status, target) {
		return contracts.FlightRequest{}, &RuleError{Message: fmt.Sprintf(
			"اقدام «%s» از وضعیت «%s» مجاز نیست.", actionFa(in.Action), domain.FlightStatusFa[current.Status])}
	}

	switch target {
	case domain.FlightRejected, domain.FlightCancelled:
		if strings.TrimSpace(in.NoteFa) == "" {
			return contracts.FlightRequest{}, &ValidationError{Details: map[string]string{"noteFa": "برای رد یا لغو، ثبت دلیل الزامی است."}}
		}
	case domain.FlightIssued:
		if strings.TrimSpace(in.TicketNo) == "" {
			return contracts.FlightRequest{}, &ValidationError{Details: map[string]string{"ticketNo": "برای ثبت صدور بلیت، شمارهٔ بلیت الزامی است."}}
		}
	}

	current.Status = target
	if in.NoteFa != "" {
		current.NoteFa = in.NoteFa
	}
	if in.TicketNo != "" {
		current.TicketNo = strings.TrimSpace(in.TicketNo)
	}
	current.DecidedBy = actor
	saved, err := s.repo.SaveFlight(ctx, current)
	if err != nil {
		return contracts.FlightRequest{}, err
	}
	s.recordFlight(actor, in.Action, saved, "اقدام روی فرم پرواز: "+actionFa(in.Action))
	return saved.ToContract(), nil
}

func actionFa(action string) string {
	switch action {
	case domain.FlightActionApprove:
		return "تأیید"
	case domain.FlightActionReject:
		return "رد"
	case domain.FlightActionIssue:
		return "صدور بلیت"
	case domain.FlightActionSettle:
		return "تسویه"
	case domain.FlightActionCancel:
		return "لغو"
	default:
		return action
	}
}

func (s *Service) recordDependent(actor, action string, d domain.Dependent, detail string) {
	if s.audit != nil {
		s.audit.Append(actor, "staff_services", action, d.ID, detail+" — "+d.FullName)
	}
	if s.sync != nil {
		s.sync.Record("Dependent", synclog.OpUpsert, d.ID, d.Version, map[string]any{
			"id":           d.ID,
			"employeeId":   d.EmployeeID,
			"employeeName": d.EmployeeName,
			"fullName":     d.FullName,
			"relation":     d.Relation,
			"status":       d.Status,
			"insurance":    d.Insurance,
			"version":      d.Version,
		})
	}
}

func (s *Service) recordFlight(actor, action string, f domain.FlightRequest, detail string) {
	if s.audit != nil {
		s.audit.Append(actor, "staff_services", "flight-"+action, f.ID, detail+" — "+f.EmployeeName)
	}
	if s.sync != nil {
		s.sync.Record("FlightRequest", synclog.OpUpsert, f.ID, f.Version, map[string]any{
			"id":            f.ID,
			"employeeId":    f.EmployeeID,
			"employeeName":  f.EmployeeName,
			"originFa":      f.OriginFa,
			"destinationFa": f.DestinationFa,
			"departDate":    f.DepartDate,
			"status":        f.Status,
			"version":       f.Version,
		})
	}
}

func withinScope(employeeID, departmentName, scope, actorID, actorDepartment string) bool {
	switch scope {
	case "own":
		return actorID != "" && employeeID == actorID
	case "own_unit":
		return actorDepartment == "" || departmentName == actorDepartment
	default:
		return true
	}
}
