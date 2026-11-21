// رندر فرهنگ داده در قالب‌های خوانا: JSON (خودِ ساختار)، متن درختی، Mermaid (نمودار
// انتیتی-رلیشن)، Markdown (سند فرهنگ داده) و CSV (برای Excel).

package datadict

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Formats قالب‌های پشتیبانی‌شدهٔ خروجی.
var Formats = []string{"json", "tree", "mermaid", "md", "csv"}

// Render قالب خواسته‌شده را می‌سازد؛ قالب ناشناخته خطا است.
func Render(d Dictionary, format string) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		raw, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return "", "", err
		}
		return string(raw) + "\n", "application/json; charset=utf-8", nil
	case "tree":
		return RenderTree(d), "text/plain; charset=utf-8", nil
	case "mermaid":
		return RenderMermaid(d), "text/plain; charset=utf-8", nil
	case "md", "markdown":
		return RenderMarkdown(d), "text/markdown; charset=utf-8", nil
	case "csv":
		return RenderCSV(d), "text/csv; charset=utf-8", nil
	default:
		return "", "", fmt.Errorf("قالب ناشناخته «%s»؛ قالب‌های مجاز: %s", format, strings.Join(Formats, ", "))
	}
}

// RenderTree ساختار درختی «طرح ← جدول ← ستون» را برای خواندن سریع می‌سازد.
func RenderTree(d Dictionary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "فرهنگ دادهٔ %s — %s\n", d.Database.Name, d.GeneratedAt)
	fmt.Fprintf(&b, "طرح‌ها: %d · جدول‌ها: %d · ستون‌ها: %d · روابط: %d · سطرها: %d\n\n",
		d.Summary.Schemas, d.Summary.Tables, d.Summary.Columns, d.Summary.Relations, d.Summary.Rows)

	for _, schema := range d.Schemas {
		fmt.Fprintf(&b, "%s — %s\n", schema.Name, schema.TitleFa)
		for ti, table := range schema.Tables {
			last := ti == len(schema.Tables)-1
			branch := "├─"
			if last {
				branch = "└─"
			}
			pk := ""
			if len(table.PrimaryKey) > 0 {
				pk = " PK(" + strings.Join(table.PrimaryKey, ", ") + ")"
			}
			fmt.Fprintf(&b, "%s %s%s%s\n", branch, table.Name, rowSuffix(table.Rows), pk)
			indent := "│  "
			if last {
				indent = "   "
			}
			for ci, col := range table.Columns {
				mark := "├─"
				if ci == len(table.Columns)-1 {
					mark = "└─"
				}
				flags := ""
				if col.PrimaryKey {
					flags += " PK"
				}
				if col.ForeignKey {
					flags += " FK"
				}
				if col.Nullable {
					flags += " تهی‌پذیر"
				}
				fmt.Fprintf(&b, "%s%s %s : %s (%s)%s\n", indent, mark, col.Name, col.Type, col.TypeFa, flags)
			}
		}
		fmt.Fprintln(&b)
	}

	if len(d.Relations) > 0 {
		b.WriteString("روابط خارجی:\n")
		for _, r := range d.Relations {
			fmt.Fprintf(&b, "  %s(%s) → %s(%s) [%s]\n",
				r.From, strings.Join(r.FromColumn, ", "), r.To, strings.Join(r.ToColumn, ", "), r.OnDelete)
		}
	}
	return b.String()
}

// RenderMermaid نمودار انتیتی-رلیشن را به زبان Mermaid می‌سازد (قابل چسباندن در هر
// نمایشگر Mermaid، مثل VS Code، GitLab یا mermaid.live).
func RenderMermaid(d Dictionary) string {
	var b strings.Builder
	b.WriteString("erDiagram\n")
	b.WriteString("    %% فرهنگ داده — تولیدشده از کاتالوگ زنده؛ برای بروزرسانی دوباره درخواست کنید.\n")

	for _, schema := range d.Schemas {
		for _, table := range schema.Tables {
			fmt.Fprintf(&b, "    %s {\n", mermaidID(table.Path))
			for _, col := range table.Columns {
				flags := make([]string, 0, 2)
				if col.PrimaryKey {
					flags = append(flags, "PK")
				}
				if col.ForeignKey {
					flags = append(flags, "FK")
				}
				fmt.Fprintf(&b, "        %s %s%s\n", mermaidType(col.Type), mermaidID(col.Name), flagSuffix(flags))
			}
			b.WriteString("    }\n")
		}
	}

	for _, r := range d.Relations {
		link := "}o--||"
		if r.Required {
			link = "}|--||"
		}
		label := strings.Join(r.FromColumn, ",") + " → " + strings.Join(r.ToColumn, ",") + " · " + r.OnDelete
		fmt.Fprintf(&b, "    %s %s %s : \"%s\"\n", mermaidID(r.From), link, mermaidID(r.To), label)
	}
	return b.String()
}

// RenderMarkdown سند فرهنگ داده را می‌سازد: خلاصه، ماژول‌ها، هر طرح با جدول ستون‌ها و
// فهرست روابط — همان چیزی که برای طراحی و بازبینی داده لازم است.
func RenderMarkdown(d Dictionary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# فرهنگ داده — %s\n\n", d.Database.Name)
	if d.GeneratedAt == "" {
		fmt.Fprintf(&b, "> این سند **تولیدشده** و «پایدار» است (بدون مُهر زمان و بدون شمار سطرها) تا با تغییر داده کهنه نشود.\n")
	} else {
		fmt.Fprintf(&b, "> تولیدشده در %s از کاتالوگ زندهٔ %s روی %s.\n", d.GeneratedAt, d.Database.Server, d.Database.Name)
	}
	fmt.Fprintf(&b, "> برای بروزرسانی پس از هر مهاجرت، فرمان بازتولید را اجرا کنید.\n\n")

	fmt.Fprintf(&b, "| شاخص | شمار |\n| --- | --- |\n")
	rowsCell := fmt.Sprintf("%d", d.Summary.Rows)
	if d.Summary.Rows == 0 {
		rowsCell = "— (در نسخهٔ پایدار شمرده نمی‌شود؛ خروجی زندهٔ API آن را می‌دهد)"
	}
	fmt.Fprintf(&b, "| طرح‌ها | %d |\n| جدول‌ها | %d |\n| ستون‌ها | %d |\n| روابط خارجی | %d |\n| سطرهای داده | %s |\n\n",
		d.Summary.Schemas, d.Summary.Tables, d.Summary.Columns, d.Summary.Relations, rowsCell)

	if len(d.Modules) > 0 {
		b.WriteString("## ماژول‌ها و دادهٔ ذخیره‌شده\n\n")
		b.WriteString("| ماژول | عنوان | پیشوند API | طرح داده | موجودیت‌ها | جدول‌ها |\n| --- | --- | --- | --- | --- | --- |\n")
		for _, m := range d.Modules {
			fmt.Fprintf(&b, "| `%s` | %s | `%s` | `%s` | %s | %d |\n",
				m.ID, m.TitleFa, m.APIPrefix, m.DBShema,
				strings.Join(m.Entities, ", "), len(m.Tables))
		}
		b.WriteString("\n")
	}

	for _, schema := range d.Schemas {
		fmt.Fprintf(&b, "## %s — %s\n\n", schema.Name, schema.TitleFa)
		for _, table := range schema.Tables {
			fmt.Fprintf(&b, "### `%s`%s\n\n", table.Path, rowSuffix(table.Rows))
			b.WriteString("| ستون | نوع | معنای نوع | تهی‌پذیر | پیش‌فرض | کلید | توضیح |\n")
			b.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
			for _, col := range table.Columns {
				key := ""
				if col.PrimaryKey {
					key += "PK "
				}
				if col.ForeignKey {
					key += "FK"
				}
				nullable := "خیر"
				if col.Nullable {
					nullable = "بله"
				}
				fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s | %s | %s | %s |\n",
					col.Name, col.Type, col.TypeFa, nullable, mdOrDash(col.Default), strings.TrimSpace(key), mdOrDash(col.Description))
			}
			b.WriteString("\n")
		}
	}

	if len(d.Relations) > 0 {
		b.WriteString("## روابط خارجی (مبنای نمودار ERD)\n\n")
		b.WriteString("| جدول مبدأ | ستون | جدول مقصد | ستون | قاعدهٔ حذف | اجباری |\n| --- | --- | --- | --- | --- | --- |\n")
		for _, r := range d.Relations {
			required := "خیر"
			if r.Required {
				required = "بله"
			}
			fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` | %s | %s |\n",
				r.From, strings.Join(r.FromColumn, ", "), r.To, strings.Join(r.ToColumn, ", "), r.OnDelete, required)
		}
		b.WriteString("\n")
	}

	b.WriteString("## نمودار ERD (Mermaid)\n\n```mermaid\n")
	b.WriteString(strings.TrimRight(RenderMermaid(d), "\n"))
	b.WriteString("\n```\n\n")
	b.WriteString(d.NoteFa + "\n")
	return b.String()
}

// RenderCSV خروجی مسطح ستون‌محور برای Excel و ابزارهای داده (با BOM تا Excel فارسی
// را درست بخواند).
func RenderCSV(d Dictionary) string {
	var b strings.Builder
	b.WriteString("\uFEFF")
	b.WriteString("schema,table,column,type,typeFa,nullable,default,primaryKey,foreignKey,description,rows\n")
	for _, schema := range d.Schemas {
		for _, table := range schema.Tables {
			for _, col := range table.Columns {
				rowsCell := ""
				if table.Rows > 0 {
					rowsCell = fmt.Sprintf("%d", table.Rows)
				}
				fields := []string{
					schema.Name, table.Name, col.Name, col.Type, col.TypeFa,
					boolFa(col.Nullable), col.Default, boolFa(col.PrimaryKey),
					boolFa(col.ForeignKey), col.Description, rowsCell,
				}
				for i := range fields {
					fields[i] = csvCell(fields[i])
				}
				b.WriteString(strings.Join(fields, ",") + "\n")
			}
		}
	}
	return b.String()
}

// RelationIndex فهرست روابط یک جدول را برمی‌گرداند (مصرف: کلاینت‌ها و ابزارها).
func (d Dictionary) RelationIndex() map[string][]Relation {
	out := map[string][]Relation{}
	for _, r := range d.Relations {
		out[r.From] = append(out[r.From], r)
	}
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].Name < out[k][j].Name })
	}
	return out
}

func mermaidID(name string) string {
	return strings.NewReplacer(".", "_", "-", "_", " ", "_").Replace(name)
}

// mermaidType نوع را به توکن سادهٔ Mermaid تبدیل می‌کند (بدون فاصله و پرانتز).
func mermaidType(dataType string) string {
	switch {
	case strings.HasPrefix(dataType, "character varying"):
		return "varchar"
	case strings.HasPrefix(dataType, "character"):
		return "char"
	case strings.HasPrefix(dataType, "timestamp"):
		return "timestamp"
	case strings.HasPrefix(dataType, "numeric"):
		return "numeric"
	case strings.HasPrefix(dataType, "double"):
		return "double"
	}
	if i := strings.IndexAny(dataType, " ("); i > 0 {
		dataType = dataType[:i]
	}
	return strings.NewReplacer("-", "_", " ", "_").Replace(dataType)
}

func flagSuffix(flags []string) string {
	if len(flags) == 0 {
		return ""
	}
	return " " + strings.Join(flags, ",")
}

func boolFa(v bool) string {
	if v {
		return "بله"
	}
	return "خیر"
}

// rowSuffix شمار سطرها را فقط وقتی نشان می‌دهد که جدول داده داشته باشد؛ در فایل‌های
// «پایدار» (بدون شمارش سطر) این بخش خالی می‌ماند.
func rowSuffix(rows int64) string {
	if rows <= 0 {
		return ""
	}
	return fmt.Sprintf(" — %d سطر", rows)
}

func mdOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return strings.ReplaceAll(value, "|", "\\|")
}

func csvCell(value string) string {
	if strings.ContainsAny(value, ",\"\n") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return value
}
