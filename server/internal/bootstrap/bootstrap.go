// Package bootstrap سیم‌کشی سامانه: ساخت وابستگی‌ها، ثبت ماژول‌ها و ساخت هندلر ریشه.
//
// این تنها جایی است که نام همهٔ ماژول‌ها کنار هم دیده می‌شود. برای افزودن ماژول تازه
// کافی است پوشه و مانیفست آن ساخته و یک سطر به `Modules()` اضافه شود؛ نه مسیری در
// کلاینت دستی اضافه می‌شود و نه دسترسی‌ای دستی تعریف.
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/archive"
	"hrms/internal/modules/attendance"
	"hrms/internal/modules/bonus"
	"hrms/internal/modules/cards"
	"hrms/internal/modules/career"
	"hrms/internal/modules/compensation"
	// نام پکیج قراردادهای DTO «contracts» است، پس ماژول قراردادها با نام مستعار می‌آید.
	contractsmodule "hrms/internal/modules/contracts"
	"hrms/internal/modules/dashboard"
	"hrms/internal/modules/discipline"
	"hrms/internal/modules/education_bonus"
	"hrms/internal/modules/employee"
	"hrms/internal/modules/grievances"
	"hrms/internal/modules/hrstudies"
	"hrms/internal/modules/jobclass"
	"hrms/internal/modules/loans_retirement"
	"hrms/internal/modules/medical"
	"hrms/internal/modules/orgstructure"
	"hrms/internal/modules/outsourcing"
	"hrms/internal/modules/payroll"
	"hrms/internal/modules/performance"
	"hrms/internal/modules/recruitment"
	"hrms/internal/modules/reports"
	"hrms/internal/modules/salary_admin"
	"hrms/internal/modules/settings"
	"hrms/internal/modules/socialwork"
	"hrms/internal/modules/sports"
	"hrms/internal/modules/staff_services"
	"hrms/internal/modules/staffing_moves"
	"hrms/internal/modules/statistics"
	"hrms/internal/modules/training"
	"hrms/internal/modules/welfare"
	"hrms/internal/platform/access"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	platformdb "hrms/internal/platform/db"
	"hrms/internal/platform/documents"
	"hrms/internal/platform/history"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	platformmodules "hrms/internal/platform/modules"
	"hrms/internal/platform/orgs"
	"hrms/internal/platform/rbac"
	"hrms/internal/platform/synclog"
	"hrms/internal/platform/webstatic"
	"hrms/internal/rules"
)

// historyEmployeeAllowed آیا نقش کاربر مجاز به دیدن سابقهٔ این کارمند است.
//
// دامنه از همان مانیفست ماژول کارکنان می‌آید (`employee.scopes`)، پس قاعدهٔ دسترسی
// دوباره‌نویسی نمی‌شود: نقش «خود» فقط سابقهٔ خودش را می‌بیند، نقش «واحد خود» کارکنان
// واحد خود را، و دامنهٔ سراسری همه را.
func historyEmployeeAllowed(ctx context.Context, deps modulekit.Deps, user contracts.User, employeeID string) bool {
	switch deps.Policy.Scope(user.Role, "employee") {
	case "own":
		return user.EmployeeID != "" && employeeID == user.EmployeeID
	case "own_unit":
		if user.DepartmentID == "" {
			return true
		}
		reader, ok := deps.Ports.Employees()
		if !ok {
			return false
		}
		emp, found, err := reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
		if err != nil || !found {
			return false
		}
		return emp.DepartmentID == user.DepartmentID || emp.DepartmentName == user.DepartmentID
	default:
		return true
	}
}

// historyEmployeeName نام کارمند برای سرصفحهٔ سوابق؛ نبودن کارمند خطا نیست و رشتهٔ
// خالی برمی‌گرداند (رابط کاربری خودش «—» نشان می‌دهد).
func historyEmployeeName(ctx context.Context, deps modulekit.Deps, employeeID string) string {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return ""
	}
	emp, found, err := reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
	if err != nil || !found {
		return ""
	}
	return emp.FullName
}

// historyWarningsFa هشدار «سابقهٔ این ماژول خوانده نشد» را به فارسی می‌سازد.
func historyWarningsFa(modules []string) []string {
	if len(modules) == 0 {
		return nil
	}
	out := make([]string, 0, len(modules))
	for _, m := range modules {
		out = append(out, "سابقهٔ «‌"+m+"» خوانده نشد؛ بقیهٔ سابقه‌ها کامل است.")
	}
	return out
}

// Config پیکربندی راه‌اندازی.
type Config struct {
	// Modules فهرست ماژول‌های فعال با کاما؛ خالی یعنی از استقرار شرکت فعال گرفته شود.
	Modules string
	// WebDir پوشهٔ فایل‌های رابط وب؛ خالی یعنی سرور فقط API را بالا می‌آورد.
	WebDir string
	// JWTSecret کلید امضای توکن؛ در استقرار واقعی از متغیر محیطی خوانده می‌شود.
	JWTSecret []byte
	Logger    *slog.Logger
	Clock     func() time.Time
	// Org شناسهٔ شرکت (استقرار پارامتریک)؛ خالی یعنی استقرار پیش‌فرض.
	Org string
	// DatabaseURL نشانی Postgres؛ خالی یعنی اجرای درون‌حافظه‌ای (توسعه و آزمون).
	DatabaseURL string
	// ServerRoot ریشهٔ پوشهٔ سرور برای یافتن migrations؛ خالی یعنی پوشهٔ جاری.
	ServerRoot string
	// DataDir پوشهٔ دادهٔ سرور برای فایل‌های بارگذاری‌شده (نشان شرکت)؛ خالی یعنی «data»
	// کنار ریشهٔ سرور.
	DataDir string
	// Migrate اجرای مهاجرت‌های اعمال‌نشده در زمان راه‌اندازی (پیش‌فرض: اگر پایگاه‌داده تنظیم شده باشد).
	Migrate bool
	// SeedDemo بذرکاری دادهٔ نمونه روی پایگاه‌دادهٔ واقعی (نصب تازهٔ خالی).
	//
	// بی این پرچم، نصب تازهٔ Postgres همهٔ ماژول‌ها را خالی نشان می‌دهد؛ با آن، همان
	// دادهٔ آزمایشی احکام، فیش/دوره‌های حقوق، قراردادها و بقیهٔ ماژول‌ها در
	// پایگاه‌داده ساخته می‌شود — فقط برای جدول‌های خالی.
	SeedDemo bool
}

// Modules همهٔ ماژول‌های شناخته‌شده به ترتیب منو: حوزه به حوزه، اداره به اداره،
// زیرواحد به زیرواحد و ماژول به ماژول — همان ترتیبی که در منو دیده می‌شود.// dataDirOf پوشهٔ دادهٔ سرور را مطلق می‌کند: مقدار صریح، وگرنه «data» کنار ریشهٔ سرور.
func dataDirOf(cfg Config) string {
	dir := strings.TrimSpace(cfg.DataDir)
	if dir == "" {
		root := strings.TrimSpace(cfg.ServerRoot)
		if root == "" {
			root = "."
		}
		dir = filepath.Join(root, "data")
	}
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

// Modules همهٔ ماژول‌های شناخته‌شده به ترتیب منو: حوزه به حوزه، اداره به اداره،
// زیرواحد به زیرواحد و ماژول به ماژول — همان ترتیبی که در منو دیده می‌شود.
func Modules() []modulekit.Module {
	return []modulekit.Module{
		// حوزهٔ ستادی سازمان
		dashboard.New(),
		// واحد امور کارکنان — کارگزینی و خدمات پرسنلی
		employee.New(),
		contractsmodule.New(),
		staff_services.New(),
		// واحد امور کارکنان — بایگانی، طبقه‌بندی مشاغل و جبران خدمات
		archive.New(),
		jobclass.New(),
		attendance.New(),
		payroll.New(),
		salary_admin.New(),
		// نظام جبران خدمات ۴پی: مدل، قیمت‌گذاری شغل، شایستگی، عملکرد، باند و تصمیم
		compensation.New(),
		welfare.New(),
		cards.New(),
		education_bonus.New(),
		bonus.New(),
		loans_retirement.New(),
		// واحد برنامه‌ریزی نیروی انسانی
		orgstructure.New(),
		career.New(),
		recruitment.New(),
		performance.New(),
		// واحد روابط کار و مددکاری اجتماعی
		discipline.New(),
		grievances.New(),
		socialwork.New(),
		medical.New(),
		// واحد هماهنگی آموزش
		training.New(),
		// واحد آمار و گزارشات
		statistics.New(),
		staffing_moves.New(),
		reports.New(),
		// واحد برون‌سپاری
		outsourcing.New(),
		// واحد مطالعات منابع انسانی
		hrstudies.New(),
		// تنظیمات سامانه
		settings.New(),
		// واحد امور ورزش
		sports.New(),
	}
}

// EnabledFlags پرچم‌های فعال‌سازی هر ماژول (nil یعنی همه فعال).
func EnabledFlags(list string) map[string]bool {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil
	}
	out := map[string]bool{}
	for _, id := range strings.Split(list, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out[id] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NewDeps وابستگی‌های مشترک سامانه را می‌سازد.
func NewDeps(cfg Config) (modulekit.Deps, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	if len(cfg.JWTSecret) < 16 {
		return modulekit.Deps{}, errors.New("کلید امضای توکن باید حداقل ۱۶ بایت باشد")
	}

	// استقرار پارامتریک: شرکت فعال تعیین می‌کند کدام ماژول‌ها ثبت شوند و پارامترهای
	// حقوقی-مالی چه مقادیری داشته باشند. فهرست صریح `-modules` بر شرکت مقدم است تا
	// بتوان برای آزمون یا اشکال‌زدایی زیرمجموعهٔ ماژول‌ها را بالا آورد.
	org, ok := orgs.Lookup(cfg.Org)
	if !ok {
		return modulekit.Deps{}, orgs.UnknownError(cfg.Org)
	}
	enabled := EnabledFlags(cfg.Modules)
	source := "فهرست صریح ماژول‌ها"
	if enabled == nil {
		enabled = orgs.EnabledFlags(org)
		source = "استقرار شرکت"
	}
	logger.Info("استقرار پارامتریک", "org", org.ID, "company", org.TitleFa, "holding", org.HoldingFa,
		"modules", len(org.EnabledModules), "policy", source)

	registry := platformmodules.NewRegistry(enabled)

	accounts, err := auth.SeedAccounts()
	if err != nil {
		return modulekit.Deps{}, fmt.Errorf("ساخت حساب‌های نمونه ناموفق: %w", err)
	}

	params, err := json.Marshal(rules.DefaultPayrollParams())
	if err != nil {
		return modulekit.Deps{}, fmt.Errorf("سریال‌سازی پارامترهای پیش‌فرض ناموفق: %w", err)
	}

	// یک لاگ حسابرسی مشترک برای کل سامانه: رد شدن دسترسی هم باید در همان زنجیره‌ای
	// ثبت شود که صفحهٔ «لاگ حسابرسی» و بازبینی خاموشی سرور می‌بینند؛ اگر هر مصرف‌کننده
	// لاگ جداگانه بسازد، رد دسترسی‌ها از دید بازرس پنهان می‌ماند.
	auditLog := audit.New()

	// سطوح دسترسی افزودهٔ مدیر سیستم (نقش × اداره × ماژول) روی ماتریس مانیفست.
	accessStore := access.NewStore()

	// مجری سیاست با دامنهٔ فعال (شرکت/هلدینگ) ساخته می‌شود: سطح دسترسی شركت‌محور فقط
	// در همان شرکت و سطح هلدینگ‌محور فقط در شرکت‌های همان هلدینگ اعمال می‌شود.
	policy := rbac.New(registry, auditLog, accessStore).WithTenant(org.ID, orgs.HoldingKey(org))

	deps := modulekit.Deps{
		Log:      logger,
		Registry: registry,
		Auth:     auth.NewService(cfg.JWTSecret, accounts).WithClock(clock),
		Policy:   policy,
		Audit:    auditLog,
		Sync:     synclog.New(),
		Calendar: calendar.DefaultWorkCalendar(),
		Params:   params,
		WebDir:   cfg.WebDir,
		Access:   accessStore,
		Ports:    modulekit.NewPorts(),
		Clock:    clock,
		Org:      org,
		SeedDemo: cfg.SeedDemo,
		// پوشهٔ داده (:نشان شرکت و فایل‌های بارگذاری‌شده) و کلید امضای نشانی تأیید سند.
		// مسیر مطلق می‌شود تا مستقل از پوشهٔ جاری فرمان، همه‌جا یکی باشد.
		DataDir:   dataDirOf(cfg),
		VerifyKey: cfg.JWTSecret,
	}

	// پایگاه‌داده و مهاجرت: تنها جایی که سامانه به Postgres وصل می‌شود. اگر نشانی
	// تنظیم نشده باشد، همه‌چیز در حافظه می‌ماند تا اجرای توسعه و آزمون بدون هیچ
	// سرویس بیرونی کار کند.
	if dsn := strings.TrimSpace(cfg.DatabaseURL); dsn != "" {
		sqlDB, err := platformdb.Open(context.Background(), dsn)
		if err != nil {
			return modulekit.Deps{}, err
		}
		deps.DB = sqlDB
		logger.Info("پایگاه‌دادهٔ Postgres متصل شد", "driver", platformdb.Driver)

		if cfg.Migrate {
			root := cfg.ServerRoot
			if root == "" {
				root = "."
			}
			migrations, err := platformdb.Discover(root)
			if err != nil {
				return modulekit.Deps{}, fmt.Errorf("خواندن فایل‌های مهاجرت ناموفق: %w", err)
			}
			applied, err := platformdb.Migrate(context.Background(), sqlDB, migrations, logger)
			if err != nil {
				return modulekit.Deps{}, err
			}
			logger.Info("مهاجرت‌ها بررسی شد", "found", len(migrations), "applied", applied)
		}
	}

	return deps, nil
}

// Wire ماژول‌های فعال را روی mux سوار می‌کند و فهرست ثبت‌شده‌ها را برمی‌گرداند.
func Wire(mux *http.ServeMux, deps modulekit.Deps) []string {
	registered := make([]string, 0, len(Modules()))
	for _, m := range Modules() {
		if !deps.Registry.IsEnabled(m.ID()) {
			deps.Logger("bootstrap").Info("ماژول در این استقرار غیرفعال است", "module", m.ID())
			continue
		}
		m.Register(mux, deps)
		registered = append(registered, m.ID())
	}
	return registered
}

// NewHandler هندلر ریشه را می‌سازد: مسیرهای پلتفرم + همهٔ ماژول‌ها + میان‌افزار نشست.
func NewHandler(deps modulekit.Deps) (http.Handler, []string, error) {
	mux := http.NewServeMux()

	// مسیرهای پلتفرم (بیرون از ماژول‌ها، چون به خود سامانه مربوط‌اند نه به یک دامنه).
	registerPlatformRoutes(mux, deps)
	// نشان شرکت، QR کد و صفحهٔ تأیید سند (بخشی عمومی و بخشی نیازمند ورود).
	registerPublicRoutes(mux, deps)

	registered := Wire(mux, deps)

	// رابط وب روی ریشه: در اجرای محلی، خود رابط و API روی یک مبدأ واحد بالا می‌آیند
	// تا احراز هویت و CORS تنظیم اضافه نخواهد. مسیرهای دقیق‌تر (‎/api/... و /healthz‎)
	// همیشه بر این الگوی عمومی مقدم‌اند.
	if webstatic.Exists(deps.WebDir) {
		mux.Handle("/", webstatic.Handler(deps.WebDir, deps.Log))
		deps.Logger("bootstrap").Info("رابط وب روی مسیر ریشه سرو می‌شود", "dir", deps.WebDir)
	}

	return deps.Auth.Middleware(mux), registered, nil
}

func registerPlatformRoutes(mux *http.ServeMux, deps modulekit.Deps) {
	// بررسی سلامت — بدون احراز هویت.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"modules": deps.Registry.Count(),
			// شرکت فعال بخشی از شناسهٔ سلامت است: استقرار اشتباه باید از همان
			// نخستین درخواست دیده شود، نه وقتی کاربر وارد شد.
			"org":         deps.Org.ID,
			"orgTitleFa":  deps.Org.TitleFa,
			"holdingFa":   deps.Org.HoldingFa,
			"departments": len(deps.Org.EnabledDepartments),
		})
	})

	// بررسی آمادگی سرویس — برای probe سامانهٔ استقرار (Kubernetes، بارگذار، پایش).
	// تفاوت با /healthz این است: /healthz می‌گوید سرور بالا است، /readyz می‌گوید
	// پایگاه‌داده پاسخ می‌دهد، مهاجرت‌ها اعمال شده‌اند و زنجیرهٔ حسابرسی سالم است.
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		status := "ok"
		payload := map[string]any{
			"status":     status,
			"org":        deps.Org.ID,
			"modules":    deps.Registry.Count(),
			"auditChain": "valid",
			"database":   "in-memory",
		}
		if deps.DB != nil {
			if err := deps.DB.PingContext(ctx); err != nil {
				status = "degraded"
				payload["database"] = "unreachable"
				payload["databaseError"] = err.Error()
			} else {
				payload["database"] = "connected"
				if n, err := platformdb.Applied(ctx, deps.DB); err == nil {
					payload["migrationsApplied"] = n
				} else {
					status = "degraded"
					payload["migrationsApplied"] = "unreadable"
				}
			}
		}
		if err := deps.Audit.Verify(); err != nil {
			status = "degraded"
			payload["auditChain"] = "invalid"
		}
		payload["status"] = status
		code := http.StatusOK
		if status != "ok" {
			code = http.StatusServiceUnavailable
		}
		httpx.WriteJSON(w, code, payload)
	})

	// اسناد چاپی: قرارداد کار، حکم حقوق و گواهی اشتغال به‌صورت HTML راست‌چین با
	// شیوه‌نامهٔ چاپ A4. دسترسی هر سند روی ماژول مالک دادهٔ آن سنجیده می‌شود.
	mux.HandleFunc("GET /api/v1/documents/{kind}/{id}", auth.RequireAuth(documents.Handler(deps)))
	// پیش‌نمایش قالب سند با دادهٔ نمونه (صفحهٔ قالب‌های متن اسناد).
	mux.HandleFunc("GET /api/v1/documents/preview/{kind}", auth.RequireAuth(documents.PreviewHandler(deps)))

	// شرکت‌های قابل شبیه‌سازی (استقرار پارامتریک).
	//
	// دیدِ شرکت‌ها از آموزش استقرار و پشتیبانی لازم است: با همین پاسخ می‌توان دید یک
	// استقرار برای چه شرکتی بالا آمده، چه اداره‌هایی فعال‌اند و پارامترهای حقوقی-مالی
	// هر شرکت چیست. تغییر شرکت زمان اجرا انجام نمی‌شود؛ چون ماژول‌ها در زمان Register
	// روی mux سوار می‌شوند و تغییر استقرار باید با راه‌اندازی مجدد (پرچم -org یا متغیر
	// HRMS_ORG) صورت گیرد — این‌گونه رفتار سرور قابل پیش‌بینی می‌ماند.
	mux.HandleFunc("GET /api/v1/orgs", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFrom(r.Context())
		isAdmin := user.Role == contracts.RoleAdmin
		// مدیر سیستم همهٔ شرکت‌ها را می‌بیند؛ بقیه فقط شرکت‌های هلدینگ خودشان را
		// (باز کردن هلدینگ = دیدن زیرمجموعه‌های همان هلدینگ).
		list := orgs.ListSummary()
		if !isAdmin {
			list = orgs.ListSummariesOfHolding(orgs.HoldingKey(deps.Org))
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"activeId":      deps.Org.ID,
			"activeTitleFa": deps.Org.TitleFa,
			"active":        deps.Org.Summary(),
			"holdingKey":    orgs.HoldingKey(deps.Org),
			"holdingFa":     deps.Org.HoldingFa,
			"isAdmin":       isAdmin,
			"organizations": list,
		})
	}))

	// هلدینگ‌ها و شرکت‌های زیرمجموعه: مدیر سیستم همه، بقیه فقط هلدینگ خودشان.
	mux.HandleFunc("GET /api/v1/orgs/holdings", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFrom(r.Context())
		holdings := orgs.Holdings()
		if user.Role != contracts.RoleAdmin {
			own := orgs.HoldingKey(deps.Org)
			filtered := make([]orgs.Holding, 0, 1)
			for _, h := range holdings {
				if h.Key == own {
					filtered = append(filtered, h)
				}
			}
			holdings = filtered
		}
		out := make([]map[string]any, 0, len(holdings))
		for _, h := range holdings {
			companies := make([]map[string]string, 0, len(h.Companies))
			for _, c := range h.Companies {
				companies = append(companies, map[string]string{"id": c.ID, "titleFa": c.TitleFa})
			}
			out = append(out, map[string]any{"key": h.Key, "titleFa": h.TitleFa, "titleEn": h.TitleEn, "companies": companies})
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"holdings": out, "total": len(out)})
	}))

	// پروفایل شرکت فعال همراه با پارامترهای جاری همان شرکت.
	mux.HandleFunc("GET /api/v1/orgs/active", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, deps.Org.Summary())
	}))

	// کارتابل سراسری: همهٔ کارهای گردش‌کاری که نقش کاربر جاری مجاز به انجامش است،
	// از تمام ماژول‌های رکوردی. هیچ منطق مجاز بودن این‌جا نیست؛ هر منبع کارهایش را
	// بر پایهٔ همان قواعدی که در دامنه‌اش اعلام شده برمی‌گرداند.
	mux.HandleFunc("GET /api/v1/workflow/inbox", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFrom(r.Context())
		actor := contracts.WorkflowActor{
			Username: user.Username, Role: user.Role,
			DepartmentID: user.DepartmentID, EmployeeID: user.EmployeeID,
		}
		items := []contracts.WorkflowTask{}
		byModule := map[string]int{}
		seen := map[string]bool{}
		for _, src := range deps.Ports.WorkflowSources() {
			for _, t := range src.TasksFor(r.Context(), actor) {
				// تنها کارهای ماژول‌هایی نشان داده می‌شوند که کاربر در آن‌ها اجازهٔ اقدام
				// دارد؛ پس کارتابل با سطح دسترسی نقش هم‌خوان می‌ماند و پرنویز نمی‌شود.
				if !deps.Policy.Allows(user.Role, t.Module, "approve") {
					continue
				}
				key := t.Module + "|" + t.RecordID + "|" + t.Action
				if seen[key] {
					continue
				}
				seen[key] = true
				items = append(items, t)
				byModule[t.Module]++
			}
		}
		httpx.WriteJSON(w, http.StatusOK, contracts.WorkflowTaskPage{Items: items, Total: len(items), ByKind: byModule})
	}))

	// سوابق پروندهٔ کارمند: احکام قبلی، قراردادهای پیشین، ترفیعات و انتصابات، مدارک،
	// وام‌ها و هر سند دیگری که ماژول‌ها به‌عنوان «سابقه» اعلام کرده‌اند — به ترتیب نزولی
	// تاریخ مؤثر، به‌همراه سابقهٔ تغییرات خودِ هر سند.
	//
	// هیچ قاعده‌ای این‌جا نیست: هر ماژول سابقهٔ خودش را اعلام می‌کند
	// (`contracts.HistorySource`) و این مسیر فقط دسترسی نقش را می‌سنجد، سطرها را می‌چیند
	// و پاسخ را می‌سازد؛ پس افزودن ماژول تازه هیچ تغییری در این مسیر نمی‌خواهد.
	mux.HandleFunc("GET /api/v1/history", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFrom(r.Context())
		employeeID := strings.TrimSpace(r.URL.Query().Get("employeeId"))
		if employeeID == "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"برای دیدن سوابق، شناسهٔ کارمند (employeeId) لازم است.", nil)
			return
		}
		// محدودهٔ دادهٔ نقش بر پروندهٔ کارمند: نقشی که فقط دادهٔ خودش را می‌بیند نباید
		// سابقهٔ کارمند دیگری را بخواند و نقش واحدی فقط کارکنان واحد خودش را.
		if !historyEmployeeAllowed(r.Context(), deps, user, employeeID) {
			httpx.WriteErrorMsg(w, http.StatusForbidden, contracts.CodeForbidden,
				"دسترسی شما به پروندهٔ این کارمند مجاز نیست.", nil)
			return
		}

		items := []contracts.HistoryEntry{}
		failed := []string{}
		for _, src := range deps.Ports.HistorySources() {
			// ماژولی که این نقش اجازهٔ خواندنش را ندارد، سابقه‌اش هم نمی‌آید.
			if !deps.Policy.Allows(user.Role, src.Module(), "read") {
				continue
			}
			entries, err := src.HistoryOf(r.Context(), employeeID)
			if err != nil {
				// خطای یک ماژول نباید کل سابقه را از کار بیندازد؛ ولی پنهان هم نمی‌شود و
				// در پاسخ به کاربر اعلام می‌شود تا «سابقهٔ ناقص» با «سابقهٔ خالی» اشتباه نشود.
				if deps.Log != nil {
					deps.Log.Warn("خواندن سابقهٔ ماژول ناموفق بود", "module", src.Module(), "employeeId", employeeID, "error", err)
				}
				failed = append(failed, src.ModuleTitleFa())
				continue
			}
			items = append(items, entries...)
		}

		items = history.Filter(items, r.URL.Query().Get("module"), r.URL.Query().Get("kind"))
		history.Sort(items)
		history.MarkLatest(items)

		httpx.WriteJSON(w, http.StatusOK, contracts.HistoryTimeline{
			EmployeeID:   employeeID,
			EmployeeName: historyEmployeeName(r.Context(), deps, employeeID),
			Items:        items,
			Total:        len(items),
			ByModule:     history.Counts(items),
			Warnings:     historyWarningsFa(failed),
			NoteFa:       "سابقه از ماژول‌هایی می‌آید که نقش شما اجازهٔ خواندنشان را دارد. اسناد پیشین — احکام قبلی، قراردادهای پیشین، مدارک و سمت‌های پیشین — پاک نمی‌شوند و همین‌جا کنار سند جاری دیده می‌شوند.",
		})
	}))

	// ورود: مسیر عمومی.
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req contracts.LoginRequest
		if !httpx.DecodeJSON(w, r, &req) {
			return
		}
		if strings.TrimSpace(req.Username) == "" || req.Password == "" {
			httpx.WriteError(w, http.StatusBadRequest, contracts.CodeValidationFailed, map[string]string{
				"username": "نام کاربری الزامی است.",
				"password": "رمز عبور الزامی است.",
			})
			return
		}
		session, err := deps.Auth.Login(req.Username, req.Password)
		if err != nil {
			deps.Audit.Append(req.Username, "auth", "login-failed", r.RemoteAddr, err.Error())
			httpx.WriteErrorMsg(w, http.StatusUnauthorized, contracts.CodeUnauthorized, err.Error(), nil)
			return
		}
		deps.Audit.Append(session.User.Username, "auth", "login", r.RemoteAddr, "ورود موفق")
		httpx.WriteJSON(w, http.StatusOK, session)
	})

	// تمدید نشست.
	mux.HandleFunc("POST /api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var req contracts.RefreshRequest
		if !httpx.DecodeJSON(w, r, &req) {
			return
		}
		session, err := deps.Auth.Refresh(req.RefreshToken)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnauthorized, contracts.CodeUnauthorized, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, session)
	})

	// کاربر جاری همراه با منوی نقش‌محور و دامنهٔ داده.
	// کلاینت ناوبری خودش را از همین پاسخ می‌سازد، پس منو در کلاینت دستی تعریف نمی‌شود.
	mux.HandleFunc("GET /api/v1/auth/me", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFrom(r.Context())
		scopes := map[string]string{}
		for _, m := range deps.Registry.All() {
			scopes[m.ID] = deps.Policy.Scope(user.Role, m.ID)
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"user":    user,
			"roleFa":  contracts.RoleFa[user.Role],
			"menu":    deps.Policy.MenuFor(user.Role, user.DepartmentID),
			"scopes":  scopes,
			"syncSeq": deps.Sync.Seq(),
			"modules": deps.Registry.Count(),
			"org":     deps.Org.Summary(),
		})
	}))

	// هم‌گام‌سازی: دلتای تغییرات از نشانگر کلاینت.
	mux.HandleFunc("GET /api/v1/sync/changes", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		since, _ := strconv.Atoi(r.URL.Query().Get("since"))
		httpx.WriteJSON(w, http.StatusOK, deps.Sync.Since(since))
	}))

	// هم‌گام‌سازی: اعمال دسته‌ای عملیات ثبت‌شده در حالت آفلاین (ایدمپوتنت با opId).
	mux.HandleFunc("POST /api/v1/sync/mutations", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		var mutations []contracts.Mutation
		if !httpx.DecodeJSON(w, r, &mutations) {
			return
		}
		if len(mutations) > 500 {
			httpx.WriteErrorMsg(w, http.StatusRequestEntityTooLarge, contracts.CodeBadRequest, "در هر درخواست حداکثر ۵۰۰ عملیات پذیرفته می‌شود.", nil)
			return
		}
		result := deps.Sync.Apply(mutations)
		if user, ok := auth.UserFrom(r.Context()); ok {
			deps.Audit.Append(user.Username, "sync", "mutations", "", fmt.Sprintf("پذیرفته=%d ردشده=%d", len(result.Accepted), len(result.Rejected)))
		}
		httpx.WriteJSON(w, http.StatusOK, result)
	}))
}
