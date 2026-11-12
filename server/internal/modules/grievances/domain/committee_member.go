// اعضای کمیتهٔ ایثارگری: تعریف رسمی کسانی که پای صورت‌جلسهٔ مصوبه امضا می‌کنند.
//
// چرا یک نوع رکورد جدا از خود مصوبه: ترکیب اعضا در هر جلسه یکی نیست و امضاکنندگان برگه باید
// از یک فهرست مصوب سازمانی بیایند، نه از متنی که هر بار در فرم تایپ شود. این رکورد
// «تعریف عضو» است (نام، نقش، سمت، شمارهٔ ابلاغ و ترتیب امضا) و مصوبه فقط از آن استفاده
// می‌کند؛ پس تغییر ترکیب کمیته، مصوبه‌های گذشته را تغییر نمی‌دهد.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// veteranCommitteeMemberKind تعریف عضو کمیتهٔ ایثارگری.
//
// عضو «کارمندمحور» نیست: نمایندهٔ بنیاد شهید یا ناظر بیرونی هم عضو کمیته است و لازم نیست
// کارمند سازمان باشد؛ به همین دلیل کد پرسنلی اختیاری است.
func veteranCommitteeMemberKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindVeteranCommitteeMember, TitleFa: "عضو کمیتهٔ ایثارگری", IDPrefix: "VCMEM",
		EmployeeScoped: false,
		Fields: recordkit.Fields(
			"fullName|نام و نام خانوادگی|text|1",
			"role|نقش در کمیته|select|1",
			"positionTitle|سمت سازمانی|text|0",
			"employeeId|کد پرسنلی عضو (اختیاری)|text|0",
			"orderNo|شمارهٔ ابلاغ عضویت|text|0",
			"orderDate|تاریخ ابلاغ|date|0",
			"signatureTitle|عنوان امضا در برگه|text|0",
			"sortOrder|ترتیب امضا|number|0",
		),
		Statuses: recordkit.Statuses(
			"active|عضو فعال", "inactive|غیرفعال",
		),
		Initial: "active",
		Transitions: recordkit.Steps(
			"active|inactive", "inactive|active",
		),
		Actions: recordkit.Actions(
			"deactivate|غیرفعال‌کردن عضویت|ghost|inactive|1",
			"activate|فعال‌کردن عضویت|primary|active|0",
		),
		HistoryDate:      "orderDate",
		Validate:         validateVeteranCommitteeMember,
		EditableStatuses: []string{"active", "inactive"},
	}
}

// validateVeteranCommitteeMember قاعده‌های تعریف عضو: نام و نقش روشن باشد، تاریخ ابلاغ
// معتبر و ترتیب امضا در بازهٔ معقول باشد؛ نقشی که در فهرست مصوب نیست پذیرفته نمی‌شود.
func validateVeteranCommitteeMember(in contracts.RecordInput) error {
	if strings.TrimSpace(in.Fields["fullName"]) == "" {
		return &recordkit.RuleError{Message: "نام و نام خانوادگی عضو کمیته را بنویسید."}
	}
	role := strings.TrimSpace(in.Fields["role"])
	known := false
	for _, item := range rules.VeteranCommitteeRoles() {
		if item.Key == role {
			known = true
			break
		}
	}
	if !known {
		return &recordkit.RuleError{Message: "نقش عضو را از فهرست مصوب کمیته انتخاب کنید."}
	}
	if raw := calendar.NormalizeDigits(strings.TrimSpace(in.Fields["orderDate"])); raw != "" {
		if _, err := calendar.Parse(raw); err != nil {
			return &recordkit.RuleError{Message: "تاریخ ابلاغ عضویت معتبر نیست (قالب YYYY/MM/DD)."}
		}
	}
	if order := numberValue(in.Fields["sortOrder"]); order < 0 || order > 99 {
		return &recordkit.RuleError{Message: "ترتیب امضا باید بین صفر و ۹۹ باشد."}
	}
	if employeeID := strings.TrimSpace(in.Fields["employeeId"]); employeeID != "" && len(employeeID) > 20 {
		return &recordkit.RuleError{Message: "کد پرسنلی عضو معتبر نیست."}
	}
	return nil
}

// committeeMemberSeeds اعضای پیش‌فرض کمیتهٔ سازمان نمونه: رئیس، دبیر، نمایندهٔ امور
// ایثارگران، نمایندهٔ بنیاد شهید و یک عضو؛ ترتیب امضا با فیلد sortOrder تعیین می‌شود.
func committeeMemberSeeds() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: KindVeteranCommitteeMember, Title: "عضو کمیتهٔ ایثارگری — رئیس کمیته",
			Fields: map[string]string{
				"fullName": "سارا محمدی", "role": rules.VeteranCommitteeChairman,
				"positionTitle": "مدیر منابع انسانی", "employeeId": "EMP-001",
				"orderNo": "۱۴۰۵/ح/۱۰۲", "orderDate": "1405/01/05", "sortOrder": "1",
			},
			SeedStatus: "active", NoteFa: "ریاست کمیته و امضای نخست صورت‌جلسه"},
		{Kind: KindVeteranCommitteeMember, Title: "عضو کمیتهٔ ایثارگری — دبیر کمیته",
			Fields: map[string]string{
				"fullName": "مریم نوری", "role": rules.VeteranCommitteeSecretary,
				"positionTitle": "کارشناس ارشد منابع انسانی", "employeeId": "EMP-008",
				"orderNo": "۱۴۰۵/ح/۱۰۳", "orderDate": "1405/01/05", "sortOrder": "2",
			},
			SeedStatus: "active", NoteFa: "تنظیم صورت‌جلسه و ثبت مصوبه در سامانه"},
		{Kind: KindVeteranCommitteeMember, Title: "عضو کمیتهٔ ایثارگری — نمایندهٔ امور ایثارگران",
			Fields: map[string]string{
				"fullName": "رضا کاظمی", "role": rules.VeteranCommitteeVeteranRep,
				"positionTitle": "کارشناس امور ایثارگران", "employeeId": "EMP-007",
				"orderNo": "۱۴۰۵/ح/۱۰۴", "orderDate": "1405/01/05", "sortOrder": "3",
			},
			SeedStatus: "active", NoteFa: "تهیهٔ مستندات پرونده و ارائهٔ گزارش به کمیته"},
		{Kind: KindVeteranCommitteeMember, Title: "عضو کمیتهٔ ایثارگری — نمایندهٔ بنیاد شهید",
			Fields: map[string]string{
				"fullName": "مهدی رحیمی", "role": rules.VeteranCommitteeFoundation,
				"positionTitle": "نمایندهٔ بنیاد شهید و امور ایثارگران",
				"orderNo":       "۱۴۰۵/ب/۲۲۱", "orderDate": "1405/01/12", "sortOrder": "4",
				"signatureTitle": "نمایندهٔ بنیاد شهید و امور ایثارگران",
			},
			SeedStatus: "active", NoteFa: "عضو بیرونی؛ تأیید مستندات مرجع حمایتی"},
		{Kind: KindVeteranCommitteeMember, Title: "عضو کمیتهٔ ایثارگری — عضو کمیته",
			Fields: map[string]string{
				"fullName": "علی مرادی", "role": rules.VeteranCommitteeMember,
				"positionTitle": "مدیر بهره‌برداری", "employeeId": "EMP-004",
				"orderNo": "۱۴۰۵/ح/۱۰۵", "orderDate": "1405/01/05", "sortOrder": "5",
			},
			SeedStatus: "active", NoteFa: "عضو دارای حق رأی در جلسهٔ کمیته"},
	}
}
