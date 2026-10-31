// زیرساخت «documents» در پلتفرم سرور — «handler».

package documents

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hrms/internal/contracts"
	"hrms/internal/platform/auth"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/qr"
	"hrms/internal/rules"
)

// Handler مسیر سند چاپی را می‌سازد: /api/v1/documents/{kind}/{id}
//
// دسترسی روی همان ماژولی سنجیده می‌شود که مالک دادهٔ سند است (قرارداد → ماژول
// قراردادها، حکم حقوق → احکام حقوق، گواهی اشتغال → کارکنان)، پس چاپ سند راه فراری
// از ماتریس دسترسی نیست. هر چاپ در لاگ حسابرسی ثبت می‌شود.
func Handler(deps modulekit.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kind := Kind(r.PathValue("kind"))
		moduleID, known := ModuleOf(kind)
		if !known {
			httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
			return
		}
		user, ok := auth.UserFrom(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, contracts.CodeUnauthorized, nil)
			return
		}
		if !deps.Policy.Allows(user.Role, moduleID, "read") {
			deps.Audit.Append(user.Username, moduleID, "document-denied", r.PathValue("id"), "تلاش برای چاپ سند بدون دسترسی")
			httpx.WriteError(w, http.StatusForbidden, contracts.CodeForbidden, nil)
			return
		}

		data, err := buildData(r.Context(), deps, kind, r.PathValue("id"))
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, err.Error(), nil)
			return
		}

		deps.Audit.Append(user.Username, moduleID, "document-print", data.Serial, TitleFa(kind)+" — "+data.Employee.FullName)
		attachVerifyQR(deps, r, kind, data.Serial, &data)
		writeDoc(w, data)
	}
}

// attachVerifyQR نشانی تأیید سند و QR کد آن را به سند میچسباند.
//
// چرا: سند کاغذی بدون راه بررسی، قابل جعل است؛ QR کد نشانی صفحهٔ تأیید همین سرور را
// می‌گیرد که با امضای HMAC سنجیده میشود. ساخت QR کد سمت سرور است تا اگر کتابخانه
// خطا داد، سند بدون کد چاپ شود (نه اینکه چاپ سند شکست بخورد).
func attachVerifyQR(deps modulekit.Deps, r *http.Request, kind Kind, serial string, data *Data) {
	base := "http://" + r.Host
	if r.TLS != nil {
		base = "https://" + r.Host
	}
	token := qr.Sign(deps.VerifyKey, "document", string(kind), data.Employee.ID, serial)
	data.VerifyURL = qr.VerifyLink(base, "employee", data.Employee.ID, map[string]string{
		"kind": string(kind), "serial": serial, "t": token,
	})
	svg, err := qr.SVG(data.VerifyURL, 120)
	if err != nil {
		return
	}
	// محتوای SVG ساختهٔ همین سرور است (نه ورودی کاربر)، پس درج بیپردهاش امن است.
	data.QRCodeSVG = template.HTML(svg)
}

// writeDoc سند را به‌صورت HTML چاپی می‌نویسد (با سرصفحه‌های امنیتی مناسب).
func writeDoc(w http.ResponseWriter, data Data) {
	html, err := Render(data)
	if err != nil {
		httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

// PreviewHandler مسیر پیش‌نمایش قالب: /api/v1/documents/preview/{kind}
//
// پیش‌نمایش با دادهٔ نمونه (پیش‌فرض EMP-001 و قرارداد فعالش) ساخته می‌شود تا طراح قالب
// اثر جای‌نگهدارها و متن را ببیند، بدون نیاز به انتخاب یک رکورد واقعی.
func PreviewHandler(deps modulekit.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		kind := Kind(r.PathValue("kind"))
		moduleID, known := ModuleOf(kind)
		if !known {
			httpx.WriteError(w, http.StatusNotFound, contracts.CodeNotFound, nil)
			return
		}
		user, ok := auth.UserFrom(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusUnauthorized, contracts.CodeUnauthorized, nil)
			return
		}
		if !deps.Policy.Allows(user.Role, moduleID, "read") {
			httpx.WriteError(w, http.StatusForbidden, contracts.CodeForbidden, nil)
			return
		}

		sample := strings.TrimSpace(r.URL.Query().Get("employee"))
		if sample == "" {
			sample = "EMP-001"
		}
		target := sample
		if kind == KindContract {
			providers, ok := deps.Ports.ContractDocuments()
			if !ok {
				httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "دادهٔ قرارداد در این استقرار در دسترس نیست", nil)
				return
			}
			c, found, err := providers.ActiveContractOf(r.Context(), sample)
			if err != nil {
				httpx.WriteErrorMsg(w, http.StatusInternalServerError, contracts.CodeInternal, err.Error(), nil)
				return
			}
			if !found {
				httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, "قرارداد فعالی برای کارمند نمونه یافت نشد", nil)
				return
			}
			target = c.ID
		}

		data, err := buildData(r.Context(), deps, kind, target)
		if err != nil {
			httpx.WriteErrorMsg(w, http.StatusNotFound, contracts.CodeNotFound, err.Error(), nil)
			return
		}
		w.Header().Set("X-Preview", "1")
		writeDoc(w, data)
	}
}

// buildData دادهٔ سند را از پورت‌های قراردادی و استقرار فعال جمع می‌کند.
func buildData(ctx context.Context, deps modulekit.Deps, kind Kind, id string) (Data, error) {
	reader, ok := deps.Ports.Employees()
	if !ok {
		return Data{}, fmt.Errorf("دادهٔ کارکنان در این استقرار در دسترس نیست")
	}
	providers, ok := deps.Ports.ContractDocuments()
	if !ok {
		return Data{}, fmt.Errorf("دادهٔ قرارداد در این استقرار در دسترس نیست")
	}

	today, err := calendar.FromGregorian(nowOf(deps))
	if err != nil {
		return Data{}, err
	}

	var (
		employee    contracts.Employee
		employeeID  string
		contractRef *ContractInfo
		contractID  string
	)

	if kind == KindContract {
		c, found, err := providers.ContractByID(ctx, id)
		if err != nil {
			return Data{}, err
		}
		if !found {
			return Data{}, fmt.Errorf("قرارداد درخواستی یافت نشد")
		}
		contractID = c.ID
		employeeID = c.EmployeeID
		contractRef = &ContractInfo{
			ID: c.ID, KindFa: c.KindFa, StatusFa: c.StatusFa,
			StartDate: c.StartDate, EndDate: c.EndDate,
			MonthlySalary: c.MonthlySalary, ServiceDays: c.ServiceDays, ServiceYears: c.ServiceDays / 365,
		}
	} else {
		employeeID = id
	}

	emp, found, err := reader.Get(ctx, contracts.EmployeeFilter{Scope: "all"}, employeeID)
	if err != nil {
		return Data{}, err
	}
	if !found {
		return Data{}, fmt.Errorf("کارمند درخواستی یافت نشد")
	}
	employee = emp

	if contractRef == nil && (kind == KindSalaryDecree || kind == KindEmploymentCertificate) {
		if c, ok, err := providers.ActiveContractOf(ctx, employeeID); err != nil {
			return Data{}, err
		} else if ok {
			contractID = c.ID
			contractRef = &ContractInfo{
				ID: c.ID, KindFa: c.KindFa, StatusFa: c.StatusFa,
				StartDate: c.StartDate, EndDate: c.EndDate,
				MonthlySalary: c.MonthlySalary, ServiceDays: c.ServiceDays, ServiceYears: c.ServiceDays / 365,
			}
		}
	}

	data := Data{
		Kind:     kind,
		Serial:   serialOf(kind, employeeID, contractID, today),
		IssuedAt: today.String(),
		Company: Company{
			TitleFa:           deps.Org.TitleFa,
			HoldingFa:         deps.Org.HoldingFa,
			ShortCode:         deps.Org.ShortCode,
			CurrencyFa:        deps.Org.CurrencyFa,
			WorkWeekFa:        deps.Org.WorkWeekFa,
			FiscalYearStartFa: deps.Org.FiscalYearStartFa,
			LogoURL:           logoURL(deps.Org.LogoFile),
			BrandColor:        deps.Org.BrandColor,
		},
		Employee: Employee{
			ID:             employee.ID,
			FullName:       employee.FullName,
			NationalID:     employee.NationalID,
			DepartmentName: employee.DepartmentName,
			Position:       employee.Position,
			HireDate:       employee.HireDate,
			ContractType:   employee.ContractType,
			BaseSalary:     employee.BaseSalary,
		},
		Contract: contractRef,
	}

	switch kind {
	case KindContract:
		data.Params = payrollParams(deps)
		data.Structure = wageStructure(ctx, deps, employee)
		data.BodyFa = contractClauses(deps, employee, contractRef)
		data.NoteFa = "این قرارداد در دو نسخه تنظیم و پس از امضای طرفین، هر نسخه حکم واحد خود را دارد."
	case KindSalaryDecree:
		data.Params = payrollParams(deps)
		data.Structure = wageStructure(ctx, deps, employee)
		data.BodyFa = []string{
			"مبلغ پایهٔ ماهانهٔ نامبرده بر پایهٔ آخرین حکم و مصوبات شرکت تعیین شده است.",
			"مبانی محاسبهٔ سامانه (ساعت کار موظفی، ضریب اضافه‌کاری و روز پرداخت) در جدول بالا آمده و مبنای اجرای دوره‌های حقوق بعدی است.",
			"هر تغییر در مصوبات، حکم جدید با تاریخ اجرای مشخص تولید می‌کند؛ احکام گذشته بازنویسی نمی‌شوند.",
		}
		data.NoteFa = "پرداخت‌های متغیر مانند کارانه و اضافه‌کاری در دورهٔ حقوق همان ماه محاسبه و پرداخت می‌شود."
	case KindPayslip:
		data.Params = payrollParams(deps)
		info := computePayslip(ctx, deps, employee, today)
		data.Payslip = &info
		data.BodyFa = []string{
			fmt.Sprintf("فیش حقوقی دورهٔ %s برای %s در واحد %s صادر شده است.", info.Period, employee.FullName, employee.DepartmentName),
			fmt.Sprintf("جمع ناخالص %s، بیمهٔ سهم کارمند %s و مالیات %s است.", money(info.Gross), money(info.InsuranceEmployee), money(info.Tax)),
			"مبالغ بر پایهٔ پارامترهای حقوق و اقلام تعریف‌شدهٔ سامانه محاسبه شده و در سرور مرجع است.",
		}
		data.NoteFa = "این فیش از سامانهٔ جامع منابع انسانی صادر شده و در صورت مغایرت، پرداخت سامانه مرجع است."
	case KindEmploymentCertificate:
		body := fmt.Sprintf("گواهی می‌شود %s با کد پرسنلی %s از تاریخ %s در شرکت %s در واحد %s به عنوان %s اشتغال دارد.",
			employee.FullName, employee.ID, employee.HireDate, deps.Org.TitleFa, employee.DepartmentName, employee.Position)
		if contractRef != nil && contractRef.KindFa != "" {
			body += fmt.Sprintf(" نوع همکاری ایشان «%s» است.", contractRef.KindFa)
		}
		data.BodyFa = []string{
			body,
			"این گواهی بنا به درخواست نامبرده و برای ارائه به مراجع ذی‌ربط صادر شده است.",
			"اعتبار این سند با کد رهگیری درج‌شده در پایین صفحه در سامانه قابل بررسی است.",
		}
	}

	// اگر برای این نوع سند، قالب طراحى‌شدهٔ فعال وجود داشته باشد، متن آن با دادهٔ واقعی
	// جایگزین می‌شود؛ در غیر این صورت متن پیش‌فرض سامانه چاپ می‌شود.
	if tplProvider, ok := deps.Ports.DocumentTemplates(); ok {
		if tpl, found, err := tplProvider.ActiveTemplate(ctx, string(kind)); err == nil && found {
			if body := ApplyTemplate(tpl.Body, PlaceholderVars(data)); len(body) > 0 {
				data.BodyFa = body
				data.CustomBody = true
			}
			if strings.TrimSpace(tpl.Footer) != "" {
				data.NoteFa = strings.TrimSpace(tpl.Footer)
			}
		}
	}

	return data, nil
}

// computePayslip فیش حقوقی نمونهٔ کارمند را از پارامترهای فعال محاسبه می‌کند.
// عددها از موتور قواعد می‌آید، نه از UI؛ این تابع تکرار قاعده نمی‌کند. تفکیک اقلام هم
// با همان موتور مشترک (`rules.ComputeElements`) ساخته می‌شود تا فیش چاپی و شبیه‌ساز
// یک عدد نشان دهند.
func computePayslip(ctx context.Context, deps modulekit.Deps, employee contracts.Employee, today calendar.Date) PayslipInfo {
	params := rules.DefaultPayrollParams()
	structure, sourceFa := wageDocOf(ctx, deps, employee)
	base := employee.BaseSalary
	if structure.BaseSalary > 0 {
		base = structure.BaseSalary
	}
	input := rules.PayrollInput{BaseSalary: base}
	result := rules.ComputePayroll(input, params)
	info := PayslipInfo{
		Period:            fmt.Sprintf("%04d/%02d", today.Year, today.Month),
		Gross:             result.Gross,
		InsuranceEmployee: result.InsuranceEmployee,
		InsuranceEmployer: result.InsuranceEmployer,
		Tax:               result.Tax,
		Net:               result.Net,
		WageSourceFa:      sourceFa,
	}
	if provider, ok := deps.Ports.PayElements(); ok {
		env, wages := elementEnvFor(ctx, deps, params, input, result, structure)
		info.MazdBase = rules.RoundHalfUp(wages.Total())
		for _, line := range rules.ComputeElements(provider.PayElements(ctx), env) {
			info.Elements = append(info.Elements, PayslipElement{
				TitleFa:    line.TitleFa,
				CategoryFa: categoryFa(line.Category),
				AmountFa:   money(rules.RoundHalfUp(line.Amount)),
				DetailFa:   line.DetailFa,
			})
		}
	}
	return info
}

// wageStructure «اجزای مزد مبنا و فوق‌العاده‌ها» را برای حکم کارگزینی و قرارداد کار می‌سازد.
//
// ساختار دو بخشی است و همین خوانایی سند را می‌سازد:
//  1. اجزای مزد مبنا — همان اجزایی که مزد مبنا از جمع آن‌ها ساخته می‌شود (با سطر جمع).
//  2. فوق‌العاده‌ها و پرداخت‌های دوره — اقلامی که روی مزد مبنا سوار می‌شوند یا به کارکرد
//     دوره وابسته‌اند (اقلیمی درصدی، نوبت‌کاری، اضافه‌کاری، حق مسکن، بن، ...).
//
// منبع اجزا، سند کارگزینی خودِ کارمند است (حکم ابلاغ‌شده و در نبود آن قرارداد جاری)؛ پس
// سندی که به او ابلاغ می‌شود با فیش حقوقش یک عدد می‌دهد. محاسبه با همان موتور مشترک
// شبیه‌ساز انجام می‌گیرد و هیچ عددی در این فایل سخت نشده است.
func wageStructure(ctx context.Context, deps modulekit.Deps, employee contracts.Employee) []StructureRow {
	provider, ok := deps.Ports.PayElements()
	if !ok {
		return nil
	}
	structure, sourceFa := wageDocOf(ctx, deps, employee)
	base := employee.BaseSalary
	if structure.BaseSalary > 0 {
		base = structure.BaseSalary
	}
	params := rules.DefaultPayrollParams()
	input := rules.PayrollInput{BaseSalary: base}
	result := rules.ComputePayroll(input, params)
	env, wages := elementEnvFor(ctx, deps, params, input, result, structure)

	rows := []StructureRow{}
	inBase := map[string]bool{}
	for _, part := range wages.Breakdown() {
		inBase[part.Code] = true
		rows = append(rows, StructureRow{
			TitleFa:  part.TitleFa,
			KindFa:   part.KindFa,
			DetailFa: "جزء ثابت مزد مبنا",
			AmountFa: money(rules.RoundHalfUp(part.Amount)),
			GroupFa:  "اجزای مزد مبنا",
		})
	}
	rows = append(rows, StructureRow{
		TitleFa: "مزد مبنا", KindFa: "جمع اجزای ثابت",
		DetailFa: "مبنای اضافه‌کاری، شب‌کاری، نوبت‌کاری و فوق‌العاده‌های اقلیمی",
		AmountFa: money(rules.RoundHalfUp(wages.Total())),
		GroupFa:  "اجزای مزد مبنا", Total: true,
	})
	if sourceFa != "" {
		rows[0].DetailFa = rows[0].DetailFa + " — " + sourceFa
	}
	for _, line := range rules.ComputeElements(provider.PayElements(ctx), env) {
		if strings.TrimSpace(line.Category) != "earning" || inBase[line.Code] {
			continue
		}
		amount := "—"
		if line.Resolved {
			amount = money(rules.RoundHalfUp(line.Amount))
		}
		rows = append(rows, StructureRow{
			TitleFa:  line.TitleFa,
			KindFa:   categoryFa(line.Category),
			DetailFa: line.DetailFa,
			AmountFa: amount,
			GroupFa:  "فوق‌العاده‌ها و پرداخت‌های دوره",
		})
	}
	return rows
}

// wageDoc اجزای مزد یک کارمند در سند کارگزینی، همراه با منبع آن.
type wageDoc struct {
	BaseSalary int64
	contracts.WageStructure
	SourceFa string
}

// wageDocOf اجزای مزد کارمند را می‌خواند: ابتدا حکم ابلاغ‌شده و در نبود آن قرارداد جاری.
// هیچ‌کدام نبود، صفر برمی‌گردد و سند با پایهٔ پرونده و پارامترهای شرکت ساخته می‌شود.
func wageDocOf(ctx context.Context, deps modulekit.Deps, employee contracts.Employee) (wageDoc, string) {
	if reader, ok := deps.Ports.SalaryDecrees(); ok {
		if d, found, err := reader.NotifiedDecreeOf(ctx, employee.ID); err == nil && found {
			return wageDoc{BaseSalary: d.BaseSalary, WageStructure: d.WageStructure}, "از حکم کارگزینی " + d.ID
		}
	}
	if reader, ok := deps.Ports.ContractDocuments(); ok {
		if c, found, err := reader.ActiveContractOf(ctx, employee.ID); err == nil && found {
			return wageDoc{BaseSalary: c.MonthlySalary, WageStructure: c.WageStructure}, "از قرارداد کار " + c.ID
		}
	}
	return wageDoc{}, ""
}

// elementEnvFor محیط محاسبهٔ اقلام یک کارمند را می‌سازد: پارامترهای پیشفرض موتور قواعد،
// پارامترهای شرکت فعال، مقادیر پارامترهای حقوق از ماژول حقوق، اجزای مزد کارمند از سند
// کارگزینی، و ضرایب اقلیمی همان کارمند. مزد شغل همان پایهٔ سند است تا حکم، قرارداد و فیش
// با پروندهٔ او همخوان بمانند.
func elementEnvFor(ctx context.Context, deps modulekit.Deps, params rules.PayrollParams, input rules.PayrollInput, result rules.PayrollResult, structure wageDoc) (rules.ElementEnv, rules.WageComponents) {
	vars := rules.BaseElementVars(input, result)
	for k, v := range rules.ParamsVars(params) {
		vars[k] = v
	}
	companyPercent := companyWageVars(ctx, deps)
	for k, v := range companyPercent {
		vars[k] = v
	}
	// ضرایب اقلیمی خودِ کارمند بر پارامتر شرکت مقدم است.
	for k, v := range structure.Climate(companyPercent).Vars() {
		vars[k] = v
	}
	wages := structure.Components(input.BaseSalary, companyPercent)
	for k, v := range wages.Vars() {
		vars[k] = v
	}
	return rules.ElementEnv{
		Bases:     rules.BaseElementBases(input, params, result, 0, wages),
		Vars:      vars,
		Canonical: rules.CanonicalElementAmounts(result),
	}, wages
}

// companyWageVars ضرایب درصدی سازمانی (ثابت‌های شغلی و اقلیمی) را از پارامترهای شرکت و
// رکوردهای «پارامتر حقوق» می‌خواند. نبود ماژول حقوق، محاسبه را متوقف نمی‌کند.
func companyWageVars(ctx context.Context, deps modulekit.Deps) map[string]float64 {
	out := map[string]float64{}
	for _, p := range deps.Org.Params {
		if f, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64); err == nil {
			out[p.Key] = f
		}
	}
	if provider, ok := deps.Ports.PayElements(); ok {
		for k, v := range provider.PayParamValues(ctx) {
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				out[k] = f
			}
		}
	}
	return out
}

// categoryFa برچسب فارسی دستهٔ قلم حقوق (مزایا / کسور).
func categoryFa(v string) string {
	switch strings.TrimSpace(v) {
	case "earning", "مزایا":
		return "مزایا"
	case "deduction", "کسور":
		return "کسور"
	default:
		return v
	}
}

// money عدد ریالی را با جداکنندهٔ هزارگان برمی‌گرداند.
func money(v int64) string {
	s := strconv.FormatInt(v, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// logoURL مسیر نشان شرکت را به آدرس قابل مصرف در سند تبدیل می‌کند.
// سند روی همان مبدأ رابط وب سرو می‌شود، پس مسیر نسبی پروفایل شرکت کافی است؛ خالی
// بودن آن یعنی سند بدون تصویر نشان چاپ می‌شود.
func logoURL(file string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	return "/" + strings.TrimPrefix(file, "/")
}

func serialOf(kind Kind, employeeID, contractID string, today calendar.Date) string {
	switch kind {
	case KindContract:
		return contractID
	case KindSalaryDecree:
		return fmt.Sprintf("DEC-%d-%s", today.Year, strings.TrimPrefix(employeeID, "EMP-"))
	case KindPayslip:
		return fmt.Sprintf("PAY-%d-%s", today.Year, strings.TrimPrefix(employeeID, "EMP-"))
	default:
		return fmt.Sprintf("CRT-%d-%s", today.Year, strings.TrimPrefix(employeeID, "EMP-"))
	}
}

// payrollParams پارامترهای گروه «حقوق و دستمزد» از استقرار شرکت فعال؛ هیچ عددی اینجا
// ساخته نمی‌شود، بلکه همان چیزی است که سرور برای محاسبه به کار می‌برد.
func payrollParams(deps modulekit.Deps) []Param {
	out := []Param{}
	for _, p := range deps.Org.Params {
		if p.GroupFa != "حقوق و دستمزد" {
			continue
		}
		out = append(out, Param{Key: p.Key, TitleFa: p.TitleFa, Value: p.Value, UnitFa: p.UnitFa, GroupFa: p.GroupFa})
	}
	return out
}

func contractClauses(deps modulekit.Deps, e contracts.Employee, c *ContractInfo) []string {
	hours, _ := deps.Org.Param("payroll.monthlyHours")
	leave, _ := deps.Org.Param("leave.annualDays")
	work := deps.Org.WorkWeekFa
	clauses := []string{
		fmt.Sprintf("طرف دوم از تاریخ شروع در واحد %s به عنوان %s انجام وظیفه می‌کند.", e.DepartmentName, e.Position),
		fmt.Sprintf("هفتهٔ کاری شرکت: %s و ساعت کار موظفی ماهانه %s ساعت است.", work, hours),
		fmt.Sprintf("مرخصی استحقاقی سالانه %s روز و مرخصی استعلاجی طبق قانون کار و ضوابط شرکت است.", leave),
		"مبلغ پایهٔ ماهانه و سایر مزایا مطابق حکم حقوق صادرشده از واحد جبران خدمات پرداخت می‌شود.",
		"رعایت آیین‌نامه‌های داخلی، مقررات حفاظت فنی و محرمانگی اطلاعات شرکت الزامی است.",
		"این قرارداد تابع قانون کار و مقررات تأمین اجتماعی است و اختلاف‌ها از راه مراجع قانونی پیگیری می‌شود.",
	}
	return clauses
}

func nowOf(deps modulekit.Deps) time.Time {
	if deps.Clock == nil {
		return time.Now()
	}
	return deps.Clock()
}
