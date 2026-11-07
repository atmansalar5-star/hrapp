// Package employee ماژول مدیریت کارکنان.
//
// این ماژول در برش عمودی فاز ۱ کامل پیاده شده است و الگوی بقیهٔ ماژول‌ها است:
// domain (قاعده) → infra (ذخیره‌سازی) → app (کاربرد) → api (HTTP) و در پایان
// module.go که فقط سیم‌کشی می‌کند. هیچ ماژول دیگری نمی‌تواند به این پکیج‌ها import
// بزند؛ مصرف داده‌های کارکنان تنها از راه port در بستهٔ contracts ممکن است.
package employee

import (
	"context"
	"net/http"

	"hrms/internal/modules/employee/api"
	"hrms/internal/modules/employee/app"
	"hrms/internal/modules/employee/infra"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/photos"
)

// Module ثبت ماژول کارکنان.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول (هم‌نام مانیفست).
func (m *Module) ID() string { return "employee" }

// Register مسیرها و پورت‌های ماژول را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	log := deps.Logger(m.ID())

	// انتخاب انبار: روی استقرار واقعی (اتصال پایگاه‌داده) کارکنان در
	// `hr_employee.employee` ذخیره می‌شوند و ویرایش اطلاعات پایه پس از راه‌اندازی
	// دوبارهٔ سرور می‌ماند؛ در اجرای بدون پایگاه‌داده (توسعه/مستقل) همان انبار
	// درون‌حافظه‌ای با دادهٔ نمونه بالا می‌آید. هیچ لایهٔ بالاتر این تفاوت را نمی‌بیند،
	// چون هر دو همان قرارداد app.Repository را برمی‌آورند.
	var repo app.Repository = infra.NewSeededStore()
	if deps.DB != nil {
		store := infra.NewPgStore(deps.DB, nil)
		departments, employees, err := store.SeedIfEmpty(context.Background())
		if err != nil {
			// شکست بذرکاری نباید سامانه را بخواباند: انبار Postgres در هر حال
			// کار می‌کند و بذرکاری اجرای بعدی تکرار می‌شود.
			log.Warn("بذرکاری کارکنان در پایگاه‌داده انجام نشد", "error", err.Error())
		} else if employees > 0 {
			log.Info("دادهٔ نمونهٔ کارکنان در پایگاه‌داده بذرکاری شد",
				"employees", employees, "departments", departments)
		}
		repo = store
		log.Info("انبار کارکنان روی پایگاه‌داده فعال است", "schema", "hr_employee")
	}

	svc := app.New(repo, deps.Audit, deps.Sync)

	// انبار «عکس کارگزینی»: فایل‌ها در پوشهٔ دادهٔ سرور می‌مانند (نه در درخت کد) و
	// نگاشت کارمند→عکس هم روی دیسک نگه داشته می‌شود، پس با راه‌اندازی دوبارهٔ سرور
	// عکس‌ها سرِ جایشان می‌مانند. اگر پوشهٔ داده تنظیم نشده باشد، فقط مسیرهای عکس با
	// پیام روشن پاسخ می‌دهند و بقیهٔ ماژول دست‌نخورده کار می‌کند.
	if store, err := photos.New(deps.DataDir); err != nil {
		log.Warn("انبار عکس کارگزینی آماده نشد", "error", err.Error())
	} else {
		svc = svc.WithPhotos(store, deps.VerifyKey)
		log.Info("انبار عکس کارگزینی آماده شد", "dir", store.Dir())
	}

	// ثبت پورت: تنها راه ماژول‌های دیگر برای خواندن داده کارکنان.
	deps.Ports.SetEmployeeReader(svc)

	// پورت نوشتن باریک: فقط ماژول انتصابات، پس از تصویب ابلاغ، پست و واحد کارمند را
	// از همین راه عوض می‌کند؛ قاعده‌های پرونده (اعتبارسنجی، حسابرسی، سینک) همین‌جا می‌مانند.
	deps.Ports.SetEmployeeAssigner(svc)

	h := api.New(svc, deps)

	// مسیرها مطابق contracts/openapi/hrms.json و پیشوند مانیفست ماژول.
	mux.HandleFunc("GET /api/v1/employee/employees", deps.Require(m.ID(), "read", h.List))
	mux.HandleFunc("POST /api/v1/employee/employees", deps.Require(m.ID(), "write", h.Create))
	mux.HandleFunc("GET /api/v1/employee/employees/{id}", deps.Require(m.ID(), "read", h.Get))
	mux.HandleFunc("PATCH /api/v1/employee/employees/{id}", deps.Require(m.ID(), "write", h.Update))
	mux.HandleFunc("DELETE /api/v1/employee/employees/{id}", deps.Require(m.ID(), "delete", h.Delete))
	mux.HandleFunc("GET /api/v1/employee/departments", deps.Require(m.ID(), "read", h.Departments))
	mux.HandleFunc("GET /api/v1/employee/options", deps.Require(m.ID(), "read", h.Options))
	// عکس کارگزینی: خواندن با دسترسی خواندن پرونده، بارگذاری/حذف با دسترسی نوشتن.
	mux.HandleFunc("GET /api/v1/employee/employees/{id}/photo", deps.Require(m.ID(), "read", h.GetPhoto))
	mux.HandleFunc("POST /api/v1/employee/employees/{id}/photo", deps.Require(m.ID(), "write", h.UploadPhoto))
	mux.HandleFunc("DELETE /api/v1/employee/employees/{id}/photo", deps.Require(m.ID(), "write", h.DeletePhoto))

	log.Info("ماژول کارکنان ثبت شد",
		"employees", len(infra.SeedEmployees()),
		"departments", len(infra.Departments()),
	)
}
