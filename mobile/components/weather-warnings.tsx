import { ChevronDown, ChevronUp, TriangleAlert } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import { Badge, Icon, PressableCard, Text } from "@/components/ui";
import { useWeatherWarnings } from "@/lib/api/weather";
import { cn } from "@/lib/cn";
import { severityLabel, severityTone, warningSpan, type WeatherWarning } from "@/lib/weather-warnings";

const softBg = { info: "bg-info-soft", accent: "bg-accent-soft", danger: "bg-danger-soft" } as const;
const iconColor = { info: "text-info", accent: "text-accent-text", danger: "text-danger" } as const;

function WarningCard({ warning, timeZone }: { warning: WeatherWarning; timeZone: string }) {
  const [open, setOpen] = useState(false);
  const tone = severityTone(warning.severity);
  const details = warning.description !== "" || warning.instruction !== "";
  return (
    <PressableCard
      className="gap-3"
      accessibilityLabel={`${severityLabel(warning.severity)}: ${warning.headline}`}
      accessibilityState={{ expanded: open }}
      accessibilityHint={details ? (open ? "Details ausblenden" : "Details anzeigen") : undefined}
      onPress={() => details && setOpen((o) => !o)}
    >
      <View className="flex-row items-start gap-3">
        <View className={cn("rounded-tile p-2", softBg[tone])}>
          <Icon as={TriangleAlert} size={20} className={iconColor[tone]} />
        </View>
        <View className="flex-1 gap-1">
          <Badge variant={tone} label={severityLabel(warning.severity)} />
          <Text variant="bodyStrong">{warning.headline}</Text>
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
 * without warnings. Tapping a warning shows the description and the DWD's advice.
 */
export function WeatherWarnings({ timeZone }: { timeZone: string }) {
  const warnings = useWeatherWarnings();
  const list = warnings.data?.warnings ?? [];
  if (list.length === 0) return null;
  return (
    <View className="gap-3">
      {list.map((w) => (
        <WarningCard key={w.id} warning={w} timeZone={timeZone} />
      ))}
      <Text variant="caption">Quelle: Deutscher Wetterdienst</Text>
    </View>
  );
}
