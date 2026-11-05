// آزمون قاعدهٔ «محاسبهٔ فوقالعادهٔ مأموریت».
//
// چرا این آزمون‌ها: این تابع مبنای مبلغی است که به کارمند بابت مأموریت پرداخت می‌شود؛
// اگر ضریب وعده‌ها، جایگزینی سقف خواب با هزینهٔ هتل یا کرایهٔ کیلومتری جابه‌جا شود، پرداخت
// اشتباه می‌شود. هر قاعده با عدد قطعی سنجیده می‌شود.

package rules

import "testing"

// TestMissionAllowanceFullDayMission مأموریت کامل با ضریب‌های پیش‌فرض.
func TestMissionAllowanceFullDayMission(t *testing.T) {
	out := ComputeMissionAllowance(MissionAllowanceInput{
		StartDate: "1405/08/12", StartTime: "06:30", DurationDays: 2,
		PerDiem: 4_500_000,
	})
	if out.DailyRate != 4_500_000 {
		t.Fatalf("نرخ روزانه = %v", out.DailyRate)
	}
	// جمع ضریب‌ها ۱ است، پس با دو روز مأموریت جمع باید برابر نرخ روزانه × ۲ باشد.
	if out.Total != 9_000_000 {
		t.Errorf("جمع قابل پرداخت = %v، انتظار ۹٬۰۰۰٬۰۰۰", out.Total)
	}
	if out.MealsTotal != 7_200_000 {
		t.Errorf("جمع وعده‌ها = %v، انتظار ۷٬۲۰۰٬۰۰۰", out.MealsTotal)
	}
	if out.LodgingCost != 1_800_000 {
		t.Errorf("سهم خواب = %v، انتظار ۱٬۸۰۰٬۰۰۰", out.LodgingCost)
	}
	if len(out.Lines) < 4 {
		t.Errorf("سطرهای تفکیک کم است: %#v", out.Lines)
	}
	if out.NoteFa == "" {
		t.Error("شرح شروع و مدت مأموریت ساخته نشد")
	}
}

// TestMissionAllowanceMealProvidedByHost وعده‌ای که میزبان داده، پرداخت نمی‌شود.
func TestMissionAllowanceMealProvidedByHost(t *testing.T) {
	factors := DefaultMissionFactors()
	factors.Breakfast = 0 // پذیرایی صبحانه با میزبان
	out := ComputeMissionAllowance(MissionAllowanceInput{
		DurationDays: 1, PerDiem: 4_500_000, Factors: factors,
	})
	if out.Total != 4_050_000 {
		t.Errorf("با حذف صبحانه، جمع = %v، انتظار ۴٬۰۵۰٬۰۰۰", out.Total)
	}
	if out.FactorsTotal != 0.9 {
		t.Errorf("جمع عوامل = %v، انتظار ۰٫۹", out.FactorsTotal)
	}
}

// TestMissionAllowanceHotelReplacesLodging اقامت در هتل جای سهم خواب را می‌گیرد.
func TestMissionAllowanceHotelReplacesLodging(t *testing.T) {
	out := ComputeMissionAllowance(MissionAllowanceInput{
		DurationDays: 2, PerDiem: 4_500_000,
		UseHotel: true, HotelCost: 12_000_000,
	})
	if out.LodgingCost != 12_000_000 {
		t.Errorf("هزینهٔ اقامت = %v، انتظار ۱۲٬۰۰۰٬۰۰۰ (هتل، بدون پرداخت دوگانهٔ خواب)", out.LodgingCost)
	}
	// وعده‌ها ۷٬۲۰۰٬۰۰۰ + هتل ۱۲٬۰۰۰٬۰۰۰
	if out.Total != 19_200_000 {
		t.Errorf("جمع = %v، انتظار ۱۹٬۲۰۰٬۰۰۰", out.Total)
	}
	lodgingLine := findLine(out, "lodging")
	if lodgingLine == nil || lodgingLine.Amount != 0 {
		t.Errorf("سطر خواب باید صفر و پرداخت‌نشده باشد: %#v", lodgingLine)
	}
	if findLine(out, "hotel") == nil {
		t.Error("سطر هزینهٔ هتل ساخته نشد")
	}

	guest := ComputeMissionAllowance(MissionAllowanceInput{
		DurationDays: 1, PerDiem: 4_500_000,
		UseGuesthouse: true, GuesthouseCost: 2_000_000,
	})
	if guest.LodgingCost != 2_000_000 || findLine(guest, "guesthouse") == nil {
		t.Errorf("مهمانسرا جای خواب را نگرفت: %#v", guest)
	}
}

// TestMissionAllowancePersonalFacilitiesAndTransport امکانات شخصی و ایاب‌وذهاب.
func TestMissionAllowancePersonalFacilitiesAndTransport(t *testing.T) {
	out := ComputeMissionAllowance(MissionAllowanceInput{
		DurationDays: 2, PerDiem: 4_500_000,
		UsePersonalFacilities: true,
		TaxiCost:              800_000,
		BusCost:               1_200_000,
		MileageKm:             120, MileageRate: 50_000,
	})
	// امکانات شخصی: ۱۰٪ × ۴٬۵۰۰٬۰۰۰ × ۲ = ۹۰۰٬۰۰۰
	if out.Facilities != 900_000 {
		t.Errorf("جبران امکانات شخصی = %v، انتظار ۹۰۰٬۰۰۰", out.Facilities)
	}
	// ایاب‌وذهاب: ۸۰۰٬۰۰۰ + ۱٬۲۰۰٬۰۰۰ + (۱۲۰ × ۵۰٬۰۰۰ = ۶٬۰۰۰٬۰۰۰)
	if out.Transport != 8_000_000 {
		t.Errorf("ایاب‌وذهاب = %v، انتظار ۸٬۰۰۰٬۰۰۰", out.Transport)
	}
	if out.Total != 17_900_000 {
		t.Errorf("جمع = %v، انتظار ۱۷٬۹۰۰٬۰۰۰", out.Total)
	}
	if findLine(out, "mileage") == nil || findLine(out, "taxi") == nil || findLine(out, "bus") == nil {
		t.Error("سطرهای ایاب‌وذهاب ساخته نشد")
	}
}

// TestMissionAllowancePersonalFacilitiesManualAmount مبلغ دستی بر درصد پیش‌فرض مقدم است.
func TestMissionAllowancePersonalFacilitiesManualAmount(t *testing.T) {
	out := ComputeMissionAllowance(MissionAllowanceInput{
		DurationDays: 1, PerDiem: 4_500_000,
		UsePersonalFacilities: true, PersonalFacilitiesAmount: 500_000,
	})
	if out.Facilities != 500_000 {
		t.Errorf("جبران امکانات شخصی = %v، انتظار ۵۰۰٬۰۰۰", out.Facilities)
	}
}

// TestMissionAllowanceMissionFactor ضریب عوامل مأموریت نرخ روزانه را می‌سنجد.
func TestMissionAllowanceMissionFactor(t *testing.T) {
	out := ComputeMissionAllowance(MissionAllowanceInput{
		DurationDays: 1, PerDiem: 4_000_000, MissionFactor: 1.5,
	})
	if out.DailyRate != 6_000_000 {
		t.Errorf("نرخ روزانه با ضریب ۱٫۵ = %v، انتظار ۶٬۰۰۰٬۰۰۰", out.DailyRate)
	}
	if out.Total != 6_000_000 {
		t.Errorf("جمع = %v، انتظار ۶٬۰۰۰٬۰۰۰", out.Total)
	}
}

// TestMissionAllowanceHalfDayAndDefaults نیم‌روز و ضریب‌های ناقص.
func TestMissionAllowanceHalfDayAndDefaults(t *testing.T) {
	out := ComputeMissionAllowance(MissionAllowanceInput{DurationDays: 0.5, PerDiem: 4_500_000})
	if out.Days != 0.5 {
		t.Errorf("مدت = %v، انتظار ۰٫۵", out.Days)
	}
	if out.Total != 2_250_000 {
		t.Errorf("نیم‌روز مأموریت = %v، انتظار ۲٬۲۵۰٬۰۰۰", out.Total)
	}
	// ضریب‌های صفر یعنی «تنظیم نشده»؛ پیش‌فرض به‌کار می‌رود.
	zero := ComputeMissionAllowance(MissionAllowanceInput{DurationDays: 1, PerDiem: 4_500_000, Factors: MissionFactors{}})
	if zero.Total != 4_500_000 {
		t.Errorf("با ضریب‌های صفر باید پیش‌فرض اعمال شود: %v", zero.Total)
	}
}

// TestMissionAllowanceZeroDuration مدت صفر: جمع صفر با یادداشت راهنما.
func TestMissionAllowanceZeroDuration(t *testing.T) {
	out := ComputeMissionAllowance(MissionAllowanceInput{PerDiem: 4_500_000})
	if out.Total != 0 {
		t.Errorf("جمع با مدت صفر = %v", out.Total)
	}
	if out.NoteFa == "" {
		t.Error("یادداشت راهنما برای مدت صفر ساخته نشد")
	}
	if len(out.Lines) != 0 {
		t.Errorf("با مدت صفر سطر تفکیکی نباید باشد: %#v", out.Lines)
	}
}

// findLine یافتن یک سطر تفکیک با کلید.
func findLine(out MissionAllowance, key string) *MissionAllowanceLine {
	for i := range out.Lines {
		if out.Lines[i].Key == key {
			return &out.Lines[i]
		}
	}
	return nil
}
