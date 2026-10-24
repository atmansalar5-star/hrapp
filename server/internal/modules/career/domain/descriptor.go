// Package domain دامنهٔ ماژول انتصابات، ترفیعات و کارراههٔ شغلی.
//
// قواعد این ماژول به‌صورت «شناسهٔ رکورد» اعلام می‌شود: نوع‌های رکورد، فیلدهای هر نوع،
// وضعیت‌ها، انتقال‌های مجاز و اقدام‌ها. موتور مشترک `recordkit` همین شناسه را اجرا
// می‌کند، پس قاعده اینجا می‌ماند و کد تکراری نوشته نمی‌شود.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// وضعیت‌های مشترک گردش‌کار ابلاغ سازمانی.
const (
	StatusDraft     = "draft"
	StatusApproved  = "approved"
	StatusIssued    = "issued"
	StatusCancelled = "cancelled"
)

// Descriptor شناسهٔ ماژول کارراههٔ شغلی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "career",
		TitleFa: "انتصابات، ترفیعات و کارراههٔ شغلی",
		Kinds: []recordkit.Kind{
			{
				Key: "appointments", TitleFa: "انتصاب", IDPrefix: "APPT", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"position|پست سازمانی|text|1",
					"unit|واحد سازمانی|text|1",
					"effectiveDate|تاریخ اجرا|date|1",
					"decreeNo|شمارهٔ ابلاغ|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "approved|تأییدشدهٔ مدیریت", "issued|ابلاغ‌شده", "cancelled|لغوشده",
				),
				Initial:     StatusDraft,
				Transitions: recordkit.Steps("draft|approved,cancelled", "approved|issued,cancelled"),
				Actions: recordkit.Actions(
					"approve|تأیید مدیریت|primary|approved|0",
					"issue|ابلاغ|primary|issued|0",
					"cancel|لغو|danger|cancelled|1",
				),
			},
			{
				Key: "promotions", TitleFa: "ترفیع", IDPrefix: "PROMO", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"fromPosition|پست فعلی|text|1",
					"toPosition|پست جدید|text|1",
					"effectiveDate|تاریخ اجرا|date|1",
					"evaluationScore|امتیاز ارزیابی|number|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "approved|تأییدشدهٔ کمیته", "issued|ابلاغ‌شده", "cancelled|لغوشده",
				),
				Initial:     StatusDraft,
				Transitions: recordkit.Steps("draft|approved,cancelled", "approved|issued,cancelled"),
				Actions: recordkit.Actions(
					"approve|تأیید کمیته|primary|approved|0",
					"issue|ابلاغ|primary|issued|0",
					"cancel|لغو|danger|cancelled|1",
				),
			},
			{
				Key: "careerPath", TitleFa: "کارراههٔ شغلی", IDPrefix: "CAREER", EmployeeScoped: true,
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

// Seed رکوردهای نمونهٔ سازمان برای نمایش اولیهٔ صفحه‌ها.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{
			Kind: "appointments", Title: "انتصاب مدیر تولید", EmployeeID: "EMP-009",
			Fields: map[string]string{
				"position": "مدیر تولید", "unit": "تولید و عملیات",
				"effectiveDate": "1404/07/01", "decreeNo": "ABL-1404-118",
			},
			NoteFa: "ابلاغ سازمانی در پی بازنشستگی مدیر پیشین",
		},
		{
			Kind: "appointments", Title: "انتصاب سرپرست فروش", EmployeeID: "EMP-005",
			Fields: map[string]string{
				"position": "سرپرست فروش", "unit": "فروش و بازاریابی",
				"effectiveDate": "1404/08/15",
			},
			NoteFa: "در انتظار تأیید مدیریت",
		},
		{
			Kind: "promotions", Title: "ترفیع به کارشناس ارشد مالی", EmployeeID: "EMP-004",
			Fields: map[string]string{
				"fromPosition": "کارشناس مالی", "toPosition": "کارشناس ارشد مالی",
				"effectiveDate": "1404/07/01", "evaluationScore": "92",
			},
			NoteFa: "بر پایهٔ ارزیابی عملکرد دورهٔ اول سال",
		},
		{
			Kind: "promotions", Title: "ترفیع گروه شغلی تولید", EmployeeID: "EMP-007",
			Fields: map[string]string{
				"fromPosition": "سرپرست تولید", "toPosition": "کارشناس ارشد تولید",
				"effectiveDate": "1404/09/01", "evaluationScore": "88",
			},
		},
		{
			Kind: "careerPath", Title: "مسیر رشد به مدیریت منابع انسانی", EmployeeID: "EMP-008",
			Fields: map[string]string{
				"currentPosition": "کارشناس منابع انسانی", "targetPosition": "مدیر منابع انسانی",
				"skillGap":   "دورهٔ مدیریت منابع انسانی راهبردی — ۶۰ ساعت",
				"reviewDate": "1405/01/15",
			},
			NoteFa: "نیازمند تکمیل دوره‌های آموزشی و دو سال سابقهٔ سرپرستی",
		},
	}
}
