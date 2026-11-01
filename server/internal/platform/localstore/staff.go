// زیرساخت «localstore» در پلتفرم سرور — «staff».

package localstore

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"hrms/internal/contracts"
)

// StaffStore فهرست کارکنان را از یک فایل CSV می‌خواند و پورت کارکنان
// (contracts.EmployeeReader) را پیاده می‌کند.
//
// چرا CSV: کارشناس پاداش فهرست کارکنان را از سامانهٔ کارگزینی می‌گیرد و همان فایل را با
// Excel ویرایش می‌کند. ستون‌ها فارسی‌اند و سطر سرستون تشخیص داده می‌شود؛ پس افزودن یک
// نفر یعنی یک سطر تازه، و نیازی به هیچ ابزار دیگری نیست. فایل پس از تغییر روی دیسک
// خودکار دوباره بارگذاری می‌شود، بنابراین ویرایش فایل نیازی به راه‌اندازی دوباره ندارد.
type StaffStore struct {
	path string

	mu      sync.RWMutex
	modTime time.Time
	items   []contracts.Employee
	index   map[string]int
	warn    string
}

// staffSample نمونهٔ اجرای نخست.
//
// چرا همین کدها: نمونهٔ پرداخت‌های پاداش (bonus/domain/seed.go) به همین کدهای پرسنلی
// اشاره می‌کند و «واحد» هم همان مدیریت‌های پایهٔ سازمان است. پس اجرای نخست یک نمایش
// کامل و هم‌خوان نشان می‌دهد و کاربر فایل را فقط با فهرست واقعی خودش جانشین می‌کند.
const staffSample = "\uFEFF" + `کد پرسنلی,نام و نام خانوادگی,واحد,سمت,وضعیت
EMP-001,سارا محمدی,معاونت پشتیبانی خدمات,کارشناس ارشد,فعال
EMP-002,آرش رضایی,بندر امام,سرپرست عملیات,فعال
EMP-003,نرگس کاظمی,بسپاران,اپراتور,فعال
EMP-004,علی احمدی,بندر امام,کارشناس تولید,فعال
EMP-005,محمد احمدی,فراورش یک,کارشناس,فعال
EMP-006,رضا موسوی,فراورش دو,کارشناس ارشد,فعال
EMP-007,حسین کریمی,مدیریت پشتیبانی تعمیرات,تکنسین ارشد,فعال
EMP-008,مریم نوری,معاونت پشتیبانی خدمات,کارشناس,فعال
EMP-009,زهرا موسوی,کیمیا,کارشناس پروژه,فعال
EMP-010,فاطمه صادقی,آب و نیرو,کارشناس,فعال
EMP-011,امیر تهرانی,فراورش دو,کارشناس,فعال
EMP-012,لیلا رستمی,بسپاران,کارشناس نگهداشت,فعال
`

// NewStaffStore انبار کارکنان را می‌سازد و در اجرای نخست نمونهٔ نمونه را می‌کارد تا
// فرم تخصیص از همان ابتدا قابل استفاده باشد.
func NewStaffStore(path string) (*StaffStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("مسیر فایل کارکنان خالی است")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("ساخت پوشهٔ داده ناموفق: %w", err)
		}
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(staffSample), 0o644); err != nil {
			return nil, fmt.Errorf("ساخت فایل نمونهٔ کارکنان ناموفق: %w", err)
		}
	}
	s := &StaffStore{path: path}
	if err := s.reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// Path مسیر فایل کارکنان؛ برای گزارش راه‌اندازی و رابط.
func (s *StaffStore) Path() string { return s.path }

// Warning آخرین هشدار خواندن فایل (مثلاً سطر ناقص) برای نمایش در رابط.
func (s *StaffStore) Warning() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.warn
}

// Count شمار کارکنان بارگذاری‌شده.
func (s *StaffStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

/* ----------------------------- پورت کارکنان ----------------------------- */

// List فهرست کارکنان با فیلتر جستجو/واحد/وضعیت و صفحه‌بندی.
//
// محدودهٔ دسترسی (Scope) در اجرای مستقل تک‌واحدی اعمال نمی‌شود: کاربران این استقرار
// کارشناس پاداش‌اند و همهٔ فهرست را می‌بینند؛ اگر روزی چند نقش اضافه شد، همین‌جا و در
// دامنه اعمال می‌شود، نه در رابط.
func (s *StaffStore) List(_ context.Context, f contracts.EmployeeFilter) (contracts.EmployeePage, error) {
	if err := s.refreshIfChanged(); err != nil {
		return contracts.EmployeePage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := strings.ToLower(strings.TrimSpace(f.Query))
	dept := strings.TrimSpace(f.DepartmentID)
	status := strings.TrimSpace(f.Status)

	matched := make([]contracts.Employee, 0, len(s.items))
	for _, e := range s.items {
		// فیلتر تک‌مقداری یا چندانتخابی (تیک‌خورده) — همان قاعدهٔ مشترک contracts.MatchValue.
		if !contracts.MatchValue(status, f.Statuses, e.Status) {
			continue
		}
		if !contracts.MatchValue(f.EmploymentType, f.EmploymentTypes, e.EmploymentType) {
			continue
		}
		if dept != "" && e.DepartmentID != dept && e.DepartmentName != dept {
			continue
		}
		if query != "" && !staffMatch(e, query) {
			continue
		}
		matched = append(matched, e)
	}
	sort.SliceStable(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })

	total := len(matched)
	page, size := paging(f.Page, f.Size)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	items := make([]contracts.Employee, 0, end-start)
	items = append(items, matched[start:end]...)
	return contracts.EmployeePage{Items: items, Total: total, Page: page, Size: size}, nil
}

// Get خواندن یک کارمند.
func (s *StaffStore) Get(ctx context.Context, f contracts.EmployeeFilter, id string) (contracts.Employee, bool, error) {
	page, err := s.List(ctx, contracts.EmployeeFilter{Scope: f.Scope, Page: 1, Size: 100000})
	if err != nil {
		return contracts.Employee{}, false, err
	}
	for _, e := range page.Items {
		if e.ID == strings.TrimSpace(id) {
			return e, true, nil
		}
	}
	return contracts.Employee{}, false, nil
}

// CountByStatus شمار کارکنان هر وضعیت.
func (s *StaffStore) CountByStatus(ctx context.Context, f contracts.EmployeeFilter) (map[string]int, error) {
	page, err := s.List(ctx, contracts.EmployeeFilter{Scope: f.Scope, Page: 1, Size: 100000})
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, e := range page.Items {
		out[e.Status]++
	}
	return out, nil
}

// Departments واحدهای سازمانی از همین فایل کارکنان ساخته می‌شود.
//
// چرا از خود فایل و نه فهرست دستی: در اجرای مستقل یک‌رایانه‌ای، دادهٔ کارکنان در CSV است
// و واحدها هم همان ستون «واحد سازمانی» هر سطر است؛ پس درخت واحدها با دادهٔ واقعی هم‌خوان
// می‌ماند و فهرست دوم و ناسازگاری ساخته نمی‌شود.
func (s *StaffStore) Departments(_ context.Context) ([]contracts.Department, error) {
	if err := s.refreshIfChanged(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := map[string]int{}
	heads := map[string]string{}
	order := []string{}
	for _, e := range s.items {
		name := strings.TrimSpace(e.DepartmentName)
		if name == "" {
			name = "بدون واحد"
		}
		if _, seen := counts[name]; !seen {
			order = append(order, name)
		}
		counts[name]++
		if heads[name] == "" && strings.TrimSpace(e.ManagerName) != "" {
			heads[name] = strings.TrimSpace(e.ManagerName)
		}
	}
	sort.Strings(order)
	out := make([]contracts.Department, 0, len(order))
	for i, name := range order {
		out = append(out, contracts.Department{
			ID:        fmt.Sprintf("D%d", i+1),
			Name:      name,
			Head:      heads[name],
			Headcount: counts[name],
		})
	}
	return out, nil
}

/* ------------------------------ خواندن فایل ------------------------------ */

// refreshIfChanged اگر فایل روی دیسک تازه‌تر شده باشد، دوباره بارگذاری می‌کند.
func (s *StaffStore) refreshIfChanged() error {
	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("خواندن وضعیت فایل کارکنان ناموفق: %w", err)
	}
	s.mu.RLock()
	unchanged := info.ModTime().Equal(s.modTime)
	s.mu.RUnlock()
	if unchanged {
		return nil
	}
	return s.reload()
}

// reload فایل CSV را می‌خواند و فهرست کارکنان را جایگزین می‌کند.
func (s *StaffStore) reload() error {
	file, err := os.Open(s.path)
	if err != nil {
		return fmt.Errorf("باز کردن فایل کارکنان ناموفق: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// فایل ممکن است با ویرایشگرهای ویندوزی ذخیره شده باشد؛ فاصله‌های اضافی و تعداد
	// نامساوی ستون‌ها نباید خواندن را متوقف کند.
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	items := make([]contracts.Employee, 0, 64)
	index := map[string]int{}
	warn := ""
	line := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("خواندن سطر %d فایل کارکنان ناموفق: %w", line+1, err)
		}
		line++
		cells := trimCells(row)
		if line == 1 && looksLikeHeader(cells) {
			continue
		}
		if len(cells) == 0 || strings.TrimSpace(cells[0]) == "" {
			continue
		}
		emp := staffRow(cells)
		if emp.FullName == "" {
			warn = fmt.Sprintf("سطر %d فایل کارکنان نام دارد و نادیده گرفته شد.", line)
			continue
		}
		if _, dup := index[emp.ID]; dup {
			warn = fmt.Sprintf("کد پرسنلی تکراری «%s» در سطر %d؛ سطر تازه نادیده گرفته شد.", emp.ID, line)
			continue
		}
		index[emp.ID] = len(items)
		items = append(items, emp)
	}

	info, _ := os.Stat(s.path)
	s.mu.Lock()
	s.items = items
	s.index = index
	s.warn = warn
	if info != nil {
		s.modTime = info.ModTime()
	}
	s.mu.Unlock()
	return nil
}

// staffRow یک سطر CSV را به کارمند تبدیل می‌کند. ترتیب ستون‌ها همان سرستون نمونه است:
// کد پرسنلی، نام، واحد، سمت، وضعیت؛ سطر ناقص با مقدار پیش‌فرض پر می‌شود.
func staffRow(cells []string) contracts.Employee {
	at := func(i int) string {
		if i < len(cells) {
			return strings.TrimSpace(cells[i])
		}
		return ""
	}
	dept := at(2)
	return contracts.Employee{
		ID:             at(0),
		FullName:       at(1),
		DepartmentID:   departmentKey(dept),
		DepartmentName: dept,
		Position:       at(3),
		Status:         statusKey(at(4)),
		HireDate:       at(5),
		EmploymentType: employmentKey(at(6)),
	}
}

// looksLikeHeader تشخیص سطر سرستون بر پایهٔ نام‌های فارسی و لاتین ستون‌ها.
func looksLikeHeader(cells []string) bool {
	first := strings.ToLower(strings.TrimSpace(cells[0]))
	return first == "id" || first == "کد" || first == "کد پرسنلی" || strings.HasPrefix(first, "کد پرسنل")
}

// statusKey وضعیت فارسی/لاتین را به کلید وضعیت سامانه نگاشت می‌کند. نبود مقدار یعنی فعال.
func statusKey(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "active", "فعال", "شاغل":
		return "active"
	case "terminated", "خاتمه‌یافته", "خاتمه یافته", "پایان‌یافته", "پایان یافته", "غیرفعال":
		return "terminated"
	case "leave", "مرخصی", "بدون حقوق":
		return "leave"
	default:
		return strings.TrimSpace(value)
	}
}

// employmentKey نوع استخدام: ستون اختیاری برای گروه‌بندی «شرکتی/ارکان ثالث» در گزارش‌ها.
func employmentKey(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "company", "شرکتی":
		return "company"
	case "thirdparty", "third_party", "پیمانکاری", "ارکان ثالث":
		return "thirdParty"
	case "contractor", "پیمانکار":
		return "contractor"
	default:
		return strings.TrimSpace(value)
	}
}

// departmentKey کلید واحد از نام فارسی؛ برای فیلتر دقیق واحد در فهرست کارکنان.
func departmentKey(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "_", "‌", "_", "‏", "")
	return strings.ToLower(replacer.Replace(name))
}

func staffMatch(e contracts.Employee, query string) bool {
	return strings.Contains(strings.ToLower(e.ID), query) ||
		strings.Contains(strings.ToLower(e.FullName), query) ||
		strings.Contains(strings.ToLower(e.DepartmentName), query) ||
		strings.Contains(strings.ToLower(e.Position), query)
}

// trimCells نشانهٔ BOM ابتدای فایل و فاصله‌های اضافی ستون‌ها را پاک می‌کند.
func trimCells(row []string) []string {
	out := make([]string, 0, len(row))
	for i, c := range row {
		c = strings.TrimSpace(strings.TrimPrefix(c, "\uFEFF"))
		if i == 0 {
			c = strings.TrimPrefix(c, "\uFEFF")
		}
		out = append(out, c)
	}
	return out
}

func paging(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 1000 {
		size = 100
	}
	return page, size
}
