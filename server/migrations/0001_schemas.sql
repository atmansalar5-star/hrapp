-- مهاجرت پایه: ساخت schema اختصاصی هر ماژول.
--
-- قاعدهٔ مرز داده: هر ماژول فقط در schema خودش جدول دارد و هیچ کوئری‌ای بین schemaها
-- نوشته نمی‌شود؛ دسترسی بین‌ماژولی تنها از راه درخواست به API همان ماژول یا پورت
-- قراردادی انجام می‌شود. این کار مهاجرت یک ماژول را مستقل و بی‌خطر می‌کند.
--
-- ترتیب اعمال مهاجرت‌ها:
--   0001_schemas.sql      → ساخت schemaها (همین فایل)
--   000_seed_rbac.sql     → seed دسترسی‌ها (تولیدشده از مانیفست‌ها)
--   <module>/migrations/* → جدول‌های هر ماژول به ترتیب شماره

begin;

create schema if not exists hr_dashboard;
create schema if not exists hr_employee;
create schema if not exists hr_recruitment;
create schema if not exists hr_attendance;
create schema if not exists hr_payroll;
create schema if not exists hr_performance;
create schema if not exists hr_training;
create schema if not exists hr_welfare;
create schema if not exists hr_reports;
create schema if not exists hr_settings;

-- دفتر مهاجرت‌ها: هر ماژول مهاجرت‌های خودش را اینجا ثبت می‌کند.
create table if not exists public.schema_migration (
  module      text        not null,
  version     integer     not null,
  name        text        not null,
  applied_at  timestamptz not null default now(),
  primary key (module, version)
);

-- پارامترهای نسخه‌دار سامانه: تغییر قانون (نرخ بیمه، جدول مالیات) یک رکورد تازه است،
-- نه ویرایش رکورد قبلی؛ محاسبات گذشته با نسخهٔ خودشان بازتولیدپذیر می‌مانند.
create table if not exists hr_settings.sys_param_version (
  id          bigserial   primary key,
  key         text        not null,
  payload     jsonb       not null,
  effective_from date     not null,
  created_at  timestamptz not null default now(),
  created_by  text        not null,
  unique (key, effective_from)
);

comment on table hr_settings.sys_param_version is
  'پارامترهای سامانه به‌صورت نسخه‌دار؛ دوره‌های حقوق به نسخهٔ مؤثر خود ارجاع می‌دهند.';

commit;
