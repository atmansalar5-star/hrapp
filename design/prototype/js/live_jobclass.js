/* =====================================================================
   شرح شغل · شرح مشاغل · شناسنامهٔ شغل · فرم محاسباتی شرایط احراز
   ---------------------------------------------------------------------
   چهار صفحهٔ ماژول طبقه‌بندی مشاغل. قاعده‌ها در سرور هستند و این لایه فقط نمایش
   می‌دهد:

     - شرح شغل: رکورد نوع «jobDescriptions» با همهٔ فیلدهای سند (هدف، وظایف،
       مسئولیت‌ها، اختیارات، شاخص‌ها، شرایط کار و شرایط احراز)؛ فرم ثبت/ویرایش از
       موتور عمومی رکوردها می‌آید تا فیلد تازه‌ای در UI ساخته نشود.
     - شرح مشاغل: فهرست شرح شغل‌های سازمان از `/job-registry`.
     - شناسنامهٔ شغل: کارت یک‌صفحه‌ای از `/job-card` (دامنه آن را از دادهٔ جاری
       می‌سازد، پس کارت و شرح شغل هرگز واگرا نمی‌شوند).
     - فرم محاسباتی شرایط احراز: محاسبهٔ کمبود تحصیلات، شرایط معادل و مازاد از
       `/qualification-plan` — همان تابع قاعده‌ای که تصویب‌خواهی انتصاب استفاده می‌کند.
   ===================================================================== */

/** وضعیت پایدار این صفحه‌ها (انتخاب شغل، کارمند و نتیجهٔ محاسبه). */
window.JOB = window.JOB || {jobId: '', empId: '', degree: '', field: '', exp: '', manual: false};

/** پیشوند API ماژول. */
function jobPrefix() { return '/api/v1/jobclass'; }

/** پارامترها و فهرست‌های مرجع (یک‌بار در هر نشست خوانده می‌شود). */
async function jobDefaultsCached() {
  if (window.JOBDEF_CACHE) return window.JOBDEF_CACHE;
  const res = await apiFetch(jobPrefix() + '/job-defaults');
  window.JOBDEF_CACHE = res.ok ? (res.data || {}) : {};
  return window.JOBDEF_CACHE;
}

/** فهرست شرح شغل‌ها (برای جدول‌ها و انتخابگرها). */
async function jobDescriptionList() {
  const res = await apiFetch(jobPrefix() + '/records?kind=jobDescriptions&size=200');
  return res.ok ? ((res.data && res.data.items) || []) : [];
}

/** عنوان گروه شغلی از نردبان سازمان. */
function jobGradeFa(defaults, key) {
  const hit = ((defaults && defaults.grades) || []).find(g => g.key === String(key || ''));
  return hit ? hit.titleFa : (key || '—');
}

/** عنوان مقطع تحصیلی از نردبان مدرک. */
function jobDegreeFa(defaults, key) {
  const hit = ((defaults && defaults.degreeLevels) || []).find(d => d.key === String(key || ''));
  return hit ? hit.titleFa : (key || '—');
}

/** رنگ نشان نتیجهٔ سنجش شرایط احراز. */
function jobVerdictChip(result) {
  const ok = result && result.eligible;
  const color = ok ? '#16A34A' : '#DC2626';
  const bg = ok ? '#DCFCE7' : '#FEE2E2';
  return `<span class="chip" style="color:${color};background:${bg}">${esc((result && result.verdictFa) || '—')}</span>`;
}

/** اقدام‌های مجاز یک رکورد بر پایهٔ انتقال‌های اعلامی دامنه. */
function jobLegalActions(kind, rec) {
  if (!kind) return [];
  const allowed = (kind.transitions || {})[rec.status] || [];
  return (kind.actions || []).filter(a => !a.to || allowed.includes(a.to));
}

/* ------------------------------ شرح شغل ------------------------------ */

function pageJobDescriptionsLive() {
  if (!isLive()) return offlineFallback('jobDescriptions');
  liveLoad(async () => {
    const [schemaRes, defaults, items] = await Promise.all([
      apiFetch(jobPrefix() + '/schema'),
      jobDefaultsCached(),
      jobDescriptionList(),
    ]);
    const mod = (typeof rkFindModule === 'function' && rkFindModule('jobDescriptions')) || {prefix: jobPrefix()};
    let descKind = null;
    if (schemaRes.ok) {
      descKind = ((schemaRes.data || {}).kinds || []).find(k => k.key === 'jobDescriptions') || null;
      // فرم ثبت/ویرایش شرح شغل با همهٔ فیلدهای سند از موتور عمومی رکوردها می‌آید.
      rkBindKind('jobDescriptions', mod, descKind);
    }

    const approved = items.filter(r => r.status === 'approved' || r.status === 'revised').length;
    const stats = `<div class="grid-stats">
      ${statCard({ico: '📄', tint: '#EFF6FF', accent: '#2563EB', val: fa(items.length), label: 'شرح شغل ثبت‌شده'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(approved), label: 'شناسنامهٔ تأییدشده (مبنای انتصاب)'})}
      ${statCard({ico: '📝', tint: '#FEF3C7', accent: '#D97706', val: fa(items.length - approved), label: 'در انتظار تأیید'})}
      ${statCard({ico: '⚖️', tint: '#EDE9FE', accent: '#7C3AED', val: fa(items.length), label: 'قابل سنجش با فرم محاسباتی'})}
    </div>`;

    const rows = items.map(r => {
      const f = r.fields || {};
      const acts = jobLegalActions(descKind, r).map(a =>
        `<button class="btn btn-sm ${a.style === 'primary' ? 'btn-primary' : 'btn-ghost'}"
          onclick="rkAct('${esc(mod.prefix)}','${esc(r.id)}','${esc(a.value)}',${a.noteRequired ? 'true' : 'false'},'${esc(a.labelFa)}')">${esc(a.labelFa)}</button>`).join(' ');
      return `<tr>
        <td class="ltr" style="font-size:.74rem">${esc(f.jobCode || r.id)}</td>
        <td><b>${esc(r.title)}</b><div class="muted" style="font-size:.72rem">${esc(f.purpose || '—')}</div></td>
        <td>${esc(jobGradeFa(defaults, f.jobGroup))}</td>
        <td>${esc(f.unit || '—')}</td>
        <td>${esc(jobDegreeFa(defaults, f.degreeLevel))}</td>
        <td class="num">${fa(f.fields || '—')}</td>
        <td class="num">${fa(f.minExperienceYears || 0)} سال</td>
        <td>${statusChip(r.status, r.statusFa)}</td>
        <td><div class="row-actions">
          <button class="btn btn-sm btn-ghost" title="شناسنامهٔ شغل" onclick="jobOpenCard('${esc(r.id)}')">🪪 شناسنامه</button>
          <button class="btn btn-sm btn-ghost" title="ویرایش شرح شغل" onclick="rkEdit('jobDescriptions','${esc(r.id)}')">✏️</button>
          ${acts}
        </div></td></tr>`;
    });

    return stats +
      statNote('شرح شغل سند مرجع شناسنامهٔ شغل و سنجش شرایط احراز است: مقطع تحصیلی، رشته‌های مرتبط و کمینهٔ سابقه از همین رکورد خوانده می‌شود. شرح شغل پیش‌نویس مبنای انتصاب نیست و باید نخست تأیید شود.') +
      statNote(((defaults && defaults.noteFa) || '')) +
      `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">📄 شرح شغل‌های ثبت‌شده</div>
        ${liveTable(['کد شغل', 'عنوان و هدف', 'گروه شغلی', 'واحد', 'مقطع لازم', 'رشته‌های مرتبط', 'کمینهٔ سابقه', 'وضعیت', 'اقدام'], rows, rows.length)}
      </div>`;
  });
  return liveShell('شرح شغل', 'تدوین و تأیید شرح شغل: وظایف، مسئولیت‌ها، شرایط کار و شرایط احراز هر شغل',
    `<button class="btn btn-primary" onclick="rkCreate('jobDescriptions')">➕ شرح شغل تازه</button>`);
}

/** بازکردن شناسنامهٔ شغل از فهرست شرح شغل. */
function jobOpenCard(id) {
  window.JOB.jobId = id || '';
  go('jobCard');
}

/* ----------------------------- شرح مشاغل ----------------------------- */

function pageJobRegistryLive() {
  if (!isLive()) return offlineFallback('jobRegistry');
  liveLoad(async () => {
    const res = await apiFetch(jobPrefix() + '/job-registry');
    if (!res.ok) return liveErrorBox(res.message);
    const data = res.data || {};
    const items = data.items || [];
    window.JOBREG = items;

    const stats = `<div class="grid-stats">
      ${statCard({ico: '🗂️', tint: '#EFF6FF', accent: '#2563EB', val: fa(data.total || 0), label: 'شغل در مجموعهٔ شرح مشاغل'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(data.approvedCount || 0), label: 'شناسنامهٔ تأییدشده'})}
      ${statCard({ico: '📝', tint: '#FEF3C7', accent: '#D97706', val: fa(data.draftCount || 0), label: 'پیش‌نویس در انتظار تأیید'})}
      ${statCard({ico: '🏷️', tint: '#EDE9FE', accent: '#7C3AED', val: fa(new Set(items.map(i => i.groupFa || '—')).size), label: 'گروه شغلی درگیر'})}
    </div>`;

    return stats + statNote(data.noteFa || '') +
      `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:10px;flex-wrap:wrap;align-items:flex-end">
        <div class="field" style="min-width:280px"><label>جست‌وجو در شرح مشاغل</label>
          <input class="inp" id="jobRegQuery" placeholder="🔍 عنوان، کد شغل، واحد یا گروه شغلی" oninput="jobRegFilter()"></div>
        <span class="muted" style="font-size:.78rem">فیلتر در همین صفحه روی دادهٔ سرور اعمال می‌شود.</span>
      </div>
      <div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">🗂️ مجموعهٔ شرح مشاغل سازمان</div>
        <div id="jobRegBox">${jobRegTableHTML(items)}</div>
      </div>`;
  });
  return liveShell('شرح مشاغل', 'مجموعهٔ شرح شغل‌های سازمان با شرایط احراز هر شغل — مرجع بازبینی سالانهٔ مشاغل');
}

/** جدول شرح مشاغل (برای فیلتر درجا هم استفاده می‌شود). */
function jobRegTableHTML(items) {
  const rows = (items || []).map(i => `<tr>
    <td class="ltr" style="font-size:.74rem">${esc(i.code || i.id)}</td>
    <td><b>${esc(i.titleFa)}</b></td>
    <td>${esc(i.groupFa || '—')}</td>
    <td>${esc(i.unitFa || '—')}</td>
    <td>${esc(i.degreeFa || '—')}</td>
    <td class="num">${fa(i.fieldsFa || '—')}</td>
    <td class="num">${fa(i.minExperience || 0)} سال</td>
    <td>${i.approved
      ? '<span class="chip c-ok">✅ تأییدشده — مبنای انتصاب</span>'
      : '<span class="chip c-wait">📝 پیش‌نویس</span>'}</td>
    <td><div class="row-actions">
      <button class="btn btn-sm btn-ghost" onclick="jobOpenCard('${esc(i.id)}')">🪪 شناسنامه</button>
      <button class="btn btn-sm btn-ghost" onclick="jobQualifyFor('${esc(i.id)}')">🧮 سنجش شرایط</button>
    </div></td></tr>`);
  return rows.length
    ? liveTable(['کد شغل', 'عنوان شغل', 'گروه شغلی', 'واحد', 'مقطع لازم', 'رشته‌های مرتبط', 'کمینهٔ سابقه', 'وضعیت', 'اقدام'], rows, rows.length)
    : '<div class="muted" style="font-size:.8rem">شرح شغلی ثبت نشده است.</div>';
}

/** فیلتر درجای مجموعهٔ شرح مشاغل. */
function jobRegFilter() {
  const el = document.getElementById('jobRegQuery');
  const q = (el ? el.value : '').trim().toLowerCase();
  const items = (window.JOBREG || []).filter(i => !q || [i.titleFa, i.code, i.unitFa, i.groupFa, i.fieldsFa]
    .map(v => String(v || '').toLowerCase()).some(v => v.includes(q)));
  const box = document.getElementById('jobRegBox');
  if (box) box.innerHTML = jobRegTableHTML(items);
}

/** رفتن به فرم محاسباتی با شناسنامهٔ انتخاب‌شده. */
function jobQualifyFor(id) {
  window.JOB.jobId = id || '';
  go('jobQualification');
}

/* --------------------------- شناسنامهٔ شغل --------------------------- */

function pageJobCardLive() {
  if (!isLive()) return offlineFallback('jobCard');
  liveLoad(async () => {
    const defaults = await jobDefaultsCached();
    const jobs = (defaults && defaults.jobs) || [];
    if (!window.JOB.jobId && jobs.length) window.JOB.jobId = jobs[0].id;

    const picker = `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:10px;align-items:flex-end;flex-wrap:wrap">
      <div class="field" style="min-width:320px"><label>شناسنامهٔ شغل</label>
        <select id="jobCardPick" onchange="jobPickCard(this.value)">
          ${jobs.map(j => `<option value="${esc(j.id)}" ${j.id === window.JOB.jobId ? 'selected' : ''}>${esc(j.titleFa)}${j.code ? ' (' + esc(j.code) + ')' : ''}</option>`).join('') || '<option value="">—</option>'}
        </select></div>
      <button class="btn btn-ghost" onclick="window.print()">🖨 چاپ شناسنامه</button>
      <button class="btn btn-primary" onclick="jobQualifyFor('${esc(window.JOB.jobId)}')">🧮 سنجش شرایط احراز با این شناسنامه</button>
    </div>`;

    if (!window.JOB.jobId) {
      return picker + '<div class="muted">شرح شغلی ثبت نشده است؛ نخست از صفحهٔ «شرح شغل» یک شغل با شرایط احراز بسازید.</div>';
    }
    const res = await apiFetch(jobPrefix() + '/job-card?id=' + encodeURIComponent(window.JOB.jobId));
    if (!res.ok) return picker + liveErrorBox(res.message);
    return picker + jobCardHTML(res.data || {});
  });
  return liveShell('شناسنامهٔ شغل', 'کارت یک‌صفحه‌ای شغل: شناسه، وظایف، شرایط کار، شرایط احراز و مسیر ارتقا');
}

/** انتخاب شناسنامه از فهرست. */
function jobPickCard(id) {
  window.JOB.jobId = id || '';
  go('jobCard');
}

/** بدنهٔ چاپی شناسنامهٔ شغل. */
function jobCardHTML(card) {
  const blocks = (card.blocks || []).map(b => {
    const rows = (b.rows || []).map(r => `<tr>
      <th style="width:22%;text-align:right;background:#F8FAFC">${esc(r.labelFa)}</th>
      <td style="${r.long ? 'line-height:2' : ''}">${esc(r.valueFa || '—')}</td></tr>`).join('');
    return `<div class="card card-pad" style="margin-bottom:12px">
      <div class="card-title">${esc(b.titleFa)}</div>
      <table class="tbl" style="width:100%">${rows}</table>
    </div>`;
  }).join('');

  const plan = (card.executionPlanFa || []).map(s => `<li style="line-height:2">${esc(s)}</li>`).join('');
  const head = `<div class="card card-pad" style="margin-bottom:12px;display:flex;justify-content:space-between;gap:14px;flex-wrap:wrap">
    <div>
      <div style="font-size:1.1rem;font-weight:800">${esc(card.titleFa)}</div>
      <div class="muted" style="font-size:.8rem;margin-top:4px">
        کد شغل: <b class="ltr">${esc(card.jobCode || '—')}</b>
        · شناسهٔ سند: <b class="ltr">${esc(card.recordId)}</b>
        · وضعیت: ${esc(card.statusFa)}
      </div>
    </div>
    <div style="text-align:left">
      ${card.approved
        ? '<span class="chip c-ok">✅ مبنای انتصاب و ارتقا</span>'
        : '<span class="chip c-wait">📝 پیش‌نویس — مبنای انتصاب نیست</span>'}
      <div class="muted" style="font-size:.78rem;margin-top:6px">شرایط احراز: ${esc(card.requirementFa || '—')}</div>
    </div>
  </div>`;

  return head + blocks +
    `<div class="card card-pad" style="margin-bottom:12px">
      <div class="card-title">🧭 گام‌های انتصاب و ارتقا بر پایهٔ این شناسنامه</div>
      <ol style="margin:0;padding-inline-start:18px">${plan}</ol>
    </div>` +
    statNote(card.noteFa || '');
}

/* --------------------- فرم محاسباتی شرایط احراز --------------------- */

function pageJobQualificationLive() {
  if (!isLive()) return offlineFallback('jobQualification');
  liveLoad(async () => {
    const defaults = await jobDefaultsCached();
    const jobs = (defaults && defaults.jobs) || [];
    const employees = (defaults && defaults.employees) || [];
    const degrees = (defaults && defaults.degreeLevels) || [];
    if (!window.JOB.jobId && jobs.length) window.JOB.jobId = jobs[0].id;

    const jobOpts = jobs.map(j => `<option value="${esc(j.id)}" ${j.id === window.JOB.jobId ? 'selected' : ''}>
      ${esc(j.titleFa)}${j.code ? ' (' + esc(j.code) + ')' : ''}${j.approved ? '' : ' — پیش‌نویس'}</option>`).join('');
    const empOpts = employees.map(e => `<option value="${esc(e.id)}" ${e.id === window.JOB.empId ? 'selected' : ''}>${esc(e.titleFa)}</option>`).join('');
    const degOpts = degrees.map(d => `<option value="${esc(d.key)}" ${d.key === window.JOB.degree ? 'selected' : ''}>${esc(d.titleFa)}</option>`).join('');

    const form = `<div class="card card-pad" style="margin-bottom:12px">
      <div class="card-title">🧾 مشخصات سنجش</div>
      <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:12px">
        <div class="field"><label>شناسنامهٔ شغل (شرط شغل)</label>
          <select id="jqJob" onchange="jobQualPickJob(this.value)">
            <option value="" ${window.JOB.manual ? 'selected' : ''}>— شرط دستی —</option>${jobOpts}</select></div>
        <div class="field"><label>کارمند (اختیاری — برای پیش‌پرکردن سابقه)</label>
          <select id="jqEmp" onchange="jobQualPickEmployee(this.value)">
            <option value="">— بدون کارمند —</option>${empOpts}</select></div>
        <div class="field"><label>مقطع تحصیلی متقاضی <span class="req">*</span></label>
          <select id="jqDegree" onchange="window.JOB.degree=this.value">
            <option value="">— انتخاب کنید —</option>${degOpts}</select></div>
        <div class="field"><label>رشتهٔ تحصیلی متقاضی</label>
          <input class="inp" id="jqField" value="${esc(window.JOB.field || '')}" oninput="window.JOB.field=this.value" placeholder="مثال: مدیریت بازرگانی"></div>
        <div class="field"><label>سابقهٔ مرتبط (سال)</label>
          <input class="inp num" id="jqExp" value="${esc(window.JOB.exp === '' ? '' : window.JOB.exp)}" oninput="window.JOB.exp=this.value"></div>
        <div class="field"><label>سنجش با شرط دستی</label>
          <select id="jqManual" onchange="jobQualToggleManual(this.value)">
            <option value="0" ${window.JOB.manual ? '' : 'selected'}>خیر — از شناسنامه</option>
            <option value="1" ${window.JOB.manual ? 'selected' : ''}>بله — شرط را خودم می‌نویسم</option></select></div>
      </div>
      <div id="jqManualBox" style="margin-top:12px;${window.JOB.manual ? '' : 'display:none'}">
        <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:12px">
          <div class="field"><label>مقطع لازم</label><select id="jqReqDegree">${degOpts}</select></div>
          <div class="field"><label>رشته‌های مرتبط (با ویرگول)</label><input class="inp" id="jqReqFields" placeholder="مدیریت، حسابداری"></div>
          <div class="field"><label>کمینهٔ سابقه (سال)</label><input class="inp num" id="jqReqExp" placeholder="5"></div>
        </div>
      </div>
      <div class="row-actions" style="margin-top:12px">
        <button class="btn btn-primary" onclick="jobQualRun()">🧮 محاسبهٔ فرم</button>
      </div>
    </div>`;

    return form +
      '<div id="jqResult">' + (window.JOB.lastResult ? jobQualResultHTML(window.JOB.lastResult, defaults) : '') + '</div>' +
      jobInstructionHTML(defaults) +
      statNote((defaults && defaults.noteFa) || '');
  });
  return liveShell('فرم محاسباتی شرایط احراز', 'کمبود تحصیلات، شرایط معادل و مازاد بر شرایط احراز — بر پایهٔ شناسنامهٔ شغل و پارامترهای شرکت');
}

/** انتخاب شناسنامه در فرم محاسباتی. */
function jobQualPickJob(id) {
  window.JOB.jobId = id || '';
  if (id) window.JOB.manual = false;
  go('jobQualification');
}

/** انتخاب کارمند: سابقهٔ کل خدمت از سرور پیش‌پر می‌شود (فقط پرکردن، نه قاعده). */
function jobQualPickEmployee(id) {
  window.JOB.empId = id || '';
  if (!id) return;
  const list = ((window.JOBDEF_CACHE && window.JOBDEF_CACHE.employees) || []);
  const hit = list.find(e => e.id === id);
  if (hit && hit.serviceYears != null) {
    window.JOB.exp = String(hit.serviceYears);
    const el = document.getElementById('jqExp');
    if (el) el.value = window.JOB.exp;
  }
}

/** روشن/خاموش‌کردن شرط دستی. */
function jobQualToggleManual(value) {
  window.JOB.manual = String(value) === '1';
  const box = document.getElementById('jqManualBox');
  if (box) box.style.display = window.JOB.manual ? '' : 'none';
}

/** اجرای فرم محاسباتی: ورودی‌ها خوانده و به سرور فرستاده می‌شود. */
async function jobQualRun() {
  const val = id => {
    const el = document.getElementById(id);
    return el ? el.value.trim() : '';
  };
  const body = {
    employeeId: window.JOB.empId || '',
    degreeLevel: val('jqDegree'),
    field: val('jqField'),
    experienceYears: Number(val('jqExp') || 0) || 0,
  };
  if (window.JOB.manual) {
    body.requireDegreeLevel = val('jqReqDegree');
    body.requireFields = val('jqReqFields');
    body.requireMinExperienceYears = Number(val('jqReqExp') || 0) || 0;
  } else {
    body.recordId = window.JOB.jobId;
  }
  const box = document.getElementById('jqResult');
  if (box) box.innerHTML = '<div class="card card-pad muted">⏳ در حال محاسبه…</div>';
  const res = await apiFetch(jobPrefix() + '/qualification-plan', {method: 'POST', body});
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    toast(res.message, 'error');
    return;
  }
  window.JOB.lastResult = res.data || {};
  if (box) box.innerHTML = jobQualResultHTML(window.JOB.lastResult, window.JOBDEF_CACHE || {});
  toast('فرم محاسباتی ساخته شد');
}

/** خروجی فرم محاسباتی: جمع‌بندی، سطرهای گام‌به‌گام و پارامترهای به‌کاررفته. */
function jobQualResultHTML(payload, defaults) {
  const r = payload.result || {};
  const p = payload.params || {};
  const lineRows = (r.lines || []).map(l => `<tr>
    <td class="num">${fa(l.order)}</td>
    <td><b>${esc(l.titleFa)}</b></td>
    <td>${esc(l.valueFa)}</td>
    <td style="font-size:.78rem;line-height:1.9">${esc(l.detailFa)}</td></tr>`);

  const emp = payload.employee || null;
  const head = `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📋 نتیجهٔ سنجش شرایط احراز${payload.jobTitleFa ? ' — ' + esc(payload.jobTitleFa) : ''}</div>
    <div style="display:flex;gap:14px;flex-wrap:wrap;align-items:center;margin:6px 0 10px">
      ${jobVerdictChip(r)}
      <span class="muted" style="font-size:.82rem">شرط شغل: ${esc(payload.requirementFa || r.requirementFa || '—')}</span>
    </div>
    <div style="font-size:.86rem;line-height:2">${esc(r.summaryFa || '')}</div>
    ${emp ? `<div class="muted" style="font-size:.78rem;margin-top:6px">کارمند: ${esc(emp.titleFa)} — ${esc(emp.position || '')} (${esc(emp.unit || '')})</div>` : ''}
    <div class="muted" style="font-size:.76rem;margin-top:6px">تاریخ محاسبه: <b class="num">${esc(payload.today || '')}</b></div>
  </div>`;

  const table = `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧮 فرم محاسباتی خط‌به‌خط</div>
    ${liveTable(['گام', 'موضوع', 'مقدار', 'شرح محاسبه'], lineRows, (r.lines || []).length)}
  </div>`;

  const params = `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">⚙️ پارامترهای به‌کاررفته (از تنظیمات شرکت)</div>
    <table class="tbl" style="width:100%">
      <tr><th style="text-align:right;background:#F8FAFC">هر سطح کمبود مدرک = سابقه</th><td class="num">${fa(p.yearsPerDegreeLevel || 0)} سال</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">سقف جبران با سابقه</th><td class="num">${fa(p.maxCompensatedYears || 0)} سال</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">سابقهٔ اضافه برای رشتهٔ ناهمسو</th><td class="num">${fa(p.fieldMismatchExtraYears || 0)} سال</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">تخفیف هر سطح مدرک مازاد</th><td class="num">${fa(p.excessCreditMonthsPerLevel || 0)} ماه</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">تخفیف هر سال سابقهٔ مازاد</th><td class="num">${fa(p.excessCreditMonthsPerYear || 0)} ماه</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">سقف تخفیف دورهٔ انتظار</th><td class="num">${fa(p.maxExcessCreditMonths || 0)} ماه</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">شرایط معادل فعال است</th><td>${p.equivalenceEnabled ? 'بله' : 'خیر'}</td></tr>
    </table>
  </div>`;

  const note = statNote(payload.noteFa || '');
  return head + table + params + note;
}

/** دستورالعمل مصوب «کمبود تحصیلات و مازاد بر شرایط احراز». */
function jobInstructionHTML(defaults) {
  const steps = (defaults && defaults.instruction) || [];
  if (!steps.length) return '';
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📘 دستورالعمل سنجش شرایط احراز، کمبود تحصیلات و مازاد</div>
    <ol style="margin:0;padding-inline-start:20px;line-height:2.1;font-size:.84rem">
      ${steps.map(s => `<li>${esc(s)}</li>`).join('')}
    </ol>
  </div>`;
}

LIVE_PAGES.jobDescriptions = pageJobDescriptionsLive;
LIVE_PAGES.jobRegistry = pageJobRegistryLive;
LIVE_PAGES.jobCard = pageJobCardLive;
LIVE_PAGES.jobQualification = pageJobQualificationLive;