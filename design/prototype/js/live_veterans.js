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
  if (box) box.innerHTML = '';
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
