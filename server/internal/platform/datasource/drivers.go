// پیاده‌سازی درایورهای منبع داده: فایل (CSV/Excel) و پایگاه‌داده‌ها.
//
// هر درایور در `init` خودش را در فهرست رابط ثبت می‌کند؛ پس افزودن منبع تازه فقط نوشتن
// همین قرارداد است، بدون دست‌زدن به مصرف‌کننده‌ها.

package datasource

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"  // درایور MySQL/MariaDB
	_ "github.com/lib/pq"               // درایور Postgres
	_ "github.com/microsoft/go-mssqldb" // درایور SQL Server
	_ "github.com/sijms/go-ora/v2"      // درایور Oracle
	"github.com/xuri/excelize/v2"
)

func init() {
	Register(newCSVSource())
	Register(newXLSXSource())
	Register(newSQLSource(sqlDialect{
		Kind: KindPostgres, TitleFa: "Postgres", Driver: "postgres", DefaultPort: "5432",
	}))
	Register(newSQLSource(sqlDialect{
		Kind: KindSQLServer, TitleFa: "Microsoft SQL Server", Driver: "sqlserver", DefaultPort: "1433",
	}))
	Register(newSQLSource(sqlDialect{
		Kind: KindMySQL, TitleFa: "MySQL / MariaDB", Driver: "mysql", DefaultPort: "3306",
	}))
	Register(newSQLSource(sqlDialect{
		Kind: KindOracle, TitleFa: "Oracle", Driver: "oracle", DefaultPort: "1521",
	}))
}

/* ------------------------------ منابع فایلی ------------------------------ */

// csvSource منبع دادهٔ فایل CSV/متنی.
type csvSource struct{}

func newCSVSource() Source { return csvSource{} }

func (csvSource) Kind() Kind         { return KindCSV }
func (csvSource) TitleFa() string    { return "فایل CSV / متنی" }
func (csvSource) CategoryFa() string { return "فایل" }

func (csvSource) Fields() []Field {
	return []Field{
		{Key: "filePath", TitleFa: "فایل", Type: "text", Required: true,
			Example: "uploads/employees.csv", NoteFa: "فایلی که از همین صفحه بارگذاری کرده‌اید."},
	}
}

func (s csvSource) Test(ctx context.Context, cfg Config) (TestReport, error) {
	started := time.Now()
	file, err := openTextFile(cfg.filePath())
	if err != nil {
		return TestReport{}, err
	}
	defer file.Close()
	reader := newTextReader(file)
	header, err := reader.Read()
	if err != nil {
		return TestReport{}, fmt.Errorf("خواندن سرصفحهٔ فایل ممکن نشد: %w", err)
	}
	return TestReport{
		OK: true, MessageFa: fmt.Sprintf("فایل خوانده شد: %d ستون در سرصفحه.", len(header)),
		ServerFa: filepath.Base(cfg.filePath()), Tables: []string{filepath.Base(cfg.filePath())}, TableCount: 1,
		DurationMs: time.Since(started).Milliseconds(),
		WarningsFa: []string{"هر خط فایل یک ردیف داده است؛ سرصفحه باید نام ستون‌ها باشد."},
	}, nil
}

func (s csvSource) Tables(ctx context.Context, cfg Config) ([]TableInfo, error) {
	file, err := openTextFile(cfg.filePath())
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := newTextReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	cleaned := trimAll(header)
	columns := make([]Column, 0, len(cleaned))
	for _, name := range cleaned {
		columns = append(columns, Column{Name: name})
	}
	return []TableInfo{{Name: filepath.Base(cfg.filePath()), KindFa: "فایل", Columns: columns, TitleFa: "برگهٔ فایل"}}, nil
}

func (s csvSource) Preview(ctx context.Context, cfg Config, table string, limit int) (Result, error) {
	return s.Query(ctx, cfg, "", limit)
}

func (s csvSource) Query(ctx context.Context, cfg Config, statement string, limit int) (Result, error) {
	started := time.Now()
	file, err := openTextFile(cfg.filePath())
	if err != nil {
		return Result{}, err
	}
	defer file.Close()
	reader := newTextReader(file)
	header, err := reader.Read()
	if err != nil {
		return Result{}, err
	}
	result := Result{
		Columns: trimAll(header), SourceFa: filepath.Base(cfg.filePath()),
		NoteFa: "فایل متنی خوانده شد؛ جداکننده خودکار تشخیص داده می‌شود.",
	}
	limit = limitOf(limit, cfg)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return result, fmt.Errorf("خط %d فایل خوانده نشد: %w", len(result.Rows)+2, err)
		}
		result.Rows = append(result.Rows, padRow(record, len(result.Columns)))
		if len(result.Rows) >= limit {
			result.Truncated = true
			break
		}
	}
	result.DurationMs = time.Since(started).Milliseconds()
	return result, nil
}

// xlsxSource منبع دادهٔ فایل اکسل.
type xlsxSource struct{}

func newXLSXSource() Source { return xlsxSource{} }

func (xlsxSource) Kind() Kind         { return KindXLSX }
func (xlsxSource) TitleFa() string    { return "فایل اکسل (xlsx)" }
func (xlsxSource) CategoryFa() string { return "فایل" }

func (xlsxSource) Fields() []Field {
	return []Field{
		{Key: "filePath", TitleFa: "فایل", Type: "text", Required: true,
			Example: "uploads/data.xlsx", NoteFa: "فایلی که از همین صفحه بارگذاری کرده‌اید."},
		{Key: "sheet", TitleFa: "برگه", Type: "text", Required: false,
			Example: "کارکنان", NoteFa: "خالی بگذارید تا برگهٔ نخست خوانده شود."},
	}
}

func (s xlsxSource) Test(ctx context.Context, cfg Config) (TestReport, error) {
	started := time.Now()
	book, err := excelize.OpenFile(cfg.filePath())
	if err != nil {
		return TestReport{}, fmt.Errorf("بازکردن فایل اکسل ممکن نشد: %w", err)
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return TestReport{}, errors.New("فایل اکسل هیچ برگه‌ای ندارد")
	}
	return TestReport{
		OK: true, MessageFa: fmt.Sprintf("فایل اکسل خوانده شد: %d برگه.", len(sheets)),
		ServerFa: filepath.Base(cfg.filePath()), Tables: sheets, TableCount: len(sheets),
		DurationMs: time.Since(started).Milliseconds(),
		WarningsFa: []string{"هر برگهٔ فایل می‌تواند یک جدول هدف باشد؛ نام برگه را در پیکربندی بدهید."},
	}, nil
}

func (s xlsxSource) Tables(ctx context.Context, cfg Config) ([]TableInfo, error) {
	book, err := excelize.OpenFile(cfg.filePath())
	if err != nil {
		return nil, fmt.Errorf("بازکردن فایل اکسل ممکن نشد: %w", err)
	}
	defer book.Close()
	out := []TableInfo{}
	for _, sheet := range book.GetSheetList() {
		rows, err := book.Rows(sheet)
		if err != nil {
			continue
		}
		columns := []Column{}
		if rows.Next() {
			if header, err := rows.Columns(); err == nil {
				for _, name := range header {
					columns = append(columns, Column{Name: strings.TrimSpace(name)})
				}
			}
		}
		rows.Close()
		out = append(out, TableInfo{Name: sheet, KindFa: "برگه", TitleFa: "برگهٔ اکسل", Columns: columns})
	}
	return out, nil
}

func (s xlsxSource) Preview(ctx context.Context, cfg Config, table string, limit int) (Result, error) {
	started := time.Now()
	book, err := excelize.OpenFile(cfg.filePath())
	if err != nil {
		return Result{}, fmt.Errorf("بازکردن فایل اکسل ممکن نشد: %w", err)
	}
	defer book.Close()
	sheet := strings.TrimSpace(table)
	if sheet == "" {
		sheet = strings.TrimSpace(cfg.Sheet)
	}
	if sheet == "" {
		lists := book.GetSheetList()
		if len(lists) == 0 {
			return Result{}, errors.New("فایل اکسل هیچ برگه‌ای ندارد")
		}
		sheet = lists[0]
	}
	rows, err := book.Rows(sheet)
	if err != nil {
		return Result{}, fmt.Errorf("برگهٔ «%s» خوانده نشد: %w", sheet, err)
	}
	defer rows.Close()

	result := Result{SourceFa: filepath.Base(cfg.filePath()) + " / " + sheet, StatementFa: "برگهٔ «" + sheet + "»"}
	limit = limitOf(limit, cfg)
	for rows.Next() {
		values, err := rows.Columns()
		if err != nil {
			return result, err
		}
		if result.Columns == nil {
			result.Columns = trimAll(values)
			continue
		}
		result.Rows = append(result.Rows, padRow(values, len(result.Columns)))
		if len(result.Rows) >= limit {
			result.Truncated = true
			break
		}
	}
	if result.Columns == nil {
		return result, errors.New("برگهٔ انتخابی خالی است")
	}
	result.DurationMs = time.Since(started).Milliseconds()
	return result, nil
}

func (s xlsxSource) Query(ctx context.Context, cfg Config, statement string, limit int) (Result, error) {
	// در منبع فایلی «پرس‌وجو» معنایی ندارد؛ برگه خوانده می‌شود.
	return s.Preview(ctx, cfg, cfg.Sheet, limit)
}

/* ----------------------------- منابع پایگاه‌داده ----------------------------- */

// sqlDialect تفاوت‌های هر پایگاه‌داده در یک ساختار.
type sqlDialect struct {
	Kind        Kind
	TitleFa     string
	Driver      string
	DefaultPort string
}

// sqlSource منبع دادهٔ پایگاه‌داده (مشترک برای Postgres/SQL Server/MySQL/Oracle).
type sqlSource struct{ dialect sqlDialect }

func newSQLSource(dialect sqlDialect) Source { return sqlSource{dialect: dialect} }

func (s sqlSource) Kind() Kind         { return s.dialect.Kind }
func (s sqlSource) TitleFa() string    { return s.dialect.TitleFa }
func (s sqlSource) CategoryFa() string { return "پایگاه‌داده" }

func (s sqlSource) Fields() []Field {
	return []Field{
		{Key: "name", TitleFa: "نام اتصال", Type: "text", Required: true, Example: "سامانهٔ قدیم حقوق"},
		{Key: "host", TitleFa: "میزبان", Type: "text", Required: true, Example: "192.168.1.20"},
		{Key: "port", TitleFa: "درگاه", Type: "number", Required: false, Example: s.dialect.DefaultPort},
		{Key: "database", TitleFa: "پایگاه‌داده / سرویس", Type: "text", Required: true, Example: "HRMS"},
		{Key: "schema", TitleFa: "شِما (اختیاری)", Type: "text", Required: false, Example: "public"},
		{Key: "user", TitleFa: "کاربر", Type: "text", Required: true, Example: "readonly_user"},
		{Key: "secretEnv", TitleFa: "متغیر محیطی رمز", Type: "text", Required: false,
			Example: "HRMS_SOURCE_PASSWORD", NoteFa: "رمز در رکورد ذخیره نمی‌شود؛ نام متغیر محیطی نگه داشته می‌شود."},
	}
}

func (s sqlSource) Test(ctx context.Context, cfg Config) (TestReport, error) {
	started := time.Now()
	ctx, cancel := WithTimeout(ctx, cfg)
	defer cancel()
	db, err := s.open(ctx, cfg)
	if err != nil {
		return TestReport{}, err
	}
	defer db.Close()

	report := TestReport{OK: true, ServerFa: s.dialect.TitleFa}
	if version, err := querySingle(ctx, db, s.versionQuery()); err == nil {
		report.VersionFa = firstLine(version)
	}
	tables, err := s.tablesWithDB(ctx, db, cfg)
	if err != nil {
		report.WarningsFa = append(report.WarningsFa,
			"فهرست جدول‌ها خوانده نشد (کاربر ممکن است دسترسی نداشته باشد): "+err.Error())
	} else {
		report.TableCount = len(tables)
		names := make([]string, 0, len(tables))
		for _, table := range tables {
			names = append(names, table.Name)
			if len(names) >= 40 {
				break
			}
		}
		report.Tables = names
	}
	report.MessageFa = fmt.Sprintf("اتصال برقرار شد؛ %d جدول دیده شد.", report.TableCount)
	report.DurationMs = time.Since(started).Milliseconds()
	report.WarningsFa = append(report.WarningsFa,
		"برای بارگذاری داده، حساب اتصال فقط باید حق خواندن داشته باشد (توصیهٔ امنیتی).")
	return report, nil
}

func (s sqlSource) Tables(ctx context.Context, cfg Config) ([]TableInfo, error) {
	ctx, cancel := WithTimeout(ctx, cfg)
	defer cancel()
	db, err := s.open(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return s.tablesWithDB(ctx, db, cfg)
}

func (s sqlSource) tablesWithDB(ctx context.Context, db *sql.DB, cfg Config) ([]TableInfo, error) {
	statement, args := s.tablesQuery(cfg)
	rows, err := db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TableInfo{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return out, err
		}
		out = append(out, TableInfo{Name: name, KindFa: "جدول", TitleFa: "جدول"})
		if len(out) >= 2000 {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, rows.Err()
}

func (s sqlSource) Preview(ctx context.Context, cfg Config, table string, limit int) (Result, error) {
	identifier, err := safeIdentifier(table)
	if err != nil {
		return Result{}, err
	}
	limit = limitOf(limit, cfg)
	statement := s.previewQuery(identifier, limit)
	result, err := s.Query(ctx, cfg, statement, limit)
	if err != nil {
		return result, err
	}
	result.StatementFa = "SELECT از " + identifier
	return result, nil
}

func (s sqlSource) Query(ctx context.Context, cfg Config, statement string, limit int) (Result, error) {
	started := time.Now()
	if strings.TrimSpace(statement) == "" {
		return Result{}, errors.New("دستور خالی است؛ یک پرس‌وجوی خواندنی بدهید")
	}
	if err := IsReadOnlyStatement(statement); err != nil {
		return Result{}, err
	}
	ctx, cancel := WithTimeout(ctx, cfg)
	defer cancel()
	db, err := s.open(ctx, cfg)
	if err != nil {
		return Result{}, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, statement)
	if err != nil {
		return Result{}, fmt.Errorf("اجرای پرس‌وجو ممکن نشد: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return Result{}, err
	}
	result := Result{Columns: columns, SourceFa: s.dialect.TitleFa + " / " + strings.TrimSpace(cfg.DBName)}
	limit = limitOf(limit, cfg)
	for rows.Next() {
		holders := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range holders {
			pointers[i] = &holders[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return result, err
		}
		result.Rows = append(result.Rows, stringifyRow(holders))
		if len(result.Rows) >= limit {
			result.Truncated = true
			break
		}
	}
	result.DurationMs = time.Since(started).Milliseconds()
	return result, rows.Err()
}

// open اتصال پایگاه‌داده را با DSN ساختهٔ همین بسته باز می‌کند.
func (s sqlSource) open(ctx context.Context, cfg Config) (*sql.DB, error) {
	dsn, err := s.dsn(cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(s.dialect.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("ساخت اتصال ممکن نشد: %w", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(2 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("اتصال به %s برقرار نشد: %w", s.dialect.TitleFa, err)
	}
	return db, nil
}

// dsn رشتهٔ اتصال هر پایگاه‌داده را می‌سازد (رمز از بدنه یا متغیر محیطی).
func (s sqlSource) dsn(cfg Config) (string, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return "", errors.New("میزبان خالی است")
	}
	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = s.dialect.DefaultPort
	}
	database := strings.TrimSpace(cfg.DBName)
	if database == "" {
		return "", errors.New("نام پایگاه‌داده/سرویس خالی است")
	}
	user := strings.TrimSpace(cfg.User)
	password := ResolvePassword(cfg)

	switch s.dialect.Kind {
	case KindPostgres:
		sslMode := optionOf(cfg, "sslmode", "disable")
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			url.QueryEscape(user), url.QueryEscape(password), host, port, url.PathEscape(database), sslMode), nil
	case KindMySQL:
		params := optionOf(cfg, "params", "parseTime=true&charset=utf8mb4")
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, password, host, port, database, params), nil
	case KindSQLServer:
		encrypt := optionOf(cfg, "encrypt", "disable")
		return fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s&encrypt=%s",
			url.QueryEscape(user), url.QueryEscape(password), host, port, url.QueryEscape(database), encrypt), nil
	case KindOracle:
		return fmt.Sprintf("oracle://%s:%s@%s:%s/%s",
			url.QueryEscape(user), url.QueryEscape(password), host, port, strings.TrimPrefix(database, "/")), nil
	}
	return "", fmt.Errorf("ساخت DSN برای %s پیاده‌سازی نشده است", s.dialect.Kind)
}

// tablesQuery پرس‌وجوی فهرست جدول‌ها در هر گویش.
func (s sqlSource) tablesQuery(cfg Config) (string, []any) {
	schema := strings.TrimSpace(cfg.Schema)
	switch s.dialect.Kind {
	case KindPostgres:
		return `SELECT table_name FROM information_schema.tables
		        WHERE table_schema = COALESCE(NULLIF($1, ''), 'public')
		          AND table_type IN ('BASE TABLE','VIEW') ORDER BY table_name`, []any{schema}
	case KindMySQL:
		return `SELECT table_name FROM information_schema.tables
		        WHERE table_schema = COALESCE(NULLIF(?, ''), DATABASE())
		        ORDER BY table_name`, []any{schema}
	case KindSQLServer:
		return `SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES
		        WHERE TABLE_SCHEMA = COALESCE(NULLIF(@p1, ''), 'dbo') ORDER BY TABLE_NAME`, []any{schema}
	case KindOracle:
		return `SELECT table_name FROM all_tables
		        WHERE owner = UPPER(COALESCE(NULLIF(:1, ''), USER)) ORDER BY table_name`, []any{schema}
	}
	return "", nil
}

// versionQuery پرس‌وجوی نسخهٔ سرور.
func (s sqlSource) versionQuery() string {
	switch s.dialect.Kind {
	case KindPostgres:
		return "SELECT version()"
	case KindMySQL:
		return "SELECT VERSION()"
	case KindSQLServer:
		return "SELECT @@VERSION"
	case KindOracle:
		return "SELECT banner FROM v$version WHERE ROWNUM = 1"
	}
	return ""
}

// previewQuery پرس‌وجوی پیش‌نمایش با سقف ردیف هر گویش.
func (s sqlSource) previewQuery(identifier string, limit int) string {
	switch s.dialect.Kind {
	case KindSQLServer:
		return fmt.Sprintf("SELECT TOP %d * FROM %s", limit, identifier)
	case KindOracle:
		return fmt.Sprintf("SELECT * FROM %s WHERE ROWNUM <= %d", identifier, limit)
	}
	return fmt.Sprintf("SELECT * FROM %s LIMIT %d", identifier, limit)
}

/* --------------------------------- کمکی‌ها --------------------------------- */

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$#]*(\.[A-Za-z_][A-Za-z0-9_$#]*)?$`)

// safeIdentifier نام جدول/نما را می‌سنجد (جلوگیری از تزریق در پرس‌وجوی ساختهٔ ما).
func safeIdentifier(name string) (string, error) {
	value := strings.TrimSpace(name)
	if value == "" {
		return "", errors.New("نام جدول خالی است")
	}
	if !identifierPattern.MatchString(value) {
		return "", fmt.Errorf("نام «%s» معتبر نیست؛ برای نام‌های خاص از پرس‌وجوی دستی استفاده کنید", name)
	}
	return value, nil
}

// openTextFile فایل متنی را برای خواندن باز می‌کند (با بررسی مسیر).
func openTextFile(path string) (*os.File, error) {
	value := strings.TrimSpace(path)
	if value == "" {
		return nil, errors.New("مسیر فایل خالی است")
	}
	info, err := os.Stat(value)
	if err != nil {
		return nil, fmt.Errorf("فایل «%s» پیدا نشد: %w", value, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("مسیر «%s» پوشه است، نه فایل", value)
	}
	return os.Open(value)
}

// newTextReader خوانندهٔ CSV با تشخیص خودکار جداکننده (ویرگول، نقطه‌ویرگول، تب یا خط عمودی).
//
// چرا خودکار: خروجی سامانه‌های قدیمی و Excel فارسی معمولاً نقطه‌ویرگول دارد؛ کاربر نباید
// برای این جزئیات تنظیم کند. سرصفحه با Peek خوانده می‌شود، پس بایت‌ها مصرف نمی‌شوند.
func newTextReader(reader io.Reader) *csv.Reader {
	buffered := bufio.NewReader(reader)
	comma := ','
	if head, _ := buffered.Peek(4096); len(head) > 0 {
		line := string(head)
		if index := strings.IndexAny(line, "\r\n"); index >= 0 {
			line = line[:index]
		}
		comma = detectComma(line)
	}
	csvReader := csv.NewReader(buffered)
	csvReader.Comma = comma
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true
	return csvReader
}

// detectComma جداکنندهٔ محتمل سرصفحه را برمی‌گرداند (بیشترین تکرار).
func detectComma(header string) rune {
	best, bestCount := ',', 0
	for _, candidate := range []rune{',', ';', '\t', '|'} {
		if count := strings.Count(header, string(candidate)); count > bestCount {
			best, bestCount = candidate, count
		}
	}
	return best
}

// trimAll اقلام را می‌پیراید و BOM آغاز فایل را برمی‌دارد.
func trimAll(values []string) []string {
	out := make([]string, 0, len(values))
	for i, value := range values {
		item := strings.TrimSpace(value)
		if i == 0 {
			item = strings.TrimPrefix(item, "\ufeff")
		}
		out = append(out, item)
	}
	return out
}

// padRow ردیف را به اندازهٔ ستون‌ها می‌رساند (ردیف کوتاه/بلند، خطا نمی‌سازد).
func padRow(values []string, size int) []string {
	out := make([]string, size)
	for i := 0; i < size; i++ {
		if i < len(values) {
			out[i] = strings.TrimSpace(strings.TrimPrefix(values[i], "\ufeff"))
		}
	}
	return out
}

// stringifyRow مقادیر پایگاه‌داده را به متن تبدیل می‌کند (تبدیل نوع در لایهٔ نگاشت).
func stringifyRow(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case nil:
			out = append(out, "")
		case []byte:
			out = append(out, strings.TrimSpace(string(typed)))
		case time.Time:
			out = append(out, typed.Format("2006-01-02 15:04:05"))
		default:
			out = append(out, strings.TrimSpace(fmt.Sprintf("%v", typed)))
		}
	}
	return out
}

// querySingle نخستین مقدار یک پرس‌وجوی تک‌مقداری.
func querySingle(ctx context.Context, db *sql.DB, statement string) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", errors.New("پرس‌وجو خالی است")
	}
	var value sql.NullString
	if err := db.QueryRowContext(ctx, statement).Scan(&value); err != nil {
		return "", err
	}
	return value.String, nil
}

// firstLine نخستین خط یک متن چندخطی.
func firstLine(value string) string {
	if idx := strings.IndexAny(value, "\r\n"); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}

// limitOf سقف ردیف را از ورودی یا پیکربندی می‌گیرد.
func limitOf(limit int, cfg Config) int {
	if limit > 0 {
		if limit > MaxMaxRows {
			return MaxMaxRows
		}
		return limit
	}
	return cfg.MaxRowsOf()
}

// optionOf مقدار یک گزینهٔ پیکربندی با پیش‌فرض.
func optionOf(cfg Config, key, fallback string) string {
	if cfg.Options == nil {
		return fallback
	}
	if value := strings.TrimSpace(cfg.Options[key]); value != "" {
		return value
	}
	return fallback
}

// ParseLimit عدد سقف ردیف را از متن می‌خواند (برای پارامتر پرس‌وجو).
func ParseLimit(raw string) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return number
}
