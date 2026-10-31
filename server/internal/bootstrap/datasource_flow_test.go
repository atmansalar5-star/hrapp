// آزمون جریان سرتاسری «تبادل داده»: فهرست انواع منبع، بارگذاری فایل CSV، پیش‌نمایش،
// ساختار داده (manifest)، راهنمای آماده‌سازی و قالب‌های اکسل/CSV.

package bootstrap

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// uploadCSV فایل CSV را در سرور آزمون بارگذاری می‌کند و مسیر سرور را برمی‌گرداند.
func uploadCSV(t *testing.T, h http.Handler, token, name, content string) string {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("ساخت بخش فایل ناموفق: %v", err)
	}
	if _, err := part.Write([]byte("\ufeff" + content)); err != nil {
		t.Fatalf("نوشتن محتوای فایل ناموفق: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("بستن بدنهٔ multipart ناموفق: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/data-sources/upload", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("بارگذاری فایل باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Upload struct {
			Path      string `json:"path"`
			Name      string `json:"name"`
			KindFa    string `json:"kindFa"`
			SizeBytes int    `json:"sizeBytes"`
		} `json:"upload"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("پاسخ بارگذاری قابل تجزیه نیست: %v", err)
	}
	if payload.Upload.Path == "" || payload.Upload.KindFa == "" {
		t.Fatalf("پاسخ بارگذاری ناقص است: %+v", payload.Upload)
	}
	return payload.Upload.Path
}

func TestDataSourceAndTemplateFlow(t *testing.T) {
	h := newTestServer(t)
	admin := login(t, h, "admin", "demo").AccessToken
	emp := login(t, h, "employee", "demo").AccessToken

	// ---------- فهرست انواع منبع داده ----------
	rec := call(t, h, http.MethodGet, "/api/v1/settings/data-sources/kinds", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست انواع منبع باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var kindsBody struct {
		Kinds []struct {
			Kind, TitleFa, CategoryFa string
			Fields                    []struct{ Key, TitleFa string }
		} `json:"kinds"`
		NoteFa string `json:"noteFa"`
	}
	decodeInto(t, rec, &kindsBody)
	wantKinds := map[string]bool{"xlsx": false, "csv": false, "postgres": false, "sqlserver": false, "mysql": false, "oracle": false}
	for _, kind := range kindsBody.Kinds {
		if _, ok := wantKinds[kind.Kind]; ok {
			wantKinds[kind.Kind] = true
		}
		if len(kind.Fields) == 0 {
			t.Errorf("نوع «%s» باید فیلدهای پیکربندی داشته باشد", kind.Kind)
		}
	}
	for kind, found := range wantKinds {
		if !found {
			t.Errorf("نوع منبع «%s» در فهرست رابط نیست", kind)
		}
	}

	// ---------- بارگذاری فایل و پیش‌نمایش ----------
	path := uploadCSV(t, h, admin, "employees-1405.csv",
		"id;fullName;nationalId;baseSalary\nEMP-001;سارا محمدی;1234567891;52000000\nEMP-002;آرش رضایی;0023456787;67000000\n")

	rec = call(t, h, http.MethodPost, "/api/v1/settings/data-sources/test", admin, map[string]any{
		"config": map[string]any{"kind": "csv", "filePath": path, "readOnly": true},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("آزمون اتصال فایل باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var testBody struct {
		OK        bool     `json:"ok"`
		MessageFa string   `json:"messageFa"`
		Tables    []string `json:"tables"`
	}
	decodeInto(t, rec, &testBody)
	if !testBody.OK || testBody.MessageFa == "" {
		t.Errorf("آزمون اتصال باید موفق و دارای پیام فارسی باشد: %+v", testBody)
	}

	rec = call(t, h, http.MethodPost, "/api/v1/settings/data-sources/preview", admin, map[string]any{
		"config":   map[string]any{"kind": "csv", "filePath": path, "readOnly": true},
		"rowLimit": 10,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("پیش‌نمایش باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var preview struct {
		Columns  []string   `json:"columns"`
		Rows     [][]string `json:"rows"`
		RowCount int        `json:"rowCount"`
	}
	decodeInto(t, rec, &preview)
	if len(preview.Columns) != 4 || preview.RowCount != 2 {
		t.Fatalf("پیش‌نمایش باید ۴ ستون و ۲ ردیف بدهد: %+v", preview)
	}
	if preview.Columns[0] != "id" || preview.Rows[0][1] != "سارا محمدی" {
		t.Errorf("پیش‌نمایش مقادیر را درست نخواند: %+v", preview)
	}

	// ---------- پرس‌وجوی تغییردهنده روی منبع رد می‌شود ----------
	rec = call(t, h, http.MethodPost, "/api/v1/settings/data-sources/preview", admin, map[string]any{
		"config": map[string]any{"kind": "csv", "filePath": path, "readOnly": true},
		"query":  "DELETE FROM employees",
	})
	if rec.Code != http.StatusConflict {
		t.Errorf("پرس‌وجوی تغییردهنده باید رد شود، %d داد: %s", rec.Code, rec.Body.String())
	}

	// ---------- ساختار داده ----------
	rec = call(t, h, http.MethodGet, "/api/v1/settings/data-sources/manifest", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ساختار داده باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var manifest struct {
		Entities []struct {
			Key, TitleFa, Module, Kind, SheetFa, GroupFa string
			Fields                                       []struct {
				Key, TitleFa, Type string
				Required           bool
			}
		} `json:"entities"`
		Total int `json:"total"`
	}
	decodeInto(t, rec, &manifest)
	if manifest.Total < 20 {
		t.Fatalf("ساختار داده باید همهٔ جدول‌های سامانه را داشته باشد، %d آمد", manifest.Total)
	}
	keys := map[string]bool{}
	for _, entity := range manifest.Entities {
		keys[entity.Key] = true
		if entity.SheetFa == "" {
			t.Errorf("موجودیت «%s» باید نام برگه داشته باشد", entity.Key)
		}
	}
	for _, want := range []string{"employees", "companies", "bankAccounts", "deviceMap"} {
		if !keys[want] {
			t.Errorf("موجودیت «%s» در ساختار داده نیست", want)
		}
	}
	var employeeEntity struct {
		Key, TitleFa, Module, Kind, SheetFa, GroupFa string
		Fields                                       []struct {
			Key, TitleFa, Type string
			Required           bool
		}
	}
	for _, entity := range manifest.Entities {
		if entity.Key == "employees" {
			employeeEntity.Key = entity.Key
			employeeEntity.TitleFa = entity.TitleFa
			employeeEntity.SheetFa = entity.SheetFa
			employeeEntity.Module = entity.Module
			for _, field := range entity.Fields {
				employeeEntity.Fields = append(employeeEntity.Fields, struct {
					Key, TitleFa, Type string
					Required           bool
				}{field.Key, field.TitleFa, field.Type, field.Required})
			}
		}
	}
	if len(employeeEntity.Fields) < 10 {
		t.Fatalf("ساختار کارکنان باید ستون‌های کامل داشته باشد، %d آمد", len(employeeEntity.Fields))
	}
	requiredSeen := false
	for _, field := range employeeEntity.Fields {
		if field.Key == "nationalId" && field.Required {
			requiredSeen = true
		}
	}
	if !requiredSeen {
		t.Errorf("کد ملی باید ستون الزامی کارکنان باشد: %+v", employeeEntity.Fields)
	}

	// ---------- راهنمای آماده‌سازی ----------
	rec = call(t, h, http.MethodGet, "/api/v1/settings/data-sources/guide", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("راهنما باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	guide := rec.Body.String()
	for _, want := range []string{"راهنمای آماده‌سازی اطلاعات پایه", "قاعده‌های عمومی فایل", "YYYY/MM/DD", "کارکنان"} {
		if !strings.Contains(guide, want) {
			t.Errorf("راهنما باید «%s» را داشته باشد", want)
		}
	}

	// ---------- قالب اکسل ----------
	rec = call(t, h, http.MethodGet, "/api/v1/settings/data-sources/template.xlsx?entities=companies,employees", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("قالب اکسل باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "spreadsheetml") {
		t.Errorf("نوع محتوای قالب اکسل درست نیست: %s", ct)
	}
	book, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("قالب ساخته‌شده باید اکسل معتبر باشد: %v", err)
	}
	defer book.Close()
	sheets := book.GetSheetList()
	if len(sheets) < 3 {
		t.Errorf("قالب باید برگهٔ راهنما و دو برگهٔ جدول داشته باشد: %+v", sheets)
	}
	rows, err := book.GetRows("کارکنان")
	if err != nil || len(rows) == 0 {
		t.Fatalf("برگهٔ کارکنان در قالب باید سرصفحه داشته باشد: %v", err)
	}
	if rows[0][0] != "id" {
		t.Errorf("سرصفحهٔ برگهٔ کارکنان باید کلیدهای فنی باشد: %+v", rows[0][:3])
	}

	// ---------- قالب CSV ----------
	rec = call(t, h, http.MethodGet, "/api/v1/settings/data-sources/template.csv?entity=bankAccounts", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("قالب CSV باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	csvBody := rec.Body.String()
	if !strings.HasPrefix(csvBody, "\ufeff") || !strings.Contains(csvBody, "iban") {
		t.Errorf("قالب CSV باید BOM و ستون شبا داشته باشد: %.80s", csvBody)
	}

	// ---------- دسترسی ----------
	if code := call(t, h, http.MethodGet, "/api/v1/settings/data-sources/manifest", emp, nil).Code; code != http.StatusForbidden {
		t.Errorf("دسترسی کارمند به ساختار داده باید ۴۰۳ بدهد، %d داد", code)
	}
	if code := call(t, h, http.MethodPost, "/api/v1/settings/data-sources/test", "",
		map[string]any{"config": map[string]any{"kind": "csv", "filePath": path}}).Code; code != http.StatusUnauthorized {
		t.Errorf("آزمون اتصال بدون توکن باید ۴۰۱ بدهد، %d داد", code)
	}
}
