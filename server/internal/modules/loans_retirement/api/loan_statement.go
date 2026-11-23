// مسیرهای «صورت حساب و اقساط وام» — پرداخت یک‌باره، جدول اقساط ماهانه و مانده بدهی.
//
// چرا این فایل: ماژول وام تا پیش از این فقط درخواست، امتیازبندی و تأیید وام را می‌شناخت؛
// نه جدول اقساطی ساخته می‌شد و نه مانده بدهی جایی گزارش می‌شد. اکنون چهار کار انجام
// می‌شود و همهٔ حساب‌ها از بستهٔ `rules` می‌آید (یک منبع، تا صورت حساب و فیش یک عدد ببینند):
//
//	GET  /api/v1/loans/statements            کارتابل صورت حساب‌ها (فیلتر کارمند/وضعیت)
//	GET  /api/v1/loans/statement?id=...      صورت حساب کامل یک وام
//	POST /api/v1/loans/schedule/preview      پیش‌نمایش جدول اقساط (بدون ذخیره)
//	POST /api/v1/loans/{id}/schedule         ساخت/به‌روزرسانی رکوردهای قسط همان وام
//	GET  /api/v1/loans/installments-due      اقساط یک دوره برای کسر از حقوق
package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/loans_retirement/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/rules"
)

// loanStatementRequest ورودی پیش‌نمایش جدول اقساط.
type loanStatementRequest struct {
	LoanType       string  `json:"loanType,omitempty"`
	Amount         int64   `json:"amount"`
	Installments   int     `json:"installments,omitempty"`
	FeeRatePercent float64 `json:"feeRatePercent,omitempty"`
	FeeAmount      int64   `json:"feeAmount,omitempty"`
	ReceivedDateFa string  `json:"receivedDate,omitempty"`
	FirstPeriodFa  string  `json:"firstPeriod,omitempty"`
}

// Statements کارتابل صورت حساب وام‌ها: هر وام با شرایط، مانده بدهی و قسط بعدی.
func (h *Handler) Statements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	period := strings.TrimSpace(q.Get("period"))
	if period == "" {
		period = h.currentPeriodFa()
	}
	employeeID := strings.TrimSpace(q.Get("employeeId"))
	status := strings.TrimSpace(q.Get("status"))

	loans, err := h.listRecords(ctx, "loans", employeeID, status, 1000)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	installments, err := h.listRecords(ctx, "loanInstallments", employeeID, "", 3000)
	if err != nil {
		writeRuleError(w, err)
		return
	}

	page := contracts.LoanStatementPage{
		Items:    make([]contracts.LoanStatementView, 0, len(loans)),
		PeriodFa: period, TodayFa: h.currentDateFa(),
		CodeGaps: loanCodeGaps(),
		NoteFa: "هر ردیف صورت حساب کامل یک وام است: پرداخت یک‌باره، جدول اقساط، تاریخچه، " +
			"کارمزد و مانده بدهی. ستون‌های «مانده بدهی» و «اقساط باقیمانده» از همان جدول " +
			"اقساطِ ساخته‌شده در بستهٔ قواعد می‌آید.",
	}
	for _, loan := range loans {
		view := h.statementView(loan, installmentsOf(installments, loan.ID), period)
		page.Items = append(page.Items, view)
		page.Totals.Loans++
		page.Totals.Amount += view.Statement.Amount
		page.Totals.FeeAmount += view.Statement.FeeAmount
		page.Totals.TotalPayable += view.Statement.TotalPayable
		page.Totals.PaidAmount += view.Statement.Balance.PaidAmount
		page.Totals.RemainingDebt += view.Statement.Balance.RemainingDebt
		page.Totals.OverdueAmount += view.Statement.Balance.OverdueAmount
		if loan.Status == domain.StatusRepaying {
			page.Totals.Active++
			page.Totals.MonthlyDeduction += installmentOfPeriod(view.Statement, period)
		}
	}
	page.Count = len(page.Items)
	httpx.WriteJSON(w, http.StatusOK, page)
}

// Statement صورت حساب کامل یک وام.
func (h *Handler) Statement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = h.currentPeriodFa()
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		id = strings.TrimSpace(r.URL.Query().Get("loanId"))
	}
	if id == "" {
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeBadRequest,
			"شناسهٔ وام برای صورت حساب الزامی است.", nil)
		return
	}
	rec, found, err := h.svc.Get(ctx, contracts.RecordFilter{Module: "loans_retirement", Scope: "all"}, id)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	if !found {
		httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "وام با این شناسه یافت نشد.", nil)
		return
	}
	installments, err := h.listRecords(ctx, "loanInstallments", rec.EmployeeID, "", 3000)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, h.statementView(rec, installmentsOf(installments, rec.ID), period))
}

// SchedulePreview پیش‌نمایش جدول اقساط پیش از ثبت وام (هیچ رکوردی ساخته نمی‌شود).
func (h *Handler) SchedulePreview(w http.ResponseWriter, r *http.Request) {
	var req loanStatementRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	terms := rules.LoanTerms{
		LoanType:       strings.TrimSpace(req.LoanType),
		Amount:         req.Amount,
		FeeRatePercent: req.FeeRatePercent,
		FeeAmount:      req.FeeAmount,
		Installments:   req.Installments,
		ReceivedDateFa: strings.TrimSpace(req.ReceivedDateFa),
		FirstPeriodFa:  strings.TrimSpace(req.FirstPeriodFa),
	}
	schedule, problems := rules.BuildLoanSchedule(terms)
	payment, installment, _ := rules.LoanPayslipCodeFor(terms.LoanType)
	httpx.WriteJSON(w, http.StatusOK, contracts.LoanSchedulePreviewResult{
		Schedule:          schedule,
		PaymentCode:       payment.Code,
		PaymentCodeFa:     payment.TitleFa,
		InstallmentCode:   installment.Code,
		InstallmentCodeFa: installment.TitleFa,
		CodeNoteFa:        rules.LoanCodeNoteFa(terms.LoanType),
		Problems:          problems,
		NoteFa: "جدول اقساط از مبلغ وام، کارمزد و تعداد اقساط ساخته می‌شود؛ قسط آخر " +
			"باقیماندهٔ گردکردن را جبران می‌کند تا جمع اقساط با جمع قابل بازپرداخت برابر باشد.",
	})
}

// BuildSchedule رکوردهای قسط یک وام را از جدول اقساط می‌سازد یا به‌روز می‌کند.
//
// تکرارپذیر است: قسط‌های موجود (بر پایهٔ شناسهٔ وام و شمارهٔ قسط) دوباره ساخته نمی‌شوند،
// فقط ماه و مبلغشان با جدول تازه هم‌خوان می‌شود؛ قسطِ کسرشده دست نمی‌خورد.
func (h *Handler) BuildSchedule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeBadRequest,
			"شناسهٔ وام در نشانی نیامده است.", nil)
		return
	}
	rec, found, err := h.svc.Get(ctx, contracts.RecordFilter{Module: "loans_retirement", Scope: "all"}, id)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	if !found {
		httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "وام با این شناسه یافت نشد.", nil)
		return
	}
	schedule, problems := domain.LoanScheduleRows(rec)
	if len(schedule.Rows) == 0 {
		httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
			strings.Join(problems, " "), nil)
		return
	}

	existing, err := h.listRecords(ctx, "loanInstallments", rec.EmployeeID, "", 3000)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	bySeq := map[int]contracts.Record{}
	for _, inst := range existing {
		if !domain.LoanInstallmentBelongsTo(inst, rec.ID) {
			continue
		}
		bySeq[int(numOf(inst.Fields["seq"]))] = inst
	}

	actor := h.actor(ctx)
	result := contracts.LoanScheduleBuildResult{LoanID: rec.ID}
	for _, row := range schedule.Rows {
		in := domain.LoanInstallmentInput(rec, row)
		if inst, ok := bySeq[row.Seq]; ok {
			if inst.Fields["period"] == row.PeriodFa && inst.Fields["amount"] == in.Fields["amount"] && inst.Fields["principal"] == in.Fields["principal"] {
				result.Unchanged++
				continue
			}
			if inst.Status == "paid" {
				// قسط کسرشده سند مالی است و با بازسازی جدول عوض نمی‌شود.
				result.Unchanged++
				continue
			}
			if _, err := h.svc.Update(ctx, inst.ID, in, actor); err != nil {
				writeRuleError(w, err)
				return
			}
			result.Updated++
			continue
		}
		if _, err := h.svc.Create(ctx, in, actor); err != nil {
			writeRuleError(w, err)
			return
		}
		result.Created++
	}

	refreshed, _ := h.listRecords(ctx, "loanInstallments", rec.EmployeeID, "", 3000)
	result.Statement = h.statementView(rec, installmentsOf(refreshed, rec.ID), h.currentPeriodFa())
	result.NoteFa = "جدول اقساط هم‌خوان شد: " + itoa(result.Created) + " قسط تازه، " +
		itoa(result.Updated) + " به‌روزرسانی و " + itoa(result.Unchanged) + " قسط بی‌تغییر."
	if result.Created > 0 || result.Updated > 0 {
		h.deps.RecordAudit(r, "loans_retirement", "schedule", rec.ID,
			"ساخت جدول اقساط وام «"+rec.Title+"» با "+itoa(len(schedule.Rows))+" قسط")
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// DueInstallments اقساط یک دوره برای کسر از حقوق (صفحهٔ پردازش حقوق و گزارش).
func (h *Handler) DueInstallments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = h.currentPeriodFa()
	}
	employeeID := strings.TrimSpace(r.URL.Query().Get("employeeId"))
	installments, err := h.listRecords(ctx, "loanInstallments", employeeID, "", 3000)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	loans, err := h.listRecords(ctx, "loans", employeeID, "", 1000)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	loanByID := map[string]contracts.Record{}
	for _, loan := range loans {
		loanByID[loan.ID] = loan
	}
	balances := map[string]rules.LoanBalance{}
	for id, loan := range loanByID {
		insts := installmentsOf(installments, id)
		balances[id] = loanBalanceOf(loan, insts, period)
	}

	page := contracts.LoanInstallmentDuePage{PeriodFa: period, Rows: []contracts.LoanInstallmentDueRow{}}
	for _, inst := range installments {
		instPeriod := rules.LoanPeriod(inst.Fields["period"])
		if instPeriod != period {
			continue
		}
		loanID := strings.TrimSpace(inst.Fields["loan"])
		loan, ok := loanByID[loanID]
		if !ok {
			continue
		}
		if loan.Status != domain.StatusRepaying {
			continue
		}
		amount := int64(numOf(inst.Fields["amount"]))
		_, installmentCode, _ := rules.LoanPayslipCodeFor(loan.Fields["loanType"])
		row := contracts.LoanInstallmentDueRow{
			LoanID: loanID, LoanTypeFa: domain.LoanTypeTitleFa(loan.Fields["loanType"]),
			EmployeeID: inst.EmployeeID, EmployeeName: inst.EmployeeName,
			Seq: int(numOf(inst.Fields["seq"])), PeriodFa: instPeriod, Amount: amount,
			Status: inst.Status, StatusFa: inst.StatusFa,
			CodeFa: installmentCode.Code, CodeTitleFa: installmentCode.TitleFa,
			RemainingDebt: balances[loanID].RemainingDebt,
		}
		page.Rows = append(page.Rows, row)
		page.TotalAmount += amount
		if inst.Status == "paid" {
			page.PaidAmount += amount
		} else {
			page.PendingAmount += amount
		}
	}
	sort.SliceStable(page.Rows, func(i, j int) bool {
		if page.Rows[i].EmployeeID != page.Rows[j].EmployeeID {
			return page.Rows[i].EmployeeID < page.Rows[j].EmployeeID
		}
		return page.Rows[i].Seq < page.Rows[j].Seq
	})
	page.Count = len(page.Rows)
	page.NoteFa = "اقساط همین ماه از جدول اقساط وام‌های در حال بازپرداخت خوانده می‌شود؛ " +
		"فیش حقوقی همین مبلغ را به‌عنوان کسور وام کسر می‌کند."
	httpx.WriteJSON(w, http.StatusOK, page)
}

/* ----------------------------- کمک‌ها ----------------------------- */

// statementView صورت حساب یک رکورد وام با سرصفحهٔ فارسی‌اش.
func (h *Handler) statementView(loan contracts.Record, installments []contracts.Record, period string) contracts.LoanStatementView {
	statement, _ := domain.LoanStatementOf(loan, installments, period)
	return contracts.LoanStatementView{
		LoanID: loan.ID, TitleFa: loan.Title, EmployeeID: loan.EmployeeID,
		EmployeeName: loan.EmployeeName, Status: loan.Status, StatusFa: loan.StatusFa,
		BankFa:             h.bankTitle(context.Background(), loan.Fields["bank"]),
		ManagementFa:       h.managementTitle(context.Background(), loan.Fields["management"]),
		PurposeFa:          strings.TrimSpace(loan.Fields["purpose"]),
		GuarantorName:      strings.TrimSpace(loan.Fields["guarantorName"]),
		GuarantorRelation:  strings.TrimSpace(loan.Fields["guarantorRelation"]),
		InstallmentRecords: len(installments),
		Statement:          statement,
	}
}

// listRecords رکوردهای یک نوع از همین ماژول را می‌خواند.
func (h *Handler) listRecords(ctx context.Context, kind, employeeID, status string, size int) ([]contracts.Record, error) {
	page, err := h.svc.List(ctx, contracts.RecordFilter{
		Module: "loans_retirement", Kind: kind, EmployeeID: employeeID, Status: status,
		Scope: "all", Page: 1, Size: size,
	})
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// installmentsOf اقساط یک وام را جدا می‌کند.
func installmentsOf(installments []contracts.Record, loanID string) []contracts.Record {
	out := make([]contracts.Record, 0, 8)
	for _, inst := range installments {
		if domain.LoanInstallmentBelongsTo(inst, loanID) {
			out = append(out, inst)
		}
	}
	return out
}

// installmentOfPeriod مبلغ قسط همین ماه برای یک صورت حساب (صفر اگر قسطی در این ماه نباشد).
func installmentOfPeriod(statement rules.LoanStatement, period string) int64 {
	for _, row := range statement.Schedule {
		if row.PeriodFa == period {
			return row.Amount
		}
	}
	return 0
}

// loanCodeGaps انواع وامی که کد رسمی فیش ندارند.
func loanCodeGaps() []contracts.LoanCodeGap {
	types := domain.LoanTypes()
	keys := make([]string, 0, len(types))
	for _, t := range types {
		keys = append(keys, t.Key)
	}
	out := []contracts.LoanCodeGap{}
	for _, key := range rules.LoanCodeGaps(keys) {
		out = append(out, contracts.LoanCodeGap{Key: key, TitleFa: domain.LoanTypeTitleFa(key)})
	}
	return out
}

// currentPeriodFa ماه جاری شمسی (YYYY/MM).
func (h *Handler) currentPeriodFa() string {
	if today, err := calendar.FromGregorian(h.now()); err == nil {
		return monthOf(today.String())
	}
	return ""
}

// currentDateFa تاریخ جاری شمسی.
func (h *Handler) currentDateFa() string {
	if today, err := calendar.FromGregorian(h.now()); err == nil {
		return today.String()
	}
	return ""
}

// monthOf «YYYY/MM/DD» ⇒ «YYYY/MM».
func monthOf(dateFa string) string {
	if len(dateFa) >= 7 {
		return dateFa[:7]
	}
	return ""
}

// loanBalanceOf مانده بدهی یک وام از صورت حسابش.
func loanBalanceOf(loan contracts.Record, installments []contracts.Record, period string) rules.LoanBalance {
	statement, _ := domain.LoanStatementOf(loan, installments, period)
	return statement.Balance
}

// numOf عدد صحیح را از فیلد متنی رکورد می‌خواند (ارقام فارسی و جداکننده پذیرفته می‌شود).
func numOf(raw string) int64 {
	value := calendar.NormalizeDigits(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "،", "")
	if value == "" {
		return 0
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
