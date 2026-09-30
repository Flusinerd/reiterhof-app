import {
  Activity,
  Bell,
  Check,
  CloudRain,
  Eye,
  HandHelping,
  HeartPulse,
  Pill,
  Shirt,
  Stethoscope,
  TriangleAlert,
  type LucideIcon,
} from "lucide-react-native";
import { Pressable, View } from "react-native";

import { Button, Icon, Text } from "@/components/ui";
import { isInternalRoute } from "@/lib/notifications-core";
import { kindLabel, sentLabel, timeLabel, type ReminderItem } from "@/lib/reminders";

const KIND_ICONS: Record<string, LucideIcon> = {
  last_person: Shirt,
  weather_change: CloudRain,
  medication: Pill,
  health_due: Stethoscope,
  reha_checkup: HeartPulse,
  helper: HandHelping,
  new_request: HandHelping,
  training_plan: Activity,
  urgent_observation: TriangleAlert,
  observation: Eye,
};

export type ReminderRowProps = {
  item: ReminderItem;
  timeZone: string;
  dismissing?: boolean;
  /** Called with the route when the row is tapped (only for items with an internal screen). */
  onOpen: (screen: string) => void;
  onDismiss: (item: ReminderItem) => void;
};

/**
 * One reminder: kind icon, title, text and time. Tapping opens the screen of the reminder;
 * "Erledigt" (stored reminders only) removes it from the list.
 */
export function ReminderRow({ item, timeZone, dismissing = false, onOpen, onDismiss }: ReminderRowProps) {
  const target = isInternalRoute(item.screen) ? item.screen : null;
  const meta = [timeLabel(item, timeZone), kindLabel(item.kind)].filter(Boolean).join(" · ");
  const sent = sentLabel(item, timeZone);

  const content = (
    <>
      <View className="h-10 w-10 items-center justify-center rounded-tile bg-primary-soft">
        <Icon as={KIND_ICONS[item.kind] ?? Bell} size={20} className="text-primary" />
      </View>
      <View className="flex-1 gap-0.5">
        <Text variant="bodyStrong">{item.title}</Text>
        {item.body ? <Text variant="bodySm" tone="muted">{item.body}</Text> : null}
        <Text variant="caption">{sent ? `${meta} · ${sent}` : meta}</Text>
      </View>
    </>
  );

  return (
    <View className="flex-row items-center gap-3 px-5 py-4">
      {target ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${item.title}. ${item.body}`}
          className="min-h-touch flex-1 flex-row items-center gap-3"
          onPress={() => onOpen(target)}
        >
          {content}
        </Pressable>
      ) : (
        <View className="min-h-touch flex-1 flex-row items-center gap-3">{content}</View>
      )}
      {item.dismissible ? (
        <Button
          size="icon"
          variant="ghost"
          icon={Check}
          loading={dismissing}
          accessibilityLabel={`Erledigt: ${item.title}`}
          onPress={() => onDismiss(item)}
        />
      ) : null}
    </View>
  );
}
