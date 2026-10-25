import { useRouter } from "expo-router";
import { useEffect, useMemo, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { Button, PageHeader, Pill, Screen, Switch, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import { trainingError, useSaveTrainingEquipment, useTrainingEquipment } from "@/lib/api/training";
import { useAuth } from "@/lib/auth";
import {
  LOCATIONS,
  equipmentLabel,
  facilityKeysOf,
  locationSummary,
  locationsWithDefaults,
  sameEquipment,
  toggleEquipment,
  type LocationEquipment,
  type LocationKey,
} from "@/lib/training-equipment";

type Draft = Record<LocationKey, LocationEquipment>;

/**
 * What the stable has to set up for training, per place: poles, cavaletti, jumps and a calmness
 * course, and whether the place is shared (then set-up needs an agreement). The weekly plan only
 * uses exercises that fit. Admins edit it; everybody else reads it.
 */
export default function TrainingEquipment() {
  const router = useRouter();
  const { me, user } = useAuth();
  const query = useTrainingEquipment();
  const save = useSaveTrainingEquipment();
  const [draft, setDraft] = useState<Draft | null>(null);

  const data = query.data;
  const facility = facilityKeysOf(data);
  const stored = useMemo(() => (data ? locationsWithDefaults(data.locations, facility) : null), [data]);
  useEffect(() => {
    if (stored) setDraft((d) => d ?? stored);
  }, [stored]);

  const editable = !!user?.is_admin;

  if (query.isError) {
    return (
      <Screen back>
        <View className="gap-3">
          <Text variant="secondary">{trainingError(query.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void query.refetch()} />
        </View>
      </Screen>
    );
  }
  if (!data || !draft || !stored) {
    return (
      <Screen back>
        <View className="items-center py-16">
          <ActivityIndicator accessibilityLabel="Lädt" />
        </View>
      </Screen>
    );
  }

  const patch = (key: LocationKey, next: LocationEquipment) => {
    save.reset();
    setDraft((d) => (d ? { ...d, [key]: next } : d));
  };
  const changed = !sameEquipment(draft, stored);

  return (
    <Screen back>
      <PageHeader
        eyebrow={me?.stable?.name}
        title="Ausstattung"
        description="Was gibt es bei euch zum Aufbauen? Der Wochenplan nimmt nur Übungen, die dazu passen."
      />

      {!data.set ? (
        <Text variant="secondary">
          {editable
            ? "Noch nichts eingetragen. Bis dahin plant die App mit den Standardwerten, die unten stehen."
            : "Der Stall hat noch nichts eingetragen. Bis dahin plant die App mit Standardwerten."}
        </Text>
      ) : null}

      <RowList>
        {LOCATIONS.map(({ key, label }) => {
          const location = draft[key];
          return (
            <View key={key} className="gap-3 py-4">
              <Text variant="bodyStrong">{label}</Text>
              {editable ? (
                <>
                  <View className="flex-row flex-wrap gap-2">
                    {facility.map((k) => (
                      <Pill
                        key={k}
                        label={equipmentLabel(k)}
                        selected={location.equipment.includes(k)}
                        onPress={() => patch(key, toggleEquipment(location, k, facility))}
                      />
                    ))}
                  </View>
                  <Switch
                    label="Geteilt – Aufbau nur nach Absprache"
                    accessibilityLabel={`${label}: geteilt, Aufbau nur nach Absprache`}
                    value={location.shared}
                    onValueChange={(shared) => patch(key, { ...location, shared })}
                  />
                </>
              ) : (
                <Text variant="secondary">{locationSummary(location)}</Text>
              )}
            </View>
          );
        })}
      </RowList>

      <Text variant="caption">Ausritte und die Führanlage brauchen keinen Aufbau.</Text>

      {editable ? (
        <View className="gap-3">
          {save.isError ? (
            <Text variant="secondary" tone="danger" accessibilityRole="alert">
              {trainingError(save.error)}
            </Text>
          ) : null}
          <Button
            label="Speichern"
            size="lg"
            fullWidth
            disabled={!changed && data.set}
            loading={save.isPending}
            onPress={() => save.mutateAsync({ locations: draft, facility }).then(() => router.back()).catch(() => undefined)}
          />
        </View>
      ) : (
        <Text variant="secondary">Nur Admins des Stalls können die Ausstattung ändern.</Text>
      )}
    </Screen>
  );
}
