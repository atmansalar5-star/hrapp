// Package grievances ماژول شکایات و امور ایثارگران.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: شکایت کارمند و پروندهٔ ایثارگری هر یک یک
// نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package grievances

import (
	"net/http"

	"hrms/internal/modules/grievances/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول شکایات و امور ایثارگران.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "grievances" }

// Register مسیرهای استاندارد رکوردهای شکایات را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/grievances")
}
