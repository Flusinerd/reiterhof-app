import { View } from "react-native";

import { Button, Card, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";

/**
 * A load that failed and has nothing to show instead ("Heute", "Stall"): says so, where an empty screen
 * would read as "all done", and offers a retry. `onRetry` returns its promise so the button spins until
 * the reload is over.
 */
export function LoadFailed({ title, error, onRetry }: { title: string; error: unknown; onRetry: () => unknown }) {
  return (
    <Card className="gap-3">
      <Text variant="bodyStrong" tone="danger" accessibilityRole="alert">
        {title}
      </Text>
      <Text variant="secondary">{errorMessage(error)}</Text>
      <Button label="Erneut versuchen" variant="outline" onPress={onRetry} />
    </Card>
  );
}

/** A quiet line above content that still shows what was loaded earlier, after a refresh has failed. */
export function NotCurrent({ onRetry }: { onRetry: () => unknown }) {
  return (
    <View className="flex-row flex-wrap items-center">
      <Text variant="caption" accessibilityRole="alert">
        Nicht aktuell.
      </Text>
      <Button label="Erneut versuchen" variant="ghost" size="sm" onPress={onRetry} />
    </View>
  );
}
