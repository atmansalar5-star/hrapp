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
	"hrms/internal/rules"
)

// holderEducation اطلاعات تحصیلی متصدی (آخرین مدرک) برای نمایش در ساختار سازمانی.
type holderEducation struct {
	RecordID       string `json:"recordId,omitempty"`
	Level          string `json:"level,omitempty"`
	LevelFa        string `json:"levelFa,omitempty"`
	Field          string `json:"field,omitempty"`
	Orientation    string `json:"orientation,omitempty"`
	University     string `json:"university,omitempty"`
	GraduationDate string `json:"graduationDate,omitempty"`
	// RelevanceFa نتیجهٔ بررسی «مرتبط یا غیرمرتبط بودن تحصیلات» و درصد ارتباط.
	RelevanceFa      string `json:"relevanceFa,omitempty"`
	Relevance        string `json:"relevance,omitempty"`
	RelevancePercent string `json:"relevancePercent,omitempty"`
	CouncilStatusFa  string `json:"councilStatusFa,omitempty"`
	CouncilNo        string `json:"councilNo,omitempty"`
	CouncilDate      string `json:"councilDate,omitempty"`
	StatusFa         string `json:"statusFa,omitempty"`
}

// holderDegreeBasis «مدرک ملاک حقوق» متصدی: کدام مدرک، با چه صورت‌جلسه‌ای و با چه مقطعی
// مبنای گروه شغلی و پرداخت است (برای ایثارگران یک مقطع بالاتر).
type holderDegreeBasis struct {
	RecordID         string `json:"recordId,omitempty"`
	LevelFa          string `json:"levelFa,omitempty"`
	EffectiveLevelFa string `json:"effectiveLevelFa,omitempty"`
	Field            string `json:"field,omitempty"`
	Relevance        string `json:"relevance,omitempty"`
	RelevanceFa      string `json:"relevanceFa,omitempty"`
	RelevancePct     string `json:"relevancePercent,omitempty"`
	CouncilNo        string `json:"councilNo,omitempty"`
	CouncilDate      string `json:"councilDate,omitempty"`
	EffectiveDate    string `json:"effectiveDate,omitempty"`
	SalaryGroup      string `json:"salaryGroup,omitempty"`
	VeteranUpgrade   bool   `json:"veteranUpgrade"`
	UpgradedLevelFa  string `json:"upgradedLevelFa,omitempty"`
	StatusFa         string `json:"statusFa,omitempty"`
}

// holderPersonalBase پایهٔ شخصی متصدی: از حکم ابلاغ‌شدهٔ خودِ کارمند خوانده می‌شود
// (گروه شغلی حکم و حقوق پایه)، نه از فیلدی که ساختار سازمانی نگه دارد.
type holderPersonalBase struct {
	DecreeID      string `json:"decreeId,omitempty"`
	JobGroup      string `json:"jobGroup,omitempty"`
	BaseSalary    int64  `json:"baseSalary,omitempty"`
	MazdBase      int64  `json:"mazdBase,omitempty"`
	EffectiveDate string `json:"effectiveDate,omitempty"`
	StatusFa      string `json:"statusFa,omitempty"`
	NoteFa        string `json:"noteFa,omitempty"`
}

// holderOrgBase پایهٔ سازمانی سمت: از شناسنامهٔ خودِ پست سازمانی.
type holderOrgBase struct {
	Grade     string `json:"grade,omitempty"`
	JobFamily string `json:"jobFamily,omitempty"`
	Headcount string `json:"headcount,omitempty"`
	// Requirements شرط‌های احراز پست (مقطع و رشتهٔ مورد نیاز).
	Requirements string `json:"requirements,omitempty"`
}

// rosterHolder متصدی یک پست همراه عکس کارگزینی، اطلاع کسب‌وکار و نتیجهٔ احراز شرایط.
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
	// Education آخرین مدرک تحصیلی متصدی (مقطع، رشته، گرایش و ارتباط با شغل).
	Education *holderEducation `json:"education,omitempty"`
	// DegreeBasis مدرک ملاک حقوق متصدی (اگر شورای تعدیل مدرک تعیین کرده باشد).
	DegreeBasis *holderDegreeBasis `json:"degreeBasis,omitempty"`
	// PersonalGradeCode کد پایهٔ شخصی فرد (103.xxxxxxxxxx) که از شمارهٔ پرسنلی ساخته میشود.
	PersonalGradeCode string `json:"personalGradeCode,omitempty"`
	// AssignmentCode شمارهٔ انتصاب (084.xxxxxxxxxx) از شناسهٔ رکورد انتصاب.
	AssignmentCode string `json:"assignmentCode,omitempty"`
	// PersonalBase پایهٔ شخصی فرد از حکم جاری، و OrgBase پایهٔ سازمانی سمت از پست.
	PersonalBase *holderPersonalBase `json:"personalBase,omitempty"`
	OrgBase      *holderOrgBase      `json:"orgBase,omitempty"`
	// Veteran وضعیت ایثارگری متصدی (مبنای معادل‌سازی یک مقطع بالاتر).
	Veteran       bool   `json:"veteran"`
	VeteranTypeFa string `json:"veteranTypeFa,omitempty"`
	VeteranNoteFa string `json:"veteranNoteFa,omitempty"`
}

// rosterPosition یک پست سازمانی با متصدی جاری و شمار انتصاب‌هایش.
type rosterPosition struct {
	ID   string `json:"id"`
	Code string `json:"code,omitempty"`
	// PostCode شمارهٔ سمت و OrgGradeCode پایهٔ سمت سازمانی — قالب rules/orgcode.go.
	PostCode      string        `json:"postCode,omitempty"`
	OrgGradeCode  string        `json:"orgGradeCode,omitempty"`
	UnitCode      string        `json:"unitCode,omitempty"`
	JobFamilyCode string        `json:"jobFamilyCode,omitempty"`
	Title         string        `json:"title"`
	Unit          string        `json:"unit,omitempty"`
	JobFamily     string        `json:"jobFamily,omitempty"`
	Grade         string        `json:"grade,omitempty"`
	Requirements  string        `json:"requirements,omitempty"`
	Headcount     string        `json:"headcount,omitempty"`
	Status        string        `json:"status"`
	StatusFa      string        `json:"statusFa"`
	Holder        *rosterHolder `json:"holder,omitempty"`
	// OrgBase پایهٔ سازمانی خودِ سمت (گروه شغلی، خانوادهٔ شغلی و شرط‌های احراز پست).
	OrgBase *holderOrgBase `json:"orgBase,omitempty"`
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

// treeUnit یک گرهٔ درخت سازمانی: واحد، سرپرست، شمار نیرو و پست‌های همان واحد با متصدی.
type treeUnit struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Head      string           `json:"head,omitempty"`
	ParentID  string           `json:"parentId,omitempty"`
	Headcount int              `json:"headcount"`
	Positions []rosterPosition `json:"positions"`
	Filled    int              `json:"filled"`
	Vacant    int              `json:"vacant"`
	// EmployeeCount شمار کارکنان همان واحد از پروندهٔ کارکنان (برای درخت).
	EmployeeCount int `json:"employeeCount"`
}

// treeView دادهٔ «درخت سازمانی»: ریشهٔ هلدینگ/سازمان، واحدهای سلسله‌مراتبی و پست‌های هر
// واحد همراه متصدی (با عکس و تحصیلات)، و در پایان پست‌های بدون واحد مشخص.
type treeView struct {
	Root       string           `json:"root"`
	Units      []treeUnit       `json:"units"`
	Unassigned []rosterPosition `json:"unassigned,omitempty"`
	Totals     map[string]int   `json:"totals"`
	NoteFa     string           `json:"noteFa"`
}

// rosterView پاسخ صفحهٔ ساختار سازمانی.
type rosterView struct {
	Module      string             `json:"module"`
	TitleFa     string             `json:"moduleTitleFa"`
	Positions   []rosterPosition   `json:"positions"`
	Units       []rosterUnit       `json:"units"`
	Tree        treeView           `json:"tree"`
	Assignments []contracts.Record `json:"assignments"`
	Summary     rosterSummary      `json:"summary"`
	// QualificationOptions فهرست نتیجه‌های احراز (برچسب فارسی از دامنه).
	QualificationOptions []domain.Qualification `json:"qualificationOptions"`
	// CodeScheme کارت مقادیر مجاز کدها: شش نوع کد با پیش‌شمارهٔ مؤثر همین استقرار و
	// توضیح فارسی هر کدام (برای نمایش در نمودار سازمانی و سند راهنما).
	CodeScheme []rules.OrgCodeKind `json:"codeScheme"`
	Warnings   []string            `json:"warnings,omitempty"`
	NoteFa     string              `json:"noteFa"`
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

	// عکس و مشخصات کارمند از پورت قرارداد خوانده می‌شود؛ تحصیلات، مدرک ملاک، حکم جاری
	// و پروندهٔ ایثارگری هم از پورت‌های خودِ ماژول‌های مالک (یک گذر برای همهٔ متصدیان).
	employees := employeeLookup{r: r, reader: deps.Ports}
	facts := collectHolderFacts(r, deps, activeEmployees(assignments))
	scheme := orgCodeScheme(deps)
	byUnit := map[string]*rosterUnit{}
	for _, p := range positions {
		code := strings.TrimSpace(p.Fields["code"])
		unit := strings.TrimSpace(p.Fields["unit"])
		item := rosterPosition{
			ID: p.ID, Code: code, Title: p.Title, Unit: unit,
			PostCode:      domain.PostCodeOf(scheme, p.Fields, p.ID),
			OrgGradeCode:  domain.OrgGradeCodeOf(scheme, p.Fields, p.Fields["grade"]),
			UnitCode:      domain.UnitCodeOf(scheme, p.Fields, p.ID),
			JobFamilyCode: domain.JobFamilyCodeOf(scheme, p.Fields),
			JobFamily:     p.Fields["jobFamily"], Grade: p.Fields["grade"],
			Requirements: p.Fields["requirements"], Headcount: p.Fields["headcount"],
			Status: p.Status, StatusFa: p.StatusFa,
			// پایهٔ سازمانی خودِ سمت از شناسنامهٔ پست (مستقل از متصدی).
			OrgBase: &holderOrgBase{
				Grade:        strings.TrimSpace(p.Fields["grade"]),
				JobFamily:    strings.TrimSpace(p.Fields["jobFamily"]),
				Headcount:    strings.TrimSpace(p.Fields["headcount"]),
				Requirements: strings.TrimSpace(p.Fields["requirements"]),
			},
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
			holder := toHolder(scheme, holders[0], &employees, facts, p)
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
	view.Tree = buildTree(r, deps, view.Positions)
	view.CodeScheme = scheme.View()
	view.Summary = summarize(positions, assignments, view.Positions)
	view.Warnings = rosterWarnings(view)
	return view, nil
}

// activeEmployees شناسهٔ کارمندانی که انتصاب «در حال تصدی» دارند (بدون تکرار).
func activeEmployees(assignments []contracts.Record) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, a := range assignments {
		if a.Status != domain.AssignmentActive {
			continue
		}
		id := strings.TrimSpace(a.EmployeeID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// buildTree درخت سازمانی را می‌سازد: ریشهٔ سازمان، واحدهای سلسله‌مراتبی و پست‌های هر
// واحد همراه متصدی (با عکس، تحصیلات، پایهٔ شخصی و پایهٔ سازمانی سمت).
//
// منبع واحدها پورت خودِ ماژول کارکنان است (`EmployeeReader.Departments`) تا فهرست واحد
// در ساختار سازمانی تکرار نشود. تطبیق پست با واحد با نام واحد انجام می‌شود و اگر پستی به
// هیچ واحدی نخورد، در «بدون واحد سازمانی» می‌آید تا از درخت نیفتد.
func buildTree(r *http.Request, deps modulekit.Deps, positions []rosterPosition) treeView {
	root := deps.Org.TitleFa
	if root == "" {
		root = "سازمان"
	}
	view := treeView{
		Root:       root,
		Units:      []treeUnit{},
		Unassigned: []rosterPosition{},
		Totals:     map[string]int{},
		NoteFa: "درخت از واحدهای سازمانی پروندهٔ کارکنان ساخته می‌شود و پست‌های هر واحد — با " +
			"متصدی، عکس کارگزینی، آخرین مدرک تحصیلی، پایهٔ شخصی فرد و پایهٔ سازمانی سمت — زیر " +
			"همان واحد می‌آیند. جدول تفصیلی هر واحد در ادامهٔ همین صفحه است.",
	}

	departments := []contracts.Department{}
	if reader, ok := deps.Ports.Employees(); ok {
		if list, err := reader.Departments(r.Context()); err == nil {
			departments = list
		}
	}
	byID := map[string]*treeUnit{}
	order := []string{}
	for _, dept := range departments {
		// هر گره روی heap ساخته می‌شود و در پایان داخل فهرست کپی می‌شود: اگر اشاره‌گر
		// داخل فهرست گرفته شود، با رشد فهرست باطل می‌شود و پست‌ها به واحد درست نمی‌چسبند.
		byID[dept.ID] = &treeUnit{
			ID: dept.ID, Name: dept.Name, Head: dept.Head, ParentID: dept.ParentID,
			EmployeeCount: dept.Headcount, Positions: []rosterPosition{},
		}
		order = append(order, dept.ID)
	}

	// پست‌ها به واحد خود می‌چسبند؛ اول با نام واحد پست و در نبود تطبیق، با واحد متصدی.
	for _, position := range positions {
		target := matchUnit(byID, position.Unit)
		if target == nil && position.Holder != nil {
			target = matchUnit(byID, position.Holder.DepartmentName)
		}
		if target == nil {
			view.Unassigned = append(view.Unassigned, position)
			continue
		}
		target.Positions = append(target.Positions, position)
		if position.Holder != nil {
			target.Filled++
		} else {
			target.Vacant++
		}
	}

	// فهرست نهایی واحدها با ترتیب پروندهٔ کارکنان ساخته می‌شود.
	for _, id := range order {
		view.Units = append(view.Units, *byID[id])
	}
	// شمارش جمع‌بندی درخت.
	for _, unit := range view.Units {
		view.Totals["positions"] += len(unit.Positions)
		view.Totals["filled"] += unit.Filled
		view.Totals["vacant"] += unit.Vacant
	}
	view.Totals["unassigned"] = len(view.Unassigned)
	view.Totals["units"] = len(view.Units)
	return view
}

// matchUnit واحد متناظر یک نام را در درخت پیدا می‌کند.
//
// چرا تطبیق نسبی: نام واحد در شناسنامهٔ پست دستی نوشته می‌شود («امور مالی») و نام واحد
// سازمانی از پروندهٔ کارکنان می‌آید («مالی و حسابداری»)؛ تطبیق دقیق، پست‌ها را از درخت
// می‌انداخت. پس اول تطبیق دقیق، سپس هم‌ریشگی واژه‌ای سنجیده می‌شود و هیچ واحد تازه‌ای
// ساخته نمی‌شود.
func matchUnit(units map[string]*treeUnit, name string) *treeUnit {
	value := normalizeUnitName(name)
	if value == "" {
		return nil
	}
	for _, unit := range units {
		if normalizeUnitName(unit.Name) == value {
			return unit
		}
	}
	for _, unit := range units {
		if strings.Contains(normalizeUnitName(unit.Name), value) || strings.Contains(value, normalizeUnitName(unit.Name)) {
			return unit
		}
	}
	// هم‌ریشگی واژه‌ای: «امور مالی» و «مالی و حسابداری» واژهٔ «مالی» را share می‌کنند.
	for _, unit := range units {
		for _, word := range unitWords(value) {
			for _, other := range unitWords(normalizeUnitName(unit.Name)) {
				if word == other {
					return unit
				}
			}
		}
	}
	return nil
}

// normalizeUnitName نام واحد را برای مقایسه یکدست می‌کند.
func normalizeUnitName(name string) string {
	value := strings.ReplaceAll(strings.TrimSpace(name), "\u200c", " ")
	value = strings.ReplaceAll(value, "ي", "ی")
	value = strings.ReplaceAll(value, "ك", "ک")
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

// unitWords واژه‌های معنادار یک نام واحد (واژه‌های کوتاه و «و» حذف می‌شوند).
func unitWords(name string) []string {
	out := []string{}
	for _, word := range strings.Fields(name) {
		if len([]rune(word)) < 3 || word == "و" {
			continue
		}
		out = append(out, word)
	}
	return out
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
	// اطلاعات پایهٔ متصدی این پست (تحصیلات، مدرک ملاک، پایهٔ شخصی، ایثارگری).
	facts := collectHolderFacts(r, deps, activeEmployees(mine))
	history.Position.OrgBase = &holderOrgBase{
		Grade:        rec.Fields["grade"],
		JobFamily:    rec.Fields["jobFamily"],
		Headcount:    rec.Fields["headcount"],
		Requirements: rec.Fields["requirements"],
	}
	history.Position.Holder = firstActiveHolder(orgCodeScheme(deps), mine, &employees, facts)
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

// toHolder تبدیل یک انتصاب به «متصدی» با عکس، اطلاع تحصیلی، مدرک ملاک و پایه‌ها.
//
// position شناسنامهٔ پستی است که این متصدی روی آن نشسته؛ «پایهٔ سازمانی» از خودِ پست
// خوانده می‌شود تا با گروه شغلی شخصی اشتباه نشود.
func toHolder(scheme rules.OrgCodeScheme, a contracts.Record, employees *employeeLookup, facts holderFacts, position contracts.Record) rosterHolder {
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
	holder.PersonalGradeCode = domain.PersonalGradeCodeOf(scheme, a.EmployeeID)
	holder.AssignmentCode = domain.AssignmentCodeOf(scheme, a.Fields, a.ID)
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
	// پایهٔ سازمانی سمت از شناسنامهٔ پست.
	holder.OrgBase = &holderOrgBase{
		Grade:        strings.TrimSpace(position.Fields["grade"]),
		JobFamily:    strings.TrimSpace(position.Fields["jobFamily"]),
		Headcount:    strings.TrimSpace(position.Fields["headcount"]),
		Requirements: strings.TrimSpace(position.Fields["requirements"]),
	}
	// تحصیلات، مدرک ملاک، حکم جاری و ایثارگری از ماژول‌های مالک داده.
	holder.Education = facts.educationOf(a.EmployeeID)
	holder.DegreeBasis = facts.basisOf(a.EmployeeID)
	holder.PersonalBase = facts.personalBaseOf(a.EmployeeID)
	veteran := facts.veteranOf(a.EmployeeID)
	holder.Veteran = veteran.IsVeteran
	holder.VeteranTypeFa = veteran.TypeFa
	holder.VeteranNoteFa = veteran.NoteFa
	return holder
}

// firstActiveHolder نوترین انتصاب «در حال تصدی» یک پست را به «متصدی» تبدیل می‌کند.
func firstActiveHolder(scheme rules.OrgCodeScheme, assignments []contracts.Record, employees *employeeLookup, facts holderFacts) *rosterHolder {
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
	holder := toHolder(scheme, *best, employees, facts, contracts.Record{})
	return &holder
}

/* ------------------- اطلاعات پایهٔ متصدیان از ماژول‌های مالک ------------------- */

// veteranFact وضعیت ایثارگری یک کارمند.
type veteranFact struct {
	IsVeteran bool
	TypeFa    string
	StatusFa  string
	NoteFa    string
}

// holderFacts اطلاعات پایهٔ متصدیان: تحصیلات، مدرک ملاک حقوق، حکم جاری و ایثارگری.
//
// چرا یک‌جا جمع می‌شود: ساختار سازمانی برای هر پست متصدی را نشان می‌دهد و اگر برای هر
// متصدی جداگانه از ماژول‌های دیگر خوانده شود، با N پست N بار رفت‌وبرگشت می‌شود. اینجا
// هر منبع یک بار خوانده و بر پایهٔ کد پرسنلی گروه می‌شود.
type holderFacts struct {
	educations map[string][]contracts.Record
	bases      map[string]contracts.Record
	decrees    map[string]contracts.SalaryDecree
	veterans   map[string]veteranFact
}

// collectHolderFacts دادهٔ پایهٔ متصدیان را از پورت‌های قرارداد می‌خواند.
func collectHolderFacts(r *http.Request, deps modulekit.Deps, employeeIDs []string) holderFacts {
	facts := holderFacts{
		educations: map[string][]contracts.Record{},
		bases:      map[string]contracts.Record{},
		decrees:    map[string]contracts.SalaryDecree{},
		veterans:   map[string]veteranFact{},
	}
	if len(employeeIDs) == 0 {
		return facts
	}
	wanted := map[string]bool{}
	for _, id := range employeeIDs {
		wanted[id] = true
	}

	// سابقه‌های تحصیلی و تصمیم‌های ملاک حقوق از ماژول بایگانی.
	if port, ok := deps.Ports.RecordList("archive"); ok {
		if page, err := port.List(r.Context(), contracts.RecordFilter{
			Module: "archive", Kind: "educations", Scope: "all", Size: 500,
		}); err == nil {
			for _, rec := range page.Items {
				if wanted[rec.EmployeeID] {
					facts.educations[rec.EmployeeID] = append(facts.educations[rec.EmployeeID], rec)
				}
			}
		}
		if page, err := port.List(r.Context(), contracts.RecordFilter{
			Module: "archive", Kind: "degreeBases", Scope: "all", Size: 500,
		}); err == nil {
			for _, rec := range page.Items {
				if !wanted[rec.EmployeeID] || rec.Status != "active" {
					continue
				}
				facts.bases[rec.EmployeeID] = rec
			}
		}
	}

	// پروندهٔ ایثارگری از ماژول امور ایثارگران.
	//
	// دو وضعیت پذیرفته می‌شود: «تأییدشده» (قطعی) و «بررسی‌شده» (مدارک بررسی شده و در
	// انتظار تأیید نهایی). معادل‌سازی «یک مقطع بالاتر» تصمیم شورای تعدیل مدرک است، ولی
	// نمایش وضعیت ایثارگری به کارشناس کمک می‌کند تصمیم را با مستندات بگیرد.
	if port, ok := deps.Ports.RecordList("grievances"); ok {
		if page, err := port.List(r.Context(), contracts.RecordFilter{
			Module: "grievances", Kind: "veteranRecords", Scope: "all", Size: 500,
		}); err == nil {
			for _, rec := range page.Items {
				if !wanted[rec.EmployeeID] {
					continue
				}
				if rec.Status != "approved" && rec.Status != "verified" {
					continue
				}
				note := "پروندهٔ ایثارگری تأییدشده؛ معادل‌سازی «یک مقطع بالاتر» برای مدرک ملاک حقوق مجاز است."
				if rec.Status == "verified" {
					note = "پروندهٔ ایثارگری بررسی‌شده و در انتظار تأیید نهایی؛ معادل‌سازی «یک مقطع بالاتر» پس از تأیید پرونده ثبت می‌شود."
				}
				facts.veterans[rec.EmployeeID] = veteranFact{
					IsVeteran: true,
					TypeFa:    rec.Fields["veteranType"],
					StatusFa:  rec.StatusFa,
					NoteFa:    note,
				}
			}
		}
	}

	// پایهٔ شخصی فرد از حکم ابلاغ‌شدهٔ خودش.
	if reader, ok := deps.Ports.SalaryDecrees(); ok {
		for _, id := range employeeIDs {
			if decree, found, err := reader.NotifiedDecreeOf(r.Context(), id); err == nil && found {
				facts.decrees[id] = decree
			}
		}
	}
	return facts
}

// educationOf آخرین مدرک تحصیلی یک کارمند (بالاترین مقطع).
func (f holderFacts) educationOf(employeeID string) *holderEducation {
	items := f.educations[employeeID]
	if len(items) == 0 {
		return nil
	}
	best := items[0]
	bestOrder := contracts.DegreeLevelOrder(best.Fields["level"])
	for _, item := range items[1:] {
		order := contracts.DegreeLevelOrder(item.Fields["level"])
		if order > bestOrder || (order == bestOrder && item.Fields["graduationDate"] > best.Fields["graduationDate"]) {
			best, bestOrder = item, order
		}
	}
	return &holderEducation{
		RecordID:         best.ID,
		Level:            best.Fields["level"],
		LevelFa:          contracts.DegreeLevelTitleFa(best.Fields["level"]),
		Field:            best.Fields["field"],
		Orientation:      best.Fields["orientation"],
		University:       best.Fields["university"],
		GraduationDate:   best.Fields["graduationDate"],
		Relevance:        best.Fields["relevance"],
		RelevanceFa:      contracts.DegreeRelevanceTitleFa(best.Fields["relevance"]),
		RelevancePercent: best.Fields["relevancePercent"],
		CouncilStatusFa:  contracts.DegreeCouncilStatusTitleFa(best.Fields["councilStatus"]),
		CouncilNo:        best.Fields["councilNo"],
		CouncilDate:      best.Fields["councilDate"],
		StatusFa:         best.StatusFa,
	}
}

// basisOf مدرک ملاک حقوق فعال یک کارمند.
func (f holderFacts) basisOf(employeeID string) *holderDegreeBasis {
	rec, ok := f.bases[employeeID]
	if !ok {
		return nil
	}
	basis := &holderDegreeBasis{
		RecordID:         rec.ID,
		LevelFa:          contracts.DegreeLevelTitleFa(rec.Fields["level"]),
		EffectiveLevelFa: contracts.DegreeLevelTitleFa(rec.Fields["level"]),
		Field:            rec.Fields["field"],
		Relevance:        rec.Fields["relevance"],
		RelevanceFa:      contracts.DegreeRelevanceTitleFa(rec.Fields["relevance"]),
		RelevancePct:     rec.Fields["relevancePercent"],
		CouncilNo:        rec.Fields["councilNo"],
		CouncilDate:      rec.Fields["councilDate"],
		EffectiveDate:    rec.Fields["effectiveDate"],
		SalaryGroup:      rec.Fields["salaryGroup"],
		VeteranUpgrade:   rec.Fields["veteranUpgrade"] == "yes",
		StatusFa:         rec.StatusFa,
	}
	if basis.VeteranUpgrade {
		basis.UpgradedLevelFa = contracts.DegreeLevelTitleFa(rec.Fields["upgradedLevel"])
		if basis.UpgradedLevelFa != "" {
			// همان قاعده‌ای که فرم ثبت ملاک حقوق اجرا می‌کند: برای ایثارگران، مقطع
			// معادل‌سازی‌شده یک پله بالاتر، مبنای واقعی گروه شغلی است.
			basis.EffectiveLevelFa = basis.UpgradedLevelFa
		}
	}
	return basis
}

// personalBaseOf پایهٔ شخصی فرد از حکم ابلاغ‌شده.
func (f holderFacts) personalBaseOf(employeeID string) *holderPersonalBase {
	decree, ok := f.decrees[employeeID]
	if !ok {
		return nil
	}
	return &holderPersonalBase{
		DecreeID:      decree.ID,
		JobGroup:      decree.JobGroup,
		BaseSalary:    decree.BaseSalary,
		MazdBase:      decree.MazdBase,
		EffectiveDate: decree.EffectiveDate,
		StatusFa:      decree.StatusFa,
		NoteFa:        "از حکم ابلاغ‌شدهٔ خودِ کارمند خوانده می‌شود (پایهٔ شخصی فرد).",
	}
}

// veteranOf وضعیت ایثارگری یک کارمند.
func (f holderFacts) veteranOf(employeeID string) veteranFact {
	if fact, ok := f.veterans[employeeID]; ok {
		return fact
	}
	return veteranFact{}
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
