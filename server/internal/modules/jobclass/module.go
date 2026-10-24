// Package jobclass ماژول طبقه‌بندی مشاغل.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: طبقه‌بندی، ارزشیابی و شرح شغل هر یک یک
// نوع رکورد است و قواعدش در بستهٔ domain اعلام شده.
package jobclass

import (
	"net/http"

	"hrms/internal/modules/jobclass/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول طبقه‌بندی مشاغل.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "jobclass" }

// Register مسیرهای استاندارد رکوردهای طبقه‌بندی مشاغل را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/jobclass")
}
