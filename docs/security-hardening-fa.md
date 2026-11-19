# سخت‌سازی استقرار — چک‌لیست عملیاتی امنیت

> همراه بندهای REQ-13 و REQ-14 سند `docs/security-spec-fa.md`.
> مخاطب: کسی که سامانه را روی سرور سازمان بالا می‌آورد.
> قاعدهٔ کلی: کنترل‌های داخل برنامه (درهم‌سازی رمز، محدودیت نرخ، سرآیندها) **کامل‌کنندهٔ**
> این چک‌لیست‌اند، نه جایگزین آن.

---

## ۱. پیش از استقرار — سه چیز که هرگز نباید جا بیفتد

1. **کلید امضا**: `HRMS_JWT_SECRET` را با مقدار تصادفی ۳۲ بایتی بگذارید (نه متن ساده).
   با `HRMS_SECRET_STRICT=1` سرور با کلید کوتاه بالا نمی‌آید.
2. **رمز پیش‌فرض**: رمز `6468` حساب `admin` در نصب تازهٔ مستقل را همان روز عوض کنید.
3. **TLS**: بدون TLS، رمز و توکن روی شبکه خوانا هستند؛ هیچ کنترل نرم‌افزاری این را جبران نمی‌کند.

---

## ۲. TLS و HTTPS

- پروکسی (nginx/IIS) درگاه ۴۴۳ را با گواهی معتبر سازمان سرو کند و درگاه ۸۰ فقط
  `301` به `https` بدهد.
- سرور Go پشت پروکسی روی `127.0.0.1:8099` بماند (نه `0.0.0.0`).
- با TLS روشن، `HRMS_HSTS=1` بگذارید تا سرآیند `Strict-Transport-Security` فرستاده شود.
- اگر پروکسی روی میزبان دیگری است، نشانی‌اش را در `HRMS_TRUSTED_PROXIES` بگذارید تا
  نشانی واقعی کلاینت از `X-Forwarded-For` خوانده شود؛ در غیر این صورت محدودیت نرخ همهٔ
  کاربران را یک کلاینت می‌بیند.
- نسخهٔ TLS: ۱.۲ به بالا، الگوریتم‌های ضعیف (RC4، 3DES) خاموش، OCSP stapling روشن.

---

## ۳. دسترسی پایگاه‌داده

- حساب برنامه (`hrms`) فقط `SELECT/INSERT/UPDATE/DELETE` روی schema خودش داشته باشد؛
  `SUPERUSER`، `CREATEDB` و `CREATEROLE` نداشته باشد.
- مهاجرت‌ها با حساب جداگانه اجرا شوند (نه با حساب برنامه).
- `pg_hba.conf` فقط `127.0.0.1/32` را با `scram-sha-256` بپذیرد؛ رمز حساب برنامه از
  فایل `.env` با دسترسی محدود (فقط `SYSTEM` و Administrators) خوانده شود.
- پورت ۵۴۳۲ روی شبکهٔ بیرونی بسته باشد؛ دسترسی مدیریتی فقط از راه تونل/VPN.

---

## ۴. پشتیبان

- پشتیبان روزانه با `hrdump` + پوشهٔ `server/data` (نشان شرکت‌ها، عکس‌ها، فایل‌های تبادل).
- **پشتیبان رمزگذاری‌شده** و در محل جدا از سرور نگهداری شود؛ پشتیبان رمز نشدهٔ دادهٔ
  پرسنلی همان ریسک نشت پایگاه‌داده را دارد.
- آزمون بازگردانی روی پایگاه‌دادهٔ آزمایشی، دست‌کم یک بار در هر نسخه (روش در
  `AGENTS.md` بخش ۵ آمده است)؛ پشتیبانِ آزمون‌نشده را پشتیبان ندانید.
- نگه‌داری: سه بستهٔ آخر؛ حذف قدیمی‌ها با اطمینان از سالم‌بودن نسخهٔ جدید.

---

## ۵. داده حساس

- دادهٔ پرسنلی/حقوقی «محرمانه» است: دسترسی فقط نقش‌محور (همان چیزی که `rbac` اعمال
  می‌کند) و لاگ دسترسی برای گزارش‌های حساس.
- در لاگ‌ها رمز، توکن، شمارهٔ حساب و کد ملی ثبت نشود؛ لاگ حسابرسی سامانه فقط نام کاربر،
  ماژول، اقدام و نتیجه را نگه می‌دارد.
- محیط آزمون و نمایش هرگز دادهٔ واقعی نگیرد؛ بذرکاری نمونه (`HRMS_SEED_DEMO=1`) فقط
  دادهٔ ساختگی می‌سازد.
- فایل‌های خروجی (Excel/PDF فیش و حکم) در پوشهٔ اشتراکی عمومی گذاشته نشوند.

---

## ۶. DDoS

دفاع چندلایه؛ هیچ لایه‌ای تنها کافی نیست:

| لایه | کنترل | کجا |
| --- | --- | --- |
| L3/L4 | سقف پهنای باند، بستن پورت‌های بی‌استفاده، SYN cookies | دیوار آتش / CDN |
| L7 — پروکسی | `limit_req`، `limit_conn`، سقف بدنه، بستن درخواست بی‌پایان | nginx |
| L7 — برنامه | سقف نرخ ۶۰۰/دقیقه، سقف ورود ۲۰/۵‌دقیقه، قفل حساب پس از ۱۰ شکست | `websecurity` |
| منابع | `ReadHeaderTimeout=10s`، `ReadTimeout=30s`، `WriteTimeout=60s`، `IdleTimeout=90s`، سقف سرآیند ۶۴KB، سقف بدنه ۱MB | `cmd/hrmsd` + `httpx` |
| واکنش | مسدودسازی IP مهاجم (fail2ban روی لاگ nginx)، آستانهٔ هشدار روی ۴۲۹ و ۴۰۱ | میزبان |

**مسیر واکنش هنگام حمله:** ۱) تأیید با نمودار ۴۲۹/۵۰۳، ۲) بررسی `login-throttled` در
لاگ حسابرسی، ۳) مسدودسازی IP در دیوار آتش/nginx، ۴) در حملهٔ توزیع‌شده، روشن‌کردن
CDN/WAF جلوی سامانه، ۵) بازبینی رمزهای پرتکرار اگر الگوی حمله cultureنامه‌ای بود.
خاموش‌کردن محدودیت نرخ (`HRMS_SECURITY_OFF=1`) **در حمله** بدترین کار است.

### نمونه nginx

```nginx
# جلوگیری از سیل: نرخ عمومی و اتصال هم‌زمان به‌ازای هر IP
limit_req_zone  $binary_remote_addr zone=hrms_general:10m rate=10r/s;
limit_req_zone  $binary_remote_addr zone=hrms_login:10m   rate=1r/s;
limit_conn_zone $binary_remote_addr zone=hrms_conn:10m;

server {
    listen 443 ssl;
    server_name hrms.example.local;

    ssl_certificate     /etc/ssl/hrms/fullchain.pem;
    ssl_certificate_key /etc/ssl/hrms/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;

    client_max_body_size 2m;
    limit_conn hrms_conn 20;

    location /api/v1/auth/login {
        limit_req zone=hrms_login burst=10 nodelay;
        proxy_pass http://127.0.0.1:8099;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Host $host;
    }

    location / {
        limit_req zone=hrms_general burst=40 nodelay;
        proxy_pass http://127.0.0.1:8099;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Host $host;
        proxy_read_timeout 60s;
    }
}

server {
    listen 80;
    server_name hrms.example.local;
    return 301 https://$host$request_uri;
}
```

با این تنظیم، `HRMS_TRUSTED_PROXIES=127.0.0.1` لازم است (اگر nginx روی همان میزبان است،
پیش‌فرض درست است).

---

## ۷. متغیرهای محیطی امنیتی

| متغیر | پیش‌فرض | توضیح |
| --- | --- | --- |
| `HRMS_JWT_SECRET` | تصادفی در هر اجرا | کلید امضای توکن؛ در استقرار حتماً تنظیم شود (≥۳۲ بایت) |
| `HRMS_SECRET_STRICT` | خاموش | `1` = با کلید کوتاه سرور بالا نیاید |
| `HRMS_SECURITY_OFF` | خاموش | `1` = محدودیت نرخ/قفل خاموش (فقط عیب‌یابی کوتاه) |
| `HRMS_RATE_LIMIT` | `600/1m` | سقف درخواست هر کلاینت |
| `HRMS_LOGIN_RATE` | `20/5m` | سقف تلاش ورود هر کلاینت |
| `HRMS_LOGIN_LOCKOUT_AFTER` | `10` | شمار شکست پی‌درپی تا قفل حساب |
| `HRMS_LOGIN_LOCKOUT_FOR` | `15m` | مهلت قفل حساب |
| `HRMS_TRUSTED_PROXIES` | `127.0.0.1,::1` | پروکسی‌های معتمد برای `X-Forwarded-*` |
| `HRMS_HSTS` | خاموش | `1` = ارسال `Strict-Transport-Security` روی HTTPS |

---

## ۸. بازبینی پس از استقرار (یک‌بار، ۱۰ دقیقه)

```powershell
# ۱) سرآیندهای امنیتی روی پاسخ واقعی
curl.exe -sI http://127.0.0.1:8099/ | Select-String -Pattern 'Content-Security|X-Frame|X-Content|Referrer'

# ۲) سقف نرخ ورود (باید در تلاش سوم/چهارم ۴۲۹ بدهد)
1..6 | ForEach-Object {
  curl.exe -s -o NUL -w "%{http_code} " -X POST http://127.0.0.1:8099/api/v1/auth/login `
    -H "Content-Type: application/json" -d '{"username":"admin","password":"wrong"}'
}

# ۳) جهش فایل پنهان (باید ۴۰۴ بدهد)
curl.exe -s -o NUL -w "%{http_code}`n" http://127.0.0.1:8099/.env

# ۴) کلید کوتاه باید در حالت سخت‌گیرانه سرور را بالا نیاورد
$env:HRMS_JWT_SECRET="short"; $env:HRMS_SECRET_STRICT="1"; .\bin\hrmsd.exe -addr 127.0.0.1:8199
```

| نشانه | معنا |
| --- | --- |
| ۴۲۹ در تلاش سوم ورود | محدودیت نرخ فعال است |
| ۴۰۱ برای همه (بدون ۴۲۹) | محدودیت نرخ خاموش است؛ `HRMS_SECURITY_OFF` یا نرخ را بررسی کنید |
| ۲۰۰ برای `/.env` | نسخهٔ قدیمی سرور در حال اجراست؛ باینری تازه بسازید |
| نبود `Content-Security-Policy` | سرور پشت پروکسی قدیمی است یا میان‌افزار سیم‌کشی نشده |