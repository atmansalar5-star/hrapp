/** آزمون کمک‌های راست‌چین و قالب‌بندی اعداد فارسی. */

import { faDate, faDigits, faMoney, initials, normalizeDigits } from '../design/rtl';
import { layout, palette, statusColors } from '../design/tokens';

describe('ارقام فارسی', () => {
  it('ارقام لاتین را به فارسی تبدیل می‌کند', () => {
    expect(faDigits(1404)).toBe('۱۴۰۴');
    expect(faDigits('EMP-010')).toBe('EMP-۰۱۰');
  });

  it('ارقام فارسی و عربی را به لاتین برمی‌گرداند', () => {
    expect(normalizeDigits('۱۴۰۴/۰۷/۰۱')).toBe('1404/07/01');
    expect(normalizeDigits('٠١٢٣')).toBe('0123');
  });

  it('مبالغ ریالی را با جداکنندهٔ هزارگان و ارقام فارسی نشان می‌دهد', () => {
    expect(faMoney(34780000)).toBe('۳۴,۷۸۰,۰۰۰');
    expect(faMoney(-1500)).toBe('−۱,۵۰۰');
  });

  it('تاریخ شمسی را با ارقام فارسی نمایش می‌دهد', () => {
    expect(faDate('1404/07/01')).toBe('۱۴۰۴/۰۷/۰۱');
  });

  it('حرف نخست نام را برای آواتار می‌سازد', () => {
    expect(initials('سارا محمدی')).toHaveLength(3);
    expect(initials('نگار')).toBe('ن');
  });
});

describe('توکن‌های طراحی', () => {
  it('رنگ‌ها و اندازه‌های بستهٔ طراحی حفظ شده‌اند', () => {
    expect(palette.primary).toBe('#2563EB');
    expect(layout.sidebarWidth).toBe(272);
    expect(layout.headerHeight).toBe(64);
  });

  it('هر وضعیت کارمند برچسب فارسی دارد', () => {
    expect(statusColors.active!.labelFa).toBe('شاغل');
    expect(statusColors.terminated!.labelFa).toBe('ترک خدمت');
  });
});
