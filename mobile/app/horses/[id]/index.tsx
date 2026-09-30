import { useQueryClient } from "@tanstack/react-query";
import { router, useLocalSearchParams, type Href } from "expo-router";
import { Activity, ClipboardPlus, FileText, HeartPulse, Pencil, Shirt } from "lucide-react-native";
import { useState } from "react";
import { RefreshControl, View } from "react-native";

import { HorseEmergencyPreview } from "@/components/horse-emergency-preview";
import { HorseHealthTiles } from "@/components/horse-health-tiles";
import { HorseObservations } from "@/components/horse-observations";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { HorseRiders } from "@/components/horse-riders";
import { Avatar, Badge, Button, LinkRow, PageHeader, Screen, Section, Text } from "@/components/ui";
import { horseKeys, useDocuments, useEmergency, useHealth, useHorse } from "@/lib/api/horses";
import { ageText, horseRoutes, joinParts, sexLabel } from "@/lib/horse-format";
import { rehaFromObservationRoute } from "@/lib/observations";
import { colors } from "@/lib/theme";

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
      <PageHeader
        eyebrow={joinParts([sexLabel(h.sex), ageText(h.birth_year, new Date()), h.breed])}
        title={h.name}
        description={joinParts([h.box ? `Box ${h.box}` : null, h.owner ? `Besitzer: ${h.owner.name}` : null])}
        action={<Avatar name={h.name} colorKey={h.color_key} size="lg" />}
      >
        {h.is_mine || h.i_ride || overdue > 0 || h.can_manage ? (
          <View className="flex-row flex-wrap items-center gap-2">
            {h.is_mine ? <Badge variant="primary" label="Dein Pferd" /> : null}
            {h.i_ride ? <Badge variant="info" label="Du reitest" /> : null}
            {overdue > 0 ? <Badge variant="danger" label={`${overdue} überfällig`} /> : null}
            {h.can_manage ? (
              <Button
                label="Bearbeiten"
                size="sm"
                variant="ghost"
                icon={Pencil}
                accessibilityLabel="Pferd bearbeiten"
                onPress={() => go(horseRoutes.edit(h.id))}
              />
            ) : null}
          </View>
        ) : null}
      </PageHeader>

      <Section title="Notfallkarte">
        {emergency.data ? (
          <HorseEmergencyPreview card={emergency.data} onPress={() => go(horseRoutes.emergency(h.id))} />
        ) : emergency.isError ? (
          <HorseError error={emergency.error} onRetry={() => emergency.refetch()} />
        ) : (
          <HorseLoading />
        )}
      </Section>

      <Section
        title="Gesundheit"
        action={<Button label="Alle" variant="ghost" size="sm" onPress={() => go(horseRoutes.health(h.id))} />}
      >
        {health.isError ? (
          <HorseError error={health.error} onRetry={() => health.refetch()} />
        ) : (
          <HorseHealthTiles summary={health.data?.summary} onPress={() => go(horseRoutes.health(h.id))} />
        )}
      </Section>

      <Section title="Auffälligkeiten">
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
      </Section>

      <Section title="Reitbeteiligungen">
        <HorseRiders horse={h} />
      </Section>

      {canSeeDocuments ? (
        <Section title="Dokumente">
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
        </Section>
      ) : null}

      <Section title="Training und Pflege">
        <LinkRow icon={Activity} label="Trainingsprofil" onPress={() => go(horseRoutes.trainingProfile(h.id))} />
        <LinkRow icon={Shirt} label="Deckenplan" onPress={() => go(horseRoutes.blanketPlan(h.id))} />
        <LinkRow icon={HeartPulse} label="Reha" onPress={() => go(horseRoutes.reha(h.id))} />
      </Section>

      {h.helper_note ? (
        <Section title="Hinweise für Helfer">
          <Text variant="body">{h.helper_note}</Text>
        </Section>
      ) : null}
    </Screen>
  );
}
