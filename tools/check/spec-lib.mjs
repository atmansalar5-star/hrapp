/**
 * منطق مشترک دروازه‌های spec-driven: ردیابی دوطرفهٔ بندهای یک سند اسپک با
 * آزمون‌های Go، ابزارها، فایل‌ها و بخش‌های سند عملیاتی.
 *
 * شکل جدول در سند اسپک (ستون آخر «تأیید»):
 *   | OPS-03 | ... | ... | go:TestOPS03X، doc:docs/ops.md#بخش |
 * انواع تأیید:
 *   go:TestName            آزمون Go با همین نام در server/
 *   tool:name              ابزار node در tools/check/name.mjs
 *   file:path              وجود فایل در ریشهٔ مخزن
 *   doc:path#keyword       وجود کلیدواژه در سند (بی‌توجه به فاصله/نیم‌فاصله/خط تیره)
 */
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

export const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');

export const walk = dir => {
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

export const rel = p => relative(ROOT, p).replace(/\\/g, '/');

// نرمال‌سازی برای تطبیق کلیدواژه: فاصله، نیم‌فاصله، خط تیره و بزرگی حروف بی‌اثر می‌شوند.
export const normalize = s => s.replace(/[\s\u200c\-_]/g, '').toLowerCase();

/**
 * checkSpec سند اسپک را می‌خواند و همهٔ بندها را می‌سنجد.
 * @param {{specPath: string, label: string, prefix: string}} options
 *   prefix پیشوند شناسه‌های همین اسپک (مثل REQ یا OPS)؛ آزمون‌های پیشوندهای دیگر
 *   (اسپک‌های دیگر) یتیم حساب نمی‌شوند.
 */
export function checkSpec({ specPath, label, prefix }) {
  const failures = [];
  const notes = [];
  const fail = (gate, msg) => failures.push(`[${gate}] ${msg}`);
  const spec = join(ROOT, specPath);

  if (!existsSync(spec)) {
    return { failures: [`[spec] سند اسپک یافت نشد: ${specPath}`], notes, reqs: 0, tests: 0 };
  }
  const text = readFileSync(spec, 'utf8');

  // ---------- بندها ----------
  const reqs = new Map();
  for (const line of text.split(/\r?\n/)) {
    if (!/^\|\s*[A-Z]+-\d+\s*\|/.test(line)) continue;
    const cells = line.split('|').map(c => c.trim());
    const id = cells[1];
    if (reqs.has(id)) fail('spec', `بند تکراری: ${id}`);
    reqs.set(id, cells.slice(2, -1).pop() ?? '');
  }
  if (reqs.size === 0) fail('spec', 'هیچ بند شناسه‌داری در سند یافت نشد');

  // ---------- آزمون‌های سرور ----------
  const goTests = new Map();
  const testPattern = new RegExp(`func (Test${prefix}(\\d+)[A-Za-z0-9_]*)\\(`, 'g');
  for (const file of walk(join(ROOT, 'server')).filter(f => f.endsWith('_test.go'))) {
    const body = readFileSync(file, 'utf8');
    for (const m of body.matchAll(testPattern)) {
      goTests.set(m[1], { file: rel(file), id: `${prefix}-${m[2]}` });
    }
  }

  // ---------- سنجش تأیید هر بند ----------
  for (const [id, cell] of reqs) {
    const tokens = cell.split(/[،,]/).map(t => t.trim()).filter(Boolean);
    if (tokens.length === 0) {
      fail('verify', `${id} هیچ تأییدی ندارد`);
      continue;
    }
    for (const token of tokens) {
      const sep = token.indexOf(':');
      if (sep < 0) {
        fail('verify', `${id}: تأیید «${token}» شکل درست (go:/tool:/file:/doc:) ندارد`);
        continue;
      }
      const kind = token.slice(0, sep);
      const rest = token.slice(sep + 1);

      if (kind === 'go') {
        if (!goTests.has(rest)) fail('verify', `${id}: آزمون «${rest}» در server/ یافت نشد`);
      } else if (kind === 'tool') {
        const file = join(ROOT, 'tools', 'check', rest.endsWith('.mjs') ? rest : `${rest}.mjs`);
        if (!existsSync(file)) fail('verify', `${id}: ابزار «${rel(file)}» یافت نشد`);
      } else if (kind === 'file') {
        if (!existsSync(join(ROOT, rest))) fail('verify', `${id}: فایل «${rest}» یافت نشد`);
      } else if (kind === 'doc') {
        const [filePart, keyword = ''] = rest.split('#');
        const file = join(ROOT, filePart);
        if (!existsSync(file)) {
          fail('verify', `${id}: سند «${filePart}» یافت نشد`);
          continue;
        }
        if (keyword && !normalize(readFileSync(file, 'utf8')).includes(normalize(keyword))) {
          fail('verify', `${id}: سند «${filePart}» کلیدواژهٔ «${keyword}» را ندارد`);
        }
      } else {
        fail('verify', `${id}: نوع تأیید ناشناخته «${kind}»`);
      }
    }
  }

  // ---------- آزمون یتیم ----------
  for (const [name, meta] of goTests) {
    if (!reqs.has(meta.id)) fail('orphan', `آزمون ${name} (${meta.file}) بند ${meta.id} در اسپک ندارد`);
  }

  notes.push(`${label}: ${reqs.size} بند، ${goTests.size} آزمون شناسه‌دار`);
  return { failures, notes, reqs: reqs.size, tests: goTests.size };
}

/** runSpecGate دروازه را اجرا و نتیجه را چاپ می‌کند؛ کد خروج ۱ یعنی شکست. */
export function runSpecGate(options) {
  const { failures, notes } = checkSpec(options);
  if (failures.length) {
    console.error(`✖ ${options.label} شکست خورد:`);
    for (const f of failures) console.error(`  - ${f}`);
    process.exit(1);
  }
  console.log(`✔ ${options.label} هم‌خوان است`);
  for (const n of notes) console.log(`  - ${n}`);
}