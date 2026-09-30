import { CalendarDays, ChevronLeft, ChevronRight, Minus, Plus } from "lucide-react-native";
import { useState, type ReactNode } from "react";
import { Pressable, ScrollView, View } from "react-native";

import { Button, Pill, Text } from "@/components/ui";
import { addDays, addMonths, monthGrid, monthTitle, parseDate, relativeDay } from "@/lib/requests";

/** Label above a form control. */
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

const WEEKDAY_HEADS = ["Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"];

export type MonthCalendarProps = {
  /** Selected day, YYYY-MM-DD, or "". */
  value: string;
  /** First selectable day; earlier days are disabled. */
  min: string;
  today: string;
  onChange: (day: string) => void;
};

/** Month grid, Monday first, with previous/next month buttons. */
export function MonthCalendar({ value, min, today, onChange }: MonthCalendarProps) {
  const start = parseDate(value) ?? parseDate(min) ?? parseDate(today);
  const [view, setView] = useState({ year: start?.year ?? 1970, month: start?.month ?? 1 });
  const minParts = parseDate(min);
  const atMin = !!minParts && view.year * 12 + view.month <= minParts.year * 12 + minParts.month;
  const shift = (delta: number) => setView(addMonths(view.year, view.month, delta));

  return (
    <View className="gap-2 rounded-card border border-border bg-card p-3">
      <View className="flex-row items-center justify-between">
        <Button variant="ghost" size="icon" icon={ChevronLeft} accessibilityLabel="Vorheriger Monat" disabled={atMin} onPress={() => shift(-1)} />
        <Text variant="label" accessibilityRole="header">
          {monthTitle(view.year, view.month)}
        </Text>
        <Button variant="ghost" size="icon" icon={ChevronRight} accessibilityLabel="Nächster Monat" onPress={() => shift(1)} />
      </View>
      <View className="flex-row">
        {WEEKDAY_HEADS.map((w) => (
          <Text key={w} variant="caption" className="flex-1 text-center">
            {w}
          </Text>
        ))}
      </View>
      {monthGrid(view.year, view.month).map((week, i) => (
        <View key={i} className="flex-row">
          {week.map((d, j) => {
            if (!d) return <View key={j} className="h-10 flex-1" />;
            const disabled = d < min;
            const selected = d === value;
            return (
              <Pressable
                key={d}
                disabled={disabled}
                accessibilityRole="button"
                accessibilityLabel={relativeDay(d, today)}
                accessibilityState={{ selected, disabled }}
                onPress={() => onChange(d)}
                className="h-10 flex-1 items-center justify-center"
              >
                <View className={`h-9 w-9 items-center justify-center rounded-full ${selected ? "bg-primary" : d === today ? "border border-primary" : ""}`}>
                  <Text
                    variant="bodySm"
                    className={selected ? "font-sans-semibold text-white" : disabled ? "text-muted opacity-40" : ""}
                  >
                    {Number(d.slice(8))}
                  </Text>
                </View>
              </Pressable>
            );
          })}
        </View>
      ))}
    </View>
  );
}

/**
 * Horizontal row of quick day pills ("Heute", "Morgen", "Sa, 3. Okt", ...) plus an
 * "Anderer Tag" pill that opens a month calendar for anything further out.
 */
export function DayPicker({ from, value, onChange, today, days = 21, noneLabel }: DayPickerProps) {
  const [open, setOpen] = useState(false);
  const list = Array.from({ length: days }, (_, i) => addDays(from, i));
  const custom = value !== "" && !list.includes(value);
  return (
    <View className="gap-3">
      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-2" className="-mx-6" contentContainerStyle={{ paddingHorizontal: 24 }}>
        {noneLabel ? <Pill label={noneLabel} selected={value === ""} onPress={() => { setOpen(false); onChange(""); }} /> : null}
        <Pill
          label={custom ? relativeDay(value, today) : "Anderer Tag"}
          icon={CalendarDays}
          selected={custom || open}
          accessibilityState={{ expanded: open, selected: custom || open }}
          onPress={() => setOpen(!open)}
        />
        {list.map((d) => (
          <Pill key={d} label={relativeDay(d, today)} selected={value === d} onPress={() => { setOpen(false); onChange(d); }} />
        ))}
      </ScrollView>
      {open ? (
        <MonthCalendar
          value={value}
          min={from}
          today={today}
          onChange={(d) => {
            setOpen(false);
            onChange(d);
          }}
        />
      ) : null}
    </View>
  );
}
