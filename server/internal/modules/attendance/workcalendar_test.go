// آزمون «تقویم کارکرد»: وضعیت روزها، بازه‌های مرخصی/مأموریت و جمع‌های ماه.
//
// چرا این آزمون‌ها: تقویم کارکرد مبنای کنترل پیش از اجرای حقوق است (کدام روز مرخصی بود،
// کدام روز مأموریت، کدام روز بدون کارکرد مانده). اگر بازهٔ چندروزه درست پخش نشود یا رکورد
// در گردش در جمع شمرده شود، عدد کارکرد ماه غلط می‌شود؛ پس هر دو با دادهٔ ساختگی سنجیده
// می‌شود.

package attendance

import (
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

// dayMonth یک ماه کامل با روزهایش می‌سازد (بدون تعطیل رسمی).
func dayMonth(year, month int) (map[string]*calendarDay, []string) {
	length, _ := calendar.MonthLength(year, month)
	days := map[string]*calendarDay{}
	order := []string{}
	for day := 1; day <= length; day++ {
		d := calendar.Date{Year: year, Month: month, Day: day}
		weekday, _ := d.WeekdayFa()
		index, _ := d.WeekdayIndex()
		days[d.String()] = &calendarDay{
			Date: d.String(), Day: day, WeekdayIndex: index, WeekdayFa: weekday, Working: true,
		}
		order = append(order, d.String())
	}
	return days, order
}

// TestRecordDatesSpreadsRanges بازهٔ چندروزه را روزبه‌روز می‌شکند و از ماه بیرون نمی‌زند.
func TestRecordDatesSpreadsRanges(t *testing.T) {
	leave := contracts.Record{Fields: map[string]string{"fromDate": "1405/07/29", "toDate": "1405/08/03"}}
	dates := recordDates("leaveRequests", leave, "1405/08")
	want := []string{"1405/08/01", "1405/08/02", "1405/08/03"}
	if strings.Join(dates, ",") != strings.Join(want, ",") {
		t.Errorf("روزهای مرخصی = %v، انتظار %v", dates, want)
	}

	single := contracts.Record{Fields: map[string]string{"date": "1405/08/12"}}
	if got := recordDates("attendanceRecords", single, "1405/08"); len(got) != 1 || got[0] != "1405/08/12" {
		t.Errorf("روز تردد = %v", got)
	}
	// همان رکورد در ماه دیگر نباید بیاید.
	if got := recordDates("attendanceRecords", single, "1405/07"); got != nil {
		t.Errorf("رکورد ماه دیگر روی تقویم آمد: %v", got)
	}
	// رکورد بی‌تاریخ معتبر روی تقویم نمی‌آید.
	if got := recordDates("overtime", contracts.Record{Fields: map[string]string{"date": ""}}, "1405/08"); got != nil {
		t.Errorf("رکورد بی‌تاریخ روی تقویم آمد: %v", got)
	}
}

// TestSettleDayStatus وضعیت جمع‌بندی هر روز را می‌سنجد.
func TestSettleDayStatus(t *testing.T) {
	cases := []struct {
		name     string
		day      calendarDay
		wantKey  string
		missing  bool
		scenario string
	}{
		{"holiday", calendarDay{Holiday: true, Weekend: true}, "holiday", false, "تعطیل رسمی بر هفتگی مقدم است"},
		{"weekend", calendarDay{Weekend: true}, "weekend", false, "تعطیل هفتگی"},
		{"leave", calendarDay{LeaveCount: 2}, "leave", false, "روز کاری با مرخصی"},
		{"mission", calendarDay{MissionCount: 1}, "mission", false, "روز کاری با مأموریت"},
		{"mixed", calendarDay{LeaveCount: 1, MissionCount: 1}, "mixed", false, "هم مرخصی هم مأموریت"},
		{"worked", calendarDay{WorkedHours: 8}, "worked", false, "کارکرد ثبت‌شده"},
		{"missing", calendarDay{}, "missing", true, "روز کاری بدون کارکرد"},
	}
	for _, c := range cases {
		day := c.day
		settleDayStatus(&day)
		if day.StatusKey != c.wantKey {
			t.Errorf("%s: وضعیت = %s، انتظار %s", c.scenario, day.StatusKey, c.wantKey)
		}
		if day.MissingWork != c.missing {
			t.Errorf("%s: missingWork = %v، انتظار %v", c.scenario, day.MissingWork, c.missing)
		}
	}
}

// TestApplyCalendarRecordCountsAndPending رکورد مؤثر و رکورد در گردش را جدا می‌سنجد.
func TestApplyCalendarRecordCountsAndPending(t *testing.T) {
	days, order := dayMonth(1405, 8)
	totals := workCalendarTotals{}

	approved := contracts.Record{
		ID: "LV-0001", EmployeeID: "EMP-004", EmployeeName: "علی مرادی", Status: "approved", StatusFa: "تأیید نهایی منابع انسانی",
		Fields: map[string]string{"leaveType": "استحقاقی", "fromDate": "1405/08/10", "toDate": "1405/08/11", "days": "2"},
	}
	applyCalendarRecord("leaveRequests", approved, days, order, "1405/08", &totals)
	if totals.LeaveDays != 2 {
		t.Errorf("جمع روزهای مرخصی = %v، انتظار ۲", totals.LeaveDays)
	}
	if days["1405/08/10"].LeaveCount != 1 || days["1405/08/11"].LeaveCount != 1 {
		t.Errorf("روزهای مرخصی روی تقویم نشانه‌گذاری نشد")
	}
	if len(days["1405/08/10"].Chips) != 1 || days["1405/08/10"].Chips[0].Kind != "leave" {
		t.Fatalf("برچسب مرخصی روی روز ننشست: %#v", days["1405/08/10"].Chips)
	}
	if !strings.Contains(days["1405/08/10"].Chips[0].TitleFa, "علی مرادی") {
		t.Errorf("نام کارمند در برچسب مرخصی نیست: %q", days["1405/08/10"].Chips[0].TitleFa)
	}
	if !strings.Contains(days["1405/08/10"].Chips[0].DetailFa, "استحقاقی") {
		t.Errorf("نوع مرخصی در شرح نیست: %q", days["1405/08/10"].Chips[0].DetailFa)
	}

	// رکورد در گردش: برچسب می‌آید ولی در جمع کارکرد شمرده نمی‌شود.
	pending := contracts.Record{
		ID: "LV-0002", EmployeeID: "EMP-008", EmployeeName: "مریم نوری", Status: "pending",
		Fields: map[string]string{"leaveType": "بدون حقوق", "fromDate": "1405/08/14", "toDate": "1405/08/14", "days": "1"},
	}
	applyCalendarRecord("leaveRequests", pending, days, order, "1405/08", &totals)
	if totals.LeaveDays != 2 {
		t.Errorf("رکورد در گردش در جمع شمرده شد: %v", totals.LeaveDays)
	}
	if totals.PendingRecords != 1 {
		t.Errorf("شمار رکوردهای در گردش = %d، انتظار ۱", totals.PendingRecords)
	}
	if chip := days["1405/08/14"].Chips[0]; !chip.Pending || !strings.Contains(chip.DetailFa, "کسری") {
		t.Errorf("برچسب مرخصی بدون حقوق باید در گردش و کسری باشد: %#v", chip)
	}

	// رکورد ردشده اصلاً روی تقویم نمی‌آید.
	rejected := contracts.Record{
		Status: "rejected", EmployeeName: "رضا کاظمی",
		Fields: map[string]string{"fromDate": "1405/08/20", "toDate": "1405/08/20", "days": "1", "leaveType": "استحقاقی"},
	}
	applyCalendarRecord("leaveRequests", rejected, days, order, "1405/08", &totals)
	if len(days["1405/08/20"].Chips) != 0 {
		t.Errorf("مرخصی ردشده روی تقویم آمد: %#v", days["1405/08/20"].Chips)
	}
	if totals.PendingRecords != 1 {
		t.Errorf("رکورد ردشده باید نادیده گرفته شود: %d", totals.PendingRecords)
	}
}

// TestApplyCalendarRecordAttendanceAndOvertime ساعت کارکرد و اضافه‌کاری را می‌سنجد.
func TestApplyCalendarRecordAttendanceAndOvertime(t *testing.T) {
	days, order := dayMonth(1405, 8)
	totals := workCalendarTotals{}

	attendance := contracts.Record{
		ID: "ATT-0001", EmployeeID: "EMP-010", EmployeeName: "فاطمه صادقی", Status: "confirmed", StatusFa: "تأییدشده",
		Fields: map[string]string{"date": "1405/08/05", "workedHours": "7.5", "lateMinutes": "12", "nightHours": "1"},
	}
	applyCalendarRecord("attendanceRecords", attendance, days, order, "1405/08", &totals)
	if days["1405/08/05"].WorkedHours != 7.5 || days["1405/08/05"].LateMinutes != 12 {
		t.Errorf("کارکرد روز ثبت نشد: %#v", days["1405/08/05"])
	}
	if totals.WorkedHours != 7.5 || totals.LateMinutes != 12 {
		t.Errorf("جمع کارکرد ماه درست نیست: %#v", totals)
	}
	detail := days["1405/08/05"].Chips[0].DetailFa
	for _, want := range []string{"7.5 ساعت کارکرد", "12 دقیقه تأخیر", "1 ساعت شب‌کاری"} {
		if !strings.Contains(detail, want) {
			t.Errorf("شرح کارکرد «%s» را ندارد: %q", want, detail)
		}
	}

	overtime := contracts.Record{
		ID: "OT-0001", EmployeeID: "EMP-007", EmployeeName: "رضا کاظمی", Status: "approved",
		Fields: map[string]string{"date": "1405/08/05", "hours": "3"},
	}
	applyCalendarRecord("overtime", overtime, days, order, "1405/08", &totals)
	if days["1405/08/05"].OvertimeHours != 3 || totals.OvertimeHours != 3 {
		t.Errorf("اضافه‌کاری ثبت نشد: %#v", days["1405/08/05"])
	}
	settleDayStatus(days["1405/08/05"])
	if days["1405/08/05"].StatusKey != "worked" {
		t.Errorf("وضعیت روز دارای کارکرد = %s", days["1405/08/05"].StatusKey)
	}
}

// TestApplyCalendarRecordMission گزینه‌های مأموریت و مقصد را می‌سنجد.
func TestApplyCalendarRecordMission(t *testing.T) {
	days, order := dayMonth(1405, 8)
	totals := workCalendarTotals{}
	mission := contracts.Record{
		ID: "MSN-0001", EmployeeID: "EMP-002", EmployeeName: "آرش رضایی", Status: "approved",
		Fields: map[string]string{"destination": "تهران", "purpose": "جلسهٔ فنی", "fromDate": "1405/08/18", "toDate": "1405/08/19"},
	}
	applyCalendarRecord("missions", mission, days, order, "1405/08", &totals)
	if totals.MissionDays != 2 {
		t.Errorf("جمع روزهای مأموریت = %v، انتظار ۲", totals.MissionDays)
	}
	chip := days["1405/08/18"].Chips[0]
	if chip.Kind != "mission" || !strings.Contains(chip.DetailFa, "تهران") || !strings.Contains(chip.DetailFa, "جلسهٔ فنی") {
		t.Errorf("برچسب مأموریت درست نیست: %#v", chip)
	}
	settleDayStatus(days["1405/08/18"])
	if days["1405/08/18"].StatusKey != "mission" {
		t.Errorf("وضعیت روز مأموریت = %s", days["1405/08/18"].StatusKey)
	}
}

// TestParseCalendarMonth قالب ماه ورودی API را می‌سنجد.
func TestParseCalendarMonth(t *testing.T) {
	if d, ok := parseCalendarMonth("1405/08"); !ok || d.Year != 1405 || d.Month != 8 || d.Day != 1 {
		t.Errorf("ماه معتبر تجزیه نشد: %#v ok=%v", d, ok)
	}
	if d, ok := parseCalendarMonth("۱۴۰۵/۰۸/۱۳"); !ok || d.Month != 8 {
		t.Errorf("ماه با ارقام فارسی و روز تجزیه نشد: %#v ok=%v", d, ok)
	}
	for _, bad := range []string{"", "1405", "1405/13", "abc/01"} {
		if _, ok := parseCalendarMonth(bad); ok {
			t.Errorf("ورودی نامعتبر «%s» پذیرفته شد", bad)
		}
	}
}

// TestCalendarLegendCoversKinds راهنمای تقویم همهٔ برچسب‌های ممکن را توضیح می‌دهد.
func TestCalendarLegendCoversKinds(t *testing.T) {
	legend := map[string]bool{}
	for _, item := range calendarLegend() {
		if strings.TrimSpace(item.TitleFa) == "" {
			t.Errorf("برچسب %s عنوان ندارد", item.Kind)
		}
		legend[item.Kind] = true
	}
	for _, kind := range []string{"holiday", "weekend", "occasion", "leave", "mission", "overtime", "attendance", "missing"} {
		if !legend[kind] {
			t.Errorf("برچسب %s در راهنمای تقویم نیست", kind)
		}
	}
}
