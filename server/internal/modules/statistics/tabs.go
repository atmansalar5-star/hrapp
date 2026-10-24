// پیاده‌سازی ماژول «statistics» — «tabs».

package statistics

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// این فایل تب‌های تجمیعی ماژول «آمار و پایش اطلاعات پایه» را می‌سازد.
//
// قاعدهٔ معماری: هیچ قاعدهٔ تازه‌ای اینجا ساخته نمی‌شود. رکوردها فقط از راه port
// قراردادیِ ماژول مالک (`Ports.RecordList`) خوانده می‌شوند و کارکنان از پورت کارکنان؛
// سپس همان‌جا «جمع» زده می‌شود. برش‌ها و شمارش‌ها داده‌اند، نه قاعدهٔ کسب‌وکار.
func registerTabEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("statistics", "read", fn) }

	mux.HandleFunc("GET /api/v1/statistics/leave", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, leaveStats(r.Context(), deps))
	}))
	mux.HandleFunc("GET /api/v1/statistics/missions", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, missionStats(r.Context(), deps))
	}))
	mux.HandleFunc("GET /api/v1/statistics/karana", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, karanaStats(r.Context(), deps))
	}))
	mux.HandleFunc("GET /api/v1/statistics/dashboard", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, dashboardStats(r.Context(), deps))
	}))
}

// statRow یک ردیف برش آماری (نوع مرخصی، واحد سازمانی، دورهٔ کارانه…).
type statRow struct {
	Key      string  `json:"key"`
	TitleFa  string  `json:"titleFa"`
	Count    int     `json:"count"`
	Days     float64 `json:"days"`
	Hours    float64 `json:"hours"`
	Amount   float64 `json:"amount"`
	Approved int     `json:"approved"`
	Pending  int     `json:"pending"`
	Rejected int     `json:"rejected"`
	// SharePct سهم این ردیف از جمع کل سنجهٔ اصلی تب (درصد).
	SharePct float64 `json:"sharePct"`
}

// tabStat پاسخ یک تب آماری: جمع کل + ردیف‌های برش.
type tabStat struct {
	Tab       string    `json:"tab"`
	TitleFa   string    `json:"titleFa"`
	SubFa     string    `json:"subFa"`
	UnitFa    string    `json:"unitFa"`
	Total     int       `json:"total"`
	SumDays   float64   `json:"sumDays"`
	SumHours  float64   `json:"sumHours"`
	SumAmount float64   `json:"sumAmount"`
	Approved  int       `json:"approved"`
	Pending   int       `json:"pending"`
	Rejected  int       `json:"rejected"`
	Rows      []statRow `json:"rows"`
	NoteFa    string    `json:"noteFa,omitempty"`
	// AvailableFor غایب‌بودن پورت یک ماژول را شفاف می‌گوید (نه صفر پنهان).
	Available bool `json:"available"`
}

// bump شمارندهٔ وضعیت یک ردیف را بر پایهٔ دستهٔ تأیید بالا می‌برد.
func bump(row *statRow, status string) {
	switch statusBucket(status) {
	case "approved":
		row.Approved++
	case "rejected":
		row.Rejected++
	default:
		row.Pending++
	}
}

// statusBucket دستهٔ تأییدی یک وضعیت رکورد را برمی‌گرداند: approved | rejected | pending.
// نام وضعیت‌ها در دامنهٔ هر ماژول اعلام می‌شود؛ این تابع فقط آن‌ها را به سه سبد آماری
// می‌نگارد تا همهٔ تب‌ها یک زبان داشته باشند.
func statusBucket(status string) string {
	switch status {
	case "approved", "confirmed", "settled", "paid", "issued", "active", "locked", "verified":
		return "approved"
	case "rejected", "cancelled", "disputed", "terminated", "expired", "overdue":
		return "rejected"
	default:
		return "pending"
	}
}

// leaveStats برش مرخصی‌ها به تفکیک نوع مرخصی.
func leaveStats(ctx context.Context, deps modulekit.Deps) tabStat {
	out := tabStat{
		Tab: "leave", TitleFa: "مرخصی‌ها", UnitFa: "روز", Rows: []statRow{}, Available: true,
		SubFa: "برش درخواست‌های مرخصی دوره به تفکیک نوع و وضعیت تأیید",
	}
	groups := map[string]*statRow{}
	var totals statRow
	for _, rec := range fetchRecords(ctx, deps, "leaveRequests") {
		key := strings.TrimSpace(rec.Fields["leaveType"])
		if key == "" {
			key = "نامشخص"
		}
		row, ok := groups[key]
		if !ok {
			row = &statRow{Key: key, TitleFa: key}
			groups[key] = row
		}
		days := num(rec.Fields["days"])
		row.Count++
		row.Days += days
		bump(row, rec.Status)
		totals.Count++
		totals.Days += days
		bump(&totals, rec.Status)
	}
	out.Rows = sortedRows(groups, func(r statRow) float64 { return r.Days })
	for i := range out.Rows {
		out.Rows[i].SharePct = sharePct(out.Rows[i].Days, totals.Days)
	}
	out.Total, out.SumDays = totals.Count, totals.Days
	out.Approved, out.Pending, out.Rejected = totals.Approved, totals.Pending, totals.Rejected
	out.NoteFa = "ماندهٔ مرخصی و سقف سالانه در پروندهٔ کارمند و پارامترهای شرکت مرجع است؛ اینجا فقط مرخصی‌های ثبت‌شدهٔ دوره جمع زده می‌شود."
	return out
}

// missionStats برش مأموریت‌ها به تفکیک واحد سازمانی کارمند.
func missionStats(ctx context.Context, deps modulekit.Deps) tabStat {
	out := tabStat{
		Tab: "missions", TitleFa: "ماموریت‌ها", UnitFa: "روز", Rows: []statRow{}, Available: true,
		SubFa: "برش مأموریت‌ها به تفکیک واحد سازمانی و فوق‌العادهٔ برآوردی",
	}
	index := employeeIndex(ctx, deps)
	perDiem := orgParamInt(deps, "mission.perDiem")
	groups := map[string]*statRow{}
	var totals statRow
	for _, rec := range fetchRecords(ctx, deps, "missions") {
		dept := "نامشخص"
		if e, ok := index[rec.EmployeeID]; ok && strings.TrimSpace(e.DepartmentName) != "" {
			dept = e.DepartmentName
		}
		row, ok := groups[dept]
		if !ok {
			row = &statRow{Key: dept, TitleFa: dept}
			groups[dept] = row
		}
		days := recordDays(rec.Fields["fromDate"], rec.Fields["toDate"])
		row.Count++
		row.Days += days
		row.Amount += days * float64(perDiem)
		bump(row, rec.Status)
		totals.Count++
		totals.Days += days
		totals.Amount += days * float64(perDiem)
		bump(&totals, rec.Status)
	}
	out.Rows = sortedRows(groups, func(r statRow) float64 { return r.Days })
	for i := range out.Rows {
		out.Rows[i].SharePct = sharePct(out.Rows[i].Days, totals.Days)
	}
	out.Total, out.SumDays, out.SumAmount = totals.Count, totals.Days, totals.Amount
	out.Approved, out.Pending, out.Rejected = totals.Approved, totals.Pending, totals.Rejected
	if perDiem == 0 {
		out.NoteFa = "نرخ فوق‌العادهٔ روزانهٔ مأموریت در پارامترهای شرکت فعال تعریف نشده است، پس ستون مبلغ خالی می‌ماند."
	} else {
		out.NoteFa = "روز مأموریت از فاصلهٔ تاریخ شروع و پایان (شامل هر دو سر) محاسبه می‌شود. ستون فوق‌العاده برآوردی است؛ مبلغ قطعی در فیش حقوقی و موتور اقلام حقوق محاسبه می‌شود."
	}
	return out
}

// karanaStats برش کارانهٔ دوره از ماژول حقوق و دستمزد.
func karanaStats(ctx context.Context, deps modulekit.Deps) tabStat {
	out := tabStat{
		Tab: "karana", TitleFa: "کارانه و پرداخت‌های متغیر", UnitFa: "ریال", Rows: []statRow{},
		SubFa: "برش کارانه به تفکیک دورهٔ پرداخت و وضعیت تأیید",
	}
	lister, ok := deps.Ports.RecordList("payroll")
	if !ok {
		out.NoteFa = "پورت فقط‌خواندنی ماژول حقوق و دستمزد در این استقرار در دسترس نیست؛ برای دیدن کارانه، آن ماژول باید پورت رکوردهایش را ثبت کند."
		return out
	}
	out.Available = true
	page, err := lister.List(ctx, contracts.RecordFilter{Kind: "karana", Size: 500})
	if err != nil {
		out.NoteFa = "خواندن رکوردهای کارانه ناموفق بود."
		return out
	}
	groups := map[string]*statRow{}
	var totals statRow
	for _, rec := range page.Items {
		key := strings.TrimSpace(rec.Fields["period"])
		if key == "" {
			key = "دورهٔ نامشخص"
		}
		row, ok := groups[key]
		if !ok {
			row = &statRow{Key: key, TitleFa: key}
			groups[key] = row
		}
		amount := num(rec.Fields["amount"])
		row.Count++
		row.Amount += amount
		bump(row, rec.Status)
		totals.Count++
		totals.Amount += amount
		bump(&totals, rec.Status)
	}
	out.Rows = sortedRows(groups, func(r statRow) float64 { return r.Amount })
	for i := range out.Rows {
		out.Rows[i].SharePct = sharePct(out.Rows[i].Amount, totals.Amount)
	}
	out.Total, out.SumAmount = totals.Count, totals.Amount
	out.Approved, out.Pending, out.Rejected = totals.Approved, totals.Pending, totals.Rejected
	out.NoteFa = "گردش‌کار کارانه: محاسبه → تأیید مدیر واحد → ارسال به واحد آمار → پرداخت. مبالغ از رکوردهای خودِ ماژول حقوق خوانده می‌شود."
	return out
}

// dashboardStats نمای تجمیعی «داشبورد آماری»: جمعیت کارکنان، جرم حقوقی، حضور و صف تأیید.
func dashboardStats(ctx context.Context, deps modulekit.Deps) map[string]any {
	head := headcountStats(ctx, deps)
	att := attendanceStats(ctx, deps)
	leave := leaveStats(ctx, deps)
	mission := missionStats(ctx, deps)

	byDept := []statRow{}
	if raw, ok := head["byDepartment"].(map[string]int); ok {
		groups := map[string]*statRow{}
		for name, count := range raw {
			title := name
			if strings.TrimSpace(title) == "" {
				title = "نامشخص"
			}
			groups[title] = &statRow{Key: title, TitleFa: title, Count: count}
		}
		byDept = sortedRows(groups, func(r statRow) float64 { return float64(r.Count) })
		var total int
		for _, r := range byDept {
			total += r.Count
		}
		for i := range byDept {
			byDept[i].SharePct = sharePct(float64(byDept[i].Count), float64(total))
		}
	}

	// صف تأیید: رکوردهای در انتظار تصمیم، به تفکیک وضعیت (همان زبان همهٔ ماژول‌ها).
	statusRows := []statRow{}
	pendingApproval := 0
	for name, count := range att.ByStatus {
		title := name
		if label, ok := statusLabelFa(name); ok {
			title = label
		}
		statusRows = append(statusRows, statRow{Key: name, TitleFa: title, Count: count})
		if statusBucket(name) == "pending" {
			pendingApproval += count
		}
	}
	sort.SliceStable(statusRows, func(i, j int) bool { return statusRows[i].Count > statusRows[j].Count })

	return map[string]any{
		"headcount": head,
		"attendance": map[string]any{
			"records": att.Records, "workedHours": att.WorkedHours, "overtimeHours": att.OvertimeHours,
			"leaveDays": att.LeaveDays, "missionDays": mission.SumDays,
			"pending": att.Pending, "approved": att.Approved, "rejected": att.Rejected,
		},
		"leave":     map[string]any{"total": leave.Total, "days": leave.SumDays, "rows": leave.Rows},
		"missions":  map[string]any{"total": mission.Total, "days": mission.SumDays, "amount": mission.SumAmount, "rows": mission.Rows},
		"byStatus":  statusRows,
		"byDept":    byDept,
		"todoCount": pendingApproval,
		"noteFa":    "این داشبورد فقط جمع می‌زند: جمعیت و جرم حقوقی از پورت کارکنان، تردد/مرخصی/مأموریت/اضافه‌کاری از رکوردهای ماژول حضور. مبلغ قطعی هر کارمند در فیش او مرجع است.",
	}
}

// employeeIndex نمایهٔ «کد کارمند → کارمند» برای نگاشت واحد سازمانی در برش‌ها.
func employeeIndex(ctx context.Context, deps modulekit.Deps) map[string]contracts.Employee {
	index := map[string]contracts.Employee{}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return index
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 500})
	if err != nil {
		return index
	}
	for _, e := range page.Items {
		index[e.ID] = e
	}
	return index
}

// recordDays روزهای یک بازهٔ تاریخ شمسی، شامل هر دو سر. تاریخ نامعتبر صفر می‌شود.
func recordDays(from, to string) float64 {
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

// orgParamInt مقدار عددی یک پارامتر فعال شرکت را برمی‌گرداند (صفر اگر نبود).
func orgParamInt(deps modulekit.Deps, key string) int64 {
	for _, p := range deps.Org.Params {
		if p.Key != key {
			continue
		}
		if f, ok := parseNum(p.Value); ok {
			return int64(f)
		}
	}
	return 0
}

// sortedRows ردیف‌های نگاشت را بر پایهٔ سنجهٔ اصلی نزولی می‌چیند.
func sortedRows(groups map[string]*statRow, measure func(statRow) float64) []statRow {
	rows := make([]statRow, 0, len(groups))
	for _, row := range groups {
		rows = append(rows, *row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if measure(rows[i]) != measure(rows[j]) {
			return measure(rows[i]) > measure(rows[j])
		}
		return rows[i].TitleFa < rows[j].TitleFa
	})
	return rows
}

// sharePct سهم یک مقدار از جمع کل، گردشده به یک رقم اعشار (صفر اگر جمع صفر باشد).
func sharePct(part, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(part/total*1000) / 10
}

// statusLabelFa برچسب فارسی وضعیت‌های رایج رکوردها؛ اگر نبود، خود کلید نمایش داده می‌شود.
func statusLabelFa(status string) (string, bool) {
	labels := map[string]string{
		"draft": "پیش‌نویس", "recorded": "ثبت‌شده", "pending": "در انتظار تأیید",
		"unitApproved": "تأیید مدیر واحد", "approved": "تأییدشده", "confirmed": "تأییدشده",
		"calculated": "محاسبه‌شده", "sent": "ارسال‌شده", "settled": "تسویه‌شده",
		"paid": "پرداخت‌شده", "rejected": "ردشده", "cancelled": "لغوشده", "disputed": "دارای اعتراض",
	}
	label, ok := labels[status]
	return label, ok
}
