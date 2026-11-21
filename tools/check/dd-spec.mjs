/**
 * دروازهٔ ردیابی اسپک فرهنگ داده (DD-xx): هر بند باید تأیید موجود داشته باشد و هر آزمون
 * TestDDxx بند متناظر. منطق مشترک در spec-lib.mjs است.
 *
 * استعمال: node tools/check/dd-spec.mjs
 */
import { runSpecGate } from './spec-lib.mjs';

runSpecGate({ specPath: 'docs/data-dictionary-fa.md', label: 'اسپک فرهنگ داده', prefix: 'DD' });