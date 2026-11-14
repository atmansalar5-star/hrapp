// مدل جی‌بیست (G20) در لایهٔ دامنه: بیست شاخص در چهار خانواده و امتیاز هر کارمند.
//
// چرا رکورد و نه ثابتِ کد: فهرست شاخص‌ها و وزن هر کدام «سیاست شرکت» است. یک بار به‌صورت
// «پیش‌فرض پیشنهادی» بذرکاری می‌شود و بعد در پنل قابل ویرایش، بایگانی و جایگزینی است؛
// کد فقط قاعدهٔ سنجش و متن فارسی را می‌داند.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

const (
	// KindG20Index یک شاخص از بیست شاخص مدل جی‌بیست.
	KindG20Index = "g20Index"
	// KindG20Assessment امتیاز چهار خانوادهٔ جی‌بیست برای یک کارمند در یک دوره.
	KindG20Assessment = "g20Assessment"
)

// G20IndexKind شاخص جی‌بیست: کد، خانواده، عنوان، روش سنجش، منبع داده و وزن درون خانواده.
func G20IndexKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindG20Index, TitleFa: "شاخص جی‌بیست (G20)", IDPrefix: "CG20",
		Fields: recordkit.Fields(
			"indexCode|کد شاخص (مثل P1-1)|text|1",
			"family|خانوادهٔ چهارگانه|select|1",
			"titleFa|عنوان شاخص|text|1",
			"measureFa|روش سنجش|text|0",
			"sourceFa|منبع داده|select|0",
			"weightPercent|وزن درون خانواده (٪)|number|1",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "active|فعال", "archived|بایگانی‌شده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|active", "active|archived", "archived|draft",
		),
		Actions: recordkit.Actions(
			"activate|فعال‌کردن شاخص|primary|active|0",
			"archive|بایگانی شاخص|ghost|archived|1",
		),
		Validate:         validateG20Index,
		EditableStatuses: []string{"draft", "active"},
	}
}

// G20AssessmentKind امتیاز جی‌بیست یک کارمند: چهار امتیاز خانواده که امتیاز کل و طبقه از
// آن‌ها محاسبه می‌شود (محاسبه در موتور قواعد، نه در فرم).
func G20AssessmentKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindG20Assessment, TitleFa: "امتیاز جی‌بیست کارمند", IDPrefix: "CG2A",
		EmployeeScoped: true,
		Fields: recordkit.Fields(
			"period|دورهٔ سنجش|text|1",
			"personScore|امتیاز پرسن (۰ تا ۱۰۰)|number|1",
			"positionScore|امتیاز پوزیشن (۰ تا ۱۰۰)|number|1",
			"performanceScore|امتیاز پرفورمنس (۰ تا ۱۰۰)|number|1",
			"partnershipScore|امتیاز پارتنرشیپ (۰ تا ۱۰۰)|number|1",
			"assessorName|ارزیاب|text|0",
			"evidence|مستندات سنجش|text|0",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "confirmed|تأییدشده", "closed|بسته‌شده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|confirmed", "confirmed|closed",
		),
		Actions: recordkit.Actions(
			"confirm|تأیید امتیاز|primary|confirmed|0",
			"close|بستن دوره|ghost|closed|0",
		),
		Validate:         validateG20Assessment,
		EditableStatuses: []string{"draft", "confirmed"},
	}
}

// G20Families گزینه‌های چهار خانوادهٔ مدل برای کمبوباکس.
func G20Families() []ReferenceOption {
	out := make([]ReferenceOption, 0, 4)
	for _, f := range rules.G20Families() {
		out = append(out, ReferenceOption{Key: f.Key, TitleFa: f.TitleFa})
	}
	return out
}

// G20Sources منابع دادهٔ سنجش یک شاخص.
func G20Sources() []ReferenceOption {
	return []ReferenceOption{
		{Key: "competency", TitleFa: "سنجش شایستگی"},
		{Key: "jobclass", TitleFa: "ارزشیابی شغل"},
		{Key: "performance", TitleFa: "ارزیابی عملکرد"},
		{Key: "manager", TitleFa: "نظر مدیر"},
		{Key: "self", TitleFa: "خودارزیابی"},
		{Key: "document", TitleFa: "سند و مدرک"},
		{Key: "engagement", TitleFa: "نظرسنجی و تعهد سازمانی"},
	}
}

// validateG20Index قاعده‌های تعریف شاخص: کد با پیشوند خانواده هم‌خوان باشد، خانواده از
// فهرست مصوب باشد و وزن در بازهٔ معقول.
func validateG20Index(in contracts.RecordInput) error {
	code := strings.TrimSpace(in.Fields["indexCode"])
	if code == "" {
		return &recordkit.RuleError{Message: "کد شاخص لازم است"}
	}
	family := strings.TrimSpace(in.Fields["family"])
	if rules.G20FamilyTitleFa(family) == family && family != "" {
		// عنوان برنگشت یعنی کلید در فهرست چهار خانواده نیست.
		known := false
		for _, f := range rules.G20Families() {
			if f.Key == family {
				known = true
			}
		}
		if !known {
			return &recordkit.RuleError{Message: "خانوادهٔ شاخص باید یکی از پرسن، پوزیشن، پرفورمنس یا پارتنرشیپ باشد"}
		}
	}
	if strings.TrimSpace(in.Fields["titleFa"]) == "" {
		return &recordkit.RuleError{Message: "عنوان شاخص لازم است"}
	}
	weight, err := strconv.ParseFloat(strings.TrimSpace(in.Fields["weightPercent"]), 64)
	if err != nil || weight <= 0 || weight > 100 {
		return &recordkit.RuleError{Message: "وزن شاخص باید عددی بین ۰ و ۱۰۰ باشد"}
	}
	if in.Fields["sourceFa"] != "" {
		known := false
		for _, s := range G20Sources() {
			if s.Key == in.Fields["sourceFa"] {
				known = true
			}
		}
		if !known {
			return &recordkit.RuleError{Message: "منبع دادهٔ شاخص از فهرست مجاز نیست"}
		}
	}
	return nil
}

// validateG20Assessment قاعده‌های امتیاز کارمند: دوره و چهار امتیاز در بازهٔ ۰ تا ۱۰۰.
func validateG20Assessment(in contracts.RecordInput) error {
	if strings.TrimSpace(in.Fields["period"]) == "" {
		return &recordkit.RuleError{Message: "دورهٔ سنجش لازم است"}
	}
	for _, key := range []string{"personScore", "positionScore", "performanceScore", "partnershipScore"} {
		raw := strings.TrimSpace(in.Fields[key])
		if raw == "" {
			return &recordkit.RuleError{Message: "امتیاز هر چهار خانواده لازم است"}
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v < 0 || v > 100 {
			return &recordkit.RuleError{Message: "امتیاز هر خانواده باید عددی بین ۰ و ۱۰۰ باشد"}
		}
	}
	return nil
}

// G20IndexSeeds بیست شاخص پیش‌فرض را به رکورد تبدیل می‌کند (وضعیت فعال).
func G20IndexSeeds() []contracts.RecordInput {
	items := rules.G20DefaultIndices()
	out := make([]contracts.RecordInput, 0, len(items))
	for _, idx := range items {
		out = append(out, contracts.RecordInput{
			Kind:       KindG20Index,
			Title:      idx.Code + " — " + idx.TitleFa,
			NoteFa:     "پیش‌فرض پیشنهادی؛ فهرست رسمی و وزن هر شاخص را سازمان تعیین می‌کند",
			SeedStatus: "active",
			Fields: map[string]string{
				"indexCode":     idx.Code,
				"family":        idx.Family,
				"titleFa":       idx.TitleFa,
				"measureFa":     idx.MeasureFa,
				"sourceFa":      g20SourceForFamily(idx.Family),
				"weightPercent": strconv.FormatFloat(idx.Weight, 'f', -1, 64),
			},
		})
	}
	return out
}

// g20SourceForFamily منبع دادهٔ پیش‌فرض هر خانواده.
func g20SourceForFamily(family string) string {
	switch family {
	case "person":
		return "competency"
	case "position":
		return "jobclass"
	case "performance":
		return "performance"
	case "partnership":
		return "engagement"
	}
	return "manager"
}

// G20AssessmentSeeds چهار امتیاز نمونه (چهار کارمند از چهار خانوادهٔ شغلی) تا صفحهٔ طبقه‌بندی
// از روز اول خالی نباشد و توزیع طبقه‌ها دیده شود.
func G20AssessmentSeeds() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: KindG20Assessment, Title: "امتیاز جی‌بیست — سرپرست بهره‌برداری (۱۴۰۵)", EmployeeID: "EMP-001",
			NoteFa:     "پرسن ۸۲، پوزیشن ۷۸، پرفورمنس ۸۸، پارتنرشیپ ۷۰ → امتیاز کل ۸۰٫۴ و طبقهٔ پیشرو",
			SeedStatus: "confirmed",
			Fields: map[string]string{
				"period": "۱۴۰۵", "personScore": "82", "positionScore": "78",
				"performanceScore": "88", "partnershipScore": "70",
				"assessorName": "مدیر امور کارکنان", "evidence": "ارزیابی شغل، سطح شایستگی و ارزیابی عملکرد ۱۴۰۴",
			}},
		{Kind: KindG20Assessment, Title: "امتیاز جی‌بیست — کارشناس تعمیرات (۱۴۰۵)", EmployeeID: "EMP-004",
			NoteFa:     "پوزیشن بالاتر از پرسن: شغل سنگین‌تر با امتیاز شایستگی متوسط",
			SeedStatus: "confirmed",
			Fields: map[string]string{
				"period": "۱۴۰۵", "personScore": "70", "positionScore": "88",
				"performanceScore": "74", "partnershipScore": "62",
				"assessorName": "مدیر تعمیرات", "evidence": "ارزیابی شغل و عملکرد دوره",
			}},
		{Kind: KindG20Assessment, Title: "امتیاز جی‌بیست — کارشناس منابع انسانی (۱۴۰۵)", EmployeeID: "EMP-008",
			NoteFa:     "پرفورمنس بالا با پارتنرشیپ پایین‌تر: نامزد برنامهٔ انتقال دانش",
			SeedStatus: "confirmed",
			Fields: map[string]string{
				"period": "۱۴۰۵", "personScore": "66", "positionScore": "72",
				"performanceScore": "60", "partnershipScore": "58",
				"assessorName": "مدیر منابع انسانی", "evidence": "ارزیابی عملکرد و نظرسنجی تعهد سازمانی",
			}},
		{Kind: KindG20Assessment, Title: "امتیاز جی‌بیست — کارشناس شبکه (۱۴۰۵)", EmployeeID: "EMP-011",
			NoteFa:     "هر چهار خانواده بالا → طبقهٔ ممتاز (امتیاز کل ۸۷٫۱)",
			SeedStatus: "confirmed",
			Fields: map[string]string{
				"period": "۱۴۰۵", "personScore": "90", "positionScore": "84",
				"performanceScore": "92", "partnershipScore": "80",
				"assessorName": "مدیر فناوری اطلاعات", "evidence": "ارزیابی شایستگی، عملکرد و مشارکت در پروژه‌ها",
			}},
	}
}
