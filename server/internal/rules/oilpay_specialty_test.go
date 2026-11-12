package rules

import "testing"

// oilSpecialtyBase فیش پایه برای سنجش افزایش ضریب تخصصی می‌سازد.
func oilSpecialtyBase() OilPayInput {
	return OilPayInput{
		BaseSalary:  180_000_000,
		Degree:      "bachelor",
		JobGrade:    "A",
		DaysInMonth: 30,
	}
}

// specialLine سطر «فوقالعادهٔ تخصصی شغل» را از نتیجه بیرون می‌کشد.
func specialLine(res OilPayslip) (OilPayLine, bool) {
	for _, line := range res.AllowanceLines {
		if line.Key == OilAllowanceAttraction {
			return line, true
		}
	}
	return OilPayLine{}, false
}

// TestOilSpecialtyExtraIsAdditive افزایش ضریب تخصصی شغل باید روی جمع Y+Z بنشیند، نه
// جایگزین آن؛ یعنی درصد سطر تخصصی به اندازهٔ همین فیلد بیشتر شود.
func TestOilSpecialtyExtraIsAdditive(t *testing.T) {
	params := DefaultOilPayParams()

	base := ComputeOilPayslip(oilSpecialtyBase(), params)
	baseLine, ok := specialLine(base)
	if !ok {
		t.Fatal("سطر فوقالعادهٔ تخصصی شغل در فیش نیست")
	}
	if base.SpecialtyExtraPercent != 0 {
		t.Fatalf("بدون درخواست افزایش، درصد افزودنی باید صفر باشد: %v", base.SpecialtyExtraPercent)
	}

	withExtra := oilSpecialtyBase()
	withExtra.SpecialtyExtraPercent = 12
	res := ComputeOilPayslip(withExtra, params)
	extraLine, ok := specialLine(res)
	if !ok {
		t.Fatal("سطر فوقالعادهٔ تخصصی شغل در فیش نیست")
	}
	if got := extraLine.Percent - baseLine.Percent; got < 11.99 || got > 12.01 {
		t.Fatalf("افزایش باید ۱۲ واحد درصد باشد، شد: %v", got)
	}
	if res.SpecialtyPercent != extraLine.Percent {
		t.Fatalf("درصد کل تخصصی باید همان درصد سطر باشد: %v در برابر %v", res.SpecialtyPercent, extraLine.Percent)
	}
	if extraLine.Amount <= baseLine.Amount {
		t.Fatalf("مبلغ فوقالعادهٔ تخصصی باید با افزایش ضریب بیشتر شود: %d در برابر %d", extraLine.Amount, baseLine.Amount)
	}
	if res.SpecialtyExtraPercent != 12 {
		t.Fatalf("درصد افزایش اعمال‌شده باید ۱۲ باشد: %v", res.SpecialtyExtraPercent)
	}
}

// TestOilSpecialtyExtraIsCappedByParameter سقف پارامتر شرکت روی افزایش حاکم است.
func TestOilSpecialtyExtraIsCappedByParameter(t *testing.T) {
	params := DefaultOilPayParams()
	params.SpecialtyMaxExtraPercent = 20

	in := oilSpecialtyBase()
	in.SpecialtyExtraPercent = 65
	res := ComputeOilPayslip(in, params)
	if !res.SpecialtyExtraCapped {
		t.Fatal("درخواست بیش از سقف باید «محدودشده» علامت بخورد")
	}
	if res.SpecialtyExtraPercent != 20 {
		t.Fatalf("افزایش باید به سقف پارامتر (۲۰) محدود شود: %v", res.SpecialtyExtraPercent)
	}

	// پارامتر صفر یعنی سازمان افزودن ضریب را مجاز نمی‌داند.
	params.SpecialtyMaxExtraPercent = 0
	zero := ComputeOilPayslip(in, params)
	if zero.SpecialtyExtraPercent != 0 || !zero.SpecialtyExtraCapped {
		t.Fatalf("با سقف صفر نباید افزایشی اعمال شود: %v/%v", zero.SpecialtyExtraPercent, zero.SpecialtyExtraCapped)
	}

	// مقدار منفی اثر ندارد (کاهش ضریب تخصصی از این فیلد مجاز نیست).
	in.SpecialtyExtraPercent = -5
	params.SpecialtyMaxExtraPercent = 50
	neg := ComputeOilPayslip(in, params)
	if neg.SpecialtyExtraPercent != 0 {
		t.Fatalf("مقدار منفی باید صفر شود: %v", neg.SpecialtyExtraPercent)
	}
}

// TestOilSpecialtyExtraAppliesOnManualPercent افزایش، روی درصد دستی هم اضافه می‌شود؛
// چون هزینهٔ تخصصی همیشه «افزودنی» است و جایگزین انتخاب‌های فرم نمی‌شود.
func TestOilSpecialtyExtraAppliesOnManualPercent(t *testing.T) {
	params := DefaultOilPayParams()
	in := oilSpecialtyBase()
	in.AllowancePercents = map[string]float64{OilAllowanceAttraction: 40}
	manual := ComputeOilPayslip(in, params)
	manualLine, _ := specialLine(manual)
	if manualLine.Percent < 39.99 || manualLine.Percent > 40.01 {
		t.Fatalf("درصد دستی باید همان ۴۰ بماند: %v", manualLine.Percent)
	}

	in.SpecialtyExtraPercent = 5
	both := ComputeOilPayslip(in, params)
	bothLine, _ := specialLine(both)
	if got := bothLine.Percent - manualLine.Percent; got < 4.99 || got > 5.01 {
		t.Fatalf("افزایش باید روی درصد دستی هم ۵ واحد باشد: %v", got)
	}
	if bothLine.TitleFa == "" {
		t.Fatal("عنوان فارسی قلم تخصصی برای سند چاپی خالی است")
	}
}
