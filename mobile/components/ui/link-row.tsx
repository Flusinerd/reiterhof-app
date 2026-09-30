import type { LucideIcon } from "lucide-react-native";
import { ChevronRight } from "lucide-react-native";
import { View } from "react-native";

import { cn } from "@/lib/cn";

import { PressableCard } from "./card";
import { Icon } from "./icon";
import { Text } from "./text";

export type LinkRowProps = {
  icon: LucideIcon;
  /** German title of the row. */
  label: string;
  /** One line of context below the title, e.g. a summary of what is behind the link. */
  description?: string;
  onPress: () => void;
  className?: string;
};

/**
 * Navigation row: icon, title, optional description and a chevron. Stack several inside one
 * `Card padded={false}` with `Divider`s between them, or use one on its own.
 *
 * @example <LinkRow icon={FileText} label="Dokumente" description="3 Dateien" onPress={open} />
 */
export function LinkRow({ icon, label, description, onPress, className }: LinkRowProps) {
  return (
    <PressableCard
      padded={false}
      onPress={onPress}
      accessibilityLabel={label}
      className={cn("min-h-touch flex-row items-center gap-3 px-4 py-3", className)}
    >
      <Icon as={icon} size={20} className="text-primary" />
      <View className="flex-1">
        <Text variant="bodyStrong">{label}</Text>
        {description ? <Text variant="secondary">{description}</Text> : null}
      </View>
      <Icon as={ChevronRight} size={20} className="text-muted" />
    </PressableCard>
  );
}
