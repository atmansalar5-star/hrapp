// Package dashboard ماژول داشبورد.
//
// این ماژول نمونهٔ «دسترسی بین‌ماژولی از راه پورت» است: برای شمارش کارکنان به پکیج
// ماژول employee import نمی‌زند، بلکه پورت `contracts.EmployeeReader` را از
// `modulekit.Ports` می‌گیرد. اگر ماژول کارکنان در این استقرار فعال نباشد، داشبورد
// بی‌خطا با کارت‌های خالی پاسخ می‌دهد.
//
// کارت‌ها بر پایهٔ نقش ساخته می‌شوند (بخش ۱ سند طراحی) و محدودهٔ دادهٔ کاربر در کوئری
// اعمال می‌شود؛ یعنی مدیر واحد فقط آمار واحد خودش را می‌بیند.
package dashboard

import (
	"net/http"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// Module ثبت ماژول داشبورد.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "dashboard" }

// Register مسیرهای داشبورد.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	h := &handler{deps: deps}
	mux.HandleFunc("GET /api/v1/dashboard/widgets", deps.Require(m.ID(), "read", h.widgets))
	deps.Logger(m.ID()).Info("ماژول داشبورد ثبت شد")
}

type handler struct {
	deps modulekit.Deps
}

var statusTitlesFa = map[string]string{
	"active":     "کارکنان شاغل",
	"leave":      "مرخصی جاری",
	"probation":  "دورهٔ آزمایشی",
	"terminated": "ترک خدمت",
}

func (h *handler) widgets(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, contracts.CodeUnauthorized, nil)
		return
	}

	reader, available := h.deps.Ports.Employees()
	if !available {
		// ماژول کارکنان در این استقرار فعال نیست؛ پاسخ معتبر ولی خالی می‌دهیم.
		h.deps.Logger("dashboard").Warn("پورت کارکنان در دسترس نیست؛ کارت‌های آماری خالی برگشت")
		httpx.WriteJSON(w, http.StatusOK, []contracts.Widget{})
		return
	}

	// محدودهٔ دادهٔ همان نقشی که فهرست کارکنان را می‌بیند.
	scope := h.deps.Policy.Scope(user.Role, "employee")
	filter := contracts.EmployeeFilter{
		Scope:           scope,
		ActorID:         user.EmployeeID,
		ActorDepartment: user.DepartmentID,
	}

	counts, err := reader.CountByStatus(r.Context(), filter)
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}

	total := 0
	for _, n := range counts {
		total += n
	}

	order := []string{"active", "leave", "probation", "terminated"}
	widgets := make([]contracts.Widget, 0, len(order)+1)
	widgets = append(widgets, contracts.Widget{
		Key:     "headcount",
		TitleFa: "کل کارکنان",
		Value:   float64(total),
		Unit:    "نفر",
	})
	for _, status := range order {
		n, ok := counts[status]
		if !ok {
			continue
		}
		widgets = append(widgets, contracts.Widget{
			Key:     status,
			TitleFa: statusTitlesFa[status],
			Value:   float64(n),
			Unit:    "نفر",
		})
	}

	// کارمند عادی آمار سازمانی نمی‌بیند؛ فقط اطلاعات مربوط به خودش.
	if user.Role == contracts.RoleEmployee {
		widgets = []contracts.Widget{{
			Key:     "own_record",
			TitleFa: "پروندهٔ من",
			Value:   float64(counts["active"] + counts["leave"] + counts["probation"]),
			Unit:    "رکورد",
		}}
	}

	httpx.WriteJSON(w, http.StatusOK, widgets)
}
