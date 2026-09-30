import { Footprints, HandHelping, Shirt, Stethoscope, Trophy, Truck, Wheat, type LucideIcon } from "lucide-react-native";
import { View } from "react-native";

import { Icon } from "@/components/ui";
import { cn } from "@/lib/cn";
import { typeMeta } from "@/lib/requests";

// Lucide icon per request type; the names come from lib/requests.ts (REQUEST_TYPES).
const ICONS: Record<string, LucideIcon> = {
  Trophy,
  Truck,
  Footprints,
  Wheat,
  Stethoscope,
  HandHelping,
  Shirt,
};

export function requestTypeIcon(type: string): LucideIcon {
  return ICONS[typeMeta(type).icon] ?? HandHelping;
}

export type RequestTypeIconProps = {
  type: string;
  /** Tile edge in px (default 44). */
  size?: number;
  selected?: boolean;
  className?: string;
};

/** Rounded tile with the icon of a request type. */
export function RequestTypeIcon({ type, size = 44, selected = false, className }: RequestTypeIconProps) {
  return (
    <View
      className={cn(
        "items-center justify-center rounded-tile",
        selected ? "bg-primary" : "bg-primary-soft",
        className,
      )}
      style={{ width: size, height: size }}
    >
      <Icon
        as={requestTypeIcon(type)}
        size={Math.round(size * 0.5)}
        className={selected ? "text-white" : "text-primary-deep"}
      />
    </View>
  );
}
