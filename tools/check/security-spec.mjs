/**
 * دروازهٔ ردیابی اسپک امنیتی (spec-driven): هر بند REQ باید تأیید موجود داشته باشد و هر
 * آزمون TestREQxx در سرور بند متناظر. منطق مشترک در spec-lib.mjs است.
 *
 * استعمال: node tools/check/security-spec.mjs
 */
import { runSpecGate } from './spec-lib.mjs';

runSpecGate({ specPath: 'docs/security-spec-fa.md', label: 'اسپک امنیتی', prefix: 'REQ' });