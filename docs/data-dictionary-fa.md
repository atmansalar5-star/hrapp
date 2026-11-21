# فرهنگ داده و نمودار داده (Data Dictionary & ERD)

> نسخهٔ ۱.۰ — ۱۴۰۵/۰۸/۳۰
> مخاطب: طراح داده، توسعه‌دهنده و راهبر استقرار.
> روش: **spec-driven** — بندهای `DD-xx` پایین همین سند، با دروازهٔ
> `node tools/check/dd-spec.mjs` (و `make verify`) ردیابی می‌شوند.

---

## ۱. چرا و چه چیزی

ساختار دادهٔ سامانه در ۱۷ مهاجرت و ۸ طرح (`hr_*`) تعریف شده است. برای «فهم طراحی داده»
سه چیز لازم است: **درخت داده** (طرح ← جدول ← ستون)، **نمودار انتیتی-رلیشن** (روابط
خارجی) و **نگاشت به ماژول‌ها** (کدام صفحه کدام جدول را می‌خواند).

هر سه، از **کاتالوگ زندهٔ PostgreSQL** ساخته می‌شوند، نه از فایل دستی. یعنی هر بار
مهاجرت یا جدولی عوض شود، پاسخ تازه خودش ساختار تازه را نشان می‌دهد؛ «بروزرسانی» یعنی
همان درخواست تازه — نه بازتولید فایل.

---

## ۲. API فرهنگ داده

```http
GET /api/v1/platform/data-dictionary?format=<json|tree|mermaid|md|csv>
Authorization: Bearer <access token>   # فقط نقش admin
```

| قالب | نوع پاسخ | کاربرد |
| --- | --- | --- |
| `json` (پیش‌فرض) | `application/json` | ساختار کامل ماشین‌خوان: طرح‌ها، جدول‌ها، ستون‌ها، کلیدها، روابط، نگاشت ماژول‌ها |
| `tree` | `text/plain` | درخت خوانا برای مرور سریع در ترمینال |
| `mermaid` | `text/plain` | نمودار ERD به زبان Mermaid (چسباندنی در VS Code، GitLab، mermaid.live) |
| `md` | `text/markdown` | سند کامل فرهنگ داده به فارسی + بلوک ERD |
| `csv` | `text/csv` | فهرست ستون‌محور مسطح برای Excel (با BOM) |

**دسترسی:** فقط مدیر سیستم. ساختار داده نقشهٔ راه است (نام جدول‌های حساس، ستون‌ها،
روابط) و برای نقش‌های عملیاتی لازم نیست؛ نقش غیرمدیر `403` می‌گیرد و درخواست بدون نشست
`401`. روی استقراری که پایگاه‌داده ندارد، پاسخ `503` با پیام روشن است.

نمونه:

```powershell
$tok = (Invoke-RestMethod http://127.0.0.1:8099/api/v1/auth/login -Method Post `
        -ContentType 'application/json' -InFile login.json).accessToken
# درخت داده
curl.exe -s "http://127.0.0.1:8099/api/v1/platform/data-dictionary?format=tree" -H "Authorization: Bearer $tok"
# نمودار ERD
curl.exe -s "http://127.0.0.1:8099/api/v1/platform/data-dictionary?format=mermaid" -H "Authorization: Bearer $tok" > erd.mmd
# سند فرهنگ داده
curl.exe -s "http://127.0.0.1:8099/api/v1/platform/data-dictionary?format=md"    -H "Authorization: Bearer $tok" > dict.md
```

---

## ۳. فایل‌های تولیدشده (نسخه‌دار) و بازتولید

| فایل | محتوا |
| --- | --- |
| `docs/generated/data-dictionary-fa.md` | سند فرهنگ داده (فارسی) + ERD جاسازی‌شده |
| `docs/generated/erd-fa.mmd` | نمودار ERD خالص Mermaid |
| `docs/generated/data-dictionary-fa.csv` | فهرست ستون‌ها برای Excel |

```bash
make datadict          # بازتولید هر سه فایل از پایگاه‌دادهٔ فعال
make datadict-check    # بررسی هم‌خوانی فایل‌ها با کاتالوگ زنده (کد ۱ اگر کهنه باشند)
```

معادل با ابزار عملیاتی (بدون نیاز به سرور بالا):

```bash
cd server
go run ./cmd/hrops -env .env -dict -format mermaid -out D:\hrapp\.dsh-tmp\dd\erd-fa.mmd
go run ./cmd/hrops -env .env -dict -format md      -out D:\hrapp\.dsh-tmp\dd\data-dictionary-fa.md -check
```

**قاعدهٔ مهم:** این فایل‌ها «پایدار» تولید می‌شوند — بدون مُهر زمان و بدون شمار سطرها —
تا تغییر *داده* آن‌ها را کهنه نکند؛ فقط تغییر *ساختار* (مهاجرت) باعث تفاوت می‌شود.
خروجی زندهٔ API همان قالب کامل (با زمان و شمار سطر) را می‌دهد.

> نکتهٔ محیط ویندوز سندباکس: فرایندهای فرزند (مثل `hrops`) نمی‌توانند مستقیم در
> `docs/generated` بنویسند؛ ابتدا در یک پوشهٔ موقت بنویسید و سپس با ابزار ویرایش فایل
> منتقل کنید. در اجرای بدون سندباکس، `-out docs/generated/...` مستقیم کار می‌کند.

---

## ۴. فیلدهای سطح فرم (مکمل فرهنگ داده)

فرهنگ داده ستون‌های **پایگاه‌داده** را نشان می‌دهد. فیلدهای **فرم/API** هر ماژول
(برچسب فارسی، اجباری‌بودن، نوع ورودی) از مسیر خودِ ماژول خوانده می‌شود:

```http
GET /api/v1/<module>/schema        # انواع رکورد و فیلدهای هر نوع
```

این دو کنار هم تصویر کامل می‌دهند: `hr_records.record` ستون‌های فیزیکی را نگه می‌دارد و
`/schema` می‌گوید هر ماژول چه فیلدهایی با چه عنوان فارسی روی آن می‌نویسد.

**نکتهٔ مهم طراحی (چرا برخی ماژول‌ها در ستون «جدول‌ها» صفر نشان می‌دهند):** ماژول‌های
گردش‌کاری (archive, attendance, payroll, welfare, medical, settings و …) جدول اختصاصی
ندارند و داده‌شان در طرح **مشترک** `hr_records` ذخیره می‌شود:

```
hr_records.record         (module, kind, code, title_fa, payload jsonb, …)
hr_records.record_event   (تاریخچهٔ تغییر هر رکورد، با FK به record)
hr_records.seq            (شمارندهٔ شناسهٔ هر module/kind)
```

تفکیک دادهٔ این ماژول‌ها با ستون `module` و نوع با `kind` انجام می‌شود و فیلدهای
کسب‌وکاری هر نوع داخل `payload` (JSONB) می‌نشیند؛ همان چیزی که `/api/v1/<module>/schema`
عنوان فارسی‌اش را می‌دهد. جدول‌های اختصاصی فقط برای ماژول‌هایی است که پرس‌وجوی رابطه‌ای و
قید خارجی واقعی لازم دارند (`hr_employee`, `hr_contracts`, `hr_personnel`,
`hr_salary_admin`, `hr_settings`).

---

## ۵. بندهای نیازمندی

| شناسه | نیازمندی | پیاده‌سازی | تأیید |
| --- | --- | --- | --- |
| DD-01 | فرهنگ داده از کاتالوگ زنده ساخته شود: درخت طرح ← جدول ← ستون با نوع، معنای فارسی نوع، تهی‌پذیری، پیش‌فرض، کلید اصلی، کلید خارجی و شمار سطر؛ و قالب JSON ماشین‌خوان. | `server/internal/platform/datadict` | go:TestDD01TreeAndJSON، go:TestDD01TypeLabels، doc:docs/data-dictionary-fa.md#درخت |
| DD-02 | نمودار انتیتی-رلیشن از روابط خارجی ساخته شود (شامل قاعدهٔ حذف و اجباری‌بودن) و به زبان Mermaid با شناسه‌های معتبر ارائه شود. | `datadict/render.go` | go:TestDD02MermaidERD |
| DD-03 | سند Markdown فارسی (با بلوک ERD و جدول روابط) و خروجی CSV مسطح با BOM برای Excel تولید شود. | `datadict/render.go` | go:TestDD03MarkdownAndCSV |
| DD-04 | API با کنترل دسترسی مدیر سیستم، پشتیبانی از پنج قالب و پیام روشن روی استقرار بدون پایگاه‌داده ارائه شود. | `bootstrap/bootstrap.go` | go:TestDD04DictionaryAPIAdminOnly |
| DD-05 | نگاشت ماژول‌ها به دادهٔ ذخیره‌شده (نام فارسی، پیشوند API، طرح، موجودیت‌ها، جدول‌ها) در پاسخ باشد. | `datadict/datadict.go` | go:TestDD05ModuleMapping |
| DD-06 | فایل‌های تولیدشدهٔ فرهنگ داده با کاتالوگ زنده هم‌خوان بمانند و امکان بازتولید یک‌فرمانی و بررسی کهنگی وجود داشته باشد. | `cmd/hrops`، `Makefile` | file:docs/generated/data-dictionary-fa.md، file:docs/generated/erd-fa.mmd، doc:docs/data-dictionary-fa.md#بازتولید |

---

## ۶. حذف دادهٔ نمایشی پیش از دیپلوی

از این نسخه تا روز دیپلوی، `HRMS_SEED_DEMO=1` است تا همهٔ ماژول‌ها با دادهٔ نمایشی
کارکردی باشند. روز دیپلوی (به‌گفتهٔ کاربر انجام می‌شود):

```powershell
# ۱) بذرکاری را خاموش کن (server\.env)
#    HRMS_SEED_DEMO=0
#    و برای اطمینان: HRMS_DEV_ACCOUNTS=0

# ۲) سرویس را بخوابان
Stop-ScheduledTask -TaskName 'HRMS Server'

# ۳) دادهٔ نمایشی را پاک کن (ساختار و مهاجرت‌ها دست‌نخورده می‌مانند)
cd D:\hrapp\server
go run .\cmd\hrdump -db $env:HRMS_DATABASE_URL -truncate     # یا: hrdump با DSN از .env

# ۴) پشتیبان پیش از حذف (توصیه‌شده)
go run .\cmd\hrdump -db $env:HRMS_DATABASE_URL -out D:\hrappbk\data\pre-deploy-demo.sql

# ۵) سرویس را بالا بیاور
Start-ScheduledTask -TaskName 'HRMS Server'
Get-Content .\server.log -Tail 5      # باید بنویسد: «مهاجرت‌ها بررسی شد found=17 applied=0»
```

بعد از این گام‌ها، سامانه با **کاربران واقعی** (`data/users.json`) و **بدون هیچ رکورد
نمایشی** بالا می‌آید؛ دادهٔ واقعی از رابط یا API وارد می‌شود.

---

## ۷. نگهداری

| رویداد | کار |
| --- | --- |
| مهاجرت تازه (جدول/ستون تازه) | `make datadict` و کامیت فایل‌های `docs/generated/data-dictionary-*` و `erd-fa.mmd` |
| بررسی کهنگی در بازبینی | `make datadict-check` |
| نگاه سریع انسانی | `curl ...?format=tree` |
| رسم نمودار | فایل `docs/generated/erd-fa.mmd` را در VS Code (افزونهٔ Mermaid) یا mermaid.live بچسبانید |
| تغییر ساختار بدون مهاجرت | ساختار فقط از مهاجرت‌ها می‌آید؛ تغییر دستی در پایگاه‌داده مجاز نیست (فرهنگ داده آن را نشان می‌دهد ولی دروازهٔ انحراف قرارداد آن را نمی‌پسندد) |