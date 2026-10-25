import type { Href } from "expo-router";
import { Pressable, View } from "react-native";

import { AuthImage } from "@/components/auth-image";
import { Text } from "@/components/ui";
import type { Observation } from "@/lib/api/observations";
import { pushOnce } from "@/lib/navigation";
import { observationEyebrow, observationRoute, observationTitle, reportedText } from "@/lib/observations";

/**
 * Row for one observation (put rows in a `RowList`): eyebrow with urgency and status in a tone, title,
 * who reported it and when, the first lines of the description and the first photo. `actions` is a
 * slot below the text for extra buttons of other features (e.g. "Reha-Plan erstellen"); taps inside
 * it do not open the detail screen.
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
  const eyebrow = observationEyebrow(o);
  return (
    <View className="gap-3 py-3">
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${observationTitle(o)} öffnen`}
        onPress={() => pushOnce(observationRoute(o.id) as Href)}
        className="min-h-[64px] flex-row items-start gap-3 active:opacity-70"
      >
        <View className="flex-1 gap-0.5">
          <Text variant="eyebrow" tone={eyebrow.tone}>
            {eyebrow.text}
          </Text>
          <Text variant="bodyStrong">{observationTitle(o)}</Text>
          <Text variant="bodySm" tone="muted">
            {[showHorse ? o.horse_name : "", o.reporter.name, reportedText(o.created_at, new Date())].filter(Boolean).join(" · ")}
          </Text>
          {o.description ? (
            <Text variant="bodySm" numberOfLines={2} className="mt-1">
              {o.description}
            </Text>
          ) : null}
        </View>
        {photo ? (
          <AuthImage
            url={photo.url}
            accessibilityIgnoresInvertColors
            className="h-16 w-16 rounded-button-sm bg-divider"
          />
        ) : null}
      </Pressable>
      {actions ? <View className="flex-row flex-wrap gap-2">{actions}</View> : null}
    </View>
  );
}
