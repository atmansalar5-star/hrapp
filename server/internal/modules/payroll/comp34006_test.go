// آزمون پیاده‌سازی ماژول «payroll» — «comp34006_test».

package payroll

import (
	"context"
	"testing"

	"hrms/internal/platform/modulekit"
	"hrms/internal/platform/orgs"
	"hrms/internal/rules"
)

// depsComp34006 محیط سادهٔ ماژول با پارامترهای شرکت برای آزمون مدل ۳۴۰۰۶.
//
// پورت‌ها خالی ولی ساخته‌شده‌اند: مسیر پیش‌فرض‌ها فهرست کارکنان را از پورت می‌خواند و
// نبودِ کارمند باید همان فهرست خالی بدهد، نه خطا.
func depsComp34006(params ...orgs.Param) modulekit.Deps {
	return modulekit.Deps{Org: orgs.Organization{Params: params}, Ports: &modulekit.Ports{}}
}

// TestComp34006ParamsFromCompanyProfile پارامترهای شرکت باید روی پیش‌فرض‌های سند بنشینند
// و کلیدهای ماژول‌های دیگر اثری نگذارند.
func TestComp34006ParamsFromCompanyProfile(t *testing.T) {
	deps := depsComp34006(
		orgs.Param{Key: "comp34006.rialFactor", Value: "8500"},
		orgs.Param{Key: "comp34006.shift.companyPercent", Value: "40"},
		orgs.Param{Key: "comp34006.payroll.apply", Value: "1"},
		orgs.Param{Key: "payroll.overtimeFactor", Value: "1.6"},
	)

	p := comp34006Params(deps, nil)
	if p.RialFactor != 8500 {
		t.Fatalf("ضریب ریالی شرکت خوانده نشد: %d", p.RialFactor)
	}
	if p.ShiftCompanyPercent != 40 {
		t.Fatalf("درصد نوبت‌کاری شرکت خوانده نشد: %v", p.ShiftCompanyPercent)
	}
	if !p.PayrollApply {
		t.Fatal("پارامتر اعمال در پرداخت باید روشن شود.")
	}
	if p.OvertimeFactor != 1.4 {
		t.Fatalf("کلید ماژول دیگر نباید پارامتر این مدل را عوض کند: %v", p.OvertimeFactor)
	}

	over := comp34006Params(deps, map[string]string{
		"comp34006.rialFactor":       "۹٬۰۰۰",
		"oilpay.tax.exemption":       "1",
		"compensation.payroll.apply": "1",
	})
	if over.RialFactor != 9000 {
		t.Fatalf("بازنویسی با ارقام فارسی اعمال نشد: %d", over.RialFactor)
	}
}

// TestComp34006DefaultsHasDocumentTables خروجی «پیش‌فرض‌ها» باید هر ۲۵ گرید و هفت نردبان
// جدول‌های سند را برای فرم بیاورد.
func TestComp34006DefaultsHasDocumentTables(t *testing.T) {
	out := comp34006Defaults(context.Background(), depsComp34006())

	if out["scheme"] != Comp34006SchemeKey {
		t.Fatalf("کلید نظام پرداخت نادرست است: %v", out["scheme"])
	}
	grades, ok := out["grades"].([]map[string]any)
	if !ok {
		t.Fatalf("فهرست گریدها نیامد: %T", out["grades"])
	}
	if len(grades) != 25 {
		t.Fatalf("باید ۲۵ گرید شغلی باشد، شد %d", len(grades))
	}
	if grades[24]["titleFa"] != "۲۵ (E)" {
		t.Fatalf("عنوان گرید ۲۵ باید «۲۵ (E)» باشد: %v", grades[24]["titleFa"])
	}
	if grades[0]["jobPoints"] != float64(3100) {
		t.Fatalf("امتیاز گرید ۱ جدول ۱ باید ۳۱۰۰ باشد: %v", grades[0]["jobPoints"])
	}

	ladders, ok := out["ladders"].([]rules.Comp34006Ladder)
	if !ok {
		t.Fatalf("نردبان‌ها نیامدند: %T", out["ladders"])
	}
	if len(ladders) != 7 {
		t.Fatalf("باید هفت نردبان باشد (جدول ۵ با سه ستون)، شد %d", len(ladders))
	}

	instruction, ok := out["instructionFa"].([]string)
	if !ok || len(instruction) < 5 {
		t.Fatalf("راهنمای شروع ناقص است: %T", out["instructionFa"])
	}
	if out["apply"] != false {
		t.Fatalf("پیش‌فرض «اعمال در پرداخت» باید خاموش باشد: %v", out["apply"])
	}
	if note, _ := out["noteFa"].(string); note == "" {
		t.Fatal("یادداشت مرز مدل‌ها باید در پاسخ باشد.")
	}
}

// TestComp34006CalcUsesCompanyParams محاسبه باید ضریب ریالی شرکت را به‌کار ببرد و وضعیت
// «اعمال در پرداخت» را با هشدار فارسی برگرداند.
func TestComp34006CalcUsesCompanyParams(t *testing.T) {
	deps := depsComp34006(orgs.Param{Key: "comp34006.rialFactor", Value: "8500"})
	res := comp34006Calc(deps, rules.Comp34006Input{EmployeeID: "EMP-1", JobGrade: 25})

	result, ok := res["result"].(rules.Comp34006Result)
	if !ok {
		t.Fatalf("نتیجهٔ محاسبه نیامد: %T", res["result"])
	}
	if !result.Valid {
		t.Fatalf("محاسبه معتبر نشد: %s", result.MessageFa)
	}
	if want := rules.RoundHalfUp(8500 * result.JobGradePoints); result.JobGradeAllow != want {
		t.Fatalf("حق گرید شغلی باید با ضریب ۸۵۰۰ باشد: %d در برابر %d", result.JobGradeAllow, want)
	}
	if result.PayrollApply {
		t.Fatal("بدون پارامتر، مدل نباید در پرداخت اعمال شود.")
	}
	if len(result.WarningsFa) == 0 {
		t.Fatal("محاسبهٔ مدل خاموش باید هشدار بدهد.")
	}

	applied := comp34006Calc(
		depsComp34006(
			orgs.Param{Key: "comp34006.rialFactor", Value: "8500"},
			orgs.Param{Key: "comp34006.payroll.apply", Value: "1"},
		),
		rules.Comp34006Input{EmployeeID: "EMP-1", JobGrade: 25},
	)
	appliedResult, ok := applied["result"].(rules.Comp34006Result)
	if !ok || !appliedResult.PayrollApply {
		t.Fatal("با پارامتر روشن، نتیجه باید «اعمال در پرداخت» را روشن نشان دهد.")
	}
}

// TestComp34006CalcValidation گرید بیرون از نردبان باید پیام فارسی بدهد و سطر نسازد.
func TestComp34006CalcValidation(t *testing.T) {
	res := comp34006Calc(depsComp34006(), rules.Comp34006Input{EmployeeID: "EMP-2", JobGrade: 40})
	result, ok := res["result"].(rules.Comp34006Result)
	if !ok {
		t.Fatalf("نتیجهٔ محاسبه نیامد: %T", res["result"])
	}
	if result.Valid {
		t.Fatal("گرید ۴۰ نباید معتبر باشد.")
	}
	if result.MessageFa == "" {
		t.Fatal("پیام اعتبارسنجی فارسی لازم است.")
	}
	if len(result.Lines) != 0 {
		t.Fatalf("محاسبهٔ نامعتبر نباید سطر پرداخت بسازد: %d", len(result.Lines))
	}
}

// TestComp34006PayslipEndpoint فیش ماهانه از مسیر ماژول: کسور قانونی، توازن مبالغ و دورهٔ
// جاری وقتی دوره داده نشده باشد.
func TestComp34006PayslipEndpoint(t *testing.T) {
	deps := depsComp34006(orgs.Param{Key: "comp34006.rialFactor", Value: "8500"})
	out := comp34006Payslip(context.Background(), deps, nil, rules.Comp34006PayslipInput{
		Comp34006Input: rules.Comp34006Input{
			EmployeeID: "EMP-3", JobGrade: 21, PersonalGrade: 20,
			ShiftPattern: "company", ShiftDays: 30, MonthlyWorkDays: 30,
		},
		PeriodFa: "1405/08",
	})

	if out["scheme"] != Comp34006SchemeKey {
		t.Fatalf("کلید نظام پرداخت نادرست است: %v", out["scheme"])
	}
	slip, ok := out["slip"].(rules.Comp34006Payslip)
	if !ok {
		t.Fatalf("فیش در پاسخ نیامد: %T", out["slip"])
	}
	if !slip.Valid {
		t.Fatalf("فیش معتبر نشد: %s", slip.MessageFa)
	}
	if slip.PeriodFa != "1405/08" {
		t.Fatalf("دورهٔ فیش باید همان ورودی باشد: %q", slip.PeriodFa)
	}
	if slip.Net != slip.Gross-slip.TotalDeductions {
		t.Fatalf("خالص پرداختی با ناخالص و کسور نمی‌خواند: %d", slip.Net)
	}
	if slip.Tax <= 0 {
		t.Fatalf("با این ناخالص، مالیات باید بیشتر از صفر باشد: %d", slip.Tax)
	}
	if slip.InsuranceEmployee <= 0 {
		t.Fatalf("سهم کارمند از بیمه باید بیشتر از صفر باشد: %d", slip.InsuranceEmployee)
	}
	if _, ok := out["statutory"].(rules.PayrollParams); !ok {
		t.Fatalf("پارامترهای قانونی (بیمه و مالیات) در پاسخ نیست: %T", out["statutory"])
	}
	if instr, _ := out["instructionFa"].([]string); len(instr) == 0 {
		t.Fatal("راهنمای صدور فیش باید در پاسخ باشد.")
	}

	// دورهٔ خالی باید به ماه جاری پر شود.
	auto := comp34006Payslip(context.Background(), deps, nil, rules.Comp34006PayslipInput{
		Comp34006Input: rules.Comp34006Input{EmployeeID: "EMP-3", JobGrade: 21},
	})
	autoSlip, ok := auto["slip"].(rules.Comp34006Payslip)
	if !ok || autoSlip.PeriodFa == "" {
		t.Fatalf("دورهٔ خالی باید به ماه جاری پر شود: %+v", autoSlip.PeriodFa)
	}
}
