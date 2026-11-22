-- فرادادهٔ «زیرفرم حقوق و مزایا» در قرارداد.
--
-- چرا یک ستون JSONB و نه چند ستون تازه: شکل بخش حقوقی قرارداد با نوع استخدام عوض می‌شود
-- (جدول مزد رسمی / مبلغ و مبنای پیمان / تفاهم‌نامهٔ پرداخت مأموریت). ستون‌های ثابت برای
-- اجزای مزد رسمی سر جای خود می‌مانند و فقط فیلدهای مخصوص هر خانواده اینجا می‌نشیند؛
-- افزودن فیلد تازه در آینده هم مهاجرت ستونی لازم ندارد.
--
-- pay_family هم ذخیره می‌شود (نه فقط استنتاج از نوع استخدام) تا اگر فهرست انواع استخدام
-- روزی عوض شد، سند قدیمی بگوید با کدام شکل فرم ثبت شده بود.

begin;

alter table hr_contracts.contract
  add column if not exists pay_family text not null default 'official',
  add column if not exists pay_meta   jsonb not null default '{}'::jsonb;

comment on column hr_contracts.contract.pay_family is 'خانوادهٔ پرداخت: official | contracting | secondment';
comment on column hr_contracts.contract.pay_meta   is 'فیلدهای مخصوص زیرفرم همان خانواده (پیمان/مأموریت)';

create index if not exists contract_pay_family_idx on hr_contracts.contract (pay_family);

commit;
