import { X } from "lucide-react-native";
import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  Animated,
  Easing,
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  View,
  useWindowDimensions,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { cn } from "@/lib/cn";
import { sheet as sheetTokens } from "@/lib/theme";

import { Icon } from "./icon";
import { Text } from "./text";

export type SheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** German title shown in the header (Fraunces). */
  title?: string;
  /** Secondary line below the title. */
  description?: string;
  children?: ReactNode;
  /** Classes for the content area below the header. */
  className?: string;
};

/**
 * Bottom sheet: dimmed backdrop, slides up, closes on backdrop tap, close button
 * and the Android back button. Radius 20 on top, 1 px border, no shadow.
 * At most 90 % of the window high; longer content scrolls below the fixed header.
 *
 * @example
 * <Sheet open={open} onOpenChange={setOpen} title="Neue Anfrage">
 *   <Button label="Senden" fullWidth onPress={send} />
 * </Sheet>
 */
export function Sheet({ open, onOpenChange, title, description, children, className }: SheetProps) {
  const { height } = useWindowDimensions();
  const insets = useSafeAreaInsets();
  const progress = useRef(new Animated.Value(0)).current;
  const [mounted, setMounted] = useState(open);

  useEffect(() => {
    if (open) {
      setMounted(true);
      Animated.timing(progress, {
        toValue: 1,
        duration: 240,
        easing: Easing.out(Easing.cubic),
        useNativeDriver: true,
      }).start();
    } else {
      Animated.timing(progress, {
        toValue: 0,
        duration: 180,
        easing: Easing.in(Easing.cubic),
        useNativeDriver: true,
      }).start(({ finished }) => {
        if (finished) setMounted(false);
      });
    }
  }, [open, progress]);

  const close = () => onOpenChange(false);

  return (
    <Modal
      visible={mounted}
      transparent
      animationType="none"
      statusBarTranslucent
      onRequestClose={close}
    >
      <Animated.View
        style={[StyleSheet.absoluteFill, { backgroundColor: sheetTokens.backdrop, opacity: progress }]}
      >
        <Pressable
          accessibilityLabel="Schließen"
          accessibilityRole="button"
          style={StyleSheet.absoluteFill}
          onPress={close}
        />
      </Animated.View>
      <View className="flex-1 justify-end" pointerEvents="box-none">
        <Animated.View
          style={{
            transform: [
              { translateY: progress.interpolate({ inputRange: [0, 1], outputRange: [height, 0] }) },
            ],
          }}
        >
          <View
            className="rounded-t-card border border-b-0 border-border bg-card pt-3"
            // A number, not a percentage: the animated parent has no height of its own.
            style={{ maxHeight: Math.min(height * 0.9, height - insets.top - 16) }}
          >
            <View className="mb-4 h-1 w-10 self-center rounded-pill bg-border" />
            <View className="mb-4 flex-row items-start justify-between gap-4 px-6">
              <View className="flex-1 gap-1">
                {title ? (
                  <Text variant="title" accessibilityRole="header">
                    {title}
                  </Text>
                ) : null}
                {description ? <Text variant="secondary">{description}</Text> : null}
              </View>
              <Pressable
                accessibilityRole="button"
                accessibilityLabel="Schließen"
                onPress={close}
                className="h-11 w-11 items-center justify-center rounded-pill border border-border bg-card active:bg-background"
              >
                <Icon as={X} size={20} className="text-foreground" />
              </Pressable>
            </View>
            <ScrollView
              style={{ flexGrow: 0, flexShrink: 1 }}
              contentContainerStyle={{ paddingBottom: Math.max(insets.bottom, 16) + 8 }}
              keyboardShouldPersistTaps="handled"
            >
              <View className={cn("gap-4 px-6", className)}>{children}</View>
            </ScrollView>
          </View>
        </Animated.View>
      </View>
    </Modal>
  );
}
