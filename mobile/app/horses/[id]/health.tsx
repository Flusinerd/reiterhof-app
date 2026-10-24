import { router, useLocalSearchParams, type Href } from "expo-router";
import { Check, HandHelping, Layers, Plus } from "lucide-react-native";
import { useState } from "react";
import { Alert, Pressable, View } from "react-native";

import { HorseHealthSheet } from "@/components/horse-health-sheet";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Button, Icon, PageHeader, Screen, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import { errorMessage } from "@/lib/api";
import { horsesApi, useHealth, useHorse, useHorseMutation, type HealthInput, type HealthItem } from "@/lib/api/horses";
import {
  appointmentCompanionRoute,
  dueDetailText,
  dueLevel,
  formatDate,
  healthKindLabel,
  intervalRepeatText,
  joinParts,
  remindText,
} from "@/lib/horse-format";
import { bundleText, horseNamesText } from "@/lib/observations";

const dueTone = { none: "muted", overdue: "danger", soon: "accent", ok: "brass" } as const;

function itemSubtitle(item: HealthItem): string {
  if (item.daily_time) return `täglich um ${item.daily_time} Uhr`;
  if (!item.due_date) return "Kein Termin";
  return joinParts([formatDate(item.due_date), intervalRepeatText(item.interval_count, item.interval_unit)]);
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
      <PageHeader
        eyebrow={horse.data?.name ?? "Pferdeakte"}
        title="Gesundheit"
        description={
          overdue > 0
            ? `${overdue} ${overdue === 1 ? "Termin ist" : "Termine sind"} überfällig.`
            : soon > 0
              ? `${soon} ${soon === 1 ? "Termin" : "Termine"} in den nächsten 7 Tagen.`
              : "Nichts Dringendes."
        }
        action={
          canManage ? (
            <Button
              icon={Plus}
              variant="outline"
              size="icon"
              accessibilityLabel="Termin hinzufügen"
              onPress={() => {
                setError(null);
                setSheet({ item: null });
              }}
            />
          ) : undefined
        }
      />

      {items.length === 0 ? (
        <Text variant="body" tone="muted">
          Noch keine Termine.
        </Text>
      ) : (
        <RowList>
          {items.map((item) => {
            const level = dueLevel(item.days_until_due);
            const top = (
              <>
                <View className="flex-1 gap-0.5">
                  <Text variant="eyebrow">{healthKindLabel(item.kind)}</Text>
                  <Text variant="bodyStrong">{item.label}</Text>
                  <Text variant="bodySm" tone="muted">
                    {itemSubtitle(item)}
                  </Text>
                  {item.due_date && !item.daily_time ? (
                    <Text variant="caption">{remindText(item.remind_days ?? [], item.remind_time ?? "08:00")}</Text>
                  ) : null}
                </View>
                {item.due_date ? (
                  <Text variant="bodySm" tone={dueTone[level]} className="max-w-[40%] text-right font-sans-semibold">
                    {dueDetailText(item.days_until_due)}
                  </Text>
                ) : null}
              </>
            );
            return (
              <View key={item.id} className="gap-2 py-3">
                {canManage ? (
                  <Pressable
                    accessibilityRole="button"
                    accessibilityLabel={`${item.label} bearbeiten`}
                    onPress={() => {
                      setError(null);
                      setSheet({ item });
                    }}
                    className="min-h-[56px] flex-row items-start gap-3 active:opacity-70"
                  >
                    {top}
                  </Pressable>
                ) : (
                  <View className="min-h-[56px] flex-row items-start gap-3">{top}</View>
                )}
                {item.note ? <Text variant="bodySm">{item.note}</Text> : null}
                {item.bundle ? (
                  <View className="flex-row items-start gap-2">
                    <Icon as={Layers} size={18} className="mt-0.5 text-info" />
                    <View className="flex-1 gap-0.5">
                      <Text variant="bodySm" tone="info">
                        {bundleText(item.bundle.count)}
                      </Text>
                      <Text variant="caption">{horseNamesText(item.bundle.horses.map((x) => x.name))}</Text>
                    </View>
                  </View>
                ) : null}
                {canManage || item.kind === "farrier" ? (
                  <View className="-ml-4 flex-row flex-wrap">
                    {canManage && (item.due_date || item.interval_count) ? (
                      <Button
                        label="Erledigt"
                        icon={Check}
                        size="sm"
                        variant="ghost"
                        loading={done.isPending && done.variables === item.id}
                        onPress={() => void markDone(item)}
                      />
                    ) : null}
                    {item.kind === "farrier" ? (
                      <Button
                        label="Termin begleiten"
                        icon={HandHelping}
                        size="sm"
                        variant="ghost"
                        onPress={() => router.push(appointmentCompanionRoute(id) as Href)}
                      />
                    ) : null}
                  </View>
                ) : null}
              </View>
            );
          })}
        </RowList>
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
