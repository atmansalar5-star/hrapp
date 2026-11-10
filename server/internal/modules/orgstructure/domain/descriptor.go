// Package domain دامنهٔ ماژول ساختار سازمانی و تشکیلات.
//
// شناسنامهٔ پست سازمانی یک «نوع رکورد» است؛ نمودار درختی واحدها از دادهٔ ساختاری سازمان
// (departments.json) ساخته می‌شود و اینجا پست‌ها و **انتصاب افراد به آن پست‌ها** به‌عنوان
// مرجع انتصاب/ترفیع مدیریت می‌شوند.
//
// نوع رکورد دوم (`assignments`) عمداً جدا از خودِ پست است: «پست» یک شناسنامهٔ ثابت است و
// «انتصاب» یک رابطهٔ زمان‌دار بین یک کارمند و یک پست. جدا بودنشان همان چیزی است که
// «سابقهٔ انتصابات» و «تاریخچهٔ تغییرات سمت» را معنا می‌دهد: با پایان تصدی، انتصاب پاک
// نمی‌شود، به وضعیت «پایان‌یافته» می‌رود و در تاریخچه می‌ماند.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// شناسهٔ نوع‌های رکورد این ماژول.
const (
	KindPositions   = "positions"
	KindAssignments = "assignments"
)

// وضعیت‌های انتصاب.
const (
	AssignmentDraft   = "draft"
	AssignmentActive  = "active"
	AssignmentEnded   = "ended"
	AssignmentRevoked = "revoked"
)

// کلیدهای نتیجهٔ بررسی شرط‌های احراز.
const (
	QualificationMet     = "met"
	QualificationPartial = "partial"
	QualificationNotMet  = "not_met"
)

// Qualification یک نتیجهٔ ممکن برای «احراز شرایط» پست.
type Qualification struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// QualificationCatalog فهرست نتیجهٔ بررسی شرط‌های احراز.
//
// چرا در دامنه: هم فرم ثبت انتصاب و هم گزارش «چه کسی احراز نشده است» از همین یک فهرست
// می‌خوانند؛ پس برچسب فارسی در رابط ساخته نمی‌شود و دو نسخه نمی‌شود.
var QualificationCatalog = []Qualification{
	{Key: QualificationMet, TitleFa: "احرازشده"},
	{Key: QualificationPartial, TitleFa: "احراز جزئی (نیازمند تکمیل)"},
	{Key: QualificationNotMet, TitleFa: "احرازنشده"},
}

// Qualifications فهرست نتیجه‌های احراز (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func Qualifications() []Qualification {
	out := make([]Qualification, len(QualificationCatalog))
	copy(out, QualificationCatalog)
	return out
}

// QualificationTitleFa عنوان فارسی یک نتیجهٔ احراز؛ کلید ناشناخته همان‌طور برمی‌گردد.
func QualificationTitleFa(key string) string {
	value := strings.TrimSpace(key)
	for _, q := range QualificationCatalog {
		if q.Key == value {
			return q.TitleFa
		}
	}
	return value
}

// Descriptor شناسهٔ ماژول ساختار سازمانی.
func descriptorBase() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "orgstructure",
		TitleFa: "ساختار سازمانی و تشکیلات",
		Kinds: []recordkit.Kind{
			{
				Key: KindPositions, TitleFa: "پست سازمانی", IDPrefix: "POS", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"code|کد پست|text|1",
					"unit|واحد سازمانی|text|1",
					"jobFamily|خانوادهٔ شغلی|text|0",
					"grade|گروه شغلی|text|0",
					"requirements|شرط‌های احراز|text|0",
					"headcount|تعداد نفرات مصوب|number|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "active|فعال", "vacant|بلا تصدی", "archived|بایگانی‌شده",
				),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|active,archived", "active|vacant,archived", "vacant|active,archived"),
				Actions: recordkit.Actions(
					"approve|تأیید و فعال‌سازی پست|primary|active|0",
					"vacate|ثبت بلا تصدی|ghost|vacant|0",
					"archive|بایگانی پست|ghost|archived|1",
				),
			},
			{
				Key: KindAssignments, TitleFa: "انتصاب پست سازمانی", IDPrefix: "ASG", EmployeeScoped: true,
				Fields: recordkit.Fields(
					// نوع «select» عمدی است: فهرست کد پست‌ها از /options سرور می‌آید، پس
					// کاربر کد پست را دستی تایپ نمی‌کند و انتصاب به پست ناموجود بسته نمی‌شود.
					"position|کد پست سازمانی|select|1",
					"positionTitle|عنوان پست|text|0",
					"unit|واحد سازمانی|text|1",
					"startDate|تاریخ انتصاب|date|1",
					"endDate|تاریخ پایان تصدی|date|0",
					"decreeNo|شمارهٔ ابلاغ|text|0",
					"qualification|احراز شرایط|select|1",
					"evidence|شواهد احراز|text|0",
					"commitment|تعهد خدمت (ماه)|number|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس انتصاب", "active|در حال تصدی",
					"ended|پایان‌یافته", "revoked|لغوشده",
				),
				Initial: AssignmentDraft,
				Transitions: recordkit.Steps(
					"draft|active,revoked",
					"active|ended,revoked",
				),
				Actions: recordkit.Actions(
					"appoint|انتصاب و آغاز تصدی|primary|active|0",
					"end|ثبت پایان تصدی|ghost|ended|1",
					"revoke|لغو انتصاب|danger|revoked|1",
				),
				// تاریخ مؤثر سابقهٔ انتصاب، تاریخ انتصاب است (نه تاریخ ثبت رکورد).
				HistoryDate:      "startDate",
				EditableStatuses: []string{AssignmentDraft, AssignmentActive},
				ActRule:          appointmentRule,
			},
		},
	}
}

// appointmentRule قاعدهٔ اقدام‌های انتصاب.
//
// چرا اینجا و نه در موتور: «انتصاب بدون احراز شرایط» یک قاعدهٔ کسب‌وکاری این سازمان است،
// پس در دامنهٔ همین ماژول اعلام می‌شود و موتور رکورد فقط صدایش می‌زند. شرط‌ها:
//   - آغاز تصدی (appoint) تنها با نتیجهٔ «احرازشده» و شواهد ثبت‌شده ممکن است؛
//   - تاریخ پایان تصدی پیش از تاریخ انتصاب معنا ندارد.
func appointmentRule(rec recordkit.Record, action string) error {
	if action != "appoint" {
		return nil
	}
	if strings.TrimSpace(rec.Fields["qualification"]) != QualificationMet {
		return &recordkit.RuleError{Message: "انتصاب به تصدی پست تنها با نتیجهٔ «احرازشده» ممکن است؛ " +
			"نتیجهٔ بررسی شرط‌های احراز را در همین انتصاب ثبت و اصلاح کنید."}
	}
	if strings.TrimSpace(rec.Fields["evidence"]) == "" {
		return &recordkit.RuleError{Message: "برای آغاز تصدی، ثبت «شواهد احراز» الزامی است " +
			"(مدرک تحصیلی، سابقهٔ کار، گواهی مهارت یا صورت‌جلسهٔ بررسی)."}
	}
	if end := strings.TrimSpace(rec.Fields["endDate"]); end != "" && end < strings.TrimSpace(rec.Fields["startDate"]) {
		return &recordkit.RuleError{Message: "تاریخ پایان تصدی نمی‌تواند پیش از تاریخ انتصاب باشد."}
	}
	return nil
}

// Seed رکوردهای نمونهٔ سازمان.
func seedBase() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: KindPositions, Title: "کارشناس ارشد مالی", SeedStatus: "active",
			Fields: map[string]string{"code": "POS-201", "unit": "امور مالی", "jobFamily": "مالی و حسابداری", "grade": "کارشناسی ارشد", "requirements": "کارشناسی ارشد حسابداری، ۵ سال سابقه", "headcount": "2"},
			NoteFa: "پست مرجع برای ترفیع کارشناسان مالی"},
		{Kind: KindPositions, Title: "مدیر تولید", SeedStatus: "active",
			Fields: map[string]string{"code": "POS-310", "unit": "تولید و عملیات", "jobFamily": "مدیریت تولید", "grade": "مدیریت", "requirements": "کارشناسی مهندسی صنایع، ۱۰ سال سابقهٔ سرپرستی", "headcount": "1"}},
		{Kind: KindPositions, Title: "کارشناس منابع انسانی", SeedStatus: "vacant",
			Fields: map[string]string{"code": "POS-120", "unit": "منابع انسانی", "jobFamily": "منابع انسانی", "grade": "کارشناسی", "requirements": "کارشناسی مدیریت منابع انسانی، ۳ سال سابقه", "headcount": "1"},
			NoteFa: "بلا تصدی؛ درخواست نیرو ثبت شده"},
		{Kind: KindPositions, Title: "سرپرست فروش", SeedStatus: "active",
			Fields: map[string]string{"code": "POS-405", "unit": "فروش و بازاریابی", "jobFamily": "فروش", "grade": "سرپرستی", "requirements": "کارشناسی بازاریابی، ۴ سال سابقهٔ فروش", "headcount": "1"}},

		// انتصاب‌های جاری: همراه با نتیجهٔ احراز و شواهد آن، چون بدون این دو آغاز تصدی
		// ممکن نیست و همین‌ها مبنای نمایش «شرایط احراز» در ساختار سازمانی‌اند.
		{Kind: KindAssignments, Title: "انتصاب کارشناس ارشد مالی", EmployeeID: "EMP-004", SeedStatus: AssignmentActive,
			Fields: map[string]string{
				"position": "POS-201", "positionTitle": "کارشناس ارشد مالی", "unit": "امور مالی",
				"startDate": "1403/07/01", "decreeNo": "ABL-1403-224",
				"qualification": QualificationMet,
				"evidence":      "کارشناسی ارشد حسابداری (گواهی ۱۳۹۸)، ۷ سال سابقهٔ مالی، قبولی در آزمون داخلی",
				"commitment":    "60",
			},
			NoteFa: "انتصاب در پی ارتقای گروه شغلی"},
		{Kind: KindAssignments, Title: "انتصاب مدیر تولید", EmployeeID: "EMP-009", SeedStatus: AssignmentActive,
			Fields: map[string]string{
				"position": "POS-310", "positionTitle": "مدیر تولید", "unit": "تولید و عملیات",
				"startDate": "1404/07/01", "decreeNo": "ABL-1404-118",
				"qualification": QualificationMet,
				"evidence":      "کارشناسی مهندسی صنایع، ۱۱ سال سابقهٔ سرپرستی تولید، صورت‌جلسهٔ کمیتهٔ انتصابات ۱۴۰۴/۰۶/۲۵",
			},
			NoteFa: "ابلاغ سازمانی در پی بازنشستگی مدیر پیشین"},
		{Kind: KindAssignments, Title: "انتصاب سرپرست فروش", EmployeeID: "EMP-003", SeedStatus: AssignmentActive,
			Fields: map[string]string{
				"position": "POS-405", "positionTitle": "سرپرست فروش", "unit": "فروش و بازاریابی",
				"startDate":     "1404/08/15",
				"qualification": QualificationMet,
				"evidence":      "کارشناسی بازاریابی، ۵ سال سابقهٔ فروش، گواهی دورهٔ سرپرستی",
			}},
		// انتصاب پیشین همین پست: نمونهٔ «سابقهٔ انتصابات» — پایان‌یافته ولی در تاریخچه می‌ماند.
		{Kind: KindAssignments, Title: "انتصاب پیشین سرپرست فروش", EmployeeID: "EMP-005",
			Fields: map[string]string{
				"position": "POS-405", "positionTitle": "سرپرست فروش", "unit": "فروش و بازاریابی",
				"startDate": "1401/03/01", "endDate": "1404/08/14", "decreeNo": "ABL-1401-047",
				"qualification": QualificationMet,
				"evidence":      "کارشناسی مدیریت بازرگانی، ۹ سال سابقهٔ فروش",
			},
			SeedStatus: AssignmentEnded,
			NoteFa:     "پایان تصدی با جابه‌جایی به پست مدیر فروش"},
		// انتصاب در انتظار: نتیجهٔ احراز جزئی، پس گردشکار اجازهٔ آغاز تصدی نمی‌دهد.
		{Kind: KindAssignments, Title: "انتصاب کارشناس منابع انسانی", EmployeeID: "EMP-008",
			Fields: map[string]string{
				"position": "POS-120", "positionTitle": "کارشناس منابع انسانی", "unit": "منابع انسانی",
				"startDate":     "1405/09/01",
				"qualification": QualificationPartial,
				"evidence":      "کارشناسی مدیریت منابع انسانی؛ دورهٔ تخصصی منابع انسانی راهبردی در حال گذراندن",
			},
			NoteFa: "نیازمند تکمیل دوره و تأیید شواهد پیش از آغاز تصدی"},
	}
}
