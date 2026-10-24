// Package contracts ماژول کارگزینی و انعقاد قراردادها.
//
// برش عمودی کامل: domain (قواعد) → infra (حافظه یا Postgres) → app (گردش‌کار) →
// api (HTTP). این ماژول نام کارمند را از پورت قراردادی می‌گیرد و هرگز به داخلی‌های
// ماژول کارکنان import نمی‌زند؛ همین قاعده در CI بررسی می‌شود.
package contracts

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/contracts/api"
	"hrms/internal/modules/contracts/app"
	"hrms/internal/modules/contracts/infra"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/modulekit"
)

// Module ثبت ماژول قراردادها.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول (هم‌نام مانیفست).
func (m *Module) ID() string { return "contracts" }

// Register انبار، سرویس، پورت اسناد و مسیرها را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	log := deps.Logger(m.ID())
	clock := func() time.Time {
		if deps.Clock == nil {
			return time.Now()
		}
		return deps.Clock()
	}

	// انتخاب انبار: در استقرار واقعی Postgres، در اجرای توسعه و آزمون حافظه.
	var repo app.Repository
	if deps.DB != nil {
		pg := infra.NewPostgresStore(deps.DB, clock)
		repo = pg
		log.Info("انبار قراردادها روی Postgres فعال است")
		// نصب تازهٔ خالی: قراردادهای نمونهٔ آزمایشی در پایگاه‌داده بذرکاری می‌شوند تا
		// صفحهٔ قراردادها و جریان‌های وابسته (فیش، پرونده، پاداش) کارکردی باشند.
		// بذرکاری افزایشی است: شناسه‌های غایب افزوده می‌شوند و قرارداد واقعی سازمان
		// دست‌نخورده می‌ماند (SeedMissing).
		if deps.SeedDemo {
			if n, err := pg.SeedMissing(context.Background()); err != nil {
				log.Warn("بذرکاری دادهٔ نمونهٔ قراردادها ناموفق بود", "error", err)
			} else if n > 0 {
				log.Info("دادهٔ نمونهٔ قراردادها در پایگاه‌داده بذرکاری شد", "contracts", n)
			}
		}
	} else {
		repo = infra.NewSeededStore(clock)
		log.Info("انبار قراردادها در حافظه است (HRMS_DATABASE_URL تنظیم نشده)")
	}

	var employees app.EmployeeLookup
	if reader, ok := deps.Ports.Employees(); ok {
		employees = app.LookupFromReader(reader)
	}

	svc := app.New(repo, employees, deps.Audit, deps.Sync, clock)
	// پورت اسناد چاپی: لایهٔ پلتفرم برای ساخت قرارداد و حکم حقوق از همین‌جا داده می‌گیرد.
	deps.Ports.SetContractDocuments(svc)
	// پورت سابقه: «قراردادهای قبلی» کارمند (تمدیدشده، منسوخ و پایان‌یافته) در صفحهٔ
	// سوابق می‌آید تا قرارداد جاری در کنار پیشینهٔ قراردادی کارمند دیده شود.
	deps.Ports.RegisterHistorySource(contractHistory{svc: svc, titleFa: deps.ModuleTitle(m.ID())})

	h := api.New(svc, deps)
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc { return deps.Require(m.ID(), action, fn) }

	mux.HandleFunc("GET /api/v1/contracts/contracts", r("read", h.List))
	mux.HandleFunc("POST /api/v1/contracts/contracts", r("write", h.Create))
	mux.HandleFunc("GET /api/v1/contracts/contracts/{id}", r("read", h.Get))
	mux.HandleFunc("PATCH /api/v1/contracts/contracts/{id}", r("write", h.Update))
	mux.HandleFunc("DELETE /api/v1/contracts/contracts/{id}", r("delete", h.Delete))
	mux.HandleFunc("POST /api/v1/contracts/contracts/{id}/submit", r("write", h.Submit))
	mux.HandleFunc("POST /api/v1/contracts/contracts/{id}/decision", r("approve", h.Decision))
	mux.HandleFunc("POST /api/v1/contracts/contracts/{id}/terminate", r("approve", h.Terminate))
	mux.HandleFunc("POST /api/v1/contracts/contracts/{id}/renew", r("write", h.Renew))
	mux.HandleFunc("POST /api/v1/contracts/contracts/{id}/convert", r("approve", h.Convert))
	mux.HandleFunc("GET /api/v1/contracts/renewals", r("read", h.Renewals))
	mux.HandleFunc("GET /api/v1/contracts/options", r("read", h.Options))

	// قالب‌های متن قرارداد از پورت قالب‌های اسناد خوانده می‌شوند؛ متن قرارداد قابل طراحی
	// است و در صفحهٔ «قالب‌های متن اسناد» ویرایش می‌شود.
	mux.HandleFunc("GET /api/v1/contracts/templates", r("read", h.Templates))

	log.Info("ماژول کارگزینی و قراردادها ثبت شد", "routes", 12)
}

// contractHistory منبع «سابقهٔ قراردادها» یک کارمند.
//
// هر تمدید یا تبدیل، قرارداد تازه می‌سازد و قرارداد پیشین را منسوخ می‌کند؛ پس همهٔ
// قراردادها — پیش‌نویس، جاری و منسوخ — سابقه‌اند و هیچ‌کدام بازنویسی نمی‌شود. سابقهٔ
// تغییرات خودِ قرارداد (انعقاد، تأیید، تمدید، خاتمه) از رخدادهای همان سند می‌آید.
type contractHistory struct {
	svc     *app.Service
	titleFa string
}

func (h contractHistory) Module() string { return "contracts" }

func (h contractHistory) ModuleTitleFa() string { return h.titleFa }

func (h contractHistory) HistoryOf(ctx context.Context, employeeID string) ([]contracts.HistoryEntry, error) {
	employeeID = strings.TrimSpace(employeeID)
	if employeeID == "" {
		return nil, nil
	}
	page, err := h.svc.List(ctx, contracts.ContractFilter{EmployeeID: employeeID, Scope: "all", Size: 500})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.HistoryEntry, 0, len(page.Items))
	for _, c := range page.Items {
		if c.EmployeeID != employeeID {
			continue
		}
		fields := []contracts.HistoryField{}
		if c.MonthlySalary > 0 {
			fields = append(fields, contracts.HistoryField{Key: "monthlySalary", LabelFa: "حقوق ماهانه (ریال)", Value: strconv.FormatInt(c.MonthlySalary, 10), Type: "money"})
		}
		if c.EndDate != "" {
			fields = append(fields, contracts.HistoryField{Key: "endDate", LabelFa: "تاریخ پایان", Value: c.EndDate, Type: "date"})
		}
		if c.EmploymentTypeFa != "" {
			fields = append(fields, contracts.HistoryField{Key: "employmentType", LabelFa: "نوع استخدام", Value: c.EmploymentTypeFa})
		}
		if c.Position != "" {
			fields = append(fields, contracts.HistoryField{Key: "position", LabelFa: "پست سازمانی", Value: c.Position})
		}
		if c.ServiceDays > 0 {
			fields = append(fields, contracts.HistoryField{Key: "serviceDays", LabelFa: "سابقهٔ خدمت (روز)", Value: strconv.Itoa(c.ServiceDays), Type: "number"})
		}
		if c.SupersedesID != "" {
			fields = append(fields, contracts.HistoryField{Key: "supersedesId", LabelFa: "جانشینِ قرارداد", Value: c.SupersedesID})
		}
		changes := make([]contracts.HistoryChange, 0, len(c.Events))
		for _, e := range c.Events {
			changes = append(changes, contracts.HistoryChange{
				At:       calendar.FormatStamp(e.At),
				Actor:    e.Actor,
				Action:   e.Action,
				ActionFa: e.ActionFa,
				NoteFa:   e.NoteFa,
			})
		}
		out = append(out, contracts.HistoryEntry{
			Module:         h.Module(),
			ModuleTitleFa:  h.titleFa,
			Kind:           c.Kind,
			KindTitleFa:    c.KindFa,
			RecordID:       c.ID,
			Title:          "قرارداد " + c.ID,
			EmployeeID:     c.EmployeeID,
			EmployeeName:   c.EmployeeName,
			DepartmentName: c.DepartmentName,
			Date:           c.StartDate,
			DateLabelFa:    "تاریخ شروع",
			Status:         c.Status,
			StatusFa:       c.StatusFa,
			Fields:         fields,
			Changes:        changes,
			NoteFa:         c.NoteFa,
		})
	}
	return out, nil
}
