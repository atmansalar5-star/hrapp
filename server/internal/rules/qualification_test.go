// آزمون قاعدهٔ شرایط احراز، شرایط معادل، کمبود تحصیلات و مازاد.

package rules

import "testing"

// requirementBachelorManagement شرط احراز نمونه: کارشناسی مدیریت و ۵ سال سابقه.
func requirementBachelorManagement() JobRequirement {
	return JobRequirement{
		DegreeLevel:        "bachelor",
		Fields:             []string{"مدیریت", "حسابداری", "مهندسی صنایع"},
		MinExperienceYears: 5,
	}
}

// TestQualificationDirectPass شرط‌های کامل باید بی‌واسطه برآورده شود.
func TestQualificationDirectPass(t *testing.T) {
	res := EvaluateQualification(Candidate{
		DegreeLevel: "bachelor", Field: "مدیریت بازرگانی", ExperienceYears: 6,
	}, requirementBachelorManagement(), DefaultQualificationParams())

	if !res.Eligible || res.Verdict != QualificationDirect {
		t.Fatalf("نتیجه باید «برآورده» باشد، شد %q (%s)", res.Verdict, res.SummaryFa)
	}
	if res.DegreeGap != 0 || res.RemainingDeficitLevels != 0 {
		t.Fatalf("کمبود مدرک نباید داشته باشد: gap=%d remaining=%d", res.DegreeGap, res.RemainingDeficitLevels)
	}
	if !res.FieldMatched {
		t.Fatal("رشتهٔ مدیریت بازرگانی باید با رشتهٔ مدیریت بخواند")
	}
	if res.BlockingCount != 0 {
		t.Fatalf("مانعی نباید ثبت شود: %d", res.BlockingCount)
	}
	// مازاد سابقه (۱ سال) و مدرک برابر: تنها ۳ ماه تخفیف از سابقهٔ مازاد.
	if res.ExcessCreditMonths != 3 {
		t.Fatalf("تخفیف دورهٔ انتظار باید ۳ ماه باشد، شد %d", res.ExcessCreditMonths)
	}
}

// TestQualificationEquivalentExperience شرط مدرک با سابقهٔ مازاد جبران می‌شود.
func TestQualificationEquivalentExperience(t *testing.T) {
	// کاردانی (یک سطح کمبود) با ۸ سال سابقه: سابقهٔ مازاد ۳ سال و لازم ۲ سال.
	res := EvaluateQualification(Candidate{
		DegreeLevel: "associate", Field: "مدیریت", ExperienceYears: 8,
	}, requirementBachelorManagement(), DefaultQualificationParams())

	if res.DegreeGap != 1 {
		t.Fatalf("کمبود مدرک باید یک سطح باشد: %d", res.DegreeGap)
	}
	if res.RequiredEquivalentYears != 2 {
		t.Fatalf("سابقهٔ لازم برای جبران باید ۲ سال باشد: %d", res.RequiredEquivalentYears)
	}
	if !res.EquivalenceApplied || res.Verdict != QualificationEquivalent || !res.Eligible {
		t.Fatalf("باید با شرایط معادل برآورده شود: verdict=%q eligible=%v", res.Verdict, res.Eligible)
	}
	if res.CompensatedLevels != 1 {
		t.Fatalf("یک سطح باید جبران شود: %d", res.CompensatedLevels)
	}
}

// TestQualificationShortfall وقتی سابقهٔ مازاد کافی نیست، کمبود تحصیلات ثبت می‌شود.
func TestQualificationShortfall(t *testing.T) {
	// کارشناسی ارشد لازم فرض می‌شود؛ کاردانی با ۶ سال سابقه تنها ۱ سال مازاد دارد.
	req := JobRequirement{DegreeLevel: "master", Fields: []string{"مدیریت"}, MinExperienceYears: 5}
	res := EvaluateQualification(Candidate{
		DegreeLevel: "associate", Field: "مدیریت", ExperienceYears: 6,
	}, req, DefaultQualificationParams())

	if res.Eligible || res.Verdict != QualificationShortfall {
		t.Fatalf("نتیجه باید کمبود باشد: verdict=%q eligible=%v", res.Verdict, res.Eligible)
	}
	if res.DegreeGap != 2 {
		t.Fatalf("کمبود مدرک باید ۲ سطح باشد: %d", res.DegreeGap)
	}
	// ۲ سطح × ۲ سال = ۴ سال لازم؛ ۱ سال مازاد کمتر از سابقهٔ یک سطح است، پس هیچ سطحی
	// کامل جبران نشده و هر دو سطح باقی می‌ماند.
	if res.RemainingDeficitLevels != 2 {
		t.Fatalf("هیچ سطحی کامل جبران نشده؛ باید ۲ سطح باقی بماند: %d", res.RemainingDeficitLevels)
	}
	if res.BlockingCount != 1 {
		t.Fatalf("کمبود باید مانع شمرده شود: %d", res.BlockingCount)
	}

	// با ۷ سال سابقه (۲ سال مازاد) دقیقاً یک سطح کامل جبران می‌شود و یک سطح می‌ماند.
	partial := EvaluateQualification(Candidate{
		DegreeLevel: "associate", Field: "مدیریت", ExperienceYears: 7,
	}, req, DefaultQualificationParams())
	if partial.RemainingDeficitLevels != 1 {
		t.Fatalf("با ۲ سال سابقهٔ مازاد باید یک سطح باقی بماند: %d", partial.RemainingDeficitLevels)
	}
}

// TestQualificationFieldMismatchNeedsExtraYears رشتهٔ ناهمسو سابقهٔ اضافه می‌طلبد.
func TestQualificationFieldMismatchNeedsExtraYears(t *testing.T) {
	// کارشناسی با رشتهٔ نامرتبط و ۶ سال سابقه: ۱ سال مازاد < ۲ سال لازم برای ناهمسویی.
	short := EvaluateQualification(Candidate{
		DegreeLevel: "bachelor", Field: "حقوق", ExperienceYears: 6,
	}, requirementBachelorManagement(), DefaultQualificationParams())
	if short.Eligible {
		t.Fatal("با ۱ سال سابقهٔ مازاد، ناهمسویی رشته نباید جبران شود")
	}
	if short.FieldMatched {
		t.Fatal("رشتهٔ حقوق نباید با رشته‌های مرتبط بخواند")
	}

	// با ۸ سال سابقه، ۳ سال مازاد ≥ ۲ سال لازم → برآورده با شرایط معادل.
	ok := EvaluateQualification(Candidate{
		DegreeLevel: "bachelor", Field: "حقوق", ExperienceYears: 8,
	}, requirementBachelorManagement(), DefaultQualificationParams())
	if !ok.Eligible || ok.Verdict != QualificationEquivalent {
		t.Fatalf("با ۳ سال سابقهٔ مازاد باید با شرایط معادل برآورده شود: verdict=%q", ok.Verdict)
	}
}

// TestQualificationExperienceShortfall کمبود سابقه با سابقه جبران نمی‌شود.
func TestQualificationExperienceShortfall(t *testing.T) {
	res := EvaluateQualification(Candidate{
		DegreeLevel: "bachelor", Field: "مدیریت", ExperienceYears: 3,
	}, requirementBachelorManagement(), DefaultQualificationParams())

	if res.Eligible {
		t.Fatal("کمبود سابقه نباید برآورده شمرده شود")
	}
	if res.Verdict != QualificationShortfall {
		t.Fatalf("نتیجه باید کمبود باشد: %q", res.Verdict)
	}
}

// TestQualificationEquivalenceDisabled خاموش‌بودن پارامتر، معادل را می‌بندد.
func TestQualificationEquivalenceDisabled(t *testing.T) {
	params := DefaultQualificationParams()
	params.EquivalenceEnabled = false
	res := EvaluateQualification(Candidate{
		DegreeLevel: "associate", Field: "مدیریت", ExperienceYears: 12,
	}, requirementBachelorManagement(), params)

	if res.Eligible {
		t.Fatal("با خاموش‌بودن شرایط معادل نباید برآورده شود")
	}
	if res.EquivalenceApplied {
		t.Fatal("شرایط معادل نباید اعمال شود")
	}
}

// TestQualificationMaxCompensatedYears سقف جبران، معادل را رد می‌کند.
func TestQualificationMaxCompensatedYears(t *testing.T) {
	params := DefaultQualificationParams()
	params.MaxCompensatedYears = 1 // سقف کمتر از ۲ سال لازم برای یک سطح
	res := EvaluateQualification(Candidate{
		DegreeLevel: "associate", Field: "مدیریت", ExperienceYears: 20,
	}, requirementBachelorManagement(), params)

	if res.Eligible {
		t.Fatal("وقتی سابقهٔ لازم از سقف بیشتر است، معادل نباید پذیرفته شود")
	}
}

// TestQualificationExcessCredit مازاد مدرک تخفیف دورهٔ انتظار می‌سازد و سقف دارد.
func TestQualificationExcessCredit(t *testing.T) {
	params := DefaultQualificationParams()
	params.MaxExcessCreditMonths = 24
	// دو سطح مازاد (۲۴ ماه) + ۵ سال مازاد (۱۵ ماه) = ۳۹ → با سقف ۲۴.
	res := EvaluateQualification(Candidate{
		DegreeLevel: "phd", Field: "مدیریت", ExperienceYears: 10,
	}, requirementBachelorManagement(), params)

	if res.DegreeSurplus != 2 {
		t.Fatalf("مازاد مدرک باید ۲ سطح باشد: %d", res.DegreeSurplus)
	}
	if res.ExcessCreditMonths != 24 {
		t.Fatalf("تخفیف باید با سقف ۲۴ ماه محدود شود: %d", res.ExcessCreditMonths)
	}
	if !res.Eligible || res.Verdict != QualificationDirect {
		t.Fatalf("مازاد نباید مانع باشد: verdict=%q", res.Verdict)
	}
}

// TestQualificationParamsFromValues پارامتر شرکت روی پیش‌فرض می‌نشیند.
func TestQualificationParamsFromValues(t *testing.T) {
	p := QualificationParamsFromValues(map[string]float64{
		"qual.equivalence.yearsPerLevel": 3,
		"qual.equivalence.enabled":       0,
	})
	if p.YearsPerDegreeLevel != 3 {
		t.Fatalf("ضریب تبدیل باید ۳ باشد: %d", p.YearsPerDegreeLevel)
	}
	if p.EquivalenceEnabled {
		t.Fatal("مقدار صفر باید شرایط معادل را خاموش کند")
	}
	if p.MaxCompensatedYears != DefaultQualificationParams().MaxCompensatedYears {
		t.Fatal("کلید غایب باید پیش‌فرض بماند")
	}
}

// TestQualificationLinesCoverInstruction فرم محاسباتی باید همهٔ گام‌های دستورالعمل را
// پوشش دهد و شماره‌گذاری پیوسته داشته باشد.
func TestQualificationLinesCoverInstruction(t *testing.T) {
	res := EvaluateQualification(Candidate{
		DegreeLevel: "diploma", Field: "حسابداری", ExperienceYears: 2,
	}, requirementBachelorManagement(), DefaultQualificationParams())

	if len(res.Lines) < 5 {
		t.Fatalf("فرم محاسباتی باید دست‌کم ۵ سطر داشته باشد: %d", len(res.Lines))
	}
	for i, line := range res.Lines {
		if line.Order != i+1 {
			t.Fatalf("شمارهٔ سطر %d باید %d باشد، شد %d", i, i+1, line.Order)
		}
		if line.TitleFa == "" || line.DetailFa == "" {
			t.Fatalf("سطر %d باید عنوان و شرح داشته باشد", i+1)
		}
	}
	if steps := QualificationInstructionFa(); len(steps) < 8 {
		t.Fatalf("دستورالعمل باید دست‌کم ۸ گام داشته باشد: %d", len(steps))
	}
}

// TestFieldMatches تطبیق واژه‌ای رشته‌ها را می‌سنجد.
func TestFieldMatches(t *testing.T) {
	allowed := []string{"مدیریت", "مهندسی صنایع"}
	cases := []struct {
		field string
		want  bool
	}{
		{"مدیریت بازرگانی", true},
		{"مديريت", true}, // ي عربی
		{"مهندسی صنايع", true},
		{"حقوق", false},
		{"", false},
	}
	for _, c := range cases {
		if got := FieldMatches(c.field, allowed); got != c.want {
			t.Errorf("FieldMatches(%q) = %v، انتظار %v", c.field, got, c.want)
		}
	}
}

// TestDegreeLadderOrdered نردبان مدرک باید پیوسته و شناخته‌شده باشد.
func TestDegreeLadderOrdered(t *testing.T) {
	ladder := DegreeLevels()
	if len(ladder) < 6 {
		t.Fatalf("نردبان مدرک باید دست‌کم ۶ پله داشته باشد: %d", len(ladder))
	}
	for i, d := range ladder {
		if d.Rank != i+1 {
			t.Fatalf("رتبهٔ پلهٔ %d باید %d باشد، شد %d", i, i+1, d.Rank)
		}
		if DegreeRankOf(d.Key) != d.Rank {
			t.Fatalf("DegreeRankOf(%q) باید %d باشد", d.Key, d.Rank)
		}
	}
	if DegreeRankOf("ناموجود") != 0 {
		t.Fatal("کلید ناشناخته باید رتبهٔ صفر بدهد")
	}
}
