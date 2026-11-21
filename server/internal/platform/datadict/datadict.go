// Package datadict «فرهنگ داده» سامانه: ساختار درختی طرح‌ها/جدول‌ها/ستون‌ها، کلیدها،
// روابط خارجی و نگاشت آن‌ها به ماژول‌های سامانه — خوانده‌شده از کاتالوگ زندهٔ Postgres.
//
// چرا از کاتالوگ و نه از فایل ایستا: ساختار داده با هر مهاجرت عوض می‌شود؛ اگر فرهنگ
// داده دستی نگه داشته شود، از واقعیت عقب می‌ماند. این بسته در هر درخواست، تصویر
// لحظه‌ایِ پایگاه‌داده را می‌سازد، پس «بروزرسانی» یعنی همین درخواست — بدون بازتولید فایل.
//
// بسته کاملاً مستقل از دامنه است: نه به ماژولی وابسته است و نه به رجیستری؛ نام فارسی
// ماژول‌ها از بیرون تزریق می‌شود (moduleTitles).
package datadict

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Dictionary فرهنگ دادهٔ کامل یک استقرار.
type Dictionary struct {
	GeneratedAt string     `json:"generatedAt"`
	Database    Database   `json:"database"`
	Summary     Summary    `json:"summary"`
	Schemas     []Schema   `json:"schemas"`
	Relations   []Relation `json:"relations"`
	Modules     []Module   `json:"modules"`
	NoteFa      string     `json:"noteFa"`
}

// Database شناسهٔ پایگاه‌داده‌ای که فرهنگ از آن خوانده شده است.
type Database struct {
	Name    string `json:"name"`
	Server  string `json:"server"`
	Current string `json:"currentUser"`
	SizeKb  int64  `json:"sizeKb"`
}

// Summary شمارش کل برای نگاه سریع.
type Summary struct {
	Schemas   int   `json:"schemas"`
	Tables    int   `json:"tables"`
	Columns   int   `json:"columns"`
	Relations int   `json:"relations"`
	Rows      int64 `json:"rows"`
}

// Schema یک طرح (schema) پایگاه‌داده؛ در این سامانه هر ماژول یک طرح دارد.
type Schema struct {
	Name     string  `json:"name"`
	TitleFa  string  `json:"titleFa"`
	ModuleID string  `json:"moduleId,omitempty"`
	Platform bool    `json:"platform,omitempty"`
	Tables   []Table `json:"tables"`
}

// Table یک جدول پایگاه‌داده با ستون‌ها، کلیدها و شمار سطرهای لحظه‌ای.
type Table struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Rows        int64    `json:"rows"`
	PrimaryKey  []string `json:"primaryKey,omitempty"`
	Columns     []Column `json:"columns"`
	ForeignKeys []string `json:"foreignKeys,omitempty"`
}

// Column یک ستون با نوع، قیدها و توضیح (از COMMENT پایگاه‌داده، اگر باشد).
type Column struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	TypeFa      string `json:"typeFa"`
	Nullable    bool   `json:"nullable"`
	Default     string `json:"default,omitempty"`
	PrimaryKey  bool   `json:"primaryKey,omitempty"`
	ForeignKey  bool   `json:"foreignKey,omitempty"`
	Description string `json:"description,omitempty"`
}

// Relation یک رابطهٔ خارجی: از ستون(های) یک جدول به ستون(های) جدول دیگر.
type Relation struct {
	Name       string   `json:"name"`
	From       string   `json:"from"`
	FromColumn []string `json:"fromColumns"`
	To         string   `json:"to"`
	ToColumn   []string `json:"toColumns"`
	OnDelete   string   `json:"onDelete"`
	Required   bool     `json:"required"`
}

// Module نگاشت ماژول سامانه به دادهٔ ذخیره‌شدهٔ آن: پیشوند API، طرح، موجودیت‌ها و صفحه‌ها.
type Module struct {
	ID        string   `json:"id"`
	TitleFa   string   `json:"titleFa"`
	APIPrefix string   `json:"apiPrefix"`
	DBShema   string   `json:"dbSchema"`
	Entities  []string `json:"entities,omitempty"`
	Screens   []string `json:"screens,omitempty"`
	Tables    []string `json:"tables,omitempty"`
}

// ModuleInfo اطلاعاتی است که لایهٔ بالاتر (رجیستری) برای نام‌گذاری فارسی تزریق می‌کند.
type ModuleInfo struct {
	ID        string
	TitleFa   string
	APIPrefix string
	DBSchema  string
	Entities  []string
	Screens   []string
}

// Options تنظیمات ساخت فرهنگ داده.
type Options struct {
	// Stable برای فایل‌های نسخه‌دار: بدون مُهر زمان و بدون شمار سطرها، تا تغییر «داده»
	// فایل ساختار را کهنه نکند و دروازهٔ هم‌خوانی معنادار بماند.
	Stable bool
}

// Load تصویر لحظه‌ای فرهنگ داده را از کاتالوگ پایگاه‌داده می‌سازد.
func Load(ctx context.Context, db *sql.DB, modules []ModuleInfo, options Options) (Dictionary, error) {
	if db == nil {
		return Dictionary{}, fmt.Errorf("فرهنگ داده به اتصال پایگاه‌داده نیاز دارد")
	}
	dict := Dictionary{
		GeneratedAt: time.Now().Format(time.RFC3339),
		NoteFa: "این فرهنگ از کاتالوگ زندهٔ پایگاه‌داده ساخته می‌شود؛ پس از هر مهاجرت، " +
			"درخواست تازه همین ساختار را نشان می‌دهد. فیلدهای سطح فرم هر ماژول از " +
			"«/api/v1/<module>/schema» خوانده می‌شود.",
	}
	if options.Stable {
		dict.GeneratedAt = ""
	}

	if err := loadDatabase(ctx, db, &dict); err != nil {
		return Dictionary{}, err
	}
	columns, err := loadColumns(ctx, db)
	if err != nil {
		return Dictionary{}, err
	}
	primary, err := loadPrimaryKeys(ctx, db)
	if err != nil {
		return Dictionary{}, err
	}
	relations, err := loadRelations(ctx, db)
	if err != nil {
		return Dictionary{}, err
	}
	counts := map[string]int64{}
	if !options.Stable {
		if counts, err = loadRowCounts(ctx, db); err != nil {
			return Dictionary{}, err
		}
	}

	// نگاشت طرح → ماژول برای نام فارسی و اتصال به صفحه‌ها.
	bySchema := map[string]ModuleInfo{}
	for _, m := range modules {
		if m.DBSchema != "" {
			bySchema[m.DBSchema] = m
		}
	}

	fkColumns := map[string]bool{}
	tableFKs := map[string][]string{}
	for _, r := range relations {
		for _, c := range r.FromColumn {
			fkColumns[r.From+"."+c] = true
		}
		tableFKs[r.From] = append(tableFKs[r.From], r.Name)
	}

	schemaIndex := map[string]*Schema{}
	for _, c := range columns {
		schema, ok := schemaIndex[c.schema]
		if !ok {
			info := bySchema[c.schema]
			schema = &Schema{
				Name:     c.schema,
				TitleFa:  firstNonEmpty(info.TitleFa, schemaTitleFa(c.schema)),
				ModuleID: info.ID,
				Platform: c.schema == "public",
			}
			schemaIndex[c.schema] = schema
		}
		path := c.schema + "." + c.table
		table := findTable(schema, c.table)
		if table == nil {
			schema.Tables = append(schema.Tables, Table{
				Name:        c.table,
				Path:        path,
				Rows:        counts[path],
				PrimaryKey:  primary[path],
				ForeignKeys: tableFKs[path],
			})
			table = &schema.Tables[len(schema.Tables)-1]
		}
		table.Columns = append(table.Columns, Column{
			Name:        c.name,
			Type:        c.dataType,
			TypeFa:      typeLabelFa(c.dataType),
			Nullable:    c.nullable,
			Default:     c.defaultValue,
			PrimaryKey:  contains(primary[path], c.name),
			ForeignKey:  fkColumns[path+"."+c.name],
			Description: c.description,
		})
	}

	for _, schema := range schemaIndex {
		dict.Schemas = append(dict.Schemas, *schema)
		dict.Summary.Schemas++
		dict.Summary.Rows += schemaRows(*schema)
		for _, t := range schema.Tables {
			dict.Summary.Tables++
			dict.Summary.Columns += len(t.Columns)
		}
	}
	sort.Slice(dict.Schemas, func(i, j int) bool {
		if dict.Schemas[i].Platform != dict.Schemas[j].Platform {
			return !dict.Schemas[i].Platform // طرح‌های ماژولی پیش از طرح پلتفرم
		}
		return dict.Schemas[i].Name < dict.Schemas[j].Name
	})
	sort.Slice(relations, func(i, j int) bool {
		if relations[i].From == relations[j].From {
			return relations[i].Name < relations[j].Name
		}
		return relations[i].From < relations[j].From
	})
	dict.Relations = relations
	dict.Summary.Relations = len(relations)

	dict.Modules = moduleMap(modules, dict)
	return dict, nil
}

type rawColumn struct {
	schema, table, name, dataType, defaultValue, description string
	nullable                                                 bool
}

func loadDatabase(ctx context.Context, db *sql.DB, dict *Dictionary) error {
	err := db.QueryRowContext(ctx, `select current_database(), current_user,
		current_setting('server_version'),
		pg_database_size(current_database()) / 1024`).
		Scan(&dict.Database.Name, &dict.Database.Current, &dict.Database.Server, &dict.Database.SizeKb)
	if err != nil {
		return fmt.Errorf("خواندن شناسهٔ پایگاه‌داده: %w", err)
	}
	return nil
}

// loadColumns ستون‌های همهٔ جدول‌های سامانه را می‌خواند (طرح‌های hr_* و طرح پلتفرم).
func loadColumns(ctx context.Context, db *sql.DB) ([]rawColumn, error) {
	rows, err := db.QueryContext(ctx, `
		select n.nspname, cl.relname, a.attname,
		       format_type(a.atttypid, a.atttypmod),
		       not a.attnotnull,
		       coalesce(pg_get_expr(ad.adbin, ad.adrelid), ''),
		       coalesce(col_description(cl.oid, a.attnum), '')
		from pg_attribute a
		join pg_class cl on cl.oid = a.attrelid and cl.relkind in ('r', 'p')
		join pg_namespace n on n.oid = cl.relnamespace
		left join pg_attrdef ad on ad.adrelid = cl.oid and ad.adnum = a.attnum
		where a.attnum > 0 and not a.attisdropped
		  and (n.nspname like 'hr\_%' or n.nspname = 'public')
		order by n.nspname, cl.relname, a.attnum`)
	if err != nil {
		return nil, fmt.Errorf("خواندن ستون‌ها: %w", err)
	}
	defer rows.Close()

	var out []rawColumn
	for rows.Next() {
		var c rawColumn
		if err := rows.Scan(&c.schema, &c.table, &c.name, &c.dataType, &c.nullable, &c.defaultValue, &c.description); err != nil {
			return nil, fmt.Errorf("خواندن سطر ستون: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// loadPrimaryKeys کلیدهای اصلی را به شکل «schema.table → ستون‌ها» برمی‌گرداند.
func loadPrimaryKeys(ctx context.Context, db *sql.DB) (map[string][]string, error) {
	rows, err := db.QueryContext(ctx, `
		select n.nspname, cl.relname, a.attname
		from pg_index i
		join pg_class cl on cl.oid = i.indrelid
		join pg_namespace n on n.oid = cl.relnamespace
		join pg_attribute a on a.attrelid = cl.oid and a.attnum = any(i.indkey)
		where i.indisprimary and (n.nspname like 'hr\_%' or n.nspname = 'public')
		order by n.nspname, cl.relname, a.attnum`)
	if err != nil {
		return nil, fmt.Errorf("خواندن کلیدهای اصلی: %w", err)
	}
	defer rows.Close()

	out := map[string][]string{}
	for rows.Next() {
		var schema, table, column string
		if err := rows.Scan(&schema, &table, &column); err != nil {
			return nil, fmt.Errorf("خواندن سطر کلید اصلی: %w", err)
		}
		path := schema + "." + table
		out[path] = append(out[path], column)
	}
	return out, rows.Err()
}

// contains آیا رشته در فهرست هست.
func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// loadRelations روابط خارجی را می‌خواند (شامل قاعدهٔ حذف و اجباری‌بودن ستون).
func loadRelations(ctx context.Context, db *sql.DB) ([]Relation, error) {
	rows, err := db.QueryContext(ctx, `
		select con.conname,
		       sn.nspname, sc.relname, sa.attname,
		       tn.nspname, tc.relname, ta.attname,
		       con.confdeltype::text, sa.attnotnull
		from pg_constraint con
		join pg_class sc on sc.oid = con.conrelid
		join pg_namespace sn on sn.oid = sc.relnamespace
		join pg_class tc on tc.oid = con.confrelid
		join pg_namespace tn on tn.oid = tc.relnamespace
		join generate_subscripts(con.conkey, 1) s(i) on true
		join pg_attribute sa on sa.attrelid = con.conrelid and sa.attnum = con.conkey[s.i]
		join pg_attribute ta on ta.attrelid = con.confrelid and ta.attnum = con.confkey[s.i]
		where con.contype = 'f'
		order by sn.nspname, sc.relname, con.conname, s.i`)
	if err != nil {
		return nil, fmt.Errorf("خواندن روابط خارجی: %w", err)
	}
	defer rows.Close()

	index := map[string]*Relation{}
	var order []string
	for rows.Next() {
		var name, fromSchema, fromTable, fromColumn, toSchema, toTable, toColumn, deleteRule string
		var required bool
		if err := rows.Scan(&name, &fromSchema, &fromTable, &fromColumn,
			&toSchema, &toTable, &toColumn, &deleteRule, &required); err != nil {
			return nil, fmt.Errorf("خواندن سطر رابطه: %w", err)
		}
		key := fromSchema + "." + fromTable + "." + name
		rel, ok := index[key]
		if !ok {
			rel = &Relation{
				Name:     name,
				From:     fromSchema + "." + fromTable,
				To:       toSchema + "." + toTable,
				OnDelete: deleteRuleFa(deleteRule),
				Required: required,
			}
			index[key] = rel
			order = append(order, key)
		}
		rel.FromColumn = append(rel.FromColumn, fromColumn)
		rel.ToColumn = append(rel.ToColumn, toColumn)
	}
	out := make([]Relation, 0, len(order))
	for _, key := range order {
		out = append(out, *index[key])
	}
	return out, rows.Err()
}

// loadRowCounts شمار سطرهای هر جدول را دقیق می‌خواند (برای درک «کجا داده داریم»).
func loadRowCounts(ctx context.Context, db *sql.DB) (map[string]int64, error) {
	rows, err := db.QueryContext(ctx, `
		select n.nspname, cl.relname
		from pg_class cl
		join pg_namespace n on n.oid = cl.relnamespace
		where cl.relkind in ('r', 'p') and (n.nspname like 'hr\_%' or n.nspname = 'public')
		order by n.nspname, cl.relname`)
	if err != nil {
		return nil, fmt.Errorf("فهرست جدول‌ها: %w", err)
	}
	var names []string
	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			rows.Close()
			return nil, fmt.Errorf("خواندن سطر جدول: %w", err)
		}
		names = append(names, schema+"."+table)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make(map[string]int64, len(names))
	for _, path := range names {
		var n int64
		q := "select count(*) from " + QuotePath(path)
		if err := db.QueryRowContext(ctx, q).Scan(&n); err != nil {
			// جدول بدون دسترسی یا در حال قفل: صفر گزارش می‌شود، نه خطا.
			out[path] = 0
			continue
		}
		out[path] = n
	}
	return out, nil
}

// moduleMap نگاشت ماژول‌ها به جدول‌های ذخیره‌شدهٔ آن‌ها را می‌سازد.
func moduleMap(modules []ModuleInfo, dict Dictionary) []Module {
	tablesBySchema := map[string][]string{}
	for _, schema := range dict.Schemas {
		for _, t := range schema.Tables {
			tablesBySchema[schema.Name] = append(tablesBySchema[schema.Name], t.Path)
		}
	}
	out := make([]Module, 0, len(modules))
	for _, m := range modules {
		out = append(out, Module{
			ID:        m.ID,
			TitleFa:   m.TitleFa,
			APIPrefix: m.APIPrefix,
			DBShema:   m.DBSchema,
			Entities:  m.Entities,
			Screens:   m.Screens,
			Tables:    tablesBySchema[m.DBSchema],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func findTable(schema *Schema, name string) *Table {
	for i := range schema.Tables {
		if schema.Tables[i].Name == name {
			return &schema.Tables[i]
		}
	}
	return nil
}

func schemaRows(schema Schema) int64 {
	var total int64
	for _, t := range schema.Tables {
		total += t.Rows
	}
	return total
}

// QuotePath مسیر «schema.table» را برای درج در SQL امن می‌کند.
func QuotePath(path string) string {
	parts := strings.SplitN(path, ".", 2)
	for i := range parts {
		parts[i] = `"` + strings.ReplaceAll(parts[i], `"`, `""`) + `"`
	}
	return strings.Join(parts, ".")
}

// schemaTitleFa عنوان فارسی طرح را از نام آن می‌سازد (وقتی ماژولی برای آن نباشد).
func schemaTitleFa(name string) string {
	switch name {
	case "public":
		return "سطح پلتفرم (کاربران، دسترسی‌ها، مهاجرت‌ها)"
	}
	return strings.TrimPrefix(name, "hr_")
}

// typeLabelFa نام فارسی نوع داده را برمی‌گرداند (برای خوانایی سند فرهنگ داده).
func typeLabelFa(dataType string) string {
	switch {
	case strings.HasPrefix(dataType, "character varying"), strings.HasPrefix(dataType, "character"):
		return "متنی"
	case dataType == "text":
		return "متنی بلند"
	case dataType == "boolean":
		return "بولی"
	case dataType == "date":
		return "تاریخ (میلادی؛ شمسی در لایهٔ کد)"
	case strings.HasPrefix(dataType, "timestamp"):
		return "برچسب زمانی"
	case strings.HasPrefix(dataType, "numeric"), strings.HasPrefix(dataType, "double"), dataType == "real":
		return "اعشاری"
	case dataType == "integer", dataType == "smallint":
		return "عدد صحیح"
	case dataType == "bigint":
		return "عدد صحیح بزرگ"
	case dataType == "jsonb", dataType == "json":
		return "JSON"
	case dataType == "uuid":
		return "شناسه یکتا"
	case dataType == "bytea":
		return "داده دودویی"
	default:
		return dataType
	}
}

// deleteRuleFa قاعدهٔ حذف را فارسی می‌کند (کدهای pg_constraint.confdeltype).
func deleteRuleFa(code string) string {
	switch code {
	case "a":
		return "بدون عمل (NO ACTION)"
	case "r":
		return "منع حذف (RESTRICT)"
	case "c":
		return "حذف آبشاری (CASCADE)"
	case "n":
		return "تهی‌کردن (SET NULL)"
	case "d":
		return "مقدار پیش‌فرض (SET DEFAULT)"
	default:
		return code
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
