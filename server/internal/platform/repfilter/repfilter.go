// Package repfilter اعمال شرط‌های گزارش روی سطرهای یک منبع دادهٔ گزارش.
//
// چرا این بسته وجود دارد: گزارش‌ساز روی هر منبعی («کارکنان»، «وام کارکنان»…)
// کار می‌کند و کاربر باید بتواند روی اقلام آن شرط بگذارد (برابر، بزرگ‌تر، شامل،
// بین دو مقدار…). این منطق مشترک اینجا و یک‌جا نوشته می‌شود تا هر منبع داده فقط
// مقادیر سطر را بدهد، و هیچ قاعده‌ای در UI تکرار نشود. عملگرها هم همین‌جا با
// برچسب فارسی اعلام می‌شوند تا سرور تنها منبع حقیقت باشد.
package repfilter

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

// عملگرهای استاندارد شرط گزارش.
const (
	OpEq       = "eq"
	OpNe       = "ne"
	OpContains = "contains"
	OpGt       = "gt"
	OpGte      = "gte"
	OpLt       = "lt"
	OpLte      = "lte"
	OpBetween  = "between"
	OpIn       = "in"
	OpNotEmpty = "notempty"
)

// Operators فهرست عملگرهای پشتیبانی‌شده با برچسب فارسی برای نمایش در رابط.
func Operators() []contracts.ReportOperator {
	return []contracts.ReportOperator{
		{Key: OpEq, LabelFa: "برابر با"},
		{Key: OpNe, LabelFa: "نابرابر با"},
		{Key: OpContains, LabelFa: "شامل"},
		{Key: OpGt, LabelFa: "بزرگ‌تر از"},
		{Key: OpGte, LabelFa: "بزرگ‌تر یا مساوی"},
		{Key: OpLt, LabelFa: "کوچک‌تر از"},
		{Key: OpLte, LabelFa: "کوچک‌تر یا مساوی"},
		{Key: OpBetween, LabelFa: "بین دو مقدار"},
		{Key: OpIn, LabelFa: "یکی از مقادیر (با کاما)"},
		{Key: OpNotEmpty, LabelFa: "مقدار دارد"},
	}
}

// Valid بررسی می‌کند عملگر شناخته‌شده است.
func Valid(op string) bool {
	for _, o := range Operators() {
		if o.Key == strings.TrimSpace(op) {
			return true
		}
	}
	return false
}

// Validate همهٔ شرط‌ها را بررسی می‌کند و اولین خطا را برمی‌گرداند.
func Validate(conds []contracts.ReportCondition) error {
	for i, c := range conds {
		if strings.TrimSpace(c.Field) == "" {
			return &contracts.ReportConditionError{Index: i + 1, Message: "قلم شرط انتخاب نشده است."}
		}
		if !Valid(c.Op) {
			return &contracts.ReportConditionError{Index: i + 1, Message: "عملگر شرط «" + c.Op + "» شناخته نشد."}
		}
		if c.Op != OpNotEmpty && strings.TrimSpace(c.Value) == "" {
			return &contracts.ReportConditionError{Index: i + 1, Message: "مقدار شرط خالی است."}
		}
		if c.Op == OpBetween && strings.TrimSpace(c.Value2) == "" {
			return &contracts.ReportConditionError{Index: i + 1, Message: "برای «بین دو مقدار» باید هر دو سر بازه پر شود."}
		}
	}
	return nil
}

// Matches وقتی درست است که *همهٔ* شرط‌ها روی مقادیر سطر برقرار باشند.
//
// مقایسهٔ عددی وقتی هر دو طرف عدد باشند انجام می‌شود و در غیر این صورت مقایسهٔ
// رشته‌ای؛ چون تاریخ‌ها به قالب YYYY/MM/DD هستند، مقایسهٔ رشته‌ای هم درست کار می‌کند.
func Matches(values map[string]string, conds []contracts.ReportCondition) bool {
	for _, c := range conds {
		if strings.TrimSpace(c.Field) == "" {
			continue
		}
		if !matchOne(strings.TrimSpace(values[c.Field]), c) {
			return false
		}
	}
	return true
}

func matchOne(raw string, c contracts.ReportCondition) bool {
	raw = strings.TrimSpace(raw)
	switch strings.TrimSpace(c.Op) {
	case OpEq:
		return equal(raw, c.Value)
	case OpNe:
		return !equal(raw, c.Value)
	case OpContains:
		return strings.Contains(normalize(raw), normalize(c.Value))
	case OpGt:
		return compare(raw, c.Value) > 0
	case OpGte:
		return compare(raw, c.Value) >= 0
	case OpLt:
		return compare(raw, c.Value) < 0
	case OpLte:
		return compare(raw, c.Value) <= 0
	case OpBetween:
		return compare(raw, c.Value) >= 0 && compare(raw, c.Value2) <= 0
	case OpIn:
		for _, want := range strings.Split(c.Value, ",") {
			if equal(raw, strings.TrimSpace(want)) {
				return true
			}
		}
		return false
	case OpNotEmpty:
		return raw != ""
	}
	return true
}

// equal مقایسهٔ برابری؛ اگر هر دو عدد باشند عددی وگرنه رشته‌ای (با نرمال‌سازی ارقام).
func equal(a, b string) bool {
	af, aok := floatOf(a)
	bf, bok := floatOf(b)
	if aok && bok {
		return af == bf
	}
	return strings.EqualFold(normalize(a), normalize(b))
}

// compare اگر هر دو طرف عدد باشند مقایسهٔ عددی و در غیر این صورت مقایسهٔ رشته‌ای.
func compare(a, b string) int {
	af, aok := floatOf(a)
	bf, bok := floatOf(b)
	if aok && bok {
		switch {
		case af < bf:
			return -1
		case af > bf:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(normalize(a), normalize(b))
}

func normalize(v string) string {
	return calendar.NormalizeDigits(strings.TrimSpace(v))
}

func floatOf(v string) (float64, bool) {
	v = normalize(v)
	if v == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}
