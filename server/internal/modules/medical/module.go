// Package medical ماژول درمان، معاینات دوره‌ای و خدمات درمانی.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: پروندهٔ درمان، معاینهٔ دوره‌ای و خدمت
// درمانی هر یک یک نوع رکورد است و قواعدش در بستهٔ domain اعلام شده.
package medical

import (
	"net/http"

	"hrms/internal/modules/medical/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول درمان و خدمات درمانی.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "medical" }

// Register مسیرهای استاندارد رکوردهای درمان و مسیرهای کمک هزینهٔ درمانی (تعرفه، محاسبه و
// ثبت درخواست) را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/medical")
	registerMedicalAidEndpoints(mux, deps, svc)
}
