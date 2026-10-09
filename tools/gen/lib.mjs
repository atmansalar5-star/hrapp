/**
 * منبع واحد حقیقت: مانیفست ماژول‌ها.
 * این کتابخانه مانیفست‌ها را می‌خواند، اعتبارسنجی می‌کند و همهٔ خروجی‌های مشتق‌شده را می‌سازد:
 *   registry.gen.go   → رجیستری ماژول در Go
 *   modules.generated.ts → درخت ناوبری React Native
 *   000_seed_rbac.sql → دسترسی‌های نقش‌محور
 *   modules.gen.json  → قرارداد مصرفی ابزارها
 *   pages-fa.md       → مستندات فارسی هر صفحه (دستور ۵-۵ سند طراحی)
 * بدون هیچ وابستگی خارجی — فقط Node.
 */
import { readdirSync, readFileSync, existsSync } from 'node:fs';
import { join, resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

export const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..');
export const MODULES_DIR = join(ROOT, 'server', 'internal', 'modules');

export const ROLES = ['admin', 'hr_manager', 'hr_spec', 'line_manager', 'employee'];
export const ACTIONS = ['read', 'write', 'delete', 'approve', 'report', 'admin'];
export const SCOPES = ['all', 'own_unit', 'own'];
export const FEATURES_DIR = join(ROOT, 'apps', 'mobile', 'src', 'features');
export const DEPARTMENTS_FILE = join(ROOT, 'contracts', 'departments.json');
export const ORGANIZATIONS_FILE = join(ROOT, 'contracts', 'organizations.json');
export const PROTOTYPE_JS_DIR = join(ROOT, 'design', 'prototype', 'js');
export const MOBILE_CONFIG_DIR = join(ROOT, 'apps', 'mobile', 'src', 'config');
export const ORGS_GO_DIR = join(ROOT, 'server', 'internal', 'platform', 'orgs');

export const ROLE_FA = {
  admin: 'مدیر سیستم',
  hr_manager: 'مدیر منابع انسانی',
  hr_spec: 'کارشناس منابع انسانی',
  line_manager: 'مدیر واحد',
  employee: 'کارمند',
};

const isStr = (v, min = 1) => typeof v === 'string' && v.length >= min;
const isInt = v => Number.isInteger(v);
const isList = (v, min = 1) => Array.isArray(v) && v.length >= min;

/**
 * بارگذاری و اعتبارسنجی ساختار سازمانی منو.
 * سه سطح: حوزه (directorate) ← اداره (department) ← زیرواحد (subUnit).
 */
export function loadDepartments() {
  const empty = { departments: [], directorates: [], sharedGroupTitleFa: 'مشترک با سایر واحدها', errors: [] };
  if (!existsSync(DEPARTMENTS_FILE)) return { ...empty, errors: [`فایل ادارات یافت نشد: ${DEPARTMENTS_FILE}`] };
  let raw;
  try {
    raw = JSON.parse(readFileSync(DEPARTMENTS_FILE, 'utf8'));
  } catch (err) {
    return { ...empty, errors: [`departments.json نامعتبر — ${err.message}`] };
  }
  const errors = [];

  // ---- حوزه‌های سازمانی ----
  const dirs = raw.directorates;
  if (!isList(dirs)) errors.push('فهرست حوزه‌ها (directorates) خالی است');
  const dirAllowed = ['id', 'order', 'titleFa', 'titleEn', 'dutiesFa'];
  const seenDirs = new Set();
  for (const [i, d] of (dirs ?? []).entries()) {
    const tag = `directorates[${i}]`;
    if (!/^[a-z][a-z0-9_]*$/.test(d.id ?? '')) errors.push(`${tag}: id نامعتبر`);
    if (seenDirs.has(d.id)) errors.push(`${tag}: id تکراری «${d.id}»`);
    seenDirs.add(d.id);
    for (const k of Object.keys(d)) if (!dirAllowed.includes(k)) errors.push(`${tag}: کلید ناشناخته «${k}»`);
    if (!isInt(d.order)) errors.push(`${tag}: order باید عدد صحیح باشد`);
    if (!isStr(d.titleFa, 2)) errors.push(`${tag}: titleFa نامعتبر`);
    if (!isStr(d.titleEn, 2)) errors.push(`${tag}: titleEn نامعتبر`);
    if (!isList(d.dutiesFa)) errors.push(`${tag}: dutiesFa خالی است`);
  }
  const directorates = [...(dirs ?? [])].sort((a, b) => a.order - b.order);
  directorates.forEach((d, i) => { if (d.order !== i + 1) errors.push(`ترتیب حوزه‌ها پیوسته نیست: ${d.id} order=${d.order}`); });

  // ---- ادارات و زیرواحدها ----
  const list = raw.departments;
  if (!isList(list)) return { ...empty, directorates, errors: [...errors, 'فهرست ادارات خالی است'] };

  const allowed = ['id', 'order', 'titleFa', 'titleEn', 'dutiesFa', 'directorate', 'subUnits'];
  const subAllowed = ['id', 'order', 'titleFa', 'dutiesFa'];
  const seen = new Set();
  for (const [i, d] of list.entries()) {
    const tag = `departments[${i}]`;
    if (!/^[a-z][a-z0-9_]*$/.test(d.id ?? '')) errors.push(`${tag}: id نامعتبر`);
    if (seen.has(d.id)) errors.push(`${tag}: id تکراری «${d.id}»`);
    seen.add(d.id);
    for (const k of Object.keys(d)) if (!allowed.includes(k)) errors.push(`${tag}: کلید ناشناخته «${k}»`);
    if (!isInt(d.order)) errors.push(`${tag}: order باید عدد صحیح باشد`);
    if (!isStr(d.titleFa, 2)) errors.push(`${tag}: titleFa نامعتبر`);
    if (!isStr(d.titleEn, 2)) errors.push(`${tag}: titleEn نامعتبر`);
    if (!isList(d.dutiesFa)) errors.push(`${tag}: dutiesFa خالی است (شرح وظایف اداره)`);
    if (!seenDirs.has(d.directorate)) errors.push(`${tag}: حوزهٔ ناشناخته «${d.directorate}» — این کلید در directorates نیست`);
    if (d.subUnits !== undefined && !Array.isArray(d.subUnits)) errors.push(`${tag}: subUnits باید آرایه باشد`);
    const seenSubs = new Set();
    for (const [j, s] of (d.subUnits ?? []).entries()) {
      const stag = `${tag}.subUnits[${j}]`;
      if (!/^[a-z][a-z0-9_]*$/.test(s.id ?? '')) errors.push(`${stag}: id نامعتبر`);
      if (seenSubs.has(s.id)) errors.push(`${stag}: id تکراری «${s.id}»`);
      seenSubs.add(s.id);
      for (const k of Object.keys(s)) if (!subAllowed.includes(k)) errors.push(`${stag}: کلید ناشناخته «${k}»`);
      if (!isInt(s.order)) errors.push(`${stag}: order باید عدد صحیح باشد`);
      if (!isStr(s.titleFa, 2)) errors.push(`${stag}: titleFa نامعتبر`);
      if (!isList(s.dutiesFa)) errors.push(`${stag}: dutiesFa خالی است`);
    }
    if ((d.subUnits ?? []).length) {
      const subs = [...d.subUnits].sort((a, b) => a.order - b.order);
      subs.forEach((s, j) => { if (s.order !== j + 1) errors.push(`ترتیب زیرواحدهای «${d.id}» پیوسته نیست: ${s.id} order=${s.order}`); });
    }
  }
  const departments = [...list].sort((a, b) => a.order - b.order);
  departments.forEach((d, i) => { if (d.order !== i + 1) errors.push(`ترتیب ادارات پیوسته نیست: ${d.id} order=${d.order}`); });
  const sharedGroupTitleFa = isStr(raw.sharedGroupTitleFa, 2) ? raw.sharedGroupTitleFa : 'مشترک با سایر واحدها';
  return { departments, directorates, sharedGroupTitleFa, errors };
}

/** اعتبارسنجی یک مانیفست. خروجی: آرایه‌ای از پیام‌های خطا. */
export function validateManifest(m, dirName, departments = []) {
  const e = [];
  const req = ['id', 'order', 'department', 'titleFa', 'titleEn', 'apiPrefix', 'version', 'featureFlag', 'dbSchema', 'owner', 'entities', 'roles', 'scopes', 'screens'];
  const optional = ['subUnit', 'sharedWith'];
  const dept = departments.find(d => d.id === m.department);
  const subUnitIds = (dept?.subUnits ?? []).map(s => s.id);
  if (subUnitIds.length) {
    if (!isStr(m.subUnit) || !subUnitIds.includes(m.subUnit))
      e.push(`${dirName}: subUnit نامعتبر «${m.subUnit}» — زیرواحدهای ادارهٔ «${m.department}»: ${subUnitIds.join('، ')}`);
  } else {
    if (m.subUnit !== undefined && m.subUnit !== null)
      e.push(`${dirName}: ادارهٔ «${m.department}» زیرواحد ندارد، پس subUnit نباید تعریف شود`);
  }
  if (m.sharedWith !== undefined) {
    if (!Array.isArray(m.sharedWith) || m.sharedWith.length === 0) e.push(`${dirName}: sharedWith باید فهرست ناتهی از کلید ادارات باشد`);
    else if (new Set(m.sharedWith).size !== m.sharedWith.length) e.push(`${dirName}: sharedWith مقدار تکراری دارد`);
    else for (const id of m.sharedWith) {
      if (id === m.department) e.push(`${dirName}: sharedWith شامل ادارهٔ خانهٔ خود ماژول است («${id}»)`);
      else if (!departments.some(d => d.id === id)) e.push(`${dirName}: sharedWith به ادارهٔ ناشناخته «${id}» اشاره می‌کند`);
    }
  }
  for (const k of req) if (!(k in m)) e.push(`${dirName}: کلید الزامی «${k}» وجود ندارد`);
  for (const k of Object.keys(m)) if (!req.includes(k) && !optional.includes(k)) e.push(`${dirName}: کلید ناشناخته «${k}»`);

  if (!/^[a-z][a-z0-9_]*$/.test(m.id ?? '')) e.push(`${dirName}: id نامعتبر`);
  if (m.id !== dirName) e.push(`${dirName}: id («${m.id}») با نام پوشه هم‌خوان نیست`);
  if (!isInt(m.order)) e.push(`${dirName}: order باید عدد صحیح باشد`);
  if (!/^\/api\/v[0-9]+\/[a-z][a-z0-9_]*$/.test(m.apiPrefix ?? '')) e.push(`${dirName}: apiPrefix نامعتبر`);
  if (!/^[0-9]+\.[0-9]+\.[0-9]+$/.test(m.version ?? '')) e.push(`${dirName}: version باید SemVer باشد`);
  if (m.featureFlag !== `module.${m.id}`) e.push(`${dirName}: featureFlag باید «module.${m.id}» باشد`);
  if (!/^hr_[a-z][a-z0-9_]*$/.test(m.dbSchema ?? '')) e.push(`${dirName}: dbSchema نامعتبر`);
  if (!/^[a-z][a-z0-9_]*$/.test(m.department ?? '')) e.push(`${dirName}: department نامعتبر (کلید اداره در contracts/departments.json)`);
  if (!isStr(m.titleFa, 2) || !isStr(m.titleEn, 2)) e.push(`${dirName}: عنوان فارسی/انگلیسی نامعتبر`);
  if (!Array.isArray(m.entities) || m.entities.length === 0) e.push(`${dirName}: entities خالی است`);

  // نقش‌ها
  if (!m.roles || typeof m.roles !== 'object' || Object.keys(m.roles).length === 0) {
    e.push(`${dirName}: roles خالی است`);
  } else {
    for (const [role, actions] of Object.entries(m.roles)) {
      if (!ROLES.includes(role)) e.push(`${dirName}: نقش ناشناخته «${role}»`);
      if (!Array.isArray(actions) || actions.length === 0) e.push(`${dirName}: نقش «${role}» بدون اقدام است`);
      for (const a of actions ?? []) if (!ACTIONS.includes(a)) e.push(`${dirName}: اقدام ناشناخته «${a}» در نقش «${role}»`);
    }
  }
  for (const [role, scope] of Object.entries(m.scopes ?? {})) {
    if (!ROLES.includes(role)) e.push(`${dirName}: دامنه برای نقش ناشناخته «${role}»`);
    if (!SCOPES.includes(scope)) e.push(`${dirName}: دامنهٔ نامعتبر «${scope}»`);
    if (m.roles && !(role in m.roles)) e.push(`${dirName}: دامنه برای نقشی تعریف شده که دسترسی ندارد («${role}»)`);
  }

  // صفحه‌ها
  if (!Array.isArray(m.screens) || m.screens.length === 0) e.push(`${dirName}: screens خالی است`);
  const seen = new Set();
  for (const [i, s] of (m.screens ?? []).entries()) {
    const tag = `${dirName}.screens[${i}]`;
    if (!/^[a-z][A-Za-z0-9]*$/.test(s.key ?? '')) e.push(`${tag}: key نامعتبر`);
    if (seen.has(s.key)) e.push(`${tag}: key تکراری «${s.key}»`);
    seen.add(s.key);
    if (!isInt(s.order)) e.push(`${tag}: order نامعتبر`);
    if (!isStr(s.icon)) e.push(`${tag}: icon نامعتبر`);
    if (!isStr(s.titleFa, 2)) e.push(`${tag}: titleFa نامعتبر`);
    if (!/^[A-Z][A-Za-z0-9]*Screen$/.test(s.component ?? '')) e.push(`${tag}: component باید به Screen ختم شود`);
    if (!isStr(s.purposeFa, 10)) e.push(`${tag}: purposeFa کوتاه است (مستندات فارسی از آن ساخته می‌شود)`);
    if (!Array.isArray(s.actionsFa) || s.actionsFa.length === 0) e.push(`${tag}: actionsFa خالی است`);
    if (!Array.isArray(s.roles) || s.roles.length === 0) e.push(`${tag}: roles خالی است`);
    for (const r of s.roles ?? []) {
      if (!ROLES.includes(r)) e.push(`${tag}: نقش ناشناخته «${r}»`);
      else if (m.roles && !(r in m.roles)) e.push(`${tag}: نقش «${r}» در دسترسی‌های ماژول نیست`);
    }
  }
  return e;
}

/** بارگذاری همهٔ مانیفست‌ها + اعتبارسنجی متقابل. departments از loadDepartments می‌آید. */
export function loadManifests(departments = []) {
  const errors = [];
  if (!existsSync(MODULES_DIR)) return { modules: [], errors: [`پوشهٔ ماژول‌ها یافت نشد: ${MODULES_DIR}`] };

  const dirs = readdirSync(MODULES_DIR, { withFileTypes: true }).filter(d => d.isDirectory()).map(d => d.name);
  const modules = [];
  for (const dir of dirs.sort()) {
    const file = join(MODULES_DIR, dir, 'module.json');
    if (!existsSync(file)) { errors.push(`${dir}: module.json یافت نشد`); continue; }
    let m;
    try {
      m = JSON.parse(readFileSync(file, 'utf8'));
    } catch (err) {
      errors.push(`${dir}: JSON نامعتبر — ${err.message}`);
      continue;
    }
    errors.push(...validateManifest(m, dir, departments));
    modules.push(m);
  }
  modules.sort((a, b) => a.order - b.order);

  // یکتا بودن شناسه‌ها، ترتیب، پیشوند API و schema
  for (const field of ['id', 'order', 'apiPrefix', 'featureFlag', 'dbSchema']) {
    const seen = new Map();
    for (const m of modules) {
      const v = String(m[field]);
      if (seen.has(v)) errors.push(`مقدار تکراری ${field} = «${v}» در ماژول‌های ${seen.get(v)} و ${m.id}`);
      seen.set(v, m.id);
    }
  }

  // ترتیب پیوسته از ۱
  modules.forEach((m, i) => { if (m.order !== i + 1) errors.push(`ترتیب ماژول‌ها پیوسته نیست: ${m.id} order=${m.order}`); });

  // هر ماژول باید به یک ادارهٔ موجود وصل باشد و ماژول‌های هر اداره پیوسته بیایند
  // (منو به ترتیب همین فهرست رندر می‌شود، پس پراکندگی یعنی منوی نامنظم).
  const order = new Map(departments.map((d, i) => [d.id, d.order]));
  let lastRank = -1;
  for (const m of modules) {
    if (!order.has(m.department)) {
      errors.push(`${m.id}: ادارهٔ ناشناخته «${m.department}» — این کلید در contracts/departments.json نیست`);
      continue;
    }
    const rank = order.get(m.department);
    if (rank < lastRank) errors.push(`${m.id}: ماژول‌های اداره «${m.department}» پیوسته نیستند (ترتیب منو به‌هم می‌ریزد)`);
    lastRank = Math.max(lastRank, rank);
  }
  for (const d of departments) {
    if (!modules.some(m => m.department === d.id)) errors.push(`ادارهٔ «${d.id}» هیچ ماژولی ندارد`);
  }

  // کلید صفحه‌ها در کل سامانه یکتا باشد
  const screens = new Map();
  for (const m of modules) for (const s of m.screens ?? []) {
    if (screens.has(s.key)) errors.push(`کلید صفحهٔ تکراری «${s.key}» در ${screens.get(s.key)} و ${m.id}`);
    screens.set(s.key, m.id);
  }
  return { modules, errors };
}

/* ---------------------------------------------------------------------
   پروفایل پارامتریک شرکت‌ها: هر شرکت ادارات و ماژول‌های فعال، پارامترها و
   مشخصات برند خود را دارد. سامانهٔ واحد برای چند شرکت شبیه‌سازی می‌شود.
   --------------------------------------------------------------------- */
const CALENDARS = ['jalali', 'gregorian'];

/** ماژول‌های فعال یک شرکت: از ادارات فعال منهای ماژول‌های غیرفعال. */
export function orgEnabledModules(org, modules) {
  const off = new Set(org.disabledModules ?? []);
  return modules.filter(m => (org.enabledDepartments ?? []).includes(m.department) && !off.has(m.id)).map(m => m.id);
}

/** پارامترهای نهایی یک شرکت: پیش‌فرض‌های سامانه + بازنویسی‌های همان شرکت. */
export function orgParams(org, defaultParams) {
  const over = org.paramOverrides ?? {};
  return defaultParams.map(p => (p.key in over ? { ...p, value: String(over[p.key]) } : { ...p }));
}

/** بارگذاری و اعتبارسنجی پروفایل شرکت‌ها. */
export function loadOrganizations(departments = [], modules = []) {
  const empty = { organizations: [], activeOrg: null, defaultParams: [], errors: [] };
  if (!existsSync(ORGANIZATIONS_FILE)) return { ...empty, errors: [`فایل شرکت‌ها یافت نشد: ${ORGANIZATIONS_FILE}`] };
  let raw;
  try {
    raw = JSON.parse(readFileSync(ORGANIZATIONS_FILE, 'utf8'));
  } catch (err) {
    return { ...empty, errors: [`organizations.json نامعتبر — ${err.message}`] };
  }
  const errors = [];
  const allowed = ['id', 'order', 'titleFa', 'titleEn', 'holdingFa', 'holdingEn', 'shortCode', 'employeeCodePrefix',
    'calendar', 'currencyFa', 'timezone', 'workWeekFa', 'fiscalYearStartFa', 'logoEmoji', 'logoFile', 'brandColor', 'headcount',
    'enabledDepartments', 'disabledModules', 'paramOverrides', 'notesFa'];
  const required = ['id', 'order', 'titleFa', 'titleEn', 'holdingFa', 'holdingEn', 'shortCode', 'employeeCodePrefix',
    'calendar', 'currencyFa', 'timezone', 'workWeekFa', 'fiscalYearStartFa', 'logoEmoji', 'headcount',
    'enabledDepartments', 'notesFa'];

  const defaultParams = raw.defaultParams ?? [];
  if (!isList(defaultParams)) errors.push('defaultParams خالی است');
  const seenParam = new Set();
  for (const [i, p] of defaultParams.entries()) {
    const tag = `defaultParams[${i}]`;
    if (!/^[a-z][A-Za-z0-9.]*$/.test(p.key ?? '')) errors.push(`${tag}: کلید نامعتبر`);
    if (seenParam.has(p.key)) errors.push(`${tag}: کلید تکراری «${p.key}»`);
    seenParam.add(p.key);
    for (const k of Object.keys(p)) if (!['key', 'titleFa', 'value', 'unitFa', 'groupFa'].includes(k)) errors.push(`${tag}: کلید ناشناخته «${k}»`);
    if (!isStr(p.titleFa, 2)) errors.push(`${tag}: titleFa نامعتبر`);
    if (typeof p.value !== 'string' || p.value === '') errors.push(`${tag}: value باید رشتهٔ ناتهی باشد`);
    if (!isStr(p.unitFa)) errors.push(`${tag}: unitFa نامعتبر`);
    if (!isStr(p.groupFa, 2)) errors.push(`${tag}: groupFa نامعتبر`);
  }

  const list = raw.organizations;
  if (!isList(list)) return { ...empty, defaultParams, errors: [...errors, 'فهرست شرکت‌ها خالی است'] };
  const seen = new Set();
  for (const [i, o] of list.entries()) {
    const tag = `organizations[${i}]`;
    for (const k of required) if (!(k in o)) errors.push(`${tag}: کلید الزامی «${k}» وجود ندارد`);
    for (const k of Object.keys(o)) if (!allowed.includes(k)) errors.push(`${tag}: کلید ناشناخته «${k}»`);
    if (!/^[a-z][a-z0-9_]*$/.test(o.id ?? '')) errors.push(`${tag}: id نامعتبر`);
    if (seen.has(o.id)) errors.push(`${tag}: id تکراری «${o.id}»`);
    seen.add(o.id);
    if (!isInt(o.order)) errors.push(`${tag}: order باید عدد صحیح باشد`);
    for (const k of ['titleFa', 'titleEn', 'holdingFa', 'holdingEn', 'currencyFa', 'timezone', 'workWeekFa', 'fiscalYearStartFa', 'notesFa'])
      if (!isStr(o[k], 2)) errors.push(`${tag}: ${k} نامعتبر`);
    if (!/^[A-Z][A-Z0-9]{1,7}$/.test(o.shortCode ?? '')) errors.push(`${tag}: shortCode باید ۲ تا ۸ نویسهٔ لاتین بزرگ باشد`);
    if (!/^[A-Z][A-Z0-9]{1,7}$/.test(o.employeeCodePrefix ?? '')) errors.push(`${tag}: employeeCodePrefix نامعتبر`);
    if (o.logoFile != null && o.logoFile !== '' && !/^[a-zA-Z0-9_./-]+\.(svg|png|webp|jpg|jpeg)$/.test(o.logoFile))
      errors.push(`${tag}: logoFile باید مسیر یک فایل svg/png/webp/jpg باشد`);
    if (o.brandColor != null && o.brandColor !== '' && !/^#[0-9A-Fa-f]{6}$/.test(o.brandColor))
      errors.push(`${tag}: brandColor باید یک رنگ شش‌رقمی مانند #0E7490 باشد`);
    if (!CALENDARS.includes(o.calendar)) errors.push(`${tag}: تقویم «${o.calendar}» پشتیبانی نمی‌شود`);
    if (!isInt(o.headcount) || o.headcount <= 0) errors.push(`${tag}: headcount نامعتبر`);
    if (!isList(o.enabledDepartments)) errors.push(`${tag}: enabledDepartments خالی است`);
    else for (const id of o.enabledDepartments)
      if (!departments.some(d => d.id === id)) errors.push(`${tag}: ادارهٔ ناشناخته «${id}» در enabledDepartments`);
    for (const id of o.disabledModules ?? [])
      if (!modules.some(m => m.id === id)) errors.push(`${tag}: ماژول ناشناخته «${id}» در disabledModules`);
    for (const id of o.disabledModules ?? [])
      if (modules.some(m => m.id === id && !o.enabledDepartments.includes(m.department)))
        errors.push(`${tag}: ماژول «${id}» ادارهٔ غیرفعال را غیرفعال اعلام می‌کند (اضافی است)`);
    for (const [k, v] of Object.entries(o.paramOverrides ?? {})) {
      if (!seenParam.has(k)) errors.push(`${tag}: بازنویسی پارامتر ناشناخته «${k}»`);
      if (typeof v !== 'string' || v === '') errors.push(`${tag}: مقدار پارامتر «${k}» باید رشتهٔ ناتهی باشد`);
    }
    if (!orgEnabledModules(o, modules).length) errors.push(`${tag}: با این تنظیمات هیچ ماژولی فعال نمی‌شود`);
  }
  const organizations = [...list].sort((a, b) => a.order - b.order);
  organizations.forEach((o, i) => { if (o.order !== i + 1) errors.push(`ترتیب شرکت‌ها پیوسته نیست: ${o.id} order=${o.order}`); });
  if (!organizations.some(o => o.id === raw.default)) errors.push(`شرکت پیش‌فرض «${raw.default}» در فهرست شرکت‌ها نیست`);
  const activeOrg = organizations.find(o => o.id === raw.default) ?? organizations[0] ?? null;
  return { organizations, activeOrg, defaultParams, errors };
}

/** بارگذاری یکجای همهٔ ساختار: ادارات، شرکت‌ها و ماژول‌ها. */
export function loadStructure() {
  const dep = loadDepartments();
  const man = loadManifests(dep.departments);
  const org = loadOrganizations(dep.departments, man.modules);
  return { ...dep, ...man, ...org, errors: [...dep.errors, ...man.errors, ...org.errors] };
}

const goStr = s => JSON.stringify(s);
/** ارقام فارسی برای متن‌های مستندات. */
const fa = n => String(n).replace(/\d/g, d => '۰۱۲۳۴۵۶۷۸۹'[d]);
const tsStr = s => JSON.stringify(s);
const sqlStr = s => `'${String(s).replace(/'/g, "''")}'`;

/** رجیستری Go — تنها منبع رجیستری ماژول‌ها در سرور. */
export function emitRegistryGo(modules) {
  const body = modules.map(m => {
    const roles = Object.entries(m.roles).map(([r, a]) => `${goStr(r)}: {${a.map(goStr).join(', ')}}`).join(', ');
    const scopes = Object.entries(m.scopes).map(([r, s]) => `${goStr(r)}: ${goStr(s)}`).join(', ');
    const screens = m.screens.map(s =>
      `\t\t\t{Key: ${goStr(s.key)}, Order: ${s.order}, Icon: ${goStr(s.icon)}, TitleFa: ${goStr(s.titleFa)}, Component: ${goStr(s.component)}, Roles: []string{${s.roles.map(goStr).join(', ')}}},`
    ).join('\n');
    return `\t{
\t\tID:          ${goStr(m.id)},
\t\tOrder:       ${m.order},
\t\tDepartment:  ${goStr(m.department)},
\t\tSubUnit:     ${goStr(m.subUnit ?? '')},
\t\tSharedWith:  []string{${(m.sharedWith ?? []).map(goStr).join(', ')}},
\t\tTitleFa:     ${goStr(m.titleFa)},
\t\tTitleEn:     ${goStr(m.titleEn)},
\t\tAPIPrefix:   ${goStr(m.apiPrefix)},
\t\tVersion:     ${goStr(m.version)},
\t\tFeatureFlag: ${goStr(m.featureFlag)},
\t\tDBSchema:    ${goStr(m.dbSchema)},
\t\tOwner:       ${goStr(m.owner)},
\t\tEntities:    []string{${m.entities.map(goStr).join(', ')}},
\t\tRoles:       map[string][]string{${roles}},
\t\tScopes:      map[string]string{${scopes}},
\t\tScreens: []Screen{
${screens}
\t\t},
\t},`;
  }).join('\n');

  return `// Code generated by tools/gen/gen.mjs from server/internal/modules/*/module.json — DO NOT EDIT.
// برای بازتولید: make gen
package modules

// All رجیستری کامل ماژول‌های سامانه است. تنها ماژول‌های فعال (feature flag) در زمان اجرا ثبت می‌شوند.
var All = []Module{
${body}
}
`;
}

/** درخت ناوبری React Native — فقط داده، بدون منطق. */
export function emitNavTs(structure) {
  const { modules, departments, directorates, sharedGroupTitleFa } = structure;
  const directorateBody = directorates.map(d =>
    `  { id: ${tsStr(d.id)}, order: ${d.order}, titleFa: ${tsStr(d.titleFa)}, titleEn: ${tsStr(d.titleEn)}, dutiesFa: [${d.dutiesFa.map(tsStr).join(', ')}] },`
  ).join('\n');
  const subUnitsTs = d => `[${(d.subUnits ?? []).map(s => `{ id: ${tsStr(s.id)}, order: ${s.order}, titleFa: ${tsStr(s.titleFa)}, dutiesFa: [${s.dutiesFa.map(tsStr).join(', ')}] }`).join(', ')}]`;
  const departmentBody = departments.map(d =>
    `  { id: ${tsStr(d.id)}, order: ${d.order}, directorate: ${tsStr(d.directorate)}, titleFa: ${tsStr(d.titleFa)}, titleEn: ${tsStr(d.titleEn)}, dutiesFa: [${d.dutiesFa.map(tsStr).join(', ')}], subUnits: ${subUnitsTs(d)} },`
  ).join('\n');
  const subTitleFa = (deptId, subId) => (departments.find(d => d.id === deptId)?.subUnits ?? []).find(s => s.id === subId)?.titleFa ?? '';
  const body = modules.map(m => {
    const screens = m.screens.map(s =>
      `      { key: ${tsStr(s.key)}, order: ${s.order}, icon: ${tsStr(s.icon)}, titleFa: ${tsStr(s.titleFa)}, component: ${tsStr(s.component)}, roles: [${s.roles.map(tsStr).join(', ')}] },`
    ).join('\n');
    return `  {
    id: ${tsStr(m.id)},
    order: ${m.order},
    department: ${tsStr(m.department)},
    departmentTitleFa: ${tsStr(departments.find(d => d.id === m.department)?.titleFa ?? '')},
    subUnit: ${tsStr(m.subUnit ?? '')},
    subUnitTitleFa: ${tsStr(subTitleFa(m.department, m.subUnit))},
    /** اداره‌های دیگری که این ماژول (مثل طبقه‌بندی مشاغل) در منوی آن‌ها هم دیده می‌شود. */
    sharedDepartments: [${(m.sharedWith ?? []).map(tsStr).join(', ')}],
    titleFa: ${tsStr(m.titleFa)},
    titleEn: ${tsStr(m.titleEn)},
    apiPrefix: ${tsStr(m.apiPrefix)},
    featureFlag: ${tsStr(m.featureFlag)},
    screens: [
${screens}
    ],
  },`;
  }).join('\n');

  return `// Code generated by tools/gen/gen.mjs from server/internal/modules/*/module.json — DO NOT EDIT.
// برای بازتولید: make gen
export type Role = 'admin' | 'hr_manager' | 'hr_spec' | 'line_manager' | 'employee';

export interface ScreenSpec {
  key: string;
  order: number;
  icon: string;
  titleFa: string;
  component: string;
  roles: Role[];
}

export interface ModuleSpec {
  id: string;
  order: number;
  /** کلید ادارهٔ منابع انسانی؛ گروه‌بندی منو از همین فیلد می‌آید. */
  department: string;
  departmentTitleFa: string;
  /** کلید زیرواحد درون اداره؛ خالی یعنی اداره زیرواحد ندارد. */
  subUnit: string;
  subUnitTitleFa: string;
  /** اداره‌های دیگری که همین ماژول در منوی آن‌ها هم نمایش داده می‌شود. */
  sharedDepartments: string[];
  titleFa: string;
  titleEn: string;
  apiPrefix: string;
  featureFlag: string;
  screens: ScreenSpec[];
}

/** زیرواحد درون یک اداره — سطح سوم گروه‌بندی منو. */
export interface SubUnitSpec {
  id: string;
  order: number;
  titleFa: string;
  dutiesFa: string[];
}

/** ادارهٔ منابع انسانی — گروه سطح دوم منو. */
export interface DepartmentSpec {
  id: string;
  order: number;
  /** کلید حوزهٔ سازمانی (directorate). */
  directorate: string;
  titleFa: string;
  titleEn: string;
  dutiesFa: string[];
  subUnits: SubUnitSpec[];
}

/** حوزهٔ سازمانی — گروه سطح اول منو (مثلاً مدیریت سرمایه‌های انسانی). */
export interface DirectorateSpec {
  id: string;
  order: number;
  titleFa: string;
  titleEn: string;
  dutiesFa: string[];
}

/** عنوان گروه ماژول‌های مشترک میان اداره‌ها. */
export const SHARED_GROUP_TITLE_FA = ${tsStr(sharedGroupTitleFa)};

export const DIRECTORATES: DirectorateSpec[] = [
${directorateBody}
];

export const DEPARTMENTS: DepartmentSpec[] = [
${departmentBody}
];

export const MODULES: ModuleSpec[] = [
${body}
];

/** ماژول‌های یک اداره به ترتیب منو. */
export function modulesOfDepartment(departmentId: string): ModuleSpec[] {
  return MODULES.filter(m => m.department === departmentId);
}
`;
}

/** حوزه‌ها، ادارات و زیرواحدها در Go — ترتیب منو را سرور تعیین می‌کند. */
export function emitDepartmentsGo(structure) {
  const { departments, directorates } = structure;
  const subs = d => (d.subUnits ?? []).map(s => `\t\t\t{ID: ${goStr(s.id)}, Order: ${s.order}, TitleFa: ${goStr(s.titleFa)}, DutiesFa: []string{${s.dutiesFa.map(goStr).join(', ')}}},`).join('\n');
  const body = departments.map(d => `\t{
\t\tID:          ${goStr(d.id)},
\t\tOrder:       ${d.order},
\t\tDirectorate: ${goStr(d.directorate)},
\t\tTitleFa:     ${goStr(d.titleFa)},
\t\tTitleEn:     ${goStr(d.titleEn)},
\t\tDutiesFa:    []string{${d.dutiesFa.map(goStr).join(', ')}},
${(d.subUnits ?? []).length ? `\t\tSubUnits: []SubUnit{\n${subs(d)}\n\t\t},` : '\t\tSubUnits:    []SubUnit{},'}
\t},`).join('\n');
  const dirBody = directorates.map(d => `\t{
\t\tID:       ${goStr(d.id)},
\t\tOrder:    ${d.order},
\t\tTitleFa:  ${goStr(d.titleFa)},
\t\tTitleEn:  ${goStr(d.titleEn)},
\t\tDutiesFa: []string{${d.dutiesFa.map(goStr).join(', ')}},
\t},`).join('\n');

  return `// Code generated by tools/gen/gen.mjs from contracts/departments.json — DO NOT EDIT.
// برای بازتولید: make gen
package modules

// Directorates حوزه‌های سازمانی به ترتیب منو (سطح اول).
var Directorates = []Directorate{
${dirBody}
}

// Departments ادارات منابع انسانی به ترتیب منو (سطح دوم) همراه با زیرواحدها (سطح سوم).
var Departments = []Department{
${body}
}
`;
}

/**
 * منوی نمونهٔ تحت وب (طرح اولیهٔ رابط) — همین درخت منو در مرورگر رندر می‌شود،
 * پس منوی نسخهٔ نمایشی و نسخهٔ نهایی نمی‌توانند از هم جدا بیفتند.
 */
export function emitPrototypeMenuJs(structure) {
  const { modules, departments, directorates, sharedGroupTitleFa } = structure;
  const screenJs = m => m.screens.map(s =>
    `        { page: ${JSON.stringify(s.key)}, label: ${JSON.stringify(s.titleFa)}, icon: ${JSON.stringify(s.icon)}, roles: [${s.roles.map(r => `'${r}'`).join(', ')}] },`
  ).join('\n');
  const moduleJs = (m, shared) => `      {
        id: ${JSON.stringify(m.id)},
        titleFa: ${JSON.stringify(m.titleFa)},
        apiPrefix: ${JSON.stringify(m.apiPrefix)},
        subUnitFa: ${JSON.stringify(m.subUnit ? ((departments.find(d => d.id === m.department)?.subUnits ?? []).find(s => s.id === m.subUnit)?.titleFa ?? '') : '')},
        shared: ${shared ? 'true' : 'false'},
        screens: [
${screenJs(m)}
        ],
      },`;
  const groupsOf = d => {
    const home = modules.filter(m => m.department === d.id);
    const out = [];
    for (const s of d.subUnits ?? []) {
      const list = home.filter(m => m.subUnit === s.id);
      if (list.length) out.push({ id: s.id, titleFa: s.titleFa, dutiesFa: s.dutiesFa, shared: false, list });
    }
    const loose = home.filter(m => !m.subUnit);
    if (loose.length) out.push({ id: '', titleFa: '', dutiesFa: [], shared: false, list: loose });
    const foreign = modules.filter(m => (m.sharedWith ?? []).includes(d.id));
    if (foreign.length) out.push({ id: 'shared', titleFa: sharedGroupTitleFa, dutiesFa: [], shared: true, list: foreign });
    return out;
  };
  const groupJs = (d, g) => `    {
      id: ${JSON.stringify(g.id)},
      titleFa: ${JSON.stringify(g.titleFa)},
      dutiesFa: ${JSON.stringify(g.dutiesFa)},
      shared: ${g.shared ? 'true' : 'false'},
      modules: [
${g.list.map(m => moduleJs(m, g.shared || m.department !== d.id)).join('\n')}
      ],
    },`;
  const departmentJs = d => `    {
      id: ${JSON.stringify(d.id)},
      titleFa: ${JSON.stringify(d.titleFa)},
      dutiesFa: ${JSON.stringify(d.dutiesFa)},
      groups: [
${groupsOf(d).map(g => groupJs(d, g)).join('\n')}
      ],
    },`;
  const body = directorates.map(dir => `  {
    id: ${JSON.stringify(dir.id)},
    titleFa: ${JSON.stringify(dir.titleFa)},
    dutiesFa: ${JSON.stringify(dir.dutiesFa)},
    departments: [
${departments.filter(d => d.directorate === dir.id).map(departmentJs).join('\n')}
    ],
  },`).join('\n');

  return `/* Code generated by tools/gen/gen.mjs from contracts/departments.json و مانیفست ماژول‌ها — DO NOT EDIT.
   منوی نمونهٔ تحت وب: حوزه > ادارهٔ منابع انسانی > زیرواحد > ماژول > صفحه.
   برای بازتولید: npm run gen */
window.HRMS_SHARED_GROUP_TITLE_FA = ${JSON.stringify(sharedGroupTitleFa)};
window.HRMS_MENU = [
${body}
];
`;
}

/**
 * پروفایل پارامتریک شرکت‌ها برای نمونهٔ تحت وب — سوئیچ شرکت در مرورگر از همین داده
 * انجام می‌شود، پس منوی نمایشی هم مثل سرور پارامتریک است.
 */
export function emitPrototypeOrgsJs(structure) {
  const { organizations, activeOrg, defaultParams, modules } = structure;
  const rows = organizations.map(o => `  {
    id: ${JSON.stringify(o.id)},
    order: ${o.order},
    titleFa: ${JSON.stringify(o.titleFa)},
    titleEn: ${JSON.stringify(o.titleEn)},
    holdingFa: ${JSON.stringify(o.holdingFa)},
    shortCode: ${JSON.stringify(o.shortCode)},
    employeeCodePrefix: ${JSON.stringify(o.employeeCodePrefix)},
    calendar: ${JSON.stringify(o.calendar)},
    currencyFa: ${JSON.stringify(o.currencyFa)},
    timezone: ${JSON.stringify(o.timezone)},
    workWeekFa: ${JSON.stringify(o.workWeekFa)},
    fiscalYearStartFa: ${JSON.stringify(o.fiscalYearStartFa)},
    logoEmoji: ${JSON.stringify(o.logoEmoji)},
    logoFile: ${JSON.stringify(o.logoFile ?? '')},
    brandColor: ${JSON.stringify(o.brandColor ?? '')},
    headcount: ${o.headcount},
    enabledDepartments: ${JSON.stringify(o.enabledDepartments)},
    disabledModules: ${JSON.stringify(o.disabledModules ?? [])},
    enabledModules: ${JSON.stringify(orgEnabledModules(o, modules))},
    notesFa: ${JSON.stringify(o.notesFa)},
    params: [
${orgParams(o, defaultParams).map(p => `      { key: ${JSON.stringify(p.key)}, titleFa: ${JSON.stringify(p.titleFa)}, value: ${JSON.stringify(p.value)}, unitFa: ${JSON.stringify(p.unitFa)}, groupFa: ${JSON.stringify(p.groupFa)} },`).join('\n')}
    ],
  },`).join('\n');

  return `/* Code generated by tools/gen/gen.mjs from contracts/organizations.json — DO NOT EDIT.
   استقرار پارامتریک شرکت‌ها: هر شرکت ادارات، ماژول‌ها و پارامترهای خود را دارد.
   برای بازتولید: npm run gen */
window.HRMS_ORGS = {
  active: ${JSON.stringify(activeOrg?.id ?? null)},
  organizations: [
${rows}
  ],
};
`;
}

/** پروفایل شرکت‌ها برای اپ React Native. */
export function emitOrganizationsTs(structure) {
  const { organizations, activeOrg, defaultParams, modules } = structure;
  const rows = organizations.map(o => `  {
    id: ${tsStr(o.id)},
    order: ${o.order},
    titleFa: ${tsStr(o.titleFa)},
    titleEn: ${tsStr(o.titleEn)},
    holdingFa: ${tsStr(o.holdingFa)},
    shortCode: ${tsStr(o.shortCode)},
    employeeCodePrefix: ${tsStr(o.employeeCodePrefix)},
    calendar: ${tsStr(o.calendar)},
    currencyFa: ${tsStr(o.currencyFa)},
    timezone: ${tsStr(o.timezone)},
    workWeekFa: ${tsStr(o.workWeekFa)},
    fiscalYearStartFa: ${tsStr(o.fiscalYearStartFa)},
    logoEmoji: ${tsStr(o.logoEmoji)},
    logoFile: ${tsStr(o.logoFile ?? '')},
    brandColor: ${tsStr(o.brandColor ?? '')},
    headcount: ${o.headcount},
    enabledDepartments: [${o.enabledDepartments.map(tsStr).join(', ')}],
    disabledModules: [${(o.disabledModules ?? []).map(tsStr).join(', ')}],
    enabledModules: [${orgEnabledModules(o, modules).map(tsStr).join(', ')}],
    notesFa: ${tsStr(o.notesFa)},
    params: [
${orgParams(o, defaultParams).map(p => `      { key: ${tsStr(p.key)}, titleFa: ${tsStr(p.titleFa)}, value: ${tsStr(p.value)}, unitFa: ${tsStr(p.unitFa)}, groupFa: ${tsStr(p.groupFa)} },`).join('\n')}
    ],
  },`).join('\n');

  return `// Code generated by tools/gen/gen.mjs from contracts/organizations.json — DO NOT EDIT.
// برای بازتولید: make gen
// استقرار پارامتریک: یک سامانه برای چند شرکت؛ منو و پارامترها از همین پروفایل می‌آید.

export interface OrgParam {
  key: string;
  titleFa: string;
  value: string;
  unitFa: string;
  groupFa: string;
}

export interface OrganizationProfile {
  id: string;
  order: number;
  titleFa: string;
  titleEn: string;
  holdingFa: string;
  shortCode: string;
  employeeCodePrefix: string;
  calendar: 'jalali' | 'gregorian';
  currencyFa: string;
  timezone: string;
  workWeekFa: string;
  fiscalYearStartFa: string;
  logoEmoji: string;
  /** مسیر نسبی فایل نشان شرکت زیر پوشهٔ assets (خالی = استفاده از نشانهٔ ایموجی). */
  logoFile: string;
  /** رنگ اصلی برند شرکت برای سرصفحه و اسناد. */
  brandColor: string;
  headcount: number;
  /** اداره‌های فعال در این استقرار. */
  enabledDepartments: string[];
  /** ماژول‌های غیرفعال در این استقرار. */
  disabledModules: string[];
  /** ماژول‌های نهایی فعال = اداره‌های فعال منهای ماژول‌های غیرفعال. */
  enabledModules: string[];
  notesFa: string;
  params: OrgParam[];
}

/** شناسهٔ استقراری که سامانه به‌صورت پیش‌فرض برای آن بالا می‌آید. */
export const DEFAULT_ORGANIZATION_ID = ${tsStr(activeOrg?.id ?? '')};

export const ORGANIZATIONS: OrganizationProfile[] = [
${rows}
];

/** یافتن پروفایل شرکت با شناسه؛ نبودن آن یعنی ورودی نامعتبر. */
export function organizationById(id: string): OrganizationProfile | undefined {
  return ORGANIZATIONS.find(o => o.id === id);
}

export const ACTIVE_ORGANIZATION: OrganizationProfile =
  organizationById(DEFAULT_ORGANIZATION_ID) ?? ORGANIZATIONS[0];

/** آیا این ماژول در استقرار شرکت داده‌شده فعال است. */
export function isModuleEnabled(orgId: string, moduleId: string): boolean {
  const org = organizationById(orgId);
  return org ? org.enabledModules.includes(moduleId) : false;
}
`;
}

/** پروفایل شرکت‌ها در Go — پرچم فعال/غیرفعال ماژول‌ها از همین‌جا می‌آید. */
export function emitOrgsGo(structure) {
  const { organizations, activeOrg, defaultParams, modules } = structure;
  const paramRows = list => list.map(p => `\t\t\t{Key: ${goStr(p.key)}, TitleFa: ${goStr(p.titleFa)}, Value: ${goStr(p.value)}, UnitFa: ${goStr(p.unitFa)}, GroupFa: ${goStr(p.groupFa)}},`).join('\n');
  const body = organizations.map(o => `\t{
\t\tID:                 ${goStr(o.id)},
\t\tOrder:              ${o.order},
\t\tTitleFa:            ${goStr(o.titleFa)},
\t\tTitleEn:            ${goStr(o.titleEn)},
\t\tHoldingFa:          ${goStr(o.holdingFa)},
\t\tHoldingEn:          ${goStr(o.holdingEn)},
\t\tShortCode:          ${goStr(o.shortCode)},
\t\tEmployeeCodePrefix: ${goStr(o.employeeCodePrefix)},
\t\tCalendar:           ${goStr(o.calendar)},
\t\tCurrencyFa:         ${goStr(o.currencyFa)},
\t\tTimezone:           ${goStr(o.timezone)},
\t\tWorkWeekFa:         ${goStr(o.workWeekFa)},
\t\tFiscalYearStartFa:  ${goStr(o.fiscalYearStartFa)},
\t\tLogoEmoji:          ${goStr(o.logoEmoji)},
		LogoFile:           ${goStr(o.logoFile ?? '')},
		BrandColor:         ${goStr(o.brandColor ?? '')},
\t\tHeadcount:          ${o.headcount},
\t\tEnabledDepartments: []string{${o.enabledDepartments.map(goStr).join(', ')}},
\t\tDisabledModules:    []string{${(o.disabledModules ?? []).map(goStr).join(', ')}},
\t\tEnabledModules:     []string{${orgEnabledModules(o, modules).map(goStr).join(', ')}},
\t\tNotesFa:            ${goStr(o.notesFa)},
\t\tParams: []Param{
${paramRows(orgParams(o, defaultParams))}
\t\t},
\t},`).join('\n');

  return `// Code generated by tools/gen/gen.mjs from contracts/organizations.json — DO NOT EDIT.
// برای بازتولید: make gen
package orgs

// DefaultID شناسهٔ استقراری است که سرور به‌صورت پیش‌فرض برای آن بالا می‌آید.
const DefaultID = ${goStr(activeOrg?.id ?? '')}

// All پروفایل پارامتریک شرکت‌ها به ترتیب؛ فعال بودن ماژول‌ها از همین‌جا تعیین می‌شود.
var All = []Organization{
${body}
}
`;
}

/** seed دیتریبیتیو دسترسی‌های نقش‌محور (RBAC). */
export function emitRbacSql(modules) {
  const rows = [];
  for (const m of modules) {
    for (const [role, actions] of Object.entries(m.roles)) {
      const scope = m.scopes[role] ?? 'all';
      for (const a of actions) rows.push(`  (${sqlStr(role)}, ${sqlStr(m.id)}, ${sqlStr(a)}, ${sqlStr(scope)})`);
    }
  }
  return `-- Code generated by tools/gen/gen.mjs from server/internal/modules/*/module.json — DO NOT EDIT.
-- seed دسترسی‌ها: role × module × action × scope
begin;

create table if not exists rbac_permission (
  role        text not null,
  module_id   text not null,
  action      text not null,
  scope       text not null check (scope in ('all', 'own_unit', 'own')),
  primary key (role, module_id, action)
);

insert into rbac_permission (role, module_id, action, scope) values
${rows.join(',\n')}
on conflict (role, module_id, action) do update set scope = excluded.scope;

commit;
`;
}

/** قرارداد مصرفی ابزارها و کلاینت‌ها. */
export function emitModulesJson(structure) {
  const { modules, departments, directorates, organizations, activeOrg, sharedGroupTitleFa } = structure;
  return JSON.stringify({
    generatedBy: 'tools/gen/gen.mjs',
    roles: ROLES,
    actions: ACTIONS,
    sharedGroupTitleFa,
    directorates: directorates.map(d => ({ id: d.id, order: d.order, titleFa: d.titleFa, titleEn: d.titleEn, dutiesFa: d.dutiesFa })),
    departments: departments.map(d => ({
      id: d.id, order: d.order, directorate: d.directorate, titleFa: d.titleFa, titleEn: d.titleEn, dutiesFa: d.dutiesFa,
      subUnits: (d.subUnits ?? []).map(s => ({ id: s.id, order: s.order, titleFa: s.titleFa, dutiesFa: s.dutiesFa })),
    })),
    organizations: organizations.map(o => ({
      id: o.id, order: o.order, titleFa: o.titleFa, titleEn: o.titleEn, holdingFa: o.holdingFa, shortCode: o.shortCode,
      employeeCodePrefix: o.employeeCodePrefix, calendar: o.calendar, currencyFa: o.currencyFa, logoEmoji: o.logoEmoji,
      headcount: o.headcount, enabledDepartments: o.enabledDepartments, disabledModules: o.disabledModules ?? [],
      enabledModules: orgEnabledModules(o, modules), params: orgParams(o, structure.defaultParams), notesFa: o.notesFa,
    })),
    activeOrganization: activeOrg?.id ?? null,
    modules: modules.map(m => ({
      id: m.id, order: m.order, department: m.department, subUnit: m.subUnit ?? '', sharedWith: m.sharedWith ?? [],
      titleFa: m.titleFa, titleEn: m.titleEn,
      apiPrefix: m.apiPrefix, version: m.version, featureFlag: m.featureFlag, dbSchema: m.dbSchema,
      owner: m.owner, entities: m.entities, roles: m.roles, scopes: m.scopes,
      screens: m.screens.map(s => ({ key: s.key, order: s.order, titleFa: s.titleFa, component: s.component, roles: s.roles })),
    })),
  }, null, 2) + '\n';
}

/** مستندات فارسی صفحه‌به‌صفحه (دستور ۵-۵ سند طراحی) — از مانیفست تولید می‌شود. */
export function emitDocsFa(structure) {
  const { modules, departments, directorates, organizations, activeOrg, defaultParams } = structure;
  const depTitle = id => departments.find(d => d.id === id)?.titleFa ?? id;
  const subTitle = (depId, subId) => (departments.find(d => d.id === depId)?.subUnits ?? []).find(s => s.id === subId)?.titleFa ?? '';
  const lines = [
    '# مستندات صفحه‌به‌صفحه (تولیدشده از مانیفست ماژول‌ها)',
    '',
    '> این فایل به‌صورت خودکار از `server/internal/modules/*/module.json` ساخته می‌شود؛ دستی ویرایش نکنید.',
    '> برای تغییر، مانیفست ماژول را ویرایش و `npm run gen` را اجرا کنید.',
    '',
    `تعداد حوزه: **${directorates.length}** — تعداد اداره: **${departments.length}** — تعداد ماژول: **${modules.length}** — تعداد صفحه: **${modules.reduce((n, m) => n + m.screens.length, 0)}**`,
    '',
    `شرکت فعال: **${activeOrg?.titleFa ?? '—'}** از **${activeOrg?.holdingFa ?? '—'}** — تعداد استقرار پارامتریک قابل شبیه‌سازی: **${organizations.length}**`,
    '',
    '## حوزه‌ها، ادارات و زیرواحدها',
    '',
    ...directorates.flatMap(dir => [
      `### حوزهٔ ${dir.order}. ${dir.titleFa} (${dir.titleEn})`,
      '',
      ...dir.dutiesFa.map(x => `- ${x}`),
      '',
      ...departments.filter(d => d.directorate === dir.id).flatMap(d => [
        `#### ${d.order}. ${d.titleFa} (${d.titleEn})`,
        '',
        '**وظایف:**',
        '',
        ...d.dutiesFa.map(x => `- ${x}`),
        '',
        ...((d.subUnits ?? []).length ? [
          '**زیرواحدها:**',
          '',
          ...(d.subUnits ?? []).map(s => `- ${s.titleFa} — ${s.dutiesFa.join('؛ ')}`),
          '',
        ] : []),
        `**ماژول‌ها:** ${modules.filter(m => m.department === d.id).map(m => `${m.titleFa} (زیرواحد: ${subTitle(d.id, m.subUnit) || 'ندارد'})`).join('، ') || '—'}`,
        '',
      ]),
    ]),
    '## استقرار پارامتریک شرکت‌ها',
    '',
    '| # | شرکت | هلدینگ | ادارهٔ فعال | ماژول فعال | پارامترهای بازنویسی‌شده |',
    '|---:|---|---|---:|---:|---|',
    ...organizations.map(o => `| ${o.order} | ${o.titleFa}${o.id === activeOrg?.id ? ' (فعال)' : ''} | ${o.holdingFa} | ${fa(o.enabledDepartments.length)} | ${fa(orgEnabledModules(o, modules).length)} | ${Object.keys(o.paramOverrides ?? {}).join('، ') || '—'} |`),
    '',
    ...organizations.flatMap(o => [
      `### ${o.order}. ${o.titleFa} — ${o.holdingFa}`,
      '',
      `- کد شرکت: ${o.shortCode} — پیشوند کد پرسنلی: ${o.employeeCodePrefix} — تقویم: ${o.calendar === 'jalali' ? 'شمسی' : 'میلادی'}`,
      `- نشان شرکت: ${o.logoFile || 'ندارد (ایموجی ' + o.logoEmoji + ')'} — رنگ برند: ${o.brandColor || 'پیش‌فرض'}`,
      `- سال مالی: ${o.fiscalYearStartFa} — هفتهٔ کاری: ${o.workWeekFa} — تعداد کارکنان: ${o.headcount}`,
      `- ماژول‌های غیرفعال: ${(o.disabledModules ?? []).map(id => modules.find(m => m.id === id)?.titleFa ?? id).join('، ') || 'ندارد'}`,
      `- توضیح: ${o.notesFa}`,
      '',
      '| پارامتر | گروه | مقدار | واحد |',
      '|---|---|---|---|',
      ...orgParams(o, defaultParams).map(p => `| ${p.titleFa} | ${p.groupFa} | ${p.value} | ${p.unitFa} |`),
      '',
    ]),
    '## فهرست ماژول‌ها',
    '',
    '| # | اداره | زیرواحد | ماژول | پیشوند API | محدودهٔ داده | مسئول |',
    '|---:|---|---|---|---|---|---|',
    ...modules.map(m => `| ${m.order} | ${depTitle(m.department)} | ${subTitle(m.department, m.subUnit) || '—'} | ${m.titleFa} | \`${m.apiPrefix}\` | ${Object.entries(m.scopes).map(([r, s]) => `${ROLE_FA[r]}=${s}`).join('، ') || 'سازمانی'} | ${m.owner} |`),
    '',
  ];
  for (const m of modules) {
    lines.push(`## ${m.order}. ${m.titleFa} (${m.titleEn})`, '');
    lines.push(`- اداره: **${depTitle(m.department)}**`);
    lines.push(`- زیرواحد: **${subTitle(m.department, m.subUnit) || '—'}**`);
    if ((m.sharedWith ?? []).length) lines.push(`- ماژول مشترک با: ${m.sharedWith.map(depTitle).join('، ')}`);
    lines.push(`- پیشوند API: \`${m.apiPrefix}\``, `- نسخه: \`${m.version}\``, `- پرچم ویژگی: \`${m.featureFlag}\``, `- schema دیتابیس: \`${m.dbSchema}\``);
    lines.push(`- موجودیت‌ها: ${m.entities.map(x => `\`${x}\``).join('، ')}`);
    const roleLine = Object.entries(m.roles).map(([r, a]) => `${ROLE_FA[r]} (${a.join('/')})`).join('، ');
    lines.push(`- نقش‌های دارای دسترسی ماژول: ${roleLine}`, '');
    lines.push('### صفحه‌ها', '');
    for (const s of m.screens) {
      lines.push(`#### ${s.order}. ${s.titleFa}`, '');
      lines.push(`- **هدف صفحه:** ${s.purposeFa}`);
      lines.push(`- **عناصر اصلی:** قالب استاندارد (Header، Sidebar راست‌چین، ناحیهٔ محتوا، Action Bar، Breadcrumb) با کامپوننت \`${s.component}\`.`);
      lines.push(`- **اقدامات قابل انجام:** ${s.actionsFa.join('، ')}.`);
      lines.push(`- **نقش‌های مجاز:** ${s.roles.map(r => ROLE_FA[r]).join('، ')}.`);
      lines.push(`- **کلید ناوبری:** \`${m.id}.${s.key}\` — آیکون: \`${s.icon}\``, '');
    }
  }
  return lines.join('\n');
}

const ROLE_VARIANTS = { admin: 'Admin', hr_manager: 'HrManager', hr_spec: 'HrSpec', line_manager: 'LineManager', employee: 'Employee' };
const ACTION_VARIANTS = { read: 'Read', write: 'Write', delete: 'Delete', approve: 'Approve', report: 'Report', admin: 'Admin' };
const SCOPE_VARIANTS = { all: 'All', own_unit: 'OwnUnit', own: 'Own' };

/** جدول سیاست هستهٔ Rust — از همان مانیفست تا RBAC در دو میزبان یک منبع داشته باشد. */
export function emitPolicyRs(modules) {
  const body = modules.map(m => {
    const allowed = [];
    for (const [role, actions] of Object.entries(m.roles)) {
      for (const a of actions) allowed.push(`        (RoleKey::${ROLE_VARIANTS[role]}, Action::${ACTION_VARIANTS[a]}),`);
    }
    const scopes = Object.entries(m.scopes).map(([role, s]) => `        (RoleKey::${ROLE_VARIANTS[role]}, Scope::${SCOPE_VARIANTS[s]}),`);
    const screens = m.screens.map(s =>
      `        ScreenPolicy {\n            key: ${JSON.stringify(s.key)},\n            title_fa: ${JSON.stringify(s.titleFa)},\n            component: ${JSON.stringify(s.component)},\n            icon: ${JSON.stringify(s.icon)},\n            roles: &[${s.roles.map(r => `RoleKey::${ROLE_VARIANTS[r]}`).join(', ')}],\n        },`
    ).join('\n');
    return `    ModulePolicy {
        module_id: ${JSON.stringify(m.id)},
        order: ${m.order},
        title_fa: ${JSON.stringify(m.titleFa)},
        api_prefix: ${JSON.stringify(m.apiPrefix)},
        feature_flag: ${JSON.stringify(m.featureFlag)},
        allowed: &[
${allowed.join('\n')}
        ],
        scopes: &[
${scopes.join('\n')}
        ],
        screens: &[
${screens}
        ],
    },`;
  }).join('\n');

  return `// Code generated by tools/gen/gen.mjs from server/internal/modules/*/module.json — DO NOT EDIT.
// برای بازتولید: make gen
// این جدول همان منبع مانیفست سرور Go است، پس سیاست دسترسی در دو میزبان نمی‌تواند از هم جدا بیفتد.
use crate::{Action, ModulePolicy, RoleKey, Scope, ScreenPolicy};

/// همهٔ نقش‌های شناخته‌شده به ترتیب سند طراحی.
pub const ROLE_KEYS: [RoleKey; ${ROLES.length}] = [${ROLES.map(r => `RoleKey::${ROLE_VARIANTS[r]}`).join(', ')}];

/// رجیستری کامل سیاست‌ها به ترتیب منو.
pub static POLICIES: &[ModulePolicy] = &[
${body}
];
`;
}

/**
 * انواع TypeScript کلاینت از همان OpenAPI تولید می‌شوند تا دامنهٔ سرور و UI
 * نتوانند از هم جدا بیفتند (دروازهٔ CI انحراف را می‌گیرد).
 */
export function emitApiTs(spec) {
  const tsType = (schema, indent = '  ') => {
    if (!schema) return 'unknown';
    if (schema.$ref) return schema.$ref.split('/').pop();
    if (schema.enum) return schema.enum.map(v => JSON.stringify(v)).join(' | ');
    switch (schema.type) {
      case 'string': return schema.format === 'password' ? 'string' : 'string';
      case 'integer': case 'number': return 'number';
      case 'boolean': return 'boolean';
      case 'array': return `${tsType(schema.items, indent)}[]`;
      case 'object': {
        const props = Object.entries(schema.properties ?? {});
        if (props.length === 0) return 'Record<string, unknown>';
        const lines = props.map(([k, v]) => {
          const optional = (schema.required ?? []).includes(k) ? '' : '?';
          const comment = v.description ? `${indent}  /** ${v.description} */\n` : '';
          return `${comment}${indent}  ${k}${optional}: ${tsType(v, indent + '  ')};`;
        });
        return `{\n${lines.join('\n')}\n${indent}}`;
      }
      default: return 'unknown';
    }
  };

  const blocks = Object.entries(spec.components.schemas).map(([name, schema]) => {
    if (schema.enum) return `export type ${name} = ${tsType(schema)};`;
    const props = Object.entries(schema.properties ?? {});
    const lines = props.map(([k, v]) => {
      const optional = (schema.required ?? []).includes(k) ? '' : '?';
      const comment = v.description ? `  /** ${v.description} */\n` : '';
      return `${comment}  ${k}${optional}: ${tsType(v, '  ')};`;
    });
    return `export interface ${name} {\n${lines.join('\n')}\n}`;
  });

  const httpMethods = ['get', 'post', 'put', 'patch', 'delete', 'head', 'options'];
  const paths = Object.entries(spec.paths).flatMap(([path, ops]) =>
    Object.keys(ops)
      .filter(op => httpMethods.includes(op.toLowerCase()))
      .map(op => `  { method: ${JSON.stringify(op.toUpperCase())}, path: ${JSON.stringify(path)} },`)
  );

  return `// Code generated by tools/gen/gen.mjs from contracts/openapi/hrms.json — DO NOT EDIT.
// برای بازتولید: make gen
// منبع واحد دامنهٔ سرور و کلاینت؛ هر تغییر قرارداد باید از این مسیر بگذرد.

/** مسیرهای اعلام‌شده در قرارداد؛ کلاینت فقط از این‌ها استفاده می‌کند. */
export const API_ROUTES = [
${paths.join('\n')}
] as const;

${blocks.join('\n\n')}
`;
}

/** قرارداد OpenAPI سامانه (منبع تولید انواع TypeScript). */
export function loadSpec() {
  const file = join(ROOT, 'contracts', 'openapi', 'hrms.json');
  try {
    return JSON.parse(readFileSync(file, 'utf8'));
  } catch {
    return null;
  }
}

/** همهٔ خروجی‌های تولیدشده: مسیر → محتوا. */
export function buildArtifacts(structure, spec = loadSpec()) {
  const { modules } = structure;
  const orgsJson = JSON.stringify({
    generatedBy: 'tools/gen/gen.mjs',
    activeOrganization: structure.activeOrg?.id ?? null,
    organizations: structure.organizations.map(o => ({
      ...o,
      enabledModules: orgEnabledModules(o, modules),
      params: orgParams(o, structure.defaultParams),
    })),
  }, null, 2) + '\n';
  return new Map([
    [join(ROOT, 'server', 'internal', 'platform', 'modules', 'registry.gen.go'), emitRegistryGo(modules)],
    [join(ROOT, 'server', 'internal', 'platform', 'modules', 'departments.gen.go'), emitDepartmentsGo(structure)],
    [join(ROOT, 'server', 'internal', 'platform', 'orgs', 'orgs.gen.go'), emitOrgsGo(structure)],
    [join(ROOT, 'apps', 'mobile', 'src', 'navigation', 'modules.generated.ts'), emitNavTs(structure)],
    [join(ROOT, 'apps', 'mobile', 'src', 'config', 'organizations.generated.ts'), emitOrganizationsTs(structure)],
    [join(ROOT, 'server', 'migrations', '000_seed_rbac.sql'), emitRbacSql(modules)],
    [join(ROOT, 'contracts', 'generated', 'modules.gen.json'), emitModulesJson(structure)],
    [join(ROOT, 'contracts', 'generated', 'organizations.gen.json'), orgsJson],
    [join(ROOT, 'docs', 'generated', 'pages-fa.md'), emitDocsFa(structure)],
    [join(ROOT, 'core', 'crates', 'hrms-policy', 'src', 'policy_gen.rs'), emitPolicyRs(modules)],
    [join(PROTOTYPE_JS_DIR, 'menu.generated.js'), emitPrototypeMenuJs(structure)],
    [join(PROTOTYPE_JS_DIR, 'organizations.generated.js'), emitPrototypeOrgsJs(structure)],
    ...(spec ? [[join(ROOT, 'contracts', 'generated', 'api.types.ts'), emitApiTs(spec)]] : []),
  ]);
}
