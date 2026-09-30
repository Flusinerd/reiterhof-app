import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react-native";
import { useEffect, useState } from "react";
import { ScrollView, View, useWindowDimensions } from "react-native";

import { Button, Card, Input, Pill, Sheet, Text, ToggleGroup, ToggleGroupItem } from "@/components/ui";
import { blanketsApi, usePlanMutation } from "@/lib/api/blankets";
import { blanketErrorMessage, type Blanket, type Rule } from "@/lib/blankets";
import {
  MAX_RULES,
  emptyDraft,
  moveItem,
  parseDrafts,
  toDraft,
  unreachableRules,
  type RainChoice,
  type RuleDraft,
} from "@/lib/blankets-rules";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  horseId: string;
  rules: Rule[];
  blankets: Blanket[];
};

/**
 * Edit the ordered rules of a horse (JAN-30). The first rule that matches the forecast
 * wins, so the order is the priority; "nie erreicht" warns about rules that a rule above
 * already covers.
 */
export function BlanketRulesSheet({ open, onOpenChange, horseId, rules, blankets }: Props) {
  const { height } = useWindowDimensions();
  const [drafts, setDrafts] = useState<RuleDraft[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setDrafts(rules.map(toDraft));
    setError(null);
  }, [open, rules]);

  const save = usePlanMutation((values: Parameters<typeof blanketsApi.saveRules>[1]) =>
    blanketsApi.saveRules(horseId, values),
  );

  const parsed = parseDrafts(drafts);
  const unreachable = parsed.ok ? unreachableRules(parsed.value) : [];

  const patch = (key: string, p: Partial<RuleDraft>) =>
    setDrafts((list) => list.map((d) => (d.key === key ? { ...d, ...p } : d)));

  async function submit() {
    setError(null);
    if (!parsed.ok) return setError(parsed.error);
    try {
      await save.mutateAsync(parsed.value);
      onOpenChange(false);
    } catch (e) {
      setError(blanketErrorMessage(e));
    }
  }

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title="Regeln"
      description="Die erste passende Regel gilt. Leere Felder heißen „egal“."
    >
      <ScrollView style={{ maxHeight: height * 0.55 }} contentContainerClassName="gap-3" keyboardShouldPersistTaps="handled">
        {drafts.length === 0 ? (
          <Text variant="secondary">Noch keine Regeln, also keine Empfehlung.</Text>
        ) : null}
        {drafts.map((d, i) => (
          <Card key={d.key} shape="tile" className="gap-3 p-4">
            <View className="flex-row items-center justify-between">
              <Text variant="bodyStrong">Regel {i + 1}</Text>
              <View className="flex-row">
                <Button size="icon" variant="ghost" icon={ArrowUp} accessibilityLabel={`Regel ${i + 1} nach oben`} disabled={i === 0} onPress={() => setDrafts((l) => moveItem(l, i, -1))} />
                <Button size="icon" variant="ghost" icon={ArrowDown} accessibilityLabel={`Regel ${i + 1} nach unten`} disabled={i === drafts.length - 1} onPress={() => setDrafts((l) => moveItem(l, i, 1))} />
                <Button size="icon" variant="ghost" icon={Trash2} accessibilityLabel={`Regel ${i + 1} löschen`} onPress={() => setDrafts((l) => l.filter((x) => x.key !== d.key))} />
              </View>
            </View>
            {unreachable.includes(i + 1) ? (
              <Text variant="caption" tone="accent">
                Wird nie erreicht: Eine Regel darüber deckt schon alles ab.
              </Text>
            ) : null}
            <View className="flex-row gap-3">
              <View className="flex-1 gap-1">
                <Text variant="caption">Ab (°C)</Text>
                <Input value={d.temp_min} onChangeText={(v) => patch(d.key, { temp_min: v })} accessibilityLabel={`Regel ${i + 1}: ab Temperatur`} placeholder="egal" keyboardType="numbers-and-punctuation" />
              </View>
              <View className="flex-1 gap-1">
                <Text variant="caption">Unter (°C)</Text>
                <Input value={d.temp_max} onChangeText={(v) => patch(d.key, { temp_max: v })} accessibilityLabel={`Regel ${i + 1}: unter Temperatur`} placeholder="egal" keyboardType="numbers-and-punctuation" />
              </View>
            </View>
            <ToggleGroup type="single" value={d.rain} onValueChange={(v) => patch(d.key, { rain: v as RainChoice })}>
              <ToggleGroupItem value="any" label="Egal" />
              <ToggleGroupItem value="rain" label="Regen" />
              <ToggleGroupItem value="dry" label="Trocken" />
            </ToggleGroup>
            <View className="gap-2">
              <Text variant="caption">Decke</Text>
              <View className="flex-row flex-wrap gap-2">
                <Pill label="Keine Decke" selected={d.blanket_id === null} onPress={() => patch(d.key, { blanket_id: null })} />
                {blankets.map((b) => (
                  <Pill key={b.id} label={b.name} selected={d.blanket_id === b.id} onPress={() => patch(d.key, { blanket_id: b.id })} />
                ))}
              </View>
            </View>
            <Input value={d.note} onChangeText={(v) => patch(d.key, { note: v })} accessibilityLabel={`Regel ${i + 1}: Wunsch`} placeholder="Wunsch (optional)" maxLength={200} />
          </Card>
        ))}
        <Button
          label="Regel hinzufügen"
          icon={Plus}
          variant="outline"
          disabled={drafts.length >= MAX_RULES}
          onPress={() => setDrafts((l) => [...l, emptyDraft()])}
        />
      </ScrollView>
      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button label="Speichern" fullWidth loading={save.isPending} onPress={() => void submit()} />
    </Sheet>
  );
}
