/* =====================================================================
   نظام جبران خدمات دستورالعمل ۳۴۰۰۶ (IMS-HR-34006) — سومین نظام پرداخت
   ---------------------------------------------------------------------
   چرا این صفحه: سازمان سه نظام پرداخت مستقل دارد. مدل حقوق و دستمزد جاری و مدل صندوق
   بازنشستگی نفت دو تای پیشین‌اند؛ این نظام، دستورالعمل جبران خدمات شرکت است و ساختار
   فیش خودش را دارد:

       اقلام بستانکار (کدهای ۳۴۶۰۱ تا ۳۴۶۱۸)
         حق گرید شغلی = امتیاز گرید شغلی × ضریب ریالی
         + پایهٔ سنواتی  →  حقوق پایه (مبنا)
         + فوق‌العاده‌های امتیازی (پست استراتژیک، گرید شخصی، نقش، جذب بازار، منطقه)
         + اقلام مبتنی بر حقوق پایه (اضافه‌کاری، نوبت‌کاری، مأموریت، اقماری، تعطیل‌کاری)
       − کسور قانونی (بیمهٔ سهم کارمند و مالیات) و سایر کسور
       = خالص پرداختی

   هیچ عدد و قاعده‌ای اینجا نیست: ضریب ریالی، جدول‌های ۱ تا ۵، ضرایب پست و نقش، نرخ بیمه
   و پله‌های مالیات همه از پارامترهای شرکت می‌آید و محاسبه در سرور است:

     GET  /api/v1/payroll/comp34006-defaults   پارامترها، جدول‌ها، فهرست‌های مرجع، کارکنان
     POST /api/v1/payroll/comp34006-calc        محاسبهٔ بستهٔ پرداخت (فقط اقلام)
     POST /api/v1/payroll/comp34006-payslip     فیش ماهانه: اقلام + کسور + خالص
     POST /api/v1/payroll/records               ثبت فیش (نوع comp34006Payslips)

   تا وقتی پارامتر «اعمال در پرداخت» شرکت خاموش است، این فیش پیشنهادی است و جای فیش جاری
   را نمی‌گیرد.
   ===================================================================== */

var C34006_DEFAULTS = null;   // پارامترها، جدول‌ها و فهرست‌های مرجع
var C34006_SLIP = null;       // آخرین فیش محاسبه‌شده
var C34006_PERIOD = '';       // دورهٔ انتخابی

/** مقدار عددی یک ورودی (خالی = صفر).
 *
 *  ارقام فارسی و عربی هم پذیرفته می‌شوند: کاربر ممکن است عدد را با صفحه‌کلید فارسی
 *  بنویسد؛ پیش از تبدیل، رقم‌ها لاتین می‌شوند تا مقدار صفر نشود. */
function c34006Num(id) {
  const el = document.getElementById(id);
  if (!el) return 0;
  const latin = (el.value || '').replace(/[۰-۹]/g, d => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d)))
    .replace(/[٠-٩]/g, d => String('٠١٢٣٤٥٦٧٨٩'.indexOf(d)));
  return Number(latin.replace(/[^0-9.\-]/g, '')) || 0;
}

/** متن یک ورودی. */
function c34006Text(id) {
  const el = document.getElementById(id);
  return el ? (el.value || '').trim() : '';
}

/** مقدار یک فهرست کشویی. */
function c34006Select(id) {
  const el = document.getElementById(id);
  return el ? el.value : '';
}

/** گزینه‌های یک فهرست کشویی از فهرست سرور. */
function c34006Options(items, valueKey, titleKey, selected) {
  return (items || []).map(it => {
    const value = it[valueKey];
    const isSel = String(value) === String(selected) ? 'selected' : '';
    return `<option value="${esc(value)}" ${isSel}>${esc(it[titleKey])}</option>`;
  }).join('');
}

/** ورودی فیش از فرم (هیچ قاعده‌ای اینجا نیست). */
function c34006Input() {
  const marketBox = document.getElementById('c34006Market');
  return {
    employeeId: c34006Select('c34006Employee'),
    jobGrade: c34006Num('c34006JobGrade'),
    personalGrade: c34006Num('c34006PersonalGrade'),
    postClass: c34006Select('c34006PostClass'),
    roleType: c34006Select('c34006RoleType'),
    roleCount: c34006Num('c34006RoleCount'),
    region: c34006Select('c34006Region'),
    marketAttractionApproved: !!(marketBox && marketBox.checked),
    marketAttractionPercent: c34006Num('c34006MarketPercent'),
    childrenCount: c34006Num('c34006Children'),
    shiftPattern: c34006Select('c34006ShiftPattern'),
    shiftDays: c34006Num('c34006ShiftDays'),
    overtimeHours: c34006Num('c34006OvertimeHours'),
    holidayHours: c34006Num('c34006HolidayHours'),
    fridayHours: c34006Num('c34006FridayHours'),
    missionDays: c34006Num('c34006MissionDays'),
    offshoreDays: c34006Num('c34006OffshoreDays'),
    monthlyWorkDays: c34006Num('c34006WorkDays') || 30,
    periodFa: c34006Text('c34006Period') || C34006_PERIOD,
    otherDeductions: c34006Num('c34006OtherDeductions'),
  };
}

/** کارت‌های خلاصهٔ فیش (ناخالص، کسور، خالص، حقوق پایه). */
function c34006SummaryCards(slip) {
  return `<div class="grid-stats" style="margin-bottom:12px">
    ${statCard({ico: '🧾', tint: '#EFF6FF', accent: '#2563EB', val: money(slip.gross),
      label: 'جمع اقلام بستانکار (ناخالص)'})}
    ${statCard({ico: '➖', tint: '#FEE2E2', accent: '#DC2626', val: money(slip.totalDeductions),
      label: 'جمع کسور (بیمه، مالیات، سایر)'})}
    ${statCard({ico: '💵', tint: '#DCFCE7', accent: '#16A34A', val: money(slip.net),
      label: 'خالص پرداختی'})}
    ${statCard({ico: '📐', tint: '#EDE9FE', accent: '#7C3AED', val: money(slip.baseSalary),
      label: 'حقوق پایه (مبنا) — گرید ' + fa(slip.jobGrade || 0)})}
  </div>`;
}

/** جدول اقلام بستانکار (کدهای دستورالعمل). */
function c34006EarningsTable(lines) {
  const rows = (lines || []).map(line => `<tr>
    <td class="num ltr">${esc(line.code)}</td>
    <td>${esc(line.titleFa)}</td>
    <td class="num">${money(line.amount)}</td>
    <td class="muted" style="font-size:.74rem">${esc(line.formulaFa || '—')}</td>
    <td class="muted" style="font-size:.72rem">${esc(line.refFa || '')}</td>
  </tr>`);
  return liveTable(['کد', 'عنوان قلم', 'مبلغ (ریال)', 'صورت محاسبه', 'مرجع در دستورالعمل'],
    rows, rows.length);
}

/** جدول کسور فیش. */
function c34006DeductionsTable(lines) {
  const rows = (lines || []).map(line => `<tr>
    <td class="ltr" style="font-size:.72rem">${esc(line.code)}</td>
    <td>${esc(line.titleFa)}</td>
    <td class="num">${money(line.amount)}</td>
    <td class="muted" style="font-size:.74rem">${esc(line.formulaFa || '—')}</td>
    <td class="muted" style="font-size:.72rem">${esc(line.refFa || '')}</td>
  </tr>`);
  return liveTable(['کلید', 'عنوان کسر', 'مبلغ (ریال)', 'مبنای محاسبه', 'مرجع'], rows, rows.length);
}

/** هشدارهای فیش. */
function c34006WarningsBox(warnings) {
  if (!warnings || !warnings.length) return '';
  const items = warnings.filter(Boolean).map(w => `<li>${esc(w)}</li>`).join('');
  if (!items) return '';
  return `<div class="card card-pad" style="margin-top:10px;border-right:3px solid #D97706">
    <b>⚠️ نکته‌های این فیش</b>
    <ul style="margin:6px 12px 0;font-size:.8rem;line-height:2">${items}</ul>
  </div>`;
}

/** جدول نردبان‌های امتیاز (جدول‌های ۱ تا ۵) و پارامترهای کلیدی. */
function c34006TablesBox(def) {
  const ladderRows = (def.ladders || []).map(l => `<tr>
    <td>${esc(l.titleFa)}</td>
    <td class="num">${l.basePoints ? money(l.basePoints) : '—'}</td>
    <td class="num">${l.topPoints ? money(l.topPoints) : '—'}</td>
    <td class="num">${fa(String(l.growth))}</td>
  </tr>`);
  const params = def.params || {};
  return `<div class="card card-pad" style="margin-top:14px">
    <div class="card-title">📊 جدول‌های دستورالعمل و پارامترهای فعال شرکت</div>
    <div class="muted" style="font-size:.78rem;margin-bottom:8px">
      ضریب ریالی سال: <b class="num">${money(params.rialFactor)}</b> ریال —
      گرید شغلی: <b class="num">${fa(params.maxJobGrade)}</b> —
      گرید شخصی: <b class="num">${fa(params.maxPersonalGrade || params.maxJobGrade)}</b> —
      مبنای روزهای ماه: <b class="num">${fa(params.monthlyDays || 30)}</b> —
      حالت ترکیب فوق‌العادهٔ منطقه: <b>${esc(params.regionMode)}</b> —
      «اعمال در پرداخت»: <b>${params.payrollApply ? 'روشن' : 'خاموش'}</b>
    </div>
    ${liveTable(['جدول', 'لنگر گرید ۱ (امتیاز)', 'لنگر گرید آخر (امتیاز)', 'نرخ رشد هر گرید'],
      ladderRows, ladderRows.length)}
  </div>`;
}

/** محاسبهٔ فیش ماهانه با سرور. */
async function c34006Calc() {
  const box = document.getElementById('c34006Slip');
  if (box) box.innerHTML = '<div class="muted" style="font-size:.78rem">در حال محاسبه…</div>';
  const res = await apiFetch('/api/v1/payroll/comp34006-payslip', {method: 'POST', body: c34006Input()});
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    toast(res.message, 'error');
    return;
  }
  const d = res.data || {};
  const slip = d.slip || {};
  C34006_SLIP = slip;
  C34006_PERIOD = slip.periodFa || C34006_PERIOD;
  if (!slip.valid) {
    if (box) box.innerHTML = liveErrorBox(slip.messageFa || 'فیش محاسبه نشد.');
    return;
  }
  const head = c34006SummaryCards(slip) +
    `<div class="card card-pad" style="margin-bottom:10px;font-size:.8rem;line-height:2">
      <b>${slip.payrollApply ? '✅' : '🧪'} ${esc(d.schemeFa || '')}</b>
      <div class="muted" style="font-size:.76rem">${esc(d.independenceFa || '')}</div>
      <div class="muted" style="font-size:.76rem;margin-top:4px">${esc(slip.summaryFa || '')}</div>
    </div>`;
  const actions = `<div class="row-actions" style="margin:12px 0">
      <button class="btn btn-primary" onclick="c34006Print()">🖨 چاپ فیش</button>
      <button class="btn btn-ghost" onclick="c34006Save()">💾 ثبت فیش در پرونده</button>
      <span class="muted" style="font-size:.74rem">
        مبنای بیمه: <b class="num">${money(slip.insuranceBase)}</b> —
        درآمد مشمول مالیات: <b class="num">${money(slip.taxableIncome)}</b> —
        هزینهٔ کارفرما: <b class="num">${money(slip.employerCost)}</b>
      </span>
    </div>`;
  if (box) {
    box.innerHTML = head + actions +
      `<div class="card-title" style="margin-top:6px">✅ اقلام بستانکار (۳۴۶۰۱ تا ۳۴۶۱۸)</div>` +
      c34006EarningsTable(slip.earnings) +
      `<div class="card-title" style="margin-top:14px">➖ کسور</div>` +
      c34006DeductionsTable(slip.deductions) +
      c34006WarningsBox(slip.warningsFa);
  }
  const status = document.getElementById('c34006Apply');
  if (status) {
    status.textContent = slip.payrollApply
      ? 'این مدل در پرداخت اعمال می‌شود'
      : 'فیش پیشنهادی — «اعمال در پرداخت» شرکت خاموش است';
  }
}

/** چاپ فیش: پنجرهٔ مستقل با ساختار بستانکار/بدهکار فیش رسمی. */
function c34006Print() {
  const slip = C34006_SLIP;
  if (!slip || !slip.valid) { toast('ابتدا فیش را محاسبه کنید', 'error'); return; }
  const employee = c34006Text('c34006Employee');
  const employeeTitle = (C34006_DEFAULTS.employees || []).find(e => e.id === employee);
  const earningRows = (slip.earnings || []).map(l => `<tr>
    <td class="ltr">${esc(l.code)}</td><td>${esc(l.titleFa)}</td>
    <td>${esc(l.formulaFa || '')}</td><td class="num">${money(l.amount)}</td></tr>`).join('');
  const deductionRows = (slip.deductions || []).map(l => `<tr>
    <td>${esc(l.titleFa)}</td><td>${esc(l.formulaFa || '')}</td>
    <td class="num">${money(l.amount)}</td></tr>`).join('');
  const win = window.open('', '_blank');
  if (!win) { toast('پنجرهٔ چاپ باز نشد', 'error'); return; }
  win.document.write(`<!doctype html><html dir="rtl" lang="fa"><head><meta charset="utf-8">
    <title>فیش حقوقی — نظام ۳۴۰۰۶</title>
    <style>
      body{font-family:Vazirmatn,Tahoma,sans-serif;padding:24px;color:#111}
      h1{font-size:1.1rem;margin:0 0 4px}h2{font-size:.95rem;margin:18px 0 6px}
      table{width:100%;border-collapse:collapse;font-size:.8rem}
      th,td{border:1px solid #bbb;padding:5px 7px;text-align:right}
      th{background:#f3f4f6}.num{font-variant-numeric:tabular-nums;direction:ltr;text-align:left}
      .ltr{direction:ltr;text-align:left}.muted{color:#666;font-size:.78rem}
      .totals{margin-top:12px;font-size:.85rem;line-height:2}
      .totals b{font-variant-numeric:tabular-nums}
    </style></head><body>
    <h1>فیش حقوقی ماهانه — نظام جبران خدمات دستورالعمل ۳۴۰۰۶</h1>
    <div class="muted">سند IMS-HR-34006 · دورهٔ ${esc(slip.periodFa || '—')}</div>
    <div style="margin:10px 0;font-size:.85rem">
      کارمند: <b>${esc(employeeTitle ? employeeTitle.titleFa : employee)}</b> —
      گرید شغلی: <b>${esc(slip.jobGradeFa || '')}</b> —
      گرید شخصی: <b>${fa(slip.personalGrade || 0)}</b>
    </div>
    <h2>اقلام بستانکار</h2>
    <table><thead><tr><th>کد</th><th>عنوان</th><th>صورت محاسبه</th><th>مبلغ (ریال)</th></tr></thead>
      <tbody>${earningRows}
      <tr><td></td><td><b>جمع اقلام بستانکار</b></td><td></td>
        <td class="num"><b>${money(slip.gross)}</b></td></tr></tbody></table>
    <h2>کسور</h2>
    <table><thead><tr><th>عنوان</th><th>مبنای محاسبه</th><th>مبلغ (ریال)</th></tr></thead>
      <tbody>${deductionRows}
      <tr><td><b>جمع کسور</b></td><td></td><td class="num"><b>${money(slip.totalDeductions)}</b></td></tr></tbody></table>
    <div class="totals">
      خالص پرداختی: <b>${money(slip.net)}</b> ریال<br>
      هزینهٔ تمام‌شدهٔ کارفرما: <b>${money(slip.employerCost)}</b> ریال
    </div>
    <p class="muted">${esc(slip.payrollApply
      ? 'این فیش بر پایهٔ پارامترهای فعال شرکت محاسبه شده است.'
      : 'این فیش پیشنهادی است؛ تا روشن‌شدن پارامتر «اعمال در پرداخت» جای فیش جاری را نمی‌گیرد.')}</p>
    </body></html>`);
  win.document.close();
  win.print();
}

/** ثبت فیش در پروندهٔ کارمند (همان موتور رکوردهای ماژول حقوق). */
async function c34006Save() {
  const slip = C34006_SLIP;
  if (!slip || !slip.valid) { toast('ابتدا فیش را محاسبه کنید', 'error'); return; }
  const employeeId = c34006Select('c34006Employee');
  if (!employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  const employeeTitle = (C34006_DEFAULTS.employees || []).find(e => e.id === employeeId);
  const name = employeeTitle ? employeeTitle.titleFa : employeeId;
  const find = code => ((slip.earnings || []).find(l => l.code === code) || {}).amount || 0;
  const res = await apiFetch('/api/v1/payroll/records', {
    method: 'POST',
    body: {
      kind: 'comp34006Payslips',
      title: 'فیش حقوقی نظام ۳۴۰۰۶ — ' + name + ' — ' + (slip.periodFa || ''),
      employeeId: employeeId,
      fields: {
        period: slip.periodFa || '',
        jobGrade: String(slip.jobGrade || ''),
        personalGrade: String(slip.personalGrade || ''),
        baseSalary: String(slip.baseSalary || 0),
        jobGradeAllowance: String(find('34601')),
        strategicAllowance: String(find('34603')),
        personalAllowance: String(find('34604')),
        roleAllowance: String(find('34605')),
        regionAllowance: String(find('34607')),
        gross: String(slip.gross || 0),
        insuranceEmployee: String(slip.insuranceEmployee || 0),
        tax: String(slip.tax || 0),
        otherDeductions: String(slip.otherDeductions || 0),
        totalDeductions: String(slip.totalDeductions || 0),
        net: String(slip.net || 0),
        shiftPattern: c34006Select('c34006ShiftPattern'),
        shiftDays: String(c34006Num('c34006ShiftDays')),
        overtimeHours: String(c34006Num('c34006OvertimeHours')),
        note: 'محاسبه از فرم نظام ۳۴۰۰۶ — وضعیت اعمال در پرداخت: ' + (slip.payrollApply ? 'روشن' : 'خاموش'),
      },
    },
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('فیش در پروندهٔ کارمند ثبت شد (وضعیت پیش‌نویس)', 'success');
}

/** صفحهٔ زندهٔ نظام ۳۴۰۰۶. */
function pageComp34006Live() {
  if (!isLive()) return offlineFallback('comp34006');
  liveLoad(async () => {
    if (!C34006_DEFAULTS) {
      const res = await apiFetch('/api/v1/payroll/comp34006-defaults');
      if (!res.ok) return liveErrorBox(res.message);
      C34006_DEFAULTS = res.data || {};
    }
    const def = C34006_DEFAULTS;
    const params = def.params || {};
    C34006_PERIOD = C34006_PERIOD || (def.today || '').slice(0, 7);
    const grades = (def.grades || []).map(g => ({value: g.grade, titleFa: g.titleFa}));
    // گرید شخصی سقف خودش را دارد (پارامتر comp34006.maxPersonalGrade) و با نردبان شغلی
    // یکی نیست؛ فهرست این فهرست کشویی از همان پارامتر بریده می‌شود.
    const personalMax = params.maxPersonalGrade || params.maxJobGrade || grades.length;
    const personalGrades = grades.filter(g => Number(g.value) <= Number(personalMax));
    const employees = def.employees || [];

    const form = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">🧾 ورودی فیش ماهانه</div>
      <div class="form-grid" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:10px">
        <label>کارمند
          <select id="c34006Employee" style="width:100%">
            <option value="">— انتخاب کارمند —</option>
            ${c34006Options(employees, 'id', 'titleFa', '')}
          </select></label>
        <label>دورهٔ فیش (ماه)
          <input class="inp num" id="c34006Period" value="${esc(C34006_PERIOD)}" placeholder="1405/08" style="width:100%"></label>
        <label>گرید شغلی (جدول ۱)
          <select id="c34006JobGrade" style="width:100%">${c34006Options(grades, 'value', 'titleFa', Math.min(21, params.maxJobGrade || 21))}</select></label>
        <label>گرید شخصی (جدول ۳)
          <select id="c34006PersonalGrade" style="width:100%">${c34006Options(personalGrades, 'value', 'titleFa', Math.min(21, personalMax))}</select></label>
        <label>دستهٔ پست استراتژیک
          <select id="c34006PostClass" style="width:100%">${c34006Options(def.postClasses, 'key', 'titleFa', '')}</select></label>
        <label>نوع نقش
          <select id="c34006RoleType" style="width:100%">
            <option value="">— بدون نقش —</option>
            ${c34006Options(def.roleTypes, 'key', 'titleFa', '')}
          </select></label>
        <label>شمار نقش‌های همزمان
          <input class="inp num" id="c34006RoleCount" value="1" style="width:100%"></label>
        <label>منطقهٔ محل خدمت
          <select id="c34006Region" style="width:100%">
            <option value="">— بدون فوق‌العادهٔ منطقه —</option>
            ${c34006Options(def.regions, 'key', 'titleFa', '')}
          </select></label>
        <label>ترکیب شیفت
          <select id="c34006ShiftPattern" style="width:100%">
            <option value="">— بدون نوبت‌کاری —</option>
            ${c34006Options(def.shiftPatterns, 'key', 'titleFa', '')}
          </select></label>
        <label>روزهای نوبت‌کاری
          <input class="inp num" id="c34006ShiftDays" value="0" style="width:100%"></label>
        <label>ساعت اضافه‌کاری
          <input class="inp num" id="c34006OvertimeHours" value="0" style="width:100%"></label>
        <label>ساعت تعطیل‌کاری (روزهای هفته)
          <input class="inp num" id="c34006HolidayHours" value="0" style="width:100%"></label>
        <label>ساعت جمعه‌کاری
          <input class="inp num" id="c34006FridayHours" value="0" style="width:100%"></label>
        <label>روزهای مأموریت
          <input class="inp num" id="c34006MissionDays" value="0" style="width:100%"></label>
        <label>روزهای اقماری
          <input class="inp num" id="c34006OffshoreDays" value="0" style="width:100%"></label>
        <label>شمار فرزندان مشمول
          <input class="inp num" id="c34006Children" value="0" style="width:100%"></label>
        <label>روزهای کار در ماه
          <input class="inp num" id="c34006WorkDays" value="${params.monthlyDays || 30}" style="width:100%"></label>
        <label>درصد جذب بازار کار (۰ = تا سقف پارامتر)
          <input class="inp num" id="c34006MarketPercent" value="0" style="width:100%"></label>
        <label>سایر کسور ماه (مساعده، اقساط)
          <input class="inp num" id="c34006OtherDeductions" value="0" style="width:100%"></label>
      </div>
      <div style="margin-top:10px;display:flex;gap:14px;align-items:center;flex-wrap:wrap">
        <label style="display:flex;gap:6px;align-items:center">
          <input type="checkbox" id="c34006Market"> فوق‌العادهٔ جذب بازار کار تصویب شده است</label>
      </div>
      <div class="row-actions" style="margin-top:12px">
        <button class="btn btn-primary" onclick="c34006Calc()">🧮 محاسبهٔ فیش ماهانه</button>
        <span id="c34006Apply" class="muted" style="font-size:.76rem">
          ${params.payrollApply ? 'این مدل در پرداخت اعمال می‌شود' : 'فیش پیشنهادی — «اعمال در پرداخت» شرکت خاموش است'}
        </span>
      </div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">📄 فیش ماهانه</div>
      <div id="c34006Slip"><div class="muted" style="font-size:.78rem">در حال محاسبه…</div></div>
    </div>`;

    setTimeout(c34006Calc, 80);
    return form + c34006TablesBox(def) + `<div class="card card-pad" style="margin-top:14px">
      <div class="card-title">🧭 مرز این مدل با دو نظام دیگر</div>
      <div class="muted" style="font-size:.8rem;line-height:2">${esc(def.noteFa || '')}</div>
      <ol style="margin:8px 14px 0;font-size:.8rem;line-height:2">
        ${(def.payslipInstructionFa || def.instructionFa || []).map(step => `<li>${esc(step)}</li>`).join('')}
      </ol>
    </div>`;
  });
  return liveShell('نظام جبران خدمات دستورالعمل ۳۴۰۰۶',
    'فیش ماهانه بر پایهٔ امتیاز گرید شغلی × ضریب ریالی، جدول‌های ۱ تا ۵ و کسور قانونی',
    `<button class="btn btn-ghost" onclick="C34006_DEFAULTS=null;liveReload()">↻ تازه‌سازی پارامترها</button>`);
}

/* ثبت در فهرست صفحه‌های زنده. */
LIVE_PAGES.comp34006 = pageComp34006Live;