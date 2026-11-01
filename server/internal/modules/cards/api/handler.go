// Package api لایهٔ HTTP ماژول کارت‌های رفاهی و شارژ.
//
// مسیرهای استاندارد رکوردها (schema/records/action) از موتور recordkit می‌آید؛ این
// بسته فقط سه کار اختصاصی این ماژول را انجام می‌دهد: اعلام گزینه‌ها به رابط،
// ساخت ردیف‌های شارژ از روی تخصیص‌ها و نرخ‌ها، و خروجی گرفتن برای واحد مالی.
package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/cards/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Handler مسیرهای اختصاصی ماژول کارت‌ها.
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

// Options گزینه‌های پایه: کارت‌ها، بخش‌ها با نرخ جاری و مناسبت‌های فعال.
//
// رابط وب از همین می‌سازد؛ پس نرخ جدید یا مناسبت تازه بدون تغییر کد در فرم‌ها
// دیده می‌شود.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rates := h.ratesBySection(ctx)

	out := contracts.CardOptions{
		Cards:       []contracts.CardOption{},
		Sections:    []contracts.CardOption{},
		Occasions:   []contracts.CardOption{},
		RatesByCard: map[string][]contracts.CardOption{},
	}
	for _, c := range domain.Cards() {
		out.Cards = append(out.Cards, contracts.CardOption{Key: c, TitleFa: domain.CardTitleFa(c)})
	}
	for _, s := range domain.Sections() {
		out.Sections = append(out.Sections, contracts.CardOption{
			Key: s.Key, TitleFa: s.TitleFa, Card: s.Card, Amount: rates[s.Key],
		})
		out.RatesByCard[s.Card] = append(out.RatesByCard[s.Card], contracts.CardOption{
			Key: s.Key, TitleFa: s.TitleFa, Card: s.Card, Amount: rates[s.Key],
		})
	}
	for _, o := range h.activeRecords(ctx, "occasions") {
		out.Occasions = append(out.Occasions, contracts.CardOption{
			Key: o.ID, TitleFa: o.Title, Card: o.Fields["card"],
			Amount: parseInt(o.Fields["amount"]), DateFa: o.Fields["date"],
		})
	}
	if today, err := calendar.FromGregorian(h.now()); err == nil {
		out.TodayFa = today.String()
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// PreviewCharge فقط ردیف‌های شارژ را می‌سازد و چیزی ذخیره نمی‌کند (پیش‌نمایش فرم).
func (h *Handler) PreviewCharge(w http.ResponseWriter, r *http.Request) {
	var req contracts.CardChargeRequest
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

// CreateChargeFile ردیف‌ها را می‌سازد، پروندهٔ شارژ را ثبت می‌کند و همان را
// برمی‌گرداند تا رابط، جمع مبالغ و خروجی مالی را نشان دهد.
func (h *Handler) CreateChargeFile(w http.ResponseWriter, r *http.Request) {
	var req contracts.CardChargeRequest
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
		Kind:  "chargeFiles",
		Title: "فایل شارژ " + result.Period + " — " + result.CardFa,
		Fields: map[string]string{
			"card":         req.Card,
			"period":       result.Period,
			"scopeFa":      result.ScopeFa,
			"occasionFa":   occasionFaOf(r.Context(), h, req.OccasionID),
			"rows":         fmt.Sprint(result.Count),
			"total":        fmt.Sprint(result.Total),
			"deliveryFa":   "واحد مالی",
			"scopeKey":     scopeKey(req),
			"sectionsCsv":  strings.Join(req.Sections, ","),
			"employeesCsv": strings.Join(req.EmployeeIDs, ","),
			"occasionId":   req.OccasionID,
			"maxHeads":     fmt.Sprint(req.MaxHeads),
		},
		NoteFa: "ساخته‌شده از پنل فایل‌های شارژ برای ارسال به مالی",
	}, user.Username)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	result.FileID = rec.ID
	result.FileNoFa = rec.ID
	httpx.WriteJSON(w, http.StatusCreated, result)
}

// ExportChargeFile خروجی CSV آمادهٔ ارسال به واحد مالی.
//
// فایل با همان معیارهای ذخیره‌شده در پرونده بازساخته می‌شود، پس خروجی همیشه با
// پیش‌نمایشی که کاربر دیده یکی است.
func (h *Handler) ExportChargeFile(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	rec, ok, err := h.svc.Get(r.Context(), contracts.RecordFilter{Module: "cards", Scope: "all"}, r.PathValue("id"))
	if err != nil || !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	req := contracts.CardChargeRequest{
		Card:        rec.Fields["card"],
		Period:      rec.Fields["period"],
		Scope:       rec.Fields["scopeKey"],
		OccasionID:  rec.Fields["occasionId"],
		Sections:    splitCSV(rec.Fields["sectionsCsv"]),
		EmployeeIDs: splitCSV(rec.Fields["employeesCsv"]),
		MaxHeads:    int(parseInt(rec.Fields["maxHeads"])),
	}
	result, buildErr := h.build(r.Context(), req)
	if buildErr != nil {
		writeRuleError(w, buildErr)
		return
	}
	// شناسهٔ فایل در سرصفحهٔ خروجی میآید تا مالی بتواند خروجی را به همان پرونده
	// بازبگرداند.
	result.FileID = rec.ID
	result.FileNoFa = rec.ID

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+rec.ID+`.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	// BOM لازم است وگرنه Excel فارسی را نامفهوم نشان می‌دهد و CRLF می‌گذاریم تا
	// Excel ویندوز همهٔ ردیف‌ها را جدا ببیند.
	fmt.Fprint(w, "\ufeff")
	fmt.Fprint(w, csvEscapeLineEndings(csvHeader()))
	for _, row := range result.Rows {
		fmt.Fprint(w, csvEscapeLineEndings(csvRow(row)))
	}
	fmt.Fprint(w, csvEscapeLineEndings(csvFooter(result)))
	h.deps.Audit.Append(user.Username, "cards", "charge-file-export", rec.ID,
		"خروجی فایل شارژ برای مالی — "+fmt.Sprint(result.Count)+" ردیف")
}

// ExportChargeFileXLSX خروجی اکسل فایل شارژ (برای واحد مالی و کارگزینی).
//
// چرا در کنار CSV: فایل CSV فارسی در Excel وابسته به تنظیم ناحیه‌ای رایانه است؛ فایل
// xlsx خودش قلم، کدگذاری و راست‌به‌چپ را حمل می‌کند. معیارهای شارژ همان است که در پرونده
// ذخیره شده، پس خروجی با پیش‌نمایش کاربر یکی است.
func (h *Handler) ExportChargeFileXLSX(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	rec, ok, err := h.svc.Get(r.Context(), contracts.RecordFilter{Module: "cards", Scope: "all"}, r.PathValue("id"))
	if err != nil || !ok {
		httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
		return
	}
	req := contracts.CardChargeRequest{
		Card:        rec.Fields["card"],
		Period:      rec.Fields["period"],
		Scope:       rec.Fields["scopeKey"],
		OccasionID:  rec.Fields["occasionId"],
		Sections:    splitCSV(rec.Fields["sectionsCsv"]),
		EmployeeIDs: splitCSV(rec.Fields["employeesCsv"]),
		MaxHeads:    int(parseInt(rec.Fields["maxHeads"])),
	}
	result, buildErr := h.build(r.Context(), req)
	if buildErr != nil {
		writeRuleError(w, buildErr)
		return
	}
	// شمارهٔ فایل در سرصفحهٔ اکسل هم میآید (همان قاعدهٔ خروجی CSV).
	result.FileID = rec.ID
	result.FileNoFa = rec.ID
	content, err := chargeWorkbook(result)
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+rec.ID+`.xlsx"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
	h.deps.Audit.Append(user.Username, "cards", "charge-file-export-xlsx", rec.ID,
		"خروجی اکسل فایل شارژ — "+fmt.Sprint(result.Count)+" ردیف")
}

/* ------------------------------ درون ماژول ------------------------------ */

func (h *Handler) build(ctx context.Context, req contracts.CardChargeRequest) (contracts.CardChargeResult, error) {
	card := strings.TrimSpace(req.Card)
	if !domain.ValidCard(card) {
		return contracts.CardChargeResult{}, &recordkit.RuleError{Message: "کارت انتخاب‌شده نامعتبر است."}
	}
	period := strings.TrimSpace(req.Period)
	if period == "" {
		return contracts.CardChargeResult{}, &recordkit.RuleError{Message: "دورهٔ شارژ (YYYY/MM) الزامی است."}
	}

	beneficiaries, err := h.beneficiaries(ctx, req)
	if err != nil {
		return contracts.CardChargeResult{}, err
	}
	if len(beneficiaries) == 0 {
		return contracts.CardChargeResult{}, &recordkit.RuleError{Message: "هیچ فرد مشمول فعالی برای این شارژ یافت نشد؛ ابتدا در پنل تخصیص، افراد را تعریف کنید."}
	}

	var occasion *domain.OccasionCharge
	if id := strings.TrimSpace(req.OccasionID); id != "" {
		rec, ok, err := h.svc.Get(ctx, contracts.RecordFilter{Module: "cards", Scope: "all"}, id)
		if err != nil || !ok {
			return contracts.CardChargeResult{}, &recordkit.RuleError{Message: "مناسبت انتخاب‌شده یافت نشد."}
		}
		occasion = &domain.OccasionCharge{
			ID: rec.ID, TitleFa: rec.Title, DateFa: rec.Fields["date"],
			Amount: parseInt(rec.Fields["amount"]), Card: rec.Fields["card"],
		}
	}

	serviceKey := contracts.ServiceForCard(card)
	lines, warnings := domain.Build(domain.ChargeInput{
		Card:          card,
		Sections:      req.Sections,
		RateBySection: h.ratesBySection(ctx),
		Beneficiaries: beneficiaries,
		Occasion:      occasion,
		Period:        period,
		MaxHeads:      req.MaxHeads,
		ServiceKey:    serviceKey,
		ServiceKeyFa:  contracts.DependentServiceTitleFa(serviceKey),
	})

	total, bySection, count := domain.SumLines(lines)
	rows := make([]contracts.CardChargeRow, 0, len(lines))
	for _, l := range lines {
		rows = append(rows, contracts.CardChargeRow{
			EmployeeID: l.EmployeeID, EmployeeName: l.EmployeeName,
			Beneficiary: l.Beneficiary, Relation: l.Relation,
			Section: l.Section, SectionFa: l.SectionFa, OccasionFa: l.OccasionFa,
			Amount: l.Amount, BasisFa: l.BasisFa,
		})
	}
	totals := make([]contracts.CardChargeTotal, 0, len(bySection))
	for _, s := range domain.SectionsOf(card) {
		if n, ok := count[s.Key]; ok {
			totals = append(totals, contracts.CardChargeTotal{Section: s.Key, SectionFa: s.TitleFa, Count: n, Total: bySection[s.Key]})
		}
	}
	if n, ok := count["occasion"]; ok {
		totals = append(totals, contracts.CardChargeTotal{Section: "occasion", SectionFa: "شارژ مناسبتی", Count: n, Total: bySection["occasion"]})
	}

	return contracts.CardChargeResult{
		Card: card, CardFa: domain.CardTitleFa(card), Period: period,
		ScopeFa: scopeFa(req), Rows: rows, Count: len(rows), Total: total,
		BySection: totals, Warnings: warnings,
	}, nil
}

// beneficiaries افراد مشمول فعال را می‌خواند و در حالت انفرادی به کارکنان خواسته‌شده
// محدود می‌کند. افراد تحت تکفل هر کارمند از همان رکورد می‌آیند، پس تخصیص انفرادی هم
// خانوادهٔ همان کارمند را در بر می‌گیرد.
func (h *Handler) beneficiaries(ctx context.Context, req contracts.CardChargeRequest) ([]domain.Beneficiary, error) {
	records := h.activeRecords(ctx, "cardBeneficiaries")
	want := map[string]bool{}
	for _, id := range req.EmployeeIDs {
		if id = strings.TrimSpace(id); id != "" {
			want[id] = true
		}
	}
	individual := strings.TrimSpace(req.Scope) == "individual"
	if individual && len(want) == 0 {
		return nil, &recordkit.RuleError{Message: "در شارژ انفرادی باید دست‌کم یک کارمند انتخاب شود."}
	}

	// وضعیت تعلق افراد تحت تکفل از پورت ماژول امور کارکنان خوانده می‌شود و برای هر
	// کارمند یک بار (با حافظهٔ کوچک) تا با N فرد، N بار پرس‌وجو نشود.
	dependentCache := map[string][]contracts.Dependent{}
	dependentsOf := func(employeeID string) []contracts.Dependent {
		if list, ok := dependentCache[employeeID]; ok {
			return list
		}
		list := []contracts.Dependent{}
		if reader, ok := h.deps.Ports.Dependents(); ok {
			if items, err := reader.ApprovedDependents(ctx, employeeID); err == nil {
				list = items
			}
		}
		dependentCache[employeeID] = list
		return list
	}

	out := make([]domain.Beneficiary, 0, len(records))
	for _, rec := range records {
		if individual && !want[rec.EmployeeID] {
			continue
		}
		relation := strings.TrimSpace(rec.Fields["relation"])
		self := strings.Contains(relation, "خود")
		item := domain.Beneficiary{
			RecordID:     rec.ID,
			EmployeeID:   rec.EmployeeID,
			EmployeeName: rec.EmployeeName,
			Name:         rec.Title,
			Relation:     relation,
			Excellent:    isFlag(rec.Fields["excellent"]),
			Self:         self,
			Allocated: map[string]bool{
				"petrokartCultural":   isFlag(rec.Fields["petrokartCultural"]),
				"petrokartRestaurant": isFlag(rec.Fields["petrokartRestaurant"]),
				"educationBonus":      isFlag(rec.Fields["educationBonus"]),
				"naftmart":            isFlag(rec.Fields["naftmart"]),
			},
		}
		// تعلق فقط برای فرد تحت تکفل معنا دارد؛ خودِ کارمند مشمول است.
		if !self {
			if dependent, ok := matchDependent(dependentsOf(rec.EmployeeID), rec); ok {
				item.EligibleKnown = true
				item.Eligible = map[string]bool{}
				for _, key := range dependent.Services {
					item.Eligible[key] = true
				}
			}
		}
		out = append(out, item)
	}
	// ترتیب پایدار: خود کارمند نخست، سپس افراد تحت تکفل به ترتیب نام. سقف سرها به همین
	// ترتیب بسته می‌شود، پس «چه کسی از سقف افتاد» تکرارپذیر و قابل توضیح است.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].EmployeeID != out[j].EmployeeID {
			return out[i].EmployeeID < out[j].EmployeeID
		}
		if out[i].Self != out[j].Self {
			return out[i].Self
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// matchDependent فرد تحت تکفل متناظر یک رکورد «فرد مشمول» را پیدا می‌کند.
//
// ترتیب تطبیق: کد ملی (دقیق‌ترین) و سپس نام و نام خانوادگی. کد ملی اگر در رکورد مشمول
// ثبت شده باشد مقدم است؛ وگرنه نام‌ها مقایسه می‌شوند تا دادهٔ قدیمی که کد ملی ندارد هم
// از فرم تکفل بهره ببرد.
func matchDependent(dependents []contracts.Dependent, rec contracts.Record) (contracts.Dependent, bool) {
	nationalID := calendar.NormalizeDigits(strings.TrimSpace(rec.Fields["nationalId"]))
	if nationalID != "" {
		for _, dependent := range dependents {
			if calendar.NormalizeDigits(strings.TrimSpace(dependent.NationalID)) == nationalID {
				return dependent, true
			}
		}
	}
	name := normalizeName(rec.Title)
	for _, dependent := range dependents {
		if normalizeName(dependent.FullName) == name {
			return dependent, true
		}
	}
	return contracts.Dependent{}, false
}

// normalizeName نام را برای مقایسه یکدست می‌کند (فاصله‌های اضافی و نیم‌فاصله).
func normalizeName(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\u200c", " ")
	return strings.Join(strings.Fields(value), " ")
}

// ratesBySection نرخ جاری هر بخش از رکوردهای فعال؛ اگر نرخی ثبت نشده باشد، رابط و
// موتور به نرخ پیش‌فرض دامنه برمی‌گردند.
func (h *Handler) ratesBySection(ctx context.Context) map[string]int64 {
	out := domain.DefaultRates()
	for _, rec := range h.activeRecords(ctx, "cardRates") {
		key := strings.TrimSpace(rec.Fields["section"])
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
		Module: "cards", Kind: kind, Status: "active", Page: 1, Size: 1000,
	})
	if err != nil {
		return nil
	}
	return page.Items
}

func occasionFaOf(ctx context.Context, h *Handler, id string) string {
	if strings.TrimSpace(id) == "" {
		return "—"
	}
	if rec, ok, err := h.svc.Get(ctx, contracts.RecordFilter{Module: "cards", Scope: "all"}, id); err == nil && ok {
		return rec.Title
	}
	return "—"
}

func scopeKey(req contracts.CardChargeRequest) string {
	if strings.TrimSpace(req.Scope) == "individual" {
		return "individual"
	}
	return "group"
}

func scopeFa(req contracts.CardChargeRequest) string {
	if scopeKey(req) == "individual" {
		return "انفرادی — " + calendar.ToPersianDigits(fmt.Sprint(len(req.EmployeeIDs))) + " کارمند انتخاب‌شده"
	}
	return "گروهی — همهٔ مشمولان فعال"
}

func isFlag(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "بله", "دارد", "فعال":
		return true
	}
	return false
}

// parseInt مبلغ را از فیلد رکورد می‌خواند. ارقام فارسی و جداکننده‌ها (٬ و ,) را هم
// می‌فهمد چون کاربر مبلغ را فارسی وارد می‌کند.
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
