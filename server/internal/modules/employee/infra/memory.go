// Package infra لایهٔ ذخیره‌سازی ماژول کارکنان.
//
// این نسخه یک انبار درون‌حافظه‌ای با همان رفتار رابط پورت است تا برش عمودی بدون
// وابستگی به دیتابیس اجرا شود. مهاجرت‌های Postgres در همین پوشه آماده است و
// پیاده‌سازی بعدی (pgStore) همان قرارداد را برمی‌آورد؛ چیزی در لایه‌های بالا تغییر
// نمی‌کند چون همه از پورت contracts.EmployeeReader استفاده می‌کنند.
package infra

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"hrms/internal/contracts"
	"hrms/internal/modules/employee/domain"
)

// MemoryStore انبار درون‌حافظه‌ای با نسخه‌بندی رکورد.
type MemoryStore struct {
	mu        sync.RWMutex
	employees map[string]domain.Employee
	order     []string
	nextSeq   int
}

// NewMemoryStore انبار خالی می‌سازد.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{employees: make(map[string]domain.Employee)}
}

// NewSeededStore انبار با دادهٔ نمونهٔ سازمان (هم‌خوان با بستهٔ طراحی).
func NewSeededStore() *MemoryStore {
	store := NewMemoryStore()
	for _, e := range SeedEmployees() {
		store.employees[e.ID] = e
		store.order = append(store.order, e.ID)
		store.nextSeq = maxInt(store.nextSeq, parseSeq(e.ID))
	}
	return store
}

// Get خواندن یک کارمند.
func (s *MemoryStore) Get(_ context.Context, id string) (domain.Employee, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.employees[id]
	return e, ok, nil
}

// List خواندن فهرست با فیلتر، جستجو، مرتب‌سازی و صفحه‌بندی.
func (s *MemoryStore) List(_ context.Context, f contracts.EmployeeFilter) (contracts.EmployeePage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]domain.Employee, 0, len(s.employees))
	for _, id := range s.order {
		if e, ok := s.employees[id]; ok {
			all = append(all, e)
		}
	}

	// محدودهٔ دسترسی نقش: کارمند فقط خودش، مدیر واحد فقط واحد خودش.
	filtered := make([]domain.Employee, 0, len(all))
	for _, e := range all {
		switch f.Scope {
		case "own":
			if f.ActorID != "" && e.ID != f.ActorID {
				continue
			}
		case "own_unit":
			if f.ActorDepartment != "" && e.DepartmentID != f.ActorDepartment {
				continue
			}
		}
		if f.DepartmentID != "" && e.DepartmentID != f.DepartmentID {
			continue
		}
		if !contracts.MatchValue(f.Status, f.Statuses, e.Status) {
			continue
		}
		if f.ContractType != "" && e.ContractType != f.ContractType {
			continue
		}
		if !contracts.MatchValue(f.EmploymentType, f.EmploymentTypes, e.EmploymentType) {
			continue
		}
		if f.Query != "" && !matchesQuery(e, f.Query) {
			continue
		}
		filtered = append(filtered, e)
	}

	sortEmployees(filtered, f.Sort)

	total := len(filtered)
	page, size := normalizePaging(f.Page, f.Size)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}

	items := make([]contracts.Employee, 0, end-start)
	for _, e := range filtered[start:end] {
		items = append(items, e.ToContract())
	}
	return contracts.EmployeePage{Items: items, Total: total, Page: page, Size: size}, nil
}

// CountByStatus شمارش کارکنان بر پایهٔ وضعیت (برای کارت‌های داشبورد).
func (s *MemoryStore) CountByStatus(_ context.Context, f contracts.EmployeeFilter) (map[string]int, error) {
	page, err := s.List(context.Background(), contracts.EmployeeFilter{Scope: f.Scope, ActorID: f.ActorID, ActorDepartment: f.ActorDepartment, Page: 1, Size: 10000})
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, e := range page.Items {
		out[e.Status]++
	}
	return out, nil
}

// Save درج یا به‌روزرسانی کارمند؛ نسخه را یک پله جلو می‌برد.
func (s *MemoryStore) Save(_ context.Context, e domain.Employee) (domain.Employee, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.employees[e.ID]; ok {
		e.Version = existing.Version + 1
		e.LeaveBalance = existing.LeaveBalance
	} else {
		e.Version = 1
		s.order = append(s.order, e.ID)
	}

	// شناسه‌های شماره‌دار به‌صورت EMP-### ساخته می‌شوند.
	if _, err := strconv.Atoi(strings.TrimPrefix(e.ID, "EMP-")); err != nil {
		return domain.Employee{}, fmt.Errorf("شناسهٔ کارمند نامعتبر است: %s", e.ID)
	}
	s.employees[e.ID] = e
	return e, nil
}

// Delete حذف کارمند؛ حذف نرم است تا سابقه در گارد آدیت بماند.
func (s *MemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.employees[id]; !ok {
		return ErrNotFound
	}
	delete(s.employees, id)
	kept := s.order[:0]
	for _, x := range s.order {
		if x != id {
			kept = append(kept, x)
		}
	}
	s.order = kept
	return nil
}

// NextID شناسهٔ کارمند تازه.
func (s *MemoryStore) NextID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSeq++
	return fmt.Sprintf("EMP-%03d", s.nextSeq)
}

// ErrNotFound رکورد یافت نشد.
var ErrNotFound = fmt.Errorf("رکورد یافت نشد")

// Departments واحدهای سازمانی (نمودار درختی).
func Departments() []contracts.Department {
	return []contracts.Department{
		{ID: "D1", Name: "فناوری اطلاعات", Head: "آرش رضایی", Headcount: 14},
		{ID: "D6", Name: "تحقیق و توسعه", Head: "آرش رضایی", Headcount: 6, ParentID: "D1"},
		{ID: "D2", Name: "منابع انسانی", Head: "سارا محمدی", Headcount: 5},
		{ID: "D3", Name: "مالی و حسابداری", Head: "نرگس کریمی", Headcount: 8},
		{ID: "D4", Name: "فروش و بازاریابی", Head: "محمد احمدی", Headcount: 11},
		{ID: "D5", Name: "تولید و عملیات", Head: "حسین موسوی", Headcount: 22},
	}
}

// DepartmentName نام واحد از شناسه.
func DepartmentName(id string) string {
	for _, d := range Departments() {
		if d.ID == id {
			return d.Name
		}
	}
	return "—"
}

func matchesQuery(e domain.Employee, q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	return strings.Contains(strings.ToLower(e.FullName), q) ||
		strings.Contains(strings.ToLower(e.ID), q) ||
		strings.Contains(e.NationalID, q) ||
		strings.Contains(strings.ToLower(e.Position), q) ||
		strings.Contains(strings.ToLower(e.DepartmentName), q)
}

func sortEmployees(list []domain.Employee, spec string) {
	field, desc := "id", false
	if spec != "" {
		parts := strings.SplitN(spec, ":", 2)
		field = parts[0]
		if len(parts) == 2 && strings.EqualFold(parts[1], "desc") {
			desc = true
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		var less bool
		switch field {
		case "fullName":
			less = a.FullName < b.FullName
		case "hireDate":
			less = a.HireDate < b.HireDate
		case "baseSalary":
			less = a.BaseSalary < b.BaseSalary
		case "departmentName":
			less = a.DepartmentName < b.DepartmentName
		case "status":
			less = a.Status < b.Status
		default:
			less = a.ID < b.ID
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

func parseSeq(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "EMP-"))
	if err != nil {
		return 0
	}
	return n
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SeedEmployees دادهٔ نمونهٔ سازمان — همان ۱۲ کارمند بستهٔ طراحی، با کد ملی معتبر.
func SeedEmployees() []domain.Employee {
	return []domain.Employee{
		{ID: "EMP-001", FullName: "سارا محمدی", NationalID: "0012345679", DepartmentID: "D2", DepartmentName: "منابع انسانی", Position: "مدیر منابع انسانی", ContractType: "تمام‌وقت", Status: "active", HireDate: "1398/04/12", BaseSalary: 52_000_000, LeaveBalance: 12, Phone: "0912-0000001", Email: "sara.m@demo.ir", ManagerName: "مدیرعامل", EmploymentType: "official_oil"},
		{ID: "EMP-002", FullName: "آرش رضایی", NationalID: "0023456787", DepartmentID: "D1", DepartmentName: "فناوری اطلاعات", Position: "مدیر فناوری اطلاعات", ContractType: "تمام‌وقت", Status: "active", HireDate: "1397/09/01", BaseSalary: 58_000_000, LeaveBalance: 7, Phone: "0912-0000002", Email: "arash.r@demo.ir", ManagerName: "مدیرعامل", EmploymentType: "official_oil"},
		{ID: "EMP-003", FullName: "نگار حسینی", NationalID: "0034567895", DepartmentID: "D4", DepartmentName: "فروش و بازاریابی", Position: "کارشناس فروش", ContractType: "تمام‌وقت", Status: "active", HireDate: "1401/11/20", BaseSalary: 34_000_000, LeaveBalance: 15, Phone: "0912-0000003", Email: "negar.h@demo.ir", ManagerName: "محمد احمدی", EmploymentType: "fixed_1662"},
		{ID: "EMP-004", FullName: "علی مرادی", NationalID: "0045678901", DepartmentID: "D3", DepartmentName: "مالی و حسابداری", Position: "کارشناس ارشد مالی", ContractType: "تمام‌وقت", Status: "active", HireDate: "1399/06/15", BaseSalary: 41_000_000, LeaveBalance: 4, Phone: "0912-0000004", Email: "ali.m@demo.ir", ManagerName: "نرگس کریمی", EmploymentType: "official_worker"},
		{ID: "EMP-005", FullName: "محمد احمدی", NationalID: "0056789017", DepartmentID: "D4", DepartmentName: "فروش و بازاریابی", Position: "مدیر فروش", ContractType: "تمام‌وقت", Status: "active", HireDate: "1396/02/10", BaseSalary: 55_000_000, LeaveBalance: 9, Phone: "0912-0000005", Email: "m.ahmadi@demo.ir", ManagerName: "مدیرعامل", EmploymentType: "official_oil"},
		{ID: "EMP-006", FullName: "نرگس کریمی", NationalID: "0067890121", DepartmentID: "D3", DepartmentName: "مالی و حسابداری", Position: "مدیر مالی", ContractType: "تمام‌وقت", Status: "active", HireDate: "1395/08/03", BaseSalary: 56_000_000, LeaveBalance: 10, Phone: "0912-0000006", Email: "n.karimi@demo.ir", ManagerName: "مدیرعامل", EmploymentType: "official_oil"},
		{ID: "EMP-007", FullName: "رضا کاظمی", NationalID: "0078901235", DepartmentID: "D5", DepartmentName: "تولید و عملیات", Position: "سرپرست تولید", ContractType: "تمام‌وقت", Status: "active", HireDate: "1400/03/08", BaseSalary: 38_000_000, LeaveBalance: 2, Phone: "0912-0000007", Email: "r.kazemi@demo.ir", ManagerName: "حسین موسوی", EmploymentType: "official_worker"},
		{ID: "EMP-008", FullName: "مریم نوری", NationalID: "0089012348", DepartmentID: "D2", DepartmentName: "منابع انسانی", Position: "کارشناس منابع انسانی", ContractType: "تمام‌وقت", Status: "leave", HireDate: "1402/05/01", BaseSalary: 31_000_000, LeaveBalance: 6, Phone: "0912-0000008", Email: "m.nouri@demo.ir", ManagerName: "سارا محمدی", EmploymentType: "fixed_holding"},
		{ID: "EMP-009", FullName: "حسین موسوی", NationalID: "0090123451", DepartmentID: "D5", DepartmentName: "تولید و عملیات", Position: "مدیر تولید", ContractType: "تمام‌وقت", Status: "active", HireDate: "1394/10/25", BaseSalary: 54_000_000, LeaveBalance: 11, Phone: "0912-0000009", Email: "h.mousavi@demo.ir", ManagerName: "مدیرعامل", EmploymentType: "official_oil"},
		{ID: "EMP-010", FullName: "فاطمه صادقی", NationalID: "0101234562", DepartmentID: "D1", DepartmentName: "فناوری اطلاعات", Position: "برنامه‌نویس", ContractType: "پیمانکاری", Status: "active", HireDate: "1403/01/15", BaseSalary: 36_000_000, LeaveBalance: 14, Phone: "0912-0000010", Email: "f.sadeghi@demo.ir", ManagerName: "آرش رضایی", EmploymentType: "contractor_project"},
		{ID: "EMP-011", FullName: "امیر قاسمی", NationalID: "0112345670", DepartmentID: "D1", DepartmentName: "فناوری اطلاعات", Position: "کارشناس شبکه", ContractType: "تمام‌وقت", Status: "probation", HireDate: "1404/06/01", BaseSalary: 29_000_000, LeaveBalance: 18, Phone: "0912-0000011", Email: "a.ghasemi@demo.ir", ManagerName: "آرش رضایی", EmploymentType: "long_term"},
		{ID: "EMP-012", FullName: "الهام رستمی", NationalID: "0123456789", DepartmentID: "D4", DepartmentName: "فروش و بازاریابی", Position: "کارشناس بازاریابی", ContractType: "تمام‌وقت", Status: "terminated", HireDate: "1399/01/20", BaseSalary: 33_000_000, LeaveBalance: 0, Phone: "0912-0000012", Email: "e.rostami@demo.ir", ManagerName: "محمد احمدی", EmploymentType: "fixed_1662"},
	}
}
