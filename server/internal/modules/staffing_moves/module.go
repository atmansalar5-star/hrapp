// Package staffing_moves ماژول جابه‌جایی و معرفی کارکنان.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: درخواست جابه‌جایی و معرفی/ابلاغ کارمند هر
// یک یک نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package staffing_moves

import (
	"net/http"

	"hrms/internal/modules/staffing_moves/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول جابه‌جایی و معرفی کارکنان.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "staffing_moves" }

// Register مسیرهای استاندارد رکوردهای جابه‌جایی را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/moves")
}
