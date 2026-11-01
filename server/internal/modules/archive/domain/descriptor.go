// Package domain دامنهٔ ماژول بایگانی و شناسایی کارکنان.
//
// قواعد این ماژول به‌صورت «شناسهٔ رکورد» اعلام می‌شود و موتور مشترک `recordkit` آن را
// اجرا می‌کند؛ پس پروندهٔ پرسنلی، مشخصات هویتی، اسناد بایگانی، **سابقهٔ تحصیلی** و
// **مدرک ملاک حقوق** هر یک یک «نوع رکورد» با فیلدها، وضعیت‌ها، انتقال‌ها و اقدام‌های
// خودش است.
//
// دو نوع رکورد تحصیلی عمداً جدا هستند:
//   - `educations` سابقهٔ تحصیلی است: مقطع، رشته، گرایش، دانشگاه، ارتباط تحصیلات با شغل و
//     نتیجهٔ شورای تعدیل مدرک.
//   - `degreeBases` تصمیم «کدام مدرک ملاک حقوق است» است: فرم مخصوص خودش را دارد، تاریخ
//     اجرا و صورت‌جلسهٔ شورا می‌خواهد و می‌تواند برای ایثارگران یک مقطع بالاتر باشد.
//
// جدا بودنشان یعنی تغییر تصمیم ملاک حقوق، سابقهٔ تحصیلی را دست‌کاری نمی‌کند و تاریخچهٔ
// هر دو در پروندهٔ کارمند می‌ماند.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// شناسهٔ نوع‌های رکورد تحصیلی این ماژول.
const (
	KindEducations  = "educations"
	KindDegreeBases = "degreeBases"
)

// وضعیت‌های سابقهٔ تحصیلی.
const (
	EducationDraft    = "draft"
	EducationVerified = "verified"
	EducationArchived = "archived"
)

// وضعیت‌های مدرک ملاک حقوق.
const (
	BasisDraft    = "draft"
	BasisApproved = "council_approved"
	BasisActive   = "active"
	BasisReplaced = "superseded"
	BasisRejected = "rejected"
)

// Descriptor شناسهٔ ماژول بایگانی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "archive",
		TitleFa: "بایگانی و شناسایی کارکنان",
		Kinds: []recordkit.Kind{
			{
				Key: "personnelFiles", TitleFa: "پروندهٔ پرسنلی", IDPrefix: "FILE", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"fileNo|شمارهٔ پرونده|text|1",
					"docsCount|تعداد مدارک|number|0",
					"lastAudit|آخرین کنترل کامل بودن|date|0",
				),
				Statuses: recordkit.Statuses(
					"incomplete|ناقص", "complete|کامل", "archived|بایگانی‌شده",
				),
				Initial:     "incomplete",
				Transitions: recordkit.Steps("incomplete|complete,archived", "complete|archived"),
				Actions: recordkit.Actions(
					"complete|تأیید کامل بودن پرونده|primary|complete|0",
					"archive|بایگانی پرونده|ghost|archived|1",
				),
			},
			{
				Key: "employeeIdentity", TitleFa: "مشخصات هویتی", IDPrefix: "IDNT", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"nationalId|کد ملی|text|1",
					"birthDate|تاریخ تولد|date|0",
					"maritalStatus|وضعیت تأهل|select|0",
					"idCardNo|شمارهٔ کارت شناسایی|text|0",
					"cardExpiry|انقضای کارت|date|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "issued|کارت صادرشده", "expired|منقضی",
				),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|issued", "issued|expired"),
				Actions: recordkit.Actions(
					"issue|صدور کارت شناسایی|primary|issued|0",
					"expire|ثبت انقضا|ghost|expired|0",
				),
			},
			{
				Key: "archiveDocuments", TitleFa: "سند بایگانی و امانت", IDPrefix: "DOC", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"docType|نوع سند|text|1",
					"docNo|شمارهٔ سند|text|0",
					"lendDate|تاریخ امانت|date|0",
					"returnDate|تاریخ بازگشت|date|0",
				),
				Statuses: recordkit.Statuses(
					"stored|در بایگانی", "lent|امانت", "returned|بازگشتی",
				),
				Initial:     "stored",
				Transitions: recordkit.Steps("stored|lent", "lent|returned"),
				Actions: recordkit.Actions(
					"lend|ثبت امانت|primary|lent|0",
					"return|ثبت بازگشت|primary|returned|0",
				),
			},
			{
				Key: KindEducations, TitleFa: "سابقهٔ تحصیلی", IDPrefix: "EDU", EmployeeScoped: true,
				Fields: recordkit.Fields(
					// مقطع از کاتالوگ مشترک contracts (دیپلم تا فوق دکتری)؛ گزینه‌ها از
					// /api/v1/archive/options می‌آید تا در رابط ساخته نشود.
					"level|مقطع تحصیلی|select|1",
					"field|رشتهٔ تحصیلی|text|1",
					"orientation|گرایش|text|0",
					"university|دانشگاه / مؤسسه|text|0",
					"graduationDate|تاریخ فراغت از تحصیل|date|1",
					"diplomaNo|شمارهٔ دانشنامه / گواهی|text|0",
					"average|معدل کل|number|0",
					// ارتباط تحصیلات با شغل: مبنای «مرتبط یا غیرمرتبط بودن» و درصد ارتباط.
					// اجباری‌بودنش در اقدام «تأیید مدارک» سنجیده می‌شود، نه در لحظهٔ ثبت:
					// کارشناس نخست مدرک را ثبت می‌کند و در مرحلهٔ بررسی، ارتباط را ارزیابی
					// می‌کند؛ همین قاعده در دامنه (educationRule) اعلام شده است.
					"relevance|ارتباط تحصیلات با شغل|select|0",
					"relevancePercent|درصد ارتباط با شغل|number|0",
					// نتیجهٔ شورای تعدیل مدرک روی خودِ مدرک می‌ماند؛ تصمیم «ملاک حقوق» در
					// نوع رکورد جداگانه (degreeBases) ثبت می‌شود.
					"councilStatus|وضعیت شورای تعدیل مدرک|select|0",
					"councilNo|شمارهٔ صورت‌جلسهٔ شورا|text|0",
					"councilDate|تاریخ صورت‌جلسهٔ شورا|date|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "verified|مدارک تأییدشده", "archived|بایگانی‌شده",
				),
				Initial: EducationDraft,
				Transitions: recordkit.Steps(
					"draft|verified",
					"verified|archived",
				),
				Actions: recordkit.Actions(
					"verify|تأیید مدارک تحصیلی|primary|verified|0",
					"archive|بایگانی مدرک|ghost|archived|1",
				),
				// تاریخ مؤثر سابقهٔ تحصیلی، تاریخ فراغت از تحصیل است.
				HistoryDate:      "graduationDate",
				EditableStatuses: []string{EducationDraft, EducationVerified},
				ActRule:          educationRule,
			},
			{
				Key: KindDegreeBases, TitleFa: "مدرک ملاک حقوق", IDPrefix: "DBAS", EmployeeScoped: true,
				Fields: recordkit.Fields(
					// فرم مخصوص خودش: کدام مدرک، با چه ارتباطی، به تصویب کدام صورت‌جلسه و
					// از چه تاریخی مبنای حقوق است؛ و برای ایثارگران یک مقطع بالاتر.
					// کد سابقهٔ تحصیلی ملاک: اگر ثبت شود، تصمیم به همان رکورد گره می‌خورد؛
					// ولی دادهٔ قدیمی ممکن است فقط مقطع و رشته را داشته باشد، پس اختیاری است
					// و تطبیق با مقطع+رشته هم پذیرفته می‌شود (basisMatchesRecord).
					"education|کد سابقهٔ تحصیلی ملاک|text|0",
					"level|مقطع مدرک ملاک|select|1",
					"field|رشتهٔ مدرک ملاک|text|1",
					"orientation|گرایش مدرک ملاک|text|0",
					"relevance|ارتباط مدرک با شغل|select|1",
					"relevancePercent|درصد ارتباط مدرک|number|0",
					// شماره و تاریخ صورت‌جلسه در لحظهٔ تأیید شورا اجباری می‌شوند (نه در
					// ثبت پیش‌نویسِ فرم): قاعده‌اش در degreeBasisRule است.
					"councilNo|شمارهٔ صورت‌جلسهٔ شورای تعدیل|text|0",
					"councilDate|تاریخ صورت‌جلسهٔ شورای تعدیل|date|0",
					"effectiveDate|تاریخ اجرا|date|1",
					"salaryGroup|گروه شغلی مبنای حقوق|text|0",
					"veteranUpgrade|ارتقای یک مقطع (ایثارگران)|select|1",
					"upgradedLevel|مقطع معادل‌سازی‌شده|select|0",
					"salaryEffect|اثر حقوقی ملاک|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس",
					"council_approved|تأییدشدهٔ شورای تعدیل مدرک",
					"active|ملاک جاری حقوق",
					"superseded|جانشین‌شده",
					"rejected|ردشده",
				),
				Initial: BasisDraft,
				Transitions: recordkit.Steps(
					"draft|council_approved,rejected",
					"council_approved|active,rejected",
					"active|superseded",
				),
				Actions: recordkit.Actions(
					"approve|تأیید شورای تعدیل مدرک|primary|council_approved|0",
					"activate|جاری‌سازی به‌عنوان ملاک حقوق|primary|active|0",
					"supersede|جانشینی با مدرک تازه|ghost|superseded|1",
					"reject|رد مدرک ملاک|danger|rejected|1",
				),
				HistoryDate: "effectiveDate",
				// ویرایش در پیش‌نویس و پس از تأیید شورا مجاز است: کارشناس پس از صورت‌جلسه،
				// مقطع معادل‌سازی‌شده و گروه شغلی مبنای پرداخت را کامل می‌کند و بعد جاری می‌سازد.
				EditableStatuses: []string{BasisDraft, BasisApproved},
				ActRule:          degreeBasisRule,
			},
		},
	}
}

// educationRule قاعدهٔ اقدام‌های سابقهٔ تحصیلی.
//
// تأیید مدارک تحصیلی بدون ثبت نتیجهٔ بررسی ارتباط و درصد آن معنا ندارد؛ چون همین دو
// فیلد مبنای تصمیم «ملاک حقوق» می‌شوند و بعداً در فرم شورای تعدیل مدرک استفاده می‌شوند.
func educationRule(rec recordkit.Record, action string) error {
	if action != "verify" {
		return nil
	}
	if strings.TrimSpace(rec.Fields["relevance"]) == "" {
		return &recordkit.RuleError{Message: "تأیید مدارک تحصیلی بدون ثبت «ارتباط تحصیلات با شغل» ممکن نیست؛ " +
			"مرتبط یا غیرمرتبط بودن مدرک را مشخص کنید."}
	}
	if percent := strings.TrimSpace(rec.Fields["relevancePercent"]); percent == "" &&
		strings.TrimSpace(rec.Fields["relevance"]) == "partially_related" {
		return &recordkit.RuleError{Message: "برای مدرک «مرتبط نسبی»، ثبت «درصد ارتباط با شغل» الزامی است " +
			"تا میزان اثر آن در ملاک حقوق روشن باشد."}
	}
	return nil
}

// degreeBasisRule قاعدهٔ اقدام‌های «مدرک ملاک حقوق».
//
// سه شرط، همه از ضوابط سازمانی:
//  1. تأیید شورای تعدیل مدرک تنها با شماره و تاریخ صورت‌جلسه ممکن است.
//  2. جاری‌سازی (ملاک حقوق شدن) مدرکی که «غیرمرتبط» ارزیابی شده مجاز نیست.
//  3. برای ایثارگران، مقطع معادل‌سازی‌شده (یک مقطع بالاتر) باید ثبت شده باشد تا مبنای
//     پرداخت روشن و قابل ردیابی بماند.
func degreeBasisRule(rec recordkit.Record, action string) error {
	switch action {
	case "approve":
		if strings.TrimSpace(rec.Fields["councilNo"]) == "" || strings.TrimSpace(rec.Fields["councilDate"]) == "" {
			return &recordkit.RuleError{Message: "تأیید مدرک ملاک تنها با شماره و تاریخ صورت‌جلسهٔ " +
				"شورای تعدیل مدرک ممکن است."}
		}
		if strings.TrimSpace(rec.Fields["relevance"]) == "" {
			return &recordkit.RuleError{Message: "پیش از تأیید شورا، «ارتباط مدرک با شغل» و درصد آن را ثبت کنید."}
		}
	case "activate":
		if !contracts.DegreeRelated(rec.Fields["relevance"]) {
			return &recordkit.RuleError{Message: "مدرکی که ارتباط آن با شغل «غیرمرتبط» ارزیابی شده است، " +
				"نمی‌تواند ملاک حقوق شود."}
		}
		if strings.TrimSpace(rec.Fields["effectiveDate"]) == "" {
			return &recordkit.RuleError{Message: "برای جاری‌سازی ملاک حقوق، ثبت «تاریخ اجرا» الزامی است."}
		}
		if strings.TrimSpace(rec.Fields["veteranUpgrade"]) == "yes" &&
			strings.TrimSpace(rec.Fields["upgradedLevel"]) == "" {
			return &recordkit.RuleError{Message: "برای ایثارگران، «مقطع معادل‌سازی‌شده» (یک مقطع بالاتر) " +
				"باید ثبت شود تا مبنای پرداخت روشن باشد."}
		}
	}
	return nil
}

// Seed رکوردهای نمونهٔ سازمان برای نمایش اولیهٔ صفحه‌ها.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "personnelFiles", Title: "پروندهٔ پرسنلی — سارا محمدی", EmployeeID: "EMP-001",
			Fields: map[string]string{"fileNo": "FILE-1001", "docsCount": "42", "lastAudit": "1404/06/10"},
			NoteFa: "پرونده کامل و کنترل‌شده"},
		{Kind: "personnelFiles", Title: "پروندهٔ پرسنلی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"fileNo": "FILE-1003", "docsCount": "17"},
			NoteFa: "مدارک تحصیلی ناقص است"},
		{Kind: "employeeIdentity", Title: "مشخصات هویتی — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"nationalId": "0071234567", "birthDate": "1365/04/02", "maritalStatus": "متأهل", "idCardNo": "ID-55210", "cardExpiry": "1406/04/02"}},
		{Kind: "employeeIdentity", Title: "مشخصات هویتی — مریم نوری", EmployeeID: "EMP-008",
			Fields: map[string]string{"nationalId": "0089876543", "birthDate": "1370/11/19", "maritalStatus": "مجرد"}},
		{Kind: "archiveDocuments", Title: "امانت مدرک تحصیلی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"docType": "گواهی موقت تحصیلی", "docNo": "DOC-7781", "lendDate": "1404/07/05"},
			NoteFa: "جهت ارائه به دانشگاه"},
		{Kind: "archiveDocuments", Title: "بایگانی حکم کارگزینی — سارا محمدی", EmployeeID: "EMP-001",
			Fields: map[string]string{"docType": "حکم کارگزینی", "docNo": "DOC-6402"}},

		// سابقهٔ تحصیلی: از دیپلم تا مدرک پایان‌یافته، با ارتباط سنجیده‌شده و نتیجهٔ شورا.
		{Kind: KindEducations, Title: "کارشناسی ارشد حسابداری — علی مرادی", EmployeeID: "EMP-004",
			Fields: map[string]string{
				"level": "master", "field": "حسابداری", "orientation": "حسابداری مالی",
				"university": "دانشگاه شهید چمران اهواز", "graduationDate": "1398/06/30",
				"diplomaNo": "EDU-1398-22417", "average": "17.85",
				"relevance": "related", "relevancePercent": "100",
				"councilStatus": "approved", "councilNo": "SH-1404-118", "councilDate": "1404/06/25",
			},
			SeedStatus: EducationVerified,
			NoteFa:     "مدرک مبنای گروه شغلی کارشناسی ارشد؛ صورت‌جلسهٔ شورای تعدیل در پرونده است"},
		{Kind: KindEducations, Title: "کارشناسی حسابداری — علی مرادی", EmployeeID: "EMP-004",
			Fields: map[string]string{
				"level": "bachelor", "field": "حسابداری", "university": "دانشگاه اهواز",
				"graduationDate": "1395/06/31", "average": "16.40",
				"relevance": "related", "relevancePercent": "100", "councilStatus": "approved",
			},
			SeedStatus: EducationVerified,
			NoteFa:     "مدرک پیشین همان فرد؛ در سابقهٔ تحصیلی می‌ماند"},
		{Kind: KindEducations, Title: "کارشناسی مهندسی صنایع — حسین موسوی", EmployeeID: "EMP-009",
			Fields: map[string]string{
				"level": "bachelor", "field": "مهندسی صنایع", "orientation": "برنامه‌ریزی و تحلیل سیستم‌ها",
				"university": "دانشگاه صنعتی اصفهان", "graduationDate": "1393/06/31",
				"diplomaNo": "EDU-1393-11890", "average": "16.90",
				"relevance": "partially_related", "relevancePercent": "70",
				"councilStatus": "approved", "councilNo": "SH-1404-090", "councilDate": "1404/05/12",
			},
			SeedStatus: EducationVerified,
			NoteFa:     "برای مدیریت تولید «مرتبط نسبی» ارزیابی شده است"},
		{Kind: KindEducations, Title: "کارشناسی بازاریابی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{
				"level": "bachelor", "field": "مدیریت بازرگانی", "orientation": "بازاریابی",
				"university": "دانشگاه آزاد اهواز", "graduationDate": "1400/06/30", "average": "15.20",
				"relevance": "related", "relevancePercent": "100", "councilStatus": "in_review",
			},
			NoteFa: "مدرک در دست بررسی شورای تعدیل مدرک"},

		// تصمیم ملاک حقوق: مدرک کارشناسی ارشد، تأییدشدهٔ شورا و جاری.
		{Kind: KindDegreeBases, Title: "مدرک ملاک حقوق — علی مرادی", EmployeeID: "EMP-004",
			Fields: map[string]string{
				"education": "EDU-0001", "level": "master", "field": "حسابداری",
				"orientation": "حسابداری مالی", "relevance": "related", "relevancePercent": "100",
				"councilNo": "SH-1404-118", "councilDate": "1404/06/25", "effectiveDate": "1404/07/01",
				"salaryGroup": "کارشناسی ارشد (گروه ۹)", "veteranUpgrade": "no",
				"salaryEffect": "مبنای احتساب مزد پایه و گروه شغلی کارشناسی ارشد",
			},
			SeedStatus: BasisActive,
			NoteFa:     "جاری؛ مدرک کارشناسی پیشین به‌عنوان ملاک جانشین‌شده در سابقه می‌ماند"},
		// تصمیم ایثارگران: یک مقطع بالاتر با ثبت مقطع معادل‌سازی‌شده.
		{Kind: KindDegreeBases, Title: "مدرک ملاک حقوق (ایثارگری، یک مقطع بالاتر) — حسین موسوی", EmployeeID: "EMP-009",
			Fields: map[string]string{
				"education": "EDU-0003", "level": "bachelor", "field": "مهندسی صنایع",
				"orientation": "برنامه‌ریزی و تحلیل سیستم‌ها", "relevance": "partially_related", "relevancePercent": "70",
				"councilNo": "SH-1404-090", "councilDate": "1404/05/12", "effectiveDate": "1404/06/01",
				"salaryGroup":    "کارشناسی ارشد (معادل ایثارگری)",
				"veteranUpgrade": "yes", "upgradedLevel": "master",
				"salaryEffect": "معادل‌سازی یک مقطع بالاتر بر پایهٔ ضوابط ایثارگران",
			},
			SeedStatus: BasisActive,
			NoteFa:     "پروندهٔ ایثارگری این کارمند در ماژول امور ایثارگران تأیید شده است"},
	}
}
