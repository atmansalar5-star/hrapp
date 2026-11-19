// تولید رمز موقت برای نخستین راه‌اندازی عملیاتی.

package localstore

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// passwordAlphabet نویسه‌های مجاز رمز موقت: بدون نویسه‌های گیج‌کننده (O/0، l/1، I).
const passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#%*+?"

// RandomPassword رمز موقت ۲۰ نویسه‌ای می‌سازد که سیاست قدرت رمز را پاس می‌کند.
//
// کاربرد: نصب تازهٔ عملیاتی نباید با رمز پیش‌فرض مستندشده بالا بیاید. سرور در نخستین
// اجرا رمزی تصادفی می‌سازد، آن را یک‌بار در لاگ راه‌اندازی اعلام می‌کند و راهبر باید
// آن را همان روز عوض کند (سند docs/operational-spec-fa.md بند OPS-03).
func RandomPassword() (string, error) {
	const length = 20
	out := make([]byte, length)
	max := big.NewInt(int64(len(passwordAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("تولید رمز تصادفی ناموفق: %w", err)
		}
		out[i] = passwordAlphabet[n.Int64()]
	}
	return string(out), nil
}
