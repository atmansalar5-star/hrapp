// Package training ماژول هماهنگی آموزش.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: دورهٔ آموزشی، نیاز آموزشی و ثبت‌نام در
// دوره هر یک یک نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package training

import (
	"net/http"

	"hrms/internal/modules/training/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول هماهنگی آموزش.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "training" }

// Register مسیرهای استاندارد رکوردهای آموزش را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/training")
}
