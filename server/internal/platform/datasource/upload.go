// بارگذاری فایل منبع (CSV/Excel) در پوشهٔ دادهٔ سرور.
//
// قاعده‌ها: پسوند و امضای فایل سنجیده می‌شود، حجم سقف دارد، نام فایل ساختهٔ همین بسته است
// (`<نام امن>-<هش>.<پسوند>`) و مسیر بازگشتی همیشه داخل پوشهٔ بارگذاری‌ها می‌ماند.

package datasource

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MaxUploadBytes سقف حجم فایل بارگذاری‌شده (۳۲ مگابایت).
const MaxUploadBytes = 32 << 20

// uploadNamePattern نویسه‌های مجاز در نام فایل.
var uploadNamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Upload فایل ذخیره‌شده.
type Upload struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int    `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
	KindFa    string `json:"kindFa"`
}

// SaveUpload فایل بارگذاری‌شده را ذخیره می‌کند و مسیر سرور آن را برمی‌گرداند.
func SaveUpload(dataDir, originalName string, data []byte) (Upload, error) {
	dir := strings.TrimSpace(dataDir)
	if dir == "" {
		return Upload{}, errors.New("پوشهٔ دادهٔ سرور تنظیم نشده است")
	}
	if len(data) == 0 {
		return Upload{}, errors.New("فایل بارگذاری‌شده خالی است")
	}
	if len(data) > MaxUploadBytes {
		return Upload{}, fmt.Errorf("حجم فایل بیش از سقف مجاز است (حداکثر %d مگابایت)", MaxUploadBytes>>20)
	}
	extension := strings.ToLower(filepath.Ext(originalName))
	switch extension {
	case ".csv", ".txt":
	case ".xlsx":
	case ".xls":
		return Upload{}, errors.New("قالب قدیمی xls پشتیبانی نمی‌شود؛ فایل را در Excel به xlsx یا CSV ذخیره کنید")
	default:
		return Upload{}, fmt.Errorf("پسوند «%s» پشتیبانی نمی‌شود؛ CSV یا XLSX بفرستید", extension)
	}
	base := strings.TrimSuffix(filepath.Base(originalName), extension)
	base = strings.Trim(uploadNamePattern.ReplaceAllString(base, "-"), "-")
	if base == "" {
		base = "upload"
	}
	if len(base) > 48 {
		base = base[:48]
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])[:12]
	name := fmt.Sprintf("%s-%s%s", base, hash, extension)

	uploadDir := filepath.Join(dir, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return Upload{}, fmt.Errorf("ساخت پوشهٔ بارگذاری ممکن نشد: %w", err)
	}
	full := filepath.Join(uploadDir, name)
	if !strings.HasPrefix(full, uploadDir+string(os.PathSeparator)) {
		return Upload{}, errors.New("مسیر فایل بارگذاری‌شده معتبر نیست")
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return Upload{}, fmt.Errorf("ذخیرهٔ فایل ممکن نشد: %w", err)
	}
	kindFa := "فایل CSV"
	if extension == ".xlsx" {
		kindFa = "فایل اکسل"
	}
	return Upload{Name: name, Path: full, SizeBytes: len(data), SHA256: hash, KindFa: kindFa}, nil
}

// UploadDir پوشهٔ بارگذاری‌ها برای یک پوشهٔ داده.
func UploadDir(dataDir string) string { return filepath.Join(strings.TrimSpace(dataDir), "uploads") }
