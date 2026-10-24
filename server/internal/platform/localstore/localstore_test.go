// آزمون زیرساخت «localstore» در پلتفرم سرور — «localstore_test».

package localstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
)

// TestUsersFirstRunAndPasswordReset رفتار فایل کاربران را می‌سنجد: اجرای نخست یک مدیر
// می‌سازد، اجرای بعدی همان رمز را می‌پذیرد، و پرچم رمز، رمز را عوض می‌کند.
func TestUsersFirstRunAndPasswordReset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")

	accounts, created, err := Users(path, "", "")
	if err != nil {
		t.Fatalf("اجرای نخست فایل کاربران ناموفق: %v", err)
	}
	if !created {
		t.Fatal("اجرای نخست باید فایل کاربران را بسازد")
	}
	if len(accounts) != 1 || accounts[0].Username != DefaultAdminUser {
		t.Fatalf("کاربر پیش‌فرض ساخته نشد: %+v", accounts)
	}
	if accounts[0].User.Role != contracts.RoleAdmin {
		t.Fatalf("کاربر پیش‌فرض باید مدیر باشد، «%s» بود", accounts[0].User.Role)
	}

	// اجرای دوم: فایل خوانده می‌شود و رمز روی دیسک دست‌نخورده می‌ماند.
	again, createdAgain, err := Users(path, "", "")
	if err != nil {
		t.Fatalf("اجرای دوم فایل کاربران ناموفق: %v", err)
	}
	if createdAgain {
		t.Fatal("اجرای دوم نباید دوباره فایل کاربران را بسازد")
	}
	if again[0].PasswordHash != accounts[0].PasswordHash {
		t.Fatal("رمز کاربر باید بین اجراها پایدار بماند")
	}

	// بازنشانی رمز با پرچم اجرا: درهم‌سازی عوض می‌شود ولی نام کاربر همان می‌ماند.
	reset, _, err := Users(path, "", "newpass123")
	if err != nil {
		t.Fatalf("بازنشانی رمز ناموفق: %v", err)
	}
	if reset[0].PasswordHash == accounts[0].PasswordHash {
		t.Fatal("رمز باید با پرچم بازنشانی عوض شود")
	}
	if reset[0].Username != DefaultAdminUser {
		t.Fatalf("نام کاربر پس از بازنشانی باید همان بماند، «%s» شد", reset[0].Username)
	}

	// رمز متن ساده نباید در فایل بنشیند.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("خواندن فایل کاربران ناموفق: %v", err)
	}
	if strings.Contains(string(raw), "newpass123") {
		t.Fatal("رمز متن ساده در فایل کاربران ذخیره شده است")
	}
}

// TestUsersUnknownRoleFallsBackToEmployee نقش ناشناخته را می‌سنجد: یک اشتباه تایپی در
// فایل کاربران نباید دسترسی مدیریتی بسازد.
func TestUsersUnknownRoleFallsBackToEmployee(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.json")
	hash, err := auth.HashPassword("demo")
	if err != nil {
		t.Fatalf("درهم‌سازی رمز آزمون ناموفق: %v", err)
	}
	body := `{"users":[{"username":"op","role":"superuser","passwordHash":"` + hash + `"}]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("نوشتن فایل کاربران آزمون ناموفق: %v", err)
	}
	accounts, _, err := Users(path, "", "")
	if err != nil {
		t.Fatalf("خواندن فایل کاربران ناموفق: %v", err)
	}
	if accounts[0].User.Role != contracts.RoleEmployee {
		t.Fatalf("نقش ناشناخته باید به کارمند برگردد، «%s» شد", accounts[0].User.Role)
	}
}

// TestSecretIsStableAndKeepOutOfRepo می‌سنجد کلید امضا در پوشهٔ داده ساخته می‌شود و
// اجرای بعدی همان کلید را می‌خواند؛ وگرنه با هر راه‌اندازی همهٔ نشست‌ها باطل می‌شدند.
func TestSecretIsStableAndKeepOutOfRepo(t *testing.T) {
	dir := t.TempDir()
	first, err := Secret(dir)
	if err != nil {
		t.Fatalf("ساخت کلید ناموفق: %v", err)
	}
	if len(first) < 16 {
		t.Fatalf("کلید کوتاه است (%d نویسه)", len(first))
	}
	second, err := Secret(dir)
	if err != nil {
		t.Fatalf("خواندن کلید ناموفق: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("کلید امضا باید بین اجراها پایدار بماند")
	}
}

// TestStaffStoreReadsCSVAndFilters رفتار انبار کارکنان را می‌سنجد: ساخت فایل نمونه در
// اجرای نخست، تشخیص سرستون، فیلتر جستجو/واحد/وضعیت و مرتب‌سازی پایدار.
func TestStaffStoreReadsCSVAndFilters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "staff.csv")

	store, err := NewStaffStore(path)
	if err != nil {
		t.Fatalf("ساخت انبار کارکنان ناموفق: %v", err)
	}
	if store.Count() == 0 {
		t.Fatal("اجرای نخست باید فایل نمونهٔ کارکنان را بکارد")
	}

	page, err := store.List(context.Background(), contracts.EmployeeFilter{})
	if err != nil {
		t.Fatalf("فهرست کارکنان ناموفق: %v", err)
	}
	if page.Total != store.Count() {
		t.Fatalf("شمار فهرست (%d) با شمار انبار (%d) نمی‌خواند", page.Total, store.Count())
	}
	for i := 1; i < len(page.Items); i++ {
		if page.Items[i-1].ID > page.Items[i].ID {
			t.Fatalf("فهرست باید بر پایهٔ کد پرسنلی مرتب باشد: %s پیش از %s", page.Items[i-1].ID, page.Items[i].ID)
		}
	}

	// نام ستون‌های فارسی باید نگاشت شوند، نه اینکه به‌عنوان داده خوانده شوند.
	if got, _, err := store.Get(context.Background(), contracts.EmployeeFilter{}, "EMP-001"); err != nil || got.FullName == "" {
		t.Fatalf("کارمند نمونه خوانده نشد: %+v (خطا: %v)", got, err)
	}

	// فیلترها.
	byStatus, err := store.List(context.Background(), contracts.EmployeeFilter{Status: "active"})
	if err != nil {
		t.Fatalf("فیلتر وضعیت ناموفق: %v", err)
	}
	if byStatus.Total != page.Total {
		t.Fatalf("همهٔ کارکنان نمونه فعال‌اند؛ %d از %d برگشت", byStatus.Total, page.Total)
	}
	if none, _ := store.List(context.Background(), contracts.EmployeeFilter{Status: "terminated"}); none.Total != 0 {
		t.Fatalf("فیلتر وضعیت باید نتیجهٔ خالی بدهد، %d داد", none.Total)
	}
	byName, err := store.List(context.Background(), contracts.EmployeeFilter{Query: "سارا"})
	if err != nil {
		t.Fatalf("جستجو ناموفق: %v", err)
	}
	if byName.Total != 1 || byName.Items[0].ID != "EMP-001" {
		t.Fatalf("جستجوی نام باید یک نفر بدهد، %+v داد", byName.Items)
	}
	byDept, err := store.List(context.Background(), contracts.EmployeeFilter{DepartmentID: "بندر امام"})
	if err != nil {
		t.Fatalf("فیلتر واحد ناموفق: %v", err)
	}
	if byDept.Total == 0 {
		t.Fatal("فیلتر واحد بر پایهٔ نام فارسی نتیجه نداد")
	}
}

// TestStaffStoreReloadsOnFileChange می‌سنجد ویرایش فایل CSV بدون راه‌اندازی دوباره اثر
// می‌کند — همان چیزی که کاربر با Excel انجام می‌دهد.
func TestStaffStoreReloadsOnFileChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "staff.csv")
	content := "\uFEFFکد پرسنلی,نام و نام خانوادگی,واحد,سمت,وضعیت\nEMP-100,آزمون یک,واحد آزمون,کارشناس,فعال\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("نوشتن فایل کارکنان ناموفق: %v", err)
	}
	store, err := NewStaffStore(path)
	if err != nil {
		t.Fatalf("ساخت انبار ناموفق: %v", err)
	}
	if store.Count() != 1 {
		t.Fatalf("فایل دستی باید یک کارمند بدهد، %d داد", store.Count())
	}

	// حذف سطر نام‌دار و افزودن سطر تکراری: سطر بی‌نام نادیده می‌رود و کد تکراری رد می‌شود.
	content = content +
		"EMP-101,,واحد آزمون,کارشناس,فعال\n" +
		"EMP-100,آزمون دو,واحد آزمون,کارشناس,فعال\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("ویرایش فایل کارکنان ناموفق: %v", err)
	}
	// مهر زمانی فایل‌سیستم ویندوز ممکن است دقت کافی نداشته باشد؛ کمی صبر می‌کنیم تا
	// تغییر زمان فایل قطعی شود.
	waitForChange(t, path, store)

	if got, _ := store.List(context.Background(), contracts.EmployeeFilter{}); got.Total != 1 {
		t.Fatalf("سطر بی‌نام و کد تکراری نباید افزوده شوند، تعداد %d شد", got.Total)
	}
	if store.Warning() == "" {
		t.Fatal("سطر بی‌نام باید هشدار بدهد؛ کاربر باید بداند سطری نادیده رفته است")
	}
}

// waitForChange تا دیدن هشدار سطر بی‌نام صبر می‌کند: بارگذاری دوباره بر پایهٔ مهر زمانی
// فایل است و ساعت فایل‌سیستم ویندوز ممکن است تغییر کوچک را دیر ثبت کند.
func waitForChange(t *testing.T, path string, store *StaffStore) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if store.Warning() != "" {
			return
		}
		// تحریک خواندن دوباره؛ خودِ List همrefreshIfChanged را صدا می‌زند.
		if _, err := store.List(context.Background(), contracts.EmployeeFilter{}); err != nil {
			t.Fatalf("فهرست کارکنان پس از ویرایش فایل ناموفق: %v", err)
		}
		touch(t, path)
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("فایل کارکنان پس از ویرایش دوباره بارگذاری نشد: %s", path)
}

// touch مهر زمانی فایل را جلو می‌برد تا تشخیص تغییر قطعی شود.
func touch(t *testing.T, path string) {
	t.Helper()
	now := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("به‌روزرسانی زمان فایل کارکنان ناموفق: %v", err)
	}
}
