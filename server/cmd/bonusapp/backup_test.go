//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestFile یک فایل آزمون با محتوای داده‌شده می‌سازد.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("ساخت پوشهٔ «%s» ناموفق: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("نوشتن «%s» ناموفق: %v", path, err)
	}
}

// TestBackupDataCopiesUserData می‌سنجد پشتیبان، چهار فایل داده را کامل کپی می‌کند.
//
// چرا این آزمون مهم است: دکمهٔ «ساخت پشتیبان» تنها راه کاربر برای امن‌کردن داده است؛
// اگر بی‌صدا ناقص بسازد، کاربر با پشتیبان معیوب روبه‌رو می‌شود بدون آنکه بداند.
func TestBackupDataCopiesUserData(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "bonus-data")
	backupRoot := filepath.Join(root, "backup")

	writeTestFile(t, filepath.Join(dataDir, "records.json"), `{"version":1,"records":[]}`)
	writeTestFile(t, filepath.Join(dataDir, "users.json"), `{"users":[]}`)
	writeTestFile(t, filepath.Join(dataDir, "staff.csv"), "\uFEFFکد پرسنلی,نام\nEMP-001,سارا\n")
	writeTestFile(t, filepath.Join(dataDir, "secret.key"), "abcdefghijklmnop")
	writeTestFile(t, filepath.Join(dataDir, "bonusapp.log"), "log line\n")

	result, err := backupData(dataDir, backupRoot)
	if err != nil {
		t.Fatalf("پشتیبان‌گیری ناموفق: %v", err)
	}
	if result.Files != 5 {
		t.Fatalf("پشتیبان باید ۵ فایل داشته باشد، %d داشت", result.Files)
	}
	if result.Bytes == 0 {
		t.Fatal("شمار بایت‌های کپی‌شده باید بزرگ‌تر از صفر باشد")
	}
	if !strings.HasPrefix(filepath.Base(result.Path), "bonus-data-") {
		t.Fatalf("نام پوشهٔ پشتیبان باید زمان‌دار باشد: %s", result.Path)
	}
	for _, name := range []string{"records.json", "users.json", "staff.csv", "secret.key", "bonusapp.log"} {
		source, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			t.Fatalf("خواندن مبدأ «%s» ناموفق: %v", name, err)
		}
		copied, err := os.ReadFile(filepath.Join(result.Path, name))
		if err != nil {
			t.Fatalf("فایل «%s» در پشتیبان نیست: %v", name, err)
		}
		if string(source) != string(copied) {
			t.Fatalf("محتوای «%s» در پشتیبان با مبدأ یکی نیست", name)
		}
	}
}

// TestBackupDataSkipsBrowserProfile می‌سنجد پروفایل داخلی موتور مرورگر و پوشهٔ خودِ
// پشتیبان‌ها در پشتیبان نمی‌آید (بی‌ارزش و چند ده مگابایت).
func TestBackupDataSkipsBrowserProfile(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "bonus-data")
	backupRoot := filepath.Join(root, "backup")

	writeTestFile(t, filepath.Join(dataDir, "records.json"), "{}")
	writeTestFile(t, filepath.Join(dataDir, "webview2", "EBWebView", "Cache", "data_0"), "cache bytes")
	writeTestFile(t, filepath.Join(dataDir, "backup", "bonus-data-20250101-0000", "records.json"), "{}")

	result, err := backupData(dataDir, backupRoot)
	if err != nil {
		t.Fatalf("پشتیبان‌گیری ناموفق: %v", err)
	}
	if result.Files != 1 {
		t.Fatalf("پشتیبان باید فقط ۱ فایل داده داشته باشد، %d داشت", result.Files)
	}
	if _, err := os.Stat(filepath.Join(result.Path, "webview2")); !os.IsNotExist(err) {
		t.Fatal("پروفایل موتور مرورگر نباید در پشتیبان باشد")
	}
	if _, err := os.Stat(filepath.Join(result.Path, "backup")); !os.IsNotExist(err) {
		t.Fatal("پوشهٔ پشتیبان‌های پیشین نباید داخل پشتیبان تازه برود")
	}
}

// TestBackupDataKeepsSubdirectories می‌سنجد ساختار زیرپوشه‌ها حفظ می‌شود.
func TestBackupDataKeepsSubdirectories(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "bonus-data")
	backupRoot := filepath.Join(root, "backup")

	writeTestFile(t, filepath.Join(dataDir, "attachments", "1404", "decree.pdf"), "pdf bytes")
	if _, err := backupData(dataDir, backupRoot); err != nil {
		t.Fatalf("پشتیبان‌گیری ناموفق: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(backupRoot, "bonus-data-*", "attachments", "1404", "decree.pdf"))
	if err != nil {
		t.Fatalf("جستجوی فایل پشتیبان ناموفق: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("فایل زیرپوشه باید با همان ساختار کپی شود، %d مورد یافت شد", len(matches))
	}
}

// TestBackupDataReportsFailure می‌سنجد پشتیبان‌گیری ناقص، موفقیت دروغین گزارش نمی‌کند:
// پوشهٔ دادهٔ خالی و پوشهٔ دادهٔ ناموجود باید خطا برگردانند.
func TestBackupDataReportsFailure(t *testing.T) {
	root := t.TempDir()

	empty := filepath.Join(root, "empty-data")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatalf("ساخت پوشهٔ خالی ناموفق: %v", err)
	}
	if _, err := backupData(empty, filepath.Join(root, "backup")); err == nil {
		t.Fatal("پوشهٔ دادهٔ خالی باید خطا بدهد، نه پشتیبان خالی")
	}

	missing := filepath.Join(root, "does-not-exist")
	if _, err := backupData(missing, filepath.Join(root, "backup")); err == nil {
		t.Fatal("پوشهٔ دادهٔ ناموجود باید خطا بدهد")
	}

	// مسیر پشتیبان غیرقابل‌نوشتن (پوشه‌ای که فایل هم‌نامش وجود دارد).
	blocker := filepath.Join(root, "blocked")
	writeTestFile(t, blocker, "not a directory")
	if _, err := backupData(empty, blocker); err == nil {
		t.Fatal("مقصد نامعتبر باید خطا بدهد")
	}
}

// TestBackupDataIsRepeatable می‌سنجد دو پشتیبان پشت‌سرهم، یکدیگر را خراب نمی‌کنند و
// پشتیبان دوم داخل پوشهٔ پشتیبان‌ها نمی‌افتد.
func TestBackupDataIsRepeatable(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "bonus-data")
	backupRoot := filepath.Join(root, "backup")
	writeTestFile(t, filepath.Join(dataDir, "records.json"), `{"version":1}`)

	first, err := backupData(dataDir, backupRoot)
	if err != nil {
		t.Fatalf("پشتیبان اول ناموفق: %v", err)
	}
	second, err := backupData(dataDir, backupRoot)
	if err != nil {
		t.Fatalf("پشتیبان دوم ناموفق: %v", err)
	}
	if second.Path != first.Path {
		// نام زمان‌دار تا دقیقه دقت دارد؛ اگر یک دقیقه گذشته باشد مسیر تازه است.
		if first.Files != second.Files {
			t.Fatalf("شمار فایل‌ها بین دو پشتیبان تفاوت کرد: %d و %d", first.Files, second.Files)
		}
	}
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		t.Fatalf("خواندن پوشهٔ پشتیبان‌ها ناموفق: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == "backup" {
			t.Fatal("پوشهٔ پشتیبان‌ها نباید داخل خودش تکرار شود")
		}
	}
}
