// آزمون دستهٔ پرداخت در قرارداد، حکم و گزارش دسته‌بندی.
// ارجاع: docs/pay-classification-spec-fa.md (بندهای PAY-04 … PAY-07).

package bootstrap

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
	"hrms/internal/rules"
)

// TestPAY04ContractPayClassification می‌سنجد فهرست روش‌ها و گروه‌ها از مسیر فرم‌ها می‌آید و
// قرارداد، گروه و روش پرداخت را بر پایهٔ نوع استخدام ثبت می‌کند — و ترکیب ناسازگار را رد.
func TestPAY04ContractPayClassification(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	rec := call(t, h, http.MethodGet, "/api/v1/contracts/forms", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("دریافت فرم‌ها باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "payModels") || !strings.Contains(body, "payGroups") {
		t.Fatal("پاسخ فرم‌ها باید فهرست روش‌های پرداخت و گروه‌های جدول ۳ را داشته باشد")
	}
	var forms struct {
		PayModels []map[string]any `json:"payModels"`
		PayGroups []map[string]any `json:"payGroups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &forms); err != nil {
		t.Fatalf("پاسخ فرم‌ها قابل تجزیه نیست: %v", err)
	}
	if len(forms.PayModels) != 4 {
		t.Fatalf("چهار روش پرداخت انتظار می‌رفت، %d آمد", len(forms.PayModels))
	}
	if len(forms.PayGroups) != 10 {
		t.Fatalf("ده گروه جدول ۳ انتظار می‌رفت، %d آمد", len(forms.PayGroups))
	}

	// ترکیب ناسازگار: صندوق نفت برای نوع استخدام غیررسمی.
	incompatible := map[string]any{
		"employeeId": "EMP-012", "kind": "fixed_term", "employmentType": "contractor_staff",
		"startDate": jdate(t, 10), "endDate": jdate(t, 400), "monthlySalary": 25000000,
		"payModel": "oil",
		"payMeta": map[string]any{
			"contractorName": "شرکت پیمانکار نمونه", "contractNo": "P-1405-90",
			"payBasis": "monthly", "payAmount": 25000000, "insurancePayer": "contractor",
		},
	}
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts", token, incompatible)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("صندوق نفت برای پیمانکار باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "payModel") {
		t.Fatalf("خطا باید روی فیلد روش پرداخت باشد: %s", rec.Body.String())
	}

	// قرارداد کارمند رسمی نفت: گروه یکم و روش صندوق نفت (خودکار از نوع استخدام).
	official := map[string]any{
		"employeeId": "EMP-012", "kind": "permanent", "employmentType": "official_oil",
		"startDate": jdate(t, 10), "monthlySalary": 42000000,
	}
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts", token, official)
	if rec.Code != http.StatusCreated {
		t.Fatalf("قرارداد رسمی نفت باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var created contracts.Contract
	decodeInto(t, rec, &created)
	if created.PayGroup != 1 {
		t.Fatalf("کارمند رسمی نفت باید گروه یکم باشد، %d بود", created.PayGroup)
	}
	if created.PayModel != rules.PayModelOil {
		t.Fatalf("روش پرداخت رسمی نفت باید صندوق نفت باشد، «%s» بود", created.PayModel)
	}
	if created.PayModelFa == "" || created.PayGroupFa == "" {
		t.Fatal("عنوان فارسی گروه و روش پرداخت باید پر شود")
	}

	// پاک‌کردن پیش‌نویس تا کارمند آزمون برای حالت بعدی آزاد شود.
	if rec := call(t, h, http.MethodDelete, "/api/v1/contracts/contracts/"+created.ID, token, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("حذف پیش‌نویس باید ۲۰۴ بدهد، %d داد", rec.Code)
	}

	// قرارداد پیمانکاری: پیش‌فرض روش پرداخت سازمان و گروه دوم (پایهٔ سمت ۶ تا ۱۰).
	contracting := map[string]any{
		"employeeId": "EMP-012", "kind": "fixed_term", "employmentType": "contractor_staff",
		"startDate": jdate(t, 10), "endDate": jdate(t, 400), "monthlySalary": 25000000,
		"payMeta": map[string]any{
			"contractorName": "شرکت پیمانکار نمونه", "contractNo": "P-1405-91",
			"payBasis": "monthly", "payAmount": 25000000, "insurancePayer": "contractor",
		},
	}
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/contracts", token, contracting)
	if rec.Code != http.StatusCreated {
		t.Fatalf("قرارداد پیمانکاری باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var contractingSaved contracts.Contract
	decodeInto(t, rec, &contractingSaved)
	if contractingSaved.PayModel != rules.PayModelPayroll {
		t.Fatalf("پیش‌فرض پیمانکار باید حقوق و دستمزد جاری باشد، «%s» بود", contractingSaved.PayModel)
	}
	if contractingSaved.PayGroup != 2 {
		t.Fatalf("گروه پیش‌فرض پیمانکار باید دوم باشد، %d بود", contractingSaved.PayGroup)
	}
}

// TestPAY05DecreePayClassification می‌سنجد حکم کارگزینی هم گروه و روش پرداخت را می‌نویسد:
// کارمند رسمی نفت گروه یکم می‌گیرد (نه گروه گروه شغلی‌اش) و ترکیب ناسازگار رد می‌شود.
func TestPAY05DecreePayClassification(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	// ترکیب ناسازگار: رسمی نفت با «حقوق و دستمزد جاری».
	rec := call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees", token, map[string]any{
		"employeeId": "EMP-004", "employmentType": "official_oil", "jobGroup": "کارشناس ارشد",
		"baseSalary": 245000000, "effectiveDate": jdate(t, 1), "reasonFa": "افزایش سنواتی",
		"payModel": "payroll",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("رسمی نفت با حقوق و دستمزد جاری باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// حکم درست: گروه یکم و روش صندوق نفت، هرچند گروه شغلی «کارشناس ارشد» است.
	rec = call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees", token, map[string]any{
		"employeeId": "EMP-004", "employmentType": "official_oil", "jobGroup": "کارشناس ارشد",
		"baseSalary": 245000000, "effectiveDate": jdate(t, 1), "reasonFa": "افزایش سنواتی",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("صدور حکم باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var decree contracts.SalaryDecree
	decodeInto(t, rec, &decree)
	if decree.PayGroup != 1 {
		t.Fatalf("حکم کارمند رسمی نفت باید گروه یکم باشد، %d بود", decree.PayGroup)
	}
	if decree.PayModel != rules.PayModelOil {
		t.Fatalf("روش پرداخت حکم رسمی نفت باید صندوق نفت باشد، «%s» بود", decree.PayModel)
	}
	if decree.PayGroupFa == "" || decree.PayModelFa == "" {
		t.Fatal("عنوان فارسی گروه و روش پرداخت در حکم باید پر شود")
	}
}

// TestPAY06ClassificationReport می‌سنجد گزارش دسته‌بندی کارکنان، دسته‌ها را با شمار و
// فهرست کارکنان برمی‌گرداند و فهرست مرجع روش‌ها و گروه‌ها را هم می‌دهد.
func TestPAY06ClassificationReport(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	rec := call(t, h, http.MethodGet, "/api/v1/contracts/pay-classification", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("گزارش دسته‌بندی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var report struct {
		Rows       []map[string]any `json:"rows"`
		Total      int              `json:"total"`
		GroupCount int              `json:"groupCount"`
		Models     []map[string]any `json:"models"`
		Groups     []map[string]any `json:"groups"`
		NoteFa     string           `json:"noteFa"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("پاسخ گزارش قابل تجزیه نیست: %v", err)
	}
	if report.Total == 0 || len(report.Rows) == 0 {
		t.Fatalf("گزارش باید دسته‌های قراردادهای نمونه را نشان دهد: %+v", report)
	}
	if report.GroupCount != len(report.Rows) {
		t.Fatalf("شمار دسته‌ها ناهمخوان است: %d ≠ %d", report.GroupCount, len(report.Rows))
	}
	if len(report.Models) != 4 || len(report.Groups) != 10 {
		t.Fatalf("فهرست مرجع روش‌ها/گروه‌ها ناقص است: %d/%d", len(report.Models), len(report.Groups))
	}
	if strings.TrimSpace(report.NoteFa) == "" {
		t.Fatal("گزارش باید توضیح فارسی داشته باشد")
	}
	for _, row := range report.Rows {
		if row["payModel"] == nil || row["payGroup"] == nil || row["employmentType"] == nil {
			t.Fatalf("هر دسته باید نوع استخدام، گروه و روش پرداخت داشته باشد: %+v", row)
		}
	}
}

// TestPAY07OfficialOilKeyConsistency می‌سنجد کلید «کارمند رسمی نفت» در بستهٔ قواعد با فهرست
// انواع استخدام سامانه یکی است؛ اگر روزی کلید عوض شود، همین آزمون می‌گیرد.
func TestPAY07OfficialOilKeyConsistency(t *testing.T) {
	employmentType, ok := contracts.EmploymentTypeByKey(rules.OilOfficialEmploymentKey)
	if !ok {
		t.Fatalf("کلید «%s» در فهرست انواع استخدام نیست", rules.OilOfficialEmploymentKey)
	}
	if employmentType.Kind != contracts.EmploymentKindOfficial {
		t.Fatalf("کارمند رسمی نفت باید از خانوادهٔ رسمی باشد: %+v", employmentType)
	}
	if group, ok := rules.OilPayGroupOfEmployment(employmentType.Key); !ok || group.Number != 1 {
		t.Fatalf("کلید کارمند رسمی نفت باید گروه یکم بدهد: %+v", group)
	}
}

// TestPAY09ClassificationBackfill می‌سنجد عملیات «ثبت دستهٔ پرداخت روی اسناد» اجرا می‌شود،
// سندهای نمونه را به‌روز می‌کند و اجرای دوم چیزی برای تغییر نمی‌یابد (تکرارپذیر است).
func TestPAY09ClassificationBackfill(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	rec := call(t, h, http.MethodPost, "/api/v1/contracts/pay-classification/backfill", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("ثبت دستهٔ پرداخت باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var first struct {
		Updated int    `json:"updated"`
		NoteFa  string `json:"noteFa"`
	}
	decodeInto(t, rec, &first)
	if first.Updated == 0 {
		t.Fatal("دست‌کم یک قرارداد نمونه باید دستهٔ پرداخت بگیرد")
	}
	if strings.TrimSpace(first.NoteFa) == "" {
		t.Fatal("پاسخ باید توضیح فارسی داشته باشد")
	}

	// اجرای دوم: هیچ سندی برای تغییر نمانده است.
	rec = call(t, h, http.MethodPost, "/api/v1/contracts/pay-classification/backfill", token, nil)
	var second struct {
		Updated int `json:"updated"`
	}
	decodeInto(t, rec, &second)
	if second.Updated != 0 {
		t.Fatalf("اجرای دوم باید بدون تغییر باشد، %d سند عوض شد", second.Updated)
	}

	// اسناد به‌روزشده باید در گزارش دسته‌بندی با گروه و روش پرداخت دیده شوند.
	rec = call(t, h, http.MethodGet, "/api/v1/contracts/pay-classification", token, nil)
	var report struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("گزارش دسته‌بندی قابل تجزیه نیست: %v", err)
	}
	for _, row := range report.Rows {
		if row["payModel"] == "" || row["payModel"] == nil {
			t.Fatalf("هر دسته باید روش پرداخت داشته باشد: %+v", row)
		}
	}
}
