// Package webstatic میزبان فایل‌های رابط وب است تا سامانه در اجرای محلی روی یک
// مبدأ (origin) واحد بالا بیاید: خود رابط روی `/` و API ماژول‌ها روی `/api/v1/...`.
//
// این کار عملی است نه تشریفاتی: با یک مبدأ واحد، مرورگر برای احراز هویت و کوکی و
// CORS تنظیم اضافه لازم ندارد و همان چیزی که روی لپ‌تاب دیده می‌شود، روی سرور هم
// دقیقاً همان مسیر را دارد (فقط با یک پروکسی جلوی درگاه).
package webstatic

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Handler فایل‌های پوشهٔ داده‌شده را سرو می‌کند.
//
// سرصفحهٔ no-store عمدی است: در اجرای توسعه، تغییر فایل‌های رابط باید بدون پاک‌کردن
// کش مرورگر دیده شود. در استقرار واقعی این پوشه را باید وب‌سرور یا CDN با سیاست
// کش مناسب سرو کند.
//
// مسیرهای مشکوک پیش از رسیدن به FileServer رد می‌شوند: `..` (پیمایش مسیر) و هر بخشی
// که با نقطه آغاز شود (فایل پنهان مثل `.env` یا `.git`). FileServer خودش هم `..` را
// پاک می‌کند، ولی این رد صریح است تا رفتار در هر نسخهٔ Go و پشت هر پروکسی یکی بماند.
func Handler(dir string, logger *slog.Logger) http.Handler {
	dir = filepath.Clean(dir)
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		if !safePath(r.URL.Path) {
			loggerDebug(logger, "درخواست مسیر ناامن رد شد", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		loggerDebug(logger, "سرو فایل رابط", r.URL.Path)
		fs.ServeHTTP(w, r)
	})
}

// Exists بررسی می‌کند پوشهٔ رابط وجود دارد؛ اگر نباشد سرور فقط API را بالا می‌آورد.
func Exists(dir string) bool {
	if dir == "" {
		return false
	}
	st, err := os.Stat(dirpath(dir))
	return err == nil && st.IsDir()
}

// safePath آیا مسیر درخواست برای سرو فایل ایستا بی‌خطر است.
func safePath(p string) bool {
	if strings.ContainsRune(p, 0) {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return false
		}
		// فایل/پوشهٔ پنهان: هیچ دارایی عمومی‌ای در رابط وب با نقطه آغاز نمی‌شود.
		if len(seg) > 1 && strings.HasPrefix(seg, ".") {
			return false
		}
	}
	return true
}

func dirpath(dir string) string { return filepath.Clean(dir) }

func loggerDebug(logger *slog.Logger, msg, path string) {
	if logger != nil {
		logger.Debug(msg, "path", path)
	}
}
