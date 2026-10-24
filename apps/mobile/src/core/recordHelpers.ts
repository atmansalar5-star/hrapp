/**
 * کمک‌های خالصِ رکوردهای گردش‌کاری (recordkit) — بدون وابستگی به React و شبکه تا در
 * آزمون واحد پوشش داده شوند. همهٔ برچسب‌ها، فیلدها، وضعیت‌ها و اقدام‌ها از `/schema`
 * سرور می‌آیند؛ اینجا فقط «نمایش» شکل می‌گیرد و هیچ قاعدهٔ کسب‌وکاری تکرار نمی‌شود.
 * مجوز واقعی هر اقدام را سرور می‌سنجد.
 */

/** فیلد اعلام‌شده در شناسهٔ ماژول. */
export interface FieldDef {
  key: string;
  labelFa: string;
  type: string;
  required: boolean;
}

/** وضعیت گردش‌کار با برچسب فارسی. */
export interface StatusDef {
  value: string;
  labelFa: string;
}

/** اقدام اعلام‌شده در شناسهٔ ماژول. */
export interface ActionDef {
  value: string;
  labelFa: string;
  style: string;
  to: string;
  noteRequired: boolean;
  roles?: string[];
  scope?: string;
}

/** یک نوع رکورد در شناسهٔ ماژول. */
export interface KindDef {
  key: string;
  titleFa: string;
  idPrefix: string;
  fields: FieldDef[];
  statuses: StatusDef[];
  actions: ActionDef[];
  /** وضعیت فعلی → مقصدهای مجاز (از دامنه؛ در UI بازنویسی نمی‌شود). */
  transitions: Record<string, string[]>;
  initialStatus: string;
  editableStatuses?: string[];
  employeeScoped: boolean;
}

/** شناسهٔ کامل ماژول از /schema. */
export interface ModuleSchema {
  module: string;
  moduleTitleFa: string;
  kinds: KindDef[];
}

/** یک رکورد از /records. */
export interface RecordRow {
  id: string;
  kind: string;
  title: string;
  employeeId?: string;
  status: string;
  statusFa: string;
  fields?: Record<string, string>;
  noteFa?: string;
  version: number;
}

/** صفحهٔ رکوردها. */
export interface RecordPage {
  items: RecordRow[];
  total: number;
  page: number;
  size: number;
}

/** ورودی ایجاد/ویرایش رکورد (همان RecordInput سرور). */
export interface RecordInputBody {
  kind: string;
  title: string;
  employeeId?: string;
  fields: Record<string, string>;
  noteFa?: string;
}

/** نخستین فیلدهای شناسه برای ستون‌های جدول موبایل (باقی‌مانده در فرم دیده می‌شود). */
export function visibleFields(kind: KindDef, max = 4): FieldDef[] {
  return kind.fields.slice(0, max);
}

/** آیا این وضعیت، وضعیت مجاز ویرایش است (از همان فهرست اعلامی سرور). */
export function editableIn(kind: KindDef, status: string): boolean {
  const list = kind.editableStatuses && kind.editableStatuses.length > 0 ? kind.editableStatuses : [kind.initialStatus];
  return list.includes(status);
}

/**
 * اقدام‌هایی که از این وضعیت بیرون می‌روند — فقط برای نمایش دکمه‌ها؛ مجوز واقعی را
 * سرور می‌سنجد. اقدام‌های «نیازمند یادداشت» فعلاً نمایش داده نمی‌شوند چون فرم یادداشت
 * در این نسخه از صفحهٔ پویا نیست.
 */
export function actionsFor(kind: KindDef, status: string): ActionDef[] {
  const allowed = kind.transitions[status] ?? [];
  return kind.actions.filter(action => action.to !== '' && allowed.includes(action.to) && !action.noteRequired);
}

/** خطاهای پاسخ سرور (`details` با کلیدهای fields.x) را به خطای فیلدِ همان فرم تبدیل می‌کند. */
export function fieldErrorsFrom(details: Record<string, string>): { fields: Record<string, string>; form: string } {
  const fields: Record<string, string> = {};
  const general: string[] = [];
  for (const [key, message] of Object.entries(details)) {
    if (key.startsWith('fields.')) {
      fields[key.slice('fields.'.length)] = message;
    } else {
      general.push(message);
    }
  }
  return { fields, form: general.join(' ') };
}

/** جستجوی سادهٔ سمت کلاینت روی عنوان و مقادیر فیلدها. */
export function searchRows(rows: RecordRow[], query: string): RecordRow[] {
  const needle = query.trim().toLowerCase();
  if (needle === '') {
    return rows;
  }
  return rows.filter(row => {
    if (row.title.toLowerCase().includes(needle)) {
      return true;
    }
    return Object.values(row.fields ?? {}).some(value => value.toLowerCase().includes(needle));
  });
}

/** مقادیر یکتای یک فیلد برای چیپ‌های گروه‌بندی (فقط نمایش؛ قاعده‌ای در کار نیست). */
export function distinctValues(rows: RecordRow[], fieldKey: string): string[] {
  const set = new Set<string>();
  for (const row of rows) {
    const value = (row.fields ?? {})[fieldKey]?.trim();
    if (value) {
      set.add(value);
    }
  }
  return [...set].sort((a, b) => a.localeCompare(b, 'fa'));
}
