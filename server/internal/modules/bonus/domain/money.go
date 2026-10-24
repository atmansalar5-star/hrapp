// منطق دامنهٔ ماژول «bonus» — «money»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"strconv"
	"strings"

	"hrms/internal/platform/calendar"
)

// AmountOf مبلغ را از رشتهٔ فیلد رکورد می‌خواند. ارقام فارسی و جداکننده‌های هزارگان و
// ممیز فارسی پذیرفته می‌شوند تا دادهٔ واردشده از فرم یا فایل، درست تفسیر شود.
func AmountOf(v string) (int64, bool) {
	v = calendar.NormalizeDigits(strings.TrimSpace(v))
	v = strings.NewReplacer(",", "", "٬", "", "٫", ".", " ", "").Replace(v)
	if v == "" {
		return 0, false
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i, true
	}
	// ممیز اعشاری: بخش صحیح ملاک است (مبالغ ریالی اعشار ندارند).
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return int64(f), true
	}
	return 0, false
}

// PeriodOf دورهٔ شمسی «YYYY/MM» را از تاریخ «YYYY/MM/DD» بیرون می‌کشد. تاریخ ناسالم
// رشتهٔ خالی می‌دهد تا در گزارش‌ها به دورهٔ «نامشخص» نچسبد.
func PeriodOf(date string) string {
	d := calendar.NormalizeDigits(strings.TrimSpace(date))
	if len(d) < 7 {
		return ""
	}
	year, month := d[0:4], d[5:7]
	if !allDigits(year) || !allDigits(month) {
		return ""
	}
	return year + "/" + month
}

// SequenceOf شمارهٔ دفعه را می‌خواند؛ صفر یعنی اعلام نشده.
func SequenceOf(v string) int {
	n, ok := AmountOf(v)
	if !ok || n < 0 {
		return 0
	}
	return int(n)
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
