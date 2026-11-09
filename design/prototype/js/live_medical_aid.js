/* =====================================================================
   کمک هزینهٔ درمانی و تعرفه‌ها
   ---------------------------------------------------------------------
   این صفحه سه کار می‌کند و هیچ قاعده‌ای در خودش ندارد:

     ۱. فهرست تعرفه‌های مصوب (اروتز و پروتز، تجهیزات، دارو، عینک، دندانپزشکی،
        پاراکلینیک، ویزیت و سایر) را از سرور نشان می‌دهد؛ ثبت و ویرایش تعرفه از
        موتور عمومی رکوردها می‌آید.
     ۲. فرم درخواست کمک هزینه: اقلام از فهرست تعرفه انتخاب می‌شوند، سرور سهم
        سازمان/بیمه/کارمند و سقف‌ها را حساب می‌کند (`/aid-plan`) و همان محاسبه
        با «ثبت درخواست» روی رکورد نوشته می‌شود (`/aid-claims`).
     ۳. کارتابل درخواست‌ها با اقدام‌های گردشکار (ارسال، تأیید کارشناس، تصویب،
        ثبت پرداخت) تا پرداخت بی‌سند ثبت نشود.
   ===================================================================== */

window.MEDAID = window.MEDAID || {employeeId: '', category: '', rows: [], plan: null};

/** پیشوند API ماژول درمان. */
function medAidPrefix() { return '/api/v1/medical'; }

/** فهرست‌های مرجع (یک‌بار در هر نشست). */
async function medAidDefaultsCached() {
  if (window.MEDAID_DEF) return window.MEDAID_DEF;
  const res = await apiFetch(medAidPrefix() + '/aid-defaults');
  window.MEDAID_DEF = res.ok ? (res.data || {}) : {};
  return window.MEDAID_DEF;
}

/** صفحهٔ کمک هزینهٔ درمانی. */
function pageMedicalAidLive() {
  if (!isLive()) return offlineFallback('medicalAid');
  liveLoad(async () => {
    const def = await medAidDefaultsCached();
    const categories = def.categories || [];
    const tariffs = def.tariffs || [];
    const claims = await medAidClaims();
    // انتخابگر کارمند و اقدام‌های گردشکار پیش از ساخت فرم آماده می‌شوند تا فرم با مقدار
    // درست ساخته شود (نه با کادر خالی که بعد پر شود).
    const empSelect = await liveEmployeeSelect('maEmployee', window.MEDAID.employeeId || '');
    // وضعیت موتور عمومی رکوردها: تعرفه‌ها و درخواست‌ها با همان فرم و اقدام‌های عمومی ویرایش
    // می‌شوند؛ پس هر دو نوع باید به کلید صفحهٔ خودشان وصل شوند، وگرنه دکمهٔ افزودن/ویرایش
    // (rkCreate/rkEdit) بی‌اثر می‌ماند، چون وضعیت آن صفحه هرگز ساخته نشده است.
    if (!window.MEDAID_KINDS) {
      const schemaRes = await apiFetch(medAidPrefix() + '/schema');
      if (schemaRes.ok) {
        const kinds = (schemaRes.data || {}).kinds || [];
        window.MEDAID_KINDS = {
          tariffs: kinds.find(k => k.key === 'medicalTariffs') || null,
          claims: kinds.find(k => k.key === 'medicalAidClaims') || null,
          moduleID: (schemaRes.data || {}).module || '',
        };
      }
    }
    if (window.MEDAID_KINDS) {
      const mod = rkFindModule('medicalAid') || {prefix: medAidPrefix(), titleFa: 'کمک هزینهٔ درمانی'};
      rkBindKind('medicalTariffs', mod, window.MEDAID_KINDS.tariffs);
      rkBindKind('medicalAidClaims', mod, window.MEDAID_KINDS.claims);
      rkSetModuleID('medicalTariffs', window.MEDAID_KINDS.moduleID);
      rkSetModuleID('medicalAidClaims', window.MEDAID_KINDS.moduleID);
      window.MEDAID_CLAIM_ACTIONS = window.MEDAID_KINDS.claims;
    }
    // گزینه‌های فیلدهای انتخابی فرم تعرفه/درخواست از خود سرور می‌آیند (نه فهرست ثابت در UI).
    rkSetSelectOpts('category', (categories || []).map(c => ({key: c.key, titleFa: c.titleFa})));
    rkSetSelectOpts('requirePrescription', def.yesNo || []);
    rkSetSelectOpts('prescriptionApproved', def.yesNo || []);
    const approvedCalls = claims.filter(c => c.status === 'approved' || c.status === 'paid').length;
    const paidTotal = claims.filter(c => c.status === 'approved' || c.status === 'paid')
      .reduce((sum, c) => sum + Number((c.fields || {}).payableAmount || 0), 0);

    const stats = `<div class="grid-stats">
      ${statCard({ico: '📗', tint: '#EFF6FF', accent: '#2563EB', val: fa(tariffs.length), label: 'قلم تعرفهٔ جاری'})}
      ${statCard({ico: '🗂️', tint: '#EDE9FE', accent: '#7C3AED', val: fa(categories.length), label: 'دستهٔ کمک هزینه'})}
      ${statCard({ico: '🧾', tint: '#FEF3C7', accent: '#D97706', val: fa(claims.length), label: 'درخواست ثبت‌شده'})}
      ${statCard({ico: '💵', tint: '#DCFCE7', accent: '#16A34A', val: money(paidTotal), label: 'سهم پرداختی سازمان (ریال) — ' + fa(approvedCalls) + ' درخواست'})}
    </div>`;

    return stats + medAidFormHTML(def, empSelect) + '<div id="medAidPlanBox"></div>' +
      medAidClaimsHTML(claims) + medAidTariffsHTML(def) + medAidInstructionHTML(def) +
      statNote(def.noteFa || '') + medAidPrime(def);
  });
  return liveShell('کمک هزینهٔ درمانی و تعرفه‌ها',
    'اروتز و پروتز، تجهیزات و لوازم پزشکی، دارو، عینک، دندانپزشکی و سایر هزینه‌ها بر پایهٔ تعرفهٔ مصوب — با سهم سازمان و کنترل سقف',
    `<button class="btn btn-ghost" onclick="window.MEDAID_DEF=null;liveReload()">↻ تازه‌سازی تعرفه‌ها</button>`);
}

/* ------------------------------ فرم درخواست ------------------------------ */

/** فرم ثبت درخواست کمک هزینه. */
function medAidFormHTML(def, empSelect) {
  const categories = (def.categories || []).map(c =>
    `<option value="${esc(c.key)}" ${c.key === window.MEDAID.category ? 'selected' : ''}>${esc(c.titleFa)}</option>`).join('');
  const prescription = `<select id="maPrescription">
      <option value="no">خیر</option><option value="yes" ${window.MEDAID.prescription ? 'selected' : ''}>بله</option>
    </select>`;
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">🧾 درخواست کمک هزینهٔ درمانی</div>
    <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px">
      <div class="field"><label>کارمند</label>${empSelect || '<select id="maEmployee"></select>'}</div>
      <div class="field"><label>دستهٔ کمک هزینه</label>
        <select id="maCategory" onchange="window.MEDAID.category=this.value">${categories}</select></div>
      <div class="field"><label>تاریخ فاکتور (شمسی) ✻</label>
        <input id="maInvoiceDate" value="${esc(def.today || '')}" placeholder="1405/08/18"></div>
      <div class="field"><label>شمارهٔ فاکتور</label><input id="maInvoiceNo" placeholder="مثلاً 12345"></div>
      <div class="field"><label>مرکز درمانی / داروخانه / پزشک</label><input id="maProvider" placeholder="مثلاً داروخانهٔ دکتر رضایی"></div>
      <div class="field"><label>شمارهٔ نسخه یا معرفی‌نامه</label><input id="maPrescriptionNo"></div>
      <div class="field"><label>نسخه/معرفی‌نامه تأیید شده است</label>${prescription}</div>
      <div class="field"><label>سال کمک هزینه (شمسی)</label>
        <input id="maPeriodYear" class="num" value="${esc(String(def.currentYear || ''))}"></div>
    </div>
    <div id="medAidAnnual" class="muted" style="font-size:.76rem;margin:8px 0"></div>
    <div class="card-title" style="margin:10px 0 6px">اقلام درخواست</div>
    <div id="medAidRows"></div>
    <div class="row-actions" style="margin-top:8px">
      <button class="btn btn-ghost" onclick="medAidAddRow()">➕ افزودن قلم</button>
      <button class="btn btn-primary" onclick="medAidPlan()">🧮 محاسبهٔ سهم سازمان</button>
      <button class="btn" onclick="medAidSave()">💾 ثبت درخواست</button>
      <button class="btn btn-ghost" onclick="window.MEDAID.rows=[];medAidRenderRows()">🗑 خالی‌کردن اقلام</button>
    </div>
    <div class="muted" style="font-size:.74rem;margin-top:6px">
      سازمان بیش از تعرفهٔ مصوب نمی‌پردازد؛ مازاد فاکتور و مازاد سقف بر عهدهٔ کارمند می‌ماند.
    </div>
  </div>`;
}

/** گزینه‌های تعرفه به تفکیک دسته (optgroup). */
function medAidTariffOptions(selectedCode) {
  const def = window.MEDAID_DEF || {};
  const categories = def.categories || [];
  const tariffs = def.tariffs || [];
  return `<option value="">— انتخاب قلم تعرفه —</option>` + categories.map(c => {
    const items = tariffs.filter(t => t.category === c.key);
    if (!items.length) return '';
    return `<optgroup label="${esc(c.titleFa)}">${items.map(t =>
      `<option value="${esc(t.code)}" ${t.code === selectedCode ? 'selected' : ''}>${esc(t.titleFa)} — ${esc(t.amountFa)} ریال${t.unitFa ? ' (' + esc(fa(t.unitFa)) + ')' : ''}</option>`).join('')}</optgroup>`;
  }).join('');
}

/** ترسیم ردیف‌های اقلام. */
function medAidRenderRows() {
  const box = document.getElementById('medAidRows');
  if (!box) return;
  if (!window.MEDAID.rows.length) window.MEDAID.rows.push({code: '', qty: 1, amount: ''});
  const options = medAidTariffOptions('');
  box.innerHTML = window.MEDAID.rows.map((row, i) => `<div style="display:grid;grid-template-columns:minmax(0,2.4fr) minmax(0,.6fr) minmax(0,1fr) auto;gap:8px;align-items:end;margin-bottom:8px">
      <div class="field" style="min-width:0"><label>قلم تعرفه</label>
        <select id="maCode_${i}" style="width:100%">${options.replace('value="' + esc(row.code) + '"', 'value="' + esc(row.code) + '" selected')}</select></div>
      <div class="field"><label>تعداد</label><input id="maQty_${i}" class="num" value="${esc(String(row.qty || 1))}"></div>
      <div class="field"><label>مبلغ فاکتور (ریال)</label><input id="maAmount_${i}" class="num" value="${esc(String(row.amount || ''))}"></div>
      <button class="btn btn-sm btn-ghost" title="حذف قلم" onclick="medAidRemoveRow(${i})">🗑</button>
    </div>`).join('');
  window.MEDAID.rows.forEach((row, i) => {
    const sel = document.getElementById('maCode_' + i);
    if (sel && row.code) sel.value = row.code;
  });
  medAidSyncRows();
}

/** افزودن ردیف قلم. */
function medAidAddRow() {
  medAidSyncRows();
  window.MEDAID.rows.push({code: '', qty: 1, amount: ''});
  medAidRenderRows();
}

/** حذف ردیف قلم. */
function medAidRemoveRow(index) {
  medAidSyncRows();
  window.MEDAID.rows.splice(index, 1);
  medAidRenderRows();
}

/** خواندن مقدار ردیف‌ها از فرم به وضعیت. */
function medAidSyncRows() {
  window.MEDAID.rows = window.MEDAID.rows.map((row, i) => {
    const code = document.getElementById('maCode_' + i);
    const qty = document.getElementById('maQty_' + i);
    const amount = document.getElementById('maAmount_' + i);
    return {
      code: code ? code.value : row.code,
      qty: qty ? Number(qty.value) || 1 : row.qty,
      amount: amount ? amount.value : row.amount,
    };
  });
}

/** رشتهٔ اقلام برای سرور: «کد|تعداد|مبلغ» با جداکنندهٔ «؛». */
function medAidItemsText() {
  medAidSyncRows();
  return window.MEDAID.rows.filter(r => r.code).map(r => `${r.code}|${r.qty || 1}|${r.amount || 0}`).join('؛');
}

/** بدنهٔ درخواست برای سرور. */
function medAidBody() {
  const val = id => {
    const el = document.getElementById(id);
    return el ? el.value.trim() : '';
  };
  return {
    employeeId: val('maEmployee') || window.MEDAID.employeeId,
    category: val('maCategory'),
    periodYear: Number(val('maPeriodYear') || 0) || 0,
    invoiceDate: val('maInvoiceDate'),
    invoiceNo: val('maInvoiceNo'),
    provider: val('maProvider'),
    prescriptionNo: val('maPrescriptionNo'),
    prescriptionApproved: val('maPrescription') === 'yes',
    items: medAidItemsText(),
    noteFa: 'ثبت از صفحهٔ کمک هزینهٔ درمانی',
  };
}

/** محاسبهٔ پیش‌نمایش. */
async function medAidPlan() {
  const body = medAidBody();
  if (!body.items) { toast('دست‌کم یک قلم تعرفه انتخاب کنید', 'error'); return; }
  const box = document.getElementById('medAidPlanBox');
  if (box) box.innerHTML = '<div class="card card-pad muted">⏳ در حال محاسبه…</div>';
  const res = await apiFetch(medAidPrefix() + '/aid-plan', {method: 'POST', body});
  if (!res.ok) {
    if (box) box.innerHTML = liveErrorBox(res.message);
    toast(res.message, 'error');
    return;
  }
  window.MEDAID.plan = res.data || {};
  if (box) box.innerHTML = medAidResultHTML(window.MEDAID.plan);
  toast('محاسبه انجام شد');
}

/** ثبت درخواست با همان محاسبهٔ سرور. */
async function medAidSave() {
  const body = medAidBody();
  if (!body.employeeId) { toast('انتخاب کارمند الزامی است', 'error'); return; }
  if (!body.items) { toast('دست‌کم یک قلم تعرفه انتخاب کنید', 'error'); return; }
  const res = await apiFetch(medAidPrefix() + '/aid-claims', {method: 'POST', body});
  if (!res.ok) { toast(res.message, 'error'); return; }
  const claim = (res.data && res.data.claim) || {};
  toast('درخواست ثبت شد: ' + (claim.id || ''));
  window.MEDAID.rows = [];
  window.MEDAID.plan = null;
  liveReload();
}

/** خروجی محاسبه: جمع‌ها و سطرهای خط‌به‌خط. */
function medAidResultHTML(payload) {
  const r = payload.result || {};
  const rows = (r.lines || []).map(l => `<tr>
    <td class="num">${fa(l.order)}</td>
    <td><b>${esc(l.titleFa)}</b>${l.unitFa ? `<div class="muted" style="font-size:.7rem">${esc(fa(l.unitFa))}</div>` : ''}</td>
    <td class="num">${fa(l.quantity)}</td>
    <td class="num">${money(l.invoiceAmount)}</td>
    <td class="num">${money(l.tariffAmount)}</td>
    <td class="num">${money(l.baseAmount)}</td>
    <td class="num">${l.ceilingApplied ? money(l.ceilingApplied) : '—'}</td>
    <td class="num"><b>${money(l.payableAmount)}</b></td>
    <td class="num">${money(l.employeeAmount)}</td>
    <td style="font-size:.74rem">${esc(l.detailFa)}</td></tr>`);
  const chip = r.totalPayable > 0
    ? '<span class="chip c-ok">قابل پرداخت</span>'
    : '<span class="chip c-rej">قابل پرداخت نیست</span>';
  const annual = r.annualCeiling > 0
    ? `<div class="muted" style="font-size:.78rem">سقف سالانه: ${money(r.annualCeiling)} ریال — پرداخت پیشین سال: ${money(r.annualPaidBefore)} — باقی‌مانده پس از این درخواست: <b>${money(r.annualRemaining)}</b> ریال${r.annualExceeded ? ' <span class="chip c-wait">سقف پر شده و بخشی کسر شد</span>' : ''}</div>`
    : '';
  const blocking = (r.blockingFa || []).length
    ? `<div class="card card-pad" style="border-color:#fecaca;color:#b91c1c;font-size:.8rem">⛔ ${(r.blockingFa || []).map(esc).join('<br>')}</div>` : '';
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📋 نتیجهٔ محاسبه — ${esc(r.categoryFa || '')}</div>
    <div style="display:flex;gap:12px;flex-wrap:wrap;align-items:center;margin-bottom:8px">${chip}
      <span class="muted" style="font-size:.82rem">سهم سازمان: <b class="num">${money(r.totalPayable)}</b> ریال</span>
      <span class="muted" style="font-size:.82rem">بر عهدهٔ کارمند: <b class="num">${money(r.totalEmployee)}</b> ریال</span>
      <span class="muted" style="font-size:.82rem">سهم بیمه: <b class="num">${money(r.totalInsurance)}</b> ریال</span>
      <span class="muted" style="font-size:.82rem">کسر سقف: <b class="num">${money(r.ceilingApplied)}</b> ریال</span>
    </div>
    ${tbl(['#', 'قلم تعرفه', 'تعداد', 'فاکتور', 'تعرفه', 'مبنای محاسبه', 'کسر سقف', 'سهم سازمان', 'سهم کارمند', 'شرح'], rows, {foot: false})}
    ${annual}
    ${blocking}
    <div class="muted" style="font-size:.76rem;margin-top:6px">${esc(r.summaryFa || '')}</div>
  </div>`;
}

/* ------------------------------ کارتابل درخواست‌ها ------------------------------ */

/** فهرست درخواست‌های کمک هزینه. */
async function medAidClaims() {
  const res = await apiFetch(medAidPrefix() + '/records?kind=medicalAidClaims&size=100');
  return res.ok ? ((res.data && res.data.items) || []) : [];
}

/** جدول درخواست‌ها با اقدام‌های گردشکار. */
function medAidClaimsHTML(claims) {
  if (!claims.length) {
    return `<div class="card card-pad" style="margin-bottom:12px"><div class="card-title">📄 درخواست‌های کمک هزینه</div>
      <div class="muted" style="font-size:.8rem">درخواستی ثبت نشده است.</div></div>`;
  }
  const schemaActions = window.MEDAID_CLAIM_ACTIONS || null;
  const rows = claims.map(c => {
    const f = c.fields || {};
    // اقدام‌های مجاز از انتقال‌های اعلامی دامنه (هیچ مرحله‌ای در UI ساخته نمی‌شود).
    const allowed = schemaActions ? ((schemaActions.transitions || {})[c.status] || []) : [];
    const actions = schemaActions ? (schemaActions.actions || []).filter(a => !a.to || allowed.includes(a.to)) : [];
    const buttons = actions.map(a => {
      const primary = a.value === 'verify' || a.value === 'approve' || a.value === 'pay';
      return `<button class="btn btn-sm ${primary ? 'btn-primary' : 'btn-ghost'}"
        onclick="rkAct('${esc(medAidPrefix())}','${esc(c.id)}','${esc(a.value)}',${a.noteRequired ? 'true' : 'false'},'${esc(a.labelFa)}')">${esc(a.labelFa)}</button>`;
    }).join(' ');
    return `<tr>
      <td class="ltr" style="font-size:.74rem">${esc(c.id)}</td>
      <td>${personCell(c.employeeId, c.employeeName)}</td>
      <td>${esc(f.category ? medAidCategoryFa(f.category) : '—')}</td>
      <td>${faDate(f.invoiceDate)}</td>
      <td class="num">${money(f.invoiceTotal || 0)}</td>
      <td class="num"><b>${money(f.payableAmount || 0)}</b></td>
      <td class="num">${money(f.employeeAmount || 0)}</td>
      <td>${statusChip(c.status, c.statusFa)}</td>
      <td><div class="row-actions">${buttons || '<span class="muted">—</span>'}</div></td></tr>`;
  });
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📄 درخواست‌های کمک هزینه (${fa(claims.length)})</div>
    ${liveTable(['شناسه', 'کارمند', 'دسته', 'تاریخ فاکتور', 'جمع فاکتور', 'سهم سازمان', 'سهم کارمند', 'وضعیت', 'اقدام'], rows, claims.length)}
    <div class="muted" style="font-size:.74rem;margin-top:6px">اقدام‌های هر ردیف از انتقال‌های اعلامی دامنه می‌آید؛ «ثبت پرداخت» بی‌شمارهٔ سند پرداخت انجام نمی‌شود.</div>
  </div>`;
}

/** عنوان فارسی یک دسته از پرونده (بدون تکرار فهرست در UI). */
function medAidCategoryFa(key) {
  const def = window.MEDAID_DEF || {};
  const hit = (def.categories || []).find(c => c.key === key);
  return hit ? hit.titleFa : key;
}

/* ------------------------------ فهرست تعرفه‌ها ------------------------------ */

/** جدول تعرفه‌های جاری به تفکیک دسته. */
function medAidTariffsHTML(def) {
  const tariffs = def.tariffs || [];
  const categories = def.categories || [];
  if (!tariffs.length) {
    return `<div class="card card-pad" style="margin-bottom:12px"><div class="card-title">📗 تعرفه‌های مصوب</div>
      <div class="muted" style="font-size:.8rem">تعرفه‌ای ثبت نشده است؛ با دکمهٔ «افزودن تعرفه» شروع کنید.</div>
      <button class="btn btn-primary" style="margin-top:8px" onclick="rkCreate('medicalTariffs')">➕ افزودن تعرفه</button></div>`;
  }
  const sections = categories.map(c => {
    const items = tariffs.filter(t => t.category === c.key);
    if (!items.length) return '';
    // اقدام‌های مجاز هر تعرفه از انتقال‌های اعلامی دامنه می‌آید (جاری‌سازی/غیرفعال‌سازی).
    const kind = window.MEDAID_KINDS ? window.MEDAID_KINDS.tariffs : null;
    const rows = items.map(t => {
      const allowed = kind ? ((kind.transitions || {})[t.status] || []) : [];
      const actions = kind ? (kind.actions || []).filter(a => !a.to || allowed.includes(a.to)) : [];
      const buttons = actions.map(a =>
        `<button class="btn btn-sm ${a.style === 'primary' ? 'btn-primary' : 'btn-ghost'}"
          onclick="rkAct('${esc(medAidPrefix())}','${esc(t.recordId)}','${esc(a.value)}',${a.noteRequired ? 'true' : 'false'},'${esc(a.labelFa)}')">${esc(a.labelFa)}</button>`).join(' ');
      return `<tr>
      <td class="ltr" style="font-size:.72rem">${esc(t.code)}</td>
      <td><b>${esc(t.titleFa)}</b>${t.refFa ? `<div class="muted" style="font-size:.68rem">${esc(fa(t.refFa))}</div>` : ''}</td>
      <td>${esc(t.unitFa ? fa(t.unitFa) : '—')}</td>
      <td class="num">${esc(t.amountFa || money(t.amount))}</td>
      <td class="num">${t.ceilingAmount ? money(t.ceilingAmount) : '—'}</td>
      <td class="num">${t.orgSharePercent ? fa(Math.round(t.orgSharePercent * 100)) + '٪' : '۱۰۰٪'}</td>
      <td>${t.requirePrescription ? 'نیازمند نسخه' : '—'}</td>
      <td>${statusChip(t.status, t.statusFa)}</td>
      <td><div class="row-actions">
        ${buttons || '<span class="muted">—</span>'}
        <button class="btn btn-sm btn-ghost" title="ویرایش" onclick="rkEdit('medicalTariffs','${esc(t.recordId)}')">✏️</button>
      </div></td></tr>`;
    });
    return `<div class="card card-pad" style="margin-bottom:12px">
      <div class="card-title">📗 ${esc(c.titleFa)} <span class="muted" style="font-size:.74rem">— ${fa(items.length)} قلم</span></div>
      ${liveTable(['کد', 'شرح قلم', 'واحد', 'تعرفه (ریال)', 'سقف قلم', 'سهم سازمان', 'نسخه', 'وضعیت', 'اقدام'], rows, items.length)}
    </div>`;
  }).join('');
  return `<div class="card-title" style="margin:14px 0 6px">📗 تعرفه‌های مصوب ۱۴۰۵</div>` +
    `<div class="row-actions" style="margin-bottom:10px">
      <button class="btn btn-primary" onclick="rkCreate('medicalTariffs')">➕ افزودن تعرفه</button>
      <span class="muted" style="font-size:.76rem">تعرفه‌ها دادهٔ پارامتریک‌اند: با ویرایش ردیف و ثبت تاریخ اجرا، محاسبهٔ درخواست‌های بعدی با نرخ تازه انجام می‌شود.</span>
    </div>` + sections;
}

/** دستورالعمل کمک هزینه‌ها. */
function medAidInstructionHTML(def) {
  const steps = (def.instruction || []);
  if (!steps.length) return '';
  const params = def.params || {};
  const views = (def.paramViews || []).map(p =>
    `<tr><th style="text-align:right;background:#F8FAFC">${esc(p.titleFa || p.key)}</th><td class="num">${esc(fa(p.value))} <span class="muted">${esc(p.unitFa || '')}</span></td></tr>`).join('');
  return `<div class="card card-pad" style="margin-bottom:12px">
    <div class="card-title">📘 دستورالعمل کمک هزینه‌ها و پارامترهای شرکت</div>
    <ol style="margin:0;padding-inline-start:20px;line-height:2.1;font-size:.84rem">
      ${steps.map(s => `<li>${esc(s)}</li>`).join('')}
    </ol>
    <div class="card-title" style="margin:12px 0 6px">⚙️ پارامترهای به‌کاررفته</div>
    <table class="tbl" style="width:100%">
      <tr><th style="text-align:right;background:#F8FAFC">سهم پیش‌فرض سازمان</th><td class="num">${fa(Math.round((params.orgSharePercent || 0) * 100))}٪</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">سقف سالانهٔ هر کارمند</th><td class="num">${params.annualCeiling ? money(params.annualCeiling) + ' ریال' : 'بی‌سقف'}</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">سقف پیش‌فرض هر قلم</th><td class="num">${params.perItemCeiling ? money(params.perItemCeiling) + ' ریال' : 'بی‌سقف (تعرفهٔ قلم)'}</td></tr>
      <tr><th style="text-align:right;background:#F8FAFC">الزام نسخهٔ تأییدشده</th><td>${params.requirePrescription ? 'بله' : 'خیر'}</td></tr>
      ${views}
    </table>
  </div>`;
}

/* ثبت در مسیریاب زنده. */
LIVE_PAGES.medicalAid = pageMedicalAidLive;

/** آماده‌سازی پس از ترسیم: ردیف‌های اقلام، دستهٔ پیش‌فرض و مصرف سقف سالانه.
 *
 * چرا با تأخیر کوتاه: بدنهٔ صفحه پس از برگشت پوسته در `#liveRegion` می‌نشیند؛ پیش از آن
 * عنصرها در DOM نیستند و مقداردهی بی‌اثر می‌شد.
 */
function medAidPrime(def) {
  setTimeout(() => {
    const emp = document.getElementById('maEmployee');
    if (emp) {
      window.MEDAID.employeeId = emp.value || '';
      emp.addEventListener('change', () => { window.MEDAID.employeeId = emp.value; medAidAnnualHint(emp.value); });
    }
    const cat = document.getElementById('maCategory');
    if (cat && !window.MEDAID.category) window.MEDAID.category = cat.value;
    if (!window.MEDAID.rows.length) window.MEDAID.rows.push({code: '', qty: 1, amount: ''});
    medAidRenderRows();
    if (window.MEDAID.employeeId) medAidAnnualHint(window.MEDAID.employeeId);
  }, 60);
  return '';
}

/** نمایش مصرف سقف سالانهٔ کارمند. */
async function medAidAnnualHint(employeeId) {
  const box = document.getElementById('medAidAnnual');
  if (!box) return;
  const yearEl = document.getElementById('maPeriodYear');
  const year = yearEl ? yearEl.value.trim() : '';
  const res = await apiFetch(medAidPrefix() + '/aid-annual?employeeId=' + encodeURIComponent(employeeId) + '&year=' + encodeURIComponent(year));
  if (!res.ok) { box.innerHTML = ''; return; }
  const d = res.data || {};
  box.innerHTML = d.ceiling
    ? `پرداخت‌های تأییدشدهٔ سال <b class="num">${esc(String(d.year))}</b>: <b class="num">${esc(d.annualPaidFa)}</b> ریال در ${fa(d.claimsCount)} درخواست — باقی‌ماندهٔ سقف: <b class="num">${esc(d.remainingFa)}</b> ریال`
    : `پرداخت‌های تأییدشدهٔ سال <b class="num">${esc(String(d.year))}</b>: <b class="num">${esc(d.annualPaidFa)}</b> ریال در ${fa(d.claimsCount)} درخواست — سقف سالانه‌ای برای شرکت تعیین نشده است.`;
}