import { Footprints, Hand, Moon, Route, RotateCw, Trees, TrendingUp, Warehouse } from "lucide-react-native";
import type { LucideIcon } from "lucide-react-native";

import { Icon } from "@/components/ui/icon";
import { activityIconKey, type ActivityIconKey } from "@/lib/training";

const ICONS: Record<ActivityIconKey, LucideIcon> = {
  warehouse: Warehouse,
  trees: Trees,
  route: Route,
  rotate: RotateCw,
  "trending-up": TrendingUp,
  hand: Hand,
  footprints: Footprints,
  moon: Moon,
};

/** Lucide icon of an activity (hall, arena, hack, ...). */
export function ActivityIcon({
  activity,
  size = 24,
  className,
}: {
  activity: string;
  size?: number;
  className?: string;
}) {
  return <Icon as={ICONS[activityIconKey(activity)]} size={size} className={className ?? "text-primary-deep"} />;
}
