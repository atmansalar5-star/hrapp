// آزمون جریان سرتاسری «tenancy»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"hrms/internal/platform/access"
)

type holdingView struct {
	Key       string `json:"key"`
	TitleFa   string `json:"titleFa"`
	Companies []struct {
		ID      string `json:"id"`
		TitleFa string `json:"titleFa"`
	} `json:"companies"`
}

type holdingsResponse struct {
	Holdings []holdingView `json:"holdings"`
	Total    int           `json:"total"`
}

type orgsResponse struct {
	IsAdmin       bool             `json:"isAdmin"`
	HoldingFa     string           `json:"holdingFa"`
	Organizations []map[string]any `json:"organizations"`
}

type accessResponse struct {
	Grants []access.Grant `json:"grants"`
	Scope  struct {
		Kind           string `json:"kind"`
		OrganizationID string `json:"organizationId"`
		HoldingFa      string `json:"holdingFa"`
	} `json:"scope"`
	Holdings []holdingView `json:"holdings"`
}

// TestTenantScopedOrgsAndAccess می‌سنجد که دامنهٔ شرکت/هلدینگ در دسترسی اعمال می‌شود:
// مدیر سیستم همه‌چیز را می‌بیند، بقیه فقط شرکت‌های هلدینگ خودشان را؛ و سطح دسترسی
// شرکت‌محور/هلدینگ‌محور فقط در همان دامنه در کنترل پنل دیده می‌شود.
func TestTenantScopedOrgsAndAccess(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo").AccessToken
	manager := login(t, h, "hr.manager", "demo").AccessToken

	// ---- هلدینگ‌ها: مدیر سیستم همه، بقیه فقط هلدینگ خودشان ----
	var allHoldings holdingsResponse
	rec := call(t, h, http.MethodGet, "/api/v1/orgs/holdings", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست هلدینگ‌ها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &allHoldings)
	if allHoldings.Total < 3 {
		t.Fatalf("مدیر سیستم باید دست‌کم سه هلدینگ ببیند، %d دید", allHoldings.Total)
	}
	var gulf holdingView
	for _, hv := range allHoldings.Holdings {
		if strings.Contains(hv.TitleFa, "خلیج فارس") {
			gulf = hv
		}
	}
	if gulf.Key == "" {
		t.Fatal("هلدینگ خلیج فارس در فهرست مدیر سیستم نیست")
	}
	if len(gulf.Companies) < 10 {
		t.Fatalf("هلدینگ خلیج فارس باید شرکت‌های زیرمجموعه را داشته باشد، %d شرکت", len(gulf.Companies))
	}

	var ownHoldings holdingsResponse
	rec = call(t, h, http.MethodGet, "/api/v1/orgs/holdings", manager, nil)
	decodeInto(t, rec, &ownHoldings)
	if ownHoldings.Total != 1 {
		t.Fatalf("کاربر غیرمدیر باید فقط یک هلدینگ ببیند، %d دید", ownHoldings.Total)
	}
	if !strings.Contains(ownHoldings.Holdings[0].TitleFa, "خلیج فارس") {
		t.Fatalf("هلدینگ کاربر غیرمدیر نادرست است: %s", ownHoldings.Holdings[0].TitleFa)
	}

	// ---- شرکت‌ها: مدیر همه، غیرمدیر فقط شرکت‌های همان هلدینگ ----
	var adminOrgs orgsResponse
	rec = call(t, h, http.MethodGet, "/api/v1/orgs", admin, nil)
	decodeInto(t, rec, &adminOrgs)
	if !adminOrgs.IsAdmin || len(adminOrgs.Organizations) != 15 {
		t.Fatalf("مدیر سیستم باید همهٔ ۱۵ شرکت را ببیند، admin=%v count=%d", adminOrgs.IsAdmin, len(adminOrgs.Organizations))
	}
	var mgrOrgs orgsResponse
	rec = call(t, h, http.MethodGet, "/api/v1/orgs", manager, nil)
	decodeInto(t, rec, &mgrOrgs)
	if mgrOrgs.IsAdmin {
		t.Fatal("مدیر منابع انسانی نباید مدیر سیستم شناخته شود")
	}
	if len(mgrOrgs.Organizations) >= len(adminOrgs.Organizations) {
		t.Fatalf("غیرمدیر باید شرکت‌های کمتری ببیند: %d در برابر %d", len(mgrOrgs.Organizations), len(adminOrgs.Organizations))
	}

	// ---- کنترل پنل دسترسی ویژهٔ مدیر سیستم است ----
	if rec := call(t, h, http.MethodGet, "/api/v1/settings/access", manager, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("دسترسی غیرمدیر به کنترل پنل باید ۴۰۳ بدهد، %d داد", rec.Code)
	}

	// ---- ساخت سطح دسترسی شرکت‌محور برای یک شرکت دیگر ----
	create := call(t, h, http.MethodPost, "/api/v1/settings/access/grants", admin, map[string]any{
		"role": "hr_spec", "moduleId": "reports", "actions": []string{"read"},
		"organizationId": "arvand", "noteFa": "دسترسی آزمایشی شرکت اروند",
	})
	if create.Code != http.StatusOK {
		t.Fatalf("ساخت سطح شرکت‌محور باید ۲۰۰ بدهد، %d داد: %s", create.Code, create.Body.String())
	}
	var grant access.Grant
	decodeInto(t, create, &grant)
	if grant.OrganizationID != "arvand" {
		t.Fatalf("دامنهٔ سطح باید arvand باشد: %+v", grant)
	}

	viewGrants := func(query string) accessResponse {
		t.Helper()
		rec := call(t, h, http.MethodGet, "/api/v1/settings/access"+query, admin, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("نمای کنترل پنل %q باید ۲۰۰ بدهد، %d داد", query, rec.Code)
		}
		var out accessResponse
		decodeInto(t, rec, &out)
		return out
	}
	hasGrant := func(list []access.Grant) bool {
		for _, g := range list {
			if g.ID == grant.ID {
				return true
			}
		}
		return false
	}

	if g := viewGrants("?org=arvand"); !hasGrant(g.Grants) || g.Scope.OrganizationID != "arvand" {
		t.Fatalf("سطح اروند باید در نمای همان شرکت دیده شود: %+v", g.Scope)
	}
	if g := viewGrants("?org=maroun"); hasGrant(g.Grants) {
		t.Fatal("سطح شرکت اروند نباید در نمای شرکت مارون دیده شود")
	}
	if g := viewGrants("?holding=" + url.QueryEscape("Persian Gulf Holding")); !hasGrant(g.Grants) || g.Scope.Kind != "holding" {
		t.Fatalf("سطح اروند باید در نمای هلدینگ خلیج فارس دیده شود: %+v", g.Scope)
	}
	// نمای پیش‌فرض روی شرکت فعال سرور است و نباید سطح شرکت اروند را نشان دهد.
	if g := viewGrants(""); hasGrant(g.Grants) {
		t.Fatal("سطح شرکت دیگر نباید در نمای پیش‌فرض (شرکت فعال) دیده شود")
	}
}
