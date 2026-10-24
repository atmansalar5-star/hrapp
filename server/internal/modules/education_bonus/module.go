// Package education_bonus ماژول پاداش تحصیلی فرزندان ممتاز در جبران خدمات.
//
// این ماژول مستقل از کارت‌های رفاهی (پتروکارت و نفت‌کارت) است: ثبت دانش‌آموزان ممتاز،
// مبالغ مصوب هر مقطع و ساخت فایل محاسباتی پرداخت برای واحد مالی را به‌تنهایی انجام
// می‌دهد.
//
// رکوردهای این ماژول (دانش‌آموزان، مبالغ مصوب و فایل‌ها) از موتور مشترک `recordkit`
// می‌آیند و قواعدشان در بستهٔ domain اعلام شده است؛ مسیرهای اختصاصی (گزینه‌ها، محاسبهٔ
// پاداش، خروجی مالی) در بستهٔ api سوار می‌شوند.
package education_bonus

import (
	"net/http"

	"hrms/internal/modules/education_bonus/api"
	"hrms/internal/modules/education_bonus/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول پاداش تحصیلی فرزندان ممتاز.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول (هم‌نام مانیفست).
func (m *Module) ID() string { return "education_bonus" }

// Register مسیرهای رکوردهای پاداش تحصیلی و مسیرهای اختصاصی محاسبه را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/education_bonus")

	h := api.New(svc, deps)
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc { return deps.Require(m.ID(), action, fn) }

	// گزینه‌ها: مقاطع با مبلغ مصوب جاری (رابط از همین می‌سازد).
	mux.HandleFunc("GET /api/v1/education_bonus/options", r("read", h.Options))
	// پیش‌نمایش محاسبهٔ پاداش: ردیف‌ها را می‌سازد و چیزی ذخیره نمی‌کند.
	mux.HandleFunc("POST /api/v1/education_bonus/bonus/preview", r("read", h.PreviewBonus))
	// ساخت فایل محاسباتی (گروهی یا انفرادی) و ثبت آن برای مالی.
	mux.HandleFunc("POST /api/v1/education_bonus/bonus-files", r("write", h.CreateBonusFile))
	// خروجی CSV همان فایل برای واحد مالی.
	mux.HandleFunc("GET /api/v1/education_bonus/bonus-files/{id}/export", r("read", h.ExportBonusFile))

	deps.Logger(m.ID()).Info("ماژول پاداش تحصیلی فرزندان ممتاز ثبت شد", "routes", 11)
}
