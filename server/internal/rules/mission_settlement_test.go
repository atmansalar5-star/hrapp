// آزمون «تسویهٔ مأموریت»: اسناد پشتیبان، اعزام پزشکی و مأموریت خارج از کشور.
//
// چرا این آزمون‌ها: این تابع همان چیزی است که به کارمند پرداخت می‌شود. اگر سند تأییدنشده در
// جمع بیاید، یا سهم خواب همراه بیمار اشتباه محاسبه شود، یا مبلغ ارزی با ریالی قاطی شود،
// پرداخت غلط می‌شود؛ پس هر قاعده با عدد قطعی سنجیده می‌شود.

package rules

import "testing"

// TestSettlementDocumentsOnlyApproved قبض‌ها و فاکتورها فقط با تأیید پرداخت می‌شوند.
func TestSettlementDocumentsOnlyApproved(t *testing.T) {
	out := ComputeMissionSettlement(MissionSettlementInput{
		MissionAllowanceInput: MissionAllowanceInput{DurationDays: 1, PerDiem: 4_500_000},
		MissionType:           "administrative",
		Documents: []MissionDocument{
			{Key: "D1", Kind: "taxi", DocNo: "T-1201", DocDate: "1405/08/13", Amount: 350_000, Currency: "IRR", Approved: true, ApprovedBy: "کارشناس مالی"},
			{Key: "D2", Kind: "hotel", DocNo: "H-88", Amount: 7_500_000, Currency: "IRR", Approved: true},
			{Key: "D3", Kind: "taxi", DocNo: "T-1305", Amount: 420_000, Currency: "IRR", Approved: false},
		},
	})
	if out.DocumentsTotal != 7_850_000 {
		t.Errorf("جمع اسناد تأییدشده = %v، انتظار ۷٬۸۵۰٬۰۰۰", out.DocumentsTotal)
	}
	if out.PendingDocuments != 1 {
		t.Errorf("شمار اسناد در گردش = %d، انتظار ۱", out.PendingDocuments)
	}
	// فوقالعادهٔ روزانهٔ یک‌روزه (۴٬۵۰۰٬۰۰۰) + اسناد تأییدشده
	if out.TotalRial != 12_350_000 {
		t.Errorf("جمع ریالی = %v، انتظار ۱۲٬۳۵۰٬۰۰۰", out.TotalRial)
	}
	if findSettlementLine(out, "doc:D3") == nil {
		t.Error("سند تأییدنشده باید در فهرست بماند")
	}
	for _, line := range out.Lines {
		if line.Key == "doc:D3" && line.Amount != 0 {
			t.Errorf("سند تأییدنشده نباید مبلغ داشته باشد: %#v", line)
		}
	}
}

// TestSettlementForeignDualPayment مأموریت خارج از کشور: پرداخت ارزی و ریالی جدا.
func TestSettlementForeignDualPayment(t *testing.T) {
	out := ComputeMissionSettlement(MissionSettlementInput{
		MissionAllowanceInput: MissionAllowanceInput{DurationDays: 5, PerDiem: 4_500_000},
		MissionType:           "foreign",
		Currency:              "EUR",
		FxRate:                900_000,
		FxPerDiem:             120,
		FxAmount:              800,        // بلیت و اقامت خارجی
		RialAmount:            12_000_000, // مسیر داخلی و اسکان ریالی
		Documents: []MissionDocument{
			{Key: "D1", Kind: "air_ticket", DocNo: "TK-77", Amount: 350, Currency: "EUR", Approved: true},
			{Key: "D2", Kind: "taxi", DocNo: "T-88", Amount: 500_000, Currency: "IRR", Approved: true},
		},
	})
	if out.Foreign == nil {
		t.Fatal("بخش ارزی ساخته نشد")
	}
	// ارزی: ۱۲۰ × ۵ = ۶۰۰ + ۸۰۰ سایر + ۳۵۰ سند ارزی = ۱٬۷۵۰ یورو
	if out.TotalFx != 1750 {
		t.Errorf("جمع ارزی = %v، انتظار ۱۷۵۰", out.TotalFx)
	}
	// ریالی: فوقالعادهٔ ۵ روز ۲۲٬۵۰۰٬۰۰۰ + سند تاکسی ۵۰۰٬۰۰۰ + مبلغ ریالی ۱۲٬۰۰۰٬۰۰۰
	if out.TotalRial != 35_000_000 {
		t.Errorf("جمع ریالی = %v، انتظار ۳۵٬۰۰۰٬۰۰۰", out.TotalRial)
	}
	// معادل ریالی: ۳۵٬۰۰۰٬۰۰۰ + ۱٬۷۵۰ × ۹۰۰٬۰۰۰ = ۱٬۶۱۰٬۰۰۰٬۰۰۰
	if out.TotalRialEquivalent != 1_610_000_000 {
		t.Errorf("معادل ریالی = %v، انتظار ۱٬۶۱۰٬۰۰۰٬۰۰۰", out.TotalRialEquivalent)
	}
	if out.Foreign.CurrencyFa != "یورو" || out.Foreign.FxRate != 900_000 {
		t.Errorf("اطلاعات ارزی نادرست: %#v", out.Foreign)
	}
}

// TestSettlementMedicalPatient بیمار: همهٔ اجزا + اسکان واقعی جای سهم خواب.
func TestSettlementMedicalPatient(t *testing.T) {
	out := ComputeMissionSettlement(MissionSettlementInput{
		MissionAllowanceInput: MissionAllowanceInput{DurationDays: 2, PerDiem: 4_500_000},
		MissionType:           "medical",
		Beneficiary:           "patient",
		AccommodationType:     "hospital",
		AccommodationCost:     6_000_000,
	})
	if out.Medical == nil || out.Medical.BeneficiaryFa != "بیمار" {
		t.Fatalf("اطلاعات بیمار نادرست: %#v", out.Medical)
	}
	// سهم خواب (۱٬۸۰۰٬۰۰۰) حذف و اسکان واقعی ۶٬۰۰۰٬۰۰۰ جایش می‌آید.
	if out.Allowance.LodgingCost != 0 {
		t.Errorf("سهم خواب باید حذف شود: %v", out.Allowance.LodgingCost)
	}
	if out.Accommodation != 6_000_000 {
		t.Errorf("هزینهٔ اسکان = %v، انتظار ۶٬۰۰۰٬۰۰۰", out.Accommodation)
	}
	// وعده‌ها ۷٬۲۰۰٬۰۰۰ + اسکان ۶٬۰۰۰٬۰۰۰ = ۱۳٬۲۰۰٬۰۰۰
	if out.TotalRial != 13_200_000 {
		t.Errorf("جمع = %v، انتظار ۱۳٬۲۰۰٬۰۰۰", out.TotalRial)
	}
	if !out.Medical.MealsPayable {
		t.Error("برای بیمار سهم وعده‌ها باید پرداخت شود")
	}
}

// TestSettlementMedicalCompanion همراه بیمار: وعده‌ها پرداخت نمی‌شود، اسکان و ایاب‌وذهاب می‌شود.
func TestSettlementMedicalCompanion(t *testing.T) {
	out := ComputeMissionSettlement(MissionSettlementInput{
		MissionAllowanceInput: MissionAllowanceInput{
			DurationDays: 3, PerDiem: 4_500_000, TaxiCost: 700_000,
		},
		MissionType:       "medical",
		Beneficiary:       "companion",
		AccommodationType: "guesthouse",
		AccommodationCost: 4_500_000,
		CompanionCount:    1,
	})
	if out.Medical == nil || out.Medical.BeneficiaryFa != "همراه بیمار" {
		t.Fatalf("اطلاعات همراه نادرست: %#v", out.Medical)
	}
	if out.Allowance.MealsTotal != 0 {
		t.Errorf("سهم وعده‌ها برای همراه باید صفر باشد: %v", out.Allowance.MealsTotal)
	}
	if out.Medical.MealsPayable {
		t.Error("با درصد صفر، وعده‌های همراه پرداخت نمی‌شود")
	}
	// ایاب‌وذهاب ۷۰۰٬۰۰۰ + اسکان ۴٬۵۰۰٬۰۰۰
	if out.TotalRial != 5_200_000 {
		t.Errorf("جمع = %v، انتظار ۵٬۲۰۰٬۰۰۰", out.TotalRial)
	}

	// با مصوبهٔ درصدی، سهمی از وعده‌ها پرداخت می‌شود: ۵۰٪ از ۱۰٬۸۰۰٬۰۰۰ = ۵٬۴۰۰٬۰۰۰
	withShare := ComputeMissionSettlement(MissionSettlementInput{
		MissionAllowanceInput: MissionAllowanceInput{DurationDays: 3, PerDiem: 4_500_000},
		MissionType:           "medical",
		Beneficiary:           "companion",
		CompanionSharePercent: 50,
	})
	if withShare.Allowance.MealsTotal != 5_400_000 {
		t.Errorf("سهم وعده‌های همراه با مصوبهٔ ۵۰٪ = %v، انتظار ۵٬۴۰۰٬۰۰۰", withShare.Allowance.MealsTotal)
	}
	if withShare.Medical == nil || !withShare.Medical.MealsPayable {
		t.Error("با مصوبهٔ درصدی، وعده‌ها باید پرداخت‌شدنی باشند")
	}
}

// TestSettlementAdministrativeUnchanged مأموریت اداری رفتار پیشین را حفظ می‌کند.
func TestSettlementAdministrativeUnchanged(t *testing.T) {
	in := MissionAllowanceInput{DurationDays: 2, PerDiem: 4_500_000}
	out := ComputeMissionSettlement(MissionSettlementInput{MissionAllowanceInput: in, MissionType: "administrative"})
	plain := ComputeMissionAllowance(in)
	if out.TotalRial != plain.Total {
		t.Errorf("تسویهٔ اداری = %v، فوقالعادهٔ روزانه = %v", out.TotalRial, plain.Total)
	}
	if out.Medical != nil || out.Foreign != nil {
		t.Error("برای مأموریت اداری بخش پزشکی/ارزی نباید ساخته شود")
	}
	if out.MissionTypeFa != "مأموریت اداری" {
		t.Errorf("عنوان نوع مأموریت = %q", out.MissionTypeFa)
	}
}

// TestSettlementZeroDuration مدت صفر: بدون سطر و جمع صفر.
func TestSettlementZeroDuration(t *testing.T) {
	out := ComputeMissionSettlement(MissionSettlementInput{
		MissionAllowanceInput: MissionAllowanceInput{PerDiem: 4_500_000},
		MissionType:           "administrative",
	})
	if out.TotalRial != 0 || out.TotalFx != 0 {
		t.Errorf("مدت صفر باید جمع صفر بدهد: %v / %v", out.TotalRial, out.TotalFx)
	}
}

// findSettlementLine یافتن یک سطر در تفکیک تسویه (مبلغش در آزمون سنجیده می‌شود).
func findSettlementLine(out MissionSettlement, key string) *MissionAllowanceLine {
	for i := range out.Lines {
		if out.Lines[i].Key == key {
			return &out.Lines[i]
		}
	}
	return nil
}
