/**
 * دروازهٔ ردیابی اسپک «تخصیص فوق العاده‌های حقوق» (ASG-xx): جدول رسمی کدها، دامنه‌ها،
 * روش‌های محاسبه، مشمولیت کسور، API و صفحهٔ رابط. منطق مشترک در spec-lib.mjs است.
 *
 * استعمال: node tools/check/asg-spec.mjs
 */
import { runSpecGate } from './spec-lib.mjs';

runSpecGate({ specPath: 'docs/allowance-assignment-spec-fa.md', label: 'اسپک تخصیص فوق العاده', prefix: 'ASG' });