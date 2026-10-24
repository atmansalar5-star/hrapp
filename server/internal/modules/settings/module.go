// Package settings ماژول تنظیمات.
//
// پارامتر سیستمی و پارامتر سازمانی روی موتور مشترک `recordkit` کار می‌کنند و قواعدشان در
// بستهٔ domain اعلام شده؛ صفحه‌های مدیریت کاربران، نقش‌ها و لاگ حسابرسی به‌صورت دست‌ساز و
// با سیاست دسترسی همین ماژول سرو می‌شوند.
package settings

import (
	"net/http"

	"hrms/internal/modules/settings/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول تنظیمات.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "settings" }

// Register مسیرهای پارامترها و فهرست‌های سیستمی را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/settings")
	registerAdminEndpoints(mux, deps)
	registerAccessEndpoints(mux, deps)
	// تقویم کاری: تعطیل هفتگی/رسمی استقرار و پارامترهای حضور و مرخصی شرکت فعال.
	registerCalendarEndpoint(mux, deps)
	// پورت قالب‌های متن اسناد: موتور اسناد چاپی و ماژول قراردادها از همین‌جا قالب فعال
	// را می‌خوانند؛ بدون هیچ import مستقیمی بین ماژول‌ها.
	deps.Ports.SetDocumentTemplates(templateProvider{svc: svc})
	// پورت کاتالوگ‌های مرجع: مدیریت‌ها، بانک‌ها و واحدها. ماژول‌های وام و پاداش فهرست
	// فرم‌های خود را از همین‌جا می‌سازند، پس افزودن قلم تازه در پنل تنظیمات خودکار در
	// فرم‌های آن‌ها دیده می‌شود.
	deps.Ports.SetCatalogReader(catalogProvider{svc: svc})
}
