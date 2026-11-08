-- تفاوت تطبیق حقوق و مزایا روی حکم کارگزینی.
--
-- چرا: وقتی طرح حقوق تازه (طبقه‌بندی مشاغل یا بازنگری حقوق) اجرا می‌شود، ممکن است حقوق
-- کارمند کمتر شود؛ مابه‌التفاوت به‌عنوان «تفاوت تطبیق» در همان حکم ثبت می‌شود تا فیش حقوق و
-- فرمت حقوق آن را نشان دهند و با افزایش‌های بعدی جذب شود.
--
-- قاعدهٔ مقدار: صفر یعنی «تفاوت تطبیق ندارد». مبلغ جذب‌شدنی است و جزو مزد مبنا نیست.

alter table hr_salary_admin.decree
  add column if not exists pay_gap bigint not null default 0,
  add column if not exists pay_gap_since text not null default '';

comment on column hr_salary_admin.decree.pay_gap is
  'تفاوت تطبیق ماهانه (ریال)؛ مبلغ جذب‌شدنی که با افزایش حقوق کاهش می‌یابد و جزو مزد مبنا نیست';
comment on column hr_salary_admin.decree.pay_gap_since is
  'تاریخ شروع تفاوت تطبیق به قالب شمسی YYYY/MM/DD';