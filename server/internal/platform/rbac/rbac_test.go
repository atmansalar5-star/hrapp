// آزمون زیرساخت «rbac» در پلتفرم سرور — «rbac_test».

package rbac

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/platform/audit"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/modules"
)

func newAuthorizer() (*Authorizer, *audit.Log) {
	registry := modules.NewRegistry(nil) // همهٔ ماژول‌ها فعال
	log := audit.New()
	return New(registry, log), log
}

func TestRegistryLoadsEveryManifestModule(t *testing.T) {
	registry := modules.NewRegistry(nil)
	if got, want := registry.Count(), len(modules.All); got != want {
		t.Fatalf("تعداد ماژول‌های فعال %d است، از مانیفست %d انتظار می‌رفت", got, want)
	}

	// هر ماژول باید به یک ادارهٔ موجود وصل باشد و هر اداره دست‌کم یک ماژول داشته باشد؛
	// همین دو شرط است که منوی اداره‌محور را معنا می‌دهد.
	byDepartment := map[string]int{}
	for _, m := range modules.All {
		if modules.DepartmentTitleFa(m.Department) == "" {
			t.Errorf("ماژول %s به ادارهٔ ناشناخته «%s» وصل است", m.ID, m.Department)
		}
		byDepartment[m.Department]++
	}
	for _, d := range modules.Departments {
		if byDepartment[d.ID] == 0 {
			t.Errorf("ادارهٔ «%s» هیچ ماژولی ندارد", d.ID)
		}
	}
}

func TestActionMatrixFollowsDesignDocument(t *testing.T) {
	a, _ := newAuthorizer()

	cases := []struct {
		role     contracts.Role
		module   string
		action   string
		expected bool
		why      string
	}{
		{contracts.RoleAdmin, "employee", "delete", true, "مدیر سیستم دسترسی کامل دارد"},
		{contracts.RoleHRManager, "employee", "delete", true, "مدیر منابع انسانی اجازهٔ حذف دارد"},
		{contracts.RoleHRSpecialist, "employee", "write", true, "کارشناس منابع انسانی مجاز به ورود داده است"},
		{contracts.RoleHRSpecialist, "employee", "delete", false, "کارشناس اجازهٔ حذف ندارد"},
		{contracts.RoleLineManager, "employee", "read", true, "مدیر واحد کارکنان واحد خود را می‌بیند"},
		{contracts.RoleLineManager, "employee", "write", false, "مدیر واحد اجازهٔ ویرایش پرونده ندارد"},
		{contracts.RoleEmployee, "employee", "read", true, "کارمند پروندهٔ خود را می‌بیند"},
		{contracts.RoleLineManager, "attendance", "approve", true, "تأیید مرخصی کار مدیر واحد است"},
		{contracts.RoleEmployee, "attendance", "approve", false, "کارمند اجازهٔ تأیید ندارد"},
		{contracts.RoleHRManager, "payroll", "write", true, "پردازش حقوق در اختیار مدیر منابع انسانی است"},
		{contracts.RoleEmployee, "payroll", "write", false, "کارمند فقط فیش خود را می‌بیند"},
		{contracts.RoleHRSpecialist, "settings", "read", false, "تنظیمات مخصوص مدیر سیستم است"},
		{contracts.RoleHRManager, "settings", "admin", true, "تنظیمات سازمانی برای مدیر منابع انسانی باز است"},
		{contracts.RoleEmployee, "recruitment", "read", false, "استخدام برای کارمند مخفی است"},
		{contracts.RoleAdmin, "no-such-module", "read", false, "ماژول ناشناخته دسترسی ندارد"},
	}

	for _, c := range cases {
		if got := a.Allows(c.role, c.module, c.action); got != c.expected {
			t.Errorf("%s / %s / %s = %v، انتظار %v بود (%s)", c.role, c.module, c.action, got, c.expected, c.why)
		}
	}
}

func TestScopeReflectsDataBoundaries(t *testing.T) {
	a, _ := newAuthorizer()

	cases := []struct {
		role, module, want string
	}{
		{"line_manager", "employee", "own_unit"},
		{"employee", "employee", "own"},
		{"admin", "employee", "all"},
		{"employee", "payroll", "own"},
		{"hr_manager", "recruitment", "all"},
	}
	for _, c := range cases {
		if got := a.Scope(contracts.Role(c.role), c.module); got != c.want {
			t.Errorf("دامنهٔ %s در %s = %q، انتظار %q", c.role, c.module, got, c.want)
		}
	}
}

func TestMenuHidesUnauthorisedModules(t *testing.T) {
	a, _ := newAuthorizer()
	registry := modules.NewRegistry(nil)

	employeeIDs := map[string]bool{}
	for _, m := range a.Menu(contracts.RoleEmployee) {
		employeeIDs[m.ModuleID] = true
	}
	for _, forbidden := range []string{"recruitment", "performance", "settings", "reports", "discipline", "orgstructure", "archive", "statistics", "staffing_moves", "outsourcing", "jobclass"} {
		if employeeIDs[forbidden] {
			t.Errorf("ماژول %s نباید در منوی کارمند باشد", forbidden)
		}
	}
	for _, allowed := range []string{"dashboard", "employee", "attendance", "payroll", "training", "welfare", "career", "socialwork", "medical", "loans_retirement", "grievances", "hrstudies"} {
		if !employeeIDs[allowed] {
			t.Errorf("ماژول %s باید در منوی کارمند باشد", allowed)
		}
	}

	adminMenu := a.Menu(contracts.RoleAdmin)
	seen := map[string]int{}
	for _, entry := range adminMenu {
		seen[entry.ModuleID]++
	}
	if len(seen) != len(modules.All) {
		t.Errorf("منوی مدیر سیستم باید همهٔ %d ماژول را داشته باشد، %d ماژول یکتا داشت", len(modules.All), len(seen))
	}

	// ماژول مشترک (مانند طبقه‌بندی مشاغل که بین امور کارکنان و برون‌سپاری مشترک است)
	// باید یک‌بار در خانهٔ خودش و یک‌بار برای هر ادارهٔ مشترک بیاید، وگرنه کاربر برای
	// رسیدن به آن باید از منوی ادارهٔ دیگری رد شود.
	for _, m := range modules.All {
		want := 1 + len(m.SharedWith)
		if seen[m.ID] != want {
			t.Errorf("ماژول %s باید %d بار در منوی مدیر سیستم بیاید، %d بار آمد", m.ID, want, seen[m.ID])
			continue
		}
		homes, shareds := 0, 0
		for _, entry := range adminMenu {
			if entry.ModuleID != m.ID {
				continue
			}
			if entry.Shared {
				shareds++
			} else {
				homes++
			}
		}
		if homes != 1 || shareds != len(m.SharedWith) {
			t.Errorf("ماژول %s: خانه=%d (انتظار ۱)، مشترک=%d (انتظار %d)", m.ID, homes, shareds, len(m.SharedWith))
		}
	}

	// هر آیتم منو باید اداره، حوزه و زیرواحد خود را هم بگوید تا کلاینت بتواند بدون
	// نگاشت دستی، منو را به شکل «حوزه > اداره > زیرواحد > ماژول» بچیند.
	for _, entry := range adminMenu {
		if entry.Department == "" || entry.DepartmentTitleFa == "" {
			t.Errorf("آیتم منوی %s اداره را اعلام نکرده است", entry.ModuleID)
		}
		if entry.Directorate == "" || modules.DirectorateTitleFa(entry.Directorate) == "" {
			t.Errorf("آیتم منوی %s حوزهٔ سازمانی را اعلام نکرده است", entry.ModuleID)
		}
		m, ok := registry.ByID(entry.ModuleID)
		if !ok {
			t.Errorf("ماژول %s در رجیستری نیست", entry.ModuleID)
			continue
		}
		if m.SubUnit != "" && (entry.SubUnit != m.SubUnit || entry.SubUnitTitleFa == "") {
			t.Errorf("آیتم منوی %s زیرواحد «%s» را درست اعلام نکرده است", entry.ModuleID, m.SubUnit)
		}
	}

	// ترتیب منو باید اداره‌به‌اداره باشد، نه درهم.
	lastRank := 0
	for _, entry := range adminMenu {
		rank := 0
		for _, d := range modules.Departments {
			if d.ID == entry.Department {
				rank = d.Order
			}
		}
		if rank < lastRank {
			t.Errorf("ترتیب منو درهم است: %s پس از ادارهٔ رتبهٔ %d آمد", entry.ModuleID, lastRank)
		}
		lastRank = rank
	}
}

func TestRequireBlocksForbiddenAndLogsAudit(t *testing.T) {
	a, auditLog := newAuthorizer()

	handler := a.Require("settings", "read", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	service := auth.NewService([]byte("test-secret"), nil)

	// کارشناس منابع انسانی نباید به تنظیمات برسد.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings/users", nil)
	ctx := auth.WithPrincipal(req.Context(), auth.Principal{
		User: contracts.User{Username: "hr.spec", DisplayName: "مریم نوری", Role: contracts.RoleHRSpecialist},
	})
	rec := httptest.NewRecorder()
	handler(rec, req.WithContext(ctx))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("کد پاسخ %d بود، انتظار ۴۰۳ می‌رفت", rec.Code)
	}
	var body contracts.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("بدنهٔ پاسخ JSON معتبر نیست: %v", err)
	}
	if body.Code != contracts.CodeForbidden || body.Message == "" {
		t.Errorf("پاسخ باید کد forbidden و پیام فارسی داشته باشد، دریافت شد %+v", body)
	}
	if auditLog.Len() != 1 {
		t.Fatalf("رخداد رد دسترسی باید در لاگ حسابرسی ثبت شود، تعداد=%d", auditLog.Len())
	}
	if err := auditLog.Verify(); err != nil {
		t.Errorf("زنجیرهٔ لاگ حسابرسی معتبر نیست: %v", err)
	}

	// و مدیر سیستم باید بگذرد.
	reqAdmin := httptest.NewRequest(http.MethodGet, "/api/v1/settings/users", nil)
	ctxAdmin := auth.WithPrincipal(reqAdmin.Context(), auth.Principal{
		User: contracts.User{Username: "admin", Role: contracts.RoleAdmin},
	})
	recAdmin := httptest.NewRecorder()
	handler(recAdmin, reqAdmin.WithContext(ctxAdmin))
	if recAdmin.Code != http.StatusOK {
		t.Errorf("مدیر سیستم باید دسترسی داشته باشد، کد=%d", recAdmin.Code)
	}
	_ = service
}

func TestRequireAuthRejectsAnonymous(t *testing.T) {
	a, _ := newAuthorizer()
	handler := a.Require("employee", "read", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/api/v1/employee/employees", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("درخواست بی‌نشست باید ۴۰۱ بگیرد، کد=%d", rec.Code)
	}
}

func TestDisabledModuleIsDenied(t *testing.T) {
	// استقراری که فقط ماژول کارکنان در آن فعال است (پرچم ویژگی).
	registry := modules.NewRegistry(map[string]bool{"employee": true})
	a := New(registry, audit.New())

	if !a.Allows(contracts.RoleHRManager, "employee", "read") {
		t.Error("ماژول فعال باید دسترسی بدهد")
	}
	if a.Allows(contracts.RoleHRManager, "payroll", "read") {
		t.Error("ماژول غیرفعال در این استقرار نباید دسترسی بدهد")
	}
	if len(a.Menu(contracts.RoleAdmin)) != 1 {
		t.Error("منو باید فقط ماژول‌های فعال را نشان دهد")
	}
}

func TestAuthServiceRoundTrip(t *testing.T) {
	accounts, err := auth.SeedAccounts()
	if err != nil {
		t.Fatalf("ساخت حساب‌های نمونه ناموفق: %v", err)
	}
	service := auth.NewService([]byte("secret-for-test"), accounts)

	session, err := service.Login("line.mgr", "demo")
	if err != nil {
		t.Fatalf("ورود حساب نمونه ناموفق: %v", err)
	}
	if session.User.Role != contracts.RoleLineManager {
		t.Errorf("نقش بازگشتی %q است", session.User.Role)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Error("توکن‌های نشست باید صادر شوند")
	}

	claims, err := service.VerifyAccess(session.AccessToken)
	if err != nil {
		t.Fatalf("توکن دسترسی معتبر نیست: %v", err)
	}
	if claims.EmployeeID != "EMP-002" || claims.DepartmentID != "D1" {
		t.Errorf("محدودهٔ دادهٔ کاربر در توکن نادرست است: %+v", claims)
	}

	if _, err := service.Login("line.mgr", "wrong"); err == nil {
		t.Error("رمز نادرست باید رد شود")
	}
	if _, err := service.Login("ghost", "demo"); err == nil {
		t.Error("کاربر ناموجود باید رد شود")
	}

	refreshed, err := service.Refresh(session.RefreshToken)
	if err != nil {
		t.Fatalf("تمدید نشست ناموفق: %v", err)
	}
	if refreshed.User.Username != "line.mgr" {
		t.Error("نشست تازه باید همان کاربر را داشته باشد")
	}
	if _, err := service.Refresh(session.AccessToken); err == nil {
		t.Error("توکن دسترسی نباید برای تمدید پذیرفته شود")
	}
}
