// Package sports ماژول امور ورزش.
//
// پیاده‌سازی روی موتور مشترک `recordkit`: برنامهٔ ورزشی و رویداد ورزشی هر یک یک نوع
// رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده. صفحهٔ وب خودکار از
// شناسهٔ سرور ساخته می‌شود و نیازی به نوشتن صفحهٔ دستی نیست.
package sports

import (
	"net/http"

	"hrms/internal/modules/sports/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول امور ورزش.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول (هم‌نام مانیفست).
func (m *Module) ID() string { return "sports" }

// Register مسیرهای استاندارد رکوردهای ورزشی را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/sports")
}
