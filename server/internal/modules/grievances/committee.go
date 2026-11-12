// مسیرهای «کمیتهٔ ایثارگری» — فرم مصوبه و اعمال آن بر پروندهٔ ایثارگری.
//
// چرا این فایل جدا از `veterans.go`: پروندهٔ ایثارگری «واقعیت» کارمند را نگه می‌دارد و موتور
// قواعد پیش‌فرض قانونی را از آن می‌سازد؛ ولی تصمیم نهایی دربارهٔ تعلق گرفتن هر امتیاز و میزان
// آن، در کمیته گرفته می‌شود. این مسیرها همان تصمیم را به شکل یک رکورد امضاشدنی ثبت می‌کنند،
// «پیش‌فرض قانونی» را برای مقایسه نشان می‌دهند و با درخواست کارشناس، امتیازهای مصوب را روی
// پروندهٔ کارمند می‌نشانند تا فیش حقوق همان عدد را بخواند.
package grievances

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/grievances/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registerVeteranCommitteeEndpoints مسیرهای کمیتهٔ ایثارگری را ثبت می‌کند.
func registerVeteranCommitteeEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("grievances", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("grievances", "write", fn) }

	// فهرست‌های مرجع فرم مصوبه + پیش‌فرض قانونی همان کارمند برای مقایسه.
	mux.HandleFunc("GET /api/v1/grievances/veteran-committee-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		employeeID := strings.TrimSpace(r.URL.Query().Get("employeeId"))
		httpx.WriteJSON(w, http.StatusOK, veteranCommitteeDefaults(r.Context(), deps, svc, employeeID))
	}))

	// ثبت مصوبهٔ کمیته؛ با applyToFile امتیازهای مصوب روی پروندهٔ کارمند می‌نشیند.
	mux.HandleFunc("POST /api/v1/grievances/veteran-committee", write(func(w http.ResponseWriter, r *http.Request) {
		var in veteranCommitteeRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی مصوبهٔ کمیته خوانده نشد.", nil)
			return
		}
		out, msg, err := veteranCommitteeCreate(r.Context(), deps, svc, in, actorOf(r))
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

	// اعمال دوبارهٔ یک مصوبهٔ ثبت‌شده بر پرونده (مثلاً پس از تصویب آن).
	mux.HandleFunc("POST /api/v1/grievances/veteran-committee/{id}/apply", write(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		out, msg, err := veteranCommitteeApply(r.Context(), deps, svc, id, actorOf(r))
		if msg != "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, msg, nil)
			return
		}
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))
}

// veteranCommitteeRequest ورودی فرم مصوبهٔ کمیته.
type veteranCommitteeRequest struct {
	EmployeeID  string `json:"employeeId"`
	CommitteeNo string `json:"committeeNo"`
	MeetingDate string `json:"meetingDate"`
	Members     string `json:"members"`
	VeteranType string `json:"veteranGroup"`
	BasisDocs   string `json:"basisDocs"`

	EduUpgradeDecision      string  `json:"eduUpgradeDecision"`
	EduLevel                string  `json:"eduLevel"`
	TaxExemptDecision       string  `json:"taxExemptDecision"`
	TaxExemptPercent        float64 `json:"taxExemptPercent"`
	PensionExemptDecision   string  `json:"pensionExemptDecision"`
	PensionExemptPercent    float64 `json:"pensionExemptPercent"`
	WorkHourReduction       string  `json:"workHourReduction"`
	ReducedWeeklyHours      float64 `json:"reducedWeeklyHours"`
	IncentiveSteps          float64 `json:"incentiveSteps"`
	SeniorityMonths         float64 `json:"seniorityMonths"`
	SeniorityBasis          string  `json:"seniorityBasis"`
	LeaveDays               float64 `json:"leaveDays"`
	EmploymentQuotaDecision string  `json:"employmentQuotaDecision"`
	OtherBenefits           string  `json:"otherBenefits"`
	DecisionFa              string  `json:"decisionFa"`
	EffectiveDate           string  `json:"effectiveDate"`
	SignOff                 string  `json:"signOff"`
	// امضاکنندگان پایین فرم: شناسهٔ اعضای کمیته که در جلسه امضا کرده‌اند و تاریخ امضا.
	// متن «امضاکنندگان» در سرور ساخته می‌شود تا عنوان امضا یک‌جا از فهرست اعضا بیاید.
	SignedMemberIDs []string `json:"signedMemberIds"`
	SignDate        string   `json:"signDate"`
	ApplyToFile     bool     `json:"applyToFile"`
}

// veteranCommitteeDefaults فهرست‌های مرجع و پیش‌فرض قانونی همین کارمند.
func veteranCommitteeDefaults(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, employeeID string) map[string]any {
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	out := map[string]any{
		"groups":         rules.VeteranGroups(),
		"yesNo":          yesNoOptions(),
		"seniorityBases": rules.VeteranSeniorityBases(),
		"today":          todayFa(deps),
		"employeeId":     employeeID,
		"employeeName":   employeeName(ctx, deps, employeeID),
		"params":         veteranParamsView(params),
	}
	// اعضای کمیته: فهرست کامل برای پنل، فعال‌ها برای امضا، نقش‌های مصوب و متن پیش‌پر.
	members, _ := committeeMemberViews(ctx, svc, false)
	activeMembers, _ := committeeMemberViews(ctx, svc, true)
	out["members"] = members
	out["activeMembers"] = activeMembers
	out["roles"] = rules.VeteranCommitteeRoles()
	out["membersText"] = strings.Join(committeeActiveMemberNames(ctx, svc), "، ")
	out["signatures"] = committeeActiveMemberSignatures(ctx, svc)
	profile, rec, found := veteranFileOf(ctx, svc, employeeID)
	if !found {
		out["hasFile"] = false
		out["suggestion"] = nil
		out["noteFa"] = "برای این کارمند پروندهٔ ایثارگری ثبت نشده است؛ مصوبه را می‌توان ثبت کرد ولی اعمال بر پرونده و فیش " +
			"حقوق نیازمند ثبت پروندهٔ ایثارگری (گروه، درصد جانبازی یا مدت اسارت) است."
		return out
	}
	ent := rules.ComputeVeteranEntitlements(profile, params)
	out["hasFile"] = true
	out["statusFa"] = rec.StatusFa
	out["recordId"] = rec.ID
	out["suggestion"] = map[string]any{
		"veteranGroup":            profile.Group,
		"veteranGroupFa":          ent.GroupFa,
		"approved":                ent.Approved,
		"disabilityPercent":       profile.DisabilityPercent,
		"captivityMonths":         profile.CaptivityMonths,
		"frontMonths":             profile.FrontMonths,
		"taxExemptPercent":        ent.TaxExempt,
		"pensionExemptPercent":    ent.PensionExempt,
		"captivityServiceMonths":  ent.CaptivityServiceMonths,
		"erafaqiMonths":           ent.ErafaqiMonths,
		"totalExtraServiceMonths": ent.TotalExtraServiceMonths,
		"educationUpgrade":        ent.EducationUpgrade,
		"incentiveSteps":          ent.IncentiveSteps,
		"leaveDays":               ent.LeaveDays,
		"employmentQuotaPercent":  ent.EmploymentQuotaPercent,
		"lines":                   ent.Lines,
		"summaryFa":               ent.SummaryFa,
	}
	out["noteFa"] = "ستون «پیش‌فرض قانونی» از موتور قواعد ساخته می‌شود؛ هر بندی که کمیته خلاف آن مصوب کند در متن " +
		"مصوبه و در پروندهٔ کارمند ثبت می‌شود."
	return out
}

// veteranCommitteeCreate مصوبه را ثبت می‌کند و در صورت درخواست، بر پرونده اعمال می‌کند.
func veteranCommitteeCreate(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in veteranCommitteeRequest, actor string) (map[string]any, string, error) {
	employeeID := strings.TrimSpace(in.EmployeeID)
	if employeeID == "" {
		return nil, "انتخاب کارمند برای مصوبه الزامی است.", nil
	}
	if strings.TrimSpace(in.MeetingDate) == "" {
		return nil, "تاریخ جلسهٔ کمیته الزامی است.", nil
	}
	group := rules.VeteranGroupOf(in.VeteranType)
	if group == rules.VeteranGroupNone {
		return nil, "گروه ایثارگری را در مصوبه مشخص کنید.", nil
	}
	// متن «اعضای کمیته» و «امضاکنندگان» از فهرست اعضای تعریف‌شده ساخته می‌شود؛ ورودی دستی
	// فرم بر آن مقدم است تا کارشناس بتواند ترکیب همان جلسه را ثبت کند.
	if strings.TrimSpace(in.Members) == "" {
		in.Members = strings.Join(committeeActiveMemberNames(ctx, svc), "، ")
	}
	if len(in.SignedMemberIDs) > 0 {
		in.SignOff = rules.VeteranCommitteeSignOffFa(
			committeeSignedSignatures(ctx, svc, in.SignedMemberIDs), in.SignDate)
	}
	fields := committeeFields(in, group)
	title := "مصوبهٔ کمیتهٔ ایثارگری"
	if name := employeeName(ctx, deps, employeeID); name != "" {
		title += " " + name
	}
	if no := strings.TrimSpace(in.CommitteeNo); no != "" {
		title += " — صورت‌جلسهٔ " + no
	}
	rec, err := svc.Create(ctx, contracts.RecordInput{
		Kind: domain.KindVeteranCommittee, Title: title, EmployeeID: employeeID, Fields: fields,
		NoteFa: "ثبت از فرم کمیتهٔ ایثارگری",
	}, actor)
	if err != nil {
		return nil, "", err
	}
	out := map[string]any{"record": rec, "applied": false}
	if !in.ApplyToFile {
		// پیش‌نمایش اثر مصوبه بر پیش‌فرض قانونی، بدون تغییر پرونده.
		out["preview"] = committeePreview(ctx, deps, svc, employeeID, fields)
		return out, "", nil
	}
	applied, msg, err := veteranCommitteeApply(ctx, deps, svc, rec.ID, actor)
	if err != nil {
		return out, "", err
	}
	if msg != "" {
		out["applyMessage"] = msg
		return out, "", nil
	}
	out["applied"] = true
	out["apply"] = applied
	return out, "", nil
}

// veteranCommitteeApply امتیازهای مصوب را روی پروندهٔ همان کارمند می‌نشاند و پرونده را تأیید می‌کند.
func veteranCommitteeApply(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, recordID, actor string) (map[string]any, string, error) {
	if recordID == "" {
		return nil, "شناسهٔ مصوبه مشخص نیست.", nil
	}
	rec, found, err := svc.Get(ctx, contracts.RecordFilter{
		Module: "grievances", Kind: domain.KindVeteranCommittee, Scope: "all", Size: 1,
	}, recordID)
	if err != nil {
		return nil, "", err
	}
	if !found {
		return nil, "مصوبهٔ کمیته با این شناسه یافت نشد.", nil
	}
	employeeID := strings.TrimSpace(rec.EmployeeID)
	file, fileRec, hasFile := veteranFileOf(ctx, svc, employeeID)
	if !hasFile {
		return nil, "برای این کارمند پروندهٔ ایثارگری ثبت نشده است؛ نخست پرونده را ثبت کنید.", nil
	}
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	profile := file
	// مبنای اعمال، امتیاز فعلی همان پرونده است (پیش‌فرض قانونی + مقادیر تأییدشدهٔ پیشین).
	base := rules.VeteranEntitlementOfFile(fileRec.Fields, fileRec.Status == "approved", params)
	decision := rules.VeteranDecisionFromFields(rec.Fields)
	merged, changes := rules.ApplyVeteranDecision(base, decision)

	fields := map[string]string{
		"veteranType":          profile.Group,
		"percent":              floatField(profile.DisabilityPercent),
		"captivityMonths":      intField(profile.CaptivityMonths),
		"frontMonths":          intField(profile.FrontMonths),
		"erafaqiMonths":        intField(merged.ErafaqiMonths),
		"extraServiceMonths":   intField(merged.TotalExtraServiceMonths),
		"taxExemptPercent":     floatField(merged.TaxExempt),
		"pensionExemptPercent": floatField(merged.PensionExempt),
		"incentiveSteps":       intField(merged.IncentiveSteps),
		"leaveDays":            intField(merged.LeaveDays),
		"calculationLog":       entitlementLogFa(merged) + " | مصوبهٔ کمیته: " + rec.ID + decisionLogFa(rec.Fields, changes),
	}
	if merged.EducationUpgrade {
		fields["educationUpgrade"] = "yes"
	} else {
		fields["educationUpgrade"] = "no"
	}
	// واقعیت‌های پرونده از همان رکورد خوانده می‌شود تا با اعمال مصوبه گم نشود.
	for _, key := range []string{"relation", "cardNo", "issuerFa", "issueDate", "benefits"} {
		if value := strings.TrimSpace(fileRec.Fields[key]); value != "" {
			fields[key] = value
		}
	}
	// پروندهٔ تأییدشده برای ویرایش قفل است؛ اعمال مصوبه آن را با یادداشت باز می‌کند تا
	// مسیر «بازگشت → به‌روزرسانی → تأیید دوباره» در سابقهٔ پرونده بماند.
	steps := []string{}
	current := fileRec
	if current.Status == "approved" {
		reopened, err := svc.Act(ctx, current.ID, contracts.RecordAction{
			Action: "reopen",
			NoteFa: "بازگشایی برای اعمال مصوبهٔ کمیتهٔ ایثارگری " + rec.ID,
		}, actor)
		if err != nil {
			return nil, "", err
		}
		current = reopened
		steps = append(steps, "reopen")
	}
	updated, err := svc.Update(ctx, current.ID, contracts.RecordInput{
		Kind: domain.KindVeteranRecords, Title: current.Title, EmployeeID: employeeID,
		Fields: fields, NoteFa: "امتیازها بر پایهٔ مصوبهٔ کمیتهٔ ایثارگری " + rec.ID + " به‌روزرسانی شد",
	}, actor)
	if err != nil {
		return nil, "", err
	}
	// پرونده باید به «تأییدشده» برسد تا اثر مالی بگیرد.
	if updated.Status == "draft" {
		if next, err := svc.Act(ctx, updated.ID, contracts.RecordAction{Action: "verify"}, actor); err == nil {
			updated = next
			steps = append(steps, "verify")
		}
	}
	if updated.Status == "verified" {
		if next, err := svc.Act(ctx, updated.ID, contracts.RecordAction{
			Action: "approve", NoteFa: "تأیید بر پایهٔ مصوبهٔ کمیتهٔ ایثارگری " + rec.ID,
		}, actor); err == nil {
			updated = next
			steps = append(steps, "approve")
		}
	}
	return map[string]any{
		"committeeId":      rec.ID,
		"fileId":           updated.ID,
		"statusFa":         updated.StatusFa,
		"steps":            steps,
		"changes":          changes,
		"taxExempt":        merged.TaxExempt,
		"pensionExempt":    merged.PensionExempt,
		"extraMonths":      merged.TotalExtraServiceMonths,
		"incentiveSteps":   merged.IncentiveSteps,
		"leaveDays":        merged.LeaveDays,
		"educationUpgrade": merged.EducationUpgrade,
		"summaryFa":        merged.SummaryFa,
	}, "", nil
}

// committeePreview اثر مصوبه را روی پیش‌فرض قانونی نشان می‌دهد (بدون تغییر داده).
func committeePreview(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, employeeID string, fields map[string]string) map[string]any {
	_, rec, found := veteranFileOf(ctx, svc, employeeID)
	if !found {
		return map[string]any{"hasFile": false}
	}
	params := rules.VeteranParamsFromValues(veteranParamValues(deps))
	base := rules.VeteranEntitlementOfFile(rec.Fields, rec.Status == "approved", params)
	decision := rules.VeteranDecisionFromFields(fields)
	merged, changes := rules.ApplyVeteranDecision(base, decision)
	return map[string]any{
		"hasFile":   true,
		"base":      base,
		"merged":    merged,
		"changes":   changes,
		"decisions": committeeDecisionLines(fields),
	}
}

// committeeFields فیلدهای رکورد مصوبه را از ورودی فرم می‌سازد.
func committeeFields(in veteranCommitteeRequest, group string) map[string]string {
	num := func(v float64) string {
		if v <= 0 {
			return "0"
		}
		return floatField(v)
	}
	return map[string]string{
		"committeeNo":             strings.TrimSpace(in.CommitteeNo),
		"meetingDate":             calendar.NormalizeDigits(strings.TrimSpace(in.MeetingDate)),
		"members":                 strings.TrimSpace(in.Members),
		"veteranGroup":            group,
		"basisDocs":               strings.TrimSpace(in.BasisDocs),
		"eduUpgradeDecision":      strings.TrimSpace(in.EduUpgradeDecision),
		"eduLevel":                strings.TrimSpace(in.EduLevel),
		"taxExemptDecision":       strings.TrimSpace(in.TaxExemptDecision),
		"taxExemptPercent":        num(in.TaxExemptPercent),
		"pensionExemptDecision":   strings.TrimSpace(in.PensionExemptDecision),
		"pensionExemptPercent":    num(in.PensionExemptPercent),
		"workHourReduction":       strings.TrimSpace(in.WorkHourReduction),
		"reducedWeeklyHours":      num(in.ReducedWeeklyHours),
		"incentiveSteps":          num(in.IncentiveSteps),
		"seniorityMonths":         num(in.SeniorityMonths),
		"seniorityBasis":          strings.TrimSpace(in.SeniorityBasis),
		"leaveDays":               num(in.LeaveDays),
		"employmentQuotaDecision": strings.TrimSpace(in.EmploymentQuotaDecision),
		"otherBenefits":           strings.TrimSpace(in.OtherBenefits),
		"decisionFa":              strings.TrimSpace(in.DecisionFa),
		"effectiveDate":           calendar.NormalizeDigits(strings.TrimSpace(in.EffectiveDate)),
		"signOff":                 strings.TrimSpace(in.SignOff),
	}
}

// decisionLogFa متن تغییرها و بندهای مصوبه را برای ستون «سطرهای محاسبه» می‌سازد.
func decisionLogFa(fields map[string]string, changes []string) string {
	parts := []string{}
	for _, line := range committeeDecisionLines(fields) {
		parts = append(parts, line.TitleFa+": "+line.ValueFa)
	}
	for _, change := range changes {
		parts = append(parts, change)
	}
	if len(parts) == 0 {
		return " بدون تغییر نسبت به پیش‌فرض قانونی"
	}
	return " " + strings.Join(parts, "؛ ")
}

// committeeDecisionLines بندهای مصوبه به شکل سطرهای خوانا (برای پیش‌نمایش و پرونده).
func committeeDecisionLines(fields map[string]string) []rules.VeteranLine {
	return rules.VeteranDecisionLines(rules.VeteranDecisionFromFields(fields))
}
