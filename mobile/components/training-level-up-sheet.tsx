import { Button, Sheet, Text } from "@/components/ui";
import { levelUpConfirm, levelUpLabel } from "@/lib/training-goal";

/**
 * Confirms "Auf L einstufen": the profile's level becomes the class the horse has mastered, the
 * goal goes back to the next class. A sheet, like every confirmation in the app.
 */
export function LevelUpSheet({
  open,
  onOpenChange,
  horseName,
  letter,
  saving,
  error,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  horseName: string;
  letter: string;
  saving: boolean;
  error: string | null;
  onConfirm: () => void;
}) {
  const { title, description } = levelUpConfirm(horseName, letter);
  return (
    <Sheet open={open} onOpenChange={onOpenChange} title={title} description={description}>
      {error ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button label={levelUpLabel(letter)} fullWidth loading={saving} onPress={onConfirm} />
      <Button label="Abbrechen" variant="ghost" fullWidth disabled={saving} onPress={() => onOpenChange(false)} />
    </Sheet>
  );
}
