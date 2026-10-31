// Package domain دامنهٔ ماژول حقوق و دستمزد (جبران خدمات).
//
// دورهٔ پردازش حقوق، قلم حقوق، پارامتر حقوق و ضرایب، و کارانه هر یک یک «نوع رکورد» با
// گردش‌کار خودش است. هیچ مبلغ یا ضریبی در کد هاردکد نیست: پارامترها و ضرائب به‌صورت
// رکورد «پارامتر حقوق» نگه‌داری می‌شوند و اقلام حقوق با فرمول/درصد/مبلغ به آن‌ها ارجاع
// می‌دهند. موتور مشترک `recordkit` گردش‌کار و اعتبارسنجی را اجرا می‌کند و قاعدهٔ
// اختصاصی هر نوع رکورد در همین بسته اعلام می‌شود.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// دستهٔ قلم حقوق و روش محاسبه — مقادیر ماشین‌خوان با برچسب فارسی در guessUI.
const (
	CategoryEarning   = "earning"   // مزایا
	CategoryDeduction = "deduction" // کسور

	CalcFixed   = "fixed"   // مبلغ ثابت
	CalcPercent = "percent" // درصدی از یک مبنا
	CalcFormula = "formula" // فرمول آزاد بر پایهٔ پارامترها
)

// Descriptor شناسهٔ ماژول حقوق و دستمزد.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "payroll",
		TitleFa: "حقوق و دستمزد (جبران خدمات)",
		Kinds: []recordkit.Kind{
			payParamsKind(),
			payItemsKind(),
			payrollRunsKind(),
			karanaKind(),
			serviceBonusKind(),
			// فاز ۱: فایل قانونی بیمه/مالیات، حساب بانکی کارمند و فایل پرداخت بانکی.
			statutoryFilesKind(),
			bankAccountsKind(),
			bankFilesKind(),
		},
	}
}

/* ---------------------------- پارامتر حقوق ---------------------------- */

// payParamsKind پارامترها و ضرائب حقوق: نرخ بیمه، معافیت مالیاتی، حق مسکن، ضریب
// اضافه‌کاری و … . این‌ها «ورودی» محاسبه‌اند، نه عدد ثابت در کد.
func payParamsKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "payParams", TitleFa: "پارامتر و ضریب حقوق", IDPrefix: "PPARAM", EmployeeScoped: false,
		Fields: recordkit.Fields(
			"key|کلید پارامتر|text|1",
			"value|مقدار|text|1",
			"unitFa|واحد|text|0",
			"groupFa|گروه|text|0",
			"effectiveFrom|تاریخ اجرا|date|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|جاری", "inactive|غیرفعال"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|active", "active|inactive", "inactive|active"),
		Actions: recordkit.Actions(
			"activate|جاری‌سازی|primary|active|0",
			"deactivate|غیرفعال‌سازی|ghost|inactive|0",
		),
		Validate: validatePayParam,
		// پارامتر، دادهٔ پیکربندی است نه سندِ گردش‌کاری؛ مثل کاتالوگ‌های مرجع در هر
		// وضعیتی قابل ویرایش است (تغییر ضریب نباید پشت وضعیت بماند) و گردش فعال/
		// غیرفعال فقط می‌گوید همین نسخه در محاسبهٔ حقوق مؤثر است یا نه.
		EditableStatuses: []string{"draft", "active", "inactive"},
	}
}

// validatePayParam کلید پارامتر را برمی‌رسد: باید لاتین نقطه‌دار و بدون فاصله باشد تا
// در فرمول اقلام حقوق با اطمینان ارجاع شود.
func validatePayParam(in contracts.RecordInput) error {
	key := strings.TrimSpace(in.Fields["key"])
	if key == "" {
		return &recordkit.ValidationError{Details: map[string]string{"fields.key": "کلید پارامتر الزامی است."}}
	}
	for _, c := range key {
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_'
		if !ok {
			return &recordkit.ValidationError{Details: map[string]string{
				"fields.key": "کلید پارامتر فقط می‌تواند حرف لاتین، رقم، نقطه و زیرخط داشته باشد (مثل insurance.employeeRate).",
			}}
		}
	}
	value := strings.TrimSpace(in.Fields["value"])
	if value == "" {
		return &recordkit.ValidationError{Details: map[string]string{"fields.value": "مقدار پارامتر الزامی است."}}
	}
	// مقدار باید با واحدش بخواند: واحدهای عددی باید رقمی باشند و واحد «پله» فهرست
	// «سقف:نرخ»؛ ارقام فارسی پیش از سنجش یکسان‌سازی می‌شوند. واحدِ ناشناخته آزاد
	// است تا متن‌های توصیفی سازمانی پذیرفته شوند.
	unit := strings.TrimSpace(in.Fields["unitFa"])
	normalized := calendar.NormalizeDigits(value)
	badValue := func(msg string) error {
		return &recordkit.ValidationError{Details: map[string]string{"fields.value": msg}}
	}
	switch unit {
	case "ریال", "نسبت", "برابر", "درصد", "ساعت", "روز":
		if _, err := strconv.ParseFloat(normalized, 64); err != nil {
			return badValue("مقدار پارامتر با واحد «" + unit + "» باید عددی باشد.")
		}
	case "پله":
		invalid := false
		for _, part := range strings.Split(normalized, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			pair := strings.SplitN(part, ":", 2)
			if len(pair) != 2 {
				invalid = true
				break
			}
			if _, err := strconv.ParseFloat(strings.TrimSpace(pair[0]), 64); err != nil {
				invalid = true
				break
			}
			if _, err := strconv.ParseFloat(strings.TrimSpace(pair[1]), 64); err != nil {
				invalid = true
				break
			}
		}
		if invalid {
			return badValue("مقدار واحد «پله» باید فهرست «سقف:نرخ» جداشده با کاما باشد (مثل 50000000:0.10).")
		}
	}
	return nil
}

/* ------------------------------ قلم حقوق ------------------------------ */

// payItemsKind اقلام حقوق: هر قلم یک «اجزای حقوق» است که یا مزایا است یا کسور؛ محاسبهٔ
// آن می‌تواند مبلغ ثابت، درصدی از یک مبنا، یا فرمول بر پایهٔ پارامترها باشد.
func payItemsKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "payItems", TitleFa: "قلم حقوق", IDPrefix: "ITEM", EmployeeScoped: false,
		Fields: recordkit.Fields(
			"code|کد قلم|text|1",
			"category|نوع (earning مزایا / deduction کسور)|text|1",
			"calc|روش محاسبه (fixed ثابت / percent درصدی / formula فرمولی)|text|1",
			"rate|مقدار یا ضریب (عدد یا کلید پارامتر)|text|0",
			"basis|مبنا (base/gross/housing/karana/constant)|text|0",
			"formula|فرمول محاسبه|text|0",
			"taxable|مشمول مالیات (بله/خیر)|text|0",
			"insured|مشمول بیمه (بله/خیر)|text|0",
			"order|ترتیب نمایش|number|0",
			"note|توضیح|text|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال", "inactive|غیرفعال"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|active", "active|inactive", "inactive|active"),
		Actions: recordkit.Actions(
			"activate|فعال‌سازی|primary|active|0",
			"deactivate|غیرفعال‌سازی|ghost|inactive|0",
		),
		Validate: validatePayItem,
	}
}

// validatePayItem قاعدهٔ محاسبهٔ یک قلم حقوق را برمی‌رسد: هر روش محاسبه ورودی خودش را
// لازم دارد؛ این‌گونه امکان ندارد قلم «درصدی» بدون مبنا ثبت شود.
func validatePayItem(in contracts.RecordInput) error {
	f := in.Fields
	category := strings.TrimSpace(f["category"])
	if !isOneOf(category, CategoryEarning, CategoryDeduction, "مزایا", "کسور") {
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.category": "نوع قلم باید earning (مزایا) یا deduction (کسور) باشد.",
		}}
	}
	calc := strings.TrimSpace(f["calc"])
	switch calc {
	case CalcFixed:
		if strings.TrimSpace(f["rate"]) == "" {
			return &recordkit.ValidationError{Details: map[string]string{"fields.rate": "برای محاسبهٔ مبلغ ثابت، مقدار الزامی است."}}
		}
	case CalcPercent:
		if strings.TrimSpace(f["rate"]) == "" {
			return &recordkit.ValidationError{Details: map[string]string{"fields.rate": "برای محاسبهٔ درصدی، ضریب الزامی است."}}
		}
		if strings.TrimSpace(f["basis"]) == "" {
			return &recordkit.ValidationError{Details: map[string]string{"fields.basis": "برای محاسبهٔ درصدی، مبنا (base/gross/housing/karana) الزامی است."}}
		}
	case CalcFormula:
		if strings.TrimSpace(f["formula"]) == "" {
			return &recordkit.ValidationError{Details: map[string]string{"fields.formula": "برای محاسبهٔ فرمولی، فرمول الزامی است."}}
		}
	default:
		return &recordkit.ValidationError{Details: map[string]string{
			"fields.calc": "روش محاسبه باید fixed (ثابت)، percent (درصدی) یا formula (فرمولی) باشد.",
		}}
	}
	return nil
}

/* ---------------------------- دورهٔ پردازش ---------------------------- */

func payrollRunsKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "payrollRuns", TitleFa: "دورهٔ پردازش حقوق", IDPrefix: "RUN", EmployeeScoped: false,
		Fields: recordkit.Fields(
			"period|دوره|text|1",
			"gross|جمع ناخالص|money|0",
			"tax|مالیات|money|0",
			"insurance|بیمه|money|0",
			"net|خالص پرداختی|money|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "calculated|محاسبه‌شده", "approved|تأییدشده", "locked|قفل‌شده", "paid|پرداخت‌شده"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|calculated", "calculated|approved", "approved|locked,paid", "locked|paid"),
		Actions: recordkit.Actions(
			"calculate|اجرای محاسبه|primary|calculated|0",
			"approve|تأیید دوره|primary|approved|0",
			"lock|قفل دوره|ghost|locked|0",
			"pay|ثبت پرداخت|primary|paid|0",
		),
	}
}

/* ------------------------------- کارانه ------------------------------- */

func karanaKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "karana", TitleFa: "کارانه", IDPrefix: "KRN", EmployeeScoped: true,
		Fields: recordkit.Fields(
			"period|دوره|text|1",
			"amount|مبلغ کارانه|money|0",
			"basis|مبنای محاسبه|text|0",
		),
		Statuses:    recordkit.Statuses("calculated|محاسبه‌شده", "approved|تأیید مدیر واحد", "sent|ارسال به آمار", "paid|پرداخت‌شده"),
		Initial:     "calculated",
		Transitions: recordkit.Steps("calculated|approved", "approved|sent", "sent|paid"),
		Actions: recordkit.Actions(
			"approve|تأیید مدیر واحد|primary|approved|0",
			"send|ارسال به واحد آمار|primary|sent|0",
			"pay|ثبت پرداخت|primary|paid|0",
		),
	}
}

/* -------------------- پاداش پایان خدمت (پاداش سنوات) -------------------- */

// serviceBonusKind پروندهٔ محاسبهٔ پاداش پایان خدمت کارکنان رسمی.
//
// هر پرونده، یک صورت‌حساب تأییدشدنی است: سابقهٔ شمال و جنوب، پایهٔ حقوق، ترمیم حقوق،
// ضرایب به‌کاررفته و مبلغ نهایی. خودِ محاسبه در `service_bonus.go` همین بسته انجام
// می‌شود و این نوع رکورد فقط نتیجهٔ محاسبه‌شده را نگه می‌دارد تا سابقه، تأیید و گزارش
// داشته باشد — با همان گردش‌کار و همان پروسیژرهای ایجاد/ویرایش/جستجو/حذف بقیهٔ انواع.
func serviceBonusKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "serviceBonusFiles", TitleFa: "پروندهٔ پاداش پایان خدمت", IDPrefix: "SBF", EmployeeScoped: true,
		Fields: recordkit.Fields(
			"baseDate|تاریخ مبنای محاسبه|date|1",
			"baseSalary|حقوق پایه (پایه)|money|1",
			"salaryRepair|ترمیم حقوق|money|0",
			"northTenure|سابقه شمال (سال اعشاری)|number|0",
			"southTenure|سابقه جنوب (سال اعشاری)|number|0",
			"totalTenure|جمع سابقه (سال اعشاری)|number|0",
			"degreeKey|مدرک تحصیلی|text|0",
			"badClimate|ضریب بدی آب و هوا|number|0",
			"special|ضریب ویژه|number|0",
			"workshop|ضریب کارگاهی|number|0",
			"attractionPublic|ضریب جذب عمومی|number|0",
			"personalBase|ضریب جذب پایه شخصی|number|0",
			"jobSpecialty|ضریب تخصصی شغل (مدرک)|number|0",
			"attractionBonus|پاداش حق جذب ۸ درصد|money|0",
			"personalBonus|پاداش پایه شخصی و تخصصی|money|0",
			"total|مبلغ پاداش سنوات|money|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده", "paid|پرداخت‌شده"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|approved", "approved|paid"),
		Actions: recordkit.Actions(
			"approve|تأیید پرونده|primary|approved|0",
			"pay|ثبت پرداخت|primary|paid|0",
		),
		// سابقه، پایه و ترمیم حقوق اجباری‌اند؛ بقیهٔ ضرایب از پارامترهای شرکت می‌آیند و
		// قابل بازنویسی‌اند. اعتبارسنجی در همین بسته انجام می‌شود تا قاعده در UI نرود.
		Validate:         validateServiceBonusFile,
		EditableStatuses: []string{"draft"},
	}
}

// validateServiceBonusFile قاعدهٔ دامنهٔ پروندهٔ پاداش پایان خدمت.
func validateServiceBonusFile(in contracts.RecordInput) error {
	f := in.Fields
	if strings.TrimSpace(f["baseDate"]) == "" {
		return &recordkit.ValidationError{Details: map[string]string{"fields.baseDate": "تاریخ مبنای محاسبه الزامی است."}}
	}
	if parseMoney(f["baseSalary"]) <= 0 {
		return &recordkit.ValidationError{Details: map[string]string{"fields.baseSalary": "حقوق پایه باید بزرگ‌تر از صفر باشد."}}
	}
	north, south := parseMoney(f["northTenure"]), parseMoney(f["southTenure"])
	if north <= 0 && south <= 0 {
		return &recordkit.ValidationError{Details: map[string]string{"fields.northTenure": "دست‌کم یکی از سابقهٔ شمال یا جنوب باید بزرگ‌تر از صفر باشد."}}
	}
	if parseMoney(f["total"]) < 0 {
		return &recordkit.ValidationError{Details: map[string]string{"fields.total": "مبلغ پاداش نمی‌تواند منفی باشد."}}
	}
	return nil
}

// parseMoney مقدار عددی ریالی یک فیلد را می‌خواند (فارسی/خالی بی‌خطا صفر می‌شود).
func parseMoney(raw string) float64 {
	s := strings.TrimSpace(calendar.NormalizeDigits(raw))
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

/* -------------------------------- بذرها -------------------------------- */

// Seed پارامترها، اقلام حقوق، دوره‌ها و کارانهٔ نمونه.
func Seed() []contracts.RecordInput {
	return append(append(payParamSeeds(), payItemSeeds()...), []contracts.RecordInput{
		{Kind: "payrollRuns", Title: "پردازش حقوق مهر ۱۴۰۴",
			Fields: map[string]string{"period": "1404/07", "gross": "4820000000", "tax": "312000000", "insurance": "337400000", "net": "4170600000"},
			NoteFa: "محاسبه انجام و آمادهٔ تأیید"},
		{Kind: "payrollRuns", Title: "پردازش حقوق شهریور ۱۴۰۴",
			Fields: map[string]string{"period": "1404/06", "gross": "4715000000", "tax": "301000000", "insurance": "330050000", "net": "4083950000"},
			NoteFa: "پرداخت‌شده"},
		{Kind: "karana", Title: "کارانه شهریور — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"period": "1404/06", "amount": "18500000", "basis": "بهره‌وری خط تولید"}},

		// پروندهٔ پاداش پایان خدمت: نمونهٔ تأییدشده و پیش‌نویس، تا گردش‌کار و گزارش‌های
		// این بخش روی دادهٔ واقعی گراف کار کند.
		{Kind: "serviceBonusFiles", Title: "پاداش پایان خدمت — محمد احمدی", EmployeeID: "EMP-005",
			SeedStatus: "approved",
			Fields: map[string]string{
				"baseDate": "1404/07/01", "baseSalary": "246000000", "salaryRepair": "0",
				"northTenure": "2.17", "southTenure": "34.54", "totalTenure": "36.71",
				"badClimate": "0.25", "special": "0.30", "workshop": "0.10",
				"attractionPublic": "0.08", "personalBase": "0.38", "jobSpecialty": "0.27",
				"attractionBonus": "0", "personalBonus": "0", "total": "0",
			},
			NoteFa: "نمونهٔ پرونده؛ مبالغ با دکمهٔ «محاسبه» در فرم پر می‌شود"},
		{Kind: "serviceBonusFiles", Title: "پاداش پایان خدمت — سارا محمدی", EmployeeID: "EMP-001",
			Fields: map[string]string{
				"baseDate": "1404/07/01", "baseSalary": "318000000", "salaryRepair": "0",
				"northTenure": "0", "southTenure": "12.50", "totalTenure": "12.50",
				"badClimate": "0.25", "special": "0.30", "workshop": "0.10",
				"attractionPublic": "0.08", "personalBase": "0.38", "jobSpecialty": "0.27",
				"attractionBonus": "0", "personalBonus": "0", "total": "0",
			},
			NoteFa: "نمونهٔ پیش‌نویس"},
	}...)
}

// payParamSeeds ضرائب و پارامترهای پایهٔ حقوق (قابل ویرایش در صفحهٔ پارامترهای حقوق).
func payParamSeeds() []contracts.RecordInput {
	// پارامترهای نمونه «جاری» بذر می‌شوند چون پیش‌نویس در محاسبهٔ حقوق اثر ندارد؛
	// دادهٔ نمایشی باید از همان راهی اعمال شود که دادهٔ کاربر ثبت‌شده.
	p := func(title, key, value, unit, group string) contracts.RecordInput {
		return contracts.RecordInput{Kind: "payParams", Title: title, SeedStatus: "active",
			Fields: map[string]string{"key": key, "value": value, "unitFa": unit, "groupFa": group}}
	}
	return []contracts.RecordInput{
		p("نرخ بیمهٔ سهم کارمند", "insurance.employeeRate", "0.07", "نسبت", "بیمه"),
		p("نرخ بیمهٔ سهم کارفرما", "insurance.employerRate", "0.23", "نسبت", "بیمه"),
		p("سقف بیمهٔ ماهانه", "insurance.ceiling", "200000000", "ریال", "بیمه"),
		p("معافیت مالیاتی ماهانه", "tax.exemption", "24000000", "ریال", "مالیات"),
		p("پله‌های مالیات (سقف:نرخ)", "tax.brackets", "50000000:0.10,100000000:0.15,200000000:0.20,0:0.30", "پله", "مالیات"),
		p("حق مسکن ماهانه", "allowance.housing", "9000000", "ریال", "مزایا"),
		p("حق اولاد هر فرزند", "allowance.child", "10390968", "ریال", "مزایا"),
		p("ضریب اضافه‌کاری", "overtime.factor", "1.4", "برابر", "مزایا"),
		p("ضریب نوبت‌کاری", "nightShift.factor", "1.35", "برابر", "مزایا"),
		p("ضریب جمعه‌کاری", "friday.factor", "1.4", "برابر", "مزایا"),
		p("ضریب تعطیل‌کاری", "holiday.factor", "1.4", "برابر", "مزایا"),
		p("ضریب سختی شرایط کار", "hardship.coefficient", "0.15", "نسبت", "مزایا"),
		p("ضریب بهره‌وری", "productivity.coefficient", "0.20", "نسبت", "کارانه"),
		p("مبلغ پایهٔ پیشنهادی", "base.salary.default", "120000000", "ریال", "حقوق پایه"),

		// اجزای «مزد»: جمع این‌ها مزد مبنا را می‌سازد و فوق‌العاده‌ها از آن متأثر می‌شوند.
		p("مزد رتبه", "wage.rank", "0", "ریال", "مزد"),
		p("مزد سنوات", "wage.seniority", "0", "ریال", "مزد"),
		p("فوق‌العادهٔ پست", "allowance.post", "0", "ریال", "مزد"),
		p("فوق‌العادهٔ ایثارگری", "allowance.sacrifice", "0", "ریال", "مزد"),

		// فوق‌العاده‌های منطقه‌ای و شغلی (بر پایهٔ فایل «فوق‌العاده‌های حقوق»).
		p("ضریب بدی آب و هوا", "allowance.badClimate", "0.25", "نسبت", "فوق‌العاده‌ها"),
		p("ضریب محرومیت از تسهیلات زندگی", "allowance.deprivation", "0.25", "نسبت", "فوق‌العاده‌ها"),
		p("ضریب صعوبت شرایط منطقه", "allowance.regionHardship", "0.30", "نسبت", "فوق‌العاده‌ها"),
		p("ضریب جبران صرف وقت اضافی", "allowance.extraTime", "0.15", "نسبت", "فوق‌العاده‌ها"),
		p("درصد فوق‌العادهٔ شغل", "allowance.jobPercent", "0.5", "نسبت", "فوق‌العاده‌ها"),
		p("مبلغ فوق‌العادهٔ جذب", "allowance.attraction", "95964628", "ریال", "فوق‌العاده‌ها"),
		p("حق تاهل", "allowance.marriage", "5000000", "ریال", "فوق‌العاده‌ها"),
		p("بن کارگری", "allowance.grocery", "22000000", "ریال", "فوق‌العاده‌ها"),

		// فوق‌العاده‌های صورت‌حساب مرجع «پاداش پایان خدمت کارکنان رسمی صنعت نفت».
		// این‌ها همان اقلامی‌اند که در حکم و فیش حقوق هم می‌آیند و محاسبهٔ پاداش سنوات
		// مستقیم از همین پارامترها درصد می‌گیرد (نه از عدد سخت‌شده در کد).
		p("ضریب ویژه", "allowance.special", "0.30", "نسبت", "فوق‌العاده‌ها"),
		p("ضریب کارگاهی", "allowance.workshop", "0.10", "نسبت", "فوق‌العاده‌ها"),
		p("ضریب جذب عمومی", "allowance.attractionPublic", "0.08", "نسبت", "فوق‌العاده‌ها"),
		p("ضریب جذب (پایه شخصی)", "allowance.personalBase", "0.38", "نسبت", "فوق‌العاده‌ها"),
		p("ضریب تخصصی شغل (مدرک)", "allowance.jobSpecialty", "0.27", "نسبت", "فوق‌العاده‌ها"),
		p("مبلغ ترمیم حقوق", "allowance.salaryRepair", "0", "ریال", "فوق‌العاده‌ها"),

		// جدول مرجع ضریب مدرک تحصیلی (مبنای تعیین درصد تخصصی شغل در پاداش پایان خدمت).
		p("ضریب مدرک — زیردیپلم", "serviceBonus.degree.belowDiploma", "0.175", "نسبت", "پاداش پایان خدمت"),
		p("ضریب مدرک — دیپلم", "serviceBonus.degree.diploma", "0.175", "نسبت", "پاداش پایان خدمت"),
		p("ضریب مدرک — فوق دیپلم", "serviceBonus.degree.associate", "0.27", "نسبت", "پاداش پایان خدمت"),
		p("ضریب مدرک — لیسانس", "serviceBonus.degree.bachelor", "0.385", "نسبت", "پاداش پایان خدمت"),
		p("ضریب مدرک — فوق لیسانس", "serviceBonus.degree.master", "0.462", "نسبت", "پاداش پایان خدمت"),
		p("ضریب مدرک — دکترا", "serviceBonus.degree.doctorate", "0.54", "نسبت", "پاداش پایان خدمت"),
		p("سقف پرداخت نقدی پاداش پایان خدمت", "serviceBonus.cashCeiling", "0", "ریال", "پاداش پایان خدمت"),
	}
}

// payItemSeeds اقلام حقوق استاندارد صنعت پتروشیمی (کامل‌ترین الگوی مرجع).
func payItemSeeds() []contracts.RecordInput {
	return []contracts.RecordInput{
		// اجزای مزد — جمع این‌ها «مزد مبنا» را می‌سازد و مبناى فوق‌العاده‌های درصدی است.
		payItem("مزد شغل", "WAGE_JOB", CategoryEarning, CalcFixed, "base", "constant", "", "بله", "بله", "1", "سطح مزد شغل؛ از ورودی شبیه‌سازی/پروندهٔ کارمند"),
		payItem("مزد رتبه", "WAGE_RANK", CategoryEarning, CalcFixed, "wage.rank", "constant", "", "بله", "بله", "2", "از پارامتر wage.rank"),
		payItem("مزد سنوات", "WAGE_SENIORITY", CategoryEarning, CalcFixed, "wage.seniority", "constant", "", "بله", "بله", "3", "از پارامتر wage.seniority"),
		payItem("فوق‌العادهٔ پست", "ALLOW_POST", CategoryEarning, CalcFixed, "allowance.post", "constant", "", "بله", "بله", "4", "از پارامتر allowance.post"),
		payItem("فوق‌العادهٔ ایثارگری", "ALLOW_SACRIFICE", CategoryEarning, CalcFixed, "allowance.sacrifice", "constant", "", "بله", "بله", "5", "از پارامتر allowance.sacrifice"),

		// مزایا و فوق‌العاده‌های پایه.
		payItem("حق مسکن", "HOUSING", CategoryEarning, CalcFixed, "allowance.housing", "constant", "", "بله", "بله", "6", "از پارامتر allowance.housing"),
		payItem("بن کارگری", "GROCERY", CategoryEarning, CalcFixed, "allowance.grocery", "constant", "", "خیر", "بله", "7", "از پارامتر allowance.grocery"),
		payItem("حق اولاد", "CHILD", CategoryEarning, CalcFixed, "allowance.child", "constant", "", "خیر", "بله", "8", "به‌ازای هر فرزند؛ از پارامتر allowance.child"),
		payItem("حق تاهل", "MARRIAGE", CategoryEarning, CalcFixed, "allowance.marriage", "constant", "", "خیر", "بله", "9", "از پارامتر allowance.marriage"),

		// فوق‌العاده‌های منطقه‌ای و شغلی — درصدی از مزد مبنا.
		payItem("بدی آب و هوا", "BAD_CLIMATE", CategoryEarning, CalcPercent, "allowance.badClimate", "mazd", "", "بله", "بله", "10", "۲۵٪ مزد مبنا"),
		payItem("محرومیت از تسهیلات زندگی", "DEPRIVATION", CategoryEarning, CalcPercent, "allowance.deprivation", "mazd", "", "بله", "بله", "11", "۲۵٪ مزد مبنا"),
		payItem("صعوبت شرایط منطقه", "REGION_HARDSHIP", CategoryEarning, CalcPercent, "allowance.regionHardship", "mazd", "", "بله", "بله", "12", "۳۰٪ مزد مبنا"),
		payItem("جبران صرف وقت اضافی", "EXTRA_TIME", CategoryEarning, CalcPercent, "allowance.extraTime", "mazd", "", "بله", "بله", "13", "۱۵٪ مزد مبنا"),
		payItem("درصد فوق‌العادهٔ شغل", "JOB_PERCENT", CategoryEarning, CalcPercent, "allowance.jobPercent", "mazd", "", "بله", "بله", "14", "۵۰٪ مزد مبنا"),
		payItem("فوق‌العادهٔ جذب", "ATTRACTION", CategoryEarning, CalcFixed, "allowance.attraction", "constant", "", "بله", "بله", "15", "مبلغ ثابت جذب نیرو"),

		// فوق‌العاده‌های صورت‌حساب مرجع صنعت نفت (در حکم و فیش حقوق و مبنای پاداش سنوات).
		payItem("ویژه", "SPECIAL", CategoryEarning, CalcPercent, "allowance.special", "base", "", "بله", "بله", "15.1", "۳۰٪ پایه حقوق"),
		payItem("کارگاهی", "WORKSHOP", CategoryEarning, CalcPercent, "allowance.workshop", "base", "", "بله", "بله", "15.2", "۱۰٪ پایه حقوق"),
		payItem("جذب عمومی", "ATTRACTION_PUBLIC", CategoryEarning, CalcPercent, "allowance.attractionPublic", "base", "", "بله", "بله", "15.3", "۸٪ پایه حقوق"),
		payItem("جذب (پایه شخصی)", "PERSONAL_BASE", CategoryEarning, CalcPercent, "allowance.personalBase", "base", "", "بله", "بله", "15.4", "۳۸٪ پایه حقوق بر پایهٔ مدرک و گروه"),
		payItem("تخصصی شغل (مدرک)", "JOB_SPECIALTY", CategoryEarning, CalcPercent, "allowance.jobSpecialty", "base", "", "بله", "بله", "15.5", "۲۷٪ پایه حقوق؛ مبنای جدول مدرک تحصیلی"),
		payItem("ترمیم حقوق", "SALARY_REPAIR", CategoryEarning, CalcFixed, "allowance.salaryRepair", "constant", "", "بله", "بله", "15.6", "مبلغ ترمیم حقوق مصوب؛ در پاداش سنوات هم افزوده می‌شود"),

		// کار در زمان‌های خاص.
		payItem("اضافه‌کاری", "OVERTIME", CategoryEarning, CalcFormula, "", "", "overtimeHours * hourlyRate * overtime.factor", "بله", "بله", "16", "ساعت × نرخ × ضریب اضافه‌کاری"),
		payItem("نوبت‌کاری", "SHIFT", CategoryEarning, CalcPercent, "nightShift.factor", "mazd", "", "بله", "بله", "17", "ضریب نوبت‌کاری × مزد مبنا"),
		payItem("جمعه‌کاری", "FRIDAY", CategoryEarning, CalcFormula, "", "", "fridayHours * hourlyRate * friday.factor", "بله", "بله", "18", "ساعت جمعه‌کاری × نرخ × ضریب"),
		payItem("تعطیل‌کاری", "HOLIDAY", CategoryEarning, CalcFormula, "", "", "holidayHours * hourlyRate * holiday.factor", "بله", "بله", "19", "ساعت تعطیل‌کاری × نرخ × ضریب"),
		payItem("فوق‌العادهٔ سختی شرایط کار", "HARDSHIP", CategoryEarning, CalcPercent, "hardship.coefficient", "mazd", "", "بله", "بله", "20", "ضریب از پارامترهای سازمانی خوانده می‌شود"),

		// پاداش‌ها و اقلام متغیر.
		payItem("پاداش بهره‌وری", "PRODUCTIVITY", CategoryEarning, CalcPercent, "productivity.coefficient", "karana", "", "بله", "خیر", "21", "درصدی از کارانهٔ دوره"),
		payItem("کارانه", "KARANA", CategoryEarning, CalcFormula, "", "", "approvedKaranaAmount", "بله", "خیر", "22", "کارانهٔ تأییدشدهٔ مدت واحد"),
		payItem("ایاب و ذهاب", "TRANSPORT", CategoryEarning, CalcFixed, "6000000", "constant", "", "خیر", "خیر", "23", "کمک‌هزینهٔ رفت‌وآمد"),
		payItem("حق مأموریت", "MISSION", CategoryEarning, CalcFormula, "", "", "missionDays * mission.perDiem", "خیر", "خیر", "24", "بر پایهٔ فوق‌العادهٔ روزانهٔ مأموریت"),

		// کسور.
		payItem("مساعده", "ADVANCE", CategoryDeduction, CalcFixed, "0", "constant", "", "خیر", "خیر", "25", "کسور اختیاری دوره"),
		payItem("اقساط وام", "LOAN", CategoryDeduction, CalcFixed, "0", "constant", "", "خیر", "خیر", "26", "از پروندهٔ وام کارمند"),
		payItem("بیمهٔ سهم کارمند", "INSURANCE_EE", CategoryDeduction, CalcPercent, "insurance.employeeRate", "gross", "", "خیر", "خیر", "27", "سقف بیمه از پارامتر insurance.ceiling کنترل می‌شود"),
		payItem("مالیات بر درآمد", "TAX", CategoryDeduction, CalcFormula, "", "", "taxableIncome * taxRate (پله‌ای)", "خیر", "خیر", "28", "پله‌های مالیات از پارامتر tax.brackets"),
	}
}

// payItem سازندهٔ کمکی یک قلم حقوق؛ فیلدهای خالی وارد نمی‌شوند.
func payItem(title, code, category, calc, rate, basis, formula, taxable, insured, order, note string) contracts.RecordInput {
	f := map[string]string{"code": code, "category": category, "calc": calc, "taxable": taxable, "insured": insured, "order": order}
	for k, v := range map[string]string{"rate": rate, "basis": basis, "formula": formula, "note": note} {
		if strings.TrimSpace(v) != "" {
			f[k] = v
		}
	}
	return contracts.RecordInput{Kind: "payItems", Title: title, Fields: f}
}

func isOneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}
