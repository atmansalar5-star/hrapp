package rules

import (
	"math"
	"testing"
)

// TestWageComponentsTwoStepOrder می‌سنجد ثابت‌های شغلی روی «مزد شغل و ملحقات» بسته می‌شوند،
// نه روی مزد مبنا؛ اگر ترتیب عوض شود این آزمون می‌شکند (چون مزد مبنا خودش شامل همان
// ثابت‌هاست و محاسبه دوری می‌شود).
func TestWageComponentsTwoStepOrder(t *testing.T) {
	wages := WageComponentInput{
		Job:           100_000_000,
		Rank:          20_000_000,
		Seniority:     10_000_000,
		PostAllowance: 5_000_000,
		Climate:       8_000_000,
		SalaryRepair:  2_000_000,
		// ثابت‌های شغلی: ۳۰٪ + ۱۰٪ + ۸٪ + ۳۸٪ + ۲۷٪ = ۱۱۳٪ از ۱۳۰٬۰۰۰٬۰۰۰
		SpecialPercent:          0.30,
		WorkshopPercent:         0.10,
		AttractionPublicPercent: 0.08,
		PersonalBasePercent:     0.38,
		JobSpecialtyPercent:     0.27,
	}.Build()

	if got, want := wages.JobAndAttachments(), 130_000_000.0; math.Abs(got-want) > 1e-6 {
		t.Fatalf("مزد شغل و ملحقات = %v، انتظار %v", got, want)
	}
	if got, want := wages.JobConstants(), 146_900_000.0; math.Abs(got-want) > 1e-6 {
		t.Fatalf("ثابت‌های شغلی = %v، انتظار %v", got, want)
	}
	// مزد مبنا = ۱۳۰٬۰۰۰٬۰۰۰ + ۱۴۶٬۹۰۰٬۰۰۰ + ۵٬۰۰۰٬۰۰۰ + ۸٬۰۰۰٬۰۰۰ + ۲٬۰۰۰٬۰۰۰
	if got, want := wages.Total(), 291_900_000.0; math.Abs(got-want) > 1e-6 {
		t.Fatalf("مزد مبنا = %v، انتظار %v", got, want)
	}
	// اگر ثابت‌ها روی مزد مبنا بسته می‌شدند، عدد از این بزرگ‌تر می‌شد و این بررسی می‌افتد.
	if wages.Total() <= wages.JobConstants()+wages.JobAndAttachments() {
		t.Fatalf("مزد مبنا باید بیشتر از جمع مزد شغل و ملحقات با ثابت‌ها باشد")
	}
}

// TestWageComponentsBreakdown اجزای گزارش‌شده را می‌سنجد: جزء صفر نباید بیاید و اقلیمی
// باید جزو شاکلهٔ مزد مبنا باشد.
func TestWageComponentsBreakdown(t *testing.T) {
	wages := WageComponents{Job: 50_000_000, Climate: 7_000_000, PostAllowance: 3_000_000}
	parts := wages.Breakdown()
	if len(parts) != 3 {
		t.Fatalf("%d جزء گزارش شد، انتظار ۳: %+v", len(parts), parts)
	}
	found := map[string]float64{}
	for _, p := range parts {
		found[p.Code] = p.Amount
	}
	if found["CLIMATE"] != 7_000_000 {
		t.Fatalf("فوق‌العادهٔ اقلیمی در شاکله نیست: %+v", parts)
	}
	if _, ok := found["SPECIAL"]; ok {
		t.Fatalf("جزء صفر نباید در شاکله بیاید")
	}
	if got := wages.Total(); math.Abs(got-60_000_000) > 1e-6 {
		t.Fatalf("مزد مبنا = %v، انتظار ۶۰٬۰۰۰٬۰۰۰", got)
	}
}

// TestWageComponentsFromVars اولویت مقدار ریالی صریح بر درصد سازمانی را می‌سنجد.
func TestWageComponentsFromVars(t *testing.T) {
	vars := map[string]float64{
		"wage.job":          100_000_000,
		"allowance.special": 0.30,
		"wage.special":      25_000_000, // عدد قطعی حکم/قرارداد بر درصد مقدم است
	}
	wages := WageComponentsFromVars(vars)
	if math.Abs(wages.Special-25_000_000) > 1e-6 {
		t.Fatalf("مقدار ریالی صریح اعمال نشد: %v", wages.Special)
	}
	if math.Abs(wages.JobConstants()-25_000_000) > 1e-6 {
		t.Fatalf("ثابت‌های شغلی = %v", wages.JobConstants())
	}
}

// TestClimateRegionsAndPrecedence مناطق اقلیمی و ضریب پیشنهادی‌شان را می‌سنجد.
func TestClimateRegionsAndPrecedence(t *testing.T) {
	south := ClimateAllowancesFor("south_coast")
	if south.BadClimate != 0.25 || south.RegionHardship != 0.30 || south.RegionFa == "" {
		t.Fatalf("ضریب منطقهٔ جنوب درست نیست: %+v", south)
	}
	central := ClimateAllowancesFor("central")
	if central.BadClimate != 0 || central.Deprivation != 0 {
		t.Fatalf("منطقهٔ مرکزی نباید فوق‌العادهٔ اقلیمی داشته باشد: %+v", central)
	}
	if _, ok := ClimateRegionByKey("unknown-region"); ok {
		t.Fatal("منطقهٔ ناشناخته نباید پیدا شود")
	}
	vars := south.Vars()
	if vars["allowance.badClimate"] != 0.25 || vars["allowance.extraTime"] != 0.15 {
		t.Fatalf("کلیدهای ضرایب اقلیمی درست ساخته نشد: %+v", vars)
	}
}
