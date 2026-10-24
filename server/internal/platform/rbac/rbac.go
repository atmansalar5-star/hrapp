// Package rbac اجرای سیاست دسترسی نقش‌محور در سرور.
//
// ماتریس دسترسی از مانیفست ماژول‌ها تولید می‌شود (platform/modules/registry.gen.go) و
// همان مانیفست، جدول سیاست هستهٔ Rust را هم می‌سازد؛ پس کلاینت و سرور یک منبع واحد
// دارند. این پکیج سرور را «مرجع نهایی» می‌کند: حتی اگر کلاینت دستکاری شود، سرور
// درخواست غیرمجاز را رد می‌کند و رخداد را در لاگ حسابرسی ثبت می‌کند.
package rbac

import (
	"net/http"

	"hrms/internal/contracts"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modules"
)

// GrantSource منبع سطوح دسترسی افزودهٔ مدیر سیستم (پیاده‌سازی در platform/access).
// این پکیج فقط این قرارداد کوچک را می‌شناسد تا وابستگی حلقه‌ای پیش نیاید.
type GrantSource interface {
	Allows(role, departmentID, moduleID, action, organizationID, holdingFa string) bool
	// GrantedModules ماژول‌هایی که این نقش در این اداره و دامنه دسترسی افزوده دارد (برای منو).
	GrantedModules(role, departmentID, organizationID, holdingFa string) []string
}

// Authorizer اجرای سیاست روی رجیستری ماژول‌ها.
//
// دامنهٔ فعال (شرکت/هلدینگ) اینجا نگه داشته می‌شود: سطوح افزودهٔ شركت‌محور فقط در همان
// شرکت و سطوح هلدینگ‌محور فقط در شرکت‌های همان هلدینگ اعمال می‌شوند. نقش مدیر سیستم از
// ماتریس مانیفست همهٔ دسترسی‌ها را دارد و به دامنهٔ شرکت محدود نمی‌شود.
type Authorizer struct {
	registry       *modules.Registry
	audit          *audit.Log
	grants         []GrantSource
	organizationID string
	holdingFa      string
}

// New مجری سیاست می‌سازد. منابع سطح دسترسی اختیاری‌اند؛ هر کدام فقط می‌تواند دسترسی
// بیفزاید، نه بگیرد.
func New(registry *modules.Registry, auditLog *audit.Log, grants ...GrantSource) *Authorizer {
	return &Authorizer{registry: registry, audit: auditLog, grants: grants}
}

// WithTenant دامنهٔ فعال (شرکت و هلدینگ) را تنظیم می‌کند و همان مجری را برمی‌گرداند تا
// سیم‌کشی روان بماند.
func (a *Authorizer) WithTenant(organizationID, holdingFa string) *Authorizer {
	a.organizationID = organizationID
	a.holdingFa = holdingFa
	return a
}

// Tenant دامنهٔ فعال کنونی را برمی‌گرداند.
func (a *Authorizer) Tenant() (organizationID, holdingFa string) {
	return a.organizationID, a.holdingFa
}

// Allows آیا نقش اجازهٔ اقدام در ماژول را دارد (مستقل از اداره).
func (a *Authorizer) Allows(role contracts.Role, moduleID, action string) bool {
	return a.AllowsFor(role, "", moduleID, action)
}

// AllowsFor آیا نقش در ادارهٔ داده‌شده اجازهٔ اقدام در ماژول را دارد. اول ماتریس
// مانیفست، بعد سطوح افزودهٔ مدیر سیستم. ماژول خاموش در این استقرار با هیچ سطحی روشن
// نمی‌شود.
func (a *Authorizer) AllowsFor(role contracts.Role, departmentID, moduleID, action string) bool {
	if !contracts.ValidRole(role) {
		return false
	}
	m, ok := a.registry.ByID(moduleID)
	if !ok {
		// ماژول غیرفعال در این استقرار = دسترسی ندارد.
		return false
	}
	if m.Can(string(role), action) {
		return true
	}
	for _, g := range a.grants {
		if g != nil && g.Allows(string(role), departmentID, moduleID, action, a.organizationID, a.holdingFa) {
			return true
		}
	}
	return false
}

// Scope دامنهٔ دادهٔ نقش در ماژول.
func (a *Authorizer) Scope(role contracts.Role, moduleID string) string {
	m, ok := a.registry.ByID(moduleID)
	if !ok {
		return "own"
	}
	return m.ScopeFor(string(role))
}

// Menu منوی نقشمحور برای کلاینت.
func (a *Authorizer) Menu(role contracts.Role) []modules.MenuEntry {
	return a.MenuFor(role, "")
}

// MenuFor منوی نقض‌محور با در نظر گرفتن سطوح افزودهٔ همان اداره: نقشی که مدیر سیستم
// به آن دسترسی داده، ماژول را در ناوبری می‌بیند.
func (a *Authorizer) MenuFor(role contracts.Role, departmentID string) []modules.MenuEntry {
	extra := []string{}
	for _, g := range a.grants {
		if g != nil {
			extra = append(extra, g.GrantedModules(string(role), departmentID, a.organizationID, a.holdingFa)...)
		}
	}
	return a.registry.MenuForWithExtra(string(role), extra)
}

// CanOpenScreen آیا نقش اجازهٔ باز کردن صفحهٔ مشخص را دارد.
func (a *Authorizer) CanOpenScreen(role contracts.Role, moduleID, screenKey string) bool {
	m, ok := a.registry.ByID(moduleID)
	if !ok {
		return false
	}
	for _, s := range m.ScreensFor(string(role)) {
		if s.Key == screenKey {
			return true
		}
	}
	return false
}

// Require میان‌افزار مسیرهای محافظت‌شده: نخست نشست، سپس دسترسی ماژول.
func (a *Authorizer) Require(moduleID, action string, next http.HandlerFunc) http.HandlerFunc {
	return auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFrom(r.Context())
		if !a.AllowsFor(principal.User.Role, principal.User.DepartmentID, moduleID, action) {
			if a.audit != nil {
				a.audit.Append(
					principal.User.Username,
					moduleID,
					"denied:"+action,
					r.URL.Path,
					"نقش "+string(principal.User.Role)+" دسترسی نداشت",
				)
			}
			httpx.WriteErrorMsg(
				w,
				http.StatusForbidden,
				contracts.CodeForbidden,
				"نقش شما («"+contracts.RoleFa[principal.User.Role]+"») اجازهٔ این عملیات را ندارد.",
				map[string]string{"module": moduleID, "action": action},
			)
			return
		}
		next(w, r)
	})
}

// RequireRole میان‌افزار مسیرهایی که فقط نقش مشخص می‌پذیرند (مانند تنظیمات کاربران).
func (a *Authorizer) RequireRole(roles ...contracts.Role) func(http.HandlerFunc) http.HandlerFunc {
	allowed := make(map[contracts.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			principal, _ := auth.PrincipalFrom(r.Context())
			if !allowed[principal.User.Role] {
				httpx.WriteError(w, http.StatusForbidden, contracts.CodeForbidden, nil)
				return
			}
			next(w, r)
		})
	}
}
