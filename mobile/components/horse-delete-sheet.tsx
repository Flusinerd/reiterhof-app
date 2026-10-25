import { Trash2 } from "lucide-react-native";
import { useState, type ReactElement } from "react";
import { View } from "react-native";

import { Button, Input, Sheet, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { horsesApi, useHorseMutation, type Horse } from "@/lib/api/horses";
import { flash } from "@/lib/flash";
import { deletedHorses, deletedMessage, deleteSheetText, nameMatches } from "@/lib/horse-delete";
import { leaveDeletedHorse } from "@/lib/navigation";

/**
 * "Pferd löschen" (owner and admin): a sheet that says what disappears, asks for the horse's name and
 * deletes it for good. On success the user leaves every screen of the horse for Heute, which shows "Luna
 * wurde gelöscht." (`flash`); the cache forgets the horse through the `horseDeleted` effect. Render
 * `sheet` once in the screen and call `open()` from the row.
 */
export function useHorseDeleteSheet(horse: Pick<Horse, "id" | "name">): { open: () => void; sheet: ReactElement } {
  const [visible, setVisible] = useState(false);
  const [typed, setTyped] = useState("");
  const [error, setError] = useState<string | null>(null);
  const text = deleteSheetText(horse.name);

  const remove = useHorseMutation(
    async (confirmName: string) => {
      // Before the call: the realtime event of this very delete may arrive before the answer.
      deletedHorses.mark(horse.id);
      try {
        await horsesApi.remove(horse.id, confirmName);
      } catch (err) {
        deletedHorses.unmark(horse.id);
        throw err;
      }
    },
    () => [{ type: "horseDeleted", horseId: horse.id }],
    {
      // Leave first (before the cache is cleaned), so no screen of the horse is on top when its queries go.
      onSuccess: () => {
        flash.set(deletedMessage(horse.name));
        leaveDeletedHorse();
      },
    },
  );
  const busy = remove.isPending;
  const matches = nameMatches(horse.name, typed);

  async function submit() {
    if (!matches || busy) return;
    setError(null);
    try {
      await remove.mutateAsync(typed.trim());
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  const sheet = (
    <Sheet
      open={visible}
      onOpenChange={(open) => !busy && setVisible(open)} // not while it runs: an error would be invisible
      title={text.title}
    >
      <View className="gap-4">
        <Text variant="body">{text.scope}</Text>
        <Text variant="bodyStrong" tone="danger">
          {text.warning}
        </Text>
        <View className="gap-2">
          <Text variant="bodySm" tone="muted">
            {text.fieldLabel}
          </Text>
          <Input
            value={typed}
            onChangeText={setTyped}
            accessibilityLabel={text.fieldLabel}
            placeholder={text.placeholder}
            autoCapitalize="none"
            autoCorrect={false}
            editable={!busy}
            returnKeyType="done"
            onSubmitEditing={submit}
          />
        </View>
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        <Button
          label={text.confirm}
          icon={Trash2}
          variant="danger"
          fullWidth
          disabled={!matches}
          loading={busy}
          onPress={submit}
        />
        <Button label={text.cancel} variant="ghost" fullWidth disabled={busy} onPress={() => setVisible(false)} />
      </View>
    </Sheet>
  );

  return {
    open: () => {
      setTyped("");
      setError(null);
      setVisible(true);
    },
    sheet,
  };
}
