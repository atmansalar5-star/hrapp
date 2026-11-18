// Command hrmsd سرور سامانهٔ جامع منابع انسانی.
//
// یک باینری، اما ماژولار: هر ماژول پکیج مستقل با مرز اجبارشده است و می‌توان با
// `-modules` فقط بخشی از ماژول‌ها را در یک استقرار فعال کرد (پرچم ویژگی در سطح استقرار).
//
// نمونهٔ اجرا:
//
//	hrmsd -addr :8080 -org bandar_imam
//	hrmsd -addr :8080 -modules employee,dashboard,settings
//
// با پرچم `-web` پوشهٔ رابط وب روی مسیر ریشه سرو می‌شود؛ پس در اجرای محلی، خود رابط
// و API روی یک مبدأ واحد بالا می‌آیند و روی سرور هم فقط همین پوشه را وب‌سرور یا CDN
// سرو می‌کند.
//
// با پرچم `-org` (یا متغیر HRMS_ORG) سامانه برای یکی از شرکت‌های تعریف‌شده در
// contracts/organizations.json بالا می‌آید: ادارات و ماژول‌های فعال و پارامترهای
// حقوقی-مالی همان شرکت اعمال می‌شود. فهرست شرکت‌ها از `GET /api/v1/orgs` دیده می‌شود.
//
// متغیرهای محیطی از فایل `-env` (پیش‌فرض `.env`) خوانده می‌شوند؛ متغیری که از قبل در
// محیط تعریف شده باشد بازنویسی نمی‌شود، چون اولویت با محیط استقرار است.
//
// در نبود متغیر محیطی HRMS_JWT_SECRET یک کلید تصادفی ساخته می‌شود تا اجرای توسعه ساده
// باشد؛ در استقرار واقعی کلید باید از محیط تزریق شود (هیچ رمزی در کد نمی‌نشیند).
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hrms/internal/bootstrap"
	"hrms/internal/platform/events"
)

func main() {
	if err := run(); err != nil {
		slog.Error("سرور با خطا پایان یافت", "error", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", ":8080", "نشانی شنود سرور")
	moduleList := flag.String("modules", "", "فهرست ماژول‌های فعال با کاما؛ خالی یعنی از استقرار شرکت فعال")
	webDir := flag.String("web", "../design/prototype", "پوشهٔ رابط وب که روی ریشه سرو می‌شود؛ خالی یعنی فقط API")
	envFile := flag.String("env", ".env", "فایل متغیرهای محیطی KEY=VALUE؛ خالی یعنی فقط از محیط بخوان")
	orgFlag := flag.String("org", "", "شناسهٔ شرکت فعال (استقرار پارامتریک)؛ خالی یعنی پیش‌فرض organizations.json")
	dbFlag := flag.String("db", "", "نشانی Postgres؛ خالی یعنی متغیر HRMS_DATABASE_URL و در نبود آن اجرای درون‌حافظه‌ای")
	rootFlag := flag.String("root", ".", "ریشهٔ پوشهٔ سرور برای یافتن migrations (پوشهٔ migrations و internal/modules/*/migrations)")
	dataFlag := flag.String("data", "data", "پوشهٔ دادهٔ سرور برای فایل‌های بارگذاری‌شده (نشان شرکت)؛ نسبی یعنی کنار ریشهٔ سرور")
	migrateFlag := flag.Bool("migrate", true, "اجرای مهاجرت‌های اعمال‌نشده در زمان راه‌اندازی (وقتی پایگاه‌داده تنظیم شده باشد)")
	seedDemoFlag := flag.Bool("seed-demo", false, "بذرکاری دادهٔ نمونهٔ آزمایشی روی پایگاه‌دادهٔ واقعی، فقط برای جدول‌های خالی (متغیر محیطی: HRMS_SEED_DEMO=1)")
	debug := flag.Bool("debug", false, "سطح لاگ تفصیلی")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	if *envFile != "" {
		if err := loadDotEnv(*envFile); err != nil {
			logger.Warn("خواندن فایل متغیرهای محیطی ناموفق بود", "path", *envFile, "error", err)
		} else {
			logger.Debug("فایل متغیرهای محیطی بررسی شد (نبودن فایل خطا نیست)", "path", *envFile)
		}
	}

	secret, generated, err := jwtSecret()
	if err != nil {
		return err
	}
	if generated {
		logger.Warn("HRMS_JWT_SECRET تنظیم نشده است؛ برای این اجرا کلید تصادفی ساخته شد. در استقرار واقعی کلید را از محیط تزریق کنید.")
	}

	// بذرکاری دادهٔ نمونه: پرچم خط فرمان یا متغیر محیطی — و متغیر از فایل `.env` هم
	// خوانده می‌شود، چون `flag.Parse` پیش از خواندن فایل اجرا شده است؛ پس تنها تکیه بر
	// مقدار پیش‌فرض پرچم کافی نیست و اینجا دوباره بررسی می‌شود.
	seedDemo := *seedDemoFlag || envBool("HRMS_SEED_DEMO")

	// شرکت فعال: اولویت با پرچم خط فرمان، سپس متغیر محیطی، و در نهایت استقرار پیش‌فرض.
	orgID := strings.TrimSpace(*orgFlag)
	if orgID == "" {
		orgID = strings.TrimSpace(os.Getenv("HRMS_ORG"))
	}

	// نشانی پایگاه‌داده: اولویت با پرچم خط فرمان، سپس متغیر محیطی؛ خالی یعنی اجرای
	// درون‌حافظه‌ای که در توسعه و آزمون مطلوب است.
	dsn := strings.TrimSpace(*dbFlag)
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("HRMS_DATABASE_URL"))
	}

	cfg := bootstrap.Config{
		Modules:     *moduleList,
		WebDir:      *webDir,
		JWTSecret:   secret,
		Logger:      logger,
		Clock:       time.Now,
		Org:         orgID,
		DatabaseURL: dsn,
		ServerRoot:  *rootFlag,
		DataDir:     *dataFlag,
		Migrate:     *migrateFlag,
		SeedDemo:    seedDemo,
		// صندوق خروجی رخداد: مقصد وبهوک، کلید امضا و فاصلهٔ بررسی. خالی بودن مقصد
		// یعنی ارسال خاموش (رخدادها فقط ثبت می‌شوند) — همان رفتار امن پیش‌فرض.
		Events: events.Config{
			URL:      strings.TrimSpace(os.Getenv("HRMS_WEBHOOK_URL")),
			Secret:   strings.TrimSpace(os.Getenv("HRMS_WEBHOOK_SECRET")),
			Interval: envDuration("HRMS_WEBHOOK_INTERVAL", 15*time.Second),
		},
	}
	deps, err := bootstrap.NewDeps(cfg)
	if err != nil {
		return err
	}

	handler, registered, err := bootstrap.NewHandler(deps)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// گردانندهٔ صندوق خروجی رخداد: با پایان ctx (خاموشی سرور) خودش می‌ایستد. اگر
	// ارسال تنظیم نشده باشد، بی‌درنگ و بی‌صدا برمی‌گردد.
	if deps.Events != nil {
		go deps.Events.Run(ctx)
	}

	logger.Info("سرور منابع انسانی آماده است",
		"addr", *addr,
		"store", storeKind(dsn),
		"org", deps.Org.ID,
		"company", deps.Org.TitleFa,
		"holding", deps.Org.HoldingFa,
		"modules", strings.Join(registered, ","),
		"moduleCount", len(registered),
		"webDir", *webDir,
		"auditChainValid", deps.Audit.Verify() == nil,
	)

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("در حال خاموشی نرم سرور")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}

	// گزارش پایانی: سلامت زنجیرهٔ حسابرسی پیش از خاموشی بازبینی می‌شود.
	if err := deps.Audit.Verify(); err != nil {
		logger.Error("زنجیرهٔ لاگ حسابرسی نامعتبر است", "error", err)
	} else {
		logger.Info("زنجیرهٔ لاگ حسابرسی سالم است", "events", deps.Audit.Len())
	}
	return nil
}

// envBool متغیر محیطی بولی را می‌خواند؛ هر مقدار جز 0/false/خالی «روشن» است تا در
// فایل .env بتوان HRMS_SEED_DEMO=1 نوشت.
func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// envDuration متغیر محیطی مدت‌زمان را می‌خواند («15s»، «2m»)؛ نامعتبر یا خالی یعنی
// مقدار پیش‌فرض، تا یک تایپ اشتباه در .env سرور را از کار نیندازد.
func envDuration(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

// storeKind نام انبار فعال را برای گزارش راه‌اندازی برمی‌گرداند.
func storeKind(dsn string) string {
	if strings.TrimSpace(dsn) == "" {
		return "in-memory"
	}
	return "postgres"
}

// jwtSecret کلید امضا را از محیط می‌خواند یا برای توسعه کلید موقت می‌سازد.
func jwtSecret() (secret []byte, generated bool, err error) {
	if v := strings.TrimSpace(os.Getenv("HRMS_JWT_SECRET")); v != "" {
		return []byte(v), false, nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, false, err
	}
	return []byte(base64.RawURLEncoding.EncodeToString(buf)), true, nil
}
