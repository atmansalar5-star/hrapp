// Package career ماژول انتصابات، ترفیعات و کارراههٔ شغلی.
//
// این ماژول از موتور مشترک `recordkit` استفاده می‌کند: قواعدش (انواع رکورد، فیلدها،
// وضعیت‌ها، انتقال‌ها، اقدام‌ها) در بستهٔ domain اعلام شده و اینجا فقط با یک فراخوانی
// سوار می‌شود. نام و واحد کارمند از پورت مشترک کارکنان می‌آید.
package career

import (
	"net/http"

	"hrms/internal/modules/career/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول انتصابات و کارراهه.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول (هم‌نام مانیفست).
func (m *Module) ID() string { return "career" }

// Register انبار، سرویس و مسیرهای استاندارد رکوردها را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/career")
	// تصویب‌خواهی و اجرا: پارامترهای آستانه، سنجش شرط‌ها با زنجیرهٔ تصویب و اجرای ابلاغ
	// روی همان رکورد. قاعده در موتور قواعد است؛ این مسیرها فقط وصل‌کننده‌اند.
	registerApprovalEndpoints(mux, deps, svc)
}
