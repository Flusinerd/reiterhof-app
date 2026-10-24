import { useMutation } from "@tanstack/react-query";
import { UserPlus } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import { Button, Text } from "@/components/ui";
import { api, errorMessage } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { inviteExpiry, inviteMessage, shareInvite, webUrl, type Invite } from "@/lib/invite";

/** Admins only: creates an invite code and hands it to the share sheet. */
export function InviteCard() {
  const { me } = useAuth();
  const [invite, setInvite] = useState<Invite | null>(null);
  const [note, setNote] = useState<{ text: string; error: boolean } | null>(null);
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";
  const stableName = me?.stable?.name ?? "Stall";

  async function share(current: Invite) {
    const result = await shareInvite(inviteMessage(current, stableName, webUrl(), timeZone));
    if (result === "copied") setNote({ text: "Text kopiert. Füg ihn in eine Nachricht ein.", error: false });
  }

  const create = useMutation({
    mutationFn: () => api.createInvite(),
    onMutate: () => setNote(null),
    onSuccess: async (created) => {
      setInvite(created);
      await share(created).catch(() => undefined);
    },
    onError: (e) => setNote({ text: errorMessage(e), error: true }),
  });

  return (
    <View className="gap-3">
      <View className="gap-1">
        <Text variant="bodyStrong">Leute einladen</Text>
        <Text variant="secondary">Der Code gilt 7 Tage für bis zu 10 Personen.</Text>
      </View>
      {invite ? (
        <View className="items-center gap-1 rounded-tile border border-border px-4 py-4">
          <Text selectable className="font-display text-[32px] leading-[38px] tracking-widest text-brass">
            {invite.code}
          </Text>
          <Text variant="caption">Gültig bis {inviteExpiry(invite.expires_at, timeZone)}</Text>
        </View>
      ) : null}
      <Button
        label={invite ? "Nochmal teilen" : "Einladung teilen"}
        icon={UserPlus}
        variant={invite ? "outline" : "primary"}
        fullWidth
        loading={create.isPending}
        onPress={() => (invite ? void share(invite).catch(() => undefined) : create.mutate())}
      />
      {note ? (
        <Text variant="bodySm" tone={note.error ? "danger" : "success"} accessibilityRole={note.error ? "alert" : undefined}>
          {note.text}
        </Text>
      ) : null}
    </View>
  );
}
