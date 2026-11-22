/* =====================================================================
   پروندهٔ ایثارگری و امتیازهای ایثارگران
   ---------------------------------------------------------------------
   این صفحه سه کار می‌کند و هیچ قاعده‌ای در خودش ندارد:

     ۱. فهرست پرونده‌ها، فرم ثبت و اقدام‌های گردشکار (بررسی، تأیید، رد) از موتور
        عمومی رکوردها می‌آید؛ همان جدولی که بالای این پنل دیده می‌شود.
     ۲. «محاسبهٔ امتیاز» را سرور انجام می‌دهد (`/veteran-plan`): معافیت مالیاتی،
        سنوات ارفاقی، احتساب مدت اسارت، یک مقطع تحصیلی بالاتر، پایهٔ تشویقی و
        مرخصی تشویقی. اینجا فقط همان سطرها نمایش داده می‌شود.
     ۳. ثبت پرونده با امتیازهای محاسبه‌شدهٔ سرور (`/veteran-files`) و گزارش جامعهٔ
        ایثارگران (`/veterans-report`).

   نکتهٔ مهم برای کارشناس: امتیازها تا «تأیید پرونده» هیچ اثر مالی ندارند؛ فیش
   حقوق فقط پروندهٔ تأییدشده را می‌بیند و کلید پارامتری `veteran.payroll.apply`
   می‌تواند اعمال در فیش را یکجا خاموش کند.
   ===================================================================== */

window.VET = window.VET || {employeeId: '', plan: null};

/** پیشوند API ماژول امور رفاهی و ایثارگران. */
function vetPrefix() { return '/api/v1/grievances'; }

/** فهرست‌های مرجع (گروه‌ها، پارامترهای شرکت، متن دستورالعمل). */
function vetDef() { return window.VET_DEF || {}; }

/** گروه انتخاب‌شده از فرم. */
function vetSelectedGroup() {
  const el = document.getElementById('vetGroup');
  return el ? el.value : '';
}

/** یادداشت گروه انتخاب‌شده (از سرور می‌آید، نه فهرست ثابت در رابط). */
function vetGroupNoteHTML() {
  const group = (vetDef().groups || []).find(g => g.key === vetSelectedGroup());
  if (!group || !group.noteFa) return '';
  return `<div class="muted" style="font-size:.76rem;margin-top:4px">${esc(group.noteFa)}</div>`;
}

/** به‌روزرسانی یادداشت گروه با تغییر انتخاب. */
function vetGroupChanged() {
  const box = document.getElementById('vetGroupNote');
  if (box) box.innerHTML = vetGroupNoteHTML();
}

/** پنل امور ایثارگران: آمار، فرم محاسبه، گزارش و متن دستورالعمل. */
function veteransPanelHTML(view) {
  const def = view.def || {};
  const report = view.report || {};
  const stats = `<div class="grid-stats">
    ${statCard({ico: '🎖️', tint: '#EFF6FF', accent: '#2563EB', val: fa(report.total || 0), label: 'پروندهٔ ایثارگری ثبت‌شده'})}
    ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(report.approved || 0), label: 'پروندهٔ تأییدشده (با اثر مالی)'})}
    ${statCard({ico: '🧾', tint: '#FEF3C7', accent: '#D97706', val: fa(report.taxExemptCount || 0), label: 'مشمول معافیت مالیاتی'})}
    ${statCard({ico: '📅', tint: '#EDE9FE', accent: '#7C3AED', val: fa(report.extraMonthsTotal || 0), label: 'جمع افزایش سنواتی (ماه)'})}
  </div>`;
  return stats + vetFormHTML(view) + '<div id="vetPlanBox"></div>' +
    vetMembersHTML(view) +
    '<div id="vetCommitteeForm">' + vetCommitteeFormHTML(view) + '</div>' +
    vetCommitteeTableHTML(view) + '<div id="vetCommitteeResult"></div>' +
    vetProfileBoxHTML(view.profile) + vetReportHTML(report) + vetParamsHTML(def) +
    vetInstructionHTML(def) + statNote(def.noteFa || '');
}

/* ------------------------------ فرم محاسبه ------------------------------ */

/** فرم ثبت/محاسبهٔ پروندهٔ ایثارگری. */
function vetFormHTML(view) {
  const def = view.def || {};
  const groups = (def.groups || []).map(g =>
    `<option value="${esc(g.key)}">${esc(g.titleFa)}</option>`).join('');
  const yesNo = (def.yesNo || []).map(o =>
    `<option value="${esc(o.key)}">${esc(o.titleFa)}</option>`).join('');
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🎖️ محاسبهٔ امتیاز ایثارگری و ثبت پرونده</div>
    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px">
      <div class="field"><label>کارمند</label>${view.employeeSelect || '<select id="vetEmployee"></select>'}</div>
      <div class="field"><label>گروه ایثارگری ✻</label>
        <select id="vetGroup" onchange="vetGroupChanged()">${groups}</select>
        <div id="vetGroupNote">${vetGroupNoteHTML()}</div></div>
      <div class="field"><label>درصد جانبازی</label><input id="vetPercent" class="num" placeholder="مثلاً 25"></div>
      <div class="field"><label>مدت اسارت (ماه)</label><input id="vetCaptivity" class="num" placeholder="مثلاً 24"></div>
      <div class="field"><label>حضور در جبهه (ماه)</label><input id="vetFront" class="num" placeholder="مثلاً 13"></div>
      <div class="field"><label>نسبت با شهید (برای خانوادهٔ شهید)</label><input id="vetRelation" placeholder="پدر، مادر یا همسر"></div>
      <div class="field"><label>شمارهٔ کارت یا حکم ایثارگری</label><input id="vetCardNo"></div>
      <div class="field"><label>مرجع صادرکننده</label><input id="vetIssuer" placeholder="بنیاد شهید و امور ایثارگران"></div>
      <div class="field"><label>تاریخ صدور (شمسی)</label><input id="vetIssueDate" placeholder="1405/08/18"></div>
      <div class="field"><label>پرونده پیش از ثبت تأییدشده شمرده شود؟</label>
        <select id="vetApprovedFile">${yesNo}</select>
        <div class="muted" style="font-size:.72rem;margin-top:4px">پیش‌فرض «خیر» است؛ امتیازها پس از تأیید پرونده در گردشکار اثر مالی می‌گیرند.</div></div>
      <div class="field" style="grid-column:1/-1"><label>سایر امتیازات و توضیح پرونده</label>
        <input id="vetBenefits" placeholder="مثلاً مرخصی تشویقی، سهمیهٔ مسکن"></div>
    </div>
    <div class="row-actions" style="margin-top:8px">
      <button class="btn btn-primary" onclick="vetPlan()">🧮 محاسبهٔ امتیازها</button>
      <button class="btn" onclick="vetSaveFile()">💾 ثبت / به‌روزرسانی پرونده</button>
      <button class="btn btn-ghost" onclick="vetClearForm()">🗑 خالی‌کردن فرم</button>
    </div>
    <div class="muted" style="font-size:.74rem;margin-top:6px">
      هیچ درصدی در این فرم نوشته نمی‌شود: مبنای محاسبه قانون است و مقادیر سیاستی شرکت از
      «امور ایثارگران» در تنظیمات پارامترها خوانده می‌شود. امتیازهای محاسبه‌شده در پرونده ذخیره
      می‌شوند تا سند و فیش همان عدد تأییدشده را نشان دهند.
    </div>
  </div>`;
}

/** بدنهٔ درخواست محاسبه/ثبت از فرم. */
function vetBody() {
  const val = id => {
    const el = document.getElementById(id);
    return el ? el.value.trim() : '';
  };
  return {
    employeeId: val('vetEmployee') || window.VET.employeeId,
    veteranType: val('vetGroup'),
    percent: Number(val('vetPercent') || 0) || 0,
    captivityMonths: Number(val('vetCaptivity') || 0) || 0,
    frontMonths: Number(val('vetFront') || 0) || 0,
    relation: val('vetRelation'),
    cardNo: val('vetCardNo'),
    issuerFa: val('vetIssuer'),
    issueDate: val('vetIssueDate'),
    approvedFile: val('vetApprovedFile') === 'yes',
    benefits: val('vetBenefits'),
    noteFa: 'ثبت از صفحهٔ امور ایثارگران',
  };
}

/** محاسبهٔ پیش‌نمایش امتیازها (بدون تغییر داده). */
async function vetPlan() {
  const body = vetBody();
  if (!body.employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  const box = document.getElementById('vetPlanBox');
  if (box) box.innerHTML = '<div class="card card-pad muted">⏳ در حال محاسبهٔ امتیازها...</div>';
  const res = await apiFetch(vetPrefix() + '/veteran-plan', {method: 'POST', body});
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    toast(res.message, 'error');
    return;
  }
  window.VET.plan = res.data || {};
  window.VET.employeeId = body.employeeId;
  if (box) box.innerHTML = vetPlanHTML(window.VET.plan);
  toast('محاسبهٔ امتیازها انجام شد');
}

/** ثبت یا به‌روزرسانی پرونده با امتیازهای محاسبه‌شدهٔ سرور. */
async function vetSaveFile() {
  const body = vetBody();
  if (!body.employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  const res = await apiFetch(vetPrefix() + '/veteran-files', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const rec = (res.data && res.data.record) || {};
  toast('پروندهٔ ایثارگری ثبت شد: ' + (rec.id || ''));
  window.VET.plan = null;
  liveReload();
}

/** خالی‌کردن فرم و نتیجهٔ محاسبه. */
function vetClearForm() {
  ['vetPercent', 'vetCaptivity', 'vetFront', 'vetRelation', 'vetCardNo', 'vetIssuer', 'vetIssueDate', 'vetBenefits']
    .forEach(id => { const el = document.getElementById(id); if (el) el.value = ''; });
  window.VET.plan = null;
  const box = document.getElementById('vetPlanBox');
  if (box) box.replaceChildren();
}

/* ------------------------------ نتیجهٔ محاسبه ------------------------------ */

/** جدول امتیازهای محاسبه‌شدهٔ سرور. */
function vetPlanHTML(payload) {
  const r = payload.result || {};
  const profile = payload.profile || {};
  const rows = (r.lines || []).map(l => `<tr>
    <td>${esc(l.titleFa)}</td>
    <td class="num"><b class="num">${esc(l.valueFa)}</b></td>
    <td style="font-size:.76rem">${esc(l.refFa || '—')}</td></tr>`);
  const approvedChip = r.approved
    ? '<span class="chip c-ok">پروندهٔ تأییدشده — اثر مالی دارد</span>'
    : '<span class="chip c-wait">تأییدنشده — تا تأیید پرونده اثر مالی ندارد</span>';
  const payrollChip = payload.payrollApply === false
    ? '<span class="chip c-rej">اعمال در فیش با پارامتر شرکت خاموش است</span>' : '';
  const group = esc(r.groupFa || '');
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧮 نتیجهٔ محاسبهٔ امتیاز — ${esc(payload.employeeName || '')} ${group ? '(' + group + ')' : ''}</div>
    <div style="display:flex;gap:12px;flex-wrap:wrap;align-items:center;margin-bottom:8px">
      ${approvedChip}${payrollChip}
      <span class="muted" style="font-size:.82rem">معافیت مالیاتی: <b class="num">${fa(r.taxExemptPercent || 0)}٪</b></span>
      <span class="muted" style="font-size:.82rem">افزایش سنواتی: <b class="num">${fa(r.totalExtraServiceMonths || 0)}</b> ماه</span>
      <span class="muted" style="font-size:.82rem">پایهٔ تشویقی: <b class="num">${fa(r.incentiveSteps || 0)}</b></span>
      <span class="muted" style="font-size:.82rem">مرخصی تشویقی: <b class="num">${fa(r.leaveDays || 0)}</b> روز</span>
      <span class="muted" style="font-size:.82rem">سهمیهٔ استخدامی: <b class="num">${fa(r.employmentQuotaPercent || 0)}٪</b></span>
    </div>
    ${rows.length ? tbl(['امتیاز', 'مقدار', 'مبنا و ارجاع'], rows, {foot: false})
      : '<div class="muted">این گروه امتیاز مالی ندارد.</div>'}
    <div class="muted" style="font-size:.76rem;margin-top:6px">${esc(r.summaryFa || '')}
      ${profile.disabilityPercent ? ' — درصد جانبازی: ' + fa(profile.disabilityPercent) : ''}
      ${profile.captivityMonths ? ' — مدت اسارت: ' + fa(profile.captivityMonths) + ' ماه' : ''}
      ${profile.frontMonths ? ' — حضور در جبهه: ' + fa(profile.frontMonths) + ' ماه' : ''}</div>
    ${r.noteFa ? `<div class="muted" style="font-size:.76rem">${esc(r.noteFa)}</div>` : ''}
  </div>`;
}

/* ------------------------------ گزارش و پارامترها ------------------------------ */

/** وضعیت ایثارگری یک کارمند (پرس‌وجوی مستقل از فرم). */
function vetProfileBoxHTML(profile) {
  if (!profile || !profile.hasFile) {
    return `<div class="card card-pad" style="margin-bottom:12px;font-size:.8rem">
      برای کارمند انتخاب‌شده پروندهٔ ایثارگری ثبت نشده است؛ با فرم بالا محاسبه و ثبت کنید.
    </div>`;
  }
  const r = profile.result || {};
  return `<div class="card card-pad" style="margin-bottom:12px;font-size:.82rem;line-height:2">
    پروندهٔ <b>${esc(profile.employeeName || profile.employeeId)}</b> (${esc(r.groupFa || '')}) در وضعیت
    «${esc(profile.statusFa || '')}» است؛ شناسهٔ پرونده: <span class="num ltr">${esc(profile.recordId || '')}</span>.
    ${r.approved ? 'امتیازهای آن در فیش حقوق اعمال می‌شود.' : 'تا تأیید پرونده، امتیازها فقط نمایشی است.'}
  </div>`;
}

/** گزارش جامعهٔ ایثارگران: هر ردیف یک پرونده با امتیاز تأییدشدهٔ آن. */
function vetReportHTML(report) {
  const rows = (report.rows || []).map(r => `<tr>
    <td><span class="num ltr">${esc(r.recordId || '')}</span></td>
    <td>${personCell(r.employeeId, r.employeeNameFa || r.employeeId)}</td>
    <td>${esc(r.groupFa || '')}</td>
    <td>${statusChip('', r.statusFa || '')}</td>
    <td><span class="num">${esc(r.taxExemptFa || '0')}٪</span></td>
    <td><span class="num">${esc(r.extraServiceMonthsFa || '0')}</span></td>
    <td>${esc(r.educationUpgradeFa || '—')}</td>
    <td><span class="num">${esc(r.incentiveStepsFa || '0')}</span></td></tr>`);
  const body = rows.length
    ? tbl(['شناسه', 'کارمند', 'گروه ایثارگری', 'وضعیت', 'معافیت مالیاتی', 'افزایش سنواتی (ماه)', 'یک مقطع بالاتر', 'پایهٔ تشویقی'], rows, {foot: false})
    : '<div class="muted" style="font-size:.8rem">هنوز پروندهٔ ایثارگری ثبت نشده است.</div>';
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📊 گزارش جامعهٔ ایثارگران</div>
    <div class="muted" style="font-size:.78rem;margin-bottom:6px">
      جمع پرونده‌ها: <b class="num">${fa(report.total || 0)}</b> —
      تأییدشده: <b class="num">${fa(report.approved || 0)}</b> —
      مشمول معافیت مالیاتی: <b class="num">${fa(report.taxExemptCount || 0)}</b> —
      سهمیهٔ استخدامی مصوب شرکت: <b class="num">${esc(report.quotaFa || '')}</b>
    </div>
    ${body}
    <div class="muted" style="font-size:.76rem;margin-top:6px">${esc(report.noteFa || '')}</div>
  </div>`;
}

/** پارامترهای سیاستی امور ایثارگران از پروفایل شرکت (خواندنی؛ ویرایش در صفحهٔ پارامترها). */
function vetParamsHTML(def) {
  const views = def.paramViews || [];
  if (!views.length) {
    return `<div class="card card-pad" style="margin-bottom:12px;font-size:.8rem">
      پارامتر سیاستی برای امور ایثارگران در پروفایل این شرکت ثبت نشده است؛ محاسبه با پیش‌فرض
      قانونی انجام می‌شود.
    </div>`;
  }
  const params = def.params || {};
  const rows = views.map(p => `<tr>
    <td>${esc(p.titleFa)}</td>
    <td><span class="ltr" style="font-size:.72rem">${esc(p.key)}</span></td>
    <td><b class="num">${esc(fa(p.value))}</b></td>
    <td>${esc(p.unitFa || '—')}</td></tr>`);
  const chips = [
    ['اعمال در فیش حقوق', params.payrollApply ? 'بله' : 'خیر'],
    ['یک مقطع تحصیلی بالاتر', params.educationUpgrade ? 'بله' : 'خیر'],
    ['سقف پایهٔ تشویقی', fa(params.incentiveStepsMax || 0) + ' پایه'],
    ['سقف سنوات ارفاقی', fa(params.erafaqiMaxMonths || 0) + ' ماه'],
  ].map(([label, value]) =>
    `<span class="chip c-wait">${esc(label)}: <b>${esc(value)}</b></span>`).join(' ');
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">⚙️ پارامترهای سیاستی امور ایثارگران (از تنظیمات شرکت)</div>
    <div style="display:flex;gap:8px;flex-wrap:wrap;margin-bottom:8px">${chips}</div>
    ${tbl(['پارامتر', 'کلید', 'مقدار', 'واحد'], rows, {foot: false})}
    <div class="muted" style="font-size:.76rem;margin-top:6px">
      مبناهای قانونی (مادهٔ ۵۶ قانون جامع، قانون حمایت از آزادگان و قانون بازنشستگی جانبازان)
      در موتور سرور ثابتاند؛ این پارامترها فقط سیاست شرکت را می‌گویند و از صفحهٔ «تنظیمات
      پارامترها» ویرایش می‌شوند.
    </div>
  </div>`;
}

/** متن دستورالعمل امتیازها (از سرور). */
function vetInstructionHTML(def) {
  const steps = def.instruction || [];
  if (!steps.length) return '';
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📘 دستورالعمل اجرایی امتیازهای ایثارگری</div>
    <ol style="margin:6px 18px 0 0;padding:0;font-size:.82rem;line-height:2">
      ${steps.map(s => `<li>${esc(s)}</li>`).join('')}
    </ol>
  </div>`;
}


/* ------------------------- فرم مصوبهٔ کمیتهٔ ایثارگری ------------------------- */

/** پیش‌فرض قانونی همین کارمند برای مقایسه (از سرور). */
function vetCommitteeSuggestion() {
  const data = window.VET.committeeDefaults || {};
  return data.suggestion || null;
}

/** متن «پیش‌فرض قانونی» یک بند. */
function vetLegalText(fn, fallback) {
  const sug = vetCommitteeSuggestion();
  if (!sug) return 'پروندهٔ تأییدشده یافت نشد';
  return fn(sug) || fallback || '—';
}

/** خواندن شمارهٔ یک کادر فرم (ارقام فارسی هم پذیرفته است). */
function vetNum(id) {
  const el = document.getElementById(id);
  if (!el || !el.value) return 0;
  const digits = String(el.value).replace(/[۰-۹]/g, d => '۰۱۲۳۴۵۶۷۸۹'.indexOf(d)).replace(/[^0-9.\-]/g, '');
  const n = Number(digits);
  return isNaN(n) ? 0 : n;
}

/** خواندن متن یک کادر فرم. */
function vetVal(id) {
  const el = document.getElementById(id);
  return el ? String(el.value || '').trim() : '';
}

/** بازخوانی پیش‌فرض قانونی با تغییر کارمند. */
async function vetCommitteeEmpChanged() {
  window.VET.committeeEmployee = vetVal('vcEmployee');
  const res = await apiFetch(vetPrefix() + '/veteran-committee-defaults?employeeId=' +
    encodeURIComponent(window.VET.committeeEmployee));
  window.VET.committeeDefaults = res.ok ? res.data : null;
  const box = document.getElementById('vetCommitteeForm');
  if (box) box.innerHTML = vetCommitteeFormHTML({
    def: vetDef(),
    committeeDefaults: window.VET.committeeDefaults,
    committeeEmployeeSelect: document.getElementById('vcEmployee')
      ? document.getElementById('vcEmployee').outerHTML
      : '<select id="vcEmployee"></select>',
  });
  const emp = document.getElementById('vcEmployee');
  if (emp) emp.value = window.VET.committeeEmployee;
}

/** فرم مصوبهٔ کمیته: هر بند با تصمیم، مقدار و پیش‌فرض قانونی. */
function vetCommitteeFormHTML(view) {
  const def = (view && view.def) || vetDef();
  // گروه پیش‌فرض همان گروه پروندهٔ تأییدشدهٔ همین کارمند است تا کارشناس آن را دستی عوض نکند.
  const sugNow = vetCommitteeSuggestion() || {};
  const groups = (def.groups || []).map(g =>
    `<option value="${esc(g.key)}"${sugNow.veteranGroup === g.key ? ' selected' : ''}>${esc(g.titleFa)}</option>`).join('');
  const yesNo = (def.yesNo || []).map(o => `<option value="${esc(o.key)}">${esc(o.titleFa)}</option>`).join('');
  const bases = (def.seniorityBases || []).map(b => `<option value="${esc(b.key)}">${esc(b.titleFa)}</option>`).join('');
  // با تغییر کارمند، «پیش‌فرض قانونی» همان کارمند از سرور خوانده و فرم بازسازی می‌شود.
  const empSelect = ((view && view.committeeEmployeeSelect) || '<select id="vcEmployee"></select>')
    .replace('<select id="vcEmployee"', '<select id="vcEmployee" onchange="vetCommitteeEmpChanged()"');
  const sug = vetCommitteeSuggestion() || {};
  const legalGroup = sug.veteranGroupFa ? `گروه پرونده: ${esc(sug.veteranGroupFa)} — وضعیت: ${sug.approved ? 'تأییدشده' : 'تأییدنشده'}` : 'پرونده‌ای برای مقایسه یافت نشد';
  const pct = v => fa(v || 0) + '٪';
  const row = (label, legalFa, decisionID, numberID, numberValue, unitFa) => `
    <tr>
      <td><b>${esc(label)}</b><div class="muted" style="font-size:.7rem">پیش‌فرض قانونی: ${esc(legalFa)}</div></td>
      <td><select id="${decisionID}">${yesNo}</select></td>
      <td><input id="${numberID}" class="num" value="${numberValue}"> <span class="muted" style="font-size:.72rem">${esc(unitFa)}</span></td>
    </tr>`;
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧾 فرم مصوبهٔ کمیتهٔ ایثارگری — تصمیم دربارهٔ تعلق گرفتن هر مزایا و میزان آن</div>
    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px;margin-bottom:8px">
      <div class="field"><label>کارمند ✻</label>${empSelect}</div>
      <div class="field"><label>گروه ایثارگری ✻</label><select id="vcGroup">${groups}</select></div>
      <div class="field"><label>شمارهٔ صورت‌جلسه یا مصوبه</label><input id="vcCommitteeNo" placeholder="مثلاً ۱۴۰۵/ک‌ا/۰۳۲"></div>
      <div class="field"><label>تاریخ جلسهٔ کمیته ✻</label><input id="vcMeetingDate" value="${esc((view && view.def && view.def.today) || def.today || '')}" placeholder="1405/08/12"></div>
      <div class="field"><label>اعضای کمیته</label><input id="vcMembers" value="${esc((view && view.committeeDefaults && view.committeeDefaults.membersText) || '')}" placeholder="مدیر منابع انسانی، کارشناس امور ایثارگران، نمایندهٔ بنیاد شهید"></div>
      <div class="field" style="grid-column:1/-1"><label>مستندات بررسی‌شده</label>
        <input id="vcBasisDocs" placeholder="کارت ایثارگری، حکم جانبازی، سند اسارت، سابقهٔ تحصیلی"></div>
    </div>
    <div class="muted" style="font-size:.76rem;margin-bottom:6px">${esc(legalGroup)} — ستون «پیش‌فرض قانونی» از موتور قواعد همین کارمند ساخته می‌شود؛ هر بندی که متفاوت مصوب شود در پرونده و صورت‌جلسه ثبت می‌گردد.</div>
    ${tbl(['بند تصمیم', 'تصمیم کمیته', 'میزان مصوب'], [
      row('یک مقطع تحصیلی بالاتر', vetLegalText(s => s.educationUpgrade ? 'تعلق می‌گیرد' : 'تعلق نمی‌گیرد'),
        'vcEduUpgrade', 'vcEduLevel', sug.educationUpgrade ? 'مقطع معادل' : '', 'مقطع معادل‌شده (متن)'),
      row('معافیت مالیاتی حقوق', vetLegalText(s => fa(s.taxExemptPercent || 0) + '٪ از حقوق و فوق‌العاده‌ها'),
        'vcTaxExempt', 'vcTaxExemptPercent', sug.taxExemptPercent || 0, 'درصد'),
      row('معافیت کسور بازنشستگی', vetLegalText(s => pct(s.pensionExemptPercent)),
        'vcPensionExempt', 'vcPensionExemptPercent', sug.pensionExemptPercent || 0, 'درصد از کسور سهم کارمند'),
      row('کاهش ساعت کار جانباز', 'سیاست شرکت (بدون مصوبه اعمال نمی‌شود)',
        'vcWorkHourReduction', 'vcReducedWeeklyHours', 0, 'ساعت کار هفتگی مصوب'),
      row('پایهٔ تشویقی', vetLegalText(s => fa(s.incentiveSteps || 0) + ' پایه'),
        'vcIncentiveStepsYes', 'vcIncentiveSteps', sug.incentiveSteps || 0, 'پایه (۰ تا ۲)'),
      row('افزایش سنواتی', vetLegalText(s => fa(s.totalExtraServiceMonths || 0) + ' ماه'),
        'vcSeniorityYes', 'vcSeniorityMonths', sug.totalExtraServiceMonths || 0, 'ماه'),
      row('مرخصی تشویقی', vetLegalText(s => fa(s.leaveDays || 0) + ' روز'),
        'vcLeaveYes', 'vcLeaveDays', sug.leaveDays || 0, 'روز'),
      row('سهمیهٔ استخدامی', vetLegalText(s => fa(s.employmentQuotaPercent || 0) + '٪'),
        'vcEmploymentQuota', 'vcEmploymentQuotaHint', sug.employmentQuotaPercent || 0, 'درصد (گزارشی)'),
    ], {foot: false})}
    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px;margin-top:8px">
      <div class="field"><label>مبنای افزایش سنواتی</label><select id="vcSeniorityBasis">${bases}</select></div>
      <div class="field"><label>تاریخ اجرای مصوبه</label><input id="vcEffectiveDate" placeholder="1405/09/01"></div>
      <div class="field"><label>امضاکنندگان و تاریخ امضا</label><input id="vcSignOff" placeholder="امضاشده در جلسهٔ …"></div>
      <div class="field" style="grid-column:1/-1"><label>سایر امتیازات و توضیح</label><input id="vcOtherBenefits"></div>
      <div class="field" style="grid-column:1/-1"><label>خلاصهٔ متن تصمیم کمیته ✻</label>
        <input id="vcDecisionFa" placeholder="با بررسی مستندات، امتیازها به شرح بندها برقرار است…"></div>
    </div>
    ${vetSignatureBlockHTML(view)}
    <div class="row-actions" style="margin-top:8px">
      <button class="btn" onclick="vetCommitteeSave(false)">💾 ثبت مصوبه (بدون اعمال)</button>
      <button class="btn btn-primary" onclick="vetCommitteeSave(true)">✅ ثبت و اعمال بر پروندهٔ ایثارگری</button>
      <button class="btn btn-ghost" onclick="liveReload()">↻ تازه‌سازی</button>
    </div>
    <div class="muted" style="font-size:.74rem;margin-top:6px">
      اعمال مصوبه، امتیازهای مصوب را روی پروندهٔ کارمند می‌نشاند و پرونده را تأیید می‌کند تا فیش حقوق همان عدد را بخواند؛
      بدون اعمال، مصوبه فقط ثبت و چاپ می‌شود.
    </div>
  </div>`;
}

/** بدنهٔ درخواست ثبت مصوبه از فرم. */
function vetCommitteeBody() {
  const yes = id => (document.getElementById(id) && document.getElementById(id).value === 'yes') ? 'yes' : 'no';
  return {
    employeeId: vetVal('vcEmployee') || window.VET.committeeEmployee,
    committeeNo: vetVal('vcCommitteeNo'),
    meetingDate: vetVal('vcMeetingDate'),
    members: vetVal('vcMembers'),
    veteranGroup: vetVal('vcGroup'),
    basisDocs: vetVal('vcBasisDocs'),
    eduUpgradeDecision: yes('vcEduUpgrade'),
    eduLevel: vetVal('vcEduLevel'),
    taxExemptDecision: yes('vcTaxExempt'),
    taxExemptPercent: vetNum('vcTaxExemptPercent'),
    pensionExemptDecision: yes('vcPensionExempt'),
    pensionExemptPercent: vetNum('vcPensionExemptPercent'),
    workHourReduction: yes('vcWorkHourReduction'),
    reducedWeeklyHours: vetNum('vcReducedWeeklyHours'),
    incentiveSteps: vetNum('vcIncentiveSteps'),
    seniorityMonths: vetNum('vcSeniorityMonths'),
    seniorityBasis: vetVal('vcSeniorityBasis'),
    leaveDays: vetNum('vcLeaveDays'),
    employmentQuotaDecision: yes('vcEmploymentQuota'),
    otherBenefits: vetVal('vcOtherBenefits'),
    decisionFa: vetVal('vcDecisionFa'),
    effectiveDate: vetVal('vcEffectiveDate'),
    signOff: vetVal('vcSignOff'),
    signedMemberIds: vetSignatureIDs(),
    signDate: vetVal('vcSignDate'),
  };
}

/** ثبت مصوبه؛ با apply=true روی پروندهٔ کارمند اعمال می‌شود. */
async function vetCommitteeSave(apply) {
  const body = vetCommitteeBody();
  body.applyToFile = !!apply;
  if (!body.employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  if (!body.decisionFa) { toast('خلاصهٔ تصمیم کمیته را بنویسید', 'error'); return; }
  const res = await apiFetch(vetPrefix() + '/veteran-committee', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const data = res.data || {};
  const rec = data.record || {};
  if (data.applied && data.apply) {
    const ap = data.apply;
    toast('مصوبه ثبت و بر پرونده اعمال شد: ' + (ap.fileId || '') + ' — ' + (ap.statusFa || ''));
  } else if (data.applyMessage) {
    toast('مصوبه ثبت شد، ولی اعمال نشد: ' + data.applyMessage, 'error');
  } else {
    toast('مصوبه ثبت شد: ' + (rec.id || ''));
  }
  window.VET.committeeResult = data;
  liveReload();
}

/** اعمال یک مصوبهٔ ثبت‌شده بر پرونده. */
async function vetCommitteeApply(id) {
  const res = await apiFetch(vetPrefix() + '/veteran-committee/' + encodeURIComponent(id) + '/apply', {method: 'POST'});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('مصوبه بر پرونده اعمال شد: ' + ((res.data || {}).fileId || ''));
  liveReload();
}

/** چاپ صورت‌جلسهٔ مصوبه (برگهٔ امضاشدنی). */
function vetCommitteePrint(id) {
  liveOpenDoc('veteran_committee', id);
}

/** جدول مصوبه‌های ثبت‌شده. */
function vetCommitteeTableHTML(view) {
  const items = (view && view.committee) || [];
  if (!items.length) {
    return `<div class="card card-pad" style="margin-bottom:12px;font-size:.8rem">
      هنوز مصوبه‌ای برای کمیتهٔ ایثارگری ثبت نشده است؛ با فرم بالا اولین مصوبه را ثبت کنید.
    </div>`;
  }
  const rows = items.map(rec => {
    const f = rec.fields || {};
    const decisions = [
      f.taxExemptDecision === 'yes' ? 'معافیت مالیاتی ' + fa(f.taxExemptPercent || 0) + '٪' : 'بدون معافیت مالیاتی',
      f.pensionExemptDecision === 'yes' ? 'معافیت کسور ' + fa(f.pensionExemptPercent || 0) + '٪' : 'بدون معافیت کسور',
      f.eduUpgradeDecision === 'yes' ? 'یک مقطع بالاتر' : 'بدون معادل‌سازی مدرک',
      f.incentiveSteps ? fa(f.incentiveSteps) + ' پایهٔ تشویقی' : 'بدون پایهٔ تشویقی',
      f.seniorityMonths ? fa(f.seniorityMonths) + ' ماه افزایش سنواتی' : '',
      f.leaveDays ? fa(f.leaveDays) + ' روز مرخصی' : '',
      f.workHourReduction === 'yes' ? 'کاهش ساعت کار ' + fa(f.reducedWeeklyHours || 36) + ' ساعت' : '',
    ].filter(Boolean).join(' · ');
    const buttons = `<button class="btn btn-sm btn-ghost" title="چاپ صورت‌جلسه" onclick="vetCommitteePrint('${esc(rec.id)}')">🖨️</button>
      <button class="btn btn-sm btn-ghost" title="اعمال بر پرونده" onclick="vetCommitteeApply('${esc(rec.id)}')">⚙️</button>
      <button class="btn btn-sm btn-ghost" title="ویرایش" onclick="rkEdit('veteransAffairs#committee','${esc(rec.id)}')">✏️</button>
      <button class="btn btn-sm btn-ghost" title="حذف" onclick="rkDelete('${esc(vetPrefix())}','${esc(rec.id)}')">🗑️</button>`;
    return `<tr>
      <td><span class="num ltr">${esc(rec.id)}</span></td>
      <td>${personCell(rec.employeeId, rec.employeeName || rec.employeeId)}</td>
      <td>${esc(f.committeeNo || '—')}<div class="muted" style="font-size:.7rem">${esc(f.meetingDate || '')}</div></td>
      <td>${esc(rulesGroupFa(f.veteranGroup))}</td>
      <td style="font-size:.74rem">${esc(decisions)}</td>
      <td>${statusChip(rec.status, rec.statusFa)}</td>
      <td><div class="row-actions">${buttons}</div></td></tr>`;
  });
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📑 مصوبه‌های کمیتهٔ ایثارگری</div>
    <div class="muted" style="font-size:.78rem;margin-bottom:6px">هر مصوبه یک برگهٔ چاپی امضاشدنی دارد و با «اعمال بر پرونده» امتیازهای آن روی پروندهٔ کارمند می‌نشیند.</div>
    ${tbl(['شناسه', 'کارمند', 'صورت‌جلسه', 'گروه', 'بندهای مصوب', 'وضعیت', 'عملیات'], rows, {foot: false})}
  </div>`;
}

/** عنوان فارسی گروه ایثارگری از فهرست سرور. */
/* ------------------ اعضای کمیتهٔ ایثارگری و بلوک امضا ------------------ */

/** دادهٔ اعضای کمیته: از نمای صفحه، وگرنه از آخرین خواندهٔ سرور. */
function vetMemberData(view) {
  return (view && view.committeeMembers) || window.VET_MEMBER_DATA || {};
}

/** فهرست اعضای تعریف‌شدهٔ کمیته (فعال و غیرفعال). */
function vetMemberItems(view) {
  return vetMemberData(view).items || [];
}

/** اعضای فعال: همان‌هایی که پای صورت‌جلسه امضا می‌کنند. */
function vetActiveMembers(view) {
  return vetMemberItems(view).filter(m => m.active || m.status === 'active');
}

/** نقش‌های مصوب کمیته از سرور. */
function vetMemberRoles(view) {
  return vetMemberData(view).roles || (vetDef().roles || []);
}

/** پنل تعریف اعضای کمیتهٔ ایثارگری: فرم ثبت/ویرایش + فهرست اعضا. */
function vetMembersHTML(view) {
  const data = vetMemberData(view);
  const items = vetMemberItems(view);
  const active = vetActiveMembers(view);
  const roles = vetMemberRoles(view).map(r =>
    `<option value="${esc(r.key)}">${esc(r.titleFa)}</option>`).join('');
  const rows = items.map(m => {
    const toggle = m.active
      ? `<button class="btn btn-sm btn-ghost" title="غیرفعال‌کردن عضویت" onclick="vetMemberToggle('${esc(m.id)}')">⏸️</button>`
      : `<button class="btn btn-sm btn-ghost" title="فعال‌کردن عضویت" onclick="vetMemberToggle('${esc(m.id)}')">▶️</button>`;
    return `<tr>
      <td><span class="num ltr">${esc(m.id)}</span></td>
      <td>${esc(m.fullName)}${m.positionTitle ? `<div class="muted" style="font-size:.7rem">${esc(m.positionTitle)}</div>` : ''}</td>
      <td>${esc(m.roleFa)}</td>
      <td>${esc(m.orderNo || '—')}${m.orderDate ? `<div class="muted" style="font-size:.7rem">${esc(m.orderDate)}</div>` : ''}</td>
      <td class="num">${fa(m.sortOrder || 0)}</td>
      <td>${statusChip(m.status, m.statusFa)}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-ghost" title="ویرایش در فرم" onclick="vetMemberEdit('${esc(m.id)}')">✏️</button>
        ${toggle}
        <button class="btn btn-sm btn-ghost" title="حذف عضو" onclick="vetMemberDelete('${esc(m.id)}')">🗑️</button>
      </div></td></tr>`;
  });
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">👥 اعضای کمیتهٔ ایثارگری — تعریف اعضا و امضاکنندگان</div>
    <div class="muted" style="font-size:.78rem;margin-bottom:8px">${esc(data.noteFa || 'اعضای فعال پای صورت‌جلسهٔ مصوبه امضا می‌کنند و در بلوک امضای پایین فرم و برگهٔ چاپی می‌آیند.')}</div>
    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px;margin-bottom:8px">
      <div class="field"><label>نام و نام خانوادگی ✻</label><input id="vmName" placeholder="مثلاً سارا محمدی"></div>
      <div class="field"><label>نقش در کمیته ✻</label><select id="vmRole">${roles}</select></div>
      <div class="field"><label>سمت سازمانی</label><input id="vmPosition" placeholder="مدیر منابع انسانی"></div>
      <div class="field"><label>کد پرسنلی (اختیاری)</label><input id="vmEmployeeId" placeholder="EMP-001"></div>
      <div class="field"><label>شمارهٔ ابلاغ عضویت</label><input id="vmOrderNo" placeholder="۱۴۰۵/ح/۱۰۲"></div>
      <div class="field"><label>تاریخ ابلاغ</label><input id="vmOrderDate" placeholder="1405/01/05"></div>
      <div class="field"><label>ترتیب امضا</label><input id="vmSortOrder" class="num" placeholder="۱"></div>
      <div class="field"><label>عنوان امضا در برگه (اختیاری)</label><input id="vmSignTitle" placeholder="خالی یعنی عنوان نقش"></div>
      <div class="field"><label><input type="checkbox" id="vmActive" checked> عضو فعال</label></div>
    </div>
    <div class="row-actions" style="margin-bottom:10px">
      <button class="btn btn-primary" onclick="vetMemberSave()">💾 ثبت عضو کمیته</button>
      <button class="btn btn-ghost" onclick="vetMemberReset()">↺ پاک‌کردن فرم</button>
      <span class="muted" style="font-size:.74rem">مجموع: ${fa(items.length)} عضو — فعال: ${fa(active.length)}</span>
    </div>
    ${tbl(['شناسه', 'نام عضو', 'نقش', 'ابلاغ', 'ترتیب', 'وضعیت', 'عملیات'], rows, {foot: false})}
  </div>`;
}

/** پاک‌کردن فرم تعریف عضو. */
function vetMemberReset() {
  window.VET.memberEditing = '';
  ['vmName', 'vmPosition', 'vmEmployeeId', 'vmOrderNo', 'vmOrderDate', 'vmSortOrder', 'vmSignTitle']
    .forEach(id => { const el = document.getElementById(id); if (el) el.value = ''; });
  const act = document.getElementById('vmActive');
  if (act) act.checked = true;
}

/** بردن یک عضو به فرم برای ویرایش. */
function vetMemberEdit(id) {
  const m = vetMemberItems().find(x => x.id === id);
  if (!m) return;
  window.VET.memberEditing = id;
  const set = (elId, val) => { const el = document.getElementById(elId); if (el) el.value = val || ''; };
  set('vmName', m.fullName);
  set('vmRole', m.role);
  set('vmPosition', m.positionTitle);
  set('vmEmployeeId', m.employeeId);
  set('vmOrderNo', m.orderNo);
  set('vmOrderDate', m.orderDate);
  set('vmSortOrder', m.sortOrder || '');
  set('vmSignTitle', m.signatureTitle);
  const act = document.getElementById('vmActive');
  if (act) act.checked = !!m.active;
  toast('عضو ' + m.fullName + ' به فرم آمد؛ پس از تغییر «ثبت عضو کمیته» را بزنید');
}

/** ثبت یا ویرایش عضو کمیته. */
async function vetMemberSave() {
  const act = document.getElementById('vmActive');
  const body = {
    id: window.VET.memberEditing || '',
    fullName: vetVal('vmName'),
    role: vetVal('vmRole'),
    positionTitle: vetVal('vmPosition'),
    employeeId: vetVal('vmEmployeeId'),
    orderNo: vetVal('vmOrderNo'),
    orderDate: vetVal('vmOrderDate'),
    signatureTitle: vetVal('vmSignTitle'),
    sortOrder: vetNum('vmSortOrder'),
    active: act ? !!act.checked : true,
  };
  if (!body.fullName) { toast('نام و نام خانوادگی عضو را بنویسید', 'error'); return; }
  const res = await apiFetch(vetPrefix() + '/veteran-committee-members', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const rec = (res.data || {}).record || {};
  toast('عضو کمیته ثبت شد: ' + (rec.id || ''));
  window.VET.memberEditing = '';
  liveReload();
}

/** فعال/غیرفعال‌کردن عضویت. */
async function vetMemberToggle(id) {
  const res = await apiFetch(vetPrefix() + '/veteran-committee-members/' +
    encodeURIComponent(id) + '/toggle', {method: 'POST'});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('وضعیت عضویت تغییر کرد');
  liveReload();
}

/** حذف عضو کمیته. */
async function vetMemberDelete(id) {
  if (!confirm('این عضو از فهرست اعضای کمیته حذف شود؟')) return;
  const res = await apiFetch(vetPrefix() + '/veteran-committee-members/' +
    encodeURIComponent(id), {method: 'DELETE'});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('عضو کمیته حذف شد');
  liveReload();
}

/** بلوک «امضاهای پایین فرم کمیته»: هر عضو فعال با کادر امضا و تاریخ امضا. */
function vetSignatureBlockHTML(view) {
  const items = vetActiveMembers(view);
  const today = (vetDef().today) || '';
  const rows = items.map(m => `<tr>
      <td>${esc(m.roleFa)}</td>
      <td>${esc(m.fullName)}${m.positionTitle ? `<div class="muted" style="font-size:.7rem">${esc(m.positionTitle)}</div>` : ''}</td>
      <td class="muted" style="font-size:.74rem">${esc(m.orderNo || '—')}</td>
      <td style="text-align:center"><input type="checkbox" id="vsig-${esc(m.id)}" checked></td>
    </tr>`);
  const note = items.length
    ? 'اعضا از «پنل اعضای کمیتهٔ ایثارگری» می‌آیند؛ هر عضو که در جلسه امضا کرده تیک بخورد. متن «امضاکنندگان» را سرور از همین تیک‌ها می‌سازد.'
    : 'هنوز عضوی برای کمیته تعریف نشده است؛ در پنل «اعضای کمیتهٔ ایثارگری» اعضا را تعریف کنید تا این بلوک پر شود.';
  return `<div style="margin-top:12px;border-top:1px solid #E2E8F0;padding-top:10px">
    <div class="card-title" style="font-size:.95rem">🔏 امضاهای پایین فرم کمیته</div>
    <div class="muted" style="font-size:.76rem;margin-bottom:6px">${esc(note)}</div>
    ${items.length ? tbl(['عنوان امضا', 'نام امضاکننده', 'شمارهٔ ابلاغ', 'امضا شد'], rows, {foot: false}) : ''}
    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px;margin-top:8px">
      <div class="field"><label>تاریخ امضا</label><input id="vcSignDate" value="${esc(today)}" placeholder="1405/08/20"></div>
      <div class="field"><label>یادداشت امضا</label><input id="vcSignNote" placeholder="مثلاً امضاشده در جلسهٔ کمیته"></div>
    </div>
  </div>`;
}

/** شناسهٔ اعضایی که در فرم امضا شده‌اند. */
function vetSignatureIDs() {
  return vetActiveMembers()
    .filter(m => document.getElementById('vsig-' + m.id))
    .filter(m => {
      const el = document.getElementById('vsig-' + m.id);
      return el && el.checked;
    })
    .map(m => m.id);
}

/** عنوان فارسی گروه ایثارگری از فهرست سرور. */
function rulesGroupFa(key) {
  const hit = (vetDef().groups || []).find(g => g.key === key);
  return hit ? hit.titleFa : (key || '—');
}
