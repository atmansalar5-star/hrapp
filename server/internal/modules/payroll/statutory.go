// پیاده‌سازی ماژول «payroll» — «statutory»: فایل قانونی بیمهٔ تأمین اجتماعی و مالیات حقوق
// (زیرکار ۱.۴ فاز ۱ نقشهٔ راه).
//
// قاعدهٔ معماری: مبلغ‌ها از موتور قواعد و کارکرد از پورت قراردادی ماژول حضور می‌آید؛
// این لایه ردیف‌های فایل، جمع‌های کنترلی، مهلت و برآورد جریمه را از دامنه می‌گیرد
// (`domain.BuildStatutoryFile`) و خروجی CSV و سند گردش فایل را می‌سازد.

package payroll

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/payroll/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
)

// registerStatutoryEndpoints مسیرهای فایل قانونی بیمه/مالیات را ثبت می‌کند.
func registerStatutoryEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "write", fn) }

	// گزارش فایل قانونی (JSON): ردیف‌ها، جمع‌های کنترلی، مهلت و جریمهٔ برآوردی.
	mux.HandleFunc("GET /api/v1/payroll/statutory/{kind}", read(func(w http.ResponseWriter, r *http.Request) {
		file, err := statutoryFile(r.Context(), deps, svc, r.PathValue("kind"), periodOf(r))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, file)
	}))

	// خروجی CSV آمادهٔ ارسال به تأمین اجتماعی/سازمان امور مالیاتی (UTF-8 با BOM).
	mux.HandleFunc("GET /api/v1/payroll/statutory/{kind}/file", read(func(w http.ResponseWriter, r *http.Request) {
		kind := r.PathValue("kind")
		file, err := statutoryFile(r.Context(), deps, svc, kind, periodOf(r))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		body := statutoryCSV(file)
		name := fmt.Sprintf("hrms-%s-%s.csv", kind, strings.ReplaceAll(file.Period, "/", "-"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))

	// ثبت سند فایل قانونی روی گردشکار ماژول (آمادهٔ ارسال ← ارسال‌شده ← پرداخت‌شده).
	mux.HandleFunc("POST /api/v1/payroll/statutory/{kind}", write(func(w http.ResponseWriter, r *http.Request) {
		kind := r.PathValue("kind")
		file, err := statutoryFile(r.Context(), deps, svc, kind, periodOf(r))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		if file.Totals.Rows == 0 {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict,
				"برای این دوره هیچ ردیف محاسبه‌شده‌ای نیست؛ اول کارکرد دوره را محاسبه کنید.", nil)
			return
		}
		issues := ""
		if len(file.IssuesFa) > 0 {
			issues = strings.Join(file.IssuesFa, " | ")
			if len([]rune(issues)) > 900 {
				issues = string([]rune(issues)[:900]) + " …"
			}
		}
		rec, err := svc.Create(r.Context(), contracts.RecordInput{
			Kind:  "statutoryFiles",
			Title: file.TitleFa + " — " + file.Period,
			Fields: map[string]string{
				"fileType":          fileTypeFa(file.Kind),
				"period":            file.Period,
				"rows":              strconv.Itoa(file.Totals.Rows),
				"payableTotal":      strconv.FormatInt(file.PayableTotal, 10),
				"insuranceEmployee": strconv.FormatInt(file.Totals.EmployeeShare, 10),
				"insuranceEmployer": strconv.FormatInt(file.Totals.EmployerShare, 10),
				"unemployment":      strconv.FormatInt(file.Totals.Unemployment, 10),
				"taxTotal":          strconv.FormatInt(file.Totals.Tax, 10),
				"deadline":          file.DeadlineFa,
				"penalty":           strconv.FormatInt(file.Penalty, 10),
				"issues":            issues,
				"noteFa":            file.NoteFa,
			},
			NoteFa: file.NoteFa,
		}, actorOf(r))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, map[string]any{
			"record": rec, "file": file,
			"downloadUrl": "/api/v1/payroll/statutory/" + kind + "/file?period=" + file.Period,
		})
	}))
}

// statutoryFile فایل قانونی یک دوره را از کارکرد و مبلغ‌های محاسبه‌شده می‌سازد.
func statutoryFile(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, kind, period string) (domain.StatutoryFile, error) {
	if strings.TrimSpace(period) == "" {
		period = periodFa(deps)
	}
	rates := statutoryRates(ctx, deps, svc)
	inputs, err := statutoryInputs(ctx, deps, svc, period, rates)
	if err != nil {
		return domain.StatutoryFile{}, err
	}
	today := ""
	if now, err := calendar.FromGregorian(deps.Now()); err == nil {
		today = now.String()
	}
	return domain.BuildStatutoryFile(kind, period, inputs, rates, today)
}

// statutoryInputs ورودی هر ردیف فایل را از کارکرد دوره می‌سازد: روز مشمول، جمع حقوق،
// مالیات محاسبه‌شده و درآمد مشمول (جمع − سهم بیمهٔ کارمند − معافیت پارامتر).
func statutoryInputs(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, period string, rates domain.StatutoryRates) ([]domain.StatutoryInput, error) {
	rows, _, err := workTimeRows(ctx, deps, svc, period)
	if err != nil {
		return nil, err
	}
	nationalIDs := employeeNationalIDs(ctx, deps)

	out := make([]domain.StatutoryInput, 0, len(rows))
	for _, row := range rows {
		taxable := row.Gross - row.InsuranceEmployee - rates.TaxExemption
		if taxable < 0 {
			taxable = 0
		}
		days := int(row.PayableDays + 0.5)
		if days > 31 {
			days = 31
		}
		name := row.EmployeeName
		if name == "" {
			name = row.EmployeeID
		}
		out = append(out, domain.StatutoryInput{
			EmployeeID:     row.EmployeeID,
			EmployeeName:   name,
			NationalID:     nationalIDs[row.EmployeeID],
			DepartmentName: row.DepartmentName,
			Days:           days,
			Gross:          row.Gross,
			TaxExemption:   rates.TaxExemption,
			TaxableIncome:  taxable,
			Tax:            row.Tax,
			Net:            row.Net,
		})
	}
	return out, nil
}

// statutoryRates نرخ‌ها و سقف‌های فایل قانونی را از پارامترهای شرکت فعال می‌خواند.
//
// نرخ کارفرما در پارامتر «بیمهٔ کارفرما» (پیش‌فرض ۰٫۲۳) هم‌ارز ۲۰٪ سهم کارفرما + ۳٪
// بیمهٔ بیکاری است؛ پس سهم کارفرمای فایل از آن منهای نرخ بیکاری ساخته می‌شود تا مبلغ
// دو بار شمرده نشود.
func statutoryRates(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) domain.StatutoryRates {
	out := domain.DefaultStatutoryRates()
	params, values, _ := collectParams(ctx, deps, svc)
	out.EmployeeRate = params.InsuranceEmployeeRate
	out.Ceiling = params.InsuranceCeiling
	out.TaxExemption = params.TaxExemption
	unemployment := 0.03
	if v, ok := floatOf(values["insurance.unemploymentRate"]); ok && v > 0 {
		unemployment = v
	}
	if params.InsuranceEmployerRate > unemployment {
		out.EmployerRate = params.InsuranceEmployerRate - unemployment
	} else {
		out.EmployerRate = params.InsuranceEmployerRate
	}
	out.UnemploymentRate = unemployment
	if v, ok := floatOf(values["statutory.insurancePenaltyMonthly"]); ok && v >= 0 {
		out.InsurancePenalty = v
	}
	if v, ok := floatOf(values["statutory.taxPenaltyMonthly"]); ok && v >= 0 {
		out.TaxPenalty = v
	}
	// شناسه‌های قانونی شرکت: از پارامتر شرکت فعال، و اگر نبود از پروفایل شرکت.
	for _, p := range deps.Org.Params {
		switch p.Key {
		case "insurance.workshopCode", "socialSecurity.workshopCode":
			out.WorkshopCode = strings.TrimSpace(p.Value)
		case "tax.fileNumber":
			out.TaxFileNumber = strings.TrimSpace(p.Value)
		case "insurance.branchFa":
			out.InsuranceBranchFa = strings.TrimSpace(p.Value)
		}
	}
	return out
}

// statutoryCSV فایل قانونی را به CSV با BOM تبدیل می‌کند (Excel فارسی درست باز کند).
func statutoryCSV(file domain.StatutoryFile) string {
	rows := [][]string{domain.StatutoryColumns(file.Kind)}
	for _, row := range file.Rows {
		rows = append(rows, domain.StatutoryRowValues(file.Kind, row))
	}
	rows = append(rows, domain.StatutoryFooter(file)...)
	return csvText(rows)
}

// csvText جدول را به CSV با سرصفحهٔ BOM تبدیل می‌کند.
func csvText(rows [][]string) string {
	var buf bytes.Buffer
	buf.WriteString("\ufeff")
	w := csv.NewWriter(&buf)
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()
	return buf.String()
}

// fileTypeFa نام فارسی نوع فایل برای فیلد رکورد.
func fileTypeFa(kind string) string {
	if kind == domain.StatutoryTax {
		return "مالیات"
	}
	return "بیمه"
}

// employeeNationalIDs نقشهٔ «شناسهٔ کارمند ← کد ملی» از پورت کارکنان.
func employeeNationalIDs(ctx context.Context, deps modulekit.Deps) map[string]string {
	out := map[string]string{}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return out
	}
	page, err := reader.List(ctx, contracts.EmployeeFilter{Scope: "all", Size: 500})
	if err != nil {
		return out
	}
	for _, emp := range page.Items {
		out[emp.ID] = strings.TrimSpace(emp.NationalID)
	}
	return out
}

// periodOf دورهٔ درخواست را می‌خواند؛ نبودش یعنی ماه جاری سرور.
func periodOf(r *http.Request) string {
	if value := strings.TrimSpace(calendar.NormalizeDigits(r.URL.Query().Get("period"))); value != "" {
		return value
	}
	return ""
}
