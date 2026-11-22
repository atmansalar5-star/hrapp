// «گروه نظام پرداخت نفت» — بُعد گروه در جدول ۳ آیین‌نامهٔ فوق‌العاده‌ها.
//
// چرا این لایه لازم شد: پیش از این، «کارمند رسمی نفت» با پایهٔ سمت ۶ تا ۱۰ در یک گروه
// شمرده می‌شد؛ یعنی درصدهای جدول ۳ برای او و برای کارکنان پایهٔ ۶ تا ۱۰ یکی بود و
// نمی‌شد مصوبهٔ جدا برای رسمی‌های نفت گذاشت. اکنون:
//
//	گروه یکم  → کارمند رسمی نفت (گروه مستقل خودش)
//	گروه دوم  → پایهٔ سمت ۶ تا ۱۰ (همان چیزی که پیش از این «گروه یکم» شمرده می‌شد)
//	گروه سوم تا دهم → پایه‌های سمت ۱۱–۱۵، ۱۶، ۱۷، A، B، C، D، OD
//
// پس شمارهٔ گروه‌ها یک پله جابه‌جا شد و «کارمند رسمی نفت» از گروه یکم جدا شد. هر گروه
// درصدهای خودش را دارد و همهٔ آن‌ها پارامتری‌اند (`oilpay.group.<شماره>.public|rank`)،
// پس مصوبهٔ سازمان بدون تغییر کد اعمال می‌شود.
package rules

import "strings"

// OilOfficialEmploymentKey کلید «کارمند رسمی نفت» در فهرست انواع استخدام
// (`contracts.EmploymentTypeCatalog`). اینجا فقط کلید مقایسه می‌شود و بستهٔ قراردادها
// وارد نمی‌شود تا لایهٔ قواعد وابسته به لایهٔ داده نشود؛ آزمون یکپارچگی در bootstrap
// هم‌خوانی همین کلید با فهرست انواع استخدام را می‌سنجد.
const OilOfficialEmploymentKey = "official_oil"

// OilPayGroupTitleOfficialOil عنوان فارسی گروه مستقل کارمند رسمی نفت.
const OilPayGroupTitleOfficialOil = "گروه یکم — کارمند رسمی نفت"

// OilPayGroup گروه نظام پرداخت نفت (ستون «گروه» جدول ۳ آیین‌نامه).
type OilPayGroup struct {
	// Number شمارهٔ گروه (۱ = کارمند رسمی نفت).
	Number int `json:"number"`
	// Key کلید ماشین‌خوان گروه.
	Key string `json:"key"`
	// ShortFa عنوان کوتاه («گروه یکم»).
	ShortFa string `json:"shortFa"`
	// TitleFa عنوان کامل برای فرم و فیش.
	TitleFa string `json:"titleFa"`
	// Bases پایه‌های سمت جدول ۳ که در این گروه می‌نشینند (خالی = گروه مستقل، نه پایه‌محور).
	Bases []string `json:"bases,omitempty"`
	// PublicAttract درصد «حق جذب عمومی» و PositionRank درصد «حق مقام» (جمع = Y جدول ۳).
	PublicAttract float64 `json:"publicAttract"`
	PositionRank  float64 `json:"positionRank"`
	// NoteFa توضیح کوتاه برای فرم و راهنما.
	NoteFa string `json:"noteFa,omitempty"`
}

// Total درصد جمع «Y» جدول ۳ برای این گروه.
func (g OilPayGroup) Total() float64 { return g.PublicAttract + g.PositionRank }

// IsOfficialOil آیا این گروه، گروه مستقل کارمند رسمی نفت است.
func (g OilPayGroup) IsOfficialOil() bool { return g.Key == OilGroupKeyOfficialOil }

// کلیدهای گروه‌ها.
const (
	OilGroupKeyOfficialOil = "official_oil"
	OilGroupKeyBase6To10   = "base_6_10"
	OilGroupKeyBase11To15  = "base_11_15"
	OilGroupKeyBase16      = "base_16"
	OilGroupKeyBase17      = "base_17"
	OilGroupKeyBaseA       = "base_A"
	OilGroupKeyBaseB       = "base_B"
	OilGroupKeyBaseC       = "base_C"
	OilGroupKeyBaseD       = "base_D"
	OilGroupKeyBaseOD      = "base_OD"
)

// oilPayGroupCatalog فهرست گروه‌ها به ترتیب شماره.
//
// درصدهای پیش‌فرض همان درصدهای جدول ۳ آیین‌نامه‌اند: گروه یکم (رسمی نفت) و گروه دوم
// (پایهٔ سمت ۶ تا ۱۰) هر دو با ۴٪ جذب عمومی و ۸٪ حق مقام آغاز می‌شوند — یعنی جدا شدن
// رسمی نفت، هیچ عددی را خودبه‌خود تغییر نمی‌دهد؛ تنها برای آن گروه، پارامتر مستقل
// (`oilpay.group.1.public|rank`) باز می‌شود.
var oilPayGroupCatalog = []OilPayGroup{
	{Number: 1, Key: OilGroupKeyOfficialOil, ShortFa: "گروه یکم", TitleFa: OilPayGroupTitleOfficialOil,
		PublicAttract: 4, PositionRank: 8,
		NoteFa: "کارمند رسمی نفت گروه مستقل خودش را دارد (جدا از پایهٔ سمت ۶ تا ۱۰)؛ درصدهای همین گروه مبنای جدول ۳ برای اوست."},
	{Number: 2, Key: OilGroupKeyBase6To10, ShortFa: "گروه دوم", TitleFa: "گروه دوم — پایهٔ سمت ۶ تا ۱۰",
		Bases: []string{"6-10"}, PublicAttract: 4, PositionRank: 8,
		NoteFa: "همان گروهی که پیش از این «گروه یکم» شمرده می‌شد."},
	{Number: 3, Key: OilGroupKeyBase11To15, ShortFa: "گروه سوم", TitleFa: "گروه سوم — پایهٔ سمت ۱۱ تا ۱۵",
		Bases: []string{"11-15"}, PublicAttract: 8, PositionRank: 8},
	{Number: 4, Key: OilGroupKeyBase16, ShortFa: "گروه چهارم", TitleFa: "گروه چهارم — پایهٔ سمت ۱۶",
		Bases: []string{"16"}, PublicAttract: 8, PositionRank: 15},
	{Number: 5, Key: OilGroupKeyBase17, ShortFa: "گروه پنجم", TitleFa: "گروه پنجم — پایهٔ سمت ۱۷",
		Bases: []string{"17"}, PublicAttract: 8, PositionRank: 20},
	{Number: 6, Key: OilGroupKeyBaseA, ShortFa: "گروه ششم", TitleFa: "گروه ششم — پایهٔ سمت A",
		Bases: []string{"A"}, PublicAttract: 8, PositionRank: 25},
	{Number: 7, Key: OilGroupKeyBaseB, ShortFa: "گروه هفتم", TitleFa: "گروه هفتم — پایهٔ سمت B",
		Bases: []string{"B"}, PublicAttract: 8, PositionRank: 35},
	{Number: 8, Key: OilGroupKeyBaseC, ShortFa: "گروه هشتم", TitleFa: "گروه هشتم — پایهٔ سمت C",
		Bases: []string{"C"}, PublicAttract: 8, PositionRank: 60},
	{Number: 9, Key: OilGroupKeyBaseD, ShortFa: "گروه نهم", TitleFa: "گروه نهم — پایهٔ سمت D",
		Bases: []string{"D"}, PublicAttract: 8, PositionRank: 80},
	{Number: 10, Key: OilGroupKeyBaseOD, ShortFa: "گروه دهم", TitleFa: "گروه دهم — پایهٔ سمت OD",
		Bases: []string{"OD"}, PublicAttract: 8, PositionRank: 100},
}

// OilPayGroups فهرست گروه‌های نظام پرداخت نفت (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func OilPayGroups() []OilPayGroup {
	out := make([]OilPayGroup, len(oilPayGroupCatalog))
	copy(out, oilPayGroupCatalog)
	return out
}

// OilPayGroupByNumber یافتن گروه با شماره.
func OilPayGroupByNumber(number int) (OilPayGroup, bool) {
	for _, g := range oilPayGroupCatalog {
		if g.Number == number {
			return g, true
		}
	}
	return OilPayGroup{}, false
}

// OilPayGroupByKey یافتن گروه با کلید.
func OilPayGroupByKey(key string) (OilPayGroup, bool) {
	key = strings.TrimSpace(key)
	for _, g := range oilPayGroupCatalog {
		if g.Key == key {
			return g, true
		}
	}
	return OilPayGroup{}, false
}

// OilPayGroupOfBase گروهی که یک پایهٔ سمت (جدول ۳) در آن می‌نشیند.
func OilPayGroupOfBase(base string) (OilPayGroup, bool) {
	base = strings.TrimSpace(base)
	for _, g := range oilPayGroupCatalog {
		for _, b := range g.Bases {
			if b == base {
				return g, true
			}
		}
	}
	return OilPayGroup{}, false
}

// OilPayGroupOfEmployment گروه کارمند رسمی نفت (کلید لاتین یا عنوان فارسی).
func OilPayGroupOfEmployment(employmentType string) (OilPayGroup, bool) {
	if IsOfficialOilEmployment(employmentType) {
		return OilPayGroupByNumber(1)
	}
	return OilPayGroup{}, false
}

// IsOfficialOilEmployment آیا نوع استخدام «کارمند رسمی نفت» است.
func IsOfficialOilEmployment(employmentType string) bool {
	value := strings.TrimSpace(employmentType)
	return value == OilOfficialEmploymentKey || value == "کارمند رسمی نفت"
}

// OilPayGroupFor گروه نهایی یک کارمند: نخست از نوع استخدام (رسمی نفت = گروه یکم)، سپس
// از پایهٔ سمت جدول ۳. خروجی دوم false یعنی هیچ گروهی تعیین نشد.
func OilPayGroupFor(employmentType, base string) (OilPayGroup, bool) {
	if group, ok := OilPayGroupOfEmployment(employmentType); ok {
		return group, true
	}
	return OilPayGroupOfBase(base)
}

// oilEffectiveGroup گروه نهایی یک محاسبه را تعیین می‌کند و می‌گوید از کجا آمده است.
//
// ترتیب تقدم عمدی است:
//  1. گروه دستی ورودی (PayGroup) — کارشناس می‌تواند برای یک محاسبهٔ خاص گروه را صریح کند.
//  2. نوع استخدام: «کارمند رسمی نفت» همیشه گروه یکم است، حتی اگر پایهٔ سمتش ۶ تا ۱۰ باشد.
//  3. پایهٔ سمت (جدول ۳).
//
// درصدهای گروه از پارامترهای شرکت می‌آیند: `oilpay.group.<شماره>.public|rank`؛ اگر پارامتر
// نبود، مقدار جدول پیش‌فرض همان گروه به کار می‌رود.
func oilEffectiveGroup(in OilPayInput, params OilPayParams) (OilPayGroup, bool, string) {
	if in.PayGroup > 0 {
		if group, ok := oilGroupOfParams(params, in.PayGroup); ok {
			return group, true, "گروه انتخابی کاربر"
		}
	}
	if group, ok := OilPayGroupOfEmployment(in.EmploymentType); ok {
		if withParams, found := oilGroupOfParams(params, group.Number); found {
			return withParams, true, "نوع استخدام (کارمند رسمی نفت)"
		}
		return group, true, "نوع استخدام (کارمند رسمی نفت)"
	}
	if group, ok := OilPayGroupOfBase(in.JobGrade); ok {
		if withParams, found := oilGroupOfParams(params, group.Number); found {
			return withParams, true, "پایهٔ سمت (جدول ۳)"
		}
		return group, true, "پایهٔ سمت (جدول ۳)"
	}
	return OilPayGroup{}, false, ""
}

// oilGroupOfParams گروه را از پارامترهای شرکت می‌خواند (درصدها قابل بازنویسی‌اند)؛
// اگر پارامتری برای آن شماره نبود، گروه پیش‌فرض همان شماره برمی‌گردد.
func oilGroupOfParams(params OilPayParams, number int) (OilPayGroup, bool) {
	for _, g := range params.Groups {
		if g.Number == number {
			// گروهِ پارامتری ممکن است عنوان و پایه‌ها را خالی گذاشته باشد؛ از فهرست
			// پیش‌فرض کامل می‌شود تا فیش عنوان و پایه‌های درست را نشان دهد.
			if base, ok := OilPayGroupByNumber(number); ok {
				if strings.TrimSpace(g.TitleFa) == "" {
					g.TitleFa = base.TitleFa
				}
				if strings.TrimSpace(g.ShortFa) == "" {
					g.ShortFa = base.ShortFa
				}
				if strings.TrimSpace(g.Key) == "" {
					g.Key = base.Key
				}
				if len(g.Bases) == 0 {
					g.Bases = base.Bases
				}
			}
			return g, true
		}
	}
	return OilPayGroupByNumber(number)
}

// OilPayGroupTable3 گزینه‌های گروه برای فرم‌ها و پنل تنظیمات.
func OilPayGroupTable3(params OilPayParams) []OilPayGroup {
	if len(params.Groups) > 0 {
		out := make([]OilPayGroup, len(params.Groups))
		copy(out, params.Groups)
		return out
	}
	return OilPayGroups()
}

// OilPayGroupsTitleFa فهرست «شماره → عنوان» برای نمایش سریع (منو، گزارش، فیش).
func OilPayGroupsTitleFa(params OilPayParams) map[int]string {
	out := map[int]string{}
	for _, g := range OilPayGroupTable3(params) {
		out[g.Number] = g.TitleFa
	}
	return out
}

// OilPayGroupForJobGrade گروه را از نوع استخدام و «گروه شغلی» نردبان سازمان برمی‌آورد.
//
// کاربرد: حکم کارگزینی و قرارداد «پایهٔ سمت» ندارند و «گروه شغلی» (کارشناس، کارشناسی
// ارشد، مدیریت…) دارند؛ این تابع با نگاشت پیشنهادی `GradeSiteMap` پایهٔ سمت آن گروه شغلی
// را می‌یابد و بعد گروه جدول ۳ را تعیین می‌کند. خروجی دوم false یعنی گروه روشن نشد.
func OilPayGroupForJobGrade(params OilPayParams, employmentType, jobGrade string) (OilPayGroup, bool) {
	if group, ok := OilPayGroupOfEmployment(employmentType); ok {
		if withParams, found := oilGroupOfParams(params, group.Number); found {
			return withParams, true
		}
		return group, true
	}
	site := OilSiteForGrade(params, jobGrade)
	if group, ok := OilPayGroupOfBase(site); ok {
		if withParams, found := oilGroupOfParams(params, group.Number); found {
			return withParams, true
		}
		return group, true
	}
	return OilPayGroup{}, false
}

// OilPayGroupForJobGradeNumber همان تابع بالا با خروجی شمارهٔ گروه (۰ = تعیین نشده).
func OilPayGroupForJobGradeNumber(params OilPayParams, employmentType, jobGrade string) int {
	if group, ok := OilPayGroupForJobGrade(params, employmentType, jobGrade); ok {
		return group.Number
	}
	return 0
}

// OilPayGroupTitleFa عنوان فارسی گروه با شماره؛ گروه ناشناخته پیام روشن می‌دهد.
func OilPayGroupTitleFa(number int) string {
	if group, ok := OilPayGroupByNumber(number); ok {
		return group.TitleFa
	}
	if number <= 0 {
		return "تعیین نشده"
	}
	return "گروه ناشناخته"
}

// OilPayGroupShortFa عنوان کوتاه گروه با شماره.
func OilPayGroupShortFa(number int) string {
	if group, ok := OilPayGroupByNumber(number); ok {
		return group.ShortFa
	}
	if number <= 0 {
		return "تعیین نشده"
	}
	return "گروه ناشناخته"
}
