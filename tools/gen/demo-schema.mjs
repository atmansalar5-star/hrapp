#!/usr/bin/env node
/**
 * ساخت «نمای ساختار» برای صفحههای زندهٔ نمونهٔ وب
 * =================================================
 *
 * چرا این ابزار: منوی نمونهٔ وب ۱۳۲ صفحه دارد؛ صفحههایی که فقط نسخهٔ زنده دارند در
 * حالت بدون سرور به بنبست میرسیدند (پیشتر «صفحهٔ ناشناخته»، اکنون صفحهٔ راهنمای
 * «نیازمند سرور»). این ابزار از خودِ سرور میپرسد هر ماژول چه نوع رکوردی، چه فیلدهایی،
 * چه وضعیتهایی و چه اقدامهایی دارد و نتیجه را در یک فایل تولیدشده میریزد تا نمونهٔ
 * رابط در حالت بدون سرور هم ساختار واقعی صفحه را نشان دهد.
 *
 * هیچ دادهای ساخته نمیشود: فقط فیلد/وضعیت/اقدام از `/schema` سرور خوانده و به
 * `design/prototype/js/demo-schema.generated.js` نوشته میشود. رکورد نمایشی وجود ندارد
 * و در همان صفحه به کاربر گفته میشود برای دیدن داده، سرور را بالا بیاورد.
 *
 * اجرا (سرور باید بالا باشد):
 *   node tools/gen/demo-schema.mjs
 *   node tools/gen/demo-schema.mjs --url http://127.0.0.1:8099 --user hr.manager --pass demo
 *   node tools/gen/demo-schema.mjs --check     # فقط مقایسه با فایل موجود (بدون نوشتن)
 *   node tools/gen/demo-schema.mjs --help
 *
 * بیرونداد: کد خروج ۰ = فایل با سرور همخوان است.
 */
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import process from 'node:process';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..');
const MENU = resolve(ROOT, 'design/prototype/js/menu.generated.js');
const RK_PAGES = resolve(ROOT, 'design/prototype/js/live_records.js');
const OUT = resolve(ROOT, 'design/prototype/js/demo-schema.generated.js');

const DEFAULTS = { url: 'http://127.0.0.1:8099', user: 'hr.manager', pass: 'demo', check: false };

const HELP = `ساخت نمای ساختار صفحههای زنده (خواندن /schema از سرور محلی)

استعمال:
  node tools/gen/demo-schema.mjs [گزینهها]

گزینهها:
  --url <url>    آدرس سرور (پیشفرض: ${DEFAULTS.url})
  --user <name>  کاربر سامانه (پیشفرض: ${DEFAULTS.user})
  --pass <pass>  گذرواژه (پیشفرض: ${DEFAULTS.pass})
  --check        فقط بررسی همخوانی با فایل موجود؛ چیزی نوشته نمیشود
  --help         همین راهنما
`;

function parseArgs(argv) {
  const out = {...DEFAULTS};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--help') { process.stdout.write(HELP); process.exit(0); }
    else if (a === '--check') out.check = true;
    else if (a === '--url') out.url = argv[++i];
    else if (a === '--user') out.user = argv[++i];
    else if (a === '--pass') out.pass = argv[++i];
    else { process.stderr.write(`گزینهٔ ناشناخته: ${a}\n${HELP}`); process.exit(2); }
  }
  return out;
}

/** صفحههای منو بههمراه ماژول و پیشوند API آنها. */
function menuPages() {
  const text = readFileSync(MENU, 'utf8');
  const out = [];
  let current = null;
  for (const line of text.split('\n')) {
    const mod = line.match(/^\s*id:\s*"([a-z0-9_]+)",\s*$/);
    if (mod) { current = {id: mod[1], prefix: null, pages: []}; out.push(current); continue; }
    const pfx = line.match(/^\s*apiPrefix:\s*"([^"]+)"/);
    if (pfx && current) { current.prefix = pfx[1]; continue; }
    const page = line.match(/^\s*\{\s*page:\s*"([^"]+)",\s*label:\s*"([^"]+)"/);
    if (page && current) current.pages.push({key: page[1], label: page[2]});
  }
  return out.filter((m) => m.prefix && m.pages.length);
}

/** نگاشت صفحه → نوع رکورد، از فایل نمونهٔ رابط (منبع همان مسیریاب زنده). */
function recordKinds() {
  const text = readFileSync(RK_PAGES, 'utf8');
  const map = {};
  for (const m of text.matchAll(/^\s*([A-Za-z0-9_]+):\s*\{kind:\s*'([A-Za-z0-9_]+)'/gm)) map[m[1]] = m[2];
  return map;
}

/** برگهٔ ساختار یک نوع رکورد را به شکل کمحجم تبدیل میکند. */
const trimKind = (k) => ({
  titleFa: k.titleFa,
  idPrefix: k.idPrefix || '',
  initialStatus: k.initialStatus || '',
  fields: (k.fields || []).map((f) => ({k: f.key, l: f.labelFa, t: f.type, r: f.required ? 1 : 0})),
  statuses: (k.statuses || []).map((s) => ({v: s.value, l: s.labelFa})),
  actions: (k.actions || []).map((a) => ({l: a.labelFa, s: a.style || '', to: a.to || '', roles: a.roles || []})),
});

/**
 * نگاشت صریح صفحه → نوع رکورد برای صفحههایی که نامشان با نام نوع رکورد یکی نیست.
 *
 * هر سطر با شاهد نوشته شده است: صفحهٔ `payMethods` همان «روش پرداخت کارمند» است
 * (kind=employeePayModel) و صفحههای ۴پی هر یک به یکی از انواع رکورد ماژول جبران خدمات
 * وصلاند (jobPricing/performanceFactor/payBand/g20Index). صفحههایی که نگاشتشان روشن
 * نیست (مثل g20Board یا compReadiness) عمداً اینجا نیستند تا نگاشت حدسی ساخته نشود.
 */
const KIND_OVERRIDES = {
  payMethods: 'employeePayModel',
  compJobPricing: 'jobPricing',
  compPerformance: 'performanceFactor',
  compPayBands: 'payBand',
  g20Indices: 'g20Index',
  appointmentForm: 'appointments',
};

/** نوع رکورد متناظر یک صفحه: نگاشت صریح، بعد نام برابر، بعد امتیاز توکنی نامها. */
function pickKind(pageKey, kinds) {
  const override = KIND_OVERRIDES[pageKey];
  if (override) {
    const hit = kinds.find((k) => k.key === override);
    if (hit) return hit;
  }
  const norm = (s) => s.toLowerCase().replace(/[^a-z0-9]/g, '');
  const tokens = (s) => s.replace(/([a-z0-9])([A-Z])/g, '$1 $2').toLowerCase().split(/[^a-z0-9]+/).filter(Boolean);
  const p = norm(pageKey);
  const exact = kinds.find((k) => norm(k.key) === p);
  if (exact) return exact;
  const prefix = kinds.find((k) => norm(k.key).startsWith(p) || p.startsWith(norm(k.key)));
  if (prefix) return prefix;
  // امتیاز توکنی: توکنهای نام صفحه که پیشوند توکنی از نام نوع رکورد هستند؛ شرط،
  // همخوانی توکن نخست است تا نگاشتهای دلبخواه ساخته نشود.
  const pt = tokens(pageKey);
  if (!pt.length) return null;
  let best = null;
  let bestScore = 0;
  for (const k of kinds) {
    const kt = tokens(k.key);
    if (!kt.length) continue;
    if (!(kt[0].startsWith(pt[0]) || pt[0].startsWith(kt[0]))) continue;
    const score = pt.filter((t) => kt.some((x) => x.startsWith(t) || t.startsWith(x))).length;
    if (score > bestScore) { best = k; bestScore = score; }
  }
  return bestScore >= 2 ? best : null;
}

async function login(cfg) {
  const res = await fetch(`${cfg.url}/api/v1/auth/login`, {
    method: 'POST', headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({username: cfg.user, password: cfg.pass}),
  });
  if (!res.ok) throw new Error(`ورود ناموفق (${res.status})`);
  const data = await res.json();
  return data.accessToken;
}

async function schemaOf(cfg, token, prefix) {
  const res = await fetch(`${cfg.url}${prefix}/schema`, {headers: {Authorization: `Bearer ${token}`}});
  if (!res.ok) return null;
  return res.json();
}

async function build(cfg, token) {
  const rk = recordKinds();
  const pages = {};
  const skipped = [];
  for (const mod of menuPages()) {
    const schema = await schemaOf(cfg, token, mod.prefix);
    if (!schema || !Array.isArray(schema.kinds) || !schema.kinds.length) {
      skipped.push(mod.id);
      continue;
    }
    for (const page of mod.pages) {
      const explicit = rk[page.key] ? schema.kinds.find((k) => k.key === rk[page.key]) : null;
      const kind = explicit || pickKind(page.key, schema.kinds);
      if (!kind) continue;
      pages[page.key] = Object.assign({module: mod.id, moduleFa: mod.titleFa || schema.moduleTitleFa || '',
        kind: kind.key, labelFa: page.label}, trimKind(kind));
    }
  }
  return {pages, skipped};
}

const HEADER = `/* Code generated by tools/gen/demo-schema.mjs from the local server /schema — DO NOT EDIT.
   نمای ساختار صفحههای زنده: فیلد، وضعیت و اقدام هر نوع رکورد، خواندهشده از خود سرور.
   هیچ رکورد یا عددی ساخته نمیشود؛ برای دیدن داده، سرور محلی باید بالا باشد.
   برای بازتولید: node tools/gen/demo-schema.mjs */\n`;

function render(pages) {
  const keys = Object.keys(pages).sort();
  const body = keys.map((k) => `  ${JSON.stringify(k)}: ${JSON.stringify(pages[k])},`).join('\n');
  return `${HEADER}window.HRMS_DEMO_SCHEMAS = {\n${body}\n};\n`;
}

const cfg = parseArgs(process.argv.slice(2));
let token;
try {
  token = await login(cfg);
} catch (e) {
  process.stderr.write(`سرور در دسترس نیست یا ورود ناموفق بود: ${e.message}\n` +
    `این ابزار به سرور محلی نیاز دارد (start-HRMS.bat یا npm run server).\n`);
  process.exit(3);
}

const {pages, skipped} = await build(cfg, token);
const text = render(pages);
const count = Object.keys(pages).length;

if (cfg.check) {
  const current = existsSync(OUT) ? readFileSync(OUT, 'utf8') : '';
  if (current === text) {
    process.stdout.write(`✓ نمای ساختار با سرور همخوان است (${count} صفحه)\n`);
    process.exit(0);
  }
  process.stderr.write(`✗ فایل تولیدشده با سرور همخوان نیست؛ «node tools/gen/demo-schema.mjs» را اجرا کنید.\n`);
  process.exit(1);
}

writeFileSync(OUT, text, {encoding: 'utf8'});
const missing = skipped.length ? ` | ماژول بی/سکیمای فعال: ${skipped.join(', ')}` : '';
process.stdout.write(`✓ نوشته شد: design/prototype/js/demo-schema.generated.js (${count} صفحه، ` +
  `${Math.round(Buffer.byteLength(text) / 1024)} کیلوبایت)${missing}\n`);
