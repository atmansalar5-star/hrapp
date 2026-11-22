-- دستهٔ پرداخت قرارداد: گروه نظام پرداخت نفت + روش پرداخت مصوب.
--
-- چرا: «کارمند رسمی نفت» از گروه یکم جدول ۳ آیین‌نامه جدا شد (گروه یکم اکنون گروه مستقل
-- رسمی‌های نفت و گروه دوم پایهٔ سمت ۶ تا ۱۰) و روش پرداخت هر نفر هم باید در سند ثبت شود تا
-- حکم و فیش بدانند با کدام موتور حساب شوند. این دو با هم «دستهٔ پرداخت» کارمند را می‌سازند:
--   pay_group ⇒ درصدهای جدول ۳ (حق جذب عمومی و حق مقام)
--   pay_model ⇒ payroll | oil | comp34006 | fourp
--
-- مقدار صفر/خالی یعنی «تعیین نشده»؛ سرویس هنگام ثبت، از نوع استخدام و گروه شغلی استنتاج
-- و همان را ذخیره می‌کند (بندهای OPG-03 و PM-02 سند docs/pay-classification-spec-fa.md).

begin;

alter table hr_contracts.contract
  add column if not exists pay_group integer not null default 0,
  add column if not exists pay_model text not null default '';

comment on column hr_contracts.contract.pay_group is 'گروه نظام پرداخت نفت (جدول ۳): ۱ = کارمند رسمی نفت، ۲ = پایهٔ سمت ۶ تا ۱۰، …';
comment on column hr_contracts.contract.pay_model is 'روش پرداخت مصوب: payroll | oil | comp34006 | fourp';

create index if not exists contract_pay_group_idx on hr_contracts.contract (pay_group);
create index if not exists contract_pay_model_idx on hr_contracts.contract (pay_model);

commit;
