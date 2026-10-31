// پیاده‌سازی ماژول «payroll» — «bankfile»: فایل پرداخت بانکی (زیرکار ۱.۵ فاز ۱).
//
// قاعدهٔ معماری: مبلغ هر کارمند از محاسبهٔ همان دوره می‌آید (خالص پرداختی)، حساب‌ها از
// رکوردهای «حساب بانکی کارمند» خوانده می‌شود و قاعده‌های گرد کردن/جمع کنترلی/سنجش شبا
// در دامنه است (`domain.BuildBankFileFromAccounts`). این لایه فقط سیم‌کشی و خروجی است.

package payroll

import (
	"context"
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

// registerBankFileEndpoints مسیرهای فایل پرداخت بانکی را ثبت می‌کند.
func registerBankFileEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("payroll", "write", fn) }

	// گزارش فایل پرداخت (JSON): ردیف‌ها، جمع کنترلی، برگشتی‌ها و بدون‌حساب‌ها.
	mux.HandleFunc("GET /api/v1/payroll/bank-file", read(func(w http.ResponseWriter, r *http.Request) {
		file, err := bankFile(r.Context(), deps, svc, r)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, file)
	}))

	// خروجی CSV فایل پرداخت (UTF-8 با BOM) برای دادن به سامانهٔ بانک.
	mux.HandleFunc("GET /api/v1/payroll/bank-file/download", read(func(w http.ResponseWriter, r *http.Request) {
		file, err := bankFile(r.Context(), deps, svc, r)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		rows := [][]string{domain.BankColumns()}
		for _, row := range file.Rows {
			rows = append(rows, domain.BankRowValues(row))
		}
		rows = append(rows, domain.BankFooter(file)...)
		name := "hrms-bank-" + strings.ReplaceAll(file.Period, "/", "-") + ".csv"
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(csvText(rows)))
	}))

	// ثبت سند فایل پرداخت روی گردشکار (آمادهٔ ارسال ← ارسال‌شده ← واریزشده/برگشتی).
	mux.HandleFunc("POST /api/v1/payroll/bank-file", write(func(w http.ResponseWriter, r *http.Request) {
		file, err := bankFile(r.Context(), deps, svc, r)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusConflict, contracts.CodeConflict, err.Error(), nil)
			return
		}
		if err := domain.BankFileKindReady(file); err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		issues := strings.Join(file.IssuesFa, " | ")
		if len([]rune(issues)) > 900 {
			issues = string([]rune(issues)[:900]) + " …"
		}
		title := "فایل پرداخت بانکی — " + file.Period
		if file.BankNameFa != "" {
			title = "فایل پرداخت " + file.BankNameFa + " — " + file.Period
		}
		rec, err := svc.Create(r.Context(), contracts.RecordInput{
			Kind: "bankFiles", Title: title,
			Fields: map[string]string{
				"bankName":       file.BankNameFa,
				"period":         file.Period,
				"rows":           strconv.Itoa(file.Totals.Rows),
				"amount":         strconv.FormatInt(file.Totals.Amount, 10),
				"rounding":       strconv.FormatInt(file.Totals.Rounding, 10),
				"returnedRows":   strconv.Itoa(file.Totals.ReturnedRows),
				"returnedAmount": strconv.FormatInt(file.Totals.ReturnedSum, 10),
				"controlFa":      file.ControlFa,
				"issues":         issues,
				"noteFa":         file.NoteFa,
			},
			NoteFa: file.NoteFa,
		}, actorOf(r))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusUnprocessableEntity, contracts.CodeConflict, err.Error(), nil)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, map[string]any{
			"record": rec, "file": file,
			"downloadUrl": "/api/v1/payroll/bank-file/download?period=" + file.Period,
		})
	}))
}

// bankFile فایل پرداخت را برای دوره/بانک درخواست‌شده می‌سازد.
func bankFile(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, r *http.Request) (domain.BankFile, error) {
	period := periodOf(r)
	if strings.TrimSpace(period) == "" {
		period = periodFa(deps)
	}
	bank := strings.TrimSpace(r.URL.Query().Get("bank"))
	returned := map[string]bool{}
	for _, id := range strings.Split(r.URL.Query().Get("returned"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			returned[id] = true
		}
	}

	amounts, err := netAmounts(ctx, deps, svc, period)
	if err != nil {
		return domain.BankFile{}, err
	}
	accounts, err := svc.List(ctx, contracts.RecordFilter{
		Module: "payroll", Kind: "bankAccounts", Size: 500, Scope: "all"})
	if err != nil {
		return domain.BankFile{}, err
	}
	_, values, _ := collectParams(ctx, deps, svc)
	roundUnit := domain.DefaultBankRoundUnit
	if v, ok := intOf(values["bank.roundUnit"]); ok && v > 0 {
		roundUnit = v
	}
	today := ""
	if now, err := calendar.FromGregorian(deps.Now()); err == nil {
		today = now.String()
	}
	file := domain.BuildBankFileFromAccounts(period, bank, accounts.Items, amounts, returned, roundUnit, today)
	if len(file.Rows) == 0 && len(file.MissingFa) == 0 {
		file.NoteFa = fmt.Sprintf("%s هیچ حساب بانکی فعالی برای دورهٔ %s ثبت نشده است؛ اول حساب کارکنان را در «حساب بانکی کارمند» وارد کنید.",
			file.NoteFa, period)
	}
	return file, nil
}

// netAmounts مبلغ خالص پرداختی هر کارمند در دوره را از محاسبهٔ همان دوره می‌گیرد.
func netAmounts(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, period string) (map[string]int64, error) {
	rows, _, err := workTimeRows(ctx, deps, svc, period)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.EmployeeID] = row.Net
	}
	return out, nil
}
