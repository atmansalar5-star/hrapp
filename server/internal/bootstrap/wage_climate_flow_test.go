// آزمون جریان «اجزای مزد و فوق‌العاده‌های اقلیمی»: ثبت در حکم و قرارداد، و اثر آن بر
// مزد مبنا و فیش حقوق. این آزمون همان چیزی را می‌سنجد که کاربر در رابط می‌بیند.

package bootstrap

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hrms/internal/contracts"
)

// payslipResponse همان پاسخ فیش حقوقی که رابط مصرف می‌کند.
type payslipResponse struct {
	Wages         map[string]float64 `json:"wages"`
	MazdBase      float64            `json:"mazdBase"`
	MazdBreakdown []struct {
		Code    string  `json:"code"`
		TitleFa string  `json:"titleFa"`
		KindFa  string  `json:"kindFa"`
		Amount  float64 `json:"amount"`
	} `json:"mazdBreakdown"`
	Climate struct {
		BadClimate     float64 `json:"badClimate"`
		Deprivation    float64 `json:"deprivation"`
		RegionHardship float64 `json:"regionHardship"`
		ExtraTime      float64 `json:"extraTime"`
		RegionFa       string  `json:"regionFa"`
		SourceFa       string  `json:"sourceFa"`
	} `json:"climate"`
	WageSource string `json:"wageSource"`
	Elements   []struct {
		Code     string  `json:"code"`
		TitleFa  string  `json:"titleFa"`
		Amount   float64 `json:"amount"`
		Resolved bool    `json:"resolved"`
	} `json:"elements"`
}

func decodePayslip(t *testing.T, rec *httptest.ResponseRecorder) payslipResponse {
	t.Helper()
	var out payslipResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("پاسخ فیش قابل تجزیه نیست: %v (%s)", err, rec.Body.String())
	}
	return out
}

// TestPayslipUsesDecreeWageComponents می‌سنجد فیش حقوق مزد مبنا را از اجزای حکم کارمند و
// ضرایب اقلیمی همان حکم می‌سازد — نه فقط از پایهٔ پرونده و پارامترهای سازمانی.
func TestPayslipUsesDecreeWageComponents(t *testing.T) {
	h := newTestServer(t)
	token := login(t, h, "hr.manager", "demo").AccessToken

	// کارمند نمونهٔ دارای حکم ابلاغ‌شده (EMP-005 با DEC-1404-0213 در بذرها).
	rec := call(t, h, http.MethodGet, "/api/v1/payroll/payslip?employee=EMP-005", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("فیش حقوقی باید ۲۰۰ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	pay := decodePayslip(t, rec)

	if pay.WageSource == "" || !strings.Contains(pay.WageSource, "حکم کارگزینی") {
		t.Fatalf("منبع اجزای مزد باید حکم کارگزینی باشد: %q", pay.WageSource)
	}
	if pay.Wages["climate"] <= 0 {
		t.Fatalf("فوق‌العادهٔ اقلیمی مصوب حکم در فیش نیامده: %+v", pay.Wages)
	}
	// مزد مبنا باید جمع همهٔ اجزای شاکله باشد (با خطای گرد کردن زیر یک ریال).
	sum := 0.0
	for _, p := range pay.MazdBreakdown {
		sum += p.Amount
	}
	if math.Abs(sum-pay.MazdBase) > 1 {
		t.Fatalf("جمع شاکلهٔ مزد مبنا (%v) با مزد مبنا (%v) نمی‌خواند", sum, pay.MazdBase)
	}
	// مزد مبنا باید بیشتر از جمع پنج جزء قدیمی باشد؛ یعنی ثابت‌های شغلی هم داخل آن‌اند.
	old := pay.Wages["job"] + pay.Wages["rank"] + pay.Wages["seniority"] + pay.Wages["postAllowance"] + pay.Wages["sacrifice"]
	if pay.MazdBase <= old {
		t.Fatalf("مزد مبنا (%v) باید از جمع پنج جزء قدیمی (%v) بیشتر باشد", pay.MazdBase, old)
	}
	// ضریب اقلیمی از خود حکم می‌آید (۰٫۳۰) نه پیش‌فرض شرکت (۰٫۲۵).
	if math.Abs(pay.Climate.BadClimate-0.30) > 1e-9 {
		t.Fatalf("ضریب بدی آب و هوا = %v، انتظار ۰٫۳۰ از حکم", pay.Climate.BadClimate)
	}
	if !strings.Contains(pay.Climate.SourceFa, "حکم کارگزینی") {
		t.Fatalf("منبع ضریب اقلیمی باید حکم باشد: %q", pay.Climate.SourceFa)
	}
	if math.Abs(pay.Climate.RegionHardship-0.35) > 1e-9 {
		t.Fatalf("ضریب صعوبت منطقه = %v، انتظار ۰٫۳۵ از حکم", pay.Climate.RegionHardship)
	}

	// قلم «بدی آب و هوا» در تفکیک اقلام فیش باید درصدی از همان مزد مبنا باشد.
	var bad float64
	for _, e := range pay.Elements {
		if e.Code == "BAD_CLIMATE" {
			bad = e.Amount
		}
		if !e.Resolved && strings.Contains(e.TitleFa, "اقلیمی") {
			t.Fatalf("قلم اقلیمی «%s» حل نشد", e.TitleFa)
		}
	}
	if math.Abs(bad-0.30*pay.MazdBase) > 1 {
		t.Fatalf("مبلغ بدی آب و هوا = %v، انتظار ۳۰٪ مزد مبنا = %v", bad, 0.30*pay.MazdBase)
	}
}

// TestDecreeClimateFieldsRoundTrip می‌سنجد فیلدهای اقلیمی حکم ثبت، اعتبارسنجی و برگردانده
// می‌شوند و مزد مبنا در پاسخ می‌آید.
func TestDecreeClimateFieldsRoundTrip(t *testing.T) {
	h := newTestServer(t)
	token := login(t, h, "hr.manager", "demo").AccessToken

	body := map[string]any{
		"employeeId": "EMP-011", "jobGroup": "کارشناس شبکه", "baseSalary": 200_000_000,
		"effectiveDate": "1405/07/01", "reasonFa": "افزایش منطقهٔ اقلیمی محل خدمت",
		"climateRegion": "south_coast", "rankWage": 30_000_000, "seniorityWage": 15_000_000,
		"climateAllowance": 12_000_000, "salaryRepair": 3_000_000,
		"badClimatePercent": 0.25, "deprivationPercent": 0.25,
		"regionHardshipPercent": 0.30, "extraTimePercent": 0.15,
	}
	rec := call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees", token, body)
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("ثبت حکم با فیلدهای اقلیمی باید موفق باشد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var decree contracts.SalaryDecree
	if err := json.Unmarshal(rec.Body.Bytes(), &decree); err != nil {
		t.Fatalf("پاسخ حکم قابل تجزیه نیست: %v", err)
	}
	if decree.ClimateRegion != "south_coast" || decree.ClimateRegionFa == "" {
		t.Fatalf("منطقهٔ اقلیمی ثبت/برگردانده نشد: %+v", decree.WageStructure)
	}
	if decree.ClimateAllowance != 12_000_000 || decree.SalaryRepair != 3_000_000 {
		t.Fatalf("مبالغ اقلیمی/ترمیم ثبت نشد: %+v", decree.WageStructure)
	}
	// مزد مبنا ≥ مزد شغل + رتبه + سنوات + اقلیمی + ترمیم (ثابت‌های شغلی سازمانی هم اضافه می‌شوند).
	floor := int64(200_000_000 + 30_000_000 + 15_000_000 + 12_000_000 + 3_000_000)
	if decree.MazdBase < floor {
		t.Fatalf("مزد مبنای حکم = %d، دست‌کم انتظار %d", decree.MazdBase, floor)
	}

	// همان حکم در فهرست هم باید مزد مبنا و منطقهٔ اقلیمی داشته باشد.
	list := call(t, h, http.MethodGet, "/api/v1/salary_admin/decrees?size=200", token, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("فهرست احکام باید ۲۰۰ بدهد، %d داد", list.Code)
	}
	page := struct {
		Items []contracts.SalaryDecree `json:"items"`
	}{}
	if err := json.Unmarshal(list.Body.Bytes(), &page); err != nil {
		t.Fatalf("فهرست احکام قابل تجزیه نیست: %v", err)
	}
	found := false
	for _, d := range page.Items {
		if d.ID == decree.ID {
			found = true
			if d.MazdBase != decree.MazdBase {
				t.Fatalf("مزد مبنای فهرست (%d) با پاسخ ثبت (%d) یکی نیست", d.MazdBase, decree.MazdBase)
			}
		}
	}
	if !found {
		t.Fatalf("حکم تازه در فهرست پیدا نشد: %s", decree.ID)
	}

	// ضریب بیرون بازه رد می‌شود (قاعدهٔ دامنه، نه رابط).
	bad := map[string]any{
		"employeeId": "EMP-011", "jobGroup": "کارشناس شبکه", "baseSalary": 200_000_000,
		"effectiveDate": "1405/07/01", "reasonFa": "آزمون اعتبارسنجی",
		"badClimatePercent": 1.5,
	}
	if rec := call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees", token, bad); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ضریب بزرگ‌تر از یک باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
	// منطقهٔ ناشناخته هم رد می‌شود.
	badRegion := map[string]any{
		"employeeId": "EMP-011", "jobGroup": "کارشناس شبکه", "baseSalary": 200_000_000,
		"effectiveDate": "1405/07/01", "reasonFa": "آزمون اعتبارسنجی", "climateRegion": "atlantis",
	}
	if rec := call(t, h, http.MethodPost, "/api/v1/salary_admin/decrees", token, badRegion); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("منطقهٔ ناشناخته باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}
}

// TestContractClimateFields می‌سنجد قرارداد هم فوق‌العاده‌های اقلیمی را نگه می‌دارد و
// مبلغ اقلیمی قرارداد بر مزد مبنای فیش اثری دارد.
func TestContractClimateFields(t *testing.T) {
	h := newTestServer(t)
	token := login(t, h, "hr.manager", "demo").AccessToken

	// EMP-012 قرارداد جاری ندارد (قرارداد قبلی‌اش خاتمه یافته است)، پس بازهٔ همپوشان ندارد.
	body := map[string]any{
		"employeeId": "EMP-012", "kind": "fixed_term", "employmentType": "fixed_1662",
		"startDate": "1405/07/01", "endDate": "1406/06/31", "monthlySalary": 150_000_000,
		"climateRegion": "south_coast", "rankWage": 20_000_000, "climateAllowance": 9_000_000,
		"badClimatePercent": 0.30,
	}
	rec := call(t, h, http.MethodPost, "/api/v1/contracts/contracts", token, body)
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Fatalf("انعقاد قرارداد با فیلدهای اقلیمی باید موفق باشد، %d داد: %s", rec.Code, rec.Body.String())
	}
	var c contracts.Contract
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
		t.Fatalf("پاسخ قرارداد قابل تجزیه نیست: %v", err)
	}
	if c.ClimateRegion != "south_coast" || c.ClimateAllowance != 9_000_000 || c.RankWage != 20_000_000 {
		t.Fatalf("اجزای اقلیمی قرارداد ثبت نشد: %+v", c.WageStructure)
	}
	if math.Abs(c.BadClimatePercent-0.30) > 1e-9 {
		t.Fatalf("ضریب اقلیمی قرارداد = %v", c.BadClimatePercent)
	}

	// ضریب نامعتبر در قرارداد هم رد می‌شود.
	bad := map[string]any{
		"employeeId": "EMP-012", "kind": "permanent", "employmentType": "fixed_1662",
		"startDate": "1405/07/01", "monthlySalary": 150_000_000, "regionHardshipPercent": 2,
	}
	if rec := call(t, h, http.MethodPost, "/api/v1/contracts/contracts", token, bad); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ضریب نامعتبر قرارداد باید ۴۲۲ بدهد، %d داد: %s", rec.Code, rec.Body.String())
	}

	// سند چاپی قرارداد باید اجزای مزد مبنا و اقلیمی را نشان دهد.
	doc := call(t, h, http.MethodGet, "/api/v1/documents/contract/"+c.ID, token, nil)
	if doc.Code != http.StatusOK {
		t.Fatalf("سند قرارداد باید ۲۰۰ بدهد، %d داد: %s", doc.Code, doc.Body.String())
	}
	if html := doc.Body.String(); !strings.Contains(html, "مزد مبنا") || !strings.Contains(html, "بدی آب و هوا") {
		t.Fatalf("سند قرارداد اجزای مزد مبنا/اقلیمی را ندارد")
	}
}
