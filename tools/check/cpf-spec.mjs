/**
 * دروازهٔ ردیابی اسپک زیرفرم «حقوق و مزایا» (CPF-xx): هر بند باید تأیید موجود داشته باشد
 * و هر آزمون TestCPFxx بند متناظر. منطق مشترک در spec-lib.mjs است.
 *
 * استعمال: node tools/check/cpf-spec.mjs
 */
import { runSpecGate } from './spec-lib.mjs';

runSpecGate({ specPath: 'docs/contract-pay-form-spec-fa.md', label: 'اسپک زیرفرم حقوق قرارداد', prefix: 'CPF' });