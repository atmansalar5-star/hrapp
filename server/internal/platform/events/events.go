// Package events صندوق خروجی رویداد (outbox) و ارسال آن به سامانه‌های بیرونی.
//
// چرا outbox و نه ارسال مستقیم هنگام درخواست: اگر فرستادن وبهوک داخل مسیر درخواست
// انجام شود، کندی یا قطعی سامانهٔ بیرونی مستقیماً به کاربر برمی‌گردد و ممکن است تغییرِ
// ثبت‌شده بی‌اطلاع از دست برود. اینجا رویداد نخست در همان پایگاه‌دادهٔ سامانه ثبت
// می‌شود (تضمین‌شده، تراکنشی) و بعد یک گردانندهٔ پس‌زمینه آن را با تلاش مجدد و
// فاصلهٔ فزاینده تحویل می‌دهد. نبودِ این سرویس هیچ اثری بر کار سرور ندارد.
//
// قاعدهٔ مهم: این بستهٔ هیچ قاعدهٔ کسب‌وکاری ندارد. فقط «چه تغییری از API گذشت» را
// گزارش می‌کند؛ تفسیر و واکنش، کار مصرف‌کنندهٔ بیرونی (مثلاً n8n) است و پیامک/ایمیل
// نمی‌سازد. محاسبه‌های حقوقی و آیین‌نامه‌ای هم همان‌جای همیشگی، در rules، می‌مانند.
package events

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// وضعیت‌های تحویل یک رویداد در صندوق خروجی.
const (
	// StatusPending در انتظار ارسال (یا در انتظار تلاش بعدی).
	StatusPending = "pending"
	// StatusDelivered به مصرف‌کننده تحویل شد.
	StatusDelivered = "delivered"
	// StatusFailed پس از مصرف همهٔ تلاش‌ها ناموفق ماند؛ برای بازبینی دستی می‌ماند.
	StatusFailed = "failed"
)

// Event یک رویداد تغییر در سامانه.
//
// فیلدها عمداً «دادهٔ رخداد» هستند و نه دامنه: ماژول، مسیر، کاربر و شناسهٔ موضوع.
// بدنهٔ کامل تغییر عمداً حمل نمی‌شود تا رویداد سبک بماند و مصرف‌کننده در صورت نیاز
// خودش با همان توکن محدود، جزئیات را از API بخواند (یک منبع حقیقت، بدون تکرار داده).
type Event struct {
	// ID شناسهٔ یگانهٔ رخداد (برای حذف تکرار در مصرف‌کننده).
	ID string
	// Name نام رخداد، مثل `employee.updated` یا `payroll.action`.
	Name string
	// OccurredAt زمان وقوع در سرور.
	OccurredAt time.Time
	// Actor نام کاربری کنشگر (ممیز).
	Actor string
	// Role نقش کنشگر.
	Role string
	// Module نخستین بخش مسیر API (کاربرگ‌ها: employee، payroll، contracts...).
	Module string
	// Method و Path و Status همان درخواست HTTP.
	Method string
	Path   string
	Status int
	// Subject شناسهٔ موضوع درخواست (مثل EMP-011)، اگر از مسیر قابل استخراج باشد.
	Subject string
	// Payload دادهٔ کمکی رخداد (پرس‌وجو، مدت اجرا، ...).
	Payload map[string]any
}

// Delivery یک ردیف صندوق خروجی: رخداد به‌همراه وضعیت تحویلش.
type Delivery struct {
	ID            int64
	Event         Event
	Attempts      int
	NextAttemptAt time.Time
	DeliveredAt   *time.Time
	LastError     string
	// lastStatus وضعیت ثبت‌شده در انبار (pending/delivered/failed).
	lastStatus string
}

// Status وضعیت تحویل این ردیف.
func (d Delivery) Status() string { return d.status() }

// Store انبار صندوق خروجی. دو پیاده‌سازی دارد: Postgres برای استقرار واقعی و
// حافظه‌ای برای توسعه و آزمون.
type Store interface {
	// Append رخداد را ثبت می‌کند. شناسهٔ تکراری نادیده گرفته می‌شود (بی‌اثر و بی‌خطا).
	Append(ctx context.Context, e Event) error
	// Due ردیف‌های «در انتظار» و سررسیدشده را به ترتیب زمان برمی‌گرداند.
	Due(ctx context.Context, now time.Time, limit int) ([]Delivery, error)
	// MarkDelivered ردیف را تحویل‌شده ثبت می‌کند.
	MarkDelivered(ctx context.Context, id int64, at time.Time) error
	// MarkAttempt تلاش ناموفق را ثبت می‌کند: شمار تلاش، زمان تلاش بعدی، پیام خطا و
	// وضعیت تازه. وضعیت یا StatusPending است (تلاش دوباره) یا StatusFailed (پایان).
	MarkAttempt(ctx context.Context, id int64, attempts int, next time.Time, message, status string) error
	// Recent آخرین ردیف‌ها را برمی‌گرداند؛ status خالی یعنی همهٔ وضعیت‌ها.
	Recent(ctx context.Context, status string, limit int) ([]Delivery, error)
	// Requeue ردیف را برای تلاش دوباره به صف برمی‌گرداند.
	Requeue(ctx context.Context, id int64, at time.Time) error
	// Counts شمار ردیف‌ها به تفکیک وضعیت.
	Counts(ctx context.Context) (map[string]int, error)
}

// Config تنظیمات گردانندهٔ ارسال.
type Config struct {
	// URL مقصد وبهوک؛ خالی یعنی ارسال خاموش است (رخدادها فقط ثبت می‌شوند).
	URL string
	// Secret کلید امضای HMAC بدنه.
	Secret string
	// Interval فاصلهٔ بررسی صف.
	Interval time.Duration
	// Batch بیشینهٔ ردیف در هر دور.
	Batch int
	// MaxAttempts بیشینهٔ تلاش برای هر رخداد پیش از «ناموفق ماندگار».
	MaxAttempts int
	// Timeout مهلت هر درخواست HTTP.
	Timeout time.Duration
	// Now ساعت سامانه (در آزمون تزریق می‌شود).
	Now func() time.Time
	// Log لاگر؛ nil یعنی لاگر پیش‌فرض.
	Log *slog.Logger
}

// DefaultConfig تنظیمات پیش‌فرض: هر ۱۵ ثانیه، ۸ تلاش، مهلت ۱۰ ثانیه.
func DefaultConfig() Config {
	return Config{
		Interval:    15 * time.Second,
		Batch:       50,
		MaxAttempts: 8,
		Timeout:     10 * time.Second,
	}
}

func (c Config) normalized() Config {
	d := DefaultConfig()
	if c.Interval <= 0 {
		c.Interval = d.Interval
	}
	if c.Batch <= 0 {
		c.Batch = d.Batch
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = d.MaxAttempts
	}
	if c.Timeout <= 0 {
		c.Timeout = d.Timeout
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}
	return c
}

// Bus صندوق خروجی سامانه: ثبت رخداد در درخواست‌ها و ارسال پس‌زمینهٔ آن.
type Bus struct {
	store Store
	cfg   Config
	log   *slog.Logger

	mu       sync.Mutex
	emitted  int
	skipped  int
	delivered int
	failed   int
}

// NewBus اتوبوس رخداد را می‌سازد. store می‌تواند nil باشد: در آن حالت Publish
// کاری نمی‌کند و Run هم بی‌درنگ برمی‌گردد (اجرای بدون پایگاه‌داده).
func NewBus(store Store, cfg Config) *Bus {
	cfg = cfg.normalized()
	log := cfg.Log.With("component", "events")
	return &Bus{store: store, cfg: cfg, log: log}
}

// Enabled آیا ارسال به سامانهٔ بیرونی تنظیم شده است.
func (b *Bus) Enabled() bool { return b != nil && b.cfg.URL != "" }

// URL مقصد وبهوک (برای گزارش وضعیت).
func (b *Bus) URL() string {
	if b == nil {
		return ""
	}
	return b.cfg.URL
}

// Interval فاصلهٔ بررسی صف.
func (b *Bus) Interval() time.Duration {
	if b == nil {
		return 0
	}
	return b.cfg.Interval
}

// MaxAttempts بیشینهٔ تلاش برای هر رخداد.
func (b *Bus) MaxAttempts() int {
	if b == nil {
		return 0
	}
	return b.cfg.MaxAttempts
}

// Store انبار صندوق؛ ممکن است nil باشد.
func (b *Bus) Store() Store {
	if b == nil {
		return nil
	}
	return b.store
}

// Publish یک رخداد را در صندوق ثبت می‌کند. خطا فقط لاگ می‌شود و به درخواست کاربر
// برنمی‌گردد: تغییر دامنه‌ای که انجام شده نباید به‌خاطر نبودِ سامانهٔ بیرونی شکست
// بخورد؛ ولی از دست رفتن رخداد نباید هم پنهان بماند.
func (b *Bus) Publish(ctx context.Context, e Event) {
	if b == nil || b.store == nil {
		return
	}
	if e.ID == "" {
		e.ID = NewID()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = b.cfg.Now()
	}
	if e.Name == "" {
		e.Name = "unknown"
	}
	if err := b.store.Append(ctx, e); err != nil {
		b.log.Error("ثبت رخداد در صندوق خروجی ناموفق بود", "event", e.Name, "error", err)
		return
	}
	b.mu.Lock()
	b.emitted++
	b.mu.Unlock()
	b.log.Debug("رخداد در صندوق ثبت شد", "event", e.Name, "subject", e.Subject, "actor", e.Actor)
}

// Skip شمارش رخدادهای کنارگذاشته‌شده را برای گزارش وضعیت نگه می‌دارد.
func (b *Bus) Skip() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.skipped++
	b.mu.Unlock()
}

// Stats شمار رخدادهای این اجرا و وضعیت صندوق را برمی‌گرداند.
func (b *Bus) Stats(ctx context.Context) (map[string]int, error) {
	counts := map[string]int{}
	if b == nil || b.store == nil {
		return counts, nil
	}
	rows, err := b.store.Counts(ctx)
	if err != nil {
		return counts, err
	}
	b.mu.Lock()
	counts["emittedNow"] = b.emitted
	counts["skippedNow"] = b.skipped
	counts["deliveredNow"] = b.delivered
	counts["failedNow"] = b.failed
	b.mu.Unlock()
	for status, n := range rows {
		counts[status] = n
	}
	return counts, nil
}

// NewID شناسهٔ یگانهٔ رخداد می‌سازد.
func NewID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buf)
}

// Sign امضای HMAC-SHA256 را می‌سازد: hex(hmac(secret, timestamp + "." + body)).
//
// چرا مُهر زمان در امضا می‌آید: بدون آن، هر کسی می‌توانست بدنهٔ امضاشدهٔ قدیمی را دوباره
// بفرستد (replay). مصرف‌کننده فقط باید مُهر زمانی تازه را بپذیرد و همان را در امضا
// بازبینی کند.
func Sign(secret string, timestamp time.Time, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify امضای رسیده را در برابر بدنه و مُهر زمان بازبینی می‌کند.
func Verify(secret string, timestamp time.Time, body []byte, signature string) bool {
	expected := Sign(secret, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// envelope بدنهٔ JSON که به سامانهٔ بیرونی فرستاده می‌شود.
type envelope struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	OccurredAt string         `json:"occurredAt"`
	Actor      string         `json:"actor"`
	Role       string         `json:"role"`
	Module     string         `json:"module"`
	Method     string         `json:"method"`
	Path       string         `json:"path"`
	Status     int            `json:"status"`
	Subject    string         `json:"subject"`
	Payload    map[string]any `json:"payload"`
	Attempt    int            `json:"attempt"`
	SentAt     string         `json:"sentAt"`
}

// body بدنهٔ JSON رخداد را می‌سازد.
func body(d Delivery, now time.Time) ([]byte, error) {
	payload := d.Event.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	return json.Marshal(envelope{
		ID:         d.Event.ID,
		Name:       d.Event.Name,
		OccurredAt: d.Event.OccurredAt.UTC().Format(time.RFC3339),
		Actor:      d.Event.Actor,
		Role:       d.Event.Role,
		Module:     d.Event.Module,
		Method:     d.Event.Method,
		Path:       d.Event.Path,
		Status:     d.Event.Status,
		Subject:    d.Event.Subject,
		Payload:    payload,
		Attempt:    d.Attempts + 1,
		SentAt:     now.UTC().Format(time.RFC3339),
	})
}

// backoff فاصلهٔ تلاش بعدی: ۵ ثانیه، ۱۰، ۲۰... تا سقف یک ساعت.
func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := 5 * time.Second
	for i := 1; i < attempt && d < time.Hour; i++ {
		d *= 2
	}
	if d > time.Hour {
		d = time.Hour
	}
	return d
}

// errNoStore خطای داخلی وقتی انبار تنظیم نشده باشد.
var errNoStore = errors.New("صندوق خروجی رخداد تنظیم نشده است")

// String برای لاگ خواناتر.
func (d Delivery) String() string {
	return fmt.Sprintf("%s#%d(%s,تلاش %d)", d.Event.Name, d.ID, d.Event.Subject, d.Attempts)
}