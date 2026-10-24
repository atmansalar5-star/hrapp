// پیاده‌سازی ماژول «settings» — «calendar».

package settings

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// registerCalendarEndpoint مسیر «تقویم کاری» را ثبت می‌کند.
//
// این مسیر قاعدهٔ تازه‌ای نمی‌سازد: تقویم کاری سازمان (تعطیل هفتگی و تعطیلات رسمی) از
// خودِ استقرار خوانده می‌شود و پارامترهای حضور/مرخصی از پروفایل شرکت فعال می‌آید. محاسبهٔ
// روزِ کاری هم در بستهٔ `calendar` انجام می‌گیرد تا تقویم یک منبع حقیقت داشته باشد.
func registerCalendarEndpoint(mux *http.ServeMux, deps modulekit.Deps) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("settings", "read", fn) }

	mux.HandleFunc("GET /api/v1/settings/work-calendar", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, workCalendarView(r.URL.Query().Get("month"), deps))
	}))
}

// calendarDay یک روز از تقویم کاری در پاسخ API.
type calendarDay struct {
	Date string `json:"date"`
	Day  int    `json:"day"`
	// WeekdayIndex ستون هفته (۰=شنبه … ۶=جمعه) برای چیدن شبکهٔ تقویم در رابط.
	WeekdayIndex int    `json:"weekdayIndex"`
	WeekdayFa    string `json:"weekdayFa"`
	Weekend      bool   `json:"weekend"`
	Holiday      bool   `json:"holiday"`
	TitleFa      string `json:"titleFa,omitempty"`
	Working      bool   `json:"working"`
}

// workCalendarView نمای یک ماه شمسی: روزها با پرچم تعطیل هفتگی/رسمی، جمع روزهای کاری و
// پارامترهای حضور و مرخصی استقرار فعال.
func workCalendarView(month string, deps modulekit.Deps) map[string]any {
	today, err := calendar.FromGregorian(deps.Clock())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	target := today
	if parsed, ok := parseMonth(month); ok {
		target = parsed
	}

	length, err := calendar.MonthLength(target.Year, target.Month)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	days := make([]calendarDay, 0, length)
	var working, weekend, holidays int
	for day := 1; day <= length; day++ {
		d := calendar.Date{Year: target.Year, Month: target.Month, Day: day}
		weekday, _ := d.WeekdayFa()
		weekdayIndex, _ := d.WeekdayIndex()
		isWeekend, _ := deps.Calendar.IsWeekend(d)
		isHoliday := deps.Calendar.IsHoliday(d)
		isWorking, _ := deps.Calendar.IsWorkingDay(d)
		cd := calendarDay{
			Date: d.String(), Day: day, WeekdayIndex: weekdayIndex, WeekdayFa: weekday,
			Weekend: isWeekend, Holiday: isHoliday, Working: isWorking,
		}
		if isHoliday {
			cd.TitleFa = deps.Calendar.Holidays[d.String()]
			holidays++
		}
		if isWeekend {
			weekend++
		}
		if isWorking {
			working++
		}
		days = append(days, cd)
	}

	todayWeekday, _ := today.WeekdayFa()
	params := []map[string]string{}
	for _, p := range deps.Org.Params {
		switch p.Key {
		case "leave.annualDays", "leave.sickDays", "attendance.graceMinutes", "payroll.monthlyHours", "payroll.payDay", "mission.perDiem":
			params = append(params, map[string]string{
				"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa,
			})
		}
	}

	return map[string]any{
		"today": map[string]any{
			"date": today.String(), "weekdayFa": todayWeekday, "monthLabel": today.MonthLabel(),
			"weekend": func() bool { w, _ := deps.Calendar.IsWeekend(today); return w }(),
			"holiday": deps.Calendar.IsHoliday(today),
		},
		"month": map[string]any{
			"key":   strings.Join([]string{strconv.Itoa(target.Year), fmt.Sprintf("%02d", target.Month)}, "/"),
			"label": target.MonthLabel(), "daysCount": length,
			"workingDays": working, "weekendDays": weekend, "holidays": holidays,
			"days": days,
		},
		"company":    deps.Org.TitleFa,
		"params":     params,
		"noteFa":     "تعطیل هفتگی و تعطیلات رسمی از تقویم کاری استقرار همین شرکت خوانده می‌شود؛ محاسبهٔ روز کاری و روزهای ساندویچی مرخصی با همین تقویم انجام می‌گیرد.",
		"holidaysFa": len(deps.Calendar.Holidays),
	}
}

// parseMonth رشتهٔ «YYYY/MM» را به تاریخ روز نخست همان ماه تبدیل می‌کند.
func parseMonth(raw string) (calendar.Date, bool) {
	raw = strings.TrimSpace(calendar.NormalizeDigits(raw))
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return calendar.Date{}, false
	}
	year, errY := strconv.Atoi(parts[0])
	month, errM := strconv.Atoi(parts[1])
	if errY != nil || errM != nil || year <= 0 || month < 1 || month > 12 {
		return calendar.Date{}, false
	}
	if _, err := calendar.MonthLength(year, month); err != nil {
		return calendar.Date{}, false
	}
	return calendar.Date{Year: year, Month: month, Day: 1}, true
}
