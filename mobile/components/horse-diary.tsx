import type { Href } from "expo-router";
import { Megaphone } from "lucide-react-native";
import { useMemo } from "react";
import { Pressable, View } from "react-native";

import { AuthImage } from "@/components/auth-image";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Button, Text } from "@/components/ui";
import { useAuth } from "@/lib/auth";
import { useDiary, type Horse } from "@/lib/api/horses";
import { cn } from "@/lib/cn";
import { DIARY_EMPTY, diaryView, flattenDiary, type DiaryDot, type DiaryView } from "@/lib/horse-page";
import { pushOnce } from "@/lib/navigation";
import { newObservationRoute } from "@/lib/observations";
import { localDate } from "@/lib/presence-format";

const DOT_CLASS: Record<DiaryDot, string> = {
  success: "bg-success",
  accent: "bg-accent",
  danger: "bg-danger",
  muted: "bg-border",
};

function DiaryItem({ view, last }: { view: DiaryView; last: boolean }) {
  const body = (
    <View className="gap-1">
      <Text variant="eyebrow">{view.when}</Text>
      <Text variant="bodyStrong">{view.title}</Text>
      {view.body ? (
        <Text variant="bodySm" numberOfLines={3}>
          {view.body}
        </Text>
      ) : null}
      {view.quote ? (
        <Text variant="body" className="font-display-italic text-foreground/80">
          {`„${view.quote}“`}
        </Text>
      ) : null}
      {view.meta ? (
        <Text variant="bodySm" tone="muted">
          {view.meta}
        </Text>
      ) : null}
      {view.photoUrl ? (
        <View className="mt-1 flex-row items-end gap-2">
          <AuthImage
            url={view.photoUrl}
            accessibilityLabel="Foto der Auffälligkeit"
            accessibilityIgnoresInvertColors
            resizeMode="cover"
            style={{ width: 72, height: 72 }}
            className="rounded-tile bg-divider"
          />
          {view.photoCount > 1 ? (
            <Text variant="caption" tone="muted">
              + {view.photoCount - 1} weitere
            </Text>
          ) : null}
        </View>
      ) : null}
    </View>
  );
  return (
    <View className={cn("relative pb-4 pl-5 pt-1", last && "pb-0")}>
      <View className={cn("absolute left-0 top-2.5 h-2 w-2 rounded-pill", DOT_CLASS[view.dot])} />
      {view.route ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={[view.when, view.title].join(", ")}
          onPress={() => pushOnce(view.route as Href)}
          className="active:opacity-70"
        >
          {body}
        </Pressable>
      ) : (
        body
      )}
    </View>
  );
}

/**
 * The diary of a horse: training sessions and observations of everybody in one strand, newest
 * first, in pages ("Mehr laden"). Notes of the riders are set as italic quotes, observation photos
 * as thumbnails.
 */
export function HorseDiary({ horse, timeZone }: { horse: Horse; timeZone: string }) {
  const { user } = useAuth();
  const diary = useDiary(horse.id);
  const today = localDate(new Date(), timeZone);
  const views = useMemo(
    () => flattenDiary(diary.data?.pages ?? []).map((e) => diaryView(e, { today, meId: user?.id })),
    [diary.data, today, user?.id],
  );

  return (
    <View className="gap-4">
      <Button
        label="Auffälligkeit melden"
        icon={Megaphone}
        variant="ghost"
        size="sm"
        className="-ml-3"
        onPress={() => pushOnce(newObservationRoute(horse.id) as Href)}
      />
      {diary.isError ? (
        <HorseError error={diary.error} onRetry={() => diary.refetch()} />
      ) : diary.isPending ? (
        <HorseLoading />
      ) : views.length === 0 ? (
        <Text variant="body" tone="muted">
          {DIARY_EMPTY}
        </Text>
      ) : (
        <View className="relative">
          <View className="absolute bottom-1 left-[3.5px] top-3 w-px bg-border" />
          {views.map((v, i) => (
            <DiaryItem key={v.key} view={v} last={i === views.length - 1} />
          ))}
        </View>
      )}
      {diary.hasNextPage ? (
        <Button
          label="Mehr laden"
          variant="outline"
          size="sm"
          loading={diary.isFetchingNextPage}
          onPress={() => diary.fetchNextPage()}
        />
      ) : null}
    </View>
  );
}
