// پیاده‌سازی ماژول «payroll» — «worktime».

package payroll

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// این فایل «اتصال حقوق به کارکرد» است.
//
// قاعدهٔ معماری: مبلغ‌ها را موتور قواعد (`rules.ComputePayroll`) حساب می‌کند و کارکرد
// را پورت قراردادی ماژول حضور می‌دهد. این لایه فقط دادهٔ دو طرف را به هم می‌رساند و
// نتیجه را روی رکورد دورهٔ پردازش می‌نویسد؛ هیچ مبلغی اینجا ساخته نمی‌شود.
func registerWorkTimeEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "write", fn) }

	// کارکرد دوره به تفکیک کارمند + مبلغ قابل پرداخت — همان چیزی که در جدول
	// «پردازش حقوق» و «فیش حقوقی» دیده می‌شود.
	mux.HandleFunc("GET /api/v1/payroll/worktime", read(func(w http.ResponseWriter, r *http.Request) {
		period := strings.TrimSpace(r.URL.Query().Get("period"))
		rows, noteFa, err := workTimeRows(r.Context(), deps, svc, period)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"period": period,
			"rows":   rows,
			"total":  len(rows),
			"sum":    summarize(rows),
			"noteFa": noteFa,
		})
	}))

	// دوره‌های دارای کارکرد، برای انتخاب دوره در رابط.
	mux.HandleFunc("GET /api/v1/payroll/worktime-periods", read(func(w http.ResponseWriter, r *http.Request) {
		provider, ok := deps.Ports.WorkTime()
		if !ok {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"periods": []string{}})
			return
		}
		periods, err := provider.Periods(r.Context())
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"periods": periods, "serverPeriod": periodFa(deps)})
	}))

	// اجرای محاسبهٔ دوره بر پایهٔ کارکرد کارکنان.
	//
	// محاسبه مبلغ‌ها را می‌نویسد؛ تغییر وضعیت گردش‌کار (پیش‌نویس ← محاسبه‌شده) همچنان
	// کار اقدام روی رکورد است، پس تفکیک «قاعده» از «گردش‌کار» حفظ می‌شود.
	mux.HandleFunc("POST /api/v1/payroll/runs/{id}/calculate", write(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		run, ok, err := svc.Get(r.Context(), contracts.RecordFilter{Module: "payroll", Kind: "payrollRuns", Scope: "all"}, id)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
			return
		}
		if !ok {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "دورهٔ پردازش یافت نشد", nil)
			return
		}
		period := strings.TrimSpace(run.Fields["period"])
		if period == "" {
			period = periodFa(deps)
		}

		rows, noteFa, err := workTimeRows(r.Context(), deps, svc, period)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		summary := summarize(rows)

		// نوشتن نتیجه روی رکورد دوره. ویرایش فقط در وضعیت‌های مجازِ اعلام‌شدهٔ دامنه
		// انجام می‌گیرد؛ موتور رکورد اگر وضعیت اجازه ندهد، خطای قاعده برمی‌گرداند و
		// نتیجه فقط نمایش داده می‌شود (تغییر وضعیت کار اقدام گردش‌کار است).
		fields := map[string]string{
			"period":    period,
			"gross":     fmt.Sprintf("%d", summary.Gross),
			"tax":       fmt.Sprintf("%d", summary.Tax),
			"insurance": fmt.Sprintf("%d", summary.InsuranceEmployee),
			"net":       fmt.Sprintf("%d", summary.Net),
		}
		in := contracts.RecordInput{
			Kind: "payrollRuns", Title: run.Title, Fields: fields,
			NoteFa: fmt.Sprintf("محاسبه از کارکرد دوره %s برای %d کارمند انجام شد.", period, summary.Count),
		}
		applied := false
		applyNote := ""
		if _, err := svc.Update(r.Context(), id, in, actorOf(r)); err == nil {
			applied = true
		} else {
			applyNote = err.Error()
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"runId": id, "period": period, "rows": rows, "sum": summary,
			"applied": applied, "applyNoteFa": applyNote, "noteFa": noteFa,
		})
	}))
}

// workTimeRow کارکرد یک کارمند به‌همراه مبلغ محاسبه‌شده با موتور قواعد.
type workTimeRow struct {
	contracts.WorkTimeRow
	BaseSalary int64 `json:"baseSalary"`
	// DailyWage مزد روزانه و OvertimeHourlyRate نرخ ساعتی اضافه‌کاری.
	DailyWage          int64  `json:"dailyWage"`
	OvertimeHourlyRate int64  `json:"overtimeHourlyRate"`
	OvertimePay        int64  `json:"overtimePay"`
	UnpaidDeduction    int64  `json:"unpaidDeduction"`
	Gross              int64  `json:"gross"`
	InsuranceEmployee  int64  `json:"insuranceEmployee"`
	Tax                int64  `json:"tax"`
	Net                int64  `json:"net"`
	NoteFa             string `json:"noteFa,omitempty"`
}

// workTimeRows کارکرد دوره را می‌خواند و برای هر کارمند، ورودی موتور حقوق را از آن
// می‌سازد: اضافه‌کاری تأییدشده، کسر مرخصی بی‌حقوق/غیبت و مزد پایهٔ کارمند.
func workTimeRows(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, period string) ([]workTimeRow, string, error) {
	provider, ok := deps.Ports.WorkTime()
	if !ok {
		return nil, "پورت کارکرد دوره در این استقرار در دسترس نیست؛ ماژول حضور باید ثبتش کند.", nil
	}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return nil, "پورت کارکنان در این استقرار در دسترس نیست.", nil
	}
	emps, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 500})
	if err != nil {
		return nil, "", err
	}
	params, values, _ := collectParams(ctx, deps, svc)
	wt := rules.DefaultWorkTimeParams()
	if hours, ok := intOf(values["payroll.monthlyHours"]); ok && hours > 0 {
		wt.MonthHours = float64(hours)
	}

	rows, err := provider.WorkTime(ctx, contracts.WorkTimeQuery{Period: period})
	if err != nil {
		return nil, "", err
	}
	byEmployee := map[string]contracts.WorkTimeRow{}
	for _, row := range rows {
		byEmployee[row.EmployeeID] = row
	}

	out := make([]workTimeRow, 0, len(emps.Items))
	for _, emp := range emps.Items {
		wtRow, hasWork := byEmployee[emp.ID]
		if !hasWork {
			wtRow = contracts.WorkTimeRow{EmployeeID: emp.ID, EmployeeName: emp.FullName, DepartmentName: emp.DepartmentName, Period: period}
		}
		overRate := rules.OvertimeHourlyRate(emp.BaseSalary, wtRow.OvertimeFactor, wt)
		input := rules.PayrollInput{
			BaseSalary:         emp.BaseSalary,
			OvertimeHours:      int64(wtRow.OvertimeHours),
			OvertimeHourlyRate: overRate,
			OtherDeductions:    rules.UnpaidDeduction(wtRow.DeductibleDays, emp.BaseSalary, wt),
		}
		result := rules.ComputePayroll(input, params)

		note := ""
		if !hasWork {
			note = "برای این کارمند در این دوره کارکردی ثبت نشده است؛ فقط مزد پایه محاسبه شد."
		} else if wtRow.PendingRecords > 0 {
			note = fmt.Sprintf("%d رکورد در گردش است و در این محاسبه اثر ندارد.", wtRow.PendingRecords)
		}

		out = append(out, workTimeRow{
			WorkTimeRow:        wtRow,
			BaseSalary:         emp.BaseSalary,
			DailyWage:          rules.RoundHalfUp(rules.DailyWage(emp.BaseSalary, wt)),
			OvertimeHourlyRate: overRate,
			OvertimePay:        result.OvertimePay,
			UnpaidDeduction:    input.OtherDeductions,
			Gross:              result.Gross,
			InsuranceEmployee:  result.InsuranceEmployee,
			Tax:                result.Tax,
			Net:                result.Net,
			NoteFa:             note,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Net > out[j].Net })

	noteFa := "کارکرد از رکوردهای مؤثر ماژول حضور خوانده می‌شود (تردد ثبت‌شده، اضافه‌کاری/مرخصی/مأموریت تأییدشده). مبلغ‌ها با موتور قواعد و پارامترهای شرکت فعال محاسبه می‌شود."
	if period == "" {
		noteFa += " دوره‌ای انتخاب نشده، پس همهٔ رکوردهای حضور در جمع آمده است."
	}
	return out, noteFa, nil
}

// workTimeTotals جمع یک دوره برای نوشتن روی رکورد دورهٔ پردازش.
type workTimeTotals struct {
	Count             int     `json:"count"`
	Gross             int64   `json:"gross"`
	Tax               int64   `json:"tax"`
	InsuranceEmployee int64   `json:"insuranceEmployee"`
	InsuranceEmployer int64   `json:"insuranceEmployer"`
	Net               int64   `json:"net"`
	OvertimePay       int64   `json:"overtimePay"`
	UnpaidDeduction   int64   `json:"unpaidDeduction"`
	WorkedHours       float64 `json:"workedHours"`
	OvertimeHours     float64 `json:"overtimeHours"`
}

// summarize جمع ردیف‌های کارکرد را می‌سازد.
func summarize(rows []workTimeRow) workTimeTotals {
	out := workTimeTotals{Count: len(rows)}
	for _, r := range rows {
		out.Gross += r.Gross
		out.Tax += r.Tax
		out.InsuranceEmployee += r.InsuranceEmployee
		out.Net += r.Net
		out.OvertimePay += r.OvertimePay
		out.UnpaidDeduction += r.UnpaidDeduction
		out.WorkedHours += r.WorkedHours
		out.OvertimeHours += r.OvertimeHours
	}
	// بیمهٔ کارفرما از پارامترها خوانده نمی‌شود؛ جمع دوره فقط سهم کارمند را دارد.
	return out
}

// actorOf نام کاربر جاری برای ثبت رخداد ویرایش.
func actorOf(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok && user.Username != "" {
		return user.Username
	}
	return "payroll"
}
