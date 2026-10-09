/* =====================================================================
   هسته UI — کمکی‌ها، RBAC، منوها، ورود/خروج، مسیریابی
   ===================================================================== */

const $  = s => document.querySelector(s);
const $$ = s => document.querySelectorAll(s);
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const fa = n => String(n).replace(/\d/g, d => '۰۱۲۳۴۵۶۷۸۹'[d]);
const money = n => fa(Number(n).toLocaleString('en-US'));
const initial = name => name.split(' ')[0][0];

/* ---------------------------------------------------------------------
   یکدست‌سازی ارقام فارسی
   ---------------------------------------------------------------------
   قالب‌های صفحه در ده‌ها جا عدد لاتین می‌گذارند (تاریخ «1404/07/14»، مبلغ،
   شمارهٔ تماس، تعداد). اگر این‌ها یکی‌یکی دستی به `fa()` سپرده نشوند، هر بار که
   بخشی از صفحه دوباره ساخته می‌شود ارقام فارسی و لاتین قاطی می‌شوند و نمایش
   فارسی ناموزون می‌شود. این ناظرِ DOM، هر متنی که تازه به صفحه می‌رسد را بدون
   نیاز به تغییر تک‌تک قالب‌ها فارسی می‌کند — یک‌بار برای همیشه.

   سه استنثنا دارد:
   • نواحی لاتین: `input, textarea, select, code, pre، .ltr` و هر گره با
     `data-digits="latin"` دست نمی‌خورد (نام کاربری، ایمیل، کد و شناسه).
   • متنی که حرف لاتین دارد (مثل `EMP-001` یا `07:58` یا ایمیل) شناسه است و
     لاتین می‌ماند؛ فقط متنِ بدون حرف لاتین به فارسی ترجمه می‌شود.
   • ترجمه فقط وقتی می‌نویسد که نتیجه متفاوت باشد، پس حلقهٔ بی‌پایان نمی‌سازد. */
const DIGITS_LATIN_SELECTOR = 'script,style,noscript,input,textarea,select,option,code,pre,.ltr,[data-digits="latin"]';

function faDigitsTextNode(node) {
  if (!node || node.nodeType !== 3) return;      // فقط گرهٔ متنی
  const v = node.nodeValue;
  if (!v || !/[0-9]/.test(v)) return;
  // واژه‌به‌واژه: واژه‌ای که حرف لاتین دارد شناسه است (EMP-004، DEC-1405-004،
  // نام کاربری) و دست‌نخورده می‌ماند؛ بقیهٔ واژه‌ها فارسی می‌شوند. معیار نمی‌تواند
  // روی کل گره باشد چون در یک گره ممکن است شناسه و تاریخ پشت سر هم بیایند.
  const out = v.replace(/\S+/g, w => /[A-Za-z]/.test(w) ? w : fa(w));
  if (out !== v) node.nodeValue = out;
}

function faDigitsIn(root) {
  if (!root) return;
  if (root.nodeType === 3) { faDigitsTextNode(root); return; }
  if (root.nodeType !== 1) return;
  if (root.matches && root.matches(DIGITS_LATIN_SELECTOR)) return;
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode: n => (n.parentElement && n.parentElement.closest(DIGITS_LATIN_SELECTOR))
      ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT,
  });
  let n;
  while ((n = walker.nextNode())) faDigitsTextNode(n);
}

/** ناظر DOM را روی صفحهٔ زنده می‌گذارد تا هر محتوای تازه یکدست فارسی شود. */
function initFaDigits() {
  faDigitsIn(document.body);
  const obs = new MutationObserver(muts => {
    for (const m of muts) {
      if (m.type === 'characterData') { faDigitsTextNode(m.target); continue; }
      m.addedNodes.forEach(n => faDigitsIn(n));
    }
  });
  obs.observe(document.body, { childList: true, subtree: true, characterData: true });
}

const ROLE_NAMES = {
  admin:'مدیر سیستم', hr_manager:'مدیر منابع انسانی', hr_spec:'کارشناس منابع انسانی',
  line_manager:'مدیر واحد', employee:'کارمند'
};

/* ---------------------------------------------------------------------
   ادارات، منو و دسترسی‌ها — همه از فایل تولیدشدهٔ `js/menu.generated.js`
   می‌آید که خودش از `contracts/departments.json` و مانیفست ماژول‌ها ساخته شده
   است. در این فایل هیچ دسترسی‌ای دستی تعریف نمی‌شود تا منوی نسخهٔ نمایشی و
   نسخهٔ نهایی نتوانند از هم جدا بیفتند.
   --------------------------------------------------------------------- */
const DIRECTORATES = window.HRMS_MENU || [];

/* ---------------------------------------------------------------------
   استقرار پارامتریک شرکت‌ها — از `js/organizations.generated.js`.
   با انتخاب هر شرکت، ماژول‌های غیرفعال همان شرکت از منو حذف می‌شوند و پارامترهای
   حقوقی-مالی‌اش در صفحهٔ پروفایل شرکت دیده می‌شود؛ همین رفتار در سرور Go هم هست
   (پرچم `-org`)، پس نمونهٔ نمایشی و استقرار واقعی یک سیاست دارند.
   --------------------------------------------------------------------- */
const ORG_DATA = window.HRMS_ORGS || {active: null, organizations: []};
let activeOrg = ORG_DATA.organizations.find(o => o.id === ORG_DATA.active) || ORG_DATA.organizations[0] || null;

/** آیا این ماژول در استقرار شرکت فعال جاری روشن است. */
function moduleEnabled(id) {
  return !activeOrg || activeOrg.enabledModules.includes(id);
}

/** پارامترهای شرکت فعال بر پایهٔ کلید (برای صفحه‌ها و بهای محاسبه). */
function orgParam(key, fallback = '—') {
  if (!activeOrg) return fallback;
  const p = activeOrg.params.find(x => x.key === key);
  return p ? `${p.value} ${p.unitFa}` : fallback;
}

/** ماژول‌های فعال به ترتیب منو — برای گروه‌بندی و آمار بالای صفحه. */
function activeModules() {
  const out = [];
  for (const dir of DIRECTORATES)
    for (const dept of dir.departments || [])
      for (const g of dept.groups || [])
        for (const m of g.modules || []) if (moduleEnabled(m.id)) out.push({...m, dept: dept.id});
  return out;
}

/** نگاشت نام آیکون مانیفست به نماد نمایشی — فقط تزئینی، بدون منطق. */
const ICONS = {
  gauge:'🏠', users:'👥', 'id-card':'🪪', 'folder-open':'📂', clock:'🕐', 'calendar-off':'🌤️',
  'check-circle':'✅', timer:'⏱️', calendar:'📅', 'play-circle':'⚙️', receipt:'🧾', sliders:'🧮',
  'bar-chart':'📊', gift:'🎁', 'hand-helping':'📮', sitemap:'🏢', 'id-badge':'🪪', stamp:'📜',
  'trending-up':'📈', route:'🧭', 'file-plus':'📝', megaphone:'📢', 'user-search':'🧑‍💼', workflow:'🔀',
  repeat:'🔁', 'list-checks':'📋', award:'⭐', gavel:'⚖️', 'users-round':'👥', 'heart-handshake':'🤝',
  'hand-coins':'💳', stethoscope:'🩺', 'clipboard-plus':'🧪', 'heart-pulse':'💗', map:'🗺️', target:'🎯',
  presentation:'🎤', 'book-open':'🎓', scroll:'📜', 'file-text':'📑', 'user-cog':'🔑', shield:'🛡️',
  building:'🏢', settings:'🧰', history:'🕵️',
  // افزوده‌های واحدهای تازه: بایگانی، طبقه‌بندی مشاغل، وام، رفاه، آمار و برون‌سپاری
  bus:'🚌', ticket:'🎫', bed:'🛏️', layers:'🧱', hourglass:'⌛', database:'🗄️',
  percent:'٪', flag:'🏴', pin:'📍', comment:'💬', briefcase:'💼', lightbulb:'💡',
  store:'🏬', book:'📘', scale:'⚖️', search:'🔍', link:'🔗', calendarCheck:'🗓️',
};
const icon = name => ICONS[name] || '•';

/** صفحه‌های دارای نشان شمارشی در منو. */
const PILLS = { leaveApprove: () => LEAVES.filter(l => l.status === 'pending').length };

/** دسترسی هر نقش به هر صفحه — کلیدشده روی کلید صفحه برای بررسی در مسیریابی. */
/**
 * دسترسی هر نقش به هر صفحه — کلیدشده روی کلید صفحه برای بررسی در مسیریابی.
 *
 * دو سیاست در همین‌جا به هم می‌رسند: دسترسی نقش (از مانیفست ماژول) و فعال بودن ماژول
 * در استقرار شرکت (از پروفایل پارامتریک). صفحه‌ای که ماژولش در شرکت فعال جاری خاموش
 * است، در این نگاشت وجود ندارد؛ پس نه در منو می‌آید و نه با مسیر دستی باز می‌شود.
 */
let PAGE_ACCESS = {};
function buildPageAccess() {
  const map = {};
  for (const dir of DIRECTORATES)
    for (const dept of dir.departments || [])
      for (const g of dept.groups || [])
        for (const mod of g.modules || []) {
          if (!moduleEnabled(mod.id)) continue;
          for (const s of mod.screens) map[s.page] = s.roles;
        }
  PAGE_ACCESS = map;
  return map;
}
buildPageAccess();

let session = null; // {username, name, role}
let currentPage = 'dashboard';
let notifs = [...NOTIFS];/* ------------------- باز/بسته کردن ماژول‌ها در منو ------------------- */
const collapsedModules = new Set();
/* اداره‌های بازوبستهشدهٔ منو؛ پیش‌فرض باز است تا کاربر ساختار کامل را ببیند. */
const collapsedDepartments = new Set();

function toggleModule(id) {
  if (collapsedModules.has(id)) collapsedModules.delete(id);
  else collapsedModules.add(id);
  renderSidebar();
}

/* ------------------- ورود / خروج ------------------- */
const LOGIN_USERS = {
  'hr.manager':{role:'hr_manager',   name:'سارا محمدی', emp:'EMP-001'},
  'hr.spec':{role:'hr_spec',       name:'مریم نوری',  emp:'EMP-008'},
  'line.mgr':{role:'line_manager', name:'محمد احمدی', emp:'EMP-005'},
  'employee':{role:'employee',     name:'نگار حسینی', emp:'EMP-003'},
  'admin':{role:'admin',          name:'مدیر سیستم', emp:null},
};

/* وضعیت اتصال به سرور محلی: null=بررسی‌نشده، true=وصل، false=فقط حالت نمایشی. */
let serverOnline = null;
let serverSession = null;

/** آغاز نشست و آماده‌سازی پوسته. */
function startSession(sess) {
  session = sess;
  notifs = [...NOTIFS];
  $('#loginErr').classList.remove('show');
  $('#loginPage').classList.add('hidden');
  $('#app').classList.remove('hidden');
  $('#ubName').textContent = sess.name;
  $('#ubRole').textContent = ROLE_NAMES[sess.role] || sess.role;
  $('#ubAvatar').textContent = initial(sess.name);
  $('#notifDot').textContent = fa(notifs.filter(n=>n.unread).length);
  go('dashboard');
}

/**
 * ورود دو مرحله‌ای:
 * ۱. اگر سرور محلی Go بالا باشد، احراز هویت واقعی با همان حساب‌های نمونه انجام
 *    می‌شود و توکن دسترسی برای درخواست‌های واقعی نگه داشته می‌شود.
 * ۲. اگر سروری در کار نباشد (نمونهٔ استاتیک)، همان حساب‌های نمایشی کار می‌کنند.
 * این کار عمدی است: نمونهٔ رابط بدون سرور هم قابل ارائه است، و با سرور هم به همان
 * API واقعی وصل می‌شود.
 */
async function tryLogin() {
  const u = $('#loginUser').value.trim().toLowerCase();
  const p = $('#loginPass').value;
  const btn = $('#loginBtn');
  if (btn) btn.disabled = true;
  try {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({username: u, password: p}),
    });
    if (res.ok) {
      const s = await res.json();
      serverOnline = true;
      serverSession = s;
      startSession({username:u, name:s.user.displayName, role:s.user.role,
        emp:s.user.employeeId || null, token:s.accessToken});
      toast('ورود به سرور محلی انجام شد');
      return;
    }
    if (res.status === 401) {
      serverOnline = true;
      $('#loginErr').classList.add('show');
      return;
    }
  } catch (_) {
    /* سرور در دسترس نیست؛ ادامه با حالت نمایشی */
  } finally {
    if (btn) btn.disabled = false;
  }

  serverOnline = false;
  const rec = LOGIN_USERS[u];
  if (!rec || p !== 'demo') { $('#loginErr').classList.add('show'); return; }
  startSession({username:u, name:rec.name, role:rec.role, emp:rec.emp, token:null});
}
function logout() {
  session = null; currentPage = 'dashboard';
  $('#app').classList.add('hidden');
  $('#loginPage').classList.remove('hidden');
  $('#loginPass').value = '';
  $('#loginUser').value = '';
  $('#userMenu').classList.add('hidden');
}

/* ------------------- مسیریابی ------------------- */
function go(page, ...args) {
  if (!session) return;
  if (!PAGE_ACCESS[page] || !PAGE_ACCESS[page].includes(session.role)) {
    toast('شما به این صفحه دسترسی ندارید', 'error');
    return;
    }
  currentPage = page;
  closeOverlays();
  renderSidebar();
  renderPage(page, ...args);
  $('#main').scrollTop = 0;
  window.scrollTo(0, 0);
  $('#sidebar').classList.remove('open');
}

/** اداره‌های فعال یک حوزه: ماژول‌های خاموش شرکت حذف و صفحه‌های بی‌دسترسی پالایش می‌شوند. */
function activeDepartments(dir) {
  return (dir.departments || []).map(dept => ({
    ...dept,
    groups: (dept.groups || [])
      .map(g => ({
        ...g,
        modules: g.modules
          .filter(m => moduleEnabled(m.id))
          .map(m => ({...m, screens: m.screens.filter(s => s.roles.includes(session.role))}))
          .filter(m => m.screens.length),
      }))
      .filter(g => g.modules.length),
  })).filter(dept => dept.groups.length);
}

function renderSidebar() {
  const sb = $('#sidebar');
  let html = `<div class="role-chip">🎖️ ${ROLE_NAMES[session.role]}</div>`;
  if (!DIRECTORATES.length) {
    html += `<div class="nav-group-title">منو در دسترس نیست</div>
      <div class="muted" style="font-size:.75rem;padding:0 12px;line-height:1.9">
        فایل <span class="ltr">js/menu.generated.js</span> بارگذاری نشده است.
        دستور <span class="ltr">npm run gen</span> را اجرا کنید.
      </div>`;
  }
  for (const dir of DIRECTORATES) {
    const depts = activeDepartments(dir);
    if (!depts.length) continue;
    html += `<div class="nav-directorate" title="${esc((dir.dutiesFa||[]).join(' · '))}">${esc(dir.titleFa)}</div>`;
    for (const dept of depts) {
      const deptOpen = !collapsedDepartments.has(dept.id);
      html += `<button class="nav-group-title as-btn ${deptOpen?'open':''}" onclick="toggleDepartment('${dept.id}')"
        title="${esc((dept.dutiesFa||[]).join(' · '))}">
        <span class="caret">${deptOpen?'▾':'▸'}</span>${esc(dept.titleFa)}</button>`;
      if (!deptOpen) continue;
      for (const g of dept.groups) {
        // زیرواحد فقط یک برچسب گروه‌بندی است؛ خودش منطق یا دسترسی ندارد.
        if (g.titleFa) html += `<div class="nav-subunit" title="${esc((g.dutiesFa||[]).join(' · '))}">${esc(g.titleFa)}</div>`;
        for (const m of g.modules) {
          const open = !collapsedModules.has(m.id);
          html += `<button class="nav-mod ${open?'open':''}" onclick="toggleModule('${m.id}')">
            <span class="caret">${open?'▾':'▸'}</span>
            <span class="nav-mod-name">${esc(m.titleFa)}</span>
            ${m.shared?`<span class="chip c-shared">مشترک</span>`:''}
            <span class="muted num" style="font-size:.68rem">${fa(m.screens.length)}</span>
          </button>`;
          if (!open) continue;
          for (const s of m.screens) {
            const pill = PILLS[s.page] ? PILLS[s.page]() : 0;
            html += `<a class="nav-item ${currentPage===s.page?'active':''}" onclick="go('${s.page}')">
              <span class="ico">${icon(s.icon)}</span>${esc(s.label)}
              ${pill?`<span class="pill num">${fa(pill)}</span>`:''}
            </a>`;
          }
        }
      }
    }
  }
  const conn = serverOnline === true
    ? `<span class="chip c-ok">متصل به سرور محلی</span>`
    : serverOnline === false
      ? `<span class="chip c-wait">حالت نمایشی — بدون سرور</span>`
      : '';
  const org = activeOrg
    ? `<div class="sidebar-org"><div style="display:flex;justify-content:center;margin-bottom:6px">${brandMark(88, 50, 12)}</div>
        <b>${esc(activeOrg.titleFa)}</b><br>
        <span class="muted">${esc(activeOrg.holdingFa)} — ${esc(activeOrg.shortCode)}</span></div>`
    : '';
  html += `${org}<div class="sidebar-foot">سامانه جامع منابع انسانی — نسخه نمایشی<br>${conn}</div>`;
  sb.innerHTML = html;
}

/* ------------------- استقرار پارامتریک: انتخاب شرکت ------------------- */

/**
 * نشان شرکت فعال به‌صورت HTML.
 *
 * اگر شرکت فایل نشان داشته باشد (مثل آرم هلدینگ خلیج فارس برای پتروشیمی بندر امام)
 * همان تصویر نشان داده می‌شود؛ وگرنه نشانهٔ ایموجی پروفایل شرکت. هیچ آدرسی اینجا
 * ساخته نمی‌شود؛ مسیر از پروفایل پارامتریک شرکت‌ها می‌آید.
 */
function brandMark(w = 38, h = 38, radius = 10) {
  const o = activeOrg;
  if (o && o.logoFile) {
    return `<img class="brand-img" src="${esc(o.logoFile)}" alt="${esc(o.holdingFa || o.titleFa)}"
      style="width:${w}px;height:${h}px;border-radius:${radius}px">`;
  }
  return `<span class="brand-emoji" style="width:${w}px;height:${h}px;font-size:${Math.round(Math.min(w, h) * 0.55)}px">${esc(o ? o.logoEmoji : '👥')}</span>`;
}

/** نشان شرکت فعال را در سرصفحه، صفحهٔ ورود و عنوان سند می‌نشاند. */
function applyOrgBranding() {
  if (!activeOrg) return;
  const setText = (sel, txt) => { const el = $(sel); if (el) el.textContent = txt; };
  setText('#brandOrg', `${activeOrg.titleFa} — ${activeOrg.holdingFa}`);
  setText('#loginOrg', `${activeOrg.titleFa} (${activeOrg.holdingFa}) — ورود به حساب کاربری`);
  document.title = `سامانه جامع منابع انسانی — ${activeOrg.titleFa}`;
  // رنگ برند شرکت روی متغیر CSS می‌نشیند تا سرصفحه/دکمه‌ها با هویت شرکت هم‌رنگ شوند.
  applyAccent(); // رنگ تأکیدی کاربر یا رنگ برند شرکت
  // نشان شرکت در سرصفحه، صفحهٔ ورود و عنوان‌ها. اگر فایل نشان باشد، جعبهٔ مربعی
  // برداشته می‌شود تا آرم پهنِ رسمی بدون فشردگی دیده شود.
  const hasImg = !!activeOrg.logoFile;
  const topLogo = $('#brandLogo');
  if (topLogo) {
    topLogo.classList.toggle('has-img', hasImg);
    topLogo.innerHTML = brandMark(104, 40, 10);
  }
  const loginLogo = $('#loginLogo');
  if (loginLogo) {
    loginLogo.classList.toggle('has-img', hasImg);
    loginLogo.innerHTML = brandMark(184, 64, 14);
  }
}

/** پر کردن فهرست شرکت‌ها در سرصفحه. */
function initOrgSwitch() {
  const sel = $('#orgSwitch');
  if (!sel) return;
  if (!ORG_DATA.organizations.length) { sel.classList.add('hidden'); return; }
  sel.innerHTML = ORG_DATA.organizations
    .map(o => `<option value="${o.id}" ${activeOrg && o.id === activeOrg.id ? 'selected' : ''}>
      ${o.logoEmoji} ${o.titleFa} — ${o.shortCode}</option>`).join('');
  sel.onchange = () => switchOrg(sel.value);
}

/**
 * تغییر شرکت فعال. هیچ داده‌ای از سرور دوباره خوانده نمی‌شود چون نمونهٔ مرورگر
 * استاتیک است؛ اما همین انتخاب، دقیقاً همان چیزی را عوض می‌کند که پرچم `-org` در
 * استقرار واقعی عوضمی‌کند: فعال بودن ماژول‌ها و پارامترهای حقوقی-مالی.
 */
function switchOrg(id) {
  const org = ORG_DATA.organizations.find(o => o.id === id);
  if (!org) return;
  activeOrg = org;
  try { localStorage.setItem(ORG_STORAGE_KEY, org.id); } catch (_) { /* حالت‌های محدود مرورگر */ }
  buildPageAccess();
  applyOrgBranding();
  renderSidebar();
  toast(`استقرار به «${org.titleFa}» تغییر کرد — ${fa(org.enabledModules.length)} ماژول فعال`);
  const allowed = PAGE_ACCESS[currentPage] && PAGE_ACCESS[currentPage].includes(session.role);
  if (!allowed) go('dashboard');
  else renderPage(currentPage);
}

/* شرکت فعال بر پایهٔ ادراک کاربر ذخیره می‌شود تا پس از بازخوانی صفحه هم بماند. */
const ORG_STORAGE_KEY = 'hrms.activeOrg';
try {
  const saved = localStorage.getItem(ORG_STORAGE_KEY);
  const org = ORG_DATA.organizations.find(o => o.id === saved);
  if (org) activeOrg = org;
} catch (_) { /* حالت‌های محدود مرورگر */ }

function toggleDepartment(id) {
  if (collapsedDepartments.has(id)) collapsedDepartments.delete(id);
  else collapsedDepartments.add(id);
  renderSidebar();
}

/* ------------------- توست، مودال، اعلان‌ها، تم ------------------- */
function toast(msg, type='success') {
  const t = document.createElement('div');
  t.className = 'toast' + (type==='error'?' error':type==='info'?' info':'');
  t.innerHTML = `${type==='error'?'⛔':type==='info'?'ℹ️':'✅'} ${esc(msg)}`;
  $('#toastRoot').appendChild(t);
  setTimeout(()=>t.remove(), 3200);
}

let modalStack = [];
function openModal({title, body, foot, narrow}) {
  const back = document.createElement('div');
  back.className = 'modal-back';
  back.onclick = e => { if (e.target === back) closeModal(); };
  back.innerHTML = `<div class="modal ${narrow?'narrow':''}">
    <div class="modal-head"><div class="modal-title">${title}</div>
      <button class="modal-x" onclick="closeModal()">✕</button></div>
    <div class="modal-body">${body}</div>
    ${foot?`<div class="modal-foot">${foot}</div>`:''}
  </div>`;
  $('#modalRoot').appendChild(back);
  modalStack.push(back);
  return back;
}
function closeModal() {
  const m = modalStack.pop();
  if (m) m.remove();
}
function closeOverlays() {
  while (modalStack.length) closeModal();
  $('#userMenu') && $('#userMenu').classList.add('hidden');
  const np = $('#notifPanel'); np && np.remove();
  $('#sidebar').classList.remove('open');
}

function openNotif(e) {
  e && e.stopPropagation();
  const old = $('#notifPanel'); old && old.remove();
  const p = document.createElement('div');
  p.id = 'notifPanel';
  p.className = 'notif-panel';
  p.innerHTML = notifs.length
    ? notifs.map((n,i)=>`<div class="notif-item ${n.unread?'unread':''}" onclick="readNotif(${i})">
        <div class="n-ico">${n.ico}</div>
        <div>${esc(n.title)}<time>${n.time}</time></div>
      </div>`).join('')
    : '<div class="empty-state" style="padding:24px">اعلانی وجود ندارد</div>';
  $('.userbox').appendChild(p);
  p.onclick = e => e.stopPropagation();
}
function readNotif(i) {
  const n = notifs[i];
  if (n) { n.unread = false; $('#notifDot').textContent = fa(notifs.filter(x=>x.unread).length); }
  if (n && n.page && PAGE_ACCESS[n.page] && PAGE_ACCESS[n.page].includes(session.role)) go(n.page);
  else closeModal();
  const np = $('#notifPanel'); np && np.remove();
}
/* ------------------- ظاهر برنامه: تم، چینش، رنگ تأکیدی، چگالی، گردی ------------------- */
const APPEAR_STORAGE_KEY = 'hrms.appearance';
const ACCENT_PRESETS = [
  { key: 'auto',   label: 'رنگ سازمان', color: '' },
  { key: 'blue',   label: 'آبی',      color: '#2563EB' },
  { key: 'teal',   label: 'فیروزه‌ای', color: '#0D9488' },
  { key: 'green',  label: 'سبز',      color: '#16A34A' },
  { key: 'purple', label: 'بنفش',     color: '#7C3AED' },
  { key: 'red',    label: 'قرمز',     color: '#DC2626' },
  { key: 'amber',  label: 'کهربایی',  color: '#D97706' },
  { key: 'slate',  label: 'خاکستری',  color: '#475569' },
];
const APPEAR_DEFAULTS = { theme: 'light', dir: 'rtl', accent: 'auto', density: 'normal', radius: 'normal' };
let appearance = { ...APPEAR_DEFAULTS };

/** ترجیحات ظاهر را از حافظهٔ مرورگر می‌خواند. */
function loadAppearance() {
  try {
    const saved = JSON.parse(localStorage.getItem(APPEAR_STORAGE_KEY) || '{}');
    appearance = { ...APPEAR_DEFAULTS, ...saved };
  } catch (_) { appearance = { ...APPEAR_DEFAULTS }; }
}
/** ترجیحات ظاهر را ذخیره می‌کند تا پس از بازخوانی صفحه بماند. */
function saveAppearance() {
  try { localStorage.setItem(APPEAR_STORAGE_KEY, JSON.stringify(appearance)); } catch (_) {}
}

/** رنگ تأکیدی فعلی: انتخاب کاربر، وگرنه رنگ برند سازمان فعال. */
function accentColor() {
  if (appearance.accent && appearance.accent !== 'auto') {
    const p = ACCENT_PRESETS.find(x => x.key === appearance.accent);
    if (p && p.color) return p.color;
  }
  return (activeOrg && activeOrg.brandColor) || '';
}
/** رنگ تأکیدی را روی متغیر CSS می‌نشاند (یا به رنگ پیش‌فرض سازمان برمی‌گرداند). */
function applyAccent() {
  const root = document.documentElement;
  const c = accentColor();
  if (c) root.style.setProperty('--primary', c);
  else root.style.removeProperty('--primary');
}

/** همهٔ ترجیحات ظاهر را روی صفحه اعمال و دکمه‌ها را هم‌گام می‌کند. */
function applyAppearance() {
  const root = document.documentElement;
  document.body.classList.toggle('dark', appearance.theme === 'dark');
  root.dir = appearance.dir === 'ltr' ? 'ltr' : 'rtl';
  [document.documentElement, document.body].forEach(el => {
    el.classList.remove('density-compact', 'density-comfy', 'radius-soft', 'radius-square');
    if (appearance.density === 'compact') el.classList.add('density-compact');
    if (appearance.density === 'comfy') el.classList.add('density-comfy');
    if (appearance.radius === 'soft') el.classList.add('radius-soft');
    if (appearance.radius === 'square') el.classList.add('radius-square');
  });
  applyAccent();
  syncAppearanceUI();
}

/** چرخه‌های رنگ تأکیدی را در پنل می‌سازد. */
function renderSwatches() {
  const box = $('#accentSwatches');
  if (!box) return;
  box.innerHTML = ACCENT_PRESETS.map(p => {
    const bg = p.color ? p.color : 'linear-gradient(135deg,#2563EB,#0D9488)';
    return `<button data-k="${p.key}" title="${esc(p.label)}" aria-label="${esc(p.label)}" style="background:${bg}" onclick="setAccent('${p.key}')"></button>`;
  }).join('');
}

/** وضعیت «فعال» را روی بخش‌ها و چرخه‌های پنل علامت می‌زند. */
function syncAppearanceUI() {
  const mark = (id, v) => {
    const box = $(id);
    if (box) box.querySelectorAll('button').forEach(b => b.classList.toggle('on', b.dataset.v === v));
  };
  mark('#segTheme', appearance.theme);
  mark('#segDir', appearance.dir);
  mark('#segDensity', appearance.density);
  mark('#segRadius', appearance.radius);
  const box = $('#accentSwatches');
  if (box) box.querySelectorAll('button').forEach(b => b.classList.toggle('on', b.dataset.k === appearance.accent));
}

/** باز/بستن پنجرهٔ ظاهر. */
function toggleAppearance(e) {
  if (e) e.stopPropagation();
  const p = $('#appearPanel');
  if (!p) return;
  const opening = p.classList.contains('hidden');
  p.classList.toggle('hidden');
  if (opening) { renderSwatches(); syncAppearanceUI(); }
}

function setTheme(v)    { appearance.theme = v;    saveAppearance(); applyAppearance(); }
function setDirection(v){ appearance.dir = v;      saveAppearance(); applyAppearance(); }
function setDensity(v)  { appearance.density = v;  saveAppearance(); applyAppearance(); }
function setRadius(v)   { appearance.radius = v;   saveAppearance(); applyAppearance(); }
function setAccent(k)   { appearance.accent = k;   saveAppearance(); applyAppearance(); }

/** بازگشت به ظاهر پیش‌فرض (و رنگ برند سازمان فعال). */
function resetAppearance() {
  appearance = { ...APPEAR_DEFAULTS };
  try { localStorage.removeItem(APPEAR_STORAGE_KEY); } catch (_) {}
  applyAppearance();
  toast('ظاهر برنامه به پیش‌فرض بازگشت');
}
function toggleUserMenu(e) {
  e.stopPropagation();
  $('#userMenu').classList.toggle('hidden');
}
document.addEventListener('click', () => {
  $('#userMenu') && $('#userMenu').classList.add('hidden');
  const np = $('#notifPanel'); np && np.remove();
});

function requireConfirm(msg, onYes) {
  openModal({narrow:true, title:'تأیید عملیات', body:`<p style="font-size:.9rem">${msg}</p>`,
    foot:`<button class="btn btn-primary" onclick="confirmYes()">تأیید</button>
          <button class="btn btn-ghost" onclick="closeModal()">انصراف</button>`});
  window._confirmCb = onYes;
}
function confirmYes() { const cb = window._confirmCb; closeModal(); cb && cb(); }

/* ------------------- صفحه‌های مشترک ------------------- */
function pageHead(title, sub, actions='') {
  const crumbs = currentPage === 'dashboard' ? '' :
    `<div class="crumbs">منابع انسانی <span>‹</span> ${title}</div>`;
  return `
  <div class="page-head">
    <div><div class="page-title">${title}</div>${sub?`<div class="page-sub">${sub}</div>`:''}</div>
    <div class="head-actions">${actions}</div>
  </div>${crumbs}`;
}

function statCard({ico, tint, accent, val, label, link, page}) {
  const click = page ? `onclick="go('${page}')"` : '';
  const cursor = page ? 'cursor:pointer' : '';
  return `<div class="stat-card" style="--tint:${tint};--accent:${accent};${cursor}" ${click}>
    <div class="stat-ico" style="background:${tint}">${ico}</div>
    <div><div class="stat-val num">${val}</div><div class="stat-label">${label}</div></div>
    ${link?`<span class="stat-link">${link} ‹</span>`:''}
  </div>`;
}

function tbl(head, rows, {foot=true, count=null}={}) {
  const n = count ?? rows.filter(r=>r).length;
  return `<div class="card">
    <div class="tbl-wrap"><table class="tbl">
      <thead><tr>${head.map((h,i)=>`<th>${h}</th>`).join('')}</tr></thead>
      <tbody>${rows.length?rows.join(''):`<tr><td colspan="${head.length}"><div class="empty-state"><div class="big">📭</div>موردی یافت نشد</div></td></tr>`}</tbody>
    </table></div>
    ${foot?`<div class="tbl-foot"><span class="rec-count">نمایش ${fa(rows.length)} از ${fa(n)} رکورد</span>
      <div class="pager"><button>‹</button><button class="on num">۱</button><button class="num">۲</button><button class="num">۳</button><button>›</button></div>
    </div>`:''}
  </div>`;
}

/* نمودار میله‌ای دوسطری (CSS-محور) — data: [{l, a, b}] */
function barChart(data) {
  const max = Math.max(...data.map(d=>Math.max(d.a,d.b)), 1);
  return `<div style="display:flex;align-items:stretch;gap:10px;height:150px;padding-top:10px">
    ${data.map(d=>`<div style="flex:1;display:flex;flex-direction:column;align-items:center;justify-content:flex-end;gap:4px">
      <div style="display:flex;gap:3px;align-items:flex-end;flex:1;width:100%;justify-content:center">
        <div title="جذب: ${d.a}" style="width:13px;height:${Math.round(d.a/max*100)}%;background:#2563EB;border-radius:4px 4px 0 0"></div>
        <div title="ترک خدمت: ${d.b}" style="width:13px;height:${Math.round(d.b/max*100)}%;background:#DC2626;border-radius:4px 4px 0 0"></div>
      </div>
      <span class="muted" style="font-size:.68rem">${d.l}</span>
    </div>`).join('')}
  </div>`;
}

/* ------------------- راه‌اندازی ------------------- */
window.addEventListener('DOMContentLoaded', () => {
  $('#loginBtn').onclick = tryLogin;
  $('#loginPass').addEventListener('keydown', e => { if (e.key === 'Enter') tryLogin(); });
  $('#loginUser').addEventListener('keydown', e => { if (e.key === 'Enter') tryLogin(); });
  $('#burger').onclick = e => { e.stopPropagation(); $('#sidebar').classList.toggle('open'); };
  initOrgSwitch();
  initFaDigits();
  loadAppearance();
  applyAppearance();
  const ap = $('#appearPanel');
  if (ap) ap.addEventListener('click', e => e.stopPropagation());
  applyOrgBranding();
  $('#globalSearch').addEventListener('keydown', e => {
    if (e.key === 'Enter' && $('#globalSearch').value.trim()) { go('employees', $('#globalSearch').value.trim()); $('#globalSearch').value=''; }
  });
});
