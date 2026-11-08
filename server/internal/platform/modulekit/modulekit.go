// Package modulekit قرارداد ثبت ماژول و وابستگی‌های تزریقی آن است.
//
// هر ماژول فقط این قرارداد را می‌شناسد: شناسهٔ خودش و تابع Register. هیچ ماژولی
// نمی‌تواند مستقیم به داخلی‌های ماژول دیگر دست بزند؛ دسترسی بین‌ماژولی تنها از راه
// portهای تعریف‌شده در بستهٔ contracts ممکن است (اصل مرز ماژول در CI هم بررسی می‌شود).
package modulekit

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/access"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modules"
	"hrms/internal/platform/orgs"
	"hrms/internal/platform/rbac"
	"hrms/internal/platform/synclog"
	"hrms/internal/rules"
)

// Module یک ماژول ثبت‌شدنی.
type Module interface {
	// ID شناسهٔ ماژول، هم‌نام مانیفست.
	ID() string
	// Register مسیرهای ماژول را روی mux سوار می‌کند.
	Register(mux *http.ServeMux, deps Deps)
}

// Deps وابستگی‌های مشترکی است که به هر ماژول داده می‌شود.
type Deps struct {
	Log      *slog.Logger
	Registry *modules.Registry
	Auth     *auth.Service
	Policy   *rbac.Authorizer
	Audit    *audit.Log
	Sync     *synclog.Log
	Calendar calendar.WorkCalendar
	// Params پارامترهای سراسری فعال سامانه (جدول مالیات، ضرایب، سطوح تأیید).
	Params json.RawMessage
	// WebDir پوشهٔ فایل‌های رابط وب برای اجرای محلی روی یک مبدأ واحد؛ خالی یعنی فقط API.
	WebDir string
	Ports  *Ports
	Clock  func() time.Time
	// DB اتصال پایگاه‌دادهٔ سامانه؛ nil یعنی اجرای درون‌حافظه‌ای (توسعه و آزمون).
	// ماژول‌ها از این اتصال فقط برای انبار خودشان استفاده می‌کنند و هیچ کوئری
	// بین‌schemaی نمی‌نویسند.
	DB *sql.DB
	// SeedDemo بذرکاری دادهٔ نمونه را حتی روی پایگاه‌دادهٔ واقعی روشن می‌کند.
	//
	// پیش‌فرض (false) این است که در استقرار واقعی هیچ دادهٔ نمونه‌ای ساخته نشود،
	// چون سازمان دادهٔ خودش را وارد می‌کند. ولی نصب تازه با پایگاه‌دادهٔ خالی باید
	// بتواند همهٔ ماژول‌ها را کارکردی ببیند؛ پس با این پرچم همان دادهٔ نمونهٔ
	// آزمایشی (احکام، فیش و دوره‌های حقوق، قراردادها، تکفل…) در پایگاه‌داده ریخته
	// می‌شود — و فقط اگر جدول‌های آن ماژول خالی باشند (بذرکاری تکرارپذیر است و
	// دادهٔ واردشده را بازنویسی نمی‌کند).
	SeedDemo bool
	// Org استقرار پارامتریک فعال (شرکت): پارامترها، برند و ماژول‌های فعال از آن می‌آید.
	Org orgs.Organization
	// DataDir پوشهٔ دادهٔ سرور: فایل‌های بارگذاری‌شده (نشان شرکت) و پشتیبان‌ها اینجا
	// می‌مانند تا به‌روزرسانی برنامه آن‌ها را پاک نکند.
	DataDir string
	// VerifyKey کلید امضای نشانی «تأیید سند» که QR کد سندهای چاپی به آن اشاره می‌کند؛
	// با آن می‌توان سنجید که نشانی واقعاً همین سرور ساخته است.
	VerifyKey []byte
	// Access سطوح دسترسی افزودهٔ مدیر سیستم (نقش × اداره × ماژول). فقط ماژول تنظیمات
	// آن را می‌خواند و می‌نویسد؛ اجرای سیاست همچنان در rbac انجام می‌شود.
	Access *access.Store
	// Records انبار سفارشی رکوردهای گردش‌کاری. اگر مقدار داشته باشد، همهٔ ماژول‌های
	// recordkit به‌جای انبار پیش‌فرض (Postgres/حافظه) از همین استفاده می‌کنند. نوع آن
	// در recordkit.Repository اعلام شده و اینجا any است تا حلقهٔ import میان
	// modulekit و recordkit ساخته نشود؛ تنها مصرف‌کننده (recordkit.RepositoryFor)
	// با type assertion آن را می‌خواند. اجرای مستقل یک‌رایانه‌ای از همین راه انبار
	// فایل JSON را تزریق می‌کند.
	Records any
}

// Ports دسترسی بین‌ماژولی از طریق قراردادهای صادرشده.
//
// ماژول‌ها هنگام Register، پیاده‌سازی خود را اینجا ثبت می‌کنند و ماژول‌های دیگر آن را
// مصرف می‌کنند — بدون هیچ import مستقیمی بین پکیج‌های ماژول.
type Ports struct {
	mu            sync.RWMutex
	employee      contracts.EmployeeReader
	employeeWrite contracts.EmployeeAssigner
	appointer     contracts.AppointmentApplier
	documents     contracts.ContractDocumentProvider
	templates     contracts.DocumentTemplateProvider
	reportSources []contracts.ReportSource
	workflowSrcs  []contracts.WorkflowSource
	payElements   rules.PayElementProvider
	decrees       contracts.SalaryDecreeReader
	recordLists   map[string]contracts.RecordListPort
	workTime      contracts.WorkTimeProvider
	catalogs      contracts.CatalogReader
	dependents    contracts.DependentReader
	historySrcs   []contracts.HistorySource
	jobs          contracts.JobRegistryReader
}

// NewPorts مجموعهٔ خالی portها.
func NewPorts() *Ports { return &Ports{} }

// SetDependentReader ثبت پورت افراد تحت تکفل (توسط ماژول امور کارکنان) تا ماژول کارت‌های
// رفاهی بداند چه کسی مشمول کدام خدمت است و فقط افراد با تعلق ثبت‌شده را شارژ کند.
func (p *Ports) SetDependentReader(r contracts.DependentReader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dependents = r
}

// Dependents خواندن پورت افراد تحت تکفل؛ اگر ماژول ثبت نکرده باشد false برمی‌گردد.
func (p *Ports) Dependents() (contracts.DependentReader, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.dependents, p.dependents != nil
}

// SetJobRegistry ثبت پورت شناسنامهٔ شغل (توسط ماژول طبقه‌بندی مشاغل) تا تصویب‌خواهی
// انتصاب و ارتقا شرایط احراز شغل را از سند مرجع بخواند، نه از ورودی دستی کاربر.
func (p *Ports) SetJobRegistry(r contracts.JobRegistryReader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jobs = r
}

// JobRegistry خواندن پورت شناسنامهٔ شغل؛ اگر ماژول ثبت نکرده باشد false برمی‌گردد و
// مصرف‌کننده به شرط دستی برمی‌گردد.
func (p *Ports) JobRegistry() (contracts.JobRegistryReader, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.jobs, p.jobs != nil
}

// SetCatalogReader ثبت پورت کاتالوگ‌های مرجع (توسط ماژول تنظیمات) تا ماژول‌های وام و
// پاداش فهرست مدیریت‌ها، بانک‌ها و واحدها را از مرجع قابل ویرایش بخوانند.
func (p *Ports) SetCatalogReader(r contracts.CatalogReader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.catalogs = r
}

// Catalogs خواندن پورت کاتالوگ‌های مرجع. اگر ماژول تنظیمات ثبت نشده باشد، false
// برمی‌گردد و مصرف‌کننده به فهرست پایهٔ خودش برمی‌گردد.
func (p *Ports) Catalogs() (contracts.CatalogReader, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.catalogs, p.catalogs != nil
}

// SetEmployeeReader ثبت پیاده‌سازی پورت کارکنان (توسط ماژول employee).
func (p *Ports) SetEmployeeReader(r contracts.EmployeeReader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.employee = r
}

// Employees خواندن پورت کارکنان (توسط ماژول‌های مصرف‌کننده مانند داشبورد).
func (p *Ports) Employees() (contracts.EmployeeReader, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.employee, p.employee != nil
}

// SetEmployeeAssigner ثبت پورت «تغییر سازمانی پروندهٔ کارمند» (توسط ماژول employee).
//
// چرا جدا از پورت خواندن: فقط ماژول انتصابات پس از تصویب ابلاغ باید پست و واحد کارمند را
// عوض کند؛ بقیهٔ ماژول‌ها حق نوشتن در پروندهٔ کارمند را نمی‌گیرند.
func (p *Ports) SetEmployeeAssigner(a contracts.EmployeeAssigner) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.employeeWrite = a
}

// EmployeeAssigner خواندن پورت «تغییر سازمانی پروندهٔ کارمند»؛ نبودِ آن false برمی‌گرداند.
func (p *Ports) EmployeeAssigner() (contracts.EmployeeAssigner, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.employeeWrite, p.employeeWrite != nil
}

// SetAppointmentApplier ثبت پورت «ثبت تصدی پست در تشکیلات» (توسط ماژول orgstructure).
func (p *Ports) SetAppointmentApplier(a contracts.AppointmentApplier) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.appointer = a
}

// AppointmentApplier خواندن پورت «ثبت تصدی پست در تشکیلات».
func (p *Ports) AppointmentApplier() (contracts.AppointmentApplier, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.appointer, p.appointer != nil
}

// SetContractDocuments ثبت پورت دادهٔ قرارداد برای اسناد چاپی (توسط ماژول contracts).
func (p *Ports) SetContractDocuments(d contracts.ContractDocumentProvider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.documents = d
}

// ContractDocuments خواندن پورت قرارداد برای ساخت حکم حقوق و گواهی اشتغال.
func (p *Ports) ContractDocuments() (contracts.ContractDocumentProvider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.documents, p.documents != nil
}

// SetDocumentTemplates ثبت پورت قالب‌های متن اسناد (توسط ماژول تنظیمات).
func (p *Ports) SetDocumentTemplates(t contracts.DocumentTemplateProvider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.templates = t
}

// DocumentTemplates خواندن پورت قالب‌های متن اسناد (توسط موتور اسناد و ماژول قراردادها).
func (p *Ports) DocumentTemplates() (contracts.DocumentTemplateProvider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.templates, p.templates != nil
}

// SetSalaryDecreeReader ثبت پورت خواندن حکم حقوق (توسط ماژول احکام) تا محاسبهٔ پاداش
// پایان خدمت بتواند پایه و نوع استخدام را از حکم ابلاغ‌شدهٔ کارمند بردارد.
func (p *Ports) SetSalaryDecreeReader(r contracts.SalaryDecreeReader) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.decrees = r
}

// SalaryDecrees خواندن پورت حکم حقوق؛ اگر ماژول احکام ثبت نشده باشد false برمی‌گردد.
func (p *Ports) SalaryDecrees() (contracts.SalaryDecreeReader, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.decrees, p.decrees != nil
}

// SetPayElements ثبت پورت اقلام حقوق (توسط ماژول حقوق) تا موتور اسناد بتواند همان
// تفکیک اقلام شبیه‌ساز را در فیش چاپی نشان دهد.
func (p *Ports) SetPayElements(provider rules.PayElementProvider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.payElements = provider
}

// PayElements خواندن پورت اقلام حقوق (توسط موتور اسناد).
func (p *Ports) PayElements() (rules.PayElementProvider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.payElements, p.payElements != nil
}

// SetWorkTime ثبت پورت «کارکرد دوره» توسط ماژول حضور. ماژول حقوق از همین قرارداد
// کارکرد کارکنان را می‌خواند و هیچ import مستقیمی بین دو ماژول وجود ندارد.
func (p *Ports) SetWorkTime(w contracts.WorkTimeProvider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workTime = w
}

// WorkTime پورت کارکرد دوره؛ اگر ماژول حضور ثبت نکرده باشد، false برمی‌گردد.
func (p *Ports) WorkTime() (contracts.WorkTimeProvider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.workTime == nil {
		return nil, false
	}
	return p.workTime, true
}

// SetRecordList ثبت پورت فقط‌خواندنی رکوردهای یک ماژول (توسط خودِ همان ماژول) تا
// ماژول‌های آماری بتوانند روی داده‌اش جمع بزنند بدون شکستن مرز ماژول.
func (p *Ports) SetRecordList(moduleID string, l contracts.RecordListPort) {
	if l == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.recordLists == nil {
		p.recordLists = map[string]contracts.RecordListPort{}
	}
	p.recordLists[moduleID] = l
}

// RecordModules شناسهٔ ماژول‌هایی که شناسهٔ رکورد خود را ثبت کرده‌اند (به ترتیب
// الفبا). مصرف‌کننده: پنل پارامتریک تنظیمات — برای ساختن کاتالوگ فرم‌ها و فیلدهایشان
// از خودِ اعلام ماژول‌ها، بدون هیچ فهرست دستی در ماژول تنظیمات.
func (p *Ports) RecordModules() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, 0, len(p.recordLists))
	for id, l := range p.recordLists {
		if l != nil {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// RecordList خواندن پورت رکوردهای یک ماژول با شناسهٔ آن ماژول.
func (p *Ports) RecordList(moduleID string) (contracts.RecordListPort, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	l, ok := p.recordLists[moduleID]
	return l, ok && l != nil
}

// RegisterReportSource ثبت یک منبع دادهٔ گزارش (توسط ماژول مالک همان داده).
func (p *Ports) RegisterReportSource(s contracts.ReportSource) {
	if s == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reportSources = append(p.reportSources, s)
}

// ReportSources فهرست منابع دادهٔ گزارش به ترتیب ثبت.
func (p *Ports) ReportSources() []contracts.ReportSource {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]contracts.ReportSource, len(p.reportSources))
	copy(out, p.reportSources)
	return out
}

// RegisterWorkflowSource ثبت یک منبع کارهای گردش‌کاری (توسط ماژول مالک داده).
// کارتابل سراسری از همین فهرست ساخته می‌شود و هیچ ماژولی داخلی‌های ماژول دیگر را
// نمی‌بیند.
func (p *Ports) RegisterWorkflowSource(s contracts.WorkflowSource) {
	if s == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workflowSrcs = append(p.workflowSrcs, s)
}

// WorkflowSources فهرست منابع کارهای گردش‌کاری به ترتیب ثبت.
func (p *Ports) WorkflowSources() []contracts.WorkflowSource {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]contracts.WorkflowSource, len(p.workflowSrcs))
	copy(out, p.workflowSrcs)
	return out
}

// RegisterHistorySource ثبت منبع «سابقهٔ کارمند» (توسط ماژول مالک همان داده).
//
// صفحهٔ «سوابق تغییرات» پروندهٔ کارمند از همین فهرست ساخته می‌شود: هر ماژول سابقهٔ خودش
// را اعلام می‌کند و لایهٔ تجمیع، بر پایهٔ دسترسی نقش، سطرها را از همهٔ منابع می‌چیند.
func (p *Ports) RegisterHistorySource(s contracts.HistorySource) {
	if s == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.historySrcs = append(p.historySrcs, s)
}

// HistorySources فهرست منابع سابقه به ترتیب ثبت.
func (p *Ports) HistorySources() []contracts.HistorySource {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]contracts.HistorySource, len(p.historySrcs))
	copy(out, p.historySrcs)
	return out
}

// ReportSource یافتن یک منبع دادهٔ گزارش با کلید.
func (p *Ports) ReportSource(key string) (contracts.ReportSource, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, s := range p.reportSources {
		if s.Metadata().Key == key {
			return s, true
		}
	}
	return nil, false
}

// Enabled بررسی فعال بودن ماژول در این استقرار.
func (d Deps) Enabled(moduleID string) bool {
	return d.Registry.IsEnabled(moduleID)
}

// Logger برای ماژول بازمی‌گرداند که نام ماژول را به هر رکورد اضافه می‌کند.
func (d Deps) Logger(moduleID string) *slog.Logger {
	if d.Log == nil {
		return slog.Default().With("module", moduleID)
	}
	return d.Log.With("module", moduleID)
}

// Require میان‌افزار دسترسی ماژول.
func (d Deps) Require(moduleID, action string, next http.HandlerFunc) http.HandlerFunc {
	return d.Policy.Require(moduleID, action, next)
}

// ModuleTitle عنوان فارسی یک ماژول از رجیستری.
//
// چرا اینجا: منابع سابقه/گزارش باید ماژول خود را به کاربر معرفی کنند، ولی متن فارسی
// نباید در کد ماژول هاردکد شود؛ عنوان از همان مانیفست نسخه‌دار می‌آید و منبع یگانهٔ
// حقیقت می‌ماند. ماژول ناشناخته همان شناسه را برمی‌گرداند تا پاسخ صادق بماند.
func (d Deps) ModuleTitle(moduleID string) string {
	if d.Registry != nil {
		if m, ok := d.Registry.ByID(moduleID); ok && m.TitleFa != "" {
			return m.TitleFa
		}
	}
	return moduleID
}

// Now زمان جاری سامانه (قابل تزریق در آزمون).
func (d Deps) Now() time.Time {
	if d.Clock == nil {
		return time.Now()
	}
	return d.Clock()
}

// StubRoutes مسیرهای ماژول‌هایی را که در این نسخه پیاده نشده‌اند با پاسخ ساختاریافتهٔ
// ۵۰۱ ثبت می‌کند. این کار عمدی است: کلاینت باید پاسخ JSON بگیرد (نه ۴۰۴ خام) تا
// بتواند پیام «در فاز بعدی» را درست نمایش دهد و قرارداد API نیز صادق بماند.
func StubRoutes(mux *http.ServeMux, deps Deps, moduleID string, routes []string) {
	if !deps.Registry.IsEnabled(moduleID) {
		return
	}
	handler := httpx.NotImplemented(moduleID)
	for _, route := range routes {
		mux.HandleFunc(route, deps.Require(moduleID, "read", handler))
	}
}

// RecordAudit ثبت رخداد حسابرسی با کاربر جاری.
func (d Deps) RecordAudit(r *http.Request, moduleID, action, target, detail string) {
	if d.Audit == nil {
		return
	}
	actor := "ناشناس"
	if user, ok := auth.UserFrom(r.Context()); ok {
		actor = user.Username
	}
	d.Audit.Append(actor, moduleID, action, target, detail)
}
