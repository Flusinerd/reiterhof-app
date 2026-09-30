import { View } from "react-native";

import { Card, PressableCard, Text } from "@/components/ui";
import type { HealthItem } from "@/lib/api/horses";
import { dueLevel, dueText, formatDate, healthKindLabel, SUMMARY_KINDS } from "@/lib/horse-format";
import { cn } from "@/lib/cn";

const levelClass = {
  none: "text-muted",
  overdue: "text-danger",
  soon: "text-accent-text",
  ok: "text-primary",
} as const;

/**
 * The four tiles of the horse record: next vaccination, farrier, deworming and dentist with the
 * days until due. Tiles are tappable when `onPress` is given (opens the health screen).
 */
export function HorseHealthTiles({
  summary,
  onPress,
}: {
  summary: Record<string, HealthItem | null> | undefined;
  onPress?: () => void;
}) {
  return (
    <View className="flex-row flex-wrap gap-3">
      {SUMMARY_KINDS.map((kind) => {
        const item = summary?.[kind] ?? null;
        const level = dueLevel(item?.days_until_due);
        const body = (
          <>
            <Text variant="secondary">{healthKindLabel(kind)}</Text>
            <Text variant="bodyStrong" className={cn("mt-1", levelClass[level])}>
              {dueText(item?.days_until_due)}
            </Text>
            <Text variant="caption" numberOfLines={1}>
              {item?.due_date ? formatDate(item.due_date) : "–"}
            </Text>
          </>
        );
        const label = `${healthKindLabel(kind)}: ${dueText(item?.days_until_due)}`;
        return onPress ? (
          <PressableCard
            key={kind}
            shape="tile"
            onPress={onPress}
            accessibilityLabel={label}
            className="min-h-[88px] w-[48%] grow"
          >
            {body}
          </PressableCard>
        ) : (
          <Card key={kind} shape="tile" accessibilityLabel={label} className="min-h-[88px] w-[48%] grow">
            {body}
          </Card>
        );
      })}
    </View>
  );
}
