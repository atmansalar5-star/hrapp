// آزمون دامنهٔ ماژول «hrstudies» (indices_test).

package domain

import (
	"strconv"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// TestEvaluateIndicator سنجش شاخص با هدف را برای هر دو جهت مطلوب می‌سنجد.
//
// چرا این آزمون مهم است: اگر جهت مطلوب نادیده گرفته شود، شاخص «کمتر بهتر» (نرخ ترک
// خدمت) نتیجهٔ وارونه می‌دهد — یعنی سازمانی که وضعش بد شده، «در محدودهٔ هدف» نشان
// داده می‌شود. این آزمون همان مرزها را قفل می‌کند.
func TestEvaluateIndicator(t *testing.T) {
	cases := []struct {
		name     string
		current  float64
		target   float64
		dir      Direction
		wantGap  float64
		wantVerd Assessment
	}{
		{"درست روی هدف — کمتر بهتر", 5, 5, DirectionLower, 0, AssessmentInTarget},
		{"انحراف کوچک — کمتر بهتر", 5.2, 5, DirectionLower, 4, AssessmentInTarget},
		{"نزدیک هدف — کمتر بهتر", 5.7, 5, DirectionLower, 14, AssessmentNearTarget},
		{"خارج از هدف — کمتر بهتر", 7, 5, DirectionLower, 40, AssessmentOutOfTarget},
		{"عبور از هدف بهتر — کمتر بهتر", 3, 5, DirectionLower, -40, AssessmentInTarget},

		{"درست روی هدف — بیشتر بهتر", 24, 24, DirectionHigher, 0, AssessmentInTarget},
		{"نزدیک هدف — بیشتر بهتر", 21, 24, DirectionHigher, 12.5, AssessmentNearTarget},
		{"خارج از هدف — بیشتر بهتر", 17, 24, DirectionHigher, 29.2, AssessmentOutOfTarget},
		{"عبور از هدف بهتر — بیشتر بهتر", 27, 24, DirectionHigher, -12.5, AssessmentInTarget},

		{"هدف صفر و مقدار صفر", 0, 0, DirectionLower, 0, AssessmentInTarget},
		{"هدف صفر و مقدار غیرصفر", 3, 0, DirectionHigher, 100, AssessmentOutOfTarget},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, gap, verdict := EvaluateIndicator(c.current, c.target, c.dir)
			if verdict != c.wantVerd {
				t.Fatalf("وضعیت سنجش باید %s باشد، %s شد", c.wantVerd, verdict)
			}
			if gap != c.wantGap {
				t.Fatalf("انحراف باید %v درصد باشد، %v شد", c.wantGap, gap)
			}
		})
	}
}

// TestEvaluateIndicatorAchieved درصدد تحقق هدف را می‌سنجد (برای نمایش پیشرفت).
func TestEvaluateIndicatorAchieved(t *testing.T) {
	achieved, _, verdict := EvaluateIndicator(4.2, 5, DirectionLower)
	if achieved != 84 {
		t.Fatalf("درصد تحقق باید ۸۴ باشد، %v شد", achieved)
	}
	if verdict != AssessmentInTarget {
		t.Fatalf("نرخ ترک خدمت ۴٫۲ در برابر هدف ۵ باید در هدف باشد، %s شد", verdict)
	}
	achieved, _, _ = EvaluateIndicator(27, 24, DirectionHigher)
	if achieved != 112.5 {
		t.Fatalf("درصد تحقق باید ۱۱۲٫۵ باشد، %v شد", achieved)
	}
}

// TestParticipationRate نرخ مشارکت و مرزهای آن (جامعهٔ صفر، بیش از صد درصد).
func TestParticipationRate(t *testing.T) {
	cases := []struct {
		participants, population int
		want                     float64
	}{
		{312, 380, 82.1},
		{156, 240, 65},
		{0, 100, 0},
		{50, 0, 0},
		{-5, 100, 0},
		{120, 100, 100},
	}
	for _, c := range cases {
		if got := ParticipationRate(c.participants, c.population); got != c.want {
			t.Fatalf("نرخ مشارکت %d/%d باید %v باشد، %v شد", c.participants, c.population, c.want, got)
		}
	}
}

// TestParticipationBand مرزهای دستهٔ کیفی مشارکت را قفل می‌کند.
func TestParticipationBand(t *testing.T) {
	cases := []struct {
		rate  float64
		wantK string
	}{
		{82.1, "good"},
		{75, "good"},
		{74.9, "acceptable"},
		{50, "acceptable"},
		{49.9, "weak"},
		{0, "weak"},
	}
	for _, c := range cases {
		if key, label := ParticipationBand(c.rate); key != c.wantK || label == "" {
			t.Fatalf("دستهٔ مشارکت %v باید %s باشد، %s شد", c.rate, c.wantK, key)
		}
	}
}

// TestValidIndicatorDirection فقط دو مقدار اعلامی را می‌پذیرد.
func TestValidIndicatorDirection(t *testing.T) {
	for _, v := range []string{"higher", "lower", "HIGHER", " lower "} {
		if !ValidIndicatorDirection(v) {
			t.Fatalf("«%s» باید جهت مجاز باشد", v)
		}
	}
	for _, v := range []string{"", "up", "بیشتر بهتر", "both"} {
		if ValidIndicatorDirection(v) {
			t.Fatalf("«%s» نباید جهت مجاز باشد", v)
		}
	}
}

// TestSeedCoversIndicatorAndSurveyStates تضمین می‌کند دادهٔ نمونه هر سه حالت سنجش شاخص
// و همهٔ وضعیت‌های نظرسنجی را پوشش می‌دهد، وگرنه صفحهٔ تحلیل خالی/یکنواخت می‌شود.
func TestSeedCoversIndicatorAndSurveyStates(t *testing.T) {
	counts := map[string]int{}
	bands := map[string]int{}
	surveyStatuses := map[string]int{}
	indices := 0

	for _, in := range Seed() {
		switch in.Kind {
		case "hrIndices":
			indices++
			if !ValidIndicatorDirection(in.Fields["direction"]) {
				t.Fatalf("جهت مطلوب شاخص «%s» نامعتبر است: «%s»", in.Title, in.Fields["direction"])
			}
			if numOr(in.Fields["target"]) == 0 {
				t.Fatalf("شاخص «%s» هدف ندارد", in.Title)
			}
			_, _, verdict := EvaluateIndicator(
				numOr(in.Fields["current"]), numOr(in.Fields["target"]),
				ParseDirection(in.Fields["direction"]))
			counts[string(verdict)]++
		case "surveys":
			surveyStatuses[defaultStatus(in.SeedStatus)]++
			rate := ParticipationRate(int(numOr(in.Fields["participants"])), int(numOr(in.Fields["population"])))
			key, _ := ParticipationBand(rate)
			bands[key]++
		}
	}
	if indices == 0 {
		t.Fatal("هیچ شاخص نمونه‌ای بذرکاری نشده است")
	}
	for _, want := range []string{string(AssessmentInTarget), string(AssessmentNearTarget), string(AssessmentOutOfTarget)} {
		if counts[want] == 0 {
			t.Fatalf("دادهٔ نمونه حالت سنجش «%s» را پوشش نمی‌دهد", want)
		}
	}
	for _, want := range []string{"good", "acceptable", "weak"} {
		if bands[want] == 0 {
			t.Fatalf("دادهٔ نمونه دستهٔ مشارکت «%s» را پوشش نمی‌دهد", want)
		}
	}
	for _, want := range []string{"draft", "open", "closed"} {
		if surveyStatuses[want] == 0 {
			t.Fatalf("دادهٔ نمونه وضعیت نظرسنجی «%s» را پوشش نمی‌دهد", want)
		}
	}
}

// TestDescriptorDeclaresIndicatorKind بررسی می‌کند نوع رکورد شاخص با فیلدهای اجباری
// اعلام شده باشد، چون صفحهٔ رکوردی از همین شناسه فرم و جدول می‌سازد.
func TestDescriptorDeclaresIndicatorKind(t *testing.T) {
	kind, ok := Descriptor().KindByKey("hrIndices")
	if !ok {
		t.Fatal("نوع رکورد «شاخص سرمایهٔ انسانی» در شناسهٔ ماژول اعلام نشده است")
	}
	required := map[string]bool{}
	for _, f := range kind.Fields {
		if f.Required {
			required[f.Key] = true
		}
	}
	for _, key := range []string{"formula", "target", "current", "direction"} {
		if !required[key] {
			t.Fatalf("فیلد «%s» در نوع شاخص باید اجباری باشد", key)
		}
	}
	if err := kind.Validate(contracts.RecordInput{Kind: "hrIndices", Fields: map[string]string{"direction": "sideways"}}); err == nil {
		t.Fatal("جهت مطلوب ناشناخته باید رد شود")
	}
	if err := kind.Validate(contracts.RecordInput{Kind: "hrIndices", Fields: map[string]string{"direction": "lower"}}); err != nil {
		t.Fatalf("جهت «کمتر بهتر» باید پذیرفته شود: %v", err)
	}
}

// defaultStatus وضعیت آغازینِ پیش‌فرض دادهٔ نمونه را برمی‌گرداند (خالی یعنی draft).
func defaultStatus(seed string) string {
	if seed == "" {
		return "draft"
	}
	return seed
}

// numOr مقدار عددی فیلد دادهٔ نمونه را می‌خواند (ارقام لاتین؛ نامعتبر یعنی صفر).
func numOr(raw string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return v
}
