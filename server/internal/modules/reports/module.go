// Package reports ماژول گزارش‌ها.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: اجرای گزارش و خروجی گزارش هر یک یک نوع
// رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package reports

import (
	"net/http"

	"hrms/internal/modules/reports/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول گزارش‌ها.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "reports" }

// Register مسیرهای استاندارد رکوردهای گزارش و مسیرهای گزارش‌ساز را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/reports")
	registerBuilderEndpoints(mux, deps)
	// منبع دادهٔ گزارش «کارکنان»؛ منبع «اقلام حقوق» را ماژول حقوق و دستمزد ثبت می‌کند.
	deps.Ports.RegisterReportSource(employeeReportSource{deps: deps})
	// منبع «شاخص‌های منابع انسانی شرکت‌های هلدینگ»: نمای تجمیعی/مقایسه‌ای شرکت‌های
	// زیرمجموعهٔ هلدینگ برای مدیران ارشد و واحد آمار و گزارشات.
	deps.Ports.RegisterReportSource(holdingHrSource{deps: deps})
	deps.Logger(m.ID()).Info("ماژول گزارش‌ها با گزارش‌ساز بر پایهٔ اقلام ثبت شد")
}
