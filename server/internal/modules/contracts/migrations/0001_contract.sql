-- ماژول کارگزینی و قراردادها — جدول‌های قرارداد کار و سابقهٔ رخدادها.
--
-- تاریخ‌ها شمسی و به‌صورت text ذخیره می‌شوند (نه date میلادی): همان چیزی که کاربر
-- وارد و تأیید می‌کند ذخیره می‌شود و تبدیل تقویمی در لایهٔ کد انجام می‌گیرد؛ این‌گونه
-- خطای یک‌روزهٔ منطقهٔ زمانی به داده راه پیدا نمی‌کند.

create schema if not exists hr_contracts;

create sequence if not exists hr_contracts.contract_seq start 200;

create table if not exists hr_contracts.contract (
  id              text        primary key,
  employee_id     text        not null,
  employee_name   text        not null,
  department_name text        not null default '',
  position_title  text        not null default '',
  kind            text        not null,
  start_date      text        not null,
  end_date        text        not null default '',
  monthly_salary  bigint      not null,
  status          text        not null,
  supersedes_id   text        not null default '',
  renew_count     integer     not null default 0,
  note_fa         text        not null default '',
  created_by      text        not null default '',
  decided_by      text        not null default '',
  version         integer     not null default 1,
  updated_at      timestamptz not null default now()
);

-- فهرست‌های پرکار: قراردادهای یک کارمند و صف تأیید/سرآمد.
create index if not exists contract_employee_idx on hr_contracts.contract (employee_id, status);
create index if not exists contract_status_idx   on hr_contracts.contract (status, end_date);

comment on table hr_contracts.contract is
  'قرارداد کار؛ یک قرارداد جاری در هر زمان برای هر کارمند (قاعده در دامنه اجرا می‌شود).';

create table if not exists hr_contracts.contract_event (
  id          bigserial   primary key,
  contract_id text        not null references hr_contracts.contract (id) on delete cascade,
  at          timestamptz not null default now(),
  actor       text        not null default '',
  action      text        not null,
  note_fa     text        not null default ''
);

create index if not exists contract_event_contract_idx on hr_contracts.contract_event (contract_id, at desc);

comment on table hr_contracts.contract_event is
  'سابقهٔ زندگی قرارداد: انعقاد، تأیید، رد، تمدید، تبدیل و خاتمه — هرگز ویرایش نمی‌شود.';
