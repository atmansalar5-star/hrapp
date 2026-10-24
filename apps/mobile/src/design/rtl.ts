/**
 * کمکی‌های راست‌چین (RTL).
 *
 * الزام سند طراحی: منوی کنار در سمت راست، متن‌ها راست‌چین و ترتیب ستون‌های جدول از
 * راست به چپ. React Native با `I18nManager.forceRTL(true)` این کار را انجام می‌دهد،
 * اما دو نکته را باید خودمان رعایت کنیم:
 *
 * ۱. `forceRTL` فقط در اجرای بعدی برنامه اثر می‌گذارد، پس در همان اولین اجرا هم باید
 *    ظاهر درست باشد؛ از `row` معکوس‌شده استفاده می‌کنیم نه وابستگی به ترتیب DOM.
 * ۲. اعداد و مبالغ باید با ارقام فارسی و بدون جابه‌جایی علامت منفی نمایش داده شوند.
 */

import { I18nManager, Platform, StyleSheet, TextStyle, ViewStyle } from 'react-native';

/** فعال‌کردن راست‌چین سراسری؛ پیش از رندر اول صدا زده می‌شود. */
export function enableRTL(): void {
  I18nManager.allowRTL(true);
  I18nManager.forceRTL(true);
}

/** آیا چیدمان به‌صورت بومی راست‌چین است. */
export const isRTLLayout: boolean = I18nManager.isRTL || Platform.OS === 'android';

/** سبک پایهٔ هر متن فارسی: راست‌چین با ارتفاع خط مناسب. */
export const rtlText: TextStyle = {
  textAlign: 'right',
  writingDirection: 'rtl',
};

/** سبک ردیف راست‌چین (آیکون در راست، متن پس از آن). */
export const rtlRow: ViewStyle = {
  flexDirection: 'row-reverse',
  alignItems: 'center',
};

/** سبک محتوایی که باید چپ‌چین بماند: کد ملی، شمارهٔ حساب، ایمیل، تاریخ میلادی. */
export const ltrText: TextStyle = {
  textAlign: 'left',
  writingDirection: 'ltr',
};

const PERSIAN_DIGITS = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];

/** تبدیل ارقام لاتین به فارسی. */
export function faDigits(value: string | number): string {
  return String(value).replace(/\d/g, d => PERSIAN_DIGITS[Number(d)] ?? d);
}

/** تبدیل ارقام فارسی/عربی به لاتین (ورودی کاربر). */
export function normalizeDigits(value: string): string {
  return value
    .replace(/[۰-۹]/g, d => String(d.charCodeAt(0) - 0x06f0))
    .replace(/[٠-٩]/g, d => String(d.charCodeAt(0) - 0x0660));
}

/** قالب‌بندی مبلغ ریالی با جداکنندهٔ هزارگان و ارقام فارسی. */
export function faMoney(amount: number): string {
  const sign = amount < 0 ? '−' : '';
  const grouped = Math.abs(amount).toLocaleString('en-US');
  return `${sign}${faDigits(grouped)}`;
}

/** نمایش تاریخ شمسی ذخیره‌شده (YYYY/MM/DD) با ارقام فارسی. */
export function faDate(jalali: string): string {
  return faDigits(jalali);
}

/** تبدیل «کارشناس ارشد مالی» به «ک.ا.م» برای آواتار. */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).slice(0, 2);
  return parts.map(p => p.charAt(0)).join('‌');
}

/** سبک‌های مشترک راست‌چین که در همهٔ صفحه‌ها استفاده می‌شود. */
export const rtlStyles = StyleSheet.create({
  fill: { flex: 1 },
  center: { alignItems: 'center', justifyContent: 'center' },
  row: rtlRow,
  text: rtlText,
  ltr: ltrText,
});
