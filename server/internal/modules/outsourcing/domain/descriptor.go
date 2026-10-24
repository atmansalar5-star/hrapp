// Package domain دامنهٔ ماژول برون‌سپاری و نیروهای ارکان ثالث.
//
// پیمانکار، دورهٔ حقوق نیروهای ارکان ثالث و پروندهٔ مشاغل سخت و زیان‌آور هر یک یک
// «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول برون‌سپاری.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "outsourcing",
		TitleFa: "برون‌سپاری و نیروهای ارکان ثالث",
		Kinds: []recordkit.Kind{
			{
				Key: "contractors", TitleFa: "پیمانکار", IDPrefix: "CON", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"company|نام شرکت|text|1",
					"contractNo|شمارهٔ قرارداد|text|0",
					"startDate|تاریخ شروع|date|0",
					"workers|تعداد نیرو|number|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال", "expired|منقضی", "terminated|فسخ‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|active", "active|expired,terminated"),
				Actions: recordkit.Actions(
					"activate|فعال‌سازی قرارداد|primary|active|0",
					"expire|ثبت انقضا|ghost|expired|0",
					"terminate|فسخ قرارداد|danger|terminated|1",
				),
			},
			{
				Key: "outsourcePayRuns", TitleFa: "دورهٔ حقوق ارکان ثالث", IDPrefix: "OPAY", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"period|دوره|text|1",
					"contractor|پیمانکار|text|0",
					"gross|جمع ناخالص|money|0",
					"insurance|بیمه|money|0",
					"net|خالص|money|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "submitted|ارسال‌شده", "approved|تأییدشده", "paid|پرداخت‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|submitted", "submitted|approved", "approved|paid"),
				Actions: recordkit.Actions(
					"submit|ارسال برای تأیید|primary|submitted|0",
					"approve|تأیید|primary|approved|0",
					"pay|ثبت پرداخت|primary|paid|0",
				),
			},
			{
				Key: "hardshipFiles", TitleFa: "پروندهٔ مشاغل سخت و زیان‌آور", IDPrefix: "HRD", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"jobTitle|شغل|text|1",
					"hazardFactor|عوامل زیان‌آور|text|0",
					"retireYears|سابقهٔ بازنشستگی|number|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "reviewed|بررسی‌شده", "approved|تأییدشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|reviewed", "reviewed|approved"),
				Actions: recordkit.Actions(
					"review|بررسی پرونده|primary|reviewed|0",
					"approve|تأیید سخت و زیان‌آور|primary|approved|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "contractors", Title: "شرکت خدمات پشتیبانی پارس",
			Fields: map[string]string{"company": "خدمات پشتیبانی پارس", "contractNo": "CON-1404-07", "startDate": "1404/01/01", "workers": "42"},
			NoteFa: "قرارداد فعال"},
		{Kind: "outsourcePayRuns", Title: "حقوق ارکان ثالث مهر ۱۴۰۴",
			Fields: map[string]string{"period": "1404/07", "contractor": "خدمات پشتیبانی پارس", "gross": "980000000", "insurance": "68600000", "net": "911400000"},
			NoteFa: "در انتظار تأیید"},
		{Kind: "hardshipFiles", Title: "پروندهٔ سخت و زیان‌آور — جوشکار", EmployeeID: "EMP-007",
			Fields: map[string]string{"jobTitle": "جوشکار", "hazardFactor": "دود و پرتو جوشکاری", "retireYears": "20"},
			NoteFa: "بررسی‌شده"},
	}
}
