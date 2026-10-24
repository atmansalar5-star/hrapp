// Package domain دامنهٔ ماژول ساختار سازمانی و تشکیلات.
//
// شناسنامهٔ پست سازمانی یک «نوع رکورد» است؛ نمودار درختی واحدها از دادهٔ ساختاری سازمان
// (departments.json) ساخته می‌شود و اینجا فقط پست‌ها به‌عنوان مرجع انتصاب/ترفیع مدیریت می‌شوند.
package domain

import (
	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول ساختار سازمانی.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "orgstructure",
		TitleFa: "ساختار سازمانی و تشکیلات",
		Kinds: []recordkit.Kind{
			{
				Key: "positions", TitleFa: "پست سازمانی", IDPrefix: "POS", EmployeeScoped: false,
				Fields: recordkit.Fields(
					"code|کد پست|text|1",
					"unit|واحد سازمانی|text|1",
					"jobFamily|خانوادهٔ شغلی|text|0",
					"grade|گروه شغلی|text|0",
					"requirements|شرط‌های احراز|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "active|فعال", "vacant|بلا تصدی", "archived|بایگانی‌شده",
				),
				Initial:     "draft",
				Transitions: recordkit.Steps("draft|active,archived", "active|vacant,archived", "vacant|active,archived"),
				Actions: recordkit.Actions(
					"approve|تأیید و فعال‌سازی پست|primary|active|0",
					"vacate|ثبت بلا تصدی|ghost|vacant|0",
					"archive|بایگانی پست|ghost|archived|1",
				),
			},
		},
	}
}

// Seed رکوردهای نمونهٔ سازمان.
func Seed() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: "positions", Title: "کارشناس ارشد مالی",
			Fields: map[string]string{"code": "POS-201", "unit": "امور مالی", "jobFamily": "مالی و حسابداری", "grade": "کارشناسی ارشد", "requirements": "کارشناسی ارشد حسابداری، ۵ سال سابقه"},
			NoteFa: "پست مرجع برای ترفیع کارشناسان مالی"},
		{Kind: "positions", Title: "مدیر تولید",
			Fields: map[string]string{"code": "POS-310", "unit": "تولید و عملیات", "jobFamily": "مدیریت تولید", "grade": "مدیریت", "requirements": "کارشناسی مهندسی صنایع، ۱۰ سال سابقهٔ سرپرستی"}},
		{Kind: "positions", Title: "کارشناس منابع انسانی",
			Fields: map[string]string{"code": "POS-120", "unit": "منابع انسانی", "jobFamily": "منابع انسانی", "grade": "کارشناسی"},
			NoteFa: "بلا تصدی؛ درخواست نیرو ثبت شده"},
		{Kind: "positions", Title: "سرپرست فروش",
			Fields: map[string]string{"code": "POS-405", "unit": "فروش و بازاریابی", "jobFamily": "فروش", "grade": "سرپرستی"}},
	}
}
