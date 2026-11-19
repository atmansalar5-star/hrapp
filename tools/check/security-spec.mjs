/**
 * دروازهٔ ردیابی اسپک امنیتی (spec-driven):
 *  ۱. هر بند REQ در docs/security-spec-fa.md باید تأیید داشته باشد.
 *  ۲. هر تأیید باید واقعاً وجود داشته باشد: آزمون go:… در سرور، ابزار tool:… در tools/check،
 *     یا کلیدواژهٔ doc:…#… در سند عملیاتی.
 *  ۳. هیچ آزمون TestREQxx در سرور نباید بدون بند متناظر باشد (آزمون یتیم).
 *
 * استعمال: node tools/check/security-spec.mjs
 */
import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const SPEC = join(ROOT, 'docs', 'security-spec-fa.md');

const failures = [];
const notes = [];
const fail = (gate, msg) => failures.push(`[${gate}] ${msg}`);

const walk = dir => {
  if (!existsSync(dir)) return [];
  const out = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (['node_modules', '.git', 'bin'].includes(entry.name)) continue;
    const p = join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(p));
    else out.push(p);
  }
  return out;
};
const rel = p => relative(ROOT, p).replace(/\\/g, '/');

// نرمال‌سازی برای تطبیق کلیدواژه: فاصله، نیم‌فاصله، خط تیره و بزرگی حروف بی‌اثر می‌شوند.
const normalize = s => s.replace(/[\s\u200c\-_]/g, '').toLowerCase();

if (!existsSync(SPEC)) {
  console.error(`✖ اسپک امنیتی یافت نشد: ${rel(SPEC)}`);
  process.exit(1);
}
const spec = readFileSync(SPEC, 'utf8');

// ---------- ۱. خواندن بندها ----------
const reqs = new Map();
for (const line of spec.split(/\r?\n/)) {
  if (!/^\|\s*REQ-\d+\s*\|/.test(line)) continue;
  const cells = line.split('|').map(c => c.trim());
  const id = cells[1];
  if (reqs.has(id)) fail('spec', `بند تکراری: ${id}`);
  // ستون «تأیید» آخرین ستون پرشده پیش از ستون خالی انتهایی است.
  const verify = cells.slice(2, -1).pop() ?? '';
  reqs.set(id, verify);
}
if (reqs.size === 0) fail('spec', 'هیچ بند REQ-xx در سند یافت نشد');
notes.push(`بندها: ${reqs.size}`);

// ---------- ۲. جمع‌آوری آزمون‌های سرور ----------
const goTests = new Map(); // نام آزمون → فایل
for (const file of walk(join(ROOT, 'server')).filter(f => f.endsWith('_test.go'))) {
  const text = readFileSync(file, 'utf8');
  for (const m of text.matchAll(/func (TestREQ\d+[A-Za-z0-9_]*)\(/g)) {
    goTests.set(m[1], rel(file));
  }
}
notes.push(`آزمون‌های REQ: ${goTests.size}`);

// ---------- ۳. سنجش تأیید هر بند ----------
for (const [id, cell] of reqs) {
  const tokens = cell.split(/[،,]/).map(t => t.trim()).filter(Boolean);
  if (tokens.length === 0) {
    fail('verify', `${id} هیچ تأییدی ندارد`);
    continue;
  }
  for (const token of tokens) {
    const sep = token.indexOf(':');
    if (sep < 0) {
      fail('verify', `${id}: تأیید «${token}» شکل درست (go:/tool:/doc:) ندارد`);
      continue;
    }
    const kind = token.slice(0, sep);
    const rest = token.slice(sep + 1);

    if (kind === 'go') {
      if (!goTests.has(rest)) fail('verify', `${id}: آزمون «${rest}» در server/ یافت نشد`);
    } else if (kind === 'tool') {
      const file = join(ROOT, 'tools', 'check', rest.endsWith('.mjs') ? rest : `${rest}.mjs`);
      if (!existsSync(file)) fail('verify', `${id}: ابزار «${rel(file)}» یافت نشد`);
    } else if (kind === 'doc') {
      const [filePart, keyword = ''] = rest.split('#');
      const file = join(ROOT, filePart);
      if (!existsSync(file)) {
        fail('verify', `${id}: سند «${filePart}» یافت نشد`);
        continue;
      }
      const body = normalize(readFileSync(file, 'utf8'));
      if (keyword && !body.includes(normalize(keyword))) {
        fail('verify', `${id}: سند «${filePart}» کلیدواژهٔ «${keyword}» را ندارد`);
      }
    } else {
      fail('verify', `${id}: نوع تأیید ناشناخته «${kind}»`);
    }
  }
}

// ---------- ۴. آزمون یتیم ----------
for (const [name, file] of goTests) {
  const id = `REQ-${name.slice(7, 9)}`;
  if (!reqs.has(id)) fail('orphan', `آزمون ${name} (${file}) بند ${id} در اسپک ندارد`);
}

// ---------- گزارش ----------
if (failures.length) {
  console.error('✖ دروازهٔ اسپک امنیتی شکست خورد:');
  for (const f of failures) console.error(`  - ${f}`);
  process.exit(1);
}
console.log('✔ اسپک امنیتی هم‌خوان است');
for (const n of notes) console.log(`  - ${n}`);