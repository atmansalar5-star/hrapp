// Package orgstructure ماژول ساختار سازمانی و تشکیلات.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: شناسنامهٔ پست سازمانی یک نوع رکورد است و
// قواعدش در بستهٔ domain اعلام شده؛ نمودار درختی واحدها از دادهٔ ساختاری سازمان می‌آید.
package orgstructure

import (
	"net/http"
	"strings"

	"hrms/internal/modules/orgstructure/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// orgCodeScheme پیش‌شماره‌های مؤثر کدهای نمودار سازمانی از پارامترهای سازمانی.
//
// کلید پارامتر: orgcode.prefix.<کلید نوع کد>؛ مقدار نامعتبر نادیده گرفته می‌شود و پیش‌فرض
// (۰۸۰ تا ۰۸۵) می‌ماند.
func orgCodeScheme(deps modulekit.Deps) rules.OrgCodeScheme {
	values := map[string]string{}
	for _, p := range deps.Org.Params {
		if strings.HasPrefix(p.Key, "orgcode.prefix.") {
			values[p.Key] = p.Value
		}
	}
	return rules.OrgCodeSchemeFromValues(values)
}

// Module ثبت ماژول ساختار سازمانی و تشکیلات.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "orgstructure" }

// Register مسیرهای استاندارد رکوردهای پست‌های سازمانی را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	// پیش‌شماره‌های شش کد نمودار سازمانی از پارامترهای سازمانی همین استقرار خوانده می‌شود
	// (`orgcode.prefix.*`)؛ پس تغییر پیش‌شماره نیازی به انتشار نسخهٔ تازه ندارد.
	scheme := orgCodeScheme(deps)
	svc := recordkit.Mount(mux, deps, domain.DescriptorWith(scheme), domain.SeedWith(scheme), "/api/v1/orgstructure")
	// مسیرهای ترکیبی «ساختار سازمانی»: متصدی هر پست با عکس کارگزینی و نتیجهٔ احراز
	// شرایط، سابقهٔ انتصاب‌ها و تاریخچهٔ تغییرات هر سمت.
	registerRosterEndpoints(mux, deps, svc)
	// پورت نوشتن: ماژول انتصابات پس از تصویب ابلاغ، تصدی پست را از همین راه ثبت می‌کند
	// تا نمودار سازمانی متصدی تازه را ببیند.
	registerApplier(deps, svc)
}
