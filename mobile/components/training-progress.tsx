import { Goal } from "lucide-react-native";
import type { ReactElement } from "react";
import { View } from "react-native";

import { Icon } from "@/components/ui";
import { ListRow } from "@/components/ui/list-row";
import { nextLections, type NextLection } from "@/lib/training-goal";

/**
 * The rows of the level goal: one row with the sentence of the API ("Auf dem Weg zu L: 7 von 12
 * Lektionen gefestigt.") and one row per lection that comes next, each opening the exercise. Put
 * them into a `RowList` (they are returned as a list so that the hairlines sit between them).
 */
export function progressRows({
  text,
  next,
  onOpenExercise,
}: {
  text: string | null;
  next: readonly NextLection[] | undefined;
  onOpenExercise: (id: string) => void;
}): ReactElement[] {
  const rows: ReactElement[] = [];
  if (text) {
    rows.push(
      <ListRow
        key="goal"
        leading={<Icon as={Goal} size={20} className="text-muted" />}
        eyebrow="Fortschritt"
        eyebrowTone="brass"
        title={text}
        accessibilityLabel={`Fortschritt: ${text}`}
      />,
    );
  }
  for (const lection of nextLections({ next })) {
    rows.push(
      <ListRow
        key={lection.id}
        leading={<View className="w-5" />}
        eyebrow="Als Nächstes"
        title={lection.title}
        accessibilityLabel={`Als Nächstes: ${lection.title}`}
        onPress={() => onOpenExercise(lection.id)}
      />,
    );
  }
  return rows;
}
