// آزمون جریان سرتاسری «modules»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
)

// jdate تاریخ شمسی امروز به‌اضافهٔ افست روز.
func jdate(t *testing.T, offset int) string {
	t.Helper()
	today, err := calendar.FromGregorian(time.Now())
	if err != nil {
		t.Fatalf("تبدیل تاریخ امروز ناموفق: %v", err)
	}
	d, err := today.AddDays(offset)
	if err != nil {
		t.Fatalf("محاسبهٔ تاریخ آزمون ناموفق: %v", err)
	}
	return d.String()
}

func decodeInto(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dst); err != nil {
		t.Fatalf("پاسخ قابل تجزیه نیست: %v (%s)", err, rec.Body.String())
	}
}

// TestContractLifecycleAndDocuments جریان کامل قرارداد را از انعقاد تا ابلاغ و سند چاپی
// می‌سنجد؛ همین مسیر در مرورگر با همان API کار می‌کند.
func TestContractLifecycleAndDocuments(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	// فهرست قراردادهای نمونه باید پر باشد.
	rec := call(t, h, http.MethodGet, "/api/v1/contracts/contracts?size=100", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست قراردادها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var page contracts.ContractPage
	decodeInto(t, rec, &page)
	if page.Total < 10 {
		t.Fatalf("قراردادهای نمونه بارگذاری نشده‌اند (total=%d)", page.Total)
	}

	// انعقاد قرارداد تازه برای کارمندی که قرارداد جاری ندارد (EMP-012 خاتمه‌یافته است).
	create := map[string]any{
		"employeeId":    "EMP-012",
		"kind":          "fixed_term",
		"startDate":     jdate(t, 10),
		"endDate":       jdate(t, 375),
		"monthlySalary": 33000000,
		"noteFa":        "بازگشت به کار با قرارداد مدت موقت",
	}
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts", token, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("انعقاد قرارداد باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var contract contracts.Contract
	decodeInto(t, rec, &contract)
	if contract.Status != "draft" {
		t.Fatalf("قرارداد تازه باید پیش‌نویس باشد، «%s» بود", contract.Status)
	}

	// قاعدهٔ هم‌پوشانی: قرارداد دوم هم‌بازه باید رد شود.
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts", token, map[string]any{
		"employeeId": "EMP-012", "kind": "fixed_term",
		"startDate": jdate(t, 20), "endDate": jdate(t, 200), "monthlySalary": 30000000,
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("قرارداد هم‌پوشان باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// ارسال برای تأیید و سپس ابلاغ.
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts/"+contract.ID+"/submit", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ارسال برای تأیید باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &contract)
	if contract.Status != "pending_approval" {
		t.Fatalf("وضعیت پس از ارسال باید pending_approval باشد، «%s» بود", contract.Status)
	}

	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts/"+contract.ID+"/decision", token, map[string]any{"approve": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("تأیید قرارداد باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &contract)
	if contract.Status != "active" {
		t.Fatalf("وضعیت پس از تأیید باید active باشد، «%s» بود", contract.Status)
	}
	if len(contract.Events) < 3 {
		t.Fatalf("سابقهٔ رخدادهای قرارداد ثبت نشده است (%d رخداد)", len(contract.Events))
	}

	// قاعدهٔ سابقهٔ خدمت: تبدیل قرارداد تازه به مدت دائم باید رد شود.
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts/"+contract.ID+"/convert", token, map[string]any{})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تبدیل بدون یک سال سابقه باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// تمدید: قرارداد تازه‌ای در انتظار تأیید با شروع پس از پایان قرارداد جاری.
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts/"+contract.ID+"/renew", token, map[string]any{"monthlySalary": 36000000})
	if rec.Code != http.StatusCreated {
		t.Fatalf("تمدید باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var renewal contracts.Contract
	decodeInto(t, rec, &renewal)
	if renewal.SupersedesID != contract.ID || renewal.Status != "pending_approval" {
		t.Fatalf("قرارداد تمدید درست ساخته نشد: %+v", renewal)
	}
	if renewal.StartDate <= contract.EndDate {
		t.Fatalf("شروع تمدید باید پس از پایان قرارداد قبلی باشد: %s <= %s", renewal.StartDate, contract.EndDate)
	}

	// سند چاپی قرارداد (HTML راست‌چین با شیوه‌نامهٔ چاپ).
	rec = call(t, h, http.MethodGet, "/api/v1/documents/contract/"+contract.ID+"?print=1", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("سند قرارداد باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`dir="rtl"`, "@page", "قرارداد کار", "الهام رستمی", "window.print"} {
		if !strings.Contains(body, want) {
			t.Fatalf("سند چاپی «%s» را ندارد", want)
		}
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("نوع محتوای سند باید HTML باشد، «%s» بود", ct)
	}

	// سند نوع ناشناخته و سند بدون دسترسی.
	if rec := call(t, h, http.MethodGet, "/api/v1/documents/unknown/1", token, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("نوع سند ناشناخته باید ۴۰۴ بدهد، %d داد", rec.Code)
	}
	if rec := call(t, h, http.MethodGet, "/api/v1/documents/contract/CT-0000-0000", token, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("قرارداد ناموجود باید ۴۰۴ بدهد، %d داد", rec.Code)
	}
	if rec := call(t, h, http.MethodGet, "/api/v1/documents/contract/"+contract.ID, "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("سند بدون توکن باید ۴۰۱ بدهد، %d داد", rec.Code)
	}
}

// TestDependentWorkflow گردش‌کار تکفل را می‌سنجد: مدرک پیش از تأیید، تکراری نبودن کد ملی
// و خروج از تکفل با دلیل.
func TestDependentWorkflow(t *testing.T) {
	h := newTestServer(t)
	spec := login(t, h, "hr.spec", "demo")
	manager := login(t, h, "hr.manager", "demo")

	create := map[string]any{
		"employeeId":    "EMP-011",
		"fullName":      "زهرا قاسمی",
		"relation":      "child",
		"nationalId":    "0033445567",
		"birthDate":     jdate(t, -1500),
		"coverageStart": jdate(t, 5),
		"insurance":     true,
		"documentsFa":   "کپی شناسنامه",
	}
	rec := call(t, h, http.MethodPost, "/api/v1/personnel/dependents", spec.AccessToken, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت تکفل باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var dependent contracts.Dependent
	decodeInto(t, rec, &dependent)
	if dependent.Status != "pending" {
		t.Fatalf("تکفل تازه باید در انتظار بررسی باشد، «%s» بود", dependent.Status)
	}

	// کد ملی تکراری برای همان کارمند.
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/dependents", spec.AccessToken, create)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("کد ملی تکراری باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// تأیید بدون مدرک: قاعده باید بگیرد.
	noDocs := map[string]any{
		"employeeId": "EMP-004", "fullName": "سمیرا مرادی", "relation": "spouse",
		"nationalId": "0087654326", "coverageStart": jdate(t, 1), "insurance": true,
	}
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/dependents", spec.AccessToken, noDocs)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت تکفل باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var noDocsDep contracts.Dependent
	decodeInto(t, rec, &noDocsDep)
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/dependents/"+noDocsDep.ID+"/decision", manager.AccessToken, map[string]any{"approve": true})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تأیید بدون مدرک باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// تأیید با مدرک، سپس خروج از تکفل با دلیل.
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/dependents/"+dependent.ID+"/decision", manager.AccessToken, map[string]any{"approve": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("تأیید تکفل باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &dependent)
	if dependent.Status != "approved" {
		t.Fatalf("وضعیت تکفل پس از تأیید باید approved باشد، «%s» بود", dependent.Status)
	}
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/dependents/"+dependent.ID+"/remove", manager.AccessToken, map[string]any{"noteFa": "اشتغال و پوشش بیمهٔ مستقل"})
	if rec.Code != http.StatusOK {
		t.Fatalf("خروج از تکفل باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &dependent)
	if dependent.Status != "removed" || dependent.Insurance {
		t.Fatalf("پس از خروج از تکفل، وضعیت و پوشش بیمه باید بسته شود: %+v", dependent)
	}
}

// TestFlightRequestWorkflow فرم پرواز را از ثبت تا تسویه می‌سنجد و اعتبارسنجی تاریخ و
// شمارهٔ بلیت را بررسی می‌کند.
func TestFlightRequestWorkflow(t *testing.T) {
	h := newTestServer(t)
	line := login(t, h, "line.mgr", "demo")

	// EMP-010 در واحد «فناوری اطلاعات» است و مدیر همان واحد (line.mgr) محدودهٔ داده دارد.
	rec := call(t, h, http.MethodPost, "/api/v1/personnel/flight-requests", line.AccessToken, map[string]any{
		"employeeId": "EMP-010", "originFa": "ماهشهر", "destinationFa": "تهران",
		"departDate": jdate(t, -3), "purpose": "mission", "passengers": 1, "estimatedCost": 15000000,
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تاریخ پرواز گذشته باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	rec = call(t, h, http.MethodPost, "/api/v1/personnel/flight-requests", line.AccessToken, map[string]any{
		"employeeId": "EMP-010", "originFa": "ماهشهر", "destinationFa": "تهران",
		"departDate": jdate(t, 14), "returnDate": jdate(t, 18), "purpose": "mission",
		"passengers": 2, "companionsFa": "همراه با یک تکفل‌شده", "estimatedCost": 32000000,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت فرم پرواز باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var flight contracts.FlightRequest
	decodeInto(t, rec, &flight)
	if flight.Status != "pending" {
		t.Fatalf("فرم تازه باید در انتظار تأیید باشد، «%s» بود", flight.Status)
	}

	// صدور بلیت بدون شمارهٔ بلیت باید رد شود، ولی نخست باید تأیید شده باشد.
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/flight-requests/"+flight.ID+"/decision", line.AccessToken, map[string]any{"action": "issue", "ticketNo": "IR-1"})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("صدور بلیت پیش از تأیید باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/flight-requests/"+flight.ID+"/decision", line.AccessToken, map[string]any{"action": "approve"})
	if rec.Code != http.StatusOK {
		t.Fatalf("تأیید فرم پرواز باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &flight)
	if flight.Status != "approved" {
		t.Fatalf("وضعیت پس از تأیید باید approved باشد، «%s» بود", flight.Status)
	}
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/flight-requests/"+flight.ID+"/decision", line.AccessToken, map[string]any{"action": "issue"})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("صدور بلیت بدون شماره باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/flight-requests/"+flight.ID+"/decision", line.AccessToken, map[string]any{"action": "issue", "ticketNo": "IR-9901"})
	if rec.Code != http.StatusOK {
		t.Fatalf("صدور بلیت باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/personnel/flight-requests/"+flight.ID+"/decision", line.AccessToken, map[string]any{"action": "settle"})
	if rec.Code != http.StatusOK {
		t.Fatalf("تسویه باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &flight)
	if flight.Status != "settled" || flight.TicketNo != "IR-9901" {
		t.Fatalf("فرم پس از تسویه نادرست است: %+v", flight)
	}
}

// TestSalaryDecreeAndCertificateWorkflow گردش‌کار حکم حقوق (پیش‌نویس ← صدور ← ابلاغ ← اصلاح)
// و گواهی اشتغال (درخواست ← تأیید ← صدور ← تحویل) را می‌سنجد.
func TestSalaryDecreeAndCertificateWorkflow(t *testing.T) {
	h := newTestServer(t)
	spec := login(t, h, "hr.spec", "demo")
	manager := login(t, h, "hr.manager", "demo")

	// فهرست نمونه باید پر باشد.
	rec := call(t, h, http.MethodGet, "/api/v1/salary_admin/decrees?size=100", spec.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست احکام باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var page contracts.SalaryDecreePage
	decodeInto(t, rec, &page)
	if page.Total < 3 {
		t.Fatalf("احکام نمونه بارگذاری نشده‌اند (total=%d)", page.Total)
	}

	// صدور پیش‌نویس حکم تازه.
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees", spec.AccessToken, map[string]any{
		"employeeId": "EMP-004", "jobGroup": "کارشناس ارشد", "baseSalary": 245000000,
		"effectiveDate": jdate(t, 1), "reasonFa": "افزایش سنواتی",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("صدور حکم باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var decree contracts.SalaryDecree
	decodeInto(t, rec, &decree)
	if decree.Status != "draft" {
		t.Fatalf("حکم تازه باید پیش‌نویس باشد، «%s» بود", decree.Status)
	}

	// ابلاغ پیش از صدور باید رد شود.
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/"+decree.ID+"/notify", manager.AccessToken, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ابلاغ پیش از صدور باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// صدور و ابلاغ.
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/"+decree.ID+"/issue", spec.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("صدور حکم باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/"+decree.ID+"/notify", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ابلاغ حکم باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &decree)
	if decree.Status != "notified" {
		t.Fatalf("وضعیت پس از ابلاغ باید notified باشد، «%s» بود", decree.Status)
	}

	// اصلاح حکم: حکم تازهٔ پیش‌نویس و منسوخ‌شدن حکم پیشین پس از ابلاغ.
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/"+decree.ID+"/revise", spec.AccessToken, map[string]any{
		"baseSalary": 260000000, "reasonFa": "بازنگری مأخذ پرداخت",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("اصلاح حکم باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var revised contracts.SalaryDecree
	decodeInto(t, rec, &revised)
	if revised.SupersedesID != decree.ID || revised.BaseSalary != 260000000 {
		t.Fatalf("حکم اصلاحی نادرست ساخته شد: %+v", revised)
	}
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/"+revised.ID+"/issue", spec.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("صدور حکم اصلاحی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/"+revised.ID+"/notify", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ابلاغ حکم اصلاحی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodGet, "/api/v1/salary_admin/decrees/"+decree.ID, manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("خواندن حکم پیشین باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	var previous contracts.SalaryDecree
	decodeInto(t, rec, &previous)
	if previous.Status != "superseded" {
		t.Fatalf("حکم پیشین پس از ابلاغ جانشین باید منسوخ شود، «%s» بود", previous.Status)
	}

	// حکم حقوق چاپی (سند پلتفرم) باید برای کارمند صادر شود.
	rec = call(t, h, http.MethodGet, "/api/v1/documents/salary_decree/EMP-004?print=1", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("سند حکم حقوق باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "حکم حقوق") {
		t.Fatalf("سند حکم حقوق، عنوان درست را ندارد")
	}

	// گردش‌کار گواهی اشتغال.
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/certificates", spec.AccessToken, map[string]any{
		"employeeId": "EMP-010", "kind": "bank", "purposeFa": "بانک ملی",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت درخواست گواهی باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var cert contracts.EmploymentCertificate
	decodeInto(t, rec, &cert)
	if cert.Status != "requested" {
		t.Fatalf("گواهی تازه باید در وضعیت درخواست باشد، «%s» بود", cert.Status)
	}
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/certificates/"+cert.ID+"/issue", spec.AccessToken, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("صدور گواهی پیش از تأیید باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/certificates/"+cert.ID+"/decision", manager.AccessToken, map[string]any{"approve": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("تأیید گواهی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/certificates/"+cert.ID+"/issue", spec.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("صدور گواهی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/certificates/"+cert.ID+"/deliver", spec.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("تحویل گواهی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	decodeInto(t, rec, &cert)
	if cert.Status != "delivered" || cert.IssuedAt == "" || cert.DeliveredAt == "" {
		t.Fatalf("گواهی پس از تحویل نادرست است: %+v", cert)
	}

	// گواهی اشتغال چاپی.
	rec = call(t, h, http.MethodGet, "/api/v1/documents/employment_certificate/EMP-010", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("سند گواهی اشتغال باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
}

// TestReadyzReportsStore آمادگی سرویس را می‌سنجد: بدون پایگاه‌داده باید in-memory گزارش شود.
func TestReadyzReportsStore(t *testing.T) {
	h := newTestServer(t)
	rec := call(t, h, http.MethodGet, "/readyz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("بررسی آمادگی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	decodeInto(t, rec, &body)
	if body["status"] != "ok" || body["database"] != "in-memory" {
		t.Fatalf("گزارش آمادگی نادرست است: %v", body)
	}
}

// TestParametricPayrollTemplatesAndReportBuilder پارامتریک بودن حقوق، قالب‌های متن اسناد
// و گزارش‌ساز بر پایهٔ اقلام را روی API واقعی می‌سنجد.
func TestParametricPayrollTemplatesAndReportBuilder(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	// پارامترها و ضرائب حقوق: باید از رکوردهای پارامتریک برگردد (نه عدد ثابت در کد).
	rec := call(t, h, http.MethodGet, "/api/v1/payroll/parameters", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("پارامترهای حقوق باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var params struct {
		Values map[string]string `json:"values"`
		Typed  struct {
			TaxExemption int64 `json:"taxExemption"`
		} `json:"typed"`
	}
	decodeInto(t, rec, &params)
	if params.Values["insurance.employeeRate"] != "0.07" {
		t.Fatalf("نرخ بیمهٔ کارمند باید از پارامتر خوانده شود: %v", params.Values)
	}
	if params.Typed.TaxExemption <= 0 {
		t.Fatalf("معافیت مالیاتی باید مثبت باشد: %+v", params.Typed)
	}

	// اقلام حقوق: باید اقلام پارامتریک بذر شده باشند.
	rec = call(t, h, http.MethodGet, "/api/v1/payroll/records?kind=payItems&size=100", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("اقلام حقوق باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var items contracts.RecordPage
	decodeInto(t, rec, &items)
	if items.Total < 10 {
		t.Fatalf("اقلام حقوق بذر نشده‌اند (total=%d)", items.Total)
	}

	// شبیه‌سازی فیش بر پایهٔ پارامترها و اقلام فعال.
	sim := call(t, h, http.MethodPost, "/api/v1/payroll/simulate", token, map[string]any{
		"baseSalary": 120000000, "children": 2, "overtimeHours": 10, "overtimeHourlyRate": 700000,
	})
	if sim.Code != http.StatusOK {
		t.Fatalf("شبیه‌سازی فیش باید ۲۰۰ بدهد، %d داد: %s", sim.Code, sim.Body.String())
	}
	var simOut struct {
		Result struct {
			Gross int64 `json:"gross"`
			Net   int64 `json:"net"`
		} `json:"result"`
		Elements []struct {
			Code     string  `json:"code"`
			Calc     string  `json:"calc"`
			Amount   float64 `json:"amount"`
			Resolved bool    `json:"resolved"`
		} `json:"elements"`
	}
	decodeInto(t, sim, &simOut)
	if simOut.Result.Gross <= 0 || simOut.Result.Net >= simOut.Result.Gross {
		t.Fatalf("محاسبهٔ فیش نادرست است: %+v", simOut.Result)
	}
	if len(simOut.Elements) < 10 {
		t.Fatalf("تفکیک اقلام در شبیه‌سازی کم است: %d", len(simOut.Elements))
	}
	// اقلام فرمولی باید عدد قطعی بگیرند، نه resolved:false.
	byCode := map[string]struct {
		Amount   float64
		Resolved bool
	}{}
	for _, e := range simOut.Elements {
		byCode[e.Code] = struct {
			Amount   float64
			Resolved bool
		}{e.Amount, e.Resolved}
	}
	for _, code := range []string{"TAX", "OVERTIME", "INSURANCE_EE", "WAGE_JOB", "BAD_CLIMATE", "MARRIAGE"} {
		line, ok := byCode[code]
		if !ok {
			t.Fatalf("قلم %s در تفکیک شبیه‌سازی نیست", code)
		}
		if !line.Resolved || line.Amount <= 0 {
			t.Fatalf("قلم فرمولی %s حل نشده است: resolved=%v amount=%v", code, line.Resolved, line.Amount)
		}
	}

	// فیش پارامتریک کارمند: باید مزد مبنا و فوق‌العاده‌ها را عددی برگرداند.
	ps := call(t, h, http.MethodGet, "/api/v1/payroll/payslip?employee=EMP-001", token, nil)
	if ps.Code != http.StatusOK {
		t.Fatalf("فیش پارامتریک باید ۲۰۰ بدهد، %d داد: %s", ps.Code, ps.Body.String())
	}
	var slip struct {
		Period string `json:"period"`
		Wages  struct {
			Job float64 `json:"job"`
		} `json:"wages"`
		Elements []struct {
			Code     string  `json:"code"`
			Category string  `json:"category"`
			Resolved bool    `json:"resolved"`
			Amount   float64 `json:"amount"`
		} `json:"elements"`
	}
	decodeInto(t, ps, &slip)
	if slip.Period == "" {
		t.Fatal("دورهٔ فیش پارامتریک خالی است")
	}
	if slip.Wages.Job <= 0 {
		t.Fatalf("مزد شغل در فیش نیامده است: %+v", slip.Wages)
	}
	foundClimate := false
	for _, e := range slip.Elements {
		if e.Code == "BAD_CLIMATE" {
			foundClimate = true
			if !e.Resolved || e.Amount <= 0 {
				t.Fatalf("فوق‌العادهٔ بدی آب و هوا در فیش عددی نیست: %+v", e)
			}
		}
	}
	if !foundClimate {
		t.Fatal("فوق‌العادهٔ بدی آب و هوا در فیش کارمند نیست")
	}

	// قالب متن اسناد: پیش‌نمایش فیش باید HTML برگرداند.
	prev := call(t, h, http.MethodGet, "/api/v1/documents/preview/payslip", token, nil)
	if prev.Code != http.StatusOK {
		t.Fatalf("پیش‌نمایش قالب فیش باید ۲۰۰ بدهد، %d داد: %s", prev.Code, prev.Body.String())
	}
	if ct := prev.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("پیش‌نمایش باید HTML باشد: %s", ct)
	}
	// فیش چاپی باید همان تفکیک اقلام شبیه‌ساز را نشان دهد.
	if body := prev.Body.String(); !strings.Contains(body, "تفکیک اقلام حقوق") {
		t.Fatalf("فیش چاپی تفکیک اقلام حقوق را ندارد")
	}

	// حکم کارگزینی و قرارداد کار باید جدول اجزای مزد و فوق‌العاده‌ها را داشته باشند.
	for _, docKind := range []string{"salary_decree", "contract"} {
		doc := call(t, h, http.MethodGet, "/api/v1/documents/preview/"+docKind, token, nil)
		if doc.Code != http.StatusOK {
			t.Fatalf("پیش‌نمایش %s باید ۲۰۰ بدهد، %d داد: %s", docKind, doc.Code, doc.Body.String())
		}
		body := doc.Body.String()
		if !strings.Contains(body, "اجزای مزد و فوق") {
			t.Fatalf("سند %s جدول اجزای مزد را ندارد", docKind)
		}
		if !strings.Contains(body, "بدی آب و هوا") {
			t.Fatalf("سند %s فوق‌العادهٔ بدی آب و هوا را ندارد", docKind)
		}
	}

	// گزارش‌ساز: منابع داده و اجرای گزارش بر پایهٔ اقلام انتخابی.
	src := call(t, h, http.MethodGet, "/api/v1/reports/sources", token, nil)
	if src.Code != http.StatusOK {
		t.Fatalf("منابع گزارش باید ۲۰۰ بدهد، %d داد: %s", src.Code, src.Body.String())
	}
	if !strings.Contains(src.Body.String(), "payroll_elements") {
		t.Fatalf("منبع اقلام حقوق در فهرست منابع نیست: %s", src.Body.String())
	}
	run := call(t, h, http.MethodPost, "/api/v1/reports/run", token, map[string]any{
		"source": "payroll_elements", "items": []string{"code", "title", "rate"}, "limit": 100,
	})
	if run.Code != http.StatusOK {
		t.Fatalf("اجرای گزارش باید ۲۰۰ بدهد، %d داد: %s", run.Code, run.Body.String())
	}
	var report contracts.ReportResult
	decodeInto(t, run, &report)
	if len(report.Columns) != 3 || report.Total == 0 {
		t.Fatalf("خروجی گزارش نادرست است: columns=%d total=%d", len(report.Columns), report.Total)
	}
	// اقلام انتخابی باید به ترتیب ستون‌ها برگردند تا خروجی چاپی همین گزارش را بازسازی کند.
	if len(report.Items) != 3 || report.Items[0] != "code" {
		t.Fatalf("اقلام گزارش در خروجی نیست: %v", report.Items)
	}

	// سرصفحهٔ برند روی گزارش: نشان و نام شرکت از پروفایل پارامتریک می‌آید، نه از کلاینت.
	if report.Brand == nil {
		t.Fatal("سرصفحهٔ برند روی گزارش ننشسته است")
	}
	if report.Brand.ShortCode != "BIPC" {
		t.Fatalf("کد شرکت در سرصفحهٔ گزارش نادرست است: %+v", report.Brand)
	}
	if report.Brand.LogoURL != "/assets/brand/logo-pgpic.jpg" {
		t.Fatalf("نشان شرکت در سرصفحهٔ گزارش نادرست است: %q", report.Brand.LogoURL)
	}
	if report.Brand.GeneratedAt == "" || report.Brand.GeneratedBy == "" {
		t.Fatalf("تاریخ/تهیه‌کنندهٔ گزارش ثبت نشده است: %+v", report.Brand)
	}

	// فیلتر چندانتخابی گزارش کارکنان: سرور باید فیلتر «نوع استخدام» را چندانتخابی با
	// گزینه‌های کاتالوگ اعلام کند (تا رابط کشویی بازشو با تیک بسازد) و چند مقدار
	// کاما-جدا را «هر یک از این انواع» بفهمد.
	var sourceMeta struct {
		Sources []contracts.ReportSourceMeta `json:"sources"`
	}
	decodeInto(t, src, &sourceMeta)
	var employmentFilter *contracts.ReportFilterDef
	for i, m := range sourceMeta.Sources {
		if m.Key != "employees" {
			continue
		}
		for j, f := range m.Filters {
			if f.Key == "employmentType" {
				employmentFilter = &sourceMeta.Sources[i].Filters[j]
			}
		}
	}
	if employmentFilter == nil {
		t.Fatal("منبع گزارش کارکنان فیلتر «نوع استخدام» ندارد")
	}
	if !employmentFilter.Multiple || employmentFilter.Type != "select" || len(employmentFilter.Options) != 10 {
		t.Fatalf("فیلتر «نوع استخدام» باید کشویی و چندانتخابی با ۱۰ گزینه باشد: %+v", *employmentFilter)
	}

	empRun := call(t, h, http.MethodPost, "/api/v1/reports/run", token, map[string]any{
		"source": "employees", "items": []string{"id", "employmentTypeFa"},
		"filters": map[string]string{"employmentType": "official_worker,fixed_1662"}, "limit": 200,
	})
	if empRun.Code != http.StatusOK {
		t.Fatalf("گزارش کارکنان با فیلتر چندانتخابی باید ۲۰۰ بدهد، %d داد: %s", empRun.Code, empRun.Body.String())
	}
	var empReport contracts.ReportResult
	decodeInto(t, empRun, &empReport)
	if empReport.Total == 0 {
		t.Fatal("فیلتر چندانتخابی گزارش کارکنان هیچ سطری برنگرداند")
	}
	for _, row := range empReport.Rows {
		if len(row) != 2 || (row[1] != "کارگر رسمی" && row[1] != "کارمند مدت معین ۱۶۶۲") {
			t.Fatalf("سطر خارج از فیلتر چندانتخابی گزارش: %v", row)
		}
	}

	// ‌پس‌رفت‌گیری: فیلتر تک‌مقداری همان رفتار پیشین را دارد.
	singleRun := call(t, h, http.MethodPost, "/api/v1/reports/run", token, map[string]any{
		"source": "employees", "items": []string{"id", "employmentTypeFa"},
		"filters": map[string]string{"employmentType": "official_oil"}, "limit": 200,
	})
	if singleRun.Code != http.StatusOK {
		t.Fatalf("گزارش کارکنان با فیلتر تک‌مقداری باید ۲۰۰ بدهد، %d داد: %s", singleRun.Code, singleRun.Body.String())
	}
	var singleReport contracts.ReportResult
	decodeInto(t, singleRun, &singleReport)
	for _, row := range singleReport.Rows {
		if len(row) != 2 || row[1] != "کارمند رسمی نفت" {
			t.Fatalf("فیلتر تک‌مقداری گزارش نشت کرد: %v", row)
		}
	}
}

// TestAccessGrantLifecycle می‌سنجد که مدیر سیستم می‌تواند برای هر اداره و هر ماژول به یک
// نقش دسترسی بدهد، دسترسی فوراً روی همان درخواست‌ها اعمال شود و با حذف سطح برگردد.
func TestAccessGrantLifecycle(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo")
	emp := login(t, h, "employee", "demo")

	// سطوح دسترسی ویژهٔ مدیر سیستم است.
	if rec := call(t, h, http.MethodGet, "/api/v1/settings/access", emp.AccessToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("کارمند نباید به سطوح دسترسی برسد، %d داد", rec.Code)
	}
	if rec := call(t, h, http.MethodGet, "/api/v1/settings/records?kind=sysParams", emp.AccessToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("کارمند پیش از سطح دسترسی نباید به تنظیمات برسد، %d داد", rec.Code)
	}

	// ماتریس دسترسی برای مدیر سیستم.
	rec := call(t, h, http.MethodGet, "/api/v1/settings/access", admin.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ماتریس دسترسی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var matrix struct {
		Roles   []map[string]string `json:"roles"`
		Modules []struct {
			ID       string              `json:"id"`
			Manifest map[string][]string `json:"manifest"`
		} `json:"modules"`
		Actions []map[string]string `json:"actions"`
	}
	decodeInto(t, rec, &matrix)
	if len(matrix.Roles) == 0 || len(matrix.Modules) == 0 || len(matrix.Actions) == 0 {
		t.Fatalf("ماتریس دسترسی ناقص است: roles=%d modules=%d actions=%d", len(matrix.Roles), len(matrix.Modules), len(matrix.Actions))
	}

	// سطح دسترسی تازه: کارمند در ادارهٔ D1 ماژول تنظیمات را بخواند.
	grant := call(t, h, http.MethodPost, "/api/v1/settings/access/grants", admin.AccessToken, map[string]any{
		"role": "employee", "moduleId": "settings", "departmentId": "D1", "actions": []string{"read"},
	})
	if grant.Code != http.StatusOK {
		t.Fatalf("ثبت سطح دسترسی باید ۲۰۰ بدهد، %d داد: %s", grant.Code, grant.Body.String())
	}
	var g struct {
		ID string `json:"id"`
	}
	decodeInto(t, grant, &g)
	if g.ID == "" {
		t.Fatal("شناسهٔ سطح دسترسی ساخته نشد")
	}
	if rec := call(t, h, http.MethodGet, "/api/v1/settings/records?kind=sysParams", emp.AccessToken, nil); rec.Code != http.StatusOK {
		t.Fatalf("پس از سطح دسترسی، کارمند باید تنظیمات را بخواند، %d داد: %s", rec.Code, rec.Body.String())
	}
	// نقش دیگر همان اداره نباید دسترسی بگیرد.
	if rec := call(t, h, http.MethodGet, "/api/v1/settings/records?kind=sysParams", login(t, h, "line.mgr", "demo").AccessToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("سطح دسترسی نباید به نقش دیگر سرایت کند، %d داد", rec.Code)
	}

	// ورودی نامعتبر رد می‌شود.
	bad := call(t, h, http.MethodPost, "/api/v1/settings/access/grants", admin.AccessToken, map[string]any{
		"role": "nobody", "moduleId": "ghost", "actions": []string{},
	})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("سطح دسترسی نامعتبر باید ۴۰۰ بدهد، %d داد", bad.Code)
	}

	// حذف سطح، دسترسی را برمی‌گرداند.
	if del := call(t, h, http.MethodDelete, "/api/v1/settings/access/grants/"+g.ID, admin.AccessToken, nil); del.Code != http.StatusOK {
		t.Fatalf("حذف سطح دسترسی باید ۲۰۰ بدهد، %d داد", del.Code)
	}
	if rec := call(t, h, http.MethodGet, "/api/v1/settings/records?kind=sysParams", emp.AccessToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("پس از حذف سطح، دسترسی باید برگردد، %d داد", rec.Code)
	}
}

// TestEmploymentTypesAndGroupDecree انواع استخدام و صدور گروهی حکم کارگزینی را می‌سنجد:
// فهرست رسمی، فیلتر کارکنان بر پایهٔ نوع استخدام، پیش‌نمایش و صدور واقعی گروهی.
func TestEmploymentTypesAndGroupDecree(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo")

	// ۱) فهرست رسمی انواع استخدام با ۱۰ نوع و همهٔ کلیدهای خواسته‌شده.
	rec := call(t, h, http.MethodGet, "/api/v1/salary_admin/options", admin.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست گزینه‌های احکام باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var options struct {
		EmploymentTypes []contracts.EmploymentType `json:"employmentTypes"`
	}
	decodeInto(t, rec, &options)
	if len(options.EmploymentTypes) != 10 {
		t.Fatalf("باید ۱۰ نوع استخدام باشد، %d بود", len(options.EmploymentTypes))
	}
	want := []string{"official_oil", "official_worker", "fixed_1662", "fixed_holding", "long_term",
		"contractor_staff", "contractor_project", "contractor_current", "seconded_to", "seconded_from"}
	got := map[string]bool{}
	for _, tp := range options.EmploymentTypes {
		if tp.TitleFa == "" || tp.GroupFa == "" || tp.DecreeFa == "" {
			t.Fatalf("نوع استخدام «%s» باید عنوان، گروه و حکم داشته باشد", tp.Key)
		}
		got[tp.Key] = true
	}
	for _, k := range want {
		if !got[k] {
			t.Fatalf("نوع استخدام «%s» در فهرست رسمی نیست", k)
		}
	}

	// ۲) فیلتر کارکنان بر پایهٔ نوع استخدام واقعاً کار می‌کند.
	rec = call(t, h, http.MethodGet, "/api/v1/employee/employees?employmentType=contractor_project&size=100", admin.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فیلتر نوع استخدام باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var contractors contracts.EmployeePage
	decodeInto(t, rec, &contractors)
	if contractors.Total == 0 {
		t.Fatal("باید دست‌کم یک پیمانکار پروژه‌ای نمونه وجود داشته باشد")
	}
	for _, e := range contractors.Items {
		if e.EmploymentType != "contractor_project" {
			t.Fatalf("فیلتر نوع استخدام نشت کرد: %s با نوع «%s»", e.ID, e.EmploymentType)
		}
		if e.EmploymentTypeFa == "" {
			t.Fatalf("عنوان فارسی نوع استخدام کارمند %s خالی است", e.ID)
		}
	}

	// ۲ب) فیلتر چندانتخابی: «کارگر رسمی» و «کارمند مدت معین ۱۶۶۲» با هم. این همان
	// شکلی است که رابط فیلترهای کشویی تیک‌خورنی می‌فرستد (مقدارها با کاما).
	rec = call(t, h, http.MethodGet,
		"/api/v1/employee/employees?employmentType=official_worker,fixed_1662&size=100", admin.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فیلتر چندانتخابی نوع استخدام باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var multiTypes contracts.EmployeePage
	decodeInto(t, rec, &multiTypes)
	if multiTypes.Total == 0 {
		t.Fatal("فیلتر چندانتخابی باید دست‌کم یک کارمند برگرداند")
	}
	seenType := map[string]bool{}
	for _, e := range multiTypes.Items {
		if e.EmploymentType != "official_worker" && e.EmploymentType != "fixed_1662" {
			t.Fatalf("فیلتر چندانتخابی نشت کرد: %s با نوع «%s»", e.ID, e.EmploymentType)
		}
		seenType[e.EmploymentType] = true
	}
	if !seenType["official_worker"] || !seenType["fixed_1662"] {
		t.Fatalf("فیلتر چندانتخابی باید هر دو نوع تیک‌خورده را برگرداند، %v برگشت", seenType)
	}

	// ۳) پیش‌نمایش صدور گروهی: هیچ حکمی ساخته نمی‌شود.
	groupBody := map[string]any{
		"employmentType": "official_oil", "increasePercent": 20,
		"effectiveDate": jdate(t, 30), "reasonFa": "افزایش سالانه ۱۴۰۵",
	}
	previewBody := map[string]any{}
	for k, v := range groupBody {
		previewBody[k] = v
	}
	previewBody["dryRun"] = true
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/group", admin.AccessToken, previewBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("پیش‌نمایش گروهی باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var preview contracts.SalaryDecreeGroupResult
	decodeInto(t, rec, &preview)
	if !preview.DryRun {
		t.Fatal("پاسخ پیش‌نمایش باید dryRun باشد")
	}
	if preview.Created != 0 {
		t.Fatalf("پیش‌نمایش نباید حکمی بسازد، %d ساخت", preview.Created)
	}
	if len(preview.Rows) == 0 {
		t.Fatal("پیش‌نمایش باید دست‌کم یک ردیف هدف داشته باشد")
	}
	for _, row := range preview.Rows {
		if row.NewSalary <= row.OldSalary {
			t.Fatalf("با افزایش ۲۰٪ مبلغ تازه باید بیشتر باشد: %d → %d", row.OldSalary, row.NewSalary)
		}
		if row.EmploymentTypeFa == "" {
			t.Fatalf("ردیف پیش‌نمایش باید عنوان نوع استخدام داشته باشد: %s", row.EmployeeID)
		}
	}

	// ۴) صدور واقعی گروهی و تأیید ساخته شدن احکام.
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/group", admin.AccessToken, groupBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("صدور گروهی باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var result contracts.SalaryDecreeGroupResult
	decodeInto(t, rec, &result)
	if result.Created == 0 || result.Created != len(preview.Rows) {
		t.Fatalf("صدور گروهی باید %d حکم بسازد، %d ساخت", len(preview.Rows), result.Created)
	}
	for _, row := range result.Rows {
		if row.Status != "draft" || row.DecreeID == "" {
			t.Fatalf("ردیف صدور گروهی باید حکم پیش‌نویس با شناسه داشته باشد: %+v", row)
		}
	}

	// ۵) حکم ساخته‌شده در فهرست احکام دیده می‌شود و نوع استخدام دارد.
	rec = call(t, h, http.MethodGet, "/api/v1/salary_admin/decrees?size=200", admin.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست احکام باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	var decrees contracts.SalaryDecreePage
	decodeInto(t, rec, &decrees)
	found := false
	for _, d := range decrees.Items {
		if d.ID == result.Rows[0].DecreeID {
			found = true
			if d.EmploymentType != "official_oil" {
				t.Fatalf("حکم گروهی باید نوع استخدام داشته باشد، «%s» بود", d.EmploymentType)
			}
		}
	}
	if !found {
		t.Fatalf("حکم گروهی %s در فهرست احکام پیدا نشد", result.Rows[0].DecreeID)
	}

	// ۶) ورودی نامعتبر (بدون افزایش و بدون هدف) رد می‌شود.
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees/group", admin.AccessToken, map[string]any{
		"employmentType": "official_oil", "effectiveDate": jdate(t, 5), "reasonFa": "بدون افزایش",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("صدور گروهی بدون افزایش باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
}
