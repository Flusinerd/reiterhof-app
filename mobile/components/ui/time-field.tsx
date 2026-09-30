import DateTimePicker, { DateTimePickerAndroid } from "@react-native-community/datetimepicker";
import { Clock, X } from "lucide-react-native";
import { useState } from "react";
import { Platform, Pressable, View } from "react-native";

import { cn } from "@/lib/cn";
import { dateToTime, timeToDate } from "@/lib/time-picker";

import { Icon } from "./icon";
import { Text } from "./text";

export type TimeFieldProps = {
  /** "HH:MM", or "" when unset. */
  value: string;
  onChange: (value: string) => void;
  /** German accessibility label. */
  accessibilityLabel: string;
  placeholder?: string;
  /** Show a button that resets the value to "". */
  clearable?: boolean;
  /** Minute steps of the picker (iOS and web only). */
  minuteInterval?: 1 | 5 | 10 | 15 | 30;
  className?: string;
};

/**
 * Time field with the platform's native picker: the system dialog on Android, an inline wheel on iOS
 * (web: `<input type="time">`, see `time-field.web.tsx`). Looks like `Input`; the value is "HH:MM".
 *
 * @example <TimeField value={time} onChange={setTime} accessibilityLabel="Uhrzeit" placeholder="18:00" />
 */
export function TimeField({
  value,
  onChange,
  accessibilityLabel,
  placeholder = "--:--",
  clearable = false,
  minuteInterval,
  className,
}: TimeFieldProps) {
  const [iosOpen, setIosOpen] = useState(false);

  function open() {
    if (Platform.OS === "android") {
      DateTimePickerAndroid.open({
        value: timeToDate(value),
        mode: "time",
        is24Hour: true,
        onValueChange: (_e, date) => onChange(dateToTime(date)),
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
          accessibilityValue={{ text: value ? `${value} Uhr` : "nicht gesetzt" }}
          onPress={open}
          className="h-full flex-1 flex-row items-center gap-2 px-4 active:opacity-70"
        >
          <Icon as={Clock} size={16} className="text-muted" />
          <Text variant="body" tone={value ? "default" : "muted"}>
            {value || placeholder}
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
            value={timeToDate(value)}
            mode="time"
            display="spinner"
            is24Hour
            locale="de-DE"
            themeVariant="light"
            minuteInterval={minuteInterval}
            onValueChange={(_e, date) => onChange(dateToTime(date))}
          />
        </View>
      ) : null}
    </View>
  );
}
