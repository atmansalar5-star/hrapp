/* =====================================================================
   «پردازش حقوق» — لینک حقوق به کارکرد
   =====================================================================

   این صفحه اتصال حقوق به کارکرد را نشان می‌دهد: کارکرد هر کارمند در دوره از سرور
   می‌آید و مبلغ هر ردیف با موتور قواعد همان‌جا محاسبه می‌شود. رابط هیچ مبلغی
   نمی‌سازد؛ فقط نمایش می‌دهد و دستور «اجرای محاسبه» را می‌فرستد.
   ===================================================================== */

/** دورهٔ انتخاب‌شده برای نمایش کارکرد. */
window.__PAY_PERIOD = window.__PAY_PERIOD || '';

function pagePayrollRunLive() {
  if (!isLive()) return offlineFallback('payrollRun');
  liveLoad(async () => {
    const [runsRes, periodsRes, schemaRes] = await Promise.all([
      apiFetch('/api/v1/payroll/records?kind=payrollRuns&size=50'),
      apiFetch('/api/v1/payroll/worktime-periods'),
      apiFetch('/api/v1/payroll/schema'),
    ]);
    if (!runsRes.ok) return liveErrorBox(runsRes.message);
    const runs = (runsRes.data && runsRes.data.items) || [];
    // اقدام‌های گردش‌کار دورهٔ پردازش از شناسهٔ ماژول می‌آید (هیچ مرحله‌ای در UI ساخته نمی‌شود).
    const runKind = (schemaRes.ok && ((schemaRes.data.kinds || []).find(k => k.key === 'payrollRuns'))) || null;
    const periods = (periodsRes.ok && (periodsRes.data.periods || [])) || [];
    const serverPeriod = (periodsRes.ok && periodsRes.data.serverPeriod) || '';

    // دورهٔ پیش‌فرض: دورهٔ انتخابی کاربر، وگرنه ماه جاری سرور، وگرنه تازه‌ترین دورهٔ کارکرد.
    if (!window.__PAY_PERIOD) {
      const runPeriods = runs.map(r => (r.fields || {}).period).filter(Boolean);
      window.__PAY_PERIOD = periods.includes(serverPeriod) ? serverPeriod : (periods[0] || runPeriods[0] || serverPeriod);
    }
    const period = window.__PAY_PERIOD;

    const wtRes = await apiFetch('/api/v1/payroll/worktime' + (period ? '?period=' + encodeURIComponent(period) : ''));
    if (!wtRes.ok) return liveErrorBox(wtRes.message);
    const wt = wtRes.data || {};
    const rows = wt.rows || [];
    const sum = wt.sum || {};

    const periodOpts = [...new Set([...periods, ...runs.map(r => (r.fields || {}).period).filter(Boolean), serverPeriod].filter(Boolean))]
      .map(p => `<option value="${esc(p)}" ${p === period ? 'selected' : ''}>${esc(p)}</option>`).join('');

    const label = p => {
      const [y, m] = String(p || '').split('/');
      const names = ['', 'فروردین', 'اردیبهشت', 'خرداد', 'تیر', 'مرداد', 'شهریور', 'مهر', 'آبان', 'آذر', 'دی', 'بهمن', 'اسفند'];
      return names[Number(m)] ? `${names[Number(m)]} ${fa(y)}` : p;
    };

    const stats = `<div class="grid-stats">
      ${statCard({ico: '💵', tint: '#EFF6FF', accent: '#2563EB', val: money(sum.net || 0), label: `خالص دورهٔ ${label(period)} (ریال)`})}
      ${statCard({ico: '🧾', tint: '#EDE9FE', accent: '#7C3AED', val: money(sum.gross || 0), label: 'جمع ناخالص (ریال)'})}
      ${statCard({ico: '⏱️', tint: '#FEF3C7', accent: '#D97706', val: fa(sum.overtimeHours || 0), label: 'اضافه‌کاری تأییدشده (ساعت)'})}
      ${statCard({ico: '➖', tint: '#FEE2E2', accent: '#DC2626', val: money(sum.unpaidDeduction || 0), label: 'کسر مرخصی بی‌حقوق و غیبت (ریال)'})}
    </div>`;

    const periodPicker = `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:10px;align-items:flex-end;flex-wrap:wrap">
      <div class="field"><label>دورهٔ پرداخت (کارکرد)</label>
        <select id="payPeriod" onchange="payrollPickPeriod(this.value)">${periodOpts || '<option value="">—</option>'}</select></div>
      <span class="muted" style="font-size:.78rem">دورهٔ کارکرد از رکوردهای حضور می‌آید؛ ماه جاری سرور: <b class="num">${esc(serverPeriod)}</b></span>
    </div>`;

    const runRows = runs.map(r => {
      const f = r.fields || {};
      const editable = r.status === 'draft';
      // اقدام‌های مجاز در وضعیت کنونی، بر پایهٔ انتقال‌های اعلامی دامنه.
      const runActions = runKind ? (runKind.actions || [])
        .filter(a => !a.to || ((runKind.transitions || {})[r.status] || []).includes(a.to))
        .map(a => a.value === 'calculate'
          // «اجرای محاسبه» ابتدا مبلغ را از کارکرد می‌نویسد و سپس مرحله را جلو می‌برد.
          ? `<button class="btn btn-sm btn-primary" onclick="payrollCalculateAndAdvance('${esc(r.id)}','${esc(a.value)}',${a.noteRequired ? 'true' : 'false'},'${esc(a.labelFa)}')">${esc(a.labelFa)} ←</button>`
          : `<button class="btn btn-sm ${a.style === 'primary' ? 'btn-primary' : 'btn-ghost'}" onclick="rkAct('/api/v1/payroll','${esc(r.id)}','${esc(a.value)}',${a.noteRequired ? 'true' : 'false'},'${esc(a.labelFa)}')">${esc(a.labelFa)}</button>`).join('') : '';
      return `<tr>
        <td class="num ltr">${esc(r.id)}</td>
        <td>${esc(r.title)}</td>
        <td class="num">${esc(f.period || '—')}</td>
        <td><span class="num">${money(f.gross || 0)}</span></td>
        <td><span class="num">${money(f.tax || 0)}</span></td>
        <td><span class="num">${money(f.insurance || 0)}</span></td>
        <td><span class="num">${money(f.net || 0)}</span></td>
        <td>${statusChip(r.status, r.statusFa)}</td>
        <td><div class="row-actions">
          <button class="btn btn-sm btn-ghost" title="فقط محاسبه و پیش‌نمایش" onclick="payrollCalculate('${esc(r.id)}')">🧮 محاسبه از کارکرد</button>
          ${runActions}
          ${editable ? '' : '<span class="muted" style="font-size:.72rem">ویرایش قفل است</span>'}
        </div></td></tr>`;
    });

    const wtRows = rows.map(r => `<tr>
      <td>${personCell(r.employeeId, r.employeeName)}</td>
      <td>${esc(r.departmentName || '—')}</td>
      <td class="num">${fa(r.workedHours)}</td>
      <td class="num">${fa(r.overtimeHours)}${r.pendingOvertimeHours ? ` <span class="muted">(${fa(r.pendingOvertimeHours)} در گردش)</span>` : ''}</td>
      <td class="num">${fa(r.paidLeaveDays)}${r.pendingLeaveDays ? ` <span class="muted">(${fa(r.pendingLeaveDays)} در گردش)</span>` : ''}</td>
      <td class="num">${fa(r.unpaidLeaveDays)}</td>
      <td class="num">${fa(r.missionDays)}</td>
      <td class="num">${fa(Math.round(r.presenceRatio * 100))}٪</td>
      <td class="num">${money(r.baseSalary)}</td>
      <td class="num">${money(r.overtimePay)}</td>
      <td class="num">${money(r.unpaidDeduction)}</td>
      <td class="num"><b>${money(r.net)}</b></td></tr>`);

    return stats + periodPicker +
      statNote(wt.noteFa || '') +
      statNote('هستهٔ پیوند: کارکرد کارمند (ساعت تردد، اضافه‌کاری و مرخصی تأییدشده) پیش از پرداخت به موتور حقوق داده می‌شود. با تأیید یک رکورد اضافه‌کاری یا مرخصی، مبلغ همین جدول بی‌درنگ تغییر می‌کند.') +
      statNote(runKind ? `چرخهٔ گردش‌کار دورهٔ پردازش: ${(runKind.statuses || []).map(s => esc(s.labelFa)).join(' ← ')} — اقدام‌های هر مرحله در ستون عملیات است؛ «اجرای محاسبه» فقط مبلغ را می‌نویسد و وضعیت را تغییر نمی‌دهد.` : '') +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">دوره‌های پردازش حقوق</div>
        ${liveTable(['شماره', 'عنوان', 'دوره', 'جمع ناخالص', 'مالیات', 'بیمه', 'خالص', 'وضعیت', 'عملیات'], runRows, runRows.length)}
      </div>
      <div id="payResult"></div>
      <div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">کارکرد دوره به تفکیک کارمند (${esc(label(period))})</div>
        ${liveTable(['کارمند', 'واحد', 'ساعت کارکرد', 'اضافه‌کاری (ساعت)', 'مرخصی پرداختی (روز)', 'مرخصی بی‌حقوق (روز)', 'مأموریت (روز)', 'حضور', 'مزد پایه (ریال)', 'اضافه‌کاری (ریال)', 'کسر (ریال)', 'خالص (ریال)'], wtRows, rows.length)}
      </div>`;
  });
  return liveShell('پردازش حقوق', 'اجرای دورهٔ حقوق بر پایهٔ کارکرد واقعی کارکنان — محاسبه با موتور قواعد سرور');
}

/** انتخاب دورهٔ کارکرد. */
function payrollPickPeriod(period) {
  window.__PAY_PERIOD = period || '';
  go('payrollRun');
}

/** اجرای محاسبهٔ دوره از کارکرد روی سرور. true یعنی محاسبه انجام شد. */
async function payrollCalculate(runId) {
  const box = document.getElementById('payResult');
  if (box) box.innerHTML = '<div class="card card-pad muted">⏳ در حال محاسبه از کارکرد…</div>';
  const res = await apiFetch('/api/v1/payroll/runs/' + encodeURIComponent(runId) + '/calculate', {method: 'POST'});
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    toast(res.message, 'error');
    return false;
  }
  const d = res.data || {};
  const s = d.sum || {};
  if (box) {
    box.innerHTML = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">نتیجهٔ محاسبهٔ ${esc(runId)} — دورهٔ ${esc(d.period || '')}</div>
      <div class="grid-stats" style="margin-bottom:0">
        ${statCard({ico: '👥', tint: '#EFF6FF', accent: '#2563EB', val: fa(s.count || 0), label: 'کارمند محاسبه‌شده'})}
        ${statCard({ico: '🧾', tint: '#EDE9FE', accent: '#7C3AED', val: money(s.gross || 0), label: 'جمع ناخالص (ریال)'})}
        ${statCard({ico: '💵', tint: '#DCFCE7', accent: '#16A34A', val: money(s.net || 0), label: 'جمع خالص (ریال)'})}
        ${statCard({ico: '⏱️', tint: '#FEF3C7', accent: '#D97706', val: fa(s.overtimeHours || 0), label: 'اضافه‌کاری (ساعت)'})}
      </div>
      <div class="muted" style="font-size:.78rem;margin-top:10px">
        ${d.applied ? '✅ مبالغ روی رکورد دوره نوشته شد.' : '⚠️ ' + esc(d.applyNoteFa || 'مبالغ نوشته نشد.')}
        برای تغییر وضعیت دوره به «محاسبه‌شده»، از اقدام گردش‌کار روی رکورد استفاده کنید.
      </div>
    </div>`;
  }
  toast('محاسبه از کارکرد انجام شد');
  // جدول دوره‌ها با دکمهٔ «↻» بازخوانی می‌شود تا نتیجهٔ همین محاسبه ناپدید نشود.
  return true;
}

/** اجرای محاسبه از کارکرد و سپس انجام اقدام گردش‌کار («اجرای محاسبه» روی رکورد). */
async function payrollCalculateAndAdvance(runId, action, noteRequired, labelFa) {
  const done = await payrollCalculate(runId);
  if (!done) return;
  let note = '';
  if (noteRequired) {
    note = prompt('دلیل «' + labelFa + '» را وارد کنید:') || '';
    if (!note) return;
  }
  const res = await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(runId) + '/action',
    {method: 'POST', body: {action, noteFa: note}});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('«' + labelFa + '» انجام شد');
  go('payrollRun');
}

LIVE_PAGES.payrollRun = pagePayrollRunLive;
