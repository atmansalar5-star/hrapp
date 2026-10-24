package rules

// KpiItem یک شاخص ارزیابی با وزن و نمره (مقیاس ۰ تا ۵).
type KpiItem struct {
	KpiID  string  `json:"kpiId"`
	Weight float64 `json:"weight"`
	Score  float64 `json:"score"`
}

// KpiOutcome نتیجهٔ ارزیابی یک کارمند.
type KpiOutcome struct {
	TotalWeight   float64 `json:"totalWeight"`
	WeightedScore float64 `json:"weightedScore"`
	GradeFa       string  `json:"gradeFa"`
}

// WeightedScore میانگین موزون نمره‌ها؛ جمع وزن صفر یعنی نمرهٔ صفر.
func WeightedScore(items []KpiItem) float64 {
	var totalWeight float64
	for _, i := range items {
		totalWeight += i.Weight
	}
	if totalWeight <= 0 {
		return 0
	}
	var sum float64
	for _, i := range items {
		sum += i.Weight * i.Score
	}
	return round2(sum / totalWeight)
}

// GradeFa رتبهٔ فارسی بر پایهٔ نمرهٔ موزون.
func GradeFa(score float64, hasWeight bool) string {
	if !hasWeight {
		return "بدون ارزیابی"
	}
	switch {
	case score >= 4.5:
		return "عالی"
	case score >= 3.5:
		return "خوب"
	case score >= 2.5:
		return "قابل قبول"
	default:
		return "نیازمند بهبود"
	}
}

// Evaluate ارزیابی کامل یک فرم.
func Evaluate(items []KpiItem) KpiOutcome {
	var totalWeight float64
	for _, i := range items {
		totalWeight += i.Weight
	}
	score := WeightedScore(items)
	return KpiOutcome{
		TotalWeight:   totalWeight,
		WeightedScore: score,
		GradeFa:       GradeFa(score, totalWeight > 0),
	}
}
