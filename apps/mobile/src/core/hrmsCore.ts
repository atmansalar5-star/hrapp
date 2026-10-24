/**
 * پل کلاینت با هستهٔ بومی (Rust).
 *
 * قاعدهٔ معماری: UI هیچ قاعدهٔ کسب‌وکاری ندارد. هر تصمیم دامنه‌ای — منوی نقش‌محور،
 * اعتبارسنجی، محاسبهٔ حقوق و مرخصی، تقویم شمسی و صف آفلاین — از هستهٔ بومی گرفته
 * می‌شود. اگر هسته بار نشده باشد (اجرای پیش‌نمایش بدون بیلد بومی)، این ماژول **محاسبه
 * نمی‌کند**؛ فقط سیاست دسترسی و منو را از مانیفست تولیدشده می‌خواند و برای بقیهٔ
 * روش‌ها خطای صریح برمی‌گرداند تا هرگز نسخهٔ دومی از قواعد در جاوااسکریپت شکل نگیرد.
 */

import type { MenuEntry, Role } from './types';
import { DEPARTMENTS, DIRECTORATES, MODULES } from '../navigation/modules.generated';

/** حوزهٔ سازمانی یک اداره (برای گروه‌بندی منو). */
function directoryOfDepartment(departmentId: string): string {
  return DEPARTMENTS.find(d => d.id === departmentId)?.directorate ?? '';
}

/** عنوان فارسی یک ادارهٔ منابع انسانی. */
function departmentTitleOf(departmentId: string): string {
  return DEPARTMENTS.find(d => d.id === departmentId)?.titleFa ?? '';
}

/** پاسخ استاندارد پل. */
export interface CoreEnvelope<T> {
  ok: boolean;
  result?: T;
  error?: string;
}

/** قرارداد TurboModule سمت بومی (تولیدشده از hrms-ffi). */
interface NativeHrmsCore {
  call: (method: string, payloadJson: string) => string;
  version: () => string;
}

let nativeCore: NativeHrmsCore | null = null;

try {
  // در بیلد بومی، ماژول NativeHrmsCore وجود دارد؛ در پیش‌نمایش، غایب است.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const loaded = require('./NativeHrmsCore').default as NativeHrmsCore;
  nativeCore = loaded ?? null;
} catch {
  nativeCore = null;
}

/** آیا هستهٔ بومی در این اجرا در دسترس است. */
export const hasNativeCore = (): boolean => nativeCore !== null;

/** فراخوانی یک روش هسته. */
export function callCore<T>(method: string, payload?: unknown): CoreEnvelope<T> {
  if (nativeCore) {
    try {
      return JSON.parse(nativeCore.call(method, JSON.stringify(payload ?? null))) as CoreEnvelope<T>;
    } catch (error) {
      return { ok: false, error: `پاسخ هسته قابل تجزیه نیست: ${String(error)}` };
    }
  }
  return fallback(method, payload);
}

/** خواندن نتیجه یا پرتاب خطا با پیام فارسی هسته. */
export function callCoreOrThrow<T>(method: string, payload?: unknown): T {
  const envelope = callCore<T>(method, payload);
  if (!envelope.ok || envelope.result === undefined) {
    throw new Error(envelope.error ?? 'خطای نامشخص در هستهٔ کلاینت');
  }
  return envelope.result;
}

/** خلاصهٔ راه‌اندازی هسته. */
export function bootstrap(): { coreVersion: string; schemaVersion: number; methods: string[]; modules: unknown[] } {
  return callCoreOrThrow('core.bootstrap');
}

/** منوی نقش‌محور؛ منبع رسمی پاسخ سرور است ولی پیش از ورود هم از هسته خوانده می‌شود. */
export function menu(role: Role): MenuEntry[] {
  return callCoreOrThrow<MenuEntry[]>('policy.menu', { role });
}

/** بررسی دسترسی: UI فقط چیزی را نشان می‌دهد که سرور هم می‌پذیرد. */
export function authorize(role: Role, moduleId: string, action: string): boolean {
  const result = callCore<{ allowed: boolean }>('policy.authorize', { role, module: moduleId, action });
  if (!result.ok || !result.result) {
    return false;
  }
  return result.result.allowed;
}

/** اعتبارسنجی ورودی کارمند پیش از ارسال به سرور (خطاها کنار فیلد نمایش داده می‌شوند). */
export function validateEmployee(input: unknown): { valid: boolean; errors: Record<string, string> } {
  return callCoreOrThrow('employee.validate', input);
}

/** محاسبهٔ فیش حقوقی (نتیجه در سرور هم بازبینی می‌شود). */
export function computePayroll(input: unknown, params: unknown): unknown {
  return callCoreOrThrow('payroll.compute', { input, params });
}

/** شمار روز و ماندهٔ مرخصی. */
export function computeLeave(input: unknown): {
  days: number;
  workingDays: number;
  sandwichDays: number;
  eligible: boolean;
  balanceAfter?: number | null;
} {
  return callCoreOrThrow('leave.compute', input);
}

/** اطلاعات تقویمی یک تاریخ شمسی. */
export function calendarInfo(date: string): {
  jalali: string;
  jalaliPersian: string;
  gregorian: string;
  weekdayFa: string;
  monthLabelFa: string;
  monthLength: number;
  leap: boolean;
} {
  return callCoreOrThrow('calendar.info', { date });
}

// ------------------------------------------------------------------ صف آفلاین

/** ثبت یک تغییر محلی در صف آفلاین. */
export function enqueueMutation(mutation: {
  opId: string;
  entity: string;
  op: 'create' | 'update' | 'delete';
  entityId: string;
  payload: unknown;
  clientTs: string;
  baseVersion?: number;
}): { enqueued: boolean; pending: number } {
  return callCoreOrThrow('store.enqueue', mutation);
}

/** عملیات‌های در انتظار ارسال. */
export function pendingMutations(): { entries: unknown[]; pending: number } {
  return callCoreOrThrow('store.pending');
}

/** علامت‌گذاری عملیات‌های پذیرفته‌شدهٔ سرور. */
export function markMutationsSent(opIds: string[]): { pending: number } {
  return callCoreOrThrow('store.markSent', { opIds });
}

/** ثبت دلیل رد عملیات توسط سرور. */
export function markMutationRejected(opId: string, reason: string): { pending: number } {
  return callCoreOrThrow('store.markRejected', { opId, reason });
}

/** پاک‌سازی عملیات‌های پایان‌یافته. */
export function pruneMutations(): { removed: number; pending: number } {
  return callCoreOrThrow('store.prune');
}

/** خواندن و ثبت نشانگر سینک (فقط روبه‌جلو). */
export function syncCursor(seq?: number): { cursor: number } {
  return seq === undefined ? callCoreOrThrow('store.cursor') : callCoreOrThrow('store.setCursor', { seq });
}

// ------------------------------------------------------------------ پشتیبان پیش‌نمایش

/**
 * پشتیبان اجرای پیش‌نمایش: تنها چیزی که تولید می‌کند سیاست دسترسی و منو از مانیفست
 * تولیدشده است (همان فایل خروجی سرور). هیچ محاسبهٔ دامنه‌ای اینجا انجام نمی‌شود.
 */
function fallback<T>(method: string, payload?: unknown): CoreEnvelope<T> {
  const input = (payload ?? {}) as Record<string, unknown>;

  if (method === 'core.bootstrap') {
    return {
      ok: true,
      result: {
        coreVersion: 'preview',
        schemaVersion: 1,
        methods: ['policy.menu', 'policy.authorize'],
        modules: MODULES.map(m => m.id),
      } as unknown as T,
    };
  }

  if (method === 'policy.menu') {
    const role = input.role as Role | undefined;
    if (!role) {
      return { ok: false, error: 'نقش برای ساخت منو لازم است' };
    }
    const entries: MenuEntry[] = MODULES.flatMap(module => {
      const screens = module.screens
        .filter(screen => screen.roles.includes(role))
        .map(screen => ({
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
      };
      // ماژول مشترک یک‌بار در خانهٔ خودش و یک‌بار برای هر ادارهٔ مشترک می‌آید
      // (مانند طبقه‌بندی مشاغل که بین امور کارکنان و برون‌سپاری مشترک است).
      return [
        { ...base, directorate: directoryOfDepartment(module.department) },
        ...module.sharedDepartments.map(dep => ({
          ...base,
          shared: true,
          department: dep,
          departmentTitleFa: departmentTitleOf(dep),
          directorate: directoryOfDepartment(dep),
        })),
      ];
    });
    return { ok: true, result: entries as unknown as T };
  }

  if (method === 'policy.authorize') {
    const role = input.role as Role | undefined;
    const moduleId = input.module as string | undefined;
    const module = MODULES.find(m => m.id === moduleId);
    const allowed = Boolean(role && module?.screens.some(s => s.roles.includes(role)));
    return { ok: true, result: { allowed } as unknown as T };
  }

  return {
    ok: false,
    error: 'در این اجرا هستهٔ بومی بار نشده است؛ محاسبات نیاز به بیلد بومی دارد. (npm run link-core:android)',
  };
}
