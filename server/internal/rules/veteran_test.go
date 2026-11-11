// آزمون موتور امتیازهای ایثارگری.
//
// چرا آزمون مهم است: این امتیازها روی مبلغ مالیات و کسور فیش اثر میگذارند؛ پس هر تغییر
// پارامتر یا جدول باید با آزمون همراه باشد تا اشتباه پرهزینه نشود.
package rules

import (
	"strings"
	"testing"
)

// approved یک پروندهٔ تأییدشده میسازد.
func approved(group string, percent float64, captivityMonths, frontMonths int) VeteranProfile {
	return VeteranProfile{
		Group:             group,
		DisabilityPercent: percent,
		CaptivityMonths:   captivityMonths,
		FrontMonths:       frontMonths,
		ApprovedFile:      true,
	}
}

func TestVeteranGroupsComplete(t *testing.T) {
	groups := VeteranGroups()
	if len(groups) < 7 {
		t.Fatalf("شمار گروه‌های ایثارگری کم است: %d", len(groups))
	}
	seen := map[string]bool{}
	for _, g := range groups {
		if g.Key == "" || g.TitleFa == "" {
			t.Errorf("گروه ناقص: %+v", g)
		}
		if seen[g.Key] {
			t.Errorf("گروه تکراری: %s", g.Key)
		}
		seen[g.Key] = true
	}
	if got := VeteranGroupTitleFa(VeteranGroupDisabled); got != "جانباز" {
		t.Errorf("عنوان گروه جانباز نادرست است: %q", got)
	}
}

func TestVeteranGroupOfReadsLegacyText(t *testing.T) {
	cases := map[string]string{
		"":                   VeteranGroupNone,
		"جانباز":             VeteranGroupDisabled,
		"آزاده":              VeteranGroupFreed,
		"آزاده ۲۴ ماه اسارت": VeteranGroupFreed,
		"رزمنده جبهه":        VeteranGroupCombatant,
		"همسر شهید":          VeteranGroupMartyrFamily,
		"مادر شهید":          VeteranGroupMartyrFamily,
		"فرزند شهید":         VeteranGroupMartyrChild,
		"فرزند جانباز":       VeteranGroupVeteranChild,
		"disabled":           VeteranGroupDisabled,
		"مورد ناشناخته":      VeteranGroupOther,
	}
	for raw, want := range cases {
		if got := VeteranGroupOf(raw); got != want {
			t.Errorf("گروه «%s» باید %q باشد، شد %q", raw, want, got)
		}
	}
}

func TestVeteranTaxExemptionPerArticle56(t *testing.T) {
	params := DefaultVeteranParams()
	for _, group := range []string{
		VeteranGroupDisabled, VeteranGroupFreed, VeteranGroupMartyrFamily,
	} {
		ent := ComputeVeteranEntitlements(approved(group, 25, 0, 0), params)
		if ent.TaxExempt != 100 {
			t.Errorf("گروه %s باید ۱۰۰٪ معافیت مالیاتی داشته باشد، شد %v", group, ent.TaxExempt)
		}
	}
	// رزمنده: معافیت مشروط به حداقل دوازده ماه حضور در جبهه است.
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupCombatant, 0, 0, 12), params); ent.TaxExempt != 100 {
		t.Errorf("رزمنده با ۱۲ ماه حضور باید مشمول معافیت باشد: %v", ent.TaxExempt)
	}
	combatant := ComputeVeteranEntitlements(approved(VeteranGroupCombatant, 0, 0, 6), params)
	if combatant.TaxExempt != 0 {
		t.Errorf("رزمنده با ۶ ماه حضور نباید مشمول معافیت مالیاتی باشد: %v", combatant.TaxExempt)
	}
	// فرزند جانباز مشمول معافیت مالیاتی ماده ۵۶ نیست.
	child := ComputeVeteranEntitlements(approved(VeteranGroupVeteranChild, 0, 0, 0), params)
	if child.TaxExempt != 0 {
		t.Errorf("فرزند جانباز نباید معافیت مالیاتی حقوق داشته باشد: %v", child.TaxExempt)
	}
}

func TestVeteranCaptivityAddsDoubleService(t *testing.T) {
	ent := ComputeVeteranEntitlements(approved(VeteranGroupFreed, 0, 24, 0), DefaultVeteranParams())
	if ent.CaptivityServiceMonths != 48 {
		t.Fatalf("هر سال اسارت باید دو سال سابقه بدهد؛ ۲۴ ماه اسارت ⇒ ۴۸ ماه، شد %d", ent.CaptivityServiceMonths)
	}
	if ent.TotalExtraServiceMonths != 48 {
		t.Errorf("جمع افزایش سنواتی نادرست است: %d", ent.TotalExtraServiceMonths)
	}
	// مرخصی تشویقی: یک‌دهم مدت اسارت با کف دو ماه ⇒ ۲٫۴ ماه = ۷۲ روز.
	if ent.LeaveDays != 72 {
		t.Errorf("مرخصی تشویقی آزاده باید ۷۲ روز باشد، شد %d", ent.LeaveDays)
	}
	// اسارت کوتاه: کف دو ماه رعایت می‌شود.
	short := ComputeVeteranEntitlements(approved(VeteranGroupFreed, 0, 6, 0), DefaultVeteranParams())
	if short.LeaveDays != 60 {
		t.Errorf("کف مرخصی تشویقی دو ماه (۶۰ روز) است، شد %d", short.LeaveDays)
	}
}

func TestVeteranErafaqiTable(t *testing.T) {
	cases := map[float64]int{10: 0, 20: 36, 25: 48, 30: 60, 40: 72, 70: 72}
	for percent, want := range cases {
		if got := ErafaqiMonthsForPercent(percent); got != want {
			t.Errorf("سنوات ارفاقی %v٪ باید %d ماه باشد، شد %d", percent, want, got)
		}
	}
	ent := ComputeVeteranEntitlements(approved(VeteranGroupDisabled, 25, 0, 0), DefaultVeteranParams())
	if ent.ErafaqiMonths != 48 || ent.TotalExtraServiceMonths != 48 {
		t.Errorf("جانباز ۲۵٪ باید ۴۸ ماه سنوات ارفاقی بگیرد: %+v", ent)
	}
}

func TestVeteranErafaqiOverrideAndCap(t *testing.T) {
	params := DefaultVeteranParams()
	params.ErafaqiMonthsOverride = 30
	params.ErafaqiMaxMonths = 120
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupDisabled, 40, 0, 0), params); ent.ErafaqiMonths != 30 {
		t.Errorf("بازنویسی شرکت باید جدول را کنار بزند: %d", ent.ErafaqiMonths)
	}
	params.ErafaqiMonthsOverride = 200
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupDisabled, 40, 0, 0), params); ent.ErafaqiMonths != 120 {
		t.Errorf("سقف سنوات ارفاقی باید ۱۲۰ ماه باشد: %d", ent.ErafaqiMonths)
	}
}

func TestVeteranDraftFileHasNoEntitlement(t *testing.T) {
	profile := approved(VeteranGroupFreed, 0, 24, 0)
	profile.ApprovedFile = false
	ent := ComputeVeteranEntitlements(profile, DefaultVeteranParams())
	if !ent.IsVeteran {
		t.Error("پروندهٔ بررسی‌شده باید «ایثارگر» شمرده شود")
	}
	if ent.Approved {
		t.Error("پروندهٔ تأییدنشده نباید تأییدشده شمرده شود")
	}
	if ent.TaxExempt != 0 || ent.TotalExtraServiceMonths != 0 || ent.LeaveDays != 0 {
		t.Errorf("پروندهٔ تأییدنشده نباید امتیاز مالی بدهد: %+v", ent)
	}
	if !strings.Contains(ent.NoteFa, "تأیید") {
		t.Errorf("یادداشت باید به انتظار تأیید اشاره کند: %q", ent.NoteFa)
	}
}

func TestVeteranNoFile(t *testing.T) {
	ent := ComputeVeteranEntitlements(VeteranProfile{Group: VeteranGroupNone}, DefaultVeteranParams())
	if ent.IsVeteran || ent.TaxExempt != 0 {
		t.Errorf("کارمند بدون پرونده نباید امتیاز بگیرد: %+v", ent)
	}
	if !strings.Contains(ent.SummaryFa, "بدون امتیاز") {
		t.Errorf("جمع‌بندی نادرست است: %q", ent.SummaryFa)
	}
}

func TestVeteranEducationUpgrade(t *testing.T) {
	params := DefaultVeteranParams()
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupFreed, 0, 12, 0), params); !ent.EducationUpgrade {
		t.Error("آزاده باید امتیاز یک مقطع بالاتر داشته باشد")
	}
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupDisabled, 30, 0, 0), params); !ent.EducationUpgrade {
		t.Error("جانباز باید امتیاز یک مقطع بالاتر داشته باشد")
	}
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupCombatant, 0, 0, 24), params); ent.EducationUpgrade {
		t.Error("رزمنده امتیاز یک مقطع بالاتر ندارد")
	}
	params.EducationUpgrade = false
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupFreed, 0, 12, 0), params); ent.EducationUpgrade {
		t.Error("با خاموش‌کردن پارامتر، معادل‌سازی نباید اعمال شود")
	}
}

func TestVeteranIncentiveStepsPolicy(t *testing.T) {
	params := DefaultVeteranParams()
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupMartyrFamily, 0, 0, 0), params); ent.IncentiveSteps != 2 {
		t.Errorf("خانوادهٔ شهید پیش‌فرض دو پایهٔ تشویقی دارد: %d", ent.IncentiveSteps)
	}
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupDisabled, 25, 0, 0), params); ent.IncentiveSteps != 1 {
		t.Errorf("جانباز پیش‌فرض یک پایهٔ تشویقی دارد: %d", ent.IncentiveSteps)
	}
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupVeteranChild, 0, 0, 0), params); ent.IncentiveSteps != 0 {
		t.Errorf("فرزند جانباز پایهٔ تشویقی ندارد: %d", ent.IncentiveSteps)
	}
	params.IncentiveStepsMax = 1
	if ent := ComputeVeteranEntitlements(approved(VeteranGroupMartyrFamily, 0, 0, 0), params); ent.IncentiveSteps != 1 {
		t.Errorf("سقف پارامتری پایهٔ تشویقی رعایت نشد: %d", ent.IncentiveSteps)
	}
}

func TestApplyVeteranReliefTax(t *testing.T) {
	ent := ComputeVeteranEntitlements(approved(VeteranGroupDisabled, 25, 0, 0), DefaultVeteranParams())
	relief := ApplyVeteranRelief(VeteranReliefInput{TaxableIncome: 100_000_000, EmployeeDeduction: 7_000_000}, ent)
	if relief.TaxExemptAmount != 100_000_000 {
		t.Fatalf("با معافیت ۱۰۰٪ تمام درآمد مشمول باید معاف شود: %d", relief.TaxExemptAmount)
	}
	if relief.PensionExemptAmount != 0 {
		t.Errorf("بدون سیاست شرکت، کسور نباید کاهش یابد: %d", relief.PensionExemptAmount)
	}
	if !relief.Applied {
		t.Error("اعمال امتیاز باید گزارش شود")
	}
}

func TestApplyVeteranReliefPension(t *testing.T) {
	params := DefaultVeteranParams()
	params.PensionExemptPercent = 50
	ent := ComputeVeteranEntitlements(approved(VeteranGroupDisabled, 25, 0, 0), params)
	if ent.PensionExempt != 50 {
		t.Fatalf("درصد معافیت کسور در امتیاز ثبت نشد: %v", ent.PensionExempt)
	}
	relief := ApplyVeteranRelief(VeteranReliefInput{TaxableIncome: 60_000_000, EmployeeDeduction: 7_000_000}, ent)
	if relief.PensionExemptAmount != 3_500_000 {
		t.Errorf("نصف کسور ۷٬۰۰۰٬۰۰۰ باید معاف شود: %d", relief.PensionExemptAmount)
	}
	// تعدیل نباید از خود مبنا بیشتر شود.
	params.PensionExemptPercent = 100
	ent = ComputeVeteranEntitlements(approved(VeteranGroupDisabled, 25, 0, 0), params)
	full := ApplyVeteranRelief(VeteranReliefInput{TaxableIncome: 0, EmployeeDeduction: 5_000_000}, ent)
	if full.PensionExemptAmount != 5_000_000 {
		t.Errorf("معافیت کسور نباید از کسور بیشتر شود: %d", full.PensionExemptAmount)
	}
}

func TestApplyVeteranReliefIgnoresUnapproved(t *testing.T) {
	profile := approved(VeteranGroupFreed, 0, 24, 0)
	profile.ApprovedFile = false
	ent := ComputeVeteranEntitlements(profile, DefaultVeteranParams())
	relief := ApplyVeteranRelief(VeteranReliefInput{TaxableIncome: 100_000_000, EmployeeDeduction: 7_000_000}, ent)
	if relief.Applied || relief.TaxExemptAmount != 0 || relief.PensionExemptAmount != 0 {
		t.Errorf("پروندهٔ تأییدنشده نباید تعدیل مالی بدهد: %+v", relief)
	}
}

func TestVeteranParamsFromValues(t *testing.T) {
	params := VeteranParamsFromValues(map[string]float64{
		"veteran.tax.exemptPercent":             80,
		"veteran.pension.exemptPercent":         60,
		"veteran.pension.erafaqiMonthsOverride": 24,
		"veteran.education.upgrade":             0,
		"veteran.incentive.steps.freed":         2,
		"veteran.quota.employmentPercent":       30,
		"veteran.payroll.apply":                 0,
		"veteran.unknown.key":                   9,
	})
	if params.TaxExemptPercent != 80 || params.PensionExemptPercent != 60 {
		t.Errorf("پارامترهای درصدی خوانده نشد: %+v", params)
	}
	if params.ErafaqiMonthsOverride != 24 || params.EducationUpgrade || params.PayrollApply {
		t.Errorf("پارامترهای منطقی/عددی خوانده نشد: %+v", params)
	}
	if params.IncentiveSteps[VeteranGroupFreed] != 2 || params.EmploymentQuotaPercent != 30 {
		t.Errorf("پارامترهای گروهی خوانده نشد: %+v", params)
	}
	// مقدار نامعتبر درصد نباید اعمال شود.
	bad := VeteranParamsFromValues(map[string]float64{"veteran.tax.exemptPercent": 150})
	if bad.TaxExemptPercent != DefaultVeteranParams().TaxExemptPercent {
		t.Errorf("درصد بیرون بازه نباید اعمال شود: %v", bad.TaxExemptPercent)
	}
}

// TestVeteranIncentiveParamKeyForms نشان می‌دهد کلید پارامتر گروه‌های چندبخشی هم به شکل
// camelCase (قاعدهٔ کلید پارامترهای شرکت) و هم به شکل قدیمی با زیرخط خوانده می‌شود.
func TestVeteranIncentiveParamKeyForms(t *testing.T) {
	camel := VeteranParamsFromText(map[string]string{
		"veteran.incentive.steps.martyrFamily": "3",
		"veteran.incentive.steps.martyrChild":  "2",
		"veteran.incentive.steps.veteranChild": "1",
	})
	if camel.IncentiveSteps[VeteranGroupMartyrFamily] != 3 {
		t.Fatalf("کلید camelCase خانوادهٔ شهید خوانده نشد: %+v", camel.IncentiveSteps)
	}
	if camel.IncentiveSteps[VeteranGroupMartyrChild] != 2 || camel.IncentiveSteps[VeteranGroupVeteranChild] != 1 {
		t.Fatalf("کلید camelCase فرزندان خوانده نشد: %+v", camel.IncentiveSteps)
	}
	legacy := VeteranParamsFromText(map[string]string{"veteran.incentive.steps.martyr_family": "2"})
	if legacy.IncentiveSteps[VeteranGroupMartyrFamily] != 2 {
		t.Fatalf("کلید قدیمی با زیرخط نباید بی‌اثر شود: %+v", legacy.IncentiveSteps)
	}
	if got := incentiveParamKey(VeteranGroupMartyrFamily); got != "martyrFamily" {
		t.Fatalf("تبدیل نام گروه به کلید پارامتر درست نیست: %q", got)
	}
	if got := incentiveParamKey(VeteranGroupDisabled); got != "disabled" {
		t.Fatalf("نام گروه تک‌بخشی نباید تغییر کند: %q", got)
	}
}

// TestVeteranParamsFromText پارامترهای شرکت را با ارقام فارسی، جداکننده و علامت درصد
// می‌خواند و کلید بی‌مقدار یا ناخوانا را نادیده می‌گیرد.
func TestVeteranParamsFromText(t *testing.T) {
	params := VeteranParamsFromText(map[string]string{
		"veteran.tax.exemptPercent":        "۱۰۰٪",
		"veteran.quota.employmentPercent":  "۲۵",
		"veteran.payroll.apply":            "",
		"veteran.leave.captivityRatio":     "0.1",
		"veteran.pension.erafaqiMaxMonths": "1,200",
		"medical.aid.orgSharePercent":      "50",
		"veteran.education.upgrade":        "0",
		"veteran.incentive.stepsMax":       "۲",
	})
	if params.TaxExemptPercent != 100 {
		t.Fatalf("درصد با ارقام فارسی و علامت درصد خوانده نشد: %v", params.TaxExemptPercent)
	}
	if params.EmploymentQuotaPercent != 25 || params.IncentiveStepsMax != 2 {
		t.Fatalf("درصد سهمیه و سقف پایه خوانده نشد: %+v", params)
	}
	if !params.PayrollApply {
		t.Fatal("کلید بی‌مقدار نباید پیش‌فرض «اعمال در فیش» را خاموش کند")
	}
	if params.ErafaqiMaxMonths != 1200 {
		t.Fatalf("جداکنندهٔ هزارگان نباید عدد را خراب کند: %v", params.ErafaqiMaxMonths)
	}
	if params.EducationUpgrade {
		t.Fatal("مقدار صفر باید امتیاز «یک مقطع بالاتر» را خاموش کند")
	}
}

func TestVeteranSummaryAndInstruction(t *testing.T) {
	params := DefaultVeteranParams()
	ent := ComputeVeteranEntitlements(approved(VeteranGroupFreed, 0, 24, 0), params)
	for _, want := range []string{"آزاده", "معافیت مالیاتی", "افزایش سنواتی", "مرخصی تشویقی"} {
		if !strings.Contains(ent.SummaryFa, want) {
			t.Errorf("جمع‌بندی باید «%s» را داشته باشد: %q", want, ent.SummaryFa)
		}
	}
	if len(VeteranInstructionFa()) < 10 {
		t.Errorf("دستورالعمل باید دست‌کم ۱۰ گام داشته باشد: %d", len(VeteranInstructionFa()))
	}
	if ent.EmploymentQuotaPercent != 25 {
		t.Errorf("سهمیهٔ استخدامی پیش‌فرض باید ۲۵٪ باشد: %v", ent.EmploymentQuotaPercent)
	}
	if len(ent.Lines) == 0 || ent.Lines[0].TitleFa == "" {
		t.Error("سطرهای امتیاز باید شرح فارسی داشته باشند")
	}
}
