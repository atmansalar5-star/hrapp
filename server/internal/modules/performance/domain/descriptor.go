// Package domain دامنهٔ ماژول مدیریت عملکرد.
//
// دورهٔ ارزیابی، فرم ارزیابی و نتیجهٔ ارزیابی هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول مدیریت عملکرد.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "performance",
		TitleFa: "مدیریت عملکرد",
		Kinds: []recordkit.Kind{
			{
				Key: "perfCycles", TitleFa: "دورهٔ ارزیابی", IDPrefix: "CYC", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"title|عنوان دوره|text|1",
					"startDate|تاریخ شروع|date|0",
					"endDate|تاریخ پایان|date|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "open|در حال اجرا", "closed|بسته‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|open", "open|closed"),
				Actions: recordkit.Actions(
					"open|آغاز دوره|primary|open|0",
					"close|بستن دوره|ghost|closed|1",
				),
			},
			{
				Key: "perfForms", TitleFa: "فرم ارزیابی", IDPrefix: "FRM", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"cycle|دوره|text|1",
					"score|امتیاز|number|0",
					"reviewer|ارزیاب|text|0",
				),
				Statuses: recordkit.Statuses(
					"assigned|ارجاع‌شده", "inProgress|در حال تکمیل", "submitted|ارسال‌شده", "reviewed|بازبینی‌شده",
				),
				Initial:     "assigned",
				Transitions: recordkit.Steps("assigned|inProgress", "inProgress|submitted", "submitted|reviewed"),
				Actions: recordkit.Actions(
					"start|شروع تکمیل|primary|inProgress|0",
					"submit|ارسال فرم|primary|submitted|0",
					"review|بازبینی|ghost|reviewed|0",
				),
			},
			{
				Key: "perfResults", TitleFa: "نتیجهٔ ارزیابی", IDPrefix: "RES", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"score|امتیاز نهایی|number|0",
					"grade|رتبه|text|0",
					"recommendation|پیشنهاد|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "final|نهایی", "archived|بایگانی‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|final", "final|archived"),
				Actions: recordkit.Actions(
					"finalize|نهایی‌سازی نتیجه|primary|final|0",
					"archive|بایگانی|ghost|archived|1",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "perfCycles", Title: "ارزیابی نیمهٔ اول ۱۴۰۴",
			Fields: map[string]string{"title": "ارزیابی نیمهٔ اول ۱۴۰۴", "startDate": "1404/04/01", "endDate": "1404/06/31"},
			NoteFa: "در حال اجرا"},
		{Kind: "perfForms", Title: "فرم ارزیابی — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"cycle": "1404/04", "score": "88", "reviewer": "مدیر تولید"}},
		{Kind: "perfResults", Title: "نتیجهٔ ارزیابی — مریم نوری", EmployeeID: "EMP-008",
			Fields: map[string]string{"score": "92", "grade": "عالی", "recommendation": "شایستهٔ ترفیع"},
			NoteFa: "نهایی‌شده"},
	}
}
