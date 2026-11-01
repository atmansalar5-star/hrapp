// پیاده‌سازی ماژول «orgstructure» — «roster»: متصدیان پست‌ها، شرایط احراز و تاریخچه.
//
// چرا این لایه: صفحهٔ ساختار سازمانی باید برای هر پست بگوید «الان چه کسی روی این سمت
// است، عکس کارگزینی‌اش چیست و شرط‌های احراز را احراز کرده یا نه»؛ و برای هر پست،
// سابقهٔ انتصاب‌ها و سابقهٔ تغییرها را نشان دهد. این خواندن ترکیبی است (پست + انتصاب +
// کارمند)، پس در ماژول مالکِ همان داده ساخته می‌شود و رابط فقط نمایش می‌دهد.
//
// قاعده‌ها:
//   - هیچ داده‌ای خوانده نمی‌شود مگر از راه پورت‌های اعلام‌شده: رکوردهای همین ماژول با
//     سرویس رکوردکیت خودش، و دادهٔ کارکنان (نام، واحد، عکس کارگزینی) با پورت
//     `contracts.EmployeeReader`. هیچ import مستقیمی بین ماژول‌ها نیست.
//   - «متصدی جاری» از وضعیت انتصاب می‌آید (`active`)، نه از فیلدی که دستی نگه داشته شود؛
//     پس با پایان تصدی، خودبه‌خود پست بلا تصدی می‌شود و سابقه دست‌نخورده می‌ماند.
//   - تطبیق انتصاب با پست با «کد پست» است و اگر کد در انتصاب خالی باشد، با عنوان پست؛
//     پس دادهٔ قدیمی که فقط عنوان داشته هم جا می‌افتد و گم نمی‌شود.
package orgstructure

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/orgstructure/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// rosterHolder متصدی یک پست همراه عکس کارگزینی و نتیجهٔ احراز شرایط.
type rosterHolder struct {
	AssignmentID    string `json:"assignmentId"`
	AssignmentTitle string `json:"assignmentTitle,omitempty"`
	EmployeeID      string `json:"employeeId"`
	FullName        string `json:"fullName"`
	DepartmentName  string `json:"departmentName,omitempty"`
	// PhotoURL نشانی سرو عکس کارگزینی (امضاشده)؛ خالی یعنی عکس بارگذاری نشده است.
	PhotoURL       string `json:"photoUrl,omitempty"`
	HasPhoto       bool   `json:"hasPhoto"`
	PhotoUpdatedAt string `json:"photoUpdatedAt,omitempty"`
	Qualification  string `json:"qualification"`
	// QualificationFa عنوان فارسی نتیجهٔ احراز از کاتالوگ دامنه.
	QualificationFa string `json:"qualificationFa"`
	Evidence        string `json:"evidence,omitempty"`
	StartDate       string `json:"startDate,omitempty"`
	EndDate         string `json:"endDate,omitempty"`
	DecreeNo        string `json:"decreeNo,omitempty"`
	Status          string `json:"status"`
	StatusFa        string `json:"statusFa"`
}

// rosterPosition یک پست سازمانی با متصدی جاری و شمار انتصاب‌هایش.
type rosterPosition struct {
	ID           string        `json:"id"`
	Code         string        `json:"code,omitempty"`
	Title        string        `json:"title"`
	Unit         string        `json:"unit,omitempty"`
	JobFamily    string        `json:"jobFamily,omitempty"`
	Grade        string        `json:"grade,omitempty"`
	Requirements string        `json:"requirements,omitempty"`
	Headcount    string        `json:"headcount,omitempty"`
	Status       string        `json:"status"`
	StatusFa     string        `json:"statusFa"`
	Holder       *rosterHolder `json:"holder,omitempty"`
	// HolderCount شمار متصدیان فعال (بیش از یک یعنی دادهٔ ناسازگار و هشدار).
	HolderCount int `json:"holderCount"`
	// AssignmentCount شمار همهٔ انتصاب‌های این پست (جاری + پیشین).
	AssignmentCount int `json:"assignmentCount"`
	// HasPhoto عکس کارگزینی متصدی جاری بارگذاری شده است.
	HasPhoto bool `json:"hasPhoto"`
	// RequirementsMet نتیجهٔ احراز متصدی جاری «احرازشده» است.
	RequirementsMet bool `json:"requirementsMet"`
}

// rosterUnit گرهٔ واحد سازمانی برای نمودار درختی.
type rosterUnit struct {
	Unit      string `json:"unit"`
	Positions int    `json:"positions"`
	Filled    int    `json:"filled"`
	Vacant    int    `json:"vacant"`
	Headcount int    `json:"headcount"`
}

// rosterSummary جمع‌بندی ساختار سازمانی.
type rosterSummary struct {
	Positions    int `json:"positions"`
	Filled       int `json:"filled"`
	Vacant       int `json:"vacant"`
	Unverified   int `json:"unverified"`
	WithoutPhoto int `json:"withoutPhoto"`
	Assignments  int `json:"assignments"`
	Active       int `json:"active"`
	Historical   int `json:"historical"`
}

// rosterView پاسخ صفحهٔ ساختار سازمانی.
type rosterView struct {
	Module      string             `json:"module"`
	TitleFa     string             `json:"moduleTitleFa"`
	Positions   []rosterPosition   `json:"positions"`
	Units       []rosterUnit       `json:"units"`
	Assignments []contracts.Record `json:"assignments"`
	Summary     rosterSummary      `json:"summary"`
	// QualificationOptions فهرست نتیجه‌های احراز (برچسب فارسی از دامنه).
	QualificationOptions []domain.Qualification `json:"qualificationOptions"`
	Warnings             []string               `json:"warnings,omitempty"`
	NoteFa               string                 `json:"noteFa"`
}

// timelineEntry یک سطر «سابقهٔ تغییرات» سمت سازمانی.
type timelineEntry struct {
	At       string `json:"at"`
	Date     string `json:"date"`
	Source   string `json:"source"`
	SourceFa string `json:"sourceFa"`
	RecordID string `json:"recordId"`
	Action   string `json:"action"`
	ActionFa string `json:"actionFa"`
	Actor    string `json:"actor,omitempty"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// positionHistory پاسخ «سابقهٔ انتصابات و تغییرات» یک پست.
type positionHistory struct {
	Position    rosterPosition     `json:"position"`
	Assignments []contracts.Record `json:"assignments"`
	Timeline    []timelineEntry    `json:"timeline"`
	Summary     rosterSummary      `json:"summary"`
	NoteFa      string             `json:"noteFa"`
}

// registerRosterEndpoints مسیرهای ترکیبی ساختار سازمانی را ثبت می‌کند.
func registerRosterEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	if svc == nil {
		return
	}
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("orgstructure", "read", fn) }

	// ساختار سازمانی: پست‌ها + متصدی جاری + عکس کارگزینی + نتیجهٔ احراز + فهرست انتصاب‌ها.
	mux.HandleFunc("GET /api/v1/orgstructure/roster", read(func(w http.ResponseWriter, r *http.Request) {
		view, err := buildRoster(r, deps, svc)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, view)
	}))

	// سابقهٔ انتصاب‌ها و تاریخچهٔ تغییرات یک پست سازمانی.
	mux.HandleFunc("GET /api/v1/orgstructure/positions/{id}/history", read(func(w http.ResponseWriter, r *http.Request) {
		history, ok, err := buildPositionHistory(r, deps, svc, r.PathValue("id"))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "پست سازمانی با این شناسه پیدا نشد", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, history)
	}))

	// گزینه‌های فرم انتصاب (نتیجهٔ احراز، کد پست‌ها و واحدها) از خود سرور.
	mux.HandleFunc("GET /api/v1/orgstructure/options", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, buildRosterOptions(r, deps, svc))
	}))
}

// buildRoster ساختار سازمانی را از پست‌ها، انتصاب‌ها و پروندهٔ کارکنان می‌سازد.
func buildRoster(r *http.Request, deps modulekit.Deps, svc *recordkit.Service) (rosterView, error) {
	positions, err := listRecords(r, svc, domain.KindPositions)
	if err != nil {
		return rosterView{}, err
	}
	assignments, err := listRecords(r, svc, domain.KindAssignments)
	if err != nil {
		return rosterView{}, err
	}

	view := rosterView{
		Module:               svc.Module(),
		TitleFa:              svc.Schema().ModuleTitleFa,
		Positions:            []rosterPosition{},
		Units:                []rosterUnit{},
		Assignments:          assignments,
		QualificationOptions: domain.Qualifications(),
		NoteFa: "متصدی هر پست از آخرین انتصاب در وضعیت «در حال تصدی» خوانده می‌شود؛ " +
			"انتصاب‌های پایان‌یافته پاک نمی‌شوند و در «سابقهٔ انتصابات» همان پست می‌مانند.",
	}

	// متصدیان فعال به تفکیک پست (کد پست، و در نبود کد، عنوان پست).
	activeByPosition := map[string][]contracts.Record{}
	for _, a := range assignments {
		if a.Status != domain.AssignmentActive {
			continue
		}
		for _, key := range positionKeys(a) {
			activeByPosition[key] = append(activeByPosition[key], a)
		}
	}

	// عکس و مشخصات کارمند از پورت قرارداد خوانده می‌شود.
	employees := employeeLookup{r: r, reader: deps.Ports}
	byUnit := map[string]*rosterUnit{}
	for _, p := range positions {
		code := strings.TrimSpace(p.Fields["code"])
		unit := strings.TrimSpace(p.Fields["unit"])
		item := rosterPosition{
			ID: p.ID, Code: code, Title: p.Title, Unit: unit,
			JobFamily: p.Fields["jobFamily"], Grade: p.Fields["grade"],
			Requirements: p.Fields["requirements"], Headcount: p.Fields["headcount"],
			Status: p.Status, StatusFa: p.StatusFa,
		}

		holders := []contracts.Record{}
		if code != "" {
			holders = activeByPosition[code]
		}
		if len(holders) == 0 {
			holders = activeByPosition[p.Title]
		}
		item.HolderCount = len(holders)
		item.AssignmentCount = countAssignments(assignments, p)
		if len(holders) > 0 {
			holder := toHolder(holders[0], &employees)
			item.Holder = &holder
			item.HasPhoto = holder.HasPhoto
			item.RequirementsMet = holder.Qualification == domain.QualificationMet
		}
		view.Positions = append(view.Positions, item)

		key := unit
		if key == "" {
			key = "بدون واحد"
		}
		group, ok := byUnit[key]
		if !ok {
			group = &rosterUnit{Unit: key}
			byUnit[key] = group
		}
		group.Positions++
		if len(holders) > 0 {
			group.Filled++
		} else {
			group.Vacant++
		}
		if n := atoiOrZero(p.Fields["headcount"]); n > 0 {
			group.Headcount += n
		}
	}

	view.Units = sortedUnits(byUnit)
	view.Summary = summarize(positions, assignments, view.Positions)
	view.Warnings = rosterWarnings(view)
	return view, nil
}

// buildPositionHistory سابقهٔ انتصاب‌های یک پست و تاریخچهٔ تغییرات آن را می‌سازد.
func buildPositionHistory(r *http.Request, deps modulekit.Deps, svc *recordkit.Service, id string) (positionHistory, bool, error) {
	rec, ok, err := svc.Get(r.Context(), contracts.RecordFilter{
		Module: svc.Module(), Kind: domain.KindPositions, Scope: "all",
	}, strings.TrimSpace(id))
	if err != nil || !ok || rec.Kind != domain.KindPositions {
		return positionHistory{}, false, err
	}
	assignments, err := listRecords(r, svc, domain.KindAssignments)
	if err != nil {
		return positionHistory{}, false, err
	}
	employees := employeeLookup{r: r, reader: deps.Ports}

	var mine []contracts.Record
	for _, a := range assignments {
		if assignmentMatchesPosition(a, rec) {
			mine = append(mine, a)
		}
	}
	sort.SliceStable(mine, func(i, j int) bool {
		return strings.TrimSpace(mine[i].Fields["startDate"]) > strings.TrimSpace(mine[j].Fields["startDate"])
	})

	history := positionHistory{
		Position: rosterPosition{
			ID: rec.ID, Code: rec.Fields["code"], Title: rec.Title, Unit: rec.Fields["unit"],
			JobFamily: rec.Fields["jobFamily"], Grade: rec.Fields["grade"],
			Requirements: rec.Fields["requirements"], Headcount: rec.Fields["headcount"],
			Status: rec.Status, StatusFa: rec.StatusFa,
			AssignmentCount: len(mine),
		},
		Assignments: mine,
		Timeline:    []timelineEntry{},
		NoteFa: "تاریخچه از دو جا ساخته می‌شود: تغییرهای خودِ شناسنامهٔ پست (ثبت، ویرایش، تغییر وضعیت) " +
			"و رخدادهای هر انتصاب (ثبت انتصاب، آغاز تصدی، پایان تصدی). هیچ سطری حذف نمی‌شود.",
	}

	for _, a := range mine {
		if a.Status == domain.AssignmentActive {
			history.Position.HolderCount++
		}
	}
	history.Position.Holder = firstActiveHolder(mine, &employees)
	if history.Position.Holder != nil {
		history.Position.HasPhoto = history.Position.Holder.HasPhoto
		history.Position.RequirementsMet = history.Position.Holder.Qualification == domain.QualificationMet
	}

	// رخدادهای شناسنامهٔ پست.
	for _, e := range rec.Events {
		history.Timeline = append(history.Timeline, timelineEntry{
			At: e.At, Date: calendar.FormatStamp(e.At),
			Source: "position", SourceFa: "شناسنامهٔ پست", RecordID: rec.ID,
			Action: e.Action, ActionFa: e.ActionFa, Actor: e.Actor, NoteFa: e.NoteFa,
		})
	}
	// رخدادهای هر انتصاب؛ با تاریخ مؤثر انتصاب تا چیدن زمانی با تاریخ شروع هم‌خوان باشد.
	for _, a := range mine {
		for _, e := range a.Events {
			history.Timeline = append(history.Timeline, timelineEntry{
				At: e.At, Date: calendar.FormatStamp(e.At),
				Source: "assignment", SourceFa: "انتصاب", RecordID: a.ID,
				Action: e.Action, ActionFa: e.ActionFa, Actor: e.Actor, NoteFa: e.NoteFa,
			})
		}
	}
	// نوترین بالا: بر پایهٔ تاریخ مؤثر انتصاب و سپس زمان رخداد.
	sort.SliceStable(history.Timeline, func(i, j int) bool {
		if history.Timeline[i].Date != history.Timeline[j].Date {
			return history.Timeline[i].Date > history.Timeline[j].Date
		}
		return history.Timeline[i].At > history.Timeline[j].At
	})
	history.Summary = summarize(nil, mine, []rosterPosition{history.Position})
	return history, true, nil
}

// buildRosterOptions گزینه‌های فرم انتصاب را می‌سازد.
func buildRosterOptions(r *http.Request, deps modulekit.Deps, svc *recordkit.Service) map[string]any {
	positions, _ := listRecords(r, svc, domain.KindPositions)
	units := []map[string]string{}
	seen := map[string]bool{}
	positionOptions := make([]map[string]string, 0, len(positions))
	for _, p := range positions {
		code := strings.TrimSpace(p.Fields["code"])
		if code != "" {
			positionOptions = append(positionOptions, map[string]string{
				"key": code, "titleFa": p.Title + " — " + code,
			})
		}
		unit := strings.TrimSpace(p.Fields["unit"])
		if unit != "" && !seen[unit] {
			seen[unit] = true
			units = append(units, map[string]string{"key": unit, "titleFa": unit})
		}
	}
	return map[string]any{
		"module":                svc.Module(),
		"qualifications":        domain.Qualifications(),
		"positions":             positionOptions,
		"units":                 units,
		"assignmentKind":        domain.KindAssignments,
		"assignmentKindTitleFa": "انتصاب پست سازمانی",
		"noteFa": "فهرست نتیجهٔ احراز، کد پست‌ها و واحدها از خود سرور می‌آید؛ هیچ گزینه‌ای در رابط ساخته نمی‌شود و " +
			"قاعدهٔ «انتصاب بدون احراز شرایط مجاز نیست» در دامنهٔ سرور اجرا می‌شود.",
	}
}

/* ------------------------------ کمکی‌ها ------------------------------ */

// listRecords رکوردهای یک نوع را (تا سقف پشتیبانی‌شدهٔ انبار) می‌خواند.
func listRecords(r *http.Request, svc *recordkit.Service, kind string) ([]contracts.Record, error) {
	page, err := svc.List(r.Context(), contracts.RecordFilter{
		Module: svc.Module(), Kind: kind, Scope: "all", Size: 500,
	})
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// positionKeys کلیدهای تطبیق یک انتصاب با پست (کد پست و عنوان پست).
func positionKeys(a contracts.Record) []string {
	out := make([]string, 0, 2)
	if code := strings.TrimSpace(a.Fields["position"]); code != "" {
		out = append(out, code)
	}
	if title := strings.TrimSpace(a.Fields["positionTitle"]); title != "" {
		out = append(out, title)
	}
	if len(out) == 0 {
		out = append(out, strings.TrimSpace(a.Title))
	}
	return out
}

// assignmentMatchesPosition آیا این انتصاب به همین پست تعلق دارد.
func assignmentMatchesPosition(a contracts.Record, p contracts.Record) bool {
	code := strings.TrimSpace(p.Fields["code"])
	for _, key := range positionKeys(a) {
		if code != "" && key == code {
			return true
		}
		if key == p.Title && p.Title != "" {
			return true
		}
	}
	return false
}

// countAssignments شمار انتصاب‌های یک پست.
func countAssignments(assignments []contracts.Record, p contracts.Record) int {
	n := 0
	for _, a := range assignments {
		if assignmentMatchesPosition(a, p) {
			n++
		}
	}
	return n
}

// toHolder تبدیل یک انتصاب به «متصدی» با عکس و نتیجهٔ احراز.
func toHolder(a contracts.Record, employees *employeeLookup) rosterHolder {
	holder := rosterHolder{
		AssignmentID:    a.ID,
		AssignmentTitle: a.Title,
		EmployeeID:      a.EmployeeID,
		FullName:        a.EmployeeName,
		DepartmentName:  a.DepartmentName,
		Qualification:   strings.TrimSpace(a.Fields["qualification"]),
		Evidence:        strings.TrimSpace(a.Fields["evidence"]),
		StartDate:       strings.TrimSpace(a.Fields["startDate"]),
		EndDate:         strings.TrimSpace(a.Fields["endDate"]),
		DecreeNo:        strings.TrimSpace(a.Fields["decreeNo"]),
		Status:          a.Status,
		StatusFa:        a.StatusFa,
	}
	holder.QualificationFa = domain.QualificationTitleFa(holder.Qualification)
	if employee, ok := employees.get(a.EmployeeID); ok {
		if holder.FullName == "" {
			holder.FullName = employee.FullName
		}
		if holder.DepartmentName == "" {
			holder.DepartmentName = employee.DepartmentName
		}
		holder.PhotoURL = employee.PhotoURL
		holder.PhotoUpdatedAt = employee.PhotoUpdatedAt
		holder.HasPhoto = employee.PhotoURL != ""
	}
	return holder
}

// firstActiveHolder نوترین انتصاب «در حال تصدی» یک پست را به «متصدی» تبدیل می‌کند.
func firstActiveHolder(assignments []contracts.Record, employees *employeeLookup) *rosterHolder {
	var best *contracts.Record
	for i := range assignments {
		a := assignments[i]
		if a.Status != domain.AssignmentActive {
			continue
		}
		if best == nil || strings.TrimSpace(a.Fields["startDate"]) > strings.TrimSpace(best.Fields["startDate"]) {
			best = &assignments[i]
		}
	}
	if best == nil {
		return nil
	}
	holder := toHolder(*best, employees)
	return &holder
}

// employeeLookup خواندن کارمند از پورت قرارداد (با حافظهٔ کوچک برای پرهیز از تکرار).
type employeeLookup struct {
	r      *http.Request
	reader *modulekit.Ports
	cache  map[string]contracts.Employee
}

// get کارمند را می‌خواند؛ نبودِ پورت یا خطا یعنی «مشخصات تازه در دسترس نیست» و سطر با
// دادهٔ خودِ رکورد انتصاب نشان داده می‌شود (بدون رقم یا نام ساختگی).
func (l *employeeLookup) get(employeeID string) (contracts.Employee, bool) {
	id := strings.TrimSpace(employeeID)
	if id == "" || l.reader == nil {
		return contracts.Employee{}, false
	}
	if l.cache == nil {
		l.cache = map[string]contracts.Employee{}
	}
	if employee, ok := l.cache[id]; ok {
		return employee, true
	}
	reader, ok := l.reader.Employees()
	if !ok {
		return contracts.Employee{}, false
	}
	employee, found, err := reader.Get(l.r.Context(), contracts.EmployeeFilter{Scope: "all"}, id)
	if err != nil || !found {
		return contracts.Employee{}, false
	}
	l.cache[id] = employee
	return employee, true
}

// summarize جمع‌بندی ساختار سازمانی.
func summarize(positions, assignments []contracts.Record, rows []rosterPosition) rosterSummary {
	summary := rosterSummary{Assignments: len(assignments)}
	for _, a := range assignments {
		if a.Status == domain.AssignmentActive {
			summary.Active++
		} else {
			summary.Historical++
		}
	}
	summary.Positions = len(rows)
	for _, row := range rows {
		switch {
		case row.Holder != nil:
			summary.Filled++
		default:
			summary.Vacant++
		}
		if row.Holder != nil && !row.RequirementsMet {
			summary.Unverified++
		}
		if row.Holder != nil && !row.HasPhoto {
			summary.WithoutPhoto++
		}
	}
	if len(positions) > 0 && summary.Positions == 0 {
		summary.Positions = len(positions)
	}
	return summary
}

// rosterWarnings هشدارهای صادقانه دربارهٔ داده‌ای که کاربر باید ببیند.
func rosterWarnings(view rosterView) []string {
	warnings := []string{}
	multi := []string{}
	unverified := []string{}
	for _, p := range view.Positions {
		if p.HolderCount > 1 {
			multi = append(multi, p.Title)
		}
		if p.Holder != nil && p.Holder.Qualification != "" && !p.RequirementsMet {
			unverified = append(unverified, p.Holder.FullName+" ("+p.Title+")")
		}
	}
	if len(multi) > 0 {
		warnings = append(warnings, "برای این پست‌ها بیش از یک انتصاب «در حال تصدی» ثبت شده است: "+
			strings.Join(multi, "، ")+"؛ پیش از گزارش‌گیری، انتصاب پیشین را «پایان تصدی» کنید.")
	}
	if len(unverified) > 0 {
		warnings = append(warnings, "شرایط احراز این متصدیان کامل احراز نشده است: "+
			strings.Join(unverified, "، ")+"؛ وضعیت احراز نباید انتصاب را متوقف کند، ولی باید پیگیری شود.")
	}
	if view.Summary.WithoutPhoto > 0 {
		warnings = append(warnings, "برای "+itoa(view.Summary.WithoutPhoto)+" متصدی، عکس کارگزینی بارگذاری نشده است؛ "+
			"از پروندهٔ پرسنلی همان کارمند، عکس رسمی را بارگذاری کنید.")
	}
	return warnings
}

// sortedUnits واحدها را به ترتیب نام برمی‌گرداند (ترتیب پایدار نمودار).
func sortedUnits(byUnit map[string]*rosterUnit) []rosterUnit {
	out := make([]rosterUnit, 0, len(byUnit))
	for _, unit := range byUnit {
		out = append(out, *unit)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Positions != out[j].Positions {
			return out[i].Positions > out[j].Positions
		}
		return out[i].Unit < out[j].Unit
	})
	return out
}

// atoiOrZero عدد سادهٔ فیلد متنی (بدون خطای پیچیده).
func atoiOrZero(raw string) int {
	value := 0
	for _, ch := range calendar.NormalizeDigits(strings.TrimSpace(raw)) {
		if ch < '0' || ch > '9' {
			return value
		}
		value = value*10 + int(ch-'0')
	}
	return value
}

// itoa عدد را به رشتهٔ فارسی برمی‌گرداند.
//
// چرا فارسی: این عدد داخل پیام فارسی کاربر می‌نشیند («برای ۳ متصدی...») و قاعدهٔ پروژه
// این است که در متنِ کاربر ارقام فارسی دیده شود، نه لاتین.
func itoa(n int) string {
	return calendar.ToPersianDigits(strconv.Itoa(n))
}
