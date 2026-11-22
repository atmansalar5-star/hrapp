// روش‌های پرداخت مصوب سامانه — «این کارمند با کدام موتور محاسبه و فیش شود؟»
//
// چرا در بستهٔ قواعد و نه در ماژول حقوق: انتخاب روش پرداخت، هم در قرارداد و حکم ثبت
// می‌شود، هم فیش و گزارش‌های آن را تعیین می‌کند و هم پنل‌های جبران خدمات به آن نگاه
// می‌کنند. اگر فهرست در ماژول حقوق می‌ماند، ماژول قراردادها و احکام برای اعتبارسنجی
// باید به داخلی‌های ماژول دیگر نگاه می‌کردند — که مرز ماژول را می‌شکند. پس فهرست اینجا
// یک منبع حقیقت است و ماژول حقوق همان را بازنمایی می‌کند.
//
// چهار روش پرداخت مصوب پیشین، بدون تغییر کلیدها:
//
//	payroll   حقوق و دستمزد جاری (پیش‌فرض سازمان)
//	oil       نظام پرداخت صندوق بازنشستگی نفت
//	comp34006 نظام جبران خدمات دستورالعمل IMS-HR-34006
//	fourp     مدل توسعه‌ای جبران خدمات ۴پی
package rules

import "strings"

// کلیدهای روش پرداخت.
const (
	// PayModelPayroll مدل حقوق و دستمزد جاری (پیش‌فرض سازمان).
	PayModelPayroll = "payroll"
	// PayModelOil نظام پرداخت صندوق بازنشستگی نفت.
	PayModelOil = "oil"
	// PayModelComp34006 نظام جبران خدمات دستورالعمل IMS-HR-34006 (مستقل و اختیاری).
	PayModelComp34006 = "comp34006"
	// PayModelFourP مدل توسعه‌ای و اختیاری جبران خدمات ۴پی.
	PayModelFourP = "fourp"
)

// PayModelChoice یک گزینهٔ روش پرداخت برای فهرست انتخابی فرم و پنل.
type PayModelChoice struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa,omitempty"`
	// Optional یعنی مدل توسعه‌ای/اختیاری است و پیش از انتخاب باید پارامتر شرکت روشن شود.
	Optional bool `json:"optional,omitempty"`
}

// PayModelChoices فهرست چهار روش پرداخت مصوب.
func PayModelChoices() []PayModelChoice {
	return []PayModelChoice{
		{Key: PayModelPayroll, TitleFa: "حقوق و دستمزد جاری (پیش‌فرض سازمان)",
			NoteFa: "محاسبهٔ ماهانه با پردازش حقوق و اقلام پارامتری همین ماژول."},
		{Key: PayModelOil, TitleFa: "نظام پرداخت صندوق بازنشستگی نفت",
			NoteFa: "فیش صندوق نفت با جدول‌های ۱ تا ۳ آیین‌نامهٔ فوق‌العاده‌ها و گروه‌های همان جدول."},
		{Key: PayModelComp34006, TitleFa: "نظام جبران خدمات دستورالعمل ۳۴۰۰۶", Optional: true,
			NoteFa: "مدل مستقل با جدول‌های امتیاز و ضریب ریالی خودش."},
		{Key: PayModelFourP, TitleFa: "مدل توسعه‌ای جبران خدمات ۴پی", Optional: true,
			NoteFa: "مدل مستقل و اختیاری؛ مستقل از مدل جی‌بیست (G20) و بدون هیچ پارامتر مشترک با آن."},
	}
}

// PayModelKeys کلیدهای مجاز روش پرداخت.
func PayModelKeys() []string {
	out := make([]string, 0, len(PayModelChoices()))
	for _, choice := range PayModelChoices() {
		out = append(out, choice.Key)
	}
	return out
}

// PayModelTitleFa عنوان فارسی یک روش پرداخت؛ روش ناشناخته خودِ کلید را برمی‌گرداند.
func PayModelTitleFa(key string) string {
	for _, choice := range PayModelChoices() {
		if choice.Key == key {
			return choice.TitleFa
		}
	}
	return strings.TrimSpace(key)
}

// PayModelOf روش پرداخت معتبر؛ مقدار ناشناخته یعنی روش پیش‌فرض سازمان (حقوق و دستمزد جاری).
func PayModelOf(raw string) string {
	value := strings.TrimSpace(raw)
	for _, choice := range PayModelChoices() {
		if choice.Key == value {
			return choice.Key
		}
	}
	return PayModelPayroll
}

// ValidPayModel آیا کلید روش پرداخت در فهرست مصوب است.
func ValidPayModel(raw string) bool {
	value := strings.TrimSpace(raw)
	for _, choice := range PayModelChoices() {
		if choice.Key == value {
			return true
		}
	}
	return false
}

// PayModelDefaultFor روش پرداخت پیشنهادی برای یک نوع استخدام: کارمند رسمی نفت روی نظام
// صندوق نفت می‌نشیند و بقیه روی حقوق و دستمزد جاری سازمان.
func PayModelDefaultFor(employmentType string) string {
	if IsOfficialOilEmployment(employmentType) {
		return PayModelOil
	}
	return PayModelPayroll
}

// PayModelAllowed بررسی سازگاری نوع استخدام و روش پرداخت.
//
// قاعده‌ها:
//   - کارمند رسمی نفت فقط با نظام صندوق بازنشستگی نفت محاسبه می‌شود؛ روش دیگر برای او
//     یعنی فیش بی‌مبنا.
//   - نظام صندوق نفت برای کارکنان غیررسمی مجاز نیست (مشمول صندوق نیستند).
//   - دو مدل توسعه‌ای (۳۴۰۰۶ و ۴پی) اختیاری‌اند و با انتخاب صریح پذیرفته می‌شوند.
//
// خروجی دوم پیام فارسی دلیل ناسازگاری است (خالی یعنی مجاز).
func PayModelAllowed(employmentType, payModel string) (bool, string) {
	model := PayModelOf(payModel)
	officialOil := IsOfficialOilEmployment(employmentType)
	switch {
	case officialOil && model != PayModelOil:
		return false, "برای «کارمند رسمی نفت» تنها روش پرداخت «نظام پرداخت صندوق بازنشستگی نفت» معتبر است."
	case !officialOil && model == PayModelOil:
		return false, "نظام پرداخت صندوق بازنشستگی نفت تنها برای کارمندان رسمی نفت است."
	default:
		return true, ""
	}
}
