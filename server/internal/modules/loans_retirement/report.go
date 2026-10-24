// تعریف گزارش‌های ماژول «loans_retirement» برای سامانهٔ گزارش‌ساز.

package loans_retirement

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/loans_retirement/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/platform/repfilter"
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
		{Key: "supplementFa", LabelFa: "متمم وام", Type: "text"},
		{Key: "supplementOf", LabelFa: "متممِ وام (شناسهٔ وام پایه)", Type: "text"},
		{Key: "guarantorName", LabelFa: "نام ضامن", Type: "text"},
		{Key: "guarantorNationalId", LabelFa: "کد ملی ضامن", Type: "text"},
		{Key: "guarantorRelation", LabelFa: "نسبت ضامن", Type: "text"},
		{Key: "receivedDate", LabelFa: "زمان دریافت", Type: "date"},
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

	rows := make([][]string, 0, len(page.Items))
	var amountTotal, feeTotal float64
	// فیلترهای سادهٔ منبع (نوع وام، بانک عامل، وضعیت) چندانتخابی‌اند: کاربر می‌تواند
	// چند گزینه را تیک بزند و رابط آن‌ها را با کاما می‌فرستد.
	loanTypes := contracts.SplitValues(req.Filters["loanType"])
	banks := contracts.SplitValues(req.Filters["bank"])
	statuses := contracts.SplitValues(req.Filters["status"])
	for _, rec := range page.Items {
		values := loanValues(rec)
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
		},
	}, nil
}

// loanValues مقادیر هر قلم را برای یک رکورد وام می‌سازد.
func loanValues(rec contracts.Record) map[string]string {
	supplement := "خیر"
	if strings.TrimSpace(rec.Fields["supplementOf"]) != "" {
		supplement = "بله"
	}
	return map[string]string{
		"id":                  rec.ID,
		"employeeName":        rec.EmployeeName,
		"loanTypeFa":          domain.LoanTypeTitleFa(rec.Fields["loanType"]),
		"bankFa":              bankFa(rec.Fields["bank"]),
		"amount":              num(rec.Fields["amount"]),
		"installments":        num(rec.Fields["installments"]),
		"installmentAmount":   num(rec.Fields["installmentAmount"]),
		"feeRate":             num(rec.Fields["feeRate"]),
		"feeAmount":           num(rec.Fields["feeAmount"]),
		"supplementFa":        supplement,
		"supplementOf":        rec.Fields["supplementOf"],
		"guarantorName":       rec.Fields["guarantorName"],
		"guarantorNationalId": rec.Fields["guarantorNationalId"],
		"guarantorRelation":   rec.Fields["guarantorRelation"],
		"receivedDate":        rec.Fields["receivedDate"],
		"statusFa":            rec.StatusFa,
		"purpose":             rec.Fields["purpose"],
	}
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
