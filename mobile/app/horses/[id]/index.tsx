import { useQueryClient } from "@tanstack/react-query";
import { router, useLocalSearchParams, type Href } from "expo-router";
import { Activity, ChevronRight, ClipboardPlus, FileText, HeartPulse, Pencil, Shirt } from "lucide-react-native";
import type { LucideIcon } from "lucide-react-native";
import { useState } from "react";
import { RefreshControl, View } from "react-native";

import { HorseEmergencyPreview } from "@/components/horse-emergency-preview";
import { HorseHealthTiles } from "@/components/horse-health-tiles";
import { HorseObservations } from "@/components/horse-observations";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { HorseRiders } from "@/components/horse-riders";
import { Avatar, Badge, Button, Card, Hero, Icon, PressableCard, Screen, SectionLabel, Text } from "@/components/ui";
import { horseKeys, useDocuments, useEmergency, useHealth, useHorse } from "@/lib/api/horses";
import { ageText, horseRoutes, joinParts, sexLabel } from "@/lib/horse-format";
import { rehaFromObservationRoute } from "@/lib/observations";
import { colors } from "@/lib/theme";

function LinkRow({ icon, label, description, onPress }: { icon: LucideIcon; label: string; description?: string; onPress: () => void }) {
  return (
    <PressableCard
      padded={false}
      onPress={onPress}
      accessibilityLabel={label}
      className="min-h-touch flex-row items-center gap-3 px-4 py-3"
    >
      <Icon as={icon} size={20} className="text-primary" />
      <View className="flex-1">
        <Text variant="bodyStrong">{label}</Text>
        {description ? <Text variant="secondary">{description}</Text> : null}
      </View>
      <Icon as={ChevronRight} size={20} className="text-muted" />
    </PressableCard>
  );
}

/** Horse record ("Pferdeakte", JAN-48). */
export default function HorseRecord() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const queryClient = useQueryClient();
  const horse = useHorse(id);
  const emergency = useEmergency(id);
  const health = useHealth(id);
  const canSeeDocuments = !!horse.data && (horse.data.can_manage || horse.data.i_ride);
  const documents = useDocuments(id, canSeeDocuments);
  const [refreshing, setRefreshing] = useState(false);
  const go = (path: string) => router.push(path as Href);

  async function refresh() {
    setRefreshing(true);
    await queryClient.invalidateQueries({ queryKey: horseKeys.all });
    setRefreshing(false);
  }

  if (!horse.data) {
    return (
      <Screen back>
        {horse.isError ? <HorseError error={horse.error} onRetry={() => horse.refetch()} /> : <HorseLoading />}
      </Screen>
    );
  }

  const h = horse.data;
  const overdue = (health.data?.items ?? []).filter((i) => (i.days_until_due ?? 0) < 0).length;

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={refresh} tintColor={colors.primary.DEFAULT} />
      }
    >
      <Hero
        eyebrow={joinParts([sexLabel(h.sex), ageText(h.birth_year, new Date()), h.breed])}
        title={h.name}
        description={joinParts([h.box ? `Box ${h.box}` : null, h.owner ? `Besitzer: ${h.owner.name}` : null])}
      >
        <View className="flex-row items-center gap-3">
          <Avatar name={h.name} colorKey={h.color_key} size="lg" />
          <View className="flex-1 flex-row flex-wrap gap-2">
            {h.is_mine ? <Badge variant="primary" label="Dein Pferd" /> : null}
            {h.i_ride ? <Badge variant="info" label="Du reitest" /> : null}
            {overdue > 0 ? <Badge variant="danger" label={`${overdue} überfällig`} /> : null}
          </View>
          {h.can_manage ? (
            <Button
              size="icon"
              variant="secondary"
              icon={Pencil}
              accessibilityLabel="Pferd bearbeiten"
              onPress={() => go(horseRoutes.edit(h.id))}
            />
          ) : null}
        </View>
      </Hero>

      <View className="gap-3">
        <SectionLabel>Notfallkarte</SectionLabel>
        {emergency.data ? (
          <HorseEmergencyPreview card={emergency.data} onPress={() => go(horseRoutes.emergency(h.id))} />
        ) : emergency.isError ? (
          <HorseError error={emergency.error} onRetry={() => emergency.refetch()} />
        ) : (
          <HorseLoading />
        )}
      </View>

      <View className="gap-3">
        <SectionLabel
          action={
            <Button label="Alle" variant="ghost" size="sm" onPress={() => go(horseRoutes.health(h.id))} />
          }
        >
          Gesundheit
        </SectionLabel>
        {health.isError ? (
          <HorseError error={health.error} onRetry={() => health.refetch()} />
        ) : (
          <HorseHealthTiles summary={health.data?.summary} onPress={() => go(horseRoutes.health(h.id))} />
        )}
      </View>

      <View className="gap-3">
        <SectionLabel>Auffälligkeiten</SectionLabel>
        <HorseObservations
          horseId={h.id}
          renderActions={(o) =>
            h.can_manage && !o.reha_plan_id ? (
              <Button
                label="Reha-Plan erstellen"
                icon={ClipboardPlus}
                variant="outline"
                size="sm"
                fullWidth
                onPress={() => go(rehaFromObservationRoute(o.id, h.id))}
              />
            ) : null
          }
        />
      </View>

      <View className="gap-3">
        <SectionLabel>Reitbeteiligungen</SectionLabel>
        <HorseRiders horse={h} />
      </View>

      {canSeeDocuments ? (
        <View className="gap-3">
          <SectionLabel>Dokumente</SectionLabel>
          <LinkRow
            icon={FileText}
            label="Alle Dokumente"
            description={
              documents.data
                ? documents.data.length === 0
                  ? "Noch keine Dokumente"
                  : `${documents.data.length} ${documents.data.length === 1 ? "Dokument" : "Dokumente"}`
                : "Equidenpass, Impfpass, Versicherung"
            }
            onPress={() => go(horseRoutes.documents(h.id))}
          />
        </View>
      ) : null}

      <View className="gap-3">
        <SectionLabel>Training und Pflege</SectionLabel>
        <LinkRow icon={Activity} label="Trainingsprofil" onPress={() => go(horseRoutes.trainingProfile(h.id))} />
        <LinkRow icon={Shirt} label="Deckenplan" onPress={() => go(horseRoutes.blanketPlan(h.id))} />
        <LinkRow icon={HeartPulse} label="Reha" onPress={() => go(horseRoutes.reha(h.id))} />
      </View>

      {h.helper_note ? (
        <View className="gap-3">
          <SectionLabel>Hinweise für Helfer</SectionLabel>
          <Card>
            <Text variant="body">{h.helper_note}</Text>
          </Card>
        </View>
      ) : null}
    </Screen>
  );
}
