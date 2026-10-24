# راه‌اندازی محیط توسعه

## ۰. وضعیت همین حالا روی این ماشین

| ابزار | وضعیت | پیامد |
|---|---|---|
| Node 26 | ✅ نصب | تولید خروجی‌ها از مانیفست و دروازه‌های بررسی |
| Python 3.14 | ✅ نصب | ابزار مرجع fixtureها |
| **Go 1.27.1** | ✅ نصب محلی (`~/tools/go`، بدون دسترسی مدیر) | سرور کامپایل می‌شود و همهٔ آزمون‌هایش سبز است |
| Rust / cargo | ❌ نصب نیست | هستهٔ بومی ساخته نمی‌شود |
| JDK / Android SDK | ❌ نصب نیست | اپ موبایل روی دستگاه اجرا نمی‌شود |

گام‌هایی که **همین حالا روی این ماشین کار می‌کنند** (رابط وب + API روی یک مبدأ واحد):

```bash
# ۱) خروجی‌ها را از مانیفست‌ها بساز و دروازه‌ها را ببند
npm run gen && npm run verify

# ۲) سرور و رابط وب با هم (Go باید در PATH باشد؛ در غیر این صورت:
#    export PATH="$HOME/tools/go/bin:$PATH")
cd server && go run ./cmd/hrmsd -addr 127.0.0.1:8080 -web ../design/prototype

# ۳) در مرورگر: http://127.0.0.1:8080/   — ورود با حساب‌های نمونه زیر
#    (سرور روی همان مبدأ است، پس ورود واقعی با توکن JWT انجام می‌شود)

# ۴) آزمون سرور (حدود ۲۰ ثانیه)
cd server && go test ./...
```

> متغیرهای محیطی از `server/.env` خوانده می‌شوند (همین حالا با یک کلید امضای تصادفی
> ساخته شده است). نبودن این فایل خطا نیست؛ سرور برای توسعه یک کلید موقت می‌سازد و
> در لاگ هشدار می‌دهد.

## ۱. Go (برای سرور)

نسخهٔ ۱.۲۴ به بالا لازم است، چون سرور از `crypto/pbkdf2` کتابخانهٔ استاندارد استفاده
می‌کند و هیچ وابستگی خارجی ندارد.

**راه اول — بدون دسترسی مدیر (همین روش روی این ماشین استفاده شد):** بستهٔ زیپ رسمی را در
پوشهٔ کاربر باز کنید و `bin` آن را به PATH کاربر بیفزایید:

```bash
mkdir -p "$HOME/tools" && cd "$HOME/tools"
curl -sL -o go.zip https://go.dev/dl/go1.27.1.windows-amd64.zip
unzip -q -o go.zip            # نتیجه: ~/tools/go/bin/go.exe
```

سپس یک‌بار (بدون نیاز به دسترسی مدیر، فقط برای همین کاربر) PATH را اضافه کنید:

```powershell
$p=[Environment]::GetEnvironmentVariable('Path','User')
[Environment]::SetEnvironmentVariable('Path', ($p.TrimEnd(';') + ';' + $HOME + '\tools\go\bin'),'User')
```

**راه دوم — نصب سیستمی:**

```powershell
winget install --id GoLang.Go -e
```

سپس:

```bash
cd server && go build ./... && go test ./...
```

## ۲. Rust (برای هستهٔ کلاینت)

```powershell
winget install --id Rustlang.Rustup -e
rustup default stable-x86_64-pc-windows-gnu
rustup component add rustfmt clippy
rustup target add aarch64-linux-android x86_64-linux-android
cargo install cargo-ndk
```

> روی ویندوز، ابزار `stable-x86_64-pc-windows-gnu` همراه بخش `rust-mingw` نصب می‌شود و
> پیونددهندهٔ لازم را می‌آورد؛ نیازی به نصب Visual Studio Build Tools نیست.

سپس:

```bash
cd core && cargo test --workspace --all-features
cargo run -p hrms-conformance     # دروازهٔ تطبیق قواعد
```

## ۳. JDK و Android SDK (برای اجرای اپ موبایل)

```powershell
winget install --id Microsoft.OpenJDK.17 -e
winget install --id Google.AndroidStudio -e
```

پس از نصب Android Studio، در `SDK Manager` این‌ها را نصب کنید: `Android SDK Platform 35`,
`Android SDK Build-Tools`, `NDK (Side by side) 27`, `Android SDK Command-line Tools`.

متغیرهای محیطی:

```
ANDROID_HOME = C:\Users\<user>\AppData\Local\Android\Sdk
ANDROID_NDK_HOME = %ANDROID_HOME%\ndk\27.0.12077973
PATH += %ANDROID_HOME%\platform-tools;%ANDROID_HOME%\cmdline-tools\latest\bin
```

## ۴. کلاینت موبایل

```bash
cd apps/mobile
npm install
npm run typecheck     # بررسی نوع‌ها
npm test              # آزمون‌های منو، راست‌چین و کلاینت API
```

## ۵. ساخت کتابخانهٔ بومی هسته برای اندروید

پیش از اجرای اپ روی دستگاه، کتابخانهٔ هسته باید برای معماری‌های اندروید ساخته شود:

```bash
cd core
cargo ndk -t arm64-v8a -t x86_64 -o ../apps/mobile/android/app/src/main/jniLibs build -p hrms-ffi --release
```

یا با دستور آماده: `make android-libs`

## ۶. اجرای سرور و رابط وب روی یک مبدأ واحد

```bash
cd server
go run ./cmd/hrmsd -addr 127.0.0.1:8080 -web ../design/prototype
```

پنج پرچم مهم:

| پرچم | پیش‌فرض | کار |
|---|---|---|
| `-web` | `../design/prototype` | پوشهٔ رابط وب که روی `/` سرو می‌شود (`no-store` برای دیدن فوری تغییرات در توسعه) |
| `-env` | `.env` | فایل متغیرهای محیطی؛ متغیری که در محیط تعریف شده باشد بازنویسی نمی‌شود |
| `-org` | خالی (پیش‌فرض `organizations.json`) | شرکت فعال (استقرار پارامتریک)؛ معادل متغیر `HRMS_ORG` |
| `-modules` | خالی (از شرکت فعال) | فهرست صریح ماژول‌های فعال؛ بر `-org` مقدم است (برای آزمون و اشکال‌زدایی) |
| `-debug` | `false` | سطح لاگ تفصیلی |

پس از اجرا:

- رابط وب: `http://127.0.0.1:8080/`
- بررسی سلامت: `curl http://127.0.0.1:8080/healthz` (شرکت فعال و تعداد ماژول‌هایش را هم برمی‌گرداند)
- منوی نقش‌محور (حوزه > اداره > زیرواحد > ماژول): `GET /api/v1/auth/me`
- فهرست شرکت‌های قابل شبیه‌سازی و پارامترهای هرکدام: `GET /api/v1/orgs`

## ۶-۱. استقرار پارامتریک برای شرکت دیگر

همین باینری برای هر شرکت دیگری با عوض کردن یک پرچم بالا می‌آید. ماژول‌های غیرفعال همان
شرکت **ثبت نمی‌شوند** (مسیرشان وجود ندارد) و پارامترهای حقوقی-مالی‌اش جای پارامترهای
شرکت پیش‌فرض می‌نشیند:

```bash
# استقرار مرجع: پتروشیمی بندر امام (هلدینگ خلیج فارس) — همهٔ ۲۶ ماژول
cd server && go run ./cmd/hrmsd -addr 127.0.0.1:8080

# همان سامانه برای یک شرکت پیمانکاری نمونه — فقط ۱۴ ماژول
cd server && go run ./cmd/hrmsd -addr 127.0.0.1:8090 -org contractor_services

# فهرست شناسه‌های مجاز شرکت‌ها
curl -s http://127.0.0.1:8090/healthz   # {"org":"contractor_services","modules":11,…}
```

تعریف شرکت تازه یک ورودی در `contracts/organizations.json` است و بعد از آن
`npm run gen` (کلاینت + نمونهٔ تحت وب + سرور از همان فایل تولید می‌شوند). در نمونهٔ تحت وب
همین کار با انتخابگر شرکت در نوار بالا انجام می‌شود — بدون راه‌اندازی مجدد، برای نمایش
و آموزش.

نمونهٔ ورود:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"hr.manager","password":"demo"}'
```

برای فعال‌کردن فقط بخشی از ماژول‌ها (پرچم ویژگی در سطح استقرار):

```bash
go run ./cmd/hrmsd -modules employee,dashboard,settings
```

در استقرار واقعی، کلید امضای توکن را از محیط تزریق کنید. اولویت با محیط است؛ فایل
`server/.env` فقط پرکنندهٔ جاهای خالی است:

```bash
export HRMS_JWT_SECRET="<یک رشتهٔ تصادفی بلند>"
```

## ۷. دروازه‌های پیش از کامیت

```bash
npm run gen          # بازتولید خروجی‌ها از مانیفست‌ها
make check           # انحراف قرارداد + مرز ماژول + آزمون‌های Go و Rust + تطبیق قواعد
```

## ۸. اجرای خروجی طراحی (بدون هیچ وابستگی)

اگر سرور را بالا نیاورید، بستهٔ طراحی همچنان مستقل کار می‌کند: `design/prototype/index.html`
را با هر وب‌سرور استاتیک (یا مستقیم در مرورگر) باز کنید. در این حالت ورود به شکل
«حالت نمایشی» انجام می‌شود و در پای منو برچسب �حالت نمایشی — بدون سرور» دیده می‌شود؛
با بالا آمدن سرور، همان صفحه برچسب «متصل به سرور محلی» می‌گیرد و ورود واقعی با JWT
انجام می‌شود.

منوی این نمونه از فایل تولیدشدهٔ `design/prototype/js/menu.generated.js` می‌آید که خودش
از `contracts/departments.json` و مانیفست ماژول‌ها ساخته می‌شود؛ پس ساختار منو در نمونهٔ
تحت وب، اپ React Native و سرور همیشه یکی است.

## ۹. بازتولید fixtureهای طلایی

fixtureهای `core/testdata/conformance/` مرجع تطبیق Rust و Go هستند. برای بازتولید
دستی پس از تغییر آگاهانهٔ قواعد:

```bash
cd core && cargo run -p hrms-conformance -- --write
```

ابزار مرجع پایتونی (`tools/check/ref_fixtures.py`) فقط برای صحت‌سنجی مستقل هنگام
نوشتن اولیهٔ fixtureها استفاده شد و بخشی از بیلد نیست.
