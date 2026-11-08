// گردش‌کار تصویب و ابطال ابلاغ انتصاب و ترفیع — امضای سطح، بازگشت و ابطال.
//
// چرا این فایل جدا از `approval.go`: آن فایل تصویب‌خواهی و اجرا را می‌سازد؛ اینجا
// اقدام‌های میان راه است. هر سه مسیر قاعده‌سنج‌اند: زنجیرهٔ تصویب از روی خودِ رکورد و
// پارامترهای شرکت دوباره ساخته می‌شود و به مقدار ذخیره‌شده در فیلدها تکیه نمی‌کند؛ پس
// اگر پارامترهای شرکت عوض شوند، سطح‌های باقی‌مانده با پارامتر روز سنجیده می‌شوند.
package career

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/career/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// approvalBody بدنهٔ مشترک مسیرهای امضا، بازگشت و ابطال.
type approvalBody struct {
	ID     string `json:"id"`
	NoteFa string `json:"noteFa"`
}

// approvalRequestBody شناسهٔ درخواست و یادداشت را از بدنه می‌خواند؛ خطا را خودش می‌نویسد.
func approvalRequestBody(w http.ResponseWriter, r *http.Request) (string, approvalBody, bool) {
	var body approvalBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
			"بدنهٔ درخواست خوانده نشد.", nil)
		return "", body, false
	}
	id := strings.TrimSpace(body.ID)
	if id == "" {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
			"شناسهٔ درخواست انتصاب یا ترفیع الزامی است.", nil)
		return "", body, false
	}
	return id, body, true
}

// signApprovalLevel یک سطح از زنجیرهٔ تصویب را امضا می‌کند.
//
// قاعده‌ها (همه در همین‌جا اعلام و در سرور سنجیده می‌شوند، نه در رابط):
//
//	۱. امضا تنها در وضعیت «در انتظار تصویب» و برای درخواست بی‌مانع ممکن است.
//	۲. سطح جاری از زنجیرهٔ تصویب همین رکورد و پارامترهای شرکت خوانده می‌شود.
//	۳. نقش امضاکننده باید همان نقش آن سطح باشد (مدیر سیستم می‌تواند جانشین شود).
//	۴. یک نفر دو سطح پیاپی را امضا نمی‌کند؛ وگرنه «تصویب چندسطحی» در عمل یک امضا است.
//	۵. آخرین امضا تصویب نهایی را می‌بندد و رکورد به «تصویب‌شده» می‌رود؛ پیش از آن
//	   هیچ راهی برای رسیدن به «تصویب‌شده» وجود ندارد.
func signApprovalLevel(w http.ResponseWriter, r *http.Request, deps modulekit.Deps,
	svc *recordkit.Service, id, note string) {
	rec, ok, err := svc.Get(r.Context(), contracts.RecordFilter{Module: "career", Scope: "all"}, id)
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}
	if !ok {
		httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "درخواست یافت نشد.", nil)
		return
	}
	if rec.Status != domain.StatusPending {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"امضای سطح تصویب تنها برای درخواست «در انتظار تصویب» ممکن است.", map[string]string{"status": rec.Status})
		return
	}
	plan := appointmentPlan(r.Context(), deps, svc, inputFromRecord(rec))
	if !plan.Allowed {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"امضا ممکن نیست؛ درخواست مانع دارد: "+blockingSummaryFa(plan), map[string]string{
				"blockingCount": strconv.Itoa(plan.BlockingCount),
				"summary":       plan.SummaryFa,
			})
		return
	}
	total := len(plan.Approvals)
	if total == 0 {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"این درخواست سطح تصویب ندارد.", nil)
		return
	}
	progress := intOf(rec.Fields[domain.FieldApprovalProgress])
	if progress < 0 {
		progress = 0
	}
	if progress >= total {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"همهٔ سطح‌های تصویب امضا شده‌اند؛ گام بعدی اجرای ابلاغ است.", map[string]string{
				"progress": strconv.Itoa(progress), "total": strconv.Itoa(total),
			})
		return
	}

	actor := actorOf(r)
	user, _ := auth.UserFrom(r.Context())
	level := plan.Approvals[progress]
	if !roleMatchesLevel(user.Role, level.RoleKey) {
		httpx.WriteErrorMsg(w, http.StatusForbidden, contracts.CodeForbidden,
			"سطح «"+level.TitleFa+"» باید با نقش "+roleTitleFa(level.RoleKey)+" امضا شود."+
				" این سطح را با همان نقش انجام دهید.", map[string]string{
				"level": level.Key, "role": level.RoleKey,
			})
		return
	}
	// تفکیک وظایف: یک نفر نباید با دو نقش متفاوت دو سطح پیاپی را ببندد؛ ولی اگر دو سطح
	// پیاپی همان نقش را داشته باشند (کمیته و سطح مدیرعامل در این استقرار)، همان دارندهٔ
	// نقش امضا می‌کند وگرنه تصویب چندسطحی هرگز کامل نمی‌شد.
	if lastRole, lastUser := lastSignature(rec.Fields[domain.FieldApprovalLog]); lastUser != "" &&
		lastUser == actor && lastRole != level.RoleKey {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"همین کاربر سطح پیشین را با نقش دیگری امضا کرده است؛ برای حفظ تفکیک وظایف، "+
				"این سطح را با نقش "+roleTitleFa(level.RoleKey)+" فرد دیگری امضا کند.", nil)
		return
	}

	nextProgress := progress + 1
	fields := map[string]string{
		domain.FieldApprovalTotal:    strconv.Itoa(total),
		domain.FieldApprovalProgress: strconv.Itoa(nextProgress),
		domain.FieldApprovalLog:      appendSignature(rec.Fields[domain.FieldApprovalLog], level, actor, todayFa(deps)),
		domain.FieldApprovalLevels:   levelTitles(plan),
	}
	action, status := "sign", domain.StatusPending
	if nextProgress >= total {
		action, status = "approve", domain.StatusApproved
	}
	updated, err := svc.Act(r.Context(), id, contracts.RecordAction{Action: action, NoteFa: note, Fields: fields}, actor)
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
		return
	}
	noteFa := "سطح «" + level.TitleFa + "» امضا شد؛ " + strconv.Itoa(nextProgress) + " از " +
		strconv.Itoa(total) + " سطح کامل است."
	if status == domain.StatusApproved {
		noteFa = "آخرین سطح تصویب امضا شد؛ درخواست تصویب‌شده و آمادهٔ اجرای ابلاغ است."
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"record": updated, "level": level, "progress": nextProgress, "total": total, "noteFa": noteFa,
	})
}

// returnApproval درخواست را برای اصلاح به پیش‌نویس برمی‌گرداند.
//
// چرا لازم است: پیش از این تنها راه «نه گفتن»، لغو بود و درخواست با یک نقص کوچک می‌سوخت.
// اکنون بازبین دلیل را می‌نویسد، پیشرفت تصویب صفر می‌شود و درخواست به دست درخواست‌دهنده
// برمی‌گردد؛ رکورد و سابقه پاک نمی‌شوند.
func returnApproval(w http.ResponseWriter, r *http.Request, svc *recordkit.Service, id, note string) {
	reason := strings.TrimSpace(note)
	if reason == "" {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
			"دلیل بازگشت برای اصلاح را بنویسید؛ درخواست بی‌دلیل به پیش‌نویس برنمی‌گردد.", nil)
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
	if rec.Status != domain.StatusPending && rec.Status != domain.StatusApproved {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"بازگشت برای اصلاح تنها پیش از اجرای ابلاغ ممکن است.", map[string]string{"status": rec.Status})
		return
	}
	fields := map[string]string{
		domain.FieldApprovalProgress: "0",
		domain.FieldApprovalLog: appendSignature(rec.Fields[domain.FieldApprovalLog],
			rules.ApprovalStep{Key: "return", TitleFa: "بازگشت برای اصلاح"}, actorOf(r), ""),
	}
	updated, err := svc.Act(r.Context(), id, contracts.RecordAction{Action: "return", NoteFa: reason, Fields: fields}, actorOf(r))
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"record": updated, "noteFa": "درخواست به پیش‌نویس بازگشت؛ پیشرفت تصویب صفر شد و دلیل در سابقه ماند.",
	})
}

// revokeDecree ابلاغ اجراشده را باطل می‌کند: تصدی تشکیلات بسته و پروندهٔ کارمند برمی‌گردد.
//
// چرا لازم است: وضعیت «ابلاغ‌شده» پایانی بود و ابلاغ اشتباه فقط با تغییر دستی داده جبران
// می‌شد. ترتیب عمدی است: نخست تصدی بسته می‌شود (تا تشکیلات متصدی تازه نشان ندهد)، سپس
// پروندهٔ کارمند به وضع پیشین برمی‌گردد و در پایان رکورد «ابطال‌شده» می‌شود؛ اگر هر گام
// شکست بخورد، رکورد اجراشده می‌ماند تا وضعیت نیمه‌کاره ساخته نشود.
func revokeDecree(w http.ResponseWriter, r *http.Request, deps modulekit.Deps,
	svc *recordkit.Service, id, note string) {
	reason := strings.TrimSpace(note)
	if reason == "" {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
			"دلیل ابطال ابلاغ را بنویسید.", nil)
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
	if rec.Status != domain.StatusIssued {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"ابطال تنها برای ابلاغ اجراشده ممکن است؛ برای درخواست تصویب‌نشده از «لغو» استفاده کنید.",
			map[string]string{"status": rec.Status})
		return
	}
	actor := actorOf(r)
	assignmentID := strings.TrimSpace(rec.Fields[domain.FieldAssignmentID])
	if assignmentID == "" {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
			"این ابلاغ تصدی ثبت‌شده‌ای در تشکیلات ندارد؛ ابطال آن از مسیر «لغو» انجام می‌شود.", nil)
		return
	}

	// ۱) بستن تصدی در تشکیلات.
	if applier, ok := deps.Ports.AppointmentApplier(); ok {
		if _, err := applier.Release(r.Context(), contracts.PositionRelease{
			AssignmentID: assignmentID,
			EmployeeID:   rec.EmployeeID,
			DecreeNo:     strings.TrimSpace(rec.Fields["decreeNo"]),
			ReasonFa:     reason,
		}, actor); err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
				"بستن تصدی در تشکیلات انجام نشد: "+err.Error(), nil)
			return
		}
	}

	// ۲) برگرداندن پروندهٔ کارمند به وضع پیشین (اگر ابطال اطلاعات پیشین را داشته باشد).
	beforePosition := strings.TrimSpace(rec.Fields[domain.FieldBeforePosition])
	beforeUnit := strings.TrimSpace(rec.Fields[domain.FieldBeforeUnit])
	beforeGrade := strings.TrimSpace(rec.Fields[domain.FieldBeforeGrade])
	// گروه شغلی پیشین اگر خالی بوده، باید با نشانهٔ «-» پاک شود (فیلد خالی یعنی تغییر نده).
	if beforeGrade == "" {
		beforeGrade = "-"
	}
	if assigner, ok := deps.Ports.EmployeeAssigner(); ok && (beforePosition != "" || beforeUnit != "" || beforeGrade != "") {
		if _, err := assigner.Assign(r.Context(), rec.EmployeeID, contracts.EmployeeAssignment{
			Position:       beforePosition,
			DepartmentName: beforeUnit,
			JobGrade:       beforeGrade,
			DecreeNo:       strings.TrimSpace(rec.Fields["decreeNo"]),
			ReasonFa:       "ابطال ابلاغ: " + reason,
		}, actor); err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
				"برگرداندن پروندهٔ کارمند انجام نشد: "+err.Error(), nil)
			return
		}
	}

	// ۳) ابطال رکورد.
	updated, err := svc.Act(r.Context(), id, contracts.RecordAction{Action: "revoke", NoteFa: reason}, actor)
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"record": updated, "noteFa": "ابلاغ باطل شد؛ تصدی تشکیلات بسته و پروندهٔ کارمند به وضع پیشین برگشت.",
	})
}

/* ------------------------------ کمک‌تابع‌ها ------------------------------ */

// roleMatchesLevel آیا نقش کنشگر همان نقش سطح تصویب است (مدیر سیستم جانشین می‌شود).
func roleMatchesLevel(role contracts.Role, levelRole string) bool {
	if role == contracts.RoleAdmin {
		return true
	}
	return string(role) == strings.TrimSpace(levelRole)
}

// roleTitleFa عنوان فارسی یک نقش برای پیام خطا؛ کلید ناشناخته خودش برمی‌گردد.
func roleTitleFa(role string) string {
	switch strings.TrimSpace(role) {
	case string(contracts.RoleLineManager):
		return "مدیر واحد"
	case string(contracts.RoleHRManager):
		return "مدیر منابع انسانی"
	case string(contracts.RoleHRSpecialist):
		return "کارشناس منابع انسانی"
	case string(contracts.RoleAdmin):
		return "مدیر سیستم"
	case string(contracts.RoleEmployee):
		return "کارمند"
	default:
		return strings.TrimSpace(role)
	}
}

// levelTitles متن سطوح تصویب برای نمایش در رکورد.
func levelTitles(plan rules.ApprovalPlan) string {
	titles := make([]string, 0, len(plan.Approvals))
	for _, step := range plan.Approvals {
		titles = append(titles, step.TitleFa)
	}
	return strings.Join(titles, " ← ")
}

// appendSignature یک خط به سابقهٔ امضاها اضافه می‌کند: «سطح|نقش|کاربر|تاریخ».
//
// چرا نقش هم ذخیره می‌شود: سنجش تفکیک وظایف به نقش هر امضا نیاز دارد، وگرنه نمی‌شد فهمید
// دو سطح پیاپی یک نقش بوده‌اند یا دو نقش متفاوت.
func appendSignature(log string, level rules.ApprovalStep, actor, date string) string {
	parts := []string{level.TitleFa, strings.TrimSpace(level.RoleKey), actor, strings.TrimSpace(date)}
	line := strings.Join(parts, "|")
	existing := strings.TrimSpace(log)
	if existing == "" {
		return line
	}
	return existing + "\n" + line
}

// lastSignature نقش و کاربر آخرین امضا را از سابقه می‌خواند (خالی = امضایی نیست).
func lastSignature(log string) (string, string) {
	lines := strings.Split(strings.TrimSpace(log), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		parts := strings.Split(lines[i], "|")
		if len(parts) >= 3 {
			return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
		}
	}
	return "", ""
}

// todayFa تاریخ امروز شمسی برای ثبت در سابقهٔ امضا.
func todayFa(deps modulekit.Deps) string {
	now, err := calendar.FromGregorian(deps.Clock())
	if err != nil {
		return ""
	}
	return now.String()
}

// ensureContext استفاده از context در امضای توابع کمکی (برای خوانایی امضاها).
var _ = context.Background
