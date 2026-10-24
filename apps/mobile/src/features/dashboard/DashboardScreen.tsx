/**
 * داشبورد نقش‌محور — دستور ۳-۴ سند طراحی.
 *
 * چیدمان کارتی، هر کارت شامل عنوان، مقدار عددی، آیکون و لینک به جزئیات؛ همهٔ کارت‌ها از
 * سرور می‌آید و بر پایهٔ نقش و محدودهٔ دادهٔ کاربر ساخته شده‌اند. کارت‌ها برای ماژول‌های
 * پیاده‌نشده نمایش داده نمی‌شوند تا عدد ساختگی نشان داده نشود.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';

import { useAuth } from '../../auth/AuthContext';
import { useTheme } from '../../design/theme';
import { rtlText } from '../../design/rtl';
import { EmptyState, ErrorBanner, Loading, StatCard } from '../../components/ui';
import { useRoute } from '../../navigation/RouteContext';
import type { Widget } from '../../api/contracts';

/** نگاشت کلید کارت به آیکون. */
const ICONS: Record<string, string> = {
  headcount: '👥',
  active: '✅',
  leave: '🌴',
  probation: '🧪',
  terminated: '📤',
  own_record: '🪪',
};

export default function DashboardScreen() {
  const { theme } = useTheme();
  const { client, user, roleFa, scopes } = useAuth();
  const { navigate } = useRoute();
  const [widgets, setWidgets] = useState<Widget[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | undefined>(undefined);

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      setWidgets(await client.widgets());
    } catch (caught) {
      setWidgets([]);
      setError('دریافت کارت‌های آماری ناموفق بود.');
      void caught;
    } finally {
      setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    void load();
  }, [load]);

  const styles = useMemo(
    () =>
      StyleSheet.create({
        wrapper: { flex: 1 },
        greeting: { marginBottom: theme.spacing.md },
        greetingTitle: { color: theme.colors.text, fontSize: theme.typography.size.xl, fontWeight: '800', ...rtlText },
        greetingMeta: { color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText },
        grid: { flexDirection: 'row-reverse', flexWrap: 'wrap', gap: theme.spacing.md },
        section: {
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radius,
          padding: theme.spacing.lg,
          marginTop: theme.spacing.md,
          gap: theme.spacing.sm,
        },
        sectionTitle: { color: theme.colors.text, fontWeight: '800', ...rtlText },
        sectionBody: { color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText },
      }),
    [theme],
  );

  // میان‌برهای اقدام سریع بر پایهٔ دامنهٔ دادهٔ کاربر (سند طراحی: «اقدامات سریع»).
  const quickActions = useMemo(() => {
    const actions: { labelFa: string; moduleId: string; screenKey: string }[] = [
      { labelFa: 'فهرست کارکنان', moduleId: 'employee', screenKey: 'employees' },
      { labelFa: 'پروندهٔ من', moduleId: 'employee', screenKey: 'profile' },
      { labelFa: 'ساختار سازمانی', moduleId: 'employee', screenKey: 'org' },
    ];
    return actions;
  }, []);

  return (
    <ScrollView
      style={styles.wrapper}
      refreshControl={<RefreshControl refreshing={loading} onRefresh={load} />}
      contentContainerStyle={{ paddingBottom: theme.spacing.xl }}>
      <View style={styles.greeting}>
        <Text style={styles.greetingTitle}>سلام {user?.displayName ?? 'کاربر'} 👋</Text>
        <Text style={styles.greetingMeta}>
          نقش شما: {roleFa} — محدودهٔ دادهٔ کارکنان: {scopes.employee === 'own_unit' ? 'واحد خودتان' : scopes.employee === 'own' ? 'فقط پروندهٔ خودتان' : 'کل سازمان'}
        </Text>
      </View>

      {error ? <ErrorBanner message={error} onRetry={load} /> : null}
      {loading && widgets.length === 0 ? <Loading label="در حال دریافت شاخص‌ها…" /> : null}

      {!loading && widgets.length === 0 && !error ? (
        <EmptyState message="برای نقش شما شاخص آماری تعریف نشده است." />
      ) : (
        <View style={styles.grid}>
          {widgets.map(widget => (
            <StatCard
              key={widget.key}
              title={widget.titleFa}
              value={widget.value}
              unit={widget.unit}
              trend={widget.trend}
              icon={ICONS[widget.key]}
              onPress={() => navigate({ moduleId: 'employee', screenKey: 'employees' })}
            />
          ))}
        </View>
      )}

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>اقدام‌های سریع</Text>
        <Text style={styles.sectionBody}>
          {quickActions.map(action => action.labelFa).join(' • ')}
        </Text>
        <Text style={styles.sectionBody}>
          ماژول‌های در دسترس شما همان چیزی است که در منوی سمت راست می‌بینید؛ آیتم‌های غیرمجاز از منو حذف شده‌اند
          (دستور ۳-۱) و درخواست دستی به مسیر غیرمجاز هم در سرور با خطای دسترسی رد می‌شود.
        </Text>
      </View>
    </ScrollView>
  );
}
