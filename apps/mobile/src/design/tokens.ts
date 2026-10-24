/**
 * توکن‌های طراحی سامانه.
 *
 * این مقادیر عیناً از بستهٔ طراحی موجود (`design/prototype/index.html`) برداشته شده‌اند
 * تا کلاینت React Native با همان زبان بصری پروتوتایپ تأییدشده کار کند. هیچ رنگی در
 * کامپوننت‌ها مستقیم نوشته نمی‌شود؛ همه از اینجا می‌آید تا تغییر هویت بصری یک‌جا انجام شود.
 */

/** رنگ‌های پایه — هم‌خوان با متغیرهای CSS بستهٔ طراحی. */
export const palette = {
  primary: '#2563EB',
  primaryDark: '#1D4ED8',
  primaryLight: '#EFF6FF',
  teal: '#0D9488',
  amber: '#D97706',
  red: '#DC2626',
  green: '#16A34A',
  purple: '#7C3AED',
  text: '#1F2937',
  muted: '#6B7280',
  border: '#E5E7EB',
  surface: '#FFFFFF',
  bg: '#F4F6FB',
} as const;

/** رنگ‌های حالت تیره (کلاس .dark در پروتوتایپ). */
export const darkPalette = {
  ...palette,
  text: '#F3F4F6',
  muted: '#9CA3AF',
  border: '#334155',
  surface: '#0F172A',
  bg: '#020617',
  primaryLight: '#172554',
} as const;

export type Palette = { readonly [K in keyof typeof palette]: string };

/** اندازه‌های چیدمان. */
export const layout = {
  /** عرض منوی کنار در حالت راست‌چین. */
  sidebarWidth: 272,
  headerHeight: 64,
  radius: 14,
  radiusSm: 10,
  hairline: 1,
} as const;

/** فاصله‌های استاندارد. */
export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
} as const;

/** اندازه‌های قلم. */
export const typography = {
  fontFamily: 'Vazirmatn',
  fontFamilyFallback: 'Tahoma',
  size: {
    xxs: 10,
    xs: 12,
    sm: 13,
    md: 14,
    lg: 16,
    xl: 20,
    xxl: 26,
  },
  weight: {
    regular: '400',
    medium: '500',
    bold: '700',
    black: '800',
  },
  /** خط پایهٔ متن فارسی؛ متن راست‌چین به ارتفاع خط بیشتری نیاز دارد. */
  lineHeightRatio: 1.8,
} as const;

/** سایه‌ها (معادل --shadow و --shadow-lg). */
export const shadows = {
  sm: {
    shadowColor: '#101828',
    shadowOpacity: 0.08,
    shadowRadius: 3,
    shadowOffset: { width: 0, height: 1 },
    elevation: 1,
  },
  lg: {
    shadowColor: '#101828',
    shadowOpacity: 0.14,
    shadowRadius: 32,
    shadowOffset: { width: 0, height: 12 },
    elevation: 8,
  },
} as const;

/** رنگ وضعیت‌ها؛ در جدول‌ها و برچسب‌ها استفاده می‌شود. */
export const statusColors: Record<string, { bg: string; fg: string; labelFa: string }> = {
  active: { bg: '#DCFCE7', fg: '#166534', labelFa: 'شاغل' },
  leave: { bg: '#FEF3C7', fg: '#92400E', labelFa: 'مرخصی' },
  probation: { bg: '#DBEAFE', fg: '#1E40AF', labelFa: 'آزمایشی' },
  terminated: { bg: '#FEE2E2', fg: '#991B1B', labelFa: 'ترک خدمت' },
};
