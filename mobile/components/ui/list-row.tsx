import { ChevronRight } from "lucide-react-native";
import { Children, Fragment, type ReactNode } from "react";
import { Pressable, View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Divider } from "./card";
import { Icon } from "./icon";
import { Text, type TextTone } from "./text";

export type ListRowProps = {
  /** Small upper-case line above the title (kind, weekday, state). */
  eyebrow?: string;
  eyebrowTone?: TextTone;
  /** German title of the row. */
  title: string;
  /** One muted line below the title. */
  detail?: string;
  /** Left of the text: an `Avatar`, an icon, a big number. */
  leading?: ReactNode;
  /** Right of the text, outside the tap area: a quiet `Button`, a value in brass. */
  trailing?: ReactNode;
  /** Extra content below the texts (badges, a second line, a show block). */
  children?: ReactNode;
  /** Makes the text area tappable. */
  onPress?: () => void;
  /** Chevron at the end of the tap area (default: shown when `onPress` is given). */
  chevron?: boolean;
  accessibilityLabel?: string;
  accessibilityHint?: string;
  className?: string;
};

/**
 * A row of a list on the page background, no card: leading element, eyebrow, title, detail and a
 * trailing element. Stack rows in a `RowList`, which puts a hairline between them
 * (docs/design-system.md, rule 1).
 *
 * @example
 * <RowList>
 *   <ListRow eyebrow="Impfung" title="Influenza" detail="12.03.2027" trailing={<Text tone="brass">in 3 Tagen</Text>} onPress={open} />
 * </RowList>
 */
export function ListRow({
  eyebrow,
  eyebrowTone,
  title,
  detail,
  leading,
  trailing,
  children,
  onPress,
  chevron,
  accessibilityLabel,
  accessibilityHint,
  className,
}: ListRowProps) {
  const showChevron = chevron ?? !!onPress;
  const body = (
    <>
      {leading}
      <View className="flex-1 gap-0.5">
        {eyebrow ? (
          <Text variant="eyebrow" tone={eyebrowTone}>
            {eyebrow}
          </Text>
        ) : null}
        <Text variant="bodyStrong">{title}</Text>
        {detail ? (
          <Text variant="bodySm" tone="muted">
            {detail}
          </Text>
        ) : null}
        {children}
      </View>
      {showChevron ? <Icon as={ChevronRight} size={20} className="text-muted" /> : null}
    </>
  );
  return (
    <View className={cn("min-h-[56px] flex-row items-center gap-3 py-3", className)}>
      {onPress ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={accessibilityLabel ?? [eyebrow, title, detail].filter(Boolean).join(", ")}
          accessibilityHint={accessibilityHint}
          onPress={onPress}
          className="flex-1 flex-row items-center gap-3 active:opacity-70"
        >
          {body}
        </Pressable>
      ) : (
        <View
          accessible={accessibilityLabel !== undefined}
          accessibilityLabel={accessibilityLabel}
          className="flex-1 flex-row items-center gap-3"
        >
          {body}
        </View>
      )}
      {trailing}
    </View>
  );
}

/** Stacks `ListRow`s (or any rows) with a hairline between them, nothing around them. */
export function RowList({ children, className, ...props }: ViewProps & { children: ReactNode; className?: string }) {
  const rows = Children.toArray(children);
  return (
    <View className={className} {...props}>
      {rows.map((row, i) => (
        <Fragment key={i}>
          {i > 0 ? <Divider /> : null}
          {row}
        </Fragment>
      ))}
    </View>
  );
}
