// Package hrstudies ماژول مطالعات منابع انسانی.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: پروژهٔ مطالعاتی، نظرسنجی و پیشنهاد کارکنان
// هر یک یک نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package hrstudies

import (
	"net/http"

	"hrms/internal/modules/hrstudies/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول مطالعات منابع انسانی.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "hrstudies" }

// Register مسیرهای استاندارد رکوردهای مطالعات را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/hrstudies")
	// دو تبِ اعلام‌شدهٔ قرارداد (شاخص‌ها و نظرسنجی‌ها) روی همان رکوردها سوار می‌شوند:
	// محاسبهٔ دامنه‌ای انجام می‌شود و رابط فقط نتیجه را نمایش می‌دهد.
	registerAnalyticsEndpoints(mux, deps, svc)
	deps.Logger(m.ID()).Info("ماژول مطالعات با تب شاخص‌ها و تحلیل نظرسنجی ثبت شد")
}
