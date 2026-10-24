// آزمون دامنهٔ ماژول «staff_services» (personnel_test).

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

func TestValidateDependent(t *testing.T) {
	base := contracts.DependentInput{
		EmployeeID: "EMP-001", FullName: "فاطمه احمدی", Relation: RelationSpouse,
		NationalID: "0079845630", BirthDate: "1370/05/12", CoverageStart: "1404/01/01",
	}
	if errs := ValidateDependent(base); len(errs) != 0 {
		t.Fatalf("تکفل درست نباید خطا بدهد: %v", errs)
	}

	cases := []struct {
		name  string
		mut   func(in *contracts.DependentInput)
		field string
	}{
		{"بدون کارمند", func(in *contracts.DependentInput) { in.EmployeeID = "" }, "employeeId"},
		{"نام کوتاه", func(in *contracts.DependentInput) { in.FullName = "ب" }, "fullName"},
		{"نسبت نامعتبر", func(in *contracts.DependentInput) { in.Relation = "cousin" }, "relation"},
		{"کد ملی نامعتبر", func(in *contracts.DependentInput) { in.NationalID = "1234567890" }, "nationalId"},
		{"تاریخ تولد نامعتبر", func(in *contracts.DependentInput) { in.BirthDate = "1370/13/12" }, "birthDate"},
		{"تاریخ پوشش نامعتبر", func(in *contracts.DependentInput) { in.CoverageStart = "1404/13/01" }, "coverageStart"},
	}
	for _, c := range cases {
		in := base
		c.mut(&in)
		if _, ok := ValidateDependent(in)[c.field]; !ok {
			t.Fatalf("%s: انتظار خطای «%s» بود", c.name, c.field)
		}
	}

	// ارقام فارسی و عربی هم باید پذیرفته شوند (کاربر فارسی‌زبان همین را وارد می‌کند).
	persian := base
	persian.NationalID = "۰۰۷۹۸۴۵۶۳۰"
	if errs := ValidateDependent(persian); len(errs) != 0 {
		t.Fatalf("کد ملی با ارقام فارسی باید پذیرفته شود: %v", errs)
	}
}

func TestValidateFlight(t *testing.T) {
	today := date(t, "1404/07/07")
	base := contracts.FlightRequestInput{
		EmployeeID: "EMP-001", OriginFa: "ماهشهر", DestinationFa: "تهران",
		DepartDate: "1404/07/20", ReturnDate: "1404/07/25", Purpose: PurposeMission,
		Passengers: 1, EstimatedCost: 15_000_000,
	}
	if errs := ValidateFlight(base, today); len(errs) != 0 {
		t.Fatalf("فرم درست نباید خطا بدهد: %v", errs)
	}

	cases := []struct {
		name  string
		mut   func(in *contracts.FlightRequestInput)
		field string
	}{
		{"تاریخ گذشته", func(in *contracts.FlightRequestInput) { in.DepartDate = "1404/07/01" }, "departDate"},
		{"تاریخ نامعتبر (ماه ۱۳)", func(in *contracts.FlightRequestInput) { in.DepartDate = "1404/13/01" }, "departDate"},
		{"تاریخ نامعتبر (بدون روز)", func(in *contracts.FlightRequestInput) { in.DepartDate = "1404/07" }, "departDate"},
		{"مبدأ تکراری", func(in *contracts.FlightRequestInput) { in.DestinationFa = "ماهشهر" }, "destinationFa"},
		{"بدون مقصد", func(in *contracts.FlightRequestInput) { in.DestinationFa = "" }, "destinationFa"},
		{"هدف نامعتبر", func(in *contracts.FlightRequestInput) { in.Purpose = "picnic" }, "purpose"},
		{"بازگشت پیش از پرواز", func(in *contracts.FlightRequestInput) { in.ReturnDate = "1404/07/18" }, "returnDate"},
		{"بدون مسافر", func(in *contracts.FlightRequestInput) { in.Passengers = 0 }, "passengers"},
		{"مسافر بیش از حد", func(in *contracts.FlightRequestInput) { in.Passengers = 40 }, "passengers"},
		{"هزینهٔ منفی", func(in *contracts.FlightRequestInput) { in.EstimatedCost = -1 }, "estimatedCost"},
	}
	for _, c := range cases {
		in := base
		c.mut(&in)
		if _, ok := ValidateFlight(in, today)[c.field]; !ok {
			t.Fatalf("%s: انتظار خطای «%s» بود، خطاها: %v", c.name, c.field, ValidateFlight(in, today))
		}
	}
}

func TestFlightWorkflowTransitions(t *testing.T) {
	targets := map[string]string{
		FlightActionApprove: FlightApproved,
		FlightActionReject:  FlightRejected,
		FlightActionIssue:   FlightIssued,
		FlightActionSettle:  FlightSettled,
		FlightActionCancel:  FlightCancelled,
	}
	for action, target := range targets {
		if _, ok := TargetOfFlightAction(action); !ok {
			t.Fatalf("اقدام «%s» ناشناخته است", action)
		}
		_ = target
	}
	if _, ok := TargetOfFlightAction("archive"); ok {
		t.Fatal("اقدام ناشناخته نباید وضعیت مقصد بدهد")
	}

	allowed := [][2]string{
		{FlightPending, FlightApproved},
		{FlightPending, FlightRejected},
		{FlightPending, FlightCancelled},
		{FlightApproved, FlightIssued},
		{FlightIssued, FlightSettled},
	}
	for _, c := range allowed {
		if !CanTransitionFlight(c[0], c[1]) {
			t.Fatalf("انتقال %s → %s باید مجاز باشد", c[0], c[1])
		}
	}
	denied := [][2]string{
		{FlightPending, FlightIssued},
		{FlightIssued, FlightCancelled},
		{FlightSettled, FlightCancelled},
		{FlightRejected, FlightApproved},
	}
	for _, c := range denied {
		if CanTransitionFlight(c[0], c[1]) {
			t.Fatalf("انتقال %s → %s نباید مجاز باشد", c[0], c[1])
		}
	}
}

func TestDependentToContractComputesCoverage(t *testing.T) {
	today := date(t, "1404/07/07")
	d := Dependent{
		ID: "DEP-1", EmployeeID: "EMP-001", EmployeeName: "سارا محمدی", FullName: "فاطمه محمدی",
		Relation: RelationSpouse, NationalID: "0079845630", CoverageStart: "1403/07/07",
		Insurance: true, Status: DependentApproved,
	}
	out := d.ToContract(today)
	if out.RelationFa != "همسر" || out.StatusFa != "تکفل تأییدشده" {
		t.Fatalf("عنوان‌های فارسی تکفل درست پر نشده‌اند: %+v", out)
	}
	if out.CoverageMonths < 11 || out.CoverageMonths > 13 {
		t.Fatalf("مدت پوشش باید حدود ۱۲ ماه باشد، %d شد", out.CoverageMonths)
	}
	if out.InsuranceFa == "" {
		t.Fatal("عنوان پوشش بیمه پر نشده است")
	}
}
