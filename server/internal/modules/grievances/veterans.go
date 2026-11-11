// مسیرهای امور ایثارگران — پروندهٔ ایثارگری، محاسبهٔ امتیازها و تسهیلات.
//
// چرا این فایل: پروندهٔ ایثارگری فقط یک رکورد نیست؛ امتیازهای آن (معافیت مالیاتی، معافیت
// کسور، سنوات ارفاقی، افزایش سنواتی، یک مقطع بالاتر، پایهٔ تشویقی و مرخصی تشویقی) باید
// محاسبه شوند، در پرونده به‌عنوان «مقدار تأییدشده» بنشینند و موتور حقوق و پروندهٔ تحصیلی
// همان مقدار تأییدشده را بخوانند. قاعده‌ها در `internal/rules/veteran.go` است؛ این فایل فقط
// ورودی را به قاعده می‌دهد و برمی‌گرداند.
package grievances

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/grievances/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registerVeteranEndpoints مسیرهای امور ایثارگران را ثبت می‌کند.
func registerVeteranEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("grievances", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("grievances", "write", fn) }

	// فهرست‌های مرجع: گروه‌های ایثارگری، پارامترهای شرکت، متن دستورالعمل و وضعیت پرونده‌ها.
	mux.HandleFunc("GET /api/v1/grievances/veteran-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, veteranDefaults(r.Context(), deps, svc))
	}))

	// محاسبهٔ پیش‌نمایش امتیازها (بدون تغییر داده).
	mux.HandleFunc("POST /api/v1/grievances/veteran-plan", read(func(w http.ResponseWriter, r *http.Request) {
		var in veteranRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی محاسبهٔ امتیاز ایثارگری خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, veteranPlan(r.Context(), deps, in))
	}))

	// ثبت یا به‌روزرسانی پرونده با امتیازهای محاسبه‌شدهٔ سرور.
	mux.HandleFunc("POST /api/v1/grievances/veteran-files", write(func(w http.ResponseWriter, r *http.Request) {
		var in veteranRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی پروندهٔ ایثارگری خوانده نشد.", nil)
			return
		}
		out, msg, err := veteranSave(r.Context(), deps, svc, in, actorOf(r))
		if msg != "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, msg, nil)
			return
		}
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, out)
	}))

	// وضعیت ایثارگری یک کارمند برای مصرف‌کننده‌های دیگر (حقوق، تحصیلات، ساختار سازمانی).
	mux.HandleFunc("GET /api/v1/grievances/veteran-profile", read(func(w http.ResponseWriter, r *http.Request) {
		employeeID := strings.TrimSpace(r.URL.Query().Get("employeeId"))
		httpx.WriteJSON(w, http.StatusOK, veteranProfileOut(r.Context(), deps, svc, employeeID))
	}))

	// گزارش جامعهٔ ایثارگران: شمار پرونده‌ها به تفکیک گروه و وضعیت.
	mux.HandleFunc("GET /api/v1/grievances/veterans-report", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, veteranReport(r.Context(), deps, svc))
	}))
}

// veteranRequest ورودی محاسبه یا ثبت پروندهٔ ایثارگری.
type veteranRequest struct {
	// RecordID اگر پر شده باشد، همان پرونده به‌روزرسانی می‌شود (نه پروندهٔ تازه).
	RecordID   string `json:"recordId,omitempty"`
	EmployeeID string `json:"employeeId"`
	// VeteranType کلید گروه یا عنوان فارسی (هر دو پذیرفته است).
	VeteranType     string  `json:"veteranType"`
	Percent         float64 `json:"percent"`
	CaptivityMonths int     `json:"captivityMonths"`
	FrontMonths     int     `json:"frontMonths"`
	Relation        string  `json:"relation"`
	CardNo          string  `json:"cardNo"`
	IssuerFa        string  `json:"issuerFa"`
	IssueDate       string  `json:"issueDate"`
	// ApprovedFile آیا کارشناس پرونده را تأییدشده ثبت می‌کند (تا آن زمان امتیاز مالی ندارد).
	ApprovedFile bool   `json:"approvedFile"`
	Benefits     string `json:"benefits"`
	NoteFa       string `json:"noteFa"`
}

// veteranDefaults فهرست‌های مرجع و متن دستورالعمل.
func veteranDefaults(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) map[string]any {
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	files := veteranFiles(ctx, svc)
	byStatus := map[string]int{}
	byGroup := map[string]int{}
	for _, rec := range files {
		byStatus[rec.StatusFa]++
		byGroup[rules.VeteranGroupTitleFa(rules.VeteranGroupOf(rec.Fields["veteranType"]))]++
	}
	return map[string]any{
		"groups":      rules.VeteranGroups(),
		"yesNo":       yesNoOptions(),
		"instruction": rules.VeteranInstructionFa(),
		"params":      veteranParamsView(params),
		"paramViews":  veteranParamViews(deps),
		"today":       todayFa(deps),
		"currentYear": jalaliYear(deps),
		"byStatus":    byStatus,
		"byGroup":     byGroup,
		"filesCount":  len(files),
		"noteFa": "پروندهٔ ایثارگری فقط با مستندات مرجع حمایتی ثبت می‌شود؛ امتیازها تا «تأیید پرونده» " +
			"محاسبه و نمایش داده می‌شوند ولی اثر مالی ندارند و پس از تأیید در فیش حقوق اعمال می‌شوند.",
	}
}

// veteranPlan محاسبهٔ امتیازها بدون تغییر داده.
func veteranPlan(ctx context.Context, deps modulekit.Deps, in veteranRequest) map[string]any {
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	profile := veteranProfileFromRequest(in)
	ent := rules.ComputeVeteranEntitlements(profile, params)
	return map[string]any{
		"employeeId":   strings.TrimSpace(in.EmployeeID),
		"employeeName": employeeName(ctx, deps, in.EmployeeID),
		"profile":      profile,
		"result":       ent,
		"payrollApply": params.PayrollApply,
	}
}

// veteranSave پرونده را با امتیازهای محاسبه‌شدهٔ سرور ثبت یا به‌روزرسانی می‌کند.
func veteranSave(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in veteranRequest, actor string) (map[string]any, string, error) {
	employeeID := strings.TrimSpace(in.EmployeeID)
	if employeeID == "" {
		return nil, "انتخاب کارمند الزامی است.", nil
	}
	group := rules.VeteranGroupOf(in.VeteranType)
	if group == rules.VeteranGroupNone {
		return nil, "گروه ایثارگری را انتخاب کنید.", nil
	}
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	profile := veteranProfileFromRequest(in)
	ent := rules.ComputeVeteranEntitlements(profile, params)

	fields := map[string]string{
		"veteranType":          group,
		"percent":              floatField(profile.DisabilityPercent),
		"captivityMonths":      intField(profile.CaptivityMonths),
		"frontMonths":          intField(profile.FrontMonths),
		"relation":             strings.TrimSpace(in.Relation),
		"cardNo":               strings.TrimSpace(in.CardNo),
		"issuerFa":             strings.TrimSpace(in.IssuerFa),
		"issueDate":            calendar.NormalizeDigits(strings.TrimSpace(in.IssueDate)),
		"erafaqiMonths":        intField(ent.ErafaqiMonths),
		"extraServiceMonths":   intField(ent.TotalExtraServiceMonths),
		"taxExemptPercent":     floatField(ent.TaxExempt),
		"pensionExemptPercent": floatField(ent.PensionExempt),
		"incentiveSteps":       intField(ent.IncentiveSteps),
		"leaveDays":            intField(ent.LeaveDays),
		"benefits":             strings.TrimSpace(in.Benefits),
		"calculationLog":       entitlementLogFa(ent),
	}
	if ent.EducationUpgrade {
		fields["educationUpgrade"] = "yes"
	} else {
		fields["educationUpgrade"] = "no"
	}

	title := "پروندهٔ ایثارگری — " + ent.GroupFa
	if name := employeeName(ctx, deps, employeeID); name != "" {
		title = "پروندهٔ ایثارگری " + name + " — " + ent.GroupFa
	}
	input := contracts.RecordInput{
		Kind: domain.KindVeteranRecords, Title: title, EmployeeID: employeeID,
		Fields: fields, NoteFa: strings.TrimSpace(in.NoteFa),
	}

	var rec contracts.Record
	var err error
	if id := strings.TrimSpace(in.RecordID); id != "" {
		rec, err = svc.Update(ctx, id, input, actor)
	} else {
		rec, err = svc.Create(ctx, input, actor)
	}
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"record":  rec,
		"result":  ent,
		"profile": profile,
	}, "", nil
}

// veteranProfileOut وضعیت ایثارگری یک کارمند (برای حقوق، تحصیلات و ساختار سازمانی).
func veteranProfileOut(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, employeeID string) map[string]any {
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	profile, rec, found := veteranFileOf(ctx, svc, employeeID)
	ent := rules.ComputeVeteranEntitlements(profile, params)
	out := map[string]any{
		"employeeId":   employeeID,
		"employeeName": employeeName(ctx, deps, employeeID),
		"hasFile":      found,
		"profile":      profile,
		"result":       ent,
		"payrollApply": params.PayrollApply,
	}
	if found {
		out["recordId"] = rec.ID
		out["statusFa"] = rec.StatusFa
	}
	return out
}

// veteranReport گزارش جامعهٔ ایثارگران.
func veteranReport(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) map[string]any {
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	files := veteranFiles(ctx, svc)
	type row struct {
		RecordID      string `json:"recordId"`
		EmployeeID    string `json:"employeeId"`
		EmployeeName  string `json:"employeeNameFa"`
		GroupFa       string `json:"groupFa"`
		StatusFa      string `json:"statusFa"`
		TaxExemptFa   string `json:"taxExemptFa"`
		ExtraMonthsFa string `json:"extraServiceMonthsFa"`
		EducationFa   string `json:"educationUpgradeFa"`
		IncentiveFa   string `json:"incentiveStepsFa"`
	}
	names := employeeNames(ctx, deps)
	rows := make([]row, 0, len(files))
	approved, taxExempt, extraMonths := 0, 0, 0
	for _, rec := range files {
		profile := profileOfRecord(rec)
		ent := rules.ComputeVeteranEntitlements(profile, params)
		if ent.Approved {
			approved++
		}
		if ent.TaxExempt > 0 {
			taxExempt++
		}
		extraMonths += ent.TotalExtraServiceMonths
		rows = append(rows, row{
			RecordID: rec.ID, EmployeeID: rec.EmployeeID, EmployeeName: names[rec.EmployeeID],
			GroupFa:     rules.VeteranGroupTitleFa(rules.VeteranGroupOf(rec.Fields["veteranType"])),
			StatusFa:    rec.StatusFa,
			TaxExemptFa: rules.IntFa(ent.TaxExempt), ExtraMonthsFa: rules.IntFa(ent.TotalExtraServiceMonths),
			EducationFa: yesNoFa(ent.EducationUpgrade), IncentiveFa: rules.IntFa(ent.IncentiveSteps),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].EmployeeID < rows[j].EmployeeID })
	return map[string]any{
		"total":            len(files),
		"approved":         approved,
		"taxExemptCount":   taxExempt,
		"extraMonthsTotal": extraMonths,
		"quotaFa":          rules.IntFa(params.EmploymentQuotaPercent) + "٪",
		"rows":             rows,
		"noteFa":           "امتیازها از مقدار تأییدشدهٔ هر پرونده خوانده می‌شود؛ پروندهٔ تأییدنشده در این گزارش امتیاز مالی ندارد.",
	}
}

/* --------------------------- خواندن و تبدیل داده --------------------------- */

// veteranFiles همهٔ پرونده‌های ایثارگری ماژول را برمی‌گرداند.
func veteranFiles(ctx context.Context, svc *recordkit.Service) []contracts.Record {
	if svc == nil {
		return nil
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "grievances", Kind: domain.KindVeteranRecords, Scope: "all", Size: 500,
	})
	if err != nil {
		return nil
	}
	return page.Items
}

// veteranFileOf پروندهٔ ایثارگری یک کارمند؛ تأییدشده بر بررسی‌شده اولویت دارد.
func veteranFileOf(ctx context.Context, svc *recordkit.Service, employeeID string) (rules.VeteranProfile, contracts.Record, bool) {
	if svc == nil || employeeID == "" {
		return rules.VeteranProfile{}, contracts.Record{}, false
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "grievances", Kind: domain.KindVeteranRecords,
		EmployeeID: employeeID, Scope: "all", Size: 50,
	})
	if err != nil {
		return rules.VeteranProfile{}, contracts.Record{}, false
	}
	best := contracts.Record{}
	bestRank := -1
	for _, rec := range page.Items {
		if rec.EmployeeID != employeeID {
			continue
		}
		rank := 0
		switch rec.Status {
		case "approved":
			rank = 3
		case "verified":
			rank = 2
		}
		if rank > bestRank {
			best, bestRank = rec, rank
		}
	}
	if bestRank < 0 {
		return rules.VeteranProfile{}, contracts.Record{}, false
	}
	return profileOfRecord(best), best, true
}

// profileOfRecord پروندهٔ رکورد را به پروفایل موتور تبدیل می‌کند.
func profileOfRecord(rec contracts.Record) rules.VeteranProfile {
	f := rec.Fields
	return rules.VeteranProfile{
		Group:             rules.VeteranGroupOf(f["veteranType"]),
		DisabilityPercent: numberValue(f["percent"]),
		CaptivityMonths:   int(numberValue(f["captivityMonths"])),
		FrontMonths:       int(numberValue(f["frontMonths"])),
		ApprovedFile:      rec.Status == "approved",
	}
}

// veteranProfileFromRequest پروفایل موتور از ورودی فرم.
func veteranProfileFromRequest(in veteranRequest) rules.VeteranProfile {
	return rules.VeteranProfile{
		Group:             rules.VeteranGroupOf(in.VeteranType),
		DisabilityPercent: in.Percent,
		CaptivityMonths:   in.CaptivityMonths,
		FrontMonths:       in.FrontMonths,
		ApprovedFile:      in.ApprovedFile,
	}
}

// entitlementLogFa سطرهای محاسبه را به یک متن خوانا برای پرونده تبدیل می‌کند.
func entitlementLogFa(ent rules.VeteranEntitlement) string {
	parts := make([]string, 0, len(ent.Lines))
	for _, line := range ent.Lines {
		parts = append(parts, line.TitleFa+": "+line.ValueFa)
	}
	return strings.Join(parts, "؛ ")
}

// veteranParamValues پارامترهای شرکت با پیشوند `veteran.` را عدد می‌کند.
func veteranParamValues(deps modulekit.Deps) map[string]float64 {
	out := map[string]float64{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "veteran.") {
			continue
		}
		value := strings.ReplaceAll(calendar.NormalizeDigits(strings.TrimSpace(p.Value)), ",", "")
		value = strings.ReplaceAll(value, "٪", "")
		value = strings.ReplaceAll(value, "%", "")
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			out[p.Key] = v
		}
	}
	return out
}

// veteranParamsView پارامترهای مؤثر برای نمایش (نه فهرست ثابت در رابط).
func veteranParamsView(params rules.VeteranParams) map[string]any {
	return map[string]any{
		"taxExemptPercent":        params.TaxExemptPercent,
		"taxMinFrontMonths":       params.TaxMinFrontMonths,
		"pensionExemptPercent":    params.PensionExemptPercent,
		"captivityServiceFactor":  params.CaptivityServiceFactor,
		"erafaqiMaxMonths":        params.ErafaqiMaxMonths,
		"educationUpgrade":        params.EducationUpgrade,
		"incentiveSteps":          params.IncentiveSteps,
		"incentiveStepsMax":       params.IncentiveStepsMax,
		"captivityLeaveRatio":     params.CaptivityLeaveRatio,
		"captivityLeaveMinMonths": params.CaptivityLeaveMinMonths,
		"captivityLeaveMaxMonths": params.CaptivityLeaveMaxMonths,
		"employmentQuotaPercent":  params.EmploymentQuotaPercent,
		"payrollApply":            params.PayrollApply,
	}
}

// veteranParamViews پارامترهای ایثارگری که کارشناس می‌بیند (از تنظیمات شرکت).
func veteranParamViews(deps modulekit.Deps) []map[string]string {
	out := []map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "veteran.") {
			continue
		}
		out = append(out, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa,
		})
	}
	return out
}

/* ------------------------------ کمک‌های کوچک ------------------------------ */

// actorOf نام کاربر جاری برای ثبت در رکورد.
func actorOf(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok {
		return user.Username
	}
	return "ناشناس"
}

// employeeName نام کارمند از پورت مشترک کارکنان.
func employeeName(ctx context.Context, deps modulekit.Deps, employeeID string) string {
	id := strings.TrimSpace(employeeID)
	if id == "" {
		return ""
	}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return ""
	}
	employee, found, err := reader.Get(ctx, contracts.EmployeeFilter{Size: 1}, id)
	if err != nil || !found {
		return ""
	}
	return employee.FullName
}

// employeeNames نقشهٔ «کد کارمند → نام» با یک پرس‌وجو.
//
// چرا یکجا: گزارش جامعهٔ ایثارگران برای هر پرونده نام کارمند را نشان می‌دهد؛ خواندن نام
// به‌تفکیک پرونده، به تعداد پرونده‌ها پرس‌وجو می‌فرستاد.
func employeeNames(ctx context.Context, deps modulekit.Deps) map[string]string {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return nil
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Size: 500})
	if err != nil {
		return nil
	}
	out := make(map[string]string, len(page.Items))
	for _, employee := range page.Items {
		out[employee.ID] = employee.FullName
	}
	return out
}

// yesNoOptions مقادیر مجاز فیلدهای بله/خیر (دامنه، نه رابط).
func yesNoOptions() []map[string]string {
	return []map[string]string{
		{"key": "yes", "titleFa": "بله"},
		{"key": "no", "titleFa": "خیر"},
	}
}

// yesNoFa برچسب فارسی بولی.
func yesNoFa(v bool) string {
	if v {
		return "بله"
	}
	return "خیر"
}

// todayFa تاریخ امروز شمسی برای پیش‌فرض فرم.
func todayFa(deps modulekit.Deps) string {
	if deps.Clock == nil {
		return ""
	}
	date, err := calendar.FromGregorian(deps.Clock())
	if err != nil {
		return ""
	}
	return date.String()
}

// jalaliYear سال شمسی امروز.
func jalaliYear(deps modulekit.Deps) int {
	if deps.Clock == nil {
		return 0
	}
	date, err := calendar.FromGregorian(deps.Clock())
	if err != nil {
		return 0
	}
	return date.Year
}

// numberValue عدد یک فیلد متنی را می‌خواند (ارقام فارسی هم پذیرفته می‌شود).
func numberValue(value string) float64 {
	v := strings.TrimSpace(calendar.NormalizeDigits(value))
	v = strings.ReplaceAll(v, ",", "")
	v = strings.ReplaceAll(v, "٬", "")
	if v == "" {
		return 0
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return n
}

// intField عدد صحیح را برای فیلد رکورد آماده می‌کند.
func intField(v int) string { return strconv.Itoa(v) }

// floatField عدد اعشاری را بدون صفر اضافه می‌نویسد.
func floatField(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
