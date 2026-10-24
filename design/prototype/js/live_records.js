/* =====================================================================
   صفحه‌های زندهٔ عمومی رکوردهای گردش‌کاری (موتور recordkit)
   =====================================================================

   این فایل هیچ قاعدهٔ کسب‌وکاری ندارد: هر ماژول اداری (انتصابات، بایگانی،
   انضباط کار، حقوق، آموزش، رفاه…) شناسهٔ خودش را از سرور می‌خواند و این لایه
   از روی همان شناسه، جدول و فرم و دکمهٔ اقدام‌ها را می‌سازد. افزودن ماژول تازه
   در سرور، خودکار در این رابط دیده می‌شود؛ نیازی به نوشتن صفحهٔ دستی نیست.

   نگاشت زیر می‌گوید هر «صفحهٔ منو» به کدام «نوع رکورد» وصل شود. اگر صفحه‌ای
   اینجا نباشد، همان صفحهٔ نمایشی بخش ۵ نشان داده می‌شود.
   ===================================================================== */

const RK_PAGES = {
  // بایگانی و شناسایی کارکنان
  personnelFiles: {kind: 'personnelFiles'},
  employeeIdentity: {kind: 'employeeIdentity'},
  archiveDocuments: {kind: 'archiveDocuments'},
  // حضور و غیاب، مرخصی و مأموریت
  attendance: {kind: 'attendanceRecords'},
  leaveRequests: {kind: 'leaveRequests'},
  leaveApprove: {kind: 'leaveRequests', status: 'pending'},
  overtime: {kind: 'overtime'},
  missions: {kind: 'missions'},
  // انتصابات، ترفیعات و کارراههٔ شغلی
  appointments: {kind: 'appointments'},
  promotions: {kind: 'promotions'},
  careerPath: {kind: 'careerPath'},
  // کمیتهٔ انضباط کار
  disciplineCases: {kind: 'disciplineCases'},
  disciplineMeetings: {kind: 'disciplineMeetings'},
  // شکایات و ایثارگران
  complaints: {kind: 'complaints'},
  veteransAffairs: {kind: 'veteranRecords'},
  // مطالعات منابع انسانی
  studies: {kind: 'studies'},
  // تب نظرسنجی: رکوردها برای ثبت و گردش‌کار، و پنل تحلیلِ برپایهٔ /surveys.
  surveys: {kind: 'surveys', panel: 'surveys'},
  // تب شاخص‌های سرمایهٔ انسانی: همان موتور رکورد، ولی با سنجش «در محدودهٔ هدف»
  // که سرور از /indices می‌دهد. بدون این سطر، صفحه به قالب نمایشی ثابت برمی‌گشت.
  hrIndices: {kind: 'hrIndices', panel: 'indices'},
  suggestions: {kind: 'suggestions'},
  // طبقه‌بندی مشاغل
  jobCategories: {kind: 'jobCategories'},
  jobEvaluation: {kind: 'jobEvaluations'},
  jobDescriptions: {kind: 'jobDescriptions'},
  // وام و بازنشستگی
  loans: {kind: 'loans', panel: 'loans'},
  loanInstallments: {kind: 'loanInstallments'},
  retirementFiles: {kind: 'retirementFiles'},
  // درمان
  treatmentCases: {kind: 'treatmentCases'},
  periodicExams: {kind: 'periodicExams'},
  medicalServices: {kind: 'medicalServices'},
  supplementaryInsurance: {kind: 'supplementaryInsurance'},
  // امور ورزش
  sportsPrograms: {kind: 'sportsPrograms'},
  sportsEvents: {kind: 'sportsEvents'},
  // ساختار سازمانی — شناسنامهٔ پست‌ها (نمودار درختی صفحهٔ جدا دارد)
  positions: {kind: 'positions'},
  // برون‌سپاری
  contractors: {kind: 'contractors'},
  thirdPartyPayroll: {kind: 'outsourcePayRuns'},
  hardshipFiles: {kind: 'hardshipFiles'},
  // حقوق و دستمزد
  payrollRun: {kind: 'payrollRuns'},
  payItems: {kind: 'payItems'},
  payParams: {kind: 'payParams'},
  karana: {kind: 'karana'},
  // مدیریت عملکرد
  perfCycles: {kind: 'perfCycles'},
  perfForms: {kind: 'perfForms'},
  perfResults: {kind: 'perfResults'},
  // تأمین و جذب نیرو
  hiringReqs: {kind: 'hiringRequests'},
  jobs: {kind: 'jobPostings'},
  candidates: {kind: 'candidates'},
  // گزارش‌ها
  reports: {kind: 'reportRuns'},
  // mgmtReports صفحهٔ اختصاصی «داشبورد مدیریتی هلدینگ» در live.js است (نه صفحهٔ رکوردی).
  // تنظیمات (کاربران/نقش‌ها/لاگ حسابرسي صفحهٔ اختصاصی دارند)
  sysParams: {kind: 'sysParams'},
  orgSettings: {kind: 'orgSettings'},
  // قالب‌های متن اسناد (قرارداد، فیش حقوق، حکم، گواهی)
  docTemplates: {kind: 'docTemplates'},
  // مددکاری اجتماعی
  socialWorkCases: {kind: 'socialWorkCases'},
  aidRequests: {kind: 'aidRequests'},
  // جابه‌جایی و معرفی
  transfers: {kind: 'transfers'},
  introductions: {kind: 'introductions'},
  // آمار
  statsOverview: {kind: 'statSnapshots'},
  // آموزش
  trainingPlan: {kind: 'trainingNeeds'},
  trainingNeeds: {kind: 'trainingNeeds'},
  trainingExecution: {kind: 'courses'},
  training: {kind: 'courses'},
  trainingHist: {kind: 'enrollments'},
  // رفاه و اسکان
  welfare: {kind: 'facilities'},
  welfareReq: {kind: 'welfareRequests'},
  schoolService: {kind: 'welfareRequests'},
  welfareTours: {kind: 'facilities'},
  accommodation: {kind: 'facilities'},
  // کارت‌های رفاهی و شارژ (پتروکارت، نفت‌کارت)
  petrokart: {kind: 'petrokartCharges', panel: 'cards'},
  cardAllocations: {kind: 'cardBeneficiaries'},
  naftmart: {kind: 'naftmartCharges', panel: 'cards'},
  cardRates: {kind: 'cardRates'},
  occasions: {kind: 'occasions'},
  chargeFiles: {kind: 'chargeFiles', panel: 'cards'},
  // پاداش تحصیلی فرزندان ممتاز (سر فصل مستقل از پتروکارت و نفت‌کارت)
  outstandingStudents: {kind: 'outstandingStudents'},
  educationBonusRates: {kind: 'bonusRates'},
  educationBonusFiles: {kind: 'bonusFiles', panel: 'bonus'},
  // پاداش‌ها (سر فصل مستقل): داشبورد مدیریتی، فرم تخصیص و گزارش پرداخت
  bonusDashboard: {kind: 'bonusPayments', panel: 'bonusDash'},
  bonusPayments: {kind: 'bonusPayments', panel: 'bonusPay'},
  bonusReports: {kind: 'bonusPayments', panel: 'bonusRep'},
};

/** وضعیت بارگذاری‌شدهٔ هر صفحه: {mod, kind} — برای فرم و اقدام‌ها. */
const RK_STATE = {};

/** متن جستجوی هر صفحهٔ رکوردی: {کلید صفحه: عبارت}. سرور با پارامتر «query»
 *  شناسه/عنوان/کارمند/مقدار فیلدها را می‌گردد؛ پس فیلتر در سرور اجرا می‌شود، نه در UI. */
const RK_QUERY = {};

/**
 * عبارت جستجوی صفحه‌های اختصاصی (احکام، گواهی، قرارداد، تکفل، پرواز).
 *
 * چرا جدا از RK_QUERY: آن یکی برای صفحه‌های عمومی موتور recordkit است؛ این یکی برای
 * صفحه‌هایی که فهرست خود را از مسیر اختصاصی ماژول می‌خوانند. در هر دو حالت فیلتر در
 * سرور (پارامتر query) اعمال می‌شود و رابط فقط عبارت را نگه می‌دارد تا با بازخوانی
 * از دست نرود.
 */
const LIVE_QUERY = {};

/** نشانی فهرست یک صفحهٔ اختصاصی را با پارامتر جستجوی جاری کامل می‌کند. */
function liveQueryURL(page, base) {
  const term = (LIVE_QUERY[page] || '').trim();
  return term ? base + (base.includes('?') ? '&' : '?') + 'query=' + encodeURIComponent(term) : base;
}

/** کادر جستجوی سروری صفحه‌های اختصاصی — بدون قاعدهٔ کسب‌وکار، فقط فرستادن عبارت. */
function liveSearchBar(page, placeholder) {
  return `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:8px;align-items:center;flex-wrap:wrap">
      <input class="inp w-lg" id="liveSearch" value="${esc(LIVE_QUERY[page] || '')}"
        placeholder="${esc(placeholder || '🔍 جستجو…')}"
        onkeydown="if(event.key==='Enter'){liveSearchApply('${esc(page)}')}">
      <button class="btn btn-primary btn-sm" onclick="liveSearchApply('${esc(page)}')">🔍 جستجو</button>
      <button class="btn btn-ghost btn-sm" onclick="liveSearchClear('${esc(page)}')">✖ پاک‌کردن</button>
    </div>`;
}

/** اعمال عبارت جستجو و بازخوانی صفحه (فیلتر در سرور اجرا می‌شود). */
function liveSearchApply(page) {
  const el = document.getElementById('liveSearch');
  LIVE_QUERY[page] = el ? el.value.trim() : '';
  liveReload();
}

function liveSearchClear(page) {
  LIVE_QUERY[page] = '';
  liveReload();
}

/**
 * گزینه‌های فیلدهای «select» هر نوع رکورد: {کلید فیلد: [{key, titleFa}]}.
 *
 * سرور مالک فهرست‌هاست (/options هر ماژول) و این لایه فقط همان فهرست را در فرم و
 * جدول نشان می‌دهد؛ پس هیچ گزینه‌ای در UI ساخته نمی‌شود. بدون این نگاشت، فیلد
 * انتخابی به ورودی متنی تبدیل می‌شد و کد خام (مثل melli) در جدول دیده می‌شد.
 */
const RK_SELECT_OPTS = {};

/** ثبت گزینه‌های یک فیلد انتخابی از پاسخ /options سرور. */
function rkSetSelectOpts(fieldKey, pairs) {
  const list = (pairs || []).filter(p => p && p.key);
  if (list.length) RK_SELECT_OPTS[fieldKey] = list;
}

/** عنوان فارسی گزینهٔ یک فیلد انتخابی؛ اگر ناشناخته باشد خودِ مقدار برگردانده می‌شود. */
function rkSelectTitle(fieldKey, value) {
  const opts = RK_SELECT_OPTS[fieldKey];
  if (!opts) return value;
  const hit = opts.find(o => String(o.key) === String(value));
  return hit ? hit.titleFa : value;
}

/** یافتن ماژول مالک یک صفحه از منوی تولیدشده (برای پیشوند API). */
function rkFindModule(page) {
  for (const dir of (window.HRMS_MENU || []))
    for (const dep of dir.departments || [])
      for (const grp of dep.groups || [])
        for (const m of grp.modules || [])
          if (m.apiPrefix && (m.screens || []).some(s => s.page === page))
            return {prefix: m.apiPrefix, titleFa: m.titleFa};
  return null;
}

/** نمایش مقدار یک فیلد بر پایهٔ نوع اعلام‌شده در دامنهٔ سرور. */
function rkValue(field, value) {
  if (value === undefined || value === null || value === '') return '<span class="muted">—</span>';
  if (field.type === 'money') return faNum(money(Number(value) || 0));
  if (field.type === 'date') return faDate(value);
  return `<span>${esc(rkSelectTitle(field.key, value))}</span>`;
}

/** ورودی فرم بر پایهٔ نوع فیلد. */
function rkInput(id, field, value) {
  const v = value == null ? '' : value;
  if (field.type === 'number' || field.type === 'money')
    return `<input id="${id}" type="number" min="0" value="${esc(v)}">`;
  if (field.type === 'date')
    return `<input id="${id}" placeholder="1404/07/01" value="${esc(v)}">`;
  // فیلد انتخابی تنها وقتی فهرست دارد؛ وگرنه ورودی متنی می‌ماند.
  const opts = RK_SELECT_OPTS[field.key];
  if (field.type === 'select' && opts && opts.length) {
    return `<select id="${id}"><option value="">— انتخاب کنید —</option>${opts.map(o =>
      `<option value="${esc(o.key)}" ${String(v) === String(o.key) ? 'selected' : ''}>${esc(o.titleFa)}</option>`).join('')}</select>`;
  }
  return `<input id="${id}" value="${esc(v)}">`;
}

/** صفحهٔ زندهٔ یک ماژول رکوردی: جدول، آمار وضعیت، فرم و اقدام‌های گردش‌کار. */
function pageRecordkitLive(page) {
  if (!isLive()) return offlineFallback(page);
  const mod = rkFindModule(page);
  const cfg = RK_PAGES[page];
  if (!mod || !cfg) return liveErrorBox('این صفحه به ماژولی زنده وصل نیست.');

  liveLoad(async () => {
    const schemaRes = await apiFetch(mod.prefix + '/schema');
    if (!schemaRes.ok) return liveErrorBox(schemaRes.message);
    const schema = schemaRes.data;
    const kind = (schema.kinds || []).find(k => k.key === cfg.kind) || (schema.kinds || [])[0];
    if (!kind) return liveErrorBox('این ماژول نوع رکوردی ندارد.');

    const q = new URLSearchParams({size: '200', kind: kind.key});
    if (cfg.status) q.set('status', cfg.status);
    const term = (RK_QUERY[page] || '').trim();
    if (term) q.set('query', term);
    const listRes = await apiFetch(mod.prefix + '/records?' + q.toString());
    if (!listRes.ok) return liveErrorBox(listRes.message);
    const items = (listRes.data && listRes.data.items) || [];
    RK_STATE[page] = {mod, kind};

    // پنل شارژ کارت‌های رفاهی: گزینه‌ها (کارت‌ها، بخش‌ها با نرخ جاری و
    // مناسبت‌های فعال) از خود سرور می‌آید، پس نرخ تازه بدون تغییر کد دیده می‌شود.
    let cardOptions = null;
    if (cfg.panel === 'cards') {
      const optRes = await apiFetch(mod.prefix + '/options');
      cardOptions = optRes.ok ? optRes.data : null;
    }

    // پنل پاداش تحصیلی: مقاطع با مبلغ مصوب جاری از خود سرور می‌آید.
    let bonusOptions = null;
    if (cfg.panel === 'bonus') {
      const optRes = await apiFetch(mod.prefix + '/options');
      bonusOptions = optRes.ok ? optRes.data : null;
    }

    // پنل وام: انواع وام، بانک‌های عامل، مدیریت‌های جدول سهمیه و معیارهای امتیازبندی
    // تخصیص از سرور می‌آید.
    let loanOptions = null;
    let loanQuota = null;
    if (cfg.panel === 'loans') {
      const optRes = await apiFetch(mod.prefix + '/options');
      loanOptions = optRes.ok ? optRes.data : null;
      if (loanOptions) {
        rkSetSelectOpts('loanType', (loanOptions.loanTypes || []).map(t => ({key: t.key, titleFa: t.titleFa})));
        rkSetSelectOpts('bank', (loanOptions.banks || []).map(b => ({key: b.key, titleFa: b.titleFa})));
        rkSetSelectOpts('management', (loanOptions.managementUnits || []).map(u => ({key: u.key, titleFa: u.titleFa})));
        // واحد سازمانی از کاتالوگ مرجع تنظیمات؛ هم فیلد انتخاب فرم و هم ستون جدول وام.
        rkSetSelectOpts('unit', (loanOptions.units || []).map(u => ({key: u.key, titleFa: u.titleFa})));
      }
      // داشبورد تخصیص سهمیه: سرور همهٔ برش‌ها را می‌سازد.
      const quoRes = await apiFetch(mod.prefix + '/quota-dashboard');
      loanQuota = quoRes.ok ? quoRes.data : null;
    }

    // پنل پاداش: گزینه‌ها (انواع پاداش، مدیریت‌های بندر امام، روش‌های پرداخت) هم فهرست
    // جدول و فرم را فارسی می‌کند و هم فرم تخصیص را می‌سازد؛ داشبورد مدیریتی هم همان
    // برش‌های سرور (مدیریت، نفر، نوع، دوره، وضعیت) را برای نمودارها می‌آورد.
    let bonusOpts = null;
    let bonusDash = null;
    if (cfg.panel === 'bonusDash' || cfg.panel === 'bonusPay' || cfg.panel === 'bonusRep') {
      const optRes = await apiFetch(mod.prefix + '/options');
      bonusOpts = optRes.ok ? optRes.data : null;
      if (bonusOpts) {
        rkSetSelectOpts('bonusType', (bonusOpts.types || []).map(t => ({key: t.key, titleFa: t.titleFa})));
        rkSetSelectOpts('management', (bonusOpts.managements || []).map(m => ({key: m.key, titleFa: m.titleFa})));
        rkSetSelectOpts('paymentMethod', (bonusOpts.paymentMethods || []).map(m => ({key: m.key, titleFa: m.titleFa})));
      }
      const dashRes = await apiFetch(mod.prefix + '/dashboard');
      bonusDash = dashRes.ok ? dashRes.data : null;
    }

    // تب شاخص‌های سرمایهٔ انسانی: فهرست فیلد انتخابی (جهت مطلوب) از /options و
    // سنجش با هدف از /indices می‌آید. آستانهٔ «در محدودهٔ هدف» در دامنهٔ سرور است،
    // پس UI فقط همان برچسب و درصد را نشان می‌دهد.
    let indicesView = null;
    if (cfg.panel === 'indices') {
      const optRes = await apiFetch(mod.prefix + '/options');
      if (optRes.ok)
        rkSetSelectOpts('direction', ((optRes.data || {}).directions || []).map(d => ({key: d.key, titleFa: d.titleFa})));
      const anRes = await apiFetch(mod.prefix + '/indices');
      indicesView = anRes.ok ? anRes.data : null;
    }

    // تب نظرسنجی: نرخ مشارکت، دستهٔ کیفی و میانگین‌ها همه در سرور حساب می‌شوند؛ اینجا
    // فقط نمایش‌اند تا عدد رابط با گزارش سرور یکی بماند.
    let surveysView = null;
    if (cfg.panel === 'surveys') {
      const anRes = await apiFetch(mod.prefix + '/surveys');
      surveysView = anRes.ok ? anRes.data : null;
    }

    // شمارش هر وضعیت برای کارت‌های آماری.
    const byStatus = {};
    for (const it of items) byStatus[it.status] = (byStatus[it.status] || 0) + 1;
    const stats = `<div class="grid-stats">
      ${statCard({ico: '🗂️', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.length), label: 'کل ' + kind.titleFa})}
      ${(kind.statuses || []).slice(0, 3).map(s =>
        statCard({ico: '•', tint: '#F8FAFC', accent: '#64748B', val: fa(byStatus[s.value] || 0), label: s.labelFa})).join('')}
    </div>`;

    const fieldCols = (kind.fields || []);
    const head = ['شناسه', 'عنوان']
      .concat(kind.employeeScoped ? ['کارمند'] : [])
      .concat(fieldCols.map(f => f.labelFa))
      .concat(['وضعیت', 'عملیات']);

    const rows = items.map(r => {
      const legal = (kind.actions || []).filter(a => !a.to || ((kind.transitions || {})[r.status] || []).includes(a.to));
      const actionBtns = legal.map(a =>
        `<button class="btn btn-sm ${a.style === 'primary' ? 'btn-primary' : 'btn-ghost'}" onclick="rkAct('${esc(mod.prefix)}','${esc(r.id)}','${esc(a.value)}',${a.noteRequired ? 'true' : 'false'},'${esc(a.labelFa)}')">${esc(a.labelFa)}</button>`).join('');
      // فهرست وضعیت‌های مجاز ویرایش از خود سرور می‌آید؛ اگر اعلام نشده باشد، همان وضعیت آغازین.
      const canEdit = (kind.editableStatuses || [kind.initialStatus]).includes(r.status);
      const editBtn = canEdit
        ? `<button class="btn btn-sm btn-ghost" title="ویرایش" onclick="rkEdit('${esc(page)}','${esc(r.id)}')">✏️</button>
           <button class="btn btn-sm btn-ghost" title="حذف" onclick="rkDelete('${esc(mod.prefix)}','${esc(r.id)}')">🗑️</button>`
        : '';
      // فایل شارژ و فایل پاداش تحصیلی برای تحویل به مالی ساخته می‌شوند؛ خروجی CSV همان ردیف‌ها را می‌دهد.
      const exportBtn = kind.key === 'chargeFiles'
        ? `<button class="btn btn-sm btn-ghost" title="خروجی CSV برای مالی" onclick="cardsExport('${esc(r.id)}')">⬇️</button>`
        : kind.key === 'bonusFiles'
        ? `<button class="btn btn-sm btn-ghost" title="خروجی CSV برای مالی" onclick="bonusExport('${esc(r.id)}')">⬇️</button>` : '';
      const cells = [
        `<span class="num ltr">${esc(r.id)}</span>`,
        `<span>${esc(r.title)}</span>`,
      ].concat(kind.employeeScoped ? [personCell(r.employeeId, r.employeeName)] : [])
       .concat(fieldCols.map(f => rkValue(f, (r.fields || {})[f.key])))
       .concat([statusChip(r.status, r.statusFa)]);
      return `<tr>${cells.map(c => `<td>${c}</td>`).join('')}<td><div class="row-actions">${actionBtns}${editBtn}${exportBtn}</div></td></tr>`;
    });

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      ℹ️ این صفحه روی دادهٔ واقعی سرور کار می‌کند و قواعد گردش‌کار (وضعیت‌ها، انتقال‌ها، اقدام‌ها) از شناسهٔ ماژول در سرور می‌آید.
      ${cfg.status ? `فقط رکوردهای وضعیت «${esc((kind.statuses.find(s => s.value === cfg.status) || {}).labelFa || cfg.status)}» نمایش داده می‌شود.` : ''}
    </div>`;

    const panel = cfg.panel === 'cards' ? cardsPanelHTML(page, cardOptions)
      : cfg.panel === 'indices' ? hrIndicesPanelHTML(indicesView)
      : cfg.panel === 'surveys' ? hrSurveysPanelHTML(surveysView)
      : cfg.panel === 'bonus' ? bonusPanelHTML(page, bonusOptions)
      : cfg.panel === 'bonusPay' ? bonusAllocPanelHTML(bonusOpts)
      // صفحهٔ وام: اول فرم تخصیص وام (ساخت رکورد درخواست وام با کارمزد محاسبه‌شده)،
      // بعد پنل امتیازبندی و رتبه‌بندی تخصیص.
      : cfg.panel === 'loans' ? loanGrantPanelHTML(loanOptions) + loansPanelHTML(page, loanOptions) : '';
    // داشبورد گرافیکی وام: بالای پنل امتیازبندی می‌آید تا ترکیب و روند وام‌ها دیده شود.
    // داشبورد پاداش هم نمودارها و جدول «مبلغ هر نفر در هر مدیریت» را بالای رکوردها می‌آورد.
    const chart = cfg.panel === 'loans'
      ? loansChartHTML(items, loanOptions, kind) + loansQuotaHTML(loanQuota)
      : cfg.panel === 'bonusDash' ? bonusMgmtDashboardHTML(bonusDash)
      : cfg.panel === 'bonusRep' ? bonusMgmtReportHTML(bonusDash)
      : '';
    // در دو تبِ تحلیلی مطالعات، کارت‌های آمار خودِ پنل وضعیت‌ها را پوشش می‌دهند؛ پس
    // نوار آمار عمومی رکوردها تکرار می‌شد و دیده نمی‌شود.
    const hasAnalyticsPanel = cfg.panel === 'indices' || cfg.panel === 'surveys';
    return (hasAnalyticsPanel ? '' : stats) + chart + note + panel + liveTable(head, rows, items.length);
  });

  const extra = (typeof RK_EXTRA !== 'undefined' && RK_EXTRA[page]) ? RK_EXTRA[page] : '';
  // کادر جستجو در «پوسته» می‌ماند (نه در ناحیهٔ بارگذاری‌شده) تا متن تایپ‌شده با هر
  // بازخوانی از دست نرود؛ عبارت در سرور اعمال می‌شود و کاربر هم‌زمان کل/فیلترشده را می‌بیند.
  const search = `<input class="inp w-lg" id="rkSearch" placeholder="🔍 جستجوی شناسه، عنوان، کارمند یا مقدار…"
      value="${esc(RK_QUERY[page] || '')}"
      onkeydown="if(event.key==='Enter'){rkSearch('${esc(page)}')}">
    <button class="btn btn-ghost btn-sm" onclick="rkSearch('${esc(page)}')">جستجو</button>`;
  return liveShell(mod.titleFa, 'مدیریت رکوردها روی دادهٔ واقعی سرور — ایجاد، ویرایش، جستجو و حذف',
    extra + search + `<button class="btn btn-primary" onclick="rkCreate('${esc(page)}')">➕ رکورد تازه</button>`);
}

/* ------------------------- فرم ایجاد و ویرایش ------------------------- */

function rkCreate(page) {
  const st = RK_STATE[page];
  if (!st) { toast('شناسهٔ ماژول هنوز بارگذاری نشده است', 'error'); return; }
  rkOpenForm(page, st.kind, null);
}

/** جستجو: عبارت را از کادر می‌خواند، در سرور اعمال می‌کند و صفحه را بازخوانی می‌کند.
 *  خواندن عبارت پیش از بازخوانی ضروری است، چون بازخوانی کادر را از نو می‌سازد. */
function rkSearch(page) {
  const el = document.getElementById('rkSearch');
  RK_QUERY[page] = el ? el.value.trim() : '';
  liveReload();
}
function rkEdit(page, id) {
  const st = RK_STATE[page];
  if (!st) return;
  apiFetch(st.mod.prefix + '/records/' + encodeURIComponent(id)).then(res => {
    if (!res.ok) { toast(res.message, 'error'); return; }
    rkOpenForm(page, st.kind, res.data);
  });
}
async function rkOpenForm(page, kind, existing) {
  const isEdit = !!existing;
  const empField = kind.employeeScoped
    ? `<div class="field"><label>کارمند <span class="req">*</span></label>${await liveEmployeeSelect('rkEmp', existing && existing.employeeId)}</div>`
    : '';
  const fields = (kind.fields || []).map(f =>
    `<div class="field"><label>${esc(f.labelFa)}${f.required ? ' <span class="req">*</span>' : ''}</label>${rkInput('rkF_' + f.key, f, existing && existing.fields ? existing.fields[f.key] : '')}</div>`).join('');
  openModal({
    title: (isEdit ? 'ویرایش ' : 'ثبت ') + kind.titleFa,
    body: `<div class="form-grid">
      <div class="field"><label>عنوان <span class="req">*</span></label><input id="rkTitle" value="${esc(existing ? existing.title : '')}"></div>
      ${empField}
      ${fields}
      <div class="field" style="grid-column:1/-1"><label>یادداشت</label><input id="rkNote" value="${esc(existing ? existing.noteFa || '' : '')}"></div>
    </div>`,
    foot: `<button class="btn btn-primary" onclick="rkSave('${esc(page)}',${isEdit ? `'${esc(existing.id)}'` : 'null'})">${isEdit ? 'ذخیره' : 'ثبت'}</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}
async function rkSave(page, id) {
  const st = RK_STATE[page];
  if (!st) return;
  const kind = st.kind;
  const body = {kind: kind.key, title: $('#rkTitle').value.trim(), noteFa: $('#rkNote').value.trim(), fields: {}};
  for (const f of kind.fields || []) {
    const el = document.getElementById('rkF_' + f.key);
    if (el) body.fields[f.key] = el.value.trim();
  }
  if (kind.employeeScoped && $('#rkEmp')) body.employeeId = $('#rkEmp').value;
  const path = st.mod.prefix + '/records' + (id ? '/' + encodeURIComponent(id) : '');
  const res = await apiFetch(path, {method: id ? 'PATCH' : 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  closeModal();
  toast(id ? 'رکورد ویرایش شد' : 'رکورد ثبت شد');
  liveReload();
}

/* ---------------------------- اقدام گردش‌کار ---------------------------- */

function rkAct(prefix, id, action, noteRequired, labelFa) {
  let note = '';
  if (noteRequired) {
    note = prompt('دلیل «' + labelFa + '» را وارد کنید:') || '';
    if (!note) return;
  }
  liveSend(prefix + '/records/' + encodeURIComponent(id) + '/action',
    {method: 'POST', body: {action, noteFa: note}}, '«' + labelFa + '» انجام شد')
    .then(ok => ok && liveReload());
}

function rkDelete(prefix, id) {
  if (!confirm('این رکورد حذف شود؟')) return;
  liveSend(prefix + '/records/' + encodeURIComponent(id), {method: 'DELETE'}, 'رکورد حذف شد')
    .then(ok => ok && liveReload());
}

/* =====================================================================
   واحد مطالعات منابع انسانی — تب شاخص‌ها و تب نظرسنجی
   =====================================================================

   این دو پنل روی مسیرهای تحلیلی خودِ ماژول کار می‌کنند (/indices و /surveys).
   سنجش «در محدودهٔ هدف» و نرخ مشارکت در دامنهٔ سرور حساب می‌شود؛ این لایه فقط
   همان برچسب و درصد اعلام‌شده را نمایش می‌دهد، پس عددِ رابط با گزارش سرور یکی
   می‌ماند. رکوردها هم پایین همین پنل با موتور عمومی ثبت و ویرایش می‌شوند.
   --------------------------------------------------------------------- */

/** نگاشت وضعیت سنجش شاخص به چیپ رنگی (برچسب فارسی از سرور می‌آید). */
const HR_ASSESSMENT_CLS = {in_target: 'c-ok', near_target: 'c-wait', out_of_target: 'c-rej'};

/** نگاشت دستهٔ کیفی مشارکت نظرسنجی به چیپ رنگی. */
const HR_PARTICIPATION_CLS = {good: 'c-ok', acceptable: 'c-wait', weak: 'c-rej'};

/**
 * پنل «شاخص‌های سرمایهٔ انسانی»: جمع سنجش با هدف + فهرست رتبه‌بندی‌شده.
 *
 * ترتیب فهرست را سرور می‌دهد (خارج از هدف اول) و اینجا دست‌نخورده می‌ماند؛
 * اگر هر دو طرف خودشان مرتب کنند، دو ترتیب متفاوت دیده می‌شود.
 */
function hrIndicesPanelHTML(view) {
  if (!view || !view.items) return '';
  const s = view.summary || {};
  const stats = `<div class="grid-stats">
    ${statCard({ico: '🎯', tint: '#EFF6FF', accent: '#2563EB', val: fa(s.total || 0), label: 'شاخص فعال دوره'})}
    ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(s.inTarget || 0), label: 'در محدودهٔ هدف'})}
    ${statCard({ico: '📉', tint: '#FEE2E2', accent: '#DC2626', val: fa(s.outOfTarget || 0), label: 'خارج از هدف'})}
    ${statCard({ico: '📈', tint: '#EDE9FE', accent: '#7C3AED', val: `${fa(s.targetRatePct || 0)}٪`, label: 'نرخ تحقق هدف'})}
  </div>`;

  const bars = hBarChart((view.items || []).slice(0, 8).map(it => ({
    label: it.titleFa,
    value: it.achievedPct,
    pct: it.achievedPct,
    color: it.assessment === 'in_target' ? '#16A34A' : it.assessment === 'near_target' ? '#D97706' : '#DC2626',
    text: `${fa(it.achievedPct)}٪ از هدف`,
  })), 'شاخصی برای نمایش ثبت نشده است');
  const chart = `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">درصد تحقق هدف هر شاخص</div>${bars}
    ${s.needsReview ? `<div class="muted" style="font-size:.76rem;margin-top:8px">
      ${fa(s.needsReview)} شاخص در وضعیت «نیازمند بازنگری تعریف» است و باید پیش از تفسیر نتیجه بازبینی شود.
    </div>` : ''}
  </div>`;

  const rows = (view.items || []).map(it => `<tr>
    <td><span>${esc(it.titleFa)}</span>
      <div class="muted ltr" style="font-size:.68rem">${esc(it.id)}</div></td>
    <td style="font-size:.76rem">${esc(it.formula || '—')}</td>
    <td class="num">${fa(it.target)}${it.unit ? ' ' + esc(it.unit) : ''}</td>
    <td class="num">${fa(it.current)}${it.unit ? ' ' + esc(it.unit) : ''}</td>
    <td class="num">${fa(it.achievedPct)}٪</td>
    <td>${pChip(HR_ASSESSMENT_CLS[it.assessment] || 'c-wait', it.assessmentFa || '—')}</td>
    <td style="font-size:.76rem">${esc(it.owner || '—')}<div class="muted num" style="font-size:.68rem">${esc(it.period || '')}</div></td>
    <td>${statusChip(it.status, it.statusFa)}</td>
  </tr>`);
  const table = `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">سنجش شاخص‌ها با هدف دوره (${fa((view.items || []).length)} شاخص)</div>
    ${liveTable(['شاخص', 'فرمول محاسبه', 'هدف', 'مقدار دوره', 'نسبت به هدف', 'سنجش', 'مالک / دوره', 'وضعیت'], rows, rows.length)}
  </div>`;

  return stats + chart + table + statNote(view.noteFa) + statNote(s.noteFa);
}

/**
 * پنل «نظرسنجی و جو سازمانی»: تحلیل مشارکت و رضایت + فهرست دوره‌ها.
 *
 * نرخ مشارکت و دستهٔ کیفی از سرور می‌آید؛ جمع‌های میانگین هم همان چیزی است که
 * گزارش واحد مطالعات نشان می‌دهد.
 */
function hrSurveysPanelHTML(view) {
  if (!view || !view.items) return '';
  const s = view.summary || {};
  const stats = `<div class="grid-stats">
    ${statCard({ico: '📋', tint: '#EFF6FF', accent: '#2563EB', val: fa(s.total || 0), label: 'کل نظرسنجی'})}
    ${statCard({ico: '🟢', tint: '#DCFCE7', accent: '#16A34A', val: `${fa(s.avgParticipationPct || 0)}٪`, label: 'میانگین نرخ مشارکت'})}
    ${statCard({ico: '😊', tint: '#EDE9FE', accent: '#7C3AED', val: `${fa(s.avgSatisfactionPct || 0)}٪`, label: 'میانگین شاخص رضایت'})}
    ${statCard({ico: '🔧', tint: '#FEF3C7', accent: '#D97706', val: fa(s.actionsIdentified || 0), label: 'نظرسنجی دارای اقدام اصلاحی'})}
  </div>`;

  const participation = hBarChart((view.items || []).map(it => ({
    label: it.titleFa,
    value: it.participationPct,
    pct: it.participationPct,
    color: it.band === 'good' ? '#16A34A' : it.band === 'acceptable' ? '#D97706' : '#DC2626',
    text: `${fa(it.participationPct)}٪ — ${it.bandFa}`,
  })), 'نظرسنجی ثبت نشده است');
  const chart = `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">نرخ مشارکت هر نظرسنجی (${fa(s.participants || 0)} پاسخ ثبت‌شده)</div>${participation}
    <div class="muted" style="font-size:.76rem;margin-top:8px">
      ${fa(s.open || 0)} نظرسنجی در حال اجرا، ${fa(s.closed || 0)} بسته‌شده و
      ${fa((s.total || 0) - (s.open || 0) - (s.closed || 0))} در مرحلهٔ طراحی است؛
      تحلیل فقط جمع می‌گیرد و تصمیم دربارهٔ اقدام اصلاحی در رکورد همان نظرسنجی ثبت می‌شود.
    </div>
  </div>`;

  const rows = (view.items || []).map(it => `<tr>
    <td><span>${esc(it.titleFa)}</span><div class="muted ltr" style="font-size:.68rem">${esc(it.id)}</div></td>
    <td class="num">${fa(it.participants)} از ${fa(it.population)}</td>
    <td class="num">${fa(it.participationPct)}٪</td>
    <td>${pChip(HR_PARTICIPATION_CLS[it.band] || 'c-wait', it.bandFa || '—')}</td>
    <td class="num">${it.satisfactionIndex ? fa(it.satisfactionIndex) + '٪' : '—'}</td>
    <td style="font-size:.78rem">${esc(it.weakAxis || '—')}</td>
    <td>${faDate(it.startDate)} تا ${faDate(it.endDate)}</td>
    <td>${statusChip(it.status, it.statusFa)}</td>
  </tr>`);
  const table = `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">دوره‌های نظرسنجی و نتیجهٔ تحلیل‌شده (${fa((view.items || []).length)} دوره)</div>
    ${liveTable(['نظرسنجی', 'مشارکت', 'نرخ', 'دستهٔ مشارکت', 'شاخص رضایت', 'محور ضعیف', 'دوره', 'وضعیت'], rows, rows.length)}
  </div>`;

  return stats + chart + table + statNote(view.noteFa) + statNote(s.noteFa);
}

/* =====================================================================
   پنل شارژ کارت‌های رفاهی (پتروکارت و نفت‌کارت)
   =====================================================================

   این پنل روی همان API سرور کار می‌کند: گزینه‌ها از /options، پیش‌نمایش از
   /charge/preview و ثبت فایل از /charge-files؛ خروجی مالی هم CSV همان فایل است.
   هیچ نرخ یا برچسبی در این لایه هاردکد نیست، پس بازتعریف نرخ‌ها و مناسبت‌ها
   خودکار در همین فرم دیده می‌شود.
   --------------------------------------------------------------------- */

/** پنل شارژ؛ اگر گزینه‌ها بارگذاری نشده باشند چیزی نشان داده نمی‌شود. */
function cardsPanelHTML(page, options) {
  if (!options) return '';
  const defCard = page === 'naftmart' ? 'naftmart' : 'petrokart';
  const period = (options.todayFa || '').slice(0, 7);

  const cardOpts = (options.cards || []).map(c =>
    `<option value="${esc(c.key)}" ${c.key === defCard ? 'selected' : ''}>${esc(c.titleFa)}</option>`).join('');

  // فقط بخش‌های همان کارت فعال و دیده می‌شوند؛ با تغییر کارت، همین قاعده اجرا می‌شود.
  const sectionRows = (options.sections || []).map(s => {
    const on = s.card === defCard;
    return `<label class="chk" data-card="${esc(s.card)}" style="display:${on ? 'inline-flex' : 'none'};align-items:center;gap:6px">
      <input type="checkbox" class="cupSec" value="${esc(s.key)}" ${on ? 'checked' : 'disabled'}>
      <span>${esc(s.titleFa)} <span class="muted num">${faNum(money(s.amount))} ریال</span></span>
    </label>`;
  }).join('');

  const occOpts = '<option value="">— بدون شارژ مناسبتی —</option>' +
    (options.occasions || []).map(o =>
      `<option value="${esc(o.key)}">${esc(o.titleFa)}${o.dateFa ? ' — ' + esc(o.dateFa) : ''}${o.amount ? ' — ' + faNum(money(o.amount)) + ' ریال' : ''}</option>`).join('');

  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧾 پنل شارژ و ساخت فایل مالی</div>
    <div class="grid-3" style="gap:10px">
      <div class="field"><label>کارت</label><select id="cupCard" onchange="cardsSyncSections()">${cardOpts}</select></div>
      <div class="field"><label>دورهٔ شارژ (YYYY/MM)</label><input id="cupPeriod" value="${esc(period)}" placeholder="1404/07"></div>
      <div class="field"><label>دامنهٔ شارژ</label><select id="cupScope">
        <option value="group">گروهی — همهٔ مشمولان فعال</option>
        <option value="individual">انفرادی — کارکنان انتخابی</option>
      </select></div>
      <div class="field"><label>کد کارکنان برای شارژ انفرادی (با کاما)</label><input id="cupEmps" placeholder="EMP-005, EMP-010"></div>
      <div class="field"><label>مناسبت تقویمی</label><select id="cupOccasion">${occOpts}</select></div>
    </div>
    <div style="margin:6px 0 10px">
      <div class="muted" style="margin-bottom:4px">بخش‌های شارژ — نرخ جاری هر بخش کنارش آمده است</div>
      <div id="cupSections" style="display:flex;flex-wrap:wrap;gap:8px 18px">${sectionRows}</div>
    </div>
    <div class="row-actions">
      <button class="btn btn-primary" onclick="cardsPreview()">🔎 پیش‌نمایش</button>
      <button class="btn btn-ghost" onclick="cardsBuild()">🧾 ثبت فایل شارژ</button>
      <button class="btn btn-ghost" onclick="cardsClearResult()">↩️ پاک‌کردن</button>
    </div>
    <div id="cupResult" style="margin-top:12px"></div>
  </div>`;
}

/** نمایش فقط بخش‌های کارت انتخابی. */
function cardsSyncSections() {
  const card = ($('#cupCard') || {}).value;
  document.querySelectorAll('#cupSections label.chk').forEach(l => {
    const on = l.dataset.card === card;
    l.style.display = on ? 'inline-flex' : 'none';
    const box = l.querySelector('input');
    box.disabled = !on;
    box.checked = on;
  });
}

/** درخواست شارژ از روی فرم پنل. */
function cardsRequest() {
  return {
    card: $('#cupCard').value,
    scope: $('#cupScope').value,
    sections: [...document.querySelectorAll('#cupSections input.cupSec:checked')].map(i => i.value),
    period: ($('#cupPeriod').value || '').trim(),
    occasionId: $('#cupOccasion').value,
    employeeIds: ($('#cupEmps').value || '').split(',').map(s => s.trim()).filter(Boolean),
  };
}

function cardsPreview() { cardsRun(true); }
function cardsBuild() { cardsRun(false); }

/** اجرای پیش‌نمایش یا ثبت فایل شارژ. */
async function cardsRun(trial) {
  const box = $('#cupResult');
  if (!box) return;
  box.innerHTML = '<div class="muted">در حال محاسبه…</div>';
  const path = trial ? '/api/v1/cards/charge/preview' : '/api/v1/cards/charge-files';
  const res = await apiFetch(path, {method: 'POST', body: cardsRequest()});
  if (!res.ok) {
    box.innerHTML = `<div class="empty-state"><div class="big">⚠️</div>${esc(res.message)}</div>`;
    toast(res.message, 'error');
    return;
  }
  if (!trial) {
    toast('فایل شارژ ' + (res.data.fileNoFa || '') + ' ثبت شد');
    liveReload();
    // فهرست رکوردها تازه می‌شود و پنل از نو ساخته می‌شود؛ نتیجه را پس از همان
    // بازسازی دوباره می‌نشانیم تا کاربر خروجی مالی را از دست ندهد.
    cardsRenderAfterReload(res.data);
    return;
  }
  cardsRenderResult(res.data, trial);
}

/** پس از تازه‌سازی فهرست، نتیجهٔ فایل ثبت‌شده را در پنل تازه می‌نشاند. */
async function cardsRenderAfterReload(data) {
  for (let i = 0; i < 40; i++) {
    await new Promise(r => setTimeout(r, 150));
    if (document.getElementById('cupResult') && document.getElementById('cupCard')) {
      cardsRenderResult(data, false);
      return;
    }
  }
}

/** نتیجهٔ شارژ: جمع هر بخش، ردیف‌ها، هشدارها و خروجی مالی. */
function cardsRenderResult(r, trial) {
  const box = $('#cupResult');
  if (!box) return;
  const totals = (r.bySection || []).map(t =>
    `<span class="chip c-ok num">${esc(t.sectionFa)}: ${faNum(money(t.total))} ریال — ${fa(t.count)} ردیف</span>`).join(' ');
  const warnings = (r.warnings || []).length
    ? `<div class="login-err show" style="text-align:right">⚠️ ${r.warnings.map(esc).join(' — ')}</div>` : '';
  const exportBtn = r.fileId
    ? `<button class="btn btn-primary" onclick="cardsExport('${esc(r.fileId)}')">⬇️ خروجی CSV برای مالی</button>` : '';
  const head = ['کارمند', 'ذی‌نفع', 'نسبت', 'بخش', 'مناسبت', 'مبلغ (ریال)'];
  const rows = (r.rows || []).slice(0, 60).map(x => `<tr>
    <td>${personCell(x.employeeId, x.employeeName)}</td>
    <td>${esc(x.beneficiary)}</td>
    <td>${esc(x.relation || '—')}</td>
    <td>${esc(x.sectionFa)}</td>
    <td>${esc(x.occasionFa || '—')}</td>
    <td class="num">${faNum(money(x.amount))}</td></tr>`);
  box.innerHTML = warnings +
    `<div style="display:flex;gap:8px;flex-wrap:wrap;margin-bottom:8px">${totals}</div>
     <div class="muted" style="margin-bottom:8px">${trial ? 'پیش‌نمایش' : 'فایل ثبت‌شده'}
       — ${fa(r.count)} ردیف — جمع ${faNum(money(r.total))} ریال
       ${r.fileId ? ' — شناسهٔ فایل: ' + esc(r.fileId) : ''}</div>
     <div style="margin-bottom:8px">${exportBtn}</div>` +
    liveTable(head, rows, (r.rows || []).length);
}

function cardsClearResult() {
  const box = $('#cupResult');
  if (box) box.innerHTML = '';
}

/** دانلود CSV فایل شارژ برای واحد مالی (با هدر احراز هویت، پس Blob لازم است). */
async function cardsExport(id) {
  const res = await apiRaw('/api/v1/cards/charge-files/' + encodeURIComponent(id) + '/export');
  if (!res.ok) { toast('خروجی گرفتن ناموفق بود', 'error'); return; }
  const text = await res.text();
  const url = URL.createObjectURL(new Blob([text], {type: 'text/csv;charset=utf-8'}));
  const a = document.createElement('a');
  a.href = url;
  a.download = id + '.csv';
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
  toast('فایل شارژ آمادهٔ ارسال به مالی دانلود شد');
}

/* =====================================================================
   پنل پاداش تحصیلی فرزندان ممتاز (سر فصل مستقل از پتروکارت و نفت‌کارت)
   =====================================================================

   این پنل روی همان API سرور کار می‌کند: گزینه‌ها (مقاطع با مبلغ مصوب جاری) از
   /options، پیش‌نمایش از /bonus/preview و ثبت فایل از /bonus-files؛ خروجی مالی هم
   CSV همان فایل است. هیچ مبلغ یا برچسبی در این لایه هاردکد نیست.
   --------------------------------------------------------------------- */

/** سال تحصیلی جاری را از تاریخ شمسی سرور می‌سازد (سال تحصیلی از مهر آغاز می‌شود). */
function bonusAcademicYear(todayFa) {
  const parts = String(todayFa || '').split('/');
  const y = Number(parts[0]) || 1404;
  const m = Number(parts[1]) || 7;
  return m >= 7 ? y + '/' + (y + 1) : (y - 1) + '/' + y;
}

/** پنل محاسبهٔ پاداش؛ اگر گزینه‌ها بارگذاری نشده باشند چیزی نشان داده نمی‌شود. */
function bonusPanelHTML(page, options) {
  if (!options) return '';
  const period = bonusAcademicYear(options.todayFa);
  const levels = (options.levels || []).map(l =>
    `<label class="chk" style="display:inline-flex;align-items:center;gap:6px">
      <input type="checkbox" class="ebpLevel" value="${esc(l.key)}" checked>
      <span>${esc(l.titleFa)} <span class="muted num">${l.amount ? faNum(money(l.amount)) + ' ریال' : '— مبلغ مصوب ثبت نشده'}</span></span>
    </label>`).join('');

  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🎓 پنل محاسبهٔ پاداش تحصیلی و ساخت فایل مالی</div>
    <div class="grid-3" style="gap:10px">
      <div class="field"><label>دورهٔ پرداخت (سال تحصیلی)</label><input id="ebpPeriod" value="${esc(period)}" placeholder="1404/1405"></div>
      <div class="field"><label>دامنهٔ محاسبه</label><select id="ebpScope">
        <option value="group">گروهی — همهٔ مشمولان ممتاز</option>
        <option value="individual">انفرادی — کارکنان انتخابی</option>
      </select></div>
      <div class="field"><label>کد کارکنان برای محاسبهٔ انفرادی (با کاما)</label><input id="ebpEmps" placeholder="EMP-005, EMP-010"></div>
    </div>
    <div style="margin:6px 0 10px">
      <div class="muted" style="margin-bottom:4px">مقطع‌های مشمول محاسبه — مبلغ مصوب هر مقطع کنارش آمده است</div>
      <div id="ebpLevels" style="display:flex;flex-wrap:wrap;gap:8px 18px">${levels}</div>
    </div>
    <div class="row-actions">
      <button class="btn btn-primary" onclick="bonusPreview()">🔎 پیش‌نمایش</button>
      <button class="btn btn-ghost" onclick="bonusBuild()">🧾 ثبت فایل پرداخت</button>
      <button class="btn btn-ghost" onclick="bonusClearResult()">↩️ پاک‌کردن</button>
    </div>
    <div id="ebpResult" style="margin-top:12px"></div>
  </div>`;
}

/** درخواست محاسبهٔ پاداش از روی فرم پنل. */
function bonusRequest() {
  return {
    levels: [...document.querySelectorAll('#ebpLevels input.ebpLevel:checked')].map(i => i.value),
    scope: $('#ebpScope').value,
    period: ($('#ebpPeriod').value || '').trim(),
    employeeIds: ($('#ebpEmps').value || '').split(',').map(s => s.trim()).filter(Boolean),
  };
}

function bonusPreview() { bonusRun(true); }
function bonusBuild() { bonusRun(false); }

/** اجرای پیش‌نمایش یا ثبت فایل پرداخت پاداش تحصیلی. */
async function bonusRun(trial) {
  const box = $('#ebpResult');
  if (!box) return;
  box.innerHTML = '<div class="muted">در حال محاسبه…</div>';
  const path = trial ? '/api/v1/education_bonus/bonus/preview' : '/api/v1/education_bonus/bonus-files';
  const res = await apiFetch(path, {method: 'POST', body: bonusRequest()});
  if (!res.ok) {
    box.innerHTML = `<div class="empty-state"><div class="big">⚠️</div>${esc(res.message)}</div>`;
    toast(res.message, 'error');
    return;
  }
  if (!trial) {
    toast('فایل پرداخت ' + (res.data.fileNoFa || '') + ' ثبت شد');
    liveReload();
    bonusRenderAfterReload(res.data);
    return;
  }
  bonusRenderResult(res.data, trial);
}

/** پس از تازه‌سازی فهرست، نتیجهٔ فایل ثبت‌شده را در پنل تازه می‌نشاند. */
async function bonusRenderAfterReload(data) {
  for (let i = 0; i < 40; i++) {
    await new Promise(r => setTimeout(r, 150));
    if (document.getElementById('ebpResult') && document.getElementById('ebpPeriod')) {
      bonusRenderResult(data, false);
      return;
    }
  }
}

/** نتیجهٔ پاداش: جمع هر مقطع، ردیف‌ها، هشدارها و خروجی مالی. */
function bonusRenderResult(r, trial) {
  const box = $('#ebpResult');
  if (!box) return;
  const totals = (r.byLevel || []).map(t =>
    `<span class="chip c-ok num">${esc(t.levelFa)}: ${faNum(money(t.total))} ریال — ${fa(t.count)} ردیف</span>`).join(' ');
  const warnings = (r.warnings || []).length
    ? `<div class="login-err show" style="text-align:right">⚠️ ${r.warnings.map(esc).join(' — ')}</div>` : '';
  const exportBtn = r.fileId
    ? `<button class="btn btn-primary" onclick="bonusExport('${esc(r.fileId)}')">⬇️ خروجی CSV برای مالی</button>` : '';
  const head = ['کارمند', 'دانش‌آموز ممتاز', 'نسبت', 'مقطع', 'معدل', 'مبلغ (ریال)'];
  const rows = (r.rows || []).slice(0, 60).map(x => `<tr>
    <td>${personCell(x.employeeId, x.employeeName)}</td>
    <td>${esc(x.student)}</td>
    <td>${esc(x.relation || '—')}</td>
    <td>${esc(x.levelFa)}</td>
    <td>${esc(x.grade || '—')}</td>
    <td class="num">${faNum(money(x.amount))}</td></tr>`);
  box.innerHTML = warnings +
    `<div style="display:flex;gap:8px;flex-wrap:wrap;margin-bottom:8px">${totals}</div>
     <div class="muted" style="margin-bottom:8px">${trial ? 'پیش‌نمایش' : 'فایل ثبت‌شده'}
       — ${fa(r.count)} ردیف — جمع ${faNum(money(r.total))} ریال
       ${r.fileId ? ' — شناسهٔ فایل: ' + esc(r.fileId) : ''}</div>
     <div style="margin-bottom:8px">${exportBtn}</div>` +
    liveTable(head, rows, (r.rows || []).length);
}

function bonusClearResult() {
  const box = $('#ebpResult');
  if (box) box.innerHTML = '';
}

/** دانلود CSV فایل پاداش تحصیلی برای واحد مالی (با هدر احراز هویت، پس Blob لازم است). */
async function bonusExport(id) {
  const res = await apiRaw('/api/v1/education_bonus/bonus-files/' + encodeURIComponent(id) + '/export');
  if (!res.ok) { toast('خروجی گرفتن ناموفق بود', 'error'); return; }
  const text = await res.text();
  const url = URL.createObjectURL(new Blob([text], {type: 'text/csv;charset=utf-8'}));
  const a = document.createElement('a');
  a.href = url;
  a.download = id + '.csv';
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
  toast('فایل پاداش تحصیلی آمادهٔ ارسال به مالی دانلود شد');
}

/* =====================================================================
   پنل وام کارکنان و امتیازبندی تخصیص وام
   =====================================================================

   انواع وام، بانک‌های عامل و معیارهای امتیازبندی از /options سرور می‌آید و
   رتبه‌بندی از /allocation/preview؛ هیچ نرخ، معیار یا ضریبی در این لایه
   هاردکد نیست، پس افزودن نوع وام یا معیار تازه در دامنه خودکار دیده می‌شود.
   --------------------------------------------------------------------- */

/** پنل وام؛ اگر گزینه‌ها بارگذاری نشده باشند چیزی نشان داده نمی‌شود. */
function loansPanelHTML(page, options) {
  if (!options) return '';
  const types = (options.loanTypes || []).map(t =>
    `<span class="chip c-ok">${esc(t.titleFa)}${t.isSupplement ? ' — متمم' : ''}
       <span class="muted num">کارمزد ${faNum(t.feeRatePercent || 0)}٪${t.maxInstallments ? ' · تا ' + fa(t.maxInstallments) + ' قسط' : ''}</span></span>`).join(' ');
  const banks = (options.banks || []).map(b =>
    `<span class="chip">${esc(b.titleFa)}</span>`).join(' ');
  const crit = (options.criteria || []).map(c =>
    `<span class="chip">${esc(c.titleFa)} <span class="muted num">×${fa(c.weight)}</span></span>`).join(' ');

  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🏦 پنل وام و امتیازبندی تخصیص</div>
    <div style="margin-bottom:8px"><div class="muted" style="margin-bottom:4px">انواع وام (کارمزد و سقف اقساط)</div>${types}</div>
    <div style="margin-bottom:8px"><div class="muted" style="margin-bottom:4px">بانک‌های عامل</div>${banks}</div>
    <div style="margin-bottom:10px"><div class="muted" style="margin-bottom:4px">معیارهای امتیازبندی تخصیص (ضریب اهمیت)</div>${crit}</div>
    <div class="form-grid" style="gap:10px">
      <div class="field"><label>کد متقاضیان برای امتیازبندی (با کاما؛ خالی = همهٔ نمونه‌ها)</label><input id="lnEmps" placeholder="EMP-005, EMP-013"></div>
    </div>
    <div class="row-actions" style="margin-top:8px">
      <button class="btn btn-primary" onclick="loansPreview()">🏅 امتیازبندی و رتبه‌بندی تخصیص</button>
      <button class="btn btn-ghost" onclick="loansClearResult()">↩️ پاک‌کردن</button>
    </div>
    <div id="lnResult" style="margin-top:12px"></div>
  </div>`;
}

/** اجرای امتیازبندی تخصیص وام روی سرور. */
async function loansPreview() {
  const box = $('#lnResult');
  if (!box) return;
  box.innerHTML = '<div class="muted">در حال امتیازبندی…</div>';
  const employeeIds = ($('#lnEmps').value || '').split(',').map(s => s.trim()).filter(Boolean);
  const res = await apiFetch('/api/v1/loans/allocation/preview', {method: 'POST', body: {employeeIds}});
  if (!res.ok) {
    box.innerHTML = `<div class="empty-state"><div class="big">⚠️</div>${esc(res.message)}</div>`;
    toast(res.message, 'error');
    return;
  }
  loansRenderResult(res.data);
}

/** نمایش رتبه‌بندی امتیازبندی با ریز امتیاز هر معیار. */
function loansRenderResult(r) {
  const box = $('#lnResult');
  if (!box) return;
  const head = ['رتبه', 'کارمند', 'امتیاز کل', 'ریز امتیاز معیارها'];
  const rows = (r.rows || []).map(x => {
    const parts = (x.breakdown || []).filter(b => b.points)
      .map(b => `${esc(b.criterionFa)}: ${fa(b.points)} (${esc(b.valueFa || '—')})`).join(' · ');
    return `<tr>
      <td class="num">${fa(x.rank)}</td>
      <td>${personCell(x.employeeId, x.employeeName)}</td>
      <td class="num"><b>${fa(x.total)}</b></td>
      <td style="font-size:.78rem">${parts || '<span class="muted">—</span>'}</td></tr>`;
  });
  box.innerHTML = `<div class="muted" style="margin-bottom:8px">${fa(r.count)} متقاضی — ${esc(r.noteFa || '')}</div>` +
    liveTable(head, rows, (r.rows || []).length);
}

function loansClearResult() {
  const box = $('#lnResult');
  if (box) box.innerHTML = '';
}

/**
 * فرم تخصیص وام (گروهی به چند کارمند یا تک‌کارمند).
 *
 * گزینه‌ها از /api/v1/loans/options می‌آید؛ نرخ کارمزد، سقف مبلغ و سقف اقساط هر نوع وام
 * از همان پاسخ خوانده می‌شود و مبالغ نهایی (کارمزد، هر قسط) را سرور حساب می‌کند — همان
 * قاعده‌ای که ماژول پاداش دارد: UI هیچ رقمی از خودش نمی‌سازد.
 */
function loanGrantPanelHTML(options) {
  if (!options) return '';
  const typeOpts = (options.loanTypes || []).map(t =>
    // داخل <option> فقط متن ساده مجاز است؛ faNum یک <span> می‌سازد، پس اینجا fa به کار می‌رود.
    `<option value="${esc(t.key)}" data-max-amount="${esc(String(t.maxAmount || 0))}"` +
    ` data-max-installments="${esc(String(t.maxInstallments || 0))}"` +
    ` data-fee-rate="${esc(String(t.feeRatePercent || 0))}"` +
    ` data-supplement="${t.isSupplement ? '1' : '0'}">${esc(t.titleFa)}` +
    `${t.feeRatePercent ? ' — کارمزد ' + fa(t.feeRatePercent) + '٪' : ''}` +
    `${t.maxAmount ? ' — سقف ' + fa(money(t.maxAmount)) + ' ریال' : ''}</option>`).join('');
  const bankOpts = (options.banks || []).map(b =>
    `<option value="${esc(b.key)}">${esc(b.titleFa)}</option>`).join('');
  const mgmtOpts = (options.managementUnits || []).map(m =>
    `<option value="${esc(m.key)}">${esc(m.titleFa)}</option>`).join('');
  // مبلغ و اقساط با سیاست نخستین نوع وام پیش‌پر می‌شوند تا فرم از همان ابتدا معنادار
  // باشد؛ با تغییر نوع، همان مقادیر از گزینهٔ تازه به‌روز می‌شوند (loanGrantSyncType).
  const first = (options.loanTypes || [])[0] || {};
  // فهرست واحدها برای فیلترکردن بر پایهٔ مدیریت متقاضی در حافظهٔ همین فرم نگه داشته می‌شود.
  LOAN_GRANT_UNITS = options.units || [];

  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🏦 فرم تخصیص وام (گروهی به چند کارمند یا تک‌کارمند)</div>
    <div class="form-grid" style="gap:10px">
      <div class="field"><label>نوع وام</label><select id="lngType" onchange="loanGrantSyncType()">${typeOpts}</select></div>
      <div class="field"><label>مبلغ وام هر نفر (ریال)</label><input id="lngAmount" type="number" min="0" value="${esc(String(first.maxAmount || ''))}"></div>
      <div class="field"><label>تعداد اقساط <span class="muted" id="lngInstHint">${first.maxInstallments ? '— سقف مصوب ' + fa(first.maxInstallments) : ''}</span></label><input id="lngInstallments" type="number" min="0" value="${esc(String(first.maxInstallments || ''))}"></div>
      <div class="field"><label>بانک عامل</label><select id="lngBank"><option value="">— بدون بانک عامل (صندوق داخلی) —</option>${bankOpts}</select></div>
      <div class="field"><label>مدیریت متقاضی</label><select id="lngMgmt" onchange="loanGrantSyncUnits()"><option value="">—</option>${mgmtOpts}</select></div>
      <div class="field"><label>واحد سازمانی (اختیاری)</label><select id="lngUnit">${loanGrantUnitOptions('')}</select></div>
      <div class="field"><label>تاریخ دریافت (YYYY/MM/DD)</label><input id="lngDate" value="${esc(options.todayFa || '')}" placeholder="1404/07/01"></div>
      <div class="field"><label>نام ضامن</label><input id="lngGuarantor" placeholder="نام و نام خانوادگی ضامن"></div>
      <div class="field"><label>کد ملی ضامن</label><input id="lngGuarantorId" placeholder="۰۰۷۹۸۴۳۲۱۱"></div>
      <div class="field"><label>نسبت ضامن</label><input id="lngGuarantorRel" placeholder="برادر / همسر"></div>
      <div class="field"><label>متممِ وام (شناسهٔ وام پایه) <span class="muted" id="lngSuppHint">${first.isSupplement ? '— برای این نوع الزامی است' : '— اختیاری'}</span></label><input id="lngSupplement" placeholder="LOAN-0001"></div>
      <div class="field" style="grid-column:1/-1"><label>هدف وام</label><input id="lngPurpose" placeholder="ودیعهٔ مسکن…"></div>
      <div class="field" style="grid-column:1/-1"><label>کد کارکنان مقصد (با کاما؛ خالی = همهٔ کارکنان فعال سازمان)</label><input id="lngEmps" placeholder="EMP-005, EMP-010"></div>
      <div class="field" style="grid-column:1/-1"><label>شرح / مرجع</label><input id="lngNote" placeholder="مصوبهٔ صندوق…"></div>
    </div>
    <div class="row-actions" style="margin-top:8px">
      <button class="btn btn-primary" onclick="loanGrantPreview()">🔎 پیش‌نمایش تخصیص وام</button>
      <button class="btn btn-ghost" onclick="loanGrantBuild()">🧾 ثبت تخصیص وام</button>
      <button class="btn btn-ghost" onclick="loanGrantClear()">↩️ پاک‌کردن</button>
    </div>
    <div id="lngResult" style="margin-top:12px"></div>
  </div>`;
}

/** واحدهای سازمانی آخرین بارگذاری‌شدهٔ فرم تخصیص وام (برای فیلترکردن بر پایهٔ مدیریت). */
var LOAN_GRANT_UNITS = [];

/** گزینه‌های واحد سازمانی؛ اگر مدیریتی انتخاب شده باشد، فقط واحدهای همان مدیریت. */
function loanGrantUnitOptions(management) {
  const list = LOAN_GRANT_UNITS.filter(u => !management || !u.parentKey || u.parentKey === management);
  return '<option value="">— بدون واحد مشخص —</option>' +
    list.map(u => `<option value="${esc(u.key)}">${esc(u.titleFa)}</option>`).join('');
}

/** با تغییر مدیریت متقاضی، فهرست واحدها به واحدهای زیر همان مدیریت محدود می‌شود. */
function loanGrantSyncUnits() {
  const sel = $('#lngUnit');
  if (!sel) return;
  sel.innerHTML = loanGrantUnitOptions(($('#lngMgmt') || {}).value || '');
}

/** با تغییر نوع وام، مبلغ و اقساط پیشنهادی و الزام «متممِ وام» را از همان گزینهٔ سرور اعمال می‌کند. */
function loanGrantSyncType() {
  const sel = $('#lngType');
  if (!sel) return;
  const opt = sel.options[sel.selectedIndex];
  const amount = $('#lngAmount');
  if (amount && !amount.value) amount.value = opt ? (opt.dataset.maxAmount || '') : '';
  const inst = $('#lngInstallments');
  if (inst && !inst.value) inst.value = opt ? (opt.dataset.maxInstallments || '') : '';
  const hint = $('#lngInstHint');
  if (hint) hint.textContent = opt && opt.dataset.maxInstallments ? '— سقف مصوب ' + fa(opt.dataset.maxInstallments) : '';
  const suppHint = $('#lngSuppHint');
  if (suppHint) suppHint.textContent = opt && opt.dataset.supplement === '1' ? '— برای این نوع الزامی است' : '— اختیاری';
}

/** درخواست تخصیص از روی فرم پنل. */
function loanGrantRequest() {
  return {
    loanType: $('#lngType').value,
    bank: $('#lngBank').value,
    management: $('#lngMgmt').value,
    unit: ($('#lngUnit').value || '').trim(),
    amount: Number($('#lngAmount').value) || 0,
    installments: Number($('#lngInstallments').value) || 0,
    receivedDate: ($('#lngDate').value || '').trim(),
    purpose: ($('#lngPurpose').value || '').trim(),
    guarantorName: ($('#lngGuarantor').value || '').trim(),
    guarantorNationalId: ($('#lngGuarantorId').value || '').trim(),
    guarantorRelation: ($('#lngGuarantorRel').value || '').trim(),
    supplementOf: ($('#lngSupplement').value || '').trim(),
    noteFa: ($('#lngNote').value || '').trim(),
    employeeIds: ($('#lngEmps').value || '').split(',').map(s => s.trim()).filter(Boolean),
  };
}

function loanGrantPreview() { loanGrantRun(true); }
function loanGrantBuild() { loanGrantRun(false); }

/** اجرای پیش‌نمایش یا ثبت تخصیص؛ هر دو مسیر همان قواعد و همان محاسبهٔ سرور را می‌بینند. */
async function loanGrantRun(trial) {
  const box = $('#lngResult');
  if (!box) return;
  box.innerHTML = '<div class="muted">در حال محاسبه…</div>';
  const path = trial ? '/api/v1/loans/allocate/preview' : '/api/v1/loans/allocate';
  const res = await apiFetch(path, {method: 'POST', body: loanGrantRequest()});
  if (!res.ok) {
    box.innerHTML = `<div class="empty-state"><div class="big">⚠️</div>${esc(res.message)}</div>`;
    toast(res.message, 'error');
    return;
  }
  if (!trial) {
    toast('تخصیص وام ثبت شد — ' + fa(res.data.created || 0) + ' رکورد درخواست وام');
    liveReload();
    loanGrantRenderAfterReload(res.data);
    return;
  }
  loanGrantRenderResult(res.data, trial);
}

/** پس از تازه‌سازی فهرست، نتیجهٔ تخصیص ثبت‌شده را در پنل تازه می‌نشاند. */
async function loanGrantRenderAfterReload(data) {
  for (let i = 0; i < 40; i++) {
    await new Promise(r => setTimeout(r, 150));
    if (document.getElementById('lngResult') && document.getElementById('lngType')) {
      loanGrantRenderResult(data, false);
      return;
    }
  }
}

/** نتیجهٔ تخصیص: جمع مبلغ، جمع کارمزد و ردیف هر نفر با مبلغ قسط. */
function loanGrantRenderResult(r, trial) {
  const box = $('#lngResult');
  if (!box) return;
  const head = ['کارمند', 'نوع وام', 'بانک', 'مدیریت', 'واحد سازمانی', 'مبلغ وام (ریال)', 'کارمزد', 'اقساط', 'مبلغ هر قسط', 'شناسهٔ رکورد'];
  const rows = (r.rows || []).map(x => `<tr>
    <td>${personCell(x.employeeId, x.employeeName)}</td>
    <td>${esc(x.loanTypeFa)}</td>
    <td>${esc(x.bankFa || '—')}</td>
    <td>${esc(x.managementFa || '—')}</td>
    <td>${esc(x.unitFa || '—')}</td>
    <td class="num">${faNum(money(x.amount))}</td>
    <td class="num">${faNum(money(x.feeAmount || 0))} <span class="muted">(${fa(x.feeRatePercent || 0)}٪)</span></td>
    <td class="num">${fa(x.installments)}</td>
    <td class="num">${faNum(money(x.installmentAmount || 0))}</td>
    <td class="num ltr">${esc(x.recordId || '—')}</td></tr>`);
  box.innerHTML = `<div class="muted" style="margin-bottom:8px">${trial ? 'پیش‌نمایش' : 'تخصیص ثبت‌شده'}
      — ${fa(r.count || 0)} نفر — جمع وام ${faNum(money(r.totalAmount || 0))} ریال — جمع کارمزد ${faNum(money(r.totalFee || 0))} ریال${r.created ? ' — ' + fa(r.created) + ' رکورد ساخته شد' : ''}
      <br>${esc(r.noteFa || '')}</div>` +
    liveTable(head, rows, (r.rows || []).length);
}

function loanGrantClear() {
  const box = $('#lngResult');
  if (box) box.innerHTML = '';
}

/* =====================================================================
   داشبورد گرافیکی وام
   =====================================================================

   فقط نمایش: ترکیب انواع وام، بانک‌های عامل، روند دریافت و وضعیت درخواست‌ها را از همان
   رکوردهایی می‌سازد که جدول پایین صفحه نشان می‌دهد. سقف و نرخ از دامنه (options) می‌آید.
   ================================================================================== */
function loansChartHTML(items, options, kind) {
  if (!items || !items.length) return '';
  const num = v => Number(v) || 0;
  const typeTitle = k => { const t = ((options || {}).loanTypes || []).find(x => x.key === k); return t ? t.titleFa : (k || 'نامشخص'); };
  const bankTitle = k => { const b = ((options || {}).banks || []).find(x => x.key === k); return b ? b.titleFa : (k || 'بدون بانک'); };

  let sumAmount = 0, sumFee = 0, sumInstallment = 0;
  const byType = {}, byBank = {}, byYear = {}, byStatus = {};
  for (const r of items) {
    const f = r.fields || {};
    const amount = num(f.amount);
    sumAmount += amount;
    sumFee += num(f.feeAmount) || amount * num(f.feeRate) / 100;
    sumInstallment += num(f.installmentAmount);
    const type = f.loanType || 'نامشخص';
    byType[type] = byType[type] || {count: 0, amount: 0};
    byType[type].count++; byType[type].amount += amount;
    const bank = f.bank || '';
    if (bank) { byBank[bank] = byBank[bank] || {count: 0, amount: 0}; byBank[bank].count++; byBank[bank].amount += amount; }
    const year = String(f.receivedDate || '').trim().slice(0, 4) || 'نامشخص';
    byYear[year] = byYear[year] || {count: 0, amount: 0};
    byYear[year].count++; byYear[year].amount += amount;
    byStatus[r.status] = (byStatus[r.status] || 0) + 1;
  }

  const pct = (part, total) => total > 0 ? Math.round(part / total * 1000) / 10 : 0;
  const groupBars = (groups, color) => {
    const keys = Object.keys(groups).sort((a, b) => groups[b].amount - groups[a].amount);
    return hBarChart(keys.map(k => ({
      label: k, value: groups[k].amount, pct: pct(groups[k].amount, sumAmount), color,
      text: money(groups[k].amount) + ' (' + fa(groups[k].count) + ')',
    })), 'داده‌ای برای نمایش نیست');
  };
  const years = {};
  for (const [y, g] of Object.entries(byYear)) years[y] = {count: g.count, amount: g.amount};

  const stats = `<div class="grid-stats">
    ${statCard({ico: '🏦', tint: '#EFF6FF', accent: '#2563EB', val: money(sumAmount), label: 'مجموع مبلغ وام‌ها (ریال)'})}
    ${statCard({ico: '💸', tint: '#FEF3C7', accent: '#D97706', val: money(sumFee), label: 'مجموع کارمزد (ریال)'})}
    ${statCard({ico: '🧾', tint: '#EDE9FE', accent: '#7C3AED', val: money(sumInstallment), label: 'جمع اقساط ماهانه (ریال)'})}
    ${statCard({ico: '📊', tint: '#DCFCE7', accent: '#16A34A', val: money(Math.round(sumAmount / items.length)), label: 'میانگین مبلغ هر وام (ریال)'})}
  </div>`;

  const statusChips = (kind && kind.statuses ? kind.statuses : []).map(s =>
    `<span class="chip ${s.value === 'paid' || s.value === 'settled' ? 'c-ok' : 'c-wait'}">${esc(s.labelFa)}
      <span class="num">${fa(byStatus[s.value] || 0)}</span></span>`).join(' ');

  const typeTitles = byType;
  const typeLabeled = {};
  for (const [k, g] of Object.entries(typeTitles)) typeLabeled[typeTitle(k)] = g;
  const bankLabeled = {};
  for (const [k, g] of Object.entries(byBank)) bankLabeled[bankTitle(k)] = g;

  return stats + `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📊 ترکیب انواع وام بر پایهٔ مبلغ (تعداد در پرانتز)</div>
    ${groupBars(typeLabeled, '#2563EB')}
  </div>
  <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:12px;margin-bottom:12px">
    <div class="card card-pad">
      <div class="card-title">🏛️ بانک‌های عامل بر پایهٔ مبلغ</div>
      ${Object.keys(bankLabeled).length ? groupBars(bankLabeled, '#0EA5E9') : '<div class="muted" style="font-size:.8rem">بانکی برای وام‌های ثبت‌شده تعیین نشده است.</div>'}
    </div>
    <div class="card card-pad">
      <div class="card-title">📅 روند دریافت وام به تفکیک سال</div>
      ${groupBars(years, '#7C3AED')}
    </div>
  </div>
  <div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧭 وضعیت درخواست‌ها</div>
    <div style="display:flex;gap:6px;flex-wrap:wrap">${statusChips || '<span class="muted">—</span>'}</div>
  </div>`;
}

/* =====================================================================
   داشبورد تخصیص سهمیهٔ تسهیلات
   =====================================================================

   همهٔ برش‌ها از /api/v1/loans/quota-dashboard می‌آید: سهمیهٔ مصوب هر مدیریت بندر امام،
   مشمولین و متقاضیان، مرحلهٔ ارسال به مدیریت مالی، سهم شرکتی/ارکان ثالث و طبقهٔ نرخ
   کارمزد تسهیلات. این لایه فقط شکل می‌دهد و هیچ درصدی را خودش حساب نمی‌کند.
   ================================================================== */
function loansQuotaHTML(d) {
  if (!d || !(d.units || []).length) return '';
  // برچسب نوارها در hBarChart با esc نمایش داده می‌شود، پس اینجا فقط ارقام فارسی
  // ساده لازم است (faNum برچسب "num" HTML می‌سازد که در برچسب خام دیده می‌شود).
  const pctText = v => fa(Number(v) || 0) + '٪';
  const units = d.units;
  const byQuota = units.slice().sort((a, b) => (b.quotaAmount || 0) - (a.quotaAmount || 0));

  const stats = `<div class="grid-stats">
    ${statCard({ico: '🏛️', tint: '#EFF6FF', accent: '#2563EB', val: money(d.quotaTotalAmount || 0), label: 'کل صندوق تسهیلات (ریال)'})}
    ${statCard({ico: '👥', tint: '#F1F5F9', accent: '#475569', val: fa(d.eligibleTotal || 0), label: 'مشمولین (سرشماری مصوب)'})}
    ${statCard({ico: '📝', tint: '#EDE9FE', accent: '#7C3AED', val: fa(d.applicantTotal || 0), label: 'متقاضیان'})}
    ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa(d.pendingTotal || 0), label: 'در انتظار بررسی واحد وام'})}
    ${statCard({ico: '📤', tint: '#DCFCE7', accent: '#16A34A', val: fa(d.sentTotal || 0), label: 'ارسالی به مدیریت مالی'})}
  </div>`;

  // ۱) تخصیص سهمیه بر پایهٔ مدیریت‌های بندر امام (درصدها سهمیه، متن ردیف درصد تعلق).
  const quotaBars = hBarChart(byQuota.map(u => ({
    label: u.unitFa, value: u.quotaAmount, pct: u.sharePercent, color: '#2563EB',
    text: money(u.quotaAmount) + ' · ' + pctText(u.sharePercent) + ' · تعلق ' + pctText(u.claimPercent),
  })), 'سهمیه‌ای برای مدیریت‌ها تعریف نشده است');

  // ۲) مشمولین در برابر متقاضیان و متقاضیان در انتظار.
  const eligibleBars = hBarChart(byQuota.map(u => ({
    label: u.unitFa, value: u.eligible, pct: 100, color: '#64748B',
    text: fa(u.eligible) + ' نفر · پوشش ' + pctText(u.coveragePercent),
  })), 'سرشماری مشمولین خالی است');
  const applicantBars = hBarChart(byQuota.map(u => ({
    label: u.unitFa, value: u.applicants, pct: u.eligible ? Math.round(u.applicants / u.eligible * 1000) / 10 : 0, color: '#7C3AED',
    text: fa(u.applicants) + ' متقاضی از ' + fa(u.eligible) + ' مشمول',
  })), 'متقاضی ثبت نشده است');
  const pendingBars = hBarChart(byQuota.map(u => ({
    label: u.unitFa, value: u.pending, pct: u.recordsCount ? Math.round(u.pending / u.recordsCount * 1000) / 10 : 0, color: '#D97706',
    text: fa(u.pending) + ' در انتظار از ' + fa(u.recordsCount) + ' درخواست',
  })), 'درخواست در انتظاری نیست');

  // ۳) جدول ریز تخصیص هر مدیریت.
  const unitRows = byQuota.map(u => `<tr>
    <td>${esc(u.unitFa)}</td>
    <td class="num">${pctText(u.sharePercent)}</td>
    <td class="num">${faNum(money(u.quotaAmount))}</td>
    <td class="num">${fa(u.eligible)}</td>
    <td class="num">${fa(u.applicants)}</td>
    <td class="num">${fa(u.pending)}</td>
    <td class="num">${fa(u.sent)}</td>
    <td class="num">${u.rejected ? `<span class="chip c-wait">${fa(u.rejected)}</span>` : '—'}</td>
    <td class="num">${faNum(money(u.requestedAmount))}</td>
    <td class="num">${pctText(u.claimPercent)}</td>
    <td class="num">${pctText(u.coveragePercent)}</td>
    <td style="font-size:.74rem"><span class="chip">شرکتی ${fa(u.companyCount)}</span> <span class="chip">ارکان ثالث ${fa(u.thirdPartyCount)}</span></td></tr>`);
  const unitTable = liveTable(
    ['مدیریت', 'سهمیه', 'سهمیهٔ ریالی', 'مشمولین', 'متقاضیان', 'در انتظار', 'ارسالی به مالی', 'ردشده', 'مبلغ درخواستی', 'درصد تعلق', 'پوشش مشمولین', 'ترکیب کارگزینی'],
    unitRows, unitRows.length);

  // ۴) مرحلهٔ ارسال به مدیریت مالی.
  const stages = d.stages || [];
  const stageBars = hBarChart(stages.filter(s => s.count).map(s => ({
    label: s.stageFa, value: s.amount, pct: d.requestedTotalAmount ? Math.round(s.amount / d.requestedTotalAmount * 1000) / 10 : 0,
    color: s.sentToFinance ? '#16A34A' : (s.stageKey === 'rejected' ? '#DC2626' : '#0EA5E9'),
    text: fa(s.count) + ' درخواست · ' + money(s.amount),
  })), 'درخواستی در گردش نیست');
  const stageChips = stages.map(s => `<span class="chip ${s.sentToFinance ? 'c-ok' : (s.stageKey === 'rejected' ? 'c-wait' : '')}">${esc(s.stageFa)}
    <span class="num">${fa(s.count)}</span></span>`).join(' ');

  // ۵) درصد سهمیهٔ ارسالی به تفکیک شرکتی و ارکان ثالث.
  const groups = d.groups || [];
  const groupBars = hBarChart(groups.map(g => ({
    label: g.groupFa, value: g.sentAmount, pct: g.sentSharePercent, color: g.groupKey === 'thirdParty' ? '#D97706' : '#2563EB',
    text: 'ارسالی ' + pctText(g.sentSharePercent) + ' · ' + money(g.sentAmount) + ' (' + fa(g.sentCount) + ' پرونده)',
  })), 'پروندهٔ ارسالی به مالی ثبت نشده است');
  const groupChips = groups.map(g => `<span class="chip">${esc(g.groupFa)}
    <span class="num">${fa(g.count)} درخواست · ${pctText(g.sharePercent)} از مبلغ</span></span>`).join(' ');

  // ۶) میزان درخواست‌ها به تفکیک درصد کارمزد تسهیلات.
  const tiers = d.feeTiers || [];
  const tierBars = hBarChart(tiers.map(t => ({
    label: 'کارمزد ' + pctText(t.feeRatePercent), value: t.amount,
    pct: d.requestedTotalAmount ? Math.round(t.amount / d.requestedTotalAmount * 1000) / 10 : 0, color: '#0EA5E9',
    text: fa(t.count) + ' درخواست · ' + money(t.amount),
  })), 'نرخ کارمزدی ثبت نشده است');
  const tierRows = tiers.map(t => `<tr>
    <td class="num">${pctText(t.feeRatePercent)}</td>
    <td class="num">${fa(t.count)}</td>
    <td class="num">${faNum(money(t.amount))}</td>
    <td class="num">${fa(t.companyCount)} <span class="muted">(${faNum(money(t.companyAmount))})</span></td>
    <td class="num">${fa(t.thirdPartyCount)} <span class="muted">(${faNum(money(t.thirdPartyAmount))})</span></td></tr>`);
  const tierTable = liveTable(
    ['درصد کارمزد تسهیلات', 'تعداد درخواست', 'جمع مبلغ (ریال)', 'شرکتی (مبلغ)', 'ارکان ثالث (مبلغ)'],
    tierRows, tierRows.length);

  return stats + `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🏛️ تخصیص سهمیه بر اساس مدیریت‌های بندر امام</div>
    ${quotaBars}
  </div>
  <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:12px;margin-bottom:12px">
    <div class="card card-pad">
      <div class="card-title">👥 مشمولین هر مدیریت (سرشماری مصوب)</div>
      ${eligibleBars}
    </div>
    <div class="card card-pad">
      <div class="card-title">📝 متقاضیان در برابر مشمولین</div>
      ${applicantBars}
    </div>
  </div>
  <div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">⏳ متقاضیان در انتظار (هنوز به مالی نرفته‌اند)</div>
    ${pendingBars}
  </div>
  <div class="card" style="margin-bottom:12px"><div class="card-pad" style="padding-bottom:0">
    <div class="card-title">🧮 ریز تخصیص سهمیه هر مدیریت</div>
  </div>${unitTable}</div>
  <div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📤 وضعیت ارسال به مدیریت مالی</div>
    ${stageBars}
    <div style="display:flex;gap:6px;flex-wrap:wrap;margin-top:12px">${stageChips}</div>
  </div>
  <div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧩 درصد سهمیهٔ ارسالی — شرکتی در برابر ارکان ثالث</div>
    ${groupBars}
    <div style="display:flex;gap:6px;flex-wrap:wrap;margin-top:12px">${groupChips}</div>
  </div>
  <div class="card" style="margin-bottom:12px"><div class="card-pad" style="padding-bottom:0">
    <div class="card-title">٪ میزان درخواست‌ها به تفکیک درصد کارمزد تسهیلات</div>
    ${tierBars}
  </div>${tierTable}</div>` + statNote(d.noteFa);
}

/* =====================================================================
   داشبورد مدیریتی پاداش، فرم تخصیص و گزارش پرداخت
   =====================================================================

   همهٔ داده از سرور می‌آید: /api/v1/bonus/dashboard (پرداخت به تفکیک مدیریت، نفر،
   نوع، دوره و وضعیت)، /api/v1/bonus/options (انواع پاداش، مدیریت‌های بندر امام و
   روش‌های پرداخت) و /api/v1/bonus/allocate[preview]. هیچ نرخ، مبلغ پیشنهادی یا
   برچسبی در این لایه ساخته نمی‌شود؛ «تعداد دفعات پرداخت» و «مبلغ هر نفر در هر
   مدیریت» از همان رکوردهای واقعی پرداخت جمع می‌خورد.
   --------------------------------------------------------------------- */

/** درصد یک مقدار از کل، با یک رقم اعشار (کل صفر یعنی صفر). */
function bonusPct(part, total) {
  return total > 0 ? Math.round((Number(part) || 0) / total * 1000) / 10 : 0;
}

/** بخش نموداری داشبورد پاداش: مبلغ، تعداد دفعات و مبلغ هر نفر به تفکیک مدیریت. */
function bonusMgmtDashboardHTML(d) {
  if (!d || !(d.managementRows || []).length) return '';
  const pctText = v => fa(Number(v) || 0) + '٪';
  const mg = d.managementRows;
  const total = d.totalAmount || 0;
  const maxPayments = mg.reduce((n, u) => Math.max(n, u.payments || 0), 0);
  const byAmount = mg.slice().sort((a, b) => (b.totalAmount || 0) - (a.totalAmount || 0));

  const stats = `<div class="grid-stats">
    ${statCard({ico: '🎁', tint: '#EDE9FE', accent: '#7C3AED', val: money(d.paidAmount || 0), label: 'پاداش پرداخت‌شده (ریال)'})}
    ${statCard({ico: '🧾', tint: '#EFF6FF', accent: '#2563EB', val: fa(d.totalPayments || 0), label: 'تعداد دفعات پرداخت'})}
    ${statCard({ico: '👥', tint: '#F1F5F9', accent: '#475569', val: fa(d.totalBeneficiaries || 0), label: 'نفرات بهره‌مند'})}
    ${statCard({ico: '📊', tint: '#DCFCE7', accent: '#16A34A', val: money(d.averagePerPerson || 0), label: 'میانگین هر نفر (ریال)'})}
    ${statCard({ico: '🏛️', tint: '#FEF3C7', accent: '#D97706', val: fa(d.managementsCovered || 0) + ' / ' + fa(d.managementsTotal || 0), label: 'مدیریت‌های دارای پرداخت'})}
  </div>`;

  // ۱) مبلغ پاداش هر مدیریت (مبنای «مبالغ پرداختی به هر مدیریت»).
  const amountBars = hBarChart(byAmount.map(u => ({
    label: u.managementFa, value: u.totalAmount, pct: bonusPct(u.totalAmount, total), color: '#7C3AED',
    text: money(u.totalAmount) + ' · ' + fa(u.payments) + ' دفعه · ' + fa(u.beneficiaries) + ' نفر',
  })), 'پرداخت پاداشی ثبت نشده است');

  // ۲) تعداد دفعات پرداخت هر مدیریت (شمارش رویدادها، نه نفرات).
  const paymentBars = hBarChart(byAmount.map(u => ({
    label: u.managementFa, value: u.payments, pct: bonusPct(u.payments, maxPayments), color: '#2563EB',
    text: fa(u.payments) + ' دفعه · میانگین هر نفر ' + money(u.averagePerPerson || 0),
  })), 'دفعهٔ پرداختی ثبت نشده است');

  // ۳) ترکیب انواع پاداش و روند دوره‌ای.
  const typeBars = hBarChart((d.typeRows || []).map(t => ({
    label: t.typeFa, value: t.amount, pct: t.sharePercent, color: '#0EA5E9',
    text: money(t.amount) + ' · ' + fa(t.payments) + ' دفعه',
  })), 'نوع پاداشی ثبت نشده است');
  const periodBars = hBarChart((d.periodRows || []).map(p => ({
    label: fa(p.period), value: p.amount, pct: bonusPct(p.amount, total), color: '#16A34A',
    text: money(p.amount) + ' · ' + fa(p.payments) + ' دفعه · ' + fa(p.beneficiaries) + ' نفر',
  })), 'دوره‌ای برای پرداخت ثبت نشده است');

  // ۴) ترکیب شرکتی در برابر ارکان ثالث.
  const splitBars = hBarChart([
    {label: 'شرکتی', value: d.companyAmount || 0, pct: bonusPct(d.companyAmount, total), color: '#2563EB', text: money(d.companyAmount || 0)},
    {label: 'ارکان ثالث', value: d.thirdPartyAmount || 0, pct: bonusPct(d.thirdPartyAmount, total), color: '#D97706', text: money(d.thirdPartyAmount || 0)},
  ], 'دادهٔ ترکیب کارگزینی ثبت نشده است');

  const statusChips = (d.statusRows || []).map(s =>
    `<span class="chip ${s.statusKey === 'paid' ? 'c-ok' : (s.statusKey === 'cancelled' ? 'c-wait' : '')}">${esc(s.statusFa)}
      <span class="num">${fa(s.payments)}</span></span>`).join(' ');

  // ۵) جدول «مبلغ و تعداد دفعات پرداختی به هر نفر در هر مدیریت».
  const rows = (d.employeeRows || []).slice(0, 80).map(e => `<tr>
    <td>${personCell(e.employeeId, e.employeeName)}</td>
    <td>${esc(e.managementFa)}</td>
    <td class="num">${fa(e.payments)}</td>
    <td class="num">${faNum(money(e.totalAmount))}</td>
    <td class="num">${faNum(money(e.averagePayment))}</td>
    <td style="font-size:.76rem">${(e.typeTitlesFa || []).map(esc).join(' · ') || '<span class="muted">—</span>'}</td>
    <td class="num ltr" style="font-size:.76rem">${e.firstDate ? fa(e.firstDate) : '—'} … ${e.lastDate ? fa(e.lastDate) : '—'}</td></tr>`);
  const empTable = liveTable(
    ['کارمند', 'مدیریت', 'دفعات', 'جمع مبلغ (ریال)', 'میانگین هر دفعه', 'انواع پاداش', 'بازهٔ پرداخت'],
    rows, (d.employeeRows || []).length);

  return stats + `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🏛️ مبلغ پاداش به تفکیک مدیریت (سهم، تعداد دفعات و نفرات)</div>
    ${amountBars}
  </div>
  <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:12px;margin-bottom:12px">
    <div class="card card-pad">
      <div class="card-title">🧾 تعداد دفعات پرداخت به تفکیک مدیریت</div>
      ${paymentBars}
    </div>
    <div class="card card-pad">
      <div class="card-title">🎯 ترکیب انواع پاداش بر پایهٔ مبلغ</div>
      ${typeBars}
    </div>
  </div>
  <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:12px;margin-bottom:12px">
    <div class="card card-pad">
      <div class="card-title">📅 روند پرداخت به تفکیک دوره</div>
      ${periodBars}
    </div>
    <div class="card card-pad">
      <div class="card-title">🧩 سهم شرکتی در برابر ارکان ثالث</div>
      ${splitBars}
    </div>
  </div>
  <div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧭 وضعیت کارتابل پرداخت‌ها</div>
    <div style="display:flex;gap:6px;flex-wrap:wrap">${statusChips || '<span class="muted">—</span>'}</div>
  </div>
  <div class="card" style="margin-bottom:12px"><div class="card-pad" style="padding-bottom:0">
    <div class="card-title">👤 مبلغ و تعداد دفعات پرداختی به هر نفر در هر مدیریت</div>
  </div>${empTable}</div>
  ${statNote('دورهٔ گزارش: ' + fa(d.managementsCovered || 0) + ' مدیریت دارای پرداخت از ' + fa(d.managementsTotal || 0) + ' مدیریت — ' + pctText(bonusPct(d.paidAmount, total)) + ' از مبلغ کل پرداخت شده است.')}`;
}

/** گزارش‌های پاداش: تفکیک مدیریت، نوع، دوره و نفرات — بر پایهٔ همان داشبورد سرور. */
function bonusMgmtReportHTML(d) {
  if (!d || !(d.managementRows || []).length) return '';
  const pctText = v => fa(Number(v) || 0) + '٪';
  const total = d.totalAmount || 0;

  const mgRows = (d.managementRows || []).map(u => `<tr>
    <td>${esc(u.managementFa)}</td>
    <td class="num">${fa(u.payments)}</td>
    <td class="num">${fa(u.beneficiaries)}</td>
    <td class="num">${faNum(money(u.totalAmount))}</td>
    <td class="num">${pctText(bonusPct(u.totalAmount, total))}</td>
    <td class="num">${faNum(money(u.averagePerPerson || 0))}</td>
    <td class="num">${faNum(money(u.maxPayment || 0))}</td>
    <td class="num ltr" style="font-size:.76rem">${u.firstDate ? fa(u.firstDate) : '—'} … ${u.lastDate ? fa(u.lastDate) : '—'}</td></tr>`);
  const mgTable = liveTable(
    ['مدیریت', 'دفعات', 'نفرات', 'جمع مبلغ (ریال)', 'سهم', 'میانگین هر نفر', 'بیشترین پرداخت', 'بازهٔ پرداخت'],
    mgRows, mgRows.length);

  const typeRows = (d.typeRows || []).map(t => `<tr>
    <td>${esc(t.typeFa)}</td>
    <td class="num">${fa(t.payments)}</td>
    <td class="num">${faNum(money(t.amount))}</td>
    <td class="num">${pctText(t.sharePercent)}</td></tr>`);
  const typeTable = liveTable(['نوع پاداش', 'دفعات', 'جمع مبلغ (ریال)', 'سهم'], typeRows, typeRows.length);

  const periodRows = (d.periodRows || []).map(p => `<tr>
    <td class="num ltr">${p.period ? fa(p.period) : '—'}</td>
    <td class="num">${fa(p.payments)}</td>
    <td class="num">${fa(p.beneficiaries)}</td>
    <td class="num">${faNum(money(p.amount))}</td></tr>`);
  const periodTable = liveTable(['دوره', 'دفعات', 'نفرات', 'جمع مبلغ (ریال)'], periodRows, periodRows.length);

  const empRows = (d.employeeRows || []).slice(0, 120).map(e => `<tr>
    <td>${personCell(e.employeeId, e.employeeName)}</td>
    <td>${esc(e.managementFa)}</td>
    <td class="num">${fa(e.payments)}</td>
    <td class="num">${faNum(money(e.totalAmount))}</td>
    <td class="num">${faNum(money(e.averagePayment))}</td></tr>`);
  const empTable = liveTable(['کارمند', 'مدیریت', 'دفعات', 'جمع مبلغ (ریال)', 'میانگین هر دفعه'], empRows, (d.employeeRows || []).length);

  const stat = `<div class="grid-stats">
    ${statCard({ico: '🎁', tint: '#EDE9FE', accent: '#7C3AED', val: money(d.paidAmount || 0), label: 'پرداخت‌شده (ریال)'})}
    ${statCard({ico: '🧾', tint: '#EFF6FF', accent: '#2563EB', val: fa(d.totalPayments || 0), label: 'کل دفعات پرداخت'})}
    ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: fa((d.approvedPayments || 0) + (d.draftPayments || 0)), label: 'در گردش (تأییدشده و پیش‌نویس)'})}
    ${statCard({ico: '🚫', tint: '#FEE2E2', accent: '#DC2626', val: fa(d.cancelledPayments || 0), label: 'لغوشده'})}
  </div>`;

  return stat + `<div class="card" style="margin-bottom:12px"><div class="card-pad" style="padding-bottom:0">
      <div class="card-title">📋 گزارش پرداخت به تفکیک مدیریت</div>
    </div>${mgTable}</div>
  <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:12px;margin-bottom:12px">
    <div class="card"><div class="card-pad" style="padding-bottom:0">
      <div class="card-title">🎯 به تفکیک نوع پاداش</div>
    </div>${typeTable}</div>
    <div class="card"><div class="card-pad" style="padding-bottom:0">
      <div class="card-title">📅 به تفکیک دورهٔ پرداخت</div>
    </div>${periodTable}</div>
  </div>
  <div class="card" style="margin-bottom:12px"><div class="card-pad" style="padding-bottom:0">
      <div class="card-title">👤 مبالغ پرداختی به هر نفر در هر مدیریت</div>
    </div>${empTable}</div>
  <div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
    📤 برای گزارش دلخواه (انتخاب ستون، فیلتر مدیریت/نوع/روش پرداخت/وضعیت، شرط‌های عددی) و خروجی CSV/چاپ،
    از «گزارش‌ساز بر پایهٔ اقلام» با منبع «پاداش کارکنان» استفاده کنید.
    <div style="margin-top:8px"><button class="btn btn-ghost" onclick="renderPage('reportBuilder')">→ رفتن به گزارش‌ساز</button></div>
  </div>`;
}

/** فرم تخصیص گروهی پاداش؛ گزینه‌ها (نوع، مدیریت، روش پرداخت، امروز) از /options سرور. */
function bonusAllocPanelHTML(options) {
  if (!options) return '';
  const typeOpts = (options.types || []).map(t =>
    // داخل <option> فقط متن ساده مجاز است؛ faNum یک <span> می‌سازد، پس اینجا از fa استفاده می‌شود.
    `<option value="${esc(t.key)}" data-default="${esc(String(t.defaultAmount || 0))}" data-decree="${t.requiresDecree ? '1' : '0'}">${esc(t.titleFa)}${t.defaultAmount ? ' — پیشنهادی ' + fa(money(t.defaultAmount)) + ' ریال' : ''}</option>`).join('');
  const mgmtOpts = (options.managements || []).map(m =>
    `<option value="${esc(m.key)}">${esc(m.titleFa)}</option>`).join('');
  const methodOpts = (options.paymentMethods || []).map(m =>
    `<option value="${esc(m.key)}">${esc(m.titleFa)}</option>`).join('');

  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🎁 فرم تخصیص پاداش (گروهی به یک مدیریت یا تک‌کارمند)</div>
    <div class="grid-3" style="gap:10px">
      <div class="field"><label>نوع پاداش</label><select id="bnpType" onchange="bonusAllocSyncType()">${typeOpts}</select></div>
      <div class="field"><label>مدیریت مقصد</label><select id="bnpMgmt">${mgmtOpts}</select></div>
      <div class="field"><label>مبلغ هر نفر (ریال)</label><input id="bnpAmount" type="number" min="0"></div>
      <div class="field"><label>تاریخ پرداخت (YYYY/MM/DD)</label><input id="bnpDate" value="${esc(options.todayFa || '')}" placeholder="1404/07/01"></div>
      <div class="field"><label>روش پرداخت</label><select id="bnpMethod">${methodOpts}</select></div>
      <div class="field"><label>شمارهٔ مصوبه / حکم <span class="muted" id="bnpDecreeHint"></span></label><input id="bnpDecree" placeholder="1404/م/0151"></div>
      <div class="field" style="grid-column:1/-1"><label>کد کارکنان مقصد (با کاما؛ خالی = همهٔ کارکنان فعال سازمان)</label><input id="bnpEmps" placeholder="EMP-005, EMP-010, EMP-012"></div>
      <div class="field" style="grid-column:1/-1"><label>شرح / مرجع</label><input id="bnpRef" placeholder="پاداش مناسبتی…"></div>
    </div>
    <div class="row-actions" style="margin-top:8px">
      <button class="btn btn-primary" onclick="bonusAllocPreview()">🔎 پیش‌نمایش تخصیص</button>
      <button class="btn btn-ghost" onclick="bonusAllocBuild()">🧾 ثبت تخصیص گروهی</button>
      <button class="btn btn-ghost" onclick="bonusAllocClearResult()">↩️ پاک‌کردن</button>
    </div>
    <div id="bnpResult" style="margin-top:12px"></div>
  </div>`;
}

/** با تغییر نوع پاداش، مبلغ پیشنهادی و الزام شمارهٔ مصوبه را از همان گزینهٔ سرور اعمال می‌کند. */
function bonusAllocSyncType() {
  const sel = $('#bnpType');
  if (!sel) return;
  const opt = sel.options[sel.selectedIndex];
  const amount = $('#bnpAmount');
  if (amount && !amount.value) amount.value = opt ? (opt.dataset.default || '') : '';
  const hint = $('#bnpDecreeHint');
  if (hint) hint.textContent = opt && opt.dataset.decree === '1' ? '— برای این نوع الزامی است' : '— اختیاری';
}

/** درخواست تخصیص از روی فرم پنل. */
function bonusAllocRequest() {
  return {
    bonusType: $('#bnpType').value,
    management: $('#bnpMgmt').value,
    amount: Number($('#bnpAmount').value) || 0,
    paymentDate: ($('#bnpDate').value || '').trim(),
    paymentMethod: $('#bnpMethod').value,
    decreeNo: ($('#bnpDecree').value || '').trim(),
    reference: ($('#bnpRef').value || '').trim(),
    employeeIds: ($('#bnpEmps').value || '').split(',').map(s => s.trim()).filter(Boolean),
  };
}

function bonusAllocPreview() { bonusAllocRun(true); }
function bonusAllocBuild() { bonusAllocRun(false); }

/** اجرای پیش‌نمایش یا ثبت تخصیص؛ هر دو مسیر همان قاعدهٔ دامنهٔ سرور را می‌بینند. */
async function bonusAllocRun(trial) {
  const box = $('#bnpResult');
  if (!box) return;
  box.innerHTML = '<div class="muted">در حال محاسبه…</div>';
  const path = trial ? '/api/v1/bonus/allocate/preview' : '/api/v1/bonus/allocate';
  const res = await apiFetch(path, {method: 'POST', body: bonusAllocRequest()});
  if (!res.ok) {
    box.innerHTML = `<div class="empty-state"><div class="big">⚠️</div>${esc(res.message)}</div>`;
    toast(res.message, 'error');
    return;
  }
  if (!trial) {
    toast('تخصیص ثبت شد — ' + fa(res.data.created || 0) + ' رکورد پرداخت در وضعیت پیش‌نویس');
    liveReload();
    bonusAllocRenderAfterReload(res.data);
    return;
  }
  bonusAllocRenderResult(res.data, trial);
}

/** پس از تازه‌سازی فهرست، نتیجهٔ تخصیص ثبت‌شده را در پنل تازه می‌نشاند. */
async function bonusAllocRenderAfterReload(data) {
  for (let i = 0; i < 40; i++) {
    await new Promise(r => setTimeout(r, 150));
    if (document.getElementById('bnpResult') && document.getElementById('bnpType')) {
      bonusAllocRenderResult(data, false);
      return;
    }
  }
}

/** نتیجهٔ تخصیص: جمع، تعداد و ردیف هر نفر با شمارهٔ دفعه. */
function bonusAllocRenderResult(r, trial) {
  const box = $('#bnpResult');
  if (!box) return;
  const head = ['کارمند', 'مدیریت', 'نوع پاداش', 'مبلغ (ریال)', 'شمارهٔ دفعه', 'شناسهٔ رکورد'];
  const rows = (r.rows || []).map(x => `<tr>
    <td>${personCell(x.employeeId, x.employeeName)}</td>
    <td>${esc(x.managementFa)}</td>
    <td>${esc(x.bonusTypeFa)}</td>
    <td class="num">${faNum(money(x.amount))}</td>
    <td class="num">${fa(x.sequence)}</td>
    <td class="num ltr">${esc(x.recordId || '—')}</td></tr>`);
  box.innerHTML = `<div class="muted" style="margin-bottom:8px">${trial ? 'پیش‌نمایش' : 'تخصیص ثبت‌شده'}
      — ${fa(r.count || 0)} نفر — جمع ${faNum(money(r.totalAmount || 0))} ریال${r.created ? ' — ' + fa(r.created) + ' رکورد ساخته شد' : ''}
      <br>${esc(r.noteFa || '')}</div>` +
    liveTable(head, rows, (r.rows || []).length);
}

function bonusAllocClearResult() {
  const box = $('#bnpResult');
  if (box) box.innerHTML = '';
}

/* =====================================================================
   پنل کاتالوگ‌های مرجع (مدیریت‌ها، بانک‌ها و واحدهای سازمانی)
   =====================================================================

   چرا اینجا: مرجع مدیریت‌ها، بانک‌های عامل و واحدهای سازمانی در ماژول «تنظیمات»
   نگه‌داری می‌شود و فرم‌های تخصیص وام و پاداش و اعتبارسنجی سرور همان فهرست را
   می‌خوانند. پس افزودن یا ویرایش یک قلم در همین پنل، بی‌درنگ در همهٔ فرم‌ها دیده
   می‌شود. هیچ فهرستی در این لایه هاردکد نیست: برچسب، ترتیب فیلد و گردش‌کار از
   شناسهٔ (schema) سرور می‌آید و فهرست‌ها از رکوردهای همان ماژول.
   --------------------------------------------------------------------- */

/** کاتالوگ‌های مرجع؛ کلید باید همان مقدار فیلد `kind` در ماژول تنظیمات باشد. */
const CATALOG_KINDS = [
  {key: 'managements', titleFa: 'مدیریت‌های سازمانی',
   hint: 'مدیریت‌هایی که در فرم‌های وام و پاداش به‌عنوان مدیریت انتخاب و مبنای سهمیه می‌شوند.'},
  {key: 'banks', titleFa: 'بانک‌های عامل',
   hint: 'بانک‌های پرداخت‌کنندهٔ وام؛ خروجی، جدول سهمیه و فرم تخصیص وام این فهرست را می‌بینند.'},
  {key: 'units', titleFa: 'واحدهای سازمانی',
   hint: 'واحدهای زیر هر مدیریت؛ برای این‌که بدانیم وام به کدام واحد رفته است و سهمیهٔ ریزتر بستن.'},
];

/** کلید صفحهٔ یک کاتالوگ در RK_STATE؛ فرم و اقدام‌ها از همین کلید سرور را صدا می‌زنند. */
const catalogPage = kindKey => 'catalogs:' + kindKey;

/** اتصال یک نوع رکورد کاتالوگ به وضعیت صفحه تا فرم عمومی recordkit کار کند. */
function catalogBind(kind) {
  const mod = rkFindModule('catalogs');
  if (!mod) return null;
  RK_STATE[catalogPage(kind.key)] = {mod, kind};
  return RK_STATE[catalogPage(kind.key)];
}

/** جدول یک کاتالوگ: عنوان، کلید ماشین‌خوان، فیلدهای ویژه، وضعیت و عملیات. */
function catalogCardHTML(kind, items, prefix) {
  const extra = (kind.fields || []).filter(f => f.key !== 'key');
  const head = ['عنوان فارسی', 'کلید ماشین‌خوان']
    .concat(extra.map(f => f.labelFa)).concat(['وضعیت', 'عملیات']);
  const rows = items.map(r => {
    const f = r.fields || {};
    const editable = (kind.editableStatuses || [kind.initialStatus]).includes(r.status);
    const legal = (kind.actions || []).filter(a => !a.to || ((kind.transitions || {})[r.status] || []).includes(a.to));
    const actBtns = legal.map(a =>
      `<button class="btn btn-sm ${a.style === 'primary' ? 'btn-primary' : 'btn-ghost'}" onclick="rkAct('${esc(prefix)}','${esc(r.id)}','${esc(a.value)}',${a.noteRequired ? 'true' : 'false'},'${esc(a.labelFa)}')">${esc(a.labelFa)}</button>`).join('');
    const editBtn = editable
      ? `<button class="btn btn-sm btn-ghost" title="ویرایش" onclick="rkEdit('${esc(catalogPage(kind.key))}','${esc(r.id)}')">✏️</button>
         <button class="btn btn-sm btn-ghost" title="حذف" onclick="rkDelete('${esc(prefix)}','${esc(r.id)}')">🗑️</button>`
      : '';
    const cells = [`<span>${esc(r.title)}</span>`, `<span class="num ltr">${esc(f.key || '')}</span>`]
      .concat(extra.map(x => rkValue(x, f[x.key])))
      .concat([statusChip(r.status, r.statusFa)]);
    return `<tr>${cells.map(c => `<td>${c}</td>`).join('')}<td><div class="row-actions">${actBtns}${editBtn}</div></td></tr>`;
  });
  return liveTable(head, rows, items.length);
}

/** صفحهٔ پنل کاتالوگ‌های مرجع: سه کارت (مدیریت/بانک/واحد) با فرم افزودن و ویرایش. */
function pageCatalogsLive() {
  if (!isLive()) return offlineFallback('catalogs');
  const mod = rkFindModule('catalogs');
  if (!mod) return liveErrorBox('این صفحه به ماژول زنده وصل نیست.');

  liveLoad(async () => {
    const schemaRes = await apiFetch(mod.prefix + '/schema');
    if (!schemaRes.ok) return liveErrorBox(schemaRes.message);
    const schema = schemaRes.data || {};

    const fetched = await Promise.all(CATALOG_KINDS.map(c =>
      apiFetch(mod.prefix + '/records?size=500&kind=' + encodeURIComponent(c.key))));

    // مدیریت بالادست واحدها: همین فهرست، هم گزینهٔ فیلد انتخابی فرم را می‌سازد و هم
    // عنوان فارسی ردیف جدول را؛ پس هیچ عنوانی در UI تکرار نمی‌شود.
    rkSetSelectOpts('management', (fetched[0] && fetched[0].ok ? (fetched[0].data.items || []) : [])
      .filter(r => r.fields && r.fields.key)
      .map(r => ({key: r.fields.key, titleFa: r.title})));

    const cards = CATALOG_KINDS.map((c, i) => {
      const kind = (schema.kinds || []).find(k => k.key === c.key);
      const res = fetched[i];
      if (!kind) return '';
      if (!res || !res.ok) return `<div class="card card-pad" style="margin-bottom:12px;border-color:#fecaca;color:#b91c1c">⛔ ${esc(c.titleFa)}: ${esc((res && res.message) || 'بارگذاری نشد')}</div>`;
      catalogBind(kind);
      const items = (res.data && res.data.items) || [];
      return `<div class="card card-pad" style="margin-bottom:12px">
        <div class="card-title">🗂️ ${esc(c.titleFa)}
          <span class="muted" style="font-weight:400;font-size:.82rem">— ${fa(items.length)} قلم</span></div>
        <div class="muted" style="margin:4px 0 10px;font-size:.82rem;line-height:2">${esc(c.hint)}</div>
        ${catalogCardHTML(kind, items, mod.prefix)}
        <div style="margin-top:10px">
          <button class="btn btn-primary" onclick="rkCreate('${esc(catalogPage(c.key))}')">➕ افزودن قلم تازه</button>
        </div>
      </div>`;
    });

    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      ℹ️ این اقلام مرجع زمان‌اجرای سامانه‌اند: فرم تخصیص وام و پاداش و اعتبارسنجی سرور،
      فهرست‌هایشان را از همین رکوردها می‌خوانند. «غیرفعال‌سازی» قلم را از فرم‌ها بیرون
      می‌برد ولی دادهٔ قدیمی که به آن اشاره دارد دست‌نخورده می‌ماند.
    </div>`;
    return note + cards.join('');
  });

  return liveShell('کاتالوگ‌های مرجع', 'مدیریت‌ها، بانک‌های عامل و واحدهای سازمانی — مرجع فرم‌های وام و پاداش');
}

/* ثبت همهٔ صفحه‌های رکوردی در مسیریاب زنده؛ مسیریاب این‌ها را پیش از قالب نمایشی می‌بیند. */
for (const page of Object.keys(RK_PAGES)) {
  LIVE_PAGES[page] = () => pageRecordkitLive(page);
}

/* پنل کاتالوگ‌های مرجع: یک صفحهٔ اختصاصی با سه کاتالوگ. */
LIVE_PAGES.catalogs = () => pageCatalogsLive();
