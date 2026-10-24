import { ChevronRight } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { Icon, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import type { EmergencyCard } from "@/lib/api/horses";

/**
 * Compact preview of the emergency card on the horse record: the emergency note, vet and owner with
 * their numbers in brass. Tapping opens the full card with tap-to-call.
 */
export function HorseEmergencyPreview({ card, onPress }: { card: EmergencyCard; onPress: () => void }) {
  const lines: { label: string; name: string | null | undefined; phone: string | null | undefined }[] = [
    { label: "Tierarzt", name: card.vet_name, phone: card.vet_phone },
    { label: "Besitzer", name: card.owner?.name, phone: card.owner?.phone },
  ];
  const shown = lines.filter((l) => l.name || l.phone);
  const empty = !card.emergency_note && shown.length === 0;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel="Notfallkarte öffnen"
      onPress={onPress}
      className="min-h-touch flex-row items-center gap-3 active:opacity-70"
    >
      <View className="flex-1">
        {card.emergency_note ? (
          <Text variant="bodySm" numberOfLines={3} className="pb-3">
            {card.emergency_note}
          </Text>
        ) : null}
        {shown.length > 0 ? (
          <RowList>
            {shown.map((l) => (
              <View key={l.label} className="flex-row items-baseline gap-3 py-3">
                <Text variant="eyebrow" className="w-20">
                  {l.label}
                </Text>
                <View className="flex-1">
                  {l.name ? <Text variant="bodySm">{l.name}</Text> : null}
                  {l.phone ? (
                    <Text variant="heroNumberSm" tone="brass" className="text-[22px] leading-[28px]">
                      {l.phone}
                    </Text>
                  ) : null}
                </View>
              </View>
            ))}
          </RowList>
        ) : null}
        {empty ? <Text variant="secondary">Noch keine Angaben.</Text> : null}
      </View>
      <Icon as={ChevronRight} size={20} className="text-muted" />
    </Pressable>
  );
}
