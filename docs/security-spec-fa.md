# اسپک امنیتی پیش از استقرار — سامانهٔ منابع انسانی

> نسخهٔ ۱.۰ — تاریخ تدوین: ۱۴۰۵/۰۸/۲۸
> دامنه: سرور Go (`server/`)، رابط وب (`design/prototype/`)، پایگاه‌داده PostgreSQL و
> اسکریپت‌های استقرار ویندوزی.
> روش کار: **spec-driven** — هر نیازمند با شناسهٔ `REQ-xx` نوشته می‌شود، پیاده‌سازی به
> همان شناسه ارجاع می‌دهد، و آزمونی هم‌نام همان شناسه پذیرش را می‌سنجد. دروازهٔ
> `node tools/check/security-spec.mjs` (و `make verify`) ردیابی را اجباری می‌کند: نیازمندی
> بدون آزمون یا آزمون بدون نیازمندی، شکست است.
> سند عملیاتی همراه: `docs/security-hardening-fa.md` (بندهای REQ-13 و REQ-14).

---

## ۱. مدل تهدید (خلاصه)

دارایی‌ها: رمز و نشست کاربران، دادهٔ پرسنلی و حقوقی (شناسنامه، حکم، فیش، کارت بانکی)،
لاگ حسابرسی، و دسترس‌پذیری سامانه در ساعات اداری.

| # | تهدید | مسیر حمله | پاسخ این اسپک |
| --- | --- | --- | --- |
| T1 | سرقت پایگاه‌داده و شکستن رمزها | پشتیبان لو رفته یا SQLi | درهم‌سازی حافظه‌سخت (REQ-01)، ارتقای خودکار (REQ-02)، حداقل دسترسی DB (REQ-13) |
| T2 | رمزشکنی آنلاین (fcredential stuffing) | تلاش پی‌درپی ورود | قفل حساب (REQ-05)، محدودیت نرخ (REQ-06) |
| T3 | شمارش کاربران | تفاوت زمان/پیام پاسخ | زمان یکسان و پیام یکسان (REQ-04) |
| T4 | جعل یا بازپخش توکن | امضای ضعیف، alg=none، توکن سامانهٔ دیگر | سخت‌سازی JWT (REQ-08) |
| T5 | XSS و Clickjacking در رابط | دادهٔ سرور در `innerHTML` | CSP و سرآیندها (REQ-07، REQ-11) |
| T6 | DoS لایهٔ ۷ و slowloris | سیل درخواست، بدنهٔ بزرگ، سرآیند بی‌پایان | سقف نرخ/بدنه/سرآیند و timeouts (REQ-06، REQ-09، REQ-14) |
| T7 | نشت فایل از مسیر ایستا | `../` و فایل پنهان | رد مسیر ناامن (REQ-10) |
| T8 | پنهان‌ماندن حمله | نبود رد پا | حسابرسی ورود و مسدودسازی (REQ-12) |

**خارج از دامنهٔ این نسخه (آگاهانه):** تست نفوذ شبکه و زیرساخت (کار بستهٔ
`.claude/skills/hrms-pentest-package`)، رمزنگاری داده در حالت سکون (وابسته به انتخاب
دیسک/TDE سازمان)، و MFA.

---

## ۲. نیازمندی‌ها

ستون «تأیید» سه شکل دارد: `go:TestX` آزمون Go، `tool:file` ابزار دروازه، و
`doc:file#کلیدواژه` الزام محتوایی سند استقرار.

| شناسه | نیازمندی | پیاده‌سازی | تأیید |
| --- | --- | --- | --- |
| REQ-01 | رمزها فقط با Argon2id (۱۹ مگابایت، ۲ گذر، ۱ رشته) و نمک تصادفی ۱۶ بایتی درهم شوند؛ قالب ذخیره نسخه‌دار باشد و رمز خام هرگز ذخیره یا لاگ نشود. | `server/internal/platform/auth/password.go` | go:TestREQ01Argon2idHashFormat |
| REQ-02 | درهم‌های قدیمی `pbkdf2_sha256` همچنان پذیرفته شوند و در نخستین ورود موفق بدون دخالت کاربر به Argon2id ارتقا یابند. | `auth/password.go`، `auth/service.go` | go:TestREQ02LegacyHashUpgradesOnLogin |
| REQ-03 | سیاست قدرت رمز (حداقل ۸ نویسه، فهرست سیاه رمزهای پرتکرار و پیش‌فرض‌های سامانه، رد الگوی تکرارشونده، سقف ۱۰۲۴ بایت) در مسیر «تعیین رمز توسط راهبر» اجباری شود. | `auth/password.go`، `platform/localstore/users.go` | go:TestREQ03PasswordStrengthPolicy |
| REQ-04 | ورود با نام کاربری ناموجود همان خطا و همان هزینهٔ زمانی رمز نادرست را بدهد (سنجش درهم ساختگی) تا شمارش کاربران ممکن نباشد. | `auth/service.go` | go:TestREQ04UnknownUserTimingEqualized |
| REQ-05 | پس از ۱۰ شکست پی‌درپی، حساب (کلید: نام کاربری) برای ۱۵ دقیقه قفل شود؛ ورود موفق شمارش را صفر کند؛ قفل خودرها باشد. | `platform/websecurity/guard.go` | go:TestREQ05AccountLockout |
| REQ-06 | سقف نرخ برای هر کلاینت: عمومی ۶۰۰/دقیقه و ورود ۲۰/۵‌دقیقه؛ پاسخ ۴۲۹ با `Retry-After` و کد `rate_limited`؛ `X-Forwarded-For` فقط از پروکسی معتمد باور شود؛ در استقرار پیش‌فرض روشن باشد. | `websecurity/limiter.go`، `websecurity/security.go`، `websecurity/clientip.go`، `cmd/hrmsd/main.go` | go:TestREQ06LoginRateLimit، go:TestREQ06DeploymentDefaults |
| REQ-07 | همهٔ پاسخ‌ها (API، ایستا، خطا) سرآیندهای `Content-Security-Policy`، `X-Content-Type-Options: nosniff`، `X-Frame-Options: DENY`، `Referrer-Policy`، `Permissions-Policy`، COOP/CORP و در HTTPS `Strict-Transport-Security` بگیرند؛ پاسخ API `Cache-Control: no-store` باشد. | `websecurity/headers.go`، `websecurity/security.go` | go:TestREQ07SecurityHeaders، go:TestREQ07SecurityHeadersOnStaticAlso |
| REQ-08 | توکن: الگوریتم باید `HS256` باشد (`alg=none` رد)، ادعاهای `iss=hrms` و `aud=hrms-web` بازبینی شوند، توکن بدون `exp` یا با `exp` گذشته رد شود، و کلید کوتاه‌تر از ۳۲ بایت در استقرار پذیرفته نشود (`HRMS_SECRET_STRICT=1` = توقف). | `auth/token.go`، `cmd/hrmsd/main.go` | go:TestREQ08TokenHardening |
| REQ-09 | سقف منابع درخواست: بدنهٔ JSON حداکثر ۱ مگابایت (۴۱۳)، فیلد ناشناخته رد، سرآیند حداکثر ۶۴ کیلوبایت، و timeouts خواندن/نوشتن/بی‌کاری تنظیم. | `platform/httpx/http.go`، `cmd/hrmsd/main.go` | go:TestREQ09RequestBodyLimit، go:TestREQ09ServerBounds |
| REQ-10 | سرو فایل ایستا مسیرهای `..`، کاراکتر تهی و فایل‌های پنهان (`.env`، `.git`) را با ۴۰۴ رد کند. | `platform/webstatic/webstatic.go` | go:TestREQ10StaticPathTraversal |
| REQ-11 | فرانت‌اند: CSP درج تگ `script` درون‌خطی و مبدأ بیرونی را ببندد (`script-src 'self'`)، `frame-ancestors 'none'` و `base-uri 'self'`؛ توکن نشست در `localStorage` نگه داشته نشود؛ شمار نقاط پرخطر (innerHTML/هندلر درون‌خطی) از سقف ثبت‌شده در همین سند بیشتر نشود. | `websecurity/headers.go`، `design/prototype/js/` | go:TestREQ11ContentSecurityPolicy، tool:frontend-security |
| REQ-12 | هر ورود ناموفق، هر مسدودسازی نرخ/قفل و هر ورود موفق در زنجیرهٔ حسابرسی ثبت شود و پاسخ ۴۲۹ همان سرآیندهای امنیتی را داشته باشد. | `bootstrap/bootstrap.go` | go:TestREQ12LoginThrottleIsAudited |
| REQ-13 | کنترل‌های داده در استقرار: TLS اجباری، حساب پایگاه‌داده با حداقل دسترسی، پشتیبان رمزگذاری‌شده و آزمون بازگردانی، ممنوعیت دادهٔ واقعی در محیط آزمون و ممنوعیت ثبت دادهٔ حساس در لاگ. | `docs/security-hardening-fa.md` | doc:docs/security-hardening-fa.md#tls، doc:docs/security-hardening-fa.md#دسترسی-پایگاه-داده، doc:docs/security-hardening-fa.md#پشتیبان، doc:docs/security-hardening-fa.md#داده-حساس |
| REQ-14 | دفاع DDoS چندلایه: محدودیت نرخ برنامه (REQ-06)، سقف اتصال هم‌زمان و نرخ در پروکسی، بستن slowloris، و مسیر واکنش به حمله (fail2ban/CDN/آستانهٔ هشدار). | `docs/security-hardening-fa.md` | doc:docs/security-hardening-fa.md#ddos، doc:docs/security-hardening-fa.md#nginx |

---

## ۳. طرح اجرا (پلن)

| گام | کار | خروجی | وضعیت |
| --- | --- | --- | --- |
| ۱ | تدوین همین اسپک و تعیین معیار پذیرش هر بند | `docs/security-spec-fa.md` | انجام شد |
| ۲ | سخت‌سازی رمز: Argon2id، ارتقای خودکار، سیاست قدرت، زمان یکسان | `auth/password.go`، `auth/service.go` | انجام شد |
| ۳ | سخت‌سازی توکن و کلید | `auth/token.go`، `cmd/hrmsd/main.go` | انجام شد |
| ۴ | بستهٔ `websecurity`: محدودیت نرخ، قفل حساب، سرآیندها، قاعدهٔ پروکسی معتمد | `platform/websecurity/` | انجام شد |
| ۵ | سیم‌کشی در bootstrap و cmd: روشن‌بودن پیش‌فرض، پاسخ ۴۲۹، ثبت حسابرسی | `bootstrap/bootstrap.go`، `cmd/hrmsd/main.go` | انجام شد |
| ۶ | سخت‌سازی سرو فایل و سقف درخواست | `webstatic/webstatic.go`، `httpx/http.go` | انجام شد |
| ۷ | فرانت‌اند: CSP و دروازهٔ ratchet نقاط پرخطر | `websecurity/headers.go`، `tools/check/frontend-security.mjs` | انجام شد (مرحلهٔ ۲ باز) |
| ۸ | سند عملیاتی استقرار | `docs/security-hardening-fa.md` | انجام شد |
| ۹ | دو دروازهٔ spec-driven و افزودن به `make verify` | `tools/check/security-spec.mjs`، `tools/check/frontend-security.mjs` | انجام شد |
| ۱۰ | اجرای دروازه‌ها و آزمون زندهٔ سرور | گزارش پایین | انجام شد |

### ریسک‌های پذیرفته‌شدهٔ این نسخه

1. **`script-src-attr 'unsafe-inline'`**: رابط نمونهٔ فعلی صدها هندلر `onclick` تولیدشده
   در رشته‌های JS دارد؛ بستن کامل آن نیازمند جابه‌جایی به `data-act` است (کار مستقل).
   تا آن زمان، درج تگ `<script>` و منابع بیرونی بسته است ولی درج *صفت* رویداد نه.
2. **رمز نمایشی `demo` و پیش‌فرض `6468`**: برای نصب تازه و آزمون باقی مانده‌اند؛
   `HRMS_STRICT_PASSWORD` هنوز پیاده نشده و کنترل آن در سند استقرار «تغییر فوری رمز پس از
   نصب» است.
3. **قفل حساب می‌تواند به ابزار آزار تبدیل شود** (قفل کردن حساب دیگران با ۱۰ تلاش
   خراب): مهلت کوتاه و خودرها و ثبت در حسابرسی، هزینه را متعادل می‌کند.
4. **حالت درون‌حافظه‌ای** (بدون Postgres) محدودیت نرخ را در همان فرایند نگه می‌دارد؛ در
   استقرار چندنمونه‌ای باید در پروکسی/لایهٔ مشترک اعمال شود (REQ-14).

---

## ۴. سقف نقاط پرخطر فرانت‌اند (ratchet)

ابزار `node tools/check/frontend-security.mjs` شمار نقاط زیر را می‌شمارد و اگر از سقف
این سطر بیشتر شود، شکست می‌دهد. هر بار که بخشی به `data-act` یا `textContent` منتقل شد،
سقف را پایین بیاورید (فقط پایین، نه بالا).

<!-- frontend-baseline: innerhtml=136 handlers=541 -->

- `innerHTML` در `design/prototype/js/*.js`: نوشتن رشتهٔ HTML از دادهٔ سرور.
- `on*=` تولیدشده در رشته‌های HTML: هندلر درون‌خطی (دلیل `script-src-attr`).

---

## ۵. نگاشت نیازمندی به آزمون (اجرای خودکار)

```
node tools/check/security-spec.mjs     # ردیابی REQ ⇄ آزمون/سند/ابزار
node tools/check/frontend-security.mjs # سقف نقاط پرخطر فرانت‌اند
cd server && go test ./internal/platform/... ./internal/bootstrap/ ./cmd/hrmsd/
```

دروازه شکست می‌خورد اگر: نیازمندی بدون تأیید باشد، آزمون `TestREQxx...` بدون بند متناظر
باشد، یا سند عملیاتی کلیدواژهٔ لازم را نداشته باشد.

---

## ۶. گزارش تأیید (اجرای واقعی)

### ۶.۱ دروازه‌های مخزن

| دروازه | فرمان | نتیجه |
| --- | --- | --- |
| انحراف قرارداد | `node tools/gen/gen.mjs --check` | ✅ ۱۳ خروجی هم‌خوان (۱۰ اداره، ۳۱ ماژول، ۱۳۲ صفحه، ۱۵ شرکت) |
| مرز معماری و قرارداد API | `node tools/check/verify.mjs` | ✅ همهٔ دروازه‌ها سبز |
| قالب اسکیل‌ها | `node tools/skills/check-skills.mjs` | ✅ ۸ اسکیل |
| ردیابی اسپک | `node tools/check/security-spec.mjs` | ✅ ۱۴ بند، ۱۵ آزمون `TestREQ*` |
| سقف فرانت‌اند | `node tools/check/frontend-security.mjs` | ✅ `innerHTML=136`، هندلر درون‌خطی `541` (زیر سقف) |
| آزمون‌های Go (کامل) | `cd server && go test -count=1 ./...` | ✅ همهٔ بسته‌ها سبز |
| آزمون بسته‌های دست‌خورده | `go test ./internal/platform/{auth,websecurity,webstatic,httpx} ./cmd/hrmsd ./internal/bootstrap` | ✅ auth 6.8s · websecurity 3.4s · webstatic 5.4s · httpx 4.5s · hrmsd 2.5s · bootstrap 14.7s |

> `go test -race` روی این ماشین اجرا نشد (بدون cgo/gcc)؛ همان گام در CI لینوکسی با
> `go test -race -count=1 ./...` اجرا می‌شود.

### ۶.۲ آزمون زنده روی سرور واقعی (باینری ساخته‌شده، انبار درون‌حافظه‌ای)

| سنجش | انتظار | نتیجهٔ واقعی |
| --- | --- | --- |
| `GET /readyz` | ۲۰۰ | ۲۰۰ |
| سرآیندهای امنیتی روی `/` | CSP و ۵ سرآیند دیگر | همه حاضر (CSP با `script-src 'self'` و `frame-ancestors 'none'`) |
| `GET /.env` | ۴۰۴ (فایل پنهان) | ۴۰۴ |
| ورود درست (`admin`/`demo`) و `GET /api/v1/auth/me` با توکن | ۲۰۰ | ۲۰۰ (۳۱ ماژول) |
| سقف ورود `HRMS_LOGIN_RATE=3/1m` | ۴۲۹ در تلاش چهارم با `Retry-After` | ۴۲۹ با `Retry-After: 59` |
| قفل حساب `HRMS_LOGIN_LOCKOUT_AFTER=2` | ۴۲۹ در تلاش دوم با مهلت ۱۵ دقیقه | ۴۲۹ با `Retry-After: 900` |
| توکن دست‌کاری‌شده | ۴۰۱ | ۴۰۱ |
| لاگ راه‌اندازی | اعلام روشن‌بودن لایهٔ امنیتی و هشدار سقف نامتعارف | `لایهٔ امنیتی وب روشن است generalRate=600 loginRate=100 lockoutAfter=2 lockoutFor=15m0s` + هشدار «سقف شکست ورود کمتر از ۳» |

### ۶.۳ کارهای باز (آگاهانه، خارج از این نسخه)

1. جابه‌جایی هندلرهای درون‌خطی به `data-act` و حذف `script-src-attr 'unsafe-inline'`
   (سقف ratchet را پس از هر دسته پایین بیاورید).
2. فرار HTML در `innerHTML`های `live_*.js:live_records.js` (پربازدیدترین نقطه).
3. پیاده‌سازی `HRMS_STRICT_PASSWORD` برای توقف راه‌اندازی با رمز پیش‌فرض `6468`.
4. اعمال محدودیت نرخ در لایهٔ مشترک (Redis/پروکسی) برای استقرار چندنمونه‌ای.