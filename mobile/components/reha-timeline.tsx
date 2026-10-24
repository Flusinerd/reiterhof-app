import { Check, Circle } from "lucide-react-native";
import { View } from "react-native";

import { Icon, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import type { RehaPhase } from "@/lib/api/reha";
import { dateRange, phaseSummary } from "@/lib/reha";

/**
 * The phases of a plan as rows with hairlines: finished ones with a green check, the current phase
 * with a brass "Jetzt" eyebrow and a brass dot, the others muted.
 */
export function RehaTimeline({ phases }: { phases: RehaPhase[] }) {
  return (
    <RowList>
      {phases.map((p, i) => {
        const current = p.status === "current";
        const past = p.status === "past";
        return (
          <View
            key={`${i}-${p.name}`}
            accessible
            accessibilityLabel={`${p.name}, ${phaseSummary({ ...p })}, ${dateRange(p.start_date, p.end_date)}${current ? ", aktuelle Phase" : past ? ", abgeschlossen" : ""}`}
            className="flex-row gap-3 py-4"
          >
            <View className="w-6 items-center pt-0.5">
              {current ? (
                <View className="mt-1.5 h-3 w-3 rounded-pill bg-brass" />
              ) : (
                <Icon as={past ? Check : Circle} size={20} className={past ? "text-success" : "text-muted"} />
              )}
            </View>
            <View className="flex-1 gap-0.5">
              {current ? (
                <Text variant="eyebrow" tone="brass">
                  Jetzt
                </Text>
              ) : null}
              <Text variant="bodyStrong" tone={past ? "muted" : "default"}>
                {p.name}
              </Text>
              <Text variant="bodySm" tone="muted">
                {phaseSummary({ ...p })}
              </Text>
              <Text variant="caption">{dateRange(p.start_date, p.end_date)}</Text>
              {p.conditions ? <Text variant="secondary">{p.conditions}</Text> : null}
            </View>
          </View>
        );
      })}
    </RowList>
  );
}
