// نقش‌های عضو کمیتهٔ ایثارگری و ترکیب متن امضای پایین فرم.
//
// چرا این فهرست در موتور قواعد است و نه در رابط: عنوان امضای برگهٔ رسمی («رئیس کمیته — نام
// عضو») یک قاعدهٔ نمایشیِ مشترک است؛ هم برگهٔ چاپی سمت سرور و هم فرم رابط باید یک متن را
// بسازند. تعریف یک‌جا در `internal/rules` جلوی دوشقه‌شدن عنوان‌ها را می‌گیرد و ترتیب نشستن
// امضاها را هم همان‌جا تعیین می‌کند.
package rules

import "strings"

// نقش‌های استاندارد عضو کمیتهٔ ایثارگری.
const (
	// VeteranCommitteeChairman رئیس کمیته.
	VeteranCommitteeChairman = "chairman"
	// VeteranCommitteeSecretary دبیر کمیته.
	VeteranCommitteeSecretary = "secretary"
	// VeteranCommitteeVeteranRep نمایندهٔ امور ایثارگران سازمان.
	VeteranCommitteeVeteranRep = "veteran_rep"
	// VeteranCommitteeFoundation نمایندهٔ بنیاد شهید و امور ایثارگران.
	VeteranCommitteeFoundation = "foundation_rep"
	// VeteranCommitteeMember عضو عادی کمیته.
	VeteranCommitteeMember = "member"
	// VeteranCommitteeObserver ناظر یا مشاور کمیته (بدون حق رأی).
	VeteranCommitteeObserver = "observer"
	// VeteranCommitteeHRSeal مهر و امضای مدیریت منابع انسانی پای سند.
	VeteranCommitteeHRSeal = "hr_seal"
)

// VeteranCommitteeRole یک نقش در کمیته: کلید ماشین‌خوان، عنوان فارسی، ترتیب نشستن امضا و
// توضیح کوتاه برای راهنمای فرم.
type VeteranCommitteeRole struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa"`
	Order   int    `json:"order"`
}

// VeteranCommitteeRoles فهرست نقش‌ها به ترتیب نشستن امضا در پایین برگه.
func VeteranCommitteeRoles() []VeteranCommitteeRole {
	return []VeteranCommitteeRole{
		{Key: VeteranCommitteeChairman, TitleFa: "رئیس کمیته",
			NoteFa: "ریاست جلسه و امضای نخست صورت‌جلسه.", Order: 1},
		{Key: VeteranCommitteeSecretary, TitleFa: "دبیر کمیته",
			NoteFa: "تنظیم صورت‌جلسه و ثبت مصوبه در سامانه.", Order: 2},
		{Key: VeteranCommitteeVeteranRep, TitleFa: "نمایندهٔ امور ایثارگران",
			NoteFa: "کارشناس امور ایثارگران سازمان؛ تهیهٔ مستندات پرونده.", Order: 3},
		{Key: VeteranCommitteeFoundation, TitleFa: "نمایندهٔ بنیاد شهید و امور ایثارگران",
			NoteFa: "عضو بیرونی؛ تأیید مستندات مرجع حمایتی.", Order: 4},
		{Key: VeteranCommitteeMember, TitleFa: "عضو کمیته",
			NoteFa: "سایر اعضای دارای حق رأی.", Order: 5},
		{Key: VeteranCommitteeObserver, TitleFa: "ناظر یا مشاور",
			NoteFa: "بدون حق رأی؛ فقط امضای اطلاع از مفاد مصوبه.", Order: 6},
		{Key: VeteranCommitteeHRSeal, TitleFa: "مهر و امضای مدیریت منابع انسانی",
			NoteFa: "تأیید نهایی و مهر سازمان پای برگه.", Order: 7},
	}
}

// VeteranCommitteeRoleOf کلید نقش را نرمال می‌کند؛ مقدار ناشناخته «عضو کمیته» می‌شود تا
// عضویت تازه هرگز بی‌نقش نماند.
func VeteranCommitteeRoleOf(raw string) string {
	value := strings.TrimSpace(raw)
	for _, role := range VeteranCommitteeRoles() {
		if role.Key == value {
			return value
		}
	}
	return VeteranCommitteeMember
}

// VeteranCommitteeRoleTitleFa عنوان فارسی نقش؛ کلید ناشناخته همان «عضو کمیته» است.
func VeteranCommitteeRoleTitleFa(key string) string {
	normalized := VeteranCommitteeRoleOf(key)
	for _, role := range VeteranCommitteeRoles() {
		if role.Key == normalized {
			return role.TitleFa
		}
	}
	return "عضو کمیته"
}

// VeteranCommitteeRoleOrder ترتیب نشستن امضای نقش در پایین برگه.
func VeteranCommitteeRoleOrder(key string) int {
	normalized := VeteranCommitteeRoleOf(key)
	for _, role := range VeteranCommitteeRoles() {
		if role.Key == normalized {
			return role.Order
		}
	}
	return 90
}

// VeteranCommitteeSignatureFa عنوان یک امضای پایین فرم: «نقش — نام عضو».
// اگر عنوان دلخواهی برای همان عضو ثبت شده باشد، همان مقدم است (مثلاً «نمایندهٔ بنیاد شهید»).
func VeteranCommitteeSignatureFa(role, name, custom string) string {
	label := strings.TrimSpace(custom)
	if label == "" {
		label = VeteranCommitteeRoleTitleFa(role)
	}
	who := strings.TrimSpace(name)
	if who == "" {
		return label
	}
	if label == "" {
		return who
	}
	return label + " — " + who
}

// VeteranCommitteeSignOffFa متن «امضاکنندگان و تاریخ امضا» را از نام امضاکنندگان و تاریخ
// جلسه می‌سازد؛ خالی یعنی هنوز کسی امضا نکرده است.
func VeteranCommitteeSignOffFa(signatures []string, dateFa string) string {
	names := make([]string, 0, len(signatures))
	for _, item := range signatures {
		if value := strings.TrimSpace(item); value != "" {
			names = append(names, value)
		}
	}
	if len(names) == 0 {
		return ""
	}
	text := "امضا شدگان: " + strings.Join(names, "، ")
	if date := strings.TrimSpace(dateFa); date != "" {
		text += " — تاریخ امضا " + date
	}
	return text
}
