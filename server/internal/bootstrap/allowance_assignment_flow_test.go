// آزمون مسیرهای «تخصیص فوق العاده‌های حقوق»: جدول رسمی کدها، پیش‌نمایش تخصیص و اعتبارسنجی.
// ارجاع: docs/allowance-assignment-spec-fa.md (بندهای ASG-05 … ASG-07).

package bootstrap

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// allowancePreviewAnswer پاسخ پیش‌نمایش تخصیص.
type allowancePreviewAnswer struct {
	Lines []struct {
		Code       string `json:"code"`
		TitleFa    string `json:"titleFa"`
		Category   string `json:"category"`
		Calc       string `json:"calc"`
		Amount     int64  `json:"amount"`
		Taxable    bool   `json:"taxable"`
		Insured    bool   `json:"insured"`
		SourceFa   string `json:"sourceFa"`
		Resolved   bool   `json:"resolved"`
		DetailFa   string `json:"detailFa"`
		Assignment string `json:"assignmentId"`
	} `json:"lines"`
	Totals struct {
		Earnings        int64 `json:"earnings"`
		Deductions      int64 `json:"deductions"`
		TaxableEarnings int64 `json:"taxableEarnings"`
		Lines           int   `json:"lines"`
	} `json:"totals"`
	Period string `json:"period"`
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
