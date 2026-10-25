// پیاده‌سازی ماژول «attendance» — «worktime».

package attendance

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/attendance/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// workTimeProvider پورت «کارکرد دوره».
//
// رکوردهای همین ماژول را می‌خواند و به قرارداد `contracts.WorkTimeProvider` تبدیل
// می‌کند. مالکیت داده، قواعد کارکرد و دامنهٔ دسترسی همین‌جا می‌ماند و ماژول حقوق فقط
// از راه قرارداد مصرف می‌کند (بدون هیچ import مستقیمی بین ماژول‌ها).
type workTimeProvider struct {
	svc  *recordkit.Service
	deps modulekit.Deps
}

// workTimeBuckets جمع خام هر کارمند پیش از تبدیل با موتور قواعد.
type workTimeBuckets struct {
	employeeID     string
	employeeName   string
	departmentName string
	records        int
	pendingRecords int
	workedHours    float64
	overtime       float64
	overtimeWeight float64
	pendingOt      float64
	paidLeave      float64
	unpaidLeave    float64
	sickLeave      float64
	pendingLeave   float64
	missionDays    float64
	// ساعت‌های نوبت/شب/تعطیل و تأخیر/تعجیل از ترددهای همان دوره.
	shiftHours   float64
	nightHours   float64
	fridayHours  float64
	lateMinutes  int
	earlyMinutes int
	// attendedDates روزهای دارای تردد و coveredDates روزهایی که مرخصی/مأموریت
	// پوشش داده است؛ مبنای سنجش «غیبت» در برابر روزهای موظفی نوبت مصوب.
	attendedDates map[string]bool
	coveredDates  map[string]bool
}

// WorkTime کارکرد دوره را به تفکیک کارمند می‌سازد. فقط رکوردهای مؤثر شمرده می‌شوند و
// رکوردهای در گردش جدا گزارش می‌شوند تا پرداخت پیش از تأیید انجام نشود.
func (p workTimeProvider) WorkTime(ctx context.Context, q contracts.WorkTimeQuery) ([]contracts.WorkTimeRow, error) {
	period := strings.TrimSpace(q.Period)
	wanted := map[string]bool{}
	for _, id := range q.EmployeeIDs {
		if id = strings.TrimSpace(id); id != "" {
			wanted[id] = true
		}
	}

	buckets := map[string]*workTimeBuckets{}
	get := func(rec contracts.Record) *workTimeBuckets {
		b, ok := buckets[rec.EmployeeID]
		if !ok {
			b = &workTimeBuckets{
				employeeID: rec.EmployeeID, employeeName: rec.EmployeeName, departmentName: rec.DepartmentName,
				attendedDates: map[string]bool{}, coveredDates: map[string]bool{},
			}
			buckets[rec.EmployeeID] = b
		}
		if b.employeeName == "" {
			b.employeeName = rec.EmployeeName
		}
		if b.departmentName == "" {
			b.departmentName = rec.DepartmentName
		}
		return b
	}

	// نوبت مصوب کارکنان و مهلت تأخیر مجاز شرکت: مبنای سنجش شب‌کاری، تأخیر/تعجیل و
	// روزهای موظفی (غیبت). هر دو یک‌بار خوانده می‌شوند، نه در هر رکورد.
	rost := loadRoster(ctx, p.svc)
	grace := graceMinutes(p.deps)

	for _, spec := range []struct {
		kind      string
		dateField string
	}{
		{"attendanceRecords", "date"},
		{"overtime", "date"},
		{"leaveRequests", "fromDate"},
		{"missions", "fromDate"},
	} {
		records, err := p.list(ctx, spec.kind)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			if len(wanted) > 0 && !wanted[rec.EmployeeID] {
				continue
			}
			// رکوردهای دارای اعتراض در کارکرد نمی‌آیند تا تکلیف مغایرت روشن شود.
			if rec.Status == "disputed" {
				continue
			}
			if period != "" && !inPeriod(recordPeriod(rec.Fields[spec.dateField]), period) {
				continue
			}
			b := get(rec)
			effective := domain.EffectiveStatus(rec.Status)
			if effective {
				b.records++
			} else {
				b.pendingRecords++
			}

			switch spec.kind {
			case "attendanceRecords":
				if effective {
					hours := number(rec.Fields["workedHours"])
					b.workedHours += hours
					date := strings.TrimSpace(rec.Fields["date"])
					if date != "" {
						b.attendedDates[date] = true
					}
					if shift, ok := rost.ShiftFor(rec.EmployeeID, date); ok && !shift.Rest {
						b.shiftHours += hours
					}
					b.nightHours += number(rec.Fields["nightHours"])
					b.fridayHours += number(rec.Fields["fridayHours"])
					late, early := lateEarlyOf(rec, rost, grace)
					b.lateMinutes += late
					b.earlyMinutes += early
				}
			case "overtime":
				hours := number(rec.Fields["hours"])
				if effective {
					b.overtime += hours
					b.overtimeWeight += hours * factorOf(rec.Fields["rate"])
				} else {
					b.pendingOt += hours
				}
			case "leaveRequests":
				days := number(rec.Fields["days"])
				if days == 0 {
					days = rangeDays(rec.Fields["fromDate"], rec.Fields["toDate"])
				}
				class := domain.ClassifyLeave(rec.Fields["leaveType"])
				if effective {
					// روزهای پوشش‌داده‌شده با مرخصی، «غیبت» شمرده نمی‌شوند.
					markRange(b.coveredDates, rec.Fields["fromDate"], rec.Fields["toDate"])
				}
				switch {
				case !effective:
					b.pendingLeave += days
				case class.Sick:
					b.sickLeave += days
				case class.Paid:
					b.paidLeave += days
				default:
					b.unpaidLeave += days
				}
			case "missions":
				if effective {
					b.missionDays += rangeDays(rec.Fields["fromDate"], rec.Fields["toDate"])
					markRange(b.coveredDates, rec.Fields["fromDate"], rec.Fields["toDate"])
				}
			}
		}
	}

	params := p.workTimeParams()

	ids := make([]string, 0, len(buckets))
	for id := range buckets {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	rows := make([]contracts.WorkTimeRow, 0, len(ids))
	for _, id := range ids {
		b := buckets[id]
		factor := rules.DefaultOvertimeFactor
		if b.overtime > 0 {
			factor = math2(b.overtimeWeight / b.overtime)
		}
		// غیبت: روزهای موظفی نوبت مصوب که نه تردد دارند و نه مرخصی/مأموریت پوشششان
		// داده است. روزهای آیندهٔ دوره «غیبت» نیستند (کسی برای فردا غایب شمرده
		// نمی‌شود)، پس سنجش فقط تا امروز سرور جلو می‌رود. بدون دوره یا بدون تخصیص
		// مصوب، محاسبه انجام نمی‌شود (صفر می‌ماند).
		expected, missing := 0, 0
		if windowFrom, windowTo, ok := absenceWindow(period, p.deps.Now()); ok {
			dates := rost.ExpectedWorkDates(b.employeeID, windowFrom, windowTo)
			expected = len(dates)
			for _, d := range dates {
				if b.attendedDates[d] || b.coveredDates[d] {
					continue
				}
				missing++
			}
		}
		summary := rules.ComputeWorkTime(rules.WorkTimeInput{
			Period:          period,
			WorkedHours:     round2(b.workedHours),
			OvertimeHours:   round2(b.overtime),
			PaidLeaveDays:   round2(b.paidLeave),
			UnpaidLeaveDays: round2(b.unpaidLeave),
			SickLeaveDays:   round2(b.sickLeave),
			MissionDays:     round2(b.missionDays),
			AbsenceDays:     float64(missing),
		}, params)

		rows = append(rows, contracts.WorkTimeRow{
			EmployeeID:           b.employeeID,
			EmployeeName:         b.employeeName,
			DepartmentName:       b.departmentName,
			Period:               period,
			WorkedHours:          summary.WorkedHours,
			WorkedDays:           summary.WorkedDays,
			OvertimeHours:        summary.OvertimeHours,
			OvertimeFactor:       factor,
			PendingOvertimeHours: round2(b.pendingOt),
			PaidLeaveDays:        summary.PaidLeaveDays,
			UnpaidLeaveDays:      summary.UnpaidLeaveDays,
			SickLeaveDays:        summary.SickLeaveDays,
			PendingLeaveDays:     round2(b.pendingLeave),
			MissionDays:          summary.MissionDays,
			AbsenceDays:          summary.AbsenceDays,
			ShiftHours:           round2(b.shiftHours),
			NightHours:           round2(b.nightHours),
			FridayHours:          round2(b.fridayHours),
			LateMinutes:          b.lateMinutes,
			EarlyLeaveMinutes:    b.earlyMinutes,
			ExpectedDays:         float64(expected),
			MissingDays:          float64(missing),
			PayableDays:          summary.PayableDays,
			DeductibleDays:       summary.DeductibleDays,
			PresenceRatio:        summary.PresenceRatio,
			RecordsCount:         b.records,
			PendingRecords:       b.pendingRecords,
		})
	}
	return rows, nil
}

// lateEarlyOf تأخیر و تعجیل یک رکورد تردد.
//
// قاعده: اگر ورود فایل دستگاه این دو را نوشته باشد همان ملاک است؛ وگرنه از ساعت
// ورود/خروج همان رکورد و نوبت مصوب آن روز دوباره سنجیده می‌شود (تردد دستی هم باید
// تأخیرش شمرده شود).
func lateEarlyOf(rec contracts.Record, rost *roster, grace int) (int, int) {
	if raw := strings.TrimSpace(rec.Fields["lateMinutes"]); raw != "" {
		late, _ := strconv.Atoi(strings.TrimSpace(calendar.NormalizeDigits(raw)))
		early := 0
		if rawEarly := strings.TrimSpace(rec.Fields["earlyMinutes"]); rawEarly != "" {
			early, _ = strconv.Atoi(strings.TrimSpace(calendar.NormalizeDigits(rawEarly)))
		}
		return late, early
	}
	date := strings.TrimSpace(rec.Fields["date"])
	if strings.TrimSpace(rec.Fields["checkIn"]) == "" || strings.TrimSpace(rec.Fields["checkOut"]) == "" {
		return 0, 0
	}
	shift, ok := rost.ShiftFor(rec.EmployeeID, date)
	holiday := false
	if ok {
		holiday = shift.Holiday
	}
	metrics := domain.PunchMetricsFor(domain.DevicePunch{
		Date: date, CheckIn: rec.Fields["checkIn"], CheckOut: rec.Fields["checkOut"],
		WorkedHours: number(rec.Fields["workedHours"]),
	}, shift, grace, holiday)
	return metrics.LateMinutes, metrics.EarlyLeaveMinutes
}

// markRange روزهای یک بازهٔ شمسی را در مجموعه علامت می‌زند.
func markRange(set map[string]bool, from, to string) {
	if len(set) == 0 {
		return
	}
	a, errA := calendar.Parse(strings.TrimSpace(from))
	b, errB := calendar.Parse(strings.TrimSpace(to))
	if errA != nil || errB != nil {
		return
	}
	days, err := calendar.EachDay(a, b)
	if err != nil {
		return
	}
	for _, d := range days {
		set[d.String()] = true
	}
}

// periodRange بازهٔ روزهای یک دورهٔ «YYYY/MM» را می‌سازد.
func periodRange(period string) (string, string) {
	parts := strings.Split(strings.TrimSpace(period), "/")
	if len(parts) != 2 {
		return "", ""
	}
	year, errY := strconv.Atoi(parts[0])
	month, errM := strconv.Atoi(parts[1])
	if errY != nil || errM != nil || year < 1300 || month < 1 || month > 12 {
		return "", ""
	}
	length, err := calendar.MonthLength(year, month)
	if err != nil {
		return "", ""
	}
	return fmt.Sprintf("%04d/%02d/01", year, month), fmt.Sprintf("%04d/%02d/%02d", year, month, length)
}

// Periods دوره‌های دارای کارکرد را از رکوردهای همین ماژول استخراج می‌کند.
func (p workTimeProvider) Periods(ctx context.Context) ([]string, error) {
	set := map[string]bool{}
	for _, spec := range []struct {
		kind      string
		dateField string
	}{
		{"attendanceRecords", "date"}, {"overtime", "date"},
		{"leaveRequests", "fromDate"}, {"missions", "fromDate"},
	} {
		records, err := p.list(ctx, spec.kind)
		if err != nil {
			return nil, err
		}
		for _, rec := range records {
			if period := recordPeriod(rec.Fields[spec.dateField]); period != "" {
				set[period] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for period := range set {
		out = append(out, period)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out, nil
}

// list رکوردهای یک نوع را از انبار همین ماژول می‌خواند.
func (p workTimeProvider) list(ctx context.Context, kind string) ([]contracts.Record, error) {
	if p.svc == nil {
		return nil, nil
	}
	page, err := p.svc.List(ctx, contracts.RecordFilter{Module: "attendance", Kind: kind, Size: 500, Scope: "all"})
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// workTimeParams پارامترهای تبدیل کارکرد را از پروفایل شرکت فعال می‌سازد.
func (p workTimeProvider) workTimeParams() rules.WorkTimeParams {
	out := rules.DefaultWorkTimeParams()
	for _, param := range p.deps.Org.Params {
		if param.Key != "payroll.monthlyHours" {
			continue
		}
		if hours, ok := parseFloat(param.Value); ok && hours > 0 {
			out.MonthHours = hours
		}
	}
	return out
}

// recordPeriod ماه یک تاریخ شمسی (YYYY/MM/DD یا YYYY/MM) را برمی‌گرداند.
func recordPeriod(raw string) string {
	raw = strings.TrimSpace(calendar.NormalizeDigits(raw))
	parts := strings.Split(raw, "/")
	if len(parts) < 2 || len(parts[0]) != 4 || len(parts[1]) != 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}

// inPeriod آیا ماه یک تاریخ با دورهٔ خواسته‌شده یکی است.
func inPeriod(period, want string) bool { return period != "" && period == want }

// rangeDays روزهای یک بازهٔ شمسی، شامل هر دو سر.
func rangeDays(from, to string) float64 {
	a, errA := calendar.Parse(strings.TrimSpace(from))
	b, errB := calendar.Parse(strings.TrimSpace(to))
	if errA != nil || errB != nil {
		return 0
	}
	n, err := calendar.DaysBetween(a, b)
	if err != nil || n < 0 {
		return 0
	}
	return float64(n + 1)
}

// factorOf ضریب اضافه‌کاری یک رکورد؛ اگر نبود، ضریب پیش‌فرض قواعد.
func factorOf(raw string) float64 {
	if f, ok := parseFloat(raw); ok && f > 0 {
		return f
	}
	return rules.DefaultOvertimeFactor
}

// number عدد یک فیلد رکورد (صفر اگر نامعتبر باشد).
func number(raw string) float64 {
	if f, ok := parseFloat(raw); ok {
		return f
	}
	return 0
}

func parseFloat(raw string) (float64, bool) {
	raw = strings.TrimSpace(calendar.NormalizeDigits(raw))
	if raw == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func round2(x float64) float64 { return math2(x) }

// absenceWindow بازهٔ سنجش غیبت یک دوره: از آغاز دوره تا پایان آن، ولی نه پس از
// امروز سرور — چون روزهای نرسیده «غیبت» نیستند و نباید مبنای کسر از مزد شوند.
//
// ok=false یعنی در این دوره روز سپری‌شده‌ای نیست (دورهٔ آینده) یا دوره نامعتبر است.
func absenceWindow(period string, now time.Time) (string, string, bool) {
	from, to := periodRange(period)
	if from == "" {
		return "", "", false
	}
	first, errFirst := calendar.Parse(from)
	last, errLast := calendar.Parse(to)
	today, errToday := calendar.FromGregorian(now)
	if errFirst != nil || errLast != nil || errToday != nil {
		return "", "", false
	}
	if gap, err := calendar.DaysBetween(first, today); err != nil || gap < 0 {
		return "", "", false
	}
	if gap, err := calendar.DaysBetween(today, last); err != nil || gap > 0 {
		to = today.String()
	}
	return from, to, true
}

func math2(x float64) float64 {
	if x < 0 {
		return float64(int64(x*100-0.5)) / 100
	}
	return float64(int64(x*100+0.5)) / 100
}
