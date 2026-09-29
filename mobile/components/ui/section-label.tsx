import type { ReactNode } from "react";
import { View } from "react-native";

import { cn } from "@/lib/cn";

import { Text } from "./text";

export type SectionLabelProps = {
  /** German label, e.g. "Heute". */
  children: string;
  /** Optional element on the right, e.g. a ghost `Button` "Alle". */
  action?: ReactNode;
  className?: string;
};

/**
 * Section heading above a group of cards: Geist 13 px / 600, muted.
 *
 * @example <SectionLabel>Nächste Termine</SectionLabel>
 */
export function SectionLabel({ children, action, className }: SectionLabelProps) {
  return (
    <View className={cn("min-h-6 flex-row items-center justify-between", className)}>
      <Text variant="label" accessibilityRole="header">
        {children}
      </Text>
      {action}
    </View>
  );
}
