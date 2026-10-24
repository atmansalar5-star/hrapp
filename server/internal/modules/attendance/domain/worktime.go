package domain

import "strings"

// LeaveClass دستهٔ یک نوع مرخصی از دید «کارکرد و حقوق».
//
// این جدول قاعدهٔ دامنه است، نه رابط کاربری: می‌گوید کدام مرخصی روزِ پرداختی است و
// کدام روزِ کسری. افزودن نوع مرخصی تازه فقط اینجا انجام می‌شود.
type LeaveClass struct {
	Key    string `json:"key"`
	Paid   bool   `json:"paid"`
	Sick   bool   `json:"sick"`
	NoteFa string `json:"noteFa"`
}

// leaveClasses فهرست انواع مرخصی شناخته‌شده در این استقرار.
func leaveClasses() []LeaveClass {
	return []LeaveClass{
		{Key: "استحقاقی", Paid: true, NoteFa: "مرخصی استحقاقی؛ روز پرداختی و از ماندهٔ مرخصی کسر می‌شود."},
		{Key: "استعلاجی", Paid: true, Sick: true, NoteFa: "مرخصی استعلاجی؛ روز پرداختی با تأییدیهٔ پزشک."},
		{Key: "ساعتی", Paid: true, NoteFa: "مرخصی ساعتی؛ ساعت کسری در کارکرد دوره لحاظ می‌شود."},
		{Key: "زایمان", Paid: true, NoteFa: "مرخصی زایمان؛ طبق قانون کار پرداختی است."},
		{Key: "بدون حقوق", Paid: false, NoteFa: "مرخصی بدون حقوق؛ روز کسری و مبنای کسر از مزد دوره."},
		{Key: "غیبت", Paid: false, NoteFa: "غیبت؛ روز کسری و مبنای کسر از مزد دوره."},
	}
}

// LeaveClasses فهرست انواع مرخصی برای نمایش و مصرف‌کننده‌های دیگر.
func LeaveClasses() []LeaveClass { return leaveClasses() }

// ClassifyLeave نوع مرخصی را به دستهٔ کارکرد نگاشت می‌کند. نوع ناشناخته محافظه‌کارانه
// «پرداختی» در نظر گرفته می‌شود تا کسر ناخواسته از حقوق رخ ندهد؛ کارشناس منابع انسانی
// اگر لازم باشد نوع را در فهرست بالا می‌افزاید.
func ClassifyLeave(leaveType string) LeaveClass {
	key := strings.TrimSpace(leaveType)
	for _, c := range leaveClasses() {
		if c.Key == key {
			return c
		}
	}
	return LeaveClass{
		Key: key, Paid: true,
		NoteFa: "نوع مرخصی در فهرست دامنه نبود؛ محافظه‌کارانه پرداختی در نظر گرفته شد.",
	}
}

// EffectiveStatus می‌گوید رکورد در محاسبهٔ کارکرد «مؤثر» است یا هنوز «در گردش».
//
// رکورد مؤثر = ثبت‌شده/تأییدشده/تسویه‌شده؛ رکورد در گردش = پیش‌نویس، در انتظار
// تأیید، تأیید مدیر واحد یا دارای اعتراض. کارکرد حقوق فقط از رکوردهای مؤثر ساخته
// می‌شود تا پرداخت پیش از تأیید رخ ندهد.
func EffectiveStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "recorded", "confirmed", "approved", "settled":
		return true
	}
	return false
}
