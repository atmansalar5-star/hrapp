/**
 * تم روشن و تیره.
 *
 * در بستهٔ طراحی، حالت تیره با کلاس `.dark` و بازنویسی متغیرهای CSS انجام شده بود.
 * همان رویکرد اینجا با یک Context پیاده شده است تا هیچ کامپوننتی رنگ ثابت نداشته باشد.
 */

import { createContext, useContext, useMemo, useState, type ReactNode } from 'react';
import { darkPalette, layout, palette, spacing, typography, type Palette } from './tokens';

/** تم فعال. */
export interface Theme {
  dark: boolean;
  colors: Palette;
  layout: typeof layout;
  spacing: typeof spacing;
  typography: typeof typography;
}

interface ThemeContextValue {
  theme: Theme;
  toggle: () => void;
  setDark: (dark: boolean) => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

/** فراهم‌کنندهٔ تم؛ انتخاب حالت تیره به‌صورت محلی نگه داشته می‌شود. */
export function ThemeProvider({ children, initialDark = false }: { children: ReactNode; initialDark?: boolean }) {
  const [dark, setDark] = useState(initialDark);

  const value = useMemo<ThemeContextValue>(
    () => ({
      theme: {
        dark,
        colors: dark ? darkPalette : palette,
        layout,
        spacing,
        typography,
      },
      toggle: () => setDark(previous => !previous),
      setDark,
    }),
    [dark],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

/** دسترسی به تم جاری. */
export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme باید داخل ThemeProvider استفاده شود.');
  }
  return context;
}
