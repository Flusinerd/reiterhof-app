import { useEffect, useState } from "react";
import { View } from "react-native";

import { Button, DateField, Input, NumberStepper, Pill, Sheet, Text, TimeField } from "@/components/ui";
import type { HealthInput, HealthItem } from "@/lib/api/horses";
import {
  HEALTH_KINDS,
  healthKindLabel,
  INTERVAL_UNITS,
  intervalPresets,
  intervalText,
  intervalUnitLabel,
  MAX_INTERVAL_COUNT,
  MAX_REMIND_DAYS,
  parseTime,
  remindDayLabel,
  remindDayOptions,
  toggleRemindDay,
  type IntervalUnit,
} from "@/lib/horse-format";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The item to edit, or null to add a new one. */
  item: HealthItem | null;
  saving: boolean;
  /** German error text of the last attempt. */
  error?: string | null;
  onSave: (input: HealthInput) => void;
  onDelete?: () => void;
};

type Interval = { count: number; unit: IntervalUnit } | null;

const DEFAULT_REMIND_DAYS = [7, 1];
const DEFAULT_REMIND_TIME = "08:00";

function itemInterval(item: HealthItem | null): Interval {
  if (!item?.interval_count) return null;
  const unit = (INTERVAL_UNITS as readonly string[]).includes(item.interval_unit)
    ? (item.interval_unit as IntervalUnit)
    : "day";
  return { count: item.interval_count, unit };
}

/** Add / edit sheet for a health item (due date, interval, reminders, daily medication time). */
export function HorseHealthSheet({ open, onOpenChange, item, saving, error, onSave, onDelete }: Props) {
  const [kind, setKind] = useState("farrier");
  const [label, setLabel] = useState("");
  const [due, setDue] = useState("");
  const [interval, setInterval_] = useState<Interval>(null);
  const [customOpen, setCustomOpen] = useState(false);
  const [remindDays, setRemindDays] = useState<number[]>(DEFAULT_REMIND_DAYS);
  const [remindTime, setRemindTime] = useState(DEFAULT_REMIND_TIME);
  const [dailyTime, setDailyTime] = useState("");
  const [note, setNote] = useState("");
  const [problem, setProblem] = useState<string | null>(null);

  // Reset the fields whenever the sheet opens for another item.
  useEffect(() => {
    if (!open) return;
    setKind(item?.kind ?? "farrier");
    setLabel(item?.label ?? "");
    setDue(item?.due_date ?? "");
    setInterval_(itemInterval(item));
    setCustomOpen(false);
    setRemindDays(item?.remind_days ?? DEFAULT_REMIND_DAYS);
    setRemindTime(item?.remind_time ?? DEFAULT_REMIND_TIME);
    setDailyTime(item?.daily_time ?? "");
    setNote(item?.note ?? "");
    setProblem(null);
  }, [open, item]);

  const medication = kind === "medication";
  const presets = intervalPresets(kind);
  const isPreset = (p: { count: number; unit: string }) =>
    interval !== null && interval.count === p.count && interval.unit === p.unit;
  const custom = interval !== null && (customOpen || !presets.some(isPreset));
  function choose(next: Interval) {
    setCustomOpen(false);
    setInterval_(next);
  }

  function submit() {
    const time = dailyTime.trim() === "" ? "" : parseTime(dailyTime);
    const finalLabel = label.trim() || healthKindLabel(kind);
    if (time === null) return setProblem("Uhrzeit als HH:MM eingeben.");
    setProblem(null);
    onSave({
      kind,
      label: finalLabel,
      due_date: due,
      interval_count: medication || !interval ? 0 : interval.count,
      interval_unit: medication || !interval ? "day" : interval.unit,
      remind_days: remindDays,
      remind_time: remindTime || DEFAULT_REMIND_TIME,
      note: note.trim(),
      daily_time: medication ? time : "",
    });
  }

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title={item ? "Termin bearbeiten" : "Termin hinzufügen"}
      description="Erinnerungen gehen an Besitzer und Reitbeteiligungen."
    >
      <View className="flex-row flex-wrap gap-2">
        {HEALTH_KINDS.map((k) => (
          <Pill key={k} label={healthKindLabel(k)} selected={kind === k} onPress={() => setKind(k)} />
        ))}
      </View>
      <Input
        value={label}
        onChangeText={setLabel}
        accessibilityLabel="Bezeichnung"
        placeholder={`Bezeichnung, z. B. ${healthKindLabel(kind)}`}
      />
      <View className="gap-2">
        <Text variant="eyebrow">{medication ? "Ende der Gabe (optional)" : "Fällig am"}</Text>
        <DateField value={due} onChange={setDue} accessibilityLabel="Fälligkeitsdatum" clearable />
      </View>
      {medication ? (
        <View className="gap-2">
          <Text variant="eyebrow">Tägliche Erinnerung um</Text>
          <TimeField
            value={dailyTime}
            onChange={setDailyTime}
            accessibilityLabel="Uhrzeit der täglichen Erinnerung"
            placeholder="z. B. 08:00"
            clearable
          />
        </View>
      ) : (
        <>
          <View className="gap-2">
            <Text variant="eyebrow">Wiederholung</Text>
            <View className="flex-row flex-wrap gap-2">
              <Pill label="Einmalig" selected={interval === null} onPress={() => choose(null)} />
              {presets.map((p) => (
                <Pill
                  key={`${p.count}-${p.unit}`}
                  label={intervalText(p.count, p.unit)}
                  selected={!custom && isPreset(p)}
                  onPress={() => choose(p)}
                />
              ))}
              <Pill
                label="Eigenes"
                selected={custom}
                onPress={() => {
                  setCustomOpen(true);
                  setInterval_(interval ?? { count: 4, unit: "week" });
                }}
              />
            </View>
            {custom && interval ? (
              <View className="gap-2">
                <NumberStepper
                  value={interval.count}
                  min={1}
                  max={MAX_INTERVAL_COUNT[interval.unit]}
                  onChange={(count) => setInterval_({ ...interval, count })}
                  accessibilityLabel="Anzahl"
                  format={(v) => `alle ${v}`}
                />
                <View className="flex-row flex-wrap gap-2">
                  {INTERVAL_UNITS.map((u) => (
                    <Pill
                      key={u}
                      label={intervalUnitLabel(u)}
                      selected={interval.unit === u}
                      onPress={() =>
                        setInterval_({ unit: u, count: Math.min(interval.count, MAX_INTERVAL_COUNT[u]) })
                      }
                    />
                  ))}
                </View>
              </View>
            ) : null}
            <Text variant="caption">
              {interval
                ? `Mit „Erledigt“ rückt der Termin um ${intervalText(interval.count, interval.unit)} weiter.`
                : "Mit „Erledigt“ ist der Termin abgeschlossen."}
            </Text>
          </View>
          <View className="gap-2">
            <Text variant="eyebrow">Erinnerung</Text>
            <View className="flex-row flex-wrap gap-2">
              {remindDayOptions(item?.remind_days ?? []).map((d) => (
                <Pill
                  key={d}
                  label={remindDayLabel(d)}
                  selected={remindDays.includes(d)}
                  onPress={() => setRemindDays((cur) => toggleRemindDay(cur, d))}
                />
              ))}
            </View>
            <View className="flex-row items-center gap-3">
              <Text variant="bodySm">um</Text>
              <TimeField
                value={remindTime}
                onChange={setRemindTime}
                accessibilityLabel="Uhrzeit der Erinnerung"
                minuteInterval={15}
                className="flex-1"
              />
            </View>
            <Text variant="caption">
              {remindDays.length === 0
                ? "Ohne Auswahl gibt es keine Erinnerung."
                : `Bis zu ${MAX_REMIND_DAYS} Zeitpunkte, als Mitteilung an Besitzer und Reitbeteiligungen.`}
            </Text>
          </View>
        </>
      )}
      <Input value={note} onChangeText={setNote} accessibilityLabel="Notiz" placeholder="Notiz (optional)" />
      {problem || error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {problem ?? error}
        </Text>
      ) : null}
      <Button label="Speichern" fullWidth loading={saving} onPress={submit} />
      {item && onDelete ? <Button label="Termin löschen" variant="ghost" fullWidth onPress={onDelete} /> : null}
    </Sheet>
  );
}
