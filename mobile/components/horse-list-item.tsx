import { ChevronRight } from "lucide-react-native";
import { View } from "react-native";

import { Avatar, Icon, PressableCard, Text } from "@/components/ui";
import type { Horse } from "@/lib/api/horses";
import { joinParts } from "@/lib/horse-format";

/** One row of the horse list: avatar (color key + initial), name, box and owner. */
export function HorseListItem({ horse, onPress }: { horse: Horse; onPress: () => void }) {
  const sub = joinParts([
    horse.box ? `Box ${horse.box}` : null,
    horse.owner ? (horse.is_mine ? "Dein Pferd" : horse.owner.name) : "ohne Besitzer",
  ]);
  return (
    <PressableCard
      padded={false}
      onPress={onPress}
      accessibilityLabel={`${horse.name}, ${sub}`}
      className="min-h-[68px] flex-row items-center gap-4 px-4 py-3"
    >
      <Avatar name={horse.name} colorKey={horse.color_key} size="md" />
      <View className="flex-1">
        <Text variant="bodyStrong" numberOfLines={1}>
          {horse.name}
        </Text>
        <Text variant="secondary" numberOfLines={1}>
          {sub}
        </Text>
      </View>
      <Icon as={ChevronRight} size={20} className="text-muted" />
    </PressableCard>
  );
}
