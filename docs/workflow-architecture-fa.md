# معماری ورک‌فلو — طراحی گردش‌کارهای منابع انسانی و بازآرایی لایه‌ها

این سند مکمل [architecture-fa.md](architecture-fa.md) است و مشخص می‌کند کارهای
گردش‌کاری منابع انسانی چگونه مدل، اجبار و نمایش داده می‌شوند. اینجا **منبع حقیقت قاعدهٔ
گردش‌کار** دامنهٔ هر ماژول است، نه رابط کاربری.

## ۱. مسئله

بیشتر کار منابع انسانی گردش‌کاری است: درخواست مرخصی، تخصیص وام، صدور حکم، پروندهٔ
انضباطی، جذب نیرو، مطالبات رفاهی، پروندهٔ بازنشستگی و… . هرکدام چند مرحلهٔ تصمیم دارند و
«چه کسی در هر مرحله تصمیم می‌گیرد» خودش یک قاعدهٔ کسب‌وکار است.

وضعیت پیش از این بازآرایی:

- موتور `recordkit` هر نوع رکورد را با **وضعیت + انتقال + اقدام** مدل می‌کرد، ولی هیچ‌جا
  نمی‌گفت *کدام نقش* در هر مرحله مجاز است؛ اجبار فقط در سطح ماژول بود (نقش یا دسترسی
  `approve` در ماژول را دارد یا نه). یعنی در ماژول حضور، هم مدیر واحد و هم منابع انسانی
  می‌توانستند یک مرخصی را بدون هیچ زنجیره‌ای تأیید کنند.
- هیچ **کارتابل** مشترکی نبود؛ کاربر باید وضعیت‌ها را دستی فیلتر می‌کرد.
- زنجیرهٔ مرحله‌ها در UI بازنویسی می‌شد، نه از دامنه.

## ۲. مدل هدف: گردش‌کار اعلامی

قاعده در سه جای مشخص می‌نشیند و هر لایه فقط کار خودش را می‌کند:

| لایه | مسئولیت | جای کد |
|---|---|---|
| دامنهٔ ماژول | اعلام وضعیت‌ها، انتقال‌های مجاز، اقدام‌ها و **نقش/دامنهٔ هر اقدام** | `internal/modules/<id>/domain` |
| موتور | اجبار قواعد، محاسبهٔ اقدام‌های مجاز، ثبت رخداد، ساخت زنجیرهٔ مرحله‌ها | `internal/platform/recordkit` |
| حمل (HTTP) | احراز هویت، سیاست نقش‌محور ماژول، مسیرهای استاندارد | `internal/platform/{rbac,recordkit}` |
| نمایش | فقط نمایش؛ از شناسهٔ سرور می‌سازد | `design/prototype`, `apps/mobile` |

### ۲-۱. اجبار نقش در هر مرحله (تفکیک وظایف)

هر `recordkit.Action` دو میدان تازه گرفت:

- `Roles []contracts.Role` — نقش‌هایی که در این مرحله حق تصمیم دارند. **خالی = هر نقشی
  که در سطح ماژول اجازهٔ اقدام دارد** (سازگاری با ماژول‌های مهاجرت‌نکرده). مدیر سیستم
  همیشه مجاز است.
- `Scope` — حداقل دامنهٔ دادهٔ لازم: `all|own_unit|own` (رزرو برای «تأیید فقط رکورد واحد
  خودم»).

قالب اعلام: `value|برچسب|style|مقصد|noteRequired|roles|scope`

```go
Actions: recordkit.Actions(
    "endorse|تأیید مدیر واحد|primary|unitApproved|0|line_manager",
    "approve|تأیید نهایی منابع انسانی|primary|approved|0|hr_manager",
)
```

### ۲-۲. زنجیرهٔ مرحله‌ها روی شناسهٔ رکورد

`Kind.Schema()` حالا `workflow` هم می‌دهد: برای هر وضعیتی که از آن اقدامی خارج می‌شود،
یک مرحله با نقش‌ها و اقدام‌هایش. کلاینت از همین، نمودار «چه کسی این‌جاست» را می‌سازد؛ UI
هیچ مرحله‌ای را از خودش نمی‌سازد. `Action.Roles` و `Action.Scope` هم در DTO می‌آید.

### ۲-۳. کارتابل سراسری

- هر ماژول رکوردی هنگام `recordkit.Mount` خودش را به‌عنوان `contracts.WorkflowSource`
  در `modulekit.Ports` ثبت می‌کند (بدون شکستن مرز ماژول).
- سرویس، برای هر رکورد اقدام‌های مجازِ نقش کاربر را می‌سازد و به «کار» تبدیل می‌کند.
- `GET /api/v1/workflow/inbox` همهٔ کارها را جمع می‌کند و تنها کارهای ماژول‌هایی را
  نشان می‌دهد که کاربر در آن‌ها اجازهٔ اقدام دارد (سیاست `rbac`)، پس کارتابل پرنویز
  نمی‌شود.

هر کار می‌گوید *چه اقدامی* از کاربر انتظار می‌رود (`actionLabelFa`)، نه فقط اینکه رکوردی
باز است.

## ۳. طراحی گردش‌کار هر بخش

ستون «مرحله‌ها» زنجیرهٔ تصمیم را نشان می‌دهد؛ نقش داخل پرانتز، مجاز هر مرحله است.
`مدیر واحد` = `line_manager`، `م.ا` = `hr_manager`، `کارشناس` = `hr_spec`،
`کارمند` = `employee`.

### ۳-۱. حضور و غیاب، مرخصی، مأموریت (پیاده‌شده — مرجع)

| نوع | مرحله‌ها | یادداشت |
|---|---|---|
| `leaveRequests` | draft → pending **(مدیر واحد)** → unitApproved **(م.ا)** → approved / rejected | **دومرحله‌ای واقعی:** مدیر واحد شایستگی و جانشینی را تأیید می‌کند، سپس منابع انسانی سوابق و ماندهٔ مرخصی را نهایی می‌کند. هر طرف نمی‌تواند مرحلهٔ دیگری را دور بزند. |
| `missions` | draft **(کارمند/کارشناس)** → pending → approved **(م.ا/مدیر واحد)** → settled **(م.ا)** | تسویهٔ مأموریت فقط منابع انسانی. |
| `overtime` | draft → approved/rejected **(م.ا/مدیر واحد)** | |
| `attendanceRecords` | recorded → confirmed → disputed | تأیید تردد توسط سرپرست/کارشناس. |

### ۳-۲. وام و بازنشستگی (نقش‌ها پیاده‌شده)

| نوع | مرحله‌ها |
|---|---|
| `loans` | requested → **score (کارشناس/م.ا)** → scored → **approve (م.ا)** → approved → **disburse (م.ا)** → repaying → **settle (م.ا)** → settled؛ `reject` فقط **م.ا** |
| `loanInstallments` | pending → **pay (م.ا)** → paid / **overdue (م.ا)** |
| `retirementFiles` | draft → **verify (کارشناس/م.ا)** → verified → **approve (م.ا)** → approved |

### ۳-۳. سایر بخش‌ها — طراحی هدف (در صف مهاجرت)

| ماژول | نوع رکورد | زنجیرهٔ پیشنهادی |
|---|---|---|
| انتصابات/ترفیعات (`career`) | `appointments`, `promotions` | draft **(کارشناس)** → approve **(م.ا)** → issued **(م.ا)** — «صدور» جدا از «تأیید» تا سند صادرشده بازنویسی‌پذیر نباشد |
| کارگزینی حقوق (`payroll`) | `payrollRuns` | draft **(کارشناس)** → calculated → approve **(م.ا)** → locked → paid **(م.ا)**؛ پرداخت تنها پس از قفل |
| `karana` | calculated → approve **(م.ا)** → sent **(م.ا)** → paid | |
| احکام (`salary_admin`) | پیش‌نویس → صدور **(کارشناس)** → ابلاغ **(م.ا)**؛ اصلاح، حکم تازه می‌سازد | جدول سفارشی، نه recordkit |
| قراردادها (`contracts`) | draft **(کارشناس)** → pending_approval → active **(م.ا)** | سرویس سفارشی |
| انضباط کار (`discipline`) | draft → investigating **(کارشناس+مدیر واحد)** → hearing **(کمیته)** → decided **(م.ا)** → archived | مرحلهٔ «کمیته» به نقش جدا نیاز دارد |
| شکایات (`grievances`) | received → investigating **(م.ا)** → resolved/rejected **(م.ا)** → closed |
| جذب (`recruitment`) | `hiringRequests` draft → approve **(م.ا)** → published → closed؛ `candidates` applied → screening/interview/offer/hire **(کارشناس، گام hire با م.ا)** |
| عملکرد (`performance`) | `perfForms` assigned **(کارمند)** → inProgress → submitted → reviewed **(مدیر واحد)**؛ `perfResults` finalize **(م.ا)** |
| آموزش (`training`) | `trainingNeeds` draft → approve **(م.ا)** → planned؛ `courses` planned → open **(کارشناس)** → running → finished |
| رفاه (`welfare`) | `welfareRequests` requested **(کارمند)** → approve/reject **(م.ا)** → delivered **(کارشناس)** |
| مددکاری (`socialwork`) | `aidRequests` requested → approve **(م.ا)** → paid **(م.ا)**؛ `socialWorkCases` open → treating **(مددکار)** → closed |
| خدمات درمانی (`medical`) | `medicalServices` draft → approve **(م.ا)**؛ `periodicExams` scheduled → done → passed/failed/followup **(پزشک)** |
| جابه‌جایی (`staffing_moves`) | `transfers` draft **(کارشناس)** → submit → pending → approve **(م.ا مبدأ و مقصد)** → executed |
| برون‌سپاری (`outsourcing`) | `outsourcePayRuns` draft → submit **(کارشناس)** → approve **(م.ا)** → paid؛ سهم پیمانکار باید پس از تأیید قفل شود |
| طبقه‌بندی مشاغل (`jobclass`) | `jobEvaluations` draft → score **(کمیته)** → approved **(م.ا)** |
| بایگانی (`archive`) | `personnelFiles` incomplete → complete **(کارشناس)** → archived **(م.ا)** |
| ورزش (`sports`) | `sportsPrograms` draft → activate **(م.ا)** → closed |
| مطالعات (`hrstudies`) | `suggestions` submitted **(کارمند)** → reviewing → accepted/rejected **(م.ا)** |
| آمار (`statistics`) | `dataApprovals` pending → approve/reject **(م.ا)** |
| تنظیمات (`settings`) | `orgSettings` draft → approve **(م.ا)** |

## ۴. شکاف شناخته‌شدهٔ حمل (باید در گام بعد رفع شود)

اکنون **همهٔ** اقدام‌های رکورد از یک مسیر و یک اجازه عبور می‌کنند:
`POST /records/{id}/action` با اجازهٔ `approve` ماژول. نتیجه:

- «ارسال برای تأیید» توسط کارمند و «رد درخواست» توسط منابع انسانی از یک دروازهٔ دسترسی
  می‌گذرند؛ پس نقش‌هایی مثل `employee` که اجازهٔ `approve` ندارند عملاً نمی‌توانند
  «ارسال» کنند (اگرچه در دامنه مجازند).
- راه‌حل هدف: اجازهٔ هر اقدام از همان `Action` بخواند (مثلاً یک میدان `Permission`
  یا نگاشت action→permission) تا `recordkit.Register` به‌ازای هر اقدام دقیق‌ترین اجازه
  را بخواهد. تا آن زمان، نقش‌های دامنه به‌صورت «قصد درست» اعلام شده‌اند و موتور آن‌ها را
  به‌عنوان شرط دوم اعمال می‌کند.

## ۵. آنچه انجام شد و آنچه می‌ماند

**انجام‌شده**

- `contracts/workflow.go`: `WorkflowActor`, `WorkflowTask`, `WorkflowTaskPage`,
  `WorkflowSource`.
- `contracts/record.go`: `RecordActionDef.Roles/Scope` و `RecordWorkflowStep` روی شناسهٔ
  رکورد.
- `recordkit`: `Actor`, `Action.Roles/Scope`, `CanAct`/`AvailableActions`/`WorkflowSteps`,
  اجبار در `Act`، `TasksFor`.
- `modulekit.Ports`: ثبت منابع کارتابل؛ `recordkit.Mount` خودش را ثبت می‌کند.
- `GET /api/v1/workflow/inbox` با فیلتر سیاست نقش.
- مهاجرت مرجع: حضور و غیاب (زنجیرهٔ دومرحله‌ای مرخصی) و وام (تفکیک نقش).
- آزمون‌ها: `recordkit/workflow_test.go` و `bootstrap/workflow_flow_test.go` (زنجیرهٔ
  کامل از API + کارتابل).

**باقی‌مانده**

1. اعلام `Roles` برای بقیهٔ انواع رکوردِ جدول ۳-۳ (تغییر فقط در دامنه، بدون تغییر موتور).
2. رفع شکاف بند ۴ (اجازهٔ هر اقدام).
3. کارتابل در UI (صفحهٔ «کارتابل من») و نمایش نمودار مرحله‌ها از `workflow`.
4. قابلیت‌های بعدی ورک‌فلو: SLA/یادآور، نمایندگی (delegation)، تأیید موازی،
   شرط‌گذاری روی مقادیر فیلدها، و اعلان.

## ۶. تضمین سازگاری

- نقش خالی روی اقدام = رفتار پیشین؛ هیچ ماژول مهاجرت‌نکرده‌ای نمی‌شکند.
- مدیر سیستم همیشه مجاز است؛ گردش‌کار قفل نمی‌شود.
- کنشگر بی‌هویت (بذرکاری/فراخوانی داخلی) محدود نمی‌شود.
- کارتابل با سیاست `rbac` هم‌خوان است و چیزی بیش از دسترسی نقش نشان نمی‌دهد.
