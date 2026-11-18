// گردانندهٔ ارسال صندوق خروجی: ردیف‌های سررسیدشده را با امضای HMAC و تلاش مجدد
// به نشانی وبهوک مصرف‌کننده می‌فرستد.
//
// چرا این کار در پس‌زمینه است: سرعت پاسخ API نباید به سامانهٔ بیرونی گره بخورد. اگر
// مقصد خاموش باشد، رخداد در صندوق می‌ماند و با فاصلهٔ فزاینده دوباره تلاش می‌شود؛
// پس از بیشینهٔ تلاش‌ها «ناموفق ماندگار» ثبت می‌شود تا از فهرست مدیریتی قابل مشاهده و
// با یک درخواست، دوباره در صف گذاشتنی باشد.

package events

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"time"
)

// سرآیندهای ارسال؛ مصرف‌کننده با همین‌ها رخداد را بازبینی می‌کند.
const (
	// HeaderEvent نام رخداد (مثل employee.updated).
	HeaderEvent = "X-HRMS-Event"
	// HeaderEventID شناسهٔ یگانهٔ رخداد (برای حذف تکرار).
	HeaderEventID = "X-HRMS-Event-Id"
	// HeaderTimestamp زمان ارسال به ثانیهٔ یونیکس؛ در امضا هم می‌آید.
	HeaderTimestamp = "X-HRMS-Timestamp"
	// HeaderSignature امضای HMAC-SHA256 به شکل `sha256=<hex>`.
	HeaderSignature = "X-HRMS-Signature"
	// HeaderAttempt شمارهٔ تلاش (۱ برای اولین ارسال).
	HeaderAttempt = "X-HRMS-Attempt"
	// HeaderOrigin سرآیندی که مصرف‌کننده باید در درخواست‌های خودش به سامانه بگذارد
	// تا رخدادهای خودش دوباره ثبت نشوند (جلوگیری از حلقه).
	HeaderOrigin = "X-HRMS-Integration-Origin"
)

// Run گردانندهٔ ارسال را تا پایان عمر ctx اجرا می‌کند.
//
// اگر ارسال خاموش باشد یا انبار نداشته باشیم، بی‌درنگ و بی‌صدا برمی‌گردد؛ صدا زدنش
// همیشه بی‌خطر است.
func (b *Bus) Run(ctx context.Context) {
	if b == nil || b.store == nil {
		return
	}
	if !b.Enabled() {
		b.log.Info("ارسال رخداد به سامانهٔ بیرونی تنظیم نشده است؛ صندوق فقط ثبت می‌کند")
		return
	}
	client := &http.Client{Timeout: b.cfg.Timeout}
	ticker := time.NewTicker(b.cfg.Interval)
	defer ticker.Stop()
	b.log.Info("گردانندهٔ صندوق خروجی راه افتاد",
		"url", b.cfg.URL, "intervalSeconds", int(b.cfg.Interval.Seconds()), "maxAttempts", b.cfg.MaxAttempts)

	for {
		select {
		case <-ctx.Done():
			b.log.Info("گردانندهٔ صندوق خروجی متوقف شد")
			return
		case <-ticker.C:
			b.drain(ctx, client)
		}
	}
}

// DrainOnce یک دور ارسال را همین حالا انجام می‌دهد (برای آزمون و اجرای دستی).
func (b *Bus) DrainOnce(ctx context.Context) (sent, failed int) {
	if b == nil || b.store == nil || !b.Enabled() {
		return 0, 0
	}
	client := &http.Client{Timeout: b.cfg.Timeout}
	return b.drain(ctx, client)
}

// drain یک دور: خواندن سررسیدها و تلاش برای هرکدام به ترتیب.
func (b *Bus) drain(ctx context.Context, client *http.Client) (sent, failed int) {
	now := b.cfg.Now()
	rows, err := b.store.Due(ctx, now, b.cfg.Batch)
	if err != nil {
		b.log.Error("خواندن صف رخداد ناموفق بود", "error", err)
		return 0, 0
	}
	for _, row := range rows {
		if err := b.deliver(ctx, client, row); err != nil {
			failed++
			continue
		}
		sent++
	}
	if len(rows) > 0 {
		b.log.Info("دور ارسال رخداد تمام شد", "due", len(rows), "sent", sent, "failed", failed)
	}
	return sent, failed
}

// deliver یک ردیف را می‌فرستد و نتیجه را در انبار ثبت می‌کند.
func (b *Bus) deliver(ctx context.Context, client *http.Client, row Delivery) error {
	now := b.cfg.Now()
	payload, err := body(row, now)
	if err != nil {
		b.recordFailure(ctx, row, "ساخت بدنهٔ رخداد ناموفق: "+err.Error(), now)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.cfg.URL, bytes.NewReader(payload))
	if err != nil {
		b.recordFailure(ctx, row, "ساخت درخواست ناموفق: "+err.Error(), now)
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", "hrms-events/1")
	req.Header.Set(HeaderEvent, row.Event.Name)
	req.Header.Set(HeaderEventID, row.Event.ID)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(now.Unix(), 10))
	req.Header.Set(HeaderAttempt, strconv.Itoa(row.Attempts+1))
	req.Header.Set(HeaderSignature, "sha256="+Sign(b.cfg.Secret, now, payload))

	resp, err := client.Do(req)
	if err != nil {
		b.recordFailure(ctx, row, "ارسال ناموفق: "+err.Error(), now)
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := b.store.MarkDelivered(ctx, row.ID, now); err != nil {
			b.log.Error("ثبت تحویل رخداد ناموفق بود", "id", row.ID, "error", err)
			return err
		}
		b.mu.Lock()
		b.delivered++
		b.mu.Unlock()
		return nil
	}
	err = errStatus(resp.StatusCode)
	b.recordFailure(ctx, row, err.Error(), now)
	return err
}

// recordFailure تلاش ناموفق را ثبت و در صورت لزوم ردیف را «ناموفق ماندگار» می‌کند.
func (b *Bus) recordFailure(ctx context.Context, row Delivery, message string, now time.Time) {
	attempts := row.Attempts + 1
	status := StatusPending
	next := now.Add(backoff(attempts))
	if attempts >= b.cfg.MaxAttempts {
		status = StatusFailed
		// زمان تلاش بعدی «نزدیک» گذاشته می‌شود تا اگر مدیر دستی در صف گذاشت،
		// بلافاصله ارسال شود؛ تا آن موقع وضعیت failed جلوی ارسال را می‌گیرد.
		next = now
	}
	if err := b.store.MarkAttempt(ctx, row.ID, attempts, next, message, status); err != nil {
		b.log.Error("ثبت تلاش ناموفق رخداد ناموفق بود", "id", row.ID, "error", err)
		return
	}
	b.mu.Lock()
	b.failed++
	b.mu.Unlock()
	b.log.Warn("ارسال رخداد ناموفق بود",
		"event", row.Event.Name, "id", row.ID, "attempt", attempts,
		"status", status, "nextAttemptIn", next.Sub(now).String(), "error", message)
}

// errStatus خطای پاسخ غیرموفق سامانهٔ بیرونی.
type errStatus int

func (e errStatus) Error() string { return "پاسخ ناموفق سامانهٔ بیرونی: " + strconv.Itoa(int(e)) }