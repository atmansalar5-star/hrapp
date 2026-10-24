// Package auth احراز هویت: درهم‌سازی رمز، صدور و بازبینی توکن، و میان‌افزار نشست.
//
// سامانه داده‌های پرسنلی حساس نگه می‌دارد، پس رمزها هرگز خام ذخیره نمی‌شوند و توکن‌ها
// کوتاه‌عمر هستند. برای همین نسخه فقط از کتابخانهٔ استاندارد Go استفاده شده است تا
// سطح وابستگی کم بماند؛ برای ارتقای رمز به Argon2id تنها همین فایل تغییر می‌کند.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	passwordIterations = 210_000
	passwordKeyLength  = 32
	passwordSaltLength = 16
	passwordScheme     = "pbkdf2_sha256"
)

// ErrInvalidHash قالب درهم ذخیره‌شده نامعتبر است.
var ErrInvalidHash = errors.New("قالب درهم رمز نامعتبر است")

// HashPassword رمز را با نمک تصادفی درهم می‌کند.
// قالب خروجی: pbkdf2_sha256$iter$salt$key
func HashPassword(password string) (string, error) {
	if len(password) < 4 {
		return "", errors.New("رمز باید حداقل ۴ نویسه باشد")
	}
	salt := make([]byte, passwordSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("تولید نمک ناموفق: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeyLength)
	if err != nil {
		return "", fmt.Errorf("درهم‌سازی رمز ناموفق: %w", err)
	}
	return fmt.Sprintf(
		"%s$%d$%s$%s",
		passwordScheme,
		passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword رمز ورودی را با درهم ذخیره‌شده می‌سنجد.
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != passwordScheme {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iterations, len(want))
	if err != nil {
		return false
	}
	// مقایسهٔ زمان‌ثابت تا زمان پاسخ، اطلاعاتی دربارهٔ رمز لو ندهد.
	return subtle.ConstantTimeCompare(got, want) == 1
}
