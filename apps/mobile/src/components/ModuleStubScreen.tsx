/**
 * صفحهٔ موقت ماژول‌های فاز بعد.
 *
 * ماژول‌های حضور، حقوق، عملکرد، آموزش، رفاه، گزارش‌ها و بخشی از تنظیمات در فاز ۳
 * کامل می‌شوند. تا آن زمان این صفحه توضیح می‌دهد که صفحه چه کاری خواهد کرد و کدام
 * بخش پشت صحنه آماده است — به‌جای نمایش دادهٔ ساختگی که به‌اشتباه واقعی به نظر برسد.
 */

import { ScrollView, StyleSheet, Text, View } from 'react-native';

import { useTheme } from '../design/theme';
import { rtlText } from '../design/rtl';
import { EmptyState } from '../components/ui';
import { resolveScreen } from '../navigation/screenRegistry';
import { MODULES } from '../navigation/modules.generated';

/** صفحه‌ای که از مانیفست ورودی می‌گیرد و پیام «فاز بعد» را نشان می‌دهد. */
export function ModuleStubScreen({ component }: { component: string }) {
  const { theme } = useTheme();

  const screen = MODULES.flatMap(module =>
    module.screens.map(s => ({ moduleId: module.id, moduleTitle: module.titleFa, screen: s })),
  ).find(entry => entry.screen.component === component);

  const styles = StyleSheet.create({
    wrapper: { flex: 1, gap: theme.spacing.md },
    title: { color: theme.colors.text, fontSize: theme.typography.size.xl, fontWeight: '800', ...rtlText },
    body: { color: theme.colors.text, ...rtlText, lineHeight: theme.typography.size.md * theme.typography.lineHeightRatio },
    meta: { color: theme.colors.muted, fontSize: theme.typography.size.xs, ...rtlText },
    card: {
      backgroundColor: theme.colors.surface,
      borderWidth: 1,
      borderColor: theme.colors.border,
      borderRadius: theme.layout.radius,
      padding: theme.spacing.lg,
      gap: theme.spacing.sm,
    },
  });

  if (!screen) {
    return <EmptyState message={`صفحهٔ «${component}» در مانیفست ماژول‌ها ثبت نشده است.`} />;
  }

  return (
    <ScrollView style={styles.wrapper} contentContainerStyle={{ gap: theme.spacing.md }}>
      <Text style={styles.title}>{screen.screen.titleFa}</Text>
      <View style={styles.card}>
        <Text style={styles.body}>{screen.screen.purposeFa}</Text>
        <Text style={styles.meta}>
          کلید ناوبری: {screen.moduleId}.{screen.screen.key} — ماژول: {screen.moduleTitle}
        </Text>
      </View>
      <View style={styles.card}>
        <Text style={styles.body}>وضعیت پیاده‌سازی</Text>
        <Text style={styles.meta}>
          رابط این صفحه در فاز ۳ تکمیل می‌شود. رجیستری ماژول، دسترسی نقش‌محور، قرارداد API و هستهٔ قواعد از همین
          حالا آماده است، پس افزودن این صفحه به معنی نوشتن لایهٔ نمایش روی همین پایه است.
        </Text>
      </View>
    </ScrollView>
  );
}

/** ساخت یک صفحه با نام مشخص؛ برای همهٔ صفحه‌های فازهای بعد استفاده می‌شود. */
export function makeStubScreen(component: string) {
  return function StubScreen() {
    return <ModuleStubScreen component={component} />;
  };
}

/** بررسی اینکه کامپوننت در رجیستری وجود دارد (برای آزمون ناهم‌خوانی مانیفست و کد). */
export function isRegistered(component: string): boolean {
  return resolveScreen(component) !== undefined;
}
