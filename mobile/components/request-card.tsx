import { Clock, MapPin, Repeat } from "lucide-react-native";
import { View } from "react-native";

import { RequestTypeIcon } from "@/components/request-type-icon";
import { Avatar, Badge, Button, Icon, PressableCard, Text } from "@/components/ui";
import {
  acceptLabel,
  describeRule,
  formatWhen,
  helperCountText,
  requestTitle,
  statusBadge,
  taskChips,
  type HelpRequest,
} from "@/lib/requests";

export type RequestCardProps = {
  request: HelpRequest;
  onOpen: () => void;
  /** Shown as the accept button when the server says the user can accept. */
  onAccept?: () => void;
  accepting?: boolean;
};

/** One request in a list: type tile, title, when/where, task chips, helpers x/y, accept button. */
export function RequestCard({ request: r, onOpen, onAccept, accepting }: RequestCardProps) {
  const status = statusBadge(r.status);
  const chips = taskChips(r);
  const rule = describeRule(r.recurring_rule, r.date);
  return (
    <PressableCard onPress={onOpen} accessibilityLabel={requestTitle(r)} className="gap-3">
      <View className="flex-row items-start gap-3">
        <RequestTypeIcon type={r.type} />
        <View className="flex-1 gap-1">
          <Text variant="bodyStrong">{requestTitle(r)}</Text>
          <Text variant="secondary">von {r.is_creator ? "dir" : r.creator_name}</Text>
        </View>
        <Badge label={status.label} variant={status.variant} />
      </View>

      <View className="gap-1.5">
        <View className="flex-row items-center gap-2">
          <Icon as={Clock} size={16} className="text-muted" />
          <Text variant="bodySm">{formatWhen(r)}</Text>
        </View>
        {r.location ? (
          <View className="flex-row items-center gap-2">
            <Icon as={MapPin} size={16} className="text-muted" />
            <Text variant="bodySm" className="flex-1">
              {r.location}
            </Text>
          </View>
        ) : null}
        {rule ? (
          <View className="flex-row items-center gap-2">
            <Icon as={Repeat} size={16} className="text-muted" />
            <Text variant="bodySm">{rule}</Text>
          </View>
        ) : null}
      </View>

      {chips.length > 0 ? (
        <View className="flex-row flex-wrap gap-2">
          {chips.map((c) => (
            <Badge key={c} label={c} />
          ))}
        </View>
      ) : null}

      <View className="flex-row items-center justify-between gap-3">
        <View className="flex-1 flex-row items-center gap-2">
          <View className="flex-row">
            {r.helpers.slice(0, 4).map((h, i) => (
              <Avatar key={h.user_id} name={h.name} colorKey={h.avatar_color} size="sm" className={i > 0 ? "-ml-2" : undefined} />
            ))}
          </View>
          <Text variant="secondary">{helperCountText(r.helpers_count, r.helpers_needed, r.type)}</Text>
        </View>
        {r.can_accept && onAccept ? (
          <Button label={acceptLabel(r.type)} size="sm" loading={accepting} onPress={onAccept} />
        ) : r.is_helper && (r.status === "open" || r.status === "assigned") ? (
          <Badge label="Du hilfst" variant="primary" />
        ) : null}
      </View>
    </PressableCard>
  );
}
