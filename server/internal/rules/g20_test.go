// آزمون‌های نظام جیب‌یست (G20): بیست شاخص، چهار خانواده، امتیاز کل، طبقه‌بندی و مرزبندی با نظام ۴پی.
package rules

import (
	"reflect"
	"strings"
	"testing"
)

// TestG20DefaultIndices بیست شاخص و پنج شاخص در هر خانواده با مجموع وزن ۱۰۰.
func TestG20DefaultIndices(t *testing.T) {
	all := G20DefaultIndices()
	if len(all) != 20 {
		t.Fatalf("تعداد شاخص‌ها باید ۲۰ باشد، شد %d", len(all))
	}
	codes := map[string]bool{}
	for _, idx := range all {
		if codes[idx.Code] {
			t.Fatalf("کد شاخص تکراری: %s", idx.Code)
		}
		codes[idx.Code] = true
		if idx.TitleFa == "" || idx.MeasureFa == "" {
			t.Fatalf("شاخص %s عنوان یا روش سنجش ندارد", idx.Code)
		}
		if idx.Weight <= 0 {
			t.Fatalf("وزن شاخص %s باید مثبت باشد", idx.Code)
		}
	}
	for _, fam := range G20Families() {
		items := G20IndexesByFamily(fam.Key)
		if len(items) != 5 {
			t.Fatalf("خانوادهٔ %s باید ۵ شاخص داشته باشد، دارد %d", fam.Key, len(items))
		}
		var sum float64
		for _, idx := range items {
			sum += idx.Weight
		}
		if sum != 100 {
			t.Fatalf("مجموع وزن شاخص‌های %s باید ۱۰۰ باشد، شد %v", fam.Key, sum)
		}
	}
}

// TestG20ParamsFromText نرمال‌سازی وزن خانواده‌ها و نادیده‌گرفتن مقدار نامعتبر.
func TestG20ParamsFromText(t *testing.T) {
	p := G20ParamsFromText(map[string]string{
		"compensation.g20.family.person":      "1",
		"compensation.g20.family.position":    "1",
		"compensation.g20.family.performance": "1",
		"compensation.g20.family.partnership": "1",
		"compensation.g20.index.max":          "-5",   // نامعتبر: پیش‌فرض می‌ماند
		"compensation.g20.cut.b":              "72.0", // معتبر
	})
	if got := p.PersonWeight + p.PositionWeight + p.PerformanceWeight + p.PartnershipWeight; got < 99.99 || got > 100.01 {
		t.Fatalf("مجموع وزن خانواده‌ها باید ۱۰۰ شود، شد %v", got)
	}
	if p.IndexMax != 100 {
		t.Fatalf("سقف شاخص نامعتبر باید پیش‌فرض ۱۰۰ بماند، شد %v", p.IndexMax)
	}
	if p.CutB != 72 {
		t.Fatalf("آستانهٔ B باید ۷۲ شود، شد %v", p.CutB)
	}
}

// TestG20Calc امتیاز کل و طبقهٔ نمونهٔ محاسبه‌شده.
func TestG20Calc(t *testing.T) {
	p := DefaultG20Params()
	res := G20Calc(G20Input{Person: 80, Position: 90, Performance: 70, Partnership: 60}, p)
	// 80*.25 + 90*.25 + 70*.30 + 60*.20 = 20 + 22.5 + 21 + 12 = 75.5
	if res.Total != 75.5 {
		t.Fatalf("امتیاز کل باید ۷۵٫۵ باشد، شد %v", res.Total)
	}
	if res.SegmentKey != "B" {
		t.Fatalf("طبقه باید B (پیشرو) باشد، شد %s", res.SegmentKey)
	}
	if len(res.Contrib) != 4 {
		t.Fatalf("سهم چهار خانواده باید گزارش شود، شد %d", len(res.Contrib))
	}
	var shareSum float64
	for _, c := range res.Contrib {
		shareSum += c.Share
	}
	if diff := shareSum - res.Total; diff > 0.05 || diff < -0.05 {
		t.Fatalf("جمع سهم خانواده‌ها باید برابر امتیاز کل باشد: %v در برابر %v", shareSum, res.Total)
	}
}

// TestG20CalcClamp امتیاز بیرون از بازه به بازهٔ معتبر بریده می‌شود.
func TestG20CalcClamp(t *testing.T) {
	p := DefaultG20Params()
	high := G20Calc(G20Input{Person: 500, Position: 120, Performance: 100, Partnership: 100}, p)
	if high.Total != 100 {
		t.Fatalf("امتیاز بیش از سقف باید ۱۰۰ شود، شد %v", high.Total)
	}
	if high.SegmentKey != "A" {
		t.Fatalf("امتیاز صد باید طبقهٔ A باشد، شد %s", high.SegmentKey)
	}
	low := G20Calc(G20Input{Person: -10, Position: 0, Performance: 0, Partnership: 0}, p)
	if low.Total != 0 || low.SegmentKey != "E" {
		t.Fatalf("امتیاز منفی باید صفر و طبقهٔ E شود: %v / %s", low.Total, low.SegmentKey)
	}
}

// TestG20SegmentBoundaries مرزهای طبقه‌بندی بر پایهٔ پارامترهای شرکت.
func TestG20SegmentBoundaries(t *testing.T) {
	p := DefaultG20Params()
	cases := []struct {
		total float64
		want  string
	}{
		{100, "A"}, {85, "A"}, {84.9, "B"}, {70, "B"}, {69.9, "C"},
		{55, "C"}, {54.9, "D"}, {40, "D"}, {39.9, "E"}, {0, "E"},
	}
	for _, c := range cases {
		key, _ := G20SegmentOf(p, c.total)
		if key != c.want {
			t.Fatalf("امتیاز %v باید طبقهٔ %s باشد، شد %s", c.total, c.want, key)
		}
	}
	// آستانه‌ها پارامتری‌اند: با تغییر پارامتر طبقه‌بندی هم جابه‌جا می‌شود.
	p2 := G20ParamsFromText(map[string]string{"compensation.g20.cut.a": "60"})
	if key, _ := G20SegmentOf(p2, 75.5); key != "A" {
		t.Fatalf("با آستانهٔ ۶۰، امتیاز ۷۵٫۵ باید A شود، شد %s", key)
	}
}

// TestG20ScoreFromIndices میانگین وزنی شاخص‌های یک خانواده.
func TestG20ScoreFromIndices(t *testing.T) {
	items := G20IndexesByFamily("performance")
	scores := make([]float64, len(items))
	weights := make([]float64, len(items))
	for i, idx := range items {
		scores[i] = 100
		weights[i] = idx.Weight
	}
	if got := G20ScoreFromIndices(scores, weights); got != 100 {
		t.Fatalf("همهٔ شاخص‌ها ۱۰۰: میانگین باید ۱۰۰ شود، شد %v", got)
	}
	if got := G20ScoreFromIndices(nil, nil); got != 0 {
		t.Fatalf("بی‌شاخص باید صفر شود، شد %v", got)
	}
}

// TestG20IndependenceFa متن استقلال مدل در دو حالت.
func TestG20IndependenceFa(t *testing.T) {
	off := G20IndependenceFa(false)
	if !strings.Contains(off, "خاموش") || !strings.Contains(off, "صندوق نفت") {
		t.Fatalf("متن خاموش باید مرز مدل‌ها را بگوید: %s", off)
	}
	on := G20IndependenceFa(true)
	if !strings.Contains(on, "احکام") || on == off {
		t.Fatalf("متن روشن باید تفاوت داشته باشد: %s", on)
	}
	if G20InstructionFa() == "" {
		t.Fatal("راهنمای شروع نباید خالی باشد")
	}
}

// ---- مرزبندی دو نظام: جبران خدمات ۴پی و نظام پرداخت جیب‌یست (G20) ----

// g20DecoupleParamKeys کلیدهای پارامتریک نظام جیب‌یست؛ همان کلیدهایی که در پاسخ
// /api/v1/compensation/g20-options دیده می‌شوند.
func g20DecoupleParamKeys() []string {
	return []string{
		"compensation.g20.family.person",
		"compensation.g20.family.position",
		"compensation.g20.family.performance",
		"compensation.g20.family.partnership",
		"compensation.g20.index.max",
		"compensation.g20.cut.a",
		"compensation.g20.cut.b",
		"compensation.g20.cut.c",
		"compensation.g20.cut.d",
		"compensation.g20.payroll.apply",
	}
}

// TestG20DecoupleNoLinkParameter نظام جیب‌یست هیچ پارامتر یا فهرست پیوندی به نظام ۴پی
// ندارد: پارامتر پیوند پارتنرشیپ و میدان آن در نوع پارامترها حذف شده است، هیچ کلید
// پارامتریک از فضای نام خودش بیرون نمی‌زند و کلیدهای ۴پی پارامتر جیب‌یست را عوض نمی‌کنند.
func TestG20DecoupleNoLinkParameter(t *testing.T) {
	// کلید حذف‌شدهٔ پیوند، دوتکه نوشته می‌شود تا در جست‌وجوی کد با پارامتر زندهٔ نظام
	// دیگری اشتباه گرفته نشود؛ همین که نوشتنش بی‌اثر است، خودِ آزمون است.
	removedLinkKey := "compensation.g20.partnership" + ".maxFactor"
	if got := G20ParamsFromText(map[string]string{removedLinkKey: "1.8"}); got != DefaultG20Params() {
		t.Fatalf("کلید پیوند نباید هیچ پارامتری را عوض کند: %+v", got)
	}
	if _, ok := reflect.TypeOf(G20Params{}).FieldByName("PartnershipMaxFactor"); ok {
		t.Fatal("میدان PartnershipMaxFactor (ضریب پیوند پارتنرشیپ با ۴پی) باید حذف شده باشد")
	}
	for _, key := range g20DecoupleParamKeys() {
		if !strings.HasPrefix(key, "compensation.g20.") {
			t.Fatalf("کلید پارامتری جیب‌یست باید در فضای نام خودش بماند: %s", key)
		}
		if strings.Contains(key, "maxFactor") || strings.Contains(key, "fourp") || strings.Contains(key, "4p") {
			t.Fatalf("کلید پارامتری جیب‌یست نباید به نظام ۴پی اشاره کند: %s", key)
		}
	}
	// هیچ کلید ۴پی هم پارامتر جیب‌یست را عوض نمی‌کند.
	fourPKeys := map[string]string{
		"compensation.payroll.apply":          "1",
		"compensation.person.maxFactor":       "1.5",
		"compensation.band.baseMidpoint":      "999999999",
		"compensation.performance.factorGood": "1.9",
	}
	if got := G20ParamsFromText(fourPKeys); got != DefaultG20Params() {
		t.Fatalf("کلیدهای ۴پی نباید پارامترهای جیب‌یست را تغییر دهند: %+v", got)
	}
	// خانوادهٔ پارتنرشیپ فقط یک خانوادهٔ امتیاز درون جیب‌یست است، نه پیوند با ۴پی.
	for _, fam := range G20Families() {
		if strings.Contains(fam.TitleFa, "۴پی") || strings.Contains(fam.BriefFa, "۴پی") || strings.Contains(fam.BriefFa, "مدل ۴پی") {
			t.Fatalf("خانوادهٔ %s نباید به نظام ۴پی اشاره کند", fam.Key)
		}
	}
}

// TestG20DecoupleG20ParamsDoNotAffect4P تغییر پارامترهای نظام جیب‌یست هیچ اثری بر خروجی
// مدل ۴پی ندارد، حتی اگر «اعمال در پرداخت» جیب‌یست روشن شود.
func TestG20DecoupleG20ParamsDoNotAffect4P(t *testing.T) {
	in := CompensationInput{
		EmployeeID: "EMP-001", JobScore: 350, CompetencyLevel: 4,
		PerformanceScore: 88, CurrentPay: 210_000_000,
	}
	p := DefaultCompensationParams()
	want := CompensationCalc(in, p)

	changed := G20ParamsFromText(map[string]string{
		"compensation.g20.family.person":      "90",
		"compensation.g20.family.position":    "5",
		"compensation.g20.family.performance": "1",
		"compensation.g20.family.partnership": "60",
		"compensation.g20.index.max":          "250",
		"compensation.g20.cut.a":              "10",
		"compensation.g20.cut.d":              "1",
		"compensation.g20.payroll.apply":      "1",
	})
	if changed == DefaultG20Params() {
		t.Fatal("پارامترهای جیب‌یست باید با کلیدهای شرکت عوض شوند (وگرنه این آزمون بی‌اثر است)")
	}
	if got := CompensationCalc(in, p); !reflect.DeepEqual(want, got) {
		t.Fatalf("تغییر پارامترهای جیب‌یست نباید خروجی ۴پی را عوض کند:\n خواسته %+v\n گرفته %+v", want, got)
	}
	// «اعمال در پرداخت» جیب‌یست کلید ۴پی نیست و نباید مدل ۴پی را روشن کند.
	if on := CompensationParamsFromText(map[string]string{"compensation.g20.payroll.apply": "1"}); on != DefaultCompensationParams() {
		t.Fatalf("کلید اعمال در پرداخت جیب‌یست نباید پارامتر ۴پی را عوض کند: %+v", on)
	}
}

// TestG20Decouple4PParamsDoNotAffectG20 تغییر پارامترهای نظام ۴پی هیچ اثری بر خروجی نظام
// جیب‌یست ندارد.
func TestG20Decouple4PParamsDoNotAffectG20(t *testing.T) {
	in := G20Input{Person: 82, Position: 78, Performance: 88, Partnership: 70}
	g20 := DefaultG20Params()
	want := G20Calc(in, g20)

	changed := CompensationParamsFromText(map[string]string{
		"compensation.band.baseMidpoint":         "500000000",
		"compensation.band.slopePercent":         "50",
		"compensation.band.spreadPercent":        "150",
		"compensation.person.maxFactor":          "2",
		"compensation.performance.goodScore":     "55",
		"compensation.increase.maxAnnualPercent": "90",
		"compensation.payroll.apply":             "1",
	})
	if changed == DefaultCompensationParams() {
		t.Fatal("پارامترهای ۴پی باید با کلیدهای شرکت عوض شوند (وگرنه این آزمون بی‌اثر است)")
	}
	if got := G20Calc(in, g20); !reflect.DeepEqual(want, got) {
		t.Fatalf("تغییر پارامترهای ۴پی نباید خروجی جیب‌یست را عوض کند:\n خواسته %+v\n گرفته %+v", want, got)
	}
	// «اعمال در پرداخت» ۴پی هم پارامتر جیب‌یست را روشن نمی‌کند.
	if on := G20ParamsFromText(map[string]string{"compensation.payroll.apply": "1"}); on != DefaultG20Params() {
		t.Fatalf("کلید اعمال در پرداخت ۴پی نباید پارامتر جیب‌یست را عوض کند: %+v", on)
	}
}
