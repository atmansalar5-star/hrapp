package contracts

import "strings"

// EmploymentType یک «نوع استخدام» در سامانهٔ منابع انسانی.
//
// انواع استخدام شاکلهٔ اصلی کارگزینی‌اند: قرارداد، حکم کارگزینی، مزد و فواید تابع
// نوع استخدام‌اند و گزارش‌ها/صدور گروهی حکم بر پایهٔ همین نوع سوار می‌شوند. این فهرست
// «مرجع واحد» است: هم سرور Go و هم هستهٔ Rust و هم رابط کاربری از همین یک منبع
// می‌خوانند تا قاعده در چند جا تکرار نشود.
//
// ترتیب فهرست، ترتیب منطقی نمایش است (از پایدارترین به موقت‌ترین): رسمی نفت، رسمی
// کارگری، مدت معین، قراردادی طویل‌المدت، مدت معین هلدینگ، پیمانکاری (ستادی/پروژه/جاری)
// و در پایان مأموریت‌ها.
type EmploymentType struct {
	// Key شناسهٔ ماشین‌خوان (لاتین، پایدار در داده و API).
	Key string `json:"key"`
	// TitleFa عنوان فارسی نمایشی.
	TitleFa string `json:"titleFa"`
	// GroupFa گروه کارگزینی (رسمی / مدت معین / قراردادی / پیمانکاری / مأموریت).
	GroupFa string `json:"groupFa"`
	// Kind خانوادهٔ مزد: official (رسمی/نفتی) یا contracting (پیمانکاری/ارکان ثالث).
	Kind string `json:"kind"`
	// DecreeFa عنوان حکم کارگزینی متناظر.
	DecreeFa string `json:"decreeFa"`
	// NoteFa توضیح کوتاه برای فرم و راهنما.
	NoteFa string `json:"noteFa,omitempty"`
}

// خانوادهٔ مزد: مبنای انتخاب جدول مزد و ضرایب.
const (
	EmploymentKindOfficial    = "official"
	EmploymentKindContracting = "contracting"
)

// EmploymentTypeCatalog فهرست رسمی انواع استخدام. هر تغییری در اینجا خودکار در
// API، فرم‌ها، حکم گروهی و گزارش‌ها دیده می‌شود.
var EmploymentTypeCatalog = []EmploymentType{
	{Key: "official_oil", TitleFa: "کارمند رسمی نفت", GroupFa: "رسمی", Kind: EmploymentKindOfficial, DecreeFa: "حکم کارگزینی رسمی نفت", NoteFa: "مشمول آیین‌نامهٔ استخدامی شرکت ملی نفت؛ مبنای مزد رسمی."},
	{Key: "official_worker", TitleFa: "کارگر رسمی", GroupFa: "رسمی", Kind: EmploymentKindOfficial, DecreeFa: "حکم کارگزینی کارگری رسمی", NoteFa: "مشمول قانون کار با پایداری استخدام رسمی."},
	{Key: "fixed_1662", TitleFa: "کارمند مدت معین ۱۶۶۲", GroupFa: "مدت معین", Kind: EmploymentKindOfficial, DecreeFa: "حکم کارگزینی مدت معین (مادهٔ ۱۶۶۲)", NoteFa: "قرارداد مدت معین موضوع مادهٔ ۱۶۶۲ قانون کار."},
	{Key: "fixed_holding", TitleFa: "کارمند مدت معین هلدینگ", GroupFa: "مدت معین", Kind: EmploymentKindOfficial, DecreeFa: "حکم کارگزینی مدت معین هلدینگ", NoteFa: "مدت معین در سطح هلدینگ خلیج فارس."},
	{Key: "long_term", TitleFa: "قراردادی طویل‌المدت", GroupFa: "قراردادی", Kind: EmploymentKindOfficial, DecreeFa: "حکم کارگزینی قراردادی طویل‌المدت", NoteFa: "قرارداد بلندمدت غیررسمی با مزایای توافقی."},
	{Key: "contractor_staff", TitleFa: "پیمانکار ستادی", GroupFa: "پیمانکاری", Kind: EmploymentKindContracting, DecreeFa: "حکم کارگزینی پیمانکار ستادی", NoteFa: "نیروی پیمانکاری در ستاد سازمان."},
	{Key: "contractor_project", TitleFa: "پیمانکار پروژه‌ای", GroupFa: "پیمانکاری", Kind: EmploymentKindContracting, DecreeFa: "حکم کارگزینی پیمانکار پروژه‌ای", NoteFa: "نیروی پیمانکاری وابسته به یک پروژه."},
	{Key: "contractor_current", TitleFa: "پیمانکاری جاری", GroupFa: "پیمانکاری", Kind: EmploymentKindContracting, DecreeFa: "حکم کارگزینی پیمانکاری جاری", NoteFa: "پیمانکار جاری با قرارداد در حال اجرا."},
	{Key: "seconded_to", TitleFa: "مأمور به سایر شرکت‌ها", GroupFa: "مأموریت", Kind: EmploymentKindOfficial, DecreeFa: "حکم مأموریت به سایر شرکت‌ها", NoteFa: "کارمند سازمان که به شرکت دیگری مأمور شده است."},
	{Key: "seconded_from", TitleFa: "مأمور از سایر شرکت‌ها", GroupFa: "مأموریت", Kind: EmploymentKindOfficial, DecreeFa: "حکم مأموریت از سایر شرکت‌ها", NoteFa: "کارمند شرکت دیگر که به این سازمان مأمور شده است."},
}

// EmploymentTypes فهرست انواع استخدام.
func EmploymentTypes() []EmploymentType {
	out := make([]EmploymentType, len(EmploymentTypeCatalog))
	copy(out, EmploymentTypeCatalog)
	return out
}

// EmploymentTypeByKey یافتن نوع استخدام با کلید (پذیرش هر دو شکل کلید لاتین و عنوان
// فارسی، تا داده‌های قدیمی و فرم‌ها هر دو کار کنند).
func EmploymentTypeByKey(key string) (EmploymentType, bool) {
	key = strings.TrimSpace(key)
	for _, t := range EmploymentTypeCatalog {
		if t.Key == key || t.TitleFa == key {
			return t, true
		}
	}
	return EmploymentType{}, false
}

// ValidEmploymentType آیا کلید/عنوان یک نوع استخدام معتبر است.
func ValidEmploymentType(key string) bool {
	_, ok := EmploymentTypeByKey(key)
	return ok
}

// EmploymentTypeTitleFa عنوان فارسی یک نوع استخدام؛ برای کلید ناشناخته خودِ ورودی
// برگردانده می‌شود تا نمایش داده‌های قدیمی خراب نشود.
func EmploymentTypeTitleFa(key string) string {
	if t, ok := EmploymentTypeByKey(key); ok {
		return t.TitleFa
	}
	return strings.TrimSpace(key)
}

// EmploymentKeys فهرست کلیدهای معتبر (برای اعتبارسنجی فرم و ساخت گزینه‌ها).
func EmploymentKeys() []string {
	out := make([]string, 0, len(EmploymentTypeCatalog))
	for _, t := range EmploymentTypeCatalog {
		out = append(out, t.Key)
	}
	return out
}

// EmploymentGroups گروه‌های کارگزینی به ترتیب ظهور در فهرست.
func EmploymentGroups() []string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range EmploymentTypeCatalog {
		if !seen[t.GroupFa] {
			seen[t.GroupFa] = true
			out = append(out, t.GroupFa)
		}
	}
	return out
}
