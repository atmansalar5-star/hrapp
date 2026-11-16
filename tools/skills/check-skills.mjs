#!/usr/bin/env node
/**
 * check-skills.mjs — اعتبارسنجی کتابخانهٔ اسکیل‌های پروژه بر پایهٔ استاندارد Agent Skills.
 *
 * چرا این ابزار: اسکیل فقط وقتی کار می‌کند که پیشوند YAML سالم و «description» گویا باشد؛
 * خطای تایپی در نام یا توضیح، اسکیل را بی‌صدا از دسترس عامل بیرون می‌برد. این ابزار همان
 * قاعده‌های استاندارد را می‌سنجد:
 *
 *   name          لازم، ≤۶۴ نویسه، فقط [a-z0-9-]، بدون خط تیره در آغاز/پایان، هم‌نام پوشه
 *   description   لازم، غیرخالی، ≤۱۰۲۴ نویسه
 *   license       اختیاری
 *   compatibility اختیاری، ≤۵۰۰ نویسه
 *   metadata      اختیاری، نقشهٔ رشته→رشته
 *   allowed-tools اختیاری، رشته
 *   پیکرهٔ متن    نباید خالی باشد
 *
 * اجرا: node tools/skills/check-skills.mjs [--dir .claude/skills]
 */
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, basename } from 'node:path';
import { existsSync } from 'node:fs';

const argv = process.argv.slice(2);
const dirFlag = argv.indexOf('--dir');
const ROOT = dirFlag >= 0 && argv[dirFlag + 1] ? argv[dirFlag + 1] : '.claude/skills';

const REQUIRED = ['name', 'description'];
const OPTIONAL = ['license', 'compatibility', 'metadata', 'allowed-tools'];
const NAME_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/;

const errors = [];
const warnings = [];
const skills = [];

/** خواندن پیشوند YAML بین دو خط «---» (نگاشت سادهٔ کلید: مقدار در سطح اول). */
function parseFrontmatter(text) {
  const lines = text.replace(/\r\n/g, '\n').split('\n');
  if (lines[0].trim() !== '---') return { ok: false, reason: 'فایل با خط «---» آغاز نمی‌شود' };
  const end = lines.findIndex((l, i) => i > 0 && l.trim() === '---');
  if (end < 0) return { ok: false, reason: 'خط پایان پیشوند («---») پیدا نشد' };
  const fm = {};
  const nested = {};
  let currentKey = null;
  for (const raw of lines.slice(1, end)) {
    if (!raw.trim() || raw.trim().startsWith('#')) continue;
    const indent = raw.match(/^\s*/)[0].length;
    const line = raw.trim();
    const colon = line.indexOf(':');
    if (colon < 0) continue;
    const key = line.slice(0, colon).trim();
    const value = line.slice(colon + 1).trim();
    if (indent > 0 && currentKey) {
      nested[currentKey] = nested[currentKey] || {};
      nested[currentKey][key] = value;
      continue;
    }
    currentKey = key;
    fm[key] = value;
  }
  const body = lines.slice(end + 1).join('\n').trim();
  return { ok: true, fm, nested, body };
}

/** بررسی یک پوشهٔ اسکیل. */
function checkSkill(name, dir) {
  const file = join(dir, 'SKILL.md');
  let text;
  try {
    text = readFileSync(file, 'utf8');
  } catch {
    errors.push(`${name}: فایل SKILL.md ندارد`);
    return;
  }
  const parsed = parseFrontmatter(text);
  if (!parsed.ok) {
    errors.push(`${name}: ${parsed.reason}`);
    return;
  }
  const { fm, nested, body } = parsed;

  for (const key of REQUIRED) {
    if (!fm[key]) errors.push(`${name}: کلید لازم «${key}» خالی یا نبود`);
  }
  for (const key of Object.keys(fm)) {
    if (!REQUIRED.includes(key) && !OPTIONAL.includes(key)) {
      warnings.push(`${name}: کلید ناشناخته در پیشوند «${key}» (استاندارد آن را نمی‌شناسد)`);
    }
  }
  if (fm.name) {
    if (fm.name.length > 64) errors.push(`${name}: نام بلندتر از ۶۴ نویسه است`);
    if (!NAME_RE.test(fm.name)) errors.push(`${name}: نام باید فقط [a-z0-9-] و بدون خط تیره در دو سر باشد`);
    if (fm.name !== name) errors.push(`${name}: نام پیشوند («${fm.name}») با نام پوشه یکی نیست`);
  }
  if (fm.description) {
    if (fm.description.length > 1024) errors.push(`${name}: توضیح بلندتر از ۱۰۲۴ نویسه است`);
    if (fm.description.length < 40) warnings.push(`${name}: توضیح کوتاه است؛ «چه کاری» و «کِی» را روشن بگو`);
  }
  if (fm.compatibility && fm.compatibility.length > 500) {
    errors.push(`${name}: مقدار compatibility بلندتر از ۵۰۰ نویسه است`);
  }
  if (fm.metadata && Object.keys(nested.metadata || {}).length === 0 && !/^\{\s*\}/.test(fm.metadata)) {
    warnings.push(`${name}: metadata باید نقشهٔ کلید:مقدار باشد`);
  }
  if (!body) errors.push(`${name}: پیکرهٔ دستورها خالی است`);

  // ارجاع به فایل‌های همراه (references/…، scripts/…، assets/…) باید واقعاً وجود داشته باشد؛
  // ارجاع مرده، عامل را به فایل ناموجود می‌فرستد و همان‌جا متوقف می‌کند.
  const refs = [...text.matchAll(/(?:^|[^\w/])((?:references|scripts|assets)\/[\w.\-\u0600-\u06FF/]+)/g)].map(m => m[1]);
  for (const ref of new Set(refs)) {
    if (!existsSync(join(dir, ref))) warnings.push(`${name}: ارجاع به فایل ناموجود «${ref}»`);
  }

  skills.push({ name, descriptionLength: (fm.description || '').length, bodyLines: body.split('\n').length });
}

let entries;
try {
  entries = readdirSync(ROOT, { withFileTypes: true });
} catch {
  console.error(`✖ پوشهٔ اسکیل‌ها پیدا نشد: ${ROOT}`);
  process.exit(2);
}

for (const e of entries) {
  const abs = join(ROOT, e.name);
  if (!e.isDirectory()) continue;              // README و فایل‌های سطح بالا اسکیل نیستند
  if (e.name.startsWith('.')) continue;
  if (!statSync(abs).isDirectory()) continue;
  checkSkill(e.name, abs);
}

if (skills.length === 0) errors.push(`هیچ اسکیلی در ${ROOT} پیدا نشد`);

console.log(`اسکیل‌های بررسی‌شده: ${skills.length}`);
for (const s of skills) console.log(`  ✓ ${s.name.padEnd(24)} توضیح: ${String(s.descriptionLength).padStart(4)} نویسه · بدنه: ${s.bodyLines} خط`);
for (const w of warnings) console.warn(`! ${w}`);
for (const e of errors) console.error(`✖ ${e}`);

if (errors.length) {
  console.error(`✖ ${errors.length} خطا — کتابخانهٔ اسکیل معتبر نیست.`);
  process.exit(1);
}
console.log(`✓ همهٔ اسکیل‌ها با استاندارد Agent Skills هم‌خوان‌اند${warnings.length ? ` (${warnings.length} هشدار)` : ''}.`);