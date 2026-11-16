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
	// گزارش تطبیق کارکرد با حقوق (زیرکار ۱.۳ فاز ۱): کارکرد، مبلغ محاسبه‌شده و مبلغ
	// نوشته‌شده روی دورهٔ پردازش کنار هم، با فهرست مغایرت‌ها.
	registerReconcileEndpoints(mux, deps, svc)
	// فایل قانونی بیمه و مالیات (زیرکار ۱.۴) و فایل پرداخت بانکی (زیرکار ۱.۵): ساخت،
	// خروجی CSV و ثبت سند گردش فایل روی همان موتور رکوردها.
	registerStatutoryEndpoints(mux, deps, svc)
	registerBankFileEndpoints(mux, deps, svc)
	// محاسبهٔ پاداش پایان خدمت (پاداش سنوات کارکنان رسمی): درصدها از همان پارامترهای
	// حقوق خوانده می‌شود؛ پس اگر مدیر درصدی را در صفحهٔ پارامترها عوض کند، محاسبه هم
	// بی‌نیاز از انتشار نسخهٔ تازه همان مقدار را می‌بیند.
	registerServiceBonusEndpoints(mux, deps, svc)
	// نظام پرداخت صندوق بازنشستگی نفت: پایهٔ حقوق، فوق‌العادهٔ مناطق گرمسیری و
	// فوق‌العاده‌های درصدی از «مبنای فوق‌العاده‌ها» (اقلیمی و نوبت‌کاری).
	registerOilPayEndpoints(mux, deps, svc)
	// نظام جبران خدمات دستورالعمل IMS-HR-34006: سومین نظام پرداخت مستقل سازمان، با
	// جدول‌های امتیاز خودش (امتیاز گرید شغلی، پست استراتژیک، گرید شخصی، نقش و منطقه) و
	// مبنای «امتیاز × ضریب ریالی».
	registerComp34006Endpoints(mux, deps, svc)
	// کنترل‌پنل «روش پرداخت کارکنان»: تعیین اینکه هر کارمند با کدام مدل محاسبه شود
	// (حقوق و دستمزد جاری، صندوق نفت، دستورالعمل ۳۴۰۰۶ یا مدل توسعه‌ای ۴پی) — بدون
	// تغییر در محاسبهٔ موجود.
	registerPayMethodEndpoints(mux, deps, svc)
	// تطبیق حقوق و «تفاوت تطبیق»: سنجش کاهش حقوق در زمان اجرای طرح تازه و ساخت فرم
	// محاسباتی و نمودار جذب آن (قاعده در بستهٔ rules).
	registerPayGapEndpoints(mux, deps, svc)

	// منبع دادهٔ گزارش‌ساز: اقلام حقوق را به‌عنوان ستون‌های انتخاب‌شدنی معرفی می‌کند.
	deps.Ports.RegisterReportSource(payrollReportSource{svc: svc})
	deps.Logger(m.ID()).Info("ماژول حقوق و دستمزد با پارامترها و قلم‌های پارامتریک ثبت شد")
}
