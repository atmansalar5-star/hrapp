// آزمون دامنهٔ ماژول «loans_retirement» (allocation_test).

package domain

import (
	"testing"

	"hrms/internal/contracts"
)

// TestAcceptRuntimeManagements تضمین می‌کند قاعدهٔ رکورد «درخواست وام» با اقلام کاتالوگ
// مرجع تنظیمات گسترش می‌یابد؛ وگرنه مدیریتی که کاربر در پنل تنظیمات می‌سازد در فرم تخصیص
// انتخاب می‌شود ولی ثبت، رد می‌شود.
func TestAcceptRuntimeManagements(t *testing.T) {
	desc := AcceptRuntimeManagements(Descriptor(), func(key string) bool { return key == "it_mgmt" })
	kind, ok := desc.KindByKey(KindLoans)
	if !ok {
		t.Fatal("نوع رکورد درخواست وام در شناسهٔ ماژول نیست")
	}

	base := func(management string) contracts.RecordInput {
		return contracts.RecordInput{Fields: map[string]string{"loanType": "bank", "management": management}}
	}

	if err := kind.Validate(base("it_mgmt")); err != nil {
		t.Fatalf("مدیریت کاتالوگ مرجع زمان‌اجرا باید پذیرفته شود: %v", err)
	}
	if err := kind.Validate(base("ghost_mgmt")); err == nil {
		t.Fatal("مدیریت ناشناخته نباید پذیرفته شود")
	}
	if err := kind.Validate(base("process_one")); err != nil {
		t.Fatalf("مدیریت جدول ثابت سهمیه باید همچنان پذیرفته شود: %v", err)
	}
	// قاعده‌های دیگر نباید با این گسترش از دست بروند.
	if err := kind.Validate(contracts.RecordInput{Fields: map[string]string{"loanType": "housingDepositSupplement"}}); err == nil {
		t.Fatal("متمم بدون وام پایه باید خطا بدهد")
	}
}

// validGrant یک درخواست تخصیص معتبر می‌سازد تا هر آزمون فقط یک چیز را عوض کند.
func validGrant() LoanGrantRequest {
	return LoanGrantRequest{
		LoanType: "housingDeposit", Bank: "melli", Management: "process_one",
		Amount: 800_000_000, Installments: 60, ReceivedDate: "1404/07/10",
		Purpose: "ودیعهٔ مسکن", GuarantorName: "علی احمدی", GuarantorRelation: "برادر",
	}
}

// TestComputeLoanTermsAppliesTypePolicy تضمین می‌کند کارمزد و مبلغ هر قسط از سیاست همان
// نوع وام حساب می‌شود. این اعداد به کاربر و مالی وعده داده می‌شوند، پس اشتباه در آن‌ها
// خطای پولی است.
func TestComputeLoanTermsAppliesTypePolicy(t *testing.T) {
	cases := []struct {
		name         string
		loanType     string
		amount       int64
		installments int
		wantFeeRate  float64
		wantFee      int64
		wantN        int
		wantPer      int64
	}{
		// ۸۰۰م با کارمزد ۴٪ → کارمزد ۳۲م؛ (۸۰۰+۳۲)م ÷ ۶۰ = ۱۳٬۸۶۶٬۶۶۶٫۶۷ → گرد به بالا.
		{"ودیعهٔ مسکن", "housingDeposit", 800_000_000, 60, 4, 32_000_000, 60, 13_866_667},
		// وام اضطراری بدون کارمزد؛ ۵۰م ÷ ۱۲ → گرد به بالا.
		{"اضطراری بدون کارمزد", "emergency", 50_000_000, 12, 0, 0, 12, 4_166_667},
		// ۳۰۰م با کارمزد ۱۸٪ → ۵۴م؛ (۳۰۰+۵۴)م ÷ ۲۴ = ۱۴٬۷۵۰٬۰۰۰ دقیق.
		{"وام بانکی", "bank", 300_000_000, 24, 18, 54_000_000, 24, 14_750_000},
		// اقساط خالی یعنی سقف اقساط همان نوع وام (مددکاری ۲۴ قسط).
		{"اقساط پیش‌فرض از نوع وام", "socialWork", 120_000_000, 0, 2, 2_400_000, 24, 5_100_000},
	}
	for _, c := range cases {
		req := validGrant()
		req.LoanType, req.Amount, req.Installments = c.loanType, c.amount, c.installments
		terms, err := ComputeLoanTerms(req)
		if err != nil {
			t.Fatalf("%s: نباید خطا بدهد: %v", c.name, err)
		}
		if terms.FeeRatePercent != c.wantFeeRate {
			t.Fatalf("%s: نرخ کارمزد %v شد، انتظار %v", c.name, terms.FeeRatePercent, c.wantFeeRate)
		}
		if terms.FeeAmount != c.wantFee {
			t.Fatalf("%s: مبلغ کارمزد %d شد، انتظار %d", c.name, terms.FeeAmount, c.wantFee)
		}
		if terms.Installments != c.wantN {
			t.Fatalf("%s: تعداد اقساط %d شد، انتظار %d", c.name, terms.Installments, c.wantN)
		}
		if terms.InstallmentAmount != c.wantPer {
			t.Fatalf("%s: مبلغ هر قسط %d شد، انتظار %d", c.name, terms.InstallmentAmount, c.wantPer)
		}
		// جمع اقساط هرگز نباید از اصل وام + کارمزد کمتر باشد (گرد به بالا همین را تضمین می‌کند).
		if sum := terms.InstallmentAmount * int64(terms.Installments); sum < c.amount+c.wantFee {
			t.Fatalf("%s: جمع اقساط (%d) از مبلغ وام به‌همراه کارمزد کمتر است", c.name, sum)
		}
	}
}

// TestComputeLoanTermsCarriesTypeCeilings تضمین می‌کند سقف‌های مصوب همان نوع وام به فرم
// می‌رسد؛ اگر اشتباه بیاید، کاربر مبلغی بالاتر از مصوبه را پیشنهاد می‌بیند.
func TestComputeLoanTermsCarriesTypeCeilings(t *testing.T) {
	req := validGrant()
	req.LoanType, req.Amount, req.Installments = "bank", 300_000_000, 24
	terms, err := ComputeLoanTerms(req)
	if err != nil {
		t.Fatalf("نباید خطا بدهد: %v", err)
	}
	typ, _ := LoanTypeByKey("bank")
	if terms.MaxAmount != typ.MaxAmount || terms.MaxInstallments != typ.MaxInstallments {
		t.Fatalf("سقف‌های نوع وام درست منتقل نشده: %+v", terms)
	}
}

// TestValidateGrantRequestRejectsBadInput قواعد دامنه را می‌سنجد: نوع ناشناخته، مبلغ
// نامثبت، عبور از سقف مبلغ و اقساط، بانک غیرمجاز، مدیریت ناشناخته، تاریخ غیرشمسی و
// متمم بدون وام پایه.
func TestValidateGrantRequestRejectsBadInput(t *testing.T) {
	if err := ValidateGrantRequest(validGrant()); err != nil {
		t.Fatalf("درخواست معتبر نباید خطا بدهد: %v", err)
	}

	bad := []struct {
		name   string
		mutate func(r *LoanGrantRequest)
	}{
		{"نوع وام خالی", func(r *LoanGrantRequest) { r.LoanType = "" }},
		{"نوع وام ناشناخته", func(r *LoanGrantRequest) { r.LoanType = "ghost" }},
		{"مبلغ صفر", func(r *LoanGrantRequest) { r.Amount = 0 }},
		{"مبلغ منفی", func(r *LoanGrantRequest) { r.Amount = -1 }},
		{"عبور از سقف مبلغ", func(r *LoanGrantRequest) { r.Amount = 2_000_000_000 }},
		{"عبور از سقف اقساط", func(r *LoanGrantRequest) { r.Installments = 120 }},
		{"اقساط منفی", func(r *LoanGrantRequest) { r.Installments = -3 }},
		{"بانک غیرمجاز", func(r *LoanGrantRequest) { r.Bank = "crypto-bank" }},
		{"مدیریت ناشناخته", func(r *LoanGrantRequest) { r.Management = "tehran" }},
		{"تاریخ غیرشمسی", func(r *LoanGrantRequest) { r.ReceivedDate = "not-a-date" }},
		{"تاریخ ناقص", func(r *LoanGrantRequest) { r.ReceivedDate = "1404/07" }},
		{"ماه نامعتبر", func(r *LoanGrantRequest) { r.ReceivedDate = "1404/13/01" }},
		{"تاریخ خالی", func(r *LoanGrantRequest) { r.ReceivedDate = "" }},
		{"متمم بدون وام پایه", func(r *LoanGrantRequest) {
			r.LoanType, r.Amount, r.SupplementOf = "housingDepositSupplement", 200_000_000, ""
		}},
	}
	for _, c := range bad {
		req := validGrant()
		c.mutate(&req)
		if err := ValidateGrantRequest(req); err == nil {
			t.Fatalf("«%s» باید رد می‌شد ولی رد نشد", c.name)
		}
	}
}

// TestValidateGrantRequestAllowsEmptyOptionalFields تضمین می‌کند فیلدهای اختیاری (بانک
// عامل، مدیریت، ضامن) خالی هم مجاز باشند؛ وام‌های داخلی صندوق بانک و ضامن ندارند.
func TestValidateGrantRequestAllowsEmptyOptionalFields(t *testing.T) {
	req := validGrant()
	req.Bank, req.Management, req.GuarantorName, req.GuarantorRelation = "", "", "", ""
	if err := ValidateGrantRequest(req); err != nil {
		t.Fatalf("فیلدهای اختیاری خالی نباید خطا بدهند: %v", err)
	}
}

// TestLoanFieldsReflectTerms تضمین می‌کند فیلدهای رکورد همان چیزی است که محاسبه شده؛
// اگر این دو از هم جدا شوند، فرم و گزارش دو عدد متفاوت نشان می‌دهند.
func TestLoanFieldsReflectTerms(t *testing.T) {
	req := validGrant()
	req.SupplementOf = "LOAN-0001"
	terms, err := ComputeLoanTerms(req)
	if err != nil {
		t.Fatalf("نباید خطا بدهد: %v", err)
	}
	f := LoanFields(req, terms)

	want := map[string]string{
		"loanType": "housingDeposit", "bank": "melli", "management": "process_one",
		"amount": "800000000", "installments": "60", "installmentAmount": "13866667",
		"feeRate": "4", "feeAmount": "32000000", "supplementOf": "LOAN-0001",
		"guarantorName": "علی احمدی", "guarantorRelation": "برادر",
		"receivedDate": "1404/07/10", "purpose": "ودیعهٔ مسکن",
	}
	for k, v := range want {
		if f[k] != v {
			t.Fatalf("فیلد %s = %q شد، انتظار %q", k, f[k], v)
		}
	}
	// همهٔ فیلدهای اعلامی نوع رکورد وام باید پر شده باشند تا فرم خالی نماند.
	for _, key := range []string{
		"loanType", "bank", "management", "amount", "installments", "installmentAmount",
		"feeRate", "feeAmount", "supplementOf", "guarantorName", "guarantorNationalId",
		"guarantorRelation", "receivedDate", "purpose",
	} {
		if _, ok := f[key]; !ok {
			t.Fatalf("فیلد %s در رکورد وام ساخته نشد", key)
		}
	}
}
