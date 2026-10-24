// Package api لایهٔ HTTP اختصاصی ماژول امور وام و بازنشستگی.
//
// مسیرهای استاندارد رکوردها (schema/records/action) از موتور recordkit می‌آید؛ این
// بسته دو کار اختصاصی را انجام می‌دهد: اعلام گزینه‌های پایه (انواع وام، بانک‌های عامل
// و معیارهای امتیازبندی) و پیش‌نمایش امتیازبندی تخصیص وام. هیچ نرخ یا معیاری در این
// لایه هاردکد نیست و همه از دامنه می‌آید.
package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/loans_retirement/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Handler مسیرهای اختصاصی ماژول وام.
type Handler struct {
	svc  *recordkit.Service
	deps modulekit.Deps
}

// New هندلر می‌سازد.
func New(svc *recordkit.Service, deps modulekit.Deps) *Handler {
	return &Handler{svc: svc, deps: deps}
}

func (h *Handler) now() time.Time {
	if h.deps.Clock != nil {
		return h.deps.Clock()
	}
	return time.Now()
}

// Options گزینه‌های پایه: انواع وام، بانک‌های عامل و معیارهای امتیازبندی.
//
// رابط از همین می‌سازد؛ افزودن نوع وام یا بانک تازه در دامنه، خودکار در فرم دیده می‌شود.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	out := contracts.LoanOptions{
		LoanTypes:       []contracts.LoanOption{},
		Banks:           []contracts.LoanOption{},
		ManagementUnits: []contracts.LoanOption{},
		Units:           []contracts.CatalogItem{},
		Criteria:        []contracts.LoanCriterion{},
	}
	for _, t := range domain.LoanTypes() {
		out.LoanTypes = append(out.LoanTypes, contracts.LoanOption{
			Key: t.Key, TitleFa: t.TitleFa, IsSupplement: t.IsSupplement, BaseKey: t.BaseKey,
			FeeRatePercent: t.FeeRatePercent, MaxInstallments: t.MaxInstallments,
			MaxAmount: t.MaxAmount, PurposeFa: t.PurposeFa,
		})
	}
	for _, b := range h.bankOptions(r.Context()) {
		out.Banks = append(out.Banks, b)
	}
	// مدیریت‌های جدول سهمیهٔ تسهیلات: مبنای فیلد «مدیریت متقاضی» در فرم وام و
	// نمودارهای تخصیص سهمیه. اقلام کاتالوگ مرجع (پنل تنظیمات) روی همین فهرست می‌نشینند.
	for _, m := range h.managementOptions(r.Context()) {
		out.ManagementUnits = append(out.ManagementUnits, m)
	}
	// واحدهای سازمانی از کاتالوگ مرجع تنظیمات؛ هر واحد زیر یک مدیریت.
	for _, u := range h.catalogItems(r.Context(), contracts.CatalogUnits) {
		if !u.Active {
			continue
		}
		out.Units = append(out.Units, u)
	}
	for _, c := range domain.Criteria() {
		opts := make([]contracts.LoanCriterionOption, 0, len(c.Options))
		for _, o := range c.Options {
			opts = append(opts, contracts.LoanCriterionOption{Value: o.Value, TitleFa: o.TitleFa, Score: o.Score})
		}
		out.Criteria = append(out.Criteria, contracts.LoanCriterion{
			Key: c.Key, TitleFa: c.TitleFa, Weight: c.Weight, Options: opts,
		})
	}
	if today, err := calendar.FromGregorian(h.now()); err == nil {
		out.TodayFa = today.String()
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

/* --------------------------- کاتالوگ مرجع --------------------------- */

// catalogItems اقلام یک کاتالوگ مرجع را از پنل تنظیمات می‌خواند.
//
// اگر ماژول تنظیمات در این استقرار ثبت نشده باشد یا خواندن خطا بدهد، فهرست خالی
// برمی‌گردد و فرم به فهرست پایهٔ دامنه تکیه می‌کند؛ یعنی نبود آن، سامانه را نمی‌شکند.
func (h *Handler) catalogItems(ctx context.Context, kind string) []contracts.CatalogItem {
	if h.deps.Ports == nil {
		return nil
	}
	reader, ok := h.deps.Ports.Catalogs()
	if !ok {
		return nil
	}
	items, err := reader.Catalog(ctx, kind)
	if err != nil {
		return nil
	}
	return items
}

// bankOptions بانک‌های عامل: فهرست پایهٔ دامنه به‌علاوهٔ اقلام کاتالوگ مرجع.
// قلم هم‌نام، عنوان تازه می‌گیرد؛ قلم غیرفعال کنار گذاشته می‌شود.
func (h *Handler) bankOptions(ctx context.Context) []contracts.LoanOption {
	merged := map[string]contracts.LoanOption{}
	order := []string{}
	for _, b := range domain.Banks() {
		merged[b] = contracts.LoanOption{Key: b, TitleFa: domain.BankTitleFa(b)}
		order = append(order, b)
	}
	for _, it := range h.catalogItems(ctx, contracts.CatalogBanks) {
		if it.Key == "" {
			continue
		}
		if !it.Active {
			delete(merged, it.Key)
			continue
		}
		if _, seen := merged[it.Key]; !seen {
			order = append(order, it.Key)
		}
		merged[it.Key] = contracts.LoanOption{Key: it.Key, TitleFa: it.TitleFa}
	}
	out := make([]contracts.LoanOption, 0, len(order))
	for _, k := range order {
		if o, ok := merged[k]; ok {
			out = append(out, o)
		}
	}
	return out
}

// managementOptions مدیریت‌های جدول سهمیه: درصد سهمیه و مشمولین از سیاست دامنه می‌آید و
// اقلام کاتالوگ مرجع روی همان می‌نشینند. مدیریت تازهٔ پنل با سهمیهٔ صفر می‌آید تا در
// داشبورد سهمیه گم نشود.
func (h *Handler) managementOptions(ctx context.Context) []contracts.LoanOption {
	merged := map[string]contracts.LoanOption{}
	order := []string{}
	for _, u := range domain.QuotaUnits() {
		merged[u.Key] = contracts.LoanOption{
			Key: u.Key, TitleFa: u.TitleFa, SharePercent: u.SharePercent, Eligible: u.Eligible,
		}
		order = append(order, u.Key)
	}
	for _, it := range h.catalogItems(ctx, contracts.CatalogManagements) {
		if it.Key == "" {
			continue
		}
		if !it.Active {
			delete(merged, it.Key)
			continue
		}
		base, seen := merged[it.Key]
		if !seen {
			order = append(order, it.Key)
		}
		base.Key, base.TitleFa = it.Key, it.TitleFa
		merged[it.Key] = base
	}
	out := make([]contracts.LoanOption, 0, len(order))
	for _, k := range order {
		if o, ok := merged[k]; ok {
			out = append(out, o)
		}
	}
	return out
}

// bankTitle عنوان فارسی بانک عامل از فهرست پایه به‌علاوهٔ کاتالوگ مرجع؛ برای کلید
// ناشناخته خودِ ورودی برمی‌گردد تا نمایش دادهٔ قدیمی خراب نشود. اگر بانک تازه‌ای در
// پنل تنظیمات ساخته شود، خروجی و جدول وام هم همان عنوان فارسی را می‌بینند.
func (h *Handler) bankTitle(ctx context.Context, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	for _, b := range h.bankOptions(ctx) {
		if b.Key == key {
			return b.TitleFa
		}
	}
	return key
}

// managementTitle عنوان فارسی مدیریت متقاضی از جدول سهمیه به‌علاوهٔ کاتالوگ مرجع.
func (h *Handler) managementTitle(ctx context.Context, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	for _, m := range h.managementOptions(ctx) {
		if m.Key == key {
			return m.TitleFa
		}
	}
	return key
}

// unitTitle عنوان فارسی واحد سازمانی؛ برای کلید ناشناخته خودِ ورودی برمی‌گردد تا
// نمایش دادهٔ قدیمی خراب نشود.
func (h *Handler) unitTitle(ctx context.Context, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	for _, u := range h.catalogItems(ctx, contracts.CatalogUnits) {
		if u.Key == key {
			return u.TitleFa
		}
	}
	return key
}

// policy اقلام کاتالوگ مرجع را به‌شکل موردانتظار اعتبارسنجی دامنه درمی‌آورد تا بانک یا
// مدیریت تازهٔ پنل تنظیمات در فرم وام پذیرفته شود.
func (h *Handler) policy(ctx context.Context) domain.GrantPolicy {
	p := domain.GrantPolicy{ExtraBanks: map[string]string{}, ExtraManagements: map[string]string{}}
	for _, it := range h.catalogItems(ctx, contracts.CatalogBanks) {
		if it.Active && it.Key != "" {
			p.ExtraBanks[it.Key] = it.TitleFa
		}
	}
	for _, it := range h.catalogItems(ctx, contracts.CatalogManagements) {
		if it.Active && it.Key != "" {
			p.ExtraManagements[it.Key] = it.TitleFa
		}
	}
	return p
}

// PreviewAllocation متقاضیان را امتیازبندی و رتبه‌بندی می‌کند تا واحد وام بر پایهٔ
// امتیاز (مدرک تحصیلی، ایثارگری، نوبتکاری، سابقه و…) تخصیص دهد. این مسیر چیزی ذخیره
// نمی‌کند و فقط پیش‌نمایش می‌سازد.
func (h *Handler) PreviewAllocation(w http.ResponseWriter, r *http.Request) {
	var req contracts.LoanScoringRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	want := map[string]bool{}
	for _, id := range req.EmployeeIDs {
		if id = strings.TrimSpace(id); id != "" {
			want[id] = true
		}
	}
	applicants := domain.DefaultApplicants()
	if len(want) > 0 {
		filtered := applicants[:0]
		for _, a := range applicants {
			if want[a.EmployeeID] {
				filtered = append(filtered, a)
			}
		}
		applicants = filtered
	}
	if len(applicants) == 0 {
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeBadRequest,
			"برای امتیازبندی، دست‌کم یک متقاضی نمونه لازم است؛ کد کارکنان را بررسی کنید.", nil)
		return
	}

	ranked := domain.Rank(applicants)
	rows := make([]contracts.LoanScoreRow, 0, len(ranked))
	for i, res := range ranked {
		breakdown := make([]contracts.LoanScoreLine, 0, len(res.Lines))
		for _, l := range res.Lines {
			breakdown = append(breakdown, contracts.LoanScoreLine{
				Criterion: l.Criterion, CriterionFa: l.CriterionFa, ValueFa: l.ValueFa,
				Score: l.Score, Weight: l.Weight, Points: l.Points,
			})
		}
		rows = append(rows, contracts.LoanScoreRow{
			EmployeeID: res.EmployeeID, EmployeeName: res.EmployeeName,
			Total: res.Total, Rank: i + 1, Breakdown: breakdown,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, contracts.LoanAllocationResult{
		Rows:   rows,
		Count:  len(rows),
		NoteFa: "امتیاز هر معیار = امتیاز گزینه × ضریب اهمیت؛ ترتیب ردیف‌ها همان اولویت تخصیص وام است.",
	})
}

// PreviewGrantAllocation پیش‌نمایش تخصیص وام: چه کسانی، با چه کارمزد و چه مبلغ قسطی
// وام می‌گیرند. چیزی ذخیره نمی‌شود و همان قواعد ثبت را می‌بیند.
func (h *Handler) PreviewGrantAllocation(w http.ResponseWriter, r *http.Request) {
	var req contracts.LoanGrantRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	res, err := h.grantAllocation(r.Context(), req, false)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// GrantAllocation ثبت تخصیص وام: برای هر کارمند هدف یک رکورد «درخواست وام» در وضعیت
// آغازین ساخته می‌شود تا از همان گردش‌کار (امتیازبندی ← تأیید ← پرداخت) بگذرد.
func (h *Handler) GrantAllocation(w http.ResponseWriter, r *http.Request) {
	var req contracts.LoanGrantRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	res, err := h.grantAllocation(r.Context(), req, true)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	if res.Created > 0 {
		h.deps.RecordAudit(r, "loans_retirement", "allocate", req.Management,
			"تخصیص وام «"+domain.LoanTypeTitleFa(strings.TrimSpace(req.LoanType))+"» برای "+
				itoa(res.Created)+" نفر — جمع "+itoa64(int64(res.TotalAmount))+" ریال")
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

// grantAllocation پیش‌نمایش یا ثبت تخصیص را انجام می‌دهد. هر دو مسیر یک قاعده و یک
// محاسبه دارند تا پیش‌نمایش و نتیجهٔ ثبت هرگز از هم جدا نشوند.
func (h *Handler) grantAllocation(ctx context.Context, req contracts.LoanGrantRequest, create bool) (contracts.LoanGrantResult, error) {
	grant := domain.LoanGrantRequest{
		LoanType: req.LoanType, Bank: req.Bank, Management: req.Management,
		Amount: req.Amount, Installments: req.Installments, ReceivedDate: req.ReceivedDate,
		Purpose: req.Purpose, GuarantorName: req.GuarantorName,
		GuarantorNationalID: req.GuarantorNationalID, GuarantorRelation: req.GuarantorRelation,
		SupplementOf: req.SupplementOf, Unit: req.Unit,
	}
	terms, err := domain.ComputeLoanTermsWith(grant, h.policy(ctx))
	if err != nil {
		return contracts.LoanGrantResult{}, err
	}
	targets, err := h.targets(ctx, req.EmployeeIDs)
	if err != nil {
		return contracts.LoanGrantResult{}, err
	}
	if len(targets) == 0 {
		return contracts.LoanGrantResult{}, &recordkit.RuleError{Message: "برای تخصیص وام، دست‌کم یک کارمند لازم است."}
	}

	actor := h.actor(ctx)
	loanTypeFa := domain.LoanTypeTitleFa(strings.TrimSpace(req.LoanType))
	// عنوان بانک و مدیریت از همان فهرستی می‌آید که فرم نشان می‌دهد (پایه + کاتالوگ
	// مرجع)، پس قلم تازهٔ پنل تنظیمات در خروجی هم با نام فارسی‌اش دیده می‌شود.
	bankFa := h.bankTitle(ctx, req.Bank)
	managementFa := h.managementTitle(ctx, req.Management)
	fields := domain.LoanFields(grant, terms)

	out := contracts.LoanGrantResult{
		Rows:   make([]contracts.LoanGrantRow, 0, len(targets)),
		NoteFa: "هر ردیف یک رکورد درخواست وام است؛ کارمزد و مبلغ هر قسط از سیاست همان نوع وام محاسبه شده است.",
	}
	for _, e := range targets {
		row := contracts.LoanGrantRow{
			EmployeeID: e.ID, EmployeeName: e.FullName, DepartmentName: e.DepartmentName,
			LoanTypeFa: loanTypeFa, BankFa: bankFa,
			ManagementKey: req.Management, ManagementFa: managementFa,
			UnitKey: strings.TrimSpace(req.Unit), UnitFa: h.unitTitle(ctx, req.Unit),
			Amount: float64(req.Amount), FeeRatePercent: terms.FeeRatePercent,
			FeeAmount: float64(terms.FeeAmount), Installments: terms.Installments,
			InstallmentAmount: float64(terms.InstallmentAmount),
		}
		if create {
			rec, err := h.svc.Create(ctx, contracts.RecordInput{
				Kind:       "loans",
				Title:      loanTypeFa + " — " + e.FullName,
				EmployeeID: e.ID,
				Fields:     fields,
				NoteFa:     strings.TrimSpace(req.NoteFa),
			}, actor)
			if err != nil {
				return out, err
			}
			row.RecordID = rec.ID
			out.Created++
		}
		out.Rows = append(out.Rows, row)
		out.TotalAmount += row.Amount
		out.TotalFee += row.FeeAmount
	}
	out.Count = len(out.Rows)
	if create {
		out.NoteFa = "تخصیص ثبت شد: " + itoa(out.Created) + " رکورد درخواست وام ساخته شد و باید از گردش‌کار امتیازبندی و تأیید بگذرد."
	}
	return out, nil
}

// targets کارکنان هدف تخصیص را از پورت کارکنان می‌خواند. اگر فهرست شناسه خالی باشد،
// همهٔ کارکنان فعال سازمان هدف می‌شوند.
func (h *Handler) targets(ctx context.Context, ids []string) ([]contracts.Employee, error) {
	if h.deps.Ports == nil {
		return nil, &recordkit.RuleError{Message: "دادهٔ کارکنان در این استقرار در دسترس نیست."}
	}
	reader, ok := h.deps.Ports.Employees()
	if !ok {
		return nil, &recordkit.RuleError{Message: "دادهٔ کارکنان در این استقرار در دسترس نیست."}
	}
	want := map[string]bool{}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			want[id] = true
		}
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Page: 1, Size: 1000})
	if err != nil {
		return nil, err
	}
	out := make([]contracts.Employee, 0, len(page.Items))
	for _, e := range page.Items {
		if len(want) > 0 && !want[e.ID] {
			continue
		}
		if len(want) == 0 && e.Status == "terminated" {
			continue // تخصیص وام به کارکنان خاتمه‌یافته معنا ندارد.
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (h *Handler) actor(ctx context.Context) string {
	if u, ok := auth.UserFrom(ctx); ok {
		return u.Username
	}
	return "system"
}

// writeRuleError خطای قاعده را با کد ۴۲۲ برمی‌گرداند (خطای کاربر، نه خطای سرور).
func writeRuleError(w http.ResponseWriter, err error) {
	httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeBadRequest, err.Error(), nil)
}

func itoa(v int) string { return strconv.Itoa(v) }

func itoa64(v int64) string { return strconv.FormatInt(v, 10) }
