/* =====================================================================
   داشبورد زنده — چهار بخش داشبورد با دادهٔ واقعی سرور
   =====================================================================

   چرا این فایل: صفحهٔ «داشبورد» پیشتر فقط نسخهٔ نمایشی داشت، پس در حالت اتصال به
   سرور هم عددهای نمونه نشان میداد؛ در حالی که سرور همین حالا اینها را دارد:

     • GET /api/v1/dashboard/widgets      → کارتهای جمعیت کارکنان (بر پایهٔ نقش و محدودهٔ داده)
     • GET /api/v1/statistics/dashboard   → توزیع واحدها، جرم حقوقی، حضور و صف تأیید
     • صفهای کاری هر ماژول (مرخصی، درخواست نیرو، دورهٔ پردازش، دورهٔ ارزیابی)

   این فایل هیچ قاعدهای نمیسازد: فقط همان عددها را میخواند، شکل میدهد و نشان
   میدهد. وقتی سرور در دسترس نیست (`!isLive()`)، همان داشبورد نمایشی بخش ۵
   (pageDashboard) برگردانده میشود تا ارائهٔ بدون سرور دستنخورده بماند.
   ===================================================================== */

/** آیکون و رنگ هر کارت آماری؛ برچسب و عدد از خود سرور میآید. */
const DASH_WIDGET_UI = {
  headcount:  {ico: '👥', tint: '#EFF6FF', accent: '#2563EB', page: 'employees'},
  active:     {ico: '🟢', tint: '#DCFCE7', accent: '#16A34A', page: 'employees'},
  leave:      {ico: '🏖️', tint: '#EDE9FE', accent: '#7C3AED', page: 'leaveApprove'},
  probation:  {ico: '🧪', tint: '#FEF3C7', accent: '#D97706', page: 'employees'},
  terminated: {ico: '🚪', tint: '#FEE2E2', accent: '#DC2626', page: 'employees'},
  own_record: {ico: '🪪', tint: '#E0F2FE', accent: '#0284C7', page: 'profile'},
};

/** شمار کل یک فهرست رکوردی بدون خواندن همهٔ سطرها (size=1 و total سرور). */
async function apiTotal(path) {
  const res = await apiFetch(path + (path.includes('?') ? '&' : '?') + 'size=1');
  if (!res.ok || !res.data) return null;
  if (typeof res.data.total === 'number') return res.data.total;
  return (res.data.items || []).length;
}

/** بدنهٔ زندهٔ داشبورد: چهار بخش، همه از دادهٔ سرور. */
async function dashLiveBody() {
  // پنجرهٔ درخواستها: همهٔ عددهای داشبورد یکجا خوانده میشوند تا صفحه یکبار بسازد.
  const [stats, widgets, leaves, recentLeaves, hiring, payrollDraft, perfOpen] = await Promise.all([
    apiFetch('/api/v1/statistics/dashboard'),
    apiFetch('/api/v1/dashboard/widgets'),
    apiTotal('/api/v1/attendance/records?kind=leaveRequests&status=pending'),
    apiFetch('/api/v1/attendance/records?kind=leaveRequests&size=5'),
    apiTotal('/api/v1/recruitment/records?kind=hiringRequests&status=pending'),
    apiTotal('/api/v1/payroll/records?kind=payrollRuns&status=draft'),
    apiTotal('/api/v1/performance/records?kind=perfCycles&status=open'),
  ]);

  const empty = '<span class="muted">—</span>';
  const num = (n) => (typeof n === 'number' ? fa(n) : empty);
  const s = stats.ok ? stats.data || {} : null;
  const head = s && s.headcount ? s.headcount : {};

  /* ---- کارت اقدام: همان قالب داشبورد نمایشی، با عدد واقعی ---- */
  const act = (cls, ico, title, note, n, page, unitFa) => `
    <div class="act ${cls}" onclick="go('${page}')">
      <div class="act-ico">${ico}</div>
      <div style="min-width:0"><b>${title}</b><div class="muted">${note}</div></div>
      <div class="act-num num">${num(n)}<span class="muted" style="font-size:.6rem"> ${unitFa}</span></div>
    </div>`;

  const todo = [
    act(leaves ? 'urgent' : 'ok', '🌤️', 'تأیید مرخصی',
      leaves ? 'درخواست در انتظار بررسی' : 'صف خالی است', leaves, 'leaveApprove', 'درخواست'),
    act(hiring ? 'soon' : 'ok', '📝', 'درخواست نیرو',
      hiring ? 'در انتظار تصویب' : 'صف خالی است', hiring, 'hiringReqs', 'درخواست'),
    act(payrollDraft ? 'soon' : 'ok', '🧾', 'پردازش حقوق',
      payrollDraft ? 'دورهٔ پیشنویس' : 'همهٔ دورهها بسته است', payrollDraft, 'payrollRun', 'دوره'),
    act(perfOpen ? 'soon' : 'ok', '⭐', 'ارزیابی عملکرد',
      perfOpen ? 'دورهٔ در حال اجرا' : 'دورهٔ بازی نیست', perfOpen, 'perfCycles', 'دوره'),
  ].join('');

  /* ---- کارتهای آماری: برچسب و عدد از /dashboard/widgets ---- */
  const widgetCards = widgets.ok && (widgets.data || []).length
    ? widgets.data.map((w) => {
      const ui = DASH_WIDGET_UI[w.key] || {ico: '📊', tint: '#F1F5F9', accent: '#475569'};
      const val = `${fa(w.value)}${w.unit ? `<span class="muted" style="font-size:.6rem"> ${esc(w.unit)}</span>` : ''}`;
      return statCard({ico: ui.ico, tint: ui.tint, accent: ui.accent, val,
        label: esc(w.titleFa), link: ui.page ? 'مشاهده' : '', page: ui.page});
    }).join('')
    : `<div class="card card-pad muted">کارت آماری برنگشت — ماژول کارکنان در این استقرار فعال نیست یا نقش شما محدودهٔ دادهٔ سازمانی ندارد.</div>`;

  /* ---- توزیع واحدها: همان میلههای داشبورد نمایشی، با عدد واقعی ---- */
  const dept = (s && Array.isArray(s.byDept)) ? s.byDept.filter((d) => d.count > 0) : [];
  const maxD = Math.max(...dept.map((d) => d.count), 1);
  const deptBars = dept.length
    ? dept.slice(0, 6).map((d, i) => `
      <div class="bar-row"><span class="bar-lbl">${esc(d.titleFa)}</span>
        <span class="bar-val num">${fa(d.count)} نفر</span></div>
      <div class="progress" style="margin-bottom:9px"><span
        style="width:${Math.round(d.count / maxD * 100)}%;background:${i === 0 ? 'var(--primary)' : '#93C5FD'}"></span></div>`).join('')
    : `<div class="muted">توزیع واحدها برنگشت.</div>`;

  /* ---- جرم حقوقی و حضور: همه از نمای تجمیعی سرور ---- */
  const att = (s && s.attendance) || {};
  const kv = (label, value) => `<div class="kpi-row"><span>${label}</span><span class="num">${value}</span></div>`;
  const moneyRow = (n) => (typeof n === 'number' && n > 0 ? `${money(n)} <span class="muted" style="font-size:.6rem">ریال</span>` : empty);
  const kpis = s
    ? `${kv('جمع حقوق پایه ماهانه', moneyRow(head.payrollTotal))}
       ${kv('میانگین حقوق پایه', moneyRow(head.avgSalary))}
       ${kv('رکورد تردد دوره', num(att.records))}
       ${kv('ساعات کارکرد', typeof att.workedHours === 'number' ? fa(att.workedHours) : empty)}
       ${kv('ساعات اضافهکاری', typeof att.overtimeHours === 'number' ? fa(att.overtimeHours) : empty)}
       ${kv('روز مرخصی', num(s.leave && s.leave.days))}
       ${kv('روز مأموریت', num(att.missionDays))}
       ${kv('در انتظار تصمیم (تردد، مرخصی، مأموریت)', num(s.todoCount))}`
    : '';

  /* ---- آخرین مرخصیها: همان فهرست زنده، کلیکپذیر به صف تأیید ---- */
  const lvItems = recentLeaves.ok ? (recentLeaves.data.items || []) : [];
  const lvRows = lvItems.map((it) => {
    const f = it.fields || {};
    return `<tr class="clickable" onclick="go('leaveApprove')">
      <td>${esc(it.employeeName || it.title || '—')}</td>
      <td>${esc(f.leaveType || '—')}</td>
      <td class="num">${esc(f.fromDate || '—')}</td>
      <td class="num">${esc(f.toDate || '—')}</td>
      <td class="num">${esc(f.days || '—')}</td>
      <td>${statusChip(it.status, it.statusFa)}</td></tr>`;
  });

  /* ---- صف تأیید به تفکیک وضعیت: از byStatus سرور ----
     یک برچسب وضعیت در چند ماژول تکرار میشود (مثلاً «تأییدشده» در مرخصی و قرارداد)؛
     برای اینکه فهرست دو سطر همنام نشان ندهد، شمار آنها یکجا جمع میشود. */
  const statusMap = new Map();
  for (const r of (s && Array.isArray(s.byStatus) ? s.byStatus : [])) {
    if (!r.count) continue;
    statusMap.set(r.titleFa, (statusMap.get(r.titleFa) || 0) + r.count);
  }
  const statusRows = [...statusMap.entries()]
    .map(([titleFa, count]) => ({titleFa, count}))
    .sort((a, b) => b.count - a.count);
  const maxS = Math.max(...statusRows.map((r) => r.count), 1);
  const statusList = statusRows.length
    ? statusRows.map((r, i) => `
      <div class="bar-row"><span class="bar-lbl">${esc(r.titleFa)}</span>
        <span class="bar-val num">${fa(r.count)}</span></div>
      <div class="progress" style="margin-bottom:9px"><span
        style="width:${Math.round(r.count / maxS * 100)}%;background:${i === 0 ? 'var(--primary)' : '#93C5FD'}"></span></div>`).join('')
    : `<div class="muted">صف تأییدی ثبت نشده است.</div>`;

  const statsNote = s && s.noteFa
    ? `<div class="card card-pad muted" style="font-size:.72rem;line-height:2"><b>یادداشت سرور:</b> ${esc(s.noteFa)}</div>`
    : '';

  return `
  <div class="db-sec"><h3>اقدامهای امروز</h3><span class="muted">صف کارهایی که منتظر شماست</span></div>
  <div class="act-grid mb">${todo}</div>

  <div class="db-sec"><h3>نمای کلی نیرو</h3><span class="muted">از API سرور محلی</span></div>
  <div class="grid-stats mb">${widgetCards}</div>

  <div class="db-sec"><h3>توزیع و ترکیب</h3>
    <span class="muted">${dept.length ? fa(head.total || 0) + ' پرونده' : ''}</span></div>
  <div class="grid-2eq mb">
    <div class="card card-pad">
      <div class="card-title">توزیع نیرو در واحدها</div>
      ${deptBars}
    </div>
    <div class="card card-pad">
      <div class="card-title">جرم حقوقی و حضور <span class="muted">دورهٔ جاری</span></div>
      ${kpis || '<div class="muted">نمای تجمیعی برنگشت؛ ماژول آمار در این شرکت فعال نیست.</div>'}
    </div>
  </div>

  <div class="db-sec"><h3>آخرین رخدادها</h3><span class="muted">کلیکپذیر — میروید به همان صفحه</span></div>
  <div class="grid-2eq mb">
    <div class="card">
      <div class="card-title" style="padding:16px 18px 0">درخواستهای مرخصی اخیر
        <button class="btn btn-sm btn-ghost" onclick="go('leaveApprove')">مشاهدهٔ همه</button></div>
      ${lvRows.length
        ? tbl(['کارمند', 'نوع', 'از', 'تا', 'روز', 'وضعیت'], lvRows, {foot: false})
        : `<div class="card-pad muted">درخواست مرخصیای ثبت نشده است.</div>`}
    </div>
    <div class="card card-pad">
      <div class="card-title">صف تأیید به تفکیک وضعیت</div>
      ${statusList}
    </div>
  </div>
  ${statsNote}`;
}

/** صفحهٔ زندهٔ داشبورد؛ بدون سرور همان داشبورد نمایشی برمیگردد. */
function pageDashboardLive() {
  if (!isLive()) return pageDashboard();
  liveLoad(() => dashLiveBody());
  return liveShell('داشبورد',
    `خوش آمدید ${session.name} — نمای کلی سامانه (دادهٔ زنده از سرور محلی)`,
    `<button class="btn btn-ghost" onclick="go('employees')">👥 فهرست کارکنان</button>`);
}

LIVE_PAGES.dashboard = pageDashboardLive;
