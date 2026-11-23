// نوع رکورد «تخصیص فوق العاده» — پیوند یک کد رسمی به یک کارمند یا گروهی از کارکنان.
//
// چرا رکورد و نه فیلد روی پروندهٔ کارمند: تخصیص، سند سازمانی است با تاریخ اعتبار، روش
// محاسبه، مشمولیت کسور و دامنهٔ اثر؛ با تاریخ اعتبار میتوان تغییر مصوبه را در آینده ثبت
// کرد و سابقهٔ تخصیص‌های پیشین را نگه داشت. قاعده‌های محاسبه و تطبیق دامنه در بستهٔ
// `rules` است (`rules/allowance_assignment.go`) و اینجا فقط شکل داده و اعتبارسنجی است.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// KindAllowanceAssignment کلید نوع رکورد «تخصیص فوق العاده».
const KindAllowanceAssignment = "allowanceAssignment"

// allowanceAssignmentKind تعریف رکورد تخصیص فوق العاده برای فرم‌ساز سراسری.
func allowanceAssignmentKind() recordkit.Kind {
	return recordkit.Kind{
		// پیشوند ALW (نه ASG) چون ASG پیش‌تر برای «انتصاب پست سازمانی» در ماژول ساختار
		// سازمانی گرفته شده است و جدول رکوردها مشترک است (دروازهٔ یکتایی پیشوند).
		Key: KindAllowanceAssignment, TitleFa: "تخصیص فوق العادهٔ حقوق", IDPrefix: "ALW",
		EmployeeScoped: false,
		Fields: recordkit.Fields(
			"code|کد فوق العاده (جدول رسمی)|select|1",
			"titleFa|عنوان قلم (خالی = عنوان رسمی همان کد)|text|0",
			"category|نوع قلم (earning مزایا / deduction کسور)|select|0",
			"calc|روش محاسبه (fixed ماهانه ثابت / once یک‌بار مرتبه / percent درصدی / coefficient ضریبی / formula فرمولی)|select|1",
			"rate|مقدار، درصد یا ضریب|text|0",
			"basis|مبنای محاسبه (base/allowanceBase/attractionBase/gross/insuranceBase/constant)|select|0",
			"formula|فرمول محاسبه|text|0",
			"taxable|مشمول مالیات (درج در گزارش مالیات) (بله/خیر)|select|0",
			"insured|مشمول بیمه و بازنشستگی (درج در گزارش بیمه) (بله/خیر)|select|0",
			// جای نمایش و درج قلم در خروجی‌ها: فیش حقوقی، گزارش‌های چاپی و کد گزارش رسمی
			// (کدال). خالی یعنی «بله» تا رفتار گذشته حفظ شود.
			"showOnPayslip|نمایش در فیش حقوقی (بله/خیر)|select|0",
			"showInReports|درج در گزارش‌های چاپی (بله/خیر)|select|0",
			"reportCode|کد گزارش (کدال) — حرف و رقم|text|0",
			"targetKind|دامنهٔ تخصیص (employee/department/employmentType/jobGrade/payGroup/all)|select|1",
			"targetValue|مقدار دامنه (شناسهٔ کارمند، واحد، نوع استخدام یا گروه)|text|0",
			"effectiveFrom|ماه شروع اعتبار (YYYY/MM)|text|1",
			"effectiveTo|ماه پایان اعتبار (YYYY/MM — خالی یعنی تا اطلاع)|text|0",
			"noteFa|یادداشت (مصوبه، شمارهٔ بخشنامه)|text|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال", "closed|پایان‌یافته"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|active", "active|closed", "closed|active"),
		Actions: recordkit.Actions(
			"activate|فعال‌سازی تخصیص|primary|active|0",
			"close|پایان تخصیص|ghost|closed|1",
		),
		Validate: validateAllowanceAssignment,
	}
}

// validateAllowanceAssignment شکل دادهٔ تخصیص را می‌سنجد.
//
// همهٔ قاعده‌ها از بستهٔ `rules` می‌آید تا فرم و موتور یک تعریف ببینند: ماژول فقط فیلدهای
// رکورد را به ساختار قواعد نگاشت می‌کند و پیام‌های خطا را با نام فیلد برمی‌گرداند.
func validateAllowanceAssignment(in contracts.RecordInput) error {
	f := in.Fields

	// اگر عنوان قلم نگذاشته باشند و کد رسمی باشد، عنوان از جدول رسمی می‌آید.
	if strings.TrimSpace(f["titleFa"]) == "" {
		if code, ok := rules.PayslipCodeByNumber(f["code"]); ok {
			f["titleFa"] = code.TitleFa
		}
	}
	// جای نمایش قلم: خالی یعنی «بله» (هم فیش حقوقی و هم گزارش‌های چاپی) تا رفتار پیش از
	// افزودن این دو فیلد عوض نشود.
	if strings.TrimSpace(f["showOnPayslip"]) == "" {
		f["showOnPayslip"] = "بله"
	}
	if strings.TrimSpace(f["showInReports"]) == "" {
		f["showInReports"] = "بله"
	}
	// نوع قلم هم اگر خالی باشد از جدول رسمی استنتاج می‌شود (کسور/مزایا).
	if strings.TrimSpace(f["category"]) == "" {
		if code, ok := rules.PayslipCodeByNumber(f["code"]); ok {
			if code.Kind == rules.PayslipKindEarning {
				f["category"] = rules.AllowanceCategoryEarning
			} else {
				f["category"] = rules.AllowanceCategoryDeduction
			}
		}
	}

	assignment := rules.AllowanceAssignment{
		Code:          strings.TrimSpace(f["code"]),
		TitleFa:       strings.TrimSpace(f["titleFa"]),
		Category:      strings.TrimSpace(f["category"]),
		Calc:          strings.TrimSpace(f["calc"]),
		Rate:          strings.TrimSpace(f["rate"]),
		Basis:         strings.TrimSpace(f["basis"]),
		Formula:       strings.TrimSpace(f["formula"]),
		Taxable:       isYes(f["taxable"]),
		Insured:       isYes(f["insured"]),
		ShowOnPayslip: isYesOrEmpty(f["showOnPayslip"]),
		ShowInReports: isYesOrEmpty(f["showInReports"]),
		ReportCode:    strings.TrimSpace(f["reportCode"]),
		Target:        rules.AllowanceTarget{Kind: strings.TrimSpace(f["targetKind"]), Value: strings.TrimSpace(f["targetValue"])},
		EffectiveFrom: strings.TrimSpace(f["effectiveFrom"]),
		EffectiveTo:   strings.TrimSpace(f["effectiveTo"]),
		NoteFa:        strings.TrimSpace(f["noteFa"]),
	}

	if errs := rules.ValidateAllowanceAssignment(assignment); len(errs) > 0 {
		details := map[string]string{}
		for field, message := range errs {
			// نام فیلد در پاسخ، مسیر کامل رکورد است تا فرم همان کادر را نشان دهد.
			key := "fields." + strings.TrimPrefix(field, "target.")
			if strings.HasPrefix(field, "target.") {
				if field == "target.kind" {
					key = "fields.targetKind"
				} else {
					key = "fields.targetValue"
				}
			}
			details[key] = message
		}
		return &recordkit.ValidationError{Details: details}
	}
	return nil
}

// allowanceAssignmentSeeds نمونهٔ تخصیص برای نصب تازه: یک قلم گروهی و یک قلم فردی.
//
// چرا نمونه لازم است: صفحهٔ «تخصیص فوق العاده‌های حقوق» باید شکل کار را در نصب خالی هم
// نشان دهد؛ این رکوردها فقط در نصب تازه و وقتی جدول خالی است ساخته می‌شوند.
func allowanceAssignmentSeeds() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: KindAllowanceAssignment, Title: "تخصیص مسکن — همهٔ کارکنان",
			NoteFa: "مبلغ ثابت ماهانهٔ کمک هزینهٔ مسکن برای همهٔ کارکنان",
			Fields: map[string]string{
				"code": "151", "category": rules.AllowanceCategoryEarning,
				"calc": rules.AllowanceCalcFixed, "rate": "9000000",
				"taxable": "خیر", "insured": "خیر",
				"targetKind": rules.AllowanceTargetAll, "targetValue": "",
				"effectiveFrom": "1405/01/01", "noteFa": "بخشنامهٔ رفاهی سال",
			},
			SeedStatus: "active"},
		{Kind: KindAllowanceAssignment, Title: "تخصیص حق اولاد — فرزندان مشمول",
			NoteFa: "درصدی از پایهٔ حقوق برای کارکنان مشمول حق اولاد",
			Fields: map[string]string{
				"code": "181", "category": rules.AllowanceCategoryEarning,
				"calc": rules.AllowanceCalcPercent, "rate": "3", "basis": "base",
				"taxable": "خیر", "insured": "خیر",
				"targetKind": rules.AllowanceTargetEmployment, "targetValue": "official_oil",
				"effectiveFrom": "1405/01/01", "noteFa": "مصوبهٔ حق اولاد",
			},
			SeedStatus: "active"},
		{Kind: KindAllowanceAssignment, Title: "تخصیص کسر وام اضطراری — نمونه",
			NoteFa: "کسر ماهانهٔ قسط وام اضطراری برای یک کارمند",
			Fields: map[string]string{
				"code": "5400", "category": rules.AllowanceCategoryDeduction,
				"calc": rules.AllowanceCalcFixed, "rate": "2500000",
				"taxable": "خیر", "insured": "خیر",
				"targetKind": rules.AllowanceTargetEmployee, "targetValue": "EMP-004",
				"effectiveFrom": "1405/01/01", "noteFa": "قرارداد وام اضطراری",
			},
			SeedStatus: "active"},
	}
}

// isYesOrEmpty مثل isYes است، با این تفاوت که مقدار خالی «بله» شمرده می‌شود (پیش‌فرض
// نمایش در فیش و گزارش‌ها برای تخصیص‌های قدیمی که این دو فیلد را ندارند).
func isYesOrEmpty(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return true
	}
	return isYes(raw)
}

// isYes پاسخ مثبت فارسی/لاتین را می‌سنجد (فیلدهای بله/خیر فرم).
func isYes(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "بله", "yes", "true", "1", "y":
		return true
	default:
		return false
	}
}
