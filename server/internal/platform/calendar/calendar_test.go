// آزمون زیرساخت «calendar» در پلتفرم سرور — «calendar_test».

package calendar

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// مسیر fixture طلایی (تولیدشده توسط هستهٔ Rust).
func fixturePath() string {
	return filepath.Join("..", "..", "..", "..", "core", "testdata", "conformance", "calendar.json")
}

type calendarFixture struct {
	Years []struct {
		Year                int    `json:"year"`
		Leap                bool   `json:"leap"`
		EsfandLength        int    `json:"esfandLength"`
		Farvardin1Gregorian string `json:"farvardin1Gregorian"`
	} `json:"years"`
	Conversions []struct {
		Jalali       string `json:"jalali"`
		Gregorian    string `json:"gregorian"`
		JDN          int64  `json:"jdn"`
		WeekdayIndex int    `json:"weekdayIndex"`
		WeekdayFa    string `json:"weekdayFa"`
		MonthLabelFa string `json:"monthLabelFa"`
	} `json:"conversions"`
	MonthLengths []struct {
		Year  int `json:"year"`
		Month int `json:"month"`
		Days  int `json:"days"`
	} `json:"monthLengths"`
	WorkWindows []struct {
		Name         string            `json:"name"`
		From         string            `json:"from"`
		To           string            `json:"to"`
		Weekend      []int             `json:"weekend"`
		Holidays     map[string]string `json:"holidays"`
		WorkingDays  int               `json:"workingDays"`
		SandwichDays int               `json:"sandwichDays"`
	} `json:"workWindows"`
}

func loadFixture(t *testing.T) calendarFixture {
	t.Helper()
	raw, err := os.ReadFile(fixturePath())
	if err != nil {
		t.Fatalf("خواندن fixture تقویم ناموفق: %v", err)
	}
	var f calendarFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("fixture تقویم قابل تجزیه نیست: %v", err)
	}
	return f
}

// TestCalendarParityWithGoldenFixtures دروازهٔ تطبیق تقویم با هستهٔ Rust است.
func TestCalendarParityWithGoldenFixtures(t *testing.T) {
	f := loadFixture(t)

	for _, y := range f.Years {
		first, err := ToGregorian(y.Year, 1, 1)
		if err != nil {
			t.Fatalf("نوروز %d: %v", y.Year, err)
		}
		got := first.Format("2006-01-02")
		if got != y.Farvardin1Gregorian {
			t.Errorf("نوروز %d: سرور=%s هسته=%s", y.Year, got, y.Farvardin1Gregorian)
		}
		leap, err := IsLeapYear(y.Year)
		if err != nil {
			t.Fatalf("کبیسهٔ %d: %v", y.Year, err)
		}
		if leap != y.Leap {
			t.Errorf("کبیسهٔ %d: سرور=%v هسته=%v", y.Year, leap, y.Leap)
		}
		esfand, err := MonthLength(y.Year, 12)
		if err != nil {
			t.Fatalf("اسفند %d: %v", y.Year, err)
		}
		if esfand != y.EsfandLength {
			t.Errorf("طول اسفند %d: سرور=%d هسته=%d", y.Year, esfand, y.EsfandLength)
		}
	}

	for _, c := range f.Conversions {
		d, err := Parse(c.Jalali)
		if err != nil {
			t.Fatalf("تجزیهٔ %s ناموفق: %v", c.Jalali, err)
		}
		if d.String() != c.Jalali {
			t.Errorf("نمایش تاریخ: %s != %s", d.String(), c.Jalali)
		}
		g, err := ToGregorian(d.Year, d.Month, d.Day)
		if err != nil {
			t.Fatalf("تبدیل %s ناموفق: %v", c.Jalali, err)
		}
		if got := g.Format("2006-01-02"); got != c.Gregorian {
			t.Errorf("تبدیل %s: سرور=%s هسته=%s", c.Jalali, got, c.Gregorian)
		}
		back, err := FromGregorian(g)
		if err != nil || back != d {
			t.Errorf("بازگشت %s ناموفق بود: %v", c.Jalali, err)
		}
		idx, err := d.WeekdayIndex()
		if err != nil {
			t.Fatalf("روز هفتهٔ %s: %v", c.Jalali, err)
		}
		if idx != c.WeekdayIndex {
			t.Errorf("اندیس روز هفتهٔ %s: سرور=%d هسته=%d", c.Jalali, idx, c.WeekdayIndex)
		}
		fa, err := d.WeekdayFa()
		if err != nil || fa != c.WeekdayFa {
			t.Errorf("نام روز هفتهٔ %s: سرور=%q هسته=%q", c.Jalali, fa, c.WeekdayFa)
		}
		if label := d.MonthLabel(); label != c.MonthLabelFa {
			t.Errorf("برچسب ماه %s: سرور=%q هسته=%q", c.Jalali, label, c.MonthLabelFa)
		}
	}

	for _, m := range f.MonthLengths {
		got, err := MonthLength(m.Year, m.Month)
		if err != nil {
			t.Fatalf("طول ماه %d/%02d: %v", m.Year, m.Month, err)
		}
		if got != m.Days {
			t.Errorf("طول ماه %d/%02d: سرور=%d هسته=%d", m.Year, m.Month, got, m.Days)
		}
	}

	for _, w := range f.WorkWindows {
		from, err := Parse(w.From)
		if err != nil {
			t.Fatalf("تاریخ %s: %v", w.From, err)
		}
		to, err := Parse(w.To)
		if err != nil {
			t.Fatalf("تاریخ %s: %v", w.To, err)
		}
		cal := WorkCalendar{Weekend: w.Weekend, Holidays: w.Holidays}
		working, err := cal.WorkingDays(from, to)
		if err != nil {
			t.Fatalf("شمارش روز کاری %s: %v", w.Name, err)
		}
		if working != w.WorkingDays {
			t.Errorf("«%s»: روز کاری سرور=%d هسته=%d", w.Name, working, w.WorkingDays)
		}
		sandwich, err := cal.SandwichDays(from, to)
		if err != nil {
			t.Fatalf("شمارش ساندویچ %s: %v", w.Name, err)
		}
		if sandwich != w.SandwichDays {
			t.Errorf("«%s»: روز ساندویچی سرور=%d هسته=%d", w.Name, sandwich, w.SandwichDays)
		}
	}
}

func TestWeekdayIndexIsPersianWeek(t *testing.T) {
	// شنبه = ۰ و جمعه = ۶
	nowruz1404, _ := Parse("1404/01/01") // ۲۰۲۵-۰۳-۲۱ → جمعه
	friday, _ := Parse("1404/07/10")     // ۲۰۲۵-۱۰-۰۲ پنجشنبه
	saturday, _ := Parse("1404/07/05")   // ۲۰۲۵-۰۹-۲۷ شنبه

	if idx, _ := nowruz1404.WeekdayIndex(); idx != 6 {
		t.Errorf("نوروز ۱۴۰۴ باید جمعه (۶) باشد، %d بود", idx)
	}
	if name, _ := friday.WeekdayFa(); name != "پنجشنبه" {
		t.Errorf("روز پنجشنبه انتظار می‌رفت، %q دریافت شد", name)
	}
	if idx, _ := saturday.WeekdayIndex(); idx != 0 {
		t.Errorf("شنبه باید اندیس ۰ باشد، %d بود", idx)
	}
}

func TestPersianDigitsAndNormalisation(t *testing.T) {
	if got := NormalizeDigits("۱۴۰۴/۰۷/۰۱"); got != "1404/07/01" {
		t.Errorf("تبدیل ارقام فارسی ناموفق: %s", got)
	}
	d, err := Parse("۱۴۰۴/۰۷/۰۱")
	if err != nil {
		t.Fatalf("تجزیهٔ تاریخ با ارقام فارسی ناموفق: %v", err)
	}
	if d.String() != "1404/07/01" {
		t.Errorf("تاریخ تجزیه‌شده نادرست است: %s", d.String())
	}
	if got := ToPersianDigits("1404"); got != "۱۴۰۴" {
		t.Errorf("تبدیل ارقام به فارسی ناموفق: %s", got)
	}
}

func TestInvalidInputsAreRejectedWithPersianMessages(t *testing.T) {
	cases := []struct {
		in  string
		why string
	}{
		{"1404/13/01", "ماه نامعتبر"},
		{"1404/12/30", "۱۴۰۴ کبیسه نیست"},
		{"1404/07/31", "مهر ۳۰ روز است"},
		{"not-a-date", "قالب نامعتبر"},
	}
	for _, c := range cases {
		if _, err := Parse(c.in); err == nil {
			t.Errorf("%s (%s) باید رد شود", c.in, c.why)
		}
	}
	if _, err := Parse("1403/12/30"); err != nil {
		t.Errorf("۱۴۰۳ کبیسه است و ۳۰ اسفند معتبر باید باشد: %v", err)
	}
}

func TestAddDaysCrossesYearBoundary(t *testing.T) {
	last, _ := Parse("1404/12/29")
	next, err := last.AddDays(1)
	if err != nil {
		t.Fatalf("افزودن روز ناموفق: %v", err)
	}
	if next.String() != "1405/01/01" {
		t.Errorf("روز پس از پایان سال باید ۱۴۰۵/۰۱/۰۱ باشد، %s بود", next.String())
	}
	if n, _ := DaysBetween(last, next); n != 1 {
		t.Errorf("فاصلهٔ روزها باید ۱ باشد، %d بود", n)
	}
}
