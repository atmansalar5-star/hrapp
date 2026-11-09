// پیاده‌سازی ماژول «درمان» — کمک هزینه‌های درمانی بر پایهٔ تعرفهٔ مصوب.
//
// چرا این فایل: دستورالعمل سازمان بهداشت و درمان صنعت نفت، تعرفهٔ هر قلم (اروتز و پروتز،
// دارو، عینک، دندانپزشکی، پاراکلینیک، ویزیت و سایر) را سالانه تعیین می‌کند. تعرفه‌ها در
// رکوردهای همین ماژول نگه داشته می‌شوند و قاعدهٔ محاسبه در موتور قواعد است
// (`rules.ComputeMedicalAid`)؛ این فایل فقط تعرفه‌ها را می‌خواند، قلم‌های درخواست را تجزیه
// می‌کند، سهم سازمان/بیمه/کارمند را حساب می‌کند و نتیجه را در رکورد می‌نویسد. پس فرم وب،
// سند چاپی و گزارش مالی یک عدد می‌بینند و هیچ نرخی در رابط ساخته نمی‌شود.
package medical

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/medical/domain"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registerMedicalAidEndpoints مسیرهای کمک هزینهٔ درمانی را ثبت می‌کند.
func registerMedicalAidEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("medical", "read", fn) }
	write := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("medical", "write", fn) }

	// فهرست‌های مرجع: دسته‌ها، تعرفه‌های جاری، پارامترها و متن دستورالعمل.
	mux.HandleFunc("GET /api/v1/medical/aid-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, aidDefaults(r.Context(), deps, svc))
	}))

	// محاسبهٔ پیش‌نمایش درخواست (بدون تغییر داده): سهم سازمان، بیمه، کارمند و سقف‌ها.
	mux.HandleFunc("POST /api/v1/medical/aid-plan", read(func(w http.ResponseWriter, r *http.Request) {
		var in aidRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی محاسبهٔ کمک هزینه خوانده نشد.", nil)
			return
		}
		out, msg := aidPlan(r.Context(), deps, svc, in)
		if msg != "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, msg, nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
	}))

	// ثبت درخواست با محاسبهٔ سرور: مبالغ از همین‌جا نوشته می‌شوند، نه از رابط.
	mux.HandleFunc("POST /api/v1/medical/aid-claims", write(func(w http.ResponseWriter, r *http.Request) {
		var in aidRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی درخواست کمک هزینه خوانده نشد.", nil)
			return
		}
		out, msg := aidCreate(r.Context(), deps, svc, in, actorOf(r))
		if msg != "" {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, msg, nil)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, out)
	}))

	// جمع پرداخت‌های همان سال برای کنترل سقف سالانه.
	mux.HandleFunc("GET /api/v1/medical/aid-annual", read(func(w http.ResponseWriter, r *http.Request) {
		employeeID := strings.TrimSpace(r.URL.Query().Get("employeeId"))
		year := intOf(r.URL.Query().Get("year"))
		httpx.WriteJSON(w, http.StatusOK, aidAnnual(r.Context(), deps, svc, employeeID, year))
	}))
}

// aidRequest ورودی محاسبه یا ثبت درخواست کمک هزینه.
type aidRequest struct {
	ClaimID        string `json:"claimId,omitempty"`
	EmployeeID     string `json:"employeeId"`
	Category       string `json:"category"`
	PeriodYear     int    `json:"periodYear"`
	InvoiceDate    string `json:"invoiceDate"`
	InvoiceNo      string `json:"invoiceNo"`
	Provider       string `json:"provider"`
	PrescriptionNo string `json:"prescriptionNo"`
	// PrescriptionApproved نسخه/معرفی‌نامه تأیید شده است.
	PrescriptionApproved bool `json:"prescriptionApproved"`
	// Items اقلام درخواست: «کد تعرفه|تعداد|مبلغ فاکتور»، جداکنندهٔ قلم‌ها «؛».
	Items string `json:"items"`
	// AnnualPaidOverride جمع پرداخت پیشین سال (اگر خالی باشد سرور خودش جمع می‌زند).
	AnnualPaidOverride int64  `json:"annualPaidOverride"`
	NoteFa             string `json:"noteFa"`
}

// claimItemRef یک قلم تجزیه‌شدهٔ درخواست.
type claimItemRef struct {
	Code     string
	Quantity int
	Amount   int64
}

// aidDefaults فهرست‌های مرجع و متن دستورالعمل.
func aidDefaults(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service) map[string]any {
	tariffs := activeTariffs(ctx, svc)
	byCategory := map[string][]map[string]any{}
	for _, rec := range tariffs {
		category := strings.TrimSpace(rec.Fields["category"])
		byCategory[category] = append(byCategory[category], tariffOption(rec))
	}
	options := make([]map[string]any, 0, len(tariffs))
	for _, rec := range tariffs {
		options = append(options, tariffOption(rec))
	}
	return map[string]any{
		"categories": rules.MedicalAidCategories(),
		// فهرست گزینه‌های بله/خیر: مقدارهای مجاز فیلدهای انتخابی دامنه است و رابط همان را
		// نشان می‌دهد (در UI هیچ گزینه‌ای ساخته نمی‌شود).
		"yesNo":           yesNoOptions(),
		"tariffs":         options,
		"byCategory":      byCategory,
		"params":          medicalAidParams(deps),
		"paramViews":      medicalAidParamViews(deps),
		"instruction":     rules.MedicalAidInstructionFa(),
		"today":           todayFa(deps),
		"currentYear":     jalaliYear(deps),
		"requirePrescFa":  yesNoFa(medicalAidParams(deps).RequirePrescription),
		"annualCeilingFa": rules.IntMoneyFa(medicalAidParams(deps).AnnualCeiling),
		"noteFa": "هر قلم درخواست از فهرست تعرفه‌های جاری انتخاب می‌شود؛ سازمان بیش از تعرفهٔ " +
			"مصوب نمی‌پردازد و مازاد فاکتور بر عهدهٔ کارمند است. تعرفهٔ هر سال در همین ماژول " +
			"ثبت و با تاریخ اجرا جاری می‌شود.",
	}
}

// tariffOption یک ردیف تعرفه به شکل گزینهٔ فرم.
func tariffOption(rec contracts.Record) map[string]any {
	fields := rec.Fields
	return map[string]any{
		"code":                  fields["code"],
		"titleFa":               rec.Title,
		"category":              fields["category"],
		"categoryFa":            rules.MedicalAidCategoryTitleFa(fields["category"]),
		"unitFa":                fields["unitFa"],
		"amount":                moneyOf(fields["amount"]),
		"amountFa":              rules.IntMoneyFa(moneyOf(fields["amount"])),
		"ceilingAmount":         moneyOf(fields["ceilingAmount"]),
		"orgSharePercent":       percentOf(fields["orgSharePercent"]),
		"insuranceSharePercent": percentOf(fields["insuranceSharePercent"]),
		"requirePrescription":   strings.TrimSpace(fields["requirePrescription"]) == "yes",
		"effectiveFrom":         fields["effectiveFrom"],
		"refFa":                 fields["refFa"],
		"recordId":              rec.ID,
		"status":                rec.Status,
		"statusFa":              rec.StatusFa,
	}
}

// aidPlan محاسبهٔ پیش‌نمایش یک درخواست.
func aidPlan(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in aidRequest) (map[string]any, string) {
	built, msg := buildAidInput(ctx, deps, svc, in)
	if msg != "" {
		return nil, msg
	}
	params := medicalAidParams(deps)
	result := rules.ComputeMedicalAid(built, params)
	return map[string]any{
		"result":       result,
		"params":       params,
		"items":        built.Items,
		"annualPaid":   built.AnnualPaid,
		"periodYear":   built.PeriodYear,
		"today":        todayFa(deps),
		"categoryHint": rules.MedicalAidCategoryTitleFa(built.Category),
		"noteFa": "این مبلغ مبنای ثبت درخواست و پرداخت است؛ برای ثبت، دکمهٔ «ثبت درخواست» را " +
			"بزنید تا سرور همان محاسبه را روی رکورد بنویسد.",
	}, ""
}

// aidCreate محاسبه و ثبت درخواست کمک هزینه روی موتور رکوردها.
//
// چرا محاسبه در سرور: مبالغ پرداختی سازمان باید از یک جای معتبر بیاید؛ اگر رابط مبلغ بفرستد،
// سند مالی به دادهٔ کلاینت تکیه می‌کند. پس هر بار از تعرفهٔ جاری محاسبه و در رکورد نوشته می‌شود.
func aidCreate(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in aidRequest, actor string) (map[string]any, string) {
	built, msg := buildAidInput(ctx, deps, svc, in)
	if msg != "" {
		return nil, msg
	}
	params := medicalAidParams(deps)
	result := rules.ComputeMedicalAid(built, params)

	title := rules.MedicalAidCategoryTitleFa(built.Category) + " — " +
		firstNonEmpty(strings.TrimSpace(in.Provider), "درخواست کمک هزینه")
	fields := map[string]string{
		"category":             built.Category,
		"periodYear":           strconv.Itoa(built.PeriodYear),
		"invoiceDate":          strings.TrimSpace(in.InvoiceDate),
		"invoiceNo":            strings.TrimSpace(in.InvoiceNo),
		"provider":             strings.TrimSpace(in.Provider),
		"prescriptionNo":       strings.TrimSpace(in.PrescriptionNo),
		"prescriptionApproved": yesNoKey(in.PrescriptionApproved),
		"items":                strings.TrimSpace(in.Items),
		"invoiceTotal":         strconv.FormatInt(result.TotalInvoice, 10),
		"totalBase":            strconv.FormatInt(result.TotalBase, 10),
		"payableAmount":        strconv.FormatInt(result.TotalPayable, 10),
		"employeeAmount":       strconv.FormatInt(result.TotalEmployee, 10),
		"insuranceAmount":      strconv.FormatInt(result.TotalInsurance, 10),
		"ceilingApplied":       strconv.FormatInt(result.CeilingApplied, 10),
		"annualPaidBefore":     strconv.FormatInt(result.AnnualPaidBefore, 10),
		"annualPaidAfter":      strconv.FormatInt(result.AnnualPaidAfter, 10),
		"calculationLog":       calculationLog(result),
	}
	rec, err := svc.Create(ctx, contracts.RecordInput{
		Kind: domain.KindClaims, Title: title, EmployeeID: built.EmployeeID,
		Fields: fields, NoteFa: strings.TrimSpace(in.NoteFa),
	}, actor)
	if err != nil {
		return nil, err.Error()
	}
	return map[string]any{
		"claim":  rec,
		"result": result,
		"params": params,
		"noteFa": "درخواست ثبت شد و مبالغ محاسبه‌شده روی رکورد نوشته شد؛ با اقدام‌های گردشکار " +
			"(ارسال، تأیید کارشناس، تصویب، ثبت پرداخت) پیش می‌رود.",
	}, ""
}

// aidAnnual جمع پرداخت‌های همان سال برای یک کارمند.
func aidAnnual(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, employeeID string, year int) map[string]any {
	params := medicalAidParams(deps)
	if year <= 0 {
		year = jalaliYear(deps)
	}
	paid, count := annualPaidOf(ctx, svc, employeeID, year)
	remaining := int64(0)
	if params.AnnualCeiling > 0 {
		remaining = params.AnnualCeiling - paid
		if remaining < 0 {
			remaining = 0
		}
	}
	return map[string]any{
		"employeeId":   employeeID,
		"year":         year,
		"annualPaid":   paid,
		"annualPaidFa": rules.IntMoneyFa(paid),
		"claimsCount":  count,
		"ceiling":      params.AnnualCeiling,
		"remaining":    remaining,
		"remainingFa":  rules.IntMoneyFa(remaining),
		"noteFa":       "در سقف سالانه، درخواست‌های تأییدشده و پرداخت‌شدهٔ همان سال شمرده می‌شوند (پیش‌نویس و ردشده شمرده نمی‌شوند).",
	}
}

// buildAidInput ورودی موتور قواعد را از درخواست و تعرفهٔ جاری می‌سازد.
func buildAidInput(ctx context.Context, deps modulekit.Deps, svc *recordkit.Service, in aidRequest) (rules.MedicalAidInput, string) {
	refs, msg := parseClaimItems(in.Items)
	if msg != "" {
		return rules.MedicalAidInput{}, msg
	}
	if len(refs) == 0 {
		return rules.MedicalAidInput{}, "دست‌کم یک قلم درخواست لازم است (کد تعرفه|تعداد|مبلغ فاکتور)."
	}
	index := tariffIndex(ctx, svc)
	items := make([]rules.MedicalAidClaimItem, 0, len(refs))
	for _, ref := range refs {
		rec, ok := index[ref.Code]
		if !ok {
			return rules.MedicalAidInput{}, "کد تعرفهٔ «" + ref.Code + "» در فهرست تعرفه‌های جاری نیست؛ قلم را از فهرست انتخاب کنید."
		}
		items = append(items, rules.MedicalAidClaimItem{
			Code: ref.Code, TitleFa: rec.Title, Quantity: ref.Quantity, InvoiceAmount: ref.Amount,
			Tariff:               tariffFromRecord(rec),
			PrescriptionNo:       strings.TrimSpace(in.PrescriptionNo),
			PrescriptionApproved: in.PrescriptionApproved,
		})
	}
	year := in.PeriodYear
	if year <= 0 {
		year = jalaliYear(deps)
	}
	annualPaid := in.AnnualPaidOverride
	if annualPaid <= 0 {
		annualPaid, _ = annualPaidOf(ctx, svc, in.EmployeeID, year)
	}
	return rules.MedicalAidInput{
		EmployeeID:   strings.TrimSpace(in.EmployeeID),
		EmployeeName: employeeName(ctx, deps, in.EmployeeID),
		Category:     strings.TrimSpace(in.Category),
		PeriodYear:   year,
		Items:        items,
		AnnualPaid:   annualPaid,
	}, ""
}

// parseClaimItems رشتهٔ اقلام را به برش قلم‌ها می‌شکند. قالب هر قلم «کد|تعداد|مبلغ فاکتور» و
// جداکنندهٔ قلم‌ها «؛» یا «;» یا خط تازه است. تعداد خالی یعنی یک و مبلغ خالی یعنی صفر
// (تعرفه تنها مبنا).
func parseClaimItems(raw string) ([]claimItemRef, string) {
	out := []claimItemRef{}
	replacer := strings.NewReplacer("\n", "؛", "\r", "؛", ";", "؛")
	for _, part := range strings.Split(replacer.Replace(raw), "؛") {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		pieces := strings.Split(item, "|")
		if len(pieces) < 2 {
			return nil, "قالب قلم درخواست نادرست است؛ هر قلم باید «کد تعرفه|تعداد|مبلغ فاکتور» باشد."
		}
		code := strings.TrimSpace(pieces[0])
		if code == "" {
			return nil, "کد تعرفهٔ یک قلم خالی است."
		}
		qty := 1
		if v := strings.TrimSpace(pieces[1]); v != "" {
			n, err := strconv.Atoi(calendar.NormalizeDigits(v))
			if err != nil || n <= 0 {
				return nil, "تعداد قلم «" + code + "» باید عددی بزرگ‌تر از صفر باشد."
			}
			qty = n
		}
		amount := int64(0)
		if len(pieces) >= 3 {
			v := strings.TrimSpace(pieces[2])
			if v != "" {
				n, err := strconv.ParseInt(calendar.NormalizeDigits(v), 10, 64)
				if err != nil || n < 0 {
					return nil, "مبلغ فاکتور قلم «" + code + "» باید عددی غیرمنفی باشد."
				}
				amount = n
			}
		}
		out = append(out, claimItemRef{Code: code, Quantity: qty, Amount: amount})
	}
	return out, ""
}

// activeTariffs ردیف‌های تعرفهٔ «جاری» را برمی‌گرداند.
//
// قاعده: تنها تعرفهٔ جاری مبنای محاسبه است؛ پیش‌نویس و غیرفعال می‌مانند ولی اثر ندارند. اگر
// چند ردیف جاری با یک کد باشد، تازه‌ترین تاریخ اجرا برنده است (تاریخ‌گذاری مؤثر، بدون
// پاک‌کردن سابقهٔ تعرفهٔ سال‌های پیش).
func activeTariffs(ctx context.Context, svc *recordkit.Service) []contracts.Record {
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "medical", Kind: domain.KindTariffs, Scope: "all", Size: 500,
	})
	if err != nil {
		return nil
	}
	byCode := map[string]contracts.Record{}
	order := []string{}
	for _, rec := range page.Items {
		if rec.Status != "active" {
			continue
		}
		code := strings.TrimSpace(rec.Fields["code"])
		if code == "" {
			continue
		}
		current, ok := byCode[code]
		if !ok {
			byCode[code] = rec
			order = append(order, code)
			continue
		}
		if strings.TrimSpace(rec.Fields["effectiveFrom"]) > strings.TrimSpace(current.Fields["effectiveFrom"]) {
			byCode[code] = rec
		}
	}
	out := make([]contracts.Record, 0, len(order))
	for _, code := range order {
		out = append(out, byCode[code])
	}
	return out
}

// tariffIndex نگاشت کد تعرفه → رکورد.
func tariffIndex(ctx context.Context, svc *recordkit.Service) map[string]contracts.Record {
	out := map[string]contracts.Record{}
	for _, rec := range activeTariffs(ctx, svc) {
		out[strings.TrimSpace(rec.Fields["code"])] = rec
	}
	return out
}

// tariffFromRecord رکورد تعرفه را به شکل موتور قواعد تبدیل می‌کند (درصدها به نسبت).
func tariffFromRecord(rec contracts.Record) rules.MedicalTariffItem {
	fields := rec.Fields
	return rules.MedicalTariffItem{
		Code:                  strings.TrimSpace(fields["code"]),
		Category:              strings.TrimSpace(fields["category"]),
		TitleFa:               rec.Title,
		UnitFa:                strings.TrimSpace(fields["unitFa"]),
		Amount:                moneyOf(fields["amount"]),
		CeilingAmount:         moneyOf(fields["ceilingAmount"]),
		OrgSharePercent:       percentOf(fields["orgSharePercent"]),
		InsuranceSharePercent: percentOf(fields["insuranceSharePercent"]),
		RequirePrescription:   strings.TrimSpace(fields["requirePrescription"]) == "yes",
		RefFa:                 strings.TrimSpace(fields["refFa"]),
	}
}

// annualPaidOf جمع پرداخت‌های تأییدشده/پرداخت‌شدهٔ یک کارمند در یک سال شمسی.
func annualPaidOf(ctx context.Context, svc *recordkit.Service, employeeID string, year int) (int64, int) {
	if strings.TrimSpace(employeeID) == "" || year <= 0 {
		return 0, 0
	}
	page, err := svc.List(ctx, contracts.RecordFilter{
		Module: "medical", Kind: domain.KindClaims, EmployeeID: employeeID, Scope: "all", Size: 500,
	})
	if err != nil {
		return 0, 0
	}
	var total int64
	count := 0
	for _, rec := range page.Items {
		if rec.Status != "approved" && rec.Status != "paid" {
			continue
		}
		if intOf(rec.Fields["periodYear"]) != year {
			continue
		}
		total += moneyOf(rec.Fields["payableAmount"])
		count++
	}
	return total, count
}

// calculationLog متن خط‌به‌خط محاسبه برای سند تصویب و چاپ.
func calculationLog(res rules.MedicalAidResult) string {
	parts := make([]string, 0, len(res.Lines))
	for _, line := range res.Lines {
		text := strconv.Itoa(line.Order) + ") " + line.TitleFa + ": " + rules.IntMoneyFa(line.PayableAmount) +
			" ریال سهم سازمان"
		if line.CeilingApplied > 0 {
			text += " (کسر سقف " + rules.IntMoneyFa(line.CeilingApplied) + ")"
		}
		if !line.Payable {
			text += " — پرداخت نشد"
		}
		parts = append(parts, text)
	}
	parts = append(parts, "جمع سهم سازمان: "+rules.IntMoneyFa(res.TotalPayable)+" ریال")
	return strings.Join(parts, "\n")
}

// medicalAidParams آستانه‌های کمک هزینه از پارامترهای شرکت فعال.
func medicalAidParams(deps modulekit.Deps) rules.MedicalAidParams {
	values := map[string]float64{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "medical.aid.") {
			continue
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64); err == nil {
			values[p.Key] = v
		}
	}
	return rules.MedicalAidParamsFromValues(values)
}

// medicalAidParamViews پارامترهای کمک هزینه برای نمایش در فرم.
func medicalAidParamViews(deps modulekit.Deps) []map[string]string {
	out := []map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "medical.aid.") {
			continue
		}
		out = append(out, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa,
		})
	}
	return out
}

/* --------------------------- کمک‌های کوچک --------------------------- */

// actorOf نام کاربر جاری برای ثبت در رکورد.
func actorOf(r *http.Request) string {
	if user, ok := auth.UserFrom(r.Context()); ok {
		return user.Username
	}
	return "ناشناس"
}

// employeeName نام کارمند از پورت مشترک کارکنان (برای عنوان رکورد و سند).
func employeeName(ctx context.Context, deps modulekit.Deps, employeeID string) string {
	id := strings.TrimSpace(employeeID)
	if id == "" {
		return ""
	}
	reader, ok := deps.Ports.Employees()
	if !ok {
		return ""
	}
	employee, found, err := reader.Get(ctx, contracts.EmployeeFilter{Size: 1}, id)
	if err != nil || !found {
		return ""
	}
	return employee.FullName
}

// moneyOf مبلغ ریالی یک فیلد متنی را می‌خواند (ارقام فارسی هم پذیرفته می‌شود).
func moneyOf(value string) int64 {
	v := strings.TrimSpace(calendar.NormalizeDigits(value))
	v = strings.ReplaceAll(v, ",", "")
	v = strings.ReplaceAll(v, "٬", "")
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		if f, err2 := strconv.ParseFloat(v, 64); err2 == nil {
			return int64(f)
		}
		return 0
	}
	return n
}

// percentOf درصد یک فیلد را به نسبت تبدیل می‌کند: «۷۰» و «۰٫۷» هر دو ۰٫۷ می‌شوند.
func percentOf(value string) float64 {
	v := strings.TrimSpace(calendar.NormalizeDigits(value))
	v = strings.ReplaceAll(v, "٪", "")
	v = strings.ReplaceAll(v, "%", "")
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	if f > 1 {
		f = f / 100
	}
	return f
}

// intOf عدد صحیح یک فیلد متنی.
func intOf(value string) int {
	v := strings.TrimSpace(calendar.NormalizeDigits(value))
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return int(moneyOf(v))
}

// yesNoKey «yes/no» از یک بولی (مقدار فیلدهای select).
func yesNoKey(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// yesNoFa برچسب فارسی بولی.
func yesNoFa(v bool) string {
	if v {
		return "بله"
	}
	return "خیر"
}

// yesNoOptions گزینه‌های بله/خیر برای فیلدهای انتخابی فرم (مقدار ماشین‌خوان + برچسب فارسی).
func yesNoOptions() []map[string]string {
	return []map[string]string{
		{"key": "yes", "titleFa": "بله"},
		{"key": "no", "titleFa": "خیر"},
	}
}

// jalaliYear سال شمسی امروز (از ساعت سرور).
func jalaliYear(deps modulekit.Deps) int {
	if now, err := calendar.FromGregorian(deps.Clock()); err == nil {
		return now.Year
	}
	return 0
}

// todayFa تاریخ امروز شمسی.
func todayFa(deps modulekit.Deps) string {
	if now, err := calendar.FromGregorian(deps.Clock()); err == nil {
		return now.String()
	}
	return ""
}

// firstNonEmpty نخستین مقدار غیرخالی.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
