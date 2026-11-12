// نظام جبران خدمات ۴پی — موتور محاسبهٔ سمت سرور.
//
// چرا این فایل جدا از `payroll.go`: پرداخت (محاسبهٔ فیش ماهانه) با «طراحی بستهٔ جبران
// خدمات» یکی نیست. طراحی بسته به چهار عامل بستگی دارد و همان چیزی است که در ادبیات
// مدیریت جبران خدمات «۴پی» خوانده میشود:
//
//	P1 — Position  (شغل)      ارزشیابی و رتبه‌بندی شغل، امتیاز شغل و پلهٔ گروه شغلی
//	P2 — Person    (شخص)      شایستگی و صلاحیت فرد، سطح مهارت و سابقهٔ مرتبط
//	P3 — Performance (عملکرد) نتیجهٔ ارزیابی عملکرد دوره
//	P4 — Pay       (پرداخت)   باند و پله‌های پرداخت، میانهٔ باند، نسبت پرداخت به میانه
//
// قاعدهٔ معماری: هر عددی که «سیاست شرکت» است در پارامترها می‌نشیند
// (`compensation.*`) و در این فایل فقط قاعدهٔ محاسبه و متن فارسی اعلام می‌شود؛ هیچ
// نصاب پولی در رابط کاربری نوشته نمی‌شود.
package rules

import (
	"math"
	"strconv"
	"strings"
)

// CompensationParams پارامترهای طراحی بستهٔ جبران خدمات (کلیدها با پیشوند `compensation.`).
type CompensationParams struct {
	// BandBaseMidpoint میانهٔ باند پلهٔ نخست گروه شغلی (ریال).
	BandBaseMidpoint int64 `json:"bandBaseMidpoint"`
	// BandSlopePercent رشد میانهٔ باند از هر پله به پلهٔ بالاتر (درصد).
	BandSlopePercent float64 `json:"bandSlopePercent"`
	// BandSpreadPercent گسترهٔ باند: کف و سقف هر باند به این نسبت از میانه ساخته می‌شود.
	BandSpreadPercent float64 `json:"bandSpreadPercent"`
	// BandStepPercent گام پله درون باند (درصد از میانه) برای بازنگری‌های خرد.
	BandStepPercent float64 `json:"bandStepPercent"`
	// JobPointsPerGrade امتیاز ارزشیابی شغل که هر پلهٔ گروه شغلی را می‌سازد.
	JobPointsPerGrade float64 `json:"jobPointsPerGrade"`
	// JobMaxGrade بالاترین پلهٔ گروه شغلی که باند پرداخت برایش ساخته می‌شود.
	JobMaxGrade int `json:"jobMaxGrade"`
	// PersonMinFactor و PersonMaxFactor بازهٔ ضریب شایستگی فرد.
	PersonMinFactor float64 `json:"personMinFactor"`
	PersonMaxFactor float64 `json:"personMaxFactor"`
	// PersonLevels شمار سطوح مدل شایستگی (۱ تا این عدد).
	PersonLevels int `json:"personLevels"`
	// Thresholds امتیاز ارزیابی عملکرد که سطح عملکرد را تعیین می‌کند.
	PerfExcellentScore float64 `json:"perfExcellentScore"`
	PerfGoodScore      float64 `json:"perfGoodScore"`
	PerfMeetsScore     float64 `json:"perfMeetsScore"`
	// Factors ضریب هر سطح عملکرد.
	PerfFactorExcellent float64 `json:"perfFactorExcellent"`
	PerfFactorGood      float64 `json:"perfFactorGood"`
	PerfFactorMeets     float64 `json:"perfFactorMeets"`
	PerfFactorBelow     float64 `json:"perfFactorBelow"`
	// MarketAdjustPercent تعدیل سالانهٔ بازار (فقط با درخواست صریح اعمال می‌شود).
	MarketAdjustPercent float64 `json:"marketAdjustPercent"`
	// CompaMinPercent و CompaMaxPercent بازهٔ قابل قبول نسبت پرداخت به میانهٔ باند.
	CompaMinPercent float64 `json:"compaMinPercent"`
	CompaMaxPercent float64 `json:"compaMaxPercent"`
	// MaxAnnualIncreasePercent سقف افزایش سالانهٔ یک کارمند (درصد پرداخت فعلی).
	MaxAnnualIncreasePercent float64 `json:"maxAnnualIncreasePercent"`
	// ReviewMonths دورهٔ بازنگری بسته (ماه).
	ReviewMonths int `json:"reviewMonths"`
	// PayrollApply آیا خروجی این مدل در صدور حکم و محاسبهٔ فیش مصرف شود.
	//
	// پیش‌فرض «خاموش» است و عمدی است: مدل ۴پی یک مدل «توسعه‌ای و اختیاری» است که در
	// کنار مدل‌های پیشین (حقوق و دستمزد جاری و مدل صندوق نفت) و مستقل از آن‌ها زندگی
	// می‌کند. تا وقتی شرکت این پارامتر را روشن نکند، هیچ عددی از این مدل به حکم یا فیش
	// راه نمی‌یابد؛ و حتی با روشن شدنش، احکام صادرشدهٔ پیشین دست‌نخورده می‌مانند.
	PayrollApply bool `json:"payrollApply"`
}

// DefaultCompensationParams پیش‌فرض‌های آغازین؛ در استقرار واقعی از پارامترهای شرکت
// خوانده می‌شوند و بدون انتشار نسخهٔ تازه تغییر می‌کنند.
func DefaultCompensationParams() CompensationParams {
	return CompensationParams{
		BandBaseMidpoint:         200_000_000,
		BandSlopePercent:         18,
		BandSpreadPercent:        40,
		BandStepPercent:          3,
		JobPointsPerGrade:        60,
		JobMaxGrade:              6,
		PersonMinFactor:          0.90,
		PersonMaxFactor:          1.10,
		PersonLevels:             5,
		PerfExcellentScore:       90,
		PerfGoodScore:            80,
		PerfMeetsScore:           65,
		PerfFactorExcellent:      1.15,
		PerfFactorGood:           1.08,
		PerfFactorMeets:          1.00,
		PerfFactorBelow:          0.92,
		MarketAdjustPercent:      10,
		CompaMinPercent:          80,
		CompaMaxPercent:          120,
		MaxAnnualIncreasePercent: 25,
		ReviewMonths:             12,
		// «خاموش» یعنی مدل ۴پی اثری در پرداخت ندارد: مدل توسعه‌ای اختیاری است.
		PayrollApply: false,
	}
}

// CompensationParamsFromValues پارامترهای شرکت (عدد) را روی پیش‌فرض می‌نشاند؛ کلید
// نبوده یا خارج از بازهٔ معقول، پیش‌فرض را دست‌نخورده می‌گذارد.
// CompensationIndependenceFa متن رسمی «استقلال مدل ۴پی» — همان چیزی که در فرم، سند و
// مستندات تکرار می‌شود تا مرز سه مدل پرداخت در سازمان روشن بماند.
//
// چرا در کد و نه فقط در مستند: این جمله یک «قاعدهٔ طراحی» است؛ متن روشن‌بودن پارامتر هم
// می‌گوید از کجا اثر می‌گذارد و احکام پیشین چه وضعی دارند.
func CompensationIndependenceFa(apply bool) string {
	common := "مدل جبران خدمات ۴پی مستقل از مدل حقوق و دستمزد جاری و مدل صندوق نفت است؛ " +
		"باندها، ضرایب و تصمیم‌های آن جداگانه محاسبه و نگهداری می‌شوند."
	if apply {
		return common + " در این استقرار پارامتر «اعمال در پرداخت» روشن است، پس از این پس " +
			"تصمیم مصوب می‌تواند مبنای صدور حکم تازه شود؛ احکام و پرداخت‌های پیشین بر پایهٔ " +
			"مدل پیشین خودشان معتبر می‌مانند و بازنویسی نمی‌شوند."
	}
	return common + " اکنون «اعمال در پرداخت» خاموش است: این مدل توسعه‌ای و اختیاری است و " +
		"هیچ عددی از آن به حکم یا فیش راه نمی‌یابد تا شرکت خودش آن را روشن کند."
}

func CompensationParamsFromValues(values map[string]float64) CompensationParams {
	p := DefaultCompensationParams()
	set := func(key string, apply func(float64)) {
		if v, found := values[key]; found {
			apply(v)
		}
	}
	set("compensation.band.baseMidpoint", func(v float64) {
		if v > 0 {
			p.BandBaseMidpoint = int64(v)
		}
	})
	set("compensation.band.slopePercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.BandSlopePercent = v
		}
	})
	set("compensation.band.spreadPercent", func(v float64) {
		if v >= 0 && v <= 200 {
			p.BandSpreadPercent = v
		}
	})
	set("compensation.band.stepPercent", func(v float64) {
		if v >= 0 && v <= 50 {
			p.BandStepPercent = v
		}
	})
	set("compensation.job.pointsPerGrade", func(v float64) {
		if v > 0 {
			p.JobPointsPerGrade = v
		}
	})
	set("compensation.job.maxGrade", func(v float64) {
		if v >= 1 && v <= 30 {
			p.JobMaxGrade = int(v)
		}
	})
	set("compensation.person.minFactor", func(v float64) {
		if v > 0 && v <= 2 {
			p.PersonMinFactor = v
		}
	})
	set("compensation.person.maxFactor", func(v float64) {
		if v > 0 && v <= 2 {
			p.PersonMaxFactor = v
		}
	})
	set("compensation.person.levels", func(v float64) {
		if v >= 2 && v <= 20 {
			p.PersonLevels = int(v)
		}
	})
	set("compensation.performance.excellentScore", func(v float64) {
		if v > 0 && v <= 100 {
			p.PerfExcellentScore = v
		}
	})
	set("compensation.performance.goodScore", func(v float64) {
		if v > 0 && v <= 100 {
			p.PerfGoodScore = v
		}
	})
	set("compensation.performance.meetsScore", func(v float64) {
		if v >= 0 && v <= 100 {
			p.PerfMeetsScore = v
		}
	})
	set("compensation.performance.factorExcellent", func(v float64) {
		if v > 0 && v <= 2 {
			p.PerfFactorExcellent = v
		}
	})
	set("compensation.performance.factorGood", func(v float64) {
		if v > 0 && v <= 2 {
			p.PerfFactorGood = v
		}
	})
	set("compensation.performance.factorMeets", func(v float64) {
		if v > 0 && v <= 2 {
			p.PerfFactorMeets = v
		}
	})
	set("compensation.performance.factorBelow", func(v float64) {
		if v > 0 && v <= 2 {
			p.PerfFactorBelow = v
		}
	})
	set("compensation.market.adjustPercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.MarketAdjustPercent = v
		}
	})
	set("compensation.compa.minPercent", func(v float64) {
		if v > 0 && v <= 200 {
			p.CompaMinPercent = v
		}
	})
	set("compensation.compa.maxPercent", func(v float64) {
		if v > 0 && v <= 300 {
			p.CompaMaxPercent = v
		}
	})
	set("compensation.increase.maxAnnualPercent", func(v float64) {
		if v >= 0 && v <= 100 {
			p.MaxAnnualIncreasePercent = v
		}
	})
	set("compensation.review.months", func(v float64) {
		if v >= 1 && v <= 120 {
			p.ReviewMonths = int(v)
		}
	})
	if v, ok := values["compensation.payroll.apply"]; ok {
		p.PayrollApply = v != 0
	}
	// سازگاری: کف ضریب نباید از سقف بزرگ‌تر بماند (پارامتر اشتباه، پیش‌فرض را برمی‌گرداند).
	if p.PersonMinFactor > p.PersonMaxFactor {
		p.PersonMinFactor, p.PersonMaxFactor = 0.90, 1.10
	}
	if p.CompaMinPercent > p.CompaMaxPercent {
		p.CompaMinPercent, p.CompaMaxPercent = 80, 120
	}
	return p
}

// CompensationParamsFromText همان نرمال‌سازی متن پارامترهای شرکت (ارقام فارسی، درصد،
// جداکنندهٔ هزارگان) پیش از تبدیل به عدد.
func CompensationParamsFromText(values map[string]string) CompensationParams {
	out := map[string]float64{}
	for key, raw := range values {
		if !strings.HasPrefix(key, "compensation.") {
			continue
		}
		text := strings.TrimSpace(strings.NewReplacer("٪", "", "%", "", ",", "", "،", "", "٬", "", "'", "", " ", "").Replace(raw))
		if text == "" {
			continue
		}
		// ارقام فارسی/عربی به لاتین تا پارامتر نوشته‌شده در پنل هم خوانده شود.
		text = faDigitsToLatin(text)
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			continue
		}
		out[key] = value
	}
	return CompensationParamsFromValues(out)
}

// faDigitsToLatin ارقام فارسی و عربی را به ارقام لاتین برمی‌گرداند.
func faDigitsToLatin(text string) string {
	var b strings.Builder
	for _, r := range text {
		switch {
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// CompensationBand باند پرداخت یک پلهٔ گروه شغلی.
type CompensationBand struct {
	Rank     int    `json:"rank"`
	GradeFa  string `json:"gradeFa"`
	Min      int64  `json:"min"`
	Midpoint int64  `json:"midpoint"`
	Max      int64  `json:"max"`
	// Step مبلغ یک گام پله درون باند.
	Step int64 `json:"step"`
}

// CompensationBandFor باند پرداخت یک پلهٔ گروه شغلی.
//
// میانهٔ باند به‌صورت مرکب رشد می‌کند (پلهٔ بعد = میانه × (۱ + شیب)) و کف/سقف به
// نسبت گستره از میانه ساخته می‌شود؛ پس تغییر یک پارامتر همهٔ باندها را هم‌زمان و
// سازگار جابه‌جا می‌کند.
func CompensationBandFor(rank int, p CompensationParams) CompensationBand {
	if rank < 1 {
		rank = 1
	}
	if p.JobMaxGrade > 0 && rank > p.JobMaxGrade {
		rank = p.JobMaxGrade
	}
	mid := float64(p.BandBaseMidpoint) * math.Pow(1+p.BandSlopePercent/100, float64(rank-1))
	half := p.BandSpreadPercent / 200
	return CompensationBand{
		Rank:     rank,
		GradeFa:  GradeTitleFa(rank),
		Min:      RoundHalfUp(mid / (1 + half)),
		Midpoint: RoundHalfUp(mid),
		Max:      RoundHalfUp(mid * (1 + half)),
		Step:     RoundHalfUp(mid * p.BandStepPercent / 100),
	}
}

// CompensationBands باندهای همهٔ پله‌های گروه شغلی از ۱ تا سقف پارامتر.
func CompensationBands(p CompensationParams) []CompensationBand {
	max := p.JobMaxGrade
	if max < 1 {
		max = 1
	}
	out := make([]CompensationBand, 0, max)
	for rank := 1; rank <= max; rank++ {
		out = append(out, CompensationBandFor(rank, p))
	}
	return out
}

// CompensationGradeFromScore پلهٔ گروه شغلی از امتیاز ارزشیابی شغل (P1).
func CompensationGradeFromScore(score float64, p CompensationParams) int {
	per := p.JobPointsPerGrade
	if per <= 0 {
		per = 60
	}
	if score < 0 {
		score = 0
	}
	rank := 1 + int(score/per)
	if p.JobMaxGrade > 0 && rank > p.JobMaxGrade {
		rank = p.JobMaxGrade
	}
	if rank < 1 {
		rank = 1
	}
	return rank
}

// CompensationLevelTitleFa عنوان فارسی سطح شایستگی (P2).
func CompensationLevelTitleFa(level int) string {
	switch level {
	case 1:
		return "مبتدی"
	case 2:
		return "در حال توسعه"
	case 3:
		return "مستقل"
	case 4:
		return "پیشرفته"
	case 5:
		return "مرجع/خبره"
	default:
		if level < 1 {
			return "نامشخص"
		}
		return "سطح " + IntFa(level)
	}
}

// CompensationPersonFactor ضریب شایستگی فرد از سطح او (P2).
func CompensationPersonFactor(level int, p CompensationParams) float64 {
	levels := p.PersonLevels
	if levels < 2 {
		levels = 2
	}
	if level < 1 {
		level = 1
	}
	if level > levels {
		level = levels
	}
	span := p.PersonMaxFactor - p.PersonMinFactor
	factor := p.PersonMinFactor + span*float64(level-1)/float64(levels-1)
	return math.Round(factor*1000) / 1000
}

// CompensationPerformanceLevel عنوان سطح عملکرد و ضریب آن از امتیاز ارزیابی (P3).
type CompensationPerformanceLevel struct {
	Key     string  `json:"key"`
	TitleFa string  `json:"titleFa"`
	Factor  float64 `json:"factor"`
}

// CompensationPerformanceLevelOf سطح عملکرد از امتیاز ارزیابی.
func CompensationPerformanceLevelOf(score float64, p CompensationParams) CompensationPerformanceLevel {
	switch {
	case score >= p.PerfExcellentScore:
		return CompensationPerformanceLevel{Key: "excellent", TitleFa: "عالی", Factor: p.PerfFactorExcellent}
	case score >= p.PerfGoodScore:
		return CompensationPerformanceLevel{Key: "good", TitleFa: "خوب", Factor: p.PerfFactorGood}
	case score >= p.PerfMeetsScore:
		return CompensationPerformanceLevel{Key: "meets", TitleFa: "قابل قبول", Factor: p.PerfFactorMeets}
	default:
		return CompensationPerformanceLevel{Key: "below", TitleFa: "نیازمند بهبود", Factor: p.PerfFactorBelow}
	}
}

// CompensationLine یک سطر محاسبه با شرح فارسی (نمایش در فرم و سند تصمیم).
type CompensationLine struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	ValueFa string `json:"valueFa"`
	RefFa   string `json:"refFa,omitempty"`
}

// CompensationInput ورودی محاسبهٔ بستهٔ جبران خدمات یک کارمند.
type CompensationInput struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	// JobScore امتیاز ارزشیابی شغل (P1).
	JobScore float64 `json:"jobScore"`
	// GradeOverride پلهٔ گروه شغلی دستی؛ صفر یعنی از امتیاز شغل محاسبه شود.
	GradeOverride int `json:"gradeOverride"`
	// CompetencyLevel سطح شایستگی فرد (P2).
	CompetencyLevel int `json:"competencyLevel"`
	// PerformanceScore امتیاز ارزیابی عملکرد دوره (P3).
	PerformanceScore float64 `json:"performanceScore"`
	// CurrentPay پرداخت فعلی کارمند (پایهٔ حقوق + فوقالعاده‌های ثابت ماهانه).
	CurrentPay int64 `json:"currentPay"`
	// MarketAdjust آیا تعدیل بازار در این بازنگری اعمال شود.
	MarketAdjust bool `json:"marketAdjust"`
}

// CompensationResult خروجی محاسبهٔ بستهٔ جبران خدمات.
type CompensationResult struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName,omitempty"`
	// P1 — شغل
	JobScore float64          `json:"jobScore"`
	Rank     int              `json:"rank"`
	GradeFa  string           `json:"gradeFa"`
	Band     CompensationBand `json:"band"`
	// P2 — شخص
	CompetencyLevel int     `json:"competencyLevel"`
	LevelFa         string  `json:"levelFa"`
	PersonFactor    float64 `json:"personFactor"`
	// P3 — عملکرد
	PerformanceScore  float64 `json:"performanceScore"`
	PerformanceFa     string  `json:"performanceFa"`
	PerformanceFactor float64 `json:"performanceFactor"`
	// P4 — پرداخت
	TargetPay     int64   `json:"targetPay"`
	CurrentPay    int64   `json:"currentPay"`
	Increase      int64   `json:"increase"`
	IncreasePct   float64 `json:"increasePercent"`
	CompaRatioPct float64 `json:"compaRatioPercent"`
	BandStateFa   string  `json:"bandStateFa"`
	// CappedByMax و CappedByIncrease نشان می‌دهد سقف باند یا سقف افزایش سالانه اثر کرده است.
	CappedByMax      bool `json:"cappedByMax"`
	CappedByIncrease bool `json:"cappedByIncrease"`
	// NoDecrease پرداخت فعلی از پرداخت هدف بیشتر بوده و کاهش پیشنهاد نشده است.
	NoDecrease    bool               `json:"noDecrease"`
	MarketApplied bool               `json:"marketApplied"`
	Lines         []CompensationLine `json:"lines"`
	SummaryFa     string             `json:"summaryFa"`
	WarningsFa    []string           `json:"warningsFa,omitempty"`
	Valid         bool               `json:"valid"`
	MessageFa     string             `json:"messageFa,omitempty"`
}

// CompensationCalc محاسبهٔ خط‌به‌خط بستهٔ جبران خدمات بر پایهٔ چهار عامل.
//
// قاعده‌ها به ترتیب اعمال می‌شوند:
//
//	۱. پلهٔ گروه شغلی از امتیاز ارزشیابی شغل (یا پلهٔ دستی) و باند همان پله.
//	۲. پرداخت هدف = میانهٔ باند × ضریب شایستگی × ضریب عملکرد (و تعدیل بازار در صورت درخواست).
//	۳. پرداخت هدف هرگز از سقف باند بیشتر نمی‌شود.
//	۴. کاهش پرداخت پیشنهاد نمی‌شود: اگر پرداخت فعلی از هدف بیشتر باشد، افزایش صفر است.
//	۵. افزایش از سقف افزایش سالانهٔ پارامتر شرکت بیشتر نمی‌شود.
//	۶. نسبت پرداخت فعلی به میانهٔ باند بیرون بازهٔ قابل قبول، هشدار بازنگری می‌دهد.
func CompensationCalc(in CompensationInput, p CompensationParams) CompensationResult {
	out := CompensationResult{
		EmployeeID:       strings.TrimSpace(in.EmployeeID),
		EmployeeName:     strings.TrimSpace(in.EmployeeName),
		JobScore:         in.JobScore,
		CurrentPay:       in.CurrentPay,
		CompetencyLevel:  in.CompetencyLevel,
		PerformanceScore: in.PerformanceScore,
	}

	rank := in.GradeOverride
	if rank <= 0 {
		rank = CompensationGradeFromScore(in.JobScore, p)
	}
	if rank < 1 || (p.JobMaxGrade > 0 && rank > p.JobMaxGrade) {
		out.MessageFa = "پلهٔ گروه شغلی بیرون از نردبان گروه‌های سازمان است."
		return out
	}
	out.Rank = rank
	out.GradeFa = GradeTitleFa(rank)
	out.Band = CompensationBandFor(rank, p)

	out.LevelFa = CompensationLevelTitleFa(in.CompetencyLevel)
	out.PersonFactor = CompensationPersonFactor(in.CompetencyLevel, p)
	perf := CompensationPerformanceLevelOf(in.PerformanceScore, p)
	out.PerformanceFa = perf.TitleFa
	out.PerformanceFactor = perf.Factor

	base := float64(out.Band.Midpoint) * out.PersonFactor * out.PerformanceFactor
	if in.MarketAdjust {
		base *= 1 + p.MarketAdjustPercent/100
		out.MarketApplied = true
	}
	target := RoundHalfUp(base)
	if target > out.Band.Max {
		target = out.Band.Max
		out.CappedByMax = true
	}
	if target < out.Band.Min {
		target = out.Band.Min
	}
	out.TargetPay = target

	increase := target - out.CurrentPay
	if increase < 0 {
		increase = 0
		out.NoDecrease = true
	}
	if capAmount := RoundHalfUp(float64(out.CurrentPay) * p.MaxAnnualIncreasePercent / 100); capAmount >= 0 && increase > capAmount {
		increase = capAmount
		out.CappedByIncrease = true
	}
	out.Increase = increase
	if out.CurrentPay > 0 {
		out.IncreasePct = math.Round(float64(increase)/float64(out.CurrentPay)*1000) / 10
		out.CompaRatioPct = math.Round(float64(out.CurrentPay)/float64(out.Band.Midpoint)*1000) / 10
	}
	switch {
	case out.CurrentPay < out.Band.Min:
		out.BandStateFa = "زیر کف باند"
	case out.CurrentPay > out.Band.Max:
		out.BandStateFa = "بالای سقف باند"
	default:
		out.BandStateFa = "در باند"
	}

	out.Lines = []CompensationLine{
		{Key: "position", TitleFa: "P1 — شغل: امتیاز ارزشیابی و پلهٔ گروه شغلی",
			ValueFa: IntFa(int(in.JobScore)) + " امتیاز → پلهٔ " + IntFa(rank) + " («" + out.GradeFa + "»)",
			RefFa:   "پارامتر شرکت compensation.job.pointsPerGrade"},
		{Key: "band", TitleFa: "P4 — پرداخت: باند پلهٔ " + out.GradeFa,
			ValueFa: "کف " + IntMoneyFa(out.Band.Min) + " — میانه " + IntMoneyFa(out.Band.Midpoint) + " — سقف " + IntMoneyFa(out.Band.Max),
			RefFa:   "پارامترهای compensation.band.*"},
		{Key: "person", TitleFa: "P2 — شخص: سطح شایستگی «" + out.LevelFa + "»",
			ValueFa: "ضریب " + TrimFloatFa(out.PersonFactor),
			RefFa:   "پارامترهای compensation.person.*"},
		{Key: "performance", TitleFa: "P3 — عملکرد: سطح «" + out.PerformanceFa + "»",
			ValueFa: TrimFloatFa(out.PerformanceScore) + " از ۱۰۰ → ضریب " + TrimFloatFa(out.PerformanceFactor),
			RefFa:   "پارامترهای compensation.performance.*"},
	}
	if out.MarketApplied {
		out.Lines = append(out.Lines, CompensationLine{Key: "market", TitleFa: "تعدیل بازار",
			ValueFa: "+" + TrimFloatFa(p.MarketAdjustPercent) + "٪", RefFa: "پارامتر شرکت compensation.market.adjustPercent"})
	}
	out.Lines = append(out.Lines,
		CompensationLine{Key: "target", TitleFa: "پرداخت هدف", ValueFa: IntMoneyFa(out.TargetPay),
			RefFa: "میانهٔ باند × ضریب شایستگی × ضریب عملکرد"},
		CompensationLine{Key: "increase", TitleFa: "افزایش پیشنهادی", ValueFa: IntMoneyFa(out.Increase) + " («" + TrimFloatFa(out.IncreasePct) + "٪»)",
			RefFa: "سقف پارامتر شرکت compensation.increase.maxAnnualPercent"},
		CompensationLine{Key: "compa", TitleFa: "نسبت پرداخت فعلی به میانهٔ باند",
			ValueFa: TrimFloatFa(out.CompaRatioPct) + "٪ — " + out.BandStateFa,
			RefFa:   "بازهٔ قابل قبول پارامترهای compensation.compa.*"},
	)

	if out.CappedByMax {
		out.WarningsFa = append(out.WarningsFa, "پرداخت هدف به سقف باند رسید؛ برای پرداخت بیشتر باید پلهٔ گروه شغلی ارتقا یابد.")
	}
	if out.CappedByIncrease {
		out.WarningsFa = append(out.WarningsFa, "افزایش پیشنهادی به سقف افزایش سالانهٔ پارامتر شرکت محدود شد.")
	}
	if out.NoDecrease {
		out.WarningsFa = append(out.WarningsFa, "پرداخت فعلی بیش از پرداخت هدف است؛ کاهش پرداخت پیشنهاد نمی‌شود و افزایش صفر ماند.")
	}
	if !out.NoDecrease && out.CompaRatioPct > 0 && (out.CompaRatioPct < p.CompaMinPercent || out.CompaRatioPct > p.CompaMaxPercent) {
		out.WarningsFa = append(out.WarningsFa, "نسبت پرداخت به میانهٔ باند بیرون بازهٔ قابل قبول است؛ بستهٔ این کارمند در بازنگری بعدی اصلاح شود.")
	}
	if out.Increase == 0 && !out.NoDecrease {
		out.WarningsFa = append(out.WarningsFa, "پرداخت فعلی همان پرداخت هدف است؛ افزایشی لازم نیست.")
	}

	out.Valid = true
	out.SummaryFa = compensationSummaryFa(out)
	return out
}

// compensationSummaryFa جمع‌بندی یک‌خطی فارسی برای فرم و سند.
func compensationSummaryFa(r CompensationResult) string {
	parts := []string{
		"شغل: پلهٔ " + IntFa(r.Rank) + " («" + r.GradeFa + "»)",
		"شخص: " + r.LevelFa + " ×" + TrimFloatFa(r.PersonFactor),
		"عملکرد: " + r.PerformanceFa + " ×" + TrimFloatFa(r.PerformanceFactor),
		"پرداخت هدف: " + IntMoneyFa(r.TargetPay),
	}
	if r.Increase > 0 {
		parts = append(parts, "افزایش: "+IntMoneyFa(r.Increase)+" («"+TrimFloatFa(r.IncreasePct)+"٪»)")
	} else {
		parts = append(parts, "بدون افزایش")
	}
	return strings.Join(parts, "؛ ") + "."
}

// CompensationFactorTable جدول ضریب‌ها برای نمایش در فرم (سطح شایستگی و سطح عملکرد).
func CompensationFactorTable(p CompensationParams) []CompensationLine {
	out := make([]CompensationLine, 0, p.PersonLevels+4)
	for level := 1; level <= p.PersonLevels; level++ {
		out = append(out, CompensationLine{
			Key:     "person." + IntFa(level),
			TitleFa: "شایستگی سطح " + IntFa(level) + " («" + CompensationLevelTitleFa(level) + "»)",
			ValueFa: "×" + TrimFloatFa(CompensationPersonFactor(level, p)),
			RefFa:   "پارامترهای compensation.person.*",
		})
	}
	for _, score := range []float64{95, 85, 70, 50} {
		level := CompensationPerformanceLevelOf(score, p)
		out = append(out, CompensationLine{
			Key:     "performance." + level.Key,
			TitleFa: "عملکرد «" + level.TitleFa + "» (امتیاز نمونه " + TrimFloatFa(score) + ")",
			ValueFa: "×" + TrimFloatFa(level.Factor),
			RefFa:   "پارامترهای compensation.performance.*",
		})
	}
	return out
}

// CompensationInstructionFa راهنمای فارسی «از کجا شروع کنیم» برای استقرار این نظام.
//
// چرا در کد و نه در رابط: ترتیب تکمیل داده قاعدهٔ اجرایی است (هر مرحله ورودی مرحلهٔ
// بعد را می‌سازد)؛ اگر در UI نوشته شود، با تغییر رابط از دست می‌رود.
func CompensationInstructionFa() []string {
	return []string{
		"۱. ساختار و تشکیلات: شرکت/واحدهای سازمانی و پست‌های مصوب را کامل کنید؛ بدون پست و واحد، ارزشیابی شغل بی‌مبنا می‌ماند.",
		"۲. طبقه‌بندی مشاغل: شرح شغل و شناسنامهٔ شغل را برای هر پست ثبت کنید و ارزشیابی شغل (عوامل شغل) را انجام دهید تا امتیاز و پلهٔ هر شغل به دست آید.",
		"۳. پروندهٔ کارکنان: سابقهٔ تحصیلی، سابقهٔ خدمت و سمت جاری کارکنان را کامل کنید؛ سطح شایستگی از همین داده سنجیده می‌شود.",
		"۴. مدل شایستگی: سطوح شایستگی هر خانوادهٔ شغلی را تعریف و برای هر کارمند سطح جاری را ثبت کنید.",
		"۵. ارزیابی عملکرد: دورهٔ ارزیابی را ببندید و امتیاز نهایی هر کارمند را تأیید کنید؛ ضریب عملکرد از همین امتیاز می‌آید.",
		"۶. پارامترهای شرکت: میانهٔ باند پایه، شیب باند، گسترهٔ باند، گام پله، ضرایب شایستگی و عملکرد و سقف افزایش سالانه را تصویب و ثبت کنید.",
		"۷. احکام حقوق: پرداخت فعلی هر کارمند (پایهٔ حقوق و فوق‌العاده‌های ثابت) را از آخرین حکم ابلاغ‌شده بیاورید تا افزایش پیشنهادی درست محاسبه شود.",
		"۸. اجرای بازنگری: بستهٔ هر کارمند را محاسبه، با کمیتهٔ جبران خدمات تصویب و برای صدور حکم به واحد امور کارکنان بفرستید.",
	}
}
