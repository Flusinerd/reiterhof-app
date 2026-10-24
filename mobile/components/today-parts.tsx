import { router, type Href } from "expo-router";
import { Pressable, useColorScheme, View } from "react-native";

import { Avatar, Button, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import { colorsForKey } from "@/lib/color-keys";
import { nextAction, nextEyebrow, type TodayEntry } from "@/lib/today";

function open(screen: string) {
  if (screen) router.push(screen as Href);
}

/** The one next task, on the raised surface with its single primary action. */
export function NextTask({ entry, horseColor }: { entry: TodayEntry; horseColor?: string }) {
  return (
    <View className="gap-4 rounded-card border border-border bg-raised p-5">
      <View className="flex-row items-center gap-4">
        {entry.horseName ? <Avatar name={entry.horseName} colorKey={horseColor} size="lg" /> : null}
        <View className="flex-1 gap-1">
          <Text variant="eyebrow" tone={entry.tone === "danger" ? "danger" : "brass"}>
            {nextEyebrow(entry)}
          </Text>
          <Text variant="titleLg">{entry.title}</Text>
          {entry.detail ? <Text variant="bodySm" tone="muted">{entry.detail}</Text> : null}
        </View>
      </View>
      {entry.screen ? <Button label={nextAction(entry)} fullWidth onPress={() => open(entry.screen)} /> : null}
    </View>
  );
}

/** One line of the timeline: time in brass, title and detail; done lines fade, overdue lines are red. */
export function TimelineRow({ entry, last, horseColor }: { entry: TodayEntry; last: boolean; horseColor?: string }) {
  const scheme = useColorScheme() === "dark" ? "dark" : "light";
  const dot = !entry.time && entry.horseId && entry.tone !== "danger";
  return (
    <Pressable
      accessibilityRole={entry.screen ? "button" : undefined}
      accessibilityLabel={[entry.time, entry.title, entry.detail].filter(Boolean).join(", ")}
      disabled={!entry.screen}
      onPress={() => open(entry.screen)}
      className={cn("min-h-touch flex-row gap-4 py-3 active:opacity-70", !last && "border-b border-divider", entry.done && "opacity-50")}
    >
      {dot ? (
        <View className="w-12 pt-1.5">
          <View className="h-2.5 w-2.5 rounded-pill" style={{ backgroundColor: colorsForKey(horseColor, scheme).bg }} />
        </View>
      ) : (
        <Text variant="bodySm" tone={entry.tone === "danger" ? "danger" : "brass"} className="w-12 font-sans-semibold">
          {entry.time ?? (entry.tone === "danger" ? "fällig" : "·")}
        </Text>
      )}
      <View className="flex-1 gap-0.5">
        <Text variant="bodyStrong">{entry.title}</Text>
        {entry.detail ? (
          <Text variant="bodySm" tone={entry.tone === "danger" ? "danger" : "muted"}>
            {entry.detail}
          </Text>
        ) : null}
      </View>
    </Pressable>
  );
}

export function Timeline({ entries, colorOf }: { entries: readonly TodayEntry[]; colorOf: (horseId?: string) => string | undefined }) {
  return (
    <View>
      {entries.map((e, i) => (
        <TimelineRow key={e.id} entry={e} last={i === entries.length - 1} horseColor={colorOf(e.horseId)} />
      ))}
    </View>
  );
}
