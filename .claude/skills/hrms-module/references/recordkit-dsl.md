# DSL رکوردکیت (نوع رکورد، فیلد، گردش‌کار)

همه در `server/internal/platform/recordkit/descriptor.go` تعریف شده‌اند و هر ماژول آن‌ها را
در `domain/descriptor.go` صدا می‌زند.

## `recordkit.Kind`

```go
recordkit.Kind{
	Key:            "oilPayslips",   // کلید نوع رکورد (لاتین، یکتا در ماژول)
	TitleFa:        "فیش حقوقی صندوق نفت",
	IDPrefix:       "OILP",          // پیشوند شناسه — یکتا در کل _سامانه_
	EmployeeScoped: true,            // رکورد به یک کارمند گره می‌خورد
	Fields:         recordkit.Fields(...),
	Statuses:       recordkit.Statuses(...),
	Initial:        "draft",
	Transitions:    recordkit.Steps(...),
	Actions:        recordkit.Actions(...),
	Validate:       validateFn,      // اختیاری: قاعده‌های کسب‌وکاری
	EditableStatuses: []string{"draft", "active"},
	HistoryDate:    "effectiveDate", // فیلد تاریخ برای تاریخ‌گذاری مؤثر
}
```

## فیلدها

`Fields("key|برچسب فارسی|type|required")`

| type | کاربرد |
| --- | --- |
| `text` | متن (پیش‌فرض اگر خالی بماند) |
| `date` | تاریخ شمسی `YYYY/MM/DD` |
| `number` | عدد (ساعت، روز، امتیاز) |
| `money` | مبلغ ریالی (نمایش با جداکننده) |
| `select` | انتخاب از فهرست |

`required` = `1` یا `0`. مثال: `"gross|جمع ناخالص (ریال)|money|0"`.

## وضعیت‌ها و انتقال‌ها

```go
Statuses:  recordkit.Statuses("draft|پیش‌نویس", "approved|تأییدشده", "issued|صادرشده")
Initial:   "draft"
Transitions: recordkit.Steps("draft|approved,cancelled", "approved|issued", "issued|paid")
```

- `Statuses("value|برچسب")` — برچسب فارسی اجباری‌ست برای UI.
- `Steps("مبدا|مقصد۱,مقصد۲")` — هر انتقال مجاز یک رشته.

## اقدام‌ها

`Actions("value|برچسب فارسی|style|مقصد|noteRequired|roles|scope")`

| قسمت | معنا |
| --- | --- |
| `style` | `primary` / `ghost` / `danger` (خالی ⇒ `ghost`) |
| `مقصد` | وضعیت بعدی؛ خالی ⇒ وضعیت عوض نمی‌شود و فقط رخداد ثبت می‌شود |
| `noteRequired` | `1` ⇒ دلیل/یادداشت اجباری |
| `roles` | نقش‌های مجاز با کاما (خالی ⇒ همهٔ نقش‌های دارای دسترسی اقدام) |
| `scope` | `all` / `own_unit` / `own` (حداقل دامنهٔ لازم) |

مثال واقعی: `"approve|تأیید فیش|primary|approved|0|hr_manager"`.

## قاعده‌های آزمون‌شده

- پیشوند شناسهٔ هر نوع رکورد باید در کل سامانه یکتا باشد (آزمون `bootstrap`).
- شناسهٔ ساخته‌شده باید همان پیشوند نوع خودش را داشته باشد.
- رکورد کارمندمحور باید `EmployeeScoped: true` داشته باشد و `employeeId` بگیرد.
- فیلدهای `Validate` باید پیام فارسی با `&recordkit.RuleError{Message: "..."}` بدهند.

## ماژول‌های موجود و پیشوندهایشان

فهرست کامل با `Select-String 'IDPrefix' server/internal/modules/*/domain/descriptor.go`؛
و شمار ماژول/صفحه با `node tools/check/verify.mjs` (امروز: ۳۱ ماژول، ۱۳۲ صفحه).