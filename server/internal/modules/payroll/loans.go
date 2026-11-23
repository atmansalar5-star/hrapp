// اقساط وام در فیش حقوقی — بلوک `loans` و کسر واقعی از خالص.
//
// چرا این فایل: قاعدهٔ وام در ماژول وام و بازنشستگی است و فیش در ماژول حقوق ساخته می‌شود.
// ماژول حقوق رکوردهای وام را از پورت فقط‌خواندنی همان ماژول می‌خواند (بدون import بین دو
// ماژول) و حساب را از بستهٔ مشترک `rules` می‌گیرد؛ پس مبلغ قسط و مانده بدهی در فیش و در
// صورت حساب وام همیشه یکی است.
//
// جمع اقساط همین دوره به «سایر کسور» موتور فیش داده می‌شود؛ یعنی مبلغ از خالص کم می‌شود و
// در همان حال سطرهای وام (کد رسمی، مبلغ قسط، مانده بدهی) کنار فیش دیده و چاپ می‌شوند.
package payroll

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/rules"
)

// loansModuleID شناسهٔ ماژول مالک رکوردهای وام (پورت فقط‌خواندنی).
const loansModuleID = "loans_retirement"

// loanPayslipLine یک وام در فیش: شرایط، قسط همین دوره و مانده بدهی.
type loanPayslipLine struct {
	LoanID            string            `json:"loanId"`
	TitleFa           string            `json:"titleFa"`
	LoanTypeFa        string            `json:"loanTypeFa,omitempty"`
	StatusFa          string            `json:"statusFa,omitempty"`
	Amount            int64             `json:"amount"`
	FeeAmount         int64             `json:"feeAmount"`
	TotalPayable      int64             `json:"totalPayable"`
	Installments      int               `json:"installments"`
	InstallmentAmount int64             `json:"installmentAmount"`
	Seq               int               `json:"seq,omitempty"`
	PeriodFa          string            `json:"periodFa,omitempty"`
	DueAmount         int64             `json:"dueAmount"`
	Deducted          bool              `json:"deducted"`
	Paid              bool              `json:"paid"`
	StatusFa2         string            `json:"periodStatusFa,omitempty"`
	CodeFa            string            `json:"codeFa,omitempty"`
	CodeTitleFa       string            `json:"codeTitleFa,omitempty"`
	PaymentCodeFa     string            `json:"paymentCodeFa,omitempty"`
	Balance           rules.LoanBalance `json:"balance"`
	NoteFa            string            `json:"noteFa,omitempty"`
}

// loanPayslipDisbursement پرداخت یک‌بارهٔ وام که در همین ماه انجام شده است.
type loanPayslipDisbursement struct {
	LoanID      string `json:"loanId"`
	TitleFa     string `json:"titleFa"`
	Amount      int64  `json:"amount"`
	DateFa      string `json:"dateFa,omitempty"`
	CodeFa      string `json:"codeFa,omitempty"`
	CodeTitleFa string `json:"codeTitleFa,omitempty"`
}

// loanPayslipTotals جمع‌های بلوک وام در فیش.
type loanPayslipTotals struct {
	Loans          int   `json:"loans"`
	DueAmount      int64 `json:"dueAmount"`
	Deducted       int64 `json:"deducted"`
	PreviouslyPaid int64 `json:"previouslyPaid"`
	RemainingDebt  int64 `json:"remainingDebt"`
	Disbursed      int64 `json:"disbursed"`
}

// loanPayslipBlock بلوک اقساط وام برای پاسخ فیش و جمع کسر همین دوره.
func loanPayslipBlock(ctx context.Context, deps modulekit.Deps, employeeID, period string) (map[string]any, int64) {
	empty := map[string]any{
		"lines": []loanPayslipLine{}, "disbursements": []loanPayslipDisbursement{},
		"totals": loanPayslipTotals{}, "period": period,
		"noteFa": "هیچ وام در حال بازپرداختی برای این کارمند ثبت نشده است.",
	}
	port, ok := deps.Ports.RecordList(loansModuleID)
	if !ok || strings.TrimSpace(employeeID) == "" {
		return empty, 0
	}
	loans, err := port.List(ctx, contracts.RecordFilter{
		Module: loansModuleID, Kind: "loans", EmployeeID: employeeID, Scope: "all", Page: 1, Size: 200,
	})
	if err != nil {
		return empty, 0
	}
	installments, err := port.List(ctx, contracts.RecordFilter{
		Module: loansModuleID, Kind: "loanInstallments", EmployeeID: employeeID, Scope: "all", Page: 1, Size: 2000,
	})
	if err != nil {
		installments = contracts.RecordPage{}
	}

	lines := []loanPayslipLine{}
	disbursements := []loanPayslipDisbursement{}
	totals := loanPayslipTotals{}
	var deduction int64

	for _, loan := range loans.Items {
		// فقط وام در حال بازپرداخت قسط ماهانه دارد؛ وام تأییدنشده یا تسویه‌شده کسر ندارد.
		if loan.Status != "repaying" {
			continue
		}
		terms := rules.LoanTermsFromFields(loan.ID, loan.Fields)
		schedule, problems := rules.BuildLoanSchedule(terms)
		payments := rules.LoanPaymentsFromRecords(loan.Fields, loanRecordedPayments(installments.Items, loan.ID))
		balance := rules.LoanBalanceFor(schedule, payments, period)
		paymentCode, installmentCode, _ := rules.LoanPayslipCodeFor(terms.LoanType)

		// پرداخت یک‌بارهٔ وام در ماه پرداخت: یک سطر بستانکار با کد رسمی همان نوع وام.
		if disbursed := rules.LoanPeriod(terms.ReceivedDateFa); disbursed != "" && disbursed == period {
			disbursements = append(disbursements, loanPayslipDisbursement{
				LoanID: loan.ID, TitleFa: loan.Title, Amount: terms.Amount,
				DateFa: terms.ReceivedDateFa, CodeFa: paymentCode.Code, CodeTitleFa: paymentCode.TitleFa,
			})
			totals.Disbursed += terms.Amount
		}

		line := loanPayslipLine{
			LoanID: loan.ID, TitleFa: loan.Title,
			LoanTypeFa: rules.LoanTypeTitleFa(loan.Fields["loanType"]),
			StatusFa:   loan.StatusFa,
			Amount:     terms.Amount, FeeAmount: terms.FeeAmount,
			TotalPayable: schedule.TotalPayable, Installments: terms.Installments,
			InstallmentAmount: schedule.InstallmentAmount,
			CodeFa:            installmentCode.Code, CodeTitleFa: installmentCode.TitleFa,
			PaymentCodeFa: paymentCode.Code, Balance: balance,
		}
		paidSeq := map[int]bool{}
		for _, p := range payments {
			if p.Kind == rules.LoanEventInstallment {
				paidSeq[p.Seq] = true
			}
		}
		for _, row := range schedule.Rows {
			if row.PeriodFa != period {
				continue
			}
			line.Seq = row.Seq
			line.PeriodFa = row.PeriodFa
			line.DueAmount = row.Amount
			line.Paid = paidSeq[row.Seq]
			line.Deducted = !line.Paid
			if line.Paid {
				line.StatusFa2 = "قبلاً کسر شده است"
				totals.PreviouslyPaid += row.Amount
			} else {
				line.StatusFa2 = "در همین فیش کسر می‌شود"
				deduction += row.Amount
				totals.Deducted += row.Amount
			}
			break
		}
		if line.Seq == 0 {
			line.NoteFa = "قسطی از این وام در ماه " + period + " سررسید نشده است."
		}
		if len(problems) > 0 {
			line.NoteFa = strings.Join(problems, " ")
		}
		lines = append(lines, line)
		totals.Loans++
		totals.DueAmount += line.DueAmount
		totals.RemainingDebt += balance.RemainingDebt
	}

	sort.SliceStable(lines, func(i, j int) bool { return lines[i].LoanID < lines[j].LoanID })
	return map[string]any{
		"lines": lines, "disbursements": disbursements, "totals": totals, "period": period,
		"noteFa": "قسط هر وام در حال بازپرداخت برای ماه " + period + " از جدول اقساط همان وام " +
			"خوانده می‌شود و جمع آن به سایر کسور فیش افزوده میشود؛ مانده بدهی از همان جدول حساب شده است.",
	}, deduction
}

// loanRecordedPayments اقساط یک وام را به ساختار خنثی قواعد نگاشت می‌کند.
func loanRecordedPayments(installments []contracts.Record, loanID string) []rules.LoanRecordedPayment {
	out := make([]rules.LoanRecordedPayment, 0, len(installments))
	for _, inst := range installments {
		if strings.TrimSpace(inst.Fields["loan"]) != loanID {
			continue
		}
		out = append(out, rules.LoanRecordedPayment{
			Seq:        int(loanNumber(inst.Fields["seq"])),
			PeriodFa:   inst.Fields["period"],
			PaidDateFa: inst.Fields["paidDate"],
			Amount:     loanNumber(inst.Fields["amount"]),
			MethodKey:  inst.Fields["method"],
			RefFa:      inst.ID,
			Paid:       inst.Status == "paid",
		})
	}
	return out
}

// loanNumber عدد صحیح متنی (با ارقام فارسی/جداکننده) را می‌خواند.
func loanNumber(raw string) int64 {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "،", "")
	var out int64
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			out = out*10 + int64(r-'0')
		case r >= '۰' && r <= '۹':
			out = out*10 + int64(r-'۰')
		case r >= '٠' && r <= '٩':
			out = out*10 + int64(r-'٠')
		default:
			return out
		}
	}
	return out
}

// registerLoanEndpoints مسیر اقساط وام برای یک کارمند و یک دوره.
func registerLoanEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }
	mux.HandleFunc("GET /api/v1/payroll/loan-deductions", read(func(w http.ResponseWriter, r *http.Request) {
		employeeID := strings.TrimSpace(r.URL.Query().Get("employeeId"))
		if employeeID == "" {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeBadRequest,
				"انتخاب کارمند برای اقساط وام الزامی است.", nil)
			return
		}
		period := strings.TrimSpace(r.URL.Query().Get("period"))
		if period == "" {
			period = oilPeriod(deps)
		}
		block, deduction := loanPayslipBlock(r.Context(), deps, employeeID, period)
		block["deduction"] = deduction
		httpx.WriteJSON(w, http.StatusOK, block)
	}))
}
