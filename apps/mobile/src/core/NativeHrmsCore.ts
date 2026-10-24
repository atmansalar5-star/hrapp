/**
 * مشخصات TurboModule برای هستهٔ بومی.
 *
 * این فایل ورودی Codegen در React Native (معماری جدید) است. لایهٔ بومی موبایل با
 * `uniffi` از `core/crates/hrms-ffi` ساخته می‌شود و دو تابع زیر را در اختیار
 * جاوااسکریپت می‌گذارد. سطح پل باریک است تا تغییر دامنه نیازی به بازتولید لایهٔ
 * بومی نداشته باشد: ورودی و خروجی همیشه JSON است.
 */
import type { TurboModule } from 'react-native';
import { TurboModuleRegistry } from 'react-native';

export interface Spec extends TurboModule {
  /**
   * فراخوانی یک روش هسته.
   * @param method نام روش (فهرست در `hrms-ffi::METHODS`)
   * @param payloadJson بار JSON؛ در نبود مقدار `"null"`
   * @returns پاسخ JSON به شکل `{"ok":true,"result":…}` یا `{"ok":false,"error":"…"}`
   */
  call(method: string, payloadJson: string): string;

  /** نسخهٔ هستهٔ بومی برای تشخیص ناهم‌خوانی با کلاینت. */
  version(): string;
}

export default TurboModuleRegistry.getEnforcing<Spec>('HrmsCore');
