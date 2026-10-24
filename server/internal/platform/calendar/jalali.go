// Package calendar تقویم شمسی (هجری شمسی) — همان الگوریتمی که در core/crates/hrms-calendar پیاده شده
// است. اختلاف این دو پیاده‌سازی با fixtureهای core/testdata/conformance/calendar.json در CI گرفته می‌شود.
package calendar

import (
	"fmt"
	"strings"
	"time"
)

var breaks = []int{-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178}

// Date یک تاریخ شمسی.
type Date struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

var persianDigits = []string{"۰", "۱", "۲", "۳", "۴", "۵", "۶", "۷", "۸", "۹"}
var arabicDigits = []string{"٠", "١", "٢", "٣", "٤", "٥", "٦", "٧", "٨", "٩"}

// NormalizeDigits ارقام فارسی/عربی را به لاتین تبدیل می‌کند.
func NormalizeDigits(s string) string {
	replacer := strings.NewReplacer(
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	)
	return replacer.Replace(s)
}

// ToPersianDigits نمایش ارقام به فارسی برای پیام‌های کاربر.
func ToPersianDigits(s string) string {
	out := s
	for i := 0; i < 10; i++ {
		out = strings.ReplaceAll(out, string(rune('0'+i)), persianDigits[i])
	}
	return out
}

func intDiv(a, b int) int { return a / b }
func intMod(a, b int) int { return a - intDiv(a, b)*b }

type jalCalResult struct {
	leap  int
	gy    int
	march int
}

func jalCal(jy int) (jalCalResult, error) {
	bl := len(breaks)
	if jy < breaks[0] || jy >= breaks[bl-1] {
		return jalCalResult{}, fmt.Errorf("سال شمسی خارج از بازهٔ پشتیبانی‌شده است: %d", jy)
	}
	gy := jy + 621
	leapJ := -14
	jp := breaks[0]
	jump := 0

	i := 1
	for ; i < bl; i++ {
		jm := breaks[i]
		jump = jm - jp
		if jy < jm {
			break
		}
		leapJ = leapJ + intDiv(jump, 33)*8 + intDiv(intMod(jump, 33), 4)
		jp = jm
	}
	n := jy - jp
	leapJ = leapJ + intDiv(n, 33)*8 + intDiv(intMod(n, 33)+3, 4)
	if intMod(jump, 33) == 4 && jump-n == 4 {
		leapJ++
	}
	leapG := intDiv(gy, 4) - intDiv((intDiv(gy, 100)+1)*3, 4) - 150
	march := 20 + leapJ - leapG

	if jump-n < 6 {
		n = n - jump + intDiv(jump+4, 33)*33
	}
	leap := intMod(intMod(n+1, 33)-1, 4)
	if leap == -1 {
		leap = 4
	}
	return jalCalResult{leap: leap, gy: gy, march: march}, nil
}

func g2d(gy, gm, gd int) int {
	d := intDiv((gy+intDiv(gm-8, 6)+100100)*1461, 4) + intDiv(153*intMod(gm+9, 12)+2, 5) + gd - 34840408
	d = d - intDiv(intDiv(gy+100100+intDiv(gm-8, 6), 100)*3, 4) + 752
	return d
}

func d2g(jdn int) (int, int, int) {
	j := 4*jdn + 139361631
	j = j + intDiv(intDiv(4*jdn+183187720, 146097)*3, 4)*4 - 3908
	i := intDiv(intMod(j, 1461), 4)*5 + 308
	gd := intDiv(intMod(i, 153), 5) + 1
	gm := intMod(intDiv(i, 153), 12) + 1
	gy := intDiv(j, 1461) - 100100 + intDiv(8-gm, 6)
	return gy, gm, gd
}

func j2d(jy, jm, jd int) (int, error) {
	r, err := jalCal(jy)
	if err != nil {
		return 0, err
	}
	return g2d(r.gy, 3, r.march) + (jm-1)*31 - intDiv(jm, 7)*(jm-7) + jd - 1, nil
}

func d2j(jdn int) (Date, error) {
	gy, _, _ := d2g(jdn)
	jy := gy - 621
	r, err := jalCal(jy)
	if err != nil {
		return Date{}, err
	}
	jdn1f := g2d(gy, 3, r.march)
	k := jdn - jdn1f
	if k >= 0 {
		if k <= 185 {
			return Date{Year: jy, Month: 1 + intDiv(k, 31), Day: intMod(k, 31) + 1}, nil
		}
		k -= 186
	} else {
		jy--
		k += 179
		if r.leap == 1 {
			k++
		}
	}
	return Date{Year: jy, Month: 7 + intDiv(k, 30), Day: intMod(k, 30) + 1}, nil
}

// IsLeapYear آیا سال شمسی کبیسه است.
func IsLeapYear(jy int) (bool, error) {
	r, err := jalCal(jy)
	if err != nil {
		return false, err
	}
	return r.leap == 0, nil
}

// MonthLength طول ماه شمسی.
func MonthLength(jy, jm int) (int, error) {
	if jm < 1 || jm > 12 {
		return 0, fmt.Errorf("شمارهٔ ماه نامعتبر است: %d", jm)
	}
	if jm <= 6 {
		return 31, nil
	}
	if jm <= 11 {
		return 30, nil
	}
	leap, err := IsLeapYear(jy)
	if err != nil {
		return 0, err
	}
	if leap {
		return 30, nil
	}
	return 29, nil
}

// ToGregorian تبدیل تاریخ شمسی به میلادی.
func ToGregorian(jy, jm, jd int) (time.Time, error) {
	if jm < 1 || jm > 12 {
		return time.Time{}, fmt.Errorf("شمارهٔ ماه نامعتبر است: %d", jm)
	}
	mlen, err := MonthLength(jy, jm)
	if err != nil {
		return time.Time{}, err
	}
	if jd < 1 || jd > mlen {
		return time.Time{}, fmt.Errorf("روز %d برای ماه %d/%02d نامعتبر است", jd, jy, jm)
	}
	jdn, err := j2d(jy, jm, jd)
	if err != nil {
		return time.Time{}, err
	}
	gy, gm, gd := d2g(jdn)
	return time.Date(gy, time.Month(gm), gd, 0, 0, 0, 0, time.UTC), nil
}

// FromGregorian تبدیل تاریخ میلادی به شمسی.
func FromGregorian(t time.Time) (Date, error) {
	return d2j(g2d(t.Year(), int(t.Month()), t.Day()))
}

// Format تاریخ شمسی به قالب استاندارد YYYY/MM/DD از یک زمان میلادی.
//
// چرا لازم است: رخدادهای سامانه (ثبت، ویرایش، اقدام گردش‌کار) با زمان میلادی ذخیره
// می‌شوند، ولی هر تاریخی که به کاربر نشان داده می‌شود باید شمسی و هم‌قالب با تاریخ‌های
// داده (مثل تاریخ اجرای حکم) باشد تا مقایسه و چیدن زمانی سوابق ساده بماند.
// زمان نامعتبر رشتهٔ خالی می‌دهد (هیچ تاریخ ساختگی ساخته نمی‌شود).
func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d, err := FromGregorian(t)
	if err != nil {
		return ""
	}
	return d.String()
}

// FormatStamp همان Format برای رشتهٔ زمان RFC3339 (شکل ذخیره‌شدهٔ رخدادها در API).
func FormatStamp(stamp string) string {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(stamp))
	if err != nil {
		return ""
	}
	return Format(t.In(time.Local))
}

// Parse تجزیهٔ تاریخ شمسی به قالب YYYY/MM/DD (ارقام فارسی هم پذیرفته می‌شود).
func Parse(s string) (Date, error) {
	s = NormalizeDigits(strings.TrimSpace(s))
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == '-' })
	if len(parts) != 3 {
		return Date{}, fmt.Errorf("قالب تاریخ باید YYYY/MM/DD باشد، دریافت‌شده: %s", s)
	}
	var nums [3]int
	for i, p := range parts {
		n := 0
		if p == "" {
			return Date{}, fmt.Errorf("قالب تاریخ نامعتبر است: %s", s)
		}
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return Date{}, fmt.Errorf("قالب تاریخ نامعتبر است: %s", s)
			}
			n = n*10 + int(ch-'0')
		}
		nums[i] = n
	}
	d := Date{Year: nums[0], Month: nums[1], Day: nums[2]}
	if _, err := ToGregorian(d.Year, d.Month, d.Day); err != nil {
		return Date{}, err
	}
	return d, nil
}

// String نمایش استاندارد YYYY/MM/DD.
func (d Date) String() string {
	return fmt.Sprintf("%04d/%02d/%02d", d.Year, d.Month, d.Day)
}

// Persian نمایش با ارقام فارسی.
func (d Date) Persian() string {
	return ToPersianDigits(d.String())
}

// WeekdayIndex روز هفته شمسی: ۰=شنبه ... ۶=جمعه.
func (d Date) WeekdayIndex() (int, error) {
	g, err := ToGregorian(d.Year, d.Month, d.Day)
	if err != nil {
		return 0, err
	}
	// time.Saturday = 6 → شنبه باید صفر شود.
	return int(g.Weekday()+1) % 7, nil
}

var weekdayFa = []string{"شنبه", "یکشنبه", "دوشنبه", "سه‌شنبه", "چهارشنبه", "پنجشنبه", "جمعه"}

// WeekdayFa نام فارسی روز هفته.
func (d Date) WeekdayFa() (string, error) {
	i, err := d.WeekdayIndex()
	if err != nil {
		return "", err
	}
	return weekdayFa[i], nil
}

// MonthNamesFa نام ماه‌های شمسی.
var MonthNamesFa = []string{"فروردین", "اردیبهشت", "خرداد", "تیر", "مرداد", "شهریور", "مهر", "آبان", "آذر", "دی", "بهمن", "اسفند"}

// MonthLabel برچسب فارسی ماه مانند «مهر ۱۴۰۴».
func (d Date) MonthLabel() string {
	if d.Month < 1 || d.Month > 12 {
		return ""
	}
	return MonthNamesFa[d.Month-1] + " " + ToPersianDigits(fmt.Sprint(d.Year))
}

// AddDays افزودن/کاستن روز از تاریخ شمسی.
func (d Date) AddDays(n int) (Date, error) {
	g, err := ToGregorian(d.Year, d.Month, d.Day)
	if err != nil {
		return Date{}, err
	}
	return FromGregorian(g.AddDate(0, 0, n))
}

// DaysBetween تعداد روز بین دو تاریخ شمسی (b - a).
func DaysBetween(a, b Date) (int, error) {
	ga, err := ToGregorian(a.Year, a.Month, a.Day)
	if err != nil {
		return 0, err
	}
	gb, err := ToGregorian(b.Year, b.Month, b.Day)
	if err != nil {
		return 0, err
	}
	return int(gb.Sub(ga).Hours() / 24), nil
}

// EachDay پیمایش روزهای یک بازه شامل هر دو سر.
func EachDay(from, to Date) ([]Date, error) {
	n, err := DaysBetween(from, to)
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, fmt.Errorf("بازهٔ تاریخ نامعتبر است: %s تا %s", from, to)
	}
	out := make([]Date, 0, n+1)
	cur := from
	for i := 0; i <= n; i++ {
		out = append(out, cur)
		next, err := cur.AddDays(1)
		if err != nil {
			return nil, err
		}
		cur = next
	}
	return out, nil
}

// WorkCalendar تقویم کاری سازمان: تعطیلات هفتگی و تعطیلات رسمی.
type WorkCalendar struct {
	// Weekend ایندکس روزهای تعطیل هفتگی (۰=شنبه).
	Weekend []int
	// Holidays تعطیلات رسمی به قالب YYYY/MM/DD.
	Holidays map[string]string
}

// DefaultWorkCalendar تقویم پیش‌فرض: جمعه تعطیل است.
func DefaultWorkCalendar() WorkCalendar {
	return WorkCalendar{Weekend: []int{6}, Holidays: map[string]string{}}
}

// IsWeekend آیا روز تعطیل هفتگی است.
func (c WorkCalendar) IsWeekend(d Date) (bool, error) {
	i, err := d.WeekdayIndex()
	if err != nil {
		return false, err
	}
	for _, w := range c.Weekend {
		if w == i {
			return true, nil
		}
	}
	return false, nil
}

// IsHoliday آیا روز تعطیل رسمی است.
func (c WorkCalendar) IsHoliday(d Date) bool {
	_, ok := c.Holidays[d.String()]
	return ok
}

// IsWorkingDay روز کاری = نه تعطیل هفتگی و نه تعطیل رسمی.
func (c WorkCalendar) IsWorkingDay(d Date) (bool, error) {
	if c.IsHoliday(d) {
		return false, nil
	}
	w, err := c.IsWeekend(d)
	if err != nil {
		return false, err
	}
	return !w, nil
}

// SandwichDays روزهای کاری «ساندویچی»: از نخستین تا واپسین روز کاری بازه؛
// تعطیل‌های میانی جزء مرخصی حساب می‌شوند ولی تعطیل‌های دو سر نه.
func (c WorkCalendar) SandwichDays(from, to Date) (int, error) {
	days, err := EachDay(from, to)
	if err != nil {
		return 0, err
	}
	first, last := -1, -1
	for i, d := range days {
		ok, err := c.IsWorkingDay(d)
		if err != nil {
			return 0, err
		}
		if ok {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return 0, nil
	}
	return last - first + 1, nil
}

// WorkingDays تعداد روزهای کاری در یک بازه.
func (c WorkCalendar) WorkingDays(from, to Date) (int, error) {
	days, err := EachDay(from, to)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, d := range days {
		ok, err := c.IsWorkingDay(d)
		if err != nil {
			return 0, err
		}
		if ok {
			n++
		}
	}
	return n, nil
}
