import { useEffect, useState } from "react";
import { View } from "react-native";

import { Button, Input, Pill, Sheet, Text, TimeField } from "@/components/ui";
import type { HealthInput, HealthItem } from "@/lib/api/horses";
import {
  formatDate,
  HEALTH_KINDS,
  healthKindLabel,
  parseGermanDate,
  parseTime,
  parseWholeNumber,
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

/** Add / edit sheet for a health item (due date, interval, daily medication time). */
export function HorseHealthSheet({ open, onOpenChange, item, saving, error, onSave, onDelete }: Props) {
  const [kind, setKind] = useState("farrier");
  const [label, setLabel] = useState("");
  const [due, setDue] = useState("");
  const [interval, setInterval_] = useState("");
  const [dailyTime, setDailyTime] = useState("");
  const [note, setNote] = useState("");
  const [problem, setProblem] = useState<string | null>(null);

  // Reset the fields whenever the sheet opens for another item.
  useEffect(() => {
    if (!open) return;
    setKind(item?.kind ?? "farrier");
    setLabel(item?.label ?? "");
    setDue(formatDate(item?.due_date));
    setInterval_(item?.interval_days ? String(item.interval_days) : "");
    setDailyTime(item?.daily_time ?? "");
    setNote(item?.note ?? "");
    setProblem(null);
  }, [open, item]);

  function submit() {
    const dueIso = due.trim() === "" ? "" : parseGermanDate(due);
    const days = interval.trim() === "" ? 0 : parseWholeNumber(interval);
    const time = dailyTime.trim() === "" ? "" : parseTime(dailyTime);
    const finalLabel = label.trim() || healthKindLabel(kind);
    if (dueIso === null) return setProblem("Datum als TT.MM.JJJJ eingeben.");
    if (days === null || days > 3650) return setProblem("Intervall in Tagen als Zahl eingeben.");
    if (time === null) return setProblem("Uhrzeit als HH:MM eingeben.");
    setProblem(null);
    onSave({
      kind,
      label: finalLabel,
      due_date: dueIso,
      interval_days: days,
      note: note.trim(),
      daily_time: kind === "medication" ? time : "",
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
        <Text variant="secondary">{kind === "medication" ? "Ende der Gabe (optional)" : "Fällig am"}</Text>
        <Input
          value={due}
          onChangeText={setDue}
          accessibilityLabel="Fälligkeitsdatum"
          placeholder="TT.MM.JJJJ"
          keyboardType="numbers-and-punctuation"
        />
      </View>
      {kind === "medication" ? (
        <View className="gap-2">
          <Text variant="secondary">Tägliche Erinnerung um</Text>
          <TimeField
            value={dailyTime}
            onChange={setDailyTime}
            accessibilityLabel="Uhrzeit der täglichen Erinnerung"
            placeholder="z. B. 08:00"
            clearable
          />
        </View>
      ) : (
        <View className="gap-2">
          <Text variant="secondary">Wiederholung alle (Tage)</Text>
          <Input
            value={interval}
            onChangeText={setInterval_}
            accessibilityLabel="Intervall in Tagen"
            placeholder="z. B. 42"
            keyboardType="number-pad"
          />
          <Text variant="caption">Mit „Erledigt“ rückt der Termin um dieses Intervall weiter.</Text>
        </View>
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
