// آزمون پیاده‌سازی ماژول «grievances» — «veterans_test». آزمون خودکار؛ هر تغییر رفتار پیاده‌سازی باید
// آزمون همین پرونده را به‌روز کند.

package grievances

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/modules/grievances/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/orgs"
	"hrms/internal/rules"
)

// depsWithParams محیط سادهٔ ماژول با پارامترهای ایثارگری دلخواه می‌سازد.
func depsWithParams(params ...orgs.Param) modulekit.Deps {
	return modulekit.Deps{Org: orgs.Organization{Params: params}}
}

// veteranRecord یک پروندهٔ رکورد با وضعیت دلخواه می‌سازد.
func veteranRecord(status string, fields map[string]string) contracts.Record {
	return contracts.Record{
		ID: "VET-0001", Kind: domain.KindVeteranRecords, EmployeeID: "EMP-009",
		Status: status, Fields: fields,
	}
}

// TestProfileOfRecordHonoursApprovalGate پرونده تا «تأییدشده» نشده باشد مبنای امتیاز مالی نیست.
func TestProfileOfRecordHonoursApprovalGate(t *testing.T) {
	fields := map[string]string{
		"veteranType": "جانباز", "percent": "۲۵", "captivityMonths": "24", "frontMonths": "13",
	}
	approved := profileOfRecord(veteranRecord("approved", fields))
	if !approved.ApprovedFile {
		t.Fatal("پروندهٔ تأییدشده باید مبنای اعمال امتیاز باشد")
	}
	if approved.Group != rules.VeteranGroupDisabled || approved.DisabilityPercent != 25 {
		t.Fatalf("گروه/درصد پرونده درست خوانده نشد: %+v", approved)
	}
	if approved.CaptivityMonths != 24 || approved.FrontMonths != 13 {
		t.Fatalf("مدت اسارت/حضور در جبهه درست خوانده نشد: %+v", approved)
	}
	for _, status := range []string{"draft", "verified", "rejected"} {
		if profileOfRecord(veteranRecord(status, fields)).ApprovedFile {
			t.Fatalf("پروندهٔ %q نباید تأییدشده شمرده شود", status)
		}
	}
}

// TestVeteranProfileFromRequestMapsFormInput ورودی فرم باید همان پروفایل موتور شود.
func TestVeteranProfileFromRequestMapsFormInput(t *testing.T) {
	in := veteranRequest{
		EmployeeID: "EMP-005", VeteranType: rules.VeteranGroupFreed, CaptivityMonths: 24,
		ApprovedFile: true,
	}
	profile := veteranProfileFromRequest(in)
	if profile.Group != rules.VeteranGroupFreed || !profile.ApprovedFile {
		t.Fatalf("پروفایل از فرم درست ساخته نشد: %+v", profile)
	}
	if profile.DisabilityPercent != 0 || profile.FrontMonths != 0 {
		t.Fatalf("مقدارهای نامرتبط باید صفر بمانند: %+v", profile)
	}
}

// TestVeteranReliefFollowsApprovalAndParams از پروندهٔ رکورد تا اثر مالی در فیش: پروندهٔ
// تأییدشدهٔ جانباز ۲۵٪ کل مالیات را معاف می‌کند، ولی پروندهٔ پیش‌نویس هیچ اثری ندارد.
func TestVeteranReliefFollowsApprovalAndParams(t *testing.T) {
	params := rules.VeteranParamsFromValues(nil)
	fields := map[string]string{"veteranType": "جانباز", "percent": "25"}
	in := rules.VeteranReliefInput{TaxableIncome: 100_000_000, EmployeeDeduction: 7_000_000}

	approved := rules.ApplyVeteranRelief(in, rules.ComputeVeteranEntitlements(profileOfRecord(veteranRecord("approved", fields)), params))
	if !approved.Applied || approved.TaxExemptAmount != in.TaxableIncome {
		t.Fatalf("معافیت مالیاتی پروندهٔ تأییدشده درست نیست: %+v", approved)
	}
	draft := rules.ApplyVeteranRelief(in, rules.ComputeVeteranEntitlements(profileOfRecord(veteranRecord("draft", fields)), params))
	if draft.Applied || draft.TaxExemptAmount != 0 || draft.PensionExemptAmount != 0 {
		t.Fatalf("پروندهٔ تأییدنشده نباید اثر مالی داشته باشد: %+v", draft)
	}
}

// TestVeteranReliefHonoursCompanyPolicy وقتی شرکت سیاست معافیت کسور را روشن می‌کند، همان
// درصد اعمال می‌شود و هرگز از پایه بیشتر نمی‌شود.
func TestVeteranReliefHonoursCompanyPolicy(t *testing.T) {
	deps := depsWithParams(
		orgs.Param{Key: "veteran.pension.exemptPercent", Value: "100"},
		orgs.Param{Key: "veteran.tax.exemptPercent", Value: "50"},
	)
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	if params.PensionExemptPercent != 100 || params.TaxExemptPercent != 50 {
		t.Fatalf("پارامترهای شرکت خوانده نشد: %+v", params)
	}
	profile := rules.VeteranProfile{Group: rules.VeteranGroupCombatant, FrontMonths: 24, ApprovedFile: true}
	ent := rules.ComputeVeteranEntitlements(profile, params)
	out := rules.ApplyVeteranRelief(rules.VeteranReliefInput{TaxableIncome: 10_000_000, EmployeeDeduction: 6_000_000}, ent)
	if out.TaxExemptAmount != 5_000_000 {
		t.Fatalf("معافیت مالیاتی ۵۰٪ باید ۵٬۰۰۰٬۰۰۰ باشد، شد %d", out.TaxExemptAmount)
	}
	if out.PensionExemptAmount != 6_000_000 {
		t.Fatalf("معافیت کسور ۱۰۰٪ باید کل کسور کارمند باشد، شد %d", out.PensionExemptAmount)
	}
}

// TestVeteranParamValuesReadsOnlyVeteranPrefix فقط پارامترهای همین موضوع خوانده می‌شوند و
// ارقام فارسی و علامت درصد هم پذیرفته می‌شود.
func TestVeteranParamValuesReadsOnlyVeteranPrefix(t *testing.T) {
	deps := depsWithParams(
		orgs.Param{Key: "veteran.tax.exemptPercent", Value: "۱۰۰٪"},
		orgs.Param{Key: "veteran.incentive.stepsMax", Value: "2"},
		orgs.Param{Key: "medical.aid.annualCap", Value: "999"},
		orgs.Param{Key: "veteran.quota.employmentPercent", Value: "۲۵"},
		orgs.Param{Key: "veteran.payroll.apply", Value: ""},
	)
	values := veteranParamValues(deps)
	if values["veteran.tax.exemptPercent"] != 100 {
		t.Fatalf("درصد معافیت با ارقام فارسی و علامت درصد خوانده نشد: %v", values)
	}
	if values["veteran.quota.employmentPercent"] != 25 {
		t.Fatalf("درصد سهمیهٔ استخدامی خوانده نشد: %v", values)
	}
	if _, ok := values["medical.aid.annualCap"]; ok {
		t.Fatal("پارامتر ماژول درمان نباید در پارامترهای ایثارگری بیاید")
	}
	if _, ok := values["veteran.payroll.apply"]; ok {
		t.Fatal("پارامتر بی‌مقدار باید نادیده گرفته شود تا پیش‌فرض بماند")
	}
	// پارامتر بی‌مقدار نباید پیش‌فرض را خنثی کند.
	params := rules.VeteranParamsFromValues(values)
	if !params.PayrollApply {
		t.Fatal("خالی بودن پارامتر اعمال در حقوق نباید پیش‌فرض «بله» را خاموش کند")
	}
	if params.EmploymentQuotaPercent != 25 {
		t.Fatalf("سهمیهٔ استخدامی از پارامتر خوانده نشد: %v", params.EmploymentQuotaPercent)
	}
}

// TestVeteranParamViewsOnlyVeteranKeys فهرست پارامترهای نمایشی برای کارشناس.
func TestVeteranParamViewsOnlyVeteranKeys(t *testing.T) {
	deps := depsWithParams(
		orgs.Param{Key: "veteran.tax.exemptPercent", TitleFa: "درصد معافیت مالیاتی", Value: "100", UnitFa: "درصد"},
		orgs.Param{Key: "payroll.tax.exemption", TitleFa: "معافیت مالیاتی پایه", Value: "240000000"},
	)
	views := veteranParamViews(deps)
	if len(views) != 1 || views[0]["key"] != "veteran.tax.exemptPercent" {
		t.Fatalf("فهرست پارامترهای ایثارگری درست نیست: %v", views)
	}
	if views[0]["unitFa"] != "درصد" || views[0]["titleFa"] != "درصد معافیت مالیاتی" {
		t.Fatalf("عنوان و واحد پارامتر منتقل نشد: %v", views[0])
	}
}

// TestEntitlementLogRecordsCalculation سطرهای محاسبه باید در پرونده بمانند تا سند و بازرسی
// بتوانند مبنای هر عدد را ببینند.
func TestEntitlementLogRecordsCalculation(t *testing.T) {
	params := rules.VeteranParamsFromValues(nil)
	profile := rules.VeteranProfile{Group: rules.VeteranGroupDisabled, DisabilityPercent: 25, ApprovedFile: true}
	ent := rules.ComputeVeteranEntitlements(profile, params)
	log := entitlementLogFa(ent)
	if log == "" {
		t.Fatal("سطرهای محاسبه نباید خالی باشند")
	}
	if got := strings.Count(log, "؛") + 1; len(ent.Lines) > 1 && got != len(ent.Lines) {
		t.Fatalf("تعداد سطرهای متن محاسبه (%d) با سطرهای موتور (%d) نمی‌خواند", got, len(ent.Lines))
	}
	for _, line := range ent.Lines {
		if !strings.Contains(log, line.TitleFa) {
			t.Fatalf("عنوان سطر %q در متن محاسبه نیست: %s", line.TitleFa, log)
		}
	}
}

// TestVeteranDefaultsCarriesEditorialLists فهرست‌های مرجع باید از سرور بیایند، نه از رابط.
func TestVeteranDefaultsCarriesEditorialLists(t *testing.T) {
	deps := depsWithParams(orgs.Param{Key: "veteran.tax.exemptPercent", Value: "100", TitleFa: "درصد معافیت مالیاتی"})
	out := veteranDefaults(nil, deps, nil)
	groups, ok := out["groups"].([]rules.VeteranGroup)
	if !ok || len(groups) == 0 {
		t.Fatalf("فهرست گروه‌های ایثارگری در پاسخ نیست: %v", out["groups"])
	}
	instruction, ok := out["instruction"].([]string)
	if !ok || len(instruction) < 5 {
		t.Fatalf("متن دستورالعمل امتیازها ناقص است: %v", out["instruction"])
	}
	if _, ok := out["params"]; !ok {
		t.Fatal("پارامترهای مؤثر ایثارگری در پاسخ نیست")
	}
	if _, ok := out["today"]; !ok {
		t.Fatal("تاریخ امروز شمسی برای پیش‌فرض فرم لازم است")
	}
	if _, ok := out["yesNo"]; !ok {
		t.Fatal("گزینه‌های بله/خیر برای فیلدهای انتخابی لازم است")
	}
}

// TestNumberValueAcceptsPersianDigits خواندن عدد از فیلد متنی رکورد (ارقام فارسی و جداکننده).
func TestNumberValueAcceptsPersianDigits(t *testing.T) {
	cases := map[string]float64{
		"۲۵":      25,
		"1,500":   1500,
		"1٬500":   1500,
		"  48  ":  48,
		"":        0,
		"نامعتبر": 0,
	}
	for raw, want := range cases {
		if got := numberValue(raw); got != want {
			t.Fatalf("numberValue(%q) = %v، انتظار %v", raw, got, want)
		}
	}
	if intField(48) != "48" || floatField(100) != "100" {
		t.Fatal("تبدیل عدد به متن فیلد رکورد درست نیست")
	}
}
