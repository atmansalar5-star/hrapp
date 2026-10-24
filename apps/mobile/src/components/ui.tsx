/**
 * اجزای کوچک مشترک رابط: دکمه، کارت آماری، برچسب وضعیت، فیلد فرم و حالت‌های خالی.
 * همه رنگ‌ها از تم می‌آید و متن‌ها راست‌چین است (دستور ۵-۳ سند طراحی).
 */

import { useMemo, type ReactNode } from 'react';
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';

import { useTheme } from '../design/theme';
import { faDigits, faMoney, normalizeDigits, rtlText } from '../design/rtl';
import { statusColors } from '../design/tokens';

/** دکمهٔ اصلی/ثانویه/خطر. */
export function Button({
  label,
  onPress,
  variant = 'primary',
  disabled = false,
  loading = false,
}: {
  label: string;
  onPress?: () => void;
  variant?: 'primary' | 'secondary' | 'danger';
  disabled?: boolean;
  loading?: boolean;
}) {
  const { theme } = useTheme();
  const background =
    variant === 'primary' ? theme.colors.primary : variant === 'danger' ? theme.colors.red : theme.colors.surface;
  const color = variant === 'secondary' ? theme.colors.text : '#FFFFFF';

  return (
    <Pressable
      onPress={onPress}
      disabled={disabled || loading}
      accessibilityRole="button"
      style={{
        backgroundColor: background,
        borderWidth: variant === 'secondary' ? 1 : 0,
        borderColor: theme.colors.border,
        borderRadius: theme.layout.radiusSm,
        paddingHorizontal: theme.spacing.lg,
        paddingVertical: theme.spacing.sm,
        flexDirection: 'row-reverse',
        alignItems: 'center',
        gap: theme.spacing.sm,
        opacity: disabled ? 0.5 : 1,
      }}>
      {loading ? <ActivityIndicator size="small" color={color} /> : null}
      <Text style={{ color, fontWeight: '700' }}>{label}</Text>
    </Pressable>
  );
}

/** کارت آماری داشبورد — دستور ۳-۴: عنوان، مقدار عددی، آیکون و لینک جزئیات. */
export function StatCard({
  title,
  value,
  unit,
  icon,
  trend,
  onPress,
}: {
  title: string;
  value: number;
  unit?: string;
  icon?: string;
  trend?: number;
  onPress?: () => void;
}) {
  const { theme } = useTheme();
  const styles = useMemo(
    () =>
      StyleSheet.create({
        card: {
          flex: 1,
          minWidth: 160,
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radius,
          padding: theme.spacing.lg,
          gap: theme.spacing.sm,
        },
        head: { flexDirection: 'row-reverse', justifyContent: 'space-between', alignItems: 'center' },
        title: { color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText },
        icon: { fontSize: 18 },
        value: { color: theme.colors.text, fontSize: theme.typography.size.xxl, fontWeight: '800', ...rtlText },
        unit: { color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText },
        trend: { fontSize: theme.typography.size.xxs, fontWeight: '700' },
      }),
    [theme],
  );

  return (
    <Pressable style={styles.card} onPress={onPress} accessibilityRole={onPress ? 'button' : undefined}>
      <View style={styles.head}>
        <Text style={styles.title}>{title}</Text>
        <Text style={styles.icon}>{icon ?? '📊'}</Text>
      </View>
      <Text style={styles.value}>{faDigits(value.toLocaleString('en-US'))}</Text>
      {unit ? <Text style={styles.unit}>{unit}</Text> : null}
      {trend !== undefined ? (
        <Text style={[styles.trend, { color: trend >= 0 ? theme.colors.green : theme.colors.red }]}>
          {trend >= 0 ? '▲' : '▼'} {faDigits(Math.abs(trend))}٪
        </Text>
      ) : null}
    </Pressable>
  );
}

/** برچسب وضعیت کارمند (شاغل، مرخصی، آزمایشی، ترک خدمت). */
export function StatusBadge({ status }: { status: string }) {
  const { theme } = useTheme();
  const meta = statusColors[status] ?? { bg: theme.colors.bg, fg: theme.colors.muted, labelFa: status };
  return (
    <View style={{ backgroundColor: meta.bg, borderRadius: 999, paddingHorizontal: 10, paddingVertical: 2, alignSelf: 'flex-end' }}>
      <Text style={{ color: meta.fg, fontSize: theme.typography.size.xxs, fontWeight: '700' }}>{meta.labelFa}</Text>
    </View>
  );
}

/** فیلد فرم با ستارهٔ الزامی و پیام خطای کنار فیلد (دستور ۳-۳). */
export function FormField({
  label,
  value,
  onChangeText,
  error,
  required = false,
  placeholder,
  numeric = false,
  multiline = false,
}: {
  label: string;
  value: string;
  onChangeText: (text: string) => void;
  error?: string;
  required?: boolean;
  placeholder?: string;
  numeric?: boolean;
  multiline?: boolean;
}) {
  const { theme } = useTheme();

  return (
    <View style={{ marginBottom: theme.spacing.md }}>
      <Text style={{ color: theme.colors.text, fontWeight: '700', marginBottom: 4, ...rtlText }}>
        {label} {required ? <Text style={{ color: theme.colors.red }}>*</Text> : null}
      </Text>
      <TextInput
        value={value}
        onChangeText={text => onChangeText(numeric ? normalizeDigits(text) : text)}
        placeholder={placeholder}
        placeholderTextColor={theme.colors.muted}
        keyboardType={numeric ? 'numeric' : 'default'}
        multiline={multiline}
        style={{
          borderWidth: 1,
          borderColor: error ? theme.colors.red : theme.colors.border,
          borderRadius: theme.layout.radiusSm,
          paddingHorizontal: theme.spacing.md,
          paddingVertical: theme.spacing.sm,
          color: theme.colors.text,
          backgroundColor: theme.colors.surface,
          textAlign: 'right',
        }}
      />
      {error ? <Text style={{ color: theme.colors.red, fontSize: theme.typography.size.xxs, ...rtlText }}>{error}</Text> : null}
    </View>
  );
}

/** پیام خطای سراسری صفحه. */
export function ErrorBanner({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { theme } = useTheme();
  return (
    <View
      style={{
        backgroundColor: '#FEE2E2',
        borderRadius: theme.layout.radiusSm,
        padding: theme.spacing.md,
        marginBottom: theme.spacing.md,
        flexDirection: 'row-reverse',
        alignItems: 'center',
        justifyContent: 'space-between',
        gap: theme.spacing.md,
      }}>
      <Text style={{ color: '#991B1B', flex: 1, ...rtlText }}>{message}</Text>
      {onRetry ? <Button label="تلاش دوباره" variant="secondary" onPress={onRetry} /> : null}
    </View>
  );
}

/** پیام موفقیت (دستور ۴-۳: پس از تأیید فرم، پیام موفقیت نمایش داده شود). */
export function Toast({ message }: { message: string }) {
  const { theme } = useTheme();
  if (!message) {
    return null;
  }
  return (
    <View
      style={{
        backgroundColor: '#DCFCE7',
        borderRadius: theme.layout.radiusSm,
        padding: theme.spacing.md,
        marginBottom: theme.spacing.md,
      }}>
      <Text style={{ color: '#166534', ...rtlText }}>{message}</Text>
    </View>
  );
}

/** حالت بارگذاری. */
export function Loading({ label = 'در حال بارگذاری…' }: { label?: string }) {
  const { theme } = useTheme();
  return (
    <View style={{ padding: theme.spacing.xl, alignItems: 'center', gap: theme.spacing.md }}>
      <ActivityIndicator color={theme.colors.primary} />
      <Text style={{ color: theme.colors.muted, ...rtlText }}>{label}</Text>
    </View>
  );
}

/** کارت خالی با پیام. */
export function EmptyState({ message }: { message: string }) {
  const { theme } = useTheme();
  return (
    <View style={{ backgroundColor: theme.colors.surface, borderWidth: 1, borderColor: theme.colors.border, borderRadius: theme.layout.radius, padding: theme.spacing.xl, alignItems: 'center' }}>
      <Text style={{ color: theme.colors.muted, ...rtlText }}>{message}</Text>
    </View>
  );
}

/** نمایش مبلغ ریالی با ارقام فارسی. */
export function Money({ amount }: { amount: number }) {
  const { theme } = useTheme();
  return <Text style={{ color: theme.colors.text, ...rtlText }}>{faMoney(amount)} ریال</Text>;
}
