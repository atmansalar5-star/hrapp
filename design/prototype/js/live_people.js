/* =====================================================================
   صفحه‌های زندهٔ کارکنان و پروندهٔ کارمند (با تب‌های مشخصات)
   =====================================================================

   فهرست کارکنان و پروندهٔ کارمند روی دادهٔ واقعی سرور کار می‌کنند و پرونده در تب‌ها
   چیده می‌شود: «مشخصات شخصی»، «مشخصات شغلی»، «قرارداد و اسناد»، «حضور و کارکرد» و
   «سوابق تغییرات». هیچ قاعده‌ای اینجا نیست؛ فقط خواندن، فیلتر کردن و چیدن نمایشی.

   تب سوابق همان چیزی را می‌نشاند که سرور اعلام می‌کند: هر ماژول سابقهٔ کارمند را
   می‌دهد (احکام قبلی، قراردادهای پیشین، ترفیعات، مدارک، وام‌ها…) و این‌جا فقط چیده
   می‌شود. سند جاری با نشانهٔ «سند جاری» از اسناد پیشین جدا می‌شود.

   احتیاط صادقانه: هر فیلدی که سرور نداشته باشد «—» نشان داده می‌شود، نه مقدار ساختگی.
   ===================================================================== */

/** وضعیت کارمند → چیپ رنگی. */
const EMP_STATUS_CLS = {active: 'c-ok', leave: 'c-wait', probation: 'c-wait', terminated: 'c-rej'};

/** نگاشت کلید نوع استخدام به برچسب فارسی از گزینه‌های سرور. */
function employmentTypeOptions(options) {
  return (options && options.employmentTypes) || [];
}

/**
 * سلول کارمند با عکس کارگزینی در فهرست کارکنان.
 *
 * اگر عکس بارگذاری نشده باشد، همان آواتار حرف اول نام می‌ماند؛ هیچ تصویر ساختگی
 * ساخته نمی‌شود. نشانی عکس از سرور می‌آید (امضاشده) و با <img> دیده می‌شود.
 */
function photoPersonCell(employee) {
  const id = employee.id || '';
  const name = employee.fullName || '—';
  const photo = employee.photoUrl;
  const img = photo
    ? `<img class="photo sm" src="${esc(photo)}" alt="عکس کارگزینی ${esc(name)}" loading="lazy">`
    : `<div class="avatar sq">${esc(initial(name === '—' ? '؟' : name))}</div>`;
  return `<div class="emp-cell">${img}<div>${esc(name)}<div class="muted ltr" style="font-size:.66rem">${esc(id)}</div></div></div>`;
}

/* ======================= فهرست کارکنان ======================= */

function pageEmployeesLive() {
  if (!isLive()) return offlineFallback('employees');
  liveLoad(async () => {
    const q = new URLSearchParams({size: '200'});
    if (window.__EMP_QUERY) q.set('query', window.__EMP_QUERY);
    if (window.__EMP_DEPT) q.set('departmentId', window.__EMP_DEPT);
    if (window.__EMP_STATUS) q.set('status', window.__EMP_STATUS);
    const [listRes, optRes] = await Promise.all([
      apiFetch('/api/v1/employee/employees?' + q.toString()),
      apiFetch('/api/v1/employee/options'),
    ]);
    if (!listRes.ok) return liveErrorBox(listRes.message);
    const items = (listRes.data && listRes.data.items) || [];
    const options = optRes.ok ? optRes.data : {};
    window.__EMP_OPTIONS = options;

    const byStatus = {};
    for (const e of items) byStatus[e.status] = (byStatus[e.status] || 0) + 1;
    const totalPayroll = items.reduce((s, e) => s + (Number(e.baseSalary) || 0), 0);

    const stats = `<div class="grid-stats">
      ${statCard({ico: '👥', tint: '#EFF6FF', accent: '#2563EB', val: fa((listRes.data && listRes.data.total) || items.length), label: 'کل کارکنان'})}
      ${statCard({ico: '🟢', tint: '#DCFCE7', accent: '#16A34A', val: fa(byStatus.active || 0), label: 'شاغل'})}
      ${statCard({ico: '🟡', tint: '#FEF3C7', accent: '#D97706', val: fa((byStatus.leave || 0) + (byStatus.probation || 0)), label: 'مرخصی / آزمایشی'})}
      ${statCard({ico: '💰', tint: '#EDE9FE', accent: '#7C3AED', val: money(totalPayroll), label: 'جمع حقوق پایه (ریال)'})}
    </div>`;

    const deptOpts = ((options.departments) || [])
      .map(d => `<option value="${esc(d.id)}" ${window.__EMP_DEPT === d.id ? 'selected' : ''}>${esc(d.name)}</option>`).join('');
    const statusOpts = [['active', 'شاغل'], ['leave', 'مرخصی'], ['probation', 'آزمایشی'], ['terminated', 'پایان خدمت']]
      .map(([v, l]) => `<option value="${v}" ${window.__EMP_STATUS === v ? 'selected' : ''}>${l}</option>`).join('');
    const filters = `<div class="card card-pad" style="margin-bottom:12px;display:flex;gap:10px;align-items:flex-end;flex-wrap:wrap">
      <div class="field" style="flex:1;min-width:200px"><label>جست‌وجو (نام یا کد پرسنلی)</label>
        <input id="empQuery" value="${esc(window.__EMP_QUERY || '')}" placeholder="مثلاً سارا یا EMP-001"></div>
      <div class="field"><label>واحد سازمانی</label>
        <select id="empDept"><option value="">همهٔ واحدها</option>${deptOpts}</select></div>
      <div class="field"><label>وضعیت</label>
        <select id="empStatus"><option value="">همه</option>${statusOpts}</select></div>
      <button class="btn btn-primary" onclick="employeesApplyFilter()">🔍 اعمال</button>
      <button class="btn btn-ghost" onclick="employeesClearFilter()">✖ پاک‌کردن</button>
    </div>`;

    const rows = items.map(e => `<tr>
      <td>${photoPersonCell(e)}</td>
      <td>${esc(e.departmentName || '—')}</td>
      <td>${esc(e.position || '—')}</td>
      <td>${e.employmentTypeFa ? `<span class="chip" style="font-size:.72rem">${esc(e.employmentTypeFa)}</span>` : esc(e.contractType || '—')}</td>
      <td>${faDate(e.hireDate)}</td>
      <td><span class="num">${money(e.baseSalary || 0)}</span></td>
      <td>${faNum(e.leaveBalance || 0)}</td>
      <td>${pChip(EMP_STATUS_CLS[e.status] || 'c-wait', empStatusFa(e.status))}</td>
      <td class="ltr num" style="font-size:.72rem">${esc(e.nationalId || '—')}</td>
      <td><div class="row-actions">
        <button class="btn btn-sm btn-primary" onclick="go('profile','${esc(e.id)}')">پرونده</button>
        <button class="btn btn-sm btn-ghost" title="ویرایش اطلاعات پایه" onclick="liveEmployeeForm('${esc(e.id)}')">✏️</button>
        <button class="btn btn-sm btn-ghost" title="گواهی اشتغال" onclick="liveOpenDoc('employment_certificate','${esc(e.id)}')">📄</button>
        <button class="btn btn-sm btn-ghost" title="فیش حقوقی" onclick="liveOpenDoc('payslip','${esc(e.id)}')">🧾</button>
      </div></td></tr>`);

    return stats + filters + statNote('فهرست روی محدودهٔ دسترسی نقش شما خوانده می‌شود: مدیر سیستم همهٔ کارکنان را می‌بیند و نقش‌های دیگر بر پایهٔ سیاست سرور. فیلترها روی سرور اعمال می‌شوند.') +
      liveTable(['کارمند', 'واحد سازمانی', 'پست', 'نوع استخدام', 'تاریخ استخدام', 'حقوق پایه (ریال)', 'مانده مرخصی', 'وضعیت', 'کد ملی', 'عملیات'],
        rows, (listRes.data && listRes.data.total) || items.length);
  });
  return liveShell('فهرست کارکنان', 'کارکنان این استقرار روی دادهٔ واقعی سرور — با پروندهٔ تبیندی برای هر کارمند',
    `<button class="btn btn-primary" onclick="liveEmployeeForm()">➕ کارمند تازه</button>
     <button class="btn btn-ghost" onclick="toast('خروجی اکسل در فاز اتصال به مالی فعال می‌شود','info')">⬇️ خروجی</button>`);
}

/* =====================================================================
   فرم ایجاد و ویرایش کارمند — اطلاعات پایهٔ پرونده
   ---------------------------------------------------------------------
   این فرم مرجع اطلاعات پایه است: کد ملی، شمارهٔ موبایل، شمارهٔ منزل، کد پستی و
   نشانی محل سکونت همین‌جا ثبت می‌شوند و فرم‌های دیگر (قرارداد، حکم، انتصاب...)
   آن‌ها را از همین پرونده می‌خوانند. قاعده‌های اعتبارسنجی در سرور است؛ پیام
   خطای هر فیلد همان‌جا برمی‌گردد و برچسبش در فرم نشان داده می‌شود.
   ===================================================================== */
async function liveEmployeeForm(id) {
  const optRes = await apiFetch('/api/v1/employee/options');
  const options = (optRes.ok && optRes.data) || {};
  let e = {id: '', fullName: '', nationalId: '', gender: '', birthDate: '', mobile: '', homePhone: '', phone: '', postalCode: '', address: '', residenceSite: '', email: '', departmentId: '', position: '', contractType: 'تمام‌وقت', employmentType: '', hireDate: '', baseSalary: 0, status: 'active'};
  if (id) {
    const res = await apiFetch('/api/v1/employee/employees/' + encodeURIComponent(id));
    if (!res.ok) { toast(res.message, 'error'); return; }
    e = Object.assign(e, res.data || {});
  }
  const deptOpts = ((options.departments) || []).map(d =>
    `<option value="${esc(d.id)}" ${d.id === e.departmentId ? 'selected' : ''}>${esc(d.name)}</option>`).join('');
  const typeOpts = ((options.employmentTypes) || []).map(t =>
    `<option value="${esc(t.key)}" ${t.key === e.employmentType ? 'selected' : ''}>${esc(t.titleFa)}</option>`).join('');
  const genderOpts = [`<option value="">ثبت نشده</option>`].concat(((options.genderOptions) || []).map(g =>
    `<option value="${esc(g.key)}" ${g.key === e.gender ? 'selected' : ''}>${esc(g.titleFa)}</option>`)).join('');
  const contractOpts = (['تمام‌وقت', 'پیمانکاری', 'پاره‌وقت']).map(t =>
    `<option ${t === e.contractType ? 'selected' : ''}>${t}</option>`).join('');
  const statusOpts = ((options.statusOptions) || []).map(s =>
    `<option value="${esc(s.key)}" ${s.key === e.status ? 'selected' : ''}>${esc(s.titleFa)}</option>`).join('');
  // محل سکونت: کلید جدول ۱ و ۲ آیین‌نامهٔ فوق‌العاده‌های صندوق نفت؛ درصد فوق‌العادهٔ
  // محرومیت و جبران صرف وقت در فیش حقوقی از همین انتخاب خوانده می‌شود.
  const residenceOpts = [`<option value="">ثبت نشده (درصد پیش‌فرض شرکت)</option>`]
    .concat(((options.residenceOptions) || []).map(r =>
      `<option value="${esc(r.key)}" ${r.key === e.residenceSite ? 'selected' : ''}>${esc(r.titleFa)}</option>`)).join('');
  openModal({
    title: id ? 'ویرایش کارمند — ' + (e.fullName || e.id) : 'ثبت کارمند تازه',
    body: `<fieldset class="field-group"><legend>اطلاعات هویتی و تماس</legend>
      <div class="form-grid">
        <div class="field"><label>نام و نام خانوادگی <span class="req">*</span></label><input id="neName" value="${esc(e.fullName || '')}"></div>
        <div class="field"><label>کد ملی <span class="req">*</span></label><input id="neNid" class="ltr num" value="${esc(e.nationalId || '')}" placeholder="۱۰ رقم"></div>
        <div class="field"><label>شمارهٔ موبایل</label><input id="neMobile" class="ltr num" value="${esc(e.mobile || '')}" placeholder="09xxxxxxxxx"></div>
        <div class="field"><label>شمارهٔ منزل</label><input id="neHomePhone" class="ltr num" value="${esc(e.homePhone || '')}" placeholder="061xxxxxxxx"></div>
        <div class="field"><label>کد پستی</label><input id="nePostal" class="ltr num" value="${esc(e.postalCode || '')}" placeholder="۱۰ رقم"></div>
        <div class="field"><label>تلفن تماس دیگر</label><input id="nePhone" class="ltr num" value="${esc(e.phone || '')}"></div>
        <div class="field full"><label>آدرس محل سکونت</label><input id="neAddress" value="${esc(e.address || '')}" placeholder="شهر، خیابان، پلاک"></div>
        <div class="field"><label>جنسیت</label><select id="neGender">${genderOpts}</select></div>
        <div class="field"><label>محل سکونت (مبنای جدول ۱ و ۲ فوق‌العاده‌ها)</label><select id="neResidence">${residenceOpts}</select></div>
        <div class="field"><label>تاریخ تولد (شمسی)</label><input id="neBirth" class="num" value="${esc(e.birthDate || '')}" placeholder="1368/04/12"></div>
        <div class="field"><label>ایمیل</label><input id="neEmail" class="ltr" value="${esc(e.email || '')}"></div>
      </div></fieldset>
      <fieldset class="field-group"><legend>اطلاعات شغلی</legend>
      <div class="form-grid">
        <div class="field"><label>واحد سازمانی <span class="req">*</span></label><select id="neDept">${deptOpts}</select></div>
        <div class="field"><label>پست سازمانی <span class="req">*</span></label><input id="nePosition" value="${esc(e.position || '')}"></div>
        <div class="field"><label>نوع قرارداد</label><select id="neContract">${contractOpts}</select></div>
        <div class="field"><label>نوع استخدام</label><select id="neEmploymentType">${typeOpts}</select></div>
        <div class="field"><label>تاریخ استخدام (شمسی) <span class="req">*</span></label><input id="neHire" class="num" value="${esc(e.hireDate || '')}" placeholder="1404/07/01"></div>
        <div class="field"><label>حقوق پایه (ریال)</label><input id="neBase" type="number" min="0" value="${esc(String(e.baseSalary || ''))}"></div>
        <div class="field"><label>وضعیت</label><select id="neStatus">${statusOpts}</select></div>
      </div></fieldset>`,
    foot: `<button class="btn btn-primary" onclick="liveEmployeeSave('${esc(id || '')}')">💾 ذخیره</button>
           <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`,
  });
}

/** ذخیرهٔ کارمند: قاعدهٔ اعتبارسنجی در سرور است و پیام هر فیلد همان‌جا می‌آید. */
async function liveEmployeeSave(id) {
  const body = {
    fullName: ($('#neName').value || '').trim(),
    nationalId: digitsOf($('#neNid').value),
    gender: $('#neGender').value,
    birthDate: digitsOf($('#neBirth').value),
    mobile: digitsOf($('#neMobile').value),
    homePhone: digitsOf($('#neHomePhone').value),
    postalCode: digitsOf($('#nePostal').value),
    phone: digitsOf($('#nePhone').value),
    address: ($('#neAddress').value || '').trim(),
    residenceSite: $('#neResidence').value,
    email: ($('#neEmail').value || '').trim(),
    departmentId: $('#neDept').value,
    position: ($('#nePosition').value || '').trim(),
    contractType: $('#neContract').value,
    employmentType: $('#neEmploymentType').value,
    hireDate: ($('#neHire').value || '').trim(),
    baseSalary: Number(digitsOf($('#neBase').value)) || 0,
    status: $('#neStatus').value,
  };
  const res = await apiFetch('/api/v1/employee/employees' + (id ? '/' + encodeURIComponent(id) : ''),
    {method: id ? 'PATCH' : 'POST', body});
  if (!res.ok) {
    // پیام‌های اعتبارسنجی فیلدبه‌فیلد از سرور می‌آید (کلید = نام فیلد).
    const details = (res.data && res.data.details) || {};
    const labels = {fullName: 'نام', nationalId: 'کد ملی', gender: 'جنسیت', birthDate: 'تاریخ تولد', mobile: 'موبایل', homePhone: 'شمارهٔ منزل',
      postalCode: 'کد پستی', address: 'نشانی', email: 'ایمیل', departmentId: 'واحد', position: 'پست',
      hireDate: 'تاریخ استخدام', baseSalary: 'حقوق پایه', phone: 'تلفن'};
    const first = Object.keys(details)[0];
    toast(first ? (labels[first] || first) + ': ' + details[first] : res.message, 'error');
    return;
  }
  EMP_BASE_CACHE = {};
  closeModal();
  toast(id ? 'اطلاعات کارمند ویرایش شد' : 'کارمند تازه ثبت شد');
  liveReload();
}

/** برچسب فارسی وضعیت کارمند. */
function empStatusFa(status) {
  const map = {active: 'شاغل', leave: 'مرخصی', probation: 'آزمایشی', terminated: 'پایان خدمت'};
  return map[status] || status || '—';
}

/** اعمال فیلترهای فهرست کارکنان — همان درخواست، این بار با پارامترهای فیلتر سرور. */
function employeesApplyFilter() {
  window.__EMP_QUERY = (document.getElementById('empQuery') || {}).value || '';
  window.__EMP_DEPT = (document.getElementById('empDept') || {}).value || '';
  window.__EMP_STATUS = (document.getElementById('empStatus') || {}).value || '';
  go('employees');
}

function employeesClearFilter() {
  window.__EMP_QUERY = ''; window.__EMP_DEPT = ''; window.__EMP_STATUS = '';
  go('employees');
}

/* ======================= پروندهٔ کارمند (تبیندی) ======================= */

/** تب فعال پروندهٔ کارمند زنده. */
let profileLiveTab = 'personal';

/** پروندهٔ کارمند با چهار تب مشخصات — هر تب دادهٔ واقعی خودش را می‌خواند. */
function pageProfileLive(id) {
  if (!isLive()) return offlineFallback('profile');
  if (!id) {
    return pageHead('پرونده کارمند', 'کارمندی انتخاب نشده است') +
      `<div class="card card-pad" style="line-height:2">
        برای دیدن پرونده، از <a href="#" onclick="go('employees');return false" style="color:var(--primary)">فهرست کارکنان</a> روی «پرونده» بزنید.
      </div>`;
  }

  liveLoad(async () => {
    const [empRes, conRes, attRes, attSchemaRes, histRes, photoRes] = await Promise.all([
      apiFetch('/api/v1/employee/employees/' + encodeURIComponent(id)),
      apiFetch('/api/v1/contracts/contracts?size=100&employeeId=' + encodeURIComponent(id)),
      apiFetch('/api/v1/attendance/records?size=100&employeeId=' + encodeURIComponent(id)),
      apiFetch('/api/v1/attendance/schema'),
      apiFetch('/api/v1/history?employeeId=' + encodeURIComponent(id)),
      apiFetch('/api/v1/employee/employees/' + encodeURIComponent(id) + '/photo'),
    ]);
    if (!empRes.ok) return liveErrorBox(empRes.message);
    const e = empRes.data || {};
    // وضعیت عکس کارگزینی از مسیر اختصاصی خودش می‌آید (تاریخ، حجم، اثر انگشت)؛ اگر آن
    // مسیر در دسترس نباشد، همان نشانی عکسِ پرونده مبنای نمایش است.
    const photoState = photoRes.ok && photoRes.data ? photoRes.data : {hasPhoto: !!e.photoUrl, url: e.photoUrl || ''};
    const contracts = (conRes.ok && (conRes.data.items || [])) || [];
    const attItems = (attRes.ok && (attRes.data.items || [])) || [];
    // برچسب فارسی فیلدها از شناسهٔ سرور می‌آید؛ هیچ برچسبی در رابط ساخته نمی‌شود.
    const fieldLabels = {};
    for (const kind of (attSchemaRes.ok && (attSchemaRes.data.kinds || [])) || []) {
      fieldLabels[kind.key] = {};
      for (const f of kind.fields || []) fieldLabels[kind.key][f.key] = f.labelFa || f.key;
    }

    const tabs = [['personal', 'مشخصات شخصی'], ['job', 'مشخصات شغلی'], ['education', 'تحصیلات و مدرک ملاک'], ['photo', 'عکس پرسنلی'], ['docs', 'قرارداد و اسناد'], ['time', 'حضور و کارکرد'], ['history', 'سوابق تغییرات']];
    const tabBar = `<div class="tabs">${tabs.map(([k, l]) =>
      `<button class="tab ${profileLiveTab === k ? 'active' : ''}" onclick="profileLiveTab='${k}';go('profile','${esc(id)}')">${l}</button>`).join('')}</div>`;

    // سرصفحهٔ پرونده: عکس کارگزینی واقعی اگر بارگذاری شده باشد، وگرنه حرف اول نام.
    const headAvatar = e.photoUrl
      ? `<img class="photo lg" src="${esc(e.photoUrl)}" alt="عکس کارگزینی ${esc(e.fullName || '')}">`
      : `<div class="avatar lg">${esc(initial(e.fullName || '؟'))}</div>`;
    const head = `<div class="card card-pad" style="margin-bottom:12px">
      <div class="profile-head">
        ${headAvatar}
        <div>
          <div class="profile-name">${esc(e.fullName || '—')}</div>
          <div class="profile-job">${esc(e.position || '—')} — ${esc(e.departmentName || '—')}</div>
          <div class="profile-tags">
            ${pChip(EMP_STATUS_CLS[e.status] || 'c-wait', empStatusFa(e.status))}
            <span class="chip contract">${esc(e.contractType || '—')}</span>
            ${e.employmentTypeFa ? `<span class="chip">${esc(e.employmentTypeFa)}</span>` : ''}
            <span class="chip c-done num">${esc(e.id)}</span>
            ${photoState.hasPhoto ? `<span class="chip">🖼️ عکس کارگزینی ثبت شده</span>` : `<span class="chip c-wait">بدون عکس کارگزینی</span>`}
          </div>
        </div>
      </div>
    </div>`;

    const kv = pairs => `<dl class="kv">${pairs.map(([k, v]) =>
      `<dt>${esc(k)}</dt><dd>${v}</dd>`).join('')}</dl>`;

    const personal = kv([
      ['نام و نام خانوادگی', esc(e.fullName || '—')],
      ['کد پرسنلی', `<span class="ltr num">${esc(e.id)}</span>`],
      ['کد ملی', e.nationalId ? `<span class="ltr num">${esc(e.nationalId)}</span>` : '—'],
      ['جنسیت', esc(e.genderFa || '—')],
      ['تاریخ تولد', e.birthDate ? `<span class="num">${fa(e.birthDate)}</span>` : '—'],
      ['شمارهٔ موبایل', e.mobile ? `<span class="ltr num">${esc(e.mobile)}</span>` : '—'],
      ['شمارهٔ منزل', e.homePhone ? `<span class="ltr num">${esc(e.homePhone)}</span>` : '—'],
      ['تلفن تماس', e.phone ? `<span class="ltr num">${esc(e.phone)}</span>` : '—'],
      ['کد پستی', e.postalCode ? `<span class="ltr num">${esc(e.postalCode)}</span>` : '—'],
      ['آدرس محل سکونت', e.address ? esc(e.address) : '—'],
      ['ایمیل', e.email ? `<span class="ltr">${esc(e.email)}</span>` : '—'],
      ['مدیر مستقیم', esc(e.managerName || '—')],
      ['عکس کارگزینی', photoState.hasPhoto
        ? `ثبت‌شده ${photoState.updatedAt ? '— ' + fa(photoState.updatedAt) : ''} <button class="btn btn-sm btn-ghost" onclick="profileLiveTab='photo';go('profile','${esc(id)}')">نمایش و تغییر</button>`
        : `ثبت نشده — <button class="btn btn-sm btn-primary" onclick="profileLiveTab='photo';go('profile','${esc(id)}')">بارگذاری عکس</button>`],
    ]) + `<div class="row-actions" style="margin-top:8px">
      <button class="btn btn-ghost" onclick="liveEmployeeForm('${esc(e.id)}')">✏️ ویرایش اطلاعات پایه</button>
    </div>` + statNote('اطلاعات پایهٔ همین تب مرجع همهٔ فرم‌های دیگر است: فرم قرارداد، حکم حقوق، انتصاب و جابه‌جایی همین کد ملی، شماره‌ها، کد پستی، نشانی و مدرک تحصیلی را از پرونده می‌خوانند و دوباره از کاربر نمی‌پرسند. مدارک تحصیلی تفصیلی در «بایگانی و شناسایی کارکنان» ثبت می‌شود.');

    // تب عکس پرسنلی: نمایش «عکس کارگزینی» و بارگذاری/حذف آن. فایل در پوشهٔ دادهٔ سرور
    // می‌ماند و نشانی نمایش آن امضاشده است؛ قاعدهٔ نوع و حجم فایل در سرور سنجیده می‌شود.
    const photoPreview = photoState.hasPhoto && photoState.url
      ? `<img class="photo-xl" src="${esc(photoState.url)}" alt="عکس کارگزینی ${esc(e.fullName || '')}">`
      : `<div class="empty-state" style="border:1px dashed var(--border);border-radius:12px;padding:24px 12px">
           <div class="big">🖼️</div>برای این کارمند عکس کارگزینی بارگذاری نشده است</div>`;
    const photoInfo = kv([
      ['وضعیت عکس', photoState.hasPhoto ? 'بارگذاری‌شده' : 'ثبت‌نشده'],
      ['تاریخ بارگذاری', photoState.updatedAt ? fa(photoState.updatedAt) : '—'],
      ['حجم فایل', photoState.sizeKb ? `${fa(photoState.sizeKb)} کیلوبایت` : '—'],
      ['نوع فایل', esc(photoState.contentType || '—')],
      ['نام فایل اصلی', photoState.originalName ? `<span class="ltr">${esc(photoState.originalName)}</span>` : '—'],
      ['بارگذاری‌کننده', esc(photoState.updatedBy || '—')],
    ]);
    const photo = `<div class="grid-2eq">
      <div style="text-align:center">
        ${photoPreview}
        <div class="muted" style="font-size:.78rem;margin-top:8px">عکس کارگزینی — همان عکسی که در ساختار سازمانی و کارت پرسنلی دیده می‌شود.</div>
      </div>
      <div>
        ${photoInfo}
        <div class="field" style="margin-top:14px">
          <label>بارگذاری عکس تازه (PNG، JPEG یا WEBP تا ۴ مگابایت)</label>
          <input id="empPhotoFile" type="file" accept="image/png,image/jpeg,image/webp">
        </div>
        <div class="row-actions" style="margin-top:8px">
          <button class="btn btn-primary" onclick="empPhotoUpload('${esc(e.id)}')">⬆️ بارگذاری عکس</button>
          <button class="btn btn-ghost" onclick="empPhotoDelete('${esc(e.id)}')">🗑 حذف عکس</button>
        </div>
        <div class="muted" style="font-size:.76rem;line-height:2;margin-top:10px">${esc(photoState.noteFa || '')}</div>
      </div>
    </div>` + statNote('عکس در پوشهٔ دادهٔ سرور نگه داشته می‌شود (نه در باینری برنامه) و با بازآوری نسخه یا به‌روزرسانی سامانه پاک نمی‌شود. اگر عکس بارگذاری نشود، به‌جای آن حرف اول نام نمایش داده می‌شود.');

    const job = kv([
      ['واحد سازمانی', esc(e.departmentName || '—')],
      ['پست سازمانی', esc(e.position || '—')],
      ['نوع قرارداد', esc(e.contractType || '—')],
      ['نوع استخدام', esc(e.employmentTypeFa || e.employmentType || '—')],
      ['وضعیت', empStatusFa(e.status)],
      ['مدیر مستقیم', esc(e.managerName || '—')],
      ['تاریخ استخدام', `<span class="num">${esc(e.hireDate || '—')}</span>`],
      ['حقوق پایه', `<span class="num">${money(e.baseSalary || 0)}</span> ریال`],
      ['گروه شغلی', esc(e.jobGradeFa || e.jobGrade || '—')],
      ['محل سکونت (مبنای فوقالعادهها)', esc(e.residenceSiteFa || e.residenceSite || '—')],
      ['مانده مرخصی', `<span class="num">${fa(e.leaveBalance || 0)}</span> روز`],
    ]) + `<div class="row-actions" style="margin-top:10px">
      <button class="btn btn-ghost" onclick="liveOpenDoc('salary_decree','${esc(e.id)}')">📜 حکم کارگزینی</button>
      <button class="btn btn-ghost" onclick="liveOpenDoc('employment_certificate','${esc(e.id)}')">📄 گواهی اشتغال</button>
      <button class="btn btn-ghost" onclick="liveOpenDoc('payslip','${esc(e.id)}')">🧾 فیش حقوقی</button>
    </div>`;

    const docRows = contracts.map(c => `<tr>
      <td class="num ltr">${esc(c.id)}</td>
      <td>${pChip(c.kind === 'permanent' ? 'c-ok' : 'c-wait', c.kindFa || c.kind)}</td>
      <td>${faDate(c.startDate)}</td>
      <td>${faDate(c.endDate)}</td>
      <td>${statusChip(c.status, c.statusFa)}</td>
      <td><button class="btn btn-sm btn-ghost" onclick="liveOpenDoc('contract','${esc(c.id)}')">🖨️ متن قرارداد</button></td></tr>`);
    const docs = liveTable(['شماره', 'نوع', 'شروع', 'پایان', 'وضعیت', 'سند'], docRows, docRows.length) +
      (contracts.length ? '' : statNote('قراردادی برای این کارمند ثبت نشده است؛ قرارداد تازه از ماژول «قراردادهای کارکنان» افزوده می‌شود.'));

    const kindFa = {attendanceRecords: 'تردد روزانه', leaveRequests: 'مرخصی', missions: 'مأموریت', overtime: 'اضافه‌کاری'};
    const timeRows = attItems.map(r => `<tr>
      <td class="num ltr">${esc(r.id)}</td>
      <td>${esc(kindFa[r.kind] || r.kind)}</td>
      <td>${esc(r.title || '—')}</td>
      <td style="font-size:.74rem">${Object.keys(r.fields || {}).map(k =>
        `<span class="muted">${esc((fieldLabels[r.kind] || {})[k] || k)}: ${esc((r.fields || {})[k])}</span>`).join(' · ') || '—'}</td>
      <td>${statusChip(r.status, r.statusFa)}</td></tr>`);
    const time = liveTable(['شناسه', 'نوع', 'عنوان', 'فیلدها', 'وضعیت'], timeRows, timeRows.length) +
      statNote('این رکوردها از ماژول «حضور و غیاب، مرخصی و مأموریت» خوانده می‌شود و مبنای کارکرد دوره و محاسبهٔ حقوق است.');

    // سوابق: هر سند جاری و پیشین پرونده با تاریخ مؤثر، اقلام کلیدی و سابقهٔ تغییرات خودش.
    // ماژول‌ها و برچسب‌ها همه از پاسخ سرور می‌آید و هیچ‌چیز این‌جا ساخته نمی‌شود.
    const hist = (histRes.ok && histRes.data) || {};
    const histItems = hist.items || [];
    if (!histRes.ok) {
      var history = liveErrorBox(histRes.message || 'خواندن سوابق پرونده ناموفق بود.');
    } else {
      const byModuleFa = {};
      for (const h of histItems) {
        const name = h.moduleTitleFa || h.module;
        byModuleFa[name] = (byModuleFa[name] || 0) + 1;
      }
      const histStats = histItems.length ? `<div class="grid-stats">
        ${Object.entries(byModuleFa).slice(0, 4).map(([name, n]) =>
          statCard({ ico: '🗂️', tint: '#EFF6FF', accent: '#2563EB', val: fa(n), label: name })).join('')}
        ${statCard({ ico: '🕘', tint: '#EDE9FE', accent: '#7C3AED', val: fa(hist.total || histItems.length), label: 'کل سوابق' })}
      </div>` : '';
      const histRows = histItems.map(h => `<tr>
        <td class="num">${faDate(h.date)}${h.dateLabelFa ? `<div class="muted" style="font-size:.66rem">${esc(h.dateLabelFa)}</div>` : ''}</td>
        <td>${esc(h.moduleTitleFa || h.module)}${h.kindTitleFa ? `<div class="muted" style="font-size:.68rem">${esc(h.kindTitleFa)}</div>` : ''}</td>
        <td>${esc(h.title || '—')}${h.latest ? ' ' + pChip('c-ok', 'سند جاری') : ''}
          ${h.recordId ? `<div class="muted ltr" style="font-size:.66rem">${esc(h.recordId)}</div>` : ''}</td>
        <td style="font-size:.74rem">${(h.fields || []).map(f =>
          `<span class="muted">${esc(f.labelFa || f.key)}: ${esc(f.value)}</span>`).join(' · ') || '—'}
          ${h.noteFa ? `<div class="muted" style="font-size:.68rem">${esc(h.noteFa)}</div>` : ''}</td>
        <td>${statusChip(h.status, h.statusFa)}</td>
        <td style="font-size:.72rem">${(h.changes || []).map(c =>
          `<div>${faDate(c.at)} — ${esc(c.actionFa || c.action || '—')}${c.actor ? ' — ' + esc(c.actor) : ''}${c.noteFa ? `<div class="muted">${esc(c.noteFa)}</div>` : ''}</div>`).join('') || '<span class="muted">—</span>'}</td>
      </tr>`);
      var history = histStats + liveTable(['تاریخ مؤثر', 'ماژول / نوع سند', 'سند', 'اقلام کلیدی', 'وضعیت', 'سابقهٔ تغییرات'], histRows, histItems.length) +
        (histItems.length ? '' : statNote('سابقه‌ای برای این کارمند ثبت نشده است.')) +
        ((hist.warnings || []).length ? statNote(hist.warnings.join(' ')) : '') +
        statNote(hist.noteFa || 'سوابق از ماژول‌هایی خوانده می‌شود که نقش شما اجازهٔ خواندنشان را دارد.');
    }

    // تب تحصیلات فقط وقتی ساخته می‌شود که کاربر روی آن باشد (خواندن جداگانه از سرور).
    var education = '';
    if (profileLiveTab === 'education') {
      education = await educationTabHTML(e, id);
    }

    const bodies = {personal, job, education, photo, docs, time, history};
    const body = `<div class="card"><div class="card-pad">${tabBar}${bodies[profileLiveTab] || personal}</div></div>`;

    return head + body;
  });

  return liveShell('پرونده کارمند', 'پروندهٔ تبیندی: مشخصات، عکس پرسنلی، قرارداد و اسناد، حضور و کارکرد، سوابق تغییرات',
    `<button class="btn btn-ghost" onclick="go('employees')">‹ فهرست کارکنان</button>`);
}

/* ------------------------- عکس کارگزینی پرونده ------------------------- */

/**
 * بارگذاری عکس کارگزینی از فرم پرونده (multipart با هدر احراز هویت).
 *
 * چرا fetch خام: FormData نباید Content-Type دستی بگیرد و هدر Authorization لازم است.
 * سنجش نوع/حجم فایل در سرور انجام می‌شود و پیام فارسی همان‌جا ساخته می‌شود.
 */
async function empPhotoUpload(employeeId) {
  const input = document.getElementById('empPhotoFile');
  if (!input || !input.files || !input.files.length) {
    toast('اول یک فایل عکس انتخاب کنید.', 'error');
    return;
  }
  const body = new FormData();
  body.append('file', input.files[0]);
  const headers = {};
  if (serverSession && serverSession.accessToken) headers['Authorization'] = 'Bearer ' + serverSession.accessToken;
  try {
    const res = await fetch('/api/v1/employee/employees/' + encodeURIComponent(employeeId) + '/photo', {method: 'POST', headers, body});
    const data = await res.json().catch(() => null);
    if (!res.ok) { toast(jsonMessage(res.status, data), 'error'); return; }
    toast('عکس کارگزینی ذخیره شد و در پرونده و ساختار سازمانی دیده می‌شود.');
    liveReload();
  } catch (_) {
    toast('ارتباط با سرور برقرار نشد.', 'error');
  }
}

/** حذف عکس کارگزینی کارمند. */
async function empPhotoDelete(employeeId) {
  if (!confirm('عکس کارگزینی این کارمند حذف شود؟')) return;
  const res = await apiFetch('/api/v1/employee/employees/' + encodeURIComponent(employeeId) + '/photo', {method: 'DELETE'});
  if (!res.ok) { toast(res.message, 'error'); return; }
  toast('عکس کارگزینی حذف شد.');
  liveReload();
}

LIVE_PAGES.employees = pageEmployeesLive;
LIVE_PAGES.profile = pageProfileLive;
