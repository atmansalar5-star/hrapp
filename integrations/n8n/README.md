# یکپارچه‌سازی n8n با سامانهٔ منابع انسانی

این پوشه «لایهٔ یکپارچه‌سازی» است: رویدادهای سامانه به n8n می‌رسد و n8n آن‌ها را به
پیامک، ایمیل، پیام‌رسان یا سامانه‌های دیگر می‌رساند. **هیچ قاعدهٔ کسب‌وکاری این‌جا نیست؛**
محاسبات حقوقی و آیین‌نامه‌ای همان‌جای همیشگی (`server/internal/rules`) می‌مانند.

## ۱) پیش‌نیازها

| جزء | مسیر این ماشین |
|---|---|
| Node 22 LTS قابل‌حمل | `%LOCALAPPDATA%\tools\node22` (نسخهٔ ۲۲٫۲۳٫۵) |
| n8n (نصب‌شده با همان Node) | `%LOCALAPPDATA%\tools\n8n` |
| داده و لاگ n8n | `%LOCALAPPDATA%\tools\n8n-data` |
| اسکریپت کنترل | `tools/n8n/n8n.ps1` |

```powershell
# نصب (یک‌بار) — Node 22 نخست، سپس n8n در پیشوند جدا
# نسخهٔ ۲۲ را از nodejs.org بگیرید و در %LOCALAPPDATA%\tools\node22 باز کنید، سپس:
$env:npm_config_prefix = "$env:LOCALAPPDATA\tools\n8n"
& "$env:LOCALAPPDATA\tools\node22\npm.cmd" install -g n8n

# اجرا
powershell -NoProfile -ExecutionPolicy Bypass -File tools\n8n\n8n.ps1 -Start
powershell -NoProfile -ExecutionPolicy Bypass -File tools\n8n\n8n.ps1 -Status
powershell -NoProfile -ExecutionPolicy Bypass -File tools\n8n\n8n.ps1 -Stop
```

رابط وب: <http://127.0.0.1:5678/> — بار اول حساب مالک ساخته می‌شود.

## ۲) تنظیم دو طرف

**سمت سرور** (`server/.env`):

```env
HRMS_WEBHOOK_URL=http://127.0.0.1:5678/webhook/hrms-events
HRMS_WEBHOOK_SECRET=<کلید تصادفی بلند>
HRMS_WEBHOOK_INTERVAL=15s
```

**سمت n8n** (`integrations/n8n/.env`، از `.env.example` کپی کنید): همان
`HRMS_WEBHOOK_SECRET`، به‌همراه `HRMS_API_BASE` و کاربر/رمز سرویس
(`HRMS_API_USER`/`HRMS_API_PASS`). توکن دسترسی ۳۰ دقیقه اعتبار دارد، پس گردش‌کار در هر
اجرا با همین حساب وارد می‌شود و توکن همان اجرا را به کار می‌برد — هیچ توکن ثابتی ذخیره
نمی‌شود و حساب سرویس باید کمترین دسترسی را داشته باشد.

برای ساخت کلید تصادفی:

```powershell
$b = New-Object byte[] 32
[Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b)
[Convert]::ToBase64String($b)
```

## ۳) ورود گردش‌کار

```powershell
$n8n = "$env:LOCALAPPDATA\tools\n8n\n8n.cmd"
$env:N8N_USER_FOLDER = "$env:LOCALAPPDATA\tools\n8n-data"
$env:N8N_ENCRYPTION_KEY = (Get-Content "$env:N8N_USER_FOLDER\encryption.key" -Raw).Trim()
& $n8n import:workflow --input=integrations\n8n\hrms-events.workflow.json
& $n8n publish:workflow --id=hrms-events-1     # n8n نسخهٔ ۲: به‌جای update:workflow --all
& $n8n list:workflow
```

> `N8N_ENCRYPTION_KEY` در هر فرمان CLI هم لازم است؛ اگر با کلید دیگری اجرا شود، n8n با
> خطای «Mismatching encryption keys» بالا نمی‌آید.

پس از فعال‌سازی، نشانی دریافت:

```
http://127.0.0.1:5678/webhook/hrms-events
```

## ۴) گردش‌کار چه می‌کند

1. **Webhook** — بدنه را «خام» نگه می‌دارد (امضا روی بایت‌های خام است، نه JSON بازسازی‌شده).
2. **بازبینی امضا** (نود Code) — HMAC-SHA256 روی `timestamp.body` و رد کردن مُهر زمانی
   قدیمی‌تر از ۵ دقیقه (ضدِ بازپخش).
3. **ورود سرویس** — با `HRMS_API_USER`/`HRMS_API_PASS` توکن تازه می‌گیرد (توکن ۳۰ دقیقه‌ای است).
4. **رخداد کارمند؟** (IF) — شاخه‌بندی بر اساس نام رخداد.
5. **خواندن پرونده از API** — جزئیات را از خود سامانه می‌خواند (نه از بدنهٔ رخداد)، با
   سرآیند `X-HRMS-Integration-Origin: n8n` تا رخداد تازه‌ای ساخته نشود (ضدِ حلقه).
6. **آمادهٔ اطلاع‌رسانی** — فیلدهای `employeeName`، `department`، `eventName`.
7. **جای نود پیامک/ایمیل** — این‌جا نود دلخواه (SMS، SMTP، بله/ایتا/تلگرام) را بگذارید.

> در n8n نسخهٔ ۲ دسترسی نود Code به `$env` پیش‌فرض بسته است و گردش‌کار با خطای
> «access to env vars denied» شکست می‌خورد؛ اسکریپت `tools\n8n\n8n.ps1` با
> `N8N_BLOCK_ENV_ACCESS_IN_NODE=false` اجرا می‌شود و crypto را هم آزاد می‌کند.

## ۵) قرارداد رخداد

سرآیندها:

| سرآیند | معنا |
|---|---|
| `X-HRMS-Event` | نام رخداد، مثل `employee.updated` |
| `X-HRMS-Event-Id` | شناسهٔ یگانه (برای حذف تکرار) |
| `X-HRMS-Timestamp` | ثانیهٔ یونیکس؛ در امضا هم می‌آید |
| `X-HRMS-Signature` | `sha256=<hex>` = HMAC(secret, `timestamp.body`) |
| `X-HRMS-Attempt` | شمارهٔ تلاش |

بدنه:

```json
{
  "id": "…", "name": "employee.updated", "occurredAt": "2026-11-18T…Z",
  "actor": "hr.manager", "role": "hr_manager", "module": "employee",
  "method": "PATCH", "path": "/api/v1/employee/employees/EMP-011",
  "status": 200, "subject": "EMP-011",
  "payload": {"method": "PATCH", "path": "…", "durationMs": 12},
  "attempt": 1, "sentAt": "…"
}
```

نام‌گذاری رخداد ثابت است: `<ماژول>.<کنش>` با کنش‌های `created`، `updated`، `deleted` و
`action` (برای مسیرهای `.../action` یعنی تأیید/رد گردش‌کار).

## ۶) پایش

مدیر سیستم می‌تواند وضعیت صندوق را ببیند و ردیف ناموفق را به صف برگرداند:

```powershell
curl -H "Authorization: Bearer <توکن مدیر>" http://127.0.0.1:8099/api/v1/events/config
curl -H "Authorization: Bearer <توکن مدیر>" "http://127.0.0.1:8099/api/v1/events/outbox?status=failed"
curl -X POST -H "Authorization: Bearer <توکن مدیر>" http://127.0.0.1:8099/api/v1/events/outbox/12/retry
```