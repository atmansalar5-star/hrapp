package rules

import "testing"

// TestOilPayslipMatchesSamplePayslip راستی‌آزمایی زنجیرهٔ محاسبه با یک فیش واقعی صندوق نفت.
//
// دادهٔ مبنا همان فیش پیوست است: پایهٔ حقوق ۶۲۶٫۵۱۰٫۶۰۰ ریال، محل سکونت با درصدهای جدول
// ۱ و ۲ برابر ۲۷٫۵٪ (محرومیت) و ۱۶٫۵٪ (صرف وقت روزکار) که در همان فیش آمده است. هر عدد در
// این آزمون از خودِ فیش خوانده شده است؛ اگر روزی قاعده یا گرد کردن عوض شود، همین آزمون
// می‌شکند.
func TestOilPayslipMatchesSamplePayslip(t *testing.T) {
	const base = int64(626_510_600)
	in := OilPayInput{
		BaseSalary: base,
		Residence:  "jarrahi",
		// درصد «فوقالعادهٔ تخصصی» همان فیش (۱۱۹٫۲٪ مبنای جذب) دستی گذاشته می‌شود؛ در حالت
		// جدولی، این قلم از جمع جدول ۳ و ضریب مدرک ساخته می‌شود.
		AllowancePercents: map[string]float64{OilAllowanceAttraction: 119.2},
	}
	slip := ComputeOilPayslip(in, DefaultOilPayParams())

	want := []struct {
		key    string
		title  string
		amount int64
	}{
		{"baseSalary", "پایهٔ حقوق", 626_510_600},
		{OilAllowanceClimate, "فوقالعادهٔ گرمسیری", 156_627_650},
		{"allowanceBase", "مبنای فوقالعاده‌ها", 783_138_250},
		{OilAllowanceSpecial, "فوقالعادهٔ ویژه", 234_941_475},
		{OilAllowanceDeprivation, "محرومیت از تسهیلات", 215_363_019},
		{"workshop", "فوقالعادهٔ کارگاهی", 78_313_825},
		{OilAllowanceTimeConsumed, "جبران صرف وقت", 129_217_811},
		{"production", "مراکز تولیدی فرآورشی", 234_941_475},
		{OilAllowanceAttraction, "فوقالعادهٔ تخصصی", 1_213_551_032},
	}
	for _, w := range want {
		line, ok := oilLineByKey(slip, w.key)
		if !ok {
			t.Fatalf("سطر %s در فیش نیست", w.title)
		}
		if line.Amount != w.amount {
			t.Errorf("%s: مبلغ %d است، فیش نمونه %d می‌دهد", w.title, line.Amount, w.amount)
		}
	}
	if slip.AttractionBase != 1_018_079_725 {
		t.Errorf("مبنای جذب باید ۱٫۰۱۸٫۰۷۹٫۷۲۵ باشد، %d شد", slip.AttractionBase)
	}
}

// TestOilPayslipCoefficientsAreOverridable هر ضریب با بازنویسی همان محاسبه عوض می‌شود و
// مبنای محاسبه هم بی‌تغییر نمی‌ماند (تا فرم فیش بتواند پیش از اجرا آزمایش کند).
func TestOilPayslipCoefficientsAreOverridable(t *testing.T) {
	base := int64(100_000_000)
	plain := ComputeOilPayslip(OilPayInput{BaseSalary: base}, DefaultOilPayParams())

	custom := ComputeOilPayslip(OilPayInput{
		BaseSalary: base,
		AllowancePercents: map[string]float64{
			OilAllowanceSpecial: 35, // ویژه ۳۵٪ به‌جای ۳۰٪
			"workshop":          12,
		},
		ClimatePercent: 30, // گرمسیری ۳۰٪ به‌جای ۲۵٪
	}, DefaultOilPayParams())

	if custom.Climate != RoundHalfUp(float64(base)*0.30) {
		t.Errorf("درصد گرمسیری دستی اعمال نشد: %d", custom.Climate)
	}
	if custom.Special != RoundHalfUp(float64(custom.AllowanceBase)*0.35) {
		t.Errorf("درصد ویژهٔ دستی اعمال نشد: %d", custom.Special)
	}
	if custom.Gross <= plain.Gross {
		t.Errorf("با درصدهای بالاتر، جمع ناخالص باید بیشتر شود: %d در برابر %d", custom.Gross, plain.Gross)
	}
}

// TestOilPayParamsDefaultsAreDocumented مطمئن می‌شود پیش‌فرض‌های موتور همان زنجیرهٔ
// آیین‌نامه است (۲۵٪ گرمسیری، ۳۰٪ ویژه، کسور سهم کارمند ۶٪ و کارفرما ۱۹٪).
func TestOilPayParamsDefaultsAreDocumented(t *testing.T) {
	params := DefaultOilPayParams()
	if params.ClimatePercent != 25 {
		t.Errorf("گرمسیری باید ۲۵٪ باشد: %v", params.ClimatePercent)
	}
	special := 0.0
	for _, a := range params.Allowances {
		if a.Key == OilAllowanceSpecial {
			special = a.Percent
		}
	}
	if special != 30 {
		t.Errorf("فوقالعادهٔ ویژه باید ۳۰٪ باشد: %v", special)
	}
	if params.FundEmployeePercent != 6 || params.FundEmployerPercent != 19 {
		t.Errorf("کسور صندوق باید ۶٪ و ۱۹٪ باشد: %v / %v",
			params.FundEmployeePercent, params.FundEmployerPercent)
	}
}

// oilLineByKey یافتن یک سطر فیش با کلید.
func oilLineByKey(slip OilPayslip, key string) (OilPayLine, bool) {
	for _, line := range slip.Lines {
		if line.Key == key {
			return line, true
		}
	}
	return OilPayLine{}, false
}

// TestOilMahshahrMatchesSamplePayslip جدول ۱ و ۲ برای ماهشهر با فیش رسمی هم‌خوان شد:
// محرومیت ۲۷٫۵٪ و صرف وقت روزکار ۱۶٫۵٪ (پیش از این ۲۵٪ و ۹٫۵٪ بود).
func TestOilMahshahrMatchesSamplePayslip(t *testing.T) {
	slip := ComputeOilPayslip(OilPayInput{
		BaseSalary: 626_510_600,
		Residence:  "mahshahr",
	}, DefaultOilPayParams())

	deprivation, ok := oilLineByKey(slip, OilAllowanceDeprivation)
	if !ok || deprivation.Amount != 215_363_019 || deprivation.Percent != 27.5 {
		t.Fatalf("محرومیت ماهشهر باید ۲۱۵٬۳۶۳٬۰۱۹ (۲۷٫۵٪) باشد: %+v", deprivation)
	}
	timeConsumed, ok := oilLineByKey(slip, OilAllowanceTimeConsumed)
	if !ok || timeConsumed.Amount != 129_217_811 || timeConsumed.Percent != 16.5 {
		t.Fatalf("صرف وقت ماهشهر باید ۱۲۹٬۲۱۷٬۸۱۱ (۱۶٫۵٪) باشد: %+v", timeConsumed)
	}
}

// TestOilPayslipReproducesSampleNet بازتولید کامل فیش رسمی: با ثبت مبلغ اقلام ثابت،
// مبنای کسور مصوبه، پایهٔ تشویقی ایثارگر و سطرهای دستی بندهای ۲ و ۳، خالص فیش سامانه
// باید عیناً همان ۷۶۸٬۱۶۷٬۱۷۰ ریال فیش پیوست باشد.
func TestOilPayslipReproducesSampleNet(t *testing.T) {
	in := OilPayInput{
		BaseSalary: 626_510_600,
		Residence:  "jarrahi",
		AllowancePercents: map[string]float64{
			OilAllowanceAttraction: 119.2,
		},
		AllowanceAmounts: map[string]int64{
			OilAllowanceFamily:   25_208_250,
			OilAllowanceHousing:  50_112_000,
			OilAllowanceChildren: 61_897_554,
			OilAllowanceExtra:    30_597_600,
		},
		Veteran: &VeteranEntitlement{Approved: true, IsVeteran: true, IncentiveSteps: 1},
		ExtraLines: []OilExtraLine{
			// بند ۱ — کسور قانونی که مدل عمومی ندارند.
			{"بیمهٔ صندوق تأمین آتیه — سهم کارمند", 5_090_399, OilSideDebit, OilBandLegal},
			{"بازنشستگی فوق‌العادهٔ تخصصی شغل", 49_580_483, OilSideDebit, OilBandLegal},
			{"بازنشستگی سنوات ایثارگر (م ۱۷۰۱)", 1_485, OilSideDebit, OilBandLegal},
			{"مالیات بر درآمد (سطر اول)", 26_698_006, OilSideDebit, OilBandLegal},
			{"مالیات بر درآمد (سطر دوم)", 366_612_479, OilSideDebit, OilBandLegal},
			// بند ۲ — تطبیق جهت رعایت سقف.
			{"کسر مازاد بر سقف ماه جاری", 64_045_843, OilSideDebit, OilBandCeiling},
			{"کسر مازاد بر سقف ماه جاری", 879_466_642, OilSideDebit, OilBandCeiling},
			{"تطبیق مازاد بر سقف", 263_839_993, OilSideDebit, OilBandCeiling},
			{"تطبیق مازاد بر سقف", 19_213_753, OilSideDebit, OilBandCeiling},
			// بند ۳ — سایر پرداخت‌ها.
			{"پاداش تحصیلی", 31_172_904, OilSideCredit, OilBandOther},
			{"هزینهٔ درمان عهدهٔ سازمان بهداشت", 75_360_000, OilSideCredit, OilBandOther},
			{"هزینهٔ درمان عهدهٔ سازمان بهداشت", 99_999_999, OilSideCredit, OilBandOther},
			{"هزینهٔ درمان عهدهٔ سازمان بهداشت", 164_001, OilSideCredit, OilBandOther},
			{"هزینهٔ درمان عهدهٔ سازمان بهداشت", 34_872_000, OilSideCredit, OilBandOther},
			{"برگشت ۱۹٪ فوق‌العادهٔ تخصصی شغل", 156_987_894, OilSideCredit, OilBandOther},
			{"برگشت ۵۵۰۷ سنوات ایثارگر (م ۱۷۰۱)", 4_705, OilSideCredit, OilBandOther},
			{"ترمیم حقوق کارمندان", 20_862_000, OilSideCredit, OilBandOther},
			// بند ۳ — سایر کسورات.
			{"اقساط وام خرید مسکن", 83_290, OilSideDebit, OilBandOther},
			{"وام درمانی خارج از کشور", 28_000_000, OilSideDebit, OilBandOther},
			{"اقساط وام خرید ۲ مسکن", 311_076, OilSideDebit, OilBandOther},
			{"پس‌انداز سهم کارمند", 65_783_613, OilSideDebit, OilBandOther},
			{"هزینهٔ آب و برق منازل", 198_900, OilSideDebit, OilBandOther},
			{"حق بیمهٔ وام خرید مسکن", 4_164, OilSideDebit, OilBandOther},
			{"حق بیمهٔ وام خرید ۲ مسکن", 15_554, OilSideDebit, OilBandOther},
			{"حق بیمهٔ وام تعمیر مسکن", 11_972, OilSideDebit, OilBandOther},
			{"حق بیمهٔ وام تبدیل به احسن", 1_276_407, OilSideDebit, OilBandOther},
			{"حق بیمهٔ متمم وام تبدیل به احسن", 606_663, OilSideDebit, OilBandOther},
			{"عضو قرض‌الحسنه بسیج", 40_000_000, OilSideDebit, OilBandOther},
			{"اقساط علی‌الحساب حقوق", 433_784_583, OilSideDebit, OilBandOther},
			{"اقساط وام قرض‌الحسنه", 29_750_000, OilSideDebit, OilBandOther},
			{"اقساط متمم وام تبدیل به احسن", 12_133_255, OilSideDebit, OilBandOther},
			{"قسط وام اضطراری بانک مهر", 71_430_000, OilSideDebit, OilBandOther},
			{"اقساط وام پس‌انداز", 94_106_690, OilSideDebit, OilBandOther},
			{"اقساط وام تعمیر مسکن", 239_431, OilSideDebit, OilBandOther},
			{"اقساط وام تبدیل به احسن", 25_528_136, OilSideDebit, OilBandOther},
			{"۱۹٪ سهم کارفرما سنوات ایثارگر", 4_705, OilSideDebit, OilBandOther},
			{"بازنشستگی ف تخصصی شغل کارفرما", 156_987_894, OilSideDebit, OilBandOther},
		},
	}
	params := DefaultOilPayParams()
	// مصوبهٔ همین فیش: کسور ۷٪ روی مبنای مشمول ۱٬۰۵۳٬۶۸۵٬۳۱۴ و مالیات از سطرهای مصوبه؛
	// پله‌های عمومی مالیات صفر می‌شوند تا مالیات دوباره حساب نشود.
	params.FundEmployeePercent = 7
	params.FundBase = 1_053_685_314
	params.VeteranIncentiveStep = 24_764
	params.TaxBrackets = nil

	slip := ComputeOilPayslip(in, params)

	if slip.Gross != 3_057_307_055 {
		t.Errorf("جمع بستانکار بند ۱ باید ۳٬۰۵۷٬۳۰۷٬۰۵۵ باشد: %d", slip.Gross)
	}
	want := []struct {
		band   int
		credit int64
		debit  int64
	}{
		{OilBandLegal, 3_057_307_055, 521_740_824},
		{OilBandCeiling, 0, 1_226_566_231},
		{OilBandOther, 419_423_503, 960_256_333},
	}
	if len(slip.Bands) != 3 {
		t.Fatalf("فیش باید سه بند داشته باشد: %d", len(slip.Bands))
	}
	for i, w := range want {
		band := slip.Bands[i]
		if band.Credit != w.credit || band.Debit != w.debit {
			t.Errorf("بند %d: بستانکار/بدهکار %d/%d است، فیش %d/%d می‌دهد",
				w.band, band.Credit, band.Debit, w.credit, w.debit)
		}
	}
	if slip.Bands[0].Net != 2_535_566_231 {
		t.Errorf("خالص بند ۱ باید ۲٬۵۳۵٬۵۶۶٬۲۳۱ باشد: %d", slip.Bands[0].Net)
	}
	if slip.Bands[2].Net != -540_832_830 {
		t.Errorf("خالص بند ۳ باید −۵۴۰٬۸۳۲٬۸۳۰ باشد: %d", slip.Bands[2].Net)
	}
	if slip.Net != 768_167_170 {
		t.Errorf("خالص نهایی باید ۷۶۸٬۱۶۷٬۱۷۰ باشد: %d", slip.Net)
	}
}

// TestOilPayslipWithoutExtrasKeepsNet فیش بدون سطر دستی همان خالص پیشین را می‌دهد
// (سه‌بندی نباید رفتار قبلی را عوض کند).
func TestOilPayslipWithoutExtrasKeepsNet(t *testing.T) {
	slip := ComputeOilPayslip(OilPayInput{BaseSalary: 120_000_000}, DefaultOilPayParams())
	if slip.Net != slip.Gross-slip.Deductions {
		t.Errorf("خالص باید همان ناخالص منهای کسور باشد: %d در برابر %d",
			slip.Net, slip.Gross-slip.Deductions)
	}
	if len(slip.Bands) != 3 || slip.Bands[1].Credit != 0 || slip.Bands[1].Debit != 0 {
		t.Errorf("بندهای ۲ و ۳ باید خالی باشند: %+v", slip.Bands)
	}
}

// TestOilFixedAllowancesAreOptional اقلام ثابت با مبلغ پیش‌فرض صفر سطر نمی‌سازند.
func TestOilFixedAllowancesAreOptional(t *testing.T) {
	slip := ComputeOilPayslip(OilPayInput{BaseSalary: 100_000_000}, DefaultOilPayParams())
	for _, line := range slip.Lines {
		switch line.Key {
		case OilAllowanceFamily, OilAllowanceHousing, OilAllowanceChildren, OilAllowanceExtra:
			t.Fatalf("قلم ثابت بدون مبلغ نباید سطر بسازد: %+v", line)
		}
	}
}
