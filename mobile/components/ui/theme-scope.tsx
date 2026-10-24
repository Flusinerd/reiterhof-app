import { vars } from "nativewind";
import { useEffect, useMemo, type ReactNode } from "react";
import { Platform, View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";
import { SchemeOverrideContext } from "@/lib/theme";
import { paletteVariables, palettes } from "@/lib/tokens";

const schemeVars = (scheme: "light" | "dark") => paletteVariables(palettes[scheme]);

export type ThemeScopeProps = Omit<ViewProps, "children"> & {
  /** The scheme for this subtree, whatever the phone is set to. Default `dark`. */
  scheme?: "light" | "dark";
  children: ReactNode;
  className?: string;
};

/**
 * Pins a subtree to one color scheme. The token class names (`bg-background`, `text-muted`, ...) read
 * CSS variables, so the scope overrides them for its children with NativeWind's `vars()`, and
 * `useColors()` inside returns the same palette for values that cannot be class names. Used by
 * flows that belong to the night and stay dark in daylight, like the evening blanket round
 * (docs/design-system.md, "Always-dark flows").
 *
 * On the web a `Modal` (our `Sheet`) is rendered outside the wrapper, so the variables are also set
 * on `<html>` while the scope is mounted and restored when it unmounts. Set the status bar yourself
 * (`<StatusBar style="light" />`) and the stack's `contentStyle` if the screen has one.
 *
 * @example
 * <ThemeScope>
 *   <Screen back>...</Screen>
 * </ThemeScope>
 */
export function ThemeScope({ scheme = "dark", className, style, children, ...props }: ThemeScopeProps) {
  const themeVars = useMemo(() => vars(schemeVars(scheme)), [scheme]);

  useEffect(() => {
    if (Platform.OS !== "web" || typeof document === "undefined") return;
    const root = document.documentElement;
    const values = schemeVars(scheme);
    const previous = Object.keys(values).map((k) => [k, root.style.getPropertyValue(k)] as const);
    const previousScheme = root.style.colorScheme;
    for (const [k, v] of Object.entries(values)) root.style.setProperty(k, v);
    root.style.colorScheme = scheme;
    return () => {
      for (const [k, v] of previous) {
        if (v) root.style.setProperty(k, v);
        else root.style.removeProperty(k);
      }
      root.style.colorScheme = previousScheme;
    };
  }, [scheme]);

  return (
    <SchemeOverrideContext.Provider value={scheme}>
      <View className={cn("flex-1", className)} style={[themeVars, style]} {...props}>
        {children}
      </View>
    </SchemeOverrideContext.Provider>
  );
}
