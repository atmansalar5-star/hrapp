/* =====================================================================
   صفحهٔ زندهٔ «پاداش پایان خدمت (سنوات)» — ماژول حقوق و دستمزد
   =====================================================================

   این لایه هیچ قاعدهٔ محاسبه‌ای ندارد: فرمول‌ها در سرور (بستهٔ domain ماژول حقوق) و
   بر پایهٔ صورت‌حساب مرجع صنعت نفت اجرا می‌شوند. اینجا فقط:

     ۱. پارامترها و جدول‌های مرجع از /payroll/service-bonus/parameters خوانده و فرم
        ساخته می‌شود (هیچ درصدی در جاوااسکریپت هاردکد نیست)،
     ۲. ورودی‌ها از کاربر یا از پروندهٔ پرسنلی و آخرین حکم ابلاغ‌شده بارگذاری می‌شود،
     ۳. صورت‌حساب از /calculate گرفته و نمایش داده می‌شود،
     ۴. و پروندهٔ تأییدشدنی با همان پروسیژرهای رکوردها (ایجاد/ویرایش/جستجو/حذف)
        ذخیره می‌شود.
   ===================================================================== */

/** حالت صفحه: پارامترها، کارکنان، آخرین محاسبه و فهرست پرونده‌ها. */
const SBF = {params: null, employees: [], result: null, files: [], query: ''};

const SBF_PREFIX = '/api/v1/payroll';
const sbfMoney = v => (v == null ? '—' : faNum(money(Math.round(Number(v) || 0))));
/** تبدیل سهم (۰٫۲۵) به درصد نمایشی (۲۵) و برعکس. */
const sbfToPercent = ratio => Math.round((Number(ratio) || 0) * 10000) / 100;
const sbfToRatio = percent => (Number(percent) || 0) / 100;

function pageServiceBonusLive() {
  if (!isLive()) return offlineFallback('serviceBonus');
  liveLoad(async () => {
    const [params, employees, files] = await Promise.all([
      apiFetch(SBF_PREFIX + '/service-bonus/parameters'),
      apiFetch('/api/v1/employee/employees?size=200'),
      apiFetch(SBF_PREFIX + '/records?kind=serviceBonusFiles&size=200' + (SBF.query ? '&query=' + encodeURIComponent(SBF.query) : '')),
    ]);
    if (!params.ok) return liveErrorBox(params.message);
    SBF.params = params.data;
    SBF.employees = (employees.ok && employees.data && employees.data.items) || [];
    SBF.files = (files.ok && files.data && files.data.items) || [];
    return sbfHTML();
  });
  return liveShell('پاداش پایان خدمت (سنوات)',
    'محاسبهٔ پاداش سنوات کارکنان رسمی بر پایهٔ صورت‌حساب مرجع؛ داده از پروندهٔ پرسنلی، حکم ابلاغ‌شده و پارامترهای حقوق',
    `<button class="btn btn-ghost" onclick="sbfLoadFromPersonnel()">⬇️ بارگذاری از پرونده و حکم</button>
     <button class="btn btn-primary" onclick="sbfCalculate()">🧮 محاسبه</button>`);
}

/** فرم ورودی + منطقهٔ نتیجه + جدول پرونده‌ها. */
function sbfHTML() {
  const p = SBF.params || {rates: {}, items: [], degrees: []};
  const r = p.rates || {};
  const empOpts = SBF.employees.map(e =>
    `<option value="${esc(e.id)}">${esc(e.fullName)} — ${esc(e.departmentName || '')}</option>`).join('');
  const row = (id, label, value, step) =>
    `<label class="muted" style="font-size:.75rem">${label}
       <input class="inp" id="${id}" type="number" step="${step || 'any'}" value="${value == null ? '' : value}"
              style="direction:ltr;text-align:left"></label>`;

  const degrees = (p.degrees || []).map(d =>
    `<span class="chip c-wait" style="margin-inline-end:4px">${esc(d.titleFa)}: ${esc(fa(sbfToPercent(d.value)))}٪</span>`).join('');

  return `
  <div class="card card-pad" style="margin-bottom:12px">
    <div class="grid-stats" style="grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:8px">
      <label class="muted" style="font-size:.75rem">کارمند
        <select class="sel" id="sbfEmp" style="width:100%"><option value="">— انتخاب کارمند —</option>${empOpts}</select></label>
      <label class="muted" style="font-size:.75rem">تاریخ مبنای محاسبه (شمسی)
        <input class="inp" id="sbfBaseDate" placeholder="1404/07/01" value="${esc((SBF.params && SBF.params.todayFa) || '')}"
               style="direction:ltr;text-align:left"></label>
      ${row('sbfBaseSalary', 'حقوق پایه (ریال)', '')}
      ${row('sbfRepair', 'ترمیم حقوق (ریال)', '')}
    </div>
    <div style="margin-top:10px;font-size:.78rem;color:var(--muted)">سابقهٔ خدمت (سال / ماه / روز)</div>
    <div class="grid-stats" style="grid-template-columns:repeat(auto-fit,minmax(120px,1fr));gap:8px">
      ${row('sbfNYear', 'شمال — سال', 0, '1')} ${row('sbfNMonth', 'شمال — ماه', 0, '1')} ${row('sbfNDay', 'شمال — روز', 0, '1')}
      ${row('sbfSYear', 'جنوب — سال', 0, '1')} ${row('sbfSMonth', 'جنوب — ماه', 0, '1')} ${row('sbfSDay', 'جنوب — روز', 0, '1')}
    </div>
    <div style="margin-top:10px;font-size:.78rem;color:var(--muted)">درصد فوق‌العاده‌ها (پیش‌فرض از پارامترهای شرکت؛ قابل بازنویسی)</div>
    <div class="grid-stats" style="grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:8px">
      ${row('sbfBadClimate', 'بدی آب و هوا (٪)', sbfToPercent(r.badClimate))}
      ${row('sbfSpecial', 'ویژه (٪)', sbfToPercent(r.special))}
      ${row('sbfWorkshop', 'کارگاهی (٪)', sbfToPercent(r.workshop))}
      ${row('sbfAttractionPublic', 'جذب عمومی (٪)', sbfToPercent(r.attractionPublic))}
      ${row('sbfPersonalBase', 'جذب پایه شخصی (٪)', sbfToPercent(r.personalBase))}
      ${row('sbfJobSpecialty', 'تخصصی شغل/مدرک (٪)', sbfToPercent(r.jobSpecialty))}
    </div>
    <div style="margin-top:8px;font-size:.74rem;color:var(--muted)">جدول مرجع مدرک تحصیلی: ${degrees || '—'}</div>
    <div style="margin-top:12px;display:flex;gap:8px;flex-wrap:wrap">
      <button class="btn btn-primary" onclick="sbfCalculate()">🧮 محاسبهٔ پاداش</button>
      <button class="btn btn-ghost" onclick="sbfLoadFromPersonnel()">⬇️ بارگذاری از پرونده و حکم</button>
      <button class="btn btn-ghost" onclick="sbfSaveFile()">💾 ثبت پروندهٔ محاسبه</button>
    </div>
  </div>
  <div id="sbfResult">${sbfResultHTML()}</div>
  ${sbfFilesHTML()}`;
}

/** صورت‌حساب: دو ستون (شمال و جنوب) + سطرهای پاداش و جمع نهایی. */
function sbfResultHTML() {
  const res = SBF.result;
  if (!res) return `<div class="card card-pad muted">برای دیدن صورت‌حساب، کارمند را انتخاب و «محاسبه» را بزنید.</div>`;
  const line = l => `<tr><td>${esc(l.titleFa)}</td>
      <td class="num">${esc(fa(sbfToPercent(l.percent)))}٪</td>
      <td class="muted" style="font-size:.72rem">${esc(l.baseFa || '')}</td>
      <td class="num">${sbfMoney(l.amountRial)}</td></tr>`;
  const block = (title, lines, total) => `
    <div class="card">
      <div class="card-pad" style="font-weight:700">${title}</div>
      <table class="tbl"><thead><tr><th>عنوان</th><th>درصد</th><th>مبنا</th><th>مبلغ (ریال)</th></tr></thead>
      <tbody>${lines.map(line).join('')}
      <tr><td><b>جمع ستون</b></td><td></td><td class="muted" style="font-size:.72rem">با ترمیم حقوق</td><td class="num"><b>${sbfMoney(total)}</b></td></tr>
      </tbody></table>
    </div>`;

  return `
  <div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
    <b>${esc(res.employeeName || res.employeeId || '')}</b>
    ${res.noteFa ? `<span class="muted"> — ${esc(res.noteFa)}</span>` : ''}
    <div class="muted">تاریخ مبنا: ${faDate(res.baseDate)} — سابقهٔ شمال: ${esc(fa(res.northYears))} سال
      — سابقهٔ جنوب: ${esc(fa(res.southYears))} سال (در محاسبهٔ ۸٪: ${esc(fa(res.southYearsUsed))})
      — جمع سابقه: ${esc(fa(res.totalTenure.years))} سال و ${esc(fa(res.totalTenure.months))} ماه و ${esc(fa(res.totalTenure.days))} روز</div>
    ${(res.sources || []).length ? `<div class="muted">منابع داده: ${esc((res.sources || []).join('، '))}</div>` : ''}
  </div>
  <div class="grid-2" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:12px">
    ${block('شمال (بدی آب و هوا صفر)', res.northLines || [], res.northTotal)}
    ${block('جنوب', res.southLines || [], res.southTotal)}
  </div>
  <div class="card card-pad" style="margin-top:12px">
    <table class="tbl"><tbody>
      <tr><td>پاداش حق جذب ۸ درصد — شمال</td><td class="num">${sbfMoney(res.attractionBonusNorth)}</td></tr>
      <tr><td>پاداش حق جذب ۸ درصد — جنوب</td><td class="num">${sbfMoney(res.attractionBonusSouth)}</td></tr>
      <tr><td><b>پاداش حق جذب ۸ درصد (جمع)</b></td><td class="num"><b>${sbfMoney(res.attractionBonus)}</b></td></tr>
      <tr><td>پاداش پایه شخصی و تخصصی — شمال</td><td class="num">${sbfMoney(res.personalBonusNorth)}</td></tr>
      <tr><td>پاداش پایه شخصی و تخصصی — جنوب</td><td class="num">${sbfMoney(res.personalBonusSouth)}</td></tr>
      <tr><td><b>مبلغ محاسبه‌شدهٔ پاداش سنوات</b></td><td class="num"><b>${sbfMoney(res.totalRial)}</b></td></tr>
      <tr><td>نیمی از پاداش (سطر کمکی صورت‌حساب)</td><td class="num">${sbfMoney(res.halfTotalRial)}</td></tr>
    </tbody></table>
  </div>`;
}

/** جدول پرونده‌های ثبت‌شده با جستجو و پروسیژرهای ویرایش/حذف/اقدام. */
function sbfFilesHTML() {
  const rows = (SBF.files || []).map(f => {
    const st = (f.statusFa || f.status || '');
    const act = (f.actions || []).map(a =>
      `<button class="btn btn-sm ${a.style === 'primary' ? 'btn-primary' : 'btn-ghost'}"
        onclick="sbfAct('${esc(f.id)}','${esc(a.value)}','${esc(a.labelFa)}')">${esc(a.labelFa)}</button>`).join(' ');
    const editable = !f.status || f.status === 'draft';
    return `<tr>
      <td class="ltr">${esc(f.id)}</td>
      <td>${personCell(f.employeeId, f.employeeName)}</td>
      <td class="num">${esc(fa(f.fields.baseDate || ''))}</td>
      <td class="num">${sbfMoney(f.fields.total)}</td>
      <td>${statusChip(f.status, st)}</td>
      <td>${act}
        ${editable ? `<button class="btn btn-sm btn-ghost" title="ویرایش" onclick="sbfEdit('${esc(f.id)}')">✏️</button>
        <button class="btn btn-sm btn-ghost" title="حذف" onclick="sbfDelete('${esc(f.id)}')">🗑️</button>` : ''}</td>
    </tr>`;
  });
  return `
  <div class="card card-pad" style="margin-top:16px">
    <div class="toolbar" style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
      <b>پرونده‌های پاداش پایان خدمت</b>
      <input class="inp w-lg" id="sbfQuery" placeholder="🔍 جستجوی کارمند، شناسه یا مبلغ…" value="${esc(SBF.query || '')}">
      <button class="btn btn-ghost btn-sm" onclick="sbfSearch()">جستجو</button>
      <button class="btn btn-ghost btn-sm" onclick="SBF.query='';liveReload()">پاک‌سازی</button>
      <span class="muted" style="margin-inline-start:auto;font-size:.75rem">تعداد: ${esc(fa((SBF.files || []).length))}</span>
    </div>
    ${liveTable(['شناسه', 'کارمند', 'تاریخ مبنا', 'مبلغ پاداش', 'وضعیت', 'عملیات'], rows, (SBF.files || []).length)}
  </div>`;
}

/* ------------------------------ عملیات ------------------------------ */

function sbfNum(id) {
  const el = document.getElementById(id);
  return el && el.value !== '' ? Number(el.value) : 0;
}
function sbfVal(id) {
  const el = document.getElementById(id);
  return el ? el.value.trim() : '';
}

/** ورودی محاسبه را از فرم می‌خواند (درصدها به سهم تبدیل می‌شوند). */
function sbfBodyFromForm() {
  return {
    employeeId: sbfVal('sbfEmp'),
    baseDate: sbfVal('sbfBaseDate'),
    baseSalary: sbfNum('sbfBaseSalary'),
    salaryRepair: sbfNum('sbfRepair'),
    north: {years: sbfNum('sbfNYear'), months: sbfNum('sbfNMonth'), days: sbfNum('sbfNDay')},
    south: {years: sbfNum('sbfSYear'), months: sbfNum('sbfSMonth'), days: sbfNum('sbfSDay')},
    rates: {
      badClimate: sbfToRatio(sbfNum('sbfBadClimate')),
      special: sbfToRatio(sbfNum('sbfSpecial')),
      workshop: sbfToRatio(sbfNum('sbfWorkshop')),
      attractionPublic: sbfToRatio(sbfNum('sbfAttractionPublic')),
      personalBase: sbfToRatio(sbfNum('sbfPersonalBase')),
      jobSpecialty: sbfToRatio(sbfNum('sbfJobSpecialty')),
    },
  };
}

/** پرکردن فرم از پاسخ «پیش‌پرکردن» سرور. */
function sbfFillForm(d) {
  const set = (id, v) => { const el = document.getElementById(id); if (el) el.value = v == null ? '' : v; };
  set('sbfBaseDate', d.baseDate || '');
  set('sbfBaseSalary', d.baseSalary || '');
  set('sbfRepair', d.salaryRepair || '');
  set('sbfNYear', d.north ? d.north.years : 0);
  set('sbfNMonth', d.north ? d.north.months : 0);
  set('sbfNDay', d.north ? d.north.days : 0);
  set('sbfSYear', d.south ? d.south.years : 0);
  set('sbfSMonth', d.south ? d.south.months : 0);
  set('sbfSDay', d.south ? d.south.days : 0);
  if (d.rates) {
    set('sbfBadClimate', sbfToPercent(d.rates.badClimate));
    set('sbfSpecial', sbfToPercent(d.rates.special));
    set('sbfWorkshop', sbfToPercent(d.rates.workshop));
    set('sbfAttractionPublic', sbfToPercent(d.rates.attractionPublic));
    set('sbfPersonalBase', sbfToPercent(d.rates.personalBase));
    set('sbfJobSpecialty', sbfToPercent(d.rates.jobSpecialty));
  }
}

/** بارگذاری پایهٔ حقوق و سابقه از پروندهٔ پرسنلی و آخرین حکم ابلاغ‌شده. */
async function sbfLoadFromPersonnel() {
  const emp = sbfVal('sbfEmp');
  if (!emp) { toast('اول کارمند را انتخاب کنید', 'error'); return; }
  const q = '?employee=' + encodeURIComponent(emp) + (sbfVal('sbfBaseDate') ? '&baseDate=' + encodeURIComponent(sbfVal('sbfBaseDate')) : '');
  const res = await apiFetch(SBF_PREFIX + '/service-bonus/prefill' + q);
  if (!res.ok) { toast(res.message, 'error'); return; }
  sbfFillForm(res.data);
  SBF.result = null;
  document.getElementById('sbfResult').innerHTML = sbfResultHTML();
  toast('اطلاعات از ' + ((res.data.sources || ['پرونده']).join(' و ')));
}

async function sbfCalculate() {
  const body = sbfBodyFromForm();
  if (!body.employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  if (!body.baseDate) { toast('تاریخ مبنای محاسبه را وارد کنید', 'error'); return; }
  const res = await apiFetch(SBF_PREFIX + '/service-bonus/calculate', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  SBF.result = res.data;
  document.getElementById('sbfResult').innerHTML = sbfResultHTML();
  toast('پاداش سنوات محاسبه شد');
}

/** ثبت صورت‌حساب محاسبه‌شده به‌عنوان پروندهٔ تأییدشدنی. */
async function sbfSaveFile() {
  const r = SBF.result;
  if (!r) { toast('اول محاسبه را انجام دهید', 'error'); return; }
  const body = {
    kind: 'serviceBonusFiles',
    title: 'پاداش پایان خدمت — ' + (r.employeeName || r.employeeId),
    employeeId: r.employeeId,
    noteFa: r.noteFa || '',
    fields: {
      baseDate: r.baseDate || '',
      baseSalary: String(r.baseSalary || 0),
      salaryRepair: String(r.salaryRepair || 0),
      northTenure: String(r.northYears || 0),
      southTenure: String(r.southYears || 0),
      totalTenure: String((r.northYears || 0) + (r.southYears || 0)),
      badClimate: String((r.rates && r.rates.badClimate) || 0),
      special: String((r.rates && r.rates.special) || 0),
      workshop: String((r.rates && r.rates.workshop) || 0),
      attractionPublic: String((r.rates && r.rates.attractionPublic) || 0),
      personalBase: String((r.rates && r.rates.personalBase) || 0),
      jobSpecialty: String((r.rates && r.rates.jobSpecialty) || 0),
      attractionBonus: String(Math.round(r.attractionBonus || 0)),
      personalBonus: String(Math.round((r.personalBonusNorth || 0) + (r.personalBonusSouth || 0))),
      total: String(Math.round(r.totalRial || 0)),
    },
  };
  const res = await apiFetch(SBF_PREFIX + '/records', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('پروندهٔ پاداش ثبت شد');
  liveReload();
}

function sbfSearch() {
  SBF.query = sbfVal('sbfQuery');
  liveReload();
}

/** ویرایش پروندهٔ پیش‌نویس (سابقه، پایه و ترمیم حقوق). */
function sbfEdit(id) {
  const f = (SBF.files || []).find(x => x.id === id);
  if (!f) return;
  const n = k => esc(f.fields[k] || '');
  openModal({
    title: 'ویرایش پروندهٔ ' + id,
    narrow: true,
    body: `<div style="display:grid;gap:8px">
      <label class="muted" style="font-size:.75rem">تاریخ مبنا <input class="inp" id="sbfE_baseDate" value="${n('baseDate')}"></label>
      <label class="muted" style="font-size:.75rem">حقوق پایه <input class="inp" id="sbfE_baseSalary" value="${n('baseSalary')}"></label>
      <label class="muted" style="font-size:.75rem">ترمیم حقوق <input class="inp" id="sbfE_salaryRepair" value="${n('salaryRepair')}"></label>
      <label class="muted" style="font-size:.75rem">سابقه شمال (سال) <input class="inp" id="sbfE_northTenure" value="${n('northTenure')}"></label>
      <label class="muted" style="font-size:.75rem">سابقه جنوب (سال) <input class="inp" id="sbfE_southTenure" value="${n('southTenure')}"></label>
      <label class="muted" style="font-size:.75rem">مبلغ پاداش <input class="inp" id="sbfE_total" value="${n('total')}"></label>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="sbfSaveEdit('${esc(id)}')">ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}

async function sbfSaveEdit(id) {
  const f = (SBF.files || []).find(x => x.id === id);
  if (!f) return;
  const fields = Object.assign({}, f.fields);
  for (const k of ['baseDate', 'baseSalary', 'salaryRepair', 'northTenure', 'southTenure', 'total']) {
    const el = document.getElementById('sbfE_' + k);
    if (el) fields[k] = el.value.trim();
  }
  const body = {kind: 'serviceBonusFiles', title: f.title, employeeId: f.employeeId, noteFa: f.noteFa || '', fields};
  const res = await apiFetch(SBF_PREFIX + '/records/' + encodeURIComponent(id), {method: 'PATCH', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal();
  toast('پرونده ویرایش شد');
  liveReload();
}

function sbfDelete(id) {
  requireConfirm('این پروندهٔ پاداش حذف شود؟', () => {
    liveSend(SBF_PREFIX + '/records/' + encodeURIComponent(id), {method: 'DELETE'}, 'پرونده حذف شد')
      .then(ok => ok && liveReload());
  });
}

function sbfAct(id, action, labelFa) {
  liveSend(SBF_PREFIX + '/records/' + encodeURIComponent(id) + '/action',
    {method: 'POST', body: {action, noteFa: ''}}, '«' + labelFa + '» انجام شد')
    .then(ok => ok && liveReload());
}

LIVE_PAGES.serviceBonus = pageServiceBonusLive;
