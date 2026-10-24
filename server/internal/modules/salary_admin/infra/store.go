// Package infra لایهٔ ذخیره‌سازی ماژول احکام حقوق و گواهی اشتغال.
//
// دو پیاده‌سازی از یک قرارداد: انبار درون‌حافظه‌ای (توسعه و آزمون) و انبار Postgres
// (استقرار واقعی). انتخاب با HRMS_DATABASE_URL انجام می‌شود.
package infra

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/salary_admin/domain"
	"hrms/internal/platform/calendar"
)

// ErrDecreeNotFound حکم حقوق درخواستی وجود ندارد.
var ErrDecreeNotFound = fmt.Errorf("حکم حقوق یافت نشد")

// ErrCertificateNotFound گواهی درخواستی وجود ندارد.
var ErrCertificateNotFound = fmt.Errorf("گواهی اشتغال یافت نشد")

// MemoryStore انبار درون‌حافظه‌ای احکام و گواهی‌ها.
type MemoryStore struct {
	mu      sync.RWMutex
	decrees map[string]domain.SalaryDecree
	dOrder  []string
	certs   map[string]domain.EmploymentCertificate
	cOrder  []string
	dSeq    int
	cSeq    int
	clock   func() time.Time
}

// NewMemoryStore انبار خالی می‌سازد.
func NewMemoryStore(clock func() time.Time) *MemoryStore {
	if clock == nil {
		clock = time.Now
	}
	return &MemoryStore{
		decrees: map[string]domain.SalaryDecree{},
		certs:   map[string]domain.EmploymentCertificate{},
		clock:   clock,
	}
}

// NewSeededStore انبار با دادهٔ نمونهٔ سازمان.
func NewSeededStore(clock func() time.Time) *MemoryStore {
	s := NewMemoryStore(clock)
	today, err := calendar.FromGregorian(clock())
	if err != nil {
		today = calendar.Date{Year: 1400, Month: 1, Day: 1}
	}
	for _, d := range SeedDecrees(today) {
		s.decrees[d.ID] = d
		s.dOrder = append(s.dOrder, d.ID)
		s.dSeq = maxInt(s.dSeq, seqOf(d.ID))
	}
	for _, c := range SeedCertificates(today) {
		s.certs[c.ID] = c
		s.cOrder = append(s.cOrder, c.ID)
		s.cSeq = maxInt(s.cSeq, seqOf(c.ID))
	}
	return s
}

/* ------------------------------- حکم حقوق ------------------------------- */

// GetDecree خواندن یک حکم.
func (s *MemoryStore) GetDecree(_ context.Context, id string) (domain.SalaryDecree, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.decrees[id]
	return d, ok, nil
}

// NotifiedForEmployee حکم‌های ابلاغ‌شدهٔ کارمند (برای اصلاح و منسوخ‌سازی).
func (s *MemoryStore) NotifiedForEmployee(_ context.Context, employeeID string) ([]domain.SalaryDecree, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []domain.SalaryDecree{}
	for _, id := range s.dOrder {
		d, ok := s.decrees[id]
		if ok && d.EmployeeID == employeeID && d.Status == domain.DecreeNotified {
			out = append(out, d)
		}
	}
	return out, nil
}

// ListDecrees فهرست احکام با فیلتر و محدودهٔ دسترسی نقش.
func (s *MemoryStore) ListDecrees(_ context.Context, f contracts.SalaryDecreeFilter) (contracts.SalaryDecreePage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.SalaryDecree{}
	for _, id := range s.dOrder {
		d, ok := s.decrees[id]
		if !ok || !matchDecree(d, f) {
			continue
		}
		items = append(items, d)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID > items[j].ID })
	total := len(items)
	page, size := normalizePaging(f.Page, f.Size)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := minInt(start+size, total)
	out := make([]contracts.SalaryDecree, 0, end-start)
	for _, d := range items[start:end] {
		out = append(out, d.ToSalaryDecree())
	}
	return contracts.SalaryDecreePage{Items: out, Total: total, Page: page, Size: size}, nil
}

// SaveDecree درج یا به‌روزرسانی حکم.
func (s *MemoryStore) SaveDecree(_ context.Context, d domain.SalaryDecree) (domain.SalaryDecree, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.decrees[d.ID]; ok {
		d.Version = existing.Version + 1
	} else {
		d.Version = 1
		s.dOrder = append(s.dOrder, d.ID)
	}
	s.decrees[d.ID] = d
	return d, nil
}

// DeleteDecree حذف حکم پیش‌نویس.
func (s *MemoryStore) DeleteDecree(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.decrees[id]; !ok {
		return ErrDecreeNotFound
	}
	delete(s.decrees, id)
	s.dOrder = removeID(s.dOrder, id)
	return nil
}

// NextDecreeID شناسهٔ حکم تازه به شکل DEC-<سال شمسی>-<شماره>.
func (s *MemoryStore) NextDecreeID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dSeq++
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("DEC-%d-%04d", d.Year, s.dSeq), nil
}

/* ------------------------------ گواهی اشتغال ------------------------------ */

// GetCertificate خواندن یک گواهی.
func (s *MemoryStore) GetCertificate(_ context.Context, id string) (domain.EmploymentCertificate, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.certs[id]
	return c, ok, nil
}

// ListCertificates فهرست گواهی‌ها با فیلتر و محدودهٔ دسترسی نقش.
func (s *MemoryStore) ListCertificates(_ context.Context, f contracts.CertificateFilter) (contracts.CertificatePage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.EmploymentCertificate{}
	for _, id := range s.cOrder {
		c, ok := s.certs[id]
		if !ok || !matchCertificate(c, f) {
			continue
		}
		items = append(items, c)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID > items[j].ID })
	total := len(items)
	page, size := normalizePaging(f.Page, f.Size)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := minInt(start+size, total)
	out := make([]contracts.EmploymentCertificate, 0, end-start)
	for _, c := range items[start:end] {
		out = append(out, c.ToCertificate())
	}
	return contracts.CertificatePage{Items: out, Total: total, Page: page, Size: size}, nil
}

// SaveCertificate درج یا به‌روزرسانی گواهی.
func (s *MemoryStore) SaveCertificate(_ context.Context, c domain.EmploymentCertificate) (domain.EmploymentCertificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.certs[c.ID]; ok {
		c.Version = existing.Version + 1
	} else {
		c.Version = 1
		s.cOrder = append(s.cOrder, c.ID)
	}
	s.certs[c.ID] = c
	return c, nil
}

// DeleteCertificate حذف گواهی در وضعیت درخواست.
func (s *MemoryStore) DeleteCertificate(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.certs[id]; !ok {
		return ErrCertificateNotFound
	}
	delete(s.certs, id)
	s.cOrder = removeID(s.cOrder, id)
	return nil
}

// NextCertificateID شناسه و سریال گواهی تازه.
func (s *MemoryStore) NextCertificateID() (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cSeq++
	d, err := calendar.FromGregorian(s.clock())
	if err != nil {
		return "", "", err
	}
	seq := fmt.Sprintf("%04d", s.cSeq)
	return "CRT-" + seq, fmt.Sprintf("CRT-%d-%s", d.Year, seq), nil
}

/* -------------------------------- کمکی‌ها -------------------------------- */

func matchDecree(d domain.SalaryDecree, f contracts.SalaryDecreeFilter) bool {
	if !matchScope(d.EmployeeID, d.DepartmentName, f.Scope, f.ActorEmployeeID, f.ActorDepartment) {
		return false
	}
	if f.EmployeeID != "" && d.EmployeeID != f.EmployeeID {
		return false
	}
	if f.Status != "" && d.Status != f.Status {
		return false
	}
	if f.JobGroup != "" && d.JobGroup != f.JobGroup {
		return false
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		q = strings.ToLower(q)
		hit := strings.Contains(strings.ToLower(d.EmployeeName), q) ||
			strings.Contains(strings.ToLower(d.ID), q) ||
			strings.Contains(strings.ToLower(d.JobGroup), q)
		if !hit {
			return false
		}
	}
	return true
}

func matchCertificate(c domain.EmploymentCertificate, f contracts.CertificateFilter) bool {
	if !matchScope(c.EmployeeID, c.DepartmentName, f.Scope, f.ActorEmployeeID, f.ActorDepartment) {
		return false
	}
	if f.EmployeeID != "" && c.EmployeeID != f.EmployeeID {
		return false
	}
	if f.Status != "" && c.Status != f.Status {
		return false
	}
	if f.Kind != "" && c.Kind != f.Kind {
		return false
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		q = strings.ToLower(q)
		hit := strings.Contains(strings.ToLower(c.EmployeeName), q) ||
			strings.Contains(strings.ToLower(c.Serial), q) ||
			strings.Contains(strings.ToLower(c.ID), q)
		if !hit {
			return false
		}
	}
	return true
}

func matchScope(employeeID, department, scope, actorID, actorDepartment string) bool {
	switch scope {
	case "own":
		return actorID != "" && employeeID == actorID
	case "own_unit":
		return actorDepartment == "" || department == actorDepartment
	default:
		return true
	}
}

func removeID(list []string, id string) []string {
	out := list[:0]
	for _, x := range list {
		if x != id {
			out = append(out, x)
		}
	}
	return out
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
	var n int
	fmt.Sscanf(parts[len(parts)-1], "%d", &n)
	return n
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
