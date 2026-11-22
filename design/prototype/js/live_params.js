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
var PARAM_DOCUMENTS = null;
/** تعریف گزارش‌ها و منابع دادهٔ آن‌ها (تب «تعریف گزارش‌ها»). */
var PARAM_DEFS = null;
var PARAM_DEF_SOURCES = null;

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
    const [formsRes, reportsRes, docsRes, defsRes] = await Promise.all([
      apiFetch('/api/v1/settings/forms'),
      apiFetch('/api/v1/settings/reports'),
      apiFetch('/api/v1/settings/documents'),
      apiFetch('/api/v1/settings/report-defs'),
    ]);
    if (!formsRes.ok) return liveErrorBox(formsRes.message);
    PARAM_FORMS = (formsRes.data && formsRes.data.forms) || [];
    PARAM_REPORTS = (reportsRes.ok && reportsRes.data && reportsRes.data.reports) || [];
      PARAM_DOCUMENTS = (docsRes.ok && docsRes.data && docsRes.data.documents) || [];
      PARAM_DEFS = (defsRes.ok && defsRes.data && defsRes.data.reports) || [];
      PARAM_DEF_SOURCES = (defsRes.ok && defsRes.data && defsRes.data.sources) || [];

    const tabBar = `<div class="tabs">
      <button class="tab ${PARAM_TAB === 'forms' ? 'active' : ''}" onclick="PARAM_TAB='forms';liveReload()">🧾 فیلدهای فرم‌ها (${fa(PARAM_FORMS.length)})</button>
      <button class="tab ${PARAM_TAB === 'reports' ? 'active' : ''}" onclick="PARAM_TAB='reports';liveReload()">📊 ستون‌های گزارش‌ها (${fa(PARAM_REPORTS.length)})</button>
      <button class="tab ${PARAM_TAB === 'defs' ? 'active' : ''}" onclick="PARAM_TAB='defs';liveReload()">📑 تعریف گزارش‌ها (${fa((PARAM_DEFS || []).length)})</button>
      <button class="tab ${PARAM_TAB === 'documents' ? 'active' : ''}" onclick="PARAM_TAB='documents';liveReload()">🖨️ اسناد چاپی (${fa((PARAM_DOCUMENTS || []).length)})</button>
    </div>`;

      const body = PARAM_TAB === 'forms' ? paramsFormsTable(PARAM_FORMS)
        : PARAM_TAB === 'reports' ? paramsReportsCards(PARAM_REPORTS)
        : PARAM_TAB === 'defs' ? paramsReportDefs(PARAM_DEFS || [], PARAM_DEF_SOURCES || [])
        : paramsDocumentsCards(PARAM_DOCUMENTS || []);
    return tabBar +
      statNote('فهرست فرم‌ها و فیلدهای هر فرم از شناسهٔ خودِ ماژول‌ها خوانده می‌شود؛ پس با افزودن فیلد تازه در دامنه، همین‌جا خودکار دیده می‌شود. پارامتری که «غیرفعال» شود داده‌اش می‌ماند ولی در فرم اثر نمی‌گذارد.') +
      body;
  });
  return liveShell('پنل پارامتریک فرم‌ها و گزارش‌ها',
    'کم و زیاد کردن فیلدهای فرم‌ها و ستون‌های گزارش‌ها به‌صورت پارامتریک — بدون تغییر نرم‌افزار',
    `<button class="btn btn-ghost" onclick="PARAM_FORMS=null;PARAM_REPORTS=null;PARAM_DOCUMENTS=null;liveReload()">↻ تازه‌سازی کاتالوگ</button>`);
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

/** فهرست تعریف گزارش‌ها با امکان ساخت گزارش تازه، فعال‌سازی و غیرفعال‌سازی.
 *
 * چرا اینجا: پنل پارامترها باید گزارش را «کم و زیاد» کند، نه فقط ستون‌هایش را. تعریف گزارش
 * همان رکورد نوع «reportDefs» ماژول گزارش‌هاست؛ این بخش آن را می‌خواند و برای ساخت/تغییر،
 * مسیر رسمی همان ماژول را صدا می‌زند تا قاعدهٔ گردشکار در یک جا بماند.
 */
function paramsReportDefs(defs, sources) {
  const rows = defs.map(d => `<tr>
    <td class="ltr" style="font-size:.72rem">${esc(d.id)}</td>
    <td>${esc(d.titleFa)}</td>
    <td class="ltr" style="font-size:.72rem">${esc(d.source)}</td>
    <td class="ltr" style="font-size:.72rem">${esc(d.items || '—')}</td>
    <td class="num">${esc(d.limit || '—')}</td>
    <td>${statusChip(d.status, d.statusFa || d.status)}</td>
    <td><div class="row-actions">
      <button class="btn btn-sm btn-ghost" onclick="paramDefToggle('${esc(d.id)}','${d.status === 'active' ? 'revise' : 'activate'}')">
        ${d.status === 'active' ? '⏸ غیرفعال' : '↩️ فعال‌سازی'}</button>
    </div></td></tr>`);

  const sourceOpts = (sources || []).map(s =>
    `<option value="${esc(s.key)}">${esc(s.titleFa)} — ${esc(s.key)}</option>`).join('');

  const createCard = `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">➕ گزارش تازه</div>
    <div class="muted" style="font-size:.75rem;line-height:2;margin-bottom:8px">
      گزارش تازه از یکی از منابع دادهٔ سامانه ساخته می‌شود؛ اقلام (ستون‌ها) را با کلید لاتین و جداکنندهٔ ویرگول بنویسید.
      غیرفعال‌سازی، تعریف را پاک نمی‌کند و با «فعال‌سازی» برمی‌گردد.
    </div>
    <div style="display:flex;gap:10px;flex-wrap:wrap;align-items:flex-end">
      <div class="field" style="min-width:260px"><label>منبع داده</label><select id="pdSource">${sourceOpts || '<option value="">—</option>'}</select></div>
      <div class="field" style="min-width:220px"><label>عنوان گزارش</label><input class="inp" id="pdTitle" placeholder="مثلاً گزارش نیروی انسانی واحد"></div>
      <div class="field" style="min-width:280px"><label>اقلام (با ویرگول)</label><input class="inp ltr" id="pdItems" placeholder="id,fullName,departmentName"></div>
      <div class="field"><label>گروه‌بندی</label><input class="inp ltr" id="pdGroup" placeholder="departmentName"></div>
      <div class="field"><label>سقف سطر</label><input class="inp num" id="pdLimit" value="200"></div>
      <button class="btn btn-primary" onclick="paramDefCreate()">➕ ساخت گزارش</button>
    </div>
    <div id="pdMsg" style="margin-top:8px"></div>
  </div>`;

  const fieldsHint = (sources || []).length
    ? `<div class="card card-pad" style="margin-bottom:12px">
        <div class="card-title">🧭 اقلام هر منبع داده</div>
        <div class="muted" style="font-size:.76rem;line-height:2">
          ${(sources || []).map(s => `<b>${esc(s.titleFa)}</b> (<span class="ltr">${esc(s.key)}</span>): ${esc(s.fields || '—')}`).join('<br>')}
        </div>
      </div>` : '';

  return createCard + fieldsHint +
    `<div class="card card-pad">
      <div class="card-title">📑 تعریف گزارش‌های سامانه (${fa(defs.length)})</div>
      ${defs.length
        ? liveTable(['شناسه', 'عنوان', 'منبع داده', 'اقلام', 'سقف سطر', 'وضعیت', 'اقدام'], rows, defs.length)
        : '<div class="muted" style="font-size:.8rem">گزارشی تعریف نشده است.</div>'}
    </div>`;
}

/** ساخت تعریف گزارش تازه. */
async function paramDefCreate() {
  const val = id => {
    const el = document.getElementById(id);
    return el ? el.value.trim() : '';
  };
  const source = val('pdSource');
  const title = val('pdTitle');
  const items = val('pdItems');
  if (!source || !title || !items) { toast('منبع، عنوان و اقلام گزارش الزامی است', 'error'); return; }
  const res = await apiFetch('/api/v1/reports/records', {
    method: 'POST',
    body: {
      kind: 'reportDefs', title,
      fields: {source, items, groupBy: val('pdGroup'), limit: val('pdLimit')},
      noteFa: 'ساخت تعریف گزارش از پنل پارامترها',
    },
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = res.data && res.data.id;
  if (id) await apiFetch('/api/v1/reports/records/' + encodeURIComponent(id) + '/action', {
    method: 'POST', body: {action: 'activate', noteFa: 'فعال‌سازی خودکار تعریف تازه'},
  });
  toast('تعریف گزارش ساخته و فعال شد');
  PARAM_DEFS = null;
  liveReload();
}

/** فعال/غیرفعال کردن تعریف گزارش. */
async function paramDefToggle(id, action) {
  const res = await apiFetch('/api/v1/reports/records/' + encodeURIComponent(id) + '/action', {
    method: 'POST', body: {action},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast(action === 'activate' ? 'تعریف گزارش فعال شد' : 'تعریف گزارش غیرفعال شد');
  PARAM_DEFS = null;
  liveReload();
}

/** کارت هر سند چاپی: تیک بلوک‌ها و عنوان دلخواه هر بلوک. */
function paramsDocumentsCards(docs) {
  if (!docs.length) return statNote('کاتالوگ اسناد چاپی در دسترس نیست.');
  return docs.map(d => {
    const texts = paramDocTextAreas(d);
    const rows = d.blocks.map(b => `<tr>
      <td><label style="display:inline-flex;align-items:center;gap:6px">
        <input type="checkbox" class="docBlock" data-doc="${esc(d.kind)}" data-key="${esc(b.key)}" ${b.visible ? 'checked' : ''}>
        <span>${esc(b.titleFa)}</span></label></td>
      <td><input class="docBlockTitle" data-doc="${esc(d.kind)}" data-key="${esc(b.key)}"
            value="${esc(b.customTitle || '')}" placeholder="${esc(b.titleFa)}" style="font-size:.78rem"></td>
    </tr>`);
    return `<div class="card card-pad" style="margin-bottom:10px">
      <div class="card-title">🖨️ ${esc(d.titleFa)} <span class="muted" style="font-size:.75rem">— ${esc(d.kind)} — ${fa(d.visibleCount)} از ${fa(d.blocks.length)} بلوک روشن</span>
        ${d.paramsId ? `<span class="chip">${esc(statusFaText(d.status))}</span>` : ''}</div>
      <div class="muted" style="font-size:.75rem;margin-bottom:6px">هر بلوک را می‌توانید از سند حذف کنید (تیک را بردارید) یا عنوانش را فارسی‌تر کنید؛ خالی‌بودن عنوان یعنی همان عنوان پیش‌فرض سامانه. متن پاراگراف و یادداشت پای سند را هم در کادرهای زیر همین جدول می‌توانید ویرایش کنید.</div>
      ${liveTable(['بلوک روی سند', 'عنوان سفارشی (اختیاری)'], rows, d.blocks.length)}
      ${texts}
      <div class="row-actions" style="margin-top:8px">
        <button class="btn btn-primary" onclick="paramSaveDocument('${esc(d.kind)}')">💾 ذخیرهٔ تنظیمات سند</button>
        ${d.paramsId ? `<button class="btn btn-ghost" onclick="paramToggle('${esc(d.paramsId)}','${d.status === 'inactive' ? 'activate' : 'deactivate'}')">${d.status === 'inactive' ? '↩️ بازفعال' : '⏸ غیرفعال'}</button>` : ''}
      </div>
    </div>`;
  }).join('');
}

/** کادرهای متن نوشتنی یک سند: پاراگراف متن و یادداشت پای برگه.
 *
 * چرا textarea و نه ورودی یک‌خطی: متن گواهی اشتغال چند پاراگراف است و هر خط یک
 * پاراگراف روی کاغذ می‌شود؛ همان متنی که در سرور به قالب پیش‌فرض سند تبدیل می‌شود.
 */
function paramDocTextAreas(d) {
  const texts = d.texts || [];
  if (!texts.length) return '';
  return `<div style="margin-top:12px">
    ${texts.map(t => `<div style="margin-bottom:10px">
      <div class="muted" style="font-size:.78rem;margin-bottom:4px">📝 ${esc(t.titleFa)} <span class="chip">${t.custom ? 'ویرایش‌شده' : 'پیش‌فرض'}</span></div>
      <textarea class="docText" data-doc="${esc(d.kind)}" data-slot="${esc(t.key)}" rows="4"
        style="width:100%;font-size:.8rem;line-height:1.9;resize:vertical">${esc(t.text || '')}</textarea>
      <div class="row-actions" style="margin-top:4px">
        <button class="btn btn-sm btn-ghost" onclick="paramResetDocText('${esc(d.kind)}','${esc(t.key)}')">↺ بازگرداندن پیش‌فرض</button>
        <span class="muted" style="font-size:.72rem">${esc(t.hintFa)}</span>
      </div>
    </div>`).join('')}
  </div>`;
}

/** بازگرداندن یک جای متن به متن پیش‌فرض مؤثر سند (پیش از ذخیره). */
function paramResetDocText(kind, slot) {
  const doc = (PARAM_DOCUMENTS || []).find(d => d.kind === kind);
  if (!doc) return;
  const view = (doc.texts || []).find(t => t.key === slot);
  const el = document.querySelector(`.docText[data-doc="${cssEscape(kind)}"][data-slot="${cssEscape(slot)}"]`);
  if (!view || !el) return;
  el.value = view.default || '';
  toast('متن پیش‌فرض در کادر گذاشته شد؛ برای اعمال، ذخیره کنید');
}

/** بلوک‌های تیک‌خورده و عنوان‌های سفارشی یک سند. */
function paramSelectedDocBlocks(kind) {
  const visible = [];
  const labels = [];
  for (const el of document.querySelectorAll(`.docBlock[data-doc="${cssEscape(kind)}"]`)) {
    if (el.checked) visible.push(el.dataset.key);
  }
  for (const el of document.querySelectorAll(`.docBlockTitle[data-doc="${cssEscape(kind)}"]`)) {
    const value = (el.value || '').trim();
    if (value) labels.push(el.dataset.key + '=' + value);
  }
  return { visible, labels };
}

/** ذخیرهٔ تنظیمات یک سند چاپی: بلوک‌ها، عنوان‌ها و متن پاراگراف. */
async function paramSaveDocument(kind) {
  const doc = (PARAM_DOCUMENTS || []).find(d => d.kind === kind);
  if (!doc) return;
  const picked = paramSelectedDocBlocks(kind);
  if (!picked.visible.length) { toast('دست‌کم یک بلوک باید روشن بماند', 'error'); return; }
  const fields = {
    scope: 'document',
    target: kind,
    visibleFields: picked.visible.join(','),
    labels: picked.labels.join(','),
    note: 'بلوک‌های سند چاپی — از پنل پارامتریک',
  };
  // متن‌های نوشتنی سند: اگر کادر همان متن پیش‌فرض مؤثر باشد خالی ذخیره می‌شود تا
  // تغییرهای بعدیِ پیش‌فرض هم به سند برسد؛ وگرنه متن نوشته‌شده ذخیره می‌شود.
  for (const el of document.querySelectorAll(`.docText[data-doc="${cssEscape(kind)}"]`)) {
    const slot = el.dataset.slot;
    const view = (doc.texts || []).find(t => t.key === slot) || {};
    const value = (el.value || '').replace(/\r\n/g, '\n').trim();
    const fallback = (view.default || '').trim();
    fields[slot + 'Text'] = value === fallback ? '' : value;
  }
  const res = doc.paramsId
    ? await apiFetch('/api/v1/settings/records/' + encodeURIComponent(doc.paramsId), {method: 'PATCH', body: {kind: 'formParams', title: 'تنظیمات سند ' + doc.titleFa, fields}})
    : await apiFetch('/api/v1/settings/records', {method: 'POST', body: {kind: 'formParams', title: 'تنظیمات سند ' + doc.titleFa, fields, noteFa: 'ثبت از پنل پارامتریک'}});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = doc.paramsId || (res.data && res.data.id);
  await apiFetch('/api/v1/settings/records/' + encodeURIComponent(id) + '/action', {method: 'POST', body: {action: 'activate'}});
  PARAM_DOCUMENTS = null;
  paramClearFormCache();
  toast('تنظیمات سند ذخیره و جاری شد');
  liveReload();
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
    <td>${f.custom ? '<span class="chip c-ok" style="font-size:.7rem">فیلد سفارشی</span>'
      : (f.source ? '<span class="chip" style="font-size:.7rem">از پروندهٔ کارمند</span>' : '<span class="muted">دستی</span>')}</td>
    <td><label style="display:inline-flex;align-items:center;gap:6px">
      <input type="checkbox" class="fpRequired" data-key="${esc(f.key)}" ${f.required ? 'checked' : ''}> اجباری</label></td>
    <td><input class="fpLabel" data-key="${esc(f.key)}" value="${esc(f.labelFa)}" style="font-size:.78rem"></td>
    <td>${f.custom ? `<button class="btn btn-sm btn-ghost" onclick="paramRemoveExtraField('${esc(form.target)}','${esc(f.key)}')">🗑 حذف فیلد</button>` : ''}</td>
  </tr>`);
  const customCount = form.fields.filter(f => f.custom).length;
  openModal({
    title: 'تنظیم فیلدهای فرم — ' + form.titleFa,
    body: `<div class="muted" style="font-size:.78rem;margin-bottom:8px">هدف: <span class="ltr">${esc(form.target)}</span> — با برداشتن تیک هر فیلد، آن فیلد از فرم حذف می‌شود و با زدن تیک برمی‌گردد. برچسب را می‌توانید فارسی‌تر کنید.</div>
      ${liveTable(['نمایش', 'منبع مقدار', 'اجباری', 'برچسب', 'حذف فیلد سفارشی'], rows, form.fields.length)}
      <div class="card-title" style="margin:14px 0 6px">➕ افزودن فیلد سفارشی (${fa(customCount)} فیلد سفارشی)</div>
      <div class="muted" style="font-size:.75rem;line-height:2;margin-bottom:8px">
        فیلد سفارشی، فیلدی است که در دامنهٔ ماژول نیست و شما به فرم اضافه می‌کنید (مثل «کد مرکز هزینه» یا «شمارهٔ حکم قبلی»).
        کلید لاتین و یکتا، برچسب فارسی و نوع فیلد را بنویسید؛ مقدار آن در همان رکورد ذخیره می‌شود. حذف فیلد سفارشی، تعریف آن را برمی‌دارد.
      </div>
      <div style="display:flex;gap:10px;flex-wrap:wrap;align-items:flex-end">
        <div class="field"><label>کلید (لاتین)</label><input class="inp ltr" id="fpNewKey" placeholder="costCenter"></div>
        <div class="field"><label>برچسب فارسی</label><input class="inp" id="fpNewLabel" placeholder="کد مرکز هزینه"></div>
        <div class="field"><label>نوع</label>
          <select id="fpNewType">
            <option value="text">متن</option><option value="number">عدد</option>
            <option value="money">مبلغ</option><option value="date">تاریخ شمسی</option>
            <option value="select">انتخابی</option>
          </select></div>
        <button class="btn btn-primary" onclick="paramAddExtraField('${esc(form.target)}')">➕ افزودن فیلد</button>
      </div>
      <div id="fpExtraMsg" style="margin-top:8px"></div>`,
    foot: `<button class="btn btn-primary" onclick="paramSaveForm('${esc(form.target)}')">💾 ذخیره و جاری‌سازی</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}

/** پیام نتیجهٔ فیلد سفارشی داخل مودال. */
function paramExtraMsg(message, isError) {
  const box = document.getElementById('fpExtraMsg');
  if (!box) return;
  box.innerHTML = isError
    ? liveErrorBox(message)
    : `<div class="card card-pad" style="font-size:.8rem;line-height:2">✅ ${esc(message)}</div>`;
}

/** افزودن فیلد سفارشی به فرم از مودال تنظیم فیلدها. */
async function paramAddExtraField(target) {
  const key = ((document.getElementById('fpNewKey') || {}).value || '').trim();
  const labelFa = ((document.getElementById('fpNewLabel') || {}).value || '').trim();
  const type = ((document.getElementById('fpNewType') || {}).value || 'text');
  if (!key || !labelFa) { toast('کلید و برچسب فیلد سفارشی الزامی است', 'error'); return; }
  const res = await apiFetch('/api/v1/settings/form-fields', {
    method: 'POST', body: {target, key, labelFa, type},
  });
  if (!res.ok) { paramExtraMsg(res.message, true); toast(res.message, 'error'); return; }
  toast((res.data && res.data.noteFa) || 'فیلد سفارشی افزوده شد');
  PARAM_FORMS = null;
  paramClearFormCache();
  closeModal();
  liveReload();
}

/** حذف فیلد سفارشی از فرم. */
async function paramRemoveExtraField(target, key) {
  const res = await apiFetch('/api/v1/settings/form-fields', {
    method: 'POST', body: {target, key, remove: true},
  });
  if (!res.ok) { paramExtraMsg(res.message, true); toast(res.message, 'error'); return; }
  toast((res.data && res.data.noteFa) || 'فیلد سفارشی حذف شد');
  PARAM_FORMS = null;
  paramClearFormCache();
  closeModal();
  liveReload();
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
  // فیلدهای سفارشی جدا از فیلدهای دامنه شمرده می‌شوند؛ اگر در فهرست نمایش نباشند، ذخیرهٔ
  // پارامتر نباید آنها را پاک کند (ذخیره، کل فیلدهای رکورد را جایگزین می‌کند).
  const customVisible = form.fields.filter(f => f.custom)
    .filter(f => {
      const box = document.querySelector(`.fpVisible[data-key="${cssEscape(f.key)}"]`);
      return !box || box.checked;
    }).map(f => f.key);
  const domainCount = form.fieldCount - form.fields.filter(f => f.custom).length;
  const domainVisible = visible.filter(k => !form.fields.some(f => f.custom && f.key === k));
  const finalVisible = (domainVisible.length === domainCount && customVisible.length === form.fields.filter(f => f.custom).length)
    ? ''
    : visible.join(',');
  const fields = {
    scope: 'form',
    target: target,
    visibleFields: finalVisible || visibleValue,
    requiredFields: required.join(','),
    labels: labels.join(','),
    // فیلدهای سفارشی دست‌نخورده می‌مانند؛ افزودن/حذف آنها از دکمه‌های همین مودال است.
    extraFields: form.fields.filter(f => f.custom).map(f => f.key + '|' + f.labelFa + '|' + f.type).join('؛'),
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
  if (!columns.length) { box.textContent = 'دست‌کم یک ستون انتخاب کنید.'; return; }
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