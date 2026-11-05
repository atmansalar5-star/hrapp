/* =====================================================================
   فرم محاسبهٔ مأموریت (ماژول حضور و غیاب)
   ---------------------------------------------------------------------
   چرا این صفحه: فوقالعادهٔ مأموریت از چند جزء ساخته می‌شود (سهم صبحانه، نهار، شام و خواب
   از نرخ روزانهٔ شرکت، اقامت در هتل/مهمانسرا، جبران امکانات شخصی و هزینه‌های ایاب‌وذهاب).
   این فرم همان اجزا را از کاربر می‌گیرد و محاسبه را از سرور می‌خواهد
   (`POST /api/v1/attendance/missions/allowance`)؛ هیچ قاعده‌ای در جاوااسکریپت نیست و
   مبلغ نهایی همان عددی است که موتور قواعد می‌دهد. ذخیرهٔ فرم یک رکورد «مأموریت» می‌سازد.
   ===================================================================== */

var MIS_DEFAULTS = null;   // پارامترهای پیش‌فرض شرکت
var MIS_RESULT = null;     // آخرین محاسبهٔ سرور (تفکیک سطرها)
var MIS_EMPLOYEES = null;  // فهرست کارکنان برای انتخاب
var MIS_TIMER = null;      // زمان‌سنج محاسبهٔ خودکار

/** فهرست بُعدهای فرم: کلید و برچسب فارسی. */
var MIS_YESNO = [{ key: 'no', titleFa: 'خیر' }, { key: 'yes', titleFa: 'بله' }];

/** مقدار عددی یک ورودی فرم (خالی = صفر). */
function misNum(id) {
  const el = document.getElementById(id);
  if (!el) return 0;
  const raw = (el.value || '').replace(/[^0-9.\-]/g, '');
  return Number(raw) || 0;
}

/** متن یک ورودی فرم. */
function misText(id) {
  const el = document.getElementById(id);
  return el ? (el.value || '').trim() : '';
}

/** مقدار انتخاب‌شدهٔ یک select. */
function misSelect(id) {
  const el = document.getElementById(id);
  return el ? el.value : '';
}

/** ساخت بدنهٔ درخواست محاسبه از فیلدهای فرم. */
function misInput() {
  const def = MIS_DEFAULTS || {};
  return {
    startDate: misText('misStartDate'),
    startTime: misText('misStartTime'),
    durationDays: misNum('misDuration'),
    perDiem: misNum('misPerDiem'),
    missionFactor: misNum('misFactor'),
    factors: {
      breakfast: misNum('misBreakfast') / 100,
      lunch: misNum('misLunch') / 100,
      dinner: misNum('misDinner') / 100,
      lodging: misNum('misLodging') / 100,
    },
    usePersonalFacilities: misSelect('misUsePersonal') === 'yes',
    personalFacilitiesAmount: misNum('misPersonalAmount'),
    useHotel: misSelect('misUseHotel') === 'yes',
    hotelCost: misNum('misHotelCost'),
    useGuesthouse: misSelect('misUseGuesthouse') === 'yes',
    guesthouseCost: misNum('misGuesthouseCost'),
    taxiCost: misNum('misTaxi'),
    busCost: misNum('misBus'),
    mileageKm: misNum('misKm'),
    mileageRate: misNum('misKmRate') || Number(def.mileageRate || 0),
    personalFacilitiesPercent: Number(def.personalPercent || 0),
  };
}

/** درخواست محاسبه از سرور و بازترسیم نتیجه (با تأخیر کوتاه برای تایپ). */
function misCalcSoon() {
  if (MIS_TIMER) clearTimeout(MIS_TIMER);
  MIS_TIMER = setTimeout(misCalc, 250);
}

/** محاسبهٔ فوقالعادهٔ مأموریت روی سرور. */
async function misCalc() {
  const box = document.getElementById('misResult');
  if (box) box.innerHTML = '<div class="muted" style="font-size:.78rem">⏳ در حال محاسبه...</div>';
  const res = await apiFetch('/api/v1/attendance/missions/allowance', { method: 'POST', body: misInput() });
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    return;
  }
  MIS_RESULT = res.data || {};
  misRenderResult();
}

/** نمایش تفکیک محاسبه و جمع قابل پرداخت. */
function misRenderResult() {
  const box = document.getElementById('misResult');
  if (!box || !MIS_RESULT) return;
  const a = MIS_RESULT.allowance || {};
  const lines = a.lines || [];
  const rows = lines.map(l => `<tr>
    <td>${esc(l.titleFa)}</td>
    <td class="num">${l.factor ? fa(Math.round(l.factor * 1000) / 10) + '٪' : '—'}</td>
    <td class="num">${l.amount ? fa(money(l.amount)) : '<span class="muted">—</span>'}</td>
    <td style="font-size:.74rem">${esc(l.detailFa || '—')}</td></tr>`);
  box.innerHTML = `
    <div class="grid-stats">
      ${statCard({ ico: '💵', tint: '#DCFCE7', accent: '#16A34A', val: fa(money(a.total || 0)), label: 'فوقالعادهٔ مأموریت قابل پرداخت (ریال)' })}
      ${statCard({ ico: '🧮', tint: '#EFF6FF', accent: '#2563EB', val: fa(money(a.dailyRate || 0)), label: 'نرخ روزانه (ریال)' })}
      ${statCard({ ico: '📅', tint: '#FEF3C7', accent: '#D97706', val: fa(a.days || 0), label: 'مدت مأموریت (روز)' })}
      ${statCard({ ico: '➗', tint: '#EDE9FE', accent: '#7C3AED', val: fa(a.factorsTotal || 0), label: 'جمع عوامل' })}
    </div>
    <div class="card card-pad" style="margin-top:10px">
      <div class="card-title">تفکیک محاسبه</div>
      ${liveTable(['جزء', 'ضریب', 'مبلغ (ریال)', 'شرح'], rows, rows.length)}
      ${a.noteFa ? `<div class="muted" style="font-size:.76rem;margin-top:6px">${esc(a.noteFa)}</div>` : ''}
    </div>`;
}

/** ذخیرهٔ فرم به‌صورت رکورد «مأموریت» (با مبلغ محاسبه‌شدهٔ سرور). */
async function misSave() {
  if (!MIS_RESULT || !MIS_RESULT.allowance) { toast('ابتدا محاسبه انجام شود', 'error'); return; }
  const perDiem = misNum('misPerDiem');
  const destination = misText('misDestination');
  if (!destination) { toast('مقصد مأموریت الزامی است', 'error'); return; }
  const employeeId = misSelect('misEmployee');
  if (!employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  const startDate = misText('misStartDate');
  if (!startDate) { toast('تاریخ شروع مأموریت الزامی است', 'error'); return; }

  const allowance = MIS_RESULT.allowance;
  const fields = {
    destination: destination,
    fromDate: startDate,
    toDate: MIS_RESULT.toDate || startDate,
    purpose: misText('misPurpose'),
    startTime: misText('misStartTime'),
    durationDays: String(misNum('misDuration')),
    missionFactor: String(misNum('misFactor')),
    factorBreakfast: String(misNum('misBreakfast')),
    factorLunch: String(misNum('misLunch')),
    factorDinner: String(misNum('misDinner')),
    factorLodging: String(misNum('misLodging')),
    usePersonalFacilities: misSelect('misUsePersonal'),
    personalFacilitiesAmount: String(misNum('misPersonalAmount')),
    useHotel: misSelect('misUseHotel'),
    hotelCost: String(misNum('misHotelCost')),
    useGuesthouse: misSelect('misUseGuesthouse'),
    guesthouseCost: String(misNum('misGuesthouseCost')),
    taxiCost: String(misNum('misTaxi')),
    busCost: String(misNum('misBus')),
    mileageKm: String(misNum('misKm')),
    mileageRate: String(misNum('misKmRate')),
    allowanceTotal: String(Math.round(allowance.total || 0)),
  };
  const title = 'مأموریت — ' + destination + ' — ' + esc(startDate) + ' (' + fa(misNum('misDuration')) + ' روز)';
  const res = await apiFetch('/api/v1/attendance/records', {
    method: 'POST',
    body: { kind: 'missions', title: title, employeeId: employeeId, fields: fields,
      noteFa: 'فرم محاسبهٔ مأموریت — فوقالعادهٔ محاسبه‌شده: ' + money(allowance.total || 0) + ' ریال' },
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = (res.data && res.data.id) || '';
  toast('مأموریت ذخیره شد: ' + id);
  const box = document.getElementById('misSaved');
  if (box) {
    box.innerHTML = `<div class="card card-pad" style="margin-top:10px;font-size:.82rem;line-height:2">
      رکورد مأموریت <b class="ltr">${esc(id)}</b> ساخته شد با فوقالعادهٔ قابل پرداخت
      <b class="num">${fa(money(allowance.total || 0))}</b> ریال.
      <div class="row-actions" style="margin-top:8px">
        <button class="btn btn-sm btn-primary" onclick="misSubmit('${esc(id)}')">📤 ارسال برای تأیید</button>
        <button class="btn btn-sm btn-ghost" onclick="go('statsMission')">📊 آمار مأموریت‌ها</button>
      </div></div>`;
  }
}

/** ارسال رکورد مأموریت برای تأیید (گردشکار استاندارد ماژول). */
async function misSubmit(id) {
  const res = await apiFetch('/api/v1/attendance/records/' + encodeURIComponent(id) + '/action',
    { method: 'POST', body: { action: 'submit' } });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('مأموریت برای تأیید ارسال شد');
}

/** یک ردیف ورودی عددی با واحد. */
function misField(id, label, value, unit, hint) {
  return `<div class="field"><label>${esc(label)}${unit ? ` <span class="muted">(${esc(unit)})</span>` : ''}</label>
    <input id="${id}" class="num" value="${esc(value)}" placeholder="${esc(hint || '')}" oninput="misCalcSoon()"></div>`;
}

/** فهرست کشویی بله/خیر برای یک گزینهٔ فرم. */
function misYesNo(id, value) {
  const opts = MIS_YESNO.map(o => `<option value="${o.key}" ${o.key === value ? 'selected' : ''}>${esc(o.titleFa)}</option>`).join('');
  return `<select id="${id}" onchange="misCalcSoon()">${opts}</select>`;
}

/** صفحهٔ «فرم محاسبهٔ مأموریت». */
function pageMissionCalcLive() {
  if (!isLive()) return offlineFallback('missionCalc');
  liveLoad(async () => {
    const [defRes, empRes] = await Promise.all([
      apiFetch('/api/v1/attendance/missions/allowance-defaults'),
      MIS_EMPLOYEES ? Promise.resolve({ ok: true, data: { items: MIS_EMPLOYEES } }) : apiFetch('/api/v1/employee/employees?size=200'),
    ]);
    if (!defRes.ok) return liveErrorBox(defRes.message);
    MIS_DEFAULTS = defRes.data || {};
    if (empRes.ok && empRes.data) MIS_EMPLOYEES = empRes.data.items || [];
    const def = MIS_DEFAULTS;
    const factors = def.factors || {};
    const today = def.today || '';
    const empOpts = (MIS_EMPLOYEES || []).map(e =>
      `<option value="${esc(e.id)}">${esc(e.fullName)} — ${esc(e.id)}</option>`).join('');

    // فرم: اطلاعات مأموریت، عوامل، اقامت، امکانات شخصی و ایاب‌وذهاب.
    const form = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">📝 مشخصات مأموریت</div>
      <div class="form-grid">
        <div class="field"><label>کارمند <span class="req">*</span></label><select id="misEmployee">${empOpts}</select></div>
        <div class="field"><label>مقصد <span class="req">*</span></label><input id="misDestination" placeholder="شهر یا محل مأموریت"></div>
        <div class="field"><label>تاریخ شروع مأموریت (شمسی) <span class="req">*</span></label>
          <input id="misStartDate" class="num" value="${esc(today)}" placeholder="1405/08/13" oninput="misCalcSoon()"></div>
        <div class="field"><label>ساعت شروع مأموریت</label>
          <input id="misStartTime" class="num" value="07:30" placeholder="07:30" oninput="misCalcSoon()"></div>
        ${misField('misDuration', 'مدت مأموریت', '1', 'روز', 'مثلاً ۲ یا ۰٫۵')}
        ${misField('misFactor', 'ضریب عوامل مأموریت', String(def.missionFactor || 1), 'ضریب', 'پیش‌فرض ۱')}
        ${misField('misPerDiem', 'فوقالعادهٔ روزانهٔ مأموریت', String(def.perDiem || 0), 'ریال', 'از پارامتر شرکت')}
        <div class="field full"><label>هدف مأموریت</label><input id="misPurpose" placeholder="مثلاً جلسهٔ هماهنگی ستاد"></div>
      </div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">🧮 عوامل مأموریت (سهم هر وعده و خواب از نرخ روزانه)</div>
      <div class="form-grid">
        ${misField('misBreakfast', 'عامل صبحانه', String(Math.round((factors.breakfast || 0) * 100)), '٪')}
        ${misField('misLunch', 'عامل نهار', String(Math.round((factors.lunch || 0) * 100)), '٪')}
        ${misField('misDinner', 'عامل شام', String(Math.round((factors.dinner || 0) * 100)), '٪')}
        ${misField('misLodging', 'عامل خواب', String(Math.round((factors.lodging || 0) * 100)), '٪')}
      </div>
      <div class="muted" style="font-size:.74rem;margin-top:6px">اگر پذیرایی یکی از وعده‌ها با میزبان باشد، درصد همان وعده را صفر بگذارید.</div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">🏨 اقامت، امکانات شخصی و ایاب‌وذهاب</div>
      <div class="form-grid">
        <div class="field"><label>استفاده از امکانات شخصی</label>${misYesNo('misUsePersonal', 'no')}</div>
        ${misField('misPersonalAmount', 'مبلغ جبران امکانات شخصی', '0', 'ریال', 'خالی = درصد شرکت')}
        <div class="field"><label>استفاده از هتل</label>${misYesNo('misUseHotel', 'no')}</div>
        ${misField('misHotelCost', 'هزینهٔ هتل', '0', 'ریال')}
        <div class="field"><label>استفاده از مهمانسرا</label>${misYesNo('misUseGuesthouse', 'no')}</div>
        ${misField('misGuesthouseCost', 'هزینهٔ مهمانسرا', '0', 'ریال')}
        ${misField('misTaxi', 'هزینهٔ تاکسی از محل اقامت تا فرودگاه', '0', 'ریال')}
        ${misField('misBus', 'هزینهٔ اتوبوس', '0', 'ریال')}
        ${misField('misKm', 'مسافت کرایهٔ کیلومتری', '0', 'کیلومتر')}
        ${misField('misKmRate', 'نرخ هر کیلومتر', String(def.mileageRate || 0), 'ریال')}
      </div>
      <div class="row-actions" style="margin-top:10px">
        <button class="btn btn-primary" onclick="misCalc()">🧮 محاسبهٔ فوقالعادهٔ مأموریت</button>
        <button class="btn btn-ghost" onclick="misSave()">💾 ذخیرهٔ مأموریت</button>
      </div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">💵 فوقالعادهٔ مأموریت قابل پرداخت</div>
      <div id="misResult"><div class="muted" style="font-size:.78rem">برای محاسبه، دکمهٔ «محاسبهٔ فوقالعادهٔ مأموریت» را بزنید یا فیلدها را تغییر دهید.</div></div>
      <div id="misSaved"></div>
    </div>`;

    const paramRows = (def.params || []).map(p => `<tr>
      <td>${esc(p.titleFa)}</td>
      <td class="ltr" style="font-size:.72rem">${esc(p.key)}</td>
      <td class="num">${fa(p.value)}</td>
      <td>${esc(p.unitFa)}</td></tr>`);

    const shell = form +
      (paramRows.length ? `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">پارامترهای مأموریت شرکت فعال</div>
        ${liveTable(['پارامتر', 'کلید', 'مقدار', 'واحد'], paramRows, paramRows.length)}
      </div>` : '') +
      statNote(def.noteFa);

    // نخستین محاسبه پس از چیده‌شدن فرم انجام می‌شود تا کاربر عدد آماده ببیند.
    setTimeout(misCalc, 60);
    return statNote('این فرم یک رکورد «مأموریت» می‌سازد: پس از ذخیره، با دکمهٔ «ارسال برای تأیید» به گردش تأیید می‌رود و در تقویم کارکرد و کارتابل دیده می‌شود.') + shell;
  });
  return liveShell('فرم محاسبهٔ مأموریت',
    'محاسبهٔ فوقالعادهٔ مأموریت با عوامل وعده‌ها و خواب، اقامت، امکانات شخصی و ایاب‌وذهاب',
    `<button class="btn btn-ghost" onclick="MIS_DEFAULTS=null;liveReload()">↻ تازه‌سازی پارامترها</button>`);
}

/* ثبت در مسیریاب زنده (پیش از قالب نمایشی بررسی می‌شود). */
LIVE_PAGES.missionCalc = pageMissionCalcLive;
