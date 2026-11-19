// آزمون انتخاب حساب‌های عملیاتی: بدون داده/حساب نمایشی، رمز قوی در نخستین اجرا.
// ارجاع: docs/operational-spec-fa.md (بندهای OPS-02 و OPS-03).

package bootstrap

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hrms/internal/platform/auth"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestOPS02DemoAccountsOnlyWhenAllowed می‌سنجد با پایگاه‌دادهٔ واقعی، حساب‌های نمایشی
// (admin/demo) فقط با اجازهٔ صریح ساخته شوند؛ در غیر این صورت فایل کاربران عملیاتی
// ملاک است.
func TestOPS02DemoAccountsOnlyWhenAllowed(t *testing.T) {
	dir := t.TempDir()

	demo, err := operationalAccounts(Config{
		DatabaseURL:       "postgres://hrms@127.0.0.1:5432/hrms?sslmode=disable",
		AllowDemoAccounts: true,
	}, dir, quietLogger())
	if err != nil {
		t.Fatalf("حالت نمایشی نباید خطا بدهد: %v", err)
	}
	if len(demo) != 5 {
		t.Fatalf("حالت نمایشی باید ۵ حساب نمونه بدهد، %d داد", len(demo))
	}

	operational, err := operationalAccounts(Config{
		DatabaseURL: "postgres://hrms@127.0.0.1:5432/hrms?sslmode=disable",
	}, dir, quietLogger())
	if err != nil {
		t.Fatalf("حالت عملیاتی نباید خطا بدهد: %v", err)
	}
	if len(operational) != 1 {
		t.Fatalf("نصب تازه باید فقط حساب مدیر بسازد، %d حساب داد", len(operational))
	}
	if operational[0].Username != "admin" {
		t.Fatalf("نخستین حساب باید مدیر باشد: %s", operational[0].Username)
	}
	for _, account := range operational {
		if account.Username == "employee" || account.Username == "hr.manager" {
			t.Fatalf("حساب نمایشی «%s» نباید در استقرار عملیاتی وجود داشته باشد", account.Username)
		}
	}
	if _, statErr := os.Stat(filepath.Join(dir, "users.json")); statErr != nil {
		t.Fatalf("فایل کاربران عملیاتی ساخته نشد: %v", statErr)
	}
}

// TestOPS03OperationalAccountsFromFile می‌سنجد حساب راهبر با رمز انتخابی ساخته شود،
// رمز خام در فایل نباشد و ورود با آن رمز کار کند.
func TestOPS03OperationalAccountsFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops-users.json")
	const password = "Bandar-1405-Ops!"

	accounts, err := operationalAccounts(Config{
		DatabaseURL:   "postgres://hrms@127.0.0.1:5432/hrms?sslmode=disable",
		UsersFile:     path,
		AdminUser:     "ops.admin",
		AdminPassword: password,
	}, dir, quietLogger())
	if err != nil {
		t.Fatalf("ساخت حساب عملیاتی ناموفق: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Username != "ops.admin" {
		t.Fatalf("حساب راهبر ساخته نشد: %+v", accounts)
	}
	if !strings.HasPrefix(accounts[0].PasswordHash, "$argon2id$") {
		t.Fatalf("رمز باید Argon2id درهم شود: %s", accounts[0].PasswordHash)
	}
	if !auth.VerifyPassword(accounts[0].PasswordHash, password) {
		t.Fatal("رمز انتخابی راهبر باید پذیرفته شود")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("خواندن فایل کاربران ناموفق: %v", err)
	}
	if strings.Contains(string(raw), password) {
		t.Fatal("رمز خام در فایل کاربران ذخیره شده است")
	}

	// اجرای دوم: همان رمز روی دیسک می‌ماند و حساب تکرار نمی‌شود.
	again, err := operationalAccounts(Config{
		DatabaseURL: "postgres://hrms@127.0.0.1:5432/hrms?sslmode=disable",
		UsersFile:   path,
	}, dir, quietLogger())
	if err != nil {
		t.Fatalf("اجرای دوم ناموفق: %v", err)
	}
	if len(again) != 1 || again[0].PasswordHash != accounts[0].PasswordHash {
		t.Fatal("رمز کاربر باید بین اجراها پایدار بماند")
	}
}

// TestOPS03WeakAdminPasswordRejected می‌سنجد رمز پیش‌فرض/ضعیف راهبر در مسیر
// راه‌اندازی عملیاتی رد شود (نه اینکه فقط در سند هشدار داده شود).
func TestOPS03WeakAdminPasswordRejected(t *testing.T) {
	dir := t.TempDir()
	for _, weak := range []string{"6468", "demo", "admin", "12345678"} {
		_, err := operationalAccounts(Config{
			DatabaseURL:   "postgres://hrms@127.0.0.1:5432/hrms?sslmode=disable",
			UsersFile:     filepath.Join(dir, "weak-"+weak+".json"),
			AdminPassword: weak,
		}, dir, quietLogger())
		if err == nil {
			t.Fatalf("رمز ضعیف «%s» نباید پذیرفته شود", weak)
		}
	}
}

// TestOPS03RandomFirstRunPassword می‌سنجد نصب تازه با رمز تصادفی بالا بیاید، نه با
// رمز پیش‌فرض مستندشدهٔ 6468.
func TestOPS03RandomFirstRunPassword(t *testing.T) {
	dir := t.TempDir()
	accounts, err := operationalAccounts(Config{
		DatabaseURL: "postgres://hrms@127.0.0.1:5432/hrms?sslmode=disable",
	}, dir, quietLogger())
	if err != nil {
		t.Fatalf("راه‌اندازی نخست ناموفق: %v", err)
	}
	for _, account := range accounts {
		if auth.VerifyPassword(account.PasswordHash, "6468") || auth.VerifyPassword(account.PasswordHash, "demo") {
			t.Fatalf("حساب «%s» با رمز پیش‌فرض بالا آمده است", account.Username)
		}
	}
	if len(accounts[0].PasswordHash) < 30 {
		t.Fatal("درهم رمز معتبر ساخته نشد")
	}
}
