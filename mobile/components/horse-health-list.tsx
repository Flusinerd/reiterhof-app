import { Pressable, View } from "react-native";

import { Text } from "@/components/ui";
import type { HealthItem } from "@/lib/api/horses";
import { cn } from "@/lib/cn";
import { dueLevel, dueText, formatDate, healthKindLabel } from "@/lib/horse-format";

const levelTone = { none: "muted", overdue: "danger", soon: "accent", ok: "brass" } as const;

/**
 * Compact list of the health items of a horse (vaccination, farrier, ...): name and date left, the
 * time until it is due right in brass (red when overdue). Tapping a row opens the health screen.
 */
export function HorseHealthList({ items, onPress }: { items: readonly HealthItem[]; onPress: () => void }) {
  return (
    <View>
      {items.map((item, i) => {
        const level = dueLevel(item.days_until_due);
        const name = item.label || healthKindLabel(item.kind);
        const right = item.daily_time ? `täglich ${item.daily_time}` : dueText(item.days_until_due);
        return (
          <Pressable
            key={item.id}
            accessibilityRole="button"
            accessibilityLabel={`${name}, ${right}`}
            onPress={onPress}
            className={cn("min-h-touch flex-row items-center gap-3 py-3 active:opacity-70", i < items.length - 1 && "border-b border-divider")}
          >
            <View className="flex-1">
              <Text variant="bodyStrong" numberOfLines={1}>
                {name}
              </Text>
              <Text variant="bodySm" tone="muted" numberOfLines={1}>
                {item.due_date && !item.daily_time ? formatDate(item.due_date) : healthKindLabel(item.kind)}
              </Text>
            </View>
            <Text variant="bodySm" tone={item.daily_time ? "muted" : levelTone[level]} className="font-sans-semibold">
              {right}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}
