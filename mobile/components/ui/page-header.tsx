import type { ReactNode } from "react";
import { View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Text } from "./text";

export type PageHeaderProps = Omit<ViewProps, "children"> & {
  /** Small muted line above the title, e.g. the horse's name or today's date. */
  eyebrow?: string;
  /** Page title (Fraunces 32 px). Give a `title`, a `value`, or both. */
  title?: string;
  /** Big figure (Fraunces), e.g. "12" or "7°". Sits below the title. */
  value?: string;
  /** Size of the figure: 44, 56 or 72 px. */
  valueSize?: "sm" | "md" | "lg";
  /** Unit or caption next to the figure, e.g. "Pferde". */
  unit?: string;
  /** One supporting sentence below, muted. */
  description?: string;
  /** Element right of the title row, e.g. an icon `Button`. */
  action?: ReactNode;
  /** Extra content below the text (badges, a progress bar, a button row). */
  children?: ReactNode;
  className?: string;
};

const valueVariant = { sm: "heroNumberSm", md: "heroNumber", lg: "heroNumberLg" } as const;

/**
 * The head of every screen: title and, if there is one, the screen's key figure, set
 * directly on the page background. No box, no fill. Exactly one per screen.
 *
 * @example
 * <PageHeader title="Pferde" value="12" unit="Pferde" description="Zwei davon von dir." />
 * <PageHeader eyebrow="Luna" title="Gesundheit" action={<Button size="icon" ... />} />
 */
export function PageHeader({
  eyebrow,
  title,
  value,
  valueSize = "md",
  unit,
  description,
  action,
  children,
  className,
  ...props
}: PageHeaderProps) {
  return (
    <View className={cn("gap-2", className)} {...props}>
      {eyebrow ? <Text variant="secondary">{eyebrow}</Text> : null}
      {title || action ? (
        <View className="flex-row items-start justify-between gap-4">
          {title ? (
            <Text variant="display" accessibilityRole="header" className="flex-1">
              {title}
            </Text>
          ) : (
            <View className="flex-1" />
          )}
          {action}
        </View>
      ) : null}
      {value ? (
        <View className="flex-row items-baseline gap-2">
          <Text variant={valueVariant[valueSize]} accessibilityRole={title ? undefined : "header"}>
            {value}
          </Text>
          {unit ? (
            <Text variant="body" tone="muted" className="flex-1">
              {unit}
            </Text>
          ) : null}
        </View>
      ) : null}
      {description ? (
        <Text variant="body" tone="muted">
          {description}
        </Text>
      ) : null}
      {children ? <View className="mt-2 gap-3">{children}</View> : null}
    </View>
  );
}
