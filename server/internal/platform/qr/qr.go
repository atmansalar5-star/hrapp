// Package qr تولید QR کد سامانه (بدون نوشتن رمزگذار از صفر).
//
// قاعده‌های این لایه:
//   - محتوا همیشه یک نشانی «تأیید» روی همین سامانه است، نه دادهٔ خام؛ پس با اسکن کد
//     هیچ دادهٔ پرسنلی بیرون نمی‌رود و می‌توان کد را روی سند چاپی گذاشت.
//   - سطح تصحیح خطا M و حاشیهٔ ساکت چهار ماژول است تا چاپ با کیفیت پایین هم خوانده شود.
//   - SVG پیش‌فرض است چون در سندهای چاپی و کنار متن فارسی بی‌افت کیفیت می‌ماند؛ PNG هم
//     برای کارت و برچسب پشتیبانی می‌شود.
package qr

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/color"
	"image/png"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// Level سطح تصحیح خطا (پیش‌فرض M = حدود ۱۵٪ بازیافت).
const Level = qrcode.Medium

// DefaultSize اندازهٔ پیش‌فرض تصویر به پیکسل/ماژول‌سنج.
const DefaultSize = 240

// MinSize و MaxSize کران‌های اندازه.
const (
	MinSize = 96
	MaxSize = 1024
)

// SVG یک QR کد به شکل SVG برمی‌گرداند (بی‌هم‌وابستگی به فایل بیرونی).
func SVG(content string, size int) (string, error) {
	code, err := newCode(content)
	if err != nil {
		return "", err
	}
	size = clampSize(size)
	bitmap := code.Bitmap()
	modules := len(bitmap)
	scale := float64(size) / float64(modules)

	var buf strings.Builder
	fmt.Fprintf(&buf, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(&buf, "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\" viewBox=\"0 0 %d %d\" role=\"img\" aria-label=\"QR\">\n",
		size, size, size, size)
	fmt.Fprintf(&buf, "<rect width=\"%d\" height=\"%d\" fill=\"#ffffff\"/>\n", size, size)
	buf.WriteString("<g fill=\"#000000\">\n")
	for y, row := range bitmap {
		x := 0
		for x < modules {
			if !row[x] {
				x++
				continue
			}
			run := 1
			for x+run < modules && row[x+run] {
				run++
			}
			fmt.Fprintf(&buf, "<rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\"/>\n",
				formatCoord(float64(x)*scale), formatCoord(float64(y)*scale),
				formatCoord(float64(run)*scale), formatCoord(scale))
			x += run
		}
	}
	buf.WriteString("</g>\n</svg>\n")
	return buf.String(), nil
}

// PNG یک QR کد به شکل PNG برمی‌گرداند.
func PNG(content string, size int) ([]byte, error) {
	code, err := newCode(content)
	if err != nil {
		return nil, err
	}
	size = clampSize(size)
	img := code.Image(size)
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		return nil, fmt.Errorf("ساخت تصویر QR ممکن نشد: %w", err)
	}
	return buf.Bytes(), nil
}

// ASCII یک نمایش متنی (برای آزمون‌های خودکار و لاگ) برمی‌گرداند.
func ASCII(content string) (string, error) {
	code, err := newCode(content)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	for _, row := range code.Bitmap() {
		for _, dark := range row {
			if dark {
				buf.WriteString("██")
			} else {
				buf.WriteString("  ")
			}
		}
		buf.WriteString("\n")
	}
	return buf.String(), nil
}

// Sign امضای کوتاه یک نشانی تأیید را می‌سازد (HMAC-SHA256 با کلید سرور).
//
// چرا امضا: نشانی تأیید بیرون از سامانه اسکن می‌شود؛ با امضا می‌توان سنجید که همین سرور
// نشانی را ساخته است و کسی کد/سریال را دستکاری نکرده است. بخش‌ها با «|» به هم می‌چسبند و
// ترتیبشان مهم است (kind|id|serial).
func Sign(key []byte, parts ...string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

// Check امضای نشانی را می‌سنجد.
func Check(key []byte, token string, parts ...string) bool {
	expected := Sign(key, parts...)
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(token)))
}

// VerifyLink نشانی کامل صفحهٔ تأیید را می‌سازد (پایه = scheme://host).
func VerifyLink(base, scope, id string, query map[string]string) string {
	url := strings.TrimRight(strings.TrimSpace(base), "/") + "/verify/" + scope + "/" + id
	if len(query) == 0 {
		return url
	}
	sep := "?"
	for _, key := range sortedKeys(query) {
		value := strings.TrimSpace(query[key])
		if value == "" {
			continue
		}
		url += sep + key + "=" + queryEscape(value)
		sep = "&"
	}
	return url
}

// sortedKeys کلیدها را مرتب می‌کند تا نشانی همیشه یک شکل باشد (کش و مقایسهٔ آزمون).
func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// queryEscape کدگذاری امن مقدارهای پرس‌وجو.
func queryEscape(value string) string {
	var buf strings.Builder
	for _, b := range []byte(value) {
		switch {
		case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9',
			b == '-', b == '_', b == '.', b == '~':
			buf.WriteByte(b)
		default:
			fmt.Fprintf(&buf, "%%%02X", b)
		}
	}
	return buf.String()
}

// newCode رمزگذار کتابخانه را با تنظیمات مشترک سامانه می‌سازد.
func newCode(content string) (*qrcode.QRCode, error) {
	value := strings.TrimSpace(content)
	if value == "" {
		return nil, fmt.Errorf("محتوای QR خالی است")
	}
	code, err := qrcode.New(value, Level)
	if err != nil {
		return nil, fmt.Errorf("ساخت QR کد ممکن نشد: %w", err)
	}
	code.DisableBorder = false
	code.ForegroundColor = color.Black
	code.BackgroundColor = color.White
	return code, nil
}

// clampSize اندازه را در کران‌های مجاز نگه می‌دارد.
func clampSize(size int) int {
	if size <= 0 {
		return DefaultSize
	}
	if size < MinSize {
		return MinSize
	}
	if size > MaxSize {
		return MaxSize
	}
	return size
}

// formatCoord عدد مختصات را بدون اعشار زائد می‌نویسد.
func formatCoord(value float64) string {
	out := fmt.Sprintf("%.2f", value)
	out = strings.TrimRight(out, "0")
	return strings.TrimSuffix(out, ".")
}
