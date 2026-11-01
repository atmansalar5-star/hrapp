// Package cards ماژول کارت‌های رفاهی و شارژ در جبران خدمات.
//
// پتروکارت (بخش فرهنگی و رستوران) و نفت‌کارت (نفت‌مارت / فارس‌کارت / سامان‌کارت با
// چهار قلم رفاهی) از یک دامنه شارژ می‌آیند: بخش‌ها و نرخ‌های قابل بازتعریف، افراد
// مشمول و تحت تکفل با تخصیص/عدم تخصیص، مناسبت‌های تقویمی و فایل شارژ برای مالی.
//
// رکوردهای این ماژول (افراد مشمول، نرخ‌ها، مناسبت‌ها، شارژها و فایل‌ها) از موتور
// مشترک `recordkit` می‌آیند و قواعدشان در بستهٔ domain اعلام شده است؛ مسیرهای
// اختصاصی (گزینه‌ها، ساخت شارژ، خروجی مالی) در بستهٔ api سوار می‌شوند.
package cards

import (
	"net/http"

	"hrms/internal/modules/cards/api"
	"hrms/internal/modules/cards/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول کارت‌های رفاهی و شارژ.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول (هم‌نام مانیفست).
func (m *Module) ID() string { return "cards" }

// Register مسیرهای رکوردهای کارت و مسیرهای اختصاصی شارژ را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/cards")

	h := api.New(svc, deps)
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc { return deps.Require(m.ID(), action, fn) }

	// گزینه‌ها: کارت‌ها، بخش‌ها با نرخ جاری و مناسبت‌های فعال (رابط از همین می‌سازد).
	mux.HandleFunc("GET /api/v1/cards/options", r("read", h.Options))
	// پیش‌نمایش شارژ: ردیف‌ها را می‌سازد و چیزی ذخیره نمی‌کند.
	mux.HandleFunc("POST /api/v1/cards/charge/preview", r("read", h.PreviewCharge))
	// ساخت فایل شارژ (گروهی یا انفرادی) و ثبت آن برای مالی.
	mux.HandleFunc("POST /api/v1/cards/charge-files", r("write", h.CreateChargeFile))
	// خروجی CSV همان فایل برای واحد مالی.
	mux.HandleFunc("GET /api/v1/cards/charge-files/{id}/export", r("read", h.ExportChargeFile))
	mux.HandleFunc("GET /api/v1/cards/charge-files/{id}/xlsx", r("read", h.ExportChargeFileXLSX))

	deps.Logger(m.ID()).Info("ماژول کارت‌های رفاهی و شارژ ثبت شد", "routes", 11)
}
