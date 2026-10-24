-- موتور مشترک «رکورد گردش‌کاری» — جدول مشترک ماژول‌های اداری.
--
-- ماژول‌هایی مثل انتصابات، ترفیعات، بایگانی، انضباط کار و مطالعات منابع انسانی یک الگوی
-- یکسان دارند: رکوردی با فیلدها، وضعیت، گردش‌کار و سابقهٔ تغییرات. یک جدول مشترک با
-- ستون `module` و `kind` از تکرار schema برای هر ماژول جلوگیری می‌کند و قاعدهٔ هر ماژول
-- در بستهٔ domain خودش اعلام می‌شود (نه در SQL).
--
-- فیلدها در ستون jsonb نگه داشته می‌شوند چون هر ماژول مجموعه‌ای متفاوت دارد؛ ولی
-- جستجوی متنی روی همان ستون هم کار می‌کند (fields::text ilike).
--
-- رخدادها در جدول جدا و فقط-افزودنی ثبت می‌شوند تا «چه کسی و چرا» پس از سال‌ها پاسخ
-- داشته باشد.

create schema if not exists hr_records;

create table if not exists hr_records.record (
  id              text        primary key,
  module          text        not null,
  kind            text        not null,
  title           text        not null default '',
  employee_id     text        not null default '',
  employee_name   text        not null default '',
  department_name text        not null default '',
  status          text        not null,
  fields          jsonb       not null default '{}'::jsonb,
  note_fa         text        not null default '',
  created_by      text        not null default '',
  decided_by      text        not null default '',
  version         integer     not null default 1,
  updated_at      timestamptz not null default now()
);

create index if not exists record_module_kind_idx on hr_records.record (module, kind, status);
create index if not exists record_employee_idx    on hr_records.record (employee_id, status);
create index if not exists record_fields_idx      on hr_records.record using gin (fields);

comment on table hr_records.record is
  'رکورد گردش‌کاری ماژول‌های اداری؛ قاعدهٔ هر ماژول در بستهٔ domain همان ماژول است.';

-- شمارندهٔ مستقل برای شناسه‌گذاری هر نوع رکورد (سبک، انتصاب، پرونده…).
create table if not exists hr_records.seq (
  key text   primary key,
  n   bigint not null default 0
);

comment on table hr_records.seq is
  'شمارندهٔ شناسهٔ هر نوع رکورد؛ شناسه‌ها بدون هم‌پوشانی و به‌ترتیب صادر می‌شوند.';

create table if not exists hr_records.record_event (
  id        bigserial   primary key,
  record_id text        not null references hr_records.record (id) on delete cascade,
  at        timestamptz not null default now(),
  actor     text        not null default '',
  action    text        not null,
  note_fa   text        not null default ''
);

create index if not exists record_event_record_idx on hr_records.record_event (record_id, at);

comment on table hr_records.record_event is
  'سابقهٔ گردش‌کار رکورد: ثبت، ویرایش، تأیید، رد، اجرا و بستن — هرگز ویرایش نمی‌شود.';
