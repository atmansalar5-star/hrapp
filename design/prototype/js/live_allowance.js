/* =====================================================================
   تخصیص فوق العاده‌های حقوق — کد رسمی برای یک کارمند یا گروهی از کارکنان
   ---------------------------------------------------------------------
   هیچ قاعده‌ای اینجا نیست: جدول رسمی کدها، روش‌های محاسبه، مبناها، دامنه‌ها و کارکنان از
   سرور می‌آید (`/api/v1/payroll/allowance-defaults`) و محاسبه/تطبیق دامنه هم در سرور
   انجام می‌شود (`/api/v1/payroll/allowance-preview`). این صفحه فقط:

      ۱) فرم تخصیص را می‌سازد: کد ⟵ عنوان و نوع قلم خودکار، روش محاسبه (ماهانه ثابت،
         یک‌بار مرتبه، درصدی، ضریبی، فرمولی)، مقدار/درصد/ضریب یا فرمول، مشمولیت مالیات و
         بیمه، دامنه (کارمند/واحد/نوع استخدام/گروه شغلی/گروه نظام پرداخت/همه) و ماه اعتبار.
      ۲) تخصیص‌های ثبت‌شده را فهرست و فعال/پایان می‌کند (رکورد ماژول حقوق).
      ۳) پیش‌نمایش می‌دهد: برای یک کارمند، سطرهای تخصیص‌یافته با کد، مبلغ و مشمولیت کسور.
   ===================================================================== */

var ASG_DEFAULTS = null;
var ASG_ROWS = [];

/** فرم تخصیص تازه. */
function asgFormHtml() {
  const d = ASG_DEFAULTS || {};
  const opts = (list, selected, label) => (list || []).map(x =>
    `<option value="${esc(x.key || x.code)}" ${(x.key || x.code) === selected ? 'selected' : ''}>${esc(label ? label(x) : (x.titleFa || x.code))}</option>`).join('');
  const codeOpts = (d.codes || []).map(c =>
    `<option value="${esc(c.code)}">${esc(c.code)} — ${esc(c.titleFa)} (${esc(c.kind === 'earning' ? 'بستانکار' : c.kind === 'deduction' ? 'کسور' : 'وام')})</option>`).join('');
  const empOpts = (d.employees || []).map(e => `<option value="${esc(e.id)}">${esc(e.titleFa)}</option>`).join('');
  const unitOpts = [...new Set((d.employees || []).map(e => e.unit).filter(Boolean))]
    .map(u => `<option value="${esc(u)}">${esc(u)}</option>`).join('');
  const typeOpts = [...new Set((d.employees || []).map(e => e.employmentType).filter(Boolean))]
    .map(t => `<option value="${esc(t)}">${esc(t)}</option>`).join('');
  const gradeOpts = [...new Set((d.employees || []).map(e => e.jobGrade).filter(Boolean))]
    .map(g => `<option value="${esc(g)}">${esc(g)}</option>`).join('');
  const groupOpts = [...new Set((d.employees || []).map(e => e.payGroup).filter(Boolean))]
    .map(g => `<option value="${esc(g)}">گروه ${fa(g)}</option>`).join('');

  const field = (label, inner, hint) => `<div class="field"><label>${label}</label>${inner}${hint ? `<span class="muted" style="font-size:.7rem">${esc(hint)}</span>` : ''}</div>`;

  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="font-weight:600;margin-bottom:8px">تخصیص تازه</div>
      <div class="form-grid">
        ${field('کد فوق العاده <span class="req">*</span>', `<select id="asgCode">${codeOpts}</select>`, 'جدول رسمی سازمان؛ عنوان و نوع قلم خودکار پر می‌شود.')}
        ${field('عنوان قلم', `<input id="asgTitle" placeholder="خالی = عنوان رسمی همان کد">`)}
        ${field('نوع قلم', `<select id="asgCategory">
            <option value="earning">مزایا (بستانکار)</option>
            <option value="deduction">کسور (بدهکار)</option>
          </select>`)}
        ${field('روش محاسبه <span class="req">*</span>', `<select id="asgCalc">${opts(d.calcKinds, 'fixed')}</select>`)}
        ${field('مقدار / درصد / ضریب', `<input id="asgRate" class="num" placeholder="مثلاً 9000000 یا 15 یا 1.5">`)}
        ${field('مبنا (برای درصدی و ضریبی)', `<select id="asgBasis"><option value="">—</option>${opts(d.bases, '')}</select>`)}
        ${field('فرمول (برای روش فرمولی)', `<input id="asgFormula" class="ltr" placeholder="مثلاً BASE*0.15+CHILDREN*1000000">`, 'متغیرها: BASE، ALLOWANCEBASE، ATTRACTIONBASE، GROSS و پارامترها.')}
        ${field('مشمول مالیات', `<select id="asgTaxable"><option value="خیر">خیر</option><option value="بله">بله</option></select>`)}
        ${field('مشمول بیمه/بازنشستگی', `<select id="asgInsured"><option value="خیر">خیر</option><option value="بله">بله</option></select>`)}
        ${field('دامنهٔ تخصیص <span class="req">*</span>', `<select id="asgTargetKind">${opts(d.targetKinds, 'employee')}</select>`)}
        ${field('مقدار دامنه', `<input id="asgTargetValue" list="asgTargetList" placeholder="شناسهٔ کارمند، واحد، نوع استخدام یا گروه">
            <datalist id="asgTargetList">${empOpts}${unitOpts}${typeOpts}${gradeOpts}${groupOpts}</datalist>`, 'برای دامنهٔ «همهٔ کارکنان» خالی بماند.')}
        ${field('ماه شروع اعتبار <span class="req">*</span>', `<input id="asgFrom" placeholder="${esc((d.period || '1405/01'))}">`)}
        ${field('ماه پایان اعتبار', `<input id="asgTo" placeholder="خالی = تا اطلاع">`)}
        ${field('یادداشت / شمارهٔ مصوبه', `<input id="asgNote">`)}
      </div>
      <div style="display:flex;gap:8px;align-items:center;margin-top:8px;flex-wrap:wrap">
        <button class="btn btn-primary" id="asgSave">ثبت تخصیص</button>
        <button class="btn btn-ghost" id="asgReset">پاک‌کردن فرم</button>
        <span class="muted" style="font-size:.74rem">${esc((ASG_DEFAULTS || {}).noteFa || '')}</span>
      </div>
    </div>`;
}

/** فهرست تخصیص‌های ثبت‌شده. */
function asgListHtml() {
  if (!ASG_ROWS.length) return `<div class="card card-pad muted">هنوز تخصیصی ثبت نشده است.</div>`;
  const rows = ASG_ROWS.map(r => {
    const f = r.fields || {};
    const target = f.targetKind === 'all' ? 'همهٔ کارکنان'
      : `${esc(f.targetKind || '')}: ${esc(f.targetValue || '')}`;
    const value = f.calc === 'formula' ? esc(f.formula || '')
      : `${esc(f.rate || '')}${f.calc === 'percent' ? '٪' : ''}${f.basis ? ' × ' + esc(f.basis) : ''}`;
    const flags = `${f.taxable === 'بله' ? 'مالیات' : '—'} / ${f.insured === 'بله' ? 'بیمه' : '—'}`;
    const actions = r.status === 'active'
      ? `<button class="btn btn-sm btn-ghost" data-asg-act="close" data-asg-id="${esc(r.id)}">پایان</button>`
      : `<button class="btn btn-sm btn-primary" data-asg-act="activate" data-asg-id="${esc(r.id)}">فعال‌سازی</button>`;
    return `<tr>
      <td class="num ltr">${esc(r.id)}</td>
      <td>${esc(f.code || '')} — ${esc(f.titleFa || r.title || '')}</td>
      <td>${esc(f.calc || '')}${f.calc === 'percent' || f.calc === 'coefficient' ? ` (${esc((ASG_DEFAULTS.calcKinds || []).find(c => c.key === f.calc) ? '' : '')})` : ''}</td>
      <td>${target}</td>
      <td class="num">${value}</td>
      <td class="muted" style="font-size:.72rem">${flags}</td>
      <td>${esc(f.effectiveFrom || '')}${f.effectiveTo ? ' تا ' + esc(f.effectiveTo) : ''}</td>
      <td>${pChip(r.status === 'active' ? 'c-ok' : 'c-wait', r.statusFa || r.status)}</td>
      <td><div class="row-actions">${actions}</div></td>
    </tr>`;
  });
  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="font-weight:600;margin-bottom:6px">تخصیص‌های ثبت‌شده (${fa(ASG_ROWS.length)})</div>
      ${tbl(['شماره', 'کد و عنوان', 'روش', 'دامنه', 'مقدار', 'مشمول', 'اعتبار', 'وضعیت', 'عملیات'], rows, {count: ASG_ROWS.length})}
    </div>`;
}

/** کارت پیش‌نمایش تخصیص‌های یک کارمند. */
function asgPreviewHtml() {
  const empOpts = (ASG_DEFAULTS.employees || []).map(e =>
    `<option value="${esc(e.id)}">${esc(e.titleFa)} — ${esc(e.payModelFa || '')}</option>`).join('');
  return `<div class="card card-pad">
      <div style="font-weight:600;margin-bottom:6px">پیش‌نمایش تخصیص‌ها</div>
      <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
        <select id="asgPreviewEmployee" style="min-width:320px"><option value="">— انتخاب کارمند —</option>${empOpts}</select>
        <button class="btn btn-primary" id="asgPreviewRun">نمایش اقلام</button>
        <span class="muted" style="font-size:.72rem">فقط تخصیص‌های فعال و در پنجرهٔ اعتبار همین ماه می‌آید.</span>
      </div>
      <div id="asgPreviewOut" style="margin-top:10px"></div>
    </div>`;
}

/** صفحهٔ «تخصیص فوق العاده‌های حقوق». */
function pageAllowanceAssignLive() {
  if (!isLive()) return offlineFallback('allowanceAssign');
  (async () => {
    await liveLoad(async () => {
      const [defRes, listRes] = await Promise.all([
        apiFetch('/api/v1/payroll/allowance-defaults'),
        apiFetch('/api/v1/payroll/records?kind=allowanceAssignment&size=200'),
      ]);
      if (!defRes.ok) return liveErrorBox(defRes.message);
      ASG_DEFAULTS = defRes.data || {};
      ASG_ROWS = (listRes.ok && listRes.data && listRes.data.items) || [];
      return asgFormHtml() + asgListHtml() + asgPreviewHtml();
    });
    asgBind();
  })();
  return liveShell('تخصیص فوق العاده‌های حقوق',
    'کد رسمی فوق العاده را به یک کارمند یا گروهی از کارکنان تخصیص بدهید: مبلغ ثابت، یک‌بار مرتبه، درصدی، ضریبی یا فرمولی — با تعیین مشمولیت مالیات و بیمه', '');
}

/** اتصال رویدادها (بدون هندلر درون‌خطی). */
function asgBind() {
  const code = document.getElementById('asgCode');
  if (code && !code.dataset.asgBound) {
    code.dataset.asgBound = '1';
    // کد ⟵ عنوان و نوع قلم خودکار از جدول رسمی.
    code.addEventListener('change', () => {
      const item = (ASG_DEFAULTS.codes || []).find(c => c.code === code.value);
      if (!item) return;
      const title = document.getElementById('asgTitle');
      if (title && !title.value) title.value = item.titleFa;
      const category = document.getElementById('asgCategory');
      if (category) category.value = item.kind === 'earning' ? 'earning' : 'deduction';
    });
  }
  const save = document.getElementById('asgSave');
  if (save && !save.dataset.asgBound) {
    save.dataset.asgBound = '1';
    save.addEventListener('click', () => asgSave());
  }
  const reset = document.getElementById('asgReset');
  if (reset && !reset.dataset.asgBound) {
    reset.dataset.asgBound = '1';
    reset.addEventListener('click', () => { ASG_DEFAULTS = ASG_DEFAULTS; liveReload(); });
  }
  const run = document.getElementById('asgPreviewRun');
  if (run && !run.dataset.asgBound) {
    run.dataset.asgBound = '1';
    run.addEventListener('click', () => asgPreview());
  }
  // عملیات سطرها (فعال‌سازی/پایان) با شنوندهٔ واگذارشده روی ناحیهٔ صفحه.
  const region = document.getElementById('liveRegion');
  if (region && !region.dataset.asgBound) {
    region.dataset.asgBound = '1';
    region.addEventListener('click', ev => {
      const btn = ev.target.closest('[data-asg-act]');
      if (!btn) return;
      asgAction(btn.dataset.asgId, btn.dataset.asgAct);
    });
  }
}

/** نشاندن HTML در یک ظرف؛ تنها نقطهٔ ورود HTML این صفحه (سقف امنیتی رابط). */
function asgPaint(el, html) { if (el) el.innerHTML = html; }

/** مقدار یک ورودی فرم. */
function asgVal(id) {
  const el = document.getElementById(id);
  return el ? (el.value || '').trim() : '';
}

/** ثبت تخصیص تازه. */
async function asgSave() {
  const fields = {
    code: asgVal('asgCode'), titleFa: asgVal('asgTitle'), category: asgVal('asgCategory'),
    calc: asgVal('asgCalc'), rate: asgVal('asgRate'), basis: asgVal('asgBasis'),
    formula: asgVal('asgFormula'), taxable: asgVal('asgTaxable'), insured: asgVal('asgInsured'),
    targetKind: asgVal('asgTargetKind'), targetValue: asgVal('asgTargetValue'),
    effectiveFrom: asgVal('asgFrom'), effectiveTo: asgVal('asgTo'), noteFa: asgVal('asgNote'),
  };
  const codeItem = (ASG_DEFAULTS.codes || []).find(c => c.code === fields.code);
  const title = fields.titleFa || (codeItem ? codeItem.titleFa : 'تخصیص فوق العاده');
  const res = await apiFetch('/api/v1/payroll/records', {
    method: 'POST', body: {kind: 'allowanceAssignment', title, fields},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  // تخصیص تازه پیش‌نویس است؛ برای اثرگذاری باید فعال شود.
  const created = res.data || {};
  if (created.id) {
    await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(created.id) + '/action', {
      method: 'POST', body: {action: 'activate'},
    });
  }
  toast('تخصیص ثبت و فعال شد');
  liveReload();
}

/** فعال‌سازی یا پایان یک تخصیص. */
async function asgAction(id, action) {
  if (!id || !action) return;
  const res = await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(id) + '/action', {
    method: 'POST', body: {action},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast(action === 'activate' ? 'تخصیص فعال شد' : 'تخصیص پایان یافت');
  liveReload();
}

/** پیش‌نمایش اقلام یک کارمند. */
async function asgPreview() {
  const out = document.getElementById('asgPreviewOut');
  const employeeId = asgVal('asgPreviewEmployee');
  if (!out) return;
  if (!employeeId) { out.textContent = 'کارمند را انتخاب کنید.'; return; }
  asgPaint(out, liveSpinner());
  const res = await apiFetch('/api/v1/payroll/allowance-preview', {method: 'POST', body: {employeeId}});
  if (!res.ok) { asgPaint(out, liveErrorBox(res.message)); return; }
  const d = res.data || {};
  const lines = d.lines || [];
  const totals = d.totals || {};
  const cards = `<div class="grid-stats" style="margin-bottom:8px">
      ${statCard({ico: '➕', tint: '#DCFCE7', accent: '#16A34A', val: money(totals.earnings || 0), label: 'جمع مزایا'})}
      ${statCard({ico: '➖', tint: '#FEE2E2', accent: '#DC2626', val: money(totals.deductions || 0), label: 'جمع کسور'})}
      ${statCard({ico: '🧾', tint: '#FEF3C7', accent: '#D97706', val: money(totals.taxableEarnings || 0), label: 'مشمول مالیات'})}
      ${statCard({ico: '🛡️', tint: '#EDE9FE', accent: '#7C3AED', val: money(totals.insuredEarnings || 0), label: 'مشمول بیمه'})}
    </div>`;
  if (!lines.length) {
    asgPaint(out, cards + `<div class="card card-pad muted">برای این کارمند در این ماه تخصیص فعالی نیست.</div>`);
    return;
  }
  const rows = lines.map(l => `<tr>
      <td class="num ltr">${esc(l.code)}</td>
      <td>${esc(l.titleFa)}</td>
      <td>${esc(l.categoryFa || l.category)}</td>
      <td>${esc(l.calcFa || l.calc)}</td>
      <td class="num">${money(l.amount)}</td>
      <td>${l.taxable ? '✅' : '—'}</td>
      <td>${l.insured ? '✅' : '—'}</td>
      <td class="muted" style="font-size:.72rem">${esc(l.detailFa || '')}</td>
    </tr>`);
  asgPaint(out, cards + tbl(['کد', 'عنوان', 'نوع', 'روش محاسبه', 'مبلغ (ریال)', 'مالیات', 'بیمه', 'شرح'],
    rows, {count: lines.length}));
}

if (typeof LIVE_PAGES !== 'undefined') LIVE_PAGES.allowanceAssign = pageAllowanceAssignLive;
