package contracts

// SalaryDecree حکم حقوق یک کارمند (گروه شغلی، پایه و تاریخ اجرا).
//
// حکم حقوق سند مبنای پرداخت است، پس هر تغییر یک حکم تازه می‌سازد و حکم پیشین را
// منسوخ می‌کند؛ احکام گذشته بازنویسی نمی‌شوند تا سابقهٔ پرداخت قابل ردیابی بماند.
//
// اجزای ثابت مزد و فوق‌العاده‌های اقلیمی همین‌جا ثبت می‌شوند (WageStructure): فیش حقوق
// مزد مبنا را از این اجزا می‌سازد و ضرایب اقلیمی را از همین حکم می‌خواند. هر فیلد صفر
// یعنی ثبت نشده و پارامتر شرکت به کار می‌رود.
type SalaryDecree struct {
	ID               string `json:"id"`
	EmployeeID       string `json:"employeeId"`
	EmployeeName     string `json:"employeeName"`
	DepartmentName   string `json:"departmentName"`
	Position         string `json:"position,omitempty"`
	EmploymentType   string `json:"employmentType,omitempty"`
	EmploymentTypeFa string `json:"employmentTypeFa,omitempty"`
	JobGroup         string `json:"jobGroup"`
	BaseSalary       int64  `json:"baseSalary"`
	EffectiveDate    string `json:"effectiveDate"`
	ReasonFa         string `json:"reasonFa"`
	Status           string `json:"status"`
	StatusFa         string `json:"statusFa"`
	SupersedesID     string `json:"supersedesId,omitempty"`
	NoteFa           string `json:"noteFa,omitempty"`
	CreatedBy        string `json:"createdBy,omitempty"`
	DecidedBy        string `json:"decidedBy,omitempty"`
	Version          int    `json:"version"`

	WageStructure

	// MazdBase مزد مبنا: از اجزای همین حکم ساخته می‌شود (قاعده در بستهٔ rules) تا رابط
	// کاربری همان عددی را نشان دهد که فیش و سند چاپی حساب می‌کنند. فقط‌خواندنی است.
	MazdBase int64 `json:"mazdBase,omitempty"`
	// ClimateRegionFa عنوان فارسی منطقهٔ اقلیمی برای نمایش.
	ClimateRegionFa string `json:"climateRegionFa,omitempty"`
}

// SalaryDecreeInput ورودی صدور یا ویرایش پیش‌نویس حکم حقوق.
type SalaryDecreeInput struct {
	EmployeeID     string `json:"employeeId"`
	EmploymentType string `json:"employmentType,omitempty"`
	JobGroup       string `json:"jobGroup"`
	BaseSalary     int64  `json:"baseSalary"`
	EffectiveDate  string `json:"effectiveDate"`
	ReasonFa       string `json:"reasonFa"`
	NoteFa         string `json:"noteFa,omitempty"`

	WageStructure
}

// SalaryDecreeGroupInput ورودی صدور گروهی حکم کارگزینی.
//
// صدور گروهی بر پایهٔ «نوع استخدام» انجام می‌شود: مدیر منابع انسانی یک نوع استخدام را
// برمی‌گزیند، درصد/مبلغ افزایش را می‌دهد و سامانه برای همهٔ کارکنان فعال آن نوع حکم
// می‌سازد. اگر EmployeeIDs داده شود، فقط همان‌ها هدف می‌شوند (صدور گزینشی).
//
// افزایش می‌تواند درصدی یا مبلغی باشد؛ مقدار نهایی با `BaseSalary` هر کارمند حساب
// می‌شود و همان موتور مشترک قواعد، شاکلهٔ مزد را برای حکم می‌سازد.
//
// DryRun برای پیش‌نمایش است: هیچ حکمی ذخیره نمی‌شود و فقط فهرست هدف‌ها و مبلغ تازه
// برگردانده می‌شود تا کاربر پیش از تأیید ببیند.
type SalaryDecreeGroupInput struct {
	// EmploymentType نوع استخدام هدف؛ خالی یعنی همهٔ انواع.
	EmploymentType string `json:"employmentType,omitempty"`
	// DepartmentID محدودسازی اختیاری به یک اداره.
	DepartmentID string `json:"departmentId,omitempty"`
	// EmployeeIDs انتخاب دستی چند کارمند (صدور گزینشی).
	EmployeeIDs []string `json:"employeeIds,omitempty"`
	// IncreasePercent درصد افزایش (مثلاً ۱۸ برای ۱۸٪).
	IncreasePercent float64 `json:"increasePercent,omitempty"`
	// IncreaseAmount مبلغ افزایش ثابت به ریال.
	IncreaseAmount int64 `json:"increaseAmount,omitempty"`
	// RoundTo گام گرد کردن مبلغ نهایی؛ صفر یعنی بدون گرد کردن (پیش‌فرض ۱۰٬۰۰۰ ریال).
	RoundTo int64 `json:"roundTo,omitempty"`
	// EffectiveDate تاریخ اجرای حکم‌ها.
	EffectiveDate string `json:"effectiveDate"`
	// ReasonFa علت صدور (مثلاً «افزایش سالانه ۱۴۰۵»).
	ReasonFa string `json:"reasonFa"`
	// NoteFa یادداشت اختیاری.
	NoteFa string `json:"noteFa,omitempty"`
	// DryRun پیش‌نمایش بدون ذخیره.
	DryRun bool `json:"dryRun,omitempty"`
}

// SalaryDecreeGroupRow یک ردیف نتیجهٔ صدور گروهی (یا پیش‌نمایش).
//
// Status یکی از draft (ساخته‌شد), skipped (کارمند فعال نبود) یا failed (خطا) است؛
// ErrorFa فقط برای failed پر می‌شود. این ساختار هم برای پیش‌نمایش و هم نتیجهٔ واقعی
// یکسان است تا رابط کاربری یک جدول را نشان دهد.
type SalaryDecreeGroupRow struct {
	EmployeeID       string `json:"employeeId"`
	EmployeeName     string `json:"employeeName"`
	DepartmentName   string `json:"departmentName,omitempty"`
	EmploymentType   string `json:"employmentType,omitempty"`
	EmploymentTypeFa string `json:"employmentTypeFa,omitempty"`
	OldSalary        int64  `json:"oldSalary"`
	NewSalary        int64  `json:"newSalary"`
	Delta            int64  `json:"delta"`
	DecreeID         string `json:"decreeId,omitempty"`
	Status           string `json:"status"`
	ErrorFa          string `json:"errorFa,omitempty"`
}

// SalaryDecreeGroupResult نتیجهٔ صدور گروهی حکم.
type SalaryDecreeGroupResult struct {
	DryRun         bool                   `json:"dryRun"`
	EmploymentType string                 `json:"employmentType,omitempty"`
	Rows           []SalaryDecreeGroupRow `json:"rows"`
	Created        int                    `json:"created"`
	Skipped        int                    `json:"skipped"`
	Failed         int                    `json:"failed"`
	TotalDelta     int64                  `json:"totalDelta"`
}

// SalaryDecreeDecision ورودی اصلاح حکم (نسخهٔ تازه) با مبلغ و علت.
//
// اجزای مزد و ضرایب اقلیمی هم قابل اصلاح‌اند: افزایش منطقهٔ اقلیمی یا افزودن فوق‌العادهٔ
// اقلیمی، حکم تازه می‌سازد و همان لحظه در فیش حقوق دیده می‌شود.
type SalaryDecreeDecision struct {
	BaseSalary int64  `json:"baseSalary,omitempty"`
	ReasonFa   string `json:"reasonFa,omitempty"`
	NoteFa     string `json:"noteFa,omitempty"`

	WageStructure
}

// SalaryDecreePage صفحهٔ احکام حقوق.
type SalaryDecreePage struct {
	Items []SalaryDecree `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// SalaryDecreeFilter فیلترهای فهرست احکام؛ Scope محدودهٔ دادهٔ نقش را حمل می‌کند.
type SalaryDecreeFilter struct {
	Query           string
	EmployeeID      string
	Status          string
	JobGroup        string
	EmploymentType  string
	Sort            string
	Page            int
	Size            int
	Scope           string
	ActorEmployeeID string
	ActorDepartment string
}

// EmploymentCertificate گواهی اشتغال به کار.
//
// گواهی برای مراجع بیرونی (بانک، سفارت) صادر می‌شود، پس شمارهٔ سریال یکتا و تاریخ
// صدور دارد و متن آن از دادهٔ کارمند و قرارداد جاری ساخته می‌شود.
type EmploymentCertificate struct {
	ID             string `json:"id"`
	Serial         string `json:"serial"`
	EmployeeID     string `json:"employeeId"`
	EmployeeName   string `json:"employeeName"`
	DepartmentName string `json:"departmentName"`
	Position       string `json:"position,omitempty"`
	Kind           string `json:"kind"`
	KindFa         string `json:"kindFa"`
	PurposeFa      string `json:"purposeFa,omitempty"`
	RequestedAt    string `json:"requestedAt"`
	IssuedAt       string `json:"issuedAt,omitempty"`
	DeliveredAt    string `json:"deliveredAt,omitempty"`
	Status         string `json:"status"`
	StatusFa       string `json:"statusFa"`
	NoteFa         string `json:"noteFa,omitempty"`
	DecidedBy      string `json:"decidedBy,omitempty"`
	Version        int    `json:"version"`
}

// CertificateInput ورودی ثبت یا ویرایش درخواست گواهی اشتغال.
type CertificateInput struct {
	EmployeeID string `json:"employeeId"`
	Kind       string `json:"kind"`
	PurposeFa  string `json:"purposeFa,omitempty"`
	NoteFa     string `json:"noteFa,omitempty"`
}

// CertificateDecision ورودی تأیید/رد درخواست گواهی.
type CertificateDecision struct {
	Approve bool   `json:"approve"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// CertificatePage صفحهٔ گواهی‌های اشتغال.
type CertificatePage struct {
	Items []EmploymentCertificate `json:"items"`
	Total int                     `json:"total"`
	Page  int                     `json:"page"`
	Size  int                     `json:"size"`
}

// CertificateFilter فیلترهای فهرست گواهی‌ها.
type CertificateFilter struct {
	Query           string
	EmployeeID      string
	Kind            string
	Status          string
	Page            int
	Size            int
	Scope           string
	ActorEmployeeID string
	ActorDepartment string
}
