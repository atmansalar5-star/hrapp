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

    // شاکلهٔ مزد مبنا از خود سرور می‌آید (`breakdown`): هر جزء با عنوان و گروه، پس کارت
    // هیچ‌وقت فهرست ثابت خودش را نمی‌سازد و با فیش و حکم یک عدد می‌دهد.
    const parts = (s.breakdown || []).map(p => [p.titleFa, p.kindFa, p.amount]);
    const wageRows = parts.length
      ? parts.map(([label, kindFa, val]) => `<tr><td>${esc(label)} <span class="muted" style="font-size:.7rem">${esc(kindFa || '')}</span></td><td>${faNum(Math.round(val || 0))}</td></tr>`)
      : [['مزد شغل', '', s.wageJob], ['مزد رتبه', '', s.wageRank], ['مزد سنوات', '', s.wageSeniority]]
          .map(([label, kindFa, val]) => `<tr><td>${esc(label)}</td><td>${faNum(Math.round(val || 0))}</td></tr>`);
    wageRows.push(`<tr><td><b>مزد مبنا</b></td><td><b>${faNum(Math.round(s.mazdBase || 0))}</b></td></tr>`);
    const mazd = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">شاکلهٔ مزد مبنا — جمع: <span class="num">${fa(Math.round(s.mazdBase || 0))}</span> ریال</div>
      ${liveTable(['جزء مزد مبنا', 'مبلغ (ریال)'], wageRows, wageRows.length)}
      <p class="muted" style="font-size:.76rem;line-height:2;margin:8px 0 0">
        مزد مبنا جمع همهٔ اجزای ثابت است (مزد شغل و ملحقات ${fa(Math.round(s.jobAttachments || 0))} ریال + ثابت‌های شغلی ${fa(Math.round((s.special || 0) + (s.workshop || 0) + (s.attractionPublic || 0) + (s.personalBase || 0) + (s.jobSpecialty || 0)))} ریال + مبالغ مصوب و اقلیمی).
        فوق‌العاده‌های اقلیمی، نوبت‌کاری، شب‌کاری و اضافه‌کاری درصدی از همین مزد مبنا هستند و جداگانه پرداخت می‌شوند.
      </p>
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

/* --------------------------- تقویم کارکرد ---------------------------
   نمای ماهانهٔ کارکرد: هر روز با وضعیتش (تعطیل هفتگی/رسمی، مناسبت، مرخصی، مأموریت،
   اضافه‌کاری، کارکرد و روزهای کاری بدون کارکرد). وضعیت‌ها از دو مسیر سرور می‌آید:
   تقویم کاری استقرار (تعطیل‌ها و پارامترها) و «تقویم کارکرد» ماژول حضور (رکوردهای
   تردد/مرخصی/مأموریت/اضافه‌کاری). این فایل هیچ محاسبه‌ای نمی‌کند؛ فقط می‌چیند و رنگ می‌دهد.
   ------------------------------------------------------------------- */

var CAL_MONTH = '';        // ماه انتخابی «YYYY/MM» — خالی یعنی ماه جاری سرور
var CAL_SERVER_MONTH = ''; // ماه جاری بر پایهٔ ساعت سرور (مبنای دکمه‌های قبل/بعد)
var CAL_EMPLOYEE = '';     // کد کارمند برای تقویم یک نفر (خالی = همه)
var CAL_EMPLOYEES = null;  // فهرست کارکنان فیلتر (یک‌بار خوانده می‌شود)

/** نشان و رنگ هر وضعیت روز — تزئینی، بدون منطق. */
var CAL_STYLE = {
  holiday: { icon: '⛔', bg: '#FEE2E2', fg: '#B91C1C' },
  weekend: { icon: '🛑', bg: '#F1F5F9', fg: '#475569' },
  leave: { icon: '🟨', bg: '#FEF3C7', fg: '#92400E' },
  mission: { icon: '📍', bg: '#EDE9FE', fg: '#5B21B6' },
  mixed: { icon: '🗂️', bg: '#EDE9FE', fg: '#5B21B6' },
  worked: { icon: '✅', bg: '#DCFCE7', fg: '#166534' },
  missing: { icon: '⚠️', bg: '#FEF2F2', fg: '#B91C1C' },
  pending: { icon: '•', bg: '#F8FAFC', fg: '#64748B' },
};

/** سبک وضعیت روز (پیش‌فرض: بی‌وضعیت). */
function calStyle(key) { return CAL_STYLE[key] || CAL_STYLE.pending; }

/** شکستن «YYYY/MM» به سال و ماه عددی. */
function calParts(key) {
  const m = /^(\d{4})\/(\d{1,2})$/.exec((key || '').trim());
  return m ? { y: Number(m[1]), m: Number(m[2]) } : null;
}

/** جابه‌جایی ماه تقویم (delta = ±۱) و بازخوانی صفحه. */
function calShift(delta) {
  const base = calParts(CAL_MONTH) || calParts(CAL_SERVER_MONTH);
  if (!base) return;
  let y = base.y, m = base.m + delta;
  if (m < 1) { m = 12; y--; }
  if (m > 12) { m = 1; y++; }
  CAL_MONTH = y + '/' + String(m).padStart(2, '0');
  liveReload();
}

/** نوار ماه و فیلتر کارمند. */
function calendarNav(monthLabel) {
  const list = CAL_EMPLOYEES || [];
  const opts = [`<option value="">همهٔ کارکنان</option>`].concat(list.map(e =>
    `<option value="${esc(e.id)}" ${e.id === CAL_EMPLOYEE ? 'selected' : ''}>${esc(e.fullName)} — ${esc(e.id)}</option>`)).join('');
  return `<div class="row-actions" style="flex-wrap:wrap;gap:8px;align-items:center">
    <button class="btn btn-ghost btn-sm" onclick="calShift(-1)">‹ ماه قبل</button>
    <b style="font-size:.92rem">${esc(monthLabel || '')}</b>
    <button class="btn btn-ghost btn-sm" onclick="calShift(1)">ماه بعد ›</button>
    <button class="btn btn-ghost btn-sm" onclick="CAL_MONTH='';liveReload()">ماه جاری</button>
    <label style="font-size:.78rem">تقویم کارمند:
      <select id="calEmployee" style="font-size:.78rem" onchange="CAL_EMPLOYEE=this.value;liveReload()">${opts}</select>
    </label>
  </div>`;
}

/** خلاصهٔ برچسب‌های یک روز: هر نوع یک سطر کوتاه، با نام کارمندان (تا دو نفر). */
function calendarDayLines(day) {
  const groups = [
    ['occasion', '🎉', 'مناسبت'],
    ['leave', '🟨', 'مرخصی'],
    ['mission', '📍', 'مأموریت'],
    ['overtime', '⏱️', 'اضافه‌کاری'],
    ['attendance', '✅', 'کارکرد'],
  ];
  const lines = [];
  for (const [kind, icon, label] of groups) {
    const chips = (day.chips || []).filter(c => c.kind === kind);
    if (!chips.length) continue;
    if (kind === 'occasion') {
      lines.push(`<div style="font-size:.62rem;color:#92400E" title="${esc(chips.map(c => c.detailFa || '').join(' — '))}">${chips.map(c => icon + ' ' + esc(c.titleFa)).join('<br>')}</div>`);
      continue;
    }
    const names = chips.map(c => c.employeeFa || '').filter(Boolean).slice(0, 2).join('، ');
    const more = chips.length > 2 ? ` و ${fa(chips.length - 2)} نفر دیگر` : '';
    const pending = chips.filter(c => c.pending).length;
    let extra = '';
    if (kind === 'overtime') {
      const hours = chips.reduce((s, c) => s + (c.hours || 0), 0);
      extra = ` — ${fa(hours)} ساعت`;
    } else if (kind === 'attendance') {
      const hours = chips.reduce((s, c) => s + (c.hours || 0), 0);
      extra = ` — ${fa(hours)} ساعت`;
    } else if (kind === 'leave' || kind === 'mission') {
      extra = '';
    }
    lines.push(`<div style="font-size:.62rem;line-height:1.7" title="${esc(chips.map(c => c.titleFa + (c.detailFa ? ' — ' + c.detailFa : '')).join(' | '))}">
      ${icon} ${label}${chips.length > 1 ? ' (' + fa(chips.length) + ')' : ''}${extra}${pending ? ` <span style="color:#B45309">⏳${fa(pending)}</span>` : ''}
      ${names ? '<br><span class="muted">' + esc(names + more) + '</span>' : ''}
    </div>`);
  }
  return lines.join('');
}

/** شبکهٔ ماهانهٔ کارکرد. */
function calendarGrid(days) {
  const weekdays = ['شنبه', 'یکشنبه', 'دوشنبه', 'سه‌شنبه', 'چهارشنبه', 'پنجشنبه', 'جمعه'];
  const cells = [];
  const lead = days.length ? days[0].weekdayIndex : 0;
  for (let i = 0; i < lead; i++) cells.push('<div></div>');
  for (const day of days) {
    const style = calStyle(day.statusKey);
    const border = day.today ? 'var(--primary)' : 'var(--border)';
    cells.push(`<div style="border:1px solid ${border};border-radius:10px;padding:6px;min-height:104px;background:${style.bg};${day.today ? 'box-shadow:0 0 0 2px rgba(37,99,235,.25)' : ''}">
      <div style="display:flex;justify-content:space-between;align-items:center;gap:4px">
        <span class="num" style="font-weight:700;color:${style.fg}">${fa(day.day)}</span>
        <span style="font-size:.6rem;color:${style.fg}">${style.icon} ${esc(day.statusFa)}</span>
      </div>
      <div class="muted" style="font-size:.62rem">${esc(day.weekdayFa)}${day.holidayTitleFa ? ' — ' + esc(day.holidayTitleFa) : ''}</div>
      ${calendarDayLines(day)}
    </div>`);
  }
  return `<div class="card card-pad" style="margin-bottom:14px">
    <div style="display:grid;grid-template-columns:repeat(7,1fr);gap:6px">
      ${weekdays.map(w => `<div class="muted" style="text-align:center;font-size:.7rem;font-weight:700">${w}</div>`).join('')}
      ${cells.join('')}
    </div>
  </div>`;
}

/** راهنمای وضعیت‌های تقویم (از سرور می‌آید تا با موتور یکی بماند). */
function calendarLegendCard(legend) {
  const items = (legend || []).map(item => {
    const style = calStyle(item.kind);
    return `<span style="display:inline-flex;align-items:center;gap:6px;font-size:.74rem;margin:0 12px 6px 0" title="${esc(item.noteFa || '')}">
      <span style="width:10px;height:10px;border-radius:3px;background:${style.bg};border:1px solid ${style.fg}"></span>
      ${style.icon} ${esc(item.titleFa)}</span>`;
  }).join('');
  return `<div class="card card-pad" style="margin-bottom:14px"><div style="display:flex;flex-wrap:wrap">${items}</div></div>`;
}

/** کارت‌های جمع ماه از تقویم کارکرد. */
function calendarTotalsCards(totals) {
  const t = totals || {};
  return `<div class="grid-stats">
    ${statCard({ ico: '🗓️', tint: '#EFF6FF', accent: '#2563EB', val: fa(t.workingDays || 0), label: 'روز کاری ماه' })}
    ${statCard({ ico: '🛑', tint: '#F1F5F9', accent: '#64748B', val: fa(t.weekendDays || 0), label: 'تعطیل هفتگی' })}
    ${statCard({ ico: '⛔', tint: '#FEE2E2', accent: '#DC2626', val: fa(t.holidays || 0), label: 'تعطیل رسمی' })}
    ${statCard({ ico: '🟨', tint: '#FEF3C7', accent: '#D97706', val: fa(t.leaveDays || 0), label: 'روز مرخصی (تأییدشده)' })}
    ${statCard({ ico: '📍', tint: '#EDE9FE', accent: '#7C3AED', val: fa(t.missionDays || 0), label: 'روز مأموریت' })}
    ${statCard({ ico: '⏱️', tint: '#E0F2FE', accent: '#0284C7', val: fa(t.overtimeHours || 0), label: 'ساعت اضافه‌کاری' })}
    ${statCard({ ico: '🕐', tint: '#DCFCE7', accent: '#16A34A', val: fa(t.workedHours || 0), label: 'ساعت کارکرد ثبت‌شده' })}
    ${statCard({ ico: '⚠️', tint: '#FEF2F2', accent: '#B91C1C', val: fa(t.missingWorkDays || 0), label: 'روز کاری بدون کارکرد' })}
  </div>`;
}

/** صفحهٔ «تقویم کارکرد». */
function pageCalendarLive() {
  if (!isLive()) return offlineFallback('calendar');
  liveLoad(async () => {
    const monthQuery = CAL_MONTH ? '?month=' + encodeURIComponent(CAL_MONTH) : '';
    const workQuery = (CAL_MONTH ? '?month=' + encodeURIComponent(CAL_MONTH) : '?') +
      (CAL_EMPLOYEE ? (CAL_MONTH ? '&' : '') + 'employee=' + encodeURIComponent(CAL_EMPLOYEE) : '');
    const employeeCall = CAL_EMPLOYEES
      ? Promise.resolve({ ok: true, data: { items: CAL_EMPLOYEES } })
      : apiFetch('/api/v1/employee/employees?size=200');

    const [calRes, workRes, empRes] = await Promise.all([
      apiFetch('/api/v1/settings/work-calendar' + monthQuery),
      apiFetch('/api/v1/attendance/work-calendar' + workQuery),
      employeeCall,
    ]);
    if (!workRes.ok) return liveErrorBox(workRes.message);
    const work = workRes.data || {};
    const month = work.month || {};
    const days = month.days || [];
    CAL_SERVER_MONTH = month.key || '';
    if (empRes.ok && empRes.data && empRes.data.items) {
      CAL_EMPLOYEES = (empRes.data.items || []).map(e => ({ id: e.id, fullName: e.fullName }));
    }

    const settings = (calRes.ok && calRes.data) || {};
    const params = (settings.params || []).map(p => `<tr>
      <td>${esc(p.titleFa)}</td>
      <td class="ltr" style="font-size:.72rem">${esc(p.key)}</td>
      <td class="num">${fa(p.value)}</td>
      <td>${esc(p.unitFa)}</td></tr>`);

    const today = (settings.today) || work.today || {};
    const todayBox = `<div class="card card-pad" style="margin-bottom:14px;font-size:.8rem;line-height:2">
      امروز: <b class="num">${fa(today.date || '')}</b> ${esc(today.weekdayFa || '')} — ${esc(today.monthLabel || '')}
      ${CAL_EMPLOYEE ? ` · تقویم کارمند: <b>${esc(CAL_EMPLOYEE)}</b>` : ''}
      · ${esc(work.scopeFa || '')}
    </div>`;

    return calendarTotalsCards(month.totals) +
      `<div class="card card-pad" style="margin-bottom:14px">${calendarNav(month.label)}
        ${work.noteFa ? `<div class="muted" style="font-size:.76rem;margin-top:8px;line-height:2">${esc(work.noteFa)}</div>` : ''}</div>` +
      calendarLegendCard(work.legend) +
      todayBox +
      calendarGrid(days) +
      (params.length ? `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">پارامترهای حضور، مرخصی و پرداخت شرکت فعال</div>
        ${liveTable(['پارامتر', 'کلید', 'مقدار', 'واحد'], params, params.length)}
      </div>` : '');
  });
  return liveShell('تقویم کارکرد',
    'نمای ماهانهٔ کارکرد با وضعیت هر روز: تعطیل هفتگی و رسمی، مناسبت، مرخصی، مأموریت، اضافه‌کاری و روزهای بدون کارکرد',
    `<button class="btn btn-ghost" onclick="CAL_MONTH='';CAL_EMPLOYEE='';liveReload()">↻ ماه جاری</button>`);
}
/* ثبت در مسیریاب زنده — پیش از قالب نمایشی بخش ۵ بررسی می‌شود. */
LIVE_PAGES.statsOverview = pageStatsOverviewLive;
LIVE_PAGES.statsAttendance = pageStatsAttendanceLive;
LIVE_PAGES.statsLeave = pageStatsLeaveLive;
LIVE_PAGES.statsMission = pageStatsMissionLive;
LIVE_PAGES.statsBasePay = pageStatsBasePayLive;
LIVE_PAGES.statsKarana = pageStatsKaranaLive;
LIVE_PAGES.calendar = pageCalendarLive;
