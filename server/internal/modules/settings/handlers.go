// پایانه‌های HTTP تنظیمات (کاربران، نقش‌ها، ماژول‌ها) — بررسی privilege escalation.

package settings

import (
	"net/http"

	"hrms/internal/contracts"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
)

// registerAdminEndpoints مسیرهای دست‌ساز مدیریت کاربران، نقش‌ها و لاگ حسابرسی را ثبت می‌کند.
//
// این صفحه‌ها ذاتاً «فهرست‌های سیستمی» هستند (نه رکورد گردش‌کاری)، پس مثل پارامترها روی
// موتور recordkit سوار نمی‌شوند و اینجا مستقیم سرو می‌شوند. دسترسی همه با نقش‌محوری
// کنترل می‌شود؛ این صفحه‌ها ویژهٔ مدیر سیستم‌اند و حتی مدیر منابع انسانی به لاگ حسابرسی
// نمی‌رسد.
func registerAdminEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	read := deps.Policy.RequireRole(contracts.RoleAdmin)

	mux.HandleFunc("GET /api/v1/settings/users", read(func(w http.ResponseWriter, r *http.Request) {
		users := []contracts.User{}
		if deps.Auth != nil {
			users = deps.Auth.Users()
		}
		httpx.WriteJSON(w, http.StatusOK, users)
	}))

	mux.HandleFunc("GET /api/v1/settings/roles", read(func(w http.ResponseWriter, r *http.Request) {
		type roleView struct {
			Value   string `json:"value"`
			LabelFa string `json:"labelFa"`
		}
		roles := make([]roleView, 0, len(contracts.Roles))
		for _, role := range contracts.Roles {
			roles = append(roles, roleView{Value: string(role), LabelFa: contracts.RoleFa[role]})
		}
		httpx.WriteJSON(w, http.StatusOK, roles)
	}))

	mux.HandleFunc("GET /api/v1/settings/audit", read(func(w http.ResponseWriter, r *http.Request) {
		events := []audit.Event{}
		chainValid := true
		if deps.Audit != nil {
			events = deps.Audit.All()
			chainValid = deps.Audit.Verify() == nil
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"events":     events,
			"chainValid": chainValid,
			"total":      len(events),
		})
	}))
}
