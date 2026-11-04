// پیاده‌سازی ماژول «attendance» — «تقویم کارکرد».
//
// چرا این فایل: تقویم کاری شرکت فقط تعطیل هفتگی و تعطیلات رسمی را نشان می‌داد، در حالی
// که نسخه‌های پیشین تقویم، روزها را با «وضعیت» نشان می‌داد (تعطیل رسمی، مرخصی، مأموریت،
// اضافه‌کاری، دوره/مناسبت). این مسیر همان نما را روی دادهٔ واقعی می‌سازد: هر روز ماه با
// برچسب‌های وضعیتش، از رکوردهای خودِ همین ماژول (تردد، مرخصی، مأموریت، اضافه‌کاری) و از
// تقویم کاری استقرار (تعطیل هفتگی/رسمی) و مناسبت‌های تقویمی.
//
// قاعدهٔ معماری: هیچ قاعدهٔ تازه‌ای اینجا ساخته نمی‌شود. «مؤثر بودن» رکورد از دامنه
// (`domain.EffectiveStatus`) و «روز کاری» از بستهٔ تقویم (`deps.Calendar`) می‌آید؛ این
// فایل فقط همان‌ها را به نمای ماهانه تبدیل می‌کند.

package attendance

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/attendance/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// registerWorkCalendarEndpoint مسیر «تقویم کارکرد» را ثبت می‌کند.
func registerWorkCalendarEndpoint(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("attendance", "read", fn) }

	// تقویم یک ماه با وضعیت هر روز: تعطیل هفتگی، تعطیل رسمی، مناسبت، مرخصی، مأموریت،
	// اضافه‌کاری، کارکرد ثبت‌شده و روزهای کاری بدون کارکرد.
	mux.HandleFunc("GET /api/v1/attendance/work-calendar", read(func(w http.ResponseWriter, r *http.Request) {
		view := workCalendarView(r.Context(), deps, svc, r)
		httpx.WriteJSON(w, http.StatusOK, view)
	}))
}

/* ----------------------------- مدل پاسخ ----------------------------- */

// calendarChip یک برچسب وضعیت روی یک روز (مرخصی، مأموریت، اضافه‌کاری، مناسبت...).
type calendarChip struct {
	Kind     string `json:"kind"`
	TitleFa  string `json:"titleFa"`
	DetailFa string `json:"detailFa,omitempty"`
	// EmployeeFa/EmployeeID برای برچسب‌های وابسته به کارمند؛ برچسب عمومی (تعطیل،
	// مناسبت) خالی می‌ماند.
	EmployeeFa string `json:"employeeFa,omitempty"`
	EmployeeID string `json:"employeeId,omitempty"`
	RecordID   string `json:"recordId,omitempty"`
	// Hours و Days مقدار همان رکورد (ساعت اضافه‌کاری/کارکرد یا روز مرخصی).
	Hours float64 `json:"hours,omitempty"`
	Days  float64 `json:"days,omitempty"`
	// Pending یعنی رکورد هنوز در گردش تأیید است و در جمع کارکرد شمرده نمی‌شود.
	Pending  bool   `json:"pending,omitempty"`
	Status   string `json:"status,omitempty"`
	StatusFa string `json:"statusFa,omitempty"`
}

// calendarDay یک روز از تقویم کارکرد.
type calendarDay struct {
	Date         string `json:"date"`
	Day          int    `json:"day"`
	WeekdayIndex int    `json:"weekdayIndex"`
	WeekdayFa    string `json:"weekdayFa"`
	Weekend      bool   `json:"weekend"`
	Holiday      bool   `json:"holiday"`
	// HolidayTitleFa عنوان تعطیل رسمی (از تقویم کاری استقرار).
	HolidayTitleFa string `json:"holidayTitleFa,omitempty"`
	Working        bool   `json:"working"`
	// StatusKey/StatusFa وضعیت جمع‌بندی‌شدهٔ روز برای رنگ‌آمیزی و متن رابط:
	// holiday | weekend | leave | mission | mixed | worked | missing | off.
	StatusKey string `json:"statusKey"`
	StatusFa  string `json:"statusFa"`
	// Today آیا این روز، امروزِ استقرار است.
	Today         bool           `json:"today,omitempty"`
	Chips         []calendarChip `json:"chips"`
	LeaveCount    int            `json:"leaveCount"`
	MissionCount  int            `json:"missionCount"`
	WorkedHours   float64        `json:"workedHours"`
	OvertimeHours float64        `json:"overtimeHours"`
	LateMinutes   int            `json:"lateMinutes"`
	// MissingWork یعنی روز کاریِ موظفی که نه ترددی دارد و نه مرخصی/مأموریتی پوشش داده.
	MissingWork bool `json:"missingWork,omitempty"`
}

// workCalendarTotals جمع ماه — همان اعدادی که پیش از اجرای حقوق نگاه می‌شود.
type workCalendarTotals struct {
	DaysCount       int     `json:"daysCount"`
	WorkingDays     int     `json:"workingDays"`
	WeekendDays     int     `json:"weekendDays"`
	Holidays        int     `json:"holidays"`
	Occasions       int     `json:"occasions"`
	LeaveDays       float64 `json:"leaveDays"`
	MissionDays     float64 `json:"missionDays"`
	OvertimeHours   float64 `json:"overtimeHours"`
	WorkedHours     float64 `json:"workedHours"`
	LateMinutes     int     `json:"lateMinutes"`
	MissingWorkDays int     `json:"missingWorkDays"`
	PendingRecords  int     `json:"pendingRecords"`
}

// legendItem راهنمای برچسب‌ها برای رابط (عنوان‌ها از سرور، رنگ در رابط).
type legendItem struct {
	Kind    string `json:"kind"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa,omitempty"`
}

/* ----------------------------- ساخت نما ----------------------------- */

// workCalendarView نمای یک ماه شمسی از تقویم کارکرد را می‌سازد.
//
// پارامترها: `month` (YYYY/MM یا YYYY/MM/DD؛ خالی یعنی ماه جاری) و `employee` (کد کارمند
// برای تقویم یک نفر؛ خالی یعنی همهٔ کارکنان در محدودهٔ دسترسی کاربر).
func workCalendarView(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, r *http.Request) map[string]any {
	today, err := calendar.FromGregorian(deps.Clock())
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	target := today
	if parsed, ok := parseCalendarMonth(r.URL.Query().Get("month")); ok {
		target = parsed
	}
	length, err := calendar.MonthLength(target.Year, target.Month)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	monthKey := strconv.Itoa(target.Year) + "/" + pad2(target.Month)

	employeeID := strings.TrimSpace(r.URL.Query().Get("employee"))
	scope := "all"
	actorEmployee, actorDepartment := "", ""
	if user, ok := auth.UserFrom(r.Context()); ok {
		switch user.Role {
		case contracts.RoleEmployee:
			scope, actorEmployee = "own", user.EmployeeID
		case contracts.RoleLineManager:
			scope, actorDepartment = "own_unit", user.DepartmentID
		}
	}

	days := map[string]*calendarDay{}
	order := make([]string, 0, length)
	for day := 1; day <= length; day++ {
		d := calendar.Date{Year: target.Year, Month: target.Month, Day: day}
		weekday, _ := d.WeekdayFa()
		index, _ := d.WeekdayIndex()
		weekend, _ := deps.Calendar.IsWeekend(d)
		holiday := deps.Calendar.IsHoliday(d)
		working, _ := deps.Calendar.IsWorkingDay(d)
		item := &calendarDay{
			Date: d.String(), Day: day, WeekdayIndex: index, WeekdayFa: weekday,
			Weekend: weekend, Holiday: holiday, Working: working, Today: d.String() == today.String(),
		}
		if holiday {
			item.HolidayTitleFa = strings.TrimSpace(deps.Calendar.Holidays[d.String()])
			item.StatusKey, item.StatusFa = "holiday", "تعطیل رسمی"
			item.Chips = append(item.Chips, calendarChip{Kind: "holiday", TitleFa: "تعطیل رسمی", DetailFa: item.HolidayTitleFa})
		} else if weekend {
			item.StatusKey, item.StatusFa = "weekend", "تعطیل هفتگی"
		} else {
			item.StatusKey, item.StatusFa = "pending", "در انتظار داده"
		}
		days[d.String()] = item
		order = append(order, d.String())
	}

	totals := workCalendarTotals{DaysCount: length}
	for _, key := range order {
		switch days[key].StatusKey {
		case "holiday":
			totals.Holidays++
		case "weekend":
			totals.WeekendDays++
		}
		if days[key].Working {
			totals.WorkingDays++
		}
	}

	// مناسبت‌های تقویمی: برچسب روز، مستقل از حضور و مرخصی.
	occasionCount := addOccasions(ctx, deps, days, order, target)
	totals.Occasions = occasionCount

	// رکوردهای حضور، مرخصی، مأموریت و اضافه‌کاری همان ماه.
	filter := func(kind string) contracts.RecordFilter {
		return contracts.RecordFilter{
			Module: "attendance", Kind: kind, Scope: scope, Size: 500,
			EmployeeID: employeeID, ActorEmployeeID: actorEmployee, ActorDepartment: actorDepartment,
		}
	}
	for _, spec := range []struct {
		kind      string
		dateField string
	}{
		{"attendanceRecords", "date"},
		{"overtime", "date"},
		{"leaveRequests", "fromDate"},
		{"missions", "fromDate"},
	} {
		records := listRecords(ctx, svc, filter(spec.kind))
		for _, rec := range records {
			applyCalendarRecord(spec.kind, rec, days, order, monthKey, &totals)
		}
	}

	// وضعیت جمع‌بندی هر روز و شمار روزهای کاری بدون کارکرد.
	out := make([]calendarDay, 0, length)
	for _, key := range order {
		item := days[key]
		settleDayStatus(item)
		if item.MissingWork {
			totals.MissingWorkDays++
		}
		if item.Chips == nil {
			item.Chips = []calendarChip{}
		}
		out = append(out, *item)
	}

	sort.Strings(order)
	totals.LeaveDays = round1(totals.LeaveDays)
	totals.MissionDays = round1(totals.MissionDays)
	totals.OvertimeHours = round1(totals.OvertimeHours)
	totals.WorkedHours = round1(totals.WorkedHours)

	return map[string]any{
		"month": map[string]any{
			"key": monthKey, "label": target.MonthLabel(), "days": out,
			"totals": totals,
		},
		"today": map[string]any{
			"date": today.String(), "weekdayFa": func() string { w, _ := today.WeekdayFa(); return w }(),
			"monthLabel": today.MonthLabel(),
		},
		"company":  deps.Org.TitleFa,
		"employee": employeeID,
		"scopeFa":  scopeLabelFa(scope),
		"legend":   calendarLegend(),
		"noteFa": "هر روز ماه با وضعیتش نشان داده می‌شود: تعطیل هفتگی و تعطیل رسمی از تقویم کاری شرکت، " +
			"مناسبت‌ها از تقویم مناسبت‌ها، و مرخصی/مأموریت/اضافه‌کاری/کارکرد از رکوردهای همین ماژول. " +
			"رکوردهای در گردش تأیید با نشان «در انتظار» می‌آیند و در جمع کارکرد شمرده نمی‌شوند؛ " +
			"رکوردهای ردشده و لغوشده اصلاً روی تقویم نمی‌آیند.",
	}
}

// addOccasions مناسبت‌های تقویمی همان ماه را روی روزها می‌نشاند.
//
// مناسبت‌ها مالِ ماژول «کارت و رفاه» هستند و از راه پورت رکوردها خوانده می‌شوند.
func addOccasions(ctx context.Context, deps modulekit.Deps, days map[string]*calendarDay, order []string, target calendar.Date) int {
	port, ok := deps.Ports.RecordList("cards")
	if !ok {
		return 0
	}
	page, err := port.List(ctx, contracts.RecordFilter{Module: "cards", Kind: "occasions", Scope: "all", Size: 500})
	if err != nil {
		return 0
	}
	count := 0
	for _, rec := range page.Items {
		if strings.TrimSpace(rec.Status) == "inactive" {
			continue
		}
		date := strings.TrimSpace(calendar.NormalizeDigits(rec.Fields["date"]))
		item, ok := days[date]
		if !ok {
			continue
		}
		detail := strings.TrimSpace(rec.Fields["categoryFa"])
		if scope := strings.TrimSpace(rec.Fields["scopeFa"]); scope != "" {
			detail = strings.TrimSpace(detail + " — " + scope)
		}
		item.Chips = append(item.Chips, calendarChip{
			Kind: "occasion", TitleFa: rec.Title, DetailFa: strings.TrimSuffix(strings.TrimPrefix(detail, " — "), " — "),
			RecordID: rec.ID,
		})
		count++
	}
	return count
}

// applyCalendarRecord اثر یک رکورد حضور/مرخصی/مأموریت/اضافه‌کاری را روی روزها می‌گذارد.
func applyCalendarRecord(kind string, rec contracts.Record, days map[string]*calendarDay, order []string, monthKey string, totals *workCalendarTotals) {
	status := strings.TrimSpace(rec.Status)
	// رکورد ردشده/لغوشده روی تقویم نمی‌آید؛ نه کارکرد است نه تعهد.
	if status == "rejected" || status == "cancelled" {
		return
	}
	effective := domain.EffectiveStatus(status)
	pending := !effective
	if pending {
		totals.PendingRecords++
	}
	statusFa := strings.TrimSpace(rec.StatusFa)
	if statusFa == "" {
		statusFa = attendanceStatusFa(status)
	}

	dates := recordDates(kind, rec, monthKey)
	if len(dates) == 0 {
		return
	}
	daysCount := float64(len(dates))

	switch kind {
	case "attendanceRecords":
		hours := number(rec.Fields["workedHours"])
		late := int(number(rec.Fields["lateMinutes"]))
		for _, date := range dates {
			item, ok := days[date]
			if !ok {
				continue
			}
			item.Chips = append(item.Chips, calendarChip{
				Kind: "attendance", TitleFa: "کارکرد", EmployeeFa: rec.EmployeeName, EmployeeID: rec.EmployeeID,
				RecordID: rec.ID, Hours: hours, Pending: pending, Status: status, StatusFa: statusFa,
				DetailFa: attendanceDetailFa(rec, hours, late),
			})
			if effective {
				item.WorkedHours += hours
				item.LateMinutes += late
			}
		}
		if effective {
			totals.WorkedHours += hours * daysCount
			totals.LateMinutes += late
		}
	case "overtime":
		hours := number(rec.Fields["hours"])
		for _, date := range dates {
			item, ok := days[date]
			if !ok {
				continue
			}
			item.Chips = append(item.Chips, calendarChip{
				Kind: "overtime", TitleFa: "اضافه‌کاری", EmployeeFa: rec.EmployeeName, EmployeeID: rec.EmployeeID,
				RecordID: rec.ID, Hours: hours, Pending: pending, Status: status, StatusFa: statusFa,
			})
			if effective {
				item.OvertimeHours += hours
			}
		}
		if effective {
			totals.OvertimeHours += hours
		}
	case "leaveRequests":
		daysField := number(rec.Fields["days"])
		if daysField > 0 {
			daysCount = daysField
		}
		leaveType := strings.TrimSpace(rec.Fields["leaveType"])
		class := domain.ClassifyLeave(leaveType)
		for _, date := range dates {
			item, ok := days[date]
			if !ok {
				continue
			}
			item.Chips = append(item.Chips, calendarChip{
				Kind: "leave", TitleFa: "مرخصی — " + rec.EmployeeName, EmployeeFa: rec.EmployeeName,
				EmployeeID: rec.EmployeeID, RecordID: rec.ID, Days: daysCount, Pending: pending,
				Status: status, StatusFa: statusFa, DetailFa: leaveDetailFa(leaveType, class, rec),
			})
			item.LeaveCount++
		}
		if effective {
			totals.LeaveDays += daysCount
		}
	case "missions":
		destination := strings.TrimSpace(rec.Fields["destination"])
		for _, date := range dates {
			item, ok := days[date]
			if !ok {
				continue
			}
			detail := destination
			if purpose := strings.TrimSpace(rec.Fields["purpose"]); purpose != "" {
				detail = strings.TrimSpace(detail + " — " + purpose)
			}
			item.Chips = append(item.Chips, calendarChip{
				Kind: "mission", TitleFa: "مأموریت — " + rec.EmployeeName, EmployeeFa: rec.EmployeeName,
				EmployeeID: rec.EmployeeID, RecordID: rec.ID, Days: daysCount, Pending: pending,
				Status: status, StatusFa: statusFa, DetailFa: detail,
			})
			item.MissionCount++
		}
		if effective {
			totals.MissionDays += daysCount
		}
	}
}

// recordDates روزهای شمسی یک رکورد در ماه خواسته‌شده را برمی‌گرداند.
func recordDates(kind string, rec contracts.Record, monthKey string) []string {
	if kind == "leaveRequests" || kind == "missions" {
		from := strings.TrimSpace(rec.Fields["fromDate"])
		to := strings.TrimSpace(rec.Fields["toDate"])
		if to == "" {
			to = from
		}
		a, errA := calendar.Parse(from)
		b, errB := calendar.Parse(to)
		if errA != nil || errB != nil {
			return nil
		}
		each, err := calendar.EachDay(a, b)
		if err != nil {
			return nil
		}
		out := []string{}
		for _, d := range each {
			if recordPeriod(d.String()) == monthKey {
				out = append(out, d.String())
			}
		}
		return out
	}
	date := strings.TrimSpace(calendar.NormalizeDigits(rec.Fields["date"]))
	if recordPeriod(date) != monthKey {
		return nil
	}
	if _, err := calendar.Parse(date); err != nil {
		return nil
	}
	return []string{date}
}

// settleDayStatus وضعیت جمع‌بندی روز را از برچسب‌هایش می‌سازد.
func settleDayStatus(day *calendarDay) {
	switch {
	case day.Holiday:
		day.StatusKey, day.StatusFa = "holiday", "تعطیل رسمی"
		return
	case day.Weekend:
		day.StatusKey, day.StatusFa = "weekend", "تعطیل هفتگی"
		return
	}
	switch {
	case day.LeaveCount > 0 && day.MissionCount > 0:
		day.StatusKey, day.StatusFa = "mixed", "مرخصی و مأموریت"
	case day.LeaveCount > 0:
		day.StatusKey, day.StatusFa = "leave", "مرخصی"
	case day.MissionCount > 0:
		day.StatusKey, day.StatusFa = "mission", "مأموریت"
	case day.WorkedHours > 0:
		day.StatusKey, day.StatusFa = "worked", "کارکرد ثبت‌شده"
	default:
		day.MissingWork = true
		day.StatusKey, day.StatusFa = "missing", "بدون کارکرد ثبت‌شده"
	}
}

// listRecords رکوردهای یک نوع را با محدودهٔ دسترسی نقش می‌خواند.
func listRecords(ctx context.Context, svc *recordkit.Service, filter contracts.RecordFilter) []contracts.Record {
	if svc == nil {
		return nil
	}
	page, err := svc.List(ctx, filter)
	if err != nil {
		return nil
	}
	return page.Items
}

// calendarLegend راهنمای برچسب‌های تقویم برای رابط.
func calendarLegend() []legendItem {
	return []legendItem{
		{Kind: "holiday", TitleFa: "تعطیل رسمی", NoteFa: "از تقویم کاری استقرار شرکت"},
		{Kind: "weekend", TitleFa: "تعطیل هفتگی", NoteFa: "مطابق هفتهٔ کاری شرکت"},
		{Kind: "occasion", TitleFa: "مناسبت تقویمی", NoteFa: "از ماژول کارت و رفاه"},
		{Kind: "leave", TitleFa: "مرخصی"},
		{Kind: "mission", TitleFa: "مأموریت"},
		{Kind: "overtime", TitleFa: "اضافه‌کاری"},
		{Kind: "attendance", TitleFa: "کارکرد تردد"},
		{Kind: "missing", TitleFa: "بدون کارکرد ثبت‌شده", NoteFa: "روز کاریِ موظفی بدون تردد و بدون مرخصی/مأموریت"},
	}
}

/* ----------------------------- کمک‌کارها ----------------------------- */

// attendanceStatusFa برچسب فارسی وضعیت رکوردهای حضور (وقتی سرور عنوان نداده باشد).
func attendanceStatusFa(status string) string {
	switch strings.TrimSpace(status) {
	case "recorded", "draft":
		return "ثبت‌شده"
	case "confirmed", "approved":
		return "تأییدشده"
	case "unitApproved":
		return "تأیید مدیر واحد"
	case "pending":
		return "در انتظار تأیید"
	case "settled":
		return "تسویه‌شده"
	case "disputed":
		return "دارای اعتراض"
	}
	return strings.TrimSpace(status)
}

// attendanceDetailFa شرح یک تردد: ساعت کارکرد، تأخیر و نوبت روز.
func attendanceDetailFa(rec contracts.Record, hours float64, late int) string {
	parts := []string{}
	if hours > 0 {
		parts = append(parts, strconv.FormatFloat(hours, 'f', -1, 64)+" ساعت کارکرد")
	}
	if late > 0 {
		parts = append(parts, strconv.Itoa(late)+" دقیقه تأخیر")
	}
	if night := number(rec.Fields["nightHours"]); night > 0 {
		parts = append(parts, strconv.FormatFloat(night, 'f', -1, 64)+" ساعت شب‌کاری")
	}
	if holiday := number(rec.Fields["fridayHours"]); holiday > 0 {
		parts = append(parts, strconv.FormatFloat(holiday, 'f', -1, 64)+" ساعت تعطیل‌کاری")
	}
	return strings.Join(parts, " — ")
}

// leaveDetailFa شرح یک مرخصی: نوع، پرداختی/کسری و روزها.
func leaveDetailFa(leaveType string, class domain.LeaveClass, rec contracts.Record) string {
	parts := []string{}
	if strings.TrimSpace(leaveType) != "" {
		parts = append(parts, leaveType)
	}
	if class.Paid {
		parts = append(parts, "پرداختی")
	} else {
		parts = append(parts, "کسری از مزد")
	}
	if days := number(rec.Fields["days"]); days > 0 {
		parts = append(parts, strconv.FormatFloat(days, 'f', -1, 64)+" روز")
	}
	return strings.Join(parts, " — ")
}

// scopeLabelFa توضیح محدودهٔ دادهٔ تقویم برای رابط.
func scopeLabelFa(scope string) string {
	switch scope {
	case "own":
		return "فقط رکوردهای خودِ کاربر"
	case "own_unit":
		return "فقط کارکنان واحد کاربر"
	}
	return "همهٔ کارکنان"
}

// pad2 عدد ماه را دو رقمی میکند.
// parseCalendarMonth رشتهٔ «YYYY/MM» (یا «YYYY/MM/DD») را به تاریخ روز نخست همان ماه
// تبدیل می‌کند؛ ورودی نامعتبر ok=false می‌دهد تا ماه جاری به‌کار برود.
func parseCalendarMonth(raw string) (calendar.Date, bool) {
	raw = strings.TrimSpace(calendar.NormalizeDigits(raw))
	parts := strings.Split(raw, "/")
	if len(parts) < 2 || len(parts) > 3 {
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

// pad2 عدد ماه را دو رقمی می‌کند.
func pad2(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

// round1 عدد را به یک رقم اعشار گرد میکند تا جمع‌های شناور روی تقویم تمیز بماند.
func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
