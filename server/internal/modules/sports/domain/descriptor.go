// Package domain دامنهٔ ماژول امور ورزش.
//
// رشتهٔ ورزشی و رویداد ورزشی هر یک یک «نوع رکورد» با گردش‌کار خودش است؛ قواعد در همین
// بسته اعلام می‌شوند و موتور مشترک `recordkit` آن‌ها را اجرا می‌کند.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول امور ورزش.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "sports",
		TitleFa: "امور ورزش",
		Kinds: []recordkit.Kind{
			{
				Key: "sportsPrograms", TitleFa: "برنامهٔ ورزشی", IDPrefix: "SPP", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"sport|رشتهٔ ورزشی|text|1",
					"coach|مربی|text|0",
					"place|محل برگزاری|text|0",
					"capacity|ظرفیت|number|0",
					"fee|حق ثبت‌نام|money|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال", "closed|بسته‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|active,closed", "active|closed"),
				Actions: recordkit.Actions(
					"activate|فعال‌سازی برنامه|primary|active|0",
					"close|بستن برنامه|ghost|closed|0",
				),
			},
			{
				Key: "sportsEvents", TitleFa: "رویداد ورزشی", IDPrefix: "SPE", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"eventDate|تاریخ رویداد|date|1",
					"sport|رشتهٔ ورزشی|text|1",
					"venue|محل برگزاری|text|0",
					"participants|تعداد شرکت‌کنندگان|number|0",
				),
				Statuses:    recordkit.Statuses("planned|برنامه‌ریزی‌شده", "held|برگزارشده", "canceled|لغو‌شده"),
				Initial:     "planned",
				Transitions: recordkit.Steps("planned|held,canceled"),
				Actions: recordkit.Actions(
					"hold|ثبت برگزاری|primary|held|0",
					"cancel|لغو رویداد|danger|canceled|1",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "sportsPrograms", Title: "برنامهٔ ورزشی — والیبال سالنی",
			Fields: map[string]string{"sport": "والیبال", "coach": "مربی رسمی فدراسیون", "place": "سالن ورزشی سازمان", "capacity": "40", "fee": "0"},
			NoteFa: "برنامهٔ فعال برای کارکنان"},
		{Kind: "sportsPrograms", Title: "برنامهٔ ورزشی — شنا",
			Fields: map[string]string{"sport": "شنا", "coach": "مربی نجات غریق", "place": "استخر سازمان", "capacity": "30", "fee": "5000000"},
			NoteFa: "پیش‌نویس؛ در انتظار تأیید"},
		{Kind: "sportsEvents", Title: "المپیاد ورزشی کارکنان — دورهٔ پاییز",
			Fields: map[string]string{"eventDate": "1404/09/15", "sport": "فوتبال، والیبال، تنیس روی میز", "venue": "مجموعهٔ ورزشی سازمان", "participants": "120"},
			NoteFa: "در حال برنامه‌ریزی"},
	}
}
