/* =====================================================================
   تعریف و تخصیص فوق العادههای حقوق — دو گام پشت سر هم
   ---------------------------------------------------------------------
   هیچ قاعدهای اینجا نیست: جدول رسمی کدها، روشهای محاسبه، مبناها، دامنهها و کارکنان از
   سرور میآید (`/api/v1/payroll/allowance-defaults`) و سنجش اعتبار، محاسبه و تطبیق دامنه
   هم در سرور انجام میشود (`/api/v1/payroll/allowance-preview`). این صفحه فقط:

      ۱) گام یکم — فوق العاده را تعریف میکند: کد (رسمی از جدول سازمان یا کد داخلی)،
         عنوان و نوع قلم، روش محاسبه (ثابت، یکبار مرتبه، درصدی، ضریبی، فرمولی)،
         مقدار/مبنا/فرمول، متغیرهای تعریف، مشمولیت مالیات و بیمه، جای نمایش در فیش و
         گزارش و کد گزارش (کدال).
      ۲) گام دوم — همان تعریف را به یک کارمند یا گروهی از کارکنان تخصیص میدهد:
         دامنه (کارمند/واحد/نوع استخدام/گروه شغلی/گروه نظام پرداخت/همه)، بازنویسی مقدار و
         مبنا برای همین تخصیص، و پنجرهٔ اعتبار.
      ۳) تعریفها و تخصیصها را فهرست و فعال/پایان میکند (رکوردهای ماژول حقوق).
      ۴) پیشنمایش میدهد: برای یک کارمند، سطرهای تخصیصیافته با کد، مبلغ و مشمولیت کسور.

   چرا دو گام: تعریف یک سند مرجع است و چند تخصیص به آن وصل میشود؛ تغییر نرخ یا فرمول
   فقط در یک جا انجام میگیرد و همهٔ تخصیصهای وابسته همان را میبینند. تخصیصهای قدیمی
   که هنوز شناسهٔ تعریف ندارند، با کد و عنوان خودشان نشان داده میشوند.
   ===================================================================== */

var ASG_DEFAULTS = null;
var ASG_DEFS = [];
var ASG_ROWS = [];

/** شمار تخصیصهای هر تعریف از دل فهرست تخصیصها (وقتی سرور عدد را ندهد). */
function asgAssignCount(definitionId) {
  const id = String(definitionId || '').trim();
  if (!id) return 0;
  return ASG_ROWS.filter(r => String((r.fields || {}).definition || '').trim() === id).length;
}

/**
 * فهرست تعریفها.
 *
 * نخست از `definitions` پاسخ پیشفرضها خوانده میشود؛ اگر سرور آن را نداده باشد، از
 * رکوردهای `allowanceDefinition` ساخته میشود تا این گام با هر دو حالت کار کند.
 */
function asgDefRows() {
  const fromDefaults = ((ASG_DEFAULTS || {}).definitions || []).filter(d => d && (d.id || d.code));
  if (fromDefaults.length) return fromDefaults;
  return ASG_DEFS.filter(r => r && r.id).map(r => {
    const f = r.fields || {};
    return {
      id: r.id, titleFa: f.titleFa || r.title || '', code: f.code || '', codeTitleFa: '',
      category: f.category || '', calc: f.calc || '', rate: f.rate || '', basis: f.basis || '',
      formula: f.formula || '', vars: f.vars || '', taxable: f.taxable || '', insured: f.insured || '',
      showOnPayslip: f.showOnPayslip || '', showInReports: f.showInReports || '',
      reportCode: f.reportCode || '', status: r.status || '', statusFa: r.statusFa || '',
      noteFa: f.noteFa || '', assignmentCount: asgAssignCount(r.id),
    };
  });
}

/** عنوان فارسی روش محاسبه/مبنا/دامنه از فهرستهای سرور. */
function asgMeta(list, key) {
  const item = ((ASG_DEFAULTS || {})[list] || []).find(x => x.key === key);
  return item ? item.titleFa : '';
}

/** یک تعریف با شناسه. */
function asgDefById(id) {
  const want = String(id || '').trim();
  return want ? (asgDefRows().find(d => String(d.id) === want) || null) : null;
}

/** گزینههای یک فهرست مرجع سرور؛ برچسب هر گزینه از خود سرور میآید. */
function asgOptions(list, selected) {
  return (list || []).map(x => {
    const value = String(x.key == null ? (x.code || x.id) : x.key);
    const title = x.titleFa || x.code || value;
    return `<option value="${esc(value)}" ${value === String(selected == null ? '' : selected) ? 'selected' : ''}>${esc(title)}</option>`;
  }).join('');
}

/** گزینههای جدول رسمی کد فوق العاده. */
function asgCodeOptions() {
  return ((ASG_DEFAULTS || {}).codes || []).map(c =>
    `<option value="${esc(c.code)}" ${String(c.code) === '151' ? 'selected' : ''}>${esc(c.code)} — ${esc(c.titleFa)}</option>`).join('');
}

/** گزینههای تعریف فعال در گام دوم: شناسه — کد و عنوان (روش محاسبه). */
function asgDefinitionOptions() {
  const defs = asgDefRows().filter(d => (d.status || 'active') === 'active' && d.id);
  if (!defs.length) return '<option value="">— نخست در گام ۱ یک فوق العاده تعریف کنید —</option>';
  return defs.map(d => {
    const calc = asgMeta('calcKinds', d.calc) || d.calc || '';
    const code = String(d.code || '').trim().replace(/^\d+$/, s => fa(s));
    const tail = [code ? esc(code) + ' — ' + esc(d.titleFa || '') : esc(d.titleFa || ''), calc ? esc(calc) : '']
      .filter(Boolean).join(' — ');
    return `<option value="${esc(d.id)}">${esc(d.id)} — ${tail}</option>`;
  }).join('');
}

/** گزینههای بخش «دامنهٔ تخصیص»: کارکنان، واحدها، انواع استخدام، گروههای شغلی و نظام پرداخت. */
function asgTargetOptions() {
  const employees = (ASG_DEFAULTS || {}).employees || [];
  const uniq = (values) => [...new Set(values.filter(v => v != null && String(v).trim() !== ''))].map(String);
  const emp = employees.map(e =>
    `<option value="${esc(e.id)}">${esc((e.id || '') + ' — ' + (e.titleFa || ''))}</option>`).join('');
  const units = uniq(employees.map(e => e.unit)).map(u => `<option value="${esc(u)}">${esc(u)}</option>`).join('');
  const types = employees.filter(e => e.employmentType).map(e =>
    `<option value="${esc(e.employmentType)}">${esc((e.employmentTypeFa || e.employmentType) + ' (' + e.employmentType + ')')}</option>`).join('');
  const grades = employees.filter(e => e.jobGrade).map(e =>
    `<option value="${esc(e.jobGrade)}">${esc((e.jobGradeFa || e.jobGrade) + ' (' + e.jobGrade + ')')}</option>`).join('');
  const groups = employees.filter(e => e.payGroup).map(e =>
    `<option value="${esc(e.payGroup)}">${esc((e.payGroupFa || '') + ' (' + e.payGroup + ')')}</option>`).join('');
  return emp + units + types + grades + groups;
}

/** عنوان یک دامنه برای نمایش در فهرست. */
function asgTargetTitle(kind, value) {
  if (kind === 'all') return 'همهٔ کارکنان';
  const meta = asgMeta('targetKinds', kind) || kind || '';
  const val = String(value || '').trim().replace(/^\d+$/, s => fa(s));
  return val ? esc(meta) + ': ' + esc(val) : esc(meta);
}

/** یک قلم بله/خیر فهرست (خالی یعنی «بله»، مثل رفتار سرور). */
function asgYes(value) {
  const v = String(value == null ? '' : value).trim();
  return v === '' || v === 'بله' || v === 'yes' || v === 'true' || v === '1';
}

/** یک کادر فرم با برچسب و توضیح کوتاه. */
function asgField(label, inner, hint) {
  return `<div class="field"><label>${label}</label>${inner}${hint ? `<span class="muted" style="font-size:.7rem">${esc(hint)}</span>` : ''}</div>`;
}

/** کارت گام ۱ — تعریف فوق العاده. */
function asgDefFormHtml() {
  const d = ASG_DEFAULTS || {};
  const y = (d.today || '').slice(0, 4).replace(/^\d{4}$/, m => fa(m));
  const yesNo = (id, on) => `<select id="${id}">
      <option value="بله" ${on ? 'selected' : ''}>بله</option>
      <option value="خیر" ${on ? '' : 'selected'}>خیر</option>
    </select>`;
  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="font-weight:600;margin-bottom:8px"><div>گام ۱ — تعریف فوق العاده</div></div>
      <div class="form-grid">
        ${asgField('کد فوق العاده <span class="req">*</span>', `<select id="asgDefCode">${asgCodeOptions()}</select>`, 'جدول رسمی ۲۸ کد سازمان؛ عنوان قلم خودکار پر میشود.')}
        ${asgField('یا کد داخلی', `<input id="asgDefCodeCustom" class="ltr" placeholder="یا کد داخلی مثل X-100">`, 'برای اقلام خارج از جدول رسمی؛ حرف و رقم، ۲ تا ۲۰ نویسه.')}
        ${asgField('عنوان قلم', `<input id="asgDefTitle" placeholder="خالی = عنوان رسمی همان کد">`)}
        ${asgField('نوع قلم', `<select id="asgDefCategory">
            <option value="earning">مزایا (بستانکار)</option>
            <option value="deduction">کسور (بدهکار)</option>
          </select>`)}
        ${asgField('روش محاسبه <span class="req">*</span>', `<select id="asgDefCalc">${asgOptions(d.calcKinds, 'fixed')}</select>`)}
        ${asgField('مقدار / درصد / ضریب', `<input id="asgDefRate" class="num" placeholder="مثلاً 9000000 یا 15 یا 1.5">`)}
        ${asgField('مبنا (برای درصدی و ضریبی)', `<select id="asgDefBasis"><option value="">—</option>${asgOptions(d.bases, '')}</select>`)}
        ${asgField('فرمول (برای روش فرمولی)', `<input id="asgDefFormula" class="ltr" placeholder="مثلاً BASE*0.15+CHILDREN*1000000">`, 'متغیرهای پایه: BASE، ALLOWANCEBASE، ATTRACTIONBASE، GROSS؛ متغیرهای تعریف از کادر روبرو میآید.')}
        ${asgField('متغیرهای تعریف', `<input id="asgDefVars" class="ltr" placeholder="CHILDREN=2,RATE=0.15">`, 'متغیرهای فرمول این تعریف؛ با کاما جدا شوند.')}
        ${asgField('مشمول مالیات (درج در گزارش مالیات)', yesNo('asgDefTaxable', false))}
        ${asgField('مشمول بیمه/بازنشستگی (درج در گزارش بیمه)', yesNo('asgDefInsured', false))}
        ${asgField('نمایش در فیش حقوقی', yesNo('asgDefPayslip', true))}
        ${asgField('درج در گزارشهای چاپی', yesNo('asgDefReports', true))}
        ${asgField('کد گزارش (کدال)', `<input id="asgDefReportCode" class="ltr" placeholder="مثلاً 151-1">`, 'کد رسمی گزارش سازمان؛ خالی بماند، در گزارش رسمی با عنوان قلم درج میشود.')}
        ${asgField('یادداشت / شمارهٔ مصوبه', `<input id="asgDefNote">`, y ? `ماه جاری: ${y}` : '')}
      </div>
      <div style="display:flex;gap:8px;align-items:center;margin-top:8px;flex-wrap:wrap">
        <button class="btn btn-primary" id="asgDefSave">ثبت تعریف فوق العاده</button>
        <button class="btn btn-ghost" id="asgDefReset">پاککردن فرم</button>
        <span class="muted" style="font-size:.74rem">${esc(d.noteFa || '')}</span>
      </div>
    </div>`;
}

/** کارت فهرست تعریفهای ثبتشده. */
function asgDefListHtml() {
  const defs = asgDefRows();
  const gaps = ((ASG_DEFAULTS || {}).gaps || []).filter(Boolean);
  const gapsNote = gaps.length
    ? `<div class="card card-pad muted" style="margin-bottom:12px;font-size:.76rem">کدهای رسمی بیپیادهسازی در موتور فعلی: ${esc(gaps.join('، '))}</div>`
    : '';
  if (!defs.length) {
    return gapsNote + `<div class="card card-pad muted" style="margin-bottom:12px">هنوز فوق العادهای تعریف نشده است؛ گام ۱ را کامل کنید.</div>`;
  }
  // جای نمایش قلم در خروجیها: فیش حقوقی، گزارشهای چاپی و کد گزارش رسمی (کدال).
  const rows = defs.map(d => {
    const count = d.assignmentCount == null ? asgAssignCount(d.id) : d.assignmentCount;
    const code = String(d.code || '').trim().replace(/^\d+$/, s => fa(s));
    const codeTitle = d.codeTitleFa || '';
    const value = d.calc === 'formula'
      ? `<span class="ltr">${esc(d.formula || '')}</span>`
      : `${esc(d.rate || '')}${d.calc === 'percent' ? '٪' : ''}${d.basis ? ' × ' + esc(asgMeta('bases', d.basis) || d.basis) : ''}`;
    // متغیرها ممکن است متن («CHILDREN=2,RATE=0.15») یا نگاشت نام→مقدار بیایند.
    const varsRaw = typeof d.vars === 'object' && d.vars ? Object.keys(d.vars).map(k => k + '=' + d.vars[k]).join(',') : d.vars;
    const vars = String(varsRaw || d.varsText || d.varsFa || '').trim();
    const flags = `${asgYes(d.taxable) ? 'مالیات' : '—'} / ${asgYes(d.insured) ? 'بیمه' : '—'}`;
    const showFlags = `${asgYes(d.showOnPayslip) ? 'فیش' : 'بدون فیش'} / ${asgYes(d.showInReports) ? 'گزارش' : 'بدون گزارش'}${d.reportCode ? ' · کد ' + esc(d.reportCode) : ''}`;
    const actions = `<button class="btn btn-sm btn-ghost" data-asg-pick="${esc(d.id)}">تخصیص</button>` +
      ((d.status || 'active') === 'active'
        ? `<button class="btn btn-sm btn-ghost" data-asg-def-act="close" data-asg-id="${esc(d.id)}">پایان</button>`
        : `<button class="btn btn-sm btn-primary" data-asg-def-act="activate" data-asg-id="${esc(d.id)}">فعالسازی</button>`);
    return `<tr>
      <td class="num ltr">${esc(d.id || '')}</td>
      <td>${code ? esc(code) + ' — ' : ''}${esc(d.titleFa || '')}${codeTitle && d.titleFa !== codeTitle ? `<div class="muted" style="font-size:.68rem">${esc(codeTitle)}</div>` : ''}</td>
      <td>${esc(d.categoryFa || (d.category === 'deduction' ? 'کسور' : d.category === 'earning' ? 'مزایا' : d.category || ''))}</td>
      <td>${esc(asgMeta('calcKinds', d.calc) || d.calc || '')}</td>
      <td class="num">${value}</td>
      <td class="ltr" style="font-size:.7rem">${vars ? esc(vars) : '—'}</td>
      <td class="muted" style="font-size:.72rem">${flags}</td>
      <td class="muted" style="font-size:.72rem">${showFlags}</td>
      <td class="num">${fa(count == null ? 0 : count)}</td>
      <td>${pChip((d.status || 'active') === 'active' ? 'c-ok' : 'c-wait', d.statusFa || (d.status === 'closed' ? 'پایانیافته' : 'فعال'))}</td>
      <td><div class="row-actions">${actions}</div></td>
    </tr>`;
  });
  return gapsNote + `<div class="card card-pad" style="margin-bottom:12px">
      <div style="font-weight:600;margin-bottom:6px">فهرست تعریفهای فوق العاده (${fa(defs.length)})</div>
      ${tbl(['شماره', 'کد و عنوان', 'نوع', 'روش محاسبه', 'مقدار/مبنا', 'متغیرها', 'مشمول', 'نمایش/گزارش', 'تعداد تخصیص', 'وضعیت', 'عملیات'],
        rows, {count: defs.length})}
    </div>`;
}

/** کارت گام ۲ — تخصیص به گروه استخدامی یا افراد. */
function asgAssignFormHtml() {
  const d = ASG_DEFAULTS || {};
  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="font-weight:600;margin-bottom:8px"><div>گام ۲ — تخصیص به گروه استخدامی یا افراد</div></div>
      <div class="form-grid">
        ${asgField('تعریف فوق العاده <span class="req">*</span>', `<select id="asgDefinition">${asgDefinitionOptions()}</select>`, 'فقط تعریفهای فعال؛ شناسه و عنوان در فهرست بالا آمده است.')}
        ${asgField('دامنهٔ تخصیص <span class="req">*</span>', `<select id="asgTargetKind">${asgOptions(d.targetKinds, 'employee')}</select>`)}
        ${asgField('مقدار دامنه', `<input id="asgTargetValue" list="asgTargetList" placeholder="شناسهٔ کارمند، واحد، نوع استخدام یا گروه">
            <datalist id="asgTargetList">${asgTargetOptions()}</datalist>`, 'برای دامنهٔ «همهٔ کارکنان» خالی بماند.')}
        ${asgField('بازنویسی مقدار', `<input id="asgRateOverride" class="num" placeholder="خالی = مقدار تعریف">`, 'خالی = مقدار تعریف.')}
        ${asgField('بازنویسی مبنا', `<select id="asgBasisOverride"><option value="">خالی = مبنای تعریف</option>${asgOptions(d.bases, '')}</select>`)}
        ${asgField('ماه شروع اعتبار <span class="req">*</span>', `<input id="asgFrom" placeholder="${esc(d.period || '1405/01')}">`)}
        ${asgField('ماه پایان اعتبار', `<input id="asgTo" placeholder="خالی = تا اطلاع">`)}
        ${asgField('یادداشت / شمارهٔ مصوبه', `<input id="asgNote">`)}
      </div>
      <div style="display:flex;gap:8px;align-items:center;margin-top:8px;flex-wrap:wrap">
        <button class="btn btn-primary" id="asgSave">ثبت تخصیص</button>
        <button class="btn btn-ghost" id="asgReset">پاککردن فرم</button>
      </div>
    </div>`;
}

/** کارت فهرست تخصیصهای ثبتشده. */
function asgListHtml() {
  if (!ASG_ROWS.length) return `<div class="card card-pad muted" style="margin-bottom:12px">هنوز تخصیصی ثبت نشده است؛ گام ۲ را کامل کنید.</div>`;
  const rows = ASG_ROWS.map(r => {
    const f = r.fields || {};
    const defId = String(f.definition || '').trim();
    const def = asgDefById(defId);
    // تخصیصهای قدیمی شناسهٔ تعریف ندارند؛ با کد و عنوان خودشان نشان داده میشوند.
    const code = String((def && def.code) || f.code || '').trim().replace(/^\d+$/, s => fa(s));
    const title = (def && def.titleFa) || f.titleFa || r.title || '';
    const label = defId
      ? `<span class="ltr" style="font-size:.7rem">${esc(defId)}</span> — ${code ? esc(code) + ' — ' : ''}${esc(title)}`
      : `${code ? esc(code) + ' — ' : ''}${esc(title)}`;
    const calc = (def && def.calc) || f.calc || '';
    const rate = String(f.rateOverride || (def && def.rate) || f.rate || '');
    const basis = String(f.basisOverride || (def && def.basis) || f.basis || '');
    const over = `${rate ? esc(rate) + (calc === 'percent' ? '٪' : '') : '—'}${basis ? ' × ' + esc(asgMeta('bases', basis) || basis) : ''}`;
    const actions = r.status === 'active'
      ? `<button class="btn btn-sm btn-ghost" data-asg-act="close" data-asg-id="${esc(r.id)}">پایان</button>`
      : `<button class="btn btn-sm btn-primary" data-asg-act="activate" data-asg-id="${esc(r.id)}">فعالسازی</button>`;
    return `<tr>
      <td class="num ltr">${esc(r.id)}</td>
      <td>${label}</td>
      <td>${asgTargetTitle(f.targetKind, f.targetValue)}</td>
      <td class="num">${over}</td>
      <td>${esc(f.effectiveFrom || '')}${f.effectiveTo ? ' تا ' + esc(f.effectiveTo) : ''}</td>
      <td>${pChip(r.status === 'active' ? 'c-ok' : 'c-wait', r.statusFa || r.status)}</td>
      <td><div class="row-actions">${actions}</div></td>
    </tr>`;
  });
  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="font-weight:600;margin-bottom:6px">تخصیصهای ثبتشده (${fa(ASG_ROWS.length)})</div>
      ${tbl(['شماره', 'تعریف/کد و عنوان', 'دامنه', 'بازنویسی', 'اعتبار', 'وضعیت', 'عملیات'], rows, {count: ASG_ROWS.length})}
    </div>`;
}

/** کارت پیشنمایش تخصیصهای یک کارمند. */
function asgPreviewHtml() {
  const empOpts = ((ASG_DEFAULTS || {}).employees || []).map(e =>
    `<option value="${esc(e.id)}">${esc(e.titleFa || e.id)}${e.unit ? ' — ' + esc(e.unit) : ''}</option>`).join('');
  return `<div class="card card-pad">
      <div style="font-weight:600;margin-bottom:6px">پیشنمایش تخصیصها</div>
      <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
        <select id="asgPreviewEmployee" style="min-width:320px"><option value="">— انتخاب کارمند —</option>${empOpts}</select>
        <button class="btn btn-primary" id="asgPreviewRun">نمایش اقلام</button>
        <span class="muted" style="font-size:.72rem">فقط تخصیصهای فعال و در پنجرهٔ اعتبار همین ماه میآید.</span>
      </div>
      <div id="asgPreviewOut" style="margin-top:10px"></div>
    </div>`;
}

/** صفحهٔ «تعریف و تخصیص فوق العادههای حقوق». */
function pageAllowanceAssignLive() {
  if (!isLive()) return offlineFallback('allowanceAssign');
  (async () => {
    await liveLoad(async () => {
      const [defRes, asgRes, defListRes] = await Promise.all([
        apiFetch('/api/v1/payroll/allowance-defaults'),
        apiFetch('/api/v1/payroll/records?kind=allowanceAssignment&size=200'),
        apiFetch('/api/v1/payroll/records?kind=allowanceDefinition&size=200'),
      ]);
      if (!defRes.ok) return liveErrorBox(defRes.message);
      ASG_DEFAULTS = defRes.data || {};
      ASG_ROWS = (asgRes.ok && asgRes.data && asgRes.data.items) || [];
      ASG_DEFS = (defListRes.ok && defListRes.data && defListRes.data.items) || [];
      return asgDefFormHtml() + asgDefListHtml() + asgAssignFormHtml() + asgListHtml() + asgPreviewHtml();
    });
    asgBind();
  })();
  return liveShell('تعریف و تخصیص فوق العادههای حقوق',
    'گام یکم: تعریف فوق العاده (کد، روش محاسبه، متغیرها، مشمولیت کسور و جای نمایش) — گام دوم: تخصیص آن به گروه استخدامی یا افراد', '');
}

/** اتصال رویدادها (بدون هندلر درونخطی). */
function asgBind() {
  // کد رسمی ⟵ عنوان و نوع قلم خودکار از جدول سازمان (فقط وقتی عنوان خالی است).
  const codeSel = document.getElementById('asgDefCode');
  if (codeSel && !codeSel.dataset.asgBound) {
    codeSel.dataset.asgBound = '1';
    codeSel.addEventListener('change', () => {
      const item = ((ASG_DEFAULTS || {}).codes || []).find(c => String(c.code) === codeSel.value);
      if (!item) return;
      const title = document.getElementById('asgDefTitle');
      if (title && !title.value) title.value = item.titleFa;
      const category = document.getElementById('asgDefCategory');
      if (category) category.value = item.kind === 'earning' ? 'earning' : 'deduction';
    });
  }
  const defSave = document.getElementById('asgDefSave');
  if (defSave && !defSave.dataset.asgBound) {
    defSave.dataset.asgBound = '1';
    defSave.addEventListener('click', () => asgDefSave());
  }
  const defReset = document.getElementById('asgDefReset');
  if (defReset && !defReset.dataset.asgBound) {
    defReset.dataset.asgBound = '1';
    defReset.addEventListener('click', () => liveReload());
  }
  const save = document.getElementById('asgSave');
  if (save && !save.dataset.asgBound) {
    save.dataset.asgBound = '1';
    save.addEventListener('click', () => asgSave());
  }
  const reset = document.getElementById('asgReset');
  if (reset && !reset.dataset.asgBound) {
    reset.dataset.asgBound = '1';
    reset.addEventListener('click', () => liveReload());
  }
  const run = document.getElementById('asgPreviewRun');
  if (run && !run.dataset.asgBound) {
    run.dataset.asgBound = '1';
    run.addEventListener('click', () => asgPreview());
  }
  // عملیات سطرها (تخصیص/فعالسازی/پایان) با شنوندهٔ واگذارشده روی ناحیهٔ صفحه.
  const region = document.getElementById('liveRegion');
  if (region && !region.dataset.asgBound) {
    region.dataset.asgBound = '1';
    region.addEventListener('click', ev => {
      const pick = ev.target.closest('[data-asg-pick]');
      if (pick) { asgPickDefinition(pick.dataset.asgPick); return; }
      const defAct = ev.target.closest('[data-asg-def-act]');
      if (defAct) { asgDefAction(defAct.dataset.asgId, defAct.dataset.asgDefAct); return; }
      const act = ev.target.closest('[data-asg-act]');
      if (act) { asgAction(act.dataset.asgId, act.dataset.asgAct); }
    });
  }
}

/** نشاندن HTML در یک ظرف؛ تنها نقطهٔ ورود HTML این صفحه (سقف امنیتی رابط). */
function asgPaint(el, html) { if (el) el.innerHTML = html; }

/** مقدار یک ورودی فرم. */
function asgVal(id) {
  const el = document.getElementById(id);
  return el ? (el.value || '').trim() : '';
}

/**
 * تعریف انتخابشده در گام ۱: کد داخلی مقدم است، وگرنه کد رسمی جدول سازمان.
 *
 * عنوان هم اگر خالی بماند از عنوان رسمی همان کد پر میشود (سرور هم همین کار را میکند).
 */
function asgDefPicked() {
  const custom = asgVal('asgDefCodeCustom');
  const code = custom || asgVal('asgDefCode');
  const item = ((ASG_DEFAULTS || {}).codes || []).find(c => String(c.code) === code);
  return {code, item, title: asgVal('asgDefTitle') || (item ? item.titleFa : '')};
}

/** ثبت تعریف فوق العاده (گام ۱): نخست ایجاد پیشنویس، سپس فعالسازی. */
async function asgDefSave() {
  const picked = asgDefPicked();
  const fields = {
    code: picked.code,
    titleFa: picked.title,
    category: asgVal('asgDefCategory'),
    calc: asgVal('asgDefCalc'),
    rate: asgVal('asgDefRate'),
    basis: asgVal('asgDefBasis'),
    formula: asgVal('asgDefFormula'),
    vars: asgVal('asgDefVars'),
    taxable: asgVal('asgDefTaxable'),
    insured: asgVal('asgDefInsured'),
    showOnPayslip: asgVal('asgDefPayslip'),
    showInReports: asgVal('asgDefReports'),
    reportCode: asgVal('asgDefReportCode'),
    noteFa: asgVal('asgDefNote'),
  };
  const res = await apiFetch('/api/v1/payroll/records', {
    method: 'POST', body: {kind: 'allowanceDefinition', title: picked.title || 'تعریف فوق العاده', fields},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  // تعریف تازه پیشنویس است؛ تا فعال نشود در گام ۲ نمیآید.
  const created = res.data || {};
  if (created.id) {
    await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(created.id) + '/action', {
      method: 'POST', body: {action: 'activate'},
    });
  }
  toast('تعریف فوق العاده ثبت و فعال شد');
  liveReload();
}

/** ثبت تخصیص تازه (گام ۲): نخست ایجاد پیشنویس، سپس فعالسازی. */
async function asgSave() {
  const fields = {
    definition: asgVal('asgDefinition'),
    targetKind: asgVal('asgTargetKind'), targetValue: asgVal('asgTargetValue'),
    rateOverride: asgVal('asgRateOverride'), basisOverride: asgVal('asgBasisOverride'),
    effectiveFrom: asgVal('asgFrom'), effectiveTo: asgVal('asgTo'), noteFa: asgVal('asgNote'),
  };
  const def = asgDefById(fields.definition);
  const title = def ? (def.titleFa || def.code || 'تخصیص فوق العاده') : 'تخصیص فوق العاده';
  const res = await apiFetch('/api/v1/payroll/records', {
    method: 'POST', body: {kind: 'allowanceAssignment', title, fields},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  // تخصیص تازه پیشنویس است؛ برای اثرگذاری باید فعال شود.
  const created = res.data || {};
  if (created.id) {
    await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(created.id) + '/action', {
      method: 'POST', body: {action: 'activate'},
    });
  }
  toast('تخصیص ثبت و فعال شد');
  liveReload();
}

/** فعالسازی یا پایان یک تخصیص. */
async function asgAction(id, action) {
  if (!id || !action) return;
  const res = await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(id) + '/action', {
    method: 'POST', body: {action},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast(action === 'activate' ? 'تخصیص فعال شد' : 'تخصیص پایان یافت');
  liveReload();
}

/** فعالسازی یا پایان یک تعریف فوق العاده. */
async function asgDefAction(id, action) {
  if (!id || !action) return;
  const res = await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(id) + '/action', {
    method: 'POST', body: {action},
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast(action === 'activate' ? 'تعریف فوق العاده فعال شد' : 'تعریف فوق العاده پایان یافت');
  liveReload();
}

/** «تخصیص» یک سطر گام ۱: همان تعریف در فرم گام ۲ انتخاب و به آن اسکرول میشود. */
function asgPickDefinition(id) {
  const sel = document.getElementById('asgDefinition');
  if (!sel) return;
  sel.value = String(id || '');
  sel.focus();
  if (sel.scrollIntoView) sel.scrollIntoView({block: 'center', behavior: 'smooth'});
}

/** پیشنمایش اقلام یک کارمند. */
async function asgPreview() {
  const out = document.getElementById('asgPreviewOut');
  const employeeId = asgVal('asgPreviewEmployee');
  if (!out) return;
  if (!employeeId) { out.textContent = 'کارمند را انتخاب کنید.'; return; }
  asgPaint(out, liveSpinner());
  const res = await apiFetch('/api/v1/payroll/allowance-preview', {method: 'POST', body: {employeeId}});
  if (!res.ok) { asgPaint(out, liveErrorBox(res.message)); return; }
  const d = res.data || {};
  const lines = d.lines || [];
  const totals = d.totals || {};
  const cards = `<div class="grid-stats" style="margin-bottom:8px">
      ${statCard({ico: '➕', tint: '#DCFCE7', accent: '#16A34A', val: money(totals.earnings || 0), label: 'جمع مزایا'})}
      ${statCard({ico: '➖', tint: '#FEE2E2', accent: '#DC2626', val: money(totals.deductions || 0), label: 'جمع کسور'})}
      ${statCard({ico: '🧾', tint: '#FEF3C7', accent: '#D97706', val: money(totals.taxableEarnings || 0), label: 'مشمول مالیات'})}
      ${statCard({ico: '🛡️', tint: '#EDE9FE', accent: '#7C3AED', val: money(totals.insuredEarnings || 0), label: 'مشمول بیمه'})}
      ${statCard({ico: '🖨️', tint: '#DBEAFE', accent: '#2563EB', val: money(totals.payslipEarnings || 0), label: 'نمایش در فیش'})}
    </div>`;
  if (!lines.length) {
    asgPaint(out, cards + `<div class="card card-pad muted">برای این کارمند در این ماه تخصیص فعالی نیست.</div>`);
    return;
  }
  // هشدار هم‌پوشانی: اگر یک قلم از دو تخصیص فعال بیاید، مبلغ دو بار محاسبه می‌شود.
  const warn = (typeof liveWarnBox === 'function') ? liveWarnBox(d.warningsFa || []) : '';
  const rows = lines.map(l => `<tr>
      <td class="num ltr">${esc(l.code)}</td>
      <td>${esc(l.titleFa)}</td>
      <td>${esc(l.categoryFa || l.category)}</td>
      <td>${esc(l.calcFa || l.calc)}</td>
      <td class="num">${money(l.amount)}</td>
      <td>${l.taxable ? '✅' : '—'}</td>
      <td>${l.insured ? '✅' : '—'}</td>
      <td>${l.showOnPayslip ? '✅' : '—'}</td>
      <td>${l.showInReports ? '✅' : '—'}</td>
      <td class="num ltr">${esc(l.reportCode || '')}</td>
      <td class="muted" style="font-size:.72rem">${esc(l.detailFa || '')}</td>
    </tr>`);
  asgPaint(out, cards + warn + tbl(['کد', 'عنوان', 'نوع', 'روش محاسبه', 'مبلغ (ریال)', 'مالیات', 'بیمه', 'فیش', 'گزارش', 'کد گزارش', 'شرح'],
    rows, {count: lines.length}));
}

if (typeof LIVE_PAGES !== 'undefined') LIVE_PAGES.allowanceAssign = pageAllowanceAssignLive;
