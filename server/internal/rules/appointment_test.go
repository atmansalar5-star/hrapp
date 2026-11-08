package rules

import "testing"

// baseInput ورودی پایهٔ آزمون: ترفیع یک‌پله‌ای با همهٔ شرط‌ها.
func baseInput() AppointmentInput {
	return AppointmentInput{
		Kind:               "promotion",
		CurrentRank:        2,
		ProposedRank:       3,
		ServiceMonths:      30,
		EvaluationScore:    88,
		EvaluationRecorded: true,
		VacancyAvailable:   true,
		VacancyChecked:     true,
		CommitmentMonths:   24,
		EffectiveDate:      "1405/09/01",
	}
}

// checkOf یک شرط را از خروجی پیدا می‌کند.
func checkOf(plan ApprovalPlan, key string) (AppointmentCheck, bool) {
	for _, c := range plan.Checks {
		if c.Key == key {
			return c, true
		}
	}
	return AppointmentCheck{}, false
}

// stepKeys کلیدهای زنجیرهٔ تصویب را به ترتیب برمی‌گرداند.
func stepKeys(plan ApprovalPlan) string {
	out := ""
	for i, s := range plan.Approvals {
		if i > 0 {
			out += ","
		}
		out += s.Key
	}
	return out
}

func TestPromotionFullyApproved(t *testing.T) {
	plan := PlanApproval(baseInput(), DefaultApprovalParams())
	if !plan.Allowed || plan.BlockingCount != 0 {
		t.Fatalf("ترفیع کامل باید مجاز باشد: allowed=%v blocking=%d", plan.Allowed, plan.BlockingCount)
	}
	if plan.GradeJump != 1 || plan.KindFa != "ترفیع" {
		t.Fatalf("جهش/عنوان نوع درست نیست: jump=%d kindFa=%s", plan.GradeJump, plan.KindFa)
	}
	// مدیر واحد + مدیر منابع انسانی + کمیته (چون ترفیع است)؛ پلهٔ ۳ زیر سطح مدیرعامل است.
	if got := stepKeys(plan); got != "lineManager,hrManager,committee" {
		t.Fatalf("زنجیرهٔ تصویب ترفیع نادرست است: %s", got)
	}
	if plan.CurrentGrade != "کارشناس" || plan.ProposedGrade != "کارشناسی ارشد" {
		t.Fatalf("عنوان گروه‌ها نادرست است: %s → %s", plan.CurrentGrade, plan.ProposedGrade)
	}
}

func TestPromotionBlocksOnServiceMonths(t *testing.T) {
	in := baseInput()
	in.ServiceMonths = 12
	plan := PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed || plan.BlockingCount != 1 {
		t.Fatalf("سابقهٔ کم باید یک مانع بسازد: allowed=%v blocking=%d", plan.Allowed, plan.BlockingCount)
	}
	c, ok := checkOf(plan, "serviceMonths")
	if !ok || c.Passed || !c.Blocking {
		t.Fatalf("شرط سابقه باید مانع باشد: %+v", c)
	}
}

func TestPromotionBlocksOnMissingEvaluation(t *testing.T) {
	in := baseInput()
	in.EvaluationRecorded = false
	plan := PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed {
		t.Fatal("ترفیع بدون ارزیابی ثبت‌شده نباید تصویب شود")
	}
	c, _ := checkOf(plan, "evaluation")
	if c.Passed {
		t.Fatalf("شرط ارزیابی باید رد شود: %+v", c)
	}
}

func TestPromotionBlocksOnGradeJumpAndCommitment(t *testing.T) {
	in := baseInput()
	in.ProposedRank = 4 // جهش دو پله و ورود به سطح کمیته
	in.CommitmentMonths = 0
	plan := PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed || plan.BlockingCount < 2 {
		t.Fatalf("جهش دو پله و تعهد صفر باید دو مانع بسازد: blocking=%d", plan.BlockingCount)
	}
	if c, _ := checkOf(plan, "gradeStep"); c.Passed {
		t.Fatal("شرط اندازهٔ جهش باید رد شود")
	}
	if c, _ := checkOf(plan, "commitment"); c.Passed {
		t.Fatal("شرط تعهد خدمت باید رد شود")
	}
}

func TestPromotionToManagerNeedsCEO(t *testing.T) {
	in := baseInput()
	in.CurrentRank = 4
	in.ProposedRank = 5
	in.CommitmentMonths = 36
	plan := PlanApproval(in, DefaultApprovalParams())
	if !plan.Allowed {
		t.Fatalf("ترفیع به مدیریت با تعهد ۳۶ ماه باید مجاز باشد: blocking=%d", plan.BlockingCount)
	}
	if got := stepKeys(plan); got != "lineManager,hrManager,committee,ceo" {
		t.Fatalf("زنجیرهٔ تصویب مدیریتی نادرست است: %s", got)
	}
}

func TestAppointmentToLowerGradeIsBlocked(t *testing.T) {
	in := baseInput()
	in.Kind = "appointment"
	in.CurrentRank = 4
	in.ProposedRank = 3
	plan := PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed {
		t.Fatal("انتصاب به گروه پایین‌تر نباید مجاز باشد")
	}
	if c, _ := checkOf(plan, "direction"); c.Passed {
		t.Fatal("شرط جهت گروه باید رد شود")
	}
}

func TestTransferToLowerGradeIsAllowed(t *testing.T) {
	in := baseInput()
	in.Kind = "transfer"
	in.CurrentRank = 4
	in.ProposedRank = 3
	plan := PlanApproval(in, DefaultApprovalParams())
	if !plan.Allowed {
		t.Fatalf("جابه‌جایی به گروه پایین‌تر مانع ندارد: blocking=%d", plan.BlockingCount)
	}
	if plan.KindFa != "جابه‌جایی" {
		t.Fatalf("عنوان نوع درست نیست: %s", plan.KindFa)
	}
}

func TestMissingEffectiveDateBlocks(t *testing.T) {
	in := baseInput()
	in.EffectiveDate = ""
	plan := PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed {
		t.Fatal("درخواست بدون تاریخ اجرا نباید تصویب شود")
	}
	if c, _ := checkOf(plan, "effectiveDate"); c.Passed {
		t.Fatal("شرط تاریخ اجرا باید رد شود")
	}
}

func TestVacancyUncheckedIsWarningOnly(t *testing.T) {
	in := baseInput()
	in.VacancyChecked = false
	plan := PlanApproval(in, DefaultApprovalParams())
	if !plan.Allowed {
		t.Fatalf("بررسی‌نشدن ظرفیت باید هشدار باشد نه مانع: blocking=%d", plan.BlockingCount)
	}
	// دو هشدار: ظرفیت بررسی‌نشده و شرایط احراز شغل که شناسنامه‌اش انتخاب نشده است. هر دو
	// مانع نیستند، ولی تصمیم‌گیرنده باید بداند مبنای سنجش کامل نبوده است.
	if plan.WarningCount != 2 {
		t.Fatalf("باید دو هشدار باشد: %d", plan.WarningCount)
	}
	if c, _ := checkOf(plan, "qualification"); c.Passed || c.Blocking {
		t.Fatalf("سنجش‌نشدن شرایط احراز باید هشدار غیرمانع باشد: %+v", c)
	}
}

func TestVacancyFullBlocks(t *testing.T) {
	in := baseInput()
	in.VacancyAvailable = false
	plan := PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed {
		t.Fatal("پست پر نباید تصویب شود")
	}
	if c, _ := checkOf(plan, "vacancy"); c.Passed || !c.Blocking {
		t.Fatalf("شرط ظرفیت باید مانع باشد: %+v", c)
	}
}

func TestCommitteeNotRequiredWhenParamDisabled(t *testing.T) {
	params := DefaultApprovalParams()
	params.CommitteeForPromotion = false
	plan := PlanApproval(baseInput(), params)
	if got := stepKeys(plan); got != "lineManager,hrManager" {
		t.Fatalf("با خاموش‌بودن کمیته، زنجیره باید دو سطح باشد: %s", got)
	}
}

func TestGradeLadderTitles(t *testing.T) {
	if GradeTitleFa(1) != "کارمند" || GradeTitleFa(6) != "مدیریت ارشد" {
		t.Fatalf("عنوان پله‌های نردبان نادرست است: %s / %s", GradeTitleFa(1), GradeTitleFa(6))
	}
	if GradeRankOf("supervisor") != 4 || GradeRankOf("ناموجود") != 0 {
		t.Fatalf("رتبهٔ کلید گروه نادرست است: %d / %d", GradeRankOf("supervisor"), GradeRankOf("ناموجود"))
	}
	if EvaluationPercent(150) != 100 || EvaluationPercent(-5) != 0 {
		t.Fatal("درصد امتیاز ارزیابی باید در بازهٔ ۰ تا ۱۰۰ بماند")
	}
}

// ---------- شرایط احراز شغل در تصویب‌خواهی ----------

// qualifiedInput ورودی ترفیع با شرط شغل از شناسنامه و متقاضی واجد شرایط (کارشناسی مدیریت،
// ۸ سال سابقه) در برابر شرط «کارشناسی، مدیریت، ۵ سال».
func qualifiedInput() AppointmentInput {
	in := baseInput()
	in.JobCode = "SAL-SUP-01"
	in.RequireDegreeLevel = "bachelor"
	in.RequireFields = "مدیریت، حسابداری"
	in.RequireMinExperienceYears = 5
	in.CandidateDegreeLevel = "bachelor"
	in.CandidateField = "مدیریت بازرگانی"
	in.CandidateExperienceYears = 8
	return in
}

// TestQualificationPassesWithFullConditions متقاضی واجد شرایط نباید مانع بگیرد.
func TestQualificationPassesWithFullConditions(t *testing.T) {
	plan := PlanApproval(qualifiedInput(), DefaultApprovalParams())
	if !plan.Allowed {
		t.Fatalf("درخواست با شرایط احراز کامل باید مجاز باشد: %d مانع", plan.BlockingCount)
	}
	c, found := checkOf(plan, "qualification")
	if !found || !c.Passed || c.Blocking {
		t.Fatalf("شرط شرایط احراز باید بگذرد: %+v", c)
	}
	if plan.Qualification == nil || !plan.Qualification.Eligible {
		t.Fatal("نتیجهٔ سنجش شرایط احراز باید در خروجی باشد")
	}
}

// TestQualificationShortfallBlocks کمبود تحصیلات مانع تصویب است.
func TestQualificationShortfallBlocks(t *testing.T) {
	in := qualifiedInput()
	in.RequireDegreeLevel = "master" // متقاضی کارشناسی است و سابقه‌اش کم است
	in.CandidateExperienceYears = 4
	in.RequireMinExperienceYears = 3
	plan := PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed {
		t.Fatal("کمبود جبران‌نشدهٔ شرایط احراز باید مانع تصویب باشد")
	}
	c, _ := checkOf(plan, "qualification")
	if c.Passed || !c.Blocking {
		t.Fatalf("شرط شرایط احراز باید مانع باشد: %+v", c)
	}
	if plan.Qualification == nil || plan.Qualification.Eligible {
		t.Fatal("نتیجهٔ سنجش باید «برآورده نیست» باشد")
	}
}

// TestQualificationEquivalentAllows شرایط معادل با سابقهٔ مازاد مانع نیست و ثبت می‌شود.
func TestQualificationEquivalentAllows(t *testing.T) {
	in := qualifiedInput()
	in.RequireDegreeLevel = "master" // یک سطح کمبود
	in.RequireMinExperienceYears = 3 // سابقهٔ مازاد ۵ سال ≥ ۲ سال لازم
	in.CandidateDegreeLevel = "bachelor"
	in.CandidateExperienceYears = 8
	plan := PlanApproval(in, DefaultApprovalParams())
	if !plan.Allowed {
		t.Fatalf("شرایط معادل باید تصویب را باز کند: %d مانع", plan.BlockingCount)
	}
	if c, found := checkOf(plan, "equivalence"); !found || !c.Passed {
		t.Fatalf("سطر شرایط معادل باید ثبت شود: %+v", c)
	}
	if plan.Qualification == nil || !plan.Qualification.EquivalenceApplied {
		t.Fatal("نتیجه باید «برآورده با شرایط معادل» باشد")
	}
}

// TestQualificationExcessShortensWaiting مازاد بر شرایط احراز دورهٔ انتظار را کوتاه می‌کند.
func TestQualificationExcessShortensWaiting(t *testing.T) {
	params := DefaultApprovalParams()
	params.MinServiceMonths = 24
	params.Qualification.ExcessCreditMonthsPerYear = 6
	params.Qualification.ExcessCreditMonthsPerLevel = 12
	params.Qualification.MaxExcessCreditMonths = 60

	// متقاضی کارشناسی ارشد با ۸ سال سابقه: ۱ سطح مازاد (۱۲ ماه) + ۳ سال مازاد (۱۸ ماه)
	// = ۳۰ ماه تخفیف؛ پس ۲۴ ماه لازم به صفر می‌رسد و ۱۰ ماه سابقه کافی است.
	in := qualifiedInput()
	in.CandidateDegreeLevel = "master"
	in.ServiceMonths = 10
	plan := PlanApproval(in, params)
	if c, _ := checkOf(plan, "serviceMonths"); !c.Passed {
		t.Fatalf("با تخفیف مازاد، سابقهٔ ۱۰ ماه باید کافی باشد: %+v", c)
	}
	if plan.Qualification == nil || plan.Qualification.ExcessCreditMonths < 24 {
		t.Fatalf("تخفیف مازاد باید دست‌کم ۲۴ ماه باشد: %+v", plan.Qualification)
	}

	// بی مازاد مدرک و بی سابقهٔ مازاد، همان درخواست باید مانع بگیرد.
	plain := qualifiedInput()
	plain.ServiceMonths = 10
	plain.CandidateDegreeLevel = "bachelor"
	plain.CandidateExperienceYears = 5 // برابر کمینهٔ شرط: سابقهٔ مازادی نیست
	if p := PlanApproval(plain, params); p.Allowed {
		t.Fatal("بدون تخفیف مازاد، سابقهٔ ۱۰ ماه برای شرط ۲۴ ماه کافی نیست")
	}
}

// TestQualificationUnknownRequirementBlocks شرط ناشناخته یا بی‌مقطع متقاضی مانع است.
func TestQualificationUnknownRequirementBlocks(t *testing.T) {
	in := qualifiedInput()
	in.RequireDegreeLevel = "ناموجود"
	plan := PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed {
		t.Fatal("شرط شغل ناشناخته باید مانع باشد")
	}

	in = qualifiedInput()
	in.CandidateDegreeLevel = ""
	plan = PlanApproval(in, DefaultApprovalParams())
	if plan.Allowed {
		t.Fatal("نبود مقطع متقاضی باید مانع سنجش باشد")
	}
}
