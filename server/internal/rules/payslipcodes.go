// کدهای رسمی اقلام فیش («کد فوق العاده») — جدول مرجع سازمان.
//
// چرا در بستهٔ قواعد: روی فیش چاپی و در مکاتبه با مالی و بیمه، هر قلم با **کد** شناخته
// می‌شود، نه با عنوان فارسی. پیش از این، اقلام سامانه فقط کلید و عنوان داشتند و کد رسمی
// جایی نگه‌داری نمی‌شد؛ نتیجه این بود که فیش سامانه با فیش رسمی سازمان سطر‌به‌سطر قابل
// تطبیق نبود و هر گزارش مالی به فهرست دستی کدها وابسته می‌شد.
//
// اکنون این جدول یک منبع حقیقت است و سه جا از آن می‌خواند:
//   - موتور فیش صندوق نفت (`ComputeOilPayslip`) — کد روی هر سطر می‌نشیند،
//   - موتور حقوق و دستمزد جاری (`ComputePayroll`) — سطرهای کددار فیش عمومی,
//   - موتور ۳۴۰۰۶ کدهای خودش را دارد (۳۴۶۰۱ تا ۳۴۶۱۸) و در جدول جدا می‌ماند؛ کدهای این
//     جدول برای آن نظام معنا ندارد.
//
// «کلید داخلی» (Key) پیوند کد رسمی با قلم سامانه است. کدی که در موتورهای فعلی پیاده نشده
// باشد، کلید خالی می‌گیرد ولی در فهرست می‌ماند تا فرم/گزارش بتواند کد را نشان دهد و
// نبودش در محاسبه صریح باشد (نه پنهان).
package rules

import "strings"

// انواع قلم فیش بر پایهٔ جای آن در جمع‌ها.
const (
	// PayslipKindEarning قلم بستانکار (به کارمند پرداخت می‌شود).
	PayslipKindEarning = "earning"
	// PayslipKindDeduction قلم کسور (از پرداخت کم می‌شود).
	PayslipKindDeduction = "deduction"
	// PayslipKindLoan وام و اقساط آن؛ ماهیتاً جابه‌جایی تعهد است نه مزایا.
	PayslipKindLoan = "loan"
)

// PayslipCode یک ردیف از جدول رسمی «کد فوق العاده».
type PayslipCode struct {
	// Code کد رسمی قلم (رشته است، نه عدد: صفرهای ابتدایی و ترتیب الفبایی حفظ شود).
	Code string `json:"code"`
	// TitleFa عنوان رسمی همان‌گونه که در جدول سازمان آمده است.
	TitleFa string `json:"titleFa"`
	// Key کلید قلم در موتورهای سامانه؛ خالی یعنی در موتورهای فعلی پیاده نشده است.
	Key string `json:"key,omitempty"`
	// Kind نوع قلم: earning | deduction | loan.
	Kind string `json:"kind"`
	// NoteFa توضیح کوتاه برای فرم و گزارش.
	NoteFa string `json:"noteFa,omitempty"`
}

// payslipCodeCatalog جدول رسمی کدهای فوق العاده (۲۸ ردیف مصوب سازمان).
var payslipCodeCatalog = []PayslipCode{
	{Code: "101", TitleFa: "حقوق پایه", Key: "baseSalary", Kind: PayslipKindEarning,
		NoteFa: "پایهٔ حقوق ماهانه (A) — مبنای همهٔ اقلام درصدی."},
	{Code: "102", TitleFa: "بدی آب و هوا", Key: "badClimate", Kind: PayslipKindEarning,
		NoteFa: "فوقالعادهٔ مناطق گرمسیری؛ درصدی از پایهٔ حقوق و سازندهٔ «مبنای فوقالعاده‌ها»."},
	{Code: "104", TitleFa: "محرومیت از تسهیلات زندگی", Key: "deprivation", Kind: PayslipKindEarning,
		NoteFa: "ردیف ۴ آیین‌نامه — درصد از جدول ۱ بر پایهٔ محل سکونت."},
	{Code: "127", TitleFa: "جبران صرف وقت اضافی", Key: "timeConsumed", Kind: PayslipKindEarning,
		NoteFa: "ردیف ۶ آیین‌نامه — درصد از جدول ۲ (ستون روزکار/نوبتکار)."},
	{Code: "144", TitleFa: "درصد فوق العاده شغل", Key: "attractionSpecialty", Kind: PayslipKindEarning,
		NoteFa: "درصد فوق‌العادهٔ شغلی بر پایهٔ گروه/پایهٔ سمت جدول ۳ به‌همراه ضریب مدرک."},
	{Code: "103", TitleFa: "مبلغ فوق العاده جذب", Key: "attraction", Kind: PayslipKindEarning,
		NoteFa: "مبلغ (نه درصد) فوق‌العادهٔ جذب؛ در موتور صندوق نفت هنوز پیاده نشده و «مبنای جذب» از فوق‌العادهٔ ویژه ساخته می‌شود."},
	{Code: "122", TitleFa: "فوق العاده کارگاهی", Key: "workshop", Kind: PayslipKindEarning,
		NoteFa: "ردیف ۵ آیین‌نامه — شرایط محیط کار."},
	{Code: "151", TitleFa: "مسکن", Key: "housing", Kind: PayslipKindEarning,
		NoteFa: "کمک هزینهٔ مسکن؛ مبلغ ثابت مصوب (پارامتر oilpay.allowance.housing)."},
	{Code: "181", TitleFa: "حق اولاد", Key: "children", Kind: PayslipKindEarning,
		NoteFa: "جمع حق اولاد فرزندان مشمول (پارامتر oilpay.allowance.children)."},
	{Code: "159", TitleFa: "عائله مندی", Key: "family", Kind: PayslipKindEarning,
		NoteFa: "کمک هزینهٔ عائله‌مندی؛ مبلغ ثابت مصوب (پارامتر oilpay.allowance.family)."},
	{Code: "353", TitleFa: "فوق العاده ماموریت", Key: "mission", Kind: PayslipKindEarning,
		NoteFa: "فوقالعادهٔ مأموریت؛ محاسبه‌اش در قواعد مأموریت است."},
	{Code: "558", TitleFa: "هزینه نگهداشت منازل سازمانی", Key: "orgHousing", Kind: PayslipKindEarning,
		NoteFa: "هزینهٔ نگهداشت منازل سازمانی که به کارمند پرداخت می‌شود."},
	{Code: "394", TitleFa: "کمک هزینه مهد کودک", Key: "kindergarten", Kind: PayslipKindEarning,
		NoteFa: "کمک هزینهٔ مهد کودک؛ مصوبهٔ سالانهٔ سازمان."},
	{Code: "361", TitleFa: "حق التدریس ساعتی", Key: "hourlyTeaching", Kind: PayslipKindEarning,
		NoteFa: "حق‌التدریس بر پایهٔ ساعت آموزش (ماژول آموزش)."},
	{Code: "637", TitleFa: "صندوق تامین آتیه کارمندان", Key: "futureFund", Kind: PayslipKindDeduction,
		NoteFa: "سهم کارمند در صندوق تأمین آتیه."},
	{Code: "559", TitleFa: "هزینه عوارض برق منازل سازمانی", Key: "orgHousingPower", Kind: PayslipKindDeduction,
		NoteFa: "عوارض برق منازل سازمانی؛ از پرداخت کارمند کسر می‌شود."},
	{Code: "213", TitleFa: "فوق العاده ماموریت موقت داخل کشور", Key: "missionDomestic", Kind: PayslipKindEarning,
		NoteFa: "مأموریت موقت داخلی؛ جدا از فوقالعادهٔ مأموریت (کد ۳۵۳)."},
	{Code: "364", TitleFa: "پاداش نوروزی /عیدی", Key: "nowruzBonus", Kind: PayslipKindEarning,
		NoteFa: "عیدی و پاداش نوروزی؛ مصوبهٔ سالانه."},
	{Code: "335", TitleFa: "پاداش افزایش تولید پتروشیمی", Key: "productionBonus", Kind: PayslipKindEarning,
		NoteFa: "پاداش افزایش تولید (مصوبهٔ صنعت پتروشیمی)."},
	{Code: "500", TitleFa: "وام اضطراری", Key: "loanEmergency", Kind: PayslipKindLoan,
		NoteFa: "وام اضطراری؛ پرداختش در فیش و بازپرداختش با کد ۵۴۰۰."},
	{Code: "502", TitleFa: "وام پس انداز", Key: "loanSavings", Kind: PayslipKindLoan,
		NoteFa: "وام از محل پس‌انداز؛ بازپرداختش با کد ۵۴۰۲."},
	{Code: "519", TitleFa: "وام مددکاری", Key: "loanSocialWork", Kind: PayslipKindLoan,
		NoteFa: "وام مددکاری اجتماعی."},
	{Code: "503", TitleFa: "وام شرکتی (دوبرابر حقوق)", Key: "loanCompany", Kind: PayslipKindLoan,
		NoteFa: "وام شرکت تا دو برابر حقوق."},
	{Code: "551", TitleFa: "بازنشستگی سهم کارمند", Key: "fundEmployee", Kind: PayslipKindDeduction,
		NoteFa: "کسور صندوق بازنشستگی سهم کارمند (درصد پارامتر oilpay.fund.employeePercent)."},
	{Code: "552", TitleFa: "پس انداز سهم کارمند", Key: "savingsEmployee", Kind: PayslipKindDeduction,
		NoteFa: "پس‌انداز سهم کارمند."},
	{Code: "557", TitleFa: "مالیات بر درآمد", Key: "tax", Kind: PayslipKindDeduction,
		NoteFa: "مالیات بر درآمد ماه؛ از جدول پله‌های مالیات پارامتر شرکت."},
	{Code: "5400", TitleFa: "کسر اقساط وام اضطراری", Key: "loanInstallmentEmergency", Kind: PayslipKindDeduction,
		NoteFa: "قسط ماهانهٔ وام اضطراری (کد ۵۰۰)."},
	{Code: "5402", TitleFa: "کسر اقساط وام پس انداز", Key: "loanInstallmentSavings", Kind: PayslipKindDeduction,
		NoteFa: "قسط ماهانهٔ وام پس‌انداز (کد ۵۰۲)."},
}

// PayslipCodes فهرست رسمی کدهای فوق العاده (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func PayslipCodes() []PayslipCode {
	out := make([]PayslipCode, len(payslipCodeCatalog))
	copy(out, payslipCodeCatalog)
	return out
}

// PayslipCodeByNumber یافتن کد رسمی با شمارهٔ آن (با پذیرش فاصله و صفر ابتدایی).
func PayslipCodeByNumber(code string) (PayslipCode, bool) {
	want := normalizePayslipCode(code)
	for _, c := range payslipCodeCatalog {
		if normalizePayslipCode(c.Code) == want {
			return c, true
		}
	}
	return PayslipCode{}, false
}

// PayslipCodeOf کلید قلم سامانه را به کد رسمی نگاشت می‌کند.
func PayslipCodeOf(key string) (PayslipCode, bool) {
	want := strings.TrimSpace(key)
	if want == "" {
		return PayslipCode{}, false
	}
	for _, c := range payslipCodeCatalog {
		if c.Key != "" && c.Key == want {
			return c, true
		}
	}
	return PayslipCode{}, false
}

// PayslipCodeNumberOf شمارهٔ رسمی یک کلید قلم (خالی یعنی کد رسمی ندارد).
func PayslipCodeNumberOf(key string) string {
	if c, ok := PayslipCodeOf(key); ok {
		return c.Code
	}
	return ""
}

// PayslipCodeTitleFa عنوان رسمی یک کد؛ کد ناشناخته رشتهٔ خالی برمی‌گرداند.
func PayslipCodeTitleFa(code string) string {
	if c, ok := PayslipCodeByNumber(code); ok {
		return c.TitleFa
	}
	return ""
}

// PayslipCodeKindsFa عنوان فارسی نوع قلم (برای چیپ و گزارش).
func PayslipCodeKindsFa(kind string) string {
	switch kind {
	case PayslipKindEarning:
		return "بستانکار"
	case PayslipKindDeduction:
		return "کسور"
	case PayslipKindLoan:
		return "وام و اقساط"
	default:
		return kind
	}
}

// PayslipCodesByKind فهرست کدها به تفکیک نوع قلم.
func PayslipCodesByKind(kind string) []PayslipCode {
	out := []PayslipCode{}
	for _, c := range payslipCodeCatalog {
		if c.Kind == kind {
			out = append(out, c)
		}
	}
	return out
}

// PayslipCodeGaps کلیدهای قلمی که در موتورهای سامانه وجود دارند ولی در جدول رسمی کد
// ندارند؛ گزارش‌شان برای شفافیت است تا «قلم بی‌کد» پنهان نماند.
func PayslipCodeGaps() []string {
	keys := []string{
		// اقلام موتور صندوق نفت که در جدول رسمی سازمان کدی ندارند.
		"special", "production", "shiftWorker", "climate", "allowanceBase", "attractionBase",
		"overtime", "veteranIncentive", "rankWage", "seniorityWage", "postAllowance",
		"climateAllowance", "sacrificeAllowance", "salaryRepair", "payGap", "extra",
		// اقلام موتور جاری که در جدول نیامده‌اند (بیمهٔ سهم کارمند در جدول رسمی نیست).
		"insurance", "insuranceEmployee", "insuranceEmployer", "gross", "net",
	}
	out := []string{}
	for _, key := range keys {
		if _, ok := PayslipCodeOf(key); !ok {
			out = append(out, key)
		}
	}
	return out
}

// normalizePayslipCode کد را برای مقایسه یکدست می‌کند (ارقام فارسی، فاصله، صفر ابتدایی).
func normalizePayslipCode(code string) string {
	value := strings.TrimSpace(code)
	// ارقام فارسی/عربی به لاتین: کد ممکن است از فرم یا پارامتر شرکت فارسی آمده باشد.
	value = strings.Map(func(r rune) rune {
		if r >= '۰' && r <= '۹' { // ۰..۹
			return '0' + (r - '۰')
		}
		if r >= '٠' && r <= '٩' { // ٠..٩ عربی
			return '0' + (r - '٠')
		}
		return r
	}, value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.TrimLeft(value, "0")
	if value == "" {
		return "0"
	}
	return value
}
