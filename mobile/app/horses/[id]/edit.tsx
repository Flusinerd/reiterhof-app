import { router, useLocalSearchParams } from "expo-router";
import { useState } from "react";

import { HorseForm } from "@/components/horse-form";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Hero, Screen } from "@/components/ui";
import { ApiError, errorMessage } from "@/lib/api";
import { horsesApi, useEmergency, useHorse, useHorseMutation, useMembers } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";

export default function EditHorse() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { user } = useAuth();
  const horse = useHorse(id);
  const card = useEmergency(id);
  const members = useMembers();
  const update = useHorseMutation((input: Parameters<typeof horsesApi.update>[1]) => horsesApi.update(id, input));
  const [error, setError] = useState<string | null>(null);

  const failed = horse.error ?? card.error;
  const canEdit = horse.data?.can_manage ?? false;

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      {horse.data && card.data && canEdit ? (
        <>
          <Hero title={`${horse.data.name} bearbeiten`} description="Änderungen sehen alle im Stall." />
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
