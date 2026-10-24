/**
 * منطق ناوبری نقش‌محور.
 *
 * داده‌های منو از مانیفست تولیدشده می‌آید (`modules.generated.ts`) و منبع نهایی همان
 * پاسخ `/api/v1/auth/me` و هستهٔ بومی است. این فایل فقط کمک می‌کند پیش از رسیدن پاسخ
 * سرور، منو و دسترسی‌ها قابل‌محاسبه باشند (مثلاً برای پیش‌نمایش صفحهٔ ورود).
 *
 * هیچ‌جای UI صفحه‌ای را بدون بررسی دسترسی رندر نمی‌کند؛ حتی اگر کسی مسیر را دستی
 * تغییر دهد، سرور درخواست را با ۴۰۳ رد می‌کند.
 */

import { DEPARTMENTS, DIRECTORATES, MODULES } from './modules.generated';
import type { ModuleSpec, Role, ScreenSpec } from './modules.generated';
import type { MenuEntry } from '../core/types';

/** ماژول‌هایی که نقش دست‌کم به یک صفحهٔ آن‌ها دسترسی دارد. */
export function modulesForRole(role: Role): ModuleSpec[] {
  return MODULES.filter(module => module.screens.some(screen => screen.roles.includes(role)));
}

/** صفحه‌های مجاز یک نقش در یک ماژول. */
export function screensForRole(role: Role, moduleId: string): ScreenSpec[] {
  const module = MODULES.find(m => m.id === moduleId);
  return module ? module.screens.filter(screen => screen.roles.includes(role)) : [];
}

/** آیا نقش اجازهٔ باز کردن این صفحه را دارد. */
export function canOpenScreen(role: Role, moduleId: string, screenKey: string): boolean {
  return screensForRole(role, moduleId).some(screen => screen.key === screenKey);
}

/** ساخت منوی محلی از مانیفست (پیش از پاسخ سرور). */
export function localMenu(role: Role): MenuEntry[] {
  return modulesForRole(role).flatMap(module => {
    const screens = screensForRole(role, module.id).map(screen => ({
      key: screen.key,
      titleFa: screen.titleFa,
      component: screen.component,
      icon: screen.icon,
    }));
    if (screens.length === 0) {
      return [];
    }
    const base = {
      moduleId: module.id,
      titleFa: module.titleFa,
      order: module.order,
      screens,
      department: module.department,
      departmentTitleFa: module.departmentTitleFa,
      subUnit: module.subUnit,
      subUnitTitleFa: module.subUnitTitleFa,
      directorate: directionOf(module.department),
    };
    // ماژول مشترک (طبقه‌بندی مشاغل) در منوی هر دو اداره دیده می‌شود تا کاربر برای
    // رسیدن به آن مجبور نباشد از منوی ادارهٔ دیگری رد شود.
    return [
      base,
      ...module.sharedDepartments.map(dep => ({
        ...base,
        shared: true,
        department: dep,
        departmentTitleFa: departmentTitleOf(dep),
        directorate: directionOf(dep),
      })),
    ];
  });
}

/** حوزهٔ سازمانی یک اداره. */
function directionOf(departmentId: string): string {
  return DEPARTMENTS.find(d => d.id === departmentId)?.directorate ?? '';
}

/** عنوان فارسی یک ادارهٔ منابع انسانی. */
function departmentTitleOf(departmentId: string): string {
  return DEPARTMENTS.find(d => d.id === departmentId)?.titleFa ?? '';
}

/** زیرواحد در منو: فقط یک برچسب گروه‌بندی، بدون منطق دسترسی. */
export interface MenuTreeSubUnit {
  id: string;
  titleFa: string;
  modules: MenuEntry[];
}

/** اداره در منو. */
export interface MenuTreeDepartment {
  id: string;
  titleFa: string;
  subUnits: MenuTreeSubUnit[];
}

/** حوزهٔ سازمانی در منو — سطح بالای درخت. */
export interface MenuTreeDirectorate {
  id: string;
  titleFa: string;
  departments: MenuTreeDepartment[];
}

/**
 * چیدن منوی نقش‌محور در درخت «حوزه > اداره > زیرواحد > ماژول».
 *
 * همین تابع هم برای پاسخ سرور و هم برای منوی محلی کار می‌کند؛ اگر فیلدهای گروه‌بندی
 * نباشند (نسخهٔ قدیمی‌تر سرور)، همه در یک حوزهٔ خالی می‌نشینند و منو تخت اما سالم می‌ماند.
 */
export function menuTree(menu: MenuEntry[]): MenuTreeDirectorate[] {
  const tree: MenuTreeDirectorate[] = [];
  for (const entry of menu) {
    const dirId = entry.directorate ?? '';
    let dir = tree.find(d => d.id === dirId);
    if (!dir) {
      dir = { id: dirId, titleFa: DIRECTORATES.find(d => d.id === dirId)?.titleFa ?? '', departments: [] };
      tree.push(dir);
    }
    const depId = entry.department ?? '';
    let dep = dir.departments.find(d => d.id === depId);
    if (!dep) {
      dep = { id: depId, titleFa: entry.departmentTitleFa ?? '', subUnits: [] };
      dir.departments.push(dep);
    }
    const subId = entry.subUnit ?? '';
    let sub = dep.subUnits.find(s => s.id === subId);
    if (!sub) {
      sub = { id: subId, titleFa: entry.subUnitTitleFa ?? '', modules: [] };
      dep.subUnits.push(sub);
    }
    sub.modules.push(entry);
  }
  return tree;
}

/** مسیر پیش‌فرض هر نقش: نخستین صفحهٔ نخستین ماژول مجاز. */
export function defaultRoute(role: Role): { moduleId: string; screenKey: string } | undefined {
  const modules = modulesForRole(role);
  for (const module of modules) {
    const screen = module.screens.find(s => s.roles.includes(role));
    if (screen) {
      return { moduleId: module.id, screenKey: screen.key };
    }
  }
  return undefined;
}

/** عنوان فارسی مسیر جاری برای نمایش در Breadcrumb و نوار بالا. */
export function routeTitle(moduleId: string, screenKey: string): string {
  const module = MODULES.find(m => m.id === moduleId);
  const screen = module?.screens.find(s => s.key === screenKey);
  return screen ? `${module?.titleFa ?? ''} › ${screen.titleFa}` : '';
}

/** یک گام Breadcrumb؛ مطابق دستور ۳-۱ سند طراحی: «منابع انسانی > کارکنان > پرونده». */
export interface BreadcrumbStep {
  label: string;
  moduleId?: string;
  screenKey?: string;
}

/** مسیر ناوبری برای نوار Breadcrumb. */
export function breadcrumb(moduleId: string, screenKey: string): BreadcrumbStep[] {
  const module = MODULES.find(m => m.id === moduleId);
  const screen = module?.screens.find(s => s.key === screenKey);
  const steps: BreadcrumbStep[] = [{ label: 'منابع انسانی' }];
  if (module) {
    steps.push({ label: module.titleFa, moduleId: module.id });
  }
  if (screen) {
    steps.push({ label: screen.titleFa, moduleId, screenKey });
  }
  return steps;
}

/** آیا نقش به این اقدام در این ماژول دسترسی دارد (بر پایهٔ مانیفست؛ تصمیم نهایی با سرور است). */
export function roleHasScreenRole(role: Role, moduleId: string, screenKey: string): boolean {
  return canOpenScreen(role, moduleId, screenKey);
}
