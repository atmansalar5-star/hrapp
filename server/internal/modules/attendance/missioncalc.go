// پیاده‌سازی ماژول «attendance» — «فرم محاسبهٔ مأموریت».
//
// چرا این فایل: فوقالعادهٔ مأموریت فقط یک مبلغ ثابت روزانه نیست؛ ضریب عوامل مأموریت،
// سهم صبحانه/نهار/شام/خواب، پذیرایی میزبان، اقامت در هتل یا مهمانسرا، استفاده از امکانات
// شخصی و هزینه‌های ایاب‌وذهاب (تاکسی، اتوبوس، کرایهٔ کیلومتری) مبلغ قابل پرداخت را می‌سازند.
// قاعده در موتور قواعد (`rules.ComputeMissionAllowance`) است و این فایل فقط پارامترهای
// استقرار را می‌خواند و فرم را به همان موتور وصل می‌کند؛ پس فرم، رکورد مأموریت و حقوق یک
// عدد می‌دهند.
//
// هیچ قاعده‌ای در رابط وب نیست: رابط فیلدها را می‌فرستد و تفکیک محاسبه را از همین مسیر
// می‌گیرد.

package attendance

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/modules/attendance/domain"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// registerMissionCalcEndpoints مسیرهای فرم محاسبهٔ مأموریت را ثبت می‌کند.
func registerMissionCalcEndpoints(mux *http.ServeMux, deps modulekit.Deps, svc *recordkit.Service) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("attendance", "read", fn) }

	// پارامترهای پیش‌فرض فرم: نرخ روزانه، عامل‌ها، نرخ کیلومتری و درصد امکانات شخصی از
	// استقرار شرکت فعال خوانده می‌شود تا فرم با عدد درست باز شود.
	mux.HandleFunc("GET /api/v1/attendance/missions/allowance-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, missionDefaults(deps))
	}))

	// محاسبهٔ فوقالعادهٔ مأموریت: ورودی همان فیلدهای فرم، خروجی تفکیک سطرها و جمع قابل پرداخت.
	// ورودی می‌تواند اسناد پشتیبان (قبض تاکسی و فاکتورهای تأییدشده)، بخش اعزام پزشکی و بخش
	// ارزی مأموریت خارج از کشور را هم داشته باشد؛ خروجی هم `allowance` (فوقالعادهٔ روزانه) و
	// هم `settlement` (تسویهٔ کامل) را برمی‌گرداند.
	mux.HandleFunc("POST /api/v1/attendance/missions/allowance", read(func(w http.ResponseWriter, r *http.Request) {
		var in rules.MissionSettlementInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی محاسبهٔ مأموریت خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, missionAllowance(in, deps))
	}))

	// اسناد پشتیبان یک مأموریت: قبض تاکسی و فاکتورهایی که ثبت یا تأیید شده‌اند. فرم از همین
	// مسیر فهرست را می‌خواند تا سندهای تأییدشده را در جمع بیاورد.
	mux.HandleFunc("GET /api/v1/attendance/missions/documents", read(func(w http.ResponseWriter, r *http.Request) {
		mission := strings.TrimSpace(r.URL.Query().Get("mission"))
		docs := missionDocuments(r.Context(), svc, mission, r.URL.Query().Get("employee"))
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"documents": docs, "total": len(docs), "mission": mission,
			"types": domain.MissionDocumentTypes(),
		})
	}))
}

// missionDefaults پارامترهای محاسبهٔ مأموریت را از استقرار فعال می‌سازد.
func missionDefaults(deps modulekit.Deps) map[string]any {
	params := missionParams(deps)
	// تاریخ امروز از ساعت سرور می‌آید تا فرم با تاریخ شمسی درست باز شود و تبدیل تقویم
	// در رابط انجام نشود.
	today := ""
	if now, err := calendar.FromGregorian(deps.Clock()); err == nil {
		today = now.String()
	}
	return map[string]any{
		"perDiem":            params.PerDiem,
		"missionFactor":      params.MissionFactor,
		"factors":            params.Factors,
		"factorsTotal":       params.Factors.Total(),
		"mileageRate":        missionParam(deps, "mission.mileageRate", DefaultMileageRate),
		"personalPercent":    params.PersonalFacilitiesPercent,
		"company":            deps.Org.TitleFa,
		"today":              today,
		"yesNo":              []map[string]string{{"key": "yes", "titleFa": "بله"}, {"key": "no", "titleFa": "خیر"}},
		"params":             missionParamViews(deps),
		"noteFa":             missionNoteFa(),
		"defaultDurationDay": 1,
		// فهرست‌های مرجع (نوع مأموریت، مشمول اعزام پزشکی، نوع اسکان، ارز و نوع سند).
		"missionTypes":        domain.MissionTypes(),
		"beneficiaries":       domain.MedicalBeneficiaries(),
		"accommodations":      domain.Accommodations(),
		"currencies":          domain.MissionCurrencies(),
		"documentTypes":       domain.MissionDocumentTypes(),
		"defaultCurrency":     defaultCurrency(deps),
		"fxRates":             fxRates(deps),
		"companionShare":      missionParam(deps, "mission.medicalCompanionPercent", 0),
		"fxPerDiemDefault":    missionParam(deps, "mission.fx.perDiem", 0),
		"payerOptions":        []map[string]string{{"key": "company", "titleFa": "سازمان"}, {"key": "employee", "titleFa": "کارمند"}, {"key": "insurer", "titleFa": "بیمه"}},
		"noteMedical":         domain.MissionTypeTitleFa("medical"),
		"noteForeignCurrency": "مبلغ ارزی و ریالی جداگانه محاسبه می‌شود؛ معادل ریالی با نرخ تسعیر ثبت‌شده ساخته می‌شود.",
	}
}

// defaultCurrency ارز پیش‌فرض مأموریت خارج از کشور (پارامتر شرکت، وگرنه یورو).
func defaultCurrency(deps modulekit.Deps) string {
	code := strings.ToUpper(strings.TrimSpace(missionParamText(deps, "mission.fx.defaultCurrency")))
	if code == "" || !domain.ValidCurrency(code) {
		return "EUR"
	}
	return code
}

// fxRates نرخ تسعیر هر ارز به ریال از پارامترهای استقرار (کلید mission.fx.rate.<CODE>).
func fxRates(deps modulekit.Deps) map[string]float64 {
	out := map[string]float64{}
	for _, option := range domain.MissionCurrencies() {
		if option.Key == "IRR" {
			continue
		}
		if value := missionParam(deps, "mission.fx.rate."+option.Key, 0); value > 0 {
			out[option.Key] = value
		}
	}
	return out
}

// missionParamText مقدار رشته‌ای یک پارامتر استقرار.
func missionParamText(deps modulekit.Deps, key string) string {
	for _, p := range deps.Org.Params {
		if p.Key == key {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

// missionDocumentView یک سند پشتیبان در پاسخ API.
type missionDocumentView struct {
	ID       string  `json:"id"`
	Mission  string  `json:"mission"`
	Employee string  `json:"employeeId"`
	TitleFa  string  `json:"titleFa"`
	Kind     string  `json:"kind"`
	KindFa   string  `json:"kindFa"`
	DocNo    string  `json:"docNo"`
	DocDate  string  `json:"docDate"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Status   string  `json:"status"`
	StatusFa string  `json:"statusFa"`
	Approved bool    `json:"approved"`
	Payer    string  `json:"payer"`
	NoteFa   string  `json:"noteFa,omitempty"`
}

// missionDocuments اسناد پشتیبان یک مأموریت (یا همهٔ اسناد یک کارمند) را می‌خواند.
func missionDocuments(ctx context.Context, svc *recordkit.Service, mission, employee string) []missionDocumentView {
	if svc == nil {
		return []missionDocumentView{}
	}
	filter := contracts.RecordFilter{
		Module: "attendance", Kind: "missionDocuments", Scope: "all", Size: 500,
		EmployeeID: strings.TrimSpace(employee), Query: strings.TrimSpace(mission),
	}
	page, err := svc.List(ctx, filter)
	if err != nil {
		return []missionDocumentView{}
	}
	out := make([]missionDocumentView, 0, len(page.Items))
	for _, rec := range page.Items {
		if mission != "" && strings.TrimSpace(rec.Fields["mission"]) != strings.TrimSpace(mission) {
			continue
		}
		amount, _ := strconv.ParseFloat(strings.TrimSpace(rec.Fields["amount"]), 64)
		kind := strings.TrimSpace(rec.Fields["docType"])
		out = append(out, missionDocumentView{
			ID: rec.ID, Mission: strings.TrimSpace(rec.Fields["mission"]),
			Employee: rec.EmployeeID, TitleFa: rec.Title, Kind: kind,
			KindFa: domain.DocumentTypeTitleFa(kind), DocNo: strings.TrimSpace(rec.Fields["docNo"]),
			DocDate: strings.TrimSpace(rec.Fields["docDate"]), Amount: amount,
			Currency: strings.ToUpper(strings.TrimSpace(rec.Fields["currency"])),
			Status:   rec.Status, StatusFa: documentStatusFa(rec.Status),
			Approved: rec.Status == "approved", Payer: strings.TrimSpace(rec.Fields["payer"]),
			NoteFa: strings.TrimSpace(rec.Fields["note"]),
		})
	}
	return out
}

// documentStatusFa برچسب فارسی وضعیت سند پشتیبان.
func documentStatusFa(status string) string {
	switch strings.TrimSpace(status) {
	case "draft":
		return "ثبت‌شده"
	case "pending":
		return "در انتظار تأیید مالی"
	case "approved":
		return "تأییدشده"
	case "rejected":
		return "ردشده"
	}
	return strings.TrimSpace(status)
}

// documentsToRules تبدیل اسناد ذخیره‌شده به شکل موتور قواعد (مبنا: سند تأییدشده).
func documentsToRules(docs []missionDocumentView) []rules.MissionDocument {
	out := make([]rules.MissionDocument, 0, len(docs))
	for _, doc := range docs {
		out = append(out, rules.MissionDocument{
			Key: doc.ID, Kind: doc.Kind, TitleFa: doc.KindFa, DocNo: doc.DocNo, DocDate: doc.DocDate,
			Amount: doc.Amount, Currency: doc.Currency, Approved: doc.Approved, Payer: doc.Payer,
			NoteFa: doc.NoteFa,
		})
	}
	return out
}

// DefaultMileageRate نرخ پیش‌فرض کرایهٔ کیلومتری وقتی پارامتر شرکت تنظیم نشده باشد.
const DefaultMileageRate = 50_000

// missionParams پارامترهای محاسبه را از پروفایل شرکت فعال می‌خواند.
//
// چرا از پارامتر و نه عدد در کد: نرخ روزانه، سهم وعده‌ها و درصد امکانات شخصی هر شرکت
// متفاوت است؛ هیچ‌کدام در کد ثابت نمی‌شوند.
func missionParams(deps modulekit.Deps) rules.MissionParams {
	params := rules.DefaultMissionParams()
	if value := missionParam(deps, "mission.perDiem", 0); value > 0 {
		params.PerDiem = value
	}
	// سهم وعده‌ها و خواب در استقرار به درصد ثبت شده است (۱۰ = ۱۰٪).
	if value := missionParam(deps, "mission.factorBreakfast", 0); value > 0 {
		params.Factors.Breakfast = value / 100
	}
	if value := missionParam(deps, "mission.factorLunch", 0); value > 0 {
		params.Factors.Lunch = value / 100
	}
	if value := missionParam(deps, "mission.factorDinner", 0); value > 0 {
		params.Factors.Dinner = value / 100
	}
	if value := missionParam(deps, "mission.factorLodging", 0); value > 0 {
		params.Factors.Lodging = value / 100
	}
	if value := missionParam(deps, "mission.personalFacilitiesPercent", 0); value > 0 {
		params.PersonalFacilitiesPercent = value
	}
	return params
}

// missionParam مقدار عددی یک پارامتر استقرار (صفر اگر نبود).
func missionParam(deps modulekit.Deps, key string, fallback float64) float64 {
	for _, p := range deps.Org.Params {
		if p.Key != key {
			continue
		}
		if value, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 64); err == nil {
			return value
		}
	}
	return fallback
}

// missionParamViews پارامترهای مأموریت را برای نمایش در فرم برمی‌گرداند.
func missionParamViews(deps modulekit.Deps) []map[string]string {
	out := []map[string]string{}
	for _, p := range deps.Org.Params {
		if !strings.HasPrefix(p.Key, "mission.") {
			continue
		}
		out = append(out, map[string]string{
			"key": p.Key, "titleFa": p.TitleFa, "value": p.Value, "unitFa": p.UnitFa,
		})
	}
	return out
}

// missionAllowance ورودی فرم را با پارامترهای استقرار کامل می‌کند و محاسبه را انجام می‌دهد.
//
// قاعده: مقدارهای خالی (صفر) فرم با پارامتر شرکت پر می‌شوند؛ اگر کاربر عددی گذاشته باشد
// همان مقدم است. نرخ کیلومتری و نرخ تسعیر ارز هم اگر در فرم نیامده باشند از پارامتر شرکت
// می‌آیند. خروجی هم فوقالعادهٔ روزانه (`allowance`) و هم تسویهٔ کامل (`settlement`) را دارد.
func missionAllowance(in rules.MissionSettlementInput, deps modulekit.Deps) map[string]any {
	params := missionParams(deps)
	if in.PerDiem <= 0 {
		in.PerDiem = params.PerDiem
	}
	if in.MissionFactor <= 0 {
		in.MissionFactor = params.MissionFactor
	}
	if in.PersonalFacilitiesPercent <= 0 {
		in.PersonalFacilitiesPercent = params.PersonalFacilitiesPercent
	}
	if in.Factors.Total() <= 0 {
		in.Factors = params.Factors
	}
	if in.MileageKm > 0 && in.MileageRate <= 0 {
		in.MileageRate = missionParam(deps, "mission.mileageRate", DefaultMileageRate)
	}
	// مأموریت خارج از کشور: ارز، نرخ تسعیر و فوقالعادهٔ روزانهٔ ارزی از پارامتر شرکت می‌آید
	// اگر در فرم ثبت نشده باشد.
	if strings.EqualFold(strings.TrimSpace(in.MissionType), "foreign") {
		if strings.TrimSpace(in.Currency) == "" {
			in.Currency = defaultCurrency(deps)
		}
		if in.FxRate <= 0 {
			in.FxRate = missionParam(deps, "mission.fx.rate."+strings.ToUpper(in.Currency), 0)
		}
		if in.FxPerDiem <= 0 {
			in.FxPerDiem = missionParam(deps, "mission.fx.perDiem", 0)
		}
	}
	// اعزام پزشکی: سهم وعده‌های همراه بیمار از پارامتر شرکت.
	if strings.EqualFold(strings.TrimSpace(in.MissionType), "medical") && in.CompanionSharePercent <= 0 {
		in.CompanionSharePercent = missionParam(deps, "mission.medicalCompanionPercent", 0)
	}
	// اسناد پشتیبان (قبض تاکسی و فاکتورها) از فرم می‌آید؛ اگر فرم سندی نفرستد، جمع اسناد
	// صفر می‌ماند و رابط می‌تواند فهرست اسناد ثبت‌شدهٔ مأموریت را جدا نمایش دهد.
	allowance := rules.ComputeMissionAllowance(in.MissionAllowanceInput)
	settlement := rules.ComputeMissionSettlement(in)
	// پایان مأموریت در سرور حساب می‌شود، نه در رابط: تبدیل تقویم فقط در لایهٔ کد انجام
	// می‌گیرد. یک روز اول مأموریت است، پس پایان = شروع + (مدت − ۱) روز.
	endDate := ""
	if start, err := calendar.Parse(strings.TrimSpace(in.StartDate)); err == nil {
		days := int(math.Ceil(in.DurationDays))
		if days < 1 {
			days = 1
		}
		if end, err := start.AddDays(days - 1); err == nil {
			endDate = end.String()
		}
	}
	return map[string]any{
		"allowance":  allowance,
		"settlement": settlement,
		"input":      in,
		"toDate":     endDate,
		"noteFa":     missionNoteFa(),
	}
}

// missionNoteFa توضیح قاعده‌ها برای فرم؛ متن از سرور می‌آید تا در رابط تکرار نشود.
func missionNoteFa() string {
	return "فوقالعادهٔ مأموریت از نرخ روزانهٔ شرکت (پارامتر mission.perDiem) و ضریب عوامل " +
		"مأموریت ساخته می‌شود: سهم صبحانه، نهار، شام و خواب درصدی از همان نرخ روزانه‌اند " +
		"(پارامترهای mission.factor*). اگر پذیرایی وعده‌ای با میزبان باشد، عامل همان وعده را " +
		"صفر بگذارید. با اقامت در هتل یا مهمانسرا، سهم خواب پرداخت نمی‌شود و هزینهٔ واقعی اقامت " +
		"جای آن می‌آید. استفاده از امکانات شخصی جبران درصدی دارد و تاکسی، اتوبوس و کرایهٔ " +
		"کیلومتری (کیلومتر × نرخ) مستقیم جمع می‌شوند.\n" +
		"قبض تاکسی و فاکتورها به‌صورت «سند و فاکتور مأموریت» ثبت و پس از تأیید مالی در جمع " +
		"می‌آیند؛ سند تأییدنشده در فهرست می‌ماند ولی پرداخت نمی‌شود.\n" +
		"در اعزام پزشکی، هزینه بر پایهٔ مشمول است: بیمار همهٔ اجزا و همراه بیمار اسکان و " +
		"ایاب‌وذهاب (سهم وعده‌ها تابع پارامتر mission.medicalCompanionPercent).\n" +
		"در مأموریت خارج از کشور پرداخت دوگانه است: مبلغ ارزی (فوقالعادهٔ ارزی × مدت + سایر " +
		"مبالغ ارزی) و مبلغ ریالی؛ معادل ریالی با نرخ تسعیر پارامتر mission.fx.rate.<ارز> " +
		"ساخته می‌شود. الگوی نرخ‌ها بر پایهٔ ساختار دستورالعمل مأموریت شرکت ملی نفت ایران است و " +
		"با جزئیات دقیق شما جایگزین می‌شود."
}
