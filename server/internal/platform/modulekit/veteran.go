// خواندن امتیاز ایثارگری یک کارمند از ماژول امور ایثارگران برای مصرف‌کننده‌های دیگر.
//
// چرا در لایهٔ پورت‌ها: هم ماژول حقوق (فیش، شبیه‌ساز، کارکرد دوره، فیش صندوق نفت) و هم
// لایهٔ سندهای چاپی (فیش حقوقی چاپی) به همین امتیاز نیاز دارند. اگر هر کدام خودش
// پرونده را تفسیر کند، دو محاسبهٔ ناهمخوان پیدا می‌شود؛ پس خواندن پرونده یک‌جا و از راه
// پورت قرارداد (`RecordList`) انجام می‌شود و هیچ ماژولی به دامنهٔ ماژول دیگر وابسته
// نمی‌شود. قاعدهٔ خودِ امتیازها در `internal/rules/veteran.go` است.
package modulekit

import (
	"context"
	"strings"

	"hrms/internal/contracts"
	"hrms/internal/rules"
)

// veteransModuleID ماژول مالک پروندهٔ ایثارگری و veteranKind نوع رکورد آن.
const (
	veteransModuleID = "grievances"
	veteranKind      = "veteranRecords"
)

// VeteranEntitlement امتیاز ایثارگری تأییدشدهٔ یک کارمند را برمی‌گرداند.
//
// nil یعنی هیچ امتیازی نباید اعمال شود: کد کارمند خالی است، پرونده‌ای ثبت نشده، پرونده
// تأیید نهایی نشده، یا سیاست شرکت (`veteran.payroll.apply`) اعمال در فیش را خاموش کرده
// است. پروندهٔ در حال بررسی هیچ اثر مالی ندارد و فقط در ماژول امور ایثارگران نمایش
// داده می‌شود.
func (d Deps) VeteranEntitlement(ctx context.Context, employeeID string) *rules.VeteranEntitlement {
	id := strings.TrimSpace(employeeID)
	if id == "" {
		return nil
	}
	port, ok := d.veteranPort()
	if !ok {
		return nil
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: veteransModuleID, Kind: veteranKind, EmployeeID: id, Scope: "all", Size: 20,
	})
	if err != nil {
		return nil
	}
	params, ok := d.veteranParams()
	if !ok {
		return nil
	}
	rec, found := bestVeteranRecord(page.Items, id)
	if !found {
		return nil
	}
	return veteranEntitlementOf(rec, params)
}

// VeteranEntitlements نقشهٔ کد کارمند → امتیاز برای جدول‌هایی که یک‌جا حساب می‌کنند
// (مثل پردازش حقوق دوره).
//
// چرا یک فهرست و نه یک پرس‌وجو برای هر کارمند: جدول کارکرد برای همهٔ کارکنان مبلغ
// می‌سازد و تکرار پرس‌وجو به تعداد کارکنان، هم کند است و هم یک پرس‌وجو را چند بار به
// انبار می‌فرستد. پرس‌وجو یک‌بار انجام می‌شود و نقشه در حافظه ساخته می‌شود.
func (d Deps) VeteranEntitlements(ctx context.Context) map[string]*rules.VeteranEntitlement {
	port, ok := d.veteranPort()
	if !ok {
		return nil
	}
	params, ok := d.veteranParams()
	if !ok {
		return nil
	}
	page, err := port.List(ctx, contracts.RecordFilter{
		Module: veteransModuleID, Kind: veteranKind, Scope: "all", Size: 500,
	})
	if err != nil {
		return nil
	}
	best := map[string]contracts.Record{}
	for _, rec := range page.Items {
		if rec.EmployeeID == "" || !veteranStatusBetter(rec, best[rec.EmployeeID]) {
			continue
		}
		best[rec.EmployeeID] = rec
	}
	out := make(map[string]*rules.VeteranEntitlement, len(best))
	for employeeID, rec := range best {
		if ent := veteranEntitlementOf(rec, params); ent != nil {
			out[employeeID] = ent
		}
	}
	return out
}

// veteranPort پورت فقط‌خواندنی رکوردهای ماژول امور ایثارگران.
func (d Deps) veteranPort() (contracts.RecordListPort, bool) {
	if d.Ports == nil {
		return nil, false
	}
	return d.Ports.RecordList(veteransModuleID)
}

// veteranParams پارامترهای سیاستی امتیاز ایثارگری از پروفایل شرکت فعال.
//
// false یعنی سیاست شرکت اعمال در فیش را خاموش کرده است؛ در آن حالت پرونده خوانده هم
// نمی‌شود و فیش عیناً مثل کارمند بدون پرونده می‌ماند.
func (d Deps) veteranParams() (rules.VeteranParams, bool) {
	values := map[string]string{}
	for _, p := range d.Org.Params {
		if !strings.HasPrefix(p.Key, "veteran.") {
			continue
		}
		values[p.Key] = p.Value
	}
	params := rules.VeteranParamsFromText(values)
	if !params.PayrollApply {
		return params, false
	}
	return params, true
}

// veteranEntitlementOf پرونده را به موتور می‌دهد و تنها امتیاز «تأییدشده» را برمی‌گرداند.
func veteranEntitlementOf(rec contracts.Record, params rules.VeteranParams) *rules.VeteranEntitlement {
	// خواندن فیلدها هم در موتور قواعد است تا ماژول امور ایثارگران و حقوق یک تفسیر
	// مشترک داشته باشند؛ «تأییدشده» فقط وضعیت approved است. مقادیر تأییدشدهٔ ذخیره‌شده
	// (از جمله مصوبهٔ کمیتهٔ ایثارگری) هم روی پیش‌فرض قانونی می‌نشیند.
	ent := rules.VeteranEntitlementOfFile(rec.Fields, rec.Status == "approved", params)
	if !ent.Approved || !ent.IsVeteran {
		return nil
	}
	return &ent
}

// bestVeteranRecord پروندهٔ همین کارمند را برمی‌گزیند؛ تأییدشده بر بررسی‌شده اولویت دارد.
//
// فیلتر کارمند در انبار یک قید است و اینجا دوباره سنجیده می‌شود تا پروندهٔ کارمند دیگر
// هرگز در فیش این کارمند اثر نگذارد.
func bestVeteranRecord(items []contracts.Record, employeeID string) (contracts.Record, bool) {
	best := contracts.Record{}
	found := false
	for _, rec := range items {
		if rec.EmployeeID != employeeID {
			continue
		}
		if !found || veteranStatusBetter(rec, best) {
			best, found = rec, true
		}
	}
	return best, found
}

// veteranStatusBetter رتبهٔ وضعیت پرونده: تأییدشده بر بررسی‌شده بر بقیه.
func veteranStatusBetter(candidate, current contracts.Record) bool {
	return veteranStatusRank(candidate.Status) > veteranStatusRank(current.Status)
}

// veteranStatusRank رتبهٔ عددی وضعیت پروندهٔ ایثارگری.
func veteranStatusRank(status string) int {
	switch status {
	case "approved":
		return 2
	case "verified":
		return 1
	}
	return 0
}
