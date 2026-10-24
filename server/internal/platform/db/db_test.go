// آزمون زیرساخت «db» در پلتفرم سرور — «db_test».

package db

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDiscoverOrder ترتیب مهاجرت‌ها را می‌سنجد: نخست مهاجرت‌های پایه و سپس مهاجرت‌های
// ماژول‌ها به ترتیب نام، چون schemaها باید پیش از جدول‌ها ساخته شوند.
func TestDiscoverOrder(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("ساخت پوشه ناموفق: %v", err)
		}
		if err := os.WriteFile(path, []byte("select 1;"), 0o644); err != nil {
			t.Fatalf("نوشتن فایل ناموفق: %v", err)
		}
	}
	write("migrations/0001_schemas.sql")
	write("migrations/000_seed_rbac.sql")
	write("migrations/notes.txt") // نادیده گرفته می‌شود
	write("internal/modules/contracts/migrations/0001_contract.sql")
	write("internal/modules/staff_services/migrations/0001_personnel.sql")
	write("internal/modules/staff_services/migrations/0002_add_index.sql")

	got, err := Discover(root)
	if err != nil {
		t.Fatalf("کشف مهاجرت‌ها ناموفق: %v", err)
	}
	want := []string{
		"core/0001_schemas.sql",
		"core/000_seed_rbac.sql",
		"contracts/0001_contract.sql",
		"staff_services/0001_personnel.sql",
		"staff_services/0002_add_index.sql",
	}
	if len(got) != len(want) {
		t.Fatalf("تعداد مهاجرت‌ها باید %d باشد، %d شد: %+v", len(want), len(got), got)
	}
	for i, w := range want {
		item := got[i].Module + "/" + got[i].Name
		if item != w {
			t.Fatalf("ترتیب مهاجرت‌ها نادرست است: جای %d باید %s باشد، %s شد", i, w, item)
		}
	}
	if got[2].Version != 1 || got[4].Version != 2 {
		t.Fatalf("شمارهٔ نسخه از نام فایل خوانده نشد: %+v", got)
	}
}

// TestDiscoverWithoutModuleTree ریشه‌ای بدون پوشهٔ ماژول‌ها هم باید کار کند.
func TestDiscoverWithoutModuleTree(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "migrations"), 0o755); err != nil {
		t.Fatalf("ساخت پوشه ناموفق: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "migrations", "0001_schemas.sql"), []byte("select 1;"), 0o644); err != nil {
		t.Fatalf("نوشتن فایل ناموفق: %v", err)
	}
	got, err := Discover(root)
	if err != nil {
		t.Fatalf("کشف مهاجرت‌ها ناموفق: %v", err)
	}
	if len(got) != 1 || got[0].Module != "core" {
		t.Fatalf("انتظار یک مهاجرت پایه بود: %+v", got)
	}
}

func TestOpenRejectsEmptyDSN(t *testing.T) {
	if _, err := Open(t.Context(), "   "); err == nil {
		t.Fatal("نشانی خالی باید خطا بدهد")
	}
}
