// میان‌افزار احراز هویت: خواندن توکن، تزریق کاربر به درخواست و مسیرهای عمومی — نقطهٔ حیاتی تست نفوذ.

package auth

import (
	"context"
	"net/http"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/httpx"
)

type principalKey struct{}

// Principal کاربر شناسایی‌شدهٔ درخواست جاری.
type Principal struct {
	User   contracts.User
	Claims Claims
}

// Middleware توکن موجود در هدر را (اگر باشد) تفسیر و در context می‌گذارد؛
// رد کردن درخواست‌های بی‌نشست کار RequireAuth است تا مسیرهای عمومی مانند ورود
// همچنان کار کنند.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			httpx.WriteError(w, http.StatusUnauthorized, contracts.CodeUnauthorized, nil)
			return
		}
		claims, err := s.VerifyAccess(strings.TrimSpace(parts[1]))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnauthorized, contracts.CodeUnauthorized, err.Error(), nil)
			return
		}
		user, ok := s.User(claims.Subject)
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, contracts.CodeUnauthorized, nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), Principal{User: user, Claims: claims})))
	})
}

// RequireAuth درخواست بی‌نشست را با 401 برمی‌گرداند.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFrom(r.Context()); !ok {
			httpx.WriteError(w, http.StatusUnauthorized, contracts.CodeUnauthorized, nil)
			return
		}
		next(w, r)
	}
}

// WithPrincipal پرینسیپال را در context می‌گذارد (برای آزمون‌ها هم مفید است).
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom پرینسیپال را از context می‌خواند.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// UserFrom میان‌بُری برای خواندن کاربر جاری.
func UserFrom(ctx context.Context) (contracts.User, bool) {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return contracts.User{}, false
	}
	return p.User, true
}

// BearerToken توکن را از هدر می‌خواند (برای آزمون‌های یکپارچه).
func BearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
