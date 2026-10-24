// آزمون جریان سرتاسری «workflow»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"testing"

	"hrms/internal/contracts"
)

// TestLeaveWorkflowChainAndInbox زنجیرهٔ دومرحله‌ای مرخصی را از راه API می‌سنجد: مدیر
// واحد مرحلهٔ اول را می‌بندد و منابع انسانی نهایی می‌کند؛ هر تلاش برای دور زدن مرحله رد
// می‌شود و کارتابل هر نقش، کارِ همان مرحله را نشان می‌دهد.
func TestLeaveWorkflowChainAndInbox(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	lm := login(t, h, "line.mgr", "demo").AccessToken

	rec := call(t, h, http.MethodPost, "/api/v1/attendance/records", hr, map[string]any{
		"kind": "leaveRequests", "title": "مرخصی آزمون گردش‌کار", "employeeId": "EMP-001",
		"fields": map[string]string{"leaveType": "استحقاقی", "fromDate": "1404/08/01", "toDate": "1404/08/02", "days": "2"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ایجاد درخواست مرخصی باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var created contracts.Record
	decodeInto(t, rec, &created)
	if created.Status != "draft" {
		t.Fatalf("وضعیت آغازین مرخصی باید draft باشد: %s", created.Status)
	}

	act := func(token, action string) int {
		return call(t, h, http.MethodPost, "/api/v1/attendance/records/"+created.ID+"/action", token,
			map[string]any{"action": action}).Code
	}

	// ارسال برای تأیید؛ سپس منابع انسانی نباید بتواند مرحلهٔ مدیر واحد را دور بزند.
	if code := act(hr, "submit"); code != http.StatusOK {
		t.Fatalf("ارسال برای تأیید باید ۲۰۰ بدهد، %d داد", code)
	}
	if code := act(hr, "endorse"); code != http.StatusUnprocessableEntity {
		t.Fatalf("منابع انسانی نباید مرحلهٔ مدیر واحد را دور بزند (انتظار ۴۲۲، گرفت %d)", code)
	}

	// کارتابل مدیر واحد باید کارِ «تأیید مدیر واحد» روی همین رکورد را داشته باشد.
	inbox := call(t, h, http.MethodGet, "/api/v1/workflow/inbox", lm, nil)
	if inbox.Code != http.StatusOK {
		t.Fatalf("کارتابل باید ۲۰۰ بدهد، %d داد: %s", inbox.Code, inbox.Body.String())
	}
	var tasks contracts.WorkflowTaskPage
	decodeInto(t, inbox, &tasks)
	foundEndorse := false
	for _, tk := range tasks.Items {
		if tk.RecordID == created.ID && tk.Action == "endorse" {
			foundEndorse = true
		}
	}
	if !foundEndorse {
		t.Fatalf("کارتابل مدیر واحد باید کار endorse روی %s را داشته باشد (%d کار)", created.ID, tasks.Total)
	}

	// تأیید مدیر واحد؛ سپس خودش نباید نهایی کند.
	if code := act(lm, "endorse"); code != http.StatusOK {
		t.Fatalf("تأیید مدیر واحد باید ۲۰۰ بدهد، %d داد", code)
	}
	if code := act(lm, "approve"); code != http.StatusUnprocessableEntity {
		t.Fatalf("مدیر واحد نباید تأیید نهایی کند (انتظار ۴۲۲، گرفت %d)", code)
	}

	// تأیید نهایی منابع انسانی و کنترل وضعیت پایانی.
	if code := act(hr, "approve"); code != http.StatusOK {
		t.Fatalf("تأیید نهایی منابع انسانی باید ۲۰۰ بدهد، %d داد", code)
	}
	got := call(t, h, http.MethodGet, "/api/v1/attendance/records/"+created.ID, hr, nil)
	if got.Code != http.StatusOK {
		t.Fatalf("خواندن رکورد باید ۲۰۰ بدهد، %d داد", got.Code)
	}
	var final contracts.Record
	decodeInto(t, got, &final)
	if final.Status != "approved" {
		t.Fatalf("وضعیت پایانی باید approved باشد، %s شد", final.Status)
	}
}
