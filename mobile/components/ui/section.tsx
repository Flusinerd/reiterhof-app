import type { ReactNode } from "react";
import { View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Text } from "./text";

export type SectionTitleProps = {
  /** German heading, e.g. "Noch offen". */
  children: string;
  /** Optional element on the right, e.g. a ghost `Button` "Alle". */
  action?: ReactNode;
  className?: string;
};

/**
 * Heading of a group of content: Fraunces 20 px on the page background. Use it only where
 * a screen really has several groups; a single list or form needs no heading.
 *
 * @example <SectionTitle action={<Button label="Alle" variant="ghost" size="sm" />}>Offene Anfragen</SectionTitle>
 */
export function SectionTitle({ children, action, className }: SectionTitleProps) {
  return (
    <View className={cn("min-h-8 flex-row items-end justify-between gap-4", className)}>
      <Text variant="heading" accessibilityRole="header" className="flex-1">
        {children}
      </Text>
      {action}
    </View>
  );
}

export type SectionProps = Omit<ViewProps, "children"> & {
  title: string;
  action?: ReactNode;
  /** Optional muted line under the heading. */
  description?: string;
  children: ReactNode;
  className?: string;
};

/**
 * `SectionTitle` plus its content, 12 px apart. Sections are 24 px apart (the `Screen` gap).
 *
 * @example
 * <Section title="Erledigt">
 *   {done.map((h) => <HorseRow key={h.id} horse={h} />)}
 * </Section>
 */
export function Section({ title, action, description, children, className, ...props }: SectionProps) {
  return (
    <View className={cn("gap-3", className)} {...props}>
      <View className="gap-1">
        <SectionTitle action={action}>{title}</SectionTitle>
        {description ? <Text variant="secondary">{description}</Text> : null}
      </View>
      {children}
    </View>
  );
}
