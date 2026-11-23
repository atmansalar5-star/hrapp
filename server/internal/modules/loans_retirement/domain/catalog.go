// Package domain دامنهٔ ماژول امور وام و بازنشستگی.
//
// درخواست وام، قسط و پروندهٔ بازنشستگی هر یک یک «نوع رکورد» با گردش‌کار خودش است.
// فهرست انواع وام، بانک‌های عامل، نرخ کارمزد و معیارهای امتیازبندی تخصیص وام هم اینجا
// اعلام می‌شوند؛ سرور تنها منبع حقیقت است و UI هیچ نرخ یا معیاری از خودش نمی‌سازد.
package domain

import (
	"strings"

	"hrms/internal/rules"
)

// LoanType یک نوع وام با پارامترهای پیش‌فرض آن.
//
// IsSupplement یعنی این وام «متمم» یک وام پایه است و BaseKey نشان می‌دهد متممِ کدام
// وام است (مثل متمم ودیعهٔ مسکن). FeeRatePercent نرخ کارمزد پیشنهادی و MaxInstallments
// و MaxAmount سقف‌های مصوب هستند؛ محاسبهٔ مالی همچنان در سرور انجام می‌شود.
type LoanType struct {
	Key             string
	TitleFa         string
	IsSupplement    bool
	BaseKey         string
	FeeRatePercent  float64
	MaxInstallments int
	MaxAmount       int64
	PurposeFa       string
}

// loanTypes فهرست مصوب انواع وام کارکنان.
//
// عنوان فارسی از بستهٔ rules می‌آید تا فیش حقوقی و صورت حساب وام همان نام را نشان دهند؛
// نرخ کارمزد و سقف‌های مصوب سیاست سازمان‌اند و همین‌جا می‌مانند.
func loanTypes() []LoanType {
	return []LoanType{
		{Key: "housingDeposit", TitleFa: rules.LoanTypeTitleFa("housingDeposit"), FeeRatePercent: 4, MaxInstallments: 60, MaxAmount: 1_000_000_000, PurposeFa: "ودیعهٔ مسکن"},
		{Key: "housingDepositSupplement", TitleFa: rules.LoanTypeTitleFa("housingDepositSupplement"), IsSupplement: true, BaseKey: "housingDeposit", FeeRatePercent: 4, MaxInstallments: 36, MaxAmount: 300_000_000, PurposeFa: "تکمیل ودیعهٔ مسکن"},
		{Key: "bank", TitleFa: rules.LoanTypeTitleFa("bank"), FeeRatePercent: 18, MaxInstallments: 36, MaxAmount: 500_000_000, PurposeFa: "وام بانکی"},
		{Key: "emergency", TitleFa: rules.LoanTypeTitleFa("emergency"), FeeRatePercent: 0, MaxInstallments: 12, MaxAmount: 100_000_000, PurposeFa: "هزینهٔ اضطراری"},
		{Key: "socialWork", TitleFa: rules.LoanTypeTitleFa("socialWork"), FeeRatePercent: 2, MaxInstallments: 24, MaxAmount: 150_000_000, PurposeFa: "حمایت مددکاری"},
		{Key: "firstDegreeDeath", TitleFa: rules.LoanTypeTitleFa("firstDegreeDeath"), FeeRatePercent: 0, MaxInstallments: 24, MaxAmount: 80_000_000, PurposeFa: "هزینهٔ فوت بستگان درجه یک"},
		{Key: "unforeseen", TitleFa: rules.LoanTypeTitleFa("unforeseen"), FeeRatePercent: 1, MaxInstallments: 36, MaxAmount: 200_000_000, PurposeFa: "جبران خسارت حوادث مترقبه"},
	}
}

// LoanTypes فهرست انواع وام برای رابط و گزارش‌ها.
func LoanTypes() []LoanType { return loanTypes() }

// LoanTypeByKey یافتن نوع وام با کلید.
func LoanTypeByKey(key string) (LoanType, bool) {
	key = strings.TrimSpace(key)
	for _, t := range loanTypes() {
		if t.Key == key {
			return t, true
		}
	}
	return LoanType{}, false
}

// LoanTypeTitleFa عنوان فارسی نوع وام (اگر ناشناخته باشد همان مقدار ورودی).
func LoanTypeTitleFa(key string) string {
	if t, ok := LoanTypeByKey(key); ok {
		return t.TitleFa
	}
	return key
}

// ValidLoanType بررسی شناخته‌شده بودن نوع وام.
func ValidLoanType(key string) bool {
	_, ok := LoanTypeByKey(key)
	return ok
}

// IsSupplement بررسی متمم بودن نوع وام (مثل متمم ودیعهٔ مسکن).
func IsSupplement(key string) bool {
	if t, ok := LoanTypeByKey(key); ok {
		return t.IsSupplement
	}
	return false
}

// banks فهرست بانک‌های عامل مجاز.
func banks() []string {
	return []string{"melli", "mellat", "mehreghtesad", "saderat", "tejarat", "sepah", "refah", "keshavarzi", "maskan"}
}

// Banks فهرست کلید بانک‌های عامل.
func Banks() []string { return banks() }

// BankTitleFa عنوان فارسی بانک عامل.
func BankTitleFa(key string) string {
	switch strings.TrimSpace(key) {
	case "melli":
		return "بانک ملی"
	case "mellat":
		return "بانک ملت"
	case "mehreghtesad":
		return "بانک مهر اقتصاد"
	case "saderat":
		return "بانک صادرات"
	case "tejarat":
		return "بانک تجارت"
	case "sepah":
		return "بانک سپه"
	case "refah":
		return "بانک رفاه"
	case "keshavarzi":
		return "بانک کشاورزی"
	case "maskan":
		return "بانک مسکن"
	default:
		return key
	}
}

// ValidBank بررسی مجاز بودن بانک عامل.
func ValidBank(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return true // بانک عامل اختیاری است (وام‌های داخلی صندوق بانک ندارند).
	}
	for _, b := range banks() {
		if b == key {
			return true
		}
	}
	return false
}
