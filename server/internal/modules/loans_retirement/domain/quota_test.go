package domain

import "testing"

// TestQuotaUnitsSumsToHundred تضمین می‌کند جدول سهمیه یک تخصیص کامل و بی‌کم‌وکاست است:
// مجموع درصدها ۱۰۰ و کلیدها یگانه باشند، وگرنه نمودار سهمیه معنایش را از دست می‌دهد.
func TestQuotaUnitsSumsToHundred(t *testing.T) {
	units := QuotaUnits()
	if len(units) != 8 {
		t.Fatalf("جدول سهمیه باید ۸ مدیریت بندر امام داشته باشد، %d شد", len(units))
	}
	if total := QuotaShareTotal(); total != 100 {
		t.Fatalf("مجموع درصد سهمیه باید ۱۰۰ باشد، %v شد", total)
	}
	seen := map[string]bool{}
	for _, u := range units {
		if seen[u.Key] {
			t.Fatalf("کلید مدیریت تکراری است: %s", u.Key)
		}
		seen[u.Key] = true
		if u.SharePercent <= 0 || u.Eligible <= 0 || u.TitleFa == "" {
			t.Fatalf("مدیریت %s باید درصد، مشمولین و عنوان مثبت داشته باشد: %+v", u.Key, u)
		}
	}
	if QuotaEligibleTotal() <= 0 {
		t.Fatal("مجموع مشمولین باید مثبت باشد")
	}
	if !ValidQuotaUnit("bandar_imam") || ValidQuotaUnit("unknown-unit") {
		t.Fatal("اعتبارسنجی مدیریت درست کار نمی‌کند")
	}
	if QuotaUnitTitleFa("kimia") != "کیمیا" {
		t.Fatalf("عنوان مدیریت کیمیا درست نیست: %s", QuotaUnitTitleFa("kimia"))
	}
}

// TestQuotaStagesCoverAllLoanStatuses تضمین می‌کند هر وضعیت گردش‌کار وام در یکی از
// مراحل ارسال به مدیریت مالی جای می‌گیرد و تفکیک «ارسالی/نارسالی» بی‌ابهام است.
func TestQuotaStagesCoverAllLoanStatuses(t *testing.T) {
	kind, ok := Descriptor().KindByKey("loans")
	if !ok {
		t.Fatal("نوع رکورد وام یافت نشد")
	}
	for _, st := range kind.Statuses {
		stage := QuotaStageOf(st.Value)
		if stage.Key != "rejected" && stage.TitleFa == "" {
			t.Fatalf("وضعیت %s به مرحله‌ای نگاشت نشد", st.Value)
		}
	}
	if !SentToFinance("approved") || !SentToFinance("repaying") || !SentToFinance("settled") {
		t.Fatal("وضعیت‌های تأییدشده، در بازپرداخت و تسویه‌شده باید «ارسالی به مالی» باشند")
	}
	if SentToFinance("requested") || SentToFinance("scored") || SentToFinance("rejected") {
		t.Fatal("وضعیت‌های درخواست‌شده، امتیازبندی‌شده و ردشده نباید «ارسالی به مالی» باشند")
	}
}

// TestEmploymentGroupSplit تفکیک نیروی شرکتی از ارکان ثالث را بر پایهٔ خانوادهٔ مزد
// کاتالوگ انواع استخدام می‌سنجد.
func TestEmploymentGroupSplit(t *testing.T) {
	cases := map[string]string{
		"official_oil":       EmploymentGroupCompany,
		"official_worker":    EmploymentGroupCompany,
		"fixed_1662":         EmploymentGroupCompany,
		"long_term":          EmploymentGroupCompany,
		"contractor_staff":   EmploymentGroupThirdParty,
		"contractor_project": EmploymentGroupThirdParty,
		"":                   EmploymentGroupCompany,
	}
	for input, want := range cases {
		if got := EmploymentGroupOf(input); got != want {
			t.Fatalf("گروه استخدام %q باید %s باشد، %s شد", input, want, got)
		}
	}
	if EmploymentGroupTitleFa(EmploymentGroupThirdParty) == "" ||
		EmploymentGroupTitleFa(EmploymentGroupCompany) == "" {
		t.Fatal("عنوان فارسی گروه‌های کارگزینی باید تعریف شده باشد")
	}
}

// TestBankMehrEqtesad شبکهٔ عامل بانک مهر اقتصاد باید در کاتالوگ باشد و عنوان فارسی
// درست داشته باشد.
func TestBankMehrEqtesad(t *testing.T) {
	if !ValidBank("mehreghtesad") {
		t.Fatal("بانک مهر اقتصاد باید در فهرست بانک‌های عامل مجاز باشد")
	}
	if got := BankTitleFa("mehreghtesad"); got != "بانک مهر اقتصاد" {
		t.Fatalf("عنوان بانک مهر اقتصاد درست نیست: %s", got)
	}
}
