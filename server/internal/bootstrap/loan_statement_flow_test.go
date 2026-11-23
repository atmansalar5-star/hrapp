// آزمون جریان صورت حساب و اقساط وام: کارتابل و صورت حساب، ساخت جدول اقساط، کسر قسط در
// فیش حقوقی و گزارش‌گیری. ارجاع: docs/loan-statement-spec-fa.md (بندهای LOAN-07 تا LOAN-10)

package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// loanInstallmentRecordID شناسهٔ رکورد قسط یک وام با شمارهٔ قسط داده‌شده را می‌یابد.
func loanInstallmentRecordID(t *testing.T, h http.Handler, token, loanID string, seq int) string {
	t.Helper()
	rec := call(t, h, http.MethodGet, "/api/v1/loans/records?kind=loanInstallments&size=500", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فهرست اقساط باید ۲۰۰ بدهد، %d داد", rec.Code)
	}
	var page contracts.RecordPage
	decodeInto(t, rec, &page)
	for _, item := range page.Items {
		if strings.TrimSpace(item.Fields["loan"]) != loanID {
			continue
		}
		if strings.TrimSpace(item.Fields["seq"]) == itoaTest(seq) {
			return item.ID
		}
	}
	t.Fatalf("قسط شمارهٔ %d وام %s یافت نشد", seq, loanID)
	return ""
}

func itoaTest(v int) string {
	if v == 0 {
		return "0"
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	return out
}

// TestLOAN07Statement کارتابل صورت حساب، صورت حساب یک وام، پیش‌نمایش جدول اقساط و ساخت
// جدول اقساط (تکرارپذیر) را می‌سنجد.
func TestLOAN07Statement(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	// ---------- کارتابل صورت حساب‌ها ----------
	rec := call(t, h, http.MethodGet, "/api/v1/loans/statements?period=1404/04", manager.AccessToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("کارتابل صورت حساب باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var page contracts.LoanStatementPage
	decodeInto(t, rec, &page)
	if page.Count == 0 || len(page.Items) != page.Count {
		t.Fatalf("کارتابل صورت حساب خالی است: %+v", page)
	}
	if page.PeriodFa != "1404/04" || page.TodayFa == "" {
		t.Fatalf("دوره و تاریخ کارتابل درست نیست: %+v", page)
	}
	if len(page.CodeGaps) == 0 {
		t.Fatal("انواع وام بدون کد رسمی باید صریح گزارش شوند")
	}
	var housing *contracts.LoanStatementView
	for i := range page.Items {
		if page.Items[i].LoanID == "LOAN-0001" {
			housing = &page.Items[i]
		}
	}
	if housing == nil {
		t.Fatalf("وام نمونهٔ ودیعهٔ مسکن در کارتابل نیست: %+v", page.Items)
	}
	if housing.Statement.Installments != 60 || len(housing.Statement.Schedule) != 60 {
		t.Fatalf("جدول اقساط وام مسکن باید ۶۰ قسط باشد: %+v", housing.Statement)
	}
	if housing.Statement.Balance.RemainingDebt != housing.Statement.TotalPayable {
		t.Fatalf("پیش از هر کسر، مانده بدهی باید برابر جمع قابل بازپرداخت باشد: %+v", housing.Statement.Balance)
	}
	if housing.Statement.FirstPeriodFa != "1404/04" || housing.Statement.LastPeriodFa == "" {
		t.Fatalf("ماه شروع/پایان اقساط درست نیست: %+v", housing.Statement)
	}
	if page.Totals.RemainingDebt <= 0 || page.Totals.Amount <= 0 {
		t.Fatalf("جمع‌های کارتابل باید مثبت باشند: %+v", page.Totals)
	}

	// ---------- پیش‌نمایش جدول اقساط (بدون ذخیره‌سازی) ----------
	preview := call(t, h, http.MethodPost, "/api/v1/loans/schedule/preview", manager.AccessToken,
		map[string]any{"loanType": "emergency", "amount": 24_000_000, "installments": 12, "receivedDate": "1405/01/10"})
	if preview.Code != http.StatusOK {
		t.Fatalf("پیش‌نمایش جدول اقساط باید ۲۰۰ بدهد، %d داد: %s", preview.Code, preview.Body.String())
	}
	var plan contracts.LoanSchedulePreviewResult
	decodeInto(t, preview, &plan)
	if len(plan.Schedule.Rows) != 12 {
		t.Fatalf("پیش‌نمایش باید ۱۲ قسط بدهد، %d داد", len(plan.Schedule.Rows))
	}
	var sum int64
	for _, row := range plan.Schedule.Rows {
		sum += row.Amount
	}
	if sum != plan.Schedule.TotalPayable {
		t.Fatalf("جمع اقساط (%d) باید برابر جمع قابل بازپرداخت (%d) باشد", sum, plan.Schedule.TotalPayable)
	}
	if plan.PaymentCode != "500" || plan.InstallmentCode != "5400" {
		t.Fatalf("کدهای رسمی وام اضطراری باید ۵۰۰/۵۴۰۰ باشد: %+v", plan)
	}
	if plan.Schedule.Rows[0].PeriodFa != "1405/02" {
		t.Fatalf("شروع اقساط باید ماه بعد از پرداخت باشد، %q شد", plan.Schedule.Rows[0].PeriodFa)
	}

	// ---------- ساخت جدول اقساط روی رکوردهای قسط ----------
	build := call(t, h, http.MethodPost, "/api/v1/loans/LOAN-0001/schedule", manager.AccessToken, nil)
	if build.Code != http.StatusOK {
		t.Fatalf("ساخت جدول اقساط باید ۲۰۰ بدهد، %d داد: %s", build.Code, build.Body.String())
	}
	var built contracts.LoanScheduleBuildResult
	decodeInto(t, build, &built)
	if built.Created != 60 {
		t.Fatalf("باید ۶۰ رکورد قسط ساخته شود، %d ساخته شد: %s", built.Created, build.Body.String())
	}
	if built.Statement.InstallmentRecords != 60 {
		t.Fatalf("صورت حساب باید ۶۰ رکورد قسط نشان دهد، %d داد", built.Statement.InstallmentRecords)
	}
	// تکرارپذیری: بار دوم نباید سطر تکراری بسازد.
	again := call(t, h, http.MethodPost, "/api/v1/loans/LOAN-0001/schedule", manager.AccessToken, nil)
	if again.Code != http.StatusOK {
		t.Fatalf("ساخت دوبارهٔ جدول باید ۲۰۰ بدهد، %d داد", again.Code)
	}
	var rebuilt contracts.LoanScheduleBuildResult
	decodeInto(t, again, &rebuilt)
	if rebuilt.Created != 0 || rebuilt.Unchanged != 60 {
		t.Fatalf("ساخت دوباره نباید قسط تکراری بسازد: %+v", rebuilt)
	}

	// ---------- صورت حساب یک وام ----------
	one := call(t, h, http.MethodGet, "/api/v1/loans/statement?id=LOAN-0001&period=1404/04", manager.AccessToken, nil)
	if one.Code != http.StatusOK {
		t.Fatalf("صورت حساب وام باید ۲۰۰ بدهد، %d داد: %s", one.Code, one.Body.String())
	}
	var view contracts.LoanStatementView
	decodeInto(t, one, &view)
	if !view.Statement.Disbursed || view.Statement.ReceivedDateFa == "" {
		t.Fatalf("وام با تاریخ پرداخت باید پرداخت‌شده شمرده شود: %+v", view.Statement)
	}
	if len(view.Statement.History) != 61 {
		t.Fatalf("تاریخچه باید پرداخت یک‌باره + ۶۰ قسط باشد، %d رخداد داد", len(view.Statement.History))
	}
	if view.Statement.History[0].Kind != "disburse" || view.Statement.History[0].Amount != view.Statement.Amount {
		t.Fatalf("رخداد اول تاریخچه باید پرداخت یک‌بارهٔ کل وام باشد: %+v", view.Statement.History[0])
	}
	if view.Statement.CodeNoteFa == "" {
		t.Fatal("وضعیت کد رسمی وام باید توضیح داده شود")
	}
}

// TestLOAN08PayslipDeduction می‌سنجد اقساط وام در فیش کسر می‌شود، در بلوک «وام‌ها» دیده
// می‌شود، و پس از ثبت کسر از فیش دیگر کسر نمی‌گردد.
func TestLOAN08PayslipDeduction(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	// ---------- وام اضطراری تازه و گذر از گردشکار تا «در حال بازپرداخت» ----------
	created := call(t, h, http.MethodPost, "/api/v1/loans/records", manager.AccessToken, map[string]any{
		"kind": "loans", "title": "وام اضطراری — آزمون اقساط", "employeeId": "EMP-004",
		"fields": map[string]string{
			"loanType": "emergency", "management": "services_support", "amount": "12000000",
			"installments": "12", "feeRate": "0", "receivedDate": "1404/10/10",
		},
	})
	if created.Code != http.StatusCreated && created.Code != http.StatusOK {
		t.Fatalf("ثبت وام باید ۲۰۱ بدهد، %d داد: %s", created.Code, created.Body.String())
	}
	var loan contracts.Record
	decodeInto(t, created, &loan)
	if loan.Fields["installmentAmount"] != "1000000" || loan.Fields["firstPeriod"] != "1404/11" {
		t.Fatalf("مبلغ قسط و ماه شروع باید خودکار پر شود: %+v", loan.Fields)
	}
	for _, action := range []string{"approve", "disburse"} {
		act := call(t, h, http.MethodPost, "/api/v1/loans/records/"+loan.ID+"/action",
			manager.AccessToken, map[string]any{"action": action})
		if act.Code != http.StatusOK {
			t.Fatalf("اقدام %s باید ۲۰۰ بدهد، %d داد: %s", action, act.Code, act.Body.String())
		}
	}
	buildSchedule := call(t, h, http.MethodPost, "/api/v1/loans/"+loan.ID+"/schedule", manager.AccessToken, nil)
	if buildSchedule.Code != http.StatusOK {
		t.Fatalf("ساخت جدول اقساط وام تازه باید ۲۰۰ بدهد، %d داد: %s", buildSchedule.Code, buildSchedule.Body.String())
	}

	// ---------- فیش ماه پرداخت: سطر پرداخت یک‌بارهٔ تسهیلات ----------
	disburseSlip := call(t, h, http.MethodPost, "/api/v1/payroll/payslip", manager.AccessToken,
		map[string]any{"employeeId": "EMP-004", "periodFa": "1404/10"})
	if disburseSlip.Code != http.StatusOK {
		t.Fatalf("فیش ماه پرداخت باید ۲۰۰ بدهد، %d داد: %s", disburseSlip.Code, disburseSlip.Body.String())
	}
	var payMonth struct {
		Loans struct {
			Disbursements []struct {
				LoanID string `json:"loanId"`
				Amount int64  `json:"amount"`
				CodeFa string `json:"codeFa"`
			} `json:"disbursements"`
		} `json:"loans"`
	}
	decodeInto(t, disburseSlip, &payMonth)
	foundDisburse := false
	for _, d := range payMonth.Loans.Disbursements {
		if d.LoanID == loan.ID {
			foundDisburse = true
			if d.Amount != 12_000_000 || d.CodeFa != "500" {
				t.Fatalf("پرداخت یک‌بارهٔ وام اضطراری باید ۱۲ میلیون با کد ۵۰۰ باشد: %+v", d)
			}
		}
	}
	if !foundDisburse {
		t.Fatalf("ماه پرداخت باید سطر پرداخت یک‌بارهٔ وام را نشان دهد: %s", disburseSlip.Body.String())
	}

	// ---------- فیش ماه قسط: کسر واقعی از خالص ----------
	slip := call(t, h, http.MethodPost, "/api/v1/payroll/payslip", manager.AccessToken,
		map[string]any{"employeeId": "EMP-004", "periodFa": "1404/11"})
	if slip.Code != http.StatusOK {
		t.Fatalf("فیش ماه قسط باید ۲۰۰ بدهد، %d داد: %s", slip.Code, slip.Body.String())
	}
	var answer struct {
		Slip  map[string]any `json:"slip"`
		Loans struct {
			Lines []struct {
				LoanID    string `json:"loanId"`
				Seq       int    `json:"seq"`
				DueAmount int64  `json:"dueAmount"`
				Deducted  bool   `json:"deducted"`
				CodeFa    string `json:"codeFa"`
			} `json:"lines"`
			Totals struct {
				Deducted      int64 `json:"deducted"`
				RemainingDebt int64 `json:"remainingDebt"`
			} `json:"totals"`
		} `json:"loans"`
	}
	decodeInto(t, slip, &answer)
	if answer.Loans.Totals.Deducted != 1_000_000 {
		t.Fatalf("جمع کسر اقساط این ماه باید ۱ میلیون باشد: %+v", answer.Loans.Totals)
	}
	foundLine := false
	for _, l := range answer.Loans.Lines {
		if l.LoanID != loan.ID {
			continue
		}
		foundLine = true
		if l.Seq != 1 || !l.Deducted || l.DueAmount != 1_000_000 {
			t.Fatalf("سطر قسط همین ماه در فیش درست نیست: %+v", l)
		}
		if l.CodeFa != "5400" {
			t.Fatalf("سطر قسط باید کد رسمی ۵۴۰۰ داشته باشد: %+v", l)
		}
	}
	if !foundLine {
		t.Fatalf("سطر وام تازه در بلوک وام‌های فیش نیست: %+v", answer.Loans.Lines)
	}
	// تا وقتی کسر قسط ثبت نشده، مانده بدهی همان جمع قابل بازپرداخت است؛ فیش فقط کسر
	// را نشان می‌دهد و سند قسط با اقدام «ثبت پرداخت قسط» بسته می‌شود.
	if answer.Loans.Totals.RemainingDebt != 12_000_000 {
		t.Fatalf("مانده بدهی پیش از ثبت کسر باید ۱۲ میلیون باشد: %+v", answer.Loans.Totals)
	}
	if _, ok := answer.Slip["net"]; !ok {
		t.Fatalf("فیش باید خالص داشته باشد: %+v", answer.Slip)
	}

	// ---------- فهرست اقساط دوره برای پردازش حقوق ----------
	due := call(t, h, http.MethodGet, "/api/v1/loans/installments-due?period=1404/11", manager.AccessToken, nil)
	if due.Code != http.StatusOK {
		t.Fatalf("اقساط دوره باید ۲۰۰ بدهد، %d داد: %s", due.Code, due.Body.String())
	}
	var duePage contracts.LoanInstallmentDuePage
	decodeInto(t, due, &duePage)
	if duePage.PendingAmount != 1_000_000 {
		t.Fatalf("قسط همین ماه باید در انتظار کسر باشد: %+v", duePage)
	}

	// ---------- ثبت کسر قسط: از این پس در فیش کسر نمی‌شود ----------
	installmentID := loanInstallmentRecordID(t, h, manager.AccessToken, loan.ID, 1)
	pay := call(t, h, http.MethodPost, "/api/v1/loans/records/"+installmentID+"/action",
		manager.AccessToken, map[string]any{"action": "pay", "fields": map[string]string{"paidDate": "1404/11/20"}})
	if pay.Code != http.StatusOK {
		t.Fatalf("ثبت کسر قسط باید ۲۰۰ بدهد، %d داد: %s", pay.Code, pay.Body.String())
	}
	after := call(t, h, http.MethodPost, "/api/v1/payroll/payslip", manager.AccessToken,
		map[string]any{"employeeId": "EMP-004", "periodFa": "1404/11"})
	if after.Code != http.StatusOK {
		t.Fatalf("فیش پس از کسر باید ۲۰۰ بدهد، %d داد", after.Code)
	}
	var paid struct {
		Loans struct {
			Totals struct {
				Deducted       int64 `json:"deducted"`
				PreviouslyPaid int64 `json:"previouslyPaid"`
			} `json:"totals"`
		} `json:"loans"`
	}
	decodeInto(t, after, &paid)
	if paid.Loans.Totals.Deducted != 0 || paid.Loans.Totals.PreviouslyPaid != 1_000_000 {
		t.Fatalf("قسط کسرشده نباید دوباره کسر شود: %+v", paid.Loans.Totals)
	}
	// خالص فیش قسط‌کسرشده باید بزرگ‌تر از خالص فیش قسط‌دار باشد.
	netBefore, _ := answer.Slip["net"].(float64)
	netAfter := 0.0
	{
		var s struct {
			Slip map[string]any `json:"slip"`
		}
		decodeInto(t, after, &s)
		netAfter, _ = s.Slip["net"].(float64)
	}
	if netAfter <= netBefore {
		t.Fatalf("خالص فیش پس از کسر قسط باید بیشتر شود: قبل %v، بعد %v", netBefore, netAfter)
	}
	// پس از ثبت کسر، مانده بدهی صورت حساب یک قسط کم می‌شود.
	statement := call(t, h, http.MethodGet, "/api/v1/loans/statement?id="+loan.ID+"&period=1404/11", manager.AccessToken, nil)
	if statement.Code != http.StatusOK {
		t.Fatalf("صورت حساب پس از کسر باید ۲۰۰ بدهد، %d داد", statement.Code)
	}
	var view contracts.LoanStatementView
	decodeInto(t, statement, &view)
	if view.Statement.Balance.RemainingDebt != 11_000_000 || view.Statement.Balance.PaidCount != 1 {
		t.Fatalf("مانده بدهی پس از ثبت یک قسط باید ۱۱ میلیون و یک قسط کسرشده باشد: %+v", view.Statement.Balance)
	}
}

// TestLOAN09Report می‌سنجد مانده بدهی و اقساط روی منبع گزارش وام و منبع «اقساط وام‌ها»
// قابل گزارش‌گیری‌اند.
func TestLOAN09Report(t *testing.T) {
	h := newTestServer(t)
	manager := login(t, h, "hr.manager", "demo")

	build := call(t, h, http.MethodPost, "/api/v1/loans/LOAN-0001/schedule", manager.AccessToken, nil)
	if build.Code != http.StatusOK {
		t.Fatalf("ساخت جدول اقساط برای گزارش باید ۲۰۰ بدهد، %d داد: %s", build.Code, build.Body.String())
	}

	rec := call(t, h, http.MethodPost, "/api/v1/reports/run", manager.AccessToken, map[string]any{
		"source": "loans", "limit": 50,
		"items": []string{"id", "remainingDebt", "remainingInstallments", "installmentAmount", "feeAmount", "paymentCode", "installmentCode"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("گزارش وام باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var out contracts.ReportResult
	decodeInto(t, rec, &out)
	if len(out.Columns) != 7 || out.Total == 0 {
		t.Fatalf("گزارش وام باید ستون‌های صورت حساب را بدهد: %+v", out)
	}
	if len(out.Summary) == 0 {
		t.Fatal("گزارش وام باید جمع خلاصه داشته باشد")
	}
	foundRemaining := false
	for _, s := range out.Summary {
		if s.Key == "remainingDebt" && s.Total > 0 {
			foundRemaining = true
		}
	}
	if !foundRemaining {
		t.Fatalf("جمع «مانده بدهی» در خلاصهٔ گزارش نیست: %+v", out.Summary)
	}
	// ستون مانده بدهی برای وام نمونه باید پر باشد.
	remainingIndex := -1
	for i, c := range out.Columns {
		if c.Key == "remainingDebt" {
			remainingIndex = i
		}
	}
	if remainingIndex < 0 {
		t.Fatal("ستون مانده بدهی در گزارش نیست")
	}
	nonEmpty := false
	for _, row := range out.Rows {
		if strings.TrimSpace(row[remainingIndex]) != "" && row[remainingIndex] != "0" {
			nonEmpty = true
		}
	}
	if !nonEmpty {
		t.Fatalf("مانده بدهی هیچ سطری پر نشده است: %+v", out.Rows)
	}

	// ---------- منبع تازهٔ «اقساط وام‌ها» ----------
	srcRec := call(t, h, http.MethodGet, "/api/v1/reports/sources", manager.AccessToken, nil)
	if srcRec.Code != http.StatusOK {
		t.Fatalf("فهرست منابع گزارش باید ۲۰۰ بدهد، %d داد", srcRec.Code)
	}
	var sources struct {
		Sources []contracts.ReportSourceMeta `json:"sources"`
	}
	decodeInto(t, srcRec, &sources)
	found := false
	for _, s := range sources.Sources {
		if s.Key == "loanInstallments" {
			found = true
			if len(s.Fields) < 8 {
				t.Fatalf("منبع اقساط باید اقلام کامل داشته باشد: %+v", s.Fields)
			}
		}
	}
	if !found {
		t.Fatal("منبع گزارش «اقساط وام‌ها» ثبت نشده است")
	}
	instReport := call(t, h, http.MethodPost, "/api/v1/reports/run", manager.AccessToken, map[string]any{
		"source": "loanInstallments", "limit": 500,
		"items": []string{"loan", "seq", "period", "amount", "principal", "fee", "statusFa"},
	})
	if instReport.Code != http.StatusOK {
		t.Fatalf("گزارش اقساط باید ۲۰۰ بدهد، %d داد: %s", instReport.Code, instReport.Body.String())
	}
	var instOut contracts.ReportResult
	decodeInto(t, instReport, &instOut)
	if instOut.Total < 60 {
		t.Fatalf("گزارش اقساط باید دست‌کم ۶۰ سطر وام نمونه را بدهد، %d داد", instOut.Total)
	}
	// پیوند قسط به وام، ستون «شناسهٔ وام» را پر می‌کند.
	loanIndex := -1
	for i, c := range instOut.Columns {
		if c.Key == "loan" {
			loanIndex = i
		}
	}
	if loanIndex < 0 {
		t.Fatal("ستون شناسهٔ وام در گزارش اقساط نیست")
	}
	linked := 0
	for _, row := range instOut.Rows {
		if strings.TrimSpace(row[loanIndex]) != "" {
			linked++
		}
	}
	if linked < 60 {
		t.Fatalf("دست‌کم ۶۰ قسط باید به وام پیوند داشته باشند، %d داشت", linked)
	}
	if len(instOut.Summary) < 3 {
		t.Fatalf("گزارش اقساط باید جمع‌های خلاصه بدهد: %+v", instOut.Summary)
	}
}

// TestLOAN10ScreenRegistered می‌سنجد صفحهٔ «صورت حساب وام‌ها» در مانیفست ماژول وام ثبت
// شده و کامپوننت اپ موبایل را اعلام کرده است.
func TestLOAN10ScreenRegistered(t *testing.T) {
	deps, err := NewDeps(Config{
		JWTSecret: []byte("test-secret-key-for-loan-screen"),
		DataDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("ساخت وابستگی‌ها ناموفق: %v", err)
	}
	module, ok := deps.Registry.ByID("loans_retirement")
	if !ok {
		t.Fatal("ماژول وام و بازنشستگی در رجیستری نیست")
	}
	for _, screen := range module.Screens {
		if screen.Key != "loanStatement" {
			continue
		}
		if screen.Component != "LoanStatementScreen" {
			t.Fatalf("کامپوننت صفحهٔ صورت حساب باید LoanStatementScreen باشد: %s", screen.Component)
		}
		if screen.TitleFa == "" {
			t.Fatal("صفحهٔ صورت حساب باید عنوان فارسی داشته باشد")
		}
		return
	}
	t.Fatal("صفحهٔ loanStatement در مانیفست ماژول وام ثبت نشده است")
}
