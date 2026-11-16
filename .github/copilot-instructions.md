# راهنمای عامل‌های کدنویس برای این مخزن

> این فایل برای GitHub Copilot و عامل‌های مشابه است و **منبع حقیقت نیست**؛ فقط به دو فایل
> اصلی ارجاع می‌دهد.

1. **`AGENTS.md`** (ریشهٔ مخزن) — قاعده‌های کاری، زبان فارسی/RTL، محیط ویندوز، گیت و بک‌آپ.
2. **`.claude/skills/`** — کتابخانهٔ اسکیل‌های عامل (استاندارد Agent Skills): برای هر حوزه یک
   پوشه با `SKILL.md`.

## پیش از تغییر کد

| اگر کار دربارهٔ… است | این اسکیل را بخوان |
| --- | --- |
| دروازه‌ها، `make check`، انحراف قرارداد، CI | `.claude/skills/hrms-verify/SKILL.md` |
| پارامتر تازه یا پارامتری‌کردن یک عدد | `.claude/skills/hrms-parameter/SKILL.md` |
| ماژول/صفحه/نوع رکورد تازه | `.claude/skills/hrms-module/SKILL.md` |
| متن فارسی، RTL، ارقام و تاریخ شمسی | `.claude/skills/hrms-persian-text/SKILL.md` |
| مدل‌های پرداخت و فیش حقوقی | `.claude/skills/hrms-compensation/SKILL.md` |
| بستهٔ تست نفوذ و حذف اسرار | `.claude/skills/hrms-pentest-package/SKILL.md` |
| سرور، سرویس، ری‌استارت | `.claude/skills/hrms-deploy-windows/SKILL.md` |
| سند Word/Excel/PDF | `.claude/skills/hrms-office-export/SKILL.md` |

## سه قاعدهٔ کوتاه

- خروجی‌های `tools/gen/gen.mjs` دستی ویرایش نمی‌شوند؛ منبع حقیقت `contracts/` و مانیفست ماژول‌ها است.
- هیچ قاعدهٔ کسب‌وکاری در UI نوشته نمی‌شود (دامنه/`internal/rules`).
- پاسخ‌ها و متن‌های محصول فارسی‌اند؛ کلیدهای ماشین‌خوان لاتین می‌مانند.

پیش از تحویل: `node tools/gen/gen.mjs --check && node tools/check/verify.mjs && node tools/skills/check-skills.mjs && (cd server && go test ./...)`.