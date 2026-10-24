// Package discipline ماژول کمیتهٔ انضباط کار.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: پروندهٔ انضباطی و جلسهٔ کمیته هر یک یک
// نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package discipline

import (
	"net/http"

	"hrms/internal/modules/discipline/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول کمیتهٔ انضباط کار.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "discipline" }

// Register مسیرهای استاندارد رکوردهای انضباط کار را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/discipline")
}
