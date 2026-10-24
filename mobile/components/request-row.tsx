import { Pressable, View } from "react-native";

import { Avatar, Button, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import type { HelpRequest } from "@/lib/requests";
import { requestRow } from "@/lib/stable-tab";

export type RequestRowProps = {
  request: HelpRequest;
  /** Local day, YYYY-MM-DD ("heute", "morgen"). */
  today: string;
  /** The signed-in user, so "Du hilfst" can be said. */
  userId?: string;
  /** False inside a list grouped by day: the time alone is enough. */
  withDay?: boolean;
  /** No hairline below the last row of a list. */
  last?: boolean;
  onOpen: () => void;
  /** Shown as the accept button when the server says the user can accept. */
  onAccept?: () => void;
  accepting?: boolean;
};

/**
 * One help request as a message: the avatar of who asked, name and time in brass, the text, what
 * it is about, and the quiet answer ("Mach ich" and who already helps). Rows sit on the page with
 * hairlines between them, never in cards. Done and cancelled requests fade.
 */
export function RequestRow({ request, today, userId, withDay = true, last = false, onOpen, onAccept, accepting }: RequestRowProps) {
  const m = requestRow(request, today, userId, withDay);
  const faded = m.state === "done" || m.state === "cancelled";
  return (
    <View className={cn("flex-row gap-3 py-4", !last && "border-b border-divider", faded && "opacity-60")}>
      <Avatar name={request.creator_name} colorKey={m.colorKey} size="md" />
      <View className="flex-1 gap-2">
        <Pressable accessibilityRole="button" accessibilityLabel={m.a11y} onPress={onOpen} className="gap-1 active:opacity-70">
          <Text variant="bodyStrong">
            {m.name}
            <Text variant="bodySm" tone="brass" className="font-sans-semibold">
              {` · ${m.when}`}
            </Text>
          </Text>
          <Text variant="body">{m.text}</Text>
          {m.context ? <Text variant="secondary">{m.context}</Text> : null}
        </Pressable>
        <View className="flex-row flex-wrap items-center gap-x-3 gap-y-1">
          {m.action && onAccept ? (
            <Button label={m.action} variant="outline" size="sm" loading={accepting} onPress={onAccept} />
          ) : null}
          <Text variant="secondary" tone={m.state === "helping" ? "success" : "muted"} className="flex-shrink">
            {m.helpers}
          </Text>
        </View>
      </View>
    </View>
  );
}
