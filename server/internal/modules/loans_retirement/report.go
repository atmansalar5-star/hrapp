// تعریف گزارش‌های ماژول «loans_retirement» برای سامانهٔ گزارش‌ساز.

package loans_retirement

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/loans_retirement/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/platform/repfilter"
	"hrms/internal/rules"
)

// loansReportSource منبع دادهٔ «وام کارکنان» برای گزارش‌ساز است.
//
// گزارش‌های وام روی همین منبع ساخته می‌شوند: نوع وام (ودیعهٔ مسکن، متمم، بانکی،
// اضطراری، مددکاری، فوت بستگان درجه یک و حوادث مترقبه)، بانک عامل، کارمزد، میزان
// اقساط، اطلاعات ضامن و زمان دریافت. اقلام در همین‌جا اعلام می‌شوند تا کاربر بتواند
// ستون‌های دلخواه را انتخاب کند، فیلتر بگذارد و روی هر قلم شرط بگذارد.
type loansReportSource struct{ svc *recordkit.Service }

// reportLoanFields اقلام انتخاب‌شدنی گزارش وام.
func reportLoanFields() []contracts.ReportField {
	return []contracts.ReportField{
		{Key: "id", LabelFa: "شناسهٔ وام", Type: "text"},
		{Key: "employeeName", LabelFa: "نام کارمند", Type: "text"},
		{Key: "loanTypeFa", LabelFa: "نوع وام", Type: "text"},
		{Key: "bankFa", LabelFa: "بانک عامل", Type: "text"},
		{Key: "amount", LabelFa: "مبلغ وام", Type: "money"},
		{Key: "installments", LabelFa: "تعداد اقساط", Type: "number"},
		{Key: "installmentAmount", LabelFa: "میزان هر قسط", Type: "money"},
		{Key: "feeRate", LabelFa: "نرخ کارمزد (درصد)", Type: "number"},
		{Key: "feeAmount", LabelFa: "مبلغ کارمزد", Type: "money"},
		// اقلام صورت حساب: جمع قابل بازپرداخت، کسرشده و مانده بدهی از جدول اقساط می‌آید.
		{Key: "totalPayable", LabelFa: "جمع قابل بازپرداخت", Type: "money"},
		{Key: "paidAmount", LabelFa: "جمع اقساط کسرشده", Type: "money"},
		{Key: "paidInstallments", LabelFa: "اقساط کسرشده", Type: "number"},
		{Key: "remainingInstallments", LabelFa: "اقساط باقی‌مانده", Type: "number"},
		{Key: "remainingDebt", LabelFa: "مانده بدهی", Type: "money"},
		{Key: "overdueAmount", LabelFa: "اقساط معوق (مبلغ)", Type: "money"},
		{Key: "firstPeriod", LabelFa: "شروع کسر اقساط", Type: "text"},
		{Key: "lastPeriod", LabelFa: "پایان اقساط", Type: "text"},
		{Key: "paymentCode", LabelFa: "کد پرداخت وام", Type: "text"},
		{Key: "installmentCode", LabelFa: "کد کسر قسط", Type: "text"},
		{Key: "supplementFa", LabelFa: "متمم وام", Type: "text"},
		{Key: "supplementOf", LabelFa: "متممِ وام (شناسهٔ وام پایه)", Type: "text"},
		{Key: "guarantorName", LabelFa: "نام ضامن", Type: "text"},
		{Key: "guarantorNationalId", LabelFa: "کد ملی ضامن", Type: "text"},
		{Key: "guarantorRelation", LabelFa: "نسبت ضامن", Type: "text"},
		{Key: "receivedDate", LabelFa: "تاریخ پرداخت وام", Type: "date"},
		{Key: "statusFa", LabelFa: "وضعیت", Type: "text"},
		{Key: "purpose", LabelFa: "هدف وام", Type: "text"},
	}
}

// reportLoanFilters فیلترهای اعلامی منبع وام: نوع وام، بانک عامل و وضعیت.
func reportLoanFilters() []contracts.ReportFilterDef {
	types := make([]contracts.ReportFilterOption, 0)
	for _, t := range domain.LoanTypes() {
		types = append(types, contracts.ReportFilterOption{Value: t.Key, LabelFa: t.TitleFa})
	}
	banks := make([]contracts.ReportFilterOption, 0)
	for _, b := range domain.Banks() {
		banks = append(banks, contracts.ReportFilterOption{Value: b, LabelFa: domain.BankTitleFa(b)})
	}
	statuses := make([]contracts.ReportFilterOption, 0)
	if kind, ok := domain.Descriptor().KindByKey("loans"); ok {
		for _, s := range kind.Statuses {
			statuses = append(statuses, contracts.ReportFilterOption{Value: s.Value, LabelFa: s.LabelFa})
		}
	}
	return []contracts.ReportFilterDef{
		{Key: "loanType", LabelFa: "نوع وام", Type: "select", Options: types, Multiple: true},
		{Key: "bank", LabelFa: "بانک عامل", Type: "select", Options: banks, Multiple: true},
		{Key: "status", LabelFa: "وضعیت وام", Type: "select", Options: statuses, Multiple: true},
	}
}

// Metadata معرفی منبع وام.
func (s loansReportSource) Metadata() contracts.ReportSourceMeta {
	return contracts.ReportSourceMeta{
		Key: "loans", TitleFa: "وام کارکنان", Fields: reportLoanFields(), Filters: reportLoanFilters(),
	}
}

// Rows سطرهای گزارش وام را بر پایهٔ اقلام، فیلترها و شرط‌های انتخابی می‌سازد.
func (s loansReportSource) Rows(ctx context.Context, req contracts.ReportRequest) (contracts.ReportResult, error) {
	if s.svc == nil {
		return contracts.ReportResult{}, fmt.Errorf("دادهٔ وام در این استقرار در دسترس نیست")
	}
	fields := reportLoanFields()
	byKey := map[string]contracts.ReportField{}
	for _, f := range fields {
		byKey[f.Key] = f
	}
	cols := fields
	if len(req.Items) > 0 {
		cols = cols[:0]
		for _, key := range req.Items {
			if f, ok := byKey[key]; ok {
				cols = append(cols, f)
			}
		}
		if len(cols) == 0 {
			cols = fields
		}
	}

	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	page, err := s.svc.List(ctx, contracts.RecordFilter{
		Module: "loans_retirement", Kind: "loans", Scope: "all", Page: 1, Size: limit,
	})
	if err != nil {
		return contracts.ReportResult{}, err
	}
	// اقساط برای محاسبهٔ مانده بدهی و «کسرشده/باقی‌مانده» یک بار خوانده می‌شود؛ گزارش
	// ستون‌های صورت حساب را از همان جدول اقساط می‌دهد، نه از عدد دستی.
	installments, err := s.svc.List(ctx, contracts.RecordFilter{
		Module: "loans_retirement", Kind: "loanInstallments", Scope: "all", Page: 1, Size: 5000,
	})
	if err != nil {
		installments = contracts.RecordPage{}
	}
	byLoan := map[string][]contracts.Record{}
	for _, inst := range installments.Items {
		id := strings.TrimSpace(inst.Fields["loan"])
		if id == "" {
			continue
		}
		byLoan[id] = append(byLoan[id], inst)
	}
	period := reportPeriodFa()

	rows := make([][]string, 0, len(page.Items))
	var amountTotal, feeTotal, remainingTotal float64
	// فیلترهای سادهٔ منبع (نوع وام، بانک عامل، وضعیت) چندانتخابی‌اند: کاربر می‌تواند
	// چند گزینه را تیک بزند و رابط آن‌ها را با کاما می‌فرستد.
	loanTypes := contracts.SplitValues(req.Filters["loanType"])
	banks := contracts.SplitValues(req.Filters["bank"])
	statuses := contracts.SplitValues(req.Filters["status"])
	for _, rec := range page.Items {
		statement, _ := domain.LoanStatementOf(rec, byLoan[rec.ID], period)
		values := loanValues(rec, statement)
		if !contracts.MatchValue("", loanTypes, rec.Fields["loanType"]) {
			continue
		}
		if !contracts.MatchValue("", banks, rec.Fields["bank"]) {
			continue
		}
		if !contracts.MatchValue("", statuses, rec.Status) {
			continue
		}
		// شرط‌های دلخواه کاربر روی هر قلم؛ همه باید برقرار باشند.
		if !repfilter.Matches(values, req.Conditions) {
			continue
		}
		amountTotal += floatOf(values["amount"])
		feeTotal += floatOf(values["feeAmount"])
		remainingTotal += floatOf(values["remainingDebt"])
		row := make([]string, 0, len(cols))
		for _, c := range cols {
			row = append(row, values[c.Key])
		}
		rows = append(rows, row)
	}

	return contracts.ReportResult{
		Source:  "loans",
		TitleFa: "وام کارکنان",
		Columns: cols,
		Rows:    rows,
		Total:   len(rows),
		NoteFa:  "گزارش وام‌ها روی رکوردهای واقعی ماژول وام و بازنشستگی ساخته شده و با نوع، بانک عامل و شرط‌های انتخابی فیلتر می‌شود.",
		Summary: []contracts.ReportSummary{
			{Key: "amount", LabelFa: "جمع مبلغ وام‌ها", Total: amountTotal},
			{Key: "feeAmount", LabelFa: "جمع کارمزد وام‌ها", Total: feeTotal},
			{Key: "remainingDebt", LabelFa: "جمع مانده بدهی وام‌ها", Total: remainingTotal},
		},
	}, nil
}

// loanValues مقادیر هر قلم را برای یک رکورد وام می‌سازد. اقلام صورت حساب (جمع قابل
// بازپرداخت، کسرشده، مانده بدهی، کدها) از صورت حساب همان وام می‌آید تا گزارش و صورت حساب
// و فیش یک عدد بدهند.
func loanValues(rec contracts.Record, statement rules.LoanStatement) map[string]string {
	supplement := "خیر"
	if strings.TrimSpace(rec.Fields["supplementOf"]) != "" {
		supplement = "بله"
	}
	return map[string]string{
		"id":                    rec.ID,
		"employeeName":          rec.EmployeeName,
		"loanTypeFa":            domain.LoanTypeTitleFa(rec.Fields["loanType"]),
		"bankFa":                bankFa(rec.Fields["bank"]),
		"amount":                num(rec.Fields["amount"]),
		"installments":          num(rec.Fields["installments"]),
		"installmentAmount":     num(rec.Fields["installmentAmount"]),
		"feeRate":               num(rec.Fields["feeRate"]),
		"feeAmount":             num(rec.Fields["feeAmount"]),
		"totalPayable":          itoa64r(statement.TotalPayable),
		"paidAmount":            itoa64r(statement.Balance.PaidAmount),
		"paidInstallments":      itoa64r(int64(statement.Balance.PaidCount)),
		"remainingInstallments": itoa64r(int64(statement.Balance.RemainingInstallments)),
		"remainingDebt":         itoa64r(statement.Balance.RemainingDebt),
		"overdueAmount":         itoa64r(statement.Balance.OverdueAmount),
		"firstPeriod":           statement.FirstPeriodFa,
		"lastPeriod":            statement.LastPeriodFa,
		"paymentCode":           statement.PaymentCode,
		"installmentCode":       statement.InstallmentCode,
		"supplementFa":          supplement,
		"supplementOf":          rec.Fields["supplementOf"],
		"guarantorName":         rec.Fields["guarantorName"],
		"guarantorNationalId":   rec.Fields["guarantorNationalId"],
		"guarantorRelation":     rec.Fields["guarantorRelation"],
		"receivedDate":          rec.Fields["receivedDate"],
		"statusFa":              rec.StatusFa,
		"purpose":               rec.Fields["purpose"],
	}
}

// itoa64r عدد را برای ستون گزارش به رشته تبدیل می‌کند (صفر ⇒ رشتهٔ خالی تا ستون خالی بماند).
func itoa64r(v int64) string {
	if v == 0 {
		return "0"
	}
	return strconv.FormatInt(v, 10)
}

// reportPeriodFa ماه جاری شمسی برای مرجع «معوق» در گزارش.
func reportPeriodFa() string {
	today, err := calendar.FromGregorian(time.Now())
	if err != nil {
		return ""
	}
	text := today.String()
	if len(text) < 7 {
		return ""
	}
	return text[:7]
}

func bankFa(key string) string {
	if strings.TrimSpace(key) == "" {
		return ""
	}
	return domain.BankTitleFa(key)
}

// num ارقام فارسی و جداکننده‌ها را به رشتهٔ عددی لاتین تبدیل می‌کند تا گزارش و
// شرط‌های عددی درست کار کنند.
func num(v string) string {
	v = calendar.NormalizeDigits(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	return v
}

func floatOf(v string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	return f
}

/* --------------------------- منبع گزارش اقساط --------------------------- */

// loanInstallmentsReportSource منبع «اقساط وام‌ها» برای گزارش‌ساز: هر قسط یک سطر با ماه
// کسر، سهم اصل و کارمزد، وضعیت کسر و کد رسمی؛ مبنای پیگیری ماه‌به‌ماه کسور وام.
type loanInstallmentsReportSource struct{ svc *recordkit.Service }

// reportInstallmentFields اقلام انتخاب‌شدنی گزارش اقساط.
func reportInstallmentFields() []contracts.ReportField {
	return []contracts.ReportField{
		{Key: "id", LabelFa: "شناسهٔ قسط", Type: "text"},
		{Key: "loan", LabelFa: "شناسهٔ وام", Type: "text"},
		{Key: "employeeName", LabelFa: "نام کارمند", Type: "text"},
		{Key: "seq", LabelFa: "شمارهٔ قسط", Type: "number"},
		{Key: "period", LabelFa: "ماه کسر", Type: "text"},
		{Key: "dueDate", LabelFa: "سررسید", Type: "date"},
		{Key: "amount", LabelFa: "مبلغ قسط", Type: "money"},
		{Key: "principal", LabelFa: "سهم اصل", Type: "money"},
		{Key: "fee", LabelFa: "سهم کارمزد", Type: "money"},
		{Key: "statusFa", LabelFa: "وضعیت قسط", Type: "text"},
		{Key: "paidDate", LabelFa: "تاریخ کسر", Type: "date"},
		{Key: "methodFa", LabelFa: "روش پرداخت", Type: "text"},
	}
}

// Metadata معرفی منبع اقساط.
func (s loanInstallmentsReportSource) Metadata() contracts.ReportSourceMeta {
	statuses := []contracts.ReportFilterOption{}
	if kind, ok := domain.Descriptor().KindByKey("loanInstallments"); ok {
		for _, st := range kind.Statuses {
			statuses = append(statuses, contracts.ReportFilterOption{Value: st.Value, LabelFa: st.LabelFa})
		}
	}
	return contracts.ReportSourceMeta{
		Key: "loanInstallments", TitleFa: "اقساط وام‌ها", Fields: reportInstallmentFields(),
		Filters: []contracts.ReportFilterDef{
			{Key: "status", LabelFa: "وضعیت قسط", Type: "select", Options: statuses, Multiple: true},
		},
	}
}

// Rows سطرهای گزارش اقساط.
func (s loanInstallmentsReportSource) Rows(ctx context.Context, req contracts.ReportRequest) (contracts.ReportResult, error) {
	if s.svc == nil {
		return contracts.ReportResult{}, fmt.Errorf("دادهٔ اقساط وام در این استقرار در دسترس نیست")
	}
	fields := reportInstallmentFields()
	byKey := map[string]contracts.ReportField{}
	for _, f := range fields {
		byKey[f.Key] = f
	}
	cols := fields
	if len(req.Items) > 0 {
		cols = cols[:0]
		for _, key := range req.Items {
			if f, ok := byKey[key]; ok {
				cols = append(cols, f)
			}
		}
		if len(cols) == 0 {
			cols = fields
		}
	}
	limit := req.Limit
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}
	page, err := s.svc.List(ctx, contracts.RecordFilter{
		Module: "loans_retirement", Kind: "loanInstallments", Scope: "all", Page: 1, Size: limit,
	})
	if err != nil {
		return contracts.ReportResult{}, err
	}
	statuses := contracts.SplitValues(req.Filters["status"])
	rows := make([][]string, 0, len(page.Items))
	var total, pending, paid float64
	for _, rec := range page.Items {
		if !contracts.MatchValue("", statuses, rec.Status) {
			continue
		}
		values := map[string]string{
			"id":           rec.ID,
			"loan":         rec.Fields["loan"],
			"employeeName": rec.EmployeeName,
			"seq":          num(rec.Fields["seq"]),
			"period":       rec.Fields["period"],
			"dueDate":      rec.Fields["dueDate"],
			"amount":       num(rec.Fields["amount"]),
			"principal":    num(rec.Fields["principal"]),
			"fee":          num(rec.Fields["fee"]),
			"statusFa":     rec.StatusFa,
			"paidDate":     rec.Fields["paidDate"],
			"methodFa":     rules.LoanPaymentMethodFa(rec.Fields["method"]),
		}
		if !repfilter.Matches(values, req.Conditions) {
			continue
		}
		amount := floatOf(values["amount"])
		total += amount
		if rec.Status == "paid" {
			paid += amount
		} else {
			pending += amount
		}
		row := make([]string, 0, len(cols))
		for _, c := range cols {
			row = append(row, values[c.Key])
		}
		rows = append(rows, row)
	}
	return contracts.ReportResult{
		Source:  "loanInstallments",
		TitleFa: "اقساط وام‌ها",
		Columns: cols,
		Rows:    rows,
		Total:   len(rows),
		NoteFa:  "هر سطر یک قسط وام است؛ مبلغ قسط، سهم اصل و کارمزد از جدول اقساط همان وام می‌آید.",
		Summary: []contracts.ReportSummary{
			{Key: "amount", LabelFa: "جمع اقساط", Total: total},
			{Key: "paid", LabelFa: "جمع اقساط کسرشده", Total: paid},
			{Key: "pending", LabelFa: "جمع اقساط در انتظار", Total: pending},
		},
	}, nil
}
