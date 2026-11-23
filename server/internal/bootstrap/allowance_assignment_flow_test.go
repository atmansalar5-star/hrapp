// آزمون مسیرهای «تخصیص فوق العاده‌های حقوق»: جدول رسمی کدها، پیش‌نمایش تخصیص و اعتبارسنجی.
// ارجاع: docs/allowance-assignment-spec-fa.md (بندهای ASG-05 … ASG-07).

package bootstrap

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// allowancePreviewLine یک سطر نتیجهٔ پیش‌نمایش تخصیص.
type allowancePreviewLine struct {
	Code          string `json:"code"`
	TitleFa       string `json:"titleFa"`
	Category      string `json:"category"`
	Calc          string `json:"calc"`
	Amount        int64  `json:"amount"`
	Taxable       bool   `json:"taxable"`
	Insured       bool   `json:"insured"`
	ShowOnPayslip bool   `json:"showOnPayslip"`
	SourceFa      string `json:"sourceFa"`
	Resolved      bool   `json:"resolved"`
	DetailFa      string `json:"detailFa"`
	Assignment    string `json:"assignmentId"`
	DefinitionID  string `json:"definitionId"`
	ReportCode    string `json:"reportCode"`
}

// allowancePreviewAnswer پاسخ پیش‌نمایش تخصیص.
type allowancePreviewAnswer struct {
	Lines  []allowancePreviewLine `json:"lines"`
	Totals struct {
		Earnings        int64 `json:"earnings"`
		Deductions      int64 `json:"deductions"`
		TaxableEarnings int64 `json:"taxableEarnings"`
		Lines           int   `json:"lines"`
	} `json:"totals"`
	Bases  map[string]float64 `json:"bases"`
	Period string             `json:"period"`
}

// TestASG05AllowanceDefaults می‌سنجد جدول رسمی کدها و فهرست‌های مرجع فرم تخصیص برمی‌گردد.
func TestASG05AllowanceDefaults(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	rec := call(t, h, http.MethodGet, "/api/v1/payroll/allowance-defaults", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("پیش‌فرض‌های تخصیص باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Codes []struct {
			Code    string `json:"code"`
			TitleFa string `json:"titleFa"`
			Kind    string `json:"kind"`
		} `json:"codes"`
		CalcKinds   []map[string]string `json:"calcKinds"`
		Bases       []map[string]string `json:"bases"`
		TargetKinds []map[string]string `json:"targetKinds"`
		Employees   []map[string]string `json:"employees"`
		Gaps        []string            `json:"gaps"`
		NoteFa      string              `json:"noteFa"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("پاسخ پیش‌فرض قابل تجزیه نیست: %v", err)
	}
	if len(payload.Codes) < 28 {
		t.Fatalf("جدول رسمی کدها باید دست‌کم ۲۸ ردیف داشته باشد، %d آمد", len(payload.Codes))
	}
	if len(payload.CalcKinds) != 5 {
		t.Fatalf("پنج روش محاسبه انتظار می‌رفت، %d آمد", len(payload.CalcKinds))
	}
	if len(payload.Bases) < 4 || len(payload.TargetKinds) < 6 {
		t.Fatalf("مبناها/دامنه‌ها ناقص است: %d/%d", len(payload.Bases), len(payload.TargetKinds))
	}
	if len(payload.Employees) == 0 {
		t.Fatal("فهرست کارکنان برای انتخاب دامنه نباید خالی باشد")
	}
	// ویژگی‌های لازم تطبیق دامنه روی هر کارمند.
	first := payload.Employees[0]
	for _, key := range []string{"id", "unit", "employmentType", "jobGrade", "payGroup"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("کارمند باید فیلد «%s» را داشته باشد: %+v", key, first)
		}
	}
	if strings.TrimSpace(payload.NoteFa) == "" {
		t.Fatal("پاسخ باید توضیح فارسی داشته باشد")
	}
}

// TestASG06AllowancePreview می‌سنجد پیش‌نمایش، اقلام تخصیص‌یافتهٔ همان کارمند را با کد، مبلغ و
// مشمولیت کسور برمی‌گرداند و جمع‌ها درست است.
func TestASG06AllowancePreview(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	// کارمند رسمی نفت: تخصیص گروهی مسکن (۱۵۱) و حق اولاد (۱۸۱) به او می‌خورد.
	rec := call(t, h, http.MethodPost, "/api/v1/payroll/allowance-preview", token,
		map[string]any{"employeeId": "EMP-001"})
	if rec.Code != http.StatusOK {
		t.Fatalf("پیش‌نمایش تخصیص باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var preview allowancePreviewAnswer
	decodeInto(t, rec, &preview)
	if preview.Totals.Lines == 0 {
		t.Fatalf("تخصیص‌های بذر‌شده باید برای کارمند رسمی نفت سطر بسازند: %s", rec.Body.String())
	}
	codes := map[string]bool{}
	for _, line := range preview.Lines {
		codes[line.Code] = true
		if line.Code == "" || line.TitleFa == "" || !line.Resolved {
			t.Fatalf("هر سطر باید کد، عنوان و مبلغ حل‌شده داشته باشد: %+v", line)
		}
		if line.Category != "earning" && line.Category != "deduction" {
			t.Fatalf("نوع قلم سطر نامعتبر است: %+v", line)
		}
	}
	if !codes["151"] || !codes["181"] {
		t.Fatalf("کدهای ۱۵۱ (مسکن) و ۱۸۱ (حق اولاد) باید در پیش‌نمایش باشند: %+v", codes)
	}
	if preview.Totals.Earnings <= 0 {
		t.Fatalf("جمع مزایا باید مثبت باشد: %+v", preview.Totals)
	}

	// کارمندی که تخصیص کسر وام برای او ثبت شده است.
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/allowance-preview", token,
		map[string]any{"employeeId": "EMP-004"})
	if rec.Code != http.StatusOK {
		t.Fatalf("پیش‌نمایش کارمند دوم باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var second allowancePreviewAnswer
	decodeInto(t, rec, &second)
	found := false
	for _, line := range second.Lines {
		if line.Code == "5400" && line.Category == "deduction" {
			found = true
		}
	}
	if !found {
		t.Fatalf("کد ۵۴۰۰ (کسر اقساط وام) باید برای EMP-004 بیاید: %+v", second.Lines)
	}
	if second.Totals.Deductions <= 0 {
		t.Fatalf("جمع کسور باید مثبت باشد: %+v", second.Totals)
	}

	// کارمند ناشناخته: ۴۰۴.
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/allowance-preview", token,
		map[string]any{"employeeId": "EMP-999"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("کارمند ناشناخته باید ۴۰۴ بدهد، %d داد", rec.Code)
	}
}

// TestASG07AssignmentRecordValidation می‌سنجد ثبت تخصیص از راه رکوردها کار می‌کند و ورودی
// ناقص با خطای فیلدبه‌فیلد رد می‌شود.
func TestASG07AssignmentRecordValidation(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	// فهرست بذر‌شده.
	rec := call(t, h, http.MethodGet, "/api/v1/payroll/records?kind=allowanceAssignment&size=50", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست تخصیص‌ها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var page struct {
		Total int `json:"total"`
	}
	decodeInto(t, rec, &page)
	if page.Total < 3 {
		t.Fatalf("تخصیص‌های نمونه بذر نشده‌اند (total=%d)", page.Total)
	}

	// ثبت تخصیص تازه برای یک گروه (گروه نظام پرداخت ۱ = کارمند رسمی نفت).
	create := map[string]any{
		"kind":  "allowanceAssignment",
		"title": "تخصیص فوق‌العادهٔ کارگاهی — گروه یکم",
		"fields": map[string]string{
			"code": "122", "calc": "percent", "rate": "10", "basis": "allowanceBase",
			"taxable": "بله", "insured": "بله",
			"targetKind": "payGroup", "targetValue": "1",
			"effectiveFrom": "1405/01/01", "noteFa": "آزمون تخصیص گروهی",
		},
	}
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/records", token, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت تخصیص باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID     string            `json:"id"`
		Fields map[string]string `json:"fields"`
	}
	decodeInto(t, rec, &created)
	if created.Fields["titleFa"] == "" {
		t.Fatal("عنوان قلم باید از جدول رسمی همان کد پر شود")
	}

	// فعال‌سازی و دیدنش در پیش‌نمایش کارمند رسمی نفت.
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/records/"+created.ID+"/action", token,
		map[string]any{"action": "activate"})
	if rec.Code != http.StatusOK {
		t.Fatalf("فعال‌سازی تخصیص باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/allowance-preview", token, map[string]any{"employeeId": "EMP-001"})
	var withWorkshop allowancePreviewAnswer
	decodeInto(t, rec, &withWorkshop)
	found := false
	for _, line := range withWorkshop.Lines {
		if line.Code == "122" {
			found = true
			if !line.Taxable || !line.Insured {
				t.Fatalf("سطر کارگاهی باید مشمول مالیات و بیمه باشد: %+v", line)
			}
		}
	}
	if !found {
		t.Fatalf("تخصیص فعال‌شدهٔ گروهی باید در پیش‌نمایش بیاید: %+v", withWorkshop.Lines)
	}

	// ورودی ناقص: کد نامعتبر.
	bad := map[string]any{
		"kind": "allowanceAssignment", "title": "تخصیص نادرست",
		"fields": map[string]string{"code": "999", "calc": "fixed", "rate": "1000",
			"targetKind": "all", "effectiveFrom": "1405/01/01"},
	}
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/records", token, bad)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("کد نامعتبر باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "fields.code") {
		t.Fatalf("خطا باید روی فیلد کد باشد: %s", rec.Body.String())
	}
}

// TestASG11PayslipVisibility می‌سنجد جای نمایش هر قلم تخصیص (فیش حقوقی / گزارش چاپی / کد
// گزارش) در بلوک فیش رعایت می‌شود: قلمی که «نمایش در فیش» ندارد از سطرهای فیش کنار گذاشته
// می‌شود ولی در گزارش‌های چاپی می‌آید.
func TestASG11PayslipVisibility(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	created := call(t, h, http.MethodPost, "/api/v1/payroll/records", manager.AccessToken, map[string]any{
		"kind": "allowanceAssignment", "title": "پاداش نوروزی — فقط گزارش",
		"fields": map[string]string{
			"code": "364", "calc": "fixed", "rate": "5000000",
			"taxable": "خیر", "insured": "خیر",
			"showOnPayslip": "خیر", "showInReports": "بله", "reportCode": "364-R",
			"targetKind": "employee", "targetValue": "EMP-001",
			"effectiveFrom": "1405/01",
		},
	})
	if created.Code != http.StatusCreated && created.Code != http.StatusOK {
		t.Fatalf("ثبت تخصیص باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var record contracts.Record
	decodeInto(t, created, &record)
	// پرچم‌ها در رکورد ذخیره شده‌اند.
	if record.Fields["showOnPayslip"] != "خیر" || record.Fields["showInReports"] != "بله" {
		t.Fatalf("پرچم‌های نمایش ذخیره نشدند: %+v", record.Fields)
	}
	if record.Fields["reportCode"] != "364-R" {
		t.Fatalf("کد گزارش ذخیره نشد: %+v", record.Fields)
	}
	act := call(t, h, http.MethodPost, "/api/v1/payroll/records/"+record.ID+"/action",
		manager.AccessToken, map[string]any{"action": "activate"})
	if act.Code != http.StatusOK {
		t.Fatalf("فعال‌سازی باید ۲۰۰ بدهد، %d داد: %s", act.Code, act.Body.String())
	}

	slip := call(t, h, http.MethodPost, "/api/v1/payroll/payslip", manager.AccessToken,
		map[string]any{"employeeId": "EMP-001"})
	if slip.Code != http.StatusOK {
		t.Fatalf("فیش باید ۲۰۰ بدهد، %d داد: %s", slip.Code, slip.Body.String())
	}
	type line struct {
		Code          string `json:"code"`
		ShowOnPayslip bool   `json:"showOnPayslip"`
		ShowInReports bool   `json:"showInReports"`
		ReportCode    string `json:"reportCode"`
	}
	var answer struct {
		Assignments struct {
			Lines        []line `json:"lines"`
			PayslipLines []line `json:"payslipLines"`
			ReportLines  []line `json:"reportLines"`
		} `json:"assignments"`
	}
	decodeInto(t, slip, &answer)

	inLines, inPayslip, inReport := false, false, false
	for _, l := range answer.Assignments.Lines {
		if l.Code != "364" {
			continue
		}
		inLines = true
		if l.ShowOnPayslip || !l.ShowInReports || l.ReportCode != "364-R" {
			t.Fatalf("پرچم‌های قلم ۳۶۴ در بلوک فیش درست نیست: %+v", l)
		}
	}
	for _, l := range answer.Assignments.PayslipLines {
		if l.Code == "364" {
			inPayslip = true
		}
	}
	for _, l := range answer.Assignments.ReportLines {
		if l.Code == "364" {
			inReport = true
		}
	}
	if !inLines {
		t.Fatalf("قلم ۳۶۴ باید در سطرهای تخصیص باشد: %+v", answer.Assignments.Lines)
	}
	if inPayslip {
		t.Fatal("قلمی که «نمایش در فیش» ندارد نباید در سطرهای فیش بیاید")
	}
	if !inReport {
		t.Fatal("قلمی که «درج در گزارش» دارد باید در سطرهای گزارش باشد")
	}
}

// allowanceDefinitionView یک تعریف فوق العاده در پاسخ پیش‌فرض‌ها.
type allowanceDefinitionView struct {
	ID              string          `json:"id"`
	TitleFa         string          `json:"titleFa"`
	Code            string          `json:"code"`
	CodeOfficial    bool            `json:"codeOfficial"`
	Category        string          `json:"category"`
	Calc            string          `json:"calc"`
	CalcFa          string          `json:"calcFa"`
	Rate            string          `json:"rate"`
	Basis           string          `json:"basis"`
	Vars            string          `json:"vars"`
	Taxable         bool            `json:"taxable"`
	Insured         bool            `json:"insured"`
	ShowOnPayslip   bool            `json:"showOnPayslip"`
	ShowInReports   bool            `json:"showInReports"`
	ReportCode      string          `json:"reportCode"`
	Status          string          `json:"status"`
	StatusFa        string          `json:"statusFa"`
	AssignmentCount int             `json:"assignmentCount"`
	Active          bool            `json:"active"`
	Detail          json.RawMessage `json:"-"`
}

// TestASG14DefinitionFlow می‌سنجد گام یکم کار (تعریف فوق العاده) سرتاسری کار می‌کند:
// تعریف‌های نمونه در پیش‌فرض‌ها می‌آیند، تعریف تازه با کد داخلی و متغیرها ثبت و فعال می‌شود
// و تعریف نامعتبر با خطای فیلدی رد می‌گردد.
func TestASG14DefinitionFlow(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	// ---------- پیش‌فرض‌ها: فهرست تعریف‌ها با شمار تخصیص ----------
	rec := call(t, h, http.MethodGet, "/api/v1/payroll/allowance-defaults", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("پیش‌فرض‌ها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var defaults struct {
		Definitions []allowanceDefinitionView `json:"definitions"`
		Codes       []map[string]string       `json:"codes"`
	}
	decodeInto(t, rec, &defaults)
	if len(defaults.Definitions) == 0 {
		t.Fatalf("تعریف‌های نمونه باید در پیش‌فرض‌ها بیایند: %s", rec.Body.String())
	}
	var withVars, withAssignments int
	for _, def := range defaults.Definitions {
		if def.Code == "" || def.TitleFa == "" || def.Calc == "" || def.CalcFa == "" || def.StatusFa == "" {
			t.Fatalf("هر تعریف باید کد، عنوان، روش و وضعیت فارسی داشته باشد: %+v", def)
		}
		if def.Vars != "" {
			withVars++
		}
		if def.AssignmentCount > 0 {
			withAssignments++
		}
	}
	if withVars == 0 {
		t.Fatalf("باید دست‌کم یک تعریف با متغیر باشد: %+v", defaults.Definitions)
	}
	if withAssignments == 0 {
		t.Fatalf("تخصیص‌های نمونه باید به تعریف‌ها وصل باشند: %+v", defaults.Definitions)
	}

	// ---------- ثبت تعریف تازه با کد داخلی، فرمول و متغیر ----------
	created := call(t, h, http.MethodPost, "/api/v1/payroll/records", manager.AccessToken, map[string]any{
		"kind": "allowanceDefinition", "title": "فوق العادهٔ داخلی آزمون",
		"fields": map[string]string{
			"code": "X-100", "calc": "formula", "formula": "BASE*CHILDREN", "vars": "CHILDREN=2, SITE=1",
			"category": "earning", "showOnPayslip": "بله", "showInReports": "بله", "reportCode": "X100-R",
		},
	})
	if created.Code != http.StatusCreated && created.Code != http.StatusOK {
		t.Fatalf("ثبت تعریف باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var record contracts.Record
	decodeInto(t, created, &record)
	if record.Fields["vars"] != "CHILDREN=2, SITE=1" {
		t.Fatalf("متغیرها باید خوانا در رکورد بمانند: %+v", record.Fields)
	}
	if record.Fields["category"] != "earning" {
		t.Fatalf("نوع قلم تعریف باید ذخیره شود: %+v", record.Fields)
	}
	act := call(t, h, http.MethodPost, "/api/v1/payroll/records/"+record.ID+"/action",
		manager.AccessToken, map[string]any{"action": "activate"})
	if act.Code != http.StatusOK {
		t.Fatalf("فعال‌سازی تعریف باید ۲۰۰ بدهد، %d داد: %s", act.Code, act.Body.String())
	}

	// تعریف تازه در پیش‌فرض‌ها با کد داخلی (غیررسمی) و متغیر دیده می‌شود.
	rec = call(t, h, http.MethodGet, "/api/v1/payroll/allowance-defaults", manager.AccessToken, nil)
	var after []allowanceDefinitionView
	var refreshed struct {
		Definitions []allowanceDefinitionView `json:"definitions"`
	}
	decodeInto(t, rec, &refreshed)
	after = refreshed.Definitions
	found := false
	for _, def := range after {
		if def.ID != record.ID {
			continue
		}
		found = true
		if def.CodeOfficial {
			t.Fatal("کد داخلی نباید رسمی شمرده شود")
		}
		if def.Vars != "CHILDREN=2, SITE=1" || !def.Active {
			t.Fatalf("تعریف تازه باید متغیر و وضعیت فعال داشته باشد: %+v", def)
		}
	}
	if !found {
		t.Fatalf("تعریف تازه در پیش‌فرض‌ها نیست: %+v", after)
	}

	// ---------- تعریف نامعتبر: مقدار صفر برای مبلغ ثابت ----------
	bad := call(t, h, http.MethodPost, "/api/v1/payroll/records", manager.AccessToken, map[string]any{
		"kind": "allowanceDefinition", "title": "تعریف نادرست",
		"fields": map[string]string{"code": "151", "calc": "fixed", "rate": "0"},
	})
	if bad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تعریف نامعتبر باید ۴۲۲ بدهد، %d داد: %s", bad.Code, bad.Body.String())
	}
	if !strings.Contains(bad.Body.String(), "fields.rate") {
		t.Fatalf("خطا باید روی فیلد مقدار باشد: %s", bad.Body.String())
	}
}

// TestASG15GroupAssignment می‌سنجد گام دوم کار کار می‌کند: یک تعریف به گروه استخدامی و به
// یک کارمند تخصیص می‌یابد، مبلغ از تعریف می‌آید، بازنویسی مقدار اثر می‌کند و بلوک فیش همان
// سطر تعریف‌محور را نشان می‌دهد.
func TestASG15GroupAssignment(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	// ---------- گام ۱: تعریف فوق العادهٔ کارگاهی (درصدی از مبنای فوق العاده‌ها) ----------
	created := call(t, h, http.MethodPost, "/api/v1/payroll/records", manager.AccessToken, map[string]any{
		"kind": "allowanceDefinition", "title": "فوق العادهٔ کارگاهی — تعریف آزمون",
		"fields": map[string]string{
			"code": "122", "calc": "percent", "rate": "10", "basis": "allowanceBase",
			"taxable": "بله", "insured": "بله",
			"showOnPayslip": "بله", "showInReports": "بله", "reportCode": "122-R",
		},
	})
	if created.Code != http.StatusCreated && created.Code != http.StatusOK {
		t.Fatalf("ثبت تعریف باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var definition contracts.Record
	decodeInto(t, created, &definition)
	activate := func(kind, id string) {
		t.Helper()
		rec := call(t, h, http.MethodPost, "/api/v1/payroll/records/"+id+"/action",
			manager.AccessToken, map[string]any{"action": "activate"})
		if rec.Code != http.StatusOK {
			t.Fatalf("فعال‌سازی %s باید ۲۰۰ بدهد، %d داد: %s", kind, rec.Code, rec.Body.String())
		}
	}
	activate("تعریف", definition.ID)

	// ---------- گام ۲: تخصیص تعریف به گروه استخدامی «رسمی نفت» ----------
	groupAssignment := call(t, h, http.MethodPost, "/api/v1/payroll/records", manager.AccessToken, map[string]any{
		"kind": "allowanceAssignment", "title": "تخصیص کارگاهی — گروه استخدامی رسمی نفت",
		"fields": map[string]string{
			"definition": definition.ID, "targetKind": "employmentType", "targetValue": "official_oil",
			"effectiveFrom": "1405/01",
		},
	})
	if groupAssignment.Code != http.StatusCreated && groupAssignment.Code != http.StatusOK {
		t.Fatalf("تخصیص تعریف‌محور باید ثبت شود، %d داد: %s", groupAssignment.Code, groupAssignment.Body.String())
	}
	var group contracts.Record
	decodeInto(t, groupAssignment, &group)
	activate("تخصیص گروهی", group.ID)

	// ---------- پیش‌نمایش کارمند گروه: مبلغ از تعریف (۱۰٪ مبنای فوق العاده‌ها) ----------
	preview := call(t, h, http.MethodPost, "/api/v1/payroll/allowance-preview", manager.AccessToken,
		map[string]any{"employeeId": "EMP-001"})
	if preview.Code != http.StatusOK {
		t.Fatalf("پیش‌نمایش باید ۲۰۰ بدهد، %d داد: %s", preview.Code, preview.Body.String())
	}
	var answer allowancePreviewAnswer
	decodeInto(t, preview, &answer)
	base := answer.Bases["allowanceBase"]
	if base <= 0 {
		t.Fatalf("مبنای فوق العاده‌ها باید در پاسخ باشد: %+v", answer.Bases)
	}
	wantAmount := int64(base * 10 / 100)
	var line *allowancePreviewLine
	for i := range answer.Lines {
		if answer.Lines[i].DefinitionID == definition.ID {
			line = &answer.Lines[i]
		}
	}
	if line == nil {
		t.Fatalf("سطر تعریف‌محور در پیش‌نمایش نیامد: %+v", answer.Lines)
	}
	if line.Code != "122" || line.Amount != wantAmount || !line.Taxable || !line.Insured || line.ReportCode != "122-R" {
		t.Fatalf("سطر تعریف‌محور درست نیست (انتظار %d): %+v", wantAmount, line)
	}

	// ---------- بازنویسی مقدار برای یک کارمند خاص (۱۵٪) ----------
	personAssignment := call(t, h, http.MethodPost, "/api/v1/payroll/records", manager.AccessToken, map[string]any{
		"kind": "allowanceAssignment", "title": "تخصیص کارگاهی — کارمند خاص",
		"fields": map[string]string{
			"definition": definition.ID, "rateOverride": "15",
			"targetKind": "employee", "targetValue": "EMP-003", "effectiveFrom": "1405/01",
		},
	})
	if personAssignment.Code != http.StatusCreated && personAssignment.Code != http.StatusOK {
		t.Fatalf("تخصیص فردی باید ثبت شود، %d داد: %s", personAssignment.Code, personAssignment.Body.String())
	}
	var person contracts.Record
	decodeInto(t, personAssignment, &person)
	activate("تخصیص فردی", person.ID)

	personPreview := call(t, h, http.MethodPost, "/api/v1/payroll/allowance-preview", manager.AccessToken,
		map[string]any{"employeeId": "EMP-003"})
	var personAnswer allowancePreviewAnswer
	decodeInto(t, personPreview, &personAnswer)
	var overrideLine *allowancePreviewLine
	for i := range personAnswer.Lines {
		if personAnswer.Lines[i].DefinitionID == definition.ID {
			overrideLine = &personAnswer.Lines[i]
		}
	}
	if overrideLine == nil {
		t.Fatalf("سطر تعریف‌محور برای کارمند خاص نیامد: %+v", personAnswer.Lines)
	}
	personBase := personAnswer.Bases["allowanceBase"]
	if overrideLine.Amount != int64(personBase*15/100) {
		t.Fatalf("بازنویسی ۱۵٪ باید اثر کند: %+v (مبنا %v)", overrideLine, personBase)
	}

	// ---------- بلوک فیش: همان سطر تعریف‌محور ----------
	slip := call(t, h, http.MethodPost, "/api/v1/payroll/payslip", manager.AccessToken,
		map[string]any{"employeeId": "EMP-001"})
	if slip.Code != http.StatusOK {
		t.Fatalf("فیش باید ۲۰۰ بدهد، %d داد: %s", slip.Code, slip.Body.String())
	}
	var slipAnswer struct {
		Assignments struct {
			Lines []struct {
				Code         string `json:"code"`
				Amount       int64  `json:"amount"`
				DefinitionID string `json:"definitionId"`
			} `json:"lines"`
		} `json:"assignments"`
	}
	decodeInto(t, slip, &slipAnswer)
	found := false
	for _, l := range slipAnswer.Assignments.Lines {
		if l.DefinitionID == definition.ID && l.Code == "122" {
			found = true
		}
	}
	if !found {
		t.Fatalf("بلوک فیش باید سطر تعریف‌محور را نشان دهد: %+v", slipAnswer.Assignments.Lines)
	}
}
