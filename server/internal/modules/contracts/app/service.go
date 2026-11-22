// Package app لایهٔ کاربرد ماژول قراردادها.
//
// گردش‌کار اینجاست: پیش‌نویس → در انتظار تأیید → جاری (یا رد)، تمدید، تبدیل مدت موقت
// به دائم و خاتمه. قواعد خالص در domain است و این لایه فقط آن‌ها را به هم می‌بندد،
// دسترسی به کارمند را از پورت می‌گیرد و رخداد حسابرسی/هم‌گام‌سازی را ثبت می‌کند.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/contracts/domain"
	"hrms/internal/modules/contracts/infra"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/synclog"
)

// Repository قرارداد ذخیره‌سازی؛ هر دو انبار (حافظه و Postgres) آن را برمی‌آورند.
type Repository interface {
	Get(ctx context.Context, id string) (domain.Contract, bool, error)
	ActiveForEmployee(ctx context.Context, employeeID string) ([]domain.Contract, error)
	List(ctx context.Context, f contracts.ContractFilter) (contracts.ContractPage, error)
	Save(ctx context.Context, c domain.Contract) (domain.Contract, error)
	Delete(ctx context.Context, id string) error
	NextID() (string, error)
}

// EmployeeLookup پورت خواندن کارمند — از راه contracts.EmployeeReader می‌آید، پس این
// ماژول هیچ import‌ی به داخلی‌های ماژول کارکنان ندارد.
type EmployeeLookup interface {
	ByID(ctx context.Context, employeeID string) (contracts.Employee, bool, error)
}

// LookupFromReader پوشش کوچک روی پورت کارکنان: خواندن یک کارمند در محدودهٔ سازمانی.
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

func (e *ValidationError) Error() string { return "اطلاعات قرارداد معتبر نیست." }

// RuleError نقض یک قاعدهٔ کسب‌وکار که به فیلد خاصی وصل نیست.
type RuleError struct{ Message string }

func (e *RuleError) Error() string { return e.Message }

// Service خدمات قراردادها.
type Service struct {
	repo      Repository
	employees EmployeeLookup
	audit     *audit.Log
	sync      *synclog.Log
	clock     func() time.Time
}

// New سرویس می‌سازد؛ آدیت، سینک و ساعت اختیاری‌اند تا آزمون ساده بماند.
func New(repo Repository, employees EmployeeLookup, auditLog *audit.Log, syncLog *synclog.Log, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{repo: repo, employees: employees, audit: auditLog, sync: syncLog, clock: clock}
}

// payFieldErrors خطاهای زیرفرم را به شکل «فیلد → پیام» برای پاسخ ۴۲۲ آماده می‌کند.
func payFieldErrors(errs []domain.PayFieldError) map[string]string {
	out := make(map[string]string, len(errs))
	for _, e := range errs {
		if _, exists := out[e.Field]; !exists {
			out[e.Field] = e.Message
		}
	}
	return out
}

// cleanPayMeta فرادادهٔ زیرفرم را پاک‌سازی می‌کند: کلید/مقدار تهی نگه داشته نمی‌شود و
// کلیدهای اجزای مزد رسمی (که ستون ثابت خودشان را دارند) هرگز در فراداده نمی‌نشیند.
func cleanPayMeta(meta map[string]any) map[string]any {
	if len(meta) == 0 {
		return nil
	}
	out := make(map[string]any, len(meta))
	for key, value := range meta {
		key = strings.TrimSpace(key)
		if key == "" || officialWageKeys[key] {
			continue
		}
		switch v := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(v) == "" {
				continue
			}
			out[key] = strings.TrimSpace(v)
		default:
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// officialWageKeys کلیدهایی که ستون ثابت خودشان را دارند و در فراداده تکرار نمی‌شوند.
var officialWageKeys = map[string]bool{
	"climateRegion": true, "rankWage": true, "seniorityWage": true, "postAllowance": true,
	"sacrificeAllowance": true, "climateAllowance": true, "salaryRepair": true, "payGap": true,
	"payGapSince": true, "badClimatePercent": true, "deprivationPercent": true,
	"regionHardshipPercent": true, "extraTimePercent": true, "monthlySalary": true,
}

// today تاریخ امروز شمسی.
func (s *Service) today() calendar.Date {
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return calendar.Date{Year: 1400, Month: 1, Day: 1}
	}
	return d
}

// List فهرست قراردادها با محدودهٔ دسترسی نقش.
func (s *Service) List(ctx context.Context, f contracts.ContractFilter) (contracts.ContractPage, error) {
	if f.Today == "" {
		f.Today = s.today().String()
	}
	return s.repo.List(ctx, f)
}

// Get خواندن یک قرارداد با بررسی محدودهٔ دسترسی (خارج از محدوده = یافت نشد).
func (s *Service) Get(ctx context.Context, f contracts.ContractFilter, id string) (contracts.Contract, bool, error) {
	c, ok, err := s.repo.Get(ctx, id)
	if err != nil || !ok {
		return contracts.Contract{}, false, err
	}
	if !withinScope(c, f) {
		return contracts.Contract{}, false, nil
	}
	return c.ToContract(s.today()), true, nil
}

// ActiveOfContractEmployee قرارداد جاری یک کارمند (پایهٔ سند حکم حقوق و گواهی اشتغال).
func (s *Service) ContractByID(ctx context.Context, id string) (contracts.Contract, bool, error) {
	c, ok, err := s.repo.Get(ctx, id)
	if err != nil || !ok {
		return contracts.Contract{}, false, err
	}
	return c.ToContract(s.today()), true, nil
}

// ActiveContractOf قرارداد جاری کارمند؛ برای اسناد چاپی و صفحهٔ کارمند.
func (s *Service) ActiveContractOf(ctx context.Context, employeeID string) (contracts.Contract, bool, error) {
	list, err := s.repo.ActiveForEmployee(ctx, employeeID)
	if err != nil {
		return contracts.Contract{}, false, err
	}
	today := s.today()
	for _, c := range list {
		if c.Status == domain.StatusActive {
			return c.ToContract(today), true, nil
		}
	}
	return contracts.Contract{}, false, nil
}

// Create انعقاد قرارداد تازه (وضعیت پیش‌نویس).
func (s *Service) Create(ctx context.Context, in contracts.ContractInput, actor string) (contracts.Contract, error) {
	if errs := domain.Validate(in); len(errs) > 0 {
		return contracts.Contract{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.Contract{}, err
	}
	if !ok {
		return contracts.Contract{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}

	employmentType := employmentTypeOf(in.EmploymentType, in.Kind, employee.EmploymentType)
	// زیرفرم «حقوق و مزایا» بر پایهٔ نوع استخدام اعتبارسنجی می‌شود: همان تعریفی که سرور
	// به رابط می‌دهد (GET /api/v1/contracts/forms)، پس UI و API هرگز واگرا نمی‌شوند.
	if errs := domain.ValidatePayForm(domain.PayInput{
		Kind:           in.Kind,
		EmploymentType: employmentType,
		MonthlySalary:  in.MonthlySalary,
		Wage:           in.WageStructure,
		Meta:           in.PayMeta,
		PayModel:       in.PayModel,
		PayGroup:       in.PayGroup,
	}); len(errs) > 0 {
		return contracts.Contract{}, &ValidationError{Details: payFieldErrors(errs)}
	}
	classification := domain.PayClassificationOf(employmentType, in.PayGroup, in.PayModel)

	draft := domain.Contract{
		EmployeeID:     employee.ID,
		EmployeeName:   employee.FullName,
		DepartmentName: employee.DepartmentName,
		Position:       employee.Position,
		Kind:           in.Kind,
		EmploymentType: employmentType,
		StartDate:      in.StartDate,
		EndDate:        in.EndDate,
		MonthlySalary:  in.MonthlySalary,
		Status:         domain.StatusDraft,
		NoteFa:         in.NoteFa,
		CreatedBy:      actor,
		WageStructure:  in.WageStructure,
		PayFamily:      domain.PayFamilyOf(employmentType),
		PayMeta:        cleanPayMeta(in.PayMeta),
		PayGroup:       classification.PayGroup,
		PayModel:       classification.PayModel,
	}
	if err := s.ensureNoActiveOverlap(ctx, draft, ""); err != nil {
		return contracts.Contract{}, err
	}

	id, err := s.repo.NextID()
	if err != nil {
		return contracts.Contract{}, err
	}
	draft.ID = id
	draft.Events = append(draft.Events, s.event(actor, domain.ActionCreate, "انعقاد قرارداد"))

	saved, err := s.repo.Save(ctx, draft)
	if err != nil {
		return contracts.Contract{}, err
	}
	s.record(actor, "create", saved, "انعقاد "+domain.KindFa[saved.Kind])
	return saved.ToContract(s.today()), nil
}

// Update ویرایش قرارداد پیش‌نویس یا ردشده.
func (s *Service) Update(ctx context.Context, id string, in contracts.ContractInput, actor string) (contracts.Contract, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Contract{}, err
	}
	if !ok {
		return contracts.Contract{}, infra.ErrNotFound
	}
	if current.Status != domain.StatusDraft && current.Status != domain.StatusRejected {
		return contracts.Contract{}, &RuleError{Message: "تنها قرارداد پیش‌نویس یا ردشده قابل ویرایش است؛ برای قرارداد جاری از تمدید یا بازنگری حکم استفاده کنید."}
	}
	if errs := domain.Validate(in); len(errs) > 0 {
		return contracts.Contract{}, &ValidationError{Details: errs}
	}
	employee, ok, err := s.employees.ByID(ctx, in.EmployeeID)
	if err != nil {
		return contracts.Contract{}, err
	}
	if !ok {
		return contracts.Contract{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
	}

	employmentType := employmentTypeOf(in.EmploymentType, in.Kind, employee.EmploymentType)
	if errs := domain.ValidatePayForm(domain.PayInput{
		Kind:           in.Kind,
		EmploymentType: employmentType,
		MonthlySalary:  in.MonthlySalary,
		Wage:           in.WageStructure,
		Meta:           in.PayMeta,
		PayModel:       in.PayModel,
		PayGroup:       in.PayGroup,
	}); len(errs) > 0 {
		return contracts.Contract{}, &ValidationError{Details: payFieldErrors(errs)}
	}
	classification := domain.PayClassificationOf(employmentType, in.PayGroup, in.PayModel)

	updated := current
	updated.EmployeeID = employee.ID
	updated.EmployeeName = employee.FullName
	updated.DepartmentName = employee.DepartmentName
	updated.Position = employee.Position
	updated.Kind = in.Kind
	updated.EmploymentType = employmentType
	updated.StartDate = in.StartDate
	updated.EndDate = in.EndDate
	updated.MonthlySalary = in.MonthlySalary
	updated.NoteFa = in.NoteFa
	updated.WageStructure = in.WageStructure
	// زیرفرم تازه: اگر نوع استخدام عوض شده باشد، خانواده و فراداده هم عوض می‌شوند و
	// باقی‌ماندهٔ خانوادهٔ قبلی (مثل مشخصات پیمان) پاک می‌شود.
	updated.PayFamily = domain.PayFamilyOf(employmentType)
	updated.PayMeta = cleanPayMeta(in.PayMeta)
	updated.PayGroup = classification.PayGroup
	updated.PayModel = classification.PayModel
	if updated.Status == domain.StatusRejected {
		updated.Status = domain.StatusDraft
	}
	if err := s.ensureNoActiveOverlap(ctx, updated, id); err != nil {
		return contracts.Contract{}, err
	}
	updated.Events = append(updated.Events, s.event(actor, domain.ActionUpdate, "ویرایش پیش‌نویس قرارداد"))

	saved, err := s.repo.Save(ctx, updated)
	if err != nil {
		return contracts.Contract{}, err
	}
	s.record(actor, "update", saved, "ویرایش قرارداد")
	return saved.ToContract(s.today()), nil
}

// Submit ارسال قرارداد برای تأیید مدیر منابع انسانی.
func (s *Service) Submit(ctx context.Context, id, actor string) (contracts.Contract, error) {
	return s.transition(ctx, id, actor, domain.StatusPending, domain.ActionSubmit, "ارسال برای تأیید")
}

// Decide تأیید یا رد قرارداد در انتظار تأیید.
func (s *Service) Decide(ctx context.Context, id string, in contracts.ContractDecision, actor string) (contracts.Contract, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Contract{}, err
	}
	if !ok {
		return contracts.Contract{}, infra.ErrNotFound
	}
	if current.Status != domain.StatusPending {
		return contracts.Contract{}, &RuleError{Message: "تنها قرارداد در انتظار تأیید قابل تصمیم است."}
	}
	if !in.Approve && in.NoteFa == "" {
		return contracts.Contract{}, &ValidationError{Details: map[string]string{"noteFa": "برای رد قرارداد ثبت دلیل الزامی است."}}
	}

	target := domain.StatusActive
	action := domain.ActionApprove
	detail := "تأیید و ابلاغ قرارداد"
	if !in.Approve {
		target = domain.StatusRejected
		action = domain.ActionReject
		detail = "رد قرارداد"
	}
	if !domain.CanTransition(current.Status, target) {
		return contracts.Contract{}, &RuleError{Message: "این تغییر وضعیت مجاز نیست."}
	}

	if in.Approve {
		// دو قاعدهٔ تأیید: نبود قرارداد جاری هم‌زمان، و شرط سابقه برای تبدیل به دائم.
		if err := s.ensureNoActiveOverlap(ctx, current, current.ID); err != nil {
			return contracts.Contract{}, err
		}
		if current.SupersedesID != "" {
			previous, ok, err := s.repo.Get(ctx, current.SupersedesID)
			if err != nil {
				return contracts.Contract{}, err
			}
			if ok && current.Kind == domain.KindPermanent {
				if errs := domain.ValidateConversion(previous, s.today()); len(errs) > 0 {
					return contracts.Contract{}, &RuleError{Message: errs["serviceDays"]}
				}
			}
		}
	}

	current.Status = target
	current.DecidedBy = actor
	if in.NoteFa != "" {
		current.NoteFa = in.NoteFa
	}
	current.Events = append(current.Events, s.event(actor, action, detail))

	saved, err := s.repo.Save(ctx, current)
	if err != nil {
		return contracts.Contract{}, err
	}
	// ابلاغ قراردادی که جانشین قرارداد دیگری است، قرارداد پیشین را می‌بندد. بدون
	// این کار، پروندهٔ کارمند دو قرارداد جاری نشان می‌داد و بررسی هم‌پوشانی هم در
	// تصمیم‌های بعدی گیر می‌کرد.
	if in.Approve && current.SupersedesID != "" {
		if err := s.closeSuperseded(ctx, current, actor); err != nil {
			return contracts.Contract{}, err
		}
	}
	s.record(actor, action, saved, detail)
	return saved.ToContract(s.today()), nil
}

// closeSuperseded قرارداد پیشینِ یک تمدید/تبدیل را با ثبت رخداد می‌بندد.
func (s *Service) closeSuperseded(ctx context.Context, current domain.Contract, actor string) error {
	previous, ok, err := s.repo.Get(ctx, current.SupersedesID)
	if err != nil || !ok {
		return err
	}
	if previous.Status != domain.StatusActive && previous.Status != domain.StatusPending {
		return nil
	}
	target := domain.StatusExpired
	reason := "پایان مدت قرارداد با جانشینی "
	if previous.Kind == domain.KindPermanent {
		target = domain.StatusTerminated
		reason = "خاتمهٔ قرارداد دائم با جانشینی "
	}
	previous.Status = target
	previous.DecidedBy = actor
	previous.NoteFa = reason + current.ID
	previous.Events = append(previous.Events, s.event(actor, domain.ActionExpire, previous.NoteFa))
	if _, err := s.repo.Save(ctx, previous); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Append(actor, "contracts", "supersede", previous.ID, reason+current.ID)
	}
	return nil
}

// Terminate خاتمهٔ قرارداد جاری (پایان همکاری).
func (s *Service) Terminate(ctx context.Context, id string, in contracts.ContractDecision, actor string) (contracts.Contract, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Contract{}, err
	}
	if !ok {
		return contracts.Contract{}, infra.ErrNotFound
	}
	if !domain.CanTransition(current.Status, domain.StatusTerminated) {
		return contracts.Contract{}, &RuleError{Message: "تنها قرارداد جاری قابل خاتمه است."}
	}
	if in.NoteFa == "" {
		return contracts.Contract{}, &ValidationError{Details: map[string]string{"noteFa": "برای خاتمهٔ قرارداد ثبت دلیل الزامی است."}}
	}
	// تمدیدهای در انتظار تأیید برای قراردادی که خاتمه می‌یابد باید رد شوند تا صف تأیید
	// پروندهٔ بسته را نشان ندهد.
	pending, err := s.repo.ActiveForEmployee(ctx, current.EmployeeID)
	if err != nil {
		return contracts.Contract{}, err
	}
	for _, p := range pending {
		if p.Status != domain.StatusPending || p.SupersedesID != current.ID {
			continue
		}
		p.Status = domain.StatusRejected
		p.DecidedBy = actor
		p.NoteFa = "رد خودکار به دلیل خاتمهٔ قرارداد پایه"
		p.Events = append(p.Events, s.event(actor, domain.ActionReject, "رد خودکار پس از خاتمهٔ قرارداد پایه"))
		if _, err := s.repo.Save(ctx, p); err != nil {
			return contracts.Contract{}, err
		}
	}

	current.Status = domain.StatusTerminated
	current.DecidedBy = actor
	current.NoteFa = in.NoteFa
	current.Events = append(current.Events, s.event(actor, domain.ActionTerminate, in.NoteFa))

	saved, err := s.repo.Save(ctx, current)
	if err != nil {
		return contracts.Contract{}, err
	}
	s.record(actor, "terminate", saved, "خاتمهٔ قرارداد")
	return saved.ToContract(s.today()), nil
}

// Renew تمدید قرارداد مدت موقت: قراردادی تازه با شروع روز پس از پایان قرارداد فعلی.
func (s *Service) Renew(ctx context.Context, id string, in contracts.ContractRenewInput, actor string) (contracts.Contract, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Contract{}, err
	}
	if !ok {
		return contracts.Contract{}, infra.ErrNotFound
	}
	if errs := domain.ValidateRenew(current, in); len(errs) > 0 {
		return contracts.Contract{}, &ValidationError{Details: errs}
	}
	start, ok := domain.NextStart(current)
	if !ok {
		return contracts.Contract{}, &RuleError{Message: "تاریخ پایان قرارداد فعلی معتبر نیست؛ ابتدا آن را اصلاح کنید."}
	}
	end := in.EndDate
	if end == "" {
		end, ok = domain.DefaultRenewEnd(current)
		if !ok {
			return contracts.Contract{}, &RuleError{Message: "محاسبهٔ پایان دورهٔ تمدید ممکن نشد."}
		}
	}
	salary := current.MonthlySalary
	if in.MonthlySalary > 0 {
		salary = in.MonthlySalary
	}

	draft := current
	draft.ID, err = s.repo.NextID()
	if err != nil {
		return contracts.Contract{}, err
	}
	draft.Kind = domain.KindFixedTerm
	draft.StartDate = start
	draft.EndDate = end
	draft.MonthlySalary = salary
	draft.Status = domain.StatusPending
	draft.SupersedesID = current.ID
	draft.RenewCount = current.RenewCount + 1
	draft.DecidedBy = ""
	draft.CreatedBy = actor
	draft.Version = 0
	draft.NoteFa = in.NoteFa
	// اجزای مزد و اقلیمی از قرارداد پیشین ارث می‌رسد و مقدارهای داده‌شده آن را بازنویسی
	// می‌کنند؛ تمدید قرارداد نباید بی‌صدا فوق‌العاده‌های اقلیمی کارمند را صفر کند.
	draft.WageStructure.ApplyOverride(in.WageStructure)
	draft.Events = []domain.Event{s.event(actor, domain.ActionRenew, "تمدید قرارداد پیشین "+current.ID)}

	if err := s.ensureNoActiveOverlap(ctx, draft, ""); err != nil {
		return contracts.Contract{}, err
	}
	saved, err := s.repo.Save(ctx, draft)
	if err != nil {
		return contracts.Contract{}, err
	}
	s.record(actor, "renew", saved, "تمدید قرارداد "+current.ID)
	return saved.ToContract(s.today()), nil
}

// ConvertToPermanent تبدیل قرارداد مدت موقت به مدت دائم (نیازمند دست‌کم یک سال سابقه).
func (s *Service) ConvertToPermanent(ctx context.Context, id string, in contracts.ContractRenewInput, actor string) (contracts.Contract, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Contract{}, err
	}
	if !ok {
		return contracts.Contract{}, infra.ErrNotFound
	}
	if errs := domain.ValidateConversion(current, s.today()); len(errs) > 0 {
		msg := errs["kind"]
		if msg == "" {
			msg = errs["status"]
		}
		if msg == "" {
			msg = errs["serviceDays"]
		}
		return contracts.Contract{}, &RuleError{Message: msg}
	}

	salary := current.MonthlySalary
	if in.MonthlySalary > 0 {
		salary = in.MonthlySalary
	}
	draft := current
	draft.ID, err = s.repo.NextID()
	if err != nil {
		return contracts.Contract{}, err
	}
	draft.Kind = domain.KindPermanent
	draft.EndDate = ""
	draft.StartDate = s.today().String()
	draft.MonthlySalary = salary
	draft.Status = domain.StatusPending
	draft.SupersedesID = current.ID
	draft.RenewCount = 0
	draft.DecidedBy = ""
	draft.CreatedBy = actor
	draft.Version = 0
	draft.NoteFa = in.NoteFa
	draft.Events = []domain.Event{s.event(actor, domain.ActionConvert, "تبدیل قرارداد "+current.ID+" به مدت دائم")}

	saved, err := s.repo.Save(ctx, draft)
	if err != nil {
		return contracts.Contract{}, err
	}
	s.record(actor, "convert", saved, "تبدیل به قرارداد مدت دائم")
	return saved.ToContract(s.today()), nil
}

// Delete حذف قرارداد پیش‌نویس.
func (s *Service) Delete(ctx context.Context, id, actor string) error {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return infra.ErrNotFound
	}
	if current.Status != domain.StatusDraft {
		return &RuleError{Message: "تنها پیش‌نویس قرارداد قابل حذف است."}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Append(actor, "contracts", "delete", id, "حذف پیش‌نویس قرارداد "+current.EmployeeName)
	}
	return nil
}

// Expiring قراردادهای مدت موقت نزدیک به سرآمد (پایهٔ صفحهٔ تمدید).
func (s *Service) Expiring(ctx context.Context, days int) ([]contracts.Contract, error) {
	if days <= 0 {
		days = 60
	}
	page, err := s.repo.List(ctx, contracts.ContractFilter{
		ExpiringDays: days, Today: s.today().String(), Size: 200, Sort: "endDate:asc",
	})
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// ensureNoActiveOverlap جلوی دو قرارداد جاری هم‌زمان برای یک کارمند را می‌گیرد.
// فقط قراردادهای فعال و در انتظار تأیید بررسی می‌شوند؛ پیش‌نویس‌های دیگر مانع نیستند.
// BackfillPayClassification دستهٔ پرداخت (گروه جدول ۳ + روش پرداخت) را روی قراردادهایی
// که آن را ثبت نکرده‌اند می‌نویسد و شمار قراردادهای به‌روزشده را برمی‌گرداند.
//
// چرا لازم است: قراردادهای پیش از این تغییر ستون خالی دارند؛ نمایش درست است (سرویس هنگام
// خواندن استنتاج می‌کند) ولی برای بازبینی و گزارش‌گیری باید سند خودش دسته را داشته باشد.
// احکام گذشته عمداً دست‌نخورده می‌مانند: حکم سند تاریخی است و بازنویسی نمی‌شود.
func (s *Service) BackfillPayClassification(ctx context.Context, actor string) (int, error) {
	page, err := s.repo.List(ctx, contracts.ContractFilter{Scope: "all", Size: 1000, Sort: "id:asc"})
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, item := range page.Items {
		c, ok, err := s.repo.Get(ctx, item.ID)
		if err != nil {
			return updated, err
		}
		if !ok {
			continue
		}
		classification := domain.PayClassificationOf(c.EmploymentType, c.PayGroup, c.PayModel)
		if c.PayGroup == classification.PayGroup && c.PayModel == classification.PayModel {
			continue
		}
		c.PayGroup = classification.PayGroup
		c.PayModel = classification.PayModel
		c.Events = append(c.Events, s.event(actor, domain.ActionUpdate, "ثبت دستهٔ پرداخت روی سند"))
		if _, err := s.repo.Save(ctx, c); err != nil {
			return updated, err
		}
		updated++
	}
	if updated > 0 && s.audit != nil {
		s.audit.Append(actor, "contracts", "pay-classification-backfill", "",
			fmt.Sprintf("قراردادهای به‌روزشده=%d", updated))
	}
	return updated, nil
}

func (s *Service) ensureNoActiveOverlap(ctx context.Context, candidate domain.Contract, ignoreID string) error {
	list, err := s.repo.ActiveForEmployee(ctx, candidate.EmployeeID)
	if err != nil {
		return err
	}
	for _, other := range list {
		if other.ID == ignoreID || other.ID == candidate.ID || other.ID == candidate.SupersedesID {
			continue
		}
		if other.Status != domain.StatusActive && other.Status != domain.StatusPending && other.Status != domain.StatusDraft {
			continue
		}
		if domain.Overlaps(candidate, other) {
			return &RuleError{Message: fmt.Sprintf(
				"برای این کارمند قرارداد «%s» با شمارهٔ %s در بازهٔ هم‌پوشان وجود دارد؛ ابتدا آن را تعیین تکلیف کنید.",
				domain.StatusFa[other.Status], other.ID)}
		}
	}
	return nil
}

func (s *Service) transition(ctx context.Context, id, actor, target, action, detail string) (contracts.Contract, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Contract{}, err
	}
	if !ok {
		return contracts.Contract{}, infra.ErrNotFound
	}
	if !domain.CanTransition(current.Status, target) {
		return contracts.Contract{}, &RuleError{Message: "این تغییر وضعیت از وضعیت فعلی مجاز نیست."}
	}
	current.Status = target
	current.Events = append(current.Events, s.event(actor, action, detail))
	saved, err := s.repo.Save(ctx, current)
	if err != nil {
		return contracts.Contract{}, err
	}
	s.record(actor, action, saved, detail)
	return saved.ToContract(s.today()), nil
}

func (s *Service) event(actor, action, note string) domain.Event {
	return domain.Event{At: s.clock(), Actor: actor, Action: action, NoteFa: note}
}

func (s *Service) record(actor, action string, c domain.Contract, detail string) {
	if s.audit != nil {
		s.audit.Append(actor, "contracts", action, c.ID, detail+" — "+c.EmployeeName)
	}
	if s.sync != nil {
		s.sync.Record("Contract", synclog.OpUpsert, c.ID, c.Version, map[string]any{
			"id":            c.ID,
			"employeeId":    c.EmployeeID,
			"employeeName":  c.EmployeeName,
			"kind":          c.Kind,
			"startDate":     c.StartDate,
			"endDate":       c.EndDate,
			"monthlySalary": c.MonthlySalary,
			"status":        c.Status,
			"version":       c.Version,
		})
	}
}

// employmentTypeOf نوع استخدام نهایی قرارداد را برمی‌گرداند: ورودی فرم، سپس نوع استخدام
// کارمند، و در نهایت استنتاج از نوع قرارداد. این‌گونه قراردادهای قدیمی هم نوع معتبر دارند.
func employmentTypeOf(input, kind, employeeType string) string {
	if t, ok := contracts.EmploymentTypeByKey(input); ok {
		return t.Key
	}
	if t, ok := contracts.EmploymentTypeByKey(employeeType); ok {
		return t.Key
	}
	if kind == domain.KindPermanent {
		return "official_oil"
	}
	return "fixed_1662"
}

func withinScope(c domain.Contract, f contracts.ContractFilter) bool {
	switch f.Scope {
	case "own":
		return f.ActorEmployeeID != "" && c.EmployeeID == f.ActorEmployeeID
	case "own_unit":
		return f.ActorDepartment == "" || c.DepartmentName == f.ActorDepartment
	default:
		return true
	}
}
