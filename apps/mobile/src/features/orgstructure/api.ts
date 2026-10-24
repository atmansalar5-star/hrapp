/**
 * لایهٔ دادهٔ ماژول ساختار سازمانی.
 *
 * منبع اصلی واحدها و پست‌ها در پایان فاز ۳ همین ماژول است
 * (`/api/v1/orgstructure/units`). تا آن زمان مخزن واحدها در ماژول امور کارکنان
 * نگهداری می‌شود، چون پروندهٔ کارمند و فیلتر واحدها به همان داده وابسته‌اند؛ پس این
 * تابع نخست نقطهٔ پایانی اختصاصی ماژول را می‌خواند و اگر سرور آن را «فاز بعد» اعلام
 * کند (۵۰۱) به همان منبع اصلی برمی‌گردد. صفحه در هر دو حالت کار می‌کند و پس از
 * انتقال پیاده‌سازی، هیچ تغییری در رابط لازم نیست.
 */

import { ApiError, type Client } from '../../api/client';
import type { Department } from '../../api/contracts';

/** دریافت درخت واحدهای سازمانی. */
export async function fetchOrgUnits(client: Client): Promise<Department[]> {
  try {
    return await client.orgUnits();
  } catch (error) {
    if (error instanceof ApiError && error.status === 501) {
      return client.departments();
    }
    throw error;
  }
}
