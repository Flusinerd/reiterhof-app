import { useTrainingHorses } from "@/lib/api/training";

import { HorseRecord } from "../horses/[id]/index";
import { Horses } from "../horses/index";

/**
 * The horse tab: the user's only horse (owner or rider) directly, otherwise all horses. The tab is
 * named after that horse (app/(tabs)/_layout.tsx).
 */
export default function HorseTab() {
  const mine = useTrainingHorses().data ?? [];
  if (mine.length === 1) return <HorseRecord id={mine[0].id} back={false} />;
  return <Horses back={false} />;
}
