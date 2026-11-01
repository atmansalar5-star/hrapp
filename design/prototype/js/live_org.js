/* =====================================================================
   ساختار سازمانی: متصدیان پست، عکس کارگزینی، شرایط احراز و تاریخچهٔ سمت
   =====================================================================

   این فایل قاعدهٔ کسب‌وکاری ندارد. هرچه می‌بیند از سه جای سرور می‌آید:
     • `/api/v1/orgstructure/roster`      — پست‌ها + متصدی جاری + عکس + نتیجهٔ احراز
     • `/api/v1/orgstructure/positions/{id}/history` — سابقهٔ انتصاب‌ها + رخدادها
     • `/api/v1/orgstructure/options`     — فهرست نتیجهٔ احراز، کد پست و واحد

   قاعدهٔ «انتصاب بدون احراز شرایط مجاز نیست» در دامنهٔ سرور اجرا می‌شود؛ اینجا فقط
   پیام فارسی همان قاعده نشان داده می‌شود.

   دو نکتهٔ نمایشی:
     ۱. عکس کارگزینی با نشانی امضاشده می‌آید، پس همان <img> بدون توکن هم کار می‌کند؛
        اگر عکس بارگذاری نشده باشد، حرف اول نام می‌نشیند (هیچ تصویر ساختگی نمی‌سازیم).
     ۲. «متصدی» از وضعیت انتصاب خوانده می‌شود، نه از فیلدی که دستی نگه داشته شود؛ با
        «پایان تصدی» پست خودبه‌خود بلا تصدی می‌شود و سابقه دست‌نخورده می‌ماند.
   ===================================================================== */

/** نگاشت نتیجهٔ احراز شرایط به چیپ رنگی (برچسب فارسی از سرور می‌آید). */
const ROSTER_QUAL_CLS = {met: 'c-ok', partial: 'c-wait', not_met: 'c-rej'};

/** نگاشت نتیجهٔ «ارتباط تحصیلات با شغل» به چیپ رنگی. */
const ROSTER_RELEVANCE_CLS = {related: 'c-ok', partially_related: 'c-wait', not_related: 'c-rej'};

/**
 * ثبت گزینه‌های فیلدهای «احراز شرایط» و «کد پست» از پاسخ /options سرور.
 *
 * چرا از سرور: فهرست کد پست‌ها و نتیجه‌های احراز در دامنهٔ ماژول اعلام شده‌اند؛ اگر
 * اینجا ساخته شوند، با تغییر دامنه دو نسخه می‌شوند.
 */
function rkBindRosterOptions(options) {
  const quals = (options && options.qualifications) || [];
  if (quals.length) rkSetSelectOpts('qualification', quals.map(q => ({key: q.key, titleFa: q.titleFa})));
  const positions = (options && options.positions) || [];
  if (positions.length) rkSetSelectOpts('position', positions.map(p => ({key: p.key, titleFa: p.titleFa})));
}

/** اندازهٔ تصویر کارگزینی بر پایهٔ اندازهٔ خواسته‌شده. */
function photoClass(size) {
  return size === 'lg' ? 'photo lg' : size === 'sm' ? 'photo sm' : 'photo';
}

/**
 * سلول «کارمند با عکس کارگزینی»: تصویر واقعی اگر باشد، وگرنه حرف اول نام.
 * هیچ تصویر یا نامی ساخته نمی‌شود؛ داده یا از سرور است یا «—».
 */
function personPhotoCell(id, name, photoUrl, size) {
  const label = name || '—';
  const img = photoUrl
    ? `<img class="${photoClass(size)}" src="${esc(photoUrl)}" alt="عکس کارگزینی ${esc(label)}" loading="lazy">`
    : `<div class="avatar ${size === 'lg' ? 'lg' : 'sq'}" title="عکس کارگزینی بارگذاری نشده است">${esc(initial(label === '—' ? '؟' : label))}</div>`;
  return `<div class="emp-cell">${img}<div>${esc(label)}<div class="muted ltr" style="font-size:.66rem">${esc(id || '')}</div></div></div>`;
}

/** نشانگر نتیجهٔ احراز شرایط با توضیح شواهد. */
function qualificationChip(holder) {
  if (!holder) return '<span class="muted">—</span>';
  const key = holder.qualification || '';
  const chip = pChip(ROSTER_QUAL_CLS[key] || 'c-wait', esc(holder.qualificationFa || key || '—'));
  const evidence = holder.evidence
    ? `<div class="muted" style="font-size:.68rem;line-height:1.8;max-width:260px">${esc(holder.evidence)}</div>`
    : '<div class="muted" style="font-size:.68rem">شواهد احراز ثبت نشده است</div>';
  return chip + evidence;
}

/**
 * جدول پست‌ها با متصدی، عکس کارگزینی و شرایط احراز.
 *
 * assignPage کلید صفحه‌ای است که فرم انتصاب روی آن کار می‌کند (`rkBindKind` ثبتش کرده)
 * و prefix پیشوند API ماژول برای اقدام گردشکار.
 */
function rosterTableHTML(view, assignPage, prefix) {
  const rows = ((view && view.positions) || []).map(p => {
    const holder = p.holder;
    const holderCell = holder
      ? personPhotoCell(holder.employeeId, holder.fullName, holder.photoUrl, 'sm')
      : '<span class="muted">— بلا تصدی —</span>';
    const rowActions = `<div class="row-actions">
      <button class="btn btn-sm btn-primary" title="ثبت انتصاب تازه برای این پست"
        onclick="rosterAppoint('${esc(assignPage)}','${esc(p.code || '')}','${esc(p.title || '')}','${esc(p.unit || '')}')">➕ انتصاب</button>
      <button class="btn btn-sm btn-ghost" title="سابقهٔ انتصاب‌ها و تاریخچهٔ تغییرات"
        onclick="rosterHistory('${esc(prefix)}','${esc(p.id)}','${esc(p.title || '')}')">🕘 تاریخچه</button>
      ${holder ? `<button class="btn btn-sm btn-ghost" title="ثبت پایان تصدی متصدی جاری"
        onclick="rosterEnd('${esc(prefix)}','${esc(holder.assignmentId)}')">⏹ پایان تصدی</button>` : ''}
    </div>`;
    return `<tr>
      <td><b>${esc(p.title || '—')}</b><div class="muted ltr" style="font-size:.66rem">${esc(p.code || '')}</div></td>
      <td>${esc(p.unit || '—')}${p.grade ? `<div class="muted" style="font-size:.68rem">${esc(p.grade)}</div>` : ''}</td>
      <td style="font-size:.74rem;line-height:1.9;max-width:240px">${esc(p.requirements || '—')}</td>
      <td>${holderCell}</td>
      <td>${holder ? holderEducationHTML(holder) : '—'}</td>
      <td>${holder ? holderBasisHTML(holder) : '—'}</td>
      <td>${qualificationChip(holder)}</td>
      <td class="num">${holder && holder.startDate ? fa(holder.startDate) : '—'}
        ${holder && holder.decreeNo ? `<div class="muted" style="font-size:.66rem">${esc(holder.decreeNo)}</div>` : ''}</td>
      <td class="num">${fa(p.assignmentCount || 0)}${p.holderCount > 1 ? ' ' + pChip('c-rej', 'بیش از یک متصدی') : ''}</td>
      <td>${rowActions}</td>
    </tr>`;
  });
  return liveTable(['پست سازمانی', 'واحد', 'شرط‌های احراز', 'متصدی (عکس کارگزینی)', 'تحصیلات متصدی', 'مدرک ملاک حقوق', 'احراز شرایط', 'شروع تصدی', 'سابقه', 'عملیات'],
    rows, rows.length);
}

/** کارت‌های آماری ساختار سازمانی. */
function rosterStatsHTML(view) {
  const s = (view && view.summary) || {};
  return `<div class="grid-stats">
    ${statCard({ico: '🏛️', tint: '#EFF6FF', accent: '#2563EB', val: fa(s.positions || 0), label: 'پست سازمانی'})}
    ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(s.filled || 0), label: 'دارای متصدی'})}
    ${statCard({ico: '🕳️', tint: '#FEF3C7', accent: '#D97706', val: fa(s.vacant || 0), label: 'بلا تصدی'})}
    ${statCard({ico: '🧾', tint: '#EDE9FE', accent: '#7C3AED', val: fa(s.active || 0), label: 'انتصاب جاری'})}
    ${statCard({ico: '🕘', tint: '#F8FAFC', accent: '#64748B', val: fa(s.historical || 0), label: 'انتصاب پیشین (سابقه)'})}
    ${statCard({ico: '📸', tint: '#FEE2E2', accent: '#DC2626', val: fa(s.withoutPhoto || 0), label: 'متصدی بدون عکس کارگزینی'})}
  </div>`;
}

/**
 * پنل ساختار سازمانی در صفحهٔ شناسنامهٔ پست‌ها (موتور عمومی رکورد).
 * بالا: آمار و هشدارها؛ پایین: جدول پست‌ها با متصدی، عکس و احراز شرایط.
 */
function rosterPanelHTML(view, assignPage, prefix) {
  if (!view) return '';
  const warnings = (view.warnings || []).length ? liveWarnBox(view.warnings) : '';
  const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
    🖼️ متصدی هر پست با <b>عکس کارگزینی</b> و <b>نتیجهٔ احراز شرایط</b> نشان داده می‌شود.
    عکس از پروندهٔ پرسنلی همان کارمند خوانده می‌شود؛ برای بارگذاری، از صفحهٔ «کارکنان» وارد پرونده شوید.
    ${esc(view.noteFa || '')}
  </div>`;
  return rosterStatsHTML(view) + warnings + note + rosterTableHTML(view, assignPage, prefix) +
    `<div class="card-title" style="margin:16px 0 6px">📜 سابقهٔ انتصابات (جاری و پیشین)</div>` +
    rosterWorklistHTML(view, assignPage, prefix) +
    `<div class="muted" style="font-size:.76rem;line-height:2;margin-top:8px">
       انتصاب تازه در وضعیت «پیش‌نویس» ثبت می‌شود؛ با «انتصاب و آغاز تصدی» متصدی جاری می‌شود
       (به شرط احراز شرایط و ثبت شواهد) و با «ثبت پایان تصدی» در همین سابقه می‌ماند.
     </div>`;
}

/** باز کردن فرم انتصاب تازه برای یک پست (پیش‌پر از کد و عنوان پست). */
function rosterAppoint(assignPage, code, title, unit) {
  rkCreate(assignPage, {
    position: code,
    positionTitle: title,
    unit: unit,
    __title: 'انتصاب ' + title,
  });
}

/** ثبت پایان تصدی متصدی جاری (اقدام گردشکار؛ دلیل الزامی است). */
function rosterEnd(prefix, assignmentId) {
  const note = prompt('دلیل پایان تصدی را وارد کنید (مثلاً جابه‌جایی، بازنشستگی یا پایان مأموریت):') || '';
  if (!note) return;
  liveSend(prefix + '/records/' + encodeURIComponent(assignmentId) + '/action',
    {method: 'POST', body: {action: 'end', noteFa: note}}, 'پایان تصدی ثبت شد و در سابقهٔ پست ماند')
    .then(ok => ok && liveReload());
}

/**
 * دکمه‌های گردشکار یک انتصاب (انتصاب و آغاز تصدی، پایان تصدی، لغو، ویرایش، حذف).
 *
 * مجاز بودن هر اقدام از شناسهٔ سرور خوانده می‌شود (transitions و editableStatuses)؛
 * قاعدهٔ محتوایی (مثل «بدون احراز شرایط، آغاز تصدی ممکن نیست») در دامنهٔ سرور اجرا
 * می‌شود و پیام فارسی همان قاعده در توست دیده می‌شود.
 */
function rosterAssignmentActionsHTML(assignPage, prefix, a) {
  const st = RK_STATE[assignPage];
  const kind = st && st.kind;
  if (!kind) return '';
  const legal = (kind.actions || []).filter(x => !x.to || ((kind.transitions || {})[a.status] || []).includes(x.to));
  const canEdit = (kind.editableStatuses || [kind.initialStatus]).includes(a.status);
  const buttons = legal.map(x =>
    `<button class="btn btn-sm ${x.style === 'primary' ? 'btn-primary' : 'btn-ghost'}"
      onclick="rkAct('${esc(prefix)}','${esc(a.id)}','${esc(x.value)}',${x.noteRequired ? 'true' : 'false'},'${esc(x.labelFa)}')">${esc(x.labelFa)}</button>`).join('');
  const edit = canEdit
    ? `<button class="btn btn-sm btn-ghost" title="ویرایش" onclick="rkEdit('${esc(assignPage)}','${esc(a.id)}')">✏️</button>
       <button class="btn btn-sm btn-ghost" title="حذف" onclick="rkDelete('${esc(prefix)}','${esc(a.id)}')">🗑️</button>`
    : '';
  return `<div class="row-actions">${buttons}${edit}</div>`;
}

/**
 * «سابقهٔ انتصابات» به‌همراه اقدام‌های گردشکار: هر انتصاب — جاری یا پیشین — یک سطر
 * با دکمه‌های مجاز خودش. این جدول هم در صفحهٔ ساختار سازمانی و هم در پنل شناسنامهٔ
 * پست‌ها می‌آید، پس کاربر بدون ترک صفحه، انتصاب را آغاز/پایان می‌دهد.
 */
function rosterWorklistHTML(view, assignPage, prefix) {
  const items = (view && view.assignments) || [];
  const rows = items.map(a => {
    const f = a.fields || {};
    return `<tr>
      <td class="num">${f.startDate ? fa(f.startDate) : '—'}</td>
      <td>${esc(f.positionTitle || f.position || '—')}<div class="muted ltr" style="font-size:.64rem">${esc(f.position || '')}</div></td>
      <td>${personCell(a.employeeId, a.employeeName)}</td>
      <td class="num">${f.endDate ? fa(f.endDate) : 'اکنون'}</td>
      <td>${pChip(ROSTER_QUAL_CLS[f.qualification] || 'c-wait',
        esc(rkSelectTitle('qualification', f.qualification) || '—'))}
        ${f.evidence ? `<div class="muted" style="font-size:.64rem;line-height:1.7;max-width:200px">${esc(f.evidence)}</div>` : ''}</td>
      <td>${statusChip(a.status, a.statusFa)}</td>
      <td style="font-size:.7rem">${(a.events || []).length ? fa((a.events || []).length) + ' رخداد' : '—'}</td>
      <td>${rosterAssignmentActionsHTML(assignPage, prefix, a)}</td>
    </tr>`;
  });
  if (!rows.length) {
    return `<div class="card card-pad"><div class="empty-state"><div class="big">📭</div>
      هنوز انتصابی ثبت نشده است؛ از دکمهٔ «➕ انتصاب» هر پست شروع کنید.</div></div>`;
  }
  return liveTable(['شروع تصدی', 'پست', 'کارمند', 'پایان تصدی', 'احراز شرایط و شواهد', 'وضعیت', 'سابقهٔ تغییرات', 'عملیات'],
    rows, rows.length);
}

/* ------------------- سابقهٔ انتصاب‌ها و تاریخچهٔ یک سمت ------------------- */

/**
 * جدول انتصاب‌های یک پست (جاری و پیشین) با دورهٔ تصدی.
 * ستون شواهد حذف شده تا جدول در مودال جا شود؛ شواهد هر انتصاب زیر عنوان آن می‌آید.
 */
function rosterAssignmentsTableHTML(items) {
  const rows = (items || []).map(a => `<tr>
    <td class="num ltr">${esc(a.id)}</td>
    <td>${esc(a.title || '—')}
      ${a.employeeName ? `<div class="muted" style="font-size:.68rem">${esc(a.employeeName)}</div>` : ''}
      ${(a.fields || {}).evidence ? `<div class="muted" style="font-size:.66rem;line-height:1.8;max-width:230px">${esc(a.fields.evidence)}</div>` : ''}</td>
    <td class="num">${(a.fields || {}).startDate ? fa(a.fields.startDate) : '—'}</td>
    <td class="num">${(a.fields || {}).endDate ? fa(a.fields.endDate) : 'اکنون'}</td>
    <td>${statusChip(a.status, a.statusFa)}</td>
    <td>${pChip(ROSTER_QUAL_CLS[(a.fields || {}).qualification] || 'c-wait',
      esc(rkSelectTitle('qualification', (a.fields || {}).qualification) || '—'))}</td>
    <td>${esc((a.fields || {}).decreeNo || '—')}</td>
  </tr>`);
  return liveTable(['شناسه', 'عنوان انتصاب و شواهد احراز', 'شروع تصدی', 'پایان تصدی', 'وضعیت', 'احراز شرایط', 'شمارهٔ ابلاغ'],
    rows, rows.length);
}

/** جدول تاریخچهٔ تغییرات سمت (رخدادهای شناسنامهٔ پست + رخدادهای انتصاب‌ها). */
function rosterTimelineTableHTML(items) {
  const rows = (items || []).map(t => `<tr>
    <td class="num">${t.date ? fa(t.date) : '—'}</td>
    <td>${esc(t.sourceFa || t.source || '—')}<div class="muted ltr" style="font-size:.66rem">${esc(t.recordId || '')}</div></td>
    <td>${esc(t.actionFa || t.action || '—')}</td>
    <td class="ltr">${esc(t.actor || '—')}</td>
    <td style="font-size:.74rem">${esc(t.noteFa || '—')}</td>
  </tr>`);
  return liveTable(['تاریخ', 'منبع', 'اقدام', 'کاربر', 'یادداشت'], rows, rows.length);
}

/** مودال «سابقهٔ انتصابات و تاریخچهٔ تغییرات» یک پست سازمانی. */
function rosterHistory(prefix, positionId, titleFa) {
  apiFetch(prefix + '/positions/' + encodeURIComponent(positionId) + '/history').then(res => {
    if (!res.ok) { toast(res.message, 'error'); return; }
    const h = res.data || {};
    const p = h.position || {};
    // متصدی جاری این پست (اگر باشد) با عکس کارگزینی و نتیجهٔ احراز شرایط.
    const current = p.holder || null;
    const holderBlock = current
      ? `<div style="display:flex;gap:12px;align-items:center;margin:6px 0 12px">
           ${personPhotoCell(current.employeeId, current.fullName, current.photoUrl, 'lg')}
           <div style="font-size:.8rem;line-height:2">
             <div>${qualificationChip(current)}</div>
             <div class="muted">شروع تصدی: ${current.startDate ? fa(current.startDate) : '—'}${current.decreeNo ? ' — ابلاغ ' + esc(current.decreeNo) : ''}</div>
           </div>
         </div>`
      : '<div class="muted" style="margin:6px 0 12px">این پست در حال حاضر متصدی ندارد.</div>';
    const info = `<div class="card card-pad" style="margin-bottom:12px;font-size:.8rem;line-height:2">
      <b>${esc(p.title || titleFa || '—')}</b>${p.code ? `<span class="muted ltr"> (${esc(p.code)})</span>` : ''}
      <div class="muted">واحد: ${esc(p.unit || '—')} · گروه شغلی: ${esc(p.grade || '—')} · وضعیت: ${esc(p.statusFa || '—')}</div>
      <div class="muted">شرط‌های احراز: ${esc(p.requirements || '—')}</div>
      ${holderBlock}
    </div>`;
    openModal({
      title: 'سابقه و تاریخچهٔ سمت — ' + (p.title || titleFa || 'پست سازمانی'),
      body: info +
        `<div class="card-title" style="margin:10px 0 6px">سابقهٔ انتصابات</div>` +
        rosterAssignmentsTableHTML(h.assignments) +
        `<div class="card-title" style="margin:14px 0 6px">تاریخچهٔ تغییرات سمت</div>` +
        rosterTimelineTableHTML(h.timeline) +
        (h.noteFa ? `<div class="muted" style="font-size:.76rem;line-height:2;margin-top:8px">${esc(h.noteFa)}</div>` : ''),
      foot: `<button class="btn btn-ghost" onclick="closeModal()">بستن</button>`,
    });
  });
}

/* =====================================================================
   صفحهٔ زندهٔ «ساختار سازمانی» (نمودار واحدها + متصدیان + سابقه)
   ===================================================================== */

/** وضعیت صفحهٔ ساختار سازمانی: کد پست و واحد برای فرم انتصاب. */
const ORG_STATE = {view: null, options: null, prefix: '/api/v1/orgstructure', assignPage: 'org#assignments'};

/* ------------------- کارت‌های نمایشی متصدی و پایه‌های او ------------------- */

/** خلاصهٔ تحصیلات متصدی: آخرین مدرک، رشته/گرایش و ارتباط آن با شغل. */
function holderEducationHTML(holder) {
  const edu = holder && holder.education;
  if (!edu) return '<span class="muted">مدرک تحصیلی ثبت نشده</span>';
  return `<b>${esc(edu.levelFa || '—')}</b>
    <div class="muted" style="font-size:.68rem">${esc(edu.field || '—')}${edu.orientation ? ' — ' + esc(edu.orientation) : ''}</div>
    <div style="font-size:.68rem">${pChip(ROSTER_RELEVANCE_CLS[edu.relevance] || 'c-wait',
      esc(edu.relevanceFa || '—') + (edu.relevancePercent ? ` (${fa(edu.relevancePercent)}٪)` : ''))}</div>`;
}

/** تصمیم مدرک ملاک حقوق متصدی (و معادل‌سازی ایثارگری، اگر باشد). */
function holderBasisHTML(holder) {
  const basis = holder && holder.degreeBasis;
  if (!basis) return '<span class="muted">ملاک حقوق تعیین نشده</span>';
  const upgrade = basis.veteranUpgrade
    ? `<div class="muted" style="font-size:.66rem">ایثارگری: + یک مقطع (${esc(basis.upgradedLevelFa || '—')})</div>`
    : '';
  return `<b>${esc(basis.effectiveLevelFa || basis.levelFa || '—')}</b>
    <div class="muted" style="font-size:.68rem">مدرک: ${esc(basis.levelFa || '—')} — ${esc(basis.field || '')}</div>
    <div style="font-size:.68rem">${pChip(ROSTER_RELEVANCE_CLS[basis.relevance] || 'c-wait',
      esc(basis.relevanceFa || '—') + (basis.relevancePct ? ` (${fa(basis.relevancePct)}٪)` : ''))}</div>
    <div class="muted ltr" style="font-size:.64rem">${esc(basis.councilNo || '')}${basis.councilDate ? ' — ' + esc(basis.councilDate) : ''}</div>
    ${upgrade}`;
}

/** پایهٔ شخصی فرد (از حکم جاری) و پایهٔ سازمانی سمت (از شناسنامهٔ پست). */
function holderBasesHTML(holder, position) {
  const personal = holder && holder.personalBase;
  const org = (position && position.orgBase) || (holder && holder.orgBase);
  const personText = personal
    ? `<b>${esc(personal.jobGroup || '—')}</b>
       <div class="muted" style="font-size:.66rem">پایهٔ شخصی — حکم ${esc(personal.decreeId || '—')}</div>`
    : '<span class="muted">حکم ابلاغ‌شده ثبت نشده</span>';
  const orgText = org
    ? `<b>${esc(org.grade || '—')}</b>
       <div class="muted" style="font-size:.66rem">پایهٔ سازمانی سمت${org.jobFamily ? ' — ' + esc(org.jobFamily) : ''}</div>`
    : '<span class="muted">گروه شغلی پست ثبت نشده</span>';
  return `<div style="font-size:.72rem;line-height:1.9">${personText}</div>
    <div style="font-size:.72rem;line-height:1.9;margin-top:4px">${orgText}</div>`;
}

/** چیپ ایثارگری متصدی (مبنای معادل‌سازی یک مقطع بالاتر). */
function holderVeteranChip(holder) {
  if (!holder || !holder.veteran) return '';
  return pChip('c-wait', 'ایثارگر') +
    (holder.veteranTypeFa ? `<div class="muted" style="font-size:.66rem">${esc(holder.veteranTypeFa)}</div>` : '');
}

/**
 * درخت سازمانی: واحدها به‌صورت سلسله‌مراتبی و زیر هر واحد، پست‌های آن با متصدی.
 *
 * قالب همان درخت‌وارهٔ آشنای کارگزینی است (`ul.org-tree` و `.org-node`)، ولی گره‌ها دادهٔ
 * واقعی دارند: نام واحد، سرپرست، شمار نیرو و برای هر پست، متصدی با عکس کارگزینی، آخرین
 * مدرک تحصیلی، مدرک ملاک حقوق، پایهٔ شخصی فرد و پایهٔ سازمانی سمت.
 */
function orgTreeHTML(tree) {
  if (!tree || !tree.units) return '';
  const byId = {};
  for (const unit of tree.units) byId[unit.id] = unit;
  const children = {};
  for (const unit of tree.units) {
    const parent = unit.parentId && byId[unit.parentId] ? unit.parentId : '__root__';
    (children[parent] = children[parent] || []).push(unit);
  }

  const positionNode = (position, unit) => {
    const holder = position.holder;
    const body = holder
      ? `<div style="display:flex;gap:8px;align-items:flex-start;text-align:right">
           ${personPhotoCell(holder.employeeId, holder.fullName, holder.photoUrl, 'sm')}
           <div style="font-size:.7rem;line-height:1.9">
             <div>${holderEducationHTML(holder)}</div>
             <div>${holderBasisHTML(holder)}</div>
             <div>${holderBasesHTML(holder, position)}</div>
             ${holderVeteranChip(holder)}
           </div>
         </div>`
      : '<span class="muted" style="font-size:.72rem">بلا تصدی</span>';
    return `<li>
      <div class="org-node" style="min-width:230px;text-align:right;border-color:var(--border)">
        <div class="o-name">${esc(position.title || '—')}
          <span class="muted ltr" style="font-weight:400;font-size:.66rem">${esc(position.code || '')}</span></div>
        ${body}
        <div style="margin-top:6px;display:flex;gap:4px;justify-content:flex-end">
          <button class="btn btn-sm btn-ghost" title="سابقه و تاریخچه" onclick="rosterHistory('${esc(ORG_STATE.prefix)}','${esc(position.id)}','${esc(position.title || '')}')">🕘</button>
          <button class="btn btn-sm btn-ghost" title="انتصاب تازه" onclick="rosterAppoint('${esc(ORG_STATE.assignPage)}','${esc(position.code || '')}','${esc(position.title || '')}','${esc(unit || '')}')">➕</button>
          ${holder ? `<button class="btn btn-sm btn-ghost" title="پایان تصدی" onclick="rosterEnd('${esc(ORG_STATE.prefix)}','${esc(holder.assignmentId)}')">⏹</button>` : ''}
        </div>
      </div>
    </li>`;
  };

  const renderUnit = unit => `<li>
    <div class="org-node ${(children[unit.id] || []).length ? '' : 'leaf'}">
      <div class="o-name">${esc(unit.name)}</div>
      <div class="o-head">${unit.head ? 'سرپرست: ' + esc(unit.head) : `${fa(unit.headcount)} نفر در واحد`}</div>
      <div class="o-count num">${fa(unit.positions.length)} پست · ${fa(unit.filled)} دارای متصدی${unit.vacant ? ' · ' + fa(unit.vacant) + ' بلا تصدی' : ''}</div>
    </div>
    ${unit.positions.length ? `<ul>${unit.positions.map(p => positionNode(p, unit.name)).join('')}</ul>` : ''}
    ${(children[unit.id] || []).length ? `<ul>${(children[unit.id] || []).map(renderUnit).join('')}</ul>` : ''}
  </li>`;

  const roots = children['__root__'] || [];
  const unassigned = (tree.unassigned || []).length
    ? `<div class="card card-pad" style="margin-top:12px">
         <div class="card-title">🧷 پست‌های بدون واحد سازمانی مشخص (${fa((tree.unassigned || []).length)})</div>
         <ul class="org-tree"><ul>${(tree.unassigned || []).map(p => positionNode(p, '')).join('')}</ul></ul>
       </div>`
    : '';

  return `<div class="card org-wrap" style="margin-bottom:12px"><ul class="org-tree">
      <li>
        <div class="org-node root">
          <div class="o-name">${esc(tree.root || 'سازمان')}</div>
          <div class="o-count num">${fa(tree.totals.positions || 0)} پست در ${fa(tree.totals.units || 0)} واحد</div>
          <div class="o-head">${fa(tree.totals.filled || 0)} دارای متصدی · ${fa(tree.totals.vacant || 0)} بلا تصدی</div>
        </div>
        <ul>${roots.map(renderUnit).join('')}</ul>
      </li>
    </ul>
    <div class="muted" style="padding:0 18px 14px;font-size:.76rem;line-height:2">${esc(tree.noteFa || '')}</div>
  </div>` + unassigned;
}

/** کارت یک واحد سازمانی با پست‌ها و متصدیانش (جدول تفصیلی زیر درخت). */
function orgUnitCardHTML(unit, positions) {
  const rows = positions.map(p => {
    const holder = p.holder;
    const holderCell = holder
      ? personPhotoCell(holder.employeeId, holder.fullName, holder.photoUrl, 'sm')
      : '<span class="muted">بلا تصدی</span>';
    return `<tr>
      <td><b>${esc(p.title || '—')}</b><div class="muted ltr" style="font-size:.64rem">${esc(p.code || '')}</div></td>
      <td>${holderCell}</td>
      <td>${holder ? holderEducationHTML(holder) : '—'}</td>
      <td>${holder ? holderBasisHTML(holder) : '—'}</td>
      <td>${holder ? holderBasesHTML(holder, p) : '—'}</td>
      <td>${holder ? pChip(ROSTER_QUAL_CLS[holder.qualification] || 'c-wait', esc(holder.qualificationFa || '—')) : '—'}</td>
      <td class="num">${holder && holder.startDate ? fa(holder.startDate) : '—'}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-ghost" title="سابقه و تاریخچه" onclick="rosterHistory('${esc(ORG_STATE.prefix)}','${esc(p.id)}','${esc(p.title || '')}')">🕘</button>
        <button class="btn btn-sm btn-ghost" title="انتصاب تازه" onclick="rosterAppoint('${esc(ORG_STATE.assignPage)}','${esc(p.code || '')}','${esc(p.title || '')}','${esc(p.unit || '')}')">➕</button>
        ${holder ? `<button class="btn btn-sm btn-ghost" title="پایان تصدی" onclick="rosterEnd('${esc(ORG_STATE.prefix)}','${esc(holder.assignmentId)}')">⏹</button>` : ''}
      </div></td>
    </tr>`;
  });
  const filled = positions.filter(p => p.holder).length;
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🏢 ${esc(unit)} <span class="muted" style="font-weight:400;font-size:.8rem">
      — ${fa(positions.length)} پست · ${fa(filled)} دارای متصدی · ${fa(positions.length - filled)} بلا تصدی</span></div>
    ${liveTable(['پست سازمانی', 'متصدی (عکس کارگزینی)', 'تحصیلات متصدی', 'مدرک ملاک حقوق', 'پایهٔ شخصی / سازمانی', 'احراز شرایط', 'شروع تصدی', 'عملیات'], rows, rows.length)}
  </div>`;
}

/** صفحهٔ زندهٔ «ساختار سازمانی». */
function pageOrgLive() {
  if (!isLive()) return offlineFallback('org');
  const mod = rkFindModule('org') || {prefix: '/api/v1/orgstructure', titleFa: 'ساختار سازمانی و تشکیلات'};
  ORG_STATE.prefix = mod.prefix;

  liveLoad(async () => {
    const [rosterRes, optRes, schemaRes] = await Promise.all([
      apiFetch(mod.prefix + '/roster'),
      apiFetch(mod.prefix + '/options'),
      apiFetch(mod.prefix + '/schema'),
    ]);
    if (!rosterRes.ok) return liveErrorBox(rosterRes.message);
    const view = rosterRes.data || {};
    const options = optRes.ok ? optRes.data : null;
    ORG_STATE.view = view;
    ORG_STATE.options = options;
    rkBindRosterOptions(options);
    if (schemaRes.ok) {
      rkBindKind(ORG_STATE.assignPage, mod, ((schemaRes.data || {}).kinds || []).find(k => k.key === 'assignments'));
    }

    // پست‌ها به تفکیک واحد؛ واحدهای بدون پست هم نشان داده می‌شوند تا نمودار کامل بماند.
    const byUnit = {};
    for (const p of (view.positions || [])) {
      const key = p.unit || 'بدون واحد';
      (byUnit[key] = byUnit[key] || []).push(p);
    }
    // ۱) درخت‌وارهٔ سازمانی (سلسله‌مراتب واحدها و پست‌های هر واحد با متصدی).
    const tree = orgTreeHTML(view.tree);
    // ۲) دیتیل چارت زیر درخت: جدول تفصیلی هر واحد با تحصیلات، مدرک ملاک و پایه‌های متصدی.
    const unitCards = Object.keys(byUnit).sort().map(unit => orgUnitCardHTML(unit, byUnit[unit])).join('');

    const warnings = (view.warnings || []).length ? liveWarnBox(view.warnings) : '';
    const note = `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
      🧭 این صفحه دو بخش دارد: <b>درخت‌وارهٔ سازمانی</b> (سلسله‌مراتب واحدها و زیر هر واحد، پست‌ها با متصدی)
      و <b>جدول تفصیلی هر واحد</b> در ادامه. برای هر متصدی، عکس کارگزینی، آخرین مدرک تحصیلی (مقطع، رشته و
      ارتباط با شغل)، مدرک ملاک حقوق، پایهٔ شخصی فرد (از حکم جاری) و پایهٔ سازمانی سمت (از شناسنامهٔ پست)
      نشان داده می‌شود؛ برای ایثارگران، معادل‌سازی «یک مقطع بالاتر» هم روی مدرک ملاک دیده می‌شود.
      ${esc(view.noteFa || '')}
    </div>`;

    // سابقهٔ انتصابات همهٔ پست‌ها با اقدام‌های گردشکار: یک نگاه برای کارگزینی.
    const historyCard = `<div class="card-title" style="margin:16px 0 6px">📜 سابقهٔ انتصابات سازمان</div>` +
      rosterWorklistHTML(view, ORG_STATE.assignPage, ORG_STATE.prefix) +
      `<div class="muted" style="font-size:.76rem;line-height:2;margin-top:8px">
        انتصاب‌های پایان‌یافته حذف نمی‌شوند؛ همین سطرها تاریخچهٔ تصدی سمت‌ها را می‌سازند.
        برای دیدن رخدادهای یک سمت (ثبت، ویرایش، آغاز و پایان تصدی) از دکمهٔ «🕘» هر پست استفاده کنید.
      </div>`;

    return rosterStatsHTML(view) + warnings + note + tree +
      `<div class="card-title" style="margin:16px 0 6px">📋 جدول تفصیلی واحدها (دیتیل چارت)</div>` +
      unitCards + historyCard;
  });

  return liveShell('ساختار سازمانی',
    'درخت‌وارهٔ سازمانی با متصدی هر پست، تحصیلات، مدرک ملاک حقوق، پایه‌ها و عکس کارگزینی',
    `<button class="btn btn-primary" onclick="rosterAppoint('${esc(ORG_STATE.assignPage)}','','','')">➕ ثبت انتصاب</button>`);
}

LIVE_PAGES.org = () => pageOrgLive();