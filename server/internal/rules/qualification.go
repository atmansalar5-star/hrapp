// Package rules — قاعدهٔ «شرایط احراز شغل، شرایط معادل، کمبود تحصیلات و مازاد».
//
// چرا اینجا و نه در رابط: تصمیم «آیا این کارمند شرایط احراز این شغل را دارد؟ اگر ندارد،
// چند سال سابقه یا چند سطح مدرک کم دارد؟ و مازادش چه اعتباری می‌سازد؟» یک قاعدهٔ منابع
// انسانی است، نه یک محاسبهٔ نمایشی. اگر در فرم نوشته شود، فرم وب، اپلیکیشن و فرم محاسباتی
// چاپی سه عدد متفاوت می‌سازند.
//
// همهٔ آستانه‌ها پارامتریک‌اند (پارامتر شرکت) تا هیچ عددی در کد ثابت نماند:
//
//	qual.equivalence.enabled          آیا شرایط معادل پذیرفته می‌شود
//	qual.equivalence.yearsPerLevel    هر یک سطح کمبود مدرک با چند سال سابقه جبران می‌شود
//	qual.equivalence.maxYears         سقف سابقه‌ای که برای جبران پذیرفته می‌شود
//	qual.equivalence.fieldMismatchYears  رشتهٔ ناهمسو چند سال سابقهٔ اضافه می‌خواهد
//	qual.excess.monthsPerDegreeLevel  هر سطح مدرک مازاد چند ماه تخفیف دورهٔ انتظار می‌دهد
//	qual.excess.monthsPerExtraYear    هر سال سابقهٔ مازاد چند ماه تخفیف می‌دهد
//	qual.excess.maxCreditMonths       سقف تخفیف دورهٔ انتظار
package rules

import (
	"strconv"
	"strings"
	"unicode"
)

// DegreeLevel یک پله از نردبان مدرک تحصیلی.
//
// رتبه مبنای مقایسه است: «کمبود تحصیلات» یعنی اختلاف رتبهٔ مدرک لازم و مدرک کارمند.
type DegreeLevel struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	Rank    int    `json:"rank"`
}

// DefaultDegreeLadder نردبان مدرک تحصیلی رایج در نظام اداری ایران.
//
// کلیدها و عنوان‌ها عیناً همان فهرست `contracts.DegreeLevels` است (مقطع‌های پروندهٔ
// تحصیلی بایگانی و مدرک ملاک حقوق). چرا هم‌واژه: شناسنامهٔ شغل شرط مدرک را با همین کلید
// ذخیره می‌کند و پروندهٔ کارمند هم با همان کلید مدرک را نگه می‌دارد؛ اگر دو فهرست واگرا
// شوند، یک مقطع در یک جا «کاردانی» و در جای دیگر ناشناخته می‌شود و سنجش شرایط احراز
// بی‌صدا خطا می‌دهد. یکتایی فهرست در آزمون نگه داشته می‌شود.
func DefaultDegreeLadder() []DegreeLevel {
	return []DegreeLevel{
		{Key: "diploma", TitleFa: "دیپلم", Rank: 1},
		{Key: "associate", TitleFa: "کاردانی (فوق دیپلم)", Rank: 2},
		{Key: "bachelor", TitleFa: "کارشناسی", Rank: 3},
		{Key: "master", TitleFa: "کارشناسی ارشد", Rank: 4},
		{Key: "phd", TitleFa: "دکتری", Rank: 5},
		{Key: "postdoc", TitleFa: "فوق دکتری (پسادکتری)", Rank: 6},
	}
}

// DegreeLevels فهرست مقاطع (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func DegreeLevels() []DegreeLevel {
	ladder := DefaultDegreeLadder()
	out := make([]DegreeLevel, len(ladder))
	copy(out, ladder)
	return out
}

// DegreeLevelByKey مقطع را با کلید پیدا می‌کند.
func DegreeLevelByKey(key string) (DegreeLevel, bool) {
	value := strings.TrimSpace(key)
	for _, d := range DefaultDegreeLadder() {
		if d.Key == value {
			return d, true
		}
	}
	return DegreeLevel{}, false
}

// DegreeRankOf رتبهٔ مقطع از کلید آن؛ صفر یعنی ناشناخته (مدرک ثبت نشده).
func DegreeRankOf(key string) int {
	if d, ok := DegreeLevelByKey(key); ok {
		return d.Rank
	}
	return 0
}

// DegreeTitleFa عنوان فارسی مقطع؛ کلید ناشناخته «ثبت نشده» است (نه یک مقطع ساختگی).
func DegreeTitleFa(key string) string {
	if d, ok := DegreeLevelByKey(key); ok {
		return d.TitleFa
	}
	return "ثبت نشده"
}

// QualificationParams آستانه‌های شرایط احراز که از پارامترهای شرکت می‌آید.
type QualificationParams struct {
	// EquivalenceEnabled آیا «شرایط معادل» پذیرفته می‌شود (سابقه جای مدرک).
	EquivalenceEnabled bool `json:"equivalenceEnabled"`
	// YearsPerDegreeLevel هر یک سطح کمبود مدرک با چند سال سابقهٔ مازاد جبران می‌شود.
	YearsPerDegreeLevel int `json:"yearsPerDegreeLevel"`
	// MaxCompensatedYears سقف سال‌هایی که برای جبران کمبود مدرک پذیرفته می‌شود.
	MaxCompensatedYears int `json:"maxCompensatedYears"`
	// FieldMismatchExtraYears سابقهٔ اضافه‌ای که رشتهٔ ناهمسو می‌طلبد.
	FieldMismatchExtraYears int `json:"fieldMismatchExtraYears"`
	// ExcessCreditMonthsPerLevel تخفیف دورهٔ انتظار برای هر سطح مدرک مازاد (ماه).
	ExcessCreditMonthsPerLevel int `json:"excessCreditMonthsPerLevel"`
	// ExcessCreditMonthsPerYear تخفیف دورهٔ انتظار برای هر سال سابقهٔ مازاد (ماه).
	ExcessCreditMonthsPerYear int `json:"excessCreditMonthsPerYear"`
	// MaxExcessCreditMonths سقف تخفیف دورهٔ انتظار (ماه).
	MaxExcessCreditMonths int `json:"maxExcessCreditMonths"`
}

// DefaultQualificationParams پیش‌فرض‌های اعلام‌شده؛ در استقرار واقعی از پارامتر شرکت
// (نسخه‌دار) بازنویسی می‌شوند و تغییر سیاست نیازی به انتشار نسخهٔ تازه ندارد.
func DefaultQualificationParams() QualificationParams {
	return QualificationParams{
		EquivalenceEnabled:         true,
		YearsPerDegreeLevel:        2,
		MaxCompensatedYears:        8,
		FieldMismatchExtraYears:    2,
		ExcessCreditMonthsPerLevel: 12,
		ExcessCreditMonthsPerYear:  3,
		MaxExcessCreditMonths:      36,
	}
}

// QualificationParamsFromValues پارامترهای شرکت را روی پیش‌فرض‌ها می‌نشاند. کلید غایب
// یعنی «پیش‌فرض بمان»، پس پارامتر ناقص محاسبه را متوقف نمی‌کند.
func QualificationParamsFromValues(values map[string]float64) QualificationParams {
	p := DefaultQualificationParams()
	intOf := func(key string) (int, bool) {
		if v, ok := values[key]; ok && v > 0 {
			return int(v), true
		}
		return 0, false
	}
	if v, ok := intOf("qual.equivalence.yearsPerLevel"); ok {
		p.YearsPerDegreeLevel = v
	}
	if v, ok := intOf("qual.equivalence.maxYears"); ok {
		p.MaxCompensatedYears = v
	}
	if v, ok := intOf("qual.equivalence.fieldMismatchYears"); ok {
		p.FieldMismatchExtraYears = v
	}
	if v, ok := intOf("qual.excess.monthsPerDegreeLevel"); ok {
		p.ExcessCreditMonthsPerLevel = v
	}
	if v, ok := intOf("qual.excess.monthsPerExtraYear"); ok {
		p.ExcessCreditMonthsPerYear = v
	}
	if v, ok := intOf("qual.excess.maxCreditMonths"); ok {
		p.MaxExcessCreditMonths = v
	}
	if v, ok := values["qual.equivalence.enabled"]; ok {
		p.EquivalenceEnabled = v != 0
	}
	return p
}

// JobRequirement شرایط احراز یک شغل (از شناسنامهٔ شغل خوانده می‌شود).
type JobRequirement struct {
	// DegreeLevel مقطع تحصیلی لازم (کلید نردبان مدرک)؛ خالی یعنی بدون شرط مدرک.
	DegreeLevel string `json:"degreeLevel"`
	// Fields رشته‌های تحصیلی مرتبط؛ خالی یعنی بدون شرط رشته.
	Fields []string `json:"fields"`
	// MinExperienceYears کمینهٔ سابقهٔ مرتبط (سال).
	MinExperienceYears int `json:"minExperienceYears"`
	// Skills مهارت‌ها و دوره‌های لازم (فهرست نمایشی؛ سنجش آن در این قاعده نیست).
	Skills []string `json:"skills"`
}

// Candidate مشخصات احراز یک کارمند برای مقایسه با شرایط شغل.
type Candidate struct {
	DegreeLevel string `json:"degreeLevel"`
	Field       string `json:"field"`
	// ExperienceYears سابقهٔ مرتبط کارمند (سال).
	ExperienceYears int `json:"experienceYears"`
}

// عنوان نتیجهٔ سنجش شرایط احراز.
const (
	// QualificationDirect: همهٔ شرط‌ها بی‌واسطه برآورده است.
	QualificationDirect = "direct"
	// QualificationEquivalent: شرط مدرک بی‌واسطه برآورده نیست ولی با شرایط معادل
	// (سابقهٔ مازاد) جبران می‌شود.
	QualificationEquivalent = "equivalent"
	// QualificationShortfall: کمبود تحصیلات جبران‌نشده دارد.
	QualificationShortfall = "shortfall"
)

// VerdictFa عنوان فارسی نتیجهٔ سنجش.
func VerdictFa(verdict string) string {
	switch verdict {
	case QualificationDirect:
		return "برآورده — شرایط احراز کامل است"
	case QualificationEquivalent:
		return "برآورده با شرایط معادل"
	case QualificationShortfall:
		return "برآورده نیست — کمبود شرایط احراز"
	default:
		return verdict
	}
}

// QualificationLine یک سطر از فرم محاسباتی شرایط احراز.
//
// فرم محاسباتی باید «چرا» را نشان دهد، نه فقط «چند»: هر سطر یک گام از دستورالعمل است با
// مقدار محاسبه‌شده و شرح فارسی. همین سطرها در چاپ و در پاسخ سرور یکسان‌اند.
type QualificationLine struct {
	Order    int    `json:"order"`
	Key      string `json:"key"`
	TitleFa  string `json:"titleFa"`
	ValueFa  string `json:"valueFa"`
	DetailFa string `json:"detailFa"`
	// Gap مقدار عددی گام است (برای رنگی‌کردن و جمع‌زدن در فرم)؛ معنایش به کلید بستگی دارد.
	Gap int `json:"gap"`
}

// QualificationResult نتیجهٔ کامل سنجش شرایط احراز یک کارمند برای یک شغل.
type QualificationResult struct {
	Eligible  bool   `json:"eligible"`
	Verdict   string `json:"verdict"`
	VerdictFa string `json:"verdictFa"`

	// RequirementFa و CandidateFa خلاصهٔ دو سمت مقایسه برای سرصفحهٔ فرم.
	RequirementFa string `json:"requirementFa"`
	CandidateFa   string `json:"candidateFa"`

	// DegreeGap تعداد سطح کمبود مدرک (مثبت = کمبود، صفر = برابر یا بیشتر).
	DegreeGap int `json:"degreeGap"`
	// DegreeSurplus تعداد سطح مازاد مدرک.
	DegreeSurplus int `json:"degreeSurplus"`
	// FieldMatched آیا رشتهٔ کارمند با رشته‌های مرتبط شغل می‌خواند.
	FieldMatched bool `json:"fieldMatched"`
	// FieldBounded آیا شغل شرط رشته دارد (اگر نه، FieldMatched بی‌معنا و true است).
	FieldBounded bool `json:"fieldBounded"`
	// ExperienceYears و MinExperienceYears دو سمت شرط سابقه.
	ExperienceYears    int `json:"experienceYears"`
	MinExperienceYears int `json:"minExperienceYears"`
	// ExtraYears سابقهٔ مازاد بر کمینه (منفی نمی‌شود).
	ExtraYears int `json:"extraYears"`

	// RequiredEquivalentYears سابقه‌ای که برای جبران کمبود لازم است.
	RequiredEquivalentYears int `json:"requiredEquivalentYears"`
	// CompensatedLevels تعداد سطح کمبودی که با سابقه جبران شد.
	CompensatedLevels int `json:"compensatedLevels"`
	// RemainingDeficitLevels سطح کمبود جبران‌نشده.
	RemainingDeficitLevels int `json:"remainingDeficitLevels"`
	// EquivalenceApplied آیا نتیجه از راه شرایط معادل به دست آمده است.
	EquivalenceApplied bool `json:"equivalenceApplied"`
	// ExcessCreditMonths تخفیف دورهٔ انتظار ارتقا از مازاد شرایط احراز (ماه).
	ExcessCreditMonths int `json:"excessCreditMonths"`

	Lines         []QualificationLine `json:"lines"`
	BlockingCount int                 `json:"blockingCount"`
	SummaryFa     string              `json:"summaryFa"`
	NoteFa        string              `json:"noteFa"`
}

// EvaluateQualification شرایط احراز یک کارمند را برای یک شغل می‌سنجد و فرم محاسباتی را
// خط‌به‌خط می‌سازد.
//
// قاعده‌ها (همه در همین تابع اعلام شده‌اند):
//
//	۱. مدرک: رتبهٔ مدرک کارمند با مدرک لازم سنجیده می‌شود؛ اختلاف مثبت «کمبود سطح» است.
//	۲. رشته: اگر شغل شرط رشته داشته باشد، رشتهٔ کارمند باید با یکی از رشته‌های مرتبط
//	   بخواند (تطبیق واژه‌ای نسبی)؛ رشتهٔ ناهمسو یک شرط جبران‌شدنی است، نه رد قطعی.
//	۳. سابقه: سابقهٔ کارمند با کمینهٔ سابقهٔ شغل سنجیده می‌شود؛ مازاد آن «سابقهٔ مازاد» است.
//	۴. شرایط معادل: اگر کمبود مدرک یا ناهمسویی رشته باشد، هر سطح کمبود با
//	   «qual.equivalence.yearsPerLevel» سال سابقهٔ مازاد و ناهمسویی رشته با
//	   «qual.equivalence.fieldMismatchYears» سال جبران می‌شود.
//	۵. سقف جبران: اگر سابقهٔ لازم از «qual.equivalence.maxYears» بیشتر شود، جبران
//	   پذیرفته نمی‌شود و نتیجه «کمبود شرایط احراز» است.
//	۶. مازاد: مدرک بالاتر از شرط و سابقهٔ مازاد، تخفیف دورهٔ انتظار ارتقا می‌سازد
//	   (به ماه)، با سقف «qual.excess.maxCreditMonths».
//	۷. مدرک ناشناخته (ثبت‌نشده) کمبود حساب می‌شود ولی صفرِ بی‌معنا نیست: در شرح
//	   نتیجه گفته می‌شود «مدرک ثبت نشده» تا کاربر بداند باید پرونده تکمیل شود.
func EvaluateQualification(c Candidate, r JobRequirement, p QualificationParams) QualificationResult {
	candidateRank := DegreeRankOf(c.DegreeLevel)
	requiredRank := DegreeRankOf(r.DegreeLevel)
	// شغل بدون شرط مدرک: رتبهٔ لازم صفر است و شرط مدرک سنجیده نمی‌شود.
	degreeBounded := requiredRank > 0

	res := QualificationResult{
		RequirementFa:      RequirementFa(r),
		CandidateFa:        CandidateFa(c),
		ExperienceYears:    nonNegative(c.ExperienceYears),
		MinExperienceYears: nonNegative(r.MinExperienceYears),
		Lines:              []QualificationLine{},
	}

	// ---------- گام ۱: مدرک تحصیلی ----------
	gap := 0
	surplus := 0
	if degreeBounded {
		gap = requiredRank - candidateRank
		if gap < 0 {
			surplus = -gap
			gap = 0
		}
	}
	res.DegreeGap = gap
	res.DegreeSurplus = surplus
	degreeTitle := "بدون شرط مدرک"
	degreeDetail := "شناسنامهٔ شغل شرط مدرک تعیین نکرده است."
	if degreeBounded {
		degreeTitle = DegreeTitleFa(r.DegreeLevel)
		switch {
		case candidateRank == 0:
			degreeDetail = "مدرک کارمند در پرونده ثبت نشده است؛ برای سنجش دقیق، سابقهٔ تحصیلی تکمیل شود."
		case gap > 0:
			degreeDetail = "مدرک کارمند «" + DegreeTitleFa(c.DegreeLevel) + "» است؛ " +
				strconv.Itoa(gap) + " سطح پایین‌تر از شرط شغل."
		case surplus > 0:
			degreeDetail = "مدرک کارمند «" + DegreeTitleFa(c.DegreeLevel) + "» است؛ " +
				strconv.Itoa(surplus) + " سطح بالاتر از شرط شغل (مازاد)."
		default:
			degreeDetail = "مدرک کارمند با شرط شغل برابر است."
		}
	}
	res.Lines = append(res.Lines, QualificationLine{
		Order: 1, Key: "degree", TitleFa: "مدرک تحصیلی",
		ValueFa: degreeTitle, DetailFa: degreeDetail, Gap: gap,
	})

	// ---------- گام ۲: رشتهٔ تحصیلی ----------
	fieldBounded := len(r.Fields) > 0
	res.FieldBounded = fieldBounded
	res.FieldMatched = true
	fieldValue := "بدون شرط رشته"
	fieldDetail := "شناسنامهٔ شغل رشتهٔ خاصی را الزام نکرده است."
	if fieldBounded {
		res.FieldMatched = FieldMatches(c.Field, r.Fields)
		fieldValue = strings.Join(r.Fields, " / ")
		switch {
		case strings.TrimSpace(c.Field) == "":
			fieldDetail = "رشتهٔ کارمند ثبت نشده است؛ تطبیق رشته ممکن نیست."
		case res.FieldMatched:
			fieldDetail = "رشتهٔ «" + strings.TrimSpace(c.Field) + "» با رشته‌های مرتبط شغل می‌خواند."
		default:
			fieldDetail = "رشتهٔ «" + strings.TrimSpace(c.Field) + "» در فهرست رشته‌های مرتبط نیست؛ " +
				"با شرایط معادل جبران‌شدنی است."
		}
	}
	fieldGap := 0
	if fieldBounded && !res.FieldMatched {
		fieldGap = 1
	}
	res.Lines = append(res.Lines, QualificationLine{
		Order: 2, Key: "field", TitleFa: "رشتهٔ تحصیلی",
		ValueFa: fieldValue, DetailFa: fieldDetail, Gap: fieldGap,
	})

	// ---------- گام ۳: سابقهٔ مرتبط ----------
	extra := res.ExperienceYears - res.MinExperienceYears
	if extra < 0 {
		extra = 0
	}
	res.ExtraYears = extra
	experienceDetail := "سابقهٔ کارمند با کمینهٔ شرط شغل برابر یا بیشتر است."
	if res.ExperienceYears < res.MinExperienceYears {
		experienceDetail = "سابقهٔ کارمند " + strconv.Itoa(res.ExperienceYears) + " سال و کمینهٔ شرط شغل " +
			strconv.Itoa(res.MinExperienceYears) + " سال است."
	}
	res.Lines = append(res.Lines, QualificationLine{
		Order: 3, Key: "experience", TitleFa: "سابقهٔ مرتبط",
		ValueFa:  strconv.Itoa(res.ExperienceYears) + " سال (کمینه " + strconv.Itoa(res.MinExperienceYears) + " سال)",
		DetailFa: experienceDetail,
		Gap:      res.MinExperienceYears - res.ExperienceYears,
	})

	// ---------- گام‌های ۴ و ۵: شرایط معادل و کمبود ----------
	experienceShort := res.ExperienceYears < res.MinExperienceYears
	needEquivalence := gap > 0 || fieldGap > 0 || experienceShort
	requiredYears := 0
	if needEquivalence {
		requiredYears = gap * p.YearsPerDegreeLevel
		if fieldGap > 0 {
			requiredYears += p.FieldMismatchExtraYears
		}
		if experienceShort {
			// کمبود سابقه خودش با سابقه جبران نمی‌شود؛ باید با خدمت آینده پر شود.
			requiredYears += 0
		}
	}
	res.RequiredEquivalentYears = requiredYears

	equivalenceValue := "لازم نیست"
	equivalenceDetail := "شرط مدرک و رشته بی‌واسطه برآورده است؛ نیازی به شرایط معادل نیست."
	equivalenceGap := 0
	if needEquivalence {
		switch {
		case experienceShort:
			equivalenceValue = "قابل بررسی نیست"
			equivalenceDetail = "کمبود سابقه با سابقه جبران نمی‌شود؛ کارمند باید " +
				strconv.Itoa(res.MinExperienceYears-res.ExperienceYears) + " سال سابقهٔ مرتبط کسب کند."
			equivalenceGap = 1
		case !p.EquivalenceEnabled:
			equivalenceValue = "پذیرفته نمی‌شود"
			equivalenceDetail = "شرایط معادل در پارامترهای شرکت فعال نیست؛ شرط‌های ناقص جبران نمی‌شوند."
			equivalenceGap = 1
		case requiredYears > p.MaxCompensatedYears:
			equivalenceValue = "خارج از سقف"
			equivalenceDetail = "برای جبران " + strconv.Itoa(requiredYears) + " سال سابقه لازم است که از سقف " +
				strconv.Itoa(p.MaxCompensatedYears) + " سال بیشتر است؛ شرایط معادل پذیرفته نمی‌شود."
			equivalenceGap = 1
		case extra >= requiredYears:
			res.EquivalenceApplied = true
			res.CompensatedLevels = gap
			equivalenceValue = "برآورده شد"
			equivalenceDetail = "سابقهٔ مازاد " + strconv.Itoa(extra) + " سال، " + strconv.Itoa(requiredYears) +
				" سال لازم برای جبران را می‌پوشاند."
		default:
			short := requiredYears - extra
			equivalenceValue = "ناقص — " + strconv.Itoa(short) + " سال کم است"
			equivalenceDetail = "سابقهٔ مازاد " + strconv.Itoa(extra) + " سال است و برای جبران " +
				strconv.Itoa(requiredYears) + " سال لازم است."
			equivalenceGap = 1
		}
	}
	res.Lines = append(res.Lines, QualificationLine{
		Order: 4, Key: "equivalence", TitleFa: "شرایط معادل",
		ValueFa:  equivalenceValue,
		DetailFa: equivalenceDetail, Gap: equivalenceGap,
	})

	// ---------- کمبود تحصیلات: چند سطح جبران‌نشده ماند ----------
	remaining := 0
	if gap > 0 && !res.EquivalenceApplied {
		if p.YearsPerDegreeLevel <= 0 {
			remaining = gap
		} else {
			coveredYears := extra
			if fieldGap > 0 {
				coveredYears -= p.FieldMismatchExtraYears
			}
			if coveredYears < 0 {
				coveredYears = 0
			}
			coveredLevels := coveredYears / p.YearsPerDegreeLevel
			remaining = gap - coveredLevels
			if remaining < 0 {
				remaining = 0
			}
		}
	}
	res.RemainingDeficitLevels = remaining
	if remaining > 0 && !res.EquivalenceApplied {
		res.Lines = append(res.Lines, QualificationLine{
			Order: 5, Key: "degreeDeficit", TitleFa: "کمبود تحصیلات جبران‌نشده",
			ValueFa: strconv.Itoa(remaining) + " سطح",
			DetailFa: "برای هر سطح کمبود، " + strconv.Itoa(p.YearsPerDegreeLevel) +
				" سال سابقهٔ مازاد لازم است؛ این کمبود پیش از انتصاب یا ارتقا باید جبران شود.",
			Gap: remaining,
		})
	}

	// ---------- گام ۶: مازاد بر شرایط احراز ----------
	credit := 0
	if surplus > 0 {
		credit += surplus * p.ExcessCreditMonthsPerLevel
	}
	if extra > 0 {
		credit += extra * p.ExcessCreditMonthsPerYear
	}
	if p.MaxExcessCreditMonths > 0 && credit > p.MaxExcessCreditMonths {
		credit = p.MaxExcessCreditMonths
	}
	res.ExcessCreditMonths = credit
	excessDetail := "مازادی ثبت نشده است."
	if credit > 0 {
		excessDetail = "مازاد مدرک و سابقه، " + strconv.Itoa(credit) +
			" ماه تخفیف در دورهٔ انتظار ارتقا می‌سازد (سقف " + strconv.Itoa(p.MaxExcessCreditMonths) + " ماه)."
	}
	res.Lines = append(res.Lines, QualificationLine{
		Order: 6, Key: "excess", TitleFa: "مازاد بر شرایط احراز",
		ValueFa:  strconv.Itoa(credit) + " ماه تخفیف",
		DetailFa: excessDetail,
		Gap:      -credit,
	})

	// ---------- جمع‌بندی ----------
	switch {
	case res.EquivalenceApplied:
		res.Verdict = QualificationEquivalent
		res.Eligible = true
	case gap > 0 || fieldGap > 0 || experienceShort:
		res.Verdict = QualificationShortfall
		res.Eligible = false
	default:
		res.Verdict = QualificationDirect
		res.Eligible = true
	}
	res.VerdictFa = VerdictFa(res.Verdict)
	if !res.Eligible {
		res.BlockingCount = 1
	}
	res.SummaryFa = qualificationSummaryFa(res)
	res.NoteFa = "این فرم محاسباتی بر پایهٔ شناسنامهٔ شغل و شرایط احراز مصوب سازمان و پارامترهای " +
		"شرکت ساخته شده است؛ سابقهٔ مازاد تنها در چارچوب شرایط معادل پذیرفته می‌شود و مازاد " +
		"شرایط، تنها تخفیف دورهٔ انتظار ارتقا می‌سازد."
	res.SummaryFa = persianDigits(res.SummaryFa)
	res.NoteFa = persianDigits(res.NoteFa)
	for i := range res.Lines {
		res.Lines[i].ValueFa = persianDigits(res.Lines[i].ValueFa)
		res.Lines[i].DetailFa = persianDigits(res.Lines[i].DetailFa)
	}
	return res
}

// qualificationSummaryFa جملهٔ جمع‌بندی فرم محاسباتی.
func qualificationSummaryFa(res QualificationResult) string {
	switch res.Verdict {
	case QualificationEquivalent:
		return "شرایط احراز با شرایط معادل برآورده می‌شود — " + strconv.Itoa(res.CompensatedLevels) +
			" سطح کمبود با سابقه جبران شد."
	case QualificationShortfall:
		parts := []string{}
		if res.RemainingDeficitLevels > 0 {
			parts = append(parts, strconv.Itoa(res.RemainingDeficitLevels)+" سطح کمبود مدرک")
		}
		if res.FieldBounded && !res.FieldMatched {
			parts = append(parts, "ناهمسویی رشته")
		}
		if res.ExperienceYears < res.MinExperienceYears {
			parts = append(parts, strconv.Itoa(res.MinExperienceYears-res.ExperienceYears)+" سال کمبود سابقه")
		}
		if len(parts) == 0 {
			parts = append(parts, "شرایط احراز برآورده نیست")
		}
		return "شرایط احراز برآورده نیست — " + strings.Join(parts, " و ") + "."
	default:
		summary := "شرایط احراز بی‌واسطه برآورده است."
		if res.ExcessCreditMonths > 0 {
			summary += " مازاد شرایط، " + strconv.Itoa(res.ExcessCreditMonths) + " ماه تخفیف دورهٔ انتظار می‌سازد."
		}
		return summary
	}
}

// RequirementFa خلاصهٔ فارسی شرایط احراز یک شغل (برای سرصفحهٔ فرم و کارت شغل).
func RequirementFa(r JobRequirement) string {
	parts := []string{}
	if rank := DegreeRankOf(r.DegreeLevel); rank > 0 {
		parts = append(parts, DegreeTitleFa(r.DegreeLevel))
	}
	if len(r.Fields) > 0 {
		parts = append(parts, "رشتهٔ "+strings.Join(r.Fields, " / "))
	}
	if r.MinExperienceYears > 0 {
		parts = append(parts, strconv.Itoa(r.MinExperienceYears)+" سال سابقه")
	}
	if len(parts) == 0 {
		return "بدون شرط احراز اعلام‌شده"
	}
	return persianDigits(strings.Join(parts, "، "))
}

// CandidateFa خلاصهٔ فارسی مشخصات احراز یک کارمند.
func CandidateFa(c Candidate) string {
	parts := []string{DegreeTitleFa(c.DegreeLevel)}
	if field := strings.TrimSpace(c.Field); field != "" {
		parts = append(parts, "رشتهٔ "+field)
	}
	if c.ExperienceYears > 0 {
		parts = append(parts, strconv.Itoa(c.ExperienceYears)+" سال سابقه")
	}
	return persianDigits(strings.Join(parts, "، "))
}

// FieldMatches آیا رشتهٔ کارمند با یکی از رشته‌های مرتبط شغل می‌خواند.
//
// تطبیق واژه‌ای نسبی است: «مدیریت بازرگانی» با «مدیریت» می‌خواند، ولی «حسابداری» با
// «مدیریت» نه. این کار لازم است چون رشته در پروندهٔ کارمند با جزئیات نوشته می‌شود و در
// شناسنامهٔ شغل با کلیت.
func FieldMatches(field string, allowed []string) bool {
	candidate := normalizeField(field)
	if candidate == "" {
		return false
	}
	for _, allow := range allowed {
		target := normalizeField(allow)
		if target == "" {
			continue
		}
		if candidate == target || strings.Contains(candidate, target) || strings.Contains(target, candidate) {
			return true
		}
	}
	return false
}

// normalizeField رشته را برای مقایسه یکدست می‌کند: حروف عربی/فارسی هم‌ارز، نیم‌فاصله و
// فاصله‌های تکراری حذف، و «ی/ک» یکدست می‌شود تا دو نوشتار یک رشته، دو رشته شمرده نشود.
func normalizeField(value string) string {
	replacer := strings.NewReplacer(
		"ي", "ی", "ك", "ک", "ۀ", "ه", "ة", "ه", "\u200c", " ", "\u200f", "", "\u200e", "",
	)
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, replacer.Replace(strings.TrimSpace(value)))
	return strings.Join(strings.Fields(normalized), " ")
}

// PersianDigits ارقام لاتین یک متن را به ارقام فارسی تبدیل می‌کند (برای متن نمایشی).
//
// چرا صادرشده: ماژول‌هایی که سند چاپی می‌سازند (کارت شغل، فرم محاسباتی) باید همان قاعدهٔ
// «عدد در متن کاربر فارسی، در داده لاتین» را رعایت کنند و نسخهٔ خودشان را نسازند.
func PersianDigits(value string) string { return persianDigits(value) }

// PersianDigitsInt یک عدد صحیح را با ارقام فارسی برمی‌گرداند.
func PersianDigitsInt(v int) string { return persianDigits(strconv.Itoa(v)) }

// nonNegative مقدار منفی را صفر می‌کند (سال سابقهٔ منفی معنا ندارد).
func nonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// QualificationInstructionFa متن دستورالعمل «کمبود تحصیلات و مازاد شرایط احراز».
//
// چرا در دامنه: دستورالعمل بخشی از قاعده است، نه متن نمایشی؛ فرم محاسباتی وب، نسخهٔ
// چاپی و اپلیکیشن باید همان گام‌ها را نشان دهند تا کسی گام تازه‌ای از خودش نسازد.
func QualificationInstructionFa() []string {
	return []string{
		"۱. شرایط احراز شغل از «شناسنامهٔ شغل» خوانده می‌شود: مقطع تحصیلی، رشته‌های مرتبط و کمینهٔ سابقه. شناسنامهٔ تأییدنشده مبنای انتصاب نیست.",
		"۲. مدرک کارمند با مقطع لازم سنجیده می‌شود؛ اختلاف، «کمبود سطح» یا «مازاد سطح» است و پایهٔ محاسبهٔ همین فرم است.",
		"۳. اگر شغل شرط رشته داشته باشد، رشتهٔ کارمند با رشته‌های مرتبط تطبیق داده می‌شود؛ رشتهٔ ناهمسو رد قطعی نیست و از راه شرایط معادل جبران می‌شود.",
		"۴. سابقهٔ کارمند با کمینهٔ سابقهٔ شغل سنجیده می‌شود؛ آنچه بیش از کمینه است «سابقهٔ مازاد» شمرده می‌شود.",
		"۵. شرایط معادل: هر یک سطح کمبود مدرک با سابقهٔ مازاد جبران می‌شود؛ ضریب تبدیل و سقف آن پارامتر شرکت است و در همین فرم نمایش داده می‌شود.",
		"۶. رشتهٔ ناهمسو نیز با سابقهٔ اضافه جبران می‌شود؛ مقدار آن پارامتر شرکت است.",
		"۷. اگر سابقهٔ لازم برای جبران از سقف پارامتر بیشتر شود، شرایط معادل پذیرفته نمی‌شود و پرونده «کمبود شرایط احراز» ثبت می‌شود.",
		"۸. کمبود سابقه با سابقه جبران نمی‌شود؛ کارمند باید سابقهٔ مرتبط را کسب کند و پس از آن پرونده دوباره سنجیده شود.",
		"۹. مازاد مدرک و سابقه، تخفیف دورهٔ انتظار ارتقا می‌سازد (به ماه، با سقف پارامتر) و مبنای ترفیع زودتر است؛ مازاد به‌تنهایی مجوز ارتقا نیست.",
		"۱۰. خروجی این فرم، مبنای تصویب‌خواهی انتصاب و ارتقا است: نتیجهٔ «برآورده» یا «برآورده با شرایط معادل» شرط لازم است و تصمیم نهایی با سطوح تصویب سازمان است.",
		"۱۱. هر تغییر در پارامترهای شرکت یا شناسنامهٔ شغل، محاسبهٔ پیشین را بی‌اعتبار می‌کند؛ فرم باید با نسخهٔ جاری دوباره اجرا شود.",
	}
}
