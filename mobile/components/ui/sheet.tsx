import { X } from "lucide-react-native";
import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  Animated,
  Easing,
  Modal,
  Pressable,
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
            className="max-h-[90%] rounded-t-card border border-b-0 border-border bg-card px-6 pt-3"
            style={{ paddingBottom: Math.max(insets.bottom, 16) + 8 }}
          >
            <View className="mb-4 h-1 w-10 self-center rounded-pill bg-border" />
            <View className="mb-4 flex-row items-start justify-between gap-4">
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
            <View className={cn("gap-4", className)}>{children}</View>
          </View>
        </Animated.View>
      </View>
    </Modal>
  );
}
