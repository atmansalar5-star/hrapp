// آزمون انبار Postgres صندوق خروجی.
//
// با متغیر HRMS_TEST_DATABASE_URL اجرا می‌شود (همان الگوی آزمون دودی Postgres)؛ بدون
// آن skip می‌شود تا «go test ./...» محلی به پایگاه‌دادهٔ کسی دست نزند.

package events

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	platformdb "hrms/internal/platform/db"
)

// postgresTestStore انبار آزمون را روی پایگاه‌دادهٔ تست می‌سازد و جدول را اگر نبود
// می‌سازد (همان DDL مهاجرت 003).
func postgresTestStore(t *testing.T) *PostgresStore {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("HRMS_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("آزمون انبار Postgres اجرا نشد: HRMS_TEST_DATABASE_URL تنظیم نشده است")
	}
	db, err := platformdb.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("اتصال پایگاه‌داده: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ddl, err := os.ReadFile("../../../migrations/003_event_outbox.sql")
	if err == nil {
		if _, err := db.ExecContext(context.Background(), string(ddl)); err != nil {
			t.Fatalf("اعمال مهاجرت صندوق: %v", err)
		}
	}
	return NewPostgresStore(db)
}

// TestPostgresStoreRoundTrip چرخهٔ ثبت، خواندن سررسید، تحویل و بازگرداندن به صف را
// روی پایگاه‌دادهٔ واقعی می‌سنجد؛ ریشهٔ ایراد «status صفر» در همین چرخه بود.
func TestPostgresStoreRoundTrip(t *testing.T) {
	store := postgresTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	event := Event{
		ID: NewID(), Name: "employee.updated", OccurredAt: now,
		Actor: "hr.manager", Role: "hr_manager", Module: "employee",
		Method: "PATCH", Path: "/api/v1/employee/employees/EMP-011",
		Status: 200, Subject: "EMP-011", Payload: map[string]any{"durationMs": float64(12)},
	}
	if err := store.Append(ctx, event); err != nil {
		t.Fatalf("ثبت رخداد: %v", err)
	}
	// ثبت دوباره با همان شناسه بی‌اثر است.
	if err := store.Append(ctx, event); err != nil {
		t.Fatalf("ثبت تکراری باید بی‌اثر باشد: %v", err)
	}

	rows, err := store.Due(ctx, now.Add(time.Minute), 200)
	if err != nil {
		t.Fatalf("خواندن سررسیدها: %v", err)
	}
	var got *Delivery
	for i := range rows {
		if rows[i].Event.ID == event.ID {
			got = &rows[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("رخداد تازه در صف پیدا نشد")
	}
	if got.Event.Status != 200 {
		t.Errorf("کد پاسخ در رخداد %d است، انتظار ۲۰۰", got.Event.Status)
	}
	if got.Event.Subject != "EMP-011" || got.Event.Method != httpMethodPatch() {
		t.Errorf("دادهٔ درخواست نادرست خوانده شد: %+v", got.Event)
	}
	if got.Status() != StatusPending {
		t.Errorf("وضعیت ردیف تازه %q است، انتظار pending", got.Status())
	}
	if v, ok := got.Event.Payload["durationMs"]; !ok || v == nil {
		t.Errorf("دادهٔ کمکی رخداد بازیابی نشد: %+v", got.Event.Payload)
	}

	if err := store.MarkAttempt(ctx, got.ID, 1, now.Add(time.Hour), "خطای نمایشی", StatusFailed); err != nil {
		t.Fatalf("ثبت تلاش: %v", err)
	}
	failed, err := store.Recent(ctx, StatusFailed, 50)
	if err != nil {
		t.Fatalf("خواندن ناموفق‌ها: %v", err)
	}
	foundFailed := false
	for _, row := range failed {
		if row.Event.ID == event.ID {
			foundFailed = true
			if row.Attempts != 1 || row.LastError == "" {
				t.Errorf("دادهٔ تلاش ناموفق ناقص است: %+v", row)
			}
		}
	}
	if !foundFailed {
		t.Fatal("رخداد ناموفق در فهرست نیست")
	}

	if err := store.Requeue(ctx, got.ID, now); err != nil {
		t.Fatalf("بازگرداندن به صف: %v", err)
	}
	if err := store.MarkDelivered(ctx, got.ID, now); err != nil {
		t.Fatalf("ثبت تحویل: %v", err)
	}
	recent, err := store.Recent(ctx, StatusDelivered, 50)
	if err != nil {
		t.Fatalf("خواندن تحویل‌شده‌ها: %v", err)
	}
	found := false
	for _, row := range recent {
		if row.Event.ID == event.ID {
			found = true
			if row.DeliveredAt == nil {
				t.Error("زمان تحویل ثبت نشده است")
			}
			if row.Status() != StatusDelivered {
				t.Errorf("وضعیت %q است، انتظار delivered", row.Status())
			}
		}
	}
	if !found {
		t.Fatal("رخداد تحویل‌شده در فهرست نیست")
	}
}

// TestPostgresStoreCounts شمارش وضعیت‌ها را می‌سنجد.
func TestPostgresStoreCounts(t *testing.T) {
	store := postgresTestStore(t)
	ctx := context.Background()
	before, err := store.Counts(ctx)
	if err != nil {
		t.Fatalf("شمارش اولیه: %v", err)
	}
	event := Event{ID: NewID(), Name: "smoke.created", OccurredAt: time.Now().UTC(), Status: 201}
	if err := store.Append(ctx, event); err != nil {
		t.Fatalf("ثبت رخداد: %v", err)
	}
	after, err := store.Counts(ctx)
	if err != nil {
		t.Fatalf("شمارش دوم: %v", err)
	}
	if after[StatusPending] != before[StatusPending]+1 {
		t.Errorf("شمار pending از %d به %d رسید، انتظار +۱", before[StatusPending], after[StatusPending])
	}
}

// httpMethodPatch رشتهٔ روش PATCH را بدون import اضافه برمی‌گرداند.
func httpMethodPatch() string { return "PATCH" }

// اطمینان از اینکه انبار به اتصال زنده نیاز دارد (نگه‌داشتن import های ابزار).
var _ = sql.ErrNoRows