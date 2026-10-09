/* =====================================================================
   صفحه‌های زندهٔ کارکنان و پروندهٔ کارمند (با تب‌های مشخصات)
   =====================================================================

   فهرست کارکنان و پروندهٔ کارمند روی دادهٔ واقعی سرور کار می‌کنند و پرونده در تب‌ها
   چیده می‌شود: «مشخصات شخصی»، «مشخصات شغلی»، «قرارداد و اسناد»، «حضور و کارکرد».
   هیچ قاعده‌ای اینجا نیست؛ فقط خواندن، فیلتر کردن و چیدن نمایشی.

   احتیاط صادقانه: هر فیلدی که سرور نداشته باشد «—» نشان داده می‌شود، نه مقدار ساختگی.
   ===================================================================== */

/** وضعیت کارمند → چیپ رنگی. */
const EMP_STATUS_CLS = {active: 'c-ok', leave: 'c-wait', probation: 'c-wait', terminated: 'c-rej'};

/** نگاشت کلید نوع استخدام به برچسب فارسی از گزینه‌های سرور. */
function employmentTypeOptions(options) {
  return (options && options.employmentTypes) || [];
}

/* ======================= فهرست کارکنان ======================= */

function pageEmployeesLive() {
  if (!isLive()) return offlineFallback('employees');
  liveLoad(async () => {
    const q = new URLSearchParams({size: '200'});
    if (window.__EMP_QUERY) q.set('query', window.__EMP_QUERY);
    if (window.__EMP_DEPT) q.set('departmentId', window.__EMP_DEPT);
    if (window.__EMP_STATUS) q.set('status', window.__EMP_STATUS);
    const [listRes, optRes] = await Promise.all([
      apiFetch('/api/v1/employee/employees?' + q.toString()),
      apiFetch('/api/v1/employee/options'),
    ]);
    if (!listRes.ok) return liveErrorBox(listRes.message);
    const items = (listRes.data && listRes.data.items) || [];
    const options = optRes.ok ? optRes.data : {};
    window.__EMP_OPTIONS = options;

    const byStatus = {};
    for (const e of items) byStatus[e.status] = (byStatus[e.status] || 0) + 1;
    const totalPayroll = items.reduce((s, e) => s + (Number(e.baseSalary) || 0), 0);

    const stats = `<div class="grid-stats">
      ${statCard({ico: '👥', tint: '#EFF6FF', accent: '#2563EB', val: fa((listRes.data && listRes.data.total) || items.length), label: 'کل کارکنان'})}
      ${statCard({ico: '🟢', tint: '#DCFCE7', accent: '#16A34A', val: fa(byStatus.active || 0), label: 'شاغل'})}
      ${statCard({ico: '🟡', tint: '#FEF3C7', accent: '#D97706', val: fa((byStatus.leave || 0) + (byStatus.probation || 0)), label: 'مرخصی / آزمایشی'})}
      ${statCard({ico: '💰', tint: '#EDE9FE', accent: '#7C3AED', val: money(totalPayroll), label: 'جمع حقوق پایه (ریال)'})}
    </div>`;

    const deptOpts = ((options.departments) || [])
      .map(d => `<option value="${esc(d.id)}" ${window.__EMP_DEPT === d.id ? 'selected' : ''}>${esc(d.name)}</option>`).join('');
    const statusOpts = [['active', 'شاغل'], ['leave', 'مرخصی'], ['probation', 'آزمایشی'], ['terminated', 'پایان خدمت']]
      .map(([v, l]) => `<option value="${v}" ${window.__EMP_STATUS === v ? 'selected' : ''}>${l}</option>`).join('');
    const filters = `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:10px;align-items:flex-end;flex-wrap:wrap">
      <div class="field" style="flex:1;min-width:200px"><label>جست‌وجو (نام یا کد پرسنلی)</label>
        <input id="empQuery" value="${esc(window.__EMP_QUERY || '')}" placeholder="مثلاً سارا یا EMP-001"></div>
      <div class="field"><label>واحد سازمانی</label>
        <select id="empDept"><option value="">همهٔ واحدها</option>${deptOpts}</select></div>
      <div class="field"><label>وضعیت</label>
        <select id="empStatus"><option value="">همه</option>${statusOpts}</select></div>
      <button class="btn btn-primary" onclick="employeesApplyFilter()">🔍 اعمال</button>
      <button class="btn btn-ghost" onclick="employeesClearFilter()">✖ پاک‌کردن</button>
    </div>`;

    const rows = items.map(e => `<tr>
      <td>${personCell(e.id, e.fullName)}</td>
      <td>${esc(e.departmentName || '—')}</td>
      <td>${esc(e.position || '—')}</td>
      <td>${e.employmentTypeFa ? `<span class="chip" style="font-size:.72rem">${esc(e.employmentTypeFa)}</span>` : esc(e.contractType || '—')}</td>
      <td>${faDate(e.hireDate)}</td>
      <td><span class="num">${money(e.baseSalary || 0)}</span></td>
      <td>${faNum(e.leaveBalance || 0)}</td>
      <td>${pChip(EMP_STATUS_CLS[e.status] || 'c-wait', empStatusFa(e.status))}</td>
      <td class="ltr num" style="font-size:.72rem">${esc(e.nationalId || '—')}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-primary" onclick="go('profile','${esc(e.id)}')">پرونده</button>
        <button class="btn btn-sm btn-ghost" title="گواهی اشتغال" onclick="liveOpenDoc('employment_certificate','${esc(e.id)}')">📄</button>
        <button class="btn btn-sm btn-ghost" title="فیش حقوقی" onclick="liveOpenDoc('payslip','${esc(e.id)}')">🧾</button>
      </div></td></tr>`);

    return stats + filters + statNote('فهرست روی محدودهٔ دسترسی نقش شما خوانده می‌شود: مدیر سیستم همهٔ کارکنان را می‌بیند و نقش‌های دیگر بر پایهٔ سیاست سرور. فیلترها روی سرور اعمال می‌شوند.') +
      liveTable(['کارمند', 'واحد سازمانی', 'پست', 'نوع استخدام', 'تاریخ استخدام', 'حقوق پایه (ریال)', 'مانده مرخصی', 'وضعیت', 'کد ملی', 'عملیات'],
        rows, (listRes.data && listRes.data.total) || items.length);
  });
  return liveShell('فهرست کارکنان', 'کارکنان این استقرار روی دادهٔ واقعی سرور — با پروندهٔ تبیندی برای هر کارمند',
    `<button class="btn btn-ghost" onclick="toast('خروجی اکسل در فاز اتصال به مالی فعال می‌شود','info')">⬇️ خروجی</button>`);
}

/** برچسب فارسی وضعیت کارمند. */
function empStatusFa(status) {
  const map = {active: 'شاغل', leave: 'مرخصی', probation: 'آزمایشی', terminated: 'پایان خدمت'};
  return map[status] || status || '—';
}

/** اعمال فیلترهای فهرست کارکنان — همان درخواست، این بار با پارامترهای فیلتر سرور. */
function employeesApplyFilter() {
  window.__EMP_QUERY = (document.getElementById('empQuery') || {}).value || '';
  window.__EMP_DEPT = (document.getElementById('empDept') || {}).value || '';
  window.__EMP_STATUS = (document.getElementById('empStatus') || {}).value || '';
  go('employees');
}

function employeesClearFilter() {
  window.__EMP_QUERY = ''; window.__EMP_DEPT = ''; window.__EMP_STATUS = '';
  go('employees');
}

/* ======================= پروندهٔ کارمند (تبیندی) ======================= */

/** تب فعال پروندهٔ کارمند زنده. */
let profileLiveTab = 'personal';

/** پروندهٔ کارمند با چهار تب مشخصات — هر تب دادهٔ واقعی خودش را می‌خواند. */
function pageProfileLive(id) {
  if (!isLive()) return offlineFallback('profile');
  if (!id) {
    return pageHead('پرونده کارمند', 'کارمندی انتخاب نشده است') +
      `<div class="card card-pad" style="line-height:2">
        برای دیدن پرونده، از <a href="#" onclick="go('employees');return false" style="color:var(--primary)">فهرست کارکنان</a> روی «پرونده» بزنید.
      </div>`;
  }

  liveLoad(async () => {
    const [empRes, conRes, attRes, attSchemaRes] = await Promise.all([
      apiFetch('/api/v1/employee/employees/' + encodeURIComponent(id)),
      apiFetch('/api/v1/contracts/contracts?size=100&employeeId=' + encodeURIComponent(id)),
      apiFetch('/api/v1/attendance/records?size=100&employeeId=' + encodeURIComponent(id)),
      apiFetch('/api/v1/attendance/schema'),
    ]);
    if (!empRes.ok) return liveErrorBox(empRes.message);
    const e = empRes.data || {};
    const contracts = (conRes.ok && (conRes.data.items || [])) || [];
    const attItems = (attRes.ok && (attRes.data.items || [])) || [];
    // برچسب فارسی فیلدها از شناسهٔ سرور می‌آید؛ هیچ برچسبی در رابط ساخته نمی‌شود.
    const fieldLabels = {};
    for (const kind of (attSchemaRes.ok && (attSchemaRes.data.kinds || [])) || []) {
      fieldLabels[kind.key] = {};
      for (const f of kind.fields || []) fieldLabels[kind.key][f.key] = f.labelFa || f.key;
    }

    const tabs = [['personal', 'مشخصات شخصی'], ['job', 'مشخصات شغلی'], ['docs', 'قرارداد و اسناد'], ['time', 'حضور و کارکرد']];
    const tabBar = `<div class="tabs">${tabs.map(([k, l]) =>
      `<button class="tab ${profileLiveTab === k ? 'active' : ''}" onclick="profileLiveTab='${k}';go('profile','${esc(id)}')">${l}</button>`).join('')}</div>`;

    const head = `<div class="card card-pad" style="margin-bottom:12px">
      <div class="profile-head">
        <div class="avatar lg">${esc(initial(e.fullName || '؟'))}</div>
        <div>
          <div class="profile-name">${esc(e.fullName || '—')}</div>
          <div class="profile-job">${esc(e.position || '—')} — ${esc(e.departmentName || '—')}</div>
          <div class="profile-tags">
            ${pChip(EMP_STATUS_CLS[e.status] || 'c-wait', empStatusFa(e.status))}
            <span class="chip contract">${esc(e.contractType || '—')}</span>
            ${e.employmentTypeFa ? `<span class="chip">${esc(e.employmentTypeFa)}</span>` : ''}
            <span class="chip c-done num">${esc(e.id)}</span>
          </div>
        </div>
      </div>
    </div>`;

    const kv = pairs => `<dl class="kv">${pairs.map(([k, v]) =>
      `<dt>${esc(k)}</dt><dd>${v}</dd>`).join('')}</dl>`;

    const personal = kv([
      ['نام و نام خانوادگی', esc(e.fullName || '—')],
      ['کد پرسنلی', `<span class="ltr num">${esc(e.id)}</span>`],
      ['کد ملی', e.nationalId ? `<span class="ltr num">${esc(e.nationalId)}</span>` : '—'],
      ['تلفن همراه', e.phone ? `<span class="ltr num">${esc(e.phone)}</span>` : '—'],
      ['ایمیل', e.email ? `<span class="ltr">${esc(e.email)}</span>` : '—'],
      ['مدیر مستقیم', esc(e.managerName || '—')],
    ]) + statNote('سایر مشخصات هویتی و مدارک (شناسنامه، مدارک تحصیلی، آدرس) در ماژول «بایگانی و شناسایی کارکنان» ثبت می‌شود؛ اینجا فقط فیلدهایی نشان داده می‌شود که سرور کارکنان دارد.');

    const job = kv([
      ['واحد سازمانی', esc(e.departmentName || '—')],
      ['پست سازمانی', esc(e.position || '—')],
      ['نوع قرارداد', esc(e.contractType || '—')],
      ['نوع استخدام', esc(e.employmentTypeFa || e.employmentType || '—')],
      ['وضعیت', empStatusFa(e.status)],
      ['مدیر مستقیم', esc(e.managerName || '—')],
      ['تاریخ استخدام', `<span class="num">${esc(e.hireDate || '—')}</span>`],
      ['حقوق پایه', `<span class="num">${money(e.baseSalary || 0)}</span> ریال`],
      ['مانده مرخصی', `<span class="num">${fa(e.leaveBalance || 0)}</span> روز`],
    ]) + `<div class="row-actions" style="margin-top:10px">
      <button class="btn btn-ghost" onclick="liveOpenDoc('salary_decree','${esc(e.id)}')">📜 حکم کارگزینی</button>
      <button class="btn btn-ghost" onclick="liveOpenDoc('employment_certificate','${esc(e.id)}')">📄 گواهی اشتغال</button>
      <button class="btn btn-ghost" onclick="liveOpenDoc('payslip','${esc(e.id)}')">🧾 فیش حقوقی</button>
    </div>`;

    const docRows = contracts.map(c => `<tr>
      <td class="num ltr">${esc(c.id)}</td>
      <td>${pChip(c.kind === 'permanent' ? 'c-ok' : 'c-wait', c.kindFa || c.kind)}</td>
      <td>${faDate(c.startDate)}</td>
      <td>${faDate(c.endDate)}</td>
      <td>${statusChip(c.status, c.statusFa)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="liveOpenDoc('contract','${esc(c.id)}')">🖨️ متن قرارداد</button></td></tr>`);
    const docs = liveTable(['شماره', 'نوع', 'شروع', 'پایان', 'وضعیت', 'سند'], docRows, docRows.length) +
      (contracts.length ? '' : statNote('قراردادی برای این کارمند ثبت نشده است؛ قرارداد تازه از ماژول «قراردادهای کارکنان» افزوده می‌شود.'));

    const kindFa = {attendanceRecords: 'تردد روزانه', leaveRequests: 'مرخصی', missions: 'مأموریت', overtime: 'اضافه‌کاری'};
    const timeRows = attItems.map(r => `<tr>
      <td class="num ltr">${esc(r.id)}</td>
      <td>${esc(kindFa[r.kind] || r.kind)}</td>
      <td>${esc(r.title || '—')}</td>
      <td style="font-size:.74rem">${Object.keys(r.fields || {}).map(k =>
        `<span class="muted">${esc((fieldLabels[r.kind] || {})[k] || k)}: ${esc((r.fields || {})[k])}</span>`).join(' · ') || '—'}</td>
      <td>${statusChip(r.status, r.statusFa)}</td></tr>`);
    const time = liveTable(['شناسه', 'نوع', 'عنوان', 'فیلدها', 'وضعیت'], timeRows, timeRows.length) +
      statNote('این رکوردها از ماژول «حضور و غیاب، مرخصی و مأموریت» خوانده می‌شود و مبنای کارکرد دوره و محاسبهٔ حقوق است.');

    const bodies = {personal, job, docs, time};
    const body = `<div class="card"><div class="card-pad">${tabBar}${bodies[profileLiveTab] || personal}</div></div>`;

    return head + body;
  });

  return liveShell('پرونده کارمند', 'پروندهٔ تبیندی: مشخصات شخصی، مشخصات شغلی، قرارداد و اسناد، حضور و کارکرد',
    `<button class="btn btn-ghost" onclick="go('employees')">‹ فهرست کارکنان</button>`);
}

LIVE_PAGES.employees = pageEmployeesLive;
LIVE_PAGES.profile = pageProfileLive;
