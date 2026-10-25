// Package domain دامنهٔ ماژول تنظیمات.
//
// پارامتر سیستمی و پارامتر سازمانی هر یک یک «نوع رکورد» است. کاربران، نقش‌ها و لاگ
// حسابرسی صفحه‌های اختصاصی خودشان را دارند و اینجا فقط پارامترها مدیریت می‌شوند.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// DocType نوع سند چاپی که متنش قابل طراحی است.
type DocType struct {
	Value   string
	LabelFa string
}

// DocTypes انواع سند قابل قالب‌بندی.
var DocTypes = []DocType{
	{"contract", "قرارداد کار"},
	{"payslip", "فیش حقوقی"},
	{"salary_decree", "حکم حقوق"},
	{"employment_certificate", "گواهی اشتغال به کار"},
}

// DocTypeFa عنوان فارسی یک نوع سند.
func DocTypeFa(value string) string {
	for _, t := range DocTypes {
		if t.Value == value {
			return t.LabelFa
		}
	}
	return value
}

// ValidDocType بررسی مجاز بودن نوع سند.
func ValidDocType(value string) bool {
	for _, t := range DocTypes {
		if t.Value == value {
			return true
		}
	}
	return false
}

// Descriptor شناسهٔ ماژول تنظیمات.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "settings",
		TitleFa: "تنظیمات",
		Kinds: []recordkit.Kind{
			{
				Key: "sysParams", TitleFa: "پارامتر سیستم", IDPrefix: "PARAM", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"key|کلید پارامتر|text|1",
					"value|مقدار|text|0",
					"group|گروه|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال", "inactive|غیرفعال"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|active", "active|inactive", "inactive|active"),
				Actions: recordkit.Actions(
					"activate|فعال‌سازی|primary|active|0",
					"deactivate|غیرفعال‌سازی|ghost|inactive|0",
				),
			},
			docTemplatesKind(),
			// شرکت‌ها و استقرارهای هلدینگ: فرم ورود شرکت تازه و ویرایش شرکت‌های موجود.
			companyProfilesKind(),
			{
				Key: "orgSettings", TitleFa: "پارامتر سازمانی", IDPrefix: "SET", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"key|کلید|text|1",
					"value|مقدار|text|0",
					"group|گروه|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved"),
				Actions: recordkit.Actions(
					"approve|تأیید پارامتر|primary|approved|0",
				),
			},
			// کاتالوگ‌های مرجع زمان‌اجرا: مدیریت‌ها، بانک‌ها و واحدها. ماژول‌های وام و پاداش
			// فهرست‌هایشان را از همین رکوردها می‌خوانند (از راه پورت کاتالوگ)، پس افزودن
			// مدیریت یا بانک تازه در همین پنل، خودکار در فرم‌های وام و پاداش دیده می‌شود.
			catalogKind("managements", "مدیریت", "MGMT", recordkit.Fields(
				"key|کلید لاتین (ماشین‌خوان)|text|1",
			)),
			catalogKind("banks", "بانک عامل", "BANK", recordkit.Fields(
				"key|کلید لاتین (ماشین‌خوان)|text|1",
				"code|کد بانک|text|0",
			)),
			catalogKind("units", "واحد سازمانی", "UNIT", recordkit.Fields(
				"key|کلید لاتین (ماشین‌خوان)|text|1",
				// مدیریت بالادست: واحد همیشه زیر یک مدیریت می‌نشیند؛ برای سهمیهٔ ریزتر و
				// تعیین اینکه وام به کدام واحد رفته است.
				"management|مدیریت بالادست|select|0",
				"costCenter|مرکز هزینه|text|0",
			)),
		},
	}
}

// CatalogKinds کلیدهای کاتالوگ‌های مرجع. همان مقدار در پورت کاتالوگ و در فیلد `kind`
// رکوردهای تنظیمات به‌کار می‌رود.
var CatalogKinds = []string{contracts.CatalogManagements, contracts.CatalogBanks, contracts.CatalogUnits}

// ValidCatalogKind بررسی شناخته‌شده بودن کاتالوگ.
func ValidCatalogKind(kind string) bool {
	for _, k := range CatalogKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// CatalogKindFa عنوان فارسی یک کاتالوگ.
func CatalogKindFa(kind string) string {
	switch kind {
	case contracts.CatalogManagements:
		return "مدیریت‌ها"
	case contracts.CatalogBanks:
		return "بانک‌های عامل"
	case contracts.CatalogUnits:
		return "واحدهای سازمانی"
	}
	return kind
}

// catalogKind یک نوع رکورد کاتالوگ مرجع می‌سازد: یک قلم با کلید ماشین‌خوان، عنوان
// فارسی (عنوان خود رکورد) و وضعیت فعال/غیرفعال. غیرفعال کردن یعنی قلم از فرم‌ها و
// اعتبارسنجی بیرون می‌رود ولی دادهٔ قدیمی که به آن اشاره دارد از بین نمی‌رود.
func catalogKind(key, titleFa, prefix string, fields []recordkit.Field) recordkit.Kind {
	return recordkit.Kind{
		Key: key, TitleFa: titleFa, IDPrefix: prefix, EmployeeScoped: false,
		Fields:      fields,
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال", "inactive|غیرفعال"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|active", "active|inactive", "inactive|active"),
		Actions: recordkit.Actions(
			"activate|فعال‌سازی|primary|active|0",
			"deactivate|غیرفعال‌سازی|ghost|inactive|0",
		),
		Validate: validateCatalogItem,
		// کاتالوگ مرجع، دادهٔ پایه است نه سند گردش‌کاری؛ پس ویرایش در هر وضعیتی مجاز
		// است و «فعال/غیرفعال» فقط یعنی قلم در فرم‌ها دیده شود یا نه.
		EditableStatuses: []string{"draft", "active", "inactive"},
	}
}

// validateCatalogItem قاعدهٔ قلم کاتالوگ: کلید ماشین‌خوان باید لاتین و بی‌فاصله باشد
// (چون در داده و API ذخیره می‌شود) و عنوان فارسی داشته باشد (چون در فرم‌ها نمایش داده
// می‌شود). هیچ‌کدام در UI تکرار نمی‌شود.
func validateCatalogItem(in contracts.RecordInput) error {
	key := strings.TrimSpace(in.Fields["key"])
	if key == "" {
		return &recordkit.RuleError{Message: "کلید کاتالوگ الزامی است."}
	}
	if !machineKey(key) {
		return &recordkit.RuleError{Message: "کلید باید فقط حروف و ارقام لاتین و خط تیره/زیرخط داشته باشد (مثل bandar_imam)."}
	}
	if strings.TrimSpace(in.Title) == "" {
		return &recordkit.RuleError{Message: "عنوان فارسی قلم الزامی است."}
	}
	return nil
}

// machineKey آیا رشته یک کلید ماشین‌خوان مجاز است.
func machineKey(s string) bool {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-', r == '.':
		default:
			return false
		}
	}
	return true
}

// docTemplatesKind قالب‌های متن اسناد چاپی: متن قرارداد، فیش حقوق، حکم حقوق و گواهی
// اشتغال به‌صورت قابل طراحی نگه‌داری می‌شود و موتور اسناد قالب فعال را می‌خواند.
func docTemplatesKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "docTemplates", TitleFa: "قالب متن اسناد", IDPrefix: "TPL", EmployeeScoped: false,
		Fields: recordkit.Fields(
			"docType|نوع سند (contract/payslip/salary_decree/employment_certificate)|text|1",
			"header|سرصفحهٔ دلخواه|text|0",
			"body|متن قالب|text|1",
			"footer|پانویس دلخواه|text|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|قالب فعال", "archived|بایگانی‌شده"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|active", "active|archived", "archived|active"),
		Actions: recordkit.Actions(
			"activate|فعال‌سازی قالب|primary|active|0",
			"archive|بایگانی قالب|ghost|archived|0",
		),
		Validate: validateDocTemplate,
	}
}

func validateDocTemplate(in contracts.RecordInput) error {
	if !ValidDocType(strings.TrimSpace(in.Fields["docType"])) {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.docType": "نوع سند باید contract، payslip، salary_decree یا employment_certificate باشد.",
		}}
	}
	if strings.TrimSpace(in.Fields["body"]) == "" {
		return &recordkit.ValidationError{Details: map[string]string{"fields.body": "متن قالب الزامی است."}}
	}
	return nil
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	out := []contracts.RecordInput{
		{Kind: "sysParams", Title: "ساعت کاری ماهانه",
			Fields: map[string]string{"key": "payroll.monthlyHours", "value": "192", "group": "حقوق و دستمزد"},
			NoteFa: "مبنای محاسبهٔ نرخ ساعتی"},
		{Kind: "sysParams", Title: "روزهای مرخصی سالانه",
			Fields: map[string]string{"key": "leave.annualDays", "value": "26", "group": "مرخصی"}},
		{Kind: "orgSettings", Title: "ضریب فوق‌العادهٔ سختی شرایط کار",
			Fields: map[string]string{"key": "hardship.coefficient", "value": "0.15", "group": "مزایا"},
			NoteFa: "پارامتریک؛ در محاسبهٔ حقوق خوانده می‌شود"},
		{Kind: "docTemplates", Title: "قالب قرارداد کار",
			Fields: map[string]string{
				"docType": "contract",
				"body":    "طرف اول: {{company.title}} ({{company.holding}}).\nطرف دوم: {{employee.fullName}} به کد پرسنلی {{employee.id}}.\nعنوان شغل: {{employee.position}} در واحد {{employee.department}}.\nتاریخ شروع همکاری: {{contract.startDate}}.\nساعت کار موظفی ماهانه: {{params.payroll.monthlyHours}}.\nمرخصی استحقاقی سالانه: {{params.leave.annualDays}}.\nمبلغ پایهٔ ماهانه: {{contract.monthlySalary}} {{company.currency}}.",
			},
			NoteFa: "جای‌نگهدارها هنگام چاپ با دادهٔ واقعی کارمند جایگزین می‌شوند."},
		// — کاتالوگ مرجع مدیریت‌ها (هشت مدیریت بندر امام) —
		mgmtCatalog("bandar_imam", "بندر امام", "مدیریت عملیات بندر"),
		mgmtCatalog("repair_support", "مدیریت پشتیبانی تعمیرات", "تعمیرات اساسی و نگهداری"),
		mgmtCatalog("water_power", "آب و نیرو", "آب، برق و نیروگاه"),
		mgmtCatalog("basparan", "بسپاران", "واحد بسپاران"),
		mgmtCatalog("process_one", "فراورش یک", "فراورش یک"),
		mgmtCatalog("process_two", "فراورش دو", "فراورش دو"),
		mgmtCatalog("services_support", "معاونت پشتیبانی خدمات", "خدمات پشتیبانی"),
		mgmtCatalog("kimia", "کیمیا", "شرکت کیمیا"),
		// — کاتالوگ مرجع بانک‌های عامل —
		bankCatalog("melli", "بانک ملی"),
		bankCatalog("mellat", "بانک ملت"),
		bankCatalog("mehreghtesad", "بانک مهر اقتصاد"),
		bankCatalog("saderat", "بانک صادرات"),
		bankCatalog("tejarat", "بانک تجارت"),
		bankCatalog("sepah", "بانک سپه"),
		bankCatalog("refah", "بانک رفاه"),
		bankCatalog("keshavarzi", "بانک کشاورزی"),
		bankCatalog("maskan", "بانک مسکن"),
		// — کاتالوگ مرجع واحدهای سازمانی (زیر هر مدیریت) —
		unitCatalog("bandar_ops", "واحد عملیات اسکله", "bandar_imam"),
		unitCatalog("repair_mech", "واحد تعمیرات مکانیک", "repair_support"),
		unitCatalog("power_plant", "واحد نیروگاه", "water_power"),
		unitCatalog("kimia_lab", "واحد آزمایشگاه کیمیا", "kimia"),
		{Kind: "docTemplates", Title: "قالب فیش حقوقی",
			Fields: map[string]string{
				"docType": "payslip",
				"body":    "فیش حقوقی {{employee.fullName}} — کد پرسنلی {{employee.id}}.\nجمع ناخالص: {{payslip.gross}} {{company.currency}}.\nبیمهٔ سهم کارمند: {{payslip.insuranceEmployee}}.\nمالیات: {{payslip.tax}}.\nخالص پرداختی: {{payslip.net}} {{company.currency}}.",
			},
			NoteFa: "قالب متن فیش را می‌توانید ویرایش کنید؛ محاسبه از پارامترهای حقوق می‌آید."},
	}
	// شرکت‌های همین استقرار از فهرست استقرارها بذر می‌شوند تا فرم ویرایش از ابتدا پر
	// باشد و افزودن شرکت تازه کنار دادهٔ موجود بنشیند.
	return append(out, CompanySeed()...)
}

// mgmtCatalog یک قلم کاتالوگ مدیریت.
func mgmtCatalog(key, titleFa, noteFa string) contracts.RecordInput {
	return catalogRecord(contracts.CatalogManagements, key, titleFa, noteFa, nil)
}

// bankCatalog یک قلم کاتالوگ بانک عامل.
func bankCatalog(key, titleFa string) contracts.RecordInput {
	return catalogRecord(contracts.CatalogBanks, key, titleFa, "", nil)
}

// unitCatalog یک قلم کاتالوگ واحد سازمانی زیر یک مدیریت.
func unitCatalog(key, titleFa, management string) contracts.RecordInput {
	return catalogRecord(contracts.CatalogUnits, key, titleFa, "", map[string]string{"management": management})
}

// catalogRecord رکورد کاتالوگ را با وضعیت «فعال» می‌سازد تا از همان ابتدا در فرم‌ها
// مصرف شود. SeedStatus فقط در دادهٔ نمونه اعمال می‌شود و از API قابل تنظیم نیست.
func catalogRecord(kind, key, titleFa, noteFa string, extra map[string]string) contracts.RecordInput {
	fields := map[string]string{"key": key}
	for k, v := range extra {
		fields[k] = v
	}
	return contracts.RecordInput{Kind: kind, Title: titleFa, SeedStatus: "active", Fields: fields, NoteFa: noteFa}
}
