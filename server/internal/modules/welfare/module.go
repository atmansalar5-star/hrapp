// Package welfare ماژول رفاه، سرویس مدارس و اسکان.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: درخواست رفاهی و تسهیلات رفاهی هر یک یک
// نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package welfare

import (
	"net/http"

	"hrms/internal/modules/welfare/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول رفاه.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "welfare" }

// Register مسیرهای استاندارد رکوردهای رفاه را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/welfare")
}
