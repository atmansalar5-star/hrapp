// اقلام و ساختار تازهٔ فیش نظام پرداخت صندوق بازنشستگی نفت.
//
// چرا فایل جدا: این بخش‌ها از راستی‌آزمایی فیش رسمی سازمان با فیش سامانه بیرون آمدند
// (`docs/oil-payslip-sample-check-fa.md`): اقلام ثابتی که در فیش رسمی هستند و در آیین‌نامهٔ
// درصدی نیامده‌اند، سطر پایهٔ تشویقی ایثارگر، و ساختار سه‌بندی فیش رسمی با دو ستون
// بستانکار/بدهکار. قاعده‌ها همان‌جای همیشگی می‌مانند؛ اینجا فقط اعلان داده می‌شود.
package rules

// OilFixedAllowances اقلام «مبلغ ثابت ماهانه» فیش رسمی.
//
// چرا جدا از فهرست درصدی و با مبلغ پیش‌فرض صفر: این اقلام در فیش رسمی سازمان سطر دارند ولی
// مبلغشان مصوبه/پروندهٔ همان کارمند است، نه درصدی از یک مبنا. تا وقتی شرکت مبلغ مصوب را در
// پارامتر `oilpay.allowance.<کلید>` (یا در همان محاسبه از فرم) نگذارد، سطر خالی هم ساخته
// نمی‌شود؛ پس افزودن این اقلام رفتار قبلی را عوض نمی‌کند.
func OilFixedAllowances() []OilAllowance {
	return []OilAllowance{
		{Key: OilAllowanceFamily, TitleFa: "کمک هزینهٔ عایله‌مندی", GroupFa: OilGroupClimate,
			Fixed:  true,
			NoteFa: "مبلغ ثابت ماهانه — پارامتر oilpay.allowance.family (ریال)."},
		{Key: OilAllowanceHousing, TitleFa: "کمک هزینهٔ مسکن", GroupFa: OilGroupClimate,
			Fixed:  true,
			NoteFa: "مبلغ ثابت ماهانه — پارامتر oilpay.allowance.housing (ریال)."},
		{Key: OilAllowanceChildren, TitleFa: "حق اولاد", GroupFa: OilGroupClimate,
			Fixed:  true,
			NoteFa: "جمع حق اولاد فرزندان مشمول — پارامتر oilpay.allowance.children (ریال)."},
		{Key: OilAllowanceExtra, TitleFa: "فوقالعادهٔ خاص", GroupFa: OilGroupClimate,
			Fixed:  true,
			NoteFa: "مبلغ ثابت ماهانه بر پایهٔ مصوبهٔ سازمان — پارامتر oilpay.allowance.extra (ریال)."},
	}
}

// OilAllowanceCatalog فهرست کامل اقلام فیش: اقلام پارامتری به‌همراه اقلام ثابت، بدون تکرار.
//
// چرا یک تابع و نه دو فهرست جدا: هم موتور محاسبه و هم لایهٔ پارامترها باید یک فهرست
// ببینند؛ اگر هر کدام فهرست خودش را بسازد، قلم ثابت دو بار در فیش می‌آید یا مبلغش از
// پارامتر خوانده نمی‌شود.
func OilAllowanceCatalog(allowances []OilAllowance) []OilAllowance {
	out := make([]OilAllowance, 0, len(allowances)+4)
	out = append(out, allowances...)
	have := map[string]bool{}
	for _, a := range out {
		have[a.Key] = true
	}
	for _, extra := range OilFixedAllowances() {
		if have[extra.Key] {
			continue
		}
		out = append(out, extra)
	}
	return out
}

// OilExtraLine یک سطر دستی فیش: چیزی که قاعدهٔ عمومی ندارد ولی در فیش رسمی می‌آید
// (کسورات تطبیق جهت رعایت سقف، وام، هزینهٔ درمان، پس‌انداز، علی‌الحساب، برگشت‌ها و ...).
//
// چرا سطر دستی و نه قاعدهٔ ساختگی: این اقلام مصوبه/قرارداد همان دوره‌اند و فرمول عمومی
// ندارند؛ سامانه آن‌ها را به‌عنوان داده می‌گیرد، در بند درست فیش می‌نشاند و در جمع‌ها
// لحاظ می‌کند. هیچ عددی از این‌ها در رابط ساخته نمی‌شود.
type OilExtraLine struct {
	TitleFa string `json:"titleFa"`
	Amount  int64  `json:"amount"`
	// Side «credit» یعنی بستانکار (پرداخت به کارمند) و «debit» یعنی بدهکار (کسور).
	Side string `json:"side"`
	// Band بند فیش: ۱ = حقوق و فوق‌العاده‌ها و کسورات قانونی، ۲ = کسورات تطبیق جهت
	// رعایت سقف، ۳ = سایر پرداخت‌ها و کسورات مؤثر در خالص. صفر همان بند ۳ است.
	Band int `json:"band,omitempty"`
}

// کلیدهای بندهای فیش رسمی.
const (
	OilBandLegal    = 1
	OilBandCeiling  = 2
	OilBandOther    = 3
	OilSideCredit   = "credit"
	OilSideDebit    = "debit"
	OilBandLegalFa  = "حقوق و فوق‌العاده‌ها و کسورات قانونی ماه جاری"
	OilBandCeilFa   = "کسورات تطبیق حقوق جهت رعایت سقف"
	OilBandOtherFa  = "سایر پرداخت‌ها و کسورات مؤثر در خالص پرداختی"
	oilBandOtherKey = "other"
)

// OilPayslipBand یک بند فیش با دو ستون بستانکار و بدهکار (ساختار فیش رسمی سازمان).
type OilPayslipBand struct {
	Key     string       `json:"key"`
	TitleFa string       `json:"titleFa"`
	Lines   []OilPayLine `json:"lines"`
	// Credit جمع بستانکار (پرداخت) و Debit جمع بدهکار (کسور) این بند.
	Credit int64 `json:"credit"`
	Debit  int64 `json:"debit"`
	// Net خالص بند = بستانکار − بدهکار.
	Net int64 `json:"net"`
}

// oilBandOf بند یک سطر دستی؛ مقدار ناشناخته به بند «سایر» می‌رود.
func oilBandOf(line OilExtraLine) int {
	switch line.Band {
	case OilBandLegal:
		return OilBandLegal
	case OilBandCeiling:
		return OilBandCeiling
	}
	return OilBandOther
}

// oilIsDebit آیا سطر دستی بدهکار است (پیش‌فرض: بستانکار).
func oilIsDebit(line OilExtraLine) bool {
	return line.Side == OilSideDebit
}

// oilExtraPayLine سطر دستی را به سطر فیش تبدیل می‌کند.
// علامت مبلغ: کسور در سطرهای فیش با مبلغ مثبت و گروه «کسور» می‌آیند تا با بقیهٔ سطرها
// یکدست باشند؛ جمع بند است که بستانکار و بدهکار را جدا می‌کند.
func oilExtraPayLine(line OilExtraLine) OilPayLine {
	group := OilGroupClimate
	if oilIsDebit(line) {
		group = OilGroupDeduct
	}
	return OilPayLine{
		Key: "extra", TitleFa: line.TitleFa, GroupFa: group, Amount: line.Amount,
		DetailFa: "سطر دستی همین فیش (مصوبه/قرارداد همان دوره)",
	}
}

// کلیدهای سطرهای «اطلاعاتی» فیش: مبنای فوق‌العاده‌ها و مبنای جذب جمع نیستند و در جمع
// بندها شمرده نمی‌شوند (فقط برای خوانایی نمایش داده می‌شوند).
const (
	oilInfoAllowanceBaseKey  = "allowanceBase"
	oilInfoAttractionBaseKey = "attractionBase"
)

// oilVeteranIncentiveDetailFa شرح سطر پایهٔ تشویقی ایثارگر.
func oilVeteranIncentiveDetailFa(steps int) string {
	return "پایهٔ تشویقی " + IntFa(steps) + " × مبلغ هر پایه (پارامتر oilpay.veteran.incentiveStep)"
}

// oilFixedDetailFa شرح سطر اقلام ثابت.
const oilFixedDetailFa = "مبلغ ثابت مصوب — پیش از اجرا قابل تغییر است"

// OilVeteranIncentiveTitleFa عنوان سطر افزایش سنواتی تشویقی ایثارگر.
func OilVeteranIncentiveTitleFa() string {
	return "افزایش سنواتی تشویقی ایثارگر"
}

// oilBuildBands سه بند فیش رسمی را از سطرهای محاسبه‌شده و سطرهای دستی می‌سازد.
//
// سطرهای «اطلاعاتی» (مبنای فوق‌العاده‌ها، مبنای جذب) نمایش داده می‌شوند ولی در جمع بند
// شمرده نمی‌شوند؛ وگرنه همان مبلغ دو بار در بستانکار می‌نشست.
func oilBuildBands(lines []OilPayLine, extras []OilExtraLine) []OilPayslipBand {
	bands := []OilPayslipBand{
		{Key: "legal", TitleFa: OilBandLegalFa, Lines: []OilPayLine{}},
		{Key: "ceiling", TitleFa: OilBandCeilFa, Lines: []OilPayLine{}},
		{Key: oilBandOtherKey, TitleFa: OilBandOtherFa, Lines: []OilPayLine{}},
	}
	for _, line := range lines {
		band := &bands[0]
		band.Lines = append(band.Lines, line)
		if line.Key == oilInfoAllowanceBaseKey || line.Key == oilInfoAttractionBaseKey {
			continue
		}
		if line.GroupFa == OilGroupDeduct {
			band.Debit += line.Amount
		} else {
			band.Credit += line.Amount
		}
	}
	for _, extra := range extras {
		idx := oilBandOf(extra) - 1
		band := &bands[idx]
		band.Lines = append(band.Lines, oilExtraPayLine(extra))
		if oilIsDebit(extra) {
			band.Debit += extra.Amount
		} else {
			band.Credit += extra.Amount
		}
	}
	for i := range bands {
		bands[i].Net = bands[i].Credit - bands[i].Debit
	}
	return bands
}
