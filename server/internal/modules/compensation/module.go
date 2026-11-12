// Package compensation ماژول «جبران خدمات ۴پی».
//
// پیاده‌سازی روی موتور مشترک `recordkit` است: مدل جبران خدمات، قیمت‌گذاری شغل، سطح
// شایستگی، ضریب عملکرد، باند پرداخت و تصمیم چهارعاملی هر یک یک نوع رکورد با گردشکار
// خودش‌اند و قواعدشان در بستهٔ domain اعلام شده است. قاعدهٔ محاسبهٔ چهارعاملی در موتور
// قواعد (`internal/rules/compensation.go`) است و این لایه فقط داده را می‌خواند، به
// موتور می‌دهد و نتیجه را برمی‌گرداند؛ پس هیچ عددی در رابط کاربری نیست.
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

// Register مسیرهای رکوردهای جبران خدمات و مسیرهای اختصاصی ۴پی را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/compensation")
	// سه مسیر اختصاصی این ماژول:
	//   /options          فهرست‌های مرجع (خانوادهٔ شغلی، سطوح شایستگی و عملکرد)
	//   /4p-defaults      پارامترها، باندها، ضرایب و داده‌های پیش‌فرض یک کارمند
	//   /4p-calc          محاسبهٔ خط‌به‌خط بستهٔ جبران خدمات و ثبت تصمیم
	//   /data-readiness   راهنمای شروع به‌روزرسانی داده و وضعیت تکمیل جداول
	registerFourPEndpoints(mux, deps, svc)
	m.registerReadinessEndpoints(mux, deps, svc)
	deps.Logger(m.ID()).Info("ماژول جبران خدمات ۴پی ثبت شد", "routes", 7)
}
