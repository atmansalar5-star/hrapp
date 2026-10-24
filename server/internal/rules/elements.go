// عناصر و عملگرهای موتور قواعد (محاسبات فرمولی حقوق/جایزه).

package rules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// PayElement تعریف پارامتریک یک قلم حقوق. تعریف قلم در ماژول حقوق نگه‌داری می‌شود، ولی
// محاسبهٔ آن در همین موتور قواعد انجام می‌گیرد تا شبیه‌ساز و فیش چاپی یک عدد واحد
// ببینند و قاعده در دو جا تکرار نشود.
type PayElement struct {
	Code     string
	TitleFa  string
	Category string
	Calc     string
	Rate     string
	Basis    string
	Formula  string
}

// PayElementLine نتیجهٔ محاسبهٔ یک قلم برای نمایش در شبیه‌ساز و فیش.
// `Resolved` یعنی توانستیم برای این قلم عدد قطعی حساب کنیم.
type PayElementLine struct {
	Code     string  `json:"code"`
	TitleFa  string  `json:"titleFa"`
	Category string  `json:"category"`
	Calc     string  `json:"calc"`
	Amount   float64 `json:"amount"`
	Resolved bool    `json:"resolved"`
	DetailFa string  `json:"detailFa"`
}

// PayElementProvider پورت ماژول حقوق: اقلام فعال حقوق را در اختیار موتور اسناد
// می‌گذارد تا فیش چاپی همان تفکیک شبیه‌ساز را نشان دهد، بدون import مستقیم ماژول.
type PayElementProvider interface {
	PayElements(ctx context.Context) []PayElement
	// PayParamValues مقادیر عددی پارامترهای فعال حقوق (کلید→مقدار) را می‌دهد تا
	// اسناد چاپی بدون دسترسی مستقیم به انبار ماژول حقوق، اقلام را حساب کنند.
	PayParamValues(ctx context.Context) map[string]string
}

// روش‌های محاسبهٔ قلم حقوق (هم‌معنی با مقادیر ماشین‌خوان ماژول حقوق).
const (
	ElementCalcFixed   = "fixed"
	ElementCalcPercent = "percent"
	ElementCalcFormula = "formula"
)

// ElementEnv محیط محاسبهٔ اقلام:
//   - Bases: مبنای اقلام درصدی (base/gross/housing/karana/constant).
//   - Vars: متغیرهای فرمول (ورودی شبیه‌سازی، پارامترهای نقطه‌دار، نتیجهٔ موتور قواعد).
//   - Canonical: مقدارهای مرجع برای اقلامی که فرمولشان توصیفی یا پله‌ای است (مثل مالیات).
type ElementEnv struct {
	Bases     map[string]float64
	Vars      map[string]float64
	Canonical map[string]float64
}

// BaseElementVars متغیرهای مشترک فرمول‌ها را از ورودی و نتیجهٔ محاسبه می‌سازد؛ این‌ها
// نام‌های ثابتی هستند که قلم‌های پارامتریک به آن‌ها ارجاع می‌دهند.
func BaseElementVars(input PayrollInput, result PayrollResult) map[string]float64 {
	return map[string]float64{
		"base":               float64(input.BaseSalary),
		"baseSalary":         float64(input.BaseSalary),
		"children":           float64(input.Children),
		"overtimeHours":      float64(input.OvertimeHours),
		"hourlyRate":         float64(input.OvertimeHourlyRate),
		"overtimeHourlyRate": float64(input.OvertimeHourlyRate),
		"otherDeductions":    float64(input.OtherDeductions),
		"gross":              float64(result.Gross),
		"taxableIncome":      float64(result.TaxableIncome),
		"insuranceBase":      float64(result.InsuranceBase),
		"insuranceEmployee":  float64(result.InsuranceEmployee),
		"insuranceEmployer":  float64(result.InsuranceEmployer),
		"overtimePay":        float64(result.OvertimePay),
		"tax":                float64(result.Tax),
		"net":                float64(result.Net),
	}
}

// ParamsVars نگاشت پارامترهای پیش‌فرض موتور قواعد به متغیرهای نقطه‌دار فرمول. اگر
// رکورد «پارامتر حقوق» یا پارامتر شرکت همین کلیدها را داشته باشد، caller آن‌ها را
// روی این مقادیر می‌نویسد.
func ParamsVars(params PayrollParams) map[string]float64 {
	return map[string]float64{
		"insurance.employeeRate": params.InsuranceEmployeeRate,
		"insurance.employerRate": params.InsuranceEmployerRate,
		"insurance.ceiling":      float64(params.InsuranceCeiling),
		"allowance.housing":      float64(params.HousingAllowance),
		"allowance.child":        float64(params.ChildAllowancePerChild),
		"tax.exemption":          float64(params.TaxExemption),
	}
}

// BaseElementBases مبناهای محاسبهٔ اقلام درصدی را می‌سازد. `mazdBase` همان مزد مبنا
// است که فوق‌العاده‌های درصدی روی آن سوار می‌شوند.
func BaseElementBases(input PayrollInput, params PayrollParams, result PayrollResult, karanaAmount, mazdBase float64) map[string]float64 {
	return map[string]float64{
		"base":     float64(input.BaseSalary),
		"gross":    float64(result.Gross),
		"housing":  float64(params.HousingAllowance),
		"karana":   karanaAmount,
		"constant": 0,
		"mazd":     mazdBase,
		"wage":     mazdBase,
	}
}

// CanonicalElementAmounts مقدارهای مرجع اقلامی که فرمولشان عددی نیست: مالیات پله‌ای و
// بیمهٔ سقف‌دار را موتور قواعد حساب کرده است، پس همان عدد مرجع بازگردانده می‌شود.
func CanonicalElementAmounts(result PayrollResult) map[string]float64 {
	return map[string]float64{
		"TAX":          float64(result.Tax),
		"INSURANCE_EE": float64(result.InsuranceEmployee),
	}
}

// ComputeElements اقلام حقوق را با محیط داده‌شده محاسبه می‌کند. این تنها نقطهٔ محاسبهٔ
// اقلام است؛ هم شبیه‌سازی سرور و هم فیش چاپی از همین تابع استفاده می‌کنند.
func ComputeElements(elements []PayElement, env ElementEnv) []PayElementLine {
	out := make([]PayElementLine, 0, len(elements))
	for _, el := range elements {
		line := PayElementLine{Code: el.Code, TitleFa: el.TitleFa, Category: el.Category, Calc: el.Calc}
		switch strings.TrimSpace(el.Calc) {
		case ElementCalcFixed:
			if v, ok := resolveElementValue(el.Rate, env.Vars); ok {
				line.Amount, line.Resolved, line.DetailFa = v, true, "مبلغ ثابت"
			} else {
				line.DetailFa = "مقدار ثابت تعیین نشده است"
			}
		case ElementCalcPercent:
			rate, okRate := resolveElementValue(el.Rate, env.Vars)
			basisVal, okBasis := env.Bases[strings.TrimSpace(el.Basis)]
			if okRate && okBasis {
				line.Amount, line.Resolved = rate*basisVal, true
				line.DetailFa = fmt.Sprintf("%.4g × %s", rate, BasisFa(el.Basis))
			} else {
				line.DetailFa = "ضریب یا مبنای محاسبه در دسترس نیست"
			}
		case ElementCalcFormula:
			formula := strings.TrimSpace(el.Formula)
			if v, err := EvalFormula(formula, env.Vars); err == nil {
				line.Amount, line.Resolved, line.DetailFa = v, true, "فرمول: "+formula
			} else if v, ok := env.Canonical[strings.TrimSpace(el.Code)]; ok {
				line.Amount, line.Resolved = v, true
				line.DetailFa = "محاسبهٔ موتور قواعد — " + formula
			} else {
				line.DetailFa = "فرمول با متغیرهای موجود قابل محاسبه نیست: " + formula
			}
		default:
			line.DetailFa = "روش محاسبه نامشخص"
		}
		out = append(out, line)
	}
	return out
}

// resolveElementValue مقدار یک فیلد را که می‌تواند کلید پارامتر/متغیر یا عدد باشد به
// عدد تبدیل می‌کند.
func resolveElementValue(raw string, vars map[string]float64) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if v, ok := vars[raw]; ok {
		return v, true
	}
	if v, err := strconv.ParseFloat(raw, 64); err == nil {
		return v, true
	}
	return 0, false
}

// BasisFa برچسب فارسی مبنای محاسبهٔ اقلام درصدی.
func BasisFa(basis string) string {
	switch strings.TrimSpace(basis) {
	case "base":
		return "پایهٔ حقوق"
	case "gross":
		return "جمع ناخالص"
	case "housing":
		return "حق مسکن"
	case "karana":
		return "کارانه"
	case "constant":
		return "مقدار ثابت"
	case "mazd", "wage":
		return "مزد مبنا"
	default:
		return basis
	}
}
