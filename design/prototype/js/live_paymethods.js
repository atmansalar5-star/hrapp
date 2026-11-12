/* =====================================================================
   روش پرداخت کارکنان — کنترل‌پنل تعیین مدل محاسبهٔ پرداخت هر کارمند
   =====================================================================

   هیچ قاعده‌ای اینجا نیست: فهرست روش‌ها، وضعیت مدل ۴پی و کارکنان با روش جاری‌شان از
   سرور می‌آید (`/api/v1/payroll/pay-method-options` و `/pay-methods`). این انتخاب
   محاسبهٔ حقوق موجود را عوض نمی‌کند؛ فقط تعیین می‌کند کارشناس کدام فرم و کدام موتور
   محاسبه را برای این کارمند به‌کار ببرد.
   ===================================================================== */

/** وضعیت بارگذاری‌شدهٔ صفحهٔ روش پرداخت (کارمند انتخاب‌شده و عبارت جستجو). */
const PAY_METHOD_STATE = {employeeId: '', query: ''};

function pagePayMethodsLive() {
  if (!isLive()) return offlineFallback('payMethods');
  liveLoad(async () => {
    const q = PAY_METHOD_STATE.query ? '&query=' + encodeURIComponent(PAY_METHOD_STATE.query) : '';
    const res = await apiFetch('/api/v1/payroll/pay-methods?x=1' + q);
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    const items = d.items || [];
    const methods = d.methods || [];
    const rows = items.map(row => {
      const options = methods.map(m =>
        `<option value="${esc(m.key)}" ${m.key === row.payModel ? 'selected' : ''}>${esc(m.titleFa)}</option>`).join('');
      return `<tr>
        <td class="num ltr">${esc(row.employeeId)}</td>
        <td>${esc(row.fullName)}<div class="muted" style="font-size:.68rem">${esc(row.departmentName || '')}</div></td>
        <td>${pChip(row.sourceFa === 'ثبت‌شده' ? 'c-ok' : 'c-wait', row.payModelFa)}</td>
        <td>${row.effectiveDate ? faDate(row.effectiveDate) : '—'}</td>
        <td class="muted" style="font-size:.74rem">${esc(row.noteFa || row.sourceFa)}</td>
        <td><select id="pmModel_${esc(row.employeeId)}" style="min-width:190px">${options}</select></td>
        <td><button class="btn btn-sm btn-primary" onclick="payMethodSave('${esc(row.employeeId)}')">💾 ثبت</button></td>
      </tr>`;
    });

    const stats = `<div class="grid-stats" style="margin-bottom:12px">
      ${statCard({ico: '👥', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.length), label: 'کارمند بررسی‌شده'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A',
        val: fa(items.filter(r => r.sourceFa === 'ثبت‌شده').length), label: 'روش ثبت‌شده'})}
      ${statCard({ico: '🛢️', tint: '#FEF3C7', accent: '#D97706',
        val: fa(items.filter(r => r.payModel === 'oil').length), label: 'صندوق نفت'})}
      ${statCard({ico: '🧪', tint: '#EDE9FE', accent: '#7C3AED',
        val: fa(items.filter(r => r.payModel === 'fourp').length), label: 'مدل ۴پی (توسعه‌ای)'})}
    </div>`;

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      <b>${d.fourPApplied ? '✅' : '🧪'} وضعیت مدل ۴پی در این استقرار:</b>
      ${d.fourPEnabled ? 'فعال' : 'غیرفعال'} — «اعمال در پرداخت» ${d.fourPApplied ? 'روشن' : 'خاموش'}.
      <div class="muted" style="font-size:.78rem">${esc(d.independenceNoteFa || '')}</div>
    </div>`;

    const search = `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:8px;align-items:center;flex-wrap:wrap">
      <input class="inp w-lg" id="pmSearch" value="${esc(PAY_METHOD_STATE.query)}" placeholder="🔍 جستجوی کارمند (نام یا کد پرسنلی)">
      <button class="btn btn-primary btn-sm" onclick="payMethodSearch()">جستجو</button>
      <button class="btn btn-ghost btn-sm" onclick="payMethodSearchClear()">✖ پاک‌کردن</button>
    </div>`;

    return note + stats + search +
      tbl(['کد', 'کارمند', 'روش جاری', 'تاریخ اجرا', 'دلیل/منبع', 'روش تازه', 'عملیات'], rows, {count: items.length});
  });
  return liveShell('روش پرداخت کارکنان', 'تعیین مدل محاسبهٔ پرداخت هر کارمند — بدون تغییر در محاسبهٔ جاری');
}

/** اعمال جستجوی سروری کارکنان. */
function payMethodSearch() {
  const el = document.getElementById('pmSearch');
  PAY_METHOD_STATE.query = el ? el.value.trim() : '';
  liveReload();
}

function payMethodSearchClear() {
  PAY_METHOD_STATE.query = '';
  liveReload();
}

/** ثبت روش پرداخت انتخاب‌شده برای یک کارمند. */
async function payMethodSave(employeeId) {
  const select = document.getElementById('pmModel_' + employeeId);
  const payModel = select ? select.value : '';
  if (!payModel) { toast('روش پرداخت را انتخاب کنید', 'error'); return; }
  const reason = prompt('دلیل انتخاب یا تغییر روش پرداخت (اختیاری):') || '';
  const res = await apiFetch('/api/v1/payroll/pay-methods', {
    method: 'POST',
    body: {employeeId: employeeId, payModel: payModel, reasonFa: reason},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  const note = (res.data || {}).noteFa;
  toast(note || 'روش پرداخت ثبت شد', note ? 'info' : 'success');
  liveReload();
}

LIVE_PAGES.payMethods = pagePayMethodsLive;