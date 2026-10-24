// ذخیرهٔ فایل‌های بارگذاری‌شده (نام، پسوند، مسیر) — بررسی ریسک آپلود مخرب و پیمایش مسیر.

package recordkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"hrms/internal/contracts"
)

// FileStore انبار رکوردهای گردش‌کاری روی یک فایل JSON — برای اجرای مستقل یک‌رایانه‌ای.
//
// چرا این انبار وجود دارد: در استقرار سازمانی، داده در Postgres می‌نشیند؛ ولی یک
// زیرسامانهٔ مستقل (مثل «سیستم پاداش») باید بدون هیچ پایگاه‌داده و هیچ سرویس
// بیرونی روی یک رایانهٔ شخصی کار کند. اینجا کل داده در یک فایل خوانا ذخیره می‌شود:
// پشتیبان‌گیری یعنی کپی همان یک فایل، و بازرسی داده با باز کردن آن در ویرایشگر ممکن است.
//
// پیاده‌سازی عمداً روی MemoryStore سوار شده تا منطق فهرست/فیلتر/صفحه‌بندی/شناسه‌گذاری
// حتی یک نسخهٔ دوم هم نداشته باشد؛ تنها کاری که اینجا اضافه می‌شود «ماندگاری» است.
type FileStore struct {
	mu   sync.Mutex
	path string
	mem  *MemoryStore
	// saves شمارش نوشتن‌های موفق روی دیسک است؛ برای گزارش وضعیت و آزمون ماندگاری.
	saves int
}

// snapshot شکل ذخیره‌شده روی دیسک. نسخه برای تغییر شکل در آینده نگه داشته می‌شود.
type snapshot struct {
	Version int            `json:"version"`
	Seq     map[string]int `json:"seq"`
	Order   []string       `json:"order"`
	Records []Record       `json:"records"`
}

// snapshotVersion نسخهٔ کنونی شکل فایل.
const snapshotVersion = 1

// NewFileStore انبار را می‌سازد و اگر فایل موجود باشد از آن بازیابی می‌کند. پوشهٔ
// والد در صورت نبود ساخته می‌شود؛ نبودِ فایل خطا نیست (اجرای نخست).
func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, fmt.Errorf("مسیر فایل انبار خالی است")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("ساخت پوشهٔ داده ناموفق: %w", err)
		}
	}
	s := &FileStore{path: path, mem: NewMemoryStore()}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Path مسیر فایل داده؛ برای گزارش راه‌اندازی و پشتیبان‌گیری.
func (s *FileStore) Path() string { return s.path }

// Saves شمار رکوردهای نوشته‌شده روی دیسک از زمان راه‌اندازی.
func (s *FileStore) Saves() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saves
}

// load فایل موجود را می‌خواند؛ نبودِ فایل یعنی انبار خالی و اجرای نخست.
func (s *FileStore) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("خواندن فایل انبار ناموفق: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}
	var snap snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		// فایل خراب نباید اجرای سامانه را بی‌صدا با دادهٔ خالی بالا بیاورد: خطا با نام
		// فایل برگردانده می‌شود تا کاربر بتواند پشتیبان را بازگرداند.
		return fmt.Errorf("فایل انبار «%s» خوانا نیست (پشتیبان را بازگردانید): %w", s.path, err)
	}
	if snap.Seq != nil {
		s.mem.seq = snap.Seq
	}
	s.mem.order = make([]string, 0, len(snap.Records))
	for _, r := range snap.Records {
		if r.ID == "" {
			continue
		}
		if r.Fields == nil {
			r.Fields = map[string]string{}
		}
		s.mem.records[r.ID] = r
		s.mem.order = append(s.mem.order, r.ID)
	}
	return nil
}

/* ---------------------------- قرارداد Repository ---------------------------- */

// Get خواندن یک رکورد.
func (s *FileStore) Get(ctx context.Context, id string) (Record, bool, error) {
	return s.mem.Get(ctx, id)
}

// List فهرست رکوردها با همان فیلترهای انبار مشترک.
func (s *FileStore) List(ctx context.Context, f contracts.RecordFilter) (Page, error) {
	return s.mem.List(ctx, f)
}

// Save درج/به‌روزرسانی و نوشتن فوری روی دیسک.
func (s *FileStore) Save(ctx context.Context, r Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out, err := s.mem.Save(ctx, r)
	if err != nil {
		return Record{}, err
	}
	if err := s.persistLocked(); err != nil {
		return Record{}, err
	}
	return out, nil
}

// Delete حذف رکورد و نوشتن فوری روی دیسک.
func (s *FileStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.mem.Delete(ctx, id); err != nil {
		return err
	}
	return s.persistLocked()
}

// NextID شناسهٔ تازه و ماندگارکردن شمارنده تا شناسه‌ها پس از راه‌اندازی دوباره تکرار نشوند.
func (s *FileStore) NextID(kind, prefix string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := s.mem.NextID(kind, prefix)
	if err != nil {
		return "", err
	}
	if err := s.persistLocked(); err != nil {
		return "", err
	}
	return id, nil
}

// persistLocked کل انبار را در فایل می‌نویسد.
//
// نخست در فایل موقت نوشته و سپس جایگزین می‌شود تا قطع برق میان نوشتن، فایل داده را
// نیمه‌کاره نکند. حجم داده یک زیرسامانهٔ تک‌واحد کوچک است، پس نوشتن کامل در هر تغییر
// ساده‌ترین رفتار قابل توضیح و قابل اتکاست.
func (s *FileStore) persistLocked() error {
	snap := snapshot{
		Version: snapshotVersion,
		Seq:     s.mem.seq,
		Order:   s.mem.order,
		Records: make([]Record, 0, len(s.mem.order)),
	}
	for _, id := range s.mem.order {
		if r, ok := s.mem.records[id]; ok {
			snap.Records = append(snap.Records, r)
		}
	}
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("سریال‌سازی داده ناموفق: %w", err)
	}
	raw = append(raw, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("نوشتن فایل موقت داده ناموفق: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("جایگزینی فایل داده ناموفق: %w", err)
	}
	s.saves++
	return nil
}

// ModTime زمان آخرین تغییر فایل داده؛ برای نمایش در رابط و گزارش پشتیبان‌گیری.
func (s *FileStore) ModTime() time.Time {
	if info, err := os.Stat(s.path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}
