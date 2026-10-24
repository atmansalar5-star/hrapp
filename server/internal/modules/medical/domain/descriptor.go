// Package domain دامنهٔ ماژول درمان، معاینات دوره‌ای و خدمات درمانی.
//
// پروندهٔ درمان، معاینهٔ دوره‌ای و خدمت درمانی هر یک یک «نوع رکورد» با گردش‌کار خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول درمان و خدمات درمانی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "medical",
		TitleFa: "درمان، معاینات دوره‌ای و خدمات درمانی",
		Kinds: []recordkit.Kind{
			{
				Key: "treatmentCases", TitleFa: "پروندهٔ درمان", IDPrefix: "MED", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"diagnosis|تشخیص|text|1",
					"visitDate|تاریخ مراجعه|date|0",
					"provider|پزشک/مرکز|text|0",
					"prescription|نسخه|text|0",
				),
				Statuses:    recordkit.Statuses("open|باز", "treating|در حال درمان", "closed|بسته‌شده"),
				Initial:     "open",
				Transitions: recordkit.Steps("open|treating,closed", "treating|closed"),
				Actions: recordkit.Actions(
					"treat|ثبت درمان|primary|treating|0",
					"close|بستن پرونده|ghost|closed|0",
				),
			},
			{
				Key: "periodicExams", TitleFa: "معاینهٔ دوره‌ای", IDPrefix: "EXM", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"examDate|تاریخ معاینه|date|1",
					"examType|نوع معاینه|text|0",
					"result|نتیجه|select|0",
					"nextDate|معاینهٔ بعدی|date|0",
				),
				Statuses:    recordkit.Statuses("scheduled|زمان‌بندی‌شده", "done|انجام‌شده", "passed|سالم", "failed|نیازمند پیگیری"),
				Initial:     "scheduled",
				Transitions: recordkit.Steps("scheduled|done", "done|passed,failed"),
				Actions: recordkit.Actions(
					"record|ثبت انجام معاینه|primary|done|0",
					"pass|تأیید سلامتی|primary|passed|0",
					"followup|نیازمند پیگیری|danger|failed|1",
				),
			},
			{
				Key: "supplementaryInsurance", TitleFa: "بیمهٔ تکمیلی", IDPrefix: "INS", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"policyNo|شمارهٔ بیمه‌نامه|text|0",
					"insurer|شرکت بیمه‌گر|text|0",
					"coverage|درصد پوشش|number|0",
					"dependents|تعداد افراد تحت تکفل|number|0",
					"premium|حق بیمهٔ ماهانه|money|0",
					"startDate|آغاز اعتبار|date|0",
					"endDate|پایان اعتبار|date|0",
				),
				Statuses:    recordkit.Statuses("active|فعال", "suspended|معلق", "expired|منقضی"),
				Initial:     "active",
				Transitions: recordkit.Steps("active|suspended,expired", "suspended|active", "expired|active"),
				Actions: recordkit.Actions(
					"renew|تمدید اعتبار|primary|active|0",
					"activate|فعال‌سازی|primary|active|0",
					"suspend|تعلیق پوشش|danger|suspended|1",
				),
				EditableStatuses: []string{"active", "suspended", "expired"},
			},
			{
				Key: "medicalServices", TitleFa: "خدمت درمانی", IDPrefix: "MSV", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"serviceName|نام خدمت|text|1",
					"coverage|پوشش بیمه|text|0",
					"cost|هزینه|money|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved"),
				Actions: recordkit.Actions(
					"approve|تأیید خدمت|primary|approved|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "treatmentCases", Title: "پروندهٔ درمان — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"diagnosis": "کمردرد ناشی از کار", "visitDate": "1404/06/12", "provider": "درمانگاه سازمان", "prescription": "فیزیوتراپی ۱۰ جلسه"},
			NoteFa: "در حال درمان"},
		{Kind: "periodicExams", Title: "معاینهٔ دوره‌ای — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"examDate": "1404/05/20", "examType": "معاینهٔ شغلی سالانه", "result": "سالم", "nextDate": "1405/05/20"},
			NoteFa: "معاینه انجام و تأیید شد"},
		{Kind: "medicalServices", Title: "خدمت درمانی — دندان‌پزشکی",
			Fields: map[string]string{"serviceName": "دندان‌پزشکی", "coverage": "۷۰٪ بیمهٔ تکمیلی", "cost": "4500000"}},
		{Kind: "supplementaryInsurance", Title: "بیمهٔ تکمیلی — محمد احمدی", EmployeeID: "EMP-005",
			Fields: map[string]string{"policyNo": "SUP-1404-00182", "insurer": "بیمهٔ دانا", "coverage": "80", "dependents": "4", "premium": "3200000", "startDate": "1404/01/01", "endDate": "1404/12/29"},
			NoteFa: "پوشش فعال کارمند و خانواده"},
		{Kind: "supplementaryInsurance", Title: "بیمهٔ تکمیلی — نگار حسینی", EmployeeID: "EMP-003",
			Fields: map[string]string{"policyNo": "SUP-1404-00097", "insurer": "بیمهٔ ایران", "coverage": "70", "dependents": "1", "premium": "2100000", "startDate": "1404/02/15", "endDate": "1405/02/14"},
			NoteFa: "در انتظار تمدید"},
	}
}
