// Command bonusd «سیستم پاداش»: زیرسامانهٔ تفکیکی پاداش برای اجرا روی یک رایانه.
//
// چرا یک باینری جدا: سامانهٔ جامع منابع انسانی ماژول پاداش را در خود دارد، ولی برای
// واحد پاداش که فقط همین یک کار را می‌کند، بالا آوردن کل سامانه لازم نیست. این باینری
// تنها ماژول پاداش را سوار می‌کند و کمترین نیازمندی را دارد:
//
//   - هیچ پایگاه‌داده‌ای لازم نیست (داده در فایل JSON پوشهٔ داده).
//   - هیچ Docker، Node، Python یا اینترنتی لازم نیست.
//   - رابط وب داخل خود فایل اجرایی جاسازی شده است؛ اجرا = یک فایل.
//   - فهرست کارکنان از یک فایل CSV سادهٔ قابل ویرایش با Excel خوانده می‌شود.
//
// نمونهٔ اجرا:
//
//	bonusd                                   → http://127.0.0.1:8099 با پوشهٔ ./bonus-data
//	bonusd -addr 127.0.0.1:9000 -open        → اجرا روی درگاه دیگر و باز شدن مرورگر
//	bonusd -data D:\bonus -admin-password "…" → تغییر پوشهٔ داده و بازنشانی رمز مدیر
package main

import (
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"context"
	"errors"
	"flag"
	"net/http"

	"hrms/internal/bootstrap"
)

func main() {
	if err := run(); err != nil {
		slog.Error("سیستم پاداش با خطا پایان یافت", "error", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("addr", "127.0.0.1:8099", "نشانی شنود؛ پیش‌فرض فقط روی همین رایانه قابل دسترسی است")
	dataDir := flag.String("data", "bonus-data", "پوشهٔ داده (رکوردها، کاربران، کارکنان و کلید امضا)")
	adminUser := flag.String("admin-user", "admin", "نام کاربری مدیر برای ساخت یا بازنشانی رمز")
	adminPassword := flag.String("admin-password", "", "رمز مدیر؛ خالی یعنی رمز فعلی فایل کاربران دست‌نخورده بماند")
	orgFlag := flag.String("org", "", "شناسهٔ شرکت فعال؛ خالی یعنی استقرار پیش‌فرض")
	secretFlag := flag.String("jwt-secret", "", "کلید امضای نشست؛ خالی یعنی کلید پایدار پوشهٔ داده")
	openBrowser := flag.Bool("open", false, "باز کردن مرورگر پیش‌فرض روی آدرس سامانه پس از راه‌اندازی")
	debug := flag.Bool("debug", false, "سطح لاگ تفصیلی")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	// کلید امضا: اولویت با پرچم، سپس متغیر محیطی، و در نهایت کلید پایدار پوشهٔ داده.
	secret := strings.TrimSpace(*secretFlag)
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("BONUS_JWT_SECRET"))
	}
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("HRMS_JWT_SECRET"))
	}

	handler, info, err := bootstrap.NewStandalone(bootstrap.StandaloneConfig{
		DataDir:       *dataDir,
		AdminUser:     *adminUser,
		AdminPassword: *adminPassword,
		Org:           strings.TrimSpace(*orgFlag),
		JWTSecret:     []byte(secret),
		Logger:        logger,
		Clock:         time.Now,
	})
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

	// نشانی قابل نمایش: اگر روی همهٔ کارت‌های شبکه گوش داده شود، آدرس محلی نشان داده
	// می‌شود تا کاربر بداند مرورگر را کجا باز کند.
	url := displayURL(*addr)
	logger.Info("سیستم پاداش آماده است",
		"url", url,
		"store", info.Store,
		"dataDir", info.DataDir,
		"records", info.RecordsPath,
		"staff", info.StaffPath,
		"staffCount", info.StaffCount,
		"recordsCount", info.RecordCount,
		"modules", strings.Join(info.Modules, ","),
		"adminUser", info.AdminUser,
	)
	// اگر روی همهٔ کارت‌های شبکه گوش داده شود، نشانی‌های شبکهٔ محلی هم گزارش می‌شوند تا
	// کاربر بتواند از رایانه‌های دیگر (مثلاً رایانه‌ای با ویندوز قدیمی‌تر که خودش نمی‌تواند
	// برنامه را اجرا کند) با مرورگر به همین سامانه وصل شود.
	if lan := lanURLs(*addr); len(lan) > 0 {
		logger.Info("دسترسی از رایانه‌های دیگرِ شبکه با این نشانی‌ها", "urls", strings.Join(lan, "  "))
	}
	logger.Info("برای پشتیبان‌گیری همین پوشهٔ داده را کپی کنید", "dataDir", info.DataDir)
	if info.StaffWarning != "" {
		logger.Warn("هشدار فهرست کارکنان", "warning", info.StaffWarning)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	if *openBrowser {
		go func() {
			// کمی درنگ تا شنود واقعاً برقرار شود، بعد مرورگر باز می‌شود.
			time.Sleep(700 * time.Millisecond)
			if err := openInBrowser(url); err != nil {
				logger.Warn("باز کردن مرورگر ناموفق بود؛ آدرس را دستی باز کنید", "url", url, "error", err)
			}
		}()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("در حال خاموشی نرم؛ داده‌ها روی دیسک ذخیره شده‌اند")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

// displayURL آدرس قابل باز کردن در مرورگر را از نشانی شنود می‌سازد.
func displayURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

// lanURLs وقتی برنامه روی همهٔ کارت‌های شبکه گوش می‌دهد، نشانی‌های IPv4 غیرحلقه‌ای را
// می‌سازد؛ خالی یعنی برنامه فقط روی loopback شنود می‌کند و از بیرون دیده نمی‌شود.
func lanURLs(addr string) []string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}
	if host != "" && host != "0.0.0.0" && host != "::" {
		return nil
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
				continue
			}
			out = append(out, "http://"+net.JoinHostPort(ipnet.IP.String(), port))
		}
	}
	return out
}

// openInBrowser آدرس را در مرورگر پیش‌فرض سامانه باز می‌کند (بدون هیچ وابستگی بیرونی).
func openInBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
