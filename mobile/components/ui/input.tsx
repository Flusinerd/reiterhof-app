import { TextInput, type TextInputProps } from "react-native";

import { cn } from "@/lib/cn";
import { colors } from "@/lib/theme";

export type InputProps = TextInputProps & { className?: string };

/**
 * Single-line text field: white card surface, 1 px border, 48 px high.
 *
 * @example
 * <Input value={email} onChangeText={setEmail} placeholder="E-Mail-Adresse" keyboardType="email-address" />
 */
export function Input({ className, ...props }: InputProps) {
  return (
    <TextInput
      placeholderTextColor={colors.muted}
      className={cn(
        "h-12 rounded-button-md border border-border bg-card px-4 font-sans text-body text-foreground",
        className,
      )}
      {...props}
    />
  );
}
