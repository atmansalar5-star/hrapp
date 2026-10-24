// سیم‌کشی سرور: ثبت ماژول‌ها، سیاست‌ها، دادهٔ اولیه و مسیرهای HTTP — «standalone».

package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/bonus"
	"hrms/internal/platform/access"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/localstore"
	"hrms/internal/platform/localstore/webui"
	"hrms/internal/platform/modulekit"
	platformmodules "hrms/internal/platform/modules"
	"hrms/internal/platform/orgs"
	"hrms/internal/platform/rbac"
	"hrms/internal/platform/recordkit"
	"hrms/internal/platform/synclog"
	"hrms/internal/rules"
)

// این فایل «سیستم پاداش» را می‌سازد: یک زیرسامانهٔ تفکیکی برای اجرا روی یک
// رایانه، بدون پایگاه‌داده، بدون Docker، بدون Node و بدون هیچ سرویس بیرونی.
//
// چرا در بستهٔ bootstrap: همان‌جایی که سیم‌کشی کامل سامانه زندگی می‌کند، یک اجرای
// تفکیکی هم سوار می‌شود؛ پس مسیرهای پلتفرم (ورود، /healthz، کاربر جاری) یک نسخهٔ دوم
// ندارند و رفتار ورود و دسترسی دقیقاً همان رفتار استقرار سازمانی می‌ماند.
//
// تفاوت‌های عمدی با استقرار سازمانی:
//   - تنها ماژول فعال «پاداش» است؛ منو و مسیرهای بقیهٔ ماژول‌ها ثبت نمی‌شوند.
//   - داده در انبار فایل JSON می‌نشیند (recordkit.FileStore)، نه Postgres.
//   - فهرست کارکنان از فایل CSV محلی می‌آید (پورت کارکنان) تا ماژول کارکنان لازم نباشد.
//   - کاربران و کلید امضا از پوشهٔ داده خوانده می‌شوند تا ورود بین اجراها پایدار بماند.
//   - رابط وب داخل خود باینری جاسازی شده است؛ اجرا فقط یک فایل اجرایی می‌خواهد.

// StandaloneConfig پیکربندی اجرای مستقل.
type StandaloneConfig struct {
	// DataDir پوشهٔ داده: رکوردها، کاربران، کارکنان و کلید امضا.
	DataDir string
	// AdminUser و AdminPassword برای ساخت یا بازنشانی رمز کاربر مدیر.
	AdminUser     string
	AdminPassword string
	// Org شناسهٔ شرکت فعال (پیش‌فرض: استقرار پیش‌فرض سازمان).
	Org string
	// JWTSecret کلید امضا؛ خالی یعنی از فایل secret.key در پوشهٔ داده خوانده/ساخته شود.
	JWTSecret []byte
	Logger    *slog.Logger
	Clock     func() time.Time
}

// StandaloneInfo وضعیت اجرای مستقل برای گزارش راه‌اندازی و نمایش در رابط.
type StandaloneInfo struct {
	AppTitleFa   string   `json:"appTitleFa"`
	Version      string   `json:"version"`
	Store        string   `json:"store"`
	DataDir      string   `json:"dataDir"`
	RecordsPath  string   `json:"recordsPath"`
	UsersPath    string   `json:"usersPath"`
	StaffPath    string   `json:"staffPath"`
	StaffCount   int      `json:"staffCount"`
	StaffWarning string   `json:"staffWarning,omitempty"`
	RecordCount  int      `json:"recordCount"`
	Modules      []string `json:"modules"`
	AdminUser    string   `json:"adminUser"`
	UsersCreated bool     `json:"usersCreated"`
}

// standaloneVersion نسخهٔ زیرسامانهٔ مستقل.
const standaloneVersion = "1.1.0"

// standaloneModuleID شناسهٔ تنها ماژول فعال در اجرای مستقل.
const standaloneModuleID = "bonus"

// NewStandalone هندلر ریشه و وضعیت اجرای مستقل را می‌سازد.
func NewStandalone(cfg StandaloneConfig) (http.Handler, StandaloneInfo, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	dataDir := strings.TrimSpace(cfg.DataDir)
	if dataDir == "" {
		dataDir = "bonus-data"
	}
	absDir, err := filepath.Abs(dataDir)
	if err != nil {
		absDir = dataDir
	}
	recordsPath := filepath.Join(absDir, "records.json")
	usersPath := filepath.Join(absDir, "users.json")
	staffPath := filepath.Join(absDir, "staff.csv")

	// کلید امضا: اولویت با کلید تزریق‌شده (متغیر محیطی)، وگرنه کلید پایدار پوشهٔ داده.
	secret := cfg.JWTSecret
	if len(secret) == 0 {
		generated, err := localstore.Secret(absDir)
		if err != nil {
			return nil, StandaloneInfo{}, err
		}
		secret = generated
		logger.Info("کلید امضای نشست از پوشهٔ داده خوانده شد", "path", filepath.Join(absDir, "secret.key"))
	}

	accounts, usersCreated, err := localstore.Users(usersPath, cfg.AdminUser, cfg.AdminPassword)
	if err != nil {
		return nil, StandaloneInfo{}, err
	}
	if usersCreated {
		logger.Warn("فایل کاربران ساخته شد؛ رمز پیش‌فرض را عوض کنید",
			"path", usersPath, "user", localstore.DefaultAdminUser, "defaultPassword", localstore.DefaultAdminPassword)
	}

	staff, err := localstore.NewStaffStore(staffPath)
	if err != nil {
		return nil, StandaloneInfo{}, err
	}
	if w := staff.Warning(); w != "" {
		logger.Warn("هشدار فهرست کارکنان", "warning", w)
	}

	store, err := recordkit.NewFileStore(recordsPath)
	if err != nil {
		return nil, StandaloneInfo{}, err
	}

	org, ok := orgs.Lookup(cfg.Org)
	if !ok {
		return nil, StandaloneInfo{}, orgs.UnknownError(cfg.Org)
	}

	// تنها ماژول پاداش فعال است؛ منوی نقش و مسیرهای بقیهٔ ماژول‌ها ساخته نمی‌شود.
	registry := platformmodules.NewRegistry(map[string]bool{standaloneModuleID: true})

	params, err := json.Marshal(rules.DefaultPayrollParams())
	if err != nil {
		return nil, StandaloneInfo{}, fmt.Errorf("سریال‌سازی پارامترهای پیش‌فرض ناموفق: %w", err)
	}

	auditLog := audit.New()
	accessStore := access.NewStore()
	policy := rbac.New(registry, auditLog, accessStore).WithTenant(org.ID, orgs.HoldingKey(org))
	ports := modulekit.NewPorts()
	// پورت کارکنان: فهرست CSV محلی. بدین‌گونه فرم تخصیص بدون ماژول کارکنان کار می‌کند.
	ports.SetEmployeeReader(staff)

	deps := modulekit.Deps{
		Log:      logger,
		Registry: registry,
		Auth:     auth.NewService(secret, accounts).WithClock(clock),
		Policy:   policy,
		Audit:    auditLog,
		Sync:     synclog.New(),
		Calendar: calendar.DefaultWorkCalendar(),
		Params:   params,
		Ports:    ports,
		Clock:    clock,
		Org:      org,
		Access:   accessStore,
		// انبار سفارشی: همهٔ ماژول‌های recordkit روی فایل JSON کار می‌کنند.
		Records: store,
	}

	base := StandaloneInfo{
		AppTitleFa: "سیستم پاداش", Version: standaloneVersion, Store: "json-file",
		DataDir: absDir, RecordsPath: recordsPath, UsersPath: usersPath, StaffPath: staffPath,
		Modules:   []string{standaloneModuleID},
		AdminUser: firstNonEmpty(cfg.AdminUser, localstore.DefaultAdminUser), UsersCreated: usersCreated,
	}

	mux := http.NewServeMux()
	// مسیرهای پلتفرم (ورود، تمدید نشست، کاربر جاری، سلامت، کارتابل، هم‌گام‌سازی).
	registerPlatformRoutes(mux, deps)
	// ماژول پاداش: رکوردها، گزینه‌ها، داشبورد و فرم تخصیص. بذرکاری نمونه هم همین
	// فراخوانی انجام می‌شود، پس شمار رکوردها پس از این سطر خوانده می‌شود تا گزارش
	// راه‌اندازی همان وضعیت واقعی روی دیسک را نشان دهد.
	bonus.New().Register(mux, deps)
	// مسیرهای اختصاصی اجرای مستقل (وضعیت و فهرست کارکنان).
	registerStandaloneRoutes(mux, deps, store, staff, base)
	// مسیرهای /api که هیچ هندلری برایشان ثبت نشده است باید ۴۰۴ JSON بدهند، نه صفحهٔ
	// HTML با کد ۲۰۰. بدون این سطر، همهٔ مسیرهای ناشناس به رابط وب می‌افتند و کلاینتی
	// که نشانی را اشتباه نوشته «موفقیت» می‌بیند — خطایی که پیدا کردنش گران است.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound,
			"این مسیر در اجرای مستقل پاداش وجود ندارد", map[string]string{"path": r.URL.Path})
	})
	// رابط وب جاسازی‌شده روی ریشه؛ باید آخر ثبت شود تا مسیرهای دقیق‌تر مقدم بمانند.
	mux.Handle("/", webui.Handler())

	base.StaffCount = staff.Count()
	base.StaffWarning = staff.Warning()
	if page, err := store.List(context.Background(), contracts.RecordFilter{Page: 1, Size: 1}); err == nil {
		base.RecordCount = page.Total
	}
	return deps.Auth.Middleware(mux), base, nil
}

// registerStandaloneRoutes مسیرهای ویژهٔ اجرای مستقل را سوار می‌کند.
//
// این مسیرها داخل هیچ ماژولی نیستند چون به خودِ «اجرای مستقل» مربوط‌اند، نه به دامنهٔ
// پاداش: وضعیت انبار محلی و فهرست کارکنانِ همان پوشهٔ داده.
func registerStandaloneRoutes(mux *http.ServeMux, deps modulekit.Deps, store *recordkit.FileStore, staff *localstore.StaffStore, base StandaloneInfo) {
	// وضعیت: مسیر فایل‌های داده، شمار کارکنان و رکوردها. برای پشتیبان‌گیری همین کافی است.
	mux.HandleFunc("GET /api/v1/standalone/info", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		out := base
		out.StaffCount = staff.Count()
		out.StaffWarning = staff.Warning()
		if page, err := store.List(r.Context(), contracts.RecordFilter{Page: 1, Size: 1}); err == nil {
			out.RecordCount = page.Total
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))

	// فهرست کارکنان از فایل CSV؛ همان پورتی که فرم تخصیص از آن تغذیه می‌شود.
	mux.HandleFunc("GET /api/v1/standalone/staff", auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		size, _ := strconv.Atoi(q.Get("size"))
		result, err := staff.List(r.Context(), contracts.EmployeeFilter{
			Query: q.Get("query"), DepartmentID: q.Get("departmentId"), Status: q.Get("status"),
			Scope: "all", Page: page, Size: size,
		})
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeBadRequest, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, result)
	}))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
