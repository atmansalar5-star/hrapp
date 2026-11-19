/**
 * دروازهٔ «ratchet» امنیت فرانت‌اند:
 * شمار نقاط پرخطر رابط نمونه را می‌شمارد و اگر از سقف ثبت‌شده در
 * docs/security-spec-fa.md بیشتر شود، شکست می‌دهد. قاعده: سقف فقط پایین می‌آید.
 *
 *   innerHTML = N   → درج رشتهٔ HTML (خطر XSS اگر دادهٔ سرور فرار نکند)
 *   handlers  = N   → صفت on* تولیدشده در رشتهٔ HTML (دلیل script-src-attr 'unsafe-inline')
 *
 * همچنین می‌سنجد توکن نشست در localStorage نگه داشته نمی‌شود.
 *
 * استعمال: node tools/check/frontend-security.mjs
 */
import { readFileSync, readdirSync, existsSync } from 'node:fs';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const JS_DIR = join(ROOT, 'design', 'prototype', 'js');
const SPEC = join(ROOT, 'docs', 'security-spec-fa.md');

const failures = [];
const rel = p => relative(ROOT, p).replace(/\\/g, '/');

if (!existsSync(JS_DIR)) {
  console.error(`✖ پوشهٔ JS رابط یافت نشد: ${rel(JS_DIR)}`);
  process.exit(1);
}

let innerHTML = 0;
let handlers = 0;
let perFile = [];
for (const name of readdirSync(JS_DIR)) {
  if (!name.endsWith('.js')) continue;
  const file = join(JS_DIR, name);
  const text = readFileSync(file, 'utf8');
  const inner = (text.match(/innerHTML/g) ?? []).length;
  const on = (text.match(/\son(click|change|submit|input|load|keyup|keydown|error)\s*=/g) ?? []).length;
  innerHTML += inner;
  handlers += on;
  if (inner || on) perFile.push({ file: name, inner, on });
  // توکن نشست هرگز نباید در localStorage بنشیند (با XSS خواندنی است).
  if (/localStorage\.setItem\([^\n]*token/i.test(text) || /localStorage\.setItem\(\s*['"][^'"]*token/i.test(text)) {
    failures.push(`[token] ${name} توکن نشست را در localStorage می‌نویسد`);
  }
}

const spec = readFileSync(SPEC, 'utf8');
const base = spec.match(/<!--\s*frontend-baseline:\s*innerhtml=(\d+)\s+handlers=(\d+)\s*-->/);
if (!base) {
  failures.push('[spec] سطر سقف فرانت‌اند (frontend-baseline) در اسپک نیست');
} else {
  const [maxInner, maxHandlers] = [Number(base[1]), Number(base[2])];
  if (innerHTML > maxInner) {
    failures.push(`[ratchet] innerHTML از سقف گذشته: ${innerHTML} > ${maxInner} (نقطهٔ تازه را با textContent/escape ببندید)`);
  }
  if (handlers > maxHandlers) {
    failures.push(`[ratchet] هندلر درون‌خطی از سقف گذشته: ${handlers} > ${maxHandlers} (به data-act منتقل کنید)`);
  }
  if (innerHTML < maxInner || handlers < maxHandlers) {
    console.log(`↓ سقف را پایین بیاورید: <!-- frontend-baseline: innerhtml=${innerHTML} handlers=${handlers} -->`);
  }
}

perFile.sort((a, b) => b.inner + b.on - (a.inner + a.on));
for (const row of perFile.slice(0, 5)) {
  console.log(`  - ${row.file}: innerHTML=${row.inner} on*=${row.on}`);
}

if (failures.length) {
  console.error('✖ دروازهٔ امنیت فرانت‌اند شکست خورد:');
  for (const f of failures) console.error(`  - ${f}`);
  process.exit(1);
}
console.log(`✔ فرانت‌اند در سقف است: innerHTML=${innerHTML}، هندلر درون‌خطی=${handlers}`);