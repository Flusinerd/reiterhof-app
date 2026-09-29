import { createContext, useContext, type ReactNode } from "react";
import { Pressable, View, type ViewProps } from "react-native";

import { cn } from "@/lib/cn";

import { Text } from "./text";

type TabsContextValue = { value: string; onValueChange: (value: string) => void };
const TabsContext = createContext<TabsContextValue | null>(null);

function useTabs(): TabsContextValue {
  const ctx = useContext(TabsContext);
  if (!ctx) throw new Error("TabsList/TabsTrigger/TabsContent must be used inside <Tabs>");
  return ctx;
}

export type TabsProps = ViewProps & {
  /** Currently selected tab value (controlled). */
  value: string;
  onValueChange: (value: string) => void;
  className?: string;
};

/**
 * In-page segmented tabs (shadcn API). Not the bottom navigation, which is
 * configured in `app/(tabs)/_layout.tsx`.
 *
 * @example
 * <Tabs value={tab} onValueChange={setTab}>
 *   <TabsList>
 *     <TabsTrigger value="today" label="Heute" />
 *     <TabsTrigger value="week" label="Woche" />
 *   </TabsList>
 *   <TabsContent value="today"><Text>...</Text></TabsContent>
 * </Tabs>
 */
export function Tabs({ value, onValueChange, className, ...props }: TabsProps) {
  return (
    <TabsContext.Provider value={{ value, onValueChange }}>
      <View className={cn("gap-4", className)} {...props} />
    </TabsContext.Provider>
  );
}

export function TabsList({ className, ...props }: ViewProps & { className?: string }) {
  return (
    <View
      accessibilityRole="tablist"
      className={cn("flex-row gap-1 rounded-button-lg bg-divider p-1", className)}
      {...props}
    />
  );
}

export type TabsTriggerProps = {
  value: string;
  /** German label. */
  label: string;
  className?: string;
};

export function TabsTrigger({ value, label, className }: TabsTriggerProps) {
  const tabs = useTabs();
  const selected = tabs.value === value;
  return (
    <Pressable
      accessibilityRole="tab"
      accessibilityState={{ selected }}
      onPress={() => tabs.onValueChange(value)}
      className={cn(
        "min-h-touch flex-1 items-center justify-center rounded-button-md border px-3",
        selected ? "border-border bg-card" : "border-transparent",
        className,
      )}
    >
      <Text
        className={cn(
          "text-body-sm",
          selected ? "font-sans-semibold text-foreground" : "font-sans-medium text-muted",
        )}
      >
        {label}
      </Text>
    </Pressable>
  );
}

export type TabsContentProps = { value: string; children: ReactNode; className?: string };

/** Renders its children only while its tab is selected. */
export function TabsContent({ value, children, className }: TabsContentProps) {
  const tabs = useTabs();
  if (tabs.value !== value) return null;
  return <View className={className}>{children}</View>;
}
