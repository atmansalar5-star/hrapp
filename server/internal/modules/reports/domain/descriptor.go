// Package domain دامنهٔ ماژول گزارش‌ها.
//
// اجرای گزارش و خروجی گزارش هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول گزارش‌ها.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "reports",
		TitleFa: "گزارش‌ها",
		Kinds: []recordkit.Kind{
			{
				Key: "reportRuns", TitleFa: "اجرای گزارش", IDPrefix: "RPT", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"reportType|نوع گزارش|text|1",
					"period|دوره|text|0",
					"scope|دامنه|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "queued|در صف اجرا", "done|آماده", "failed|ناموفق"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|queued", "queued|done,failed", "failed|queued"),
				Actions: recordkit.Actions(
					"run|اجرای گزارش|primary|queued|0",
					"complete|پایان اجرا|primary|done|0",
					"archive|بایگانی|ghost|done|0",
				),
			},
			reportDefsKind(),
			{
				Key: "exports", TitleFa: "خروجی گزارش", IDPrefix: "EXP", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"format|قالب|select|1",
					"reportType|نوع گزارش|text|0",
					"requestedAt|تاریخ درخواست|date|0",
				),
				Statuses:    recordkit.Statuses("requested|درخواست‌شده", "ready|آماده", "failed|ناموفق"),
				Initial:     "requested",
				Transitions: recordkit.Steps("requested|ready,failed"),
				Actions: recordkit.Actions(
					"prepare|آماده‌سازی خروجی|primary|ready|0",
				),
			},
		},
	}
}

// reportDefsKind تعریف گزارش بر پایهٔ اقلام انتخاب‌شده: کاربر منبع داده و اقلام مورد
// نظر را می‌سازد و می‌تواند بارها همان گزارش را اجرا کند.
func reportDefsKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "reportDefs", TitleFa: "تعریف گزارش", IDPrefix: "DEF", EmployeeScoped: false,
		Fields: recordkit.Fields(
			"source|منبع داده|text|1",
			"items|اقلام انتخاب‌شده (با کاما)|text|1",
			"groupBy|گروه‌بندی|text|0",
			"limit|سقف سطر|number|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|active", "active|draft"),
		Actions: recordkit.Actions(
			"activate|فعال‌سازی تعریف|primary|active|0",
			"revise|بازگشت به پیش‌نویس|ghost|draft|0",
		),
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "reportRuns", Title: "گزارش ترکیب نیروی انسانی",
			Fields: map[string]string{"reportType": "ترکیب نیروی انسانی", "period": "1404/07", "scope": "کل سازمان"},
			NoteFa: "آماده"},
		{Kind: "reportRuns", Title: "گزارش حضور و غیاب",
			Fields: map[string]string{"reportType": "حضور و غیاب", "period": "1404/06", "scope": "تولید و عملیات"}},
		{Kind: "exports", Title: "خروجی Excel گزارش بیمه",
			Fields: map[string]string{"format": "Excel", "reportType": "بیمه", "requestedAt": "1404/07/09"},
			NoteFa: "آمادهٔ دریافت"},
		{Kind: "reportDefs", Title: "گزارش اقلام حقوق و ضرائب",
			Fields: map[string]string{"source": "payroll_elements", "items": "code,title,categoryFa,calcFa,rate,basisFa,taxable", "limit": "200"},
			NoteFa: "بر پایهٔ اقلام انتخابی حقوق ساخته شده است"},
	}
}
