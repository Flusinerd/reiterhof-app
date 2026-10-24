import { ChevronDown, ChevronUp, TriangleAlert } from "lucide-react-native";
import { useState } from "react";
import { Pressable, View } from "react-native";

import { Icon, PressableCard, Text } from "@/components/ui";
import { useWeatherWarnings } from "@/lib/api/weather";
import { cn } from "@/lib/cn";
import { severityLabel, severityTone, warningSpan, type WeatherWarning } from "@/lib/weather-warnings";

const stripe = { info: "border-info", accent: "border-accent", danger: "border-danger" } as const;

/** The quiet strip of the "Heute" timeline. */
function CompactWarning({ warning, timeZone }: { warning: WeatherWarning; timeZone: string }) {
  const [open, setOpen] = useState(false);
  const details = warning.description !== "" || warning.instruction !== "";
  return (
    <PressableCard
      className="gap-3 border-0 bg-accent-soft p-4"
      accessibilityLabel={`${severityLabel(warning.severity)}: ${warning.headline}`}
      accessibilityState={{ expanded: open }}
      accessibilityHint={details ? (open ? "Details ausblenden" : "Details anzeigen") : undefined}
      onPress={() => details && setOpen((o) => !o)}
    >
      <View className="flex-row items-start gap-3">
        <View className="flex-1 gap-1">
          <Text variant="bodyStrong" tone="accent">
            {warning.headline}
          </Text>
          <Text variant="secondary">{warningSpan(warning.onset, warning.expires, timeZone)}</Text>
        </View>
        {details ? <Icon as={open ? ChevronUp : ChevronDown} size={20} className="text-muted" /> : null}
      </View>
      {open ? <WarningDetails warning={warning} /> : null}
    </PressableCard>
  );
}

function WarningDetails({ warning }: { warning: WeatherWarning }) {
  return (
    <View className="gap-2">
      {warning.description ? <Text variant="bodySm">{warning.description}</Text> : null}
      {warning.instruction ? <Text variant="caption">{warning.instruction}</Text> : null}
    </View>
  );
}

/** A warning as a row with a stripe in the tone of its level, on the page: level in the eyebrow, headline, time span. */
function WarningRow({ warning, timeZone }: { warning: WeatherWarning; timeZone: string }) {
  const [open, setOpen] = useState(false);
  const tone = severityTone(warning.severity);
  const details = warning.description !== "" || warning.instruction !== "";
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${severityLabel(warning.severity)}: ${warning.headline}`}
      accessibilityState={{ expanded: open }}
      accessibilityHint={details ? (open ? "Details ausblenden" : "Details anzeigen") : undefined}
      onPress={() => details && setOpen((o) => !o)}
      className={cn("min-h-touch flex-row items-start gap-3 border-l-2 py-1 pl-4 active:opacity-70", stripe[tone])}
    >
      <View className="flex-1 gap-1">
        <View className="flex-row items-center gap-1.5">
          <Icon as={TriangleAlert} size={14} className={tone === "info" ? "text-info" : tone === "danger" ? "text-danger" : "text-accent-text"} />
          <Text variant="eyebrow" tone={tone}>
            {severityLabel(warning.severity)}
          </Text>
        </View>
        <Text variant="bodyStrong">{warning.headline}</Text>
        <Text variant="secondary">{warningSpan(warning.onset, warning.expires, timeZone)}</Text>
        {open ? <WarningDetails warning={warning} /> : null}
      </View>
      {details ? <Icon as={open ? ChevronUp : ChevronDown} size={20} className="mt-1 text-muted" /> : null}
    </Pressable>
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
      {list.map((w) =>
        compact ? <CompactWarning key={w.id} warning={w} timeZone={timeZone} /> : <WarningRow key={w.id} warning={w} timeZone={timeZone} />,
      )}
      <Text variant="caption">Quelle: Deutscher Wetterdienst</Text>
    </View>
  );
}
