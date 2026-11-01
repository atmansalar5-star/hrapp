// Package api لایهٔ HTTP ماژول پاداش تحصیلی فرزندان ممتاز.
//
// مسیرهای استاندارد رکوردها (schema/records/action) از موتور recordkit می‌آید؛ این
// بسته فقط کارهای اختصاصی این ماژول را انجام می‌دهد: اعلام گزینه‌ها به رابط، محاسبهٔ
// ردیف‌های پاداش از روی مبالغ مصوب، و خروجی گرفتن برای واحد مالی.
package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/education_bonus/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Handler مسیرهای اختصاصی ماژول پاداش تحصیلی.
type Handler struct {
	svc  *recordkit.Service
	deps modulekit.Deps
}

// New هندلر می‌سازد.
func New(svc *recordkit.Service, deps modulekit.Deps) *Handler {
	return &Handler{svc: svc, deps: deps}
}

// now ساعت سرور (یا ساعت تزریقی آزمون) را برمی‌گرداند.
func (h *Handler) now() time.Time {
	if h.deps.Clock != nil {
		return h.deps.Clock()
	}
	return time.Now()
}

// Options گزینه‌های پایه: مقاطع با مبلغ مصوب جاری و تاریخ امروز سرور.
//
// رابط وب از همین می‌سازد؛ پس مبلغ مصوب جدید بدون تغییر کد در فرم‌ها دیده می‌شود.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rates := h.ratesByLevel(ctx)

	out := contracts.BonusOptions{Levels: []contracts.BonusOption{}}
	for _, l := range domain.Levels() {
		amount := rates[l.Key]
		out.Levels = append(out.Levels, contracts.BonusOption{Key: l.Key, TitleFa: l.TitleFa, Amount: amount})
	}
	if today, err := calendar.FromGregorian(h.now()); err == nil {
		out.TodayFa = today.String()
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// PreviewBonus فقط ردیف‌های پاداش را می‌سازد و چیزی ذخیره نمی‌کند (پیش‌نمایش فرم).
func (h *Handler) PreviewBonus(w http.ResponseWriter, r *http.Request) {
	var req contracts.BonusRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	req.TrialRun = true
	result, err := h.build(r.Context(), req)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// CreateBonusFile ردیف‌ها را می‌سازد، فایل محاسباتی را ثبت می‌کند و همان را
// برمی‌گرداند تا رابط، جمع مبالغ و خروجی مالی را نشان دهد.
func (h *Handler) CreateBonusFile(w http.ResponseWriter, r *http.Request) {
	var req contracts.BonusRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	req.TrialRun = false
	result, err := h.build(r.Context(), req)
	if err != nil {
		writeRuleError(w, err)
		return
	}

	user, _ := auth.UserFrom(r.Context())
	rec, err := h.svc.Create(r.Context(), contracts.RecordInput{
		Kind:  "bonusFiles",
		Title: "فایل پاداش تحصیلی " + result.Period,
		Fields: map[string]string{
			"period":       result.Period,
			"scopeFa":      result.ScopeFa,
			"levelsCsv":    strings.Join(req.Levels, ","),
			"employeesCsv": strings.Join(req.EmployeeIDs, ","),
			"rows":         fmt.Sprint(result.Count),
			"total":        fmt.Sprint(result.Total),
			"deliveryFa":   "واحد مالی",
		},
		NoteFa: "ساخته‌شده از پنل فایل محاسباتی پاداش تحصیلی برای ارسال به مالی",
	}, user.Username)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	result.FileID = rec.ID
	result.FileNoFa = rec.ID
	httpx.WriteJSON(w, http.StatusCreated, result)
}

// ExportBonusFile خروجی CSV آمادهٔ ارسال به واحد مالی.
//
// فایل با همان معیارهای ذخیره‌شده در پرونده بازساخته می‌شود، پس خروجی همیشه با
// پیش‌نمایشی که کاربر دیده یکی است.
func (h *Handler) ExportBonusFile(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	rec, ok, err := h.svc.Get(r.Context(), contracts.RecordFilter{Module: "education_bonus", Scope: "all"}, r.PathValue("id"))
	if err != nil || !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	req := contracts.BonusRequest{
		Period:      rec.Fields["period"],
		Scope:       strings.TrimSpace(rec.Fields["scopeFa"]),
		Levels:      splitCSV(rec.Fields["levelsCsv"]),
		EmployeeIDs: splitCSV(rec.Fields["employeesCsv"]),
	}
	// دامنهٔ گروهی در رکورد با متن فارسی آمده؛ مبنای بازسازی، فهرست کارکنان است.
	if len(req.EmployeeIDs) > 0 {
		req.Scope = "individual"
	} else {
		req.Scope = "group"
	}
	result, buildErr := h.build(r.Context(), req)
	if buildErr != nil {
		writeRuleError(w, buildErr)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+rec.ID+`.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	// BOM لازم است وگرنه Excel فارسی را نامفهوم نشان می‌دهد و CRLF می‌گذاریم تا Excel
	// ویندوز همهٔ ردیف‌ها را جدا ببیند (هم‌خوان با فایل شارژ کارت‌های رفاهی).
	fmt.Fprint(w, "\ufeff")
	fmt.Fprint(w, csvWithCRLF(csvHeader()))
	for _, row := range result.Rows {
		fmt.Fprint(w, csvWithCRLF(csvRow(row)))
	}
	fmt.Fprint(w, csvWithCRLF(csvFooter(result)))
	h.deps.Audit.Append(user.Username, "education_bonus", "bonus-file-export", rec.ID,
		"خروجی فایل پاداش تحصیلی برای مالی — "+fmt.Sprint(result.Count)+" ردیف")
}

/* ------------------------------ درون ماژول ------------------------------ */

func (h *Handler) build(ctx context.Context, req contracts.BonusRequest) (contracts.BonusResult, error) {
	period := strings.TrimSpace(req.Period)
	if period == "" {
		return contracts.BonusResult{}, &recordkit.RuleError{Message: "دورهٔ پرداخت (سال تحصیلی) الزامی است."}
	}

	students, err := h.students(ctx, req)
	if err != nil {
		return contracts.BonusResult{}, err
	}
	if len(students) == 0 {
		return contracts.BonusResult{}, &recordkit.RuleError{Message: "هیچ دانش‌آموز ممتاز مشمولی یافت نشد؛ ابتدا در پنل دانش‌آموزان ممتاز، مشمولان را تعریف کنید."}
	}

	lines, warnings := domain.Build(domain.BonusInput{
		Levels:   req.Levels,
		Rates:    h.ratesByLevel(ctx),
		Students: students,
		Period:   period,
	})

	total, byLevel, count := domain.SumLines(lines)
	rows := make([]contracts.BonusRow, 0, len(lines))
	for _, l := range lines {
		rows = append(rows, contracts.BonusRow{
			EmployeeID: l.EmployeeID, EmployeeName: l.EmployeeName,
			Student: l.Student, Relation: l.Relation,
			Level: l.Level, LevelFa: l.LevelFa, Grade: l.Grade,
			Amount: l.Amount, BasisFa: l.BasisFa,
		})
	}
	totals := make([]contracts.BonusTotal, 0, len(byLevel))
	for _, l := range domain.Levels() {
		if n, ok := count[l.Key]; ok {
			totals = append(totals, contracts.BonusTotal{Level: l.Key, LevelFa: l.TitleFa, Count: n, Total: byLevel[l.Key]})
		}
	}

	return contracts.BonusResult{
		Period: period, ScopeFa: scopeFa(req), Rows: rows, Count: len(rows),
		Total: total, ByLevel: totals, Warnings: warnings,
	}, nil
}

// students دانش‌آموزان ممتاز فعال را می‌خواند و در حالت انفرادی به کارکنان
// خواسته‌شده محدود می‌کند.
func (h *Handler) students(ctx context.Context, req contracts.BonusRequest) ([]domain.Student, error) {
	records := h.activeRecords(ctx, "outstandingStudents")
	want := map[string]bool{}
	for _, id := range req.EmployeeIDs {
		if id = strings.TrimSpace(id); id != "" {
			want[id] = true
		}
	}
	individual := strings.TrimSpace(req.Scope) == "individual"
	if individual && len(want) == 0 {
		return nil, &recordkit.RuleError{Message: "در محاسبهٔ انفرادی باید دست‌کم یک کارمند انتخاب شود."}
	}

	out := make([]domain.Student, 0, len(records))
	for _, rec := range records {
		if individual && !want[rec.EmployeeID] {
			continue
		}
		out = append(out, domain.Student{
			RecordID:     rec.ID,
			EmployeeID:   rec.EmployeeID,
			EmployeeName: rec.EmployeeName,
			Name:         rec.Title,
			Relation:     rec.Fields["relation"],
			Level:        strings.TrimSpace(rec.Fields["level"]),
			Grade:        rec.Fields["grade"],
			Excellent:    isFlag(rec.Fields["excellent"]),
			Eligible:     isFlag(rec.Fields["eligible"]),
		})
	}
	return out, nil
}

// ratesByLevel مبلغ مصوب جاری هر مقطع از رکوردهای فعال؛ اگر مبلغی ثبت نشده باشد،
// موتور به مبلغ پیش‌فرض دامنه برمی‌گردد.
func (h *Handler) ratesByLevel(ctx context.Context) map[string]int64 {
	out := domain.DefaultRates()
	for _, rec := range h.activeRecords(ctx, "bonusRates") {
		key := strings.TrimSpace(rec.Fields["level"])
		amount := parseInt(rec.Fields["amount"])
		if key == "" || amount <= 0 {
			continue
		}
		out[key] = amount
	}
	return out
}

func (h *Handler) activeRecords(ctx context.Context, kind string) []contracts.Record {
	page, err := h.svc.List(ctx, contracts.RecordFilter{
		Module: "education_bonus", Kind: kind, Status: "active", Page: 1, Size: 1000,
	})
	if err != nil {
		return nil
	}
	return page.Items
}

func scopeKey(req contracts.BonusRequest) string {
	if strings.TrimSpace(req.Scope) == "individual" {
		return "individual"
	}
	return "group"
}

func scopeFa(req contracts.BonusRequest) string {
	if scopeKey(req) == "individual" {
		return "انفرادی — " + fmt.Sprint(len(req.EmployeeIDs)) + " کارمند انتخاب‌شده"
	}
	return "گروهی — همهٔ مشمولان ممتاز"
}

func isFlag(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "بله", "دارد", "فعال":
		return true
	}
	return false
}

// parseInt مبلغ را از فیلد رکورد می‌خواند و ارقام فارسی و جداکننده‌ها را هم می‌فهمد.
func parseInt(v string) int64 {
	v = calendar.NormalizeDigits(strings.TrimSpace(v))
	var n int64
	for _, c := range v {
		if c < '0' || c > '9' {
			continue
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

func splitCSV(v string) []string {
	out := []string{}
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func writeRuleError(w http.ResponseWriter, err error) {
	if re, ok := err.(*recordkit.RuleError); ok {
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeBadRequest, re.Message, nil)
		return
	}
	if ve, ok := err.(*recordkit.ValidationError); ok {
		httpx.WriteError(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, ve.Details)
		return
	}
	httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
}

// csvWithCRLF پایان‌خط‌ها را به CRLF برمی‌گرداند تا Excel ویندوز هر ردیف را جدا نشان دهد.
func csvWithCRLF(body string) string {
	return strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n")
}
