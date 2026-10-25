// Package attendance ماژول حضور و غیاب، مرخصی و مأموریت.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: تردد، مرخصی، مأموریت و اضافه‌کاری هر یک
// یک نوع رکورد با گردش‌کار تأیید هستند و قواعدشان در بستهٔ domain اعلام شده است.
package attendance

import (
	"net/http"

	"hrms/internal/modules/attendance/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول حضور و غیاب.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "attendance" }

// Register مسیرهای استاندارد رکوردهای حضور و غیاب را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/attendance")
	// پورت فقط‌خواندنی رکوردهای حضور: ماژول آمار از همین راه جمع می‌زند و مرز ماژول
	// می‌شکند نمی‌شود (مالک داده و دامنهٔ دسترسی همان‌جا می‌ماند).
	deps.Ports.SetRecordList(m.ID(), svc)
	// پورت «کارکرد دوره»: جمع تردد/مرخصی/ماموریت/اضافه‌کاری را به قرارداد کارکرد
	// تبدیل می‌کند تا ماژول حقوق از همان راه به کارکرد لینک شود.
	deps.Ports.SetWorkTime(workTimeProvider{svc: svc, deps: deps})
	// ورود فایل دستگاه تردد، برنامهٔ نوبت و پوشش شیفت (زیرکارهای ۱.۲ و ۱.۳ فاز ۱).
	registerDeviceEndpoints(mux, deps, svc)
}
