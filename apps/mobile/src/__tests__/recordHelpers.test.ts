/**
 * آزمون کمک‌های خالصِ رکوردهای پویا — همان منطقی که صفحهٔ پارامترهای حقوق و اجزای
 * حقوق روی آن ساخته می‌شود. اینجا قواعد نمایشی سنجیده می‌شوند؛ قواعد کسب‌وکار در
 * سرور است و آزمون‌هایش در سمت Go اجرا می‌شود.
 */

import {
  actionsFor,
  distinctValues,
  editableIn,
  fieldErrorsFrom,
  searchRows,
  visibleFields,
  type FieldDef,
  type KindDef,
  type RecordRow,
} from '../core/recordHelpers';

function field(key: string, labelFa: string, type = 'text', required = false): FieldDef {
  return { key, labelFa, type, required };
}

function row(id: string, title: string, fields: Record<string, string>, status = 'draft'): RecordRow {
  return { id, kind: 'payParams', title, status, statusFa: status, fields, version: 1 };
}

const kind: KindDef = {
  key: 'payParams',
  titleFa: 'پارامتر و ضریب حقوق',
  idPrefix: 'PPARAM',
  fields: [
    field('key', 'کلید پارامتر', 'text', true),
    field('value', 'مقدار', 'text', true),
    field('unitFa', 'واحد'),
    field('groupFa', 'گروه'),
    field('effectiveFrom', 'تاریخ اجرا', 'date'),
  ],
  statuses: [
    { value: 'draft', labelFa: 'پیش‌نویس' },
    { value: 'active', labelFa: 'جاری' },
    { value: 'inactive', labelFa: 'غیرفعال' },
  ],
  actions: [
    { value: 'activate', labelFa: 'جاری‌سازی', style: 'primary', to: 'active', noteRequired: false },
    { value: 'deactivate', labelFa: 'غیرفعال‌سازی', style: 'ghost', to: 'inactive', noteRequired: false },
    { value: 'reject', labelFa: 'رد با دلیل', style: 'danger', to: 'rejected', noteRequired: true },
  ],
  transitions: { draft: ['active'], active: ['inactive'], inactive: ['active'] },
  initialStatus: 'draft',
  editableStatuses: ['draft', 'active', 'inactive'],
  employeeScoped: false,
};

describe('کمک‌های رکوردهای پویا', () => {
  it('ستون‌های جدول از نخستین فیلدهای شناسه می‌آیند', () => {
    expect(visibleFields(kind).map(f => f.key)).toEqual(['key', 'value', 'unitFa', 'groupFa']);
    expect(visibleFields(kind, 2).map(f => f.key)).toEqual(['key', 'value']);
  });

  it('اقدام‌های نمایشی فقط از وضعیت و جدول انتقال‌ها می‌آیند و نیازمند یادداشت را رد می‌کنند', () => {
    expect(actionsFor(kind, 'draft').map(a => a.value)).toEqual(['activate']);
    expect(actionsFor(kind, 'active').map(a => a.value)).toEqual(['deactivate']);
    expect(actionsFor(kind, 'inactive').map(a => a.value)).toEqual(['activate']);
    // «رد با دلیل» هم به مقصد نامعتبر «rejected» اشاره دارد و هم نیازمند یادداشت است.
    expect(actionsFor(kind, 'draft').find(a => a.value === 'reject')).toBeUndefined();
    expect(actionsFor(kind, 'missing-status')).toEqual([]);
  });

  it('وضعیت مجاز ویرایش از فهرست اعلامی سرور می‌آید و بدون فهرست، وضعیت آغازین است', () => {
    expect(editableIn(kind, 'active')).toBe(true);
    expect(editableIn({ ...kind, editableStatuses: undefined }, 'active')).toBe(false);
    expect(editableIn({ ...kind, editableStatuses: undefined }, 'draft')).toBe(true);
  });

  it('خطاهای سرور بین فیلد فرم و پیام کلی تفکیک می‌شوند', () => {
    const mapped = fieldErrorsFrom({
      'fields.key': 'کلید پارامتر الزامی است.',
      'fields.value': 'مقدار پارامتر با واحد «ریال» باید عددی باشد.',
      title: 'عنوان الزامی است.',
    });
    expect(mapped.fields).toEqual({
      key: 'کلید پارامتر الزامی است.',
      value: 'مقدار پارامتر با واحد «ریال» باید عددی باشد.',
    });
    expect(mapped.form).toBe('عنوان الزامی است.');
  });

  it('جستجو روی عنوان و مقادیر فیلدها کار می‌کند و عبارت خالی همه را برمی‌گرداند', () => {
    const rows = [
      row('PPARAM-0001', 'نرخ بیمهٔ سهم کارمند', { key: 'insurance.employeeRate', value: '0.07' }),
      row('PPARAM-0002', 'حق مسکن ماهانه', { key: 'allowance.housing', value: '9000000' }),
    ];
    expect(searchRows(rows, '')).toHaveLength(2);
    expect(searchRows(rows, 'مسکن').map(r => r.id)).toEqual(['PPARAM-0002']);
    expect(searchRows(rows, 'employeeRate').map(r => r.id)).toEqual(['PPARAM-0001']);
    expect(searchRows(rows, 'چیزی‌که نیست')).toHaveLength(0);
  });

  it('گروه‌بندی مقادیر یکتای، غیرخالی و مرتب‌شده برمی‌گرداند', () => {
    const rows = [
      row('a', '۱', { groupFa: 'بیمه' }),
      row('b', '۲', { groupFa: 'مزایا' }),
      row('c', '۳', { groupFa: 'بیمه' }),
      row('d', '۴', {}),
      row('e', '۵', { groupFa: '  ' }),
    ];
    expect(distinctValues(rows, 'groupFa')).toEqual(['بیمه', 'مزایا']);
  });
});
