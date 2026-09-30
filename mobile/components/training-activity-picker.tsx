import { View } from "react-native";

import { Pill, Sheet } from "@/components/ui";
import { ACTIVITIES, activityLabel, type Activity } from "@/lib/training";

/** "Wählen": pick another activity than the recommended ones. */
export function ActivityPicker({
  hidden,
  selected,
  open,
  onOpenChange,
  onPick,
}: {
  hidden: readonly Activity[];
  selected: string | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onPick: (activity: Activity) => void;
}) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange} title="Aktivität wählen">
      <View className="flex-row flex-wrap gap-2">
        {ACTIVITIES.filter((a) => !hidden.includes(a)).map((a) => (
          <Pill
            key={a}
            label={activityLabel(a)}
            selected={selected === a}
            onPress={() => {
              onPick(a);
              onOpenChange(false);
            }}
          />
        ))}
      </View>
    </Sheet>
  );
}
