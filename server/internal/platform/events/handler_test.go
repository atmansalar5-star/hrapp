// آزمون مسیرهای مدیریتی صندوق: دسترسی، فهرست و بازگرداندن به صف.

package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
)

// asUser درخواست را با نقش داده‌شده اجرا می‌کند.
func asUser(h http.HandlerFunc, role contracts.Role, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	user := contracts.User{Username: "u", Role: role}
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{User: user}))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// TestAdminRoutesRequireAdminRole نقش غیرمدیر را رد می‌کند.
func TestAdminRoutesRequireAdminRole(t *testing.T) {
	bus := NewBus(NewMemoryStore(0), Config{})
	cases := map[string]http.HandlerFunc{
		"config": ConfigHandler(bus),
		"outbox": OutboxHandler(bus),
		"retry":  RetryHandler(bus),
	}
	for name, h := range cases {
		rec := asUser(h, contracts.RoleHRManager, "/api/v1/events/outbox")
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: نقش غیرمدیر کد %d گرفت، انتظار ۴۰۳", name, rec.Code)
		}
	}

	// بدون نشست: ۴۰۱
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/config", nil)
	rec := httptest.NewRecorder()
	ConfigHandler(bus)(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("بدون نشست کد %d گرفت، انتظار ۴۰۱", rec.Code)
	}
}

// TestConfigAndOutboxForAdmin وضعیت و فهرست را برای مدیر می‌سنجد.
func TestConfigAndOutboxForAdmin(t *testing.T) {
	store := NewMemoryStore(0)
	bus := NewBus(store, Config{URL: "http://127.0.0.1:9/hook", Secret: "k", MaxAttempts: 3})
	bus.Publish(t.Context(), Event{Name: "employee.updated", Subject: "EMP-011", Status: 200})

	rec := asUser(ConfigHandler(bus), contracts.RoleAdmin, "/api/v1/events/config")
	if rec.Code != http.StatusOK {
		t.Fatalf("config کد %d داد", rec.Code)
	}
	var cfg map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("پاسخ config نامعتبر: %v", err)
	}
	if cfg["enabled"] != true || cfg["secretSet"] != true {
		t.Errorf("وضعیت ارسال درست گزارش نشد: %+v", cfg)
	}
	counts, _ := cfg["counts"].(map[string]any)
	if counts["pending"] != float64(1) {
		t.Errorf("شمار در انتظار باید ۱ باشد: %+v", counts)
	}

	rec = asUser(OutboxHandler(bus), contracts.RoleAdmin, "/api/v1/events/outbox")
	if rec.Code != http.StatusOK {
		t.Fatalf("outbox کد %d داد", rec.Code)
	}
	var page struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("پاسخ outbox نامعتبر: %v", err)
	}
	if page.Total != 1 || page.Items[0]["name"] != "employee.updated" {
		t.Fatalf("فهرست رخداد درست نیست: %+v", page)
	}
	if page.Items[0]["deliveryStatus"] != StatusPending {
		t.Errorf("وضعیت تحویل %v است، انتظار pending", page.Items[0]["deliveryStatus"])
	}
}

// TestRetryRequeuesFailed ردیف ناموفق را به صف برمی‌گرداند.
func TestRetryRequeuesFailed(t *testing.T) {
	store := NewMemoryStore(0)
	bus := NewBus(store, Config{URL: "http://127.0.0.1:9/hook", Secret: "k"})
	bus.Publish(t.Context(), Event{Name: "payroll.action", Subject: "PAY-1"})

	rows, err := store.Recent(t.Context(), StatusPending, 1)
	if err != nil || len(rows) != 1 {
		t.Fatalf("ردیف تازه پیدا نشد: %v", err)
	}
	if err := store.MarkAttempt(t.Context(), rows[0].ID, 3, time.Now(), "خطا", StatusFailed); err != nil {
		t.Fatalf("ثبت ناموفق: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/outbox/1/retry", nil)
	req.SetPathValue("id", "1")
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{
		User: contracts.User{Username: "admin", Role: contracts.RoleAdmin},
	}))
	rec := httptest.NewRecorder()
	RetryHandler(bus)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("بازگرداندن به صف کد %d داد: %s", rec.Code, rec.Body.String())
	}

	pending, err := store.Recent(t.Context(), StatusPending, 5)
	if err != nil || len(pending) != 1 || pending[0].Attempts != 0 {
		t.Fatalf("ردیف باید با شمار صفر به صف برگشته باشد: %+v (%v)", pending, err)
	}
}

// TestRetryUnknownID پاسخ ۴۰۴ برای شناسهٔ ناشناخته را می‌سنجد.
func TestRetryUnknownID(t *testing.T) {
	bus := NewBus(NewMemoryStore(0), Config{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/outbox/999/retry", nil)
	req.SetPathValue("id", "999")
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{
		User: contracts.User{Username: "admin", Role: contracts.RoleAdmin},
	}))
	rec := httptest.NewRecorder()
	RetryHandler(bus)(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("شناسهٔ ناشناخته کد %d داد، انتظار ۴۰۴", rec.Code)
	}
}