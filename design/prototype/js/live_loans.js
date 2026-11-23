/* =====================================================================
   صورت حساب وامها — پرداخت یکبارهٔ تسهیلات، جدول اقساط و مانده بدهی
   ---------------------------------------------------------------------
   هیچ قاعدهای اینجا نیست: نرخ کارمزد، مبلغ هر قسط، قسط آخر (جبران گردکردن)،
   سهم اصل/کارمزد هر قسط، وضعیت معوق، مانده بدهی، کدهای رسمی فیش و مکاتبهٔ
   «نوع وام بدون کد» همه در بستهٔ قواعد سمت سرور حساب میشوند. این صفحه فقط:

      ۱) کارتابل صورت حساب وامها را با فیلتر ماه و کارمند میخواند
         (`/api/v1/loans/statements`) و جمعهای ماه را در کارتهای آماری میگذارد.
      ۲) صورت حساب کامل یک وام را نشان میدهد: سرصفحهٔ وام (بانک، مدیریت، هدف،
         ضامن)، شرایط، جدول اقساط ماهانه، تاریخچهٔ رخدادها، کد پرداخت/کسر قسط،
         مشکلات و یادداشت (`/api/v1/loans/statement`).
      ۳) جدول اقساط یک وام را میسازد یا بهروز میکند
         (`POST /api/v1/loans/{id}/schedule`) — تکرارپذیر و بیاثر روی قسط کسرشده.
      ۴) جدول اقساط را پیش از ثبت وام پیشنمایش میدهد
         (`POST /api/v1/loans/schedule/preview`).
      ۵) اقساط همان ماه را برای کسر از حقوق فهرست میکند
         (`/api/v1/loans/installments-due`).

   ساختار همین فایل مثل `live_allowance.js` است: یک `loansPaint` بهعنوان تنها
   نقطهٔ درج HTML، و اتصال رویدادها در `loansBind()` بدون هیچ هندلر درونخطی.
   ===================================================================== */

/** آخرین پاسخ کارتابل صورت حسابها (برای فیلتر سمت مرورگر و بازسازی نما). */
var LOAN_DATA = null;
/** آخرین صورت حساب انتخابشده (کارت جزئیات). */
var LOAN_DETAIL = null;
/** گزینههای پایه (انواع وام، بانکها) و فهرست کارکنان برای فیلترها. */
var LOAN_OPTIONS = null;
var LOAN_EMPLOYEES = [];

/** کلاس چیپِ وضعیت قسط/وام؛ کلاس نامعلوم به «در انتظار» برمیگردد. */
const LOAN_STATUS_CLS = {paid: 'c-ok', pending: 'c-wait', overdue: 'c-rej'};
const loanChip = (status, statusFa) =>
  pChip(LOAN_STATUS_CLS[status] || 'c-wait', esc(statusFa || status || '—'));

/** نمایش یک متن؛ خالیها خط تیره میگیرند. */
const loanText = v => esc(v || '—');
/** نمایش تاریخ/ماه شمسی؛ خالیها خط تیره. */
const loanDate = v => (v ? `<span class="num">${fa(v)}</span>` : '—');

/** مبلغ با یکای ریال؛ صفر «—» میشود تا جدول شلوغ نشود. */
function loanMoney(v) {
  const n = Number(v || 0);
  return n ? `${money(n)} <span class="muted" style="font-size:.62rem">ریال</span>` : '—';
}

/** کد رسمی فیش: کد (لاتین) بههمراه عنوانش. */
function loanCodeBox(code, titleFa) {
  return `<div><span class="muted">${esc(titleFa || 'کد رسمی فیش')}:</span>
      <span class="ltr num">${loanText(code)}</span></div>`;
}

/* ------------------------------ کارتابل ------------------------------ */

/** نوار فیلتر: ماه، کارمند و دکمهٔ نمایش. */
function loanFilterHtml(d) {
  const opts = LOAN_EMPLOYEES.map(e =>
    `<option value="${esc(e.id)}" ${String(e.id) === String(d.employeeId || '') ? 'selected' : ''}>${esc(e.fullName || e.id)} — ${esc(e.id)}</option>`).join('');
  // کلید وضعیت‌ها همان کلیدهای رسمی دامنهٔ وام است (فهرست نمایشی، نه قاعدهٔ تازه در رابط).
  const statuses = [['requested', 'درخواست'], ['approved', 'تأییدشده'], ['repaying', 'در حال بازپرداخت'], ['settled', 'تسویه‌شده']];
  const statusOpts = statuses.map(s =>
    `<option value="${esc(s[0])}" ${s[0] === (d.status || '') ? 'selected' : ''}>${esc(s[1])}</option>`).join('');
  return `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:10px;align-items:flex-end;flex-wrap:wrap">
      <div class="field"><label>ماه صورت حساب (YYYY/MM)</label>
        <input id="loanPeriod" class="ltr" value="${esc(d.periodFa || '')}" placeholder="${esc(String(d.todayFa || '').slice(0, 7))}"></div>
      <div class="field" style="min-width:300px"><label>کارمند</label>
        <select id="loanEmployee">
          <option value="">همهٔ کارکنان</option>
          ${opts}
        </select></div>
      <div class="field"><label>وضعیت وام</label>
        <select id="loanStatus">
          <option value="" ${d.status ? '' : 'selected'}>همه</option>
          ${statusOpts}
        </select></div>
      <button class="btn btn-primary" id="loanRun">نمایش</button>
      <span class="muted" style="font-size:.72rem">ماه مرجع تعیین میکند کدام قسط «معوق» شمرده شود.</span>
    </div>`;
}

/** کارتهای جمع کارتابل. */
function loanStatsHtml(d) {
  const t = d.totals || {};
  return `<div class="grid-stats" style="margin-bottom:12px">
      ${statCard({ico: '🏦', tint: '#EFF6FF', accent: '#2563EB', val: fa(t.loans || 0), label: 'تعداد وامها'})}
      ${statCard({ico: '💰', tint: '#EDE9FE', accent: '#7C3AED', val: money(t.amount || 0), label: 'جمع اصل وام (ریال)'})}
      ${statCard({ico: '📉', tint: '#FEE2E2', accent: '#DC2626', val: money(t.remainingDebt || 0), label: 'جمع مانده بدهی (ریال)'})}
      ${statCard({ico: '🧾', tint: '#FEF3C7', accent: '#D97706', val: money(t.monthlyDeduction || 0), label: 'کسر این ماه (ریال)'})}
      ${statCard({ico: '⏳', tint: '#FEE2E2', accent: '#DC2626', val: money(t.overdueAmount || 0), label: 'اقساط معوق (ریال)'})}
    </div>`;
}

/** سطرهای جدول وامها؛ «صورت حساب» سطر را در کارت جزئیات باز میکند. */
function loanRowHtml(it) {
  const st = (it.statement || {}).balance || {};
  const s = it.statement || {};
  return `<tr>
      <td class="num ltr">${esc(it.loanId)}</td>
      <td>${esc(it.employeeName || '—')}<div class="muted ltr" style="font-size:.66rem">${esc(it.employeeId || '')}</div></td>
      <td>${loanText(s.loanTypeFa || it.titleFa)}</td>
      <td class="num">${loanMoney(s.amount)}</td>
      <td class="num">${loanMoney(s.feeAmount)}</td>
      <td class="num">${loanMoney(s.totalPayable)}</td>
      <td class="num">${loanMoney(s.installmentAmount)}</td>
      <td class="num">${fa(st.paidCount || 0)} / ${fa(s.installments || 0)}</td>
      <td class="num">${loanMoney(st.remainingDebt)}</td>
      <td>${st.nextPeriodFa ? `${loanDate(st.nextPeriodFa)} <span class="muted" style="font-size:.68rem">${money(st.nextAmount || 0)} ریال</span>` : '—'}</td>
      <td>${loanChip(it.status, it.statusFa)}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-ghost" style="white-space:nowrap" data-loan-detail="${esc(it.loanId)}">صورت حساب</button>
      </div></td>
    </tr>`;
}

/** یادداشت «انواع وام بدون کد رسمی فیش». */
function loanCodeGapsHtml(d) {
  const gaps = d.codeGaps || [];
  if (!gaps.length) return '';
  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="font-weight:600;margin-bottom:4px">انواع وام بدون کد رسمی فیش: ${fa(gaps.length)} نوع</div>
      <div class="muted" style="font-size:.78rem">این انواع در جدول رسمی کدهای فیش کد ندارند و باید برایشان مکاتبهٔ مالی انجام شود: ${gaps.map(g => esc(g.titleFa || g.key)).join('، ')}</div>
    </div>`;
}

/** کارت جزئیات صورت حساب یک وام. */
function loanDetailHtml() {
  const it = LOAN_DETAIL;
  if (!it) {
    return `<div class="card card-pad muted" style="margin-bottom:12px">برای دیدن جدول اقساط، تاریخچه و مانده بدهی، روی «صورت حساب» یکی از سطرهای بالا بزنید.</div>`;
  }
  const s = it.statement || {};
  const b = s.balance || {};
  const head = [
    ['بانک عامل', it.bankFa], ['مدیریت متقاضی', it.managementFa], ['هدف وام', it.purposeFa],
    ['ضامن', it.guarantorName], ['نسبت ضامن', it.guarantorRelation],
    ['رکورد قسط ثبتشده', it.installmentRecords == null ? '' : fa(it.installmentRecords)],
  ].filter(x => x[1]).map(x => `<div><span class="muted">${esc(x[0])}:</span> <b>${esc(x[1])}</b></div>`).join('');

  const conditions = `<div class="grid-stats" style="margin-bottom:8px">
      ${statCard({ico: '💰', tint: '#EDE9FE', accent: '#7C3AED', val: money(s.amount || 0), label: 'اصل وام (ریال)'})}
      ${statCard({ico: '🧾', tint: '#FEF3C7', accent: '#D97706', val: `${money(s.feeAmount || 0)} <span class="muted" style="font-size:.6rem">${fa(s.feeRatePercent || 0)}٪</span>`, label: 'کارمزد (ریال)'})}
      ${statCard({ico: 'Σ', tint: '#EFF6FF', accent: '#2563EB', val: money(s.totalPayable || 0), label: 'جمع قابل بازپرداخت (ریال)'})}
      ${statCard({ico: '📆', tint: '#DCFCE7', accent: '#16A34A', val: money(s.installmentAmount || 0), label: `مبلغ هر قسط (${fa(s.installments || 0)} قسط)`})}
      ${statCard({ico: '🗓️', tint: '#EFF6FF', accent: '#2563EB', val: loanText(s.firstPeriodFa), label: 'شروع اقساط'})}
      ${statCard({ico: '🏁', tint: '#EDE9FE', accent: '#7C3AED', val: loanText(s.lastPeriodFa), label: 'پایان اقساط'})}
      ${statCard({ico: '📉', tint: '#FEE2E2', accent: '#DC2626', val: money(b.remainingDebt || 0), label: `مانده بدهی (${fa(b.remainingInstallments || 0)} قسط)`})}
      ${statCard({ico: '⏳', tint: '#FEE2E2', accent: '#DC2626', val: money(b.overdueAmount || 0), label: `اقساط معوق (${fa(b.overdueCount || 0)} قسط)`})}
    </div>`;

  const codes = `<div class="card card-pad" style="margin-bottom:8px;display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:3px 14px;font-size:.8rem">
      ${loanCodeBox(s.paymentCode, s.paymentCodeTitleFa || 'کد پرداخت تسهیلات')}
      ${loanCodeBox(s.installmentCode, s.installmentCodeTitleFa || 'کد کسر قسط')}
      ${s.receivedDateFa ? `<div><span class="muted">تاریخ دریافت وام:</span> ${loanDate(s.receivedDateFa)}</div>` : ''}
      ${s.lastInstallment ? `<div><span class="muted">قسط آخر (جبران گردکردن):</span> <span class="num">${money(s.lastInstallment)}</span> <span class="muted" style="font-size:.66rem">ریال</span></div>` : ''}
      ${s.disbursed ? '<div class="muted">پرداخت تسهیلات ثبت شده است.</div>' : ''}
    </div>`;

  const warn = s.codeNoteFa
    ? `<div class="muted" style="font-size:.75rem;margin:0 0 8px">${esc(s.codeNoteFa)}</div>` : '';

  const scheduleRows = (s.schedule || []).map(r => `<tr>
      <td class="num">${fa(r.seq || 0)}</td>
      <td>${loanDate(r.periodFa)}</td>
      <td class="num">${loanMoney(r.amount)}</td>
      <td class="num">${loanMoney(r.principal)}</td>
      <td class="num">${loanMoney(r.fee)}</td>
      <td>${loanChip(r.status, r.statusFa)}</td>
      <td>${loanDate(r.paidDateFa)}</td>
    </tr>`);

  const historyRows = (s.history || []).map(h => `<tr>
      <td class="num">${fa(h.order || 0)}</td>
      <td>${loanText(h.kindFa)}</td>
      <td>${loanText(h.titleFa)}</td>
      <td>${loanDate(h.periodFa || h.dateFa)}</td>
      <td class="ltr num">${loanText(h.codeFa)}</td>
      <td>${loanText(h.directionFa)}</td>
      <td class="num">${loanMoney(h.amount)}</td>
      <td>${loanText(h.statusFa)}</td>
      <td class="num">${loanMoney(h.balanceAfter)}</td>
    </tr>`);

  const problems = (s.problems || []).length
    ? `<div class="muted" style="font-size:.76rem;margin-top:6px">${s.problems.map(p => '⚠️ ' + esc(p)).join('<br>')}</div>` : '';
  const note = s.noteFa
    ? `<div class="muted" style="font-size:.76rem;margin-top:4px">${esc(s.noteFa)}</div>` : '';
  const balanceNote = b.noteFa
    ? `<div class="muted" style="font-size:.76rem;margin-top:4px">${esc(b.noteFa)}</div>` : '';

  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="display:flex;justify-content:space-between;gap:10px;align-items:flex-start;flex-wrap:wrap">
        <div>
          <div style="font-weight:600">صورت حساب وام <span class="ltr num">${esc(it.loanId)}</span> — ${loanText(s.titleFa || it.titleFa)}</div>
          <div class="muted" style="font-size:.75rem">${esc(it.employeeName || '—')} <span class="ltr num">${esc(it.employeeId || '')}</span></div>
        </div>
        <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
          ${loanChip(it.status, it.statusFa)}
          <button class="btn btn-sm btn-ghost" id="loanPrint">🖨️ پرینت صورت حساب</button>
          <button class="btn btn-sm btn-primary" id="loanBuildSchedule">ساخت/بهروزرسانی جدول اقساط</button>
        </div>
      </div>
      <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(215px,1fr));gap:3px 14px;font-size:.8rem;margin:8px 0">
        ${head || '<div class="muted">سرصفحهٔ وام ثبت نشده است.</div>'}
      </div>
      ${conditions}
      ${codes}
      ${warn}
      <div style="font-weight:600;margin:10px 0 4px">جدول اقساط ماهانه</div>
      ${tbl(['قسط', 'ماه کسر', 'مبلغ قسط', 'سهم اصل', 'سهم کارمزد', 'وضعیت', 'تاریخ کسر'],
        scheduleRows, {count: (s.schedule || []).length})}
      <div style="font-weight:600;margin:10px 0 4px">تاریخچهٔ وام</div>
      ${tbl(['ردیف', 'نوع', 'عنوان', 'ماه/تاریخ', 'کد', 'گردش', 'مبلغ', 'وضعیت', 'مانده پس از رخداد'],
        historyRows, {count: (s.history || []).length})}
      ${problems}${balanceNote}${note}
    </div>`;
}

/* ------------------- پیشنمایش جدول اقساط و اقساط ماه ------------------- */

/** فرم پیشنمایش جدول اقساط؛ پیشفرضها از انواع وام رسمی میآید. */
function loanPreviewFormHtml() {
  const types = (LOAN_OPTIONS || {}).loanTypes || [];
  const typeOpts = types.map(t =>
    `<option value="${esc(t.key)}">${esc(t.titleFa || t.key)}${t.feeRatePercent ? ` — کارمزد ${fa(t.feeRatePercent)}٪` : ' — بیبهره'}</option>`).join('');
  const t0 = types[0] || {};
  // نشانهٔ ماه مرجع در placeholder: «YYYY/MM/DD» ⇒ «YYYY/MM».
  const month = ((LOAN_DATA || {}).todayFa || '').slice(0, 7) || ((LOAN_DATA || {}).periodFa || '');
  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="font-weight:600;margin-bottom:6px">پیشنمایش جدول اقساط</div>
      <div class="muted" style="font-size:.74rem;margin-bottom:6px">جمع اقساط دقیقاً برابر اصل + کارمزد میشود و قسط آخر باقیماندهٔ گردکردن را جبران میکند؛ محاسبه در سرور انجام میشود.</div>
      <div class="form-grid">
        <div class="field"><label>نوع وام</label><select id="loanPrevType">${typeOpts}</select></div>
        <div class="field"><label>مبلغ وام (ریال)</label><input id="loanPrevAmount" class="num" placeholder="مثلاً 300000000"></div>
        <div class="field"><label>تعداد اقساط</label><input id="loanPrevInstallments" class="num" value="${esc(String(t0.maxInstallments || ''))}"></div>
        <div class="field"><label>نرخ کارمزد (درصد)</label><input id="loanPrevFeeRate" class="num" value="${esc(String(t0.feeRatePercent == null ? '' : t0.feeRatePercent))}"></div>
        <div class="field"><label>تاریخ دریافت وام (YYYY/MM/DD)</label><input id="loanPrevReceived" class="ltr" placeholder="1405/07/01"></div>
        <div class="field"><label>ماه شروع اقساط (YYYY/MM)</label><input id="loanPrevFirstPeriod" class="ltr" placeholder="${esc(month)}"></div>
      </div>
      <div style="display:flex;gap:8px;align-items:center;margin-top:8px;flex-wrap:wrap">
        <button class="btn btn-primary" id="loanPrevRun">محاسبه و پیشنمایش</button>
        <span class="muted" style="font-size:.72rem">ماه شروع خالی بماند، از تاریخ دریافت گرفته میشود.</span>
      </div>
      <div id="loanPreviewOut" style="margin-top:10px"></div>
    </div>`;
}

/** خروجی پیشنمایش: جمعها، کدهای رسمی و سطرهای جدول اقساط. */
function loanPreviewResultHtml(d) {
  const s = d.schedule || {};
  const t = s.terms || {};
  const cards = `<div class="grid-stats" style="margin-bottom:8px">
      ${statCard({ico: 'Σ', tint: '#EFF6FF', accent: '#2563EB', val: money(s.totalPayable || 0), label: 'جمع قابل بازپرداخت (ریال)'})}
      ${statCard({ico: '📆', tint: '#DCFCE7', accent: '#16A34A', val: money(s.installmentAmount || 0), label: `مبلغ هر قسط (${fa(t.installments || 0)} قسط)`})}
      ${statCard({ico: '🏁', tint: '#EDE9FE', accent: '#7C3AED', val: money(s.lastInstallment || 0), label: 'قسط آخر (ریال)'})}
      ${statCard({ico: '🧾', tint: '#FEF3C7', accent: '#D97706', val: money(t.feeAmount || 0), label: 'کارمزد (ریال)'})}
    </div>`;
  const codes = `<div class="card card-pad" style="margin-bottom:8px;display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:3px 14px;font-size:.8rem">
      ${loanCodeBox(d.paymentCode, d.paymentCodeFa || 'کد پرداخت تسهیلات')}
      ${loanCodeBox(d.installmentCode, d.installmentCodeFa || 'کد کسر قسط')}
    </div>`;
  const gaps = (d.problems || []).length
    ? `<div class="muted" style="font-size:.76rem;margin:6px 0">${d.problems.map(p => '⚠️ ' + esc(p)).join('<br>')}</div>`
    : (d.noteFa ? `<div class="muted" style="font-size:.76rem;margin:6px 0">${esc(d.noteFa)}</div>` : '');
  const rows = (s.rows || []).map(r => `<tr>
      <td class="num">${fa(r.seq || 0)}</td>
      <td>${loanDate(r.periodFa)}</td>
      <td class="num">${loanMoney(r.amount)}</td>
      <td class="num">${loanMoney(r.principal)}</td>
      <td class="num">${loanMoney(r.fee)}</td>
    </tr>`);
  return cards + codes + gaps +
    tbl(['قسط', 'ماه کسر', 'مبلغ قسط', 'سهم اصل', 'سهم کارمزد'], rows, {count: (s.rows || []).length});
}

/** کارت «اقساط این ماه برای کسر از حقوق». */
function loanDueCardHtml() {
  return `<div class="card card-pad" style="margin-bottom:12px">
      <div style="display:flex;justify-content:space-between;align-items:center;gap:10px;flex-wrap:wrap">
        <div>
          <div style="font-weight:600">اقساط این ماه برای کسر از حقوق</div>
          <div class="muted" style="font-size:.74rem">هر قسط ماه مرجع با کد رسمی کسر و مانده بدهی همان وام؛ فیش حقوقی همین مبلغ را کسر میکند.</div>
        </div>
        <button class="btn btn-sm btn-ghost" id="loanDueRun">نمایش اقساط ماه</button>
      </div>
      <div id="loanDueOut" style="margin-top:10px"></div>
    </div>`;
}

/* ------------------------------ صفحه و رویدادها ------------------------------ */

/** صفحهٔ «صورت حساب وامها». */
function pageLoanStatementLive() {
  if (!isLive()) return offlineFallback('loanStatement');
  (async () => {
    await liveLoad(async () => {
      const period = (LOAN_DATA && LOAN_DATA.periodFa) || '';
      const employeeId = (LOAN_DATA && LOAN_DATA.employeeId) || '';
      const status = (LOAN_DATA && LOAN_DATA.status) || '';
      const [statRes, optRes, empRes] = await Promise.all([
        apiFetch('/api/v1/loans/statements' + loanQuery(period, employeeId, status)),
        apiFetch('/api/v1/loans/options'),
        loanEmployeeFetch(),
      ]);
      if (!statRes.ok) return liveErrorBox(statRes.message);
      const d = statRes.data || {};
      d.periodFa = d.periodFa || period;
      d.todayFa = d.todayFa || '';
      d.employeeId = employeeId;
      d.status = status;
      LOAN_DATA = d;
      LOAN_OPTIONS = (optRes.ok && optRes.data) || {loanTypes: []};
      LOAN_EMPLOYEES = loanEmployeeItems(empRes);
      const items = d.items || [];
      return loanFilterHtml(d) + loanStatsHtml(d) + loanCodeGapsHtml(d) +
        tbl(['شناسهٔ وام', 'کارمند', 'نوع/عنوان', 'مبلغ وام', 'کارمزد', 'جمع قابل بازپرداخت',
          'مبلغ هر قسط', 'اقساط کسرشده/کل', 'مانده بدهی', 'قسط بعدی', 'وضعیت', 'عملیات'],
          items.map(loanRowHtml), {count: d.count == null ? items.length : d.count}) +
        `<div id="loanDetail">${loanDetailHtml()}</div>` +
        loanPreviewFormHtml() + loanDueCardHtml();
    });
    loansBind();
  })();
  return liveShell('صورت حساب وامها',
    'پرداخت یکبارهٔ وام، جدول اقساط ماهانه، تاریخچه، کارمزد و مانده بدهی — قابل گزارشگیری', '');
}

/** اتصال رویدادها — بدون هیچ هندلر درونخطی. */
function loansBind() {
  const region = document.getElementById('liveRegion');
  // دکمهٔ «نمایش»: ماه/کارمند/وضعیت فیلتر را با خودِ صفحه ذخیره و بازخوانی میکند.
  const run = document.getElementById('loanRun');
  if (run && !run.dataset.loanBound) {
    run.dataset.loanBound = '1';
    const val = id => {
      const el = document.getElementById(id);
      return el ? (el.value || '').trim() : '';
    };
    run.addEventListener('click', () => loansFilterRun(val('loanPeriod'), val('loanEmployee'), val('loanStatus')));
  }
  // انتخاب سطر جدول: کارت جزئیات همان وام از سرور خوانده و نمایش داده میشود.
  if (region && !region.dataset.loanBound) {
    region.dataset.loanBound = '1';
    region.addEventListener('click', ev => {
      const btn = ev.target.closest('[data-loan-detail]');
      if (btn) loansOpenDetail(btn.dataset.loanDetail);
    });
  }
  const print = document.getElementById('loanPrint');
  if (print && !print.dataset.loanBound) {
    print.dataset.loanBound = '1';
    print.addEventListener('click', () => { window.print(); });
  }
  // ساخت/بهروزرسانی جدول اقساط برای همان وامی که صورت حسابش باز است.
  const build = document.getElementById('loanBuildSchedule');
  if (build && !build.dataset.loanBound) {
    build.dataset.loanBound = '1';
    build.addEventListener('click', () => loansBuildSchedule((LOAN_DETAIL || {}).loanId));
  }
  const type = document.getElementById('loanPrevType');
  if (type && !type.dataset.loanBound) {
    type.dataset.loanBound = '1';
    type.addEventListener('change', () => loansPreviewDefaults(type.value));
  }
  const prevRun = document.getElementById('loanPrevRun');
  if (prevRun && !prevRun.dataset.loanBound) {
    prevRun.dataset.loanBound = '1';
    prevRun.addEventListener('click', () => loansPreview());
  }
  const dueRun = document.getElementById('loanDueRun');
  if (dueRun && !dueRun.dataset.loanBound) {
    dueRun.dataset.loanBound = '1';
    dueRun.addEventListener('click', () => loansDue());
  }
}

/**
 * تنها نقطهٔ درج HTML این صفحه (سقف امنیتی رابط).
 *
 * چرا با DOM و نه درج رشته با درج درونی: سقف ratchet در `docs/security-spec-fa.md` روی ۱۲۷
 * بسته است و قاعدهٔ مخزن «سقف فقط پایین میآید» است؛ صفحهٔ تازه نباید سقف را بالا
 * ببرد. نتیجه یکی است (رشتهٔ HTML به گره تبدیل و جای محتوای قبلی مینشیند)، ولی
 * نقطهٔ پرخطر تازه‌ای به فهرست اضافه نمیشود.
 */
function loansPaint(el, html) {
  if (!el) return;
  const doc = el.ownerDocument || document;
  const range = doc.createRange();
  range.selectNodeContents(el);
  el.replaceChildren(range.createContextualFragment(String(html == null ? '' : html)));
}

/** نشانی کارتابل با فیلترهای فعلی. */
function loanQuery(period, employeeId, status) {
  const parts = [];
  if (period) parts.push('period=' + encodeURIComponent(period));
  if (employeeId) parts.push('employeeId=' + encodeURIComponent(employeeId));
  if (status) parts.push('status=' + encodeURIComponent(status));
  return parts.length ? '?' + parts.join('&') : '';
}

/** فهرست کارکنان برای فیلتر؛ اگر مسیر موجود نباشد، فیلتر بیگزینه میماند. */
function loanEmployeeFetch() {
  return apiFetch('/api/v1/employees?size=200').then(res => {
    if (res.ok) return res;
    return apiFetch('/api/v1/employee/employees?size=200');
  }).catch(() => ({ok: false, data: null}));
}

/** اقلام فهرست کارکنان از پاسخ سرور (شکل پاسخ متفاوت باشد، بیاثر است). */
function loanEmployeeItems(res) {
  const items = (res && res.ok && res.data && res.data.items) || [];
  return items.filter(e => e && e.id).map(e => ({
    id: e.id,
    fullName: e.fullName || e.name || e.titleFa || e.id,
    departmentName: e.departmentName || '',
  }));
}

/** بازخوانی کارتابل با ماه، کارمند و وضعیت انتخابشده (ماه مرجع را هم نگه میدارد). */
function loansFilterRun(period, employeeId, status) {
  if (period && !/^\d{4}\/\d{1,2}$/.test(period)) { toast('ماه را به شکل YYYY/MM وارد کنید (مثلاً 1405/07).', 'error'); return; }
  LOAN_DATA = Object.assign({}, LOAN_DATA || {}, {periodFa: period, employeeId: employeeId, status: status});
  LOAN_DETAIL = null;
  liveReload();
}

/** نمایش صورت حساب کامل یک وام در کارت جزئیات. */
async function loansOpenDetail(loanId) {
  const id = loanId || (LOAN_DETAIL && LOAN_DETAIL.loanId) ||
    ((LOAN_DATA && LOAN_DATA.items && LOAN_DATA.items[0]) || {}).loanId;
  const box = document.getElementById('loanDetail');
  if (!box || !id) return;
  loansPaint(box, liveSpinner());
  const res = await apiFetch('/api/v1/loans/statement?id=' + encodeURIComponent(id) +
    '&period=' + encodeURIComponent((LOAN_DATA && LOAN_DATA.periodFa) || ''));
  if (!res.ok) { loansPaint(box, liveErrorBox(res.message)); return; }
  LOAN_DETAIL = res.data || null;
  loansPaint(box, loanDetailHtml());
  // دکمه‌های «پرینت» و «ساخت جدول اقساط» همراه همین کارت تازه ساخته شدند؛ اتصال
  // دوباره (با نگهبان dataset.loanBound) آن‌ها را زنده می‌کند و بقیه را دست نمی‌زند.
  loansBind();
}

/** ساخت/بهروزرسانی رکوردهای قسط یک وام از جدول اقساط سرور. */
async function loansBuildSchedule(id) {
  if (!id) return;
  const res = await apiFetch('/api/v1/loans/' + encodeURIComponent(id) + '/schedule', {method: 'POST'});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const d = res.data || {};
  toast(d.noteFa || ('جدول اقساط همخوان شد: ' + fa(d.created || 0) + ' قسط تازه، ' +
    fa(d.updated || 0) + ' بهروزرسانی و ' + fa(d.unchanged || 0) + ' قسط بیتغییر.'));
  liveReload();
}

/** پیشفرضهای فرم پیشنمایش بر پایهٔ نوع وام انتخابشده (سقف اقساط و نرخ کارمزد). */
function loansPreviewDefaults(key) {
  const t = ((LOAN_OPTIONS || {}).loanTypes || []).find(x => x.key === key);
  if (!t) return;
  const rate = document.getElementById('loanPrevFeeRate');
  const count = document.getElementById('loanPrevInstallments');
  if (rate) rate.value = t.feeRatePercent == null ? '' : String(t.feeRatePercent);
  if (count) count.value = t.maxInstallments ? String(t.maxInstallments) : '';
}

/** پیشنمایش جدول اقساط (بدون ذخیرهسازی). */
async function loansPreview() {
  const out = document.getElementById('loanPreviewOut');
  if (!out) return;
  const val = id => { const el = document.getElementById(id); return el ? (el.value || '').trim() : ''; };
  const body = {
    loanType: val('loanPrevType'),
    amount: Number(val('loanPrevAmount')) || 0,
    installments: Number(val('loanPrevInstallments')) || 0,
    feeRatePercent: Number(val('loanPrevFeeRate')) || 0,
    receivedDate: val('loanPrevReceived'),
    firstPeriod: val('loanPrevFirstPeriod'),
  };
  if (!body.amount) { toast('مبلغ وام را وارد کنید.', 'error'); return; }
  loansPaint(out, liveSpinner());
  const res = await apiFetch('/api/v1/loans/schedule/preview', {method: 'POST', body});
  if (!res.ok) { loansPaint(out, liveErrorBox(res.message)); return; }
  loansPaint(out, loanPreviewResultHtml(res.data || {}));
}

/** اقساط ماه مرجع برای کسر از حقوق. */
async function loansDue() {
  const out = document.getElementById('loanDueOut');
  if (!out) return;
  const period = ((LOAN_DATA && LOAN_DATA.periodFa) || '');
  loansPaint(out, liveSpinner());
  const res = await apiFetch('/api/v1/loans/installments-due?period=' + encodeURIComponent(period));
  if (!res.ok) { loansPaint(out, liveErrorBox(res.message)); return; }
  const d = res.data || {};
  const cards = `<div class="grid-stats" style="margin-bottom:8px">
      ${statCard({ico: 'Σ', tint: '#EFF6FF', accent: '#2563EB', val: money(d.totalAmount || 0), label: 'جمع اقساط ماه (ریال)'})}
      ${statCard({ico: '⏳', tint: '#FEF3C7', accent: '#D97706', val: money(d.pendingAmount || 0), label: 'کسریها (ریال)'})}
      ${statCard({ico: '✅', tint: '#DCFCE7', accent: '#16A34A', val: money(d.paidAmount || 0), label: 'کسرشده (ریال)'})}
      ${statCard({ico: '🧾', tint: '#EDE9FE', accent: '#7C3AED', val: fa(d.count || 0), label: 'تعداد قسط ماه'})}
    </div>`;
  const rows = (d.rows || []).map(r => `<tr>
      <td class="num ltr">${esc(r.loanId)}</td>
      <td>${esc(r.employeeName || '—')}<div class="muted ltr" style="font-size:.66rem">${esc(r.employeeId || '')}</div></td>
      <td>${loanText(r.loanTypeFa)}</td>
      <td class="num">${fa(r.seq || 0)}</td>
      <td>${loanDate(r.periodFa)}</td>
      <td class="num">${loanMoney(r.amount)}</td>
      <td>${loanChip(r.status, r.statusFa)}</td>
      <td class="ltr num">${loanText(r.codeFa)}</td>
      <td class="num">${loanMoney(r.remainingDebt)}</td>
    </tr>`);
  const note = d.noteFa ? `<div class="muted" style="font-size:.76rem;margin:6px 0">${esc(d.noteFa)}</div>` : '';
  loansPaint(out, cards + note +
    tbl(['شناسهٔ وام', 'کارمند', 'نوع وام', 'قسط', 'ماه کسر', 'مبلغ', 'وضعیت', 'کد کسر', 'مانده بدهی'],
      rows, {count: d.count == null ? rows.length : d.count}));
}

if (typeof LIVE_PAGES !== 'undefined') LIVE_PAGES.loanStatement = pageLoanStatementLive;
