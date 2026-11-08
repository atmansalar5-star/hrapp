// Package domain دامنهٔ ماژول کارکنان.
//
// این بسته هیچ وابستگی به HTTP، دیتابیس یا پلتفرم ندارد؛ فقط نوع داده و قاعدهٔ
// اعتبارسنجی. همین قواعد در هستهٔ Rust نیز وجود دارد (`hrms-domain`) و ورودی کلاینت
// پیش از رسیدن به سرور هم در همان‌جا سنجیده می‌شود؛ اینجا برای اطمینان، دوباره.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/rules"
)

// کارمند در حافظهٔ سامانه.
type Employee struct {
	ID             string
	FullName       string
	NationalID     string
	DepartmentID   string
	DepartmentName string
	Position       string
	ContractType   string
	EmploymentType string
	Status         string
	HireDate       string
	BaseSalary     int64
	LeaveBalance   float64
	Gender         string
	BirthDate      string
	Phone          string
	Mobile         string
	HomePhone      string
	PostalCode     string
	Address        string
	// ResidenceSite کلید محل سکونت (فهرست `rules.OilResidences`)؛ مبنای درصد جدول ۱ و ۲
	// آیین‌نامهٔ فوق‌العاده‌های صندوق بازنشستگی نفت. خالی یعنی ثبت نشده.
	ResidenceSite string
	// JobGrade کلید گروه شغلی کارمند (نردبان `rules.DefaultGradeLadder`). اجرای ابلاغ
	// انتصاب یا ترفیع آن را می‌نویسد و فیش حقوق صندوق نفت «پایهٔ سمت» را از همین گروه
	// پیشنهاد می‌دهد؛ خالی یعنی ثبت نشده.
	JobGrade    string
	Email       string
	ManagerName string
	Version     int
}

// ToContract تبدیل به شکل انتقالی قرارداد API.
func (e Employee) ToContract() contracts.Employee {
	return contracts.Employee{
		ID:               e.ID,
		FullName:         e.FullName,
		NationalID:       e.NationalID,
		DepartmentID:     e.DepartmentID,
		DepartmentName:   e.DepartmentName,
		Position:         e.Position,
		ContractType:     e.ContractType,
		EmploymentType:   e.EmploymentType,
		EmploymentTypeFa: contracts.EmploymentTypeTitleFa(e.EmploymentType),
		Status:           e.Status,
		HireDate:         e.HireDate,
		BaseSalary:       e.BaseSalary,
		LeaveBalance:     int(e.LeaveBalance),
		Gender:           e.Gender,
		GenderFa:         contracts.GenderTitleFa(e.Gender),
		BirthDate:        e.BirthDate,
		Phone:            e.Phone,
		Mobile:           e.Mobile,
		HomePhone:        e.HomePhone,
		PostalCode:       e.PostalCode,
		Address:          e.Address,
		ResidenceSite:    e.ResidenceSite,
		ResidenceSiteFa:  rules.OilResidenceTitleFa(e.ResidenceSite),
		JobGrade:         e.JobGrade,
		JobGradeFa:       gradeTitleFa(e.JobGrade),
		Email:            e.Email,
		ManagerName:      e.ManagerName,
		Version:          e.Version,
	}
}

// FromInput ساخت کارمند از ورودی قرارداد.
func FromInput(id string, in contracts.EmployeeInput, departmentName string) Employee {
	status := in.Status
	if status == "" {
		status = "active"
	}
	return Employee{
		ID:             id,
		FullName:       strings.TrimSpace(in.FullName),
		NationalID:     strings.TrimSpace(in.NationalID),
		DepartmentID:   in.DepartmentID,
		DepartmentName: departmentName,
		Position:       strings.TrimSpace(in.Position),
		ContractType:   in.ContractType,
		EmploymentType: resolveEmploymentType(in.EmploymentType, in.ContractType),
		Status:         status,
		HireDate:       in.HireDate,
		BaseSalary:     in.BaseSalary,
		Phone:          in.Phone,
		Gender:         strings.TrimSpace(in.Gender),
		BirthDate:      calendar.NormalizeDigits(strings.TrimSpace(in.BirthDate)),
		Mobile:         strings.TrimSpace(in.Mobile),
		HomePhone:      strings.TrimSpace(in.HomePhone),
		PostalCode:     calendar.NormalizeDigits(strings.TrimSpace(in.PostalCode)),
		Address:        strings.TrimSpace(in.Address),
		ResidenceSite:  resolveResidenceSite(in.ResidenceSite),
		JobGrade:       resolveJobGrade(in.JobGrade),
		Email:          in.Email,
	}
}

// Apply روی رکورد موجود اعمال می‌کند (ویرایش).
func (e Employee) Apply(in contracts.EmployeeInput) Employee {
	if strings.TrimSpace(in.FullName) != "" {
		e.FullName = strings.TrimSpace(in.FullName)
	}
	if strings.TrimSpace(in.NationalID) != "" {
		e.NationalID = strings.TrimSpace(in.NationalID)
	}
	if in.DepartmentID != "" {
		e.DepartmentID = in.DepartmentID
	}
	if strings.TrimSpace(in.Position) != "" {
		e.Position = strings.TrimSpace(in.Position)
	}
	if in.ContractType != "" {
		e.ContractType = in.ContractType
	}
	if in.EmploymentType != "" {
		e.EmploymentType = in.EmploymentType
	}
	if in.Status != "" {
		e.Status = in.Status
	}
	if in.HireDate != "" {
		e.HireDate = in.HireDate
	}
	if in.Gender != "" {
		e.Gender = strings.TrimSpace(in.Gender)
	}
	if in.BirthDate != "" {
		e.BirthDate = calendar.NormalizeDigits(strings.TrimSpace(in.BirthDate))
	}
	if in.BaseSalary > 0 {
		e.BaseSalary = in.BaseSalary
	}
	if in.Phone != "" {
		e.Phone = in.Phone
	}
	if in.Mobile != "" {
		e.Mobile = strings.TrimSpace(in.Mobile)
	}
	if in.HomePhone != "" {
		e.HomePhone = strings.TrimSpace(in.HomePhone)
	}
	if in.PostalCode != "" {
		e.PostalCode = calendar.NormalizeDigits(strings.TrimSpace(in.PostalCode))
	}
	if in.Address != "" {
		e.Address = strings.TrimSpace(in.Address)
	}
	if site := resolveResidenceSite(in.ResidenceSite); site != "" {
		e.ResidenceSite = site
	}
	if grade := resolveJobGrade(in.JobGrade); grade != "" {
		e.JobGrade = grade
	}
	if in.Email != "" {
		e.Email = in.Email
	}
	return e
}

// Statuses وضعیت‌های مجاز کارمند — از کاتالوگ مشترک قراردادها می‌آید تا فهرست
// گزینه‌های API، اعتبارسنجی و فیلتر گزارش همه یک جا داشته باشند.
var Statuses = contracts.EmployeeStatusKeys()

// ContractTypes انواع قرارداد مجاز (هم‌خوان با سند طراحی).
var ContractTypes = []string{"تمام‌وقت", "پیمانکاری", "پاره‌وقت"}

// EmploymentTypes فهرست انواع استخدام از کاتالوگ مشترک قراردادها.
func EmploymentTypes() []contracts.EmploymentType { return contracts.EmploymentTypes() }

// resolveEmploymentType کلید نوع استخدام را برمی‌گرداند؛ اگر خالی بماند از نوع قرارداد
// استنتاج می‌شود (پیمانکاری → پیمانکاری جاری، وگرنه رسمی نفت) تا رکوردهای قدیمی هم
// نوع استخدام معتبر داشته باشند.
func resolveEmploymentType(employmentType, contractType string) string {
	if t, ok := contracts.EmploymentTypeByKey(employmentType); ok {
		return t.Key
	}
	if strings.Contains(contractType, "پیمانکار") {
		return "contractor_current"
	}
	return "official_oil"
}

// gradeTitleFa عنوان فارسی گروه شغلی؛ گروه ثبت‌نشده خالی برمی‌گردد (نه «نامشخص») تا
// رابط بتواند «ثبت نشده» را خودش بنویسد.
func gradeTitleFa(key string) string {
	if strings.TrimSpace(key) == "" {
		return ""
	}
	return rules.GradeTitleFa(rules.GradeRankOf(key))
}

// ResolveJobGrade کلید معتبر گروه شغلی را برمی‌گرداند؛ کلید ناشناخته خالی می‌شود.
//
// صادرشده تا لایهٔ کاربرد (پورت تغییر سازمانی) همان اعتبارسنجی دامنه را به کار ببرد و
// گروهی بیرون از نردبان گروه‌ها در پرونده ننشیند.
func ResolveJobGrade(key string) string { return resolveJobGrade(key) }

// resolveJobGrade کلید معتبر گروه شغلی را برمی‌گرداند؛ کلید ناشناخته خالی می‌شود تا
// گروهی بیرون از نردبان گروه‌های سازمان در پرونده ثبت نشود.
func resolveJobGrade(key string) string {
	value := strings.TrimSpace(key)
	if rules.GradeRankOf(value) == 0 {
		return ""
	}
	return value
}

// resolveResidenceSite کلید معتبر محل سکونت را برمی‌گرداند؛ کلید ناشناخته خالی می‌شود تا
// درصدی از جدول ۱ و ۲ آیین‌نامهٔ فوق‌العاده‌ها به اشتباه به کارمند نسبت داده نشود.
// اعتبار کلید از کاتالوگ مشترک `rules.OilResidences` می‌آید.
func resolveResidenceSite(key string) string {
	if site, ok := rules.OilResidenceByKey(key); ok {
		return site.Key
	}
	return ""
}

func validNationalID(input string) bool {
	if len(input) != 10 {
		return false
	}
	digits := make([]int, 10)
	for i, c := range input {
		if c < '0' || c > '9' {
			return false
		}
		digits[i] = int(c - '0')
	}
	allSame := true
	for _, d := range digits {
		if d != digits[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return false
	}
	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * (10 - i)
	}
	r := sum % 11
	if r < 2 {
		return digits[9] == r
	}
	return digits[9] == 11-r
}

// Validate اعتبارسنجی ورودی کارمند؛ پیام‌ها فارسی و کلیدها نام فیلدهای فرم هستند.
func Validate(in contracts.EmployeeInput) map[string]string {
	errs := map[string]string{}

	if len([]rune(strings.TrimSpace(in.FullName))) < 3 {
		errs["fullName"] = "نام و نام خانوادگی باید حداقل ۳ نویسه باشد."
	}
	if !validNationalID(strings.TrimSpace(in.NationalID)) {
		errs["nationalId"] = "کد ملی باید ۱۰ رقم و دارای رقم کنترل معتبر باشد."
	}
	if strings.TrimSpace(in.DepartmentID) == "" {
		errs["departmentId"] = "انتخاب واحد سازمانی الزامی است."
	}
	if strings.TrimSpace(in.Position) == "" {
		errs["position"] = "عنوان پست الزامی است."
	}
	if _, err := calendar.Parse(in.HireDate); err != nil {
		errs["hireDate"] = "تاریخ استخدام باید به قالب شمسی YYYY/MM/DD باشد."
	}
	if in.BaseSalary < 0 {
		errs["baseSalary"] = "حقوق پایه نمی‌تواند منفی باشد."
	}
	if in.ContractType != "" {
		ok := false
		for _, t := range ContractTypes {
			if t == in.ContractType {
				ok = true
				break
			}
		}
		if !ok {
			errs["contractType"] = "نوع قرارداد باید یکی از تمام‌وقت، پیمانکاری یا پاره‌وقت باشد."
		}
	}
	if in.EmploymentType != "" && !contracts.ValidEmploymentType(in.EmploymentType) {
		errs["employmentType"] = "نوع استخدام باید یکی از انواع فهرست رسمی باشد."
	}
	if in.Status != "" {
		ok := false
		for _, s := range Statuses {
			if s == in.Status {
				ok = true
				break
			}
		}
		if !ok {
			errs["status"] = "وضعیت کارمند نامعتبر است."
		}
	}
	// اطلاعات پرسنلی: جنسیت از کاتالوگ مشترک و تاریخ تولد شمسی با بازهٔ سنی معقول.
	// گروه سنی در آمار روی همین تاریخ ساخته می‌شود، پس قالب درست آن مهم است.
	if gender := strings.TrimSpace(in.Gender); gender != "" && !contracts.ValidGender(gender) {
		errs["gender"] = "جنسیت باید یکی از گزینه‌های «مرد» یا «زن» باشد."
	}
	if birth := calendar.NormalizeDigits(strings.TrimSpace(in.BirthDate)); birth != "" {
		date, err := calendar.Parse(birth)
		if err != nil {
			errs["birthDate"] = "تاریخ تولد باید به قالب شمسی YYYY/MM/DD باشد."
		} else if date.Year < 1300 || date.Year > 1420 {
			errs["birthDate"] = "سال تولد باید بین ۱۳۰۰ و ۱۴۲۰ باشد."
		}
	}
	if in.Email != "" && !looksLikeEmail(in.Email) {
		errs["email"] = "قالب رایانامه معتبر نیست."
	}
	if in.Phone != "" && countDigits(in.Phone) < 8 {
		errs["phone"] = "شمارهٔ تماس باید حداقل ۸ رقم داشته باشد."
	}
	// اطلاعات پایهٔ پرونده: قاعده‌ها اینجا هستند تا فرم‌های مصرف‌کننده (قرارداد، حکم،
	// کارگزینی) همان‌ها را نشان دهند و هیچ قاعده‌ای در رابط تکرار نشود.
	if mobile := calendar.NormalizeDigits(strings.TrimSpace(in.Mobile)); mobile != "" {
		if len(mobile) != 11 || !strings.HasPrefix(mobile, "09") {
			errs["mobile"] = "شمارهٔ موبایل باید ۱۱ رقم و با ۰۹ آغاز شود."
		}
	}
	if home := calendar.NormalizeDigits(strings.TrimSpace(in.HomePhone)); home != "" {
		if len(home) < 8 {
			errs["homePhone"] = "شمارهٔ منزل باید حداقل ۸ رقم داشته باشد."
		}
	}
	if postal := calendar.NormalizeDigits(strings.TrimSpace(in.PostalCode)); postal != "" {
		if len(postal) != 10 {
			errs["postalCode"] = "کد پستی باید ۱۰ رقم باشد."
		}
	}
	if addr := strings.TrimSpace(in.Address); addr != "" && len([]rune(addr)) < 10 {
		errs["address"] = "نشانی محل سکونت باید حداقل ۱۰ نویسه باشد."
	}
	return errs
}

func looksLikeEmail(s string) bool {
	at := strings.Index(s, "@")
	if at <= 0 || at == len(s)-1 {
		return false
	}
	domain := s[at+1:]
	if strings.Contains(domain, "@") || !strings.Contains(domain, ".") {
		return false
	}
	return !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}

func countDigits(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n++
		}
	}
	return n
}
