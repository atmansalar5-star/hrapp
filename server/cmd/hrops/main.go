// Command hrops ابزار عملیاتی پایگاه‌دادهٔ سامانه: بررسی اتصال و سطح دسترسی، ساخت
// پایگاه‌دادهٔ تازه و شمارش سطرها — بدون نیاز به psql/pg_dump روی ماشین.
//
// چرا این ابزار: استقرار روی ویندوز فقط سرور PostgreSQL را دارد (نه ابزارهای کلاینت)؛
// و «آمادگی عملیاتی» باید سنجیدنی باشد، نه حدس. `-ping` برای دروازهٔ استقرار و
// بازبینی پس از نصب است و اگر پایگاه‌داده نقش سوپرکاربر داشته باشد یا مهاجرت‌ها/جدول‌ها
// کامل نباشند با کد خطا برمی‌گردد.
//
// نمونه:
//
//	hrops -db "postgres://hrms@127.0.0.1:5432/hrms?sslmode=disable" -ping
//	hrops -db "postgres://hrms@127.0.0.1:5432/postgres?sslmode=disable" -create hrms_ops
//	hrops -db "..." -counts
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"

	_ "github.com/lib/pq"

	"hrms/internal/platform/datadict"
	platformmodules "hrms/internal/platform/modules"
)

func main() {
	dsn := flag.String("db", "", "نشانی پایگاه‌داده (پیش‌فرض: HRMS_DATABASE_URL)")
	ping := flag.Bool("ping", false, "بررسی اتصال، سطح دسترسی، جدول‌ها و مهاجرت‌ها (کد خروج ۱ اگر ناسالم باشد)")
	create := flag.String("create", "", "نام پایگاه‌دادهٔ تازه که باید ساخته شود")
	owner := flag.String("owner", "", "مالک پایگاه‌دادهٔ تازه در -create (خالی یعنی کاربر جاری)")
	createRole := flag.String("create-role", "", "ساخت نقش اجرای برنامه با حداقل دسترسی (LOGIN، بدون SUPERUSER/CREATEDB/CREATEROLE)")
	rolePassword := flag.String("role-password", "", "رمز نقش تازه در -create-role (خالی یعنی رمز تصادفی چاپ می‌شود)")
	drop := flag.String("drop", "", "حذف پایگاه‌داده (برای تنظیم دوبارهٔ محیط یا پاک‌کردن پایگاه‌دادهٔ آزمون بازگردانی)")
	confirmed := flag.Bool("yes", false, "تأیید عملیات برگشت‌ناپذیر (همراه با -drop)")
	counts := flag.Bool("counts", false, "شمارش سطرهای جدول‌های hr_*")
	envFile := flag.String("env", "", "فایل متغیرهای محیطی KEY=VALUE برای خواندن HRMS_DATABASE_URL (مثل .env)")
	dict := flag.Bool("dict", false, "ساخت «فرهنگ داده» از کاتالوگ زنده (ساختار درختی، ERD، سند و CSV)")
	format := flag.String("format", "tree", "قالب خروجی فرهنگ داده: "+strings.Join(datadict.Formats, " | "))
	out := flag.String("out", "", "مسیر فایل خروجی فرهنگ داده (خالی یعنی چاپ در خروجی استاندارد)")
	check := flag.Bool("check", false, "فقط بررسی هم‌خوانی فایل -out با کاتالوگ زنده (کد ۱ اگر قدیمی باشد)")
	flag.Parse()

	if *envFile != "" {
		loadEnvFile(*envFile)
	}
	if *dsn == "" {
		*dsn = strings.TrimSpace(os.Getenv("HRMS_DATABASE_URL"))
	}
	if *dsn == "" && !*dict {
		fmt.Fprintln(os.Stderr, "استفاده: hrops -db <dsn> -ping | -create <name> [-owner <role>] | -create-role <role> | -drop <name> -yes | -counts | -dict [-format tree|mermaid|md|csv|json] [-out file]")
		os.Exit(2)
	}

	if *dict {
		renderDictionary(*dsn, *format, *out, *check)
		return
	}

	if *create != "" {
		createDatabase(*dsn, *create, *owner)
		return
	}

	if *createRole != "" {
		createAppRole(*dsn, *createRole, *rolePassword)
		return
	}

	if *drop != "" {
		if !*confirmed {
			fmt.Fprintln(os.Stderr, "✖ حذف پایگاه‌داده برگشت‌پذیر نیست؛ برای تأیید -yes را هم بدهید.")
			os.Exit(2)
		}
		dropDatabase(*dsn, *drop)
		return
	}

	db := open(*dsn)
	defer db.Close()

	switch {
	case *ping:
		if !pingReport(db) {
			os.Exit(1)
		}
	case *counts:
		printCounts(db)
	default:
		fmt.Fprintln(os.Stderr, "یکی از -ping، -create یا -counts را بدهید.")
		os.Exit(2)
	}
}

func open(dsn string) *sql.DB {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fail("اتصال به پایگاه‌داده", err)
	}
	return db
}

// pingReport گزارش سلامت پایگاه‌داده را چاپ می‌کند و می‌گوید آماده است یا نه.
func pingReport(db *sql.DB) bool {
	ok := true

	var (
		version     string
		currentUser string
		dbName      string
		isSuper     bool
		canCreateDB bool
	)
	err := db.QueryRow(`select version(), current_user, current_database(),
		coalesce((select rolsuper from pg_roles where rolname = current_user), false),
		coalesce((select rolcreatedb from pg_roles where rolname = current_user), false)`).
		Scan(&version, &currentUser, &dbName, &isSuper, &canCreateDB)
	if err != nil {
		fail("پرس‌وجوی وضعیت پایگاه‌داده", err)
		return false
	}
	if i := strings.IndexByte(version, ','); i > 0 {
		version = version[:i]
	}
	fmt.Printf("پایگاه‌داده: %s\nکاربر: %s\nنسخه: %s\n", dbName, currentUser, version)
	if isSuper {
		fmt.Println("✖ کاربر پایگاه‌داده سوپرکاربر است؛ نقش برنامه باید حداقل دسترسی داشته باشد.")
		ok = false
	} else {
		fmt.Println("✓ نقش برنامه سوپرکاربر نیست")
	}
	if canCreateDB {
		fmt.Println("· نقش برنامه اجازهٔ ساخت پایگاه‌داده دارد (برای راهبر استقرار لازم است، برای حساب اجرا بهتر است خاموش باشد)")
	}

	var schemaCount, tableCount int
	if err := db.QueryRow(`select count(*) from information_schema.schemata where schema_name like 'hr\_%'`).Scan(&schemaCount); err != nil {
		fail("شمارش schemaها", err)
		ok = false
	}
	if err := db.QueryRow(`select count(*) from information_schema.tables
		where table_schema like 'hr\_%' and table_type = 'BASE TABLE'`).Scan(&tableCount); err != nil {
		fail("شمارش جدول‌ها", err)
		ok = false
	}
	fmt.Printf("schemaها: %d · جدول‌ها: %d\n", schemaCount, tableCount)
	if schemaCount == 0 || tableCount == 0 {
		fmt.Println("✖ مهاجرت‌ها روی این پایگاه‌داده اعمال نشده‌اند (سرور را یک‌بار با DSN همین پایگاه‌داده بالا بیاورید).")
		ok = false
	}

	var migrations int
	if err := db.QueryRow(`select count(*) from public.schema_migration`).Scan(&migrations); err != nil {
		fmt.Println("✖ دفتر مهاجرت‌ها (public.schema_migration) خوانده نشد.")
		ok = false
	} else {
		fmt.Printf("مهاجرت‌های اعمال‌شده: %d\n", migrations)
		if migrations == 0 {
			fmt.Println("✖ هیچ مهاجرتی اعمال نشده است.")
			ok = false
		}
	}

	// جدول‌های خالی از دادهٔ کسب‌وکار: در نصب عملیاتی، تنها جدول‌های پارامتری باید پر باشند.
	rows, err := db.Query(`select table_schema || '.' || table_name from information_schema.tables
		where table_schema like 'hr\_%' and table_type = 'BASE TABLE' order by 1`)
	if err == nil {
		var total int
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				var n int64
				if err := db.QueryRow(`select count(*) from ` + quoteIdentPath(name)).Scan(&n); err == nil {
					total += int(n)
				}
			}
		}
		rows.Close()
		fmt.Printf("سطرهای دادهٔ کسب‌وکار: %d\n", total)
	}

	if ok {
		fmt.Println("✓ پایگاه‌داده برای اجرا آماده است")
	}
	return ok
}

// createDatabase پایگاه‌دادهٔ تازه با کدگذاری UTF8 می‌سازد. اتصال به پایگاه‌دادهٔ
// «postgres» انجام می‌شود، چون CREATE DATABASE داخل تراکنش و روی اتصال جاری ممکن نیست.
func createDatabase(dsn, name, owner string) {
	target := strings.TrimSpace(name)
	if target == "" || strings.ContainsAny(target, "\"'`;") {
		fmt.Fprintln(os.Stderr, "✖ نام پایگاه‌دادهٔ تازه نامعتبر است")
		os.Exit(2)
	}
	adminDSN := replaceDatabase(dsn, "postgres")
	db, err := sql.Open("postgres", adminDSN)
	if err != nil {
		fail("اتصال به پایگاه‌دادهٔ مدیریتی", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fail("پاسخ پایگاه‌دادهٔ مدیریتی", err)
	}

	var exists bool
	if err := db.QueryRow(`select exists(select 1 from pg_database where datname = $1)`, target).Scan(&exists); err != nil {
		fail("بررسی وجود پایگاه‌داده", err)
	}
	if exists {
		fmt.Printf("· پایگاه‌دادهٔ «%s» از قبل هست؛ تغییری انجام نشد.\n", target)
		return
	}

	ownerName := strings.TrimSpace(owner)
	if ownerName == "" {
		if err := db.QueryRow(`select current_user`).Scan(&ownerName); err != nil {
			fail("خواندن کاربر جاری", err)
		}
	}
	if !validIdent(ownerName) {
		fmt.Fprintln(os.Stderr, "✖ نام نقش مالک نامعتبر است")
		os.Exit(2)
	}
	stmt := fmt.Sprintf(`create database %s with owner %s encoding 'UTF8' template template0`,
		quoteIdent(target), quoteIdent(ownerName))
	if _, err := db.Exec(stmt); err != nil {
		fail("ساخت پایگاه‌داده", err)
	}
	fmt.Printf("✓ پایگاه‌دادهٔ «%s» با مالک «%s» ساخته شد.\n", target, ownerName)
	fmt.Println("  گام بعد: HRMS_DATABASE_URL را به همین پایگاه‌داده تغییر دهید و سرور را یک‌بار بالا بیاورید تا مهاجرت‌ها اعمال شوند.")
}

// createAppRole نقش اجرای برنامه را با حداقل دسترسی می‌سازد: ورود دارد، ولی نه
// SUPERUSER، نه CREATEDB و نه CREATEROLE؛ بنابراین نمی‌تواند پایگاه‌داده یا نقش تازه
// بسازد و به دادهٔ پایگاه‌داده‌های دیگر دست نمی‌زند.
func createAppRole(dsn, name, password string) {
	roleName := strings.TrimSpace(name)
	if !validIdent(roleName) {
		fmt.Fprintln(os.Stderr, "✖ نام نقش نامعتبر است")
		os.Exit(2)
	}
	adminDSN := replaceDatabase(dsn, "postgres")
	db, err := sql.Open("postgres", adminDSN)
	if err != nil {
		fail("اتصال به پایگاه‌دادهٔ مدیریتی", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fail("پاسخ پایگاه‌دادهٔ مدیریتی", err)
	}

	var exists bool
	if err := db.QueryRow(`select exists(select 1 from pg_roles where rolname = $1)`, roleName).Scan(&exists); err != nil {
		fail("بررسی وجود نقش", err)
	}
	if exists {
		fmt.Printf("· نقش «%s» از قبل هست؛ تغییری انجام نشد.\n", roleName)
		fmt.Println("  برای بازنشانی رمز: alter role … با رمز تازه (یا نقش را حذف و همین فرمان را تکرار کنید).")
		return
	}

	secret := strings.TrimSpace(password)
	generated := false
	if secret == "" {
		secret = randomSecret(24)
		generated = true
	}
	stmt := fmt.Sprintf(`create role %s login password %s nosuperuser nocreatedb nocreaterole`,
		quoteIdent(roleName), quoteLiteral(secret))
	if _, err := db.Exec(stmt); err != nil {
		fail("ساخت نقش اجرای برنامه", err)
	}
	fmt.Printf("✓ نقش «%s» با حداقل دسترسی ساخته شد (LOGIN، بدون SUPERUSER/CREATEDB/CREATEROLE).\n", roleName)
	if generated {
		fmt.Printf("  رمز تصادفی نقش: %s\n", secret)
	}
	fmt.Println("  این رمز را در HRMS_DATABASE_URL بگذارید و همان‌جا محافظت کنید (server/.env).")
}

// validIdent نام نقش/پایگاه‌داده را می‌سنجد (حروف کوچک، رقم، زیرخط).
func validIdent(value string) bool {
	if value == "" || len(value) > 63 {
		return false
	}
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r == '_':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

// randomSecret رمز تصادفی می‌سازد (بدون وابستگی به بستهٔ داخلی سامانه).
func randomSecret(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	buf := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			fail("تولید رمز تصادفی", err)
		}
		buf[i] = alphabet[n.Int64()]
	}
	return string(buf)
}

// dropDatabase پایگاه‌داده را حذف می‌کند (اتصال‌های باز قطع می‌شوند). برای تنظیم دوبارهٔ
// محیط استقرار یا پاک‌کردن پایگاه‌دادهٔ آزمون بازگردانی؛ با -yes تأیید می‌شود.
func dropDatabase(dsn, name string) {
	target := strings.TrimSpace(name)
	if !validIdent(target) {
		fmt.Fprintln(os.Stderr, "✖ نام پایگاه‌داده نامعتبر است")
		os.Exit(2)
	}
	db := open(replaceDatabase(dsn, "postgres"))
	defer db.Close()
	if _, err := db.Exec(`select pg_terminate_backend(pid) from pg_stat_activity
		where datname = $1 and pid <> pg_backend_pid()`, target); err != nil {
		fail("قطع اتصال‌های پایگاه‌داده", err)
	}
	if _, err := db.Exec(fmt.Sprintf(`drop database if exists %s`, quoteIdent(target))); err != nil {
		fail("حذف پایگاه‌داده", err)
	}
	fmt.Printf("✓ پایگاه‌دادهٔ «%s» حذف شد.\n", target)
}

// renderDictionary فرهنگ داده را می‌سازد و چاپ/ذخیره می‌کند.
//
// برای فایل‌های نسخه‌دار (docs/generated) حالت «پایدار» به کار می‌رود: بدون مُهر زمان و
// بدون شمار سطرها، تا با هر تغییر داده (که ساختار را عوض نمی‌کند) فایل کهنه نشود. خروجی
// زندهٔ API همان قالب کامل را با زمان و شمار سطر می‌دهد.
func renderDictionary(dsn, format, outPath string, checkOnly bool) {
	if strings.TrimSpace(dsn) == "" {
		fmt.Fprintln(os.Stderr, "✖ فرهنگ داده به نشانی پایگاه‌داده نیاز دارد (HRMS_DATABASE_URL یا -db یا -env .env)")
		os.Exit(2)
	}
	db := open(dsn)
	defer db.Close()
	if err := db.Ping(); err != nil {
		fail("پاسخ پایگاه‌داده", err)
	}

	options := datadict.Options{}
	if outPath != "" {
		options.Stable = true
	}
	dict, err := datadict.Load(context.Background(), db, moduleInfos(), options)
	if err != nil {
		fail("ساخت فرهنگ داده", err)
	}
	body, _, err := datadict.Render(dict, format)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✖ "+err.Error())
		os.Exit(2)
	}

	if outPath == "" {
		fmt.Print(body)
		return
	}

	if checkOnly {
		current, readErr := os.ReadFile(outPath)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "✖ فایل «%s» خوانده نشد: %v\n  برای ساخت: hrops -env .env -dict -format %s -out %s\n", outPath, readErr, format, outPath)
			os.Exit(1)
		}
		if string(current) != body {
			fmt.Fprintf(os.Stderr, "✖ «%s» با کاتالوگ زنده هم‌خوان نیست؛ بازتولید کنید:\n  hrops -env .env -dict -format %s -out %s\n", outPath, format, outPath)
			os.Exit(1)
		}
		fmt.Printf("✓ «%s» با کاتالوگ زنده هم‌خوان است\n", outPath)
		return
	}

	if err := os.WriteFile(outPath, []byte(body), 0o644); err != nil {
		fail("نوشتن فایل خروجی", err)
	}
	fmt.Printf("✓ فرهنگ داده در «%s» نوشته شد — %d طرح · %d جدول · %d ستون · %d رابطه\n",
		outPath, dict.Summary.Schemas, dict.Summary.Tables, dict.Summary.Columns, dict.Summary.Relations)
}

// moduleInfos نگاشت ماژول‌های سامانه را از رجیستری مانیفست‌ها می‌سازد (نام فارسی،
// پیشوند API، طرح داده، موجودیت‌ها و صفحه‌ها) تا فرهنگ داده به زبان سامانه حرف بزند.
func moduleInfos() []datadict.ModuleInfo {
	registry := platformmodules.NewRegistry(nil)
	out := make([]datadict.ModuleInfo, 0, registry.Count())
	for _, m := range registry.All() {
		screens := make([]string, 0, len(m.Screens))
		for _, s := range m.Screens {
			screens = append(screens, s.TitleFa)
		}
		out = append(out, datadict.ModuleInfo{
			ID:        m.ID,
			TitleFa:   m.TitleFa,
			APIPrefix: m.APIPrefix,
			DBSchema:  m.DBSchema,
			Entities:  m.Entities,
			Screens:   screens,
		})
	}
	return out
}

// loadEnvFile متغیرهای محیطی را از فایل سادهٔ KEY=VALUE می‌خواند (نبودن فایل خطا نیست و
// متغیری که در محیط تعریف شده باشد بازنویسی نمی‌شود — همان قاعدهٔ سرور).
func loadEnvFile(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

// printCounts شمار سطرهای هر جدول hr_* را چاپ می‌کند (برای مقایسهٔ پشتیبان و بازگردانی).
func printCounts(db *sql.DB) {
	rows, err := db.Query(`select table_schema, table_name from information_schema.tables
		where table_schema like 'hr\_%' and table_type = 'BASE TABLE'
		order by table_schema, table_name`)
	if err != nil {
		fail("فهرست جدول‌ها", err)
	}
	defer rows.Close()

	type table struct{ schema, name string }
	var tables []table
	for rows.Next() {
		var t table
		if err := rows.Scan(&t.schema, &t.name); err != nil {
			fail("خواندن فهرست جدول‌ها", err)
		}
		tables = append(tables, t)
	}
	sort.Slice(tables, func(i, j int) bool {
		if tables[i].schema == tables[j].schema {
			return tables[i].name < tables[j].name
		}
		return tables[i].schema < tables[j].schema
	})

	var total int64
	for _, t := range tables {
		var n int64
		if err := db.QueryRow(`select count(*) from ` + quoteIdent(t.schema) + "." + quoteIdent(t.name)).Scan(&n); err != nil {
			fmt.Printf("%-38s ✖ %v\n", t.schema+"."+t.name, err)
			continue
		}
		total += n
		fmt.Printf("%-38s %6d\n", t.schema+"."+t.name, n)
	}
	fmt.Printf("%-38s %6d\n", "مجموع", total)
}

// replaceDatabase نام پایگاه‌داده را در DSN عوض می‌کند (بدون بازنویسی بقیهٔ بخش‌ها).
func replaceDatabase(dsn, name string) string {
	query := ""
	if i := strings.IndexByte(dsn, '?'); i >= 0 {
		query = dsn[i:]
		dsn = dsn[:i]
	}
	if i := strings.LastIndexByte(dsn, '/'); i >= 0 {
		dsn = dsn[:i+1] + name
	}
	return dsn + query
}

// quoteIdentPath مسیر «schema.table» را برای درج در SQL امن می‌کند.
func quoteIdentPath(path string) string {
	parts := strings.SplitN(path, ".", 2)
	for i := range parts {
		parts[i] = quoteIdent(parts[i])
	}
	return strings.Join(parts, ".")
}

// quoteIdent شناسه را مثل pq.QuoteIdentifier امن می‌کند (بدون وابستگی اضافه).
func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

// quoteLiteral رشتهٔ متنی را برای درج در SQL امن می‌کند (رمز نقش، پیام‌ها).
func quoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "✖ %s ناموفق بود: %v\n", what, err)
	os.Exit(1)
}
