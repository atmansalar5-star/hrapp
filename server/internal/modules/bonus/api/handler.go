// Package api لایهٔ HTTP اختصاصی ماژول پاداش.
//
// مسیرهای استاندارد رکوردها (schema/records/action) از موتور recordkit می‌آید؛ این بسته
// چهار کار اختصاصی را انجام می‌دهد: اعلام گزینه‌ها، داشبورد مدیریتی، پیش‌نمایش تخصیص
// پاداش و ثبت گروهی تخصیص. هیچ نرخ یا برچسبی اینجا هاردکد نیست و همه از دامنه می‌آید.
package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/bonus/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Handler مسیرهای اختصاصی ماژول پاداش.
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

func (h *Handler) today() string {
	if t, err := calendar.FromGregorian(h.now()); err == nil {
		return t.String()
	}
	return ""
}

// Options گزینه‌های پایه: انواع پاداش، مدیریت‌ها، روش‌های پرداخت و امروز.
//
// رابط از همین فرم و پنل تخصیص را می‌سازد؛ افزودن نوع پاداش یا مدیریت تازه در دامنه،
// خودکار در فرم دیده می‌شود.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	out := contracts.BonusChoices{
		Types:          []contracts.BonusChoice{},
		Managements:    h.managementOptions(r.Context()),
		PaymentMethods: []contracts.BonusChoice{},
		TodayFa:        h.today(),
	}
	for _, t := range domain.BonusTypes() {
		out.Types = append(out.Types, contracts.BonusChoice{
			Key: t.Key, TitleFa: t.TitleFa, DefaultAmount: t.DefaultAmount,
			RequiresDecree: t.RequiresDecree, NoteFa: t.NoteFa,
		})
	}
	for _, m := range domain.PaymentMethods() {
		out.PaymentMethods = append(out.PaymentMethods, contracts.BonusChoice{Key: m.Key, TitleFa: m.TitleFa})
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// Dashboard داشبورد مدیریتی پاداش: پرداخت به تفکیک مدیریت، نفر، نوع، دوره و وضعیت.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	facts, err := h.facts(r.Context())
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, domain.BuildDashboard(facts))
}

// PreviewAllocation پیش‌نمایش تخصیص: چه کسانی چه مبلغی می‌گیرند و شمارهٔ دفعهٔ هرکس
// چندم است. چیزی ذخیره نمی‌شود.
func (h *Handler) PreviewAllocation(w http.ResponseWriter, r *http.Request) {
	var req contracts.BonusAllocationRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	res, err := h.allocate(r.Context(), req, false)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// Allocate ثبت تخصیص: برای هر ردیف پیش‌نمایش، یک رکورد پرداخت پاداش در وضعیت پیش‌نویس
// ساخته می‌شود تا از همان مسیر گردش‌کار (تأیید → پرداخت) بگذرد.
func (h *Handler) Allocate(w http.ResponseWriter, r *http.Request) {
	var req contracts.BonusAllocationRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	res, err := h.allocate(r.Context(), req, true)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	if len(res.Rows) > 0 {
		h.deps.RecordAudit(r, "bonus", "allocate", req.Management,
			"تخصیص پاداش برای "+itoa(res.Created)+" نفر — جمع "+itoa64(int64(res.TotalAmount))+" ریال")
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

/* --------------------------- منطق تخصیص --------------------------- */

// allocate پیش‌نمایش یا ثبت تخصیص گروهی را انجام می‌دهد. هر دو مسیر یک قاعده دارند تا
// پیش‌نمایش و نتیجهٔ ثبت هرگز از هم جدا نشوند.
func (h *Handler) allocate(ctx context.Context, req contracts.BonusAllocationRequest, create bool) (contracts.BonusAllocationResult, error) {
	if err := h.validateAllocationWith(ctx, req); err != nil {
		return contracts.BonusAllocationResult{}, err
	}
	targets, err := h.targets(ctx, req.EmployeeIDs)
	if err != nil {
		return contracts.BonusAllocationResult{}, err
	}
	if len(targets) == 0 {
		return contracts.BonusAllocationResult{}, &recordkit.RuleError{Message: "برای تخصیص، دست‌کم یک کارمند لازم است."}
	}

	// شمارهٔ دفعه از پرداخت‌های پیشین همان کارمند در همان مدیریت ادامه می‌یابد تا
	// «چندمین پرداخت» در گزارش‌ها درست باشد.
	sequences, err := h.sequences(ctx, req.Management)
	if err != nil {
		return contracts.BonusAllocationResult{}, err
	}

	actor := h.actor(ctx)
	bonusTypeFa := domain.BonusTypeTitleFa(req.BonusType)
	managementFa := contracts.ManagementTitleFa(req.Management)
	fields := map[string]string{
		"bonusType":     req.BonusType,
		"management":    req.Management,
		"amount":        itoa64(req.Amount),
		"paymentDate":   strings.TrimSpace(req.PaymentDate),
		"paymentMethod": strings.TrimSpace(req.PaymentMethod),
		"decreeNo":      strings.TrimSpace(req.DecreeNo),
		"reference":     strings.TrimSpace(req.Reference),
	}

	out := contracts.BonusAllocationResult{
		Rows:   make([]contracts.BonusAllocationRow, 0, len(targets)),
		NoteFa: "هر ردیف یک رویداد پرداخت است؛ شمارهٔ دفعه از پرداخت‌های پیشین همان کارمند در همان مدیریت ادامه می‌یابد.",
	}
	for _, e := range targets {
		seq := sequences[e.ID] + 1
		row := contracts.BonusAllocationRow{
			EmployeeID: e.ID, EmployeeName: e.FullName, DepartmentName: e.DepartmentName,
			ManagementKey: req.Management, ManagementFa: managementFa,
			BonusTypeFa: bonusTypeFa, Amount: float64(req.Amount), Sequence: seq,
		}
		if create {
			rec, err := h.svc.Create(ctx, contracts.RecordInput{
				Kind:       "bonusPayments",
				Title:      bonusTypeFa + " — " + e.FullName,
				EmployeeID: e.ID,
				Fields:     withSequence(fields, seq),
				NoteFa:     strings.TrimSpace(req.NoteFa),
			}, actor)
			if err != nil {
				return out, err
			}
			row.RecordID = rec.ID
			sequences[e.ID] = seq
			out.Created++
		}
		out.Rows = append(out.Rows, row)
		out.TotalAmount += row.Amount
	}
	out.Count = len(out.Rows)
	if create {
		out.NoteFa = "تخصیص ثبت شد: " + itoa(out.Created) + " رکورد پرداخت در وضعیت پیش‌نویس ساخته شد و باید از گردش‌کار تأیید و پرداخت شود."
	}
	return out, nil
}

// managementOptions مدیریت‌های مقصد: کاتالوگ پایهٔ قراردادها به‌علاوهٔ اقلام پنل تنظیمات.
// قلم هم‌نام، عنوان تازه می‌گیرد و قلم غیرفعال کنار گذاشته می‌شود؛ پس مدیریت تازه‌ای که
// در پنل تعریف می‌شود، خودکار در فرم تخصیص پاداش هم دیده می‌شود.
func (h *Handler) managementOptions(ctx context.Context) []contracts.Management {
	base := contracts.MergeManagementKeys(contracts.Managements(), h.catalogItems(ctx, contracts.CatalogManagements))
	keys := make([]string, 0, len(base))
	for k := range base {
		keys = append(keys, k)
	}
	// ترتیب کاتالوگ پایه حفظ می‌شود و اقلام تازهٔ پنل به انتهای فهرست می‌آیند.
	sort.Slice(keys, func(i, j int) bool { return managementOrder(keys[i]) < managementOrder(keys[j]) })
	out := make([]contracts.Management, 0, len(keys))
	for _, k := range keys {
		out = append(out, contracts.Management{Key: k, TitleFa: base[k]})
	}
	return out
}

// managementOrder جایگاه یک مدیریت در ترتیب نمایش: فهرست پایه به ترتیب اعلامی سازمان و
// اقلام تازهٔ پنل بعد از آن‌ها.
func managementOrder(key string) int {
	for i, m := range contracts.Managements() {
		if m.Key == key {
			return i
		}
	}
	return len(contracts.Managements())
}

// catalogItems اقلام کاتالوگ مرجع را از پنل تنظیمات می‌خواند؛ نبودِ آن ماژول یعنی فهرست
// خالی و بازگشت به کاتالوگ پایه.
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

// validManagement آیا مدیریت مقصد شناخته‌شده است (کاتالوگ پایه یا پنل تنظیمات).
func (h *Handler) validManagement(ctx context.Context, key string) bool {
	if contracts.ValidManagement(key) {
		return true
	}
	for _, it := range h.catalogItems(ctx, contracts.CatalogManagements) {
		if it.Active && it.Key == strings.TrimSpace(key) {
			return true
		}
	}
	return false
}

// validateAllocationWith قواعد ورودی تخصیص — همان قواعد دامنه، پیش از ساخت هر رکورد.
// کاتالوگ مرجع تنظیمات را هم می‌بیند تا مدیریت تازهٔ پنل پذیرفته شود.
func (h *Handler) validateAllocationWith(ctx context.Context, req contracts.BonusAllocationRequest) error {
	bonusType := strings.TrimSpace(req.BonusType)
	t, ok := domain.BonusTypeByKey(bonusType)
	if !ok {
		return &recordkit.RuleError{Message: "انتخاب نوع پاداش معتبر الزامی است."}
	}
	management := strings.TrimSpace(req.Management)
	if management == "" || !h.validManagement(ctx, management) {
		return &recordkit.RuleError{Message: "انتخاب مدیریت مقصد الزامی است."}
	}
	if req.Amount <= 0 {
		return &recordkit.RuleError{Message: "مبلغ هر نفر باید عددی مثبت باشد."}
	}
	if _, err := calendar.Parse(strings.TrimSpace(req.PaymentDate)); err != nil {
		return &recordkit.RuleError{Message: "تاریخ پرداخت باید به قالب شمسی YYYY/MM/DD باشد."}
	}
	if !domain.ValidPaymentMethod(req.PaymentMethod) {
		return &recordkit.RuleError{Message: "روش پرداخت انتخابی مجاز نیست."}
	}
	if t.RequiresDecree && strings.TrimSpace(req.DecreeNo) == "" {
		return &recordkit.RuleError{Message: "برای «" + t.TitleFa + "» ثبت شمارهٔ مصوبه / حکم الزامی است."}
	}
	return nil
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
			continue // تخصیص گروهی به کارکنان خاتمه‌یافته معنا ندارد.
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// sequences شمارهٔ آخرین دفعهٔ پرداخت هر کارمند در یک مدیریت را برمی‌گرداند.
func (h *Handler) sequences(ctx context.Context, management string) (map[string]int, error) {
	out := map[string]int{}
	page, err := h.svc.List(ctx, contracts.RecordFilter{
		Module: "bonus", Kind: "bonusPayments", Scope: "all", Page: 1, Size: 1000,
	})
	if err != nil {
		return out, err
	}
	for _, rec := range page.Items {
		if rec.Status == "cancelled" {
			continue
		}
		if strings.TrimSpace(rec.Fields["management"]) != strings.TrimSpace(management) {
			continue
		}
		if seq := domain.SequenceOf(rec.Fields["sequence"]); seq > out[rec.EmployeeID] {
			out[rec.EmployeeID] = seq
		}
	}
	return out, nil
}

// facts رکوردهای پرداخت را به «واقعیت»های دامنه تبدیل می‌کند و گروه کارگزینی هر کارمند
// را از پورت کارکنان می‌گیرد (مبنای برش شرکتی / ارکان ثالث).
func (h *Handler) facts(ctx context.Context) ([]domain.PaymentFact, error) {
	page, err := h.svc.List(ctx, contracts.RecordFilter{
		Module: "bonus", Kind: "bonusPayments", Scope: "all", Page: 1, Size: 1000,
	})
	if err != nil {
		return nil, err
	}
	groups := map[string]string{}
	if h.deps.Ports != nil {
		if reader, ok := h.deps.Ports.Employees(); ok {
			if empPage, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Page: 1, Size: 1000}); err == nil {
				for _, e := range empPage.Items {
					groups[e.ID] = employmentGroupOf(e.EmploymentType)
				}
			}
		}
	}
	facts := make([]domain.PaymentFact, 0, len(page.Items))
	for _, rec := range page.Items {
		amount, _ := domain.AmountOf(rec.Fields["amount"])
		facts = append(facts, domain.PaymentFact{
			EmployeeID:   rec.EmployeeID,
			EmployeeName: rec.EmployeeName,
			Management:   strings.TrimSpace(rec.Fields["management"]),
			BonusType:    strings.TrimSpace(rec.Fields["bonusType"]),
			Amount:       amount,
			Date:         strings.TrimSpace(rec.Fields["paymentDate"]),
			Period:       domain.PeriodOf(rec.Fields["paymentDate"]),
			Status:       rec.Status,
			Group:        groups[rec.EmployeeID],
		})
	}
	return facts, nil
}

// employmentGroupOf گروه کارگزینی یک نوع استخدام: هر نوع با خانوادهٔ مزد «contracting»
// در ارکان ثالث می‌افتد؛ بقیه شرکتی شمرده می‌شوند.
func employmentGroupOf(employmentType string) string {
	if t, ok := contracts.EmploymentTypeByKey(employmentType); ok {
		if t.Kind == contracts.EmploymentKindContracting {
			return domain.EmploymentGroupThirdParty
		}
	}
	return domain.EmploymentGroupCompany
}

func (h *Handler) actor(ctx context.Context) string {
	if u, ok := auth.UserFrom(ctx); ok {
		return u.Username
	}
	return "system"
}

func withSequence(fields map[string]string, seq int) map[string]string {
	out := make(map[string]string, len(fields)+1)
	for k, v := range fields {
		out[k] = v
	}
	out["sequence"] = itoa(seq)
	return out
}

// writeRuleError خطای قاعده را با کد ۴۲۲ برمی‌گرداند (خطای کاربر، نه خطای سرور).
func writeRuleError(w http.ResponseWriter, err error) {
	if _, ok := err.(*recordkit.ValidationError); ok {
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
		return
	}
	httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeBadRequest, err.Error(), nil)
}

func itoa(v int) string { return strconv.Itoa(v) }

func itoa64(v int64) string { return strconv.FormatInt(v, 10) }
