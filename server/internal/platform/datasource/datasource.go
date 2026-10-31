// Package datasource رابط بازِ اتصال به منابع داده بیرونی و تبادل داده.
//
// هدف: یک «رابط» مشترک که هر منبع داده (فایل اکسل، CSV، Postgres، SQL Server، MySQL،
// Oracle و هر درایوری که بعداً اضافه شود) پشت آن بنشیند؛ مصرف‌کنندهٔ رابط (ماژول
// یکپارچه‌سازی) نه درایور را می‌شناسد و نه DSN می‌سازد.
//
// قاعده‌های ایمنی (چون این رابط به دادهٔ سازمانی دسترسی دارد):
//   - **رمز هرگز ذخیره و لاگ نمی‌شود**؛ پیکربندی فقط نام متغیر محیطی رمز را نگه می‌دارد
//     (`secretEnv`) و مقدار رمز فقط در همان درخواست از بدنه یا محیط خوانده می‌شود.
//   - پیش‌فرض **فقط‌خواندنی** است: `Query` تنها دستور `SELECT`/`WITH` را می‌پذیرد و
//     دستور تغییردهنده رد می‌شود؛ تعداد ردیف و مهلت اجرا هم سقف دارد.
//   - هر اتصال و هر خواندن، خلاصهٔ عملش (بدون دادهٔ حساس) برگردانده می‌شود تا در
//     لاگ حسابرسی ثبت شود.
package datasource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Kind شناسهٔ نوع منبع داده.
type Kind string

// انواع پشتیبانی‌شده.
const (
	KindXLSX      Kind = "xlsx"
	KindCSV       Kind = "csv"
	KindPostgres  Kind = "postgres"
	KindSQLServer Kind = "sqlserver"
	KindMySQL     Kind = "mysql"
	KindOracle    Kind = "oracle"
)

// سقف‌های پیش‌فرض خواندن.
const (
	DefaultMaxRows        = 500
	MaxMaxRows            = 20_000
	DefaultTimeoutSeconds = 20
	MaxTimeoutSeconds     = 120
)

// Field یک فیلد پیکربندی اتصال (برای فرم رابط و برای سنجش ورودی).
type Field struct {
	Key      string `json:"key"`
	TitleFa  string `json:"titleFa"`
	Type     string `json:"type"` // text | number | password | bool
	Required bool   `json:"required"`
	Example  string `json:"example,omitempty"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// Config پیکربندی یک اتصال. رمز در این ساختار نگه داشته نمی‌شود مگر برای همان درخواست.
type Config struct {
	Kind   Kind   `json:"kind"`
	Name   string `json:"name"`
	Host   string `json:"host"`
	Port   string `json:"port"`
	DBName string `json:"database"`
	Schema string `json:"schema"`
	// FilePath مسیر فایل برای منابع فایلی (CSV/Excel) روی سرور.
	FilePath string `json:"filePath"`
	// Sheet نام برگه در فایل اکسل؛ خالی یعنی برگهٔ نخست.
	Sheet string `json:"sheet"`
	User  string `json:"user"`
	// SecretEnv نام متغیر محیطی که رمز از آن خوانده می‌شود (روش پیشنهادی).
	SecretEnv string `json:"secretEnv"`
	// Password رمز در همان درخواست؛ هرگز ذخیره یا برگردانده نمی‌شود.
	Password string `json:"password,omitempty"`
	// ReadOnly پیش‌فرض true است؛ false فقط با تأیید صریح پذیرفته می‌شود.
	ReadOnly bool `json:"readOnly"`
	// MaxRows سقف ردیف هر خواندن و TimeoutSeconds مهلت اجرا.
	MaxRows        int `json:"maxRows"`
	TimeoutSeconds int `json:"timeoutSeconds"`
	// Options گزینه‌های آزاد درایور (مثل sslmode در Postgres، یا جداکنندهٔ CSV).
	Options map[string]string `json:"options,omitempty"`
}

// EffectiveReadOnly می‌گوید اتصال فقط‌خواندنی است یا نه (پیش‌فرض: هست).
func (c Config) EffectiveReadOnly() bool { return true }

// MaxRowsOf سقف ردیف مؤثر.
func (c Config) MaxRowsOf() int {
	if c.MaxRows <= 0 {
		return DefaultMaxRows
	}
	if c.MaxRows > MaxMaxRows {
		return MaxMaxRows
	}
	return c.MaxRows
}

// Timeout مهلت مؤثر اجرا.
func (c Config) Timeout() time.Duration {
	seconds := c.TimeoutSeconds
	if seconds <= 0 {
		seconds = DefaultTimeoutSeconds
	}
	if seconds > MaxTimeoutSeconds {
		seconds = MaxTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

// filePath پاک‌شدهٔ فایل منبع فایلی.
func (c Config) filePath() string { return strings.TrimSpace(c.FilePath) }

// Column یک ستون منبع داده.
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Nullable bool   `json:"nullable,omitempty"`
}

// Result خروجی یک خواندن (پیش‌نمایش یا استخراج): همیشه متنی است تا تبدیل نوع در لایهٔ
// نگاشت (map/import) انجام شود، نه در درایور.
type Result struct {
	Columns     []string   `json:"columns"`
	Rows        [][]string `json:"rows"`
	RowCount    int        `json:"rowCount"`
	Truncated   bool       `json:"truncated"`
	SourceFa    string     `json:"sourceFa"`
	StatementFa string     `json:"statementFa,omitempty"`
	DurationMs  int64      `json:"durationMs"`
	NoteFa      string     `json:"noteFa,omitempty"`
}

// TableInfo یک جدول/برگه/نمای منبع.
type TableInfo struct {
	Name    string   `json:"name"`
	TitleFa string   `json:"titleFa,omitempty"`
	Rows    int64    `json:"rows,omitempty"`
	Columns []Column `json:"columns,omitempty"`
	KindFa  string   `json:"kindFa,omitempty"` // جدول | نما | برگه | فایل
}

// TestReport نتیجهٔ آزمون اتصال.
type TestReport struct {
	OK         bool     `json:"ok"`
	MessageFa  string   `json:"messageFa"`
	ServerFa   string   `json:"serverFa,omitempty"`
	VersionFa  string   `json:"versionFa,omitempty"`
	Tables     []string `json:"tables,omitempty"`
	TableCount int      `json:"tableCount,omitempty"`
	DurationMs int64    `json:"durationMs"`
	WarningsFa []string `json:"warningsFa,omitempty"`
}

// Source رابط مشترک همهٔ منابع داده.
//
// هر درایور تازه فقط همین چند متد را پیاده می‌کند و بی‌درنگ در فهرست انواع و در فرم
// رابط دیده می‌شود؛ هیچ مصرف‌کننده‌ای به کد آن دست نمی‌زند.
type Source interface {
	// Kind شناسهٔ نوع.
	Kind() Kind
	// TitleFa نام فارسی نوع.
	TitleFa() string
	// CategoryFa دستهٔ نمایشی (فایل | پایگاه‌داده).
	CategoryFa() string
	// Fields فیلدهای پیکربندی همین نوع (برای فرم و سنجش ورودی).
	Fields() []Field
	// Test اتصال را می‌آزماید.
	Test(ctx context.Context, cfg Config) (TestReport, error)
	// Tables فهرست جدول‌ها/برگه‌ها را برمی‌گرداند (برای انتخاب در رابط).
	Tables(ctx context.Context, cfg Config) ([]TableInfo, error)
	// Preview چند ردیف نخست یک جدول/برگه را می‌خواند.
	Preview(ctx context.Context, cfg Config, table string, limit int) (Result, error)
	// Query یک پرس‌وجوی فقط‌خواندنی را اجرا می‌کند (SELECT/WITH).
	Query(ctx context.Context, cfg Config, statement string, limit int) (Result, error)
}

// KindInfo توصیف یک نوع منبع داده در فهرست رابط.
type KindInfo struct {
	Kind       Kind    `json:"kind"`
	TitleFa    string  `json:"titleFa"`
	CategoryFa string  `json:"categoryFa"`
	Fields     []Field `json:"fields"`
	NoteFa     string  `json:"noteFa,omitempty"`
}

var registry = map[Kind]Source{}

// Register یک درایور را در فهرست رابط ثبت می‌کند (در drivers.go صدا زده می‌شود).
func Register(source Source) {
	if source == nil || source.Kind() == "" {
		return
	}
	registry[source.Kind()] = source
}

// Get درایور یک نوع را برمی‌گرداند.
func Get(kind Kind) (Source, bool) {
	source, ok := registry[Kind(strings.ToLower(strings.TrimSpace(string(kind))))]
	return source, ok
}

// Catalog فهرست انواع پشتیبانی‌شده با فیلدهای پیکربندی؛ برای ساخت فرم در رابط.
func Catalog() []KindInfo {
	out := make([]KindInfo, 0, len(registry))
	for _, source := range registry {
		out = append(out, KindInfo{
			Kind:       source.Kind(),
			TitleFa:    source.TitleFa(),
			CategoryFa: source.CategoryFa(),
			Fields:     source.Fields(),
			NoteFa:     noteOf(source.Kind()),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CategoryFa != out[j].CategoryFa {
			return out[i].CategoryFa < out[j].CategoryFa
		}
		return out[i].TitleFa < out[j].TitleFa
	})
	return out
}

// noteOf یادداشت فارسی هر نوع (راهنمای کوتاه در فرم).
func noteOf(kind Kind) string {
	switch kind {
	case KindXLSX:
		return "فایل اکسل (xlsx) را می‌توانید از همین صفحه بارگذاری کنید؛ هر برگهٔ فایل یک «جدول» است."
	case KindCSV:
		return "فایل متنی جداشده با ویرگول یا نقطه‌ویرگول؛ سبک‌ترین راه انتقال داده از هر سامانه‌ای."
	case KindPostgres:
		return "اتصال مستقیم به Postgres (همان درایور استقرار سامانه)."
	case KindSQLServer:
		return "اتصال مستقیم به Microsoft SQL Server (بدون نیاز به ODBC)."
	case KindMySQL:
		return "اتصال مستقیم به MySQL/MariaDB."
	case KindOracle:
		return "اتصال مستقیم به Oracle (بدون نیاز به کلاینت نصب‌شده)."
	}
	return ""
}

// ResolvePassword رمز اتصال را برمی‌گرداند: اول از بدنهٔ درخواست، وگرنه از متغیر محیطی
// نام‌برده‌شده در SecretEnv. اگر هیچ‌کدام نبود، رشتهٔ خالی برمی‌گردد (اتصال‌های بی‌رمز
// مثل فایل مجاز است).
func ResolvePassword(cfg Config) string {
	if password := strings.TrimSpace(cfg.Password); password != "" {
		return password
	}
	if name := strings.TrimSpace(cfg.SecretEnv); name != "" {
		return os.Getenv(name)
	}
	return ""
}

// Validate پیکربندی را پیش از استفاده می‌سنجد.
func Validate(cfg Config) error {
	source, ok := Get(cfg.Kind)
	if !ok {
		return fmt.Errorf("نوع منبع داده «%s» پشتیبانی نمی‌شود", cfg.Kind)
	}
	for _, field := range source.Fields() {
		if !field.Required || field.Type == "bool" {
			continue
		}
		if valueOf(cfg, field.Key) == "" {
			return fmt.Errorf("فیلد «%s» الزامی است", field.TitleFa)
		}
	}
	if !cfg.ReadOnly {
		return errors.New("اتصال باید فقط‌خواندنی باشد؛ ورود داده از راه بارگذاری انجام می‌گیرد، نه نوشتن روی منبع")
	}
	return nil
}

// valueOf مقدار یک فیلد پیکربندی را از کلید آن می‌خواند.
func valueOf(cfg Config, key string) string {
	switch key {
	case "name":
		return strings.TrimSpace(cfg.Name)
	case "host":
		return strings.TrimSpace(cfg.Host)
	case "port":
		return strings.TrimSpace(cfg.Port)
	case "database":
		return strings.TrimSpace(cfg.DBName)
	case "schema":
		return strings.TrimSpace(cfg.Schema)
	case "filePath":
		return cfg.filePath()
	case "sheet":
		return strings.TrimSpace(cfg.Sheet)
	case "user":
		return strings.TrimSpace(cfg.User)
	case "secretEnv":
		return strings.TrimSpace(cfg.SecretEnv)
	}
	return ""
}

// WithTimeout بافت اجرا را با مهلت پیکربندی می‌کند.
func WithTimeout(ctx context.Context, cfg Config) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, cfg.Timeout())
}

// IsReadOnlyStatement می‌سنجد که دستور فقط‌خواندنی است.
//
// قاعده: تنها `SELECT` و `WITH` مجاز است؛ نقطه‌ویرگول چنددستوری، دستورهای تغییردهنده و
// بلوک‌های رویه‌ای رد می‌شوند. این سنجش ساده و محافظه‌کارانه است و به‌جای تکیه بر آن،
// حساب اتصال هم باید فقط‌خواندنی باشد (دو لایه ایمنی).
func IsReadOnlyStatement(statement string) error {
	value := strings.TrimSpace(statement)
	if value == "" {
		return errors.New("دستور خالی است")
	}
	lower := strings.ToLower(value)
	trimmed := strings.TrimRight(lower, "; \t\r\n")
	if strings.Contains(trimmed, ";") {
		return errors.New("اجرای چند دستور در یک درخواست مجاز نیست")
	}
	if !strings.HasPrefix(trimmed, "select") && !strings.HasPrefix(trimmed, "with") {
		return errors.New("تنها پرس‌وجوی خواندنی (SELECT یا WITH) مجاز است")
	}
	forbidden := []string{
		" insert ", " update ", " delete ", " drop ", " alter ", " create ", " truncate ",
		" grant ", " revoke ", " merge ", " call ", " execute ", " exec ", " into ",
	}
	padded := " " + trimmed + " "
	for _, word := range forbidden {
		if strings.Contains(padded, word) {
			return fmt.Errorf("واژهٔ «%s» در پرس‌وجو مجاز نیست (اتصال فقط‌خواندنی است)", strings.TrimSpace(word))
		}
	}
	return nil
}

// LimitResult سقف ردیف را اعمال می‌کند و «بریده‌شدن» را گزارش می‌دهد.
func LimitResult(result Result, limit int) Result {
	if limit <= 0 {
		limit = DefaultMaxRows
	}
	if len(result.Rows) > limit {
		result.Rows = result.Rows[:limit]
		result.Truncated = true
	}
	result.RowCount = len(result.Rows)
	return result
}
