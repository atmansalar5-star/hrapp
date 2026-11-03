-- پیکربندی ستون‌های «اطلاعات پایهٔ کارمند» روی جدول موجود (schema: hr_employee)
--
-- چرا این مهاجرت: جدول کارکنان پیش از افزودن اطلاعات پایهٔ پرونده (شمارهٔ همراه، تلفن
-- منزل، کد پستی و نشانی) و نوع استخدام ساخته شده بود. افزودن ستون‌ها با
-- `add column if not exists` است تا روی پایگاه‌دادهٔ موجود بی‌خطر و تکرارپذیر باشد و
-- مقدار پیش‌فرض خالی، رکوردهای قدیمی را دست‌نخورده بگذارد.

begin;

-- نوع استخدام (کلید کاتالوگ contracts.EmploymentType) — مبنای حکم، قرارداد و مزد.
alter table hr_employee.employee add column if not exists employment_type text not null default '';

-- اطلاعات پایهٔ پرونده: مرجع همهٔ فرم‌ها (قرارداد، حکم، کارگزینی و گزارش‌ها).
alter table hr_employee.employee add column if not exists mobile      text not null default '';
alter table hr_employee.employee add column if not exists home_phone  text not null default '';
alter table hr_employee.employee add column if not exists postal_code text not null default '';
alter table hr_employee.employee add column if not exists address     text not null default '';

-- جستجوی کارکنان با شمارهٔ همراه/کد پستی در گزارش‌های بیمه و مکاتبات پرکاربرد است.
create index if not exists employee_mobile_idx on hr_employee.employee (mobile) where not deleted;
create index if not exists employee_postal_idx on hr_employee.employee (postal_code) where not deleted;

insert into public.schema_migration (module, version, name)
values ('employee', 2, '0002_employee_base_info')
on conflict do nothing;

commit;
