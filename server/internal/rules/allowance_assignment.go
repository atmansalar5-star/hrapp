// تخصیص فوق‌العاده‌های حقوق — «کدام کد فوق العاده، برای کدام کارمند یا گروه، با کدام
// روش محاسبه و با کدام مشمولیت کسور».
//
// چرا این لایه لازم شد: تا پیش از این، اقلام حقوق (ماژول «اجزای حقوق») فقط *تعریف* می‌شدند
// و کد رسمی «فوق العاده» جایی به کارمند گره نمی‌خورد؛ هر بار باید در همان محاسبه دستی وارد
// می‌شد. اکنون یک «تخصیص» ساخته می‌شود که یک کد رسمی را با روش محاسبه، مقدار/درصد/ضریب/
// فرمول، مشمولیت مالیات و بازنشستگی، و دامنهٔ اثر (یک کارمند یا گروهی از کارکنان) به هم
// می‌بندد؛ سپس موتور با دیدن پروندهٔ کارمند می‌گوید کدام تخصیص‌ها به او می‌خورد و چقدر
// می‌شود.
//
// قاعده‌ها اینجا زندگی می‌کنند (نه در ماژول و نه در رابط): تطبیق دامنه، پنجرهٔ زمانی،
// محاسبهٔ مبلغ و پرچم مشمولیت. ماژول فقط داده را نگه می‌دارد و رابط فقط نمایش می‌دهد.
package rules

import (
	"strconv"
	"strings"
)

// نوع دامنهٔ تخصیص: تخصیص می‌تواند به یک کارمند، یا به گروهی از کارکنان بخورد.
const (
	AllowanceTargetEmployee   = "employee"
	AllowanceTargetDepartment = "department"
	AllowanceTargetEmployment = "employmentType"
	AllowanceTargetJobGrade   = "jobGrade"
	AllowanceTargetPayGroup   = "payGroup"
	AllowanceTargetAll        = "all"
)

// AllowanceTargetKind یک نوع دامنه برای فهرست انتخابی فرم.
type AllowanceTargetKind struct {
	Key     string `json:"key"`
	TitleFa string `json:"titleFa"`
	NoteFa  string `json:"noteFa,omitempty"`
}

// AllowanceTargetKinds فهرست دامنه‌های پشتیبانی‌شده.
func AllowanceTargetKinds() []AllowanceTargetKind {
	return []AllowanceTargetKind{
		{Key: AllowanceTargetEmployee, TitleFa: "یک کارمند",
			NoteFa: "شناسهٔ کارمند (مثلاً EMP-004) — دقیق‌ترین دامنه."},
		{Key: AllowanceTargetDepartment, TitleFa: "یک واحد سازمانی",
			NoteFa: "نام یا شناسهٔ واحد در پروندهٔ کارمند."},
		{Key: AllowanceTargetEmployment, TitleFa: "یک نوع استخدام",
			NoteFa: "کلید نوع استخدام (مثلاً official_oil یا contractor_staff)."},
		{Key: AllowanceTargetJobGrade, TitleFa: "یک گروه شغلی",
			NoteFa: "کلید نردبان گروه‌های شغلی (employee، expert، senior_expert، …)."},
		{Key: AllowanceTargetPayGroup, TitleFa: "یک گروه نظام پرداخت نفت",
			NoteFa: "شمارهٔ گروه جدول ۳ (۱ = کارمند رسمی نفت، ۲ = پایهٔ سمت ۶ تا ۱۰، …)."},
		{Key: AllowanceTargetAll, TitleFa: "همهٔ کارکنان",
			NoteFa: "برای اقلام عمومی مثل حق اولاد؛ با احتیاط به کار ببرید."},
	}
}

// AllowanceTargetTitleFa عنوان فارسی یک دامنه.
func AllowanceTargetTitleFa(kind string) string {
	for _, k := range AllowanceTargetKinds() {
		if k.Key == kind {
			return k.TitleFa
		}
	}
	return strings.TrimSpace(kind)
}

// AllowanceTarget دامنهٔ اثر یک تخصیص.
type AllowanceTarget struct {
	Kind  string `json:"kind"`
	Value string `json:"value,omitempty"`
}

// AllowanceEmployee تصویر کوچک کارمند که برای تطبیق دامنه لازم است.
type AllowanceEmployee struct {
	ID             string `json:"id"`
	DepartmentName string `json:"departmentName,omitempty"`
	EmploymentType string `json:"employmentType,omitempty"`
	JobGrade       string `json:"jobGrade,omitempty"`
	PayGroup       int    `json:"payGroup,omitempty"`
}

// Matches آیا این دامنه به کارمند داده‌شده می‌خورد.
//
// تطبیق با معیار دقیق و ساده انجام می‌شود: کلید یا نام همان‌گونه که در پرونده است. برای
// نوع استخدام و گروه شغلی، کلید لاتین و عنوان فارسی هر دو پذیرفته می‌شوند تا دادهٔ قدیمی و
// فرم تازه هر دو کار کنند.
func (t AllowanceTarget) Matches(employee AllowanceEmployee) bool {
	value := strings.TrimSpace(t.Value)
	switch strings.TrimSpace(t.Kind) {
	case AllowanceTargetAll:
		return true
	case AllowanceTargetEmployee:
		return value != "" && value == strings.TrimSpace(employee.ID)
	case AllowanceTargetDepartment:
		return value != "" && (value == strings.TrimSpace(employee.DepartmentName) ||
			value == strings.TrimSpace(employee.EmploymentType)) && strings.TrimSpace(employee.DepartmentName) != ""
	case AllowanceTargetEmployment:
		// کلید نوع استخدام؛ عنوان فارسی اینجا تطبیق داده نمی‌شود چون بستهٔ قواعد به فهرست
		// انواع استخدام وابسته نیست (وگرنه حلقهٔ import ساخته می‌شود) — فرم همان کلید را
		// می‌فرستد.
		return value != "" && value == employee.EmploymentType
	case AllowanceTargetJobGrade:
		return value != "" && value == employee.JobGrade
	case AllowanceTargetPayGroup:
		if value == "" {
			return false
		}
		want, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return false
		}
		return want > 0 && want == employee.PayGroup
	default:
		return false
	}
}

// روش‌های محاسبهٔ یک قلم تخصیص‌یافته.
const (
	// AllowanceCalcFixed مبلغ ثابت ماهانه (ریال).
	AllowanceCalcFixed = "fixed"
	// AllowanceCalcOnce یک‌بار مرتبه (تنها در ماه شروع اعتبار).
	AllowanceCalcOnce = "once"
	// AllowanceCalcPercent درصدی از یک مبنا.
	AllowanceCalcPercent = "percent"
	// AllowanceCalcCoefficient ضریب در یک مبنا.
	AllowanceCalcCoefficient = "coefficient"
	// AllowanceCalcFormula فرمول محاسباتی (متغیرها از مبناها و محیط می‌آید).
	AllowanceCalcFormula = "formula"
)

// نوع قلم تخصیص: مزایا (بستانکار) یا کسور (بدهکار).
const (
	// AllowanceCategoryEarning قلم پرداختنی.
	AllowanceCategoryEarning = "earning"
	// AllowanceCategoryDeduction قلم کسور.
	AllowanceCategoryDeduction = "deduction"
)

// AllowanceCalcKinds روش‌های محاسبهٔ یک قلم تخصیص‌یافته.
func AllowanceCalcKinds() []AllowanceTargetKind {
	return []AllowanceTargetKind{
		{Key: AllowanceCalcFixed, TitleFa: "ماهانهٔ ثابت (ریال)",
			NoteFa: "مبلغ ثابت ماهانه؛ تا پایان اعتبار تخصیص، هر ماه پرداخت/کسر می‌شود."},
		{Key: AllowanceCalcOnce, TitleFa: "یک‌بار مرتبه (ریال)",
			NoteFa: "فقط در ماه شروع اعتبار پرداخت می‌شود."},
		{Key: AllowanceCalcPercent, TitleFa: "درصدی از یک مبنا",
			NoteFa: "درصد (مثلاً ۱۵) × مبنای انتخابی."},
		{Key: AllowanceCalcCoefficient, TitleFa: "ضریب در یک مبنا",
			NoteFa: "ضریب (مثلاً ۱٫۵) × مبنای انتخابی."},
		{Key: AllowanceCalcFormula, TitleFa: "فرمول محاسباتی",
			NoteFa: "فرمول حسابی با متغیرها؛ مثل BASE*0.15+CHILDREN*1000000."},
	}
}

// AllowanceCalcTitleFa عنوان فارسی روش محاسبه.
func AllowanceCalcTitleFa(calc string) string {
	for _, k := range AllowanceCalcKinds() {
		if k.Key == calc {
			return k.TitleFa
		}
	}
	return strings.TrimSpace(calc)
}

// AllowanceBasisKeys مبناهای قابل انتخاب برای اقلام درصدی/ضریبی.
func AllowanceBasisKeys() []AllowanceTargetKind {
	return []AllowanceTargetKind{
		{Key: "base", TitleFa: "پایهٔ حقوق (A)"},
		{Key: "allowanceBase", TitleFa: "مبنای فوق‌العاده‌ها (A+B)"},
		{Key: "attractionBase", TitleFa: "مبنای جذب (A+B+C)"},
		{Key: "gross", TitleFa: "جمع ناخالص ماه"},
		{Key: "insuranceBase", TitleFa: "مبنای بیمه"},
		{Key: "constant", TitleFa: "مقدار ثابت واردشده در فیلد مقدار"},
	}
}

// AllowanceBasisTitleFa عنوان فارسی یک مبنا.
func AllowanceBasisTitleFa(basis string) string {
	for _, b := range AllowanceBasisKeys() {
		if b.Key == basis {
			return b.TitleFa
		}
	}
	return strings.TrimSpace(basis)
}

// AllowanceAssignment یک تخصیص فوق العاده.
type AllowanceAssignment struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	TitleFa  string `json:"titleFa,omitempty"`
	Category string `json:"category"`
	Calc     string `json:"calc"`
	// Rate مقدار/درصد/ضریب؛ در روش فرمولی نادیده گرفته می‌شود.
	Rate string `json:"rate,omitempty"`
	// Basis مبنای اقلام درصدی و ضریبی.
	Basis   string `json:"basis,omitempty"`
	Formula string `json:"formula,omitempty"`
	// Taxable مشمول مالیات و Insured مشمول بیمه/بازنشستگی.
	Taxable bool `json:"taxable"`
	Insured bool `json:"insured"`
	// ShowOnPayslip نمایش این قلم در فیش حقوقی و ShowInReports درج آن در گزارش‌های چاپی.
	// خالی‌بودن در فرم یعنی «بله» (رفتار گذشته حفظ می‌شود).
	ShowOnPayslip bool `json:"showOnPayslip"`
	ShowInReports bool `json:"showInReports"`
	// ReportCode کد گزارش (کدال / کد رسمی گزارش سازمان) برای درج در گزارش‌های رسمی.
	ReportCode string `json:"reportCode,omitempty"`
	// Target دامنهٔ اثر: یک کارمند یا گروهی از کارکنان.
	Target AllowanceTarget `json:"target"`
	// EffectiveFrom ماه شروع (YYYY/MM) و EffectiveTo ماه پایان (خالی = تا اطلاع).
	EffectiveFrom string `json:"effectiveFrom"`
	EffectiveTo   string `json:"effectiveTo,omitempty"`
	NoteFa        string `json:"noteFa,omitempty"`
	Status        string `json:"status,omitempty"`
}

// AllowanceEnv محیط محاسبه: مقدار مبناها، متغیرهای فرمول و ماه محاسبه.
type AllowanceEnv struct {
	Bases  map[string]float64 `json:"bases,omitempty"`
	Vars   map[string]float64 `json:"vars,omitempty"`
	Period string             `json:"period,omitempty"`
}

// AllowanceLine سطر نتیجهٔ یک تخصیص برای یک کارمند.
type AllowanceLine struct {
	Code          string  `json:"code"`
	TitleFa       string  `json:"titleFa"`
	Category      string  `json:"category"`
	CategoryFa    string  `json:"categoryFa"`
	Calc          string  `json:"calc"`
	CalcFa        string  `json:"calcFa"`
	Amount        int64   `json:"amount"`
	Rate          float64 `json:"rate,omitempty"`
	Basis         string  `json:"basis,omitempty"`
	BasisFa       string  `json:"basisFa,omitempty"`
	Taxable       bool    `json:"taxable"`
	Insured       bool    `json:"insured"`
	ShowOnPayslip bool    `json:"showOnPayslip"`
	ShowInReports bool    `json:"showInReports"`
	ReportCode    string  `json:"reportCode,omitempty"`
	Resolved      bool    `json:"resolved"`
	SourceFa      string  `json:"sourceFa"`
	AssignmentID  string  `json:"assignmentId,omitempty"`
	DetailFa      string  `json:"detailFa,omitempty"`
}

// ValidateAllowanceAssignment ورودی یک تخصیص را می‌سنجد و خطاهای فیلدبه‌فیلد می‌دهد.
func ValidateAllowanceAssignment(in AllowanceAssignment) map[string]string {
	errs := map[string]string{}

	code := strings.TrimSpace(in.Code)
	if code == "" {
		errs["code"] = "کد فوق العاده الزامی است."
	} else if _, ok := PayslipCodeByNumber(code); !ok {
		errs["code"] = "کد فوق العاده در جدول رسمی سازمان نیست."
	}

	category := strings.TrimSpace(in.Category)
	if category == "" {
		// بی‌نوع بودن قلم پذیرفته نیست؛ ولی می‌توان از کد رسمی استنتاج کرد.
		if official, ok := PayslipCodeByNumber(code); ok {
			switch official.Kind {
			case PayslipKindEarning:
				category = AllowanceCategoryEarning
			default:
				category = AllowanceCategoryDeduction
			}
		} else {
			errs["category"] = "نوع قلم (مزایا یا کسور) الزامی است."
		}
	}

	calc := strings.TrimSpace(in.Calc)
	switch calc {
	case AllowanceCalcFixed, AllowanceCalcOnce:
		if !isPositiveNumberText(in.Rate) {
			errs["rate"] = "برای مبلغ ثابت یا یک‌بار مرتبه، مقدار مثبت (ریال) الزامی است."
		}
	case AllowanceCalcPercent, AllowanceCalcCoefficient:
		if !isPositiveNumberText(in.Rate) {
			errs["rate"] = "برای محاسبهٔ درصدی یا ضریبی، مقدار عددی الزامی است."
		}
		if strings.TrimSpace(in.Basis) == "" {
			errs["basis"] = "مبنای محاسبه الزامی است."
		}
	case AllowanceCalcFormula:
		expr := strings.TrimSpace(in.Formula)
		if expr == "" {
			errs["formula"] = "فرمول محاسبه الزامی است."
		} else if _, err := EvalFormula(expr, trialFormulaVars(expr)); err != nil {
			if _, ok := err.(*FormulaError); ok && strings.Contains(err.Error(), "متغیر") {
				// متغیر ناشناخته در آزمون: خطای فرمول نیست، فقط متغیر را نمی‌شناسیم.
			} else {
				errs["formula"] = "فرمول معتبر نیست: " + err.Error()
			}
		}
	default:
		errs["calc"] = "روش محاسبه الزامی است (مبلغ ثابت، یک‌بار مرتبه، درصدی، ضریبی یا فرمولی)."
	}

	if !isMonthText(in.EffectiveFrom) {
		errs["effectiveFrom"] = "ماه شروع اعتبار باید به قالب YYYY/MM باشد."
	}
	if to := strings.TrimSpace(in.EffectiveTo); to != "" && !isMonthText(to) {
		errs["effectiveTo"] = "ماه پایان اعتبار باید به قالب YYYY/MM باشد."
	}
	// کد گزارش (کدال/کد رسمی گزارش) اختیاری است؛ اگر آمد باید کوتاه و ماشین‌خوان باشد
	// تا در گزارش رسمی بی‌ابهام درج شود.
	if code := strings.TrimSpace(in.ReportCode); code != "" && !isReportCodeText(code) {
		errs["reportCode"] = "کد گزارش باید فقط حرف و رقم باشد (حداکثر ۲۰ نویسه)."
	}

	kind := strings.TrimSpace(in.Target.Kind)
	if kind == "" {
		errs["target.kind"] = "دامنهٔ تخصیص الزامی است."
	} else {
		known := false
		for _, k := range AllowanceTargetKinds() {
			if k.Key == kind {
				known = true
				break
			}
		}
		if !known {
			errs["target.kind"] = "دامنهٔ انتخابی شناخته‌شده نیست."
		}
		if kind != AllowanceTargetAll && strings.TrimSpace(in.Target.Value) == "" {
			errs["target.value"] = "برای این دامنه، مقدار (کارمند/واحد/نوع استخدام/گروه) الزامی است."
		}
	}

	return errs
}

// ResolveAllowanceAssignments تخصیص‌های فعال یک کارمند را در ماه داده‌شده حساب می‌کند.
//
// ترتیب کار: نخست دامنه و پنجرهٔ زمانی سنجیده می‌شود، سپس مبلغ با روش همان تخصیص حساب
// می‌شود. هر سطر می‌گوید مشمول مالیات و بیمه است یا نه، و منبعش (شناسهٔ تخصیص) را هم
// برمی‌گرداند تا بازبینی سند ممکن باشد.
func ResolveAllowanceAssignments(assignments []AllowanceAssignment, employee AllowanceEmployee, env AllowanceEnv) []AllowanceLine {
	out := []AllowanceLine{}
	for _, a := range assignments {
		if status := strings.TrimSpace(a.Status); status != "" && status != "active" {
			continue
		}
		if !a.Target.Matches(employee) {
			continue
		}
		if !allowanceInWindow(a, env.Period) {
			continue
		}
		out = append(out, resolveAllowanceAssignment(a, env))
	}
	return out
}

// AllowanceTotals جمع سطرهای تخصیص به تفکیک نوع و مشمولیت.
type AllowanceTotals struct {
	Earnings        int64 `json:"earnings"`
	Deductions      int64 `json:"deductions"`
	TaxableEarnings int64 `json:"taxableEarnings"`
	InsuredEarnings int64 `json:"insuredEarnings"`
	TaxableDeduct   int64 `json:"taxableDeductions"`
	// PayslipEarnings/PayslipDeductions جمع اقلامی که در فیش حقوقی نمایش داده می‌شوند.
	PayslipEarnings   int64 `json:"payslipEarnings"`
	PayslipDeductions int64 `json:"payslipDeductions"`
	// ReportEarnings/ReportDeductions جمع اقلامی که در گزارش‌های چاپی درج می‌شوند.
	ReportEarnings   int64 `json:"reportEarnings"`
	ReportDeductions int64 `json:"reportDeductions"`
	Lines            int   `json:"lines"`
}

// SumAllowanceLines جمع سطرها را می‌سازد.
func SumAllowanceLines(lines []AllowanceLine) AllowanceTotals {
	var t AllowanceTotals
	for _, l := range lines {
		t.Lines++
		if l.Category == AllowanceCategoryDeduction {
			t.Deductions += l.Amount
			if l.Taxable {
				t.TaxableDeduct += l.Amount
			}
			if l.ShowOnPayslip {
				t.PayslipDeductions += l.Amount
			}
			if l.ShowInReports {
				t.ReportDeductions += l.Amount
			}
			continue
		}
		t.Earnings += l.Amount
		if l.Taxable {
			t.TaxableEarnings += l.Amount
		}
		if l.Insured {
			t.InsuredEarnings += l.Amount
		}
		if l.ShowOnPayslip {
			t.PayslipEarnings += l.Amount
		}
		if l.ShowInReports {
			t.ReportEarnings += l.Amount
		}
	}
	return t
}

// AllowancePayslipLines سطرهایی که باید روی فیش حقوقی دیده شوند.
//
// چرا جدا: تخصیص می‌تواند برای گزارش‌های مالی و رسمی فعال باشد ولی روی فیش کارمند نیاید
// (مثل اقلام رفاهی که سازمان نخواهد در فیش چاپ شود)؛ این فیلتر همان تصمیم فرم است.
func AllowancePayslipLines(lines []AllowanceLine) []AllowanceLine {
	out := []AllowanceLine{}
	for _, l := range lines {
		if l.ShowOnPayslip {
			out = append(out, l)
		}
	}
	return out
}

// AllowanceReportLines سطرهایی که در گزارش‌های چاپی/رسمی درج می‌شوند.
func AllowanceReportLines(lines []AllowanceLine) []AllowanceLine {
	out := []AllowanceLine{}
	for _, l := range lines {
		if l.ShowInReports {
			out = append(out, l)
		}
	}
	return out
}

// allowanceInWindow آیا ماه محاسبه در پنجرهٔ اعتبار تخصیص است.
//
// تاریخ‌ها هم به شکل «YYYY/MM» و هم «YYYY/MM/DD» پذیرفته می‌شوند (فرم‌ها و پارامترهای
// سازمانی هر دو شکل را دارند) و پیش از مقایسه به ماه تبدیل می‌شوند؛ وگرنه مقایسهٔ رشته‌ای
// «1405/07» با «1405/07/01» نتیجهٔ غلط می‌داد.
func allowanceInWindow(a AllowanceAssignment, period string) bool {
	month := allowanceMonth(period)
	if month == "" {
		return true
	}
	from := allowanceMonth(a.EffectiveFrom)
	if from != "" && month < from {
		return false
	}
	if to := allowanceMonth(a.EffectiveTo); to != "" && month > to {
		return false
	}
	// «یک‌بار مرتبه» فقط در ماه شروع اعتبار پرداخت می‌شود.
	if strings.TrimSpace(a.Calc) == AllowanceCalcOnce && from != "" && month != from {
		return false
	}
	return true
}

// allowanceMonth ماه «YYYY/MM» را از ورودی «YYYY/MM» یا «YYYY/MM/DD» بیرون می‌کشد و ارقام
// فارسی/عربی را هم می‌پذیرد؛ ورودی نامعتبر رشتهٔ خالی می‌دهد.
func allowanceMonth(raw string) string {
	value := normalizeAllowanceDigits(strings.TrimSpace(raw))
	if len(value) != 7 && len(value) != 10 {
		return ""
	}
	if value[4] != '/' || (len(value) == 10 && value[7] != '/') {
		return ""
	}
	month := value[5:7]
	if month < "01" || month > "12" {
		return ""
	}
	year := value[:4]
	if year < "1300" || year > "1500" {
		return ""
	}
	return value[:7]
}

// normalizeAllowanceDigits ارقام فارسی/عربی را به لاتین برمی‌گرداند.
func normalizeAllowanceDigits(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= '۰' && r <= '۹' {
			return '0' + (r - '۰')
		}
		if r >= '٠' && r <= '٩' {
			return '0' + (r - '٠')
		}
		return r
	}, value)
}

// resolveAllowanceAssignment مبلغ یک تخصیص را حساب می‌کند.
func resolveAllowanceAssignment(a AllowanceAssignment, env AllowanceEnv) AllowanceLine {
	category := strings.TrimSpace(a.Category)
	if category == "" {
		category = AllowanceCategoryEarning
		if official, ok := PayslipCodeByNumber(a.Code); ok && official.Kind != PayslipKindEarning {
			category = AllowanceCategoryDeduction
		}
	}
	title := strings.TrimSpace(a.TitleFa)
	if title == "" {
		if official, ok := PayslipCodeByNumber(a.Code); ok {
			title = official.TitleFa
		}
	}
	line := AllowanceLine{
		Code:          strings.TrimSpace(a.Code),
		TitleFa:       title,
		Category:      category,
		CategoryFa:    CategoryFa(category),
		Calc:          strings.TrimSpace(a.Calc),
		CalcFa:        AllowanceCalcTitleFa(a.Calc),
		Taxable:       a.Taxable,
		Insured:       a.Insured,
		ShowOnPayslip: a.ShowOnPayslip,
		ShowInReports: a.ShowInReports,
		ReportCode:    strings.TrimSpace(a.ReportCode),
		AssignmentID:  a.ID,
		SourceFa:      "تخصیص " + a.Target.Kind + ": " + a.Target.Value,
	}
	if a.Target.Kind == AllowanceTargetAll {
		line.SourceFa = "تخصیص برای همهٔ کارکنان"
	}

	basisValue := 0.0
	if basis := strings.TrimSpace(a.Basis); basis != "" {
		line.Basis, line.BasisFa = basis, AllowanceBasisTitleFa(basis)
		basisValue = env.Bases[basis]
	}
	rate, _ := strconv.ParseFloat(strings.TrimSpace(a.Rate), 64)

	switch line.Calc {
	case AllowanceCalcFixed, AllowanceCalcOnce:
		line.Amount = int64(rate)
		line.Resolved = true
		line.DetailFa = "مبلغ " + IntFa(int64(rate)) + " ریال — " + AllowanceCalcTitleFa(line.Calc)
	case AllowanceCalcPercent:
		line.Rate = rate
		line.Amount = int64(basisValue * rate / 100)
		line.Resolved = true
		line.DetailFa = numberFa(rate) + "٪ × " + AllowanceBasisTitleFa(line.Basis) +
			" (" + moneyFa(basisValue) + " ریال)"
	case AllowanceCalcCoefficient:
		line.Rate = rate
		line.Amount = int64(basisValue * rate)
		line.Resolved = true
		line.DetailFa = numberFa(rate) + " × " + AllowanceBasisTitleFa(line.Basis) +
			" (" + moneyFa(basisValue) + " ریال)"
	case AllowanceCalcFormula:
		vars := map[string]float64{}
		for k, v := range env.Bases {
			vars[strings.ToUpper(k)] = v
		}
		for k, v := range env.Vars {
			vars[k] = v
		}
		if value, err := EvalFormula(strings.TrimSpace(a.Formula), vars); err == nil {
			line.Amount = int64(value)
			line.Resolved = true
			line.DetailFa = "فرمول: " + a.Formula
		} else {
			line.DetailFa = "فرمول با متغیرهای موجود قابل محاسبه نیست: " + err.Error()
		}
	}
	return line
}

// CategoryFa عنوان فارسی نوع قلم.
func CategoryFa(category string) string {
	switch category {
	case AllowanceCategoryEarning:
		return "مزایا (بستانکار)"
	case AllowanceCategoryDeduction:
		return "کسور (بدهکار)"
	default:
		return category
	}
}

// trialFormulaVars برای اعتبارسنجی فرمول: هر متغیری که در عبارت آمده با ۱ جایگزین می‌شود
// تا فقط «نماد نامعتبر» و ساختار فرمول سنجیده شود، نه شناخته‌بودن متغیرها.
func trialFormulaVars(expr string) map[string]float64 {
	out := map[string]float64{}
	for _, name := range FormulaVariables(expr) {
		out[name] = 1
	}
	return out
}

// isReportCodeText آیا متن یک کد گزارش معتبر است (حرف/رقم لاتین یا فارسی، حداکثر ۲۰ نویسه).
func isReportCodeText(raw string) bool {
	value := normalizeAllowanceDigits(strings.TrimSpace(raw))
	if value == "" || len([]rune(value)) > 20 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

// isPositiveNumberText آیا متن یک عدد مثبت است.
func isPositiveNumberText(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	return err == nil && parsed > 0
}

// isMonthText آیا متن به قالب ماه شمسی است (YYYY/MM یا YYYY/MM/DD با ارقام فارسی).
func isMonthText(raw string) bool { return allowanceMonth(raw) != "" }
