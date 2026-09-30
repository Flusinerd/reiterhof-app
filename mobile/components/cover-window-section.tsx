import { Minus, Plus } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import { Button, Card, Text } from "@/components/ui";
import { blanketsApi, usePlanMutation } from "@/lib/api/blankets";
import {
  COVER_END_MAX,
  COVER_END_MIN,
  COVER_START_MAX,
  COVER_START_MIN,
  blanketErrorMessage,
  stepCoverTime,
  type CoverWindowField,
} from "@/lib/blankets";

const FIELDS: { field: CoverWindowField; label: string; min: string; max: string }[] = [
  { field: "start", label: "Eindecken ab", min: COVER_START_MIN, max: COVER_START_MAX },
  { field: "end", label: "Abdecken bis", min: COVER_END_MIN, max: COVER_END_MAX },
];

/**
 * Deckenzeitraum of a horse (`horses.cover_start` to `horses.cover_end`): the time the horse is covered and
 * the forecast is summarised for. Owners and admins change both ends in quarter-hour steps and save;
 * everybody else only sees it.
 */
export function CoverWindowSection({
  horseId,
  start,
  end,
  canManage,
}: {
  horseId: string;
  start: string;
  end: string;
  canManage: boolean;
}) {
  const [draft, setDraft] = useState<{ start: string; end: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const save = usePlanMutation((w: { start: string; end: string }) => blanketsApi.saveCoverWindow(horseId, w.start, w.end));

  const shown = draft ?? { start, end };
  const changed = draft !== null && (draft.start !== start || draft.end !== end);

  function submit() {
    setError(null);
    save.mutate(shown, {
      onSuccess: () => setDraft(null),
      onError: (e) => setError(blanketErrorMessage(e)),
    });
  }

  return (
    <Card className="gap-4">
      <View className="gap-1">
        <Text variant="bodyStrong">Deckenzeitraum</Text>
        <Text variant="secondary">
          Wetter und Empfehlung gelten für diesen Zeitraum.
        </Text>
      </View>
      {canManage ? (
        <>
          {FIELDS.map(({ field, label, min, max }) => (
            <View key={field} className="gap-2">
              <Text variant="secondary">{label}</Text>
              <View className="flex-row items-center justify-between gap-4">
                <Button
                  size="icon"
                  variant="outline"
                  icon={Minus}
                  accessibilityLabel={`${label}: 15 Minuten früher`}
                  disabled={shown[field] === min}
                  onPress={() => setDraft({ ...shown, [field]: stepCoverTime(field, shown[field], -1) })}
                />
                <Text variant="heroNumberSm" accessibilityLabel={`${shown[field]} Uhr`}>
                  {shown[field]}
                </Text>
                <Button
                  size="icon"
                  variant="outline"
                  icon={Plus}
                  accessibilityLabel={`${label}: 15 Minuten später`}
                  disabled={shown[field] === max}
                  onPress={() => setDraft({ ...shown, [field]: stepCoverTime(field, shown[field], 1) })}
                />
              </View>
            </View>
          ))}
          {error ? (
            <Text variant="bodySm" tone="danger" accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
          {changed ? (
            <View className="flex-row gap-3">
              <Button label="Speichern" loading={save.isPending} onPress={submit} />
              <Button label="Verwerfen" variant="ghost" disabled={save.isPending} onPress={() => setDraft(null)} />
            </View>
          ) : null}
        </>
      ) : (
        <Text variant="body">
          <Text variant="bodyStrong">{start} bis {end} Uhr</Text>. Nur Besitzer und Admins können das ändern.
        </Text>
      )}
    </Card>
  );
}
