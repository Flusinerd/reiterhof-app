import { Minus, Plus } from "lucide-react-native";
import type { ReactNode } from "react";
import { ScrollView, View } from "react-native";

import { Button, Pill, Text } from "@/components/ui";
import { addDays, relativeDay } from "@/lib/requests";

/** Label above a form control, inside a card. */
export function FieldRow({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <View className="gap-2">
      <View className="gap-0.5">
        <Text variant="label">{label}</Text>
        {hint ? <Text variant="caption">{hint}</Text> : null}
      </View>
      {children}
    </View>
  );
}

export type StepperProps = {
  value: number;
  min?: number;
  max?: number;
  onChange: (value: number) => void;
  /** Accessibility name, e.g. "Anzahl Helfer". */
  label: string;
};

/** Minus / number / plus. */
export function Stepper({ value, min = 1, max = 20, onChange, label }: StepperProps) {
  return (
    <View className="flex-row items-center gap-4">
      <Button
        variant="outline"
        size="icon"
        icon={Minus}
        accessibilityLabel={`${label} verringern`}
        disabled={value <= min}
        onPress={() => onChange(Math.max(min, value - 1))}
      />
      <Text variant="title" accessibilityLabel={`${label}: ${value}`} className="min-w-8 text-center">
        {value}
      </Text>
      <Button
        variant="outline"
        size="icon"
        icon={Plus}
        accessibilityLabel={`${label} erhöhen`}
        disabled={value >= max}
        onPress={() => onChange(Math.min(max, value + 1))}
      />
    </View>
  );
}

export type DayPickerProps = {
  /** First selectable day, YYYY-MM-DD. */
  from: string;
  /** Selected day or "" when nothing is selected. */
  value: string;
  onChange: (day: string) => void;
  /** Reference for "Heute" / "Morgen". */
  today: string;
  days?: number;
  /** Adds a first pill that clears the selection. */
  noneLabel?: string;
};

/** Horizontal row of day pills ("Heute", "Morgen", "Sa, 3. Okt", ...). */
export function DayPicker({ from, value, onChange, today, days = 21, noneLabel }: DayPickerProps) {
  const list = Array.from({ length: days }, (_, i) => addDays(from, i));
  return (
    <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-2" className="-mx-5" contentContainerStyle={{ paddingHorizontal: 20 }}>
      {noneLabel ? <Pill label={noneLabel} selected={value === ""} onPress={() => onChange("")} /> : null}
      {list.map((d) => (
        <Pill key={d} label={relativeDay(d, today)} selected={value === d} onPress={() => onChange(d)} />
      ))}
    </ScrollView>
  );
}
