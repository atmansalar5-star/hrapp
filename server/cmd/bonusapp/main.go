//go:build windows

// Command bonusapp «سیستم پاداش» را به شکل یک برنامهٔ دسکتاپ ویندوزی اجرا می‌کند:
// پنجرهٔ بومی، آیکون و ورودی مستقل در نوار وظیفه، بدون کنسول و بدون نوار نشانی مرورگر.
//
// چرا این برنامه جدا از bonusd وجود دارد: bonusd همان سامانه را با رابط وب سرو می‌کند و
// کاربر باید خودش مرورگر را باز کند. برای واحد پاداش که روی یک رایانه کار می‌کند، رفتار
// درست یک «برنامه» است: دوبار کلیک، پنجره باز می‌شود، کار می‌کند، بسته می‌شود.
//
// سه تصمیم معماری:
//
//   - سرور داخل همین فرآیند اجرا می‌شود (in-process) و روی 127.0.0.1 گوش می‌دهد؛ پس
//     پنجره و داده یک برنامهٔ واحدند و هیچ فرآیند پس‌زمینهٔ گم‌شده باقی نمی‌ماند.
//   - پنجره با WebView2 ساخته می‌شود که روی ویندوز ۱۰/۱۱ پیش‌فرض نصب است. موتور آن
//     داخل خود فایل اجرایی جاسازی شده است (بدون DLL کنار برنامه، بدون CGO).
//   - هیچ نصب و نوشتنی خارج از پوشهٔ خودِ برنامه لازم نیست: داده در bonus-data کنار
//     فایل اجرایی می‌نشیند، پس کل پوشه با کپی بردن، منتقل می‌شود.
//
// نمونهٔ اجرا:
//
//	bonusapp.exe                              → پنجره روی http://127.0.0.1:8099
//	bonusapp.exe -data D:\BonusData           → پوشهٔ دادهٔ دیگر
//	bonusapp.exe -port 8123 -debug            → درگاه دیگر و لاگ تفصیلی
//	bonusapp.exe -admin-password "…"          → بازنشانی رمز مدیر پیش از شروع
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"hrms/internal/bootstrap"
)

// appTitleFa عنوان پنجره و همهٔ پیام‌های بومی.
const appTitleFa = "سیستم پاداش"

// appVersion نسخهٔ برنامهٔ دسکتاپ؛ با نسخهٔ خود سامانه یکی است.
const appVersion = "1.1.0"

func main() {
	// پنجرهٔ بومی و چرخهٔ پیام ویندوز باید روی رشتهٔ اصلی سیستم اجرا شوند؛ اگر Go
	// این فوریت (goroutine) را جابه‌جا کند، پیام‌های ویندوز به دست پنجره نمی‌رسند.
	runtime.LockOSThread()

	if err := run(); err != nil {
		failFatal(err)
	}
}

// run کل چرخهٔ عمر برنامه را می‌سازد: پیکربندی، تک‌نسخه‌ای، سرور درون‌فرآیندی، پنجره و
// خاموشی نرم. خطای برگشتی در main به پیام بومی تبدیل می‌شود.
func run() error {
	dataFlag := flag.String("data", "", "پوشهٔ داده؛ خالی یعنی پوشهٔ bonus-data کنار همین فایل اجرایی")
	portFlag := flag.Int("port", 8099, "درگاه محلی رابط")
	adminUser := flag.String("admin-user", "admin", "نام کاربری مدیر")
	adminPassword := flag.String("admin-password", "", "رمز مدیر؛ خالی یعنی رمز فعلی دست‌نخورده بماند")
	debug := flag.Bool("debug", false, "سطح لاگ تفصیلی")
	flag.Parse()

	baseDir, err := executableDir()
	if err != nil {
		return err
	}
	dataDir := strings.TrimSpace(*dataFlag)
	if dataDir == "" {
		// داده کنار فایل اجرایی: شرط انتقال کل پوشه با یک کپی.
		dataDir = filepath.Join(baseDir, "bonus-data")
	}
	if abs, err := filepath.Abs(dataDir); err == nil {
		dataDir = abs
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("ساخت پوشهٔ داده «%s» ناموفق: %w", dataDir, err)
	}

	logger, closeLog, err := newLogger(dataDir, *debug)
	if err != nil {
		return err
	}
	defer func() { _ = closeLog.Close() }()

	// تک‌نسخه‌ای: دو نمونه روی یک پوشهٔ داده می‌توانند فایل رکوردها را خراب کنند.
	switch err := acquireSingleInstance(dataDir); {
	case errors.Is(err, errAlreadyRunning):
		showMessage(appTitleFa, "این برنامه از قبل باز است.\n\n"+
			"اگر پنجره‌اش را نمی‌بینید، در نوار وظیفه (Taskbar) دنبال «"+appTitleFa+"» بگردید\n"+
			"یا آن را ببندید و دوباره امتحان کنید.")
		return nil
	case err != nil:
		// نبودِ قفل نباید برنامه را متوقف کند؛ فقط در لاگ ثبت می‌شود.
		logger.Warn("قفل تک‌نسخه‌ای برقرار نشد", "error", err)
	}

	logger.Info("راه‌اندازی برنامهٔ دسکتاپ",
		"version", appVersion, "dataDir", dataDir, "baseDir", baseDir)

	handler, info, err := bootstrap.NewStandalone(bootstrap.StandaloneConfig{
		DataDir:       dataDir,
		AdminUser:     *adminUser,
		AdminPassword: *adminPassword,
		Logger:        logger,
		Clock:         time.Now,
	})
	if err != nil {
		return err
	}

	listener, port, err := listenLoopback(*portFlag)
	if err != nil {
		return fmt.Errorf("گشودن درگاه محلی ناموفق: %w", err)
	}
	if port != *portFlag {
		// درگاه دلخواه اشغال بود؛ درگاه آزاد گرفته شد. توکن ورود ذخیره‌شده در مرورگر
		// به مبدأ (Origin) گره خورده است، پس این تغییر یعنی یک بار ورود دوباره.
		logger.Warn("درگاه درخواستی اشغال بود؛ درگاه آزاد انتخاب شد",
			"requested", *portFlag, "actual", port)
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	logger.Info("سیستم آماده است",
		"url", url, "store", info.Store, "records", info.RecordCount,
		"staff", info.StaffCount, "modules", strings.Join(info.Modules, ","), "adminUser", info.AdminUser)
	if info.StaffWarning != "" {
		logger.Warn("هشدار فهرست کارکنان", "warning", info.StaffWarning)
	}

	// پنجرهٔ بومی. اگر WebView2 در دسترس نبود، به‌جای خطای مبهم، راهنمای روشن نشان
	// داده می‌شود و برنامه با خیال راحت بسته می‌شود (داده سالم است).
	windowErr := runWindow(windowConfig{
		Title:   appTitleFa,
		URL:     url,
		DataDir: dataDir,
		BaseDir: baseDir,
		Debug:   *debug,
		Logger:  logger,
		Port:    port,
	})
	if windowErr != nil {
		logger.Error("اجرای پنجرهٔ بومی ناموفق بود", "error", windowErr, "url", url)
		showMessage(appTitleFa, fmt.Sprintf(
			"پنجرهٔ برنامه باز نشد.\n\n%s\n\n"+
				"راه‌حل: بستهٔ رایگان «Microsoft Edge WebView2 Runtime» را از سایت مایکروسافت نصب کنید.\n"+
				"(روی ویندوز ۱۰ و ۱۱ معمولاً از پیش نصب است.)\n\n"+
				"در همین حال می‌توانید این نشانی را در مرورگر باز کنید:\n%s\n\n"+
				"پوشهٔ داده (سالم و دست‌نخورده):\n%s", windowErr.Error(), url, dataDir))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	select {
	case err := <-serverErr:
		logger.Error("سرور با خطا متوقف شد", "error", err)
	default:
	}
	logger.Info("برنامه بسته شد؛ داده روی دیسک سالم است", "dataDir", dataDir)
	return nil
}

// executableDir پوشهٔ فایل اجرایی را برمی‌گرداند؛ مبنای پوشهٔ داده و پشتیبان.
func executableDir() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("مسیر فایل اجرایی خوانده نشد: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}
	return filepath.Dir(resolved), nil
}

// listenLoopback اول درگاه دلخواه را می‌آزماید و اگر اشغال بود درگاه آزاد سیستم را
// می‌گیرد. درگاه ثابت مهم است چون نشست ورود کاربر به مبدأ گره خورده است.
func listenLoopback(preferred int) (net.Listener, int, error) {
	if preferred > 0 {
		if listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", preferred)); err == nil {
			return listener, preferred, nil
		}
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, err
	}
	return listener, listener.Addr().(*net.TCPAddr).Port, nil
}

// newLogger گزارش را در فایل پوشهٔ داده می‌نویسد، نه روی خروجی استاندارد: این برنامه
// بدون کنسول ساخته می‌شود، پس نوشتن روی stdout جایی برای دیده‌شدن ندارد.
func newLogger(dataDir string, debug bool) (*slog.Logger, io.Closer, error) {
	path := filepath.Join(dataDir, "bonusapp.log")
	// چرخش ساده: فایل بزرگ به .1 تغییر نام می‌یابد تا لاگ بی‌نهایت رشد نکند.
	if info, err := os.Stat(path); err == nil && info.Size() > 1<<20 {
		_ = os.Remove(path + ".1")
		_ = os.Rename(path, path+".1")
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("باز کردن فایل لاگ «%s» ناموفق: %w", path, err)
	}
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{Level: level})), file, nil
}

// failFatal خطای کشندهٔ راه‌اندازی را در فایل لاگ پوشهٔ داده و در یک پیام بومی نشان
// می‌دهد؛ بدون کنسول، این تنها راه دیده‌شدن خطاست.
func failFatal(err error) {
	baseDir, _ := executableDir()
	dataDir := filepath.Join(baseDir, "bonus-data")
	logPath := filepath.Join(dataDir, "bonusapp.log")
	if file, openErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); openErr == nil {
		fmt.Fprintf(file, "level=ERROR msg=\"%s\"\n", strings.ReplaceAll(err.Error(), "\"", "'"))
		_ = file.Close()
	}
	showMessage(appTitleFa, "شروع برنامه ناموفق بود:\n\n"+err.Error()+"\n\nگزارش در این فایل است:\n"+logPath)
}
