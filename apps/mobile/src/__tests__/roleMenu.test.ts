/**
 * آزمون منوی نقش‌محور.
 *
 * این آزمون همان انتظاراتی را بررسی می‌کند که در `core/crates/hrms-policy` (Rust) و
 * `server/internal/platform/rbac` (Go) هم آزموده می‌شود. چون هر سه از یک مانیفست
 * تولید می‌شوند، این سه‌گانه یک تضمین سرتاسری برای RBAC می‌سازد: چیزی که در کلاینت
 * پنهان است، در سرور هم رد می‌شود.
 */

import { MODULES } from '../navigation/modules.generated';
import { canOpenScreen, defaultRoute, localMenu, modulesForRole, screensForRole } from '../navigation/roleMenu';
import { SCREEN_REGISTRY, resolveScreen } from '../navigation/screenRegistry';
import type { Role } from '../navigation/modules.generated';

const ALL_ROLES: Role[] = ['admin', 'hr_manager', 'hr_spec', 'line_manager', 'employee'];

describe('مانیفست ماژول‌ها', () => {
  it('مانیفست، ماژول‌هایی با ترتیب پیوسته دارد', () => {
    expect(MODULES.length).toBeGreaterThan(0);
    expect(MODULES.map(m => m.order)).toEqual(MODULES.map((_, index) => index + 1));
  });

  it('هر صفحهٔ اعلام‌شده کامپوننت واقعی دارد', () => {
    const missing: string[] = [];
    MODULES.forEach(module =>
      module.screens.forEach(screen => {
        if (!resolveScreen(screen.component)) {
          missing.push(screen.component);
        }
      }),
    );
    expect(missing).toEqual([]);
    // دقیقاً همان کامپوننت‌هایی که مانیفست اعلام کرده — نه تعداد ثابت — در رجیستری‌اند.
    const declared = new Set(MODULES.flatMap(m => m.screens.map(s => s.component)));
    expect(new Set(Object.keys(SCREEN_REGISTRY))).toEqual(declared);
  });

  it('کلید صفحه‌ها در کل سامانه یکتاست', () => {
    const keys = MODULES.flatMap(m => m.screens.map(s => `${m.id}.${s.key}`));
    expect(new Set(keys).size).toBe(keys.length);
  });
});

describe('دسترسی نقش‌محور', () => {
  it('مدیر سیستم همهٔ ماژول‌ها را می‌بیند', () => {
    expect(modulesForRole('admin')).toHaveLength(MODULES.length);
    // هر ماژول یک ورودی دارد و ماژولِ مشترک (طبقه‌بندی مشاغل) به ازای هر ادارهٔ
    // هم‌مصرف‌کننده نسخهٔ دوم هم می‌گیرد تا از منوی هر دو اداره قابل باز شدن باشد.
    const expectedEntries = MODULES.reduce((sum, module) => sum + 1 + module.sharedDepartments.length, 0);
    expect(localMenu('admin')).toHaveLength(expectedEntries);
  });

  it('کارمند ماژول‌های مدیریتی را نمی‌بیند', () => {
    const ids = modulesForRole('employee').map(m => m.id);
    expect(ids).not.toContain('recruitment');
    expect(ids).not.toContain('performance');
    expect(ids).not.toContain('reports');
    expect(ids).not.toContain('settings');
    expect(ids).toEqual(expect.arrayContaining(['dashboard', 'employee', 'attendance', 'payroll', 'training', 'welfare']));
  });

  it('مدیر واحد تأیید مرخصی دارد ولی پردازش حقوق ندارد', () => {
    expect(canOpenScreen('line_manager', 'attendance', 'leaveApprove')).toBe(true);
    expect(canOpenScreen('line_manager', 'payroll', 'payslip')).toBe(true);
    expect(canOpenScreen('line_manager', 'payroll', 'payrollRun')).toBe(false);
  });

  it('تنظیمات کاربران فقط برای مدیر سیستم است', () => {
    expect(canOpenScreen('admin', 'settings', 'users')).toBe(true);
    expect(canOpenScreen('hr_manager', 'settings', 'users')).toBe(false);
    expect(canOpenScreen('hr_manager', 'settings', 'orgSettings')).toBe(true);
  });

  it('کارمند در ماژول امور کارکنان فقط پروندهٔ خود و خدمات پرسنلی را می‌بیند', () => {
    expect(screensForRole('employee', 'employee').map(s => s.key)).toEqual(['profile', 'personnelServices']);
    expect(screensForRole('hr_spec', 'employee').map(s => s.key)).toEqual(['employees', 'profile', 'personnelServices']);
  });

  it('هر نقش مسیر پیش‌فرض معتبر دارد', () => {
    ALL_ROLES.forEach(role => {
      const route = defaultRoute(role);
      expect(route).toBeDefined();
      expect(canOpenScreen(role, route!.moduleId, route!.screenKey)).toBe(true);
    });
  });
});
