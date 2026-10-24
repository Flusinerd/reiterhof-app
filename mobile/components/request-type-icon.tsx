import { Footprints, HandHelping, Shirt, Stethoscope, Trophy, Truck, Wheat, type LucideIcon } from "lucide-react-native";

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

/** The Lucide icon of a request type (unknown types get the helping hand), e.g. for a `Pill`. */
export function requestTypeIcon(type: string): LucideIcon {
  return ICONS[typeMeta(type).icon] ?? HandHelping;
}
