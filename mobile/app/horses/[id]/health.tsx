import { router, useLocalSearchParams, type Href } from "expo-router";
import { Check, HandHelping, Layers, Plus, Undo2 } from "lucide-react-native";
import { useState } from "react";
import { Alert, Pressable, View } from "react-native";

import { HorseHealthSheet } from "@/components/horse-health-sheet";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Button, Icon, PageHeader, Screen, Sheet, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import { errorMessage } from "@/lib/api";
import { horseKeys, horsesApi, useHealth, useHorse, useHorseMutation, type HealthInput, type HealthItem } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";
import { canMarkDone, doneLabel, doneState, endCourseSheet, healthErrorMessage, isCourse } from "@/lib/health-done";
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
import type { Effect } from "@/lib/query-effects";
import { mutationKeys } from "@/lib/query-keys";
import { useFrozenOrder } from "@/lib/use-frozen-order";
import { usePullToRefresh } from "@/lib/use-pull-to-refresh";
import { useRefreshOnFocus } from "@/lib/use-refresh-on-focus";

const dueTone = { none: "muted", overdue: "danger", soon: "accent", ok: "brass" } as const;

/** Every write to a health item changes the same things (lists, reminder center, Heute). */
const healthChanged = (): Effect[] => [{ type: "health" }];

function itemSubtitle(item: HealthItem): string {
  if (item.daily_time) return `täglich um ${item.daily_time} Uhr`;
  if (!item.due_date) return "Kein Termin";
  return joinParts([formatDate(item.due_date), intervalRepeatText(item.interval_count, item.interval_unit)]);
}

/** Due list of a horse (JAN-12): vaccination, farrier, deworming, dentist, physio, medication, vet. */
export default function Health() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { user } = useAuth();
  const horse = useHorse(id);
  const health = useHealth(id);
  const items = useFrozenOrder(health.data?.items ?? []);
  useRefreshOnFocus([horseKeys.health(id), horseKeys.detail(id)]);
  const refresh = usePullToRefresh([health, horse]);
  const save = useHorseMutation(
    (v: { itemId: string | null; input: HealthInput }) =>
      v.itemId ? horsesApi.updateHealthItem(v.itemId, v.input) : horsesApi.addHealthItem(id, v.input),
    healthChanged,
  );
  // "Erledigt" moves the date once per day on the server; it also refreshes after a conflict (somebody else was first).
  const done = useHorseMutation((item: HealthItem) => horsesApi.markDone(item.id, item.due_date), healthChanged, {
    mutationKey: mutationKeys.healthDone,
    errorEffects: healthChanged,
  });
  const undo = useHorseMutation((item: HealthItem) => horsesApi.undoDone(item.id), healthChanged, {
    mutationKey: mutationKeys.healthUndo,
    errorEffects: healthChanged,
  });
  const remove = useHorseMutation((itemId: string) => horsesApi.removeHealthItem(itemId), healthChanged);
  const [sheet, setSheet] = useState<{ item: HealthItem | null } | null>(null);
  const [ending, setEnding] = useState<HealthItem | null>(null);
  const [error, setError] = useState<string | null>(null);
  /** The error of "Erledigt" or "Rückgängig", shown under the row that was tapped. */
  const [rowError, setRowError] = useState<{ id: string; text: string } | null>(null);

  if (!health.data) {
    return (
      <Screen back>
        {health.isError ? <HorseError error={health.error} onRetry={() => health.refetch()} /> : <HorseLoading />}
      </Screen>
    );
  }

  const canManage = health.data.can_manage;
  const today = health.data.today;
  const overdue = items.filter((i) => (i.days_until_due ?? 0) < 0).length;
  const soon = items.filter((i) => dueLevel(i.days_until_due) === "soon").length;

  /** Runs the call and shows its error under the row. The returned promise keeps the button busy until the list is fresh. */
  async function run(item: HealthItem, call: () => Promise<unknown>) {
    setRowError(null);
    try {
      await call();
    } catch (err) {
      setRowError({ id: item.id, text: healthErrorMessage(err) });
    }
  }

  /** "Erledigt"; ending a medication course asks first. */
  function markDone(item: HealthItem) {
    if (isCourse(item)) {
      setEnding(item);
      return;
    }
    return run(item, () => done.mutateAsync(item));
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
    <Screen back keyboardShouldPersistTaps="handled" refreshControl={refresh}>
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
            const state = doneState(item, today, user?.id);
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
                {rowError?.id === item.id ? (
                  <Text variant="bodySm" tone="danger" accessibilityRole="alert">
                    {rowError.text}
                  </Text>
                ) : null}
                {canManage || item.kind === "farrier" ? (
                  <View className="-ml-4 flex-row flex-wrap">
                    {state ? (
                      <View className="ml-4 flex-row flex-wrap items-center gap-3">
                        <View className="flex-row items-center gap-2">
                          <Icon as={Check} size={18} className="text-success" />
                          <View>
                            <Text variant="bodyStrong" tone="success">
                              {state.title}
                            </Text>
                            <Text variant="caption">{state.detail}</Text>
                          </View>
                        </View>
                        {canManage ? (
                          <Button
                            label="Rückgängig"
                            icon={Undo2}
                            size="sm"
                            variant="ghost"
                            onPress={() => run(item, () => undo.mutateAsync(item))}
                          />
                        ) : null}
                      </View>
                    ) : canManage && canMarkDone(item) ? (
                      <Button
                        label={doneLabel(item)}
                        icon={Check}
                        size="sm"
                        variant="ghost"
                        onPress={() => markDone(item)}
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

      <Sheet
        open={ending !== null}
        onOpenChange={(open) => !open && setEnding(null)}
        title={ending ? endCourseSheet(ending).title : undefined}
        description={ending ? endCourseSheet(ending).description : undefined}
      >
        <View className="gap-3">
          <Button
            label={ending ? endCourseSheet(ending).confirm : "Gabe beenden"}
            variant="danger"
            fullWidth
            onPress={() => {
              const item = ending;
              setEnding(null);
              if (item) return run(item, () => done.mutateAsync(item));
            }}
          />
          <Button label="Zurück" variant="ghost" fullWidth onPress={() => setEnding(null)} />
        </View>
      </Sheet>
    </Screen>
  );
}
