/* =====================================================================
   فرم تصویب‌خواهی انتصاب و ترفیع (ماژول انتصابات، ترفیعات و کارراههٔ شغلی)
   ---------------------------------------------------------------------
   چرا این صفحه: پیش از ثبت یک انتصاب یا ترفیع باید روشن شود که (۱) شرط‌های سازمانی
   برآورده است یا نه، (۲) چه کسانی و به چه ترتیبی باید تصویب کنند، و (۳) اجرای ابلاغ
   چه گام‌هایی دارد. هر سه از سرور می‌آید:

     GET  /api/v1/career/appointment-defaults   پارامترها، نردبان گروه شغلی، کارکنان، پست‌ها
     POST /api/v1/career/appointment-plan       سنجش شرط‌ها + زنجیرهٔ تصویب + گام‌های اجرا
     POST /api/v1/career/appointment-execute    اجرا و ابلاغ پس از تصویب

   هیچ قاعده‌ای در این فایل نیست: آستانه‌ها (سابقه، امتیاز، جهش گروه، تعهد) و سطح تصویب
   همه پارامتر شرکت‌اند و در موتور قواعد سنجیده می‌شوند. این فایل فیلدها را می‌فرستد و
   پاسخ را می‌چیند.
   ===================================================================== */

var APA_DEFAULTS = null;   // پارامترها و فهرست‌های مرجع
var APA_RESULT = null;     // آخرین پاسخ تصویب‌خواهی
var APA_EMPLOYEES = [];    // کارکنان
var APA_POSITIONS = [];    // پست‌های سازمانی
var APA_TIMER = null;      // زمان‌سنج محاسبهٔ خودکار
var APA_RECORDS = [];      // درخواست‌های جاری
var APA_SCHEMA = null;     // شناسهٔ نوع‌های رکورد (اقدام‌های مجاز هر وضعیت)
var APA_EDIT_ID = '';      // شناسهٔ درخواستی که در فرم باز شده است (خالی = درخواست تازه)
var APA_EXEC_STEPS = [];   // گزارش گام‌های اجرای آخرین ابلاغ

/** مقدار عددی یک ورودی (خالی = صفر). */
function apaNum(id) {
  const el = document.getElementById(id);
  if (!el) return 0;
  return Number((el.value || '').replace(/[^0-9.\-]/g, '')) || 0;
}

/** متن یک ورودی. */
function apaText(id) {
  const el = document.getElementById(id);
  return el ? (el.value || '').trim() : '';
}

/** مقدار یک فهرست کشویی. */
function apaSelect(id) {
  const el = document.getElementById(id);
  return el ? el.value : '';
}

/** رتبهٔ یک پلهٔ گروه شغلی از کلید آن (۰ = انتخاب‌نشده). */
function apaRank(key) {
  const grades = (APA_DEFAULTS && APA_DEFAULTS.grades) || [];
  const hit = grades.find(g => g.key === key);
  return hit ? hit.rank : 0;
}

/** عنوان پلهٔ گروه شغلی از رتبه. */
function apaGradeTitle(rank) {
  const grades = (APA_DEFAULTS && APA_DEFAULTS.grades) || [];
  const hit = grades.find(g => g.rank === rank);
  return hit ? hit.titleFa : '—';
}

/** نوع رکورد متناظر با نوع درخواست (ترفیع نوع جدا دارد). */
function apaRecordKind() {
  return apaSelect('apaKind') === 'promotion' ? 'promotions' : 'appointments';
}

/** ساخت بدنهٔ درخواست تصویب‌خواهی از فیلدهای فرم. */
function apaInput() {
  const score = apaNum('apaEvaluation');
  return {
    kind: apaSelect('apaKind') || 'appointment',
    currentRank: apaRank(apaSelect('apaCurrentGrade')),
    proposedRank: apaRank(apaSelect('apaProposedGrade')),
    serviceMonths: apaNum('apaServiceMonths'),
    evaluationScore: score,
    evaluationRecorded: score > 0,
    vacancyAvailable: apaSelect('apaVacancy') === 'yes',
    vacancyChecked: apaSelect('apaVacancy') !== '',
    commitmentMonths: apaNum('apaCommitment'),
    effectiveDate: apaText('apaEffectiveDate'),
    // شرایط احراز: شغل مرجع از شناسنامهٔ شغل و مشخصات احراز متقاضی. سنجش در سرور با
    // همان قاعدهٔ فرم محاسباتی طبقه‌بندی مشاغل انجام می‌شود.
    jobCode: apaSelect('apaJobRef') || '',
    candidateDegreeLevel: apaSelect('apaCandidateDegree') || '',
    candidateField: apaText('apaCandidateField'),
    candidateExperienceYears: apaNum('apaCandidateExperience'),
  };
}

/** محاسبهٔ تصویب‌خواهی با تأخیر کوتاه (برای تایپ). */
function apaPlanSoon() {
  if (APA_TIMER) clearTimeout(APA_TIMER);
  APA_TIMER = setTimeout(apaPlanNow, 250);
}

/** فراخوانی مسیر تصویب‌خواهی و بازترسیم نتیجه. */
async function apaPlanNow() {
  const box = document.getElementById('apaPlan');
  if (box) box.innerHTML = '<div class="muted" style="font-size:.78rem">⏳ در حال سنجش شرط‌ها...</div>';
  const res = await apiFetch('/api/v1/career/appointment-plan', { method: 'POST', body: apaInput() });
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    return;
  }
  APA_RESULT = res.data || {};
  apaRenderPlan();
}

/** نمایش شرط‌ها، زنجیرهٔ تصویب و گام‌های اجرا. */
function apaRenderPlan() {
  const box = document.getElementById('apaPlan');
  if (!box || !APA_RESULT || !APA_RESULT.plan) return;
  const plan = APA_RESULT.plan;

  const checkRows = (plan.checks || []).map(c => {
    const mark = c.passed
      ? '<span class="chip c-ok">✔ برقرار</span>'
      : (c.blocking ? '<span class="chip c-rej">⛔ مانع</span>' : '<span class="chip c-wait">⚠ هشدار</span>');
    return `<tr><td>${esc(c.titleFa)}</td><td>${mark}</td><td style="font-size:.76rem">${esc(c.messageFa)}</td></tr>`;
  });

  const chainRows = (plan.approvals || []).map(s => `<tr>
    <td class="num">${fa(s.order)}</td>
    <td>${esc(s.titleFa)}</td>
    <td class="ltr" style="font-size:.72rem">${esc(s.roleKey || '')}</td>
    <td style="font-size:.74rem">${esc(s.reasonFa || '—')}</td></tr>`);

  const execRows = (APA_RESULT.execution || []).map(s => `<tr>
    <td class="num">${fa(s.order)}</td>
    <td>${esc(s.titleFa)}</td>
    <td style="font-size:.74rem">${esc(s.detailFa || '')}</td></tr>`);

  const allowed = plan.allowed
    ? '<span class="chip c-ok">آمادهٔ تصویب</span>'
    : '<span class="chip c-rej">مانع دارد</span>';
  const jumpText = plan.gradeJump > 0 ? 'به‌اندازهٔ ' + fa(plan.gradeJump) + ' پله بالاتر'
    : (plan.gradeJump === 0 ? 'بدون تغییر پله' : fa(Math.abs(plan.gradeJump)) + ' پله پایین‌تر');

  box.innerHTML = `
    <div class="grid-stats">
      ${statCard({ico: '📌', tint: '#EFF6FF', accent: '#2563EB', val: esc(plan.kindFa || ''), label: 'نوع درخواست'})}
      ${statCard({ico: '🪜', tint: '#EDE9FE', accent: '#7C3AED',
        val: esc(plan.currentGradeFa || '—') + ' ← ' + esc(plan.proposedGradeFa || '—'), label: 'گروه شغلی (' + jumpText + ')'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa((plan.approvals || []).length), label: 'سطح تصویب لازم'})}
      ${statCard({ico: plan.blockingCount ? '⛔' : '🟢', tint: plan.blockingCount ? '#FEE2E2' : '#DCFCE7',
        accent: plan.blockingCount ? '#DC2626' : '#16A34A', val: allowed, label: esc(plan.summaryFa || '')})}
    </div>

    <div class="card card-pad" style="margin-top:10px">
      <div class="card-title">شرط‌های تصویب</div>
      ${liveTable(['شرط', 'نتیجه', 'توضیح'], checkRows, checkRows.length)}
    </div>

    <div class="card card-pad" style="margin-top:10px">
      <div class="card-title">زنجیرهٔ تصویب</div>
      ${liveTable(['ترتیب', 'سطح', 'نقش', 'چرا لازم است'], chainRows, chainRows.length)}
      <div class="muted" style="font-size:.74rem;margin-top:6px">${esc(APA_DEFAULTS.chainNoteFa || '')}</div>
    </div>

    <div class="card card-pad" style="margin-top:10px">
      <div class="card-title">گام‌های اجرا پس از تصویب</div>
      ${liveTable(['گام', 'اقدام', 'شرح'], execRows, execRows.length)}
      ${APA_RESULT.commitmentEnd ? `<div class="muted" style="font-size:.76rem;margin-top:6px">
        پایان تعهد خدمت با تقویم شمسی: <b class="num">${fa(APA_RESULT.commitmentEnd)}</b></div>` : ''}
    </div>`;
}

/* --------------------------- اقدام‌های درخواست --------------------------- */

/** ذخیرهٔ درخواست: یک رکورد انتصاب یا ترفیع با همان فیلدهای فرم. */
async function apaSave() {
  if (!APA_RESULT || !APA_RESULT.plan) { toast('ابتدا شرط‌ها سنجیده شود', 'error'); return; }
  const employeeId = apaSelect('apaEmployee');
  if (!employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  const plan = APA_RESULT.plan;
  if (!plan.allowed) {
    toast('درخواست مانع دارد؛ ابتدا مانع‌ها را برطرف کنید', 'error');
    return;
  }
  const kind = apaSelect('apaKind');
  const recKind = apaRecordKind();
  const proposedTitle = apaPositionTitle() || apaText('apaPosition');
  const employeeName = apaEmployeeName();

  const base = {
    unit: apaText('apaUnit'),
    // کد پست پیشنهادی: اجرای ابلاغ با همین کد، تصدی پست را در تشکیلات ثبت می‌کند.
    positionCode: apaSelect('apaPosition'),
    currentGrade: apaSelect('apaCurrentGrade'),
    proposedGrade: apaSelect('apaProposedGrade'),
    serviceMonths: String(apaNum('apaServiceMonths')),
    evaluationScore: String(apaNum('apaEvaluation')),
    vacancy: apaSelect('apaVacancy'),
    commitment: String(apaNum('apaCommitment')),
    effectiveDate: apaText('apaEffectiveDate'),
    decreeNo: apaText('apaDecreeNo'),
    reason: apaText('apaReason'),
    approvalLevels: (plan.approvals || []).map(s => s.titleFa).join(' ← '),
    // شرایط احراز سنجیده‌شده (سند تصویب باید بازتولیدشدنی باشد).
    jobCode: apaSelect('apaJobRef') || '',
    candidateDegree: apaSelect('apaCandidateDegree') || '',
    candidateField: apaText('apaCandidateField'),
    candidateExperience: String(apaNum('apaCandidateExperience')),
    requireDegree: (apaJobRef() || {}).requireDegree || '',
    requireFields: (apaJobRef() || {}).requireFields || '',
    requireMinExperience: String((apaJobRef() || {}).minExperience || 0),
    qualificationVerdict: (plan.qualification || {}).verdictFa || '',
  };
  const fields = recKind === 'promotions'
    ? Object.assign({}, base, { fromPosition: apaText('apaCurrentPosition'), toPosition: proposedTitle })
    : Object.assign({}, base, {
      currentPosition: apaText('apaCurrentPosition'), currentUnit: apaText('apaCurrentUnit'),
      position: proposedTitle,
    });

  const gradeTitle = plan.proposedGradeFa || apaGradeTitle(apaRank(apaSelect('apaProposedGrade')));
  const title = (plan.kindFa || 'درخواست') + ' — ' + employeeName + ' — ' + gradeTitle;
  const editing = APA_EDIT_ID !== '';
  const res = await apiFetch('/api/v1/career/records' + (editing ? '/' + encodeURIComponent(APA_EDIT_ID) : ''), {
    method: editing ? 'PATCH' : 'POST',
    body: {
      kind: recKind, title: title, employeeId: employeeId, fields: fields,
      noteFa: 'تصویب‌خواهی: ' + (plan.summaryFa || '') + ' — دلیل: ' + apaText('apaReason'),
    },
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = (res.data && res.data.id) || '';
  toast(editing ? 'درخواست اصلاح شد: ' + id : 'درخواست ثبت شد: ' + id);
  apaShowSaved(id);
  apaLoadRecords();
}

/** بارگذاری یک درخواست موجود در فرم تا بتوان تکمیل و دوباره ثبتش کرد.
 *
 *  چرا لازم است: درخواستی که از راه «رکورد تازه» یا بذر اولیه ساخته شده ممکن است گروه
 *  شغلی، مظرفیت پست یا «دلیل و ملاک تصویب» نداشته باشد؛ بی اینبارگذاری، آن درخواست تا
 *  ابد در وضعیت پیش‌نویس می‌ماند، چون گردش‌کار ارسال برای تصویب همان‌ها را می‌خواهد. */
function apaLoadRecord(id) {
  const rec = APA_RECORDS.find(r => r.id === id);
  if (!rec) { toast('درخواست یافت نشد', 'error'); return; }
  const f = rec.fields || {};
  const kind = rec.kind === 'promotions' ? 'promotion' : 'appointment';
  const set = (id, value) => { const el = document.getElementById(id); if (el) el.value = value; };
  set('apaKind', kind);
  set('apaEmployee', rec.employeeId || '');
  set('apaCurrentUnit', f.currentUnit || '');
  set('apaCurrentPosition', f.currentPosition || f.fromPosition || '');
  set('apaUnit', f.unit || '');
  set('apaCurrentGrade', f.currentGrade || '');
  set('apaProposedGrade', f.proposedGrade || '');
  set('apaServiceMonths', f.serviceMonths || '0');
  set('apaEvaluation', f.evaluationScore || '0');
  set('apaVacancy', f.vacancy || '');
  set('apaCommitment', f.commitment || '0');
  set('apaEffectiveDate', f.effectiveDate || '');
  set('apaDecreeNo', f.decreeNo || '');
  set('apaReason', f.reason || '');
  // پست از روی عنوان یا کد در فهرست پست‌ها پیدا می‌شود؛ select با کد کار می‌کند.
  const wanted = f.positionCode || '';
  const title = (f.position || f.toPosition || '').trim();
  const hit = APA_POSITIONS.find(p => p.code === wanted)
    || APA_POSITIONS.find(p => positionTitleOf(p.titleFa) === title);
  set('apaPosition', hit ? hit.code : (APA_POSITIONS[0] ? APA_POSITIONS[0].code : ''));
  APA_EDIT_ID = id;
  const box = document.getElementById('apaEditing');
  if (box) {
    box.innerHTML = `<span class="chip c-wait">✎ در حال ویرایش درخواست <b class="ltr">${esc(id)}</b></span>`;
  }
  apaPlanNow();
  toast('درخواست ' + id + ' در فرم باز شد؛ پس از اصلاح، «ذخیرهٔ تغییرات» را بزنید');
}

/** عنوان پست بدون کد انتهای آن («سرپرست فروش (POS-405)» ← «سرپرست فروش»). */
function positionTitleOf(label) {
  return String(label || '').replace(/\s*\([^)]*\)\s*$/, '').trim();
}

/** جعبهٔ پس از ثبت: گام بعدی همان گردش‌کار است. */
function apaShowSaved(id) {
  const box = document.getElementById('apaSaved');
  if (!box) return;
  box.innerHTML = `<div class="card card-pad" style="margin-top:10px;font-size:.82rem;line-height:2">
    درخواست <b class="ltr">${esc(id)}</b> ثبت شد. گام بعدی: ارسال برای تصویب، سپس تصویب سطوح زنجیره و اجرای ابلاغ.
    <div class="row-actions" style="margin-top:8px">
      <button class="btn btn-sm btn-primary" onclick="apaAct('${esc(id)}','submit')">📤 ارسال برای تصویب</button>
      <button class="btn btn-sm btn-ghost" onclick="apaAct('${esc(id)}','approve')">✅ تصویب</button>
      <button class="btn btn-sm btn-ghost" onclick="apaExecute('${esc(id)}')">📜 اجرا و ابلاغ</button>
      <button class="btn btn-sm btn-ghost" onclick="go('appointmentForm')">↻ فرم تازه</button>
    </div></div>`;
}

/** انجام یک اقدام گردش‌کار روی درخواست (ارسال/تصویب/لغو). */
async function apaAct(id, action) {
  const res = await apiFetch('/api/v1/career/records/' + encodeURIComponent(id) + '/action',
    { method: 'POST', body: { action: action } });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('اقدام انجام شد: ' + apaActionTitle(action));
  apaLoadRecords();
}

/** اجرا و ابلاغ: سرور شرط‌ها را با پارامترهای شرکت دوباره می‌سنجد و گام‌های اجرا را
 *  واقعاً انجام می‌دهد (پروندهٔ کارمند، تصدی پست در تشکیلات، اعلام به حقوق). */
async function apaExecute(id) {
  const res = await apiFetch('/api/v1/career/appointment-execute', { method: 'POST', body: { id: id } });
  if (!res.ok) { toast(res.message, 'error'); return; }
  APA_EXEC_STEPS = (res.data && res.data.steps) || [];
  const pending = APA_EXEC_STEPS.filter(s => !s.done);
  toast(pending.length
    ? 'ابلاغ اجرا شد، ولی ' + pending.length + ' گام ناتمام ماند — گزارش اجرا را ببینید'
    : 'ابلاغ صادر و اجرا شد: ' + id);
  apaRenderSteps(id);
  apaLoadRecords();
}

/** امضای سطح بعدی تصویب: سطح و نقشش را سرور تعیین می‌کند. */
async function apaSign(id) {
  const res = await apiFetch('/api/v1/career/appointment-sign', { method: 'POST', body: { id: id } });
  if (!res.ok) { toast(res.message || (res.data && res.data.message) || 'امضا انجام نشد', 'error'); return; }
  toast((res.data && res.data.noteFa) || ('سطح تصویب امضا شد: ' + id));
  apaLoadRecords();
}

/** بازگشت برای اصلاح: دلیل بازگشت الزامی است و در سابقهٔ رکورد می‌ماند. */
async function apaReturn(id) {
  const reason = (window.prompt('دلیل بازگشت برای اصلاح را بنویسید:') || '').trim();
  if (!reason) { toast('بازگشت بدون دلیل انجام نمی‌شود', 'error'); return; }
  const res = await apiFetch('/api/v1/career/appointment-return', { method: 'POST', body: { id: id, noteFa: reason } });
  if (!res.ok) { toast(res.message || 'بازگشت انجام نشد', 'error'); return; }
  toast('درخواست به پیش‌نویس بازگشت: ' + id);
  apaLoadRecords();
}

/** ابطال ابلاغ اجراشده: تصدی تشکیلات بسته و پروندهٔ کارمند برمی‌گردد. */
async function apaRevoke(id) {
  const reason = (window.prompt('دلیل ابطال ابلاغ را بنویسید:') || '').trim();
  if (!reason) { toast('ابطال بدون دلیل انجام نمی‌شود', 'error'); return; }
  const res = await apiFetch('/api/v1/career/appointment-revoke', { method: 'POST', body: { id: id, noteFa: reason } });
  if (!res.ok) { toast(res.message || 'ابطال انجام نشد', 'error'); return; }
  toast('ابلاغ باطل شد: ' + id);
  apaLoadRecords();
}

/** گزارش گام‌های اجرای ابلاغ: چه چیزی واقعاً انجام شد و چه چیزی نه. */
function apaRenderSteps(id) {
  const box = document.getElementById('apaExec');
  if (!box) return;
  if (!APA_EXEC_STEPS.length) { box.replaceChildren(); return; }
  const rows = APA_EXEC_STEPS.map(s => {
    const mark = s.done
      ? '<span class="chip c-ok">✔ انجام شد</span>'
      : '<span class="chip c-wait">⚠ ناتمام</span>';
    return `<tr><td class="num">${fa(s.order)}</td><td>${esc(s.titleFa)}</td><td>${mark}</td>
      <td style="font-size:.76rem">${esc(s.done ? (s.detailFa || '—') : (s.messageFa || '—'))}</td></tr>`;
  });
  box.innerHTML = `<div class="card card-pad" style="margin-top:10px">
    <div class="card-title">📜 گزارش اجرای ابلاغ <b class="ltr">${esc(id)}</b></div>
    ${liveTable(['گام', 'اقدام', 'نتیجه', 'شرح'], rows, rows.length)}
  </div>`;
}

/** عنوان فارسی اقدام‌ها. */
function apaActionTitle(action) {
  const map = { submit: 'ارسال برای تصویب', approve: 'تصویب', issue: 'اجرا و ابلاغ', cancel: 'لغو' };
  return map[action] || action;
}

/** بارگذاری درخواست‌های جاری (انتصاب و ترفیع) و نمایش جدول. */
async function apaLoadRecords() {
  const [listRes, schemaRes] = await Promise.all([
    apiFetch('/api/v1/career/records?size=100'),
    APA_SCHEMA ? Promise.resolve({ ok: true, data: APA_SCHEMA }) : apiFetch('/api/v1/career/schema'),
  ]);
  if (schemaRes.ok) APA_SCHEMA = schemaRes.data;
  if (!listRes.ok) return;
  const items = (listRes.data && listRes.data.items) || [];
  APA_RECORDS = items.filter(r => r.kind === 'appointments' || r.kind === 'promotions');
  apaRenderRecords();
}

/** اقدام‌های مجاز یک رکورد از شناسهٔ نوع و وضعیت فعلی آن. */
function apaLegalActions(rec) {
  const kind = ((APA_SCHEMA && APA_SCHEMA.kinds) || []).find(k => k.key === rec.kind);
  if (!kind) return [];
  const allowed = (kind.transitions || {})[rec.status] || [];
  return (kind.actions || []).filter(a => !a.to || allowed.includes(a.to));
}

/** جدول درخواست‌های جاری با دکمهٔ اقدام مجاز هر وضعیت. */
function apaRenderRecords() {
  const box = document.getElementById('apaRecords');
  if (!box) return;
  const rows = APA_RECORDS.map(r => {
    // هر اقدام قاعده‌سنج به مسیر خودش وصل است: امضای سطح، بازگشت برای اصلاح، اجرا و ابطال.
    const handlers = {
      sign: `apaSign('${esc(r.id)}')`,
      return: `apaReturn('${esc(r.id)}')`,
      issue: `apaExecute('${esc(r.id)}')`,
      revoke: `apaRevoke('${esc(r.id)}')`,
    };
    const btns = apaLegalActions(r).map(a => {
      const call = handlers[a.value] || `apaAct('${esc(r.id)}','${esc(a.value)}')`;
      const style = a.value === 'issue' || a.value === 'sign' ? 'btn-primary'
        : (a.style === 'primary' ? 'btn-primary' : (a.style === 'danger' ? 'btn-ghost' : 'btn-ghost'));
      return `<button class="btn btn-sm ${style}" onclick="${call}">${esc(a.labelFa)}</button>`;
    }).join(' ');
    // پس از تصویب، دکمهٔ «اجرا و ابلاغ» از همان فهرست اقدام‌های مجاز می‌آید و به مسیر
    // قاعده‌سنج وصل است (نه اقدام خام گردش‌کار). فقط «تکمیل در فرم» را اینجا اضافه
    // می‌کنیم: پیش‌نویسی که از راه رکورد تازه ساخته شده، بی آن هرگز از پیش‌نویس بیرون نمی‌آید.
    const editBtn = r.status === 'draft'
      ? `<button class="btn btn-sm btn-ghost" onclick="apaLoadRecord('${esc(r.id)}')">✎ تکمیل در فرم</button>` : '';
    return `<tr>
      <td class="ltr" style="font-size:.74rem">${esc(r.id)}</td>
      <td>${esc(r.title)}</td>
      <td>${esc(r.employeeName || '—')}</td>
      <td>${statusChip(r.status, r.statusFa)}</td>
      <td class="num">${fa((r.fields || {}).effectiveDate || '—')}</td>
      <td class="num">${apaProgressFa(r)}</td>
      <td>${[btns, editBtn].filter(Boolean).join(' ') || '<span class="muted">—</span>'}</td></tr>`;
  });
  const head = ['شناسه', 'عنوان', 'کارمند', 'وضعیت', 'تاریخ اجرا', 'پیشرفت تصویب', 'اقدام'];
  box.innerHTML = rows.length ? liveTable(head, rows, rows.length)
    : '<div class="muted" style="font-size:.78rem">درخواستی ثبت نشده است.</div>';
}

/** پیشرفت تصویب یک درخواست: «۲ از ۴ سطح» یا خط تیره وقتی زنجیره‌ای ثبت نشده است. */
function apaProgressFa(rec) {
  const fields = rec.fields || {};
  const total = Number(fields.approvalTotal || 0);
  const progress = Number(fields.approvalProgress || 0);
  if (!total) return '—';
  return fa(progress) + ' از ' + fa(total);
}

/* --------------------------- کمک‌های فهرست‌ها --------------------------- */

/** نام کارمند انتخاب‌شده. */
function apaEmployeeName() {
  const id = apaSelect('apaEmployee');
  const hit = APA_EMPLOYEES.find(e => e.id === id);
  return hit ? hit.titleFa.split(' — ')[0] : id;
}

/** پست انتخاب‌شده در فهرست. */
function apaSelectedPosition() {
  const code = apaSelect('apaPosition');
  return APA_POSITIONS.find(p => p.code === code) || null;
}

/** عنوان پست پیشنهادی. */
function apaPositionTitle() {
  const p = apaSelectedPosition();
  return p ? p.titleFa.replace(/\s*\([^)]*\)\s*$/, '') : '';
}

/** با انتخاب کارمند، پست و واحد فعلی پر می‌شود (فقط پرکردن؛ قاعده‌ای نیست). */
function apaOnEmployee() {
  const id = apaSelect('apaEmployee');
  const hit = APA_EMPLOYEES.find(e => e.id === id);
  if (hit) {
    const pos = document.getElementById('apaCurrentPosition');
    const unit = document.getElementById('apaCurrentUnit');
    if (pos && !pos.value) pos.value = hit.position || '';
    if (unit && !unit.value) unit.value = hit.unit || '';
  }
  apaPlanSoon();
}

/** با انتخاب پست پیشنهادی، واحد و گروه شغلی پیشنهادی از تشکیلات پر می‌شود. */
function apaOnPosition() {
  const p = apaSelectedPosition();
  if (p) {
    const unit = document.getElementById('apaUnit');
    if (unit) unit.value = p.unit || '';
    const grades = (APA_DEFAULTS && APA_DEFAULTS.grades) || [];
    const match = grades.find(g => g.titleFa === (p.grade || '').trim());
    if (match) {
      const sel = document.getElementById('apaProposedGrade');
      if (sel) sel.value = match.key;
    }
    const vacancy = document.getElementById('apaVacancy');
    if (vacancy) vacancy.value = p.status === 'vacant' ? 'yes' : (p.vacancy === 'بله' ? 'yes' : 'no');
  }
  apaPlanSoon();
}

/* ------------------------------ ساخت فرم ------------------------------ */

/** یک ردیف ورودی عددی/متنی. */
function apaField(id, label, value, unit, hint, type) {
  const cls = type === 'text' ? '' : ' class="num"';
  return `<div class="field"><label>${esc(label)}${unit ? ` <span class="muted">(${esc(unit)})</span>` : ''}</label>
    <input id="${id}"${cls} value="${esc(value)}" placeholder="${esc(hint || '')}" oninput="apaPlanSoon()"></div>`;
}

/** فهرست کشویی از گزینه‌های سرور. */
function apaOptions(id, list, selected, onChange, labelKey) {
  const key = labelKey || 'titleFa';
  return `<select id="${id}" ${onChange ? `onchange="${onChange}"` : 'onchange="apaPlanSoon()"'}>
    ${(list || []).map(o => `<option value="${esc(o.key || o.code)}" ${(o.key || o.code) === selected ? 'selected' : ''}>${esc(o[key] || o.titleFa || o.key)}</option>`).join('')}
  </select>`;
}

/** صفحهٔ «فرم تصویب‌خواهی انتصاب و ترفیع». */
function pageAppointmentFormLive() {
  if (!isLive()) return offlineFallback('appointmentForm');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/career/appointment-defaults');
    if (!res.ok) return liveErrorBox(res.message);
    APA_DEFAULTS = res.data || {};
    APA_EMPLOYEES = APA_DEFAULTS.employees || [];
    APA_POSITIONS = APA_DEFAULTS.positions || [];
    const def = APA_DEFAULTS;
    const grades = def.grades || [];
    const empOpts = APA_EMPLOYEES.map(e => `<option value="${esc(e.id)}">${esc(e.titleFa)}</option>`).join('');
    const posOpts = APA_POSITIONS.map(p =>
      `<option value="${esc(p.code)}">${esc(p.titleFa)} — ${esc(p.unit || '')}</option>`).join('');
    const midGrades = grades.map(g => `<option value="${esc(g.key)}">${esc(g.titleFa)}</option>`).join('');
    const jobList = def.jobs || [];
    // نخستین شناسنامهٔ تأییدشده به‌عنوان شغل مرجع پیش‌فرض (شناسنامهٔ پیش‌نویس مبنای سنجش نیست).
    const firstApprovedJob = (jobList.find(j => j.approved) || {});
    const defaultJob = firstApprovedJob.code || firstApprovedJob.titleFa || '';
    const jobOpts = `<option value="">— بدون شغل مرجع —</option>` + jobList.map(j =>
      `<option value="${esc(j.code || j.titleFa)}" data-degree="${esc(j.requireDegree || '')}"
        data-fields="${esc(j.requireFields || '')}" data-exp="${esc(String(j.minExperience || 0))}">
        ${esc(j.titleFa)}${j.approved ? '' : ' (پیش‌نویس)'}</option>`).join('');
    const degreeOpts = (def.degreeLevels || []).map(d =>
      `<option value="${esc(d.key)}">${esc(d.titleFa)}</option>`).join('');
    const defaultCurrent = (grades[1] && grades[1].key) || '';
    const defaultProposed = (grades[2] && grades[2].key) || '';

    const form = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">📝 مشخصات درخواست</div>
      <div class="form-grid">
        <div class="field"><label>نوع درخواست <span class="req">*</span></label>
          ${apaOptions('apaKind', def.kinds, 'appointment', 'apaPlanSoon()')}</div>
        <div class="field"><label>کارمند <span class="req">*</span></label>
          <select id="apaEmployee" onchange="apaOnEmployee()">${empOpts}</select></div>
        <div class="field"><label>واحد فعلی</label><input id="apaCurrentUnit" oninput="apaPlanSoon()"></div>
        <div class="field"><label>پست فعلی</label><input id="apaCurrentPosition" oninput="apaPlanSoon()"></div>
        <div class="field"><label>پست پیشنهادی <span class="req">*</span></label>
          <select id="apaPosition" onchange="apaOnPosition()">${posOpts}</select></div>
        <div class="field"><label>واحد سازمانی پیشنهادی <span class="req">*</span></label>
          <input id="apaUnit" oninput="apaPlanSoon()"></div>
        <div class="field"><label>گروه شغلی فعلی <span class="req">*</span></label>
          <select id="apaCurrentGrade" onchange="apaPlanSoon()">${midGrades}</select></div>
        <div class="field"><label>گروه شغلی پیشنهادی <span class="req">*</span></label>
          <select id="apaProposedGrade" onchange="apaPlanSoon()">${midGrades}</select></div>
        ${apaField('apaServiceMonths', 'سابقهٔ گروه فعلی', '0', 'ماه', 'مثلاً ۳۰')}
        ${apaField('apaEvaluation', 'امتیاز ارزیابی عملکرد', '0', 'از ۱۰۰', 'خالی = ثبت نشده')}
        <div class="field"><label>ظرفیت پست پیشنهادی خالی است</label>
          <select id="apaVacancy" onchange="apaPlanSoon()"><option value="">— بررسی نشده —</option>
            <option value="yes">بله</option><option value="no">خیر</option></select></div>
        ${apaField('apaCommitment', 'تعهد خدمت', '0', 'ماه', 'برای گروه‌های مدیریتی')}
        ${apaField('apaEffectiveDate', 'تاریخ اجرا (شمسی) ✻', def.today || '', '1405/09/01', '', 'text')}
        ${apaField('apaDecreeNo', 'شمارهٔ ابلاغ', '', 'ABL-1405-101', 'برای اجرا الزامی', 'text')}
        <div class="field full"><label>دلیل و ملاک تصویب <span class="req">*</span></label>
          <input id="apaReason" placeholder="مثلاً کسب امتیاز ارزیابی ۸۵ و خالی‌بودن پست مطابق تشکیلات"></div>
      </div>
      <div class="card-title" style="margin:14px 0 6px">🎓 شرایط احراز شغل</div>
      <div class="form-grid">
        <div class="field"><label>شغل مرجع (شناسنامهٔ شغل)</label>
          <select id="apaJobRef" onchange="apaOnJobRef()">${jobOpts}</select>
          <span class="muted" style="font-size:.72rem">شرط مدرک، رشته و سابقه از شناسنامهٔ شغل تأییدشده خوانده می‌شود.</span></div>
        <div class="field"><label>مقطع تحصیلی متقاضی</label>
          <select id="apaCandidateDegree" onchange="apaPlanSoon()"><option value="">— ثبت نشده —</option>${degreeOpts}</select></div>
        <div class="field"><label>رشتهٔ تحصیلی متقاضی</label>
          <input id="apaCandidateField" oninput="apaPlanSoon()" placeholder="مثلاً مدیریت بازرگانی"></div>
        ${apaField('apaCandidateExperience', 'سابقهٔ مرتبط متقاضی', '0', 'سال', 'مبنای شرایط معادل')}
        <div class="field full"><label>شرط شغل انتخاب‌شده</label>
          <div id="apaJobReq" class="muted" style="font-size:.78rem">—</div></div>
      </div>
      <div class="muted" style="font-size:.74rem;margin-top:6px">${esc(def.noteFa || '').split('\n').join('<br>')}</div>
      <div class="muted" style="font-size:.74rem;margin-top:4px">${esc(def.qualificationNoteFa || '')}</div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">🧮 تصویب‌خواهی</div>
      <div id="apaPlan"><div class="muted" style="font-size:.78rem">در حال سنجش شرط‌ها...</div></div>
      <div class="row-actions" style="margin-top:10px">
        <button class="btn btn-primary" onclick="apaPlanNow()">🧮 سنجش مجدد</button>
        <button class="btn btn-ghost" onclick="apaSave()">💾 ثبت درخواست / ذخیرهٔ تغییرات</button>
        <button class="btn btn-ghost" onclick="APA_EDIT_ID='';go('appointmentForm')">🗒 درخواست تازه</button>
      </div>
      <div id="apaEditing" style="margin-top:8px"></div>
      <div id="apaSaved"></div>
      <div id="apaExec"></div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">📋 درخواست‌های جاری انتصاب و ترفیع</div>
      <div id="apaRecords"><div class="muted" style="font-size:.78rem">در حال خواندن...</div></div>
    </div>`;

    const paramRows = (def.editableParams || []).map(p => `<tr>
      <td>${esc(p.titleFa)}<div class="muted ltr" style="font-size:.68rem">${esc(p.key)}</div></td>
      <td><span class="num">${fa(p.value)}</span></td>
      <td>${esc(p.unitFa)}</td>
      <td>${paramSourceChip(p)}</td>
      <td><input class="inp num" id="apaParam_${esc(p.key)}" style="max-width:110px" value="${esc(String(p.value))}"></td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-primary" onclick="apaParamSave('${esc(p.key)}')">💾 ثبت</button>
        ${p.source === 'record'
          ? `<button class="btn btn-sm btn-ghost" onclick="apaParamRemove('${esc(p.key)}')">↩ بازگشت به پیش‌فرض</button>` : ''}
      </div></td></tr>`);

    const shell = form + (paramRows.length ? `<div class="card card-pad">
        <div class="card-title">⚙️ پارامترهای انتصاب و ترفیع — قابل ویرایش از همین فرم</div>
        <div class="muted" style="font-size:.76rem;line-height:2;margin-bottom:8px">${esc(def.paramsNoteFa || '')}</div>
        ${liveTable(['پارامتر', 'مقدار مؤثر', 'واحد', 'منبع', 'مقدار تازه', 'اقدام'], paramRows, paramRows.length)}
        <div style="display:flex;gap:10px;flex-wrap:wrap;align-items:flex-end;margin-top:10px">
          <div class="field" style="min-width:260px"><label>افزودن/تغییر پارامتر دیگر</label>
            <select id="apaParamKey">${(def.editableParams || []).map(p =>
              `<option value="${esc(p.key)}">${esc(p.titleFa)}</option>`).join('')}</select></div>
          <div class="field"><label>مقدار</label><input class="inp num" id="apaParamValue" placeholder="مثلاً 24"></div>
          <div class="field" style="min-width:220px"><label>دلیل تغییر</label><input class="inp" id="apaParamReason" placeholder="مثلاً مصوبهٔ کمیتهٔ انتصابات"></div>
          <button class="btn btn-primary" onclick="apaParamAdd()">➕ ثبت مقدار تازه</button>
        </div>
        <div id="apaParamMsg" style="margin-top:8px"></div>
      </div>` : '');

    // مقدار پیش‌فرض گروه‌ها و نخستین شناسنامهٔ تأییدشده، سپس سنجش شرط‌ها.
    setTimeout(() => {
      const cur = document.getElementById('apaCurrentGrade');
      const prop = document.getElementById('apaProposedGrade');
      if (cur && defaultCurrent) cur.value = defaultCurrent;
      if (prop && defaultProposed) prop.value = defaultProposed;
      // شغل مرجع پیش‌فرض: نخستین شناسنامهٔ تأییدشده تا شرط احراز از ابتدا در سنجش باشد
      // (کاربر می‌تواند «بدون شغل مرجع» را انتخاب کند و سنجش تحصیلی انجام نشود).
      const jobSel = document.getElementById('apaJobRef');
      if (jobSel && !jobSel.value && defaultJob) jobSel.value = defaultJob;
      apaOnEmployee();
      apaOnJobRef();
      apaPlanNow();
      apaLoadRecords();
    }, 60);
    return shell;
  });
  return liveShell('فرم تصویب‌خواهی انتصاب و ترفیع',
    'سنجش شرط‌ها، زنجیرهٔ تصویب و اجرای ابلاغ در یک جا',
    `<button class="btn btn-ghost" onclick="APA_DEFAULTS=null;liveReload()">↻ تازه‌سازی پارامترها</button>`);
}

/* ثبت در مسیریاب زنده. */
LIVE_PAGES.appointmentForm = pageAppointmentFormLive;

/* ------------------- شرط شغل و پارامترهای قابل ویرایش ------------------- */

/** شغل مرجع انتخاب‌شده از فهرست شناسنامه‌ها. */
function apaJobRef() {
  const code = apaSelect('apaJobRef');
  if (!code) return null;
  return ((APA_DEFAULTS && APA_DEFAULTS.jobs) || []).find(j => (j.code || j.titleFa) === code) || null;
}

/** با انتخاب شغل مرجع، شرط احراز همان شغل نشان داده و در سنجش به کار می‌رود. */
function apaOnJobRef() {
  const job = apaJobRef();
  const box = document.getElementById('apaJobReq');
  if (box) {
    box.innerHTML = job
      ? `${esc(job.requirementFa || '—')}${job.approved ? '' : ' <span class="chip c-wait">شناسنامه پیش‌نویس — مبنای انتصاب نیست</span>'}`
      : 'شغل مرجع انتخاب نشده است؛ شرط تحصیلی سنجیده نمی‌شود.';
  }
  apaPlanSoon();
}

/** نشان منبع مقدار پارامتر. */
function paramSourceChip(p) {
  if (p.source === 'record') return '<span class="chip c-ok">مقدار ثبت‌شده در فرم</span>';
  if (p.source === 'company') return '<span class="chip c-wait">پارامتر شرکت</span>';
  return '<span class="chip">پیش‌فرض سامانه</span>';
}

/** پیام نتیجهٔ تغییر پارامتر در پنل. */
function apaParamShow(payload, isError) {
  const box = document.getElementById('apaParamMsg');
  if (!box) return;
  box.innerHTML = isError
    ? liveErrorBox(payload)
    : `<div class="card card-pad" style="font-size:.82rem;line-height:2">✅ ${esc(payload || 'ثبت شد')}</div>`;
}

/** ثبت مقدار یک پارامتر از ردیف جدول. */
async function apaParamSave(key) {
  const el = document.getElementById('apaParam_' + key);
  await apaParamSubmit({key, value: el ? el.value.trim() : '', reasonFa: ''});
}

/** کنارگذاشتن مقدار ثبت‌شده و بازگشت به پارامتر شرکت/پیش‌فرض. */
async function apaParamRemove(key) {
  await apaParamSubmit({key, remove: true, reasonFa: 'بازگشت به مقدار شرکت/پیش‌فرض از فرم انتصابات.'});
}

/** افزودن/تغییر پارامتر از کادر پایین پنل. */
async function apaParamAdd() {
  const key = (document.getElementById('apaParamKey') || {}).value || '';
  const value = ((document.getElementById('apaParamValue') || {}).value || '').trim();
  const reasonFa = ((document.getElementById('apaParamReason') || {}).value || '').trim();
  if (!key || !value) { toast('کلید و مقدار پارامتر را وارد کنید', 'error'); return; }
  await apaParamSubmit({key, value, reasonFa});
}

/** فراخوانی سرور برای ثبت/کنارگذاشتن پارامتر و سپس بازخوانی فرم. */
async function apaParamSubmit(body) {
  const res = await apiFetch('/api/v1/career/appointment-params', {method: 'POST', body});
  if (!res.ok) {
    apaParamShow(res.message, true);
    toast(res.message, 'error');
    return;
  }
  APA_DEFAULTS = null;
  toast((res.data && res.data.noteFa) || 'پارامتر ثبت شد');
  go('appointmentForm');
}