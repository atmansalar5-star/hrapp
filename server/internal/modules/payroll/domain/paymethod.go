// روش پرداخت هر کارمند — «این کارمند با کدام مدل محاسبه شود؟»
//
// چرا یک رکورد جدا و نه فیلد روی پروندهٔ کارمند: انتخاب روش پرداخت سند سازمانی است (با
// تاریخ اجرا و دلیل)، نه مشخصهٔ ثابت هویتی؛ با تاریخ اجرا می‌توان تغییر روش را در آینده
// ثبت کرد و سابقهٔ روش‌های پیشین را نگه داشت.
//
// قاعدهٔ استقلال مدل‌ها: این انتخاب فقط «کدام موتور محاسبه مسئول این کارمند است» را
// ثبت می‌کند و هیچ محاسبه‌ای را خودبه‌خود عوض نمی‌کند. مدل‌های حقوق و دستمزد جاری و
// صندوق نفت مستقل و جاری‌اند؛ مدل ۴پی هم مدل توسعه‌ای و اختیاری است و انتخاب آن فقط
// وقتی معنا دارد که همان ماژول در استقرار فعال و «اعمال در پرداخت» آن روشن باشد.
package domain

import (
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/platform/calendar"
	"hrms/internal/platform/recordkit"
	"hrms/internal/rules"
)

// کلیدهای روش پرداخت — بازنمایی فهرست مشترک `rules` تا یک منبع حقیقت بماند.
const (
	// PayModelPayroll مدل حقوق و دستمزد جاری (پیش‌فرض سازمان).
	PayModelPayroll = rules.PayModelPayroll
	// PayModelOil نظام پرداخت صندوق بازنشستگی نفت.
	PayModelOil = rules.PayModelOil
	// PayModelComp34006 نظام جبران خدمات دستورالعمل IMS-HR-34006 (مستقل و اختیاری).
	PayModelComp34006 = rules.PayModelComp34006
	// PayModelFourP مدل توسعه‌ای و اختیاری جبران خدمات ۴پی.
	PayModelFourP = rules.PayModelFourP
)

// KindEmployeePayModel کلید نوع رکورد «روش پرداخت کارمند».
const KindEmployeePayModel = "employeePayModel"

// PayModelChoice یک گزینهٔ روش پرداخت برای فهرست انتخابی فرم.
type PayModelChoice = rules.PayModelChoice

// PayModelChoices فهرست روش‌های پرداخت مصوب (همان چهار روش سامانه).
var PayModelChoices = rules.PayModelChoices

// PayModelTitleFa عنوان فارسی یک روش پرداخت؛ روش ناشناخته خودِ کلید را برمی‌گرداند.
func PayModelTitleFa(key string) string { return rules.PayModelTitleFa(key) }

// PayModelOf کلید معتبر روش پرداخت؛ مقدار ناشناخته یعنی روش پیش‌فرض.
func PayModelOf(raw string) string { return rules.PayModelOf(raw) }

// employeePayModelKind روش پرداخت کارمند.
func employeePayModelKind() recordkit.Kind {
	return recordkit.Kind{
		Key: KindEmployeePayModel, TitleFa: "روش پرداخت کارمند", IDPrefix: "EPM",
		EmployeeScoped: true,
		Fields: recordkit.Fields(
			"payModel|روش محاسبهٔ پرداخت|select|1",
			"effectiveDate|تاریخ اجرا|date|1",
			"reasonFa|دلیل انتخاب یا تغییر روش|text|0",
		),
		Statuses: recordkit.Statuses(
			"draft|پیش‌نویس", "active|جاری", "superseded|منسوخ‌شده",
		),
		Initial: "draft",
		Transitions: recordkit.Steps(
			"draft|active", "active|superseded",
		),
		Actions: recordkit.Actions(
			"activate|جاری‌سازی روش|primary|active|0",
			// منسوخ‌کردن روش پیشین دلیل الزامی دارد تا معلوم بماند چه کسی و چرا روش را عوض کرده است.
			"supersede|منسوخ‌کردن|ghost|superseded|1",
		),
		Validate:         validateEmployeePayModel,
		EditableStatuses: []string{"draft", "active"},
		HistoryDate:      "effectiveDate",
	}
}

// validateEmployeePayModel قاعده‌های روش پرداخت: روش از فهرست مصوب، تاریخ اجرا معتبر و
// یک کارمند در هر لحظه بیشتر از یک رکورد جاری نداشته باشد (در لایهٔ مسیر سنجیده می‌شود).
func validateEmployeePayModel(in contracts.RecordInput) error {
	f := in.Fields
	raw := strings.TrimSpace(f["payModel"])
	known := false
	for _, choice := range PayModelChoices() {
		if choice.Key == raw {
			known = true
			break
		}
	}
	if !known {
		return &recordkit.RuleError{Message: "روش پرداخت را از فهرست مصوب انتخاب کنید."}
	}
	date := strings.TrimSpace(f["effectiveDate"])
	if date == "" {
		return &recordkit.RuleError{Message: "تاریخ اجرای روش پرداخت را وارد کنید."}
	}
	if _, err := calendar.Parse(calendar.NormalizeDigits(date)); err != nil {
		return &recordkit.RuleError{Message: "تاریخ اجرای روش پرداخت معتبر نیست (قالب YYYY/MM/DD)."}
	}
	return nil
}

// employeePayModelSeeds نمونهٔ روش پرداخت: دو کارمند تابع صندوق نفت.
//
// چرا نمونه لازم است: صفحهٔ «روش پرداخت کارکنان» باید در نصب تازه هم نشان دهد که
// انتخاب روش چه شکلی دارد؛ این داده فقط در نصب خالی ساخته می‌شود.
func employeePayModelSeeds() []contracts.RecordInput {
	return []contracts.RecordInput{
		{Kind: KindEmployeePayModel, Title: "روش پرداخت — امیر قاسمی (صندوق نفت)", EmployeeID: "EMP-011",
			NoteFa: "کارمند تابع صندوق بازنشستگی نفت؛ فیش ماهانه از نظام صندوق محاسبه می‌شود",
			Fields: map[string]string{
				"payModel": PayModelOil, "effectiveDate": "1405/01/01",
				"reasonFa": "حکم استخدامی بر پایهٔ آیین‌نامهٔ صندوق بازنشستگی نفت",
			},
			SeedStatus: "active"},
		{Kind: KindEmployeePayModel, Title: "روش پرداخت — محمد احمدی (صندوق نفت)", EmployeeID: "EMP-005",
			NoteFa: "کارمند تابع صندوق بازنشستگی نفت",
			Fields: map[string]string{
				"payModel": PayModelOil, "effectiveDate": "1405/01/01",
				"reasonFa": "حکم استخدامی بر پایهٔ آیین‌نامهٔ صندوق بازنشستگی نفت",
			},
			SeedStatus: "active"},
		{Kind: KindEmployeePayModel, Title: "روش پرداخت — سارا محمدی (حقوق و دستمزد جاری)", EmployeeID: "EMP-001",
			NoteFa: "روش پیش‌فرض سازمان",
			Fields: map[string]string{
				"payModel": PayModelPayroll, "effectiveDate": "1405/01/01",
				"reasonFa": "روش پیش‌فرض سازمان برای کارکنان مشمول قانون کار",
			},
			SeedStatus: "active"},
	}
}
