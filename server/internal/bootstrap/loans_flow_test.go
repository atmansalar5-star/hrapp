// آزمون جریان سرتاسری «loans»: سیم‌کشی ماژول‌ها، سیاست‌ها و گردش‌کار.

package bootstrap

import (
	"net/http"
	"testing"

	"hrms/internal/contracts"
	loansdomain "hrms/internal/modules/loans_retirement/domain"
)

// TestLoansScoringAndReportFilters جریان وام را می‌سنجد: گزینه‌های وام، امتیازبندی
// تخصیص، گزارش وام با فیلتر نوع/بانک و شرط‌های دلخواه، و بخش بیمهٔ تکمیلی.
func TestLoansScoringAndReportFilters(t *testing.T) {
	h := newTestServer(t)
	token := login(t, h, "hr.manager", "demo").AccessToken

	// ---------- گزینه‌های وام: انواع، بانک‌های عامل و معیارهای امتیازبندی ----------
	optRec := call(t, h, http.MethodGet, "/api/v1/loans/options", token, nil)
	if optRec.Code != http.StatusOK {
		t.Fatalf("گزینه‌های وام باید ۲۰۰ بدهد، %d داد: %s", optRec.Code, optRec.Body.String())
	}
	var opts contracts.LoanOptions
	decodeInto(t, optRec, &opts)
	if len(opts.LoanTypes) != 7 {
		t.Fatalf("باید ۷ نوع وام اعلام شود، %d شد", len(opts.LoanTypes))
	}
	if len(opts.Banks) < 6 {
		t.Fatalf("بانک‌های عامل اعلام نشده‌اند (%d)", len(opts.Banks))
	}
	if len(opts.Criteria) < 10 {
		t.Fatalf("دست‌کم ۱۰ معیار امتیازبندی لازم است، %d شد", len(opts.Criteria))
	}
	if opts.TodayFa == "" {
		t.Fatal("تاریخ امروز سرور برای فرم وام خالی است")
	}

	// ---------- امتیازبندی و رتبه‌بندی تخصیص ----------
	alloc := call(t, h, http.MethodPost, "/api/v1/loans/allocation/preview", token, map[string]any{})
	if alloc.Code != http.StatusOK {
		t.Fatalf("امتیازبندی تخصیص باید ۲۰۰ بدهد، %d داد: %s", alloc.Code, alloc.Body.String())
	}
	var ranked contracts.LoanAllocationResult
	decodeInto(t, alloc, &ranked)
	if ranked.Count == 0 || len(ranked.Rows) != ranked.Count {
		t.Fatalf("رتبه‌بندی امتیاز خالی است: %+v", ranked)
	}
	for i, row := range ranked.Rows {
		if row.Rank != i+1 {
			t.Fatalf("رتبهٔ ردیف %d باید %d باشد، %d شد", i, i+1, row.Rank)
		}
		if i > 0 && ranked.Rows[i-1].Total < row.Total {
			t.Fatalf("ترتیب امتیاز نزولی نیست: %d قبل از %d", ranked.Rows[i-1].Total, row.Total)
		}
		if len(row.Breakdown) != len(opts.Criteria) {
			t.Fatalf("ریز امتیاز «%s» باید %d معیار داشته باشد، %d داشت", row.EmployeeID, len(opts.Criteria), len(row.Breakdown))
		}
	}

	// ---------- گزارش وام: فیلتر نوع وام ----------
	runLoansReport := func(body map[string]any) contracts.ReportResult {
		t.Helper()
		rec := call(t, h, http.MethodPost, "/api/v1/reports/run", token, body)
		if rec.Code != http.StatusOK {
			t.Fatalf("اجرای گزارش وام باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
		}
		var out contracts.ReportResult
		decodeInto(t, rec, &out)
		return out
	}
	items := []string{"loanTypeFa", "bankFa", "amount", "feeAmount", "guarantorName", "receivedDate"}
	// تعداد وام‌های نمونه از خود دامنه خوانده می‌شود؛ افزودن نمونهٔ تازه (مثلاً برای
	// پوشش مدیریت‌های بندر امام در داشبورد سهمیه) نباید این سنجش را بشکند، ولی
	// همچنان تضمین می‌کند هر وام نمونه در گزارش دیده شود.
	wantLoans := 0
	for _, in := range loansdomain.Seed() {
		if in.Kind == "loans" {
			wantLoans++
		}
	}
	all := runLoansReport(map[string]any{"source": "loans", "items": items, "limit": 200})
	if wantLoans == 0 {
		t.Fatal("دامنهٔ وام هیچ رکورد نمونه‌ای اعلام نکرده است")
	}
	if all.Total != wantLoans {
		t.Fatalf("همهٔ %d وام نمونه باید گزارش شوند، %d شد", wantLoans, all.Total)
	}
	housing := runLoansReport(map[string]any{
		"source": "loans", "items": items, "filters": map[string]string{"loanType": "housingDeposit"},
	})
	if housing.Total != 1 {
		t.Fatalf("فیلتر «ودیعهٔ مسکن» باید یک ردیف بدهد، %d داد", housing.Total)
	}
	bankFiltered := runLoansReport(map[string]any{
		"source": "loans", "items": items, "filters": map[string]string{"bank": "melli"},
	})
	if bankFiltered.Total != 2 {
		t.Fatalf("فیلتر بانک ملی باید دو ردیف (ودیعه + متمم) بدهد، %d داد", bankFiltered.Total)
	}

	// ---------- شرط دلخواه: «شامل» و «بزرگ‌تر یا مساوی» ----------
	onlySupplement := runLoansReport(map[string]any{
		"source": "loans", "items": items,
		"conditions": []map[string]string{{"field": "loanTypeFa", "op": "contains", "value": "متمم"}},
	})
	if onlySupplement.Total != 1 {
		t.Fatalf("شرط «شامل متمم» باید یک ردیف بدهد، %d داد", onlySupplement.Total)
	}
	big := runLoansReport(map[string]any{
		"source": "loans", "items": items,
		"conditions": []map[string]string{{"field": "amount", "op": "gte", "value": "300000000"}},
	})
	if big.Total != 2 {
		t.Fatalf("شرط «مبلغ ≥ ۳۰۰ میلیون» باید دو ردیف بدهد، %d داد", big.Total)
	}

	// ---------- شرط نامعتبر رد می‌شود ----------
	bad := call(t, h, http.MethodPost, "/api/v1/reports/run", token, map[string]any{
		"source": "loans", "items": items,
		"conditions": []map[string]string{{"field": "amount", "op": "wat", "value": "1"}},
	})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("شرط با عملگر ناشناخته باید ۴۰۰ بدهد، %d داد: %s", bad.Code, bad.Body.String())
	}

	// ---------- منبع دادهٔ گزارش، فیلترها و عملگرها را اعلام می‌کند ----------
	srcRec := call(t, h, http.MethodGet, "/api/v1/reports/sources", token, nil)
	if srcRec.Code != http.StatusOK {
		t.Fatalf("فهرست منابع گزارش باید ۲۰۰ بدهد، %d داد", srcRec.Code)
	}
	var sources struct {
		Sources   []contracts.ReportSourceMeta `json:"sources"`
		Operators []contracts.ReportOperator   `json:"operators"`
	}
	decodeInto(t, srcRec, &sources)
	var loansMeta *contracts.ReportSourceMeta
	for i := range sources.Sources {
		if sources.Sources[i].Key == "loans" {
			loansMeta = &sources.Sources[i]
		}
	}
	if loansMeta == nil {
		t.Fatal("منبع دادهٔ «وام کارکنان» در فهرست منابع گزارش نیست")
	}
	if len(loansMeta.Filters) < 3 {
		t.Fatalf("منبع وام باید فیلتر نوع/بانک/وضعیت اعلام کند، %d فیلتر داشت", len(loansMeta.Filters))
	}
	if len(sources.Operators) < 5 {
		t.Fatalf("عملگرهای شرط گزارش اعلام نشده‌اند (%d)", len(sources.Operators))
	}

	// ---------- بخش بیمهٔ تکمیلی در ماژول درمان ----------
	insRec := call(t, h, http.MethodGet, "/api/v1/medical/records?kind=supplementaryInsurance&size=50", token, nil)
	if insRec.Code != http.StatusOK {
		t.Fatalf("فهرست بیمهٔ تکمیلی باید ۲۰۰ بدهد، %d داد: %s", insRec.Code, insRec.Body.String())
	}
	var insPage contracts.RecordPage
	decodeInto(t, insRec, &insPage)
	if len(insPage.Items) == 0 {
		t.Fatal("رکوردهای نمونهٔ بیمهٔ تکمیلی یافت نشد")
	}
	for _, it := range insPage.Items {
		if it.Fields["policyNo"] == "" || it.Fields["insurer"] == "" {
			t.Fatalf("رکورد بیمهٔ تکمیلی باید شمارهٔ بیمه‌نامه و شرکت بیمه‌گر داشته باشد: %+v", it)
		}
	}

	// ---------- داشبورد تخصیص سهمیه: مدیریت‌ها، مشمولین/متقاضیان، مرحلهٔ مالی و نرخ کارمزد ----------
	if len(opts.ManagementUnits) != 8 {
		t.Fatalf("گزینه‌های وام باید ۸ مدیریت جدول سهمیه بدهد، %d داد", len(opts.ManagementUnits))
	}
	shareTotal := 0.0
	for _, u := range opts.ManagementUnits {
		shareTotal += u.SharePercent
		if u.Key == "" || u.TitleFa == "" || u.Eligible <= 0 {
			t.Fatalf("مدیریت جدول سهمیه ناقص است: %+v", u)
		}
	}
	if shareTotal != 100 {
		t.Fatalf("مجموع سهمیهٔ مدیریت‌ها باید ۱۰۰ باشد، %v شد", shareTotal)
	}
	foundMehr := false
	for _, b := range opts.Banks {
		if b.Key == "mehreghtesad" && b.TitleFa == "بانک مهر اقتصاد" {
			foundMehr = true
		}
	}
	if !foundMehr {
		t.Fatal("بانک مهر اقتصاد در فهرست بانک‌های عامل نیست")
	}

	quoRec := call(t, h, http.MethodGet, "/api/v1/loans/quota-dashboard", token, nil)
	if quoRec.Code != http.StatusOK {
		t.Fatalf("داشبورد سهمیه باید ۲۰۰ بدهد، %d داد: %s", quoRec.Code, quoRec.Body.String())
	}
	var quota contracts.LoanQuotaDashboard
	decodeInto(t, quoRec, &quota)
	if len(quota.Units) != 8 {
		t.Fatalf("داشبورد سهمیه باید ۸ مدیریت داشته باشد، %d داشت", len(quota.Units))
	}
	if quota.QuotaTotal != 20_000_000_000 {
		t.Fatalf("کل صندوق تسهیلات باید ۲۰ میلیارد ریال باشد، %v شد", quota.QuotaTotal)
	}
	covered := 0
	for _, u := range quota.Units {
		if u.QuotaAmount <= 0 || u.Eligible <= 0 {
			t.Fatalf("ردیف مدیریت ناقص است: %+v", u)
		}
		if u.RecordsCount > 0 {
			covered++
		}
		if u.CompanyCount+u.ThirdPartyCount != u.RecordsCount {
			t.Fatalf("ترکیب کارگزینی مدیریت «%s» با تعداد درخواست نمی‌خواند: %+v", u.UnitFa, u)
		}
	}
	if covered != 8 {
		t.Fatalf("همهٔ ۸ مدیریت باید درخواست داشته باشند، %d داشت", covered)
	}
	if total := quota.Pending + quota.Sent + quota.Rejected; total != wantLoans {
		t.Fatalf("جمع مراحل سهمیه (%d) با تعداد وام‌ها (%d) نمی‌خواند", total, wantLoans)
	}
	if quota.Applicants <= 0 || quota.Applicants > wantLoans {
		t.Fatalf("تعداد متقاضیان نامعتبر است: %d", quota.Applicants)
	}
	if quota.Sent == 0 || quota.SentAmount <= 0 {
		t.Fatalf("باید دست‌کم یک پروندهٔ ارسالی به مدیریت مالی باشد: %+v", quota)
	}
	if len(quota.Stages) != len(loansdomain.QuotaStages()) {
		t.Fatalf("داشبورد باید همهٔ مراحل ارسال را بدهد، %d داد", len(quota.Stages))
	}
	for _, st := range quota.Stages {
		if st.SentToFinance && st.StageKey == "rejected" {
			t.Fatalf("مرحلهٔ ردشده نباید ارسالی به مالی باشد: %+v", st)
		}
	}
	if len(quota.Groups) != 2 {
		t.Fatalf("تفکیک شرکتی/ارکان ثالث باید دو گروه باشد، %d شد", len(quota.Groups))
	}
	if len(quota.FeeTiers) == 0 {
		t.Fatal("طبقهٔ نرخ کارمزد تسهیلات خالی است")
	}
	for i := 1; i < len(quota.FeeTiers); i++ {
		if quota.FeeTiers[i-1].FeeRatePercent >= quota.FeeTiers[i].FeeRatePercent {
			t.Fatalf("طبقهٔ کارمزد باید صعودی باشد: %+v", quota.FeeTiers)
		}
	}
}
