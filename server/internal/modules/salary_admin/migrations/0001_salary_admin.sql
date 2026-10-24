-- ماژول احکام حقوق و گواهی اشتغال — جدول‌های حکم حقوق و گواهی اشتغال.
--
-- تاریخ‌ها شمسی و به‌صورت text ذخیره می‌شوند (نه date میلادی): همان چیزی که کاربر
-- وارد می‌کند ذخیره می‌شود و تبدیل تقویمی در لایهٔ کد انجام می‌گیرد.

create schema if not exists hr_salary_admin;

create sequence if not exists hr_salary_admin.decree_seq start 300;
create sequence if not exists hr_salary_admin.certificate_seq start 400;

create table if not exists hr_salary_admin.decree (
  id             text        primary key,
  employee_id    text        not null,
  employee_name  text        not null,
  department_name text       not null default '',
  position_title text        not null default '',
  job_group      text        not null,
  base_salary    bigint      not null,
  effective_date text        not null,
  reason_fa      text        not null default '',
  status         text        not null,
  supersedes_id  text        not null default '',
  note_fa        text        not null default '',
  created_by     text        not null default '',
  decided_by     text        not null default '',
  version        integer     not null default 1,
  updated_at     timestamptz not null default now()
);

create index if not exists decree_employee_idx on hr_salary_admin.decree (employee_id, status);
create index if not exists decree_status_idx   on hr_salary_admin.decree (status, effective_date);

comment on table hr_salary_admin.decree is
  'حکم حقوق: یک حکم ابلاغ‌شده در هر زمان برای هر کارمند؛ اصلاح، حکم تازه می‌سازد و پیشین را منسوخ می‌کند.';

create table if not exists hr_salary_admin.certificate (
  id             text        primary key,
  serial         text        not null,
  employee_id    text        not null,
  employee_name  text        not null,
  department_name text       not null default '',
  position_title text        not null default '',
  kind           text        not null,
  purpose_fa     text        not null default '',
  requested_at   text        not null default '',
  issued_at      text        not null default '',
  delivered_at   text        not null default '',
  status         text        not null,
  note_fa        text        not null default '',
  decided_by     text        not null default '',
  version        integer     not null default 1,
  updated_at     timestamptz not null default now()
);

create unique index if not exists certificate_serial_idx on hr_salary_admin.certificate (serial);
create index if not exists certificate_employee_idx on hr_salary_admin.certificate (employee_id, status);

comment on table hr_salary_admin.certificate is
  'گواهی اشتغال به کار: گردش‌کار درخواست ← تأیید ← صدور ← تحویل با سریال یکتا.';
