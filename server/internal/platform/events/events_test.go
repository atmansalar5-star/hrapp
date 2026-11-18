// آزمون‌های صندوق خروجی رخداد: امضا، میان‌افزار، تحویل و تلاش مجدد.

package events

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
)

// fixedClock ساعت ثابت برای آزمون.
func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

// TestSignAndVerify امضای HMAC و رد شدن بدنهٔ دست‌کاری‌شده را می‌سنجد.
func TestSignAndVerify(t *testing.T) {
	at := time.Unix(1_700_000_000, 0)
	body := []byte(`{"name":"employee.updated"}`)
	sig := Sign("secret-1", at, body)
	if sig == "" {
		t.Fatal("امضا خالی است")
	}
	if !Verify("secret-1", at, body, sig) {
		t.Fatal("امضای درست باید پذیرفته شود")
	}
	if Verify("secret-2", at, body, sig) {
		t.Error("امضا با کلید دیگر نباید پذیرفته شود")
	}
	if Verify("secret-1", at.Add(time.Minute), body, sig) {
		t.Error("امضا با مُهر زمانی دیگر نباید پذیرفته شود")
	}
	if Verify("secret-1", at, []byte(`{"name":"employee.deleted"}`), sig) {
		t.Error("امضا با بدنهٔ دیگر نباید پذیرفته شود")
	}
}

// TestNameFor راهبرد نام‌گذاری رخداد را می‌سنجد.
func TestNameFor(t *testing.T) {
	cases := []struct {
		method, path, name, subject string
	}{
		{"POST", "/api/v1/employee/employees", "employee.created", ""},
		{"PATCH", "/api/v1/employee/employees/EMP-011", "employee.updated", "EMP-011"},
		{"DELETE", "/api/v1/employee/employees/EMP-011", "employee.deleted", "EMP-011"},
		{"POST", "/api/v1/payroll/records/PAY-1404-07/action", "payroll.action", "PAY-1404-07"},
		{"POST", "/api/v1/compensation/simulate", "compensation.created", ""},
		{"GET", "/api/v1/employee/employees/EMP-011", "", ""},
	}
	for _, c := range cases {
		name, _, subject := nameFor(c.method, c.path)
		if name != c.name || subject != c.subject {
			t.Errorf("%s %s → (%q,%q)، انتظار (%q,%q)", c.method, c.path, name, subject, c.name, c.subject)
		}
	}
}

// TestMiddlewareEmitsOnlySuccessfulMutations پوشش و پرهیزهای میان‌افزار را می‌سنجد.
func TestMiddlewareEmitsOnlySuccessfulMutations(t *testing.T) {
	store := NewMemoryStore(0)
	bus := NewBus(store, Config{Now: fixedClock(time.Unix(1_700_000_000, 0))})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/employee/employees", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"EMP-999"}`))
	})
	mux.HandleFunc("PATCH /api/v1/employee/employees/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/v1/employee/broken", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("GET /api/v1/employee/employees/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/v1/events/outbox/{id}/retry", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := authMiddlewareForTest(Middleware(bus, nil)(mux))

	calls := []struct {
		method, path string
		header       string
	}{
		{"POST", "/api/v1/employee/employees", ""},
		{"PATCH", "/api/v1/employee/employees/EMP-011", ""},
		{"POST", "/api/v1/employee/broken", ""},          // خطای سرور → رخداد نیست
		{"GET", "/api/v1/employee/employees/EMP-011", ""}, // خواندن → رخداد نیست
		{"POST", "/api/v1/events/outbox/1/retry", ""},     // خود صندوق → حلقه
		{"POST", "/api/v1/auth/login", ""},                // نشست → رخداد نیست
		{"PATCH", "/api/v1/employee/employees/EMP-011", "n8n"}, // درخواست خود مصرف‌کننده
	}
	for _, c := range calls {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader("{}"))
		if c.header != "" {
			req.Header.Set(HeaderOrigin, c.header)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	rows, err := store.Recent(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("خواندن صندوق: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("انتظار ۲ رخداد بود، %d ثبت شد", len(rows))
	}
	byName := map[string]Delivery{}
	for _, row := range rows {
		byName[row.Event.Name] = row
	}
	created, ok := byName["employee.created"]
	if !ok {
		t.Fatalf("رخداد employee.created ثبت نشد: %+v", byName)
	}
	if created.Event.Actor != "hr.manager" || created.Event.Role != string(contracts.RoleHRManager) {
		t.Errorf("کنشگر/نقش رخداد درست نیست: %q/%q", created.Event.Actor, created.Event.Role)
	}
	updated, ok := byName["employee.updated"]
	if !ok {
		t.Fatal("رخداد employee.updated ثبت نشد")
	}
	if updated.Event.Subject != "EMP-011" {
		t.Errorf("شناسهٔ موضوع %q است، انتظار EMP-011", updated.Event.Subject)
	}
	if updated.Event.Method != http.MethodPatch || updated.Event.Status != http.StatusOK {
		t.Errorf("دادهٔ درخواست در رخداد درست نیست: %s/%d", updated.Event.Method, updated.Event.Status)
	}
}

// authMiddlewareForTest کاربر ثابت را (بی‌نیاز از توکن) در زمینه می‌گذارد.
func authMiddlewareForTest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contracts.User{Username: "hr.manager", Role: contracts.RoleHRManager, EmployeeID: "EMP-001"}
		next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), auth.Principal{User: user})))
	})
}

// TestDispatcherDeliversAndRetries تحویل با امضا و تلاش مجدد پس از خطای موقت را می‌سنجد.
func TestDispatcherDeliversAndRetries(t *testing.T) {
	var (
		mu       sync.Mutex
		attempts int
		lastBody []byte
		lastSig  string
		lastTS   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		attempts++
		lastBody, _ = io.ReadAll(r.Body)
		lastSig = r.Header.Get(HeaderSignature)
		lastTS = r.Header.Get(HeaderTimestamp)
		if attempts == 1 {
			w.WriteHeader(http.StatusBadGateway) // خطای موقت مقصد
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	now := time.Unix(1_700_000_000, 0)
	clock := now
	store := NewMemoryStore(0)
	bus := NewBus(store, Config{
		URL: srv.URL, Secret: "s3cret", MaxAttempts: 3, Batch: 10,
		Now: func() time.Time { return clock },
	})
	bus.Publish(context.Background(), Event{
		Name: "employee.updated", Actor: "hr.manager", Module: "employee",
		Method: "PATCH", Path: "/api/v1/employee/employees/EMP-011",
		Status: 200, Subject: "EMP-011", Payload: map[string]any{"path": "/x"},
	})

	if sent, failed := bus.DrainOnce(context.Background()); sent != 0 || failed != 1 {
		t.Fatalf("دور اول: انتظار (۰ ارسال، ۱ خطا) بود، شد (%d,%d)", sent, failed)
	}
	rows, _ := store.Recent(context.Background(), StatusPending, 0)
	if len(rows) != 1 || rows[0].Attempts != 1 {
		t.Fatalf("پس از خطا باید یک ردیف با یک تلاش بماند: %+v", rows)
	}
	// زمان تلاش بعدی در آینده است، پس دور فوری نباید چیزی بفرستد.
	if sent, _ := bus.DrainOnce(context.Background()); sent != 0 {
		t.Fatalf("پیش از سررسید نباید ارسال شود، %d ارسال شد", sent)
	}
	// زمان را جلو می‌بریم تا تلاش بعدی سر برسد.
	clock = now.Add(10 * time.Second)
	if sent, failed := bus.DrainOnce(context.Background()); sent != 1 || failed != 0 {
		t.Fatalf("دور دوم: انتظار (۱ ارسال، ۰ خطا) بود، شد (%d,%d)", sent, failed)
	}
	rows, _ = store.Recent(context.Background(), StatusDelivered, 0)
	if len(rows) != 1 {
		t.Fatalf("رخداد باید تحویل‌شده باشد: %+v", rows)
	}

	mu.Lock()
	defer mu.Unlock()
	if attempts != 2 {
		t.Errorf("انتظار ۲ درخواست بود، %d رسید", attempts)
	}
	if lastSig == "" || !strings.HasPrefix(lastSig, "sha256=") {
		t.Errorf("سرآیند امضا درست نیست: %q", lastSig)
	}
	tsSec, err := strconv.ParseInt(lastTS, 10, 64)
	if err != nil {
		t.Fatalf("مُهر زمانی نامعتبر: %q", lastTS)
	}
	if !Verify("s3cret", time.Unix(tsSec, 0), lastBody, strings.TrimPrefix(lastSig, "sha256=")) {
		t.Error("امضای رسیده با کلید و مُهر زمانی معتبر بازبینی نشد")
	}
	var env envelope
	if err := json.Unmarshal(lastBody, &env); err != nil {
		t.Fatalf("بدنهٔ JSON نامعتبر: %v", err)
	}
	if env.Name != "employee.updated" || env.Subject != "EMP-011" || env.Attempt != 2 {
		t.Errorf("بدنهٔ رخداد نادرست است: %+v", env)
	}
}

// TestDispatcherGivesUpAfterMaxAttempts ناموفق‌ماندگار شدن و بازگرداندن به صف را می‌سنجد.
func TestDispatcherGivesUpAfterMaxAttempts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	clock := time.Unix(1_700_000_000, 0)
	store := NewMemoryStore(0)
	bus := NewBus(store, Config{
		URL: srv.URL, Secret: "k", MaxAttempts: 2, Batch: 10,
		Now: func() time.Time { return clock },
	})
	bus.Publish(context.Background(), Event{Name: "employee.deleted"})

	bus.DrainOnce(context.Background())
	clock = clock.Add(10 * time.Second)
	bus.DrainOnce(context.Background())

	rows, _ := store.Recent(context.Background(), StatusFailed, 0)
	if len(rows) != 1 {
		t.Fatalf("پس از دو تلاش باید وضعیت failed باشد: %+v", rows)
	}
	if rows[0].LastError == "" {
		t.Error("پیام خطای آخر باید ثبت شود")
	}
	// ردیف ناموفق دیگر در صف ارسال نیست.
	if sent, _ := bus.DrainOnce(context.Background()); sent != 0 {
		t.Errorf("ردیف ناموفق نباید خودکار دوباره فرستاده شود، %d ارسال شد", sent)
	}
	// بازگرداندن دستی به صف.
	if err := store.Requeue(context.Background(), rows[0].ID, clock); err != nil {
		t.Fatalf("بازگرداندن به صف: %v", err)
	}
	pending, _ := store.Recent(context.Background(), StatusPending, 0)
	if len(pending) != 1 || pending[0].Attempts != 0 {
		t.Fatalf("ردیف باید با شمار تلاش صفر به صف برگردد: %+v", pending)
	}
}

// TestBusWithoutStoreIsNoop نبودِ انبار را می‌سنجد: هیچ خطا و هیچ ترسی.
func TestBusWithoutStoreIsNoop(t *testing.T) {
	bus := NewBus(nil, Config{})
	if bus.Enabled() {
		t.Error("بدون نشانی نباید ارسال روشن باشد")
	}
	bus.Publish(context.Background(), Event{Name: "x"})
	bus.Skip()
	counts, err := bus.Stats(context.Background())
	if err != nil {
		t.Fatalf("گزارش وضعیت: %v", err)
	}
	if counts[StatusPending] != 0 {
		t.Errorf("شمارش صندوق خالی باید صفر باشد: %+v", counts)
	}
	if sent, failed := bus.DrainOnce(context.Background()); sent != 0 || failed != 0 {
		t.Error("ارسال بدون انبار باید بی‌اثر باشد")
	}
}