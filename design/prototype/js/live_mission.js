/* =====================================================================
   فرم محاسبهٔ مأموریت (ماژول حضور و غیاب)
   ---------------------------------------------------------------------
   چرا این صفحه: تسویهٔ مأموریت از چند بخش ساخته می‌شود — فوقالعادهٔ روزانه (سهم صبحانه،
   نهار، شام و خواب از نرخ روزانهٔ شرکت)، اقامت، ایاب‌وذهاب، قبض تاکسی و فاکتورهای مورد تأیید،
   اعزام پزشکی (هزینه بر پایهٔ بیمار یا همراه بیمار) و مأموریت خارج از کشور (پرداخت ریالی و
   ارزی). محاسبه روی سرور انجام می‌شود (`POST /api/v1/attendance/missions/allowance`) و
   این فایل هیچ قاعده‌ای ندارد؛ فقط فیلدها را می‌فرستد و نتیجه را می‌چیند.

   ذخیره: یک رکورد «مأموریت» با همهٔ مبالغ محاسبه‌شده، و برای هر سند پشتیبان یک رکورد
   «سند و فاکتور مأموریت» (که پس از تأیید مالی در جمع می‌آید).
   ===================================================================== */

var MIS_DEFAULTS = null;    // پارامترها و فهرست‌های مرجع از سرور
var MIS_RESULT = null;      // آخرین پاسخ محاسبه (فوقالعاده + تسویه)
var MIS_EMPLOYEES = null;   // فهرست کارکنان
var MIS_DOCS = [];          // اسناد پشتیبان فرم (قبض تاکسی و فاکتورها)
var MIS_TIMER = null;       // زمان‌سنج محاسبهٔ خودکار

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

/** نوع مأموریت انتخاب‌شده. */
function misType() { return misSelect('misType') || 'administrative'; }

/** ساخت بدنهٔ درخواست محاسبه از فیلدهای فرم + اسناد. */
function misInput() {
  const def = MIS_DEFAULTS || {};
  return {
    missionType: misType(),
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
    // اعزام پزشکی
    beneficiary: misSelect('misBeneficiary') || 'patient',
    accommodationType: misSelect('misAccommodation'),
    accommodationCost: misNum('misAccommodationCost'),
    companionCount: misNum('misCompanions'),
    companionSharePercent: Number(def.companionShare || 0),
    // مأموریت خارج از کشور
    currency: misSelect('misCurrency') || def.defaultCurrency || 'EUR',
    fxRate: misNum('misFxRate'),
    fxPerDiem: misNum('misFxPerDiem'),
    fxAmount: misNum('misFxAmount'),
    rialAmount: misNum('misRialAmount'),
    // اسناد پشتیبان (قبض تاکسی و فاکتورهای مورد تأیید)
    documents: misDocsPayload(),
  };
}

/** اسناد فرم به شکل ورودی موتور قواعد. */
function misDocsPayload() {
  return MIS_DOCS.map((doc, i) => ({
    key: 'D' + (i + 1),
    kind: doc.kind,
    docNo: doc.docNo,
    docDate: doc.docDate,
    amount: Number(doc.amount) || 0,
    currency: doc.currency || 'IRR',
    approved: !!doc.approved,
    approvedBy: doc.approvedBy || '',
    payer: doc.payer || 'company',
    noteFa: doc.noteFa || '',
  }));
}

/** درخواست محاسبه از سرور و بازترسیم نتیجه (با تأخیر کوتاه برای تایپ). */
function misCalcSoon() {
  if (MIS_TIMER) clearTimeout(MIS_TIMER);
  MIS_TIMER = setTimeout(misCalc, 250);
}

/** محاسبهٔ تسویهٔ مأموریت روی سرور. */
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

/** نمایش تفکیک تسویه، جمع ریالی و جمع ارزی. */
function misRenderResult() {
  const box = document.getElementById('misResult');
  if (!box || !MIS_RESULT) return;
  const s = MIS_RESULT.settlement || {};
  // تفکیک و اجزای نمایش از «تسویه» خوانده می‌شود، نه فوقالعادهٔ خام: در اعزام پزشکی
  // و مأموریت خارج از کشور، تسویه سهم خواب/وعدهها را اصلاح می‌کند.
  const a = s.allowance || MIS_RESULT.allowance || {};
  const rows = (s.lines || []).map(l => `<tr>
    <td>${esc(l.titleFa)}</td>
    <td class="num">${l.factor ? fa(Math.round(l.factor * 1000) / 10) + '٪' : '—'}</td>
    <td class="num">${l.amount ? fa(money(l.amount)) : '<span class="muted">—</span>'}</td>
    <td style="font-size:.74rem">${esc(l.detailFa || '—')}</td></tr>`);

  const foreign = s.foreign ? `<div class="card card-pad" style="margin-top:10px;background:var(--tint,#EFF6FF);border:0">
      <div class="card-title">💱 ${esc(s.foreign.currencyFa)} (${esc(s.foreign.currency)})</div>
      <div class="grid-stats">
        ${statCard({ ico: '💰', tint: '#EDE9FE', accent: '#7C3AED', val: fa(money(s.totalFx || 0)), label: 'جمع ارزی قابل پرداخت' })}
        ${statCard({ ico: '📈', tint: '#EFF6FF', accent: '#2563EB', val: fa(money(s.foreign.fxRate || 0)), label: 'نرخ تسعیر (ریال)' })}
        ${statCard({ ico: '🔁', tint: '#DCFCE7', accent: '#16A34A', val: fa(money(s.foreign.rialEquivalent || 0)), label: 'معادل ریالی ارز' })}
      </div>
      <div class="muted" style="font-size:.76rem;margin-top:6px">${esc(s.foreign.noteFa || '')}<br>
        جمع کل قابل ثبت (مبلغ ریالی + معادل ارز): <b class="num">${fa(money(s.totalRialEquivalent || 0))}</b> ریال</div>
    </div>` : '';

  const medical = s.medical ? `<div class="muted" style="font-size:.78rem;margin-top:8px">
      🩺 اعزام پزشکی — مشمول: <b>${esc(s.medical.beneficiaryFa)}</b> · اسکان: ${esc(s.medical.accommodationFa)}
      · سهم وعده‌ها: ${s.medical.mealsPayable ? fa(money(s.medical.mealsAmount || 0)) + ' ریال' : 'پرداخت نمی‌شود'}
      ${s.medical.noteFa ? ' — ' + esc(s.medical.noteFa) : ''}
    </div>` : '';

  box.innerHTML = `
    <div class="grid-stats">
      ${statCard({ ico: '💵', tint: '#DCFCE7', accent: '#16A34A', val: fa(money(s.totalRial || 0)), label: 'جمع ریالی قابل پرداخت (ریال)' })}
      ${statCard({ ico: '🏷️', tint: '#EFF6FF', accent: '#2563EB', val: esc(s.missionTypeFa || ''), label: 'نوع مأموریت' })}
      ${statCard({ ico: '🧾', tint: '#FEF3C7', accent: '#D97706', val: fa(money(s.documentsTotal || 0)), label: 'اسناد تأییدشدهٔ ریالی (ریال)' })}
      ${statCard({ ico: '⏳', tint: '#FEE2E2', accent: '#DC2626', val: fa(s.pendingDocuments || 0), label: 'سند در انتظار تأیید' })}
    </div>
    ${foreign}${medical}
    <div class="card card-pad" style="margin-top:10px">
      <div class="card-title">تفکیک محاسبه — نرخ روزانه <span class="num">${fa(money(a.dailyRate || 0))}</span> ریال،
        مدت <span class="num">${fa(a.days || 0)}</span> روز، جمع عوامل <span class="num">${fa(a.factorsTotal || 0)}</span></div>
      ${liveTable(['جزء', 'ضریب', 'مبلغ (ریال)', 'شرح'], rows, rows.length)}
      ${a.noteFa ? `<div class="muted" style="font-size:.76rem;margin-top:6px">${esc(a.noteFa)}</div>` : ''}
    </div>`;
}

/* ----------------------------- اسناد پشتیبان ----------------------------- */

/** افزودن یک سند تازه به فرم. */
function misAddDoc() {
  const types = (MIS_DEFAULTS && MIS_DEFAULTS.documentTypes) || [];
  MIS_DOCS.push({
    kind: (types[0] && types[0].key) || 'taxi', docNo: '', docDate: misText('misStartDate'),
    amount: 0, currency: 'IRR', approved: false, payer: 'company', noteFa: '',
  });
  misRenderDocs();
  misCalcSoon();
}

/** حذف یک سند از فرم. */
function misRemoveDoc(index) {
  MIS_DOCS.splice(index, 1);
  misRenderDocs();
  misCalcSoon();
}

/** به‌روزرسانی یک فیلد سند. */
function misDocField(index, field, value) {
  if (!MIS_DOCS[index]) return;
  MIS_DOCS[index][field] = field === 'approved' ? !!value : value;
  misCalcSoon();
}

/** نمایش جدول اسناد پشتیبان (قبض تاکسی و فاکتورها). */
function misRenderDocs() {
  const box = document.getElementById('misDocs');
  if (!box) return;
  if (!MIS_DOCS.length) {
    box.innerHTML = '<div class="muted" style="font-size:.78rem">سندی ثبت نشده است؛ قبض تاکسی و فاکتورها را با دکمهٔ «افزودن سند» وارد کنید.</div>';
    return;
  }
  const types = (MIS_DEFAULTS && MIS_DEFAULTS.documentTypes) || [];
  const currencies = (MIS_DEFAULTS && MIS_DEFAULTS.currencies) || [];
  const rows = MIS_DOCS.map((doc, i) => `<tr>
    <td><select onchange="misDocField(${i},'kind',this.value)" style="font-size:.78rem">
      ${types.map(t => `<option value="${esc(t.key)}" ${t.key === doc.kind ? 'selected' : ''}>${esc(t.titleFa)}</option>`).join('')}
    </select></td>
    <td><input value="${esc(doc.docNo || '')}" oninput="misDocField(${i},'docNo',this.value)" placeholder="شمارهٔ قبض" style="font-size:.78rem"></td>
    <td><input class="num" value="${esc(doc.docDate || '')}" oninput="misDocField(${i},'docDate',this.value)" placeholder="1405/08/13" style="font-size:.78rem"></td>
    <td><input class="num" value="${esc(String(doc.amount || ''))}" oninput="misDocField(${i},'amount',this.value)" style="font-size:.78rem"></td>
    <td><select onchange="misDocField(${i},'currency',this.value)" style="font-size:.78rem">
      ${currencies.map(c => `<option value="${esc(c.key)}" ${c.key === (doc.currency || 'IRR') ? 'selected' : ''}>${esc(c.titleFa)}</option>`).join('')}
    </select></td>
    <td><label style="display:inline-flex;align-items:center;gap:6px;font-size:.78rem">
      <input type="checkbox" ${doc.approved ? 'checked' : ''} onchange="misDocField(${i},'approved',this.checked)"> تأییدشده
    </label></td>
    <td><button class="btn btn-sm btn-ghost" onclick="misRemoveDoc(${i})">✕</button></td></tr>`);
  box.innerHTML = liveTable(['نوع سند', 'شماره', 'تاریخ', 'مبلغ', 'ارز', 'وضعیت', ''], rows, rows.length);
}

/* ----------------------------- ذخیره ----------------------------- */

/** ذخیرهٔ فرم: رکورد مأموریت + رکورد اسناد پشتیبان. */
async function misSave() {
  if (!MIS_RESULT || !MIS_RESULT.settlement) { toast('ابتدا محاسبه انجام شود', 'error'); return; }
  const destination = misText('misDestination');
  if (!destination) { toast('مقصد مأموریت الزامی است', 'error'); return; }
  const employeeId = misSelect('misEmployee');
  if (!employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  const startDate = misText('misStartDate');
  if (!startDate) { toast('تاریخ شروع مأموریت الزامی است', 'error'); return; }

  const s = MIS_RESULT.settlement;
  // فیلدهای ویژهٔ هر نوع مأموریت فقط برای همان نوع ذخیره می‌شوند: رکورد مأموریت اداری
  // نباید «مشمول هزینه: بیمار» یا ارز و نرخ تسعیر بی‌ربط با خود حمل کند.
  const isMedical = misType() === 'medical';
  const isForeign = misType() === 'foreign';
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
    allowanceTotal: String(Math.round(((s.allowance || MIS_RESULT.allowance || {}).total) || 0)),
    missionType: misType(),
    beneficiary: isMedical ? misSelect('misBeneficiary') : '',
    accommodationType: isMedical ? misSelect('misAccommodation') : '',
    accommodationCost: isMedical ? String(misNum('misAccommodationCost')) : '0',
    companionCount: isMedical ? String(misNum('misCompanions')) : '0',
    medicalMealsAmount: isMedical ? String(Math.round(((s.medical || {}).mealsAmount) || 0)) : '0',
    currency: isForeign ? misSelect('misCurrency') : '',
    fxRate: isForeign ? String(misNum('misFxRate')) : '0',
    fxPerDiem: isForeign ? String(misNum('misFxPerDiem')) : '0',
    fxAmount: isForeign ? String(misNum('misFxAmount')) : '0',
    rialAmount: isForeign ? String(misNum('misRialAmount')) : '0',
    documentsTotal: String(Math.round(s.documentsTotal || 0)),
    fxDocumentsTotal: String(Math.round(s.documentsFxTotal || 0)),
    totalFx: String(Math.round(s.totalFx || 0)),
    rialEquivalent: String(Math.round(s.totalRialEquivalent || 0)),
  };
  // عنوان رکورد از همان عنوان نوع مأموریت ساخته می‌شود (که خودش «مأموریت» دارد).
  const title = misTypeTitleFa(misType()) + ' — ' + destination + ' — ' + startDate +
    ' (' + fa(misNum('misDuration')) + ' روز)';
  const res = await apiFetch('/api/v1/attendance/records', {
    method: 'POST',
    body: {
      kind: 'missions', title: title, employeeId: employeeId, fields: fields,
      noteFa: 'فرم محاسبهٔ مأموریت — جمع ریالی: ' + money(s.totalRial || 0) +
        ' ریال' + (s.totalFx ? ' + ' + money(s.totalFx) + ' ' + (s.foreign || {}).currency : ''),
    },
  });
  if (!res.ok) { toast(res.message, 'error'); return; }
  const id = (res.data && res.data.id) || '';

  // اسناد پشتیبان: هر سند یک رکورد «سند و فاکتور مأموریت» می‌شود.
  let savedDocs = 0;
  for (const doc of MIS_DOCS) {
    if (!doc.docNo) continue;
    const docRes = await apiFetch('/api/v1/attendance/records', {
      method: 'POST',
      body: {
        kind: 'missionDocuments', employeeId: employeeId,
        title: misDocTypeTitleFa(doc.kind) + ' — ' + doc.docNo + ' — ' + destination,
        fields: {
          mission: id, employeeId: employeeId, docType: doc.kind, docNo: doc.docNo,
          docDate: doc.docDate || startDate, amount: String(doc.amount || 0),
          currency: doc.currency || 'IRR', payer: doc.payer || 'company',
        },
        noteFa: 'سند پشتیبان مأموریت ' + id,
      },
    });
    if (docRes.ok) savedDocs++;
  }

  toast('مأموریت ذخیره شد: ' + id + (savedDocs ? ' با ' + fa(savedDocs) + ' سند پشتیبان' : ''));
  const box = document.getElementById('misSaved');
  if (box) {
    box.innerHTML = `<div class="card card-pad" style="margin-top:10px;font-size:.82rem;line-height:2">
      رکورد مأموریت <b class="ltr">${esc(id)}</b> ساخته شد: جمع ریالی
      <b class="num">${fa(money(s.totalRial || 0))}</b> ریال${s.totalFx ? ` و <b class="num">${fa(money(s.totalFx))}</b> ${esc((s.foreign || {}).currency || '')}` : ''}
      ${savedDocs ? ` · ${fa(savedDocs)} سند پشتیبان ثبت شد (پس از تأیید مالی در جمع می‌آید)` : ''}
      <div class="row-actions" style="margin-top:8px">
        <button class="btn btn-sm btn-primary" onclick="misSubmit('${esc(id)}')">📤 ارسال برای تأیید</button>
        <button class="btn btn-sm btn-ghost" onclick="go('missionCalc')">↻ فرم تازه</button>
      </div></div>`;
  }
}

/** ارسال رکورد مأموریت برای تأیید. */
async function misSubmit(id) {
  const res = await apiFetch('/api/v1/attendance/records/' + encodeURIComponent(id) + '/action',
    { method: 'POST', body: { action: 'submit' } });
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('مأموریت برای تأیید ارسال شد');
}

/** عنوان فارسی یک گزینه از فهرست‌های سرور. */
function misOptionTitle(list, key) {
  const found = (list || []).find(o => o.key === key);
  return found ? found.titleFa : key;
}

/** عنوان نوع مأموریت. */
function misTypeTitleFa(key) { return misOptionTitle((MIS_DEFAULTS || {}).missionTypes, key); }

/** عنوان نوع سند. */
function misDocTypeTitleFa(key) { return misOptionTitle((MIS_DEFAULTS || {}).documentTypes, key); }

/* ----------------------------- ساخت فرم ----------------------------- */

/** یک ردیف ورودی عددی. */
function misField(id, label, value, unit, hint) {
  return `<div class="field"><label>${esc(label)}${unit ? ` <span class="muted">(${esc(unit)})</span>` : ''}</label>
    <input id="${id}" class="num" value="${esc(value)}" placeholder="${esc(hint || '')}" oninput="misCalcSoon()"></div>`;
}

/** فهرست کشویی بله/خیر. */
function misYesNo(id, value) {
  const opts = [{ key: 'no', titleFa: 'خیر' }, { key: 'yes', titleFa: 'بله' }]
    .map(o => `<option value="${o.key}" ${o.key === value ? 'selected' : ''}>${esc(o.titleFa)}</option>`).join('');
  return `<select id="${id}" onchange="misCalcSoon()">${opts}</select>`;
}

/** فهرست کشویی از گزینه‌های سرور. */
function misOptions(id, list, selected, onChange) {
  return `<select id="${id}" ${onChange ? `onchange="${onChange}"` : 'onchange="misCalcSoon()"'}>
    ${(list || []).map(o => `<option value="${esc(o.key)}" ${o.key === selected ? 'selected' : ''}>${esc(o.titleFa)}</option>`).join('')}
  </select>`;
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
    const rates = def.fxRates || {};
    const defaultRate = rates[def.defaultCurrency] || 0;

    const form = `<div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">📝 مشخصات مأموریت</div>
      <div class="form-grid">
        <div class="field"><label>نوع مأموریت <span class="req">*</span></label>
          ${misOptions('misType', def.missionTypes, 'administrative', "misToggleSections();misCalcSoon()")}</div>
        <div class="field"><label>کارمند <span class="req">*</span></label><select id="misEmployee">${empOpts}</select></div>
        <div class="field"><label>مقصد <span class="req">*</span></label><input id="misDestination" placeholder="شهر یا محل مأموریت"></div>
        <div class="field"><label>تاریخ شروع مأموریت (شمسی) <span class="req">*</span></label>
          <input id="misStartDate" class="num" value="${esc(today)}" placeholder="1405/08/15" oninput="misCalcSoon()"></div>
        <div class="field"><label>ساعت شروع مأموریت</label>
          <input id="misStartTime" class="num" value="07:30" oninput="misCalcSoon()"></div>
        ${misField('misDuration', 'مدت مأموریت', '1', 'روز', 'مثلاً ۲ یا ۰٫۵')}
        ${misField('misFactor', 'ضریب عوامل مأموریت', String(def.missionFactor || 1), 'ضریب')}
        ${misField('misPerDiem', 'فوقالعادهٔ روزانهٔ مأموریت', String(def.perDiem || 0), 'ریال')}
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
    </div>

    <div class="card card-pad" style="margin-bottom:14px" id="misMedicalSection">
      <div class="card-title">🩺 اعزام پزشکی — هزینه بر پایهٔ بیمار یا همراه بیمار</div>
      <div class="form-grid">
        <div class="field"><label>مشمول هزینه</label>${misOptions('misBeneficiary', def.beneficiaries, 'patient')}</div>
        <div class="field"><label>نوع اسکان</label>${misOptions('misAccommodation', def.accommodations, 'hospital')}</div>
        ${misField('misAccommodationCost', 'هزینهٔ اسکان', '0', 'ریال', 'جای سهم خواب')}
        ${misField('misCompanions', 'شمار همراهان', '0', 'نفر')}
      </div>
      <div class="muted" style="font-size:.74rem;margin-top:6px">
        بیمار: همهٔ اجزای فوقالعاده، اسکان و ایاب‌وذهاب — همراه بیمار: اسکان و ایاب‌وذهاب
        (سهم وعده‌ها تابع پارامتر شرکت است). هزینهٔ اسکان جای سهم خواب پرداخت می‌شود.
      </div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px" id="misForeignSection">
      <div class="card-title">🌍 مأموریت خارج از کشور — پرداخت ریالی و ارزی</div>
      <div class="form-grid">
        <div class="field"><label>ارز پرداخت</label>${misOptions('misCurrency', def.currencies, def.defaultCurrency || 'EUR', "misApplyFxRate();misCalcSoon()")}</div>
        ${misField('misFxRate', 'نرخ تسعیر (ریال برای هر واحد ارز)', String(defaultRate), 'ریال')}
        ${misField('misFxPerDiem', 'فوقالعادهٔ روزانهٔ ارزی', String(def.fxPerDiemDefault || 0), 'ارز در روز')}
        ${misField('misFxAmount', 'سایر مبالغ ارزی (بلیت/اقامت)', '0', 'ارز')}
        ${misField('misRialAmount', 'مبالغ ریالی مأموریت', '0', 'ریال')}
      </div>
      <div class="muted" style="font-size:.74rem;margin-top:6px">${esc(def.noteForeignCurrency || '')}</div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">🧾 قبض تاکسی و فاکتورهای مورد تأیید</div>
      <div class="muted" style="font-size:.74rem;margin-bottom:8px">
        هر قبض یا فاکتور یک سند است؛ فقط سند «تأییدشده» در جمع پرداخت می‌آید و بقیه در فهرست می‌ماند.
      </div>
      <div id="misDocs"></div>
      <div class="row-actions" style="margin-top:8px">
        <button class="btn btn-sm btn-ghost" onclick="misAddDoc()">➕ افزودن سند (قبض تاکسی / فاکتور)</button>
      </div>
    </div>

    <div class="card card-pad" style="margin-bottom:14px">
      <div class="card-title">💵 تسویهٔ مأموریت</div>
      <div id="misResult"><div class="muted" style="font-size:.78rem">در حال محاسبه...</div></div>
      <div class="row-actions" style="margin-top:10px">
        <button class="btn btn-primary" onclick="misCalc()">🧮 محاسبهٔ مجدد</button>
        <button class="btn btn-ghost" onclick="misSave()">💾 ذخیرهٔ مأموریت و اسناد</button>
      </div>
      <div id="misSaved"></div>
    </div>`;

    const paramRows = (def.params || []).map(p => `<tr>
      <td>${esc(p.titleFa)}</td>
      <td class="ltr" style="font-size:.72rem">${esc(p.key)}</td>
      <td class="num">${fa(p.value)}</td>
      <td>${esc(p.unitFa)}</td></tr>`);

    const shell = form +
      (paramRows.length ? `<div class="card card-pad" style="margin-bottom:14px">
        <div class="card-title">پارامترهای مأموریت شرکت فعال (ریالی، ارزی و اعزام پزشکی)</div>
        ${liveTable(['پارامتر', 'کلید', 'مقدار', 'واحد'], paramRows, paramRows.length)}
      </div>` : '') +
      statNote(def.noteFa);

    setTimeout(() => { misRenderDocs(); misToggleSections(); misCalc(); }, 80);
    return statNote('این فرم رکورد «مأموریت» و برای هر سند پشتیبان یک رکورد «سند و فاکتور مأموریت» می‌سازد؛ سند پس از تأیید مالی در جمع پرداخت می‌آید.') + shell;
  });
  return liveShell('فرم محاسبهٔ مأموریت',
    'تسویهٔ مأموریت اداری، اعزام پزشکی و مأموریت خارج از کشور با اسناد و فاکتورهای تأییدشده',
    `<button class="btn btn-ghost" onclick="MIS_DEFAULTS=null;MIS_DOCS=[];liveReload()">↻ تازه‌سازی پارامترها</button>`);
}

/** نمایش/پنهان‌کردن بخش‌های ویژه بر پایهٔ نوع مأموریت. */
function misToggleSections() {
  const kind = misType();
  const medical = document.getElementById('misMedicalSection');
  const foreign = document.getElementById('misForeignSection');
  // بخش اعزام پزشکی فقط برای همان نوع مأموریت؛ در مأموریت اداری فرم را شلوغ نمی‌کند.
  if (medical) medical.style.display = (kind === 'medical') ? '' : 'none';
  if (foreign) foreign.style.display = (kind === 'foreign') ? '' : 'none';
}

/** گذاشتن نرخ تسعیر ارز انتخاب‌شده از پارامترهای شرکت. */
function misApplyFxRate() {
  const def = MIS_DEFAULTS || {};
  const rates = def.fxRates || {};
  const code = misSelect('misCurrency');
  const el = document.getElementById('misFxRate');
  if (el && rates[code]) el.value = String(rates[code]);
}

/* ثبت در مسیریاب زنده (پیش از قالب نمایشی بررسی می‌شود). */
LIVE_PAGES.missionCalc = pageMissionCalcLive;
