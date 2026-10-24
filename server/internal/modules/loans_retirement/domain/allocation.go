// منطق دامنهٔ ماژول «loans_retirement» — «allocation»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"strconv"
	"strings"

	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// این فایل «تخصیص وام» را اعلام می‌کند: واحد وام یک نوع وام را با مبلغ و اقساط مشخص
// به یک یا چند کارمند تخصیص می‌دهد و سرور مبالغ کارمزد و هر قسط را از دامنه می‌سازد.
//
// چرا در دامنه و نه در UI: نرخ کارمزد، سقف مبلغ و سقف اقساط «سیاست سازمان»‌اند. رابط وب
// فقط نمایش می‌دهد و هیچ مبلغی از خودش نمی‌سازد؛ اگر کارمزد یا سقفی عوض شود، همان فرم و
// همان مبالغ بدون تغییر کد به‌روز می‌شوند.

// LoanGrantRequest درخواست تخصیص (اعطای) وام به یک یا چند کارمند با شرایط یکسان.
//
// Amount مبلغ وام «هر نفر» است و Installments تعداد اقساط؛ اگر Installments صفر بماند،
// سقف اقساط همان نوع وام اعمال می‌شود.
type LoanGrantRequest struct {
	LoanType            string
	Bank                string
	Management          string
	Amount              int64
	Installments        int
	ReceivedDate        string
	Purpose             string
	GuarantorName       string
	GuarantorNationalID string
	GuarantorRelation   string
	// SupplementOf شناسهٔ وام پایه؛ فقط برای انواع «متمم» الزامی است.
	SupplementOf string
	// Unit واحد سازمانی متقاضی (اختیاری) از کاتالوگ مرجع واحدها.
	Unit string
}

// GrantPolicy فهرست‌های اضافهٔ زمان‌اجرا که اعتبارسنجی باید بپذیرد.
//
// چرا جدا از کاتالوگ پایه: بانک‌ها و مدیریت‌های پایه در دامنه اعلام می‌شوند، ولی پنل
// تنظیمات می‌تواند قلم تازه اضافه کند. اعتبارسنجی هر دو را می‌پذیرد تا افزودن قلم در
// پنل، بی‌نیاز از تغییر کد، در فرم وام کار کند. نگاشت کلید ← عنوان فارسی است.
type GrantPolicy struct {
	ExtraBanks       map[string]string
	ExtraManagements map[string]string
}

// LoanTerms شرایط مالی محاسبه‌شدهٔ یک وام: نرخ و مبلغ کارمزد و مبلغ هر قسط.
type LoanTerms struct {
	FeeRatePercent    float64
	FeeAmount         int64
	Installments      int
	InstallmentAmount int64
	// MaxAmount و MaxInstallments سقف‌های مصوب همان نوع وام (برای نمایش در فرم).
	MaxAmount       int64
	MaxInstallments int
}

// ValidateGrantRequest قواعد تخصیص وام — همان قواعد دامنه، پیش از ساخت هر رکورد.
//
// قاعده‌ها: نوع وام باید از فهرست اعلامی باشد، مبلغ مثبت و در سقف مصوب، اقساط در سقف
// همان نوع، بانک عامل (اگر آمده) از شبکهٔ مجاز، مدیریت متقاضی در جدول سهمیه، تاریخ
// دریافت شمسی، و وام متمم بدون وام پایه ثبت نشود.
func ValidateGrantRequest(req LoanGrantRequest) error {
	return ValidateGrantRequestWith(req, GrantPolicy{})
}

// ValidateGrantRequestWith همان قواعد، به‌علاوهٔ اقلام کاتالوگ مرجع که پنل تنظیمات
// اضافه کرده است.
func ValidateGrantRequestWith(req LoanGrantRequest, extra GrantPolicy) error {
	loanType := strings.TrimSpace(req.LoanType)
	t, ok := LoanTypeByKey(loanType)
	if !ok {
		return &recordkit.RuleError{Message: "انتخاب نوع وام از فهرست انواع وام الزامی است."}
	}
	if req.Amount <= 0 {
		return &recordkit.RuleError{Message: "مبلغ وام باید عددی مثبت باشد."}
	}
	if t.MaxAmount > 0 && req.Amount > t.MaxAmount {
		return &recordkit.RuleError{Message: "مبلغ وام از سقف مصوب «" + t.TitleFa + "» بیشتر است."}
	}
	if req.Installments < 0 {
		return &recordkit.RuleError{Message: "تعداد اقساط باید عددی نامنفی باشد."}
	}
	if t.MaxInstallments > 0 && req.Installments > t.MaxInstallments {
		return &recordkit.RuleError{Message: "تعداد اقساط از سقف مصوب «" + t.TitleFa + "» بیشتر است."}
	}
	if bank := strings.TrimSpace(req.Bank); bank != "" && !ValidBank(bank) {
		if _, ok := extra.ExtraBanks[bank]; !ok {
			return &recordkit.RuleError{Message: "بانک عامل انتخابی در فهرست شبکه‌های مجاز نیست."}
		}
	}
	if m := strings.TrimSpace(req.Management); m != "" && !ValidQuotaUnit(m) {
		if _, ok := extra.ExtraManagements[m]; !ok {
			return &recordkit.RuleError{Message: "مدیریت «" + m + "» در جدول سهمیهٔ تسهیلات شناخته نشد."}
		}
	}
	if _, err := calendar.Parse(strings.TrimSpace(req.ReceivedDate)); err != nil {
		return &recordkit.RuleError{Message: "تاریخ دریافت وام باید به قالب شمسی YYYY/MM/DD باشد."}
	}
	if t.IsSupplement && strings.TrimSpace(req.SupplementOf) == "" {
		return &recordkit.RuleError{Message: "برای وام متمم باید شناسهٔ وام پایه (متممِ وام) مشخص شود."}
	}
	return nil
}

// ComputeLoanTerms شرایط مالی وام را از نوع وام و مبلغ آن می‌سازد.
//
// قاعدهٔ محاسبه (تصریح‌شده تا در گزارش‌ها قابل ردیابی باشد):
//
//	مبلغ کارمزد   = مبلغ وام × نرخ کارمزد همان نوع وام / ۱۰۰
//	مبلغ هر قسط   = (مبلغ وام + مبلغ کارمزد) ÷ تعداد اقساط، گرد به ریال به بالا
//
// تعداد اقساط خالی یعنی «سقف اقساط همان نوع وام»؛ اگر آن هم صفر باشد، ۱۲ قسط پیش‌فرض
// می‌شود تا تقسیم بر صفر رخ ندهد.
func ComputeLoanTerms(req LoanGrantRequest) (LoanTerms, error) {
	return ComputeLoanTermsWith(req, GrantPolicy{})
}

// ComputeLoanTermsWith همان محاسبه، با پذیرش اقلام کاتالوگ مرجع زمان‌اجرا.
func ComputeLoanTermsWith(req LoanGrantRequest, extra GrantPolicy) (LoanTerms, error) {
	if err := ValidateGrantRequestWith(req, extra); err != nil {
		return LoanTerms{}, err
	}
	t, _ := LoanTypeByKey(strings.TrimSpace(req.LoanType))

	installments := req.Installments
	if installments <= 0 {
		installments = t.MaxInstallments
	}
	if installments <= 0 {
		installments = 12
	}

	feeAmount := int64(float64(req.Amount) * t.FeeRatePercent / 100)
	total := req.Amount + feeAmount
	perInstallment := total / int64(installments)
	if total%int64(installments) != 0 {
		perInstallment++ // گرد به ریال به بالا تا جمع اقساط از مبلغ وام کم نیاید.
	}

	return LoanTerms{
		FeeRatePercent:    t.FeeRatePercent,
		FeeAmount:         feeAmount,
		Installments:      installments,
		InstallmentAmount: perInstallment,
		MaxAmount:         t.MaxAmount,
		MaxInstallments:   t.MaxInstallments,
	}, nil
}

// LoanFields فیلدهای رکورد وام را از درخواست و شرایط محاسبه‌شده می‌سازد.
//
// یک‌جا ساخته می‌شود تا ذخیره‌سازی فقط یک مسیر داشته باشد و پیش‌نمایش با ثبت هرگز از هم
// جدا نشود (همان قاعده‌ای که ماژول پاداش هم دارد).
func LoanFields(req LoanGrantRequest, terms LoanTerms) map[string]string {
	return map[string]string{
		"loanType":            strings.TrimSpace(req.LoanType),
		"bank":                strings.TrimSpace(req.Bank),
		"management":          strings.TrimSpace(req.Management),
		"amount":              itoa64(req.Amount),
		"installments":        itoa(terms.Installments),
		"installmentAmount":   itoa64(terms.InstallmentAmount),
		"feeRate":             trimFloat(terms.FeeRatePercent),
		"feeAmount":           itoa64(terms.FeeAmount),
		"supplementOf":        strings.TrimSpace(req.SupplementOf),
		"guarantorName":       strings.TrimSpace(req.GuarantorName),
		"guarantorNationalId": strings.TrimSpace(req.GuarantorNationalID),
		"guarantorRelation":   strings.TrimSpace(req.GuarantorRelation),
		"receivedDate":        strings.TrimSpace(req.ReceivedDate),
		"purpose":             strings.TrimSpace(req.Purpose),
		"unit":                strings.TrimSpace(req.Unit),
	}
}

func itoa(v int) string { return strconv.Itoa(v) }

func itoa64(v int64) string { return strconv.FormatInt(v, 10) }

// trimFloat عدد اعشاری را بی‌صفرهای انتهایی به رشته تبدیل می‌کند تا فیلد «نرخ کارمزد»
// در داده و API خوانا بماند (۴ و ۲.۵، نه ۴.۰۰۰۰۰۰).
func trimFloat(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
