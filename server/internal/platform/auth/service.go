// سرویس ورود، کاربران و تمدید نشست؛ هش رمز با PBKDF2 — نقطهٔ حیاتی تست نفوذ.

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"hrms/internal/contracts"
)

// خطاهای ورود.
var (
	ErrBadCredentials = errors.New("نام کاربری یا رمز عبور نادرست است")
	ErrUserDisabled   = errors.New("حساب کاربری کاربر غیرفعال است")
)

// Account یک کاربر سامانه با رمز درهم‌شده.
type Account struct {
	Username     string
	PasswordHash string
	User         contracts.User
	Disabled     bool
}

// Service ورود، تمدید نشست و نگهداری کاربران.
type Service struct {
	signer     *Signer
	accounts   map[string]Account
	mu         sync.RWMutex
	accessTTL  time.Duration
	refreshTTL time.Duration
	clock      func() time.Time
}

// NewService سرویس احراز هویت می‌سازد.
func NewService(secret []byte, accounts []Account) *Service {
	index := make(map[string]Account, len(accounts))
	for _, a := range accounts {
		index[a.Username] = a
	}
	return &Service{
		signer:     NewSigner(secret),
		accounts:   index,
		accessTTL:  30 * time.Minute,
		refreshTTL: 14 * 24 * time.Hour,
		clock:      time.Now,
	}
}

// WithClock برای آزمون‌پذیری، منبع زمان را جایگزین می‌کند.
func (s *Service) WithClock(clock func() time.Time) *Service {
	s.clock = clock
	return s
}

// AccessTTL مدت اعتبار توکن دسترسی.
func (s *Service) AccessTTL() time.Duration { return s.accessTTL }

// Login ورود با نام کاربری و رمز.
//
// سه نکتهٔ امنیتی که عمدی‌اند:
//   - برای نام کاربری ناموجود هم یک درهم‌سازی ساختگی سنجیده می‌شود تا زمان پاسخ با
//     حالت «رمز نادرست» یکی بماند و شمارش کاربران ممکن نباشد.
//   - پیام یکسان برای کاربر ناموجود و رمز نادرست.
//   - درهم‌های قدیمی در همین ورود به Argon2id ارتقا می‌یابند (تنها لحظهٔ دسترسی به رمز).
func (s *Service) Login(username, password string) (contracts.Session, error) {
	s.mu.RLock()
	account, ok := s.accounts[username]
	s.mu.RUnlock()

	if !ok {
		VerifyDummy(password)
		return contracts.Session{}, ErrBadCredentials
	}
	verified, upgraded := VerifyAndRehash(account.PasswordHash, password)
	if !verified {
		return contracts.Session{}, ErrBadCredentials
	}
	if account.Disabled {
		return contracts.Session{}, ErrUserDisabled
	}
	if upgraded != "" {
		s.rehash(username, upgraded)
	}
	return s.issue(account.User)
}

// rehash درهم رمز را با نسخهٔ قوی‌تر جایگزین می‌کند.
func (s *Service) rehash(username, hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[username]
	if !ok {
		return
	}
	account.PasswordHash = hash
	s.accounts[username] = account
}

// passwordHash درهم ذخیره‌شدهٔ یک حساب را برمی‌گرداند (برای حسابرسی و آزمون؛ هرگز
// از API بیرون نمی‌رود).
func (s *Service) passwordHash(username string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[username]
	if !ok {
		return "", false
	}
	return account.PasswordHash, true
}

// Refresh نشست تازه از روی توکن تازه‌سازی صادر می‌کند.
func (s *Service) Refresh(refreshToken string) (contracts.Session, error) {
	claims, err := s.signer.Verify(refreshToken, s.clock())
	if err != nil {
		return contracts.Session{}, err
	}
	if claims.Kind != KindRefresh {
		return contracts.Session{}, ErrTokenKind
	}
	s.mu.RLock()
	account, ok := s.accounts[claims.Subject]
	s.mu.RUnlock()
	if !ok {
		return contracts.Session{}, ErrBadCredentials
	}
	if account.Disabled {
		return contracts.Session{}, ErrUserDisabled
	}
	return s.issue(account.User)
}

// User کاربر را با نام کاربری برمی‌گرداند.
func (s *Service) User(username string) (contracts.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[username]
	return account.User, ok
}

// Users فهرست کاربران سامانه را به ترتیب نام کاربری برمی‌گرداند.
//
// ماژول تنظیمات این فهرست را برای صفحهٔ مدیریت کاربران می‌خواند؛ رمز درهم‌شده هرگز
// بیرون نمی‌رود.
func (s *Service) Users() []contracts.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]contracts.User, 0, len(s.accounts))
	for _, a := range s.accounts {
		out = append(out, a.User)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Username < out[j].Username })
	return out
}

// VerifyAccess توکن دسترسی را بازبینی می‌کند.
func (s *Service) VerifyAccess(token string) (Claims, error) {
	claims, err := s.signer.Verify(token, s.clock())
	if err != nil {
		return Claims{}, err
	}
	if claims.Kind != KindAccess {
		return Claims{}, ErrTokenKind
	}
	return claims, nil
}

func (s *Service) issue(user contracts.User) (contracts.Session, error) {
	now := s.clock()
	access, err := s.signer.Sign(Claims{
		Subject:      user.Username,
		Role:         user.Role,
		DisplayName:  user.DisplayName,
		EmployeeID:   user.EmployeeID,
		DepartmentID: user.DepartmentID,
		Kind:         KindAccess,
		IssuedAt:     now.Unix(),
		ExpiresAt:    now.Add(s.accessTTL).Unix(),
		ID:           newID(),
	})
	if err != nil {
		return contracts.Session{}, fmt.Errorf("صدور توکن دسترسی ناموفق: %w", err)
	}
	refresh, err := s.signer.Sign(Claims{
		Subject:      user.Username,
		Role:         user.Role,
		DisplayName:  user.DisplayName,
		EmployeeID:   user.EmployeeID,
		DepartmentID: user.DepartmentID,
		Kind:         KindRefresh,
		IssuedAt:     now.Unix(),
		ExpiresAt:    now.Add(s.refreshTTL).Unix(),
		ID:           newID(),
	})
	if err != nil {
		return contracts.Session{}, fmt.Errorf("صدور توکن تازه‌سازی ناموفق: %w", err)
	}
	return contracts.Session{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    now.Add(s.accessTTL).Unix(),
		User:         user,
	}, nil
}

func newID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("t-%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// demoHashOnce درهم رمز نمایشی را یک‌بار می‌سازد و برای همهٔ حساب‌های نمونه به کار
// می‌برد. چرا: Argon2id عمداً گران است و آزمون‌ها ده‌ها بار وابستگی می‌سازند. این
// حساب‌ها رمز عمومی «demo» دارند و فقط در حالت توسعه/نمایش ساخته می‌شوند، پس نمک
// یکسان نشت معناداری ندارد؛ حساب‌های واقعی از مسیر settings/localstore ساخته می‌شوند
// و هرکدام نمک تازه می‌گیرند.
var (
	demoHashOnce sync.Once
	demoHash     string
	demoHashErr  error
)

// SeedAccounts کاربران نمونهٔ سامانه — همان حساب‌های بستهٔ طراحی (رمز همه: demo).
//
// این تابع فقط برای حالت توسعه و نمایش است؛ در استقرار واقعی کاربران از ماژول
// تنظیمات و با رمزهای سازمانی ساخته می‌شوند.
func SeedAccounts() ([]Account, error) {
	defs := []struct {
		username string
		user     contracts.User
	}{
		{"admin", contracts.User{Username: "admin", DisplayName: "مدیر سیستم", Role: contracts.RoleAdmin}},
		{"hr.manager", contracts.User{Username: "hr.manager", DisplayName: "سارا محمدی", Role: contracts.RoleHRManager, EmployeeID: "EMP-001", DepartmentID: "D2"}},
		{"hr.spec", contracts.User{Username: "hr.spec", DisplayName: "مریم نوری", Role: contracts.RoleHRSpecialist, EmployeeID: "EMP-008", DepartmentID: "D2"}},
		{"line.mgr", contracts.User{Username: "line.mgr", DisplayName: "آرش رضایی", Role: contracts.RoleLineManager, EmployeeID: "EMP-002", DepartmentID: "D1"}},
		{"employee", contracts.User{Username: "employee", DisplayName: "فاطمه صادقی", Role: contracts.RoleEmployee, EmployeeID: "EMP-010", DepartmentID: "D1"}},
	}
	demoHashOnce.Do(func() {
		demoHash, demoHashErr = HashPassword("demo")
	})
	if demoHashErr != nil {
		return nil, demoHashErr
	}
	accounts := make([]Account, 0, len(defs))
	for _, d := range defs {
		accounts = append(accounts, Account{Username: d.username, PasswordHash: demoHash, User: d.user})
	}
	return accounts, nil
}
