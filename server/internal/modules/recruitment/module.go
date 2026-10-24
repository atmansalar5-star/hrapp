// Package recruitment ماژول تأمین و جذب نیرو.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: درخواست نیرو، آگهی شغلی و متقاضی هر یک
// یک نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package recruitment

import (
	"net/http"

	"hrms/internal/modules/recruitment/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول تأمین و جذب نیرو.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "recruitment" }

// Register مسیرهای استاندارد رکوردهای تأمین نیرو را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/recruitment")
}
