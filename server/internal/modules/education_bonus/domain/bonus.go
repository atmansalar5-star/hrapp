// Package domain دامنهٔ ماژول پاداش تحصیلی فرزندان ممتاز.
//
// این ماژول مستقل از پتروکارت و نفت‌کارت است: مقاطع تحصیلی، مبالغ مصوب هر مقطع و
// موتور محاسبهٔ پاداش را به‌تنهایی نگه می‌دارد. رابط و API هیچ برچسب یا قاعده‌ای را
// از خودشان نمی‌سازند.
package domain

import (
	"sort"
	"strings"
)

// Level یک مقطع تحصیلی با مبلغ مصوب پیش‌فرض پاداش.
type Level struct {
	Key         string
	TitleFa     string
	DefaultRate int64
}

// Levels فهرست مقاطع تحصیلی — تنها منبع حقیقت پاداش تحصیلی.
func Levels() []Level {
	return []Level{
		{Key: "preschool", TitleFa: "پیش‌دبستانی", DefaultRate: 4_000_000},
		{Key: "elementary", TitleFa: "ابتدایی", DefaultRate: 8_000_000},
		{Key: "middle1", TitleFa: "متوسطه اول", DefaultRate: 10_000_000},
		{Key: "high2", TitleFa: "متوسطه دوم", DefaultRate: 15_000_000},
		{Key: "vocational", TitleFa: "هنرستان و فنی‌وحرفه‌ای", DefaultRate: 15_000_000},
		{Key: "university", TitleFa: "دانشگاهی", DefaultRate: 20_000_000},
	}
}

// LevelByKey یافتن یک مقطع با کلید.
func LevelByKey(key string) (Level, bool) {
	for _, l := range Levels() {
		if l.Key == key {
			return l, true
		}
	}
	return Level{}, false
}

// LevelTitleFa عنوان فارسی یک مقطع؛ برای کلید ناشناخته خود کلید برمی‌گردد.
func LevelTitleFa(key string) string {
	if l, ok := LevelByKey(key); ok {
		return l.TitleFa
	}
	return key
}

// DefaultRates مبلغ مصوب پیش‌فرض هر مقطع (برای بذرکاری و پشتیبان).
func DefaultRates() map[string]int64 {
	out := map[string]int64{}
	for _, l := range Levels() {
		out[l.Key] = l.DefaultRate
	}
	return out
}

// Student یک دانش‌آموز/دانشجوی ممتاز مشمول پاداش.
type Student struct {
	RecordID     string
	EmployeeID   string
	EmployeeName string
	Name         string
	Relation     string
	Level        string
	Grade        string
	Excellent    bool
	Eligible     bool
}

// BonusInput ورودی محاسبهٔ پاداش تحصیلی.
type BonusInput struct {
	Levels   []string // مقطع‌های خواسته‌شده؛ خالی یعنی همهٔ مقاطع
	Rates    map[string]int64
	Students []Student
	Period   string
}

// BonusLine یک ردیف پاداش برای یک دانش‌آموز ممتاز.
type BonusLine struct {
	EmployeeID   string
	EmployeeName string
	Student      string
	Relation     string
	Level        string
	LevelFa      string
	Grade        string
	Amount       int64
	BasisFa      string
}

// Build ردیف‌های پاداش را از روی مشمولان و مبالغ مصوب می‌سازد.
//
// قواعد:
//   - فقط مقطع‌های خواسته‌شده ساخته می‌شوند؛ مقطع ناشناخته هشدار می‌دهد.
//   - دانش‌آموزی که نشانهٔ «ممتاز» ندارد مشمول پاداش نیست (هشدار می‌دهد).
//   - دانش‌آموز غیرمشمول (eligible=0) بی‌صدا نادیده گرفته می‌شود.
//   - مبلغ مصوب نبودن یا صفر بودن به مبلغ پیش‌فرض همان مقطع برمی‌گردد؛ اگر آن هم
//     صفر باشد، ردیفی ساخته نمی‌شود و هشدار می‌دهد.
func Build(in BonusInput) ([]BonusLine, []string) {
	warnings := []string{}

	allowed := Levels()
	chosen := allowed
	if len(in.Levels) > 0 {
		chosen = nil
		for _, key := range in.Levels {
			key = strings.TrimSpace(key)
			l, ok := LevelByKey(key)
			if !ok {
				warnings = append(warnings, "مقطع ناشناخته نادیده گرفته شد: "+key)
				continue
			}
			chosen = append(chosen, l)
		}
		if len(chosen) == 0 {
			warnings = append(warnings, "هیچ مقطعی انتخاب نشده است.")
		}
	}
	inChosen := map[string]bool{}
	for _, l := range chosen {
		inChosen[l.Key] = true
	}

	rate := func(l Level) (int64, bool) {
		if v, ok := in.Rates[l.Key]; ok && v > 0 {
			return v, true
		}
		if l.DefaultRate > 0 {
			warnings = append(warnings, "مبلغ مصوب «"+l.TitleFa+"» تعریف نشده است؛ مبلغ پیش‌فرض به‌کار رفت.")
			return l.DefaultRate, true
		}
		warnings = append(warnings, "مبلغ مصوب «"+l.TitleFa+"» تعریف نشده است؛ ردیفی ساخته نشد.")
		return 0, false
	}

	lines := make([]BonusLine, 0, len(in.Students))
	for _, s := range in.Students {
		if !s.Eligible {
			continue
		}
		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = s.EmployeeName
		}
		if !inChosen[s.Level] {
			continue
		}
		if !s.Excellent {
			warnings = append(warnings, "پاداش تحصیلی برای «"+name+"» تعلق نگرفت: نشانهٔ دانش‌آموز ممتاز ثبت نشده است.")
			continue
		}
		l, ok := LevelByKey(s.Level)
		if !ok {
			warnings = append(warnings, "مقطع «"+s.Level+"» برای «"+name+"» ناشناخته است؛ ردیفی ساخته نشد.")
			continue
		}
		amount, ok := rate(l)
		if !ok {
			continue
		}
		lines = append(lines, BonusLine{
			EmployeeID:   s.EmployeeID,
			EmployeeName: s.EmployeeName,
			Student:      name,
			Relation:     s.Relation,
			Level:        l.Key,
			LevelFa:      l.TitleFa,
			Grade:        s.Grade,
			Amount:       amount,
			BasisFa:      "مبلغ مصوب مقطع " + l.TitleFa,
		})
	}

	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].EmployeeID != lines[j].EmployeeID {
			return lines[i].EmployeeID < lines[j].EmployeeID
		}
		return lines[i].Student < lines[j].Student
	})
	return lines, warnings
}

// SumLines جمع مبالغ ردیف‌ها و جمع هر مقطع.
func SumLines(lines []BonusLine) (int64, map[string]int64, map[string]int) {
	total := int64(0)
	byLevel := map[string]int64{}
	count := map[string]int{}
	for _, l := range lines {
		total += l.Amount
		byLevel[l.Level] += l.Amount
		count[l.Level]++
	}
	return total, byLevel, count
}
