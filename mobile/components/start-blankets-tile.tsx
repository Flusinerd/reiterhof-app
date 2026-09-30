import { router, type Href } from "expo-router";
import { ChevronRight } from "lucide-react-native";
import { View } from "react-native";

import { Badge, Icon, PressableCard, Text } from "@/components/ui";
import {
  progressFraction,
  progressLabel,
  progressText,
  reminderHint,
  type Today,
} from "@/lib/blankets";

/**
 * "Decken heute" tile of the start screen (JAN-35): progress N/total, how many horses are
 * open, the reminder time, and a hint when one of my own horses is still open.
 */
export function StartBlanketsTile({ today, mineIds }: { today: Today | undefined; mineIds: ReadonlySet<string> }) {
  const openMine = today?.horses.filter((h) => !h.done && mineIds.has(h.horse.id)) ?? [];
  const summary = today ? `Decken: ${progressLabel(today.progress)}. ${progressText(today.progress)}` : "Decken werden geladen";
  return (
    <PressableCard accessibilityLabel={summary} onPress={() => router.push("/blankets" as Href)}>
      <View className="gap-3">
        <View className="flex-row items-center gap-4">
          <View className="flex-1 gap-1">
            <Text variant="label">Decken heute</Text>
            <View className="flex-row items-baseline gap-2">
              <Text variant="heroNumberSm">{today ? progressLabel(today.progress) : "–"}</Text>
              <Text variant="secondary">versorgt</Text>
            </View>
            <Text variant="secondary">
              {today ? `${progressText(today.progress)} · ${reminderHint(today.reminder_time)}` : "Wird geladen ..."}
            </Text>
          </View>
          <Icon as={ChevronRight} size={20} className="text-muted" />
        </View>
        {today ? (
          <View
            className="h-2 overflow-hidden rounded-pill bg-divider"
            accessibilityRole="progressbar"
            accessibilityValue={{ min: 0, max: today.progress.total, now: today.progress.done }}
          >
            <View className="h-2 rounded-pill bg-primary" style={{ width: `${progressFraction(today.progress) * 100}%` }} />
          </View>
        ) : null}
        {openMine.length > 0 ? (
          <View className="flex-row flex-wrap gap-2">
            {openMine.map((h) => (
              <Badge key={h.horse.id} variant="accent" label={`${h.horse.name} offen`} />
            ))}
          </View>
        ) : null}
      </View>
    </PressableCard>
  );
}
