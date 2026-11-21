// آزمون‌های فرهنگ داده: درخت، ERD، سند و CSV — با نمونهٔ ساختاری (بدون نیاز به پایگاه‌داده).
// ارجاع: docs/data-dictionary-fa.md (بندهای DD-01 … DD-05).

package datadict

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// fixture یک فرهنگ دادهٔ کوچک با دو جدول و یک رابطه می‌سازد.
func fixture() Dictionary {
	return Dictionary{
		GeneratedAt: "2026-01-01T00:00:00Z",
		Database:    Database{Name: "hrms_test", Server: "PostgreSQL 17", Current: "hrms_app", SizeKb: 1024},
		Summary:     Summary{Schemas: 2, Tables: 2, Columns: 4, Relations: 1, Rows: 7},
		Schemas: []Schema{
			{
				Name: "hr_employee", TitleFa: "امور کارکنان", ModuleID: "employee",
				Tables: []Table{
					{
						Name: "department", Path: "hr_employee.department", Rows: 2, PrimaryKey: []string{"id"},
						Columns: []Column{
							{Name: "id", Type: "text", TypeFa: "متنی بلند", PrimaryKey: true},
							{Name: "name", Type: "text", TypeFa: "متنی بلند", Description: "نام واحد"},
						},
					},
					{
						Name: "employee", Path: "hr_employee.employee", Rows: 5, PrimaryKey: []string{"id"},
						ForeignKeys: []string{"employee_department_fk"},
						Columns: []Column{
							{Name: "id", Type: "text", TypeFa: "متنی بلند", PrimaryKey: true},
							{Name: "department_id", Type: "text", TypeFa: "متنی بلند", Nullable: true, ForeignKey: true},
						},
					},
				},
			},
			{Name: "public", TitleFa: "سطح پلتفرم", Platform: true, Tables: []Table{}},
		},
		Relations: []Relation{{
			Name: "employee_department_fk", From: "hr_employee.employee", FromColumn: []string{"department_id"},
			To: "hr_employee.department", ToColumn: []string{"id"}, OnDelete: "منع حذف (RESTRICT)", Required: false,
		}},
		Modules: []Module{{
			ID: "employee", TitleFa: "امور کارکنان", APIPrefix: "/api/v1/employee",
			DBShema: "hr_employee", Entities: []string{"employee"}, Tables: []string{"hr_employee.department", "hr_employee.employee"},
		}},
	}
}

// TestDD01TreeAndJSON می‌سنجد ساختار درختی و JSON هم طرح/جدول/ستون و هم کلید و شمار
// سطرها را نشان دهند و JSON قابل تجزیه باشد.
func TestDD01TreeAndJSON(t *testing.T) {
	dict := fixture()

	tree, contentType, err := Render(dict, "tree")
	if err != nil {
		t.Fatalf("رندر درختی ناموفق: %v", err)
	}
	if !strings.Contains(contentType, "text/plain") {
		t.Fatalf("نوع محتوای درخت نامناسب: %s", contentType)
	}
	for _, want := range []string{"hr_employee", "امور کارکنان", "employee", "5 سطر", "PK", "FK", "تهی‌پذیر"} {
		if !strings.Contains(tree, want) {
			t.Fatalf("درخت باید «%s» را داشته باشد:\n%s", want, tree)
		}
	}

	raw, _, err := Render(dict, "json")
	if err != nil {
		t.Fatalf("رندر JSON ناموفق: %v", err)
	}
	var back Dictionary
	if err := json.Unmarshal([]byte(raw), &back); err != nil {
		t.Fatalf("JSON فرهنگ داده قابل تجزیه نیست: %v", err)
	}
	if back.Summary.Columns != 4 || len(back.Schemas) != 2 {
		t.Fatalf("JSON ناقص است: %+v", back.Summary)
	}

	if _, _, err := Render(dict, "nope"); err == nil {
		t.Fatal("قالب ناشناخته باید خطا بدهد")
	}
}

// TestDD02MermaidERD می‌سنجد نمودار انتیتی-رلیشن از روابط خارجی ساخته شود و شناسه‌های
// Mermaid بدون نقطه (قاعدهٔ زبان) باشند.
func TestDD02MermaidERD(t *testing.T) {
	erd := RenderMermaid(fixture())
	if !strings.HasPrefix(erd, "erDiagram") {
		t.Fatalf("خروجی باید با erDiagram شروع شود:\n%s", erd)
	}
	for _, want := range []string{
		"hr_employee_employee {",
		"hr_employee_department {",
		"text id PK",
		"text department_id FK",
		"}o--||",
		"منع حذف",
	} {
		if !strings.Contains(erd, want) {
			t.Fatalf("نمودار باید «%s» را داشته باشد:\n%s", want, erd)
		}
	}
	if strings.Contains(erd, "hr_employee.employee {") {
		t.Fatal("شناسهٔ Mermaid نباید نقطه داشته باشد")
	}
}

// TestDD03MarkdownAndCSV می‌سنجد سند Markdown (شامل بلوک Mermaid و جدول روابط) و CSV
// ستون‌محور تولید شوند.
func TestDD03MarkdownAndCSV(t *testing.T) {
	doc := RenderMarkdown(fixture())
	for _, want := range []string{"# فرهنگ داده", "| ستون |", "روابط خارجی", "```mermaid", "hr_employee.employee"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("سند باید «%s» را داشته باشد", want)
		}
	}

	csv := RenderCSV(fixture())
	if !strings.HasPrefix(csv, "\uFEFF") {
		t.Fatal("CSV باید با BOM شروع شود تا Excel فارسی را درست بخواند")
	}
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) != 5 { // سرصفحه + ۴ ستون
		t.Fatalf("CSV باید ۵ سطر داشته باشد، %d داشت", len(lines))
	}
	if !strings.Contains(lines[0], "schema,table,column,type") {
		t.Fatalf("سرصفحهٔ CSV نادرست است: %s", lines[0])
	}
}

// TestDD05ModuleMapping می‌سنجد نگاشت ماژول به جدول‌های ذخیره‌شده در JSON بیاید.
func TestDD05ModuleMapping(t *testing.T) {
	raw, _, err := Render(fixture(), "json")
	if err != nil {
		t.Fatalf("رندر ناموفق: %v", err)
	}
	for _, want := range []string{`"moduleId": "employee"`, `"dbSchema": "hr_employee"`, `/api/v1/employee`} {
		if !strings.Contains(raw, want) {
			t.Fatalf("JSON باید «%s» را داشته باشد", want)
		}
	}
}

// TestDD01LoadNeedsDatabase می‌سنجد بدون اتصال پایگاه‌داده، ساخت فرهنگ داده با پیام
// روشن رد شود (نه پنیک).
func TestDD01LoadNeedsDatabase(t *testing.T) {
	if _, err := Load(context.Background(), nil, nil, Options{}); err == nil {
		t.Fatal("بدون پایگاه‌داده باید خطا بدهد")
	}
}

// TestDD01TypeLabels می‌سنجد برچسب فارسی نوع‌ها و قاعدهٔ حذف درست نگاشت شوند.
func TestDD01TypeLabels(t *testing.T) {
	cases := map[string]string{
		"text":                     "متنی بلند",
		"character varying(80)":    "متنی",
		"bigint":                   "عدد صحیح بزرگ",
		"timestamp with time zone": "برچسب زمانی",
		"jsonb":                    "JSON",
		"boolean":                  "بولی",
		"numeric(12,2)":            "اعشاری",
		"date":                     "تاریخ (میلادی؛ شمسی در لایهٔ کد)",
		"something_unknown":        "something_unknown",
	}
	for in, want := range cases {
		if got := typeLabelFa(in); got != want {
			t.Fatalf("typeLabelFa(%q) = %q، انتظار %q", in, got, want)
		}
	}
	if got := deleteRuleFa("c"); !strings.Contains(got, "CASCADE") {
		t.Fatalf("قاعدهٔ حذف آبشاری نادرست: %s", got)
	}
	if got := QuotePath("hr_employee.employee"); got != `"hr_employee"."employee"` {
		t.Fatalf("QuotePath نادرست: %s", got)
	}
}
