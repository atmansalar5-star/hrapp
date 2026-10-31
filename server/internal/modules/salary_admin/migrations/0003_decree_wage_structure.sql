-- حکم حقوق روی Postgres: اجزای ثابت مزد و فوق‌العاده‌های اقلیمی.
--
-- چرا: فیش حقوق «مزد مبنا» را از همین اجزا می‌سازد و فوق‌العاده‌های اقلیمی (بدی آب و هوا،
-- محرومیت از تسهیلات زندگی، صعوبت شرایط منطقه، جبران صرف وقت) به ازای هر کارمند در
-- حکم/قرارداد تعهد می‌شوند، نه فقط با یک ضریب سازمانی. بدون این ستون‌ها حکم روی Postgres
-- همان اطلاعات را از دست می‌داد (روی حافظه می‌ماند) و فیش هر دو استقرار یک عدد نمی‌داد.
--
-- قاعدهٔ مقدار: صفر یعنی «ثبت نشده» و مقدار پیش‌فرض سازمانی به کار می‌رود؛ پس ستون‌ها با
-- پیش‌فرض صفر افزوده می‌شوند و رکوردهای موجود دست‌نخورده می‌مانند.

alter table hr_salary_admin.decree
  add column if not exists climate_region text not null default '',
  add column if not exists rank_wage bigint not null default 0,
  add column if not exists seniority_wage bigint not null default 0,
  add column if not exists post_allowance bigint not null default 0,
  add column if not exists sacrifice_allowance bigint not null default 0,
  add column if not exists climate_allowance bigint not null default 0,
  add column if not exists salary_repair bigint not null default 0,
  add column if not exists bad_climate_percent double precision not null default 0,
  add column if not exists deprivation_percent double precision not null default 0,
  add column if not exists region_hardship_percent double precision not null default 0,
  add column if not exists extra_time_percent double precision not null default 0;

comment on column hr_salary_admin.decree.climate_region is
  'کلید منطقهٔ اقلیمی محل خدمت (جنوب و سواحل خلیج فارس، شمال، بیابانی، کوهستانی، مرکزی)';
comment on column hr_salary_admin.decree.climate_allowance is
  'فوق‌العادهٔ اقلیمی مصوب (ریال در ماه)؛ ثابت و جزو مزد مبنا';
comment on column hr_salary_admin.decree.bad_climate_percent is
  'ضریب بدی آب و هوا (۰٫۲۵ = ۲۵٪ مزد مبنا)؛ صفر یعنی از پارامتر شرکت خوانده شود';
comment on column hr_salary_admin.decree.deprivation_percent is
  'ضریب محرومیت از تسهیلات زندگی؛ صفر یعنی از پارامتر شرکت';
comment on column hr_salary_admin.decree.region_hardship_percent is
  'ضریب صعوبت شرایط منطقه؛ صفر یعنی از پارامتر شرکت';
comment on column hr_salary_admin.decree.extra_time_percent is
  'ضریب جبران صرف وقت اضافی؛ صفر یعنی از پارامتر شرکت';