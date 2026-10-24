import { Check } from "lucide-react-native";
import { Pressable, TextInput, View } from "react-native";

import { Icon, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import { useColors } from "@/lib/theme";

/**
 * The big number of "Wie war's?": the minutes, prefilled from the tracker and editable, set in
 * Fraunces with the place line next to it ("Minuten in der Halle, aus dem Tracking").
 */
export function DurationField({
  value,
  onChangeText,
  caption,
}: {
  value: string;
  onChangeText: (text: string) => void;
  caption: string;
}) {
  const c = useColors();
  return (
    <View className="flex-row items-end gap-3">
      <TextInput
        accessibilityLabel="Dauer in Minuten"
        value={value}
        onChangeText={(t) => onChangeText(t.replace(/\D/g, "").slice(0, 3))}
        keyboardType="number-pad"
        inputMode="numeric"
        maxLength={3}
        selectTextOnFocus
        placeholder="0"
        placeholderTextColor={c.muted}
        className="min-w-[72px] border-b border-border p-0 font-display text-hero-sm text-foreground"
      />
      <Text variant="bodySm" tone="muted" className="flex-1 pb-2">
        {caption}
      </Text>
    </View>
  );
}

/** One row of the exercise checklist: round-cornered box, ticked in `success`, 44 px high. */
export function TickRow({ label, checked, onPress }: { label: string; checked: boolean; onPress: () => void }) {
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked }}
      accessibilityLabel={label}
      onPress={onPress}
      className="min-h-touch flex-row items-center gap-3 py-1.5 active:opacity-70"
    >
      <View
        className={cn(
          "h-6 w-6 items-center justify-center rounded-sm border-[1.5px]",
          checked ? "border-success bg-success" : "border-muted",
        )}
      >
        {checked ? <Icon as={Check} size={14} strokeWidth={3} className="text-primary-foreground" /> : null}
      </View>
      <Text variant="body" className={cn("flex-1", !checked && "text-muted")}>
        {label}
      </Text>
    </Pressable>
  );
}

/** The note below the checklist: no box, a hairline under the text, like a line in a diary. */
export function NoteField({
  value,
  onChangeText,
  placeholder,
}: {
  value: string;
  onChangeText: (text: string) => void;
  placeholder: string;
}) {
  const c = useColors();
  return (
    <TextInput
      accessibilityLabel="Notiz"
      value={value}
      onChangeText={onChangeText}
      placeholder={placeholder}
      placeholderTextColor={c.muted}
      multiline
      maxLength={2000}
      textAlignVertical="top"
      className="min-h-[44px] border-b border-border pb-2 pt-2 font-sans text-body text-foreground"
    />
  );
}
