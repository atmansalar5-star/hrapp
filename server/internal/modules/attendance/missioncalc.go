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
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/httpx"
	"hrms/internal/platform/modulekit"
	"hrms/internal/rules"
)

// registerMissionCalcEndpoints مسیرهای فرم محاسبهٔ مأموریت را ثبت می‌کند.
func registerMissionCalcEndpoints(mux *http.ServeMux, deps modulekit.Deps) {
	read := func(fn http.HandlerFunc) http.HandlerFunc { return deps.Require("attendance", "read", fn) }

	// پارامترهای پیش‌فرض فرم: نرخ روزانه، عامل‌ها، نرخ کیلومتری و درصد امکانات شخصی از
	// استقرار شرکت فعال خوانده می‌شود تا فرم با عدد درست باز شود.
	mux.HandleFunc("GET /api/v1/attendance/missions/allowance-defaults", read(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, missionDefaults(deps))
	}))

	// محاسبهٔ فوقالعادهٔ مأموریت: ورودی همان فیلدهای فرم، خروجی تفکیک سطرها و جمع قابل پرداخت.
	mux.HandleFunc("POST /api/v1/attendance/missions/allowance", read(func(w http.ResponseWriter, r *http.Request) {
		var in rules.MissionAllowanceInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			httpx.WriteErrorMsg(w, http.StatusBadRequest, contracts.CodeBadRequest,
				"ورودی محاسبهٔ مأموریت خوانده نشد.", nil)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, missionAllowance(in, deps))
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
	}
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
// همان مقدم است. نرخ کیلومتری هم اگر در فرم نیامده باشد از پارامتر شرکت می‌آید.
func missionAllowance(in rules.MissionAllowanceInput, deps modulekit.Deps) map[string]any {
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
	allowance := rules.ComputeMissionAllowance(in)
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
		"allowance": allowance,
		"input":     in,
		"toDate":    endDate,
		"noteFa":    missionNoteFa(),
	}
}

// missionNoteFa توضیح قاعده‌ها برای فرم؛ متن از سرور می‌آید تا در رابط تکرار نشود.
func missionNoteFa() string {
	return "فوقالعادهٔ مأموریت از نرخ روزانهٔ شرکت (پارامتر mission.perDiem) و ضریب عوامل " +
		"مأموریت ساخته می‌شود: سهم صبحانه، نهار، شام و خواب درصدی از همان نرخ روزانه‌اند " +
		"(پارامترهای mission.factor*). اگر پذیرایی وعده‌ای با میزبان باشد، عامل همان وعده را " +
		"صفر بگذارید. با اقامت در هتل یا مهمانسرا، سهم خواب پرداخت نمی‌شود و هزینهٔ واقعی اقامت " +
		"جای آن می‌آید. استفاده از امکانات شخصی جبران درصدی دارد و تاکسی، اتوبوس و کرایهٔ " +
		"کیلومتری (کیلومتر × نرخ) مستقیم جمع می‌شوند."
}
