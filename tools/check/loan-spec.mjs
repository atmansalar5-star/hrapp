/**
 * دروازهٔ ردیابی اسپک «صورت حساب و اقساط وام» (LOAN-xx): شرایط وام، جدول اقساط ماهانه،
 * مانده بدهی، تاریخچه، کد رسمی فیش، API صورت حساب، کسر در فیش، گزارش و صفحهٔ رابط.
 * منطق مشترک در spec-lib.mjs است.
 *
 * استعمال: node tools/check/loan-spec.mjs
 */
import { runSpecGate } from './spec-lib.mjs';

runSpecGate({ specPath: 'docs/loan-statement-spec-fa.md', label: 'اسپک صورت حساب وام', prefix: 'LOAN' });