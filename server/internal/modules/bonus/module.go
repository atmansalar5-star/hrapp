// Package bonus ماژول پاداش‌ها.
//
// پیاده‌سازی روی موتور مشترک `recordkit`: هر پرداخت پاداش یک رکورد با گردش‌کار
// پیش‌نویس → تأیید → پرداخت است و قواعدش در بستهٔ domain اعلام شده. مسیرهای اختصاصی
// (گزینه‌ها، داشبورد مدیریتی و تخصیص گروهی) در بستهٔ api سوار می‌شوند و منبع دادهٔ
// گزارش پاداش در گزارش‌ساز ثبت می‌شود.
package bonus

import (
	"net/http"

	"hrms/internal/modules/bonus/api"
	"hrms/internal/modules/bonus/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول پاداش.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "bonus" }

// Register مسیرهای رکوردهای پاداش، مسیرهای اختصاصی داشبورد و تخصیص، و منبع گزارش
// پاداش را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/bonus")

	h := api.New(svc, deps)
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc { return deps.Require(m.ID(), action, fn) }

	// گزینه‌ها: انواع پاداش، مدیریت‌های بندر امام و روش‌های پرداخت (فرم از همین می‌سازد).
	mux.HandleFunc("GET /api/v1/bonus/options", r("read", h.Options))
	// داشبورد مدیریتی: پرداخت به تفکیک مدیریت، نفر، نوع، دوره و وضعیت.
	mux.HandleFunc("GET /api/v1/bonus/dashboard", r("read", h.Dashboard))
	// پیش‌نمایش تخصیص: فقط محاسبه، بی هیچ ذخیره‌سازی.
	mux.HandleFunc("POST /api/v1/bonus/allocate/preview", r("read", h.PreviewAllocation))
	// ثبت تخصیص گروهی: ساخت رکوردهای پرداخت در وضعیت پیش‌نویس.
	mux.HandleFunc("POST /api/v1/bonus/allocate", r("write", h.Allocate))

	// منبع دادهٔ گزارش «پاداش کارکنان» برای گزارش‌ساز.
	deps.Ports.RegisterReportSource(bonusReportSource{svc: svc})

	deps.Logger(m.ID()).Info("ماژول پاداش با داشبورد مدیریتی، فرم تخصیص و منبع گزارش ثبت شد", "routes", 11)
}
