/* =====================================================================
   اطلاعات پایهٔ پرسنلی — «تحصیلات و مدرک ملاک حقوق»
   =====================================================================

   این بخش پروندهٔ کارمند، پاسخ سه پرسش کارگزینی است:
     ۱. سابقهٔ تحصیلی کارمند چیست (مقطع، رشته، گرایش) و تحصیلاتش با شغل مرتبط است یا نه؟
     ۲. آخرین مدرک تحصیلی کدام است و «مدرک ملاک حقوق» بعد از تأیید شورای تعدیل مدرک کدام
        مدرک شده است و میزان ارتباطش چند درصد است؟
     ۳. برای ایثارگران، آیا «یک مقطع بالاتر» معادل‌سازی شده است؟

   هیچ قاعده‌ای اینجا نیست: مقطع‌ها، نتیجه‌های ارتباط و وضعیت شورا از `/api/v1/archive/options`
   می‌آید و قاعده‌های گردشکار (تأیید شورا، جاری‌سازی، معادل‌سازی ایثارگری) در دامنهٔ سرور
   اجرا می‌شود. دو فرم این بخش، فرم‌های مخصوص خودشان‌اند: «ثبت سابقهٔ تحصیلی» و
   «تعیین مدرک ملاک حقوق».
   ===================================================================== */

/** وضعیت پروندهٔ تحصیلی آخرین بارگذاری‌شده (برای فرم‌ها و چاپ). */
const EDU_STATE = {
  employeeId: '',
  // کلید صفحه‌های فرم: هر نوع رکورد یک کلید جدا می‌خواهد تا شناسهٔ نوع در فرم گم نشود.
  educationPage: 'profile#educations',
  basisPage: 'profile#degreeBases',
  prefix: '/api/v1/archive',
  view: null,
  options: null,
};

/** نگاشت وضعیت‌ها و نتیجه‌ها به چیپ رنگی (برچسب فارسی از سرور می‌آید). */
const EDU_BASIS_CLS = {active: 'c-ok', council_approved: 'c-wait', draft: 'c-wait', superseded: 'c-term', rejected: 'c-rej'};
const EDU_RELEVANCE_CLS = {related: 'c-ok', partially_related: 'c-wait', not_related: 'c-rej'};

/** ثبت گزینه‌های فیلدهای انتخابی فرم‌های تحصیلی از پاسخ /options سرور. */
function eduBindOptions(options) {
  const opt = options || {};
  const pairs = value => (value || []).map(x => ({key: x.key, titleFa: x.titleFa}));
  rkSetSelectOpts('level', pairs(opt.degreeLevels));
  rkSetSelectOpts('upgradedLevel', pairs(opt.degreeLevels));
  rkSetSelectOpts('relevance', pairs(opt.relevances));
  rkSetSelectOpts('councilStatus', pairs(opt.councilStatuses));
  rkSetSelectOpts('veteranUpgrade', pairs(opt.yesNo));
}

/** مقطع یک پله بالاتر از فهرست خودِ سرور (سیاست ایثارگران) — بدون ساخت مقدار در رابط. */
function eduNextLevel(levelKey, options) {
  const levels = ((options || {}).degreeLevels || []).slice().sort((a, b) => (a.order || 0) - (b.order || 0));
  const index = levels.findIndex(l => l.key === levelKey);
  return index >= 0 && index + 1 < levels.length ? levels[index + 1] : null;
}

/** عنوان فارسی یک کلید از فهرست گزینه‌ها (اگر نباشد، خود کلید). */
function eduTitle(options, listName, key) {
  const hit = (((options || {})[listName]) || []).find(x => x.key === key);
  return hit ? hit.titleFa : (key || '—');
}

/** چیپ وضعیت تصمیم مدرک ملاک. */
function eduBasisChip(item) {
  return pChip(EDU_BASIS_CLS[item.status] || 'c-wait', esc(item.statusFa || item.status || '—'));
}

/** چیپ ارتباط تحصیلات با شغل. */
function eduRelevanceChip(item) {
  const key = item.relevance || '';
  const percent = item.relevancePercent ? ` (${fa(item.relevancePercent)}٪)` : '';
  return pChip(EDU_RELEVANCE_CLS[key] || 'c-wait', esc(item.relevanceFa || key || '—') + percent);
}

/** کارت‌های آماری پروندهٔ تحصیلی. */
function eduStatsHTML(view, options) {
  const latest = view.latest || null;
  const basis = view.activeBasis || null;
  const veteran = view.veteran || {};
  const levelFa = key => eduTitle(options, 'degreeLevels', key);
  return `<div class="grid-stats">
    ${statCard({ico: '🎓', tint: '#EFF6FF', accent: '#2563EB', val: fa((view.records || []).length), label: 'سابقهٔ تحصیلی ثبت‌شده'})}
    ${statCard({ico: '🏅', tint: '#DCFCE7', accent: '#16A34A',
      val: latest ? esc(latest.levelFa) : '—', label: latest ? 'آخرین مدرک: ' + esc(latest.field || '') : 'آخرین مدرک'})}
    ${statCard({ico: '⚖️', tint: '#EDE9FE', accent: '#7C3AED',
      val: basis ? esc(basis.effectiveLevelFa || basis.levelFa) : '—',
      label: basis ? 'مدرک ملاک حقوق' : 'مدرک ملاک حقوق تعیین نشده'})}
    ${statCard({ico: '🛡️', tint: veteran.isVeteran ? '#FEF3C7' : '#F8FAFC', accent: veteran.isVeteran ? '#D97706' : '#64748B',
      val: veteran.isVeteran ? 'ایثارگر' : 'خیر',
      label: veteran.isVeteran ? (veteran.upgradeNoteFa || 'یک مقطع بالاتر مجاز است') : 'وضعیت ایثارگری'})}
  </div>`;
}

/** جدول سابقه‌های تحصیلی با نشان «آخرین مدرک» و «مدرک ملاک حقوق». */
function eduRecordsTableHTML(view, options, employeeId) {
  const rows = (view.records || []).map(item => `<tr>
    <td><b>${esc(item.levelFa || '—')}</b>${item.latestDegree ? ' ' + pChip('c-ok', 'آخرین مدرک') : ''}
      ${item.isSalaryBasis ? ' ' + pChip('c-done', 'مدرک ملاک حقوق') : ''}</td>
    <td>${esc(item.field || '—')}${item.orientation ? `<div class="muted" style="font-size:.68rem">گرایش: ${esc(item.orientation)}</div>` : ''}</td>
    <td>${esc(item.university || '—')}</td>
    <td class="num">${item.graduationDate ? fa(item.graduationDate) : '—'}
      ${item.average ? `<div class="muted" style="font-size:.66rem">معدل ${fa(item.average)}</div>` : ''}</td>
    <td>${eduRelevanceChip(item)}</td>
    <td>${esc(item.councilStatusFa || '—')}${item.councilNo ? `<div class="muted ltr" style="font-size:.66rem">${esc(item.councilNo)}${item.councilDate ? ' — ' + fa(item.councilDate) : ''}</div>` : ''}</td>
    <td>${statusChip(item.status, item.statusFa)}</td>
    <td><div class="row-actions">
      ${item.status === 'draft' ? `<button class="btn btn-sm btn-primary" onclick="rkAct('${esc(EDU_STATE.prefix)}','${esc(item.id)}','verify',false,'تأیید مدارک تحصیلی')">✔ تأیید مدارک</button>` : ''}
      <button class="btn btn-sm btn-ghost" title="ویرایش سابقهٔ تحصیلی" onclick="rkEdit('${esc(EDU_STATE.educationPage)}','${esc(item.id)}')">✏️</button>
      <button class="btn btn-sm btn-ghost" title="تعیین به‌عنوان مدرک ملاک حقوق" onclick="eduBasisForm('${esc(employeeId)}','${esc(item.id)}')">⚖️ ملاک حقوق</button>
    </div></td>
  </tr>`);
  if (!rows.length) {
    return `<div class="card card-pad"><div class="empty-state"><div class="big">🎓</div>
      برای این کارمند سابقهٔ تحصیلی ثبت نشده است؛ از «➕ ثبت سابقهٔ تحصیلی» شروع کنید.</div></div>`;
  }
  return liveTable(['مقطع', 'رشته و گرایش', 'دانشگاه', 'تاریخ فراغت', 'ارتباط با شغل', 'شورای تعدیل مدرک', 'وضعیت', 'عملیات'],
    rows, rows.length);
}

/** جدول تصمیم‌های «مدرک ملاک حقوق» با فرم و چاپ مخصوص خودش. */
function eduBasesTableHTML(view, options) {
  const rows = (view.bases || []).map(item => `<tr>
    <td><b>${esc(item.levelFa || '—')}</b>${item.veteranUpgrade ? `<div class="muted" style="font-size:.68rem">معادل‌سازی ایثارگری: ${esc(item.upgradedLevelFa || '—')}</div>` : ''}</td>
    <td>${esc(item.field || '—')}${item.orientation ? `<div class="muted" style="font-size:.68rem">${esc(item.orientation)}</div>` : ''}</td>
    <td>${eduRelevanceChip(item)}</td>
    <td>${esc(item.councilNo || '—')}<div class="muted" style="font-size:.66rem">${item.councilDate ? fa(item.councilDate) : ''}</div></td>
    <td class="num">${item.effectiveDate ? fa(item.effectiveDate) : '—'}</td>
    <td>${esc(item.salaryGroup || '—')}</td>
    <td>${eduBasisChip(item)}</td>
    <td><div class="row-actions">
      ${item.status === 'draft' ? `<button class="btn btn-sm btn-primary" onclick="rkAct('${esc(EDU_STATE.prefix)}','${esc(item.id)}','approve',false,'تأیید شورای تعدیل مدرک')">✔ تأیید شورا</button>` : ''}
      ${item.status === 'council_approved' ? `<button class="btn btn-sm btn-primary" onclick="rkAct('${esc(EDU_STATE.prefix)}','${esc(item.id)}','activate',false,'جاری‌سازی به‌عنوان ملاک حقوق')">⚖️ جاری‌سازی</button>` : ''}
      ${item.status === 'active' ? `<button class="btn btn-sm btn-ghost" onclick="rkAct('${esc(EDU_STATE.prefix)}','${esc(item.id)}','supersede',true,'جانشینی با مدرک تازه')">↻ جانشینی</button>` : ''}
      <button class="btn btn-sm btn-ghost" title="ویرایش تصمیم" onclick="rkEdit('${esc(EDU_STATE.basisPage)}','${esc(item.id)}')">✏️</button>
      <button class="btn btn-sm btn-ghost" title="چاپ فرم تعیین مدرک ملاک حقوق"
        onclick="liveOpenDoc('degree_basis','${esc(item.id)}')">🖨️ فرم</button>
    </div></td>
  </tr>`);
  if (!rows.length) {
    return `<div class="card card-pad"><div class="empty-state"><div class="big">⚖️</div>
      «مدرک ملاک حقوق» تعیین نشده است؛ از «⚖️ تعیین مدرک ملاک حقوق» فرم مخصوص آن را پر کنید.</div></div>`;
  }
  return liveTable(['مقطع ملاک', 'رشته', 'ارتباط و درصد', 'صورت‌جلسهٔ شورا', 'تاریخ اجرا', 'گروه شغلی مبنا', 'وضعیت', 'عملیات'],
    rows, rows.length);
}

/**
 * فرم مخصوص «تعیین مدرک ملاک حقوق».
 *
 * چرا فرم جدا: این تصمیم یک فرایند مستقل است — صورت‌جلسهٔ شورای تعدیل مدرک، تاریخ اجرا،
 * گروه شغلی مبنا و (برای ایثارگران) معادل‌سازی یک مقطع بالاتر. فرم با دادهٔ همان سابقهٔ
 * تحصیلی انتخاب‌شده پیش‌پر می‌شود تا کارشناس دوباره‌کاری نکند، ولی همهٔ فیلدها قابل
 * اصلاح‌اند؛ تصمیم نهایی در سرور اعتبارسنجی و ثبت می‌شود.
 */
function eduBasisForm(employeeId, educationId) {
  const view = EDU_STATE.view || {};
  const options = EDU_STATE.options || {};
  const record = (view.records || []).find(r => r.id === educationId);
  const latest = view.latest || null;
  const veteran = view.veteran || {};
  const source = record || latest;
  const prefill = {
    __employeeId: employeeId,
    __title: 'مدرک ملاک حقوق' + (source && source.field ? ' — ' + source.field : ''),
  };
  if (source) {
    prefill.education = source.id;
    prefill.level = source.level || '';
    prefill.field = source.field || '';
    prefill.orientation = source.orientation || '';
    prefill.relevance = source.relevance || '';
    prefill.relevancePercent = source.relevancePercent || '';
    if (source.councilNo) prefill.councilNo = source.councilNo;
    if (source.councilDate) prefill.councilDate = source.councilDate;
  }
  // سیاست ایثارگران: «یک مقطع بالاتر» پیش‌انتخاب می‌شود و مقطع معادل‌سازی‌شده از فهرست
  // مقاطع خودِ سرور ساخته می‌شود؛ کارشناس می‌تواند آن را تغییر دهد.
  prefill.veteranUpgrade = veteran.isVeteran ? 'yes' : 'no';
  if (veteran.isVeteran && source && source.level) {
    const next = eduNextLevel(source.level, options);
    if (next) prefill.upgradedLevel = next.key;
  }
  rkCreate(EDU_STATE.basisPage, prefill);
  if (veteran.isVeteran) {
    toast('این کارمند ایثارگر است؛ «یک مقطع بالاتر» پیش‌انتخاب شد. با این حال، تأیید شورای تعدیل مدرک و ثبت صورت‌جلسه الزامی است.', 'info');
  }
}

/**
 * تب «تحصیلات و مدرک ملاک» در پروندهٔ کارمند.
 *
 * داده از خلاصهٔ خودِ سرور می‌آید (`/api/v1/archive/education/{id}`) و گزینه‌های فرم از
 * `/api/v1/archive/options`؛ درخت تصمیم‌گیری (شورا، معادل‌سازی ایثارگری) در دامنه اجرا می‌شود.
 */
async function educationTabHTML(employee, employeeId) {
  const [eduRes, optRes, schemaRes] = await Promise.all([
    apiFetch('/api/v1/archive/education/' + encodeURIComponent(employeeId)),
    apiFetch('/api/v1/archive/options'),
    apiFetch('/api/v1/archive/schema'),
  ]);
  if (!eduRes.ok) return liveErrorBox(eduRes.message);
  const view = eduRes.data || {};
  const options = optRes.ok ? optRes.data : {};
  EDU_STATE.employeeId = employeeId;
  EDU_STATE.view = view;
  EDU_STATE.options = options;
  EDU_STATE.prefix = '/api/v1/archive';
  eduBindOptions(options);
  if (schemaRes.ok) {
    const kinds = (schemaRes.data || {}).kinds || [];
    const mod = {prefix: '/api/v1/archive'};
    rkBindKind(EDU_STATE.educationPage, mod, kinds.find(k => k.key === 'educations'));
    rkBindKind(EDU_STATE.basisPage, mod, kinds.find(k => k.key === 'degreeBases'));
  }

  const warnings = (view.warnings || []).length ? liveWarnBox(view.warnings) : '';
  const basisNote = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
    <b>ملاک حقوق:</b> ${esc(view.basisNoteFa || '—')}
    ${view.veteran && view.veteran.isVeteran
      ? `<div class="muted">${esc(view.veteran.upgradeNoteFa || '')}</div>` : ''}
    ${view.veteran && view.veteran.noteFa ? `<div class="muted">${esc(view.veteran.noteFa)}</div>` : ''}
  </div>`;
  const actions = `<div class="row-actions" style="margin-bottom:12px">
    <button class="btn btn-primary" onclick="eduAddEducation('${esc(employeeId)}','${esc((employee && employee.fullName) || '')}')">➕ ثبت سابقهٔ تحصیلی</button>
    <button class="btn" onclick="eduBasisForm('${esc(employeeId)}','')">⚖️ تعیین مدرک ملاک حقوق</button>
  </div>`;

  return eduStatsHTML(view, options) + warnings + basisNote + actions +
    `<div class="card-title" style="margin:12px 0 6px">📚 سابقهٔ تحصیلی</div>` +
    eduRecordsTableHTML(view, options, employeeId) +
    `<div class="card-title" style="margin:16px 0 6px">⚖️ تصمیم‌های مدرک ملاک حقوق</div>` +
    eduBasesTableHTML(view, options) +
    statNote('هر سابقهٔ تحصیلی یک رکورد است و «آخرین مدرک» از بالاترین مقطع خوانده می‌شود. ' +
      'تصمیم «مدرک ملاک حقوق» فرم جداگانهٔ خودش را دارد و تنها با شماره و تاریخ صورت‌جلسهٔ شورای تعدیل مدرک تأیید می‌شود؛ ' +
      'برای ایثارگران، مقطع معادل‌سازی‌شدهٔ یک پله بالاتر مبنای پرداخت است.');
}

/** ثبت سابقهٔ تحصیلی تازه برای همین کارمند. */
function eduAddEducation(employeeId, employeeName) {
  rkCreate(EDU_STATE.educationPage, {
    __employeeId: employeeId,
    __title: 'سابقهٔ تحصیلی' + (employeeName ? ' — ' + employeeName : ''),
  });
}
