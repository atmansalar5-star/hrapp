// آزمون دامنهٔ ماژول «contracts» (employment_test).

package domain

import (
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

func date(t *testing.T, s string) calendar.Date {
	t.Helper()
	d, err := calendar.Parse(s)
	if err != nil {
		t.Fatalf("تاریخ آزمون نامعتبر است: %s", s)
	}
	return d
}

func TestValidateContractRules(t *testing.T) {
	base := contracts.ContractInput{
		EmployeeID: "EMP-001", Kind: KindFixedTerm,
		StartDate: "1404/01/01", EndDate: "1405/01/01", MonthlySalary: 30_000_000,
	}
	if errs := Validate(base); len(errs) != 0 {
		t.Fatalf("قرارداد درست نباید خطا بدهد: %v", errs)
	}

	cases := []struct {
		name  string
		mut   func(in *contracts.ContractInput)
		field string
	}{
		{"بدون کارمند", func(in *contracts.ContractInput) { in.EmployeeID = "" }, "employeeId"},
		{"نوع نامعتبر", func(in *contracts.ContractInput) { in.Kind = "temporary" }, "kind"},
		{"بدون تاریخ پایان مدت موقت", func(in *contracts.ContractInput) { in.EndDate = "" }, "endDate"},
		{"پایان پیش از شروع", func(in *contracts.ContractInput) { in.EndDate = "1403/12/29" }, "endDate"},
		{"بیش از ۵ سال", func(in *contracts.ContractInput) { in.EndDate = "1410/01/01" }, "endDate"},
		{"مبلغ صفر", func(in *contracts.ContractInput) { in.MonthlySalary = 0 }, "monthlySalary"},
		{"تاریخ شروع نامعتبر (ماه ۱۳)", func(in *contracts.ContractInput) { in.StartDate = "1404/13/01" }, "startDate"},
		{"تاریخ شروع نامعتبر (روز ۳۲)", func(in *contracts.ContractInput) { in.StartDate = "1404/01/32" }, "startDate"},
		{"دائم با تاریخ پایان", func(in *contracts.ContractInput) { in.Kind = KindPermanent }, "endDate"},
	}
	for _, c := range cases {
		in := base
		c.mut(&in)
		errs := Validate(in)
		if _, ok := errs[c.field]; !ok {
			t.Fatalf("%s: انتظار خطای «%s» بود، خطاها: %v", c.name, c.field, errs)
		}
	}

	permanent := contracts.ContractInput{
		EmployeeID: "EMP-001", Kind: KindPermanent, StartDate: "1404/01/01", MonthlySalary: 30_000_000,
	}
	if errs := Validate(permanent); len(errs) != 0 {
		t.Fatalf("قرارداد دائم درست نباید خطا بدهد: %v", errs)
	}
}

func TestTransitionRules(t *testing.T) {
	allowedCases := [][2]string{
		{StatusDraft, StatusPending},
		{StatusPending, StatusActive},
		{StatusPending, StatusRejected},
		{StatusRejected, StatusPending},
		{StatusActive, StatusTerminated},
		{StatusActive, StatusExpired},
	}
	for _, c := range allowedCases {
		if !CanTransition(c[0], c[1]) {
			t.Fatalf("انتقال %s → %s باید مجاز باشد", c[0], c[1])
		}
	}
	denied := [][2]string{
		{StatusDraft, StatusActive},
		{StatusActive, StatusPending},
		{StatusTerminated, StatusActive},
		{StatusExpired, StatusActive},
		{StatusRejected, StatusActive},
	}
	for _, c := range denied {
		if CanTransition(c[0], c[1]) {
			t.Fatalf("انتقال %s → %s نباید مجاز باشد", c[0], c[1])
		}
	}
}

func TestOverlapDetection(t *testing.T) {
	fixed := func(id, start, end, status string) Contract {
		return Contract{ID: id, EmployeeID: "EMP-001", Kind: KindFixedTerm, StartDate: start, EndDate: end, Status: status}
	}
	cases := []struct {
		name string
		a, b Contract
		want bool
	}{
		{"هم‌پوشان", fixed("a", "1404/01/01", "1404/12/29", StatusActive), fixed("b", "1404/06/01", "1405/06/01", StatusActive), true},
		{"پیوسته (بدون هم‌پوشانی)", fixed("a", "1404/01/01", "1404/12/29", StatusActive), fixed("b", "1404/12/30", "1405/12/29", StatusActive), false},
		{"جدا", fixed("a", "1404/01/01", "1404/06/01", StatusActive), fixed("b", "1405/01/01", "1405/06/01", StatusActive), false},
	}
	for _, c := range cases {
		if got := Overlaps(c.a, c.b); got != c.want {
			t.Fatalf("%s: انتظار %v بود، %v شد", c.name, c.want, got)
		}
	}

	permanent := Contract{ID: "p", EmployeeID: "EMP-001", Kind: KindPermanent, StartDate: "1400/01/01", Status: StatusActive}
	if !Overlaps(permanent, fixed("b", "1404/06/01", "1405/06/01", StatusActive)) {
		t.Fatal("قرارداد دائم با قرارداد موقت هم‌پوشان است")
	}
	if Overlaps(permanent, fixed("c", "1390/01/01", "1395/01/01", StatusTerminated)) {
		t.Fatal("قرارداد موقت پایان‌یافته پیش از شروع قرارداد دائم هم‌پوشانی ندارد")
	}
}

func TestRenewWindows(t *testing.T) {
	// سال ۱۴۰۴ کبیسه نیست، پس ۱۴۰۴/۱۲/۲۹ پایان سال است و تمدید از ۱۴۰۵/۰۱/۰۱ آغاز می‌شود.
	c := Contract{ID: "CT-1", Kind: KindFixedTerm, StartDate: "1404/01/01", EndDate: "1404/12/29", Status: StatusActive}
	start, ok := NextStart(c)
	if !ok || start != "1405/01/01" {
		t.Fatalf("شروع تمدید باید روز پس از پایان باشد، «%s» شد", start)
	}
	end, ok := DefaultRenewEnd(c)
	if !ok || end == "" {
		t.Fatal("پایان پیش‌فرض تمدید محاسبه نشد")
	}
	if days := ContractDays(KindFixedTerm, start, end); days <= 350 || days > 370 {
		t.Fatalf("طول تمدید باید همان طول قرارداد قبلی باشد، %d روز شد", days)
	}

	if errs := ValidateRenew(Contract{Kind: KindPermanent, Status: StatusActive}, contracts.ContractRenewInput{}); errs["kind"] == "" {
		t.Fatal("تمدید قرارداد دائم باید خطا بدهد")
	}
	if errs := ValidateRenew(Contract{Kind: KindFixedTerm, Status: StatusDraft}, contracts.ContractRenewInput{}); errs["status"] == "" {
		t.Fatal("تمدید پیش‌نویس باید خطا بدهد")
	}
}

func TestConversionRequiresService(t *testing.T) {
	today := date(t, "1404/07/01")
	short := Contract{Kind: KindFixedTerm, Status: StatusActive, StartDate: "1404/06/01"}
	if errs := ValidateConversion(short, today); errs["serviceDays"] == "" {
		t.Fatal("تبدیل با کمتر از یک سال سابقه باید خطا بدهد")
	}
	long := Contract{Kind: KindFixedTerm, Status: StatusActive, StartDate: "1402/01/01"}
	if errs := ValidateConversion(long, today); len(errs) != 0 {
		t.Fatalf("تبدیل با سابقهٔ کافی نباید خطا بدهد: %v", errs)
	}
	if errs := ValidateConversion(Contract{Kind: KindPermanent, Status: StatusActive}, today); errs["kind"] == "" {
		t.Fatal("تبدیل قرارداد دائم باید خطا بدهد")
	}
}

func TestServiceDaysAndNationalID(t *testing.T) {
	today := date(t, "1404/01/11")
	if days := ServiceDays("1404/01/01", today); days != 10 {
		t.Fatalf("سابقهٔ خدمت باید ۱۰ روز باشد، %d شد", days)
	}
	if days := ServiceDays("نامعتبر", today); days != 0 {
		t.Fatalf("تاریخ نامعتبر باید سابقهٔ صفر بدهد، %d شد", days)
	}
	if !IsValidNationalID("0012345679") {
		t.Fatal("کد ملی نمونه باید معتبر باشد")
	}
	if IsValidNationalID("0012345678") || IsValidNationalID("1111111111") || IsValidNationalID("123") {
		t.Fatal("کد ملی نامعتبر نباید پذیرفته شود")
	}
}
