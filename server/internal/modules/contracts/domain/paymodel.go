// دستهٔ پرداخت کارمند: گروه جدول ۳ آیین‌نامه و روش پرداخت مصوب.
//
// «دستهٔ پرداخت» = (نوع استخدام/قرارداد) × (گروه نظام پرداخت نفت) × (روش پرداخت). این سه با
// هم تعیین می‌کنند درصدهای جدول ۳ چه باشد و کدام موتور، حکم و فیش را حساب کند:
//
//	گروه ⇒ درصد «حق جذب عمومی + حق مقام» (Y جدول ۳)
//	روش  ⇒ payroll | oil | comp34006 | fourp
//
// قاعده‌ها اینجا زندگی می‌کنند (نه در رابط) و سرور همان‌ها را اعتبارسنجی می‌کند.
package domain

import (
	"strings"

	"hrms/internal/rules"
)

// PayClassification دستهٔ پرداخت یک سند (قرارداد یا حکم).
type PayClassification struct {
	// PayGroup شمارهٔ گروه جدول ۳ (۰ = تعیین نشده).
	PayGroup int `json:"payGroup,omitempty"`
	// PayGroupFa عنوان فارسی گروه پرداخت.
	PayGroupFa string `json:"payGroupFa,omitempty"`
	// PayModel کلید روش پرداخت مصوب.
	PayModel   string `json:"payModel,omitempty"`
	PayModelFa string `json:"payModelFa,omitempty"`
}

// PayGroupOfEmployment گروه پیش‌فرض یک نوع استخدام بدون دانستن پایهٔ سمت.
//
// کارمند رسمی نفت همیشه گروه یکم است؛ برای بقیه، گروه دوم (پایهٔ سمت ۶ تا ۱۰) نقطهٔ شروع
// قرارداد است و حکم کارگزینی با دانستن «گروه شغلی» گروه دقیق را می‌نویسد.
func PayGroupOfEmployment(employmentType string) int {
	if rules.IsOfficialOilEmployment(employmentType) {
		return 1
	}
	if group, ok := rules.OilPayGroupByNumber(2); ok {
		return group.Number
	}
	return 2
}

// PayClassificationOf دستهٔ پرداخت نهایی را می‌سازد: گروه از ورودی (اگر داده شده) و در
// نبود آن از نوع استخدام، و روش پرداخت از ورودی (اگر معتبر باشد) و در نبود آن پیش‌فرض
// همان نوع استخدام. کارمند رسمی نفت هرگز از گروه یکم جدا نمی‌شود.
func PayClassificationOf(employmentType string, payGroup int, payModel string) PayClassification {
	group := payGroup
	if group <= 0 {
		group = PayGroupOfEmployment(employmentType)
	}
	if rules.IsOfficialOilEmployment(employmentType) {
		group = 1
	}
	model := rules.PayModelOf(payModel)
	if strings.TrimSpace(payModel) == "" {
		model = rules.PayModelDefaultFor(employmentType)
	}
	return PayClassification{
		PayGroup:   group,
		PayGroupFa: rules.OilPayGroupTitleFa(group),
		PayModel:   model,
		PayModelFa: rules.PayModelTitleFa(model),
	}
}

// ValidatePayClassification سازگاری نوع استخدام با روش پرداخت و گروه را می‌سنجد.
//
// خروجی، خطاهای فیلدبه‌فیلد است تا رابط همان‌ها را کنار فیلدها نشان دهد.
func ValidatePayClassification(employmentType string, payGroup int, payModel string) []PayFieldError {
	var errs []PayFieldError

	if raw := strings.TrimSpace(payModel); raw != "" {
		if !rules.ValidPayModel(raw) {
			errs = append(errs, PayFieldError{Field: "payModel",
				Message: "روش پرداخت انتخابی در فهرست روش‌های مصوب نیست."})
		} else if ok, reason := rules.PayModelAllowed(employmentType, raw); !ok {
			errs = append(errs, PayFieldError{Field: "payModel", Message: reason})
		}
	}

	if payGroup > 0 {
		if _, ok := rules.OilPayGroupByNumber(payGroup); !ok {
			errs = append(errs, PayFieldError{Field: "payGroup",
				Message: "گروه نظام پرداخت انتخابی معتبر نیست."})
		} else if rules.IsOfficialOilEmployment(employmentType) && payGroup != 1 {
			errs = append(errs, PayFieldError{Field: "payGroup",
				Message: "کارمند رسمی نفت همیشه در «گروه یکم» است؛ گروه انتخابی با نوع استخدام سازگار نیست."})
		}
	}

	return errs
}

// PayModelOptions گزینه‌های روش پرداخت برای فرم‌ها (همان چهار روش مصوب).
func PayModelOptions() []rules.PayModelChoice { return rules.PayModelChoices() }

// PayGroupOptions گزینه‌های گروه جدول ۳ برای فرم‌ها.
func PayGroupOptions() []rules.OilPayGroup { return rules.OilPayGroups() }
