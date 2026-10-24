// سرویس CRUD و گردش‌کار رکوردکیت با اعتبارسنجی شمای سند و ثبت تاریخچه/حسابرسی.

package recordkit

import (
	"context"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/synclog"
)

// EmployeeLookup پورت خواندن کارمند؛ ماژول‌های مصرف‌کننده آن را از پورت مشترک می‌گیرند.
type EmployeeLookup interface {
	ByID(ctx context.Context, employeeID string) (contracts.Employee, bool, error)
}

// LookupFromReader پوشش روی contracts.EmployeeReader.
func LookupFromReader(reader contracts.EmployeeReader) EmployeeLookup {
	return employeeAdapter{reader: reader}
}

type employeeAdapter struct{ reader contracts.EmployeeReader }

func (a employeeAdapter) ByID(ctx context.Context, employeeID string) (contracts.Employee, bool, error) {
	if a.reader == nil {
		return contracts.Employee{}, false, nil
	}
	return a.reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
}

// Service موتور گردش‌کار رکوردها برای یک ماژول.
type Service struct {
	desc      Descriptor
	repo      Repository
	employees EmployeeLookup
	audit     *audit.Log
	sync      *synclog.Log
	clock     func() time.Time
}

// New سرویس می‌سازد.
func New(desc Descriptor, repo Repository, employees EmployeeLookup, auditLog *audit.Log, syncLog *synclog.Log, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{desc: desc, repo: repo, employees: employees, audit: auditLog, sync: syncLog, clock: clock}
}

// Schema شناسهٔ ماژول برای ساخت خودکار رابط.
func (s *Service) Schema() contracts.RecordSchema { return s.desc.Schema() }

// Module شناسهٔ ماژول.
func (s *Service) Module() string { return s.desc.Module }

// List فهرست رکوردها با محدودهٔ دسترسی نقش.
func (s *Service) List(ctx context.Context, f contracts.RecordFilter) (contracts.RecordPage, error) {
	page, err := s.repo.List(ctx, f)
	if err != nil {
		return contracts.RecordPage{}, err
	}
	items := make([]contracts.Record, 0, len(page.Items))
	for _, r := range page.Items {
		items = append(items, r.ToDTO(s.kindOf(r.Kind)))
	}
	return contracts.RecordPage{Items: items, Total: page.Total, Page: page.Page, Size: page.Size}, nil
}

// Get خواندن یک رکورد با بررسی محدودهٔ دسترسی.
func (s *Service) Get(ctx context.Context, f contracts.RecordFilter, id string) (contracts.Record, bool, error) {
	r, ok, err := s.repo.Get(ctx, id)
	if err != nil || !ok {
		return contracts.Record{}, false, err
	}
	if !matchScope(r, f) {
		return contracts.Record{}, false, nil
	}
	return r.ToDTO(s.kindOf(r.Kind)), true, nil
}

// Create ایجاد رکورد تازه در وضعیت آغازین نوع خودش.
func (s *Service) Create(ctx context.Context, in contracts.RecordInput, actor string) (contracts.Record, error) {
	kind, ok := s.desc.KindByKey(strings.TrimSpace(in.Kind))
	if !ok {
		return contracts.Record{}, &ValidationError{Details: map[string]string{"kind": "نوع رکورد نامعتبر است."}}
	}
	if errs := kind.ValidateInput(in, false); len(errs) > 0 {
		return contracts.Record{}, &ValidationError{Details: errs}
	}
	if kind.Validate != nil {
		if err := kind.Validate(in); err != nil {
			return contracts.Record{}, err
		}
	}

	rec := Record{
		Module:    s.desc.Module,
		Kind:      kind.Key,
		Title:     strings.TrimSpace(in.Title),
		Status:    kind.Initial,
		Fields:    cleanFields(in.Fields),
		NoteFa:    strings.TrimSpace(in.NoteFa),
		UpdatedAt: s.clock(),
	}
	if kind.EmployeeScoped {
		employee, ok, err := s.employee(ctx, in.EmployeeID)
		if err != nil {
			return contracts.Record{}, err
		}
		if !ok {
			return contracts.Record{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
		}
		rec.EmployeeID = employee.ID
		rec.EmployeeName = employee.FullName
		rec.DepartmentName = employee.DepartmentName
	}
	rec.CreatedBy = actor
	rec.Events = append(rec.Events, Event{At: s.clock(), Actor: actor, Action: ActionCreate, NoteFa: "ثبت رکورد"})

	// شمارنده به تفکیک نوع رکورد جلو می‌رود و یکتایی شناسه در کل سامانه به یکتایی
	// «پیشوند شناسه» تکیه دارد، نه به کلید شمارنده: جدول رکوردها مشترک است و کلید
	// اصلی‌اش خودِ شناسه. دروازهٔ TestRecordKindIDPrefixesAreUniqueAcrossModules همین
	// یکتایی را در CI تضمین می‌کند. عمداً کلید شمارنده به «ماژول|نوع» تغییر نکرد: تغییر
	// کلید، شمارندهٔ قدیمی را رها می‌کند و شناسهٔ تازه از ۱ شروع می‌شود؛ روی پایگاه‌دادهٔ
	// موجود، همان شناسهٔ تازه روی سطر موجود می‌نویسد و رکورد سازمان را بازنویسی می‌کند.
	id, err := s.repo.NextID(kind.Key, kind.IDPrefix)
	if err != nil {
		return contracts.Record{}, err
	}
	rec.ID = id

	saved, err := s.repo.Save(ctx, rec)
	if err != nil {
		return contracts.Record{}, err
	}
	s.record(actor, ActionCreate, kind, saved, "ثبت "+kind.TitleFa)
	return saved.ToDTO(kind), nil
}

// Update ویرایش رکورد در وضعیت‌های مجازِ اعلام‌شده.
func (s *Service) Update(ctx context.Context, id string, in contracts.RecordInput, actor string) (contracts.Record, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Record{}, err
	}
	if !ok {
		return contracts.Record{}, ErrNotFound
	}
	kind := s.kindOf(current.Kind)
	if !kind.Editable(current.Status) {
		return contracts.Record{}, &RuleError{Message: "در این وضعیت ویرایش مجاز نیست؛ از اقدام گردش‌کار استفاده کنید."}
	}
	if errs := kind.ValidateInput(in, false); len(errs) > 0 {
		return contracts.Record{}, &ValidationError{Details: errs}
	}
	if kind.Validate != nil {
		if err := kind.Validate(in); err != nil {
			return contracts.Record{}, err
		}
	}
	current.Title = strings.TrimSpace(in.Title)
	current.Fields = cleanFields(in.Fields)
	current.NoteFa = strings.TrimSpace(in.NoteFa)
	current.UpdatedAt = s.clock()
	if kind.EmployeeScoped && strings.TrimSpace(in.EmployeeID) != "" {
		employee, ok, err := s.employee(ctx, in.EmployeeID)
		if err != nil {
			return contracts.Record{}, err
		}
		if !ok {
			return contracts.Record{}, &ValidationError{Details: map[string]string{"employeeId": "کارمند انتخابی یافت نشد."}}
		}
		current.EmployeeID = employee.ID
		current.EmployeeName = employee.FullName
		current.DepartmentName = employee.DepartmentName
	}
	current.Events = append(current.Events, Event{At: s.clock(), Actor: actor, Action: ActionUpdate, NoteFa: "ویرایش رکورد"})

	saved, err := s.repo.Save(ctx, current)
	if err != nil {
		return contracts.Record{}, err
	}
	s.record(actor, ActionUpdate, kind, saved, "ویرایش "+kind.TitleFa)
	return saved.ToDTO(kind), nil
}

// Act انجام یک اقدام گردش‌کار (تأیید، رد، اجرا، بستن…).
func (s *Service) Act(ctx context.Context, id string, in contracts.RecordAction, actor string) (contracts.Record, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.Record{}, err
	}
	if !ok {
		return contracts.Record{}, ErrNotFound
	}
	kind := s.kindOf(current.Kind)
	action, ok := kind.ActionByValue(strings.TrimSpace(in.Action))
	if !ok {
		return contracts.Record{}, &ValidationError{Details: map[string]string{"action": "اقدام درخواستی برای این نوع رکورد تعریف نشده است."}}
	}
	note := strings.TrimSpace(in.NoteFa)
	if action.NoteRequired && note == "" {
		return contracts.Record{}, &ValidationError{Details: map[string]string{"noteFa": "برای این اقدام ثبت دلیل الزامی است."}}
	}
	if action.To != "" && !kind.CanTransition(current.Status, action) {
		return contracts.Record{}, &RuleError{Message: "این اقدام از وضعیت فعلی رکورد مجاز نیست."}
	}
	// تفکیک وظایف: مرحلهٔ گردش‌کار تعیین می‌کند چه نقشی در این وضعیت مجاز به تصمیم است؛
	// این قاعده در دامنهٔ همان نوع رکورد اعلام شده و این‌جا فقط اجرا می‌شود.
	workflowActor := actorFrom(ctx, actor)
	if !roleAllowed(action, workflowActor) {
		return contracts.Record{}, &RuleError{Message: "این اقدام در مرحلهٔ کنونی برای نقش شما مجاز نیست."}
	}
	if !scopeAllows(action.Scope, workflowActor, current.EmployeeID, current.DepartmentName) {
		return contracts.Record{}, &RuleError{Message: "محدودهٔ سازمانی شما برای این اقدام کافی نیست."}
	}
	if action.To != "" {
		current.Status = action.To
		current.DecidedBy = actor
	}
	if note != "" {
		current.NoteFa = note
	}
	if len(in.Fields) > 0 {
		if current.Fields == nil {
			current.Fields = map[string]string{}
		}
		for k, v := range in.Fields {
			current.Fields[k] = strings.TrimSpace(v)
		}
	}
	current.Events = append(current.Events, Event{At: s.clock(), Actor: actor, Action: action.Value, NoteFa: note})
	current.UpdatedAt = s.clock()

	saved, err := s.repo.Save(ctx, current)
	if err != nil {
		return contracts.Record{}, err
	}
	s.record(actor, action.Value, kind, saved, action.LabelFa)
	return saved.ToDTO(kind), nil
}

// SeedIfEmpty اگر هیچ رکوردی در انبار نباشد، رکوردهای نمونه را می‌سازد.
//
// این کار عمدی است: صفحه‌ای که روی دادهٔ واقعی کار می‌کند ولی هنوز دادهٔ سازمانی
// واردشده ندارد، باید حالت اولیهٔ قابل‌فهم نشان دهد؛ رکوردهای نمونه فقط یک بار و
// تنها در اجرای درون‌حافظه‌ای (نمونهٔ نمایشی) ساخته می‌شوند، نه در استقرار واقعی.
func (s *Service) SeedIfEmpty(ctx context.Context, actor string, inputs []contracts.RecordInput) error {
	page, err := s.repo.List(ctx, contracts.RecordFilter{Module: s.desc.Module, Size: 1})
	if err != nil {
		return err
	}
	if page.Total > 0 {
		return nil
	}
	for _, in := range inputs {
		created, err := s.Create(ctx, in, actor)
		if err != nil {
			return err
		}
		if target := strings.TrimSpace(in.SeedStatus); target != "" && target != created.Status {
			if err := s.seedAdvance(ctx, created.ID, target, actor); err != nil {
				return err
			}
		}
	}
	return nil
}

// SeedMissing رکوردهای نمونه‌ای را که هنوز در انبار نیستند می‌سازد و تعداد ساخته‌شده
// را برمی‌گرداند.
//
// تفاوتش با SeedIfEmpty: آن یکی فقط وقتی کار می‌کند که انبار کامل خالی باشد، ولی این
// یکی «افزایشی» است — هر رکورد نمونه را با کلید نوع+عنوان می‌سنجد و تنها اگر نبود
// می‌سازد. کاربردش نصب تازه روی پایگاه‌داده‌ای است که بخشی از داده را دارد (مثلاً
// احکام ریخته ولی فوق‌العاده‌های تازه‌اضافه‌شدهٔ نسخهٔ نو را ندارد). هیچ‌گاه رکورد
// موجود را بازنویسی یا پاک نمی‌کند.
func (s *Service) SeedMissing(ctx context.Context, actor string, inputs []contracts.RecordInput) (int, error) {
	existing := map[string]bool{}
	// فهرست صفحه‌به‌صفحه خوانده می‌شود؛ سقف هر صفحه ۵۰۰ رکورد است و یک صفحه کافی نیست.
	for page := 1; page <= 200; page++ {
		res, err := s.repo.List(ctx, contracts.RecordFilter{Module: s.desc.Module, Page: page, Size: 500})
		if err != nil {
			return 0, err
		}
		for _, r := range res.Items {
			existing[r.Kind+"|"+r.Title] = true
		}
		if len(res.Items) < 500 {
			break
		}
	}

	created := 0
	for _, in := range inputs {
		if existing[in.Kind+"|"+in.Title] {
			continue
		}
		rec, err := s.Create(ctx, in, actor)
		if err != nil {
			return created, err
		}
		if target := strings.TrimSpace(in.SeedStatus); target != "" && target != rec.Status {
			if err := s.seedAdvance(ctx, rec.ID, target, actor); err != nil {
				return created, err
			}
		}
		created++
	}
	return created, nil
}

// seedAdvance یک رکورد نمونه را تا وضعیت خواسته‌شده می‌برد.
//
// دادهٔ نمونه باید وضعیت‌های واقعی گردش‌کار را نشان دهد (مثلاً وام تأییدشده و وام در
// بازپرداخت)، وگرنه همهٔ رکوردها در وضعیت آغازین می‌مانند و گزارش‌های گردش‌کار خالی
// به نظر می‌رسند. این تابع فقط از «گراف انتقال‌های اعلامی» همان نوع رکورد می‌گذرد تا
// داده با گردش‌کار ناسازگار نشود، ولی کنترل نقش مرحله را اعمال نمی‌کند — چون این داده
// را سازمان وارد نکرده و سیستمی ساخته می‌شود، نه کاربر.
func (s *Service) seedAdvance(ctx context.Context, id, target, actor string) error {
	rec, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	kind := s.kindOf(rec.Kind)
	path, ok := transitionPath(kind, rec.Status, target)
	if !ok {
		return &RuleError{Message: "وضعیت نمونهٔ «" + target + "» از وضعیت " + rec.Status + " در گردش‌کار «" + kind.TitleFa + "» قابل دسترسی نیست."}
	}
	for _, action := range path {
		current, ok, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		current.Status = action.To
		current.Events = append(current.Events, Event{
			At: s.clock(), Actor: actor, Action: action.Value,
			NoteFa: "دادهٔ نمونه — " + action.LabelFa,
		})
		current.UpdatedAt = s.clock()
		if _, err := s.repo.Save(ctx, current); err != nil {
			return err
		}
	}
	return nil
}

// transitionPath کوتاه‌ترین زنجیرهٔ اقدام‌های اعلامی از وضعیت مبدأ به وضعیت مقصد را
// می‌یابد (پیمایش سطح‌به‌سطح) و false برمی‌گرداند اگر مقصد دست‌نیافتنی باشد.
func transitionPath(kind Kind, from, to string) ([]Action, bool) {
	if from == to {
		return nil, true
	}
	known := map[string]bool{}
	for _, st := range kind.Statuses {
		known[st.Value] = true
	}
	if !known[to] {
		return nil, false
	}
	type node struct {
		status string
		path   []Action
	}
	queue := []node{{status: from}}
	seen := map[string]bool{from: true}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, a := range kind.Actions {
			if a.To == "" || !kind.CanTransition(cur.status, a) || seen[a.To] {
				continue
			}
			next := append(append([]Action{}, cur.path...), a)
			if a.To == to {
				return next, true
			}
			seen[a.To] = true
			queue = append(queue, node{status: a.To, path: next})
		}
	}
	return nil, false
}

// Delete حذف رکورد در وضعیت آغازین.
func (s *Service) Delete(ctx context.Context, id, actor string) error {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	kind := s.kindOf(current.Kind)
	if !kind.Editable(current.Status) {
		return &RuleError{Message: "تنها رکورد در وضعیت قابل‌ویرایش حذف می‌شود."}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Append(actor, s.desc.Module, ActionDelete, id, "حذف "+kind.TitleFa+" — "+current.Title)
	}
	return nil
}

// TasksFor کارهای قابل انجام این کنشگر در ماژول را برمی‌گرداند (کارتابل).
//
// برای هر رکورد، اقدام‌هایی که نقش کنشگر در وضعیت جاری‌اش مجاز است پیدا و به یک «کار»
// تبدیل می‌شوند؛ پس کاربر در کارتابل دقیقاً می‌بیند چه اقدامی از او انتظار می‌رود. قاعده
// مجاز بودن در دامنه اعلام شده و این‌جا بازنویسی نمی‌شود.
func (s *Service) TasksFor(ctx context.Context, actor contracts.WorkflowActor) []contracts.WorkflowTask {
	page, err := s.repo.List(ctx, contracts.RecordFilter{Module: s.desc.Module, Size: 500})
	if err != nil {
		return nil
	}
	workActor := Actor{Username: actor.Username, Role: actor.Role, DepartmentID: actor.DepartmentID, EmployeeID: actor.EmployeeID}
	out := []contracts.WorkflowTask{}
	for _, r := range page.Items {
		kind := s.kindOf(r.Kind)
		for _, action := range kind.AvailableActions(r.Status, workActor, r.EmployeeID, r.DepartmentName) {
			out = append(out, contracts.WorkflowTask{
				Module:         s.desc.Module,
				ModuleTitleFa:  s.desc.TitleFa,
				Kind:           r.Kind,
				KindTitleFa:    kind.TitleFa,
				RecordID:       r.ID,
				Title:          r.Title,
				Status:         r.Status,
				StatusFa:       kind.StatusFa(r.Status),
				Action:         action.Value,
				ActionLabelFa:  action.LabelFa,
				ActionStyle:    action.Style,
				NoteRequired:   action.NoteRequired,
				EmployeeID:     r.EmployeeID,
				EmployeeName:   r.EmployeeName,
				DepartmentName: r.DepartmentName,
				CreatedBy:      r.CreatedBy,
				UpdatedAt:      r.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
	}
	return out
}

// ModuleTitleFa عنوان فارسی ماژول برای منابع سابقه (هیچ متنی در UI ساخته نمی‌شود).
func (s *Service) ModuleTitleFa() string { return s.desc.TitleFa }

// historyPageSize سقف خواندن رکوردهای یک نوع در ساخت سابقه؛ انبار خودش هر صفحه را به
// ۵۰۰ رکورد محدود می‌کند و سابقهٔ یک کارمند در یک نوع هرگز از این عدد نمی‌گذرد.
const historyPageSize = 500

// HistoryOf سابقهٔ یک کارمند در این ماژول را می‌سازد (سطر «سند» + سابقهٔ تغییرات خودش).
//
// هر رکورد از هر نوعی که به این کارمند مربوط است یک سطر سابقه می‌شود — مستقل از وضعیتش؛
// پس احکام منسوخ، قراردادهای پایان‌یافته، مدارک و سمت‌های پیشین هم دیده می‌شوند و همین
// «سابقهٔ قبلی» است که کاربر می‌خواهد. تاریخ مؤثر از فیلد اعلامی/تاریخی خود نوع رکورد
// می‌آید و اگر نوع هیچ فیلد تاریخی نداشت، زمان ثبت/ویرایش مبنای چیدن زمانی می‌شود.
func (s *Service) HistoryOf(ctx context.Context, employeeID string) ([]contracts.HistoryEntry, error) {
	id := strings.TrimSpace(employeeID)
	if id == "" {
		return nil, nil
	}
	out := []contracts.HistoryEntry{}
	for _, kind := range s.desc.Kinds {
		page, err := s.repo.List(ctx, contracts.RecordFilter{
			Module: s.desc.Module, Kind: kind.Key, EmployeeID: id, Scope: "all", Size: historyPageSize,
		})
		if err != nil {
			return nil, err
		}
		for _, rec := range page.Items {
			// فیلتر کارمند در انبار یک قید است، ولی اینجا دوباره سنجیده می‌شود تا سابقهٔ
			// کارمند دیگر هرگز داخل پروندهٔ این کارمند نیاید.
			if rec.EmployeeID != id {
				continue
			}
			out = append(out, s.historyEntry(kind, rec))
		}
	}
	return out, nil
}

// historyEntry تبدیل یک رکورد به سطر سابقه با اقلام اعلامی و سابقهٔ تغییراتش.
func (s *Service) historyEntry(kind Kind, rec Record) contracts.HistoryEntry {
	dateKey, dateLabel := kind.HistoryDateField()
	date := ""
	if dateKey != "" {
		date = strings.TrimSpace(rec.Fields[dateKey])
	}
	if date == "" {
		date = calendar.Format(rec.UpdatedAt)
		dateLabel = "تاریخ ثبت/ویرایش"
	}

	fields := make([]contracts.HistoryField, 0, len(kind.Fields))
	for _, f := range kind.Fields {
		if f.Key == dateKey {
			continue // تاریخ مؤثر جداگانه در Date می‌آید تا دوبار تکرار نشود
		}
		v := strings.TrimSpace(rec.Fields[f.Key])
		if v == "" {
			continue
		}
		fields = append(fields, contracts.HistoryField{Key: f.Key, LabelFa: f.LabelFa, Value: v, Type: f.Type})
	}

	changes := make([]contracts.HistoryChange, 0, len(rec.Events))
	for _, e := range rec.Events {
		changes = append(changes, contracts.HistoryChange{
			At:       calendar.Format(e.At),
			Actor:    e.Actor,
			Action:   e.Action,
			ActionFa: actionFa(kind, e.Action),
			NoteFa:   e.NoteFa,
		})
	}

	return contracts.HistoryEntry{
		Module:         s.desc.Module,
		ModuleTitleFa:  s.desc.TitleFa,
		Kind:           kind.Key,
		KindTitleFa:    kind.TitleFa,
		RecordID:       rec.ID,
		Title:          rec.Title,
		EmployeeID:     rec.EmployeeID,
		EmployeeName:   rec.EmployeeName,
		DepartmentName: rec.DepartmentName,
		Date:           date,
		DateLabelFa:    dateLabel,
		Status:         rec.Status,
		StatusFa:       kind.StatusFa(rec.Status),
		Fields:         fields,
		Changes:        changes,
		NoteFa:         rec.NoteFa,
	}
}

/* -------------------------------- کمکی‌ها -------------------------------- */

// actorFrom کنشگر گردش‌کار را از زمینهٔ درخواست می‌سازد؛ اگر هویتی نباشد، نقش خالی
// می‌ماند و موتور او را محدود نمی‌کند (فراخوانی داخلی و بذرکاری).
func actorFrom(ctx context.Context, username string) Actor {
	if user, ok := auth.UserFrom(ctx); ok {
		return Actor{Username: username, Role: user.Role, DepartmentID: user.DepartmentID, EmployeeID: user.EmployeeID}
	}
	return Actor{Username: username}
}

// ActionCreate/ActionUpdate/ActionDelete رخدادهای مشترک همهٔ ماژول‌ها.
const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

func (s *Service) kindOf(key string) Kind {
	if k, ok := s.desc.KindByKey(key); ok {
		return k
	}
	return Kind{Key: key}
}

func (s *Service) employee(ctx context.Context, id string) (contracts.Employee, bool, error) {
	if s.employees == nil {
		return contracts.Employee{}, false, nil
	}
	return s.employees.ByID(ctx, id)
}

func (s *Service) record(actor, action string, kind Kind, r Record, detail string) {
	if s.audit != nil {
		s.audit.Append(actor, s.desc.Module, action, r.ID, detail+" — "+r.Title)
	}
	if s.sync != nil {
		payload := map[string]any{
			"id":      r.ID,
			"kind":    r.Kind,
			"title":   r.Title,
			"status":  r.Status,
			"version": r.Version,
			"module":  s.desc.Module,
		}
		if r.EmployeeID != "" {
			payload["employeeId"] = r.EmployeeID
		}
		s.sync.Record(kind.TitleFa, synclog.OpUpsert, r.ID, r.Version, payload)
	}
}

func cleanFields(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if strings.TrimSpace(v) != "" {
			out[k] = strings.TrimSpace(v)
		}
	}
	return out
}
