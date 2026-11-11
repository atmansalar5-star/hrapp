package modulekit

import (
	"context"
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/orgs"
	"hrms/internal/rules"
)

// veteransFake پورت فقط‌خواندنی ساختگی ماژول امور ایثارگران.
//
// فیلتر «نوع رکورد» را همان‌جا اعمال می‌کند تا همان مسیری آزموده شود که در اجرای واقعی
// طی می‌شود، نه یک میان‌بر در لایهٔ مصرف‌کننده.
type veteransFake struct{ items []contracts.Record }

func (f veteransFake) List(_ context.Context, filter contracts.RecordFilter) (contracts.RecordPage, error) {
	items := []contracts.Record{}
	for _, rec := range f.items {
		if filter.Kind != "" && rec.Kind != filter.Kind {
			continue
		}
		items = append(items, rec)
	}
	return contracts.RecordPage{Items: items, Total: len(items), Page: 1, Size: len(items)}, nil
}

func (f veteransFake) Schema() contracts.RecordSchema {
	return contracts.RecordSchema{Module: "grievances", ModuleTitleFa: "امور رفاهی و ایثارگران"}
}

// depsWithVeterans محیطی با پرونده‌های ایثارگری و پارامترهای شرکت می‌سازد.
func depsWithVeterans(records []contracts.Record, params ...orgs.Param) Deps {
	ports := NewPorts()
	ports.SetRecordList(veteransModuleID, veteransFake{items: records})
	return Deps{Org: orgs.Organization{Params: params}, Ports: ports}
}

// veteranRecord پروندهٔ ایثارگری با وضعیت و فیلدهای دلخواه می‌سازد.
func veteranRecord(status, employeeID string, fields map[string]string) contracts.Record {
	return contracts.Record{
		ID: "VET-" + employeeID, Kind: veteranKind, EmployeeID: employeeID,
		Status: status, Fields: fields,
	}
}

// TestVeteranEntitlementReadsApprovedFile پروندهٔ تأییدشده باید به امتیاز مالی برسد.
func TestVeteranEntitlementReadsApprovedFile(t *testing.T) {
	deps := depsWithVeterans([]contracts.Record{
		veteranRecord("approved", "EMP-001", map[string]string{"veteranType": "جانباز", "percent": "۲۵"}),
	})
	ent := deps.VeteranEntitlement(context.Background(), "EMP-001")
	if ent == nil {
		t.Fatal("پروندهٔ تأییدشدهٔ جانباز باید امتیاز بدهد")
	}
	if !ent.Approved || !ent.IsVeteran || ent.TaxExempt != 100 {
		t.Fatalf("امتیاز پروندهٔ تأییدشده درست خوانده نشد: %+v", ent)
	}
	if ent.GroupFa != "جانباز" {
		t.Fatalf("عنوان گروه باید «جانباز» باشد: %q", ent.GroupFa)
	}
}

// TestVeteranEntitlementSkipsUnapproved پروندهٔ در حال بررسی نباید حقوق را تغییر دهد.
func TestVeteranEntitlementSkipsUnapproved(t *testing.T) {
	for _, status := range []string{"draft", "verified", "rejected"} {
		deps := depsWithVeterans([]contracts.Record{
			veteranRecord(status, "EMP-001", map[string]string{"veteranType": "آزاده", "captivityMonths": "24"}),
		})
		if ent := deps.VeteranEntitlement(context.Background(), "EMP-001"); ent != nil {
			t.Fatalf("پروندهٔ %q نباید در فیش اثر بگذارد: %+v", status, ent)
		}
	}
}

// TestVeteranEntitlementPrefersApproved پروندهٔ تأییدشده بر بررسی‌شده اولویت دارد.
func TestVeteranEntitlementPrefersApproved(t *testing.T) {
	deps := depsWithVeterans([]contracts.Record{
		veteranRecord("verified", "EMP-001", map[string]string{"veteranType": "جانباز", "percent": "10"}),
		veteranRecord("approved", "EMP-001", map[string]string{"veteranType": "جانباز", "percent": "40"}),
	})
	ent := deps.VeteranEntitlement(context.Background(), "EMP-001")
	if ent == nil || !ent.Approved {
		t.Fatalf("پروندهٔ تأییدشده باید برگزیده شود: %+v", ent)
	}
	if ent.ErafaqiMonths == 0 {
		t.Fatalf("سنوات ارفاقی پروندهٔ تأییدشده باید محاسبه شود: %+v", ent)
	}
}

// TestVeteranEntitlementIgnoresOtherEmployee پروندهٔ کارمند دیگر نباید اعمال شود.
func TestVeteranEntitlementIgnoresOtherEmployee(t *testing.T) {
	deps := depsWithVeterans([]contracts.Record{
		veteranRecord("approved", "EMP-002", map[string]string{"veteranType": "جانباز", "percent": "25"}),
	})
	if ent := deps.VeteranEntitlement(context.Background(), "EMP-001"); ent != nil {
		t.Fatalf("پروندهٔ کارمند دیگر نباید اعمال شود: %+v", ent)
	}
	if ent := deps.VeteranEntitlement(context.Background(), ""); ent != nil {
		t.Fatalf("کد کارمند خالی نباید پرونده‌ای برگرداند: %+v", ent)
	}
}

// TestVeteranEntitlementHonoursCompanySwitch کلید `veteran.payroll.apply` اعمال در فیش را
// خاموش می‌کند؛ در آن حالت هیچ امتیازی به فیش نمی‌رسد.
func TestVeteranEntitlementHonoursCompanySwitch(t *testing.T) {
	records := []contracts.Record{
		veteranRecord("approved", "EMP-001", map[string]string{"veteranType": "جانباز", "percent": "25"}),
	}
	on := depsWithVeterans(records, orgs.Param{Key: "veteran.payroll.apply", Value: "1"})
	if on.VeteranEntitlement(context.Background(), "EMP-001") == nil {
		t.Fatal("با مقدار ۱ امتیاز باید اعمال شود")
	}
	off := depsWithVeterans(records, orgs.Param{Key: "veteran.payroll.apply", Value: "0"})
	if ent := off.VeteranEntitlement(context.Background(), "EMP-001"); ent != nil {
		t.Fatalf("با مقدار ۰ هیچ امتیازی نباید اعمال شود: %+v", ent)
	}
}

// TestVeteranEntitlementFeedsPayslip از پرونده تا فیش: خالص پرداختی کارمند جانباز باید
// از همان کارمند بدون پرونده بیشتر باشد.
func TestVeteranEntitlementFeedsPayslip(t *testing.T) {
	deps := depsWithVeterans([]contracts.Record{
		veteranRecord("approved", "EMP-001", map[string]string{"veteranType": "جانباز", "percent": "25"}),
	})
	params := rules.DefaultPayrollParams()

	plain := rules.ComputePayroll(rules.PayrollInput{BaseSalary: 120_000_000}, params)
	input := rules.PayrollInput{
		BaseSalary: 120_000_000,
		Veteran:    deps.VeteranEntitlement(context.Background(), "EMP-001"),
	}
	withVeteran := rules.ComputePayroll(input, params)

	if !withVeteran.VeteranApplied {
		t.Fatal("فیش کارمند جانباز باید امتیاز اعمال‌شده را نشان دهد")
	}
	if withVeteran.Tax >= plain.Tax {
		t.Fatalf("مالیات کارمند جانباز باید کمتر از کارمند بدون پرونده باشد: %d در برابر %d",
			withVeteran.Tax, plain.Tax)
	}
	if withVeteran.Net <= plain.Net {
		t.Fatalf("خالص پرداختی کارمند جانباز باید بیشتر باشد: %d در برابر %d",
			withVeteran.Net, plain.Net)
	}
	if withVeteran.VeteranGroupFa != "جانباز" || withVeteran.VeteranNoteFa == "" {
		t.Fatalf("گروه و توضیح امتیاز باید در فیش بیاید: %+v", withVeteran)
	}
}

// TestVeteranEntitlementPensionExemptFollowsParam کاهش کسور سهم کارمند تنها با مصوبهٔ
// شرکت (`veteran.pension.exemptPercent`) اعمال می‌شود.
func TestVeteranEntitlementPensionExemptFollowsParam(t *testing.T) {
	records := []contracts.Record{
		veteranRecord("approved", "EMP-001", map[string]string{"veteranType": "آزاده", "captivityMonths": "24"}),
	}
	plain := depsWithVeterans(records).VeteranEntitlement(context.Background(), "EMP-001")
	if plain == nil || plain.PensionExempt != 0 {
		t.Fatalf("بدون مصوبه، کاهش کسور نباید اعمال شود: %+v", plain)
	}
	approved := depsWithVeterans(records,
		orgs.Param{Key: "veteran.pension.exemptPercent", Value: "60"}).
		VeteranEntitlement(context.Background(), "EMP-001")
	if approved == nil || approved.PensionExempt != 60 {
		t.Fatalf("با مصوبه، درصد کاهش کسور باید ۶۰ باشد: %+v", approved)
	}
}

// TestVeteranEntitlementsMapCoversAllEmployees نقشهٔ کارکرد دوره: یک فهرست برای همه، و
// پروندهٔ تأییدنشده در نقشه نمی‌آید.
func TestVeteranEntitlementsMapCoversAllEmployees(t *testing.T) {
	deps := depsWithVeterans([]contracts.Record{
		veteranRecord("approved", "EMP-001", map[string]string{"veteranType": "جانباز", "percent": "25"}),
		veteranRecord("approved", "EMP-003", map[string]string{"veteranType": "آزاده", "captivityMonths": "24"}),
		veteranRecord("verified", "EMP-002", map[string]string{"veteranType": "رزمنده", "frontMonths": "18"}),
	})
	entitlements := deps.VeteranEntitlements(context.Background())
	if len(entitlements) != 2 {
		t.Fatalf("فقط دو پروندهٔ تأییدشده باید در نقشه باشند: %d", len(entitlements))
	}
	if entitlements["EMP-001"] == nil || entitlements["EMP-003"] == nil {
		t.Fatalf("پروندهٔ تأییدشدهٔ هر دو کارمند باید در نقشه باشد: %+v", entitlements)
	}
	if entitlements["EMP-002"] != nil {
		t.Fatalf("پروندهٔ بررسی‌شده نباید در نقشهٔ پرداخت باشد: %+v", entitlements["EMP-002"])
	}
}

// TestVeteranEntitlementOilSlip فیش صندوق نفت هم همان پرونده را می‌بیند.
func TestVeteranEntitlementOilSlip(t *testing.T) {
	deps := depsWithVeterans([]contracts.Record{
		veteranRecord("approved", "EMP-001", map[string]string{"veteranType": "جانباز", "percent": "25"}),
	})
	in := rules.OilPayInput{EmployeeID: "EMP-001", BaseSalary: 120_000_000}
	in.Veteran = deps.VeteranEntitlement(context.Background(), in.EmployeeID)
	slip := rules.ComputeOilPayslip(in, rules.DefaultOilPayParams())
	if !slip.VeteranApplied || slip.VeteranGroupFa != "جانباز" {
		t.Fatalf("فیش صندوق نفت باید امتیاز ایثارگری را نشان دهد: %+v", slip)
	}
}

// TestVeteranEntitlementWithoutPort بدون پورت (اجرای بدون ماژول امور ایثارگران) هیچ
// امتیازی نباید اعمال شود و اجرا هم نباید بشکند.
func TestVeteranEntitlementWithoutPort(t *testing.T) {
	if ent := (Deps{}).VeteranEntitlement(context.Background(), "EMP-001"); ent != nil {
		t.Fatalf("بدون پورت هیچ امتیازی نباید ساخته شود: %+v", ent)
	}
}

// TestVeteranParamsShippedInCompanyProfile پارامترهای امور ایثارگران باید در پروفایل شرکت
// (تنظیمات) حاضر باشند؛ وگرنه شرکت نمیتواند سیاستش را ببیند و تغییر دهد. مقادیر پیشفرض
// باید همان پیشفرضهای مستند به قانون باشند تا رفتار پیش از تغییر سیاست عوض نشود.
func TestVeteranParamsShippedInCompanyProfile(t *testing.T) {
	keys := []string{
		"veteran.tax.exemptPercent", "veteran.tax.minFrontMonths",
		"veteran.pension.exemptPercent", "veteran.pension.captivityServiceFactor",
		"veteran.pension.erafaqiMonthsOverride", "veteran.pension.erafaqiMaxMonths",
		"veteran.education.upgrade", "veteran.incentive.stepsMax",
		"veteran.leave.captivityRatio", "veteran.leave.minMonths", "veteran.leave.maxMonths",
		"veteran.quota.employmentPercent", "veteran.payroll.apply",
		"veteran.incentive.steps.martyrFamily", "veteran.incentive.steps.disabled",
		"veteran.incentive.steps.freed", "veteran.incentive.steps.combatant",
		"veteran.incentive.steps.martyrChild", "veteran.incentive.steps.veteranChild",
	}
	for _, org := range []orgs.Organization{orgs.Default(), mustOrg(t, "bandar_imam")} {
		byKey := map[string]string{}
		for _, p := range org.Params {
			byKey[p.Key] = p.Value
			if strings.HasPrefix(p.Key, "veteran.") && p.GroupFa != "امور ایثارگران" {
				t.Fatalf("پارامتر %q در گروه «امور ایثارگران» نیست: %q", p.Key, p.GroupFa)
			}
		}
		for _, key := range keys {
			value, ok := byKey[key]
			if !ok {
				t.Fatalf("پارامتر %q در پروفایل شرکت %q نیست", key, org.ID)
			}
			if strings.TrimSpace(value) == "" {
				t.Fatalf("پارامتر %q در شرکت %q بی‌مقدار است", key, org.ID)
			}
		}
		values := map[string]string{}
		for key, value := range byKey {
			if strings.HasPrefix(key, "veteran.") {
				values[key] = value
			}
		}
		params := rules.VeteranParamsFromText(values)
		want := rules.DefaultVeteranParams()
		if params.TaxExemptPercent != want.TaxExemptPercent ||
			params.TaxMinFrontMonths != want.TaxMinFrontMonths ||
			params.PensionExemptPercent != want.PensionExemptPercent ||
			params.CaptivityServiceFactor != want.CaptivityServiceFactor ||
			params.ErafaqiMaxMonths != want.ErafaqiMaxMonths ||
			params.EducationUpgrade != want.EducationUpgrade ||
			params.EmploymentQuotaPercent != want.EmploymentQuotaPercent ||
			params.PayrollApply != want.PayrollApply {
			t.Fatalf("مقادیر پارامتری شرکت %q با پیش‌فرض قانونی نمی‌خواند: %+v", org.ID, params)
		}
		if params.IncentiveSteps[rules.VeteranGroupMartyrFamily] != 2 ||
			params.IncentiveSteps[rules.VeteranGroupVeteranChild] != 0 {
			t.Fatalf("پایهٔ تشویقی شرکت %q از پروفایل خوانده نشد: %+v", org.ID, params.IncentiveSteps)
		}
	}
}

// mustOrg یک شرکت را از فهرست استقرارها می‌گیرد یا تست را می‌شکند.
func mustOrg(t *testing.T, id string) orgs.Organization {
	t.Helper()
	org, ok := orgs.Lookup(id)
	if !ok {
		t.Fatalf("شرکت %q در استقرارها نیست", id)
	}
	return org
}
