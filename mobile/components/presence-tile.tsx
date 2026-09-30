import { useQuery } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { ChevronRight } from "lucide-react-native";
import { View } from "react-native";

import { Avatar, Icon, PressableCard, Text } from "@/components/ui";
import { presenceApi, PRESENCE_EVENT, PRESENCE_KEY } from "@/lib/api/presence";
import { useInvalidateOnEvents } from "@/lib/realtime";

/**
 * "Anwesenheit" tile for the start screen: who is at the stable right now. Tapping it opens
 * `/presence`. Keeps itself up to date through the realtime stream.
 */
export function PresenceTile() {
  const router = useRouter();
  const { data } = useQuery({ queryKey: PRESENCE_KEY, queryFn: presenceApi.overview });
  useInvalidateOnEvents({ [PRESENCE_EVENT]: [PRESENCE_KEY] });

  const others = data?.here ?? [];
  const total = others.length + (data?.me.open_visit ? 1 : 0);
  const summary = !data
    ? "Wird geladen ..."
    : total === 0
      ? "Gerade ist niemand im Stall"
      : total === 1
        ? "1 Person im Stall"
        : `${total} Personen im Stall`;

  return (
    <PressableCard accessibilityLabel={`Anwesenheit: ${summary}`} onPress={() => router.push("/presence")}>
      <View className="flex-row items-center gap-4">
        <View className="flex-1 gap-1">
          <Text variant="label">Anwesenheit</Text>
          <Text variant="bodyStrong">{summary}</Text>
          <View className="mt-2 flex-row">
            {others.slice(0, 5).map((p, i) => (
              <View key={p.user_id} className={i === 0 ? "" : "-ml-2"}>
                <Avatar name={p.name} colorKey={p.avatar_color} size="sm" />
              </View>
            ))}
          </View>
        </View>
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </View>
    </PressableCard>
  );
}
