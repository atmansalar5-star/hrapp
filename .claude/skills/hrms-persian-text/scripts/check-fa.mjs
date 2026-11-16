#!/usr/bin/env node
/**
 * check-fa.mjs — سنجش سالم‌بودن متن فارسی در فایل‌های یک پروژه.
 *
 * چرا: در این پروژه یک‌بار بازنویسی از راه خط فرمان، کدگذاری فایل های فارسی را دو بار عوض
 * کرد و کل رابط ناخوانا شد. نشانهٔ خرابی ساده است: شمار نویسه‌های فارسی صفر شده یا نویسهٔ
 * «Ã» (نشانهٔ موجی‌بکه) زیاد شده یا BOM اضافه شده است.
 *
 * اجرا:
 *   node .claude/skills/hrms-persian-text/scripts/check-fa.mjs            # کل مخزن (فایل‌های متنی)
 *   node .claude/skills/hrms-persian-text/scripts/check-fa.mjs src docs   # فقط این مسیرها
 *   ... --quiet                                                          # فقط خطاها
 *
 * خروجی: کد ۰ اگر همه سالم باشند؛ ۱ اگر فایلی مشکل داشته باشد.
 */
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

const argv = process.argv.slice(2);
const quiet = argv.includes('--quiet');
const roots = argv.filter(a => !a.startsWith('--'));
const targets = roots.length ? roots : ['.'];

const EXT = new Set(['.md', '.html', '.css', '.js', '.mjs', '.ts', '.tsx', '.json', '.go', '.rs', '.sql', '.yml', '.yaml', '.txt', '.bat', '.cmd', '.ps1']);
const SKIP_DIR = new Set(['.git', 'node_modules', 'target', 'dist', 'dist-pentest', '.freebuff', '.dsh-tmp', 'bin', 'tmp', 'out', 'build']);

function walk(p, out = []) {
  let entries;
  try {
    entries = readdirSync(p, { withFileTypes: true });
  } catch {
    return out;
  }
  for (const e of entries) {
    if (e.isDirectory()) {
      if (SKIP_DIR.has(e.name)) continue;
      walk(join(p, e.name), out);
      continue;
    }
    const dot = e.name.lastIndexOf('.');
    const ext = dot >= 0 ? e.name.slice(dot).toLowerCase() : '';
    if (EXT.has(ext)) out.push(join(p, e.name));
  }
  return out;
}

const files = targets.flatMap(t => {
  try {
    return statSync(t).isDirectory() ? walk(t) : [t];
  } catch {
    console.error(`✖ مسیر پیدا نشد: ${t}`);
    process.exit(2);
  }
});

const problems = [];
const warnings = [];
let withPersian = 0;
for (const abs of files) {
  const buf = readFileSync(abs);
  const rel = relative(process.cwd(), abs).split('\\').join('/');
  // فایل‌های دوتایی/غیرمتنی
  if (buf.includes(0)) continue;
  let text;
  try {
    text = buf.decode?.() ?? buf.toString('utf8');
  } catch {
    problems.push({ rel, reason: 'UTF-8 معتبر نیست' });
    continue;
  }
  const fa = (text.match(/[\u0600-\u06FF]/g) || []).length;
  if (fa === 0) continue;                       // فایل فارسی ندارد؛ موضوع سنجش نیست
  withPersian++;
  const mojibake = (text.match(/\u00c3/g) || []).length;
  const bom = buf[0] === 0xef && buf[1] === 0xbb && buf[2] === 0xbf;
  const replacement = (text.match(/\ufffd/g) || []).length; // U+FFFD نشانهٔ بایت نامعتبر
  // نسبت موجی‌بکه: در متن فارسی سالم تقریباً صفر است (مگر فایلی که خودِ نشانه را توضیح می‌دهد)
  const ratio = mojibake / fa;
  // BOM در اسکریپت‌های PowerShell/CMD لازم است (PowerShell 5.1 بدون BOM فایل را ANSI می‌خواند
  // و فارسی را خراب نشان می‌دهد)؛ در بقیهٔ فایل‌ها BOM ممنوع است.
  const extName = (rel.slice(rel.lastIndexOf('.')) || '').toLowerCase();
  const psLike = extName === '.ps1' || extName === '.cmd' || extName === '.bat';
  if (bom && !psLike) problems.push({ rel, reason: 'BOM دارد (باید UTF-8 بدون BOM باشد)' });
  if (!bom && psLike && fa > 0) warnings.push({ rel, reason: 'اسکریپت PowerShell بدون BOM؛ در PowerShell 5.1 فارسی ممکن است ANSI خوانده شود' });
  if (replacement > 0) problems.push({ rel, reason: `${replacement} نویسهٔ جایگزین (بایت نامعتبر)` });
  if (ratio > 0.02) problems.push({ rel, reason: `نسبت موجی‌بکه بالا: ${mojibake} نشانه در ${fa} نویسهٔ فارسی` });
  else if (!quiet) console.log(`✓ ${rel}  (فارسی: ${fa}${mojibake ? `، «Ã»: ${mojibake} — احتمالاً توضیح خودِ نشانه` : ''})`);
}

if (!quiet) console.log(`\nفایل‌های دارای متن فارسی: ${withPersian} از ${files.length} فایل متنی`);
for (const w of warnings) console.warn(`! ${w.rel}: ${w.reason}`);
for (const p of problems) console.error(`✖ ${p.rel}: ${p.reason}`);
if (problems.length) {
  console.error(`\n✖ ${problems.length} فایل مشکل دارد.`);
  process.exit(1);
}
console.log('✓ کدگذاری متن فارسی همهٔ فایل‌ها سالم است.');