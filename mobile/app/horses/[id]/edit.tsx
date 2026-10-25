import { router, useLocalSearchParams } from "expo-router";
import { Trash2 } from "lucide-react-native";
import { useState } from "react";
import { Pressable, View } from "react-native";

import { useHorseDeleteSheet } from "@/components/horse-delete-sheet";
import { HorseForm } from "@/components/horse-form";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Icon, PageHeader, Screen, Text } from "@/components/ui";
import { ApiError, errorMessage } from "@/lib/api";
import { horsesApi, useEmergency, useHorse, useHorseMutation, useMembers } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";

export default function EditHorse() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { user } = useAuth();
  const horse = useHorse(id);
  const card = useEmergency(id);
  const members = useMembers();
  const update = useHorseMutation(
    (input: Parameters<typeof horsesApi.update>[1]) => horsesApi.update(id, input),
    () => [{ type: "horseUpdated", horseId: id }],
  );
  const [error, setError] = useState<string | null>(null);
  // Hooks cannot be conditional: the name only matters once the horse has loaded (the row needs can_manage).
  const del = useHorseDeleteSheet({ id, name: horse.data?.name ?? "" });

  const failed = horse.error ?? card.error;
  const canEdit = horse.data?.can_manage ?? false;

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      {horse.data && card.data && canEdit ? (
        <>
          <PageHeader eyebrow={horse.data.name} title="Bearbeiten" description="Änderungen sehen alle im Stall." />
          <HorseForm
            key={horse.data.id}
            horse={horse.data}
            card={card.data}
            isAdmin={!!user?.is_admin}
            members={members.data}
            submitLabel="Speichern"
            saving={update.isPending}
            error={error}
            onSubmit={async (input) => {
              setError(null);
              try {
                await update.mutateAsync(input);
                router.back();
              } catch (err) {
                setError(errorMessage(err));
              }
            }}
          />
          <View className="border-t border-divider pt-2">
            <Pressable
              accessibilityRole="button"
              accessibilityLabel="Pferd löschen"
              onPress={del.open}
              className="min-h-touch flex-row items-center gap-3 py-3 active:opacity-70"
            >
              <Icon as={Trash2} size={20} className="text-danger" />
              <Text variant="body" tone="danger">
                Pferd löschen
              </Text>
            </Pressable>
          </View>
          {del.sheet}
        </>
      ) : failed ? (
        <HorseError error={failed} onRetry={() => void Promise.all([horse.refetch(), card.refetch()])} />
      ) : horse.data && !canEdit ? (
        <HorseError
          error={new ApiError(403, "forbidden", "forbidden")}
          forbiddenText="Nur Besitzer und Admins dürfen bearbeiten."
        />
      ) : (
        <HorseLoading />
      )}
    </Screen>
  );
}
