// سرآیندهای امنیتی پاسخ — بستن XSS، Clickjacking، MIME sniffing و ارجاع ناخواسته.

package websecurity

import "net/http"

// ContentSecurityPolicy سیاست محتوای پیش‌فرض سامانه.
//
// چرا این شکل:
//   - default-src 'self': هیچ منبع بیرونی (اسکریپت، تصویر، قالب، اتصال) به‌طور پیش‌فرض
//     بارگذاری نمی‌شود؛ سامانه روی یک مبدأ واحد اجرا می‌شود و به CDN بیرونی نیاز ندارد.
//   - script-src 'self': درج تگ <script> تازه — چه از XSS و چه از دادهٔ سرور — اجرا
//     نمی‌شود. این مؤثرترین لایهٔ دفاع در برابر XSS در همین سامانه است.
//   - script-src-attr 'unsafe-inline': رابط نمونهٔ فعلی صدها هندلر onclick تولیدشده در
//     قالب رشته‌ای دارد؛ تا مرحلهٔ دوم (جابه‌جایی به data-act) این اجازه لازم است. این
//     ضعیف‌ترین بخش سیاست است و در spec به‌عنوان ریسک باقی‌مانده ثبت شده است.
//   - style-src 'unsafe-inline': چیدمان زنده از style="..." استفاده می‌کند.
//   - frame-ancestors 'none' و X-Frame-Options: DENY: Clickjacking بسته می‌شود.
const ContentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"script-src-attr 'unsafe-inline'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; " +
	"font-src 'self' data:; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

// applyHeaders سرآیندهای امنیتی را روی پاسخ می‌نشاند.
//
// https یعنی درخواست از مسیر TLS رسیده (خود سرور یا پروکسی معتمد)؛ HSTS فقط در آن حالت
// فرستاده می‌شود، چون روی HTTP ساده بی‌اثر است و مرورگر نادیده می‌گیرد.
func applyHeaders(h http.Header, https bool, hsts bool) {
	h.Set("Content-Security-Policy", ContentSecurityPolicy)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=(), usb=()")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
	h.Set("X-Permitted-Cross-Domain-Policies", "none")
	if https && hsts {
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
}

// isHTTPS درخواست را از نظر TLS می‌سنجد؛ سرآیند X-Forwarded-Proto تنها وقتی باور
// می‌شود که درخواست از پروکسی معتمد رسیده باشد (همان قاعدهٔ ClientIP).
func isHTTPS(r *http.Request, trusted []string) bool {
	if r.TLS != nil {
		return true
	}
	if !isTrusted(hostOf(r.RemoteAddr), trusted) {
		return false
	}
	return equalFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
