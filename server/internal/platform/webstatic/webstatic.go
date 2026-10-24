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
)

// Handler فایل‌های پوشهٔ داده‌شده را سرو می‌کند.
//
// سرصفحهٔ no-store عمدی است: در اجرای توسعه، تغییر فایل‌های رابط باید بدون پاک‌کردن
// کش مرورگر دیده شود. در استقرار واقعی این پوشه را باید وب‌سرور یا CDN با سیاست
// کش مناسب سرو کند.
func Handler(dir string, logger *slog.Logger) http.Handler {
	dir = filepath.Clean(dir)
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		if logger != nil {
			logger.Debug("سرو فایل رابط", "path", r.URL.Path)
		}
		fs.ServeHTTP(w, r)
	})
}

// Exists بررسی می‌کند پوشهٔ رابط وجود دارد؛ اگر نباشد سرور فقط API را بالا می‌آورد.
func Exists(dir string) bool {
	if dir == "" {
		return false
	}
	st, err := os.Stat(filepath.Clean(dir))
	return err == nil && st.IsDir()
}
