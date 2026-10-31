-- قرارداد کار روی Postgres: نوع استخدام، اجزای ثابت مزد و فوق‌العاده‌های اقلیمی.
--
-- چرا: (۱) نوع استخدام در قرارداد ذخیره نمی‌شد و فیش حقوق/سند چاپی، نوع استخدام کارمند را
-- فقط از حکم می‌گرفت؛ روی حافظه درست بود و روی Postgres نه. (۲) فوق‌العاده‌های اقلیمی
-- (بدی آب و هوا، محرومیت از تسهیلات زندگی، صعوبت شرایط منطقه، جبران صرف وقت) و اجزای ثابت
-- مزد در قرارداد تعهد می‌شوند و فیش حقوق مزد مبنا را از همان‌ها می‌سازد.
--
-- قاعدهٔ مقدار: صفر یعنی «ثبت نشده» و مقدار پیش‌فرض سازمانی به کار می‌رود؛ پس ستون‌ها با
-- پیش‌فرض خالی/صفر افزوده می‌شوند و قراردادهای موجود دست‌نخورده می‌مانند.

alter table hr_contracts.contract
  add column if not exists employment_type text not null default '',
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

comment on column hr_contracts.contract.employment_type is
  'نوع استخدام (official_oil، official_worker، long_term، fixed_1662، contractor_project...)';
comment on column hr_contracts.contract.climate_region is
  'کلید منطقهٔ اقلیمی محل خدمت که ضرایب پیشنهادی فوق‌العاده‌های اقلیمی را می‌آورد';
comment on column hr_contracts.contract.climate_allowance is
  'فوق‌العادهٔ اقلیمی مصوب (ریال در ماه)؛ ثابت و جزو مزد مبنا';
comment on column hr_contracts.contract.bad_climate_percent is
  'ضریب بدی آب و هوا (۰٫۲۵ = ۲۵٪ مزد مبنا)؛ صفر یعنی از پارامتر شرکت';
comment on column hr_contracts.contract.deprivation_percent is
  'ضریب محرومیت از تسهیلات زندگی؛ صفر یعنی از پارامتر شرکت';
comment on column hr_contracts.contract.region_hardship_percent is
  'ضریب صعوبت شرایط منطقه؛ صفر یعنی از پارامتر شرکت';
comment on column hr_contracts.contract.extra_time_percent is
  'ضریب جبران صرف وقت اضافی؛ صفر یعنی از پارامتر شرکت';