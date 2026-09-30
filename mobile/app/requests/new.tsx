import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocalSearchParams, useRouter } from "expo-router";
import { Plus, Send, Trash2 } from "lucide-react-native";
import { useMemo, useState } from "react";
import { View } from "react-native";

import { DayPicker, FieldRow, Stepper } from "@/components/request-form-parts";
import { RequestTypeIcon } from "@/components/request-type-icon";
import { Button, Card, Hero, Input, PressableCard, Pill, Screen, SectionLabel, Switch, Text } from "@/components/ui";
import { requestKeys, requestsApi, type RequestInput } from "@/lib/api/requests";
import {
  CREATABLE_TYPES,
  REMINDER_OPTIONS,
  SHOW_TASKS,
  buildRule,
  describeRule,
  isRequestType,
  normalizeTime,
  reminderAt,
  toIsoDate,
  typeMeta,
  type Recurrence,
  type ReminderOption,
  type RequestType,
  type ShowClass,
} from "@/lib/requests";
import { requestErrorMessage } from "@/lib/requests-errors";

const NEEDS_HORSE: RequestType[] = ["exercise", "feed_or_turnout", "appointment_companion"];

export default function NewRequest() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const params = useLocalSearchParams<{ type?: string; horse?: string }>();
  const today = useMemo(() => toIsoDate(new Date()), []);

  const [type, setType] = useState<RequestType>(
    isRequestType(params.type) && params.type !== "blanket" ? params.type : "other",
  );
  const [horseId, setHorseId] = useState<string>(typeof params.horse === "string" ? params.horse : "");
  const [date, setDate] = useState(today);
  const [dateEnd, setDateEnd] = useState("");
  const [timeFrom, setTimeFrom] = useState("");
  const [timeTo, setTimeTo] = useState("");
  const [location, setLocation] = useState("");
  const [description, setDescription] = useState("");
  const [helpers, setHelpers] = useState(1);
  const [tasks, setTasks] = useState<string[]>([]);
  const [taskDraft, setTaskDraft] = useState("");
  const [reminder, setReminder] = useState<ReminderOption>("default");
  const [recurrence, setRecurrence] = useState<Recurrence>("none");
  const [error, setError] = useState<string | null>(null);

  // show_helper
  const [showName, setShowName] = useState("");
  const [classes, setClasses] = useState<ShowClass[]>([]);
  const [rideAlong, setRideAlong] = useState(false);
  // ride_share
  const [destination, setDestination] = useState("");
  const [seats, setSeats] = useState(2);
  // exercise, feed_or_turnout, appointment_companion
  const [mode, setMode] = useState<"lunge" | "ride">("lunge");
  const [rulesNote, setRulesNote] = useState("");
  const [what, setWhat] = useState<"feed" | "turnout" | "bring_in">("feed");
  const [withWhom, setWithWhom] = useState<"farrier" | "vet" | "other">("farrier");
  const [note, setNote] = useState("");

  const options = useQuery({ queryKey: requestKeys.options, queryFn: requestsApi.options });

  const create = useMutation({
    mutationFn: (input: RequestInput) => requestsApi.create(input),
    onSuccess: (created) => {
      void queryClient.invalidateQueries({ queryKey: requestKeys.all });
      router.replace({ pathname: "/requests/[id]", params: { id: created.id } });
    },
    onError: (e) => setError(requestErrorMessage(e)),
  });

  const isRide = type === "ride_share";

  function submit() {
    setError(null);
    const from = timeFrom.trim() ? normalizeTime(timeFrom) : null;
    const to = timeTo.trim() ? normalizeTime(timeTo) : null;
    if (timeFrom.trim() && !from) return setError("Bitte gib die Uhrzeit so ein: 18:00");
    if (timeTo.trim() && !to) return setError("Bitte gib das Ende so ein: 19:30");
    if (NEEDS_HORSE.includes(type) && !horseId) return setError("Bitte wähle ein Pferd aus.");

    let payload: Record<string, unknown> = {};
    let departure: string | null = null;
    switch (type) {
      case "show_helper":
        if (!showName.trim()) return setError("Wie heißt das Turnier?");
        payload = {
          show_name: showName.trim(),
          classes: classes.filter((c) => c.name.trim()).map((c) => ({ name: c.name.trim(), ...(c.time ? { time: c.time } : {}) })),
          tasks,
          ride_along: rideAlong,
        };
        break;
      case "ride_share":
        departure = from;
        if (!destination.trim()) return setError("Wohin geht die Fahrt?");
        if (!departure) return setError("Bitte gib die Abfahrtszeit an, z. B. 07:15");
        payload = { destination: destination.trim(), departure_time: departure, seats_free: seats };
        break;
      case "exercise":
        payload = { mode, ...(rulesNote.trim() ? { rules_note: rulesNote.trim() } : {}) };
        break;
      case "feed_or_turnout":
        payload = { what };
        break;
      case "appointment_companion":
        payload = { with: withWhom, ...(note.trim() ? { note: note.trim() } : {}) };
        break;
      default:
        break;
    }

    const input: RequestInput = {
      type,
      date,
      payload,
      helpers_needed: isRide ? seats : helpers,
    };
    if (horseId) input.horse_id = horseId;
    if (dateEnd && dateEnd > date) input.date_end = dateEnd;
    if (from) input.time_from = from;
    if (to) input.time_to = to;
    if (location.trim()) input.location = location.trim();
    if (description.trim()) input.description = description.trim();
    if (type !== "show_helper" && tasks.length > 0) input.tasks = tasks;
    const rule = dateEnd ? "" : buildRule(recurrence, date);
    if (rule) input.recurring_rule = rule;
    const remind = reminderAt(reminder, date, from);
    if (remind) input.remind_helper_at = remind;
    create.mutate(input);
  }

  function addTask() {
    const t = taskDraft.trim();
    if (t && !tasks.includes(t)) setTasks([...tasks, t]);
    setTaskDraft("");
  }

  const ruleText = describeRule(buildRule(recurrence, date), date);

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <Hero
        eyebrow="Neue Anfrage"
        title="Wobei brauchst du Hilfe?"
        description="Die Anfrage geht an alle im Stall, auch wenn sie gerade nicht da sind."
      />

      <View className="gap-3">
        <SectionLabel>Art der Anfrage</SectionLabel>
        <View className="flex-row flex-wrap gap-3">
          {CREATABLE_TYPES.map((t) => (
            <PressableCard
              key={t.type}
              shape="tile"
              padded={false}
              accessibilityState={{ selected: type === t.type }}
              accessibilityLabel={t.label}
              onPress={() => {
                setType(t.type);
                setTasks([]);
              }}
              className={`w-[48%] flex-row items-center gap-3 p-3 ${type === t.type ? "border-primary bg-primary-soft" : ""}`}
            >
              <RequestTypeIcon type={t.type} size={40} selected={type === t.type} />
              <View className="flex-1">
                <Text variant="bodySm" className="font-sans-semibold">
                  {t.label}
                </Text>
                <Text variant="caption">{t.hint}</Text>
              </View>
            </PressableCard>
          ))}
        </View>
      </View>

      <View className="gap-3">
        <SectionLabel>{typeMeta(type).label}</SectionLabel>
        <Card className="gap-5">
          <FieldRow label="Pferd" hint={NEEDS_HORSE.includes(type) ? undefined : "Optional"}>
            <View className="flex-row flex-wrap gap-2">
              {NEEDS_HORSE.includes(type) ? null : (
                <Pill label="Kein Pferd" selected={horseId === ""} onPress={() => setHorseId("")} />
              )}
              {(options.data?.horses ?? []).map((h) => (
                <Pill key={h.id} label={h.name} selected={horseId === h.id} onPress={() => setHorseId(h.id)} />
              ))}
            </View>
          </FieldRow>

          <FieldRow label="Wann">
            <DayPicker from={today} today={today} value={date} onChange={(d) => { setDate(d); if (dateEnd && dateEnd <= d) setDateEnd(""); }} />
            <View className="flex-row gap-3">
              <View className="flex-1 gap-1">
                <Text variant="caption">{isRide ? "Abfahrt" : "Von"}</Text>
                <Input value={timeFrom} onChangeText={setTimeFrom} placeholder="18:00" keyboardType="numbers-and-punctuation" accessibilityLabel={isRide ? "Abfahrtszeit" : "Uhrzeit von"} />
              </View>
              {isRide ? null : (
                <View className="flex-1 gap-1">
                  <Text variant="caption">Bis</Text>
                  <Input value={timeTo} onChangeText={setTimeTo} placeholder="19:30" keyboardType="numbers-and-punctuation" accessibilityLabel="Uhrzeit bis" />
                </View>
              )}
            </View>
          </FieldRow>

          {type === "feed_or_turnout" ? (
            <FieldRow label="Mehrere Tage" hint="Bis einschließlich">
              <DayPicker from={date} today={today} value={dateEnd} onChange={setDateEnd} noneLabel="Nur ein Tag" />
            </FieldRow>
          ) : null}

          <FieldRow label="Wo">
            <Input value={location} onChangeText={setLocation} placeholder={isRide ? "Treffpunkt" : "z. B. Reithalle"} accessibilityLabel="Ort" />
          </FieldRow>

          <FieldRow label={isRide ? "Freie Plätze" : "Helfer"}>
            {isRide ? (
              <Stepper value={seats} min={1} max={8} onChange={setSeats} label="Freie Plätze" />
            ) : (
              <Stepper value={helpers} onChange={setHelpers} label="Anzahl Helfer" />
            )}
          </FieldRow>
        </Card>
      </View>

      {type === "show_helper" ? (
        <View className="gap-3">
          <SectionLabel>Turnier</SectionLabel>
          <Card className="gap-5">
            <FieldRow label="Name des Turniers">
              <Input value={showName} onChangeText={setShowName} placeholder="z. B. Herbstturnier Haltern" accessibilityLabel="Name des Turniers" />
            </FieldRow>
            <FieldRow label="Prüfungen">
              {classes.map((c, i) => (
                <View key={i} className="flex-row items-center gap-2">
                  <Input className="flex-1" value={c.name} placeholder="Prüfung" accessibilityLabel={`Prüfung ${i + 1}`} onChangeText={(name) => setClasses(classes.map((x, j) => (j === i ? { ...x, name } : x)))} />
                  <Input className="w-24" value={c.time ?? ""} placeholder="09:30" keyboardType="numbers-and-punctuation" accessibilityLabel={`Uhrzeit Prüfung ${i + 1}`} onChangeText={(time) => setClasses(classes.map((x, j) => (j === i ? { ...x, time: normalizeTime(time) ?? time } : x)))} />
                  <Button variant="ghost" size="icon" icon={Trash2} accessibilityLabel={`Prüfung ${i + 1} entfernen`} onPress={() => setClasses(classes.filter((_, j) => j !== i))} />
                </View>
              ))}
              <Button label="Prüfung hinzufügen" icon={Plus} variant="outline" size="sm" onPress={() => setClasses([...classes, { name: "" }])} />
            </FieldRow>
            <FieldRow label="Aufgaben">
              <View className="flex-row flex-wrap gap-2">
                {SHOW_TASKS.map((t) => (
                  <Pill
                    key={t.value}
                    label={t.label}
                    selected={tasks.includes(t.value)}
                    onPress={() => setTasks(tasks.includes(t.value) ? tasks.filter((x) => x !== t.value) : [...tasks, t.value])}
                  />
                ))}
              </View>
            </FieldRow>
            <Switch label="Mitfahrgelegenheit" description="Helfer können im Hänger mitfahren." value={rideAlong} onValueChange={setRideAlong} />
          </Card>
        </View>
      ) : null}

      {isRide ? (
        <View className="gap-3">
          <SectionLabel>Fahrt</SectionLabel>
          <Card className="gap-5">
            <FieldRow label="Ziel">
              <Input value={destination} onChangeText={setDestination} placeholder="z. B. Turnierplatz Haltern" accessibilityLabel="Ziel" />
            </FieldRow>
          </Card>
        </View>
      ) : null}

      {type === "exercise" ? (
        <View className="gap-3">
          <SectionLabel>Bewegen</SectionLabel>
          <Card className="gap-5">
            <FieldRow label="Wie">
              <View className="flex-row gap-2">
                <Pill label="Longieren" selected={mode === "lunge"} onPress={() => setMode("lunge")} />
                <Pill label="Reiten" selected={mode === "ride"} onPress={() => setMode("ride")} />
              </View>
            </FieldRow>
            <FieldRow label="Regeln für dieses Pferd" hint="Optional, zum Beispiel: nur Schritt und Trab">
              <Input value={rulesNote} onChangeText={setRulesNote} accessibilityLabel="Regeln" />
            </FieldRow>
          </Card>
        </View>
      ) : null}

      {type === "feed_or_turnout" ? (
        <View className="gap-3">
          <SectionLabel>Aufgabe</SectionLabel>
          <Card>
            <View className="flex-row flex-wrap gap-2">
              <Pill label="Füttern" selected={what === "feed"} onPress={() => setWhat("feed")} />
              <Pill label="Rausstellen" selected={what === "turnout"} onPress={() => setWhat("turnout")} />
              <Pill label="Reinholen" selected={what === "bring_in"} onPress={() => setWhat("bring_in")} />
            </View>
          </Card>
        </View>
      ) : null}

      {type === "appointment_companion" ? (
        <View className="gap-3">
          <SectionLabel>Termin</SectionLabel>
          <Card className="gap-5">
            <FieldRow label="Mit wem">
              <View className="flex-row flex-wrap gap-2">
                <Pill label="Hufschmied" selected={withWhom === "farrier"} onPress={() => setWithWhom("farrier")} />
                <Pill label="Tierarzt" selected={withWhom === "vet"} onPress={() => setWithWhom("vet")} />
                <Pill label="Sonstiges" selected={withWhom === "other"} onPress={() => setWithWhom("other")} />
              </View>
            </FieldRow>
            <FieldRow label="Hinweis" hint="Optional">
              <Input value={note} onChangeText={setNote} accessibilityLabel="Hinweis zum Termin" />
            </FieldRow>
          </Card>
        </View>
      ) : null}

      <View className="gap-3">
        <SectionLabel>Weitere Angaben</SectionLabel>
        <Card className="gap-5">
          {type !== "show_helper" ? (
            <FieldRow label="Checkliste" hint="Optional, was der Helfer tun soll">
              <View className="flex-row items-center gap-2">
                <Input className="flex-1" value={taskDraft} onChangeText={setTaskDraft} onSubmitEditing={addTask} placeholder="z. B. Hufe auskratzen" accessibilityLabel="Neue Aufgabe" />
                <Button variant="outline" size="icon" icon={Plus} accessibilityLabel="Aufgabe hinzufügen" onPress={addTask} />
              </View>
              {tasks.length > 0 ? (
                <View className="flex-row flex-wrap gap-2">
                  {tasks.map((t) => (
                    <Pill key={t} label={t} selected icon={Trash2} onPress={() => setTasks(tasks.filter((x) => x !== t))} />
                  ))}
                </View>
              ) : null}
            </FieldRow>
          ) : null}

          <FieldRow label="Beschreibung" hint="Optional">
            <Input
              className="h-24 py-3"
              value={description}
              onChangeText={setDescription}
              multiline
              textAlignVertical="top"
              accessibilityLabel="Beschreibung"
            />
          </FieldRow>

          <FieldRow label="Erinnerung für Helfer">
            <View className="flex-row flex-wrap gap-2">
              {REMINDER_OPTIONS.map((o) => (
                <Pill key={o.value} label={o.label} selected={reminder === o.value} onPress={() => setReminder(o.value)} />
              ))}
            </View>
          </FieldRow>

          {dateEnd ? null : (
            <FieldRow label="Wiederholung" hint={ruleText ? `Wird ${ruleText} neu angelegt.` : undefined}>
              <View className="flex-row flex-wrap gap-2">
                <Pill label="Einmalig" selected={recurrence === "none"} onPress={() => setRecurrence("none")} />
                <Pill label="Täglich" selected={recurrence === "daily"} onPress={() => setRecurrence("daily")} />
                <Pill label="Wöchentlich" selected={recurrence === "weekly"} onPress={() => setRecurrence("weekly")} />
              </View>
            </FieldRow>
          )}
        </Card>
      </View>

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button label="Anfrage senden" icon={Send} size="lg" fullWidth loading={create.isPending} onPress={submit} />
    </Screen>
  );
}
