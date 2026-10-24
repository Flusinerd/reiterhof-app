import { View } from "react-native";

import { Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";

/** The steps of an exercise as a numbered list: the number in Fraunces brass, the step beside it. */
export function NumberedSteps({ steps }: { steps: readonly string[] }) {
  return (
    <RowList>
      {steps.map((step, i) => (
        <View key={i} className="flex-row items-start gap-4 py-4" accessible accessibilityLabel={`Schritt ${i + 1}: ${step}`}>
          <Text variant="heroNumberSm" tone="brass" className="w-9 text-[32px] leading-[36px]" accessibilityElementsHidden>
            {i + 1}
          </Text>
          <Text variant="body" className="flex-1 pt-1">
            {step}
          </Text>
        </View>
      ))}
    </RowList>
  );
}
