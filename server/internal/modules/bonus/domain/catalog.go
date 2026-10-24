// Package domain دامنهٔ ماژول پاداش.
//
// پرداخت پاداش یک فرایند مستقل از حقوق و کارانه است: مصوبهٔ خودش را دارد، به «مدیریت»
// (واحد تولیدی/پشتیبانی) تخصیص می‌یابد و ممکن است به یک نفر چند بار پرداخت شود. این
// بسته سیاست سازمان را اعلام می‌کند (انواع پاداش، مبلغ پیشنهادی هر نفر، روش‌های پرداخت)
// و قواعد را اعتبارسنجی می‌کند؛ هیچ نرخی در لایهٔ HTTP یا UI ساخته نمی‌شود.
package domain

import "strings"

// BonusType یک نوع پاداش با پارامترهای پیش‌فرض آن.
//
// DefaultAmount مبلغ پیشنهادی «هر نفر» است و در فرم تخصیص، مقدار اولیهٔ قابل بازنویسی
// می‌شود. RequiresDecree یعنی پرداخت بدون شمارهٔ مصوبه/حکم مجاز نیست — این قاعدهٔ
// سازمانی است و در دامنه می‌ماند، نه در فرم.
type BonusType struct {
	Key            string
	TitleFa        string
	DefaultAmount  int64
	RequiresDecree bool
	NoteFa         string
}

// bonusTypes فهرست مصوب انواع پاداش کارکنان.
func bonusTypes() []BonusType {
	return []BonusType{
		{Key: "occasional", TitleFa: "پاداش مناسبتی", DefaultAmount: 20_000_000, RequiresDecree: true, NoteFa: "مناسبت‌های تقویمی مانند نوروز و روز کارگر"},
		{Key: "performance", TitleFa: "پاداش عملکرد", DefaultAmount: 35_000_000, RequiresDecree: true, NoteFa: "بر پایهٔ نتیجهٔ ارزیابی عملکرد دوره"},
		{Key: "production", TitleFa: "پاداش افزایش تولید", DefaultAmount: 25_000_000, RequiresDecree: true, NoteFa: "تحقق هدف تولید واحد"},
		{Key: "project", TitleFa: "پاداش پروژه", DefaultAmount: 40_000_000, RequiresDecree: true, NoteFa: "پایان یا نقطهٔ عطف پروژه"},
		{Key: "retention", TitleFa: "پاداش نگهداشت نیرو", DefaultAmount: 15_000_000, RequiresDecree: true, NoteFa: "حفظ نیروی کلیدی در واحدهای بحرانی"},
		{Key: "special", TitleFa: "پاداش ویژه مدیرعامل", DefaultAmount: 60_000_000, RequiresDecree: true, NoteFa: "با تصویب و ابلاغ مستقیم مدیرعامل"},
	}
}

// BonusTypes فهرست انواع پاداش برای رابط و گزارش‌ها.
func BonusTypes() []BonusType { return bonusTypes() }

// BonusTypeByKey یافتن نوع پاداش با کلید (یا عنوان فارسی، برای داده‌های قدیمی).
func BonusTypeByKey(key string) (BonusType, bool) {
	key = strings.TrimSpace(key)
	for _, t := range bonusTypes() {
		if t.Key == key || t.TitleFa == key {
			return t, true
		}
	}
	return BonusType{}, false
}

// BonusTypeTitleFa عنوان فارسی نوع پاداش؛ برای کلید ناشناخته خودِ ورودی برمی‌گردد.
func BonusTypeTitleFa(key string) string {
	if t, ok := BonusTypeByKey(key); ok {
		return t.TitleFa
	}
	return strings.TrimSpace(key)
}

// ValidBonusType بررسی شناخته‌شده بودن نوع پاداش.
func ValidBonusType(key string) bool {
	_, ok := BonusTypeByKey(key)
	return ok
}

// PaymentMethod روش پرداخت پاداش.
type PaymentMethod struct {
	Key     string
	TitleFa string
}

// paymentMethods روش‌های مجاز پرداخت پاداش.
func paymentMethods() []PaymentMethod {
	return []PaymentMethod{
		{Key: "bank_transfer", TitleFa: "واریز به حساب"},
		{Key: "payroll", TitleFa: "پرداخت با فیش حقوقی"},
		{Key: "gift_card", TitleFa: "کارت هدیه / پتروکارت"},
		{Key: "cash", TitleFa: "نقدی"},
	}
}

// PaymentMethods فهرست روش‌های پرداخت.
func PaymentMethods() []PaymentMethod { return paymentMethods() }

// PaymentMethodByKey یافتن روش پرداخت با کلید.
func PaymentMethodByKey(key string) (PaymentMethod, bool) {
	key = strings.TrimSpace(key)
	for _, m := range paymentMethods() {
		if m.Key == key || m.TitleFa == key {
			return m, true
		}
	}
	return PaymentMethod{}, false
}

// PaymentMethodTitleFa عنوان فارسی روش پرداخت.
func PaymentMethodTitleFa(key string) string {
	if m, ok := PaymentMethodByKey(key); ok {
		return m.TitleFa
	}
	return strings.TrimSpace(key)
}

// ValidPaymentMethod بررسی مجاز بودن روش پرداخت (خالی مجاز است: اختیاری).
func ValidPaymentMethod(key string) bool {
	if strings.TrimSpace(key) == "" {
		return true
	}
	_, ok := PaymentMethodByKey(key)
	return ok
}
