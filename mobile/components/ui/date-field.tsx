import DateTimePicker, { DateTimePickerAndroid } from "@react-native-community/datetimepicker";
import { CalendarDays, X } from "lucide-react-native";
import { useState } from "react";
import { Platform, Pressable, View } from "react-native";

import { cn } from "@/lib/cn";
import { dateToIso, isoToDate } from "@/lib/date-field";
import { formatDate } from "@/lib/training";

import { Icon } from "./icon";
import { Text } from "./text";

export type DateFieldProps = {
  /** "YYYY-MM-DD", or "" when unset. */
  value: string;
  onChange: (value: string) => void;
  /** German accessibility label. */
  accessibilityLabel: string;
  /** Shown while the value is empty. Default "Datum wählen". */
  placeholder?: string;
  /** Show a button that resets the value to "". */
  clearable?: boolean;
  /** Earliest date the picker offers. */
  minimumDate?: Date;
  className?: string;
};

/**
 * Date field with the platform's native picker: the system dialog on Android, an inline wheel on iOS
 * (web: `<input type="date">`, see `date-field.web.tsx`). Looks like `Input`; the value is "YYYY-MM-DD",
 * shown as "17.05.2026".
 *
 * @example <DateField value={date} onChange={setDate} accessibilityLabel="Datum" clearable />
 */
export function DateField({
  value,
  onChange,
  accessibilityLabel,
  placeholder = "Datum wählen",
  clearable = false,
  minimumDate,
  className,
}: DateFieldProps) {
  const [iosOpen, setIosOpen] = useState(false);

  function open() {
    if (Platform.OS === "android") {
      DateTimePickerAndroid.open({
        value: isoToDate(value),
        mode: "date",
        minimumDate,
        onValueChange: (_e, date) => onChange(dateToIso(date)),
      });
    } else {
      setIosOpen((o) => !o);
    }
  }

  return (
    <View className={cn("gap-2", className)}>
      <View className="h-12 flex-row items-center rounded-button-md border border-border bg-card">
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={accessibilityLabel}
          accessibilityValue={{ text: value ? formatDate(value) : "nicht gesetzt" }}
          onPress={open}
          className="h-full flex-1 flex-row items-center gap-2 px-4 active:opacity-70"
        >
          <Icon as={CalendarDays} size={16} className="text-muted" />
          <Text variant="body" tone={value ? "default" : "muted"}>
            {value ? formatDate(value) : placeholder}
          </Text>
        </Pressable>
        {clearable && value ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`${accessibilityLabel} entfernen`}
            hitSlop={8}
            onPress={() => {
              setIosOpen(false);
              onChange("");
            }}
            className="h-full items-center justify-center px-3 active:opacity-70"
          >
            <Icon as={X} size={16} className="text-muted" />
          </Pressable>
        ) : null}
      </View>
      {Platform.OS === "ios" && iosOpen ? (
        <View className="items-center rounded-button-md border border-border bg-card">
          <DateTimePicker
            value={isoToDate(value)}
            mode="date"
            display="spinner"
            locale="de-DE"
            themeVariant="light"
            minimumDate={minimumDate}
            onValueChange={(_e, date) => onChange(dateToIso(date))}
          />
        </View>
      ) : null}
    </View>
  );
}
