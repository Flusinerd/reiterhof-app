import { createContext, useContext, type ComponentType, type ReactNode } from "react";
import { ScrollView, View, type ScrollViewProps } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { cn } from "@/lib/cn";

import { BackButton } from "./back-button";

type Edge = "top" | "bottom";

/**
 * The app menu shown top right on every screen (presence, reminders, settings). The app
 * provides it once at the root while someone is signed in; without a provider there is none.
 */
export const ScreenMenuContext = createContext<ComponentType | null>(null);

export type ScreenProps = {
  children: ReactNode;
  /** Scrollable content (default). Set `false` for fixed-height layouts. */
  scroll?: boolean;
  /**
   * Safe-area edges to pad. Default `["top"]` because tab screens have a tab bar
   * below. Sub pages (no tab bar) use `["top", "bottom"]`, which is what `back` does.
   */
  edges?: readonly Edge[];
  /** Sub page: shows a round `BackButton` above the content and pads the bottom edge. */
  back?: boolean;
  /** Shows the app menu top right (default). Set `false` on screens that must not be left by accident. */
  menu?: boolean;
  /** Extra classes for the scroll content / inner container. */
  contentClassName?: string;
  /** Pass a `RefreshControl` for pull-to-refresh. */
  refreshControl?: ScrollViewProps["refreshControl"];
  keyboardShouldPersistTaps?: ScrollViewProps["keyboardShouldPersistTaps"];
  className?: string;
};

/**
 * Root of every screen: safe area, warm background, 24 px page padding, and 24 px
 * gap between its direct children (Hero, SectionLabel, Card, ...). A top bar carries the
 * back button (left) and the app menu (right).
 *
 * @example
 * <Screen>
 *   <Hero title="Decken" description="..." />
 *   <SectionLabel>Heute</SectionLabel>
 *   <Card>...</Card>
 * </Screen>
 */
export function Screen({
  children,
  scroll = true,
  edges,
  back = false,
  menu = true,
  contentClassName,
  refreshControl,
  keyboardShouldPersistTaps,
  className,
}: ScreenProps) {
  const insets = useSafeAreaInsets();
  const activeEdges = edges ?? (back ? ["top", "bottom"] : ["top"]);
  const paddingTop = activeEdges.includes("top") ? insets.top : 0;
  const paddingBottom = activeEdges.includes("bottom") ? insets.bottom : 0;
  const inner = cn("gap-6 px-6 pb-8 pt-4", contentClassName);
  const Menu = useContext(ScreenMenuContext);
  const showMenu = menu && Menu !== null;
  const topBar =
    back || showMenu ? (
      <View className="min-h-touch flex-row items-center justify-between gap-3">
        {back ? <BackButton /> : <View />}
        {showMenu ? <Menu /> : null}
      </View>
    ) : null;

  return (
    <View className={cn("flex-1 bg-background", className)} style={{ paddingTop, paddingBottom }}>
      {scroll ? (
        <ScrollView
          contentContainerClassName={inner}
          refreshControl={refreshControl}
          keyboardShouldPersistTaps={keyboardShouldPersistTaps}
          showsVerticalScrollIndicator={false}
        >
          {topBar}
          {children}
        </ScrollView>
      ) : (
        <View className={cn("flex-1", inner)}>
          {topBar}
          {children}
        </View>
      )}
    </View>
  );
}
