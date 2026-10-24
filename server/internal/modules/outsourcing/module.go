// Package outsourcing ماژول برون‌سپاری و نیروهای ارکان ثالث.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: پیمانکار، دورهٔ حقوق ارکان ثالث و پروندهٔ
// مشاغل سخت و زیان‌آور هر یک یک نوع رکورد است و قواعدش در بستهٔ domain اعلام شده.
package outsourcing

import (
	"net/http"

	"hrms/internal/modules/outsourcing/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول برون‌سپاری.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "outsourcing" }

// Register مسیرهای استاندارد رکوردهای برون‌سپاری را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/outsourcing")
}
