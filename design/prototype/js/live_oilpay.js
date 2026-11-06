/* =====================================================================
   فیش حقوقی صندوق نفت (ماژول حقوق و دستمزد) — نظام پرداخت دوم سازمان
   ---------------------------------------------------------------------
   چرا این صفحه: سازمان دو نظام پرداخت دارد. نظام عمومی (تأمین اجتماعی) مزد مبنا و اقلام
   خودش را دارد؛ کارکنان تابع صندوق بازنشستگی نفت ساختار دیگری دارند:

       پایهٔ حقوق
       + فوق‌العادهٔ بدی آب و هوا (مناطق گرمسیری) — درصدی از پایهٔ حقوق
       = «مبنای فوق‌العاده‌ها»
       بقیهٔ فوق‌العاده‌ها (محرومیت، ویژه، صرف وقت، کارگاهی، جذب و تخصصی شغل،
       فرآورشی و نوبت‌کار) — درصدی از همین مبنا
       − کسور صندوق، بیمه و مالیات
       = خالص پرداختی

   محاسبه در سرور است (`POST /api/v1/payroll/oil-payslip`) و این فایل هیچ درصدی را
   خودش حساب نمی‌کند: درصدهای مصوب (`oilpay.*`)، فهرست اقلام و مبناها همه از سرور می‌آید.

     GET  /api/v1/payroll/oil-payslip-defaults   پارامترها، اقلام، کارکنان
     POST /api/v1/payroll/oil-payslip            محاسبهٔ فیش و تفکیک اقلام
     POST /api/v1/payroll/records                ثبت فیش به‌عنوان رکورد (نوع oilPayslips)
   ===================================================================== */

var OIL_DEFAULTS = null;   // پارامترها و فهرست‌های مرجع
var OIL_RESULT = null;     // آخرین فیش محاسبه‌شده
var OIL_EMPLOYEES = [];    // کارکنان
var OIL_TIMER = null;      // زمان‌سنج محاسبهٔ خودکار

/** مقدار عددی یک ورودی (خالی = صفر). */
function oilNum(id) {
  const el = document.getElementById(id);
  if (!el) return 0;
  return Number((el.value || '').replace(/[^0-9.\-]/g, '')) || 0;
}

/** متن یک ورودی. */
function oilText(id) {
  const el = document.getElementById(id);
  return el ? (el.value || '').trim() : '';
}

/** مقدار یک فهرست کشویی. */
function oilSelect(id) {
  const el = document.getElementById(id);
  return el ? el.value : '';
}

/** درصد دستی هر قلم فوق‌العاده از فرم. */
function oilAllowancePercents() {
  const out = {};
  for (const a of ((OIL_DEFAULTS && OIL_DEFAULTS.allowances) || [])) {
    const el = document.getElementById('oilAllow_' + a.key);
    if (!el) continue;
    const value = Number((el.value || '').replace(/[^0-9.\-]/g, ''));
    if (!isNaN(value) && el.value !== '') out[a.key] = value;
  }
  return out;
}

/** ساخت بدنهٔ درخواست محاسبه از فیلدهای فرم. */
function oilInput() {
  return {
    baseSalary: oilNum('oilBaseSalary'),
    climatePercent: oilNum('oilClimatePercent'),
    allowancePercents: oilAllowancePercents(),
    shift: oilSelect('oilShift') === 'yes',
    shiftDays: oilNum('oilShiftDays'),
    daysInMonth: oilNum('oilDaysInMonth'),
    overtimeHours: oilNum('oilOvertime'),
    otherDeductions: oilNum('oilDeductions'),
  };
}

/** محاسبه با تأخیر کوتاه (برای تایپ). */
function oilCalcSoon() {
  if (OIL_TIMER) clearTimeout(OIL_TIMER);
  OIL_TIMER = setTimeout(oilCalc, 250);
}

/** فراخوانی محاسبهٔ فیش و بازترسیم. */
async function oilCalc() {
  const box = document.getElementById('oilSlip');
  if (box) box.innerHTML = '<div class="muted" style="font-size:.78rem">⏳ در حال محاسبه...</div>';
  const res = await apiFetch('/api/v1/payroll/oil-payslip', { method: 'POST', body: oilInput() });
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    return;
  }
  OIL_RESULT = res.data || {};
  oilRenderSlip();
}

/** نمایش فیش: کارت‌های جمع، تفکیک اقلام و مبنای محاسبه. */
function oilRenderSlip() {
  const box = document.getElementById('oilSlip');
  if (!box || !OIL_RESULT || !OIL_RESULT.slip) return;
  const s = OIL_RESULT.slip;

  const rows = (s.lines || []).map(l => {
    const isDeduct = l.groupFa === 'کسور';
    const amount = isDeduct ? '-' + money(l.amount) : money(l.amount);
    const color = isDeduct ? 'color:#dc2626' : '';
    return `<tr>
      <td>${esc(l.titleFa)}</td>
      <td style="font-size:.72rem">${esc(l.groupFa || '')}</td>
      <td class="num">${l.percent ? fa(Math.round(l.percent * 10) / 10) + '٪' : '—'}</td>
      <td class="num">${l.base ? fa(money(l.base)) : '—'}</td>
      <td class="num" style="${color}"><b>${fa(amount)}</b></td>
      <td style="font-size:.74rem">${esc(l.detailFa || '—')}</td></tr>`;
  });

  box.innerHTML = `
    <div class="grid-stats">
      ${statCard({ico: '💼', tint: '#EFF6FF', accent: '#2563EB', val: fa(money(s.baseSalary)), label: 'پایهٔ حقوق (ریال)'})}
      ${statCard({ico: '🌡️', tint: '#FEF3C7', accent: '#D97706',
        val: fa(money(s.climate)), label: 'فوقالعادهٔ بدی آب و هوا ' + fa(Math.round(s.climatePercent * 10) / 10) + '٪'})}
      ${statCard({ico: '🧮', tint: '#EDE9FE', accent: '#7C3AED', val: fa(money(s.allowanceBase)), label: 'مبنای فوقالعادهها (ریال)'})}
      ${statCard({ico: '➕', tint: '#DCFCE7', accent: '#16A34A', val: fa(money(s.gross)), label: 'جمع ناخالص (ریال)'})}
      ${statCard({ico: '➖', tint: '#FEE2E2', accent: '#DC2626', val: fa(money(s.deductions)), label: 'جمع کسور (ریال)'})}
      ${statCard({ico: '💵', tint: '#DCFCE7', accent: '#16A34A', val: fa(money(s.net)), label: 'خالص پرداختی (ریال)'})}
    </div>
    <div class="card card-pad" style="margin-top:10px">
      <div class="card-title">تفکیک اقلام فیش — ${esc(s.schemeFa || '')}</div>
      ${liveTable(['قلم', 'گروه', 'درصد', 'مبنای محاسبه (ریال)', 'مبلغ (ریال)', 'شرح'],
        rows, rows.length)}
      <div class="muted" style="font-size:.74rem;margin-top:6px">
        کسور سهم کارفرما (صندوق بازنشستگی نفت): <b class="num">${fa(money(s.fundEmployer || 0))}</b> ریال
      </div>
    </div>
    <div class="card card-pad" style="margin-top:10px;font-size:.8rem;line-height:1.9">
      ${esc(OIL_RESULT.noteFa || '').split('\n').join('<br>')}
    </div>`;
}

/** با انتخاب کارمند، پایهٔ حقوق پرونده در فرم می‌نشیند. */
function oilOnEmployee() {
  const hit = OIL_EMPLOYEES.find(e => e.id === oilSelect('oilEmployee'));
  const field = document.getElementById('oilBaseSalary');
  if (hit && field) {
    const base = Number(hit.baseSalary || 0);
    if (base > 0) field.value = String(base);
  }
  oilCalcSoon();
}

/** با نوبت‌کار کردن کارمند، روزهای ماه برای نوبت کامل پر می‌شود. */
function oilOnShift() {
  const days = document.getElementById('oilShiftDays');
  const month = oilNum('oilDaysInMonth');
  if (days && oilSelect('oilShift') === 'yes' && (!days.value || Number(days.value) === 0)) {
    days.value = String(month || 30);
  }
  oilCalcSoon();
}

/* --------------------------- ثبت فیش در پرونده --------------------------- */

/** ثبت فیش محاسبه‌شده به‌عنوان رکورد «فیش حقوقی صندوق نفت». */
async function oilSave() {
  if (!OIL_RESULT || !OIL_RESULT.slip) { toast('ابتدا فیش محاسبه شود', 'error'); return; }
  const employeeId = oilSelect('oilEmployee');
  if (!employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  const s = OIL_RESULT.slip;
  const period = oilText('oilPeriod') || OIL_RESULT.period || '';
  const employeeName = (OIL_EMPLOYEES.find(e => e.id === employeeId) || {}).titleFa || employeeId;
  const res = await apiFetch('/api/v1/payroll/records', {
    method: 'POST',
    body: {
      kind: 'oilPayslips',
      title: 'فیش حقوقی صندوق نفت — ' + employeeName.split(' — ')[0] + ' — ' + period,
      employeeId: employeeId,
      fields: {
        period: period,
        baseSalary: String(s.baseSalary),
        climatePercent: String(s.climatePercent),
        climate: String(s.climate),
        allowanceBase: String(s.allowanceBase),
        allowancesTotal: String(s.allowances),
        gross: String(s.gross),
        deductions: String(s.deductions),
        net: String(s.net),
        shift: oilSelect('oilShift'),
        shiftDays: String(oilNum('oilShiftDays')),
        overtimeHours: String(oilNum('oilOvertime')),
        note: 'مبنای فوقالعاده‌ها: ' + money(s.allowanceBase) + ' ریال',
      },
      noteFa: 'فیش نظام پرداخت صندوق بازنشستگی نفت برای دورهٔ ' + period,
    },
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = (res.data && res.data.id) || '';
  toast('فیش ثبت شد: ' + id);
  const saved = document.getElementById('oilSaved');
  if (saved) {
    saved.innerHTML = `<div class="card card-pad" style="margin-top:10px;font-size:.82rem;line-height:2">
      فیش <b class="ltr">${esc(id)}</b> ثبت شد — خالص <b class="num">${fa(money(s.net))}</b> ریال.
      گام بعدی: تأیید فیش و ثبت پرداخت در فهرست رکوردهای همین نوع.
      <div class="row-actions" style="margin-top:8px">
        <button class="btn btn-sm btn-primary" onclick="oilAct('${esc(id)}','approve')">✅ تأیید فیش</button>
        <button class="btn btn-sm btn-ghost" onclick="oilAct('${esc(id)}','issue')">📄 صدور فیش</button>
        <button class="btn btn-sm btn-ghost" onclick="oilAct('${esc(id)}','pay')">💰 ثبت پرداخت</button>
      </div></div>`;
  }
}

/** انجام یک اقدام گردش‌کار روی فیش ثبت‌شده. */
async function oilAct(id, action) {
  const res = await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(id) + '/action',
    { method: 'POST', body: { action: action } });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('اقدام انجام شد: ' + action);
}

/** چاپ فیش جاری (نسخهٔ کاغذی همان تفکیک اقلام). */
function oilPrint() {
  if (!OIL_RESULT || !OIL_RESULT.slip) { toast('ابتدا فیش محاسبه شود', 'error'); return; }
  const s = OIL_RESULT.slip;
  const employee = (OIL_EMPLOYEES.find(e => e.id === oilSelect('oilEmployee')) || {}).titleFa || '';
  const rows = (s.lines || []).map(l => `<tr>
    <td>${esc(l.titleFa)}</td>
    <td>${l.percent ? fa(Math.round(l.percent * 10) / 10) + '٪' : '—'}</td>
    <td>${l.base ? fa(money(l.base)) : '—'}</td>
    <td>${fa(l.groupFa === 'کسور' ? '-' + money(l.amount) : money(l.amount))}</td>
    <td>${esc(l.detailFa || '')}</td></tr>`).join('');
  const win = window.open('', '_blank');
  if (!win) { toast('پنجرهٔ چاپ باز نشد', 'error'); return; }
  win.document.write(`<!doctype html><html dir="rtl" lang="fa"><head><meta charset="utf-8">
    <title>فیش حقوقی صندوق نفت</title>
    <style>body{font-family:Vazirmatn,Tahoma,sans-serif;padding:24px;color:#111}
    h1{font-size:18px;margin:0 0 4px} .sub{color:#555;font-size:12px;margin-bottom:14px}
    table{width:100%;border-collapse:collapse;margin-top:10px}
    th,td{border:1px solid #ccc;padding:6px;font-size:12px;text-align:right}
    th{background:#f3f4f6} .tot{margin-top:12px;font-size:13px;line-height:2}</style></head><body>
    <h1>فیش حقوقی — نظام پرداخت صندوق بازنشستگی نفت</h1>
    <div class="sub">${esc(employee)} | دورهٔ ${fa(oilText('oilPeriod'))}</div>
    <table><thead><tr><th>قلم</th><th>درصد</th><th>مبنای محاسبه (ریال)</th><th>مبلغ (ریال)</th><th>شرح</th></tr></thead>
    <tbody>${rows}</tbody></table>
    <div class="tot">
      پایهٔ حقوق: <b>${fa(money(s.baseSalary))}</b> ریال ·
      فوق‌العادهٔ بدی آب و هوا: <b>${fa(money(s.climate))}</b> ریال ·
      مبنای فوق‌العاده‌ها: <b>${fa(money(s.allowanceBase))}</b> ریال<br>
      جمع ناخالص: <b>${fa(money(s.gross))}</b> ریال ·
      جمع کسور: <b>${fa(money(s.deductions))}</b> ریال ·
      خالص پرداختی: <b>${fa(money(s.net))}</b> ریال
    </div></body></html>`);
  win.document.close();
  win.focus();
  win.print();
}

/* ------------------------------ ساخت فرم ------------------------------ */

/** یک ردیف ورودی عددی. */
function oilField(id, label, value, unit, hint) {
  return `<div class="field"><label>${esc(label)}${unit ? ` <span class="muted">(${esc(unit)})</span>` : ''}</label>
    <input id="${id}" class="num" value="${esc(value)}" placeholder="${esc(hint || '')}" oninput="oilCalcSoon()"></div>`;
}

/** صفحهٔ «فیش حقوقی صندوق نفت». */
function pageOilPayslipLive() {
  if (!isLive()) return offlineFallback('oilPayslip');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/payroll/oil-payslip-defaults');
    if (!res.ok) return liveErrorBox(res.message);
    OIL_DEFAULTS = res.data || {};
    OIL_EMPLOYEES = OIL_DEFAULTS.employees || [];
    const def = OIL_DEFAULTS;
    const params = def.params || {};
    const today = def.today || '';
    const period = def.period || '';
    const empOpts = OIL_EMPLOYEES.map(e => `<option value="${esc(e.id)}">${esc(e.titleFa)}</option>`).join('');
    const firstBase = (OIL_EMPLOYEES[0] && OIL_EMPLOYEES[0].baseSalary) || 0;

    // اقلام فوق‌العاده: درصد مصوب شرکت در ورودی می‌آید تا فرم با عدد واقعی باز شود.
    const allowanceFields = (def.allowances || []).map(a => `<div class="field">
      <label>${esc(a.titleFa)} <span class="muted">(${esc(a.base === 'baseSalary' ? 'از پایهٔ حقوق' : 'از مبنای فوق‌العاده‌ها')})</span></label>
      <input id="oilAllow_${esc(a.key)}" class="num" value="${esc(String(a.percent))}" oninput="oilCalcSoon()"></div>`).join('');

    const shiftOpts = (def.shiftOptions || []).map(s =>
      `<option value="${esc(s.key)}">${esc(s.titleFa)}</option>`).join('');

    const form = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">💼 نظام پرداخت صندوق بازنشستگی نفت</div>
      <div class="form-grid">
        <div class="field"><label>کارمند <span class="req">*</span></label>
          <select id="oilEmployee" onchange="oilOnEmployee()">${empOpts}</select></div>
        <div class="field"><label>دورهٔ حقوق (ماه)</label>
          <input id="oilPeriod" class="num" value="${esc(period)}" oninput="oilCalcSoon()"></div>
        ${oilField('oilBaseSalary', 'پایهٔ حقوق', String(firstBase), 'ریال', 'از پروندهٔ کارمند پر می‌شود')}
        ${oilField('oilClimatePercent', 'درصد فوق‌العادهٔ بدی آب و هوا (گرمسیری)', String(params.climatePercent || 0), '٪', 'مصوب شرکت')}
        ${oilField('oilDaysInMonth', 'روزهای ماه', String(def.daysInMonth || 30), 'روز')}
        <div class="field"><label>وضعیت نوبت‌کاری</label>
          <select id="oilShift" onchange="oilOnShift()">${shiftOpts}</select></div>
        ${oilField('oilShiftDays', 'روزهای نوبت‌کاری در ماه', String(def.daysInMonth || 30), 'روز')}
        ${oilField('oilOvertime', 'اضافه‌کاری', '0', 'ساعت')}
        ${oilField('oilDeductions', 'سایر کسور (وام/مساعده)', '0', 'ریال')}
      </div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">📊 فوق‌العاده‌های مصوب (درصدی از مبنای فوق‌العاده‌ها)</div>
      <div class="form-grid">${allowanceFields || '<div class="muted">قلمی تعریف نشده است.</div>'}</div>
      <div class="muted" style="font-size:.74rem;margin-top:6px">${esc(def.noteFa || '').split('\n').join('<br>')}</div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">🧾 فیش محاسبه‌شده</div>
      <div id="oilSlip"><div class="muted" style="font-size:.78rem">در حال محاسبه...</div></div>
      <div class="row-actions" style="margin-top:10px">
        <button class="btn btn-primary" onclick="oilCalc()">🧮 محاسبهٔ مجدد</button>
        <button class="btn btn-ghost" onclick="oilPrint()">🖨️ چاپ فیش</button>
        <button class="btn btn-ghost" onclick="oilSave()">💾 ثبت فیش در پرونده</button>
      </div>
      <div id="oilSaved"></div>
    </div>`;

    const paramRows = (def.paramViews || []).map(p => `<tr>
      <td>${esc(p.titleFa)}</td><td class="ltr" style="font-size:.72rem">${esc(p.key)}</td>
      <td class="num">${fa(p.value)}</td><td>${esc(p.unitFa)}</td></tr>`);

    const shell = form + (paramRows.length ? `<div class="card card-pad">
      <div class="card-title">پارامترهای نظام پرداخت صندوق نفت در شرکت فعال</div>
      ${liveTable(['پارامتر', 'کلید', 'مقدار', 'واحد'], paramRows, paramRows.length)}
      <div class="muted" style="font-size:.74rem;margin-top:6px">تاریخ امروز سرور: <b class="num">${fa(today)}</b></div>
    </div>` : '');

    setTimeout(() => { oilOnEmployee(); oilCalc(); }, 80);
    return shell;
  });
  return liveShell('فیش حقوقی صندوق نفت',
    'پایهٔ حقوق، فوق‌العادهٔ بدی آب و هوا و فوق‌العاده‌های مصوب روی مبنای فوق‌العاده‌ها',
    `<button class="btn btn-ghost" onclick="OIL_DEFAULTS=null;liveReload()">↻ تازه‌سازی پارامترها</button>`);
}

/* ثبت در مسیریاب زنده. */
LIVE_PAGES.oilPayslip = pageOilPayslipLive;