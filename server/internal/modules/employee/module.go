// Package employee ماژول مدیریت کارکنان.
//
// این ماژول در برش عمودی فاز ۱ کامل پیاده شده است و الگوی بقیهٔ ماژول‌ها است:
// domain (قاعده) → infra (ذخیره‌سازی) → app (کاربرد) → api (HTTP) و در پایان
// module.go که فقط سیم‌کشی می‌کند. هیچ ماژول دیگری نمی‌تواند به این پکیج‌ها import
// بزند؛ مصرف داده‌های کارکنان تنها از راه port در بستهٔ contracts ممکن است.
package employee

import (
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

	repo := infra.NewSeededStore()
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
