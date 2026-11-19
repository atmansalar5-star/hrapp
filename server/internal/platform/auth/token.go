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
	ErrSecretTooShort = errors.New("کلید امضای توکن کوتاه‌تر از حد لازم است")
)

// نوع توکن‌ها: توکن دسترسی کوتاه‌عمر و توکن تازه‌سازی بلندعمر.
const (
	KindAccess  = "access"
	KindRefresh = "refresh"
)

// ادعاهای ثابت توکن. سامانه یک API دارد، پس issuer و audience ثابت‌اند؛ ولی بودنشان
// در توکن و بازبینی‌شان جلوی «توکنِ سامانهٔ دیگر با کلید مشترک» و بازپخش کور را می‌گیرد.
const (
	TokenIssuer   = "hrms"
	TokenAudience = "hrms-web"
	// SecretMinBytes کمینهٔ طول کلید HS256؛ کوتاه‌تر یعنی امضا با فضای جست‌وجوی کوچک.
	SecretMinBytes = 32
	// clockSkew تحمل اختلاف ساعت بین سرورها در بازبینی iat.
	clockSkew = 60 * time.Second
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
	Iss  string `json:"iss"`
	Aud  string `json:"aud"`
	Iat  int64  `json:"iat"`
	Exp  int64  `json:"exp"`
	Jti  string `json:"jti"`
}

// ValidateSecret طول کلید امضا را می‌سنجد. کلید کوتاه (یا پیش‌فرض مستندشده) در استقرار
// واقعی پذیرفتنی نیست؛ محاسبهٔ امضا با کلید کوتاه، شکستنش را عملی می‌کند.
func ValidateSecret(secret []byte) error {
	if len(secret) < SecretMinBytes {
		return fmt.Errorf("%w: %d بایت (کمینه %d)", ErrSecretTooShort, len(secret), SecretMinBytes)
	}
	return nil
}

// NewSigner امضاکننده با کلید مخفی می‌سازد.
func NewSigner(secret []byte) *Signer {
	return &Signer{secret: secret}
}

// NewSignerChecked مانند NewSigner است ولی کلید کوتاه را رد می‌کند. مسیر راه‌اندازی
// استقرار باید همین را به کار ببرد، نه NewSigner.
func NewSignerChecked(secret []byte) (*Signer, error) {
	if err := ValidateSecret(secret); err != nil {
		return nil, err
	}
	return &Signer{secret: secret}, nil
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
		Iss:  TokenIssuer,
		Aud:  TokenAudience,
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
//
// ترتیب بازبینی عمدی است: نخست امضا (پس از خواندن سرآیند و رد الگوریتم نادرست)، سپس
// ادعاها. سرآیند پیش از هر چیز سنجیده می‌شود تا توکن با alg دیگر (مثل none) حتی به
// مرحلهٔ مقایسهٔ امضا نرسد.
func (s *Signer) Verify(token string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrTokenMalformed
	}
	if err := checkHeader(parts[0]); err != nil {
		return Claims{}, err
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
	if p.Iss != TokenIssuer || p.Aud != TokenAudience {
		return Claims{}, fmt.Errorf("%w: صادرکننده یا گیرندهٔ توکن ناشناخته است", ErrTokenMalformed)
	}
	if p.Exp <= 0 {
		return Claims{}, fmt.Errorf("%w: توکن بدون انقضا پذیرفته نمی‌شود", ErrTokenMalformed)
	}
	if p.Exp < now.Unix() {
		return Claims{}, ErrTokenExpired
	}
	if p.Iat > now.Add(clockSkew).Unix() {
		return Claims{}, fmt.Errorf("%w: زمان صدور توکن در آینده است", ErrTokenMalformed)
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

// checkHeader الگوریتم و نوع توکن را می‌سنجد.
func checkHeader(segment string) error {
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return ErrTokenMalformed
	}
	var h jwtHeader
	if err := json.Unmarshal(raw, &h); err != nil {
		return ErrTokenMalformed
	}
	if h.Alg != "HS256" {
		return fmt.Errorf("%w: الگوریتم امضا باید HS256 باشد", ErrTokenMalformed)
	}
	if h.Typ != "" && h.Typ != "JWT" {
		return fmt.Errorf("%w: نوع توکن باید JWT باشد", ErrTokenMalformed)
	}
	return nil
}

func (s *Signer) mac(input string) []byte {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(input))
	return m.Sum(nil)
}

func encodeSegment(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
