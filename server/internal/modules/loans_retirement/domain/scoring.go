// منطق دامنهٔ ماژول «loans_retirement» — «scoring»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"sort"
	"strings"
)

// CriterionOption یک گزینهٔ معیار امتیازبندی با امتیاز پایهٔ خودش.
type CriterionOption struct {
	Value   string
	TitleFa string
	Score   int
}

// Criterion یک معیار امتیازبندی تخصیص وام؛ Weight ضریب اهمیت معیار است و امتیاز نهایی
// هر معیار = امتیاز گزینه × ضریب. این معیارها در دامنه اعلام می‌شوند و UI فقط نمایش
// می‌دهد، پس امتیازبندی در سرور یکسان و قابل‌حسابرسی است.
type Criterion struct {
	Key     string
	TitleFa string
	Weight  int
	Options []CriterionOption
}

// criteria معیارهای امتیازبندی تخصیص وام.
func criteria() []Criterion {
	return []Criterion{
		{Key: "education", TitleFa: "مدرک تحصیلی", Weight: 2, Options: []CriterionOption{
			{Value: "belowDiploma", TitleFa: "زیر دیپلم", Score: 2},
			{Value: "diploma", TitleFa: "دیپلم", Score: 5},
			{Value: "associate", TitleFa: "کاردانی", Score: 10},
			{Value: "bachelor", TitleFa: "کارشناسی", Score: 15},
			{Value: "master", TitleFa: "کارشناسی ارشد", Score: 20},
			{Value: "phd", TitleFa: "دکتری", Score: 25},
		}},
		{Key: "orgUnit", TitleFa: "واحد سازمانی", Weight: 1, Options: []CriterionOption{
			{Value: "port", TitleFa: "عملیاتی بندری", Score: 12},
			{Value: "technical", TitleFa: "فنی", Score: 10},
			{Value: "staff", TitleFa: "ستادی", Score: 6},
			{Value: "support", TitleFa: "پشتیبانی", Score: 4},
		}},
		{Key: "shift", TitleFa: "نوبتکاری", Weight: 2, Options: []CriterionOption{
			{Value: "none", TitleFa: "بدون نوبت", Score: 0},
			{Value: "day", TitleFa: "روزکار", Score: 4},
			{Value: "rotational", TitleFa: "نوبتی", Score: 8},
			{Value: "night", TitleFa: "شب‌کار", Score: 12},
		}},
		{Key: "workDays", TitleFa: "روزکاری", Weight: 1, Options: []CriterionOption{
			{Value: "full", TitleFa: "تمام‌وقت", Score: 10},
			{Value: "part", TitleFa: "پاره‌وقت", Score: 4},
		}},
		{Key: "veteran", TitleFa: "ایثارگری", Weight: 3, Options: []CriterionOption{
			{Value: "none", TitleFa: "بدون سابقهٔ ایثارگری", Score: 0},
			{Value: "family", TitleFa: "خانوادهٔ ایثارگر", Score: 8},
			{Value: "veteran", TitleFa: "ایثارگر", Score: 20},
			{Value: "freed", TitleFa: "آزاده", Score: 25},
		}},
		{Key: "position", TitleFa: "سمت سازمانی", Weight: 2, Options: []CriterionOption{
			{Value: "employee", TitleFa: "کارمند", Score: 5},
			{Value: "supervisor", TitleFa: "سرپرست", Score: 10},
			{Value: "head", TitleFa: "رئیس", Score: 15},
			{Value: "manager", TitleFa: "مدیر", Score: 20},
		}},
		{Key: "seniority", TitleFa: "سابقهٔ خدمت", Weight: 2, Options: []CriterionOption{
			{Value: "lt5", TitleFa: "کمتر از ۵ سال", Score: 5},
			{Value: "5to10", TitleFa: "۵ تا ۱۰ سال", Score: 10},
			{Value: "10to15", TitleFa: "۱۰ تا ۱۵ سال", Score: 15},
			{Value: "gt15", TitleFa: "بیش از ۱۵ سال", Score: 20},
		}},
		{Key: "marital", TitleFa: "وضعیت تأهل", Weight: 1, Options: []CriterionOption{
			{Value: "single", TitleFa: "مجرد", Score: 3},
			{Value: "married", TitleFa: "متأهل", Score: 8},
			{Value: "children", TitleFa: "متأهل با فرزند", Score: 12},
		}},
		{Key: "housing", TitleFa: "وضعیت سکونت", Weight: 2, Options: []CriterionOption{
			{Value: "withHouse", TitleFa: "خانهٔ شخصی", Score: 0},
			{Value: "rented", TitleFa: "استیجاری", Score: 6},
			{Value: "dorm", TitleFa: "اسکان سازمان", Score: 8},
			{Value: "homeless", TitleFa: "بدون مسکن", Score: 10},
		}},
		{Key: "disability", TitleFa: "معلولیت/بیماری خاص", Weight: 2, Options: []CriterionOption{
			{Value: "none", TitleFa: "ندارد", Score: 0},
			{Value: "partial", TitleFa: "جزئی", Score: 8},
			{Value: "severe", TitleFa: "شدید", Score: 12},
		}},
	}
}

// Criteria فهرست معیارهای امتیازبندی برای رابط و گزارش‌ها.
func Criteria() []Criterion { return criteria() }

// Applicant متقاضی وام با مقادیر هر معیار (کلید معیار → مقدار گزینه).
type Applicant struct {
	EmployeeID   string
	EmployeeName string
	Values       map[string]string
}

// ScoreLine امتیاز یک معیار در نتیجهٔ امتیازبندی.
type ScoreLine struct {
	Criterion   string
	CriterionFa string
	ValueFa     string
	Score       int
	Weight      int
	Points      int
}

// ScoreResult نتیجهٔ امتیازبندی یک متقاضی.
type ScoreResult struct {
	EmployeeID   string
	EmployeeName string
	Total        int
	Lines        []ScoreLine
}

// optionOf یافتن گزینهٔ یک معیار با مقدار.
func optionOf(c Criterion, value string) (CriterionOption, bool) {
	value = strings.TrimSpace(value)
	for _, o := range c.Options {
		if o.Value == value {
			return o, true
		}
	}
	return CriterionOption{}, false
}

// Score امتیاز یک متقاضی را بر پایهٔ معیارها می‌سازد. گزینهٔ ناشناخته یا خالی صفر
// می‌گیرد ولی از نتیجه حذف نمی‌شود تا ریز امتیاز همیشه کامل و قابل حسابرسی باشد.
func Score(a Applicant) ScoreResult {
	res := ScoreResult{EmployeeID: a.EmployeeID, EmployeeName: a.EmployeeName}
	for _, c := range criteria() {
		o, ok := optionOf(c, a.Values[c.Key])
		line := ScoreLine{
			Criterion: c.Key, CriterionFa: c.TitleFa, Weight: c.Weight,
		}
		if ok {
			line.ValueFa = o.TitleFa
			line.Score = o.Score
			line.Points = o.Score * c.Weight
		}
		res.Total += line.Points
		res.Lines = append(res.Lines, line)
	}
	return res
}

// Rank متقاضیان را امتیازبندی و از بیشترین امتیاز به کمترین مرتب می‌کند؛ ترتیب
// تخصیص وام از همین رتبه‌بندی می‌آید.
func Rank(applicants []Applicant) []ScoreResult {
	out := make([]ScoreResult, 0, len(applicants))
	for _, a := range applicants {
		out = append(out, Score(a))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Total == out[j].Total {
			return out[i].EmployeeID < out[j].EmployeeID
		}
		return out[i].Total > out[j].Total
	})
	return out
}

// DefaultApplicants متقاضیان نمونه برای پیش‌نمایش امتیازبندی در رابط.
func DefaultApplicants() []Applicant {
	return []Applicant{
		{EmployeeID: "EMP-005", EmployeeName: "محمد احمدی", Values: map[string]string{
			"education": "master", "orgUnit": "port", "shift": "night", "workDays": "full",
			"veteran": "veteran", "position": "head", "seniority": "gt15", "marital": "children",
			"housing": "rented", "disability": "none"}},
		{EmployeeID: "EMP-003", EmployeeName: "نگار حسینی", Values: map[string]string{
			"education": "bachelor", "orgUnit": "staff", "shift": "day", "workDays": "full",
			"veteran": "none", "position": "employee", "seniority": "5to10", "marital": "married",
			"housing": "withHouse", "disability": "none"}},
		{EmployeeID: "EMP-007", EmployeeName: "رضا کریمی", Values: map[string]string{
			"education": "associate", "orgUnit": "technical", "shift": "rotational", "workDays": "full",
			"veteran": "family", "position": "supervisor", "seniority": "10to15", "marital": "married",
			"housing": "dorm", "disability": "partial"}},
		{EmployeeID: "EMP-011", EmployeeName: "زهرا موسوی", Values: map[string]string{
			"education": "diploma", "orgUnit": "support", "shift": "none", "workDays": "part",
			"veteran": "none", "position": "employee", "seniority": "lt5", "marital": "single",
			"housing": "homeless", "disability": "none"}},
		{EmployeeID: "EMP-002", EmployeeName: "سعید نادری", Values: map[string]string{
			"education": "phd", "orgUnit": "port", "shift": "night", "workDays": "full",
			"veteran": "freed", "position": "manager", "seniority": "gt15", "marital": "children",
			"housing": "homeless", "disability": "severe"}},
	}
}
