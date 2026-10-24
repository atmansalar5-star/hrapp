// کنترل دسترسی ریزدانهٔ ماژول تنظیمات (دپارتمان/شرکت/نقش/دامنهٔ گزارش) — نقطهٔ حیاتی تست نفوذ.

package settings

import (
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/access"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/modules"
	"hrms/internal/platform/orgs"
)

// accessActions اقدام‌های استاندارد سامانه برای ماتریس دسترسی.
var accessActions = []string{"read", "write", "delete", "approve", "admin"}

// actionFa برچسب فارسی اقدام‌ها.
var actionFa = map[string]string{
	"read": "مشاهده", "write": "ایجاد/ویرایش", "delete": "حذف", "approve": "تأیید", "admin": "مدیریت",
}

// registerAccessEndpoints مسیرهای «سطوح دسترسی» را ثبت می‌کند. این‌ها ویژهٔ مدیر سیستم‌اند
// و اجازه می‌دهند برای هر اداره و هر ماژول، به یک نقش دسترسی بیشتری بدهد یا بگیرد.
//
// ماتریس پایه از مانیفست ماژول‌ها می‌آید؛ این مسیرها فقط لایهٔ افزودنی را مدیریت می‌کنند.
func registerAccessEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	admin := deps.Policy.RequireRole(contracts.RoleAdmin)

	mux.HandleFunc("GET /api/v1/settings/access", admin(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, accessMatrix(deps, resolveScope(deps, r)))
	}))

	mux.HandleFunc("POST /api/v1/settings/access/grants", admin(func(w http.ResponseWriter, r *http.Request) {
		if deps.Access == nil {
			httpx.WriteErrorMsg(w, http.StatusNotImplemented, contracts.CodeNotImplemented, "انبار سطوح دسترسی در این اجرا فعال نیست", nil)
			return
		}
		var body struct {
			Role           string   `json:"role"`
			ModuleID       string   `json:"moduleId"`
			DepartmentID   string   `json:"departmentId"`
			Actions        []string `json:"actions"`
			NoteFa         string   `json:"noteFa"`
			OrganizationID string   `json:"organizationId"`
			HoldingFa      string   `json:"holdingFa"`
			Scope          string   `json:"scope"` // company | holding | all
		}
		if !httpx.DecodeJSON(w, r, &body) {
			return
		}
		orgID, holdingFa := grantScope(deps, body.OrganizationID, body.HoldingFa, body.Scope)
		if details := validateGrant(deps, body.Role, body.ModuleID, body.DepartmentID, body.Actions, orgID, holdingFa); details != nil {
			httpx.WriteError(w, http.StatusBadRequest, contracts.CodeValidationFailed, details)
			return
		}
		actor := ""
		if p, ok := auth.PrincipalFrom(r.Context()); ok {
			actor = p.User.Username
		}
		g := deps.Access.Upsert(access.Grant{
			Role:           strings.TrimSpace(body.Role),
			ModuleID:       strings.TrimSpace(body.ModuleID),
			DepartmentID:   strings.TrimSpace(body.DepartmentID),
			OrganizationID: orgID,
			HoldingFa:      holdingFa,
			Actions:        body.Actions,
			NoteFa:         body.NoteFa,
			CreatedBy:      actor,
		})
		deps.RecordAudit(r, "settings", "access-grant", g.ID, "سطح دسترسی "+g.Role+" × "+g.ModuleID+" — دامنه "+grantScopeFa(g))
		httpx.WriteJSON(w, http.StatusOK, g)
	}))

	mux.HandleFunc("DELETE /api/v1/settings/access/grants/{id}", admin(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if deps.Access == nil || !deps.Access.Delete(id) {
			httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
			return
		}
		deps.RecordAudit(r, "settings", "access-revoke", id, "حذف سطح دسترسی")
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": id})
	}))
}

// accessMatrix ماتریس کامل دسترسی را برای صفحهٔ مدیریت می‌سازد: نقش‌ها، اداره‌ها،
// ماژول‌ها با اقدام‌های مانیفست و اقدام‌های افزوده، و فهرست سطوح.
func accessMatrix(deps modulekit.Deps, scope tenantScope) map[string]any {
	roles := make([]map[string]string, 0, len(contracts.Roles))
	for _, role := range contracts.Roles {
		roles = append(roles, map[string]string{"value": string(role), "labelFa": contracts.RoleFa[role]})
	}
	deptList, deptTitles := knownDepartments(deps)
	departments := deptList

	// اقدام‌های افزوده را یک‌بار بر پایهٔ ماژول و نقش جمع می‌کنیم (مستقل از اداره).
	granted := map[string]map[string][]string{}
	if deps.Access != nil {
		for _, g := range deps.Access.Grants() {
			// فقط سطوحی که به دامنهٔ فعال (شرکت یا هلدینگ باز) مربوط‌اند نمایش داده می‌شوند.
			if !grantVisible(g, scope) {
				continue
			}
			if granted[g.ModuleID] == nil {
				granted[g.ModuleID] = map[string][]string{}
			}
			granted[g.ModuleID][g.Role] = append(granted[g.ModuleID][g.Role], g.Actions...)
		}
	}

	type moduleView struct {
		ID           string              `json:"id"`
		TitleFa      string              `json:"titleFa"`
		Department   string              `json:"department"`
		DepartmentFa string              `json:"departmentFa"`
		Manifest     map[string][]string `json:"manifest"`
		Granted      map[string][]string `json:"granted"`
	}
	views := make([]moduleView, 0, deps.Registry.Count())
	for _, m := range deps.Registry.All() {
		mv := moduleView{
			ID: m.ID, TitleFa: m.TitleFa, Department: m.Department,
			DepartmentFa: deptTitles[m.Department],
			Manifest:     map[string][]string{}, Granted: map[string][]string{},
		}
		for _, role := range contracts.Roles {
			if acts := m.ActionsFor(string(role)); len(acts) > 0 {
				mv.Manifest[string(role)] = acts
			}
		}
		for role, acts := range granted[m.ID] {
			mv.Granted[role] = dedupe(acts)
		}
		views = append(views, mv)
	}

	actions := make([]map[string]string, 0, len(accessActions)+1)
	for _, a := range accessActions {
		actions = append(actions, map[string]string{"value": a, "labelFa": actionFa[a]})
	}
	actions = append(actions, map[string]string{"value": access.AnyAction, "labelFa": "همهٔ اقدام‌ها"})

	grants := []access.Grant{}
	if deps.Access != nil {
		for _, g := range deps.Access.Grants() {
			if grantVisible(g, scope) {
				grants = append(grants, g)
			}
		}
	}

	return map[string]any{
		"roles":       roles,
		"departments": departments,
		"modules":     views,
		"actions":     actions,
		"grants":      grants,
		"holdings":    holdingsView(),
		"scope":       scope.view(),
	}
}

// validateGrant ورودی سطح دسترسی را می‌سنجد؛ nil یعنی معتبر.
func validateGrant(deps modulekit.Deps, role, moduleID, departmentID string, actions []string, organizationID, holdingFa string) map[string]string {
	details := map[string]string{}
	if organizationID = strings.TrimSpace(organizationID); organizationID != "" {
		if _, ok := orgs.Lookup(organizationID); !ok {
			details["organizationId"] = "شرکت نامعتبر است."
		}
	}
	if holdingFa = strings.TrimSpace(holdingFa); holdingFa != "" {
		if _, ok := orgs.HoldingByKey(holdingFa); !ok {
			details["holdingFa"] = "هلدینگ نامعتبر است."
		}
	}
	if !contracts.ValidRole(contracts.Role(strings.TrimSpace(role))) {
		details["role"] = "نقش نامعتبر است."
	}
	moduleID = strings.TrimSpace(moduleID)
	if moduleID != access.AnyModule {
		if _, ok := deps.Registry.ByID(moduleID); !ok {
			details["moduleId"] = "ماژول در این استقرار فعال نیست."
		}
	}
	if departmentID = strings.TrimSpace(departmentID); departmentID != "" {
		_, titles := knownDepartments(deps)
		if _, ok := titles[departmentID]; !ok {
			details["departmentId"] = "ادارهٔ نامعتبر است."
		}
	}
	if len(actions) == 0 {
		details["actions"] = "حداقل یک اقدام لازم است."
	}
	for _, a := range actions {
		if a == access.AnyAction {
			continue
		}
		if _, ok := actionFa[a]; !ok {
			details["actions"] = "اقدام نامعتبر است: " + a
		}
	}
	if len(details) > 0 {
		return details
	}
	return nil
}

// knownDepartments فهرست اداره‌های معتبر را می‌سازد: اداره‌های منو (با عنوان فارسی) به‌علاوهٔ
// اداره‌هایی که حساب‌های کاربری واقعاً دارند (مثل D1/D2). این دو فضای شناسه جدا هستند و
// سطح دسترسی باید بتواند هر دو را هدف بگیرد؛ برای دومی عنوان در دسترس نیست، پس همان
// شناسه نمایش داده می‌شود.
func knownDepartments(deps modulekit.Deps) ([]map[string]string, map[string]string) {
	seen := map[string]bool{}
	titles := map[string]string{}
	out := []map[string]string{}
	for _, d := range modules.Departments {
		if seen[d.ID] {
			continue
		}
		seen[d.ID] = true
		titles[d.ID] = d.TitleFa
		out = append(out, map[string]string{"id": d.ID, "titleFa": d.TitleFa})
	}
	if deps.Auth != nil {
		for _, u := range deps.Auth.Users() {
			id := strings.TrimSpace(u.DepartmentID)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			titles[id] = id
			out = append(out, map[string]string{"id": id, "titleFa": id})
		}
	}
	return out, titles
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

/* ------------------------- دامنهٔ شرکت و هلدینگ ------------------------- */

// tenantScope دامنهٔ نمای کنترل پنل: یک شرکت یا یک هلدینگ همراه شرکت‌های زیرمجموعه.
type tenantScope struct {
	OrgID      string // نمای شرکت (در نمای هلدینگ خالی است)
	HoldingFa  string // نمای هلدینگ
	CompanyIDs map[string]bool
	TitleFa    string
}

func (s tenantScope) view() map[string]any {
	ids := make([]string, 0, len(s.CompanyIDs))
	for id := range s.CompanyIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	kind := "company"
	if s.OrgID == "" {
		kind = "holding"
	}
	return map[string]any{
		"kind": kind, "organizationId": s.OrgID, "holdingFa": s.HoldingFa,
		"titleFa": s.TitleFa, "companies": ids,
	}
}

// resolveScope دامنهٔ فعال را از پارامترهای درخواست می‌سازد: ?holding=… یا ?org=…؛
// پیش‌فرض همان شرکت فعال سرور است. این کار «باز کردن کنترل پنلِ یک شرکت یا یک هلدینگ»
// است: هلدینگ یعنی همهٔ شرکت‌های زیرمجموعه‌اش.
func resolveScope(deps modulekit.Deps, r *http.Request) tenantScope {
	if key := strings.TrimSpace(r.URL.Query().Get("holding")); key != "" {
		if h, ok := orgs.HoldingByKey(key); ok {
			ids := map[string]bool{}
			for _, c := range h.Companies {
				ids[c.ID] = true
			}
			return tenantScope{HoldingFa: h.Key, CompanyIDs: ids, TitleFa: h.TitleFa}
		}
	}
	if id := strings.TrimSpace(r.URL.Query().Get("org")); id != "" {
		if o, ok := orgs.Lookup(id); ok {
			return tenantScope{OrgID: o.ID, HoldingFa: orgs.HoldingKey(o), CompanyIDs: map[string]bool{o.ID: true}, TitleFa: o.TitleFa}
		}
	}
	o := deps.Org
	return tenantScope{OrgID: o.ID, HoldingFa: orgs.HoldingKey(o), CompanyIDs: map[string]bool{o.ID: true}, TitleFa: o.TitleFa}
}

// grantVisible آیا این سطح به دامنهٔ فعال مربوط است (برای فهرست کنترل پنل).
func grantVisible(g access.Grant, scope tenantScope) bool {
	if g.OrganizationID == "" && g.HoldingFa == "" {
		return true // سطح جهانی
	}
	if g.OrganizationID != "" {
		return scope.CompanyIDs[g.OrganizationID]
	}
	return scope.HoldingFa != "" && g.HoldingFa == scope.HoldingFa
}

// grantScope دامنهٔ نهایی سطح تازه: شرکت صریح، هلدینگ صریح، «all» برای سطح جهانی، و
// در غیر این صورت شرکت فعال سرور.
func grantScope(deps modulekit.Deps, orgID, holdingFa, scope string) (string, string) {
	orgID = strings.TrimSpace(orgID)
	holdingFa = strings.TrimSpace(holdingFa)
	if orgID != "" {
		return orgID, ""
	}
	if holdingFa != "" {
		return "", holdingFa
	}
	switch strings.TrimSpace(scope) {
	case "all":
		return "", ""
	case "holding":
		return "", orgs.HoldingKey(deps.Org)
	default:
		return deps.Org.ID, ""
	}
}

// grantScopeFa عنوان فارسی دامنهٔ یک سطح.
func grantScopeFa(g access.Grant) string {
	if g.OrganizationID != "" {
		if o, ok := orgs.Lookup(g.OrganizationID); ok {
			return o.TitleFa
		}
		return g.OrganizationID
	}
	if g.HoldingFa != "" {
		if h, ok := orgs.HoldingByKey(g.HoldingFa); ok {
			return "هلدینگ " + h.TitleFa
		}
		return "هلدینگ " + g.HoldingFa
	}
	return "جهانی (همهٔ شرکت‌ها)"
}

// holdingsView فهرست هلدینگ‌ها و شرکت‌های زیرمجموعه برای انتخابگر دامنهٔ کنترل پنل.
func holdingsView() []map[string]any {
	out := []map[string]any{}
	for _, h := range orgs.Holdings() {
		companies := make([]map[string]string, 0, len(h.Companies))
		for _, c := range h.Companies {
			companies = append(companies, map[string]string{"id": c.ID, "titleFa": c.TitleFa})
		}
		out = append(out, map[string]any{"key": h.Key, "titleFa": h.TitleFa, "titleEn": h.TitleEn, "companies": companies})
	}
	return out
}
