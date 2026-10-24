/**
 * دروازه‌های بررسی (جایگزین ابزارهای سنگین، بدون وابستگی خارجی):
 *  ۱. اعتبار مانیفست‌ها و هم‌خوانی خروجی‌های تولیدشده (انحراف قرارداد)
 *  ۲. وجود هر کامپوننت صفحه در اپ React Native
 *  ۳. مرزهای معماری: هیچ ماژول Go داخلی‌های ماژول دیگر را import نکند
 *  ۴. انحراف قرارداد: هر پیشوند API مانیفست در OpenAPI وجود داشته باشد و
 *     هر نام فیلد schema در DTO متناظر Go به‌صورت json tag موجود باشد
 * استعمال: node tools/check/verify.mjs
 */
import { readFileSync, existsSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';
import { loadStructure, buildArtifacts, orgEnabledModules, FEATURES_DIR, ROOT, ROLES } from '../gen/lib.mjs';

const failures = [];
const notes = [];
const fail = (gate, msg) => failures.push(`[${gate}] ${msg}`);

const walk = dir => {
  if (!existsSync(dir)) return [];
  const out = [];
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, e.name);
    if (e.isDirectory()) out.push(...walk(p));
    else out.push(p);
  }
  return out;
};
const rel = p => relative(ROOT, p).replace(/\\/g, '/');

// ---------- ۱. مانیفست + انحراف خروجی‌های تولیدشده ----------
const structure = loadStructure();
const { departments, directorates, modules, organizations } = structure;
let screenCount = 0;
for (const e of structure.errors) fail('structure', e);

for (const [path, content] of buildArtifacts(structure)) {
  const current = existsSync(path) ? readFileSync(path, 'utf8') : null;
  if (current !== content) fail('drift', `${rel(path)} با مانیفست‌ها هم‌خوان نیست — «node tools/gen/gen.mjs» را اجرا کنید`);
}
notes.push(`ساختار: ${directorates.length} حوزه، ${departments.length} اداره، ${modules.length} ماژول، ${modules.reduce((n, m) => n + m.screens.length, 0)} صفحه`);
notes.push(`استقرار پارامتریک: ${organizations.length} شرکت — فعال: ${structure.activeOrg?.titleFa ?? '—'} (${orgEnabledModules(structure.activeOrg, modules).length} ماژول)`);
for (const o of organizations) {
  if (orgEnabledModules(o, modules).length === 0) fail('orgs', `شرکت «${o.titleFa}» هیچ ماژول فعالی ندارد`);
}
notes.push(`زیرواحدها: ${departments.reduce((n, d) => n + (d.subUnits ?? []).length, 0)} — ماژول مشترک: ${modules.filter(m => (m.sharedWith ?? []).length).map(m => m.id).join('، ') || 'ندارد'}`);

// ---------- ۲. وجود کامپوننت صفحه در RN ----------
let componentFiles = 0;
for (const m of modules) {
  for (const s of m.screens) {
    screenCount++;
    const expected = join(FEATURES_DIR, m.id, `${s.component}.tsx`);
    if (!existsSync(expected)) fail('rn-screen', `کامپوننت «${s.component}» در apps/mobile/src/features/${m.id}/ ساخته نشده است`);
    else componentFiles++;
  }
}

// ---------- ۲ب. ثبت شدن هر کامپوننت در رجیستری ناوبری RN ----------
//
// وجود فایل کافی نیست: صفحه‌ای که در `SCREEN_REGISTRY` ثبت نشده باشد در عمل
// غیرقابل‌رسیدن است و کاربر با خطای «صفحه یافت نشد» روبه‌رو می‌شود. این دروازه
// هر سه شرط را می‌گیرد: وجود فایل، وجود import و وجود سطر ثبت در ثابت رجیستری.
const registryFile = join(ROOT, 'apps', 'mobile', 'src', 'navigation', 'screenRegistry.ts');
const registrySrc = existsSync(registryFile) ? readFileSync(registryFile, 'utf8') : '';
let registeredScreens = 0;
for (const m of modules) {
  for (const s of m.screens) {
    const imports = [...registrySrc.matchAll(new RegExp(`^import\\s+${s.component}\\s+from`, 'gm'))].length;
    if (imports === 0) {
      fail('rn-registry', `کامپوننت «${s.component}» در screenRegistry.ts وارد نشده است`);
      continue;
    }
    if (imports > 1) {
      fail('rn-registry', `کامپوننت «${s.component}» ${imports} بار import شده است`);
      continue;
    }
    if (!new RegExp(`^\\s+${s.component},\\s*$`, 'm').test(registrySrc)) {
      fail('rn-registry', `کامپوننت «${s.component}» در ثابت SCREEN_REGISTRY ثبت نشده است`);
      continue;
    }
    registeredScreens++;
  }
}
notes.push(`رجیستری ناوبری RN: ${registeredScreens}/${screenCount} کامپوننت ثبت‌شده`);

// ---------- ۳. مرزهای معماری Go ----------
const modulesRoot = join(ROOT, 'server', 'internal', 'modules');
const violations = [];
for (const file of walk(modulesRoot).filter(f => f.endsWith('.go'))) {
  const relPath = rel(file);
  const owner = relPath.split('/')[3]; // server/internal/modules/<owner>/...
  if (!modulesRoot || !owner) continue;
  const src = readFileSync(file, 'utf8');
  const imports = [...src.matchAll(/^\s*(?:[\w.]+\s+)?"(hrms\/internal\/modules\/([a-z0-9_]+)[^"]*)"/gm)];
  for (const [, full, target] of imports) {
    if (target !== owner) violations.push(`${relPath} → ${full} (مالکیت: ${owner})`);
  }
}
for (const v of violations) fail('arch', `نقض مرز ماژول: ${v}`);
notes.push(`مرز معماری: ${walk(modulesRoot).filter(f => f.endsWith('.go')).length} فایل Go ماژول بررسی شد`);

// ---------- ۴. انحراف قرارداد با OpenAPI ----------
const specPath = join(ROOT, 'contracts', 'openapi', 'hrms.json');
const spec = JSON.parse(readFileSync(specPath, 'utf8'));
const specPaths = Object.keys(spec.paths);

for (const m of modules) {
  if (!specPaths.some(p => p === m.apiPrefix || p.startsWith(`${m.apiPrefix}/`))) {
    fail('openapi', `ماژول «${m.id}» هیچ مسیری با پیشوند ${m.apiPrefix} در قرارداد ندارد`);
  }
}

// هر نام فیلد schema باید در DTO متناظر Go به‌صورت json tag باشد.
const dtoTargets = {
  LoginRequest: 'server/internal/contracts/auth.go',
  Session: 'server/internal/contracts/auth.go',
  User: 'server/internal/contracts/auth.go',
  Employee: 'server/internal/contracts/employee.go',
  EmployeeInput: 'server/internal/contracts/employee.go',
  EmployeePage: 'server/internal/contracts/employee.go',
  Department: 'server/internal/contracts/employee.go',
  Widget: 'server/internal/contracts/dashboard.go',
  ChangeItem: 'server/internal/contracts/sync.go',
  ChangeSet: 'server/internal/contracts/sync.go',
  Mutation: 'server/internal/contracts/sync.go',
  RejectedMutation: 'server/internal/contracts/sync.go',
  MutationResult: 'server/internal/contracts/sync.go',
  // قلم کاتالوگ مرجع (مدیریت/بانک/واحد) که پنل تنظیمات می‌سازد و وام و پاداش مصرف می‌کنند.
  CatalogItem: 'server/internal/contracts/catalogs.go',
  Error: 'server/internal/contracts/errors.go',
};
const schemas = spec.components.schemas;
for (const [name, target] of Object.entries(dtoTargets)) {
  const schema = schemas[name];
  if (!schema) { fail('openapi', `schema «${name}» در قرارداد نیست`); continue; }
  const goPath = join(ROOT, target);
  if (!existsSync(goPath)) { fail('dto', `DTO متناظر «${name}» ساخته نشده: ${target}`); continue; }
  const src = readFileSync(goPath, 'utf8');
  for (const prop of Object.keys(schema.properties ?? {})) {
    if (!new RegExp(`json:"${prop}(,|")`, 'm').test(src)) {
      fail('dto', `فیلد «${name}.${prop}» در ${target} به‌صورت json tag وجود ندارد`);
    }
  }
}

// نقش‌های قرارداد و مانیفست باید یکی باشند
const specRoleEnum = schemas.User.properties.role.enum;
if (JSON.stringify([...specRoleEnum].sort()) !== JSON.stringify([...ROLES].sort())) {
  fail('openapi', `نقش‌های قرارداد با مانیفست هم‌خوان نیست: ${specRoleEnum.join(',')}`);
}

// ---------- گزارش ----------
console.log(`ماژول: ${modules.length} | صفحه: ${screenCount} | کامپوننت RN یافته‌شده: ${componentFiles}`);
for (const n of notes) console.log(`· ${n}`);
if (failures.length) {
  console.error(`\n✖ ${failures.length} ایراد یافت شد:`);
  for (const f of failures) console.error(`  - ${f}`);
  process.exit(1);
}
console.log('\n✓ همهٔ دروازه‌ها سبز است.');
