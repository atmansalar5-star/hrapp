// آزمون مسیر یکپارچهٔ فیش بر پایهٔ روش پرداخت.
// ارجاع: docs/pay-classification-spec-fa.md بند PAY-10.

package bootstrap

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// payslipPayload بدنهٔ پاسخ مسیر یکپارچهٔ فیش.
type payslipPayload struct {
	PayModel   string         `json:"payModel"`
	PayModelFa string         `json:"payModelFa"`
	SourceFa   string         `json:"sourceFa"`
	EngineFa   string         `json:"engineFa"`
	NotesFa    []string       `json:"notesFa"`
	Slip       map[string]any `json:"slip"`
}

// TestPAY10PayslipDispatch می‌سنجد یک نقطهٔ ورود، موتور را بر پایهٔ روش پرداخت همان کارمند
// انتخاب می‌کند: کارمند رسمی نفت ⇒ فیش صندوق نفت (گروه یکم)، کارمند غیررسمی ⇒ موتور جاری؛
// و روش پرداخت ناسازگار از راه درخواست پذیرفته نمی‌شود.
func TestPAY10PayslipDispatch(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")
	token := manager.AccessToken

	// ۱) فهرست پیش‌فرض‌ها: چهار موتور و کارکنان با روش پرداخت مؤثرشان.
	rec := call(t, h, http.MethodGet, "/api/v1/payroll/payslip-defaults", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست پیش‌فرض فیش باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var defaults struct {
		Engines   []map[string]string `json:"engines"`
		Employees []map[string]string `json:"employees"`
		NoteFa    string              `json:"noteFa"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &defaults); err != nil {
		t.Fatalf("پاسخ فهرست پیش‌فرض قابل تجزیه نیست: %v", err)
	}
	if len(defaults.Engines) != 4 {
		t.Fatalf("چهار موتور پرداخت انتظار می‌رفت، %d آمد", len(defaults.Engines))
	}
	if len(defaults.Employees) == 0 {
		t.Fatal("فهرست کارکنان نباید خالی باشد")
	}
	officialOil, nonOfficial := "", ""
	for _, e := range defaults.Employees {
		if e["payModel"] == "" {
			t.Fatalf("هر کارمند باید روش پرداخت مؤثر داشته باشد: %+v", e)
		}
		switch {
		case e["employmentType"] == "official_oil" && officialOil == "":
			officialOil = e["id"]
		case e["employmentType"] != "official_oil" && nonOfficial == "":
			nonOfficial = e["id"]
		}
	}
	if officialOil == "" {
		t.Fatal("کارمند رسمی نفت در دادهٔ نمونه پیدا نشد")
	}

	// ۲) کارمند رسمی نفت: موتور صندوق نفت و گروه یکم جدول ۳.
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/payslip", token,
		map[string]any{"employeeId": officialOil})
	if rec.Code != http.StatusOK {
		t.Fatalf("فیش کارمند رسمی نفت باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var oil payslipPayload
	decodeInto(t, rec, &oil)
	if oil.PayModel != "oil" {
		t.Fatalf("کارمند رسمی نفت باید با نظام صندوق نفت فیش شود، «%s» آمد", oil.PayModel)
	}
	if !strings.Contains(oil.EngineFa, "صندوق") {
		t.Fatalf("نام موتور باید صندوق نفت باشد: %s", oil.EngineFa)
	}
	if group, _ := oil.Slip["payGroup"].(float64); int(group) != 1 {
		t.Fatalf("فیش کارمند رسمی نفت باید گروه یکم داشته باشد: %v", oil.Slip["payGroup"])
	}
	if oil.SourceFa == "" || len(oil.NotesFa) == 0 {
		t.Fatalf("منبع روش پرداخت و یادداشت‌ها باید پر باشند: %+v", oil)
	}

	// ۳) کارمند غیررسمی: موتور جاری سازمان (نه صندوق نفت) با خالص محاسبه‌شده.
	if nonOfficial == "" {
		t.Skip("کارمند غیررسمی در دادهٔ نمونه نیست؛ بخش موتور جاری آزمون نشد")
	}
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/payslip", token,
		map[string]any{"employeeId": nonOfficial})
	if rec.Code != http.StatusOK {
		t.Fatalf("فیش کارمند غیررسمی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var current payslipPayload
	decodeInto(t, rec, &current)
	if current.PayModel == "oil" {
		t.Fatalf("صندوق نفت برای کارمند غیررسمی مجاز نیست: %+v", current)
	}
	if current.Slip["net"] == nil || current.Slip["gross"] == nil {
		t.Fatalf("فیش موتور جاری باید ناخالص و خالص داشته باشد: %+v", current.Slip)
	}

	// ۴) تحمیل روش ناسازگار از راه درخواست: ۴۲۲.
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/payslip", token,
		map[string]any{"employeeId": nonOfficial, "payModel": "oil"})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("تحمیل صندوق نفت برای کارمند غیررسمی باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// ۵) کارمند ناشناخته: ۴۰۴.
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/payslip", token,
		map[string]any{"employeeId": "EMP-999"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("کارمند ناشناخته باید ۴۰۴ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
}
