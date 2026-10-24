// پیاده‌سازی ماژول «reports» — «sources».

package reports

import (
	"context"
	"fmt"

	"hrms/internal/contracts"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/repfilter"
)

// employeeReportSource منبع دادهٔ گزارش «کارکنان»؛ از پورت قراردادی کارکنان می‌خواند و
// هرگز به داخلی‌های ماژول کارکنان دست نمی‌زند.
type employeeReportSource struct{ deps modulekit.Deps }

func employeeFields() []contracts.ReportField {
	return []contracts.ReportField{
		{Key: "id", LabelFa: "کد پرسنلی", Type: "text"},
		{Key: "fullName", LabelFa: "نام و نام خانوادگی", Type: "text"},
		{Key: "nationalId", LabelFa: "کد ملی", Type: "text"},
		{Key: "departmentName", LabelFa: "واحد سازمانی", Type: "text"},
		{Key: "position", LabelFa: "پست سازمانی", Type: "text"},
		{Key: "contractType", LabelFa: "نوع همکاری", Type: "text"},
		{Key: "employmentTypeFa", LabelFa: "نوع استخدام", Type: "text"},
		{Key: "status", LabelFa: "وضعیت", Type: "text"},
		{Key: "hireDate", LabelFa: "تاریخ استخدام", Type: "date"},
		{Key: "baseSalary", LabelFa: "حقوق پایه", Type: "money"},
	}
}

// employeeFilters فیلترهای سادهٔ اعلامی منبع کارکنان. رابط وب از همین می‌سازد و
// کاربر می‌تواند علاوه بر آن روی هر قلم دلخواه شرط بگذارد.
//
// گزینه‌های «نوع استخدام» و «وضعیت» از کاتالوگ‌های مرجع (contracts) می‌آید و هیچ
// گزینه‌ای در رابط هاردکد نمی‌شود. هر دو چندانتخابی‌اند (Multiple) تا کاربر بتواند
// مثلادر گزارش امور کارکنان «کارگر رسمی» و «کارمند مدت معین» را با هم تیک بزند.
func employeeFilters() []contracts.ReportFilterDef {
	types := make([]contracts.ReportFilterOption, 0, len(contracts.EmploymentTypeCatalog))
	for _, t := range contracts.EmploymentTypes() {
		types = append(types, contracts.ReportFilterOption{Value: t.Key, LabelFa: t.TitleFa})
	}
	statuses := make([]contracts.ReportFilterOption, 0, len(contracts.EmployeeStatusCatalog))
	for _, s := range contracts.EmployeeStatuses() {
		statuses = append(statuses, contracts.ReportFilterOption{Value: s.Key, LabelFa: s.TitleFa})
	}
	return []contracts.ReportFilterDef{
		{Key: "employmentType", LabelFa: "نوع استخدام", Type: "select", Options: types, Multiple: true},
		{Key: "status", LabelFa: "وضعیت کارمند", Type: "select", Options: statuses, Multiple: true},
		{Key: "departmentId", LabelFa: "شناسهٔ واحد سازمانی", Type: "text"},
	}
}

// Metadata معرفی منبع کارکنان.
func (s employeeReportSource) Metadata() contracts.ReportSourceMeta {
	return contracts.ReportSourceMeta{
		Key: "employees", TitleFa: "کارکنان", Fields: employeeFields(), Filters: employeeFilters(),
	}
}

// Rows سطرهای گزارش کارکنان را بر پایهٔ اقلام انتخاب‌شده می‌سازد.
func (s employeeReportSource) Rows(ctx context.Context, req contracts.ReportRequest) (contracts.ReportResult, error) {
	reader, ok := s.deps.Ports.Employees()
	if !ok {
		return contracts.ReportResult{}, fmt.Errorf("دادهٔ کارکنان در این استقرار در دسترس نیست")
	}
	fields := employeeFields()
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
	// فیلترهای چندانتخابی رابط با کاما می‌آیند؛ این‌جا به فهرست تبدیل می‌شوند و
	// انبار کارکنان سطر را با «هر یک از» این مقادیر تطبیق می‌دهد (نه فقط یکی).
	page, err := reader.List(ctx, contracts.EmployeeFilter{
		Scope:           "all",
		Size:            limit,
		EmploymentTypes: contracts.SplitValues(req.Filters["employmentType"]),
		Statuses:        contracts.SplitValues(req.Filters["status"]),
		DepartmentID:    req.Filters["departmentId"],
	})
	if err != nil {
		return contracts.ReportResult{}, err
	}

	rows := make([][]string, 0, len(page.Items))
	var salaryTotal float64
	for _, e := range page.Items {
		values := map[string]string{
			"id":               e.ID,
			"fullName":         e.FullName,
			"nationalId":       e.NationalID,
			"departmentName":   e.DepartmentName,
			"position":         e.Position,
			"contractType":     e.ContractType,
			"employmentTypeFa": contracts.EmploymentTypeTitleFa(e.EmploymentType),
			"status":           e.Status,
			"hireDate":         e.HireDate,
			"baseSalary":       fmt.Sprintf("%d", e.BaseSalary),
		}
		// شرط‌های دلخواه کاربر روی هر قلم، پیش از افزودن سطر اعمال می‌شوند.
		if !repfilter.Matches(values, req.Conditions) {
			continue
		}
		salaryTotal += float64(e.BaseSalary)
		row := make([]string, 0, len(cols))
		for _, c := range cols {
			row = append(row, values[c.Key])
		}
		rows = append(rows, row)
	}

	return contracts.ReportResult{
		Source:  "employees",
		TitleFa: "کارکنان",
		Columns: cols,
		Rows:    rows,
		Total:   len(rows),
		NoteFa:  "داده از پورت کارکنان خوانده شده است؛ متناسب با دسترسی نقش فیلتر می‌شود.",
		Summary: []contracts.ReportSummary{{Key: "baseSalary", LabelFa: "جمع حقوق پایه", Total: salaryTotal}},
	}, nil
}
