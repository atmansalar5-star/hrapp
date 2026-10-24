// Package loans_retirement ماژول امور وام و بازنشستگی.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: درخواست وام، قسط و پروندهٔ بازنشستگی هر
// یک یک نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده. مسیرهای
// اختصاصی (گزینه‌های وام و امتیازبندی تخصیص) در بستهٔ api سوار می‌شوند و منبع دادهٔ
// گزارش وام برای گزارش‌ساز ثبت می‌شود.
package loans_retirement

import (
	"context"
	"net/http"

	"hrms/internal/contracts"
	"hrms/internal/modules/loans_retirement/api"
	"hrms/internal/modules/loans_retirement/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول وام و بازنشستگی.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "loans_retirement" }

// Register مسیرهای استاندارد رکوردهای وام و بازنشستگی، مسیرهای اختصاصی و منبع گزارش
// وام را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	// قاعدهٔ رکورد «درخواست وام» با اقلام کاتالوگ مرجع تنظیمات گسترش می‌یابد تا مدیریت
	// تازه‌ای که کاربر در پنل تنظیمات ساخته، در ثبت تخصیص وام هم پذیرفته شود. وام فقط
	// قرارداد پلتفرم را می‌شناسد و داخلی‌های ماژول تنظیمات را import نمی‌کند.
	desc := domain.AcceptRuntimeManagements(domain.Descriptor(), func(key string) bool {
		reader, ok := deps.Ports.Catalogs()
		if !ok {
			return false
		}
		items, err := reader.Catalog(context.Background(), contracts.CatalogManagements)
		if err != nil {
			return false
		}
		for _, it := range items {
			if it.Key == key && it.Active {
				return true
			}
		}
		return false
	})
	svc := recordkit.Mount(mux, deps, desc, domain.Seed(), "/api/v1/loans")

	h := api.New(svc, deps)
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc { return deps.Require(m.ID(), action, fn) }

	// گزینه‌ها: انواع وام، بانک‌های عامل، مدیریت‌های جدول سهمیه و معیارهای امتیازبندی
	// (رابط از همین می‌سازد).
	mux.HandleFunc("GET /api/v1/loans/options", r("read", h.Options))
	// پیش‌نمایش امتیازبندی تخصیص وام: رتبه‌بندی متقاضیان بر پایهٔ معیارها.
	mux.HandleFunc("POST /api/v1/loans/allocation/preview", r("read", h.PreviewAllocation))
	// تخصیص وام: پیش‌نمایش (بدون ذخیره‌سازی) و ثبت رکوردهای درخواست وام با کارمزد و
	// مبلغ قسط محاسبه‌شده از دامنه.
	mux.HandleFunc("POST /api/v1/loans/allocate/preview", r("read", h.PreviewGrantAllocation))
	mux.HandleFunc("POST /api/v1/loans/allocate", r("write", h.GrantAllocation))
	// داشبورد تخصیص سهمیه: تخصیص به تفکیک مدیریت‌های بندر امام، متقاضیان و مشمولین،
	// مرحلهٔ ارسال به مدیریت مالی، سهم شرکتی/ارکان ثالث و طبقهٔ نرخ کارمزد.
	mux.HandleFunc("GET /api/v1/loans/quota-dashboard", r("read", h.QuotaDashboard))

	// منبع دادهٔ گزارش «وام کارکنان» برای گزارش‌ساز؛ شامل انواع وام، بانک عامل،
	// کارمزد، اقساط، اطلاعات ضامن و زمان دریافت.
	deps.Ports.RegisterReportSource(loansReportSource{svc: svc})

	deps.Logger(m.ID()).Info("ماژول وام و بازنشستگی با فرم تخصیص وام، پنل امتیازبندی، داشبورد سهمیه و منبع گزارش وام ثبت شد", "routes", 16)
}
