// Package infra لایهٔ ذخیره‌سازی ماژول تکفل و درخواست پرواز.
//
// دو انبار از یک قرارداد: حافظه (توسعه و آزمون) و Postgres (استقرار واقعی). انتخاب با
// وجود اتصال پایگاه‌داده در وابستگی‌های ماژول انجام می‌شود.
package infra

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/staff_services/domain"
	"hrms/internal/platform/calendar"
)

// ErrNotFound رکورد درخواستی وجود ندارد.
var ErrNotFound = fmt.Errorf("رکورد یافت نشد")

// MemoryStore انبار درون‌حافظه‌ای هر دو موجودیت ماژول.
type MemoryStore struct {
	mu          sync.RWMutex
	dependents  map[string]domain.Dependent
	order       []string
	flights     map[string]domain.FlightRequest
	flightOrder []string
	depSeq      int
	flightSeq   int
	clock       func() time.Time
}

// NewMemoryStore انبار خالی می‌سازد.
func NewMemoryStore(clock func() time.Time) *MemoryStore {
	if clock == nil {
		clock = time.Now
	}
	return &MemoryStore{
		dependents: map[string]domain.Dependent{},
		flights:    map[string]domain.FlightRequest{},
		clock:      clock,
	}
}

// NewSeededStore انبار با دادهٔ نمونه؛ تاریخ‌ها نسبت به امروز ساخته می‌شوند تا صفحه‌های
// تکفل و درخواست پرواز بعد از چند ماه هم معنادار بمانند.
func NewSeededStore(clock func() time.Time) *MemoryStore {
	s := NewMemoryStore(clock)
	today, err := calendar.FromGregorian(clock())
	if err != nil {
		today = calendar.Date{Year: 1400, Month: 1, Day: 1}
	}
	for _, d := range SeedDependents(today) {
		s.dependents[d.ID] = d
		s.order = append(s.order, d.ID)
	}
	for _, f := range SeedFlights(today) {
		s.flights[f.ID] = f
		s.flightOrder = append(s.flightOrder, f.ID)
	}
	s.depSeq = 100 + len(s.order)
	s.flightSeq = 200 + len(s.flightOrder)
	return s
}

// ---------- تکفل ----------

// GetDependent خواندن یک تکفل.
func (s *MemoryStore) GetDependent(_ context.Context, id string) (domain.Dependent, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.dependents[id]
	return d, ok, nil
}

// ListDependents فهرست تکفل‌شدگان با فیلتر و محدودهٔ دسترسی نقش.
func (s *MemoryStore) ListDependents(_ context.Context, f contracts.DependentFilter) (contracts.DependentPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	today := s.today()

	items := []domain.Dependent{}
	for _, id := range s.order {
		d, ok := s.dependents[id]
		if !ok || !matchDependentScope(d.EmployeeID, d.DepartmentName, f.Scope, f.ActorEmployeeID, f.ActorDepartment) {
			continue
		}
		if f.EmployeeID != "" && d.EmployeeID != f.EmployeeID {
			continue
		}
		if f.Relation != "" && d.Relation != f.Relation {
			continue
		}
		if f.Status != "" && d.Status != f.Status {
			continue
		}
		if q := strings.ToLower(strings.TrimSpace(f.Query)); q != "" &&
			!strings.Contains(strings.ToLower(d.FullName), q) &&
			!strings.Contains(strings.ToLower(d.EmployeeName), q) &&
			!strings.Contains(d.NationalID, q) {
			continue
		}
		items = append(items, d)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return dependentPage(items, f.Page, f.Size, today), nil
}

// SaveDependent درج/به‌روزرسانی تکفل.
func (s *MemoryStore) SaveDependent(_ context.Context, d domain.Dependent) (domain.Dependent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.dependents[d.ID]; ok {
		d.Version = existing.Version + 1
	} else {
		d.Version = 1
		s.order = append(s.order, d.ID)
	}
	s.dependents[d.ID] = d
	return d, nil
}

// DeleteDependent حذف تکفل ثبت‌شدهٔ ردشده (پاک‌سازی ورودی نادرست).
func (s *MemoryStore) DeleteDependent(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dependents[id]; !ok {
		return ErrNotFound
	}
	delete(s.dependents, id)
	kept := s.order[:0]
	for _, x := range s.order {
		if x != id {
			kept = append(kept, x)
		}
	}
	s.order = kept
	return nil
}

// NextDependentID شناسهٔ تکفل تازه.
func (s *MemoryStore) NextDependentID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.depSeq++
	return fmt.Sprintf("DEP-%04d", s.depSeq), nil
}

// ---------- درخواست پرواز ----------

// GetFlight خواندن یک درخواست پرواز.
func (s *MemoryStore) GetFlight(_ context.Context, id string) (domain.FlightRequest, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flights[id]
	return f, ok, nil
}

// ListFlights فهرست درخواست‌های پرواز با فیلتر و محدودهٔ دسترسی نقش.
func (s *MemoryStore) ListFlights(_ context.Context, filt contracts.FlightRequestFilter) (contracts.FlightRequestPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := []domain.FlightRequest{}
	for _, id := range s.flightOrder {
		f, ok := s.flights[id]
		if !ok || !matchDependentScope(f.EmployeeID, f.DepartmentName, filt.Scope, filt.ActorEmployeeID, filt.ActorDepartment) {
			continue
		}
		if filt.EmployeeID != "" && f.EmployeeID != filt.EmployeeID {
			continue
		}
		if filt.Status != "" && f.Status != filt.Status {
			continue
		}
		if filt.Purpose != "" && f.Purpose != filt.Purpose {
			continue
		}
		if q := strings.ToLower(strings.TrimSpace(filt.Query)); q != "" &&
			!strings.Contains(strings.ToLower(f.EmployeeName), q) &&
			!strings.Contains(strings.ToLower(f.OriginFa), q) &&
			!strings.Contains(strings.ToLower(f.DestinationFa), q) &&
			!strings.Contains(strings.ToLower(f.ID), q) {
			continue
		}
		items = append(items, f)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].DepartDate > items[j].DepartDate })

	total := len(items)
	page, size := normalizePaging(filt.Page, filt.Size)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := min(start+size, total)
	out := make([]contracts.FlightRequest, 0, end-start)
	for _, f := range items[start:end] {
		out = append(out, f.ToContract())
	}
	return contracts.FlightRequestPage{Items: out, Total: total, Page: page, Size: size}, nil
}

// SaveFlight درج/به‌روزرسانی درخواست پرواز.
func (s *MemoryStore) SaveFlight(_ context.Context, f domain.FlightRequest) (domain.FlightRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.flights[f.ID]; ok {
		f.Version = existing.Version + 1
	} else {
		f.Version = 1
		s.flightOrder = append(s.flightOrder, f.ID)
	}
	s.flights[f.ID] = f
	return f, nil
}

// NextFlightID شناسهٔ درخواست پرواز تازه به شکل FL-<سال شمسی>-<شماره>.
func (s *MemoryStore) NextFlightID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flightSeq++
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("FL-%d-%04d", d.Year, s.flightSeq), nil
}

func (s *MemoryStore) today() calendar.Date {
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return calendar.Date{Year: 1400, Month: 1, Day: 1}
	}
	return d
}

func dependentPage(items []domain.Dependent, page, size int, today calendar.Date) contracts.DependentPage {
	total := len(items)
	page, size = normalizePaging(page, size)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := min(start+size, total)
	out := make([]contracts.Dependent, 0, end-start)
	for _, d := range items[start:end] {
		out = append(out, d.ToContract(today))
	}
	return contracts.DependentPage{Items: out, Total: total, Page: page, Size: size}
}

// matchDependentScope محدودهٔ دادهٔ نقش را روی رکورد اعمال می‌کند.
func matchDependentScope(employeeID, departmentName, scope, actorID, actorDepartment string) bool {
	switch scope {
	case "own":
		return actorID != "" && employeeID == actorID
	case "own_unit":
		return actorDepartment == "" || departmentName == actorDepartment
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
	if size > 200 {
		size = 200
	}
	return page, size
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
