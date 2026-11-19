// Package websecurity سخت‌سازی لایهٔ وب: محدودیت نرخ، قفل حساب و سرآیندهای امنیتی.
//
// چرا یک بستهٔ جدا: این‌ها هیچ‌کدام «قاعدهٔ کسب‌وکار» نیستند و به هیچ ماژولی وابسته
// نیستند؛ مثل لاگ حسابرسی، دارایی پلتفرم‌اند. اگر در bootstrap نوشته می‌شدند، آزمون
// رفتارشان (شمارش نرخ، مهلت قفل، سرآیندها) به کل سرور گره می‌خورد.
//
// نگاشت هر کنترل به تهدید:
//   - Limiter: حملهٔ فرهنگ‌نامه‌ای رمز، اسپم درخواست و اشغال منابع (DoS سبک) را کند می‌کند.
//   - Guard: پس از چند شکست پیاپی، حساب (نه فقط IP) را موقتاً می‌بندد تا رمزشکنی
//     توزیع‌شده از چند IP هم گران شود.
//   - Headers: XSS، Clickjacking، MIME sniffing و ارجاع‌دهی ناخواسته را می‌بندد.
//
// این کنترل‌ها «کامل‌کنندهٔ» کنترل‌های استقرارند، نه جایگزین آن‌ها: TLS، دیوار آتش و
// سخت‌سازی پایگاه‌داده در سند `docs/security-hardening-fa.md` آمده‌اند.
package websecurity

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// ClientIP نشانی واقعی کلاینت را برمی‌گرداند.
//
// قاعدهٔ اعتماد عمدی و سخت است: سرآیند X-Forwarded-For تنها وقتی باور می‌شود که
// درخواست از یک پروکسی معتمد (پیش‌فرض: نشانی‌های محلی) رسیده باشد. اگر سرور مستقیم
// روی اینترنت باشد، RemoteAddr خودش نشانی واقعی است و مقدار سرآیند — که کلاینت
// می‌تواند جعلش کند — نادیده گرفته می‌شود؛ وگرنه با گرداندن آن سرآیند می‌شد
// محدودیت نرخ را دور زد.
func ClientIP(r *http.Request, trustedProxies []string) string {
	remote := hostOf(r.RemoteAddr)
	if !isTrusted(remote, trustedProxies) {
		return remote
	}
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded == "" {
		return remote
	}
	// نخستین مقدار زنجیره، کلاینت اصلی است.
	first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
	if first == "" {
		return remote
	}
	return hostOf(first)
}

// hostOf بخش میزبان را از host:port برمی‌دارد.
func hostOf(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	return value
}

// isTrusted آیا نشانی در فهرست پروکسی‌های معتمد است (تطابق کامل یا ردهٔ CIDR).
func isTrusted(ip string, trusted []string) bool {
	for _, entry := range trusted {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if entry == ip {
			return true
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			if parsed := net.ParseIP(ip); parsed != nil && network.Contains(parsed) {
				return true
			}
		}
	}
	return false
}

// DefaultTrustedProxies پروکسی‌های معتمد پیش‌فرض: فقط نشانی‌های محلی.
//
// چرا: در استقرار معمول این ماشین، nginx روی همان میزبان است و RemoteAddr همهٔ
// درخواست‌ها 127.0.0.1 می‌شود؛ در آن حالت باید نشانی واقعی از X-Forwarded-For خوانده
// شود. اگر سرور بدون پروکسی مستقیم در معرض باشد، RemoteAddr عمومی است و سرآیند
// نادیده گرفته می‌شود.
func DefaultTrustedProxies() []string { return []string{"127.0.0.1", "::1"} }

// ParseTrustedProxies فهرست جدا‌شده با کاما را می‌خواند؛ خالی یعنی پیش‌فرض.
func ParseTrustedProxies(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultTrustedProxies()
	}
	out := make([]string, 0, 4)
	for _, part := range strings.Split(value, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// errBadWindow بازهٔ نامعتبر در رشتهٔ نرخ.
var errBadWindow = errors.New("بازهٔ زمانی نامعتبر است")

// ParseRate رشتهٔ «حداکثر/بازه» را می‌خواند، مثل «600/1m» یا «20/5m».
// خروجی: limit و windowSeconds؛ مقدار نامعتبر ok=false برمی‌گرداند.
func ParseRate(value string) (limit int, windowSeconds int, ok bool) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 {
		return 0, 0, false
	}
	limit, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || limit <= 0 {
		return 0, 0, false
	}
	window, err := parseWindow(strings.TrimSpace(parts[1]))
	if err != nil || window <= 0 {
		return 0, 0, false
	}
	return limit, window, true
}

// parseWindow بازه را می‌خواند: «30s»، «1m»، «15m»، «1h» یا عدد خام به‌معنی ثانیه.
func parseWindow(value string) (int, error) {
	if value == "" {
		return 0, errBadWindow
	}
	unit := value[len(value)-1]
	number := value
	multiplier := 1
	switch unit {
	case 's', 'S':
		number = value[:len(value)-1]
	case 'm', 'M':
		number = value[:len(value)-1]
		multiplier = 60
	case 'h', 'H':
		number = value[:len(value)-1]
		multiplier = 3600
	}
	n, err := strconv.Atoi(strings.TrimSpace(number))
	if err != nil {
		return 0, errBadWindow
	}
	return n * multiplier, nil
}
