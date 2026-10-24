// Package webui رابط وب اجرای مستقل را داخل خود باینری جاسازی می‌کند.
//
// چرا جاسازی‌شده: شرط «حداقل نیازمندی روی یک رایانه» این است که کاربر فقط یک فایل
// اجرایی داشته باشد — نه پوشهٔ رابط، نه Node، نه هیچ ابزار ساخت. فایل HTML در زمان
// کامپایل داخل باینری می‌نشیند و روی مسیر ریشه سرو می‌شود، پس رابط و API روی یک مبدأ
// واحد بالا می‌آیند و احراز هویت و CORS تنظیم اضافه نمی‌خواهد.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

// جاسازی هم فایل HTML و هم دارایی‌های آن (لوگوی برند سازمان). جاسازی پوشهٔ assets
// یعنی سرصفحهٔ برنامه بدون هیچ فایل کنارِ باینری، آرم شرکت را نشان می‌دهد — همان
// تعهد «فقط یک فایل اجرایی».
//
//go:embed index.html assets
var files embed.FS

// Handler رابط تک‌صفحه‌ای را روی مسیر ریشه سرو می‌کند.
func Handler() http.Handler {
	sub, err := fs.Sub(files, ".")
	if err != nil {
		// جاسازی در زمان کامپایل انجام شده است؛ این حالت عملاً رخ نمی‌دهد و اگر رخ
		// دهد یعنی باینری خراب ساخته شده، پس خطای صریح بهتر از صفحهٔ سفید است.
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "رابط وب در این باینری جاسازی نشده است", http.StatusInternalServerError)
		})
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// صفحهٔ تک‌فایلی: هر مسیری که فایل نیست، همان رابط را می‌دهد تا بازخوانی مرورگر
		// یا مسیر تازه در آدرس، کاربر را به خطای ۴۰۴ نبرد.
		if _, err := fs.Stat(sub, cleanPath(r.URL.Path)); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

func cleanPath(p string) string {
	if p == "" || p == "/" {
		return "index.html"
	}
	if len(p) > 0 && p[0] == '/' {
		return p[1:]
	}
	return p
}
