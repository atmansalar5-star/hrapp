/* =====================================================================
   صفحه‌های زندهٔ «آمار و پایش اطلاعات پایه» و «تقویم کاری»
   =====================================================================

   این فایل هیچ قاعدهٔ کسب‌وکاری ندارد: هر تب یک برش آماری است که سرور آن را از
   رکوردهای ماژول مالک جمع می‌زند، و اینجا فقط شکل می‌گیرد. اگر سرور نباشد، هر صفحه
   به قالب نمایشی بخش ۵ برمی‌گردد.

   نگاشت: هر تابع در انتهای فایل در LIVE_PAGES ثبت می‌شود؛ پس مسیریاب همگام
   (renderPage) آن‌ها را پیش از قالب ثابت می‌بیند.
   ===================================================================== */

/* --------------------------- کمک‌کننده‌های نمودار --------------------------- */

/** نوار سهم افقی (درصد) — تزئینی، بدون منطق. */
function statBar(pct, color = '#2563EB') {
  const w = Math.max(0, Math.min(100, Number(pct) || 0));
  return `<div class="progress" style="flex:1;min-width:60px"><span style="width:${w}%;background:${color}"></span></div>`;
}

/** نمودار میله‌ای افقی: [{label, value, pct, color}] — برای برش‌های رتبه‌بندی‌شده. */
function hBarChart(rows, emptyMsg = 'داده‌ای برای نمایش نیست') {
  if (!rows.length) return `<div class="empty-state" style="padding:18px"><div class="big">📭</div>${esc(emptyMsg)}</div>`;
  return `<div style="display:flex;flex-direction:column;gap:10px">
    ${rows.map(r => `<div style="display:flex;align-items:center;gap:10px;font-size:.8rem">
      <span style="min-width:150px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${esc(r.label)}</span>
      ${statBar(r.pct, r.color)}
      <span class="num" style="min-width:90px;text-align:end">${r.text != null ? r.text : fa(r.value)}</span>
    </div>`).join('')}
  </div>`;
}

/** سبد سه‌گانهٔ وضعیت‌ها (در انتظار / تأییدشده / ردشده) به‌همراه شمارش. */
function statusBucketsHTML(o) {
  const total = (o.pending || 0) + (o.approved || 0) + (o.rejected || 0);
  const pct = v => total ? Math.round(v / total * 1000) / 10 : 0;
  return `<div class="card card-pad" style="margin-bottom:14px">
    <div class="card-title">صف تأیید</div>
    <div class="grid-stats" style="margin-bottom:0">
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(o.pending || 0), label: 'در انتظار تصمیم'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(o.approved || 0), label: 'تأییدشده'})}
      ${statCard({ico: '⛔', tint: '#FEE2E2', accent: '#DC2626', val: fa(o.rejected || 0), label: 'ردشده'})}
    </div>
    <div style="display:flex;gap:4px;margin-top:12px;height:10px">
      <div style="width:${pct(o.approved || 0)}%;background:#16A34A;border-radius:99px"></div>
      <div style="width:${pct(o.pending || 0)}%;background:#D97706;border-radius:99px"></div>
      <div style="width:${pct(o.rejected || 0)}%;background:#DC2626;border-radius:99px"></div>
    </div>
    <div class="muted" style="font-size:.74rem;margin-top:6px">مجموع ${fa(total)} رکورد در گردش است.</div>
  </div>`;
}

/** جعبهٔ یادداشت سرور (اگر داشته باشد). */
function statNote(noteFa) {
  return noteFa ? `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">ℹ️ ${esc(noteFa)}</div>` : '';
}

/* =====================================================================
   تب «داشبورد آماری»
   ===================================================================== */

function pageStatsOverviewLive() {
  if (!isLive()) return offlineFallback('statsOverview');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/statistics/dashboard');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    const head = d.headcount || {};
    const att = d.attendance || {};
    const leave = d.leave || d.leaveSummary || {};

    const stats = `<div class="grid-stats">
      ${statCard({ico: '👥', tint: '#EFF6FF', accent: '#2563EB', val: fa(head.total || 0), label: 'کل کارکنان'})}
      ${statCard({ico: '🟢', tint: '#DCFCE7', accent: '#16A34A', val: fa((head.byStatus || {}).active || 0), label: 'کارکنان شاغل'})}
      ${statCard({ico: '💰', tint: '#EDE9FE', accent: '#7C3AED', val: money(head.payrollTotal || 0), label: 'جرم حقوقی ماهانه (ریال)'})}
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(d.todoCount || 0), label: 'رکورد در انتظار تأیید'})}
    </div>`;

    const attBox = `<div class="grid-stats">
      ${statCard({ico: '📋', tint: '#F8FAFC', accent: '#64748B', val: fa(att.records || 0), label: 'رکورد حضور دوره'})}
      ${statCard({ico: '🕐', tint: '#F8FAFC', accent: '#64748B', val: fa(att.workedHours || 0), label: 'مجموع کارکرد (ساعت)'})}
      ${statCard({ico: '⏱️', tint: '#F8FAFC', accent: '#64748B', val: fa(att.overtimeHours || 0), label: 'اضافه‌کاری (ساعت)'})}
      ${statCard({ico: '🗓️', tint: '#F8FAFC', accent: '#64748B', val: fa(att.leaveDays || 0), label: 'مرخصی (روز)'})}
    </div>`;

    const deptRows = (d.byDept || []).map(r => ({
      label: r.titleFa, value: r.count, pct: r.sharePct, text: `${fa(r.count)} نفر`,
    }));
    const chart = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">جمعیت کارکنان به تفکیک واحد سازمانی</div>
      ${hBarChart(deptRows)}
    </div>`;

    const statusRows = (d.byStatus || []).map(r => `<tr>
      <td>${esc(r.titleFa)}</td><td>${faNum(r.count)}</td>
      <td>${statBar(r.count ? 100 : 0, '#94A3B8')}</td></tr>`);
    const statusTbl = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">وضعیت رکوردهای حضور دوره</div>
      ${liveTable(['وضعیت', 'تعداد', 'سهم'], statusRows, statusRows.length)}
    </div>`;

    const leaveRows = (leave.rows || []).map(r => `<tr>
      <td>${esc(r.titleFa)}</td>
      <td>${faNum(r.count)}</td>
      <td>${faNum(r.days)}</td>
      <td>${statBar(r.sharePct, '#2563EB')}</td>
      <td class="num">${fa(r.sharePct)}٪</td></tr>`);
    const leaveTbl = (leave.rows || []).length ? `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">مرخصی‌ها به تفکیک نوع</div>
      ${liveTable(['نوع مرخصی', 'تعداد', 'روز', 'سهم', 'درصد'], leaveRows, leaveRows.length)}
    </div>` : '';

    return stats + attBox + chart + statusBucketsHTML(att) + statusTbl + leaveTbl + statNote(d.noteFa);
  });
  return liveShell('داشبورد آماری', 'نمای تجمیعی آمار منابع انسانی — همهٔ اعداد از سرور و روی دادهٔ واقعی');
}

/* =====================================================================
   تب «حضور و غیاب و تردد»
   ===================================================================== */

function pageStatsAttendanceLive() {
  if (!isLive()) return offlineFallback('statsAttendance');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/statistics/attendance');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};

    const stats = `<div class="grid-stats">
      ${statCard({ico: '📋', tint: '#EFF6FF', accent: '#2563EB', val: fa(d.records || 0), label: 'کل رکوردهای دوره'})}
      ${statCard({ico: '🕐', tint: '#DCFCE7', accent: '#16A34A', val: fa(d.workedHours || 0), label: 'مجموع کارکرد (ساعت)'})}
      ${statCard({ico: '⏱️', tint: '#EDE9FE', accent: '#7C3AED', val: fa(d.overtimeHours || 0), label: 'اضافه‌کاری (ساعت)'})}
      ${statCard({ico: '🗓️', tint: '#FEF3C7', accent: '#D97706', val: fa(d.leaveDays || 0), label: 'مرخصی (روز)'})}
    </div>`;

    const kindRows = (d.rows || []).map(r => `<tr>
      <td>${esc(r.titleFa)}</td>
      <td>${faNum(r.count)}</td>
      <td><span class="ltr" style="font-size:.72rem">${esc(r.kind)}</span></td></tr>`);

    const deptRows = (d.deptRows || []).map(r => `<tr>
      <td>${esc(r.titleFa)}</td>
      <td>${faNum(r.count)}</td>
      <td>${faNum(r.hours)}</td>
      <td>${statBar(r.sharePct, '#2563EB')}</td>
      <td class="num">${fa(r.sharePct)}٪</td></tr>`);

    const leaveRows = Object.keys(d.leaveTypes || {}).map(k => `<tr>
      <td>${esc(k)}</td><td>${faNum((d.leaveTypes || {})[k])}</td></tr>`);

    return stats +
      statNote('این تب کارکرد تردد را پیش از اجرای حقوق جمع می‌زند. جمع ساعت کارکرد از رکوردهای «تردد روزانه» و روزهای مرخصی از رکوردهای مرخصیِ همان دوره می‌آید.') +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">کارکرد به تفکیک واحد سازمانی</div>
        ${liveTable(['واحد سازمانی', 'رکورد', 'ساعت کارکرد', 'سهم', 'درصد'], deptRows, deptRows.length)}
      </div>` +
      (leaveRows.length ? `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">مرخصی دوره به تفکیک نوع</div>
        ${liveTable(['نوع مرخصی', 'تعداد'], leaveRows, leaveRows.length)}
      </div>` : '') +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">ترکیب رکوردهای حضور</div>
        ${liveTable(['نوع رکورد', 'تعداد', 'کلید ماشین‌خوان'], kindRows, kindRows.length)}
      </div>` +
      statusBucketsHTML(d);
  });
  return liveShell('حضور و غیاب و تردد', 'بررسی و تأیید کارکرد تردد پیش از اجرای حقوق و اعلام مغایرت به امور کارکنان');
}

/* =====================================================================
   تب «مرخصی‌ها»
   ===================================================================== */

function pageStatsLeaveLive() {
  if (!isLive()) return offlineFallback('statsLeave');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/statistics/leave');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};

    const stats = `<div class="grid-stats">
      ${statCard({ico: '🗓️', tint: '#EFF6FF', accent: '#2563EB', val: fa(d.total || 0), label: 'مرخصی ثبت‌شدهٔ دوره'})}
      ${statCard({ico: '📆', tint: '#EDE9FE', accent: '#7C3AED', val: fa(d.sumDays || 0), label: 'مجموع روز مرخصی'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(d.approved || 0), label: 'تأییدشده'})}
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(d.pending || 0), label: 'در انتظار تأیید'})}
    </div>`;

    const rows = (d.rows || []).map(r => `<tr>
      <td>${esc(r.titleFa)}</td>
      <td>${faNum(r.count)}</td>
      <td>${faNum(r.days)}</td>
      <td>${statBar(r.sharePct, '#2563EB')}</td>
      <td class="num">${fa(r.sharePct)}٪</td>
      <td>${pChip('c-ok', fa(r.approved))}</td>
      <td>${pChip('c-wait', fa(r.pending))}</td>
      <td>${pChip('c-rej', fa(r.rejected))}</td></tr>`);

    const chart = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">سهم هر نوع مرخصی از مجموع روزها</div>
      ${hBarChart((d.rows || []).map(r => ({
        label: r.titleFa, value: r.days, pct: r.sharePct, text: `${fa(r.days)} روز`,
      })))}
    </div>`;

    return stats + statNote(d.noteFa) + chart +
      statusBucketsHTML(d) +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">برش مرخصی‌ها به تفکیک نوع</div>
        ${liveTable(['نوع مرخصی', 'تعداد', 'روز', 'سهم', 'درصد', 'تأییدشده', 'در انتظار', 'ردشده'], rows, rows.length)}
      </div>`;
  });
  return liveShell('مرخصی‌ها', 'برش و پایش مرخصی‌های دوره به تفکیک نوع و وضعیت تأیید');
}

/* =====================================================================
   تب «ماموریت‌ها»
   ===================================================================== */

function pageStatsMissionLive() {
  if (!isLive()) return offlineFallback('statsMission');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/statistics/missions');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};

    const stats = `<div class="grid-stats">
      ${statCard({ico: '📍', tint: '#EFF6FF', accent: '#2563EB', val: fa(d.total || 0), label: 'مأموریت دوره'})}
      ${statCard({ico: '📆', tint: '#EDE9FE', accent: '#7C3AED', val: fa(d.sumDays || 0), label: 'مجموع روز مأموریت'})}
      ${statCard({ico: '💰', tint: '#DCFCE7', accent: '#16A34A', val: money(d.sumAmount || 0), label: 'فوق‌العادهٔ برآوردی (ریال)'})}
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(d.pending || 0), label: 'در انتظار تأیید'})}
    </div>`;

    const rows = (d.rows || []).map(r => `<tr>
      <td>${esc(r.titleFa)}</td>
      <td>${faNum(r.count)}</td>
      <td>${faNum(r.days)}</td>
      <td>${faNum(r.amount)}</td>
      <td>${statBar(r.sharePct, '#2563EB')}</td>
      <td class="num">${fa(r.sharePct)}٪</td></tr>`);

    const chart = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">روزهای مأموریت به تفکیک واحد سازمانی</div>
      ${hBarChart((d.rows || []).map(r => ({
        label: r.titleFa, value: r.days, pct: r.sharePct, text: `${fa(r.days)} روز`,
      })))}
    </div>`;

    return stats + statNote(d.noteFa) + chart +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">برش مأموریت‌ها به تفکیک واحد</div>
        ${liveTable(['واحد سازمانی', 'تعداد', 'روز', 'فوق‌العادهٔ برآوردی (ریال)', 'سهم', 'درصد'], rows, rows.length)}
      </div>` +
      statusBucketsHTML(d);
  });
  return liveShell('ماموریت‌ها', 'بررسی و تأیید مأموریت‌ها و فوق‌العادهٔ روزانه پیش از ارسال به حقوق');
}

/* =====================================================================
   تب «اطلاعات پایهٔ حقوق و مزایا»
   ===================================================================== */

function pageStatsBasePayLive() {
  if (!isLive()) return offlineFallback('statsBasePay');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/statistics/pay-elements');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    const s = d.summary || {};

    const stats = `<div class="grid-stats">
      ${statCard({ico: '🧮', tint: '#EFF6FF', accent: '#2563EB', val: fa(s.count || 0), label: 'قلم حقوق فعال'})}
      ${statCard({ico: '➕', tint: '#DCFCE7', accent: '#16A34A', val: money(s.earningTotal || 0), label: 'جمع مزایا (ریال)'})}
      ${statCard({ico: '➖', tint: '#FEE2E2', accent: '#DC2626', val: money(s.deductionTotal || 0), label: 'جمع کسور (ریال)'})}
      ${statCard({ico: '💵', tint: '#EDE9FE', accent: '#7C3AED', val: money(s.net || 0), label: 'خالص محاسبه‌شده (ریال)'})}
    </div>`;

    const wageRows = [
      ['مزد شغل', s.wageJob], ['مزد رتبه', s.wageRank], ['مزد سنوات', s.wageSeniority],
      ['فوق‌العادهٔ پست', s.postAllowance], ['فوق‌العادهٔ ایثارگری', s.sacrifice],
    ].map(([label, val]) => `<tr><td>${esc(label)}</td><td>${faNum(Math.round(val || 0))}</td></tr>`);
    const mazd = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">شاکلهٔ مزد مبنا — جمع: <span class="num">${fa(Math.round(s.mazdBase || 0))}</span> ریال</div>
      ${liveTable(['جزء مزد', 'مبلغ (ریال)'], wageRows, wageRows.length)}
    </div>`;

    const rows = (d.elements || []).map(e => `<tr>
      <td>${esc(e.titleFa)}</td>
      <td class="ltr" style="font-size:.72rem">${esc(e.code)}</td>
      <td>${pChip(e.category === 'earning' ? 'c-ok' : 'c-rej', e.category === 'earning' ? 'مزایا' : 'کسور')}</td>
      <td>${esc(e.calc || '—')}</td>
      <td>${esc(e.basisFa || e.basis || '—')}</td>
      <td class="num">${e.resolved ? fa(Math.round(e.amount || 0)) : '<span class="muted">حل‌نشده</span>'}</td>
      <td style="font-size:.74rem">${esc(e.detailFa || '—')}</td></tr>`);

    const calcRows = Object.keys(d.byCalc || {}).map(k => `<tr>
      <td>${esc(k)}</td><td>${faNum((d.byCalc || {})[k])}</td></tr>`);

    return stats + statNote(d.noteFa) + mazd +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">اقلام فعال حقوق و مزایا</div>
        ${liveTable(['قلم', 'کد', 'نوع', 'روش محاسبه', 'مبنا', 'مبلغ محاسبه‌شده (ریال)', 'توضیح'], rows, rows.length)}
      </div>` +
      (calcRows.length ? `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">ترکیب اقلام بر پایهٔ روش محاسبه</div>
        ${liveTable(['روش محاسبه', 'تعداد قلم'], calcRows, calcRows.length)}
      </div>` : '');
  });
  return liveShell('اطلاعات پایهٔ حقوق و مزایا', 'کنترل و تأیید اطلاعات پایهٔ مزایا و کسورات — محاسبه با موتور مشترک قواعد');
}

/* =====================================================================
   تب «کارانه»
   ===================================================================== */

function pageStatsKaranaLive() {
  if (!isLive()) return offlineFallback('statsKarana');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/statistics/karana');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};

    const stats = `<div class="grid-stats">
      ${statCard({ico: '💳', tint: '#EFF6FF', accent: '#2563EB', val: fa(d.total || 0), label: 'رکورد کارانه'})}
      ${statCard({ico: '💰', tint: '#DCFCE7', accent: '#16A34A', val: money(d.sumAmount || 0), label: 'مجموع کارانه (ریال)'})}
      ${statCard({ico: '✅', tint: '#EDE9FE', accent: '#7C3AED', val: fa(d.approved || 0), label: 'تأییدشده'})}
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(d.pending || 0), label: 'در انتظار تأیید'})}
    </div>`;

    const rows = (d.rows || []).map(r => `<tr>
      <td>${esc(r.titleFa)}</td>
      <td>${faNum(r.count)}</td>
      <td>${faNum(r.amount)}</td>
      <td>${statBar(r.sharePct, '#7C3AED')}</td>
      <td class="num">${fa(r.sharePct)}٪</td></tr>`);

    const chart = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">سهم هر دورهٔ پرداخت از مجموع کارانه</div>
      ${hBarChart((d.rows || []).map(r => ({
        label: r.titleFa, value: r.amount, pct: r.sharePct, text: money(r.amount), color: '#7C3AED',
      })), 'کارانه‌ای ثبت نشده است')}
    </div>`;

    return stats + statNote(d.noteFa) + chart +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">برش کارانه به تفکیک دوره</div>
        ${liveTable(['دوره', 'تعداد', 'مبلغ (ریال)', 'سهم', 'درصد'], rows, rows.length)}
      </div>` +
      statusBucketsHTML(d);
  });
  return liveShell('کارانه و پرداخت‌های متغیر', 'پایش کارانه و پرداخت‌های متغیر پیش از تأیید و ارسال به مالی');
}

/* =====================================================================
   «تقویم کاری» — تقویم واقعی استقرار، نه قالب ثابت
   ===================================================================== */

function pageCalendarLive() {
  if (!isLive()) return offlineFallback('calendar');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/settings/work-calendar');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    if (d.error) return liveErrorBox(d.error);

    const today = d.today || {};
    const month = d.month || {};
    const days = month.days || [];

    const stats = `<div class="grid-stats">
      ${statCard({ico: '📅', tint: '#EFF6FF', accent: '#2563EB', val: `<span class="num">${fa(today.date || '')}</span>`, label: `${esc(today.weekdayFa || '')} — ${esc(today.monthLabel || '')}`})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(month.workingDays || 0), label: 'روز کاری این ماه'})}
      ${statCard({ico: '🛑', tint: '#FEE2E2', accent: '#DC2626', val: fa(month.weekendDays || 0), label: 'تعطیل هفتگی'})}
      ${statCard({ico: '🎉', tint: '#FEF3C7', accent: '#D97706', val: fa(month.holidays || 0), label: 'تعطیل رسمی ثبت‌شده'})}
    </div>`;

    // شبکهٔ تقویم: ۷ ستون (شنبه … جمعه) با جاهای خالی ابتدای ماه.
    const weekdays = ['شنبه', 'یکشنبه', 'دوشنبه', 'سه‌شنبه', 'چهارشنبه', 'پنجشنبه', 'جمعه'];
    const lead = days.length ? days[0].weekdayIndex : 0;
    const cells = [];
    for (let i = 0; i < lead; i++) cells.push('<div></div>');
    for (const day of days) {
      const isToday = day.date === today.date;
      const cls = day.weekend ? 'c-term' : day.holiday ? 'c-rej' : 'c-ok';
      cells.push(`<div style="border:1px solid var(--border);border-radius:10px;padding:8px 10px;min-height:62px;
          background:${isToday ? 'var(--tint,#EFF6FF)' : 'var(--surface)'};${isToday ? 'border-color:var(--primary)' : ''}">
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span class="num" style="font-weight:700">${fa(day.day)}</span>
          ${day.weekend || day.holiday ? `<span class="chip ${cls}" style="font-size:.62rem">${day.holiday ? 'تعطیل رسمی' : 'تعطیل'}</span>` : ''}
        </div>
        <div class="muted" style="font-size:.66rem">${esc(day.weekdayFa)}</div>
        ${day.titleFa ? `<div style="font-size:.64rem;line-height:1.6">${esc(day.titleFa)}</div>` : ''}
      </div>`);
    }
    const grid = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">تقویم ${esc(month.label || '')} — ${esc(d.company || '')}</div>
      <div style="display:grid;grid-template-columns:repeat(7,1fr);gap:6px">
        ${weekdays.map(w => `<div class="muted" style="text-align:center;font-size:.7rem;font-weight:700">${w}</div>`).join('')}
        ${cells.join('')}
      </div>
    </div>`;

    const paramRows = (d.params || []).map(p => `<tr>
      <td>${esc(p.titleFa)}</td>
      <td class="ltr" style="font-size:.72rem">${esc(p.key)}</td>
      <td class="num">${fa(p.value)}</td>
      <td>${esc(p.unitFa)}</td></tr>`);

    return stats + statNote(d.noteFa) + grid +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">پارامترهای حضور، مرخصی و پرداخت شرکت فعال</div>
        ${liveTable(['پارامتر', 'کلید', 'مقدار', 'واحد'], paramRows, paramRows.length)}
      </div>`;
  });
  return liveShell('تقویم کاری', 'تعطیل هفتگی، تعطیلات رسمی و روزهای کاری استقرار شرکت فعال');
}

/* ثبت در مسیریاب زنده — پیش از قالب نمایشی بخش ۵ بررسی می‌شود. */
LIVE_PAGES.statsOverview = pageStatsOverviewLive;
LIVE_PAGES.statsAttendance = pageStatsAttendanceLive;
LIVE_PAGES.statsLeave = pageStatsLeaveLive;
LIVE_PAGES.statsMission = pageStatsMissionLive;
LIVE_PAGES.statsBasePay = pageStatsBasePayLive;
LIVE_PAGES.statsKarana = pageStatsKaranaLive;
LIVE_PAGES.calendar = pageCalendarLive;
