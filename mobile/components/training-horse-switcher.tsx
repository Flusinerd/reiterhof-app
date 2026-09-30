import { ScrollView } from "react-native";

import { Pill } from "@/components/ui";
import type { TrainingHorse } from "@/lib/api/training";

/** Horizontal chips to switch between the horses the user owns or rides. */
export function HorseSwitcher({
  horses,
  selected,
  onSelect,
}: {
  horses: readonly TrainingHorse[];
  selected: string | undefined;
  onSelect: (id: string) => void;
}) {
  if (horses.length < 2) return null;
  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerClassName="gap-2"
      accessibilityRole="tablist"
    >
      {horses.map((h) => (
        <Pill key={h.id} label={h.name} selected={h.id === selected} onPress={() => onSelect(h.id)} />
      ))}
    </ScrollView>
  );
}
