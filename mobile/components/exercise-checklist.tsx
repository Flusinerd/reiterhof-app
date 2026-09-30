import { Check } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { Card, Divider, Icon, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import { checklistLabel } from "@/lib/tracking-exercises";

/** "Ablauf" of an exercise as a checklist the rider ticks off during the session. */
export function ExerciseChecklist({
  title,
  steps,
  checked,
  onToggle,
}: {
  title: string;
  steps: string[];
  checked: number[];
  onToggle: (index: number) => void;
}) {
  return (
    <Card padded={false}>
      <View className="gap-1 p-5 pb-3">
        <Text variant="bodyStrong">{title}</Text>
        <Text variant="secondary">{checklistLabel(checked.length, steps.length)}</Text>
      </View>
      {steps.map((step, i) => {
        const done = checked.includes(i);
        return (
          <View key={i}>
            <Divider />
            <Pressable
              accessibilityRole="checkbox"
              accessibilityState={{ checked: done }}
              accessibilityLabel={`Schritt ${i + 1}: ${step}`}
              onPress={() => onToggle(i)}
              className="min-h-touch flex-row items-center gap-3 px-5 py-3 active:bg-background"
            >
              <View
                className={cn(
                  "h-7 w-7 items-center justify-center rounded-pill border",
                  done ? "border-primary bg-primary" : "border-border bg-card",
                )}
              >
                {done ? <Icon as={Check} size={16} className="text-white" /> : null}
              </View>
              <Text variant="body" className={cn("flex-1", done && "text-muted line-through")}>
                {step}
              </Text>
            </Pressable>
          </View>
        );
      })}
    </Card>
  );
}
