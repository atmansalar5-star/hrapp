package contracts

import "strings"

// کاتالوگ «مدیریت‌های پتروشیمی بندر امام» — مرجع واحد ساختار تولیدی/پشتیبانی سازمان.
//
// چرا در contracts و نه در یک ماژول: چند ماژول به این فهرست نیاز دارند (وام برای جدول
// سهمیه، پاداش برای تخصیص به مدیریت‌ها، گزارش‌ها برای گروه‌بندی) و هیچ ماژولی نباید
// داخلی‌های ماژول دیگر را import کند. اینجا همان الگوی `EmploymentTypeCatalog` است:
// فهرست «دادهٔ مرجع» است، نه منطق؛ هر ماژول فقط کلیدها را مصرف می‌کند و قاعدهٔ خودش
// را در دامنهٔ خودش می‌نویسد.
//
// کلیدها لاتین و پایدارند (داده و API)، عنوان‌ها فارسی و نمایشی.

// Management یک مدیریت (واحد تولیدی یا پشتیبانی) در سازمان.
type Management struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// ManagementCatalog فهرست مدیریت‌های بندر امام به ترتیب اعلامی سازمان.
var ManagementCatalog = []Management{
	{Key: "bandar_imam", TitleFa: "بندر امام", NoteFa: "مدیریت عملیات بندر"},
	{Key: "repair_support", TitleFa: "مدیریت پشتیبانی تعمیرات", NoteFa: "تعمیرات اساسی و نگهداری"},
	{Key: "water_power", TitleFa: "آب و نیرو", NoteFa: "آب، برق و نیروگاه"},
	{Key: "basparan", TitleFa: "بسپاران", NoteFa: "واحد بسپاران"},
	{Key: "process_one", TitleFa: "فراورش یک", NoteFa: "فراورش یک"},
	{Key: "process_two", TitleFa: "فراورش دو", NoteFa: "فراورش دو"},
	{Key: "services_support", TitleFa: "معاونت پشتیبانی خدمات", NoteFa: "خدمات پشتیبانی"},
	{Key: "kimia", TitleFa: "کیمیا", NoteFa: "شرکت کیمیا"},
}

// Managements فهرست مدیریت‌ها (کپی، تا مصرف‌کننده نتواند کاتالوگ را تغییر دهد).
func Managements() []Management {
	out := make([]Management, len(ManagementCatalog))
	copy(out, ManagementCatalog)
	return out
}

// ManagementByKey یافتن مدیریت با کلید (یا با عنوان فارسی، برای داده‌های قدیمی).
func ManagementByKey(key string) (Management, bool) {
	key = strings.TrimSpace(key)
	for _, m := range ManagementCatalog {
		if m.Key == key || m.TitleFa == key {
			return m, true
		}
	}
	return Management{}, false
}

// ManagementTitleFa عنوان فارسی مدیریت؛ برای کلید ناشناخته خودِ ورودی برمی‌گردد تا
// نمایش داده‌های قدیمی خراب نشود.
func ManagementTitleFa(key string) string {
	if m, ok := ManagementByKey(key); ok {
		return m.TitleFa
	}
	return strings.TrimSpace(key)
}

// ValidManagement آیا کلید/عنوان یک مدیریت شناخته‌شده است.
func ValidManagement(key string) bool {
	_, ok := ManagementByKey(key)
	return ok
}
