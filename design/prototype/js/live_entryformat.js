/* =====================================================================
   قالب ورود اطلاعات دستی — فرم‌ها و گزارش‌ها
   =====================================================================

   هیچ قاعده‌ای اینجا ساخته نمی‌شود: الگوی هر ستون (تاریخ، مبلغ، عدد، انتخابی)،
   مقادیر مجاز و سطر نمونه از سرور می‌آید
   (`/settings/data-sources/entry-format` و `/reports/entry-format`)، چون هر دو از
   همان ساختار دادهٔ ماژول‌ها ساخته می‌شوند که قالب اکسل هم از آن می‌آید.

   این کارت پس از رسم صفحهٔ «تبادل داده و قالب‌های بارگذاری» تزریق می‌شود.
   ===================================================================== */

const EF_STATE = {forms: [], sources: [], sheet: null, source: null, titleFa: ''};

/** کارت قالب ورود دستی را می‌سازد و در انتهای ناحیهٔ زنده تزریق می‌کند. */
async function deRenderEntryFormat() {
  const region = document.getElementById('liveRegion');
  if (!region || document.getElementById('efCard')) return;

  const [formsRes, sourcesRes] = await Promise.all([
    apiFetch('/api/v1/settings/data-sources/entry-format'),
    apiFetch('/api/v1/reports/entry-format'),
  ]);
  if (!formsRes.ok) return;
  EF_STATE.forms = ((formsRes.data || {}).entities) || [];
  EF_STATE.sources = (sourcesRes.ok && ((sourcesRes.data || {}).sources)) || [];

  const formOpts = EF_STATE.forms.map(e =>
    `<option value="${esc(e.key)}">${esc(e.titleFa)} — ${esc(e.groupFa || e.moduleFa || '')}</option>`).join('');
  const sourceOpts = EF_STATE.sources.map(s =>
    `<option value="${esc(s.key)}">${esc(s.titleFa)} (${fa(s.fields)} ستون)</option>`).join('');

  const card = document.createElement('div');
  card.id = 'efCard';
  card.className = 'card card-pad';
  card.style.marginTop = '12px';
  card.innerHTML = `
    <div class="card-title">📋 قالب ورود اطلاعات دستی — فرم‌ها و گزارش‌ها</div>
    <div class="muted" style="font-size:.78rem;line-height:2;margin-bottom:8px">
      این برگه می‌گوید هر ستون چه نوعی دارد، چه شکلی نوشته می‌شود و یک مقدار نمونه چیست؛
      همان ساختاری که قالب اکسل و اعتبارسنجی سرور از آن ساخته می‌شود.
    </div>
    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:10px;align-items:end">
      <div class="field"><label>فرم / جدول</label>
        <select id="efFormSelect">${formOpts}</select></div>
      <div class="field"><label>منبع گزارش</label>
        <select id="efSourceSelect">${sourceOpts}</select></div>
      <div class="row-actions">
        <button class="btn btn-primary" onclick="efShowForm()">نمایش قالب فرم</button>
        <button class="btn" onclick="efShowSource()">نمایش قالب گزارش</button>
        <button class="btn btn-ghost" onclick="efPrint()">🖨️ چاپ برگهٔ قالب</button>
      </div>
    </div>
    <div id="efSheet" style="margin-top:10px"></div>`;
  region.appendChild(card);
  efShowForm();
}

/** نمایش برگهٔ قالب یک فرم/جدول. */
async function efShowForm() {
  const select = document.getElementById('efFormSelect');
  if (!select) return;
  const key = select.value;
  const res = await apiFetch('/api/v1/settings/data-sources/entry-format?entity=' + encodeURIComponent(key));
  if (!res.ok) { toast(res.message, 'error'); return; }
  const sheet = (res.data || {}).sheet || {};
  EF_STATE.sheet = sheet;
  EF_STATE.source = null;
  EF_STATE.titleFa = 'قالب ورود دستی — ' + (sheet.titleFa || key);
  efPaint();
}

/** نمایش برگهٔ قالب ستون‌های یک منبع گزارش. */
async function efShowSource() {
  const select = document.getElementById('efSourceSelect');
  if (!select) return;
  const key = select.value;
  const res = await apiFetch('/api/v1/reports/entry-format?source=' + encodeURIComponent(key));
  if (!res.ok) { toast(res.message, 'error'); return; }
  const d = res.data || {};
  EF_STATE.sheet = {key: d.source, titleFa: d.titleFa, format: d.format || [], sampleRow: d.sampleRow || [], noteFa: d.noteFa};
  EF_STATE.source = key;
  EF_STATE.titleFa = 'قالب ورود دستی — گزارش ' + (d.titleFa || key);
  efPaint();
}

/** رسم جدول برگهٔ قالب. */
function efPaint() {
  const box = document.getElementById('efSheet');
  if (!box || !EF_STATE.sheet) return;
  const sheet = EF_STATE.sheet;
  const rows = (sheet.format || []).map(col => `<tr>
      <td class="num">${fa(col.column)}</td>
      <td>${esc(col.labelFa)}</td>
      <td class="ltr" style="font-size:.76rem">${esc(col.key)}</td>
      <td>${esc(col.typeFa || col.type)}${col.required ? ' <span style="color:#dc2626">✻</span>' : ''}</td>
      <td style="font-size:.76rem">${esc(col.formatFa || '—')}</td>
      <td style="font-size:.74rem">${(col.valuesFa || []).map(esc).join('، ') || '—'}</td>
      <td style="font-size:.76rem">${esc(col.example || '—')}</td>
    </tr>`);
  const sample = (sheet.sampleRow || []).length
    ? `<div class="muted" style="font-size:.78rem;margin-top:8px">سطر نمونه: ${(sheet.sampleRow || []).map(esc).join(' · ')}</div>`
    : '';
  box.innerHTML = tbl(['ستون', 'عنوان', 'کلید', 'نوع', 'الگوی ورود', 'مقادیر مجاز', 'نمونه'], rows, {foot: false}) +
    sample +
    (sheet.noteFa ? `<div class="muted" style="font-size:.76rem;margin-top:6px">${esc(sheet.noteFa)}</div>` : '');
}

/** چاپ برگهٔ قالب در زبانهٔ تازه (همان الگوی چاپ گزارش). */
function efPrint() {
  if (!EF_STATE.sheet) { toast('اول قالب را نمایش دهید', 'error'); return; }
  const sheet = EF_STATE.sheet;
  const brand = (window.activeOrg || {}).titleFa || '';
  const head = ['ستون', 'عنوان', 'کلید', 'نوع', 'الزامی', 'الگوی ورود', 'مقادیر مجاز', 'نمونه'];
  const rows = (sheet.format || []).map(col => `<tr>
      <td class="num">${fa(col.column)}</td><td>${esc(col.labelFa)}</td>
      <td class="ltr">${esc(col.key)}</td><td>${esc(col.typeFa || col.type)}</td>
      <td>${col.required ? 'بله' : 'خیر'}</td><td>${esc(col.formatFa || '—')}</td>
      <td>${(col.valuesFa || []).map(esc).join('، ') || '—'}</td><td>${esc(col.example || '—')}</td>
    </tr>`).join('');
  const html = `<!DOCTYPE html><html lang="fa" dir="rtl"><head><meta charset="utf-8">
<title>${esc(EF_STATE.titleFa)}</title>
<style>
  @page{size:A4 portrait;margin:12mm}
  ${printFontFacesSafe()}
  body{font-family:Vazirmatn,Vazirmatn-fallback,Tahoma,sans-serif;color:#1F2937;margin:0}
  h1{font-size:1rem;margin:0 0 4px}
  .sub{font-size:.76rem;color:#6B7280;margin-bottom:10px}
  table{width:100%;border-collapse:collapse;font-size:.76rem}
  th,td{border:1px solid #E5E7EB;padding:5px 7px;text-align:start}
  th{background:#F3F4F6}
  .note{margin-top:10px;font-size:.74rem;color:#6B7280;border-top:1px dashed #E5E7EB;padding-top:6px}
</style></head><body>
<h1>${esc(EF_STATE.titleFa)}</h1>
<div class="sub">${esc(brand)} — راهنمای ورود دستی داده (الگو، مقادیر مجاز و نمونه)</div>
<table><thead><tr>${head.map(h => `<th>${esc(h)}</th>`).join('')}</tr></thead><tbody>${rows}</tbody></table>
${sheet.noteFa ? `<p class="note">${esc(sheet.noteFa)}</p>` : ''}
<script>window.addEventListener('load',()=>setTimeout(()=>window.print(),350));<\/script>
</body></html>`;
  const url = URL.createObjectURL(new Blob([html], {type: 'text/html;charset=utf-8'}));
  window.open(url, '_blank');
  setTimeout(() => URL.revokeObjectURL(url), 60000);
}