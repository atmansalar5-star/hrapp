// آزمون پیش‌فرض‌های امنیتی راه‌اندازی سرور (بند REQ-06 و REQ-09 اسپک امنیتی).

package main

import (
	"testing"

	"hrms/internal/platform/websecurity"
)

// TestREQ06DeploymentDefaults می‌سنجد در استقرار، محدودیت نرخ و قفل حساب به‌طور
// پیش‌فرض روشن‌اند و مقادیر پیش‌فرض از constantهای بستهٔ امنیتی می‌آیند.
func TestREQ06DeploymentDefaults(t *testing.T) {
	for _, name := range []string{
		"HRMS_SECURITY_OFF", "HRMS_RATE_LIMIT", "HRMS_LOGIN_RATE",
		"HRMS_LOGIN_LOCKOUT_AFTER", "HRMS_LOGIN_LOCKOUT_FOR", "HRMS_TRUSTED_PROXIES",
	} {
		t.Setenv(name, "")
	}
	cfg := securityConfig()
	if !cfg.Enabled {
		t.Fatal("محدودیت نرخ باید در استقرار به‌طور پیش‌فرض روشن باشد")
	}
	if cfg.LockoutAfter != websecurity.DefaultLockoutHits {
		t.Fatalf("سقف شکست ورود باید %d باشد، %d شد", websecurity.DefaultLockoutHits, cfg.LockoutAfter)
	}
	if len(cfg.TrustedProxies) == 0 {
		t.Fatal("فهرست پروکسی معتمد نباید خالی بماند (پیش‌فرض: نشانی‌های محلی)")
	}

	// خاموش‌کردن باید صریح باشد.
	t.Setenv("HRMS_SECURITY_OFF", "1")
	if securityConfig().Enabled {
		t.Fatal("HRMS_SECURITY_OFF=1 باید محدودیت نرخ را خاموش کند")
	}
}

// TestREQ09ServerBounds می‌سنجد سقف سرآیند درخواست کوچک و صریح تنظیم شده است.
func TestREQ09ServerBounds(t *testing.T) {
	if maxHeaderBytes != 64<<10 {
		t.Fatalf("سقف سرآیند باید ۶۴ کیلوبایت باشد، %d است", maxHeaderBytes)
	}
}
