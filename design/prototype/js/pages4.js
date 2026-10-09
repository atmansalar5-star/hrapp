/* =====================================================================
   صفحات — بخش ۴: زیرمجموعه‌های ادارات منابع انسانی
   واحد امور کارکنان · واحد برنامه‌ریزی نیروی انسانی ·
   واحد روابط کار و مددکاری اجتماعی · واحد هماهنگی آموزش
   ===================================================================== */

/** چیپ وضعیت با کلاس و متن دلخواه. */
const pChip = (cls, txt) => `<span class="chip ${cls}">${txt}</span>`;
const pMoney = v => `${money(v)} <span class="muted" style="font-size:.68rem">ریال</span>`;

/* ---------------------------------------------------------------------
   واحد امور کارکنان — خدمات پرسنلی و جبران خدمات
   --------------------------------------------------------------------- */

function pagePersonnelServices() {
  const rows = [
    {id:'SRV-1404-118', emp:'EMP-003', type:'گواهی اشتغال به کار', date:'۱۴۰۴/۰۷/۱۲', dest:'بانک ملت', st:'done'},
    {id:'SRV-1404-117', emp:'EMP-004', type:'معرفی‌نامه وام', date:'۱۴۰۴/۰۷/۱۰', dest:'صندوق کارکنان', st:'wait'},
    {id:'SRV-1404-116', emp:'EMP-008', type:'تمدید قرارداد', date:'۱۴۰۴/۰۷/۰۸', dest:'—', st:'wait'},
    {id:'SRV-1404-115', emp:'EMP-011', type:'تسویه‌حساب پایان همکاری', date:'۱۴۰۴/۰۷/۰۵', dest:'—', st:'progress'},
    {id:'SRV-1404-114', emp:'EMP-006', type:'گواهی سابقه بیمه', date:'۱۴۰۴/۰۷/۰۳', dest:'تأمین اجتماعی', st:'done'},
  ];
  const chip = s => ({done:pChip('c-ok','صادر شد'), wait:pChip('c-wait','در انتظار بررسی'), progress:pChip('c-prob','در حال انجام')})[s] || s;
  return `
  ${pageHead('خدمات پرسنلی', 'صدور گواهی‌ها، تمدید قرارداد و تسویه‌حساب پایان همکاری',
    `<button class="btn btn-primary" onclick="toast('فرم درخواست خدمت باز شد (نمونه)')">➕ ثبت درخواست خدمت</button>
     <button class="btn btn-ghost" onclick="toast('خروجی اکسل شبیه‌سازی شد','info')">⬇️ خروجی</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'📄', tint:'#EFF6FF', accent:'#2563EB', val:fa(5), label:'درخواست خدمت در جریان این هفته'})}
    ${statCard({ico:'⏳', tint:'#FEF3C7', accent:'#D97706', val:fa(2), label:'در انتظار بررسی کارشناس'})}
    ${statCard({ico:'🧾', tint:'#DCFCE7', accent:'#16A34A', val:fa(1), label:'تسویه‌حساب در حال انجام'})}
    ${statCard({ico:'🗂️', tint:'#EDE9FE', accent:'#7C3AED', val:fa(3), label:'مدرک بارگذاری‌نشده کارکنان'})}
  </div>
  ${tbl(['شماره درخواست','کارمند','نوع خدمت','تاریخ ثبت','مرجع','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td class="num">${r.id}</td><td>${empCell(r.emp)}</td><td>${r.type}</td>
      <td class="num">${r.date}</td><td>${r.dest}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('جزئیات درخواست خدمت (نمونه)','info')">👁️</button></td></tr>`))}`;
}

/* ---------------------------------------------------------------------
   واحد برنامه‌ریزی نیروی انسانی — ساختار، انتصابات، ترفیعات، کارراهه
   --------------------------------------------------------------------- */

function pagePositions() {
  const rows = [
    {code:'P-101', title:'مدیر منابع انسانی', unit:'منابع انسانی', group:'مدیریت', req:'کارشناسی ارشد + ۸ سال سابقه', holders:1},
    {code:'P-205', title:'کارشناس ارشد برنامه‌ریزی نیروی انسانی', unit:'منابع انسانی', group:'کارشناسی ارشد', req:'کارشناسی ارشد مدیریت + ۵ سال سابقه', holders:2},
    {code:'P-310', title:'کارشناس آموزش', unit:'منابع انسانی', group:'کارشناسی', req:'کارشناسی + ۳ سال سابقه آموزش', holders:2},
    {code:'P-412', title:'کارشناس مددکاری اجتماعی', unit:'منابع انسانی', group:'کارشناسی', req:'کارشناسی مددکاری اجتماعی', holders:1},
    {code:'P-501', title:'مسئول طب کار', unit:'بهداشت و درمان', group:'کارشناسی ارشد', req:'پزشک عمومی + دورهٔ طب کار', holders:1},
    {code:'P-604', title:'کارشناس انضباط کار', unit:'روابط کار', group:'کارشناسی', req:'کارشناسی حقوق + آشنایی با قانون کار', holders:1},
  ];
  return `
  ${pageHead('شناسنامهٔ پست‌ها', 'شرح شغل، شرط‌های احراز و گروه شغلی هر پست — مرجع انتصاب، ترفیع و نیازسنجی',
    `<button class="btn btn-primary" onclick="toast('فرم افزودن پست باز شد (نمونه)')">➕ افزودن پست</button>
     <button class="btn btn-ghost" onclick="toast('خروجی شناسنامهٔ پست‌ها شبیه‌سازی شد','info')">⬇️ خروجی</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'🗂️', tint:'#EFF6FF', accent:'#2563EB', val:fa(48), label:'پست مصوب سازمانی'})}
    ${statCard({ico:'👥', tint:'#DCFCE7', accent:'#16A34A', val:fa(41), label:'پست دارای تصدی'})}
    ${statCard({ico:'⚠️', tint:'#FEF3C7', accent:'#D97706', val:fa(7), label:'پست خالی (نیازمند تأمین نیرو)'})}
    ${statCard({ico:'🧩', tint:'#EDE9FE', accent:'#7C3AED', val:fa(5), label:'شرح شغل نیازمند بازنگری'})}
  </div>
  ${tbl(['کد پست','عنوان پست','واحد','گروه شغلی','شرط احراز','تعداد تصدی','عملیات'],
    rows.map(r=>`<tr>
      <td class="num">${r.code}</td><td><b>${r.title}</b></td><td>${r.unit}</td>
      <td>${pChip('c-ok', r.group)}</td><td class="txth" style="white-space:normal;max-width:320px">${r.req}</td>
      <td class="num">${fa(r.holders)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('ویرایش شرح شغل (نمونه)','info')">✏️</button></td></tr>`))}`;
}

function pageAppointments() {
  const rows = [
    {id:'ABL-1404-071', emp:'EMP-008', post:'کارشناس ارشد برنامه‌ریزی نیروی انسانی', unit:'منابع انسانی', type:'انتصاب', date:'۱۴۰۴/۰۷/۰۱', st:'issued'},
    {id:'ABL-1404-070', emp:'EMP-005', post:'مدیر فروش', unit:'فروش و بازاریابی', type:'تمدید تصدی', date:'۱۴۰۴/۰۶/۲۵', st:'issued'},
    {id:'ABL-1404-069', emp:'EMP-012', post:'سرپرست تولید', unit:'تولید و عملیات', type:'جابه‌جایی', date:'۱۴۰۴/۰۶/۱۸', st:'approve'},
    {id:'ABL-1404-068', emp:'EMP-009', post:'کارشناس آموزش', unit:'منابع انسانی', type:'انتصاب', date:'۱۴۰۴/۰۶/۱۰', st:'approve'},
    {id:'ABL-1404-067', emp:'EMP-004', post:'رئیس حسابداری', unit:'مالی و حسابداری', type:'پایان تصدی', date:'۱۴۰۴/۰۶/۰۵', st:'issued'},
  ];
  const chip = s => ({issued:pChip('c-ok','ابلاغ صادر شد'), approve:pChip('c-wait','در انتظار تأیید')})[s] || s;
  return `
  ${pageHead('انتصابات', 'صدور و پیگیری ابلاغ‌های سازمانی: انتصاب، جابه‌جایی و پایان تصدی',
    `<button class="btn btn-primary" onclick="toast('فرم پیشنهاد انتصاب باز شد (نمونه)')">➕ ثبت پیشنهاد انتصاب</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'📜', tint:'#EFF6FF', accent:'#2563EB', val:fa(2), label:'ابلاغ صادرشده در مهر ۱۴۰۴'})}
    ${statCard({ico:'⏳', tint:'#FEF3C7', accent:'#D97706', val:fa(2), label:'در انتظار تأیید مدیر منابع انسانی'})}
    ${statCard({ico:'🔁', tint:'#EDE9FE', accent:'#7C3AED', val:fa(1), label:'جابه‌جایی داخلی'})}
    ${statCard({ico:'🏁', tint:'#FEE2E2', accent:'#DC2626', val:fa(1), label:'پایان تصدی ثبت‌شده'})}
  </div>
  ${tbl(['شماره ابلاغ','کارمند','پست','واحد','نوع ابلاغ','تاریخ','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td class="num">${r.id}</td><td>${empCell(r.emp)}</td><td>${r.post}</td><td>${r.unit}</td>
      <td>${r.type}</td><td class="num">${r.date}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('مشاهدهٔ متن ابلاغ (نمونه)','info')">👁️</button></td></tr>`))}`;
}

function pagePromotions() {
  const rows = [
    {emp:'EMP-008', from:'کارشناسی', to:'کارشناسی ارشد', score:92, years:6, st:'committee'},
    {emp:'EMP-006', from:'کارشناسی', to:'کارشناسی ارشد', score:88, years:5, st:'approved'},
    {emp:'EMP-011', from:'کاردانی', to:'کارشناسی', score:79, years:4, st:'committee'},
    {emp:'EMP-012', from:'کارشناسی', to:'کارشناسی ارشد', score:74, years:3, st:'pending'},
  ];
  const chip = s => ({committee:pChip('c-wait','در کمیتهٔ ترفیعات'), approved:pChip('c-ok','تأیید و اجرا شد'), pending:pChip('c-prob','تکمیل پرونده')})[s] || s;
  return `
  ${pageHead('ترفیعات', 'بررسی ترفیع و ارتقای گروه شغلی بر پایهٔ امتیاز ارزیابی، سابقه و ظرفیت تشکیلات',
    `<button class="btn btn-primary" onclick="toast('فرم ثبت درخواست ترفیع باز شد (نمونه)')">➕ ثبت درخواست ترفیع</button>
     <button class="btn btn-ghost" onclick="toast('محاسبهٔ امتیازها اجرا شد (نمونه)','info')">🧮 محاسبهٔ امتیاز</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'📈', tint:'#EFF6FF', accent:'#2563EB', val:fa(4), label:'پروندهٔ ترفیع در جریان'})}
    ${statCard({ico:'✅', tint:'#DCFCE7', accent:'#16A34A', val:fa(1), label:'ترفیع اجراشده در نیم‌سال اخیر'})}
    ${statCard({ico:'🪑', tint:'#FEF3C7', accent:'#D97706', val:fa(7), label:'ظرفیت تشکیلاتی آزاد در گروه ارشد'})}
    ${statCard({ico:'⭐', tint:'#EDE9FE', accent:'#7C3AED', val:'۸۳', label:'میانگین امتیاز ارزیابی پرونده‌ها'})}
  </div>
  ${tbl(['کارمند','گروه فعلی','گروه پیشنهادی','امتیاز ارزیابی','سابقه (سال)','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td>${empCell(r.emp)}</td><td>${r.from}</td><td><b>${r.to}</b></td>
      <td class="num">${fa(r.score)}</td><td class="num">${fa(r.years)}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('بررسی پرونده در کمیتهٔ ترفیعات (نمونه)','info')">👁️</button></td></tr>`))}
  <div class="card card-pad">
    <div class="card-title">مبنای تصمیم کمیتهٔ ترفیعات</div>
    <div class="muted" style="font-size:.82rem;line-height:1.9">
      امتیاز نهایی = ۵۰٪ امتیاز ارزیابی عملکرد دورهٔ آخر + ۳۰٪ سابقهٔ خدمت در گروه فعلی + ۲۰٪ امتیاز آموزش و گواهی‌نامه‌ها.
      اجرای ترفیع فقط در صورت وجود ظرفیت خالی در گروه شغلی بالاتر و تأیید کمیتهٔ ترفیعات امکان‌پذیر است.
    </div>
  </div>`;
}

function pageCareerPath() {
  const steps = [
    {title:'کارشناس', years:'۲–۴ سال', done:true},
    {title:'کارشناس ارشد', years:'۴–۷ سال', done:true, current:true},
    {title:'سرپرست گروه', years:'۷–۱۰ سال', done:false},
    {title:'مدیر واحد', years:'۱۰ سال به بالا', done:false},
  ];
  const succ = [
    {post:'مدیر منابع انسانی', holder:'سارا محمدی', backup:'مریم نوری', ready:'آماده در ۱ سال'},
    {post:'کارشناس ارشد برنامه‌ریزی', holder:'مریم نوری', backup:'نگار حسینی', ready:'آماده اکنون'},
    {post:'مسئول طب کار', holder:'دکتر رضوانی', backup:'—', ready:'بدون جانشین — دارای ریسک'},
  ];
  return `
  ${pageHead('کارراههٔ شغلی', 'مسیر رشد شغلی هر کارمند، اهداف شغلی و جانشین‌پروری پست‌های کلیدی',
    `<button class="btn btn-ghost" onclick="toast('ثبت هدف شغلی (نمونه)','info')">🎯 ثبت هدف شغلی</button>`)}
  <div class="grid-2 mb">
    <div class="card card-pad">
      <div class="card-title">مسیر شغلی پیشنهادی — ${esc((EMPS.find(e=>e.id==='EMP-008')||{}).name||'')}</div>
      ${steps.map((s,i)=>`
        <div class="kpi-row" style="align-items:center">
          <span>${s.done?'✅':'⚪'} <b>${s.title}</b> <span class="muted" style="font-size:.72rem">${s.years}</span></span>
          <span>${s.current?pChip('c-ok','موقعیت فعلی'):''}</span>
        </div>
        ${i<steps.length-1?'<div class="progress" style="margin:2px 0 10px"><span style="width:'+(s.done?'100%':'0%')+';background:#2563EB"></span></div>':''}`).join('')}
    </div>
    <div class="card card-pad">
      <div class="card-title">جانشین‌پروری پست‌های کلیدی</div>
      ${succ.map(s=>`
        <div class="kpi-row"><span><b>${s.post}</b><br><span class="muted" style="font-size:.72rem">تصدی: ${s.holder}</span></span>
        <span style="text-align:left">${s.backup}<br><span class="muted" style="font-size:.72rem">${s.ready}</span></span></div>`).join('')}
    </div>
  </div>
  ${tbl(['کارمند','گروه شغلی','هدف شغلی ثبت‌شده','آمادگی','اقدام بعدی'],
    [['EMP-008','کارشناسی ارشد','سرپرست برنامه‌ریزی نیروی انسانی','۸۵٪','دورهٔ مدیریت تیم آموزشی'],
     ['EMP-003','کارشناسی','کارشناس ارشد فروش','۶۰٪','تکمیل دورهٔ مذاکره پیشرفته'],
     ['EMP-006','کارشناسی','کارشناس ارشد آموزش','۷۰٪','نیازسنجی آموزشی و شرکت در دورهٔ طراحی آموزش']
    ].map(([e,g,t,r,a])=>`<tr><td>${empCell(e)}</td><td>${g}</td><td>${t}</td>
      <td class="num">${r}</td><td>${a}</td></tr>`), {foot:false})}`;
}

/* ---------------------------------------------------------------------
   واحد روابط کار و مددکاری اجتماعی — انضباط، مددکاری، درمان
   --------------------------------------------------------------------- */

function pageDisciplineCases() {
  const rows = [
    {id:'DSP-1404-014', emp:'EMP-011', kind:'غیبت غیرموجه', date:'۱۴۰۴/۰۶/۲۸', stage:'در انتظار دفاعیات', ruling:'—'},
    {id:'DSP-1404-013', emp:'EMP-009', kind:'تأخیر مکرر', date:'۱۴۰۴/۰۶/۲۰', stage:'جلسهٔ کمیته', ruling:'—'},
    {id:'DSP-1404-012', emp:'EMP-010', kind:'عدم رعایت دستورالعمل ایمنی', date:'۱۴۰۴/۰۶/۱۱', stage:'بسته‌شده', ruling:'تذکر کتبی'},
    {id:'DSP-1404-011', emp:'EMP-007', kind:'استفادهٔ نامناسب از اموال سازمان', date:'۱۴۰۴/۰۵/۳۰', stage:'بسته‌شده', ruling:'کسر از حقوق'},
  ];
  const chip = s => s==='بسته‌شده' ? pChip('c-ok', s) : pChip('c-wait', s);
  return `
  ${pageHead('پرونده‌های انضباطی', 'ثبت تخلف، تشکیل پرونده، دفاعیات کارمند و رأی کمیتهٔ انضباط کار',
    `<button class="btn btn-primary" onclick="toast('فرم ثبت گزارش تخلف باز شد (نمونه)')">➕ ثبت گزارش تخلف</button>`)}
  <div class="card card-pad" style="border-inline-start:4px solid var(--amber);margin-bottom:14px">
    <b>دسترسی محرمانه:</b>
    <span class="muted" style="font-size:.82rem">
      پرونده‌های انضباطی تنها برای اعضای کمیته و مدیر منابع انسانی قابل مشاهده است و دسترسی کارمند و مدیر واحد به آن بسته است؛
      هر مشاهده در لاگ حسابرسی ثبت می‌شود.
    </span>
  </div>
  <div class="grid-stats">
    ${statCard({ico:'📁', tint:'#EFF6FF', accent:'#2563EB', val:fa(2), label:'پروندهٔ باز'})}
    ${statCard({ico:'🗓️', tint:'#FEF3C7', accent:'#D97706', val:fa(1), label:'جلسهٔ پیش‌رو'})}
    ${statCard({ico:'⚖️', tint:'#EDE9FE', accent:'#7C3AED', val:fa(2), label:'رأی صادرشده در سال جاری'})}
    ${statCard({ico:'📉', tint:'#DCFCE7', accent:'#16A34A', val:'۳۵٪', label:'کاهش پرونده نسبت به سال گذشته'})}
  </div>
  ${tbl(['شماره پرونده','کارمند','نوع تخلف','تاریخ گزارش','مرحله','رأی کمیته','عملیات'],
    rows.map(r=>`<tr>
      <td class="num">${r.id}</td><td>${empCell(r.emp)}</td><td>${r.kind}</td>
      <td class="num">${r.date}</td><td>${chip(r.stage)}</td><td>${r.ruling}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('مشاهدهٔ پرونده (نمونه) — رخداد در لاگ ثبت شد','info')">👁️</button></td></tr>`))}`;
}

function pageDisciplineMeetings() {
  const rows = [
    {id:'MTG-1404-09', date:'۱۴۰۴/۰۷/۰۹', cases:'DSP-1404-014', members:'مدیر منابع انسانی، کارشناس انضباط، نمایندهٔ کارکنان', minutes:'ثبت نشده', st:'scheduled'},
    {id:'MTG-1404-08', date:'۱۴۰۴/۰۶/۲۲', cases:'DSP-1404-013', members:'مدیر منابع انسانی، کارشناس انضباط، نمایندهٔ کارکنان', minutes:'ثبت شده', st:'held'},
    {id:'MTG-1404-07', date:'۱۴۰۴/۰۶/۱۲', cases:'DSP-1404-012', members:'مدیر منابع انسانی، کارشناس انضباط', minutes:'ثبت شده', st:'held'},
  ];
  const chip = s => ({scheduled:pChip('c-wait','برنامه‌ریزی‌شده'), held:pChip('c-ok','برگزار شده')})[s] || s;
  return `
  ${pageHead('جلسات کمیتهٔ انضباط', 'زمان‌بندی جلسه، دعوت اعضا و کارمند، صورت‌جلسه و پیگیری مصوبات',
    `<button class="btn btn-primary" onclick="toast('فرم زمان‌بندی جلسه باز شد (نمونه)')">➕ زمان‌بندی جلسه</button>
     <button class="btn btn-ghost" onclick="toast('ارسال دعوت‌نامه‌ها شبیه‌سازی شد','info')">✉️ ارسال دعوت‌نامه</button>`)}
  ${tbl(['شماره جلسه','تاریخ','پرونده‌های دستور','اعضای مدعو','صورت‌جلسه','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td class="num">${r.id}</td><td class="num">${r.date}</td><td class="num">${r.cases}</td>
      <td class="txth" style="white-space:normal;max-width:300px">${r.members}</td>
      <td>${r.minutes==='ثبت شده'?pChip('c-ok','ثبت شده'):pChip('c-wait','ثبت نشده')}</td>
      <td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('مشاهدهٔ صورت‌جلسه (نمونه)','info')">👁️</button></td></tr>`))}
  <div class="card card-pad">
    <div class="card-title">مصوبات جلسهٔ گذشته</div>
    <ul class="list-tight">
      <li><span>⚖️</span><div>رأی پروندهٔ DSP-1404-012: تذکر کتبی و درج در پرونده — <b>اجرا شد</b></div></li>
      <li><span>📚</span><div>الزام شرکت ۳ نفر از واحد عملیات در دورهٔ ایمنی کار — <b>ارجاع به واحد هماهنگی آموزش</b></div></li>
      <li><span>🤝</span><div>ارجاع پروندهٔ DSP-1404-013 به مددکاری اجتماعی برای بررسی وضعیت کارمند — <b>در جریان</b></div></li>
    </ul>
  </div>`;
}

function pageSocialWorkCases() {
  const rows = [
    {id:'SW-1404-021', emp:'EMP-009', subject:'بررسی علت تأخیرهای مکرر', worker:'مریم نوری', last:'جلسهٔ مشاوره — ۱۴۰۴/۰۶/۲۴', st:'open'},
    {id:'SW-1404-020', emp:'EMP-011', subject:'حمایت در دورهٔ بیماری', worker:'مریم نوری', last:'ارجاع به درمانگاه سازمان', st:'open'},
    {id:'SW-1404-019', emp:'EMP-007', subject:'مشکل معیشتی — کمک موردی', worker:'زهرا کاظمی', last:'تأیید کمک — ۱۴۰۴/۰۶/۰۲', st:'closed'},
  ];
  const chip = s => s==='open' ? pChip('c-wait','باز — در پیگیری') : pChip('c-ok','بسته‌شده');
  return `
  ${pageHead('پرونده‌های مددکاری', 'شناسایی مشکل، اقدام‌های حمایتی، جلسات مشاوره و نتیجهٔ پیگیری',
    `<button class="btn btn-primary" onclick="toast('فرم گشودن پروندهٔ مددکاری باز شد (نمونه)')">➕ گشودن پرونده</button>`)}
  <div class="card card-pad" style="border-inline-start:4px solid var(--amber);margin-bottom:14px">
    <b>رعایت محرمانگی:</b>
    <span class="muted" style="font-size:.82rem">
      محتوای پرونده‌های مددکاری تنها برای مددکار و مدیر منابع انسانی قابل مشاهده است. کارمند صرفاً وضعیت درخواست خود را می‌بیند، نه شرح پرونده را.
    </span>
  </div>
  <div class="grid-stats">
    ${statCard({ico:'🤝', tint:'#EFF6FF', accent:'#2563EB', val:fa(2), label:'پروندهٔ باز'})}
    ${statCard({ico:'🗣️', tint:'#EDE9FE', accent:'#7C3AED', val:fa(14), label:'جلسهٔ مشاوره در فصل جاری'})}
    ${statCard({ico:'🏥', tint:'#FEF3C7', accent:'#D97706', val:fa(3), label:'ارجاع به خدمات درمانی'})}
    ${statCard({ico:'✅', tint:'#DCFCE7', accent:'#16A34A', val:fa(9), label:'پروندهٔ بسته‌شده در سال جاری'})}
  </div>
  ${tbl(['شماره پرونده','کارمند','موضوع','مددکار','آخرین اقدام','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td class="num">${r.id}</td><td>${empCell(r.emp)}</td><td>${r.subject}</td><td>${r.worker}</td>
      <td>${r.last}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('مشاهدهٔ پروندهٔ مددکاری (نمونه)','info')">👁️</button></td></tr>`))}`;
}

function pageAidRequests() {
  const rows = [
    {id:'AID-1404-032', emp:'EMP-011', kind:'کمک درمانی', amount:45000000, date:'۱۴۰۴/۰۷/۰۶', st:'approve'},
    {id:'AID-1404-031', emp:'EMP-009', kind:'وام ضروری', amount:120000000, date:'۱۴۰۴/۰۷/۰۳', st:'review'},
    {id:'AID-1404-030', emp:'EMP-007', kind:'کمک معیشتی', amount:25000000, date:'۱۴۰۴/۰۶/۲۵', st:'paid'},
    {id:'AID-1404-029', emp:'EMP-003', kind:'کمک ازدواج', amount:30000000, date:'۱۴۰۴/۰۶/۱۸', st:'paid'},
  ];
  const chip = s => ({approve:pChip('c-wait','در انتظار تأیید'), review:pChip('c-prob','بررسی مستندات'), paid:pChip('c-ok','پرداخت شد')})[s] || s;
  const isEmp = session.role === 'employee';
  const mine = rows.filter(r=>r.emp===(session.emp||'EMP-003'));
  const show = isEmp ? mine : rows;
  return `
  ${pageHead('کمک‌های موردی', 'درخواست کمک درمانی، معیشتی و وام ضروری کارکنان و پیگیری پرداخت',
    `<button class="btn btn-primary" onclick="toast('فرم ثبت درخواست کمک باز شد (نمونه)')">➕ ثبت درخواست</button>`)}
  ${isEmp?`<div class="card card-pad" style="margin-bottom:14px"><b>نمای کارمند:</b>
    <span class="muted" style="font-size:.82rem">شما فقط درخواست‌های خودتان را می‌بینید.</span></div>`:''}
  <div class="grid-stats">
    ${statCard({ico:'📮', tint:'#EFF6FF', accent:'#2563EB', val:fa(2), label:'درخواست در جریان'})}
    ${statCard({ico:'💳', tint:'#DCFCE7', accent:'#16A34A', val:'۳۳۰ م', label:'جمع پرداختی سال جاری (ریال)'})}
    ${statCard({ico:'⏱️', tint:'#FEF3C7', accent:'#D97706', val:'۲ روز', label:'میانگین زمان بررسی'})}
    ${statCard({ico:'🧾', tint:'#EDE9FE', accent:'#7C3AED', val:fa(4), label:'درخواست ثبت‌شده در مهر'})}
  </div>
  ${tbl(['شماره درخواست','کارمند','نوع کمک','مبلغ درخواستی','تاریخ','وضعیت','عملیات'],
    show.map(r=>`<tr>
      <td class="num">${r.id}</td><td>${empCell(r.emp)}</td><td>${r.kind}</td>
      <td class="num">${pMoney(r.amount)}</td><td class="num">${r.date}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('جزئیات درخواست (نمونه)','info')">👁️</button></td></tr>`))}`;
}

function pageTreatmentCases() {
  const rows = [
    {id:'TRT-1404-044', emp:'EMP-011', center:'بیمارستان امام خمینی', doc:38000000, org:30400000, st:'refund'},
    {id:'TRT-1404-043', emp:'EMP-010', center:'آزمایشگاه پارس', doc:6200000, org:4960000, st:'refund'},
    {id:'TRT-1404-042', emp:'EMP-006', center:'کلینیک تخصصی مهر', doc:14500000, org:11600000, st:'paid'},
    {id:'TRT-1404-041', emp:'EMP-009', center:'بیمارستان میلاد', doc:92000000, org:73600000, st:'paid'},
  ];
  const chip = s => s==='paid' ? pChip('c-ok','بازپرداخت شد') : pChip('c-wait','در انتظار بازپرداخت');
  return `
  ${pageHead('پروندهٔ درمان', 'ثبت اسناد بیمارستانی، محاسبهٔ سهم سازمان و بیمه و پیگیری بازپرداخت',
    `<button class="btn btn-primary" onclick="toast('فرم گشودن پروندهٔ درمان باز شد (نمونه)')">➕ گشودن پروندهٔ درمان</button>
     <button class="btn btn-ghost" onclick="toast('خروجی اسناد درمان شبیه‌سازی شد','info')">⬇️ خروجی</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'🏥', tint:'#EFF6FF', accent:'#2563EB', val:fa(2), label:'پروندهٔ در انتظار بازپرداخت'})}
    ${statCard({ico:'💰', tint:'#FEF3C7', accent:'#D97706', val:'۱۱۶ م', label:'جمع مبلغ اسناد فصل جاری (ریال)'})}
    ${statCard({ico:'🛡️', tint:'#DCFCE7', accent:'#16A34A', val:'۸۰٪', label:'سهم تعهد سازمان'})}
    ${statCard({ico:'⏱️', tint:'#EDE9FE', accent:'#7C3AED', val:'۵ روز', label:'میانگین زمان بازپرداخت'})}
  </div>
  ${tbl(['شماره پرونده','کارمند','مرکز درمانی','مبلغ سند','سهم سازمان','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td class="num">${r.id}</td><td>${empCell(r.emp)}</td><td>${r.center}</td>
      <td class="num">${pMoney(r.doc)}</td><td class="num">${pMoney(r.org)}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('مشاهدهٔ اسناد و محاسبهٔ سهم (نمونه)','info')">👁️</button></td></tr>`))}`;
}

function pagePeriodicExams() {
  const rows = [
    {emp:'EMP-012', unit:'تولید و عملیات', kind:'طب کار — ریسک بالا', due:'۱۴۰۴/۰۷/۱۵', done:'—', result:'—', st:'soon'},
    {emp:'EMP-011', unit:'تولید و عملیات', kind:'طب کار — ریسک متوسط', due:'۱۴۰۴/۰۷/۰۲', done:'—', result:'—', st:'overdue'},
    {emp:'EMP-003', unit:'فروش و بازاریابی', kind:'معاینهٔ عمومی سالانه', due:'۱۴۰۴/۰۶/۲۰', done:'۱۴۰۴/۰۶/۲۱', result:'سالم', st:'ok'},
    {emp:'EMP-006', unit:'تحقیق و توسعه', kind:'شنوایی‌سنجی', due:'۱۴۰۴/۰۶/۱۰', done:'۱۴۰۴/۰۶/۱۰', result:'سالم', st:'ok'},
    {emp:'EMP-010', unit:'مالی و حسابداری', kind:'معاینهٔ عمومی سالانه', due:'۱۴۰۴/۰۶/۰۵', done:'۱۴۰۴/۰۶/۰۸', result:'نیازمند پیگیری', st:'follow'},
  ];
  const chip = s => ({soon:pChip('c-wait','سررسید نزدیک'), overdue:pChip('c-rej','عقب‌افتاده'),
    ok:pChip('c-ok','انجام شده'), follow:pChip('c-prob','نیازمند پیگیری')})[s] || s;
  return `
  ${pageHead('معاینات دوره‌ای', 'برنامه‌ریزی طب کار و معاینات دوره‌ای بر پایهٔ ریسک شغلی و پیگیری عقب‌افتادگی‌ها',
    `<button class="btn btn-primary" onclick="toast('زمان‌بندی معاینه (نمونه)','info')">➕ تعیین نوبت معاینه</button>
     <button class="btn btn-ghost" onclick="toast('یادآور برای کارکنان عقب‌افتاده ارسال شد','info')">🔔 یادآوری</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'🩺', tint:'#EFF6FF', accent:'#2563EB', val:fa(41), label:'معاینهٔ ثبت‌شده در سال جاری'})}
    ${statCard({ico:'⚠️', tint:'#FEE2E2', accent:'#DC2626', val:fa(2), label:'معاینهٔ عقب‌افتاده'})}
    ${statCard({ico:'🗓️', tint:'#FEF3C7', accent:'#D97706', val:fa(9), label:'سررسید ۳۰ روز آینده'})}
    ${statCard({ico:'🧪', tint:'#EDE9FE', accent:'#7C3AED', val:fa(3), label:'نتیجهٔ نیازمند پیگیری'})}
  </div>
  ${tbl(['کارمند','واحد','نوع معاینه','تاریخ سررسید','تاریخ انجام','نتیجه','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td>${empCell(r.emp)}</td><td>${r.unit}</td><td>${r.kind}</td>
      <td class="num">${r.due}</td><td class="num">${r.done}</td><td>${r.result}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('ثبت نتیجهٔ معاینه (نمونه)','info')">✏️</button></td></tr>`))}`;
}

function pageMedicalServices() {
  const services = [
    ['درمانگاه سازمان', 'پزشک عمومی، تزریقات و پانسمان', 'همهٔ کارکنان و افراد تحت تکفل'],
    ['آزمایشگاه', 'آزمایش‌های روتین با ۸۰٪ تعهد سازمان', 'همهٔ کارکنان'],
    ['مشاورهٔ پزشک متخصص', 'داخلی، ارتوپدی، زنان و روان‌پزشکی', 'کارکنان با معرفی‌نامه'],
    ['بیمهٔ تکمیلی', 'پوشش پاراکلینیک و بستری', 'کارکنان رسمی و قراردادی'],
    ['دندان‌پزشکی', 'معاینه و خدمات پایه — سقف سالانه', 'همهٔ کارکنان'],
    ['بانک دارو', 'داروهای مزمن با نسخهٔ پزشک سازمان', 'کارکنان و افراد تحت تکفل'],
  ];
  return `
  ${pageHead('خدمات درمانی و بیمه', 'معرفی خدمات درمانی سازمان و وضعیت پوشش بیمهٔ درمان و تکمیلی',
    `<button class="btn btn-primary" onclick="toast('معرفی‌نامهٔ درمان صادر شد (نمونه)')">📄 صدور معرفی‌نامه</button>`)}
  <div class="grid-2eq mb">
    <div class="card card-pad">
      <div class="card-title">فهرست خدمات درمانی</div>
      ${services.map(([n,d,w])=>`
        <div class="kpi-row"><span><b>${n}</b><br><span class="muted" style="font-size:.72rem">${d}</span></span>
        <span class="muted" style="font-size:.72rem;max-width:150px;text-align:left">${w}</span></div>`).join('')}
    </div>
    <div class="card card-pad">
      <div class="card-title">وضعیت پوشش بیمهٔ کارکنان</div>
      ${[['بیمهٔ درمان پایه (تأمین اجتماعی)','۱۰۰٪ کارکنان تحت پوشش','c-ok'],
         ['بیمهٔ تکمیلی درمان','۶۲ نفر از ۶۶ نفر','c-ok'],
         ['بیمهٔ عمر و حادثه','۶۶ نفر','c-ok'],
         ['معرفی‌نامهٔ فعال کارکنان','۵ معرفی‌نامهٔ در جریان','c-wait']]
        .map(([t,d,c])=>`<div class="kpi-row"><span>${t}</span><span>${pChip(c,d)}</span></div>`).join('')}
      <div class="muted" style="font-size:.76rem;margin-top:8px;line-height:1.8">
        ثبت فرد تحت تکفل و تغییر سطح پوشش از همین صفحه انجام می‌شود؛ تغییری که مستقیماً روی اجزای حقوقی مؤثر است
        گزارش آن به واحد امور کارکنان ارسال می‌شود.
      </div>
    </div>
  </div>
  <div class="muted" style="font-size:.78rem">
    اطلاعات پزشکی کارکنان طبقه‌بندی‌شده است: نتیجهٔ معاینه و پروندهٔ درمان برای مدیر واحد قابل مشاهده نیست.
  </div>`;
}

/* ---------------------------------------------------------------------
   واحد هماهنگی آموزش — از برنامه‌ریزی تا اجرا
   --------------------------------------------------------------------- */

function pageTrainingPlan() {
  const rows = [
    {title:'مدیریت تیم و سرپرستی', unit:'تولید و عملیات', count:12, span:'مهر — آبان ۱۴۰۴', budget:180000000, st:'approved'},
    {title:'ایمنی و بهداشت کار', unit:'تولید و عملیات', count:40, span:'مهر — آذر ۱۴۰۴', budget:95000000, st:'running'},
    {title:'اکسل پیشرفته و تحلیل داده', unit:'همهٔ واحدها', count:25, span:'آبان — دی ۱۴۰۴', budget:120000000, st:'approved'},
    {title:'قانون کار و روابط کار', unit:'منابع انسانی', count:8, span:'آذر ۱۴۰۴', budget:60000000, st:'draft'},
    {title:'طراحی آموزش و نیازسنجی', unit:'منابع انسانی', count:5, span:'بهمن ۱۴۰۴', budget:70000000, st:'draft'},
  ];
  const chip = s => ({approved:pChip('c-ok','تصویب‌شده'), running:pChip('c-prob','در حال اجرا'), draft:pChip('c-wait','پیش‌نویس')})[s] || s;
  const total = rows.reduce((s,r)=>s+r.budget,0);
  return `
  ${pageHead('برنامه‌ریزی آموزشی', 'تدوین برنامهٔ آموزشی سالانه بر پایهٔ نیازسنجی، اولویت واحدها و بودجهٔ مصوب',
    `<button class="btn btn-primary" onclick="toast('فرم افزودن دوره به برنامه باز شد (نمونه)')">➕ افزودن دوره به برنامه</button>
     <button class="btn btn-ghost" onclick="toast('برنامه برای تصویب ارسال شد (نمونه)','info')">📤 ارسال برای تصویب</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'🗺️', tint:'#EFF6FF', accent:'#2563EB', val:fa(5), label:'ردیف برنامهٔ سال ۱۴۰۴'})}
    ${statCard({ico:'👥', tint:'#DCFCE7', accent:'#16A34A', val:fa(90), label:'نفر-دورهٔ پیش‌بینی‌شده'})}
    ${statCard({ico:'💵', tint:'#FEF3C7', accent:'#D97706', val:fa(Math.round(total/1000000))+' م', label:'بودجهٔ برآوردی (ریال)'})}
    ${statCard({ico:'✅', tint:'#EDE9FE', accent:'#7C3AED', val:fa(3), label:'ردیف تصویب‌شده'})}
  </div>
  ${tbl(['عنوان دوره','واحد هدف','تعداد نفرات','بازهٔ اجرا','بودجهٔ برآوردی','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td><b>${r.title}</b></td><td>${r.unit}</td><td class="num">${fa(r.count)}</td>
      <td class="num">${r.span}</td><td class="num">${pMoney(r.budget)}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('ویرایش ردیف برنامه (نمونه)','info')">✏️</button></td></tr>`))}`;
}

function pageTrainingExecution() {
  const rows = [
    {course:'ایمنی و بهداشت کار', teacher:'مهندس رحیمی', place:'سالن آموزش — طبقهٔ ۲', date:'۱۴۰۴/۰۷/۰۹', seats:40, reg:37, attend:null, st:'planned'},
    {course:'مدیریت تیم و سرپرستی', teacher:'دکتر شریفی', place:'سایت آموزش آنلاین', date:'۱۴۰۴/۰۷/۰۵', seats:12, reg:12, attend:11, st:'running'},
    {course:'اکسل پیشرفته', teacher:'مهندس یوسفی', place:'سالن آموزش — طبقهٔ ۲', date:'۱۴۰۴/۰۶/۲۵', seats:25, reg:24, attend:22, st:'held'},
    {course:'ایمنی کار — دورهٔ مقدماتی', teacher:'مهندس رحیمی', place:'سالن تولید', date:'۱۴۰۴/۰۶/۱۲', seats:30, reg:28, attend:26, st:'held'},
  ];
  const chip = s => ({planned:pChip('c-wait','در انتظار برگزاری'), running:pChip('c-prob','در حال برگزاری'), held:pChip('c-ok','برگزار شد')})[s] || s;
  return `
  ${pageHead('اجرا و هماهنگی دوره‌ها', 'زمان‌بندی جلسات، تعیین مدرس و محل، دعوت‌نامه و ثبت حضور و غیاب',
    `<button class="btn btn-primary" onclick="toast('زمان‌بندی جلسهٔ جدید (نمونه)','info')">➕ زمان‌بندی جلسه</button>
     <button class="btn btn-ghost" onclick="toast('دعوت‌نامه‌ها برای ۳۷ نفر ارسال شد (نمونه)')">✉️ ارسال دعوت‌نامه</button>`)}
  <div class="grid-stats">
    ${statCard({ico:'🎓', tint:'#EFF6FF', accent:'#2563EB', val:fa(1), label:'دورهٔ در حال برگزاری'})}
    ${statCard({ico:'🗓️', tint:'#FEF3C7', accent:'#D97706', val:fa(1), label:'دورهٔ در انتظار برگزاری'})}
    ${statCard({ico:'✅', tint:'#DCFCE7', accent:'#16A34A', val:fa(2), label:'دورهٔ برگزارشده در مهر'})}
    ${statCard({ico:'📉', tint:'#EDE9FE', accent:'#7C3AED', val:'۸٪', label:'نرخ غیبت در دوره‌های گذشته'})}
  </div>
  ${tbl(['عنوان دوره','مدرس','محل برگزاری','تاریخ','ظرفیت','ثبت‌نام','حضور','وضعیت','عملیات'],
    rows.map(r=>`<tr>
      <td>${r.course}</td><td>${r.teacher}</td><td>${r.place}</td><td class="num">${r.date}</td>
      <td class="num">${fa(r.seats)}</td><td class="num">${fa(r.reg)}</td>
      <td class="num">${r.attend===null?'—':fa(r.attend)}</td><td>${chip(r.st)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="toast('ثبت حضور و غیاب دوره (نمونه)','info')">📝</button></td></tr>`))}`;
}
