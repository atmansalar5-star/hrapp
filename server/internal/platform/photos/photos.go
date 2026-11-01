// Package photos انبار «عکس کارگزینی» کارکنان روی دیسک.
//
// چرا بستهٔ جدا: عکس پرسنلی یک فایل دودویی است، نه یک فیلد متنی؛ باید بیرون از درخت
// کد و در پوشهٔ دادهٔ سرور بماند تا به‌روزرسانی برنامه یا بازآوری باینری آن را پاک نکند.
//
// قاعده‌ها:
//   - فایل‌ها در `<data>/photos/` ذخیره می‌شوند و نامشان ساختهٔ همین بسته است
//     (`<کد پرسنلی>-<هش>.<پسوند>`)؛ نام ورودی کاربر هرگز مسیر نمی‌سازد.
//   - نوع فایل با «امضای دودویی» سنجیده می‌شود، نه با پسوند تنها. برای عکس پرسنلی فقط
//     تصویر واقعی (PNG/JPEG/WEBP) پذیرفته می‌شود؛ SVG عمداً پذیرفته نمی‌شود چون عکس
//     پرسنلی همیشه دودویی است و SVG می‌تواند اسکریپت داشته باشد.
//   - نگاشت «کارمند → عکس» در فایل `index.json` (یونیکد UTF-8) نگهداری می‌شود، پس عکس
//     پس از راه‌اندازی دوبارهٔ سرور هم سرِ جایش می‌ماند — مستقل از این‌که انبار خودِ
//     کارکنان حافظه‌ای باشد یا پایگاه‌داده.
//   - نشانی سرو عکس «امضاشده» است (`?e=<کد>&t=<امضا>`): برچسب <img> نمی‌تواند هدر
//     احراز هویت بفرستد، ولی امضا جلوی خواندن عکس کارمند دیگر را با حدس نام می‌گیرد.
package photos

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"hrms/internal/platform/calendar"
)

// MaxBytes سقف حجم عکس پرسنلی (۴ مگابایت). عکس کارگزینی بیش از این لازم نیست و
// بزرگ‌تر از آن فقط پهنای باند و پشتیبان‌گیری را سنگین می‌کند.
const MaxBytes = 4 << 20

// indexFile نام پروندهٔ نگاشت کارمند → عکس.
const indexFile = "index.json"

// fileNamePattern نام فایل مجاز: فقط حرف/رقم لاتین، نقطه، خط تیره و زیرخط.
var fileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,96}$`)

// employeePattern شناسهٔ کارمند برای نام‌گذاری امن فایل (کد پرسنلی مثل EMP-001).
var employeePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)

// Store انبار عکس‌های کارگزینی.
type Store struct {
	dir       string
	indexPath string
	mu        sync.Mutex
}

// Photo اطلاعات یک عکس ذخیره‌شده.
type Photo struct {
	EmployeeID   string `json:"employeeId"`
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	SizeBytes    int    `json:"sizeBytes"`
	SHA256       string `json:"sha256"`
	UpdatedAt    string `json:"updatedAt"`
	UpdatedBy    string `json:"updatedBy,omitempty"`
	OriginalName string `json:"originalName,omitempty"`
}

// index ساختار پروندهٔ نگاشت.
type index struct {
	// Photos کلید آن شناسهٔ کارمند است.
	Photos map[string]Photo `json:"photos"`
}

// ErrEmptyDataDir پوشهٔ داده تنظیم نشده است.
var ErrEmptyDataDir = errors.New("پوشهٔ داده برای عکس پرسنلی تنظیم نشده است")

// New انبار را در پوشهٔ داده می‌سازد (پوشه در صورت نبود ساخته می‌شود).
func New(dataDir string) (*Store, error) {
	dir := strings.TrimSpace(dataDir)
	if dir == "" {
		return nil, ErrEmptyDataDir
	}
	full := filepath.Join(dir, "photos")
	if err := os.MkdirAll(full, 0o755); err != nil {
		return nil, fmt.Errorf("ساخت پوشهٔ عکس پرسنلی ممکن نشد: %w", err)
	}
	return &Store{dir: full, indexPath: filepath.Join(full, indexFile)}, nil
}

// Dir پوشهٔ عکس‌ها (برای آزمون و پشتیبان‌گیری).
func (s *Store) Dir() string { return s.dir }

// Save عکس یک کارمند را ذخیره می‌کند و اطلاعاتش را برمی‌گرداند.
//
// نام تازه از کد پرسنلی و هش محتوا ساخته می‌شود؛ پس بارگذاری دوبارهٔ همان تصویر فایل
// تکراری نمی‌سازد و تصویر تازه نشانی تازه می‌گیرد (کش مرورگر بی‌درنگ تازه می‌شود).
// عکس پیشین همان کارمند پاک می‌شود تا فایل بی‌استفاده نماند.
func (s *Store) Save(employeeID, originalName string, data []byte, actor string, at time.Time) (Photo, error) {
	id := strings.TrimSpace(employeeID)
	if !employeePattern.MatchString(id) {
		return Photo{}, errors.New("کد پرسنلی برای نام‌گذاری عکس معتبر نیست (حرف/رقم لاتین، خط تیره و زیرخط)")
	}
	if len(data) == 0 {
		return Photo{}, errors.New("فایل عکس خالی است")
	}
	if len(data) > MaxBytes {
		return Photo{}, fmt.Errorf("حجم عکس بیش از حد مجاز است (حداکثر %d کیلوبایت)", MaxBytes/1024)
	}
	ext, contentType, err := detectType(data)
	if err != nil {
		return Photo{}, err
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])[:16]
	name := fmt.Sprintf("%s-%s%s", id, hash, ext)
	if !fileNamePattern.MatchString(name) {
		return Photo{}, errors.New("ساخت نام فایل عکس ممکن نشد")
	}
	if err := os.WriteFile(filepath.Join(s.dir, name), data, 0o644); err != nil {
		return Photo{}, fmt.Errorf("نوشتن فایل عکس ممکن نشد: %w", err)
	}

	photo := Photo{
		EmployeeID:   id,
		Name:         name,
		ContentType:  contentType,
		SizeBytes:    len(data),
		SHA256:       hash,
		UpdatedAt:    calendar.Format(at),
		UpdatedBy:    strings.TrimSpace(actor),
		OriginalName: safeOriginalName(originalName),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.load()
	if err != nil {
		return Photo{}, err
	}
	previous := idx.Photos[id]
	idx.Photos[id] = photo
	if err := s.persist(idx); err != nil {
		return Photo{}, err
	}
	if previous.Name != "" && previous.Name != name {
		_ = os.Remove(filepath.Join(s.dir, previous.Name))
	}
	return photo, nil
}

// Get عکس فعلی یک کارمند را برمی‌گرداند.
func (s *Store) Get(employeeID string) (Photo, bool, error) {
	id := strings.TrimSpace(employeeID)
	if id == "" {
		return Photo{}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.load()
	if err != nil {
		return Photo{}, false, err
	}
	photo, ok := idx.Photos[id]
	if !ok || photo.Name == "" {
		return Photo{}, false, nil
	}
	// فایل ممکن است دستی پاک شده باشد؛ در این حالت «عکس ندارد» پاسخ صادق است.
	if !s.exists(photo.Name) {
		return Photo{}, false, nil
	}
	return photo, true, nil
}

// All نگاشت کامل کارمند → عکس (برای غنی‌سازی فهرست کارکنان در یک گذر).
func (s *Store) All() (map[string]Photo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make(map[string]Photo, len(idx.Photos))
	for id, photo := range idx.Photos {
		if photo.Name != "" && s.exists(photo.Name) {
			out[id] = photo
		}
	}
	return out, nil
}

// Remove عکس یک کارمند را پاک می‌کند (نبودنش خطا نیست).
func (s *Store) Remove(employeeID string) error {
	id := strings.TrimSpace(employeeID)
	if id == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, err := s.load()
	if err != nil {
		return err
	}
	photo, ok := idx.Photos[id]
	delete(idx.Photos, id)
	if err := s.persist(idx); err != nil {
		return err
	}
	if ok && photo.Name != "" {
		if err := os.Remove(filepath.Join(s.dir, photo.Name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("پاک‌کردن فایل عکس ممکن نشد: %w", err)
		}
	}
	return nil
}

// Open فایل عکس را برای سرو کردن باز می‌کند.
func (s *Store) Open(name string) (io.ReadCloser, string, error) {
	path, ok := s.path(name)
	if !ok {
		return nil, "", errors.New("نام فایل عکس معتبر نیست")
	}
	handle, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	return handle, contentTypeOf(strings.ToLower(filepath.Ext(path))), nil
}

// Exists آیا فایل با این نام هست.
func (s *Store) Exists(name string) bool { return s.exists(name) }

// URL نشانی سرو امضاشدهٔ عکس یک کارمند.
//
// امضا روی «نام فایل + شناسهٔ کارمند» است تا نشانی یک کارمند برای کارمند دیگر کار نکند
// و تغییر نام فایل هم امضا را باطل کند.
func URL(name, employeeID string, key []byte) string {
	escaped := url.PathEscape(strings.TrimSpace(name))
	token := sign(key, name, employeeID)
	return "/api/v1/photos/" + escaped + "?e=" + url.QueryEscape(employeeID) + "&t=" + token
}

// Check درستی امضای نشانی عکس را می‌سنجد.
func Check(key []byte, name, employeeID, token string) bool {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(token) == "" {
		return false
	}
	want := sign(key, name, employeeID)
	return hmac.Equal([]byte(want), []byte(strings.TrimSpace(token)))
}

// NameFromURL نام فایل را از نشانی سرو‌شده بیرون می‌کشد (برای پاک‌کردن عکس پیشین).
func NameFromURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "?"); idx >= 0 {
		value = value[:idx]
	}
	if idx := strings.LastIndex(value, "/"); idx >= 0 {
		value = value[idx+1:]
	}
	if decoded, err := url.PathUnescape(value); err == nil {
		value = decoded
	}
	if !fileNamePattern.MatchString(value) {
		return ""
	}
	return value
}

/* ------------------------------ درونی‌ها ------------------------------ */

// sign امضای کوتاه HMAC-SHA256 (۱۶ نویسهٔ نخست) برای نشانی عکس.
func sign(key []byte, parts ...string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("photo|" + strings.Join(parts, "|")))
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

// load پروندهٔ نگاشت را می‌خواند؛ نبودِ آن یعنی نگاشت خالی.
func (s *Store) load() (index, error) {
	idx := index{Photos: map[string]Photo{}}
	raw, err := os.ReadFile(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return idx, nil
		}
		return idx, fmt.Errorf("خواندن نگاشت عکس‌ها ممکن نشد: %w", err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return idx, nil
	}
	if err := json.Unmarshal(raw, &idx); err != nil {
		return index{Photos: map[string]Photo{}}, fmt.Errorf("نگاشت عکس‌ها معتبر نیست: %w", err)
	}
	if idx.Photos == nil {
		idx.Photos = map[string]Photo{}
	}
	return idx, nil
}

// persist نگاشت را می‌نویسد.
//
// خروجی JSON با UTF-8 و بدون فرار دادن نویسه‌های فارسی نوشته می‌شود تا فایل هم برای
// انسان خوانا بماند و هم برای پشتیبان‌گیری متنی بی‌ابهام باشد.
func (s *Store) persist(idx index) error {
	if idx.Photos == nil {
		idx.Photos = map[string]Photo{}
	}
	buf := &strings.Builder{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	// کلیدهای maps را encoding/json خودش مرتب می‌نویسد؛ پس خروجی پایدار و قابل diff است.
	if err := encoder.Encode(idx); err != nil {
		return fmt.Errorf("ساخت نگاشت عکس‌ها ممکن نشد: %w", err)
	}
	// نوشتن اتمیک: فایل موقت سپس جابه‌جایی، تا قطع برق نگاشت را نیمه‌کاره نکند.
	tmp := s.indexPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("نوشتن نگاشت عکس‌ها ممکن نشد: %w", err)
	}
	if err := os.Rename(tmp, s.indexPath); err != nil {
		return fmt.Errorf("جابه‌جایی نگاشت عکس‌ها ممکن نشد: %w", err)
	}
	return nil
}

// exists آیا فایل با این نام در پوشهٔ عکس‌ها هست.
func (s *Store) exists(name string) bool {
	path, ok := s.path(name)
	if !ok {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// path نام را بازبینی و به مسیر کامل تبدیل می‌کند؛ نام نامعتبر یعنی «نیست».
func (s *Store) path(name string) (string, bool) {
	value := strings.TrimSpace(name)
	if !fileNamePattern.MatchString(value) || strings.Contains(value, "..") || value == indexFile {
		return "", false
	}
	full := filepath.Join(s.dir, value)
	if !strings.HasPrefix(full, s.dir+string(os.PathSeparator)) {
		return "", false
	}
	return full, true
}

// safeOriginalName نام اصلی فایل را برای نمایش ذخیره می‌کند (بدون مسیر).
func safeOriginalName(name string) string {
	value := strings.TrimSpace(name)
	if value == "" {
		return ""
	}
	if idx := strings.LastIndexAny(value, `/\`); idx >= 0 {
		value = value[idx+1:]
	}
	if len([]rune(value)) > 96 {
		value = string([]rune(value)[:96])
	}
	return value
}

// detectType نوع تصویر را از امضای دودویی می‌شناسد.
func detectType(data []byte) (string, string, error) {
	if len(data) >= 4 {
		switch {
		case data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G':
			return ".png", "image/png", nil
		case data[0] == 0xFF && data[1] == 0xD8:
			return ".jpg", "image/jpeg", nil
		case string(data[:4]) == "RIFF" && len(data) > 12 && string(data[8:12]) == "WEBP":
			return ".webp", "image/webp", nil
		}
	}
	return "", "", errors.New("نوع فایل عکس شناخته نشد؛ عکس پرسنلی باید PNG، JPEG یا WEBP باشد")
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
	}
	return "application/octet-stream"
}
