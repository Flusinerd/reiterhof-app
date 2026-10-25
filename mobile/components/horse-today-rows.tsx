import type { Href } from "expo-router";
import { Pressable, View } from "react-native";

import { Button, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import type { RowAction, TodayRow } from "@/lib/horse-page";
import { pushOnce } from "@/lib/navigation";

/**
 * "Heute für <Name>": up to three rows with hairlines (training today, tonight's blanket, the next
 * health item), each with at most one action. The training row's "Starten" is the page's one
 * primary button; the others are outlined. `busy` is the row key whose action is running.
 */
export function HorseTodayRows({
  rows,
  busy,
  onAction,
}: {
  rows: readonly TodayRow[];
  busy?: TodayRow["key"] | null;
  /** May return a promise: the row's button stays busy until it settles. */
  onAction: (action: RowAction) => void | Promise<unknown>;
}) {
  return (
    <View>
      {rows.map((row, i) => (
        <View
          key={row.key}
          className={cn("min-h-[64px] flex-row items-center gap-3 py-3", i < rows.length - 1 && "border-b border-divider")}
        >
          <Pressable
            accessibilityRole={row.route ? "button" : undefined}
            accessibilityLabel={[row.eyebrow, row.title, row.detail].filter(Boolean).join(", ")}
            disabled={!row.route}
            onPress={() => pushOnce(row.route as Href)}
            className="flex-1 gap-0.5 active:opacity-70"
          >
            <Text variant="eyebrow" tone={row.eyebrowTone}>
              {row.eyebrow}
            </Text>
            <Text variant="bodyStrong">{row.title}</Text>
            {row.detail ? (
              <Text variant="bodySm" tone="muted">
                {row.detail}
              </Text>
            ) : null}
          </Pressable>
          {row.action ? (
            <Button
              label={row.action.label}
              size="sm"
              variant={row.action.kind === "start" ? "primary" : row.action.kind === "undo" ? "ghost" : "outline"}
              loading={busy === row.key}
              onPress={() => onAction(row.action as RowAction)}
            />
          ) : null}
        </View>
      ))}
    </View>
  );
}
