import { Check, ChevronRight, MessageSquareQuote } from "lucide-react-native";
import { useState } from "react";
import { Pressable, View } from "react-native";

import { BlanketActions } from "@/components/blanket-actions";
import { BlanketChoiceSheet } from "@/components/blanket-choice-sheet";
import { BlanketPhoto } from "@/components/blanket-photo";
import { Avatar, Badge, Icon, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import {
  deviatesFromRule,
  recommendationDetail,
  recommendationTitle,
  ruleHint,
  stateByline,
  stateLabel,
  type StateAction,
  type TodayHorse,
} from "@/lib/blankets";

type Props = {
  item: TodayHorse;
  /** Action being sent for this horse, or null. */
  pending: StateAction | null;
  timeZone: string;
  /** No hairline below the last row of a list. */
  last?: boolean;
  onAction: (action: StateAction) => void;
  onOpenPlan: () => void;
};

/**
 * A horse that still needs its blanket decision tonight, as a row with a hairline: avatar, name and
 * box, tonight's blanket in italic, and the state buttons in one compact row below. Tapping the
 * head opens the blanket plan of the horse.
 */
export function BlanketHorseRow({ item, pending, last = false, onAction, onOpenPlan }: Props) {
  const { horse, recommendation: rec } = item;
  const [choiceOpen, setChoiceOpen] = useState(false);
  const hint = ruleHint(rec, item.rule_recommendation);
  const photo = rec.blanket?.photo_url ?? null;
  return (
    <View className={cn("gap-3 py-4", !last && "border-b border-divider")}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${horse.name}: Deckenplan öffnen`}
        onPress={onOpenPlan}
        className="min-h-touch flex-row items-center gap-3 active:opacity-70"
      >
        <Avatar name={horse.name} colorKey={horse.color_key} size="md" />
        <View className="flex-1 gap-0.5">
          <Text variant="bodyStrong">
            {horse.name}
            {horse.box ? (
              <Text variant="secondary">{`  Box ${horse.box}`}</Text>
            ) : null}
          </Text>
          <Text variant="body" className="font-display-italic">
            {recommendationTitle(rec)}
          </Text>
          <Text variant="secondary">{recommendationDetail(rec)}</Text>
        </View>
        {photo ? <BlanketPhoto url={photo} size={48} /> : null}
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </Pressable>

      {item.is_mine || rec.manual || hint ? (
        <View className="flex-row flex-wrap items-center gap-x-3 gap-y-1">
          {item.is_mine ? (
            <Text variant="eyebrow" tone="brass">
              Dein Pferd
            </Text>
          ) : null}
          {rec.manual ? <Badge variant="accent" label="Abweichend von Regel" /> : null}
          {hint ? <Text variant="caption">{hint}</Text> : null}
        </View>
      ) : null}

      {rec.note ? (
        <View className="flex-row items-start gap-2">
          <Icon as={MessageSquareQuote} size={16} className="mt-0.5 text-accent-text" />
          <Text variant="bodySm" className="flex-1">
            <Text variant="bodySm" tone="accent">
              {rec.manual ? "Grund: " : "Besitzerwunsch: "}
            </Text>
            {rec.note}
          </Text>
        </View>
      ) : null}

      <BlanketActions recommendation={rec} pending={pending} disabled={pending !== null} onAction={onAction} />
      <Pressable
        accessibilityRole="button"
        onPress={() => setChoiceOpen(true)}
        disabled={pending !== null}
        className="min-h-touch justify-center self-start"
      >
        <Text variant="bodySm" tone="primary">
          Mit anderer Decke eindecken
        </Text>
      </Pressable>
      <BlanketChoiceSheet open={choiceOpen} onOpenChange={setChoiceOpen} horseId={horse.id} mode="cover" />
    </View>
  );
}

/** A finished horse as a faded row; tapping it shows the buttons again to correct the state. */
export function BlanketDoneRow({ item, pending, timeZone, last = false, onAction, onOpenPlan }: Props) {
  const [expanded, setExpanded] = useState(false);
  const [choiceOpen, setChoiceOpen] = useState(false);
  const { horse, state } = item;
  const deviates = item.recommendation.manual || deviatesFromRule(state, item.rule_recommendation);
  return (
    <View className={cn(!last && "border-b border-divider")}>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${horse.name}: ${state ? stateLabel(state) : "erledigt"}. Zum Ändern tippen`}
        onPress={() => setExpanded((v) => !v)}
        className={cn("min-h-[60px] flex-row items-center gap-3 py-3 active:opacity-70", !expanded && "opacity-60")}
      >
        <Avatar name={horse.name} colorKey={horse.color_key} size="sm" />
        <View className="flex-1 gap-0.5">
          <Text variant="bodyStrong">{horse.name}</Text>
          {state ? (
            <Text variant="secondary" numberOfLines={1}>
              {stateLabel(state)} · {stateByline(state, timeZone)}
            </Text>
          ) : null}
          {deviates ? (
            <Text variant="caption" tone="accent">
              Abweichend von der Regel
            </Text>
          ) : null}
        </View>
        <Icon as={Check} size={20} className="text-success" />
      </Pressable>
      {expanded ? (
        <View className="gap-3 pb-4">
          <BlanketActions
            recommendation={item.recommendation}
            pending={pending}
            disabled={pending !== null}
            onAction={(a) => {
              onAction(a);
              setExpanded(false);
            }}
          />
          {ruleHint(item.recommendation, item.rule_recommendation) ? (
            <Text variant="caption">
              {recommendationTitle(item.recommendation)} von Hand gewählt,{" "}
              {ruleHint(item.recommendation, item.rule_recommendation)}
            </Text>
          ) : null}
          <Pressable
            accessibilityRole="button"
            onPress={() => setChoiceOpen(true)}
            disabled={pending !== null}
            className="min-h-touch justify-center"
          >
            <Text variant="bodySm" tone="primary">
              Mit anderer Decke eindecken
            </Text>
          </Pressable>
          <Pressable accessibilityRole="button" onPress={onOpenPlan} className="min-h-touch justify-center">
            <Text variant="bodySm" tone="primary">
              Deckenplan ansehen
            </Text>
          </Pressable>
        </View>
      ) : null}
      <BlanketChoiceSheet open={choiceOpen} onOpenChange={setChoiceOpen} horseId={horse.id} mode="cover" />
    </View>
  );
}
