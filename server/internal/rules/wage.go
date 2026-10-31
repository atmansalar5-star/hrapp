package rules

// اجزای مزد و «مزد مبنا» — یک تعریف، همهٔ مشتقات.
//
// مزد مبنا مبنای محاسبهٔ افزودنی‌های دوره است (اضافه‌کاری، نوبت‌کاری، شب‌کاری،
// فوق‌العاده‌های اقلیمی و سختی کار)، پس اگر ناقص حساب شود همهٔ آن اقلام کم برآورد
// می‌شوند. اینجا مزد مبنا از **همهٔ اجزای ثابت و مستمر** ساخته می‌شود، نه فقط از
// مزد شغل و رتبه:
//
//	مزد شغل و ملحقات = مزد شغل + مزد رتبه + مزد سنوات
//	ثابت‌های شغلی   = ویژه + کارگاهی + جذب عمومی + جذب (پایهٔ شخصی) + تخصصی شغل
//	مبالغ ثابت      = فوق‌العادهٔ پست + ایثارگری + ترمیم حقوق + فوق‌العادهٔ اقلیمی مصوب
//	مزد مبنا        = مزد شغل و ملحقات + ثابت‌های شغلی + مبالغ ثابت
//
// **ترتیب محاسبه عمداً دو مرحله‌ای است و دوری نیست:** ثابت‌های شغلی درصدی از «مزد شغل
// و ملحقات» گرفته می‌شوند (نه از خود مزد مبنا)، و تازه جمع آن‌ها مزد مبنا را می‌سازد.
// بعد از آن، فوق‌العاده‌های درصدی اقلیمی روی همین مزد مبنا سوار می‌شوند. اگر این
// ترتیب شکسته شود (درصد از مزد مبنا، آن هم داخل خود مزد مبنا) معادله دوری می‌شود و
// عدد نهایی به ترتیب محاسبه وابسته می‌ماند.
//
// همهٔ اجزا دادهٔ پارامتریک‌اند: از رکورد «پارامتر حقوق»، از حکم کارگزینی یا از قرارداد
// کارمند می‌آیند؛ هیچ عددی در این فایل سخت نشده است.

// WageComponents اجزای مزد یک کارمند در یک دوره.
//
// فیلدها به ترتیب شاکلهٔ استاندارد فیش حقوق و حکم کارگزینی چیده شده‌اند تا نمایش چاپی
// و محاسبه یک ترتیب داشته باشند.
type WageComponents struct {
	Job              float64 `json:"job"`              // مزد شغل (پایهٔ حقوق)
	Rank             float64 `json:"rank"`             // مزد رتبه
	Seniority        float64 `json:"seniority"`        // مزد سنوات
	Special          float64 `json:"special"`          // فوق‌العادهٔ ویژه
	Workshop         float64 `json:"workshop"`         // فوق‌العادهٔ کارگاهی
	AttractionPublic float64 `json:"attractionPublic"` // جذب عمومی
	PersonalBase     float64 `json:"personalBase"`     // جذب (پایهٔ شخصی)
	JobSpecialty     float64 `json:"jobSpecialty"`     // تخصصی شغل (مدرک)
	PostAllowance    float64 `json:"postAllowance"`    // فوق‌العادهٔ پست
	Sacrifice        float64 `json:"sacrifice"`        // فوق‌العادهٔ ایثارگری
	SalaryRepair     float64 `json:"salaryRepair"`     // ترمیم حقوق
	Climate          float64 `json:"climate"`          // فوق‌العادهٔ اقلیمی مصوب (اقلیمِ محل خدمت)
}

// JobAndAttachments «مزد شغل و ملحقات»: مبنای ثابت‌های شغلی درصدی.
func (w WageComponents) JobAndAttachments() float64 {
	return w.Job + w.Rank + w.Seniority
}

// JobConstants ثابت‌های شغلی درصدی (ویژه، کارگاهی، جذب عمومی، جذب پایه، تخصصی شغل).
func (w WageComponents) JobConstants() float64 {
	return w.Special + w.Workshop + w.AttractionPublic + w.PersonalBase + w.JobSpecialty
}

// Total مزد مبنا: جمع همهٔ اجزای ثابت (مبنای اقلام درصدی دوره).
func (w WageComponents) Total() float64 {
	return w.JobAndAttachments() + w.JobConstants() + w.PostAllowance + w.Sacrifice + w.SalaryRepair + w.Climate
}

// WagePart یک سطر از شاکلهٔ مزد مبنا برای نمایش در فیش و حکم.
type WagePart struct {
	Code    string  `json:"code"`
	TitleFa string  `json:"titleFa"`
	KindFa  string  `json:"kindFa"`
	Amount  float64 `json:"amount"`
}

// Breakdown شاکلهٔ مزد مبنا را خط‌به‌خط می‌دهد: هم فیش حقوق و هم حکم کارگزینی همین را
// نشان می‌دهند تا کاربر ببیند عدد مزد مبنا از کجا آمده است. جزءهای صفر حذف می‌شوند.
func (w WageComponents) Breakdown() []WagePart {
	all := []WagePart{
		{Code: "WAGE_JOB", TitleFa: "مزد شغل", KindFa: "مزد شغل و ملحقات", Amount: w.Job},
		{Code: "WAGE_RANK", TitleFa: "مزد رتبه", KindFa: "مزد شغل و ملحقات", Amount: w.Rank},
		{Code: "WAGE_SENIORITY", TitleFa: "مزد سنوات", KindFa: "مزد شغل و ملحقات", Amount: w.Seniority},
		{Code: "SPECIAL", TitleFa: "فوق‌العادهٔ ویژه", KindFa: "ثابت شغلی (درصدی از مزد شغل و ملحقات)", Amount: w.Special},
		{Code: "WORKSHOP", TitleFa: "فوق‌العادهٔ کارگاهی", KindFa: "ثابت شغلی (درصدی از مزد شغل و ملحقات)", Amount: w.Workshop},
		{Code: "ATTRACTION_PUBLIC", TitleFa: "جذب عمومی", KindFa: "ثابت شغلی (درصدی از مزد شغل و ملحقات)", Amount: w.AttractionPublic},
		{Code: "PERSONAL_BASE", TitleFa: "جذب (پایهٔ شخصی)", KindFa: "ثابت شغلی (درصدی از مزد شغل و ملحقات)", Amount: w.PersonalBase},
		{Code: "JOB_SPECIALTY", TitleFa: "تخصصی شغل (مدرک)", KindFa: "ثابت شغلی (درصدی از مزد شغل و ملحقات)", Amount: w.JobSpecialty},
		{Code: "ALLOW_POST", TitleFa: "فوق‌العادهٔ پست", KindFa: "مبلغ ثابت مصوب", Amount: w.PostAllowance},
		{Code: "ALLOW_SACRIFICE", TitleFa: "فوق‌العادهٔ ایثارگری", KindFa: "مبلغ ثابت مصوب", Amount: w.Sacrifice},
		{Code: "SALARY_REPAIR", TitleFa: "ترمیم حقوق", KindFa: "مبلغ ثابت مصوب", Amount: w.SalaryRepair},
		{Code: "CLIMATE", TitleFa: "فوق‌العادهٔ اقلیمی", KindFa: "مبلغ ثابت مصوب (اقلیم محل خدمت)", Amount: w.Climate},
	}
	out := make([]WagePart, 0, len(all))
	for _, p := range all {
		if p.Amount != 0 {
			out = append(out, p)
		}
	}
	return out
}

// Vars متغیرهای فرمول اجزای مزد را می‌سازد تا اقلام حقوق بتوانند به هر جزء، به «مزد
// شغل و ملحقات» یا به کل مزد مبنا ارجاع دهند.
func (w WageComponents) Vars() map[string]float64 {
	total := w.Total()
	return map[string]float64{
		"wage.job":               w.Job,
		"wage.rank":              w.Rank,
		"wage.seniority":         w.Seniority,
		"wage.special":           w.Special,
		"wage.workshop":          w.Workshop,
		"wage.attractionPublic":  w.AttractionPublic,
		"wage.personalBase":      w.PersonalBase,
		"wage.jobSpecialty":      w.JobSpecialty,
		"allowance.post":         w.PostAllowance,
		"allowance.sacrifice":    w.Sacrifice,
		"allowance.salaryRepair": w.SalaryRepair,
		"allowance.climate":      w.Climate,
		"wage.jobAttachments":    w.JobAndAttachments(),
		"wage.constants":         w.JobConstants(),
		"wage.total":             total,
		"mazd.base":              total,
		"mazd":                   total,
	}
}

// WageComponentInput مقادیر خام اجزای مزد پیش از محاسبهٔ دو مرحله‌ای.
//
// درصدهای شغلی نسبت‌اند (۰٫۳ = ۳۰٪) و روی «مزد شغل و ملحقات» بسته می‌شوند؛ مبالغ
// ریالی‌اند. هر کلید غایب صفر می‌ماند، پس نبود یک پارامتر محاسبه را متوقف نمی‌کند.
type WageComponentInput struct {
	Job           float64
	Rank          float64
	Seniority     float64
	PostAllowance float64
	Sacrifice     float64
	SalaryRepair  float64
	Climate       float64

	SpecialPercent          float64
	WorkshopPercent         float64
	AttractionPublicPercent float64
	PersonalBasePercent     float64
	JobSpecialtyPercent     float64
}

// Build اجزای مزد را با ترتیب دو مرحله‌ای می‌سازد: اول مزد شغل و ملحقات، بعد ثابت‌های
// درصدی از همان، و در پایان مزد مبنا. تنها نقطهٔ محاسبهٔ شاکلهٔ مزد است؛ شبیه‌ساز فیش،
// حکم کارگزینی، قرارداد و اسناد چاپی همه از همین‌جا عدد می‌گیرند تا هیچ‌جا دو قاعده نباشد.
func (in WageComponentInput) Build() WageComponents {
	attachments := in.Job + in.Rank + in.Seniority
	return WageComponents{
		Job:              in.Job,
		Rank:             in.Rank,
		Seniority:        in.Seniority,
		Special:          in.SpecialPercent * attachments,
		Workshop:         in.WorkshopPercent * attachments,
		AttractionPublic: in.AttractionPublicPercent * attachments,
		PersonalBase:     in.PersonalBasePercent * attachments,
		JobSpecialty:     in.JobSpecialtyPercent * attachments,
		PostAllowance:    in.PostAllowance,
		Sacrifice:        in.Sacrifice,
		SalaryRepair:     in.SalaryRepair,
		Climate:          in.Climate,
	}
}

// WageComponentsFromVars اجزای مزد را از یک نگاشت متغیر/پارامتر می‌خواند (کلیدهای
// نقطه‌دار پارامتر حقوق و ورودی شبیه‌ساز) و با همان ترتیب دو مرحله‌ای می‌سازد.
//
// درصدهای شغلی از کلیدهای `allowance.special`، `allowance.workshop`،
// `allowance.attractionPublic`، `allowance.personalBase` و `allowance.jobSpecialty`
// خوانده می‌شوند؛ مقدار ریالی هر ثابت اگر مستقیم داده شده باشد (`wage.special` و…)
// بر درصد اولویت دارد تا حکم/قرارداد بتواند عدد قطعی خودش را تحمیل کند.
func WageComponentsFromVars(values map[string]float64) WageComponents {
	in := WageComponentInput{
		Job:                     values["wage.job"],
		Rank:                    values["wage.rank"],
		Seniority:               values["wage.seniority"],
		PostAllowance:           values["allowance.post"],
		Sacrifice:               values["allowance.sacrifice"],
		SalaryRepair:            values["allowance.salaryRepair"],
		Climate:                 values["allowance.climate"],
		SpecialPercent:          values["allowance.special"],
		WorkshopPercent:         values["allowance.workshop"],
		AttractionPublicPercent: values["allowance.attractionPublic"],
		PersonalBasePercent:     values["allowance.personalBase"],
		JobSpecialtyPercent:     values["allowance.jobSpecialty"],
	}
	wages := in.Build()
	// مقدار ریالی صریح بر درصد اولویت دارد (حکم کارگزینی می‌تواند عدد قطعی بدهد).
	if v, ok := values["wage.special"]; ok && v != 0 {
		wages.Special = v
	}
	if v, ok := values["wage.workshop"]; ok && v != 0 {
		wages.Workshop = v
	}
	if v, ok := values["wage.attractionPublic"]; ok && v != 0 {
		wages.AttractionPublic = v
	}
	if v, ok := values["wage.personalBase"]; ok && v != 0 {
		wages.PersonalBase = v
	}
	if v, ok := values["wage.jobSpecialty"]; ok && v != 0 {
		wages.JobSpecialty = v
	}
	return wages
}

// ClimateAllowances ضرایب اقلیمی یک کارمند: همیشه از خود کارمند (حکم یا قرارداد) و در
// نبود آن از پارامتر شرکت خوانده می‌شود. این ضرایب روی «مزد مبنا» بسته می‌شوند و خودشان
// جزو مزد مبنا نیستند؛ پس معادله دوری نمی‌شود.
type ClimateAllowances struct {
	BadClimate     float64 `json:"badClimate"`     // بدی آب و هوا
	Deprivation    float64 `json:"deprivation"`    // محرومیت از تسهیلات زندگی
	RegionHardship float64 `json:"regionHardship"` // صعوبت شرایط منطقه
	ExtraTime      float64 `json:"extraTime"`      // جبران صرف وقت اضافی
	RegionFa       string  `json:"regionFa,omitempty"`
	SourceFa       string  `json:"sourceFa,omitempty"` // «حکم کارگزینی» / «قرارداد» / «پارامتر شرکت»
}

// Vars ضرایب اقلیمی را به کلیدهای پارامتریک اقلام حقوق تبدیل می‌کند؛ کلیدها همان
// پیش‌فرض شرکت‌اند، پس اولویت کارمند بدون ساختن کلید تازه اعمال می‌شود.
func (c ClimateAllowances) Vars() map[string]float64 {
	return map[string]float64{
		"allowance.badClimate":     c.BadClimate,
		"allowance.deprivation":    c.Deprivation,
		"allowance.regionHardship": c.RegionHardship,
		"allowance.extraTime":      c.ExtraTime,
	}
}

// ClimateRegion مناطق اقلیمی نمونهٔ صنعت و ضریب پیشنهادی هر منطقه.
//
// چرا فهرست در دامنه: منطقهٔ محل خدمت دادهٔ کارگزینی است و باید یکدست انتخاب شود؛
// ضریب هر منطقه پیشنهاد است و حکم/قرارداد می‌تواند آن را دقیق‌تر کند.
type ClimateRegion struct {
	Key            string  `json:"key"`
	TitleFa        string  `json:"titleFa"`
	BadClimate     float64 `json:"badClimate"`
	Deprivation    float64 `json:"deprivation"`
	RegionHardship float64 `json:"regionHardship"`
	ExtraTime      float64 `json:"extraTime"`
}

// ClimateRegionsTable مناطق اقلیمی مصوب (پیشنهاد ضرایب؛ قابل بازنویسی در حکم).
var ClimateRegionsTable = []ClimateRegion{
	{Key: "south_coast", TitleFa: "جنوب و سواحل خلیج فارس", BadClimate: 0.25, Deprivation: 0.25, RegionHardship: 0.30, ExtraTime: 0.15},
	{Key: "north", TitleFa: "شمال و سواحل خزر", BadClimate: 0.10, Deprivation: 0.10, RegionHardship: 0.10, ExtraTime: 0.05},
	{Key: "desert", TitleFa: "مناطق بیابانی و مرکزی", BadClimate: 0.20, Deprivation: 0.20, RegionHardship: 0.25, ExtraTime: 0.10},
	{Key: "mountain", TitleFa: "مناطق کوهستانی و سردسیر", BadClimate: 0.15, Deprivation: 0.15, RegionHardship: 0.20, ExtraTime: 0.08},
	{Key: "central", TitleFa: "مرکزی (بدون فوق‌العادهٔ اقلیمی)", BadClimate: 0, Deprivation: 0, RegionHardship: 0, ExtraTime: 0},
}

// ClimateRegionByKey منطقهٔ اقلیمی را با کلید پیدا می‌کند.
func ClimateRegionByKey(key string) (ClimateRegion, bool) {
	for _, r := range ClimateRegionsTable {
		if r.Key == key {
			return r, true
		}
	}
	return ClimateRegion{}, false
}

// ClimateAllowancesFor ضرایب اقلیمی یک منطقه را برمی‌گرداند؛ منطقهٔ ناشناخته صفر است.
func ClimateAllowancesFor(regionKey string) ClimateAllowances {
	if r, ok := ClimateRegionByKey(regionKey); ok {
		return ClimateAllowances{
			BadClimate:     r.BadClimate,
			Deprivation:    r.Deprivation,
			RegionHardship: r.RegionHardship,
			ExtraTime:      r.ExtraTime,
			RegionFa:       r.TitleFa,
		}
	}
	return ClimateAllowances{}
}
