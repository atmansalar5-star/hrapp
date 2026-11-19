// آزمون جریان امنیتی ورود روی هندلر کامل: سقف تلاش، پاسخ ۴۲۹ و ثبت در لاگ حسابرسی.
// ارجاع: docs/security-spec-fa.md بند REQ-12.

package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/platform/websecurity"
)

// TestREQ12LoginThrottleIsAudited می‌سنجد پس از پر شدن سقف تلاش ورود، پاسخ ۴۲۹ با
// سرآیند Retry-After و بدنهٔ استاندارد برگردد، سرآیندهای امنیتی روی همان پاسخ هم
// بنشینند و رخدادهای «ورود ناموفق» و «مسدود شده» در زنجیرهٔ حسابرسی ثبت شوند.
func TestREQ12LoginThrottleIsAudited(t *testing.T) {
	deps, err := NewDeps(Config{
		JWTSecret: []byte("test-secret-key-for-security-flow"),
		DataDir:   t.TempDir(),
		Security: websecurity.Config{
			Enabled:      true,
			LoginRate:    "2/5m",
			GeneralRate:  "600/1m",
			LockoutAfter: 5,
			LockoutFor:   "15m",
		},
	})
	if err != nil {
		t.Fatalf("ساخت وابستگی‌ها ناموفق: %v", err)
	}
	handler, _, err := NewHandler(deps)
	if err != nil {
		t.Fatalf("ساخت هندلر ناموفق: %v", err)
	}

	wrong := map[string]string{"username": "admin", "password": "definitely-wrong"}
	for attempt := 1; attempt <= 2; attempt++ {
		rec := call(t, handler, http.MethodPost, "/api/v1/auth/login", "", wrong)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("تلاش %d باید ۴۰۱ بدهد، %d داد: %s", attempt, rec.Code, rec.Body.String())
		}
	}

	blocked := call(t, handler, http.MethodPost, "/api/v1/auth/login", "", wrong)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("تلاش سوم باید ۴۲۹ بدهد، %d داد: %s", blocked.Code, blocked.Body.String())
	}
	if blocked.Header().Get("Retry-After") == "" {
		t.Fatal("پاسخ ۴۲۹ باید سرآیند Retry-After داشته باشد")
	}
	if blocked.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("سرآیندهای امنیتی باید روی پاسخ ۴۲۹ هم بنشینند")
	}
	if body := blocked.Body.String(); !strings.Contains(body, "rate_limited") {
		t.Fatalf("کد خطای پاسخ باید rate_limited باشد: %s", body)
	}

	actions := map[string]bool{}
	for _, event := range deps.Audit.All() {
		actions[event.Action] = true
	}
	if !actions["login-failed"] {
		t.Fatal("ورود ناموفق باید در لاگ حسابرسی ثبت شود")
	}
	if !actions["login-throttled"] {
		t.Fatal("مسدود شدن تلاش ورود باید در لاگ حسابرسی ثبت شود")
	}

	// ورود درست پس از پر شدن سقف هم مسدود است؛ سقف به IP بسته است نه به رمز درست.
	rec := call(t, handler, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"username": "admin", "password": "demo"})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("پس از پر شدن سقف، حتی رمز درست هم نباید بی‌درنگ پذیرفته شود؛ کد %d", rec.Code)
	}
}

// TestREQ07SecurityHeadersOnStaticAlso می‌سنجد سرآیندهای امنیتی روی پاسخ‌های API و
// مسیرهای ۴۰۴ هم فرستاده می‌شوند (لایهٔ بیرونی، نه وابسته به ماژول).
func TestREQ07SecurityHeadersOnStaticAlso(t *testing.T) {
	handler := newTestServer(t)
	rec := call(t, handler, http.MethodGet, "/api/v1/does-not-exist", "", nil)
	for _, name := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy"} {
		if rec.Header().Get(name) == "" {
			t.Fatalf("سرآیند امنیتی %s روی پاسخ ۴۰۴ فرستاده نشد", name)
		}
	}
}
