// برنامهٔ نوبت و پوشش شیفت — زیرکار ۱.۲ فاز ۱ نقشهٔ راه.
//
// این فایل «نوبت هر روز» و «پوشش هر شیفت» را از همان چرخهٔ نوبت‌کاری (shifts.go)
// می‌سازد. هیچ محاسبه‌ای در رابط کاربری تکرار نمی‌شود: سرور نوبت هر روز را با
// ResolveShiftDay حل می‌کند و کمبود پوشش را با «حداقل پوشش» الگو می‌سنجد.
//
// قاعده‌ها:
//   - فقط تخصیص مصوب (یا خاتمه‌یافته برای روزهای گذشته) در برنامه می‌آید؛ تخصیص
//     پیش‌نویس و در انتظار تأیید نوبت نمی‌سازد (پرداخت و کارکرد پیش از تأیید نیست).
//   - روز استراحت در برنامه دیده می‌شود ولی «کارکرد موظفی» ساخته نمی‌شود.
//   - پوشش هر شیفت در هر روز شمرده می‌شود و کمبودش با حداقل پوشش الگو سنجیده می‌شود.
package domain

import (
	"sort"
	"strings"

	"hrms/internal/platform/calendar"
)

// PlanInput ورودی حل نوبت یک تخصیص: چرخهٔ الگو، مبنای شروع و جایگاه روز شروع در چرخه.
type PlanInput struct {
	AssignmentID string
	EmployeeID   string
	EmployeeName string
	PatternTitle string
	// Cycle ترتیب چرخهٔ شیفت‌ها (همان چیزی که ParseCycle از فیلد rotation می‌سازد).
	Cycle []string
	// CycleStart تاریخی که چرخه از آن آغاز شده و CycleOffset جایگاه همان روز در چرخه.
	CycleStart  string
	CycleOffset int
	// EndDate پایان تخصیص؛ خالی یعنی بی‌پایان.
	EndDate string
	// Status وضعیت رکورد تخصیص؛ فقط مصوب/خاتمه‌یافته در برنامه می‌آید.
	Status string
}

// PlanDay نوبت یک کارمند در یک روز.
type PlanDay struct {
	Date         string  `json:"date"`
	WeekdayFa    string  `json:"weekdayFa"`
	EmployeeID   string  `json:"employeeId"`
	EmployeeName string  `json:"employeeName,omitempty"`
	AssignmentID string  `json:"assignmentId,omitempty"`
	PatternTitle string  `json:"patternTitle,omitempty"`
	Shift        string  `json:"shift"`
	ShiftTitleFa string  `json:"shiftTitleFa"`
	Start        string  `json:"start,omitempty"`
	End          string  `json:"end,omitempty"`
	Hours        float64 `json:"hours"`
	// Night شیفت شبانه و Rest روز استراحت؛ مبنای فواصل قانونی در کارکرد.
	Night bool `json:"night"`
	Rest  bool `json:"rest"`
	// Holiday روز تعطیل رسمی یا پایان هفته.
	Holiday bool `json:"holiday"`
}

// shiftByKey شیفت را از جدول شیفت‌های این استقرار پیدا می‌کند.
func shiftByKey(key string) (ShiftDef, bool) {
	for _, sh := range knownShifts() {
		if sh.Key == strings.TrimSpace(key) {
			return sh, true
		}
	}
	return ShiftDef{}, false
}

// shiftOrder جایگاه شیفت در چرخهٔ نمایش (صبح، عصر، شب، استراحت).
func shiftOrder(key string) int {
	for i, sh := range knownShifts() {
		if sh.Key == key {
			return i
		}
	}
	return len(knownShifts())
}

// planStatusAllowed آیا وضعیت تخصیص اجازهٔ ساخت نوبت می‌دهد.
func planStatusAllowed(status string) bool {
	switch strings.TrimSpace(status) {
	case "approved", "ended":
		return true
	}
	return false
}

// PlanStatusAllowed آیا وضعیت تخصیص برای حل نوبت معتبر است.
func PlanStatusAllowed(status string) bool { return planStatusAllowed(status) }

// BuildPlan نوبت هر روز یک تخصیص را در بازهٔ [from, to] حل می‌کند.
//
// تاریخ پایان تخصیص بازه را کوتاه می‌کند؛ بازهٔ معکوس خطای قاعده می‌دهد تا برنامهٔ
// خالی بی‌صدا ساخته نشود.
func BuildPlan(in PlanInput, from, to string) ([]PlanDay, error) {
	if !planStatusAllowed(in.Status) {
		return nil, ruleErr("فقط تخصیص نوبت مصوب (یا خاتمه‌یافته) در برنامهٔ نوبت می‌آید؛ وضعیت این تخصیص «" +
			strings.TrimSpace(in.Status) + "» است.")
	}
	cycle, err := ParseCycle(strings.Join(in.Cycle, ","))
	if err != nil {
		return nil, err
	}
	start, err := calendar.Parse(strings.TrimSpace(from))
	if err != nil {
		return nil, ruleErr("تاریخ شروع برنامه معتبر نیست؛ قالب YYYY/MM/DD لازم است.")
	}
	end, err := calendar.Parse(strings.TrimSpace(to))
	if err != nil {
		return nil, ruleErr("تاریخ پایان برنامه معتبر نیست؛ قالب YYYY/MM/DD لازم است.")
	}
	if span, err := calendar.DaysBetween(start, end); err != nil || span < 0 {
		return nil, ruleErr("تاریخ پایان برنامه نباید پیش از تاریخ شروع باشد.")
	}
	// کوتاه‌کردن بازه با تاریخ پایان تخصیص.
	if raw := strings.TrimSpace(in.EndDate); raw != "" {
		if aEnd, err := calendar.Parse(raw); err == nil {
			if span, err := calendar.DaysBetween(start, aEnd); err == nil && span >= 0 {
				if gap, err := calendar.DaysBetween(aEnd, end); err == nil && gap > 0 {
					end = aEnd
				}
			}
		}
	}
	// کوتاه‌کردن بازه با مبنای چرخه: پیش از تاریخ مؤثر تخصیص نوبتی معلق نمی‌شود، پس
	// آن روزها در برنامه نمی‌آیند — نه اینکه پرس‌وجوی یک ماه کامل خطا شود.
	if cycleStart, err := calendar.Parse(strings.TrimSpace(in.CycleStart)); err == nil {
		if gap, err := calendar.DaysBetween(cycleStart, start); err == nil && gap < 0 {
			start = cycleStart
		}
	}
	// اگر بازهٔ درخواستی کاملاً پیش از آغاز تخصیص بود، برنامه خالی برمی‌گردد.
	if span, err := calendar.DaysBetween(start, end); err != nil || span < 0 {
		return []PlanDay{}, nil
	}

	days, err := calendar.EachDay(start, end)
	if err != nil {
		return nil, ruleErr("بازهٔ برنامه قابل پیمایش نیست.")
	}
	cal := calendar.DefaultWorkCalendar()

	out := make([]PlanDay, 0, len(days))
	for _, d := range days {
		key, _, err := ResolveShiftDay(cycle, in.CycleStart, in.CycleOffset, d.String())
		if err != nil {
			return nil, err
		}
		sh, ok := shiftByKey(key)
		if !ok {
			return nil, ruleErr("شیفت «" + key + "» در جدول شیفت‌های این استقرار نیست.")
		}
		weekday := ""
		if name, err := d.WeekdayFa(); err == nil {
			weekday = name
		}
		weekend, _ := cal.IsWeekend(d)
		out = append(out, PlanDay{
			Date: d.String(), WeekdayFa: weekday,
			EmployeeID: in.EmployeeID, EmployeeName: in.EmployeeName,
			AssignmentID: in.AssignmentID, PatternTitle: in.PatternTitle,
			Shift: sh.Key, ShiftTitleFa: sh.TitleFa,
			Start: sh.Start, End: sh.End, Hours: sh.Hours,
			Night: sh.Night, Rest: sh.Rest,
			Holiday: weekend || cal.IsHoliday(d),
		})
	}
	return out, nil
}

// CoverageRow پوشش یک شیفت در یک روز و کمبود آن نسبت به حداقل پوشش.
type CoverageRow struct {
	Date         string   `json:"date"`
	WeekdayFa    string   `json:"weekdayFa"`
	Shift        string   `json:"shift"`
	ShiftTitleFa string   `json:"shiftTitleFa"`
	Assigned     int      `json:"assigned"`
	MinCoverage  int      `json:"minCoverage"`
	Shortfall    int      `json:"shortfall"`
	Employees    []string `json:"employees,omitempty"`
	NoteFa       string   `json:"noteFa,omitempty"`
}

// Coverage پوشش روزانهٔ شیفت‌ها را می‌شمارد. روزهای استراحت شیفت کاری محسوب نمی‌شوند،
// پس پوششی هم برایشان سنجیده نمی‌شود. حداقل پوشش از الگو می‌آید و اگر چند الگو در
// بازه باشد، سخت‌گیرانه‌ترین (بیشترین) مقدار ملاک است.
func Coverage(days []PlanDay, minCoverage int) []CoverageRow {
	type key struct{ date, shift string }
	order := []key{}
	seen := map[key]*CoverageRow{}
	for _, d := range days {
		if d.Rest {
			continue
		}
		k := key{d.Date, d.Shift}
		row, ok := seen[k]
		if !ok {
			sh, _ := shiftByKey(d.Shift)
			row = &CoverageRow{
				Date: d.Date, WeekdayFa: d.WeekdayFa,
				Shift: d.Shift, ShiftTitleFa: sh.TitleFa, MinCoverage: minCoverage,
			}
			seen[k] = row
			order = append(order, k)
		}
		row.Assigned++
		if d.EmployeeID != "" {
			row.Employees = append(row.Employees, d.EmployeeID)
		}
	}

	out := make([]CoverageRow, 0, len(order))
	for _, k := range order {
		row := seen[k]
		sort.Strings(row.Employees)
		if row.MinCoverage > 0 && row.Assigned < row.MinCoverage {
			row.Shortfall = row.MinCoverage - row.Assigned
			row.NoteFa = "کمبود پوشش: " + itoaFa(row.Shortfall) + " نفر نسبت به حداقل پوشش الگو."
		}
		out = append(out, *row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return shiftOrder(out[i].Shift) < shiftOrder(out[j].Shift)
	})
	return out
}

// CoverageShortfalls فقط ردیف‌های دارای کمبود را برمی‌گرداند (گزارش مغایرت).
func CoverageShortfalls(rows []CoverageRow) []CoverageRow {
	out := make([]CoverageRow, 0, len(rows))
	for _, r := range rows {
		if r.Shortfall > 0 {
			out = append(out, r)
		}
	}
	return out
}

// itoaFa عدد را با ارقام فارسی می‌نویسد (پیام‌های کاربر پسند).
func itoaFa(n int) string {
	digits := []rune("۰۱۲۳۴۵۶۷۸۹")
	if n == 0 {
		return "۰"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := []rune{}
	for n > 0 {
		buf = append([]rune{digits[n%10]}, buf...)
		n /= 10
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}
