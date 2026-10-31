// Package branding انبار فایل‌های برند سازمان (نشان/لوگوی شرکت‌ها).
//
// قاعده‌ها:
//   - فایل‌ها بیرون از درخت کد و در پوشهٔ دادهٔ سرور ذخیره می‌شوند
//     (`<data>/branding/`)، پس بازآوری باینری یا به‌روزرسانی برنامه آن‌ها را پاک نمی‌کند.
//   - نوع فایل با «امضای دودویی» (magic bytes) سنجیده می‌شود، نه با پسوند تنها؛ SVG هم
//     متن است و باید تگ اسکریپت نداشته باشد (فایل برند روی صفحهٔ ورود بی‌سپر باز می‌شود).
//   - نام فایل ساختهٔ همین بسته است (`<owner>-<hash>.<ext>`) و هنگام خواندن هم سخت
//     بازبینی می‌شود تا هیچ مسیر بیرونی از راه نام فایل خوانده/نوشته نشود.
package branding

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MaxBytes سقف حجم فایل نشان (۲ مگابایت)؛ لوگوی چاپی بیش از این لازم نیست.
const MaxBytes = 2 << 20

// fileNamePattern نام فایل مجاز: فقط حرف/رقم لاتین، نقطه، خط تیره و زیرخط.
var fileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,96}$`)

// ownerPattern شناسهٔ مالک فایل (کد شرکت) برای نام‌گذاری امن.
var ownerPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)

// Store انبار فایل‌های برند روی دیسک.
type Store struct {
	dir string
}

// New انبار را در پوشهٔ داده می‌سازد (پوشه در صورت نبود ساخته می‌شود).
func New(dataDir string) (*Store, error) {
	dir := strings.TrimSpace(dataDir)
	if dir == "" {
		return nil, errors.New("پوشهٔ داده برای نشان شرکت تنظیم نشده است")
	}
	full := filepath.Join(dir, "branding")
	if err := os.MkdirAll(full, 0o755); err != nil {
		return nil, fmt.Errorf("ساخت پوشهٔ نشان شرکت ممکن نشد: %w", err)
	}
	return &Store{dir: full}, nil
}

// Dir پوشهٔ فایل‌های برند (برای آزمون و پشتیبان‌گیری).
func (s *Store) Dir() string { return s.dir }

// File فایل ذخیره‌شده.
type File struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	SizeBytes   int    `json:"sizeBytes"`
	SHA256      string `json:"sha256"`
}

// Save فایل نشان یک شرکت را ذخیره می‌کند و اطلاعاتش را برمی‌گرداند.
//
// نام تازه از کد شرکت و هش محتوا ساخته می‌شود؛ پس بارگذاری دوبارهٔ همان تصویر، فایل
// تکراری نمی‌سازد و بارگذاری تصویر تازه، نشانی تازه می‌دهد (کش مرورگر بی‌درنگ تازه می‌شود).
func (s *Store) Save(owner, originalName string, data []byte) (File, error) {
	owner = strings.TrimSpace(owner)
	if !ownerPattern.MatchString(owner) {
		return File{}, errors.New("کد شرکت برای نام‌گذاری نشان معتبر نیست (حرف/رقم لاتین، خط تیره و زیرخط)")
	}
	if len(data) == 0 {
		return File{}, errors.New("فایل نشان خالی است")
	}
	if len(data) > MaxBytes {
		return File{}, fmt.Errorf("حجم فایل نشان بیش از حد مجاز است (حداکثر %d کیلوبایت)", MaxBytes/1024)
	}
	ext, contentType, err := detectType(data, originalName)
	if err != nil {
		return File{}, err
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])[:16]
	name := fmt.Sprintf("%s-%s%s", owner, hash, ext)
	if !fileNamePattern.MatchString(name) {
		return File{}, errors.New("ساخت نام فایل نشان ممکن نشد")
	}
	path := filepath.Join(s.dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return File{}, fmt.Errorf("نوشتن فایل نشان ممکن نشد: %w", err)
	}
	return File{Name: name, URL: URL(name), ContentType: contentType, SizeBytes: len(data), SHA256: hash}, nil
}

// Remove فایل نشان را پاک می‌کند (نبود فایل خطا نیست).
func (s *Store) Remove(name string) error {
	path, ok := s.path(name)
	if !ok {
		return errors.New("نام فایل نشان معتبر نیست")
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("پاک‌کردن فایل نشان ممکن نشد: %w", err)
	}
	return nil
}

// Open فایل نشان را برای سرو کردن باز می‌کند.
func (s *Store) Open(name string) (io.ReadCloser, string, error) {
	path, ok := s.path(name)
	if !ok {
		return nil, "", errors.New("نام فایل نشان معتبر نیست")
	}
	handle, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	contentType := contentTypeOf(strings.ToLower(filepath.Ext(path)))
	return handle, contentType, nil
}

// Exists آیا فایل نشان هست.
func (s *Store) Exists(name string) bool {
	path, ok := s.path(name)
	if !ok {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// NameFromURL نام فایل را از نشانی سروشده بیرون می‌کشد (برای پاک‌کردن نشان پیشین).
func NameFromURL(url string) string {
	value := strings.TrimSpace(url)
	if value == "" {
		return ""
	}
	if idx := strings.LastIndex(value, "/"); idx >= 0 {
		value = value[idx+1:]
	}
	if idx := strings.IndexAny(value, "?#"); idx >= 0 {
		value = value[:idx]
	}
	if !fileNamePattern.MatchString(value) {
		return ""
	}
	return value
}

// URL نشانی عمومی فایل نشان روی همین سرور.
func URL(name string) string { return "/api/v1/branding/" + name }

// path نام را بازبینی و به مسیر کامل تبدیل می‌کند؛ نام نامعتبر یعنی «نیست».
func (s *Store) path(name string) (string, bool) {
	value := strings.TrimSpace(name)
	if !fileNamePattern.MatchString(value) || strings.Contains(value, "..") {
		return "", false
	}
	full := filepath.Join(s.dir, value)
	// بازبینی پایانی: مسیر باید داخل پوشهٔ برند بماند (در برابر ترفندهای نام).
	if !strings.HasPrefix(full, s.dir+string(os.PathSeparator)) {
		return "", false
	}
	return full, true
}

// detectType نوع فایل را از امضای دودویی و پسوند می‌شناسد.
func detectType(data []byte, originalName string) (string, string, error) {
	if len(data) >= 4 {
		switch {
		case data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G':
			return ".png", "image/png", nil
		case data[0] == 0xFF && data[1] == 0xD8:
			return ".jpg", "image/jpeg", nil
		case string(data[:4]) == "RIFF" && len(data) > 12 && string(data[8:12]) == "WEBP":
			return ".webp", "image/webp", nil
		case string(data[:3]) == "GIF":
			return ".gif", "image/gif", nil
		}
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "<svg") || strings.HasPrefix(trimmed, "<?xml") {
		if !strings.Contains(trimmed, "<svg") {
			return "", "", errors.New("فایل SVG معتبر نیست (تگ svg پیدا نشد)")
		}
		lower := strings.ToLower(trimmed)
		for _, forbidden := range []string{"<script", "javascript:", "onload=", "onerror="} {
			if strings.Contains(lower, forbidden) {
				return "", "", errors.New("فایل SVG نشان نباید اسکریپت یا رویداد داشته باشد")
			}
		}
		return ".svg", "image/svg+xml", nil
	}
	if ext := strings.ToLower(filepath.Ext(originalName)); ext != "" {
		return "", "", fmt.Errorf("نوع فایل «%s» پشتیبانی نمی‌شود؛ PNG، JPEG، WEBP، GIF یا SVG بفرستید", ext)
	}
	return "", "", errors.New("نوع فایل نشان شناخته نشد؛ PNG، JPEG، WEBP، GIF یا SVG بفرستید")
}

// contentTypeOf نوع محتوای یک پسوند (برای سرو فایل‌های ذخیره‌شده).
func contentTypeOf(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	}
	return "application/octet-stream"
}
