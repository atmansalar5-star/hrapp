// پیاده‌سازی ماژول «طبقه‌بندی مشاغل» — شرح شغل، شرح مشاغل، شناسنامهٔ شغل و فرم محاسباتی
// شرایط احراز.
//
// چرا این فایل: سه سند منابع انسانی که کاربر می‌خواهد از یک رکورد ساخته می‌شوند و نباید
// قاعده‌شان در رابط نوشته شود:
//
//	شرح شغل       ← رکورد نوع «jobDescriptions» (فرم کامل در همان صفحه ثبت می‌شود)
//	شرح مشاغل     ← فهرست شرح شغل‌های سازمان از همان رکوردها
//	شناسنامهٔ شغل ← کارت یک‌صفحه‌ای که دامنه از شرح شغل + طبقه + ارزشیابی می‌سازد
//
// فرم محاسباتی «کمبود تحصیلات و مازاد بر شرایط احراز» هم همین‌جا سرو می‌شود و محاسبه‌اش در
// موتور قواعد (`rules.EvaluateQualification`) است؛ پس فرم وب، سند چاپی و تصویب‌خواهی
// انتصاب یک عدد می‌بینند و پارامترهای شرکت (نه عددهای ثابت کد) مبنای محاسبه‌اند.
package jobclass

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/jobclass/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registerJobEndpoints مسیرهای شرح مشاغل، شناسنامهٔ شغل و فرم محاسباتی را ثبت می‌کند.
func registerJobEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("jobclass", "read", fn) }

	// فهرست‌های مرجع و پارامترها: مقاطع تحصیلی، نردبان گروه شغلی، شناسنامه‌های موجود،
	// کارکنان (برای پیش‌پرکردن سابقه) و متن دستورالعمل.
	mux.HandleFunc("GET /api/v1/jobclass/job-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, jobDefaults(r.Context(), deps, svc))
	}))

	// شرح مشاغل: فهرست همهٔ شرح شغل‌های سازمان با شناسهٔ شغل و شرایط احراز.
	mux.HandleFunc("GET /api/v1/jobclass/job-registry", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, jobRegistry(r.Context(), svc))
	}))

	// شناسنامهٔ شغل: کارت یک شغل بر پایهٔ شناسهٔ رکورد شرح شغل (یا کد شغل).
	mux.HandleFunc("GET /api/v1/jobclass/job-card", read(func(w http.ResponseWriter, r *http.Request) {
		card, ok := jobCard(r.Context(), svc, r.URL.Query().Get("id"), r.URL.Query().Get("jobCode"))
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound,
				"شرح شغل یا شناسنامهٔ شغل با این شناسه یافت نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, card)
	}))

	// فرم محاسباتی شرایط احراز: کمبود تحصیلات، شرایط معادل و مازاد.
	mux.HandleFunc("POST /api/v1/jobclass/qualification-plan", read(func(w http.ResponseWriter, r *http.Request) {
		var in qualificationRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی فرم محاسباتی خوانده نشد.", nil)
			return
		}
		out, msg := qualificationPlan(r.Context(), deps, svc, in)
		if msg != "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, msg, nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))
}

// qualificationRequest ورودی فرم محاسباتی شرایط احراز.
//
// دو راه برای تعیین شرط شغل: انتخاب شناسنامهٔ شغل (recordId یا jobCode) یا نوشتن دستی شرط
// در همان فرم. شرط دستی برای سنجش فرضی («اگر مدرک کارشناسی ارشد می‌خواست چه می‌شد؟») است.
type qualificationRequest struct {
	RecordID   string `json:"recordId"`
	JobCode    string `json:"jobCode"`
	EmployeeID string `json:"employeeId"`

	DegreeLevel     string `json:"degreeLevel"`
	Field           string `json:"field"`
	ExperienceYears int    `json:"experienceYears"`

	RequireDegreeLevel        string `json:"requireDegreeLevel"`
	RequireFields             string `json:"requireFields"`
	RequireMinExperienceYears int    `json:"requireMinExperienceYears"`
}

// jobDefaults پارامترها و فهرست‌های مرجع فرم‌های شرح شغل و شرایط احراز.
func jobDefaults(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) map[string]any {
	today := ""
	if now, err := calendar.FromGregorian(deps.Clock()); err == nil {
		today = now.String()
	}
	page, _ := svc.List(ctx, contracts.RecordFilter{
		Module: "jobclass", Kind: domain.KindDescriptions, Scope: "all", Size: 300,
	})
	jobs := make([]map[string]any, 0, len(page.Items))
	for _, rec := range page.Items {
		req := domain.Requirement(rec.Fields)
		jobs = append(jobs, map[string]any{
			"id": rec.ID, "titleFa": rec.Title, "code": rec.Fields[domain.FieldJobCode],
			"status": rec.Status, "statusFa": rec.StatusFa,
			"approved":      rec.Status == domain.StatusApproved || rec.Status == domain.StatusRevised,
			"degreeLevel":   req.DegreeLevel,
			"degreeFa":      rules.DegreeTitleFa(req.DegreeLevel),
			"fieldsFa":      strings.Join(req.Fields, " / "),
			"minExperience": req.MinExperienceYears,
		})
	}
	return map[string]any{
		"degreeLevels":     rules.DegreeLevels(),
		"grades":           rules.Grades(),
		"params":           qualificationParams(deps),
		"paramViews":       qualificationParamViews(deps),
		"instruction":      rules.QualificationInstructionFa(),
		"fieldSuggestions": fieldSuggestions(),
		"employees":        employeeOptions(ctx, deps),
		"jobs":             jobs,
		"today":            today,
		"company":          deps.Org.TitleFa,
		"noteFa": "فرم محاسباتی بر پایهٔ شناسنامهٔ شغل مصوب و پارامترهای شرکت ساخته می‌شود؛ " +
			"نتیجهٔ «برآورده» یا «برآورده با شرایط معادل» شرط لازم تصویب‌خواهی انتصاب و ارتقاست.",
	}
}

// jobRegistry فهرست شرح شغل‌های سازمان (سند «شرح مشاغل») با جمع‌بندی وضعیت.
func jobRegistry(ctx context.Context, svc *recordkit.Service) map[string]any {
	page, _ := svc.List(ctx, contracts.RecordFilter{
		Module: "jobclass", Kind: domain.KindDescriptions, Scope: "all", Size: 300,
	})
	items := make([]map[string]any, 0, len(page.Items))
	approved, draft := 0, 0
	for _, rec := range page.Items {
		req := domain.Requirement(rec.Fields)
		isApproved := rec.Status == domain.StatusApproved || rec.Status == domain.StatusRevised
		if isApproved {
			approved++
		} else {
			draft++
		}
		items = append(items, map[string]any{
			"id": rec.ID, "titleFa": rec.Title,
			"code":          rec.Fields[domain.FieldJobCode],
			"groupFa":       gradeTitleFa(rec.Fields[domain.FieldJobGroup]),
			"unitFa":        rec.Fields[domain.FieldUnit],
			"rankFa":        rec.Fields[domain.FieldJobRank],
			"status":        rec.Status,
			"statusFa":      rec.StatusFa,
			"approved":      isApproved,
			"degreeFa":      rules.DegreeTitleFa(req.DegreeLevel),
			"fieldsFa":      strings.Join(req.Fields, " / "),
			"minExperience": req.MinExperienceYears,
			"requirementFa": rules.RequirementFa(req),
			"revisionDate":  rec.Fields[domain.FieldRevisionDate],
		})
	}
	return map[string]any{
		"items": items, "total": len(items),
		"approvedCount": approved, "draftCount": draft,
		"noteFa": "شرح مشاغل، مجموعهٔ شرح شغل‌های سازمان است؛ تنها شناسنامهٔ تأییدشده مبنای " +
			"انتصاب و ارتقاست و شرح شغل پیش‌نویس باید نخست تأیید شود.",
	}
}

// jobCard شناسنامهٔ شغل را با شناسهٔ رکورد یا کد شغل می‌سازد.
func jobCard(ctx context.Context, svc *recordkit.Service, id, jobCode string) (domain.JobCard, bool) {
	id = strings.TrimSpace(id)
	if id != "" {
		rec, found, err := svc.Get(ctx, contracts.RecordFilter{Module: "jobclass", Scope: "all"}, id)
		if err != nil || !found {
			return domain.JobCard{}, false
		}
		if rec.Kind != domain.KindDescriptions {
			return domain.JobCard{}, false
		}
		return domain.BuildCard(rec), true
	}
	code := strings.TrimSpace(jobCode)
	if code == "" {
		return domain.JobCard{}, false
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "jobclass", Kind: domain.KindDescriptions, Scope: "all", Size: 300,
	})
	if err != nil {
		return domain.JobCard{}, false
	}
	if rec, found := domain.FindDescription(page.Items, code); found {
		return domain.BuildCard(rec), true
	}
	return domain.JobCard{}, false
}

// qualificationPlan فرم محاسباتی شرایط احراز را می‌سازد. پیام فارسی خطا (اگر باشد) دومین
// خروجی است تا لایهٔ HTTP فقط آن را برگرداند و قاعده اینجا نماند.
func qualificationPlan(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in qualificationRequest) (map[string]any, string) {
	requirement, jobTitle, jobID, msg := requirementOf(ctx, svc, in)
	if msg != "" {
		return nil, msg
	}
	if rules.DegreeRankOf(in.DegreeLevel) == 0 {
		return nil, "مقطع تحصیلی متقاضی را از نردبان مدرک انتخاب کنید."
	}
	candidate := rules.Candidate{
		DegreeLevel:     strings.TrimSpace(in.DegreeLevel),
		Field:           strings.TrimSpace(in.Field),
		ExperienceYears: in.ExperienceYears,
	}
	params := qualificationParams(deps)
	result := rules.EvaluateQualification(candidate, requirement, params)

	out := map[string]any{
		"result":        result,
		"requirement":   requirement,
		"requirementFa": rules.RequirementFa(requirement),
		"candidateFa":   rules.CandidateFa(candidate),
		"candidate":     candidate,
		"params":        params,
		"jobTitleFa":    jobTitle,
		"today":         todayFa(deps),
		"noteFa":        domain.QualificationNoteFa(),
	}
	if jobID != "" {
		out["jobId"] = jobID
	}
	if employee, ok := findEmployee(ctx, deps, in.EmployeeID); ok {
		out["employee"] = map[string]any{
			"id": employee.ID, "titleFa": employee.FullName,
			"position": employee.Position, "unit": employee.DepartmentName,
			"hireDate": employee.HireDate, "jobGrade": employee.JobGradeFa,
		}
	}
	return out, ""
}

// requirementOf شرط شغل را از شناسنامه یا از ورودی دستی می‌سازد و عنوان شغل را برمی‌گرداند.
func requirementOf(ctx context.Context, svc *recordkit.Service, in qualificationRequest) (rules.JobRequirement, string, string, string) {
	id := strings.TrimSpace(in.RecordID)
	code := strings.TrimSpace(in.JobCode)
	if id != "" || code != "" {
		rec, found := lookupDescription(ctx, svc, id, code)
		if !found {
			return rules.JobRequirement{}, "", "", "شناسنامهٔ شغل انتخاب‌شده یافت نشد."
		}
		return domain.Requirement(rec.Fields), rec.Title, rec.ID, ""
	}
	if rules.DegreeRankOf(in.RequireDegreeLevel) == 0 {
		return rules.JobRequirement{}, "", "", "ابتدا شناسنامهٔ شغل را انتخاب کنید یا مقطع تحصیلی لازم را در شرط دستی بنویسید."
	}
	return rules.JobRequirement{
		DegreeLevel:        strings.TrimSpace(in.RequireDegreeLevel),
		Fields:             domain.FieldList(in.RequireFields),
		MinExperienceYears: in.RequireMinExperienceYears,
	}, "شرط دستی (بدون شناسنامه)", "", ""
}

// lookupDescription شرح شغل را با شناسه یا کد شغل پیدا می‌کند.
func lookupDescription(ctx context.Context, svc *recordkit.Service, id, code string) (contracts.Record, bool) {
	if id != "" {
		rec, found, err := svc.Get(ctx, contracts.RecordFilter{Module: "jobclass", Scope: "all"}, id)
		if err == nil && found && rec.Kind == domain.KindDescriptions {
			return rec, true
		}
		return contracts.Record{}, false
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "jobclass", Kind: domain.KindDescriptions, Scope: "all", Size: 300,
	})
	if err != nil {
		return contracts.Record{}, false
	}
	return domain.FindDescription(page.Items, code)
}

// qualificationParams آستانه‌های شرایط احراز را از پارامترهای شرکت فعال می‌خواند.
func qualificationParams(deps modulekit.Deps) rules.QualificationParams {
	values := map[string]float64{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "qual.") {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64)
		if err != nil {
			continue
		}
		values[p.Key] = v
	}
	return rules.QualificationParamsFromValues(values)
}

// qualificationParamViews پارامترهای شرایط احراز را برای نمایش در فرم برمی‌گرداند.
func qualificationParamViews(deps modulekit.Deps) []map[string]string {
	out := []map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "qual.") {
			continue
		}
		out = append(out, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa,
		})
	}
	return out
}

// employeeOptions کارکنان با سابقهٔ محاسبه‌شده از تاریخ استخدام (برای پیش‌پرکردن فرم).
func employeeOptions(ctx context.Context, deps modulekit.Deps) []map[string]any {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return []map[string]any{}
	}
	list, err := reader.List(ctx, contracts.EmployeeFilter{Size: 500})
	if err != nil {
		return []map[string]any{}
	}
	now, _ := calendar.FromGregorian(deps.Clock())
	out := make([]map[string]any, 0, len(list.Items))
	for _, e := range list.Items {
		out = append(out, map[string]any{
			"id": e.ID, "titleFa": e.FullName + " — " + e.ID,
			"position": e.Position, "unit": e.DepartmentName,
			"hireDate": e.HireDate, "serviceYears": serviceYears(e.HireDate, now),
		})
	}
	return out
}

// findEmployee یک کارمند را از پورت مشترک کارکنان می‌خواند.
func findEmployee(ctx context.Context, deps modulekit.Deps, id string) (contracts.Employee, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return contracts.Employee{}, false
	}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return contracts.Employee{}, false
	}
	employee, found, err := reader.Get(ctx, contracts.EmployeeFilter{Size: 1}, id)
	if err != nil || !found {
		return contracts.Employee{}, false
	}
	return employee, true
}

// serviceYears سابقهٔ کل خدمت را از تاریخ استخدام (شمسی) تا امروز به سال کامل می‌سنجد.
func serviceYears(hireDate string, now calendar.Date) int {
	hire, err := calendar.Parse(strings.TrimSpace(hireDate))
	if err != nil {
		return 0
	}
	years := now.Year - hire.Year
	if now.Month < hire.Month || (now.Month == hire.Month && now.Day < hire.Day) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

// todayFa تاریخ امروز شمسی برای سرصفحهٔ فرم.
func todayFa(deps modulekit.Deps) string {
	if now, err := calendar.FromGregorian(deps.Clock()); err == nil {
		return now.String()
	}
	return ""
}

// gradeTitleFa عنوان فارسی گروه شغلی از نردبان سازمان.
func gradeTitleFa(key string) string {
	value := strings.TrimSpace(key)
	if value == "" {
		return ""
	}
	if rank := rules.GradeRankOf(value); rank > 0 {
		return rules.GradeTitleFa(rank)
	}
	return value
}

// fieldSuggestions پیشنهاد رشته‌های تحصیلی برای فرم شرح شغل (نمایشی؛ انتخاب آزاد است).
func fieldSuggestions() []string {
	return []string{
		"مدیریت", "مدیریت بازرگانی", "مدیریت مالی", "حسابداری", "اقتصاد",
		"مهندسی صنایع", "مهندسی شیمی", "مهندسی مکانیک", "مهندسی برق",
		"حقوق", "منابع انسانی", "فناوری اطلاعات", "ایمنی و بهداشت کار",
	}
}
