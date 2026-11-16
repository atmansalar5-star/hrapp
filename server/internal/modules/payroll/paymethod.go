// مسیرهای «روش پرداخت کارکنان» — کنترل‌پنلی که برای هر کارمند تعیین می‌کند پرداختش با
// کدام مدل محاسبه شود.
//
// مرز مدل‌ها: این مسیر فقط انتخاب را ثبت و نمایش می‌دهد؛ هیچ محاسبه‌ای را عوض نمی‌کند.
// مدل حقوق و دستمزد جاری و مدل صندوق نفت مستقل و جاری می‌مانند و مدل ۴پی، مدل توسعه‌ای
// و اختیاری است (به همین دلیل انتخاب آن پیش از فعال‌بودن ماژول و پارامتر «اعمال در
// پرداخت» رد می‌شود و پس از آن هم فقط یادآوری می‌گیرد).
package payroll

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/modules/payroll/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// compensationModuleID شناسهٔ ماژول مدل توسعه‌ای ۴پی (فقط برای بررسی فعال‌بودن).
const compensationModuleID = "compensation"

// payMethodRequest ورودی ثبت روش پرداخت یک کارمند.
type payMethodRequest struct {
	EmployeeID    string `json:"employeeId"`
	PayModel      string `json:"payModel"`
	EffectiveDate string `json:"effectiveDate"`
	ReasonFa      string `json:"reasonFa"`
}

// payMethodRow یک سطر کنترل‌پنل: کارمند و روش پرداخت جاری او.
type payMethodRow struct {
	EmployeeID     string `json:"employeeId"`
	FullName       string `json:"fullName"`
	DepartmentName string `json:"departmentName"`
	PayModel       string `json:"payModel"`
	PayModelFa     string `json:"payModelFa"`
	RecordID       string `json:"recordId,omitempty"`
	EffectiveDate  string `json:"effectiveDate,omitempty"`
	Status         string `json:"status,omitempty"`
	StatusFa       string `json:"statusFa,omitempty"`
	// SourceFa منبع انتخاب: «ثبت‌شده» یا «پیش‌فرض سازمان».
	SourceFa string `json:"sourceFa"`
	NoteFa   string `json:"noteFa,omitempty"`
}

// registerPayMethodEndpoints مسیرهای کنترل‌پنل روش پرداخت را ثبت می‌کند.
func registerPayMethodEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "write", fn) }

	// فهرست روش‌های مصوب + وضعیت مدل ۴پی در این استقرار.
	mux.HandleFunc("GET /api/v1/payroll/pay-method-options", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, payMethodOptions(deps))
	}))

	// فهرست کارکنان با روش پرداخت جاری‌شان.
	mux.HandleFunc("GET /api/v1/payroll/pay-methods", read(func(w http.ResponseWriter, r *http.Request) {
		rows, err := payMethodRows(r.Context(), deps, svc, strings.TrimSpace(r.URL.Query().Get("query")))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		out := payMethodOptions(deps)
		out["items"] = rows
		out["count"] = len(rows)
		httpx.WriteJSON(w, http.StatusOK, out)
	}))

	// ثبت یا تغییر روش پرداخت یک کارمند؛ رکورد جاری پیشین منسوخ می‌شود.
	mux.HandleFunc("POST /api/v1/payroll/pay-methods", write(func(w http.ResponseWriter, r *http.Request) {
		var in payMethodRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "ورودی روش پرداخت خوانده نشد.", nil)
			return
		}
		employeeID := strings.TrimSpace(in.EmployeeID)
		if employeeID == "" {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, "کارمند را انتخاب کنید.", nil)
			return
		}
		payModel := strings.TrimSpace(in.PayModel)
		known := false
		for _, choice := range domain.PayModelChoices() {
			if choice.Key == payModel {
				known = true
				break
			}
		}
		if !known {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"روش پرداخت را از فهرست مصوب انتخاب کنید.", nil)
			return
		}
		if payModel == domain.PayModelFourP && !fourPEnabled(deps) {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"مدل ۴پی در این استقرار فعال نیست؛ نخست ماژول «نظام جبران خدمات ۴پی» را فعال کنید.", nil)
			return
		}
		effective := strings.TrimSpace(in.EffectiveDate)
		if effective == "" {
			effective = payTodayFa(deps)
		}
		if _, err := calendar.Parse(calendar.NormalizeDigits(effective)); err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed,
				"تاریخ اجرای روش پرداخت معتبر نیست (قالب YYYY/MM/DD).", nil)
			return
		}
		noteFa := ""
		current, found := activePayModelRecord(r.Context(), svc, employeeID)
		if found && current.Fields["payModel"] == payModel {
			rows, err := payMethodRows(r.Context(), deps, svc, "")
			if err != nil {
				httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
				return
			}
			out := payMethodOptions(deps)
			out["items"] = rows
			out["noteFa"] = "روش پرداخت این کارمند از پیش همین بود؛ تغییری ثبت نشد."
			httpx.WriteJSON(w, http.StatusOK, out)
			return
		}
		actor := actorOf(r)
		if found {
			if _, err := svc.Act(r.Context(), current.ID, contracts.RecordAction{
				Action: "supersede",
				NoteFa: "تغییر روش پرداخت از «" + domain.PayModelTitleFa(current.Fields["payModel"]) + "» به «" +
					domain.PayModelTitleFa(payModel) + "»",
			}, actor); err != nil {
				httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
				return
			}
		}
		created, err := svc.Create(r.Context(), contracts.RecordInput{
			Kind:       domain.KindEmployeePayModel,
			Title:      "روش پرداخت — " + employeeLabel(r.Context(), deps, employeeID),
			EmployeeID: employeeID,
			Fields: map[string]string{
				"payModel":      payModel,
				"effectiveDate": calendar.NormalizeDigits(effective),
				"reasonFa":      strings.TrimSpace(in.ReasonFa),
			},
			NoteFa: "ثبت از کنترل‌پنل روش پرداخت کارکنان",
		}, actor)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		// روش تازه باید «جاری» شود تا در محاسبه و کنترل‌پنل دیده شود.
		if _, err := svc.Act(r.Context(), created.ID, contracts.RecordAction{
			Action: "activate", NoteFa: "جاری‌سازی روش پرداخت از کنترل‌پنل",
		}, actor); err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeValidationFailed, err.Error(), nil)
			return
		}
		if payModel == domain.PayModelFourP && !fourPApplied(deps) {
			noteFa = "روش ۴پی انتخاب شد، ولی پارامتر «اعمال در پرداخت» مدل ۴پی در تنظیمات شرکت خاموش است؛ " +
				"تا روشن‌شدن آن، محاسبهٔ حقوق با مدل پیشین انجام می‌شود."
		}
		if payModel == domain.PayModelComp34006 && !comp34006Applied(deps) {
			noteFa = "روش «دستورالعمل ۳۴۰۰۶» انتخاب شد، ولی پارامتر «اعمال در پرداخت» این مدل در " +
				"تنظیمات شرکت خاموش است؛ تا روشن‌شدن آن، محاسبهٔ این کارمند پیشنهادی است و فیش ماهانه " +
				"با مدل پیشین محاسبه می‌شود."
		}
		rows, err := payMethodRows(r.Context(), deps, svc, "")
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		out := payMethodOptions(deps)
		out["items"] = rows
		out["recordId"] = created.ID
		if noteFa != "" {
			out["noteFa"] = noteFa
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))
}

// payMethodOptions فهرست روش‌های مصوب و وضعیت مدل ۴پی.
func payMethodOptions(deps modulekit.Deps) map[string]any {
	return map[string]any{
		"methods":          domain.PayModelChoices(),
		"grades":           nil,
		"fourPEnabled":     fourPEnabled(deps),
		"fourPApplied":     fourPApplied(deps),
		"comp34006Applied": comp34006Applied(deps),
		"today":            payTodayFa(deps),
		"independenceNoteFa": "انتخاب روش پرداخت، محاسبهٔ موجود را عوض نمی‌کند: مدل حقوق و دستمزد جاری، " +
			"مدل صندوق نفت و مدل دستورالعمل ۳۴۰۰۶ مستقل و جدا از هم‌اند و مدل ۴پی توسعه‌ای و " +
			"اختیاری است. این انتخاب تعیین می‌کند کارشناس کدام فرم و کدام موتور محاسبه را برای این " +
			"کارمند به‌کار ببرد.",
	}
}

// comp34006Applied آیا پارامتر شرکت، اعمال مدل دستورالعمل ۳۴۰۰۶ در پرداخت را روشن کرده است.
//
// چرا جدا از fourPApplied: هر مدل پارامتر خاموشی خودش را دارد تا روشن‌کردن یکی، دیگری را
// ناخواسته فعال نکند.
func comp34006Applied(deps modulekit.Deps) bool {
	for _, p := range deps.Org.Params {
		if p.Key != "comp34006.payroll.apply" {
			continue
		}
		v := strings.TrimSpace(p.Value)
		return v == "1" || strings.EqualFold(v, "yes") || strings.EqualFold(v, "true")
	}
	return false
}

// payTodayFa تاریخ امروز به شمسی (YYYY/MM/DD) بر پایهٔ ساعت سامانه.
func payTodayFa(deps modulekit.Deps) string {
	if deps.Clock == nil {
		return calendar.Format(time.Now())
	}
	return calendar.Format(deps.Clock())
}

// fourPEnabled آیا ماژول مدل ۴پی در این استقرار فعال است.
func fourPEnabled(deps modulekit.Deps) bool {
	return deps.Registry != nil && deps.Registry.IsEnabled(compensationModuleID)
}

// fourPApplied آیا پارامتر شرکت، اعمال مدل ۴پی در پرداخت را روشن کرده است.
func fourPApplied(deps modulekit.Deps) bool {
	for _, p := range deps.Org.Params {
		if p.Key != "compensation.payroll.apply" {
			continue
		}
		v := strings.TrimSpace(p.Value)
		return v == "1" || strings.EqualFold(v, "yes") || strings.EqualFold(v, "true")
	}
	return false
}

// activePayModelRecord رکورد جاری روش پرداخت یک کارمند.
func activePayModelRecord(ctx context.Context, svc *recordkit.Service, employeeID string) (contracts.Record, bool) {
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "payroll", Kind: domain.KindEmployeePayModel, EmployeeID: employeeID, Scope: "all", Size: 50,
	})
	if err != nil || len(page.Items) == 0 {
		return contracts.Record{}, false
	}
	var best contracts.Record
	found := false
	for _, rec := range page.Items {
		if rec.Status != "active" {
			continue
		}
		if !found || rec.Fields["effectiveDate"] > best.Fields["effectiveDate"] {
			best, found = rec, true
		}
	}
	return best, found
}

// payMethodRows کارکنان را با روش پرداخت جاری‌شان ترکیب می‌کند؛ کارمند بدون رکورد،
// «پیش‌فرض سازمان» (مدل حقوق و دستمزد جاری) نشان می‌دهد.
func payMethodRows(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, query string) ([]payMethodRow, error) {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return nil, nil
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 300, Query: query})
	if err != nil {
		return nil, err
	}
	records, _ := svc.List(ctx, contracts.RecordFilter{
		Module: "payroll", Kind: domain.KindEmployeePayModel, Scope: "all", Size: 500,
	})
	current := map[string]contracts.Record{}
	for _, rec := range records.Items {
		if rec.Status != "active" || rec.EmployeeID == "" {
			continue
		}
		if previous, ok := current[rec.EmployeeID]; !ok || rec.Fields["effectiveDate"] > previous.Fields["effectiveDate"] {
			current[rec.EmployeeID] = rec
		}
	}
	rows := make([]payMethodRow, 0, len(page.Items))
	for _, emp := range page.Items {
		row := payMethodRow{
			EmployeeID: emp.ID, FullName: emp.FullName, DepartmentName: emp.DepartmentName,
			PayModel: domain.PayModelPayroll, PayModelFa: domain.PayModelTitleFa(domain.PayModelPayroll),
			SourceFa: "پیش‌فرض سازمان",
		}
		if rec, ok := current[emp.ID]; ok {
			row.RecordID = rec.ID
			row.PayModel = domain.PayModelOf(rec.Fields["payModel"])
			row.PayModelFa = domain.PayModelTitleFa(row.PayModel)
			row.EffectiveDate = rec.Fields["effectiveDate"]
			row.Status = rec.Status
			row.StatusFa = rec.StatusFa
			row.SourceFa = "ثبت‌شده"
			row.NoteFa = rec.Fields["reasonFa"]
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].EmployeeID < rows[j].EmployeeID })
	return rows, nil
}

// employeeLabel نام نمایشی کارمند برای عنوان رکورد.
func employeeLabel(ctx context.Context, deps modulekit.Deps, employeeID string) string {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return employeeID
	}
	emp, found, err := reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
	if err != nil || !found {
		return employeeID
	}
	return emp.FullName
}
