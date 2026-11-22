// دسته‌بندی کارکنان بر پایهٔ نوع استخدام، گروه نظام پرداخت و روش پرداخت.
//
// چرا این گزارش: با جدا شدن «کارمند رسمی نفت» از گروه یکم جدول ۳ و افزودن روش پرداخت به
// قرارداد و حکم، باید بتوان یک‌نگاه دید «هر نفر در کدام دستهٔ پرداخت است» و شمار هر دسته
// چند نفر است. دسته از قراردادهای جاری خوانده می‌شود (سندی که تعهد پرداخت را ثبت کرده)
// و خروجی، هم خلاصهٔ شمارشی و هم فهرست کارکنان هر دسته را می‌دهد.
package api

import (
	"net/http"
	"sort"

	"hrms/internal/contracts"
	"hrms/internal/modules/contracts/domain"
	"hrms/internal/platform/httpx"
)

// payClassificationRow یک دستهٔ پرداخت با شمار کارکنانش.
type payClassificationRow struct {
	EmploymentType   string `json:"employmentType"`
	EmploymentTypeFa string `json:"employmentTypeFa"`
	PayFamily        string `json:"payFamily"`
	PayFamilyFa      string `json:"payFamilyFa"`
	PayGroup         int    `json:"payGroup"`
	PayGroupFa       string `json:"payGroupFa"`
	PayModel         string `json:"payModel"`
	PayModelFa       string `json:"payModelFa"`
	Count            int    `json:"count"`
	// Employees فهرست کارکنان همین دسته (شناسه و نام) برای بازبینی سریع.
	Employees []payClassificationEmployee `json:"employees"`
}

// payClassificationEmployee یک کارمند در دستهٔ پرداخت با شناسهٔ قراردادش.
type payClassificationEmployee struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	ContractID   string `json:"contractId"`
}

// PayClassification گزارش «دستهٔ پرداخت» کارکنان: نوع استخدام × گروه جدول ۳ × روش پرداخت.
func (h *Handler) PayClassification(w http.ResponseWriter, r *http.Request) {
	filter := h.filterFrom(r)
	filter.Status = "active"
	filter.Scope = "all"
	filter.Size = 500
	filter.Page = 1

	page, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.fail(w, err)
		return
	}

	index := map[string]*payClassificationRow{}
	for _, c := range page.Items {
		classification := domain.PayClassificationOf(c.EmploymentType, c.PayGroup, c.PayModel)
		key := c.EmploymentType + "|" + classification.PayModel + "|" +
			strconvItoa(classification.PayGroup)
		row, ok := index[key]
		if !ok {
			row = &payClassificationRow{
				EmploymentType:   c.EmploymentType,
				EmploymentTypeFa: c.EmploymentTypeFa,
				PayFamily:        domain.PayFamilyOf(c.EmploymentType),
				PayFamilyFa:      domain.PayFamilyTitleFa(domain.PayFamilyOf(c.EmploymentType)),
				PayGroup:         classification.PayGroup,
				PayGroupFa:       classification.PayGroupFa,
				PayModel:         classification.PayModel,
				PayModelFa:       classification.PayModelFa,
				Employees:        []payClassificationEmployee{},
			}
			index[key] = row
		}
		row.Count++
		row.Employees = append(row.Employees, payClassificationEmployee{
			EmployeeID:   c.EmployeeID,
			EmployeeName: c.EmployeeName,
			ContractID:   c.ID,
		})
	}

	rows := make([]payClassificationRow, 0, len(index))
	for _, row := range index {
		sort.Slice(row.Employees, func(i, j int) bool { return row.Employees[i].EmployeeID < row.Employees[j].EmployeeID })
		rows = append(rows, *row)
	}
	// ترتیب: نخست روش پرداخت (payroll، oil، …)، سپس شمارهٔ گروه، سپس نوع استخدام.
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].PayModel != rows[j].PayModel {
			return rows[i].PayModel < rows[j].PayModel
		}
		if rows[i].PayGroup != rows[j].PayGroup {
			return rows[i].PayGroup < rows[j].PayGroup
		}
		return rows[i].EmploymentType < rows[j].EmploymentType
	})

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"rows":       rows,
		"total":      page.Total,
		"groupCount": len(rows),
		"noteFa": "دسته‌ها از قراردادهای جاری خوانده می‌شوند: نوع استخدام × گروه جدول ۳ " +
			"آیین‌نامهٔ فوق‌العاده‌ها × روش پرداخت مصوب (payroll | oil | comp34006 | fourp).",
		"models": domain.PayModelOptions(),
		"groups": domain.PayGroupOptions(),
	})
}

// strconvItoa تبدیل کوچک عدد به رشته (بدون افزودن وابستگی در این فایل).
func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

var _ = contracts.ContractFilter{}
