#!/usr/bin/env node
/**
 * تولید همهٔ خروجی‌های مشتق‌شده از مانیفست ماژول‌ها.
 * استعمال: node tools/gen/gen.mjs [--check]
 *   --check  فقط بررسی می‌کند که فایل‌های تولیدشده به‌روز هستند (دروازهٔ CI) و چیزی نمی‌نویسد.
 */
import { mkdirSync, writeFileSync, readFileSync, existsSync } from 'node:fs';
import { dirname, relative } from 'node:path';
import { loadStructure, buildArtifacts, ROOT } from './lib.mjs';

const check = process.argv.includes('--check');
const structure = loadStructure();
const { departments, modules, organizations } = structure;
const errors = structure.errors;

if (errors.length) {
  console.error(`\n✖ اعتبارسنجی مانیفست‌ها شکست خورد (${errors.length} خطا):`);
  for (const e of errors) console.error(`  - ${e}`);
  process.exit(1);
}

const artifacts = buildArtifacts(structure);
let drift = 0;

for (const [path, content] of artifacts) {
  const rel = relative(ROOT, path).replace(/\\/g, '/');
  if (check) {
    const current = existsSync(path) ? readFileSync(path, 'utf8') : null;
    if (current !== content) {
      drift++;
      console.error(`✖ منحرف: ${rel}${current === null ? ' (وجود ندارد)' : ' (محتوا به‌روز نیست)'}`);
    } else {
      console.log(`✓ به‌روز: ${rel}`);
    }
  } else {
    mkdirSync(dirname(path), { recursive: true });
    writeFileSync(path, content, 'utf8');
    console.log(`→ نوشته شد: ${rel}`);
  }
}

if (check && drift) {
  console.error('\n✖ دروازهٔ انحراف قرارداد شکست خورد: مانیفست‌ها تغییر کرده‌اند ولی خروجی‌ها بازتولید نشده‌اند. «npm run gen» را اجرا کنید.');
  process.exit(1);
}

const summary = `${departments.length} اداره، ${modules.length} ماژول، ${modules.reduce((n, m) => n + m.screens.length, 0)} صفحه و ${organizations.length} استقرار شرکت`;
if (check) console.log(`\n✓ همهٔ ${artifacts.size} خروجی با ${summary} هم‌خوان است.`);
else console.log(`\n✓ ${artifacts.size} خروجی از ${summary} ساخته شد.`);
