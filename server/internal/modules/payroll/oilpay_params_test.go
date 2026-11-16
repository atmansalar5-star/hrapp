// آزمون پیاده‌سازی ماژول «payroll» — «oilpay_params_test». آزمون خودکار؛ هر تغییر رفتار پیاده‌سازی
// باید آزمون همین پرونده را به‌روز کند.

package payroll

import (
	"testing"

	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/orgs"
)

// depsOilParams محیط سادهٔ ماژول با پارامترهای نظام صندوق می‌سازد.
func depsOilParams(params ...orgs.Param) modulekit.Deps {
	return modulekit.Deps{Org: orgs.Organization{Params: params}}
}

// TestOilParamOverridesWinOverCompanyProfile بازنویسی همین محاسبه بر پروفایل شرکت مقدم است
// و پروفایل را تغییر نمی‌دهد.
func TestOilParamOverridesWinOverCompanyProfile(t *testing.T) {
	deps := depsOilParams(
		orgs.Param{Key: "oilpay.fund.employeePercent", Value: "6"},
		orgs.Param{Key: "oilpay.fund.employerPercent", Value: "19"},
		orgs.Param{Key: "oilpay.tax.exemption", Value: "24000000"},
		orgs.Param{Key: "oilpay.allowance.workshop", Value: "10"},
	)

	base := oilPayParams(deps, nil)
	if base.FundEmployeePercent != 6 || base.TaxExemption != 24_000_000 {
		t.Fatalf("مقادیر پروفایل شرکت درست خوانده نشد: %+v", base)
	}

	over := oilPayParams(deps, map[string]string{
		"oilpay.fund.employeePercent": "7.5",
		"oilpay.fund.employerPercent": "20",
		"oilpay.tax.exemption":        "0",
		"oilpay.allowance.workshop":   "12",
	})
	if over.FundEmployeePercent != 7.5 || over.FundEmployerPercent != 20 {
		t.Errorf("درصد کسور بازنویسی نشد: %v / %v", over.FundEmployeePercent, over.FundEmployerPercent)
	}
	if over.TaxExemption != 0 {
		t.Errorf("معافیت مالیاتی باید صفر شود: %d", over.TaxExemption)
	}
	workshop := 0.0
	for _, a := range over.Allowances {
		if a.Key == "workshop" {
			workshop = a.Percent
		}
	}
	if workshop != 12 {
		t.Errorf("درصد قلم کارگاهی بازنویسی نشد: %v", workshop)
	}
	// بازنویسی برای همین محاسبه است و پروفایل شرکت را تغییر نمی‌دهد.
	if again := oilPayParams(deps, nil); again.FundEmployeePercent != 6 {
		t.Errorf("محاسبهٔ بعدی باید دوباره مقدار پروفایل را ببیند: %v", again.FundEmployeePercent)
	}
}

// TestOilParamOverridesIgnoreForeignKeys بازنویسی فقط کلیدهای oilpay.* را می‌پذیرد؛ وگرنه
// فرم فیش می‌توانست پارامترهای دیگر سامانه را دست بزند.
func TestOilParamOverridesIgnoreForeignKeys(t *testing.T) {
	deps := depsOilParams(orgs.Param{Key: "oilpay.tax.exemption", Value: "24000000"})
	params := oilPayParams(deps, map[string]string{
		"payroll.tax.exemption":  "1",
		"medical.aid.annualCap":  "1",
		"veteran.payroll.apply":  "0",
		"oilpay.tax.exemption":   "50000000",
		"unknown.key.that.isBad": "abc",
	})
	if params.TaxExemption != 50_000_000 {
		t.Errorf("کلید درست نباید نادیده گرفته شود: %d", params.TaxExemption)
	}
	if len(params.TaxBrackets) == 0 {
		t.Error("پله‌های مالیات باید پیش‌فرض قانونی بمانند")
	}
}

// TestOilTaxBracketsOverrideText پله‌های مالیات از قالب متنی خوانده می‌شوند و قالب نامعتبر
// پیش‌فرض را خراب نمی‌کند.
func TestOilTaxBracketsOverrideText(t *testing.T) {
	deps := depsOilParams()
	params := oilPayParams(deps, map[string]string{
		"oilpay.tax.brackets": "80000000:10,200000000:20,0:35",
	})
	if len(params.TaxBrackets) != 3 {
		t.Fatalf("سه پلهٔ مالیات انتظار می‌رفت: %+v", params.TaxBrackets)
	}
	if params.TaxBrackets[0].Upto != 80_000_000 || params.TaxBrackets[0].Rate != 0.10 {
		t.Errorf("پلهٔ اول درست خوانده نشد: %+v", params.TaxBrackets[0])
	}
	if params.TaxBrackets[2].Upto != 0 || params.TaxBrackets[2].Rate != 0.35 {
		t.Errorf("پلهٔ آخر درست خوانده نشد: %+v", params.TaxBrackets[2])
	}

	// نرخ بدون درصد (۰٫۲) هم همان ۲۰٪ است.
	decimals := oilPayParams(deps, map[string]string{"oilpay.tax.brackets": "100000000:0.2,0:0.3"})
	if decimals.TaxBrackets[0].Rate != 0.2 {
		t.Errorf("نرخ اعشاری درست خوانده نشد: %+v", decimals.TaxBrackets[0])
	}

	// قالب نامعتبر: پیش‌فرض قانونی می‌ماند.
	bad := oilPayParams(deps, map[string]string{"oilpay.tax.brackets": "بیست درصد"})
	if len(bad.TaxBrackets) != len(params.TaxBrackets) && len(bad.TaxBrackets) != 4 {
		t.Errorf("قالب نامعتبر نباید پله‌ها را خراب کند: %+v", bad.TaxBrackets)
	}
}

// TestOilTableCellsOverridable خانه‌های جدول ۱ تا ۳ هم با کلید پارامتری همین محاسبه
// قابل بازنویسی‌اند (فرم فیش می‌تواند درصد مصوب یک محل یا یک پایه را موقتاً عوض کند).
func TestOilTableCellsOverridable(t *testing.T) {
	params := oilPayParams(depsOilParams(), map[string]string{
		"oilpay.site.abadan.deprivation": "30",
		"oilpay.grade.OD.rank":           "120",
	})
	var deprivation, rank float64
	for _, site := range params.Residences {
		if site.Key == "abadan" {
			deprivation = site.Deprivation
		}
	}
	for _, grade := range params.JobGrades {
		if grade.Key == "OD" {
			rank = grade.PositionRank
		}
	}
	if deprivation != 30 {
		t.Errorf("درصد محرومیت آبادان بازنویسی نشد: %v", deprivation)
	}
	if rank != 120 {
		t.Errorf("حق مقام پایهٔ OD بازنویسی نشد: %v", rank)
	}
}
