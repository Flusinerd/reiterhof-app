import { useEffect, useState } from "react";
import { View } from "react-native";

import { Button, Input, Pill, Sheet, Text } from "@/components/ui";
import { PHASE_ACTIVITIES, checkPhase, type PhaseDraft } from "@/lib/reha";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The phase to edit, or null for a new one. */
  initial: PhaseDraft | null;
  /** 1-based position, used in the title and the messages. */
  position: number;
  onSave: (phase: PhaseDraft) => void;
};

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <View className="gap-1">
      <Text variant="secondary">{label}</Text>
      {children}
    </View>
  );
}

/** Bottom sheet that edits one phase of a reha plan (name, activity, days, minutes, conditions). */
export function RehaPhaseSheet({ open, onOpenChange, initial, position, onSave }: Props) {
  const [draft, setDraft] = useState<PhaseDraft | null>(initial);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setDraft(initial);
      setError(null);
    }
  }, [open, initial]);

  if (!draft) return null;
  const patch = (p: Partial<PhaseDraft>) => setDraft({ ...draft, ...p });
  const rest = draft.activity === "rest";

  const save = () => {
    const r = checkPhase(draft, position);
    if (!r.ok) {
      setError(r.error);
      return;
    }
    onSave(draft);
    onOpenChange(false);
  };

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title={`Phase ${position}`}
      description="Die Minuten steigen von „Von“ am ersten bis „Bis“ am letzten Tag der Phase."
      className="gap-4"
    >
      <Field label="Name">
        <Input
          accessibilityLabel="Name der Phase"
          placeholder="z. B. Schritt führen"
          value={draft.name}
          maxLength={80}
          onChangeText={(name) => patch({ name })}
        />
      </Field>

      <Field label="Was ist erlaubt?">
        <View className="flex-row flex-wrap gap-2">
          {PHASE_ACTIVITIES.map((a) => (
            <Pill key={a.value} label={a.label} selected={draft.activity === a.value} onPress={() => patch({ activity: a.value })} />
          ))}
        </View>
      </Field>

      <Field label="Dauer in Tagen">
        <Input
          accessibilityLabel="Dauer in Tagen"
          keyboardType="number-pad"
          value={draft.days}
          maxLength={3}
          onChangeText={(days) => patch({ days })}
        />
      </Field>

      {rest ? (
        <Text variant="secondary">Boxenruhe: In dieser Phase wird das Pferd nicht bewegt.</Text>
      ) : (
        <View className="flex-row gap-3">
          <View className="flex-1">
            <Field label="Von (Minuten)">
              <Input
                accessibilityLabel="Minuten am ersten Tag"
                keyboardType="number-pad"
                value={draft.minMinutes}
                maxLength={3}
                onChangeText={(minMinutes) => patch({ minMinutes })}
              />
            </Field>
          </View>
          <View className="flex-1">
            <Field label="Bis (Minuten)">
              <Input
                accessibilityLabel="Minuten am letzten Tag"
                keyboardType="number-pad"
                value={draft.maxMinutes}
                maxLength={3}
                onChangeText={(maxMinutes) => patch({ maxMinutes })}
              />
            </Field>
          </View>
        </View>
      )}

      <Field label="Bedingungen (optional)">
        <Input
          accessibilityLabel="Bedingungen"
          placeholder="z. B. nur auf festem Boden"
          multiline
          textAlignVertical="top"
          className="h-auto min-h-20 py-3"
          value={draft.conditions}
          maxLength={500}
          onChangeText={(conditions) => patch({ conditions })}
        />
      </Field>

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button label="Übernehmen" fullWidth onPress={save} />
    </Sheet>
  );
}
