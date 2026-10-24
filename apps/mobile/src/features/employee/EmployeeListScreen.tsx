/**
 * فهرست کارکنان — یکی از هفت صفحهٔ حداقلی سند طراحی (دستور ۴-۴).
 *
 * تعاملات خواسته‌شده در دستور ۴-۳:
 * - جستجو و فیلتر → به‌روزرسانی جدول
 * - کلیک روی ردیف → باز شدن پرونده
 * - کلیک روی «ایجاد» → باز شدن فرم
 * - تأیید فرم → بازگشت به جدول با پیام موفقیت
 *
 * محدودهٔ دسترسی نقش در سرور اعمال می‌شود؛ مدیر واحد تنها کارکنان واحد خود را
 * دریافت می‌کند و کلاینت فیلتر اضافه‌ای اعمال نمی‌کند.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { ApiError } from '../../api/client';
import type { Department, Employee, EmployeeInput } from '../../api/contracts';
import { useAuth } from '../../auth/AuthContext';
import DataTable, { type Column } from '../../components/DataTable';
import { Button, EmptyState, ErrorBanner, Loading, StatusBadge, Toast } from '../../components/ui';
import { faMoney, rtlText } from '../../design/rtl';
import { useTheme } from '../../design/theme';
import { useRoute } from '../../navigation/RouteContext';
import { createEmployee, deleteEmployee, fetchDepartments, fetchEmployees, updateEmployee } from './api';
import EmployeeFormSheet from './EmployeeFormSheet';

type Mode = { kind: 'list' } | { kind: 'create' } | { kind: 'edit'; employee: Employee };

export default function EmployeeListScreen() {
  const { theme } = useTheme();
  const { client, user, scopes } = useAuth();
  const { openEmployee } = useRoute();

  const [rows, setRows] = useState<Employee[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [size] = useState(20);
  const [sort, setSort] = useState('id:asc');
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [contractFilter, setContractFilter] = useState('');
  const [departmentFilter, setDepartmentFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);
  const [toast, setToast] = useState('');
  const [mode, setMode] = useState<Mode>({ kind: 'list' });
  const [saving, setSaving] = useState(false);
  const [formMessage, setFormMessage] = useState<string | undefined>(undefined);

  const canWrite = scopes.employee === 'all';
  const canDelete = user?.role === 'admin' || user?.role === 'hr_manager';

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const pageResult = await fetchEmployees(client, {
        query,
        departmentId: departmentFilter,
        status: statusFilter,
        contractType: contractFilter,
        sort,
        page,
        size,
      });
      setRows(pageResult.items ?? []);
      setTotal(pageResult.total ?? 0);
    } catch (caught) {
      if (caught instanceof ApiError && caught.isForbidden) {
        setError(caught.message);
      } else {
        setError('دریافت فهرست کارکنان ناموفق بود. اتصال به سرور را بررسی کنید.');
      }
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [client, query, departmentFilter, statusFilter, contractFilter, sort, page, size]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    fetchDepartments(client)
      .then(setDepartments)
      .catch(() => setDepartments([]));
  }, [client]);

  const columns: Column<Employee>[] = useMemo(
    () => [
      { key: 'id', title: 'شناسه', sortable: true, flex: 0.7, render: row => <Text style={stylesFor(theme).cellText}>{row.id}</Text> },
      { key: 'fullName', title: 'نام و نام خانوادگی', sortable: true, flex: 1.6, render: row => <Text style={stylesFor(theme).cellText}>{row.fullName}</Text> },
      { key: 'departmentName', title: 'واحد', sortable: true, flex: 1.4, render: row => <Text style={stylesFor(theme).cellText}>{row.departmentName}</Text> },
      { key: 'position', title: 'پست', flex: 1.4, render: row => <Text style={stylesFor(theme).cellText}>{row.position}</Text> },
      { key: 'status', title: 'وضعیت', sortable: true, flex: 0.8, render: row => <StatusBadge status={row.status} /> },
      { key: 'baseSalary', title: 'حقوق پایه', sortable: true, flex: 1.2, render: row => <Text style={stylesFor(theme).cellText}>{faMoney(row.baseSalary ?? 0)}</Text> },
      {
        key: 'actions',
        title: 'عملیات',
        flex: 1.1,
        render: row => (
          <View style={{ flexDirection: 'row-reverse', gap: 6 }}>
            <Button label="پرونده" variant="secondary" onPress={() => openEmployee(row.id)} />
            {canDelete ? (
              <Button
                label="حذف"
                variant="danger"
                onPress={() => {
                  void (async () => {
                    const result = await deleteEmployee(client, row.id);
                    setToast(result.message ?? 'انجام شد.');
                    await load();
                  })();
                }}
              />
            ) : null}
          </View>
        ),
      },
    ],
    [theme, canDelete, client, load, openEmployee],
  );

  const submitForm = useCallback(
    async (input: EmployeeInput) => {
      setSaving(true);
      setFormMessage(undefined);
      const result =
        mode.kind === 'edit' ? await updateEmployee(client, mode.employee.id, input) : await createEmployee(client, input);
      setSaving(false);

      if (result.errors && Object.keys(result.errors).length > 0) {
        setFormMessage(`${result.message ?? 'خطای اعتبارسنجی'} — ${Object.values(result.errors).join(' ')}`);
        return;
      }
      if (result.employee || result.queued) {
        setToast(result.message ?? (mode.kind === 'edit' ? 'تغییرات ذخیره شد.' : 'کارمند با موفقیت ثبت شد.'));
        setMode({ kind: 'list' });
        await load();
      }
    },
    [client, load, mode],
  );

  if (mode.kind !== 'list') {
    return (
      <EmployeeFormSheet
        departments={departments}
        employee={mode.kind === 'edit' ? mode.employee : undefined}
        busy={saving}
        errorMessage={formMessage}
        onSubmit={input => void submitForm(input)}
        onCancel={() => setMode({ kind: 'list' })}
      />
    );
  }

  const activeFilters = [
    statusFilter ? `وضعیت: ${statusFilter}` : '',
    contractFilter ? `قرارداد: ${contractFilter}` : '',
    departmentFilter ? `واحد: ${departments.find(d => d.id === departmentFilter)?.name ?? departmentFilter}` : '',
  ].filter(Boolean);

  return (
    <ScrollView>
      <Toast message={toast} />
      {error ? <ErrorBanner message={error} onRetry={load} /> : null}

      {user?.role === 'line_manager' ? (
        <Text style={{ color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText, marginBottom: theme.spacing.sm }}>
          شما فقط کارکنان واحد خودتان را می‌بینید (محدودهٔ دادهٔ نقش: واحد).
        </Text>
      ) : null}

      <View style={{ flexDirection: 'row-reverse', alignItems: 'center', gap: theme.spacing.md, marginBottom: theme.spacing.md }}>
        <TextInput
          value={query}
          onChangeText={text => {
            setPage(1);
            setQuery(text);
          }}
          placeholder="جستجوی نام، شناسه، کد ملی یا پست…"
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
        {canWrite ? <Button label="ایجاد کارمند" onPress={() => setMode({ kind: 'create' })} /> : null}
      </View>

      <View style={{ flexDirection: 'row-reverse', flexWrap: 'wrap', gap: theme.spacing.sm, marginBottom: theme.spacing.md }}>
        {[
          { key: '', labelFa: 'همه وضعیت‌ها' },
          { key: 'active', labelFa: 'شاغل' },
          { key: 'leave', labelFa: 'مرخصی' },
          { key: 'probation', labelFa: 'آزمایشی' },
          { key: 'terminated', labelFa: 'ترک خدمت' },
        ].map(option => (
          <Button
            key={`status-${option.key}`}
            label={option.labelFa}
            variant={statusFilter === option.key ? 'primary' : 'secondary'}
            onPress={() => {
              setPage(1);
              setStatusFilter(option.key);
            }}
          />
        ))}
        {[
          { key: '', labelFa: 'همه انواع قرارداد' },
          { key: 'تمام‌وقت', labelFa: 'تمام‌وقت' },
          { key: 'پیمانکاری', labelFa: 'پیمانکاری' },
          { key: 'پاره‌وقت', labelFa: 'پاره‌وقت' },
        ].map(option => (
          <Button
            key={`contract-${option.key}`}
            label={option.labelFa}
            variant={contractFilter === option.key ? 'primary' : 'secondary'}
            onPress={() => {
              setPage(1);
              setContractFilter(option.key);
            }}
          />
        ))}
      </View>

      {departments.length > 0 ? (
        <View style={{ flexDirection: 'row-reverse', flexWrap: 'wrap', gap: theme.spacing.sm, marginBottom: theme.spacing.md }}>
          <Button
            label="همه واحدها"
            variant={departmentFilter === '' ? 'primary' : 'secondary'}
            onPress={() => {
              setPage(1);
              setDepartmentFilter('');
            }}
          />
          {departments.map(department => (
            <Button
              key={department.id}
              label={department.name}
              variant={departmentFilter === department.id ? 'primary' : 'secondary'}
              onPress={() => {
                setPage(1);
                setDepartmentFilter(department.id);
              }}
            />
          ))}
        </View>
      ) : null}

      {activeFilters.length > 0 ? (
        <Text style={{ color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText, marginBottom: theme.spacing.sm }}>
          فیلترهای فعال: {activeFilters.join(' • ')}
        </Text>
      ) : null}

      {loading && rows.length === 0 ? (
        <Loading label="در حال دریافت فهرست کارکنان…" />
      ) : rows.length === 0 && !error ? (
        <EmptyState message="با این فیلترها رکوردی یافت نشد." />
      ) : (
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={row => row.id}
          total={total}
          page={page}
          size={size}
          sort={sort}
          onSortChange={next => {
            setPage(1);
            setSort(next);
          }}
          onPageChange={setPage}
          onRowPress={row => openEmployee(row.id)}
        />
      )}
    </ScrollView>
  );
}

/** سبک‌های سلول؛ جدا شده تا در `useMemo` ستون‌ها هم قابل استفاده باشد. */
function stylesFor(theme: ReturnType<typeof useTheme>['theme']) {
  return StyleSheet.create({
    cellText: { color: theme.colors.text, textAlign: 'right', fontSize: theme.typography.size.sm },
  });
}
