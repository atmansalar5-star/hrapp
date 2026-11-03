// Package domain دامنهٔ ماژول جابه‌جایی و معرفی کارکنان.
//
// درخواست جابه‌جایی و معرفی/ابلاغ کارمند هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول جابه‌جایی و معرفی کارکنان.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "staffing_moves",
		TitleFa: "جابه‌جایی و معرفی کارکنان",
		Kinds: []recordkit.Kind{
			{
				Key: "transfers", TitleFa: "درخواست جابه‌جایی", IDPrefix: "TR", EmployeeScoped: true,
				Fields: recordkit.WithEmployeeBase(recordkit.Fields(
					"fromUnit|واحد مبدأ|text|1",
					"toUnit|واحد مقصد|text|1",
					"effectiveDate|تاریخ اجرا|date|0",
					"reason|علت|text|0",
				)),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "pending|در انتظار تأیید", "approved|تأییدشده", "rejected|ردشده", "executed|اجراشده",
				),
				Initial: "draft",
				Transitions: recordkit.Steps(
					"draft|pending", "pending|approved,rejected", "approved|executed",
				),
				Actions: recordkit.Actions(
					"submit|ارسال برای تأیید|primary|pending|0",
					"approve|تأیید|primary|approved|0",
					"execute|اجرای جابه‌جایی|primary|executed|0",
					"reject|رد درخواست|danger|rejected|1",
				),
			},
			{
				Key: "introductions", TitleFa: "معرفی و ابلاغ کارمند", IDPrefix: "INTRO", EmployeeScoped: true,
				Fields: recordkit.WithEmployeeBase(recordkit.Fields(
					"toUnit|واحد مقصد|text|1",
					"effectiveDate|تاریخ اجرا|date|0",
					"letterNo|شمارهٔ ابلاغ|text|0",
				)),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "issued|ابلاغ‌شده", "cancelled|لغوشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|issued,cancelled"),
				Actions: recordkit.Actions(
					"issue|صدور ابلاغ|primary|issued|0",
					"cancel|لغو ابلاغ|danger|cancelled|1",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "transfers", Title: "جابه‌جایی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"fromUnit": "فروش و بازاریابی", "toUnit": "منابع انسانی", "effectiveDate": "1404/08/01", "reason": "نیاز واحد منابع انسانی"},
			NoteFa: "در انتظار تأیید مدیران دو واحد"},
		{Kind: "transfers", Title: "جابه‌جایی — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"fromUnit": "تولید و عملیات", "toUnit": "تولید و عملیات", "effectiveDate": "1404/06/01", "reason": "جابه‌جایی بین خطوط"},
			NoteFa: "اجراشده"},
		{Kind: "introductions", Title: "معرفی مریم نوری به واحد آموزش", EmployeeID: "EMP-008",
			Fields: map[string]string{"toUnit": "آموزش و توسعه", "effectiveDate": "1404/08/15", "letterNo": "INT-1404-33"},
			NoteFa: "ابلاغ‌شده"},
	}
}
