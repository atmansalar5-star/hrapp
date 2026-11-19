// آزمون کنترل‌های لایهٔ وب: قفل حساب، محدودیت نرخ، سرآیندها و سیاست محتوا.
// ارجاع: docs/security-spec-fa.md (REQ-05، REQ-06، REQ-07، REQ-11).

package websecurity

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestREQ05AccountLockout می‌سنجد پس از سقف شکست پی‌درپی، حساب قفل می‌شود، مهلت که
// گذشت خودرها می‌شود و ورود موفق شمارش را صفر می‌کند.
func TestREQ05AccountLockout(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	guard := NewGuard(3, 10*time.Minute).WithClock(func() time.Time { return now })

	for i := 0; i < 2; i++ {
		if locked, _ := guard.Fail("Admin"); locked {
			t.Fatalf("قفل نباید پیش از سقف فعال شود (شکست %d)", i+1)
		}
	}
	locked, remaining := guard.Fail("admin")
	if !locked || remaining != 10*time.Minute {
		t.Fatalf("سومین شکست باید حساب را ۱۰ دقیقه قفل کند؛ locked=%v remaining=%v", locked, remaining)
	}
	if isLocked, wait := guard.Locked("ADMIN"); !isLocked || wait <= 0 {
		t.Fatalf("حساب باید قفل بماند (کلید بی‌توجه به بزرگی حروف): %v %v", isLocked, wait)
	}

	now = now.Add(11 * time.Minute)
	if isLocked, _ := guard.Locked("admin"); isLocked {
		t.Fatal("پس از پایان مهلت، حساب باید خودرها شود")
	}

	guard.Fail("admin")
	guard.Succeed("admin")
	if guard.Failures("admin") != 0 {
		t.Fatal("ورود موفق باید شمارش شکست را صفر کند")
	}
	// قفل خاموش (threshold صفر) هرگز فعال نمی‌شود.
	off := NewGuard(0, time.Minute)
	if locked, _ := off.Fail("x"); locked {
		t.Fatal("قفل خاموش نباید فعال شود")
	}
}

// TestREQ06LoginRateLimit می‌سنجد سقف تلاش ورود اعمال می‌شود، پس از پر شدن ۴۲۹ (با
// تصمیم limited) می‌دهد و کلید شمارش با جعل X-Forwarded-For عوض نمی‌شود.
func TestREQ06LoginRateLimit(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	limiter := NewLimiter(2, time.Minute).WithClock(func() time.Time { return now })
	if ok, _ := limiter.Allow("1.2.3.4"); !ok {
		t.Fatal("نخستین درخواست باید مجاز باشد")
	}
	if ok, _ := limiter.Allow("1.2.3.4"); !ok {
		t.Fatal("دومین درخواست باید مجاز باشد")
	}
	ok, retry := limiter.Allow("1.2.3.4")
	if ok {
		t.Fatal("سومین درخواست در همان بازه باید رد شود")
	}
	if retry <= 0 || retry > time.Minute {
		t.Fatalf("زمان انتظار پیشنهادی نامعقول است: %v", retry)
	}
	now = now.Add(time.Minute + time.Second)
	if ok, _ := limiter.Allow("1.2.3.4"); !ok {
		t.Fatal("پس از بازه، شمارش باید از نو شود")
	}

	security, warnings, err := New(Config{
		Enabled:     true,
		LoginRate:   "2/5m",
		GeneralRate: "600/1m",
		LockoutFor:  "15m",
	})
	if err != nil {
		t.Fatalf("ساخت لایهٔ امنیتی ناموفق: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("پیکربندی درست نباید هشدار بدهد: %v", warnings)
	}
	if d, _ := security.BeginLogin("5.6.7.8", "admin"); d != LoginAllow {
		t.Fatalf("نخستین ورود باید مجاز باشد: %s", d)
	}
	if d, _ := security.BeginLogin("5.6.7.8", "hr.manager"); d != LoginAllow {
		t.Fatalf("دومین ورود باید مجاز باشد: %s", d)
	}
	if d, wait := security.BeginLogin("5.6.7.8", "employee"); d != LoginLimited || wait <= 0 {
		t.Fatalf("سومین ورود از همان نشانی باید محدود شود: %s %v", d, wait)
	}

	// پروکسی معتمد: صرفاً نشانی‌های محلی؛ در غیر آن سرآیند جعلی نادیده گرفته می‌شود.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.9:5555"
	req.Header.Set("X-Forwarded-For", "192.0.2.1")
	if ip := ClientIP(req, DefaultTrustedProxies()); ip != "203.0.113.9" {
		t.Fatalf("سرآیند X-Forwarded-For از کلاینت نامعتمد باید نادیده گرفته شود: %s", ip)
	}
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "192.0.2.1")
	if ip := ClientIP(req, DefaultTrustedProxies()); ip != "192.0.2.1" {
		t.Fatalf("از پروکسی معتمد باید نشانی واقعی خوانده شود: %s", ip)
	}
}

// TestREQ07SecurityHeaders می‌سنجد سرآیندهای امنیتی روی پاسخ می‌نشینند، HSTS فقط روی
// HTTPS فرستاده می‌شود و پاسخ API کش نمی‌شود.
func TestREQ07SecurityHeaders(t *testing.T) {
	security, _, err := New(Config{Enabled: false, HSTS: true})
	if err != nil {
		t.Fatalf("ساخت لایهٔ امنیتی ناموفق: %v", err)
	}
	handler := security.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.RemoteAddr = "203.0.113.9:5555"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := map[string]string{
		"X-Content-Type-Options":            "nosniff",
		"X-Frame-Options":                   "DENY",
		"Referrer-Policy":                   "no-referrer",
		"Cross-Origin-Opener-Policy":        "same-origin",
		"X-Permitted-Cross-Domain-Policies": "none",
	}
	for name, value := range want {
		if got := rec.Header().Get(name); got != value {
			t.Fatalf("سرآیند %s باید «%s» باشد، «%s» بود", name, value, got)
		}
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("سیاست محتوا فرستاده نشد")
	}
	if got := rec.Header().Get("Permissions-Policy"); !strings.Contains(got, "microphone=()") {
		t.Fatalf("Permissions-Policy ناقص است: %s", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("پاسخ API نباید کش شود؛ Cache-Control=%q", got)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("HSTS روی HTTP ساده نباید فرستاده شود: %q", got)
	}

	// همان درخواست با TLS.
	tlsReq := httptest.NewRequest(http.MethodGet, "/", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	tlsRec := httptest.NewRecorder()
	handler.ServeHTTP(tlsRec, tlsReq)
	if got := tlsRec.Header().Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=") {
		t.Fatalf("روی HTTPS باید HSTS فرستاده شود: %q", got)
	}
}

// TestREQ11ContentSecurityPolicy می‌سنجد سیاست محتوا درج اسکریپت درون‌خطی و مبدأ بیرونی
// را می‌بندد و Clickjacking و تغییر base-uri را غیرممکن می‌کند.
func TestREQ11ContentSecurityPolicy(t *testing.T) {
	csp := ContentSecurityPolicy
	for _, required := range []string{
		"default-src 'self'",
		"script-src 'self'",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	} {
		if !strings.Contains(csp, required) {
			t.Fatalf("سیاست محتوا باید «%s» را داشته باشد: %s", required, csp)
		}
	}
	if strings.Contains(csp, "script-src *") || strings.Contains(csp, "script-src 'unsafe-eval'") {
		t.Fatalf("سیاست محتوا نباید اسکریپت از هر مبدأ یا eval را مجاز کند: %s", csp)
	}
	// بخش اسکریپت نباید unsafe-inline داشته باشد (فقط style و script-src-attr مجازند).
	for _, directive := range strings.Split(csp, ";") {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "script-src ") && strings.Contains(directive, "unsafe-inline") {
			t.Fatalf("script-src نباید unsafe-inline داشته باشد: %s", directive)
		}
	}
}
