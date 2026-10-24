//go:build windows

package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// backupResult گزارش پشتیبان ساخته‌شده؛ رابط وب همین را به کاربر نشان می‌دهد.
type backupResult struct {
	Path  string
	Files int
	Bytes int64
}

// skipDirs پوشه‌هایی که بخشی از دادهٔ کاربر نیستند و در پشتیبان نمی‌آیند.
//
// webview2 پروفایل و کش موتور مرورگر داخلی است (چند ده مگابایت و بی‌ارزش برای بازیابی)
// و backup خودِ پوشهٔ پشتیبان‌هاست که کپی کردنش یعنی پشتیبانِ پشتیبان.
var skipDirs = map[string]bool{"webview2": true, "backup": true}

// backupData یک کپی زمان‌دار از پوشهٔ داده می‌سازد.
//
// چرا این کار بی‌خطر است حتی وقتی برنامه باز است: انبار رکوردها هر تغییر را با نوشتن
// فایل موقت و جای‌گزینی کامل ذخیره می‌کند، پس خواننده همیشه یک فایل کامل (کهنه یا نو)
// می‌بیند، نه فایل نیمه‌کاره.
//
// خطا هم بی‌صدا رد نمی‌شود: هر شکستی در کپی برگردانده می‌شود تا رابط بگوید پشتیبان
// ناقص مانده است، نه اینکه موفقیت دروغین نشان دهد.
func backupData(dataDir, backupRoot string) (backupResult, error) {
	stamp := time.Now().Format("20060102-1504")
	target := filepath.Join(backupRoot, "bonus-data-"+stamp)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return backupResult{}, fmt.Errorf("ساخت پوشهٔ پشتیبان «%s» ناموفق: %w", target, err)
	}

	result := backupResult{Path: target}
	walkErr := filepath.WalkDir(dataDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dataDir {
			return nil
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skipDirs[entry.Name()] {
				return fs.SkipDir
			}
			return os.MkdirAll(filepath.Join(target, rel), 0o755)
		}
		written, err := copyFile(path, filepath.Join(target, rel))
		if err != nil {
			return err
		}
		result.Files++
		result.Bytes += written
		return nil
	})
	if walkErr != nil {
		return backupResult{}, fmt.Errorf("پشتیبان‌گیری از «%s» ناتمام ماند: %w", dataDir, walkErr)
	}
	if result.Files == 0 {
		return backupResult{}, fmt.Errorf("پوشهٔ داده «%s» هیچ فایلی برای پشتیبان‌گیری نداشت", dataDir)
	}
	return result, nil
}

// copyFile یک فایل را کامل کپی می‌کند و شمار بایت‌های نوشته‌شده را برمی‌گرداند.
func copyFile(src, dst string) (int64, error) {
	source, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("خواندن «%s» ناموفق: %w", src, err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, err
	}
	destination, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, fmt.Errorf("نوشتن «%s» ناموفق: %w", dst, err)
	}
	written, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return written, fmt.Errorf("کپی «%s» ناموفق: %w", src, copyErr)
	}
	if closeErr != nil {
		return written, fmt.Errorf("بستن فایل پشتیبان «%s» ناموفق: %w", dst, closeErr)
	}
	return written, nil
}
