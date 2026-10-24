// منطق دامنهٔ ماژول «loans_retirement» — «quota»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"strings"

	"hrms/internal/contracts"
)

// این فایل «جدول سهمیهٔ تسهیلات» را اعلام می‌کند: سهمیهٔ مصوب هر مدیریت بندر امام،
// سرشماری مشمولین، مبلغ کل صندوق و مراحل ارسال درخواست‌ها به مدیریت مالی.
//
// چرا در دامنه و نه در UI: این اعداد «سیاست سازمان»‌اند، نه محاسبه. رابط وب فقط
// نمایش می‌دهد و هیچ درصد یا سهمیه‌ای از خودش نمی‌سازد؛ با تغییر اعداد در همین
// کاتالوگ، همهٔ نمودارها و API بدون تغییر کد به‌روز می‌شوند.

// QuotaTotalAmount مبلغ کل صندوق تسهیلات کارکنان در دورهٔ جاری (ریال).
// سهمیهٔ ریالی هر مدیریت = SharePercent × این مبلغ.
const QuotaTotalAmount int64 = 20_000_000_000

// QuotaUnit یک مدیریت بندر امام در جدول سهمیهٔ تسهیلات.
//
// هویت مدیریت (کلید و عنوان) از کاتالوگ مرجع `contracts.ManagementCatalog` می‌آید تا
// همین ۸ مدیریت در ماژول پاداش و گزارش‌ها هم یک تعریف داشته باشند؛ آنچه اینجا اضافه
// می‌شود «سیاست سهمیهٔ تسهیلات» است: SharePercent درصد سهمیهٔ مصوب از کل صندوق و
// Eligible سرشماری مشمولین همان مدیریت. مجموع SharePercent باید ۱۰۰ باشد؛ آزمون
// دامنه این را تضمین می‌کند.
type QuotaUnit struct {
	contracts.Management
	SharePercent float64
	Eligible     int
}

// quotaShare سهمیهٔ مصوب یک مدیریت در جدول تسهیلات (درصد و سرشماری مشمولین).
type quotaShare struct {
	percent  float64
	eligible int
}

// quotaShares سهمیهٔ مصوب هر مدیریت: درصد از کل صندوق و سرشماری مشمولین.
func quotaShares() map[string]quotaShare {
	return map[string]quotaShare{
		"bandar_imam":      {22, 420},
		"repair_support":   {14, 260},
		"water_power":      {13, 240},
		"basparan":         {12, 210},
		"process_one":      {13, 250},
		"process_two":      {12, 230},
		"services_support": {8, 150},
		"kimia":            {6, 90},
	}
}

// quotaUnits جدول سهمیه: هر مدیریت مرجع با سهمیهٔ مصوب خودش، به ترتیب کاتالوگ.
// مدیریت بدون سهمیهٔ اعلامی با درصد صفر می‌آید تا در نمودارها گم نشود.
func quotaUnits() []QuotaUnit {
	shares := quotaShares()
	managements := contracts.Managements()
	out := make([]QuotaUnit, 0, len(managements))
	for _, m := range managements {
		s := shares[m.Key]
		out = append(out, QuotaUnit{Management: m, SharePercent: s.percent, Eligible: s.eligible})
	}
	return out
}

// QuotaUnits فهرست مدیریت‌های جدول سهمیه.
func QuotaUnits() []QuotaUnit { return quotaUnits() }

// QuotaUnitByKey یافتن مدیریت با کلید.
func QuotaUnitByKey(key string) (QuotaUnit, bool) {
	key = strings.TrimSpace(key)
	for _, u := range quotaUnits() {
		if u.Key == key {
			return u, true
		}
	}
	return QuotaUnit{}, false
}

// QuotaUnitTitleFa عنوان فارسی مدیریت؛ برای کلید ناشناخته خودِ ورودی برمی‌گردد تا
// داده‌های قدیمی در نمایش خراب نشوند.
func QuotaUnitTitleFa(key string) string {
	if u, ok := QuotaUnitByKey(key); ok {
		return u.TitleFa
	}
	return contracts.ManagementTitleFa(key)
}

// ValidQuotaUnit بررسی شناخته‌شده بودن مدیریت در جدول سهمیه.
func ValidQuotaUnit(key string) bool {
	_, ok := QuotaUnitByKey(key)
	return ok
}

// QuotaEligibleTotal مجموع مشمولین جدول سهمیه.
func QuotaEligibleTotal() int {
	total := 0
	for _, u := range quotaUnits() {
		total += u.Eligible
	}
	return total
}

// QuotaShareTotal مجموع درصدهای سهمیه (باید ۱۰۰ باشد).
func QuotaShareTotal() float64 {
	total := 0.0
	for _, u := range quotaUnits() {
		total += u.SharePercent
	}
	return total
}

// ---------- مراحل ارسال به مدیریت مالی ----------

// QuotaStage مرحلهٔ گردش درخواست تسهیلات از دید «ارسال به مدیریت مالی».
//
// هر مرحله یک یا چند وضعیت رکورد وام را پوشش می‌دهد؛ SentToFinance می‌گوید پروندهٔ
// آن مرحله در دست مدیریت مالی است یا هنوز در واحد وام.
type QuotaStage struct {
	Key           string
	TitleFa       string
	Statuses      []string
	SentToFinance bool
}

// quotaStages مراحل اعلامی تخصیص و ارسال تسهیلات.
func quotaStages() []QuotaStage {
	return []QuotaStage{
		{Key: "awaitingReview", TitleFa: "در انتظار بررسی واحد وام", Statuses: []string{"requested"}},
		{Key: "scoredReady", TitleFa: "امتیازبندی‌شده — آمادهٔ ارسال به مالی", Statuses: []string{"scored"}},
		{Key: "sentToFinance", TitleFa: "ارسال‌شده به مدیریت مالی", Statuses: []string{"approved"}, SentToFinance: true},
		{Key: "inRepayment", TitleFa: "در بازپرداخت — تحت کنترل مالی", Statuses: []string{"repaying", "settled"}, SentToFinance: true},
		{Key: "rejected", TitleFa: "ردشده", Statuses: []string{"rejected"}},
	}
}

// QuotaStages فهرست مراحل ارسال تسهیلات به مدیریت مالی.
func QuotaStages() []QuotaStage { return quotaStages() }

// QuotaStageOf مرحلهٔ یک وضعیت رکورد وام. اگر وضعیت ناشناخته باشد، مرحلهٔ
// «در انتظار بررسی» با کلید خودِ وضعیت برگردانده می‌شود تا دادهٔ تازه گم نشود.
func QuotaStageOf(loanStatus string) QuotaStage {
	loanStatus = strings.TrimSpace(loanStatus)
	for _, st := range quotaStages() {
		for _, s := range st.Statuses {
			if s == loanStatus {
				return st
			}
		}
	}
	return QuotaStage{Key: loanStatus, TitleFa: loanStatus}
}

// SentToFinance آیا وضعیت این رکورد وام در دایرهٔ مدیریت مالی است.
func SentToFinance(loanStatus string) bool { return QuotaStageOf(loanStatus).SentToFinance }

// ---------- گروه کارگزینی: شرکتی / ارکان ثالث ----------

// گروه‌های کارگزینی در جدول سهمیه: نیروی خودِ شرکت در برابر نیروی پیمانکاری
// (ارکان ثالث). مبنا «خانوادهٔ مزد» در کاتالوگ انواع استخدام است.
const (
	EmploymentGroupCompany    = "company"
	EmploymentGroupThirdParty = "thirdParty"
)

// EmploymentGroupOf گروه کارگزینی یک نوع استخدام. هر نوعی که خانوادهٔ مزدش
// contracting باشد در «ارکان ثالث» می‌افتد؛ بقیه «شرکتی» شمرده می‌شوند.
func EmploymentGroupOf(employmentTypeKey string) string {
	if t, ok := contracts.EmploymentTypeByKey(employmentTypeKey); ok {
		if t.Kind == contracts.EmploymentKindContracting {
			return EmploymentGroupThirdParty
		}
	}
	return EmploymentGroupCompany
}

// EmploymentGroupTitleFa عنوان فارسی گروه کارگزینی.
func EmploymentGroupTitleFa(key string) string {
	if key == EmploymentGroupThirdParty {
		return "ارکان ثالث (پیمانکاری)"
	}
	return "شرکتی (رسمی و مدت معین)"
}

// EmploymentGroups گروه‌های کارگزینی به ترتیب نمایش.
func EmploymentGroups() []string {
	return []string{EmploymentGroupCompany, EmploymentGroupThirdParty}
}
