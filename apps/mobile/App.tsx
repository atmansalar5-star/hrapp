/**
 * ریشهٔ اپلیکیشن موبایل.
 *
 * ناوبری بر پایهٔ مانیفست ماژول‌ها و پاسخ `/api/v1/auth/me` ساخته می‌شود؛ تا وقتی
 * کاربر ورود نکرده صفحهٔ ورود نمایش داده می‌شود. راست‌چین سراسری پیش از رندر اول فعال
 * می‌شود تا در همان اجرای نخست هم چیدمان درست باشد (محدودیت `forceRTL` در React Native).
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { SafeAreaView, StatusBar, StyleSheet } from 'react-native';

import { AuthProvider, useAuth } from './src/auth/AuthContext';
import AppShell, { type Route } from './src/components/AppShell';
import { enableRTL } from './src/design/rtl';
import { ThemeProvider, useTheme } from './src/design/theme';
import LoginScreen from './src/features/auth/LoginScreen';
import { defaultRoute } from './src/navigation/roleMenu';

enableRTL();

/** بدنهٔ اپ: انتخاب میان صفحهٔ ورود و پوستهٔ اصلی. */
function AppBody() {
  const { theme } = useTheme();
  const { user, menu, roleFa, pendingSyncCount, logout } = useAuth();
  const [route, setRoute] = useState<Route | undefined>(undefined);

  const initialRoute = useMemo<Route | undefined>(() => {
    if (!user) {
      return undefined;
    }
    const fallback = defaultRoute(user.role);
    return fallback ? { moduleId: fallback.moduleId, screenKey: fallback.screenKey } : undefined;
  }, [user]);

  useEffect(() => {
    setRoute(initialRoute);
  }, [initialRoute]);

  const navigate = useCallback((next: Route) => setRoute(next), []);

  if (!user || !route) {
    return <LoginScreen />;
  }

  return (
    <AppShell
      route={route}
      user={user}
      menu={menu}
      roleFa={roleFa}
      pendingSyncCount={pendingSyncCount}
      onNavigate={navigate}
      onLogout={logout}
      actions={
        <></>
      }
    />
  );
}

/** پوستهٔ درونی اپ که پس از فعال‌شدن تم رندر می‌شود. */
function ThemedApp() {
  const { theme } = useTheme();
  return (
    <SafeAreaView style={[styles.root, { backgroundColor: theme.colors.bg }]}>
      <StatusBar barStyle={theme.dark ? 'light-content' : 'dark-content'} />
      <AppBody />
    </SafeAreaView>
  );
}

/** ریشهٔ اپلیکیشن همراه فراهم‌کننده‌های تم و نشست. */
export default function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <ThemedApp />
      </AuthProvider>
    </ThemeProvider>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
});
