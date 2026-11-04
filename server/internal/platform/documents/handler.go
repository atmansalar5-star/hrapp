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
		// basisRec تصمیم «مدرک ملاک حقوق» برای ساخت فرم چاپی همان تصمیم.
		basisRec *contracts.Record
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
	} else if kind == KindDegreeBasis {
		// فرم «مدرک ملاک حقوق» با شناسهٔ خودِ رکورد تصمیم باز می‌شود (نه کد کارمند)؛
		// کارمند از همان رکورد خوانده می‌شود تا سند به تصمیم شورا گره بخورد.
		basis, found, err := degreeBasisRecord(ctx, deps, id)
		if err != nil {
			return Data{}, err
		}
		if !found {
			return Data{}, fmt.Errorf("تصمیم «مدرک ملاک حقوق» با این شناسه یافت نشد")
		}
		employeeID = basis.EmployeeID
		basisRec = &basis
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
			// اطلاعات پایهٔ پرونده: همان مقادیری که فرم‌های قرارداد و حکم نشان می‌دهند.
			Mobile:     employee.Mobile,
			HomePhone:  employee.HomePhone,
			Phone:      employee.Phone,
			PostalCode: employee.PostalCode,
			Address:    employee.Address,
			Email:      employee.Email,
		},
		Contract: contractRef,
		// پارامترهای سازمانی برای جای‌نگهدارهای متن (بندهای قرارداد).
		OrgParams: allParams(deps),
		// چیدمان پیش‌فرض همین سند؛ سپس سفارشی‌سازی پنل تنظیمات روی آن می‌نشیند.
		Layout: DefaultLayout(kind),
	}
	// مشخصات تحصیلی از پروندهٔ بایگانی خوانده می‌شود (بالاترین مدرک و مدرک ملاک حقوق).
	applyEducation(ctx, deps, employeeID, &data)
	// بلوک‌هایی که مدیر سامانه در پنل پارامتریک روشن/خاموش یا بازعنوان کرده است.
	applyLayout(ctx, deps, kind, &data)

	switch kind {
	case KindContract:
		data.Params = payrollParams(deps)
		data.Structure = wageStructure(ctx, deps, employee)
		addBaseInfoForm(&data)
	case KindSalaryDecree:
		data.Params = payrollParams(deps)
		data.Structure = wageStructure(ctx, deps, employee)
		addBaseInfoForm(&data)
	case KindPayslip:
		data.Params = payrollParams(deps)
		info := computePayslip(ctx, deps, employee, today)
		data.Payslip = &info
	case KindEmploymentCertificate:
		addBaseInfoForm(&data)
	case KindDegreeBasis:
		buildDegreeBasisForm(ctx, deps, &data, basisRec)
	}

	// متن پیش‌فرض قالب‌محور سند (پاراگراف متن و یادداشت پای سند) با دادهٔ واقعی پر
	// می‌شود؛ سپس قالب متن فعال و در پایان متن پنل پارامتریک رویش می‌نشیند.
	applyDefaultText(&data)

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

	// آخرین حرف را مدیر سامانه می‌زند: متن پاراگراف/یادداشتی که در پنل پارامتریک
	// برای همین سند نوشته شده است، بر قالب پیش‌فرض و قالب متن ساخته‌شده مقدم است.
	applyPanelTexts(&data)

	return data, nil
}

// applyDefaultText متن پیش‌فرض قالب‌محور سند را با دادهٔ واقعی روی برگه می‌نشاند.
//
// قاعده: اگر متن سند از قبل تعیین شده باشد (بندهای محاسبه‌شدهٔ قرارداد) دست نمی‌زنیم؛
// وگرنه پاراگراف متن و یادداشت پای سند از قالب پیش‌فرض سامانه ساخته می‌شوند. همین
// قالب در پنل پارامتریک هم نشان داده می‌شود، پس متن پنل و متن چاپی هم‌خوان می‌مانند.
func applyDefaultText(data *Data) {
	texts := DefaultText(data.Kind)
	if len(texts) == 0 {
		return
	}
	vars := PlaceholderVars(*data)
	if len(data.BodyFa) == 0 {
		if lines := ApplyTemplate(texts["body"], vars); len(lines) > 0 {
			data.BodyFa = lines
		}
	}
	if strings.TrimSpace(data.NoteFa) == "" {
		if lines := ApplyTemplate(texts["note"], vars); len(lines) > 0 {
			data.NoteFa = strings.Join(lines, " ")
		}
	}
}

// applyPanelTexts متن پاراگراف/یادداشت نوشته‌شده در پنل پارامتریک را جای متن سند
// می‌گذارد.
//
// چرا بالاترین اولویت: این متن آخرین تصمیم صریح مدیر سامانه برای همین سند است
// (مثلاً ویرایش متن گواهی اشتغال)، پس بر قالب متن ساخته‌شده هم مقدم است. جای‌نگهدارها
// با همان دادهٔ واقعی پر می‌شوند تا متن پنل هم مثل قالب متن کار کند.
func applyPanelTexts(data *Data) {
	vars := PlaceholderVars(*data)
	if custom := data.Layout.Text("body"); custom != "" {
		if lines := ApplyTemplate(custom, vars); len(lines) > 0 {
			data.BodyFa = lines
			data.CustomBody = true
		}
	}
	if custom := data.Layout.Text("note"); custom != "" {
		if lines := ApplyTemplate(custom, vars); len(lines) > 0 {
			data.NoteFa = strings.Join(lines, " ")
		}
	}
}

// applyLayout سفارشی‌سازی «بلوک‌های سند چاپی» را از پنل پارامتریک (تنظیمات) می‌خواند.
//
// قاعده: بلوک‌های ممکن همان فهرست پشتیبانی‌شدهٔ همین سند است؛ اگر پارامتر «جاری» برای
// این نوع سند ثبت شده باشد، فقط بلوک‌های فهرست‌شده روشن می‌مانند و عنوان‌های سفارشی
// جای پیش‌فرض را می‌گیرند. پارامتر غیرفعال یا نبودش یعنی چیدمان پیش‌فرض سامانه.
func applyLayout(ctx context.Context, deps modulekit.Deps, kind Kind, data *Data) {
	port, ok := deps.Ports.RecordList("settings")
	if !ok {
		return
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: "settings", Kind: "formParams", Scope: "all", Page: 1, Size: 200,
	})
	if err != nil {
		return
	}
	supported := map[string]bool{}
	for _, key := range KindBlocks(kind) {
		supported[key] = true
	}
	for _, rec := range page.Items {
		if strings.TrimSpace(rec.Fields["scope"]) != "document" {
			continue
		}
		if strings.TrimSpace(rec.Fields["target"]) != string(kind) {
			continue
		}
		if strings.TrimSpace(rec.Status) == "inactive" {
			continue // پارامتر غیرفعال: چیدمان پیش‌فرض
		}
		if raw := strings.TrimSpace(rec.Fields["visibleFields"]); raw != "" {
			visible := map[string]bool{}
			for _, key := range splitCSV(raw) {
				if supported[key] {
					visible[key] = true
				}
			}
			// اگر فهرست پارامتر هیچ بلوک معتبری نداشت، چیدمان پیش‌فرض می‌ماند تا سند
			// ناخواسته خالی چاپ نشود.
			if len(visible) > 0 {
				data.Layout.Visible = visible
			}
		}
		if labels := strings.TrimSpace(rec.Fields["labels"]); labels != "" {
			overrides := map[string]string{}
			for _, pair := range splitCSV(labels) {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) != 2 {
					continue
				}
				key := strings.TrimSpace(parts[0])
				title := strings.TrimSpace(parts[1])
				if supported[key] && title != "" {
					overrides[key] = title
				}
			}
			if len(overrides) > 0 {
				data.Layout.Titles = overrides
			}
		}
		// متن‌های نوشتنی سند: پاراگراف متن و یادداشت پای برگه. کلید فیلد پارامتر
		// «<کلید جای متن>+Text» است (bodyText، noteText).
		texts := map[string]string{}
		for _, slot := range KindTextSlots(kind) {
			if value := strings.TrimSpace(rec.Fields[slot.Key+"Text"]); value != "" {
				texts[slot.Key] = value
			}
		}
		if len(texts) > 0 {
			data.Layout.Texts = texts
		}
		return
	}
}

// splitCSV رشتهٔ جداشده با کاما/ویرگول فارسی را به فهرست تمیز تبدیل می‌کند.
func splitCSV(value string) []string {
	replacer := strings.NewReplacer("،", ",", "؛", ",", "|", ",")
	out := []string{}
	for _, part := range strings.Split(replacer.Replace(value), ",") {
		if item := strings.TrimSpace(part); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// applyEducation مشخصات تحصیلی کارمند را از پروندهٔ بایگانی روی سند می‌نشاند.
//
// چرا از پورت رکوردها و نه کپی در قرارداد/حکم: «آخرین مدرک» یک تصمیم پروندهٔ تحصیلی
// است (تأیید مدارک، شورای تعدیل مدرک، ایثارگران) و اگر روی سند کپی شود، با تغییر پرونده
// از آن جدا می‌ماند. بالاترین مقطع مبناست و در تساوی، مدرک با تاریخ فراغت تازه‌تر.
func applyEducation(ctx context.Context, deps modulekit.Deps, employeeID string, data *Data) {
	educations := educationRecords(ctx, deps, employeeID)
	best := contracts.Record{}
	bestOrder := -1
	for _, rec := range educations {
		// فقط مدارک تأییدشده (یا بایگانی‌شده) سند را تزئین نمی‌کنند؛ پیش‌نویس روی
		// گواهی رسمی نمی‌آید.
		if status := strings.TrimSpace(rec.Status); status != "" && status != "verified" && status != "archived" {
			continue
		}
		order := contracts.DegreeLevelOrder(rec.Fields["level"])
		if order > bestOrder || (order == bestOrder && rec.Fields["graduationDate"] > best.Fields["graduationDate"]) {
			best, bestOrder = rec, order
		}
	}
	if bestOrder > 0 {
		data.Employee.EducationLevelFa = contracts.DegreeLevelTitleFa(best.Fields["level"])
		data.Employee.EducationField = strings.TrimSpace(best.Fields["field"])
		data.Employee.EducationOrient = strings.TrimSpace(best.Fields["orientation"])
		data.Employee.EducationUniver = strings.TrimSpace(best.Fields["university"])
		data.Employee.EducationDate = strings.TrimSpace(best.Fields["graduationDate"])
		data.Employee.EducationText = joinNonEmpty(" — ", data.Employee.EducationLevelFa, data.Employee.EducationField)
		data.Employee.HasEducationInfo = true
	}

	// مدرک ملاک حقوق: اگر شورای تعدیل مدرک تصمیم فعالی ثبت کرده باشد.
	if base := activeDegreeBasis(ctx, deps, employeeID); base != nil {
		levelFa := contracts.DegreeLevelTitleFa(base.Fields["level"])
		if upgraded := contracts.DegreeLevelTitleFa(base.Fields["upgradedLevel"]); strings.TrimSpace(base.Fields["upgradedLevel"]) != "" {
			levelFa = upgraded
		}
		data.Employee.DegreeBasisFa = joinNonEmpty(" — ", levelFa, strings.TrimSpace(base.Fields["field"]))
		data.Employee.HasEducationInfo = true
	}
}

// activeDegreeBasis مدرک ملاک جاری حقوق کارمند را برمی‌گرداند (یا nil).
func activeDegreeBasis(ctx context.Context, deps modulekit.Deps, employeeID string) *contracts.Record {
	port, ok := deps.Ports.RecordList("archive")
	if !ok {
		return nil
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: "archive", Kind: "degreeBases", Status: "active", EmployeeID: employeeID, Scope: "all", Size: 10,
	})
	if err != nil {
		return nil
	}
	for i := range page.Items {
		rec := page.Items[i]
		if rec.EmployeeID != employeeID || strings.TrimSpace(rec.Fields["level"]) == "" {
			continue
		}
		return &page.Items[i]
	}
	return nil
}

// addBaseInfoForm بلوک «مشخصات پایهٔ کارمند» را روی سند می‌نشاند: شماره‌های تماس،
// کد پستی، نشانی و مدرک تحصیلی. هر ردیف فقط وقتی می‌آید که مقدارش در پرونده ثبت شده باشد،
// پس سندی با پروندهٔ ناقص ردیف خالی نشان نمی‌دهد.
func addBaseInfoForm(data *Data) {
	rows := []FormField{}
	add := func(label, value string) {
		if v := strings.TrimSpace(value); v != "" {
			rows = append(rows, FormField{LabelFa: label, ValueFa: v})
		}
	}
	add("شمارهٔ همراه", data.Employee.Mobile)
	add("تلفن منزل", data.Employee.HomePhone)
	add("تلفن تماس دیگر", data.Employee.Phone)
	add("کد پستی", data.Employee.PostalCode)
	add("نشانی محل سکونت", data.Employee.Address)
	add("مدرک تحصیلی", data.Employee.EducationText)
	add("مدرک ملاک حقوق", data.Employee.DegreeBasisFa)
	if len(rows) == 0 {
		return
	}
	data.BaseInfoTitleFa = "مشخصات پایهٔ کارمند"
	data.BaseInfoFields = append(data.BaseInfoFields, rows...)
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
	case KindDegreeBasis:
		return fmt.Sprintf("DBAS-%d-%s", today.Year, strings.TrimPrefix(employeeID, "EMP-"))
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

// allParams همهٔ پارامترهای استقرار شرکت فعال (هر گروه) برای جای‌نگهدارهای متن.
//
// چرا جدا از payrollParams: جدول «مبنای محاسبهٔ سامانه» فقط پارامترهای حقوق و دستمزد را
// نشان می‌دهد، ولی متن قرارداد به مرخصی سالانه هم نیاز دارد و نباید با اضافه‌شدن یک
// ردیف، جدول سند شلوغ شود.
func allParams(deps modulekit.Deps) []Param {
	out := make([]Param, 0, len(deps.Org.Params))
	for _, p := range deps.Org.Params {
		out = append(out, Param{Key: p.Key, TitleFa: p.TitleFa, Value: p.Value, UnitFa: p.UnitFa, GroupFa: p.GroupFa})
	}
	return out
}

func nowOf(deps modulekit.Deps) time.Time {
	if deps.Clock == nil {
		return time.Now()
	}
	return deps.Clock()
}
