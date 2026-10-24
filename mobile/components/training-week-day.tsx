import { useRouter, type Href } from "expo-router";
import { BookOpen } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { ActivityIcon } from "@/components/training-activity-icon";
import { Button, Text } from "@/components/ui";
import type { WeekDay } from "@/lib/api/training";
import { weekRehaText } from "@/lib/reha";
import { dayStatusLabel, formatDayLong, weekdayShort } from "@/lib/training";
import type { PlanDay } from "@/lib/training-plan";
import { showTestsText } from "@/lib/training-profile";

export type WeekDayRowProps = {
  day: WeekDay;
  draftDay?: PlanDay;
  editable: boolean;
  busy: boolean;
  onTake: () => void;
  onEdit: () => void;
};

/** The big day number of a row: weekday as eyebrow, the date in Fraunces, brass for today. */
function DayNumber({ date, today }: { date: string; today: boolean }) {
  return (
    <View className="w-14 items-start">
      <Text variant="eyebrow" tone={today ? "brass" : "muted"}>
        {weekdayShort(date)}
      </Text>
      <Text variant="heroNumberSm" tone={today ? "brass" : "default"} className="text-[36px] leading-[40px]">
        {Number(date.slice(8, 10))}
      </Text>
    </View>
  );
}

/**
 * One day of the week as a row (planning list): the big day number, who trains, the activity with
 * minutes, exercise and focus, then reha and show underneath. Tapping edits the day when allowed.
 */
export function WeekDayRow({ day, draftDay, editable, busy, onTake, onEdit }: WeekDayRowProps) {
  const router = useRouter();
  const label = dayStatusLabel(day.status, day.user, day.is_me, day.rest_reason);
  const open = day.status === "open" || (day.status === "today" && !day.user);
  const faded = day.status === "empty" || day.status === "rest";
  const summary = draftDay ? draftSummary(draftDay) : label;
  const head = (
    <>
      <DayNumber date={day.date} today={day.is_today} />
      <View className="flex-1 gap-0.5">
        {draftDay ? (
          <DraftBody day={draftDay} />
        ) : (
          <>
            <Text
              variant="bodyStrong"
              tone={open || faded ? "muted" : day.status === "done" ? "success" : "default"}
              className={open || faded ? "font-sans" : undefined}
            >
              {label}
            </Text>
            {day.activity ? (
              <View className="flex-row items-center gap-1.5">
                <ActivityIcon activity={day.activity} size={14} className="text-muted" />
                <Text variant="bodySm" tone="muted">
                  {day.label}
                  {day.minutes > 0 ? ` · ${day.minutes} Min.` : ""}
                </Text>
              </View>
            ) : null}
            {day.focus ? <Text variant="secondary">Schwerpunkt: {day.focus}</Text> : null}
            {day.note && day.status !== "done" ? (
              <Text variant="secondary" numberOfLines={2}>
                {day.note}
              </Text>
            ) : null}
          </>
        )}
      </View>
    </>
  );
  return (
    <View className="gap-2 py-3" accessible={false}>
      <View className="min-h-[64px] flex-row items-center gap-3">
        {editable ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`${formatDayLong(day.date)}: ${summary}`}
            accessibilityHint="Tag bearbeiten"
            onPress={onEdit}
            className="flex-1 flex-row items-center gap-3 active:opacity-70"
          >
            {head}
          </Pressable>
        ) : (
          <View
            className="flex-1 flex-row items-center gap-3"
            accessible
            accessibilityLabel={`${formatDayLong(day.date)}: ${summary}`}
          >
            {head}
          </View>
        )}
        {day.can_take && !day.is_me && !draftDay ? (
          <Button label="Ich" size="sm" variant="outline" loading={busy} onPress={onTake} />
        ) : null}
      </View>
      {!draftDay && day.exercise ? (
        <View className="ml-[68px]">
          <Button
            label={`Übung: ${day.exercise.title}`}
            variant="ghost"
            size="sm"
            icon={BookOpen}
            className="-ml-4"
            onPress={() => router.push(`/training/exercises/${day.exercise!.id}` as Href)}
          />
        </View>
      ) : null}
      {day.reha ? (
        <View className="ml-[68px] gap-0.5" accessibilityLabel={`Reha: ${weekRehaText(day.reha)}`}>
          <Text variant="eyebrow" tone={day.reha.done ? "success" : "info"}>
            Reha
          </Text>
          <Text variant="secondary">{weekRehaText(day.reha)}</Text>
        </View>
      ) : null}
      {day.show ? (
        <View
          className="ml-[68px] gap-1 border-l-2 border-brass pl-3"
          accessibilityLabel={`Turnier ${day.show.name}`}
        >
          <Text variant="eyebrow" tone="brass">
            Turnier{day.show.time ? ` · ab ${day.show.time} Uhr` : ""}
          </Text>
          <Text variant="bodyStrong">{day.show.name}</Text>
          {showTestsText(day.show.tests ?? []).map((line, i) => (
            <Text key={i} variant="secondary">
              {line}
            </Text>
          ))}
          {!day.show.tests?.length && day.show.classes ? (
            <Text variant="secondary">Prüfungen: {day.show.classes}</Text>
          ) : null}
          {day.show.helper ? <Text variant="secondary">Helfer: {day.show.helper}</Text> : null}
        </View>
      ) : null}
    </View>
  );
}

function draftSummary(day: PlanDay): string {
  const what = day.activity === "rest" ? day.label : `${day.label}${day.minutes > 0 ? `, ${day.minutes} Min.` : ""}`;
  return `Entwurf: ${what}`;
}

/** The proposal of one day in the draft: eyebrow with the source, unit, who, focus, exercise and the reason. */
function DraftBody({ day }: { day: PlanDay }) {
  const rest = day.activity === "rest";
  const source = day.edited ? "Geändert" : day.source === "ai" ? "KI" : "Regel";
  return (
    <>
      <Text variant="eyebrow" tone="accent">
        {`Entwurf · ${source}`}
      </Text>
      <View className="flex-row items-center gap-1.5">
        <ActivityIcon activity={day.activity} size={16} className="text-foreground" />
        <Text variant="bodyStrong" className="flex-1">
          {day.label}
          {day.minutes > 0 ? ` · ${day.minutes} Min.` : ""}
        </Text>
      </View>
      {day.level_label && !rest ? <Text variant="secondary">{day.level_label}</Text> : null}
      {day.user ? <Text variant="secondary">{rest ? `${day.user.name} bleibt eingetragen` : `für ${day.user.name}`}</Text> : null}
      {day.focus ? <Text variant="secondary">Schwerpunkt: {day.focus}</Text> : null}
      {day.exercise ? <Text variant="secondary">Übung: {day.exercise.title}</Text> : null}
      {!day.edited && day.reason ? <Text variant="secondary">{day.reason}</Text> : null}
      {!day.edited && day.note ? <Text variant="secondary">Bedingung: {day.note}</Text> : null}
    </>
  );
}
