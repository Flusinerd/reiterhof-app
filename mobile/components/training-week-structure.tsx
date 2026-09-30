import { ChevronRight } from "lucide-react-native";
import { useState } from "react";
import { Pressable, View } from "react-native";

import { Card, Divider, Icon, Pill, Sheet, Text } from "@/components/ui";
import { WEEKDAYS_LONG } from "@/lib/training";
import { dayChoiceLabel, dayChoices, type DayChoice, type ProfileDraft } from "@/lib/training-profile";

export type WeekStructureRowsProps = {
  /** Seven entries, Monday first. */
  days: DayChoice[];
  /** Activity modes of the profile; the sheet offers only the activities that are not off. */
  modes: ProfileDraft["modes"];
  editable: boolean;
  onChange: (days: DayChoice[]) => void;
};

/**
 * The seven weekdays with what holds on each of them ("Frei", "Ruhetag", an activity, ...); a tap
 * opens a sheet with the choices. Used by the setup wizard and the structure editor.
 */
export function WeekStructureRows({ days, modes, editable, onChange }: WeekStructureRowsProps) {
  // Weekday whose choice sheet is open (0 = Monday).
  const [dayOpen, setDayOpen] = useState<number | null>(null);

  return (
    <View>
      <Card padded={false}>
        {WEEKDAYS_LONG.map((weekday, i) => {
          const choice = days[i] ?? "";
          const label = dayChoiceLabel(choice);
          return (
            <View key={weekday}>
              {i > 0 ? <Divider /> : null}
              <Pressable
                accessibilityRole={editable ? "button" : undefined}
                accessibilityLabel={`${weekday}: ${label}`}
                disabled={!editable}
                onPress={() => setDayOpen(i)}
                className="min-h-12 flex-row items-center gap-3 px-4 py-3 active:bg-background"
              >
                <Text variant="body" className="flex-1">
                  {weekday}
                </Text>
                <Text variant={choice ? "bodyStrong" : "secondary"}>{label}</Text>
                {editable ? <Icon as={ChevronRight} size={20} className="text-muted" /> : null}
              </Pressable>
            </View>
          );
        })}
      </Card>

      <Sheet
        open={dayOpen !== null}
        onOpenChange={(open) => {
          if (!open) setDayOpen(null);
        }}
        title={dayOpen !== null ? WEEKDAYS_LONG[dayOpen] : undefined}
        description="Was an diesem Tag gilt."
      >
        <View className="flex-row flex-wrap gap-2">
          {dayChoices({ modes }).map((c) => (
            <Pill
              key={c.value || "free"}
              label={c.label}
              selected={dayOpen !== null && days[dayOpen] === c.value}
              onPress={() => {
                if (dayOpen === null) return;
                onChange(days.map((d, j) => (j === dayOpen ? c.value : d)));
                setDayOpen(null);
              }}
            />
          ))}
        </View>
        <Text variant="caption">
          Aktive Erholung: kurze, lockere Einheit. Leicht, normal und fordernd richten sich nach der Belastung (Dauer ×
          Aktivität).
        </Text>
      </Sheet>
    </View>
  );
}
