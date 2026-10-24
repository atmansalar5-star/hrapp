// آزمون جریان سرتاسری «shift»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار نوبت‌کاری.

package bootstrap

import (
	"net/http"
	"testing"

	"hrms/internal/contracts"
)

// TestShiftSchedulingFlow زیرکار ۱.۱ نوبت‌کاری را از راه API می‌سنجد: سه نوع رکورد
// تازه باید با پیشوند یکتا در شناسهٔ ماژول اعلام شوند، الگوی نامعتبر با قاعدهٔ دامنه
// ۴۲۲ بگیرد، گردش‌کار تخصیص از پیش‌نویس تا خاتمه طی شود و تعویض بی‌اثر همان موقع رد
// شود — یعنی قواعد دامنه نه فقط در واحد، بلکه روی زنجیرهٔ کامل HTTP+موتور هم بگیرند.
func TestShiftSchedulingFlow(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	lm := login(t, h, "line.mgr", "demo").AccessToken
	emp := login(t, h, "employee", "demo").AccessToken

	// ---------- شناسه: سه نوع رکورد با پیشوند یکتا ----------
	rec := call(t, h, http.MethodGet, "/api/v1/attendance/schema", hr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("شناسهٔ ماژول حضور باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var schema contracts.RecordSchema
	decodeInto(t, rec, &schema)
	wantKinds := map[string]string{"shiftPatterns": "SHP", "shiftAssignments": "SHA", "shiftSwaps": "SSW"}
	for key, prefix := range wantKinds {
		seen := false
		for _, k := range schema.Kinds {
			if k.Key != key {
				continue
			}
			seen = true
			if k.IDPrefix != prefix {
				t.Errorf("پیشوند شناسهٔ «%s» باید «%s» باشد، «%s» است", key, prefix, k.IDPrefix)
			}
		}
		if !seen {
			t.Fatalf("نوع رکورد «%s» در شناسهٔ ماژول حضور نیست", key)
		}
	}

	// ---------- قاعدهٔ الگو: چرخهٔ نامعتبر باید ۴۲۲ بگیرد ----------
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftPatterns", "title": "الگوی نامعتبر",
		"fields": map[string]string{"rotation": "صبح,دوچرخه"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("چرخهٔ نامعتبر باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// ---------- الگوی معتبر و فعال‌سازی ----------
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftPatterns", "title": "الگوی آزمون نوبت‌کاری",
		"fields": map[string]string{"rotation": "صبح,عصر,شب,استراحت", "minCoverage": "1"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ایجاد الگو باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var pattern contracts.Record
	decodeInto(t, rec, &pattern)
	if pattern.Status != "draft" {
		t.Fatalf("الگوی تازه باید پیش‌نویس باشد، «%s» بود", pattern.Status)
	}
	// شناسهٔ رکورد باید همان پیشوند اعلامی را داشته باشد.
	if len(pattern.ID) < 5 || pattern.ID[:4] != "SHP-" {
		t.Fatalf("شناسهٔ الگو باید با SHP- آغاز شود، «%s» است", pattern.ID)
	}

	act := func(token, id, action string) int {
		return call(t, h, http.MethodPost, "/api/v1/attendance/records/"+id+"/action", token,
			map[string]any{"action": action}).Code
	}

	// مرحلهٔ چرخه ویژهٔ منابع انسانی است؛ مدیر واحد نباید الگو را بازنشسته کند.
	if code := act(hr, pattern.ID, "activate"); code != http.StatusOK {
		t.Fatalf("فعال‌سازی الگو باید ۲۰۰ بدهد، %d داد", code)
	}
	if code := act(lm, pattern.ID, "retire"); code != http.StatusUnprocessableEntity {
		t.Fatalf("بازنشسته‌سازی الگو توسط مدیر واحد باید ۴۲۲ بدهد، %d داد", code)
	}

	// ---------- قاعدهٔ تخصیص: بازهٔ معکوس باید ۴۲۲ بگیرد ----------
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftAssignments", "title": "تخصیص معکوس", "employeeId": "EMP-008",
		"fields": map[string]string{"pattern": "الگوی آزمون نوبت‌کاری",
			"startDate": jdate(t, 10), "endDate": jdate(t, -10)},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تخصیص با بازهٔ معکوس باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// ---------- تخصیص معتبر و گردش‌کارش تا خاتمه ----------
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftAssignments", "title": "تخصیص آزمون نوبت", "employeeId": "EMP-008",
		"fields": map[string]string{"pattern": "الگوی آزمون نوبت‌کاری",
			"startDate": jdate(t, 1), "cycleOffset": "2"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ایجاد تخصیص باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var assignment contracts.Record
	decodeInto(t, rec, &assignment)

	// کارمند روی حضور اجازهٔ اقدام ندارد (سطح ماژول) → ۴۰۳.
	if code := act(emp, assignment.ID, "approve"); code != http.StatusForbidden {
		t.Fatalf("اقدام کارمند باید ۴۰۳ بدهد، %d داد", code)
	}
	// تأیید پیش از ارسال ← انتقال نامجوز → ۴۲۲.
	if code := act(lm, assignment.ID, "approve"); code != http.StatusUnprocessableEntity {
		t.Fatalf("تأیید پیش از ارسال باید ۴۲۲ بدهد، %d داد", code)
	}
	if code := act(hr, assignment.ID, "submit"); code != http.StatusOK {
		t.Fatalf("ارسال تخصیص باید ۲۰۰ بدهد، %d داد", code)
	}
	if code := act(lm, assignment.ID, "approve"); code != http.StatusOK {
		t.Fatalf("تأیید تخصیص توسط مدیر واحد باید ۲۰۰ بدهد، %d داد", code)
	}
	if code := act(hr, assignment.ID, "end"); code != http.StatusOK {
		t.Fatalf("خاتمهٔ تخصیص باید ۲۰۰ بدهد، %d داد", code)
	}
	got := call(t, h, http.MethodGet, "/api/v1/attendance/records/"+assignment.ID, hr, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("خواندن تخصیص باید ۲۰۰ بدهد، %d داد", got.Code)
	}
	decodeInto(t, got, &assignment)
	if assignment.Status != "ended" {
		t.Fatalf("وضعیت پایانی تخصیص باید ended باشد، «%s» شد", assignment.Status)
	}

	// ---------- قواعد تعویض: همتای خودی و تعویض هم‌شیفت ----------
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftSwaps", "title": "تعویض خودی", "employeeId": "EMP-005",
		"fields": map[string]string{"swapDate": jdate(t, 3), "partnerEmployee": "EMP-005"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تعویض با همتای خودی باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftSwaps", "title": "تعویض هم‌شیفت", "employeeId": "EMP-005",
		"fields": map[string]string{"swapDate": jdate(t, 3), "partnerEmployee": "EMP-008",
			"myShift": "صبح", "partnerShift": "صبح"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تعویض هم‌شیفت باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// ---------- تعویض معتبر: ارسال، کارتابل، تأیید، و بسته‌شدن گردش ----------
	rec = call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "shiftSwaps", "title": "تعویض آزمون نوبت", "employeeId": "EMP-005",
		"fields": map[string]string{"swapDate": jdate(t, 3), "partnerEmployee": "EMP-008",
			"myShift": "صبح", "partnerShift": "عصر", "reason": "علت خانوادگی"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ایجاد تعویض باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var swap contracts.Record
	decodeInto(t, rec, &swap)
	if swap.Status != "draft" {
		t.Fatalf("تعویض تازه باید پیش‌نویس باشد، «%s» بود", swap.Status)
	}
	if code := act(hr, swap.ID, "submit"); code != http.StatusOK {
		t.Fatalf("ارسال تعویض باید ۲۰۰ بدهد، %d داد", code)
	}

	// کارتابل مدیر واحد باید کارِ تأیید همین تعویض را داشته باشد (سیم‌کشی inbox با
	// نوع رکورد تازه؛ بدون این، دکمهٔ تأیید در رابط هرگز دیده نمی‌شود).
	inbox := call(t, h, http.MethodGet, "/api/v1/workflow/inbox", lm, nil)
	if inbox.Code != http.StatusOK {
		t.Fatalf("کارتابل باید ۲۰۰ بدهد، %d داد: %s", inbox.Code, inbox.Body.String())
	}
	var tasks contracts.WorkflowTaskPage
	decodeInto(t, inbox, &tasks)
	foundApprove := false
	for _, tk := range tasks.Items {
		if tk.RecordID == swap.ID && tk.Action == "approve" {
			foundApprove = true
		}
	}
	if !foundApprove {
		t.Fatalf("کارتابل مدیر واحد باید کار approve روی %s را داشته باشد (%d کار)", swap.ID, tasks.Total)
	}

	if code := act(lm, swap.ID, "approve"); code != http.StatusOK {
		t.Fatalf("تأیید تعویض باید ۲۰۰ بدهد، %d داد", code)
	}
	// تأیید دوباره از وضعیت پایانی ← انتقال نامجوز → ۴۲۲.
	if code := act(lm, swap.ID, "approve"); code != http.StatusUnprocessableEntity {
		t.Fatalf("تأیید مجدد باید ۴۲۲ بدهد، %d داد", code)
	}
	got = call(t, h, http.MethodGet, "/api/v1/attendance/records/"+swap.ID, hr, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("خواندن تعویض باید ۲۰۰ بدهد، %d داد", got.Code)
	}
	decodeInto(t, got, &swap)
	if swap.Status != "approved" {
		t.Fatalf("وضعیت پایانی تعویض باید approved باشد، «%s» شد", swap.Status)
	}

	// ---------- دادهٔ نمونه: هر سه نوع بذر باید از راه فهرست دیده شوند ----------
	for kind, prefix := range map[string]string{"shiftPatterns": "SHP", "shiftAssignments": "SHA", "shiftSwaps": "SSW"} {
		list := call(t, h, http.MethodGet, "/api/v1/attendance/records?kind="+kind+"&size=50", hr, nil)
		if list.Code != http.StatusOK {
			t.Fatalf("فهرست «%s» باید ۲۰۰ بدهد، %d داد", kind, list.Code)
		}
		var page contracts.RecordPage
		decodeInto(t, list, &page)
		if page.Total < 1 {
			t.Fatalf("نوع «%s» باید دست‌کم بذر داشته باشد", kind)
		}
		for _, item := range page.Items {
			if len(item.ID) < len(prefix)+1 || item.ID[:len(prefix)+1] != prefix+"-" {
				t.Errorf("شناسهٔ «%s» باید با «%s-» آغاز شود، «%s» است", kind, prefix, item.ID)
			}
		}
	}
}
