// تعریف گزارش‌های ماژول «bonus» برای سامانهٔ گزارش‌ساز.

package bonus

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/bonus/domain"
	"hrms/internal/platform/recordkit"
	"hrms/internal/platform/repfilter"
)

// bonusReportSource منبع دادهٔ «پاداش کارکنان» برای گزارش‌ساز است.
//
// گزارش‌های مورد نیاز واحد پاداش از همین منبع ساخته می‌شوند: پرداخت به تفکیک مدیریت،
// نوع پاداش، دوره، روش پرداخت، شمارهٔ دفعه و شمارهٔ مصوبه؛ کاربر می‌تواند ستون‌های
// دلخواه را انتخاب کند، فیلتر بگذارد و روی هر قلم شرط بگذارد.
type bonusReportSource struct{ svc *recordkit.Service }

// reportBonusFields اقلام انتخاب‌شدنی گزارش پاداش.
func reportBonusFields() []contracts.ReportField {
	return []contracts.ReportField{
		{Key: "id", LabelFa: "شناسهٔ پرداخت", Type: "text"},
		{Key: "employeeName", LabelFa: "نام کارمند", Type: "text"},
		{Key: "managementFa", LabelFa: "مدیریت", Type: "text"},
		{Key: "bonusTypeFa", LabelFa: "نوع پاداش", Type: "text"},
		{Key: "amount", LabelFa: "مبلغ پرداختی", Type: "money"},
		{Key: "paymentDate", LabelFa: "تاریخ پرداخت", Type: "date"},
		{Key: "period", LabelFa: "دورهٔ پرداخت", Type: "text"},
		{Key: "sequence", LabelFa: "شمارهٔ دفعه", Type: "number"},
		{Key: "paymentMethodFa", LabelFa: "روش پرداخت", Type: "text"},
		{Key: "decreeNo", LabelFa: "شمارهٔ مصوبه / حکم", Type: "text"},
		{Key: "statusFa", LabelFa: "وضعیت", Type: "text"},
		{Key: "reference", LabelFa: "شرح / مرجع", Type: "text"},
	}
}

// reportBonusFilters فیلترهای اعلامی منبع پاداش: مدیریت، نوع پاداش، روش پرداخت، وضعیت.
func reportBonusFilters() []contracts.ReportFilterDef {
	managements := make([]contracts.ReportFilterOption, 0)
	for _, m := range contracts.Managements() {
		managements = append(managements, contracts.ReportFilterOption{Value: m.Key, LabelFa: m.TitleFa})
	}
	types := make([]contracts.ReportFilterOption, 0)
	for _, t := range domain.BonusTypes() {
		types = append(types, contracts.ReportFilterOption{Value: t.Key, LabelFa: t.TitleFa})
	}
	methods := make([]contracts.ReportFilterOption, 0)
	for _, m := range domain.PaymentMethods() {
		methods = append(methods, contracts.ReportFilterOption{Value: m.Key, LabelFa: m.TitleFa})
	}
	statuses := make([]contracts.ReportFilterOption, 0)
	for _, s := range domain.BonusStatuses() {
		statuses = append(statuses, contracts.ReportFilterOption{Value: s.Value, LabelFa: s.LabelFa})
	}
	// هر چهار فیلتر چندانتخابی‌اند (Multiple) تا کاربر بتواند چند مدیریت، چند نوع پاداش،
	// چند روش پرداخت یا چند وضعیت را با هم تیک بزند (مثلاً «پرداخت‌شده» و «در انتظار»).
	return []contracts.ReportFilterDef{
		{Key: "management", LabelFa: "مدیریت", Type: "select", Options: managements, Multiple: true},
		{Key: "bonusType", LabelFa: "نوع پاداش", Type: "select", Options: types, Multiple: true},
		{Key: "paymentMethod", LabelFa: "روش پرداخت", Type: "select", Options: methods, Multiple: true},
		{Key: "status", LabelFa: "وضعیت", Type: "select", Options: statuses, Multiple: true},
	}
}

// Metadata معرفی منبع پاداش.
func (s bonusReportSource) Metadata() contracts.ReportSourceMeta {
	return contracts.ReportSourceMeta{
		Key: "bonus", TitleFa: "پاداش کارکنان",
		Fields: reportBonusFields(), Filters: reportBonusFilters(),
	}
}

// Rows سطرهای گزارش پاداش را بر پایهٔ اقلام، فیلترها و شرط‌های انتخابی می‌سازد.
func (s bonusReportSource) Rows(ctx context.Context, req contracts.ReportRequest) (contracts.ReportResult, error) {
	if s.svc == nil {
		return contracts.ReportResult{}, fmt.Errorf("دادهٔ پاداش در این استقرار در دسترس نیست")
	}
	fields := reportBonusFields()
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
		Module: "bonus", Kind: "bonusPayments", Scope: "all", Page: 1, Size: limit,
	})
	if err != nil {
		return contracts.ReportResult{}, err
	}

	// فیلترهای چندانتخابی رابط با کاما می‌آیند؛ این‌جا به فهرست تبدیل می‌شوند و هر سطر با
	// «هر یک از» مقدارهای تیک‌خورده تطبیق داده می‌شود (انتخاب خالی یعنی «همه»).
	managementValues := contracts.SplitValues(req.Filters["management"])
	bonusTypeValues := contracts.SplitValues(req.Filters["bonusType"])
	paymentMethodValues := contracts.SplitValues(req.Filters["paymentMethod"])
	statusValues := contracts.SplitValues(req.Filters["status"])

	rows := make([][]string, 0, len(page.Items))
	var amountTotal, paidTotal float64
	paidCount := 0
	for _, rec := range page.Items {
		values := bonusValues(rec)
		if !contracts.MatchValue("", managementValues, rec.Fields["management"]) {
			continue
		}
		if !contracts.MatchValue("", bonusTypeValues, rec.Fields["bonusType"]) {
			continue
		}
		if !contracts.MatchValue("", paymentMethodValues, rec.Fields["paymentMethod"]) {
			continue
		}
		if !contracts.MatchValue("", statusValues, rec.Status) {
			continue
		}
		if !repfilter.Matches(values, req.Conditions) {
			continue
		}
		amountTotal += floatOf(values["amount"])
		if rec.Status == "paid" {
			paidTotal += floatOf(values["amount"])
			paidCount++
		}
		row := make([]string, 0, len(cols))
		for _, c := range cols {
			row = append(row, values[c.Key])
		}
		rows = append(rows, row)
	}

	return contracts.ReportResult{
		Source:  "bonus",
		TitleFa: "پاداش کارکنان",
		Columns: cols,
		Rows:    rows,
		Total:   len(rows),
		NoteFa:  "گزارش پاداش روی رکوردهای واقعی ماژول پاداش ساخته شده و با مدیریت، نوع پاداش، روش پرداخت، وضعیت و شرط‌های انتخابی فیلتر می‌شود.",
		Summary: []contracts.ReportSummary{
			{Key: "amount", LabelFa: "جمع مبالغ پاداش", Total: amountTotal},
			{Key: "paidAmount", LabelFa: "جمع مبالغ پرداخت‌شده", Total: paidTotal},
			{Key: "payments", LabelFa: "تعداد دفعات پرداخت‌شده", Total: float64(paidCount)},
		},
	}, nil
}

// numberDigits مقدار عددی یک فیلد را به رشتهٔ لاتین بدون جداکننده تبدیل می‌کند تا
// شرط‌های عددی گزارش (بزرگ‌تر، مساوی…) روی مقدار درست کار کنند.
func numberDigits(v string) string {
	n, ok := domain.AmountOf(v)
	if !ok {
		return ""
	}
	return strconv.FormatInt(n, 10)
}

// floatOf رشتهٔ عددی را به عدد اعشاری تبدیل می‌کند؛ نامعتبر یعنی صفر.
func floatOf(v string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	return f
}

// bonusValues مقادیر هر قلم را برای یک رکورد پرداخت پاداش می‌سازد.
func bonusValues(rec contracts.Record) map[string]string {
	return map[string]string{
		"id":              rec.ID,
		"employeeName":    rec.EmployeeName,
		"managementFa":    contracts.ManagementTitleFa(rec.Fields["management"]),
		"bonusTypeFa":     domain.BonusTypeTitleFa(rec.Fields["bonusType"]),
		"amount":          numberDigits(rec.Fields["amount"]),
		"paymentDate":     rec.Fields["paymentDate"],
		"period":          domain.PeriodOf(rec.Fields["paymentDate"]),
		"sequence":        numberDigits(rec.Fields["sequence"]),
		"paymentMethodFa": domain.PaymentMethodTitleFa(rec.Fields["paymentMethod"]),
		"decreeNo":        rec.Fields["decreeNo"],
		"statusFa":        rec.StatusFa,
		"reference":       rec.Fields["reference"],
	}
}
