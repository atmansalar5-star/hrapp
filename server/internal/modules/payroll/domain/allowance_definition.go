// نوع رکورد «تعریف فوق العاده» — گام یکم کار: تعریف قلم پیش از تخصیص.
//
// چرا جدا از تخصیص: تعریف، سند مرجع است (کد، نوع قلم، روش محاسبه، متغیرها، مشمولیت کسور و
// جای نمایش در فیش و گزارش) و تخصیص فقط می‌گوید همین تعریف به کدام گروه استخدامی یا کدام
// کارمند، در چه پنجرهٔ زمانی، داده شود. با این تفکیک، تغییر نرخ یا فرمول در یک جا انجام
// می‌گیرد و همهٔ تخصیص‌های وابسته همان را می‌بینند؛ بازنویسی مقدار/مبنا هم برای یک گروه
// خاص ممکن است.
//
// قاعده‌ها همه در بستهٔ `rules` است (`rules/allowance_assignment.go`).
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// KindAllowanceDefinition کلید نوع رکورد «تعریف فوق العاده».
const KindAllowanceDefinition = "allowanceDefinition"

// allowanceDefinitionKind تعریف رکورد فوق العاده برای فرم‌ساز سراسری.
func allowanceDefinitionKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindAllowanceDefinition, TitleFa: "تعریف فوق العادهٔ حقوق", IDPrefix: "ALD",
		EmployeeScoped: false,
		Fields: recordkit.Fields(
			"code|کد فوق العاده (جدول رسمی یا کد داخلی)|text|1",
			"titleFa|عنوان قلم (خالی = عنوان رسمی همان کد)|text|0",
			"category|نوع قلم (earning مزایا / deduction کسور)|select|0",
			"calc|روش محاسبه (fixed ماهانه ثابت / once یکبار مرتبه / percent درصدی / coefficient ضریبی / formula فرمولی)|select|1",
			"rate|مقدار، درصد یا ضریب پیش‌فرض|text|0",
			"basis|مبنای محاسبه (base/allowanceBase/attractionBase/gross/insuranceBase/constant)|select|0",
			"formula|فرمول محاسبه|text|0",
			// متغیرهای تعریف: پایه‌های ثابت محاسبه که در فرمول/شرط به کار می‌روند و در
			// تخصیص‌ها به ارث می‌رسند (مثل CHILDREN=2 یا RATE=0.15).
			"vars|متغیرهای تعریف (نام=مقدار، جدا‌شده با کاما)|text|0",
			"taxable|مشمول مالیات (درج در گزارش مالیات) (بله/خیر)|select|0",
			"insured|مشمول بیمه و بازنشستگی (درج در گزارش بیمه) (بله/خیر)|select|0",
			"showOnPayslip|نمایش در فیش حقوقی (بله/خیر)|select|0",
			"showInReports|درج در گزارش‌های چاپی (بله/خیر)|select|0",
			"reportCode|کد گزارش (کدال) — حرف و رقم|text|0",
			"noteFa|یادداشت (مصوبه، شمارهٔ بخشنامه)|text|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال", "closed|پایان‌یافته"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|active", "active|closed", "closed|active"),
		Actions: recordkit.Actions(
			"activate|فعال‌سازی تعریف|primary|active|0",
			"close|پایان تعریف|ghost|closed|1",
		),
		Validate: validateAllowanceDefinition,
	}
}

// validateAllowanceDefinition شکل دادهٔ تعریف را می‌سنجد و عنوان/نوع قلم/متغیرها را از
// کد و متن متغیرها پر می‌کند تا فرم و گزارش یک تصویر ببینند.
func validateAllowanceDefinition(in contracts.RecordInput) error {
	f := in.Fields

	// پرکردن خودکار پیش از اعتبارسنجی: عنوان، نوع قلم و متغیرها.
	def := rules.NormalizeAllowanceDefinition(rules.AllowanceDefinition{
		Code:          f["code"],
		TitleFa:       f["titleFa"],
		Category:      f["category"],
		Calc:          f["calc"],
		Rate:          f["rate"],
		Basis:         f["basis"],
		Formula:       f["formula"],
		VarsText:      f["vars"],
		Taxable:       isYesOrEmpty(f["taxable"]),
		Insured:       isYes(f["insured"]),
		ShowOnPayslip: isYesOrEmpty(f["showOnPayslip"]),
		ShowInReports: isYesOrEmpty(f["showInReports"]),
		ReportCode:    f["reportCode"],
		NoteFa:        f["noteFa"],
	})
	f["titleFa"] = def.TitleFa
	f["category"] = def.Category
	f["vars"] = def.VarsText
	if strings.TrimSpace(f["showOnPayslip"]) == "" {
		f["showOnPayslip"] = "بله"
	}
	if strings.TrimSpace(f["showInReports"]) == "" {
		f["showInReports"] = "بله"
	}

	if errs := rules.ValidateAllowanceDefinition(def); len(errs) > 0 {
		details := map[string]string{}
		for field, message := range errs {
			details["fields."+field] = message
		}
		return &recordkit.ValidationError{Details: details}
	}
	return nil
}

// allowanceDefinitionSeeds نمونهٔ تعریف فوق العاده برای نصب تازه: یک مبلغ ثابت و یک درصدی
// با متغیر، تا صفحهٔ دو مرحله‌ای (تعریف ⟵ تخصیص) در نصب خالی هم شکل کار را نشان دهد.
func allowanceDefinitionSeeds() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: KindAllowanceDefinition, Title: "تعریف مسکن — مبلغ ثابت مصوب",
			NoteFa: "تعریف پایهٔ کمک هزینهٔ مسکن؛ تخصیص‌ها به همین تعریف وصل می‌شوند.",
			Fields: map[string]string{
				"code": "151", "category": rules.AllowanceCategoryEarning,
				"calc": rules.AllowanceCalcFixed, "rate": "9000000",
				"taxable": "خیر", "insured": "خیر",
				"showOnPayslip": "بله", "showInReports": "بله",
				"reportCode": "151-R", "noteFa": "بخشنامهٔ رفاهی سال",
			},
			SeedStatus: "active"},
		{Kind: KindAllowanceDefinition, Title: "تعریف فوق العادهٔ کارگاهی — درصدی",
			NoteFa: "درصدی از مبنای فوق العادهها؛ تخصیص آن به گروه استخدامی داده می‌شود.",
			Fields: map[string]string{
				"code": "122", "category": rules.AllowanceCategoryEarning,
				"calc": rules.AllowanceCalcPercent, "rate": "10", "basis": "allowanceBase",
				"vars":    "RATE=10,SITE=1",
				"taxable": "بله", "insured": "بله",
				"showOnPayslip": "بله", "showInReports": "بله",
				"noteFa": "ردیف ۵ آیین‌نامه — شرایط محیط کار",
			},
			SeedStatus: "active"},
		{Kind: KindAllowanceDefinition, Title: "تعریف کسر اقساط وام اضطراری",
			NoteFa: "تعریف کسور ماهانهٔ قسط وام؛ مبلغ از قرارداد وام می‌آید و اینجا فقط تعریف است.",
			Fields: map[string]string{
				"code": "5400", "category": rules.AllowanceCategoryDeduction,
				"calc": rules.AllowanceCalcFixed, "rate": "2500000",
				"taxable": "خیر", "insured": "خیر",
				"showOnPayslip": "بله", "showInReports": "بله",
				"reportCode": "5400-R", "noteFa": "قرارداد وام اضطراری",
			},
			SeedStatus: "active"},
	}
}
