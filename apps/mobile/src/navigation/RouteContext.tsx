/**
 * زمینهٔ مسیر جاری.
 *
 * صفحه‌ها بدون دریافت props از مسیر و پارامترهایش آگاه می‌شوند؛ این کار ثبت کامپوننت
 * در `screenRegistry` را ساده نگه می‌دارد (کامپوننت بدون آرگومان) و مانع از آن می‌شود
 * که ناوبری و داده به هم گره بخورند.
 */

import { createContext, useContext, type ReactNode } from 'react';
import type { Route } from '../components/AppShell';

interface RouteContextValue {
  route: Route;
  navigate: (route: Route) => void;
  /** رفتن به پروندهٔ یک کارمند (نمونهٔ استفاده از پارامتر). */
  openEmployee: (employeeId: string) => void;
}

const RouteContext = createContext<RouteContextValue | undefined>(undefined);

/** فراهم‌کنندهٔ مسیر جاری. */
export function RouteProvider({
  route,
  navigate,
  children,
}: {
  route: Route;
  navigate: (route: Route) => void;
  children: ReactNode;
}) {
  const value: RouteContextValue = {
    route,
    navigate,
    openEmployee: employeeId => navigate({ moduleId: 'employee', screenKey: 'profile', params: { employeeId } }),
  };
  return <RouteContext.Provider value={value}>{children}</RouteContext.Provider>;
}

/** دسترسی به مسیر جاری. */
export function useRoute(): RouteContextValue {
  const context = useContext(RouteContext);
  if (!context) {
    throw new Error('useRoute باید داخل RouteProvider استفاده شود.');
  }
  return context;
}

/** پارامترهای مسیر جاری. */
export function useRouteParams(): Record<string, string> {
  return useRoute().route.params ?? {};
}
