// نمونه‌ساز فیش ماهانهٔ نظام ۳۴۰۰۶ — ابزار توسعه، نه بخشی از سامانه.
//
// چرا این فایل: برای بازبینی و مستندسازی طرح فیش، یک فیش نمونه با اعداد واقعی موتور قواعد
// ساخته می‌شود و به‌صورت HTML راست‌چین در `docs/` ذخیره می‌گردد. با تغییر ضریب ریالی یا
// پارامترهای سالانه، همین فرمان نمونه را تازه می‌کند:
//
//	cd server && go run ./cmd/sample34006
//
// این بسته در باینری استقرار (`cmd/hrmsd`) نیست و فقط برای ساخت مستند به‌کار می‌رود.
package main

import (
	"fmt"
	"os"
	"strings"

	"hrms/internal/rules"
)

func main() {
	cp := rules.Comp34006ParamsFromValues(map[string]float64{
		"comp34006.rialFactor":          7500,
		"comp34006.seniority.amount":    3_000_000,
		"comp34006.fixedMonthly.amount": 5_000_000,
		"comp34006.housing.amount":      9_000_000,
		"comp34006.grocery.amount":      6_000_000,
		"comp34006.child.amount":        7_000_000,
		"comp34006.performance.percent": 20,
	})
	pp := rules.DefaultPayrollParams()

	slip := rules.Comp34006PayslipFor(rules.Comp34006PayslipInput{
		Comp34006Input: rules.Comp34006Input{
			EmployeeID: "EMP-1042", EmployeeName: "سارا محمدی",
			JobGrade: 18, PersonalGrade: 16, PostClass: "key",
			RoleType: "type1", RoleCount: 1, Region: "mahshahr",
			ShiftPattern: "company", ShiftDays: 22, OvertimeHours: 18,
			MissionDays: 2, ChildrenCount: 2, PerformanceScore: 92, MonthlyWorkDays: 30,
		},
		PeriodFa:        "1405/08",
		OtherDeductions: 12_000_000,
	}, cp, pp)

	if !slip.Valid {
		fmt.Fprintln(os.Stderr, "slip invalid:", slip.MessageFa)
		os.Exit(1)
	}

	var b strings.Builder
	b.WriteString(htmlHead())
	b.WriteString(`<div class="wrap">
  <header>
    <div class="brand">
      <div class="logo">BIPC</div>
      <div>
        <h1>فیش حقوقی ماهانه</h1>
        <div class="sub">نظام جبران خدمات دستورالعمل ۳۴۰۰۶ — سند IMS-HR-34006</div>
      </div>
    </div>
    <div class="meta">
      <div>دورهٔ پرداخت: <b>` + rules.IntFa(slip.PeriodFa) + `</b></div>
      <div>شمارهٔ فیش: <b class="ltr">C346-` + rules.IntFa("000118") + `</b></div>
    </div>
  </header>

  <section class="who">
    <div><span>کارمند</span><b>` + slip.EmployeeName + `</b></div>
    <div><span>کد پرسنلی</span><b class="ltr">` + slip.EmployeeID + `</b></div>
    <div><span>گرید شغلی</span><b>` + slip.JobGradeFa + `</b></div>
    <div><span>گرید شخصی</span><b>` + rules.IntFa(slip.PersonalGrade) + `</b></div>
    <div><span>حقوق پایه (مبنا)</span><b>` + rules.IntMoneyFa(slip.BaseSalary) + ` ریال</b></div>
  </section>

  <div class="tables">
    <section class="col">
      <h2>اقلام بستانکار</h2>
      <table>
        <thead><tr><th>کد</th><th>عنوان</th><th>صورت محاسبه</th><th>مبلغ (ریال)</th></tr></thead>
        <tbody>`)
	for _, line := range slip.Earnings {
		b.WriteString(`<tr><td class="ltr code">` + line.Code + `</td><td>` + line.TitleFa + `</td>` +
			`<td class="calc">` + line.FormulaFa + `</td>` +
			`<td class="num">` + rules.IntMoneyFa(line.Amount) + `</td></tr>`)
	}
	b.WriteString(`</tbody>
        <tfoot><tr><td colspan="3">جمع اقلام بستانکار (ناخالص)</td>
          <td class="num">` + rules.IntMoneyFa(slip.Gross) + `</td></tr></tfoot>
      </table>
    </section>

    <section class="col">
      <h2>کسور</h2>
      <table>
        <thead><tr><th>عنوان</th><th>مبنای محاسبه</th><th>مبلغ (ریال)</th></tr></thead>
        <tbody>`)
	for _, line := range slip.Deductions {
		b.WriteString(`<tr><td>` + line.TitleFa + `</td><td class="calc">` + line.FormulaFa + `</td>` +
			`<td class="num">` + rules.IntMoneyFa(line.Amount) + `</td></tr>`)
	}
	b.WriteString(`</tbody>
        <tfoot><tr><td colspan="2">جمع کسور</td>
          <td class="num">` + rules.IntMoneyFa(slip.TotalDeductions) + `</td></tr></tfoot>
      </table>
    </section>
  </div>

  <section class="totals">
    <div class="net">
      <span>خالص پرداختی این ماه</span>
      <b>` + rules.IntMoneyFa(slip.Net) + ` ریال</b>
    </div>
    <div class="side">
      <div><span>جمع ناخالص</span><b>` + rules.IntMoneyFa(slip.Gross) + `</b></div>
      <div><span>جمع کسور</span><b>` + rules.IntMoneyFa(slip.TotalDeductions) + `</b></div>
      <div><span>هزینهٔ تمام‌شدهٔ کارفرما</span><b>` + rules.IntMoneyFa(slip.EmployerCost) + `</b></div>
      <div><span>مبنای بیمه</span><b>` + rules.IntMoneyFa(slip.InsuranceBase) + `</b></div>
    </div>
  </section>

  <footer>
    <div>فیش پیشنهادی — پارامتر «اعمال در پرداخت» این مدل در تنظیمات شرکت خاموش است؛ تا روشن‌شدن آن، جای فیش جاری را نمی‌گیرد.</div>
    <div class="hint">ضریب ریالی سال: ` + rules.IntMoneyFa(cp.RialFactor) + ` ریال · جدول‌های ۱ تا ۵ دستورالعمل · کسور بیمه و مالیات از قاعدهٔ مشترک سه نظام پرداخت</div>
  </footer>
</div></body></html>`)

	out := "../docs/compensation-34006-payslip-sample.html"
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write failed:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (gross=%d net=%d)\n", out, slip.Gross, slip.Net)
}

func htmlHead() string {
	return `<!doctype html>
<html dir="rtl" lang="fa"><head><meta charset="utf-8">
<title>فیش حقوقی ماهانه — نظام ۳۴۰۰۶</title>
<style>
  :root{--ink:#111827;--muted:#6b7280;--line:#d1d5db;--acc:#1d4ed8;--good:#047857}
  *{box-sizing:border-box}
  body{margin:0;background:#eef1f6;color:var(--ink);
       font-family:Vazirmatn,"IRANSans",Tahoma,sans-serif;font-size:12.5px}
  .wrap{max-width:900px;margin:8px auto;background:#fff;padding:16px 20px;
        box-shadow:0 2px 14px rgba(15,23,42,.12);border-radius:10px}
  header{display:flex;justify-content:space-between;align-items:flex-start;
         border-bottom:2px solid var(--acc);padding-bottom:8px}
  .brand{display:flex;gap:12px;align-items:center}
  .logo{width:52px;height:52px;border-radius:10px;background:var(--acc);color:#fff;
        display:flex;align-items:center;justify-content:center;font-weight:700;letter-spacing:.5px}
  h1{font-size:1.15rem;margin:0}
  .sub{color:var(--muted);font-size:.78rem;margin-top:3px}
  .meta{text-align:left;font-size:.8rem;line-height:1.9;color:var(--muted)}
  .who{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:6px;
       margin:9px 0 11px;background:#f8fafc;border:1px solid var(--line);border-radius:8px;padding:7px 10px}
  .who span{display:block;color:var(--muted);font-size:.72rem;margin-bottom:2px}
  .who b{font-size:.86rem}
  .tables{display:grid;grid-template-columns:1.35fr 1fr;gap:16px;align-items:start}
  @media print{.tables{grid-template-columns:1.35fr 1fr}.wrap{box-shadow:none;margin:0;max-width:none}}
  h2{font-size:.9rem;margin:0 0 6px;color:var(--acc)}
  table{width:100%;border-collapse:collapse;font-size:.72rem}
  th,td{border:1px solid var(--line);padding:2.5px 5px;text-align:right;vertical-align:middle}
  th{background:#f3f4f6;font-weight:600}
  tfoot td{background:#f9fafb;font-weight:700}
  .num{direction:ltr;text-align:left;font-variant-numeric:tabular-nums;white-space:nowrap}
  .ltr{direction:ltr;text-align:center}
  .code{font-size:.72rem;color:var(--muted)}
  .calc{font-size:.68rem;color:var(--muted)}
  .totals{display:flex;gap:12px;margin-top:10px;align-items:stretch;flex-wrap:wrap}
  .net{flex:1 1 240px;background:#ecfdf5;border:1px solid #a7f3d0;border-radius:8px;
       padding:12px 14px;display:flex;flex-direction:column;justify-content:center}
  .net span{color:var(--good);font-size:.78rem}
  .net b{font-size:1.25rem;color:var(--good);font-variant-numeric:tabular-nums}
  .side{flex:2 1 320px;display:grid;grid-template-columns:1fr 1fr;gap:8px;
        border:1px solid var(--line);border-radius:8px;padding:10px 12px}
  .side div{display:flex;justify-content:space-between;font-size:.78rem;
            border-bottom:1px dashed #e5e7eb;padding-bottom:4px}
  .side span{color:var(--muted)}
  .side b{font-variant-numeric:tabular-nums}
  footer{margin-top:10px;border-top:1px solid var(--line);padding-top:7px;
         color:var(--muted);font-size:.7rem;line-height:1.7}
  .hint{margin-top:3px}
</style></head><body>`
}
