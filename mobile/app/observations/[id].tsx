import { router, useLocalSearchParams, type Href } from "expo-router";
import { Check, HeartPulse, RotateCcw, Siren } from "lucide-react-native";
import { useState } from "react";
import { Image, Linking, Pressable, View } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Horse } from "@/components/icons/horse";
import { Badge, Button, PageHeader, Screen, Section, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { useObservation, useObservationEvents, useSetObservationStatus } from "@/lib/api/observations";
import { horseRoutes, joinParts } from "@/lib/horse-format";
import {
  bodyPartLabel,
  categoryLabel,
  nextStatus,
  reportedText,
  statusActionLabel,
  statusBadgeVariant,
  statusLabel,
  urgencyBadgeVariant,
  urgencyLabel,
} from "@/lib/observations";
import { fileSource, fileUrlWithToken } from "@/lib/upload";

/** One observation with photos and the status change "Beobachten" / "Erledigt" (JAN-52). */
export default function ObservationDetail() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const query = useObservation(id);
  const setStatus = useSetObservationStatus();
  const [error, setError] = useState<string | null>(null);
  useObservationEvents();

  const o = query.data;
  if (!o) {
    return (
      <Screen back>
        {query.isError ? (
          <HorseError error={query.error} onRetry={() => query.refetch()} />
        ) : (
          <HorseLoading />
        )}
      </Screen>
    );
  }

  async function change() {
    if (!o) return;
    setError(null);
    try {
      await setStatus.mutateAsync({ id: o.id, status: nextStatus(o.status) });
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <Screen back>
      <PageHeader
        eyebrow={o.horse_name}
        title={categoryLabel(o.category)}
        description={joinParts([
          bodyPartLabel(o.body_part),
          `${o.reporter.name}, ${reportedText(o.created_at, new Date())}`,
        ])}
      >
        <View className="flex-row flex-wrap gap-2">
          <Badge variant={statusBadgeVariant(o.status)} label={statusLabel(o.status)} />
          <Badge variant={urgencyBadgeVariant(o.urgency)} label={urgencyLabel(o.urgency)} />
        </View>
      </PageHeader>

      {o.description ? (
        <Section title="Beschreibung">
          <Text variant="body">{o.description}</Text>
        </Section>
      ) : null}

      {o.media.length > 0 ? (
        <Section title="Fotos">
          <View className="flex-row flex-wrap gap-3">
            {o.media.map((m, i) => (
              <Pressable
                key={m.path}
                accessibilityRole="imagebutton"
                accessibilityLabel={`Foto ${i + 1} öffnen`}
                onPress={() => void Linking.openURL(fileUrlWithToken(m.url))}
              >
                <Image
                  source={fileSource(m.url)}
                  accessibilityIgnoresInvertColors
                  className="h-28 w-28 rounded-tile bg-divider"
                />
              </Pressable>
            ))}
          </View>
        </Section>
      ) : null}

      <View className="gap-3">
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        {o.can_change ? (
          <Button
            label={statusActionLabel(o.status)}
            icon={o.status === "watch" ? Check : RotateCcw}
            variant={o.status === "watch" ? "primary" : "outline"}
            fullWidth
            loading={setStatus.isPending}
            onPress={() => void change()}
          />
        ) : (
          <Text variant="caption">Status ändern nur Besitzer, Admins und Melder.</Text>
        )}
        {o.urgency === "urgent" ? (
          <Button
            label="Notfallkarte"
            icon={Siren}
            variant="outline"
            fullWidth
            onPress={() => router.push(horseRoutes.emergency(o.horse_id) as Href)}
          />
        ) : null}
        {o.reha_plan_id ? (
          <Button
            label="Reha-Plan"
            icon={HeartPulse}
            variant="outline"
            fullWidth
            onPress={() => router.push(horseRoutes.reha(o.horse_id) as Href)}
          />
        ) : null}
        <Button
          label="Pferdeakte"
          icon={Horse}
          variant="ghost"
          fullWidth
          onPress={() => router.push(horseRoutes.detail(o.horse_id) as Href)}
        />
      </View>
    </Screen>
  );
}
