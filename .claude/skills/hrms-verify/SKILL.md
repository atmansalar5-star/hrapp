---
name: hrms-verify
description: دروازه‌های کیفیت مخزن HRMS پیش از هر کامیت یا تحویل — انحراف قرارداد (gen --check)، مرز معماری (verify.mjs)، آزمون‌های Go و Rust، تطبیق قواعد و قالب‌بندی. وقتی تغییری در contracts/ یا server/internal/modules/ دادید، یا کاربر گفت «تست بگیر»، «دروازه‌ها را چک کن»، «make check»، «چرا CI قرمز است»، این اسکیل را بار کنید.
metadata:
  project: hrapp
  stack: go,rust,node
---

# دروازه‌های کیفیت HRMS

ترتیب کار **همیشه** همین است؛ هر گام ارزان‌تر از گام بعدی است و خطا را زودتر نشان می‌دهد.

```bash
# ۱) انحراف قرارداد: خروجی‌های تولیدشده با مانیفست‌ها یکی‌اند؟
node tools/gen/gen.mjs --check

# ۲) مرز معماری، شمار ماژول/صفحه، رجیستری RN، مرز ماژول‌های Go
node tools/check/verify.mjs

# ۳) آزمون‌های سرور (شامل تطبیق قواعد با هستهٔ Rust)
cd server && go test -race -count=1 ./...

# ۴) فقط وقتی core/ یا rules/ عوض شده: آزمون و تطبیق هسته
cd core && cargo test --workspace --all-features
cd core && cargo run -p hrms-conformance
```

معادل یک‌خطی: `make check` (verify + test-go + test-core + conformance).
اگر `make` روی ماشین نیست، دقیقاً همین چهار فرمان را جدا اجرا کنید و در گزارش بنویسید که make نصب نبود.

## قاعدهٔ طلایی انحراف قرارداد

اگر `gen.mjs --check` شکست، **هرگز** فایل تولیدشده را دستی درست نکنید. منبع حقیقت
`contracts/*.json` و `server/internal/modules/*/module.json` است:

```bash
node tools/gen/gen.mjs          # بازتولید ۱۳ خروجی
node tools/gen/gen.mjs --check  # باید «همهٔ ۱۳ خروجی … همخوان است» بدهد
```

## اگر نوشتن خروجی تولیدشده «Access is denied» داد

در محیط‌های سندباکس‌شده، فرایند فرزند نمی‌تواند فایل موجود را بازنویسی کند. راه‌حل: پوشهٔ
خروجی را پاک کنید تا بازتولید از نو بنویسد (پیش از حذف، پشتیبان بگیرید):

```bash
mkdir -p .dsh-tmp/gen-backup
# پشتیبان از ۱۳ فایل، سپس حذف همان‌ها، سپس node tools/gen/gen.mjs
```

## عیب‌یابی سریع

| نشانه | علت رایج | کار |
| --- | --- | --- |
| `verify.mjs` می‌گوید «کامپوننت RN یافته‌نشده» | صفحهٔ تازه در `module.json` هست ولی صفحهٔ RN یا `screenRegistry.ts` نه | فایل `apps/mobile/src/features/<module>/<Screen>.tsx` را بساز و در `screenRegistry.ts` ثبت کن |
| `verify.mjs` می‌گوید «مرز ماژول» | یک ماژول به بستهٔ داخلی ماژول دیگر import زده | از `deps.Ports` یا قراردادهای `internal/contracts` استفاده کن |
| `gofmt -l` فایل نشان می‌دهد | فایل تازه یا سرصفحهٔ افزوده‌شده | `cd server && gofmt -w <file>` |
| آزمون `bootstrap` کند است | جریان‌های سرتاسری با Postgres/فایل | فقط بستهٔ عوض‌شده را اجرا کن: `go test ./internal/rules/` |

## پیش از گزارش به کاربر

- خروجی هر چهار گام را جمع کن (سبز/قرمز) و **همان‌ها** را در پاسخ بیاور؛ ادعای «همه سبز
  است» بدون اجرا ممنوع.
- اگر چیزی را نتوانستی اجرا کنی (مثلاً `make` نیست، `cargo` نیست)، صریح بگو و گام معادلش را
  بنویس.

## مراجع و ابزار

- `references/gates.md` — جدول کامل دروازه‌ها، زیرمجموعه‌های سریع و خطاهای رایج.
- `scripts/gates.mjs` — اجرای ترتیبی دروازه‌ها با خلاصهٔ زمانی:
  `node .claude/skills/hrms-verify/scripts/gates.mjs --with-tests`
