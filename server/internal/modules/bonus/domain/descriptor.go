// منطق دامنهٔ ماژول «bonus» — «descriptor»: قواعد، اعتبارسنجی و محاسبات خالص بدون وابستگی به زیرساخت.

package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// Descriptor شناسهٔ ماژول پاداش.
//
// یک نوع رکورد دارد: «پرداخت پاداش». هر رکورد یک رویداد پرداخت به یک نفر است؛ پس
// «تعداد دفعات پرداخت» و «مبلغ پرداختی به هر نفر در هر مدیریت» جمع‌زدنی روی همین
// رکوردهاست و هیچ جدول تجمیعی جداگانه‌ای لازم نیست.
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{
		Module:  "bonus",
		TitleFa: "پاداش‌ها",
		Kinds: []recordkit.Kind{
			{
				Key: "bonusPayments", TitleFa: "پرداخت پاداش", IDPrefix: "BNP", EmployeeScoped: true,
				Fields: recordkit.Fields(
					"bonusType|نوع پاداش|select|1",
					// مدیریت از کاتالوگ مرجع مدیریت‌های بندر امام پر می‌شود؛ مبنای
					// نمودارهای «پاداش به تفکیک مدیریت» است.
					"management|مدیریت|select|1",
					"amount|مبلغ پرداختی|money|1",
					"paymentDate|تاریخ پرداخت|date|1",
					// شمارهٔ دفعه صریح است تا پرداخت چندباره به یک نفر و «تعداد دفعات
					// پرداخت» در گزارش‌ها مبهم نشود.
					"sequence|شمارهٔ دفعه|number|0",
					"decreeNo|شمارهٔ مصوبه / حکم|text|0",
					"paymentMethod|روش پرداخت|select|0",
					"reference|شرح / مرجع|text|0",
				),
				Statuses: recordkit.Statuses(
					"draft|پیش‌نویس", "approved|تأییدشده", "paid|پرداخت‌شده", "cancelled|لغوشده",
				),
				Initial: "draft",
				Transitions: recordkit.Steps(
					"draft|approved,cancelled",
					"approved|paid,cancelled",
				),
				Actions: recordkit.Actions(
					"approve|تأیید پرداخت|primary|approved|0|hr_manager",
					"pay|ثبت پرداخت|primary|paid|0|hr_manager",
					"cancel|لغو پرداخت|danger|cancelled|1|hr_manager",
				),
				// قاعدهٔ دامنه: نوع پاداش، مدیریت، مبلغ و تاریخ پرداخت باید معتبر باشند و
				// پرداخت‌های مصوبه‌دار بدون شمارهٔ مصوبه ثبت نشوند.
				Validate: validateBonusPayment,
				// ویرایش و حذف فقط پیش از تأیید مجاز است؛ پس از آن فقط گردش‌کار.
				EditableStatuses: []string{"draft"},
			},
		},
	}
}

// validateBonusPayment قاعدهٔ دامنهٔ پرداخت پاداش.
func validateBonusPayment(in contracts.RecordInput) error {
	bonusType := strings.TrimSpace(in.Fields["bonusType"])
	if bonusType == "" {
		return &recordkit.RuleError{Message: "انتخاب نوع پاداش الزامی است."}
	}
	if !ValidBonusType(bonusType) {
		return &recordkit.RuleError{Message: "نوع پاداش «" + bonusType + "» در فهرست انواع پاداش شناخته نشد."}
	}
	management := strings.TrimSpace(in.Fields["management"])
	if management == "" {
		return &recordkit.RuleError{Message: "انتخاب مدیریت الزامی است."}
	}
	if !contracts.ValidManagement(management) {
		return &recordkit.RuleError{Message: "مدیریت «" + management + "» در فهرست مدیریت‌های بندر امام شناخته نشد."}
	}
	if !ValidPaymentMethod(in.Fields["paymentMethod"]) {
		return &recordkit.RuleError{Message: "روش پرداخت انتخابی مجاز نیست."}
	}
	if amount, ok := AmountOf(in.Fields["amount"]); !ok || amount <= 0 {
		return &recordkit.RuleError{Message: "مبلغ پرداختی باید عددی مثبت باشد."}
	}
	date := strings.TrimSpace(in.Fields["paymentDate"])
	if _, err := calendar.Parse(date); err != nil {
		return &recordkit.RuleError{Message: "تاریخ پرداخت باید به قالب شمسی YYYY/MM/DD باشد."}
	}
	// پرداخت‌های مصوبه‌دار بدون شمارهٔ مصوبه ثبت نمی‌شوند؛ این تفاوت پاداش با کارانهٔ
	// جاری است و باید از دادهٔ سازمان قابل ردیابی باشد.
	if t, ok := BonusTypeByKey(bonusType); ok && t.RequiresDecree && strings.TrimSpace(in.Fields["decreeNo"]) == "" {
		return &recordkit.RuleError{Message: "برای «" + t.TitleFa + "» ثبت شمارهٔ مصوبه / حکم الزامی است."}
	}
	return nil
}
