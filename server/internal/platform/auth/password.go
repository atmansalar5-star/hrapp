// Package auth احراز هویت: درهم‌سازی رمز، صدور و بازبینی توکن، و میان‌افزار نشست.
//
// سامانه داده‌های پرسنلی حساس نگه می‌دارد، پس رمزها هرگز خام ذخیره نمی‌شوند و توکن‌ها
// کوتاه‌عمر هستند. الگوریتم جاری Argon2id است (برندهٔ مسابقهٔ Password Hashing، مقاوم
// به حملهٔ سخت‌افزاری) و درهم‌های قدیمی PBKDF2 همچنان پذیرفته و در نخستین ورود موفق
// به Argon2id ارتقا می‌یابند؛ قالب ذخیره‌شده نسخه‌دار است تا ارتقای بعدی هم بدون
// شکستن دادهٔ موجود ممکن باشد.
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
	"sync"

	"golang.org/x/crypto/argon2"
)

// پارامترهای درهم‌سازی.
//
// Argon2id با ۱۹ مگابایت حافظه، دو گذر و یک رشته: گزینهٔ دوم توصیهٔ OWASP. چرا نه
// حافظهٔ بزرگ‌تر: هر درهم‌سازی حافظهٔ همان درخواست را می‌گیرد و اگر ۶۴ مگابایت باشد، ده
// ورود هم‌زمان یک گیگابایت مصرف می‌کند — یعنی خود درهم‌سازی به ابزار DoS تبدیل می‌شود.
// این تعادل، با محدودیت نرخ ورود در بستهٔ websecurity کامل می‌شود.
const (
	SchemeArgon2id = "argon2id"
	schemePBKDF2   = "pbkdf2_sha256"

	argonVersion uint32 = argon2.Version
	argonTime    uint32 = 2
	argonMemory  uint32 = 19 * 1024 // کیلوبایت
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16

	// pbkdf2Iterations فقط برای بازبینی درهم‌های قدیمی؛ مقدار در خود درهم ذخیره است.
	pbkdf2Iterations = 210_000
	pbkdf2KeyLength  = 32
	pbkdf2SaltLength = 16

	// minPasswordLength کف مطلق است (سازگاری با رمزهای نمایشی و بذرها)؛ سیاست واقعی
	// استقرار minStrongPasswordLength است و با ValidateStrength اعمال می‌شود.
	minPasswordLength       = 4
	minStrongPasswordLength = 8
	// maxPasswordBytes سقف ورودی است تا رمز چندمگابایتی هزینهٔ پردازش نسازد.
	maxPasswordBytes = 1024
)

// خطاهای رمز.
var (
	ErrInvalidHash     = errors.New("قالب درهم رمز نامعتبر است")
	ErrPasswordTooLong = errors.New("رمز بیش از حد بلند است")
	ErrPasswordWeak    = errors.New("رمز انتخابی به‌اندازهٔ کافی قوی نیست")
)

// weakPasswords فهرست کوتاه رمزهای پرتکرار و پیش‌فرض‌های شناخته‌شدهٔ همین سامانه.
var weakPasswords = map[string]bool{
	"12345678": true, "123456789": true, "1234567890": true, "password": true,
	"password1": true, "qwerty123": true, "admin123": true, "admin1234": true,
	"hrms1234": true, "demo1234": true, "demo": true, "6468": true, "admin": true,
	"11111111": true, "00000000": true, "aaaaaaaa": true,
}

// HashPassword رمز را با نمک تصادفی و Argon2id درهم می‌کند.
// قالب خروجی (PHC): $argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>
func HashPassword(password string) (string, error) {
	if len(password) < minPasswordLength {
		return "", fmt.Errorf("رمز باید حداقل %d نویسه باشد", minPasswordLength)
	}
	if len(password) > maxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("تولید نمک ناموفق: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		SchemeArgon2id, argonVersion, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// ValidateStrength سیاست قدرت رمز برای رمزهایی که کاربر یا راهبر انتخاب می‌کند.
//
// کف طول تنها معیار نیست: رمزهای پرتکرار و پیش‌فرض‌های همین سامانه (<code>6468</code>،
// <code>demo</code>) در فهرست سیاه‌اند، چون در نصب‌های واقعی دیده شده‌اند.
func ValidateStrength(password string) error {
	if len(password) < minStrongPasswordLength {
		return fmt.Errorf("%w: رمز باید حداقل %d نویسه باشد", ErrPasswordWeak, minStrongPasswordLength)
	}
	if len(password) > maxPasswordBytes {
		return ErrPasswordTooLong
	}
	if weakPasswords[strings.ToLower(password)] {
		return fmt.Errorf("%w: این رمز در فهرست رمزهای پرتکرار است", ErrPasswordWeak)
	}
	distinct := map[rune]bool{}
	for _, r := range password {
		distinct[r] = true
	}
	if len(distinct) < 4 {
		return fmt.Errorf("%w: رمز باید حداقل چهار نویسهٔ متفاوت داشته باشد", ErrPasswordWeak)
	}
	// تکرار یک بلوک کوتاه («Ab1!Ab1!») تنوع ظاهری می‌سازد ولی آنتروپی واقعی ندارد.
	if repeated(password) != "" {
		return fmt.Errorf("%w: رمز نباید تکرار یک الگوی کوتاه باشد", ErrPasswordWeak)
	}
	return nil
}

// repeated بلوک تکرارشوندهٔ رمز را برمی‌گرداند؛ رشتهٔ خالی یعنی الگوی تکرارشونده نیست.
// مقایسه بی‌توجه به بزرگی/کوچکی حروف است، چون «Ab1!ab1!» هم همان الگوی تکراری است.
func repeated(password string) string {
	normalized := strings.ToLower(password)
	for size := 1; size <= len(normalized)/2; size++ {
		if len(normalized)%size != 0 {
			continue
		}
		block := normalized[:size]
		if strings.Repeat(block, len(normalized)/size) == normalized {
			return block
		}
	}
	return ""
}

// VerifyPassword رمز ورودی را با درهم ذخیره‌شده (Argon2id یا PBKDF2 قدیمی) می‌سنجد.
func VerifyPassword(encoded, password string) bool {
	ok, _ := VerifyAndRehash(encoded, password)
	return ok
}

// VerifyAndRehash رمز را می‌سنجد و اگر درهم ذخیره‌شده از سیاست جاری ضعیف‌تر باشد،
// درهم تازه (Argon2id) برمی‌گرداند تا فراخوان آن را جایگزین کند.
//
// چرا ارتقا در زمان ورود: سامانه رمز خام ندارد، پس تنها لحظه‌ای که رمز در دست است
// همین ورود است. این‌گونه نصب‌های قدیمی بدون بازنشانی دستی رمزها ارتقا می‌یابند.
func VerifyAndRehash(encoded, password string) (ok bool, upgraded string) {
	if len(password) > maxPasswordBytes {
		return false, ""
	}
	switch {
	case strings.HasPrefix(encoded, "$"+SchemeArgon2id+"$"):
		if !verifyArgon2id(encoded, password) {
			return false, ""
		}
		if !NeedsRehash(encoded) {
			return true, ""
		}
		fresh, err := HashPassword(password)
		if err != nil {
			return true, ""
		}
		return true, fresh
	case strings.HasPrefix(encoded, schemePBKDF2+"$"):
		if !verifyPBKDF2(encoded, password) {
			return false, ""
		}
		fresh, err := HashPassword(password)
		if err != nil {
			return true, ""
		}
		return true, fresh
	default:
		return false, ""
	}
}

// NeedsRehash آیا درهم ذخیره‌شده از سیاست جاری (Argon2id با همین پارامترها) ضعیف‌تر است.
func NeedsRehash(encoded string) bool {
	if !strings.HasPrefix(encoded, "$"+SchemeArgon2id+"$") {
		return true
	}
	params, _, _, ok := parseArgon2id(encoded)
	if !ok {
		return true
	}
	return params.version != argonVersion ||
		params.memory < argonMemory ||
		params.time < argonTime ||
		params.threads < argonThreads ||
		params.keyLen < argonKeyLen
}

type argonParams struct {
	version uint32
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
}

func parseArgon2id(encoded string) (argonParams, []byte, []byte, bool) {
	parts := strings.Split(strings.TrimPrefix(encoded, "$"), "$")
	if len(parts) != 5 || parts[0] != SchemeArgon2id {
		return argonParams{}, nil, nil, false
	}
	if !strings.HasPrefix(parts[1], "v=") {
		return argonParams{}, nil, nil, false
	}
	version, err := strconv.ParseUint(parts[1][2:], 10, 32)
	if err != nil {
		return argonParams{}, nil, nil, false
	}
	var memory, iter uint64
	var threads uint64
	for _, pair := range strings.Split(parts[2], ",") {
		name, value, found := strings.Cut(pair, "=")
		if !found {
			return argonParams{}, nil, nil, false
		}
		n, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
		if err != nil {
			return argonParams{}, nil, nil, false
		}
		switch name {
		case "m":
			memory = n
		case "t":
			iter = n
		case "p":
			threads = n
		default:
			return argonParams{}, nil, nil, false
		}
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(salt) == 0 {
		return argonParams{}, nil, nil, false
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(key) == 0 {
		return argonParams{}, nil, nil, false
	}
	return argonParams{
		version: uint32(version),
		memory:  uint32(memory),
		time:    uint32(iter),
		threads: uint8(threads),
		keyLen:  uint32(len(key)),
	}, salt, key, true
}

func verifyArgon2id(encoded, password string) bool {
	params, salt, want, ok := parseArgon2id(encoded)
	if !ok {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, params.keyLen)
	return subtle.ConstantTimeCompare(got, want) == 1
}

// verifyPBKDF2 درهم‌های نسخهٔ پیشین را می‌سنجد (قالب: pbkdf2_sha256$iter$salt$key).
func verifyPBKDF2(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != schemePBKDF2 {
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
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyHash درهم ساختگی برای یکسان‌سازی زمان پاسخ، تا نام کاربری ناموجود از رمز
// نادرست با زمان پاسخ تشخیص داده نشود.
var (
	dummyOnce sync.Once
	dummyVal  string
)

// VerifyDummy یک درهم‌سازی ساختگی می‌سنجد و همیشه نادرست برمی‌گرداند.
//
// چرا لازم است: اگر برای کاربر ناموجود، درهم‌سازی انجام نشود، پاسخ در چند میلی‌ثانیه
// برمی‌گردد ولی برای کاربر موجود ده‌ها میلی‌ثانیه طول می‌کشد؛ همین اختلاف برای
// شمارش کاربران کافی است.
func VerifyDummy(password string) bool {
	dummyOnce.Do(func() {
		if hash, err := HashPassword("hrms-dummy-password-for-timing"); err == nil {
			dummyVal = hash
		}
	})
	_ = VerifyPassword(dummyVal, password)
	return false
}
