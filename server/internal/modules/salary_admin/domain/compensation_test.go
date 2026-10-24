// آزمون دامنهٔ ماژول «salary_admin» (compensation_test).

package domain

import (
	"testing"

	"hrms/internal/contracts"
)

func TestDecreeTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{DecreeDraft, DecreeIssued, true},
		{DecreeIssued, DecreeNotified, true},
		{DecreeNotified, DecreeSuperseded, true},
		{DecreeDraft, DecreeNotified, false},
		{DecreeIssued, DecreeIssued, false},
		{DecreeSuperseded, DecreeDraft, false},
	}
	for _, c := range cases {
		if got := CanDecreeTransition(c.from, c.to); got != c.want {
			t.Errorf("CanDecreeTransition(%q,%q)=%v، انتظار %v", c.from, c.to, got, c.want)
		}
	}
}

func TestCertTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{CertRequested, CertApproved, true},
		{CertRequested, CertRejected, true},
		{CertApproved, CertIssued, true},
		{CertIssued, CertDelivered, true},
		{CertRequested, CertIssued, false},
		{CertDelivered, CertRequested, false},
	}
	for _, c := range cases {
		if got := CanCertTransition(c.from, c.to); got != c.want {
			t.Errorf("CanCertTransition(%q,%q)=%v، انتظار %v", c.from, c.to, got, c.want)
		}
	}
}

func TestValidateSalaryDecree(t *testing.T) {
	valid := contracts.SalaryDecreeInput{
		EmployeeID: "EMP-001", JobGroup: "کارشناس", BaseSalary: 1000,
		EffectiveDate: "1404/07/01", ReasonFa: "افزایش سنواتی",
	}
	if errs := ValidateSalaryDecree(valid); len(errs) != 0 {
		t.Fatalf("ورودی معتبر خطا داد: %v", errs)
	}
	bad := contracts.SalaryDecreeInput{}
	errs := ValidateSalaryDecree(bad)
	for _, key := range []string{"employeeId", "jobGroup", "baseSalary", "effectiveDate", "reasonFa"} {
		if errs[key] == "" {
			t.Errorf("فیلد %q باید خطا بدهد؛ خطاها: %v", key, errs)
		}
	}
	if errs := ValidateSalaryDecree(contracts.SalaryDecreeInput{
		EmployeeID: "EMP-001", JobGroup: "کارشناس", BaseSalary: 1,
		EffectiveDate: "1404/13/01", ReasonFa: "x",
	}); errs["effectiveDate"] == "" {
		t.Errorf("ماه نامعتبر باید خطا بدهد: %v", errs)
	}
}

func TestValidateRevise(t *testing.T) {
	notified := SalaryDecree{Status: DecreeNotified}
	if errs := ValidateRevise(notified, contracts.SalaryDecreeDecision{ReasonFa: "بازنگری"}); len(errs) != 0 {
		t.Fatalf("اصلاح حکم ابلاغ‌شده نباید خطا بدهد: %v", errs)
	}
	draft := SalaryDecree{Status: DecreeDraft}
	if errs := ValidateRevise(draft, contracts.SalaryDecreeDecision{ReasonFa: "بازنگری"}); errs["status"] == "" {
		t.Fatalf("اصلاح حکم پیش‌نویس باید خطا بدهد: %v", errs)
	}
}

func TestValidateCertificate(t *testing.T) {
	if errs := ValidateCertificate(contracts.CertificateInput{EmployeeID: "EMP-001", Kind: CertBank}); len(errs) != 0 {
		t.Fatalf("ورودی معتبر خطا داد: %v", errs)
	}
	if errs := ValidateCertificate(contracts.CertificateInput{Kind: "bogus"}); errs["employeeId"] == "" || errs["kind"] == "" {
		t.Fatalf("ورودی نامعتبر باید خطا بدهد: %v", errs)
	}
}
