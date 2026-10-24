// Package domain دامنهٔ ماژول آمار و پایش اطلاعات پایه.
//
// تأیید اطلاعات پایه و برش آماری هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول آمار و پایش اطلاعات پایه.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "statistics",
		TitleFa: "آمار و پایش اطلاعات پایه",
		Kinds: []recordkit.Kind{
			{
				Key: "dataApprovals", TitleFa: "تأیید اطلاعات پایه", IDPrefix: "APR", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"baseData|نوع اطلاعات پایه|text|1",
					"period|دوره|text|0",
					"unit|واحد|text|0",
				),
				Statuses:    recordkit.Statuses("pending|در انتظار تأیید", "approved|تأییدشده", "rejected|برگشتی"),
				Initial:     "pending",
				Transitions: recordkit.Steps("pending|approved,rejected", "rejected|pending"),
				Actions: recordkit.Actions(
					"approve|تأیید اطلاعات|primary|approved|0",
					"reject|برگشت برای اصلاح|danger|rejected|1",
				),
			},
			{
				Key: "statSnapshots", TitleFa: "برش آماری", IDPrefix: "SNAP", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"period|دوره|text|1",
					"metric|شاخص|text|0",
					"value|مقدار|number|0",
					"unit|واحد|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده", "published|منتشرشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved", "approved|published"),
				Actions: recordkit.Actions(
					"approve|تأیید برش|primary|approved|0",
					"publish|انتشار|primary|published|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "dataApprovals", Title: "تأیید اطلاعات پایهٔ حقوق مهر",
			Fields: map[string]string{"baseData": "اطلاعات پایهٔ حقوق", "period": "1404/07", "unit": "کل سازمان"},
			NoteFa: "در انتظار تأیید واحد آمار"},
		{Kind: "statSnapshots", Title: "شاخص نرخ ترک خدمت — نیمهٔ اول ۱۴۰۴",
			Fields: map[string]string{"period": "1404/06", "metric": "نرخ ترک خدمت", "value": "4.2", "unit": "درصد"},
			NoteFa: "تأییدشده"},
		{Kind: "statSnapshots", Title: "شاخص میانگین سابقهٔ خدمت",
			Fields: map[string]string{"period": "1404/06", "metric": "میانگین سابقه", "value": "9.6", "unit": "سال"}},
	}
}
