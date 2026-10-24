// Package db اتصال پایگاه‌داده و اجرای مهاجرت‌ها.
//
// سه چیز اینجا زندگی می‌کند و هیچ‌کدام به ماژول‌ها وابسته نیست:
//  1. باز کردن اتصال Postgres از متغیر محیطی،
//  2. پیدا کردن فایل‌های مهاجرت (پایه + مهاجرت‌های هر ماژول)،
//  3. اعمال مهاجرت‌های اعمال‌نشده و ثبت آن‌ها در دفتر `public.schema_migration`.
//
// ترتیب اعمال عمدی و صریح است: مهاجرت‌های پایه، سپس seed دسترسی‌ها (تولیدشده از
// مانیفست‌ها) و در پایان مهاجرت‌های هر ماژول به ترتیب نام پوشه. هر فایل بدون تراکنش
// صریح، به‌صورت یک دستور چندجمله‌ای اجرا می‌شود که خود Postgres آن را در یک تراکنش
// ضمنی می‌پیچد؛ پس یا همهٔ جمله‌های یک فایل اعمال می‌شوند یا هیچ‌کدام.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Driver نام درایور ثبت‌شدهٔ Postgres در database/sql.
const Driver = "postgres"

// Open اتصال پایگاه‌داده را باز و بررسی می‌کند.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("نشانی پایگاه‌داده تنظیم نشده است")
	}
	sqlDB, err := sql.Open(Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("باز کردن اتصال پایگاه‌داده ناموفق: %w", err)
	}
	sqlDB.SetMaxOpenConns(12)
	sqlDB.SetMaxIdleConns(6)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("اتصال به پایگاه‌داده برقرار نشد: %w", err)
	}
	return sqlDB, nil
}

// Migration یک فایل مهاجرت.
type Migration struct {
	// Module مالک مهاجرت؛ `core` برای مهاجرت‌های پایهٔ سرور.
	Module string
	// Version شمارهٔ نسخه که از نام فایل خوانده می‌شود.
	Version int
	// Name نام فایل.
	Name string
	// Path مسیر کامل فایل.
	Path string
}

var versionPattern = regexp.MustCompile(`^(\d+)_`)

// Discover فایل‌های مهاجرت را پیدا می‌کند.
//
// ترتیب: نخست `server/migrations/*.sql` (به ترتیب نام فایل، چون نام‌ها عمداً با شماره
// شروع می‌شوند) و سپس `server/internal/modules/<module>/migrations/*.sql` به ترتیب نام
// ماژول. این ترتیب تضمین می‌کند schema‌ها پیش از جدول‌ها ساخته شوند.
func Discover(root string) ([]Migration, error) {
	out := []Migration{}
	base := filepath.Join(root, "migrations")
	if files, err := sqlFiles(base); err == nil {
		for _, f := range files {
			out = append(out, newMigration("core", f))
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	modulesRoot := filepath.Join(root, "internal", "modules")
	entries, err := os.ReadDir(modulesRoot)
	if err != nil {
		return out, nil // در آزمون‌های کوچک ممکن است پوشهٔ ماژول‌ها نباشد.
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		files, err := sqlFiles(filepath.Join(modulesRoot, name, "migrations"))
		if err != nil {
			continue
		}
		for _, f := range files {
			out = append(out, newMigration(name, f))
		}
	}
	return out, nil
}

func sqlFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sort.Strings(out)
	return out, nil
}

func newMigration(module, path string) Migration {
	name := filepath.Base(path)
	version := 0
	if m := versionPattern.FindStringSubmatch(name); m != nil {
		version, _ = strconv.Atoi(m[1])
	}
	return Migration{Module: module, Version: version, Name: name, Path: path}
}

const migrationTable = `
create table if not exists public.schema_migration (
  module      text        not null,
  version     integer     not null,
  name        text        not null,
  applied_at  timestamptz not null default now(),
  primary key (module, version)
);`

// Migrate مهاجرت‌های اعمال‌نشده را اجرا و ثبت می‌کند و شمار اعمال‌شده را برمی‌گرداند.
func Migrate(ctx context.Context, sqlDB *sql.DB, migrations []Migration, logger *slog.Logger) (int, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if _, err := sqlDB.ExecContext(ctx, migrationTable); err != nil {
		return 0, fmt.Errorf("ساخت دفتر مهاجرت ناموفق: %w", err)
	}

	applied := map[string]bool{}
	rows, err := sqlDB.QueryContext(ctx, `select module, version from public.schema_migration`)
	if err != nil {
		return 0, fmt.Errorf("خواندن دفتر مهاجرت ناموفق: %w", err)
	}
	for rows.Next() {
		var module string
		var version int
		if err := rows.Scan(&module, &version); err != nil {
			rows.Close()
			return 0, err
		}
		applied[fmt.Sprintf("%s/%d", module, version)] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	count := 0
	for _, m := range migrations {
		if applied[fmt.Sprintf("%s/%d", m.Module, m.Version)] {
			continue
		}
		body, err := os.ReadFile(m.Path)
		if err != nil {
			return count, fmt.Errorf("خواندن مهاجرت %s ناموفق: %w", m.Name, err)
		}
		if _, err := sqlDB.ExecContext(ctx, string(body)); err != nil {
			return count, fmt.Errorf("اجرای مهاجرت %s/%s ناموفق: %w", m.Module, m.Name, err)
		}
		if _, err := sqlDB.ExecContext(ctx,
			`insert into public.schema_migration (module, version, name) values ($1,$2,$3)
			 on conflict (module, version) do update set name = excluded.name, applied_at = now()`,
			m.Module, m.Version, m.Name); err != nil {
			return count, fmt.Errorf("ثبت مهاجرت %s/%s ناموفق: %w", m.Module, m.Name, err)
		}
		count++
		logger.Info("مهاجرت اعمال شد", "module", m.Module, "name", m.Name, "version", m.Version)
	}
	return count, nil
}

// Applied شمار مهاجرت‌های ثبت‌شده را برمی‌گرداند (برای بررسی آمادگی سرویس).
func Applied(ctx context.Context, sqlDB *sql.DB) (int, error) {
	var n int
	err := sqlDB.QueryRowContext(ctx, `select count(*) from public.schema_migration`).Scan(&n)
	return n, err
}
