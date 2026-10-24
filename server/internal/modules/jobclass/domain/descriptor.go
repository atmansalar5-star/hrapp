// Package domain دامنهٔ ماژول طبقه‌بندی مشاغل.
//
// طبقه‌بندی شغلی، ارزشیابی مشاغل و شرح شغل هر یک یک «نوع رکورد» است؛ قواعد اینجا اعلام
// می‌شود و موتور مشترک `recordkit` آن را اجرا می‌کند.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول طبقه‌بندی مشاغل.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "jobclass",
		TitleFa: "طبقه‌بندی مشاغل",
		Kinds: []recordkit.Kind{
			{
				Key: "jobCategories", TitleFa: "طبقه‌بندی شغلی", IDPrefix: "JOB", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"code|کد طبقه|text|1",
					"groupName|گروه شغلی|text|1",
					"grade|رتبه|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده", "active|جاری"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved", "approved|active"),
				Actions: recordkit.Actions(
					"approve|تأیید طبقه|primary|approved|0",
					"activate|جاری‌سازی|primary|active|0",
				),
			},
			{
				Key: "jobEvaluations", TitleFa: "ارزشیابی شغل", IDPrefix: "EVA", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"jobTitle|عنوان شغل|text|1",
					"points|امتیاز ارزشیابی|number|0",
					"factor|عوامل ارزشیابی|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "scored|امتیازدهی‌شده", "approved|تأییدشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|scored", "scored|approved"),
				Actions: recordkit.Actions(
					"score|ثبت امتیاز|primary|scored|0",
					"approve|تأیید ارزشیابی|primary|approved|0",
				),
			},
			{
				Key: "jobDescriptions", TitleFa: "شرح شغل", IDPrefix: "JDD", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"jobTitle|عنوان شغل|text|1",
					"duties|وظایف اصلی|text|0",
					"requirements|شرط‌های احراز|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده", "revised|بازنگری‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved", "approved|revised", "revised|approved"),
				Actions: recordkit.Actions(
					"approve|تأیید شرح شغل|primary|approved|0",
					"revise|ثبت بازنگری|ghost|revised|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "jobCategories", Title: "گروه شغلی مالی و حسابداری",
			Fields: map[string]string{"code": "FIN", "groupName": "مالی و حسابداری", "grade": "کارشناسی تا ارشد"}},
		{Kind: "jobCategories", Title: "گروه شغلی تولید",
			Fields: map[string]string{"code": "PRD", "groupName": "تولید و عملیات", "grade": "عملیاتی تا مدیریت"}},
		{Kind: "jobEvaluations", Title: "ارزشیابی شغل کارشناس ارشد مالی",
			Fields: map[string]string{"jobTitle": "کارشناس ارشد مالی", "points": "720", "factor": "دانش، پیچیدگی، مسئولیت"},
			NoteFa: "امتیاز پایهٔ ترفیع"},
		{Kind: "jobDescriptions", Title: "شرح شغل سرپرست فروش",
			Fields: map[string]string{"jobTitle": "سرپرست فروش", "duties": "هدایت تیم فروش، هدف‌گذاری و پایش فروش", "requirements": "کارشناسی مدیریت، ۵ سال سابقه"}},
	}
}
