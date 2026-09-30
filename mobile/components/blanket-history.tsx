import { View } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Badge, Card, Divider, Text } from "@/components/ui";
import { useHistory } from "@/lib/api/blankets";
import { historyDayLabel, stateByline, stateLabel, type BlanketState } from "@/lib/blankets";

/** States grouped by blanket day, newest day first; within a day the newest state comes first. */
function groupByDay(states: readonly BlanketState[]): { day: string; states: BlanketState[] }[] {
  const groups: { day: string; states: BlanketState[] }[] = [];
  for (const s of states) {
    const last = groups[groups.length - 1];
    if (last && last.day === s.day) last.states.push(s);
    else groups.push({ day: s.day, states: [s] });
  }
  return groups;
}

/** History of the day states of one horse (JAN-31): who blanketed with what, per night. */
export function BlanketHistory({
  horseId,
  days = 14,
  limit,
  timeZone,
}: {
  horseId: string;
  days?: number;
  /** Show at most this many nights. */
  limit?: number;
  timeZone: string;
}) {
  const history = useHistory(horseId, days);
  if (!history.data) {
    return history.isError ? <HorseError error={history.error} onRetry={() => history.refetch()} /> : <HorseLoading />;
  }
  const groups = groupByDay(history.data.states).slice(0, limit);
  if (groups.length === 0) {
    return (
      <Card>
        <Text variant="secondary">In den letzten {days} Tagen wurde noch nichts eingetragen.</Text>
      </Card>
    );
  }
  return (
    <Card className="gap-0 p-0">
      {groups.map((g, gi) => (
        <View key={g.day}>
          {gi > 0 ? <Divider /> : null}
          <View className="gap-1 px-5 py-4">
            <View className="flex-row items-center justify-between gap-3">
              <Text variant="bodyStrong">{historyDayLabel(g.day, history.data.today)}</Text>
              <Badge label={stateLabel(g.states[0]!)} variant={g.states[0]!.action === "covered" ? "primary" : "neutral"} />
            </View>
            {g.states.map((s) => (
              <Text key={s.id} variant="secondary">
                {stateLabel(s)} · {stateByline(s, timeZone)}
              </Text>
            ))}
          </View>
        </View>
      ))}
    </Card>
  );
}
