import { useRouter } from "expo-router";
import { ChevronLeft } from "lucide-react-native";
import { Pressable, type PressableProps } from "react-native";

import { cn } from "@/lib/cn";

import { Icon } from "./icon";

export type BackButtonProps = Omit<PressableProps, "children"> & { className?: string };

/**
 * Round 44 px back button for sub pages (screens without tab bar). Goes back in
 * the navigation stack, or to the start screen if there is no history.
 *
 * @example <BackButton />
 */
export function BackButton({ className, onPress, ...props }: BackButtonProps) {
  const router = useRouter();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel="Zurück"
      onPress={(event) => {
        if (onPress) {
          onPress(event);
        } else if (router.canGoBack()) {
          router.back();
        } else {
          router.replace("/");
        }
      }}
      className={cn(
        "h-11 w-11 items-center justify-center rounded-pill border border-border bg-card active:bg-background",
        className,
      )}
      {...props}
    >
      <Icon as={ChevronLeft} size={22} className="text-foreground" />
    </Pressable>
  );
}
