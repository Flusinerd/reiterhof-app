import { Avatar } from "@/components/ui";
import { ListRow } from "@/components/ui/list-row";
import type { Horse } from "@/lib/api/horses";
import { joinParts } from "@/lib/horse-format";

/** One row of the horse list: photo (or initial on the horse's color), name, box and owner. Put rows in a `RowList`. */
export function HorseListItem({ horse, onPress }: { horse: Horse; onPress: () => void }) {
  const sub = joinParts([
    horse.box ? `Box ${horse.box}` : null,
    horse.owner ? (horse.is_mine ? "Dein Pferd" : horse.owner.name) : "ohne Besitzer",
  ]);
  return (
    <ListRow
      leading={<Avatar name={horse.name} colorKey={horse.color_key} photoUrl={horse.photo_url} size="lg" />}
      title={horse.name}
      detail={sub}
      accessibilityLabel={`${horse.name}, ${sub}`}
      onPress={onPress}
      className="min-h-[80px]"
    />
  );
}
