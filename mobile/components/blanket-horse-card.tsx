import { ChevronRight, MessageSquareQuote } from "lucide-react-native";
import { useState } from "react";
import { Pressable, View } from "react-native";

import { BlanketActions } from "@/components/blanket-actions";
import { BlanketPhoto } from "@/components/blanket-photo";
import { Avatar, Badge, Card, Icon, Text } from "@/components/ui";
import {
  fillLabel,
  recommendationDetail,
  recommendationTitle,
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
  onAction: (action: StateAction) => void;
  onOpenPlan: () => void;
};

/** Large card of a horse that still needs its blanket decision tonight. */
export function BlanketHorseCard({ item, pending, onAction, onOpenPlan }: Props) {
  const { horse, recommendation: rec } = item;
  return (
    <Card className="gap-4">
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${horse.name}: Deckenplan öffnen`}
        onPress={onOpenPlan}
        className="min-h-touch flex-row items-center gap-3"
      >
        <Avatar name={horse.name} colorKey={horse.color_key} size="md" />
        <View className="flex-1">
          <Text variant="bodyStrong">{horse.name}</Text>
          {horse.box ? <Text variant="secondary">Box {horse.box}</Text> : null}
        </View>
        {item.is_mine ? <Badge variant="primary" label="Dein Pferd" /> : null}
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </Pressable>

      <View className="flex-row items-center gap-3 rounded-tile bg-background p-3">
        <BlanketPhoto url={rec.blanket?.photo_url ?? null} size={64} />
        <View className="flex-1 gap-0.5">
          <Text variant="label">Heute Nacht</Text>
          <Text variant="titleLg">
            {recommendationTitle(rec)}
          </Text>
          <Text variant="secondary">{recommendationDetail(rec)}</Text>
          {rec.blanket ? <Text variant="caption">{fillLabel(rec.blanket.fill_g)}</Text> : null}
        </View>
      </View>

      {rec.note ? (
        <View className="flex-row items-start gap-2">
          <Icon as={MessageSquareQuote} size={16} className="mt-0.5 text-accent-text" />
          <Text variant="bodySm" className="flex-1">
            <Text variant="bodySm" tone="accent">
              Wunsch der Besitzerin oder des Besitzers:{" "}
            </Text>
            {rec.note}
          </Text>
        </View>
      ) : null}

      <BlanketActions recommendation={rec} pending={pending} disabled={pending !== null} onAction={onAction} />
    </Card>
  );
}

/** Compact row of a finished horse; tapping it shows the buttons again to correct the state. */
export function BlanketDoneRow({
  item,
  pending,
  timeZone,
  onAction,
  onOpenPlan,
}: Props) {
  const [expanded, setExpanded] = useState(false);
  const { horse, state } = item;
  return (
    <Card padded={false} shape="tile">
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${horse.name}: ${state ? stateLabel(state) : "erledigt"}. Zum Ändern tippen`}
        onPress={() => setExpanded((v) => !v)}
        className="min-h-[60px] flex-row items-center gap-3 px-4 py-3"
      >
        <Avatar name={horse.name} colorKey={horse.color_key} size="sm" />
        <View className="flex-1">
          <Text variant="bodyStrong">{horse.name}</Text>
          {state ? (
            <Text variant="secondary" numberOfLines={1}>
              {stateLabel(state)} · {stateByline(state, timeZone)}
            </Text>
          ) : null}
        </View>
        <Badge variant="primary" label="Erledigt" />
      </Pressable>
      {expanded ? (
        <View className="gap-3 border-t border-divider px-4 pb-4 pt-3">
          <BlanketActions
            recommendation={item.recommendation}
            pending={pending}
            disabled={pending !== null}
            onAction={(a) => {
              onAction(a);
              setExpanded(false);
            }}
          />
          <Pressable accessibilityRole="button" onPress={onOpenPlan} className="min-h-touch justify-center">
            <Text variant="bodySm" tone="primary">
              Deckenplan von {horse.name} ansehen
            </Text>
          </Pressable>
        </View>
      ) : null}
    </Card>
  );
}
