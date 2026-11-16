#!/usr/bin/env node
/**
 * gates.mjs — اجرای دروازه‌های کیفیت HRMS و گزارش یک‌خطی هر گام.
 *
 * چرا اسکریپت و نه چهار فرمان دستی: عامل‌ها اغلب یک گام را جا می‌اندازند. اینجا ترتیب
 * درست ثابت است، هر گام با زمان و کد خروج گزارش می‌شود و در اولین شکست متوقف می‌شود
 * (مگر با --keep-going).
 *
 * اجرا:
 *   node .claude/skills/hrms-verify/scripts/gates.mjs                # ۱ تا ۳ (سریع)
 *   node .claude/skills/hrms-verify/scripts/gates.mjs --with-tests   # + آزمون‌های Go
 *   node .claude/skills/hrms-verify/scripts/gates.mjs --all          # + Rust و تطبیق
 *   node .claude/skills/hrms-verify/scripts/gates.mjs --keep-going   # شکست را رد کن و ادامه بده
 */
import { spawnSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { join } from 'node:path';

const argv = process.argv.slice(2);
const withTests = argv.includes('--with-tests') || argv.includes('--all');
const all = argv.includes('--all');
const keepGoing = argv.includes('--keep-going');

const ROOT = process.cwd();

const steps = [
  { name: 'انحراف قرارداد', cmd: 'node', args: ['tools/gen/gen.mjs', '--check'], need: 'tools/gen/gen.mjs' },
  { name: 'مرز معماری و رجیستری', cmd: 'node', args: ['tools/check/verify.mjs'], need: 'tools/check/verify.mjs' },
  { name: 'قالب اسکیل‌ها', cmd: 'node', args: ['tools/skills/check-skills.mjs'], need: 'tools/skills/check-skills.mjs' },
];
if (withTests) {
  steps.push({ name: 'آزمون‌های سرور (Go)', cmd: 'go', args: ['test', '-count=1', './...'], cwd: join(ROOT, 'server'), need: 'server' });
}
if (all) {
  steps.push({ name: 'آزمون‌های هسته (Rust)', cmd: 'cargo', args: ['test', '--workspace', '--all-features'], cwd: join(ROOT, 'core'), need: 'core' });
  steps.push({ name: 'تطبیق قواعد هسته', cmd: 'cargo', args: ['run', '-p', 'hrms-conformance'], cwd: join(ROOT, 'core'), need: 'core' });
}

const results = [];
let failed = 0;
for (const step of steps) {
  if (!existsSync(join(ROOT, step.need))) {
    results.push({ name: step.name, status: 'SKIP', ms: 0, note: `نیازمند ${step.need} نبود` });
    continue;
  }
  const started = Date.now();
  process.stdout.write(`⏳ ${step.name} …\n`);
  // بدون shell اجرا می‌شود (پرهیز از تفسیر آرگومان‌ها)؛ node/go/cargo همه exe هستند.
  const run = spawnSync(step.cmd, step.args, {
    cwd: step.cwd || ROOT,
    stdio: 'inherit',
  });
  const ms = Date.now() - started;
  if (run.error && run.error.code === 'ENOENT') {
    results.push({ name: step.name, status: 'SKIP', ms, note: `${step.cmd} نصب نیست` });
    continue;
  }
  const ok = run.status === 0;
  if (!ok) failed++;
  results.push({ name: step.name, status: ok ? 'PASS' : 'FAIL', ms, note: ok ? '' : `کد خروج ${run.status}` });
  if (!ok && !keepGoing) break;
}

console.log('\n=== خلاصهٔ دروازه‌ها ===');
for (const r of results) {
  console.log(`  ${r.status.padEnd(4)} ${r.name.padEnd(26)} ${(r.ms / 1000).toFixed(1)}s ${r.note}`);
}
if (failed) {
  console.error(`\n✖ ${failed} گام شکست خورد.`);
  process.exit(1);
}
console.log('\n✓ همهٔ گام‌های اجراشده سبز است.');