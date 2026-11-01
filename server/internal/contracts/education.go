// قراردادهای مشترک «تحصیلات» — زبان مشترک مقطع‌ها، ارتباط تحصیلات و شورای تعدیل مدرک.
//
// چرا در بستهٔ contracts: سه جای مختلف به همین مرجع نیاز دارند و اگر هر کدام فهرست خودش
// را می‌ساخت، برچسب‌ها و قاعدهٔ «یک مقطع بالاتر برای ایثارگران» در چند جا تکرار می‌شد:
//   - دامنهٔ ماژول بایگانی (اعتبارسنجی رکورد تحصیلات و مدرک ملاک حقوق)
//   - فرم چاپی «تعیین مدرک ملاک حقوق» و کارتابل آموزشی
//   - ساختار سازمانی (نمایش آخرین مدرک و مدرک ملاک متصدی هر پست)
package contracts

import "strings"

// DegreeLevel یک مقطع تحصیلی. Order ترتیب مقاطع است و مبنای «آخرین مدرک» و «یک مقطع
// بالاتر» (سیاست ایثارگران) می‌شود؛ پس ترتیب در داده اعلام می‌شود، نه در کد رابط.
type DegreeLevel struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	Order   int    `json:"order"`
}

// degreeLevelCatalog مقاطع تحصیلی به ترتیب صعودی.
var degreeLevelCatalog = []DegreeLevel{
	{Key: "diploma", TitleFa: "دیپلم", Order: 1},
	{Key: "associate", TitleFa: "کاردانی (فوق دیپلم)", Order: 2},
	{Key: "bachelor", TitleFa: "کارشناسی", Order: 3},
	{Key: "master", TitleFa: "کارشناسی ارشد", Order: 4},
	{Key: "phd", TitleFa: "دکتری", Order: 5},
	{Key: "postdoc", TitleFa: "فوق دکتری (پسادکتری)", Order: 6},
}

// DegreeLevels فهرست مقاطع تحصیلی (کپی، تا مصرف‌کننده اصل را تغییر ندهد).
func DegreeLevels() []DegreeLevel {
	out := make([]DegreeLevel, len(degreeLevelCatalog))
	copy(out, degreeLevelCatalog)
	return out
}

// DegreeLevelByKey یافتن مقطع با کلید.
func DegreeLevelByKey(key string) (DegreeLevel, bool) {
	value := strings.TrimSpace(key)
	for _, level := range degreeLevelCatalog {
		if level.Key == value {
			return level, true
		}
	}
	return DegreeLevel{}, false
}

// DegreeLevelTitleFa عنوان فارسی یک مقطع؛ کلید ناشناخته خودش برمی‌گردد تا پاسخ صادق بماند.
func DegreeLevelTitleFa(key string) string {
	if level, ok := DegreeLevelByKey(key); ok {
		return level.TitleFa
	}
	return strings.TrimSpace(key)
}

// DegreeLevelOrder ترتیب یک مقطع (صفر برای کلید ناشناخته).
func DegreeLevelOrder(key string) int {
	if level, ok := DegreeLevelByKey(key); ok {
		return level.Order
	}
	return 0
}

// UpgradedDegreeLevel مقطع «یک پله بالاتر» یک مدرک.
//
// چرا در دامنه و نه در رابط: سیاست سازمان برای ایثارگران این است که مدرک تحصیلی
// یک مقطع بالاتر ملاک می‌شود؛ این یک قاعدهٔ کسب‌وکاری است، پس در دامنه اعلام می‌شود و
// هم فرم ورود و هم فرم چاپی و هم گزارش‌ها از همین تابع می‌خوانند. اگر مدرک بالاترین
// مقطع باشد، ok=false برمی‌گردد تا هیچ مقطعی از خودِ فهرست بیرون ساخته نشود.
func UpgradedDegreeLevel(key string) (DegreeLevel, bool) {
	current, ok := DegreeLevelByKey(key)
	if !ok {
		return DegreeLevel{}, false
	}
	var next DegreeLevel
	found := false
	for _, level := range degreeLevelCatalog {
		if level.Order == current.Order+1 {
			next, found = level, true
			break
		}
	}
	return next, found
}

// HighestDegreeLevel بالاترین مقطع از میان کلیدهای داده‌شده — همان «آخرین مدرک تحصیلی».
func HighestDegreeLevel(keys []string) (DegreeLevel, bool) {
	var best DegreeLevel
	found := false
	for _, key := range keys {
		level, ok := DegreeLevelByKey(key)
		if !ok {
			continue
		}
		if !found || level.Order > best.Order {
			best, found = level, true
		}
	}
	return best, found
}

// DegreeRelevance نتیجهٔ بررسی «ارتباط تحصیلات با شغل».
type DegreeRelevance struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// degreeRelevanceCatalog نتیجه‌های ممکن ارتباط مدرک.
var degreeRelevanceCatalog = []DegreeRelevance{
	{Key: "related", TitleFa: "مرتبط"},
	{Key: "partially_related", TitleFa: "مرتبط نسبی (هم‌خانواده)"},
	{Key: "not_related", TitleFa: "غیرمرتبط"},
}

// DegreeRelevances فهرست نتیجه‌های ارتباط مدرک.
func DegreeRelevances() []DegreeRelevance {
	out := make([]DegreeRelevance, len(degreeRelevanceCatalog))
	copy(out, degreeRelevanceCatalog)
	return out
}

// DegreeRelevanceTitleFa عنوان فارسی نتیجهٔ ارتباط؛ کلید ناشناخته خودش برمی‌گردد.
func DegreeRelevanceTitleFa(key string) string {
	value := strings.TrimSpace(key)
	for _, item := range degreeRelevanceCatalog {
		if item.Key == value {
			return item.TitleFa
		}
	}
	return value
}

// DegreeRelated آسان‌خوان: آیا مدرک «مرتبط» (کامل یا نسبی) ارزیابی شده است.
//
// قاعدهٔ سود: مدرکی که «غیرمرتبط» است نمی‌تواند ملاک حقوق شود؛ ولی «مرتبط نسبی» با
// درصد ارتباط ثبت‌شده پذیرفته است. این تابع همان سنجش را در یک جا نگه می‌دارد.
func DegreeRelated(key string) bool {
	value := strings.TrimSpace(key)
	return value == "related" || value == "partially_related"
}

// DegreeCouncilStatus وضعیت بررسی مدرک در «شورای تعدیل مدرک».
type DegreeCouncilStatus struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// degreeCouncilCatalog وضعیت‌های ممکن شورای تعدیل مدرک.
var degreeCouncilCatalog = []DegreeCouncilStatus{
	{Key: "not_reviewed", TitleFa: "بررسی‌نشده"},
	{Key: "in_review", TitleFa: "در دست بررسی شورا"},
	{Key: "approved", TitleFa: "تأییدشدهٔ شورای تعدیل مدرک"},
	{Key: "rejected", TitleFa: "ردشده در شورا"},
}

// DegreeCouncilStatuses فهرست وضعیت‌های شورای تعدیل مدرک.
func DegreeCouncilStatuses() []DegreeCouncilStatus {
	out := make([]DegreeCouncilStatus, len(degreeCouncilCatalog))
	copy(out, degreeCouncilCatalog)
	return out
}

// DegreeCouncilStatusTitleFa عنوان فارسی وضعیت شورا.
func DegreeCouncilStatusTitleFa(key string) string {
	value := strings.TrimSpace(key)
	for _, item := range degreeCouncilCatalog {
		if item.Key == value {
			return item.TitleFa
		}
	}
	return value
}

// YesNo یک گزینهٔ بله/خیر برای فیلدهای انتخابی رکوردها.
type YesNo struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
}

// YesNos فهرست گزینه‌های بله/خیر (کلید ماشین‌خوان می‌ماند، برچسب فارسی نمایش داده می‌شود).
func YesNos() []YesNo {
	return []YesNo{{Key: "yes", TitleFa: "بله"}, {Key: "no", TitleFa: "خیر"}}
}

// YesNoTitleFa عنوان فارسی بله/خیر؛ کلید ناشناخته خودش برمی‌گردد.
func YesNoTitleFa(key string) string {
	switch strings.TrimSpace(key) {
	case "yes":
		return "بله"
	case "no":
		return "خیر"
	default:
		return strings.TrimSpace(key)
	}
}
