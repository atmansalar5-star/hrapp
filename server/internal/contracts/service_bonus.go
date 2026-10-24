package contracts

import "context"

// پاداش پایان خدمت (پاداش سنوات) کارکنان رسمی — شکل انتقالی API.
//
// قواعد محاسبه در بستهٔ domain ماژول حقوق است؛ این ساختارها فقط داده را بین سرور و
// رابط جابه‌جا می‌کنند. همهٔ مبلغ‌ها به ریال‌اند.

// ServiceBonusTenure سابقهٔ خدمت به تفکیک سال، ماه و روز.
//
// چرا سه‌تایی و نه فقط سال: صورت‌حساب مرجع (فایل «محاسبه پاداش پایان خدمت») سابقه را
// به شکل «۳۴ سال و ۶ ماه و ۱۵ روز» ثبت می‌کند و سپس آن را به عدد اعشاری تبدیل می‌کند
// (سال + ماه/۱۲ + روز/۳۶۵). اگر فقط سال ذخیره شود، همان عدد مرجع بازتولید نمی‌شود.
type ServiceBonusTenure struct {
	Years  int `json:"years"`
	Months int `json:"months"`
	Days   int `json:"days"`
}

// ServiceBonusRates ضرایب و درصدهای محاسبه؛ از پارامترهای حقوق همان شرکت می‌آید و
// کاربر هم می‌تواند در فرم بازنویسی کند (هیچ عددی در کد سخت‌شده نیست).
type ServiceBonusRates struct {
	BadClimate       float64 `json:"badClimate"`       // بدی آب و هوا
	Special          float64 `json:"special"`          // ویژه
	Workshop         float64 `json:"workshop"`         // کارگاهی
	AttractionPublic float64 `json:"attractionPublic"` // جذب عمومی
	PersonalBase     float64 `json:"personalBase"`     // جذب (پایه شخصی)
	JobSpecialty     float64 `json:"jobSpecialty"`     // تخصصی شغل (مدرک)
}

// ServiceBonusInput ورودی محاسبهٔ پاداش پایان خدمت.
type ServiceBonusInput struct {
	EmployeeID string `json:"employeeId"`
	// BaseDate تاریخ مبنای محاسبه (شمسی YYYY/MM/DD) — همان «تاریخ مبنای محاسبه» صورت‌حساب.
	BaseDate string `json:"baseDate"`
	// BaseSalary حقوق پایه (پایه) که همهٔ درصدها از آن ساخته می‌شود.
	BaseSalary int64 `json:"baseSalary"`
	// SalaryRepair مبلغ ترمیم حقوق که به جمع افزوده می‌شود (ممکن است صفر باشد).
	SalaryRepair int64 `json:"salaryRepair"`
	// North سابقهٔ شمال (خدمت در منطقهٔ شمالی) و South سابقهٔ جنوب.
	North ServiceBonusTenure `json:"north"`
	South ServiceBonusTenure `json:"south"`
	// Rates ضرایب محاسبه؛ اگر خالی باشد از پارامترهای حقوق خوانده می‌شود.
	Rates ServiceBonusRates `json:"rates"`
	// DegreeKey مدرک تحصیلی (کلید پارامترهای خدمتی) — فقط برای گزارش و بازبینی.
	DegreeKey string `json:"degreeKey,omitempty"`
	NoteFa    string `json:"noteFa,omitempty"`
}

// ServiceBonusLine یک سطر از صورت‌حساب (عنوان، درصد، مبنا و مبلغ).
type ServiceBonusLine struct {
	TitleFa    string  `json:"titleFa"`
	Percent    float64 `json:"percent"`
	BaseFa     string  `json:"baseFa,omitempty"`
	AmountRial float64 `json:"amountRial"`
}

// ServiceBonusResult صورت‌حساب کامل پاداش پایان خدمت.
//
// نام‌گذاری از متن مرجع پیروی می‌کند: «پاداش حق جذب ۸ درصد»، «پاداش حق جذب شمال/جنوب»
// و «مبلغ محاسبه‌شدهٔ پاداش سنوات».
type ServiceBonusResult struct {
	EmployeeID     string             `json:"employeeId,omitempty"`
	EmployeeName   string             `json:"employeeName,omitempty"`
	BaseDate       string             `json:"baseDate"`
	BaseSalary     int64              `json:"baseSalary"`
	SalaryRepair   int64              `json:"salaryRepair"`
	North          ServiceBonusTenure `json:"north"`
	South          ServiceBonusTenure `json:"south"`
	Total          ServiceBonusTenure `json:"totalTenure"`
	NorthYears     float64            `json:"northYears"`
	SouthYears     float64            `json:"southYears"`
	SouthYearsUsed float64            `json:"southYearsUsed"`
	NorthLines     []ServiceBonusLine `json:"northLines"`
	SouthLines     []ServiceBonusLine `json:"southLines"`
	NorthTotal     float64            `json:"northTotal"`
	SouthTotal     float64            `json:"southTotal"`
	// AttractionBonusNorth/South: پاداش گروه ۸ درصد به تفکیک شمال و جنوب.
	AttractionBonusNorth float64 `json:"attractionBonusNorth"`
	AttractionBonusSouth float64 `json:"attractionBonusSouth"`
	// AttractionBonus: «پاداش حق جذب ۸ درصد» (جمع دو سطر بالا).
	AttractionBonus float64 `json:"attractionBonus"`
	// PersonalBonusNorth/South: پاداش پایه شخصی و تخصصی شغل به تفکیک شمال و جنوب.
	PersonalBonusNorth float64 `json:"personalBonusNorth"`
	PersonalBonusSouth float64 `json:"personalBonusSouth"`
	// TotalRial مبلغ نهایی پاداش سنوات (سطر «مبلغ محاسبه‌شده پاداش سنوات»).
	TotalRial float64 `json:"totalRial"`
	// HalfTotalRial نیمی از پاداش (سطر کمکی صورت‌حساب مرجع).
	HalfTotalRial float64 `json:"halfTotalRial"`
	// Rates ضرایب به‌کاررفته؛ در پاسخ «مقدماتی» (پیش‌پرکردن) هم همین‌جا برمی‌گردد تا
	// فرم بتواند فیلدهای درصد را با مقدار جاری شرکت نشان دهد.
	Rates ServiceBonusRates `json:"rates"`
	// Sources فهرست منابعی که داده از آن‌ها بارگذاری شد (پروندهٔ پرسنلی، حکم…‌).
	Sources []string `json:"sources,omitempty"`
	NoteFa  string   `json:"noteFa,omitempty"`
}

// SalaryDecreeReader پورت خواندن حکم حقوق کارمند برای ماژول‌های دیگر (مثل محاسبهٔ پاداش
// پایان خدمت که پایهٔ حکم ابلاغ‌شده را لازم دارد).
//
// ماژول احکام آن را ثبت می‌کند و مصرف‌کننده فقط این قرارداد را می‌بیند؛ هیچ ماژولی به
// داخلی‌های ماژول دیگر import نمی‌زند.
type SalaryDecreeReader interface {
	// NotifiedDecreeOf آخرین حکم ابلاغ‌شدهٔ کارمند را برمی‌گرداند.
	NotifiedDecreeOf(ctx context.Context, employeeID string) (SalaryDecree, bool, error)
}

// ServiceBonusParameter یک پارامتر محاسبهٔ پاداش (کلید، عنوان، مقدار و واحد) که رابط از
// همین فهرست، فرم و جدول مرجع را می‌سازد.
type ServiceBonusParameter struct {
	Key      string  `json:"key"`
	TitleFa  string  `json:"titleFa"`
	Value    float64 `json:"value"`
	UnitFa   string  `json:"unitFa,omitempty"`
	GroupFa  string  `json:"groupFa,omitempty"`
	Editable bool    `json:"editable"`
}

// ServiceBonusParameters پاسخ «پارامترها و جدول‌های مرجع» محاسبه‌گر.
type ServiceBonusParameters struct {
	Rates   ServiceBonusRates       `json:"rates"`
	Items   []ServiceBonusParameter `json:"items"`
	Degrees []ServiceBonusParameter `json:"degrees"`
	Sources []string                `json:"sources"`
	// TodayFa تاریخ امروز به شمسی (YYYY/MM/DD) — تبدیل تقویمی در لایهٔ سرور انجام
	// می‌شود و رابط فقط نمایش می‌دهد؛ پس فرم محاسبه با تاریخ مبنای درست پیش‌پر می‌شود
	// و کاربر لازم نیست تاریخ را دستی بنویسد.
	TodayFa string `json:"todayFa,omitempty"`
}
