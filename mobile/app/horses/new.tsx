import { router, type Href } from "expo-router";
import { useState } from "react";

import { HorseForm } from "@/components/horse-form";
import { Hero, Screen } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { horsesApi, useHorseMutation, useMembers } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";
import { horseRoutes } from "@/lib/horse-format";

export default function NewHorse() {
  const { user } = useAuth();
  const members = useMembers();
  const create = useHorseMutation(horsesApi.create);
  const [error, setError] = useState<string | null>(null);

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <Hero
        title="Pferd anlegen"
        description="Du wirst als Besitzer eingetragen."
      />
      <HorseForm
        isAdmin={!!user?.is_admin}
        members={members.data}
        submitLabel="Anlegen"
        saving={create.isPending}
        error={error}
        onSubmit={async (input) => {
          setError(null);
          try {
            const horse = await create.mutateAsync(input);
            router.replace(horseRoutes.detail(horse.id) as Href);
          } catch (err) {
            setError(errorMessage(err));
          }
        }}
      />
    </Screen>
  );
}
