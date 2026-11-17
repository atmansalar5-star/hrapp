// Package app لایهٔ کاربرد ماژول کارکنان: گردش کار، اعتبارسنجی، آدیت و سینک.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/employee/domain"
	"hrms/internal/modules/employee/infra"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/photos"
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
	repo     Repository
	audit    *audit.Log
	sync     *synclog.Log
	photos   *photos.Store
	photoKey []byte
}

// New سرویس می‌سازد؛ آدیت و سینک اختیاری‌اند تا آزمون‌ها ساده بمانند.
//
// photoStore انبار «عکس کارگزینی» است؛ اگر nil باشد (اجرای آزمون بدون پوشهٔ داده)
// مسیرهای عکس با خطای روشن پاسخ می‌دهند و بقیهٔ ماژول دست‌نخورده کار می‌کند.
// photoKey کلید امضای نشانی عکس است تا نشانی یک کارمند برای کارمند دیگر کار نکند.
func New(repo Repository, auditLog *audit.Log, syncLog *synclog.Log) *Service {
	return &Service{repo: repo, audit: auditLog, sync: syncLog}
}

// WithPhotos انبار عکس و کلید امضا را به سرویس می‌دهد (زنجیره‌ای، تا امضای New عوض نشود
// و آزمون‌های موجود بی‌تغییر بمانند).
func (s *Service) WithPhotos(store *photos.Store, key []byte) *Service {
	s.photos = store
	s.photoKey = key
	return s
}

// now زمان جاری (تاریخ بارگذاری عکس به تقویم شمسی از همین گرفته می‌شود).
func (s *Service) now() time.Time { return time.Now() }

// List فهرست کارکنان با اعمال محدودهٔ دسترسی.
//
// عکس کارگزینی از انبار فایل خوانده و به هر سطر چسبانده می‌شود؛ اگر انبار عکس در
// دسترس نباشد، فهرست بدون عکس برمی‌گردد و کار ماژول نمی‌ایستد.
func (s *Service) List(ctx context.Context, f contracts.EmployeeFilter) (contracts.EmployeePage, error) {
	page, err := s.repo.List(ctx, f)
	if err != nil {
		return page, err
	}
	index, err := s.photoIndex()
	if err != nil {
		return page, nil
	}
	for i := range page.Items {
		if photo, ok := index[page.Items[i].ID]; ok {
			s.attachPhoto(&page.Items[i], photo)
		}
	}
	return page, nil
}

// Get پروندهٔ کارمند با اعمال محدودهٔ دسترسی؛ در صورت خارج بودن از محدوده، یافتنشده برمی‌گردد.
func (s *Service) Get(ctx context.Context, f contracts.EmployeeFilter, id string) (contracts.Employee, bool, error) {
	e, ok, err := s.repo.Get(ctx, id)
	if err != nil || !ok {
		return contracts.Employee{}, false, err
	}
	if !withinScope(f, e) {
		// عمداً «یافت نشد» برمی‌گردانیم تا وجود کارمند در واحدهای دیگر لو نرود.
		return contracts.Employee{}, false, nil
	}
	dto := e.ToContract()
	if index, err := s.photoIndex(); err == nil {
		if photo, ok := index[dto.ID]; ok {
			s.attachPhoto(&dto, photo)
		}
	}
	return dto, true, nil
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
	dto := saved.ToContract()
	if index, err := s.photoIndex(); err == nil {
		if photo, ok := index[dto.ID]; ok {
			s.attachPhoto(&dto, photo)
		}
	}
	return dto, nil
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
	dto := saved.ToContract()
	if index, err := s.photoIndex(); err == nil {
		if photo, ok := index[dto.ID]; ok {
			s.attachPhoto(&dto, photo)
		}
	}
	return dto, nil
}

// Assign تغییر سازمانی کارمند را در پی اجرای یک ابلاغ انتصاب یا ترفیع اعمال می‌کند.
//
// چرا این متد و نه استفادهٔ مستقیم از Update: اجرای ابلاغ بخشی از یک گردش‌کار تصویب‌شده
// است و باید فقط پست و واحد را عوض کند، نه بقیهٔ اطلاعات پرونده را؛ ورودی Update همهٔ
// فیلدهای اجباری (نام، کد ملی، واحد، پست، تاریخ استخدام) را می‌خواهد و با یک ابلاغ
// ناقص رد می‌شود. اینجا فقط همان دو فیلد سازمانی تغییر می‌کند و بقیه دست‌نخورده می‌مانند.
//
// قاعده‌ها:
//   - نام واحد به شناسه تبدیل می‌شود؛ واحد ناشناخته تغییر نمی‌کند و DepartmentMatched=false
//     برمی‌گردد (اجرای ابلاغ نباید به‌خاطر یک نام واحد نانوشته شکست بخورد).
//   - اگر پست و واحد هر دو همان مقدار قبلی باشند، نوشتنی رخ نمی‌دهد (اجرای تکراری، پرونده
//     را بی‌دلیل نسخه‌تازه نمی‌کند) و NoteFa همین را می‌گوید.
//   - تغییر در دفتر حسابرسی و لاگ سینک ثبت می‌شود تا پروندهٔ کارمند سابقه داشته باشد.
func (s *Service) Assign(ctx context.Context, id string, in contracts.EmployeeAssignment, actor string) (contracts.EmployeeAssignmentResult, error) {
	current, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.EmployeeAssignmentResult{}, err
	}
	if !ok {
		return contracts.EmployeeAssignmentResult{}, infra.ErrNotFound
	}

	updated := current
	// عکس وضع پیشین همراه نتیجه برمی‌گردد تا ماژول انتصابات بتواند در «ابطال ابلاغ»
	// پرونده را به همین مقدارها برگرداند؛ مالک پرونده تنها کسی است که وضع پیشین را می‌داند.
	result := contracts.EmployeeAssignmentResult{Before: current.ToContract()}
	if position := strings.TrimSpace(in.Position); position != "" && position != current.Position {
		updated.Position = position
		result.PositionChanged = true
	}
	// نشانهٔ «-» یعنی «گروه شغلی را خالی کن» (همان ثابت دامنه؛ ابطال ابلاغی که
	// گروهی به پرونده اضافه کرده بود)؛ فیلد خالی یعنی «تغییر نده».
	if raw := strings.TrimSpace(in.JobGrade); raw == domain.ClearJobGrade {
		if current.JobGrade != "" {
			updated.JobGrade = ""
			result.GradeChanged = true
		}
	} else if grade := resolveGrade(raw); grade != "" && grade != current.JobGrade {
		updated.JobGrade = grade
		result.GradeChanged = true
	}
	if name := strings.TrimSpace(in.DepartmentName); name != "" {
		if dept, found := infra.DepartmentByName(name); found {
			result.DepartmentMatched = true
			if dept.ID != current.DepartmentID {
				updated.DepartmentID = dept.ID
				updated.DepartmentName = dept.Name
				result.DepartmentChanged = true
			}
		}
	}

	if !result.PositionChanged && !result.DepartmentChanged && !result.GradeChanged {
		result.Employee = current.ToContract()
		result.NoteFa = "پروندهٔ کارمند با ابلاغ هم‌خوان است؛ پست، واحد و گروه شغلی تغییری لازم نداشت."
		return result, nil
	}

	saved, err := s.repo.Save(ctx, updated)
	if err != nil {
		return contracts.EmployeeAssignmentResult{}, err
	}
	s.recordAssign(actor, saved, in, result)
	dto := saved.ToContract()
	if index, err := s.photoIndex(); err == nil {
		if photo, ok := index[dto.ID]; ok {
			s.attachPhoto(&dto, photo)
		}
	}
	result.Employee = dto
	result.NoteFa = assignmentNoteFa(result)
	return result, nil
}

// resolveGrade کلید گروه شغلی را می‌سنجد؛ گروه ناشناخته تغییر نمی‌کند (قاعدهٔ دامنه).
func resolveGrade(key string) string {
	return domain.ResolveJobGrade(key)
}

// recordAssign تغییر سازمانی را در دفتر حسابرسی و لاگ سینک ثبت می‌کند.
func (s *Service) recordAssign(actor string, e domain.Employee, in contracts.EmployeeAssignment, result contracts.EmployeeAssignmentResult) {
	detail := "اجرای ابلاغ " + strings.TrimSpace(in.DecreeNo) + " — پست تازه: " + e.Position
	if reason := strings.TrimSpace(in.ReasonFa); reason != "" {
		detail += " — ملاک: " + reason
	}
	if s.audit != nil {
		s.audit.Append(actor, "employee", "assign", e.ID, detail)
	}
	s.record(actor, "assign", e)
}

// assignmentNoteFa متن گزارش اجرای ابلاغ روی پرونده.
func assignmentNoteFa(result contracts.EmployeeAssignmentResult) string {
	parts := []string{}
	if result.PositionChanged {
		parts = append(parts, "پست سازمانی پرونده به‌روزرسانی شد")
	}
	if result.DepartmentChanged {
		parts = append(parts, "واحد سازمانی پرونده به‌روزرسانی شد")
	}
	if result.GradeChanged {
		parts = append(parts, "گروه شغلی پرونده به‌روزرسانی شد")
	}
	if len(parts) == 0 {
		return "پروندهٔ کارمند با ابلاغ هم‌خوان است."
	}
	return strings.Join(parts, " و ") + "."
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

// Departments واحدهای سازمانی برای فیلتر، نمودار درختی و ساختار سازمانی ماژول‌های دیگر.
//
// امضای این متد همان چیزی است که پورت `contracts.EmployeeReader` وعده می‌دهد (زمینهٔ
// درخواست + فهرست) تا مصرف‌کننده‌ها بدون پوستهٔ اضافه از همان پورت واحدها را بخوانند.
func (s *Service) Departments(context.Context) ([]contracts.Department, error) {
	return infra.Departments(), nil
}

/* --------------------------- عکس کارگزینی --------------------------- */

// ErrPhotoStoreUnavailable انبار فایل عکس در این اجرا آماده نیست.
var ErrPhotoStoreUnavailable = fmt.Errorf("انبار عکس کارگزینی در دسترس نیست")

// PhotoState وضعیت عکس کارگزینی یک کارمند را برای رابط می‌سازد.
func (s *Service) PhotoState(ctx context.Context, f contracts.EmployeeFilter, id string) (contracts.EmployeePhoto, error) {
	// وجود کارمند و محدودهٔ دسترسی همان‌جا سنجیده می‌شود؛ عکس کارمند خارج از محدوده
	// نقش نباید خوانده شود.
	if _, ok, err := s.Get(ctx, f, id); err != nil {
		return contracts.EmployeePhoto{}, err
	} else if !ok {
		return contracts.EmployeePhoto{}, infra.ErrNotFound
	}
	if s.photos == nil {
		return contracts.EmployeePhoto{}, ErrPhotoStoreUnavailable
	}
	photo, ok, err := s.photos.Get(id)
	if err != nil {
		return contracts.EmployeePhoto{}, err
	}
	state := contracts.EmployeePhoto{EmployeeID: id, HasPhoto: ok}
	if ok {
		state.URL = photos.URL(photo.Name, photo.EmployeeID, s.photoKey)
		state.UpdatedAt = photo.UpdatedAt
		state.UpdatedBy = photo.UpdatedBy
		state.SizeKb = (photo.SizeBytes + 1023) / 1024
		state.ContentType = photo.ContentType
		state.SHA256 = photo.SHA256
		state.OriginalName = photo.OriginalName
		state.NoteFa = "عکس کارگزینی این کارمند بارگذاری شده است و در پرونده و ساختار سازمانی نمایش داده می‌شود."
		return state, nil
	}
	state.NoteFa = "برای این کارمند عکس کارگزینی بارگذاری نشده است؛ می‌توانید عکس رسمی پرونده را اینجا اضافه کنید."
	return state, nil
}

// SavePhoto عکس کارگزینی کارمند را ذخیره می‌کند (فایل + نگاشت + رخداد حسابرسی).
func (s *Service) SavePhoto(ctx context.Context, id, filename string, data []byte, actor string) (contracts.EmployeePhoto, error) {
	employee, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return contracts.EmployeePhoto{}, err
	}
	if !ok {
		return contracts.EmployeePhoto{}, infra.ErrNotFound
	}
	if s.photos == nil {
		return contracts.EmployeePhoto{}, ErrPhotoStoreUnavailable
	}
	photo, err := s.photos.Save(employee.ID, filename, data, actor, s.now())
	if err != nil {
		return contracts.EmployeePhoto{}, err
	}
	if s.audit != nil {
		s.audit.Append(actor, "employee", "photo", employee.ID, "بارگذاری عکس کارگزینی — "+employee.FullName)
	}
	if s.sync != nil {
		s.sync.Record("EmployeePhoto", synclog.OpUpsert, employee.ID, 0, map[string]any{
			"employeeId": employee.ID, "name": photo.Name, "sha256": photo.SHA256,
		})
	}
	return contracts.EmployeePhoto{
		EmployeeID: employee.ID, HasPhoto: true,
		URL: photos.URL(photo.Name, employee.ID, s.photoKey), UpdatedAt: photo.UpdatedAt,
		UpdatedBy: photo.UpdatedBy, SizeKb: (photo.SizeBytes + 1023) / 1024,
		ContentType: photo.ContentType, SHA256: photo.SHA256, OriginalName: photo.OriginalName,
		NoteFa: "عکس کارگزینی ذخیره شد؛ از همین حالا در پروندهٔ پرسنلی و ساختار سازمانی دیده می‌شود.",
	}, nil
}

// DeletePhoto عکس کارگزینی کارمند را پاک می‌کند.
func (s *Service) DeletePhoto(ctx context.Context, id, actor string) error {
	employee, ok, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return infra.ErrNotFound
	}
	if s.photos == nil {
		return ErrPhotoStoreUnavailable
	}
	if err := s.photos.Remove(employee.ID); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Append(actor, "employee", "photo_delete", employee.ID, "حذف عکس کارگزینی — "+employee.FullName)
	}
	if s.sync != nil {
		s.sync.Record("EmployeePhoto", synclog.OpDelete, employee.ID, 0, nil)
	}
	return nil
}

// PhotoIndex نگاشت «کد پرسنلی → عکس» برای غنی‌سازی فهرست‌های بیرونی (ساختار سازمانی).
func (s *Service) PhotoIndex() (map[string]contracts.EmployeePhoto, error) {
	all, err := s.photoIndex()
	if err != nil {
		return nil, err
	}
	out := make(map[string]contracts.EmployeePhoto, len(all))
	for id, photo := range all {
		out[id] = contracts.EmployeePhoto{
			EmployeeID: id, HasPhoto: true,
			URL: photos.URL(photo.Name, id, s.photoKey), UpdatedAt: photo.UpdatedAt,
			SizeKb: (photo.SizeBytes + 1023) / 1024, ContentType: photo.ContentType,
		}
	}
	return out, nil
}

// photoIndex نگاشت خام عکس‌ها از انبار فایل؛ انبار نبود یعنی نگاشت خالی.
func (s *Service) photoIndex() (map[string]photos.Photo, error) {
	if s.photos == nil {
		return map[string]photos.Photo{}, nil
	}
	return s.photos.All()
}

// attachPhoto نشانی و تاریخ عکس را روی سطر کارمند می‌نشاند.
func (s *Service) attachPhoto(dto *contracts.Employee, photo photos.Photo) {
	if dto == nil || photo.Name == "" {
		return
	}
	dto.PhotoURL = photos.URL(photo.Name, photo.EmployeeID, s.photoKey)
	dto.PhotoUpdatedAt = photo.UpdatedAt
	dto.PhotoSizeKb = (photo.SizeBytes + 1023) / 1024
}

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
