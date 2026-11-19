// قفل موقت حساب پس از شکست‌های پی‌درپی ورود — کند کردن رمزشکنی توزیع‌شده.

package websecurity

import (
	"strings"
	"sync"
	"time"
)

// Guard شمار شکست‌های پی‌درپی هر حساب را نگه می‌دارد و پس از رسیدن به سقف، حساب را
// برای بازهٔ مشخصی قفل می‌کند.
//
// چرا قفل حساب در کنار محدودیت نرخ: محدودیت نرخ کلیدش نشانی است و با چند IP به‌راحتی
// دور زده می‌شود؛ قفل حساب کلیدش نام کاربری است، پس حملهٔ توزیع‌شده هم گران می‌شود.
// برای اینکه خود قفل به ابزار «قفل کردن حساب دیگران» تبدیل نشود، مهلت کوتاه و
// خودرها است و ورود موفق آن را صفر می‌کند.
type Guard struct {
	threshold int
	lockout   time.Duration
	now       func() time.Time

	mu      sync.Mutex
	entries map[string]*failure
}

type failure struct {
	count       int
	last        time.Time
	lockedUntil time.Time
}

// NewGuard قفل‌کننده می‌سازد؛ threshold کوچک‌تر از یک یعنی «قفل خاموش».
func NewGuard(threshold int, lockout time.Duration) *Guard {
	if lockout <= 0 {
		lockout = 15 * time.Minute
	}
	return &Guard{
		threshold: threshold,
		lockout:   lockout,
		now:       time.Now,
		entries:   map[string]*failure{},
	}
}

// WithClock منبع زمان را برای آزمون‌پذیری جایگزین می‌کند.
func (g *Guard) WithClock(now func() time.Time) *Guard {
	g.now = now
	return g
}

// Threshold سقف شکست پی‌درپی پیش از قفل.
func (g *Guard) Threshold() int { return g.threshold }

// Lockout مهلت قفل.
func (g *Guard) Lockout() time.Duration { return g.lockout }

func guardKey(username string) string { return strings.ToLower(strings.TrimSpace(username)) }

// Locked می‌گوید حساب اکنون قفل است و چقدر تا آزاد شدنش مانده.
func (g *Guard) Locked(username string) (bool, time.Duration) {
	if g.threshold <= 0 {
		return false, 0
	}
	now := g.now()
	g.mu.Lock()
	defer g.mu.Unlock()

	e, ok := g.entries[guardKey(username)]
	if !ok || e.lockedUntil.IsZero() {
		return false, 0
	}
	if now.Before(e.lockedUntil) {
		return true, e.lockedUntil.Sub(now)
	}
	// مهلت گذشته؛ شمارش از نو.
	delete(g.entries, guardKey(username))
	return false, 0
}

// Fail یک شکست ورود را ثبت می‌کند و می‌گوید آیا همین شکست حساب را قفل کرد.
func (g *Guard) Fail(username string) (locked bool, remaining time.Duration) {
	if g.threshold <= 0 {
		return false, 0
	}
	now := g.now()
	key := guardKey(username)

	g.mu.Lock()
	defer g.mu.Unlock()

	e, ok := g.entries[key]
	if !ok {
		e = &failure{}
		g.entries[key] = e
	}
	// شکست‌های قدیمی‌تر از مهلت قفل، شمارش تازه‌ای را شروع می‌کنند.
	if !e.last.IsZero() && now.Sub(e.last) > g.lockout {
		e.count = 0
	}
	e.count++
	e.last = now

	if e.count >= g.threshold {
		e.lockedUntil = now.Add(g.lockout)
		e.count = 0
		return true, g.lockout
	}
	return false, 0
}

// Succeed شمارش شکست‌های حساب را صفر می‌کند (ورود موفق = بازنشانی).
func (g *Guard) Succeed(username string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.entries, guardKey(username))
}

// Failures شمار شکست‌های ثبت‌شدهٔ حساب (برای گزارش و آزمون).
func (g *Guard) Failures(username string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	if e, ok := g.entries[guardKey(username)]; ok {
		return e.count
	}
	return 0
}
