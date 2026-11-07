// Package orgstructure ماژول ساختار سازمانی و تشکیلات.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: شناسنامهٔ پست سازمانی یک نوع رکورد است و
// قواعدش در بستهٔ domain اعلام شده؛ نمودار درختی واحدها از دادهٔ ساختاری سازمان می‌آید.
package orgstructure

import (
	"net/http"

	"hrms/internal/modules/orgstructure/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول ساختار سازمانی و تشکیلات.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "orgstructure" }

// Register مسیرهای استاندارد رکوردهای پست‌های سازمانی را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/orgstructure")
	// مسیرهای ترکیبی «ساختار سازمانی»: متصدی هر پست با عکس کارگزینی و نتیجهٔ احراز
	// شرایط، سابقهٔ انتصاب‌ها و تاریخچهٔ تغییرات هر سمت.
	registerRosterEndpoints(mux, deps, svc)
	// پورت نوشتن: ماژول انتصابات پس از تصویب ابلاغ، تصدی پست را از همین راه ثبت می‌کند
	// تا نمودار سازمانی متصدی تازه را ببیند.
	registerApplier(deps, svc)
}
