/**
 * صفحهٔ پویای رکوردکیت — فهرست، فرم ورود/ویرایش و اقدام‌های گردش‌کارِ هر «نوع رکورد».
 *
 * همه چیز از سرور می‌آید: /schema فیلدها، برچسب‌ها، وضعیت‌ها، جدول انتقال‌ها و
 * وضعیت‌های مجاز ویرایش را می‌دهد و فرم/جدول/دکمه‌ها از همان ساخته می‌شوند؛ پس هیچ
 * قاعده یا برچسبی در رابط تکرار نمی‌شود و تغییر دامنه بی‌درنگ در صفحه دیده می‌شود.
 * مجوزها و قواعد را سرور اعمال می‌کند؛ خطای ۴۰۳/۴۲۲ با پیام فارسی همین‌جا نمایش
 * داده می‌شود. دسترسی‌های دکمه‌ها بر پایهٔ scope نقش فقط پنهان‌نمایی است نه حکم.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { ApiError } from '../api/client';
import { useAuth } from '../auth/AuthContext';
import { useTheme } from '../design/theme';
import { rtlText } from '../design/rtl';
import {
  actionsFor,
  distinctValues,
  editableIn,
  fieldErrorsFrom,
  searchRows,
  visibleFields,
  type ActionDef,
  type KindDef,
  type ModuleSchema,
  type RecordInputBody,
  type RecordPage,
  type RecordRow,
} from '../core/recordHelpers';
import DataTable, { type Column } from './DataTable';
import { Button, EmptyState, ErrorBanner, FormField, Loading, StatusBadge, Toast } from './ui';

interface Props {
  /** پیشوند ماژول (مثل /api/v1/payroll). */
  prefix: string;
  /** کلید نوع رکورد در شناسهٔ ماژول (مثل payParams). */
  kindKey: string;
}

export default function RecordKindScreen({ prefix, kindKey }: Props) {
  const { theme } = useTheme();
  const { client, user, scopes } = useAuth();
  const moduleId = useMemo(() => prefix.split('/').pop() ?? '', [prefix]);

  const [kind, setKind] = useState<KindDef | undefined>(undefined);
  const [rows, setRows] = useState<RecordRow[]>([]);
  const [query, setQuery] = useState('');
  const [group, setGroup] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);
  const [schemaMessage, setSchemaMessage] = useState<string | undefined>(undefined);
  const [toast, setToast] = useState('');
  // undefined = فهرست؛ null = ایجاد تازه؛ RecordRow = ویرایش همان رکورد.
  const [formRow, setFormRow] = useState<RecordRow | null | undefined>(undefined);
  const [saving, setSaving] = useState(false);
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [formMessage, setFormMessage] = useState<string | undefined>(undefined);

  // scope «own» یعنی نقش فقط‌خوانندهٔ همین داده است (مثل کارمند روی پیکربندی حقوق)؛
  // حذف را فقط مدیر سیستم می‌بیند. سرور همچنان مرجع مجوزهاست.
  const canWrite = scopes[moduleId] !== 'own';
  const canDelete = user?.role === 'admin';

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const schema = await client.recordSchema<ModuleSchema>(prefix);
      const found = schema.kinds.find(candidate => candidate.key === kindKey);
      if (!found) {
        setKind(undefined);
        setSchemaMessage(`نوع رکورد «${kindKey}» در شناسهٔ ${schema.moduleTitleFa} اعلام نشده است.`);
        return;
      }
      setSchemaMessage(undefined);
      setKind(found);
      const page = await client.listRecords<RecordPage>(prefix, kindKey, { size: 200 });
      setRows(page.items ?? []);
    } catch (caught) {
      setRows([]);
      setError(caught instanceof ApiError ? caught.message : 'دریافت داده ناموفق بود. اتصال به سرور را بررسی کنید.');
    } finally {
      setLoading(false);
    }
  }, [client, prefix, kindKey]);

  useEffect(() => {
    void load();
  }, [load]);

  const groups = useMemo(() => (kind ? distinctValues(rows, 'groupFa') : []), [kind, rows]);

  const filtered = useMemo(() => {
    const byQuery = searchRows(rows, query);
    if (group === '') {
      return byQuery;
    }
    return byQuery.filter(rowItem => ((rowItem.fields ?? {}).groupFa ?? '') === group);
  }, [rows, query, group]);

  const doAction = useCallback(
    async (rowItem: RecordRow, action: ActionDef) => {
      try {
        await client.actRecord(prefix, rowItem.id, action.value);
        setToast(`اقدام «${action.labelFa}» انجام شد.`);
        await load();
      } catch (caught) {
        setError(caught instanceof ApiError ? caught.message : 'اقدام گردش‌کار ناموفق بود.');
      }
    },
    [client, prefix, load],
  );

  const confirmDelete = useCallback(
    (rowItem: RecordRow) => {
      Alert.alert('حذف رکورد', `«${rowItem.title}» حذف شود؟`, [
        { text: 'انصراف', style: 'cancel' },
        {
          text: 'حذف',
          style: 'destructive',
          onPress: () => {
            void (async () => {
              try {
                await client.deleteRecord(prefix, rowItem.id);
                setToast('رکورد حذف شد.');
                await load();
              } catch (caught) {
                setError(caught instanceof ApiError ? caught.message : 'حذف رکورد ناموفق بود.');
              }
            })();
          },
        },
      ]);
    },
    [client, prefix, load],
  );

  const submitForm = useCallback(
    async (input: RecordInputBody) => {
      setSaving(true);
      setFormErrors({});
      setFormMessage(undefined);
      try {
        if (formRow) {
          await client.updateRecord(prefix, formRow.id, input);
          setToast('تغییرات ذخیره شد.');
        } else {
          await client.createRecord(prefix, input);
          setToast('رکورد ثبت شد.');
        }
        setFormRow(undefined);
        await load();
      } catch (caught) {
        if (caught instanceof ApiError) {
          const mapped = fieldErrorsFrom(caught.details);
          setFormErrors(mapped.fields);
          setFormMessage(mapped.form || caught.message);
        } else {
          setFormMessage('ذخیرهٔ رکورد ناموفق بود. اتصال به سرور را بررسی کنید.');
        }
      } finally {
        setSaving(false);
      }
    },
    [client, prefix, formRow, load],
  );

  const columns: Column<RecordRow>[] = useMemo(() => {
    if (!kind) {
      return [];
    }
    const cellText = (value: string | undefined) => (value ? <Text style={stylesFor(theme).cellText}>{value}</Text> : <Text style={stylesFor(theme).cellMuted}>—</Text>);
    const fieldColumns: Column<RecordRow>[] = visibleFields(kind).map(field => ({
      key: field.key,
      title: field.labelFa,
      flex: 1.2,
      render: rowItem => cellText((rowItem.fields ?? {})[field.key]),
    }));
    return [
      { key: 'title', title: 'عنوان', flex: 1.5, render: rowItem => cellText(rowItem.title) },
      ...fieldColumns,
      { key: 'status', title: 'وضعیت', flex: 0.8, render: rowItem => <StatusBadge status={rowItem.status} /> },
      {
        key: 'actions',
        title: 'عملیات',
        flex: 1.7,
        render: rowItem => (
          <View style={{ flexDirection: 'row-reverse', flexWrap: 'wrap', gap: 6 }}>
            {canWrite
              ? actionsFor(kind, rowItem.status).map(action => (
                  <Button
                    key={action.value}
                    label={action.labelFa}
                    variant={action.style === 'danger' ? 'danger' : 'secondary'}
                    onPress={() => void doAction(rowItem, action)}
                  />
                ))
              : null}
            {canWrite && editableIn(kind, rowItem.status) ? (
              <Button label="ویرایش" variant="secondary" onPress={() => setFormRow(rowItem)} />
            ) : null}
            {canDelete && editableIn(kind, rowItem.status) ? (
              <Button label="حذف" variant="danger" onPress={() => confirmDelete(rowItem)} />
            ) : null}
          </View>
        ),
      },
    ];
  }, [kind, theme, canWrite, canDelete, doAction, confirmDelete]);

  if (formRow !== undefined) {
    if (!kind) {
      return <Loading label="در حال دریافت شناسهٔ ماژول…" />;
    }
    return (
      <RecordFormSheet
        kind={kind}
        row={formRow ?? undefined}
        busy={saving}
        fieldErrors={formErrors}
        errorMessage={formMessage}
        onSubmit={input => void submitForm(input)}
        onCancel={() => setFormRow(undefined)}
      />
    );
  }

  return (
    <ScrollView>
      <Toast message={toast} />
      {error ? <ErrorBanner message={error} onRetry={load} /> : null}
      {schemaMessage ? <EmptyState message={schemaMessage} /> : null}

      <View style={{ flexDirection: 'row-reverse', alignItems: 'center', gap: theme.spacing.md, marginBottom: theme.spacing.md }}>
        <TextInput
          value={query}
          onChangeText={setQuery}
          placeholder="جستجو در عنوان و مقادیر…"
          placeholderTextColor={theme.colors.muted}
          style={{
            flex: 1,
            borderWidth: 1,
            borderColor: theme.colors.border,
            borderRadius: theme.layout.radiusSm,
            paddingHorizontal: theme.spacing.md,
            paddingVertical: theme.spacing.sm,
            backgroundColor: theme.colors.surface,
            color: theme.colors.text,
            textAlign: 'right',
          }}
        />
        {canWrite && kind ? <Button label={`افزودن ${kind.titleFa}`} onPress={() => setFormRow(null)} /> : null}
      </View>

      {groups.length > 0 ? (
        <View style={{ flexDirection: 'row-reverse', flexWrap: 'wrap', gap: theme.spacing.sm, marginBottom: theme.spacing.md }}>
          <Button label="همه" variant={group === '' ? 'primary' : 'secondary'} onPress={() => setGroup('')} />
          {groups.map(value => (
            <Button key={value} label={value} variant={group === value ? 'primary' : 'secondary'} onPress={() => setGroup(value)} />
          ))}
        </View>
      ) : null}

      {loading && rows.length === 0 ? (
        <Loading label="در حال دریافت فهرست…" />
      ) : !kind && !schemaMessage ? (
        <Loading label="در حال دریافت شناسهٔ ماژول…" />
      ) : filtered.length === 0 && !error ? (
        <EmptyState message="با این جستجو/گروه رکوردی یافت نشد." />
      ) : (
        <DataTable
          columns={columns}
          rows={filtered}
          rowKey={rowItem => rowItem.id}
          total={filtered.length}
          page={1}
          size={Math.max(1, filtered.length)}
          onRowPress={rowItem => {
            if (canWrite && editableIn(kind as KindDef, rowItem.status)) {
              setFormRow(rowItem);
            }
          }}
        />
      )}

      <Text
        style={{
          color: theme.colors.muted,
          fontSize: theme.typography.size.xs,
          ...rtlText,
          marginTop: theme.spacing.md,
        }}>
        مجوزها و قواعد در سرور اعمال می‌شود؛ پارامتر جاری پس از «جاری‌سازی» در محاسبهٔ حقوق مؤثر است و پیش‌نویس اثری ندارد.
      </Text>
    </ScrollView>
  );
}

/** فرم ایجاد/ویرایش رکورد — فیلدها و برچسب‌ها از شناسهٔ سرور ساخته می‌شوند. */
function RecordFormSheet({
  kind,
  row,
  busy,
  fieldErrors,
  errorMessage,
  onSubmit,
  onCancel,
}: {
  kind: KindDef;
  row?: RecordRow;
  busy: boolean;
  fieldErrors: Record<string, string>;
  errorMessage?: string;
  onSubmit: (input: RecordInputBody) => void;
  onCancel: () => void;
}) {
  const { theme } = useTheme();
  const [title, setTitle] = useState(row?.title ?? '');
  const [employeeId, setEmployeeId] = useState(row?.employeeId ?? '');
  const [values, setValues] = useState<Record<string, string>>(() => ({ ...(row?.fields ?? {}) }));
  const [localErrors, setLocalErrors] = useState<Record<string, string>>({});

  const submit = () => {
    const errors: Record<string, string> = {};
    // بازتاب همان قاعدهٔ الزامیِ سرور برای تجربهٔ فوری؛ مرجع نهایی سرور است.
    if (title.trim() === '') {
      errors.title = 'عنوان الزامی است.';
    }
    for (const field of kind.fields) {
      if (field.required && (values[field.key] ?? '').trim() === '') {
        errors[field.key] = `«${field.labelFa}» الزامی است.`;
      }
    }
    setLocalErrors(errors);
    if (Object.keys(errors).length > 0) {
      return;
    }
    const trimmed: Record<string, string> = {};
    for (const [key, value] of Object.entries(values)) {
      trimmed[key] = value.trim();
    }
    onSubmit({
      kind: kind.key,
      title: title.trim(),
      ...(kind.employeeScoped ? { employeeId: employeeId.trim() } : {}),
      fields: trimmed,
    });
  };

  const merged = { ...localErrors, ...fieldErrors };
  const card = {
    backgroundColor: theme.colors.surface,
    borderWidth: 1,
    borderColor: theme.colors.border,
    borderRadius: theme.layout.radius,
    padding: theme.spacing.lg,
    marginBottom: theme.spacing.lg,
  };

  return (
    <ScrollView keyboardShouldPersistTaps="handled">
      {errorMessage ? <ErrorBanner message={errorMessage} /> : null}
      <View style={card}>
        <Text style={{ color: theme.colors.text, fontWeight: '800', marginBottom: theme.spacing.md, ...rtlText }}>
          {row ? `ویرایش «${row.title}»` : `رکورد تازه — ${kind.titleFa}`}
        </Text>
        <FormField label="عنوان" value={title} onChangeText={setTitle} required error={merged.title} />
        {kind.employeeScoped ? (
          <FormField
            label="شناسهٔ کارمند"
            value={employeeId}
            onChangeText={setEmployeeId}
            required
            error={merged.employeeId}
            placeholder="EMP-001"
          />
        ) : null}
        {kind.fields.map(field => (
          <FormField
            key={field.key}
            label={field.labelFa}
            value={values[field.key] ?? ''}
            onChangeText={text => setValues(previous => ({ ...previous, [field.key]: text }))}
            required={field.required}
            numeric={field.type === 'number' || field.type === 'money'}
            placeholder={field.type === 'date' ? '۱۴۰۴/۰۷/۰۱' : undefined}
            error={merged[field.key]}
          />
        ))}
        <View style={{ flexDirection: 'row-reverse', gap: theme.spacing.md, marginTop: theme.spacing.md }}>
          <Button label={row ? 'ذخیره تغییرات' : 'ثبت رکورد'} onPress={submit} loading={busy} />
          <Button label="انصراف" variant="secondary" onPress={onCancel} />
        </View>
      </View>
    </ScrollView>
  );
}

/** سبک سلول‌ها؛ جداگانه تا در `useMemo` ستون‌ها هم قابل استفاده باشد. */
function stylesFor(theme: ReturnType<typeof useTheme>['theme']) {
  return StyleSheet.create({
    cellText: { color: theme.colors.text, textAlign: 'right', fontSize: theme.typography.size.sm },
    cellMuted: { color: theme.colors.muted, textAlign: 'right', fontSize: theme.typography.size.sm },
  });
}
