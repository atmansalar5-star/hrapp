/**
 * صفحهٔ ورود — یکی از هفت صفحهٔ حداقلی سند طراحی (دستور ۴-۴).
 *
 * نکات: راست‌چین کامل، نمایش حساب‌های نمونه برای بازبینی طراحی، پیام خطای فارسی از
 * سرور، و یادآوری این‌که رمزها هرگز روی دستگاه ذخیره نمی‌شوند.
 */

import { useState } from 'react';
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';

import { useAuth } from '../../auth/AuthContext';
import { useTheme } from '../../design/theme';
import { rtlText } from '../../design/rtl';
import { Button, ErrorBanner, FormField } from '../../components/ui';

/** حساب‌های نمونه — همان کاربران seed سرور (رمز همه: demo). */
const DEMO_ACCOUNTS: { username: string; labelFa: string }[] = [
  { username: 'hr.manager', labelFa: 'مدیر منابع انسانی — سارا محمدی' },
  { username: 'hr.spec', labelFa: 'کارشناس منابع انسانی — مریم نوری' },
  { username: 'line.mgr', labelFa: 'مدیر واحد فناوری اطلاعات — آرش رضایی' },
  { username: 'employee', labelFa: 'کارمند — فاطمه صادقی' },
  { username: 'admin', labelFa: 'مدیر سیستم' },
];

export default function LoginScreen() {
  const { theme } = useTheme();
  const { login, busy, error } = useAuth();
  const [username, setUsername] = useState('hr.manager');
  const [password, setPassword] = useState('demo');
  const [touched, setTouched] = useState(false);

  const styles = StyleSheet.create({
    root: { flex: 1, backgroundColor: theme.colors.bg, alignItems: 'center', justifyContent: 'center' },
    card: {
      width: '100%',
      maxWidth: 460,
      backgroundColor: theme.colors.surface,
      borderRadius: theme.layout.radius,
      borderWidth: 1,
      borderColor: theme.colors.border,
      padding: theme.spacing.xl,
      gap: theme.spacing.md,
    },
    header: { alignItems: 'center', gap: theme.spacing.sm, marginBottom: theme.spacing.md },
    logo: {
      width: 56,
      height: 56,
      borderRadius: 16,
      backgroundColor: theme.colors.primary,
      alignItems: 'center',
      justifyContent: 'center',
    },
    logoText: { color: '#FFFFFF', fontWeight: '800', fontSize: 20 },
    title: { color: theme.colors.text, fontSize: theme.typography.size.xl, fontWeight: '800', ...rtlText },
    subtitle: { color: theme.colors.muted, fontSize: theme.typography.size.sm, ...rtlText, textAlign: 'center' },
    demoBox: {
      backgroundColor: theme.colors.bg,
      borderRadius: theme.layout.radiusSm,
      padding: theme.spacing.md,
      gap: 4,
    },
    demoTitle: { color: theme.colors.muted, fontSize: theme.typography.size.xs, fontWeight: '700', ...rtlText },
    demoRow: { paddingVertical: 6 },
    demoText: { color: theme.colors.text, fontSize: theme.typography.size.xs, ...rtlText },
    hint: { color: theme.colors.muted, fontSize: theme.typography.size.xxs, ...rtlText },
  });

  const usernameError = touched && username.trim().length === 0 ? 'نام کاربری الزامی است.' : undefined;
  const passwordError = touched && password.length === 0 ? 'رمز عبور الزامی است.' : undefined;

  const submit = () => {
    setTouched(true);
    if (username.trim().length === 0 || password.length === 0) {
      return;
    }
    void login(username, password);
  };

  return (
    <KeyboardAvoidingView style={styles.root} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <ScrollView contentContainerStyle={{ flexGrow: 1, justifyContent: 'center', padding: theme.spacing.lg }}>
        <View style={styles.card}>
          <View style={styles.header}>
            <View style={styles.logo}>
              <Text style={styles.logoText}>HR</Text>
            </View>
            <Text style={styles.title}>ورود به سامانه منابع انسانی</Text>
            <Text style={styles.subtitle}>برای مشاهده داشبورد و منوی نقش خود، وارد شوید.</Text>
          </View>

          {error ? <ErrorBanner message={error} /> : null}

          <FormField label="نام کاربری" value={username} onChangeText={setUsername} required error={usernameError} />
          <FormField label="رمز عبور" value={password} onChangeText={setPassword} required error={passwordError} />

          <Button label="ورود" onPress={submit} loading={busy} />

          <View style={styles.demoBox}>
            <Text style={styles.demoTitle}>حساب‌های نمونه (رمز همه: demo)</Text>
            {DEMO_ACCOUNTS.map(account => (
              <Pressable
                key={account.username}
                style={styles.demoRow}
                onPress={() => {
                  setUsername(account.username);
                  setPassword('demo');
                }}>
                <Text style={styles.demoText}>• {account.labelFa}</Text>
              </Pressable>
            ))}
          </View>

          <Text style={styles.hint}>
            رمز عبور روی دستگاه ذخیره نمی‌شود؛ سرور فقط توکن کوتاه‌عمر برمی‌گرداند و هستهٔ بومی آن را در
            Keystore سیستم‌عامل نگه می‌دارد.
          </Text>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
