# دروازه‌های کیفیت — جدول کامل

ترتیب اجرا ارزان‌ترین به گران‌ترین است. هر گام اگر قرمز باشد، گام بعدی را اجرا نکنید.

| # | فرمان | چه چیزی را می‌سنجد | زمان تقریبی |
| --- | --- | --- | --- |
| ۱ | `node tools/gen/gen.mjs --check` | هم‌خوانی ۱۳ خروجی تولیدشده با مانیفست‌ها و `contracts/*.json` | ~۱ ثانیه |
| ۲ | `node tools/check/verify.mjs` | ساختار (حوزه/اداره/ماژول/صفحه)، رجیستری RN، مرز معماری ۲۰۰+ فایل Go، اعتبار اسکیل‌ها | ~۲ ثانیه |
| ۳ | `node tools/skills/check-skills.mjs` | قالب `.claude/skills/*/SKILL.md` (نام، توضیح، ارجاع‌های فایل) | <۱ ثانیه |
| ۴ | `cd server && go test -race -count=1 ./...` | همهٔ آزمون‌های سرور، شامل تطبیق قواعد با هسته | ۳ تا ۶ دقیقه |
| ۵ | `cd core && cargo test --workspace --all-features` | هستهٔ Rust | ۱ تا ۳ دقیقه |
| ۶ | `cd core && cargo run -p hrms-conformance` | تطبیق قواعد Go و Rust با fixtureهای طلایی | ~۳۰ ثانیه |

`make check` = ۱+۲+۳ (داخل verify) + ۴ + ۵ + ۶.

## اجرای سریع زیرمجموعه

```bash
# فقط چیزی که عوض کردی
cd server && go test ./internal/rules/ ./internal/modules/<module>/ -count=1

# آزمون‌های سرتاسری bootstrap گران‌اند (~۲ تا ۴ دقیقه)؛ فقط وقتی سیم‌کشی/دسترسی عوض شده
cd server && go test ./internal/bootstrap/ -count=1

# فقط یک آزمون
cd server && go test ./internal/rules/ -run 'Comp34006' -count=1 -v
```

## خطاهای رایج و کار درست

| پیام | معنا | کار |
| --- | --- | --- |
| «دروازهٔ انحراف قرارداد شکست خورد» | `contracts/` یا `module.json` عوض شده و خروجی بازتولید نشده | `node tools/gen/gen.mjs` سپس دوباره `--check` |
| «کامپوننت RN یافته‌نشده» | صفحه در `module.json` هست، ثبت RN نیست | فایل stub صفحه + ثبت در `screenRegistry.ts` |
| «مرز ماژول» | import به بستهٔ داخلی ماژول دیگر | `deps.Ports` یا `internal/contracts` |
| `EPERM` / `Access is denied` روی خروجی تولیدشده | سندباکس اجازهٔ بازنویسی فایل موجود به فرایند فرزند نمی‌دهد | پشتیبان بگیر، فایل‌های تولیدشده را حذف کن، `gen.mjs` را دوباره اجرا کن |
| `go: downloading …` طولانی | کش ماژول خالی است | یک‌بار صبر کن؛ اگر شبکه بسته است، `GOFLAGS=-mod=mod` یا کش محلی |

## قاعدهٔ گزارش

هر ادعای «سبز است» باید یک فرمان اجراشده پشتش باشد. اگر ابزاری نصب نبود (`make`، `cargo`)،
همان گام را دستی اجرا کن یا صریح بنویس که اجرا نشد.