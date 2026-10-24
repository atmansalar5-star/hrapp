-- ماژول افراد تحت تکفل و درخواست پرواز — جدول‌های تکفل، پوشش بیمه و فرم پرواز.
--
-- تاریخ‌ها شمسی و به‌صورت text ذخیره می‌شوند (همان چیزی که کاربر ثبت و تأیید می‌کند).

create schema if not exists hr_personnel;

create sequence if not exists hr_personnel.dependent_seq start 100;
create sequence if not exists hr_personnel.flight_seq    start 200;

create table if not exists hr_personnel.dependent (
  id             text        primary key,
  employee_id    text        not null,
  employee_name  text        not null,
  department_name text       not null default '',
  full_name      text        not null,
  relation       text        not null,
  national_id    text        not null default '',
  birth_date     text        not null default '',
  coverage_start text        not null default '',
  insurance      boolean     not null default false,
  documents_fa   text        not null default '',
  status         text        not null,
  note_fa        text        not null default '',
  decided_by     text        not null default '',
  version        integer     not null default 1,
  updated_at     timestamptz not null default now()
);

create index if not exists dependent_employee_idx on hr_personnel.dependent (employee_id, status);
create index if not exists dependent_national_idx on hr_personnel.dependent (national_id);

comment on table hr_personnel.dependent is
  'افراد تحت تکفل کارکنان؛ مبنای پوشش بیمهٔ درمانی و خدمات درمانی خانواده.';

create table if not exists hr_personnel.flight_request (
  id             text        primary key,
  employee_id    text        not null,
  employee_name  text        not null,
  department_name text       not null default '',
  origin_fa      text        not null,
  destination_fa text        not null,
  depart_date    text        not null,
  return_date    text        not null default '',
  purpose        text        not null,
  passengers     integer     not null default 1,
  companions_fa  text        not null default '',
  estimated_cost bigint      not null default 0,
  status         text        not null,
  note_fa        text        not null default '',
  decided_by     text        not null default '',
  ticket_no      text        not null default '',
  version        integer     not null default 1,
  updated_at     timestamptz not null default now()
);

create index if not exists flight_employee_idx on hr_personnel.flight_request (employee_id, status);
create index if not exists flight_status_idx   on hr_personnel.flight_request (status, depart_date);

comment on table hr_personnel.flight_request is
  'فرم‌های درخواست پرواز کارکنان از ثبت تا تأیید مدیر واحد، صدور بلیت و تسویهٔ هزینه.';
