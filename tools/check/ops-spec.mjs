/**
 * دروازهٔ ردیابی اسپک آمادگی عملیاتی: هر بند OPS باید تأیید موجود داشته باشد و هر آزمون
 * TestOPSxx بند متناظر. منطق مشترک در spec-lib.mjs است.
 *
 * استعمال: node tools/check/ops-spec.mjs
 */
import { runSpecGate } from './spec-lib.mjs';

runSpecGate({ specPath: 'docs/operational-spec-fa.md', label: 'اسپک عملیاتی', prefix: 'OPS' });