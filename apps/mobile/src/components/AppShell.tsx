/**
 * پوستهٔ استاندارد صفحه — مطابق دستور ۳-۱ سند طراحی.
 *
 * ساختار: Header (لوگو، نام کاربر، اعلان‌ها، منوی کاربر) + Sidebar در **سمت راست** +
 * Breadcrumb + Action Bar + ناحیهٔ محتوا. منو از پاسخ سرور/هستهٔ بومی می‌آید، پس
 * آیتم غیرمجاز هرگز رندر نمی‌شود (نه اینکه مخفی شود).
 */

import { useMemo, useState, type ReactNode } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { useTheme } from '../design/theme';
import { faDigits, rtlRow, rtlText } from '../design/rtl';
import { RouteProvider } from '../navigation/RouteContext';
import type { MenuEntry, SessionUser } from '../core/types';
import { breadcrumb, menuTree } from '../navigation/roleMenu';
import { resolveScreen } from '../navigation/screenRegistry';

/** مسیر جاری. */
export interface Route {
  moduleId: string;
  screenKey: string;
  params?: Record<string, string>;
}

interface Props {
  route: Route;
  user: SessionUser;
  menu: MenuEntry[];
  roleFa: string;
  pendingSyncCount: number;
  onNavigate: (route: Route) => void;
  onLogout: () => void;
  /** دکمه‌های Action Bar صفحه. */
  actions?: ReactNode;
  children?: ReactNode;
}

export default function AppShell({
  route,
  user,
  menu,
  roleFa,
  pendingSyncCount,
  onNavigate,
  onLogout,
  actions,
  children,
}: Props) {
  const { theme, toggle } = useTheme();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [search, setSearch] = useState('');

  const steps = useMemo(() => breadcrumb(route.moduleId, route.screenKey), [route]);

  const styles = useMemo(
    () =>
      StyleSheet.create({
        root: { flex: 1, backgroundColor: theme.colors.bg },
        topbar: {
          height: theme.layout.headerHeight,
          backgroundColor: theme.colors.surface,
          borderBottomWidth: 1,
          borderBottomColor: theme.colors.border,
          flexDirection: 'row-reverse',
          alignItems: 'center',
          paddingHorizontal: theme.spacing.lg,
          gap: theme.spacing.sm,
        },
        brand: { flexDirection: 'row-reverse', alignItems: 'center', gap: theme.spacing.sm },
        logo: {
          width: 36,
          height: 36,
          borderRadius: theme.layout.radiusSm,
          backgroundColor: theme.colors.primary,
          alignItems: 'center',
          justifyContent: 'center',
        },
        logoText: { color: '#FFFFFF', fontWeight: '800' },
        brandName: { fontWeight: '800', color: theme.colors.text, fontSize: theme.typography.size.lg },
        searchWrap: {
          flex: 1,
          flexDirection: 'row-reverse',
          alignItems: 'center',
          gap: theme.spacing.sm,
          backgroundColor: theme.colors.bg,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radiusSm,
          paddingHorizontal: theme.spacing.md,
          marginHorizontal: theme.spacing.md,
        },
        searchInput: { flex: 1, color: theme.colors.text, textAlign: 'right', paddingVertical: 8 },
        iconButton: {
          width: 40,
          height: 40,
          borderRadius: theme.layout.radiusSm,
          borderWidth: 1,
          borderColor: theme.colors.border,
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: theme.colors.surface,
        },
        iconText: { color: theme.colors.text, fontSize: 16 },
        badgeDot: {
          position: 'absolute',
          top: -6,
          right: -6,
          backgroundColor: theme.colors.red,
          borderRadius: 999,
          paddingHorizontal: 6,
          paddingVertical: 1,
        },
        badgeText: { color: '#FFFFFF', fontSize: 10, fontWeight: '700' },
        userBox: { flexDirection: 'row-reverse', alignItems: 'center', gap: theme.spacing.sm },
        avatar: {
          width: 36,
          height: 36,
          borderRadius: 18,
          backgroundColor: theme.colors.purple,
          alignItems: 'center',
          justifyContent: 'center',
        },
        avatarText: { color: '#FFFFFF', fontWeight: '700' },
        userMeta: { alignItems: 'flex-end' },
        userName: { color: theme.colors.text, fontWeight: '700', fontSize: theme.typography.size.sm },
        userRole: { color: theme.colors.muted, fontSize: theme.typography.size.xxs },
        userMenu: {
          position: 'absolute',
          top: theme.layout.headerHeight - 8,
          left: theme.spacing.lg,
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          borderRadius: theme.layout.radiusSm,
          padding: theme.spacing.sm,
          minWidth: 200,
          zIndex: 50,
        },
        userMenuRow: { paddingVertical: theme.spacing.sm, paddingHorizontal: theme.spacing.md, borderRadius: theme.layout.radiusSm },
        userMenuText: { color: theme.colors.text },
        body: { flex: 1, flexDirection: 'row-reverse' },
        sidebar: {
          width: theme.layout.sidebarWidth,
          backgroundColor: theme.colors.surface,
          borderLeftWidth: 1,
          borderLeftColor: theme.colors.border,
          paddingTop: theme.spacing.md,
        },
        roleChip: {
          alignSelf: 'flex-end',
          marginHorizontal: theme.spacing.md,
          marginBottom: theme.spacing.md,
          backgroundColor: theme.colors.primaryLight,
          borderRadius: 999,
          paddingHorizontal: theme.spacing.md,
          paddingVertical: 4,
        },
        roleChipText: { color: theme.colors.primaryDark, fontWeight: '700', fontSize: theme.typography.size.xxs },
        groupTitle: {
          color: theme.colors.muted,
          fontSize: theme.typography.size.xxs,
          fontWeight: '700',
          textAlign: 'right',
          marginHorizontal: theme.spacing.md,
          marginTop: theme.spacing.md,
        },
        // سه سطح گروه‌بندی منو: حوزه > اداره > زیرواحد (ماژول با groupTitle می‌آید).
        directorateTitle: {
          color: theme.colors.primaryDark,
          backgroundColor: theme.colors.primaryLight,
          fontSize: theme.typography.size.xxs,
          fontWeight: '800',
          textAlign: 'right',
          marginHorizontal: theme.spacing.sm,
          marginTop: theme.spacing.sm,
          paddingVertical: theme.spacing.xs,
          paddingHorizontal: theme.spacing.sm,
          borderRadius: theme.layout.radiusSm,
        },
        departmentTitle: {
          color: theme.colors.text,
          fontSize: theme.typography.size.xxs,
          fontWeight: '700',
          textAlign: 'right',
          marginHorizontal: theme.spacing.md,
          marginTop: theme.spacing.sm,
        },
        subUnitTitle: {
          color: theme.colors.muted,
          fontSize: theme.typography.size.xxs,
          textAlign: 'right',
          marginHorizontal: theme.spacing.md,
          marginTop: theme.spacing.xs,
        },
        navItem: {
          flexDirection: 'row-reverse',
          alignItems: 'center',
          gap: theme.spacing.sm,
          paddingVertical: theme.spacing.sm,
          paddingHorizontal: theme.spacing.md,
          marginHorizontal: theme.spacing.sm,
          borderRadius: theme.layout.radiusSm,
        },
        navItemActive: { backgroundColor: theme.colors.primaryLight },
        navText: { color: theme.colors.text, ...rtlText, flex: 1 },
        navTextActive: { color: theme.colors.primaryDark, fontWeight: '700' },
        navIcon: { width: 22, textAlign: 'center' },
        main: { flex: 1 },
        breadcrumb: {
          flexDirection: 'row-reverse',
          alignItems: 'center',
          gap: theme.spacing.sm,
          paddingHorizontal: theme.spacing.xl,
          paddingTop: theme.spacing.md,
        },
        crumb: { color: theme.colors.muted, fontSize: theme.typography.size.xs },
        crumbCurrent: { color: theme.colors.text, fontWeight: '700' },
        actionBar: {
          flexDirection: 'row-reverse',
          alignItems: 'center',
          justifyContent: 'space-between',
          paddingHorizontal: theme.spacing.xl,
          paddingVertical: theme.spacing.md,
          gap: theme.spacing.md,
        },
        content: { flex: 1, paddingHorizontal: theme.spacing.xl, paddingBottom: theme.spacing.xl },
        sidebarToggle: { display: 'none' },
      }),
    [theme],
  );

  const activeScreens = menu.find(m => m.moduleId === route.moduleId)?.screens ?? [];
  const Screen = activeScreens.find(s => s.key === route.screenKey)
    ? resolveScreen(activeScreens.find(s => s.key === route.screenKey)?.component ?? '')
    : undefined;

  return (
    <View style={styles.root}>
      <View style={styles.topbar}>
        <View style={styles.brand}>
          <View style={styles.logo}>
            <Text style={styles.logoText}>HR</Text>
          </View>
          <Text style={styles.brandName}>سامانه منابع انسانی</Text>
        </View>

        <View style={styles.searchWrap}>
          <TextInput
            style={styles.searchInput}
            value={search}
            onChangeText={setSearch}
            placeholder="جستجوی سریع…"
            placeholderTextColor={theme.colors.muted}
            returnKeyType="search"
          />
        </View>

        <Pressable style={styles.iconButton} onPress={toggle} accessibilityLabel="تغییر حالت روشن و تیره">
          <Text style={styles.iconText}>{theme.dark ? '☀' : '☾'}</Text>
        </Pressable>

        <Pressable style={styles.iconButton} accessibilityLabel="اعلان‌ها">
          <Text style={styles.iconText}>🔔</Text>
          {pendingSyncCount > 0 ? (
            <View style={styles.badgeDot}>
              <Text style={styles.badgeText}>{faDigits(pendingSyncCount)}</Text>
            </View>
          ) : null}
        </Pressable>

        <Pressable style={styles.userBox} onPress={() => setUserMenuOpen(open => !open)}>
          <View style={styles.avatar}>
            <Text style={styles.avatarText}>{user.displayName.charAt(0)}</Text>
          </View>
          <View style={styles.userMeta}>
            <Text style={styles.userName}>{user.displayName}</Text>
            <Text style={styles.userRole}>{roleFa}</Text>
          </View>
        </Pressable>
      </View>

      {userMenuOpen ? (
        <View style={styles.userMenu}>
          <Pressable style={styles.userMenuRow}>
            <Text style={styles.userMenuText}>پرونده من</Text>
          </Pressable>
          <Pressable style={styles.userMenuRow}>
            <Text style={styles.userMenuText}>
              {pendingSyncCount > 0 ? `${faDigits(pendingSyncCount)} تغییر در انتظار هم‌گام‌سازی` : 'همه‌چیز هم‌گام است'}
            </Text>
          </Pressable>
          <Pressable
            style={styles.userMenuRow}
            onPress={() => {
              setUserMenuOpen(false);
              onLogout();
            }}>
            <Text style={[styles.userMenuText, { color: theme.colors.red }]}>خروج از حساب</Text>
          </Pressable>
        </View>
      ) : null}

      <View style={styles.body}>
        {/* منوی کنار در سمت راست (RTL) */}
        <ScrollView style={styles.sidebar} showsVerticalScrollIndicator={false}>
          <View style={styles.roleChip}>
            <Text style={styles.roleChipText}>{roleFa}</Text>
          </View>
          {menuTree(menu).map(dir => (
            <View key={`dir-${dir.id}`}>
              {dir.titleFa ? <Text style={styles.directorateTitle}>{dir.titleFa}</Text> : null}
              {dir.departments.map(dep => (
                <View key={`dep-${dir.id}-${dep.id}`}>
                  {dep.titleFa ? <Text style={styles.departmentTitle}>{dep.titleFa}</Text> : null}
                  {dep.subUnits.map(sub => (
                    <View key={`sub-${dep.id}-${sub.id}`}>
                      {sub.titleFa ? <Text style={styles.subUnitTitle}>{sub.titleFa}</Text> : null}
                      {sub.modules.map(entry => (
                        <View key={`mod-${dep.id}-${entry.moduleId}`}>
                          <Text style={styles.groupTitle}>
                            {entry.shared ? `${entry.titleFa} — مشترک` : entry.titleFa}
                          </Text>
                          {entry.screens.map(screen => {
                            const active = route.moduleId === entry.moduleId && route.screenKey === screen.key;
                            return (
                              <Pressable
                                key={screen.key}
                                onPress={() => onNavigate({ moduleId: entry.moduleId, screenKey: screen.key })}
                                style={[styles.navItem, active ? styles.navItemActive : null]}
                                accessibilityRole="button">
                                <Text style={styles.navIcon}>{screen.icon}</Text>
                                <Text style={[styles.navText, active ? styles.navTextActive : null]} numberOfLines={1}>
                                  {screen.titleFa}
                                </Text>
                              </Pressable>
                            );
                          })}
                        </View>
                      ))}
                    </View>
                  ))}
                </View>
              ))}
            </View>
          ))}
        </ScrollView>

        <View style={styles.main}>
          <View style={styles.breadcrumb}>
            {steps.map((step, index) => (
              <View key={`${step.label}-${index}`} style={rtlRow}>
                <Text style={index === steps.length - 1 ? [styles.crumb, styles.crumbCurrent] : styles.crumb}>
                  {step.label}
                </Text>
                {index < steps.length - 1 ? <Text style={styles.crumb}>›</Text> : null}
              </View>
            ))}
          </View>

          {actions ? <View style={styles.actionBar}>{actions}</View> : null}

          <View style={styles.content}>
            <RouteProvider route={route} navigate={onNavigate}>
              {Screen ? (
                <Screen />
              ) : (
                <Text style={[styles.crumb, rtlText]}>صفحه‌ای برای این مسیر در منوی نقش شما وجود ندارد.</Text>
              )}
            </RouteProvider>
          </View>
        </View>
      </View>
    </View>
  );
}
