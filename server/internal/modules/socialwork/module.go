// Package socialwork ماژول مددکاری اجتماعی.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: پروندهٔ مددکاری و درخواست کمک موردی هر یک
// یک نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package socialwork

import (
	"net/http"

	"hrms/internal/modules/socialwork/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول مددکاری اجتماعی.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "socialwork" }

// Register مسیرهای استاندارد رکوردهای مددکاری را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/socialwork")
}
