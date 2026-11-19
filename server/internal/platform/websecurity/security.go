// ترکیب کنترل‌های امنیتی لایهٔ وب: سرآیندها، محدودیت نرخ و قفل حساب ورود.

package websecurity

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// پیش‌فرض‌های استقرار. ۶۰۰ درخواست در دقیقه برای هر کلاینت یعنی رابط وب با همهٔ
// درخواست‌های همگام‌سازی‌اش راحت کار می‌کند، ولی سیل ساده را می‌بندد؛ ۲۰ تلاش ورود در
// ۵ دقیقه و قفل ۱۵ دقیقه‌ای پس از ۱۰ شکست، رمزشکنی آنلاین را عملاً بی‌ثمر می‌کند.
const (
	DefaultGeneralRate = "600/1m"
	DefaultLoginRate   = "20/5m"
	DefaultLockoutFor  = "15m"
	DefaultLockoutHits = 10
)

// Config تنظیمات لایهٔ امنیتی وب؛ مقدار صفر یعنی «خاموش» (فقط آزمون‌ها و اجرای
// درون‌حافظه‌ای) و cmd/hrmsd آن را روشن می‌کند.
type Config struct {
	// Enabled محدودیت نرخ و قفل حساب را روشن می‌کند. سرآیندهای امنیتی همیشه فرستاده
	// می‌شوند و به این پرچم وابسته نیستند.
	Enabled bool
	// TrustedProxies فهرست پروکسی‌های معتمد برای خواندن X-Forwarded-For/Proto؛
	// خالی یعنی فقط نشانی‌های محلی (nginx روی همان میزبان).
	TrustedProxies []string
	// GeneralRate سقف عمومی هر کلاینت به شکل «حداکثر/بازه»، مثل «600/1m».
	GeneralRate string
	// LoginRate سقف تلاش ورود هر کلاینت.
	LoginRate string
	// LockoutAfter شمار شکست پی‌درپی که حساب را قفل می‌کند؛ صفر یعنی پیش‌فرض.
	LockoutAfter int
	// LockoutFor مهلت قفل حساب، مثل «15m».
	LockoutFor string
	// HSTS ارسال Strict-Transport-Security روی درخواست‌های HTTPS؛ در استقرار TLS
	// باید روشن باشد.
	HSTS bool
}

// Security کنترل‌های لایهٔ وب یک استقرار.
type Security struct {
	cfg     Config
	trusted []string
	general *Limiter
	login   *Limiter
	guard   *Guard
}

// New از تنظیمات، لایهٔ امنیتی می‌سازد و هشدارهای پیکربندی را برمی‌گرداند (تا در
// راه‌اندازی لاگ شوند؛ خطای پیکربندی بدفرم خودش خطا است، نه هشدار).
func New(cfg Config) (*Security, []string, error) {
	var warnings []string

	trusted := ParseTrustedProxies(strings.Join(cfg.TrustedProxies, ","))
	if !cfg.Enabled {
		// حتی وقتی محدودیت نرخ خاموش است، سرآیندها و قاعدهٔ اعتماد پروکسی معنا دارند.
		return &Security{cfg: cfg, trusted: trusted}, warnings, nil
	}

	generalRate := orDefault(cfg.GeneralRate, DefaultGeneralRate)
	limit, windowSeconds, ok := ParseRate(generalRate)
	if !ok {
		return nil, nil, fmt.Errorf("سقف نرخ عمومی نامعتبر است: %q (نمونهٔ درست: 600/1m)", generalRate)
	}

	loginRate := orDefault(cfg.LoginRate, DefaultLoginRate)
	loginLimit, loginWindow, ok := ParseRate(loginRate)
	if !ok {
		return nil, nil, fmt.Errorf("سقف نرخ ورود نامعتبر است: %q (نمونهٔ درست: 20/5m)", loginRate)
	}

	lockoutFor, err := time.ParseDuration(orDefault(cfg.LockoutFor, DefaultLockoutFor))
	if err != nil || lockoutFor <= 0 {
		return nil, nil, fmt.Errorf("مهلت قفل حساب نامعتبر است: %q (نمونهٔ درست: 15m)", cfg.LockoutFor)
	}

	lockoutAfter := cfg.LockoutAfter
	if lockoutAfter <= 0 {
		lockoutAfter = DefaultLockoutHits
	}
	if lockoutAfter < 3 {
		warnings = append(warnings, "سقف شکست ورود کمتر از ۳ است؛ قفل حساب زودتر از حد معمول فعال می‌شود")
	}
	if loginLimit > limit {
		warnings = append(warnings, fmt.Sprintf("سقف نرخ ورود (%d) از سقف عمومی (%d) بیشتر است؛ سقف عمومی زودتر عمل می‌کند", loginLimit, limit))
	}

	return &Security{
		cfg:     cfg,
		trusted: trusted,
		general: NewLimiter(limit, time.Duration(windowSeconds)*time.Second),
		login:   NewLimiter(loginLimit, time.Duration(loginWindow)*time.Second),
		guard:   NewGuard(lockoutAfter, lockoutFor),
	}, warnings, nil
}

// Enabled آیا محدودیت نرخ روشن است.
func (s *Security) Enabled() bool { return s != nil && s.cfg.Enabled }

// Config تنظیمات فعال.
func (s *Security) Config() Config {
	if s == nil {
		return Config{}
	}
	return s.cfg
}

// TrustedProxies فهرست پروکسی‌های معتمد.
func (s *Security) TrustedProxies() []string {
	if s == nil {
		return DefaultTrustedProxies()
	}
	return s.trusted
}

// ClientIP نشانی کلاینت با قاعدهٔ اعتماد پروکسی.
func (s *Security) ClientIP(r *http.Request) string {
	return ClientIP(r, s.TrustedProxies())
}

// Limiter و Guard برای آزمون و گزارش وضعیت.
func (s *Security) Limiter() *Limiter {
	if s == nil {
		return nil
	}
	return s.general
}

// LoginLimiter محدودکنندهٔ تلاش ورود.
func (s *Security) LoginLimiter() *Limiter {
	if s == nil {
		return nil
	}
	return s.login
}

// Guard قفل‌کنندهٔ حساب.
func (s *Security) Guard() *Guard {
	if s == nil {
		return nil
	}
	return s.guard
}

// Middleware سرآیندهای امنیتی را روی همهٔ پاسخ‌ها می‌گذارد و (اگر روشن باشد) سقف
// عمومی نرخ را اعمال می‌کند.
//
// سرآیندها بیرون از محدودیت نرخ هم اعمال می‌شوند: پاسخ ۴۲۹ هم باید همان سیاست را
// داشته باشد، وگرنه سیل درخواست به سادگی پاسخ بدون CSP می‌گیرد.
func (s *Security) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s == nil {
			next.ServeHTTP(w, r)
			return
		}
		applyHeaders(w.Header(), isHTTPS(r, s.trusted), s.cfg.HSTS)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if s.cfg.Enabled && s.general != nil {
			ip := s.ClientIP(r)
			if allowed, retry := s.general.Allow(ip); !allowed {
				WriteRetryAfter(w, retry, "تعداد درخواست‌های شما بیش از حد مجاز است؛ کمی بعد دوباره تلاش کنید.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// تصمیم‌های ورود که به لاگ حسابرسی و پاسخ ۴۲۹ نگاشت می‌شوند.
const (
	LoginAllow      = "allow"
	LoginLocked     = "locked"
	LoginLimited    = "limited"
	LoginLockedFa   = "حساب شما پس از چند تلاش ناموفق موقتاً قفل شده است؛ بعداً دوباره تلاش کنید."
	LoginLimitedFa  = "تلاش ورود بیش از حد مجاز از این نشانی است؛ کمی بعد دوباره تلاش کنید."
	LoginAttemptLog = "login-throttled"
)

// BeginLogin پیش از سنجش رمز اجرا می‌شود: اگر حساب قفل است یا نشانی سقف تلاش ورود را
// پر کرده، تصمیم مسدود می‌گیرد.
//
// ترتیب عمدی: قفل حساب پیش از محدودیت نرخ سنجیده می‌شود تا پاسخ برای حساب قفل‌شده
// صریح‌تر و برای بقیه بی‌اثر باشد؛ و شمارش نرخ فقط با «تلاش» انجام می‌شود، نه با هر
// درخواست دیگر.
func (s *Security) BeginLogin(ip, username string) (decision string, retry time.Duration) {
	if !s.Enabled() {
		return LoginAllow, 0
	}
	if locked, wait := s.guard.Locked(username); locked {
		return LoginLocked, wait
	}
	if allowed, wait := s.login.Allow(ip); !allowed {
		return LoginLimited, wait
	}
	return LoginAllow, 0
}

// EndLogin نتیجهٔ سنجش رمز را به قفل حساب و شمارش نرخ می‌رساند.
func (s *Security) EndLogin(username string, success bool) (locked bool, remaining time.Duration) {
	if !s.Enabled() {
		return false, 0
	}
	if success {
		s.guard.Succeed(username)
		return false, 0
	}
	return s.guard.Fail(username)
}

func orDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
