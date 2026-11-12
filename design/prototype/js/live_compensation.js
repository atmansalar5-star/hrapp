/* =====================================================================
   نظام جبران خدمات ۴پی — صفحه‌های زندهٔ وب
   =====================================================================

   این فایل هیچ قاعدهٔ کسب‌وکاری ندارد: پارامترها، باندها، جدول ضریب‌ها، محاسبهٔ
   چهارعاملی و راهنمای تکمیل داده همه از سرور می‌آید (`/api/v1/compensation/...`).
   رابط فقط ورودی‌ها را می‌فرستد و نتیجهٔ موتور قواعد را نشان می‌دهد.

   چهار صفحهٔ رکوردی (مدل، شایستگی، عملکرد، باند پرداخت) از موتور عمومی
   `pageRecordkitLive` استفاده می‌کنند؛ سه صفحهٔ باقی‌مانده پنل اختصاصی دارند:
   قیمت‌گذاری شغل (P1)، تصمیم چهارعاملی و آمادگی داده.
   ===================================================================== */

Object.assign(RK_PAGES, {
  compModel: {kind: 'compensationModel'},
  compCompetency: {kind: 'competencyLevel'},
  compPerformance: {kind: 'performanceFactor'},
  compPayBands: {kind: 'payBand'},
});

/* صفحه‌های رکوردی این ماژول؛ همان موتور عمومی، بدون قاعدهٔ تازه. */
for (const compRecordPage of ['compModel', 'compCompetency', 'compPerformance', 'compPayBands']) {
  LIVE_PAGES[compRecordPage] = () => pageRecordkitLive(compRecordPage);
}

/* سه صفحهٔ اختصاصی. */
LIVE_PAGES.compJobPricing = () => pageCompJobPricingLive();
LIVE_PAGES.compDecision = () => pageCompDecisionLive();
LIVE_PAGES.compReadiness = () => pageCompReadinessLive();

/** پیشوند API ماژول از منوی تولیدشده؛ نبودنش یعنی ماژول در این استقرار فعال نیست. */
function compPrefix() {
  const mod = rkFindModule('compDecision');
  return mod ? mod.prefix : '/api/v1/compensation';
}

/** پارامترها، باندها، جدول ضریب‌ها و فهرست‌های مرجع (یک‌بار در هر بارخوانی). */
let COMP_OPTS = null;
async function compOptions(force) {
  if (COMP_OPTS && !force) return COMP_OPTS;
  const res = await apiFetch(compPrefix() + '/options');
  COMP_OPTS = res.ok ? (res.data || {}) : {};
  return COMP_OPTS;
}

/** گزینه‌های یک فهرست مرجع سرور به شکل <option>. */
function compOptionsHTML(list, selected) {
  return (list || []).map(o =>
    `<option value="${esc(o.key)}" ${String(o.key) === String(selected) ? 'selected' : ''}>${esc(o.titleFa)}</option>`).join('');
}

/** کارت نتیجهٔ محاسبه: پرداخت هدف، افزایش، نسبت به میانه و ساختار باند. */
function compResultHTML(result) {
  const r = result || {};
  const line = l => `<tr><td>${esc(l.titleFa)}</td><td>${esc(l.valueFa)}</td>
    <td class="muted" style="font-size:.74rem">${esc(l.refFa || '—')}</td></tr>`;
  const warnings = (r.warningsFa || []).map(w =>
    `<div class="card card-pad" style="border-color:#fde68a;color:#92400e;margin-bottom:8px;font-size:.82rem">⚠️ ${esc(w)}</div>`).join('');
  return `<div class="card card-pad" style="margin-bottom:12px">
      <div class="card-title">🧮 نتیجهٔ محاسبهٔ چهارعاملی</div>
      <div class="grid-stats" style="margin-bottom:10px">
        ${statCard({ico: '🏷️', tint: '#EFF6FF', accent: '#2563EB', val: fa(r.rank || 0) + ' — ' + esc(r.gradeFa || '—'), label: 'پلهٔ گروه شغلی (P1)'})}
        ${statCard({ico: '🎯', tint: '#EDE9FE', accent: '#7C3AED', val: '×' + fa(r.personFactor || 0), label: 'ضریب شایستگی (P2)'})}
        ${statCard({ico: '📈', tint: '#FEF3C7', accent: '#D97706', val: '×' + fa(r.performanceFactor || 0), label: 'ضریب عملکرد (P3)'})}
        ${statCard({ico: '💰', tint: '#DCFCE7', accent: '#16A34A', val: money(r.targetPay || 0), label: 'پرداخت هدف (P4)'})}
      </div>
      ${warnings}
      ${tbl(['سطر محاسبه', 'مقدار', 'مبنا'], (r.lines || []).map(line), {foot: false})}
      <div class="muted" style="font-size:.82rem;line-height:2;margin-top:8px">${esc(r.summaryFa || '')}</div>
    </div>`;
}

/* =====================================================================
   P1 — قیمت‌گذاری شغل: امتیاز ارزشیابی → پله و باند
   ===================================================================== */

function pageCompJobPricingLive() {
  if (!isLive()) return offlineFallback('compJobPricing');
  liveLoad(async () => {
    const opts = await compOptions();
    const mod = rkFindModule('compJobPricing') || {};
    const list = await apiFetch((mod.prefix || compPrefix()) + '/records?kind=jobPricing&size=200&scope=all');
    const items = (list.ok && list.data ? list.data.items : []) || [];

    const rows = items.map(rec => {
      const f = rec.fields || {};
      return `<tr>
        <td class="num ltr">${esc(rec.id)}</td>
        <td>${esc(f.positionTitle || '—')}<div class="muted ltr" style="font-size:.66rem">${esc(f.positionCode || '')}</div></td>
        <td>${rkSelectTitle('jobFamily', f.jobFamily) || '—'}</td>
        <td class="num">${fa(f.evaluationPoints || 0)}</td>
        <td class="num">${fa(f.gradeRank || 0)}</td>
        <td class="num">${f.bandMidpoint ? money(f.bandMidpoint) : '—'}</td>
        <td>${statusChip(rec.status, rec.statusFa)}</td>
        <td>${faDate(f.pricingDate)}</td>
      </tr>`;
    });

    const bands = (opts.bands || []).map(b => `<tr>
        <td class="num">${fa(b.rank)}</td><td>${esc(b.gradeFa)}</td>
        <td class="num">${money(b.min)}</td><td class="num">${money(b.midpoint)}</td>
        <td class="num">${money(b.max)}</td><td class="num">${money(b.step)}</td></tr>`);

    return `<div class="card card-pad" style="margin-bottom:12px">
        <div class="card-title">🏷️ قیمت‌گذاری شغل (P1 — شغل)</div>
        <div class="muted" style="font-size:.78rem;margin-bottom:8px">
          امتیاز ارزشیابی مصوب شغل را وارد کنید؛ سامانه پلهٔ گروه شغلی و باند پرداخت را از پارامترهای مصوب شرکت می‌سازد.
          میانهٔ باند پلهٔ ۱ در این استقرار: <b>${money((opts.params || {}).bandBaseMidpoint || 0)}</b> ریال.
        </div>
        <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px">
          <div class="field"><label>کد پست ✻</label><input id="cjpCode" placeholder="POS-0142"></div>
          <div class="field"><label>عنوان پست ✻</label><input id="cjpTitle" placeholder="سرپرست بهره‌برداری"></div>
          <div class="field"><label>خانوادهٔ شغلی ✻</label><select id="cjpFamily">${compOptionsHTML(opts.jobFamilies)}</select></div>
          <div class="field"><label>تاریخ قیمت‌گذاری ✻</label><input id="cjpDate" value="${esc(opts.today || '')}" placeholder="1405/01/12"></div>
          <div class="field"><label>عامل دانش و مهارت</label><input id="cjpKnowledge" type="number" min="0" max="100" value="0"></div>
          <div class="field"><label>عامل حل مسئله</label><input id="cjpProblem" type="number" min="0" max="100" value="0"></div>
          <div class="field"><label>عامل مسئولیت</label><input id="cjpResponsibility" type="number" min="0" max="100" value="0"></div>
          <div class="field"><label>عامل شرایط کار</label><input id="cjpConditions" type="number" min="0" max="100" value="0"></div>
          <div class="field"><label>شمارهٔ مصوبه</label><input id="cjpApproval" placeholder="۱۴۰۵/ط‌م/۰۱۸"></div>
        </div>
        <div class="row-actions" style="margin-top:10px">
          <button class="btn btn-primary" onclick="compJobPricingCalc()">🧮 محاسبهٔ پله و باند</button>
          <button class="btn" onclick="compJobPricingSave()">💾 ثبت قیمت‌گذاری</button>
        </div>
        <div id="cjpResult" style="margin-top:10px"></div>
      </div>
      <div class="card card-pad" style="margin-bottom:12px">
        <div class="card-title">📐 باندهای پرداخت مصوب (P4)</div>
        ${tbl(['پله', 'گروه شغلی', 'کف باند', 'میانهٔ باند', 'سقف باند', 'گام پله'], bands, {foot: false})}
        <div class="muted" style="font-size:.74rem;margin-top:6px">
          این باندها از پارامترهای مصوب ساخته می‌شوند؛ برای تغییرشان مدل جبران خدمات یا پارامترهای شرکت را به‌روز کنید.
        </div>
      </div>
      ${tbl(['شناسه', 'پست', 'خانوادهٔ شغلی', 'امتیاز', 'پله', 'میانهٔ باند', 'وضعیت', 'تاریخ قیمت‌گذاری'], rows, {count: items.length})}`;
  });
  return liveShell('قیمت‌گذاری شغل (P1 — شغل)', 'نگاشت امتیاز ارزشیابی شغل به پلهٔ گروه شغلی و باند پرداخت');
}

/** خواندن عامل‌های شغل از فرم و تبدیل به عدد. */
function compJobFactors() {
  const val = id => Number((document.getElementById(id) || {}).value || 0);
  return {
    knowledge: val('cjpKnowledge'), problemSolving: val('cjpProblem'),
    responsibility: val('cjpResponsibility'), workingConditions: val('cjpConditions'),
  };
}

/** محاسبهٔ امتیاز شغل، پله و باند (همه در سرور). */
async function compJobPricingCalc() {
  const res = await apiFetch(compPrefix() + '/job-pricing-calc', {method: 'POST', body: {factors: compJobFactors()}});
  const box = document.getElementById('cjpResult');
  if (!res.ok) { toast(res.message, 'error'); return; }
  const d = res.data || {};
  window.COMP_JOB = d;
  if (box) box.innerHTML = `<div class="card card-pad" style="border-color:#bfdbfe">
      جمع عامل‌های شغل: <b>${fa(d.evaluationPoints || 0)}</b> امتیاز — پلهٔ گروه شغلی: <b>${fa(d.gradeRank || 0)} («${esc((d.band || {}).gradeFa || '')}»)</b>
      — میانهٔ باند: <b>${money((d.band || {}).midpoint || 0)}</b> ریال
      <div class="muted" style="font-size:.76rem;margin-top:4px">${esc(d.noteFa || '')}</div>
    </div>`;
}

/** ثبت رکورد قیمت‌گذاری شغل با اعداد محاسبه‌شدهٔ سرور. */
async function compJobPricingSave() {
  const d = window.COMP_JOB;
  if (!d) { toast('نخست «محاسبهٔ پله و باند» را بزنید', 'error'); return; }
  const val = id => ((document.getElementById(id) || {}).value || '').trim();
  const factors = compJobFactors();
  const body = {
    kind: 'jobPricing',
    title: 'قیمت‌گذاری شغل — ' + val('cjpTitle'),
    fields: {
      positionCode: val('cjpCode'), positionTitle: val('cjpTitle'), jobFamily: val('cjpFamily'),
      evaluationPoints: String(d.evaluationPoints || 0),
      factorKnowledge: String(factors.knowledge), factorProblemSolving: String(factors.problemSolving),
      factorResponsibility: String(factors.responsibility), factorWorkingConditions: String(factors.workingConditions),
      gradeRank: String(d.gradeRank || 0), bandMidpoint: String((d.band || {}).midpoint || 0),
      pricingDate: val('cjpDate'), approvalNo: val('cjpApproval'),
    },
    noteFa: 'ثبت از پنل قیمت‌گذاری شغل (P1)',
  };
  if (!body.fields.positionTitle) { toast('عنوان پست را بنویسید', 'error'); return; }
  const res = await apiFetch(compPrefix() + '/records', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('قیمت‌گذاری شغل ثبت شد: ' + ((res.data || {}).id || ''));
  window.COMP_JOB = null;
  liveReload();
}

/* =====================================================================
   تصمیم چهارعاملی: محاسبه و ثبت تصمیم
   ===================================================================== */

function pageCompDecisionLive() {
  if (!isLive()) return offlineFallback('compDecision');
  liveLoad(async () => {
    const opts = await compOptions();
    const selected = window.COMP && window.COMP.employeeId ? window.COMP.employeeId : '';
    const defaults = selected
      ? await apiFetch(compPrefix() + '/4p-defaults?employeeId=' + encodeURIComponent(selected))
      : {ok: false, data: {}};
    const emp = (defaults.ok && defaults.data ? defaults.data.employee : {}) || {};
    const employeeSelect = await liveEmployeeSelect('compEmployee', selected);

    const decisions = selected
      ? await apiFetch(compPrefix() + '/records?kind=compensationDecision&size=200&scope=all&employeeId=' + encodeURIComponent(selected))
      : {ok: true, data: {items: []}};
    const items = (decisions.ok && decisions.data ? decisions.data.items : []) || [];

    const rows = items.map(rec => {
      const f = rec.fields || {};
      return `<tr>
        <td class="num ltr">${esc(rec.id)}</td>
        <td>${esc(f.decisionNo || '—')}</td>
        <td class="num">${fa(f.gradeRank || 0)}</td>
        <td class="num">${fa(f.competencyLevel || 0)}</td>
        <td class="num">${fa(f.performanceScore || 0)}</td>
        <td class="num">${money(f.currentPay || 0)}</td>
        <td class="num">${money(f.targetPay || 0)}</td>
        <td class="num">${money(f.increaseAmount || 0)}</td>
        <td>${statusChip(rec.status, rec.statusFa)}</td>
      </tr>`;
    });

    const factorTable = (opts.factorTable || []).map(l => `<tr>
        <td>${esc(l.titleFa)}</td><td>${esc(l.valueFa)}</td>
        <td class="muted" style="font-size:.74rem">${esc(l.refFa || '')}</td></tr>`);

    return `<div class="card card-pad" style="margin-bottom:12px">
        <div class="card-title">🧮 محاسبهٔ چهارعاملی — شغل، شخص، عملکرد، پرداخت</div>
        <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px">
          <div class="field"><label>کارمند ✻</label>${employeeSelect}</div>
          <div class="field"><label>امتیاز ارزشیابی شغل (P1)</label><input id="cdJobScore" type="number" min="0" value="${esc(emp.jobScore || '')}"></div>
          <div class="field"><label>پلهٔ گروه شغلی</label><input id="cdGrade" type="number" min="0" value="${esc(emp.gradeRank || '')}"></div>
          <div class="field"><label>سطح شایستگی (P2)</label><select id="cdLevel">${compOptionsHTML(opts.competencyLevels, emp.competencyLevel)}</select></div>
          <div class="field"><label>امتیاز عملکرد (P3)</label><input id="cdPerformance" type="number" min="0" max="100" value="${esc(emp.performanceScore || '')}"></div>
          <div class="field"><label>پرداخت فعلی ماهانه (ریال)</label><input id="cdCurrent" type="number" min="0" value="${esc(emp.currentPay || '')}"></div>
          <div class="field"><label>تعدیل بازار</label><select id="cdMarket">${compOptionsHTML(opts.yesNo, 'no')}</select></div>
          <div class="field"><label>شمارهٔ تصمیم ✻</label><input id="cdNo" placeholder="۱۴۰۵/ج‌خ/۰۲۷"></div>
          <div class="field"><label>تاریخ تصمیم ✻</label><input id="cdDate" value="${esc(opts.today || '')}"></div>
          <div class="field"><label>تاریخ اجرا ✻</label><input id="cdEffective" value="${esc(opts.today || '')}"></div>
        </div>
        <div class="field" style="margin-top:8px"><label>خلاصهٔ تصمیم و مبنا ✻</label><input id="cdDecisionFa" placeholder="مبنای تصمیم و بندهای مصوب"></div>
        <div class="muted" style="font-size:.74rem;margin-top:6px">
          ${emp.currentPaySourceFa ? 'مبنای پرداخت فعلی: ' + esc(emp.currentPaySourceFa) : 'حکم ابلاغ‌شده‌ای برای این کارمند یافت نشد؛ پرداخت فعلی را دستی وارد کنید.'}
          ${emp.performanceScoreSourceFa ? ' — مبنای امتیاز عملکرد: ' + esc(emp.performanceScoreSourceFa) : ''}
        </div>
        <div class="row-actions" style="margin-top:10px">
          <button class="btn btn-primary" onclick="compDecisionCalc()">🧮 محاسبهٔ چهارعاملی</button>
          <button class="btn" onclick="compDecisionSave()">💾 ثبت تصمیم</button>
        </div>
        <div id="cdResult" style="margin-top:10px"></div>
      </div>
      <div class="card card-pad" style="margin-bottom:12px">
        <div class="card-title">📊 جدول ضریب‌های مصوب (شخص و عملکرد)</div>
        ${tbl(['ضریب', 'مقدار', 'مبنا'], factorTable, {foot: false})}
      </div>
      ${items.length ? tbl(['شناسه', 'شمارهٔ تصمیم', 'پله', 'سطح شایستگی', 'عملکرد', 'پرداخت فعلی', 'پرداخت هدف', 'افزایش', 'وضعیت'], rows, {count: items.length}) : ''}`;
  });
  return liveShell('تصمیم چهارعاملی جبران خدمات', 'محاسبهٔ بستهٔ پرداخت بر پایهٔ شغل، شایستگی و عملکرد و ثبت تصمیم برای صدور حکم');
}

/** خواندن مقدار یک ورودی فرم تصمیم. */
function compDecisionInput(id) {
  return ((document.getElementById(id) || {}).value || '').trim();
}

/** محاسبهٔ چهارعاملی کارمند انتخاب‌شده (محاسبه در سرور). */
async function compDecisionCalc() {
  const body = {
    employeeId: compDecisionInput('compEmployee'),
    jobScore: Number(compDecisionInput('cdJobScore') || 0),
    gradeOverride: Number(compDecisionInput('cdGrade') || 0),
    competencyLevel: Number(compDecisionInput('cdLevel') || 0),
    performanceScore: Number(compDecisionInput('cdPerformance') || 0),
    currentPay: Number(compDecisionInput('cdCurrent') || 0),
    marketAdjust: compDecisionInput('cdMarket') === 'yes',
  };
  const res = await apiFetch(compPrefix() + '/4p-calc', {method: 'POST', body});
  const box = document.getElementById('cdResult');
  if (!res.ok) { toast(res.message, 'error'); if (box) box.innerHTML = liveErrorBox(res.message); return; }
  window.COMP_RESULT = res.data || {};
  if (box) box.innerHTML = compResultHTML((res.data || {}).result);
}

/** ثبت تصمیم چهارعاملی با اعداد محاسبه‌شدهٔ سرور. */
async function compDecisionSave() {
  const payload = window.COMP_RESULT;
  if (!payload || !payload.result) { toast('نخست «محاسبهٔ چهارعاملی» را بزنید', 'error'); return; }
  const r = payload.result;
  const body = {
    kind: 'compensationDecision',
    title: 'تصمیم چهارعاملی جبران خدمات — ' + (r.employeeName || r.employeeId || ''),
    employeeId: r.employeeId,
    fields: {
      decisionNo: compDecisionInput('cdNo'), decisionDate: compDecisionInput('cdDate'),
      effectiveDate: compDecisionInput('cdEffective'), jobScore: String(r.jobScore || 0),
      gradeRank: String(r.rank || 0), competencyLevel: String(r.competencyLevel || 0),
      performanceScore: String(r.performanceScore || 0), currentPay: String(r.currentPay || 0),
      targetPay: String(r.targetPay || 0), increaseAmount: String(r.increase || 0),
      compaRatioPercent: String(r.compaRatioPercent || 0),
      marketAdjust: r.marketApplied ? 'yes' : 'no',
      decisionFa: compDecisionInput('cdDecisionFa') || r.summaryFa || '',
    },
    noteFa: 'ثبت از پنل تصمیم چهارعاملی (شامل خروجی موتور قواعد)',
  };
  const res = await apiFetch(compPrefix() + '/records', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('تصمیم چهارعاملی ثبت شد: ' + ((res.data || {}).id || ''));
  window.COMP_RESULT = null;
  liveReload();
}

/* =====================================================================
   آمادگی داده: ترتیب تکمیل جداول، دادهٔ حیاتی و درصد پیشرفت
   ===================================================================== */

function pageCompReadinessLive() {
  if (!isLive()) return offlineFallback('compReadiness');
  liveLoad(async () => {
    const res = await apiFetch(compPrefix() + '/data-readiness');
    if (!res.ok) return liveErrorBox(res.message);
    const d = res.data || {};
    const steps = d.steps || [];

    // وضعیت هر گام از سرور می‌آید؛ اینجا فقط رنگ همان وضعیت انتخاب می‌شود.
    const CLS = {ok: 'c-ok', partial: 'c-wait', empty: 'c-rej'};
    const FA = {ok: 'کامل', partial: 'ناقص — در جریان', empty: 'شروع‌نشده'};
    const chipOf = status => pChip(CLS[status] || 'c-wait', FA[status] || status || '—');

    const rows = steps.map(step => `<tr>
        <td class="num">${fa(step.order)}</td>
        <td>${esc(step.titleFa)}<div class="muted" style="font-size:.68rem">${esc(step.moduleFa || '')}</div></td>
        <td>${esc(step.tableFa)}</td>
        <td class="muted" style="font-size:.74rem">${(step.criticalFa || []).map(esc).join('، ') || '—'}</td>
        <td class="muted" style="font-size:.74rem">${esc(step.whyFa || '')}</td>
        <td>${chipOf(step.status)}</td>
        <td class="muted" style="font-size:.74rem">${esc(step.countFa || '')}</td>
      </tr>`);

    const instruction = (d.instruction || []).map(s =>
      `<div style="line-height:2">${esc(s)}</div>`).join('');

    const done = Number(d.doneCount || 0);
    const total = Number(d.totalCount || 0);
    const top = `<div class="grid-stats" style="margin-bottom:12px">
        ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: fa(done), label: 'گام کامل'})}
        ${statCard({ico: '📋', tint: '#EFF6FF', accent: '#2563EB', val: fa(total), label: 'کل گام‌های داده'})}
        ${statCard({ico: '📈', tint: '#FEF3C7', accent: '#D97706', val: fa(total ? Math.round(done / total * 100) : 0) + '٪', label: 'پیشرفت تکمیل داده'})}
      </div>`;

    return top + tbl(['اولویت', 'گام', 'جدول/فهرست داده', 'دادهٔ حیاتی', 'چرا این اولویت', 'وضعیت', 'شمارش'],
        rows, {foot: false}) +
      `<div class="card card-pad" style="margin-top:12px;font-size:.82rem;line-height:2">
        <div class="card-title">🧭 ترتیب پیشنهادی شروع</div>
        ${instruction}
      </div>
      <div class="card card-pad" style="margin-top:12px;font-size:.82rem;line-height:2">
        ${esc(d.criticalNoteFa || '')}<br>${esc(d.orderNoteFa || '')}
      </div>`;
  });
  return liveShell('آمادگی داده و اولویت تکمیل جداول',
    'راهنمای شروع به‌روزرسانی دادهٔ نظام ۴پی: کدام جدول نخست، چه داده‌ای حیاتی است و هر گام چند درصد پر است');
}
