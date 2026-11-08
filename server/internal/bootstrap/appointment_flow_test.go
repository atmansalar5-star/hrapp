// آزمون جریان سرتاسری «انتصاب و ترفیع»: از تصویب‌خواهی چندسطحی تا اجرا و ابطال ابلاغ.
//
// چه چیزی سنجیده می‌شود:
//   - تصویب‌خواهی شرط‌ها، زنجیرهٔ تصویب و گام‌های اجرا را برمی‌گرداند و کمبودها را مانع می‌داند؛
//   - امضای هر سطح با نقش همان سطح انجام می‌شود؛ امضای سطح با نقش نادرست و «تصویب نهایی»
//     پیش از امضای همهٔ سطح‌ها رد می‌شود؛
//   - بازگشت برای اصلاح، درخواست را بی‌دلیل به پیش‌نویس برنمی‌گرداند و با دلیل برمی‌گرداند؛
//   - اجرای ابلاغ پنج گام را واقعاً انجام می‌دهد: پست، واحد و گروه شغلی کارمند عوض می‌شود،
//     تصدی پست در تشکیلات ثبت و انتصاب پیشین پایانیافته می‌شود؛
//   - ابطال ابلاغ، تصدی را می‌بندد و پروندهٔ کارمند را به وضع پیشین برمی‌گرداند.
package bootstrap

import (
	"net/http"
	"testing"

	"hrms/internal/contracts"
)

// appointmentFields فیلدهای یک ابلاغ کامل.
func appointmentFields(decreeNo string) map[string]string {
	return map[string]string{
		"currentPosition": "کارشناس منابع انسانی", "currentUnit": "منابع انسانی",
		"position": "کارشناس منابع انسانی", "positionCode": "POS-120", "unit": "منابع انسانی",
		"currentGrade": "expert", "proposedGrade": "senior_expert",
		"serviceMonths": "30", "evaluationScore": "90", "vacancy": "yes", "commitment": "24",
		"effectiveDate": "1405/09/01", "decreeNo": decreeNo,
		"reason": "احراز شرایط پست و تأیید کمیتهٔ انتصابات و ترفیعات",
	}
}

// createAppointment یک رکورد انتصاب در وضعیت پیش‌نویس می‌سازد.
func createAppointment(t *testing.T, h http.Handler, token string, fields map[string]string) contracts.Record {
	t.Helper()
	created := call(t, h, http.MethodPost, "/api/v1/career/records", token, map[string]any{
		"kind": "appointments", "title": "انتصاب آزمون", "employeeId": "EMP-008", "fields": fields,
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("ثبت ابلاغ باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var rec contracts.Record
	decodeInto(t, created, &rec)
	return rec
}

// submitRecord درخواست را برای تصویب می‌فرستد.
func submitRecord(t *testing.T, h http.Handler, token, id string) {
	t.Helper()
	code := call(t, h, http.MethodPost, "/api/v1/career/records/"+id+"/action", token,
		map[string]any{"action": "submit"}).Code
	if code != http.StatusOK {
		t.Fatalf("ارسال برای تصویب باید ۲۰۰ بدهد، %d داد", code)
	}
}

// signLevel یک سطح تصویب را با توکن داده‌شده امضا می‌کند و کد پاسخ را برمی‌گرداند.
func signLevel(t *testing.T, h http.Handler, token, id string) int {
	t.Helper()
	return call(t, h, http.MethodPost, "/api/v1/career/appointment-sign", token,
		map[string]any{"id": id}).Code
}

// approvalStatus وضعیت فعلی رکورد را می‌خواند.
func approvalStatus(t *testing.T, h http.Handler, token, id string) string {
	t.Helper()
	rec := call(t, h, http.MethodGet, "/api/v1/career/records/"+id, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("خواندن رکورد باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	var out contracts.Record
	decodeInto(t, rec, &out)
	return out.Status
}

// TestAppointmentPlanBlocksShortfall تصویب‌خواهی را می‌سنجد: کمبود سابقه مانع است و
// زنجیرهٔ تصویب و گام‌های اجرا همیشه برمی‌گردند.
func TestAppointmentPlanBlocksShortfall(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken

	blocked := call(t, h, http.MethodPost, "/api/v1/career/appointment-plan", hr, map[string]any{
		"kind": "promotion", "currentRank": 3, "proposedRank": 4, "serviceMonths": 6,
		"evaluationScore": 90, "evaluationRecorded": true,
		"vacancyAvailable": true, "vacancyChecked": true, "effectiveDate": "1405/09/01",
	})
	if blocked.Code != http.StatusOK {
		t.Fatalf("سنجش تصویب‌خواهی باید ۲۰۰ بدهد، %d داد: %s", blocked.Code, blocked.Body.String())
	}
	var plan struct {
		Plan struct {
			Allowed       bool `json:"allowed"`
			BlockingCount int  `json:"blockingCount"`
			Approvals     []struct {
				TitleFa string `json:"titleFa"`
				RoleKey string `json:"roleKey"`
			} `json:"approvals"`
		} `json:"plan"`
		Execution []struct {
			Key string `json:"key"`
		} `json:"execution"`
	}
	decodeInto(t, blocked, &plan)
	if plan.Plan.Allowed || plan.Plan.BlockingCount == 0 {
		t.Fatalf("سابقهٔ ۶ ماه باید مانع ترفیع باشد: %+v", plan.Plan)
	}
	if len(plan.Plan.Approvals) < 2 {
		t.Fatalf("زنجیرهٔ تصویب باید دست‌کم دو سطح داشته باشد: %d", len(plan.Plan.Approvals))
	}
	if len(plan.Execution) != 5 {
		t.Fatalf("گام‌های اجرا باید پنج گام باشند: %d", len(plan.Execution))
	}
}

// TestAppointmentApprovalRequiresEveryLevel می‌سنجد که یک تصویب، زنجیره را نمی‌بندد.
func TestAppointmentApprovalRequiresEveryLevel(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	lm := login(t, h, "line.mgr", "demo").AccessToken

	rec := createAppointment(t, h, hr, appointmentFields("ABL-1405-801"))
	submitRecord(t, h, hr, rec.ID)

	// تصویب خام پیش از امضای سطح‌ها باید رد شود (همان دروازه‌ای که قبلاً نبود).
	if code := call(t, h, http.MethodPost, "/api/v1/career/records/"+rec.ID+"/action", hr,
		map[string]any{"action": "approve"}).Code; code == http.StatusOK {
		t.Fatal("تصویب نهایی نباید پیش از امضای همهٔ سطح‌ها ممکن باشد")
	}
	// مدیر منابع انسانی نمی‌تواند سطح نخست (مدیر واحد) را امضا کند.
	if code := signLevel(t, h, hr, rec.ID); code != http.StatusForbidden {
		t.Fatalf("امضای سطح مدیر واحد با نقش مدیر منابع انسانی باید ۴۰۳ بدهد، %d داد", code)
	}
	// مدیر واحد سطح نخست را امضا می‌کند؛ رکورد باید در «در انتظار تصویب» بماند.
	if code := signLevel(t, h, lm, rec.ID); code != http.StatusOK {
		t.Fatalf("امضای سطح نخست باید ۲۰۰ بدهد، %d داد", code)
	}
	if status := approvalStatus(t, h, hr, rec.ID); status != "pending" {
		t.Fatalf("پس از یک امضا وضعیت باید pending بماند: %s", status)
	}
	// مدیر منابع انسانی سطح دوم (آخرین سطح این درخواست) را امضا می‌کند.
	if code := signLevel(t, h, hr, rec.ID); code != http.StatusOK {
		t.Fatalf("امضای سطح دوم باید ۲۰۰ بدهد، %d داد", code)
	}
	if status := approvalStatus(t, h, hr, rec.ID); status != "approved" {
		t.Fatalf("پس از امضای همهٔ سطح‌ها وضعیت باید approved باشد: %s", status)
	}
	// سطح تازه‌ای برای امضا نمانده است.
	if code := signLevel(t, h, hr, rec.ID); code != http.StatusConflict {
		t.Fatalf("امضای سطح اضافه باید ۴۰۹ بدهد، %d داد", code)
	}
}

// TestAppointmentReturnForCorrection بازگشت برای اصلاح را می‌سنجد.
func TestAppointmentReturnForCorrection(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	lm := login(t, h, "line.mgr", "demo").AccessToken

	rec := createAppointment(t, h, hr, appointmentFields("ABL-1405-802"))
	submitRecord(t, h, hr, rec.ID)
	if code := signLevel(t, h, lm, rec.ID); code != http.StatusOK {
		t.Fatalf("امضای سطح نخست باید ۲۰۰ بدهد، %d داد", code)
	}
	// بازگشت بی‌دلیل مجاز نیست.
	if code := call(t, h, http.MethodPost, "/api/v1/career/appointment-return", hr,
		map[string]any{"id": rec.ID}).Code; code != http.StatusBadRequest {
		t.Fatalf("بازگشت بی‌دلیل باید ۴۰۰ بدهد، %d داد", code)
	}
	if code := call(t, h, http.MethodPost, "/api/v1/career/appointment-return", hr,
		map[string]any{"id": rec.ID, "noteFa": "شرح وظایف پست نیازمند بازنگری است"}).Code; code != http.StatusOK {
		t.Fatalf("بازگشت برای اصلاح باید ۲۰۰ بدهد، %d داد", code)
	}
	if status := approvalStatus(t, h, hr, rec.ID); status != "draft" {
		t.Fatalf("پس از بازگشت، درخواست باید پیش‌نویس شود: %s", status)
	}
}

// TestAppointmentExecuteAppliesDecreeThenRevoke جریان کامل اجرا و ابطال را می‌سنجد.
func TestAppointmentExecuteAppliesDecreeThenRevoke(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	lm := login(t, h, "line.mgr", "demo").AccessToken

	type employeeDTO struct {
		ID             string `json:"id"`
		Position       string `json:"position"`
		DepartmentName string `json:"departmentName"`
		JobGrade       string `json:"jobGrade"`
	}
	before := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-008", hr, nil)
	var employeeBefore employeeDTO
	decodeInto(t, before, &employeeBefore)
	if employeeBefore.Position == "کارشناس منابع انسانی" && employeeBefore.JobGrade == "senior_expert" {
		t.Skip("پرونده از قبل همین وضع را دارد؛ آزمون معنایی ندارد")
	}

	rec := createAppointment(t, h, hr, appointmentFields("ABL-1405-803"))
	submitRecord(t, h, hr, rec.ID)
	for _, token := range []string{lm, hr} {
		if code := signLevel(t, h, token, rec.ID); code != http.StatusOK {
			t.Fatalf("امضای سطح باید ۲۰۰ بدهد، %d داد", code)
		}
	}

	executed := call(t, h, http.MethodPost, "/api/v1/career/appointment-execute", hr,
		map[string]any{"id": rec.ID})
	if executed.Code != http.StatusOK {
		t.Fatalf("اجرای ابلاغ باید ۲۰۰ بدهد، %d داد: %s", executed.Code, executed.Body.String())
	}
	var exec struct {
		Record contracts.Record `json:"record"`
		Steps  []struct {
			Key       string `json:"key"`
			Done      bool   `json:"done"`
			MessageFa string `json:"messageFa"`
		} `json:"steps"`
	}
	decodeInto(t, executed, &exec)
	if exec.Record.Status != "issued" {
		t.Fatalf("وضعیت رکورد پس از اجرا باید issued باشد: %s", exec.Record.Status)
	}
	if len(exec.Steps) != 5 {
		t.Fatalf("گزارش اجرا باید پنج گام داشته باشد: %d", len(exec.Steps))
	}
	for _, step := range exec.Steps {
		if !step.Done {
			t.Fatalf("گام %s انجام نشد: %s", step.Key, step.MessageFa)
		}
	}
	assignmentID := exec.Record.Fields["assignmentId"]
	if assignmentID == "" {
		t.Fatal("شناسهٔ انتصاب تشکیلات در رکورد ثبت نشده است")
	}

	// اثر اجرا در پروندهٔ کارمند: پست و گروه شغلی.
	after := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-008", hr, nil)
	var employeeAfter employeeDTO
	decodeInto(t, after, &employeeAfter)
	if employeeAfter.Position != "کارشناس منابع انسانی" {
		t.Fatalf("پست کارمند پس از اجرا نادرست است: %q", employeeAfter.Position)
	}
	if employeeAfter.JobGrade != "senior_expert" {
		t.Fatalf("گروه شغلی کارمند پس از اجرا باید senior_expert باشد: %q", employeeAfter.JobGrade)
	}

	// ابطال ابلاغ: تصدی بسته و پرونده برگشته.
	revoked := call(t, h, http.MethodPost, "/api/v1/career/appointment-revoke", hr,
		map[string]any{"id": rec.ID, "noteFa": "خطا در انتخاب پست؛ ابلاغ باطل شد"})
	if revoked.Code != http.StatusOK {
		t.Fatalf("ابطال ابلاغ باید ۲۰۰ بدهد، %d داد: %s", revoked.Code, revoked.Body.String())
	}
	if status := approvalStatus(t, h, hr, rec.ID); status != "revoked" {
		t.Fatalf("وضعیت رکورد پس از ابطال باید revoked باشد: %s", status)
	}
	restored := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-008", hr, nil)
	var employeeRestored employeeDTO
	decodeInto(t, restored, &employeeRestored)
	if employeeRestored.Position != employeeBefore.Position || employeeRestored.JobGrade != employeeBefore.JobGrade {
		t.Fatalf("پروندهٔ کارمند باید به وضع پیشین برگردد: قبل %q/%q — بعد %q/%q",
			employeeBefore.Position, employeeBefore.JobGrade, employeeRestored.Position, employeeRestored.JobGrade)
	}

	// تصدی تشکیلات باید پایانیافته باشد.
	assignments := call(t, h, http.MethodGet,
		"/api/v1/orgstructure/records?kind=assignments&size=200", hr, nil)
	var page contracts.RecordPage
	decodeInto(t, assignments, &page)
	for _, item := range page.Items {
		if item.ID == assignmentID && item.Status != "ended" {
			t.Fatalf("تصدی %s پس از ابطال باید ended باشد: %s", assignmentID, item.Status)
		}
	}
}

// TestAppointmentExecuteClosesPreviousDecree می‌سنجد که اجرای ابلاغ تازه، تصدی پیشین همان
// کارمند را پایانیافته می‌کند (کارمند نباید هم‌زمان دو تصدی فعال داشته باشد).
func TestAppointmentExecuteClosesPreviousDecree(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	lm := login(t, h, "line.mgr", "demo").AccessToken
	admin := login(t, h, "admin", "demo").AccessToken

	before := call(t, h, http.MethodGet, "/api/v1/orgstructure/records?kind=assignments&size=200", hr, nil)
	var pageBefore contracts.RecordPage
	decodeInto(t, before, &pageBefore)
	activeBefore := []string{}
	for _, item := range pageBefore.Items {
		if item.EmployeeID == "EMP-009" && item.Status == "active" {
			activeBefore = append(activeBefore, item.ID)
		}
	}
	if len(activeBefore) == 0 {
		t.Skip("دادهٔ نمونه برای EMP-009 تصدی فعالی ندارد")
	}

	fields := map[string]string{
		"currentPosition": "کارشناس ارشد فرآیند", "currentUnit": "تولید و عملیات",
		"position": "مدیر تولید", "positionCode": "POS-310", "unit": "تولید و عملیات",
		"currentGrade": "supervisor", "proposedGrade": "manager",
		"serviceMonths": "42", "evaluationScore": "91", "vacancy": "yes", "commitment": "36",
		"effectiveDate": "1405/09/01", "decreeNo": "ABL-1405-902",
		"reason": "بازنشستگی مدیر پیشین و احراز شرایط مدیریت تولید",
	}
	created := call(t, h, http.MethodPost, "/api/v1/career/records", hr, map[string]any{
		"kind": "appointments", "title": "انتصاب آزمون مدیر تولید", "employeeId": "EMP-009",
		"fields": fields,
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("ثبت ابلاغ باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var rec contracts.Record
	decodeInto(t, created, &rec)
	submitRecord(t, h, hr, rec.ID)
	// این درخواست چهار سطح دارد: مدیر واحد، مدیر منابع انسانی، کمیته، مدیرعامل.
	for _, token := range []string{lm, hr, hr, admin} {
		if code := signLevel(t, h, token, rec.ID); code != http.StatusOK {
			t.Fatalf("امضای سطح باید ۲۰۰ بدهد، %d داد", code)
		}
	}
	if code := call(t, h, http.MethodPost, "/api/v1/career/appointment-execute", hr,
		map[string]any{"id": rec.ID}).Code; code != http.StatusOK {
		t.Fatalf("اجرای ابلاغ باید ۲۰۰ بدهد، %d داد", code)
	}

	after := call(t, h, http.MethodGet, "/api/v1/orgstructure/records?kind=assignments&size=200", hr, nil)
	var pageAfter contracts.RecordPage
	decodeInto(t, after, &pageAfter)
	activeAfter := []string{}
	closed := false
	for _, item := range pageAfter.Items {
		if item.EmployeeID != "EMP-009" {
			continue
		}
		if item.Status == "active" {
			activeAfter = append(activeAfter, item.ID)
		}
		for _, id := range activeBefore {
			if item.ID == id && item.Status == "ended" {
				closed = true
			}
		}
	}
	if len(activeAfter) != 1 {
		t.Fatalf("کارمند باید پس از ابلاغ تازه فقط یک تصدی فعال داشته باشد: %v", activeAfter)
	}
	if !closed {
		t.Fatalf("تصدی پیشین EMP-009 باید پایانیافته شود: %v", activeBefore)
	}
}
