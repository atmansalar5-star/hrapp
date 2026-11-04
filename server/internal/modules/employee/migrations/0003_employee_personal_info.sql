-- اطلاعات پرسنلی کارمند: جنسیت و تاریخ تولد (schema: hr_employee)
--
-- چرا این مهاجرت: برش‌های آماری «جنسیت × واحد»، «جنسیت × نوع استخدام» و «جنسیت ×
-- تحصیلات» و همچنین «گروه سنی × سابقهٔ خدمت» به این دو داده نیاز دارند. جنسیت و تاریخ
-- تولد اطلاعات پایهٔ پروندهٔ کارمندند، پس در همان جدول کارکنان نگه داشته می‌شوند (نه در
-- جدول کنارِ ماژول آمار) تا هر ماژول از یک منبع بخواند.
--
-- افزودن ستون‌ها با `add column if not exists` است تا روی پایگاه‌دادهٔ موجود بی‌خطر و
-- تکرارپذیر باشد؛ مقدار پیش‌فرض خالی/NULL رکوردهای قدیمی را دست‌نخورده می‌گذارد.

begin;

-- جنسیت: کلید کاتالوگ contracts.GenderCatalog (male/female)؛ خالی یعنی ثبت نشده.
alter table hr_employee.employee add column if not exists gender text not null default '';

-- تاریخ تولد شمسی است ولی مثل تاریخ استخدام در ستون date میلادی نگه داشته می‌شود؛
-- تبدیل تقویمی در لایهٔ کد انجام می‌گیرد (قاعدهٔ پروژه).
alter table hr_employee.employee add column if not exists birth_date date;

-- گروه‌بندی سنی و جنسیتی گزارش‌های آماری روی همین دو ستون فیلتر/جمع می‌شود.
create index if not exists employee_gender_idx on hr_employee.employee (gender) where not deleted;
create index if not exists employee_birth_idx on hr_employee.employee (birth_date) where not deleted;

insert into public.schema_migration (module, version, name)
values ('employee', 3, '0003_employee_personal_info')
on conflict do nothing;

commit;
