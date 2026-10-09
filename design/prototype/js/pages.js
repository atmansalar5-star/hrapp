/* =====================================================================
   صفحات — بخش ۱: داشبورد، کارکنان، پرونده، ساختار سازمانی
   ===================================================================== */

let listState = {q:'', dept:'', status:'', contract:''};
let profileTab = 'info';
let calMonth = 7; // مهر

function pageDashboard() {
  const r = session.role;
  if (r === 'employee') return dashEmployee();
  if (r === 'line_manager') return dashManager();
  return dashHR();
}

/* ---------- داشبورد مدیر/کارشناس HR و Admin ---------- */
function dashHR() {
  const active = EMPS.filter(e=>e.status==='active').length;
  const pend = LEAVES.filter(l=>l.status==='pending').length;
  const onLeave = EMPS.filter(e=>e.status==='leave').length;
  const openJobs = JOBS.filter(j=>j.status==='open').length;
  const deptCounts = Object.keys(DEPTS).map(k=>({k, n:EMPS.filter(e=>e.dept===k && e.status!=='terminated').length}));
  const maxD = Math.max(...deptCounts.map(d=>d.n), 1);
  return `
  ${pageHead('داشبورد', `خوش آمدید ${session.name} — نمای کلی سامانه`,
    `<button class="btn btn-ghost" onclick="exportDlg()">⬇️ خروجی</button>
     <button class="btn btn-primary" onclick="go('leaveApprove')">✅ اقدامات سریع</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'👥', tint:'#EFF6FF', accent:'#2563EB', val:fa(active), label:'کارکنان فعال', link:'مشاهده', page:'employees'})}
    ${statCard({ico:'🌤️', tint:'#FEF3C7', accent:'#D97706', val:fa(pend), label:'درخواست مرخصی در انتظار تأیید', link:'تأیید', page:'leaveApprove'})}
    ${statCard({ico:'🏖️', tint:'#EDE9FE', accent:'#7C3AED', val:fa(onLeave), label:'در مرخصی امروز'})}
    ${statCard({ico:'📢', tint:'#DCFCE7', accent:'#16A34A', val:fa(openJobs), label:'آگهی شغلی باز', link:'مشاهده', page:'jobs'})}
  </div>
  <div class="grid-2 mb">
    <div class="card card-pad">
      <div class="card-title">نمودار توزیع نیرو در واحدها <span class="muted">تعداد نفرات</span></div>
      ${deptCounts.map(d=>`
        <div class="kpi-row"><span>${deptName(d.k)}</span><span class="num">${fa(d.n)} نفر</span></div>
        <div class="progress" style="margin-bottom:9px"><span style="width:${Math.round(d.n/maxD*100)}%;background:${d.n===maxD?'#2563EB':'#93C5FD'}"></span></div>`).join('')}
    </div>
    <div class="card card-pad">
      <div class="card-title">روند استخدام ۶ ماه اخیر <span class="muted">جذب در برابر ترک خدمت</span></div>
      ${barChart([
        {l:'فروردین', a:4, b:2},{l:'اردیبهشت', a:6, b:1},{l:'خرداد', a:3, b:3},
        {l:'تیر', a:5, b:2},{l:'مرداد', a:2, b:1},{l:'شهریور', a:4, b:2}
      ])}
      <div style="display:flex;gap:14px;font-size:.72rem;color:var(--muted);margin-top:10px">
        <span>🟦 جذب</span><span>🟥 ترک خدمت</span>
      </div>
    </div>
  </div>
  <div class="grid-2">
    <div class="card">
      <div class="card-title" style="padding:16px 18px 0">آخرین درخواست‌های مرخصی</div>
      <div class="tbl-wrap"><table class="tbl">
        <thead><tr><th>کارمند</th><th>نوع</th><th>از</th><th>تا</th><th>وضعیت</th></tr></thead>
        <tbody>${LEAVES.slice(0,5).map(l=>`<tr class="clickable" onclick="go('leaveApprove')">
          <td>${empCell(l.emp)}</td><td>${l.type}</td><td class="num">${l.from}</td><td class="num">${l.to}</td>
          <td>${lvChip(l.status)}</td></tr>`).join('')}</tbody>
      </table></div>
    </div>
    <div class="card card-pad">
      <div class="card-title">اعلان‌های اخیر</div>
      <ul class="list-tight">
        ${NOTIFS.map(n=>`<li><span>${n.ico}</span><div>${esc(n.title)}<div class="muted" style="font-size:.7rem">${n.time}</div></div></li>`).join('')}
      </ul>
    </div>
  </div>`;
}

/* ---------- داشبورد مدیر واحد ---------- */
function dashManager() {
  const team = EMPS.filter(e=>e.mgr===session.name && e.status!=='terminated');
  const pend = LEAVES.filter(l=>l.status==='pending' && team.some(t=>t.id===l.emp)).length;
  return `
  ${pageHead('داشبورد مدیر واحد', `واحد شما: ${deptName(EMPS.find(e=>e.name===session.name)?.dept||'D4')}`,
    `<button class="btn btn-primary" onclick="go('leaveApprove')">✅ تأیید مرخصی‌ها (${fa(pend)})</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'👥', tint:'#EFF6FF', accent:'#2563EB', val:fa(team.length), label:'کارکنان واحد شما', link:'مشاهده', page:'employees'})}
    ${statCard({ico:'🌤️', tint:'#FEF3C7', accent:'#D97706', val:fa(pend), label:'درخواست در انتظار تأیید', link:'بررسی', page:'leaveApprove'})}
    ${statCard({ico:'🕐', tint:'#DCFCE7', accent:'#16A34A', val:fa(3), label:'غایب امروز'})}
    ${statCard({ico:'⭐', tint:'#EDE9FE', accent:'#7C3AED', val:fa(4), label:'ارزیابی‌های باقیمانده', link:'ارزیابی', page:'perfCycles'})}
  </div>
  <div class="grid-2 mb">
    <div class="card">
      <div class="card-title" style="padding:16px 18px 0">کارکنان واحد شما</div>
      <div class="tbl-wrap"><table class="tbl">
        <thead><tr><th>نام</th><th>پست</th><th>وضعیت</th><th>مانده مرخصی</th></tr></thead>
        <tbody>${team.map(t=>`<tr class="clickable" onclick="openProfile('${t.id}')">
          <td>${empCell(t.id)}</td><td>${t.pos}</td><td>${stChip(t.status)}</td><td class="num">${fa(t.leaveBal)} روز</td></tr>`).join('')}</tbody>
      </table></div>
    </div>
    <div class="card card-pad">
      <div class="card-title">اقدامات سریع</div>
      <ul class="list-tight">
        <li>✅ <div><b>تأیید مرخصی</b><div class="muted">بررسی درخواست‌های در انتظار واحد</div></div><button class="btn btn-sm btn-primary" style="margin-inline-start:auto" onclick="go('leaveApprove')">برو</button></li>
        <li>📈 <div><b>ارزیابی عملکرد</b><div class="muted">تکمیل فرم‌های دوره جاری</div></div><button class="btn btn-sm btn-ghost" style="margin-inline-start:auto" onclick="go('perfForms')">برو</button></li>
        <li>🕐 <div><b>تردد امروز</b><div class="muted">مشاهده ورود و خروج اعضای واحد</div></div><button class="btn btn-sm btn-ghost" style="margin-inline-start:auto" onclick="go('attendance')">برو</button></li>
      </ul>
    </div>
  </div>`;
}

/* ---------- داشبورد کارمند ---------- */
function dashEmployee() {
  const me = EMPS.find(e=>e.id===session.emp);
  const myLeaves = LEAVES.filter(l=>l.emp===session.emp);
  return `
  ${pageHead('داشبورد کارمند', `خوش آمدید ${me.name} — ${me.pos}`)}
  <div class="grid-stats">
    ${statCard({ico:'🏖️', tint:'#EFF6FF', accent:'#2563EB', val:fa(me.leaveBal), label:'مانده مرخصی استحقاقی (روز)', link:'درخواست', page:'leaveRequests'})}
    ${statCard({ico:'🌤️', tint:'#FEF3C7', accent:'#D97706', val:fa(myLeaves.filter(l=>l.status==='pending').length), label:'درخواست در انتظار تأیید', link:'پیگیری', page:'leaveRequests'})}
    ${statCard({ico:'🧾', tint:'#DCFCE7', accent:'#16A34A', val:'مهر ۱۴۰۴', label:'آخرین فیش حقوقی', link:'مشاهده', page:'payslip'})}
    ${statCard({ico:'🎓', tint:'#EDE9FE', accent:'#7C3AED', val:fa(2), label:'دوره آموزشی فعال', link:'مشاهده', page:'training'})}
  </div>
  <div class="grid-2 mb">
    <div class="card">
      <div class="card-title" style="padding:16px 18px 0">درخواست‌های اخیر من</div>
      <div class="tbl-wrap"><table class="tbl">
        <thead><tr><th>نوع</th><th>از</th><th>تا</th><th>وضعیت</th></tr></thead>
        <tbody>${myLeaves.length?myLeaves.map(l=>`<tr>
          <td>${l.type}</td><td class="num">${l.from}</td><td class="num">${l.to}</td><td>${lvChip(l.status)}</td></tr>`).join(''):'<tr><td colspan="4" class="muted">درخواستی ثبت نشده است</td></tr>'}</tbody>
      </table></div>
    </div>
    <div class="card card-pad">
      <div class="card-title">اقدامات سریع</div>
      <ul class="list-tight">
        <li>🌤️ <div><b>ثبت درخواست مرخصی</b></div><button class="btn btn-sm btn-primary" style="margin-inline-start:auto" onclick="leaveForm()">ثبت</button></li>
        <li>🧾 <div><b>مشاهده فیش حقوقی</b></div><button class="btn btn-sm btn-ghost" style="margin-inline-start:auto" onclick="go('payslip')">مشاهده</button></li>
        <li>🎁 <div><b>درخواست رفاهی</b></div><button class="btn btn-sm btn-ghost" style="margin-inline-start:auto" onclick="go('welfareReq')">ثبت</button></li>
        <li>👤 <div><b>اطلاعات شخصی</b></div><button class="btn btn-sm btn-ghost" style="margin-inline-start:auto" onclick="go('profile')">مشاهده</button></li>
      </ul>
    </div>
  </div>`;
}

/* ------------------- فهرست کارکنان ------------------- */
function pageEmployees(q) {
  if (q !== undefined) { listState.q = q; }
  let list = EMPS.filter(e=>{
    if (listState.q && !e.name.includes(listState.q) && !e.id.includes(listState.q)) return false;
    if (listState.dept && e.dept !== listState.dept) return false;
    if (listState.status && e.status !== listState.status) return false;
    if (listState.contract && e.contract !== listState.contract) return false;
    if (session.role==='line_manager' && e.mgr !== session.name) return false;
    return true;
  });
  const filters = `
  <div class="toolbar">
    <input class="inp w-lg" id="fq" placeholder="🔍 جستجوی نام یا کد پرسنلی…" value="${esc(listState.q)}"
      oninput="listState.q=this.value" onchange="renderPage('employees')">
    <select class="sel" onchange="listState.dept=this.value;renderPage('employees')">
      <option value="">همه واحدها</option>
      ${Object.entries(DEPTS).map(([k,v])=>`<option value="${k}" ${listState.dept===k?'selected':''}>${v.name}</option>`).join('')}
    </select>
    <select class="sel" onchange="listState.status=this.value;renderPage('employees')">
      <option value="">همه وضعیت‌ها</option>
      <option value="active" ${listState.status==='active'?'selected':''}>فعال</option>
      <option value="leave" ${listState.status==='leave'?'selected':''}>در مرخصی</option>
      <option value="probation" ${listState.status==='probation'?'selected':''}>دوره آزمایشی</option>
      <option value="terminated" ${listState.status==='terminated'?'selected':''}>ترک خدمت</option>
    </select>
    <select class="sel" onchange="listState.contract=this.value;renderPage('employees')">
      <option value="">همه انواع قرارداد</option>
      <option value="تمام‌وقت" ${listState.contract==='تمام‌وقت'?'selected':''}>تمام‌وقت</option>
      <option value="پیمانکاری" ${listState.contract==='پیمانکاری'?'selected':''}>پیمانکاری</option>
    </select>
    <button class="btn btn-ghost btn-sm" onclick="listState={q:'',dept:'',status:'',contract:''};renderPage('employees')">پاک‌سازی فیلترها</button>
    <span class="rec-count" style="margin-inline-start:auto">تعداد رکورد: <b class="num">${fa(list.length)}</b></span>
  </div>`;
  const canEdit = ['admin','hr_manager','hr_spec'].includes(session.role);
  return `
  ${pageHead('مدیریت کارکنان', 'جستجو، فیلتر و مدیریت پرونده کارکنان',
    (canEdit?`<button class="btn btn-primary" onclick="empForm()">➕ ایجاد کارمند</button>`:'')+
    `<button class="btn btn-ghost" onclick="toast('خروجی Excel شبیه‌سازی شد','info')">⬇️ خروجی Excel</button>`)}
  <div class="card">${filters}
  <div class="tbl-wrap"><table class="tbl">
    <thead><tr>
      <th>کد پرسنلی</th><th>نام و نام خانوادگی</th><th>واحد</th><th>پست سازمانی</th><th>نوع قرارداد</th><th>وضعیت</th><th>عملیات</th>
    </tr></thead>
    <tbody>${list.map(e=>`<tr class="clickable" onclick="openProfile('${e.id}')">
      <td class="num">${e.id}</td>
      <td><div class="emp-cell"><div class="avatar sq">${initial(e.name)}</div><span class="txth">${e.name}</span></div></td>
      <td>${deptName(e.dept)}</td><td>${e.pos}</td>
      <td><span class="chip contract">${e.contract}</span></td>
      <td>${stChip(e.status)}</td>
      <td onclick="event.stopPropagation()"><div class="row-actions">
        <button class="btn btn-sm btn-ghost" title="مشاهده" onclick="openProfile('${e.id}')">👁️</button>
        ${canEdit?`<button class="btn btn-sm btn-ghost" title="ویرایش" onclick="empForm('${e.id}')">✏️</button>`:''}
        ${canEdit?`<button class="btn btn-sm btn-ghost" title="حذف" onclick="delEmp('${e.id}')">🗑️</button>`:''}
      </div></td>
    </tr>`).join('') || '<tr><td colspan="7"><div class="empty-state"><div class="big">📭</div>موردی یافت نشد</div></td></tr>'}</tbody>
  </table></div>
  <div class="tbl-foot"><span class="rec-count">نمایش ${fa(list.length)} از ${fa(EMPS.length)} رکورد</span>
    <div class="pager"><button onclick="toast('نمونه نمایشی — صفحه دوم موجود نیست','info')">‹</button><button class="on num">۱</button><button class="num" onclick="toast('نمونه نمایشی — صفحه دوم موجود نیست','info')">۲</button><button onclick="toast('نمونه نمایشی — صفحه دوم موجود نیست','info')">›</button></div>
  </div></div>`;
}

function stChip(s) {
  return {active:'<span class="chip c-active">فعال</span>', leave:'<span class="chip c-leave">در مرخصی</span>',
    probation:'<span class="chip c-prob">دوره آزمایشی</span>', terminated:'<span class="chip c-term">ترک خدمت</span>'}[s]||s;
}
function lvChip(s) {
  return {pending:'<span class="chip c-wait">در انتظار تأیید</span>', approved:'<span class="chip c-ok">تأیید شده</span>',
    rejected:'<span class="chip c-rej">رد شده</span>'}[s]||s;
}
function empCell(id) {
  const e = EMPS.find(x=>x.id===id);
  return e?`<div class="emp-cell"><div class="avatar sq">${initial(e.name)}</div>${e.name}</div>`:'—';
}

function delEmp(id) {
  const e = EMPS.find(x=>x.id===id);
  requireConfirm(`آیا از حذف کارمند «${e.name}» مطمئن هستید؟`, ()=>{
    EMPS.splice(EMPS.indexOf(e),1);
    toast('کارمند با موفقیت حذف شد');
    renderPage('employees');
  });
}

/* ------------------- فرم ایجاد/ویرایش کارمند ------------------- */
function empForm(id) {
  const e = id ? EMPS.find(x=>x.id===id) : null;
  openModal({
    title: e?`ویرایش کارمند — ${e.name}`:'ایجاد کارمند جدید',
    body: `
    <fieldset class="field-group"><legend>اطلاعات شخصی</legend>
      <div class="form-grid">
        <div class="field"><label>نام و نام خانوادگی <span class="req">*</span></label>
          <input id="efName" value="${e?e.name:''}"><div class="err">وارد کردن نام اجباری است</div></div>
        <div class="field"><label>کد ملی <span class="req">*</span></label>
          <input id="efNid" value="${e?e.nid:''}" class="ltr"><div class="err">وارد کردن کد ملی اجباری است</div></div>
        <div class="field"><label>تاریخ تولد</label><input id="efBirth" value="${e?e.birth:''}" class="num" placeholder="1370/01/01"></div>
        <div class="field"><label>وضعیت تأهل</label>
          <select id="efMarital"><option ${e&&e.marital==='مجرد'?'selected':''}>مجرد</option><option ${e&&e.marital==='متأهل'?'selected':''}>متأهل</option></select></div>
        <div class="field full"><label>نشانی</label><input id="efAddr" value="${e?e.addr:''}"></div>
      </div>
    </fieldset>
    <fieldset class="field-group"><legend>اطلاعات شغلی</legend>
      <div class="form-grid">
        <div class="field"><label>واحد سازمانی <span class="req">*</span></label>
          <select id="efDept">${Object.entries(DEPTS).map(([k,v])=>`<option value="${k}" ${e&&e.dept===k?'selected':''}>${v.name}</option>`).join('')}</select></div>
        <div class="field"><label>پست سازمانی <span class="req">*</span></label>
          <input id="efPos" value="${e?e.pos:''}"><div class="err">وارد کردن پست اجباری است</div></div>
        <div class="field"><label>مدیر مستقیم</label><input id="efMgr" value="${e?e.mgr:''}"></div>
        <div class="field"><label>تاریخ استخدام</label><input id="efHire" value="${e?e.hire:''}" class="num" placeholder="1400/01/01"></div>
      </div>
    </fieldset>
    <fieldset class="field-group"><legend>اطلاعات قرارداد</legend>
      <div class="form-grid">
        <div class="field"><label>نوع قرارداد</label>
          <select id="efContract"><option ${e&&e.contract==='تمام‌وقت'?'selected':''}>تمام‌وقت</option><option ${e&&e.contract==='پیمانکاری'?'selected':''}>پیمانکاری</option></select></div>
        <div class="field"><label>وضعیت</label>
          <select id="efStatus">
            <option value="active" ${e&&e.status==='active'?'selected':''}>فعال</option>
            <option value="leave" ${e&&e.status==='leave'?'selected':''}>در مرخصی</option>
            <option value="probation" ${e&&e.status==='probation'?'selected':''}>دوره آزمایشی</option>
            <option value="terminated" ${e&&e.status==='terminated'?'selected':''}>ترک خدمت</option>
          </select></div>
        <div class="field"><label>حقوق پایه (ریال)</label><input id="efBase" value="${e?e.base:''}" class="num ltr"></div>
      </div>
    </fieldset>`,
    foot: `<button class="btn btn-primary" onclick="saveEmp('${id||''}')">💾 ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`
  });
}
function saveEmp(id) {
  let ok = true;
  const req = [['#efName','نام'],['#efNid','کد ملی'],['#efPos','پست']];
  for (const [sel] of req) {
    const inp = $(sel), f = inp.closest('.field');
    if (!inp.value.trim()) { f.classList.add('invalid'); ok = false; } else f.classList.remove('invalid');
  }
  if (!ok) { toast('لطفاً فیلدهای اجباری را تکمیل کنید','error'); return; }
  const data = {
    name:$('#efName').value.trim(), nid:$('#efNid').value.trim(), birth:$('#efBirth').value.trim()||'—',
    marital:$('#efMarital').value, addr:$('#efAddr').value.trim()||'—',
    dept:$('#efDept').value, pos:$('#efPos').value.trim(), mgr:$('#efMgr').value.trim()||'—',
    hire:$('#efHire').value.trim()||'—', contract:$('#efContract').value, status:$('#efStatus').value,
    base:Number($('#efBase').value)||0,
  };
  if (id) { Object.assign(EMPS.find(x=>x.id===id), data); toast('اطلاعات کارمند به‌روزرسانی شد'); }
  else {
    const num = String(Math.max(...EMPS.map(x=>+x.id.split('-')[1]))+1).padStart(3,'0');
    EMPS.push({id:'EMP-'+num, email:'—', phone:'—', edu:'—', leaveBal:26, acc:'—', tel:'—', ...data});
    toast('کارمند جدید با موفقیت ایجاد شد');
  }
  closeModal(); renderPage('employees');
}

/* ------------------- پرونده کارمند ------------------- */
function openProfile(id) {
  const e = EMPS.find(x=>x.id===id);
  if (!e) return;
  profileTab = 'info';
  currentPage = 'profile';
  renderSidebar();
  renderPage('profile', id);
}
function pageProfile(id) {
  const e = EMPS.find(x=>x.id===id);
  if (!e) return pageEmployees();
  const canEdit = ['admin','hr_manager','hr_spec'].includes(session.role);
  const tabs = [['info','اطلاعات فردی'],['job','اطلاعات شغلی'],['docs','مدارک'],['skills','مهارت‌ها و آموزش'],['contract','قرارداد و حقوق']];
  const body = {
    info: () => `<dl class="kv">
      <dt>نام و نام خانوادگی</dt><dd>${e.name}</dd>
      <dt>کد پرسنلی</dt><dd class="num">${e.id}</dd>
      <dt>کد ملی</dt><dd class="num ltr">${e.nid}</dd>
      <dt>تاریخ تولد</dt><dd class="num">${e.birth}</dd>
      <dt>وضعیت تأهل</dt><dd>${e.marital||'—'}</dd>
      <dt>تلفن همراه</dt><dd class="num ltr">${e.phone}</dd>
      <dt>ایمیل</dt><dd class="ltr">${e.email}</dd>
      <dt>نشانی</dt><dd>${e.addr}</dd></dl>`,
    job: () => `<dl class="kv">
      <dt>واحد سازمانی</dt><dd>${deptName(e.dept)}</dd>
      <dt>پست سازمانی</dt><dd>${e.pos}</dd>
      <dt>مدیر مستقیم</dt><dd>${e.mgr}</dd>
      <dt>تاریخ استخدام</dt><dd class="num">${e.hire}</dd>
      <dt>سابقه در سازمان</dt><dd class="num">${fa(4)} سال و ${fa(2)} ماه</dd>
      <dt>وضعیت</dt><dd>${stChip(e.status)}</dd></dl>
      <div class="card-title" style="margin-top:18px">سوابق شغلی در سازمان</div>
      ${tbl(['از','تا','پست','واحد'],[
        `<tr><td class="num">1403/01</td><td>اکنون</td><td>${e.pos}</td><td>${deptName(e.dept)}</td></tr>`,
        `<tr><td class="num">1401/06</td><td class="num">1403/01</td><td>کارشناس</td><td>${deptName(e.dept)}</td></tr>`], {foot:false})}`,
    docs: () => `${tbl(['نوع مدرک','نام فایل','حجم','تاریخ بارگذاری','عملیات'],
      EMP_DOCS.map(d=>`<tr><td>${d.type}</td><td class="ltr">${d.name}</td><td class="num">${d.size}</td><td class="num">${d.ts}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('دانلود شبیه‌سازی شد','info')">⬇️ دانلود</button></td></tr>`), {foot:false})}
      ${canEdit?'<button class="btn btn-ghost btn-sm" style="margin-top:10px" onclick="toast(\'بارگذاری مدرک شبیه‌سازی شد\',\'info\')">➕ بارگذاری مدرک</button>':''}`,
    skills: () => `<div class="grid-2eq">
      <div><div class="card-title">مهارت‌ها</div>
        ${[['نرم‌افزار Office','پیشرفته'],['زبان انگلیسی','متوسط'],['مذاکره فروش','پیشرفته'],['مدیریت زمان','متوسط']].map(([s,l])=>`
        <div class="kpi-row"><span>${s}</span><span class="chip c-done">${l}</span></div>
        <div class="progress" style="margin-bottom:8px"><span style="width:${l==='پیشرفته'?90:l==='متوسط'?55:30}%"></span></div>`).join('')}
      </div>
      <div><div class="card-title">دوره‌های گذرانده</div>
        ${tbl(['دوره','تاریخ','نتیجه'], TRAIN_HIST.filter(t=>t.emp===e.id).map(t=>`<tr><td>${t.course}</td><td class="num">${t.date}</td><td><span class="chip c-ok">${t.result}</span></td></tr>`), {foot:false})}
      </div></div>`,
    contract: () => `<dl class="kv">
      <dt>نوع قرارداد</dt><dd><span class="chip contract">${e.contract}</span></dd>
      <dt>تاریخ شروع</dt><dd class="num">${e.hire}</dd>
      <dt>حقوق پایه</dt><dd class="num">${money(e.base)} ریال</dd>
      <dt>شماره حساب</dt><dd class="num ltr">${e.acc}</dd>
      <dt>مانده مرخصی</dt><dd class="num">${fa(e.leaveBal)} روز</dd></dl>`,
  };
  return `
  <div class="crumbs">منابع انسانی <span>‹</span> <a href="#" onclick="go('employees');return false" style="color:var(--primary);text-decoration:none">کارکنان</a> <span>‹</span> پرونده ${e.name}</div>
  ${pageHead('پرونده کارمند', '', canEdit?`<button class="btn btn-ghost" onclick="empForm('${e.id}')">✏️ ویرایش</button>`:'')}
  <div class="card">
    <div class="profile-head">
      <div class="avatar lg">${initial(e.name)}</div>
      <div>
        <div class="profile-name">${e.name}</div>
        <div class="profile-job">${e.pos} — ${deptName(e.dept)}</div>
        <div class="profile-tags">${stChip(e.status)}<span class="chip contract">${e.contract}</span><span class="chip c-done num">${e.id}</span></div>
      </div>
    </div>
    <div class="card-pad">
      <div class="tabs">${tabs.map(([k,l])=>`<button class="tab ${profileTab===k?'active':''}" onclick="profileTab='${k}';renderPage('profile','${e.id}')">${l}</button>`).join('')}</div>
      ${body[profileTab]()}
    </div>
  </div>`;
}

/* ------------------- ساختار سازمانی ------------------- */
function pageOrg() {
  const canEdit = ['admin','hr_manager'].includes(session.role);
  const renderNode = n => `<li>
    <div class="org-node ${n.children.length?'':'leaf'}">
      <div class="o-name">${n.name}</div>
      <div class="o-head">مدیر: ${n.head}</div>
      <div class="o-count num">${fa(n.count)} نفر</div>
      ${canEdit?`<div style="margin-top:6px;display:flex;gap:4px;justify-content:center">
        <button class="btn btn-sm btn-ghost" onclick="unitForm('${n.id}','${n.name}')">✏️</button>
        <button class="btn btn-sm btn-ghost" onclick="unitForm()">➕</button></div>`:''}
    </div>
    ${n.children.length?`<ul>${n.children.map(renderNode).join('')}</ul>`:''}
  </li>`;
  return `
  ${pageHead('ساختار سازمانی', 'نمودار درختی واحدها و پست‌های سازمانی',
    canEdit?`<button class="btn btn-primary" onclick="unitForm()">➕ افزودن واحد</button>`:'')}
  <div class="card org-wrap"><ul class="org-tree">
    <li><div class="org-node root"><div class="o-name">مدیریت عامل — سازمان نمونه</div><div class="o-count num">${fa(66)} نفر</div></div>
      <ul>${DEPT_TREE.map(renderNode).join('')}</ul></li>
  </ul></div>
  <div class="card">
    <div class="card-title" style="padding:16px 18px 0">فهرست واحدها</div>
    ${tbl(['کد','نام واحد','مدیر واحد','تعداد پرسنل','عملیات'],
      Object.entries(DEPTS).map(([k,v])=>`<tr><td class="num">${v.code}</td><td>${v.name}</td><td>${DEPT_TREE.find(d=>d.id===k)?.head||'—'}</td>
      <td class="num">${fa(EMPS.filter(e=>e.dept===k).length)}</td>
      <td>${canEdit?`<button class="btn btn-sm btn-ghost" onclick="unitForm('${k}','${v.name}')">✏️ ویرایش</button>`:'—'}</td></tr>`))}
  </div>`;
}
function unitForm(id, name) {
  openModal({narrow:true, title:id?'ویرایش واحد سازمانی':'افزودن واحد سازمانی',
    body:`<div class="field"><label>نام واحد <span class="req">*</span></label><input id="ufName" value="${name||''}"><div class="err">نام واحد اجباری است</div></div>
      <div class="field" style="margin-top:12px"><label>کد واحد</label><input id="ufCode" class="ltr"></div>`,
    foot:`<button class="btn btn-primary" onclick="saveUnit('${id||''}')">💾 ذخیره</button><button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`});
}
function saveUnit(id) {
  const name = $('#ufName').value.trim();
  if (!name) { $('#ufName').closest('.field').classList.add('invalid'); return; }
  toast(id?'واحد به‌روزرسانی شد':'واحد جدید ایجاد شد (نمونه نمایشی)');
  closeModal();
}
