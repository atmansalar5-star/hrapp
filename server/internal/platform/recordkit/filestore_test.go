// آزمون زیرساخت «recordkit» در پلتفرم سرور — «filestore_test».

package recordkit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// TestFileStorePersistsRecordsAcrossRestart پایه‌ای‌ترین وعدهٔ اجرای مستقل را می‌سنجد:
// رکورد پس از بسته شدن و باز شدن دوبارهٔ انبار همان است و شناسهٔ تازه هم تکرار نمی‌شود.
func TestFileStorePersistsRecordsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	ctx := context.Background()

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("ساخت انبار فایل ناموفق: %v", err)
	}
	id, err := store.NextID("bonusPayments", "BNP")
	if err != nil {
		t.Fatalf("ساخت شناسه ناموفق: %v", err)
	}
	if _, err := store.Save(ctx, Record{
		ID: id, Kind: "bonusPayments", Status: "draft",
		EmployeeID: "EMP-001", Fields: map[string]string{"amount": "1000000"},
	}); err != nil {
		t.Fatalf("ذخیرهٔ رکورد ناموفق: %v", err)
	}

	// اجرای دوباره: همان فایل، انبار تازه.
	reopened, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("باز کردن دوبارهٔ انبار ناموفق: %v", err)
	}
	got, ok, err := reopened.Get(ctx, id)
	if err != nil || !ok {
		t.Fatalf("رکورد پس از راه‌اندازی دوباره پیدا نشد (ok=%v، خطا=%v)", ok, err)
	}
	if got.Fields["amount"] != "1000000" {
		t.Fatalf("دادهٔ رکورد پس از راه‌اندازی دوباره عوض شده: %+v", got.Fields)
	}
	next, err := reopened.NextID("bonusPayments", "BNP")
	if err != nil {
		t.Fatalf("ساخت شناسهٔ دوم ناموفق: %v", err)
	}
	if next == id {
		t.Fatalf("شناسهٔ تازه باید ادامهٔ شمارنده باشد، دوباره %s ساخته شد", next)
	}
	if !strings.HasPrefix(next, "BNP-") {
		t.Fatalf("شناسهٔ تازه باید پیشوند BNP داشته باشد: %s", next)
	}
}

// TestFileStoreDeleteRemovesFromDisk می‌سنجد حذف هم ماندگار است.
func TestFileStoreDeleteRemovesFromDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	ctx := context.Background()
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("ساخت انبار ناموفق: %v", err)
	}
	if _, err := store.Save(ctx, Record{ID: "BNP-0001", Kind: "bonusPayments", Status: "draft"}); err != nil {
		t.Fatalf("ذخیره ناموفق: %v", err)
	}
	if err := store.Delete(ctx, "BNP-0001"); err != nil {
		t.Fatalf("حذف ناموفق: %v", err)
	}
	reopened, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("باز کردن دوباره ناموفق: %v", err)
	}
	if _, ok, _ := reopened.Get(ctx, "BNP-0001"); ok {
		t.Fatal("رکورد حذف‌شده نباید پس از راه‌اندازی دوباره برگردد")
	}
}

// TestFileStoreListFilters می‌سنجد فیلتر و صفحه‌بندی همان منطق انبار مشترک است و یک
// نسخهٔ دوم ندارد.
func TestFileStoreListFilters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	ctx := context.Background()
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("ساخت انبار ناموفق: %v", err)
	}
	rows := []Record{
		{ID: "BNP-0001", Kind: "bonusPayments", Status: "draft", EmployeeID: "EMP-001", Fields: map[string]string{"management": "kimia"}},
		{ID: "BNP-0002", Kind: "bonusPayments", Status: "paid", EmployeeID: "EMP-002", Fields: map[string]string{"management": "kimia"}},
		{ID: "BNP-0003", Kind: "bonusPayments", Status: "paid", EmployeeID: "EMP-003", Fields: map[string]string{"management": "bandar_imam"}},
	}
	for _, r := range rows {
		if _, err := store.Save(ctx, r); err != nil {
			t.Fatalf("ذخیرهٔ %s ناموفق: %v", r.ID, err)
		}
	}
	page, err := store.List(ctx, contracts.RecordFilter{Kind: "bonusPayments", Status: "paid"})
	if err != nil {
		t.Fatalf("فهرست ناموفق: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("فیلتر وضعیت باید ۲ رکورد بدهد، %d داد", page.Total)
	}
	scoped, err := store.List(ctx, contracts.RecordFilter{Kind: "bonusPayments", EmployeeID: "EMP-003"})
	if err != nil {
		t.Fatalf("فهرست محدود به کارمند ناموفق: %v", err)
	}
	if scoped.Total != 1 || scoped.Items[0].ID != "BNP-0003" {
		t.Fatalf("فیلتر کارمند نتیجهٔ نادرست داد: %+v", scoped.Items)
	}
}

// TestFileStoreRejectsCorruptFile می‌سنجد فایل خراب بی‌صدا با دادهٔ خالی بالا نمی‌آید؛
// کاربر باید بفهمد پشتیبان را بازگرداند، نه اینکه رکوردها را از دست‌رفته ببیند.
func TestFileStoreRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	if err := os.WriteFile(path, []byte("{این JSON نیست"), 0o644); err != nil {
		t.Fatalf("نوشتن فایل خراب ناموفق: %v", err)
	}
	if _, err := NewFileStore(path); err == nil {
		t.Fatal("فایل خراب باید خطای صریح بدهد")
	} else if !strings.Contains(err.Error(), path) {
		t.Fatalf("پیام خطا باید نام فایل را داشته باشد: %v", err)
	}
}

// TestFileStoreSnapshotShape می‌سنجد فایل داده خوانا و پایدار است (نسخه، ترتیب، رکوردها).
func TestFileStoreSnapshotShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	ctx := context.Background()
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("ساخت انبار ناموفق: %v", err)
	}
	if _, err := store.Save(ctx, Record{ID: "BNP-0001", Kind: "bonusPayments", Status: "draft"}); err != nil {
		t.Fatalf("ذخیره ناموفق: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("خواندن فایل داده ناموفق: %v", err)
	}
	var snap snapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("فایل داده باید JSON خوانا باشد: %v", err)
	}
	if snap.Version != snapshotVersion {
		t.Fatalf("نسخهٔ فایل %d است، انتظار %d", snap.Version, snapshotVersion)
	}
	if len(snap.Records) != 1 || len(snap.Order) != 1 {
		t.Fatalf("شکل فایل نادرست: %d رکورد، %d ترتیب", len(snap.Records), len(snap.Order))
	}
	if store.Saves() == 0 {
		t.Fatal("شمار نوشتن‌ها باید افزایش یابد")
	}
	// فایل موقت نباید باقی بماند.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("فایل موقت نوشتن باید پس از جایگزینی پاک شود")
	}
}
