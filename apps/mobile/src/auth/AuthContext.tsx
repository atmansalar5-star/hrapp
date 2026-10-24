/**
 * زمینهٔ احراز هویت و نشست.
 *
 * وظایف:
 * - ورود و خروج و نگه‌داشتن نشست در حافظهٔ برنامه.
 * - گرفتن منوی نقش‌محور و دامنهٔ دادهٔ کاربر از پاسخ `/api/v1/auth/me` (منبع نهایی)
 *   و در نبود سرور، از مانیفست تولیدشده (فقط برای پیش‌نمایش).
 * - شمارش عملیات در انتظار هم‌گام‌سازی از صف آفلاین هستهٔ بومی برای نمایش نشان در UI.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';

import { ApiError, createClient, type Client } from '../api/client';
import { localMenu } from '../navigation/roleMenu';
import { menu as coreMenu, pendingMutations, hasNativeCore } from '../core/hrmsCore';
import type { MenuEntry, SessionUser } from '../core/types';
import type { Role } from '../navigation/modules.generated';

/** آدرس پایهٔ سرور؛ در استقرار واقعی از پیکربندی محیط خوانده می‌شود. */
const DEFAULT_BASE_URL = 'http://10.0.2.2:8080';

interface AuthContextValue {
  user?: SessionUser;
  roleFa: string;
  menu: MenuEntry[];
  scopes: Record<string, string>;
  client: Client;
  pendingSyncCount: number;
  busy: boolean;
  error?: string;
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => void;
  refreshSyncCount: () => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

/** فراهم‌کنندهٔ نشست. */
export function AuthProvider({ children, baseUrl = DEFAULT_BASE_URL }: { children: ReactNode; baseUrl?: string }) {
  const [token, setToken] = useState<string | undefined>(undefined);
  const [user, setUser] = useState<SessionUser | undefined>(undefined);
  const [roleFa, setRoleFa] = useState('');
  const [menu, setMenu] = useState<MenuEntry[]>([]);
  const [scopes, setScopes] = useState<Record<string, string>>({});
  const [pendingSyncCount, setPendingSyncCount] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | undefined>(undefined);

  const client = useMemo(
    () =>
      createClient({
        baseUrl,
        token,
        onUnauthorized: () => {
          setToken(undefined);
          setUser(undefined);
        },
      }),
    [baseUrl, token],
  );

  useEffect(() => {
    client.setToken(token);
  }, [client, token]);

  const refreshSyncCount = useCallback(() => {
    if (!hasNativeCore()) {
      setPendingSyncCount(0);
      return;
    }
    try {
      setPendingSyncCount(pendingMutations().pending);
    } catch {
      setPendingSyncCount(0);
    }
  }, []);

  useEffect(() => {
    refreshSyncCount();
  }, [refreshSyncCount]);

  const login = useCallback(
    async (username: string, password: string): Promise<boolean> => {
      setBusy(true);
      setError(undefined);
      try {
        const session = await client.login(username.trim(), password);
        setToken(session.accessToken);
        setUser(session.user as SessionUser);

        // منوی نقش‌محور: پاسخ سرور مرجع است.
        try {
          const me = await client.me();
          setMenu(me.menu as MenuEntry[]);
          setRoleFa(me.roleFa);
          setScopes(me.scopes);
        } catch {
          setMenu(fallbackMenu(session.user.role as Role));
          setRoleFa(session.user.displayName);
          setScopes({});
        }

        refreshSyncCount();
        return true;
      } catch (caught) {
        if (caught instanceof ApiError) {
          setError(caught.message);
        } else {
          // نبود شبکه: در پیش‌نمایش بدون سرور، منوی محلی را نشان می‌دهیم تا طراحی قابل بازبینی بماند.
          setError('ارتباط با سرور برقرار نشد. برای ورود واقعی، سرور hrmsd باید در حال اجرا باشد.');
        }
        return false;
      } finally {
        setBusy(false);
      }
    },
    [client, refreshSyncCount],
  );

  const logout = useCallback(() => {
    setUser(undefined);
    setToken(undefined);
    setMenu([]);
    setScopes({});
    setRoleFa('');
    setError(undefined);
  }, []);

  const value: AuthContextValue = {
    user,
    roleFa,
    menu,
    scopes,
    client,
    pendingSyncCount,
    busy,
    error,
    login,
    logout,
    refreshSyncCount,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/** دسترسی به نشست جاری. */
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth باید داخل AuthProvider استفاده شود.');
  }
  return context;
}

/** منوی محلی: اول هستهٔ بومی، و اگر در دسترس نبود مانیفست تولیدشده. */
function fallbackMenu(role: Role): MenuEntry[] {
  try {
    return coreMenu(role);
  } catch {
    return localMenu(role);
  }
}
