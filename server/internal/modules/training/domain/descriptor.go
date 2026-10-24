// Package domain دامنهٔ ماژول هماهنگی آموزش.
//
// دورهٔ آموزشی، نیاز آموزشی و ثبت‌نام کارمند در دوره هر یک یک «نوع رکورد» با گردش‌کار
// خودش است.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول هماهنگی آموزش.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "training",
		TitleFa: "هماهنگی آموزش",
		Kinds: []recordkit.Kind{
			{
				Key: "courses", TitleFa: "دورهٔ آموزشی", IDPrefix: "CRS", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"title|عنوان دوره|text|1",
					"instructor|مدرس|text|0",
					"startDate|تاریخ شروع|date|0",
					"endDate|تاریخ پایان|date|0",
					"hours|ساعت دوره|number|0",
				),
				Statuses: recordkit.Statuses(
					"planned|برنامه‌ریزی‌شده", "open|ثبت‌نام باز", "running|در حال اجرا", "finished|پایان‌یافته", "cancelled|لغوشده",
				),
				Initial: "planned",
				Transitions: recordkit.Steps(
					"planned|open,cancelled", "open|running,cancelled", "running|finished,cancelled",
				),
				Actions: recordkit.Actions(
					"open|بازکردن ثبت‌نام|primary|open|0",
					"start|آغاز دوره|primary|running|0",
					"finish|پایان دوره|primary|finished|0",
					"cancel|لغو دوره|danger|cancelled|1",
				),
			},
			{
				Key: "trainingNeeds", TitleFa: "نیاز آموزشی", IDPrefix: "NEED", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"unit|واحد|text|1",
					"skill|مهارت مورد نیاز|text|0",
					"priority|اولویت|select|0",
					"year|سال|text|0",
				),
				Statuses:    recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده", "planned|برنامه‌ریزی‌شده"),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|approved", "approved|planned"),
				Actions: recordkit.Actions(
					"approve|تأیید نیاز|primary|approved|0",
					"plan|برنامه‌ریزی دوره|primary|planned|0",
				),
			},
			{
				Key: "enrollments", TitleFa: "ثبت‌نام در دوره", IDPrefix: "ENR", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"course|دوره|text|1",
					"score|نمره|number|0",
					"result|نتیجه|select|0",
				),
				Statuses:    recordkit.Statuses("registered|ثبت‌نام‌شده", "attended|حاضر شده", "passed|قبول", "failed|مردود"),
				Initial:     "registered",
				Transitions: recordkit.Steps("registered|attended", "attended|passed,failed"),
				Actions: recordkit.Actions(
					"attend|ثبت حضور|primary|attended|0",
					"pass|قبول در دوره|primary|passed|0",
					"fail|مردود|danger|failed|1",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "courses", Title: "دورهٔ مدیریت منابع انسانی راهبردی",
			Fields: map[string]string{"title": "مدیریت منابع انسانی راهبردی", "instructor": "دکتر رحیمی", "startDate": "1404/08/01", "endDate": "1404/09/15", "hours": "60"},
			NoteFa: "ثبت‌نام باز"},
		{Kind: "trainingNeeds", Title: "نیاز آموزشی واحد تولید",
			Fields: map[string]string{"unit": "تولید و عملیات", "skill": "ایمنی و بهداشت کار", "priority": "بالا", "year": "1404"},
			NoteFa: "تأییدشده"},
		{Kind: "enrollments", Title: "ثبت‌نام مریم نوری در دورهٔ منابع انسانی", EmployeeID: "EMP-008",
			Fields: map[string]string{"course": "مدیریت منابع انسانی راهبردی", "result": "قبول", "score": "93"},
			NoteFa: "قبول با نمرهٔ عالی"},
	}
}
