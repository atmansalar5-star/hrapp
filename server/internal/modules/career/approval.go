// پیاده‌سازی ماژول «انتصابات، ترفیعات و کارراههٔ شغلی» — تصویب‌خواهی و اجرا.
//
// چرا این فایل: فرم تصویب‌خواهی باید پیش از ثبت، سه چیز را نشان دهد: شرط‌های سنجیده‌شده
// (کدام شرط مانع است)، زنجیرهٔ تصویب لازم به ترتیب، و مراحل اجرا. محاسبهٔ شرط‌ها در موتور
// قواعد است (`rules.PlanApproval`) و این فایل فقط پارامترهای شرکت را می‌خواند، پست‌های
// سازمانی را از پورت ماژول ساختار سازمانی می‌گیرد و اجرا را به گردش‌کار رکورد می‌سپارد؛
// پس هیچ قاعده‌ای در رابط وب نمی‌نشیند و فرم، جدول رکوردها و اپلیکیشن یک جواب می‌دهند.

package career

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"

	"hrms/internal/modules/career/domain"
)

// registerApprovalEndpoints مسیرهای تصویب‌خواهی و اجرای انتصاب/ترفیع را ثبت می‌کند.
func registerApprovalEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("career", "read", fn) }
	approve := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("career", "approve", fn) }

	// پارامترهای فرم: آستانه‌های تصویب، نردبان گروه شغلی، انواع درخواست، پست‌های سازمانی،
	// کارکنان و تاریخ امروز (شمسی، از ساعت سرور).
	mux.HandleFunc("GET /api/v1/career/appointment-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, approvalDefaults(r.Context(), deps))
	}))

	// تصویب‌خواهی: شرط‌ها و زنجیرهٔ تصویب را می‌سازد (بدون تغییر داده).
	mux.HandleFunc("POST /api/v1/career/appointment-plan", read(func(w http.ResponseWriter, r *http.Request) {
		var in rules.AppointmentInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی تصویب‌خواهی خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, approvalPlan(deps, in))
	}))

	// اجرا: پیش از هر تغییری، شرط‌ها با پارامترهای شرکت دوباره سنجیده می‌شود و تنها اگر
	// مانعی نباشد اقدام «اجرا و ابلاغ» روی همان رکورد انجام می‌گیرد.
	mux.HandleFunc("POST /api/v1/career/appointment-execute", approve(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID     string `json:"id"`
			NoteFa string `json:"noteFa"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"شناسهٔ درخواست خوانده نشد.", nil)
			return
		}
		id := strings.TrimSpace(body.ID)
		if id == "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"شناسهٔ درخواست انتصاب یا ترفیع الزامی است.", nil)
			return
		}
		rec, ok, err := svc.Get(r.Context(), contracts.RecordFilter{Module: "career", Scope: "all"}, id)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "درخواست یافت نشد.", nil)
			return
		}
		plan := rules.PlanApproval(inputFromRecord(rec), approvalParams(deps))
		if !plan.Allowed {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
				"اجرای ابلاغ ممکن نیست: "+blockingSummaryFa(plan), map[string]string{
					"blockingCount": strconv.Itoa(plan.BlockingCount),
					"summary":       plan.SummaryFa,
				})
			return
		}

		actor := actorOf(r)
		steps := executeSteps(r.Context(), deps, rec, actor)
		if failed, ok := firstFailedStep(steps); ok {
			// هیچ تغییری اعمال نشده است؛ رکورد دست‌نخورده می‌ماند تا کاربر بتواند
			// نقص را برطرف کند و دوباره اجرا بزند.
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
				"اجرای ابلاغ نیمه‌کاره ماند: "+failed.TitleFa+" — "+failed.MessageFa, nil)
			return
		}

		updated, err := svc.Act(r.Context(), id, contracts.RecordAction{Action: "issue", NoteFa: body.NoteFa}, actor)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"record": updated, "plan": plan, "steps": steps,
			"noteFa": "ابلاغ صادر و اجرا شد؛ پست و واحد کارمند در پرونده و تصدی پست در تشکیلات ثبت شد.",
		})
	}))
}

// ExecuteStep اجرای یک گام از گام‌های «اجرا و ابلاغ» و نتیجهٔ واقعی آن.
//
// چرا این ساختار: گام‌های اجرا (`domain.ExecutionSteps`) اعلام رویه‌اند؛ ولی گزارش اجرا
// باید بگوید کدام گام واقعاً انجام شد. تا پیش از این، اجرا فقط وضعیت رکورد را عوض می‌کرد
// و بقیهٔ گام‌ها فقط در متن دیده می‌شدند.
type ExecuteStep struct {
	Order   int    `json:"order"`
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	Done    bool   `json:"done"`
	// Blocking یعنی بی‌این گام، اجرای ابلاغ معنا ندارد و باید با خطا برگردد. گام‌های
	// بعدی (تشکیلات، حقوق، بایگانی) اگر انجام نشوند، ابلاغ اجرا شده است ولی گزارش
	// اجرا همان گام را با پیام روشن «انجام‌نشده» نشان می‌دهد.
	Blocking  bool   `json:"blocking,omitempty"`
	DetailFa  string `json:"detailFa,omitempty"`
	MessageFa string `json:"messageFa,omitempty"`
}

// executeSteps گام‌های اجرا را واقعاً انجام می‌دهد و نتیجهٔ هر گام را برمی‌گرداند.
//
// ترتیب: صدور ابلاغ (تغییر سازمانی پروندهٔ کارمند) ← ثبت تصدی پست در تشکیلات ← اعلام به
// حقوق و دستمزد (کشف از پروندهٔ تازه‌شده) ← اطلاع و بایگانی (از رخدادهای همین رکورد).
func executeSteps(ctx context.Context, deps modulekit.Deps, rec contracts.Record, actor string) []ExecuteStep {
	assignment := assignmentFromRecord(rec)

	// گام ۱ و ۲: پست و واحد کارمند از تاریخ اجرا در پروندهٔ او ثبت می‌شود.
	employeeStep := ExecuteStep{Order: 1, Key: "decree", TitleFa: "صدور ابلاغ", Blocking: true}
	assigner, ok := deps.Ports.EmployeeAssigner()
	if !ok {
		employeeStep.MessageFa = "پورت به‌روزرسانی پروندهٔ کارمند در دسترس نیست؛ ابلاغ در پرونده ثبت نشد."
		return []ExecuteStep{employeeStep}
	}
	result, err := assigner.Assign(ctx, rec.EmployeeID, assignment, actor)
	if err != nil {
		employeeStep.MessageFa = "ثبت تغییر سازمانی در پروندهٔ کارمند انجام نشد: " + err.Error()
		return []ExecuteStep{employeeStep}
	}
	employeeStep.Done = true
	employeeStep.DetailFa = "پست تازه: " + result.Employee.Position + " — واحد: " + result.Employee.DepartmentName
	if !result.PositionChanged && !result.DepartmentChanged {
		employeeStep.DetailFa = "پرونده از قبل با ابلاغ هم‌خوان بود؛ تغییری لازم نداشت."
	}
	if !result.DepartmentMatched && assignment.DepartmentName != "" {
		employeeStep.DetailFa += " (واحد «" + assignment.DepartmentName + "» در فهرست واحدها پیدا نشد.)"
	}
	steps := []ExecuteStep{
		employeeStep,
		{Order: 2, Key: "history", TitleFa: "ثبت سابقهٔ کارمند", Done: true,
			DetailFa: "ابلاغ " + firstNonEmpty(assignment.DecreeNo, "—") + " از تاریخ " +
				firstNonEmpty(assignment.EffectiveDate, "—") + " در سابقهٔ کارمند ثبت شد."},
	}

	// گام ۳: تصدی پست در تشکیلات (نمودار سازمانی متصدی تازه را از همین رکورد می‌خواند).
	// این گام فقط پس از موفقیت گام نخست اجرا می‌شود تا مجموعه نیمه‌به‌روز نماند.
	orgStep := ExecuteStep{Order: 3, Key: "orgchart", TitleFa: "به‌روزرسانی تشکیلات"}
	if applier, ok := deps.Ports.AppointmentApplier(); !ok {
		orgStep.MessageFa = "پورت ثبت تصدی در تشکیلات در دسترس نیست؛ نمودار سازمانی به‌روز نشد."
	} else if orgResult, err := applier.Appoint(ctx, appointmentFromRecord(rec, assignment), actor); err != nil {
		orgStep.MessageFa = "ثبت تصدی در تشکیلات انجام نشد: " + err.Error()
	} else {
		orgStep.Done = true
		orgStep.DetailFa = orgResult.NoteFa
		if orgResult.AssignmentID != "" {
			orgStep.DetailFa = "انتصاب " + orgResult.AssignmentID + " — " + orgResult.NoteFa
		}
	}
	steps = append(steps, orgStep)

	// گام ۴ و ۵: بقیهٔ ماژول‌ها از خود پرونده و همین رکورد می‌خوانند؛ پس تغییر تازه
	// همان لحظه در دورهٔ حقوق و در کارتابل اطلاع دیده می‌شود و سند تازه‌ای لازم نیست.
	steps = append(steps,
		ExecuteStep{Order: 4, Key: "payroll", TitleFa: "اعلام به حقوق و دستمزد", Done: employeeStep.Done,
			DetailFa: "پست و واحد تازه در پروندهٔ کارمند نشسته است؛ دورهٔ حقوق همان ماه از همین پرونده خوانده می‌شود."},
		ExecuteStep{Order: 5, Key: "notify", TitleFa: "اطلاع به کارمند و بایگانی", Done: true,
			DetailFa: "سند ابلاغ با شماره و تاریخ اجرا در رکورد و سابقهٔ کارمند بایگانی شد."},
	)
	return steps
}

// firstFailedStep نخستین گام «مانع»ی که انجام نشد.
//
// فقط گام‌های مانع اجرا را متوقف می‌کنند؛ اگر گام تشکیلات یا اعلام به حقوق انجام نشود،
// ابلاغ اجرا شده است و کاربر باید در گزارش ببیند کدام گام ناتمام مانده — پنهان‌کردن آن،
// همان اشتباهی است که پیش‌تر رخ می‌داد.
func firstFailedStep(steps []ExecuteStep) (ExecuteStep, bool) {
	for _, step := range steps {
		if step.Blocking && !step.Done {
			return step, true
		}
	}
	return ExecuteStep{}, false
}

// assignmentFromRecord تغییر سازمانی پروندهٔ کارمند را از فیلدهای ابلاغ می‌سازد.
func assignmentFromRecord(rec contracts.Record) contracts.EmployeeAssignment {
	fields := rec.Fields
	position := strings.TrimSpace(fields["position"])
	if rec.Kind == domain.KindPromotions {
		position = firstNonEmpty(fields["toPosition"], position)
	}
	return contracts.EmployeeAssignment{
		Position:       position,
		DepartmentName: strings.TrimSpace(fields["unit"]),
		DecreeNo:       strings.TrimSpace(fields["decreeNo"]),
		EffectiveDate:  strings.TrimSpace(fields["effectiveDate"]),
		ReasonFa:       strings.TrimSpace(fields["reason"]),
	}
}

// appointmentFromRecord ورودی ثبت تصدی در تشکیلات را از ابلاغ می‌سازد.
func appointmentFromRecord(rec contracts.Record, assignment contracts.EmployeeAssignment) contracts.PositionAppointment {
	return contracts.PositionAppointment{
		EmployeeID:    rec.EmployeeID,
		PositionCode:  strings.TrimSpace(rec.Fields["positionCode"]),
		PositionTitle: assignment.Position,
		Unit:          assignment.DepartmentName,
		StartDate:     assignment.EffectiveDate,
		DecreeNo:      assignment.DecreeNo,
		// «دلیل و ملاک تصویب» همان شواهد احراز است؛ ابلاغی که زنجیرهٔ تصویب را
		// گذرانده، شرطهای احراز را برآورده کرده است.
		Evidence:   assignment.ReasonFa,
		Commitment: intOf(rec.Fields["commitment"]),
		NoteFa:     rec.Title,
	}
}

// firstNonEmpty نخستین مقدار ناتهی.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// actorOf نام کاربر جاری برای ثبت در رخدادهای رکورد.
func actorOf(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok {
		return user.Username
	}
	return "ناشناس"
}

// approvalDefaults پارامترها و فهرست‌های مرجع فرم تصویب‌خواهی.
func approvalDefaults(ctx context.Context, deps modulekit.Deps) map[string]any {
	params := approvalParams(deps)
	today := ""
	if now, err := calendar.FromGregorian(deps.Clock()); err == nil {
		today = now.String()
	}
	return map[string]any{
		"kinds":       rules.AppointmentKinds(),
		"grades":      rules.Grades(),
		"params":      params,
		"paramViews":  approvalParamViews(deps),
		"today":       today,
		"yesNo":       []map[string]string{{"key": "yes", "titleFa": "بله"}, {"key": "no", "titleFa": "خیر"}},
		"employees":   employeeOptions(ctx, deps),
		"positions":   positionOptions(ctx, deps),
		"noteFa":      approvalNoteFa(),
		"company":     deps.Org.TitleFa,
		"chainNoteFa": "زنجیرهٔ تصویب از مدیر واحد آغاز می‌شود و با بالا رفتن گروه شغلی، کمیتهٔ انتصابات و مدیرعامل اضافه می‌شوند.",
	}
}

// approvalPlan محاسبهٔ تصویب‌خواهی با پارامترهای شرکت.
func approvalPlan(deps modulekit.Deps, in rules.AppointmentInput) map[string]any {
	plan := rules.PlanApproval(in, approvalParams(deps))
	out := map[string]any{
		"plan": plan, "input": in, "params": approvalParams(deps), "noteFa": approvalNoteFa(),
		// گام‌های اجرا از دامنه می‌آید تا فرم و مستند استقرار یک ترتیب داشته باشند.
		"execution": domain.ExecutionSteps(in.Kind),
	}
	// پایان تعهد خدمت در سرور حساب می‌شود (تقویم شمسی)؛ رابط هیچ تبدیلی انجام نمی‌دهد.
	if end := commitmentEnd(in.EffectiveDate, in.CommitmentMonths); end != "" {
		out["commitmentEnd"] = end
	}
	if in.EvaluationRecorded {
		out["evaluationPercent"] = rules.EvaluationPercent(in.EvaluationScore)
	}
	return out
}

// approvalParams آستانه‌های تصویب را از پروفایل شرکت فعال می‌خواند.
func approvalParams(deps modulekit.Deps) rules.ApprovalParams {
	params := rules.DefaultApprovalParams()
	if v, ok := numericParam(deps, "appointment.minServiceMonths"); ok && v > 0 {
		params.MinServiceMonths = int(v)
	}
	if v, ok := numericParam(deps, "appointment.minEvaluationScore"); ok && v > 0 {
		params.MinEvaluationScore = v
	}
	if v, ok := numericParam(deps, "appointment.maxGradeStep"); ok && v > 0 {
		params.MaxGradeStep = int(v)
	}
	if v, ok := numericParam(deps, "appointment.commitmentMonths"); ok && v > 0 {
		params.CommitmentMonths = int(v)
	}
	if v, ok := numericParam(deps, "appointment.committeeRank"); ok && v > 0 {
		params.CommitteeRank = int(v)
	}
	if v, ok := numericParam(deps, "appointment.ceoRank"); ok && v > 0 {
		params.CEORank = int(v)
	}
	if text := stringParam(deps, "appointment.committeeForPromotion"); text != "" {
		params.CommitteeForPromotion = text != "0" && !strings.EqualFold(text, "no")
	}
	return params
}

// approvalParamViews پارامترهای تصویب را برای نمایش در فرم برمی‌گرداند.
func approvalParamViews(deps modulekit.Deps) []map[string]string {
	out := []map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "appointment.") {
			continue
		}
		out = append(out, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa,
		})
	}
	return out
}

// numericParam مقدار عددی یک پارامتر استقرار.
func numericParam(deps modulekit.Deps, key string) (float64, bool) {
	for _, p := range deps.Org.Params {
		if p.Key != key {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64)
		if err != nil {
			return 0, false
		}
		return value, true
	}
	return 0, false
}

// stringParam مقدار متنی یک پارامتر استقرار.
func stringParam(deps modulekit.Deps, key string) string {
	for _, p := range deps.Org.Params {
		if p.Key == key {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

// employeeOptions فهرست کارکنان برای انتخاب در فرم (از پورت ماژول کارکنان).
func employeeOptions(ctx context.Context, deps modulekit.Deps) []map[string]string {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return []map[string]string{}
	}
	list, err := reader.List(ctx, contracts.EmployeeFilter{Size: 500})
	if err != nil {
		return []map[string]string{}
	}
	out := make([]map[string]string, 0, len(list.Items))
	for _, e := range list.Items {
		out = append(out, map[string]string{
			"id": e.ID, "titleFa": e.FullName + " — " + e.ID,
			"position": e.Position, "unit": e.DepartmentName,
			"employmentType": e.EmploymentType, "hireDate": e.HireDate,
		})
	}
	return out
}

// positionOptions پست‌های سازمانی و ظرفیت آن‌ها (از پورت ماژول ساختار سازمانی).
//
// چرا از پورت و نه انبار: مالک پست‌ها ماژول ساختار سازمانی است و این ماژول فقط
// فهرست را می‌خواند؛ مرز ماژول شکسته نمی‌شود.
func positionOptions(ctx context.Context, deps modulekit.Deps) []map[string]string {
	port, ok := deps.Ports.RecordList("orgstructure")
	if !ok {
		return []map[string]string{}
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: "orgstructure", Kind: "positions", Scope: "all", Size: 500,
	})
	if err != nil {
		return []map[string]string{}
	}
	out := make([]map[string]string, 0, len(page.Items))
	for _, rec := range page.Items {
		code := strings.TrimSpace(rec.Fields["code"])
		if code == "" {
			continue
		}
		vacant := rec.Status == "vacant" || rec.Status == "active"
		out = append(out, map[string]string{
			"code": code, "titleFa": rec.Title + " (" + code + ")",
			"unit": rec.Fields["unit"], "grade": rec.Fields["grade"],
			"headcount": rec.Fields["headcount"], "status": rec.Status,
			"vacancy": boolTextFa(vacant),
		})
	}
	return out
}

// boolTextFa «بله/خیر» فارسی برای نمایش.
func boolTextFa(v bool) string {
	if v {
		return "بله"
	}
	return "خیر"
}

// inputFromRecord ورودی موتور قواعد را از فیلدهای یک رکورد می‌سازد.
//
// چرا از رکورد: اجرا روی رکورد ذخیره‌شده انجام می‌شود و همان عددی که تصویب شده باید
// مبنای سنجش قرار گیرد؛ اگر رابط چیزی بفرستد، مبنای اجرا با مبنای تصویب یکی نیست.
func inputFromRecord(rec contracts.Record) rules.AppointmentInput {
	fields := rec.Fields
	kind := "appointment"
	if rec.Kind == domain.KindPromotions {
		kind = "promotion"
	}
	return rules.AppointmentInput{
		Kind:               kind,
		CurrentRank:        rules.GradeRankOf(fields["currentGrade"]),
		ProposedRank:       rules.GradeRankOf(fields["proposedGrade"]),
		ServiceMonths:      intOf(fields["serviceMonths"]),
		EvaluationScore:    floatOf(fields["evaluationScore"]),
		EvaluationRecorded: floatOf(fields["evaluationScore"]) > 0,
		VacancyAvailable:   strings.TrimSpace(fields["vacancy"]) != "no",
		VacancyChecked:     strings.TrimSpace(fields["vacancy"]) != "",
		CommitmentMonths:   intOf(fields["commitment"]),
		EffectiveDate:      strings.TrimSpace(fields["effectiveDate"]),
	}
}

// intOf عدد صحیح یک فیلد متنی.
func intOf(value string) int {
	num, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return int(floatOf(value))
	}
	return num
}

// floatOf عدد اعشاری یک فیلد متنی.
func floatOf(value string) float64 {
	num, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return num
}

// blockingSummaryFa مانع‌ها را برای پیام خطا جمع می‌کند.
func blockingSummaryFa(plan rules.ApprovalPlan) string {
	parts := []string{}
	for _, c := range plan.Checks {
		if !c.Passed && c.Blocking {
			parts = append(parts, c.TitleFa)
		}
	}
	if len(parts) == 0 {
		return plan.SummaryFa
	}
	return strings.Join(parts, "، ")
}

// commitmentEnd پایان تعهد خدمت را با تقویم شمسی حساب می‌کند (شروع + ماه‌ها).
//
// تبدیل تقویمی فقط در لایهٔ کد انجام می‌شود؛ رابط هیچ تاریخ‌ای را خودش جمع نمی‌زند.
func commitmentEnd(effectiveDate string, months int) string {
	if months <= 0 {
		return ""
	}
	start, err := calendar.Parse(strings.TrimSpace(effectiveDate))
	if err != nil {
		return ""
	}
	year := start.Year
	month := start.Month + months
	for month > 12 {
		month -= 12
		year++
	}
	day := start.Day
	if length, err := calendar.MonthLength(year, month); err == nil && day > length {
		day = length
	}
	end, err := calendar.Parse(itoa(year) + "/" + pad2(month) + "/" + pad2(day))
	if err != nil {
		return ""
	}
	return end.String()
}

// itoa و pad2 ساخت عدد و ماه دورقمی برای تجزیهٔ تاریخ شمسی.
func itoa(v int) string { return strconv.Itoa(v) }

func pad2(v int) string {
	if v < 10 {
		return "0" + strconv.Itoa(v)
	}
	return strconv.Itoa(v)
}

// approvalNoteFa توضیح قاعده‌ها برای فرم؛ متن از سرور می‌آید تا در رابط تکرار نشود.
func approvalNoteFa() string {
	return "تصویب‌خواهی پیش از ثبت سنجیده می‌شود: تاریخ اجرا، گروه شغلی فعلی و پیشنهادی، " +
		"حداقل سابقهٔ گروه (پارامتر appointment.minServiceMonths)، امتیاز ارزیابی " +
		"(appointment.minEvaluationScore)، اندازهٔ جهش گروه (appointment.maxGradeStep)، " +
		"ظرفیت پست و تعهد خدمت گروه‌های مدیریتی (appointment.commitmentMonths).\n" +
		"زنجیرهٔ تصویب: مدیر واحد ← مدیر منابع انسانی ← کمیتهٔ انتصابات و ترفیعات " +
		"(از پلهٔ appointment.committeeRank) ← مدیرعامل (از پلهٔ appointment.ceoRank).\n" +
		"اجرا فقط پس از تصویب و با ثبت شمارهٔ ابلاغ ممکن است و پنج گام را واقعاً انجام می‌دهد: " +
		"صدور ابلاغ و ثبت پست و واحد تازه در پروندهٔ کارمند، ثبت سابقه، ثبت تصدی پست در " +
		"تشکیلات (انتصاب تازه در نمودار سازمانی و پایان تصدی پیشین)، اعلام به حقوق و دستمزد " +
		"و بایگانی سند. گزارش هر گام در پاسخ اجرا برمی‌گردد؛ اگر گامی ناتمام بماند، همان‌جا با " +
		"پیام روشن نشان داده می‌شود و رکورد ابلاغ‌نشده نمی‌ماند."
}
