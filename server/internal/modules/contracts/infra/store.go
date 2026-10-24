// Package infra لایهٔ ذخیره‌سازی ماژول قراردادها.
//
// دو پیاده‌سازی از یک قرارداد وجود دارد: انبار درون‌حافظه‌ای (اجرای توسعه و آزمون) و
// انبار Postgres (استقرار واقعی). انتخاب با متغیر HRMS_DATABASE_URL انجام می‌شود؛ هیچ
// لایهٔ دیگری از این تفاوت خبر ندارد، چون همه از پورت Repository در بستهٔ app استفاده
// می‌کنند. جدول‌های Postgres در migrations همین پوشه‌اند.
package infra

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/contracts/domain"
	"hrms/internal/platform/calendar"
)

// ErrNotFound رکورد درخواستی وجود ندارد.
var ErrNotFound = fmt.Errorf("قرارداد یافت نشد")

// MemoryStore انبار درون‌حافظه‌ای با نسخه‌بندی و ترتیب درج.
type MemoryStore struct {
	mu        sync.RWMutex
	contracts map[string]domain.Contract
	order     []string
	seq       int
	clock     func() time.Time
}

// NewMemoryStore انبار خالی می‌سازد.
func NewMemoryStore(clock func() time.Time) *MemoryStore {
	if clock == nil {
		clock = time.Now
	}
	return &MemoryStore{contracts: map[string]domain.Contract{}, clock: clock}
}

// NewSeededStore انبار با قراردادهای نمونهٔ سازمان؛ تاریخ‌ها نسبت به امروز ساخته
// می‌شوند تا نمونهٔ نمایشی همیشه معنادار بماند.
func NewSeededStore(clock func() time.Time) *MemoryStore {
	s := NewMemoryStore(clock)
	today, err := calendar.FromGregorian(clock())
	if err != nil {
		today = calendar.Date{Year: 1400, Month: 1, Day: 1}
	}
	for _, c := range SeedContracts(today, clock) {
		s.contracts[c.ID] = c
		s.order = append(s.order, c.ID)
		s.seq = max(s.seq, seqOf(c.ID))
	}
	return s
}

// Get خواندن یک قرارداد همراه با سابقهٔ رخدادها.
func (s *MemoryStore) Get(_ context.Context, id string) (domain.Contract, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.contracts[id]
	return c, ok, nil
}

// ActiveForEmployee قراردادهای غیرخاتمه‌یافتهٔ یک کارمند (برای بررسی هم‌پوشانی و
// تصمیم‌های تمدید/تبدیل).
func (s *MemoryStore) ActiveForEmployee(_ context.Context, employeeID string) ([]domain.Contract, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []domain.Contract{}
	for _, id := range s.order {
		c, ok := s.contracts[id]
		if !ok || c.EmployeeID != employeeID {
			continue
		}
		if c.Status == domain.StatusTerminated || c.Status == domain.StatusRejected {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// List فهرست قراردادها با فیلتر، جستجو، مرتب‌سازی و صفحه‌بندی.
func (s *MemoryStore) List(_ context.Context, f contracts.ContractFilter) (contracts.ContractPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today, err := calendar.Parse(f.Today)
	if err != nil {
		today, _ = calendar.FromGregorian(s.clock())
	}

	items := make([]domain.Contract, 0, len(s.contracts))
	for _, id := range s.order {
		c, ok := s.contracts[id]
		if !ok {
			continue
		}
		if !matchScope(c, f) || !matchFilters(c, f, today) {
			continue
		}
		items = append(items, c)
	}

	sortContracts(items, f.Sort)

	total := len(items)
	page, size := normalizePaging(f.Page, f.Size)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := min(start+size, total)

	out := make([]contracts.Contract, 0, end-start)
	for _, c := range items[start:end] {
		out = append(out, c.ToContract(today))
	}
	return contracts.ContractPage{Items: out, Total: total, Page: page, Size: size}, nil
}

// Save درج یا به‌روزرسانی؛ نسخه یک پله جلو می‌رود.
func (s *MemoryStore) Save(_ context.Context, c domain.Contract) (domain.Contract, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.contracts[c.ID]; ok {
		c.Version = existing.Version + 1
	} else {
		c.Version = 1
		s.order = append(s.order, c.ID)
	}
	s.contracts[c.ID] = c
	return c, nil
}

// Delete حذف قرارداد پیش‌نویس (تنها وضعیتی که حذف در آن مجاز است).
func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.contracts[id]; !ok {
		return ErrNotFound
	}
	delete(s.contracts, id)
	kept := s.order[:0]
	for _, x := range s.order {
		if x != id {
			kept = append(kept, x)
		}
	}
	s.order = kept
	return nil
}

// NextID شناسهٔ قرارداد تازه به شکل CT-<سال شمسی>-<شماره>.
func (s *MemoryStore) NextID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("CT-%d-%04d", d.Year, s.seq), nil
}

func matchScope(c domain.Contract, f contracts.ContractFilter) bool {
	switch f.Scope {
	case "own":
		return f.ActorEmployeeID != "" && c.EmployeeID == f.ActorEmployeeID
	case "own_unit":
		return f.ActorDepartment == "" || c.DepartmentName == f.ActorDepartment
	default:
		return true
	}
}

func matchFilters(c domain.Contract, f contracts.ContractFilter, today calendar.Date) bool {
	if f.EmployeeID != "" && c.EmployeeID != f.EmployeeID {
		return false
	}
	if f.Kind != "" && c.Kind != f.Kind {
		return false
	}
	if f.Status != "" && c.Status != f.Status {
		return false
	}
	if f.EmploymentType != "" && c.EmploymentType != f.EmploymentType {
		return false
	}
	if f.ExpiringDays > 0 {
		if c.Kind != domain.KindFixedTerm || c.Status != domain.StatusActive || c.EndDate == "" {
			return false
		}
		end, err := calendar.Parse(c.EndDate)
		if err != nil {
			return false
		}
		days, err := calendar.DaysBetween(today, end)
		if err != nil || days < 0 || days > f.ExpiringDays {
			return false
		}
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		q = strings.ToLower(q)
		hit := strings.Contains(strings.ToLower(c.EmployeeName), q) ||
			strings.Contains(strings.ToLower(c.ID), q) ||
			strings.Contains(strings.ToLower(c.DepartmentName), q) ||
			strings.Contains(strings.ToLower(c.Position), q)
		if !hit {
			return false
		}
	}
	return true
}

func sortContracts(list []domain.Contract, spec string) {
	field, desc := "startDate", true
	if spec != "" {
		parts := strings.SplitN(spec, ":", 2)
		field = parts[0]
		desc = len(parts) == 2 && strings.EqualFold(parts[1], "desc")
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		var less bool
		switch field {
		case "employeeName":
			less = a.EmployeeName < b.EmployeeName
		case "endDate":
			less = a.EndDate < b.EndDate
		case "monthlySalary":
			less = a.MonthlySalary < b.MonthlySalary
		case "status":
			less = a.Status < b.Status
		default:
			less = a.StartDate < b.StartDate
		}
		if desc {
			return !less
		}
		return less
	})
}

func normalizePaging(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return page, size
}

func seqOf(id string) int {
	parts := strings.Split(id, "-")
	if len(parts) == 0 {
		return 0
	}
	var n int
	fmt.Sscanf(parts[len(parts)-1], "%d", &n)
	return n
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
