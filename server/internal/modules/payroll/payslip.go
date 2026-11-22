// مسیر یکپارچهٔ «فیش بر پایهٔ روش پرداخت» — یک نقطهٔ ورود، چهار موتور.
//
// چرا این مسیر: نظام‌های پرداخت هر یک موتور خودشان را دارند (صندوق نفت، دستورالعمل ۳۴۰۰۶،
// حقوق و دستمزد جاری، و مدل توسعه‌ای ۴پی). پیش از این، هر موتور مسیر جدا داشت و مصرف‌کننده
// باید خودش می‌دانست «این کارمند با کدام روش پرداخت می‌شود». اکنون روش پرداختِ هر نفر روی
// سندش (قرارداد/حکم) و در پروندهٔ روش پرداخت ثبت است و همین مسیر، موتور درست را برمی‌گزیند:
//
//	oil        ⇒ rules.ComputeOilPayslip        (جدول‌های ۱ تا ۳ آیین‌نامه و گروه جدول ۳)
//	comp34006  ⇒ rules.Comp34006PayslipFor      (اقلام کددار ۳۴۶۰۱ تا ۳۴۶۱۸)
//	payroll    ⇒ rules.ComputePayroll           (حقوق و دستمزد جاری سازمان)
//	fourp      ⇒ همان موتور جاری + یادداشت مدل توسعه‌ای (کاربرگ و تصمیمش در ماژول جبران خدمات)
//
// قاعده در سرور اعمال می‌شود: روش پرداخت از درخواست گرفته نمی‌شود (مگر برای شبیه‌سازی با
// اجازهٔ صریح و سازگاری سنجیده‌شده)، بلکه از پروندهٔ روش پرداخت کارمند و در نبود آن از نوع
// استخدام استنتاج می‌گردد.
package payroll

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// PayslipRequest ورودی مشترک فیش یکپارچه.
type PayslipRequest struct {
	EmployeeID string `json:"employeeId"`
	// PayModel فقط برای شبیه‌سازی/آزمون: خالی یعنی از پرونده و نوع استخدام تعیین شود.
	// اگر مقدار بدهید، سازگاری‌اش با نوع استخدام سنجیده می‌شود.
	PayModel string `json:"payModel,omitempty"`
	// PeriodFa دورهٔ فیش به شمسی (YYYY/MM)؛ خالی یعنی ماه جاری.
	PeriodFa string `json:"periodFa,omitempty"`
	// OtherDeductions سایر کسور ماه (مساعده، اقساط) در موتورهایی که آن را می‌پذیرند.
	OtherDeductions int64 `json:"otherDeductions,omitempty"`
	// ورودی اختصاصی هر موتور (اختیاری): هر فیلدی که نیاید، از پروندهٔ کارمند و پارامترهای
	// شرکت پر می‌شود. این ساختار عمدی است تا فراخوان ساده فقط «کارمند» بدهد و کار کند.
	Oil       *rules.OilPayInput           `json:"oil,omitempty"`
	Comp34006 *rules.Comp34006PayslipInput `json:"comp34006,omitempty"`
	Payroll   *rules.PayrollInput          `json:"payroll,omitempty"`
}

// registerPayslipEndpoints مسیرهای فیش یکپارچه را ثبت می‌کند.
func registerPayslipEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }

	// فهرست کارکنان با روش پرداختشان + فهرست موتورها (برای فرم انتخاب کارمند).
	mux.HandleFunc("GET /api/v1/payroll/payslip-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, payslipDefaults(r.Context(), deps, svc))
	}))

	// محاسبهٔ فیش یک کارمند با موتور روش پرداخت خودش.
	mux.HandleFunc("POST /api/v1/payroll/payslip", read(func(w http.ResponseWriter, r *http.Request) {
		var in PayslipRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی فیش خوانده نشد.", nil)
			return
		}
		result, err := unifiedPayslip(r.Context(), deps, svc, in)
		if err != nil {
			if verr, ok := err.(*payslipError); ok {
				httpx.WriteErrorMsg(w, verr.status, contracts.CodeValidationFailed, verr.message, nil)
				return
			}
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, result)
	}))
}

// payslipError خطای قابل نمایش به کاربر (۴۲۲/۴۰۴) با پیام فارسی.
type payslipError struct {
	status  int
	message string
}

func (e *payslipError) Error() string { return e.message }

// payslipDefaults فهرست کارکنان همراه روش پرداخت مؤثرشان و توضیح موتورها.
func payslipDefaults(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) map[string]any {
	employees := []map[string]string{}
	if reader, ok := deps.Ports.Employees(); ok {
		if page, err := reader.List(ctx, contracts.EmployeeFilter{Size: 500}); err == nil {
			for _, e := range page.Items {
				model, sourceFa, mismatchFa := employeePayModel(ctx, svc, e.ID, e.EmploymentType)
				employees = append(employees, map[string]string{
					"id":               e.ID,
					"titleFa":          e.FullName + " — " + e.ID,
					"unit":             e.DepartmentName,
					"employmentType":   e.EmploymentType,
					"employmentTypeFa": e.EmploymentTypeFa,
					"payModel":         model,
					"payModelFa":       rules.PayModelTitleFa(model),
					"payModelSourceFa": sourceFa,
					"baseSalary":       intPad(e.BaseSalary),
					"warningFa":        mismatchFa,
				})
			}
		}
	}
	return map[string]any{
		"employees": employees,
		"engines": []map[string]string{
			{"key": rules.PayModelPayroll, "titleFa": rules.PayModelTitleFa(rules.PayModelPayroll),
				"engineFa": "حقوق و دستمزد جاری (پایه + مزایای قانونی + بیمه و مالیات)"},
			{"key": rules.PayModelOil, "titleFa": rules.PayModelTitleFa(rules.PayModelOil),
				"engineFa": "فیش صندوق نفت (جدول‌های ۱ تا ۳ آیین‌نامه و گروه جدول ۳)"},
			{"key": rules.PayModelComp34006, "titleFa": rules.PayModelTitleFa(rules.PayModelComp34006),
				"engineFa": "فیش دستورالعمل ۳۴۰۰۶ (اقلام کددار)"},
			{"key": rules.PayModelFourP, "titleFa": rules.PayModelTitleFa(rules.PayModelFourP),
				"engineFa": "مدل توسعه‌ای ۴پی؛ پرداخت با موتور جاری و تصمیم مدل در ماژول جبران خدمات"},
		},
		"period": oilPeriod(deps),
		"today":  payTodayFa(deps),
		"noteFa": "روش پرداخت هر نفر از پروندهٔ روش پرداخت کارمند خوانده می‌شود و در نبود آن از " +
			"نوع استخدام استنتاج می‌گردد؛ موتور فیش بر همان پایه انتخاب می‌شود.",
	}
}

// unifiedPayslip فیش را با موتور روش پرداخت همان کارمند می‌سازد.
func unifiedPayslip(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in PayslipRequest) (map[string]any, error) {
	employeeID := strings.TrimSpace(in.EmployeeID)
	if employeeID == "" {
		return nil, &payslipError{status: http.StatusUnprocessableEntity, message: "انتخاب کارمند الزامی است."}
	}
	employee, ok := employeeOf(ctx, deps, employeeID)
	if !ok {
		return nil, &payslipError{status: http.StatusNotFound, message: "کارمند انتخابی یافت نشد."}
	}

	model, sourceFa, mismatchFa := employeePayModel(ctx, svc, employee.ID, employee.EmploymentType)
	if override := strings.TrimSpace(in.PayModel); override != "" {
		if !rules.ValidPayModel(override) {
			return nil, &payslipError{status: http.StatusUnprocessableEntity,
				message: "روش پرداخت انتخابی در فهرست روش‌های مصوب نیست."}
		}
		if allowed, reason := rules.PayModelAllowed(employee.EmploymentType, override); !allowed {
			return nil, &payslipError{status: http.StatusUnprocessableEntity, message: reason}
		}
		model, sourceFa, mismatchFa = rules.PayModelOf(override), "انتخاب صریح برای شبیه‌سازی", ""
	}

	period := strings.TrimSpace(in.PeriodFa)
	if period == "" {
		period = oilPeriod(deps)
	}

	result := map[string]any{
		"payModel":   model,
		"payModelFa": rules.PayModelTitleFa(model),
		"sourceFa":   sourceFa,
		"period":     period,
		"today":      payTodayFa(deps),
		"employee": map[string]any{
			"id":               employee.ID,
			"name":             employee.FullName,
			"unit":             employee.DepartmentName,
			"employmentType":   employee.EmploymentType,
			"employmentTypeFa": employee.EmploymentTypeFa,
		},
	}
	notes := []string{}
	if mismatchFa != "" {
		notes = append(notes, mismatchFa)
	}

	switch model {
	case rules.PayModelOil:
		payload := oilPayslip(ctx, deps, oilInputFor(employee, in, oilPayParams(deps, nil)))
		result["engineFa"] = "فیش صندوق بازنشستگی نفت"
		result["slip"] = payload["slip"]
		result["details"] = payload
		notes = append(notes, "گروه نظام پرداخت (جدول ۳) از نوع استخدام و پایهٔ سمت تعیین شد؛ "+
			"برای کارمند رسمی نفت همیشه «گروه یکم» است.")

	case rules.PayModelComp34006:
		cp := comp34006InputFor(employee, in)
		payload := comp34006Payslip(ctx, deps, svc, cp)
		result["engineFa"] = "فیش دستورالعمل ۳۴۰۰۶"
		result["slip"] = payload["slip"]
		result["details"] = payload
		notes = append(notes, rules.Comp34006IndependenceFa(false))

	default: // payroll و fourp
		payload := payrollPayslip(ctx, deps, svc, employee, in, period)
		result["engineFa"] = "فیش حقوق و دستمزد جاری"
		result["slip"] = payload["slip"]
		result["details"] = payload
		if model == rules.PayModelFourP {
			result["engineFa"] = "فیش حقوق و دستمزد جاری + مدل توسعه‌ای ۴پی"
			notes = append(notes, "مدل ۴پی مدل توسعه‌ای است و خودش فیش نمی‌سازد: مبلغ مصوب مدل در حکم "+
				"ثبت می‌شود و پرداخت با موتور جاری انجام می‌گیرد؛ کارنامهٔ مدل در ماژول جبران خدمات است.")
		}
	}

	result["notesFa"] = notes
	return result, nil
}

// oilInputFor ورودی موتور صندوق نفت را از پروندهٔ کارمند کامل می‌کند.
func oilInputFor(employee contracts.Employee, in PayslipRequest, params rules.OilPayParams) rules.OilPayInput {
	out := rules.OilPayInput{}
	if in.Oil != nil {
		out = *in.Oil
	}
	if strings.TrimSpace(out.EmployeeID) == "" {
		out.EmployeeID = employee.ID
	}
	if out.BaseSalary <= 0 {
		out.BaseSalary = employee.BaseSalary
	}
	if out.OtherDeductions <= 0 {
		out.OtherDeductions = in.OtherDeductions
	}
	if strings.TrimSpace(out.EmploymentType) == "" {
		out.EmploymentType = employee.EmploymentType
	}
	if strings.TrimSpace(out.Residence) == "" {
		out.Residence = employee.ResidenceSite
	}
	if strings.TrimSpace(out.JobGrade) == "" {
		out.JobGrade = rules.OilSiteForGrade(params, employee.JobGrade)
	}
	return out
}

// comp34006InputFor ورودی موتور ۳۴۰۰۶ را کامل می‌کند (گرید شغلی از پرونده می‌آید).
func comp34006InputFor(employee contracts.Employee, in PayslipRequest) rules.Comp34006PayslipInput {
	out := rules.Comp34006PayslipInput{}
	if in.Comp34006 != nil {
		out = *in.Comp34006
	}
	if strings.TrimSpace(out.EmployeeID) == "" {
		out.EmployeeID = employee.ID
	}
	if strings.TrimSpace(out.EmployeeName) == "" {
		out.EmployeeName = employee.FullName
	}
	if strings.TrimSpace(out.PeriodFa) == "" {
		out.PeriodFa = strings.TrimSpace(in.PeriodFa)
	}
	if out.OtherDeductions <= 0 {
		out.OtherDeductions = in.OtherDeductions
	}
	return out
}

// payrollPayslip فیش موتور جاری سازمان را می‌سازد (و مبنای آن از پرونده/ورودی می‌آید).
func payrollPayslip(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service,
	employee contracts.Employee, in PayslipRequest, period string) map[string]any {

	input := rules.PayrollInput{}
	if in.Payroll != nil {
		input = *in.Payroll
	}
	if input.BaseSalary <= 0 {
		input.BaseSalary = employee.BaseSalary
	}
	if input.OtherDeductions <= 0 {
		input.OtherDeductions = in.OtherDeductions
	}
	input.Veteran = veteranEntitlement(ctx, deps, employee.ID)

	params, _, _ := collectParams(ctx, deps, svc)
	slip := rules.ComputePayroll(input, params)
	return map[string]any{
		"scheme":   "payroll",
		"schemeFa": "حقوق و دستمزد جاری سازمان",
		"slip":     slip,
		"params":   params,
		"input":    input,
		"period":   period,
		"today":    payTodayFa(deps),
		"employee": map[string]string{"id": employee.ID, "name": employee.FullName},
		"noteFa":   "پایهٔ حقوق از پروندهٔ کارمند و در صورت ورود، از درخواست خوانده می‌شود؛ بیمه و مالیات با پارامترهای جاری سازمان محاسبه می‌شود.",
	}
}

// employeePayModel روش پرداخت مؤثر یک کارمند را برمی‌گرداند.
//
// ترتیب: پروندهٔ «روش پرداخت کارمند» (رکورد فعال با آخرین تاریخ اجرا) و در نبود آن،
// پیش‌فرض نوع استخدام (رسمی نفت ⇒ صندوق نفت، بقیه ⇒ حقوق و دستمزد جاری).
//
// اگر پرونده با نوع استخدام ناسازگار باشد (مثلاً صندوق نفت برای کارمند غیررسمی)، فیش با
// پیش‌فرض نوع استخدام ساخته می‌شود و هشدار فارسی برگردانده می‌شود؛ این‌گونه یک رکورد اشتباه
// جلوی صدور فیش را نمی‌گیرد ولی پنهان هم نمی‌ماند.
func employeePayModel(ctx context.Context, svc *recordkit.Service, employeeID, employmentType string) (model, sourceFa, mismatchFa string) {
	if svc != nil {
		if rec, ok := activePayModelRecord(ctx, svc, employeeID); ok {
			fromRecord := rules.PayModelOf(rec.Fields["payModel"])
			if allowed, reason := rules.PayModelAllowed(employmentType, fromRecord); allowed {
				return fromRecord, "پروندهٔ روش پرداخت کارمند (از " + rec.Fields["effectiveDate"] + ")", ""
			} else {
				return rules.PayModelDefaultFor(employmentType),
					"پیش‌فرض نوع استخدام (پروندهٔ روش پرداخت ناسازگار است)",
					"پروندهٔ روش پرداخت کارمند با نوع استخدام او سازگار نیست و نادیده گرفته شد: " + reason
			}
		}
	}
	return rules.PayModelDefaultFor(employmentType), "پیش‌فرض نوع استخدام", ""
}

// employeeOf خواندن کارمند از پورت ماژول کارکنان.
func employeeOf(ctx context.Context, deps modulekit.Deps, employeeID string) (contracts.Employee, bool) {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return contracts.Employee{}, false
	}
	employee, found, err := reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
	if err != nil || !found {
		return contracts.Employee{}, false
	}
	return employee, true
}

// intPad تبدیل عدد به رشته برای فهرست انتخاب فرم.
func intPad(v int64) string { return strconv.FormatInt(v, 10) }
