---
name: hrms-module
description: افزودن یا تغییر ماژول، صفحه و نوع رکورد در HRMS — مانیفست module.json، دامنه (recordkit)، مسیرهای HTTP، ثبت در رجیستری، صفحهٔ React Native و صفحهٔ نمونهٔ وب. وقتی کاربر می‌گوید «ماژول تازه بساز»، «صفحه اضافه کن»، «یک نوع رکورد/گردش‌کار جدید لازم است»، این اسکیل را بار کنید.
metadata:
  project: hrapp
  area: server/internal/modules
---

# ساخت ماژول، صفحه و رکورد در HRMS

## نقشهٔ جای فایل‌ها

```
server/internal/modules/<module>/
├── module.json        مانیفست: عنوان، ترتیب، آیکون، صفحه‌ها (pages[])
├── module.go          Register(mux, deps): Mount رکوردها + ثبت مسیرها
├── handlers.go        پایانه‌های HTTP اختصاصی (اختیاری)
├── domain/descriptor.go   Descriptor() → Kinds[]: نوع رکورد، فیلدها، وضعیت‌ها، اقدام‌ها
└── domain/<x>_test.go     آزمون اعتبارسنجی و قواعد دامنه
apps/mobile/src/features/<module>/<Screen>.tsx    صفحهٔ RN (stub کافی است)
apps/mobile/src/navigation/screenRegistry.ts      import + ثبت کامپوننت
design/prototype/js/live_<page>.js                صفحهٔ زندهٔ نمونهٔ وب
design/prototype/index.html                        <script src="js/live_<page>.js?v=1">
contracts/departments.json                         جای سازمانی ماژول در اداره/زیرواحد
```

## ترتیب کار

1. **جای سازمانی**: در `contracts/departments.json` مشخص کن ماژول زیر کدام اداره/زیرواحد
   می‌نشیند؛ این فایل منبع حقیقت است و `docs/generated/pages-fa.md` از آن ساخته می‌شود.
2. **مانیفست** `module.json`: `id`, `order`, `titleFa`, `summaryFa` و `pages[]`. هر صفحه:
   `key`, `order`, `icon` (از فهرست آیکون‌های موجود؛ با `Select-String '"icon"' server/internal/modules/*/module.json`
   ببین), `titleFa`, `component`, `purposeFa`, `actionsFa[]`, `roles[]`.
3. **دامنه** در `domain/descriptor.go`:

```go
func Descriptor() recordkit.Descriptor {
	return recordkit.Descriptor{Module: "<module>", TitleFa: "…", Kinds: []recordkit.Kind{
		{Key: "items", TitleFa: "…", IDPrefix: "XYZ", EmployeeScoped: false,
		 Fields: recordkit.Fields("title|عنوان|text|1", "amount|مبلغ (ریال)|money|0"),
		 Statuses: recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده"),
		 Initial: "draft", Transitions: recordkit.Steps("draft|approved"),
		 Actions: recordkit.Actions("approve|تأیید|primary|approved|0")},
	}}
}
```

`IDPrefix` باید **یکتا در کل سامانه** باشد؛ آزمون `bootstrap` یکتایی همهٔ پیشوندها را می‌سنجد.

4. **سیم‌کشی** در `module.go`: `svc := recordkit.Mount(mux, deps, domain.Descriptor(), domain.Seed(), "/api/v1/<module>")`
   و مسیرهای اختصاصی با `deps.Require("<module>", "read"|"write", fn)`.
5. **کلاینت و نمونهٔ وب**: فایل stub صفحهٔ RN + ثبت در `screenRegistry.ts`؛ فایل `live_<page>.js`
   با `LIVE_PAGES.<page> = pageXxxLive;` و `<script>` در `index.html`.
6. **بازتولید و سنجش**: `node tools/gen/gen.mjs && node tools/gen/gen.mjs --check` سپس
   `node tools/check/verify.mjs` (شمار صفحه‌ها و «کامپوننت RN یافته‌شده» باید هم‌خوان باشد).

## قاعده‌های تغییرناپذیر

- **هیچ قاعدهٔ کسب‌وکاری در UI نیست**: اعتبارسنجی و محاسبه در دامنه/`rules`؛ صفحه فقط نمایش و فراخوانی API.
- **مرز ماژول**: ماژول A نباید بستهٔ داخلی ماژول B را import کند؛ از `deps.Ports` یا
  `server/internal/contracts` استفاده کن (verify.mjs همین را می‌سنجد).
- **فایل تولیدشده دستی ویرایش نمی‌شود**: `registry.gen.go`، `departments.gen.go`،
  `menu.generated.js`، `modules.generated.ts` و … 모두 `gen.mjs` می‌سازد.
- شناسهٔ رکورد باید پیشوند نوع خود را داشته باشد؛ آزمون‌های `bootstrap` این را هم می‌سنجند.

## مراجع

- `references/recordkit-dsl.md` — قالب دقیق Kind، فیلدها، وضعیت‌ها، اقدام‌ها و انتقال‌های رکوردکیت.
