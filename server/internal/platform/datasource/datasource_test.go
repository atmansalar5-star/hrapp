// آزمون‌های رابط منبع داده: درایور فایل (CSV/Excel)، سنجش پرس‌وجوی فقط‌خواندنی،
// بارگذاری فایل، ساختار داده و قالب‌ها.

package datasource

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"hrms/internal/contracts"
)

// writeCSV یک فایل CSV نمونه با BOM و جداکنندهٔ نقطه‌ویرگول می‌سازد.
func writeCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("\ufeff"+content), 0o644); err != nil {
		t.Fatalf("نوشتن فایل CSV آزمون ناموفق: %v", err)
	}
	return path
}

func TestCSVSourceReadsFile(t *testing.T) {
	dir := t.TempDir()
	path := writeCSV(t, dir, "employees.csv",
		"id;fullName;nationalId;baseSalary\nEMP-001;سارا محمدی;1234567891;52000000\nEMP-002;آرش رضایی;0023456787;67000000\n")

	source, ok := Get(KindCSV)
	if !ok {
		t.Fatal("درایور CSV ثبت نشده است")
	}
	cfg := Config{Kind: KindCSV, FilePath: path}

	report, err := source.Test(context.Background(), cfg)
	if err != nil || !report.OK {
		t.Fatalf("آزمون اتصال فایل باید موفق باشد: %v (%+v)", err, report)
	}
	if len(report.Tables) != 1 {
		t.Errorf("فایل باید یک «جدول» بدهد: %+v", report.Tables)
	}

	tables, err := source.Tables(context.Background(), cfg)
	if err != nil || len(tables) != 1 {
		t.Fatalf("فهرست جدول‌های فایل نباید خطا بدهد: %v", err)
	}
	// ستون نخست باید بدون BOM و درست خوانده شود.
	if got := tables[0].Columns[0].Name; got != "id" {
		t.Errorf("نام ستون نخست باید «id» باشد، «%s» شد", got)
	}

	result, err := source.Preview(context.Background(), cfg, "", 10)
	if err != nil {
		t.Fatalf("پیش‌نمایش فایل نباید خطا بدهد: %v", err)
	}
	if len(result.Columns) != 4 || len(result.Rows) != 2 {
		t.Fatalf("پیش‌نمایش باید ۴ ستون و ۲ ردیف بدهد: %+v", result)
	}
	if result.Rows[0][1] != "سارا محمدی" || result.Rows[1][3] != "67000000" {
		t.Errorf("مقادیر ردیف‌ها درست خوانده نشد: %+v", result.Rows)
	}

	// سقف ردیف: با limit=۱ باید یک ردیف و نشانهٔ بریده‌شدن بیاید.
	limited, err := source.Query(context.Background(), cfg, "", 1)
	if err != nil {
		t.Fatalf("پرس‌وجوی محدود نباید خطا بدهد: %v", err)
	}
	if len(limited.Rows) != 1 || !limited.Truncated {
		t.Errorf("سقف ردیف اعمال نشد: %+v", limited)
	}
}

func TestXLSXSourceReadsWorkbook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.xlsx")
	book := excelize.NewFile()
	if _, err := book.NewSheet("کارکنان"); err != nil {
		t.Fatalf("ساخت برگه ناموفق: %v", err)
	}
	_ = book.DeleteSheet("Sheet1")
	rows := [][]any{
		{"nationalId", "fullName", "baseSalary"},
		{"1234567891", "سارا محمدی", 52000000},
		{"0023456787", "آرش رضایی", 67000000},
	}
	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			_ = book.SetCellValue("کارکنان", cell, value)
		}
	}
	if err := book.SaveAs(path); err != nil {
		t.Fatalf("ذخیرهٔ فایل اکسل آزمون ناموفق: %v", err)
	}
	book.Close()

	source, ok := Get(KindXLSX)
	if !ok {
		t.Fatal("درایور اکسل ثبت نشده است")
	}
	cfg := Config{Kind: KindXLSX, FilePath: path, Sheet: "کارکنان"}
	report, err := source.Test(context.Background(), cfg)
	if err != nil || !report.OK {
		t.Fatalf("آزمون اتصال اکسل باید موفق باشد: %v (%+v)", err, report)
	}
	if report.TableCount != 1 {
		t.Errorf("کارپوشه باید یک برگه داشته باشد: %+v", report.Tables)
	}
	result, err := source.Preview(context.Background(), cfg, "کارکنان", 10)
	if err != nil {
		t.Fatalf("پیش‌نمایش برگه نباید خطا بدهد: %v", err)
	}
	if len(result.Columns) != 3 || len(result.Rows) != 2 {
		t.Fatalf("پیش‌نمایش اکسل باید ۳ ستون و ۲ ردیف بدهد: %+v", result)
	}
	if result.Rows[0][1] != "سارا محمدی" {
		t.Errorf("مقدار فارسی در اکسل درست خوانده نشد: %+v", result.Rows[0])
	}
}

func TestReadOnlyStatementGuard(t *testing.T) {
	allowed := []string{"SELECT * FROM employees", "  select id from t where x = 1", "WITH a AS (SELECT 1) SELECT * FROM a"}
	for _, statement := range allowed {
		if err := IsReadOnlyStatement(statement); err != nil {
			t.Errorf("دستور خواندنی «%s» باید مجاز باشد: %v", statement, err)
		}
	}
	rejected := []string{
		"", "UPDATE employees SET x = 1", "DELETE FROM employees", "DROP TABLE employees",
		"SELECT 1; DELETE FROM employees", "SELECT * INTO backup FROM employees",
		"EXEC sp_who", "CALL do_something()", "CREATE TABLE x (a int)",
	}
	for _, statement := range rejected {
		if err := IsReadOnlyStatement(statement); err == nil {
			t.Errorf("دستور «%s» باید رد شود (اتصال فقط‌خواندنی است)", statement)
		}
	}
}

func TestSafeIdentifier(t *testing.T) {
	for _, name := range []string{"employees", "public.employees", "hr_data$1"} {
		if _, err := safeIdentifier(name); err != nil {
			t.Errorf("نام «%s» باید پذیرفته شود: %v", name, err)
		}
	}
	for _, name := range []string{"", "employees; DELETE FROM x", "a b", "emp--", "select * from x"} {
		if _, err := safeIdentifier(name); err == nil {
			t.Errorf("نام «%s» باید رد شود", name)
		}
	}
}

func TestSaveUploadValidations(t *testing.T) {
	dir := t.TempDir()
	upload, err := SaveUpload(dir, "کارکنان 1405.csv", []byte("id,name\nEMP-001,الف\n"))
	if err != nil {
		t.Fatalf("بارگذاری فایل CSV نباید خطا بدهد: %v", err)
	}
	if !strings.HasPrefix(filepath.ToSlash(upload.Path), filepath.ToSlash(UploadDir(dir))+"/") {
		t.Errorf("فایل بارگذاری‌شده باید داخل پوشهٔ بارگذاری‌ها بماند: %s", upload.Path)
	}
	if strings.ContainsAny(filepath.Base(upload.Path), " ٱ") || !strings.HasSuffix(upload.Path, ".csv") {
		t.Errorf("نام فایل ذخیره‌شده باید امن و با پسوند درست باشد: %s", filepath.Base(upload.Path))
	}
	if upload.KindFa != "فایل CSV" || len(upload.SHA256) != 12 {
		t.Errorf("اطلاعات فایل بارگذاری‌شده ناقص است: %+v", upload)
	}
	cases := []struct {
		name string
		data []byte
	}{
		{"data.xls", []byte("x")},
		{"data.pdf", []byte("x")},
		{"data.csv", nil},
	}
	for _, item := range cases {
		if _, err := SaveUpload(dir, item.name, item.data); err == nil {
			t.Errorf("فایل «%s» باید رد شود", item.name)
		}
	}
	if _, err := SaveUpload("", "data.csv", []byte("x")); err == nil {
		t.Errorf("نبود پوشهٔ داده باید خطا بدهد")
	}
}

func TestManifestAndGuide(t *testing.T) {
	curated := CuratedEntities()
	if len(curated) < 8 {
		t.Fatalf("ساختار گردآوری‌شده باید داده‌های پایه را داشته باشد، %d آمد", len(curated))
	}
	found := map[string]bool{}
	for _, entity := range curated {
		found[entity.Key] = true
		if entity.TitleFa == "" || entity.SheetFa == "" || entity.Order == 0 {
			t.Errorf("موجودیت «%s» باید عنوان، برگه و ترتیب داشته باشد: %+v", entity.Key, entity)
		}
	}
	for _, want := range []string{"companies", "employees", "bankAccounts", "deviceMap", "contracts"} {
		if !found[want] {
			t.Errorf("موجودیت «%s» در ساختار گردآوری‌شده نیست", want)
		}
	}

	// ادغام با یک شناسهٔ ساختگی: نوع تازه باید به فهرست اضافه و نوع تکراری یکی شود.
	schemas := []contracts.RecordSchema{{
		Module: "attendance", ModuleTitleFa: "حضور و غیاب",
		Kinds: []contracts.RecordKindSchema{
			{Key: "shiftPatterns", TitleFa: "الگوی چرخهٔ نوبت", Fields: []contracts.RecordField{
				{Key: "rotation", LabelFa: "ترتیب چرخه", Type: "text", Required: true},
			}},
			{Key: "customKind", TitleFa: "نوع تازه", Fields: []contracts.RecordField{
				{Key: "note", LabelFa: "یادداشت", Type: "text"},
			}},
		},
	}}
	entities := Manifest(schemas, map[string]string{"attendance": "حضور و غیاب"})
	seen := map[string]int{}
	for _, entity := range entities {
		seen[entity.Key]++
	}
	if seen["shiftPatterns"] != 1 {
		t.Errorf("نوع تکراری باید یکی شود: %d", seen["shiftPatterns"])
	}
	if seen["customKind"] != 1 {
		t.Errorf("نوع تازه باید به ساختار اضافه شود: %d", seen["customKind"])
	}

	guide := GuideFa(entities, "پتروشیمی بندر امام")
	for _, want := range []string{"راهنمای آماده‌سازی اطلاعات پایه", "قاعده‌های عمومی فایل", "YYYY/MM/DD", "کارکنان"} {
		if !strings.Contains(guide, want) {
			t.Errorf("راهنما باید «%s» را داشته باشد", want)
		}
	}
}

func TestTemplateGeneration(t *testing.T) {
	entities := SelectEntities(CuratedEntities(), []string{"companies", "employees"})
	if len(entities) != 2 {
		t.Fatalf("انتخاب برگه‌ها باید دو موجودیت بدهد، %d آمد", len(entities))
	}
	content, err := TemplateXLSX(entities, "پتروشیمی بندر امام")
	if err != nil {
		t.Fatalf("ساخت قالب اکسل نباید خطا بدهد: %v", err)
	}
	if len(content) < 2000 {
		t.Errorf("حجم قالب اکسل کم است: %d بایت", len(content))
	}
	book, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("قالب ساخته‌شده باید فایل اکسل معتبر باشد: %v", err)
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) < 3 {
		t.Errorf("قالب باید برگهٔ راهنما و دو برگهٔ جدول داشته باشد: %+v", sheets)
	}
	hasGuide, hasCompanies := false, false
	for _, sheet := range sheets {
		if sheet == "راهنما" {
			hasGuide = true
		}
		if sheet == "شرکت‌ها" {
			hasCompanies = true
		}
	}
	if !hasGuide || !hasCompanies {
		t.Errorf("نام برگه‌های قالب درست نیست: %+v", sheets)
	}
	header, err := book.GetRows("شرکت‌ها")
	if err != nil || len(header) < 2 {
		t.Fatalf("برگهٔ شرکت‌ها باید سرصفحه و ردیف نمونه داشته باشد: %v", err)
	}
	if header[0][0] != "code" {
		t.Errorf("سرصفحهٔ برگه باید کلید فنی ستون‌ها باشد: %+v", header[0][:3])
	}

	csvContent, err := TemplateCSV(entities[1])
	if err != nil {
		t.Fatalf("ساخت قالب CSV نباید خطا بدهد: %v", err)
	}
	text := string(csvContent)
	if !strings.HasPrefix(text, "\ufeff") || !strings.Contains(text, "nationalId") {
		t.Errorf("قالب CSV باید BOM و سرصفحهٔ فنی داشته باشد: %.80s", text)
	}
}
