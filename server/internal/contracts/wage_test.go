package contracts

import (
	"math"
	"testing"
)

// TestWageStructureComponentsAndMazdBase می‌سنجد اجزای ثبت‌شدهٔ سند کارگزینی با ثابت‌های
// سازمانی جمع می‌شوند و «مزد مبنا» از همان جمع ساخته می‌شود.
func TestWageStructureComponentsAndMazdBase(t *testing.T) {
	w := WageStructure{
		ClimateRegion:     "south_coast",
		RankWage:          20_000_000,
		SeniorityWage:     10_000_000,
		PostAllowance:     5_000_000,
		ClimateAllowance:  6_000_000,
		BadClimatePercent: 0.30,
	}
	company := map[string]float64{
		"allowance.special":    0.30,
		"allowance.workshop":   0.10,
		"allowance.badClimate": 0.25,
	}
	wages := w.Components(100_000_000, company)
	// مزد شغل و ملحقات ۱۳۰٬۰۰۰٬۰۰۰؛ ویژه ۳۹٬۰۰۰٬۰۰۰؛ کارگاهی ۱۳٬۰۰۰٬۰۰۰؛ پست ۵٬۰۰۰٬۰۰۰؛
	// اقلیمی مصوب ۶٬۰۰۰٬۰۰۰ → مزد مبنا ۱۹۳٬۰۰۰٬۰۰۰.
	if got, want := wages.Total(), 193_000_000.0; math.Abs(got-want) > 1e-6 {
		t.Fatalf("مزد مبنا = %v، انتظار %v", got, want)
	}
	if got := w.MazdBase(100_000_000, company); got != 193_000_000 {
		t.Fatalf("مزد مبنا گرد‌شده = %d", got)
	}

	// ضریب اقلیمی خود کارمند (۳۰٪) بر پارامتر شرکت (۲۵٪) مقدم است و ضریب ثبت‌نشده از
	// پارامتر/منطقه پر می‌شود.
	climate := w.Climate(company)
	if math.Abs(climate.BadClimate-0.30) > 1e-9 {
		t.Fatalf("ضریب بدی آب و هوا = %v، انتظار ۰٫۳۰ (کارمند بر شرکت مقدم است)", climate.BadClimate)
	}
	if math.Abs(climate.Deprivation-0.25) > 1e-9 {
		t.Fatalf("ضریب محرومیت = %v، انتظار ۰٫۲۵ از پارامتر شرکت", climate.Deprivation)
	}
	if math.Abs(climate.RegionHardship-0.30) > 1e-9 {
		t.Fatalf("ضریب صعوبت = %v، انتظار ۰٫۳۰ از منطقهٔ اقلیمی", climate.RegionHardship)
	}
	if climate.RegionFa == "" {
		t.Fatal("عنوان منطقهٔ اقلیمی باید پر شود")
	}
}

// TestWageStructureClimateFallbackByRegion در نبود پارامتر شرکت، منطقهٔ اقلیمی مبنا است.
func TestWageStructureClimateFallbackByRegion(t *testing.T) {
	w := WageStructure{ClimateRegion: "desert"}
	climate := w.Climate(map[string]float64{})
	if math.Abs(climate.BadClimate-0.20) > 1e-9 || math.Abs(climate.RegionHardship-0.25) > 1e-9 {
		t.Fatalf("ضرایب منطقهٔ بیابانی اعمال نشد: %+v", climate)
	}
	if !w.HasClimatePercent() && w.BadClimatePercent != 0 {
		t.Fatal("HasClimatePercent باید ضریب ثبت‌شده را تشخیص دهد")
	}
}

// TestWageStructureApplyOverride مقدار صفر را «بدون تغییر» می‌گیرد.
func TestWageStructureApplyOverride(t *testing.T) {
	base := WageStructure{
		ClimateRegion: "central", RankWage: 10_000_000, PostAllowance: 4_000_000,
		BadClimatePercent: 0.20,
	}
	base.ApplyOverride(WageStructure{RankWage: 12_000_000, ClimateRegion: "south_coast"})
	if base.RankWage != 12_000_000 {
		t.Fatalf("مقدار داده‌شده اعمال نشد: %d", base.RankWage)
	}
	if base.PostAllowance != 4_000_000 {
		t.Fatalf("مقدار صفر نباید مقدار قبلی را پاک کند: %d", base.PostAllowance)
	}
	if base.BadClimatePercent != 0.20 {
		t.Fatalf("ضریب ثبت‌نشده نباید صفر شود: %v", base.BadClimatePercent)
	}
	if base.ClimateRegion != "south_coast" {
		t.Fatalf("منطقهٔ اقلیمی تازه اعمال نشد: %s", base.ClimateRegion)
	}
}

// TestSalaryDecreeWithWageView مزد مبنا را در شکل انتقالی حکم می‌سنجد.
func TestSalaryDecreeWithWageView(t *testing.T) {
	d := SalaryDecree{
		ID: "DEC-TEST-1", BaseSalary: 100_000_000,
		WageStructure: WageStructure{ClimateRegion: "south_coast", RankWage: 20_000_000},
	}
	view := d.WithWageView(map[string]float64{"allowance.special": 0.30})
	// مزد شغل ۱۰۰٬۰۰۰٬۰۰۰ + رتبه ۲۰٬۰۰۰٬۰۰۰ = ۱۲۰٬۰۰۰٬۰۰۰؛ ویژه ۳۰٪ = ۳۶٬۰۰۰٬۰۰۰ → ۱۵۶٬۰۰۰٬۰۰۰.
	if view.MazdBase != 156_000_000 {
		t.Fatalf("مزد مبنای حکم = %d، انتظار ۱۵۶٬۰۰۰٬۰۰۰", view.MazdBase)
	}
	if view.ClimateRegionFa == "" {
		t.Fatal("عنوان منطقهٔ اقلیمی باید در پاسخ بیاید")
	}
}
