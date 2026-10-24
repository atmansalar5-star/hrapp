// Package domain دامنهٔ ماژول مددکاری اجتماعی.
//
// پروندهٔ مددکاری و درخواست کمک موردی هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول مددکاری اجتماعی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "socialwork",
		TitleFa: "مددکاری اجتماعی",
		Kinds: []recordkit.Kind{
			{
				Key: "socialWorkCases", TitleFa: "پروندهٔ مددکاری", IDPrefix: "SW", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"caseType|نوع پرونده|text|1",
					"openedDate|تاریخ تشکیل|date|0",
					"counselor|مددکار|text|0",
					"summary|خلاصهٔ وضعیت|text|0",
				),
				Statuses:    recordkit.Statuses("open|باز", "treating|در حال پیگیری", "closed|بسته‌شده"),
				Initial:     "open",
				Transitions: recordkit.Steps("open|treating,closed", "treating|closed"),
				Actions: recordkit.Actions(
					"counsel|ثبت اقدام مددکاری|primary|treating|0",
					"close|بستن پرونده|ghost|closed|1",
				),
			},
			{
				Key: "aidRequests", TitleFa: "درخواست کمک موردی", IDPrefix: "AID", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"aidType|نوع کمک|text|1",
					"amount|مبلغ|money|0",
					"reason|دلیل|text|0",
				),
				Statuses:    recordkit.Statuses("requested|درخواست‌شده", "approved|تأییدشده", "rejected|ردشده", "paid|پرداخت‌شده"),
				Initial:     "requested",
				Transitions: recordkit.Steps("requested|approved,rejected", "approved|paid"),
				Actions: recordkit.Actions(
					"approve|تأیید کمک|primary|approved|0",
					"reject|رد درخواست|danger|rejected|1",
					"pay|ثبت پرداخت|primary|paid|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "socialWorkCases", Title: "پروندهٔ مددکاری — سرپرست خانوار", EmployeeID: "EMP-003",
			Fields: map[string]string{"caseType": "حمایت از سرپرست خانوار", "openedDate": "1404/05/12", "counselor": "مددکار واحد رفاه", "summary": "نیازمند حمایت معیشتی و مشاوره"},
			NoteFa: "در حال پیگیری"},
		{Kind: "socialWorkCases", Title: "پروندهٔ مددکاری — مشاورهٔ خانوادگی", EmployeeID: "EMP-007",
			Fields: map[string]string{"caseType": "مشاوره", "openedDate": "1404/06/02", "summary": "ارجاع به مشاور"}},
		{Kind: "aidRequests", Title: "کمک درمانی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"aidType": "کمک درمانی", "amount": "45000000", "reason": "هزینهٔ درمان جراحی"},
			NoteFa: "در انتظار تأیید کمیتهٔ مددکاری"},
	}
}
