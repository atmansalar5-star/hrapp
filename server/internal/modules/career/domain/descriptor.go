// Package domain دامنهٔ ماژول انتصابات، ترفیعات و کارراههٔ شغلی.
//
// قواعد این ماژول به‌صورت «شناسهٔ رکورد» اعلام می‌شود: نوع‌های رکورد، فیلدهای هر نوع،
// وضعیت‌ها، انتقال‌های مجاز و اقدام‌ها. موتور مشترک `recordkit` همین شناسه را اجرا
// می‌کند، پس قاعده اینجا می‌ماند و کد تکراری نوشته نمی‌شود.
//
// گردش‌کار انتصاب و ترفیع («تصویب‌خواهی و اجرا»):
//
//	پیش‌نویس ──ارسال برای تصویب──▶ در انتظار تصویب ──تصویب──▶ تصویب‌شده ──اجرا و ابلاغ──▶ ابلاغ‌شده
//	   └──────────────────────────── لغو ────────────────────────────┘
//
// چرا سه گام و نه دو گام: «تصویب» تصمیم است و «اجرا» اقدام سازمانی؛ میان این دو، شمارهٔ
// ابلاغ و تاریخ اجرا ثبت می‌شود. اگر اجرا بی‌ابلاغ انجام شود، سابقهٔ کارمند و حقوق
// دستمزد بی‌مستند می‌ماند.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// وضعیت‌های مشترک گردش‌کار ابلاغ سازمانی.
const (
	StatusDraft     = "draft"
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusIssued    = "issued"
	StatusCancelled = "cancelled"
)

// انواع رکورد این ماژول.
const (
	KindAppointments = "appointments"
	KindPromotions   = "promotions"
	KindCareerPath   = "careerPath"
)

// Descriptor شناسهٔ ماژول کارراههٔ شغلی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "career",
		TitleFa: "انتصابات، ترفیعات و کارراههٔ شغلی",
		Kinds: []recordkit.Kind{
			{
				Key: KindAppointments, TitleFa: "انتصاب", IDPrefix: "APPT", EmployeeScoped: true,
				Fields: recordkit.WithEmployeeBase(recordkit.Fields(
					// وضعیت فعلی کارمند و پست پیشنهادی؛ فرم تصویب‌خواهی هر دو را نشان
					// می‌دهد تا تصمیم‌گیرنده «از کجا به کجا» را ببیند.
					"currentPosition|پست فعلی|text|0",
					"currentUnit|واحد فعلی|text|0",
					"position|پست پیشنهادی|text|1",
					"unit|واحد سازمانی پیشنهادی|text|1",
					// گروه شغلی از نردبان گروه‌ها (select) انتخاب می‌شود؛ رتبهٔ پله در
					// موتور قواعد مبنای سنجش جهش و سطح تصویب است.
					"currentGrade|گروه شغلی فعلی|select|1",
					"proposedGrade|گروه شغلی پیشنهادی|select|1",
					"serviceMonths|سابقهٔ گروه فعلی (ماه)|number|0",
					"evaluationScore|امتیاز ارزیابی عملکرد|number|0",
					"vacancy|ظرفیت پست پیشنهادی خالی است|select|0",
					"commitment|تعهد خدمت (ماه)|number|0",
					"effectiveDate|تاریخ اجرا|date|1",
					"decreeNo|شمارهٔ ابلاغ|text|0",
					"reason|دلیل و ملاک تصویب|text|0",
					"approvalLevels|سطوح تصویب لازم|text|0",
				)),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "pending|در انتظار تصویب", "approved|تصویب‌شده",
					"issued|ابلاغ‌شده و اجراشده", "cancelled|لغوشده",
				),
				Initial: StatusDraft,
				Transitions: recordkit.Steps(
					"draft|pending,cancelled",
					"pending|approved,cancelled",
					"approved|issued,cancelled",
				),
				Actions: recordkit.Actions(
					"submit|ارسال برای تصویب|primary|pending|0|employee,hr_spec,hr_manager,line_manager",
					"approve|تصویب درخواست|primary|approved|0|hr_manager,line_manager",
					"issue|اجرا و ابلاغ|primary|issued|0|hr_manager",
					"cancel|لغو|danger|cancelled|1|hr_manager,line_manager",
				),
				// تاریخ مؤثر سابقهٔ انتصاب، تاریخ اجراست (نه تاریخ ثبت رکورد).
				HistoryDate:      "effectiveDate",
				EditableStatuses: []string{StatusDraft},
				ActRule:          approvalRule,
			},
			{
				Key: KindPromotions, TitleFa: "ترفیع", IDPrefix: "PROMO", EmployeeScoped: true,
				Fields: recordkit.WithEmployeeBase(recordkit.Fields(
					"fromPosition|پست فعلی|text|1",
					"toPosition|پست پیشنهادی|text|1",
					"unit|واحد سازمانی پیشنهادی|text|0",
					"currentGrade|گروه شغلی فعلی|select|1",
					"proposedGrade|گروه شغلی پیشنهادی|select|1",
					"serviceMonths|سابقهٔ گروه فعلی (ماه)|number|0",
					"evaluationScore|امتیاز ارزیابی عملکرد|number|0",
					"vacancy|ظرفیت پست پیشنهادی خالی است|select|0",
					"commitment|تعهد خدمت (ماه)|number|0",
					"effectiveDate|تاریخ اجرا|date|1",
					"decreeNo|شمارهٔ ابلاغ|text|0",
					"reason|دلیل و ملاک تصویب|text|0",
					"approvalLevels|سطوح تصویب لازم|text|0",
				)),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "pending|در انتظار تصویب", "approved|تصویب‌شدهٔ کمیته",
					"issued|ابلاغ‌شده و اجراشده", "cancelled|لغوشده",
				),
				Initial: StatusDraft,
				Transitions: recordkit.Steps(
					"draft|pending,cancelled",
					"pending|approved,cancelled",
					"approved|issued,cancelled",
				),
				Actions: recordkit.Actions(
					"submit|ارسال برای تصویب|primary|pending|0|employee,hr_spec,hr_manager,line_manager",
					"approve|تصویب کمیته|primary|approved|0|hr_manager,line_manager",
					"issue|اجرا و ابلاغ|primary|issued|0|hr_manager",
					"cancel|لغو|danger|cancelled|1|hr_manager,line_manager",
				),
				HistoryDate:      "effectiveDate",
				EditableStatuses: []string{StatusDraft},
				ActRule:          approvalRule,
			},
			{
				Key: KindCareerPath, TitleFa: "کارراههٔ شغلی", IDPrefix: "CAREER", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"currentPosition|پست فعلی|text|1",
					"targetPosition|پست هدف|text|1",
					"skillGap|شکاف مهارتی|text|0",
					"reviewDate|تاریخ بازنگری|date|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "active|جاری", "achieved|محقق‌شده", "archived|بایگانی‌شده",
				),
				Initial:     StatusDraft,
				Transitions: recordkit.Steps("draft|active,archived", "active|achieved,archived"),
				Actions: recordkit.Actions(
					"activate|فعال‌سازی مسیر|primary|active|0",
					"achieve|ثبت تحقق|primary|achieved|0",
					"archive|بایگانی|ghost|archived|1",
				),
			},
		},
	}
}

// approvalRule قاعدهٔ اقدام‌های انتصاب و ترفیع.
//
// چرا اینجا و نه در موتور: «تصویب‌خواهی ناقص» و «اجرای بی‌ابلاغ» قاعدهٔ کسب‌وکاری سازمان
// است، پس در دامنهٔ همین ماژول اعلام می‌شود و موتور رکورد فقط صدایش می‌زند. شرط‌ها:
//   - ارسال برای تصویب: تاریخ اجرا، گروه فعلی و پیشنهادی و «دلیل و ملاک» الزامی است؛
//     پستی که ظرفیت خالی ندارد تصویب‌خواهی نمی‌شود و ترفیع بدون امتیاز ارزیابی ارسال نمی‌شود.
//   - اجرا و ابلاغ: شمارهٔ ابلاغ و تاریخ اجرا الزامی است؛ ابلاغ بی‌شماره سند سازمانی نیست.
//
// آستانه‌های عددی (سابقه، امتیاز، جهش گروه، سطح تصویب) در موتور قواعد است و مسیر
// «تصویب‌خواهی» همان‌ها را با پارامترهای شرکت می‌سنجد؛ اینجا فقط کامل‌بودن سند بررسی می‌شود.
func approvalRule(rec recordkit.Record, action string) error {
	effectiveDate := strings.TrimSpace(rec.Fields["effectiveDate"])
	switch action {
	case "submit":
		if effectiveDate == "" {
			return &recordkit.RuleError{Message: "تاریخ اجرای ابلاغ را ثبت کنید؛ تصویب‌خواهی بدون تاریخ اجرا ارسال نمی‌شود."}
		}
		if strings.TrimSpace(rec.Fields["currentGrade"]) == "" || strings.TrimSpace(rec.Fields["proposedGrade"]) == "" {
			return &recordkit.RuleError{Message: "گروه شغلی فعلی و پیشنهادی را انتخاب کنید تا سطح تصویب و سنجش جهش ممکن شود."}
		}
		if strings.TrimSpace(rec.Fields["reason"]) == "" {
			return &recordkit.RuleError{Message: "«دلیل و ملاک تصویب» را بنویسید؛ تصویب‌خواهی بدون ملاک قابل تصمیم‌گیری نیست."}
		}
		if strings.TrimSpace(rec.Fields["vacancy"]) == "no" {
			return &recordkit.RuleError{Message: "پست پیشنهادی ظرفیت خالی ندارد؛ تصدی هم‌زمان دو نفر در یک پست مجاز نیست."}
		}
		if rec.Kind == KindPromotions && parseNumber(rec.Fields["evaluationScore"]) <= 0 {
			return &recordkit.RuleError{Message: "برای ترفیع، امتیاز ارزیابی عملکرد دوره را ثبت کنید."}
		}
	case "issue":
		if effectiveDate == "" {
			return &recordkit.RuleError{Message: "تاریخ اجرای ابلاغ را ثبت کنید؛ ابلاغ بی‌تاریخ اجرا قابل اجرا نیست."}
		}
		if strings.TrimSpace(rec.Fields["decreeNo"]) == "" {
			return &recordkit.RuleError{Message: "برای اجرا، شمارهٔ ابلاغ را ثبت کنید (مثلاً ABL-1405-101)."}
		}
	}
	return nil
}

// parseNumber عدد یک فیلد متنی را می‌خواند (صفر اگر نامعتبر باشد).
func parseNumber(value string) float64 {
	num, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return num
}

// ExecutionStep یک گام از «اجرا و ابلاغ» سازمانی.
//
// چرا در دامنه: ترتیب گام‌های اجرا (ابلاغ ← سابقه ← تشکیلات ← حقوق ← اطلاع و بایگانی)
// رویهٔ سازمان است، نه چیدمان صفحه؛ فرم، راهنمای کاربر و مستند استقرار باید یک ترتیب
// ببینند و اگر گامی اضافه شد، فقط همین‌جا اعلام شود.
type ExecutionStep struct {
	Order    int    `json:"order"`
	Key      string `json:"key"`
	TitleFa  string `json:"titleFa"`
	DetailFa string `json:"detailFa"`
}

// ExecutionSteps گام‌های اجرای ابلاغ انتصاب یا ترفیع.
func ExecutionSteps(kind string) []ExecutionStep {
	target := "پست پیشنهادی"
	if strings.TrimSpace(kind) == "promotion" {
		target = "پست و گروه شغلی جدید"
	}
	return []ExecutionStep{
		{Order: 1, Key: "decree", TitleFa: "صدور ابلاغ",
			DetailFa: "شمارهٔ ابلاغ، تاریخ اجرا و مقام تصویب‌کننده در رکورد ثبت می‌شود."},
		{Order: 2, Key: "history", TitleFa: "ثبت سابقهٔ کارمند",
			DetailFa: target + " از تاریخ اجرا در سابقهٔ سازمانی کارمند ثبت می‌شود."},
		{Order: 3, Key: "orgchart", TitleFa: "به‌روزرسانی تشکیلات",
			DetailFa: "پست پیشنهادی متصدی می‌گیرد و وضعیت پست پیشین بررسی و در صورت خالی‌شدن بلا تصدی می‌شود."},
		{Order: 4, Key: "payroll", TitleFa: "اعلام به حقوق و دستمزد",
			DetailFa: "گروه شغلی و مبنای پرداخت از تاریخ اجرا در دورهٔ حقوق همان ماه اعمال می‌شود."},
		{Order: 5, Key: "notify", TitleFa: "اطلاع به کارمند و بایگانی",
			DetailFa: "ابلاغ به کارمند اطلاع داده و نسخهٔ امضاشده در پرونده بایگانی می‌شود."},
	}
}

// Seed رکوردهای نمونهٔ سازمان برای نمایش اولیهٔ صفحه‌ها.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{
			Kind: KindAppointments, Title: "انتصاب مدیر تولید", EmployeeID: "EMP-009",
			Fields: map[string]string{
				"currentPosition": "کارشناس ارشد فرآیند", "currentUnit": "تولید و عملیات",
				"position": "مدیر تولید", "unit": "تولید و عملیات",
				"currentGrade": "senior_expert", "proposedGrade": "manager",
				"serviceMonths": "42", "evaluationScore": "91", "vacancy": "yes", "commitment": "36",
				"effectiveDate": "1404/07/01", "decreeNo": "ABL-1404-118",
				"reason":         "بازنشستگی مدیر پیشین و احراز شرایط مدیریت تولید در ارزیابی سالانه",
				"approvalLevels": "۴ سطح: مدیر واحد، مدیر منابع انسانی، کمیتهٔ انتصابات، مدیرعامل",
			},
			SeedStatus: StatusIssued,
			NoteFa:     "ابلاغ سازمانی در پی بازنشستگی مدیر پیشین",
		},
		{
			Kind: KindAppointments, Title: "انتصاب سرپرست فروش", EmployeeID: "EMP-005",
			Fields: map[string]string{
				"currentPosition": "کارشناس فروش", "currentUnit": "فروش و بازاریابی",
				"position": "سرپرست فروش", "unit": "فروش و بازاریابی",
				"currentGrade": "expert", "proposedGrade": "supervisor",
				"serviceMonths": "28", "evaluationScore": "83", "vacancy": "yes", "commitment": "24",
				"effectiveDate":  "1405/09/01",
				"reason":         "خالی‌بودن پست سرپرست فروش و پیشنهاد مدیر واحد با تأیید ارزیابی عملکرد",
				"approvalLevels": "۳ سطح: مدیر واحد، مدیر منابع انسانی، کمیتهٔ انتصابات",
			},
			SeedStatus: StatusPending,
			NoteFa:     "در انتظار تصویب کمیته",
		},
		{
			Kind: KindPromotions, Title: "ترفیع به کارشناس ارشد مالی", EmployeeID: "EMP-004",
			Fields: map[string]string{
				"fromPosition": "کارشناس مالی", "toPosition": "کارشناس ارشد مالی",
				"unit":         "امور مالی",
				"currentGrade": "expert", "proposedGrade": "senior_expert",
				"serviceMonths": "30", "evaluationScore": "92", "vacancy": "yes", "commitment": "0",
				"effectiveDate": "1404/07/01", "decreeNo": "ABL-1404-201",
				"reason":         "ارزیابی عملکرد دورهٔ اول سال و سه سال سابقهٔ کارشناسی مالی",
				"approvalLevels": "سه سطح: مدیر واحد، مدیر منابع انسانی، کمیتهٔ انتصابات و ترفیعات",
			},
			SeedStatus: StatusIssued,
			NoteFa:     "بر پایهٔ ارزیابی عملکرد دورهٔ اول سال",
		},
		{
			Kind: KindPromotions, Title: "ترفیع گروه شغلی تولید", EmployeeID: "EMP-007",
			Fields: map[string]string{
				"fromPosition": "سرپرست تولید", "toPosition": "کارشناس ارشد تولید",
				"unit":         "تولید و عملیات",
				"currentGrade": "supervisor", "proposedGrade": "senior_expert",
				"serviceMonths": "26", "evaluationScore": "88", "vacancy": "yes", "commitment": "0",
				"effectiveDate":  "1405/09/01",
				"reason":         "کسب امتیاز ارزیابی بالای حد مصوب و تأیید کمیتهٔ ترفیعات",
				"approvalLevels": "سه سطح: مدیر واحد، مدیر منابع انسانی، کمیتهٔ انتصابات و ترفیعات",
			},
			SeedStatus: StatusApproved,
			NoteFa:     "تصویب‌شده؛ در انتظار ثبت شمارهٔ ابلاغ",
		},
		{
			Kind: KindCareerPath, Title: "مسیر رشد به مدیریت منابع انسانی", EmployeeID: "EMP-008",
			Fields: map[string]string{
				"currentPosition": "کارشناس منابع انسانی", "targetPosition": "مدیر منابع انسانی",
				"skillGap":   "دورهٔ مدیریت منابع انسانی راهبردی — ۶۰ ساعت",
				"reviewDate": "1405/01/15",
			},
			NoteFa: "نیازمند تکمیل دوره‌های آموزشی و دو سال سابقهٔ سرپرستی",
		},
	}
}
