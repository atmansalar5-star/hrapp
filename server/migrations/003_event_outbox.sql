-- صندوق خروجی رخداد (event outbox) — زیرساخت یکپارچه‌سازی با سامانه‌های بیرونی.
--
-- چرا صندوق و نه ارسال مستقیم: تغییرهای سامانه باید بی‌وابستگی به در دسترس بودن
-- سامانهٔ بیرونی ثبت شوند. هر رخداد پیش از پاسخ به کاربر در همین جدول می‌نشیند و یک
-- گردانندهٔ پس‌زمینه آن را با امضای HMAC و تلاش مجدد تحویل می‌دهد؛ پس نه سرعت پاسخ
-- API به مقصد بیرونی گره می‌خورد و نه رخدادی گم می‌شود.
--
-- این جدول دارایی «پلتفرم» است، نه هیچ ماژولی؛ به همین دلیل schema جدا دارد و هیچ
-- ماژولی مستقیماً آن را نمی‌خواند (فقط بستهٔ platform/events).
--
-- وضعیت تحویل: pending (در انتظار یا در انتظار تلاش بعدی)، delivered (تحویل شد)،
-- failed (پس از مصرف همهٔ تلاش‌ها ناموفق ماند و برای بازبینی دستی می‌ماند).

create schema if not exists hr_platform;

create table if not exists hr_platform.event_outbox (
  id              bigserial   primary key,
  event_id        text        not null unique,
  name            text        not null,
  occurred_at     timestamptz not null default now(),
  actor           text        not null default '',
  role            text        not null default '',
  module          text        not null default '',
  method          text        not null default '',
  path            text        not null default '',
  status          integer     not null default 0,
  subject         text        not null default '',
  payload         jsonb       not null default '{}'::jsonb,
  status_delivery text        not null default 'pending',
  attempts        integer     not null default 0,
  next_attempt_at timestamptz not null default now(),
  delivered_at    timestamptz,
  last_error      text        not null default '',
  created_at      timestamptz not null default now()
);

-- وضعیت تحویل عمداً در ستون `status_delivery` است: `status` برای کد پاسخ HTTP درخواستی
-- است که رخداد را ساخته و اشتباه‌گرفتن این دو، پایش صف را گمراه می‌کند.
comment on table hr_platform.event_outbox is
  'رخداد تغییرهای API برای یکپارچه‌سازی بیرونی؛ ثبت در همان تراکنش دامنه و ارسال پس‌زمینه.';

create index if not exists event_outbox_due_idx
  on hr_platform.event_outbox (next_attempt_at)
  where status_delivery = 'pending';

create index if not exists event_outbox_name_idx
  on hr_platform.event_outbox (name, occurred_at desc);

create index if not exists event_outbox_created_idx
  on hr_platform.event_outbox (id desc);