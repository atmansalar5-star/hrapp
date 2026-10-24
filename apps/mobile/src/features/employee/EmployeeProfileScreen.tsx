/**
 * پروندهٔ کارمند — یکی از هفت صفحهٔ حداقلی سند طراحی (دستور ۴-۴).
 *
 * اطلاعات در تب‌های گروه‌بندی‌شده نمایش داده می‌شود (دستور ۳-۳): اطلاعات شخصی،
 * اطلاعات شغلی و قرارداد. اگر کاربر کارمند عادی باشد و شناسه نداشته باشد، پروندهٔ
 * خودش نمایش داده می‌شود؛ در هر حال تصمیم دسترسی در سرور گرفته می‌شود.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { ApiError } from '../../api/client';
import type { Employee } from '../../api/contracts';
import { useAuth } from '../../auth/AuthContext';
import { Button, ErrorBanner, Loading, StatusBadge, Toast } from '../../components/ui';
import { faDate, faDigits, faMoney, ltrText, rtlText } from '../../design/rtl';
import { useTheme } from '../../design/theme';
import { useRouteParams } from '../../navigation/RouteContext';
import { fetchEmployee } from './api';

type TabKey = 'personal' | 'job' | 'contract';

const TABS: { key: TabKey; labelFa: string }[] = [
  { key: 'personal', labelFa: 'اطلاعات شخصی' },
  { key: 'job', labelFa: 'اطلاعات شغلی' },
  { key: 'contract', labelFa: 'اطلاعات قرارداد' },
];

export default function EmployeeProfileScreen() {
  const { theme } = useTheme();
  const { client, user } = useAuth();
  const params = useRouteParams();
  const [tab, setTab] = useState<TabKey>('personal');
  const [employee, setEmployee] = useState<Employee | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);
  const [notice, setNotice] = useState('');

  const employeeId = params.employeeId ?? user?.employeeId ?? '';

  const load = useCallback(async () => {
    if (!employeeId) {
      setLoading(false);
      setError('شناسهٔ کارمند مشخص نیست. از فهرست کارکنان یک ردیف را انتخاب کنید.');
      return;
    }
    setLoading(true);
    setError(undefined);
    try {
      setEmployee(await fetchEmployee(client, employeeId));
    } catch (caught) {
      setEmployee(undefined);
      setError(caught instanceof ApiError ? caught.message : 'دریافت پروندهٔ کارمند ناموفق بود.');
    } finally {
      setLoading(false);
    }
  }, [client, employeeId]);

  useEffect(() => {
    void load();
  }, [load]);

  const styles = useMemo(
    () =>
      StyleSheet.create({
        header: {
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radius,
          padding: theme.spacing.lg,
          flexDirection: 'row-reverse',
          alignItems: 'center',
          gap: theme.spacing.lg,
          marginBottom: theme.spacing.md,
        },
        avatar: {
          width: 64,
          height: 64,
          borderRadius: 32,
          backgroundColor: theme.colors.purple,
          alignItems: 'center',
          justifyContent: 'center',
        },
        avatarText: { color: '#FFFFFF', fontWeight: '800', fontSize: 22 },
        identity: { flex: 1, gap: 4 },
        name: { color: theme.colors.text, fontSize: theme.typography.size.xl, fontWeight: '800', ...rtlText },
        meta: { color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText },
        tabs: { flexDirection: 'row-reverse', gap: theme.spacing.sm, marginBottom: theme.spacing.md },
        tab: {
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radiusSm,
          paddingHorizontal: theme.spacing.lg,
          paddingVertical: theme.spacing.sm,
          backgroundColor: theme.colors.surface,
        },
        tabActive: { backgroundColor: theme.colors.primaryLight, borderColor: theme.colors.primary },
        tabText: { color: theme.colors.text, fontSize: theme.typography.size.sm },
        tabTextActive: { color: theme.colors.primaryDark, fontWeight: '700' },
        card: {
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radius,
          padding: theme.spacing.lg,
        },
        fieldRow: {
          flexDirection: 'row-reverse',
          justifyContent: 'space-between',
          paddingVertical: theme.spacing.sm,
          borderBottomWidth: 1,
          borderBottomColor: theme.colors.border,
          gap: theme.spacing.md,
        },
        fieldLabel: { color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText, minWidth: 140 },
        fieldValue: { color: theme.colors.text, fontSize: theme.typography.size.sm, ...rtlText, flex: 1 },
        actions: { flexDirection: 'row-reverse', gap: theme.spacing.md, marginTop: theme.spacing.md },
      }),
    [theme],
  );

  if (loading) {
    return <Loading label="در حال دریافت پرونده…" />;
  }

  if (error || !employee) {
    return (
      <View>
        <ErrorBanner message={error ?? 'پرونده یافت نشد.'} onRetry={load} />
      </View>
    );
  }

  const field = (label: string, value: string, ltr = false) => (
    <View style={styles.fieldRow}>
      <Text style={styles.fieldLabel}>{label}</Text>
      <Text style={[styles.fieldValue, ltr ? ltrText : null]}>{value}</Text>
    </View>
  );

  return (
    <ScrollView>
      <Toast message={notice} />

      <View style={styles.header}>
        <View style={styles.avatar}>
          <Text style={styles.avatarText}>{employee.fullName.charAt(0)}</Text>
        </View>
        <View style={styles.identity}>
          <Text style={styles.name}>{employee.fullName}</Text>
          <Text style={styles.meta}>
            {employee.position} — {employee.departmentName}
          </Text>
          <Text style={styles.meta}>
            شناسه: {faDigits(employee.id)} • نسخهٔ رکورد: {employee.version !== undefined ? faDigits(employee.version) : '—'}
          </Text>
        </View>
        <StatusBadge status={employee.status} />
      </View>

      <View style={styles.tabs}>
        {TABS.map(item => (
          <Pressable
            key={item.key}
            style={[styles.tab, tab === item.key ? styles.tabActive : null]}
            onPress={() => setTab(item.key)}>
            <Text style={[styles.tabText, tab === item.key ? styles.tabTextActive : null]}>{item.labelFa}</Text>
          </Pressable>
        ))}
      </View>

      <View style={styles.card}>
        {tab === 'personal' ? (
          <>
            {field('کد ملی', faDigits(employee.nationalId ?? '—'), true)}
            {field('تلفن همراه', faDigits(employee.phone || '—'), true)}
            {field('رایانامه', employee.email || '—', true)}
            {field('سرپرست', employee.managerName || '—')}
          </>
        ) : null}

        {tab === 'job' ? (
          <>
            {field('واحد سازمانی', employee.departmentName ?? '—')}
            {field('پست', employee.position ?? '—')}
            {field('تاریخ استخدام', employee.hireDate ? faDate(employee.hireDate) : '—')}
            {field('مانده مرخصی', `${employee.leaveBalance !== undefined ? faDigits(employee.leaveBalance) : '—'} روز`)}
          </>
        ) : null}

        {tab === 'contract' ? (
          <>
            {field('نوع قرارداد', employee.contractType ?? '—')}
            {field('حقوق پایه', `${employee.baseSalary !== undefined ? faMoney(employee.baseSalary) : '—'} ریال`)}
            {field('وضعیت', employee.status)}
          </>
        ) : null}
      </View>

      <View style={styles.actions}>
        <Button label="بازآوری پرونده" variant="secondary" onPress={load} />
        <Button
          label="مشاهدهٔ فهرست کارکنان"
          variant="secondary"
          onPress={() => setNotice('برای بازگشت به فهرست، از منوی سمت راست «فهرست کارکنان» را انتخاب کنید.')}
        />
      </View>
    </ScrollView>
  );
}
