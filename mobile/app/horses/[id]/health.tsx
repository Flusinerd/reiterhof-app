import { router, useLocalSearchParams, type Href } from "expo-router";
import { Check, HandHelping, Plus } from "lucide-react-native";
import { useState } from "react";
import { Alert, View } from "react-native";

import { HorseHealthSheet } from "@/components/horse-health-sheet";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Badge, Button, Card, Hero, PressableCard, Screen, SectionLabel, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { horsesApi, useHealth, useHorse, useHorseMutation, type HealthInput, type HealthItem } from "@/lib/api/horses";
import {
  appointmentCompanionRoute,
  dueDetailText,
  dueLevel,
  formatDate,
  healthKindLabel,
} from "@/lib/horse-format";

const badgeVariant = { none: "neutral", overdue: "danger", soon: "accent", ok: "primary" } as const;

function itemSubtitle(item: HealthItem): string {
  if (item.daily_time) return `täglich um ${item.daily_time} Uhr`;
  if (!item.due_date) return "Kein Termin eingetragen";
  return `${formatDate(item.due_date)}${item.interval_days ? ` · alle ${item.interval_days} Tage` : ""}`;
}

/** Due list of a horse (JAN-12): vaccination, farrier, deworming, dentist, physio, medication, vet. */
export default function Health() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const horse = useHorse(id);
  const health = useHealth(id);
  const save = useHorseMutation((v: { itemId: string | null; input: HealthInput }) =>
    v.itemId ? horsesApi.updateHealthItem(v.itemId, v.input) : horsesApi.addHealthItem(id, v.input),
  );
  const done = useHorseMutation((itemId: string) => horsesApi.markDone(itemId));
  const remove = useHorseMutation((itemId: string) => horsesApi.removeHealthItem(itemId));
  const [sheet, setSheet] = useState<{ item: HealthItem | null } | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (!health.data) {
    return (
      <Screen back>
        {health.isError ? <HorseError error={health.error} onRetry={() => health.refetch()} /> : <HorseLoading />}
      </Screen>
    );
  }

  const canManage = health.data.can_manage;
  const items = health.data.items;
  const overdue = items.filter((i) => (i.days_until_due ?? 0) < 0).length;
  const soon = items.filter((i) => dueLevel(i.days_until_due) === "soon").length;

  async function markDone(item: HealthItem) {
    try {
      await done.mutateAsync(item.id);
    } catch (err) {
      Alert.alert("Das hat nicht geklappt", errorMessage(err));
    }
  }

  function confirmDelete(item: HealthItem) {
    Alert.alert("Termin löschen?", `„${item.label}“ wird gelöscht.`, [
      { text: "Abbrechen", style: "cancel" },
      {
        text: "Löschen",
        style: "destructive",
        onPress: () =>
          void remove
            .mutateAsync(item.id)
            .then(() => setSheet(null))
            .catch((err) => setError(errorMessage(err))),
      },
    ]);
  }

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <Hero
        eyebrow={horse.data?.name ?? "Pferdeakte"}
        title="Gesundheit"
        description={
          overdue > 0
            ? `${overdue} ${overdue === 1 ? "Termin ist" : "Termine sind"} überfällig.`
            : soon > 0
              ? `${soon} ${soon === 1 ? "Termin" : "Termine"} in den nächsten 7 Tagen.`
              : "Nichts Dringendes."
        }
      >
        {canManage ? (
          <Button
            label="Termin hinzufügen"
            icon={Plus}
            variant="secondary"
            onPress={() => {
              setError(null);
              setSheet({ item: null });
            }}
          />
        ) : null}
      </Hero>

      <SectionLabel>Termine</SectionLabel>
      {items.length === 0 ? (
        <Card>
          <Text variant="secondary">Noch keine Termine eingetragen.</Text>
        </Card>
      ) : (
        <View className="gap-3">
          {items.map((item) => {
            const level = dueLevel(item.days_until_due);
            const body = (
              <>
                <View className="flex-row items-start justify-between gap-3">
                  <View className="flex-1">
                    <Text variant="secondary">{healthKindLabel(item.kind)}</Text>
                    <Text variant="bodyStrong">{item.label}</Text>
                    <Text variant="secondary">{itemSubtitle(item)}</Text>
                  </View>
                  {item.due_date ? (
                    <Badge variant={badgeVariant[level]} label={dueDetailText(item.days_until_due)} />
                  ) : null}
                </View>
                {item.note ? <Text variant="bodySm">{item.note}</Text> : null}
                {canManage || item.kind === "farrier" ? (
                  <View className="flex-row flex-wrap gap-2">
                    {canManage && (item.due_date || item.interval_days) ? (
                      <Button
                        label="Erledigt"
                        icon={Check}
                        size="sm"
                        variant="secondary"
                        loading={done.isPending && done.variables === item.id}
                        onPress={() => void markDone(item)}
                      />
                    ) : null}
                    {item.kind === "farrier" ? (
                      <Button
                        label="Termin begleiten"
                        icon={HandHelping}
                        size="sm"
                        variant="outline"
                        onPress={() => router.push(appointmentCompanionRoute(id) as Href)}
                      />
                    ) : null}
                  </View>
                ) : null}
              </>
            );
            return canManage ? (
              <PressableCard
                key={item.id}
                className="gap-3"
                accessibilityLabel={`${item.label} bearbeiten`}
                onPress={() => {
                  setError(null);
                  setSheet({ item });
                }}
              >
                {body}
              </PressableCard>
            ) : (
              <Card key={item.id} className="gap-3">
                {body}
              </Card>
            );
          })}
        </View>
      )}

      {!canManage ? (
        <Text variant="caption">Nur Besitzer und Admins können Termine ändern.</Text>
      ) : null}

      <HorseHealthSheet
        open={sheet !== null}
        onOpenChange={(open) => !open && setSheet(null)}
        item={sheet?.item ?? null}
        saving={save.isPending}
        error={error}
        onSave={async (input) => {
          setError(null);
          try {
            await save.mutateAsync({ itemId: sheet?.item?.id ?? null, input });
            setSheet(null);
          } catch (err) {
            setError(errorMessage(err));
          }
        }}
        onDelete={sheet?.item ? () => confirmDelete(sheet.item as HealthItem) : undefined}
      />
    </Screen>
  );
}
