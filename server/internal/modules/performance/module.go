// Package performance ماژول مدیریت عملکرد.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: دورهٔ ارزیابی، فرم ارزیابی و نتیجهٔ
// ارزیابی هر یک یک نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package performance

import (
	"net/http"

	"hrms/internal/modules/performance/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول مدیریت عملکرد.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "performance" }

// Register مسیرهای استاندارد رکوردهای مدیریت عملکرد را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/performance")
}
