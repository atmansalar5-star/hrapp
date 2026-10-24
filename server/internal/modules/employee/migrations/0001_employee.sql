-- مهاجرت ماژول کارکنان (schema: hr_employee)
--
-- این فایل با پیاده‌سازی فعلی (انبار درون‌حافظه) سازگار است و پیش از فعال‌کردن
-- pgStore اعمال می‌شود. ستون‌های نسخه‌بندی (version, server_seq) برای سینک آفلاین
-- و ستون deleted برای tombstone لازم است تا کلاینت‌های بی‌شبکه هم به‌درستی
-- هم‌گام شوند (به core/crates/hrms-store مراجعه کنید).

begin;

create table if not exists hr_employee.department (
  id          text        primary key,
  name        text        not null,
  head_name   text,
  parent_id   text        references hr_employee.department (id),
  headcount   integer     not null default 0,
  created_at  timestamptz not null default now()
);

create table if not exists hr_employee.position (
  id            text primary key,
  title         text not null,
  department_id text references hr_employee.department (id)
);

create table if not exists hr_employee.employee (
  id              text        primary key,
  full_name       text        not null,
  national_id     text        not null unique,
  department_id   text        not null references hr_employee.department (id),
  position        text        not null,
  contract_type   text        not null check (contract_type in ('تمام‌وقت', 'پیمانکاری', 'پاره‌وقت')),
  status          text        not null check (status in ('active', 'leave', 'probation', 'terminated')),
  hire_date       date        not null,
  base_salary     bigint      not null default 0 check (base_salary >= 0),
  leave_balance   numeric(5, 2) not null default 0,
  phone           text,
  email           text,
  manager_name    text,
  version         integer     not null default 1,
  server_seq      bigint      not null default 0,
  deleted         boolean     not null default false,
  created_at      timestamptz not null default now(),
  updated_at      timestamptz not null default now()
);

-- فهرست کارکنان پرکاربردترین کوئری سامانه است: فیلتر واحد/وضعیت + جستجوی نام.
create index if not exists employee_department_idx on hr_employee.employee (department_id) where not deleted;
create index if not exists employee_status_idx on hr_employee.employee (status) where not deleted;
create index if not exists employee_fullname_idx on hr_employee.employee (full_name) where not deleted;
create index if not exists employee_contract_idx on hr_employee.employee (contract_type) where not deleted;
-- دلتای سینک بر پایهٔ نشانگر سرور خوانده می‌شود.
create index if not exists employee_seq_idx on hr_employee.employee (server_seq);

-- سابقهٔ شغلی و قرارداد کارمند (تب‌های پرونده در سند طراحی).
create table if not exists hr_employee.contract (
  id           text        primary key,
  employee_id  text        not null references hr_employee.employee (id) on delete cascade,
  contract_type text       not null,
  start_date   date        not null,
  end_date     date,
  base_salary  bigint      not null default 0,
  created_at   timestamptz not null default now()
);

create table if not exists hr_employee.document (
  id          text        primary key,
  employee_id text        not null references hr_employee.employee (id) on delete cascade,
  title       text        not null,
  file_ref    text        not null,
  uploaded_at timestamptz not null default now()
);

create table if not exists hr_employee.skill (
  employee_id text not null references hr_employee.employee (id) on delete cascade,
  name        text not null,
  level       smallint not null check (level between 1 and 5),
  primary key (employee_id, name)
);

-- ردیف‌های تغییر برای سینک: هر نوشتن یک ردیف append-only می‌سازد.
create table if not exists hr_employee.change_log (
  seq        bigserial   primary key,
  entity     text        not null,
  entity_id  text        not null,
  op         text        not null check (op in ('upsert', 'delete')),
  version    integer     not null,
  payload    jsonb,
  actor      text        not null,
  created_at timestamptz not null default now()
);

create index if not exists change_log_seq_idx on hr_employee.change_log (seq);

insert into public.schema_migration (module, version, name)
values ('employee', 1, '0001_employee')
on conflict do nothing;

commit;
