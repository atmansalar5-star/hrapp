// Package grievances ماژول شکایات و امور ایثارگران.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: شکایت کارمند و پروندهٔ ایثارگری هر یک یک
// نوع رکورد با گردشکار خودش است و قواعدش در بستهٔ domain اعلام شده.
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

// Register مسیرهای استاندارد رکوردها و مسیرهای محاسبهٔ امتیاز ایثارگری را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/grievances")
	// مسیرهای اختصاصی امور ایثارگران روی همان سرویس رکورد سوار می‌شوند تا محاسبهٔ
	// امتیازها یک‌جا انجام شود و رابط وب خودش قاعده نسازد.
	registerVeteranEndpoints(mux, deps, svc)
	// فرم مصوبهٔ کمیتهٔ ایثارگری روی همان سرویس رکورد می‌نشیند تا تصمیم کمیته ثبت،
	// چاپ و در صورت تأیید روی پروندهٔ کارمند اعمال شود.
	registerVeteranCommitteeEndpoints(mux, deps, svc)
}
