// آزمون سخت‌سازی توکن دسترسی (JWT) — سند: docs/security-spec-fa.md بند REQ-08.

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"hrms/internal/contracts"
)

func testSigner(t *testing.T) *Signer {
	t.Helper()
	signer, err := NewSignerChecked([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("ساخت امضاکننده ناموفق: %v", err)
	}
	return signer
}

func testClaims(kind string, now time.Time) Claims {
	return Claims{
		Subject:     "admin",
		Role:        contracts.RoleAdmin,
		DisplayName: "مدیر سیستم",
		Kind:        kind,
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(30 * time.Minute).Unix(),
		ID:          "jti-1",
	}
}

// TestREQ08TokenHardening می‌سنجد: توکن معتبر پذیرفته شود؛ توکن با الگوریتم none،
// امضای دست‌کاری‌شده، صادرکنندهٔ ناشناخته، بدون انقضا و منقضی رد شوند؛ و کلید کوتاه
// در مسیر استقرار پذیرفته نشود.
func TestREQ08TokenHardening(t *testing.T) {
	signer := testSigner(t)
	now := time.Now()

	token, err := signer.Sign(testClaims(KindAccess, now))
	if err != nil {
		t.Fatalf("امضای توکن ناموفق: %v", err)
	}
	claims, err := signer.Verify(token, now)
	if err != nil {
		t.Fatalf("توکن معتبر رد شد: %v", err)
	}
	if claims.Subject != "admin" || claims.Kind != KindAccess {
		t.Fatalf("محتوای توکن نادرست بازخوانی شد: %+v", claims)
	}

	// ۱) الگوریتم none با امضای خالی.
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := strings.Split(token, ".")[1]
	noneToken := header + "." + payload + "."
	if _, err := signer.Verify(noneToken, now); err == nil {
		t.Fatal("توکن alg=none پذیرفته شد")
	}

	// ۲) دست‌کاری بدنه با امضای توکن اصلی.
	tampered := strings.Split(token, ".")
	tampered[1] = base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"admin","role":"admin","kind":"access","iss":"hrms","aud":"hrms-web","iat":1,"exp":9999999999,"jti":"x"}`))
	if _, err := signer.Verify(strings.Join(tampered, "."), now); err == nil {
		t.Fatal("توکن دست‌کاری‌شده پذیرفته شد")
	}

	// ۳) صادرکننده/گیرندهٔ ناشناخته با امضای معتبر همان کلید.
	foreign := signTestPayload(t, []byte("0123456789abcdef0123456789abcdef"), map[string]any{
		"sub": "admin", "role": "admin", "kind": "access",
		"iss": "other-system", "aud": TokenAudience,
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(), "jti": "y",
	})
	if _, err := signer.Verify(foreign, now); err == nil {
		t.Fatal("توکن با صادرکنندهٔ ناشناخته پذیرفته شد")
	}

	// ۴) بدون ادعای انقضا.
	noExp := signTestPayload(t, []byte("0123456789abcdef0123456789abcdef"), map[string]any{
		"sub": "admin", "role": "admin", "kind": "access",
		"iss": TokenIssuer, "aud": TokenAudience, "iat": now.Unix(), "jti": "z",
	})
	if _, err := signer.Verify(noExp, now); err == nil {
		t.Fatal("توکن بدون انقضا پذیرفته شد")
	}

	// ۵) منقضی.
	expired, err := signer.Sign(testClaims(KindAccess, now.Add(-2*time.Hour)))
	if err != nil {
		t.Fatalf("امضای توکن منقضی ناموفق: %v", err)
	}
	if _, err := signer.Verify(expired, now); err == nil {
		t.Fatal("توکن منقضی پذیرفته شد")
	}

	// ۶) کلید کوتاه.
	if _, err := NewSignerChecked([]byte("short")); err == nil {
		t.Fatal("کلید کوتاه در مسیر استقرار باید رد شود")
	}
	if err := ValidateSecret(make([]byte, SecretMinBytes)); err != nil {
		t.Fatalf("کلید هم‌اندازهٔ کمینه نباید رد شود: %v", err)
	}
}

// signTestPayload یک توکن با محتوای دلخواه و امضای معتبر می‌سازد.
func signTestPayload(t *testing.T, secret []byte, payload map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("سریال‌سازی بدنه ناموفق: %v", err)
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	input := header + "." + body
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
