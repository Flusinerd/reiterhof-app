import { useQuery } from "@tanstack/react-query";
import { router } from "expo-router";
import { ChevronRight } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { Avatar, Icon, Text } from "@/components/ui";
import { presenceApi, PRESENCE_EVENT, PRESENCE_KEY } from "@/lib/api/presence";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/cn";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { blanketBoard, herePeople, presenceFaces, presenceParts, type BoardHorse, type BoardStatus } from "@/lib/stable-tab";
import type { TodayHorse } from "@/lib/blankets";

/** Round avatar with a ring in the page color, so overlapping faces stay apart. */
function Face({ name, colorKey, first }: { name: string; colorKey: string | null; first: boolean }) {
  return (
    <View className={cn("rounded-pill border-2 border-background", !first && "-ml-2")}>
      <Avatar name={name} colorKey={colorKey} size="sm" />
    </View>
  );
}

/**
 * "Wer ist da": the faces of the people at the stable and one sentence with their names
 * ("Anna, Mia und Lea sind da"). Tapping it opens `/presence`; it follows the realtime stream.
 */
export function PresenceRow() {
  const { user } = useAuth();
  const { data } = useQuery({ queryKey: PRESENCE_KEY, queryFn: presenceApi.overview });
  useInvalidateOnEvents({ [PRESENCE_EVENT]: [PRESENCE_KEY] });

  const people = herePeople(data, user);
  const { faces, extra } = presenceFaces(people, 4);
  const sentence = presenceParts(people, !!data);
  const spoken = sentence.rest ? `${sentence.names} ${sentence.rest}` : sentence.names;

  return (
    <View className="gap-1">
      <Text variant="eyebrow">Wer ist da</Text>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`Anwesenheit: ${spoken}`}
        accessibilityHint="Zeigt, wer im Stall ist"
        onPress={() => router.push("/presence")}
        className="min-h-[56px] flex-row items-center gap-3 active:opacity-70"
      >
        {faces.length > 0 ? (
          <View className="flex-row">
            {faces.map((p, i) => (
              <Face key={p.id} name={p.name} colorKey={p.colorKey} first={i === 0} />
            ))}
            {extra > 0 ? (
              <View className="-ml-2 h-9 w-9 items-center justify-center rounded-pill border-2 border-background bg-divider">
                <Text variant="caption" className="font-sans-semibold">{`+${extra}`}</Text>
              </View>
            ) : null}
          </View>
        ) : null}
        <Text variant="body" tone={people.length === 0 ? "muted" : "default"} className="flex-1">
          <Text variant="bodyStrong" tone={people.length === 0 ? "muted" : "default"}>
            {sentence.names}
          </Text>
          {sentence.rest ? ` ${sentence.rest}` : ""}
        </Text>
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </Pressable>
    </View>
  );
}

const DOT: Record<BoardStatus, string> = { done: "bg-success", open: "bg-muted", mine_open: "bg-accent" };

/**
 * Tonight's blankets as a board: every horse is its avatar with its name and a small dot (done
 * `success`, open `muted`, my own horse still open `accent`). Tapping a horse opens its blanket plan.
 */
export function BlanketBoard({
  horses,
  mineIds,
  onOpen,
}: {
  horses: readonly TodayHorse[];
  mineIds: ReadonlySet<string>;
  onOpen: (horseId: string) => void;
}) {
  const board: BoardHorse[] = blanketBoard(horses, mineIds);
  return (
    <View className="-mx-1 flex-row flex-wrap">
      {board.map((h) => (
        <Pressable
          key={h.id}
          accessibilityRole="button"
          accessibilityLabel={`${h.label}. Deckenplan öffnen`}
          onPress={() => onOpen(h.id)}
          className="w-1/5 items-center gap-1 px-1 py-2 active:opacity-70"
        >
          <View>
            <Avatar name={h.name} colorKey={h.colorKey} size="md" />
            <View className={cn("absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-pill border-2 border-background", DOT[h.status])} />
          </View>
          <Text variant="caption" numberOfLines={1} className={h.status === "done" ? undefined : "text-foreground"}>
            {h.name}
          </Text>
        </Pressable>
      ))}
    </View>
  );
}
