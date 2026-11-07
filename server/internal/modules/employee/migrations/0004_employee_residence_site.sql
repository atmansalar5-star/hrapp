-- محل سکونت کارمند (schema: hr_employee)
--
-- چرا این مهاجرت: آیین‌نامهٔ فوق‌العاده‌های صندوق بازنشستگی نفت، دو فوق‌العاده را بر پایهٔ
-- «محل سکونت» کارمند تعیین می‌کند: جدول ۱ (محرومیت از تسهیلات زندگی) و جدول ۲ (جبران صرف
-- وقت اضافی، تفکیک‌شده برای روزکار و نوبت‌کار). پیش از این یک درصد ثابت در پارامترهای شرکت
-- بود؛ با این ستون، درصد از پروندهٔ خود کارمند خوانده می‌شود.
--
-- مقدار ستون «کلید» است (mahshahr، bandar_imam، jarrahi، ramshir، ramhormoz، shadegan،
-- omidieh، abadan) و فهرست مرجعش contracts.OilResidences است؛ عنوان فارسی در پاسخ API از
-- همان کاتالوگ ساخته می‌شود و در جدول تکرار نمی‌شود.
--
-- افزودن ستون با `add column if not exists` است تا روی پایگاه‌دادهٔ موجود بی‌خطر و
-- تکرارپذیر باشد؛ رکوردهای قدیمی مقدار خالی می‌گیرند و رفتارشان (درصد پارامتر شرکت)
-- دست‌نخورده می‌ماند.

begin;

alter table hr_employee.employee add column if not exists residence_site text not null default '';

create index if not exists employee_residence_site_idx
  on hr_employee.employee (residence_site) where not deleted;

insert into public.schema_migration (module, version, name)
values ('employee', 4, '0004_employee_residence_site')
on conflict do nothing;

commit;