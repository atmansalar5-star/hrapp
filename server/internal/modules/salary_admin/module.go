// Package salary_admin ماژول احکام حقوق و گواهی اشتغال در جبران خدمات.
//
// برش عمودی کامل: domain (قواعد) → infra (حافظه یا Postgres) → app (گردش‌کار) → api.
// نام و واحد کارمند از پورت قراردادی می‌آید، نه از import مستقیم به ماژول کارکنان.
package salary_admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/salary_admin/api"
	"hrms/internal/modules/salary_admin/app"
	"hrms/internal/modules/salary_admin/domain"
	"hrms/internal/modules/salary_admin/infra"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/history"
	"hrms/internal/platform/modulekit"
)

// Module ثبت ماژول احکام حقوق و گواهی اشتغال.
type Module struct{}

// wagePercentOf ضرایب درصدی سازمانی (ثابت‌های شغلی و اقلیمی) را از پارامترهای شرکت و
// رکوردهای «پارامتر حقوق» می‌خواند و به سرویس می‌دهد.
//
// چرا اینجا و با تزریق تابع: حکم کارگزینی باید «مزد مبنا» را با همان اعدادى نشان دهد که
// فیش حقوق حساب می‌کند، ولی نباید به داخلی‌های ماژول حقوق import بزند؛ پس فقط عدد
// می‌گیرد. اگر ماژول حقوق در این استقرار نباشد، پارامترهای شرکت تنها منبع‌اند.
func wagePercentOf(deps modulekit.Deps) func(ctx context.Context) map[string]float64 {
	return func(ctx context.Context) map[string]float64 {
		out := map[string]float64{}
		for _, p := range deps.Org.Params {
			if f, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64); err == nil {
				out[p.Key] = f
			}
		}
		if provider, ok := deps.Ports.PayElements(); ok {
			for k, v := range provider.PayParamValues(ctx) {
				if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
					out[k] = f
				}
			}
		}
		return out
	}
}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول (هم‌نام مانیفست).
func (m *Module) ID() string { return "salary_admin" }

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
		log.Info("انبار احکام و گواهی‌ها روی Postgres فعال است")
		// نصب تازه روی Postgres خالی است؛ با پرچم بذرکاری، همان دادهٔ آزمایشی احکام و
		// گواهی‌ها در پایگاه‌داده ریخته می‌شود (فقط اگر جدول خالی باشد).
		if deps.SeedDemo {
			decrees, certs, err := pg.SeedIfEmpty(context.Background())
			if err != nil {
				log.Warn("بذرکاری دادهٔ نمونهٔ احکام/گواهی ناموفق بود", "error", err)
			} else if decrees+certs > 0 {
				log.Info("دادهٔ نمونهٔ احکام و گواهی‌ها در پایگاه‌داده بذرکاری شد", "decrees", decrees, "certificates", certs)
			}
		}
	} else {
		repo = infra.NewSeededStore(clock)
		log.Info("انبار احکام و گواهی‌ها در حافظه است (HRMS_DATABASE_URL تنظیم نشده)")
	}

	var employees app.EmployeeLookup
	if reader, ok := deps.Ports.Employees(); ok {
		employees = app.LookupFromReader(reader)
	}

	svc := app.New(repo, employees, deps.Audit, deps.Sync, clock, wagePercentOf(deps))
	// پورت خواندن حکم ابلاغ‌شده: مصرف‌کنندهٔ اصلی محاسبهٔ پاداش پایان خدمت است که پایه و
	// نوع استخدام را از حکم برمی‌دارد. این‌گونه آن ماژول به داخلی‌های این ماژول دست
	// نمی‌زند و مرز ماژول شکسته نمی‌شود.
	deps.Ports.SetSalaryDecreeReader(decreesAdapter{svc: svc})
	// پورت «سابقهٔ احکام»: صفحهٔ سوابق پروندهٔ کارمند، احکام قبلی را از همین‌جا می‌خواند
	// (شامل حکم‌های منسوخ) تا کاربر حکم جاری را در کنار پیشینهٔ حکم‌های کارمند ببیند.
	deps.Ports.RegisterHistorySource(decreesHistory{svc: svc, titleFa: deps.ModuleTitle(m.ID()), audit: deps.Audit})
	h := api.New(svc, deps)
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc { return deps.Require(m.ID(), action, fn) }

	mux.HandleFunc("GET /api/v1/salary_admin/decrees", r("read", h.ListDecrees))
	mux.HandleFunc("POST /api/v1/salary_admin/decrees", r("write", h.CreateDecree))
	mux.HandleFunc("GET /api/v1/salary_admin/decrees/{id}", r("read", h.GetDecree))
	mux.HandleFunc("PATCH /api/v1/salary_admin/decrees/{id}", r("write", h.UpdateDecree))
	mux.HandleFunc("DELETE /api/v1/salary_admin/decrees/{id}", r("delete", h.DeleteDecree))
	mux.HandleFunc("POST /api/v1/salary_admin/decrees/{id}/issue", r("write", h.IssueDecree))
	mux.HandleFunc("POST /api/v1/salary_admin/decrees/{id}/notify", r("approve", h.NotifyDecree))
	mux.HandleFunc("POST /api/v1/salary_admin/decrees/{id}/revise", r("write", h.ReviseDecree))
	// صدور گروهی حکم کارگزینی بر پایهٔ نوع استخدام (یا انتخاب دستی کارکنان).
	mux.HandleFunc("POST /api/v1/salary_admin/decrees/group", r("write", h.IssueGroupDecree))

	mux.HandleFunc("GET /api/v1/salary_admin/certificates", r("read", h.ListCertificates))
	mux.HandleFunc("POST /api/v1/salary_admin/certificates", r("write", h.CreateCertificate))
	mux.HandleFunc("GET /api/v1/salary_admin/certificates/{id}", r("read", h.GetCertificate))
	mux.HandleFunc("PATCH /api/v1/salary_admin/certificates/{id}", r("write", h.UpdateCertificate))
	mux.HandleFunc("DELETE /api/v1/salary_admin/certificates/{id}", r("delete", h.DeleteCertificate))
	mux.HandleFunc("POST /api/v1/salary_admin/certificates/{id}/decision", r("approve", h.DecideCertificate))
	mux.HandleFunc("POST /api/v1/salary_admin/certificates/{id}/issue", r("write", h.IssueCertificate))
	mux.HandleFunc("POST /api/v1/salary_admin/certificates/{id}/deliver", r("write", h.DeliverCertificate))

	mux.HandleFunc("GET /api/v1/salary_admin/options", r("read", h.Options))

	log.Info("ماژول احکام حقوق و گواهی اشتغال ثبت شد", "routes", 18)
}

// decreesAdapter پوشش پورت «خواندن حکم ابلاغ‌شدهٔ کارمند» روی سرویس این ماژول.
//
// آخرین حکم ابلاغ‌شده انتخاب می‌شود (نه هر حکمی): پاداش پایان خدمت بر پایهٔ آخرین حکم
// معتبر محاسبه می‌شود، نه پیش‌نویس یا حکم منسوخ.
type decreesAdapter struct{ svc *app.Service }

func (a decreesAdapter) NotifiedDecreeOf(ctx context.Context, employeeID string) (contracts.SalaryDecree, bool, error) {
	items, err := a.svc.NotifiedDecrees(ctx, employeeID)
	if err != nil || len(items) == 0 {
		return contracts.SalaryDecree{}, false, err
	}
	best := items[0]
	for _, d := range items[1:] {
		if d.EffectiveDate > best.EffectiveDate {
			best = d
		}
	}
	return best, true, nil
}

// decreesHistory منبع «سابقهٔ احکام» یک کارمند.
//
// همهٔ احکام — پیش‌نویس، ابلاغ‌شده و منسوخ — به‌عنوان سابقه برمی‌گردند. حکم منسوخ پاک
// نمی‌شود و همین رکورد «حکم قبلی» است که کاربر انتظار دیدنش را دارد؛ یادداشت حکم هم
// علت منسوخ‌شدن را نشان می‌دهد. سابقهٔ تغییرات حکم در داده‌های تصمیم‌گیرنده و رخداد
// حسابرسی ثبت می‌شود و اینجا فقط اقلام سند می‌آید.
type decreesHistory struct {
	svc     *app.Service
	titleFa string
	// audit زنجیرهٔ فقط-افزودن حسابرسی است که رخدادهای حکم (ثبت، صدور، ابلاغ،
	// منسوخ‌سازی) را با زمان و کاربر ثبت می‌کند؛ «سابقهٔ تغییرات» هر حکم از همین
	// رخدادهای واقعی ساخته می‌شود، نه از حدس. اگر لاگ در دسترس نباشد، سابقهٔ تغییرات
	// خالی می‌ماند و سطر سند سر جای خود می‌ماند.
	audit *audit.Log
}

func (h decreesHistory) Module() string { return "salary_admin" }

func (h decreesHistory) ModuleTitleFa() string { return h.titleFa }

func (h decreesHistory) HistoryOf(ctx context.Context, employeeID string) ([]contracts.HistoryEntry, error) {
	employeeID = strings.TrimSpace(employeeID)
	if employeeID == "" {
		return nil, nil
	}
	page, err := h.svc.ListDecrees(ctx, contracts.SalaryDecreeFilter{EmployeeID: employeeID, Scope: "all", Size: 500})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.HistoryEntry, 0, len(page.Items))
	for _, d := range page.Items {
		if d.EmployeeID != employeeID {
			continue
		}
		fields := []contracts.HistoryField{}
		if d.JobGroup != "" {
			fields = append(fields, contracts.HistoryField{Key: "jobGroup", LabelFa: "گروه شغلی", Value: d.JobGroup})
		}
		if d.BaseSalary > 0 {
			fields = append(fields, contracts.HistoryField{Key: "baseSalary", LabelFa: "پایهٔ حقوق (ریال)", Value: strconv.FormatInt(d.BaseSalary, 10), Type: "money"})
		}
		if d.EmploymentTypeFa != "" {
			fields = append(fields, contracts.HistoryField{Key: "employmentType", LabelFa: "نوع استخدام", Value: d.EmploymentTypeFa})
		}
		if d.ReasonFa != "" {
			fields = append(fields, contracts.HistoryField{Key: "reasonFa", LabelFa: "علت صدور", Value: d.ReasonFa})
		}
		if d.SupersedesID != "" {
			fields = append(fields, contracts.HistoryField{Key: "supersedesId", LabelFa: "جانشینِ حکم", Value: d.SupersedesID})
		}
		out = append(out, contracts.HistoryEntry{
			Module:         h.Module(),
			ModuleTitleFa:  h.titleFa,
			Kind:           "decrees",
			KindTitleFa:    "حکم حقوق",
			RecordID:       d.ID,
			Title:          "حکم حقوق " + d.ID,
			EmployeeID:     d.EmployeeID,
			EmployeeName:   d.EmployeeName,
			DepartmentName: d.DepartmentName,
			Date:           d.EffectiveDate,
			DateLabelFa:    "تاریخ اجرا",
			Status:         d.Status,
			StatusFa:       d.StatusFa,
			Fields:         fields,
			NoteFa:         d.NoteFa,
			Changes:        h.changes(d.ID),
		})
	}
	return out, nil
}

// changes سابقهٔ تغییرات حکم را از زنجیرهٔ حسابرسی می‌خواند.
func (h decreesHistory) changes(id string) []contracts.HistoryChange {
	if h.audit == nil {
		return nil
	}
	return history.FromAudit(h.audit.ByTarget(h.Module(), id), decreeActionFa)
}

// decreeActionFa برچسب فارسی اقدام‌های حکم از دامنهٔ همان ماژول؛ کلید ناشناخته بدون
// برچسب می‌ماند تا شرح فارسی رخداد خودش دیده شود.
func decreeActionFa(action string) string { return domain.DecreeActionFa[action] }
