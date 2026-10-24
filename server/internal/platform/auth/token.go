// صدور و اعتبارسنجی توکن دسترسی (JWT) با کلید HRMS_JWT_SECRET — نقطهٔ حیاتی تست نفوذ.

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hrms/internal/contracts"
)

// خطاهای بازبینی توکن.
var (
	ErrTokenMalformed = errors.New("توکن نامعتبر است")
	ErrTokenSignature = errors.New("امضای توکن معتبر نیست")
	ErrTokenExpired   = errors.New("نشست منقضی شده است؛ دوباره وارد شوید")
	ErrTokenKind      = errors.New("نوع توکن برای این عملیات مناسب نیست")
)

// نوع توکن‌ها: توکن دسترسی کوتاه‌عمر و توکن تازه‌سازی بلندعمر.
const (
	KindAccess  = "access"
	KindRefresh = "refresh"
)

// Signer توکن‌های HS256 را امضا و بازبینی می‌کند.
type Signer struct {
	secret []byte
}

// Claims محتوای توکن.
type Claims struct {
	Subject      string
	Role         contracts.Role
	DisplayName  string
	EmployeeID   string
	DepartmentID string
	Kind         string
	IssuedAt     int64
	ExpiresAt    int64
	ID           string
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtPayload struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Name string `json:"name,omitempty"`
	Emp  string `json:"emp,omitempty"`
	Dept string `json:"dept,omitempty"`
	Kind string `json:"kind"`
	Iat  int64  `json:"iat"`
	Exp  int64  `json:"exp"`
	Jti  string `json:"jti"`
}

// NewSigner امضاکننده با کلید مخفی می‌سازد.
func NewSigner(secret []byte) *Signer {
	return &Signer{secret: secret}
}

// Sign توکن را می‌سازد.
func (s *Signer) Sign(c Claims) (string, error) {
	header, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(jwtPayload{
		Sub:  c.Subject,
		Role: string(c.Role),
		Name: c.DisplayName,
		Emp:  c.EmployeeID,
		Dept: c.DepartmentID,
		Kind: c.Kind,
		Iat:  c.IssuedAt,
		Exp:  c.ExpiresAt,
		Jti:  c.ID,
	})
	if err != nil {
		return "", err
	}
	signingInput := encodeSegment(header) + "." + encodeSegment(payload)
	return signingInput + "." + encodeSegment(s.mac(signingInput)), nil
}

// Verify توکن را بازبینی می‌کند و محتوای آن را برمی‌گرداند.
func (s *Signer) Verify(token string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrTokenMalformed
	}
	signingInput := parts[0] + "." + parts[1]

	mac, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrTokenMalformed
	}
	if !hmac.Equal(mac, s.mac(signingInput)) {
		return Claims{}, ErrTokenSignature
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrTokenMalformed
	}
	var p jwtPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Claims{}, ErrTokenMalformed
	}
	if p.Exp < now.Unix() {
		return Claims{}, ErrTokenExpired
	}
	role := contracts.Role(p.Role)
	if !contracts.ValidRole(role) {
		return Claims{}, fmt.Errorf("%w: نقش ناشناخته", ErrTokenMalformed)
	}
	return Claims{
		Subject:      p.Sub,
		Role:         role,
		DisplayName:  p.Name,
		EmployeeID:   p.Emp,
		DepartmentID: p.Dept,
		Kind:         p.Kind,
		IssuedAt:     p.Iat,
		ExpiresAt:    p.Exp,
		ID:           p.Jti,
	}, nil
}

func (s *Signer) mac(input string) []byte {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(input))
	return m.Sum(nil)
}

func encodeSegment(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
