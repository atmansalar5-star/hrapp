// قراردادها و نوع‌های مشترک دامنهٔ «employee» — زبان مشترک بین ماژول‌ها، کلاینت‌ها و پایگاه داده.

package contracts

import (
	"context"
	"strings"
)

// Employee رکورد کارمند — ستون‌ها مطابق دستور ۳-۲ سند طراحی (شناسه، نام، واحد، پست، وضعیت).
type Employee struct {
	ID             string `json:"id"`
	FullName       string `json:"fullName"`
	NationalID     string `json:"nationalId"`
	DepartmentID   string `json:"departmentId"`
	DepartmentName string `json:"departmentName"`
	Position       string `json:"position"`
	ContractType   string `json:"contractType"`
	// EmploymentType نوع استخدام (کلید کاتالوگ employment.go)؛ مبنای صدور حکم، قرارداد و مزد.
	EmploymentType   string `json:"employmentType,omitempty"`
	EmploymentTypeFa string `json:"employmentTypeFa,omitempty"`
	Status           string `json:"status"`
	HireDate         string `json:"hireDate"`
	BaseSalary       int64  `json:"baseSalary"`
	LeaveBalance     int    `json:"leaveBalance"`
	Phone            string `json:"phone"`
	// Mobile و HomePhone شمارهٔ همراه و تلفن منزل کارمند؛ از تلفن تماس (Phone) جدا
	// هستند تا گزارش‌ها و فرم‌های قرارداد/حکم هر کدام را جداگانه نشان دهند.
	Mobile    string `json:"mobile,omitempty"`
	HomePhone string `json:"homePhone,omitempty"`
	// PostalCode و Address اطلاعات نشانی کارمند برای فرم‌های قرارداد، حکم و مکاتبات.
	PostalCode  string `json:"postalCode,omitempty"`
	Address     string `json:"address,omitempty"`
	Email       string `json:"email"`
	ManagerName string `json:"managerName"`
	Version     int    `json:"version"`
	// PhotoURL نشانی سرو «عکس کارگزینی» این کارمند (امضاشده)؛ خالی یعنی عکس
	// بارگذاری نشده است. عکس در پوشهٔ دادهٔ سرور نگه داشته می‌شود و این نشانی از
	// خودِ همان فایل ساخته می‌شود، پس با بازآوری باینری از دست نمی‌رود.
	PhotoURL string `json:"photoUrl,omitempty"`
	// PhotoUpdatedAt تاریخ شمسی آخرین بارگذاری عکس (برای نمایش در پرونده).
	PhotoUpdatedAt string `json:"photoUpdatedAt,omitempty"`
	// PhotoSizeKb حجم عکس به کیلوبایت.
	PhotoSizeKb int `json:"photoSizeKb,omitempty"`
}

// EmployeePhoto وضعیت «عکس کارگزینی» یک کارمند؛ پاسخ مسیرهای بارگذاری، خواندن و
// حذف عکس. NoteFa توضیح فارسی وضعیت است تا متن در رابط ساخته نشود.
type EmployeePhoto struct {
	EmployeeID  string `json:"employeeId"`
	HasPhoto    bool   `json:"hasPhoto"`
	URL         string `json:"url,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	UpdatedBy   string `json:"updatedBy,omitempty"`
	SizeKb      int    `json:"sizeKb,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	// OriginalName نام فایل اصلی که کاربر بارگذاری کرده است (برای پیگیری کارگزینی).
	OriginalName string `json:"originalName,omitempty"`
	NoteFa       string `json:"noteFa,omitempty"`
}

// EmployeeInput ورودی ایجاد/ویرایش کارمند (فیلدهای اجباری در domain اعتبارسنجی می‌شوند).
type EmployeeInput struct {
	FullName     string `json:"fullName"`
	NationalID   string `json:"nationalId"`
	DepartmentID string `json:"departmentId"`
	Position     string `json:"position"`
	ContractType string `json:"contractType"`
	// EmploymentType کلید نوع استخدام؛ اگر خالی بماند از ContractType استنتاج می‌شود.
	EmploymentType string `json:"employmentType,omitempty"`
	HireDate       string `json:"hireDate"`
	Status         string `json:"status,omitempty"`
	BaseSalary     int64  `json:"baseSalary,omitempty"`
	Phone          string `json:"phone,omitempty"`
	// شمارهٔ همراه، تلفن منزل، کد پستی و نشانی — اطلاعات پایه‌ای که در فرم‌های
	// قرارداد، حکم و سایر فرم‌های نیازمند اطلاعات پایه هم نمایش داده می‌شود.
	Mobile     string `json:"mobile,omitempty"`
	HomePhone  string `json:"homePhone,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	Address    string `json:"address,omitempty"`
	Email      string `json:"email,omitempty"`
}

// EmployeePage صفحهٔ کارکنان همراه با شمارش کل رکوردها.
type EmployeePage struct {
	Items []Employee `json:"items"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
}

// Department واحد سازمانی در نمودار درختی.
type Department struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Head      string `json:"head,omitempty"`
	Headcount int    `json:"headcount,omitempty"`
	ParentID  string `json:"parentId,omitempty"`
}

// EmployeeStatus یک وضعیت کارمند با عنوان فارسی.
//
// مرجع واحد وضعیت‌ها: فهرست مجاز API، اعتبارسنجی دامنه و فیلترهای گزارش همه از همین
// کاتالوگ می‌خوانند تا برچسب و کلید وضعیت در چند جا تکرار نشود.
type EmployeeStatus struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// EmployeeStatusCatalog وضعیت‌های مجاز کارمند (هم‌خوان با enum جدول کارکنان).
var EmployeeStatusCatalog = []EmployeeStatus{
	{Key: "active", TitleFa: "شاغل"},
	{Key: "leave", TitleFa: "مرخصی"},
	{Key: "probation", TitleFa: "آزمایشی"},
	{Key: "terminated", TitleFa: "پایان خدمت"},
}

// EmployeeStatuses فهرست وضعیت‌ها با عنوان فارسی.
func EmployeeStatuses() []EmployeeStatus {
	out := make([]EmployeeStatus, len(EmployeeStatusCatalog))
	copy(out, EmployeeStatusCatalog)
	return out
}

// EmployeeStatusKeys کلیدهای مجاز وضعیت (برای اعتبارسنجی و فهرست گزینه‌های API).
func EmployeeStatusKeys() []string {
	out := make([]string, 0, len(EmployeeStatusCatalog))
	for _, s := range EmployeeStatusCatalog {
		out = append(out, s.Key)
	}
	return out
}

// EmployeeStatusTitleFa عنوان فارسی یک وضعیت؛ برای کلید ناشناخته خودِ ورودی برمی‌گردد.
func EmployeeStatusTitleFa(key string) string {
	key = strings.TrimSpace(key)
	for _, s := range EmployeeStatusCatalog {
		if s.Key == key {
			return s.TitleFa
		}
	}
	return key
}

// EmployeeFilter فیلترهای فهرست کارکنان؛ Scope برای اعمال محدودهٔ دسترسی نقش استفاده می‌شود.
type EmployeeFilter struct {
	Query        string
	DepartmentID string
	Status       string
	ContractType string
	// EmploymentType یک نوع استخدام (فیلتر تک‌مقداری).
	EmploymentType string
	// EmploymentTypes فهرست انواع استخدام انتخابی؛ اگر پر باشد بر EmploymentType
	// مقدم است و سطر با «هر یک از» این انواع تطبیق داده می‌شود (فیلتر چندانتخابی).
	EmploymentTypes []string
	// Statuses وضعیت‌های انتخابی کارمند؛ همان قاعده: پر بودنش بر Status مقدم است.
	Statuses []string
	Sort     string
	Page     int
	Size     int
	// Scope محدودهٔ داده: all | own_unit | own
	Scope string
	// ActorID شناسهٔ کارمندی که کاربر جاری به آن وصل است (برای scope=own)
	ActorID string
	// ActorDepartment واحد کاربر جاری (برای scope=own_unit)
	ActorDepartment string
}

// EmployeeReader پورت خواندن کارکنان — تنها راه ماژول‌های دیگر برای دیدن داده کارکنان.
// ماژول employee آن را پیاده می‌کند و از طریق modulekit.Ports در اختیار بقیه می‌گذارد.
type EmployeeReader interface {
	List(ctx context.Context, f EmployeeFilter) (EmployeePage, error)
	// Get خواندن یک کارمند با همان محدودهٔ دادهٔ فهرست؛ داشتن فیلتر عمدی است تا
	// مصرف‌کننده نتواند کارمند خارج از محدودهٔ دسترسی نقش را بخواند.
	Get(ctx context.Context, f EmployeeFilter, id string) (Employee, bool, error)
	CountByStatus(ctx context.Context, f EmployeeFilter) (map[string]int, error)
	// Departments واحدهای سازمانی به‌همراه والد هر واحد (نمودار درختی). مصرف‌کننده:
	// ساختار سازمانی — تا درخت واحدها از مالکِ داده خوانده شود و فهرست واحدها در
	// مصرف‌کننده تکرار نشود.
	Departments(ctx context.Context) ([]Department, error)
}
