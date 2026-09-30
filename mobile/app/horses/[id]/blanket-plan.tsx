import { useQueryClient } from "@tanstack/react-query";
import { router, useLocalSearchParams, type Href } from "expo-router";
import { History, MapPin, Pencil, Plus } from "lucide-react-native";
import { useState } from "react";
import { Pressable, RefreshControl, View } from "react-native";

import { BlanketEditSheet } from "@/components/blanket-edit-sheet";
import { BlanketHistory } from "@/components/blanket-history";
import { BlanketPhoto } from "@/components/blanket-photo";
import { BlanketRulesSheet } from "@/components/blanket-rules-sheet";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Badge, Button, Card, Hero, Icon, Input, Screen, SectionLabel, Sheet, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { BLANKET_PLAN_EVENT, BLANKET_STATE_EVENT, blanketKeys, usePlan } from "@/lib/api/blankets";
import { horsesApi, useHorseMutation } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";
import {
  fillLabel,
  nightLine,
  recommendationDetail,
  recommendationTitle,
  ruleBlanketName,
  ruleCondition,
  stateByline,
  stateLabel,
  type Blanket,
} from "@/lib/blankets";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { colors } from "@/lib/theme";

/** Deckenplan of a horse (JAN-30): tonight's recommendation, rules, blankets, helper note, history. */
export default function BlanketPlan() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const queryClient = useQueryClient();
  const { me } = useAuth();
  const plan = usePlan(id);
  useInvalidateOnEvents({
    [BLANKET_STATE_EVENT]: [blanketKeys.plan(id), blanketKeys.history(id)],
    [BLANKET_PLAN_EVENT]: [blanketKeys.plan(id)],
  });
  const [rulesOpen, setRulesOpen] = useState(false);
  const [editing, setEditing] = useState<{ blanket: Blanket | null } | null>(null);
  const [noteOpen, setNoteOpen] = useState(false);
  const [note, setNote] = useState("");
  const [noteError, setNoteError] = useState<string | null>(null);
  const saveNote = useHorseMutation((text: string) => horsesApi.update(id, { helper_note: text }));
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
  const canManage = p.can_manage;

  async function submitNote() {
    setNoteError(null);
    try {
      await saveNote.mutateAsync(note.trim());
      await queryClient.invalidateQueries({ queryKey: blanketKeys.plan(id) });
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
          tintColor={colors.primary.DEFAULT}
        />
      }
    >
      <Hero eyebrow={`${p.horse.name} · ${nightLine(p.weather)}`} title="Heute Nacht" description={recommendationDetail(rec)}>
        <View className="flex-row items-center gap-4">
          <BlanketPhoto url={rec.blanket?.photo_url ?? null} size={88} />
          <View className="flex-1 gap-1">
            <Text variant="titleLg">{recommendationTitle(rec)}</Text>
            {rec.blanket ? <Text variant="body">{fillLabel(rec.blanket.fill_g)}</Text> : null}
            {p.state ? (
              <Text variant="secondary" className="text-white/80">
                {stateLabel(p.state)} · {stateByline(p.state, timeZone)}
              </Text>
            ) : null}
          </View>
        </View>
        {rec.note ? (
          <Text variant="bodySm" className="text-white/90">
            Wunsch: {rec.note}
          </Text>
        ) : null}
      </Hero>

      <View className="gap-3">
        <SectionLabel
          action={
            canManage ? (
              <Button label="Bearbeiten" icon={Pencil} variant="ghost" size="sm" onPress={() => setRulesOpen(true)} />
            ) : undefined
          }
        >
          Regeln
        </SectionLabel>
        {p.rules.length === 0 ? (
          <Card>
            <Text variant="secondary">
              {canManage
                ? "Noch keine Regeln. Lege fest, welche Decke wann gilt."
                : "Noch keine Regeln."}
            </Text>
          </Card>
        ) : (
          p.rules.map((r, i) => {
            const active = rec.rule_index === i;
            return (
              <Card key={r.id} shape="tile" className={active ? "gap-1 border-primary bg-primary-soft" : "gap-1"}>
                <View className="flex-row items-center justify-between gap-3">
                  <Text variant="bodyStrong" className="flex-1">
                    {ruleBlanketName(r, p.blankets)}
                  </Text>
                  {active ? <Badge variant="primary" label="Gilt heute Nacht" /> : null}
                </View>
                <Text variant="secondary">
                  {i + 1}. {ruleCondition(r)}
                </Text>
                {r.note ? <Text variant="bodySm">Wunsch: {r.note}</Text> : null}
              </Card>
            );
          })
        )}
      </View>

      <View className="gap-3">
        <SectionLabel
          action={
            canManage ? (
              <Button label="Hinzufügen" icon={Plus} variant="ghost" size="sm" onPress={() => setEditing({ blanket: null })} />
            ) : undefined
          }
        >
          Decken
        </SectionLabel>
        {p.blankets.length === 0 ? (
          <Card>
            <Text variant="secondary">Noch keine Decken.</Text>
          </Card>
        ) : (
          <View className="flex-row flex-wrap gap-3">
            {p.blankets.map((b) => (
              <Pressable
                key={b.id}
                disabled={!canManage}
                accessibilityRole={canManage ? "button" : undefined}
                accessibilityLabel={canManage ? `${b.name} bearbeiten` : b.name}
                onPress={() => setEditing({ blanket: b })}
                className="basis-[47%] grow"
              >
                <Card shape="tile" className="gap-2 p-3">
                  <BlanketPhoto url={b.photo_url} size={96} />
                  <View className="gap-0.5">
                    <Text variant="bodyStrong" numberOfLines={1}>
                      {b.name}
                    </Text>
                    <Text variant="caption">{[fillLabel(b.fill_g), b.color].filter(Boolean).join(" · ")}</Text>
                    {b.location ? (
                      <View className="flex-row items-center gap-1">
                        <Icon as={MapPin} size={12} className="text-muted" />
                        <Text variant="caption" numberOfLines={1} className="flex-1">
                          {b.location}
                        </Text>
                      </View>
                    ) : null}
                  </View>
                </Card>
              </Pressable>
            ))}
          </View>
        )}
      </View>

      <View className="gap-3">
        <SectionLabel
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
          Hinweise für Helfer
        </SectionLabel>
        <Card>
          <Text variant={p.helper_note ? "body" : "secondary"}>{p.helper_note || "Noch kein Hinweis."}</Text>
        </Card>
      </View>

      <View className="gap-3">
        <SectionLabel
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
          Verlauf
        </SectionLabel>
        <BlanketHistory horseId={id} days={14} limit={5} timeZone={timeZone} />
      </View>

      {canManage ? (
        <>
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
