/* =====================================================================
   پنل پارامتریک فرم‌ها و گزارش‌ها (ماژول تنظیمات)
   ---------------------------------------------------------------------
   چرا این صفحه: فهرست فیلدهای هر فرم و ستون‌های هر گزارش از خودِ سرور می‌آید
   (`/api/v1/settings/forms` و `/api/v1/settings/reports`). کاربر اینجا فقط
   پارامتر می‌سازد: کدام فیلد دیده شود، کدام اجباری باشد، برچسب چه باشد و کدام
   ستون‌ها در گزارش بیایند. پارامتر یک رکورد از نوع «پارامتر فرم و گزارش» است،
   پس در سابقهٔ حسابرسی می‌ماند و با «غیرفعال‌سازی» بی‌اثر می‌شود.
   هیچ فهرست فیلدی در این فایل هاردکد نشده است.
   ===================================================================== */

var PARAM_TAB = 'forms';
var PARAM_FORMS = null;
var PARAM_REPORTS = null;

/** صفحهٔ پنل پارامتریک.
 *
 * قالب همهٔ صفحه‌های زندهٔ سامانه: تابع همگام است، پوسته را برمی‌گرداند و بارگذاری
 * داده در liveLoad انجام می‌شود (همان الگوی فهرست کارکنان و پرونده).
 */
function pageFormParamsLive() {
  if (!isLive()) return offlineFallback('formParams');
  // محتوای صفحه در liveLoad بارگذاری می‌شود و داخل #liveRegion می‌نشیند؛ پوسته همین‌جا
  // برگردانده می‌شود (همان الگوی صفحه‌های زندهٔ دیگر).
  liveLoad(async () => {
    const [formsRes, reportsRes] = await Promise.all([
      apiFetch('/api/v1/settings/forms'),
      apiFetch('/api/v1/settings/reports'),
    ]);
    if (!formsRes.ok) return liveErrorBox(formsRes.message);
    PARAM_FORMS = (formsRes.data && formsRes.data.forms) || [];
    PARAM_REPORTS = (reportsRes.ok && reportsRes.data && reportsRes.data.reports) || [];

    const tabBar = `<div class="tabs">
      <button class="tab ${PARAM_TAB === 'forms' ? 'active' : ''}" onclick="PARAM_TAB='forms';liveReload()">🧾 فیلدهای فرم‌ها (${fa(PARAM_FORMS.length)})</button>
      <button class="tab ${PARAM_TAB === 'reports' ? 'active' : ''}" onclick="PARAM_TAB='reports';liveReload()">📊 ستون‌های گزارش‌ها (${fa(PARAM_REPORTS.length)})</button>
    </div>`;

      const body = PARAM_TAB === 'forms' ? paramsFormsTable(PARAM_FORMS) : paramsReportsCards(PARAM_REPORTS);
    return tabBar +
      statNote('فهرست فرم‌ها و فیلدهای هر فرم از شناسهٔ خودِ ماژول‌ها خوانده می‌شود؛ پس با افزودن فیلد تازه در دامنه، همین‌جا خودکار دیده می‌شود. پارامتری که «غیرفعال» شود داده‌اش می‌ماند ولی در فرم اثر نمی‌گذارد.') +
      body;
  });
  return liveShell('پنل پارامتریک فرم‌ها و گزارش‌ها',
    'کم و زیاد کردن فیلدهای فرم‌ها و ستون‌های گزارش‌ها به‌صورت پارامتریک — بدون تغییر نرم‌افزار',
    `<button class="btn btn-ghost" onclick="PARAM_FORMS=null;PARAM_REPORTS=null;liveReload()">↻ تازه‌سازی کاتالوگ</button>`);
}

/** جدول فرم‌ها با شمار فیلدهای مؤثر. */
function paramsFormsTable(forms) {
  const rows = forms.map((f, i) => `<tr>
    <td><span class="ltr num" style="font-size:.72rem">${esc(f.target)}</span></td>
    <td>${esc(f.titleFa)}</td>
    <td>${esc(f.moduleTitleFa || f.module)}</td>
    <td class="num">${fa(f.fieldCount)}</td>
    <td class="num">${fa(f.visibleCount)}</td>
    <td class="num">${fa(f.requiredCount)}</td>
    <td>${f.paramsId ? statusChip(f.status, f.statusFa || f.status) : '<span class="muted">بدون پارامتر</span>'}</td>
    <td><div class="row-actions">
      <button class="btn btn-sm btn-primary" onclick="paramEditForm('${esc(f.target)}')">⚙️ تنظیم فیلدها</button>
      ${f.paramsId ? `<button class="btn btn-sm btn-ghost" title="غیرفعال‌سازی / بازفعال‌سازی" onclick="paramToggle('${esc(f.paramsId)}','${f.status === 'inactive' ? 'activate' : 'deactivate'}')">${f.status === 'inactive' ? '↩️ بازفعال' : '⏸ غیرفعال'}</button>` : ''}
    </div></td></tr>`);
  return liveTable(['هدف (ماژول|نوع)', 'عنوان فرم', 'ماژول', 'فیلدها', 'نمایش‌داده‌شده', 'اجباری', 'پارامتر', 'عملیات'],
    rows, forms.length);
}

/** کارت هر منبع گزارش با تیک ستون‌ها و اجرای گزارش. */
function paramsReportsCards(reports) {
  if (!reports.length) return statNote('منبع دادهٔ گزارشی در دسترس نیست.');
  return reports.map(r => {
    const cols = r.columns.map(c => `<label style="display:inline-flex;align-items:center;gap:6px;font-size:.8rem;margin:2px 10px 2px 0">
      <input type="checkbox" class="repCol" data-source="${esc(r.source)}" value="${esc(c.key)}" ${c.selected ? 'checked' : ''}>
      <span>${esc(c.labelFa)}</span></label>`).join('');
    return `<div class="card card-pad" style="margin-bottom:10px">
      <div class="card-title">📊 ${esc(r.titleFa)} <span class="muted" style="font-size:.75rem">— ${esc(r.source)} — ${fa(r.columnCount)} ستون</span></div>
      <div style="margin:4px 0 8px">${cols}</div>
      <div class="row-actions">
        <button class="btn btn-primary" onclick="paramSaveReport('${esc(r.source)}')">💾 ذخیرهٔ ستون‌ها</button>
        <button class="btn btn-ghost" onclick="paramRunReport('${esc(r.source)}')">▶️ اجرای گزارش با همین ستون‌ها</button>
        ${r.paramsId ? `<span class="chip">${esc(statusFaText(r.status))}</span>` : ''}
      </div>
      <div id="repResult_${esc(r.source)}" style="margin-top:10px"></div>
    </div>`;
  }).join('');
}

/** عنوان فارسی وضعیت پارامتر در کارت گزارش. */
function statusFaText(status) {
  return status === 'active' ? 'جاری' : status === 'inactive' ? 'غیرفعال' : 'پیش‌نویس';
}

/** فرم تنظیم فیلدهای یک فرم: نمایش، اجباری و برچسب سفارشی. */
function paramEditForm(target) {
  const form = (PARAM_FORMS || []).find(f => f.target === target);
  if (!form) { toast('این فرم در کاتالوگ نیست', 'error'); return; }
  const rows = form.fields.map(f => `<tr>
    <td><label style="display:inline-flex;align-items:center;gap:6px">
      <input type="checkbox" class="fpVisible" data-key="${esc(f.key)}" ${f.visible ? 'checked' : ''}>
      <span class="ltr" style="font-size:.74rem">${esc(f.key)}</span></label></td>
    <td>${f.source ? `<span class="chip" style="font-size:.7rem">از پروندهٔ کارمند</span>` : '<span class="muted">دستی</span>'}</td>
    <td><label style="display:inline-flex;align-items:center;gap:6px">
      <input type="checkbox" class="fpRequired" data-key="${esc(f.key)}" ${f.required ? 'checked' : ''}> اجباری</label></td>
    <td><input class="fpLabel" data-key="${esc(f.key)}" value="${esc(f.labelFa)}" style="font-size:.78rem"></td>
  </tr>`);
  openModal({
    title: 'تنظیم فیلدهای فرم — ' + form.titleFa,
    body: `<div class="muted" style="font-size:.78rem;margin-bottom:8px">هدف: <span class="ltr">${esc(form.target)}</span> — با برداشتن تیک هر فیلد، آن فیلد از فرم حذف می‌شود و با زدن تیک برمی‌گردد. برچسب را می‌توانید فارسی‌تر کنید.</div>
      ${liveTable(['نمایش', 'منبع مقدار', 'اجباری', 'برچسب', ], rows, form.fields.length)}`,
    foot: `<button class="btn btn-primary" onclick="paramSaveForm('${esc(form.target)}')">💾 ذخیره و جاری‌سازی</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}

/** پاک‌کردن حافظهٔ پارامترها تا فرم‌های بازشده پارامتر تازه را ببینند.
 *
 * چرا: پارامترها یک بار در هر بارگذاری صفحه خوانده و نگه داشته می‌شوند (تا هر فرم
 * درخواست تازه نزند)؛ پس بعد از هر تغییر باید همان حافظه پاک شود وگرنه فرم‌ها
 * مقدار قدیمی را نشان می‌دهند.
 */
function paramClearFormCache() {
  if (typeof FORM_PARAMS_CACHE !== 'undefined') FORM_PARAMS_CACHE = null;
}

/** ذخیرهٔ پارامتر فرم: رکورد تازه یا ویرایش پارامتر موجود، سپس جاری‌سازی. */
async function paramSaveForm(target) {
  const form = (PARAM_FORMS || []).find(f => f.target === target);
  if (!form) return;
  const visible = [], required = [], labels = [];
  for (const f of form.fields) {
    const v = document.querySelector(`.fpVisible[data-key="${cssEscape(f.key)}"]`);
    const r = document.querySelector(`.fpRequired[data-key="${cssEscape(f.key)}"]`);
    const l = document.querySelector(`.fpLabel[data-key="${cssEscape(f.key)}"]`);
    if (v && v.checked) visible.push(f.key);
    if (r && r.checked) required.push(f.key);
    if (l && l.value.trim() && l.value.trim() !== (f.labelFa || '')) labels.push(f.key + '=' + l.value.trim());
  }
  // «همهٔ فیلدها نمایان» یعنی پارامتری لازم نیست؛ خالی ذخیره می‌شود تا فیلد تازهٔ آینده هم دیده شود.
  const visibleValue = visible.length === form.fieldCount ? '' : visible.join(',');
  const fields = {
    scope: 'form',
    target: target,
    visibleFields: visibleValue,
    requiredFields: required.join(','),
    labels: labels.join(','),
    note: 'ساخته‌شده از پنل پارامتریک',
  };
  const res = form.paramsId
    ? await apiFetch('/api/v1/settings/records/' + encodeURIComponent(form.paramsId), {method: 'PATCH', body: {kind: 'formParams', title: 'پارامتر ' + form.titleFa, fields}})
    : await apiFetch('/api/v1/settings/records', {method: 'POST', body: {kind: 'formParams', title: 'پارامتر ' + form.titleFa, fields, noteFa: 'ثبت از پنل پارامتریک'}});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = form.paramsId || (res.data && res.data.id);
  const act = await apiFetch('/api/v1/settings/records/' + encodeURIComponent(id) + '/action', {method: 'POST', body: {action: 'activate'}});
  if (!act.ok && act.message) toast(act.message, 'error');
  PARAM_FORMS = null;
  paramClearFormCache();
  closeModal();
  toast('پارامتر فرم ذخیره و جاری شد');
  liveReload();
}

/** غیرفعال/فعال کردن پارامتر از فهرست. */
async function paramToggle(id, action) {
  const res = await apiFetch('/api/v1/settings/records/' + encodeURIComponent(id) + '/action', {method: 'POST', body: {action}});
  if (!res.ok) { toast(res.message, 'error'); return; }
  PARAM_FORMS = null;
  paramClearFormCache();
  toast(action === 'activate' ? 'پارامتر جاری شد' : 'پارامتر غیرفعال شد');
  liveReload();
}

/** ستون‌های تیک‌خوردهٔ یک منبع گزارش. */
function paramSelectedColumns(source) {
  return [...document.querySelectorAll(`.repCol[data-source="${cssEscape(source)}"]`)]
    .filter(el => el.checked).map(el => el.value);
}

/** ذخیرهٔ ستون‌های انتخابی یک گزارش. */
async function paramSaveReport(source) {
  const report = (PARAM_REPORTS || []).find(r => r.source === source);
  if (!report) return;
  const columns = paramSelectedColumns(source);
  if (!columns.length) { toast('دست‌کم یک ستون انتخاب کنید', 'error'); return; }
  const fields = {scope: 'report', target: source, columns: columns.join(','), note: 'ساخته‌شده از پنل پارامتریک'};
  const res = report.paramsId
    ? await apiFetch('/api/v1/settings/records/' + encodeURIComponent(report.paramsId), {method: 'PATCH', body: {kind: 'formParams', title: 'ستون‌های گزارش ' + report.titleFa, fields}})
    : await apiFetch('/api/v1/settings/records', {method: 'POST', body: {kind: 'formParams', title: 'ستون‌های گزارش ' + report.titleFa, fields, noteFa: 'ثبت از پنل پارامتریک'}});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = report.paramsId || (res.data && res.data.id);
  await apiFetch('/api/v1/settings/records/' + encodeURIComponent(id) + '/action', {method: 'POST', body: {action: 'activate'}});
  PARAM_REPORTS = null;
  paramClearFormCache();
  toast('ستون‌های گزارش ذخیره شد');
  liveReload();
}

/** اجرای گزارش با ستون‌های تیک‌خورده و نمایش نتیجه. */
async function paramRunReport(source) {
  const box = document.getElementById('repResult_' + source);
  const columns = paramSelectedColumns(source);
  if (!box) return;
  if (!columns.length) { box.innerHTML = '<div class="muted">دست‌کم یک ستون انتخاب کنید.</div>'; return; }
  box.innerHTML = '<div class="muted">در حال اجرا...</div>';
  const res = await apiFetch('/api/v1/reports/run', {method: 'POST', body: {source, items: columns, limit: 200}});
  if (!res.ok) { box.innerHTML = `<div class="empty-state"><div class="big">⚠️</div>${esc(res.message)}</div>`; return; }
  const r = res.data || {};
  const head = (r.columns || []).map(c => c.labelFa || c.key);
  const rows = (r.rows || []).slice(0, 60).map(row => `<tr>${row.map(cell => `<td>${esc(cell == null ? '' : cell)}</td>`).join('')}</tr>`);
  box.innerHTML = `<div class="muted" style="margin-bottom:6px">${esc(r.titleFa || source)} — ${fa(r.total || 0)} سطر${(r.total || 0) > 60 ? ' (نمایش ۶۰ سطر نخست)' : ''}</div>` +
    liveTable(head, rows, r.total || 0);
}

/** فرار دادن کلید برای استفاده در انتخابگر CSS. */
function cssEscape(value) {
  return String(value == null ? '' : value).replace(/["\\]/g, '\\$&');
}