// Package payroll ماژول حقوق و دستمزد (جبران خدمات).
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: دورهٔ پردازش، آیتم حقوقی و کارانه هر یک
// یک نوع رکورد با گردش‌کار تأیید است و قواعدش در بستهٔ domain اعلام شده.
package payroll

import (
	"net/http"

	"hrms/internal/modules/payroll/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول حقوق و دستمزد.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "payroll" }

// Register مسیرهای استاندارد رکوردهای حقوق و دستمزد و مسیرهای پارامتریک (پارامترها،
// گزینه‌ها و شبیه‌سازی فیش) را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/payroll")
	// پورت فقط‌خواندنی رکوردهای حقوق (دوره‌ها، اقلام و کارانه) تا ماژول‌های دیگر — مثل
	// آمار — بتوانند از راه قرارداد جمع بزنند و مرز ماژول نشکند.
	deps.Ports.SetRecordList(m.ID(), svc)
	registerFinanceEndpoints(mux, deps, svc)
	// اتصال حقوق به کارکرد: خواندن کارکرد دوره از پورت ماژول حضور و محاسبهٔ دوره.
	registerWorkTimeEndpoints(mux, deps, svc)
	// محاسبهٔ پاداش پایان خدمت (پاداش سنوات کارکنان رسمی): درصدها از همان پارامترهای
	// حقوق خوانده می‌شود؛ پس اگر مدیر درصدی را در صفحهٔ پارامترها عوض کند، محاسبه هم
	// بی‌نیاز از انتشار نسخهٔ تازه همان مقدار را می‌بیند.
	registerServiceBonusEndpoints(mux, deps, svc)

	// منبع دادهٔ گزارش‌ساز: اقلام حقوق را به‌عنوان ستون‌های انتخاب‌شدنی معرفی می‌کند.
	deps.Ports.RegisterReportSource(payrollReportSource{svc: svc})
	deps.Logger(m.ID()).Info("ماژول حقوق و دستمزد با پارامترها و قلم‌های پارامتریک ثبت شد")
}
