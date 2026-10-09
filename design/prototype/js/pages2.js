/* =====================================================================
   صفحات — بخش ۲: استخدام، حضور و غیاب، مرخصی، تقویم
   ===================================================================== */

/* ------------------- درخواست‌های نیرو ------------------- */
function pageHiringReqs() {
  const canEdit = ['admin','hr_manager','hr_spec'].includes(session.role);
  return `
  ${pageHead('درخواست‌های نیرو', 'ثبت و پیگیری درخواست استخدام واحدها',
    canEdit?`<button class="btn btn-primary" onclick="hireReqForm()">➕ ثبت درخواست</button>`:'')}
  ${tbl(['شناسه','واحد متقاضی','پست موردنیاز','تعداد','دلیل درخواست','تاریخ','درخواست‌دهنده','وضعیت','عملیات'],
    RECRUIT_REQS.map(r=>`<tr>
      <td class="num">${r.id}</td><td>${deptName(r.dept)}</td><td>${r.pos}</td><td class="num">${fa(r.count)}</td>
      <td class="txth">${r.reason}</td><td class="num">${r.date}</td><td>${r.by}</td><td>${reqChip(r.status)}</td>
      <td><div class="row-actions">
        ${canEdit?`<button class="btn btn-sm btn-ghost" title="ویرایش" onclick="toast('ویرایش درخواست شبیه‌سازی شد','info')">✏️</button>
        <button class="btn btn-sm btn-ghost" title="حذف" onclick="toast('حذف درخواست شبیه‌سازی شد','info')">🗑️</button>`:'—'}
      </div></td></tr>`))}`;
}
function reqChip(s) {
  return {pending:'<span class="chip c-wait">در انتظار تأیید</span>', approved:'<span class="chip c-ok">تأییدشده</span>',
    rejected:'<span class="chip c-rej">ردشده</span>'}[s]||s;
}
function hireReqForm() {
  openModal({title:'ثبت درخواست نیرو',
    body:`<div class="form-grid">
      <div class="field"><label>واحد متقاضی <span class="req">*</span></label>
        <select id="hrDept">${Object.entries(DEPTS).map(([k,v])=>`<option value="${k}">${v.name}</option>`).join('')}</select></div>
      <div class="field"><label>پست موردنیاز <span class="req">*</span></label><input id="hrPos"><div class="err">اجباری</div></div>
      <div class="field"><label>تعداد <span class="req">*</span></label><input id="hrCount" type="number" min="1" value="1"></div>
      <div class="field"><label>دلیل درخواست</label><input id="hrReason"></div>
    </div>`,
    foot:`<button class="btn btn-primary" onclick="saveHireReq()">ثبت</button><button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`});
}
function saveHireReq() {
  if (!$('#hrPos').value.trim()) { $('#hrPos').closest('.field').classList.add('invalid'); return; }
  const num = 21 + RECRUIT_REQS.length;
  RECRUIT_REQS.unshift({id:'REQ-'+num, dept:$('#hrDept').value, pos:$('#hrPos').value.trim(),
    count:Number($('#hrCount').value)||1, reason:$('#hrReason').value.trim()||'—',
    status:'pending', date:'1404/07/12', by:session.name});
  closeModal(); toast('درخواست نیرو ثبت شد');
  renderPage('hiringReqs');
}

/* ------------------- آگهی‌های شغلی ------------------- */
function pageJobs() {
  const canEdit = ['admin','hr_manager','hr_spec'].includes(session.role);
  return `
  ${pageHead('آگهی‌های شغلی', 'مدیریت آگهی‌ها و کانال‌های انتشار',
    canEdit?`<button class="btn btn-primary" onclick="toast('ایجاد آگهی شبیه‌سازی شد','info')">➕ آگهی جدید</button>`:'')}
  ${tbl(['شناسه','عنوان شغلی','واحد','نوع همکاری','کانال انتشار','تاریخ انتشار','تعداد متقاضی','وضعیت','عملیات'],
    JOBS.map(j=>`<tr>
      <td class="num">${j.id}</td><td>${j.title}</td><td>${deptName(j.dept)}</td><td>${j.type}</td>
      <td class="txth">${j.channel}</td><td class="num">${j.date}</td><td class="num">${fa(j.apps)}</td>
      <td>${j.status==='open'?'<span class="chip c-ok">فعال</span>':'<span class="chip c-term">متوقف</span>'}</td>
      <td>${canEdit?`<button class="btn btn-sm btn-ghost" onclick="toast('مشاهده آگهی شبیه‌سازی شد','info')">👁️</button>`:'—'}</td></tr>`))}`;
}

/* ------------------- متقاضیان ------------------- */
function pageCands() {
  const canEdit = ['admin','hr_manager','hr_spec'].includes(session.role);
  const stMap = {'منابع انسانی':'c-done','فنی':'c-prob','مدیریت':'c-wait','پیشنهاد شغلی':'c-ok','حذف‌شده':'c-rej'};
  return `
  ${pageHead('بانک متقاضیان', 'پیگیری وضعیت بررسی، امتیازدهی و زمان‌بندی مصاحبه',
    `<button class="btn btn-ghost" onclick="toast('خروجی شبیه‌سازی شد','info')">⬇️ خروجی</button>
     ${canEdit?'<button class="btn btn-primary" onclick="toast(\'افزودن متقاضی شبیه‌سازی شد\',\'info\')">➕ افزودن</button>':''}`)}
  <div class="grid-3 mb">
    ${['منابع انسانی','فنی','پیشنهاد شغلی'].map(s=>`
      <div class="stat-card" style="--tint:#EFF6FF;--accent:#2563EB">
        <div class="stat-ico" style="background:#EFF6FF">🗂️</div>
        <div><div class="stat-val num">${fa(CANDS.filter(c=>c.stage===s).length)}</div>
        <div class="stat-label">مرحله: ${s}</div></div>
      </div>`).join('')}
  </div>
  ${tbl(['شناسه','نام متقاضی','آگهی','منبع','مرحله فعلی','امتیاز','تاریخ ثبت','عملیات'],
    CANDS.map(c=>{const j=JOBS.find(x=>x.id===c.job);return `<tr>
      <td class="num">${c.id}</td><td>${c.name}</td><td>${j?j.title:'—'}</td><td>${c.src}</td>
      <td><span class="chip ${stMap[c.stage]||'c-term'}">${c.stage}</span></td>
      <td class="num">${fa(c.score)}</td><td class="num">${c.ts}</td>
      <td><div class="row-actions">
        ${canEdit?`<button class="btn btn-sm btn-ghost" title="انتقال مرحله" onclick="moveCand('${c.id}')">➡️</button>
        <button class="btn btn-sm btn-ghost" title="زمان‌بندی مصاحبه" onclick="toast('دعوت به مصاحبه شبیه‌سازی شد','info')">📅</button>`:'—'}
      </div></td></tr>`}))}`;
}
function moveCand(id) {
  const c = CANDS.find(x=>x.id===id);
  const idx = STAGES.indexOf(c.stage);
  const next = STAGES[Math.min(idx+1, STAGES.length-1)];
  requireConfirm(`انتقال «${c.name}» به مرحله «${next}»؟`, ()=>{
    c.stage = next; toast('مرحله متقاضی به‌روزرسانی شد'); renderPage('candidates');
  });
}

/* ------------------- گردش کار جذب ------------------- */
function pageHiringFlow() {
  const stages = ['دریافت رزومه','غربالگری','مصاحبه فنی','مصاحبه مدیریت','پیشنهاد شغلی','استخدام'];
  const counts = [24, 12, 6, 3, 2, 1];
  const colors = ['#DBEAFE','#BFDBFE','#93C5FD','#60A5FA','#3B82F6','#2563EB'];
  return `
  ${pageHead('فرآیند جذب', 'گردش کار از غربالگری تا پیشنهاد نهایی — آگهی: برنامه‌نویس ارشد فرانت‌اند')}
  <div class="card card-pad">
    <div class="card-title">قیف استخدام</div>
    ${stages.map((s,i)=>`
      <div class="kpi-row"><span>${fa(i+1)}. ${s}</span><span class="num">${fa(counts[i])} نفر</span></div>
      <div class="progress" style="margin-bottom:10px"><span style="width:${Math.round(counts[i]/24*100)}%;background:${colors[i]}"></span></div>`).join('')}
  </div>
  <div class="legal-note" style="margin-top:14px">💡 در این قیف، هر مرحله شامل غربالگری رزومه، مصاحبه‌ها و ارزیابی است. متقاضیان ردشده به بانک رزومه منتقل و برای آگهی‌های بعدی قابل استفاده‌اند.</div>`;
}

/* ------------------- ثبت تردد ------------------- */
const ATTEND = [
  {emp:'EMP-003', in:'08:02', out:'17:05', late:2, oc:0},
  {emp:'EMP-004', in:'07:55', out:'17:00', late:0, oc:0},
  {emp:'EMP-007', in:'08:20', out:'19:10', late:20, oc:2},
  {emp:'EMP-010', in:'08:00', out:'18:30', late:0, oc:1.5},
  {emp:'EMP-011', in:'08:45', out:'17:00', late:45, oc:0},
  {emp:'EMP-005', in:'07:58', out:'18:00', late:0, oc:1},
];
function pageAttendance() {
  const canEdit = ['admin','hr_manager','hr_spec'].includes(session.role);
  return `
  ${pageHead('ثبت تردد', 'مشاهده و اصلاح رکوردهای ورود/خروج — امروز: دوشنبه ۱۴ مهر ۱۴۰۴',
    `<button class="btn btn-ghost" onclick="toast('خروجی شبیه‌سازی شد','info')">⬇️ خروجی</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'✅', tint:'#DCFCE7', accent:'#16A34A', val:fa(59), label:'حاضر امروز'})}
    ${statCard({ico:'⛔', tint:'#FEE2E2', accent:'#DC2626', val:fa(3), label:'غایب'})}
    ${statCard({ico:'⏰', tint:'#FEF3C7', accent:'#D97706', val:fa(4), label:'تأخیر امروز'})}
    ${statCard({ico:'🌙', tint:'#EDE9FE', accent:'#7C3AED', val:fa(6), label:'مأموریت / اضافه‌کار'})}
  </div>
  ${tbl(['کارمند','واحد','ورود','خروج','تأخیر (دقیقه)','اضافه‌کاری (ساعت)','عملیات'],
    ATTEND.map(a=>{const e=EMPS.find(x=>x.id===a.emp);return `<tr>
      <td>${empCell(a.emp)}</td><td>${deptName(e?.dept)}</td>
      <td class="num ltr">${a.in}</td><td class="num ltr">${a.out}</td>
      <td class="num">${fa(a.late)}</td><td class="num">${fa(a.oc)}</td>
      <td>${canEdit?`<button class="btn btn-sm btn-ghost" onclick="toast('اصلاح رکورد شبیه‌سازی شد','info')">✏️ اصلاح</button>`:'—'}</td></tr>`}))}`;
}

/* ------------------- مرخصی و ماموریت ------------------- */
function pageLeaveRequests() {
  const isEmp = session.role === 'employee';
  let list = LEAVES;
  if (isEmp) list = LEAVES.filter(l=>l.emp===session.emp);
  else if (session.role==='line_manager') {
    const team = EMPS.filter(e=>e.mgr===session.name).map(e=>e.id);
    list = LEAVES.filter(l=>team.includes(l.emp));
  }
  return `
  ${pageHead('مرخصی و ماموریت', isEmp?'درخواست‌ها و موجودی مرخصی شما':'درخواست‌های مرخصی کارکنان',
    `<button class="btn btn-primary" onclick="leaveForm()">➕ درخواست مرخصی</button>`)}
  ${tbl(['شناسه','کارمند','نوع','از تاریخ','تا تاریخ','مدت (روز)','دلیل','وضعیت','عملیات'],
    list.map(l=>`<tr>
      <td class="num">${l.id}</td><td>${empCell(l.emp)}</td><td>${l.type}</td>
      <td class="num">${l.from}</td><td class="num">${l.to}</td><td class="num">${fa(l.days)}</td>
      <td class="txth">${l.reason}</td><td>${lvChip(l.status)}</td>
      <td>${l.status==='pending'?`<button class="btn btn-sm btn-ghost" onclick="cancelLeave('${l.id}')">لغو</button>`:'—'}</td></tr>`))}`;
}
function cancelLeave(id) {
  requireConfirm('لغو این درخواست مرخصی؟', ()=>{
    const l = LEAVES.find(x=>x.id===id);
    if (l) { l.status='rejected'; toast('درخواست لغو شد'); renderPage(currentPage); }
  });
}

/* ------------------- فرم درخواست مرخصی ------------------- */
function leaveForm() {
  const me = EMPS.find(e=>e.id===(session.emp||'EMP-003'));
  openModal({title:'ثبت درخواست مرخصی',
    body:`
    <div class="legal-note">مانده مرخصی استحقاقی شما: <b class="num">${fa(me?me.leaveBal:0)} روز</b></div>
    <div class="form-grid">
      <div class="field"><label>نوع مرخصی <span class="req">*</span></label>
        <select id="lvType"><option>استحقاقی</option><option>مریضی</option><option>بدون حقوق</option><option>ماموریت</option></select></div>
      <div class="field"><label>از تاریخ <span class="req">*</span></label><input id="lvFrom" class="num" placeholder="1404/07/20"><div class="err">تاریخ شروع اجباری است</div></div>
      <div class="field"><label>تا تاریخ <span class="req">*</span></label><input id="lvTo" class="num" placeholder="1404/07/21"><div class="err">تاریخ پایان اجباری است</div></div>
      <div class="field"><label>مدت (روز) <span class="req">*</span></label><input id="lvDays" type="number" min="1" value="1"></div>
      <div class="field full"><label>دلیل</label><textarea id="lvReason" rows="2"></textarea></div>
    </div>`,
    foot:`<button class="btn btn-primary" onclick="saveLeave()">ثبت درخواست</button>
          <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`});
}
function saveLeave() {
  let ok = true;
  for (const s of ['#lvFrom','#lvTo','#lvDays']) {
    const i = $(s), f = i.closest('.field');
    if (!i.value.trim()) { f.classList.add('invalid'); ok=false; } else f.classList.remove('invalid');
  }
  if (!ok) { toast('فیلدهای اجباری را تکمیل کنید','error'); return; }
  const id = 'LV-'+(leaveSeq++);
  LEAVES.unshift({id, emp:session.emp||'EMP-003', type:$('#lvType').value,
    from:$('#lvFrom').value.trim(), to:$('#lvTo').value.trim(), days:Number($('#lvDays').value)||1,
    status:'pending', reason:$('#lvReason').value.trim()||'—', ts:'1404/07/12'});
  closeModal();
  toast('درخواست مرخصی ثبت شد و برای تأیید ارسال گردید');
  if (PAGE_ACCESS.leaveApprove?.includes(session.role)) renderPage('leaveRequests');
  else renderPage('leaveRequests');
}

/* ------------------- تأیید مرخصی (نمای مدیر) ------------------- */
function pageLeaveApprove() {
  const pend = LEAVES.filter(l=>l.status==='pending');
  const isEmpOnly = ['hr_spec','employee'].includes(session.role);
  if (isEmpOnly) return `<div class="card"><div class="empty-state"><div class="big">🔒</div>
    این بخش برای نقش شما فعال نیست</div></div>`;
  return `
  ${pageHead('تأیید مرخصی', 'بررسی درخواست‌های در انتظار تأیید',
    `<span class="badge-pending num">${fa(pend.length)} درخواست در انتظار</span>`)}
  ${pend.length ? tbl(['شناسه','کارمند','واحد','نوع','از','تا','مدت','دلیل','اقدام'],
    pend.map(l=>{const e=EMPS.find(x=>x.id===l.emp);return `<tr>
      <td class="num">${l.id}</td><td>${empCell(l.emp)}</td><td>${deptName(e?.dept)}</td><td>${l.type}</td>
      <td class="num">${l.from}</td><td class="num">${l.to}</td><td class="num">${fa(l.days)} روز</td>
      <td class="txth">${l.reason}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-primary" onclick="approveLeave('${l.id}')">✅ تأیید</button>
        <button class="btn btn-sm btn-danger" onclick="rejectLeave('${l.id}')">⛔ رد</button>
      </div></td></tr>`}))
    : `<div class="card"><div class="empty-state"><div class="big">🎉</div>درخواست در انتظاری وجود ندارد</div></div>`}`;
}
function approveLeave(id) {
  const l = LEAVES.find(x=>x.id===id);
  requireConfirm(`تأیید مرخصی ${l.id} برای «${EMPS.find(e=>e.id===l.emp)?.name}»؟`, ()=>{
    l.status='approved'; l.approver=session.name;
    const e = EMPS.find(e=>e.id===l.emp); if (e && l.type==='استحقاقی') e.leaveBal = Math.max(0, e.leaveBal - l.days);
    toast('مرخصی تأیید شد'); renderPage('leaveApprove'); renderSidebar();
  });
}
function rejectLeave(id) {
  const l = LEAVES.find(x=>x.id===id);
  requireConfirm(`رد مرخصی ${l.id}؟`, ()=>{
    l.status='rejected'; l.approver=session.name;
    toast('مرخصی رد شد'); renderPage('leaveApprove'); renderSidebar();
  });
}

/* ------------------- اضافه‌کاری ------------------- */
const OVERTIME = [
  {emp:'EMP-007', date:'1404/07/12', hours:2, status:'approved'},
  {emp:'EMP-010', date:'1404/07/11', hours:1.5, status:'approved'},
  {emp:'EMP-005', date:'1404/07/10', hours:1, status:'pending'},
];
function pageOvertime() {
  return `
  ${pageHead('اضافه‌کاری', 'ثبت، تأیید و محاسبه اضافه‌کاری',
    `['admin','hr_manager','hr_spec'].includes(session.role)?'<button class=\"btn btn-primary\" onclick=\"toast(\\'ثبت اضافه‌کاری شبیه‌سازی شد\\',\\'info\\')\">➕ ثبت</button>':''`)}
  ${tbl(['کارمند','تاریخ','مدت (ساعت)','نرخ','وضعیت'],
    OVERTIME.map(o=>`<tr>
      <td>${empCell(o.emp)}</td><td class="num">${o.date}</td><td class="num">${fa(o.hours)}</td><td>۱۴۰٪</td>
      <td>${lvChip(o.status)}</td></tr>`))}`;
}

/* ------------------- تقویم ------------------- */
function pageCalendar() {
  const y = 1404, m = calMonth;
  const evs = {
    4:[{cls:'ev-holiday', t:'تعطیل رسمی'}],
    8:[{cls:'ev-leave', t:'مرخصی — نگار'}],
    12:[{cls:'ev-oc', t:'اضافه‌کار — رضا'}],
    14:[{cls:'ev-leave', t:'مرخصی — نگار (۱/۳)'}],
    15:[{cls:'ev-leave', t:'مرخصی — نگار (۲/۳)'}],
    16:[{cls:'ev-leave', t:'مرخصی — نگار (۳/۳)'}],
    21:[{cls:'ev-leave', t:'مرخصی — فاطمه'}],
    22:[{cls:'ev-leave', t:'مرخصی — فاطمه'}],
    26:[{cls:'ev-shift', t:'دوره ایمنی'}],
    27:[{cls:'ev-shift', t:'دوره ایمنی'}],
  };
  const dow = ['ش','ی','د','س','چ','پ','ج'];
  return `
  ${pageHead('تقویم', 'نمایش شیفت‌ها و مرخصی‌ها به صورت ماهانه',
    `<button class="btn btn-ghost" onclick="calMonth--;renderPage('calendar')">‹ ماه قبل</button>
     <button class="btn btn-ghost" onclick="calMonth++;renderPage('calendar')">ماه بعد ›</button>`)}
  <div class="card card-pad">
    <div class="card-title">مهر ۱۴۰۴ <span class="muted">شیفت اداری — ۸ تا ۱۷</span></div>
    <div class="cal-grid">
      ${dow.map(d=>`<div class="cal-dow">${d}</div>`).join('')}
      ${Array.from({length:30},(_,i)=>i+1).map(d=>{
        const ev = evs[d]||[];
        return `<div class="cal-cell">
          <div class="cal-day num">${fa(d)}</div>
          ${ev.map(e=>`<div class="cal-ev ${e.cls}">${e.t}</div>`).join('')}
        </div>`;}).join('')}
    </div>
    <div style="display:flex;gap:14px;font-size:.72rem;color:var(--muted);margin-top:12px">
      <span>🟨 مرخصی</span><span>🟦 اضافه‌کاری</span><span>🟪 شیفت/دوره</span><span>🟥 تعطیل</span>
    </div>
  </div>`;
}
