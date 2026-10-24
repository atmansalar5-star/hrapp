// هستهٔ شمای رکوردکیت: نوع فیلدها و اعتبارسنجی سمت سرور.

package recordkit

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hrms/internal/contracts"
)

// ErrNotFound رکورد درخواستی وجود ندارد.
var ErrNotFound = fmt.Errorf("رکورد یافت نشد")

// Record رکورد گردش‌کاری در حافظهٔ سامانه (شکل ذخیره‌سازی، جدا از DTO).
type Record struct {
	ID             string
	Module         string
	Kind           string
	Title          string
	EmployeeID     string
	EmployeeName   string
	DepartmentName string
	Status         string
	Fields         map[string]string
	NoteFa         string
	CreatedBy      string
	DecidedBy      string
	Events         []Event
	Version        int
	UpdatedAt      time.Time
}

// Event رخداد گردش‌کار.
type Event struct {
	At     time.Time
	Actor  string
	Action string
	NoteFa string
}

// Page یک صفحهٔ رکوردهای دامنه (پیش از تبدیل به DTO).
type Page struct {
	Items []Record
	Total int
	Page  int
	Size  int
}

// Repository قرارداد ذخیره‌سازی مشترک (حافظه یا Postgres).
type Repository interface {
	Get(ctx context.Context, id string) (Record, bool, error)
	List(ctx context.Context, f contracts.RecordFilter) (Page, error)
	Save(ctx context.Context, r Record) (Record, error)
	Delete(ctx context.Context, id string) error
	NextID(kind, prefix string) (string, error)
}

// ToDTO تبدیل به شکل انتقالی API؛ برچسب فارسی وضعیت از شناسهٔ نوع می‌آید تا این لایه
// هیچ متن فارسی هاردکدی نداشته باشد.
func (r Record) ToDTO(k Kind) contracts.Record {
	events := make([]contracts.RecordEvent, 0, len(r.Events))
	for _, e := range r.Events {
		events = append(events, contracts.RecordEvent{
			At:       e.At.UTC().Format(time.RFC3339),
			Actor:    e.Actor,
			Action:   e.Action,
			ActionFa: actionFa(k, e.Action),
			NoteFa:   e.NoteFa,
		})
	}
	fields := map[string]string{}
	for key, v := range r.Fields {
		fields[key] = v
	}
	return contracts.Record{
		ID:             r.ID,
		Kind:           r.Kind,
		KindFa:         k.TitleFa,
		Title:          r.Title,
		EmployeeID:     r.EmployeeID,
		EmployeeName:   r.EmployeeName,
		DepartmentName: r.DepartmentName,
		Status:         r.Status,
		StatusFa:       k.StatusFa(r.Status),
		Fields:         fields,
		NoteFa:         r.NoteFa,
		CreatedBy:      r.CreatedBy,
		DecidedBy:      r.DecidedBy,
		Events:         events,
		Version:        r.Version,
	}
}

func actionFa(k Kind, action string) string {
	if a, ok := k.ActionByValue(action); ok {
		return a.LabelFa
	}
	return action
}

/* --------------------------- انبار درون‌حافظه --------------------------- */

// MemoryStore انبار درون‌حافظه‌ای رکوردها (اجرای توسعه و آزمون).
type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
	order   []string
	seq     map[string]int
}

// NewMemoryStore انبار خالی می‌سازد.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: map[string]Record{}, seq: map[string]int{}}
}

// Get خواندن یک رکورد.
func (s *MemoryStore) Get(_ context.Context, id string) (Record, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	return r, ok, nil
}

// List فهرست رکوردها با فیلتر، جستجو، وضعیت و محدودهٔ دسترسی نقش.
func (s *MemoryStore) List(_ context.Context, f contracts.RecordFilter) (Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Record, 0, len(s.records))
	for _, id := range s.order {
		r, ok := s.records[id]
		if !ok || !matchRecord(r, f) {
			continue
		}
		items = append(items, r)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID > items[j].ID })

	total := len(items)
	page, size := normalizePaging(f.Page, f.Size)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := minInt(start+size, total)
	out := make([]Record, 0, end-start)
	out = append(out, items[start:end]...)
	return Page{Items: out, Total: total, Page: page, Size: size}, nil
}

// Save درج یا به‌روزرسانی؛ نسخه یک پله جلو می‌رود.
func (s *MemoryStore) Save(_ context.Context, r Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.records[r.ID]; ok {
		r.Version = existing.Version + 1
	} else {
		r.Version = 1
		s.order = append(s.order, r.ID)
	}
	s.records[r.ID] = r
	return r, nil
}

// Delete حذف رکورد.
func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[id]; !ok {
		return ErrNotFound
	}
	delete(s.records, id)
	kept := s.order[:0]
	for _, x := range s.order {
		if x != id {
			kept = append(kept, x)
		}
	}
	s.order = kept
	return nil
}

// NextID شناسهٔ تازه به شکل <پیشوند>-<شماره> با شمارندهٔ مستقل هر نوع رکورد.
func (s *MemoryStore) NextID(kind, prefix string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seq := s.seq[kind]
	if seq == 0 {
		// نخستین شناسه پس از بزرگ‌ترین شمارهٔ موجود همان نوع ادامه می‌یابد.
		for id := range s.records {
			if !strings.HasPrefix(id, prefix+"-") {
				continue
			}
			if n := seqOf(id); n > seq {
				seq = n
			}
		}
	}
	seq++
	s.seq[kind] = seq
	return fmt.Sprintf("%s-%04d", prefix, seq), nil
}

func matchRecord(r Record, f contracts.RecordFilter) bool {
	if f.Module != "" && r.Module != f.Module {
		return false
	}
	if f.Kind != "" && r.Kind != f.Kind {
		return false
	}
	if !matchScope(r, f) {
		return false
	}
	if f.EmployeeID != "" && r.EmployeeID != f.EmployeeID {
		return false
	}
	if f.Status != "" && r.Status != f.Status {
		return false
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		q = strings.ToLower(q)
		hit := strings.Contains(strings.ToLower(r.Title), q) ||
			strings.Contains(strings.ToLower(r.ID), q) ||
			strings.Contains(strings.ToLower(r.EmployeeName), q) ||
			strings.Contains(strings.ToLower(r.DepartmentName), q)
		if !hit {
			for _, v := range r.Fields {
				if strings.Contains(strings.ToLower(v), q) {
					hit = true
					break
				}
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

func matchScope(r Record, f contracts.RecordFilter) bool {
	switch f.Scope {
	case "own":
		return f.ActorEmployeeID != "" && r.EmployeeID == f.ActorEmployeeID
	case "own_unit":
		return f.ActorDepartment == "" || r.DepartmentName == f.ActorDepartment
	default:
		return true
	}
}

func normalizePaging(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 500 {
		size = 500
	}
	return page, size
}

func seqOf(id string) int {
	idx := strings.LastIndex(id, "-")
	if idx < 0 {
		return 0
	}
	var n int
	fmt.Sscanf(id[idx+1:], "%d", &n)
	return n
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
