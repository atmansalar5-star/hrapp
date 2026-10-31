// آزمون جریان سرتاسری «فایل قانونی بیمه/مالیات و فایل پرداخت بانکی» — زیرکارهای ۱.۴ و
// ۱.۵ فاز ۱: ساخت فایل از کارکرد دوره، خروجی CSV، ثبت سند گردش و گذر وضعیت‌ها.

package bootstrap

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// testIBAN یک شبای معتبر می‌سازد (محاسبهٔ مستقل از کد تولید).
func testIBAN(body string) string {
	for i := 0; i < 100; i++ {
		candidate := fmt.Sprintf("IR%02d%s", i, body)
		remainder := 0
		for _, ch := range candidate[2:] + "1827" {
			remainder = (remainder*10 + int(ch-'0')) % 97
		}
		if remainder == 1 {
			return candidate
		}
	}
	return ""
}

func TestStatutoryAndBankFileFlow(t *testing.T) {
	h := newTestServer(t)
	hr := login(t, h, "hr.manager", "demo").AccessToken
	emp := login(t, h, "employee", "demo").AccessToken
	period := jdate(t, 0)[:7]

	// ---------- حساب بانکی کارمند: شبای نامعتبر باید رد شود ----------
	rec := call(t, h, http.MethodPost, "/api/v1/payroll/records", hr, map[string]any{
		"kind": "bankAccounts", "title": "حساب آزمون", "employeeId": "EMP-005",
		"fields": map[string]string{"bankName": "بانک ملی", "iban": "IR000000000000000000000000"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("شبای نامعتبر باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	iban := testIBAN("5401026800208179090020")
	if iban == "" {
		t.Fatal("ساخت شبای آزمون ناموفق بود")
	}
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/records", hr, map[string]any{
		"kind": "bankAccounts", "title": "حساب آزمون", "employeeId": "EMP-005",
		"fields": map[string]string{
			"bankName": "بانک ملی", "iban": iban, "accountNo": "1234567890", "nationalId": "1234567891",
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت حساب بانکی باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var account contracts.Record
	decodeInto(t, rec, &account)
	if !strings.HasPrefix(account.ID, "BNK-") {
		t.Errorf("شناسهٔ حساب بانکی باید با BNK- آغاز شود، «%s» است", account.ID)
	}
	if code := call(t, h, http.MethodPost, "/api/v1/payroll/records/"+account.ID+"/action", hr,
		map[string]any{"action": "activate"}).Code; code != http.StatusOK {
		t.Fatalf("فعال‌سازی حساب بانکی باید ۲۰۰ بدهد، %d داد", code)
	}

	// ---------- فایل بیمه: ردیف‌ها، جمع‌ها، مهلت و جریمه ----------
	rec = call(t, h, http.MethodGet, "/api/v1/payroll/statutory/insurance?period="+period, hr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("گزارش فایل بیمه باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var insurance struct {
		Kind         string   `json:"kind"`
		TitleFa      string   `json:"titleFa"`
		Period       string   `json:"period"`
		ColumnsFa    []string `json:"columnsFa"`
		DeadlineFa   string   `json:"deadlineFa"`
		PayableTotal int64    `json:"payableTotal"`
		Totals       struct {
			Rows          int   `json:"rows"`
			Gross         int64 `json:"gross"`
			InsuranceSum  int64 `json:"insuranceTotal"`
			EmployeeShare int64 `json:"employeeShare"`
			EmployerShare int64 `json:"employerShare"`
			Unemployment  int64 `json:"unemploymentShare"`
		} `json:"totals"`
		Rows2 []struct {
			EmployeeID string `json:"employeeId"`
			Days       int    `json:"days"`
			NationalID string `json:"nationalId"`
		} `json:"rows"`
		NoteFa string `json:"noteFa"`
	}
	decodeInto(t, rec, &insurance)
	if insurance.Kind != "insurance" || insurance.Period != period {
		t.Errorf("شناسه/دورهٔ فایل بیمه درست نیست: %+v", insurance)
	}
	if insurance.Totals.Rows < 10 || insurance.Totals.InsuranceSum <= 0 {
		t.Errorf("فایل بیمه باید همهٔ کارکنان با جمع مثبت را داشته باشد: %+v", insurance.Totals)
	}
	if insurance.Totals.EmployeeShare+insurance.Totals.EmployerShare+insurance.Totals.Unemployment !=
		insurance.Totals.InsuranceSum {
		t.Errorf("جمع سه سهم بیمه باید با جمع کل بخواند: %+v", insurance.Totals)
	}
	if insurance.DeadlineFa == "" || len(insurance.ColumnsFa) != 11 {
		t.Errorf("مهلت قانونی و ستون‌های فایل بیمه باید پر باشند: %+v", insurance)
	}
	if insurance.PayableTotal != insurance.Totals.InsuranceSum {
		t.Errorf("مبلغ قابل پرداخت فایل بیمه باید جمع سهم‌ها باشد: %d != %d",
			insurance.PayableTotal, insurance.Totals.InsuranceSum)
	}
	if insurance.NoteFa == "" {
		t.Errorf("یادداشت فارسی فایل بیمه باید پر باشد")
	}

	// ---------- فایل مالیات ----------
	tax := call(t, h, http.MethodGet, "/api/v1/payroll/statutory/tax?period="+period, hr, nil)
	if tax.Code != http.StatusOK {
		t.Fatalf("گزارش فایل مالیات باید ۲۰۰ بدهد، %d داد: %s", tax.Code, tax.Body.String())
	}
	var taxBody struct {
		Totals struct {
			Tax           int64 `json:"tax"`
			TaxableIncome int64 `json:"taxableIncome"`
		} `json:"totals"`
		ColumnsFa []string `json:"columnsFa"`
	}
	decodeInto(t, tax, &taxBody)
	if len(taxBody.ColumnsFa) != 9 {
		t.Errorf("ستون‌های فایل مالیات باید ۹ ستون باشد: %+v", taxBody.ColumnsFa)
	}

	// ---------- خروجی CSV ----------
	rec = call(t, h, http.MethodGet, "/api/v1/payroll/statutory/insurance/file?period="+period, hr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("خروجی CSV بیمه باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("نوع محتوای CSV درست نیست: «%s»", ct)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "\ufeff") {
		t.Errorf("خروجی CSV باید BOM داشته باشد تا Excel فارسی را درست نشان دهد")
	}
	if !strings.Contains(body, "سهم بیمه‌شده") || !strings.Contains(body, "جمع کل") {
		t.Errorf("سرصفحه و ردیف جمع فایل بیمه در CSV نیست: %s", firstRun(body, 200))
	}

	// ---------- ثبت سند فایل و گذر وضعیت ----------
	rec = call(t, h, http.MethodPost, "/api/v1/payroll/statutory/insurance?period="+period, hr, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت سند فایل بیمه باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Record contracts.Record `json:"record"`
	}
	decodeInto(t, rec, &created)
	if !strings.HasPrefix(created.Record.ID, "STF-") {
		t.Errorf("شناسهٔ سند فایل قانونی باید با STF- آغاز شود، «%s» است", created.Record.ID)
	}
	if created.Record.Fields["fileType"] != "بیمه" || created.Record.Fields["period"] != period {
		t.Errorf("فیلدهای سند فایل درست نیست: %+v", created.Record.Fields)
	}
	act := func(action string, fields map[string]string) int {
		body := map[string]any{"action": action}
		if fields != nil {
			body["fields"] = fields
		}
		return call(t, h, http.MethodPost, "/api/v1/payroll/records/"+created.Record.ID+"/action", hr, body).Code
	}
	if code := act("markSent", map[string]string{"trackingNo": "1405081234"}); code != http.StatusOK {
		t.Fatalf("ثبت ارسال فایل باید ۲۰۰ بدهد، %d داد", code)
	}
	if code := act("markPaid", map[string]string{"receiptNo": "R-99", "paidAt": jdate(t, 0)}); code != http.StatusOK {
		t.Fatalf("ثبت پرداخت فایل باید ۲۰۰ بدهد، %d داد", code)
	}
	got := call(t, h, http.MethodGet, "/api/v1/payroll/records/"+created.Record.ID, hr, nil)
	var single contracts.Record
	decodeInto(t, got, &single)
	if single.Status != "paid" {
		t.Errorf("وضعیت پایانی سند فایل باید paid باشد، «%s» شد", single.Status)
	}
	if single.Fields["trackingNo"] != "1405081234" {
		t.Errorf("شمارهٔ پیگیری باید روی سند ثبت شود: %+v", single.Fields)
	}

	// ---------- فایل پرداخت بانکی ----------
	// نام بانک در پرس‌وجو کدگذاری می‌شود چون فاصله دارد (وگرنه آدرس معتبر نیست).
	bankQuery := url.QueryEscape("بانک ملی")
	rec = call(t, h, http.MethodGet, "/api/v1/payroll/bank-file?period="+period+"&bank="+bankQuery, hr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("گزارش فایل پرداخت باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var bank struct {
		Period    string   `json:"period"`
		ColumnsFa []string `json:"columnsFa"`
		ControlFa string   `json:"controlFa"`
		RoundUnit int64    `json:"roundUnit"`
		Rows      []struct {
			EmployeeID string `json:"employeeId"`
			Amount     int64  `json:"amount"`
			RawAmount  int64  `json:"rawAmount"`
			Rounding   int64  `json:"rounding"`
			IBAN       string `json:"iban"`
			StatusFa   string `json:"statusFa"`
		} `json:"rows"`
		Totals struct {
			Rows     int   `json:"rows"`
			Amount   int64 `json:"amount"`
			Rounding int64 `json:"rounding"`
		} `json:"totals"`
		MissingFa []string `json:"missingAccountsFa"`
		NoteFa    string   `json:"noteFa"`
	}
	decodeInto(t, rec, &bank)
	if bank.Totals.Rows != 1 {
		t.Fatalf("فقط کارمند دارای حساب فعال باید در فایل باشد، %d ردیف آمد: %+v", bank.Totals.Rows, bank.Rows)
	}
	if bank.Rows[0].EmployeeID != "EMP-005" || bank.Rows[0].IBAN != iban {
		t.Errorf("ردیف پرداخت درست ساخته نشد: %+v", bank.Rows[0])
	}
	if bank.Rows[0].Amount%bank.RoundUnit != 0 {
		t.Errorf("مبلغ پرداخت باید مضرب %d باشد: %d", bank.RoundUnit, bank.Rows[0].Amount)
	}
	if bank.Rows[0].Amount-bank.Rows[0].RawAmount != bank.Rows[0].Rounding {
		t.Errorf("تفاوت گرد کردن درست نیست: %+v", bank.Rows[0])
	}
	if bank.Totals.Amount != bank.Rows[0].Amount || bank.ControlFa == "" {
		t.Errorf("جمع کنترلی فایل پرداخت درست نیست: %+v", bank)
	}
	if len(bank.MissingFa) == 0 {
		t.Errorf("کارمندان بدون حساب فعال باید گزارش شوند")
	}

	rec = call(t, h, http.MethodGet, "/api/v1/payroll/bank-file/download?period="+period+"&bank="+bankQuery, hr, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "شمارهٔ شبا") {
		t.Errorf("خروجی CSV فایل پرداخت درست نیست (%d): %s", rec.Code, firstRun(rec.Body.String(), 160))
	}

	rec = call(t, h, http.MethodPost, "/api/v1/payroll/bank-file?period="+period+"&bank="+bankQuery, hr, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("ثبت سند فایل پرداخت باید ۲۰۱ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var bankCreated struct {
		Record contracts.Record `json:"record"`
	}
	decodeInto(t, rec, &bankCreated)
	if !strings.HasPrefix(bankCreated.Record.ID, "BNKF-") {
		t.Errorf("شناسهٔ سند فایل پرداخت باید با BNKF- آغاز شود، «%s» است", bankCreated.Record.ID)
	}
	if bankCreated.Record.Fields["controlFa"] == "" {
		t.Errorf("جمع کنترلی باید روی سند فایل پرداخت نوشته شود: %+v", bankCreated.Record.Fields)
	}

	// ---------- دسترسی: کارمند نباید فایل بسازد ----------
	if code := call(t, h, http.MethodPost,
		"/api/v1/payroll/statutory/insurance?period="+period, emp, nil).Code; code != http.StatusForbidden {
		t.Errorf("ساخت فایل قانونی توسط کارمند باید ۴۰۳ بدهد، %d داد", code)
	}
	if code := call(t, h, http.MethodPost,
		"/api/v1/payroll/bank-file?period="+period, emp, nil).Code; code != http.StatusForbidden {
		t.Errorf("ساخت فایل پرداخت توسط کارمند باید ۴۰۳ بدهد، %d داد", code)
	}
}

// firstRun نخستین بخش متن را برای پیام خطای کوتاه برمی‌گرداند.
func firstRun(text string, limit int) string {
	runes := []rune(strings.ReplaceAll(text, "\n", " | "))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + " …"
}
