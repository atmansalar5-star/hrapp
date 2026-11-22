// آزمون بندهای ASG-08 و ASG-09: بلوک تخصیص‌ها کنار فیش، و ثبت صفحهٔ رابط در مانیفست.
// ارجاع: docs/allowance-assignment-spec-fa.md

package bootstrap

import (
	"net/http"
	"testing"
)

// TestASG08PayslipAssignmentsBlock می‌سنجد پاسخ فیش یکپارچه، بلوک تخصیص‌ها را با سطرهای
// کددار و جمع‌های مشمول همراه دارد.
func TestASG08PayslipAssignmentsBlock(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	rec := call(t, h, http.MethodPost, "/api/v1/payroll/payslip", manager.AccessToken,
		map[string]any{"employeeId": "EMP-001"})
	if rec.Code != http.StatusOK {
		t.Fatalf("فیش یکپارچه باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var answer struct {
		Assignments struct {
			Lines []struct {
				Code     string `json:"code"`
				Amount   int64  `json:"amount"`
				Taxable  bool   `json:"taxable"`
				Insured  bool   `json:"insured"`
				Category string `json:"category"`
			} `json:"lines"`
			Totals struct {
				Earnings   int64 `json:"earnings"`
				Deductions int64 `json:"deductions"`
				Lines      int   `json:"lines"`
			} `json:"totals"`
		} `json:"assignments"`
	}
	decodeInto(t, rec, &answer)
	if answer.Assignments.Totals.Lines == 0 || len(answer.Assignments.Lines) == 0 {
		t.Fatalf("بلوک تخصیص‌ها باید سطر داشته باشد: %s", rec.Body.String())
	}
	codes := map[string]bool{}
	for _, line := range answer.Assignments.Lines {
		codes[line.Code] = true
		if line.Code == "" {
			t.Fatalf("هر سطر تخصیص باید کد رسمی داشته باشد: %+v", line)
		}
	}
	if !codes["151"] {
		t.Fatalf("تخصیص گروهی مسکن (۱۵۱) باید در بلوک فیش باشد: %+v", codes)
	}
	if answer.Assignments.Totals.Earnings <= 0 {
		t.Fatalf("جمع مزایای تخصیص باید مثبت باشد: %+v", answer.Assignments.Totals)
	}
}

// TestASG09AllowanceScreenRegistered می‌سنجد صفحهٔ «تخصیص فوق العاده‌های حقوق» در مانیفست
// ماژول حقوق ثبت شده و کامپوننت اپ موبایل را اعلام کرده است.
func TestASG09AllowanceScreenRegistered(t *testing.T) {
	deps, err := NewDeps(Config{
		JWTSecret: []byte("test-secret-key-for-allowance-screen"),
		DataDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("ساخت وابستگی‌ها ناموفق: %v", err)
	}
	module, ok := deps.Registry.ByID("payroll")
	if !ok {
		t.Fatal("ماژول حقوق در رجیستری نیست")
	}
	for _, screen := range module.Screens {
		if screen.Key != "allowanceAssign" {
			continue
		}
		if screen.Component != "AllowanceAssignScreen" {
			t.Fatalf("کامپوننت صفحهٔ تخصیص باید AllowanceAssignScreen باشد: %s", screen.Component)
		}
		if screen.TitleFa == "" {
			t.Fatal("صفحهٔ تخصیص باید عنوان فارسی داشته باشد")
		}
		return
	}
	t.Fatal("صفحهٔ allowanceAssign در مانیفست ماژول حقوق ثبت نشده است")
}
