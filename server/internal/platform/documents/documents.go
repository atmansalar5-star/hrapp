// Package documents اسناد چاپی فارسی (قرارداد، حکم حقوق، گواهی اشتغال).
//
// سند به‌صورت HTML راست‌چین با شیوه‌نامهٔ چاپ A4 ساخته می‌شود و مرورگر آن را به PDF
// تبدیل می‌کند. این انتخاب عمدی است: شکل‌دهی حروف فارسی (اتصال حروف و راست‌به‌چپ) در
// موتور چاپ مرورگر درست انجام می‌شود، درحالی‌که تولید مستقیم PDF بدون موتور متن پیچیده
// حروف فارسی را جدا و بی‌اتصال نشان می‌دهد. پس یک قالب، دو مسیر: چاپ از مرورگر برای
// کاربر و رندر دسته‌ای با مرورگر بی‌سر برای بایگانی.
//
// قالب‌ها با html/template نوشته شده‌اند تا مقادیر داده خودکار escape شوند؛ نام کارمند
// دادهٔ کاربر است و هرگز نباید در سند به‌صورت کد تفسیر شود.
package documents

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"html/template"
	"strings"
)

// Kind نوع سند.
type Kind string

// انواع سند پشتیبانی‌شده و ماژول مالک داده‌شان (برای بررسی دسترسی).
const (
	KindContract              Kind = "contract"
	KindSalaryDecree          Kind = "salary_decree"
	KindEmploymentCertificate Kind = "employment_certificate"
	KindPayslip               Kind = "payslip"
	// KindDegreeBasis فرم اداری «تعیین مدرک ملاک حقوق»: کدام مدرک، با چه ارتباطی و به
	// استناد کدام صورت‌جلسهٔ شورای تعدیل مدرک مبنای گروه شغلی و پرداخت است.
	KindDegreeBasis Kind = "degree_basis"
)

// ModuleOf ماژول مالک سند؛ دسترسی نقش روی همان ماژول سنجیده می‌شود.
func ModuleOf(kind Kind) (string, bool) {
	switch kind {
	case KindContract:
		return "contracts", true
	case KindSalaryDecree:
		return "salary_admin", true
	case KindEmploymentCertificate:
		return "employee", true
	case KindPayslip:
		return "payroll", true
	case KindDegreeBasis:
		return "archive", true
	default:
		return "", false
	}
}

// TitleFa عنوان فارسی سند.
func TitleFa(kind Kind) string {
	switch kind {
	case KindContract:
		return "قرارداد کار"
	case KindSalaryDecree:
		return "حکم حقوق"
	case KindEmploymentCertificate:
		return "گواهی اشتغال به کار"
	case KindPayslip:
		return "فیش حقوقی"
	case KindDegreeBasis:
		return "فرم تعیین مدرک ملاک حقوق"
	default:
		return string(kind)
	}
}

// Company مشخصات شرکت در سرصفحهٔ سند.
type Company struct {
	TitleFa           string
	HoldingFa         string
	ShortCode         string
	CurrencyFa        string
	WorkWeekFa        string
	FiscalYearStartFa string
	// LogoURL آدرس تصویر نشان شرکت روی همان مبدأ رابط وب (خالی = بدون تصویر).
	LogoURL string
	// BrandColor رنگ اصلی برند برای نوار سرصفحه (خالی = رنگ پیش‌فرض).
	BrandColor string
}

// Employee اطلاعات کارمند در سند.
//
// چرا اطلاعات پایه و تحصیلی هم اینجاست: حکم کارگزینی، گواهی اشتغال و قرارداد کار،
// اسنادی هستند که مراجع بیرونی (بیمه، مالیات، بانک، سفارت) از آن‌ها نشانی، شمارهٔ تماس
// و مدرک تحصیلی را می‌خواهند. این مقادیر از پروندهٔ کارمند و پروندهٔ تحصیلی خوانده
// می‌شوند، نه با تایپ دوباره در هر سند.
type Employee struct {
	ID             string
	FullName       string
	NationalID     string
	DepartmentName string
	Position       string
	HireDate       string
	ContractType   string
	BaseSalary     int64

	// اطلاعات پایهٔ پرونده (خالی یعنی در پرونده ثبت نشده و روی سند چاپ نمی‌شود).
	Mobile     string
	HomePhone  string
	Phone      string
	PostalCode string
	Address    string
	Email      string

	// مشخصات تحصیلی از پروندهٔ تحصیلی بایگانی: بالاترین مدرک و رشتهٔ آن.
	EducationLevelFa string
	EducationField   string
	EducationOrient  string
	EducationUniver  string
	EducationDate    string
	EducationText    string // «کارشناسی ارشد — حسابداری» برای نمایش یک‌خطی
	DegreeBasisFa    string // مدرک ملاک حقوق (اگر شورای تعدیل مدرک تصویب کرده باشد)
	HasEducationInfo bool
}

// ContractInfo خلاصهٔ قرارداد مرجع سند.
type ContractInfo struct {
	ID            string
	KindFa        string
	StatusFa      string
	StartDate     string
	EndDate       string
	MonthlySalary int64
	ServiceDays   int
	ServiceYears  int
}

// Param یک پارامتر مبنای محاسبه (از استقرار شرکت فعال، نه عدد ساختگی).
type Param struct {
	// Key کلید ماشین‌خوان پارامتر برای ارجاع در قالب متن (مثل payroll.monthlyHours).
	Key     string
	TitleFa string
	Value   string
	UnitFa  string
	GroupFa string
}

// PayslipInfo جزئیات محاسبهٔ فیش حقوقی برای سند فیش.
type PayslipInfo struct {
	Period            string
	Gross             int64
	InsuranceEmployee int64
	InsuranceEmployer int64
	Tax               int64
	Net               int64
	// MazdBase مزد مبنا: جمع اجزای ثابت مزد کارمند (از حکم یا قرارداد). مبنای اضافه‌کاری و
	// فوق‌العاده‌های درصدی است، پس روی فیش چاپی هم می‌آید.
	MazdBase int64
	// WageSourceFa منبع اجزای مزد («از حکم کارگزینی DEC-...») تا خوانندهٔ فیش بداند عدد از
	// کدام سند آمده است.
	WageSourceFa string
	// Elements تفکیک اقلام حقوق؛ همان محاسبه‌ای را نشان می‌دهد که شبیه‌ساز فیش می‌دهد.
	Elements []PayslipElement
	// PayGap تفاوت تطبیق ماهانهٔ این کارمند (از حکم کارگزینی) و تاریخ شروع آن.
	//
	// چرا جدا از Elements: تفاوت تطبیق قلم پارامتریک حقوق نیست، مبلغی است که روی حکم
	// کارگزینی ثبت می‌شود؛ فیش باید آن را صریح نشان دهد تا کارمند بداند این مبلغ جذب‌شدنی
	// است و بخشی از مزد مبنا نیست.
	PayGap      int64
	PayGapSince string
	// PayGapFa همان مبلغ با جداکنندهٔ هزارگان برای چاپ (خالی = بدون تفاوت تطبیق).
	PayGapFa string
	// VeteranGroupFa گروه ایثارگری اعمال‌شده در همین فیش (خالی = بدون پروندهٔ تأییدشده).
	VeteranGroupFa string
	// VeteranTaxExemptFa و VeteranPensionExemptFa معافیت مالیاتی و معافیت کسور ایثارگری
	// با جداکنندهٔ هزارگان برای چاپ (خالی = اعمال نشده).
	VeteranTaxExemptFa     string
	VeteranPensionExemptFa string
}

// PayslipElement یک سطر از تفکیک اقلام فیش حقوقی.
type PayslipElement struct {
	TitleFa    string
	CategoryFa string
	AmountFa   string
	DetailFa   string
}

// StructureRow یک سطر از «اجزای مزد و فوق‌العاده‌ها» در حکم کارگزینی و قرارداد کار.
//
// GroupFa گروه سطر است («اجزای مزد مبنا»، «فوق‌العاده‌های اقلیمی»، …) و Total یعنی این سطر
// جمع گروه است؛ همین دو، جدول چاپی را خوانا می‌کند: خواننده می‌بیند مزد مبنا از چه ساخته
// شده و کدام اقلام روی آن سوار می‌شوند.
type StructureRow struct {
	TitleFa  string
	KindFa   string
	DetailFa string
	AmountFa string
	GroupFa  string
	Total    bool
}

// FormField یک سطر «فرم اداری» روی کاغذ: برچسب و مقدار.
//
// چرا سازوکار عمومی: سازمان فرم‌های اداری گوناگونی دارد (تعیین مدرک ملاک حقوق، تعدیل
// مدرک ایثارگران، صورت‌جلسهٔ شورا) و افزودن یک قالب HTML تازه برای هر فرم، نگهداشت را
// سنگین می‌کند. با این ساختار، هر ماژول فرم کاغذی خودش را از دادهٔ خودش می‌سازد.
type FormField struct {
	LabelFa string
	ValueFa string
}

// FormTable یک جدول «فرم اداری» روی کاغذ (سرستون‌ها و سطرها).
type FormTable struct {
	TitleFa string
	Head    []string
	Rows    [][]string
	NoteFa  string
}

// Block یک بلوک نمایشی سند چاپی (جدول مشخصات، سابقهٔ تحصیلی، امضاها و...).
//
// چرا بلوک‌بندی: هر سازمان نسخهٔ خودش از حکم و گواهی را می‌خواهد؛ بعضی بلوک‌ها را لازم
// ندارد (مثلاً نشانی روی گواهی اشتغال) و بعضی را با عنوان دیگری می‌خواهد. با این کاتالوگ،
// پنل تنظیمات می‌تواند بلوک‌ها را روشن/خاموش و عنوانشان را عوض کند، بدون تغییر کد و
// بدون قالب HTML جدا برای هر سند.
type Block struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	// Default یعنی این بلوک در چیدمان پیش‌فرض همهٔ اسنادی که آن را پشتیبانی می‌کنند
	// روشن است.
	Default bool `json:"default"`
}

// blockCatalog فهرست بلوک‌های شناخته‌شدهٔ سند چاپی.
var blockCatalog = []Block{
	{Key: "employeeCard", TitleFa: "کارت مشخصات کارمند (نام، کد پرسنلی، واحد، پست)", Default: true},
	{Key: "employeeInfo", TitleFa: "جدول مشخصات کارمند (کد ملی، تاریخ استخدام، نوع همکاری)", Default: true},
	{Key: "contactInfo", TitleFa: "شماره‌های تماس، کد پستی و نشانی محل سکونت", Default: true},
	{Key: "education", TitleFa: "سابقهٔ تحصیلی (مدرک، رشته، دانشگاه، مدرک ملاک حقوق)", Default: true},
	{Key: "contract", TitleFa: "مشخصات قرارداد کار", Default: true},
	{Key: "params", TitleFa: "مبنای محاسبهٔ سامانه", Default: true},
	{Key: "payslipElements", TitleFa: "تفکیک اقلام حقوق", Default: true},
	{Key: "structure", TitleFa: "اجزای مزد مبنا و فوق‌العاده‌ها", Default: true},
	{Key: "baseInfoForm", TitleFa: "مشخصات پایهٔ کارمند (بلوک فرم)", Default: true},
	{Key: "form", TitleFa: "فرم اداری سند (فیلدها و جدول‌های اختصاصی)", Default: true},
	{Key: "body", TitleFa: "شرح / متن سند", Default: true},
	{Key: "note", TitleFa: "یادداشت پای سند", Default: true},
	{Key: "signatures", TitleFa: "محل امضاها", Default: true},
	{Key: "verify", TitleFa: "کد رهگیری و QR اصالت در پاورقی", Default: true},
}

// Blocks کاتالوگ بلوک‌ها (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func Blocks() []Block {
	out := make([]Block, len(blockCatalog))
	copy(out, blockCatalog)
	return out
}

// BlockByKey یافتن بلوک با کلید.
func BlockByKey(key string) (Block, bool) {
	value := strings.TrimSpace(key)
	for _, block := range blockCatalog {
		if block.Key == value {
			return block, true
		}
	}
	return Block{}, false
}

// BlockKeys کلیدهای مجاز بلوک (برای اعتبارسنجی پارامتر تنظیمات).
func BlockKeys() []string {
	out := make([]string, 0, len(blockCatalog))
	for _, block := range blockCatalog {
		out = append(out, block.Key)
	}
	return out
}

// DocCatalog یک نوع سند چاپی به‌همراه بلوک‌های پشتیبانی‌شده‌اش (برای پنل تنظیمات).
type DocCatalog struct {
	Kind    Kind       `json:"kind"`
	TitleFa string     `json:"titleFa"`
	Blocks  []Block    `json:"blocks"`
	Texts   []TextSlot `json:"texts"`
}

// TextSlot یک «متن آزاد» سند چاپی (پاراگراف متن، یادداشت پای سند).
//
// چرا جدا از بلوک: بلوک یعنی «این جدول/بخش روی کاغذ بیاید یا نه»، ولی پاراگراف متن
// نوشتنی است؛ متن گواهی اشتغال یا بند پایانی حکم در هر سازمانی متفاوت است و باید
// همانجا در پنل پارامتریک ویرایش شود. کلید هر جای متن همان کلید بلوکش است تا
// خاموش‌کردن بلوک، متنش را هم از سند بردارد.
type TextSlot struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	HintFa  string `json:"hintFa"`
}

// textSlots جای‌های متن قابل ویرایش در اسناد چاپی.
var textSlots = []TextSlot{
	{
		Key: "body", TitleFa: "پاراگراف متن سند",
		HintFa: "هر خط یک پاراگراف چاپ می‌شود و جای‌نگهدارها (مثل {{employee.fullName}}) با دادهٔ واقعی پر می‌شوند؛ خالی یعنی متن پیش‌فرض سامانه.",
	},
	{
		Key: "note", TitleFa: "یادداشت پای سند",
		HintFa: "یک بند کوتاه در پای برگه؛ خالی یعنی یادداشت پیش‌فرض سامانه.",
	},
}

// TextSlots همهٔ جای‌های متن قابل ویرایش (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func TextSlots() []TextSlot {
	out := make([]TextSlot, len(textSlots))
	copy(out, textSlots)
	return out
}

// KindTextSlots جای‌های متنی که یک نوع سند پشتیبانی می‌کند.
func KindTextSlots(kind Kind) []TextSlot {
	supported := map[string]bool{}
	for _, key := range KindBlocks(kind) {
		supported[key] = true
	}
	out := []TextSlot{}
	for _, slot := range textSlots {
		if supported[slot.Key] {
			out = append(out, slot)
		}
	}
	return out
}

// docKindOrder ترتیب نمایش اسناد در پنل تنظیمات.
var docKindOrder = []Kind{KindContract, KindSalaryDecree, KindEmploymentCertificate, KindPayslip, KindDegreeBasis}

// Catalog فهرست اسناد چاپی و بلوک‌های هر کدام.
//
// چرا اینجا و نه در ماژول تنظیمات: بلوک‌ها دارایی موتور اسناد است؛ پنل تنظیمات فقط
// همین کاتالوگ را نمایش می‌دهد و پارامتر ذخیره می‌کند، پس با افزودن بلوک تازه در موتور،
// پنل خودکار آن را می‌بیند.
func Catalog() []DocCatalog {
	out := make([]DocCatalog, 0, len(docKindOrder))
	for _, kind := range docKindOrder {
		blocks := make([]Block, 0)
		for _, key := range KindBlocks(kind) {
			if block, ok := BlockByKey(key); ok {
				blocks = append(blocks, block)
			}
		}
		out = append(out, DocCatalog{Kind: kind, TitleFa: TitleFa(kind), Blocks: blocks, Texts: KindTextSlots(kind)})
	}
	return out
}

// KindBlocks بلوک‌هایی که هر نوع سند پشتیبانی می‌کند؛ کلیدهایی که اینجا نیستند در آن سند
// معنا ندارند (مثلاً «تفکیک اقلام حقوق» در قرارداد).
var kindBlocks = map[Kind][]string{
	KindContract:              {"employeeCard", "employeeInfo", "contactInfo", "education", "contract", "params", "structure", "baseInfoForm", "body", "note", "signatures", "verify"},
	KindSalaryDecree:          {"employeeCard", "employeeInfo", "contactInfo", "education", "contract", "params", "structure", "baseInfoForm", "body", "note", "signatures", "verify"},
	KindEmploymentCertificate: {"employeeCard", "employeeInfo", "contactInfo", "education", "contract", "baseInfoForm", "body", "note", "signatures", "verify"},
	// فیش حقوقی: اطلاعات تماس، نشانی و تحصیلات روی فیش نمی‌آید (تصمیم کارفرما)؛ فقط
	// مشخصات پایهٔ شغلی و اقلام حقوق.
	KindPayslip:     {"employeeCard", "employeeInfo", "payslipElements", "body", "note", "signatures", "verify"},
	KindDegreeBasis: {"employeeCard", "employeeInfo", "form", "body", "note", "signatures", "verify"},
}

// KindBlocks کلیدهای بلوک پشتیبانی‌شدهٔ یک سند به ترتیب کاتالوگ.
func KindBlocks(kind Kind) []string {
	supported := map[string]bool{}
	for _, key := range kindBlocks[kind] {
		supported[key] = true
	}
	out := []string{}
	for _, block := range blockCatalog {
		if supported[block.Key] {
			out = append(out, block.Key)
		}
	}
	return out
}

// Layout چیدمان مؤثر یک سند: بلوک‌های روشن و عنوان سفارشی هر بلوک.
type Layout struct {
	// Visible نگاشت بلوک → روشن/خاموش. همیشه از چیدمان پیش‌فرض پر می‌شود.
	Visible map[string]bool
	// Titles عنوان سفارشی بلوک‌ها (کلید → عنوان).
	Titles map[string]string
	// Texts متن سفارشی جای‌های متنی سند (کلید → متن). خالی یعنی پیش‌فرض سامانه.
	Texts map[string]string
}

// Shows آیا بلوک در این سند نمایش داده شود.
func (l Layout) Shows(key string) bool { return l.Visible[key] }

// Text متن سفارشی یک جای سند (پاراگراف متن، یادداشت)؛ رشتهٔ خالی یعنی پیش‌فرض.
func (l Layout) Text(key string) string {
	if l.Texts == nil {
		return ""
	}
	return strings.TrimSpace(l.Texts[key])
}

// Title عنوان بلوک: اگر پنل تنظیمات عنوان سفارشی گذاشته باشد همان، وگرنه پیش‌فرض.
func (l Layout) Title(key, fallback string) string {
	if l.Titles != nil {
		if custom := strings.TrimSpace(l.Titles[key]); custom != "" {
			return custom
		}
	}
	return fallback
}

// DefaultLayout چیدمان پیش‌فرض یک سند: همهٔ بلوک‌های پشتیبانی‌شده روشن.
func DefaultLayout(kind Kind) Layout {
	visible := map[string]bool{}
	for _, key := range KindBlocks(kind) {
		visible[key] = true
	}
	return Layout{Visible: visible}
}

// defaultBodies متن پیش‌فرض پاراگراف هر سند، به شکل «قالب با جای‌نگهدار».
//
// چرا قالب و نه متن آماده: همین رشتهٔ قالب هم روی برگه چاپ می‌شود و هم در پنل
// پارامتریک به کاربر نشان داده می‌شود تا ویرایشش کند؛ پس یک منبع حقیقت است و متن
// پنل با متن چاپ‌شده نمی‌تواند ناهمخوان شود.
var defaultBodies = map[Kind][]string{
	KindContract: {
		"طرف دوم از تاریخ شروع در واحد {{employee.department}} به عنوان {{employee.position}} انجام وظیفه می‌کند.",
		"هفتهٔ کاری شرکت: {{company.workWeek}} و ساعت کار موظفی ماهانه {{params.payroll.monthlyHours}} ساعت است.",
		"مرخصی استحقاقی سالانه {{params.leave.annualDays}} روز و مرخصی استعلاجی طبق قانون کار و ضوابط شرکت است.",
		"مبلغ پایهٔ ماهانه و سایر مزایا مطابق حکم حقوق صادرشده از واحد جبران خدمات پرداخت می‌شود.",
		"رعایت آیین‌نامه‌های داخلی، مقررات حفاظت فنی و محرمانگی اطلاعات شرکت الزامی است.",
		"این قرارداد تابع قانون کار و مقررات تأمین اجتماعی است و اختلاف‌ها از راه مراجع قانونی پیگیری می‌شود.",
	},
	KindSalaryDecree: {
		"مبلغ پایهٔ ماهانهٔ نامبرده بر پایهٔ آخرین حکم و مصوبات شرکت تعیین شده است.",
		"مبانی محاسبهٔ سامانه (ساعت کار موظفی، ضریب اضافه‌کاری و روز پرداخت) در جدول بالا آمده و مبنای اجرای دوره‌های حقوق بعدی است.",
		"هر تغییر در مصوبات، حکم جدید با تاریخ اجرای مشخص تولید می‌کند؛ احکام گذشته بازنویسی نمی‌شوند.",
	},
	KindPayslip: {
		"فیش حقوقی دورهٔ {{payslip.period}} برای {{employee.fullName}} در واحد {{employee.department}} صادر شده است.",
		"جمع ناخالص {{payslip.grossFa}}، بیمهٔ سهم کارمند {{payslip.insuranceEmployeeFa}} و مالیات {{payslip.taxFa}} است.{{payslip.veteranClause}}",
		"مبالغ بر پایهٔ پارامترهای حقوق و اقلام تعریف‌شدهٔ سامانه محاسبه شده و در سرور مرجع است.",
	},
	KindEmploymentCertificate: {
		"گواهی می‌شود {{employee.fullName}} با کد پرسنلی {{employee.id}} از تاریخ {{employee.hireDate}} در شرکت {{company.title}} در واحد {{employee.department}} به عنوان {{employee.position}} اشتغال دارد.{{contract.kindClause}}",
		"این گواهی بنا به درخواست نامبرده و برای ارائه به مراجع ذی‌ربط صادر شده است.",
		"اعتبار این سند با کد رهگیری درج‌شده در پایین صفحه در سامانه قابل بررسی است.",
	},
}

// defaultNotes یادداشت پیش‌فرض پای هر سند.
var defaultNotes = map[Kind]string{
	KindContract:     "این قرارداد در دو نسخه تنظیم و پس از امضای طرفین، هر نسخه حکم واحد خود را دارد.",
	KindSalaryDecree: "پرداخت‌های متغیر مانند کارانه و اضافه‌کاری در دورهٔ حقوق همان ماه محاسبه و پرداخت می‌شود.",
	KindPayslip:      "این فیش از سامانهٔ جامع منابع انسانی صادر شده و در صورت مغایرت، پرداخت سامانه مرجع است.",
}

// DefaultText متن پیش‌فرض جای‌های متنی یک سند (کلید → قالب با جای‌نگهدار).
//
// خروجی خالی برای یک سند یعنی آن سند متن پیش‌فرض قالب‌محور ندارد (مثل قرارداد که
// بندهایش محاسبه‌شده است، یا فرم «مدرک ملاک حقوق» که فقط جدول دارد).
func DefaultText(kind Kind) map[string]string {
	out := map[string]string{}
	if lines := defaultBodies[kind]; len(lines) > 0 {
		out["body"] = strings.Join(lines, "\n")
	}
	if note := strings.TrimSpace(defaultNotes[kind]); note != "" {
		out["note"] = note
	}
	return out
}

// Data همهٔ دادهٔ لازم برای ساخت یک سند چاپی.
type Data struct {
	Kind     Kind
	Serial   string
	IssuedAt string
	Company  Company
	Employee Employee
	Contract *ContractInfo
	Payslip  *PayslipInfo
	// Structure اجزای مزد و فوق‌العاده‌های حکم کارگزینی و قرارداد کار؛ از همان
	// اقلام پارامتریک و همان موتور محاسبهٔ شبیه‌ساز ساخته می‌شود.
	Structure []StructureRow
	Params    []Param
	// OrgParams همهٔ پارامترهای استقرار فعال (هر گروه) برای پرکردن جای‌نگهدارهای
	// متن؛ در جدول «مبنای محاسبهٔ سامانه» چاپ نمی‌شود — آن جدول همان Params است.
	OrgParams []Param
	BodyFa    []string
	NoteFa    string
	// FormTitleFa/FormFields/FormTables فرم اداری سند (مثل «تعیین مدرک ملاک حقوق»).
	FormTitleFa string
	FormFields  []FormField
	FormTables  []FormTable
	// BaseInfoTitleFa/BaseInfoFields بلوک «مشخصات پایهٔ کارمند» (شماره‌ها، نشانی، مدرک).
	// جدا از فرم اداری است تا هر بلوک در پنل تنظیمات مستقل روشن/خاموش شود.
	BaseInfoTitleFa string
	BaseInfoFields  []FormField
	// Layout بلوک‌های نمایش‌داده‌شدهٔ این سند و عنوان سفارشی‌شان.
	Layout Layout
	// Signatures امضاهای پای سند؛ خالی یعنی سه امضای پیش‌فرض کارگزینی/کارمند/منابع انسانی.
	Signatures []string
	// CustomBody یعنی متن از قالب طراحى‌شدهٔ کاربر آمده است و به‌صورت بند‌بند نمایش
	// داده می‌شود؛ نه فهرست پیش‌فرض سامانه.
	CustomBody bool
	// VerifyURL نشانی صفحهٔ تأیید سند (مقصد QR کد چاپشده) و QRCodeSVG همان کد به شکل
	// SVG درون سند. هر دو را سرور میسازد، پس ورودی کاربر نیستند.
	VerifyURL string
	QRCodeSVG template.HTML
}

// ApplyTemplate جای‌نگهدارهای «{{key}}» را در متن قالب با مقادیر واقعی جایگزین می‌کند و
// متن را به خط‌های غیرخالی می‌شکند. مقادیر در زمان رندر توسط html/template escape
// می‌شوند، پس دادهٔ کاربر هرگز به‌صورت کد تفسیر نمی‌شود.
func ApplyTemplate(body string, vars map[string]string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	for key, value := range vars {
		body = strings.ReplaceAll(body, "{{"+key+"}}", value)
	}
	lines := []string{}
	for _, line := range strings.Split(body, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			lines = append(lines, s)
		}
	}
	return lines
}

// PlaceholderVars نگاشت جای‌نگهدارهای مجاز قالب متن بر پایهٔ دادهٔ سند می‌سازد.
// همین فهرست در صفحهٔ قالب‌های متن به کاربر نشان داده می‌شود.
func PlaceholderVars(d Data) map[string]string {
	vars := map[string]string{
		"company.title":         d.Company.TitleFa,
		"company.holding":       d.Company.HoldingFa,
		"company.shortCode":     d.Company.ShortCode,
		"company.currency":      d.Company.CurrencyFa,
		"company.workWeek":      d.Company.WorkWeekFa,
		"serial":                d.Serial,
		"issuedAt":              d.IssuedAt,
		"employee.id":           d.Employee.ID,
		"employee.fullName":     d.Employee.FullName,
		"employee.nationalId":   d.Employee.NationalID,
		"employee.department":   d.Employee.DepartmentName,
		"employee.position":     d.Employee.Position,
		"employee.hireDate":     d.Employee.HireDate,
		"employee.contractType": d.Employee.ContractType,
		// اطلاعات پایهٔ پرونده: در نامه‌های اداری، قرارداد و گواهی‌ها لازم می‌شود.
		"employee.mobile":     d.Employee.Mobile,
		"employee.homePhone":  d.Employee.HomePhone,
		"employee.phone":      d.Employee.Phone,
		"employee.postalCode": d.Employee.PostalCode,
		"employee.address":    d.Employee.Address,
		"employee.email":      d.Employee.Email,
		// تحصیلات: متن یک‌خطی و اجزای آن، تا هر قالب متن دلخواه خودش را بسازد.
		"employee.education":       d.Employee.EducationText,
		"employee.educationLevel":  d.Employee.EducationLevelFa,
		"employee.educationField":  d.Employee.EducationField,
		"employee.educationUniver": d.Employee.EducationUniver,
		"employee.educationDate":   d.Employee.EducationDate,
		"employee.degreeBasis":     d.Employee.DegreeBasisFa,
	}
	// جای‌نگهدارهای شرطی همیشه تعریف می‌شوند؛ وگرنه قالبی که به بخشی اشاره می‌کند که
	// در این سند نیست (مثل قرارداد فعالِ نداشته) جای‌نگهدار خام روی برگه چاپ می‌کند.
	vars["contract.kindClause"] = ""
	vars["payslip.grossFa"] = ""
	vars["payslip.insuranceEmployeeFa"] = ""
	vars["payslip.insuranceEmployerFa"] = ""
	vars["payslip.taxFa"] = ""
	vars["payslip.netFa"] = ""
	vars["payslip.veteranClause"] = ""
	if d.Contract != nil {
		vars["contract.id"] = d.Contract.ID
		if strings.TrimSpace(d.Contract.KindFa) != "" {
			// جملهٔ «نوع همکاری» فقط وقتی معنا دارد که قرارداد فعالی وجود داشته باشد.
			vars["contract.kindClause"] = " نوع همکاری ایشان «" + d.Contract.KindFa + "» است."
		}
		vars["contract.kind"] = d.Contract.KindFa
		vars["contract.status"] = d.Contract.StatusFa
		vars["contract.startDate"] = d.Contract.StartDate
		vars["contract.endDate"] = d.Contract.EndDate
		vars["contract.monthlySalary"] = fmt.Sprintf("%d", d.Contract.MonthlySalary)
		vars["contract.serviceYears"] = fmt.Sprintf("%d", d.Contract.ServiceYears)
	}
	if d.Payslip != nil {
		vars["payslip.period"] = d.Payslip.Period
		vars["payslip.gross"] = fmt.Sprintf("%d", d.Payslip.Gross)
		// نسخهٔ خوانا (با جداکنندهٔ هزارگان) برای متن پاراگراف؛ رقم فارسی را خود
		// موتور چاپ می‌گذارد.
		vars["payslip.grossFa"] = money(d.Payslip.Gross)
		vars["payslip.insuranceEmployeeFa"] = money(d.Payslip.InsuranceEmployee)
		vars["payslip.insuranceEmployerFa"] = money(d.Payslip.InsuranceEmployer)
		vars["payslip.taxFa"] = money(d.Payslip.Tax)
		vars["payslip.netFa"] = money(d.Payslip.Net)
		vars["payslip.insuranceEmployee"] = fmt.Sprintf("%d", d.Payslip.InsuranceEmployee)
		vars["payslip.insuranceEmployer"] = fmt.Sprintf("%d", d.Payslip.InsuranceEmployer)
		vars["payslip.tax"] = fmt.Sprintf("%d", d.Payslip.Tax)
		vars["payslip.net"] = fmt.Sprintf("%d", d.Payslip.Net)
		lines := make([]string, 0, len(d.Payslip.Elements))
		for _, el := range d.Payslip.Elements {
			lines = append(lines, fmt.Sprintf("%s: %s", el.TitleFa, el.AmountFa))
		}
		vars["payslip.elements"] = strings.Join(lines, "\n")
		vars["payslip.payGap"] = fmt.Sprintf("%d", d.Payslip.PayGap)
		vars["payslip.payGapFa"] = money(d.Payslip.PayGap)
		vars["payslip.payGapSince"] = d.Payslip.PayGapSince
		// جملهٔ امتیاز ایثارگری فقط وقتی پر می‌شود که پروندهٔ تأییدشده‌ای اعمال شده باشد؛
		// جای‌نگهدارش از پیش تعریف شده تا قالب، متن خام روی برگه چاپ نکند.
		vars["payslip.veteranClause"] = ""
		if strings.TrimSpace(d.Payslip.VeteranGroupFa) != "" {
			vars["payslip.veteranClause"] = " با اعمال امتیاز ایثارگری (" + d.Payslip.VeteranGroupFa +
				")، معافیت مالیاتی " + d.Payslip.VeteranTaxExemptFa + " ریال و معافیت کسور " +
				d.Payslip.VeteranPensionExemptFa + " ریال لحاظ شده است."
		}
	}
	if len(d.Structure) > 0 {
		lines := make([]string, 0, len(d.Structure))
		for _, row := range d.Structure {
			lines = append(lines, fmt.Sprintf("%s: %s", row.TitleFa, row.AmountFa))
		}
		vars["structure.text"] = strings.Join(lines, "\n")
	}
	for _, p := range d.Params {
		if p.Key != "" {
			vars["params."+p.Key] = p.Value
		}
	}
	// پارامترهای سازمانی خارج از جدول سند (مثل مرخصی سالانه) هم در دسترس متن هستند؛
	// اگر کلیدی همین‌جا تعریف شده باشد، همان مقدار جدول مقدم است.
	for _, p := range d.OrgParams {
		if p.Key == "" {
			continue
		}
		if _, exists := vars["params."+p.Key]; !exists {
			vars["params."+p.Key] = p.Value
		}
	}
	return vars
}

// VerifyCode کد رهگیری سند: امضای کوتاه از شمارهٔ سند و کد کارمند، تا سند کاغذی
// قابل تطبیق با نسخهٔ سامانه باشد.
func VerifyCode(serial, employeeID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(serial) + "|" + strings.TrimSpace(employeeID)))
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:8]), "=")
}

// Render سند را به HTML چاپی تبدیل می‌کند.
func Render(data Data) (string, error) {
	if _, ok := ModuleOf(data.Kind); !ok {
		return "", fmt.Errorf("نوع سند ناشناخته: %s", data.Kind)
	}
	// چیدمان خالی یعنی «پیش‌فرض همین سند»: هر سازندهٔ داده که Layout را پر نکند (آزمون،
	// پیش‌نمایش) نباید سند خالی بگیرد.
	if data.Layout.Visible == nil {
		data.Layout = DefaultLayout(data.Kind)
	}
	view := struct {
		Data
		Title      string
		VerifyCode string
	}{
		Data:       data,
		Title:      TitleFa(data.Kind),
		VerifyCode: VerifyCode(data.Serial, data.Employee.ID),
	}
	var buf bytes.Buffer
	if err := documentTemplate.Execute(&buf, view); err != nil {
		return "", fmt.Errorf("ساخت سند ناموفق: %w", err)
	}
	return PersianDigits(buf.String()), nil
}

// PersianDigits ارقام لاتین را در متنِ سند به رقم فارسی برمی‌گرداند.
//
// چرا اینجا و نه سر هر جای‌نگهدار: تاریخ، کد ملی، مبلغ و شمارهٔ تماس در دادهٔ سرور
// با رقم لاتین ذخیره می‌شوند (قالب استاندارد، مرتب‌شدنی و جست‌وجوپذیر). اما سند چاپی
// فارسی است و قاطی‌شدن ارقام لاتین و فارسی روی یک برگه آن را ناموزون نشان می‌دهد.
//
// فقط متنِ بین تگ‌ها دست می‌خورد تا ویژگی‌ها (href، class، style) سالم بمانند، و
// متنی که حرف لاتین دارد دست‌نخورده می‌ماند چون شناسه است (DEC-1405-004،
// CT-1399-0004، کد رهگیری). یعنی همان سیاستی که در رابط وب هم اجرا می‌شود.
func PersianDigits(html string) string {
	var b strings.Builder
	b.Grow(len(html))
	for i := 0; i < len(html); {
		if html[i] == '<' {
			j := strings.IndexByte(html[i:], '>')
			if j < 0 {
				b.WriteString(html[i:])
				break
			}
			b.WriteString(html[i : i+j+1])
			i += j + 1
			continue
		}
		j := strings.IndexByte(html[i:], '<')
		if j < 0 {
			j = len(html) - i
		}
		b.WriteString(faDigitsText(html[i : i+j]))
		i += j
	}
	return b.String()
}

// faDigitsText یک قطعهٔ متنی (نه مارک‌آپ) را فارسی می‌کند.
//
// واژه‌به‌واژه پیش می‌رود نه کل قطعه: در سند، شناسه و تاریخ غالباً در یک گرهٔ متنی
// پشت سر هم می‌آیند («شمارهٔ سند: DEC-1405-004 تاریخ: 1405/07/12»). اگر معیار روی
// کل قطعه بود، حضور یک شناسه همهٔ ارقام آن را لاتین نگه می‌داشت. پس فقط واژه‌های
// بدون حرف لاتین فارسی می‌شوند.
func faDigitsText(s string) string {
	if !strings.ContainsFunc(s, isASCIIDigit) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if isSpaceByte(s[i]) {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i
		for j < len(s) && !isSpaceByte(s[j]) {
			j++
		}
		word := s[i:j]
		if strings.ContainsFunc(word, isLatinLetter) {
			b.WriteString(word) // شناسه: EMP-004، CT-1399-0004، کد رهگیری
		} else {
			b.WriteString(persianizeDigits(word))
		}
		i = j
	}
	return b.String()
}

// persianizeDigits ارقام لاتین یک واژه را به رقم فارسی برمی‌گرداند.
func persianizeDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune('۰' + (r - '0'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isSpaceByte فقط جداکننده‌های واژه در سند است؛ نیم‌فاصله (U+200C) جزء واژه می‌ماند.
func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }

func isLatinLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

var documentTemplate = template.Must(template.New("document").Parse(`<!doctype html>
<html lang="fa" dir="rtl">
<head>
<meta charset="utf-8">
<title>{{.Title}} — {{.Company.TitleFa}}</title>
<style>
  /* فونت فارسی سند روی همان مبدأ رابط وب سرو می‌شود؛ مسیر مطلق است چون خود سند زیر
     /api/v1/documents/... سرو می‌شود و مسیر نسبی به آن مسیر می‌چسبید. */
  @font-face { font-family: Vazirmatn; font-style: normal; font-weight: 100 900; font-display: swap;
    src: url("/assets/fonts/Vazirmatn-var.woff2") format("woff2-variations"); }
  @page { size: A4; margin: 18mm 16mm 20mm 16mm; }
  * { box-sizing: border-box; }
  body { font-family: "Vazirmatn", "IRANSans", "Tahoma", sans-serif; color: #111827; margin: 0; padding: 24px; background: #f8fafc; font-size: 13px; line-height: 1.9; -webkit-font-smoothing: antialiased; }
  .sheet { background: #fff; max-width: 820px; margin: 0 auto; padding: 28px 30px 40px; border-radius: 10px; box-shadow: 0 4px 20px rgba(15,23,42,.08); }
  header { border-bottom: 2px solid {{if .Company.BrandColor}}{{.Company.BrandColor}}{{else}}#2563eb{{end}}; padding-bottom: 12px; margin-bottom: 18px; display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
  .brand { font-weight: 800; font-size: 16px; display: flex; align-items: center; gap: 12px; }
  .brand-logo { width: 104px; height: 56px; object-fit: contain; object-position: right; }
  .brand-text small { display: block; font-weight: 500; color: #64748b; font-size: 11px; }
  .brand small { display: block; font-weight: 500; color: #64748b; font-size: 11px; }
  .doc-meta { text-align: left; font-size: 11px; color: #475569; direction: ltr; }
  h1 { font-size: 18px; margin: 0 0 4px; }
  .subtitle { color: #64748b; font-size: 12px; margin-bottom: 18px; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 16px; }
  th, td { border: 1px solid #e2e8f0; padding: 7px 9px; text-align: right; vertical-align: top; }
  th { background: #f1f5f9; font-weight: 600; width: 22%; }
  .grid-stats { display: flex; gap: 10px; margin-bottom: 16px; }
  .stat { flex: 1; border: 1px solid #e2e8f0; border-radius: 8px; padding: 8px 10px; }
  .stat b { display: block; font-size: 14px; }
  .stat span { color: #64748b; font-size: 11px; }
  .num { font-variant-numeric: tabular-nums; }
  h2 { font-size: 14px; margin: 18px 0 8px; padding-right: 8px; border-right: 3px solid #2563eb; }
  ul { padding-right: 18px; margin: 0 0 14px; }
  .doc-body { border: 1px solid #e2e8f0; border-radius: 8px; padding: 6px 14px; margin-bottom: 16px; }
  .doc-body p { margin: 7px 0; text-align: justify; }
  .sign { display: flex; gap: 24px; margin-top: 30px; }
  .sign div { flex: 1; border-top: 1px dashed #94a3b8; padding-top: 6px; font-size: 11px; color: #475569; text-align: center; }
  footer { margin-top: 26px; border-top: 1px solid #e2e8f0; padding-top: 10px; font-size: 10px; color: #64748b; display: flex; justify-content: space-between; gap: 12px; }
  .toolbar { max-width: 820px; margin: 0 auto 12px; display: flex; gap: 8px; justify-content: flex-start; }
  .toolbar button { font-family: inherit; font-size: 13px; padding: 8px 16px; border-radius: 8px; border: 1px solid #2563eb; background: #2563eb; color: #fff; cursor: pointer; }
  .toolbar a { font-size: 12px; color: #2563eb; align-self: center; }
  @media print {
    body { background: #fff; padding: 0; }
    .sheet { box-shadow: none; border-radius: 0; padding: 0; max-width: none; }
    .toolbar { display: none; }
    h2 { break-after: avoid; }
    table { break-inside: avoid; }
  }
</style>
</head>
<body>
<div class="toolbar">
  <button onclick="window.print()">چاپ / ذخیرهٔ PDF</button>
  <a href="?">بازخوانی سند</a>
</div>
<div class="sheet">
  <header>
    <div class="brand">
      {{if .Company.LogoURL}}<img class="brand-logo" src="{{.Company.LogoURL}}" alt="{{.Company.HoldingFa}}">{{end}}
      <div class="brand-text">{{.Company.TitleFa}}
        <small>{{.Company.HoldingFa}} · کد شرکت {{.Company.ShortCode}}</small></div>
    </div>
    <div class="doc-meta">
      شمارهٔ سند: {{.Serial}}<br>
      تاریخ: {{.IssuedAt}}<br>
      کد رهگیری: {{.VerifyCode}}
    </div>
  </header>

  <h1>{{.Title}}</h1>
  <div class="subtitle">این سند از سامانهٔ جامع منابع انسانی صادر شده است.</div>

  {{if .Layout.Shows "employeeCard"}}
  <div class="grid-stats">
    <div class="stat"><b>{{.Employee.FullName}}</b><span>نام و نام خانوادگی</span></div>
    <div class="stat"><b class="num">{{.Employee.ID}}</b><span>کد پرسنلی</span></div>
    <div class="stat"><b>{{.Employee.DepartmentName}}</b><span>واحد سازمانی</span></div>
    <div class="stat"><b>{{.Employee.Position}}</b><span>پست سازمانی</span></div>
  </div>
  {{end}}

  {{if .Layout.Shows "employeeInfo"}}
  <h2>{{.Layout.Title "employeeInfo" "مشخصات کارمند"}}</h2>
  <table>
    <tr><th>کد ملی</th><td class="num">{{.Employee.NationalID}}</td><th>تاریخ استخدام</th><td class="num">{{.Employee.HireDate}}</td></tr>
    <tr><th>نوع همکاری</th><td>{{.Employee.ContractType}}</td><th>هفتهٔ کاری شرکت</th><td>{{.Company.WorkWeekFa}}</td></tr>
  </table>
  {{end}}

  {{if .Layout.Shows "contactInfo"}}
  {{if or .Employee.Mobile .Employee.HomePhone .Employee.Phone .Employee.PostalCode .Employee.Address}}
  <h2>{{.Layout.Title "contactInfo" "اطلاعات تماس و نشانی"}}</h2>
  <table>
    {{if or .Employee.Mobile .Employee.HomePhone .Employee.Phone}}
    <tr><th>شمارهٔ همراه</th><td class="num">{{if .Employee.Mobile}}{{.Employee.Mobile}}{{else}}—{{end}}</td><th>تلفن منزل</th><td class="num">{{if .Employee.HomePhone}}{{.Employee.HomePhone}}{{else}}—{{end}}</td></tr>
    {{end}}
    {{if or .Employee.PostalCode .Employee.Address}}
    <tr><th>کد پستی</th><td class="num">{{if .Employee.PostalCode}}{{.Employee.PostalCode}}{{else}}—{{end}}</td><th>نشانی محل سکونت</th><td>{{if .Employee.Address}}{{.Employee.Address}}{{else}}—{{end}}</td></tr>
    {{end}}
  </table>
  {{end}}
  {{end}}

  {{if .Layout.Shows "education"}}
  {{if .Employee.HasEducationInfo}}
  <h2>{{.Layout.Title "education" "سابقهٔ تحصیلی"}}</h2>
  <table>
    <tr><th>مدرک تحصیلی</th><td>{{if .Employee.EducationLevelFa}}{{.Employee.EducationLevelFa}}{{else}}—{{end}}</td><th>رشته</th><td>{{if .Employee.EducationField}}{{.Employee.EducationField}}{{else}}—{{end}}</td></tr>
    <tr><th>گرایش</th><td>{{if .Employee.EducationOrient}}{{.Employee.EducationOrient}}{{else}}—{{end}}</td><th>دانشگاه / مؤسسه</th><td>{{if .Employee.EducationUniver}}{{.Employee.EducationUniver}}{{else}}—{{end}}</td></tr>
    <tr><th>تاریخ فراغت از تحصیل</th><td class="num">{{if .Employee.EducationDate}}{{.Employee.EducationDate}}{{else}}—{{end}}</td><th>مدرک ملاک حقوق</th><td>{{if .Employee.DegreeBasisFa}}{{.Employee.DegreeBasisFa}}{{else}}—{{end}}</td></tr>
  </table>
  {{end}}
  {{end}}

  {{if .Layout.Shows "contract"}}
  {{if .Contract}}
  <h2>{{.Layout.Title "contract" "مشخصات قرارداد کار"}}</h2>
  <table>
    <tr><th>شمارهٔ قرارداد</th><td class="num">{{.Contract.ID}}</td><th>نوع قرارداد</th><td>{{.Contract.KindFa}}</td></tr>
    <tr><th>تاریخ شروع</th><td class="num">{{.Contract.StartDate}}</td><th>تاریخ پایان</th><td class="num">{{if .Contract.EndDate}}{{.Contract.EndDate}}{{else}}— (مدت دائم){{end}}</td></tr>
    <tr><th>وضعیت قرارداد</th><td>{{.Contract.StatusFa}}</td><th>سابقهٔ خدمت</th><td class="num">{{.Contract.ServiceYears}} سال</td></tr>
    <tr><th>مبلغ پایهٔ ماهانه</th><td class="num">{{.Contract.MonthlySalary}} {{.Company.CurrencyFa}}</td><th>سال مالی شرکت</th><td class="num">{{.Company.FiscalYearStartFa}}</td></tr>
  </table>
  {{end}}
  {{end}}

  {{if .Layout.Shows "params"}}
  {{if .Params}}
  <h2>{{.Layout.Title "params" "مبنای محاسبهٔ سامانه"}}</h2>
  <table>
    <tr><th>پارامتر</th><th>مقدار</th><th>واحد</th><th>گروه</th></tr>
    {{range .Params}}<tr><td>{{.TitleFa}}</td><td class="num">{{.Value}}</td><td>{{.UnitFa}}</td><td>{{.GroupFa}}</td></tr>{{end}}
  </table>
  {{end}}
  {{end}}

  {{if .Layout.Shows "payslipElements"}}
  {{if .Payslip}}{{if .Payslip.Elements}}
  <h2>{{.Layout.Title "payslipElements" "تفکیک اقلام حقوق"}}</h2>
  {{if .Payslip.MazdBase}}
  <table>
    <tr><th>مزد مبنا (مبنای اضافه‌کاری و فوق‌العاده‌های درصدی)</th><td class="num">{{.Payslip.MazdBase}}</td>
        <th>منبع اجزای مزد</th><td>{{if .Payslip.WageSourceFa}}{{.Payslip.WageSourceFa}}{{else}}پروندهٔ پرسنلی و پارامترهای شرکت{{end}}</td></tr>
  </table>
  {{end}}
  {{if .Payslip.PayGap}}
  <table>
    <tr><th>تفاوت تطبیق (جذب‌شدنی)</th><td class="num">{{.Payslip.PayGapFa}}</td>
        <th>جایگاه در شاکلهٔ مزد</th><td>خارج از مزد مبنا؛ مشمول بیمه و مالیات</td></tr>
    {{if .Payslip.PayGapSince}}<tr><th>از تاریخ</th><td class="num">{{.Payslip.PayGapSince}}</td>
        <th>وضعیت</th><td>با هر افزایش حقوق کاهش می‌یابد تا جذب کامل شود</td></tr>{{end}}
  </table>
  {{end}}
  <table>
    <tr><th>قلم حقوق</th><th>نوع</th><th>مبلغ محاسبه‌شده</th><th>شرح</th></tr>
    {{range .Payslip.Elements}}<tr><td>{{.TitleFa}}</td><td>{{.CategoryFa}}</td><td class="num">{{.AmountFa}}</td><td>{{.DetailFa}}</td></tr>{{end}}
  </table>
  {{end}}{{end}}
  {{end}}

  {{if .Layout.Shows "structure"}}
  {{if .Structure}}
  <h2>{{.Layout.Title "structure" "اجزای مزد مبنا و فوق‌العاده‌ها"}}</h2>
  <table>
    <tr><th>جزء / فوق‌العاده</th><th>نوع</th><th>نحوهٔ محاسبه</th><th>مبلغ (ریال)</th></tr>
    {{range .Structure}}<tr{{if .Total}} class="total"{{end}}><td>{{if .Total}}<b>{{.TitleFa}}</b>{{else}}{{.TitleFa}}{{end}}</td><td>{{.KindFa}}</td><td>{{.DetailFa}}</td><td class="num">{{.AmountFa}}</td></tr>{{end}}
  </table>
  <p class="muted">مزد مبنا جمع اجزای ثابت است و مبنای محاسبهٔ اضافه‌کاری، شب‌کاری، نوبت‌کاری و فوق‌العاده‌های اقلیمی (که درصدی از همین مزد مبنا هستند و جداگانه پرداخت می‌شوند).</p>
  {{end}}
  {{end}}

  {{if .Layout.Shows "baseInfoForm"}}
  {{if .BaseInfoFields}}
  <h2>{{if .BaseInfoTitleFa}}{{.BaseInfoTitleFa}}{{else}}مشخصات پایهٔ کارمند{{end}}</h2>
  <table>
    {{range .BaseInfoFields}}<tr><th>{{.LabelFa}}</th><td>{{.ValueFa}}</td></tr>{{end}}
  </table>
  {{end}}
  {{end}}

  {{if .Layout.Shows "form"}}
  {{if .FormFields}}
  <h2>{{if .FormTitleFa}}{{.FormTitleFa}}{{else}}مشخصات فرم{{end}}</h2>
  <table>
    {{range .FormFields}}<tr><th>{{.LabelFa}}</th><td>{{.ValueFa}}</td></tr>{{end}}
  </table>
  {{end}}

  {{range .FormTables}}
  <h2>{{.TitleFa}}</h2>
  <table>
    {{if .Head}}<tr>{{range .Head}}<th>{{.}}</th>{{end}}</tr>{{end}}
    {{range .Rows}}<tr>{{range .}}<td>{{.}}</td>{{end}}</tr>{{end}}
  </table>
  {{if .NoteFa}}<p class="muted">{{.NoteFa}}</p>{{end}}
  {{end}}
  {{end}}

  {{if .Layout.Shows "body"}}
  {{if .BodyFa}}
  {{if .CustomBody}}
  <h2>{{.Layout.Title "body" "متن سند"}}</h2>
  <div class="doc-body">{{range .BodyFa}}<p>{{.}}</p>{{end}}</div>
  {{else}}
  <h2>{{.Layout.Title "body" "شرح"}}</h2>
  <ul>{{range .BodyFa}}<li>{{.}}</li>{{end}}</ul>
  {{end}}
  {{end}}
  {{end}}

  {{if .Layout.Shows "note"}}
  {{if .NoteFa}}<div class="subtitle">📌 {{.NoteFa}}</div>{{end}}
  {{end}}

  {{if .Layout.Shows "signatures"}}
  <div class="sign">
    {{if .Signatures}}{{range .Signatures}}<div>{{.}}</div>{{end}}{{else}}
    <div>مهر و امضای واحد کارگزینی</div>
    <div>امضای کارمند</div>
    <div>تأیید مدیر منابع انسانی</div>
    {{end}}
  </div>
  {{end}}

  <footer>
    <span>سامانهٔ جامع منابع انسانی — {{.Company.TitleFa}}</span>
    {{if .Layout.Shows "verify"}}
    <span>کد رهگیری سند: {{.VerifyCode}}</span>
    {{if .QRCodeSVG}}
    <span class="doc-qr">
      {{.QRCodeSVG}}
      <small>برای بررسی اصالت سند، این کد را با دوربین تلفن بخوانید.</small>
    </span>
    {{end}}
    {{end}}
  </footer>
</div>
<script>
  // با باز شدن سند در قالب «چاپ»، پنجرهٔ چاپ خودکار باز می‌شود؛ این‌گونه کاربر یک
  // کلیک می‌کند و PDF ذخیره می‌شود، بدون هیچ ابزار نصب‌شدنی روی سرور.
  if (new URLSearchParams(location.search).get('print') === '1') {
    window.addEventListener('load', function () { window.print(); });
  }
</script>
</body>
</html>
`))
