-- دستهٔ پرداخت حکم کارگزینی: گروه نظام پرداخت نفت + روش پرداخت مصوب.
--
-- چرا: حکم سند مبنای پرداخت است؛ باید بگوید با کدام گروه جدول ۳ و کدام موتور محاسبه صادر
-- شده تا فیش و اسناد چاپی همان را بازتولید کنند. این دو با هم «دستهٔ پرداخت» کارمند را
-- می‌سازند (بندهای OPG-03 و PM-02 سند docs/pay-classification-spec-fa.md).
--
-- مقدار صفر/خالی یعنی «تعیین نشده»؛ سرویس هنگام صدور، گروه را از نوع استخدام و گروه شغلی
-- و روش پرداخت را از قرارداد جاری (یا انتخاب صریح) استنتاج و ذخیره می‌کند.

begin;

alter table hr_salary_admin.decree
  add column if not exists pay_group integer not null default 0,
  add column if not exists pay_model text not null default '';

comment on column hr_salary_admin.decree.pay_group is 'گروه نظام پرداخت نفت (جدول ۳): ۱ = کارمند رسمی نفت، ۲ = پایهٔ سمت ۶ تا ۱۰، …';
comment on column hr_salary_admin.decree.pay_model is 'روش پرداخت مصوب: payroll | oil | comp34006 | fourp';

create index if not exists decree_pay_group_idx on hr_salary_admin.decree (pay_group);
create index if not exists decree_pay_model_idx on hr_salary_admin.decree (pay_model);

commit;
