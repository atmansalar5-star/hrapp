package domain

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// committeeRecord ورودی مصوبهٔ کمیته می‌سازد.
func committeeRecord(fields map[string]string) contracts.RecordInput {
	return contracts.RecordInput{Kind: KindVeteranCommittee, Title: "مصوبهٔ آزمایشی", Fields: fields}
}

// validCommittee بندهای کامل و درست یک مصوبه.
func validCommittee() map[string]string {
	return map[string]string{
		"meetingDate":             "1405/08/12",
		"veteranGroup":            rules.VeteranGroupDisabled,
		"decisionFa":              "امتیازها به شرح بندها برقرار است.",
		"eduUpgradeDecision":      "yes",
		"taxExemptDecision":       "yes",
		"taxExemptPercent":        "100",
		"pensionExemptDecision":   "no",
		"pensionExemptPercent":    "0",
		"workHourReduction":       "yes",
		"reducedWeeklyHours":      "36",
		"incentiveSteps":          "2",
		"seniorityMonths":         "54",
		"seniorityBasis":          rules.VeteranSeniorityErafaqi,
		"leaveDays":               "30",
		"employmentQuotaDecision": "yes",
		"effectiveDate":           "1405/09/01",
	}
}

// TestVeteranCommitteeValid موبهٔ کامل باید بدون خطا پذیرفته شود.
func TestVeteranCommitteeValid(t *testing.T) {
	if err := validateVeteranCommittee(committeeRecord(validCommittee())); err != nil {
		t.Fatalf("مصوبهٔ کامل نباید خطا بدهد: %v", err)
	}
}

// TestVeteranCommitteeValidationRules هر بند ناقص یا بیرون بازه باید رد شود.
func TestVeteranCommitteeValidationRules(t *testing.T) {
	cases := []struct {
		name   string
		change func(map[string]string)
		want   string
	}{
		{"گروه نامعلوم", func(f map[string]string) { f["veteranGroup"] = "" }, "گروه ایثارگری"},
		{"بند بله/خیر نامعتبر", func(f map[string]string) { f["taxExemptDecision"] = "بله" }, "معافیت مالیاتی"},
		{"درصد مالیات بیش از صد", func(f map[string]string) { f["taxExemptPercent"] = "150" }, "معافیت مالیاتی"},
		{"درصد کسور منفی", func(f map[string]string) { f["pensionExemptPercent"] = "-5" }, "کسور بازنشستگی"},
		{"پایهٔ تشویقی بیش از سقف", func(f map[string]string) { f["incentiveSteps"] = "3" }, "پایهٔ تشویقی"},
		{"افزایش سنواتی بیش از سقف", func(f map[string]string) { f["seniorityMonths"] = "200" }, "افزایش سنواتی"},
		{"مرخصی بیش از یک سال", func(f map[string]string) { f["leaveDays"] = "400" }, "مرخصی تشویقی"},
		{"کاهش ساعت بدون ساعت مصوب", func(f map[string]string) { f["reducedWeeklyHours"] = "0" }, "ساعت کار هفتگی"},
		{"ساعت بیشتر از هفتهٔ کامل", func(f map[string]string) { f["reducedWeeklyHours"] = "50" }, "ساعت کار هفتگی"},
		{"مبنای سنواتی ناشناخته", func(f map[string]string) { f["seniorityBasis"] = "دلخواه" }, "مبنای افزایش سنواتی"},
		{"تاریخ جلسه نامعتبر", func(f map[string]string) { f["meetingDate"] = "1405/13/40" }, "تاریخ"},
		{"بدون متن تصمیم", func(f map[string]string) { f["decisionFa"] = " " }, "خلاصهٔ تصمیم"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := validCommittee()
			tc.change(fields)
			err := validateVeteranCommittee(committeeRecord(fields))
			if err == nil {
				t.Fatalf("مصوبهٔ «%s» باید رد شود", tc.name)
			}
			ruleErr, ok := err.(*recordkit.RuleError)
			if !ok {
				t.Fatalf("نوع خطا باید RuleError باشد: %T", err)
			}
			if !strings.Contains(ruleErr.Message, tc.want) {
				t.Fatalf("پیام %q باید شامل %q باشد", ruleErr.Message, tc.want)
			}
		})
	}
}

// TestVeteranCommitteeWorkflowAfterSignature مصوبه پس از امضا قابل تصویب است و پیش از آن نه.
func TestVeteranCommitteeWorkflowAfterSignature(t *testing.T) {
	kind, ok := Descriptor().KindByKey(KindVeteranCommittee)
	if !ok {
		t.Fatal("نوع رکورد مصوبهٔ کمیته در توصیفگر نیست")
	}
	approve, ok := kind.ActionByValue("approve")
	if !ok {
		t.Fatal("اقدام «تصویب و اجرا» در گردشکار نیست")
	}
	if kind.CanTransition("draft", approve) {
		t.Fatal("مصوبهٔ پیش‌نویس نباید یک‌مرحله‌ای تصویب شود؛ ابتدا امضای اعضا لازم است")
	}
	if !kind.CanTransition("signed", approve) {
		t.Fatal("مصوبهٔ امضاشده باید قابل تصویب باشد")
	}
	if got := strings.Join(kind.EditableStatuses, ","); got != "draft,signed" {
		t.Fatalf("وضعیت‌های ویرایش‌پذیر مصوبه نادرست است: %q", got)
	}
}

// TestVeteranCommitteeSchemaHasDecisionFields بندهای تصمیم باید در طرح رکورد به کلاینت برسند.
func TestVeteranCommitteeSchemaHasDecisionFields(t *testing.T) {
	schema := Descriptor().Schema()
	keys := map[string]bool{}
	for _, kind := range schema.Kinds {
		if kind.Key != KindVeteranCommittee {
			continue
		}
		for _, field := range kind.Fields {
			keys[field.Key] = true
		}
	}
	for _, want := range []string{
		"meetingDate", "veteranGroup", "basisDocs",
		"eduUpgradeDecision", "taxExemptDecision", "taxExemptPercent",
		"pensionExemptDecision", "pensionExemptPercent",
		"workHourReduction", "reducedWeeklyHours",
		"incentiveSteps", "seniorityMonths", "seniorityBasis", "leaveDays",
		"employmentQuotaDecision", "decisionFa", "effectiveDate", "signOff",
	} {
		if !keys[want] {
			t.Fatalf("فیلد %q در طرح مصوبهٔ کمیته نیست", want)
		}
	}
}
