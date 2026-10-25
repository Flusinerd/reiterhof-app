import { useEffect, useState } from "react";
import { View } from "react-native";

import { Button, DateField, Pill, Sheet, Text } from "@/components/ui";
import { goalOptions, selectedGoal, type GoalLevel, type GoalState } from "@/lib/training-goal";

/**
 * "Ziel": the class the horse is trained towards (the next one by default, or a higher one) and an
 * optional end date. The app counts how many lections of that class sit and plans the next ones.
 */
export function GoalSheet({
  open,
  onOpenChange,
  horseName,
  goal,
  saving,
  error,
  onSave,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  horseName: string;
  goal: GoalState;
  saving: boolean;
  error: string | null;
  onSave: (goal: { level: GoalLevel | null; until: string }) => unknown;
}) {
  const options = goalOptions(goal.level);
  const [level, setLevel] = useState<GoalLevel | null>(null);
  const [until, setUntil] = useState("");

  // The sheet starts from what is stored every time it opens.
  useEffect(() => {
    if (!open) return;
    setLevel(selectedGoal(goal));
    setUntil(goal.goal_until ?? "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title="Ziel"
      description={`Worauf trainiert ${horseName} hin? Die App zählt, wie viele Lektionen der Klasse sitzen.`}
    >
      {options.length > 0 ? (
        <View className="gap-2">
          <Text variant="eyebrow">Klasse</Text>
          <View className="flex-row flex-wrap gap-2">
            {options.map((o) => (
              <Pill key={o.value ?? "next"} label={o.label} selected={level === o.value} onPress={() => setLevel(o.value)} />
            ))}
          </View>
        </View>
      ) : null}
      <View className="gap-2">
        <Text variant="eyebrow">Bis (optional)</Text>
        <DateField
          accessibilityLabel="Ziel bis"
          placeholder="Kein Datum"
          clearable
          minimumDate={new Date()}
          value={until}
          onChange={setUntil}
        />
      </View>
      {error ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button label="Speichern" fullWidth loading={saving} onPress={() => onSave({ level, until })} />
    </Sheet>
  );
}
