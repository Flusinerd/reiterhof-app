import type { ReactNode } from "react";
import { ScrollView, View, type ScrollViewProps } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { cn } from "@/lib/cn";

import { BackButton } from "./back-button";

type Edge = "top" | "bottom";

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
  /** Extra classes for the scroll content / inner container. */
  contentClassName?: string;
  /** Pass a `RefreshControl` for pull-to-refresh. */
  refreshControl?: ScrollViewProps["refreshControl"];
  keyboardShouldPersistTaps?: ScrollViewProps["keyboardShouldPersistTaps"];
  className?: string;
};

/**
 * Root of every screen: safe area, warm background, 24 px page padding, and 24 px
 * gap between its direct children (Hero, SectionLabel, Card, ...).
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

  return (
    <View className={cn("flex-1 bg-background", className)} style={{ paddingTop, paddingBottom }}>
      {scroll ? (
        <ScrollView
          contentContainerClassName={inner}
          refreshControl={refreshControl}
          keyboardShouldPersistTaps={keyboardShouldPersistTaps}
          showsVerticalScrollIndicator={false}
        >
          {back ? <BackButton /> : null}
          {children}
        </ScrollView>
      ) : (
        <View className={cn("flex-1", inner)}>
          {back ? <BackButton /> : null}
          {children}
        </View>
      )}
    </View>
  );
}
