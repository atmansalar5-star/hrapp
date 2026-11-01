package photos

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// pngSample یک تصویر PNG کوچک و معتبر می‌سازد.
func pngSample() []byte {
	return []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 13}
}

func newStore(t *testing.T) *Store {
	t.Helper()
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("ساخت انبار عکس ناموفق بود: %v", err)
	}
	return store
}

// TestSaveAndGet عکس ذخیره می‌شود، در نگاشت می‌ماند و پس از باز کردن دوبارهٔ انبار
// (مثل راه‌اندازی مجدد سرور) هم پیدا می‌شود.
func TestSaveAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("ساخت انبار ناموفق بود: %v", err)
	}
	at := time.Date(2026, 10, 28, 9, 30, 0, 0, time.UTC)
	saved, err := store.Save("EMP-001", "کارت ملی.png", pngSample(), "hr.manager", at)
	if err != nil {
		t.Fatalf("ذخیرهٔ عکس ناموفق بود: %v", err)
	}
	if !strings.HasPrefix(saved.Name, "EMP-001-") || !strings.HasSuffix(saved.Name, ".png") {
		t.Fatalf("نام فایل عکس انتظار می‌رفت با کد پرسنلی شروع شود، %q آمد", saved.Name)
	}
	if saved.ContentType != "image/png" || saved.SizeBytes != len(pngSample()) {
		t.Fatalf("اطلاعات فایل درست نیست: %+v", saved)
	}
	if saved.UpdatedAt != "1405/08/06" {
		t.Fatalf("تاریخ شمسی عکس انتظار ۱۴۰۵/۰۸/۰۶ بود، %q آمد", saved.UpdatedAt)
	}
	if saved.OriginalName != "کارت ملی.png" {
		t.Fatalf("نام اصلی فایل باید نگه داشته شود، %q آمد", saved.OriginalName)
	}

	// باز کردن انبار تازه روی همان پوشه: نگاشت باید از دیسک خوانده شود.
	again, err := New(dir)
	if err != nil {
		t.Fatalf("بازکردن دوبارهٔ انبار ناموفق بود: %v", err)
	}
	got, ok, err := again.Get("EMP-001")
	if err != nil || !ok {
		t.Fatalf("عکس ذخیره‌شده پس از بازکردن دوباره پیدا نشد (ok=%v, err=%v)", ok, err)
	}
	if got.Name != saved.Name {
		t.Fatalf("نام فایل بازخوانی‌شده %q با ذخیره‌شده %q یکی نیست", got.Name, saved.Name)
	}
	if _, ok, _ := again.Get("EMP-002"); ok {
		t.Fatal("کارمند بدون عکس نباید عکس برگرداند")
	}
}

// TestSaveReplacesPreviousFile ذخیرهٔ عکس تازه فایل قبلی همان کارمند را پاک می‌کند.
func TestSaveReplacesPreviousFile(t *testing.T) {
	store := newStore(t)
	first, err := store.Save("EMP-002", "a.png", pngSample(), "hr.spec", time.Now())
	if err != nil {
		t.Fatalf("ذخیرهٔ نخست ناموفق بود: %v", err)
	}
	second, err := store.Save("EMP-002", "b.png", append(pngSample(), 1, 2, 3), "hr.spec", time.Now())
	if err != nil {
		t.Fatalf("ذخیرهٔ دوم ناموفق بود: %v", err)
	}
	if first.Name == second.Name {
		t.Fatal("تصویر تازه باید نام تازه بگیرد تا کش مرورگر باطل شود")
	}
	if store.Exists(first.Name) {
		t.Fatal("فایل عکس پیشین باید پاک شده باشد")
	}
	if !store.Exists(second.Name) {
		t.Fatal("فایل عکس تازه باید موجود باشد")
	}
}

// TestRejectInvalid فایل غیرتصویری، SVG، فایل خالی و کد پرسنلی ناامن پذیرفته نمی‌شود.
func TestRejectInvalid(t *testing.T) {
	store := newStore(t)
	cases := []struct {
		name       string
		employeeID string
		data       []byte
	}{
		{"متن ساده", "EMP-003", []byte("سلام، این عکس نیست")},
		{"SVG", "EMP-003", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		{"فایل خالی", "EMP-003", nil},
		{"کد ناامن", "../etc", pngSample()},
		{"کد خالی", "  ", pngSample()},
	}
	for _, c := range cases {
		if _, err := store.Save(c.employeeID, "x", c.data, "tester", time.Now()); err == nil {
			t.Fatalf("«%s» باید رد می‌شد", c.name)
		}
	}
}

// TestRemove عکس حذف می‌شود و فایلش هم پاک می‌شود.
func TestRemove(t *testing.T) {
	store := newStore(t)
	saved, err := store.Save("EMP-004", "a.jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0}, "hr.spec", time.Now())
	if err != nil {
		t.Fatalf("ذخیره ناموفق بود: %v", err)
	}
	if saved.ContentType != "image/jpeg" {
		t.Fatalf("نوع JPEG انتظار می‌رفت، %q آمد", saved.ContentType)
	}
	if err := store.Remove("EMP-004"); err != nil {
		t.Fatalf("حذف ناموفق بود: %v", err)
	}
	if _, ok, _ := store.Get("EMP-004"); ok {
		t.Fatal("پس از حذف نباید عکسی برگردد")
	}
	if _, err := os.Stat(filepath.Join(store.Dir(), saved.Name)); !os.IsNotExist(err) {
		t.Fatal("فایل عکس باید از دیسک پاک شده باشد")
	}
	// حذف دوباره نباید خطا بدهد.
	if err := store.Remove("EMP-004"); err != nil {
		t.Fatalf("حذف دوباره نباید خطا بدهد: %v", err)
	}
}

// TestSignedURLCheck نشانی عکس با امضا معتبر است و با شناسهٔ کارمند دیگر یا نام دیگر باطل می‌شود.
func TestSignedURLCheck(t *testing.T) {
	store := newStore(t)
	saved, err := store.Save("EMP-005", "a.png", pngSample(), "hr.spec", time.Now())
	if err != nil {
		t.Fatalf("ذخیره ناموفق بود: %v", err)
	}
	key := []byte("کلید-آزمون")
	raw := URL(saved.Name, "EMP-005", key)
	if !strings.Contains(raw, saved.Name) || !strings.Contains(raw, "t=") || !strings.Contains(raw, "e=EMP-005") {
		t.Fatalf("نشانی عکس ناقص است: %s", raw)
	}
	token := raw[strings.Index(raw, "t=")+2:]
	if !Check(key, saved.Name, "EMP-005", token) {
		t.Fatal("امضای نشست خودِ کارمند باید معتبر باشد")
	}
	if Check(key, saved.Name, "EMP-006", token) {
		t.Fatal("امضای عکس یک کارمند نباید برای کارمند دیگر معتبر باشد")
	}
	if Check([]byte("کلید-دیگر"), saved.Name, "EMP-005", token) {
		t.Fatal("امضا با کلید دیگر نباید معتبر باشد")
	}
	if URL(saved.Name, "EMP-005", key) == raw &&
		NameFromURL(raw) != saved.Name {
		t.Fatalf("نام فایل از نشانی درست استخراج نشد: %q", NameFromURL(raw))
	}
}

// TestOpenAndTraversalServing نام ناامن خوانده نمی‌شود.
func TestOpenAndTraversalServing(t *testing.T) {
	store := newStore(t)
	saved, err := store.Save("EMP-007", "a.png", pngSample(), "hr.spec", time.Now())
	if err != nil {
		t.Fatalf("ذخیره ناموفق بود: %v", err)
	}
	handle, contentType, err := store.Open(saved.Name)
	if err != nil {
		t.Fatalf("بازکردن فایل عکس ناموفق بود: %v", err)
	}
	defer handle.Close()
	if contentType != "image/png" {
		t.Fatalf("نوع محتوا %q آمد", contentType)
	}
	for _, bad := range []string{"../", "..\\", indexFile, "a/b.png", ""} {
		if _, _, err := store.Open(bad); err == nil {
			t.Fatalf("نام ناامن %q باید رد می‌شد", bad)
		}
	}
}

// TestEmptyDataDir انبار بدون پوشهٔ داده ساخته نمی‌شود.
func TestEmptyDataDir(t *testing.T) {
	if _, err := New("   "); err == nil {
		t.Fatal("پوشهٔ دادهٔ خالی باید خطا بدهد")
	}
}

// TestAllOnlyExisting فهرست کامل فقط عکس‌های موجود را برمی‌گرداند.
func TestAllOnlyExisting(t *testing.T) {
	store := newStore(t)
	first, err := store.Save("EMP-008", "a.png", pngSample(), "hr.spec", time.Now())
	if err != nil {
		t.Fatalf("ذخیره ناموفق بود: %v", err)
	}
	if _, err := store.Save("EMP-009", "b.png", append(pngSample(), 9), "hr.spec", time.Now()); err != nil {
		t.Fatalf("ذخیره ناموفق بود: %v", err)
	}
	// فایل یکی از عکس‌ها دستی پاک می‌شود؛ باید از فهرست هم بیرون برود.
	if err := os.Remove(filepath.Join(store.Dir(), first.Name)); err != nil {
		t.Fatalf("پاک‌کردن فایل آزمون ناموفق بود: %v", err)
	}
	all, err := store.All()
	if err != nil {
		t.Fatalf("خواندن فهرست ناموفق بود: %v", err)
	}
	if _, ok := all["EMP-008"]; ok {
		t.Fatal("عکسِ فایل‌پاک‌شده نباید در فهرست باشد")
	}
	if _, ok := all["EMP-009"]; !ok {
		t.Fatal("عکس موجود باید در فهرست باشد")
	}
	if len(bytes.TrimSpace([]byte(store.Dir()))) == 0 {
		t.Fatal("پوشهٔ عکس‌ها باید مسیر داشته باشد")
	}
}
