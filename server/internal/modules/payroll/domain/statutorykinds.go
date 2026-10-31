// نوع‌های رکورد فاز ۱ در ماژول حقوق: سند فایل قانونی، حساب بانکی کارمند و سند فایل پرداخت.
//
// قاعدهٔ معماری: این‌ها هم مثل بقیهٔ انواع رکورد از موتور مشترک `recordkit` می‌آیند؛ پس
// گردشکار (آماده ← ارسال‌شده ← پرداخت‌شده ← باطل)، سابقهٔ تغییرات، دسترسی و اعتبارسنجی
// خودکار است و قاعده‌های دامنه در `statutory.go` و `bankfile.go` می‌ماند.

package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
)

// statutoryFilesKind سند یک فایل قانونی (بیمه یا مالیات) در یک دوره: جمع‌ها، مهلت،
// شمارهٔ پیگیری ارسال، تاریخ و رسید پرداخت.
func statutoryFilesKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "statutoryFiles", TitleFa: "فایل قانونی بیمه/مالیات", IDPrefix: "STF", EmployeeScoped: false,
		Fields: recordkit.Fields(
			"fileType|نوع فایل|text|1",
			"period|دورهٔ پرداخت|text|1",
			"rows|شمار ردیف|number|0",
			"payableTotal|مبلغ قابل پرداخت|money|0",
			"insuranceEmployee|سهم بیمه‌شده|money|0",
			"insuranceEmployer|سهم کارفرما|money|0",
			"unemployment|بیمهٔ بیکاری|money|0",
			"taxTotal|جمع مالیات|money|0",
			"deadline|مهلت قانونی|text|0",
			"penalty|جریمهٔ دیرکرد برآوردی|money|0",
			"issues|مغایرت‌های پیش از ارسال|text|0",
			"trackingNo|شمارهٔ پیگیری ارسال|text|0",
			"sentAt|تاریخ ارسال|date|0",
			"paidAt|تاریخ پرداخت|date|0",
			"receiptNo|شمارهٔ رسید پرداخت|text|0",
			"noteFa|یادداشت|text|0",
		),
		Statuses: recordkit.Statuses(
			"prepared|آمادهٔ ارسال", "sent|ارسال‌شده", "paid|پرداخت‌شده", "void|باطل",
		),
		Initial: "prepared",
		Transitions: recordkit.Steps(
			"prepared|sent,void", "sent|paid,void", "paid|void",
		),
		Actions: recordkit.Actions(
			"markSent|ثبت ارسال|primary|sent|0|hr_manager,admin",
			"markPaid|ثبت پرداخت|primary|paid|0|hr_manager,admin",
			"void|باطل‌کردن فایل|danger|void|1|admin",
		),
		HistoryDate: "sentAt",
	}
}

// bankAccountsKind حساب بانکی کارمند: مبنای فایل پرداخت. شمارهٔ شبا با رقم کنترل
// استاندارد سنجیده می‌شود تا واریز به حساب اشتباه رخ ندهد.
func bankAccountsKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "bankAccounts", TitleFa: "حساب بانکی کارمند", IDPrefix: "BNK", EmployeeScoped: true,
		Fields: recordkit.Fields(
			"bankName|بانک|text|1",
			"iban|شمارهٔ شبا (IR)|text|1",
			"accountNo|شمارهٔ حساب|text|0",
			"holderName|نام صاحب حساب|text|0",
			"nationalId|کد ملی صاحب حساب|text|0",
		),
		Statuses:    recordkit.Statuses("draft|پیش‌نویس", "active|فعال", "inactive|غیرفعال"),
		Initial:     "draft",
		Transitions: recordkit.Steps("draft|active", "active|inactive", "inactive|active"),
		Actions: recordkit.Actions(
			"activate|فعال‌سازی حساب|primary|active|0|hr_spec,hr_manager",
			"deactivate|غیرفعال‌سازی حساب|ghost|inactive|0|hr_spec,hr_manager",
		),
		Validate: validateBankAccount,
		// حساب بانکی دادهٔ پیکربندی پروندهٔ کارمند است: اصلاح شمارهٔ شبا یا نام بانک پس
		// از فعال‌سازی هم پیش می‌آید، پس در وضعیت فعال هم ویرایش‌پذیر است.
		EditableStatuses: []string{"draft", "active"},
	}
}

// bankFilesKind سند فایل پرداخت بانکی یک دوره: جمع کنترلی، تفاوت گرد کردن، برگشتی‌ها و
// وضعیت ارسال/واریز.
func bankFilesKind() recordkit.Kind {
	return recordkit.Kind{
		Key: "bankFiles", TitleFa: "فایل پرداخت بانکی", IDPrefix: "BNKF", EmployeeScoped: false,
		Fields: recordkit.Fields(
			// نام بانک اختیاری است: یک فایل می‌تواند چند بانک را داشته باشد (هر ردیف نام بانک
			// خودش را دارد)، پس خالی‌بودنش نباید مانع ثبت سند فایل شود.
			"bankName|بانک (خالی یعنی همهٔ بانک‌ها)|text|0",
			"period|دورهٔ پرداخت|text|1",
			"rows|شمار ردیف|number|0",
			"amount|جمع مبلغ پرداختی|money|0",
			"rounding|جمع تفاوت گرد کردن|money|0",
			"returnedRows|ردیف برگشتی|number|0",
			"returnedAmount|مبلغ برگشتی|money|0",
			"controlFa|جمع کنترلی|text|0",
			"issues|مغایرت‌های پیش از ارسال|text|0",
			"trackingNo|شمارهٔ پیگیری بانک|text|0",
			"sentAt|تاریخ ارسال به بانک|date|0",
			"paidAt|تاریخ واریز|date|0",
			"receiptNo|شمارهٔ رسید بانک|text|0",
			"noteFa|یادداشت|text|0",
		),
		Statuses: recordkit.Statuses(
			"prepared|آمادهٔ ارسال", "sent|ارسال‌شده به بانک", "paid|واریزشده",
			"returned|دارای برگشتی", "void|باطل",
		),
		Initial: "prepared",
		Transitions: recordkit.Steps(
			"prepared|sent,void", "sent|paid,returned,void", "paid|returned", "returned|paid,void",
		),
		Actions: recordkit.Actions(
			"markSent|ثبت ارسال به بانک|primary|sent|0|hr_manager,admin",
			"markPaid|ثبت واریز|primary|paid|0|hr_manager,admin",
			"markReturned|ثبت برگشتی|danger|returned|1|hr_manager,admin",
			"void|باطل‌کردن فایل|danger|void|1|admin",
		),
		HistoryDate: "sentAt",
	}
}

// validateBankAccount قاعدهٔ حساب بانکی: شبا معتبر و کد ملی (اگر داده شده) درست.
func validateBankAccount(in contracts.RecordInput) error {
	if err := ValidateIBAN(in.Fields["iban"]); err != nil {
		return &recordkit.RuleError{Message: err.Error() + "."}
	}
	if raw := strings.TrimSpace(in.Fields["nationalId"]); raw != "" {
		if err := ValidateIranianNationalID(raw); err != nil {
			return &recordkit.RuleError{Message: "کد ملی صاحب حساب: " + err.Error() + "."}
		}
	}
	if _, err := periodOf(in.Fields["period"]); err != nil && strings.TrimSpace(in.Fields["period"]) != "" {
		return err
	}
	return nil
}

// periodOf دورهٔ YYYY/MM را می‌سنجد.
func periodOf(raw string) (string, error) {
	value := strings.TrimSpace(calendar.NormalizeDigits(raw))
	if value == "" {
		return "", nil
	}
	if _, _, ok := splitPeriod(value); !ok {
		return "", &recordkit.RuleError{Message: "دورهٔ پرداخت باید به قالب YYYY/MM باشد."}
	}
	return value, nil
}
