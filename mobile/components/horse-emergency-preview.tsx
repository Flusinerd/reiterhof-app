import { ChevronRight, Siren } from "lucide-react-native";
import { View } from "react-native";

import { Icon, PressableCard, Text } from "@/components/ui";
import type { EmergencyCard } from "@/lib/api/horses";

/**
 * Compact preview of the emergency card on the horse record: vet, owner phone and the
 * emergency note. Tapping opens the full card with tap-to-call.
 */
export function HorseEmergencyPreview({ card, onPress }: { card: EmergencyCard; onPress: () => void }) {
  const lines: [string, string | null | undefined][] = [
    ["Tierarzt", [card.vet_name, card.vet_phone].filter(Boolean).join(", ")],
    ["Besitzer", [card.owner?.name, card.owner?.phone].filter(Boolean).join(", ")],
  ];
  return (
    <PressableCard onPress={onPress} accessibilityLabel="Notfallkarte öffnen" className="gap-3 border-danger/30">
      <View className="flex-row items-center gap-3">
        <View className="h-9 w-9 items-center justify-center rounded-pill bg-danger-soft">
          <Icon as={Siren} size={20} className="text-danger" />
        </View>
        <Text variant="bodyStrong" className="flex-1">
          Notfallkarte
        </Text>
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </View>
      {card.emergency_note ? (
        <Text variant="bodySm" numberOfLines={3}>
          {card.emergency_note}
        </Text>
      ) : null}
      {lines
        .filter(([, value]) => !!value)
        .map(([label, value]) => (
          <View key={label} className="flex-row gap-3">
            <Text variant="secondary" className="w-20">
              {label}
            </Text>
            <Text variant="bodySm" className="flex-1">
              {value}
            </Text>
          </View>
        ))}
      {!card.emergency_note && !card.vet_name && !card.owner?.phone ? (
        <Text variant="secondary">Noch keine Angaben.</Text>
      ) : null}
    </PressableCard>
  );
}
