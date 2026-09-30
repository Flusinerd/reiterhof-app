import type { ReactNode } from "react";
import { View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Text, TextToneContext } from "./text";

export type LivePanelProps = Omit<ViewProps, "children"> & {
  /** Small line above, e.g. "Ausritt · pausiert". */
  eyebrow?: string;
  /** Title instead of, or above, the reading. */
  title?: string;
  /** The live reading (Fraunces 72 px), e.g. "12:34". */
  value?: string;
  description?: string;
  /** Gait chip, progress bar, ... Inherit white text. */
  children?: ReactNode;
  className?: string;
};

/**
 * Dark green readout for a running session: the one place with a filled block. The gait
 * colors (`gait-walk`, `gait-trot`, white canter) are designed for this background.
 * Nothing else on a screen uses a colored block; page heads are `PageHeader`.
 *
 * @example
 * <LivePanel eyebrow="Ausritt" value="12:34"><GaitChip gait="trot" /></LivePanel>
 */
export function LivePanel({ eyebrow, title, value, description, children, className, ...props }: LivePanelProps) {
  return (
    <TextToneContext.Provider value="inverse">
      <View className={cn("gap-3 rounded-card bg-primary-deeper p-6", className)} {...props}>
        {eyebrow ? (
          <Text variant="secondary" className="text-white/70">
            {eyebrow}
          </Text>
        ) : null}
        {title ? (
          <Text variant="title" accessibilityRole="header">
            {title}
          </Text>
        ) : null}
        {value ? (
          <Text variant="heroNumberLg" accessibilityRole={title ? undefined : "header"}>
            {value}
          </Text>
        ) : null}
        {description ? (
          <Text variant="body" className="text-white/85">
            {description}
          </Text>
        ) : null}
        {children}
      </View>
    </TextToneContext.Provider>
  );
}
