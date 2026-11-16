# گروه‌های پارامتر و کدِ خوانندهٔ آن‌ها

کلید پارامتر به‌تنهایی اثر ندارد؛ باید کدی باشد که همان کلید را بخواند. جدول زیر مسیر
خواندن هر حوزه را نشان می‌دهد (منبع: `deps.Org.Params` و پیشوندها).

| پیشوند / گروه | خواننده | فایل |
| --- | --- | --- |
| گروه «حقوق و دستمزد» (کلیدهای بی‌پیشوند مثل `insurance.employeeRate`، `tax.exemption`، `allowance.housing`) | `collectParams` → `paramsFromValues` | `server/internal/modules/payroll/handlers.go` |
| `payroll.*` (مثل `payroll.overtimeFactor`، `payroll.monthlyHours`) | پارامترهای اقلام حقوق | `server/internal/modules/payroll/*`, `rules/elements.go` |
| `oilpay.*` | `oilPayParams` | `server/internal/modules/payroll/oilpay.go` |
| `comp34006.*` | `comp34006Params` → `rules.Comp34006ParamsFromText` | `server/internal/modules/payroll/comp34006.go` |
| `compensation.*` و `compensation.g20.*` | `fourp.go` / `g20.go` → `rules.CompensationParamsFromText` | `server/internal/modules/compensation/` |
| `mission.*` | محاسبهٔ مأموریت و کارانه | `server/internal/modules/payroll/missioncalc.go` |
| `veteran.*` | امتیاز ایثارگری | `server/internal/modules/*/veterans.go`, `rules/veteran.go` |
| `attendance.*`، `leave.*` | حضور و مرخصی | `server/internal/modules/attendance`, `rules/leave.go` |

## قاعده‌های نگاشت

1. **کلید باید دقیقاً همان چیزی باشد که کد می‌خواند.** اگر کد `values["insurance.ceiling"]`
   می‌خواند، کلید `payroll.insurance.ceiling` اثر نمی‌کند.
2. هر خواننده معمولاً **بازهٔ معقول** دارد؛ مقدار بیرون بازه بی‌صدا نادیده گرفته می‌شود
   (پس اگر پارامتر «اثر نکرد»، اول بازه را ببین).
3. پارامتر نبوده ⇒ مقدار پیش‌فرض کد (که همان مقدار سند/قانون است) به کار می‌رود.
4. **هر سه نظام پرداخت** بیمه/مالیات را از گروه «حقوق و دستمزد» می‌خوانند؛ کلید موازی نساز.

## نمونهٔ سرتاسری (مدل ۳۴۰۰۶)

```json
{ "key": "comp34006.rialFactor", "titleFa": "ضریب ریالی سال", "value": "7500",
  "unitFa": "ریال", "groupFa": "نظام جبران خدمات ۳۴۰۰۶" }
```

→ `node tools/gen/gen.mjs` → `server/internal/platform/orgs/orgs.gen.go`
→ `comp34006Params(deps, nil)` با پیشوند `comp34006.` می‌خواند
→ `rules.Comp34006ParamsFromText` با ارقام فارسی/درصد هم کنار می‌آید
→ آزمون: `TestComp34006PersianParameterText` و `TestComp34006AllParamsDriveCalculation`.

## بررسی سریع «آیا پارامتر به کد رسیده است؟»

```bash
# کجا خوانده می‌شود؟
rg -n 'comp34006\.rialFactor|"comp34006\."' server/internal | head
# در خروجی تولیدشده هست؟
rg -n 'comp34006.rialFactor' server/internal/platform/orgs/orgs.gen.go | head -2
# شمار کلیدهای یک گروه
node -e "const j=require('./contracts/organizations.json');console.log(j.defaultParams.filter(p=>p.key.startsWith('comp34006.')).length)"
```