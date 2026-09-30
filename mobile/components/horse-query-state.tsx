import { ActivityIndicator, View } from "react-native";

import { Button, Card, Text } from "@/components/ui";
import { ApiError, errorMessage } from "@/lib/api";
import { colors } from "@/lib/theme";

/** Centered spinner for a screen or block that is still loading. */
export function HorseLoading() {
  return (
    <View className="items-center py-10" accessibilityLabel="Wird geladen">
      <ActivityIndicator color={colors.primary.DEFAULT} />
    </View>
  );
}

/** Error card with a retry button. A 403 gets its own text (the record is not visible to the user). */
export function HorseError({
  error,
  onRetry,
  forbiddenText = "Dafür fehlt dir die Berechtigung.",
}: {
  error: unknown;
  onRetry?: () => void;
  forbiddenText?: string;
}) {
  const forbidden = error instanceof ApiError && error.status === 403;
  const notFound = error instanceof ApiError && error.status === 404;
  return (
    <Card className="gap-3">
      <Text variant="bodyStrong">
        {forbidden ? "Kein Zugriff" : notFound ? "Nicht gefunden" : "Das hat nicht geklappt"}
      </Text>
      <Text variant="secondary">
        {forbidden ? forbiddenText : notFound ? "Dieses Pferd gibt es nicht (mehr)." : errorMessage(error)}
      </Text>
      {onRetry && !forbidden && !notFound ? (
        <Button label="Erneut versuchen" variant="outline" size="sm" onPress={onRetry} />
      ) : null}
    </Card>
  );
}
