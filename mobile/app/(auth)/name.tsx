import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { useState } from "react";
import { View } from "react-native";

import { Button, Input, PageHeader, Screen, Text } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { ME_KEY, useAuth } from "@/lib/auth";

/**
 * The display name. After a magic-link sign-up the account only knows the email, so the
 * root layout sends new users here first (`name_confirmed` false). From the settings it
 * edits the name.
 */
export default function Name() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user, hasStable } = useAuth();
  const editing = user?.name_confirmed ?? false;
  const [name, setName] = useState(editing ? (user?.name ?? "") : "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function save() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError("Name fehlt.");
      return;
    }
    setError(null);
    setBusy(true);
    try {
      queryClient.setQueryData(ME_KEY, await api.updateMe({ name: trimmed }));
      if (editing && router.canGoBack()) router.back();
      else router.replace(hasStable ? "/(tabs)" : "/(auth)/join");
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  }

  return (
    <Screen back={editing} keyboardShouldPersistTaps="handled" contentClassName="flex-grow pt-8">
      <PageHeader title="Wie heißt du?" description="So sehen dich die anderen im Stall." />

      <View className="gap-3">
        <Input
          value={name}
          onChangeText={setName}
          placeholder="Vor- und Nachname"
          accessibilityLabel="Name"
          autoComplete="name"
          textContentType="name"
          autoCapitalize="words"
          autoFocus
          maxLength={100}
          returnKeyType="done"
          onSubmitEditing={save}
        />
        <Button label={editing ? "Speichern" : "Weiter"} size="lg" fullWidth loading={busy} onPress={save} />
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
      </View>
    </Screen>
  );
}
