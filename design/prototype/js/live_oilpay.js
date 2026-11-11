/* =====================================================================
   فیش حقوقی صندوق نفت (ماژول حقوق و دستمزد) — نظام پرداخت دوم سازمان
   ---------------------------------------------------------------------
   چرا این صفحه: سازمان دو نظام پرداخت دارد. نظام عمومی (تأمین اجتماعی) مزد مبنا و اقلام
   خودش را دارد؛ کارکنان تابع صندوق بازنشستگی نفت ساختار دیگری دارند:

       پایهٔ حقوق
       + فوق‌العادهٔ بدی آب و هوا (مناطق گرمسیری) — درصدی از پایهٔ حقوق
       = «مبنای فوق‌العاده‌ها»
       بقیهٔ فوق‌العاده‌ها (محرومیت، ویژه، صرف وقت، کارگاهی، جذب و تخصصی شغل،
       فرآورشی و نوبت‌کار) — درصدی از همین مبنا
       − کسور صندوق، بیمه و مالیات
       = خالص پرداختی

   محاسبه در سرور است (`POST /api/v1/payroll/oil-payslip`) و این فایل هیچ درصدی را
   خودش حساب نمی‌کند: درصدهای مصوب (`oilpay.*`)، فهرست اقلام و مبناها همه از سرور می‌آید.

     GET  /api/v1/payroll/oil-payslip-defaults   پارامترها، اقلام، کارکنان
     POST /api/v1/payroll/oil-payslip            محاسبهٔ فیش و تفکیک اقلام
     POST /api/v1/payroll/records                ثبت فیش به‌عنوان رکورد (نوع oilPayslips)
   ===================================================================== */

var OIL_DEFAULTS = null;   // پارامترها و فهرست‌های مرجع
var OIL_RESULT = null;     // آخرین فیش محاسبه‌شده
var OIL_EMPLOYEES = [];    // کارکنان
var OIL_TIMER = null;      // زمان‌سنج محاسبهٔ خودکار

/** مقدار عددی یک ورودی (خالی = صفر). */
function oilNum(id) {
  const el = document.getElementById(id);
  if (!el) return 0;
  return Number((el.value || '').replace(/[^0-9.\-]/g, '')) || 0;
}

/** متن یک ورودی. */
function oilText(id) {
  const el = document.getElementById(id);
  return el ? (el.value || '').trim() : '';
}

/** مقدار یک فهرست کشویی. */
function oilSelect(id) {
  const el = document.getElementById(id);
  return el ? el.value : '';
}

/** درصد دستی هر قلم فوق‌العاده از فرم.
 *
 *  قلم‌هایی که درصدشان از جدول‌های آیین‌نامه می‌آید (drivenBy: محل سکونت یا پایهٔ سمت)
 *  فرستاده نمی‌شوند؛ وگرنه مقدار پیش‌فرض فرم همیشه بر جدول مقدم می‌شد و جدول هرگز اعمال
 *  نمی‌گشت. آن درصدها را سرور از جدول ۱/۲/۳ حساب می‌کند. */
function oilAllowancePercents() {
  const out = {};
  for (const a of ((OIL_DEFAULTS && OIL_DEFAULTS.allowances) || [])) {
    if (a.fixed) continue;
    const el = document.getElementById('oilAllow_' + a.key);
    if (!el) continue;
    // قلم جدولی فقط وقتی فرستاده می‌شود که کاربر خودش عدد را عوض کرده باشد؛ وگرنه
    // جدول ۱/۲/۳ سرور همان درصد را می‌دهد و بازنویسی دستی روی جدول نمی‌نشیند.
    if (a.drivenBy && el.dataset.edited !== '1') continue;
    const value = Number((el.value || '').replace(/[^0-9.\-]/g, ''));
    if (!isNaN(value) && el.value !== '') out[a.key] = value;
  }
  return out;
}

/** مبلغ دستی اقلام ثابت (عایله‌مندی، مسکن، حق اولاد، فوق‌العادهٔ خاص) — صفر یعنی مصوب شرکت. */
function oilAllowanceAmounts() {
  const out = {};
  for (const a of ((OIL_DEFAULTS && OIL_DEFAULTS.allowances) || [])) {
    if (!a.fixed) continue;
    const el = document.getElementById('oilAmount_' + a.key);
    if (!el) continue;
    const value = Number((el.value || '').replace(/[^0-9.\-]/g, ''));
    if (!isNaN(value) && el.value !== '') out[a.key] = value;
  }
  return out;
}

/** سطرهای دستی فیش (بند ۱/۲/۳) از جدول ویرایش‌پذیر. */
function oilExtraLines() {
  const out = [];
  for (const row of document.querySelectorAll('#oilExtraRows tr[data-extra]')) {
    const title = (row.querySelector('[data-f="title"]').value || '').trim();
    const amount = Number((row.querySelector('[data-f="amount"]').value || '').replace(/[^0-9.\-]/g, '')) || 0;
    if (!title || amount === 0) continue;
    out.push({
      titleFa: title,
      amount: amount,
      side: row.querySelector('[data-f="side"]').value,
      band: Number(row.querySelector('[data-f="band"]').value) || 3,
    });
  }
  return out;
}

/** افزودن یک سطر دستی خالی به جدول سطرهای فیش. */
function oilAddExtraRow(band, side, title, amount) {
  const body = document.getElementById('oilExtraRows');
  if (!body) return;
  const tr = document.createElement('tr');
  tr.setAttribute('data-extra', '1');
  tr.innerHTML = `
    <td><select data-f="band" onchange="oilCalcSoon()">
      <option value="1"${Number(band) === 1 ? ' selected' : ''}>۱ — حقوق و فوق‌العاده‌ها و کسورات قانونی</option>
      <option value="2"${Number(band) === 2 ? ' selected' : ''}>۲ — کسورات تطبیق جهت رعایت سقف</option>
      <option value="3"${Number(band) === 3 || !band ? ' selected' : ''}>۳ — سایر پرداخت‌ها و کسورات</option>
    </select></td>
    <td><input data-f="title" value="${esc(title || '')}" placeholder="شرح قلم" oninput="oilCalcSoon()"></td>
    <td><input data-f="amount" class="num" value="${amount ? String(amount) : ''}" oninput="oilCalcSoon()"></td>
    <td><select data-f="side" onchange="oilCalcSoon()">
      <option value="credit"${side !== 'debit' ? ' selected' : ''}>بستانکار (پرداخت)</option>
      <option value="debit"${side === 'debit' ? ' selected' : ''}>بدهکار (کسور)</option>
    </select></td>
    <td><button class="btn btn-sm btn-ghost" onclick="this.closest('tr').remove(); oilCalc()">✕</button></td>`;
  body.appendChild(tr);
}

/** افزودن سطر دستی از خود فرم. */
function oilExtraAdd() {
  oilAddExtraRow(document.getElementById('oilExtraBand').value, 'debit', '', 0);
  oilCalc();
}

/** ویرایش دستی درصد یک قلم جدولی؛ خالی‌کردن کادر یعنی بازگشت به جدول آیین‌نامه. */
function oilOnAllowanceEdit(el) {
  el.dataset.edited = (el.value || '').trim() === '' ? '0' : '1';
  oilCalcSoon();
}

/** بازگشت همهٔ درصدهای جدولی به مقدار جدول ۱ تا ۳. */
function oilResetDriven() {
  for (const el of document.querySelectorAll('[data-driven="1"]')) el.dataset.edited = '0';
  oilCalc();
}

/** بازنویسی پارامترهای نظام صندوق برای همین محاسبه (کلید oilpay.* → مقدار). */
function oilParamOverrides() {
  const out = {};
  for (const el of document.querySelectorAll('[data-oilparam]')) {
    const key = el.dataset.oilparam;
    const value = (el.value || '').trim();
    if (key && value !== '') out[key] = value;
  }
  return out;
}

/** ساخت بدنهٔ درخواست محاسبه از فیلدهای فرم. */
function oilInput() {
  return {
    // کد کارمند برای خواندن پروندهٔ ایثارگری همان کارمند در سرور فرستاده می‌شود؛
    // امتیاز از درخواست گرفته نمی‌شود، فقط پروندهٔ تأییدشدهٔ همین کارمند اعمال می‌گردد.
    employeeId: oilSelect('oilEmployee'),
    baseSalary: oilNum('oilBaseSalary'),
    climatePercent: oilNum('oilClimatePercent'),
    allowancePercents: oilAllowancePercents(),
    allowanceAmounts: oilAllowanceAmounts(),
    extraLines: oilExtraLines(),
    degree: oilSelect('oilDegree'),
    jobGrade: oilSelect('oilJobGrade'),
    residence: oilSelect('oilResidence'),
    shift: oilSelect('oilShift') === 'yes',
    shiftDays: oilNum('oilShiftDays'),
    daysInMonth: oilNum('oilDaysInMonth'),
    overtimeHours: oilNum('oilOvertime'),
    otherDeductions: oilNum('oilDeductions'),
    // ضرائب و نرخ‌های همین محاسبه؛ پروفایل شرکت دست نمی‌خورد.
    paramOverrides: oilParamOverrides(),
  };
}

/** محاسبه با تأخیر کوتاه (برای تایپ). */
function oilCalcSoon() {
  if (OIL_TIMER) clearTimeout(OIL_TIMER);
  OIL_TIMER = setTimeout(oilCalc, 250);
}

/** فراخوانی محاسبهٔ فیش و بازترسیم. */
async function oilCalc() {
  const box = document.getElementById('oilSlip');
  if (box) box.innerHTML = '<div class="muted" style="font-size:.78rem">⏳ در حال محاسبه...</div>';
  const res = await apiFetch('/api/v1/payroll/oil-payslip', { method: 'POST', body: oilInput() });
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    return;
  }
  OIL_RESULT = res.data || {};
  oilRenderSlip();
}

/** نمایش فیش: کارت‌های جمع، تفکیک اقلام و مبنای محاسبه. */
function oilRenderSlip() {
  const box = document.getElementById('oilSlip');
  if (!box || !OIL_RESULT || !OIL_RESULT.slip) return;
  const s = OIL_RESULT.slip;

  // سه بند فیش رسمی: هر بند جدولی دو ستونی (بستانکار/بدهکار) با جمع خودش.
  const bandHtml = (s.bands || []).map(band => {
    const rows = (band.lines || []).map(l => {
      const debit = l.groupFa === 'کسور';
      return `<tr>
        <td>${esc(l.titleFa)}</td>
        <td style="font-size:.72rem">${esc(l.groupFa || '')}</td>
        <td class="num">${l.percent ? fa(Math.round(l.percent * 10) / 10) + '٪' : '—'}</td>
        <td class="num">${debit ? '' : fa(money(l.amount))}</td>
        <td class="num" style="color:#dc2626">${debit ? fa(money(l.amount)) : ''}</td>
        <td style="font-size:.74rem">${esc(l.detailFa || '—')}</td></tr>`;
    });
    rows.push(`<tr><td><b>جمع ${esc(band.titleFa)}</b></td><td></td><td></td>
      <td class="num"><b>${fa(money(band.credit))}</b></td>
      <td class="num" style="color:#dc2626"><b>${fa(money(band.debit))}</b></td>
      <td class="num"><b>${fa(money(band.net))}</b></td></tr>`);
    return `<div class="card card-pad" style="margin-top:10px">
      <div class="card-title">${esc(band.titleFa)}</div>
      ${liveTable(['شرح', 'گروه', 'درصد', 'بستانکار (ریال)', 'بدهکار (ریال)', 'خالص بند'], rows, rows.length)}
    </div>`;
  }).join('');

  box.innerHTML = `
    <div class="grid-stats">
      ${statCard({ico: '💼', tint: '#EFF6FF', accent: '#2563EB', val: fa(money(s.baseSalary)), label: 'پایهٔ حقوق (ریال)'})}
      ${statCard({ico: '🌡️', tint: '#FEF3C7', accent: '#D97706',
        val: fa(money(s.climate)), label: 'فوقالعادهٔ بدی آب و هوا ' + fa(Math.round(s.climatePercent * 10) / 10) + '٪'})}
      ${statCard({ico: '🧮', tint: '#EDE9FE', accent: '#7C3AED', val: fa(money(s.allowanceBase)), label: 'مبنای فوقالعادهها (ریال)'})}
      ${statCard({ico: '⭐', tint: '#FEF2F2', accent: '#B91C1C',
        val: fa(money(s.special || 0)), label: 'فوقالعادهٔ ویژه (۳۰٪ مبنای فوقالعادهها)'})}
      ${statCard({ico: '🎯', tint: '#F0F9FF', accent: '#0369A1',
        val: fa(money(s.attractionBase || 0)), label: 'مبنای جذب (پایه + گرمسیری + ویژه)'})}
      ${statCard({ico: '➕', tint: '#DCFCE7', accent: '#16A34A', val: fa(money(s.gross)), label: 'جمع ناخالص (ریال)'})}
      ${statCard({ico: '➖', tint: '#FEE2E2', accent: '#DC2626', val: fa(money(s.deductions)), label: 'جمع کسور (ریال)'})}
      ${statCard({ico: '💵', tint: '#DCFCE7', accent: '#16A34A', val: fa(money(s.net)), label: 'خالص پرداختی (ریال)'})}
    </div>
    ${bandHtml}
    <div class="card card-pad" style="margin-top:10px">
      <div class="card-title">جمع‌های فیش</div>
      <div class="muted" style="font-size:.74rem;margin-top:6px">
        کسور سهم کارفرما (صندوق بازنشستگی نفت): <b class="num">${fa(money(s.fundEmployer || 0))}</b> ریال
      </div>
      ${s.veteranApplied ? `<div class="muted" style="font-size:.74rem;margin-top:6px">
        🎖️ امتیاز ایثارگری (${esc(s.veteranGroupFa || '')}) اعمال شد — معافیت مالیاتی
        <b class="num">${fa(money(s.veteranTaxExempt || 0))}</b> ریال و معافیت کسور
        <b class="num">${fa(money(s.veteranPensionExempt || 0))}</b> ریال
      </div>` : ''}
      <div class="slip-total" style="margin-top:8px;font-size:1rem">
        <span>خالص پرداختی (ریال)</span><span class="num"><b>${fa(money(s.net))}</b></span>
      </div>
    </div>
    <div class="card card-pad" style="margin-top:10px;font-size:.8rem;line-height:1.9">
      ${esc(OIL_RESULT.noteFa || '').split('\n').join('<br>')}
    </div>`;
  // سطرهای دستی همین محاسبه را در جدول ویرایش نشان می‌دهیم؛ فقط وقتی جدول خالی است تا
  // هنگام تایپ، فوکوس از دست نرود.
  const extraBody = document.getElementById('oilExtraRows');
  const extraInput = (OIL_RESULT.input && OIL_RESULT.input.extraLines) || [];
  if (extraBody && extraBody.children.length === 0 && extraInput.length) {
    for (const line of extraInput) oilAddExtraRow(line.band, line.side, line.titleFa, line.amount);
  }
  oilSyncDriven(s);
}

/** درصدهای جدولی (محل سکونت و پایهٔ سمت) را از فیش محاسبه‌شده در فیلدهای خواندنی می‌گذارد.
 *
 *  چرا از خروجی سرور: درصد مؤثر با جدول ۱/۲/۳ و ستون روزکار/نوبت‌کار تغییر می‌کند؛ اگر
 *  فرم عدد پیش‌فرض پارامتر را نشان می‌داد، کاربر فکر می‌کرد همان اعمال شده است. */
function oilSyncDriven(slip) {
  for (const a of ((OIL_DEFAULTS && OIL_DEFAULTS.allowances) || [])) {
    if (!a.drivenBy) continue;
    const el = document.getElementById('oilAllow_' + a.key);
    if (!el) continue;
    // درصدی که کاربر دستی عوض کرده، با مقدار جدول بازنویسی نمی‌شود.
    if (el.dataset.edited === '1') continue;
    const line = (slip.lines || []).find(l => l.key === a.key);
    el.value = line ? String(Math.round(line.percent * 10) / 10) : '0';
  }
}

/** عنوان فارسی مدرک انتخاب‌شده. */
function oilDegreeTitle() {
  const key = oilSelect('oilDegree');
  if (!key) return '';
  const hit = ((OIL_DEFAULTS && OIL_DEFAULTS.degrees) || []).find(d => d.key === key);
  return hit ? hit.titleFa : key;
}

/** مبلغ فوق‌العادهٔ تخصصی از سطرهای فیش (صفر اگر مدرک مشمول نباشد). */
function oilDegreeAmount(slip) {
  // سهم ضریب مدرک از همان سطر «فوق‌العادهٔ تخصصی شغل» می‌آید؛ سطر جداگانه‌ای در فیش نیست
  // (درصد مصوب همان دوره مبلغ کل است و ضریب مدرک دوباره به آن اضافه نمی‌شود).
  return Number(slip.degreeAmount || 0);
}

/** عنوان فارسی پایهٔ سمت انتخاب‌شده (جدول ۳). */
function oilGradeTitle() {
  const key = oilSelect('oilJobGrade');
  if (!key) return '';
  const hit = ((OIL_DEFAULTS && OIL_DEFAULTS.jobGrades) || []).find(g => g.key === key);
  return hit ? hit.titleFa : key;
}

/** عنوان فارسی محل سکونت انتخاب‌شده (جدول ۱ و ۲). */
function oilResidenceTitle() {
  const key = oilSelect('oilResidence');
  if (!key) return '';
  const hit = ((OIL_DEFAULTS && OIL_DEFAULTS.residences) || []).find(r => r.key === key);
  return hit ? hit.titleFa : key;
}

/** سطر فوق‌العادهٔ تخصصی شغل (جدول ۳ + ضریب مدرک) با درصد و مبلغ. */
function oilAttractionLine(slip) {
  return (slip.lines || []).find(l => l.key === 'attractionSpecialty') || { percent: 0, amount: 0 };
}

/** با انتخاب کارمند، پایهٔ حقوق و محل سکونت پرونده در فرم می‌نشیند. */
function oilOnEmployee() {
  const hit = OIL_EMPLOYEES.find(e => e.id === oilSelect('oilEmployee'));
  const field = document.getElementById('oilBaseSalary');
  if (hit && field) {
    const base = Number(hit.baseSalary || 0);
    if (base > 0) field.value = String(base);
  }
  // محل سکونت پرونده: درصد جدول ۱ و ۲ از همین کلید خوانده می‌شود.
  const site = document.getElementById('oilResidence');
  if (hit && site) site.value = hit.residenceSite || '';
  // گروه شغلی پرونده: پایهٔ سمت پیشنهادی جدول ۳ (نگاشت پارامتریک) در فرم می‌نشیند تا
  // نتیجهٔ ابلاغ ترفیع خودکار در فیش دیده شود؛ کاربر هنوز می‌تواند آن را عوض کند.
  const grade = document.getElementById('oilJobGrade');
  if (hit && grade && hit.jobGradeSite) grade.value = hit.jobGradeSite;
  oilCalcSoon();
}

/** با نوبت‌کار کردن کارمند، روزهای ماه برای نوبت کامل پر می‌شود. */
function oilOnShift() {
  const days = document.getElementById('oilShiftDays');
  const month = oilNum('oilDaysInMonth');
  if (days && oilSelect('oilShift') === 'yes' && (!days.value || Number(days.value) === 0)) {
    days.value = String(month || 30);
  }
  oilCalcSoon();
}

/* --------------------------- ثبت فیش در پرونده --------------------------- */

/** ثبت فیش محاسبه‌شده به‌عنوان رکورد «فیش حقوقی صندوق نفت». */
async function oilSave() {
  if (!OIL_RESULT || !OIL_RESULT.slip) { toast('ابتدا فیش محاسبه شود', 'error'); return; }
  const employeeId = oilSelect('oilEmployee');
  if (!employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  const s = OIL_RESULT.slip;
  const period = oilText('oilPeriod') || OIL_RESULT.period || '';
  const employeeName = (OIL_EMPLOYEES.find(e => e.id === employeeId) || {}).titleFa || employeeId;
  const res = await apiFetch('/api/v1/payroll/records', {
    method: 'POST',
    body: {
      kind: 'oilPayslips',
      title: 'فیش حقوقی صندوق نفت — ' + employeeName.split(' — ')[0] + ' — ' + period,
      employeeId: employeeId,
      fields: {
        period: period,
        baseSalary: String(s.baseSalary),
        climatePercent: String(s.climatePercent),
        climate: String(s.climate),
        allowanceBase: String(s.allowanceBase),
        attractionBase: String(s.attractionBase || 0),
        allowancesTotal: String(s.allowances),
        gross: String(s.gross),
        deductions: String(s.deductions),
        net: String(s.net),
        shift: oilSelect('oilShift'),
        shiftDays: String(oilNum('oilShiftDays')),
        overtimeHours: String(oilNum('oilOvertime')),
        degreeName: oilDegreeTitle(),
        degreePercent: String(s.degreePercent || 0),
        degreeAllowance: String(oilDegreeAmount(s)),
        jobGradeName: oilGradeTitle(),
        residenceName: oilResidenceTitle(),
        attractionPercent: String(oilAttractionLine(s).percent || 0),
        attractionAllowance: String(oilAttractionLine(s).amount || 0),
        note: 'مبنای فوقالعاده‌ها: ' + money(s.allowanceBase) + ' ریال',
      },
      noteFa: 'فیش نظام پرداخت صندوق بازنشستگی نفت برای دورهٔ ' + period,
    },
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = (res.data && res.data.id) || '';
  toast('فیش ثبت شد: ' + id);
  const saved = document.getElementById('oilSaved');
  if (saved) {
    saved.innerHTML = `<div class="card card-pad" style="margin-top:10px;font-size:.82rem;line-height:2">
      فیش <b class="ltr">${esc(id)}</b> ثبت شد — خالص <b class="num">${fa(money(s.net))}</b> ریال.
      گام بعدی: تأیید فیش و ثبت پرداخت در فهرست رکوردهای همین نوع.
      <div class="row-actions" style="margin-top:8px">
        <button class="btn btn-sm btn-primary" onclick="oilAct('${esc(id)}','approve')">✅ تأیید فیش</button>
        <button class="btn btn-sm btn-ghost" onclick="oilAct('${esc(id)}','issue')">📄 صدور فیش</button>
        <button class="btn btn-sm btn-ghost" onclick="oilAct('${esc(id)}','pay')">💰 ثبت پرداخت</button>
      </div></div>`;
  }
}

/** انجام یک اقدام گردش‌کار روی فیش ثبت‌شده. */
async function oilAct(id, action) {
  const res = await apiFetch('/api/v1/payroll/records/' + encodeURIComponent(id) + '/action',
    { method: 'POST', body: { action: action } });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('اقدام انجام شد: ' + action);
}

/** چاپ فیش جاری (نسخهٔ کاغذی همان تفکیک اقلام). */
function oilPrint() {
  if (!OIL_RESULT || !OIL_RESULT.slip) { toast('ابتدا فیش محاسبه شود', 'error'); return; }
  const s = OIL_RESULT.slip;
  const employee = (OIL_EMPLOYEES.find(e => e.id === oilSelect('oilEmployee')) || {}).titleFa || '';
  // چاپ هم مانند صفحه، ساختار سه‌بندی فیش رسمی را دارد: هر بند با دو ستون بستانکار/بدهکار.
  const bandTables = (s.bands || []).map(band => {
    const rows = (band.lines || []).map(l => `<tr>
    <td>${esc(l.titleFa)}</td>
    <td>${l.percent ? fa(Math.round(l.percent * 10) / 10) + '٪' : '—'}</td>
    <td>${l.groupFa === 'کسور' ? '' : fa(money(l.amount))}</td>
    <td>${l.groupFa === 'کسور' ? fa(money(l.amount)) : ''}</td>
    <td>${esc(l.detailFa || '')}</td></tr>`).join('');
    return `<h2>${esc(band.titleFa)}</h2>
    <table><thead><tr><th>شرح</th><th>درصد</th><th>بستانکار (ریال)</th><th>بدهکار (ریال)</th><th>توضیح</th></tr></thead>
    <tbody>${rows}
    <tr><td><b>جمع ${esc(band.titleFa)}</b></td><td></td>
      <td><b>${fa(money(band.credit))}</b></td><td><b>${fa(money(band.debit))}</b></td>
      <td><b>خالص بند: ${fa(money(band.net))}</b></td></tr></tbody></table>`;
  }).join('');
  const win = window.open('', '_blank');
  if (!win) { toast('پنجرهٔ چاپ باز نشد', 'error'); return; }
  win.document.write(`<!doctype html><html dir="rtl" lang="fa"><head><meta charset="utf-8">
    <title>فیش حقوقی صندوق نفت</title>
    <style>body{font-family:Vazirmatn,Tahoma,sans-serif;padding:24px;color:#111}
    h1{font-size:18px;margin:0 0 4px} .sub{color:#555;font-size:12px;margin-bottom:14px}
    table{width:100%;border-collapse:collapse;margin-top:10px}
    th,td{border:1px solid #ccc;padding:6px;font-size:12px;text-align:right}
    th{background:#f3f4f6} h2{font-size:13px;margin:16px 0 4px}
    .tot{margin-top:12px;font-size:13px;line-height:2}</style></head><body>
    <h1>فیش حقوقی — نظام پرداخت صندوق بازنشستگی نفت</h1>
    <div class="sub">${esc(employee)} | دورهٔ ${fa(oilText('oilPeriod'))}</div>
    ${bandTables}
    <div class="tot">
      پایهٔ حقوق: <b>${fa(money(s.baseSalary))}</b> ریال ·
      فوق‌العادهٔ بدی آب و هوا: <b>${fa(money(s.climate))}</b> ریال ·
      مبنای فوق‌العاده‌ها: <b>${fa(money(s.allowanceBase))}</b> ریال<br>
      جمع ناخالص: <b>${fa(money(s.gross))}</b> ریال ·
      جمع کسور: <b>${fa(money(s.deductions))}</b> ریال ·
      خالص پرداختی: <b>${fa(money(s.net))}</b> ریال
    </div></body></html>`);
  win.document.close();
  win.focus();
  win.print();
}

/* ------------------------------ ساخت فرم ------------------------------ */

/** یک ردیف ورودی عددی. */
function oilField(id, label, value, unit, hint) {
  return `<div class="field"><label>${esc(label)}${unit ? ` <span class="muted">(${esc(unit)})</span>` : ''}</label>
    <input id="${id}" class="num" value="${esc(value)}" placeholder="${esc(hint || '')}" oninput="oilCalcSoon()"></div>`;
}

/** صفحهٔ «فیش حقوقی صندوق نفت». */
function pageOilPayslipLive() {
  if (!isLive()) return offlineFallback('oilPayslip');
  liveLoad(async () => {
    const res = await apiFetch('/api/v1/payroll/oil-payslip-defaults');
    if (!res.ok) return liveErrorBox(res.message);
    OIL_DEFAULTS = res.data || {};
    OIL_EMPLOYEES = OIL_DEFAULTS.employees || [];
    const def = OIL_DEFAULTS;
    const params = def.params || {};
    const today = def.today || '';
    const period = def.period || '';
    const empOpts = OIL_EMPLOYEES.map(e => `<option value="${esc(e.id)}">${esc(e.titleFa)}</option>`).join('');
    const firstBase = (OIL_EMPLOYEES[0] && OIL_EMPLOYEES[0].baseSalary) || 0;

    // پله‌های مالیات به شکل «سقف:نرخ» برای ویرایش پیش از اجرا (نرخ درصدی نوشته می‌شود).
    const bracketsText = (params.taxBrackets || [])
      .map(b => `${b.upto}:${Math.round((b.rate || 0) * 10000) / 100}`).join(',');

    // ضرائب و نرخ‌های قابل ویرایش: هر پارامتر oilpay.* که کادر اختصاصی ندارد به‌صورت
    // کادر عددی می‌آید (درصد گرمسیری و درصد اقلام، کادر خودشان را در همین فرم دارند).
    const hasDedicatedField = key => key === 'oilpay.climatePercent' || key.indexOf('oilpay.allowance.') === 0;
    const coeffFields = (def.paramViews || []).filter(p => !hasDedicatedField(p.key)).map(p => `
      <div class="field"><label>${esc(p.titleFa || p.key)} <span class="muted">(${esc(p.unitFa || '')})</span></label>
        <input class="num" data-oilparam="${esc(p.key)}" value="${esc(p.value || '')}" oninput="oilCalcSoon()"
          title="${esc(p.key)}"></div>`).join('') + `
      <div class="field"><label>پله‌های مالیات <span class="muted">(سقف:نرخ با کاما — درصد)</span></label>
        <input class="num" data-oilparam="oilpay.tax.brackets" value="${esc(bracketsText)}" oninput="oilCalcSoon()"
          title="نمونه: 50000000:10,100000000:15,200000000:20,0:30 — سقف صفر یعنی آخرین پله"></div>`;

    // اقلام فوق‌العاده: قلم ثابت ورودی دارد و قلم جدولی (محل سکونت/پایهٔ سمت) خواندنی است.
    const allowanceFields = (def.allowances || []).map(a => {
      const baseFa = a.base === 'baseSalary' ? 'از پایهٔ حقوق'
        : (a.base === 'attractionBase' ? 'از مبنای جذب' : 'از مبنای فوق‌العاده‌ها');
      if (a.fixed) {
        return `<div class="field"><label>${esc(a.titleFa)} <span class="muted">(مبلغ ثابت ماهانه — ریال)</span></label>
          <input id="oilAmount_${esc(a.key)}" class="num" value="${a.amount ? esc(String(a.amount)) : ''}"
            placeholder="خالی = بدون این قلم" oninput="oilCalcSoon()"></div>`;
      }
      if (a.drivenBy) {
        const fromFa = a.drivenBy === 'residence' ? 'از جدول ۱ و ۲ — محل سکونت' : 'از جدول ۳ — پایهٔ سمت';
        return `<div class="field"><label>${esc(a.titleFa)} <span class="muted">(${esc(baseFa)} · ${esc(fromFa)})</span></label>
          <input id="oilAllow_${esc(a.key)}" class="num" value="${esc(String(a.percent))}"
            data-driven="1" data-edited="0" oninput="oilOnAllowanceEdit(this)"
            title="پیش‌فرض از جدول آیین‌نامه می‌آید؛ اگر عدد را عوض کنید همان عدد در این محاسبه اعمال می‌شود و خالی‌کردن کادر آن را به جدول برمی‌گرداند"></div>`;
      }
      return `<div class="field">
      <label>${esc(a.titleFa)} <span class="muted">(${esc(baseFa)})</span></label>
      <input id="oilAllow_${esc(a.key)}" class="num" value="${esc(String(a.percent))}" oninput="oilCalcSoon()"></div>`;
    }).join('');

    const shiftOpts = (def.shiftOptions || []).map(s =>
      `<option value="${esc(s.key)}">${esc(s.titleFa)}</option>`).join('');

    // مقاطع تحصیلی با ضریب فوق‌العادهٔ تخصصی هر مقطع؛ گزینهٔ خالی یعنی مشمول نیست.
    const degreeOpts = ['<option value="">— بدون فوق‌العادهٔ تخصصی —</option>']
      .concat((def.degrees || []).map(d => `<option value="${esc(d.key)}">${esc(d.titleFa)} — ${fa(Math.round(d.percent * 10) / 10)}٪</option>`))
      .join('');

    // جدول ۳: پایهٔ سمت → حق جذب عمومی + حق مقام (Y).
    const gradeOpts = ['<option value="">— بدون پایهٔ سمت (بدون جذب) —</option>']
      .concat((def.jobGrades || []).map(g =>
        `<option value="${esc(g.key)}">${esc(g.titleFa)} — ${fa(Math.round(g.total * 10) / 10)}٪</option>`))
      .join('');

    // جدول ۱ و ۲: محل سکونت → محرومیت و صرف وقت (روزکار/نوبت‌کار).
    const residenceOpts = ['<option value="">— بدون محل سکونت (درصد پیش‌فرض شرکت) —</option>']
      .concat((def.residences || []).map(r =>
        `<option value="${esc(r.key)}">${esc(r.titleFa)} — محرومیت ${fa(Math.round(r.deprivation * 10) / 10)}٪ · صرف وقت ${fa(Math.round(r.timeConsumedDay * 10) / 10)}٪/${fa(Math.round(r.timeConsumedShift * 10) / 10)}٪</option>`))
      .join('');

    const form = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">💼 نظام پرداخت صندوق بازنشستگی نفت</div>
      <div class="form-grid">
        <div class="field"><label>کارمند <span class="req">*</span></label>
          <select id="oilEmployee" onchange="oilOnEmployee()">${empOpts}</select></div>
        <div class="field"><label>دورهٔ حقوق (ماه)</label>
          <input id="oilPeriod" class="num" value="${esc(period)}" oninput="oilCalcSoon()"></div>
        ${oilField('oilBaseSalary', 'پایهٔ حقوق', String(firstBase), 'ریال', 'از پروندهٔ کارمند پر می‌شود')}
        ${oilField('oilClimatePercent', 'درصد فوق‌العادهٔ بدی آب و هوا (گرمسیری)', String(params.climatePercent || 0), '٪', 'مصوب شرکت')}
        ${oilField('oilDaysInMonth', 'روزهای ماه', String(def.daysInMonth || 30), 'روز')}
        <div class="field"><label>محل سکونت (جدول ۱ و ۲)</label>
          <select id="oilResidence" onchange="oilCalcSoon()">${residenceOpts}</select></div>
        <div class="field"><label>پایهٔ سمت (جدول ۳ — حق جذب عمومی و حق مقام)</label>
          <select id="oilJobGrade" onchange="oilCalcSoon()">${gradeOpts}</select></div>
        <div class="field"><label>مدرک تحصیلی (ضریب Z جدول ۳)</label>
          <select id="oilDegree" onchange="oilCalcSoon()">${degreeOpts}</select></div>
        <div class="field"><label>وضعیت نوبت‌کاری</label>
          <select id="oilShift" onchange="oilOnShift()">${shiftOpts}</select></div>
        ${oilField('oilShiftDays', 'روزهای نوبت‌کاری در ماه', String(def.daysInMonth || 30), 'روز')}
        ${oilField('oilOvertime', 'اضافه‌کاری', '0', 'ساعت')}
        ${oilField('oilDeductions', 'سایر کسور (وام/مساعده)', '0', 'ریال')}
      </div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">📊 جدول ۱ تا ۳: فوق‌العاده‌های مصوب (درصدی از مبنای فوق‌العاده‌ها)</div>
      <div class="form-grid">${allowanceFields || '<div class="muted">قلمی تعریف نشده است.</div>'}</div>
      <div class="muted" style="font-size:.74rem;margin-top:6px">${esc(def.noteFa || '').split('\n').join('<br>')}</div>
      <div style="border-top:1px dashed #e2e8f0;margin:12px 0 10px"></div>
      <div class="card-title" style="font-size:.86rem">⚙️ ضرائب و نرخ‌های این محاسبه (پیش از اجرا قابل تغییر است)</div>
      <div class="form-grid">${coeffFields}</div>
      <div style="border-top:1px dashed #e2e8f0;margin:12px 0 10px"></div>
      <div class="card-title" style="font-size:.86rem">🧾 سطرهای دستی فیش (تطبیق سقف، وام، درمان، پس‌انداز و ...)</div>
      <div class="muted" style="font-size:.74rem;margin-bottom:6px">هر سطر در جمع همان بند خودش می‌نشیند؛ مبلغ از مصوبه یا قرارداد همان دوره می‌آید و در رابط ساخته نمی‌شود.</div>
      <div class="row-actions" style="margin-bottom:8px">
        <select id="oilExtraBand" style="max-width:360px">
          <option value="1">بند ۱ — حقوق و فوق‌العاده‌ها و کسورات قانونی</option>
          <option value="2">بند ۲ — کسورات تطبیق جهت رعایت سقف</option>
          <option value="3" selected>بند ۳ — سایر پرداخت‌ها و کسورات</option>
        </select>
        <button class="btn btn-sm btn-ghost" onclick="oilExtraAdd()">+ افزودن سطر</button>
      </div>
      <table class="table" style="width:100%">
        <thead><tr>
          <th style="width:230px">بند</th><th>شرح قلم</th>
          <th style="width:150px">مبلغ (ریال)</th><th style="width:160px">نوع</th><th style="width:40px"></th>
        </tr></thead>
        <tbody id="oilExtraRows"></tbody>
      </table>
      <div class="muted" style="font-size:.74rem;margin-top:6px">این اعداد فقط برای همین محاسبه‌اند و پروفایل شرکت را تغییر نمی‌دهند؛ برای ماندگارکردن مصوبهٔ تازه، آن را در پارامترهای شرکت ثبت کنید.</div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">🧾 فیش محاسبه‌شده</div>
      <div id="oilSlip"><div class="muted" style="font-size:.78rem">در حال محاسبه...</div></div>
      <div class="row-actions" style="margin-top:10px">
        <button class="btn btn-primary" onclick="oilCalc()">🧮 محاسبهٔ مجدد</button>
        <button class="btn btn-ghost" onclick="oilPrint()">🖨️ چاپ فیش</button>
        <button class="btn btn-ghost" onclick="oilSave()">💾 ثبت فیش در پرونده</button>
      </div>
      <div id="oilSaved"></div>
    </div>`;

    const paramRows = (def.paramViews || []).map(p => `<tr>
      <td>${esc(p.titleFa)}</td><td class="ltr" style="font-size:.72rem">${esc(p.key)}</td>
      <td class="num">${fa(p.value)}</td><td>${esc(p.unitFa)}</td></tr>`);

    const shell = form + (paramRows.length ? `<div class="card card-pad">
      <div class="card-title">پارامترهای نظام پرداخت صندوق نفت در شرکت فعال</div>
      ${liveTable(['پارامتر', 'کلید', 'مقدار', 'واحد'], paramRows, paramRows.length)}
      <div class="muted" style="font-size:.74rem;margin-top:6px">تاریخ امروز سرور: <b class="num">${fa(today)}</b></div>
    </div>` : '');

    setTimeout(() => { oilOnEmployee(); oilCalc(); }, 80);
    return shell;
  });
  return liveShell('فیش حقوقی صندوق نفت',
    'پایهٔ حقوق، فوق‌العادهٔ بدی آب و هوا و فوق‌العاده‌های مصوب روی مبنای فوق‌العاده‌ها',
    `<button class="btn btn-ghost" onclick="OIL_DEFAULTS=null;liveReload()">↻ تازه‌سازی پارامترها</button>`);
}

/* ثبت در مسیریاب زنده. */
LIVE_PAGES.oilPayslip = pageOilPayslipLive;