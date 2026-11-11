package rules

import (
	"strings"
	"testing"
)

// committeeBase پیش‌فرض قانونی یک جانباز ۲۵٪ تأییدشده (۴۸ ماه سنوات ارفاقی).
func committeeBase() VeteranEntitlement {
	return ComputeVeteranEntitlements(VeteranProfile{
		Group: VeteranGroupDisabled, DisabilityPercent: 25, ApprovedFile: true,
	}, DefaultVeteranParams())
}

// TestVeteranDecisionFromFieldsEmptyMeansLegalDefault فرم بدون بند، هیچ تصمیمی نمی‌گیرد.
func TestVeteranDecisionFromFieldsEmptyMeansLegalDefault(t *testing.T) {
	dec := VeteranDecisionFromFields(map[string]string{"veteranGroup": "disabled"})
	if dec.Group != VeteranGroupDisabled {
		t.Fatalf("گروه مصوبه خوانده نشد: %q", dec.Group)
	}
	if dec.TaxExempt != nil || dec.PensionExempt != nil || dec.EduUpgrade != nil ||
		dec.IncentiveSteps != nil || dec.SeniorityMonths != nil || dec.LeaveDays != nil ||
		dec.EmploymentQuota != nil {
		t.Fatalf("بند تعیین‌نشده باید nil بماند (پیروی از قانون): %+v", dec)
	}
	base := committeeBase()
	merged, changes := ApplyVeteranDecision(base, dec)
	if len(changes) != 0 {
		t.Fatalf("بدون مصوبه نباید تغییری ثبت شود: %v", changes)
	}
	if merged.TaxExempt != base.TaxExempt || merged.TotalExtraServiceMonths != base.TotalExtraServiceMonths ||
		merged.IncentiveSteps != base.IncentiveSteps {
		t.Fatalf("بدون مصوبه، امتیازها باید همان پیش‌فرض قانونی بمانند: %+v", merged)
	}
}

// TestVeteranDecisionFromFieldsReadsYesNoAndNumbers خواندن بندها از فرم (بله/خیر و عدد).
func TestVeteranDecisionFromFieldsReadsYesNoAndNumbers(t *testing.T) {
	dec := VeteranDecisionFromFields(map[string]string{
		"veteranGroup":            "جانباز",
		"eduUpgradeDecision":      "no",
		"taxExemptDecision":       "yes",
		"taxExemptPercent":        "۵۰",
		"pensionExemptDecision":   "yes",
		"pensionExemptPercent":    "100",
		"workHourReduction":       "yes",
		"reducedWeeklyHours":      "۳۶",
		"incentiveSteps":          "2",
		"seniorityMonths":         "54",
		"seniorityBasis":          VeteranSeniorityErafaqi,
		"leaveDays":               "30",
		"employmentQuotaDecision": "no",
	})
	if dec.Group != VeteranGroupDisabled {
		t.Fatalf("گروه از عنوان فارسی خوانده نشد: %q", dec.Group)
	}
	if dec.EduUpgrade == nil || *dec.EduUpgrade {
		t.Fatal("بند «یک مقطع بالاتر» باید «خیر» باشد")
	}
	if dec.TaxExemptPercent == nil || *dec.TaxExemptPercent != 50 {
		t.Fatalf("درصد معافیت مالیاتی با ارقام فارسی خوانده نشد: %+v", dec.TaxExemptPercent)
	}
	if !dec.WorkHourReduction || dec.ReducedWeeklyHours != 36 {
		t.Fatalf("کاهش ساعت کار و ساعت مصوب خوانده نشد: %+v", dec)
	}
	if dec.IncentiveSteps == nil || *dec.IncentiveSteps != 2 {
		t.Fatal("پایهٔ تشویقی مصوب خوانده نشد")
	}
	if dec.SeniorityBasis != VeteranSeniorityErafaqi {
		t.Fatalf("مبنای افزایش سنواتی خوانده نشد: %q", dec.SeniorityBasis)
	}
	if dec.EmploymentQuota == nil || *dec.EmploymentQuota {
		t.Fatal("سهمیهٔ استخدامی باید «خیر» باشد")
	}
}

// TestVeteranDecisionOverridesLegalDefault مصوبهٔ کمیته بر پیش‌فرض قانونی مقدم است و سطر تغییر
// برای صورت‌جلسه ساخته می‌شود.
func TestVeteranDecisionOverridesLegalDefault(t *testing.T) {
	base := committeeBase()
	two := 2
	seniority := 54
	dec := VeteranDecisionFromFields(map[string]string{
		"veteranGroup":    "disabled",
		"incentiveSteps":  "2",
		"seniorityMonths": "54",
		"seniorityBasis":  VeteranSeniorityErafaqi,
	})
	merged, changes := ApplyVeteranDecision(base, dec)

	if merged.IncentiveSteps != two || merged.TotalExtraServiceMonths != seniority {
		t.Fatalf("مقادیر مصوبه اعمال نشد: پایه=%d سنوات=%d", merged.IncentiveSteps, merged.TotalExtraServiceMonths)
	}
	if len(changes) < 2 {
		t.Fatalf("برای هر تغییر باید سطر توضیح ساخته شود: %v", changes)
	}
	joined := strings.Join(changes, " ")
	if !strings.Contains(joined, "پایهٔ تشویقی") || !strings.Contains(joined, "افزایش سنواتی") {
		t.Fatalf("سطرهای تغییر باید بندها را نام ببرند: %s", joined)
	}
	if !strings.Contains(merged.NoteFa, "مصوبهٔ کمیته") {
		t.Fatalf("یادداشت باید به مصوبهٔ کمیته اشاره کند: %q", merged.NoteFa)
	}
	// سطر مصوبه در فهرست امتیازها می‌آید تا سند و فیش همان مقدار را نشان دهند.
	found := false
	for _, line := range merged.Lines {
		if line.Key == "seniorityDecision" && strings.Contains(line.ValueFa, "۵۴") {
			found = true
		}
	}
	if !found {
		t.Fatalf("سطر افزایش سنواتی مصوب کمیته در امتیازها نیست: %+v", merged.Lines)
	}
}

// TestVeteranDecisionCanWithholdEntitlement کمیته می‌تواند امتیازی را که قانون داده است، ندهد.
func TestVeteranDecisionCanWithholdEntitlement(t *testing.T) {
	base := committeeBase()
	no := false
	dec := VeteranDecision{Group: VeteranGroupDisabled, TaxExempt: &no, EduUpgrade: &no}
	merged, changes := ApplyVeteranDecision(base, dec)

	if merged.TaxExempt != 0 {
		t.Fatalf("با مصوبهٔ «تعلق نمی‌گیرد» معافیت مالیاتی باید صفر شود: %v", merged.TaxExempt)
	}
	if merged.EducationUpgrade {
		t.Fatal("با مصوبهٔ «خیر» امتیاز یک مقطع بالاتر باید برداشته شود")
	}
	if len(changes) != 2 {
		t.Fatalf("دو تغییر انتظار می‌رفت: %v", changes)
	}
	for _, line := range merged.Lines {
		if line.Key == "taxExempt" && !strings.Contains(line.ValueFa, "تعلق نمی‌گیرد") {
			t.Fatalf("سطر معافیت مالیاتی باید «تعلق نمی‌گیرد» را بگوید: %+v", line)
		}
	}
}

// TestVeteranDecisionTaxPercentOverride درصد مصوب کمیته جای درصد شرکت را می‌گیرد.
func TestVeteranDecisionTaxPercentOverride(t *testing.T) {
	base := committeeBase()
	half := 50.0
	merged, changes := ApplyVeteranDecision(base, VeteranDecision{
		Group: VeteranGroupDisabled, TaxExemptPercent: &half,
	})
	if merged.TaxExempt != 50 {
		t.Fatalf("درصد معافیت مالیاتی مصوبه اعمال نشد: %v", merged.TaxExempt)
	}
	if len(changes) != 1 || !strings.Contains(changes[0], "۵۰") {
		t.Fatalf("سطر تغییر درصد ساخته نشد: %v", changes)
	}
}

// TestVeteranDecisionWorkHourReduction کاهش ساعت کار فقط با مصوبهٔ صریح و با ساعت مصوب.
func TestVeteranDecisionWorkHourReduction(t *testing.T) {
	base := committeeBase()
	merged, changes := ApplyVeteranDecision(base, VeteranDecision{
		Group: VeteranGroupDisabled, WorkHourReduction: true, ReducedWeeklyHours: 36,
	})
	found := false
	for _, line := range merged.Lines {
		if line.Key == "workHourReduction" && strings.Contains(line.ValueFa, "۳۶") {
			found = true
		}
	}
	if !found {
		t.Fatalf("سطر کاهش ساعت کار با ساعت مصوب ساخته نشد: %+v", merged.Lines)
	}
	if len(changes) != 1 || !strings.Contains(changes[0], "کاهش ساعت") {
		t.Fatalf("سطر تغییر کاهش ساعت ساخته نشد: %v", changes)
	}
	// بدون ساعت مصوب، مقدار پیش‌فرض ۳۶ ساعت ثبت می‌شود (نه صفر).
	merged2, _ := ApplyVeteranDecision(base, VeteranDecision{Group: VeteranGroupDisabled, WorkHourReduction: true})
	for _, line := range merged2.Lines {
		if line.Key == "workHourReduction" && !strings.Contains(line.ValueFa, "۳۶") {
			t.Fatalf("بدون ساعت مصوب باید پیش‌فرض ۳۶ ساعت بیاید: %+v", line)
		}
	}
}

// TestVeteranDecisionLinesForForm ردیف‌های «چه چیزی مصوب شد» برای فرم و صورت‌جلسه.
func TestVeteranDecisionLinesForForm(t *testing.T) {
	dec := VeteranDecisionFromFields(map[string]string{
		"veteranGroup":            "disability",
		"taxExemptDecision":       "no",
		"incentiveSteps":          "1",
		"seniorityMonths":         "24",
		"leaveDays":               "15",
		"workHourReduction":       "yes",
		"reducedWeeklyHours":      "30",
		"employmentQuotaDecision": "yes",
	})
	lines := VeteranDecisionLines(dec)
	if len(lines) < 5 {
		t.Fatalf("ردیف‌های مصوبه ناقص است: %d", len(lines))
	}
	joined := ""
	for _, line := range lines {
		joined += line.TitleFa + ": " + line.ValueFa + " | "
	}
	for _, want := range []string{"معافیت مالیاتی", "تعلق نمی‌گیرد", "پایهٔ تشویقی", "افزایش سنواتی", "مرخصی تشویقی", "کاهش ساعت کار", "سهمیهٔ استخدامی"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("ردیف «%s» در متن مصوبه نیست: %s", want, joined)
		}
	}
}

// TestVeteranSeniorityBasesComplete فهرست مبناهای افزایش سنواتی برای فرم کامل است.
func TestVeteranSeniorityBasesComplete(t *testing.T) {
	bases := VeteranSeniorityBases()
	if len(bases) != 3 {
		t.Fatalf("سه مبنا انتظار می‌رفت: %d", len(bases))
	}
	keys := map[string]bool{}
	for _, base := range bases {
		keys[base.Key] = true
		if base.TitleFa == "" {
			t.Fatalf("عنوان فارسی مبنا خالی است: %+v", base)
		}
	}
	for _, want := range []string{VeteranSeniorityCaptivity, VeteranSeniorityErafaqi, VeteranSeniorityBoth} {
		if !keys[want] {
			t.Fatalf("مبنای %q در فهرست نیست", want)
		}
	}
	if VeteranSeniorityBasisTitleFa(VeteranSeniorityErafaqi) == "" {
		t.Fatal("عنوان فارسی مبنا برگردانده نشد")
	}
}

// TestVeteranEntitlementOfFileUsesApprovedValues مصوبهٔ کمیته که در پرونده ذخیره شده است باید
// در امتیاز مؤثر (و در نتیجه فیش) دیده شود، نه فقط پیش‌فرض قانونی.
func TestVeteranEntitlementOfFileUsesApprovedValues(t *testing.T) {
	fields := map[string]string{
		"veteranType":          "freed",
		"captivityMonths":      "24",
		"taxExemptPercent":     "100",
		"pensionExemptPercent": "50",
		"extraServiceMonths":   "60",
		"incentiveSteps":       "2",
		"leaveDays":            "60",
		"educationUpgrade":     "yes",
	}
	ent := VeteranEntitlementOfFile(fields, true, DefaultVeteranParams())
	if ent.PensionExempt != 50 {
		t.Fatalf("معافیت کسور مصوب پرونده اعمال نشد: %v", ent.PensionExempt)
	}
	if ent.TotalExtraServiceMonths != 60 || ent.IncentiveSteps != 2 || ent.LeaveDays != 60 {
		t.Fatalf("مقادیر مصوب پرونده اعمال نشد: %+v", ent)
	}
	// پروندهٔ تأییدنشده فقط پیش‌فرض قانونی می‌دهد.
	// پروندهٔ تأییدنشده هیچ امتیاز مالی و سنواتی نمی‌گیرد (نه مقادیر مصوب و نه پیش‌فرض).
	draft := VeteranEntitlementOfFile(fields, false, DefaultVeteranParams())
	if draft.TotalExtraServiceMonths != 0 || draft.PensionExempt != 0 || draft.TaxExempt != 0 {
		t.Fatalf("پروندهٔ تأییدنشده نباید امتیاز بدهد: %+v", draft)
	}
	if !draft.IsVeteran || draft.Approved {
		t.Fatalf("پروندهٔ تأییدنشده باید ایثارگر شناخته شود ولی تأییدنشده بماند: %+v", draft)
	}
	// پروندهٔ بی‌مقدار (پروندهٔ قدیمی) همان پیش‌فرض قانونی را می‌دهد.
	legacy := VeteranEntitlementOfFile(map[string]string{"veteranType": "جانباز", "percent": "25"}, true, DefaultVeteranParams())
	if legacy.ErafaqiMonths != 48 || legacy.TaxExempt != 100 {
		t.Fatalf("پروندهٔ قدیمی باید پیش‌فرض قانونی بگیرد: %+v", legacy)
	}
}

// TestVeteranDecisionFromApprovedFileپارس فیلدهای خالی و صفر.
func TestVeteranDecisionFromApprovedFileParsesZero(t *testing.T) {
	dec := VeteranDecisionFromApprovedFile(map[string]string{"pensionExemptPercent": "0", "incentiveSteps": "۰"})
	if dec.PensionExempt == nil || *dec.PensionExempt {
		t.Fatal("درصد صفر باید «تعلق نمی‌گیرد» باشد")
	}
	if dec.IncentiveSteps == nil || *dec.IncentiveSteps != 0 {
		t.Fatalf("عدد صفر با ارقام فارسی خوانده نشد: %+v", dec.IncentiveSteps)
	}
	empty := VeteranDecisionFromApprovedFile(map[string]string{"veteranType": "freed"})
	if empty.TaxExempt != nil || empty.PensionExempt != nil || empty.SeniorityMonths != nil {
		t.Fatalf("کلید غایب نباید تصمیم بسازد: %+v", empty)
	}
}
