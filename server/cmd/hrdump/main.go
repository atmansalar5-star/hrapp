// Package main ابزار یک‌بارمصرف «برون‌کشی دادهٔ اجرا» (logical dump) بدون نیاز به pg_dump.
//
// چرا: روی این ماشین فقط سرور PostgreSQL نصب است و ابزارهای کلاینت (pg_dump/psql) نیستند.
// دادهٔ کسب‌وکار در جدول‌های طرح‌های `hr_*` است و ساختار همان جدول‌ها از فایل‌های مهاجرت
// خود نرم‌افزار ساخته می‌شود؛ پس برای بک‌آپ، «داده» کافی است: هر جدول به شکل
// INSERT ... ON CONFLICT DO NOTHING (بازگردانی چندباره بی‌خطر) و در پایان مقدار
// توالی‌ها با setval. متن هر مقدار با quote_nullable خودِ پایگاه‌داده ساخته می‌شود،
// پس هیچ نوع داده‌ای نادرست درج نمی‌شود.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

type table struct {
	schema string
	name   string
	cols   []string
}

func main() {
	dsn := flag.String("db", "", "نشانی پایگاه‌داده")
	out := flag.String("out", "", "مسیر فایل خروجی SQL (حالت برون‌کشی)")
	apply := flag.String("apply", "", "مسیر فایل SQL برای اجرا روی پایگاه‌دادهٔ هدف (حالت بازگردانی)")
	counts := flag.Bool("counts", false, "شمارش سطرهای جدول‌های `hr_*` (برای مقایسهٔ مبدأ و بازگردانی)")
	withCounts := flag.Bool("with-counts", false, "پس از برون‌کشی، شمار سطرهای همان لحظه را هم چاپ کن")
	truncate := flag.Bool("truncate", false, "خالی‌کردن همهٔ جدول‌های `hr_*` (برای آزمون بازگردانی روی پایگاه‌دادهٔ آزمایشی)")
	flag.Parse()
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "استفاده: tmpdump -db <dsn> [-out file.sql | -apply file.sql | -counts]")
		os.Exit(2)
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		fail("اتصال به پایگاه‌داده", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fail("پاسخ پایگاه‌داده", err)
	}

	switch {
	case *truncate:
		truncateAll(db)
		return
	case *counts:
		printCounts(db)
		return
	case *apply != "":
		applyFile(db, *apply)
		return
	case *out == "":
		fmt.Fprintln(os.Stderr, "برای برون‌کشی باید -out بدهید.")
		os.Exit(2)
	}

	tables, skipped, err := appTables(db)
	if err != nil {
		fail("فهرست جدول‌ها", err)
	}

	var b strings.Builder
	b.WriteString("-- برون‌کشی دادهٔ اجرای سامانهٔ جامع منابع انسانی\n")
	b.WriteString("-- ساختار جدول‌ها از مهاجرت‌های خودِ نرم‌افزار ساخته می‌شود؛ این فایل فقط «داده» است.\n")
	b.WriteString("-- بازگردانی: یک‌بار سرور را روی پایگاه‌دادهٔ تازه بالا بیاورید (تا مهاجرت‌ها اعمال شوند)،\n")
	b.WriteString("-- سپس این فایل را اجرا کنید. تکرار اجرا بی‌خطر است (ON CONFLICT DO NOTHING).\n\n")
	b.WriteString("begin;\n\n")

	rowsTotal := 0
	failed := []string{}
	for _, t := range tables {
		cols := make([]string, 0, len(t.cols))
		values := make([]string, 0, len(t.cols))
		for _, c := range t.cols {
			cols = append(cols, fmt.Sprintf("%q", c))
			values = append(values, fmt.Sprintf("quote_nullable(%q::text)", c))
		}
		// سه جای‌نگهدارِ format() دقیقاً سه آرگومان می‌گیرند: نام طرح، نام جدول و متن
		// مقدارهای همان سطر (که با concat_ws از quote_nullable ساخته می‌شود). فهرست
		// ستون‌ها متن ثابتی است که همین‌جا داخل قالب می‌نشیند؛ اگر آن را هم جای‌نگهدار
		// می‌گذاشتیم، شمار جای‌نگهدارها با شمار آرگومان‌ها ناهمخوان می‌شد و Postgres
		// خطای «too few arguments for format()» می‌داد.
		q := fmt.Sprintf(`select coalesce(string_agg(
			format('insert into %%I.%%I (%s) values (%%s) on conflict do nothing;',
				%s, %s, concat_ws(', ', %s)), E'\n'), '')
			from %q.%q`,
			strings.Join(cols, ", "),
			quoteLiteral(t.schema), quoteLiteral(t.name),
			strings.Join(values, ", "), t.schema, t.name)
		var stmts sql.NullString
		if err := db.QueryRow(q).Scan(&stmts); err != nil {
			failed = append(failed, t.schema+"."+t.name)
			fmt.Fprintf(os.Stderr, "%-38s خطا: %v\n", t.schema+"."+t.name, err)
			continue
		}
		n := 0
		if stmts.Valid && stmts.String != "" {
			n = strings.Count(stmts.String, "\n") + 1
			b.WriteString("-- " + t.schema + "." + t.name + " — " + fmt.Sprint(n) + " سطر\n")
			b.WriteString(stmts.String + "\n\n")
		}
		rowsTotal += n
		fmt.Fprintf(os.Stderr, "%-38s %5d سطر\n", t.schema+"."+t.name, n)
	}

	seqs, err := sequences(db)
	if err != nil {
		fail("فهرست توالی‌ها", err)
	}
	if len(seqs) > 0 {
		b.WriteString("-- مقدار توالی‌ها (تا شناسه‌های تازه با دادهٔ بازگردانده‌شده تصادم نکنند)\n")
		for _, s := range seqs {
			var last int64
			var called bool
			if err := db.QueryRow(fmt.Sprintf("select last_value, is_called from %q.%q", s[0], s[1])).Scan(&last, &called); err != nil {
				continue
			}
			b.WriteString(fmt.Sprintf("select setval('%s.%s', %d, %t);\n", s[0], s[1], last, called))
		}
		b.WriteString("\n")
	}
	b.WriteString("commit;\n")

	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		fail("نوشتن فایل خروجی", err)
	}
	fmt.Printf("جدول: %d | سطر: %d | توالی: %d | ردشده: %d | خطا: %d | خروجی: %s\n",
		len(tables), rowsTotal, len(seqs), len(skipped), len(failed), *out)
	if len(skipped) > 0 {
		fmt.Println("جدول‌های بدون ستون (رد شد):", strings.Join(skipped, ", "))
	}
	// شمار سطرهای همان لحظهٔ برون‌کشی: برای مقایسهٔ بعدی با پایگاه‌دادهٔ بازگردانده‌شده
	// لازم است، چون پایگاه‌دادهٔ مبدأ زنده است و ممکن است بعد از این لحظه رشد کند.
	if *withCounts {
		printCounts(db)
	}
}

// appTables جدول‌های طرح‌های `hr_*` را با ستون‌هایشان برمی‌گرداند.
//
// از کاتالوگ سیستم (`pg_class`/`pg_attribute`) خوانده می‌شود، نه `information_schema`:
// آن‌جا هر چه شبیه جدول باشد می‌آید، ولی اینجا فقط جدول‌های واقعی (`relkind = 'r'`) و
// ستون‌های واقعی‌شان. جدولی که ستون ندارد (بازماندهٔ نام‌گذاری دیگر) در فهرست «ردشده»
// می‌آید تا بک‌آپ با یک جدول بی‌ستون شکست نخورد.
func appTables(db *sql.DB) ([]table, []string, error) {
	rows, err := db.Query(`
		select n.nspname, c.relname, coalesce(a.attname, '')
		from pg_class c
		join pg_namespace n on n.oid = c.relnamespace
		left join pg_attribute a
		  on a.attrelid = c.oid and a.attnum > 0 and not a.attisdropped
		where c.relkind = 'r' and n.nspname like 'hr\_%'
		order by n.nspname, c.relname, a.attnum`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []table{}
	skipped := []string{}
	for rows.Next() {
		var schema, name, col string
		if err := rows.Scan(&schema, &name, &col); err != nil {
			return nil, nil, err
		}
		if n := len(out); n > 0 && out[n-1].schema == schema && out[n-1].name == name {
			if col != "" {
				out[n-1].cols = append(out[n-1].cols, col)
			}
			continue
		}
		if col == "" {
			skipped = append(skipped, schema+"."+name)
			continue
		}
		out = append(out, table{schema: schema, name: name, cols: []string{col}})
	}
	return out, skipped, rows.Err()
}

// sequences توالی‌های طرح‌های `hr_*` را برمی‌گرداند.
func sequences(db *sql.DB) ([][2]string, error) {
	rows, err := db.Query(`
		select sequence_schema, sequence_name
		from information_schema.sequences
		where sequence_schema like 'hr\_%'
		order by sequence_schema, sequence_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := [][2]string{}
	for rows.Next() {
		var schema, name string
		if err := rows.Scan(&schema, &name); err != nil {
			return nil, err
		}
		out = append(out, [2]string{schema, name})
	}
	return out, rows.Err()
}

func fail(what string, err error) {
	fmt.Fprintf(os.Stderr, "%s ناموفق بود: %v\n", what, err)
	os.Exit(1)
}

// quoteLiteral یک رشته را به شکل «متن SQL» با نقل‌قول تکی برمی‌گرداند.
func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// applyFile فایل SQL بک‌آپ را روی پایگاه‌دادهٔ هدف اجرا می‌کند (حالت بازگردانی).
//
// چرا در همین ابزار: این ماشین psql ندارد؛ اجرای فایل از راه همان درایور Go هم نشان
// می‌دهد فایل بک‌آپ واقعاً روی یک پایگاه‌دادهٔ تازه قابل بازگردانی است یا نه.
func applyFile(db *sql.DB, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fail("خواندن فایل بک‌آپ", err)
	}
	if _, err := db.Exec(string(raw)); err != nil {
		fail("اجرای فایل بک‌آپ", err)
	}
	fmt.Printf("فایل بک‌آپ با موفقیت اجرا شد: %s (%d بایت)\n", path, len(raw))
}

// truncateAll همهٔ جدول‌های `hr_*` را خالی می‌کند.
//
// چرا لازم است: در آزمون بازگردانی، سرور یک‌بار روی پایگاه‌دادهٔ تازه اجرا می‌شود تا
// مهاجرت‌ها جدول‌ها را بسازند، و همان اجرا چند سطر عملیاتی (رخدادها) می‌نویسد. برای
// مقایسهٔ دقیق، ابتدا آن پایگاه‌داده خالی می‌شود تا شمار سطرهای پس از بازگردانی، فقط
// همان چیزی باشد که در فایل بک‌آپ است.
func truncateAll(db *sql.DB) {
	tables, _, err := appTables(db)
	if err != nil {
		fail("فهرست جدول‌ها", err)
	}
	names := make([]string, 0, len(tables))
	for _, t := range tables {
		names = append(names, fmt.Sprintf("%q.%q", t.schema, t.name))
	}
	if len(names) == 0 {
		fmt.Println("جدولی برای خالی‌کردن نبود.")
		return
	}
	if _, err := db.Exec("truncate table " + strings.Join(names, ", ") + " restart identity cascade"); err != nil {
		fail("خالی‌کردن جدول‌ها", err)
	}
	fmt.Printf("همهٔ %d جدول خالی شد.\n", len(names))
}

// printCounts شمار سطرهای هر جدول `hr_*` را چاپ می‌کند تا با مبدأ مقایسه شود.
func printCounts(db *sql.DB) {
	tables, _, err := appTables(db)
	if err != nil {
		fail("فهرست جدول‌ها", err)
	}
	for _, t := range tables {
		var n int64
		if err := db.QueryRow(fmt.Sprintf("select count(*) from %q.%q", t.schema, t.name)).Scan(&n); err != nil {
			fmt.Printf("%s.%s=?\n", t.schema, t.name)
			continue
		}
		fmt.Printf("%s.%s=%d\n", t.schema, t.name, n)
	}
}
