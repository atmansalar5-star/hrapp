// Package compensation ماژول «جبران خدمات ۴پی».
//
// پیاده‌سازی روی موتور مشترک `recordkit` است: مدل جبران خدمات، قیمت‌گذاری شغل، سطح
// شایستگی، ضریب عملکرد، باند پرداخت و تصمیم چهارعاملی هر یک یک نوع رکورد با گردشکار
// خودش‌اند و قواعدشان در بستهٔ domain اعلام شده است. قاعدهٔ محاسبهٔ چهارعاملی در موتور
// قواعد (`internal/rules/compensation.go`) است و این لایه فقط داده را می‌خواند، به
// موتور می‌دهد و نتیجه را برمی‌گرداند؛ پس هیچ عددی در رابط کاربری نیست.
//
// دو نظام مستقل در یک بسته: همین بسته میزبان «نظام پرداخت جیب‌یست (G20)» هم هست، ولی دو
// نظام هیچ پارامتر، محاسبه یا دادهٔ مرجع مشترکی ندارند:
//   - نظام ۴پی: قواعد در internal/rules/compensation.go، پارامترها با پیشوند compensation.*،
//     مسیرها در fourp.go و راهنمای شروع بهروز‌رسانی داده در readiness.go
//   - نظام جیب‌یست (G20): قواعد در internal/rules/g20.go، پارامترها با پیشوند
//     compensation.g20.*، مسیرها در g20.go
package compensation

import (
	"net/http"

	"hrms/internal/modules/compensation/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول جبران خدمات.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "compensation" }

// Register مسیرهای رکوردهای جبران خدمات و مسیرهای اختصاصی هر دو نظام (۴پی و G20) را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/compensation")
	// مسیرهای نظام ۴پی (فایل fourp.go و readiness.go):
	//   /options          فهرست‌های مرجع خود این نظام (خانوادهٔ شغلی، سطوح شایستگی و عملکرد)
	//   /4p-defaults      پارامترها، باندها، ضرایب و داده‌های پیش‌فرض یک کارمند
	//   /4p-calc          محاسبهٔ خطبه‌خط بستهٔ جبران خدمات و ثبت تصمیم
	//   /data-readiness   راهنمای شروع بهروز‌رسانی داده و وضعیت تکمیل جداول
	registerFourPEndpoints(mux, deps, svc)
	m.registerReadinessEndpoints(mux, deps, svc)
	// مسیرهای نظام جیب‌یست (G20) — در فایل g20.go و کامل‌اً جدا از مسیرهای ۴پی:
	//   /g20-options      بیست شاخص، چهار خانواده، جدول طبقه‌بندی و فهرست‌های مرجع خودِ G20
	//   /g20-defaults     پارامترها و آخرین امتیاز یک کارمند در نظام جیب‌یست
	//   /g20-calc         محاسبهٔ امتیاز کل و طبقه از چهار امتیاز خانواده
	//   /g20-board        طبقه‌بندی همهٔ کارکنان و طبقهٔ کل شرکت
	registerG20Endpoints(mux, deps, svc)
	deps.Logger(m.ID()).Info("ماژول جبران خدمات ۴پی ثبت شد", "routes", 11)
}
