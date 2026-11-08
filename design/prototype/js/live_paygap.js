/* =====================================================================
   تطبیق حقوق و «تفاوت تطبیق»
   ---------------------------------------------------------------------
   هنگام اجرای طرح تازهٔ حقوق (طبقه‌بندی مشاغل یا بازنگری حقوق) ممکن است حقوق کارمند
   کمتر شود. این صفحه همان محاسبه را نشان می‌دهد: جمع پیشین در برابر جمع طرح تازه،
   مبلغ تفاوت تطبیق، و جذب آن با افزایش‌های بعدی.

   هیچ قاعده‌ای اینجا نیست: اعداد از `/paygap-plan` و `/paygap-absorb` می‌آید و
   سرور با پارامترهای شرکت محاسبه می‌کند.
   ===================================================================== */

window.PAYGAP = window.PAYGAP || {employeeId: '', lastResult: null};

/** پارامترها، دستورالعمل و کارکنان فرم تطبیق. */
async function payGapDefaultsCached() {
  if (window.PAYGAP_DEF_CACHE) return window.PAYGAP_DEF_CACHE;
  const res = await apiFetch('/api/v1/payroll/paygap-defaults');
  window.PAYGAP_DEF_CACHE = res.ok ? (res.data || {}) : {};
  return window.PAYGAP_DEF_CACHE;
}

function pagePayGapLive() {
  if (!isLive()) return offlineFallback('payGap');
  liveLoad(async () => {
    const defs = await payGapDefaultsCached();
    const employees = defs.employees || [];
    const params = defs.params || {};
    const empOpts = employees.map(e => `<option value="${esc(e.id)}" ${e.id === window.PAYGAP.employeeId ? 'selected' : ''}>${esc(e.titleFa)}</option>`).join('');

    const form = `<div class="card card-pad" style="margin-bottom:12px">
      <div class="card-title">🧾 ورودی تطبیق</div>
      <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(230px,1fr));gap:12px">
        <div class="field"><label>کارمند</label>
          <select id="pgEmp" onchange="payGapPickEmployee(this.value)">
            <option value="">— انتخاب کنید —</option>${empOpts}</select></div>
        <div class="field"><label>جمع حقوق و مزایای پیشین (ریال)</label>
          <input class="inp num" id="pgPrevious" type="number" min="0" placeholder="۱۲۳۴۵۶۷۸۹"></div>
        <div class="field"><label>جمع حقوق و مزایا بر پایهٔ طرح تازه (ریال)</label>
          <input class="inp num" id="pgNew" type="number" min="0" placeholder="۱۰۰۰۰۰۰۰۰"></div>
        <div class="field"><label>افزایش‌های آینده برای محاسبهٔ جذب (ریال، با ویرگول)</label>
          <input class="inp" id="pgIncreases" placeholder="مثال: 2000000, 3000000"></div>
      </div>
      <div class="row-actions" style="margin-top:12px">
        <button class="btn btn-primary" onclick="payGapPlan()">🧮 محاسبهٔ تفاوت تطبیق</button>
        <button class="btn btn-ghost" onclick="payGapAbsorb()">📉 محاسبهٔ جذب</button>
      </div>
      <div class="muted" style="font-size:.74rem;margin-top:8px">هر دو جمع باید از یک مبنای محاسبه باشند: «مزد مبنا + پرداخت‌های ثابت دوره».</div>
    </div>`;

    return form + '<div id="pgBox">' + (window.PAYGAP.lastResult ? payGapResultHTML(window.PAYGAP.lastResult) : '') + '</div>' +
      payGapInstructionHTML(defs) +
      payGapParamsHTML(params, defs.paramViews || []) +
      statNote(defs.noteFa || '');
  });
  return liveShell('تطبیق حقوق و تفاوت تطبیق', 'سنجش کاهش حقوق در زمان اجرای طرح تازه، محاسبهٔ تفاوت تطبیق و جذب آن');
}

/** انتخاب کارمند: سند کارگزینی جاری بارگذاری می‌شود (مزد مبنا و تفاوت تطبیق کنونی). */
async function payGapPickEmployee(id) {
  window.PAYGAP.employeeId = id || '';
  const box = document.getElementById('pgBox');
  if (!id) {
    if (box) box.innerHTML = '';
    return;
  }
  if (box) box.innerHTML = '<div class="card card-pad muted">⏳ در حال خواندن سند کارگزینی…</div>';
  const res = await apiFetch('/api/v1/payroll/paygap-employee?id=' + encodeURIComponent(id));
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    return;
  }
  const d = res.data || {};
  if (box) box.innerHTML = payGapDocumentHTML(d);
}

/** کارت سند کارگزینی جاری کارمند. */
function payGapDocumentHTML(d) {
  const parts = (d.wageParts || []).map(p => `<tr><td>${esc(p.titleFa)}</td><td>${esc(p.kindFa)}</td><td class="num">${money(p.amount)}</td></tr>`).join('');
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📄 سند کارگزینی جاری — ${esc(d.sourceFa || '')}</div>
    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:10px;margin-bottom:10px">
      <div><span class="muted">پایهٔ حقوق:</span> <b class="num">${money(d.baseSalary || 0)}</b></div>
      <div><span class="muted">مزد مبنا:</span> <b class="num">${money(d.mazdBase || 0)}</b></div>
      <div><span class="muted">تفاوت تطبیق کنونی:</span> <b class="num">${d.hasPayGap ? money(d.payGap) : '— ندارد'}</b></div>
      <div><span class="muted">از تاریخ:</span> <b class="num">${esc(d.payGapSince || '—')}</b></div>
    </div>
    ${parts ? liveTable(['جزء مزد', 'نوع', 'مبلغ (ریال)'], [parts], (d.wageParts || []).length) : ''}
    <div class="muted" style="font-size:.76rem;margin-top:8px">${esc(d.breakdownFa || '')}</div>
  </div>`;
}

/** محاسبهٔ تفاوت تطبیق. */
async function payGapPlan() {
  const val = id => {
    const el = document.getElementById(id);
    return el ? el.value.trim() : '';
  };
  const body = {
    employeeId: window.PAYGAP.employeeId || '',
    previousTotal: Number(val('pgPrevious') || 0) || 0,
    newTotal: Number(val('pgNew') || 0) || 0,
  };
  if (body.previousTotal <= 0 || body.newTotal <= 0) {
    toast('هر دو جمع حقوق و مزایا را وارد کنید', 'error');
    return;
  }
  const box = document.getElementById('pgBox');
  if (box) box.innerHTML = '<div class="card card-pad muted">⏳ در حال محاسبه…</div>';
  const res = await apiFetch('/api/v1/payroll/paygap-plan', {method: 'POST', body});
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    toast(res.message, 'error');
    return;
  }
  window.PAYGAP.lastResult = res.data || {};
  if (box) box.innerHTML = payGapResultHTML(window.PAYGAP.lastResult) + (res.data.document ? payGapDocumentHTML(res.data.document) : '');
  toast('فرم تطبیق ساخته شد');
}

/** خروجی فرم محاسباتی تطبیق. */
function payGapResultHTML(payload) {
  const r = payload.result || {};
  const rows = (r.lines || []).map(l => `<tr>
    <td class="num">${fa(l.order)}</td>
    <td><b>${esc(l.titleFa)}</b></td>
    <td class="num">${esc(l.amountFa)}</td>
    <td style="font-size:.78rem;line-height:1.9">${esc(l.detailFa)}</td></tr>`).join('');
  const chip = r.covered
    ? '<span class="chip" style="color:#B45309;background:#FEF3C7">مشمول تفاوت تطبیق</span>'
    : '<span class="chip c-ok">مشمول نیست</span>';
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📋 نتیجهٔ تطبیق</div>
    <div style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;margin-bottom:8px">
      ${chip}
      <span class="muted" style="font-size:.82rem">مبلغ تفاوت تطبیق: <b class="num">${money(r.payGap || 0)}</b> ریال</span>
      <span class="muted" style="font-size:.82rem">مبلغی که در حکم ثبت می‌شود را همین‌جا می‌توانید برای محاسبهٔ جذب ویرایش کنید:</span>
      <input class="inp num" id="pgGapForAbsorb" type="number" min="0" value="${esc(String(r.payGap || 0))}" style="max-width:180px">
    </div>
    <div style="font-size:.88rem;line-height:2">${esc(r.summaryFa || '')}</div>
  </div>
  <div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧮 فرم محاسباتی خط‌به‌خط</div>
    ${liveTable(['گام', 'موضوع', 'مبلغ / مقدار', 'شرح'], rows ? [rows] : [], (r.lines || []).length)}
  </div>`;
}

/** محاسبهٔ جذب تفاوت تطبیق با افزایش‌های واردشده. */
async function payGapAbsorb() {
  const r = (window.PAYGAP.lastResult || {}).result || {};
  const gapEl = document.getElementById('pgGapForAbsorb');
  const gap = gapEl ? Number(gapEl.value) || 0 : Number(r.payGap || 0);
  const incEl = document.getElementById('pgIncreases');
  const increases = (incEl ? incEl.value : '').split(/[،,]/).map(s => Number(s.trim())).filter(n => n > 0);
  const box = document.getElementById('pgBox');
  if (box) box.innerHTML = '<div class="card card-pad muted">⏳ در حال محاسبهٔ جذب…</div>';
  const res = await apiFetch('/api/v1/payroll/paygap-absorb', {method: 'POST', body: {payGap: gap, increases}});
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    return;
  }
  const d = res.data || {};
  const rows = (d.rows || []).map(s => `<tr>
    <td class="num">${esc(s.order)}</td>
    <td class="num">${esc(s.increaseFa)}</td>
    <td class="num">${esc(s.reductionFa)}</td>
    <td class="num"><b>${esc(s.remainingFa)}</b></td>
    <td style="font-size:.78rem">${esc(s.detailFa)}</td></tr>`).join('');
  if (box) box.innerHTML = `<div class="card card-pad" style="margin-bottom:12px">
      <div class="card-title">📉 جذب تفاوت تطبیق</div>
      <div style="font-size:.88rem;line-height:2;margin-bottom:8px">${esc(d.summaryFa || '')}</div>
      ${liveTable(['مرحله', 'افزایش (ریال)', 'کاهش تفاوت تطبیق', 'باقی‌مانده', 'شرح'], rows ? [rows] : [], (d.rows || []).length)}
      <div class="muted" style="font-size:.76rem;margin-top:8px">${esc(d.noteFa || '')}</div>
    </div>`;
}

/** دستورالعمل تفاوت تطبیق. */
function payGapInstructionHTML(defs) {
  const steps = (defs && defs.instruction) || [];
  if (!steps.length) return '';
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📘 دستورالعمل تطبیق حقوق و تفاوت تطبیق</div>
    <ol style="margin:0;padding-inline-start:20px;line-height:2.1;font-size:.84rem">
      ${steps.map(s => `<li>${esc(s)}</li>`).join('')}
    </ol>
  </div>`;
}

/** جدول پارامترهای به‌کاررفتهٔ تطبیق. */
function payGapParamsHTML(params, views) {
  const row = (title, value) => `<tr><th style="text-align:right;background:#F8FAFC">${esc(title)}</th><td class="num">${esc(value)}</td></tr>`;
  const custom = (views || []).map(v => row(v.titleFa || v.key, (v.value || '') + ' ' + (v.unitFa || ''))).join('');
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">⚙️ پارامترهای به‌کاررفته (از تنظیمات شرکت)</div>
    <table class="tbl" style="width:100%">
      ${row('گرد کردن مبلغ تفاوت تطبیق', money(params.roundTo || 0))}
      ${row('نسبت جذب با هر افزایش', fa(Math.round((params.absorptionRatio || 0) * 100)) + '٪')}
      ${row('جزو مزد مبنا', params.inWageBase ? 'بله' : 'خیر')}
      ${row('مشمول بیمه', params.insurable ? 'بله' : 'خیر')}
      ${row('مشمول مالیات', params.taxable ? 'بله' : 'خیر')}
      ${custom}
    </table>
  </div>`;
}

LIVE_PAGES.payGap = pagePayGapLive;