// آزمون پیش‌فرض‌های پیکربندی عملیاتی (بند OPS-04 سند عملیاتی).

package main

import "testing"

// TestOPS04OperationalDefaults می‌سنجد پیش‌فرض‌ها «عملیاتی» هستند: بذرکاری دادهٔ نمونه
// و حساب‌های نمایشی خاموش‌اند و تنها با متغیر صریح روشن می‌شوند.
func TestOPS04OperationalDefaults(t *testing.T) {
	t.Setenv("HRMS_SEED_DEMO", "")
	t.Setenv("HRMS_DEV_ACCOUNTS", "")
	if envBool("HRMS_SEED_DEMO") {
		t.Fatal("بذرکاری دادهٔ نمونه باید در استقرار خاموش باشد")
	}
	if envBool("HRMS_DEV_ACCOUNTS") {
		t.Fatal("حساب‌های نمایشی باید در استقرار خاموش باشند")
	}
	if envBool("HRMS_SECURITY_OFF") {
		t.Fatal("محدودیت نرخ باید در استقرار روشن باشد")
	}

	t.Setenv("HRMS_SEED_DEMO", "0")
	if envBool("HRMS_SEED_DEMO") {
		t.Fatal("مقدار 0 باید خاموش بماند")
	}
	t.Setenv("HRMS_SEED_DEMO", "1")
	if !envBool("HRMS_SEED_DEMO") {
		t.Fatal("مقدار 1 باید روشن کند")
	}
}
