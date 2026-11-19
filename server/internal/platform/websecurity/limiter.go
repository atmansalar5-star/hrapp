// محدودیت نرخ درخواست به‌ازای هر کلید (نشانی کلاینت یا حساب) — کنترل ضد-DoS لایهٔ ۷.

package websecurity

import (
	"net/http"
	"sync"
	"time"
)

// Limiter سقف شمار درخواست یک کلید در یک بازهٔ زمانی ثابت را نگه می‌دارد.
//
// چرا پنجرهٔ ثابت و نه الگوریتم سنگین‌تر: هدف دفاع در برابر سیل ساده و
// فرهنگ‌نامه‌ای است، نه اندازه‌گیری دقیق نرخ. پنجرهٔ ثابت حافظهٔ O(۱) برای هر کلید
// می‌خواهد و در بدترین حالت دو برابر سقف را در مرز پنجره‌ها می‌گذارد — برای این هدف
// پذیرفتنی است و هیچ قفل سراسری‌ای روی مسیر درخواست‌های عادی نمی‌گذارد.
type Limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	start time.Time
	count int
}

// NewLimiter محدودکننده می‌سازد؛ limit کوچک‌تر از یک یعنی «همیشه اجازه».
func NewLimiter(limit int, window time.Duration) *Limiter {
	if window <= 0 {
		window = time.Minute
	}
	return &Limiter{
		limit:   limit,
		window:  window,
		now:     time.Now,
		buckets: map[string]*bucket{},
	}
}

// WithClock منبع زمان را برای آزمون‌پذیری جایگزین می‌کند.
func (l *Limiter) WithClock(now func() time.Time) *Limiter {
	l.now = now
	return l
}

// Limit سقف درخواست در هر بازه.
func (l *Limiter) Limit() int { return l.limit }

// Window بازهٔ شمارش.
func (l *Limiter) Window() time.Duration { return l.window }

// Allow یک درخواست را می‌شمارد. اگر سقف پر شده باشد allowed=false و retryAfter
// زمانی است که تا آزاد شدن پنجره مانده.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	if l.limit <= 0 {
		return true, 0
	}
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweepLocked(now)

	b, ok := l.buckets[key]
	if !ok || now.Sub(b.start) >= l.window {
		l.buckets[key] = &bucket{start: now, count: 1}
		return true, 0
	}
	if b.count >= l.limit {
		return false, l.window - now.Sub(b.start)
	}
	b.count++
	return true, 0
}

// Reset شمارش یک کلید را صفر می‌کند (مثلاً پس از ورود موفق).
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

// Len شمار کلیدهای در حال شمارش (برای آزمون و گزارش وضعیت).
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// sweepLocked کلیدهای منقضی را پاک می‌کند تا نقشهٔ حافظه با حملهٔ IPهای چرخشی رشد نکند.
func (l *Limiter) sweepLocked(now time.Time) {
	if len(l.buckets) < 1024 {
		return
	}
	if !l.lastSweep.IsZero() && now.Sub(l.lastSweep) < l.window {
		return
	}
	for key, b := range l.buckets {
		if now.Sub(b.start) >= l.window {
			delete(l.buckets, key)
		}
	}
	l.lastSweep = now
}

// WriteRetryAfter پاسخ ۴۲۹ استاندارد سامانه را با سرآیند Retry-After می‌نویسد.
//
// بدنه JSON است تا کلاینت وب همان قالب خطای بقیهٔ API را ببیند و پیام فارسی را
// مستقیم نمایش دهد.
func WriteRetryAfter(w http.ResponseWriter, retryAfter time.Duration, message string) {
	seconds := int(retryAfter.Seconds())
	if retryAfter > 0 && retryAfter < time.Second {
		seconds = 1
	}
	if seconds < 0 {
		seconds = 0
	}
	w.Header().Set("Retry-After", itoa(seconds))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"code":"rate_limited","message":"` + message + `","details":{"retryAfterSeconds":"` + itoa(seconds) + `"}}`))
}

// itoa تبدیل کوچک عدد به رشته بدون وابستگی به fmt در مسیر داغ پاسخ.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
