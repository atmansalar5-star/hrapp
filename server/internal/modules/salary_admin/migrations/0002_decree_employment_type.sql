-- حکم حقوق روی Postgres: افزودن «نوع استخدام» به جدول.
--
-- چرا: نوع استخدام (رسمی، پیمانی، قراردادی…‌) شاکلهٔ کارگزینی و مبنای محاسبهٔ حکم و
-- پاداش پایان خدمت است، ولی در مهاجرت نخست جا افتاده بود و حکم روی Postgres بی‌نوع
-- استخدام ذخیره می‌شد (روی حافظه درست بود و همین ناهم‌خوانی دیر پیدا شد). این مهاجرت
-- ستون را با مقدار پیش‌فرض خالی اضافه می‌کند تا رکوردهای موجود سالم بمانند.

alter table hr_salary_admin.decree
  add column if not exists employment_type text not null default '';

comment on column hr_salary_admin.decree.employment_type is
  'نوع استخدام (official_oil، official_worker، long_term، fixed_1662، contractor_project…)';
