// Package staff_services ماژول افراد تحت تکفل و درخواست پرواز.
//
// برش عمودی کامل: domain → infra (حافظه یا Postgres) → app → api. نام و واحد کارمند
// از پورت قراردادی می‌آید، نه از import مستقیم به ماژول کارکنان.
package staff_services

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/staff_services/api"
	"hrms/internal/modules/staff_services/app"
	"hrms/internal/modules/staff_services/domain"
	"hrms/internal/modules/staff_services/infra"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/history"
	"hrms/internal/platform/modulekit"
)

// Module ثبت ماژول تکفل و درخواست پرواز.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول (هم‌نام مانیفست).
func (m *Module) ID() string { return "staff_services" }

// Register انبار، سرویس و مسیرها را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	log := deps.Logger(m.ID())
	clock := func() time.Time {
		if deps.Clock == nil {
			return time.Now()
		}
		return deps.Clock()
	}

	var repo app.Repository
	if deps.DB != nil {
		pg := infra.NewPostgresStore(deps.DB, clock)
		repo = pg
		log.Info("انبار تکفل و درخواست پرواز روی Postgres فعال است")
		// نصب تازهٔ خالی: افراد تحت تکفل و درخواست‌های پرواز نمونه بذرکاری می‌شوند.
		if deps.SeedDemo {
			dependents, flights, err := pg.SeedIfEmpty(context.Background())
			if err != nil {
				log.Warn("بذرکاری دادهٔ نمونهٔ تکفل/پرواز ناموفق بود", "error", err)
			} else if dependents+flights > 0 {
				log.Info("دادهٔ نمونهٔ تکفل و درخواست پرواز بذرکاری شد", "dependents", dependents, "flights", flights)
			}
		}
	} else {
		repo = infra.NewSeededStore(clock)
		log.Info("انبار تکفل و درخواست پرواز در حافظه است (HRMS_DATABASE_URL تنظیم نشده)")
	}

	var employees app.EmployeeLookup
	if reader, ok := deps.Ports.Employees(); ok {
		employees = app.LookupFromReader(reader)
	}

	svc := app.New(repo, employees, deps.Audit, deps.Sync, clock)
	// پورت سابقه: افراد تحت تکفل و وضعیت پوشش هرکدام (شامل حذف‌شده‌ها) در صفحهٔ سوابق
	// پروندهٔ کارمند می‌آید؛ مبنای بیمهٔ درمانی کارمند در طول زمان قابل ردیابی می‌ماند.
	deps.Ports.RegisterHistorySource(dependentsHistory{svc: svc, titleFa: deps.ModuleTitle(m.ID()), audit: deps.Audit})
	h := api.New(svc, deps)
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc { return deps.Require(m.ID(), action, fn) }

	mux.HandleFunc("GET /api/v1/personnel/dependents", r("read", h.ListDependents))
	mux.HandleFunc("POST /api/v1/personnel/dependents", r("write", h.CreateDependent))
	mux.HandleFunc("GET /api/v1/personnel/dependents/{id}", r("read", h.GetDependent))
	mux.HandleFunc("PATCH /api/v1/personnel/dependents/{id}", r("write", h.UpdateDependent))
	mux.HandleFunc("POST /api/v1/personnel/dependents/{id}/decision", r("approve", h.DecideDependent))
	mux.HandleFunc("POST /api/v1/personnel/dependents/{id}/remove", r("approve", h.RemoveDependent))
	mux.HandleFunc("DELETE /api/v1/personnel/dependents/{id}", r("delete", h.DeleteDependent))

	mux.HandleFunc("GET /api/v1/personnel/flight-requests", r("read", h.ListFlights))
	mux.HandleFunc("POST /api/v1/personnel/flight-requests", r("write", h.CreateFlight))
	mux.HandleFunc("GET /api/v1/personnel/flight-requests/{id}", r("read", h.GetFlight))
	mux.HandleFunc("PATCH /api/v1/personnel/flight-requests/{id}", r("write", h.UpdateFlight))
	mux.HandleFunc("POST /api/v1/personnel/flight-requests/{id}/decision", r("approve", h.DecideFlight))
	mux.HandleFunc("GET /api/v1/personnel/options", r("read", h.Options))

	log.Info("ماژول افراد تحت تکفل و درخواست پرواز ثبت شد", "routes", 13)
}

// dependentsHistory منبع «سابقهٔ افراد تحت تکفل» یک کارمند.
//
// تکفل پایهٔ پوشش بیمهٔ درمانی است؛ پس وضعیت هر فرد (در انتظار بررسی، تأییدشده،
// حذف‌شده) و مدارک و مدت پوشش او بخشی از سابقهٔ پروندهٔ کارمند است و در گذر زمان باید
// قابل ردیابی بماند. متن مدارک از خود دادهٔ سند می‌آید، نه از این کد.
type dependentsHistory struct {
	svc     *app.Service
	titleFa string
	// audit زنجیرهٔ فقط-افزودن حسابرسی است که ثبت، ویرایش، تصمیم و خروج از تکفل را با
	// زمان و کاربر ثبت می‌کند؛ «سابقهٔ تغییرات» هر تکفل از همین رخدادهای واقعی ساخته
	// می‌شود. اگر لاگ در دسترس نباشد، سابقهٔ تغییرات خالی می‌ماند.
	audit *audit.Log
}

func (h dependentsHistory) Module() string { return "staff_services" }

// changes سابقهٔ تغییرات یک تکفل را از زنجیرهٔ حسابرسی می‌خواند.
func (h dependentsHistory) changes(id string) []contracts.HistoryChange {
	if h.audit == nil {
		return nil
	}
	return history.FromAudit(h.audit.ByTarget(h.Module(), id), func(action string) string {
		return domain.DependentActionFa[action]
	})
}

func (h dependentsHistory) ModuleTitleFa() string { return h.titleFa }

func (h dependentsHistory) HistoryOf(ctx context.Context, employeeID string) ([]contracts.HistoryEntry, error) {
	employeeID = strings.TrimSpace(employeeID)
	if employeeID == "" {
		return nil, nil
	}
	page, err := h.svc.ListDependents(ctx, contracts.DependentFilter{EmployeeID: employeeID, Scope: "all", Size: 500})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.HistoryEntry, 0, len(page.Items))
	for _, d := range page.Items {
		if d.EmployeeID != employeeID {
			continue
		}
		fields := []contracts.HistoryField{}
		if d.RelationFa != "" {
			fields = append(fields, contracts.HistoryField{Key: "relation", LabelFa: "نسبت", Value: d.RelationFa})
		}
		if d.NationalID != "" {
			fields = append(fields, contracts.HistoryField{Key: "nationalId", LabelFa: "کد ملی", Value: d.NationalID})
		}
		if d.BirthDate != "" {
			fields = append(fields, contracts.HistoryField{Key: "birthDate", LabelFa: "تاریخ تولد", Value: d.BirthDate, Type: "date"})
		}
		if d.DocumentsFa != "" {
			fields = append(fields, contracts.HistoryField{Key: "documents", LabelFa: "مدارک", Value: d.DocumentsFa})
		}
		if d.InsuranceFa != "" {
			fields = append(fields, contracts.HistoryField{Key: "insurance", LabelFa: "بیمه", Value: d.InsuranceFa})
		}
		if d.CoverageMonths > 0 {
			fields = append(fields, contracts.HistoryField{Key: "coverageMonths", LabelFa: "مدت پوشش (ماه)", Value: strconv.Itoa(d.CoverageMonths), Type: "number"})
		}
		out = append(out, contracts.HistoryEntry{
			Module:        h.Module(),
			ModuleTitleFa: h.titleFa,
			Kind:          "dependents",
			KindTitleFa:   "فرد تحت تکفل",
			RecordID:      d.ID,
			Title:         d.FullName,
			EmployeeID:    d.EmployeeID,
			EmployeeName:  d.EmployeeName,
			Date:          d.CoverageStart,
			DateLabelFa:   "تاریخ آغاز پوشش",
			Status:        d.Status,
			StatusFa:      d.StatusFa,
			Fields:        fields,
			NoteFa:        d.NoteFa,
			Changes:       h.changes(d.ID),
		})
	}
	return out, nil
}
