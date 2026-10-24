// پیاده‌سازی ماژول «payroll» — «service_bonus».

package payroll

import (
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

// registerServiceBonusEndpoints مسیرهای محاسبهٔ پاداش پایان خدمت (پاداش سنوات کارکنان
// رسمی) را ثبت می‌کند:
//
//	GET  /api/v1/payroll/service-bonus/parameters              پارامترها و جدول‌های مرجع
//	GET  /api/v1/payroll/service-bonus/prefill?employee=…      پیش‌پرکردن از پرونده و حکم
//	POST /api/v1/payroll/service-bonus/calculate               محاسبهٔ صورت‌حساب
//
// قاعدهٔ محاسبه در بستهٔ domain است (بازتولید صورت‌حساب مرجع صنعت نفت) و این فایل فقط
// دادهٔ لازم را از سه منبع می‌آورد: پارامترهای حقوق همان شرکت، پروندهٔ پرسنلی کارمند
// (پورت کارکنان) و آخرین حکم ابلاغ‌شدهٔ او (پورت احکام). هیچ عددی در UI یا این لایه
// سخت نشده است.
func registerServiceBonusEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	h := &serviceBonusHandler{deps: deps, svc: svc}
	r := func(action string, fn http.HandlerFunc) http.HandlerFunc {
		return deps.Require("payroll", action, fn)
	}
	mux.HandleFunc("GET /api/v1/payroll/service-bonus/parameters", r("read", h.parameters))
	mux.HandleFunc("GET /api/v1/payroll/service-bonus/prefill", r("read", h.prefill))
	mux.HandleFunc("POST /api/v1/payroll/service-bonus/calculate", r("read", h.calculate))
}

type serviceBonusHandler struct {
	deps modulekit.Deps
	svc  *recordkit.Service
}

// rates دفتر پارامترهای حقوق جاری شرکت را می‌خواند و به ضرایب محاسبه تبدیل می‌کند.
// هر درخواست تازه می‌خواند تا تغییر پارامترها بی‌درنگ دیده شود.
func (h *serviceBonusHandler) rates(r *http.Request) (contracts.ServiceBonusRates, map[string]string) {
	_, values, _ := collectParams(r.Context(), h.deps, h.svc)
	return domain.RatesFromParams(values), values
}

// parameters پارامترها و جدول‌های مرجع محاسبه را برمی‌گرداند تا فرم بدون هیچ عدد
// سخت‌شده‌ای ساخته شود.
func (h *serviceBonusHandler) parameters(w http.ResponseWriter, r *http.Request) {
	rates, values := h.rates(r)
	items := make([]contracts.ServiceBonusParameter, 0, 8)
	for _, key := range []string{
		domain.ParamBadClimate, domain.ParamSpecial, domain.ParamWorkshop,
		domain.ParamAttractionPublic, domain.ParamPersonalBase, domain.ParamJobSpecialty,
	} {
		items = append(items, contracts.ServiceBonusParameter{
			Key: key, TitleFa: domain.RateParameterTitles[key],
			Value: rateValue(rates, key), UnitFa: "نسبت", GroupFa: "درصدهای محاسبه", Editable: true,
		})
	}
	items = append(items, contracts.ServiceBonusParameter{
		Key: domain.ParamSalaryRepair, TitleFa: domain.RateParameterTitles[domain.ParamSalaryRepair],
		Value: paramNumber(values, domain.ParamSalaryRepair), UnitFa: "ریال",
		GroupFa: "مبالغ", Editable: true,
	})

	degrees := make([]contracts.ServiceBonusParameter, 0, len(domain.DegreeParameterKeys))
	for _, key := range domain.DegreeParameterKeys {
		degrees = append(degrees, contracts.ServiceBonusParameter{
			Key: key, TitleFa: domain.DegreeTitlesFa[key], Value: paramNumber(values, key),
			UnitFa: "نسبت", GroupFa: "جدول مدرک تحصیلی", Editable: true,
		})
	}

	// امروز به شمسی از لایهٔ سرور می‌آید تا فرم تاریخ مبنای درست را پیش‌فرض نشان دهد.
	todayFa := ""
	if today, err := calendar.FromGregorian(h.deps.Now()); err == nil {
		todayFa = today.String()
	}

	httpx.WriteJSON(w, http.StatusOK, contracts.ServiceBonusParameters{
		Rates:   rates,
		Items:   items,
		Degrees: degrees,
		TodayFa: todayFa,
		Sources: []string{
			"پارامترهای حقوق این شرکت (صفحهٔ پارامترها و ضرائب حقوق)",
			"پروندهٔ پرسنلی کارمند (تاریخ استخدام و پایهٔ حقوق)",
			"آخرین حکم ابلاغ‌شدهٔ کارمند (پایهٔ حقوق و نوع استخدام)",
		},
	})
}

// prefill ورودی محاسبه را از پروندهٔ پرسنلی و حکم ابلاغ‌شده پر می‌کند تا کاربر فقط
// موارد اختصاصی همان پرونده (سابقهٔ شمال/جنوب و ترمیم حقوق) را تکمیل کند.
func (h *serviceBonusHandler) prefill(w http.ResponseWriter, r *http.Request) {
	employeeID := strings.TrimSpace(r.URL.Query().Get("employee"))
	if employeeID == "" {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "شناسهٔ کارمند الزامی است.", nil)
		return
	}
	baseDate := strings.TrimSpace(r.URL.Query().Get("baseDate"))
	if baseDate == "" {
		if today, err := calendar.FromGregorian(h.deps.Now()); err == nil {
			baseDate = today.String()
		}
	}
	rates, values := h.rates(r)

	in := contracts.ServiceBonusInput{EmployeeID: employeeID, BaseDate: baseDate, Rates: rates}
	result := contracts.ServiceBonusResult{
		EmployeeID:   employeeID,
		BaseDate:     baseDate,
		Rates:        rates,
		SalaryRepair: int64(paramNumber(values, domain.ParamSalaryRepair)),
	}
	sources := []string{}

	// ۱) پروندهٔ پرسنلی: نام، پایهٔ حقوق و سابقهٔ اولیه از تاریخ استخدام.
	if reader, ok := h.deps.Ports.Employees(); ok {
		if emp, found, err := reader.Get(r.Context(), contracts.EmployeeFilter{Scope: "all"}, employeeID); err == nil && found {
			result.EmployeeName = emp.FullName
			in.BaseSalary = emp.BaseSalary
			sources = append(sources, "پروندهٔ پرسنلی")
			if days, derr := serviceDaysBetween(emp.HireDate, baseDate); derr == nil && days > 0 {
				in.South = domain.TenureFromDays(days)
			}
		}
	}

	// ۲) حکم ابلاغ‌شده: پایهٔ حقوق معتبر آن بر پایهٔ پرونده مقدم است.
	if reader, ok := h.deps.Ports.SalaryDecrees(); ok {
		if decree, found, err := reader.NotifiedDecreeOf(r.Context(), employeeID); err == nil && found {
			if decree.BaseSalary > 0 {
				in.BaseSalary = decree.BaseSalary
			}
			if decree.EmployeeName != "" {
				result.EmployeeName = decree.EmployeeName
			}
			result.NoteFa = "پایهٔ حقوق از حکم ابلاغ‌شده " + decree.ID + " برداشته شد"
			sources = append(sources, "حکم حقوق "+decree.ID)
		}
	}

	result.BaseSalary = in.BaseSalary
	result.North = in.North
	result.South = in.South
	result.Total = domain.AddTenures(in.North, in.South)
	result.NorthYears = domain.TenureYears(in.North)
	result.SouthYears = domain.TenureYears(in.South)
	result.SouthYearsUsed = domain.Truncate2(result.SouthYears)
	result.Sources = sources
	httpx.WriteJSON(w, http.StatusOK, result)
}

// calculate صورت‌حساب پاداش پایان خدمت را از ورودی کاربر می‌سازد.
//
// ورودی خالی/ناقص با پارامترهای شرکت و دادهٔ کارمند کامل می‌شود؛ پس فرم می‌تواند فقط
// سابقه و ترمیم حقوق را بگیرد و بقیه از دادهٔ سازمانی بیاید.
func (h *serviceBonusHandler) calculate(w http.ResponseWriter, r *http.Request) {
	var in contracts.ServiceBonusInput
	if !httpx.DecodeJSON(w, r, &in) {
		return
	}
	in.EmployeeID = strings.TrimSpace(in.EmployeeID)
	if in.EmployeeID == "" {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "انتخاب کارمند الزامی است.", nil)
		return
	}
	if strings.TrimSpace(in.BaseDate) == "" {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest, "تاریخ مبنای محاسبه الزامی است.", nil)
		return
	}
	rates, values := h.rates(r)

	// پایهٔ حقوق: اگر کاربر نداده باشد، از حکم ابلاغ‌شده و سپس پروندهٔ پرسنلی.
	if in.BaseSalary <= 0 {
		if reader, ok := h.deps.Ports.SalaryDecrees(); ok {
			if decree, found, err := reader.NotifiedDecreeOf(r.Context(), in.EmployeeID); err == nil && found {
				in.BaseSalary = decree.BaseSalary
			}
		}
	}
	var employeeName string
	if reader, ok := h.deps.Ports.Employees(); ok {
		if emp, found, err := reader.Get(r.Context(), contracts.EmployeeFilter{Scope: "all"}, in.EmployeeID); err == nil && found {
			employeeName = emp.FullName
			if in.BaseSalary <= 0 {
				in.BaseSalary = emp.BaseSalary
			}
		}
	}
	if in.BaseSalary <= 0 {
		httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
			"حقوق پایه در دسترس نیست؛ مبلغ پایه را وارد کنید یا برای کارمند حکم ابلاغ‌شده صادر کنید.", nil)
		return
	}

	// ضریب صفر یعنی «از پارامترهای شرکت بخوان»؛ کاربر فقط موارد دلخواه را بازنویسی می‌کند.
	given := in.Rates
	if given.BadClimate <= 0 {
		given.BadClimate = rates.BadClimate
	}
	if given.Special <= 0 {
		given.Special = rates.Special
	}
	if given.Workshop <= 0 {
		given.Workshop = rates.Workshop
	}
	if given.AttractionPublic <= 0 {
		given.AttractionPublic = rates.AttractionPublic
	}
	if given.PersonalBase <= 0 {
		given.PersonalBase = rates.PersonalBase
	}
	if given.JobSpecialty <= 0 {
		given.JobSpecialty = rates.JobSpecialty
	}
	in.Rates = given
	if in.SalaryRepair <= 0 {
		in.SalaryRepair = int64(paramNumber(values, domain.ParamSalaryRepair))
	}

	result := domain.ComputeServiceBonus(in)
	result.EmployeeName = employeeName
	httpx.WriteJSON(w, http.StatusOK, result)
}

// rateValue مقدار یک ضریب را از مجموعهٔ ضرایب می‌خواند (برای نمایش در فرم).
func rateValue(rates contracts.ServiceBonusRates, key string) float64 {
	switch key {
	case domain.ParamBadClimate:
		return rates.BadClimate
	case domain.ParamSpecial:
		return rates.Special
	case domain.ParamWorkshop:
		return rates.Workshop
	case domain.ParamAttractionPublic:
		return rates.AttractionPublic
	case domain.ParamPersonalBase:
		return rates.PersonalBase
	case domain.ParamJobSpecialty:
		return rates.JobSpecialty
	}
	return 0
}

// paramNumber مقدار عددی یک پارامتر را می‌خواند؛ خالی یا نامعتبر صفر می‌شود.
func paramNumber(values map[string]string, key string) float64 {
	raw := strings.TrimSpace(values[key])
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}

// serviceDaysBetween تعداد روز میان دو تاریخ شمسی را می‌شمارد (پایهٔ سابقهٔ اولیه).
func serviceDaysBetween(from, to string) (int, error) {
	a, err := calendar.Parse(from)
	if err != nil {
		return 0, err
	}
	b, err := calendar.Parse(to)
	if err != nil {
		return 0, err
	}
	return calendar.DaysBetween(a, b)
}
