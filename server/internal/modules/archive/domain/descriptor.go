// Package domain دامنهٔ ماژول بایگانی و شناسایی کارکنان.
//
// قواعد این ماژول به‌صورت «شناسهٔ رکورد» اعلام می‌شود و موتور مشترک `recordkit` آن را
// اجرا می‌کند؛ پس پروندهٔ پرسنلی، مشخصات هویتی و اسناد بایگانی هر یک یک «نوع رکورد» با
// فیلدها، وضعیت‌ها، انتقال‌ها و اقدام‌های خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول بایگانی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "archive",
		TitleFa: "بایگانی و شناسایی کارکنان",
		Kinds: []recordkit.Kind{
			{
				Key: "personnelFiles", TitleFa: "پروندهٔ پرسنلی", IDPrefix: "FILE", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"fileNo|شمارهٔ پرونده|text|1",
					"docsCount|تعداد مدارک|number|0",
					"lastAudit|آخرین کنترل کامل بودن|date|0",
				),
				Statuses: recordkit.Statuses(
					"incomplete|ناقص", "complete|کامل", "archived|بایگانی‌شده",
				),
				Initial:     "incomplete",
				Transitions: recordkit.Steps("incomplete|complete,archived", "complete|archived"),
				Actions: recordkit.Actions(
					"complete|تأیید کامل بودن پرونده|primary|complete|0",
					"archive|بایگانی پرونده|ghost|archived|1",
				),
			},
			{
				Key: "employeeIdentity", TitleFa: "مشخصات هویتی", IDPrefix: "IDNT", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"nationalId|کد ملی|text|1",
					"birthDate|تاریخ تولد|date|0",
					"maritalStatus|وضعیت تأهل|select|0",
					"idCardNo|شمارهٔ کارت شناسایی|text|0",
					"cardExpiry|انقضای کارت|date|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "issued|کارت صادرشده", "expired|منقضی",
				),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|issued", "issued|expired"),
				Actions: recordkit.Actions(
					"issue|صدور کارت شناسایی|primary|issued|0",
					"expire|ثبت انقضا|ghost|expired|0",
				),
			},
			{
				Key: "archiveDocuments", TitleFa: "سند بایگانی و امانت", IDPrefix: "DOC", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"docType|نوع سند|text|1",
					"docNo|شمارهٔ سند|text|0",
					"lendDate|تاریخ امانت|date|0",
					"returnDate|تاریخ بازگشت|date|0",
				),
				Statuses: recordkit.Statuses(
					"stored|در بایگانی", "lent|امانت", "returned|بازگشتی",
				),
				Initial:     "stored",
				Transitions: recordkit.Steps("stored|lent", "lent|returned"),
				Actions: recordkit.Actions(
					"lend|ثبت امانت|primary|lent|0",
					"return|ثبت بازگشت|primary|returned|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان برای نمایش اولیهٔ صفحه‌ها.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "personnelFiles", Title: "پروندهٔ پرسنلی — سارا محمدی", EmployeeID: "EMP-001",
			Fields: map[string]string{"fileNo": "FILE-1001", "docsCount": "42", "lastAudit": "1404/06/10"},
			NoteFa: "پرونده کامل و کنترل‌شده"},
		{Kind: "personnelFiles", Title: "پروندهٔ پرسنلی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"fileNo": "FILE-1003", "docsCount": "17"},
			NoteFa: "مدارک تحصیلی ناقص است"},
		{Kind: "employeeIdentity", Title: "مشخصات هویتی — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"nationalId": "0071234567", "birthDate": "1365/04/02", "maritalStatus": "متأهل", "idCardNo": "ID-55210", "cardExpiry": "1406/04/02"}},
		{Kind: "employeeIdentity", Title: "مشخصات هویتی — مریم نوری", EmployeeID: "EMP-008",
			Fields: map[string]string{"nationalId": "0089876543", "birthDate": "1370/11/19", "maritalStatus": "مجرد"}},
		{Kind: "archiveDocuments", Title: "امانت مدرک تحصیلی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"docType": "گواهی موقت تحصیلی", "docNo": "DOC-7781", "lendDate": "1404/07/05"},
			NoteFa: "جهت ارائه به دانشگاه"},
		{Kind: "archiveDocuments", Title: "بایگانی حکم کارگزینی — سارا محمدی", EmployeeID: "EMP-001",
			Fields: map[string]string{"docType": "حکم کارگزینی", "docNo": "DOC-6402"}},
	}
}
