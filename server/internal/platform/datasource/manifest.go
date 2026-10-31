// ساختار داده و راهنمای آماده‌سازی اطلاعات پایه برای بارگذاری.
//
// این فایل «ساختار داده» را از دو منبع می‌سازد و هرگز فهرستی دستی از جدول‌ها نگه نمی‌دارد:
//
//  1. **از خود سامانه (تولیدشده):** هر ماژول رکوردی، شناسهٔ نوع‌های رکوردش را روی پورت
//     قراردادی منتشر می‌کند؛ پس اگر فردا ماژول یا فیلدی اضافه شود، ساختار داده و قالب
//     بارگذاری خودبه‌خود تازه می‌شوند (همان قاعدهٔ «یک مانیفست، همهٔ مشتقات»).
//  2. **لایهٔ گردآوری‌شده (دستی):** برای داده‌های پایه‌ای که ماژول دست‌ساز دارند (کارکنان،
//     قراردادها، تکفل، احکام) یا قالب/قید ویژه‌ای لازم دارند (کد ملی، شبا، تاریخ شمسی،
//     نگاشت کد دستگاه)، اینجا با قالب و مقدار نمونه و منبع قید توضیح داده می‌شوند.
//
// راهنمای فارسی (`GuideFa`) هم از همین ساختار ساخته می‌شود تا راهنما و داده هرگز از هم
// جدا نیفتند.

package datasource

import (
	"fmt"
	"sort"
	"strings"

	"hrms/internal/contracts"
)

// نوع فیلد در ساختار داده (برای سنجش و برای قالب اکسل).
const (
	TypeText   = "text"
	TypeNumber = "number"
	TypeMoney  = "money"
	TypeDate   = "date"
	TypeSelect = "select"
	TypeLookup = "lookup"
	TypeBool   = "bool"
)

// EntityField یک ستون از جدول هدف در ساختار داده.
type EntityField struct {
	Key      string   `json:"key"`
	TitleFa  string   `json:"titleFa"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	FormatFa string   `json:"formatFa,omitempty"`
	ValuesFa []string `json:"valuesFa,omitempty"`
	LookupFa string   `json:"lookupFa,omitempty"`
	Example  string   `json:"example,omitempty"`
	NoteFa   string   `json:"noteFa,omitempty"`
}

// Entity یک جدول هدف برای آمادهسازی و بارگذاری داده.
type Entity struct {
	Key      string        `json:"key"`
	TitleFa  string        `json:"titleFa"`
	Module   string        `json:"module"`
	ModuleFa string        `json:"moduleFa,omitempty"`
	Kind     string        `json:"kind,omitempty"`
	SheetFa  string        `json:"sheetFa"`
	Order    int           `json:"order"`
	GroupFa  string        `json:"groupFa"`
	SourceFa string        `json:"sourceFa"`
	NoteFa   string        `json:"noteFa,omitempty"`
	Fields   []EntityField `json:"fields"`
}

// GroupFa گروهبندی مرحلهای دادههای پایه؛ ترتیب بارگذاری از همین میآید.
const (
	GroupOrg      = "۱. سازمان و ساختار"
	GroupPeople   = "۲. کارکنان و پرونده"
	GroupPayroll  = "۳. حقوق و پرداخت"
	GroupTime     = "۴. حضور، نوبت و کارکرد"
	GroupWorkflow = "۵. گردش کار و اسناد"
	GroupSettings = "۶. تنظیمات و پارامترها"
)

// CuratedEntities موجودیت‌های گردآوری‌شده (ماژول‌های دست‌ساز و داده‌های پایهٔ پرکاربرد).
//
// ترتیب این فهرست، ترتیب توصیه‌شدهٔ بارگذاری است: ساختار سازمانی، سپس کارکنان، سپس
// داده‌های وابسته (قرارداد، تکفل، حساب بانکی، نوبت و تردد).
func CuratedEntities() []Entity {
	return []Entity{
		{
			Key: "companies", TitleFa: "شرکت‌ها و استقرارها", Module: "settings", Kind: "companyProfiles",
			SheetFa: "شرکت‌ها", Order: 1, GroupFa: GroupOrg, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "هر شرکت با کد یکتا، هلدینگ، برند و شناسه‌های قانونی‌اش؛ پیش از هر دادهٔ دیگر بارگذاری شود.",
			Fields: []EntityField{
				{Key: "code", TitleFa: "کد شرکت", Type: TypeText, Required: true, Example: "bipc",
					FormatFa: "حروف لاتین کوچک/بزرگ، رقم و زیرخط؛ ۲ تا ۳۲ نویسه", NoteFa: "یکتا بودن کد شرکت‌ها در سامانه بررسی می‌شود."},
				{Key: "titleFa", TitleFa: "نام شرکت", Type: TypeText, Required: true, Example: "پتروشیمی بندر امام"},
				{Key: "holdingFa", TitleFa: "هلدینگ", Type: TypeText, Example: "هلدینگ خلیج فارس"},
				{Key: "headcount", TitleFa: "تعداد کارکنان", Type: TypeNumber, Example: "4200"},
				{Key: "brandColor", TitleFa: "رنگ برند", Type: TypeText, FormatFa: "#RRGGBB", Example: "#0E7490"},
				{Key: "calendar", TitleFa: "تقویم", Type: TypeSelect, ValuesFa: []string{"شمسی", "میلادی"}, Example: "شمسی"},
				{Key: "fiscalYearStart", TitleFa: "آغاز سال مالی", Type: TypeText, FormatFa: "MM/DD یا YYYY/MM/DD", Example: "01/01"},
				{Key: "nationalId", TitleFa: "شناسهٔ ملی", Type: TypeText, FormatFa: "۱۰ رقم با رقم کنترل معتبر"},
				{Key: "economicCode", TitleFa: "کد اقتصادی", Type: TypeText},
				{Key: "taxFileNumber", TitleFa: "شمارهٔ پروندهٔ مالیاتی", Type: TypeText},
				{Key: "workshopCode", TitleFa: "کد کارگاه تأمین اجتماعی", Type: TypeText},
				{Key: "enabledDepartments", TitleFa: "ادارات فعال", Type: TypeText,
					FormatFa: "کدهای جدا‌شده با ویرگول لاتین", Example: "staff_affairs,planning"},
			},
		},
		{
			Key: "catalogs", TitleFa: "کاتالوگ‌های مرجع (مدیریت، بانک، واحد)", Module: "settings", Kind: "catalogs",
			SheetFa: "کاتالوگ‌ها", Order: 2, GroupFa: GroupOrg, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "اقلام مرجع فرم‌ها: مدیریت‌های سازمانی، بانک‌های عامل و واحدها؛ بانک و مدیریت در فرم‌های وام و پاداش مصرف می‌شوند.",
			Fields: []EntityField{
				{Key: "catalog", TitleFa: "نوع کاتالوگ", Type: TypeSelect, Required: true,
					ValuesFa: []string{"مدیریت", "بانک", "واحد"}, Example: "بانک"},
				{Key: "key", TitleFa: "کلید", Type: TypeText, Required: true, Example: "melli", FormatFa: "حروف لاتین کوچک؛ یکتا در همان کاتالوگ"},
				{Key: "titleFa", TitleFa: "عنوان", Type: TypeText, Required: true, Example: "بانک ملی"},
				{Key: "management", TitleFa: "مدیریت بالادست (برای واحد)", Type: TypeLookup, LookupFa: "برگهٔ کاتالوگ‌ها، ستون کلید با نوع «مدیریت»"},
			},
		},
		{
			Key: "orgUnits", TitleFa: "واحدهای سازمانی و پست‌ها", Module: "orgstructure", Kind: "units",
			SheetFa: "واحدهای سازمانی", Order: 3, GroupFa: GroupOrg, SourceFa: "سامانه (از شناسهٔ ماژول)",
			NoteFa: "درخت واحدها و شناسنامهٔ پست‌ها؛ کلید واحد در پروندهٔ کارمند استفاده می‌شود.",
		},
		{
			Key: "employees", TitleFa: "کارکنان (پروندهٔ پایه)", Module: "employee",
			SheetFa: "کارکنان", Order: 4, GroupFa: GroupPeople, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "پروندهٔ پایهٔ هر کارمند: کد پرسنلی یکتا، مشخصات هویتی، واحد، پست، نوع استخدام و مزد پایه. بقیهٔ داده‌ها به کد پرسنلی وصل می‌شوند.",
			Fields: []EntityField{
				{Key: "id", TitleFa: "کد پرسنلی", Type: TypeText, Required: true, Example: "EMP-1001",
					NoteFa: "کلید همهٔ جدول‌های دیگر است؛ در سامانه یکتاست."},
				{Key: "fullName", TitleFa: "نام و نام خانوادگی", Type: TypeText, Required: true},
				{Key: "nationalId", TitleFa: "کد ملی", Type: TypeText, Required: true, FormatFa: "۱۰ رقم با رقم کنترل معتبر"},
				{Key: "departmentId", TitleFa: "کد واحد سازمانی", Type: TypeLookup, LookupFa: "برگهٔ واحدهای سازمانی، ستون کد"},
				{Key: "position", TitleFa: "عنوان شغل", Type: TypeText},
				{Key: "contractType", TitleFa: "نوع قرارداد", Type: TypeSelect,
					ValuesFa: []string{"تمام‌وقت", "پاره‌وقت", "پیمانکاری"}},
				{Key: "employmentType", TitleFa: "نوع استخدام", Type: TypeText, Example: "رسمی"},
				{Key: "status", TitleFa: "وضعیت", Type: TypeSelect, ValuesFa: []string{"شاغل", "منفصل"}, Example: "شاغل"},
				{Key: "hireDate", TitleFa: "تاریخ استخدام", Type: TypeDate, FormatFa: "YYYY/MM/DD شمسی", Example: "1398/04/12"},
				{Key: "baseSalary", TitleFa: "مزد پایه (ریال)", Type: TypeMoney, FormatFa: "عدد لاتین یا فارسی بدون جداکننده", Example: "520000000"},
				{Key: "leaveBalance", TitleFa: "مانده مرخصی (روز)", Type: TypeNumber, Example: "12"},
				{Key: "phone", TitleFa: "تلفن همراه", Type: TypeText, FormatFa: "۰۹xxxxxxxxx"},
				{Key: "email", TitleFa: "رایانامه", Type: TypeText},
				{Key: "managerName", TitleFa: "مدیر مستقیم", Type: TypeText},
			},
		},
		{
			Key: "bankAccounts", TitleFa: "حساب‌های بانکی کارکنان", Module: "payroll", Kind: "bankAccounts",
			SheetFa: "حساب‌های بانکی", Order: 5, GroupFa: GroupPayroll, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "مبنای فایل پرداخت بانکی؛ کارمندی که حساب فعال ندارد در گزارش «بدون حساب» می‌آید.",
			Fields: []EntityField{
				{Key: "employeeId", TitleFa: "کد پرسنلی", Type: TypeLookup, Required: true, LookupFa: "برگهٔ کارکنان، ستون کد پرسنلی"},
				{Key: "bankName", TitleFa: "بانک", Type: TypeLookup, Required: true, LookupFa: "برگهٔ کاتالوگ‌ها، اقلام نوع «بانک»"},
				{Key: "iban", TitleFa: "شمارهٔ شبا", Type: TypeText, Required: true,
					FormatFa: "IR + ۲۴ رقم (رقم کنترل استاندارد)", Example: "IR820540102680020817909002"},
				{Key: "accountNo", TitleFa: "شمارهٔ حساب", Type: TypeText},
				{Key: "holderName", TitleFa: "نام صاحب حساب", Type: TypeText},
				{Key: "nationalId", TitleFa: "کد ملی صاحب حساب", Type: TypeText, FormatFa: "۱۰ رقم با رقم کنترل"},
			},
		},
		{
			Key: "contracts", TitleFa: "قراردادهای کار", Module: "contracts",
			SheetFa: "قراردادها", Order: 6, GroupFa: GroupWorkflow, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "هر قرارداد به یک کارمند و نوع قرارداد (مدت موقت/دائم) وصل است؛ سند چاپی و احکام از همین می‌آید.",
			Fields: []EntityField{
				{Key: "employeeId", TitleFa: "کد پرسنلی", Type: TypeLookup, Required: true, LookupFa: "برگهٔ کارکنان"},
				{Key: "kind", TitleFa: "نوع قرارداد", Type: TypeSelect, Required: true,
					ValuesFa: []string{"مدت موقت", "مدت دائم"}, Example: "مدت موقت"},
				{Key: "startDate", TitleFa: "تاریخ شروع", Type: TypeDate, Required: true, FormatFa: "YYYY/MM/DD"},
				{Key: "endDate", TitleFa: "تاریخ پایان", Type: TypeDate, FormatFa: "YYYY/MM/DD (برای مدت موقت الزامی)"},
				{Key: "monthlySalary", TitleFa: "مزد ماهانه (ریال)", Type: TypeMoney, FormatFa: "عدد بدون جداکننده"},
				{Key: "serviceDays", TitleFa: "روزهای سابقه", Type: TypeNumber},
			},
		},
		{
			Key: "dependents", TitleFa: "افراد تحت تکفل", Module: "staff_services",
			SheetFa: "افراد تحت تکفل", Order: 7, GroupFa: GroupPeople, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "وابستگان هر کارمند برای پوشش بیمه و مدارک تکفل.",
			Fields: []EntityField{
				{Key: "employeeId", TitleFa: "کد پرسنلی", Type: TypeLookup, Required: true, LookupFa: "برگهٔ کارکنان"},
				{Key: "fullName", TitleFa: "نام و نام خانوادگی", Type: TypeText, Required: true},
				{Key: "relationFa", TitleFa: "نسبت", Type: TypeSelect, ValuesFa: []string{"همسر", "فرزند", "پدر", "مادر"}},
				{Key: "nationalId", TitleFa: "کد ملی", Type: TypeText, FormatFa: "۱۰ رقم با رقم کنترل"},
				{Key: "birthDate", TitleFa: "تاریخ تولد", Type: TypeDate, FormatFa: "YYYY/MM/DD"},
				{Key: "insuranceCovered", TitleFa: "زیر پوشش بیمه", Type: TypeBool, ValuesFa: []string{"بله", "خیر"}},
			},
		},
		{
			Key: "salaryDecrees", TitleFa: "احکام حقوق", Module: "salary_admin",
			SheetFa: "احکام حقوق", Order: 8, GroupFa: GroupPayroll, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "حکم حقوق هر کارمند با تاریخ اجرا؛ هر تغییر، حکم تازه می‌سازد و حکم قبلی بازنویسی نمی‌شود.",
			Fields: []EntityField{
				{Key: "employeeId", TitleFa: "کد پرسنلی", Type: TypeLookup, Required: true, LookupFa: "برگهٔ کارکنان"},
				{Key: "decreeNo", TitleFa: "شمارهٔ حکم", Type: TypeText, Required: true},
				{Key: "issuedAt", TitleFa: "تاریخ صدور", Type: TypeDate, FormatFa: "YYYY/MM/DD"},
				{Key: "effectiveFrom", TitleFa: "تاریخ اجرا", Type: TypeDate, Required: true, FormatFa: "YYYY/MM/DD"},
				{Key: "baseSalary", TitleFa: "مزد پایه (ریال)", Type: TypeMoney},
				{Key: "employmentType", TitleFa: "نوع استخدام", Type: TypeText},
			},
		},
		{
			Key: "payrollParams", TitleFa: "پارامترها و ضرائب حقوق", Module: "payroll", Kind: "payParams",
			SheetFa: "پارامترهای حقوق", Order: 9, GroupFa: GroupSettings, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "کلیدهای پارامتری مثل نرخ بیمه و معافیت مالیات؛ مقدار همین‌جا تعیین می‌شود و محاسبه از آن می‌خواند.",
			Fields: []EntityField{
				{Key: "key", TitleFa: "کلید پارامتر", Type: TypeText, Required: true, Example: "insurance.employeeRate",
					FormatFa: "حروف لاتین و نقطه (کلید نقطه‌دار)", NoteFa: "فهرست کلیدهای رایج در صفحهٔ «پارامترهای سیستم» دیده می‌شود."},
				{Key: "value", TitleFa: "مقدار", Type: TypeText, Required: true, Example: "0.07"},
				{Key: "unitFa", TitleFa: "واحد", Type: TypeText, Example: "نسبت"},
				{Key: "groupFa", TitleFa: "گروه", Type: TypeText, Example: "بیمه"},
				{Key: "effectiveFrom", TitleFa: "تاریخ اجرا", Type: TypeDate, FormatFa: "YYYY/MM/DD"},
			},
		},
		{
			Key: "shiftPatterns", TitleFa: "الگوهای چرخهٔ نوبت", Module: "attendance", Kind: "shiftPatterns",
			SheetFa: "الگوهای نوبت", Order: 10, GroupFa: GroupTime, SourceFa: "سامانه (از شناسهٔ ماژول)",
			NoteFa: "ترتیب شیفت‌های چرخه؛ کلیدهای مجاز: صبح، عصر، شب، استراحت (با ویرگول).",
		},
		{
			Key: "shiftAssignments", TitleFa: "تخصیص نوبت به کارکنان", Module: "attendance", Kind: "shiftAssignments",
			SheetFa: "تخصیص نوبت", Order: 11, GroupFa: GroupTime, SourceFa: "سامانه (از شناسهٔ ماژول)",
			NoteFa: "تاریخ شروع = مبنای چرخه؛ تخصیص باید تأیید (مصوب) شود تا در برنامهٔ نوبت و کارکرد اثر کند.",
		},
		{
			Key: "deviceMap", TitleFa: "نگاشت کد دستگاه تردد", Module: "attendance",
			SheetFa: "نگاشت دستگاه", Order: 12, GroupFa: GroupTime, SourceFa: "دستی (گردآوری‌شده)",
			NoteFa: "کد کارت/کد دستگاه در فایل تردد را به کد پرسنلی وصل می‌کند؛ بدون این نگاشت، تردد بی‌نگاشت در گزارش مغایرت می‌آید.",
			Fields: []EntityField{
				{Key: "deviceCode", TitleFa: "کد دستگاه", Type: TypeText, Required: true, Example: "1001"},
				{Key: "employeeId", TitleFa: "کد پرسنلی", Type: TypeLookup, Required: true, LookupFa: "برگهٔ کارکنان"},
				{Key: "note", TitleFa: "یادداشت", Type: TypeText},
			},
		},
	}
}

// Manifest ادغام دو منبع ساختار داده: گردآوری‌شده‌ها اول (با دسته‌بندی مرحله‌ای) و سپس همهٔ
// نوع‌های رکورد سامانه به ترتیب ماژول. موجودیت‌های تکراری (همان ماژول+نوع) یکی می‌شوند و
// نسخهٔ گردآوری‌شده برنده است (چون قالب و قید بیشتری دارد).
func Manifest(schemas []contracts.RecordSchema, moduleTitles map[string]string) []Entity {
	seen := map[string]bool{}
	out := []Entity{}

	for _, curated := range CuratedEntities() {
		if len(curated.Fields) == 0 {
			// داده‌های دست‌ساز (کارکنان، قرارداد، احکام) هم فیلدهایشان گردآوری‌شده است؛
			// ولی موجودیت‌هایی که به شناسهٔ ماژول وصل‌اند، از خود سامانه پر می‌شوند.
			if fields, ok := fieldsFromSchema(schemas, curated.Module, curated.Kind); ok {
				curated.Fields = fields
				curated.SourceFa = "سامانه (از شناسهٔ ماژول)"
			}
		}
		if len(curated.Fields) == 0 {
			continue
		}
		key := curated.Module + "/" + curated.Kind
		if key != "/" {
			seen[key] = true
		}
		seen[curated.Key] = true
		out = append(out, curated)
	}

	generated := []Entity{}
	for _, schema := range schemas {
		moduleTitle := strings.TrimSpace(moduleTitles[schema.Module])
		if moduleTitle == "" {
			moduleTitle = schema.ModuleTitleFa
		}
		for index, kind := range schema.Kinds {
			key := schema.Module + "/" + kind.Key
			if seen[key] || seen[kind.Key] {
				continue
			}
			fields := make([]EntityField, 0, len(kind.Fields))
			for _, field := range kind.Fields {
				fields = append(fields, EntityField{
					Key: field.Key, TitleFa: field.LabelFa, Type: field.Type,
					Required: field.Required, FormatFa: formatOf(field.Type),
				})
			}
			if len(fields) == 0 {
				continue
			}
			generated = append(generated, Entity{
				Key: kind.Key, TitleFa: kind.TitleFa, Module: schema.Module, ModuleFa: moduleTitle,
				Kind: kind.Key, SheetFa: kind.TitleFa, Order: 100 + index,
				GroupFa: GroupSettings, SourceFa: "سامانه (از شناسهٔ ماژول)",
				NoteFa: "ساختار این جدول از شناسهٔ خودِ ماژول آمده است؛ ستون‌ها همان فیلدهای اعلامی سرورند.",
				Fields: fields,
			})
		}
	}
	sort.SliceStable(generated, func(i, j int) bool {
		if generated[i].ModuleFa != generated[j].ModuleFa {
			return generated[i].ModuleFa < generated[j].ModuleFa
		}
		return generated[i].Order < generated[j].Order
	})
	return append(out, generated...)
}

// fieldsFromSchema فیلدهای یک نوع رکورد را از شناسهٔ ماژول می‌خواند.
func fieldsFromSchema(schemas []contracts.RecordSchema, module, kind string) ([]EntityField, bool) {
	for _, schema := range schemas {
		if schema.Module != module {
			continue
		}
		for _, item := range schema.Kinds {
			if item.Key != kind {
				continue
			}
			fields := make([]EntityField, 0, len(item.Fields))
			for _, field := range item.Fields {
				fields = append(fields, EntityField{
					Key: field.Key, TitleFa: field.LabelFa, Type: field.Type,
					Required: field.Required, FormatFa: formatOf(field.Type),
				})
			}
			return fields, len(fields) > 0
		}
	}
	return nil, false
}

// formatOf قالب پیشنهادی هر نوع فیلد.
func formatOf(fieldType string) string {
	switch fieldType {
	case TypeDate:
		return "YYYY/MM/DD شمسی"
	case TypeMoney:
		return "عدد ریالی بدون جداکننده"
	case TypeNumber:
		return "عدد (رقم لاتین یا فارسی)"
	case TypeSelect:
		return "یکی از مقادیر مجاز"
	}
	return ""
}

// GuideFa راهنمای فارسی آماده‌سازی داده را از ساختار داده می‌سازد.
func GuideFa(entities []Entity, orgTitle string) string {
	var b strings.Builder
	b.WriteString("# راهنمای آماده‌سازی اطلاعات پایه و بارگذاری از فایل اکسل\n\n")
	fmt.Fprintf(&b, "این راهنما از **ساختار دادهٔ خود سامانه** ساخته شده است؛ پس با افزودن هر ماژول یا فیلد تازه، خودبه‌خود تازه می‌شود. استقرار فعال: **%s**.\n\n", orgTitle)
	b.WriteString("## قاعده‌های عمومی فایل\n\n")
	b.WriteString("- قالب فایل: **Excel (xlsx)** یا **CSV**. خط نخست هر برگه، **سرصفحه** و نام ستون‌ها است.\n")
	b.WriteString("- یک برگه برای هر جدول؛ نام برگه در قالب پیشنهادی آمده و لازم نیست عوض شود.\n")
	b.WriteString("- اعداد: رقم فارسی یا لاتین، **بدون جداکنندهٔ هزارگان** (۵۲٬۰۰۰٬۰۰۰ ❌ ، 52000000 ✅).\n")
	b.WriteString("- تاریخ‌ها: **شمسی** به قالب `YYYY/MM/DD` (نمونه: `1398/04/12`).\n")
	b.WriteString("- سلول‌های خالی، «ندارد» تلقی می‌شوند؛ فیلدهای **الزامی** نباید خالی بمانند.\n")
	b.WriteString("- کدهای کلیدی (کد پرسنلی، کد شرکت، کد واحد) باید **یکتا** و بدون فاصلهٔ اضافی باشند.\n")
	b.WriteString("- ترتیب بارگذاری از بالا به پایین همین سند است (اول ساختار، بعد کارکنان، بعد داده‌های وابسته).\n\n")

	byGroup := map[string][]Entity{}
	groups := []string{}
	for _, entity := range entities {
		group := entity.GroupFa
		if group == "" {
			group = GroupSettings
		}
		if _, ok := byGroup[group]; !ok {
			groups = append(groups, group)
			byGroup[group] = []Entity{}
		}
		byGroup[group] = append(byGroup[group], entity)
	}
	sort.Strings(groups)

	step := 0
	for _, group := range groups {
		fmt.Fprintf(&b, "## %s\n\n", group)
		for _, entity := range byGroup[group] {
			step++
			fmt.Fprintf(&b, "### %d) %s\n\n", step, entity.TitleFa)
			fmt.Fprintf(&b, "- **برگهٔ پیشنهادی:** `%s`", entity.SheetFa)
			if entity.ModuleFa != "" {
				fmt.Fprintf(&b, " — ماژول: %s", entity.ModuleFa)
			}
			fmt.Fprintf(&b, "\n")
			if entity.Module != "" {
				fmt.Fprintf(&b, "- **جای بارگذاری در سامانه:** `%s`", entity.Module)
				if entity.Kind != "" {
					fmt.Fprintf(&b, " (نوع رکورد `%s`)", entity.Kind)
				}
				fmt.Fprintf(&b, "\n")
			}
			if entity.NoteFa != "" {
				fmt.Fprintf(&b, "- **نکته:** %s\n", entity.NoteFa)
			}
			b.WriteString("\n| ستون | عنوان | نوع | الزامی | قالب/مقادیر | نمونه |\n|---|---|---|---|---|---|\n")
			for _, field := range entity.Fields {
				required := "خیر"
				if field.Required {
					required = "**بله**"
				}
				details := field.FormatFa
				if len(field.ValuesFa) > 0 {
					details = strings.TrimSpace(details + " " + strings.Join(field.ValuesFa, " | "))
				}
				if field.LookupFa != "" {
					details = strings.TrimSpace(details + " ← " + field.LookupFa)
				}
				fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s |\n",
					field.Key, field.TitleFa, typeFa(field.Type), required, details, field.Example)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("## پس از تکمیل فایل\n\n")
	b.WriteString("1. فایل را از صفحهٔ **«تبادل داده و قالب‌ها»** بارگذاری کنید.\n")
	b.WriteString("2. سامانه پیوند ستون‌های فایل با ساختار جدول را نشان می‌دهد؛ ستون‌های ناشناخته نادیده می‌مانند.\n")
	b.WriteString("3. پیش از نوشتن در پایگاه‌داده، **گزارش اعتبارسنجی** (ردیف، ستون و دلیل) نمایش داده می‌شود.\n")
	b.WriteString("4. نوشتن نهایی **تراکنشی** انجام می‌شود: یا همهٔ ردیف‌های سالم می‌نشینند یا هیچ‌کدام؛ ردیف‌های ایراددار برای اصلاح گزارش می‌شوند.\n")
	return b.String()
}

// typeFa برچسب فارسی نوع فیلد.
func typeFa(fieldType string) string {
	switch fieldType {
	case TypeText:
		return "متن"
	case TypeNumber:
		return "عدد"
	case TypeMoney:
		return "مبلغ (ریال)"
	case TypeDate:
		return "تاریخ شمسی"
	case TypeSelect:
		return "انتخابی"
	case TypeLookup:
		return "ارجاع به جدول دیگر"
	case TypeBool:
		return "بله/خیر"
	}
	return fieldType
}
