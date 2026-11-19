// آزمون‌های رمز و توکن — هر آزمون به یک بند spec ارجاع دارد (docs/security-spec-fa.md).

package auth

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"hrms/internal/contracts"
)

// TestREQ01Argon2idHashFormat می‌سنجد درهم تازه با Argon2id و قالب نسخه‌دار ذخیره شود
// و رمز خام در هیچ بخشی از رشتهٔ ذخیره‌شده نباشد.
func TestREQ01Argon2idHashFormat(t *testing.T) {
	hash, err := HashPassword("Secret-pass-1405")
	if err != nil {
		t.Fatalf("درهم‌سازی ناموفق: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("قالب درهم باید Argon2id با پارامترهای اعلام‌شده باشد: %s", hash)
	}
	if strings.Contains(hash, "Secret-pass-1405") {
		t.Fatal("رمز خام در درهم ذخیره شده")
	}
	if !VerifyPassword(hash, "Secret-pass-1405") {
		t.Fatal("درهم ساخته‌شده نباید رمز درست را رد کند")
	}
	if VerifyPassword(hash, "Secret-pass-1406") {
		t.Fatal("درهم نباید رمز نادرست را بپذیرد")
	}
	// دو درهم از یک رمز باید متفاوت باشند (نمک تصادفی).
	other, err := HashPassword("Secret-pass-1405")
	if err != nil {
		t.Fatalf("درهم‌سازی دوم ناموفق: %v", err)
	}
	if other == hash {
		t.Fatal("نمک تصادفی اعمال نشده است")
	}
}

// TestREQ02LegacyHashUpgradesOnLogin می‌سنجد درهم PBKDF2 نسخهٔ پیشین همچنان پذیرفته و در
// نخستین ورود موفق به Argon2id ارتقا می‌یابد.
func TestREQ02LegacyHashUpgradesOnLogin(t *testing.T) {
	legacy, err := legacyPBKDF2Hash("demo")
	if err != nil {
		t.Fatalf("ساخت درهم قدیمی ناموفق: %v", err)
	}
	if !strings.HasPrefix(legacy, "pbkdf2_sha256$") {
		t.Fatalf("درهم قدیمی ساخته نشد: %s", legacy)
	}
	if !VerifyPassword(legacy, "demo") {
		t.Fatal("درهم قدیمی باید همچنان پذیرفته شود")
	}
	if !NeedsRehash(legacy) {
		t.Fatal("درهم PBKDF2 باید نیازمند ارتقا تشخیص داده شود")
	}

	service := NewService([]byte("test-secret-32-bytes-long-abcdef"), []Account{{
		Username:     "legacy",
		PasswordHash: legacy,
		User:         contracts.User{Username: "legacy", Role: contracts.RoleEmployee},
	}})
	if _, err := service.Login("legacy", "demo"); err != nil {
		t.Fatalf("ورود با درهم قدیمی ناموفق: %v", err)
	}
	stored, ok := service.passwordHash("legacy")
	if !ok {
		t.Fatal("حساب پس از ورود پیدا نشد")
	}
	if !strings.HasPrefix(stored, "$argon2id$") {
		t.Fatalf("درهم پس از ورود باید به Argon2id ارتقا یابد: %s", stored)
	}
	if !VerifyPassword(stored, "demo") {
		t.Fatal("درهم ارتقایافته باید همان رمز را بپذیرد")
	}
}

// TestREQ03PasswordStrengthPolicy می‌سنجد سیاست قدرت رمز، پیش‌فرض‌های شناخته‌شده و
// رمزهای کوتاه را رد می‌کند و رمز سازمانی معمول را می‌پذیرد.
func TestREQ03PasswordStrengthPolicy(t *testing.T) {
	weak := []string{"6468", "demo", "admin", "12345678", "aaaaaaaa", "Ab1!ab1!"}
	for _, pw := range weak {
		if err := ValidateStrength(pw); err == nil {
			t.Fatalf("رمز ضعیف «%s» پذیرفته شد", pw)
		}
	}
	strong := []string{"Bandar-1405!", "Hrms$Secure2026", "کارمند-۱۴۰۵-امن"}
	for _, pw := range strong {
		if err := ValidateStrength(pw); err != nil {
			t.Fatalf("رمز «%s» نباید رد شود: %v", pw, err)
		}
	}
	if _, err := HashPassword(strings.Repeat("a", maxPasswordBytes+1)); err == nil {
		t.Fatal("رمز بیش از سقف طول باید رد شود")
	}
}

// TestREQ04UnknownUserTimingEqualized می‌سنجد ورود با نام کاربری ناموجود همان خطا را
// می‌دهد و همان هزینهٔ درهم‌سازی را می‌پردازد؛ وگرنه شمارش کاربران با زمان پاسخ ممکن است.
func TestREQ04UnknownUserTimingEqualized(t *testing.T) {
	accounts, err := SeedAccounts()
	if err != nil {
		t.Fatalf("ساخت حساب‌های نمونه ناموفق: %v", err)
	}
	service := NewService([]byte("test-secret-32-bytes-long-abcdef"), accounts)

	known := time.Now()
	if _, err := service.Login("admin", "wrong-password"); err == nil {
		t.Fatal("رمز نادرست نباید پذیرفته شود")
	}
	knownDur := time.Since(known)

	unknown := time.Now()
	if _, err := service.Login("no-such-user", "wrong-password"); err == nil {
		t.Fatal("کاربر ناموجود نباید پذیرفته شود")
	}
	unknownDur := time.Since(unknown)

	if unknownDur*3 < knownDur {
		t.Fatalf("زمان پاسخ کاربر ناموجود (%v) از رمز نادرست (%v) بسیار کمتر است؛ نشت شمارش کاربر", unknownDur, knownDur)
	}
}

// legacyPBKDF2Hash برای آزمون ارتقا، یک درهم با قالب قدیمی می‌سازد.
func legacyPBKDF2Hash(password string) (string, error) {
	salt := make([]byte, pbkdf2SaltLength)
	for i := range salt {
		salt[i] = byte(i + 1)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, pbkdf2KeyLength)
	if err != nil {
		return "", err
	}
	return "pbkdf2_sha256$210000$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(key), nil
}
