// آزمون API زیرفرم «حقوق و مزایا» قرارداد: تعریف فرم‌ها، اعتبارسنجی همان تعریف در سرور
// و نگه‌داشتن فرادادهٔ خانواده. ارجاع: docs/contract-pay-form-spec-fa.md بند CPF-05.

package bootstrap

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// TestCPF05ContractFormsAPI می‌سنجد مسیر فرم‌ها سه خانواده را می‌دهد، بدون نشست ۴۰۱ است،
// و ثبت قرارداد با نوع استخدام پیمانکاری بدون مشخصات پیمان با ۴۲۲ و پیام فارسی رد می‌شود.
func TestCPF05ContractFormsAPI(t *testing.T) {
	h := newTestServer(t)

	// ۱) بدون نشست: ۴۰۱.
	if rec := call(t, h, http.MethodGet, "/api/v1/contracts/forms", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("فرم‌ها بدون نشست باید ۴۰۱ بدهد، %d داد", rec.Code)
	}

	manager := login(t, h, "hr.manager", "demo")
	rec := call(t, h, http.MethodGet, "/api/v1/contracts/forms", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("دریافت زیرفرم‌ها باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Forms           []map[string]any `json:"forms"`
		EmploymentTypes []map[string]any `json:"employmentTypes"`
		NoteFa          string           `json:"noteFa"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("پاسخ زیرفرم‌ها قابل تجزیه نیست: %v", err)
	}
	if len(payload.Forms) != 3 {
		t.Fatalf("سه خانوادهٔ پرداخت انتظار می‌رفت، %d آمد", len(payload.Forms))
	}
	families := map[string]bool{}
	for _, form := range payload.Forms {
		families[form["family"].(string)] = true
	}
	for _, want := range []string{"official", "contracting", "secondment"} {
		if !families[want] {
			t.Fatalf("خانوادهٔ «%s» در پاسخ نیست: %s", want, rec.Body.String())
		}
	}
	if len(payload.EmploymentTypes) < 8 {
		t.Fatalf("فهرست انواع استخدام ناقص است (%d)", len(payload.EmploymentTypes))
	}
	if !strings.Contains(rec.Body.String(), "payBasis") || !strings.Contains(rec.Body.String(), "hostCompany") {
		t.Fatal("پاسخ باید فیلدهای پیمان و مأموریت را داشته باشد")
	}

	// ۲) پیمانکاری بدون مشخصات پیمان: ۴۲۲ با خطای فیلد.
	start := jdate(t, 10)
	fixed := jdate(t, 400)
	incomplete := map[string]any{
		"employeeId": "EMP-012", "kind": "fixed_term", "employmentType": "contractor_staff",
		"startDate": start, "endDate": fixed, "monthlySalary": 25000000,
	}
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts", manager.AccessToken, incomplete)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("پیمانکاری ناقص باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "contractorName") {
		t.Fatalf("خطا باید روی فیلد نام پیمانکار باشد: %s", rec.Body.String())
	}

	// ۳) همان قرارداد با مشخصات کامل پیمان: ۲۰۱ و نگه‌داشتن خانواده و فراداده.
	complete := map[string]any{
		"employeeId": "EMP-012", "kind": "fixed_term", "employmentType": "contractor_staff",
		"startDate": start, "endDate": fixed, "monthlySalary": 25000000,
		"payMeta": map[string]any{
			"contractorName": "شرکت پیمانکار نمونه", "contractNo": "P-1405-01",
			"payBasis": "monthly", "payAmount": 25000000, "insurancePayer": "contractor",
		},
	}
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts", manager.AccessToken, complete)
	if rec.Code != http.StatusCreated {
		t.Fatalf("پیمانکاری کامل باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var created contracts.Contract
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("پاسخ قرارداد قابل تجزیه نیست: %v", err)
	}
	if created.PayFamily != "contracting" {
		t.Fatalf("خانوادهٔ پرداخت باید contracting باشد، «%s» بود", created.PayFamily)
	}
	if created.PayMeta["contractNo"] != "P-1405-01" {
		t.Fatalf("فرادادهٔ پیمان نگه داشته نشد: %+v", created.PayMeta)
	}
	if created.PayFamilyFa == "" {
		t.Fatal("عنوان فارسی خانوادهٔ پرداخت باید پر شود")
	}

	// ۴) همان سند با اجزای مزد رسمی: سرور باید رد کند (UI هم آن فیلدها را نشان نمی‌دهد).
	withWage := map[string]any{
		"employeeId": "EMP-011", "kind": "fixed_term", "employmentType": "contractor_current",
		"startDate": start, "endDate": fixed, "monthlySalary": 22000000,
		"rankWage": 5000000,
		"payMeta": map[string]any{
			"contractorName": "پیمانکار دیگر", "contractNo": "P-1405-02",
			"payBasis": "daily", "payAmount": 900000, "insurancePayer": "none",
		},
	}
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts", manager.AccessToken, withWage)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("اجزای مزد رسمی در پیمانکاری باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "wageStructure") {
		t.Fatalf("خطا باید به ساختار مزد اشاره کند: %s", rec.Body.String())
	}
}
