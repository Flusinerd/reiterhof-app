import { ChevronDown, ChevronUp, TriangleAlert } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import { Badge, Icon, PressableCard, Text } from "@/components/ui";
import { useWeatherWarnings } from "@/lib/api/weather";
import { cn } from "@/lib/cn";
import { severityLabel, severityTone, warningSpan, type WeatherWarning } from "@/lib/weather-warnings";

const softBg = { info: "bg-info-soft", accent: "bg-accent-soft", danger: "bg-danger-soft" } as const;
const iconColor = { info: "text-info", accent: "text-accent-text", danger: "text-danger" } as const;

function WarningCard({ warning, timeZone, compact }: { warning: WeatherWarning; timeZone: string; compact: boolean }) {
  const [open, setOpen] = useState(false);
  const tone = severityTone(warning.severity);
  const details = warning.description !== "" || warning.instruction !== "";
  return (
    <PressableCard
      className={cn("gap-3", compact && "border-0 bg-accent-soft p-4")}
      accessibilityLabel={`${severityLabel(warning.severity)}: ${warning.headline}`}
      accessibilityState={{ expanded: open }}
      accessibilityHint={details ? (open ? "Details ausblenden" : "Details anzeigen") : undefined}
      onPress={() => details && setOpen((o) => !o)}
    >
      <View className="flex-row items-start gap-3">
        {compact ? null : (
          <View className={cn("rounded-tile p-2", softBg[tone])}>
            <Icon as={TriangleAlert} size={20} className={iconColor[tone]} />
          </View>
        )}
        <View className="flex-1 gap-1">
          {compact ? null : <Badge variant={tone} label={severityLabel(warning.severity)} />}
          <Text variant="bodyStrong" tone={compact ? "accent" : "default"}>
            {warning.headline}
          </Text>
          <Text variant="secondary">{warningSpan(warning.onset, warning.expires, timeZone)}</Text>
        </View>
        {details ? <Icon as={open ? ChevronUp : ChevronDown} size={20} className="text-muted" /> : null}
      </View>
      {open ? (
        <View className="gap-2">
          {warning.description ? <Text variant="bodySm">{warning.description}</Text> : null}
          {warning.instruction ? <Text variant="caption">{warning.instruction}</Text> : null}
        </View>
      ) : null}
    </PressableCard>
  );
}

/**
 * Active official weather warnings of the DWD for the stable (most severe first); renders nothing
 * without warnings. Tapping a warning shows the description and the DWD's advice. `compact` is the
 * quiet strip of the "Heute" timeline.
 */
export function WeatherWarnings({ timeZone, compact = false }: { timeZone: string; compact?: boolean }) {
  const warnings = useWeatherWarnings();
  const list = warnings.data?.warnings ?? [];
  if (list.length === 0) return null;
  return (
    <View className="gap-3">
      {list.map((w) => (
        <WarningCard key={w.id} warning={w} timeZone={timeZone} compact={compact} />
      ))}
      <Text variant="caption">Quelle: Deutscher Wetterdienst</Text>
    </View>
  );
}
