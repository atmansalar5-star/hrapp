// آزمون‌های خالص انبار PostgreSQL کارکنان.
//
// چرا این آزمون‌ها بی‌پایگاه‌داده‌اند: جریان کامل انبار (ذخیره، خواندن، حذف نرم) با
// آزمون دودی Postgres در بستهٔ bootstrap سنجیده می‌شود که به پایگاه‌دادهٔ واقعی نیاز
// دارد. آن‌چه اینجا جدا سنجیده می‌شود منطق خالصِ همین فایل است: نگاشت امن مرتب‌سازی
// (ستون سفید تا پارامتر sort به تزریق SQL تبدیل نشود)، تبدیل رفت‌وبرگشت تاریخ شمسی و
// پاک‌سازی فهرست فیلترها.

package infra

import (
	"strings"
	"testing"
	"time"

	"hrms/internal/platform/calendar"
)

// TestOrderByUsesWhitelist بررسی می‌کند تنها ستون‌های شناخته‌شده به SQL می‌روند.
func TestOrderByUsesWhitelist(t *testing.T) {
	cases := map[string]string{
		"":                       "e.id, e.id",
		"fullName":               "e.full_name, e.id",
		"fullName:desc":          "e.full_name desc, e.id desc",
		"hireDate:desc":          "e.hire_date desc, e.id desc",
		"baseSalary":             "e.base_salary, e.id",
		"departmentName":         "coalesce(d.name, ''), e.id",
		"status:desc":            "e.status desc, e.id desc",
		"id;drop table employee": "e.id, e.id",
		"1=1 --":                 "e.id, e.id",
	}
	for spec, want := range cases {
		if got := orderBy(spec); got != want {
			t.Errorf("orderBy(%q) = %q، انتظار %q", spec, got, want)
		}
	}
	// هیچ ورودی کاربری نباید عیناً در متن SQL بیفتد.
	for spec := range cases {
		if strings.Contains(orderBy(spec), "drop") || strings.Contains(orderBy(spec), "--") {
			t.Errorf("orderBy(%q) مقدار کاربری را وارد SQL کرد", spec)
		}
	}
}

// TestJalaliDateRoundTrip تبدیل تاریخ شمسی به میلادی و برگشت آن را می‌سنجد؛ ستون
// جدول از نوع date است و داده و API شمسی می‌مانند، پس این تبدیل نباید روز را جابه‌جا کند.
func TestJalaliDateRoundTrip(t *testing.T) {
	for _, value := range []string{"1398/04/12", "1400/01/01", "1403/12/30", "1404/09/01"} {
		gregorian, err := jalaliDate(value)
		if err != nil {
			t.Fatalf("jalaliDate(%q) خطا داد: %v", value, err)
		}
		if got := jalaliString(gregorian); got != value {
			t.Errorf("رفت‌وبرگشت %q → %v → %q", value, gregorian, got)
		}
	}
	if _, err := jalaliDate("1404/13/01"); err == nil {
		t.Error("ماه ۱۳ باید رد شود")
	}
	if got := jalaliString(time.Time{}); got != "" {
		t.Errorf("تاریخ صفر باید رشتهٔ خالی بدهد، شد %q", got)
	}
}

// TestJalaliStringMatchesCalendarExit بررسی می‌کند خروجی انبار با تبدیل مشترک تقویم یکی است.
func TestJalaliStringMatchesCalendarExit(t *testing.T) {
	gregorian, err := jalaliDate("1404/07/15")
	if err != nil {
		t.Fatal(err)
	}
	date, err := calendar.FromGregorian(gregorian)
	if err != nil {
		t.Fatal(err)
	}
	want := calendar.Format(gregorian)
	if got := jalaliString(gregorian); got != want {
		t.Errorf("jalaliString = %q، calendar.Format = %q (تاریخ %+v)", got, want, date)
	}
}

// TestTrimmedValuesDropEmpty مقدارهای خالی فیلتر چندانتخابی را حذف می‌کند.
func TestTrimmedValuesDropEmpty(t *testing.T) {
	got := trimmedValues([]string{" active ", "", "  ", "leave"})
	if len(got) != 2 || got[0] != "active" || got[1] != "leave" {
		t.Errorf("trimmedValues نتیجهٔ نادرست: %#v", got)
	}
	if len(trimmedValues(nil)) != 0 {
		t.Error("فهرست nil باید خالی بماند")
	}
}
