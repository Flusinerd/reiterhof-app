import { Check, Circle, Dot } from "lucide-react-native";
import { View } from "react-native";

import { Badge, Card, Divider, Icon, Text } from "@/components/ui";
import type { RehaPhase } from "@/lib/api/reha";
import { cn } from "@/lib/cn";
import { dateRange, phaseSummary } from "@/lib/reha";

/**
 * The phases of a plan one below the other: finished ones with a check, the current phase
 * highlighted with the "Jetzt" badge, the others muted.
 */
export function RehaTimeline({ phases }: { phases: RehaPhase[] }) {
  return (
    <Card padded={false}>
      {phases.map((p, i) => {
        const current = p.status === "current";
        const past = p.status === "past";
        return (
          <View key={`${i}-${p.name}`}>
            {i > 0 ? <Divider /> : null}
            <View
              accessible
              accessibilityLabel={`${p.name}, ${phaseSummary({ ...p })}, ${dateRange(p.start_date, p.end_date)}${current ? ", aktuelle Phase" : past ? ", abgeschlossen" : ""}`}
              className={cn("flex-row gap-3 p-4", current ? "bg-primary-soft" : undefined)}
            >
              <View className="pt-0.5">
                <Icon
                  as={past ? Check : current ? Dot : Circle}
                  size={current ? 24 : 20}
                  className={current ? "text-primary" : past ? "text-primary" : "text-muted"}
                />
              </View>
              <View className="flex-1 gap-1">
                <View className="flex-row items-center gap-2">
                  <Text variant="bodyStrong" className="flex-shrink" tone={past ? "muted" : "default"}>
                    {p.name}
                  </Text>
                  {current ? <Badge variant="primary" label="Jetzt" /> : null}
                </View>
                <Text variant="secondary">{phaseSummary({ ...p })}</Text>
                <Text variant="caption">{dateRange(p.start_date, p.end_date)}</Text>
                {p.conditions ? <Text variant="secondary">{p.conditions}</Text> : null}
              </View>
            </View>
          </View>
        );
      })}
    </Card>
  );
}
