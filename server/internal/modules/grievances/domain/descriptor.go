// Package domain دامنهٔ ماژول شکایات و امور ایثارگران.
//
// شکایت کارمند و پروندهٔ ایثارگری هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول شکایات و امور ایثارگران.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "grievances",
		TitleFa: "شکایات و امور ایثارگران",
		Kinds: []recordkit.Kind{
			{
				Key: "complaints", TitleFa: "شکایت کارمند", IDPrefix: "CMP", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"subject|موضوع شکایت|text|1",
					"filedDate|تاریخ ثبت|date|0",
					"channel|کانال ثبت|text|0",
					"resolution|نتیجهٔ رسیدگی|text|0",
				),
				Statuses: recordkit.Statuses(
					"received|دریافت‌شده", "investigating|در حال رسیدگی", "resolved|حل‌شده", "rejected|ردشده", "closed|بسته‌شده",
				),
				Initial: "received",
				Transitions: recordkit.Steps(
					"received|investigating,closed", "investigating|resolved,rejected,closed",
					"resolved|closed", "rejected|closed",
				),
				Actions: recordkit.Actions(
					"investigate|شروع رسیدگی|primary|investigating|0",
					"resolve|ثبت حل شکایت|primary|resolved|0",
					"reject|رد شکایت|danger|rejected|1",
					"close|بستن پرونده|ghost|closed|1",
				),
			},
			{
				Key: "veteranRecords", TitleFa: "پروندهٔ ایثارگری", IDPrefix: "VET", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"veteranType|نوع ایثارگری|text|1",
					"percent|درصد جانبازی|number|0",
					"benefits|مزایا|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "verified|بررسی‌شده", "approved|تأییدشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|verified", "verified|approved"),
				Actions: recordkit.Actions(
					"verify|بررسی مدارک|primary|verified|0",
					"approve|تأیید پرونده|primary|approved|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "complaints", Title: "شکایت — نحوهٔ محاسبهٔ اضافه‌کاری", EmployeeID: "EMP-007",
			Fields: map[string]string{"subject": "اختلاف در محاسبهٔ اضافه‌کاری شهریور", "filedDate": "1404/07/03", "channel": "سامانهٔ داخلی"},
			NoteFa: "در حال رسیدگی"},
		{Kind: "complaints", Title: "شکایت — شرایط محیط کار", EmployeeID: "EMP-004",
			Fields: map[string]string{"subject": "نبود تهویهٔ مناسب در کارگاه", "filedDate": "1404/06/18", "resolution": "رفع نقص و نصب سیستم تهویه"}},
		{Kind: "veteranRecords", Title: "پروندهٔ ایثارگری — کارمند جانباز", EmployeeID: "EMP-009",
			Fields: map[string]string{"veteranType": "جانباز", "percent": "25", "benefits": "معافیت مالیاتی و مرخصی تشویقی"},
			NoteFa: "بررسی‌شده؛ در انتظار تأیید"},
	}
}
