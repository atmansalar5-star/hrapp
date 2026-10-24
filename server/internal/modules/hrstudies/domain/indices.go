// فایل قواعد دامنهٔ شاخص‌های سرمایهٔ انسانی و تحلیل نظرسنجی ماژول مطالعات.
//
// چرا این قواعد در بستهٔ دامنه‌اند و نه در لایهٔ API یا رابط: «چه اندازه انحراف از هدف
// قابل قبول است» و «چه نرخ مشارکتی خوب شمرده می‌شود» قاعدهٔ کسب‌وکارند. اگر در UI
// می‌نشستند، هر کلاینت (وب، موبایل، گزارش) عدد خودش را داشت. اینجا یک بار اعلام
// می‌شوند و همهٔ مصرف‌کننده‌ها فقط برچسب فارسی و درصد را از سرور می‌خوانند.
package domain

import (
	"math"
	"strings"
)

/* --------------------------- آستانه‌های سنجش شاخص --------------------------- */

// آستانه‌های سنجش شاخص با هدف (بر حسب درصد انحراف از هدف).
//
// چرا درصد و نه مقدار مطلق: هدف شاخص‌ها یکی «۵٪ نرخ ترک خدمت» است و دیگری «۶۲۰ میلیون
// ریال هزینه»؛ تنها مقیاسی که هر دو را با یک قاعده می‌سنجد، درصد انحراف است.
const (
	// IndicatorToleranceInTarget تا این اندازه انحراف، «در محدودهٔ هدف» شمرده می‌شود.
	IndicatorToleranceInTarget = 5.0
	// IndicatorToleranceNearTarget تا این اندازه انحراف، «نزدیک هدف» است؛ بیش از آن
	// «خارج از هدف» اعلام می‌شود.
	IndicatorToleranceNearTarget = 15.0

	// آستانه‌های کیفی نرخ مشارکت نظرسنجی (درصد).
	// SurveyParticipationGood مرز «مشارکت خوب» و SurveyParticipationAcceptable مرز
	// «قابل قبول»؛ پایین‌تر از آن، نتیجهٔ نظرسنجی قابل تعمیم نیست.
	SurveyParticipationGood       = 75.0
	SurveyParticipationAcceptable = 50.0
)

/* ------------------------------ جهت مطلوب شاخص ------------------------------ */

// Direction جهت مطلوب شاخص: مقدار بیشتر بهتر است یا کمتر.
type Direction string

const (
	// DirectionHigher بیشتر بهتر (مثل ساعت آموزش به ازای هر کارمند).
	DirectionHigher Direction = "higher"
	// DirectionLower کمتر بهتر (مثل نرخ ترک خدمت و هزینهٔ جبران خدمات).
	DirectionLower Direction = "lower"
)

// ParseDirection جهت را از ورودی می‌خواند؛ ناشناخته یعنی «بیشتر بهتر».
//
// چرا پیش‌فرض Higher: بیشتر شاخص‌های سرمایهٔ انسانی (پوشش آموزش، نرخ تصدی، شاخص
// رضایت) از جنس «بیشتر بهتر»اند؛ جاهایی که برعکس است، صریح اعلام می‌شود.
func ParseDirection(raw string) Direction {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "lower", "کمتر", "less", "min", "کمتر‌بهتر":
		return DirectionLower
	default:
		return DirectionHigher
	}
}

// LabelFa برچسب فارسی جهت مطلوب.
func (d Direction) LabelFa() string {
	if d == DirectionLower {
		return "کمتر بهتر"
	}
	return "بیشتر بهتر"
}

// ValidIndicatorDirection می‌گوید مقدار ورودی یک جهت مطلوب مجاز است یا نه.
//
// چرا اعتبارسنجی لازم است: بدون آن، مقدار ناشناخته در ذخیره‌سازی می‌ماند و در محاسبه
// بی‌صدا به «بیشتر بهتر» تفسیر می‌شود؛ یعنی شاخص «کمتر بهتر» نتیجهٔ وارونه می‌داد.
func ValidIndicatorDirection(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(DirectionHigher), string(DirectionLower):
		return true
	default:
		return false
	}
}

/* ------------------------------ سنجش با هدف ------------------------------ */

// Assessment وضعیت سنجش یک شاخص با هدف اعلامی.
type Assessment string

const (
	// AssessmentInTarget در محدودهٔ هدف.
	AssessmentInTarget Assessment = "in_target"
	// AssessmentNearTarget نزدیک هدف ولی خارج از محدودهٔ قابل قبول.
	AssessmentNearTarget Assessment = "near_target"
	// AssessmentOutOfTarget خارج از هدف.
	AssessmentOutOfTarget Assessment = "out_of_target"
)

// LabelFa برچسب فارسی وضعیت سنجش.
func (a Assessment) LabelFa() string {
	switch a {
	case AssessmentInTarget:
		return "در محدودهٔ هدف"
	case AssessmentNearTarget:
		return "نزدیک هدف"
	default:
		return "خارج از هدف"
	}
}

// EvaluateIndicator مقدار جاری را با هدف می‌سنجد و انحراف و وضعیت را برمی‌گرداند.
//
// GapPct در «جهت نامطلوب» حساب می‌شود: مثبت یعنی بدتر از هدف. پس در شاخص «کمتر بهتر»
// (نرخ ترک خدمت) وقتی مقدار جاری از هدف بالاتر برود، گپ مثبت می‌شود و در شاخص «بیشتر
// بهتر» (ساعت آموزش) وقتی کمتر از هدف شود. AchievedPct نسبت جاری به هدف است.
//
// وقتی هدف صفر باشد نسبت‌گیری بی‌معناست؛ در آن حالت فقط برابری با صفر «در هدف»
// شمرده می‌شود و هر مقدار دیگر «خارج از هدف».
func EvaluateIndicator(current, target float64, dir Direction) (achievedPct, gapPct float64, verdict Assessment) {
	if target == 0 {
		if current == 0 {
			return 100, 0, AssessmentInTarget
		}
		return 0, 100, AssessmentOutOfTarget
	}
	base := math.Abs(target)
	achievedPct = round1(current / base * 100)
	var gap float64
	if dir == DirectionLower {
		gap = (current - target) / base * 100
	} else {
		gap = (target - current) / base * 100
	}
	gapPct = round1(gap)
	switch {
	case gap <= IndicatorToleranceInTarget:
		// شامل «رسیدن یا عبور از هدف» (گپ منفی) و انحراف کوچک.
		verdict = AssessmentInTarget
	case gap <= IndicatorToleranceNearTarget:
		verdict = AssessmentNearTarget
	default:
		verdict = AssessmentOutOfTarget
	}
	return achievedPct, gapPct, verdict
}

/* --------------------------- تحلیل مشارکت نظرسنجی --------------------------- */

// ParticipationRate نرخ مشارکت نظرسنجی به درصد (صفر اگر جامعهٔ آماری نامعتبر باشد).
// مقدار بیش از ۱۰۰ درصد به ۱۰۰ محدود می‌شود تا خطای دادهٔ ورودی، تحلیل را خراب نکند.
func ParticipationRate(participants, population int) float64 {
	if population <= 0 {
		return 0
	}
	if participants < 0 {
		participants = 0
	}
	rate := float64(participants) / float64(population) * 100
	if rate > 100 {
		rate = 100
	}
	return round1(rate)
}

// ParticipationBand دستهٔ کیفی نرخ مشارکت را برمی‌گرداند: کلید و برچسب فارسی.
func ParticipationBand(rate float64) (key, labelFa string) {
	switch {
	case rate >= SurveyParticipationGood:
		return "good", "مشارکت خوب"
	case rate >= SurveyParticipationAcceptable:
		return "acceptable", "قابل قبول"
	default:
		return "weak", "مشارکت ضعیف"
	}
}

// round1 به یک رقم اعشار گرد می‌کند تا درصدها در رابط و API یک شکل باشند.
func round1(v float64) float64 { return math.Round(v*10) / 10 }
