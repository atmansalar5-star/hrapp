-- گروه شغلی کارمند (schema: hr_employee)
--
-- چرا این مهاجرت: اجرای ابلاغ انتصاب یا ترفیع، پست و واحد کارمند را عوض می‌کند و باید
-- گروه شغلی او را هم ثبت کند؛ وگرنه نتیجهٔ ترفیع فقط در رکورد ابلاغ می‌ماند و پرونده و
-- فیش حقوق از آن بی‌خبرند. فیش نظام صندوق نفت «پایهٔ سمت» را از همین گروه پیشنهاد
-- می‌دهد (با نگاشت پارامتریک oilpay.gradeMap.<گروه>).
--
-- مقدار ستون «کلید» نردبان گروه‌های شغلی است (employee، expert، senior_expert،
-- supervisor، manager، senior_manager) و فهرست مرجعش rules.DefaultGradeLadder است؛
-- عنوان فارسی در پاسخ API از همان کاتالوگ ساخته می‌شود و در جدول تکرار نمی‌شود.
--
-- افزودن ستون با `add column if not exists` است تا روی پایگاه‌دادهٔ موجود بی‌خطر و
-- تکرارپذیر باشد؛ رکوردهای قدیمی مقدار خالی می‌گیرند.

begin;

alter table hr_employee.employee add column if not exists job_grade text not null default '';

insert into public.schema_migration (module, version, name)
values ('employee', 5, '0005_employee_job_grade')
on conflict do nothing;

commit;