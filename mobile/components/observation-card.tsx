import { router, type Href } from "expo-router";
import { Image, View } from "react-native";

import { Badge, PressableCard, Text } from "@/components/ui";
import type { Observation } from "@/lib/api/observations";
import { observationRoute, observationTitle, reportedText, statusBadgeVariant, statusLabel, urgencyBadgeVariant, urgencyLabel } from "@/lib/observations";
import { fileSource } from "@/lib/upload";

/**
 * Card for one observation: title, badges (urgency, status "Beobachten"/"Erledigt"), first line of the
 * description and the first photo. `actions` is a slot below the text for extra buttons of other features
 * (e.g. "In Reha-Plan umwandeln"); taps inside it do not open the detail screen.
 */
export function ObservationCard({
  observation: o,
  actions,
  showHorse = false,
}: {
  observation: Observation;
  actions?: React.ReactNode;
  showHorse?: boolean;
}) {
  const photo = o.media[0];
  return (
    <View className="overflow-hidden rounded-card border border-border bg-card">
      <PressableCard
        padded={false}
        shape="card"
        className="gap-2 border-0 p-5"
        accessibilityLabel={`${observationTitle(o)} öffnen`}
        onPress={() => router.push(observationRoute(o.id) as Href)}
      >
        <View className="flex-row items-start gap-3">
          <View className="flex-1 gap-1">
            {showHorse ? <Text variant="secondary">{o.horse_name}</Text> : null}
            <Text variant="bodyStrong">{observationTitle(o)}</Text>
            <Text variant="caption">
              {o.reporter.name} · {reportedText(o.created_at, new Date())}
            </Text>
          </View>
          {photo ? (
            <Image
              source={fileSource(photo.url)}
              accessibilityIgnoresInvertColors
              className="h-14 w-14 rounded-button-sm bg-divider"
            />
          ) : null}
        </View>
        <View className="flex-row flex-wrap gap-2">
          {o.urgency !== "info" ? (
            <Badge variant={urgencyBadgeVariant(o.urgency)} label={urgencyLabel(o.urgency)} />
          ) : null}
          <Badge variant={statusBadgeVariant(o.status)} label={statusLabel(o.status)} />
        </View>
        {o.description ? (
          <Text variant="bodySm" numberOfLines={2}>
            {o.description}
          </Text>
        ) : null}
      </PressableCard>
      {actions ? <View className="flex-row flex-wrap gap-2 px-5 pb-5">{actions}</View> : null}
    </View>
  );
}
