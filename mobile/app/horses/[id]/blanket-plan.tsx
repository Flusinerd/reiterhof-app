import { useQueryClient } from "@tanstack/react-query";
import { router, useLocalSearchParams, type Href } from "expo-router";
import { Hand, History, Pencil, Plus, RotateCcw } from "lucide-react-native";
import { useState } from "react";
import { RefreshControl, View } from "react-native";

import { BlanketChoiceSheet } from "@/components/blanket-choice-sheet";
import { BlanketEditSheet } from "@/components/blanket-edit-sheet";
import { BlanketHistory } from "@/components/blanket-history";
import { BlanketPhoto } from "@/components/blanket-photo";
import { BlanketRulesSheet } from "@/components/blanket-rules-sheet";
import { CoverWindowSection } from "@/components/cover-window-section";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Button, Input, PageHeader, Screen, Section, Sheet, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { WeatherCard } from "@/components/weather-card";
import { errorMessage } from "@/lib/api";
import { BLANKET_PLAN_EVENT, BLANKET_STATE_EVENT, blanketKeys, blanketsApi, usePlan, usePlanMutation } from "@/lib/api/blankets";
import { horsesApi, useHorseMutation } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";
import {
  blanketErrorMessage,
  deviatesFromRule,
  fillLabel,
  nightLine,
  overrideUntilText,
  recommendationDetail,
  recommendationTitle,
  ruleBlanketName,
  ruleCondition,
  ruleHint,
  stateByline,
  stateLabel,
  type Blanket,
} from "@/lib/blankets";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { useColors } from "@/lib/theme";

/** Deckenplan of a horse (JAN-30): tonight's recommendation, rules, blankets, helper note, history. */
export default function BlanketPlan() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const c = useColors();
  const queryClient = useQueryClient();
  const { me } = useAuth();
  const plan = usePlan(id);
  useInvalidateOnEvents({
    [BLANKET_STATE_EVENT]: [blanketKeys.plan(id), blanketKeys.history(id)],
    [BLANKET_PLAN_EVENT]: [blanketKeys.plan(id)],
  });
  const [rulesOpen, setRulesOpen] = useState(false);
  const [editing, setEditing] = useState<{ blanket: Blanket | null } | null>(null);
  const [choiceOpen, setChoiceOpen] = useState(false);
  const [resetError, setResetError] = useState<string | null>(null);
  const resetChoice = usePlanMutation(() => blanketsApi.clearOverride(id));
  const [noteOpen, setNoteOpen] = useState(false);
  const [note, setNote] = useState("");
  const [noteError, setNoteError] = useState<string | null>(null);
  const saveNote = useHorseMutation(
    (text: string) => horsesApi.update(id, { helper_note: text }),
    () => [{ type: "horseUpdated", horseId: id }], // also refreshes the plan (the note shows there)
  );
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";

  if (!plan.data) {
    return (
      <Screen back>
        {plan.isError ? <HorseError error={plan.error} onRetry={() => plan.refetch()} /> : <HorseLoading />}
      </Screen>
    );
  }

  const p = plan.data;
  const rec = p.recommendation;
  const ruleRec = p.rule_recommendation ?? rec;
  const canManage = p.can_manage;
  const hint = ruleHint(rec, ruleRec);
  const stateDeviates = !rec.manual && deviatesFromRule(p.state, ruleRec);

  async function submitNote() {
    setNoteError(null);
    try {
      await saveNote.mutateAsync(note.trim());
      setNoteOpen(false);
    } catch (e) {
      setNoteError(errorMessage(e));
    }
  }

  return (
    <Screen
      back
      keyboardShouldPersistTaps="handled"
      refreshControl={
        <RefreshControl
          refreshing={plan.isRefetching}
          onRefresh={() => void queryClient.invalidateQueries({ queryKey: blanketKeys.all })}
          tintColor={c.primary.DEFAULT}
        />
      }
    >
      <PageHeader eyebrow={`${p.horse.name} · ${nightLine(p.weather)}`} title="Heute Nacht" description={recommendationDetail(rec)}>
        <View className="gap-4 rounded-card border border-border bg-raised p-5">
          <View className="flex-row items-center gap-4">
            <BlanketPhoto url={rec.blanket?.photo_url ?? null} size={88} />
            <View className="flex-1 gap-1">
              {rec.manual || stateDeviates ? (
                <Text variant="eyebrow" tone="accent">
                  Abweichend von Regel
                </Text>
              ) : null}
              <Text variant="title">{recommendationTitle(rec)}</Text>
              {rec.blanket ? <Text variant="body">{fillLabel(rec.blanket.fill_g)}</Text> : null}
              {p.state ? (
                <Text variant="bodySm" tone="muted">
                  {stateLabel(p.state)} · {stateByline(p.state, timeZone)}
                </Text>
              ) : null}
              {rec.manual && p.override ? <Text variant="caption">{overrideUntilText(p.override, p.day)}</Text> : null}
            </View>
          </View>
          {hint ? <Text variant="secondary">{hint}</Text> : null}
          {rec.note ? <Text variant="bodySm">{rec.manual ? "Grund" : "Wunsch"}: {rec.note}</Text> : null}
          {rec.manual && p.override?.set_by ? <Text variant="caption">Gewählt von {p.override.set_by.name}</Text> : null}
          {canManage ? (
            <View className="flex-row flex-wrap gap-2">
              <Button
                label={rec.manual ? "Wahl ändern" : "Decke von Hand wählen"}
                icon={Hand}
                variant="outline"
                size="sm"
                onPress={() => setChoiceOpen(true)}
              />
              {rec.manual ? (
                <Button
                  label="Zurück zur Regel"
                  icon={RotateCcw}
                  variant="ghost"
                  size="sm"
                  loading={resetChoice.isPending}
                  onPress={() => {
                    setResetError(null);
                    resetChoice.mutate(undefined, { onError: (e) => setResetError(blanketErrorMessage(e)) });
                  }}
                />
              ) : null}
            </View>
          ) : null}
        </View>
        {resetError ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {resetError}
          </Text>
        ) : null}
      </PageHeader>

      {p.weather ? <WeatherCard weather={p.weather} timeZone={timeZone} /> : null}

      <CoverWindowSection horseId={id} start={p.cover_start} end={p.cover_end} canManage={canManage} />

      <Section
        title="Regeln"
        action={
          canManage ? (
            <Button label="Bearbeiten" icon={Pencil} variant="ghost" size="sm" onPress={() => setRulesOpen(true)} />
          ) : undefined
        }
      >
        {p.rules.length === 0 ? (
          <Text variant="body" tone="muted">
            {canManage
              ? "Noch keine Regeln. Lege fest, welche Decke wann gilt."
              : "Noch keine Regeln."}
          </Text>
        ) : (
          <RowList>
            {p.rules.map((r, i) => {
              const active = ruleRec.rule_index === i;
              return (
                <ListRow
                  key={r.id}
                  eyebrow={active ? (rec.manual ? "Regel heute Nacht" : "Gilt heute Nacht") : `Regel ${i + 1}`}
                  eyebrowTone={active ? "brass" : undefined}
                  title={ruleBlanketName(r, p.blankets)}
                  detail={`${i + 1}. ${ruleCondition(r)}${r.note ? ` · Wunsch: ${r.note}` : ""}`}
                />
              );
            })}
          </RowList>
        )}
      </Section>

      <Section
        title="Decken"
        action={
          canManage ? (
            <Button label="Hinzufügen" icon={Plus} variant="ghost" size="sm" onPress={() => setEditing({ blanket: null })} />
          ) : undefined
        }
      >
        {p.blankets.length === 0 ? (
          <Text variant="body" tone="muted">
            Noch keine Decken.
          </Text>
        ) : (
          <RowList>
            {p.blankets.map((b) => (
              <ListRow
                key={b.id}
                leading={<BlanketPhoto url={b.photo_url} size={56} />}
                title={b.name}
                detail={[fillLabel(b.fill_g), b.color, b.location].filter(Boolean).join(" · ")}
                accessibilityLabel={canManage ? `${b.name} bearbeiten` : b.name}
                onPress={canManage ? () => setEditing({ blanket: b }) : undefined}
              />
            ))}
          </RowList>
        )}
      </Section>

      <Section
        title="Hinweise für Helfer"
        action={
          canManage ? (
            <Button
              label="Bearbeiten"
              icon={Pencil}
              variant="ghost"
              size="sm"
              onPress={() => {
                setNote(p.helper_note ?? "");
                setNoteError(null);
                setNoteOpen(true);
              }}
            />
          ) : undefined
        }
      >
        <Text variant="body" tone={p.helper_note ? "default" : "muted"}>
          {p.helper_note || "Noch kein Hinweis."}
        </Text>
      </Section>

      <Section
        title="Verlauf"
        action={
          <Button
            label="Alle"
            icon={History}
            variant="ghost"
            size="sm"
            onPress={() => router.push(`/blankets/history/${id}` as Href)}
          />
        }
      >
        <BlanketHistory horseId={id} days={14} limit={5} timeZone={timeZone} />
      </Section>

      {canManage ? (
        <>
          <BlanketChoiceSheet open={choiceOpen} onOpenChange={setChoiceOpen} horseId={id} mode="override" />
          <BlanketRulesSheet open={rulesOpen} onOpenChange={setRulesOpen} horseId={id} rules={p.rules} blankets={p.blankets} />
          <BlanketEditSheet
            open={editing !== null}
            onOpenChange={(o) => {
              if (!o) setEditing(null);
            }}
            horseId={id}
            blanket={editing?.blanket ?? null}
          />
          <Sheet open={noteOpen} onOpenChange={setNoteOpen} title="Hinweise für Helfer">
            <Input
              value={note}
              onChangeText={setNote}
              accessibilityLabel="Hinweise für Helfer"
              placeholder="z. B. Decke nie über den Kopf ziehen"
              multiline
              maxLength={2000}
              className="h-32 py-3"
              textAlignVertical="top"
            />
            {noteError ? (
              <Text variant="bodySm" tone="danger" accessibilityRole="alert">
                {noteError}
              </Text>
            ) : null}
            <Button label="Speichern" fullWidth loading={saveNote.isPending} onPress={() => void submitNote()} />
          </Sheet>
        </>
      ) : null}
    </Screen>
  );
}
