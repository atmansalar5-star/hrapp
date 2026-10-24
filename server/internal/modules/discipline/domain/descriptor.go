// Package domain دامنهٔ ماژول کمیتهٔ انضباط کار.
//
// پروندهٔ انضباطی و جلسهٔ کمیته هر یک یک «نوع رکورد» با گردش‌کار خودش است. رعایت
// محرمانگی در سطح قاعدهٔ اقدام دیده شده: صدور رأی و بایگانی نیازمند ثبت دلیل‌اند.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول کمیتهٔ انضباط کار.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "discipline",
		TitleFa: "کمیتهٔ انضباط کار",
		Kinds: []recordkit.Kind{
			{
				Key: "disciplineCases", TitleFa: "پروندهٔ انضباطی", IDPrefix: "DISC", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"charge|موضوع تخلف|text|1",
					"reportDate|تاریخ گزارش|date|0",
					"defense|دفاعیات کارمند|text|0",
					"ruling|متن رأی|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|تشکیل‌شده", "investigating|در حال بررسی", "hearing|در نوبت جلسه", "decided|رأی صادر شد", "archived|بایگانی‌شده",
				),
				Initial: "draft",
				Transitions: recordkit.Steps(
					"draft|investigating,archived", "investigating|hearing,archived",
					"hearing|decided,archived", "decided|archived",
				),
				Actions: recordkit.Actions(
					"investigate|شروع بررسی|primary|investigating|0",
					"hearing|ارجاع به جلسه|primary|hearing|0",
					"decide|ثبت رأی کمیته|primary|decided|1",
					"archive|بایگانی پرونده|ghost|archived|1",
				),
			},
			{
				Key: "disciplineMeetings", TitleFa: "جلسهٔ کمیته", IDPrefix: "MTG", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"meetingDate|تاریخ جلسه|date|1",
					"members|اعضای حاضر|text|0",
					"agenda|دستور جلسه|text|0",
					"minutes|صورت‌جلسه|text|0",
				),
				Statuses: recordkit.Statuses(
					"scheduled|زمان‌بندی‌شده", "held|برگزارشده", "reported|صورت‌جلسه ثبت شد",
				),
				Initial:     "scheduled",
				Transitions: recordkit.Steps("scheduled|held", "held|reported"),
				Actions: recordkit.Actions(
					"hold|ثبت برگزاری|primary|held|0",
					"report|ثبت صورت‌جلسه|primary|reported|0",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "disciplineCases", Title: "پروندهٔ انضباطی — تأخیر مکرر", EmployeeID: "EMP-007",
			Fields: map[string]string{"charge": "تأخیر مکرر بدون اطلاع", "reportDate": "1404/06/20"},
			NoteFa: "در حال بررسی توسط کارشناس روابط کار"},
		{Kind: "disciplineCases", Title: "پروندهٔ انضباطی — غیبت غیرموجه", EmployeeID: "EMP-009",
			Fields: map[string]string{"charge": "غیبت غیرموجه سه روز", "reportDate": "1404/07/02", "defense": "ارائهٔ گواهی پزشکی"},
			NoteFa: "ارجاع به جلسهٔ کمیته"},
		{Kind: "disciplineMeetings", Title: "جلسهٔ کمیتهٔ انضباط — مهر ۱۴۰۴",
			Fields: map[string]string{"meetingDate": "1404/07/18", "members": "مدیر منابع انسانی، کارشناس روابط کار، نمایندهٔ کارکنان", "agenda": "رسیدگی به دو پروندهٔ انضباطی"},
			NoteFa: "زمان‌بندی‌شده؛ دعوت‌نامه‌ها ارسال شد"},
	}
}
