// آزمون جریان سرتاسری «integration_test»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hrms/internal/contracts"
)

// newTestServer سرور کامل (همهٔ ماژول‌ها) روی حافظه می‌سازد.
//
// پوشهٔ دادهٔ سرور به یک پوشهٔ موقت آزمون تنظیم می‌شود: بارگذاری فایل در آزمون‌ها نباید
// در درخت مخزن (مثل server/internal/bootstrap/data) رد پا بگذارد.
func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	deps, err := NewDeps(Config{
		JWTSecret: []byte("test-secret-key-for-integration"),
		DataDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("ساخت وابستگی‌ها ناموفق: %v", err)
	}
	handler, registered, err := NewHandler(deps)
	if err != nil {
		t.Fatalf("ساخت هندلر ناموفق: %v", err)
	}
	if want := len(Modules()); len(registered) != want {
		t.Fatalf("همهٔ %d ماژول باید ثبت شوند، %d ثبت شد", want, len(registered))
	}
	return handler
}

func login(t *testing.T, h http.Handler, username, password string) contracts.Session {
	t.Helper()
	rec := call(t, h, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"username": username, "password": password})
	if rec.Code != http.StatusOK {
		t.Fatalf("ورود %s ناموفق بود (کد %d): %s", username, rec.Code, rec.Body.String())
	}
	var session contracts.Session
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("پاسخ ورود قابل تجزیه نیست: %v", err)
	}
	return session
}

func call(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("سریال‌سازی بدنه ناموفق: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodePage(t *testing.T, rec *httptest.ResponseRecorder) contracts.EmployeePage {
	t.Helper()
	var page contracts.EmployeePage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("پاسخ فهرست کارکنان قابل تجزیه نیست: %v (%s)", err, rec.Body.String())
	}
	return page
}

func TestHealthEndpointReportsModuleCount(t *testing.T) {
	h := newTestServer(t)
	rec := call(t, h, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("بررسی سلامت باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["status"] != "ok" || int(body["modules"].(float64)) != len(Modules()) {
		t.Errorf("پاسخ سلامت نادرست است: %s", rec.Body.String())
	}
}

func TestLoginRejectsWrongPasswordAndUnknownUser(t *testing.T) {
	h := newTestServer(t)

	for _, c := range []struct{ user, pass string }{{"hr.manager", "wrong"}, {"ghost", "demo"}} {
		rec := call(t, h, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"username": c.user, "password": c.pass})
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("ورود %s باید ۴۰۱ بدهد، %d داد", c.user, rec.Code)
		}
		var errBody contracts.Error
		_ = json.Unmarshal(rec.Body.Bytes(), &errBody)
		if errBody.Message == "" {
			t.Errorf("خطای ورود باید پیام فارسی داشته باشد")
		}
	}
}

func TestProtectedRoutesRequireSession(t *testing.T) {
	h := newTestServer(t)
	for _, path := range []string{"/api/v1/employee/employees", "/api/v1/auth/me", "/api/v1/sync/changes"} {
		rec := call(t, h, http.MethodGet, path, "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s بدون نشست باید ۴۰۱ بدهد، %d داد", path, rec.Code)
		}
	}
}

func TestMeReturnsRoleScopedMenuAndScopes(t *testing.T) {
	h := newTestServer(t)
	session := login(t, h, "line.mgr", "demo")

	rec := call(t, h, http.MethodGet, "/api/v1/auth/me", session.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("دریافت کاربر جاری ناموفق بود: %d", rec.Code)
	}
	var body struct {
		User contracts.User `json:"user"`
		Menu []struct {
			ModuleID string `json:"moduleId"`
			Screens  []struct {
				Key string `json:"key"`
			} `json:"screens"`
		} `json:"menu"`
		Scopes map[string]string `json:"scopes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("پاسخ /me قابل تجزیه نیست: %v", err)
	}
	if body.User.Role != contracts.RoleLineManager {
		t.Errorf("نقش کاربر نادرست است: %s", body.User.Role)
	}
	if body.Scopes["employee"] != "own_unit" {
		t.Errorf("دامنهٔ کارکنان برای مدیر واحد باید own_unit باشد: %s", body.Scopes["employee"])
	}

	// تنظیمات نباید در منوی مدیر واحد باشد.
	for _, m := range body.Menu {
		if m.ModuleID == "settings" {
			t.Error("ماژول تنظیمات نباید در منوی مدیر واحد باشد")
		}
	}
	// صفحهٔ تأیید مرخصی باید باشد.
	found := false
	for _, m := range body.Menu {
		if m.ModuleID != "attendance" {
			continue
		}
		for _, s := range m.Screens {
			if s.Key == "leaveApprove" {
				found = true
			}
		}
	}
	if !found {
		t.Error("صفحهٔ تأیید مرخصی باید در منوی مدیر واحد باشد")
	}
}

func TestScopeFiltersListAtQueryLevel(t *testing.T) {
	h := newTestServer(t)

	hr := login(t, h, "hr.manager", "demo")
	page := decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?size=100", hr.AccessToken, nil))
	if page.Total != 12 {
		t.Errorf("مدیر منابع انسانی باید ۱۲ کارمند ببیند، %d دید", page.Total)
	}

	// مدیر واحد D1 فقط کارکنان واحد خودش را می‌بیند.
	lm := login(t, h, "line.mgr", "demo")
	pageLM := decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?size=100", lm.AccessToken, nil))
	if pageLM.Total != 3 {
		t.Errorf("مدیر واحد D1 باید ۳ کارمند ببیند، %d دید: %+v", pageLM.Total, pageLM.Items)
	}
	for _, e := range pageLM.Items {
		if e.DepartmentID != "D1" {
			t.Errorf("کارمند %s از واحد %s در فهرست مدیر واحد D1 نباید باشد", e.ID, e.DepartmentID)
		}
	}

	// کارمند فقط پروندهٔ خودش را می‌بیند.
	emp := login(t, h, "employee", "demo")
	pageEmp := decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?size=100", emp.AccessToken, nil))
	if pageEmp.Total != 1 || pageEmp.Items[0].ID != "EMP-010" {
		t.Errorf("کارمند باید فقط پروندهٔ خودش را ببیند: %+v", pageEmp.Items)
	}
}

func TestScopeAlsoGuardsDirectRecordAccess(t *testing.T) {
	h := newTestServer(t)
	lm := login(t, h, "line.mgr", "demo") // مدیر واحد D1

	// پروندهٔ کارمند همان واحد: مجاز
	ok := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-010", lm.AccessToken, nil)
	if ok.Code != http.StatusOK {
		t.Errorf("دسترسی به کارمند واحد خود باید مجاز باشد، کد=%d", ok.Code)
	}
	// پروندهٔ کارمند واحد دیگر: باید «یافت نشد» بگیرد (نه ۴۰۳، تا وجود رکورد لو نرود)
	denied := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-003", lm.AccessToken, nil)
	if denied.Code != http.StatusNotFound {
		t.Errorf("کارمند واحد دیگر باید ۴۰۴ بدهد، %d داد", denied.Code)
	}
}

func TestDeletePermissionFollowsRole(t *testing.T) {
	h := newTestServer(t)

	spec := login(t, h, "hr.spec", "demo")
	forbidden := call(t, h, http.MethodDelete, "/api/v1/employee/employees/EMP-012", spec.AccessToken, nil)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("کارشناس منابع انسانی نباید حذف کند، کد=%d", forbidden.Code)
	}
	var errBody contracts.Error
	_ = json.Unmarshal(forbidden.Body.Bytes(), &errBody)
	if errBody.Code != contracts.CodeForbidden || errBody.Message == "" {
		t.Errorf("پاسخ ۴۰۳ باید کد و پیام فارسی داشته باشد: %+v", errBody)
	}

	manager := login(t, h, "hr.manager", "demo")
	ok := call(t, h, http.MethodDelete, "/api/v1/employee/employees/EMP-012", manager.AccessToken, nil)
	if ok.Code != http.StatusNoContent {
		t.Fatalf("مدیر منابع انسانی باید حذف کند، کد=%d (%s)", ok.Code, ok.Body.String())
	}

	page := decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?size=100", manager.AccessToken, nil))
	if page.Total != 11 {
		t.Errorf("پس از حذف باید ۱۱ کارمند بماند، %d ماند", page.Total)
	}
}

func TestCreateValidatesInputWithPersianFieldErrors(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo")

	// کد ملی نامعتبر و تاریخ نامعتبر
	bad := map[string]any{
		"fullName": "ا", "nationalId": "1111111111", "departmentId": "", "position": "",
		"contractType": "تمام‌وقت", "hireDate": "1404/13/01",
	}
	rec := call(t, h, http.MethodPost, "/api/v1/employee/employees", hr.AccessToken, bad)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ورودی نامعتبر باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var errBody contracts.Error
	_ = json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody.Code != contracts.CodeValidationFailed {
		t.Errorf("کد خطا باید validation_failed باشد: %s", errBody.Code)
	}
	for _, field := range []string{"fullName", "nationalId", "departmentId", "position", "hireDate"} {
		if _, ok := errBody.Details[field]; !ok {
			t.Errorf("خطای فیلد %s باید برگردد تا کنار همان فیلد نمایش داده شود: %+v", field, errBody.Details)
		}
	}
	if !containsPersian(errBody.Details["nationalId"]) {
		t.Errorf("پیام خطای کد ملی باید فارسی باشد: %q", errBody.Details["nationalId"])
	}
}

func TestCreateUpdateAndReadProfile(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo")

	valid := map[string]any{
		"fullName": "کیانا شجاعی", "nationalId": "0112345670", "departmentId": "D6",
		"position": "کارشناس تحلیل داده", "contractType": "تمام‌وقت", "hireDate": "1404/07/01",
		"baseSalary": 32000000, "phone": "0912-0000099", "email": "k.shajaei@demo.ir",
	}
	rec := call(t, h, http.MethodPost, "/api/v1/employee/employees", hr.AccessToken, valid)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ایجاد کارمند باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var created contracts.Employee
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.DepartmentName != "تحقیق و توسعه" {
		t.Errorf("نام واحد باید از ساختار سازمانی پر شود: %q", created.DepartmentName)
	}
	if created.Version != 1 || created.Status != "active" {
		t.Errorf("رکورد تازه باید نسخهٔ ۱ و وضعیت active داشته باشد: %+v", created)
	}

	// پرونده پس از ایجاد خوانده می‌شود.
	got := call(t, h, http.MethodGet, "/api/v1/employee/employees/"+created.ID, hr.AccessToken, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("خواندن پروندهٔ تازه ناموفق: %d", got.Code)
	}

	// ویرایش پست و بررسی افزایش نسخه
	updated := call(t, h, http.MethodPatch, "/api/v1/employee/employees/"+created.ID, hr.AccessToken, map[string]any{
		"fullName": "کیانا شجاعی", "nationalId": "0112345670", "departmentId": "D6",
		"position": "سرپرست تحلیل داده", "contractType": "تمام‌وقت", "hireDate": "1404/07/01",
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("ویرایش ناموفق: %d (%s)", updated.Code, updated.Body.String())
	}
	var after contracts.Employee
	_ = json.Unmarshal(updated.Body.Bytes(), &after)
	if after.Position != "سرپرست تحلیل داده" || after.Version != 2 {
		t.Errorf("ویرایش اعمال نشد یا نسخه بالا نرفت: %+v", after)
	}
}

func TestSearchFilterSortAndPaging(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo")

	// جستجو
	page := decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?query=نگار", hr.AccessToken, nil))
	if page.Total != 1 || page.Items[0].FullName != "نگار حسینی" {
		t.Errorf("جستجوی فارسی ناموفق بود: %+v", page.Items)
	}

	// فیلتر واحد
	page = decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?departmentId=D5", hr.AccessToken, nil))
	if page.Total != 2 {
		t.Errorf("فیلتر واحد D5 باید ۲ کارمند بدهد، %d داد", page.Total)
	}

	// فیلتر وضعیت
	page = decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?status=probation", hr.AccessToken, nil))
	if page.Total != 1 || page.Items[0].ID != "EMP-011" {
		t.Errorf("فیلتر وضعیت آزمایشی نادرست است: %+v", page.Items)
	}

	// فیلتر نوع قرارداد
	page = decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?contractType=پیمانکاری", hr.AccessToken, nil))
	if page.Total != 1 || page.Items[0].ID != "EMP-010" {
		t.Errorf("فیلتر نوع قرارداد نادرست است: %+v", page.Items)
	}

	// مرتب‌سازی نزولی بر حقوق
	page = decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?sort=baseSalary:desc&size=3", hr.AccessToken, nil))
	if len(page.Items) != 3 || page.Items[0].ID != "EMP-002" {
		t.Errorf("مرتب‌سازی بر حقوق نادرست است: %+v", page.Items)
	}

	// صفحه‌بندی: شمارش کل باید مستقل از اندازهٔ صفحه بماند
	first := decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?page=1&size=5", hr.AccessToken, nil))
	second := decodePage(t, call(t, h, http.MethodGet, "/api/v1/employee/employees?page=2&size=5", hr.AccessToken, nil))
	if first.Total != 12 || second.Total != 12 || len(first.Items) != 5 || len(second.Items) != 5 {
		t.Errorf("صفحه‌بندی نادرست است: page1=%d page2=%d", len(first.Items), len(second.Items))
	}
	if first.Items[0].ID == second.Items[0].ID {
		t.Error("صفحهٔ دوم نباید رکورد تکراری صفحهٔ اول را بیاورد")
	}
}

func TestDepartmentsEndpointFeedsOrgChart(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo")

	rec := call(t, h, http.MethodGet, "/api/v1/employee/departments", hr.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("دریافت ساختار سازمانی ناموفق: %d", rec.Code)
	}
	var departments []contracts.Department
	if err := json.Unmarshal(rec.Body.Bytes(), &departments); err != nil {
		t.Fatalf("پاسخ ساختار سازمانی قابل تجزیه نیست: %v", err)
	}
	if len(departments) != 6 {
		t.Errorf("ساختار سازمانی باید ۶ واحد داشته باشد، %d داشت", len(departments))
	}
	hasChild := false
	for _, d := range departments {
		if d.ParentID == "D1" {
			hasChild = true
		}
	}
	if !hasChild {
		t.Error("واحد تحقیق و توسعه باید فرزند واحد فناوری اطلاعات باشد تا نمودار درختی معنی داشته باشد")
	}
}

func TestDashboardWidgetsUseEmployeePort(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo")

	rec := call(t, h, http.MethodGet, "/api/v1/dashboard/widgets", hr.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("دریافت کارت‌های داشبورد ناموفق: %d", rec.Code)
	}
	var widgets []contracts.Widget
	if err := json.Unmarshal(rec.Body.Bytes(), &widgets); err != nil {
		t.Fatalf("پاسخ داشبورد قابل تجزیه نیست: %v", err)
	}
	var headcount float64
	for _, wgt := range widgets {
		if wgt.Key == "headcount" {
			headcount = wgt.Value
		}
	}
	if headcount != 12 {
		t.Errorf("کارت «کل کارکنان» باید ۱۲ باشد، %v بود (%+v)", headcount, widgets)
	}

	// مدیر واحد فقط آمار واحد خود را می‌بیند.
	lm := login(t, h, "line.mgr", "demo")
	recLM := call(t, h, http.MethodGet, "/api/v1/dashboard/widgets", lm.AccessToken, nil)
	var widgetsLM []contracts.Widget
	_ = json.Unmarshal(recLM.Body.Bytes(), &widgetsLM)
	for _, wgt := range widgetsLM {
		if wgt.Key == "headcount" && wgt.Value != 3 {
			t.Errorf("مدیر واحد D1 باید ۳ نفر ببیند، %v دید", wgt.Value)
		}
	}
}

func TestAuditEndpointIsAdminOnlyAndChainIsValid(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo")

	// چند رخداد بسازیم: ورود بالا و یک درخواست ردشده.
	spec := login(t, h, "hr.spec", "demo")
	_ = call(t, h, http.MethodDelete, "/api/v1/employee/employees/EMP-012", spec.AccessToken, nil)

	rec := call(t, h, http.MethodGet, "/api/v1/settings/audit", admin.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("مدیر سیستم باید لاگ حسابرسی را ببیند، کد=%d", rec.Code)
	}
	var body struct {
		Events     []map[string]any `json:"events"`
		ChainValid bool             `json:"chainValid"`
		Total      int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("پاسخ لاگ حسابرسی قابل تجزیه نیست: %v", err)
	}
	if !body.ChainValid {
		t.Error("زنجیرهٔ لاگ حسابرسی باید معتبر باشد")
	}
	if body.Total < 3 || len(body.Events) != body.Total {
		t.Errorf("رخدادها باید ثبت شده باشند: %+v", body)
	}

	// مدیر منابع انسانی به لاگ حسابرسی دسترسی ندارد (صفحهٔ audit مخصوص admin است).
	hr := login(t, h, "hr.manager", "demo")
	denied := call(t, h, http.MethodGet, "/api/v1/settings/audit", hr.AccessToken, nil)
	if denied.Code != http.StatusForbidden {
		t.Errorf("مدیر منابع انسانی نباید لاگ حسابرسی ببیند، کد=%d", denied.Code)
	}
}

func TestSyncDeltaAndIdempotentMutations(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo")

	// یک تغییر روی سرور بسازیم تا در دلتا بیاید.
	created := call(t, h, http.MethodPost, "/api/v1/employee/employees", hr.AccessToken, map[string]any{
		"fullName": "شیرین یزدانی", "nationalId": "0123456789", "departmentId": "D6",
		"position": "کارشناس", "contractType": "تمام‌وقت", "hireDate": "1404/07/02",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("ایجاد رکورد برای سینک ناموفق: %d (%s)", created.Code, created.Body.String())
	}
	var employee contracts.Employee
	_ = json.Unmarshal(created.Body.Bytes(), &employee)

	changes := call(t, h, http.MethodGet, "/api/v1/sync/changes?since=0", hr.AccessToken, nil)
	if changes.Code != http.StatusOK {
		t.Fatalf("دریافت دلتا ناموفق: %d", changes.Code)
	}
	var changeSet contracts.ChangeSet
	if err := json.Unmarshal(changes.Body.Bytes(), &changeSet); err != nil {
		t.Fatalf("دلتا قابل تجزیه نیست: %v", err)
	}
	if len(changeSet.Changes) == 0 || changeSet.NextSeq < 1 {
		t.Fatalf("دلتا باید دست‌کم یک تغییر داشته باشد: %+v", changeSet)
	}

	// عملیات آفلاین: ویرایش با همان opId دو بار فرستاده می‌شود.
	mutations := []contracts.Mutation{{
		OpID:        "op-emp-" + employee.ID,
		Entity:      "Employee",
		Op:          "update",
		ID:          employee.ID,
		BaseVersion: employee.Version,
		Payload:     map[string]any{"position": "سرپرست"},
	}}
	first := call(t, h, http.MethodPost, "/api/v1/sync/mutations", hr.AccessToken, mutations)
	if first.Code != http.StatusOK {
		t.Fatalf("اعمال عملیات ناموفق: %d (%s)", first.Code, first.Body.String())
	}
	var result contracts.MutationResult
	_ = json.Unmarshal(first.Body.Bytes(), &result)
	if len(result.Accepted) != 1 || len(result.Rejected) != 0 {
		t.Fatalf("عملیات معتبر باید پذیرفته شود: %+v", result)
	}

	after := call(t, h, http.MethodGet, "/api/v1/sync/changes?since="+fmt.Sprint(changeSet.NextSeq), hr.AccessToken, nil)
	var delta contracts.ChangeSet
	_ = json.Unmarshal(after.Body.Bytes(), &delta)
	beforeCount := len(delta.Changes)

	second := call(t, h, http.MethodPost, "/api/v1/sync/mutations", hr.AccessToken, mutations)
	var repeat contracts.MutationResult
	_ = json.Unmarshal(second.Body.Bytes(), &repeat)
	if len(repeat.Accepted) != 1 || len(repeat.Rejected) != 0 {
		t.Errorf("تکرار همان opId باید بی‌اثر و پذیرفته باشد: %+v", repeat)
	}

	again := call(t, h, http.MethodGet, "/api/v1/sync/changes?since="+fmt.Sprint(delta.NextSeq), hr.AccessToken, nil)
	var againDelta contracts.ChangeSet
	_ = json.Unmarshal(again.Body.Bytes(), &againDelta)
	if len(againDelta.Changes) != 0 {
		t.Errorf("تکرار عملیات نباید تغییر تازه بسازد (پیش از تکرار %d، پس از آن %d)", beforeCount, len(againDelta.Changes))
	}
}

func TestUnimplementedModulePathAndAccessBoundary(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo")

	// همهٔ ماژول‌های اداری اکنون پیاده‌سازی شده‌اند؛ مسیرهای قدیمیِ نمایشی نباید پاسخ
	// بدهند و باید ۴۰۴ خام بگیرند.
	gone := call(t, h, http.MethodGet, "/api/v1/payroll/runs", hr.AccessToken, nil)
	if gone.Code != http.StatusNotFound {
		t.Errorf("مسیر قدیمی ماژول نباید پاسخ دهد، کد=%d", gone.Code)
	}

	// قالب قرارداد اکنون واقعی است: متن قرارداد قابل طراحی است و از ماژول تنظیمات
	// (پورت قالب‌های اسناد) خوانده می‌شود؛ پس باید ۲۰۰ با فهرست قالب‌ها بدهد.
	rec := call(t, h, http.MethodGet, "/api/v1/contracts/templates", hr.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("مسیر قالب قرارداد باید ۲۰۰ بدهد، %d داد (%s)", rec.Code, rec.Body.String())
	}
	var tplBody struct {
		Templates []contracts.DocumentTemplate `json:"templates"`
		Total     int                          `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tplBody); err != nil {
		t.Fatalf("بدنهٔ قالب‌های قرارداد خوانده نشد: %v", err)
	}
	for _, tt := range tplBody.Templates {
		if tt.DocType != "contract" {
			t.Errorf("مسیر قرارداد باید فقط قالب نوع contract بدهد: %+v", tt)
		}
	}

	// مرز دسترسی: کارمند به رکوردهای کمیتهٔ انضباط کار دسترسی ندارد.
	emp := login(t, h, "employee", "demo")
	forbidden := call(t, h, http.MethodGet, "/api/v1/discipline/records", emp.AccessToken, nil)
	if forbidden.Code != http.StatusForbidden {
		t.Errorf("کارمند باید ۴۰۳ بگیرد، %d گرفت", forbidden.Code)
	}
}

func TestFeatureFlagDisablesModuleAtDeployment(t *testing.T) {
	deps, err := NewDeps(Config{Modules: "employee,dashboard,settings", JWTSecret: []byte("test-secret-key-for-flags")})
	if err != nil {
		t.Fatalf("ساخت وابستگی‌ها ناموفق: %v", err)
	}
	handler, registered, err := NewHandler(deps)
	if err != nil {
		t.Fatalf("ساخت هندلر ناموفق: %v", err)
	}
	if len(registered) != 3 {
		t.Fatalf("فقط ۳ ماژول باید فعال باشند، %d فعال شد", len(registered))
	}

	hr := login(t, handler, "hr.manager", "demo")
	// ماژول غیرفعال در این استقرار حتی مسیرش ثبت نشده است.
	rec := call(t, handler, http.MethodGet, "/api/v1/payroll/runs", hr.AccessToken, nil)
	if rec.Code == http.StatusOK {
		t.Fatalf("ماژول غیرفعال نباید پاسخ موفق بدهد، کد=%d", rec.Code)
	}
	// ماژول فعال باید کار کند.
	ok := call(t, handler, http.MethodGet, "/api/v1/employee/employees", hr.AccessToken, nil)
	if ok.Code != http.StatusOK {
		t.Errorf("ماژول فعال باید کار کند، کد=%d", ok.Code)
	}
}

func containsPersian(s string) bool {
	for _, r := range s {
		if r >= '\u0600' && r <= '\u06FF' {
			return true
		}
	}
	return false
}

// payslipResult همان بخش نتیجهٔ فیش که این آزمون دربارهٔ آن است.
type payslipResult struct {
	VeteranApplied       bool   `json:"veteranApplied"`
	VeteranGroupFa       string `json:"veteranGroupFa"`
	VeteranTaxExempt     int64  `json:"veteranTaxExempt"`
	VeteranPensionExempt int64  `json:"veteranPensionExempt"`
	Tax                  int64  `json:"tax"`
	Net                  int64  `json:"net"`
}

// TestVeteranEntitlementReachesPayslip سرتاسری: پروندهٔ ایثارگری «تأییدشدهٔ» کارمند
// (دادهٔ نمونهٔ EMP-009، جانباز ۲۵٪) باید از ماژول امور ایثارگران تا فیش حقوقی همین
// کارمند اثر بگذارد و کارمند بدون پرونده هیچ امتیازی نگیرد.
func TestVeteranEntitlementReachesPayslip(t *testing.T) {
	h := newTestServer(t)
	session := login(t, h, "hr.manager", "demo")

	payslip := func(employeeID string) payslipResult {
		rec := call(t, h, http.MethodGet, "/api/v1/payroll/payslip?employee="+employeeID, session.AccessToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("فیش کارمند %s خوانده نشد (کد %d): %s", employeeID, rec.Code, rec.Body.String())
		}
		var body struct {
			Result payslipResult `json:"result"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("پاسخ فیش قابل تجزیه نیست: %v", err)
		}
		return body.Result
	}

	veteran := payslip("EMP-009")
	if !veteran.VeteranApplied || veteran.VeteranGroupFa != "جانباز" {
		t.Fatalf("فیش کارمند جانباز باید امتیاز ایثارگری را نشان دهد: %+v", veteran)
	}
	// معافیت ماده ۵۶ صددرصد است، پس مالیات این فیش صفر میشود.
	if veteran.Tax != 0 || veteran.VeteranTaxExempt <= 0 {
		t.Fatalf("معافیت مالیاتی ایثارگری اعمال نشده است: %+v", veteran)
	}

	plain := payslip("EMP-001")
	if plain.VeteranApplied || plain.VeteranTaxExempt != 0 {
		t.Fatalf("کارمند بدون پروندهٔ ایثارگری نباید امتیاز بگیرد: %+v", plain)
	}
}

// TestVeteranEntitlementReachesOilPayslip سرتاسری فیش صندوق نفت: کد کارمند در ورودی،
// پروندهٔ تأییدشدهٔ همان کارمند در خروجی (EMP-005 آزادهٔ نمونه).
func TestVeteranEntitlementReachesOilPayslip(t *testing.T) {
	h := newTestServer(t)
	session := login(t, h, "hr.manager", "demo")

	rec := call(t, h, http.MethodPost, "/api/v1/payroll/oil-payslip", session.AccessToken,
		map[string]any{"employeeId": "EMP-005", "baseSalary": 120_000_000})
	if rec.Code != http.StatusOK {
		t.Fatalf("محاسبهٔ فیش صندوق نفت ناموفق بود (کد %d): %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Slip struct {
			VeteranApplied   bool   `json:"veteranApplied"`
			VeteranGroupFa   string `json:"veteranGroupFa"`
			VeteranTaxExempt int64  `json:"veteranTaxExempt"`
			TaxableIncome    int64  `json:"taxableIncome"`
			Tax              int64  `json:"tax"`
		} `json:"slip"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("پاسخ فیش صندوق نفت قابل تجزیه نیست: %v", err)
	}
	if !body.Slip.VeteranApplied || body.Slip.VeteranGroupFa != "آزاده" {
		t.Fatalf("فیش صندوق نفت باید امتیاز ایثارگری آزاده را نشان دهد: %+v", body.Slip)
	}
	if body.Slip.TaxableIncome != 0 || body.Slip.Tax != 0 {
		t.Fatalf("با معافیت صددرصد، درآمد مشمول و مالیات باید صفر باشد: %+v", body.Slip)
	}
}

// TestVeteranEntitlementWithoutFileOilPayslip کارمند بدون پرونده در فیش صندوق نفت هم
// هیچ امتیازی نمی‌گیرد (پروندهٔ کارمند دیگر به این کارمند سرایت نمی‌کند).
func TestVeteranEntitlementWithoutFileOilPayslip(t *testing.T) {
	h := newTestServer(t)
	session := login(t, h, "hr.manager", "demo")

	rec := call(t, h, http.MethodPost, "/api/v1/payroll/oil-payslip", session.AccessToken,
		map[string]any{"employeeId": "EMP-001", "baseSalary": 120_000_000})
	if rec.Code != http.StatusOK {
		t.Fatalf("محاسبهٔ فیش صندوق نفت ناموفق بود (کد %d): %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Slip struct {
			VeteranApplied bool  `json:"veteranApplied"`
			Tax            int64 `json:"tax"`
		} `json:"slip"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("پاسخ فیش صندوق نفت قابل تجزیه نیست: %v", err)
	}
	if body.Slip.VeteranApplied || body.Slip.Tax <= 0 {
		t.Fatalf("کارمند بدون پرونده نباید معافیت بگیرد: %+v", body.Slip)
	}
}
