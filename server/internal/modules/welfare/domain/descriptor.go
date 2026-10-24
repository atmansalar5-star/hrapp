// Package domain دامنهٔ ماژول رفاه، سرویس مدارس و اسکان.
//
// درخواست رفاهی و تسهیلات رفاهی هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول رفاه.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "welfare",
		TitleFa: "رفاه، سرویس مدارس و اسکان",
		Kinds: []recordkit.Kind{
			{
				Key: "welfareRequests", TitleFa: "درخواست رفاهی", IDPrefix: "WLF", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"facility|تسهیلات|text|1",
					"requestDate|تاریخ درخواست|date|0",
					"amount|مبلغ|money|0",
					"usage|نحوهٔ بهره‌مندی|text|0",
				),
				Statuses:    recordkit.Statuses("requested|درخواست‌شده", "approved|تأییدشده", "rejected|ردشده", "delivered|تحویل‌شده"),
				Initial:     "requested",
				Transitions: recordkit.Steps("requested|approved,rejected", "approved|delivered"),
				Actions: recordkit.Actions(
					"approve|تأیید درخواست|primary|approved|0",
					"reject|رد درخواست|danger|rejected|1",
					"deliver|ثبت تحویل|primary|delivered|0",
				),
			},
			{
				Key: "facilities", TitleFa: "تسهیلات رفاهی", IDPrefix: "FAC", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"name|نام تسهیلات|text|1",
					"capacity|ظرفیت|number|0",
					"location|مکان|text|0",
				),
				Statuses:    recordkit.Statuses("active|فعال", "maintenance|در تعمیر", "inactive|غیرفعال"),
				Initial:     "active",
				Transitions: recordkit.Steps("active|maintenance,inactive", "maintenance|active,inactive", "inactive|active"),
				Actions: recordkit.Actions(
					"maintain|ثبت تعمیر|ghost|maintenance|0",
					"activate|فعال‌سازی|primary|active|0",
					"deactivate|غیرفعال‌سازی|ghost|inactive|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "welfareRequests", Title: "درخواست سرویس مدرسه — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"facility": "سرویس مدرسه", "requestDate": "1404/07/01", "usage": "یک فرزند — مقطع ابتدایی"},
			NoteFa: "در انتظار تأیید واحد رفاه"},
		{Kind: "welfareRequests", Title: "درخواست اسکان — کارمند جدید", EmployeeID: "EMP-009",
			Fields: map[string]string{"facility": "اسکان سازمانی", "requestDate": "1404/06/20", "amount": "0", "usage": "واحد یک‌خوابه"}},
		{Kind: "facilities", Title: "مجتمع رفاهی ساحلی",
			Fields: map[string]string{"name": "مجتمع رفاهی ساحلی", "capacity": "120", "location": "شمال کشور"},
			NoteFa: "فعال"},
	}
}
