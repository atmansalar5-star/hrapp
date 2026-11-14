/* =====================================================================
   مدل جی‌بیست (G20) — شاخص‌ها و طبقه‌بندی کارکنان و شرکت
   =====================================================================

   هیچ قاعده‌ای اینجا نیست: بیست شاخص، وزن چهار خانواده، آستانه‌های طبقه‌بندی و
   محاسبهٔ امتیاز همه از موتور قواعد سمت سرور می‌آید
   (`/api/v1/compensation/g20-options`، `/g20-board` و `/g20-calc`).
   رابط فقط ورودی‌ها را می‌فرستد و نتیجه را نشان می‌دهد.
   ===================================================================== */

/** وضعیت صفحهٔ طبقه‌بندی جی‌بیست. */
const G20_STATE = {query: '', employeeId: '', rows: [], company: null, calc: null};

function pageG20BoardLive() {
  if (!isLive()) return offlineFallback('g20Board');
  liveLoad(async () => {
    const q = G20_STATE.query ? '?q=' + encodeURIComponent(G20_STATE.query) : '';
    const res = await apiFetch('/api/v1/compensation/g20-board' + q);
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    const rows = d.rows || [];
    G20_STATE.rows = rows;
    G20_STATE.company = d.company || {};
    const company = G20_STATE.company;

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      <b>${d.appliedToPayroll ? '✅' : '🧪'} مرز مدل:</b>
      <div class="muted" style="font-size:.78rem">${esc(d.independenceFa || '')}</div>
      <div class="muted" style="font-size:.78rem;margin-top:6px">${esc(d.instructionFa || '')}</div>
    </div>`;

    const stats = `<div class="grid-stats" style="margin-bottom:12px">
      ${statCard({ico: '👥', tint: '#EFF6FF', accent: '#2563EB', val: fa(company.assessed || 0), label: 'کارمند سنجیده‌شده'})}
      ${statCard({ico: '⚪', tint: '#F3F4F6', accent: '#6B7280', val: fa(company.unassessed || 0), label: 'بدون سنجش'})}
      ${statCard({ico: '📊', tint: '#DCFCE7', accent: '#16A34A', val: fa(company.averageFa || 0), label: 'میانگین امتیاز شرکت'})}
      ${statCard({ico: '🏷️', tint: '#EDE9FE', accent: '#7C3AED', val: 'طبقهٔ ' + esc(company.segmentKey || '-'), label: esc(company.segmentFa || '')})}
    </div>`;

    const segRows = (d.segments || []).map(s => `<tr>
      <td class="num">${esc(s.key)}</td>
      <td>${esc(s.titleFa)}</td>
      <td class="num">${esc(s.fromFa)}</td>
      <td class="muted" style="font-size:.74rem">${esc(s.noteFa)}</td></tr>`).join('');
    const distChips = (company.segments || []).map(s =>
      `<span class="chip ${s.countFa && s.countFa !== '۰' ? 'c-ok' : 'c-wait'}" style="margin-inline-end:6px">${esc(s.titleFa)}: <b>${esc(s.countFa)}</b></span>`).join('');
    const segTable = `<div class="card" style="margin-bottom:12px">
      <div class="card-title">🏷️ جدول طبقه‌بندی و توزیع کارکنان</div>
      <table class="tbl"><thead><tr><th>طبقه</th><th>عنوان</th><th>از امتیاز</th><th>تصمیم پیشنهادی</th></tr></thead>
      <tbody>${segRows}</tbody></table>
      <div style="padding:8px 2px;display:flex;gap:6px;flex-wrap:wrap;align-items:center">
        <span class="muted" style="font-size:.78rem">توزیع کارکنان:</span>${distChips}
      </div>
      <div class="muted" style="font-size:.74rem;padding:0 2px 6px">${esc(company.noteFa || '')}</div>
    </div>`;

    const options = rows.map(r =>
      `<option value="${esc(r.employeeId)}" ${r.employeeId === G20_STATE.employeeId ? 'selected' : ''}>${esc(r.employeeId)} — ${esc(r.fullName)}</option>`).join('');
    const calcCard = `<div class="card card-pad" style="margin-bottom:12px">
      <div class="card-title">🧮 محاسبهٔ امتیاز و طبقهٔ یک کارمند</div>
      <div class="muted" style="font-size:.76rem;margin-bottom:8px">چهار امتیاز خانواده را وارد کنید (۰ تا ۱۰۰)؛ امتیاز کل و طبقه در موتور قواعد سمت سرور محاسبه می‌شود.</div>
      <div style="display:flex;gap:8px;flex-wrap:wrap;align-items:flex-end">
        <div class="field" style="min-width:230px"><label>کارمند</label>
          <select id="g20Employee" onchange="g20PickEmployee()">${options}</select></div>
        <div class="field"><label>پرسن</label><input class="inp num ltr" id="g20Person" value="${esc(G20_DEFAULT_SCORES.person)}" style="width:90px"></div>
        <div class="field"><label>پوزیشن</label><input class="inp num ltr" id="g20Position" value="${esc(G20_DEFAULT_SCORES.position)}" style="width:90px"></div>
        <div class="field"><label>پرفورمنس</label><input class="inp num ltr" id="g20Performance" value="${esc(G20_DEFAULT_SCORES.performance)}" style="width:90px"></div>
        <div class="field"><label>پارتنرشیپ</label><input class="inp num ltr" id="g20Partnership" value="${esc(G20_DEFAULT_SCORES.partnership)}" style="width:90px"></div>
        <button class="btn btn-primary btn-sm" onclick="g20CalcRun()">محاسبه</button>
        <button class="btn btn-sm" onclick="g20Print()">🖨️ چاپ طبقه‌بندی</button>
      </div>
      <div id="g20CalcOut" style="margin-top:10px"></div>
    </div>`;

    const search = `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:8px;align-items:center;flex-wrap:wrap">
      <input class="inp w-lg" id="g20Search" value="${esc(G20_STATE.query)}" placeholder="🔍 جستجوی کارمند (نام یا کد پرسنلی)">
      <button class="btn btn-primary btn-sm" onclick="g20Search()">جستجو</button>
      <button class="btn btn-ghost btn-sm" onclick="g20SearchClear()">✖ پاک‌کردن</button>
    </div>`;

    const bodyRows = rows.map(r => `<tr>
      <td class="num ltr">${esc(r.employeeId)}</td>
      <td>${esc(r.fullName)}<div class="muted" style="font-size:.68rem">${esc(r.departmentName || '')}</div></td>
      <td>${r.hasAssessment ? esc(r.period || '—') : '—'}</td>
      <td class="num">${r.hasAssessment ? esc(r.personFa) : '—'}</td>
      <td class="num">${r.hasAssessment ? esc(r.positionFa) : '—'}</td>
      <td class="num">${r.hasAssessment ? esc(r.performanceFa) : '—'}</td>
      <td class="num">${r.hasAssessment ? esc(r.partnershipFa) : '—'}</td>
      <td class="num"><b>${r.hasAssessment ? esc(r.totalFa) : '—'}</b></td>
      <td>${r.hasAssessment ? pChip('c-ok', 'طبقهٔ ' + r.segmentKey + ' — ' + r.segmentFa) : pChip('c-wait', r.segmentFa)}</td>
      <td class="muted" style="font-size:.74rem">${esc(r.noteFa || '')}</td>
    </tr>`);

    return note + stats + segTable + calcCard + search +
      tbl(['کد', 'کارمند', 'دوره', 'پرسن', 'پوزیشن', 'پرفورمنس', 'پارتنرشیپ', 'امتیاز کل', 'طبقه', 'تصمیم پیشنهادی'],
        bodyRows, {count: rows.length});
  });
  return liveShell('طبقه‌بندی جی‌بیست کارکنان و شرکت', 'بیست شاخص در چهار خانوادهٔ پرسن، پوزیشن، پرفورمنس و پارتنرشیپ — نتیجه فقط برای تحلیل و طبقه‌بندی');
}

/** امتیازهای پیش‌فرض فرم محاسبه (نمونهٔ متعارف سازمان). */
const G20_DEFAULT_SCORES = {person: '75', position: '75', performance: '75', partnership: '75'};

function g20Search() {
  const el = document.getElementById('g20Search');
  G20_STATE.query = el ? el.value.trim() : '';
  liveReload();
}

function g20SearchClear() {
  G20_STATE.query = '';
  liveReload();
}

/** پر کردن چهار امتیاز از آخرین سنجش کارمند انتخاب‌شده. */
async function g20PickEmployee() {
  const sel = document.getElementById('g20Employee');
  const employeeId = sel ? sel.value : '';
  G20_STATE.employeeId = employeeId;
  const row = (G20_STATE.rows || []).find(r => r.employeeId === employeeId);
  if (row && row.hasAssessment) {
    setVal('g20Person', row.personFa);
    setVal('g20Position', row.positionFa);
    setVal('g20Performance', row.performanceFa);
    setVal('g20Partnership', row.partnershipFa);
    g20CalcRun();
  }
}

function setVal(id, v) {
  const el = document.getElementById(id);
  if (el) el.value = v;
}

/** خواندن یک ورودی عددی فرم. */
function g20Num(id) {
  const el = document.getElementById(id);
  const v = el ? parseFloat(el.value) : 0;
  return isFinite(v) ? v : 0;
}

/** محاسبهٔ امتیاز کل و طبقه برای کارمند فرم. */
async function g20CalcRun() {
  const sel = document.getElementById('g20Employee');
  const employeeId = sel ? sel.value : '';
  G20_STATE.employeeId = employeeId;
  const res = await apiFetch('/api/v1/compensation/g20-calc', {
    method: 'POST',
    body: {
      employeeId: employeeId,
      person: g20Num('g20Person'),
      position: g20Num('g20Position'),
      performance: g20Num('g20Performance'),
      partnership: g20Num('g20Partnership'),
    },
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  const d = res.data || {};
  const r = d.result || {};
  G20_STATE.calc = r;
  const rows = (r.contributions || []).map(c => `<tr>
    <td>${esc(c.titleFa)}</td>
    <td class="num">${esc(c.score)}</td>
    <td class="num">${esc(c.weight)}٪</td>
    <td class="num"><b>${esc(c.shareFa)}</b></td></tr>`).join('');
  const out = document.getElementById('g20CalcOut');
  if (out) {
    out.innerHTML = `<div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin-bottom:8px">
        <span class="chip c-ok" style="font-size:.86rem">امتیاز کل: <b>${esc(r.totalFa)}</b></span>
        <span class="chip c-ok" style="font-size:.86rem">طبقه: <b>${esc(r.segmentKey)} — ${esc(r.segmentFa)}</b></span>
        <span class="muted" style="font-size:.76rem">${esc(r.noteFa || '')}</span>
      </div>
      <table class="tbl"><thead><tr><th>خانواده</th><th>امتیاز</th><th>وزن</th><th>سهم در امتیاز کل</th></tr></thead>
      <tbody>${rows}</tbody></table>`;
  }
}

/** چاپ برگهٔ طبقه‌بندی با سرصفحهٔ شرکت و فونت فارسی. */
function g20Print() {
  const rows = G20_STATE.rows || [];
  const company = G20_STATE.company || {};
  const brand = (window.activeOrg || {}).titleFa || '';
  const body = rows.filter(r => r.hasAssessment).map(r => `<tr>
      <td class="num ltr">${esc(r.employeeId)}</td><td>${esc(r.fullName)}</td>
      <td>${esc(r.departmentName || '')}</td><td class="num">${esc(r.period || '')}</td>
      <td class="num">${esc(r.totalFa)}</td><td>${esc(r.segmentKey)} — ${esc(r.segmentFa)}</td>
    </tr>`).join('');
  const html = `<!DOCTYPE html><html lang="fa" dir="rtl"><head><meta charset="utf-8">
<title>طبقه‌بندی جی‌بیست — ${esc(brand)}</title>
<style>
  ${printFontFacesSafe()}
  @page{size:A4 portrait;margin:12mm}
  body{font-family:Vazirmatn,Vazirmatn-fallback,Tahoma,sans-serif;color:#1F2937;margin:0}
  h1{font-size:1rem;margin:0 0 4px}
  .sub{font-size:.76rem;color:#6B7280;margin-bottom:10px}
  table{width:100%;border-collapse:collapse;font-size:.76rem}
  th,td{border:1px solid #E5E7EB;padding:5px 7px;text-align:start}
  th{background:#F3F4F6}
  .note{margin-top:10px;font-size:.74rem;color:#6B7280;border-top:1px dashed #E5E7EB;padding-top:6px}
</style></head><body>
<h1>طبقه‌بندی جی‌بیست (G20) — کارکنان</h1>
<div class="sub">${esc(brand)} — میانگین شرکت: ${esc(company.averageFa || '—')} · طبقهٔ شرکت: ${esc(company.segmentKey || '-')} ${esc(company.segmentFa || '')}</div>
<table><thead><tr><th>کد</th><th>کارمند</th><th>واحد</th><th>دوره</th><th>امتیاز کل</th><th>طبقه</th></tr></thead>
<tbody>${body}</tbody></table>
<div class="note">بیست شاخص در چهار خانوادهٔ پرسن، پوزیشن، پرفورمنس و پارتنرشیپ — محاسبه در موتور قواعد سامانه انجام شده است.</div>
<script>window.addEventListener('load',()=>setTimeout(()=>window.print(),350));<\/script>
</body></html>`;
  const url = URL.createObjectURL(new Blob([html], {type: 'text/html;charset=utf-8'}));
  window.open(url, '_blank');
  setTimeout(() => URL.revokeObjectURL(url), 60000);
}

LIVE_PAGES.g20Board = pageG20BoardLive;