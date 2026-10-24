package rules

// WageComponents اجزای مزد که روی هم «مزد مبنا» را می‌سازند:
// مزد شغل + مزد رتبه + مزد سنوات + فوق‌العادهٔ پست + فوق‌العادهٔ ایثارگری.
//
// این اجزا دادهٔ پارامتریک‌اند و از رکوردهای «پارامتر حقوق» خوانده می‌شوند، نه از کد.
// مزد مبنا مبنای محاسبهٔ فوق‌العاده‌هاست؛ پس هر تغییری در اجزای مزد، فوق‌العاده‌های
// درصدی را هم جابه‌جا می‌کند.
type WageComponents struct {
	Job           float64 `json:"job"`           // مزد شغل
	Rank          float64 `json:"rank"`          // مزد رتبه
	Seniority     float64 `json:"seniority"`     // مزد سنوات
	PostAllowance float64 `json:"postAllowance"` // فوق‌العادهٔ پست
	Sacrifice     float64 `json:"sacrifice"`     // فوق‌العادهٔ ایثارگری
}

// Total مزد مبنا: جمع اجزای مزد.
func (w WageComponents) Total() float64 {
	return w.Job + w.Rank + w.Seniority + w.PostAllowance + w.Sacrifice
}

// Vars متغیرهای فرمول اجزای مزد را می‌سازد تا اقلام حقوق بتوانند به هر جزء یا به
// کل مزد مبنا ارجاع دهند.
func (w WageComponents) Vars() map[string]float64 {
	total := w.Total()
	return map[string]float64{
		"wage.job":            w.Job,
		"wage.rank":           w.Rank,
		"wage.seniority":      w.Seniority,
		"allowance.post":      w.PostAllowance,
		"allowance.sacrifice": w.Sacrifice,
		"wage.total":          total,
		"mazd.base":           total,
		"mazd":                total,
	}
}

// WageComponentsFromVars اجزای مزد را از یک نگاشت متغیر/پارامتر می‌خواند؛ هر جزء
// غایب صفر می‌ماند. همین تابع را شبیه‌ساز و سند چاپی به کار می‌برند.
func WageComponentsFromVars(values map[string]float64) WageComponents {
	return WageComponents{
		Job:           values["wage.job"],
		Rank:          values["wage.rank"],
		Seniority:     values["wage.seniority"],
		PostAllowance: values["allowance.post"],
		Sacrifice:     values["allowance.sacrifice"],
	}
}
