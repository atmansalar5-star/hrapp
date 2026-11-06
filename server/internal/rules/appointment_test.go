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
	if plan.WarningCount != 1 {
		t.Fatalf("باید یک هشدار باشد: %d", plan.WarningCount)
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
