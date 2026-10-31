#!/usr/bin/env node
/**
 * آزمون دودِ (Smoke) نمونهٔ وبِ سامانهٔ منابع انسانی
 * ==================================================
 * این اسکریپت همهٔ صفحه‌های نمونهٔ وب را یکی‌یکی باز می‌کند، خطاهای
 * جاوااسکریپت و «رندر نشدن» محتوا را می‌گیرد و در پایان گزارش فارسی می‌دهد
 * که چه صفحه‌هایی ایراد دارند.
 *
 * چرا بدون وابستگی؟ چون در این محیط نصب بستهٔ npm مطلوب نیست؛ فقط ماژول‌های
 * داخلی Node به‌همراه `fetch` و `WebSocket` سراسری (Node 20+) استفاده می‌شوند.
 *
 * پیش‌نیاز:
 *   • Node >= 20
 *   • یک مرورگر Chromium (Edge یا Chrome) روی سیستم — یا با `--browser` بدهید
 *     یا مسیر را در متغیر محیطی `HRMS_BROWSER` بگذارید.
 *   • سرور نمونهٔ وب بالا باشد (start-HRMS.bat). اگر نباشد، اسکریپت پیش از
 *     هر کاری با fetch بررسی می‌کند و با پیام راهنما خارج می‌شود.
 *
 * اجرا:
 *   node tools/check/smoke-web.mjs
 *   node tools/check/smoke-web.mjs --url http://127.0.0.1:8099/ --user hr.manager --pass demo
 *   node tools/check/smoke-web.mjs --only dashboard,employees --limit 20
 *   node tools/check/smoke-web.mjs --json tools/check/smoke-report.json
 *   node tools/check/smoke-web.mjs --headless false      # دیدن اجرا در پنجرهٔ واقعی
 *   node tools/check/smoke-web.mjs --help
 *
 * بیرون‌داد: کد خروج ۰ فقط وقتی هیچ صفحه‌ای ایراد نداشته باشد.
 */
import { spawn } from 'node:child_process';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { tmpdir } from 'node:os';
import process from 'node:process';

/* ------------------------------- پیش‌فرض‌ها ------------------------------- */

const DEFAULTS = {
  url: 'http://127.0.0.1:8099/',
  browser: null, // null = تشخیص خودکار
  user: 'hr.manager',
  pass: 'demo',
  port: 9333,
  headless: true,
  timeout: 8000, // مهلت هر انتظار/ارزیابی (میلی‌ثانیه)
  pause: 400, // زمان آرام‌گرفتن پس از رفتن به هر صفحه
  only: null, // فهرست کلید صفحه‌ها یا null
  limit: 0, // ۰ = بی‌نهایت
  json: null, // مسیر گزارش ماشین‌خوان
  help: false,
};

/* متن کمکی فارسی (`--help`) */
const HELP = `آزمون دودِ نمونهٔ وبِ سامانهٔ منابع انسانی (بدون وابستگی خارجی)

استعمال:
  node tools/check/smoke-web.mjs [گزینه‌ها]

گزینه‌ها:
  --url <url>            آدرس نمونهٔ وب (پیش‌فرض: ${DEFAULTS.url})
  --browser <path>       مسیر اجرایی مرورگر (پیش‌فرض: تشخیص خودکار؛
                         اول متغیر محیطی HRMS_BROWSER، سپس Edge و Chrome در
                         مسیرهای متعارف ویندوز/مک/لینوکس)
  --user <username>      نام کاربری ورود (پیش‌فرض: ${DEFAULTS.user})
  --pass <password>      رمز عبور ورود (پیش‌فرض: ${DEFAULTS.pass})
  --port <n>             درگاه اشکال‌زدایی CDP (پیش‌فرض: ${DEFAULTS.port})
  --headless <true|false> اجرای بی‌پنجره (پیش‌فرض: true؛ برای false پنجرهٔ مرورگر دیده می‌شود)
  --timeout <ms>         مهلت هر انتظار/ارزیابی صفحه (پیش‌فرض: ${DEFAULTS.timeout})
  --pause <ms>           مکث پس از رفتن به هر صفحه (پیش‌فرض: ${DEFAULTS.pause})
  --only a,b,c           فقط همین کلیدهای صفحه آزمایش شوند
  --limit <n>            فقط n صفحهٔ اول آزمایش شود
  --json <path>          گزارش ماشین‌خوان (JSON) هم در این مسیر نوشته شود
  --help                 همین متن

نمونه‌ها:
  node tools/check/smoke-web.mjs
  node tools/check/smoke-web.mjs --user admin --json tools/check/smoke-report.json
  node tools/check/smoke-web.mjs --headless false --only dashboard,employees

نکته: اگر سرور بالا نباشد با پیام «سرور روی <url> پاسخ نمی‌دهد؛ اول
start-HRMS.bat را اجرا کنید» و کد خروج ۱ متوقف می‌شود.
`;

/* ------------------------------ ابزارهای کوچک ----------------------------- */

/** خواب کوتاه بین تلاش‌ها. */
const sleep = ms => new Promise(resolve => setTimeout(resolve, Math.max(0, ms)));

/** ارقام فارسی برای پیام‌های کاربر (داده و JSON لاتین می‌ماند). */
const fa = value => String(value).replace(/\d/g, d => '۰۱۲۳۴۵۶۷۸۹'[Number(d)]);

/** پیام خطای فارسی روی خروجی خطا. */
const err = message => console.error(`✖ ${message}`);

/** هشدار فارسی؛ روی خروجی استاندارد نوشته می‌شود تا کد خروج را ناموفق نشان ندهد
 *  (برخی پوسته‌ها نوشتن روی stderr را به‌عنوان شکست فرمان می‌بینند). */
const warn = message => console.log(`⚠ ${message}`);

/** قالب‌بندی بایت‌اندازهٔ پرونده‌ها لازم نیست؛ فقط رخدادها گزارش می‌شوند. */
const oneLine = text => String(text ?? '').replace(/\s+/g, ' ').trim();

/* -------------------------------- آرگومان‌ها ------------------------------- */

/** تجزیهٔ آرگومان‌های خط فرمان؛ مقدار هر کلید یا با فاصله یا با «=» داده می‌شود. */
function parseArgs(argv) {
  const opts = { ...DEFAULTS };
  for (let i = 0; i < argv.length; i++) {
    const raw = argv[i];
    if (!raw.startsWith('--')) throw new Error(`آرگومان ناشناخته: ${raw} (برای راهنما --help را بزنید)`);
    const eq = raw.indexOf('=');
    const key = eq === -1 ? raw.slice(2) : raw.slice(2, eq);
    let value = eq === -1 ? null : raw.slice(eq + 1);

    // مقدار را از آرگومان بعدی می‌گیرد؛ برای کلیدهای بولی، نبودِ مقدار یعنی true.
    const take = ({ optionalTrue = false } = {}) => {
      if (value !== null) return value;
      const next = argv[i + 1];
      if (next === undefined || next.startsWith('--')) {
        if (optionalTrue) return 'true';
        throw new Error(`مقدار برای --${key} داده نشده است`);
      }
      value = next;
      i++;
      return value;
    };
    const takeNumber = field => {
      const n = Number(take());
      if (!Number.isFinite(n) || n <= 0) throw new Error(`مقدار --${key} باید عددی مثبت باشد (${field})`);
      return Math.trunc(n);
    };

    switch (key) {
      case 'help': opts.help = true; break;
      case 'url': opts.url = take(); break;
      case 'browser': opts.browser = take(); break;
      case 'user': opts.user = take(); break;
      case 'pass': opts.pass = take(); break;
      case 'port': opts.port = takeNumber('درگاه CDP'); break;
      case 'headless': opts.headless = parseBool(take({ optionalTrue: true }), key); break;
      case 'timeout': opts.timeout = takeNumber('مهلت انتظار'); break;
      case 'pause': opts.pause = takeNumber('مکث بین صفحه‌ها'); break;
      case 'limit': opts.limit = takeNumber('تعداد صفحه'); break;
      case 'json': opts.json = take(); break;
      case 'only':
        opts.only = take().split(',').map(s => s.trim()).filter(Boolean);
        if (!opts.only.length) throw new Error('--only خالی است؛ کلید صفحه‌ها را با کاما جدا کنید');
        break;
      default:
        throw new Error(`گزینهٔ ناشناخته: --${key} (برای راهنما --help را بزنید)`);
    }
  }
  return opts;
}

/** تبدیل true/false/1/0/yes/no به بولی. */
function parseBool(value, key) {
  const v = String(value).trim().toLowerCase();
  if (['true', '1', 'yes', 'y', 'on'].includes(v)) return true;
  if (['false', '0', 'no', 'n', 'off'].includes(v)) return false;
  throw new Error(`مقدار --${key} باید true یا false باشد (دریافت‌شده: ${value})`);
}

/* ------------------------------- تشخیص مرورگر ------------------------------ */

/** مسیرهای متعارف مرورگرهای Chromium، به‌ترتیب اولویت. */
function browserCandidates() {
  const win = [
    'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
    'C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe',
  ];
  const mac = [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge',
    '/Applications/Chromium.app/Contents/MacOS/Chromium',
  ];
  const nix = [
    '/usr/bin/google-chrome',
    '/usr/bin/google-chrome-stable',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
    '/snap/bin/chromium',
    '/usr/bin/microsoft-edge',
    'google-chrome',
    'chromium',
  ];
  if (process.platform === 'win32') return [...win, ...nix, ...mac];
  if (process.platform === 'darwin') return [...mac, ...nix];
  return [...nix, ...mac];
}

/** مسیر مرورگر: مقدار صریح، سپس HRMS_BROWSER، سپس تشخیص خودکار. */
function detectBrowser(explicit) {
  if (explicit) {
    if (!existsSync(explicit)) throw new Error(`مرورگر در مسیر --browser یافت نشد: ${explicit}`);
    return explicit;
  }
  const envPath = process.env.HRMS_BROWSER;
  if (envPath) {
    if (!existsSync(envPath)) throw new Error(`HRMS_BROWSER به مسیر ناموجود اشاره می‌کند: ${envPath}`);
    return envPath;
  }
  const found = browserCandidates().find(p => existsSync(p));
  if (!found) {
    throw new Error('هیچ مرورگر Chromium یافت نشد؛ مسیر آن را با --browser بدهید یا HRMS_BROWSER را تنظیم کنید');
  }
  return found;
}

/* --------------------------- گفت‌وگو با مرورگر (CDP) ------------------------- */

/**
 * کلاینت کوچک درخواست/پاسخ روی WebSocket سراسری Node.
 * رخدادهای CDP (بدون id) به شنونده‌های ثبت‌شده داده می‌شوند.
 */
function connectCdp(wsUrl, timeoutMs) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl);
    const pending = new Map();
    const handlers = new Map();
    let nextId = 1;
    let settled = false;

    const timer = setTimeout(() => {
      if (settled) return;
      settled = true;
      try { ws.close(); } catch { /* بی‌اهمیت */ }
      reject(new Error('اتصال به CDP در مهلت مقرر برقرار نشد'));
    }, timeoutMs);

    const emit = (method, params) => {
      for (const handler of handlers.get(method) ?? []) {
        try { handler(params); } catch { /* خطای شنونده نباید اجرا را متوقف کند */ }
      }
    };

    ws.addEventListener('message', event => {
      let msg;
      try { msg = JSON.parse(typeof event.data === 'string' ? event.data : String(event.data)); } catch { return; }
      if (msg.id !== undefined) {
        const entry = pending.get(msg.id);
        if (!entry) return;
        pending.delete(msg.id);
        clearTimeout(entry.timer);
        if (msg.error) entry.reject(new Error(`CDP: ${msg.error.message || 'خطای نامشخص'}`));
        else entry.resolve(msg.result ?? {});
        return;
      }
      if (msg.method) emit(msg.method, msg.params);
    });

    ws.addEventListener('error', () => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      reject(new Error('اتصال WebSocket به CDP ناموفق بود؛ اگر پیام «origin» دیدید مرورگر را با --remote-allow-origins=* اجرا کنید'));
    });

    ws.addEventListener('open', () => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      resolve({
        /** فرمان CDP با مهلت؛ پاسخ یا خطای فارسی برمی‌گرداند. */
        send(method, params = {}, ms = timeoutMs) {
          return new Promise((res, rej) => {
            const id = nextId++;
            const t = setTimeout(() => {
              pending.delete(id);
              rej(new Error(`فرمان ${method} در ${ms}ms پاسخ نداد`));
            }, ms);
            pending.set(id, { resolve: res, reject: rej, timer: t });
            ws.send(JSON.stringify({ id, method, params }));
          });
        },
        /** ثبت شنوندهٔ رخداد؛ تابع لغو برمی‌گرداند. */
        on(method, handler) {
          if (!handlers.has(method)) handlers.set(method, new Set());
          handlers.get(method).add(handler);
          return () => handlers.get(method)?.delete(handler);
        },
        close() {
          for (const entry of pending.values()) {
            clearTimeout(entry.timer);
            entry.reject(new Error('اتصال CDP بسته شد'));
          }
          pending.clear();
          try { ws.close(); } catch { /* بی‌اهمیت */ }
        },
      });
    });

    ws.addEventListener('close', () => {
      for (const entry of pending.values()) {
        clearTimeout(entry.timer);
        entry.reject(new Error('اتصال CDP بسته شد'));
      }
      pending.clear();
    });
  });
}

/** انبار کوچک مشکل‌ها: هر مشکل به صفحهٔ جاری نسبت داده می‌شود و سپس خالی می‌شود. */
function problemSink() {
  let buffer = [];
  return {
    add(text) {
      const line = oneLine(text).slice(0, 400);
      if (!line) return;
      if (buffer[buffer.length - 1] === line) return; // خطاهای تکراری (log + console) دوباره ثبت نشوند
      buffer.push(line);
    },
    drain() {
      const out = buffer;
      buffer = [];
      return out;
    },
  };
}

/** توضیح خوانا برای Runtime.exceptionThrown. */
function describeException(details) {
  if (!details) return 'خطای نامشخص';
  const where = details.url ? ` (${details.url}:${(details.lineNumber ?? 0) + 1})` : '';
  return `${details.exception?.description || details.text || 'خطای نامشخص'}${where}`;
}

/** توضیح خوانا برای Runtime.consoleAPICalled با نوع error. */
function describeConsole(params) {
  const parts = (params?.args ?? []).map(a => a?.value ?? a?.description ?? a?.type ?? '').filter(Boolean);
  return `console.error: ${parts.join(' ') || '(بدون پیام)'}`;
}

/** ثبت گیرنده‌های خطای مرورگر روی انبار مشکل‌ها. */
function attachProblemHandlers(client, sink) {
  client.on('Runtime.exceptionThrown', p => sink.add(describeException(p?.exceptionDetails)));
  client.on('Log.entryAdded', p => {
    if (p?.entry?.level !== 'error') return;
    const where = p.entry.url ? ` (${p.entry.url}:${(p.entry.lineNumber ?? 0) + 1})` : '';
    sink.add(`${p.entry.text}${where}`);
  });
  client.on('Runtime.consoleAPICalled', p => {
    if (p?.type === 'error') sink.add(describeConsole(p));
  });
}

/* ------------------------------- کنش‌های صفحه ------------------------------ */

/** ارزیابی عبارت در صفحه؛ خطای خودِ عبارت را هم برمی‌گرداند (بدون throw). */
async function evalJs(client, expression, timeoutMs) {
  const res = await client.send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true }, timeoutMs);
  if (res.exceptionDetails) {
    return { value: undefined, error: describeException(res.exceptionDetails) };
  }
  return { value: res.result ? res.result.value : undefined, error: null };
}

/**
 * تا وقتی مقدار عبارت «منتظر» است تکرار می‌کند؛ آخرین مقدار (یا 'timeout') را
 * برمی‌گرداند. خطاهای ارزیابی فقط ثبت می‌شوند تا مهلت هدر نرود.
 */
async function pollUntil(client, expression, { timeout, interval = 250 }) {
  const deadline = Date.now() + timeout;
  let last = null;
  let lastError = null;
  while (Date.now() < deadline) {
    const { value, error } = await evalJs(client, expression, timeout);
    if (error) lastError = error;
    else {
      last = value;
      if (value !== 'waiting') return value;
    }
    await sleep(interval);
  }
  if (last !== null) return last;
  return lastError ? `error:${lastError}` : 'timeout';
}

/** آمادگی فرم ورود یا پوستهٔ برنامه. */
const READY_EXPR = `(() => {
  const app = document.getElementById('app');
  const loginErr = document.getElementById('loginErr');
  const appVisible = !!app && !app.classList.contains('hidden');
  const hasSession = typeof window.session !== 'undefined' && window.session !== null;
  const sidebar = document.getElementById('sidebar');
  if (loginErr && loginErr.classList.contains('show') && !appVisible) return 'login-error';
  if (hasSession || (appVisible && !!sidebar && sidebar.innerHTML.length > 0)) return 'ready';
  return 'waiting';
})()`;

/** فرم ورود ساخته شده است؟ (پیش از کلیک ورود لازم است) */
const LOGIN_FORM_EXPR = `(() => {
  const u = document.querySelector('#loginUser');
  const p = document.querySelector('#loginPass');
  const b = document.querySelector('#loginBtn');
  if (!u || !p || !b) return 'waiting';
  if (!document.getElementById('app').classList.contains('hidden')) return 'logged-in';
  return 'form';
})()`;

/** پر کردن نام کاربری/رمز، رویداد input، و کلیک دکمهٔ ورود. */
function loginExpression(user, pass) {
  return `(() => {
  const u = document.querySelector('#loginUser');
  const p = document.querySelector('#loginPass');
  const b = document.querySelector('#loginBtn');
  if (!u || !p || !b) return 'no-form';
  u.value = ${JSON.stringify(user)};
  p.value = ${JSON.stringify(pass)};
  u.dispatchEvent(new Event('input', { bubbles: true }));
  p.dispatchEvent(new Event('input', { bubbles: true }));
  b.click();
  return 'clicked';
})()`;
}

/**
 * رفتن به یک صفحه: خطای go() در window.__smokeErr ثبت می‌شود و توست‌های قبلی
 * پاک می‌شوند تا پیام «دسترسی ندارید» به صفحهٔ بعدی نسبت داده نشود.
 */
function goExpression(key) {
  return `(() => {
  window.__smokeErr = null;
  const toasts = document.getElementById('toastRoot');
  if (toasts) toasts.innerHTML = '';
  try { go(${JSON.stringify(key)}); } catch (e) { window.__smokeErr = String((e && e.stack) || e); }
  return true;
})()`;
}

/** خواندن وضعیت صفحهٔ جاری: طول HTML، متن دیده‌شده، دسترسی و خطای go(). */
const READ_PAGE_EXPR = `(() => {
  const main = document.querySelector('#main');
  const bodyText = document.body ? (document.body.innerText || '') : '';
  return {
    len: main ? main.innerHTML.length : 0,
    snippet: bodyText.replace(/\\s+/g, ' ').trim().slice(0, 400),
    denied: /دسترسی ندارید/.test(bodyText),
    err: window.__smokeErr || null,
  };
})()`;

/** فهرست کلید همهٔ صفحه‌ها از منوی تولیدشده (`window.HRMS_MENU`). */
const MENU_EXPR = `[...new Set((window.HRMS_MENU||[]).flatMap(d=>(d.departments||[]).flatMap(dep=>(dep.groups||[]).flatMap(g=>(g.modules||[]).flatMap(m=>(m.screens||[]).map(s=>s.page))))))]`;

/* ------------------------------ راه‌اندازی مرورگر --------------------------- */

/** بررسی اینکه سرور نمونهٔ وب پاسخ می‌دهد (پیش از هر راه‌اندازی مرورگر). */
async function assertServerUp(url, timeoutMs = 5000) {
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), timeoutMs);
  try {
    const res = await fetch(url, { signal: ctrl.signal, redirect: 'follow' });
    if (!res.ok) throw new Error(`کد پاسخ ${res.status}`);
    await res.arrayBuffer(); // بدنه را کامل بخوان تا سرور واقعاً سرو می‌کند
    return res.status;
  } catch (e) {
    const reason = e.name === 'AbortError' ? 'مهلت پاسخ تمام شد' : (e.cause?.code || e.message);
    throw new Error(`سرور روی ${url} پاسخ نمی‌دهد (${reason})؛ اول start-HRMS.bat را اجرا کنید`);
  } finally {
    clearTimeout(timer);
  }
}

/** آرگومان‌های راه‌اندازی مرورگر. */
function buildLaunchArgs(opts, userDataDir) {
  const args = [
    `--remote-debugging-port=${opts.port}`,
    // اجازهٔ اتصال WebSocket از Node (سرصفحهٔ Origin نامعلوم) به درگاه CDP.
    '--remote-allow-origins=*',
    `--user-data-dir=${userDataDir}`,
    '--no-first-run',
    '--no-default-browser-check',
    '--disable-extensions',
    '--disable-gpu',
  ];
  if (opts.headless) args.push('--headless=new');
  args.push(opts.url); // صفحهٔ آغازین
  return args;
}

/**
 * راه‌اندازی مرورگر با stdio: 'ignore' — این محیط اجازهٔ لوله‌کشی خروجی برای
 * فرایندهای نواده را نمی‌دهد، پس هیچ‌جا از stdio:'pipe' استفاده نمی‌شود.
 */
function launchBrowser(browserPath, args) {
  const proc = spawn(browserPath, args, { stdio: 'ignore', windowsHide: true });
  let spawnError = null;
  proc.on('error', e => { spawnError = e; });
  return { proc, spawnErrorOf: () => spawnError };
}

/** انتظار برای هدف CDP از نوع page. */
async function findPageTarget(port, url, { tries = 20, interval = 250 } = {}) {
  for (let attempt = 1; attempt <= tries; attempt++) {
    try {
      const res = await fetch(`http://127.0.0.1:${port}/json/list`);
      if (res.ok) {
        const list = await res.json();
        const pages = (Array.isArray(list) ? list : []).filter(t => t.type === 'page' && t.webSocketDebuggerUrl);
        const match = pages.find(t => t.url && t.url.startsWith(url)) ?? pages[0];
        if (match) return match;
      }
    } catch { /* مرورگر هنوز بالا نیامده؛ تلاش بعدی */ }
    await sleep(interval);
  }
  throw new Error(`هدف CDP روی درگاه ${port} پیدا نشد؛ شاید مرورگر بالا نیامده یا درگاه اشغال است (--port را عوض کنید)`);
}

/** کشتن قطعی مرورگر (روی ویندوز با taskkill و کل درخت فرایند). */
async function killBrowser(proc) {
  if (!proc || proc.exitCode !== null || proc.signalCode !== null) return;
  const exited = new Promise(resolve => {
    proc.once('exit', () => resolve(true));
    setTimeout(() => resolve(false), 4000);
  });
  try { proc.kill(); } catch { /* بی‌اهمیت */ }
  if (await exited) return;
  if (process.platform === 'win32') {
    // در ویندوز فقط با taskkill می‌توان کل درخت فرایند را بست.
    await new Promise(resolve => {
      const tk = spawn('taskkill', ['/pid', String(proc.pid), '/t', '/f'], { stdio: 'ignore', windowsHide: true });
      tk.on('exit', resolve);
      tk.on('error', resolve);
    });
  } else {
    try { proc.kill('SIGKILL'); } catch { /* بی‌اهمیت */ }
  }
}

/* --------------------------------- گزارش --------------------------------- */

/** نوشتن گزارش JSON (فقط همین مسیر نوشته می‌شود؛ پوشهٔ لازم ساخته می‌شود). */
function writeReport(path, report) {
  const target = path;
  const dir = dirname(target);
  try {
    writeFileSync(target, `${JSON.stringify(report, null, 2)}\n`, 'utf8');
  } catch (e) {
    if (e.code !== 'ENOENT') throw e;
    mkdirSync(dir, { recursive: true }); // پوشهٔ گزارشِ خواستهٔ کاربر
    writeFileSync(target, `${JSON.stringify(report, null, 2)}\n`, 'utf8');
  }
}

/** چاپ خلاصهٔ فارسی و کد خروج را برمی‌گرداند. */
function printSummary({ opts, pagesFound, tested, passed, failures, denied }) {
  console.log('');
  console.log('— خلاصهٔ آزمون دود نمونهٔ وب —');
  console.log(`آدرس: ${opts.url} | کاربر: ${opts.user} | درگاه CDP: ${fa(opts.port)}`);
  console.log(`صفحه‌های یافته‌شده: ${fa(pagesFound)} | آزمایش‌شده: ${fa(tested)} | موفق: ${fa(passed)} | ناموفق: ${fa(failures.length)}`);
  if (denied) {
    console.log(`· ${fa(denied)} صفحه برای این نقش پیام «دسترسی ندارید» داد (ایراد نیست؛ برای پوشش کامل با --user admin اجرا کنید).`);
  }
  for (const f of failures) {
    console.log(`  - ${f.page} — ${f.reason}`);
    if (f.exception) console.log(`    خطا: ${oneLine(f.exception).slice(0, 200)}`);
  }
  if (!failures.length) {
    console.log(`\n✓ همهٔ ${fa(tested)} صفحه بدون خطا`);
    return 0;
  }
  console.log(`\n✖ ${fa(failures.length)} صفحه ایراد داشت`);
  return 1;
}

/* ---------------------------------- جریان --------------------------------- */

/**
 * وضعیت مشترک فقط برای پاک‌سازی در سیگنال/خطای ناگهانی لازم است:
 * فرایند مرورگر و پوشهٔ موقت پروفایل.
 */
const cleanupState = { proc: null, userDataDir: null, done: false };

/** بستن مرورگر و پاک‌کردن پوشهٔ موقت پروفایل (چند بار صدا زدنش بی‌خطر است). */
async function cleanup() {
  if (cleanupState.done) return;
  cleanupState.done = true;
  await killBrowser(cleanupState.proc);
  const dir = cleanupState.userDataDir;
  if (dir && dir.includes('hrms-smoke-')) {
    // ویندوز تا چند لحظه پس از خروج مرورگر پرونده‌ها را نگه می‌دارد؛ ناموفق‌بودن این
    // پاک‌سازی ایراد آزمون نیست، پس فقط هشدار داده می‌شود و کد خروج عوض نمی‌شود.
    for (let attempt = 0; attempt < 3; attempt++) {
      try {
        rmSync(dir, { recursive: true, force: true, maxRetries: 5, retryDelay: 300 });
        return;
      } catch (e) {
        if (attempt === 2) {
          warn(`پاک‌کردن پوشهٔ موقت پروفایل ناموفق بود: ${dir} (${e.message}) — دستی حذفش کنید`);
        }
        await sleep(600);
      }
    }
  }
}

/** آزمون یک صفحه؛ null یعنی موفق، وگرنه رکورد ایراد. */
async function testPage(client, key, opts, sink) {
  sink.drain(); // خطاهای صفحهٔ قبلی به این صفحه نسبت داده نشود
  const navigation = await evalJs(client, goExpression(key), opts.timeout);
  if (navigation.error) {
    const exceptions = sink.drain();
    return { failure: { page: key, reason: `اجرای go() خطا داد: ${oneLine(navigation.error)}`, exception: exceptions[0] ?? null, snippet: '' } };
  }
  await sleep(opts.pause);
  const read = await evalJs(client, READ_PAGE_EXPR, opts.timeout);
  const exceptions = sink.drain();
  if (read.error) {
    return { failure: { page: key, reason: `خواندن وضعیت صفحه خطا داد: ${oneLine(read.error)}`, exception: exceptions[0] ?? null, snippet: '' } };
  }
  const info = read.value && typeof read.value === 'object' ? read.value : {};
  const denied = info.denied === true;
  if (info.err) {
    return { failure: { page: key, reason: `خطای اجرای صفحه: ${oneLine(info.err).slice(0, 200)}`, exception: oneLine(info.err), snippet: info.snippet ?? '' }, denied };
  }
  if (exceptions.length) {
    return { failure: { page: key, reason: `خطای جاوااسکریپت: ${exceptions[0]}`, exception: exceptions[0], snippet: info.snippet ?? '' }, denied };
  }
  if (!(Number(info.len) >= 400)) {
    const extra = denied ? ' — صفحه برای این نقش دسترسی نداشت' : '';
    return { failure: { page: key, reason: `محتوای #main رندر نشد (${fa(info.len)} نویسهٔ HTML)${extra}`, exception: null, snippet: info.snippet ?? '' }, denied };
  }
  return { failure: null, denied };
}

/** جریان اصلی: بررسی سرور، راه‌اندازی مرورگر، ورود، پیمایش صفحه‌ها، گزارش. */
async function main() {
  const opts = parseArgs(process.argv.slice(2));
  if (opts.help) {
    console.log(HELP);
    return 0;
  }

  // WebSocket سراسری از Node 22 به بعد بدون پرچم در دسترس است.
  if (typeof WebSocket !== 'function') {
    throw new Error('WebSocket سراسری در این نسخهٔ Node موجود نیست؛ Node را به نسخهٔ ۲۲ یا بالاتر به‌روز کنید');
  }

  console.log(`· بررسی سرور روی ${opts.url} …`);
  await assertServerUp(opts.url);

  const browserPath = detectBrowser(opts.browser);
  console.log(`· مرورگر: ${browserPath}`);

  const userDataDir = join(tmpdir(), `hrms-smoke-${process.pid}`);
  cleanupState.userDataDir = userDataDir;

  const args = buildLaunchArgs(opts, userDataDir);
  const { proc, spawnErrorOf } = launchBrowser(browserPath, args);
  cleanupState.proc = proc;
  console.log(`· راه‌اندازی مرورگر (بی‌پنجره: ${opts.headless ? 'بله' : 'خیر'}) روی درگاه ${fa(opts.port)} …`);

  await sleep(200);
  const spawnError = spawnErrorOf();
  if (spawnError) throw new Error(`اجرای مرورگر ناموفق بود: ${spawnError.message}`);

  const target = await findPageTarget(opts.port, opts.url);
  const client = await connectCdp(target.webSocketDebuggerUrl, opts.timeout);
  const sink = problemSink();
  attachProblemHandlers(client, sink);

  let loginFailed = false;
  const failures = [];
  let pagesFound = 0;
  let tested = 0;
  let denied = 0;
  const report = {
    url: opts.url,
    user: opts.user,
    at: new Date().toISOString(),
    pagesFound: 0,
    tested: 0,
    passed: 0,
    failed: 0,
    failures: [],
    loginFailed: false,
  };

  try {
    await client.send('Runtime.enable', {}, opts.timeout);
    await client.send('Page.enable', {}, opts.timeout);
    await client.send('Log.enable', {}, opts.timeout);

    // ---- ورود ----
    // کلیک ورود ممکن است پیش از سیم‌کشی رویدادها انجام شود (اسکریپت‌ها انتهای صفحه
    // بارگذاری می‌شوند)، پس تا چند بار تلاش می‌شود و هر بار وضعیت دوباره خوانده می‌شود.
    const form = await pollUntil(client, LOGIN_FORM_EXPR, { timeout: opts.timeout });
    if (form === 'form') {
      let ready = 'waiting';
      for (let attempt = 0; attempt < 6; attempt++) {
        await evalJs(client, loginExpression(opts.user, opts.pass), opts.timeout);
        ready = await pollUntil(client, READY_EXPR, { timeout: Math.min(2500, opts.timeout) });
        if (ready === 'ready' || ready === 'login-error') break;
        await sleep(700);
      }
      if (ready !== 'ready') {
        loginFailed = true;
        err(`ورود با کاربر «${opts.user}» انجام نشد (${ready})؛ نام کاربری/رمز را با --user و --pass بررسی کنید و ببینید سرور ورود را می‌پذیرد.`);
        report.pagesFound = 0;
        report.loginFailed = true;
        report.tested = 0;
        report.passed = 0;
        report.failed = 0;
        report.failures = [];
        if (opts.json) {
          writeReport(opts.json, report);
          console.log(`· گزارش ماشین‌خوان نوشته شد: ${opts.json}`);
        }
        return 1;
      }
    } else if (form !== 'logged-in') {
      throw new Error(`فرم ورود در صفحهٔ نمونه پیدا نشد (${form})؛ بررسی کنید ${opts.url} همان نمونهٔ وب را سرو می‌کند`);
    }
    console.log(`· ورود انجام شد (${opts.user})`);

    // ---- فهرست صفحه‌ها ----
    const menu = await evalJs(client, MENU_EXPR, opts.timeout);
    let keys = Array.isArray(menu.value) ? menu.value.filter(k => typeof k === 'string' && k) : [];
    pagesFound = keys.length;
    if (!pagesFound) {
      throw new Error('فهرست صفحه‌ها خالی است؛ احتمالاً js/menu.generated.js بارگذاری نشده است (npm run gen را اجرا کنید)');
    }

    if (opts.only) {
      const unknown = opts.only.filter(k => !keys.includes(k));
      if (unknown.length) throw new Error(`کلید صفحهٔ ناشناخته در --only: ${unknown.join('، ')}`);
      keys = opts.only;
    }
    if (opts.limit > 0) keys = keys.slice(0, opts.limit);

    console.log(`· صفحه‌های یافته‌شده: ${fa(pagesFound)} — آزمایش ${fa(keys.length)} صفحه`);
    for (const key of keys) {
      tested++;
      const { failure, denied: wasDenied } = await testPage(client, key, opts, sink);
      if (wasDenied) denied++;
      if (failure) {
        failures.push(failure);
        console.log(`  ✖ ${key} — ${failure.reason}`);
      }
    }

    report.pagesFound = pagesFound;
    report.tested = tested;
    report.passed = tested - failures.length;
    report.failed = failures.length;
    report.failures = failures;
    report.loginFailed = loginFailed;
    if (opts.json) {
      writeReport(opts.json, report);
      console.log(`· گزارش ماشین‌خوان نوشته شد: ${opts.json}`);
    }
    return printSummary({ opts, pagesFound, tested, passed: tested - failures.length, failures, denied });
  } catch (e) {
    err(e.message);
    report.pagesFound = pagesFound;
    report.tested = tested;
    report.passed = tested - failures.length;
    report.failed = failures.length;
    report.failures = failures;
    report.loginFailed = loginFailed;
    if (opts.json) {
      try { writeReport(opts.json, report); } catch (we) { err(`نوشتن گزارش ناموفق بود: ${we.message}`); }
    }
    return 1;
  } finally {
    client.close();
  }
}

/* ------------------------------ اجرا و پاک‌سازی ----------------------------- */

let exiting = false;
/** خروج قطعی با پاک‌سازی مرورگر و پروفایل موقت. */
async function shutdown(code) {
  if (exiting) return;
  exiting = true;
  await cleanup();
  process.exit(code);
}

process.on('SIGINT', () => {
  err('اجرا با Ctrl+C متوقف شد؛ مرورگر و پروفایل موقت بسته می‌شوند…');
  void shutdown(130);
});
process.on('SIGTERM', () => void shutdown(143));

main()
  .then(code => shutdown(code))
  .catch(async e => {
    err(e?.message || String(e));
    await shutdown(1);
  });
