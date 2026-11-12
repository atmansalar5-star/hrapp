// مسیرهای «اعضای کمیتهٔ ایثارگری» — پنل تعریف اعضا و بلوک امضای پایین فرم.
//
// چرا این فایل جدا از `committee.go`: مصوبه «تصمیم» است و عضو کمیته «تعریف سازمانی». پنل
// اعضا یک فهرست مصوب می‌سازد (نام، نقش، سمت، شمارهٔ ابلاغ، ترتیب امضا، فعال/غیرفعال) که سه
// مصرف‌کننده دارد: پیش‌پر کردن فرم مصوبه، انتخاب امضاکنندگان پایین فرم، و ساخت بلوک امضای
// برگهٔ چاپی. نقش‌ها و ترکیب عنوان امضا در موتور قواعد (`internal/rules`) است.
package grievances

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/grievances/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registerVeteranCommitteeMemberEndpoints مسیرهای پنل «اعضای کمیتهٔ ایثارگری» را ثبت می‌کند.
// committeeMemberNoteFa دلیل ثبت‌شدهٔ تغییر وضعیت عضویت.
//
// چرا لازم است: اقدام «غیرفعال‌کردن عضویت» در دامنه ثبت دلیل را الزامی می‌داند (تا معلوم
// باشد چه کسی و چرا از فهرست امضاکنندگان بیرون رفته است)؛ پنل هم کادر یادداشت ندارد، پس
// همان تغییر وضعیتِ پنل، دلیل روشن خودش را می‌فرستد.
const committeeMemberNoteFa = "تغییر وضعیت عضویت از پنل تعریف اعضای کمیتهٔ ایثارگری"

func registerVeteranCommitteeMemberEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("grievances", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("grievances", "write", fn) }

	// فهرست اعضا (پیش‌فرض: همه، با ?status=active فقط فعال‌ها) + نقش‌های مصوب.
	mux.HandleFunc("GET /api/v1/grievances/veteran-committee-members", read(func(w http.ResponseWriter, r *http.Request) {
		onlyActive := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("status")), "active")
		items, err := committeeMemberViews(r.Context(), svc, onlyActive)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		active := 0
		for _, item := range items {
			if item["active"] == true {
				active++
			}
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"items":       items,
			"roles":       rules.VeteranCommitteeRoles(),
			"count":       len(items),
			"activeCount": active,
			"noteFa": "اعضای فعال پای صورت‌جلسهٔ مصوبه امضا می‌کنند و در بلوک امضای پایین فرم و " +
				"برگهٔ چاپی می‌آیند. عضو غیرفعال در فهرست می‌ماند ولی امضایش درج نمی‌شود.",
		})
	}))

	// ثبت عضو تازه یا ویرایش عضو موجود (با id).
	mux.HandleFunc("POST /api/v1/grievances/veteran-committee-members", write(func(w http.ResponseWriter, r *http.Request) {
		var in committeeMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ورودی عضو کمیته خوانده نشد.", nil)
			return
		}
		out, msg, err := committeeMemberSave(r.Context(), svc, in, actorOf(r))
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

	// فعال/غیرفعال‌کردن عضویت.
	mux.HandleFunc("POST /api/v1/grievances/veteran-committee-members/{id}/toggle", write(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		rec, found, err := svc.Get(r.Context(), contracts.RecordFilter{
			Module: "grievances", Kind: domain.KindVeteranCommitteeMember, Scope: "all", Size: 1,
		}, id)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		if !found {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "عضو کمیته با این شناسه یافت نشد.", nil)
			return
		}
		action := "deactivate"
		if rec.Status == "inactive" {
			action = "activate"
		}
		updated, err := svc.Act(r.Context(), rec.ID,
			contracts.RecordAction{Action: action, NoteFa: committeeMemberNoteFa}, actorOf(r))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		items, _ := committeeMemberViews(r.Context(), svc, false)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"record": updated, "items": items})
	}))

	// حذف عضو کمیته.
	mux.HandleFunc("DELETE /api/v1/grievances/veteran-committee-members/{id}", write(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		if err := svc.Delete(r.Context(), id, actorOf(r)); err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		items, _ := committeeMemberViews(r.Context(), svc, false)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	}))
}

// committeeMemberRequest ورودی پنل تعریف عضو کمیته.
type committeeMemberRequest struct {
	ID             string  `json:"id"`
	FullName       string  `json:"fullName"`
	Role           string  `json:"role"`
	PositionTitle  string  `json:"positionTitle"`
	EmployeeID     string  `json:"employeeId"`
	OrderNo        string  `json:"orderNo"`
	OrderDate      string  `json:"orderDate"`
	SignatureTitle string  `json:"signatureTitle"`
	SortOrder      float64 `json:"sortOrder"`
	Active         *bool   `json:"active"`
}

// committeeMemberSave عضو را ثبت یا ویرایش می‌کند و فهرست تازه را برمی‌گرداند.
func committeeMemberSave(ctx context.Context, svc *recordkit.Service, in committeeMemberRequest, actor string) (map[string]any, string, error) {
	fullName := strings.TrimSpace(in.FullName)
	if fullName == "" {
		return nil, "نام و نام خانوادگی عضو کمیته را بنویسید.", nil
	}
	role := rules.VeteranCommitteeRoleOf(in.Role)
	fields := map[string]string{
		"fullName":       fullName,
		"role":           role,
		"positionTitle":  strings.TrimSpace(in.PositionTitle),
		"employeeId":     strings.TrimSpace(in.EmployeeID),
		"orderNo":        strings.TrimSpace(in.OrderNo),
		"orderDate":      calendar.NormalizeDigits(strings.TrimSpace(in.OrderDate)),
		"signatureTitle": strings.TrimSpace(in.SignatureTitle),
		"sortOrder":      intField(int(in.SortOrder)),
	}
	title := "عضو کمیتهٔ ایثارگری — " + rules.VeteranCommitteeRoleTitleFa(role) + " (" + fullName + ")"
	id := strings.TrimSpace(in.ID)
	var rec contracts.Record
	var err error
	if id == "" {
		rec, err = svc.Create(ctx, contracts.RecordInput{
			Kind: domain.KindVeteranCommitteeMember, Title: title, Fields: fields,
			NoteFa: "ثبت از پنل اعضای کمیتهٔ ایثارگری",
		}, actor)
	} else {
		rec, err = svc.Update(ctx, id, contracts.RecordInput{
			Kind: domain.KindVeteranCommitteeMember, Title: title, Fields: fields,
			NoteFa: "ویرایش از پنل اعضای کمیتهٔ ایثارگری",
		}, actor)
	}
	if err != nil {
		return nil, "", err
	}
	// وضعیت عضویت با فیلد active کنترل می‌شود تا پنل بتواند عضو تازه را غیرفعال ثبت کند.
	if in.Active != nil && !*in.Active && rec.Status == "active" {
		next, actErr := svc.Act(ctx, rec.ID,
			contracts.RecordAction{Action: "deactivate", NoteFa: committeeMemberNoteFa}, actor)
		if actErr != nil {
			return nil, "", actErr
		}
		rec = next
	}
	if in.Active != nil && *in.Active && rec.Status == "inactive" {
		next, actErr := svc.Act(ctx, rec.ID,
			contracts.RecordAction{Action: "activate", NoteFa: committeeMemberNoteFa}, actor)
		if actErr != nil {
			return nil, "", actErr
		}
		rec = next
	}
	items, _ := committeeMemberViews(ctx, svc, false)
	return map[string]any{"record": rec, "items": items}, "", nil
}

// committeeMemberViews فهرست اعضای کمیته به ترتیب امضا؛ با onlyActive فقط اعضای فعال.
func committeeMemberViews(ctx context.Context, svc *recordkit.Service, onlyActive bool) ([]map[string]any, error) {
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "grievances", Kind: domain.KindVeteranCommitteeMember, Scope: "all", Size: 200,
	})
	if err != nil {
		return nil, err
	}
	records := make([]contracts.Record, 0, len(page.Items))
	for _, rec := range page.Items {
		if onlyActive && rec.Status != "active" {
			continue
		}
		records = append(records, rec)
	}
	sort.SliceStable(records, func(i, j int) bool {
		a, b := records[i].Fields, records[j].Fields
		oa, ob := committeeMemberSortValue(a), committeeMemberSortValue(b)
		if oa != ob {
			return oa < ob
		}
		ra := rules.VeteranCommitteeRoleOrder(a["role"])
		rb := rules.VeteranCommitteeRoleOrder(b["role"])
		if ra != rb {
			return ra < rb
		}
		return strings.TrimSpace(a["fullName"]) < strings.TrimSpace(b["fullName"])
	})
	items := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		items = append(items, committeeMemberView(rec))
	}
	return items, nil
}

// committeeMemberView یک عضو را برای رابط و سند آماده می‌کند.
func committeeMemberView(rec contracts.Record) map[string]any {
	f := rec.Fields
	role := rules.VeteranCommitteeRoleOf(f["role"])
	name := strings.TrimSpace(f["fullName"])
	return map[string]any{
		"id":             rec.ID,
		"fullName":       name,
		"role":           role,
		"roleFa":         rules.VeteranCommitteeRoleTitleFa(role),
		"positionTitle":  strings.TrimSpace(f["positionTitle"]),
		"employeeId":     strings.TrimSpace(f["employeeId"]),
		"orderNo":        strings.TrimSpace(f["orderNo"]),
		"orderDate":      strings.TrimSpace(f["orderDate"]),
		"signatureTitle": strings.TrimSpace(f["signatureTitle"]),
		"signatureFa":    rules.VeteranCommitteeSignatureFa(role, name, f["signatureTitle"]),
		"sortOrder":      committeeMemberSortValue(f),
		"status":         rec.Status,
		"statusFa":       rec.StatusFa,
		"active":         rec.Status == "active",
		"noteFa":         rec.NoteFa,
	}
}

// committeeMemberSortValue ترتیب امضا؛ خالی یعنی انتهای فهرست.
func committeeMemberSortValue(f map[string]string) int {
	value := numberValue(f["sortOrder"])
	if value <= 0 {
		return 900 + rules.VeteranCommitteeRoleOrder(f["role"])
	}
	return int(value)
}

// committeeActiveMemberSignatures عنوان امضای اعضای فعال به ترتیب امضا.
func committeeActiveMemberSignatures(ctx context.Context, svc *recordkit.Service) []string {
	items, err := committeeMemberViews(ctx, svc, true)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item["signatureFa"].(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, text)
		}
	}
	return out
}

// committeeActiveMemberNames نام اعضای فعال (برای پیش‌پر کردن متن «اعضای کمیته»).
func committeeActiveMemberNames(ctx context.Context, svc *recordkit.Service) []string {
	items, err := committeeMemberViews(ctx, svc, true)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if name, ok := item["fullName"].(string); ok && strings.TrimSpace(name) != "" {
			out = append(out, name)
		}
	}
	return out
}

// committeeSignedSignatures عنوان امضای اعضایی که در فرم امضا شده‌اند (بر پایهٔ شناسه).
func committeeSignedSignatures(ctx context.Context, svc *recordkit.Service, ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	wanted := map[string]bool{}
	for _, id := range ids {
		if value := strings.TrimSpace(id); value != "" {
			wanted[value] = true
		}
	}
	items, err := committeeMemberViews(ctx, svc, true)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(wanted))
	for _, item := range items {
		id, _ := item["id"].(string)
		if !wanted[id] {
			continue
		}
		if text, ok := item["signatureFa"].(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, text)
		}
	}
	return out
}
