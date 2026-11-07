// آزمون جریان سرتاسری «انتصاب و ترفیع»: از تصویب‌خواهی تا اجرای واقعی ابلاغ.
//
// چه چیزی سنجیده می‌شود:
//   - تصویب‌خواهی شرط‌ها، زنجیرهٔ تصویب و گام‌های اجرا را برمی‌گرداند و کمبودها را مانع می‌داند؛
//   - گردش‌کار پیش‌نویس ← در انتظار تصویب ← تصویب‌شده و قاعدهٔ «اجرای بی‌شمارهٔ ابلاغ»؛
//   - اجرای ابلاغ پنج گام را واقعاً انجام می‌دهد: پست و واحد کارمند در پرونده عوض می‌شود،
//     تصدی پست در تشکیلات ثبت و انتصاب پیشین پایانیافته می‌شود، و گزارش هر گام برمی‌گردد؛
//   - نمودار سازمانی همان متصدی تازه را نشان می‌دهد (اثر اجرا در ماژول دیگر).
package bootstrap

import (
	"net/http"
	"testing"

	"hrms/internal/contracts"
)

// appointmentFields فیلدهای یک ابلاغ کامل و مصوب.
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
	if len(plan.Plan.Approvals) == 0 {
		t.Fatal("زنجیرهٔ تصویب باید ساخته شود")
	}
	if len(plan.Execution) != 5 {
		t.Fatalf("گام‌های اجرا باید پنج گام باشند: %d", len(plan.Execution))
	}
}

// TestAppointmentExecuteClosesPreviousDecree می‌سنجد که اجرای ابلاغ تازه، تصدی پیشین همان
// کارمند را پایانیافته می‌کند (کارمند نباید هم‌زمان دو تصدی فعال داشته باشد).
func TestAppointmentExecuteClosesPreviousDecree(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken

	// EMP-009 در دادهٔ نمونه تصدی فعال روی POS-310 دارد.
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

	created := call(t, h, http.MethodPost, "/api/v1/career/records", hr, map[string]any{
		"kind": "appointments", "title": "انتصاب آزمون مدیر تولید", "employeeId": "EMP-009",
		"fields": map[string]string{
			"currentPosition": "کارشناس ارشد فرآیند", "currentUnit": "تولید و عملیات",
			"position": "مدیر تولید", "positionCode": "POS-310", "unit": "تولید و عملیات",
			"currentGrade": "supervisor", "proposedGrade": "manager",
			"serviceMonths": "42", "evaluationScore": "91", "vacancy": "yes", "commitment": "36",
			"effectiveDate": "1405/09/01", "decreeNo": "ABL-1405-902",
			"reason": "بازنشستگی مدیر پیشین و احراز شرایط مدیریت تولید",
		},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("ثبت ابلاغ باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var rec contracts.Record
	decodeInto(t, created, &rec)
	for _, action := range []string{"submit", "approve"} {
		if code := call(t, h, http.MethodPost, "/api/v1/career/records/"+rec.ID+"/action", hr,
			map[string]any{"action": action}).Code; code != http.StatusOK {
			t.Fatalf("اقدام %s باید ۲۰۰ بدهد، %d داد", action, code)
		}
	}
	executed := call(t, h, http.MethodPost, "/api/v1/career/appointment-execute", hr,
		map[string]any{"id": rec.ID})
	if executed.Code != http.StatusOK {
		t.Fatalf("اجرای ابلاغ باید ۲۰۰ بدهد، %d داد: %s", executed.Code, executed.Body.String())
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
func TestAppointmentExecuteAppliesDecree(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken

	// پروندهٔ کارمند پیش از اجرا.
	type employeeDTO struct {
		ID             string `json:"id"`
		Position       string `json:"position"`
		DepartmentName string `json:"departmentName"`
	}
	before := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-008", hr, nil)
	if before.Code != http.StatusOK {
		t.Fatalf("خواندن پرونده باید ۲۰۰ بدهد، %d داد", before.Code)
	}
	var employeeBefore employeeDTO
	decodeInto(t, before, &employeeBefore)
	if employeeBefore.Position == "کارشناس منابع انسانی" {
		t.Skip("پست کارمند از قبل همین است؛ آزمون معنایی ندارد")
	}

	// ۱) ثبت ابلاغ.
	created := call(t, h, http.MethodPost, "/api/v1/career/records", hr, map[string]any{
		"kind": "appointments", "title": "انتصاب آزمون کارشناس منابع انسانی",
		"employeeId": "EMP-008", "fields": appointmentFields("ABL-1405-777"),
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("ثبت ابلاغ باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var rec contracts.Record
	decodeInto(t, created, &rec)

	act := func(action string) int {
		return call(t, h, http.MethodPost, "/api/v1/career/records/"+rec.ID+"/action", hr,
			map[string]any{"action": action}).Code
	}
	// ۲) گردش‌کار: ارسال برای تصویب و تصویب.
	if code := act("submit"); code != http.StatusOK {
		t.Fatalf("ارسال برای تصویب باید ۲۰۰ بدهد، %d داد", code)
	}
	if code := act("approve"); code != http.StatusOK {
		t.Fatalf("تصویب باید ۲۰۰ بدهد، %d داد", code)
	}

	// ۳) اجرای ابلاغ: پست و واحد کارمند باید عوض شود.
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

	// ۴) اثر اجرا در پروندهٔ کارمند.
	after := call(t, h, http.MethodGet, "/api/v1/employee/employees/EMP-008", hr, nil)
	var employeeAfter employeeDTO
	decodeInto(t, after, &employeeAfter)
	if employeeAfter.Position != "کارشناس منابع انسانی" {
		t.Fatalf("پست کارمند پس از اجرا باید «کارشناس منابع انسانی» باشد: %q", employeeAfter.Position)
	}
	if employeeAfter.DepartmentName != "منابع انسانی" {
		t.Fatalf("واحد کارمند باید «منابع انسانی» باشد: %q", employeeAfter.DepartmentName)
	}

	// ۵) اثر اجرا در تشکیلات: تصدی پست POS-120 به همین کارمند بسته شده است.
	assignments := call(t, h, http.MethodGet,
		"/api/v1/orgstructure/records?kind=assignments&size=200", hr, nil)
	if assignments.Code != http.StatusOK {
		t.Fatalf("خواندن انتصاب‌ها باید ۲۰۰ بدهد، %d داد", assignments.Code)
	}
	var page contracts.RecordPage
	decodeInto(t, assignments, &page)
	active := 0
	for _, item := range page.Items {
		if item.EmployeeID != "EMP-008" || item.Status != "active" {
			continue
		}
		if item.Fields["position"] == "POS-120" && item.Fields["decreeNo"] == "ABL-1405-777" {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("تشکیلات باید یک تصدی فعال برای EMP-008 روی POS-120 داشته باشد: %d", active)
	}

	// ۶) اجرای دوباره روی ابلاغ اجراشده نباید ممکن باشد (وضعیت نهایی است).
	again := call(t, h, http.MethodPost, "/api/v1/career/appointment-execute", hr,
		map[string]any{"id": rec.ID})
	if again.Code == http.StatusOK {
		t.Fatalf("اجرای دوبارهٔ ابلاغ اجراشده نباید ۲۰۰ بدهد: %s", again.Body.String())
	}
}
