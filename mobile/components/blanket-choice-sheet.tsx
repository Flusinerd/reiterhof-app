import { Check } from "lucide-react-native";
import { useEffect, useState } from "react";
import { Pressable, View } from "react-native";

import { BlanketPhoto } from "@/components/blanket-photo";
import { Badge, Button, Icon, Input, Pill, Sheet, Text } from "@/components/ui";
import { blanketsApi, useSetState, usePlan, usePlanMutation } from "@/lib/api/blankets";
import {
  blanketChoices,
  blanketErrorMessage,
  fillLabel,
  OVERRIDE_DURATIONS,
  overrideUntilText,
  recommendationTitle,
  type Blanket,
} from "@/lib/blankets";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  horseId: string;
  /**
   * "override": owner or admin pick the blanket of the coming nights by hand (deviating from the
   * rules) or go back to the rules. "cover": anybody records "Eingedeckt" with a blanket of their choice.
   */
  mode: "override" | "cover";
};

/** One selectable row: photo, name, filling and whether the rules name it. */
function ChoiceRow({
  blanket,
  label,
  selected,
  isRule,
  onPress,
}: {
  blanket: Blanket | null;
  label: string;
  selected: boolean;
  isRule: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="radio"
      accessibilityState={{ selected }}
      accessibilityLabel={isRule ? `${label}, laut Regel` : label}
      onPress={onPress}
      className={
        selected
          ? "min-h-touch flex-row items-center gap-3 rounded-tile border border-primary bg-primary-soft p-2"
          : "min-h-touch flex-row items-center gap-3 rounded-tile border border-border bg-card p-2"
      }
    >
      <BlanketPhoto url={blanket?.photo_url ?? null} size={48} />
      <View className="flex-1">
        <Text variant="bodyStrong">{label}</Text>
        {blanket ? <Text variant="caption">{fillLabel(blanket.fill_g)}</Text> : null}
      </View>
      {isRule ? <Badge variant="neutral" label="Regel" /> : null}
      {selected ? <Icon as={Check} size={20} className="text-primary" /> : null}
    </Pressable>
  );
}

/** Pick a blanket by hand, deviating from the rules (owner/admin), or cover with another blanket (anybody). */
export function BlanketChoiceSheet({ open, onOpenChange, horseId, mode }: Props) {
  const plan = usePlan(horseId);
  const setState = useSetState();
  const save = usePlanMutation((v: { blanketId: string | null; nights: number | null; note: string }) =>
    blanketsApi.setOverride(horseId, v.blanketId, v.nights, v.note),
  );
  const reset = usePlanMutation(() => blanketsApi.clearOverride(horseId));
  // undefined = nothing picked yet, null = "Keine Decke".
  const [picked, setPicked] = useState<string | null | undefined>(undefined);
  const [nights, setNights] = useState<number | null>(1);
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | null>(null);

  const p = plan.data;
  const ruleRec = p?.rule_recommendation ?? p?.recommendation;
  const ruleBlanketId = ruleRec?.status === "blanket" ? (ruleRec.blanket?.id ?? null) : null;
  const ruleSaysNone = ruleRec?.status === "none";

  useEffect(() => {
    if (!open) return;
    setError(null);
    const o = p?.override ?? null;
    if (mode === "override" && o) {
      setPicked(o.blanket_id);
      setNights(o.day_until === null ? null : 1);
      setNote(o.note);
    } else {
      setPicked(mode === "cover" ? (p?.recommendation.blanket?.id ?? undefined) : undefined);
      setNights(1);
      setNote("");
    }
    // Only when the sheet opens; later plan refreshes must not reset the form.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const choices = blanketChoices(p?.blankets ?? [], ruleBlanketId);

  async function submit() {
    if (picked === undefined) return setError("Bitte eine Decke wählen.");
    setError(null);
    try {
      if (mode === "cover") {
        if (picked === null) return;
        await setState.mutateAsync({ horseId, action: "covered", coveredWith: picked });
      } else {
        await save.mutateAsync({ blanketId: picked, nights, note: note.trim() });
      }
      onOpenChange(false);
    } catch (e) {
      setError(blanketErrorMessage(e));
    }
  }

  async function backToRule() {
    setError(null);
    try {
      await reset.mutateAsync();
      onOpenChange(false);
    } catch (e) {
      setError(blanketErrorMessage(e));
    }
  }

  const description = ruleRec
    ? `Laut Regel: ${recommendationTitle(ruleRec)}.`
    : undefined;

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title={mode === "override" ? "Decke von Hand wählen" : "Mit anderer Decke eindecken"}
      description={description}
    >
      {!p ? (
        <Text variant="body" tone="muted">
          {plan.isError ? blanketErrorMessage(plan.error) : "Wird geladen ..."}
        </Text>
      ) : (
        <>
          <View className="gap-2" accessibilityRole="radiogroup">
            {choices.map((b) => (
              <ChoiceRow
                key={b.id}
                blanket={b}
                label={b.name}
                selected={picked === b.id}
                isRule={b.id === ruleBlanketId}
                onPress={() => setPicked(b.id)}
              />
            ))}
            {mode === "override" ? (
              <ChoiceRow
                blanket={null}
                label="Keine Decke"
                selected={picked === null}
                isRule={ruleSaysNone}
                onPress={() => setPicked(null)}
              />
            ) : null}
            {choices.length === 0 ? (
              <Text variant="body" tone="muted">
                Noch keine Decken angelegt.
              </Text>
            ) : null}
          </View>

          {mode === "override" ? (
            <>
              <View className="gap-2">
                <Text variant="secondary">Wie lange?</Text>
                <View className="flex-row flex-wrap gap-2">
                  {OVERRIDE_DURATIONS.map((d) => (
                    <Pill key={d.label} label={d.label} selected={nights === d.nights} onPress={() => setNights(d.nights)} />
                  ))}
                </View>
                {p.override ? (
                  <Text variant="caption">
                    Aktuell: {p.override.blanket?.name ?? "Keine Decke"}, {overrideUntilText(p.override, p.day)}. Speichern
                    ersetzt die Wahl ab heute Nacht.
                  </Text>
                ) : (
                  <Text variant="caption">Danach gilt wieder die Regel.</Text>
                )}
              </View>
              <Input
                value={note}
                onChangeText={setNote}
                accessibilityLabel="Grund"
                placeholder="Grund (optional), z. B. frisch geschoren"
                maxLength={200}
              />
            </>
          ) : null}

          {error ? (
            <Text variant="bodySm" tone="danger" accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
          <Button
            label={mode === "override" ? "Speichern" : "Eingedeckt"}
            fullWidth
            loading={save.isPending || setState.isPending}
            disabled={picked === undefined || (mode === "cover" && picked === null)}
            onPress={() => void submit()}
          />
          {mode === "override" && p.override ? (
            <Button
              label="Zurück zur Regel"
              variant="ghost"
              fullWidth
              loading={reset.isPending}
              onPress={() => void backToRule()}
            />
          ) : null}
        </>
      )}
    </Sheet>
  );
}
