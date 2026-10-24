/** انواع مشترک کلاینت که با قرارداد سرور و هستهٔ Rust یک‌راستا هستند. */

export type { Role, ModuleSpec, ScreenSpec } from '../navigation/modules.generated';

/**
 * یک آیتم منوی نقش‌محور (پاسخ `policy.menu` و `/api/v1/auth/me`).
 *
 * فیلدهای گروه‌بندی اختیاری‌اند چون منبع دادهٔ منو دو جاست: پاسخ سرور (که این فیلدها
 * را می‌فرستد) و منوی محلی تولیدشده از مانیفست. در هر دو حالت، همین فیلدها منو را
 * به شکل «حوزه > اداره > زیرواحد > ماژول > صفحه» می‌چینند و هیچ نگاشت دستی لازم نیست.
 */
export interface MenuEntry {
  moduleId: string;
  titleFa: string;
  order: number;
  screens: MenuScreen[];
  /** کلید حوزهٔ سازمانی (مثلاً `hr_capital`). */
  directorate?: string;
  /** کلید ادارهٔ منابع انسانی (مثلاً `staff_affairs`). */
  department?: string;
  /** عنوان ادارهٔ منابع انسانی — سطح دوم منو. */
  departmentTitleFa?: string;
  /** کلید و عنوان زیرواحد درون اداره — سطح سوم منو. */
  subUnit?: string;
  subUnitTitleFa?: string;
  /** درست اگر این ماژول در این اداره مهمان باشد (ماژول مشترک میان دو اداره). */
  shared?: boolean;
}

/** یک صفحه در منو. */
export interface MenuScreen {
  key: string;
  titleFa: string;
  component: string;
  icon: string;
}

/** کاربر واردشده. */
export interface SessionUser {
  username: string;
  displayName: string;
  role: import('../navigation/modules.generated').Role;
  employeeId?: string;
  departmentId?: string;
}

/** نقش‌های سامانه به ترتیب سند طراحی. */
export const ROLE_ORDER = ['admin', 'hr_manager', 'hr_spec', 'line_manager', 'employee'] as const;

/** دامنهٔ دادهٔ نقش در یک ماژول. */
export type Scope = 'all' | 'own_unit' | 'own';
