/* =====================================================================
   صفحات زنده — اتصال نمونهٔ رابط به API واقعی سرور Go
   =====================================================================

   این لایه عمداً «نازک» است: هیچ قاعدهٔ کسب‌وکاری اینجا نیست. تشخیص معتبر بودن،
   گردش‌کار تأیید، سرآمد، هم‌پوشانی قرارداد و دسترسی نقش همه در سرور Go اجرا می‌شوند
   و این فایل فقط داده را می‌خواند، شکل می‌دهد و اکشن را به سرور می‌فرستد. اگر سرور
   در دسترس نباشد (نمونهٔ استاتیک)، همان صفحهٔ نمایشی بخش ۵ نشان داده می‌شود تا
   ارائه بدون سرور هم ممکن بماند.

   قاعده: هر صفحهٔ «زنده» یک قالب HTML برمی‌گرداند و سپس با یک fetch، ناحیهٔ
   #liveRegion را پر می‌کند؛ پس مسیریاب همگام (renderPage) دست‌نخورده می‌ماند.
   ===================================================================== */

/** آیا نشست به سرور واقعی وصل است (توکن دارد). */
function isLive() {
  return serverOnline === true && !!(serverSession && serverSession.accessToken);
}

/** آدرس سرویس‌های چاپ سند (نوع سند → ماژول مالک در سرور بررسی می‌شود). */
const DOC_KINDS = {contract: 'contract', salary_decree: 'salary_decree', employment_certificate: 'employment_certificate', payslip: 'payslip'};

/** درخواست خام با هدر احراز هویت — برای دریافت بدنهٔ سند چاپی. */
function apiRaw(path, opts = {}) {
  const headers = Object.assign({'Accept': 'application/json'}, opts.headers || {});
  if (serverSession && serverSession.accessToken) headers['Authorization'] = 'Bearer ' + serverSession.accessToken;
  const init = {method: opts.method || 'GET', headers};
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify(opts.body);
  }
  return fetch(path, init);
}

/** درخواست JSON با هدر احراز هویت؛ پاسخ همیشه {ok, status, data, message}. */
async function apiFetch(path, opts = {}) {
  try {
    const res = await apiRaw(path, opts);
    let data = null;
    const ct = res.headers.get('content-type') || '';
    if (ct.includes('application/json')) {
      try { data = await res.json(); } catch (_) { /* بدنهٔ نامعتبر */ }
    }
    return {ok: res.ok, status: res.status, data, message: jsonMessage(res.status, data)};
  } catch (e) {
    return {ok: false, status: 0, data: null, message: 'ارتباط با سرور برقرار نشد.'};
  }
}

/** پیام فارسی از پاسخ خطای سرور (CONTRACT-API: {error:{code,message,details}}). */
function jsonMessage(status, data) {
  if (data && data.error && data.error.message) {
    const det = data.error.details;
    if (det && typeof det === 'object') {
      const first = Object.values(det)[0];
      if (first) return first;
    }
    return data.error.message;
  }
  const map = {
    400: 'درخواست نامعتبر است.', 401: 'نشست شما منقضی شده است؛ دوباره وارد شوید.',
    403: 'شما به این عملیات دسترسی ندارید.', 404: 'رکورد درخواستی یافت نشد.',
    409: 'نسخهٔ رکورد تغییر کرده؛ صفحه را بازخوانی کنید.',
    422: 'اطلاعات ارسالی معتبر نیست.', 501: 'این بخش در فاز بعدی پیاده‌سازی می‌شود.',
  };
  return map[status] || ('خطای سرور (' + status + ')');
}

/** بازخوانی صفحهٔ جاری (پس از هر اکشن موفق). */
function liveReload() { renderPage(currentPage); }

/** پیام‌ها را از پاسخ سرور به توست نشان می‌دهد؛ در موفقیت صفحه را بازخوانی می‌کند. */
async function liveSend(path, opts, okMsg) {
  const res = await apiFetch(path, opts);
  if (!res.ok) { toast(res.message, 'error'); return false; }
  toast(okMsg || 'عملیات با موفقیت انجام شد');
  return true;
}

/* ------------------------- کمک‌کننده‌های نمایشی ------------------------- */

const faDate = s => (s ? `<span class="num">${fa(s)}</span>` : '—');
const faNum = v => `<span class="num">${fa(v)}</span>`;
const personCell = (id, name) => {
  const label = name || '—';
  return `<div class="emp-cell"><div class="avatar sq">${esc(initial(label === '—' ? '؟' : label))}</div>
    <div>${esc(label)}<div class="muted ltr" style="font-size:.66rem">${esc(id || '')}</div></div></div>`;
};

/** نگاشت وضعیت قرارداد/تکفل/پرواز/حکم/گواهی به چیپ رنگی. */
const STATUS_CLS = {
  draft: 'c-wait', pending_approval: 'c-wait', active: 'c-ok', rejected: 'c-rej',
  expired: 'c-term', terminated: 'c-rej',
  pending: 'c-wait', approved: 'c-ok', removed: 'c-term',
  issued: 'c-ok', settled: 'c-ok', cancelled: 'c-term',
  notified: 'c-ok', superseded: 'c-term',
  requested: 'c-wait', delivered: 'c-ok',
};
const statusChip = (status, statusFa) =>
  pChip(STATUS_CLS[status] || 'c-wait', statusFa || status || '—');

function liveSpinner() {
  return `<div class="card card-pad" style="text-align:center;color:var(--muted)">⏳ در حال خواندن از سرور…</div>`;
}
function liveErrorBox(msg) {
  return `<div class="card card-pad" style="border-color:#fecaca;color:#b91c1c">⛔ ${esc(msg)}</div>`;
}
function liveShell(title, sub, actions) {
  return pageHead(title, sub,
    `${actions || ''}<button class="btn btn-ghost" onclick="liveReload()" title="بازخوانی">↻</button>`) +
    `<div id="liveRegion">${liveSpinner()}</div>`;
}
async function liveLoad(loader) {
  const el = () => document.getElementById('liveRegion');
  try {
    const html = await loader();
    if (el()) el().innerHTML = html;
  } catch (e) {
    if (el()) el().innerHTML = liveErrorBox(e && e.message ? e.message : String(e));
  }
}

/** حالت بدون سرور: همان صفحهٔ نمایشی بخش ۵، با یادآوری روشن. */
function offlineFallback(page) {
  const notice = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
    <b>حالت نمایشی</b> — سرور محلی در دسترس نیست، پس داده‌های نمونه نشان داده می‌شود.
    برای کار با دادهٔ واقعی، سامانه را با <span class="ltr">npm run server</span> بالا بیاورید و از حساب نمونه وارد شوید.
  </div>`;
  return notice + (typeof pageFromSpec === 'function' ? pageFromSpec(page) : '');
}

/* --------------------------- قالب جدول زنده --------------------------- */

function liveTable(head, rows, count) {
  return tbl(head, rows, {count: count == null ? rows.length : count});
}

/* =====================================================================
   قراردادهای کارکنان
   ===================================================================== */

function pageContractsLive() {
  if (!isLive()) return offlineFallback('contracts');
  liveLoad(async () => {
    const [res, opts] = await Promise.all([
      apiFetch(liveQueryURL('contracts', '/api/v1/contracts/contracts?size=200&sort=startDate:desc')),
      apiFetch('/api/v1/contracts/options'),
    ]);
    if (!res.ok) return liveErrorBox(res.message);
    const items = res.data.items || [];
    const kinds = (opts.ok && opts.data.kinds) || [];

    const active = items.filter(c => c.status === 'active').length;
    const fixed = items.filter(c => c.kind === 'fixed_term').length;
    const permanent = items.filter(c => c.kind === 'permanent').length;
    const pending = items.filter(c => c.status === 'pending_approval').length;

    const stats = `<div class="grid-stats">
      ${statCard({ico: '📜', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.length), label: 'کل قرارداد'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(active), label: 'قرارداد جاری'})}
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(fixed), label: 'مدت موقت'})}
      ${statCard({ico: '🧷', tint: '#EDE9FE', accent: '#7C3AED', val: fa(permanent), label: 'مدت دائم'})}
    </div>`;

    const rows = items.map(c => `<tr>
      <td class="num ltr">${esc(c.id)}</td>
      <td>${personCell(c.employeeId, c.employeeName)}</td>
      <td>${esc(c.departmentName || '—')}</td>
      <td>${pChip(c.kind === 'permanent' ? 'c-ok' : 'c-wait', c.kindFa || c.kind)}</td>
      <td>${c.employmentTypeFa ? `<span class="chip" style="font-size:.72rem">${esc(c.employmentTypeFa)}</span>` : '—'}</td>
      <td>${faDate(c.startDate)}</td>
      <td>${faDate(c.endDate)}</td>
      <td>${statusChip(c.status, c.statusFa)}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-ghost" title="چاپ متن قرارداد" onclick="liveOpenDoc('contract','${esc(c.id)}')">🖨️</button>
        ${(c.status === 'draft' || c.status === 'rejected') ? `<button class="btn btn-sm btn-ghost" onclick="liveContractEdit('${c.id}')">✏️ ویرایش</button>` : ''}
        ${c.status === 'draft' ? `<button class="btn btn-sm btn-ghost" onclick="liveContractDelete('${c.id}')">🗑 حذف</button>` : ''}
        ${c.status === 'draft' ? `<button class="btn btn-sm btn-primary" onclick="liveContractSubmit('${c.id}')">ارسال</button>` : ''}
        ${c.status === 'pending_approval' ? `<button class="btn btn-sm btn-primary" onclick="liveContractDecide('${c.id}',true)">تأیید</button>
          <button class="btn btn-sm btn-ghost" onclick="liveContractDecide('${c.id}',false)">رد</button>` : ''}
        ${c.status === 'active' && c.kind === 'fixed_term' ? `<button class="btn btn-sm btn-ghost" onclick="liveContractRenew('${c.id}')">تمدید</button>
          <button class="btn btn-sm btn-ghost" onclick="liveContractConvert('${c.id}')">تبدیل به دائم</button>` : ''}
        ${c.status === 'active' ? `<button class="btn btn-sm btn-ghost" onclick="liveContractTerminate('${c.id}')">خاتمه</button>` : ''}
      </div></td></tr>`);

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      ℹ️ نوع قرارداد بر پایهٔ قانون کار: <b>مدت دائم</b> و <b>مدت موقت</b>. گردش‌کار: پیش‌نویس ← ارسال ← تأیید و ابلاغ.
      ${pending ? `${fa(pending)} قرارداد در انتظار تأیید است.` : ''}
      متن قرارداد با قالب قانون کار و پارامترهای شرکت فعال چاپ می‌شود.
    </div>`;

    const types = [...new Set(items.map(c => c.employmentType).filter(Boolean))];
    const typeFilter = types.length ? `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:10px;align-items:center;flex-wrap:wrap">
      <label style="font-size:.8rem">نوع استخدام:</label>
      <select id="conTypeFilter" onchange="liveContractFilterRows()"><option value="">همه</option>
        ${types.map(t => `<option value="${esc(t)}">${esc((items.find(x => x.employmentType === t) || {}).employmentTypeFa || t)}</option>`).join('')}
      </select>
      <span class="muted" style="font-size:.75rem">فهرست قراردادها را بر پایهٔ نوع استخدام باریک کنید.</span>
    </div>` : '';

    return stats + note + typeFilter + liveTable(
      ['شماره', 'کارمند', 'واحد', 'نوع', 'نوع استخدام', 'شروع', 'پایان', 'وضعیت', 'عملیات'],
      rows, items.length);
  });
  return liveShell('قراردادهای کارکنان',
    'قراردادهای مدت موقت و مدت دائم بر پایهٔ نوع استخدام: انعقاد، تأیید، تمدید، تبدیل به دائم و چاپ متن قرارداد',
    liveSearchBar('contracts', '🔍 جستجوی شمارهٔ قرارداد، نام کارمند، کد پرسنلی یا واحد…') +
    `<button class="btn btn-primary" onclick="liveContractForm()">➕ انعقاد قرارداد تازه</button>`);
}

/** باریک کردن سطرهای جدول قراردادها بر پایهٔ نوع استخدام. */
function liveContractFilterRows() {
  const sel = $('#conTypeFilter');
  if (!sel) return;
  const want = sel.value;
  document.querySelectorAll('#liveRegion table tbody tr').forEach(tr => {
    const cell = tr.children[4];
    const chip = cell && cell.querySelector('.chip');
    const val = chip ? chip.textContent.trim() : (cell ? cell.textContent.trim() : '');
    tr.style.display = (!want || val === want || val === '—') ? '' : 'none';
  });
}

/* --------------------------- فرم‌ها و اکشن‌های قرارداد --------------------------- */

async function liveEmployeeSelect(id, selected) {
  const res = await apiFetch('/api/v1/employee/employees?size=200');
  const items = (res.ok && res.data.items) || [];
  return `<select id="${id}">${items.map(e =>
    `<option value="${esc(e.id)}" ${e.id === selected ? 'selected' : ''}>${esc(e.fullName)} — ${esc(e.id)}</option>`).join('')}</select>`;
}

async function liveContractForm() {
  const selectHtml = await liveEmployeeSelect('ctEmp', null);
  if (!RB_TYPES.length) {
    const t = await apiFetch('/api/v1/employee/options');
    RB_TYPES = (t.ok && t.data.employmentTypes) || [];
  }
  openModal({
    title: 'انعقاد قرارداد کار (قانون کار)',
    body: `<div class="form-grid">
      <div class="field"><label>کارمند <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>نوع قرارداد <span class="req">*</span></label>
        <select id="ctKind" onchange="liveContractKindToggle()">
          <option value="permanent">مدت دائم</option>
          <option value="fixed_term">مدت موقت</option>
        </select></div>
      <div class="field"><label>نوع استخدام <span class="req">*</span></label>
        <select id="ctEmploymentType">${(RB_TYPES.length ? RB_TYPES : []).map(t => `<option value="${esc(t.key)}">${esc(t.titleFa)}</option>`).join('')}</select></div>
      <div class="field"><label>تاریخ شروع (شمسی) <span class="req">*</span></label><input id="ctStart" placeholder="1404/07/01"></div>
      <div class="field" id="ctEndField" style="display:none"><label>تاریخ پایان (شمسی)</label><input id="ctEnd" placeholder="1405/06/31"></div>
      <div class="field"><label>مبلغ پایهٔ ماهانه (ریال) <span class="req">*</span></label><input id="ctSalary" type="number" min="1"></div>
      <div class="field"><label>یادداشت</label><input id="ctNote"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveContractSave()">ثبت پیش‌نویس</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
function liveContractKindToggle() {
  const el = $('#ctEndField');
  if (el) el.style.display = $('#ctKind').value === 'fixed_term' ? '' : 'none';
}
async function liveContractSave() {
  const body = {
    employeeId: $('#ctEmp').value,
    kind: $('#ctKind').value,
    employmentType: ($('#ctEmploymentType') || {}).value || '',
    startDate: $('#ctStart').value.trim(),
    endDate: $('#ctKind').value === 'fixed_term' ? $('#ctEnd').value.trim() : '',
    monthlySalary: Number($('#ctSalary').value) || 0,
    noteFa: $('#ctNote').value.trim(),
  };
  const res = await apiFetch('/api/v1/contracts/contracts', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('پیش‌نویس قرارداد ثبت شد'); liveReload();
}

/** ویرایش قرارداد — تنها پیش‌نویس یا ردشده (قاعدهٔ دامنه در سرور بازبینی می‌شود). */
async function liveContractEdit(id) {
  const res = await apiFetch('/api/v1/contracts/contracts/' + encodeURIComponent(id));
  if (!res.ok) { toast(res.message, 'error'); return; }
  const c = res.data;
  const selectHtml = await liveEmployeeSelect('ctEmp', c.employeeId);
  if (!RB_TYPES.length) {
    const t = await apiFetch('/api/v1/employee/options');
    RB_TYPES = (t.ok && t.data.employmentTypes) || [];
  }
  openModal({
    title: 'ویرایش قرارداد ' + c.id,
    body: `<div class="form-grid">
      <div class="field"><label>کارمند <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>نوع قرارداد <span class="req">*</span></label>
        <select id="ctKind" onchange="liveContractKindToggle()">
          <option value="permanent" ${c.kind === 'permanent' ? 'selected' : ''}>مدت دائم</option>
          <option value="fixed_term" ${c.kind === 'fixed_term' ? 'selected' : ''}>مدت موقت</option>
        </select></div>
      <div class="field"><label>نوع استخدام <span class="req">*</span></label>
        <select id="ctEmploymentType">${RB_TYPES.map(t => `<option value="${esc(t.key)}" ${t.key === c.employmentType ? 'selected' : ''}>${esc(t.titleFa)}</option>`).join('')}</select></div>
      <div class="field"><label>تاریخ شروع (شمسی) <span class="req">*</span></label><input id="ctStart" value="${esc(c.startDate || '')}"></div>
      <div class="field" id="ctEndField" style="display:${c.kind === 'fixed_term' ? '' : 'none'}"><label>تاریخ پایان (شمسی)</label><input id="ctEnd" value="${esc(c.endDate || '')}"></div>
      <div class="field"><label>مبلغ پایهٔ ماهانه (ریال) <span class="req">*</span></label><input id="ctSalary" type="number" min="1" value="${esc(String(c.monthlySalary || ''))}"></div>
      <div class="field"><label>یادداشت</label><input id="ctNote" value="${esc(c.noteFa || '')}"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveContractUpdate('${esc(c.id)}')">ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveContractUpdate(id) {
  const body = {
    employeeId: $('#ctEmp').value,
    kind: $('#ctKind').value,
    employmentType: ($('#ctEmploymentType') || {}).value || '',
    startDate: $('#ctStart').value.trim(),
    endDate: $('#ctKind').value === 'fixed_term' ? $('#ctEnd').value.trim() : '',
    monthlySalary: Number($('#ctSalary').value) || 0,
    noteFa: $('#ctNote').value.trim(),
  };
  const res = await apiFetch('/api/v1/contracts/contracts/' + encodeURIComponent(id), {method: 'PATCH', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('قرارداد ویرایش شد'); liveReload();
}
/** حذف پیش‌نویس قرارداد — قاعدهٔ دامنه در سرور اعمال می‌شود. */
function liveContractDelete(id) {
  if (!confirm('این پیش‌نویس قرارداد حذف شود؟')) return;
  liveSend('/api/v1/contracts/contracts/' + encodeURIComponent(id), {method: 'DELETE'}, 'قرارداد حذف شد')
    .then(ok => ok && liveReload());
}
function liveContractSubmit(id) {
  liveSend(`/api/v1/contracts/contracts/${id}/submit`, {method: 'POST'}, 'قرارداد برای تأیید ارسال شد')
    .then(ok => ok && liveReload());
}
function liveContractDecide(id, approve) {
  const note = approve ? '' : prompt('دلیل رد قرارداد:');
  if (!approve && !note) return;
  liveSend(`/api/v1/contracts/contracts/${id}/decision`, {method: 'POST', body: {approve, noteFa: note || ''}},
    approve ? 'قرارداد تأیید و ابلاغ شد' : 'قرارداد رد شد').then(ok => ok && liveReload());
}
function liveContractTerminate(id) {
  const note = prompt('دلیل خاتمهٔ قرارداد (پایان همکاری):');
  if (!note) return;
  liveSend(`/api/v1/contracts/contracts/${id}/terminate`, {method: 'POST', body: {approve: false, noteFa: note}},
    'قرارداد خاتمه یافت').then(ok => ok && liveReload());
}
async function liveContractRenew(id) {
  const end = prompt('تاریخ پایان دورهٔ تمدید (شمسی، خالی = همان طول قبلی):') || '';
  const salaryStr = prompt('مبلغ پایهٔ ماهانهٔ جدید (ریال، خالی = بدون تغییر):') || '';
  const res = await apiFetch(`/api/v1/contracts/contracts/${id}/renew`, {
    method: 'POST', body: {endDate: end.trim(), monthlySalary: Number(salaryStr) || 0},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('پیش‌نویس تمدید ثبت شد و در انتظار تأیید است'); liveReload();
}
async function liveContractConvert(id) {
  const salaryStr = prompt('مبلغ پایهٔ ماهانهٔ قرارداد دائم (ریال، خالی = بدون تغییر):') || '';
  const res = await apiFetch(`/api/v1/contracts/contracts/${id}/convert`, {
    method: 'POST', body: {monthlySalary: Number(salaryStr) || 0},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('درخواست تبدیل به مدت دائم ثبت شد و در انتظار تأیید است'); liveReload();
}

/* =====================================================================
   تمدید، تغییر و پایان قرارداد
   ===================================================================== */

function pageContractLifecycleLive() {
  if (!isLive()) return offlineFallback('contractLifecycle');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/contracts/renewals?days=180');
    if (!res.ok) return liveErrorBox(res.message);
    const items = res.data.items || [];

    const stats = `<div class="grid-stats">
      ${statCard({ico: '🔔', tint: '#FEF3C7', accent: '#D97706', val: fa(items.length), label: 'سرآمد در ۱۸۰ روز آینده'})}
      ${statCard({ico: '🧷', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.filter(c => c.kind === 'fixed_term').length), label: 'قرارداد مدت موقت'})}
      ${statCard({ico: '📆', tint: '#DCFCE7', accent: '#16A34A', val: fa(items.filter(c => (c.serviceDays || 0) >= 365).length), label: 'واجد شرط تبدیل (۱ سال سابقه)'})}
      ${statCard({ico: '⏳', tint: '#EDE9FE', accent: '#7C3AED', val: fa(items.filter(c => c.renewCount > 0).length), label: 'سابقهٔ تمدیدشده'})}
    </div>`;

    const rows = items.map(c => `<tr>
      <td>${personCell(c.employeeId, c.employeeName)}</td>
      <td>${pChip(c.kind === 'permanent' ? 'c-ok' : 'c-wait', c.kindFa || c.kind)}</td>
      <td>${faDate(c.endDate)}</td>
      <td>${faNum((c.serviceDays || 0) + ' روز')}</td>
      <td>${faNum(c.renewCount || 0)}</td>
      <td>${statusChip(c.status, c.statusFa)}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-ghost" title="چاپ" onclick="liveOpenDoc('contract','${esc(c.id)}')">🖨️</button>
        <button class="btn btn-sm btn-primary" onclick="liveContractRenew('${c.id}')">تمدید</button>
        ${(c.serviceDays || 0) >= 365 ? `<button class="btn btn-sm btn-ghost" onclick="liveContractConvert('${c.id}')">تبدیل به دائم</button>` : ''}
        <button class="btn btn-sm btn-ghost" onclick="liveContractTerminate('${c.id}')">پایان همکاری</button>
      </div></td></tr>`);

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      ℹ️ «سرآمد» تاریخ پایان قرارداد مدت موقت است. تمدید یک قرارداد تازه می‌سازد و پس از تأیید، قرارداد پیشین را می‌بندد؛
      تبدیل به مدت دائم نیازمند دست‌کم یک سال سابقهٔ خدمت است.
    </div>`;

    return stats + note + liveTable(
      ['کارمند', 'نوع کنونی', 'سرآمد', 'سابقهٔ خدمت', 'تعداد تمدید', 'وضعیت', 'عملیات'],
      rows, items.length);
  });
  return liveShell('تمدید، تغییر و پایان قرارداد',
    'پیگیری سرآمدهای قرارداد و اقدام‌های تمدید، تبدیل به دائم و پایان همکاری', '');
}

/* =====================================================================
   قالب و متن قراردادها
   ===================================================================== */

function pageContractTemplatesLive() {
  if (!isLive()) return offlineFallback('contractTemplates');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/contracts/templates');
    const note = `<div class="card card-pad" style="font-size:.85rem;line-height:2">
      ℹ️ قالب‌های قرارداد در سرور نگه داشته می‌شوند تا متن نهایی (بندهای قانون کار) از یک مرجع واحد بیاید.
      ${res.ok ? '' : esc(res.message)}
    </div>`;
    return note;
  });
  return liveShell('قالب و متن قراردادها',
    'نگهداری قالب‌های متن قرارداد کار و پیوست‌های قانونی هر نوع قرارداد', '');
}

/* =====================================================================
   افراد تحت تکفل
   ===================================================================== */

const RELATION_FA = {spouse: 'همسر', child: 'فرزند', student_child: 'فرزند (دانشجو)', parent: 'والد'};

function pageDependentsLive() {
  if (!isLive()) return offlineFallback('dependents');
  liveLoad(async () => {
    const res = await apiFetch(liveQueryURL('dependents', '/api/v1/personnel/dependents?size=200'));
    if (!res.ok) return liveErrorBox(res.message);
    const items = res.data.items || [];

    const stats = `<div class="grid-stats">
      ${statCard({ico: '👨‍👩‍👧', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.length), label: 'فرد تحت تکفل'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(items.filter(d => d.status === 'approved').length), label: 'تکفل تأییدشده'})}
      ${statCard({ico: '🩺', tint: '#FEF3C7', accent: '#D97706', val: fa(items.filter(d => d.insurance).length), label: 'دارای پوشش بیمه'})}
      ${statCard({ico: '⏳', tint: '#EDE9FE', accent: '#7C3AED', val: fa(items.filter(d => d.status === 'pending').length), label: 'در انتظار بررسی'})}
    </div>`;

    const rows = items.map(d => `<tr>
      <td class="num ltr">${esc(d.employeeId)}</td>
      <td>${personCell(d.employeeId, d.employeeName)}</td>
      <td>${esc(d.fullName)}</td>
      <td>${esc(d.relationFa || RELATION_FA[d.relation] || d.relation)}</td>
      <td>${faDate(d.coverageStart)}</td>
      <td>${d.insurance ? pChip('c-ok', d.insuranceFa || 'دارد') : pChip('c-wait', 'ندارد')}</td>
      <td>${statusChip(d.status, d.statusFa)}</td>
      <td><div class="row-actions">
        ${d.status !== 'removed' ? `<button class="btn btn-sm btn-ghost" onclick="liveDependentEdit('${d.id}')">✏️ ویرایش</button>` : ''}
        ${d.status === 'pending' ? `<button class="btn btn-sm btn-primary" onclick="liveDependentDecide('${d.id}',true)">تأیید</button>
          <button class="btn btn-sm btn-ghost" onclick="liveDependentDecide('${d.id}',false)">رد</button>` : ''}
        ${d.status === 'rejected' ? `<button class="btn btn-sm btn-ghost" onclick="liveDependentDelete('${d.id}')">🗑 حذف</button>` : ''}
        ${d.status !== 'removed' ? `<button class="btn btn-sm btn-ghost" onclick="liveDependentRemove('${d.id}')">خروج تکفل</button>` : ''}
      </div></td></tr>`);

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      ℹ️ تأیید تکفل نیازمند ثبت مدرک است و رد یا خروج آن ثبت دلیل می‌خواهد؛ کد ملی تکراری برای یک کارمند پذیرفته نمی‌شود.
    </div>`;

    return stats + note + liveTable(
      ['کد پرسنلی', 'کارمند', 'فرد تحت تکفل', 'نسبت', 'تاریخ تکفل', 'پوشش بیمه', 'وضعیت', 'عملیات'],
      rows, items.length);
  });
  return liveShell('افراد تحت تکفل',
    'ثبت همسر و فرزندان کارکنان، مدارک تکفل و پوشش بیمه و خدمات درمانی',
    liveSearchBar('dependents', '🔍 جستجوی نام فرد تحت تکفل، کد پرسنلی، کارمند یا نسبت…') +
    `<button class="btn btn-primary" onclick="liveDependentForm()">➕ ثبت فرد تحت تکفل</button>`);
}

async function liveDependentForm() {
  const selectHtml = await liveEmployeeSelect('depEmp', null);
  openModal({
    title: 'ثبت فرد تحت تکفل',
    body: `<div class="form-grid">
      <div class="field"><label>کارمند <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>نام و نام خانوادگی <span class="req">*</span></label><input id="depName"></div>
      <div class="field"><label>نسبت <span class="req">*</span></label>
        <select id="depRel">
          <option value="spouse">همسر</option><option value="child">فرزند</option>
          <option value="student_child">فرزند (دانشجو)</option><option value="parent">والد</option>
        </select></div>
      <div class="field"><label>کد ملی</label><input id="depNid" placeholder="۱۰ رقم"></div>
      <div class="field"><label>تاریخ تکفل (شمسی)</label><input id="depStart" placeholder="1404/07/01"></div>
      <div class="field"><label>پوشش بیمه</label><select id="depIns"><option value="1">دارد</option><option value="0">ندارد</option></select></div>
      <div class="field"><label>مدارک تکفل</label><input id="depDocs" placeholder="مثلاً: شناسنامه، عقدنامه"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveDependentSave()">ثبت</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveDependentSave() {
  const body = {
    employeeId: $('#depEmp').value,
    fullName: $('#depName').value.trim(),
    relation: $('#depRel').value,
    nationalId: $('#depNid').value.trim(),
    coverageStart: $('#depStart').value.trim(),
    insurance: $('#depIns').value === '1',
    documentsFa: $('#depDocs').value.trim(),
  };
  const res = await apiFetch('/api/v1/personnel/dependents', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('فرد تحت تکفل ثبت شد و در انتظار بررسی است'); liveReload();
}

/** ویرایش تکفل (وضعیت خروج‌شده قابل ویرایش نیست — قاعدهٔ سرور). */
async function liveDependentEdit(id) {
  const res = await apiFetch('/api/v1/personnel/dependents/' + encodeURIComponent(id));
  if (!res.ok) { toast(res.message, 'error'); return; }
  const d = res.data;
  const selectHtml = await liveEmployeeSelect('depEmp', d.employeeId);
  const relOpts = Object.entries(RELATION_FA)
    .map(([v, l]) => `<option value="${v}" ${v === d.relation ? 'selected' : ''}>${esc(l)}</option>`).join('');
  openModal({
    title: 'ویرایش فرد تحت تکفل ' + d.id,
    body: `<div class="form-grid">
      <div class="field"><label>کارمند <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>نام و نام خانوادگی <span class="req">*</span></label><input id="depName" value="${esc(d.fullName || '')}"></div>
      <div class="field"><label>نسبت <span class="req">*</span></label><select id="depRel">${relOpts}</select></div>
      <div class="field"><label>کد ملی</label><input id="depNid" value="${esc(d.nationalId || '')}"></div>
      <div class="field"><label>تاریخ تکفل (شمسی)</label><input id="depStart" value="${esc(d.coverageStart || '')}"></div>
      <div class="field"><label>پوشش بیمه</label><select id="depIns"><option value="1" ${d.insurance ? 'selected' : ''}>دارد</option><option value="0" ${d.insurance ? '' : 'selected'}>ندارد</option></select></div>
      <div class="field"><label>مدارک تکفل</label><input id="depDocs" value="${esc(d.documentsFa || '')}"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveDependentUpdate('${esc(d.id)}')">ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveDependentUpdate(id) {
  const body = {
    employeeId: $('#depEmp').value,
    fullName: $('#depName').value.trim(),
    relation: $('#depRel').value,
    nationalId: $('#depNid').value.trim(),
    coverageStart: $('#depStart').value.trim(),
    insurance: $('#depIns').value === '1',
    documentsFa: $('#depDocs').value.trim(),
  };
  const res = await apiFetch('/api/v1/personnel/dependents/' + encodeURIComponent(id), {method: 'PATCH', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('تکفل ویرایش شد'); liveReload();
}
/** حذف تکفل ردشده — قاعدهٔ دامنه در سرور بازبینی می‌شود. */
function liveDependentDelete(id) {
  if (!confirm('این تکفل ردشده حذف شود؟')) return;
  liveSend('/api/v1/personnel/dependents/' + encodeURIComponent(id), {method: 'DELETE'}, 'تکفل حذف شد')
    .then(ok => ok && liveReload());
}
function liveDependentDecide(id, approve) {
  const note = approve ? '' : prompt('دلیل رد تکفل:');
  if (!approve && !note) return;
  liveSend(`/api/v1/personnel/dependents/${id}/decision`, {method: 'POST', body: {approve, noteFa: note || ''}},
    approve ? 'تکفل تأیید شد' : 'تکفل رد شد').then(ok => ok && liveReload());
}
function liveDependentRemove(id) {
  const note = prompt('دلیل خروج تکفل:');
  if (!note) return;
  liveSend(`/api/v1/personnel/dependents/${id}/remove`, {method: 'POST', body: {noteFa: note}},
    'خروج تکفل ثبت شد').then(ok => ok && liveReload());
}

/* =====================================================================
   فرم درخواست پرواز
   ===================================================================== */

const PURPOSE_FA = {mission: 'ماموریت', leave: 'مرخصی', family: 'سفر خانوادگی'};

function pageFlightRequestsLive() {
  if (!isLive()) return offlineFallback('flightRequests');
  liveLoad(async () => {
    const res = await apiFetch(liveQueryURL('flightRequests', '/api/v1/personnel/flight-requests?size=200'));
    if (!res.ok) return liveErrorBox(res.message);
    const items = res.data.items || [];

    const stats = `<div class="grid-stats">
      ${statCard({ico: '✈️', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.length), label: 'درخواست پرواز'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(items.filter(f => f.status === 'issued' || f.status === 'settled').length), label: 'بلیت صادرشده'})}
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(items.filter(f => f.status === 'pending').length), label: 'در انتظار تأیید'})}
      ${statCard({ico: '💰', tint: '#EDE9FE', accent: '#7C3AED', val: money(items.reduce((s, f) => s + (f.estimatedCost || 0), 0)), label: 'جمع برآورد هزینه (ریال)'})}
    </div>`;

    const rows = items.map(f => `<tr>
      <td class="num ltr">${esc(f.id)}</td>
      <td>${personCell(f.employeeId, f.employeeName)}</td>
      <td>${esc(f.originFa)} ← ${esc(f.destinationFa)}</td>
      <td>${faDate(f.departDate)}</td>
      <td>${esc(f.purposeFa || PURPOSE_FA[f.purpose] || f.purpose)}</td>
      <td>${faNum(money(f.estimatedCost || 0))}</td>
      <td>${statusChip(f.status, f.statusFa)}</td>
      <td><div class="row-actions">
        ${f.status === 'pending' ? `<button class="btn btn-sm btn-ghost" onclick="liveFlightEdit('${f.id}')">✏️ ویرایش</button>` : ''}
        ${f.status === 'pending' ? `<button class="btn btn-sm btn-primary" onclick="liveFlightDecide('${f.id}','approve')">تأیید</button>
          <button class="btn btn-sm btn-ghost" onclick="liveFlightDecide('${f.id}','reject')">رد</button>` : ''}
        ${f.status === 'approved' ? `<button class="btn btn-sm btn-primary" onclick="liveFlightIssue('${f.id}')">صدور بلیت</button>` : ''}
        ${f.status === 'issued' ? `<button class="btn btn-sm btn-ghost" onclick="liveFlightDecide('${f.id}','settle')">تسویه</button>` : ''}
        ${(f.status === 'pending' || f.status === 'approved') ? `<button class="btn btn-sm btn-ghost" onclick="liveFlightCancel('${f.id}')">لغو</button>` : ''}
      </div></td></tr>`);

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      ℹ️ گردش‌کار پرواز: درخواست ← تأیید مدیر ← صدور بلیت (نیازمند شمارهٔ بلیت) ← تسویه. تاریخ پرواز نمی‌تواند در گذشته باشد
      و تاریخ بازگشت باید بعد از تاریخ رفت باشد.
    </div>`;

    return stats + note + liveTable(
      ['شماره', 'درخواست‌کننده', 'مسیر', 'تاریخ پرواز', 'هدف', 'برآورد هزینه', 'وضعیت', 'عملیات'],
      rows, items.length);
  });
  return liveShell('فرم درخواست پرواز',
    'ثبت درخواست پرواز کارکنان، تأیید مدیر و پیگیری صدور بلیت',
    liveSearchBar('flightRequests', '🔍 جستجوی شمارهٔ درخواست، نام کارمند، مبدأ یا مقصد…') +
    `<button class="btn btn-primary" onclick="liveFlightForm()">➕ ثبت درخواست پرواز</button>`);
}

async function liveFlightForm() {
  const selectHtml = await liveEmployeeSelect('flEmp', null);
  openModal({
    title: 'ثبت فرم درخواست پرواز',
    body: `<div class="form-grid">
      <div class="field"><label>درخواست‌کننده <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>مبدأ <span class="req">*</span></label><input id="flOrg" value="ماهشهر"></div>
      <div class="field"><label>مقصد <span class="req">*</span></label><input id="flDest" placeholder="تهران"></div>
      <div class="field"><label>تاریخ پرواز (شمسی) <span class="req">*</span></label><input id="flDepart" placeholder="1404/08/03"></div>
      <div class="field"><label>تاریخ بازگشت (شمسی)</label><input id="flReturn"></div>
      <div class="field"><label>هدف سفر</label>
        <select id="flPurpose"><option value="mission">ماموریت</option><option value="leave">مرخصی</option><option value="family">سفر خانوادگی</option></select></div>
      <div class="field"><label>تعداد مسافر</label><input id="flPax" type="number" min="1" value="1"></div>
      <div class="field"><label>برآورد هزینه (ریال)</label><input id="flCost" type="number" min="0"></div>
      <div class="field"><label>همراهان</label><input id="flComp" placeholder="نام همراهان تحت تکفل"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveFlightSave()">ثبت</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveFlightSave() {
  const body = {
    employeeId: $('#flEmp').value,
    originFa: $('#flOrg').value.trim(),
    destinationFa: $('#flDest').value.trim(),
    departDate: $('#flDepart').value.trim(),
    returnDate: $('#flReturn').value.trim(),
    purpose: $('#flPurpose').value,
    passengers: Number($('#flPax').value) || 1,
    estimatedCost: Number($('#flCost').value) || 0,
    companionsFa: $('#flComp').value.trim(),
  };
  const res = await apiFetch('/api/v1/personnel/flight-requests', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('درخواست پرواز ثبت شد'); liveReload();
}

/** ویرایش درخواست پرواز — تنها تا پیش از تصمیم مدیر (قاعدهٔ دامنه). */
async function liveFlightEdit(id) {
  const res = await apiFetch('/api/v1/personnel/flight-requests/' + encodeURIComponent(id));
  if (!res.ok) { toast(res.message, 'error'); return; }
  const f = res.data;
  const selectHtml = await liveEmployeeSelect('flEmp', f.employeeId);
  const purposeOpts = Object.entries(PURPOSE_FA)
    .map(([v, l]) => `<option value="${v}" ${v === f.purpose ? 'selected' : ''}>${esc(l)}</option>`).join('');
  openModal({
    title: 'ویرایش درخواست پرواز ' + f.id,
    body: `<div class="form-grid">
      <div class="field"><label>درخواست‌کننده <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>مبدأ <span class="req">*</span></label><input id="flOrg" value="${esc(f.originFa || '')}"></div>
      <div class="field"><label>مقصد <span class="req">*</span></label><input id="flDest" value="${esc(f.destinationFa || '')}"></div>
      <div class="field"><label>تاریخ پرواز (شمسی) <span class="req">*</span></label><input id="flDepart" value="${esc(f.departDate || '')}"></div>
      <div class="field"><label>تاریخ بازگشت (شمسی)</label><input id="flReturn" value="${esc(f.returnDate || '')}"></div>
      <div class="field"><label>هدف سفر</label><select id="flPurpose">${purposeOpts}</select></div>
      <div class="field"><label>تعداد مسافر</label><input id="flPax" type="number" min="1" value="${esc(String(f.passengers || 1))}"></div>
      <div class="field"><label>برآورد هزینه (ریال)</label><input id="flCost" type="number" min="0" value="${esc(String(f.estimatedCost || 0))}"></div>
      <div class="field"><label>همراهان</label><input id="flComp" value="${esc(f.companionsFa || '')}"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveFlightUpdate('${esc(f.id)}')">ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveFlightUpdate(id) {
  const body = {
    employeeId: $('#flEmp').value,
    originFa: $('#flOrg').value.trim(),
    destinationFa: $('#flDest').value.trim(),
    departDate: $('#flDepart').value.trim(),
    returnDate: $('#flReturn').value.trim(),
    purpose: $('#flPurpose').value,
    passengers: Number($('#flPax').value) || 1,
    estimatedCost: Number($('#flCost').value) || 0,
    companionsFa: $('#flComp').value.trim(),
  };
  const res = await apiFetch('/api/v1/personnel/flight-requests/' + encodeURIComponent(id), {method: 'PATCH', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('درخواست پرواز ویرایش شد'); liveReload();
}
function liveFlightDecide(id, action) {
  const note = action === 'reject' ? prompt('دلیل رد درخواست پرواز:') : '';
  if (action === 'reject' && !note) return;
  liveSend(`/api/v1/personnel/flight-requests/${id}/decision`, {method: 'POST', body: {action, noteFa: note || ''}},
    action === 'reject' ? 'درخواست رد شد' : 'درخواست تأیید شد').then(ok => ok && liveReload());
}
function liveFlightIssue(id) {
  const ticketNo = prompt('شمارهٔ بلیت صادرشده:');
  if (!ticketNo) return;
  liveSend(`/api/v1/personnel/flight-requests/${id}/decision`, {method: 'POST', body: {action: 'issue', ticketNo}},
    'بلیت صادر شد').then(ok => ok && liveReload());
}
function liveFlightCancel(id) {
  const note = prompt('دلیل لغو درخواست:');
  if (!note) return;
  liveSend(`/api/v1/personnel/flight-requests/${id}/decision`, {method: 'POST', body: {action: 'cancel', noteFa: note}},
    'درخواست لغو شد').then(ok => ok && liveReload());
}

/* =====================================================================
   احکام حقوق
   ===================================================================== */

function pageSalaryDecreesLive() {
  if (!isLive()) return offlineFallback('salaryDecrees');
  liveLoad(async () => {
    const res = await apiFetch(liveQueryURL('salaryDecrees', '/api/v1/salary_admin/decrees?size=200'));
    if (!res.ok) return liveErrorBox(res.message);
    const items = res.data.items || [];

    const stats = `<div class="grid-stats">
      ${statCard({ico: '📜', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.filter(d => d.status === 'notified').length), label: 'حکم ابلاغ‌شده'})}
      ${statCard({ico: '🖊️', tint: '#FEF3C7', accent: '#D97706', val: fa(items.filter(d => d.status === 'draft').length), label: 'پیش‌نویس در انتظار صدور'})}
      ${statCard({ico: '📤', tint: '#DCFCE7', accent: '#16A34A', val: fa(items.filter(d => d.status === 'issued').length), label: 'صادرشده در انتظار ابلاغ'})}
      ${statCard({ico: '🗂️', tint: '#EDE9FE', accent: '#7C3AED', val: fa(items.length), label: 'کل احکام'})}
    </div>`;

    const rows = items.map(d => `<tr>
      <td class="num ltr">${esc(d.id)}</td>
      <td>${personCell(d.employeeId, d.employeeName)}</td>
      <td>${d.employmentTypeFa ? `<span class="chip" style="font-size:.72rem">${esc(d.employmentTypeFa)}</span>` : '—'}</td>
      <td>${esc(d.jobGroup || '—')}</td>
      <td>${faNum(money(d.baseSalary || 0))}</td>
      <td>${faDate(d.effectiveDate)}</td>
      <td>${esc(d.reasonFa || '—')}</td>
      <td>${statusChip(d.status, d.statusFa)}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-ghost" title="چاپ حکم حقوق" onclick="liveOpenDoc('salary_decree','${esc(d.employeeId)}')">🖨️</button>
        ${d.status === 'draft' ? `<button class="btn btn-sm btn-ghost" onclick="liveDecreeEdit('${d.id}')">✏️ ویرایش</button>` : ''}
        ${d.status === 'draft' ? `<button class="btn btn-sm btn-ghost" onclick="liveDecreeDelete('${d.id}')">🗑 حذف</button>` : ''}
        ${d.status === 'draft' ? `<button class="btn btn-sm btn-primary" onclick="liveDecreeIssue('${d.id}')">صدور</button>` : ''}
        ${d.status === 'issued' ? `<button class="btn btn-sm btn-primary" onclick="liveDecreeNotify('${d.id}')">ابلاغ</button>` : ''}
        ${d.status === 'notified' ? `<button class="btn btn-sm btn-ghost" onclick="liveDecreeRevise('${d.id}')">اصلاح</button>` : ''}
      </div></td></tr>`);

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      ℹ️ گردش‌کار حکم: پیش‌نویس ← صدور ← ابلاغ. «اصلاح» یک حکم تازه می‌سازد و حکم پیشین را منسوخ می‌کند؛
      احکام گذشته بازنویسی نمی‌شوند. «صدور گروهی» بر پایهٔ نوع استخدام، برای همهٔ کارکنان آن نوع حکم می‌سازد.
    </div>`;

    const types = [...new Set(items.map(d => d.employmentType).filter(Boolean))];
    const typeFilter = types.length ? `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:10px;align-items:center;flex-wrap:wrap">
      <label style="font-size:.8rem">نوع استخدام:</label>
      <select id="decTypeFilter" onchange="liveDecreeFilterRows()"><option value="">همه</option>
        ${types.map(t => `<option value="${esc(t)}">${esc((items.find(x => x.employmentType === t) || {}).employmentTypeFa || t)}</option>`).join('')}
      </select>
      <span class="muted" style="font-size:.75rem">فهرست را بر پایهٔ نوع استخدام باریک کنید.</span>
    </div>` : '';

    return stats + note + typeFilter + liveTable(
      ['شمارهٔ حکم', 'کارمند', 'نوع استخدام', 'گروه شغلی', 'پایه (ریال)', 'تاریخ اجرا', 'علت صدور', 'وضعیت', 'عملیات'],
      rows, items.length);
  });
  return liveShell('احکام حقوق',
    'صدور تکی و گروهی، اصلاح و ابلاغ حکم کارگزینی بر پایهٔ نوع استخدام و پارامترهای شرکت',
    liveSearchBar('salaryDecrees', '🔍 جستجوی شمارهٔ حکم، نام کارمند، کد پرسنلی یا گروه شغلی…') +
    `<button class="btn btn-primary" onclick="liveDecreeForm()">➕ صدور حکم تکی</button>
     <button class="btn btn-ghost" onclick="liveDecreeGroupForm()">👥 صدور گروهی</button>`);
}

async function liveDecreeForm() {
  const selectHtml = await liveEmployeeSelect('decEmp', null);
  openModal({
    title: 'صدور حکم حقوق',
    body: `<div class="form-grid">
      <div class="field"><label>کارمند <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>گروه شغلی <span class="req">*</span></label><input id="decGroup" placeholder="مثلاً: کارشناس ارشد"></div>
      <div class="field"><label>مبلغ پایهٔ ماهانه (ریال) <span class="req">*</span></label><input id="decSalary" type="number" min="1"></div>
      <div class="field"><label>تاریخ اجرا (شمسی) <span class="req">*</span></label><input id="decEffective" placeholder="1404/07/01"></div>
      <div class="field"><label>علت صدور <span class="req">*</span></label><input id="decReason" placeholder="مثلاً: افزایش سنواتی"></div>
      <div class="field"><label>یادداشت</label><input id="decNote"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveDecreeSave()">ثبت پیش‌نویس</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveDecreeSave() {
  const body = {
    employeeId: $('#decEmp').value,
    jobGroup: $('#decGroup').value.trim(),
    baseSalary: Number($('#decSalary').value) || 0,
    effectiveDate: $('#decEffective').value.trim(),
    reasonFa: $('#decReason').value.trim(),
    noteFa: $('#decNote').value.trim(),
  };
  const res = await apiFetch('/api/v1/salary_admin/decrees', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('پیش‌نویس حکم حقوق ثبت شد'); liveReload();
}

/** ویرایش پیش‌نویس حکم حقوق — حکم صادرشده/ابلاغ‌شده فقط از راه «اصلاح» تغییر می‌کند. */
async function liveDecreeEdit(id) {
  const res = await apiFetch('/api/v1/salary_admin/decrees/' + encodeURIComponent(id));
  if (!res.ok) { toast(res.message, 'error'); return; }
  const d = res.data;
  const selectHtml = await liveEmployeeSelect('decEmp', d.employeeId);
  openModal({
    title: 'ویرایش پیش‌نویس حکم ' + d.id,
    body: `<div class="form-grid">
      <div class="field"><label>کارمند <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>گروه شغلی <span class="req">*</span></label><input id="decGroup" value="${esc(d.jobGroup || '')}"></div>
      <div class="field"><label>مبلغ پایهٔ ماهانه (ریال) <span class="req">*</span></label><input id="decSalary" type="number" min="1" value="${esc(String(d.baseSalary || ''))}"></div>
      <div class="field"><label>تاریخ اجرا (شمسی) <span class="req">*</span></label><input id="decEffective" value="${esc(d.effectiveDate || '')}"></div>
      <div class="field"><label>علت صدور <span class="req">*</span></label><input id="decReason" value="${esc(d.reasonFa || '')}"></div>
      <div class="field"><label>یادداشت</label><input id="decNote" value="${esc(d.noteFa || '')}"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveDecreeUpdate('${esc(d.id)}')">ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveDecreeUpdate(id) {
  const body = {
    employeeId: $('#decEmp').value,
    jobGroup: $('#decGroup').value.trim(),
    baseSalary: Number($('#decSalary').value) || 0,
    effectiveDate: $('#decEffective').value.trim(),
    reasonFa: $('#decReason').value.trim(),
    noteFa: $('#decNote').value.trim(),
  };
  const res = await apiFetch('/api/v1/salary_admin/decrees/' + encodeURIComponent(id), {method: 'PATCH', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('پیش‌نویس حکم ویرایش شد'); liveReload();
}
/** حذف پیش‌نویس حکم — قاعدهٔ دامنه در سرور اعمال می‌شود. */
function liveDecreeDelete(id) {
  if (!confirm('این پیش‌نویس حکم حذف شود؟')) return;
  liveSend('/api/v1/salary_admin/decrees/' + encodeURIComponent(id), {method: 'DELETE'}, 'پیش‌نویس حکم حذف شد')
    .then(ok => ok && liveReload());
}
/** باریک کردن سطرهای جدول احکام بر پایهٔ نوع استخدام (بدون رفت‌وبرگشت به سرور). */
function liveDecreeFilterRows() {
  const sel = $('#decTypeFilter');
  if (!sel) return;
  const want = sel.value;
  document.querySelectorAll('#liveBody tbody tr').forEach(tr => {
    const cell = tr.children[2];
    const txt = cell ? cell.textContent.trim() : '';
    const chip = cell && cell.querySelector('.chip');
    const val = chip ? chip.textContent.trim() : txt;
    tr.style.display = (!want || val === want || val === '—') ? '' : 'none';
  });
}

/* ---------------------------------------------------------------------
   صدور گروهی حکم کارگزینی
   --------------------------------------------------------------------- */

let DEC_TYPES = [];

/** فرم صدور گروهی: انتخاب نوع استخدام، درصد/مبلغ افزایش و پیش‌نمایش. */
async function liveDecreeGroupForm() {
  if (!DEC_TYPES.length) {
    const res = await apiFetch('/api/v1/salary_admin/options');
    DEC_TYPES = (res.ok && res.data.employmentTypes) || [];
  }
  const opts = DEC_TYPES.map(t =>
    `<option value="${esc(t.key)}">${esc(t.titleFa)} — ${esc(t.groupFa)}</option>`).join('');
  openModal({
    title: 'صدور گروهی حکم کارگزینی',
    body: `<div class="card card-pad" style="margin-bottom:10px;font-size:.8rem;line-height:2">
        برای همهٔ کارکنان فعال یک نوع استخدام حکم ساخته می‌شود. اول «پیش‌نمایش» بگیرید و بعد «صدور» را بزنید.
      </div>
      <div class="form-grid">
        <div class="field"><label>نوع استخدام <span class="req">*</span></label><select id="grpType">${opts}</select></div>
        <div class="field"><label>افزایش (٪)</label><input id="grpPercent" type="number" step="0.01" value="18"></div>
        <div class="field"><label>یا افزایش ثابت (ریال)</label><input id="grpAmount" type="number" value="0"></div>
        <div class="field"><label>تاریخ اجرا (شمسی) <span class="req">*</span></label><input id="grpEffective" placeholder="1405/01/01"></div>
        <div class="field"><label>علت صدور <span class="req">*</span></label><input id="grpReason" value="افزایش سالانه ۱۴۰۵"></div>
        <div class="field"><label>یادداشت</label><input id="grpNote"></div>
      </div>
      <div id="grpPreview" style="margin-top:12px"></div>`,
    foot: `<button class="btn btn-ghost" onclick="liveDecreeGroupRun(true)">👁️ پیش‌نمایش</button>
           <button class="btn btn-primary" onclick="liveDecreeGroupRun(false)">👥 صدور گروهی</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}

/** اجرای صدور گروهی؛ dryRun=true فقط پیش‌نمایش است. */
async function liveDecreeGroupRun(dryRun) {
  const body = {
    employmentType: ($('#grpType') || {}).value || '',
    increasePercent: Number(($('#grpPercent') || {}).value) || 0,
    increaseAmount: Number(($('#grpAmount') || {}).value) || 0,
    effectiveDate: (($('#grpEffective') || {}).value || '').trim(),
    reasonFa: (($('#grpReason') || {}).value || '').trim(),
    noteFa: (($('#grpNote') || {}).value || '').trim(),
    dryRun: !!dryRun,
  };
  const res = await apiFetch('/api/v1/salary_admin/decrees/group', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const d = res.data;
  const rows = (d.rows || []).map(r => `<tr>
    <td>${personCell(r.employeeId, r.employeeName)}</td>
    <td>${esc(r.employmentTypeFa || '—')}</td>
    <td class="num">${money(r.oldSalary)}</td>
    <td class="num">${money(r.newSalary)}</td>
    <td class="num" style="color:#16A34A">+${money(r.delta)}</td>
    <td>${esc(r.decreeId || '—')}</td></tr>`);
  const box = $('#grpPreview');
  if (box) {
    box.innerHTML = `<div class="grid-stats" style="margin-bottom:8px">
        ${statCard({ico: '👥', tint: '#EFF6FF', accent: '#2563EB', val: fa((d.rows || []).length), label: dryRun ? 'کارمند هدف (پیش‌نمایش)' : 'حکم ساخته‌شده'})}
        ${statCard({ico: 'Σ', tint: '#DCFCE7', accent: '#16A34A', val: money(d.totalDelta), label: 'جمع افزایش ماهانه'})}
      </div>` +
      liveTable(['کارمند', 'نوع استخدام', 'پایهٔ پیشین', 'پایهٔ تازه', 'افزایش', 'شمارهٔ حکم'], rows, (d.rows || []).length);
  }
  toast(dryRun ? 'پیش‌نمایش آماده شد' : `${fa(d.created)} حکم گروهی ساخته شد`);
}

function liveDecreeIssue(id) {
  liveSend(`/api/v1/salary_admin/decrees/${id}/issue`, {method: 'POST'}, 'حکم حقوق صادر شد')
    .then(ok => ok && liveReload());
}
function liveDecreeNotify(id) {
  liveSend(`/api/v1/salary_admin/decrees/${id}/notify`, {method: 'POST'}, 'حکم حقوق ابلاغ شد')
    .then(ok => ok && liveReload());
}
async function liveDecreeRevise(id) {
  const salaryStr = prompt('مبلغ پایهٔ جدید حکم اصلاحی (ریال، خالی = بدون تغییر):') || '';
  const reason = prompt('علت اصلاح حکم:') || '';
  if (!reason) return;
  const res = await apiFetch(`/api/v1/salary_admin/decrees/${id}/revise`, {
    method: 'POST', body: {baseSalary: Number(salaryStr) || 0, reasonFa: reason},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('پیش‌نویس حکم اصلاحی ثبت شد'); liveReload();
}

/* =====================================================================
   گواهی اشتغال
   ===================================================================== */

const CERT_KIND_FA = {simple: 'گواهی سادهٔ اشتغال', bank: 'گواهی اشتغال جهت بانک', embassy: 'گواهی اشتغال جهت سفارت', other: 'سایر'};

function pageEmploymentCertsLive() {
  if (!isLive()) return offlineFallback('employmentCerts');
  liveLoad(async () => {
    const res = await apiFetch(liveQueryURL('employmentCerts', '/api/v1/salary_admin/certificates?size=200'));
    if (!res.ok) return liveErrorBox(res.message);
    const items = res.data.items || [];

    const stats = `<div class="grid-stats">
      ${statCard({ico: '📄', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.filter(c => c.status === 'issued' || c.status === 'delivered').length), label: 'گواهی صادرشده'})}
      ${statCard({ico: '🏦', tint: '#DCFCE7', accent: '#16A34A', val: fa(items.filter(c => c.kind === 'bank').length), label: 'گواهی جهت بانک'})}
      ${statCard({ico: '🛂', tint: '#EDE9FE', accent: '#7C3AED', val: fa(items.filter(c => c.kind === 'embassy').length), label: 'گواهی جهت سفارت'})}
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(items.filter(c => c.status === 'requested').length), label: 'در انتظار تأیید'})}
    </div>`;

    const rows = items.map(c => `<tr>
      <td class="num ltr">${esc(c.serial || c.id)}</td>
      <td>${personCell(c.employeeId, c.employeeName)}</td>
      <td>${esc(c.kindFa || CERT_KIND_FA[c.kind] || c.kind)}</td>
      <td>${faDate(c.requestedAt)}</td>
      <td>${faDate(c.issuedAt)}</td>
      <td>${statusChip(c.status, c.statusFa)}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-ghost" title="چاپ گواهی" onclick="liveOpenDoc('employment_certificate','${esc(c.employeeId)}')">🖨️</button>
        ${c.status === 'requested' ? `<button class="btn btn-sm btn-ghost" onclick="liveCertEdit('${c.id}')">✏️ ویرایش</button>` : ''}
        ${c.status === 'requested' ? `<button class="btn btn-sm btn-ghost" onclick="liveCertDelete('${c.id}')">🗑 حذف</button>` : ''}
        ${c.status === 'requested' ? `<button class="btn btn-sm btn-primary" onclick="liveCertDecide('${c.id}',true)">تأیید</button>
          <button class="btn btn-sm btn-ghost" onclick="liveCertDecide('${c.id}',false)">رد</button>` : ''}
        ${c.status === 'approved' ? `<button class="btn btn-sm btn-primary" onclick="liveCertIssue('${c.id}')">صدور گواهی</button>` : ''}
        ${c.status === 'issued' ? `<button class="btn btn-sm btn-ghost" onclick="liveCertDeliver('${c.id}')">تحویل شد</button>` : ''}
      </div></td></tr>`);

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      ℹ️ گردش‌کار گواهی: درخواست ← تأیید ← صدور ← تحویل. متن گواهی از دادهٔ کارمند و قرارداد جاری ساخته می‌شود
      و کد رهگیری آن در سامانه قابل بررسی است.
    </div>`;

    return stats + note + liveTable(
      ['شمارهٔ گواهی', 'کارمند', 'نوع گواهی', 'تاریخ درخواست', 'تاریخ صدور', 'وضعیت', 'عملیات'],
      rows, items.length);
  });
  return liveShell('گواهی اشتغال',
    'ثبت درخواست و صدور گواهی اشتغال به کار، گواهی بانکی و گواهی سفارت',
    liveSearchBar('employmentCerts', '🔍 جستجوی شمارهٔ گواهی، نام کارمند یا کد پرسنلی…') +
    `<button class="btn btn-primary" onclick="liveCertForm()">➕ ثبت درخواست گواهی</button>`);
}

async function liveCertForm() {
  const selectHtml = await liveEmployeeSelect('certEmp', null);
  openModal({
    title: 'ثبت درخواست گواهی اشتغال',
    body: `<div class="form-grid">
      <div class="field"><label>کارمند <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>نوع گواهی <span class="req">*</span></label>
        <select id="certKind">
          <option value="simple">گواهی سادهٔ اشتغال</option>
          <option value="bank">گواهی اشتغال جهت بانک</option>
          <option value="embassy">گواهی اشتغال جهت سفارت</option>
          <option value="other">سایر</option>
        </select></div>
      <div class="field"><label>مرجع درخواست‌کننده</label><input id="certPurpose" placeholder="مثلاً: بانک ملت"></div>
      <div class="field"><label>یادداشت</label><input id="certNote"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveCertSave()">ثبت</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveCertSave() {
  const body = {
    employeeId: $('#certEmp').value,
    kind: $('#certKind').value,
    purposeFa: $('#certPurpose').value.trim(),
    noteFa: $('#certNote').value.trim(),
  };
  const res = await apiFetch('/api/v1/salary_admin/certificates', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('درخواست گواهی ثبت شد'); liveReload();
}

/** ویرایش درخواست گواهی — تنها تا پیش از تأیید مدیر (قاعدهٔ دامنه). */
async function liveCertEdit(id) {
  const res = await apiFetch('/api/v1/salary_admin/certificates/' + encodeURIComponent(id));
  if (!res.ok) { toast(res.message, 'error'); return; }
  const c = res.data;
  const selectHtml = await liveEmployeeSelect('certEmp', c.employeeId);
  const kindOpts = Object.entries(CERT_KIND_FA)
    .map(([v, l]) => `<option value="${v}" ${v === c.kind ? 'selected' : ''}>${esc(l)}</option>`).join('');
  openModal({
    title: 'ویرایش درخواست گواهی ' + (c.serial || c.id),
    body: `<div class="form-grid">
      <div class="field"><label>کارمند <span class="req">*</span></label>${selectHtml}</div>
      <div class="field"><label>نوع گواهی <span class="req">*</span></label><select id="certKind">${kindOpts}</select></div>
      <div class="field"><label>مرجع درخواست‌کننده</label><input id="certPurpose" value="${esc(c.purposeFa || '')}"></div>
      <div class="field"><label>یادداشت</label><input id="certNote" value="${esc(c.noteFa || '')}"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveCertUpdate('${esc(c.id)}')">ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function liveCertUpdate(id) {
  const body = {
    employeeId: $('#certEmp').value,
    kind: $('#certKind').value,
    purposeFa: $('#certPurpose').value.trim(),
    noteFa: $('#certNote').value.trim(),
  };
  const res = await apiFetch('/api/v1/salary_admin/certificates/' + encodeURIComponent(id), {method: 'PATCH', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal(); toast('درخواست گواهی ویرایش شد'); liveReload();
}
/** حذف درخواست گواهی — قاعدهٔ دامنه در سرور اعمال می‌شود. */
function liveCertDelete(id) {
  if (!confirm('این درخواست گواهی حذف شود؟')) return;
  liveSend('/api/v1/salary_admin/certificates/' + encodeURIComponent(id), {method: 'DELETE'}, 'درخواست گواهی حذف شد')
    .then(ok => ok && liveReload());
}
function liveCertDecide(id, approve) {
  const note = approve ? '' : prompt('دلیل رد درخواست گواهی:');
  if (!approve && !note) return;
  liveSend(`/api/v1/salary_admin/certificates/${id}/decision`, {method: 'POST', body: {approve, noteFa: note || ''}},
    approve ? 'درخواست گواهی تأیید شد' : 'درخواست گواهی رد شد').then(ok => ok && liveReload());
}
function liveCertIssue(id) {
  liveSend(`/api/v1/salary_admin/certificates/${id}/issue`, {method: 'POST'}, 'گواهی صادر شد')
    .then(ok => ok && liveReload());
}
function liveCertDeliver(id) {
  liveSend(`/api/v1/salary_admin/certificates/${id}/deliver`, {method: 'POST'}, 'تحویل گواهی ثبت شد')
    .then(ok => ok && liveReload());
}

/* =====================================================================
   سند چاپی: با هدر احراز هویت می‌گیریم و در زبانهٔ تازه باز می‌کنیم
   (بازکردن مستقیم URL توکن را نمی‌فرستد، پس سند ۴۰۱ می‌شد).
   ===================================================================== */

async function liveOpenDoc(kind, id) {
  if (!DOC_KINDS[kind]) { toast('نوع سند ناشناخته است', 'error'); return; }
  try {
    const res = await apiRaw(`/api/v1/documents/${kind}/${encodeURIComponent(id)}`);
    if (!res.ok) {
      let msg = 'صدور سند ناموفق بود';
      try { const d = await res.json(); msg = jsonMessage(res.status, d); } catch (_) {}
      toast(msg, 'error');
      return;
    }
    const html = await res.text();
    const url = URL.createObjectURL(new Blob([html], {type: 'text/html;charset=utf-8'}));
    window.open(url, '_blank');
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  } catch (e) {
    toast('ارتباط با سرور برقرار نشد', 'error');
  }
}

/* =====================================================================
   فیش حقوقی پارامتریک (زنده)
   ===================================================================== */

/** فیش حقوقی کارمند را از سرور می‌گیرد و اجزای مزد و فوق‌العاده‌ها را نشان می‌دهد. */
function pagePayslipLive() {
  if (!isLive()) return offlineFallback('payslip');
  const empId = session.emp || 'EMP-001';
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/payroll/payslip?employee=' + encodeURIComponent(empId));
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    const r = d.result || {};
    const wages = d.wages || {};
    const mazd = (wages.job || 0) + (wages.rank || 0) + (wages.seniority || 0) + (wages.postAllowance || 0) + (wages.sacrifice || 0);
    const elements = d.elements || [];
    const earn = elements.filter(e => e.category === 'earning');
    const ded = elements.filter(e => e.category === 'deduction');
    const line = e => `<div class="slip-row"><span>${esc(e.titleFa)}${e.resolved ? '' : ' <span class="muted">(حل‌نشده)</span>'}</span><span class="num">${e.resolved ? money(Math.round(e.amount || 0)) : '—'}</span></div>`;
    const sum = arr => arr.filter(e => e.resolved).reduce((s, e) => s + (e.amount || 0), 0);

    const stats = `<div class="grid-stats">
      ${statCard({ico: '🧮', tint: '#EDE9FE', accent: '#7C3AED', val: money(Math.round(mazd)), label: 'مزد مبنا'})}
      ${statCard({ico: '💵', tint: '#EFF6FF', accent: '#2563EB', val: money(r.gross), label: 'جمع ناخالص'})}
      ${statCard({ico: '🧾', tint: '#FEE2E2', accent: '#DC2626', val: money(r.tax), label: 'مالیات'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: money(r.net), label: 'خالص پرداختی'})}
    </div>`;

    const slip = `<div class="payslip card">
      <h3>فیش حقوقی ${esc(d.period || '')}</h3>
      <div class="muted" style="text-align:center">${esc((d.employee || {}).fullName || '')} — کد پرسنلی <span class="num ltr">${esc((d.employee || {}).id || '')}</span> — ${esc((d.employee || {}).department || '')}</div>
      <div class="slip-grid">
        <div class="slip-col">
          <b style="font-size:.85rem">اجزای دریافتی و فوق‌العاده‌ها</b>
          ${earn.map(line).join('')}
          <div class="slip-total"><span>جمع دریافتی</span><span class="num">${money(Math.round(sum(earn)))}</span></div>
        </div>
        <div class="slip-col">
          <b style="font-size:.85rem">کسورات</b>
          ${ded.map(line).join('')}
          <div class="slip-total"><span>جمع کسورات</span><span class="num">${money(Math.round(sum(ded)))}</span></div>
        </div>
      </div>
      <div class="slip-total" style="margin-top:14px;font-size:1.05rem">
        <span>خالص پرداختی (ریال)</span><span class="num">${money(r.net)}</span>
      </div>
      <div class="muted" style="margin-top:8px;font-size:.78rem;line-height:2">
        جمع ناخالص محاسبهٔ مرجع سامانه: <b class="num">${money(r.gross)}</b> —
        بیمهٔ سهم کارمند: <b class="num">${money(r.insuranceEmployee)}</b> —
        بیمهٔ کارفرما: <b class="num">${money(r.insuranceEmployer)}</b>
      </div>
    </div>`;

    const wageRows = [
      ['مزد شغل', wages.job],
      ['مزد رتبه', wages.rank],
      ['مزد سنوات', wages.seniority],
      ['فوق‌العادهٔ پست', wages.postAllowance],
      ['فوق‌العادهٔ ایثارگری', wages.sacrifice],
    ].map(([label, v]) => `<tr><td>${label}</td><td class="num">${money(Math.round(v || 0))}</td></tr>`);
    wageRows.push(`<tr><td><b>مزد مبنا</b></td><td class="num"><b>${money(Math.round(mazd))}</b></td></tr>`);
    const wageTable = liveTable(['جزء مزد', 'مبلغ (ریال)'], wageRows, wageRows.length);

    return stats + slip + `<div class="grid-2eq" style="margin-top:14px">
      <div class="card card-pad"><div class="card-title">شاکلهٔ مزد (مبنای فوق‌العاده‌ها)</div>${wageTable}</div>
      <div class="card card-pad"><div class="card-title">توضیح</div>
        <p class="muted" style="font-size:.8rem;line-height:2">مزد مبنا از جمع مزد شغل، مزد رتبه، مزد سنوات و فوق‌العاده‌های پست و ایثارگری ساخته می‌شود و فوق‌العاده‌های درصدی (بدی آب و هوا، محرومیت از تسهیلات زندگی، صعوبت شرایط منطقه، جبران صرف وقت اضافی، درصد فوق‌العادهٔ شغل و نوبت‌کاری) روی همین مبنا حساب می‌شوند.</p>
      </div>
    </div>`;
  });
  return liveShell('فیش حقوقی', 'فیش پارامتریک کارمند با اجزای مزد و فوق‌العاده‌ها',
    `<button class="btn btn-ghost" onclick="liveOpenDoc('payslip','${esc(empId)}')">🖨️ چاپ / PDF فیش</button>
     <button class="btn btn-primary" onclick="openPayrollSimulator()">🧮 شبیه‌سازی فیش</button>`);
}

/* =====================================================================
   مسیریابی صفحات زنده — مسیریاب (pages3.js) اگر کلید اینجا باشد، همین را
   جایگزین صفحهٔ نمایشی می‌کند.
   ===================================================================== */

/* =====================================================================
   نقش‌ها و سطوح دسترسی (ویژهٔ مدیر سیستم)
   ===================================================================== */

/** صفحهٔ «نقش‌ها و سطوح دسترسی»: ماتریس نقش × ماژول + دادن/گرفتن دسترسی افزوده. */
function pageRolesLive() {
  if (!isLive()) return offlineFallback('roles');
  liveLoad(async () => {
    // دامنهٔ کنترل پنل (شرکت یا هلدینگ) از انتخابگر می‌آید؛ سرور فقط سطح‌های همان
    // دامنه را برمی‌گرداند و مدیر سیستم می‌تواند بین هلدینگ و شرکت‌هایش جابه‌جا شود.
    const res = await apiFetch('/api/v1/settings/access' + (window.__ACCESS_SCOPE || ''));
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    window.__ACCESS = d;
    const roles = d.roles || [];
    const depts = d.departments || [];
    const manifests = d.modules || [];
    const grants = d.grants || [];
    const holdings = d.holdings || [];
    const scopeInfo = d.scope || {};
    const orgTitle = {};
    holdings.forEach(h => (h.companies || []).forEach(c => { orgTitle[c.id] = c.titleFa; }));
    const scopeFa = g => g.organizationId ? 'شرکت: ' + (orgTitle[g.organizationId] || g.organizationId)
      : (g.holdingFa ? 'هلدینگ: ' + g.holdingFa : 'جهانی');
    const scopeVal = scopeInfo.kind === 'holding' ? 'H:' + (scopeInfo.holdingFa || '') : 'O:' + (scopeInfo.organizationId || '');
    const scopeOpts = holdings.map(h =>
      `<option value="H:${esc(h.key)}" ${scopeVal === 'H:' + h.key ? 'selected' : ''}>🏛️ هلدینگ ${esc(h.titleFa)} — ${fa((h.companies || []).length)} شرکت</option>` +
      (h.companies || []).map(c =>
        `<option value="O:${esc(c.id)}" ${scopeVal === 'O:' + c.id ? 'selected' : ''}>🏢 ${esc(c.titleFa)}</option>`).join('')).join('');
    window.__ACCESS_SCOPE_SEL = scopeOpts;
    const actFa = a => ((d.actions || []).find(x => x.value === a) || {}).labelFa || a;
    const deptFa = id => (depts.find(x => x.id === id) || {}).titleFa || '—';
    const roleFa = r => (roles.find(x => x.value === r) || {}).labelFa || r;
    const modFa = id => id === '*' ? 'همهٔ ماژول‌ها' : ((manifests.find(m => m.id === id) || {}).titleFa || id);

    const selDept = window.__ACCESS_DEPT || '';
    const mods = selDept ? manifests.filter(m => m.department === selDept) : manifests;

    const rows = mods.map(m => {
      const cells = roles.map(r => {
        const man = m.manifest[r.value] || [];
        const gr = m.granted[r.value] || [];
        if (!man.length && !gr.length) return '<td class="muted">—</td>';
        const chips = [];
        man.forEach(a => chips.push(pChip('c-ok', actFa(a))));
        gr.forEach(a => chips.push(pChip('c-wait', 'افزوده: ' + actFa(a))));
        return `<td>${chips.join(' ')}</td>`;
      }).join('');
      return `<tr>
        <td><b>${esc(m.titleFa)}</b><div class="muted" style="font-size:.72rem">${esc(m.id)} · ${esc(m.departmentFa)}</div></td>
        ${cells}
        <td><button class="btn btn-sm btn-ghost" title="دادن دسترسی" onclick="liveGrantForm('${esc(m.id)}','${esc(selDept)}')">➕</button></td></tr>`;
    });

    const head = ['ماژول', ...roles.map(r => r.labelFa), 'افزودن'];

    const grantRows = grants.map(g => `<tr>
      <td>${esc(roleFa(g.role))}</td>
      <td>${esc(modFa(g.moduleId))}</td>
      <td>${esc(scopeFa(g))}</td>
      <td>${g.departmentId ? esc(deptFa(g.departmentId)) : 'همهٔ اداره‌ها'}</td>
      <td class="ltr">${(g.actions || []).map(actFa).join('، ')}</td>
      <td>${esc(g.createdBy || '—')}</td>
      <td><button class="btn btn-sm btn-ghost" title="حذف" onclick="liveRevokeGrant('${esc(g.id)}')">🗑️</button></td></tr>`);

    const deptSel = `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:12px;align-items:center;flex-wrap:wrap">
      <span class="muted" style="font-size:.82rem">ماتریس پایه از مانیفست ماژول‌ها می‌آید (سبز) و سطوح افزودهٔ مدیر سیستم با «افزوده» مشخص است. کنترل پنل فقط برای دامنهٔ انتخاب‌شده کار می‌کند: شرکت یا هلدینگ (با زیرمجموعه‌هایش).</span>
      <select class="sel" title="دامنهٔ کنترل پنل" onchange="window.__ACCESS_SCOPE = this.value.startsWith('H:') ? '?holding=' + encodeURIComponent(this.value.slice(2)) : '?org=' + encodeURIComponent(this.value.slice(2)); renderPage('roles')">
        ${window.__ACCESS_SCOPE_SEL || ''}
      </select>
      <select class="sel" onchange="window.__ACCESS_DEPT=this.value; renderPage('roles')">
        <option value="">همهٔ اداره‌ها</option>
        ${depts.map(x => `<option value="${esc(x.id)}" ${x.id === selDept ? 'selected' : ''}>${esc(x.titleFa)}</option>`).join('')}
      </select>
    </div>`;

    const grantBox = `<div class="card card-pad" style="margin-top:18px">
      <div class="card-title">سطوح دسترسی افزودهٔ مدیر سیستم</div>
      ${grants.length
        ? liveTable(['نقش', 'ماژول', 'دامنه', 'اداره', 'اقدام‌ها', 'سازنده', 'حذف'], grantRows, grantRows.length)
        : '<p class="muted" style="font-size:.82rem;line-height:2">هنوز سطح افزوده‌ای تعریف نشده است. با «➕» کنار هر ماژول می‌توانید به یک نقش، برای یک اداره، دسترسی بیشتری بدهید.</p>'}
    </div>`;

    return deptSel + liveTable(head, rows, rows.length) + grantBox;
  });
  return liveShell('نقش‌ها و سطوح دسترسی', 'دادن دسترسی به هر اداره و هر ماژول برای نقش‌ها (ویژهٔ مدیر سیستم)',
    `<button class="btn btn-primary" onclick="liveGrantForm('','')">➕ سطح دسترسی جدید</button>`);
}

/** فرم دادن سطح دسترسی. */
function liveGrantForm(moduleId, departmentId) {
  const d = window.__ACCESS || {roles: [], departments: [], modules: [], actions: []};
  openModal({
    title: 'سطح دسترسی جدید',
    body: `<div class="form-grid">
      <div class="field"><label>نقش</label><select id="grRole">
        ${(d.roles || []).map(r => `<option value="${esc(r.value)}">${esc(r.labelFa)}</option>`).join('')}
      </select></div>
      <div class="field"><label>ماژول</label><select id="grModule">
        <option value="*">همهٔ ماژول‌ها</option>
        ${(d.modules || []).map(m => `<option value="${esc(m.id)}" ${m.id === moduleId ? 'selected' : ''}>${esc(m.titleFa)} — ${esc(m.departmentFa)}</option>`).join('')}
      </select></div>
      <div class="field"><label>اداره (خالی = همهٔ اداره‌ها)</label><select id="grDept">
        <option value="">همهٔ اداره‌ها</option>
        ${(d.departments || []).map(x => `<option value="${esc(x.id)}" ${x.id === departmentId ? 'selected' : ''}>${esc(x.titleFa)}</option>`).join('')}
      </select></div>
      <div class="field"><label>دامنه (شرکت / هلدینگ / جهانی)</label><select id="grScope">
        <option value="company">شرکت فعال — فقط همین شرکت</option>
        <option value="holding">هلدینگ فعال — همهٔ شرکت‌های زیرمجموعه</option>
        <option value="all">جهانی — همهٔ شرکت‌ها (ویژهٔ مدیر سیستم)</option>
      </select></div>
      <div class="field"><label>یادداشت</label><input id="grNote" placeholder="مثلاً: دسترسی موقت گزارش‌گیری"></div>
    </div>
    <div class="field" style="margin-top:12px"><label>اقدام‌ها</label>
      <div style="display:flex;gap:14px;flex-wrap:wrap">
        ${(d.actions || []).map(a => `<label style="display:flex;gap:6px;align-items:center"><input type="checkbox" class="grAct" value="${esc(a.value)}" ${a.value === 'read' ? 'checked' : ''}> ${esc(a.labelFa)}</label>`).join('')}
      </div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="liveGrantSubmit()">ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}

/** ذخیرهٔ سطح دسترسی از فرم. */
async function liveGrantSubmit() {
  const actions = [...document.querySelectorAll('.grAct:checked')].map(i => i.value);
  const body = {
    role: $('#grRole').value,
    moduleId: $('#grModule').value,
    departmentId: $('#grDept').value,
    scope: ($('#grScope') || {}).value || 'company',
    noteFa: $('#grNote').value,
    actions,
  };
  const res = await apiFetch('/api/v1/settings/access/grants', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('سطح دسترسی ذخیره شد');
  closeModal();
  renderPage('roles');
}

/** حذف یک سطح دسترسی. */
async function liveRevokeGrant(id) {
  requireConfirm('این سطح دسترسی حذف شود؟', async () => {
    const res = await apiFetch('/api/v1/settings/access/grants/' + encodeURIComponent(id), {method: 'DELETE'});
    if (!res.ok) { toast(res.message, 'error'); return; }
    toast('سطح دسترسی حذف شد');
    renderPage('roles');
  });
}

/* =====================================================================
   مدیریت کاربران (زنده)
   ===================================================================== */

function pageUsersLive() {
  if (!isLive()) return offlineFallback('users');
  liveLoad(async () => {
    const [ur, rr] = await Promise.all([
      apiFetch('/api/v1/settings/users'),
      apiFetch('/api/v1/settings/roles'),
    ]);
    if (!ur.ok) return liveErrorBox(ur.message);
    const users = Array.isArray(ur.data) ? ur.data : (ur.data.items || []);
    const roles = (rr.ok && Array.isArray(rr.data)) ? rr.data : [];
    const roleFa = r => (roles.find(x => x.value === r) || {}).labelFa || r;
    const byRole = r => users.filter(u => u.role === r).length;

    const stats = `<div class="grid-stats">
      ${statCard({ico: '👥', tint: '#EFF6FF', accent: '#2563EB', val: fa(users.length), label: 'کل کاربران'})}
      ${statCard({ico: '🛡️', tint: '#EDE9FE', accent: '#7C3AED', val: fa(byRole('admin')), label: 'مدیر سیستم'})}
      ${statCard({ico: '🧑‍💼', tint: '#DCFCE7', accent: '#16A34A', val: fa(byRole('hr_manager') + byRole('hr_spec')), label: 'کارشناس/مدیر منابع انسانی'})}
      ${statCard({ico: '👤', tint: '#FEF3C7', accent: '#D97706', val: fa(byRole('employee') + byRole('line_manager')), label: 'کارمند/مدیر واحد'})}
    </div>`;

    const rows = users.map(u => `<tr>
      <td class="ltr">${esc(u.username)}</td>
      <td>${esc(u.displayName || '—')}</td>
      <td>${pChip('c-ok', esc(roleFa(u.role)))}</td>
      <td class="num ltr">${esc(u.employeeId || '—')}</td>
      <td class="num ltr">${esc(u.departmentId || '—')}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="go('roles')">🛡️ سطوح دسترسی</button></td>
    </tr>`);

    return stats + liveTable(['نام کاربری', 'نام و نام خانوادگی', 'نقش', 'کد پرسنلی', 'اداره', 'عملیات'], rows, rows.length);
  });
  return liveShell('مدیریت کاربران', 'کاربران سامانه و نقش هر کدام؛ سطوح دسترسی از صفحهٔ نقش‌ها مدیریت می‌شود',
    `<button class="btn btn-primary" onclick="go('roles')">🛡️ سطوح دسترسی</button>`);
}

/* =====================================================================
   لاگ حسابرسی (زنده)
   ===================================================================== */

function pageAuditLive() {
  if (!isLive()) return offlineFallback('audit');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/settings/audit');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    const events = (d.events || []).slice().reverse();
    const invalid = events.filter(e => e.action && e.action.indexOf('denied') === 0).length;

    const stats = `<div class="grid-stats">
      ${statCard({ico: '📜', tint: '#EFF6FF', accent: '#2563EB', val: fa(d.total || events.length), label: 'کل رویدادها'})}
      ${statCard({ico: d.chainValid ? '🔒' : '⚠️', tint: d.chainValid ? '#DCFCE7' : '#FEE2E2', accent: d.chainValid ? '#16A34A' : '#DC2626', val: d.chainValid ? 'سالم' : 'معیوب', label: 'زنجیرهٔ هش'})}
      ${statCard({ico: '⛔', tint: '#FEE2E2', accent: '#DC2626', val: fa(invalid), label: 'رد دسترسی'})}
    </div>`;

    const rows = events.map(e => `<tr>
      <td class="num ltr">${esc(e.seq)}</td>
      <td class="ltr">${esc(e.actor || '—')}</td>
      <td>${esc(e.module || '—')}</td>
      <td>${e.action && e.action.indexOf('denied') === 0 ? pChip('c-rej', esc(e.action)) : esc(e.action || '—')}</td>
      <td>${esc(e.detail || e.target || '—')}</td>
      <td class="num ltr">${esc(e.at || '')}</td>
    </tr>`);

    return stats + liveTable(['#', 'کاربر', 'ماژول', 'رویداد', 'شرح', 'زمان'], rows, rows.length);
  });
  return liveShell('لاگ حسابرسی', 'زنجیرهٔ تغییرناپذیر رویدادها و رد شدن دسترسی‌ها');
}

/* =====================================================================
   گزارش‌های مالی حقوق (زنده)
   ===================================================================== */

function pagePayReportsLive() {
  if (!isLive()) return offlineFallback('payReports');
  liveLoad(async () => {
    const [runs, items, params] = await Promise.all([
      apiFetch('/api/v1/payroll/records?kind=payrollRuns&size=50'),
      apiFetch('/api/v1/payroll/records?kind=payItems&size=200'),
      apiFetch('/api/v1/payroll/parameters'),
    ]);
    if (!runs.ok) return liveErrorBox(runs.message);
    const list = (runs.data.items || []);
    const elements = (items.ok && items.data.items) || [];
    const values = (params.ok && params.data.values) || {};
    const num = v => Number(String(v == null ? '' : v).replace(/[^0-9.-]/g, '')) || 0;
    const sum = key => list.reduce((s, r) => s + num((r.fields || {})[key]), 0);

    const stats = `<div class="grid-stats">
      ${statCard({ico: '🗓️', tint: '#EFF6FF', accent: '#2563EB', val: fa(list.length), label: 'دورهٔ پردازش'})}
      ${statCard({ico: '💵', tint: '#DCFCE7', accent: '#16A34A', val: money(sum('gross')), label: 'جمع ناخالص دوره‌ها'})}
      ${statCard({ico: '🧾', tint: '#FEE2E2', accent: '#DC2626', val: money(sum('tax')), label: 'جمع مالیات'})}
      ${statCard({ico: '✅', tint: '#EDE9FE', accent: '#7C3AED', val: money(sum('net')), label: 'جمع خالص پرداختی'})}
    </div>`;

    const rows = list.map(r => {
      const f = r.fields || {};
      return `<tr>
        <td class="num ltr">${esc(r.id)}</td>
        <td>${esc(r.title || f.period || '—')}</td>
        <td class="num">${money(num(f.gross))}</td>
        <td class="num">${money(num(f.tax))}</td>
        <td class="num">${money(num(f.insurance))}</td>
        <td class="num">${money(num(f.net))}</td>
        <td>${statusChip ? statusChip(r.status, r.statusFa) : esc(r.statusFa || r.status || '')}</td>
      </tr>`;
    });

    const keys = Object.keys(values).sort();
    const paramRows = keys.map(k => `<tr><td class="ltr">${esc(k)}</td><td class="num ltr">${esc(values[k])}</td></tr>`);

    return stats
      + liveTable(['شماره', 'دوره', 'جمع ناخالص', 'مالیات', 'بیمه', 'خالص', 'وضعیت'], rows, rows.length)
      + `<div class="grid-2eq" style="margin-top:16px">
          <div class="card card-pad"><div class="card-title">اقلام حقوق فعال (${fa(elements.length)})</div>
            ${liveTable(['قلم', 'کد', 'روش'], elements.slice(0, 12).map(e => `<tr><td>${esc(e.title)}</td><td class="ltr">${esc((e.fields || {}).code || '')}</td><td>${esc((e.fields || {}).calc || '')}</td></tr>`), elements.length)}
          </div>
          <div class="card card-pad"><div class="card-title">پارامترهای مؤثر (${fa(keys.length)})</div>
            ${liveTable(['کلید', 'مقدار'], paramRows, paramRows.length)}
          </div>
        </div>`;
  });
  return liveShell('گزارش‌های مالی حقوق', 'جمع دوره‌های پردازش، مالیات، بیمه و پارامترهای مؤثر',
    `<button class="btn btn-ghost" onclick="go('reportBuilder')">🧱 گزارش‌ساز سفارشی</button>`);
}

/* =====================================================================
   پارامترهای حقوق و شبیه‌سازی فیش (پارامتریک)
   ===================================================================== */

/** شبیه‌سازی فیش بر پایهٔ پارامترها و اقلام فعال حقوق در سرور. */
function openPayrollSimulator() {
  openModal({
    title: 'شبیه‌سازی فیش حقوقی بر پایهٔ پارامترها',
    body: `<p class="muted" style="font-size:.78rem;margin-bottom:10px">مزد مبنا از جمع مزد شغل، مزد رتبه، مزد سنوات و فوق‌العاده‌های پست و ایثارگری ساخته می‌شود و فوق‌العاده‌های درصدی روی آن حساب می‌شوند.</p>
    <div class="form-grid">
      <div class="field"><label>مزد شغل (ریال)</label><input id="simBase" type="number" value="120000000"></div>
      <div class="field"><label>مزد رتبه (ریال)</label><input id="simRank" type="number" value="0"></div>
      <div class="field"><label>مزد سنوات (ریال)</label><input id="simSeniority" type="number" value="0"></div>
      <div class="field"><label>فوق‌العادهٔ پست (ریال)</label><input id="simPost" type="number" value="0"></div>
      <div class="field"><label>فوق‌العادهٔ ایثارگری (ریال)</label><input id="simSacrifice" type="number" value="0"></div>
      <div class="field"><label>تعداد فرزند</label><input id="simChildren" type="number" value="2"></div>
      <div class="field"><label>نرخ ساعتی (ریال)</label><input id="simOtRate" type="number" value="700000"></div>
      <div class="field"><label>ساعت اضافه‌کاری</label><input id="simOtHours" type="number" value="12"></div>
      <div class="field"><label>ساعت جمعه‌کاری</label><input id="simFridayHours" type="number" value="0"></div>
      <div class="field"><label>ساعت تعطیل‌کاری</label><input id="simHolidayHours" type="number" value="0"></div>
      <div class="field"><label>سایر کسور (ریال)</label><input id="simDeduct" type="number" value="0"></div>
      <div class="field"><label>ضریب اضافه‌کاری (اختیاری)</label><input id="simOtFactor" placeholder="1.4"></div>
      <div class="field"><label>کارانهٔ تأییدشده (ریال)</label><input id="simKarana" type="number" value="0"></div>
      <div class="field"><label>روز مأموریت</label><input id="simMissionDays" type="number" value="0"></div>
      <div class="field"><label>فوق‌العادهٔ روزانهٔ مأموریت (ریال)</label><input id="simMissionPerDiem" type="number" value="0"></div>
    </div>
    <div id="simResult" style="margin-top:14px"></div>`,
    foot: `<button class="btn btn-primary" onclick="runPayrollSimulator()">محاسبه</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}

async function runPayrollSimulator() {
  const body = {
    baseSalary: Number($('#simBase').value) || 0,
    rankWage: Number($('#simRank').value) || 0,
    seniorityWage: Number($('#simSeniority').value) || 0,
    postAllowance: Number($('#simPost').value) || 0,
    sacrificeAllowance: Number($('#simSacrifice').value) || 0,
    children: Number($('#simChildren').value) || 0,
    overtimeHours: Number($('#simOtHours').value) || 0,
    overtimeHourlyRate: Number($('#simOtRate').value) || 0,
    fridayHours: Number($('#simFridayHours').value) || 0,
    holidayHours: Number($('#simHolidayHours').value) || 0,
    otherDeductions: Number($('#simDeduct').value) || 0,
    karanaAmount: Number($('#simKarana').value) || 0,
    missionDays: Number($('#simMissionDays').value) || 0,
    missionPerDiem: Number($('#simMissionPerDiem').value) || 0,
    overrides: {},
  };
  const factor = ($('#simOtFactor').value || '').trim();
  if (factor) body.overrides['overtime.factor'] = factor;
  const res = await apiFetch('/api/v1/payroll/simulate', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const r = res.data.result || {};
  const elements = res.data.elements || [];
  const sum = `<div class="grid-stats">
    ${statCard({ico: '💵', tint: '#EFF6FF', accent: '#2563EB', val: money(r.gross), label: 'جمع ناخالص'})}
    ${statCard({ico: '🛡️', tint: '#FEF3C7', accent: '#D97706', val: money(r.insuranceEmployee), label: 'بیمهٔ سهم کارمند'})}
    ${statCard({ico: '🧾', tint: '#FEE2E2', accent: '#DC2626', val: money(r.tax), label: 'مالیات'})}
    ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: money(r.net), label: 'خالص پرداختی'})}
  </div>`;
  const rows = elements.map(e => `<tr>
    <td>${esc(e.titleFa)}</td>
    <td class="ltr num">${esc(e.code || '')}</td>
    <td>${e.category === 'deduction' ? pChip('c-rej', 'کسور') : pChip('c-ok', 'مزایا')}</td>
    <td class="num">${e.resolved ? money(Math.round(e.amount || 0)) : '—'}</td>
    <td class="muted" style="font-size:.75rem">${esc(e.detailFa || '')}</td></tr>`);
  $('#simResult').innerHTML = sum + liveTable(['قلم حقوق', 'کد', 'نوع', 'مبلغ محاسبه‌شده', 'توضیح'], rows, elements.length);
}

/* =====================================================================
   پیش‌نمایش قالب متن اسناد با دادهٔ نمونه
   ===================================================================== */

function openTemplatePreview() {
  openModal({
    title: 'پیش‌نمایش قالب متن اسناد',
    narrow: true,
    body: `<div class="field"><label>نوع سند</label>
      <select id="tplKind">
        <option value="contract">قرارداد کار</option>
        <option value="payslip">فیش حقوقی</option>
        <option value="salary_decree">حکم حقوق</option>
        <option value="employment_certificate">گواهی اشتغال</option>
      </select></div>
      <p class="muted" style="font-size:.78rem;margin-top:10px">پیش‌نمایش با دادهٔ نمونه (EMP-001) و قالب فعال همان نوع سند ساخته می‌شود.</p>`,
    foot: `<button class="btn btn-primary" onclick="openDocPreview()">بازکردن پیش‌نمایش</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}

async function openDocPreview() {
  const kind = ($('#tplKind') && $('#tplKind').value) || 'contract';
  try {
    const res = await apiRaw(`/api/v1/documents/preview/${kind}`);
    if (!res.ok) {
      let msg = 'ساخت پیش‌نمایش ناموفق بود';
      try { msg = jsonMessage(res.status, await res.json()); } catch (_) { /* بدنهٔ نامعتبر */ }
      toast(msg, 'error');
      return;
    }
    const html = await res.text();
    const url = URL.createObjectURL(new Blob([html], {type: 'text/html;charset=utf-8'}));
    window.open(url, '_blank');
    setTimeout(() => URL.revokeObjectURL(url), 60000);
  } catch (e) {
    toast('ارتباط با سرور برقرار نشد', 'error');
  }
}

/* =====================================================================
   گزارش‌ساز بر پایهٔ اقلام انتخاب‌شده
   ===================================================================== */

let RB_SOURCES = [];
/* انواع استخدام برای فیلتر گزارش — از پورت کارکنان سرور خوانده می‌شود. */
let RB_TYPES = [];
/* آخرین نتیجهٔ گزارش اجراشده؛ برای چاپ و خروجی PDF همان را دوباره مصرف می‌کنیم و
   گزارش را از نو اجرا نمی‌کنیم. */
let RB_LAST = null;
/* عملگرهای شرط گزارش — از سرور می‌آید تا برچسب عملگرها در UI هاردکد نشود. */
let RB_OPERATORS = [];

function pageReportBuilderLive() {
  if (!isLive()) return offlineFallback('reportBuilder');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/reports/sources');
    if (!res.ok) return liveErrorBox(res.message);
    RB_SOURCES = res.data.sources || [];
    RB_OPERATORS = res.data.operators || [];
    const typesRes = await apiFetch('/api/v1/employee/options');
    RB_TYPES = (typesRes.ok && typesRes.data.employmentTypes) || [];
    if (!RB_SOURCES.length) return liveErrorBox('منبع دادهٔ گزارشی ثبت نشده است.');
    const opts = RB_SOURCES.map(s => `<option value="${esc(s.key)}">${esc(s.titleFa)}</option>`).join('');
    setTimeout(renderReportFields, 0);
    return `<div class="card card-pad" style="margin-bottom:12px">
      <div class="form-grid">
        <div class="field"><label>منبع داده</label><select id="rbSource" onchange="renderReportFields()">${opts}</select></div>
        <div class="field"><label>سقف سطر</label><input id="rbLimit" type="number" value="200"></div>
      </div>
      <div id="rbFields" style="margin-top:12px"></div>
      <div id="rbFilters" style="margin-top:12px"></div>
      <div style="margin-top:14px">
        <div style="display:flex;align-items:center;gap:10px;margin-bottom:6px">
          <label style="font-size:.82rem;font-weight:600">شرط‌های دلخواه روی اقلام</label>
          <button class="btn btn-sm btn-ghost" onclick="addReportCondition()">➕ افزودن شرط</button>
        </div>
        <div id="rbConds" style="display:flex;flex-direction:column;gap:8px"></div>
      </div>
      <div style="margin-top:14px;display:flex;gap:8px;flex-wrap:wrap">
        <button class="btn btn-primary" onclick="runReportBuilder()">▶ اجرای گزارش</button>
        <button class="btn btn-ghost" onclick="saveReportDefinition()">💾 ذخیرهٔ تعریف گزارش</button>
        <button class="btn btn-ghost" onclick="printReportBuilder()">🖨️ چاپ / PDF</button>
      </div>
    </div>
    <div id="rbResult"></div>`;
  });
  return liveShell('گزارش‌ساز بر پایهٔ اقلام', 'انتخاب منبع داده و اقلام، سپس اجرای گزارش روی دادهٔ واقعی سرور', '');
}

function renderReportFields() {
  const sel = $('#rbSource');
  const box = $('#rbFields');
  if (!sel || !box) return;
  const src = RB_SOURCES.find(s => s.key === sel.value);
  if (!src) return;
  box.innerHTML = `<label style="font-size:.78rem;font-weight:600">اقلام گزارش</label>
    <div style="display:flex;flex-wrap:wrap;gap:10px;margin-top:6px">${src.fields.map(f =>
      `<label style="display:flex;gap:6px;align-items:center;font-size:.82rem;border:1px solid var(--border);border-radius:8px;padding:4px 10px">
        <input type="checkbox" class="rbField" value="${esc(f.key)}" checked> ${esc(f.labelFa)}</label>`).join('')}</div>`;
  renderReportFilters(src);
  resetReportConditions();
}

/* فیلترهای اعلامی منبع (نوع استخدام، وضعیت، نوع وام، بانک عامل…) را از خود سرور
   می‌سازد؛ هیچ گزینه‌ای این‌جا هاردکد نیست. فیلترهایی که سرور `multiple` اعلام کرده
   باشد «کشویی بازشو با تیک» می‌شوند تا کاربر چند گزینه را با هم انتخاب کند (مثلاً
   «کارگر رسمی» و «کارمند مدت معین ۱۶۶۲» در گزارش کارکنان). */
function renderReportFilters(src) {
  const box = $('#rbFilters');
  if (!box) return;
  const filters = src.filters || [];
  if (!filters.length) { box.innerHTML = ''; return; }
  const controls = filters.map(f => {
    let field;
    if (f.multiple) {
      field = reportMultiSelectHTML(f);
    } else if (f.type === 'select') {
      const opts = (f.options || []).map(o => `<option value="${esc(o.value)}">${esc(o.labelFa)}</option>`).join('');
      field = `<select id="rbF_${esc(f.key)}"><option value="">همه</option>${opts}</select>`;
    } else if (f.key === 'employmentType' && RB_TYPES.length) {
      field = `<select id="rbF_${esc(f.key)}"><option value="">همه</option>${RB_TYPES.map(t => `<option value="${esc(t.key)}">${esc(t.titleFa)}</option>`).join('')}</select>`;
    } else if (f.type === 'number' || f.type === 'money') {
      field = `<input id="rbF_${esc(f.key)}" type="number">`;
    } else {
      field = `<input id="rbF_${esc(f.key)}" placeholder="${esc(f.labelFa)}">`;
    }
    return `<div class="field"><label>${esc(f.labelFa)}</label>${field}</div>`;
  }).join('');
  box.innerHTML = `<label style="font-size:.78rem;font-weight:600">فیلترهای منبع</label>
    <div class="form-grid" style="margin-top:6px">${controls}</div>`;
}

/* کشویی بازشو با تیک برای یک فیلتر چندانتخابی: خلاصهٔ انتخاب روی خود دکمه دیده
   می‌شود و با هر تیک به‌روز می‌شود. گزینه‌ها از سرور می‌آیند و ترتیبشان همان ترتیب
   فهرست رسمی است (رسمی ← مدت معین ← قراردادی ← پیمانکاری ← مأموریت). */
function reportMultiSelectHTML(f) {
  const opts = (f.options || []).map(o =>
    `<label class="ms-opt"><input type="checkbox" class="rbMValue" value="${esc(o.value)}" onchange="reportMultiLabel(this)"> <span>${esc(o.labelFa)}</span></label>`).join('');
  return `<div class="ms" id="rbF_${esc(f.key)}">
    <button type="button" class="ms-toggle" onclick="toggleReportMulti(this)">
      <span class="ms-text">همه</span><span class="ms-caret">▾</span></button>
    <div class="ms-panel" hidden>${opts || '<span class="muted" style="font-size:.78rem">گزینه‌ای ثبت نشده است</span>'}</div>
  </div>`;
}

/* باز/بسته کردن پنل گزینه‌ها. پیش از باز کردن، پنل‌های باز دیگر بسته می‌شوند تا چند
   کشویی روی هم نیفتد. */
function toggleReportMulti(btn) {
  const box = btn.closest('.ms');
  const panel = box && box.querySelector('.ms-panel');
  if (!panel) return;
  const wasOpen = !panel.hidden;
  document.querySelectorAll('#rbFilters .ms-panel').forEach(p => { p.hidden = true; });
  panel.hidden = wasOpen;
}

/* کلیک بیرون از کشویی، پنل باز را می‌بندد؛ تا پنل روی بقیهٔ صفحه نماند و کاربر
   با یک کلیک به کار خودش برگردد (رفتار آشنای کشویی‌ها). ثبت یک‌بار در سطح سند
   کافی است و برای پنل نبودن هم هزینه‌ای ندارد. */
document.addEventListener('click', ev => {
  if (ev.target.closest && ev.target.closest('.ms')) return;
  document.querySelectorAll('#rbFilters .ms-panel').forEach(p => { p.hidden = true; });
});

/* کلید Esc هم پنل باز را می‌بندد؛ رفتار آشنای کشویی‌ها برای کاربر صفحه‌کلید. */
document.addEventListener('keydown', ev => {
  if (ev.key !== 'Escape') return;
  document.querySelectorAll('#rbFilters .ms-panel').forEach(p => { p.hidden = true; });
});

/* برچسب خلاصهٔ کشویی: تعداد تیک‌خورده‌ها یا «همه». */
function reportMultiLabel(input) {
  const box = input.closest('.ms');
  if (!box) return;
  const picked = box.querySelectorAll('.rbMValue:checked').length;
  const text = box.querySelector('.ms-text');
  if (text) text.textContent = picked ? `${fa(picked)} مورد انتخاب‌شده` : 'همه';
}

/* یک ردیف شرط تازه می‌سازد: قلم + عملگر + مقدار (+ مقدار دوم برای بازه). */
function conditionRowHTML() {
  const src = RB_SOURCES.find(s => s.key === ($('#rbSource') || {}).value);
  const fields = (src && src.fields) || [];
  const fieldOpts = fields.map(f => `<option value="${esc(f.key)}">${esc(f.labelFa)}</option>`).join('');
  const opOpts = RB_OPERATORS.map(o => `<option value="${esc(o.key)}">${esc(o.labelFa)}</option>`).join('');
  return `<div class="rbCond" style="display:flex;gap:6px;align-items:center;flex-wrap:wrap">
    <select class="rbCField" style="min-width:180px">${fieldOpts}</select>
    <select class="rbCOp" onchange="this.parentElement.querySelector('.rbCVal2').style.display = this.value === 'between' ? 'inline-block' : 'none'">${opOpts}</select>
    <input class="rbCVal" placeholder="مقدار" style="min-width:140px">
    <input class="rbCVal2" placeholder="تا" style="min-width:120px;display:none">
    <button class="btn btn-sm btn-ghost" onclick="this.closest('.rbCond').remove()">✖</button>
  </div>`;
}
function addReportCondition() {
  const box = $('#rbConds');
  if (box) box.insertAdjacentHTML('beforeend', conditionRowHTML());
}
function resetReportConditions() {
  const box = $('#rbConds');
  if (box) box.innerHTML = '';
}

function reportSelection() {
  const items = Array.from(document.querySelectorAll('.rbField:checked')).map(c => c.value);
  const filters = {};
  document.querySelectorAll('#rbFilters [id^="rbF_"]').forEach(el => {
    // فیلتر چندانتخابی: ظرف گزینه‌ها است، پس مقدارهای تیک‌خورده با کاما جمع می‌شوند.
    const boxes = el.querySelectorAll('.rbMValue');
    if (boxes.length) {
      const picked = Array.from(el.querySelectorAll('.rbMValue:checked')).map(b => b.value);
      if (picked.length) filters[el.id.slice(4)] = picked.join(',');
      return;
    }
    const v = (el.value || '').trim();
    if (v) filters[el.id.slice(4)] = v;
  });
  const conditions = [];
  document.querySelectorAll('.rbCond').forEach(row => {
    const field = row.querySelector('.rbCField').value;
    const op = row.querySelector('.rbCOp').value;
    const value = (row.querySelector('.rbCVal').value || '').trim();
    const value2 = (row.querySelector('.rbCVal2').value || '').trim();
    if (field && op && (value || op === 'notempty')) conditions.push({field, op, value, value2});
  });
  return {
    source: ($('#rbSource') || {}).value || '',
    items,
    filters,
    conditions,
    limit: Number(($('#rbLimit') || {}).value) || 200,
  };
}

async function runReportBuilder() {
  const req = reportSelection();
  const res = await apiFetch('/api/v1/reports/run', {method: 'POST', body: req});
  const out = $('#rbResult');
  if (!res.ok) { if (out) out.innerHTML = liveErrorBox(res.message); return; }
  const d = res.data;
  RB_LAST = d;
  const head = (d.columns || []).map(c => c.labelFa);
  const rows = (d.rows || []).map(r => `<tr>${r.map((v, i) => {
    const t = (d.columns[i] || {}).type;
    if (t === 'money') return `<td class="num">${money(Number(v) || 0)}</td>`;
    if (t === 'number') return `<td class="num">${esc(v)}</td>`;
    return `<td>${esc(v)}</td>`;
  }).join('')}</tr>`);
  const summary = (d.summary || []).map(s => statCard({ico: 'Σ', tint: '#EFF6FF', accent: '#2563EB', val: money(s.total), label: s.labelFa})).join('');
  if (out) out.innerHTML = reportBrandHeader(d) +
    (summary ? `<div class="grid-stats">${summary}</div>` : '') +
    (d.noteFa ? `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem">ℹ️ ${esc(d.noteFa)}</div>` : '') +
    liveTable(head, rows, d.total);
}

/* ---------------------------------------------------------------------
   سرصفحهٔ برند گزارش — نشان و نام شرکت روی خود گزارش.
   سرصفحه از سرور می‌آید (`brand`)، پس گزارش هر شرکت با هویت خودش دیده و چاپ
   می‌شود و رابط هیچ نام یا نشانی از خودش نمی‌سازد.
   --------------------------------------------------------------------- */

function reportBrandHeader(d) {
  const b = d.brand;
  if (!b) return '';
  const logo = b.logoUrl
    ? `<img src="${esc(b.logoUrl)}" alt="${esc(b.titleFa)}" style="width:52px;height:52px;object-fit:contain;background:#fff;border-radius:10px;flex-shrink:0">`
    : '';
  const meta = [
    b.generatedAt ? `تاریخ گزارش: ${esc(b.generatedAt)}` : '',
    b.generatedBy ? `تهیه‌کننده: ${esc(b.generatedBy)}` : '',
    `تعداد سطر: ${fa(d.total)}`,
  ].filter(Boolean).join(' · ');
  return `<div class="card card-pad" style="margin-bottom:12px;display:flex;align-items:center;gap:12px;border-inline-start:4px solid ${esc(b.brandColor || 'var(--primary)')}">
    ${logo}
    <div style="flex:1">
      <div style="font-weight:700">${esc(d.titleFa || 'گزارش')}</div>
      <div class="muted" style="font-size:.78rem">${esc(b.titleFa)}${b.holdingFa ? ' — ' + esc(b.holdingFa) : ''}${b.shortCode ? ' · کد شرکت ' + esc(b.shortCode) : ''}</div>
    </div>
    <div class="muted" style="font-size:.72rem">${meta}</div>
  </div>`;
}

/**
 * چاپ گزارش با همان سرصفحهٔ برند.
 *
 * خروجی در زبانهٔ تازه ساخته می‌شود و بدون وابستگی به رابط برنامه چاپ (یا ذخیره به
 * PDF) می‌شود؛ نشان شرکت با آدرس مطلق بار می‌شود چون سند در مبدأ دیگری باز می‌شود.
 */
function printReportBuilder() {
  const d = RB_LAST;
  if (!d) { toast('اول گزارش را اجرا کنید', 'error'); return; }
  const b = d.brand || {};
  const abs = u => {
    if (!u) return '';
    if (/^https?:/i.test(u)) return u;
    return (location.origin && location.origin !== 'null' ? location.origin : '') + u;
  };
  const head = (d.columns || []).map(c => `<th>${esc(c.labelFa)}</th>`).join('');
  const rows = (d.rows || []).map(r => `<tr>${r.map((v, i) => {
    const t = (d.columns[i] || {}).type;
    if (t === 'money') return `<td class="num">${money(Number(v) || 0)}</td>`;
    if (t === 'number') return `<td class="num">${esc(v)}</td>`;
    return `<td>${esc(v)}</td>`;
  }).join('')}</tr>`).join('');
  const summary = (d.summary || []).map(s =>
    `<div class="sum"><span>${esc(s.labelFa)}</span><b>${money(s.total)}</b></div>`).join('');
  const logo = b.logoUrl ? `<img class="logo" src="${esc(abs(b.logoUrl))}" alt="${esc(b.titleFa || '')}">` : '';
  const accent = b.brandColor || '#2563EB';
  const html = `<!DOCTYPE html><html lang="fa" dir="rtl"><head><meta charset="utf-8">
<title>${esc(d.titleFa || 'گزارش')} — ${esc(b.titleFa || '')}</title>
<style>
  @font-face{font-family:Vazirmatn;font-style:normal;font-weight:100 900;font-display:swap;
    src:url("${esc(abs('assets/fonts/Vazirmatn-var.woff2'))}") format("woff2-variations")}
  @page{size:A4 landscape;margin:12mm}
  body{font-family:Vazirmatn,Tahoma,'Iranian Sans',sans-serif;color:#1F2937;margin:0}
  header{display:flex;align-items:center;gap:12px;border-bottom:3px solid ${accent};padding-bottom:10px;margin-bottom:14px}
  header .logo{width:58px;height:58px;object-fit:contain;background:#fff;border-radius:10px}
  header h1{font-size:1.05rem;margin:0 0 4px}
  header .sub{font-size:.78rem;color:#6B7280}
  header .meta{margin-inline-start:auto;font-size:.72rem;color:#6B7280;text-align:start;line-height:1.9}
  table{width:100%;border-collapse:collapse;font-size:.78rem}
  th,td{border:1px solid #E5E7EB;padding:6px 8px;text-align:start}
  th{background:#F3F4F6;font-weight:700}
  td.num{font-variant-numeric:tabular-nums;white-space:nowrap}
  tfoot td{background:#FAFAFA;font-weight:700}
  .sums{display:flex;flex-wrap:wrap;gap:10px;margin-top:12px}
  .sum{border:1px solid #E5E7EB;border-radius:8px;padding:6px 12px;font-size:.78rem}
  .sum b{color:${accent};margin-inline-start:8px}
  .note{margin-top:12px;font-size:.75rem;color:#6B7280;border-top:1px dashed #E5E7EB;padding-top:8px}
  footer{margin-top:16px;font-size:.7rem;color:#9CA3AF;border-top:1px solid #E5E7EB;padding-top:8px;text-align:center}
</style></head><body>
<header>${logo}
  <div><h1>${esc(d.titleFa || 'گزارش')}</h1>
    <div class="sub">${esc(b.titleFa || '')}${b.holdingFa ? ' — ' + esc(b.holdingFa) : ''}${b.shortCode ? ' · کد شرکت ' + esc(b.shortCode) : ''}</div></div>
  <div class="meta">${b.generatedAt ? 'تاریخ گزارش: ' + esc(b.generatedAt) + '<br>' : ''}${b.generatedBy ? 'تهیه‌کننده: ' + esc(b.generatedBy) + '<br>' : ''}تعداد سطر: ${fa(d.total)}</div>
</header>
<table><thead><tr>${head}</tr></thead><tbody>${rows}</tbody></table>
${summary ? `<div class="sums">${summary}</div>` : ''}
${d.noteFa ? `<p class="note">${esc(d.noteFa)}</p>` : ''}
<footer>سامانه جامع منابع انسانی — ${esc(b.titleFa || '')} · این گزارش از دادهٔ سامانه تولید شده است.</footer>
<script>window.addEventListener('load',()=>setTimeout(()=>window.print(),350));<\/script>
</body></html>`;
  const url = URL.createObjectURL(new Blob([html], {type: 'text/html;charset=utf-8'}));
  window.open(url, '_blank');
  setTimeout(() => URL.revokeObjectURL(url), 60000);
}

async function saveReportDefinition() {
  const req = reportSelection();
  const name = prompt('نام گزارش را وارد کنید:');
  if (!name) return;
  const res = await apiFetch('/api/v1/reports/records', {method: 'POST', body: {
    kind: 'reportDefs', title: name,
    fields: {source: req.source, items: req.items.join(','), limit: String(req.limit)},
  }});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('تعریف گزارش ذخیره شد');
}

/* =====================================================================
   داشبورد مدیریتی هلدینگ — تجمیع و مقایسهٔ شاخص‌های شرکت‌های زیرمجموعه
   =====================================================================

   این صفحه هیچ قاعده‌ای از خودش ندارد: منبع گزارش «شاخص‌های منابع انسانی شرکت‌های
   هلدینگ» را از سرور می‌خواند و همهٔ سطرها/جمع‌ها را همان‌جا محاسبه می‌شود. این‌جا فقط
   هلدینگ انتخاب می‌شود، گزارش اجرا و به‌صورت مقایسه‌ای (نوار نسبی) نمایش داده می‌شود.
   ===================================================================== */

/* هلدینگهای قابل‌انتخاب این نشست؛ از /api/v1/orgs/holdings می‌آید (غیرمدیر فقط هلدینگ خودش). */
let MG_HOLDINGS = [];
/* آخرین نتیجهٔ داشبورد هلدینگ؛ برای چاپ/PDF دوباره مصرف می‌شود. */
let MG_LAST = null;

function pageMgmtReportsLive() {
  if (!isLive()) return offlineFallback('mgmtReports');
  liveLoad(async () => {
    const [srcRes, holdRes] = await Promise.all([
      apiFetch('/api/v1/reports/sources'),
      apiFetch('/api/v1/orgs/holdings'),
    ]);
    if (!srcRes.ok) return liveErrorBox(srcRes.message);
    const src = (srcRes.data.sources || []).find(s => s.key === 'holdingHr');
    if (!src) return liveErrorBox('منبع گزارش هلدینگ (holdingHr) در سرور ثبت نشده است.');
    MG_HOLDINGS = (holdRes.ok && holdRes.data.holdings) || [];
    if (!MG_HOLDINGS.length) return liveErrorBox('هلدینگی برای این نشست در دسترس نیست.');
    const opts = MG_HOLDINGS.map(h => `<option value="${esc(h.key)}">${esc(h.titleFa)} (${fa((h.companies || []).length)} شرکت)</option>`).join('');
    setTimeout(runHoldingDashboard, 0);
    return `<div class="card card-pad" style="margin-bottom:12px">
      <div class="form-grid">
        <div class="field"><label>هلدینگ</label><select id="mgHolding" onchange="runHoldingDashboard()">${opts}</select></div>
        <div class="field"><label>جست‌وجوی نام شرکت</label><input id="mgCompany" placeholder="بخشی از نام شرکت" oninput="runHoldingDashboard()"></div>
      </div>
      <div class="muted" style="margin-top:8px;font-size:.78rem">شاخص‌های منابع انسانی شرکت‌های زیرمجموعهٔ هلدینگ انتخاب‌شده را کنار هم می‌گذارد و مقایسه می‌کند.</div>
    </div>
    <div id="mgResult">${liveSpinner()}</div>`;
  });
  return liveShell('داشبورد مدیریتی هلدینگ',
    'تجمیع و مقایسهٔ شاخص‌های منابع انسانی شرکت‌های زیرمجموعه بر پایهٔ پروفایل پارامتریک هر شرکت', '');
}

/** اجرای گزارش هلدینگ با فیلترهای انتخاب‌شده و نمایش آن به‌صورت مقایسه‌ای. */
async function runHoldingDashboard() {
  const box = $('#mgResult');
  if (box) box.innerHTML = liveSpinner();
  const holding = ($('#mgHolding') || {}).value || '';
  const company = (($('#mgCompany') || {}).value || '').trim();
  const filters = {};
  if (holding) filters.holding = holding;
  if (company) filters.company = company;
  const res = await apiFetch('/api/v1/reports/run', {
    method: 'POST', body: {source: 'holdingHr', filters, limit: 500},
  });
  if (!box) return;
  if (!res.ok) { box.innerHTML = liveErrorBox(res.message); return; }
  const d = res.data;
  MG_LAST = d;
  const cols = d.columns || [];
  const iHead = cols.findIndex(c => c.key === 'headcount');
  const maxHead = Math.max(1, ...(d.rows || []).map(r => Number(r[iHead]) || 0));

  const stats = (d.summary || []).map(s =>
    statCard({ico: 'Σ', tint: '#EFF6FF', accent: '#2563EB', val: money(s.total), label: s.labelFa})).join('');

  const rows = (d.rows || []).map(r => `<tr>${r.map((v, i) => {
    const t = (cols[i] || {}).type;
    if (i === iHead) {
      const n = Number(v) || 0;
      const pct = Math.round((n / maxHead) * 100);
      return `<td><div class="num">${fa(n)}</div>
        <div style="height:6px;border-radius:4px;background:var(--border);margin-top:4px">
          <div style="height:6px;border-radius:4px;background:var(--primary);width:${pct}%"></div></div></td>`;
    }
    if (t === 'money') return `<td class="num">${money(Number(v) || 0)}</td>`;
    if (t === 'number') return `<td class="num">${esc(v)}</td>`;
    return `<td>${esc(v)}</td>`;
  }).join('')}</tr>`);

  box.innerHTML = reportBrandHeader(d) +
    (stats ? `<div class="grid-stats">${stats}</div>` : '') +
    (d.noteFa ? `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">ℹ️ ${esc(d.noteFa)}</div>` : '') +
    liveTable(cols.map(c => c.labelFa), rows, d.total) +
    `<div style="margin-top:12px"><button class="btn btn-ghost" onclick="printHoldingDashboard()">🖨️ چاپ / PDF</button></div>`;
}

/** چاپ داشبورد هلدینگ با همان سازوکار چاپ گزارش‌ها (سرصفحهٔ برند شرکت فعال). */
function printHoldingDashboard() {
  if (!MG_LAST) { toast('اول دست‌کم یک گزارش هلدینگ بگیرید', 'error'); return; }
  RB_LAST = MG_LAST;
  printReportBuilder();
}

const LIVE_PAGES = {
  mgmtReports: pageMgmtReportsLive,
  roles: pageRolesLive,
  users: pageUsersLive,
  audit: pageAuditLive,
  payReports: pagePayReportsLive,
  payslip: pagePayslipLive,
  contracts: pageContractsLive,
  contractLifecycle: pageContractLifecycleLive,
  contractTemplates: pageContractTemplatesLive,
  dependents: pageDependentsLive,
  flightRequests: pageFlightRequestsLive,
  salaryDecrees: pageSalaryDecreesLive,
  employmentCerts: pageEmploymentCertsLive,
  reportBuilder: pageReportBuilderLive,
};

/* دکمه‌های افزودهٔ صفحه‌های رکوردی (پارامتر حقوق، قالب‌ها). */
const RK_EXTRA = {
  payParams: '<button class="btn btn-ghost" onclick="openPayrollSimulator()">🧮 شبیه‌سازی فیش</button>',
  docTemplates: '<button class="btn btn-ghost" onclick="openTemplatePreview()">👁️ پیش‌نمایش قالب</button>',
};

// آرگومان‌های مسیر (مثل کد کارمند در پروندهٔ کارمند) به همان صفحهٔ زنده پاس می‌شود.
function livePage(page, ...args) { return LIVE_PAGES[page](...args); }
