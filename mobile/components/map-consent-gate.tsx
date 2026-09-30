import { Map as MapIcon } from "lucide-react-native";
import type { ReactNode } from "react";
import { View } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { Button, Icon, Text } from "@/components/ui";
import { useHasConsent } from "@/lib/consent";

/**
 * Shows a map only with the `maps` consent (JAN-85): the tiles come from Apple, Google or
 * OpenFreeMap, which see the viewer's IP address and the map area. Without the consent a
 * placeholder of the same height explains that and offers "Karte anzeigen", which opens the
 * consent sheet; the ride itself is recorded either way.
 */
export function MapConsentGate({ children, height = 280 }: { children: ReactNode; height?: number }) {
  const granted = useHasConsent("maps");
  const consent = useConsentPrompt();
  if (granted) return <>{children}</>;
  return (
    <>
      <View style={{ height }} className="items-center justify-center gap-3 rounded-card border border-border bg-card px-6">
        <Icon as={MapIcon} size={24} className="text-muted" />
        <Text variant="secondary" className="text-center">
          Die Karte lädt Kartenkacheln von Apple, Google oder OpenFreeMap. Der Dienst sieht dabei deine IP-Adresse und die Gegend
          deiner Strecke. Die Aufzeichnung läuft auch ohne Karte.
        </Text>
        {granted === false ? <Button label="Karte anzeigen" variant="outline" size="sm" onPress={() => void consent.ensure("maps")} /> : null}
      </View>
      {consent.sheet}
    </>
  );
}
