import { useEffect, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { FocusRows } from "@/components/training-focus-rows";
import { Button, Sheet, Text } from "@/components/ui";
import { trainingError, useClearMyFocus, useFocuses, useMyFocus, useSetMyFocus } from "@/lib/api/training";
import {
  FOCUS_LIMIT_RIDER,
  focusNote,
  limitNotice,
  selectionCaption,
  toggleKey,
} from "@/lib/training-focus";

/**
 * "Mein Schwerpunkt": what the signed-in person wants to work on with the horse, up to two focuses
 * from the same catalog as the horse's own. The planner prefers them on the days the person rides.
 * "Kein eigener Schwerpunkt" removes them.
 */
export function FocusSheet({
  horseId,
  horseName,
  open,
  onOpenChange,
}: {
  horseId: string;
  horseName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const catalog = useFocuses();
  const mine = useMyFocus(horseId);
  const set = useSetMyFocus(horseId);
  const clear = useClearMyFocus(horseId);
  const [keys, setKeys] = useState<string[]>([]);
  const [touched, setTouched] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const saved = mine.data?.keys ?? [];
  const savedKey = saved.join(",");

  // The sheet starts from what is stored every time it opens (and takes it over when it arrives late).
  useEffect(() => {
    if (!open) return;
    setTouched(false);
    setNotice(null);
    set.reset();
    clear.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);
  useEffect(() => {
    if (!touched) setKeys(saved);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [savedKey, open, touched]);

  const entries = catalog.data ?? [];
  const toggle = (key: string) => {
    const result = toggleKey(keys, key, FOCUS_LIMIT_RIDER);
    setNotice(result.full ? limitNotice(FOCUS_LIMIT_RIDER) : null);
    if (result.full) return;
    setTouched(true);
    setKeys(result.keys);
  };

  // Above the horse's phase: the planner then uses the focus at most once a week.
  const notes: Record<string, string> = {};
  for (const key of keys) {
    const note = focusNote(
      entries.find((e) => e.key === key),
      undefined,
      mine.data?.catalog_phase,
      horseName,
    );
    if (note) notes[key] = note;
  }

  const changed = keys.join(",") !== savedKey;
  const close = () => onOpenChange(false);
  const error = set.isError ? trainingError(set.error) : clear.isError ? trainingError(clear.error) : null;
  const clearAll = () => {
    if (saved.length > 0) return clear.mutateAsync().then(close).catch(() => undefined);
    else {
      setTouched(true);
      setKeys([]);
    }
  };

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title="Mein Schwerpunkt"
      description={`Woran arbeitest du mit ${horseName}? Bis zu ${FOCUS_LIMIT_RIDER}.`}
    >
      <Text variant="secondary">{selectionCaption(keys.length, FOCUS_LIMIT_RIDER)}</Text>
      {notice ? (
        <Text variant="secondary" tone="accent" accessibilityRole="alert">
          {notice}
        </Text>
      ) : null}

      {catalog.isPending ? (
        <ActivityIndicator accessibilityLabel="Lädt" className="self-start" />
      ) : catalog.isError ? (
        <View className="gap-3">
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {trainingError(catalog.error)}
          </Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void catalog.refetch()} />
        </View>
      ) : (
        <FocusRows catalog={entries} selected={keys} onToggle={toggle} notes={notes} />
      )}

      {error ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button
        label="Speichern"
        fullWidth
        disabled={keys.length === 0 || !changed || clear.isPending}
        loading={set.isPending}
        onPress={() => set.mutateAsync(keys).then(close).catch(() => undefined)}
      />
      {saved.length > 0 || keys.length > 0 ? (
        <Button
          label="Kein eigener Schwerpunkt"
          variant="ghost"
          fullWidth
          disabled={set.isPending}
          loading={clear.isPending}
          onPress={clearAll}
        />
      ) : null}
    </Sheet>
  );
}
