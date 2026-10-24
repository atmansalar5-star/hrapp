// Package archive ماژول بایگانی و شناسایی کارکنان (زیرمجموعهٔ واحد امور کارکنان).
//
// پیاده‌سازی واقعی: قواعد در بستهٔ domain اعلام شده و اینجا با موتور مشترک `recordkit`
// سوار می‌شود؛ پروندهٔ پرسنلی، مشخصات هویتی و اسناد بایگانی همه روی دادهٔ واقعی کار می‌کنند.
package archive

import (
	"net/http"

	"hrms/internal/modules/archive/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول بایگانی و شناسایی کارکنان.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "archive" }

// Register مسیرهای استاندارد رکوردهای بایگانی را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/archive")
}
