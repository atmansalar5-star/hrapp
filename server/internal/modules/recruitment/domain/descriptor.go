// Package domain دامنهٔ ماژول تأمین و جذب نیرو.
//
// درخواست نیرو، آگهی شغلی و متقاضی هر یک یک «نوع رکورد» با گردش‌کار خودش است. متقاضی
// هنوز کارمند نیست، پس کارمندمحور نیست و نامش در عنوان رکورد می‌آید.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول تأمین و جذب نیرو.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "recruitment",
		TitleFa: "تأمین و جذب نیرو",
		Kinds: []recordkit.Kind{
			{
				Key: "hiringRequests", TitleFa: "درخواست نیرو", IDPrefix: "REQ", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"position|عنوان پست|text|1",
					"unit|واحد|text|0",
					"headcount|تعداد نفر|number|0",
					"reason|علت نیاز|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "approved|تأییدشده", "published|آگهی‌شده", "closed|بسته‌شده",
				),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved", "approved|published", "published|closed"),
				Actions: recordkit.Actions(
					"approve|تأیید درخواست|primary|approved|0",
					"publish|انتشار آگهی|primary|published|0",
					"close|بستن درخواست|ghost|closed|1",
				),
			},
			{
				// پیشوند شناسه باید در کل سامانه یکتا باشد: جدول مشترک رکوردها یک کلید
				// اصلی دارد، پس دو ماژول با پیشوند یکسان روی هم می‌نویسند. «JOB» پیش از
				// این به «گروه شغلی» طبقه‌بندی مشاغل تعلق داشت.
				Key: "jobPostings", TitleFa: "آگهی شغلی", IDPrefix: "JBP", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"title|عنوان آگهی|text|1",
					"publishDate|تاریخ انتشار|date|0",
					"deadline|مهلت|date|0",
					"channel|کانال انتشار|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "published|منتشرشده", "closed|بسته‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|published", "published|closed"),
				Actions: recordkit.Actions(
					"publish|انتشار|primary|published|0",
					"close|بستن آگهی|ghost|closed|0",
				),
			},
			{
				Key: "candidates", TitleFa: "متقاضی", IDPrefix: "CAND", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"jobTitle|پست هدف|text|1",
					"stage|مرحله|select|0",
					"score|امتیاز|number|0",
					"appliedDate|تاریخ درخواست|date|0",
				),
				Statuses: recordkit.Statuses(
					"applied|ثبت درخواست", "screening|غربالگری", "interview|مصاحبه", "offer|پیشنهاد استخدام", "hired|استخدام‌شده", "rejected|ردشده",
				),
				Initial: "applied",
				Transitions: recordkit.Steps(
					"applied|screening,rejected", "screening|interview,rejected",
					"interview|offer,rejected", "offer|hired,rejected",
				),
				Actions: recordkit.Actions(
					"screen|غربالگری|primary|screening|0",
					"interview|دعوت به مصاحبه|primary|interview|0",
					"offer|ارائهٔ پیشنهاد|primary|offer|0",
					"hire|استخدام|primary|hired|0",
					"reject|رد متقاضی|danger|rejected|1",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "hiringRequests", Title: "نیاز به کارشناس منابع انسانی",
			Fields: map[string]string{"position": "کارشناس منابع انسانی", "unit": "منابع انسانی", "headcount": "1", "reason": "بلا تصدی بودن پست POS-120"},
			NoteFa: "تأییدشده و آمادهٔ انتشار آگهی"},
		{Kind: "jobPostings", Title: "آگهی کارشناس منابع انسانی",
			Fields: map[string]string{"title": "کارشناس منابع انسانی", "publishDate": "1404/07/10", "deadline": "1404/07/30", "channel": "سایت سازمان و کاریابی"}},
		{Kind: "candidates", Title: "متقاضی — زهرا کریمی",
			Fields: map[string]string{"jobTitle": "کارشناس منابع انسانی", "stage": "مصاحبه", "score": "81", "appliedDate": "1404/07/12"}},
		{Kind: "candidates", Title: "متقاضی — رضا موسوی",
			Fields: map[string]string{"jobTitle": "کارشناس منابع انسانی", "stage": "غربالگری", "score": "74", "appliedDate": "1404/07/14"}},
	}
}
