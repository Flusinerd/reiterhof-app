import { Check, EyeOff } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { Button, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import { isInternalRoute } from "@/lib/notifications-core";
import { dismissAction, rowMeta, rowTime, type ReminderItem } from "@/lib/reminders";
import { reminderText } from "@/lib/today";

export type ReminderRowProps = {
  item: ReminderItem;
  timeZone: string;
  /** The stable-local day (YYYY-MM-DD), to tell what is overdue. */
  today: string;
  /** No hairline below the last row of a day. */
  last?: boolean;
  dismissing?: boolean;
  /** Called with the route when the row is tapped (only for items with an internal screen). */
  onOpen: (screen: string) => void;
  /** May return a promise: the button stays busy until it settles. */
  onDismiss: (item: ReminderItem) => void | Promise<unknown>;
};

/**
 * One reminder as a line of the timeline, the same look as on "Heute": the time in brass (red
 * "fällig" when overdue), title, text and kind. Tapping opens the screen of the reminder; the quiet
 * button on the right (stored reminders only) removes it: "Erledigt", or "Ausblenden" for a push about an
 * appointment, which is completed in the health list or the reha plan and only hidden here.
 */
export function ReminderRow({ item, timeZone, today, last = false, dismissing = false, onOpen, onDismiss }: ReminderRowProps) {
  const target = isInternalRoute(item.screen) ? item.screen : null;
  const time = rowTime(item, timeZone, today);
  const text = reminderText(item.kind, item.title, item.body);
  const dismiss = dismissAction(item);

  const content = (
    <>
      <Text variant="bodySm" tone={time.overdue ? "danger" : "brass"} className="w-12 font-sans-semibold">
        {time.text}
      </Text>
      <View className="flex-1 gap-0.5">
        <Text variant="bodyStrong">{text.title}</Text>
        {text.detail ? (
          <Text variant="bodySm" tone={time.overdue ? "danger" : "muted"}>
            {text.detail}
          </Text>
        ) : null}
        <Text variant="caption">{rowMeta(item, timeZone)}</Text>
      </View>
    </>
  );

  return (
    <View className={cn("min-h-touch flex-row items-center gap-2", !last && "border-b border-divider")}>
      {target ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${item.title}. ${item.body}`}
          className="min-h-touch flex-1 flex-row gap-4 py-3 active:opacity-70"
          onPress={() => onOpen(target)}
        >
          {content}
        </Pressable>
      ) : (
        <View className="min-h-touch flex-1 flex-row gap-4 py-3">{content}</View>
      )}
      {item.dismissible ? (
        <Button
          size="icon"
          variant="ghost"
          icon={dismiss.hidesOnly ? EyeOff : Check}
          loading={dismissing}
          accessibilityLabel={`${dismiss.verb}: ${item.title}`}
          onPress={() => onDismiss(item)}
        />
      ) : null}
    </View>
  );
}
