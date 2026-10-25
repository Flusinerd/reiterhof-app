import { useLocalSearchParams, useRouter } from "expo-router";
import { ArrowDown, ArrowUp, Check, CircleAlert, Dices, Pencil, Play, Share2, Square, Trash2, X } from "lucide-react-native";
import { useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, Alert, Pressable, Share, View, useWindowDimensions } from "react-native";

import { DressageArena } from "@/components/dressage-arena";
import { Button, Icon, PageHeader, Pill, Screen, Section, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import {
  dressageError,
  isUnavailable,
  useDressageTest,
  useGenerateTest,
  useRemoveTest,
  useSaveTest,
  useUpdateTest,
  type GenerateBody,
} from "@/lib/api/dressage";
import {
  arenaLabel,
  defaultTitle,
  formatSeconds,
  lineText,
  moveLine,
  musicPartText,
  musicParts,
  positionAt,
  reinBalanceText,
  removeLine,
  renumber,
  requirementRows,
  requirementSummary,
  shareText,
  spokenSeconds,
  type Plan,
} from "@/lib/dressage";
import { getPreview, newSeed, setPreview, type Preview } from "@/lib/dressage-preview";
import { cn } from "@/lib/cn";
import { useColors } from "@/lib/theme";

/**
 * The plan of a test (also the preview, `id` = "preview"): the arena with the selected line in brass, the
 * numbered lines, for a Kür the checklist and the music parts, and the actions. A saved test can be edited
 * (reorder, remove) and is re-checked by the server; a preview can be saved or re-rolled.
 */
export default function DressagePlan() {
  const { id, horse } = useLocalSearchParams<{ id: string; horse?: string }>();
  const router = useRouter();
  const c = useColors();
  const { width: screenWidth } = useWindowDimensions();
  const isPreview = id === "preview";

  const [preview, setPreviewState] = useState<Preview | null>(() => (isPreview ? getPreview() : null));
  // While a delete runs the test must not be fetched again (the effect of the delete refreshes it, and it is gone).
  const [deleting, setDeleting] = useState(false);
  const saved = useDressageTest(isPreview || deleting ? undefined : id);
  const horseId = isPreview ? (preview?.horseId ?? horse ?? "") : (saved.data?.horse_id ?? horse ?? "");

  const generate = useGenerateTest(horseId);
  const save = useSaveTest(horseId);
  const update = useUpdateTest(horseId);
  const remove = useRemoveTest(horseId);

  // A changed plan that is not stored yet (re-rolled or edited lines) of a saved test.
  const [draft, setDraft] = useState<Plan | null>(null);
  const [editing, setEditing] = useState(false);
  const [selected, setSelected] = useState<number | null>(null);
  const [shareError, setShareError] = useState<string | null>(null);
  const [savedNow, setSavedNow] = useState(false);

  // Playback
  const [playing, setPlaying] = useState(false);
  const [speed, setSpeed] = useState<1 | 2>(1);
  const [elapsed, setElapsed] = useState(0);
  const clock = useRef({ startedAt: 0, from: 0 });

  const plan: Plan | null = draft ?? (isPreview ? (preview?.plan ?? null) : (saved.data?.plan ?? null));
  const title = (!isPreview && saved.data?.title) || (plan ? defaultTitle(plan.kind, plan.class, plan.arena) : "Aufgabe");
  const total = plan?.seconds ?? 0;
  const lines = useMemo(() => plan?.lines ?? [], [plan]);
  const hasPaths = lines.some((l) => l.path.length > 1);

  useEffect(() => {
    if (!playing) return;
    let frame = 0;
    const tick = () => {
      const t = clock.current.from + ((Date.now() - clock.current.startedAt) / 1000) * speed;
      if (t >= total) {
        setElapsed(total);
        setPlaying(false);
        return;
      }
      setElapsed(t);
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [playing, speed, total]);

  const stopPlay = () => {
    setPlaying(false);
    setElapsed(0);
  };

  const startPlay = () => {
    if (!hasPaths) return;
    const from = elapsed >= total ? 0 : elapsed;
    clock.current = { startedAt: Date.now(), from };
    setElapsed(from);
    setEditing(false);
    setPlaying(true);
  };

  const changeSpeed = (s: 1 | 2) => {
    // Keep the position: the clock restarts from where the dot is.
    clock.current = { startedAt: Date.now(), from: elapsed };
    setSpeed(s);
  };

  const pos = hasPaths && (playing || elapsed > 0) ? positionAt(lines, elapsed) : null;
  const shownLine = pos ? pos.index : selected;

  if (!isPreview && saved.isPending) {
    return (
      <Screen back>
        <PageHeader eyebrow="Aufgaben und Küren" title="Aufgabe" />
        <ActivityIndicator accessibilityLabel="Lädt" color={c.primary.DEFAULT} className="self-start" />
      </Screen>
    );
  }
  if (!plan) {
    return (
      <Screen back>
        <PageHeader eyebrow="Aufgaben und Küren" title="Aufgabe" />
        {!isPreview && saved.isError ? (
          <View className="gap-3">
            <Text variant="secondary" tone="danger" accessibilityRole="alert">
              {isUnavailable(saved.error) ? "Diese Aufgabe gibt es nicht mehr." : dressageError(saved.error)}
            </Text>
            {!isUnavailable(saved.error) ? <Button label="Erneut versuchen" variant="outline" onPress={() => void saved.refetch()} /> : null}
          </View>
        ) : (
          <Text variant="body" tone="muted">
            Die Vorschau ist nicht mehr da. Erstelle die Aufgabe noch einmal.
          </Text>
        )}
      </Screen>
    );
  }

  const dirty = !isPreview && draft !== null;
  const arenaWidth = Math.min(screenWidth - 48, 400);
  const freestyle = plan.kind === "freestyle";
  const music = musicParts(plan.music);
  const balance = reinBalanceText(plan.reins);
  const reqs = requirementRows(plan.requirements);
  const failedChecks = plan.checks.filter((k) => !k.ok);

  // "Neu würfeln" needs the request: the preview has it, a stored practice test can be rebuilt from its fields.
  const rerollRequest: Omit<GenerateBody, "save"> | null = isPreview
    ? (preview?.request ?? null)
    : plan.kind === "practice"
      ? { kind: "practice", class: plan.class, arena: plan.arena, length: "normal", ...(saved.data?.focus_key ? { focus_key: saved.data.focus_key } : {}) }
      : null;

  const reroll = async () => {
    if (!rerollRequest) return;
    stopPlay();
    const request = { ...rerollRequest, seed: newSeed() };
    const { plan: next } = await generate.mutateAsync({ ...request, save: false });
    setSelected(null);
    setSavedNow(false);
    if (isPreview && preview) {
      const p = { ...preview, request, plan: next };
      setPreview(p);
      setPreviewState(p);
    } else {
      setDraft(next);
    }
  };

  const savePreview = async () => {
    if (!preview) return;
    const { stored } = await save.mutateAsync({ ...preview.request, seed: preview.plan.seed, save: true });
    setPreview(null);
    // Back from the saved test leads to the list, not to the form that made it.
    router.dismissTo({ pathname: "/training/tests", params: { horse: horseId } });
    if (stored) router.push({ pathname: "/training/tests/[id]", params: { id: stored.id, horse: horseId } });
  };

  const saveDraft = async () => {
    if (!draft || isPreview) return;
    await update.mutateAsync({ id, title: saved.data?.title || title, plan: { ...draft, lines: renumber(draft.lines) } });
    setDraft(null);
    setEditing(false);
    setSavedNow(true);
    setSelected(null);
  };

  const startEditing = () => {
    stopPlay();
    update.reset();
    setDraft(draft ?? plan);
    setEditing(true);
    setSelected(null);
  };

  const cancelEditing = () => {
    update.reset();
    setDraft(null);
    setEditing(false);
  };

  const edit = (next: Plan["lines"]) => {
    const renumbered = renumber(next);
    setDraft({ ...plan, lines: renumbered, seconds: renumbered.reduce((s, l) => s + l.seconds, 0) });
    setSavedNow(false);
  };

  const doShare = async () => {
    setShareError(null);
    try {
      await Share.share({ title, message: shareText(title, plan) });
    } catch {
      setShareError("Teilen geht hier nicht.");
    }
  };

  const confirmRemove = () =>
    Alert.alert("Aufgabe löschen?", `„${title}“ wird gelöscht.`, [
      { text: "Abbrechen", style: "cancel" },
      {
        text: "Löschen",
        style: "destructive",
        onPress: () => {
          setDeleting(true);
          void remove.mutateAsync(id).then(
            () => router.back(),
            (e) => {
              setDeleting(false);
              Alert.alert("Das hat nicht geklappt", dressageError(e));
            },
          );
        },
      },
    ]);


  return (
    <Screen back>
      <PageHeader
        eyebrow={`${freestyle ? "Kür" : "Übungsaufgabe"} · Klasse ${plan.class}`}
        title={title}
        description={`${arenaLabel(plan.arena)} · ${formatSeconds(plan.seconds)} Min.${isPreview ? " · Vorschau" : ""}`}
      />

      <View className="items-center">
        <DressageArena arena={plan.arena} lines={lines} selected={shownLine} dot={pos?.point ?? null} width={arenaWidth} />
      </View>

      {hasPaths ? (
        <View className="gap-2">
          <View className="flex-row flex-wrap items-center gap-2">
            {playing ? (
              <Button label="Stopp" icon={Square} variant="outline" size="sm" onPress={stopPlay} />
            ) : (
              <Button label="Abspielen" icon={Play} variant="outline" size="sm" onPress={startPlay} disabled={editing} />
            )}
            <Pill label="1×" selected={speed === 1} onPress={() => changeSpeed(1)} />
            <Pill label="2×" selected={speed === 2} onPress={() => changeSpeed(2)} />
            {playing || elapsed > 0 ? (
              <Text variant="bodySm" tone="brass" accessibilityLabel={`${spokenSeconds(elapsed)} von ${spokenSeconds(total)}`}>
                {`${formatSeconds(elapsed)} / ${formatSeconds(total)}`}
              </Text>
            ) : null}
          </View>
        </View>
      ) : null}

      {balance && !dirty ? (
        <Text variant="secondary" tone={balance.startsWith("Beide") ? "success" : "accent"}>
          {balance}
        </Text>
      ) : null}

      <Section title="Linien" description={editing ? "Ordne um oder entferne Linien. Der Server prüft die Regeln beim Speichern." : undefined}>
        <RowList>
          {lines.map((l, i) => {
            const active = shownLine === i;
            return (
              <View key={`${l.n}-${i}`} className={cn("flex-row items-center gap-2", active && "bg-primary-soft")}>
                <Pressable
                  accessibilityRole="button"
                  accessibilityState={{ selected: active }}
                  accessibilityLabel={`${lineText(l)}, ${spokenSeconds(l.seconds)}`}
                  onPress={() => setSelected(selected === i ? null : i)}
                  className="min-h-[56px] flex-1 flex-row items-center gap-3 px-1 py-3 active:opacity-70"
                >
                  <Text className="w-8 text-right font-display text-[17px] text-brass">{l.n}</Text>
                  <View className="flex-1 gap-0.5">
                    {l.letters ? <Text className="font-display text-[17px] leading-6">{l.letters}</Text> : null}
                    <Text variant="bodySm">{l.text}</Text>
                  </View>
                  <Text variant="bodySm" tone="brass">
                    {formatSeconds(l.seconds)}
                  </Text>
                </Pressable>
                {editing ? (
                  <View className="flex-row">
                    <Button size="icon" variant="ghost" icon={ArrowUp} accessibilityLabel={`Linie ${l.n} nach oben`} disabled={i === 0} onPress={() => edit(moveLine(lines, i, -1))} />
                    <Button
                      size="icon"
                      variant="ghost"
                      icon={ArrowDown}
                      accessibilityLabel={`Linie ${l.n} nach unten`}
                      disabled={i === lines.length - 1}
                      onPress={() => edit(moveLine(lines, i, 1))}
                    />
                    <Button size="icon" variant="ghost" icon={X} accessibilityLabel={`Linie ${l.n} entfernen`} onPress={() => edit(removeLine(lines, i))} />
                  </View>
                ) : null}
              </View>
            );
          })}
        </RowList>
      </Section>

      {freestyle && reqs.length > 0 && !dirty ? (
        <Section title="Anforderungen" description={requirementSummary(plan.requirements)}>
          <View className="gap-2">
            {reqs.map((r) => (
              <View key={r.key} className="flex-row items-center gap-2">
                <Icon as={r.met ? Check : CircleAlert} size={18} className={r.met ? "text-success" : "text-accent-text"} />
                <Text variant="body" tone={r.met ? "default" : "accent"} accessibilityLabel={`${r.text}, ${r.met ? "erfüllt" : "offen"}`}>
                  {r.text}
                </Text>
              </View>
            ))}
          </View>
        </Section>
      ) : null}

      {dirty ? (
        <Text variant="secondary" tone="muted">
          Die Regeln, die Zeit und die Anforderungen prüft der Server beim Speichern.
        </Text>
      ) : null}

      {plan.checks.length > 0 && !dirty ? (
        <Section title="Regeln" description={failedChecks.length === 0 ? "Alle Regeln sind eingehalten." : `${failedChecks.length} ${failedChecks.length === 1 ? "Regel ist" : "Regeln sind"} nicht eingehalten.`}>
          <View className="gap-2">
            {[...failedChecks, ...plan.checks.filter((k) => k.ok)].map((k) => (
              <View key={k.key} className="flex-row items-start gap-2">
                <Icon as={k.ok ? Check : CircleAlert} size={18} className={k.ok ? "mt-0.5 text-success" : "mt-0.5 text-accent-text"} />
                <Text variant="bodySm" tone={k.ok ? "muted" : "accent"} className="flex-1">
                  {k.text}
                </Text>
              </View>
            ))}
          </View>
        </Section>
      ) : null}

      {freestyle && music.length > 0 && !dirty ? (
        <Section title="Musikteile" description="So kannst du die Musik schneiden.">
          <View className="gap-2">
            {music.map((p, i) => (
              <Text key={`${p.gait}-${i}`} variant="body">
                {musicPartText(p)}
              </Text>
            ))}
          </View>
        </Section>
      ) : null}

      <View className="gap-3">
        {/* The reasons of the server stand right above the action that caused them. */}
        {save.isError ? (
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {dressageError(save.error)}
          </Text>
        ) : null}
        {update.isError ? (
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {dressageError(update.error)}
          </Text>
        ) : null}
        {generate.isError ? (
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {dressageError(generate.error)}
          </Text>
        ) : null}

        {isPreview ? (
          <Button label="Speichern" fullWidth onPress={() => savePreview().catch(() => undefined)} />
        ) : dirty ? (
          <View className="gap-2">
            <Button label="Speichern" fullWidth onPress={() => saveDraft().catch(() => undefined)} />
            <Button label="Verwerfen" variant="ghost" fullWidth onPress={cancelEditing} />
          </View>
        ) : savedNow ? (
          <View className="flex-row items-center gap-2">
            <Icon as={Check} size={18} className="text-success" />
            <Text variant="bodyStrong" tone="success">
              Gespeichert
            </Text>
          </View>
        ) : null}

        <View className="flex-row flex-wrap gap-2">
          {rerollRequest ? <Button label="Neu würfeln" icon={Dices} variant="outline" size="sm" onPress={() => reroll().catch(() => undefined)} /> : null}
          {!isPreview && !editing ? <Button label="Zeilen bearbeiten" icon={Pencil} variant="outline" size="sm" onPress={startEditing} /> : null}
          <Button label="Teilen" icon={Share2} variant="outline" size="sm" onPress={() => void doShare()} />
        </View>
        {shareError ? (
          <Text variant="caption" tone="danger">
            {shareError}
          </Text>
        ) : null}
        {!isPreview ? <Button label="Löschen" icon={Trash2} variant="ghost" size="sm" onPress={confirmRemove} /> : null}
      </View>
    </Screen>
  );
}
