/**
 * دروازهٔ ردیابی اسپک «دستهٔ پرداخت» (PAY-xx): گروه جدول ۳، روش‌های پرداخت و جداسازی
 * نظام‌های ۴پی/جی‌بیست. منطق مشترک در spec-lib.mjs است.
 *
 * استعمال: node tools/check/pcs-spec.mjs
 */
import { runSpecGate } from './spec-lib.mjs';

runSpecGate({ specPath: 'docs/pay-classification-spec-fa.md', label: 'اسپک دستهٔ پرداخت', prefix: 'PAY' });