// آزمون دامنهٔ ماژول «payroll» (service_bonus_test).

package domain_test

import (
	"math"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/modules/payroll/domain"
)

// referenceInput همان پروندهٔ واقعی صورت‌حساب مرجع است («محاسبه پاداش پایان خدمت.xlsx»):
// پاداش سنوات خدمت کارکنان رسمی صنعت نفت — سابقهٔ شمال ۲ سال و ۲ ماه، سابقهٔ جنوب
// ۳۴ سال و ۶ ماه و ۱۵ روز، پایهٔ حقوق ۱۹۱٬۰۹۸٬۴۰۰، ترمیم حقوق ۱۴٬۴۸۷٬۰۰۰ ریال.
func referenceInput() contracts.ServiceBonusInput {
	return contracts.ServiceBonusInput{
		EmployeeID:   "REF-632801",
		BaseDate:     "1403/08/22",
		BaseSalary:   191_098_400,
		SalaryRepair: 14_487_000,
		North:        contracts.ServiceBonusTenure{Years: 2, Months: 2, Days: 0},
		South:        contracts.ServiceBonusTenure{Years: 34, Months: 6, Days: 15},
		Rates: contracts.ServiceBonusRates{
			BadClimate: 0.25, Special: 0.30, Workshop: 0.10,
			AttractionPublic: 0.08, PersonalBase: 0.38, JobSpecialty: 0.27,
		},
	}
}

func TestComputeServiceBonusMatchesReferenceSheet(t *testing.T) {
	got := domain.ComputeServiceBonus(referenceInput())

	// هر عدد در برابر همان سلول صورت‌حساب مرجع سنجیده می‌شود؛ رواداری ۱ ریال برای
	// گرد کردن اعشاری شناور کافی است.
	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"سابقه شمال (A7)", got.NorthYears, 2.1666666667},
		{"سابقه جنوب (A8)", got.SouthYears, 34.5410958904},
		{"سابقه جنوب گردشده (C20)", got.SouthYearsUsed, 34.54},
		{"جمع ستون شمال (D18)", got.NorthTotal, 463_377_141.6},
		{"جمع ستون جنوب (A18)", got.SouthTotal, 575_599_677},
		{"پاداش حق جذب شمال (D19)", got.AttractionBonusNorth, 654_114_486.13},
		{"پاداش حق جذب جنوب (D20)", got.AttractionBonusSouth, 12_909_393_803.68},
		{"پاداش حق جذب ۸ درصد (A21)", got.AttractionBonus, 13_563_508_289.81},
		{"پاداش پایه شخصی شمال (A22)", got.PersonalBonusNorth, 349_869_320.67},
		{"پاداش پایه شخصی جنوب (A23)", got.PersonalBonusSouth, 6_972_040_242.84},
		{"مبلغ پاداش سنوات (A24)", got.TotalRial, 20_885_417_853.32},
		{"نیمی از پاداش (A32)", got.HalfTotalRial, 10_442_708_926.66},
	}
	for _, c := range cases {
		if math.Abs(c.got-c.want) > 1 {
			t.Errorf("%s: got %.4f, want %.4f", c.name, c.got, c.want)
		}
	}
}

func TestComputeServiceBonusLineAmountsMatchReference(t *testing.T) {
	got := domain.ComputeServiceBonus(referenceInput())
	find := func(lines []contracts.ServiceBonusLine, title string) float64 {
		for _, l := range lines {
			if l.TitleFa == title {
				return l.AmountRial
			}
		}
		t.Fatalf("سطر «%s» در صورت‌حساب نیست", title)
		return 0
	}
	if v := find(got.SouthLines, "بدی آب و هوا"); math.Abs(v-47_774_600) > 1 {
		t.Errorf("بدی آب و هوا: got %.2f, want 47774600", v)
	}
	if v := find(got.SouthLines, "ویژه"); math.Abs(v-71_661_900) > 1 {
		t.Errorf("ویژه جنوب: got %.2f, want 71661900", v)
	}
	if v := find(got.NorthLines, "ویژه"); math.Abs(v-57_329_520) > 1 {
		t.Errorf("ویژه شمال: got %.2f, want 57329520", v)
	}
	if v := find(got.SouthLines, "جذب (پایه شخصی)"); math.Abs(v-118_003_262) > 1 {
		t.Errorf("جذب پایه شخصی جنوب: got %.2f, want 118003262", v)
	}
	if v := find(got.NorthLines, "جذب (پایه شخصی)"); math.Abs(v-94_402_609.6) > 1 {
		t.Errorf("جذب پایه شخصی شمال: got %.2f, want 94402609.6", v)
	}
	if v := find(got.SouthLines, "تخصصی شغل (مدرک)"); math.Abs(v-83_844_423) > 1 {
		t.Errorf("تخصصی شغل جنوب: got %.2f, want 83844423", v)
	}
	if v := find(got.NorthLines, "تخصصی شغل (مدرک)"); math.Abs(v-67_075_538.4) > 1 {
		t.Errorf("تخصصی شغل شمال: got %.2f, want 67075538.4", v)
	}
}

func TestRatesFromParamsFallsBackToIndustryDefaults(t *testing.T) {
	empty := domain.RatesFromParams(map[string]string{})
	if empty.BadClimate != 0.25 || empty.Special != 0.30 || empty.JobSpecialty != 0.27 {
		t.Fatalf("پیش‌فرض‌های صنعت نفت اعمال نشد: %+v", empty)
	}
	custom := domain.RatesFromParams(map[string]string{
		domain.ParamBadClimate: "0.20",
		domain.ParamSpecial:    "0.35",
		domain.ParamWorkshop:   "مقدار نامعتبر",
	})
	if custom.BadClimate != 0.20 || custom.Special != 0.35 {
		t.Fatalf("مقدار پارامتر خوانده نشد: %+v", custom)
	}
	if custom.Workshop != 0.10 {
		t.Fatalf("مقدار نامعتبر باید به پیش‌فرض برگردد: %+v", custom)
	}
}

func TestTenureHelpers(t *testing.T) {
	if v := domain.TenureYears(contracts.ServiceBonusTenure{Years: 34, Months: 6, Days: 15}); math.Abs(v-34.5410958904) > 0.0000001 {
		t.Fatalf("تبدیل سابقه به سال اعشاری نادرست است: %v", v)
	}
	got := domain.AddTenures(
		contracts.ServiceBonusTenure{Years: 2, Months: 2},
		contracts.ServiceBonusTenure{Years: 34, Months: 6, Days: 15},
	)
	if got.Years != 36 || got.Months != 8 || got.Days != 15 {
		t.Fatalf("جمع سابقه نادرست است: %+v (انتظار ۳۶ سال و ۸ ماه و ۱۵ روز)", got)
	}
	if v := domain.Truncate2(34.5410958904); v != 34.54 {
		t.Fatalf("گرد کردن رو به پایین نادرست است: %v", v)
	}
}
