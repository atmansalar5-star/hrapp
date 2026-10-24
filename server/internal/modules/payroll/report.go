// تعریف گزارش‌های ماژول «payroll» برای سامانهٔ گزارش‌ساز.

package payroll

import (
	"context"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/payroll/domain"
	"hrms/internal/platform/recordkit"
)

// payrollReportSource منبع دادهٔ «اقلام حقوق» برای گزارش‌ساز است. اقلام موجود در صفحهٔ
// اجزای حقوق همین‌جا به‌عنوان ستون‌های انتخاب‌شدنی اعلام می‌شوند تا کاربر بتواند بر پایهٔ
// اقلام دلخواه گزارش بگیرد.
type payrollReportSource struct{ svc *recordkit.Service }

// reportElementFields اقلام انتخاب‌شدنی گزارش اقلام حقوق.
func reportElementFields() []contracts.ReportField {
	return []contracts.ReportField{
		{Key: "code", LabelFa: "کد قلم", Type: "text"},
		{Key: "title", LabelFa: "عنوان قلم", Type: "text"},
		{Key: "categoryFa", LabelFa: "نوع", Type: "text"},
		{Key: "calcFa", LabelFa: "روش محاسبه", Type: "text"},
		{Key: "rate", LabelFa: "مقدار / ضریب", Type: "number"},
		{Key: "basisFa", LabelFa: "مبنا", Type: "text"},
		{Key: "formula", LabelFa: "فرمول", Type: "text"},
		{Key: "taxable", LabelFa: "مشمول مالیات", Type: "text"},
		{Key: "insured", LabelFa: "مشمول بیمه", Type: "text"},
		{Key: "statusFa", LabelFa: "وضعیت", Type: "text"},
	}
}

// Metadata معرفی منبع اقلام حقوق.
func (s payrollReportSource) Metadata() contracts.ReportSourceMeta {
	return contracts.ReportSourceMeta{
		Key:     "payroll_elements",
		TitleFa: "اقلام حقوق و دستمزد",
		Fields:  reportElementFields(),
	}
}

// Rows سطرهای گزارش را بر پایهٔ اقلام انتخاب‌شده می‌سازد.
func (s payrollReportSource) Rows(ctx context.Context, req contracts.ReportRequest) (contracts.ReportResult, error) {
	fields := reportElementFields()
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

	records := activeElements(ctx, s.svc)
	limit := req.Limit
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	rows := make([][]string, 0, len(records))
	rateSum := 0.0
	for _, rec := range records {
		if limit > 0 && len(rows) >= limit {
			break
		}
		values := map[string]string{
			"code":       rec.Fields["code"],
			"title":      rec.Title,
			"categoryFa": categoryFa(rec.Fields["category"]),
			"calcFa":     calcFa(rec.Fields["calc"]),
			"rate":       rec.Fields["rate"],
			"basisFa":    basisFa(rec.Fields["basis"]),
			"formula":    rec.Fields["formula"],
			"taxable":    rec.Fields["taxable"],
			"insured":    rec.Fields["insured"],
			"statusFa":   rec.StatusFa,
		}
		if v, ok := floatOf(rec.Fields["rate"]); ok {
			rateSum += v
		}
		row := make([]string, 0, len(cols))
		for _, c := range cols {
			row = append(row, values[c.Key])
		}
		rows = append(rows, row)
	}

	return contracts.ReportResult{
		Source:  "payroll_elements",
		TitleFa: "اقلام حقوق و دستمزد",
		Columns: cols,
		Rows:    rows,
		Total:   len(rows),
		NoteFa:  "مقادیر از اقلام پارامتریک حقوق خوانده شده و در سرور مرجع است.",
		Summary: []contracts.ReportSummary{{Key: "rate", LabelFa: "جمع ضرایب", Total: rateSum}},
	}, nil
}

func categoryFa(v string) string {
	switch strings.TrimSpace(v) {
	case domain.CategoryEarning, "مزایا":
		return "مزایا"
	case domain.CategoryDeduction, "کسور":
		return "کسور"
	default:
		return v
	}
}

func calcFa(v string) string {
	switch strings.TrimSpace(v) {
	case domain.CalcFixed:
		return "مبلغ ثابت"
	case domain.CalcPercent:
		return "درصدی از مبنا"
	case domain.CalcFormula:
		return "فرمول محاسبه"
	default:
		return v
	}
}
