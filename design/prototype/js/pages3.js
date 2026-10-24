/* =====================================================================
   صفحات — بخش ۳: حقوق و دستمزد، عملکرد، آموزش، رفاه، گزارش‌ها، تنظیمات
   ===================================================================== */

/* ------------------- پردازش حقوق ------------------- */
function pagePayrollRun() {
  return `
  ${pageHead('پردازش حقوق', 'اجرای محاسبات ماهانه، فرمول‌های مالیات، بیمه و اضافه‌کاری',
    `<button class="btn btn-primary" onclick="runPayroll()">▶️ اجرای پردازش حقوق</button>
     <button class="btn btn-ghost" onclick="toast('قفل دوره مالی شبیه‌سازی شد','info')">🔒 قفل دوره</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'👥', tint:'#EFF6FF', accent:'#2563EB', val:fa(66), label:'پرسنل شامل محاسبه'})}
    ${statCard({ico:'💵', tint:'#DCFCE7', accent:'#16A34A', val:'۲٫۹ میلیارد', label:'جمع پرداختی ماه جاری (ریال)'})}
    ${statCard({ico:'🧾', tint:'#FEF3C7', accent:'#D97706', val:fa(320)+' م', label:'مالیات ماه جاری (میلیون ریال)'})}
    ${statCard({ico:'🛡️', tint:'#EDE9FE', accent:'#7C3AED', val:fa(233)+' م', label:'بیمه ماه جاری (میلیون ریال)'})}
  </div>
  ${tbl(['شناسه','ماه','تعداد','ناخالص (م ریال)','مالیات','بیمه','خالص (م ریال)','وضعیت','عملیات'],
    PAYRUNS.map(p=>`<tr>
      <td class="num">${p.id}</td><td>${p.month}</td><td class="num">${fa(p.count)}</td>
      <td class="num">${fa(p.gross)}</td><td class="num">${fa(p.tax)}</td><td class="num">${fa(p.ins)}</td>
      <td class="num">${fa(p.net)}</td>
      <td>${p.status==='پردازش‌شده'?'<span class="chip c-ok">پردازش‌شده</span>':'<span class="chip c-wait">پیش‌نویس</span>'}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('جزئیات دوره مالی شبیه‌سازی شد','info')">👁️</button></td></tr>`))}`;
}
function runPayroll() {
  requireConfirm('اجرای پردازش حقوق ماه جاری برای ۶۶ نفر؟ (نمونه نمایشی)', ()=>{
    toast('پردازش حقوق با موفقیت انجام شد (نمونه)');
    renderPage('payrollRun');
  });
}

/* ------------------- فیش حقوقی ------------------- */
function pagePayslip() {
  const me = EMPS.find(e=>e.id===(session.emp||'EMP-003'));
  const tot = a => a.reduce((s,[,v])=>s+v,0);
  const gross = tot(PAYSLIP.earnings), ded = tot(PAYSLIP.deductions), net = gross-ded;
  const isHR = ['admin','hr_manager','hr_spec'].includes(session.role);
  return `
  ${pageHead('فیش حقوقی', 'نمایش و دانلود فیش حقوقی ماهانه',
    `<button class="btn btn-ghost" onclick="toast('دانلود PDF شبیه‌سازی شد','info')">⬇️ دانلود PDF</button>`)}
  ${isHR?`<div class="toolbar card" style="border-bottom:none;margin-bottom:14px">
    <select class="sel" style="min-width:220px"><option>${me.name} — ${me.id}</option></select>
    <select class="sel"><option>مهر ۱۴۰۴</option><option>شهریور ۱۴۰۴</option><option>مرداد ۱۴۰۴</option></select>
  </div>`:''}
  <div class="payslip card">
    <h3>فیش حقوقی ${PAYSLIP.month}</h3>
    <div class="muted" style="text-align:center">${me.name} — کد پرسنلی <span class="num">${me.id}</span></div>
    <div class="slip-grid">
      <div class="slip-col">
        <b style="font-size:.85rem">اجزای دریافتی</b>
        ${PAYSLIP.earnings.map(([l,v])=>`<div class="slip-row"><span>${l}</span><span class="num">${money(v)}</span></div>`).join('')}
        <div class="slip-total"><span>جمع دریافتی</span><span class="num">${money(gross)}</span></div>
      </div>
      <div class="slip-col">
        <b style="font-size:.85rem">کسورات</b>
        ${PAYSLIP.deductions.map(([l,v])=>`<div class="slip-row"><span>${l}</span><span class="num">${money(v)}</span></div>`).join('')}
        <div class="slip-total"><span>جمع کسورات</span><span class="num">${money(ded)}</span></div>
      </div>
    </div>
    <div class="slip-total" style="margin-top:14px;font-size:1.05rem">
      <span>خالص پرداختی (ریال)</span><span class="num">${money(net)}</span>
    </div>
    <div class="muted" style="margin-top:10px;display:flex;gap:18px;flex-wrap:wrap">
      <span>🏦 ${PAYSLIP.bank} — <span class="ltr num">${PAYSLIP.acc}</span></span>
      <span>📅 روزهای کاری: <b class="num">${fa(PAYSLIP.workdays)}</b></span>
      <span>🗓️ تاریخ پرداخت: <b class="num">${PAYSLIP.payDate}</b></span>
    </div>
  </div>`;
}

/* ------------------- اجزای حقوق ------------------- */
function pagePayItems() {
  return `
  ${pageHead('اجزای حقوق', 'تعریف آیتم‌های حقوقی، فرمول‌ها و معافیت‌ها',
    `<button class="btn btn-primary" onclick="toast('تعریف آیتم جدید شبیه‌سازی شد','info')">➕ آیتم جدید</button>`)}
  <div class="grid-2eq mb">
    <div class="card card-pad">
      <div class="card-title">آیتم‌های دریافتی</div>
      ${tbl(['عنوان','نوع','فرمول / مقدار'],[
        '<tr><td>حقوق مبنا</td><td>ماهانه</td><td class="num">حداقل حقوق مصوب</td></tr>',
        '<tr><td>حق مسکن</td><td>ماهانه</td><td class="num">۹٬۰۰۰٬۰۰۰ ریال</td></tr>',
        '<tr><td>بن کارگری</td><td>ماهانه</td><td class="num">۱۴٬۰۰۰٬۰۰۰ ریال</td></tr>',
        '<tr><td>اضافه‌کاری</td><td>متغیر</td><td class="num">ساعت × نرخ مبنا × ۱٫۴</td></tr>',
        '<tr><td>عیدی و سنوات</td><td>اقساطی</td><td class="num">مبنا ÷ ۱۲</td></tr>'],{foot:false})}
    </div>
    <div class="card card-pad">
      <div class="card-title">کسورات و معافیت‌ها</div>
      ${tbl(['عنوان','نوع','فرمول / مقدار'],[
        '<tr><td>مالیات حقوق</td><td>پله‌ای</td><td class="num">طبق جدول مصوب</td></tr>',
        '<tr><td>بیمه سهم کارمند</td><td>درصدی</td><td class="num">۷٪ مشمول</td></tr>',
        '<tr><td>قسط وام</td><td>س fixed</td><td class="num">۲٬۵۰۰٬۰۰۰ ریال</td></tr>',
        '<tr><td>معافیت مسکن و بن</td><td>معافیت</td><td class="num">معاف از مالیات</td></tr>'],{foot:false})}
    </div>
  </div>`;
}

/* ------------------- گزارش‌های مالی ------------------- */
function pagePayReports() {
  return `
  ${pageHead('گزارش‌های مالی', 'لیست پرداخت، گزارش بیمه و گزارش مالیات',
    `<button class="btn btn-ghost" onclick="toast('خروجی Excel شبیه‌سازی شد','info')">⬇️ Excel</button>
     <button class="btn btn-ghost" onclick="toast('خروجی PDF شبیه‌سازی شد','info')">⬇️ PDF</button>`)}
  <div class="grid-3 mb">
    <div class="stat-card" style="--tint:#EFF6FF;--accent:#2563EB"><div class="stat-ico" style="background:#EFF6FF">💵</div>
      <div><div class="stat-val num">۲٬۹۱۰ م</div><div class="stat-label">ناخالص پرداختی مهر (میلیون ریال)</div></div></div>
    <div class="stat-card" style="--tint:#FEF3C7;--accent:#D97706"><div class="stat-ico" style="background:#FEF3C7">🧾</div>
      <div><div class="stat-val num">۳۲۰ م</div><div class="stat-label">مالیات حقوق مهر</div></div></div>
    <div class="stat-card" style="--tint:#EDE9FE;--accent:#7C3AED"><div class="stat-ico" style="background:#EDE9FE">🛡️</div>
      <div><div class="stat-val num">۲۳۳ م</div><div class="stat-label">حق بیمه مهر</div></div></div>
  </div>
  ${tbl(['گزارش','دوره','تعداد ردیف','توضیح','عملیات'],[
    '<tr><td>لیست پرداخت حقوق</td><td>مهر ۱۴۰۴</td><td class="num">۶۶</td><td>خالص پرداختی به تفکیک پرسنل</td><td><button class="btn btn-sm btn-ghost" onclick="toast(\'خروجی شبیه‌سازی شد\',\'info\')">⬇️</button></td></tr>',
    '<tr><td>گزارش لیست بیمه</td><td>مهر ۱۴۰۴</td><td class="num">۶۶</td><td>مشمولین بیمه و حق بیمه</td><td><button class="btn btn-sm btn-ghost" onclick="toast(\'خروجی شبیه‌سازی شد\',\'info\')">⬇️</button></td></tr>',
    '<tr><td>گزارش مالیات حقوق</td><td>مهر ۱۴۰۴</td><td class="num">۶۶</td><td>مالیات متعلقه به تفکیک پرسنل</td><td><button class="btn btn-sm btn-ghost" onclick="toast(\'خروجی شبیه‌سازی شد\',\'info\')">⬇️</button></td></tr>'])}`;
}

/* ------------------- دوره‌های ارزیابی ------------------- */
function pagePerfCycles() {
  const canEdit = ['admin','hr_manager'].includes(session.role);
  return `
  ${pageHead('دوره‌های ارزیابی', 'تعریف و مدیریت دوره‌های ارزیابی عملکرد',
    canEdit?`<button class="btn btn-primary" onclick="toast('ایجاد دوره شبیه‌سازی شد','info')">➕ دوره جدید</button>`:'')}
  ${tbl(['شناسه','عنوان دوره','بازه زمانی','تعداد ارزیابی‌شونده','تاریخ ایجاد','وضعیت','عملیات'],
    PERF_CYCLES.map(c=>`<tr>
      <td class="num">${c.id}</td><td>${c.name}</td><td class="num">${c.period}</td><td class="num">${fa(c.people)}</td>
      <td class="num">${c.ts}</td>
      <td>${c.status==='در حال اجرا'?'<span class="chip c-wait">در حال اجرا</span>':'<span class="chip c-ok">پایان‌یافته</span>'}</td>
      <td>${canEdit?`<button class="btn btn-sm btn-ghost" onclick="toast('ویرایش دوره شبیه‌سازی شد','info')">✏️</button>`:'—'}</td></tr>`))}`;
}

/* ------------------- فرم‌های ارزیابی ------------------- */
function pagePerfForms() {
  const team = session.role==='line_manager' ? EMPS.filter(e=>e.mgr===session.name) : EMPS.filter(e=>e.status!=='terminated');
  const done = 12, total = team.length;
  return `
  ${pageHead('فرم‌های ارزیابی', 'طراحی شاخص‌ها (KPI)، وزن‌دهی و تکمیل ارزیابی‌ها')}
  <div class="grid-2 mb">
    <div class="card card-pad">
      <div class="card-title">قالب شاخص‌های ارزیابی (KPI)</div>
      ${tbl(['شاخص','دسته','وزن'], PERF_TPL.map(t=>`<tr><td>${t.name}</td><td>${t.cat}</td><td class="num">٪${fa(t.w)}</td></tr>`), {foot:false})}
      <div class="muted" style="margin-top:8px">جمع وزن‌ها: <b class="num">٪۱۰۰</b></div>
    </div>
    <div class="card card-pad">
      <div class="card-title">پیشرفت ارزیابی دوره جاری</div>
      <div class="kpi-row"><span>تکمیل‌شده</span><span class="num">${fa(done)} از ${fa(total)}</span></div>
      <div class="progress" style="height:10px"><span style="width:${Math.round(done/total*100)}%"></span></div>
      <div class="muted" style="margin-top:12px">فرم‌های در انتظار اقدام شما:</div>
      <ul class="list-tight">
        ${team.slice(0,4).map(e=>`<li>📋 <div><b>${e.name}</b><div class="muted">${e.pos}</div></div>
          <button class="btn btn-sm btn-primary" style="margin-inline-start:auto" onclick="toast('فرم ارزیابی شبیه‌سازی شد','info')">تکمیل</button></li>`).join('')}
      </ul>
    </div>
  </div>`;
}

/* ------------------- نتایج ارزیابی ------------------- */
function pagePerfResults() {
  const avg = (PERF_ROWS.filter(r=>r.score>0).reduce((s,r)=>s+r.score,0) / PERF_ROWS.filter(r=>r.score>0).length).toFixed(1);
  return `
  ${pageHead('نتایج ارزیابی', 'مشاهده و تحلیل نتایج، بازخورد')}
  <div class="grid-stats">
    ${statCard({ico:'⭐', tint:'#FEF3C7', accent:'#D97706', val:fa(avg), label:'میانگین نمره کل (از ۵)'})}
    ${statCard({ico:'👥', tint:'#EFF6FF', accent:'#2563EB', val:fa(PERF_ROWS.length), label:'ارزیابی‌شده در این نمایش'})}
    ${statCard({ico:'🏆', tint:'#DCFCE7', accent:'#16A34A', val:fa(PERF_ROWS.filter(r=>r.score>=4).length), label:'نمره ۴ و بالاتر'})}
    ${statCard({ico:'📉', tint:'#FEE2E2', accent:'#DC2626', val:fa(PERF_ROWS.filter(r=>r.score>0&&r.score<3).length), label:'نیازمند بهبود (زیر ۳)'})}
  </div>
  ${tbl(['کارمند','دوره','نمره (از ۵)','وضعیت'],
    PERF_ROWS.map(r=>`<tr>
      <td>${empCell(r.emp)}</td><td>${r.cycle}</td>
      <td><span class="chip ${r.score>=4?'c-ok':r.score>=3?'c-done':'c-rej'} num">${r.score?fa(r.score):'—'}</span></td>
      <td>${r.status==='نهایی'?'<span class="chip c-ok">نهایی</span>':'<span class="chip c-wait">در حال ارزیابی</span>'}</td></tr>`))}`;
}

/* ------------------- دوره‌های آموزشی ------------------- */
function pageTraining() {
  const isEmp = session.role==='employee';
  return `
  ${pageHead('دوره‌های آموزشی', 'تعریف دوره، ثبت‌نام و ظرفیت',
    isEmp?'':`<button class="btn btn-primary" onclick="toast('ایجاد دوره شبیه‌سازی شد','info')">➕ دوره جدید</button>`)}
  <div class="grid-stats mb" style="grid-template-columns:repeat(auto-fit,minmax(220px,1fr))">
    ${TRAININGS.map(t=>{
      const pct = Math.round(t.enrolled/t.cap*100);
      const full = t.enrolled>=t.cap;
      return `<div class="card card-pad">
        <div style="display:flex;justify-content:space-between;align-items:flex-start;gap:8px">
          <b style="font-size:.88rem">${t.title}</b><span class="chip contract">${t.cat}</span>
        </div>
        <div class="muted" style="margin:6px 0 10px">🗓️ ${t.dates} — ${t.mode}</div>
        <div class="kpi-row"><span>ظرفیت</span><span class="num">${fa(t.enrolled)}/${fa(t.cap)}</span></div>
        <div class="progress"><span style="width:${pct}%;background:${full?'#DC2626':'#2563EB'}"></span></div>
        <button class="btn btn-sm ${full?'btn-ghost':'btn-primary'}" style="margin-top:10px;width:100%;justify-content:center"
          ${full?'disabled':''} onclick="toast('${full?'ظرفیت دوره تکمیل است':'ثبت‌نام در دوره شبیه‌سازی شد'}','${full?'info':'success'}')">
          ${full?'تکمیل ظرفیت':'ثبت‌نام'}</button>
      </div>`;}).join('')}
  </div>`;
}

/* ------------------- سوابق آموزشی ------------------- */
function pageTrainingHist() {
  const hist = session.role==='employee' ? TRAIN_HIST.filter(t=>t.emp===session.emp) : TRAIN_HIST;
  return `
  ${pageHead('سوابق آموزشی', 'گواهی‌نامه‌ها و مهارت‌های کسب‌شده')}
  ${tbl(['کارمند','دوره','تاریخ','ساعت','نتیجه','گواهی','عملیات'],
    hist.map(h=>`<tr>
      <td>${empCell(h.emp)}</td><td>${h.course}</td><td class="num">${h.date}</td><td class="num">${fa(h.hours)}</td>
      <td><span class="chip c-ok">${h.result}</span></td><td class="num ltr">${h.cert}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('دانلود گواهی شبیه‌سازی شد','info')">⬇️</button></td></tr>`))}`;
}

/* ------------------- نیازسنجی آموزشی ------------------- */
function pageTrainingNeeds() {
  return `
  ${pageHead('نیازسنجی آموزشی', 'تحلیل شکاف مهارتی کارکنان')}
  ${tbl(['مهارت','سطح موردنیاز','سطح فعلی','شکاف','اولویت','اقدام'],
    SKILL_GAP.map(g=>`<tr>
      <td>${g.skill}</td><td>${g.need}</td><td>${g.have}</td>
      <td><span class="chip ${g.gap==='زیاد'?'c-rej':g.gap==='متوسط'?'c-wait':'c-ok'}">${g.gap}</span></td>
      <td>${g.pri==='فوری'?'<span class="chip c-rej">فوری</span>':'<span class="chip c-done">معمولی</span>'}</td>
      <td><button class="btn btn-sm btn-primary" onclick="toast('معرفی به دوره مرتبط شبیه‌سازی شد','info')">معرفی به دوره</button></td></tr>`))}`;
}

/* ------------------- تسهیلات رفاهی ------------------- */
function pageWelfare() {
  return `
  ${pageHead('تسهیلات رفاهی', 'وام، بیمه تکمیلی، سفر و سایر مزایا')}
  <div class="grid-3 mb">
    ${[['🛍️','وام ضروری','تا ۳ برابر حقوق پایه با بازپرداخت ۱۸ ماهه'],
       ['🛡️','بیمه تکمیلی','پوشش همسر و فرزندان'],
       ['✈️','سفر و اردو','سفر سالانه سازمانی']].map(([i,t,d])=>`
      <div class="card card-pad" style="text-align:center">
        <div style="font-size:2rem">${i}</div><b>${t}</b>
        <div class="muted" style="margin-top:4px">${d}</div>
      </div>`).join('')}
  </div>
  ${tbl(['شناسه','کارمند','نوع تسهیل','مبلغ','توضیح','تاریخ','وضعیت'],
    WELFARE.map(w=>`<tr>
      <td class="num">${w.id}</td><td>${empCell(w.emp)}</td><td>${w.type}</td><td class="num">${w.amount}</td>
      <td class="txth">${w.desc}</td><td class="num">${w.ts}</td><td>${lvChip(w.status)}</td></tr>`))}`;
}

/* ------------------- درخواست‌های رفاهی ------------------- */
function pageWelfareReq() {
  const list = session.role==='employee' ? WELFARE.filter(w=>w.emp===session.emp) : WELFARE;
  return `
  ${pageHead('درخواست‌های رفاهی', 'ثبت و پیگیری درخواست‌های رفاهی',
    `<button class="btn btn-primary" onclick="welfareForm()">➕ درخواست جدید</button>`)}
  ${list.length?tbl(['شناسه','کارمند','نوع','مبلغ','توضیح','تاریخ','وضعیت'],
    list.map(w=>`<tr>
      <td class="num">${w.id}</td><td>${empCell(w.emp)}</td><td>${w.type}</td><td class="num">${w.amount}</td>
      <td class="txth">${w.desc}</td><td class="num">${w.ts}</td><td>${lvChip(w.status)}</td></tr>`))
    :'<div class="card"><div class="empty-state"><div class="big">📭</div>درخواستی ثبت نشده است</div></div>'}`;
}
function welfareForm() {
  openModal({title:'ثبت درخواست رفاهی',
    body:`<div class="form-grid">
      <div class="field"><label>نوع درخواست <span class="req">*</span></label>
        <select id="wfType"><option>وام ضروری</option><option>بیمه تکمیلی</option><option>سفر و اردو</option></select></div>
      <div class="field"><label>مبلغ (ریال)</label><input id="wfAmount" class="num ltr" placeholder="0"></div>
      <div class="field full"><label>توضیحات</label><textarea id="wfDesc" rows="2"></textarea></div>
    </div>`,
    foot:`<button class="btn btn-primary" onclick="saveWelfare()">ثبت</button><button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`});
}
function saveWelfare() {
  const id = 'WLF-'+(14+WELFARE.length);
  WELFARE.unshift({id, emp:session.emp||'EMP-003', type:$('#wfType').value,
    amount:Number($('#wfAmount').value)?money(+$('#wfAmount').value)+' ریال':'—',
    status:'pending', ts:'1404/07/12', desc:$('#wfDesc').value.trim()||'—'});
  closeModal(); toast('درخواست رفاهی ثبت شد'); renderPage('welfareReq');
}

/* ------------------- گزارش‌ها ------------------- */
function pageReports() {
  const cards = [
    ['📑','لیست پرسنل','عملیاتی','فهرست کامل کارکنان به تفکیک واحد و وضعیت'],
    ['🚪','گزارش ترک خدمت','عملیاتی','خروجی‌های سال جاری با دلیل خروج'],
    ['🕐','ورود و خروج','عملیاتی','خلاصه تردد ماهانه کارکنان'],
    ['📈','نرخ جذب','تحلیلی','روند جذب نیرو در ۱۲ ماه اخیر'],
    ['📉','نرخ ترک خدمت','تحلیلی','مقایسه attrition با میانگین صنعت'],
    ['⚡','بهره‌وری','تحلیلی','شاخص‌های بهره‌وری به تفکیک واحد'],
  ];
  return `
  ${pageHead('گزارش‌ها', 'گزارش‌های عملیاتی و تحلیلی با خروجی Excel / PDF')}
  <div class="grid-3 mb">
    ${cards.map(([i,t,cat,d])=>`
      <div class="card card-pad">
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span style="font-size:1.6rem">${i}</span><span class="chip ${cat==='تحلیلی'?'c-done':'c-active'}">${cat}</span>
        </div>
        <b style="display:block;margin:8px 0 4px">${t}</b>
        <div class="muted" style="margin-bottom:10px">${d}</div>
        <div style="display:flex;gap:6px">
          <button class="btn btn-sm btn-primary" onclick="toast('خروجی Excel شبیه‌سازی شد','info')">⬇️ Excel</button>
          <button class="btn btn-sm btn-ghost" onclick="toast('خروجی PDF شبیه‌سازی شد','info')">⬇️ PDF</button>
        </div>
      </div>`).join('')}
  </div>`;
}

/* ------------------- مدیریت کاربران ------------------- */
function pageUsers() {
  return `
  ${pageHead('مدیریت کاربران', 'ایجاد کاربر، تعیین نقش و وضعیت حساب',
    `<button class="btn btn-primary" onclick="userForm()">➕ کاربر جدید</button>`)}
  ${tbl(['شناسه','نام کاربری','نام و نام خانوادگی','نقش','وضعیت','آخرین ورود','عملیات'],
    USERS.map(u=>`<tr>
      <td class="num">${u.id}</td><td class="ltr">${u.username}</td><td>${u.name}</td>
      <td>${ROLE_NAMES[u.role]||u.role}</td>
      <td>${u.status==='active'?'<span class="chip c-active">فعال</span>':'<span class="chip c-rej">قفل</span>'}</td>
      <td class="num ltr">${u.last}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-ghost" onclick="userForm('${u.id}')">✏️</button>
        <button class="btn btn-sm btn-ghost" onclick="toast('${u.status==='active'?'قفل حساب شبیه‌سازی شد':'بازکردن حساب شبیه‌سازی شد'}','info')">🔒</button>
      </div></td></tr>`))}`;
}
function userForm(id) {
  const u = id ? USERS.find(x=>x.id===id) : null;
  openModal({title:u?'ویرایش کاربر':'ایجاد کاربر جدید',
    body:`<div class="form-grid">
      <div class="field"><label>نام کاربری <span class="req">*</span></label><input id="ufUser" class="ltr" value="${u?u.username:''}"><div class="err">اجباری</div></div>
      <div class="field"><label>نام و نام خانوادگی <span class="req">*</span></label><input id="ufName" value="${u?u.name:''}"><div class="err">اجباری</div></div>
      <div class="field"><label>نقش <span class="req">*</span></label>
        <select id="ufRole">${ROLES.map(r=>`<option value="${r.id}" ${u&&u.role===r.id?'selected':''}>${r.name}</option>`).join('')}</select></div>
      <div class="field"><label>وضعیت</label>
        <select id="ufStatus"><option value="active">فعال</option><option value="locked">قفل</option></select></div>
    </div>`,
    foot:`<button class="btn btn-primary" onclick="saveUser('${id||''}')">💾 ذخیره</button><button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`});
}
function saveUser(id) {
  if (!$('#ufUser').value.trim() || !$('#ufName').value.trim()) {
    ['#ufUser','#ufName'].forEach(s=>{ if(!$(s).value.trim()) $(s).closest('.field').classList.add('invalid'); });
    toast('فیلدهای اجباری را تکمیل کنید','error'); return;
  }
  toast(id?'کاربر به‌روزرسانی شد':'کاربر جدید ایجاد شد (نمونه)'); closeModal();
}

/* ------------------- نقش‌ها و دسترسی‌ها ------------------- */
function pageRoles() {
  return `
  ${pageHead('نقش‌ها و دسترسی‌ها', 'مدیریت نقش‌های سیستمی و شرح دسترسی هر نقش')}
  ${tbl(['نقش','تعداد کاربران','شرح دسترسی','عملیات'],
    ROLES.map(r=>`<tr>
      <td><b>${r.name}</b></td><td class="num">${fa(r.users)}</td><td class="txth" style="max-width:420px;white-space:normal">${r.desc}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('ویرایش دسترسی‌ها شبیه‌سازی شد','info')">🛡️ ویرایش دسترسی</button></td></tr>`))}`;
}

/* ------------------- تنظیمات سازمانی ------------------- */
function pageOrgSettings() {
  return `
  ${pageHead('تنظیمات سازمانی', 'واحدها، پست‌ها و تقویم کاری',
    `<button class="btn btn-ghost" onclick="toast('تنظیم تقویم کاری شبیه‌سازی شد','info')">🗓️ تقویم کاری</button>`)}
  <div class="grid-2eq mb">
    <div class="card card-pad">
      <div class="card-title">واحدهای سازمانی</div>
      ${tbl(['کد','نام واحد','عملیات'], Object.entries(DEPTS).map(([k,v])=>`<tr>
        <td class="num">${v.code}</td><td>${v.name}</td>
        <td><button class="btn btn-sm btn-ghost" onclick="unitForm('${k}','${v.name}')">✏️</button></td></tr>`), {foot:false})}
    </div>
    <div class="card card-pad">
      <div class="card-title">پست‌های سازمانی</div>
      ${tbl(['عنوان پست','واحد'], [['مدیر','همه واحدها'],['کارشناس','همه واحدها'],['سرپرست','تولید'],['اپراتور','تولید'],['کارمند اداری','پشتیبانی']]
        .map(([p,u])=>`<tr><td>${p}</td><td>${u}</td></tr>`), {foot:false})}
    </div>
  </div>`;
}

/* ------------------- پارامترهای سیستم ------------------- */
function pageSysParams() {
  return `
  ${pageHead('پارامترهای سیستم', 'کدهای دسترسی، سطوح تأیید و پارامترهای کلیدی',
    `<button class="btn btn-ghost" onclick="toast('تغییرات ذخیره شد (نمونه)')">💾 ذخیره</button>`)}
  ${tbl(['شناسه','نام پارامتر','گروه','مقدار فعلی','عملیات'],
    SYS_PARAMS.map(p=>`<tr>
      <td class="num">${p.id}</td><td>${p.name}</td><td><span class="chip c-done">${p.group}</span></td>
      <td>${p.value}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('ویرایش پارامتر شبیه‌سازی شد','info')">✏️ ویرایش</button></td></tr>`))}`;
}

/* ------------------- لاگ حسابرسی ------------------- */
function pageAudit() {
  return `
  ${pageHead('لاگ حسابرسی', 'ثبت رویدادهای امنیتی و تغییرات مهم سیستم',
    `<button class="btn btn-ghost" onclick="toast('خروجی شبیه‌سازی شد','info')">⬇️ خروجی</button>`)}
  ${tbl(['شناسه','کاربر','رویداد','زمان','IP'],
    AUDIT.map(a=>`<tr>
      <td class="num">${a.id}</td><td class="ltr">${a.user}</td><td>${a.action}</td>
      <td class="num">${a.ts}</td><td class="num ltr">${a.ip}</td></tr>`))}`;
}

/* =====================================================================
   مسیریاب مرکزی صفحات
   ===================================================================== */
function renderPage(page, ...args) {
  const map = {
    // داشبورد
    dashboard:pageDashboard,
    // واحد امور کارکنان
    employees:pageEmployees, profile:pageProfile, personnelServices:pagePersonnelServices,
    attendance:pageAttendance, leaveRequests:pageLeaveRequests, leaveApprove:pageLeaveApprove,
    overtime:pageOvertime, calendar:pageCalendar,
    payrollRun:pagePayrollRun, payslip:pagePayslip, payItems:pagePayItems, payReports:pagePayReports,
    welfare:pageWelfare, welfareReq:pageWelfareReq,
    // واحد برنامه‌ریزی نیروی انسانی
    org:pageOrg, positions:pagePositions,
    appointments:pageAppointments, promotions:pagePromotions, careerPath:pageCareerPath,
    hiringReqs:pageHiringReqs, jobs:pageJobs, candidates:pageCands, hiringFlow:pageHiringFlow,
    perfCycles:pagePerfCycles, perfForms:pagePerfForms, perfResults:pagePerfResults,
    // واحد روابط کار و مددکاری اجتماعی
    disciplineCases:pageDisciplineCases, disciplineMeetings:pageDisciplineMeetings,
    socialWorkCases:pageSocialWorkCases, aidRequests:pageAidRequests,
    treatmentCases:pageTreatmentCases, periodicExams:pagePeriodicExams, medicalServices:pageMedicalServices,
    // واحد هماهنگی آموزش
    trainingPlan:pageTrainingPlan, trainingNeeds:pageTrainingNeeds, trainingExecution:pageTrainingExecution,
    training:pageTraining, trainingHist:pageTrainingHist,
    // واحد آمار و گزارشات و تنظیمات
    reports:pageReports, orgProfile:pageOrgProfile,
    users:pageUsers, roles:pageRoles, orgSettings:pageOrgSettings,
    sysParams:pageSysParams, audit:pageAudit,
  };
  // صفحه‌های زنده (live.js / live_records.js) پیش از قالب‌های ثابت بررسی می‌شوند: اینها
  // همان منوهای نمایشی‌اند که به API واقعی سرور وصل شده‌اند و اگر سرور نباشد، خودشان
  // به صفحهٔ نمایشی برمی‌گردند. پس ترتیب مهم است.
  if (typeof LIVE_PAGES !== 'undefined' && LIVE_PAGES[page]) { $('#main').innerHTML = livePage(page, ...args); return; }
  if (map[page]) { $('#main').innerHTML = map[page](...args); return; }
  // صفحهٔ ناموجود هم جای خودش را نشان می‌دهد تا گم شدن صفحه در سکوت نماند.
  const fn = (typeof pageFromSpec === 'function' ? () => pageFromSpec(page) : pageDashboard);
  $('#main').innerHTML = fn(...args);
}
