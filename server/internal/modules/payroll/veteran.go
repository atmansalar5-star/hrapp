// امتیاز ایثارگری در محاسبهٔ حقوق این ماژول.
//
// خواندن پرونده در لایهٔ پورت‌ها انجام می‌شود (`modulekit.Deps.VeteranEntitlement`) چون
// سند فیش حقوقی چاپی هم در `platform/documents` همین امتیاز را لازم دارد و باید یک عدد
// بدهد. قاعدهٔ امتیازها در `internal/rules/veteran.go` و تفسیر فیلدها هم همان‌جاست؛ این
// فایل فقط میان‌بر خواندن برای مسیرهای همین ماژول است.
package payroll

import (
	"context"

	"hrms/internal/platform/modulekit"
	"hrms/internal/rules"
)

// veteranEntitlement امتیاز ایثارگری تأییدشدهٔ یک کارمند برای یک فیش.
//
// nil یعنی هیچ امتیازی نباید اعمال شود: کد کارمند خالی است، پرونده‌ای ثبت نشده، پرونده
// تأیید نهایی نشده، یا سیاست شرکت (`veteran.payroll.apply`) اعمال در فیش را خاموش کرده
// است. امتیاز از پروندهٔ تأییدشده خوانده می‌شود و از بدنهٔ درخواست HTTP قابل جعل نیست.
func veteranEntitlement(ctx context.Context, deps modulekit.Deps, employeeID string) *rules.VeteranEntitlement {
	return deps.VeteranEntitlement(ctx, employeeID)
}

// veteranEntitlements نقشهٔ کد کارمند → امتیاز برای جدول‌هایی که یک‌جا حساب می‌کنند
// (پردازش حقوق دوره): یک پرس‌وجو برای همهٔ کارکنان، نه یک پرس‌وجو برای هر کارمند.
func veteranEntitlements(ctx context.Context, deps modulekit.Deps) map[string]*rules.VeteranEntitlement {
	return deps.VeteranEntitlements(ctx)
}
