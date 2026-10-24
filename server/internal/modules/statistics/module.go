// Package statistics ماژول آمار و پایش اطلاعات پایه.
//
// پیاده‌سازی واقعی روی موتور مشترک `recordkit`: تأیید اطلاعات پایه و برش آماری هر یک یک
// نوع رکورد با گردش‌کار خودش است و قواعدش در بستهٔ domain اعلام شده.
package statistics

import (
	"net/http"

	"hrms/internal/modules/statistics/domain"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// Module ثبت ماژول آمار و پایش اطلاعات پایه.
type Module struct{}

// New ماژول می‌سازد.
func New() *Module { return &Module{} }

// ID شناسهٔ ماژول.
func (m *Module) ID() string { return "statistics" }

// Register مسیرهای استاندارد رکوردهای آمار، مسیرهای تجمیعی (برش آماری) و تب‌های
// اختصاصی آمار (مرخصی، مأموریت، کارانه، داشبورد) را ثبت می‌کند.
func (m *Module) Register(mux *http.ServeMux, deps modulekit.Deps) {
	recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/statistics")
	registerAnalyticsEndpoints(mux, deps)
	registerTabEndpoints(mux, deps)
}
