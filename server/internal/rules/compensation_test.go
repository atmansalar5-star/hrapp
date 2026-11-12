package rules

import (
	"strings"
	"testing"
)

// TestCompensationDefaults پارامترهای پیش‌فرض باید در بازهٔ معقول و سازگار باشند.
func TestCompensationDefaults(t *testing.T) {
	p := DefaultCompensationParams()
	if p.BandBaseMidpoint <= 0 {
		t.Fatal("میانهٔ باند پایه باید مثبت باشد")
	}
	if p.PersonMinFactor > p.PersonMaxFactor {
		t.Fatal("کف ضریب شایستگی نباید از سقف بزرگ‌تر باشد")
	}
	if p.CompaMinPercent >= p.CompaMaxPercent {
		t.Fatal("بازهٔ نسبت پرداخت به میانه نامعتبر است")
	}
	if p.JobMaxGrade != len(Grades()) {
		t.Fatalf("سقف پلهٔ شغل باید با نردبان گروه‌های سازمان یکی باشد: %d در برابر %d", p.JobMaxGrade, len(Grades()))
	}
}

// TestCompensationParamsFromText پارامترهای نوشته‌شده در پنل (ارقام فارسی، درصد، جداکننده) خوانده شوند.
func TestCompensationParamsFromText(t *testing.T) {
	p := CompensationParamsFromText(map[string]string{
		"compensation.band.baseMidpoint":         "۱۵۰٬۰۰۰٬۰۰۰",
		"compensation.band.slopePercent":         "۲۰٪",
		"compensation.band.spreadPercent":        "50",
		"compensation.person.maxFactor":          "1.2",
		"compensation.performance.factorGood":    "1.1",
		"compensation.increase.maxAnnualPercent": "۳۰",
		"compensation.job.maxGrade":              "999",
		"payroll.overtimeFactor":                 "9", // کلید بیگانه نباید اثر کند
	})
	if p.BandBaseMidpoint != 150_000_000 {
		t.Fatalf("میانهٔ باند از متن خوانده نشد: %d", p.BandBaseMidpoint)
	}
	if p.BandSlopePercent != 20 || p.BandSpreadPercent != 50 {
		t.Fatalf("درصدهای باند درست خوانده نشد: %v/%v", p.BandSlopePercent, p.BandSpreadPercent)
	}
	if p.PersonMaxFactor != 1.2 || p.PerfFactorGood != 1.1 {
		t.Fatalf("ضرایب درست خوانده نشد: %v/%v", p.PersonMaxFactor, p.PerfFactorGood)
	}
	if p.MaxAnnualIncreasePercent != 30 {
		t.Fatalf("سقف افزایش سالانه خوانده نشد: %v", p.MaxAnnualIncreasePercent)
	}
	if p.JobMaxGrade != DefaultCompensationParams().JobMaxGrade {
		t.Fatalf("پلهٔ بیرون بازه باید پیش‌فرض بماند: %d", p.JobMaxGrade)
	}
}

// TestCompensationBands باندها صعودی و کف/میانه/سقف مرتب باشند.
func TestCompensationBands(t *testing.T) {
	p := DefaultCompensationParams()
	bands := CompensationBands(p)
	if len(bands) != p.JobMaxGrade {
		t.Fatalf("شمار باندها باید %d باشد: %d", p.JobMaxGrade, len(bands))
	}
	for i, b := range bands {
		if b.Min >= b.Midpoint || b.Midpoint >= b.Max {
			t.Fatalf("ترتیب کف/میانه/سقف باند پلهٔ %d نادرست است: %+v", b.Rank, b)
		}
		if b.Step <= 0 {
			t.Fatalf("گام پلهٔ %d باید مثبت باشد", b.Rank)
		}
		if b.GradeFa == "" {
			t.Fatalf("عنوان فارسی پلهٔ %d خالی است", b.Rank)
		}
		if i > 0 && b.Midpoint <= bands[i-1].Midpoint {
			t.Fatalf("میانهٔ باند پلهٔ %d باید از پلهٔ قبل بیشتر باشد", b.Rank)
		}
	}
	// گسترهٔ باند پارامتریک است: با دو برابر شدن گستره، سقف بالاتر می‌رود.
	wide := CompensationBandFor(3, CompensationParamsFromValues(map[string]float64{"compensation.band.spreadPercent": 80}))
	if wide.Max <= CompensationBandFor(3, p).Max {
		t.Fatal("افزایش گسترهٔ باند باید سقف را بالا ببرد")
	}
}

// TestCompensationGradeFromScore امتیاز شغل به پلهٔ درست نگاشت شود و از سقف نگذرد.
func TestCompensationGradeFromScore(t *testing.T) {
	p := DefaultCompensationParams() // هر ۶۰ امتیاز یک پله، سقف ۶
	cases := []struct {
		score float64
		want  int
	}{
		{0, 1}, {59, 1}, {60, 2}, {180, 4}, {299, 5}, {300, 6}, {900, 6}, {-20, 1},
	}
	for _, tc := range cases {
		if got := CompensationGradeFromScore(tc.score, p); got != tc.want {
			t.Fatalf("امتیاز %v باید پلهٔ %d شود، شد %d", tc.score, tc.want, got)
		}
	}
}

// TestCompensationFactors ضریب شایستگی و سطح عملکرد درست محاسبه شوند.
func TestCompensationFactors(t *testing.T) {
	p := DefaultCompensationParams()
	if got := CompensationPersonFactor(1, p); got != 0.9 {
		t.Fatalf("کف ضریب شایستگی باید ۰٫۹ باشد: %v", got)
	}
	if got := CompensationPersonFactor(5, p); got != 1.1 {
		t.Fatalf("سقف ضریب شایستگی باید ۱٫۱ باشد: %v", got)
	}
	if got := CompensationPersonFactor(3, p); got != 1 {
		t.Fatalf("سطح میانی باید ضریب ۱ بدهد: %v", got)
	}
	if got := CompensationPersonFactor(99, p); got != 1.1 {
		t.Fatalf("سطح بیرون از مدل باید به سقف محدود شود: %v", got)
	}
	if title := CompensationLevelTitleFa(4); title != "پیشرفته" {
		t.Fatalf("عنوان سطح ۴ نادرست است: %s", title)
	}
	if lvl := CompensationPerformanceLevelOf(92, p); lvl.Key != "excellent" {
		t.Fatalf("امتیاز ۹۲ باید «عالی» باشد: %+v", lvl)
	}
	if lvl := CompensationPerformanceLevelOf(70, p); lvl.TitleFa != "قابل قبول" {
		t.Fatalf("امتیاز ۷۰ باید «قابل قبول» باشد: %+v", lvl)
	}
	if lvl := CompensationPerformanceLevelOf(40, p); lvl.Factor != p.PerfFactorBelow {
		t.Fatalf("امتیاز پایین باید ضریب «نیازمند بهبود» بگیرد: %+v", lvl)
	}
}

// TestCompensationCalcTarget پرداخت هدف = میانهٔ باند × ضریب شایستگی × ضریب عملکرد.
func TestCompensationCalcTarget(t *testing.T) {
	p := DefaultCompensationParams()
	// پرداخت فعلی کمی زیر میانهٔ باند پلهٔ ۳: داخل باند می‌ماند و افزایش لازم دارد.
	band := CompensationBandFor(3, p)
	in := CompensationInput{EmployeeID: "EMP-001", JobScore: 150, CompetencyLevel: 4, PerformanceScore: 85,
		CurrentPay: band.Midpoint * 95 / 100}
	res := CompensationCalc(in, p)
	if !res.Valid {
		t.Fatalf("محاسبه باید معتبر باشد: %s", res.MessageFa)
	}
	if res.Rank != 3 {
		t.Fatalf("امتیاز ۱۵۰ باید پلهٔ ۳ بدهد: %d", res.Rank)
	}
	want := RoundHalfUp(float64(res.Band.Midpoint) * res.PersonFactor * res.PerformanceFactor)
	if res.TargetPay != want {
		t.Fatalf("پرداخت هدف %d است، انتظار %d", res.TargetPay, want)
	}
	if res.Increase <= 0 {
		t.Fatalf("با پرداخت فعلی پایین‌تر از هدف، افزایش باید مثبت باشد: %d", res.Increase)
	}
	if res.Increase > res.TargetPay-res.CurrentPay {
		t.Fatal("افزایش نباید از فاصلهٔ پرداخت فعلی تا هدف بیشتر شود")
	}
	if len(res.Lines) < 6 || res.SummaryFa == "" {
		t.Fatalf("سطرهای محاسبه و جمع‌بندی باید ساخته شوند: %d/%s", len(res.Lines), res.SummaryFa)
	}
	if res.BandStateFa != "در باند" {
		t.Fatalf("وضعیت باند باید «در باند» باشد: %s", res.BandStateFa)
	}
}

// TestCompensationCalcCaps سه سقف: سقف باند، نبود کاهش پرداخت و سقف افزایش سالانه.
func TestCompensationCalcCaps(t *testing.T) {
	p := DefaultCompensationParams()

	// ۱) بالاترین پله + بالاترین ضریب‌ها: پرداخت هدف نباید از سقف باند بگذرد.
	top := CompensationCalc(CompensationInput{JobScore: 400, CompetencyLevel: 5, PerformanceScore: 100,
		CurrentPay: 10_000_000, MarketAdjust: true}, p)
	if !top.CappedByMax {
		t.Fatal("پرداخت هدف باید به سقف باند محدود شود")
	}
	if top.TargetPay != top.Band.Max {
		t.Fatalf("پرداخت هدف باید برابر سقف باند باشد: %d/%d", top.TargetPay, top.Band.Max)
	}

	// ۲) پرداخت فعلی بیش از سقف باند: کاهش پیشنهاد نمی‌شود.
	over := CompensationCalc(CompensationInput{JobScore: 60, CompetencyLevel: 3, PerformanceScore: 80,
		CurrentPay: 900_000_000}, p)
	if over.Increase != 0 || !over.NoDecrease {
		t.Fatalf("کاهش پرداخت نباید پیشنهاد شود: افزایش %d، پرچم %v", over.Increase, over.NoDecrease)
	}
	if over.BandStateFa != "بالای سقف باند" {
		t.Fatalf("وضعیت باید «بالای سقف باند» باشد: %s", over.BandStateFa)
	}

	// ۳) سقف افزایش سالانه: افزایش از درصد پارامتر بیشتر نشود.
	capped := CompensationCalc(CompensationInput{JobScore: 400, CompetencyLevel: 5, PerformanceScore: 100,
		CurrentPay: 50_000_000}, p)
	limit := RoundHalfUp(50_000_000 * p.MaxAnnualIncreasePercent / 100)
	if capped.Increase != limit || !capped.CappedByIncrease {
		t.Fatalf("افزایش باید به سقف سالانه محدود شود: %d در برابر %d", capped.Increase, limit)
	}
}

// TestCompensationCalcMarketAdjust تعدیل بازار فقط با درخواست صریح اثر کند.
func TestCompensationCalcMarketAdjust(t *testing.T) {
	p := DefaultCompensationParams()
	base := CompensationInput{JobScore: 120, CompetencyLevel: 3, PerformanceScore: 80, CurrentPay: 40_000_000}
	plain := CompensationCalc(base, p)
	withMarket := base
	withMarket.MarketAdjust = true
	market := CompensationCalc(withMarket, p)
	if !market.MarketApplied || market.TargetPay <= plain.TargetPay {
		t.Fatalf("تعدیل بازار باید پرداخت هدف را بالا ببرد: %d در برابر %d", market.TargetPay, plain.TargetPay)
	}
	if plain.MarketApplied {
		t.Fatal("بدون درخواست، تعدیل بازار نباید اعمال شود")
	}
}

// TestCompensationCalcInvalidGrade پلهٔ دستی بیرون از نردبان باید رد شود.
func TestCompensationCalcInvalidGrade(t *testing.T) {
	res := CompensationCalc(CompensationInput{EmployeeID: "EMP-009", GradeOverride: 99}, DefaultCompensationParams())
	if res.Valid || !strings.Contains(res.MessageFa, "نردبان") {
		t.Fatalf("پلهٔ نامعتبر باید پیام فارسی بدهد: %v/%s", res.Valid, res.MessageFa)
	}
}

// TestCompensationInstructionAndFactorTable راهنما و جدول ضریب‌ها کامل باشند.
func TestCompensationInstructionAndFactorTable(t *testing.T) {
	steps := CompensationInstructionFa()
	if len(steps) < 6 {
		t.Fatalf("راهنمای شروع باید دست‌کم ۶ گام داشته باشد: %d", len(steps))
	}
	for _, s := range steps {
		if !strings.Contains(s, "کارکنان") && !strings.Contains(s, "پارامتر") && !strings.Contains(s, "شغل") &&
			!strings.Contains(s, "عملکرد") && !strings.Contains(s, "احکام") && !strings.Contains(s, "ارزیابی") &&
			!strings.Contains(s, "طبقه‌بندی") && !strings.Contains(s, "ساختار") {
			t.Fatalf("گام راهنما باید موضوع دادهٔ روشن داشته باشد: %s", s)
		}
	}
	table := CompensationFactorTable(DefaultCompensationParams())
	if len(table) < 8 {
		t.Fatalf("جدول ضریب‌ها باید سطوح شایستگی و عملکرد را داشته باشد: %d", len(table))
	}
}
