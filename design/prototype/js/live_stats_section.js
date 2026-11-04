/* =====================================================================
   سکشن آمار — برش‌های تحلیلی جمعیت کارکنان
   ---------------------------------------------------------------------
   چرا این صفحه: کارفرما برای «آمار» فقط یک فهرست نمی‌خواهد، بلکه برش‌های ترکیبی با
   شکل‌های مختلف نمایش می‌خواهد: جدول توافقی، هیت‌مپ، میله‌ای انباشته، ستونی، دونات و
   خروجی CSV. همهٔ اعداد از سرور می‌آید (`/api/v1/statistics/section` و `/crosstab`) و
   این فایل هیچ محاسبهٔ آماری انجام نمی‌دهد؛ فقط انتخاب بُعد، حالت نمایش و شکل‌دادن.

   بُعدها (جنسیت، گروه سنی، سابقه، تحصیلات، واحد، نوع استخدام، وضعیت) از سرور می‌آید،
   پس افزودن بُعد تازه در موتور آمار، همین‌جا خودکار دیده می‌شود.
   ===================================================================== */

var STAT_SECTION = null;   // پاسخ کامل سکشن (شاخص‌ها، توزیع‌ها، برش‌های آماده)
var STAT_DIMS = null;      // فهرست بُعدها برای فهرست‌های انتخابی
var STAT_ROWS = 'age';     // بُعد سطر در برش دلخواه
var STAT_COLS = 'service'; // بُعد ستون در برش دلخواه
var STAT_CUSTOM = null;    // نتیجهٔ برش دلخواه
var STAT_MODE = 'count';   // حالت نمایش اعداد: count | share | row | col

/** رنگ‌های ثابت نمودارها — ترتیب دسته‌ها را نشان می‌دهد. */
var STAT_PALETTE = ['#2563EB', '#16A34A', '#D97706', '#7C3AED', '#DB2777', '#0891B2', '#65A30D', '#DC2626', '#0F766E', '#9333EA'];

function statColor(i) { return STAT_PALETTE[i % STAT_PALETTE.length]; }

/** برچسب حالت‌های نمایش عدد در جدول‌ها. */
var STAT_MODES = [
  { key: 'count', label: 'تعداد نفر', hint: 'شمار کارکنان هر خانه' },
  { key: 'share', label: 'سهم از کل', hint: 'درصد از کل کارکنان' },
  { key: 'row', label: 'درصد سطر', hint: 'سهم هر خانه از سطر خودش' },
  { key: 'col', label: 'درصد ستون', hint: 'سهم هر خانه از ستون خودش' },
];

/** مقدار نمایشی یک خانه در حالت جاری. */
function statModeText(cell, mode) {
  switch (mode) {
    case 'share': return fa(cell.sharePct) + '٪';
    case 'row': return fa(cell.rowPct) + '٪';
    case 'col': return fa(cell.colPct) + '٪';
    default: return fa(cell.count);
  }
}

/** مقدار عددی یک خانه در حالت جاری (برای رنگ هیت‌مپ). */
function statModeValue(cell, mode) {
  switch (mode) {
    case 'share': return cell.sharePct;
    case 'row': return cell.rowPct;
    case 'col': return cell.colPct;
    default: return cell.count;
  }
}

/** رنگ زمینهٔ هیت‌مپ بر پایهٔ شدت مقدار. */
function statHeat(value, max) {
  if (!max || !value) return '';
  const ratio = Math.min(1, value / max);
  return `background:rgba(37,99,235,${(0.06 + ratio * 0.42).toFixed(3)})`;
}

/** بیشینهٔ خانه‌های یک جدول در حالت جاری. */
function statMatrixMax(mx, mode) {
  let max = 0;
  for (const row of mx.rows || []) {
    for (const cell of row.cells || []) max = Math.max(max, statModeValue(cell, mode));
  }
  return max;
}

/* ----------------------------- شکل‌های نمایش ----------------------------- */

/** جدول توافقی با هیت‌مپ، جمع سطر و ستون. */
function statMatrixTable(mx, mode) {
  const max = statMatrixMax(mx, mode);
  const head = `<tr><th>${esc(mx.rowsDim.titleFa)}</th>` +
    (mx.cols || []).map((c, i) => `<th style="text-align:center"><span style="color:${statColor(i)}">●</span> ${esc(c.titleFa)}</th>`).join('') +
    `<th>جمع سطر</th><th>سهم</th></tr>`;
  const body = (mx.rows || []).map((row, r) => `<tr>
    <td><b>${esc(row.titleFa)}</b></td>
    ${(row.cells || []).map(cell => `<td class="num" style="text-align:center;${statHeat(statModeValue(cell, mode), max)}"
      title="سهم کل ${fa(cell.sharePct)}٪ — سطر ${fa(cell.rowPct)}٪ — ستون ${fa(cell.colPct)}٪">${cell.count ? statModeText(cell, mode) : '<span class="muted">—</span>'}</td>`).join('')}
    <td class="num"><b>${fa(row.count)}</b></td>
    <td class="num">${fa(row.sharePct)}٪</td></tr>`).join('');
  const foot = `<tr class="total"><td><b>جمع ستون</b></td>
    ${(mx.colTotals || []).map(v => `<td class="num" style="text-align:center"><b>${fa(v)}</b></td>`).join('')}
    <td class="num"><b>${fa(mx.total)}</b></td><td class="num">۱۰۰٪</td></tr>`;
  return `<div class="tbl-wrap"><table class="tbl" style="width:100%"><thead>${head}</thead><tbody>${body}${foot}</tbody></table></div>`;
}

/** نمودار میله‌ای انباشته: هر سطر یک نوار، هر ستون یک قطعه. */
function statStackedBars(mx) {
  const cols = mx.cols || [];
  if (!cols.length) return '';
  const legend = `<div style="display:flex;flex-wrap:wrap;gap:10px;font-size:.74rem;margin-bottom:8px">
    ${cols.map((c, i) => `<span><span style="color:${statColor(i)}">■</span> ${esc(c.titleFa)}</span>`).join('')}
  </div>`;
  const rows = (mx.rows || []).filter(r => r.count > 0).map(row => {
    const total = row.count || 1;
    const segs = cols.map((c, i) => {
      const value = (row.series || [])[i] || 0;
      if (!value) return '';
      return `<div title="${esc(c.titleFa)}: ${fa(value)} نفر" style="width:${(value / total * 100).toFixed(2)}%;background:${statColor(i)}"></div>`;
    }).join('');
    return `<div style="display:flex;align-items:center;gap:8px;margin-bottom:6px;font-size:.78rem">
      <span style="min-width:130px">${esc(row.titleFa)}</span>
      <div style="flex:1;display:flex;height:16px;border-radius:6px;overflow:hidden;background:var(--border)">${segs}</div>
      <span class="num" style="min-width:60px;text-align:end">${fa(row.count)} نفر</span>
    </div>`;
  }).join('');
  return legend + rows;
}

/** نمودار ستونی برای یک توزیع تک‌بُعدی. */
function statColumnChart(dim) {
  const cats = (dim.categories || []).filter(c => c.count > 0);
  if (!cats.length) return `<div class="muted" style="font-size:.78rem">دادهای برای نمایش نیست.</div>`;
  const max = Math.max(...cats.map(c => c.count)) || 1;
  return `<div style="display:flex;align-items:flex-end;gap:10px;height:150px;padding-top:6px">
    ${cats.map((c, i) => `<div style="flex:1;display:flex;flex-direction:column;align-items:center;justify-content:flex-end;min-width:0">
      <span class="num" style="font-size:.72rem;margin-bottom:4px">${fa(c.count)}</span>
      <div style="width:100%;height:${Math.max(4, c.count / max * 108).toFixed(0)}px;background:${statColor(i)};border-radius:6px 6px 0 0"
        title="${esc(c.titleFa)}: ${fa(c.count)} نفر (${fa(c.sharePct)}٪)"></div>
      <span style="font-size:.66rem;line-height:1.5;text-align:center;margin-top:4px;overflow:hidden">${esc(c.titleFa)}</span>
    </div>`).join('')}
  </div>`;
}

/** نمودار دونات (حلقه‌ای) با راهنمای اطراف. */
function statDonut(dim) {
  const cats = (dim.categories || []).filter(c => c.count > 0);
  const total = cats.reduce((sum, c) => sum + c.count, 0);
  if (!cats.length || !total) return `<div class="muted" style="font-size:.78rem">دادهای برای نمایش نیست.</div>`;
  const R = 52, C = 2 * Math.PI * R;
  let offset = 0;
  const segs = cats.map((c, i) => {
    const frac = c.count / total;
    const seg = `<circle cx="70" cy="70" r="${R}" fill="none" stroke="${statColor(i)}" stroke-width="20"
      stroke-dasharray="${(frac * C).toFixed(2)} ${C.toFixed(2)}" stroke-dashoffset="${(-offset * C).toFixed(2)}"
      transform="rotate(-90 70 70)"><title>${esc(c.titleFa)}: ${fa(c.count)} نفر</title></circle>`;
    offset += frac;
    return seg;
  }).join('');
  const legend = cats.map((c, i) => `<div style="display:flex;align-items:center;gap:6px;font-size:.76rem;margin-bottom:4px">
    <span style="color:${statColor(i)}">■</span>
    <span style="flex:1">${esc(c.titleFa)}</span>
    <span class="num">${fa(c.count)}</span>
    <span class="muted num" style="min-width:44px;text-align:end">${fa(c.sharePct)}٪</span></div>`).join('');
  return `<div style="display:flex;gap:14px;align-items:center;flex-wrap:wrap">
    <svg width="140" height="140" viewBox="0 0 140 140" role="img">
      <circle cx="70" cy="70" r="${R}" fill="none" stroke="var(--border)" stroke-width="20"></circle>
      ${segs}
      <text x="70" y="66" text-anchor="middle" style="font-size:15px;font-weight:700" fill="currentColor">${fa(total)}</text>
      <text x="70" y="84" text-anchor="middle" style="font-size:10px" fill="currentColor">نفر</text>
    </svg>
    <div style="flex:1;min-width:180px">${legend}</div>
  </div>`;
}

/** جملهٔ خواندنی از پرترین خانهٔ جدول (متن در رابط، عدد با رقم فارسی). */
function statInsight(mx) {
  const peak = mx.peak;
  if (!peak || !peak.count) return 'برای این برش دادهای ثبت نشده است.';
  const rowTitle = (mx.rowsDim.categories.find(c => c.key === peak.rowKey) || {}).titleFa || peak.rowKey;
  const colTitle = (mx.colsDim.categories.find(c => c.key === peak.colKey) || {}).titleFa || peak.colKey;
  return `بیشترین تمرکز: <b>${esc(rowTitle)}</b> × <b>${esc(colTitle)}</b> با <b>${fa(peak.count)}</b> نفر — ` +
    `${fa(peak.sharePct)}٪ از ${fa(mx.total)} نفر؛ در سطر خود ${fa(peak.rowPct)}٪ و در ستون خود ${fa(peak.colPct)}٪.`;
}

/** برچسب حالت‌های نمایش. */
function statModeBar() {
  return `<div class="row-actions" style="margin-bottom:10px;flex-wrap:wrap">
    <span class="muted" style="font-size:.76rem;align-self:center">نمایش اعداد:</span>
    ${STAT_MODES.map(m => `<button class="btn btn-sm ${STAT_MODE === m.key ? 'btn-primary' : 'btn-ghost'}"
      title="${esc(m.hint)}" onclick="statSetMode('${m.key}')">${esc(m.label)}</button>`).join('')}
  </div>`;
}

/** تغییر حالت نمایش و بازترسیم ناحیه‌ها (بدون بارگذاری دوبارهٔ داده). */
function statSetMode(mode) {
  STAT_MODE = mode;
  statRenderSection();
  statRenderCustom();
  toast('نمایش اعداد: ' + ((STAT_MODES.find(m => m.key === mode) || {}).label || mode));
}

/* ----------------------------- برش دلخواه ----------------------------- */

/** فهرست‌های انتخاب بُعد سطر/ستون. */
function statDimSelects() {
  const dims = STAT_DIMS || [];
  const opts = (selected) => dims.map(d =>
    `<option value="${esc(d.key)}" ${d.key === selected ? 'selected' : ''}>${esc(d.titleFa)}</option>`).join('');
  return `<div class="row-actions" style="flex-wrap:wrap;gap:10px;align-items:center">
    <label style="font-size:.8rem">سطرها: <select id="statRows" style="font-size:.8rem">${opts(STAT_ROWS)}</select></label>
    <label style="font-size:.8rem">ستون‌ها: <select id="statCols" style="font-size:.8rem">${opts(STAT_COLS)}</select></label>
    <button class="btn btn-primary btn-sm" onclick="statRenderCustom(true)">📊 نمایش برش</button>
    <span class="muted" style="font-size:.74rem">هر ترکیبی از بُعدها ممکن است؛ جدول و نمودار همیشه با هم ساخته می‌شوند.</span>
  </div>`;
}

/** رشد نتیجهٔ برش دلخواه از سرور. */
async function statRenderCustom(force) {
  const region = document.getElementById('statCustomRegion');
  if (!region) return;
  if (force) {
    STAT_ROWS = (document.getElementById('statRows') || {}).value || STAT_ROWS;
    STAT_COLS = (document.getElementById('statCols') || {}).value || STAT_COLS;
    const res = await apiFetch(`/api/v1/statistics/crosstab?rows=${encodeURIComponent(STAT_ROWS)}&cols=${encodeURIComponent(STAT_COLS)}`);
    if (!res.ok) { region.innerHTML = liveErrorBox(res.message); return; }
    STAT_CUSTOM = res.data && res.data.matrix;
  }
  if (!STAT_CUSTOM) return;
  region.innerHTML = statMatrixCard(STAT_CUSTOM, 'custom');
}

/** کارت یک جدول توافقی: شاخص‌ها، جدول هیت‌مپی، میله‌های انباشته و CSV. */
function statMatrixCard(mx, key) {
  return `<div class="card card-pad" style="margin-bottom:14px">
    <div class="card-title">📐 ${esc(mx.titleFa)}
      <span class="muted" style="font-size:.74rem">— ${fa(mx.total)} نفر پرونده — ${fa(mx.rows.length)} سطر × ${fa(mx.cols.length)} ستون</span></div>
    ${mx.noteFa ? `<div class="muted" style="font-size:.76rem;margin-bottom:6px">${esc(mx.noteFa)}</div>` : ''}
    <div class="card card-pad" style="background:var(--tint,#EFF6FF);border:0;margin-bottom:10px;font-size:.8rem;line-height:2">${statInsight(mx)}</div>
    ${statMatrixTable(mx, STAT_MODE)}
    <div style="margin-top:12px">${statStackedBars(mx)}</div>
    <div class="row-actions" style="margin-top:8px">
      <button class="btn btn-sm btn-ghost" onclick="statCsv('${esc(key)}')">⬇️ خروجی CSV</button>
      <span class="muted" style="font-size:.72rem">خروجی CSV با ارقام لاتین و برای پردازش در صفحه‌گسترده است.</span>
    </div>
  </div>`;
}

/** خروجی CSV یک برش (ارقام لاتین — داده، نه نمایش). */
function statCsv(key) {
  const matrices = (STAT_SECTION && STAT_SECTION.matrices) || [];
  const mx = key === 'custom' ? STAT_CUSTOM : matrices.find(m => m.key === key);
  if (!mx) { toast('برش یافت نشد', 'error'); return; }
  const esc2 = v => '"' + String(v == null ? '' : v).replace(/"/g, '""') + '"';
  const lines = [];
  lines.push([esc2(mx.rowsDim.titleFa + ' / ' + mx.colsDim.titleFa)].concat(mx.cols.map(c => esc2(c.titleFa))).concat([esc2('جمع سطر')]).join(','));
  for (const row of mx.rows) {
    lines.push([esc2(row.titleFa)].concat(row.cells.map(c => c.count)).concat([row.count]).join(','));
  }
  lines.push([esc2('جمع ستون')].concat(mx.colTotals).concat([mx.total]).join(','));
  const blob = new Blob(['\ufeff' + lines.join('\r\n')], { type: 'text/csv;charset=utf-8' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = 'stat-' + mx.key + '.csv';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  toast('خروجی CSV ساخته شد');
}

/* ----------------------------- سکشن کامل ----------------------------- */

/** کارت یک شاخص کلیدی: عدد یا نام + واحد. */
function statMetricCard(metric) {
  const value = metric.textFa
    ? `${esc(metric.textFa)} <span class="num" style="font-size:.8rem">(${fa(metric.value)} ${esc(metric.unitFa || '')})</span>`
    : `${fa(metric.value)} <span style="font-size:.8rem">${esc(metric.unitFa || '')}</span>`;
  return statCard({
    ico: '📊', tint: '#EFF6FF', accent: '#2563EB', val: value, label: metric.titleFa,
  });
}

/** بازترسیم جدول‌ها و نمودارهای سکشن (پس از تغییر حالت نمایش). */
function statRenderSection() {
  const region = document.getElementById('statMatrixRegion');
  if (!region || !STAT_SECTION) return;
  region.innerHTML = statModeBar() + (STAT_SECTION.matrices || []).map(mx => statMatrixCard(mx, mx.key)).join('');
}

/** صفحهٔ «برش‌های آماری» — تمام شکل‌های آماری روی دادهٔ واقعی پرونده. */
function pageStatsSectionLive() {
  if (!isLive()) return offlineFallback('statsSection');
  liveLoad(async () => {
    const [sectionRes, dimsRes] = await Promise.all([
      apiFetch('/api/v1/statistics/section'),
      apiFetch('/api/v1/statistics/dimensions'),
    ]);
    if (!sectionRes.ok) return liveErrorBox(sectionRes.message);
    STAT_SECTION = sectionRes.data || {};
    STAT_DIMS = (dimsRes.ok && dimsRes.data && dimsRes.data.dimensions) || STAT_SECTION.dimensions || [];

    const metrics = STAT_SECTION.metrics || [];
    const kpis = `<div class="grid-stats">${metrics.map(statMetricCard).join('')}</div>`;

    // توزیع‌های تک‌بُعدی: بُعد ترتیبی با نمودار ستونی، بُعد اسمی با دونات.
    const distributions = (STAT_SECTION.distributions || []).map(dim => `
      <div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">${dim.kind === 'ordinal' ? '📶' : '🥧'} ${esc(dim.titleFa)}
          <span class="muted" style="font-size:.74rem">— ${fa(dim.categories.reduce((s, c) => s + c.count, 0))} نفر</span></div>
        ${dim.noteFa ? `<div class="muted" style="font-size:.74rem;margin-bottom:8px">${esc(dim.noteFa)}</div>` : ''}
        ${dim.kind === 'ordinal' ? statColumnChart(dim) : statDonut(dim)}
      </div>`).join('');

    // برش‌های آمادهٔ سکشن (سن×سابقه، تحصیلات×واحد، جنسیت×نوع استخدام و...) — هر کدام
    // جدول هیت‌مپی، میله‌های انباشته، جملهٔ تحلیل و خروجی CSV.
    const matrixHtml = statModeBar() +
      (STAT_SECTION.matrices || []).map(mx => statMatrixCard(mx, mx.key)).join('');

    return kpis +
      statNote(STAT_SECTION.noteFa) +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">🧭 برش دلخواه</div>
        <div class="muted" style="font-size:.76rem;margin-bottom:8px">هر دو بُعدی را کنار هم بگذارید تا جدول توافقی و نمودارش ساخته شود (مثلاً «وضعیت اشتغال × نوع استخدام»).</div>
        ${statDimSelects()}
        <div id="statCustomRegion" style="margin-top:12px"></div>
      </div>` +
      `<div class="grid-2eq" style="align-items:start">${distributions}</div>` +
      `<div id="statMatrixRegion" style="margin-top:14px">${matrixHtml}</div>`;
  });
  return liveShell('برش‌های آماری کارکنان',
    'جدول‌های توافقی و نمودارهای جمعیت کارکنان بر پایهٔ سن، سابقه، تحصیلات، واحد، نوع استخدام و جنسیت',
    `<button class="btn btn-ghost" onclick="STAT_SECTION=null;STAT_CUSTOM=null;liveReload()">↻ بازخوانی آمار</button>`);
}

/* ثبت در مسیریاب زنده (پیش از قالب نمایشی بررسی می‌شود). */
LIVE_PAGES.statsSection = pageStatsSectionLive;
