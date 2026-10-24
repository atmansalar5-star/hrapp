// Package audit لاگ حسابرسی فقط-افزودن با زنجیرهٔ هش.
// هر رخداد هش رخداد قبلی خود را در بر می‌گیرد، پس دست‌کاری تاریخی رخدادها قابل تشخیص است.
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Event یک رخداد حسابرسی.
type Event struct {
	Seq      int64  `json:"seq"`
	At       string `json:"at"`
	Actor    string `json:"actor"`
	Module   string `json:"module"`
	Action   string `json:"action"`
	Target   string `json:"target,omitempty"`
	Detail   string `json:"detail,omitempty"`
	PrevHash string `json:"prevHash"`
	Hash     string `json:"hash"`
}

// genesis هش آغازین زنجیره.
const genesis = "0000000000000000000000000000000000000000000000000000000000000000"

// Log دفتر رخدادها در حافظه؛ در استقرار واقعی همین ساختار روی جدول append-only نوشته می‌شود.
type Log struct {
	mu     sync.Mutex
	events []Event
	tip    string
	clock  func() time.Time
}

// New دفتر حسابرسی خالی می‌سازد.
func New() *Log {
	return &Log{tip: genesis, clock: time.Now}
}

// Append رخداد تازه را به زنجیره اضافه می‌کند و رخداد مهرشده را برمی‌گرداند.
func (l *Log) Append(actor, module, action, target, detail string) Event {
	l.mu.Lock()
	defer l.mu.Unlock()

	e := Event{
		Seq:      int64(len(l.events) + 1),
		At:       l.clock().UTC().Format(time.RFC3339),
		Actor:    actor,
		Module:   module,
		Action:   action,
		Target:   target,
		Detail:   detail,
		PrevHash: l.tip,
	}
	e.Hash = hashOf(e)
	l.events = append(l.events, e)
	l.tip = e.Hash
	return e
}

// All بازه‌ای از رخدادها (فقط-خواندنی) را برمی‌گرداند.
func (l *Log) All() []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Event, len(l.events))
	copy(out, l.events)
	return out
}

// ByTarget رخدادهای یک سند مشخص (ماژول + شناسهٔ سند) را به ترتیب ثبت برمی‌گرداند.
//
// چرا این متد لازم است: پروندهٔ کارمند باید «سابقهٔ تغییرات» هر سند را نشان دهد و
// بخشی از اسناد (احکام، تکفل) رخداد تغییر را در خودِ سند نگه نمی‌دارند؛ زنجیرهٔ
// فقط-افزودن حسابرسی همان رخدادها را با زمان، کاربر و شرح فارسی ثبت کرده است، پس
// منبع حقیقی سابقهٔ تغییر آن اسناد همین‌جاست و هیچ رخدادی ساخته نمی‌شود.
//
// در استقرار واقعی این پیمایش به پرس‌وجوی نمایه‌دار روی جدول append-only تبدیل
// می‌شود؛ این‌جا (دفتر درون‌حافظه‌ای) پیمایش خطی است.
func (l *Log) ByTarget(module, target string) []Event {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []Event
	for _, e := range l.events {
		if e.Module == module && e.Target == target {
			out = append(out, e)
		}
	}
	return out
}

// Len تعداد رخدادها.
func (l *Log) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.events)
}

// Verify زنجیره را بازمحاسبه می‌کند؛ اگر رخدادی دست‌کاری شده باشد خطا برمی‌گرداند.
func (l *Log) Verify() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	prev := genesis
	for i, e := range l.events {
		if e.PrevHash != prev {
			return fmt.Errorf("گسست زنجیره در رخداد %d: prevHash نامعتبر", e.Seq)
		}
		want := hashOf(e)
		if e.Hash != want {
			return fmt.Errorf("محتوای رخداد %d دست‌کاری شده است", e.Seq)
		}
		prev = e.Hash
		_ = i
	}
	if l.tip != prev {
		return errors.New("نوک زنجیره با آخرین رخداد هم‌خوان نیست")
	}
	return nil
}

func hashOf(e Event) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d|%s|%s|%s|%s|%s|%s|%s", e.Seq, e.At, e.Actor, e.Module, e.Action, e.Target, e.Detail, e.PrevHash)
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
