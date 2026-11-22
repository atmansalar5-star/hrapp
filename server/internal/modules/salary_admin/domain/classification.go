// دستهٔ پرداخت حکم کارگزینی: گروه جدول ۳ آیین‌نامهٔ فوق‌العاده‌ها و روش پرداخت مصوب.
//
// حکم، سند مبنای پرداخت است؛ پس باید بگوید «با کدام گروه و کدام موتور» صادر شده است:
//
//	گروه  ⇒ درصد حق جذب عمومی و حق مقام (Y جدول ۳)
//	روش   ⇒ payroll | oil | comp34006 | fourp
//
// گروه از «گروه شغلی» کارمند استنتاج می‌شود (نگاشت پیشنهادی GradeSiteMap در پارامترهای
// نظام نفت، گروه شغلی را به پایهٔ سمت جدول ۳ می‌بندد) و روش پرداخت از انتخاب صریح یا
// پیش‌فرض نوع استخدام.
package domain

import (
	"strings"

	"hrms/internal/rules"
)

// DecreeClassification دستهٔ پرداخت یک حکم.
type DecreeClassification struct {
	PayGroup   int    `json:"payGroup,omitempty"`
	PayGroupFa string `json:"payGroupFa,omitempty"`
	PayModel   string `json:"payModel,omitempty"`
	PayModelFa string `json:"payModelFa,omitempty"`
}

// DecreeClassificationOf دستهٔ پرداخت نهایی حکم را می‌سازد.
//
// ترتیب: گروه صریح ورودی، سپس استنتاج از گروه شغلی (و در نهایت گروه دوم = پایهٔ سمت
// ۶ تا ۱۰)، و کارمند رسمی نفت همیشه گروه یکم. روش پرداخت صریح (اگر معتبر باشد) بر
// پیش‌فرض نوع استخدام مقدم است.
func DecreeClassificationOf(employmentType, jobGroup string, payGroup int, payModel string) DecreeClassification {
	params := rules.DefaultOilPayParams()

	group := payGroup
	if group <= 0 {
		group = rules.OilPayGroupForJobGradeNumber(params, employmentType, jobGroup)
	}
	if group <= 0 {
		if fallback, ok := rules.OilPayGroupByNumber(2); ok {
			group = fallback.Number
		} else {
			group = 2
		}
	}
	if rules.IsOfficialOilEmployment(employmentType) {
		group = 1
	}

	model := strings.TrimSpace(payModel)
	if model == "" || !rules.ValidPayModel(model) {
		model = rules.PayModelDefaultFor(employmentType)
	}

	return DecreeClassification{
		PayGroup:   group,
		PayGroupFa: rules.OilPayGroupTitleFa(group),
		PayModel:   model,
		PayModelFa: rules.PayModelTitleFa(model),
	}
}

// ValidateDecreeClassification سازگاری نوع استخدام با روش پرداخت و گروه را می‌سنجد و
// خطاهای فیلدبه‌فیلد برمی‌گرداند (پیام‌ها فارسی و آمادهٔ نمایش کنار همان فیلد).
func ValidateDecreeClassification(employmentType string, payGroup int, payModel string) map[string]string {
	errs := map[string]string{}

	if raw := strings.TrimSpace(payModel); raw != "" {
		if !rules.ValidPayModel(raw) {
			errs["payModel"] = "روش پرداخت انتخابی در فهرست روش‌های مصوب نیست."
		} else if ok, reason := rules.PayModelAllowed(employmentType, raw); !ok {
			errs["payModel"] = reason
		}
	}

	if payGroup > 0 {
		if _, ok := rules.OilPayGroupByNumber(payGroup); !ok {
			errs["payGroup"] = "گروه نظام پرداخت انتخابی معتبر نیست."
		} else if rules.IsOfficialOilEmployment(employmentType) && payGroup != 1 {
			errs["payGroup"] = "کارمند رسمی نفت همیشه در «گروه یکم» است؛ گروه انتخابی با نوع استخدام سازگار نیست."
		}
	}

	return errs
}

// PayModelOptions و PayGroupOptions گزینه‌های فهرست انتخابی فرم حکم.
func PayModelOptions() []rules.PayModelChoice { return rules.PayModelChoices() }

// PayGroupOptions گروه‌های جدول ۳.
func PayGroupOptions() []rules.OilPayGroup { return rules.OilPayGroups() }
