// آزمون بند LOAN-05: اعتبارسنجی دامنهٔ وام و قسط و پرکردن خودکار کارمزد، مبلغ قسط و ماه
// شروع کسر اقساط. ارجاع: docs/loan-statement-spec-fa.md
package domain

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/recordkit"
)

// loanKind نوع رکورد وام را از شناسهٔ دامنه برمی‌گرداند.
func loanKind(t *testing.T, key string) recordkit.Kind {
	t.Helper()
	kind, ok := Descriptor().KindByKey(key)
	if !ok {
		t.Fatalf("نوع رکورد %s در شناسهٔ ماژول نیست", key)
	}
	return kind
}

// TestLOAN05Validation می‌سنجد شرایط وام با قاعده‌های بستهٔ rules سنجیده می‌شود: مقدارهای
// محاسباتی خودکار پر می‌شوند و ورودی نامعتبر فیلدبه‌فیلد رد می‌شود.
func TestLOAN05Validation(t *testing.T) {
	loans := loanKind(t, "loans")

	// ---------- پرکردن خودکار: کارمزد، مبلغ قسط و ماه شروع کسر ----------
	in := contracts.RecordInput{
		Kind: "loans", Title: "وام ودیعهٔ مسکن — آزمایشی", EmployeeID: "EMP-005",
		Fields: map[string]string{
			"loanType": "housingDeposit", "bank": "melli", "management": "process_one",
			"amount": "800000000", "feeRate": "4", "installments": "60", "receivedDate": "1404/03/15",
		},
	}
	if err := loans.Validate(in); err != nil {
		t.Fatalf("شرایط معتبر نباید خطا بدهد: %v", err)
	}
	want := map[string]string{
		"feeAmount":         "32000000",
		"installmentAmount": "13866667",
		"firstPeriod":       "1404/04",
	}
	for key, value := range want {
		if in.Fields[key] != value {
			t.Fatalf("فیلد %s باید %q پر شود، %q شد", key, value, in.Fields[key])
		}
	}

	// ---------- ورودی نامعتبر: خطای فیلدبه‌فیلد ----------
	bad := []struct {
		name   string
		fields map[string]string
		field  string
	}{
		{"ماه شروع نامعتبر", map[string]string{
			"loanType": "emergency", "amount": "10000000", "installments": "12", "firstPeriod": "1405/13"}, "fields.firstPeriod"},
		{"مبلغ صفر", map[string]string{
			"loanType": "emergency", "amount": "0", "installments": "12"}, "fields.amount"},
		{"اقساط منفی", map[string]string{
			"loanType": "emergency", "amount": "10000000", "installments": "-3"}, "fields.installments"},
		{"اقساط بیش از سقف", map[string]string{
			"loanType": "emergency", "amount": "10000000", "installments": "900"}, "fields.installments"},
		{"نرخ کارمزد نامعتبر", map[string]string{
			"loanType": "bank", "amount": "10000000", "installments": "12", "feeRate": "160"}, "fields.feeRate"},
		{"تسویه بی‌تاریخ", map[string]string{
			"loanType": "emergency", "amount": "10000000", "installments": "12", "settledAmount": "5000000"}, "fields.settleDate"},
	}
	for _, tc := range bad {
		rec := contracts.RecordInput{Kind: "loans", Title: "آزمون", EmployeeID: "EMP-005", Fields: tc.fields}
		err := loans.Validate(rec)
		if err == nil {
			t.Fatalf("«%s» باید رد شود", tc.name)
		}
		verr, ok := err.(*recordkit.ValidationError)
		if !ok {
			t.Fatalf("«%s» باید خطای فیلدی بدهد، %T داد: %v", tc.name, err, err)
		}
		if _, ok := verr.Details[tc.field]; !ok {
			t.Fatalf("«%s» باید خطای %s بدهد: %+v", tc.name, tc.field, verr.Details)
		}
	}

	// ---------- قاعدهٔ نوع وام و متمم همچنان برقرار است ----------
	if err := loans.Validate(contracts.RecordInput{Kind: "loans", Title: "بدون نوع", Fields: map[string]string{}}); err == nil {
		t.Fatal("نبود نوع وام باید خطا بدهد")
	}
	supplement := contracts.RecordInput{Kind: "loans", Title: "متمم بی‌پایه", EmployeeID: "EMP-005",
		Fields: map[string]string{"loanType": "housingDepositSupplement", "amount": "10000000", "installments": "12"}}
	if err := loans.Validate(supplement); err == nil || !strings.Contains(err.Error(), "وام پایه") {
		t.Fatalf("متمم بدون وام پایه باید رد شود: %v", err)
	}

	// ---------- اعتبارسنجی قسط: ماه، مبلغ و جمع سهم‌ها ----------
	installments := loanKind(t, "loanInstallments")
	good := contracts.RecordInput{Kind: "loanInstallments", Title: "قسط ۱", EmployeeID: "EMP-005",
		Fields: map[string]string{"loan": "LOAN-0001", "seq": "1", "period": "1404/04", "amount": "1000000",
			"principal": "900000", "fee": "100000"}}
	if err := installments.Validate(good); err != nil {
		t.Fatalf("قسط معتبر نباید خطا بدهد: %v", err)
	}
	if good.Fields["dueDate"] != "1404/04/01" {
		t.Fatalf("سررسید باید از ماه کسر پر شود، %q شد", good.Fields["dueDate"])
	}
	mismatch := contracts.RecordInput{Kind: "loanInstallments", Title: "قسط ۲", EmployeeID: "EMP-005",
		Fields: map[string]string{"loan": "LOAN-0001", "seq": "2", "period": "1404/05", "amount": "1000000",
			"principal": "800000", "fee": "100000"}}
	if err := installments.Validate(mismatch); err == nil {
		t.Fatal("قسطی که مبلغش با جمع سهم اصل و کارمزد نمی‌خواند باید رد شود")
	}
	badPeriod := contracts.RecordInput{Kind: "loanInstallments", Title: "قسط ۳", EmployeeID: "EMP-005",
		Fields: map[string]string{"dueDate": "1404/05/01", "amount": "1000000", "period": "1404/13"}}
	if err := installments.Validate(badPeriod); err == nil {
		t.Fatal("ماه کسر نامعتبر باید رد شود")
	}
}