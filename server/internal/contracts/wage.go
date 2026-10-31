package contracts

import "hrms/internal/rules"

// WageStructure اجزای ثابت مزد و فوق‌العاده‌های اقلیمی که در «حکم کارگزینی» و «قرارداد
// کار» ثبت می‌شوند.
//
// چرا در قرارداد مشترک و یک‌بار: همان فیلدها هم در حکم نوشته می‌شوند و هم در قرارداد،
// و فیش حقوق و اسناد چاپی از هر دو می‌خوانند؛ اگر جداگانه تعریف شوند، دیر یا زود واگرا
// می‌شوند. اینجا فقط «داده» است — قاعدهٔ ساختن مزد مبنا در بستهٔ `rules` است و رابط و
// سرور عدد را از همان‌جا می‌گیرند.
//
// قاعدهٔ پرکردن: هر فیلد صفر یعنی «ثبت نشده» و مقدار پیش‌فرض سازمانی (پارامتر شرکت) به
// کار می‌رود؛ پس حکم تازه لازم نیست همهٔ ضرایب را تکرار کند و کارمند می‌تواند ضریب
// دقیق‌تر خودش را داشته باشد.
type WageStructure struct {
	// ClimateRegion کلید منطقهٔ اقلیمی محل خدمت (فهرست مناطق در `rules.ClimateRegionsTable`).
	// انتخاب منطقه، ضرایب پیشنهادی را می‌آورد و بقیهٔ فیلدهای اقلیمی آن را دقیق‌تر می‌کنند.
	ClimateRegion string `json:"climateRegion,omitempty"`

	// مبالغ ریالی ثابت سند کارگزینی (ماهانه).
	RankWage           int64 `json:"rankWage,omitempty"`           // مزد رتبه
	SeniorityWage      int64 `json:"seniorityWage,omitempty"`      // مزد سنوات
	PostAllowance      int64 `json:"postAllowance,omitempty"`      // فوق‌العادهٔ پست
	SacrificeAllowance int64 `json:"sacrificeAllowance,omitempty"` // فوق‌العادهٔ ایثارگری

	// ClimateAllowance فوق‌العادهٔ اقلیمی مصوب: مبلغ ثابت ماهانهٔ اقلیم محل خدمت که
	// مستقل از ضرایب پرداخت می‌شود و **جزو مزد مبنا** است (اقلام درصدی اقلیمی جزو مزد
	// مبنا نیستند و روی آن سوار می‌شوند).
	ClimateAllowance int64 `json:"climateAllowance,omitempty"`
	// SalaryRepair مبلغ ترمیم حقوق مصوب (ریال در ماه)؛ ثابت و جزو مزد مبنا.
	SalaryRepair int64 `json:"salaryRepair,omitempty"`

	// ضرایب اقلیمی همین کارمند (۰٫۲۵ = ۲۵٪ مزد مبنا). صفر یعنی از پارامتر شرکت خوانده شود.
	BadClimatePercent     float64 `json:"badClimatePercent,omitempty"`     // بدی آب و هوا
	DeprivationPercent    float64 `json:"deprivationPercent,omitempty"`    // محرومیت از تسهیلات زندگی
	RegionHardshipPercent float64 `json:"regionHardshipPercent,omitempty"` // صعوبت شرایط منطقه
	ExtraTimePercent      float64 `json:"extraTimePercent,omitempty"`      // جبران صرف وقت اضافی
}

// HasClimatePercent آیا سند کارگزینی ضریب اقلیمی خود کارمند را دارد؟ (تقدم بر پارامتر شرکت)
func (w WageStructure) HasClimatePercent() bool {
	return w.BadClimatePercent != 0 || w.DeprivationPercent != 0 ||
		w.RegionHardshipPercent != 0 || w.ExtraTimePercent != 0
}

// HasWageComponents آیا سند کارگزینی جزء ثابت مزد (فراتر از پایه) دارد؟
func (w WageStructure) HasWageComponents() bool {
	return w.RankWage != 0 || w.SeniorityWage != 0 || w.PostAllowance != 0 ||
		w.SacrificeAllowance != 0 || w.ClimateAllowance != 0
}

// Components اجزای مزد این سند کارگزینی را می‌سازد.
//
// مزد شغل از پایهٔ حقوق سند می‌آید، اجزای ثبت‌شدهٔ سند (رتبه، سنوات، پست، ایثارگری،
// ترمیم، اقلیمی مصوب) جای خودشان می‌نشینند، و ثابت‌های شغلی درصدی از پیش‌فرض سازمانی
// (`companyPercent` با کلیدهای پارامتر حقوق) گرفته می‌شوند. محاسبه با همان ترتیب
// دو مرحله‌ای `rules` انجام می‌گیرد تا حکم، قرارداد، فیش و سند چاپی یک عدد ببینند.
func (w WageStructure) Components(baseSalary int64, companyPercent map[string]float64) rules.WageComponents {
	in := rules.WageComponentInput{
		Job:                     float64(baseSalary),
		Rank:                    float64(w.RankWage),
		Seniority:               float64(w.SeniorityWage),
		PostAllowance:           float64(w.PostAllowance),
		Sacrifice:               float64(w.SacrificeAllowance),
		SalaryRepair:            float64(w.SalaryRepair),
		Climate:                 float64(w.ClimateAllowance),
		SpecialPercent:          companyPercent["allowance.special"],
		WorkshopPercent:         companyPercent["allowance.workshop"],
		AttractionPublicPercent: companyPercent["allowance.attractionPublic"],
		PersonalBasePercent:     companyPercent["allowance.personalBase"],
		JobSpecialtyPercent:     companyPercent["allowance.jobSpecialty"],
	}
	return in.Build()
}

// ApplyOverride اجزای داده‌شده (مقدار غیرصفر) را روی اجزای سند می‌نشاند.
//
// چرا این‌طور: در «اصلاح حکم» و «تمدید قرارداد» کاربر معمولاً فقط یکی دو جزء را عوض
// می‌کند؛ مقدار صفر یعنی «همان قبلی بماند» تا اصلاح یک جزء، بقیهٔ فوق‌العاده‌ها را پاک نکند.
func (w *WageStructure) ApplyOverride(in WageStructure) {
	if in.ClimateRegion != "" {
		w.ClimateRegion = in.ClimateRegion
	}
	if in.RankWage != 0 {
		w.RankWage = in.RankWage
	}
	if in.SeniorityWage != 0 {
		w.SeniorityWage = in.SeniorityWage
	}
	if in.PostAllowance != 0 {
		w.PostAllowance = in.PostAllowance
	}
	if in.SacrificeAllowance != 0 {
		w.SacrificeAllowance = in.SacrificeAllowance
	}
	if in.ClimateAllowance != 0 {
		w.ClimateAllowance = in.ClimateAllowance
	}
	if in.SalaryRepair != 0 {
		w.SalaryRepair = in.SalaryRepair
	}
	if in.BadClimatePercent != 0 {
		w.BadClimatePercent = in.BadClimatePercent
	}
	if in.DeprivationPercent != 0 {
		w.DeprivationPercent = in.DeprivationPercent
	}
	if in.RegionHardshipPercent != 0 {
		w.RegionHardshipPercent = in.RegionHardshipPercent
	}
	if in.ExtraTimePercent != 0 {
		w.ExtraTimePercent = in.ExtraTimePercent
	}
}

// MazdBase مزد مبنا را از اجزای این سند می‌سازد (ریال، گرد شده).
func (w WageStructure) MazdBase(baseSalary int64, companyPercent map[string]float64) int64 {
	return rules.RoundHalfUp(w.Components(baseSalary, companyPercent).Total())
}

// WithWageView مزد مبنا و عنوان منطقهٔ اقلیمی را روی حکم می‌گذارد.
//
// چرا در قرارداد مشترک: هم فهرست احکام و هم جزئیات و هم پاسخ صدور از همین یک تابع رد
// می‌شوند؛ پس کاربر در هر مسیری همان مزد مبنایی را می‌بیند که فیش حساب می‌کند.
func (d SalaryDecree) WithWageView(companyPercent map[string]float64) SalaryDecree {
	d.MazdBase = d.WageStructure.MazdBase(d.BaseSalary, companyPercent)
	if region, ok := rules.ClimateRegionByKey(d.ClimateRegion); ok {
		d.ClimateRegionFa = region.TitleFa
	}
	return d
}

// WithWageView همان کار را برای قرارداد می‌کند (پایهٔ سند، حقوق ماهانهٔ قرارداد است).
func (c Contract) WithWageView(companyPercent map[string]float64) Contract {
	c.MazdBase = c.WageStructure.MazdBase(c.MonthlySalary, companyPercent)
	if region, ok := rules.ClimateRegionByKey(c.ClimateRegion); ok {
		c.ClimateRegionFa = region.TitleFa
	}
	return c
}

// Climate ضرایب اقلیمی این کارمند را می‌سازد. ترتیب تقدم روشن است: ضریب ثبت‌شده در سند
// کارگزینی → ضریب پارامتر شرکت → ضریب پیشنهادی منطقهٔ اقلیمی انتخاب‌شده. ضریب صفر
// «ثبت نشده» معنا می‌دهد، نه «صفر درصد»، پس هیچ‌وقت پرداخت اقلیمی را بی‌دلیل حذف نمی‌کند.
func (w WageStructure) Climate(companyPercent map[string]float64) rules.ClimateAllowances {
	region := rules.ClimateAllowancesFor(w.ClimateRegion)
	pick := func(own float64, key string, fallback float64) float64 {
		if own != 0 {
			return own
		}
		if v, ok := companyPercent[key]; ok && v != 0 {
			return v
		}
		return fallback
	}
	return rules.ClimateAllowances{
		BadClimate:     pick(w.BadClimatePercent, "allowance.badClimate", region.BadClimate),
		Deprivation:    pick(w.DeprivationPercent, "allowance.deprivation", region.Deprivation),
		RegionHardship: pick(w.RegionHardshipPercent, "allowance.regionHardship", region.RegionHardship),
		ExtraTime:      pick(w.ExtraTimePercent, "allowance.extraTime", region.ExtraTime),
		RegionFa:       region.RegionFa,
	}
}
