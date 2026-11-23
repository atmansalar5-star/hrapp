// نگاشت رکوردهای وام به قواعد حساب و ساخت صورت حساب — «واسط دامنه».
//
// چرا این فایل: قاعده‌های مالی وام در بستهٔ `rules` است و شکل داده در رکوردهای ماژول. این
// واسط تنها جایی است که فیلدهای رکورد به ساختار قواعد و برگشت نگاشت می‌شوند، تا لایهٔ HTTP
// و ماژول حقوق هیچ نگاشتی از خودشان نداشته باشند و صورت حساب و فیش از یک منبع بخوانند.
package domain

import (
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/rules"
)

// StatusRepaying وضعیت «در حال بازپرداخت» وام؛ فقط وام‌های این وضعیت قسط ماهانه دارند.
const StatusRepaying = "repaying"

// StatusSettled وضعیت «تسویه‌شده».
const StatusSettled = "settled"

// LoanTermsOf شرایط وام را از فیلدهای رکورد می‌سازد (نگاشت مشترک در rules).
func LoanTermsOf(rec contracts.Record) rules.LoanTerms {
	return rules.LoanTermsFromFields(rec.ID, rec.Fields)
}

// LoanPaymentsOf پرداخت‌های ثبت‌شدهٔ یک وام: اقساط «پرداخت‌شده» و مبلغ تسویهٔ روی رکورد.
func LoanPaymentsOf(loan contracts.Record, installments []contracts.Record) []rules.LoanPayment {
	records := make([]rules.LoanRecordedPayment, 0, len(installments))
	for _, inst := range installments {
		if !LoanInstallmentBelongsTo(inst, loan.ID) {
			continue
		}
		records = append(records, rules.LoanRecordedPayment{
			Seq:        int(int64Field(inst.Fields["seq"])),
			PeriodFa:   inst.Fields["period"],
			PaidDateFa: strings.TrimSpace(inst.Fields["paidDate"]),
			Amount:     int64Field(inst.Fields["amount"]),
			MethodKey:  strings.TrimSpace(inst.Fields["method"]),
			RefFa:      inst.ID,
			Paid:       inst.Status == "paid",
		})
	}
	return rules.LoanPaymentsFromRecords(loan.Fields, records)
}

// linkedByID پیوند قسط به وام را در فیلد «loan» می‌سنجد؛ رکورد قسط قدیمی می‌تواند شناسهٔ
// وام را در یادداشت داشته باشد، پس وجود شناسه در یادداشت هم پذیرفته می‌شود.
func linkedByID(inst contracts.Record, loanID string) bool {
	if loanID == "" {
		return false
	}
	if strings.TrimSpace(inst.Fields["loan"]) == loanID {
		return true
	}
	return strings.Contains(inst.NoteFa, loanID)
}

// LoanInstallmentBelongsTo آیا این رکورد قسط به همین وام تعلق دارد (پیوند وام).
func LoanInstallmentBelongsTo(inst contracts.Record, loanID string) bool {
	return linkedByID(inst, loanID)
}

// LoanStatementOf صورت حساب کامل یک رکورد وام را می‌سازد.
func LoanStatementOf(loan contracts.Record, installments []contracts.Record, referencePeriodFa string) (rules.LoanStatement, []string) {
	return rules.BuildLoanStatement(rules.LoanStatementInput{
		LoanID:            loan.ID,
		LoanType:          strings.TrimSpace(loan.Fields["loanType"]),
		LoanTypeFa:        LoanTypeTitleFa(loan.Fields["loanType"]),
		TitleFa:           loan.Title,
		EmployeeID:        loan.EmployeeID,
		StatusFa:          loan.StatusFa,
		Terms:             LoanTermsOf(loan),
		SettledAmount:     int64Field(loan.Fields["settledAmount"]),
		SettleDateFa:      strings.TrimSpace(loan.Fields["settleDate"]),
		Payments:          LoanPaymentsOf(loan, installments),
		ReferencePeriodFa: referencePeriodFa,
	})
}

// LoanScheduleRows جدول اقساط یک رکورد وام (پیش‌نمایش/بازسازی).
func LoanScheduleRows(loan contracts.Record) (rules.LoanSchedule, []string) {
	return rules.BuildLoanSchedule(LoanTermsOf(loan))
}

// LoanInstallmentInput ورودی رکورد قسط برای یک ردیف جدول اقساط.
//
// قسط در انبار رکوردها یک سند واقعی است (نه عدد محاسباتی)، تا «کسر شد/نشد»، تاریخ کسر و
// روش پرداخت قابل پیگیری باشد و صورت حساب روی دادهٔ واقعی ساخته شود.
func LoanInstallmentInput(loan contracts.Record, row rules.LoanScheduleRow) contracts.RecordInput {
	dueDate := ""
	if row.PeriodFa != "" {
		dueDate = row.PeriodFa + "/01"
	}
	return contracts.RecordInput{
		Kind:       "loanInstallments",
		Title:      "قسط " + itoa(row.Seq) + " — " + loan.Title,
		EmployeeID: loan.EmployeeID,
		NoteFa:     "قسط " + itoa(row.Seq) + " وام " + loan.ID,
		Fields: map[string]string{
			"loan":      loan.ID,
			"seq":       itoa(row.Seq),
			"period":    row.PeriodFa,
			"dueDate":   dueDate,
			"amount":    itoa64(row.Amount),
			"principal": itoa64(row.Principal),
			"fee":       itoa64(row.Fee),
			// قسط وام از حقوق کسر می‌شود؛ روش پرداخت پیوند قسط به فیش است.
			"method": "salary",
		},
	}
}

// int64Field عدد صحیح را از فیلد متنی می‌خواند (ارقام فارسی و جداکننده پذیرفته می‌شود).
func int64Field(raw string) int64 {
	value := calendar.NormalizeDigits(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "،", "")
	if value == "" {
		return 0
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		f, ferr := strconv.ParseFloat(value, 64)
		if ferr != nil {
			return 0
		}
		return int64(f)
	}
	return n
}

// floatField عدد اعشاری را از فیلد متنی می‌خواند (نرخ کارمزد).
func floatField(raw string) float64 {
	value := calendar.NormalizeDigits(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "٪", "")
	value = strings.TrimSuffix(value, "%")
	if value == "" {
		return 0
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return f
}
